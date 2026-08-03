package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

var immutableCommitRE = regexp.MustCompile(`^[0-9a-f]{40}$`)
var assetDigestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func cmdRelease(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose release <plan|prepare|check|notes|record|status|open-next|backfill> [options]")
	}
	action := args[0]
	args = args[1:]
	switch action {
	case "plan":
		return cmdReleasePlan(root, args, stdout, stderr)
	case "prepare":
		return cmdReleasePrepare(root, args, stdout, stderr)
	case "check":
		return cmdReleaseCheck(root, args, stdout, stderr)
	case "notes":
		return cmdReleaseNotesSnapshot(root, args, stdout, stderr)
	case "record":
		return cmdReleaseRecord(root, args, stdout, stderr)
	case "status":
		return cmdReleaseStatus(root, args, stdout, stderr)
	case "open-next":
		return cmdReleaseOpenNext(root, args, stdout, stderr)
	case "backfill":
		return cmdReleaseBackfill(root, args, stdout, stderr)
	default:
		return usageError(stderr, "Usage: pose release <plan|prepare|check|notes|record|status|open-next|backfill> [options]")
	}
}

func releaseArg(args []string, name string) (string, bool, error) {
	value := ""
	found := false
	for i := 0; i < len(args); i++ {
		if args[i] == name {
			if found || i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return "", false, fmt.Errorf("%s requires one value", name)
			}
			value = args[i+1]
			found = true
			i++
		}
	}
	return value, found, nil
}
func releaseFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}
func rejectReleaseArgs(args []string, allowedValues, allowedFlags map[string]bool) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if allowedFlags[arg] {
			continue
		}
		if allowedValues[arg] {
			i++
			if i >= len(args) {
				return fmt.Errorf("%s requires a value", arg)
			}
			continue
		}
		return fmt.Errorf("unexpected argument %s", arg)
	}
	return nil
}

func releaseInputs(root, target string) (posemodel.ReleasePolicy, []posemodel.ReleaseFragment, string, map[string]string, error) {
	if err := posemodel.ValidateReleaseVersion(target); err != nil {
		return posemodel.ReleasePolicy{}, nil, "", nil, err
	}
	policy, err := posemodel.LoadReleasePolicy(root)
	if err != nil {
		return policy, nil, "", nil, err
	}
	fragments, err := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", "unreleased"))
	if err != nil {
		return policy, nil, "", nil, err
	}
	if len(fragments) == 0 && !policy.AllowEmpty {
		return policy, nil, "", nil, fmt.Errorf("empty release is forbidden by policy")
	}
	for _, fragment := range fragments {
		if _, err := os.Stat(filepath.Join(root, ".pose", "specs", fragment.Spec, "spec.md")); err != nil {
			return policy, nil, "", nil, fmt.Errorf("fragment %s references missing spec", fragment.Spec)
		}
	}
	previous := ""
	entries, _ := os.ReadDir(filepath.Join(root, ".pose", "releases"))
	for _, entry := range entries {
		if entry.IsDir() && posemodel.ValidateReleaseVersion(entry.Name()) == nil && entry.Name() != target {
			if previous == "" || versionLess(previous, entry.Name()) {
				previous = entry.Name()
			}
		}
	}
	tags := exec.Command("git", "tag", "--list", "v*")
	tags.Dir = root
	if raw, gitErr := tags.Output(); gitErr == nil {
		for _, tag := range strings.Fields(string(raw)) {
			if posemodel.ValidateReleaseVersion(tag) == nil && tag != target && (previous == "" || versionLess(previous, tag)) {
				previous = tag
			}
		}
	}
	evidence := map[string]string{"source": "pose-mcp/internal/version/version.go", "value": "v" + version.ReleaseBase()}
	if evidence["value"] != target {
		return policy, nil, "", nil, fmt.Errorf("target %s differs from authoritative version evidence %s", target, evidence["value"])
	}
	if previous != "" && !versionLess(previous, target) {
		return policy, nil, "", nil, fmt.Errorf("target %s must be newer than %s", target, previous)
	}
	return policy, fragments, previous, evidence, nil
}

func versionLess(a, b string) bool {
	parts := func(v string) []int {
		v = strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v")
		out := []int{0, 0, 0}
		for i, s := range strings.Split(v, ".") {
			if i < 3 {
				fmt.Sscanf(s, "%d", &out[i])
			}
		}
		return out
	}
	x, y := parts(a), parts(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return a < b
}

func cmdReleasePlan(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, map[string]bool{"--json": true}); err != nil {
		return usageError(stderr, err.Error())
	}
	target, ok, err := releaseArg(args, "--version")
	if err != nil || !ok {
		return usageError(stderr, "pose release plan: --version is required")
	}
	policy, fragments, previous, evidence, err := releaseInputs(root, target)
	if err != nil {
		fmt.Fprintf(stderr, "pose release plan: %v\n", err)
		return 1
	}
	manifest := posemodel.NewReleaseManifest(target, previous, "", fragments, policy, evidence)
	recommendation := "patch"
	for _, f := range fragments {
		if f.Breaking {
			recommendation = "major"
			break
		}
		if f.Category == "added" {
			recommendation = "minor"
		}
	}
	result := map[string]any{"version": target, "previous_version": previous, "fragment_count": len(fragments), "specs": manifest.Specs, "breaking": manifest.Breaking, "recommendation": recommendation, "release_input_digest": manifest.ReleaseInputDigest, "dry_run": true, "blockers": []string{}}
	if releaseFlag(args, "--json") {
		raw, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(raw))
	} else {
		fmt.Fprintf(stdout, "Release plan %s: %d fragments, recommendation=%s, digest=%s (dry-run)\n", target, len(fragments), recommendation, manifest.ReleaseInputDigest)
	}
	return 0
}

func cmdReleasePrepare(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, map[string]bool{"--apply": true, "--json": true}); err != nil {
		return usageError(stderr, err.Error())
	}
	target, ok, _ := releaseArg(args, "--version")
	if !ok {
		return usageError(stderr, "pose release prepare: --version is required")
	}
	policy, fragments, previous, evidence, err := releaseInputs(root, target)
	if err != nil {
		fmt.Fprintf(stderr, "pose release prepare: %v\n", err)
		return 1
	}
	manifest := posemodel.NewReleaseManifest(target, previous, time.Now().UTC().Format(time.RFC3339), fragments, policy, evidence)
	if existing, err := posemodel.LoadReleaseManifest(root, target); err == nil {
		if existing.ReleaseInputDigest == manifest.ReleaseInputDigest {
			fmt.Fprintf(stdout, "Release %s already prepared (idempotent).\n", target)
			return 0
		}
		fmt.Fprintln(stderr, "pose release prepare: immutable candidate already exists with different inputs")
		return 1
	}
	if !releaseFlag(args, "--apply") {
		fmt.Fprintf(stdout, "Would prepare %s with %d fragments (dry-run); rerun with --apply.\n", target, len(fragments))
		return 0
	}
	releaseDir := filepath.Join(root, ".pose", "releases", target)
	archiveDir := filepath.Join(root, ".pose", "changelogs", target)
	notesPath := filepath.Join(root, ".pose", "changelogs", target+".md")
	if _, err := os.Stat(archiveDir); err == nil {
		fmt.Fprintln(stderr, "pose release prepare: archive already exists without matching manifest")
		return 1
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		os.RemoveAll(releaseDir)
		fmt.Fprintln(stderr, err)
		return 1
	}
	rollback := func() {
		for _, f := range fragments {
			from := filepath.Join(archiveDir, f.Path)
			to := filepath.Join(root, ".pose", "changelogs", "unreleased", f.Path)
			if _, e := os.Stat(from); e == nil {
				_ = os.Rename(from, to)
			}
		}
		_ = os.Remove(notesPath)
		_ = os.RemoveAll(archiveDir)
		_ = os.RemoveAll(releaseDir)
	}
	for _, f := range fragments {
		from := filepath.Join(root, ".pose", "changelogs", "unreleased", f.Path)
		to := filepath.Join(archiveDir, f.Path)
		if err := os.Rename(from, to); err != nil {
			rollback()
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	notes := posemodel.RenderReleaseNotes(target, fragments)
	if err := os.WriteFile(notesPath, []byte(notes), 0o644); err != nil {
		rollback()
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "manifest.json"), posemodel.CanonicalJSON(manifest), 0o644); err != nil {
		rollback()
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Prepared %s: %d fragments archived; notes and manifest frozen.\n", target, len(fragments))
	return 0
}

func checkRelease(root, target string) ([]string, *posemodel.ReleaseManifest) {
	gaps := []string{}
	manifest, err := posemodel.LoadReleaseManifest(root, target)
	if err != nil {
		return []string{"manifest missing or invalid: " + err.Error()}, nil
	}
	policy, err := posemodel.LoadReleasePolicy(root)
	if err != nil {
		gaps = append(gaps, err.Error())
	} else if manifest.PolicyDigest != posemodel.ReleaseDigest(policy) {
		gaps = append(gaps, "policy digest differs from prepared manifest")
	}
	archive, err := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", target))
	if err != nil {
		gaps = append(gaps, err.Error())
	}
	notes, err := os.ReadFile(filepath.Join(root, ".pose", "changelogs", target+".md"))
	if err != nil {
		gaps = append(gaps, "canonical notes missing")
	} else if posemodel.ReleaseDigest(string(notes)) != manifest.NotesDigest {
		gaps = append(gaps, "canonical notes digest mismatch")
	}
	if len(archive) != len(manifest.Fragments) {
		gaps = append(gaps, "archived fragment count differs from manifest")
	}
	bySpec := map[string]string{}
	for _, f := range archive {
		bySpec[f.Spec] = f.Digest
	}
	for _, f := range manifest.Fragments {
		if bySpec[f.Spec] != f.Digest {
			gaps = append(gaps, "fragment digest mismatch: "+f.Spec)
		}
	}
	if policy, err := posemodel.LoadReleasePolicy(root); err == nil {
		rebuilt := posemodel.NewReleaseManifest(manifest.Version, manifest.PreviousVersion, manifest.PreparedAt, archive, policy, manifest.VersionEvidence)
		if rebuilt.ReleaseInputDigest != manifest.ReleaseInputDigest {
			gaps = append(gaps, "release input digest mismatch")
		}
	}
	pending, _ := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", "unreleased"))
	for _, p := range pending {
		if _, ok := bySpec[p.Spec]; ok {
			gaps = append(gaps, "fragment exists in pending and released locations: "+p.Spec)
		}
	}
	events, err := posemodel.LoadReleaseEvents(root, target)
	if err != nil {
		gaps = append(gaps, err.Error())
	} else {
		projection := posemodel.ProjectRelease(manifest, events)
		gaps = append(gaps, projection.Gaps...)
	}
	return gaps, manifest
}

func cmdReleaseCheck(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, map[string]bool{"--strict": true, "--json": true}); err != nil {
		return usageError(stderr, err.Error())
	}
	target, ok, _ := releaseArg(args, "--version")
	if !ok {
		return usageError(stderr, "pose release check: --version is required")
	}
	gaps, manifest := checkRelease(root, target)
	tagCommit, tagged := resolveReleaseTag(root, target)
	if tagged && manifest == nil {
		gaps = append(gaps, "tag exists without prepared manifest")
	}
	if releaseFlag(args, "--json") {
		raw, _ := json.MarshalIndent(map[string]any{"version": target, "valid": len(gaps) == 0, "tagged": tagged, "tag_commit": tagCommit, "gaps": gaps}, "", "  ")
		fmt.Fprintln(stdout, string(raw))
	} else if len(gaps) == 0 {
		fmt.Fprintf(stdout, "Release %s: valid prepared snapshot; tagged=%t.\n", target, tagged)
	} else {
		for _, gap := range gaps {
			fmt.Fprintln(stderr, "release:", gap)
		}
	}
	if len(gaps) > 0 && releaseFlag(args, "--strict") {
		return 1
	}
	return 0
}

func resolveReleaseTag(root, target string) (string, bool) {
	cmd := exec.Command("git", "rev-list", "-n", "1", target+"^{}")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return "", false
	}
	commit := strings.TrimSpace(string(raw))
	return commit, immutableCommitRE.MatchString(commit)
}

func cmdReleaseNotesSnapshot(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, nil); err != nil {
		return usageError(stderr, err.Error())
	}
	target, ok, _ := releaseArg(args, "--version")
	if !ok {
		return usageError(stderr, "pose release notes: --version is required")
	}
	if err := posemodel.ValidateReleaseVersion(target); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "changelogs", target+".md"))
	if err != nil {
		fmt.Fprintln(stderr, "pose release notes: immutable prepared snapshot not found")
		return 1
	}
	_, manifest := checkRelease(root, target)
	if manifest == nil || posemodel.ReleaseDigest(string(raw)) != manifest.NotesDigest {
		fmt.Fprintln(stderr, "pose release notes: stale or invalid snapshot")
		return 1
	}
	_, _ = stdout.Write(raw)
	return 0
}

func validateReleaseEvidence(event, target string, e posemodel.ReleaseEvidence) error {
	if e.SchemaVersion != 1 || e.Version != target || e.Tag != target || e.Provider == "" || e.Repository == "" || !immutableCommitRE.MatchString(e.Commit) {
		return fmt.Errorf("evidence identity must include schema, provider, repository, version/tag and immutable commit")
	}
	if event == "published" || event == "verified" {
		if e.URL == "" || e.PublishedAt == "" || len(e.Assets) == 0 {
			return fmt.Errorf("publication/verification evidence requires URL, time and assets")
		}
		for name, digest := range e.Assets {
			if name == "" || !assetDigestRE.MatchString(digest) {
				return fmt.Errorf("invalid asset digest for %s", name)
			}
		}
	}
	if event == "verified" && e.Publication == "" {
		return fmt.Errorf("verified evidence requires publication_digest")
	}
	return nil
}

func cmdReleaseRecord(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true, "--event": true, "--evidence": true}, nil); err != nil {
		return usageError(stderr, err.Error())
	}
	target, ok, _ := releaseArg(args, "--version")
	if !ok {
		return usageError(stderr, "pose release record: --version required")
	}
	state, ok, _ := releaseArg(args, "--event")
	if !ok || !map[string]bool{"tagged": true, "published": true, "verified": true, "failed": true, "yanked": true}[state] {
		return usageError(stderr, "pose release record: legal --event required")
	}
	evidencePath, ok, _ := releaseArg(args, "--evidence")
	if !ok {
		return usageError(stderr, "pose release record: --evidence required")
	}
	full, err := confinedProjectPath(root, evidencePath)
	if err != nil {
		fmt.Fprintln(stderr, "pose release record: evidence path must remain inside project")
		return 2
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var evidence posemodel.ReleaseEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := validateReleaseEvidence(state, target, evidence); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if commit, tagged := resolveReleaseTag(root, target); (state == "tagged" || state == "published" || state == "verified") && (!tagged || commit != evidence.Commit) {
		fmt.Fprintln(stderr, "evidence commit does not match the immutable Git tag")
		return 1
	}
	manifest, err := posemodel.LoadReleaseManifest(root, target)
	if err != nil {
		fmt.Fprintln(stderr, "prepared manifest required")
		return 1
	}
	_ = manifest
	events, err := posemodel.LoadReleaseEvents(root, target)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	candidate := posemodel.NewReleaseEvent(target, state, evidence, time.Now())
	for _, event := range events {
		if event.State == state && event.EvidenceDigest == candidate.EvidenceDigest {
			fmt.Fprintln(stdout, "Release event already recorded (idempotent).")
			return 0
		}
		if event.State == state && event.EvidenceDigest != candidate.EvidenceDigest {
			fmt.Fprintln(stderr, "conflicting event evidence")
			return 1
		}
	}
	test := append(append([]posemodel.ReleaseEvent{}, events...), candidate)
	projection := posemodel.ProjectRelease(manifest, test)
	if len(projection.Gaps) > 0 {
		fmt.Fprintln(stderr, "illegal transition:", projection.Gaps[len(projection.Gaps)-1])
		return 1
	}
	path := filepath.Join(root, ".pose", "releases", target, "events.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer file.Close()
	line, _ := json.Marshal(candidate)
	if _, err := file.Write(append(line, '\n')); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Recorded %s for %s.\n", state, target)
	return 0
}

func confinedProjectPath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute path")
	}
	full := filepath.Join(root, filepath.Clean(path))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("escape")
	}
	return full, nil
}

func cmdReleaseStatus(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, map[string]bool{"--json": true}); err != nil {
		return usageError(stderr, err.Error())
	}
	target, _, _ := releaseArg(args, "--version")
	if target != "" && posemodel.ValidateReleaseVersion(target) != nil {
		return usageError(stderr, "invalid release version")
	}
	status, err := (posemodel.Store{Root: root}).GetReleaseStatus(target)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if releaseFlag(args, "--json") {
		raw, _ := json.MarshalIndent(status, "", "  ")
		fmt.Fprintln(stdout, string(raw))
		return 0
	}
	fmt.Fprintf(stdout, "Pending fragments: %d\n", len(status.Pending))
	for _, release := range status.Releases {
		fmt.Fprintf(stdout, "%s\t%s", release.Version, release.State)
		if len(release.Gaps) > 0 {
			fmt.Fprintf(stdout, "\tgaps=%s", strings.Join(release.Gaps, "; "))
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func cmdReleaseOpenNext(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, nil); err != nil {
		return usageError(stderr, err.Error())
	}
	next, ok, _ := releaseArg(args, "--version")
	if !ok || posemodel.ValidateReleaseVersion(next) != nil {
		return usageError(stderr, "pose release open-next: valid --version required")
	}
	status, err := (posemodel.Store{Root: root}).GetReleaseStatus("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(status.Releases) == 0 || status.Releases[0].State != "verified" {
		fmt.Fprintln(stderr, "pose release open-next: latest release is not verified")
		return 1
	}
	if !versionLess(status.Releases[0].Version, next) {
		fmt.Fprintln(stderr, "pose release open-next: next version must be greater than latest verified release")
		return 1
	}
	fmt.Fprintf(stdout, "Next-cycle plan: %s after verified %s (no mutation). Update project version evidence in reviewed work.\n", next, status.Releases[0].Version)
	return 0
}

func cmdReleaseBackfill(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, nil, map[string]bool{"--from-git": true, "--json": true, "--apply": true}); err != nil {
		return usageError(stderr, err.Error())
	}
	if !releaseFlag(args, "--from-git") {
		return usageError(stderr, "pose release backfill: --from-git required")
	}
	cmd := exec.Command("git", "tag", "--list", "v*")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	tags := strings.Fields(string(raw))
	sort.Strings(tags)
	type row struct {
		Version    string   `json:"version"`
		Archive    bool     `json:"archive"`
		Manifest   bool     `json:"manifest"`
		Confidence string   `json:"confidence"`
		Gaps       []string `json:"gaps"`
	}
	rows := []row{}
	for _, tag := range tags {
		r := row{Version: tag, Confidence: "low", Gaps: []string{}}
		if info, err := os.Stat(filepath.Join(root, ".pose", "changelogs", tag)); err == nil && info.IsDir() {
			r.Archive = true
			r.Confidence = "medium"
		} else {
			r.Gaps = append(r.Gaps, "missing local fragment archive")
		}
		if _, err := os.Stat(filepath.Join(root, ".pose", "releases", tag, "manifest.json")); err == nil {
			r.Manifest = true
			r.Confidence = "high"
		} else {
			r.Gaps = append(r.Gaps, "missing manifest; publication evidence not imported")
		}
		rows = append(rows, r)
	}
	if releaseFlag(args, "--apply") {
		fmt.Fprintln(stderr, "pose release backfill: apply cannot fabricate historical manifests; review the dry-run inventory")
		return 1
	}
	if releaseFlag(args, "--json") {
		out, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Fprintln(stdout, string(out))
	} else {
		for _, r := range rows {
			fmt.Fprintf(stdout, "%s confidence=%s archive=%t manifest=%t gaps=%s\n", r.Version, r.Confidence, r.Archive, r.Manifest, strings.Join(r.Gaps, "; "))
		}
	}
	return 0
}

func appendReleasePolicyChecks(checker *nativeChecker) {
	if _, err := os.Stat(filepath.Join(checker.root, ".pose", "release-policy.json")); os.IsNotExist(err) {
		return
	}
	policy, err := posemodel.LoadReleasePolicy(checker.root)
	if err != nil {
		checker.failOrWarn("release: " + err.Error())
		return
	}
	_ = policy
	seen := map[string]string{}
	base := filepath.Join(checker.root, ".pose", "changelogs")
	if pending, err := posemodel.LoadReleaseFragments(filepath.Join(base, "unreleased")); err == nil {
		for _, fragment := range pending {
			seen[fragment.Spec] = "unreleased"
		}
	}
	entries, _ := os.ReadDir(base)
	for _, entry := range entries {
		if !entry.IsDir() || posemodel.ValidateReleaseVersion(entry.Name()) != nil {
			continue
		}
		fragments, err := posemodel.LoadReleaseFragments(filepath.Join(base, entry.Name()))
		if err != nil {
			checker.failOrWarn("release: " + err.Error())
			continue
		}
		for _, fragment := range fragments {
			if prior := seen[fragment.Spec]; prior != "" {
				checker.failOrWarn("release: fragment " + fragment.Spec + " assigned to " + prior + " and " + entry.Name())
			}
			seen[fragment.Spec] = entry.Name()
		}
	}
	cmd := exec.Command("git", "tag", "--list", "v*")
	cmd.Dir = checker.root
	raw, _ := cmd.Output()
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		tag := scanner.Text()
		if policy.LegacyCutoff != "" && !versionLess(policy.LegacyCutoff, tag) {
			continue
		}
		if _, err := posemodel.LoadReleaseManifest(checker.root, tag); err != nil {
			checker.failOrWarn("release: governed tag " + tag + " has no manifest")
		}
	}
}
