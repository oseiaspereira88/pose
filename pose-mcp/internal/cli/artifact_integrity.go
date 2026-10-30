package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/harne8/pose-mcp/internal/cli/cliout"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// gitOutputBounded reads raw object history (posemodel.RawHistoryGit), so the
// change sets a later removal is judged on are the ones it judges (review
// round 3 of spec pose-delivery-later-removal, LR-R3-01).
func gitOutputBounded(root string, limit int, args ...string) ([]byte, error) {
	cmd := posemodel.RawHistoryGit(root, args...)
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

// collectArtifactGraphInputsWithGapsIn also returns, by spec, why a part of
// its history could not be read; the empty key is every spec.
func collectArtifactGraphInputsWithGapsIn(files posemodel.Files, root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, map[string]string, error) {
	store := posemodel.Store{Root: root, FS: files}
	policy, err := posemodel.LoadArtifactPolicyIn(files, root)
	if err != nil {
		return nil, nil, nil, nil, policy, nil, err
	}
	specs, err := store.ListSpecs("", "")
	if err != nil {
		if _, statErr := files.Stat(filepath.Join(root, ".pose", "specs")); !os.IsNotExist(statErr) {
			return nil, nil, nil, nil, policy, nil, err
		}
		specs = []posemodel.Spec{}
	}
	claims := []posemodel.ArtifactClaim{}
	for i := range specs {
		full, err := store.GetSpec(specs[i].Slug)
		if err != nil {
			return nil, nil, nil, nil, policy, nil, err
		}
		parsed, _, err := posemodel.ParseArtifactClaims(*full, policy)
		if err != nil {
			return nil, nil, nil, nil, policy, nil, err
		}
		claims = append(claims, parsed...)
	}
	gaps := map[string]string{}
	tracked, err := gitTrackedPaths(root)
	if err != nil {
		tracked = []string{}
		gaps[""] = "the tracked paths could not be read: " + err.Error()
	}
	changeSetsMap := map[string]posemodel.ChangeSet{}
	for _, recorded := range loadRecordedChangeSetsIn(files, root) {
		changeSetsMap[recorded.ID] = recorded
	}
	specTrailers, err := allCommitsWithSpecTrailers(root)
	if err != nil && gaps[""] == "" {
		gaps[""] = "the commits carrying POSE-Spec trailers could not be read: " + err.Error()
	}
	// A shallow repository may hide commits carrying a trailer, and nothing
	// read from it can show which (LR-R3-02).
	if out, err := gitOutputBounded(root, 64, "rev-parse", "--is-shallow-repository"); err != nil || strings.TrimSpace(string(out)) != "false" {
		if gaps[""] == "" {
			gaps[""] = "the repository's history is shallow or its depth could not be read"
		}
	}
	for _, spec := range specs {
		if len(specTrailers[spec.Slug]) > 0 {
			if set, err := resolveGitChangeSet(root, spec.Slug, "", ""); err == nil {
				changeSetsMap[set.ID] = set
			} else {
				gaps[spec.Slug] = "its trailer change set could not be read: " + err.Error()
			}
		}
	}
	changeSets := make([]posemodel.ChangeSet, 0, len(changeSetsMap))
	for _, set := range changeSetsMap {
		changeSets = append(changeSets, set)
	}
	sort.Slice(changeSets, func(i, j int) bool { return changeSets[i].ID < changeSets[j].ID })
	return specs, claims, changeSets, tracked, policy, gaps, nil
}

// collectArtifactGraphInputs reads and writes the filesystem directly; collectArtifactGraphInputsIn takes the
// operation's files.
func collectArtifactGraphInputs(root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, error) {
	return collectArtifactGraphInputsIn(posemodel.NativeFiles(), root)
}

// collectArtifactGraphInputsIn drops what it could not read; a later removal,
// which must judge every change of a spec, reads the gaps instead
// (collectArtifactGraphInputsWithGapsIn, review round 2 LR-R2-03).
func collectArtifactGraphInputsIn(files posemodel.Files, root string) ([]posemodel.Spec, []posemodel.ArtifactClaim, []posemodel.ChangeSet, []string, posemodel.ArtifactPolicy, error) {
	specs, claims, sets, tracked, policy, _, err := collectArtifactGraphInputsWithGapsIn(files, root)
	return specs, claims, sets, tracked, policy, err
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
	specs, claims, sets, tracked, policy, gaps, err := collectArtifactGraphInputsWithGapsIn(files, root)
	if err != nil {
		return posemodel.DeliveryIntegrityGraph{}, err
	}
	return extendCurrentDeliveryGraphIn(files, root, posemodel.BuildDeliveryIntegrityWithHistoryIn(files, specs, claims, sets, tracked, policy, posemodel.LoadArchivedFragmentsIn(files, root), artifactRemovals(root, claims, sets, tracked, gaps)))
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
	// A later removal by another spec is read from every spec's claims and
	// change sets (spec pose-delivery-later-removal R4).
	var removals []posemodel.ArtifactRemoval
	if _, allClaims, allSets, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root); err == nil {
		removals = artifactRemovals(root, allClaims, allSets, tracked, gaps, sets...)
	}
	graph := posemodel.BuildDeliveryIntegrityWithHistoryIn(posemodel.NativeFiles(), []posemodel.Spec{*full}, claims, sets, tracked, policy, posemodel.LoadArchivedFragments(root), removals)
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

// removalCommitRE is a commit as change sets must name it: a full object
// name, never a revision expression or anything Git could read as an option
// (review round 1, LR-R1-11).
// diffStatusRE is a status of git diff-tree --name-status: a letter, and a
// score for a rename or a copy (LR-R3-04).
var diffStatusRE = regexp.MustCompile(`^([ADMTUX]|[RC][0-9]{1,3})$`)

var removalCommitRE = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// removalHistory reads the Git facts a later removal is judged on, with every
// path literal (never a pathspec, LR-R1-08) and every commit an immutable
// object name; each answer is kept, and an error is an error, never a
// negative answer (LR-R1-04, LR-R1-09).
type removalHistory struct {
	root    string
	shallow map[string]bool
	entries map[string]string
	parents map[string][]string
	renames map[string]map[string]bool
	ancest  map[string]bool
}

// command runs Git on raw object history: replace refs and grafts do not
// change what a removal is judged on (LR-R2-06).
func (h *removalHistory) command(args ...string) *exec.Cmd {
	return posemodel.RawHistoryGit(h.root, append([]string{"--literal-pathspecs"}, args...)...)
}

func (h *removalHistory) git(limit int, args ...string) ([]byte, error) {
	cmd := h.command(args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedBuffer{buf: &stdout, limit: limit}
	cmd.Stderr = &limitedBuffer{buf: &stderr, limit: 64 * 1024}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// entry is the tree entry of path at commit — mode, type and object — or ""
// when the path is absent there.
func (h *removalHistory) entry(commit, path string) (string, error) {
	key := commit + "\x00" + path
	if v, ok := h.entries[key]; ok {
		return v, nil
	}
	out, err := h.git(64*1024, "ls-tree", "-z", "--full-tree", commit, "--", path)
	if err != nil {
		return "", err
	}
	v := ""
	for _, rec := range strings.Split(string(out), "\x00") {
		if meta, name, ok := strings.Cut(rec, "\t"); ok && name == path {
			v = meta
		}
	}
	h.entries[key] = v
	return v, nil
}

// parentsOf reads a commit's parents from the commit object itself; a commit
// at a shallow boundary, whose parents the repository does not hold, is an
// error, never a root (LR-R2-02).
func (h *removalHistory) parentsOf(commit string) ([]string, error) {
	if v, ok := h.parents[commit]; ok {
		return v, nil
	}
	if h.shallow == nil {
		h.shallow = map[string]bool{}
		out, err := h.git(4*1024, "rev-parse", "--git-path", "shallow")
		if err != nil {
			return nil, err
		}
		path := strings.TrimSpace(string(out))
		if !filepath.IsAbs(path) {
			path = filepath.Join(h.root, path)
		}
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		for _, line := range strings.Fields(string(raw)) {
			h.shallow[line] = true
		}
	}
	if h.shallow[commit] {
		return nil, fmt.Errorf("the history is shallow at %s", commit)
	}
	kind, err := h.git(64, "cat-file", "-t", commit)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(kind)) != "commit" {
		return nil, fmt.Errorf("%s is a %s, not a commit", commit, strings.TrimSpace(string(kind)))
	}
	out, err := h.git(1024*1024, "cat-file", "commit", commit)
	if err != nil {
		return nil, err
	}
	header, _, _ := strings.Cut(string(out), "\n\n")
	parents := []string{}
	for _, line := range strings.Split(header, "\n") {
		if parent, ok := strings.CutPrefix(line, "parent "); ok {
			parents = append(parents, parent)
		}
	}
	h.parents[commit] = parents
	return parents, nil
}

// renamedAway reports whether commit, against parent, renames path away.
func (h *removalHistory) renamedAway(parent, commit, path string) (bool, error) {
	key := parent + "\x00" + commit
	if set, ok := h.renames[key]; ok {
		return set[path], nil
	}
	out, err := h.git(8*1024*1024, "diff-tree", "-r", "-M", "-z", "--name-status", "--no-commit-id", parent, commit)
	if err != nil {
		return false, err
	}
	set, err := renamedAwayPaths(out)
	if err != nil {
		return false, err
	}
	h.renames[key] = set
	return set[path], nil
}

// renamedAwayPaths reads the sources of the renames in a git diff-tree -z
// --name-status stream.
func renamedAwayPaths(out []byte) (map[string]bool, error) {
	// Records are a status and its paths: two for a rename or a copy, one
	// otherwise; a path is never read as a status (LR-R2-01).
	set := map[string]bool{}
	if len(out) > 0 && out[len(out)-1] != 0 {
		return nil, errors.New("git diff-tree: unterminated output")
	}
	fields := strings.Split(string(out), "\x00")
	fields = fields[:len(fields)-1]
	for i := 0; i < len(fields); {
		status := fields[i]
		arity := 1
		if diffStatusRE.MatchString(status) && (status[0] == 'R' || status[0] == 'C') {
			arity = 2
		}
		if !diffStatusRE.MatchString(status) || i+arity >= len(fields) {
			return nil, fmt.Errorf("git diff-tree: malformed record %q", status)
		}
		for _, path := range fields[i+1 : i+1+arity] {
			if path == "" {
				return nil, fmt.Errorf("git diff-tree: record %q without a path", status)
			}
		}
		if status[0] == 'R' {
			set[fields[i+1]] = true
		}
		i += 1 + arity
	}
	return set, nil
}

// ancestor reports whether a is an ancestor of b; Git's exit 1 is "no", any
// other failure an error (LR-R1-04).
func (h *removalHistory) ancestor(a, b string) (bool, error) {
	key := a + "\x00" + b
	if v, ok := h.ancest[key]; ok {
		return v, nil
	}
	cmd := h.command("merge-base", "--is-ancestor", a, b)
	var stderr bytes.Buffer
	cmd.Stderr = &limitedBuffer{buf: &stderr, limit: 64 * 1024}
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		h.ancest[key] = true
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		h.ancest[key] = false
	default:
		return false, fmt.Errorf("git merge-base --is-ancestor: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return h.ancest[key], nil
}

// touches reports whether commit changes path's entry against any of its
// parents — a root commit against nothing — so a creation, a modification, a
// type change, a deletion, a rename and a merge that changes it all count
// (LR-R1-02, LR-R1-03).
func (h *removalHistory) touches(commit, path string) (bool, error) {
	here, err := h.entry(commit, path)
	if err != nil {
		return false, err
	}
	parents, err := h.parentsOf(commit)
	if err != nil {
		return false, err
	}
	if len(parents) == 0 {
		return here != "", nil
	}
	for _, parent := range parents {
		there, err := h.entry(parent, path)
		if err != nil {
			return false, err
		}
		if there != here {
			return true, nil
		}
	}
	return false, nil
}

// deletes reports whether commit removes path: absent there, present in a
// parent that does not rename it away (a rename is not a removal, LR-R1-07).
func (h *removalHistory) deletes(commit, path string) (bool, error) {
	here, err := h.entry(commit, path)
	if err != nil || here != "" {
		return false, err
	}
	parents, err := h.parentsOf(commit)
	if err != nil {
		return false, err
	}
	for _, parent := range parents {
		there, err := h.entry(parent, path)
		if err != nil {
			return false, err
		}
		if there == "" {
			continue
		}
		renamed, err := h.renamedAway(parent, commit, path)
		if err != nil {
			return false, err
		}
		if !renamed {
			return true, nil
		}
	}
	return false, nil
}

// artifactRemovals reads from Git what a later removal needs to satisfy the
// existence of a declared artifact (spec pose-delivery-later-removal R1–R4).
// For each path a spec declares removed that is absent at the head: the
// commits of that spec's change sets that delete it, and, for each other spec
// that declares the path created or modified, a deletion that every commit of
// that spec touching the path is an ancestor of. Commits are gathered from all
// of a spec's change sets and judged by ancestry, never by the order the sets
// come in (LR-R1-01, LR-R1-06); a commit that is not an immutable object name,
// or history Git cannot read, refuses with its cause. overrides replaces the
// change sets of a spec, as an explicit artifact-check range does (LR-R1-05,
// LR-R2-04); gaps says, by spec (the empty key for every spec), which history
// the collection could not read, and refuses those specs (LR-R2-03).
func artifactRemovals(root string, claims []posemodel.ArtifactClaim, sets []posemodel.ChangeSet, tracked []string, gaps map[string]string, overrides ...posemodel.ChangeSet) []posemodel.ArtifactRemoval {
	present := map[string]bool{}
	for _, path := range tracked {
		present[filepath.ToSlash(path)] = true
	}
	replaced := map[string]bool{}
	for _, set := range overrides {
		replaced[set.Spec] = true
	}
	commitsOf := map[string][]string{}
	invalid := map[string]string{}
	seen := map[string]bool{}
	judged := append([]posemodel.ChangeSet{}, overrides...)
	for _, set := range sets {
		if !replaced[set.Spec] {
			judged = append(judged, set)
		}
	}
	for spec, why := range gaps {
		if spec != "" && !replaced[spec] {
			invalid[spec] = why
		}
	}
	for _, set := range judged {
		for _, commit := range set.Commits {
			if !removalCommitRE.MatchString(commit) {
				invalid[set.Spec] = "its change set names " + strconv.Quote(commit) + ", which is not an immutable commit"
				continue
			}
			if !seen[set.Spec+"\x00"+commit] {
				seen[set.Spec+"\x00"+commit] = true
				commitsOf[set.Spec] = append(commitsOf[set.Spec], commit)
			}
		}
	}
	for spec := range commitsOf {
		sort.Strings(commitsOf[spec])
	}
	h := &removalHistory{root: root, entries: map[string]string{}, parents: map[string][]string{}, renames: map[string]map[string]bool{}, ancest: map[string]bool{}}
	var removals []posemodel.ArtifactRemoval
	for _, removed := range claims {
		if removed.Action != "removed" || present[removed.Path] {
			continue
		}
		r := posemodel.ArtifactRemoval{Spec: removed.Spec, Path: removed.Path, Supersedes: map[string]string{}, Refusals: map[string]string{}}
		var deletions []string
		var unreadable error
		if why := gaps[""]; why != "" {
			unreadable = errors.New(why)
		} else if why := invalid[removed.Spec]; why != "" {
			unreadable = errors.New("spec " + removed.Spec + ": " + why)
		}
		for _, commit := range commitsOf[removed.Spec] {
			if unreadable != nil {
				break
			}
			deleted, err := h.deletes(commit, removed.Path)
			if err != nil {
				unreadable = err
			} else if deleted {
				deletions = append(deletions, commit)
			}
		}
		if len(deletions) > 0 {
			r.Commit = deletions[0]
		}
		for _, declared := range claims {
			if declared.Path != removed.Path || declared.Spec == removed.Spec || (declared.Action != "created" && declared.Action != "modified") {
				continue
			}
			if unreadable != nil {
				r.Refusals[declared.Spec] = "the history could not be read: " + unreadable.Error()
				continue
			}
			if len(deletions) == 0 {
				continue
			}
			if why := invalid[declared.Spec]; why != "" {
				r.Refusals[declared.Spec] = why
				continue
			}
			via, why := precedingDeletion(h, commitsOf[declared.Spec], declared.Path, deletions)
			if via != "" {
				r.Supersedes[declared.Spec] = via
			} else {
				r.Refusals[declared.Spec] = why
			}
		}
		if len(r.Supersedes) == 0 {
			r.Supersedes = nil
		}
		if len(r.Refusals) == 0 {
			r.Refusals = nil
		}
		removals = append(removals, r)
	}
	return removals
}

// precedingDeletion is the first of deletions that every commit of commits
// touching path is an ancestor of, or why none is.
func precedingDeletion(h *removalHistory, commits []string, path string, deletions []string) (string, string) {
	var touched []string
	for _, commit := range commits {
		ok, err := h.touches(commit, path)
		if err != nil {
			return "", "the history could not be read: " + err.Error()
		}
		if ok {
			touched = append(touched, commit)
		}
	}
	if len(touched) == 0 {
		return "", "this spec's change to it is not in its attributed change set"
	}
	for _, deletion := range deletions {
		all := true
		for _, commit := range touched {
			if commit == deletion {
				all = false
				break
			}
			ok, err := h.ancestor(commit, deletion)
			if err != nil {
				return "", "the history could not be read: " + err.Error()
			}
			if !ok {
				all = false
				break
			}
		}
		if all {
			return deletion, ""
		}
	}
	return "", "this spec's change to it does not precede the removal"
}
