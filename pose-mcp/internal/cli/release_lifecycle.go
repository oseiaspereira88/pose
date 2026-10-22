package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
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
	if policy.AdoptedAt == "" || policy.Provider == "" || policy.Repository == "" {
		return policy, nil, "", nil, fmt.Errorf("release policy is not adopted: set adopted_at, provider and repository in .pose/policy/release.json")
	}
	fragments, err := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", "unreleased"), posemodel.LoadChangelogPolicy(root))
	if err != nil {
		return policy, nil, "", nil, err
	}
	if len(fragments) == 0 && !policy.AllowEmpty {
		return policy, nil, "", nil, fmt.Errorf("empty release is forbidden by policy")
	}
	store := posemodel.Store{Root: root}
	for _, fragment := range fragments {
		if _, err := store.GetSpec(fragment.Spec); err != nil {
			return policy, nil, "", nil, fmt.Errorf("fragment %s references missing spec", fragment.Spec)
		}
	}
	previous := ""
	releasesDir := filepath.Join(root, ".pose", "releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		// Only a directory proven not to be there is a project with no earlier release.
		if absent, absentErr := posemodel.ProvenAbsent(releasesDir); absentErr != nil {
			return policy, nil, "", nil, fmt.Errorf("the release directory could not be read: %w", absentErr)
		} else if !absent {
			return policy, nil, "", nil, fmt.Errorf("the release directory could not be read: %w", err)
		}
	}
	for _, entry := range entries {
		if entry.IsDir() && posemodel.ValidateReleaseVersion(entry.Name()) == nil && entry.Name() != target {
			if previous == "" || versionLess(previous, entry.Name()) {
				previous = entry.Name()
			}
		}
	}
	raw, gitErr := posemodel.ListReleaseTags(root)
	switch {
	case gitErr == nil:
		for _, tag := range strings.Fields(string(raw)) {
			if posemodel.ValidateReleaseVersion(tag) == nil && tag != target && (previous == "" || versionLess(previous, tag)) {
				previous = tag
			}
		}
	case !gitUnavailableAt(root):
		// A repository whose tags cannot be listed has releases this cut cannot see.
		return policy, nil, "", nil, fmt.Errorf("the release tags could not be listed: %w", gitErr)
	}
	evidence, err := releaseVersionEvidence(root, policy)
	if err != nil {
		return policy, nil, "", nil, err
	}
	if evidence["value"] != target {
		return policy, nil, "", nil, fmt.Errorf("target %s differs from authoritative version evidence %s", target, evidence["value"])
	}
	if previous != "" && !versionLess(previous, target) {
		return policy, nil, "", nil, fmt.Errorf("target %s must be newer than %s", target, previous)
	}
	return policy, fragments, previous, evidence, nil
}

// releaseVersionEvidence selects the authoritative version a cut is compared
// with: the file the policy declares, or, only inside the engine repository, the
// compiled engine version. Elsewhere the engine's number belongs to another
// project, so it is refused instead of silently accepted.
func releaseVersionEvidence(root string, policy posemodel.ReleasePolicy) (map[string]string, error) {
	return posemodel.ReleaseVersionEvidence(root, policy)
}

// versionLess orders release versions by Semantic Versioning precedence: rc.2 is before rc.10
// and a release candidate is before its final version.
func versionLess(a, b string) bool { return posemodel.CompareReleasePrecedence(a, b) < 0 }

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
	manifest := posemodel.NewReleaseManifest(target, previous, "", fragments, policy, posemodel.LoadChangelogPolicy(root), evidence)
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
	blockers, err := posemodel.Store{Root: root}.ForeignFragmentClaims(fragments)
	if err != nil {
		render(stdout, stderr).Failure("pose release plan: " + err.Error())
		return 1
	}
	if blockers == nil {
		blockers = []string{}
	}
	result := map[string]any{"version": target, "previous_version": previous, "fragment_count": len(fragments), "specs": manifest.Specs, "breaking": manifest.Breaking, "recommendation": recommendation, "release_input_digest": manifest.ReleaseInputDigest, "dry_run": true, "blockers": blockers}
	if releaseFlag(args, "--json") {
		raw, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(raw))
	} else {
		fmt.Fprintf(stdout, "Release plan %s: %d fragments, recommendation=%s, digest=%s (dry-run)\n", target, len(fragments), recommendation, manifest.ReleaseInputDigest)
		for _, blocker := range blockers {
			render(stdout, stderr).Field("release.blocker", blocker)
		}
	}
	return 0
}

func cmdReleasePrepare(root string, args []string, stdout, stderr io.Writer) int {
	if err := rejectReleaseArgs(args, map[string]bool{"--version": true}, map[string]bool{"--apply": true, "--json": true, "--allow-moved-claims": true}); err != nil {
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
	manifest := posemodel.NewReleaseManifest(target, previous, time.Now().UTC().Format(time.RFC3339), fragments, policy, posemodel.LoadChangelogPolicy(root), evidence)
	existing, loadErr := posemodel.LoadReleaseManifest(root, target)
	switch {
	case loadErr == nil:
		if existing.ReleaseInputDigest == manifest.ReleaseInputDigest {
			fmt.Fprintf(stdout, "Release %s already prepared (idempotent).\n", target)
			return 0
		}
		fmt.Fprintln(stderr, "pose release prepare: immutable candidate already exists with different inputs")
		return 1
	case !errors.Is(loadErr, os.ErrNotExist) || manifestIsNotProvenAbsent(root, target):
		// A manifest that is there and cannot be read as one (invalid, or a link whose target was
		// lost) is a candidate someone must look at, not a place to write over.
		render(io.Discard, stderr).Failure(fmt.Sprintf("pose release prepare: a manifest of %s exists and is not valid; it is not overwritten: %v", target, loadErr))
		return 1
	}
	// Governed effects (spec pose-governed-effect-enforcement): under the
	// adopted capability an unsatisfied action request restricting release
	// of one of these specs refuses the freeze, and the dry-run says so.
	releaseSpecs := []string{}
	for _, fragment := range fragments {
		releaseSpecs = append(releaseSpecs, fragment.Spec)
	}
	refusals, err := posemodel.Store{Root: root}.ReleaseGovernedRefusal(releaseSpecs)
	if err != nil {
		render(stdout, stderr).Failure("pose release prepare: " + err.Error())
		return 1
	}
	if len(refusals) > 0 {
		out := render(stdout, stderr)
		for _, refusal := range refusals {
			out.Field("release.refused."+refusal.Code, refusal.Message)
		}
		out.Failure(fmt.Sprintf("pose release prepare: %d action request(s) restrict this release", len(refusals)))
		return 1
	}
	// A fragment another spec claims would leave that claim pointing at a
	// path the freeze removes (spec pose-release-prepare-guards-claimed-fragments).
	foreign, err := posemodel.Store{Root: root}.ForeignFragmentClaims(fragments)
	if err != nil {
		render(stdout, stderr).Failure("pose release prepare: " + err.Error())
		return 1
	}
	if len(foreign) > 0 && releaseFlag(args, "--allow-moved-claims") {
		// Allowed, still named: the operator sees what the freeze leaves
		// pointing at a moved path (found in review).
		for _, claim := range foreign {
			render(stdout, stderr).Field("release.allowed.moved-claim", claim)
		}
	}
	if len(foreign) > 0 && !releaseFlag(args, "--allow-moved-claims") {
		out := render(stdout, stderr)
		for _, claim := range foreign {
			out.Field("release.refused.moved-claim", claim)
		}
		out.Failure(fmt.Sprintf("pose release prepare: %d claim(s) name fragments this release moves; fix them first, or pass --allow-moved-claims", len(foreign)))
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
	// Prepare never edits a spec. A spec that declared its fragment keeps the
	// claim it wrote, and the manifest written below is what attests where the
	// fragment went (spec pose-release-archival-attested-by-the-ledger).
	// Rewriting the claim, as prepare once did, failed the action check on
	// every released spec and changed the sealed review subject of any spec
	// closed before the cut.
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
	notes := posemodel.RenderReleaseNotes(target, fragments, posemodel.LoadChangelogPolicy(root))
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
		return append([]string{"manifest missing or invalid: " + err.Error()}, releaseTagSnapshotGaps(root, target)...), nil
	}
	tagGaps, tagged := releaseTagState(root, target)
	if tagged {
		// A tagged release is judged under the inputs it was cut with: the policy and the
		// fragments of its tag. The working tree must be that same cut; it is not compared with
		// the policy the project has today, which can legitimately have moved since.
		gaps = append(gaps, tagGaps...)
		gaps = append(gaps, taggedReleaseWorktreeGaps(root, target, manifest)...)
	} else {
		gaps = append(gaps, tagGaps...)
		gaps = append(gaps, preparedReleaseGaps(root, target, manifest)...)
	}
	pending, perr := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", "unreleased"), posemodel.LoadChangelogPolicy(root))
	if perr != nil {
		gaps = append(gaps, "pending fragments could not be read: "+perr.Error())
	}
	archived := map[string]bool{}
	for _, f := range manifest.Fragments {
		archived[f.Spec] = true
	}
	for _, p := range pending {
		if archived[p.Spec] {
			gaps = append(gaps, "fragment exists in pending and released locations: "+p.Spec)
		}
	}
	events, err := posemodel.LoadReleaseEvents(root, target)
	if err != nil {
		gaps = append(gaps, err.Error())
	} else {
		projection := posemodel.ProjectRelease(manifest, events)
		gaps = append(gaps, projection.Gaps...)
		// The events state that the tag exists from "tagged" on; a repository without it
		// contradicts them, and the absence is not a candidate that was never tagged.
		if !tagged && projection.State != "prepared" && projection.State != "failed" {
			gaps = append(gaps, "the events record "+projection.State+", and the tag "+target+" does not exist")
		}
	}
	return gaps, manifest
}

// preparedReleaseGaps judges a cut that has no tag yet; the domain does it, for the status as well.
func preparedReleaseGaps(root, target string, manifest *posemodel.ReleaseManifest) []string {
	gaps, _ := posemodel.ReleaseCandidateGaps(root, target, manifest)
	return gaps
}

// taggedReleaseWorktreeGaps binds the working tree's copy of a tagged release to the cut the tag
// holds (the domain checks it; the status uses the same function).
func taggedReleaseWorktreeGaps(root, target string, manifest *posemodel.ReleaseManifest) []string {
	snap, err := posemodel.ReadReleaseSnapshot(root, target)
	if err != nil {
		return nil // reported by releaseTagState
	}
	return posemodel.TaggedReleaseWorktreeGaps(root, target, manifest, snap)
}

// releaseTagState says whether target is a tag, and what is wrong with the snapshot it holds. A
// tag that cannot be read is neither tagged nor absent: it is a gap.
func releaseTagState(root, target string) (gaps []string, tagged bool) {
	if _, _, err := posemodel.ResolveReleaseTag(root, target); err != nil {
		if errors.Is(err, posemodel.ErrReleaseTagMissing) {
			return nil, false
		}
		return []string{"the tag " + target + " could not be resolved: " + err.Error()}, false
	}
	return releaseTagSnapshotGaps(root, target), true
}

func gitUnavailableAt(root string) bool {
	// Not a repository at all: there are no tags to list, which is not a failure to list them. Any
	// other failure of git, a repository whose gitdir was lost among them, is a failure.
	return posemodel.GitNotARepository(root)
}

// releaseTagSnapshotGaps verifies, when the release has a tag, the snapshot the tag points at: the
// manifest, the notes and the frozen fragments are read from the tagged tree, never from the working tree
// (spec pose-release-manifest-tag-integrity). A release with no tag has no snapshot to verify.
func releaseTagSnapshotGaps(root, target string) []string {
	if _, _, err := posemodel.ResolveReleaseTag(root, target); err != nil {
		return nil
	}
	snap, err := posemodel.ReadReleaseSnapshot(root, target)
	if err != nil {
		return []string{"tag " + target + " has no release snapshot in its tree: " + err.Error()}
	}
	gaps := []string{}
	for _, issue := range snap.Issues {
		gaps = append(gaps, "tag snapshot manifest "+issue.String())
	}
	for _, gap := range snap.Gaps {
		gaps = append(gaps, "tag snapshot: "+gap)
	}
	return gaps
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
	// On a prepared tree the fragments sit in the archive; a claim at their
	// unreleased path by another spec is what CI's structural gate rejects.
	if archived, err := posemodel.LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", target), posemodel.LoadChangelogPolicy(root)); err == nil {
		if foreign, err := (posemodel.Store{Root: root}).ForeignFragmentClaims(archived); err == nil {
			gaps = append(gaps, foreign...)
		}
	}
	tagCommit, tagged := resolveReleaseTag(root, target)
	if tagged && manifest == nil {
		gaps = append(gaps, "tag exists without prepared manifest")
	}
	var tagLimits []string
	if tagged {
		if snap, err := posemodel.ReadReleaseSnapshot(root, target); err == nil {
			tagLimits = snap.Limits
		}
	}
	if releaseFlag(args, "--json") {
		raw, _ := json.MarshalIndent(map[string]any{"version": target, "valid": len(gaps) == 0, "tagged": tagged, "tag_commit": tagCommit, "gaps": gaps, "snapshot_limits": tagLimits}, "", "  ")
		fmt.Fprintln(stdout, string(raw))
	} else if len(gaps) == 0 {
		fmt.Fprintf(stdout, "Release %s: valid prepared snapshot; tagged=%t.\n", target, tagged)
		for _, limit := range tagLimits {
			render(stdout, stderr).Field("release.not_reverified", limit)
		}
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

// resolveReleaseTag is the commit a release tag names. Only refs/tags counts: a branch of the same
// name is not a tag, and an object that is not a commit is not a release.
func resolveReleaseTag(root, target string) (string, bool) {
	commit, _, err := posemodel.ResolveReleaseTag(root, target)
	if err != nil {
		return "", false
	}
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
	gaps, manifest := checkRelease(root, target)
	if manifest == nil || len(gaps) > 0 || posemodel.ReleaseDigest(string(raw)) != manifest.NotesDigest {
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
			if name == "" || filepath.Base(name) != name || !assetDigestRE.MatchString(digest) {
				return fmt.Errorf("invalid asset digest for %s", name)
			}
		}
		parsed, err := url.Parse(e.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("publication URL must be canonical HTTPS without credentials")
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
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
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
	if state == "tagged" || state == "published" || state == "verified" {
		// The event states that the tag is the release: the tag's own tree must hold an intact
		// snapshot, and the working tree's copy must be that same cut.
		if gaps, _ := releaseTagState(root, target); len(gaps) > 0 {
			render(io.Discard, stderr).Failure("the tag does not hold an intact release snapshot: " + strings.Join(gaps, "; "))
			return 1
		}
		if gaps := taggedReleaseWorktreeGaps(root, target, manifest); len(gaps) > 0 {
			render(io.Discard, stderr).Failure("the working tree is not the cut the tag holds: " + strings.Join(gaps, "; "))
			return 1
		}
	}
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
	return posemodel.ConfinedProjectPath(root, path)
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
	if len(status.Releases) == 0 || status.Releases[0].State != "verified" || !status.Releases[0].Sound() {
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
	raw, err := posemodel.ListReleaseTags(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	tags := strings.Fields(string(raw))
	sort.Slice(tags, func(i, j int) bool { return posemodel.CompareReleasePrecedence(tags[i], tags[j]) < 0 })
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
		path := filepath.Join(root, ".pose", "reports", "release-backfill.json")
		if err := writeAtomic(path, posemodel.CanonicalJSON(rows), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Recorded reviewed backfill inventory at %s; no release facts were fabricated.\n", path)
		return 0
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
	hasPolicy := false
	for _, rel := range []string{".pose/policy/release.json", ".pose/release-policy.json"} {
		absent, absentErr := posemodel.ProvenAbsent(filepath.Join(checker.root, filepath.FromSlash(rel)))
		if absentErr != nil {
			// A policy that is there as a link whose target was lost is a policy that was lost.
			checker.failOrWarn("release: " + absentErr.Error())
			return
		}
		hasPolicy = hasPolicy || !absent
	}
	if !hasPolicy {
		// A policy that is gone now is not a policy that never was: a cut in the working tree or
		// a tag holding a manifest still witnesses a project that adopted governed releases.
		if witness, err := governedReleaseWitness(checker.root); err != nil {
			checker.failOrWarn("release: the release policy is missing and whether the project adopted governed releases could not be told: " + err.Error())
		} else if witness != "" {
			checker.failOrWarn("release: the release policy is missing, and " + witness + " shows the project adopted governed releases")
		}
		return
	}
	policy, err := posemodel.LoadReleasePolicy(checker.root)
	if err != nil {
		checker.failOrWarn("release: " + err.Error())
		return
	}
	if policy.AdoptedAt == "" {
		return
	}
	_ = policy
	seen := map[string]string{}
	base := filepath.Join(checker.root, ".pose", "changelogs")
	if pending, err := posemodel.LoadReleaseFragments(filepath.Join(base, "unreleased"), posemodel.LoadChangelogPolicy(checker.root)); err != nil {
		checker.failOrWarn("release: the pending fragments could not be read: " + err.Error())
	} else {
		for _, fragment := range pending {
			seen[fragment.Spec] = "unreleased"
		}
	}
	entries, readErr := os.ReadDir(base)
	if readErr != nil {
		if absent, absentErr := posemodel.ProvenAbsent(base); absentErr != nil || !absent {
			checker.failOrWarn("release: the changelog directory could not be read: " + readErr.Error())
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() || posemodel.ValidateReleaseVersion(entry.Name()) != nil {
			continue
		}
		if policy.LegacyCutoff != "" && !versionLess(policy.LegacyCutoff, entry.Name()) {
			// Releases at or before the cutoff predate the governance; the policy leaves them out,
			// and a spec they name can legitimately be delivered again by a governed one.
			continue
		}
		fragments, err := posemodel.LoadArchivedFragmentIdentities(filepath.Join(base, entry.Name()))
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
	raw, listErr := posemodel.ListReleaseTags(checker.root)
	if listErr != nil && !gitUnavailableAt(checker.root) {
		checker.failOrWarn("release: the release tags could not be listed: " + listErr.Error())
	}
	// The cut of every governed release in the working tree is judged the way `release check`
	// judges it: a candidate against its inputs, a tagged one against the cut its tag holds.
	releasesDir := filepath.Join(checker.root, ".pose", "releases")
	releases, relErr := os.ReadDir(releasesDir)
	if relErr != nil {
		// The inventory of the cuts is read, or its absence is proven; a lost one is not "no cuts".
		if absent, absentErr := posemodel.ProvenAbsent(releasesDir); absentErr != nil || !absent {
			if absentErr == nil {
				absentErr = relErr
			}
			checker.failOrWarn("release: the release inventory could not be read: " + absentErr.Error())
		}
	}
	judged := map[string]bool{}
	for _, release := range releases {
		if posemodel.ValidateReleaseVersion(release.Name()) != nil || (policy.LegacyCutoff != "" && !versionLess(policy.LegacyCutoff, release.Name())) {
			continue
		}
		judged[release.Name()] = true
		gaps, manifest := checkRelease(checker.root, release.Name())
		if manifest == nil || len(gaps) > 0 {
			if manifest != nil {
				if _, tagged := releaseTagState(checker.root, release.Name()); !tagged && neverHadAnEvent(checker.root, release.Name()) && posemodel.ReleaseCandidateSuperseded(checker.root, manifest) {
					// A prepared cut the project's version has passed can no longer be tagged: it
					// is said, not failed, and its inputs are not judged against today's.
					checker.issue(cliout.StateWarning, "release: "+release.Name()+" is a prepared cut that the project's version has passed; it cannot be tagged: "+strings.Join(gaps, "; "))
					continue
				}
			}
			checker.failOrWarn("release: " + release.Name() + " is not a sound cut: " + strings.Join(gaps, "; "))
		}
	}
	// A tag name has no whitespace, and a scanner would stop silently at a line past its buffer.
	for _, tag := range strings.Fields(string(raw)) {
		if policy.LegacyCutoff != "" && !versionLess(policy.LegacyCutoff, tag) {
			continue
		}
		if snap, err := posemodel.ReadReleaseSnapshot(checker.root, tag); err != nil {
			checker.failOrWarn("release: governed tag " + tag + " has no manifest in its tagged tree")
		} else if !snap.Intact() {
			checker.failOrWarn("release: governed tag " + tag + " has a snapshot that is not intact: " + strings.Join(append(issueStrings(snap.Issues), snap.Gaps...), "; "))
		} else if !judged[tag] {
			// The tag is the witness that the release exists: a working tree that lost its directory
			// (or the whole inventory) is judged like one that kept it, not taken for "no releases".
			if gaps, manifest := checkRelease(checker.root, tag); manifest == nil || len(gaps) > 0 {
				checker.failOrWarn("release: governed tag " + tag + " is not a sound cut in the working tree: " + strings.Join(gaps, "; "))
			}
		}
	}
}

func issueStrings(issues []posemodel.ReleaseManifestIssue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.String())
	}
	return out
}

// governedReleaseWitness names what shows that governed releases existed. "" with a nil error is
// a project nothing of which says so; a loss, a Git failure or an unreadable tag is an error, never
// that answer.
func governedReleaseWitness(root string) (string, error) {
	dir := filepath.Join(root, ".pose", "releases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if absent, absentErr := posemodel.ProvenAbsent(dir); absentErr != nil {
			return "", fmt.Errorf("the release inventory could not be read: %w", absentErr)
		} else if !absent {
			return "", fmt.Errorf("the release inventory could not be read: %w", err)
		}
	}
	for _, entry := range entries {
		if posemodel.ValidateReleaseVersion(entry.Name()) == nil {
			return "the cut " + entry.Name() + " in the working tree", nil
		}
	}
	if gitUnavailableAt(root) {
		return "", nil
	}
	raw, err := posemodel.ListReleaseTags(root)
	if err != nil {
		return "", fmt.Errorf("the release tags could not be listed: %w", err)
	}
	for _, tag := range strings.Fields(string(raw)) {
		if posemodel.ValidateReleaseVersion(tag) != nil {
			continue // not a release tag: it witnesses nothing about governed releases
		}
		_, err := posemodel.ReadReleaseSnapshot(root, tag)
		switch {
		case err == nil:
			return "the manifest in the tree of tag " + tag, nil
		case errors.Is(err, posemodel.ErrReleaseSnapshotMissing):
			// A tag without a manifest is either older than the adoption or a cut that lost it: the
			// policy in its own tree tells which.
			adopted, policyErr := posemodel.TagHoldsAdoptedReleasePolicy(root, tag)
			if policyErr != nil {
				return "", fmt.Errorf("the release policy in the tree of tag %s could not be read: %w", tag, policyErr)
			}
			if adopted {
				return "the adopted release policy in the tree of tag " + tag, nil
			}
		default:
			return "", fmt.Errorf("tag %s could not be read: %w", tag, err)
		}
	}
	return "", nil
}

// neverHadAnEvent reports that the journal of version is readable and empty: only then is a
// prepared cut shown never to have been tagged, published or verified.
func neverHadAnEvent(root, version string) bool {
	events, err := posemodel.LoadReleaseEvents(root, version)
	return err == nil && len(events) == 0
}

// manifestIsNotProvenAbsent reports that the manifest of target is not proven to be absent: it is
// there, or something on the way to it could not be looked at.
func manifestIsNotProvenAbsent(root, target string) bool {
	absent, err := posemodel.ProvenAbsent(filepath.Join(root, ".pose", "releases", target, "manifest.json"))
	return err != nil || !absent
}
