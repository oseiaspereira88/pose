package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

var safeGitRevision = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/~^+-]{0,127}$`)

func validateGitRevision(value string) error {
	if !safeGitRevision.MatchString(value) || strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.HasPrefix(value, "-") {
		return fmt.Errorf("unsafe Git revision %q", value)
	}
	return nil
}

func gitOutputBounded(root string, limit int, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedBuffer{buf: &stdout, limit: limit}
	cmd.Stderr = &limitedBuffer{buf: &stderr, limit: 64 * 1024}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s failed: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

type limitedBuffer struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	if w.buf.Len()+len(p) > w.limit {
		return 0, fmt.Errorf("bounded Git output exceeded %d bytes", w.limit)
	}
	return w.buf.Write(p)
}

func resolveGitRevision(root, revision string) (string, error) {
	if err := validateGitRevision(revision); err != nil {
		return "", err
	}
	out, err := gitOutputBounded(root, 1024, "rev-parse", "--verify", revision+"^{commit}")
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(string(out))
	if !regexp.MustCompile(`^[0-9a-f]{40,64}$`).MatchString(resolved) {
		return "", fmt.Errorf("Git revision %q did not resolve to an immutable commit", revision)
	}
	return resolved, nil
}

func resolveGitChangeSet(root, spec, base, head string) (posemodel.ChangeSet, error) {
	if spec == "" || posemodel.ValidateSlug(spec) != nil {
		return posemodel.ChangeSet{}, fmt.Errorf("a valid spec slug is required")
	}
	selector := "range:" + base + ".." + head
	if base == "" && head == "" {
		commits, err := commitsWithSpecTrailer(root, spec)
		if err != nil {
			return posemodel.ChangeSet{}, err
		}
		if len(commits) == 0 {
			return posemodel.ChangeSet{}, fmt.Errorf("no commits carry POSE-Spec: %s", spec)
		}
		head = commits[len(commits)-1]
		base = commits[0] + "^"
		selector = "trailers:" + spec
	} else if base == "" || head == "" {
		return posemodel.ChangeSet{}, fmt.Errorf("--from and --to must be supplied together")
	}
	resolvedBase, err := resolveGitRevision(root, base)
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	resolvedHead, err := resolveGitRevision(root, head)
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	nameStatus, err := gitOutputBounded(root, 8*1024*1024, "diff", "--name-status", "-M", "--no-ext-diff", resolvedBase, resolvedHead, "--")
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	paths, err := parseGitNameStatus(nameStatus)
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	diff, err := gitOutputBounded(root, 32*1024*1024, "diff", "--binary", "--no-ext-diff", resolvedBase, resolvedHead, "--")
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	digest := sha256.Sum256(diff)
	commitsRaw, err := gitOutputBounded(root, 4*1024*1024, "rev-list", "--reverse", resolvedBase+".."+resolvedHead, "--")
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	commits := splitNonEmpty(string(commitsRaw), "\n")
	idSum := sha256.Sum256([]byte(spec + "\x00" + resolvedBase + "\x00" + resolvedHead + "\x00" + hex.EncodeToString(digest[:])))
	return posemodel.ChangeSet{ID: "cs-" + hex.EncodeToString(idSum[:6]), Spec: spec, Selector: selector, Base: base, Head: head, ResolvedBase: resolvedBase, ResolvedHead: resolvedHead, Commits: commits, Paths: paths, DiffDigest: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

func commitsWithSpecTrailer(root, spec string) ([]string, error) {
	out, err := gitOutputBounded(root, 8*1024*1024, "log", "--all", "--reverse", "--max-count=500", "--format=%H%x00%B%x00")
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(out, []byte{0})
	commits := []string{}
	for i := 0; i+1 < len(parts); i += 2 {
		sha, body := strings.TrimSpace(string(parts[i])), string(parts[i+1])
		for _, line := range strings.Split(body, "\n") {
			if strings.TrimSpace(line) == "POSE-Spec: "+spec {
				commits = append(commits, sha)
				break
			}
		}
	}
	return commits, nil
}

func parseGitNameStatus(raw []byte) ([]posemodel.ObservedPath, error) {
	paths := []posemodel.ObservedPath{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed git name-status output")
		}
		status := parts[0]
		var observed posemodel.ObservedPath
		switch status[0] {
		case 'A':
			observed = posemodel.ObservedPath{Action: "created", Path: filepath.ToSlash(parts[1])}
		case 'M', 'T':
			observed = posemodel.ObservedPath{Action: "modified", Path: filepath.ToSlash(parts[1])}
		case 'D':
			observed = posemodel.ObservedPath{Action: "removed", Path: filepath.ToSlash(parts[1])}
		case 'R':
			if len(parts) != 3 {
				return nil, fmt.Errorf("malformed Git rename record")
			}
			observed = posemodel.ObservedPath{Action: "renamed", OldPath: filepath.ToSlash(parts[1]), NewPath: filepath.ToSlash(parts[2])}
		default:
			return nil, fmt.Errorf("unsupported Git change status %q", status)
		}
		paths = append(paths, observed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(paths, func(i, j int) bool { return observedSortKey(paths[i]) < observedSortKey(paths[j]) })
	return paths, nil
}

func observedSortKey(path posemodel.ObservedPath) string {
	return path.Action + "\x00" + path.Path + "\x00" + path.OldPath + "\x00" + path.NewPath
}

func collectArtifactGraphInputs(root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, error) {
	store := posemodel.Store{Root: root}
	policy, err := posemodel.LoadArtifactPolicy(root)
	if err != nil {
		return nil, nil, nil, nil, policy, err
	}
	specs, err := store.ListSpecs("", "")
	if err != nil {
		if _, statErr := os.Stat(filepath.Join(root, ".pose", "specs")); !os.IsNotExist(statErr) {
			return nil, nil, nil, nil, policy, err
		}
		specs = []posemodel.Spec{}
	}
	claims := []posemodel.ArtifactClaim{}
	for i := range specs {
		full, err := store.GetSpec(specs[i].Slug)
		if err != nil {
			return nil, nil, nil, nil, policy, err
		}
		parsed, _, err := posemodel.ParseArtifactClaims(*full, policy)
		if err != nil {
			return nil, nil, nil, nil, policy, err
		}
		claims = append(claims, parsed...)
	}
	tracked, err := gitTrackedPaths(root)
	if err != nil {
		tracked = []string{}
	}
	changeSets := loadRecordedChangeSets(root)
	return specs, claims, changeSets, tracked, policy, nil
}

func gitTrackedPaths(root string) ([]string, error) {
	trackedRaw, err := gitOutputBounded(root, 32*1024*1024, "ls-files", "-z", "--")
	if err != nil {
		return nil, err
	}
	tracked := []string{}
	for _, item := range bytes.Split(trackedRaw, []byte{0}) {
		if len(item) > 0 {
			tracked = append(tracked, filepath.ToSlash(string(item)))
		}
	}
	return tracked, nil
}

func loadRecordedChangeSets(root string) []posemodel.ChangeSet {
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "reports", "history", "*.jsonl"))
	sets := map[string]posemodel.ChangeSet{}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var record reportRecord
			if json.Unmarshal(scanner.Bytes(), &record) == nil && record.ChangeSet != nil {
				sets[record.ChangeSet.ID] = *record.ChangeSet
			}
		}
		_ = file.Close()
	}
	values := make([]posemodel.ChangeSet, 0, len(sets))
	for _, set := range sets {
		values = append(values, set)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values
}

func buildCurrentDeliveryGraph(root string) (posemodel.DeliveryIntegrityGraph, error) {
	specs, claims, sets, tracked, policy, err := collectArtifactGraphInputs(root)
	if err != nil {
		return posemodel.DeliveryIntegrityGraph{}, err
	}
	return extendCurrentDeliveryGraph(root, posemodel.BuildDeliveryIntegrity(specs, claims, sets, tracked, policy))
}

func cmdArtifactCheck(root string, args []string, stdout, stderr io.Writer) int {
	var spec, from, to string
	strict, jsonOutput := true, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--spec", "--from", "--to":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose artifact-check --spec <slug> [--from rev --to rev] [--strict|--tolerant] [--json]")
			}
			i++
			switch args[i-1] {
			case "--spec":
				spec = args[i]
			case "--from":
				from = args[i]
			case "--to":
				to = args[i]
			}
		case "--strict":
			strict = true
		case "--tolerant":
			strict = false
		case "--json":
			jsonOutput = true
		default:
			return usageError(stderr, "Usage: pose artifact-check --spec <slug> [--from rev --to rev] [--strict|--tolerant] [--json]")
		}
	}
	store := posemodel.Store{Root: root}
	full, err := store.GetSpec(spec)
	if err != nil {
		fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
		return 1
	}
	policy, err := posemodel.LoadArtifactPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
		return 1
	}
	claims, found, err := posemodel.ParseArtifactClaims(*full, policy)
	if err != nil || !found || len(claims) == 0 {
		fmt.Fprintf(stderr, "pose artifact-check: spec %s has no valid artifact declaration: %v\n", spec, err)
		return 1
	}
	set, err := resolveGitChangeSet(root, spec, from, to)
	if err != nil {
		fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
		return 1
	}
	tracked, err := gitTrackedPaths(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
		return 1
	}
	graph := posemodel.BuildDeliveryIntegrity([]posemodel.Spec{*full}, claims, []posemodel.ChangeSet{set}, tracked, policy)
	for _, claim := range claims {
		paths := []string{claim.Path}
		if claim.Action == "renamed" {
			paths = []string{claim.OldPath, claim.NewPath}
		}
		for _, path := range paths {
			if err := posemodel.ValidateArtifactPath(root, path, false); err != nil {
				severity := policy.Severities["resolvability"]
				if severity == "" {
					severity = "warning"
				}
				graph.Findings = append(graph.Findings, posemodel.NewDeliveryIntegrityFinding("resolvability", severity, spec, path, set.ID, err.Error(), "declare an exact confined tracked file path"))
			}
		}
	}
	sort.Slice(graph.Findings, func(i, j int) bool { return graph.Findings[i].ID < graph.Findings[j].ID })
	if jsonOutput {
		_ = writeJSON(stdout, graph)
	} else {
		fmt.Fprintf(stdout, "artifact.spec=%s\nartifact.change_set=%s\nartifact.diff_digest=%s\nartifact.claims=%d\nartifact.observed=%d\nartifact.findings=%d\n", spec, set.ID, set.DiffDigest, len(claims), len(set.Paths), len(graph.Findings))
		for _, finding := range graph.Findings {
			fmt.Fprintf(stdout, "[%s] %s %s: %s; remediation: %s\n", strings.ToUpper(finding.Severity), finding.Code, finding.Path, finding.Message, finding.Remediation)
		}
	}
	if strict {
		for _, finding := range graph.Findings {
			if finding.Severity == "error" || finding.Severity == "critical" {
				return 1
			}
		}
	}
	return 0
}

type artifactBackfillProposal struct {
	SchemaVersion int                                 `json:"schema_version"`
	DryRun        bool                                `json:"dry_run"`
	Specs         map[string][]posemodel.ObservedPath `json:"specs"`
	Confidence    map[string]string                   `json:"confidence"`
	Conflicts     map[string][]string                 `json:"conflicts"`
}

func cmdArtifactBackfill(root string, args []string, stdout, stderr io.Writer) int {
	apply, confirm, fromGit := false, false, false
	for _, arg := range args {
		switch arg {
		case "--from-git":
			fromGit = true
		case "--apply":
			apply = true
		case "--confirm-spec-edits":
			confirm = true
		default:
			return usageError(stderr, "Usage: pose artifact-backfill --from-git [--apply --confirm-spec-edits]")
		}
	}
	if !fromGit {
		return usageError(stderr, "Usage: pose artifact-backfill --from-git [--apply --confirm-spec-edits]")
	}
	if apply && !confirm {
		fmt.Fprintln(stderr, "pose artifact-backfill: --apply requires --confirm-spec-edits")
		return 2
	}
	store := posemodel.Store{Root: root}
	specs, err := store.ListSpecs("", "")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	proposal := artifactBackfillProposal{SchemaVersion: 1, DryRun: !apply, Specs: map[string][]posemodel.ObservedPath{}, Confidence: map[string]string{}, Conflicts: map[string][]string{}}
	pathOwners := map[string][]string{}
	for _, spec := range specs {
		set, err := resolveGitChangeSet(root, spec.Slug, "", "")
		if err != nil {
			continue
		}
		proposal.Specs[spec.Slug] = set.Paths
		for _, path := range set.Paths {
			value := path.Path
			if path.Action == "renamed" {
				value = path.NewPath
			}
			pathOwners[value] = appendUniqueArtifactString(pathOwners[value], spec.Slug)
		}
	}
	for path, owners := range pathOwners {
		if len(owners) > 1 {
			sort.Strings(owners)
			proposal.Conflicts[path] = owners
		}
	}
	for slug, paths := range proposal.Specs {
		proposal.Confidence[slug] = "high"
		for _, observed := range paths {
			if len(proposal.Conflicts[firstObservedCLIPath(observed)]) > 0 {
				proposal.Confidence[slug] = "ambiguous"
				break
			}
		}
	}
	if !apply {
		return writeJSON(stdout, proposal)
	}
	for slug, paths := range proposal.Specs {
		full, err := store.GetSpec(slug)
		if err != nil {
			return 1
		}
		_, found, parseErr := posemodel.ParseArtifactClaims(*full, posemodel.ArtifactPolicy{})
		if parseErr != nil || found {
			continue
		}
		lines := []string{"\n### Artifacts"}
		for _, observed := range paths {
			value := observed.Path
			if observed.Action == "renamed" {
				value = observed.OldPath + " -> " + observed.NewPath
			}
			if len(proposal.Conflicts[firstObservedCLIPath(observed)]) == 0 {
				lines = append(lines, "- "+observed.Action+": "+value)
			}
		}
		if len(lines) == 1 {
			continue
		}
		raw, err := os.ReadFile(full.Path)
		if err != nil {
			return 1
		}
		text := string(raw)
		marker := "\n## 4. Tasks"
		if !strings.Contains(text, marker) {
			continue
		}
		text = strings.Replace(text, marker, strings.Join(lines, "\n")+"\n"+marker, 1)
		if err := writeAtomic(full.Path, []byte(text), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	proposal.DryRun = false
	return writeJSON(stdout, proposal)
}

func firstObservedCLIPath(path posemodel.ObservedPath) string {
	if path.Action == "renamed" {
		return path.NewPath
	}
	return path.Path
}

func appendUniqueArtifactString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
