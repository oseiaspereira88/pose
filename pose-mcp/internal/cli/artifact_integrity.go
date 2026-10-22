package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/harne8/pose-mcp/internal/cli/cliout"
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
	var trailerCommits []string
	if base == "" && head == "" {
		commits, err := commitsWithSpecTrailer(root, spec)
		if err != nil {
			return posemodel.ChangeSet{}, err
		}
		if len(commits) == 0 {
			return posemodel.ChangeSet{}, fmt.Errorf("no commits carry POSE-Spec: %s", spec)
		}
		trailerCommits = commits
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
	commitsRaw, err := gitOutputBounded(root, 4*1024*1024, "rev-list", "--reverse", resolvedBase+".."+resolvedHead, "--")
	if err != nil {
		return posemodel.ChangeSet{}, err
	}
	commits := splitNonEmpty(string(commitsRaw), "\n")

	var paths []posemodel.ObservedPath
	var diff []byte
	if len(trailerCommits) > 0 {
		commits = trailerCommits
		commitPathLists := make([][]posemodel.ObservedPath, 0, len(commits))
		var diffBuf bytes.Buffer
		for _, sha := range commits {
			nameStatus, err := gitOutputBounded(root, 4*1024*1024, "diff-tree", "--no-commit-id", "--name-status", "-M", "-r", sha, "--")
			if err != nil {
				return posemodel.ChangeSet{}, err
			}
			parsed, err := parseGitNameStatus(nameStatus)
			if err != nil {
				return posemodel.ChangeSet{}, err
			}
			commitPathLists = append(commitPathLists, parsed)
			commitDiff, err := gitOutputBounded(root, 16*1024*1024, "show", "--binary", "--no-ext-diff", "--format=", sha, "--")
			if err != nil {
				return posemodel.ChangeSet{}, err
			}
			diffBuf.Write(commitDiff)
		}
		paths = mergeObservedPaths(commitPathLists)
		diff = diffBuf.Bytes()
	} else {
		nameStatus, err := gitOutputBounded(root, 8*1024*1024, "diff", "--name-status", "-M", "--no-ext-diff", resolvedBase, resolvedHead, "--")
		if err != nil {
			return posemodel.ChangeSet{}, err
		}
		paths, err = parseGitNameStatus(nameStatus)
		if err != nil {
			return posemodel.ChangeSet{}, err
		}
		diff, err = gitOutputBounded(root, 32*1024*1024, "diff", "--binary", "--no-ext-diff", resolvedBase, resolvedHead, "--")
		if err != nil {
			return posemodel.ChangeSet{}, err
		}
	}
	digest := sha256.Sum256(diff)
	idSum := sha256.Sum256([]byte(spec + "\x00" + resolvedBase + "\x00" + resolvedHead + "\x00" + hex.EncodeToString(digest[:])))
	return posemodel.ChangeSet{ID: "cs-" + hex.EncodeToString(idSum[:6]), Spec: spec, Selector: selector, Base: base, Head: head, ResolvedBase: resolvedBase, ResolvedHead: resolvedHead, Commits: commits, Paths: paths, DiffDigest: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

func allCommitsWithSpecTrailers(root string) (map[string][]string, error) {
	// A rolling commit limit changes historical attribution whenever an unrelated
	// commit pushes a tagged commit out of the window. Filter in Git and scan the
	// full matching history; the output bound still fails closed for huge inputs.
	out, err := gitOutputBounded(root, 32*1024*1024, posemodel.SpecTrailerLogArgs...)
	if err != nil {
		return nil, err
	}
	specCommits := posemodel.ParseSpecTrailerLog(out)
	return specCommits, nil
}

func commitsWithSpecTrailer(root, spec string) ([]string, error) {
	specCommits, err := allCommitsWithSpecTrailers(root)
	if err != nil {
		return nil, err
	}
	return specCommits[spec], nil
}

func mergeObservedPaths(commitPathLists [][]posemodel.ObservedPath) []posemodel.ObservedPath {
	type pathState struct {
		action  string
		oldPath string
		newPath string
	}
	stateByPath := map[string]pathState{}
	for _, paths := range commitPathLists {
		for _, p := range paths {
			switch p.Action {
			case "created":
				if prev, ok := stateByPath[p.Path]; ok && prev.action == "removed" {
					stateByPath[p.Path] = pathState{action: "modified", newPath: p.Path}
				} else {
					stateByPath[p.Path] = pathState{action: "created", newPath: p.Path}
				}
			case "modified":
				if _, ok := stateByPath[p.Path]; !ok {
					stateByPath[p.Path] = pathState{action: "modified", newPath: p.Path}
				}
			case "removed":
				if prev, ok := stateByPath[p.Path]; ok && prev.action == "created" {
					delete(stateByPath, p.Path)
				} else {
					stateByPath[p.Path] = pathState{action: "removed", newPath: p.Path}
				}
			case "renamed":
				if prev, ok := stateByPath[p.OldPath]; ok && prev.action == "created" {
					delete(stateByPath, p.OldPath)
					stateByPath[p.NewPath] = pathState{action: "created", newPath: p.NewPath}
				} else {
					initialOld := p.OldPath
					if prev, ok := stateByPath[p.OldPath]; ok && prev.action == "renamed" {
						initialOld = prev.oldPath
						delete(stateByPath, p.OldPath)
					}
					stateByPath[p.NewPath] = pathState{action: "renamed", oldPath: initialOld, newPath: p.NewPath}
				}
			}
		}
	}
	result := make([]posemodel.ObservedPath, 0, len(stateByPath))
	for path, st := range stateByPath {
		switch st.action {
		case "created":
			result = append(result, posemodel.ObservedPath{Action: "created", Path: path})
		case "modified":
			result = append(result, posemodel.ObservedPath{Action: "modified", Path: path})
		case "removed":
			result = append(result, posemodel.ObservedPath{Action: "removed", Path: path})
		case "renamed":
			result = append(result, posemodel.ObservedPath{Action: "renamed", OldPath: st.oldPath, NewPath: st.newPath})
		}
	}
	sort.Slice(result, func(i, j int) bool { return observedSortKey(result[i]) < observedSortKey(result[j]) })
	return result
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

func collectArtifactGraphInputsIn(files posemodel.Files, root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, error) {
	store := posemodel.Store{Root: root, FS: files}
	policy, err := posemodel.LoadArtifactPolicyIn(files, root)
	if err != nil {
		return nil, nil, nil, nil, policy, err
	}
	specs, err := store.ListSpecs("", "")
	if err != nil {
		if _, statErr := files.Stat(filepath.Join(root, ".pose", "specs")); !os.IsNotExist(statErr) {
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
	changeSetsMap := map[string]posemodel.ChangeSet{}
	for _, recorded := range loadRecordedChangeSetsIn(files, root) {
		changeSetsMap[recorded.ID] = recorded
	}
	specTrailers, _ := allCommitsWithSpecTrailers(root)
	for _, spec := range specs {
		if len(specTrailers[spec.Slug]) > 0 {
			if set, err := resolveGitChangeSet(root, spec.Slug, "", ""); err == nil {
				changeSetsMap[set.ID] = set
			}
		}
	}
	changeSets := make([]posemodel.ChangeSet, 0, len(changeSetsMap))
	for _, set := range changeSetsMap {
		changeSets = append(changeSets, set)
	}
	sort.Slice(changeSets, func(i, j int) bool { return changeSets[i].ID < changeSets[j].ID })
	return specs, claims, changeSets, tracked, policy, nil
}

// collectArtifactGraphInputs reads and writes the filesystem directly; collectArtifactGraphInputsIn takes the
// operation's files.
func collectArtifactGraphInputs(root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, error) {
	return collectArtifactGraphInputsIn(posemodel.NativeFiles(), root)
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

func loadRecordedChangeSetsIn(files posemodel.Files, root string) []posemodel.ChangeSet {
	paths, _ := posemodel.GlobIn(files, filepath.Join(root, ".pose", "reports", "history", "*.jsonl"))
	sets := map[string]posemodel.ChangeSet{}
	for _, path := range paths {
		file, err := files.Open(path)
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

// loadRecordedChangeSets reads and writes the filesystem directly; loadRecordedChangeSetsIn takes the
// operation's files.
func loadRecordedChangeSets(root string) []posemodel.ChangeSet {
	return loadRecordedChangeSetsIn(posemodel.NativeFiles(), root)
}

func buildCurrentDeliveryGraphIn(files posemodel.Files, root string) (posemodel.DeliveryIntegrityGraph, error) {
	specs, claims, sets, tracked, policy, err := collectArtifactGraphInputsIn(files, root)
	if err != nil {
		return posemodel.DeliveryIntegrityGraph{}, err
	}
	return extendCurrentDeliveryGraphIn(files, root, posemodel.BuildDeliveryIntegrityWithReleasesIn(files, specs, claims, sets, tracked, policy, posemodel.LoadArchivedFragmentsIn(files, root)))
}

// buildCurrentDeliveryGraph reads and writes the filesystem directly; buildCurrentDeliveryGraphIn takes the
// operation's files.
func buildCurrentDeliveryGraph(root string) (posemodel.DeliveryIntegrityGraph, error) {
	return buildCurrentDeliveryGraphIn(posemodel.NativeFiles(), root)
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
	var sets []posemodel.ChangeSet
	if from != "" || to != "" {
		set, err := resolveGitChangeSet(root, spec, from, to)
		if err != nil {
			fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
			return 1
		}
		sets = append(sets, set)
	} else {
		for _, recorded := range loadRecordedChangeSets(root) {
			if recorded.Spec == spec {
				sets = append(sets, recorded)
			}
		}
		if set, err := resolveGitChangeSet(root, spec, "", ""); err == nil {
			alreadyPresent := false
			for _, existing := range sets {
				if existing.ID == set.ID {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				sets = append(sets, set)
			}
		}
		if len(sets) == 0 {
			fmt.Fprintf(stderr, "pose artifact-check: no commits carry POSE-Spec: %s\n", spec)
			return 1
		}
	}
	set := sets[0]
	tracked, err := gitTrackedPaths(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose artifact-check: %v\n", err)
		return 1
	}
	graph := posemodel.BuildDeliveryIntegrityWithReleases([]posemodel.Spec{*full}, claims, sets, tracked, policy, posemodel.LoadArchivedFragments(root))
	for _, claim := range claims {
		if claim.Action == "none" {
			continue
		}
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
		// The change set's provenance is what an operator needs first when
		// `observed` exceeds `claims`: which revisions were attributed, and how
		// many commits they span. Hiding it behind --json leaves the default
		// output advising "narrow the attributed change set" without saying what
		// the change set is (spec pose-diagnose-invisible-governance-failures).
		fmt.Fprintf(stdout, "artifact.spec=%s\nartifact.change_set=%s\nartifact.change_set.base=%s\nartifact.change_set.head=%s\nartifact.change_set.commits=%d\nartifact.diff_digest=%s\nartifact.claims=%d\nartifact.observed=%d\nartifact.findings=%d\n",
			spec, set.ID, set.ResolvedBase, set.ResolvedHead, len(set.Commits), set.DiffDigest, len(claims), len(set.Paths), len(graph.Findings))
		// A claim resolved through a release names a path that is no longer
		// there; say where it went, or the pass reads as magic (spec
		// pose-release-archival-attested-by-the-ledger).
		for _, a := range graph.Archivals {
			fmt.Fprintf(stdout, "artifact.archived=%s -> %s (release %s)\n", a.Pending, a.Archived, a.Version)
		}
		for _, finding := range graph.Findings {
			render(stdout, stderr).Finding(cliout.Finding{
				State: cliout.StateFromKey(finding.Severity), Code: finding.Code, Path: finding.Path,
				Message: finding.Message, Remediation: finding.Remediation,
			})
		}
	}
	if strict {
		for _, finding := range graph.Findings {
			if finding.Severity == "error" || finding.Severity == "critical" {
				if !jsonOutput {
					PrintContributorFailureHint(root, stderr, cliLocaleValue())
				}
				return 1
			}
		}
	}
	if deliveryGraph, err := buildCurrentDeliveryGraph(root); err == nil {
		if raw, err := deliveryGraph.IndexJSON(); err == nil {
			refreshDeliveryIndex(root, append(raw, '\n'), stderr)
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
		// A spec is the lifecycle record: the backfill takes its lifecycle
		// lock and publishes over the bytes it read, so it never reverts a
		// transition or an edit made meanwhile (spec
		// pose-writer-coordination-and-atomic-publication R2/R3).
		unlock, err := posemodel.LockSpecLifecycle(root, slug, "artifact-backfill")
		if err != nil {
			render(io.Discard, stderr).Failure("pose artifact-backfill: " + err.Error())
			return 1
		}
		raw, err := os.ReadFile(full.Path)
		if err != nil {
			unlock()
			return 1
		}
		text := string(raw)
		marker := "\n## 4. Tasks"
		if !strings.Contains(text, marker) {
			unlock()
			continue
		}
		text = strings.Replace(text, marker, strings.Join(lines, "\n")+"\n"+marker, 1)
		err = posemodel.PublishFile(full.Path, []byte(text), 0o644, posemodel.PublishAuthoritative, posemodel.ExpectBytes(raw, true))
		unlock()
		if err != nil {
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
