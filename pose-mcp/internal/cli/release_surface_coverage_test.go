// Coverage audit of the release command surface
// (spec pose-doctor-fixtures-exercise-production-path, follow-up).
//
// The doctor audit found three unreached behavioural branches in one command.
// Run over the rest of the CLI, the same measurement found something larger:
// the entire release surface executed zero statements under the suite — plan,
// check, status, record, open-next, notes and the helpers under them.
//
// That is the surface whose first execution has always been a real release, and
// it is the surface that has failed there twice this month. These cover the
// behaviour that decides what a release is, not the defensive guards.

package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/version"
)

// releaseSurfaceFixture is a repository with one done spec and one pending
// fragment: the smallest thing the release commands accept as real input.
func releaseSurfaceFixture(t *testing.T, category string, breaking bool) string {
	t.Helper()
	root := t.TempDir()
	writeReleaseFixture(t, root, ".pose/policy/release.json",
		`{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo"}`)
	writeReleaseFixture(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\n---\n")
	body := "---\nspec: alpha\ncategory: " + category + "\nbreaking: "
	if breaking {
		body += "true"
	} else {
		body += "false"
	}
	writeReleaseFixture(t, root, ".pose/changelogs/unreleased/alpha.md", body+"\n---\n\nSomething.\n")
	return root
}

func runReleaseCmd(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cmdRelease(root, args, &out, &errOut)
	return out.String(), errOut.String(), code
}

// The recommendation is the one judgement `release plan` makes on its own, and
// the three answers came from three different rules. Nothing had executed any
// of them.
func TestReleasePlanRecommendsFromTheFragmentsItReads(t *testing.T) {
	target := "v" + version.ReleaseBase()
	for _, tc := range []struct {
		category string
		breaking bool
		want     string
	}{
		{"fixed", false, "patch"},
		{"added", false, "minor"},
		{"fixed", true, "major"},
	} {
		t.Run(tc.category+"/"+tc.want, func(t *testing.T) {
			root := releaseSurfaceFixture(t, tc.category, tc.breaking)
			out, errOut, code := runReleaseCmd(t, root, "plan", "--version", target, "--json")
			if code != 0 {
				t.Fatalf("code=%d %s", code, errOut)
			}
			var plan struct {
				Version        string   `json:"version"`
				FragmentCount  int      `json:"fragment_count"`
				Breaking       bool     `json:"breaking"`
				Recommendation string   `json:"recommendation"`
				Specs          []string `json:"specs"`
				DryRun         bool     `json:"dry_run"`
			}
			if err := json.Unmarshal([]byte(out), &plan); err != nil {
				t.Fatalf("plan is not JSON: %v\n%s", err, out)
			}
			if plan.Recommendation != tc.want {
				t.Errorf("recommendation=%q, want %q", plan.Recommendation, tc.want)
			}
			if plan.FragmentCount != 1 || len(plan.Specs) != 1 || plan.Specs[0] != "alpha" {
				t.Errorf("the plan does not describe the fragment it read: %+v", plan)
			}
			if plan.Breaking != tc.breaking {
				t.Errorf("breaking=%v, want %v", plan.Breaking, tc.breaking)
			}
			if !plan.DryRun {
				t.Error("plan reported dry_run=false; planning must never be the thing that applies")
			}
		})
	}
}

// A plan that consumed the fragments would make `prepare` a no-op and the
// pending queue would empty without a manifest. Nothing asserted it does not.
func TestReleasePlanLeavesThePendingFragmentAlone(t *testing.T) {
	root := releaseSurfaceFixture(t, "added", false)
	before, _, code := runReleaseCmd(t, root, "status", "--json")
	if code != 0 {
		t.Fatal("status failed on a fresh fixture")
	}
	if _, errOut, code := runReleaseCmd(t, root, "plan", "--version", "v"+version.ReleaseBase()); code != 0 {
		t.Fatalf("plan code=%d %s", code, errOut)
	}
	after, _, _ := runReleaseCmd(t, root, "status", "--json")
	if before != after {
		t.Errorf("planning changed the release state:\nbefore %s\nafter  %s", before, after)
	}
}

// `release status` is how an operator asks what is pending and what shipped.
// Both halves were unexecuted, so "it reports the queue" was an assumption.
func TestReleaseStatusReportsThePendingQueue(t *testing.T) {
	root := releaseSurfaceFixture(t, "fixed", false)
	out, errOut, code := runReleaseCmd(t, root, "status", "--json")
	if code != 0 {
		t.Fatalf("code=%d %s", code, errOut)
	}
	var status struct {
		Pending  []map[string]any `json:"pending"`
		Releases []map[string]any `json:"releases"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("status is not JSON: %v\n%s", err, out)
	}
	if len(status.Pending) != 1 {
		t.Errorf("pending=%d, want the one unreleased fragment: %s", len(status.Pending), out)
	}

	// And the human-readable form, which is a separate branch.
	plain, _, code := runReleaseCmd(t, root, "status")
	if code != 0 || !strings.Contains(plain, "Pending fragments: 1") {
		t.Errorf("the plain form does not report the queue: %q", plain)
	}
}

// An invalid version has to be refused before anything reads a policy, or the
// error an operator sees is about the wrong thing.
func TestReleaseStatusRefusesAVersionThatIsNotOne(t *testing.T) {
	root := releaseSurfaceFixture(t, "fixed", false)
	_, errOut, code := runReleaseCmd(t, root, "status", "--version", "not-a-version")
	if code == 0 {
		t.Fatal("an invalid version was accepted")
	}
	if !strings.Contains(errOut, "invalid release version") {
		t.Errorf("the refusal does not say what is wrong: %q", errOut)
	}
}

// versionLess orders the release history, and the history is what `previous`
// resolves from. The case that matters is the one lexicographic ordering gets
// wrong, which is exactly what a numeric comparison exists for.
func TestVersionLessOrdersNumericallyNotLexicographically(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"v1.9.0", "v1.10.0", true},
		{"v1.10.0", "v1.9.0", false},
		{"v2.0.0", "v10.0.0", true},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.0.0", false},
		// A prerelease sorts by its release part first, so it never jumps ahead
		// of a higher version.
		{"v1.0.0-rc1", "v1.1.0", true},
	} {
		if got := versionLess(tc.a, tc.b); got != tc.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// confinedProjectPath is what keeps a path out of a release artifact from
// reaching outside the repository. It had no test at all.
func TestConfinedProjectPathRefusesEveryWayOut(t *testing.T) {
	root := t.TempDir()
	writeReleaseFixture(t, root, "inside/keep.txt", "x\n")

	if _, err := confinedProjectPath(root, "inside/keep.txt"); err != nil {
		t.Fatalf("a path inside the repository was refused: %v", err)
	}
	for _, path := range []string{
		"../escape.txt",
		"inside/../../escape.txt",
	} {
		if _, err := confinedProjectPath(root, path); err == nil {
			t.Errorf("%q was accepted", path)
		}
	}
	if _, err := confinedProjectPath(root, "/etc/passwd"); err == nil {
		t.Error("an absolute path was accepted")
	}
	// A symlink is the way in that a lexical check alone does not close: the
	// path never mentions the parent directory.
	if err := symlinkForTest(root, "..", "escape-link"); err == nil {
		if _, err := confinedProjectPath(root, "escape-link/anything"); err == nil {
			t.Error("a path through a symlink out of the repository was accepted")
		}
	}
}

// resolveReleaseTag must answer with a commit or not at all: a tag that does
// not resolve is what a release must refuse to be cut from.
func TestResolveReleaseTagOnlyAnswersForARealTag(t *testing.T) {
	root := t.TempDir()
	releaseGit(t, root, "-C", root, "init", "-q")
	writeReleaseFixture(t, root, "file.txt", "x\n")
	releaseGit(t, root, "-C", root, "add", ".")
	releaseGit(t, root, "-C", root, "commit", "-q", "-m", "first")
	releaseGit(t, root, "-C", root, "tag", "v9.9.9")

	commit, ok := resolveReleaseTag(root, "v9.9.9")
	if !ok {
		t.Fatalf("a real tag did not resolve: %q", commit)
	}
	if len(commit) != 40 {
		t.Errorf("commit=%q, want a full object name", commit)
	}
	if _, ok := resolveReleaseTag(root, "v0.0.0-absent"); ok {
		t.Error("a tag that does not exist resolved")
	}
}

func symlinkForTest(root, target, name string) error {
	return exec.Command("ln", "-s", target, root+"/"+name).Run()
}

// preparedReleaseFixture takes the fixture through prepare and a real tag, so
// the commands that only run against a prepared, tagged release have one.
func preparedReleaseFixture(t *testing.T) (root, target, commit string) {
	t.Helper()
	root = releaseSurfaceFixture(t, "added", false)
	target = "v" + version.ReleaseBase()
	releaseGit(t, root, "init", "-q")
	if _, errOut, code := runReleaseCmd(t, root, "prepare", "--version", target, "--apply"); code != 0 {
		t.Fatalf("prepare code=%d %s", code, errOut)
	}
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-q", "-m", "release "+target)
	releaseGit(t, root, "tag", target)
	resolved, ok := resolveReleaseTag(root, target)
	if !ok {
		t.Fatalf("the fixture tag did not resolve: %q", resolved)
	}
	return root, target, resolved
}

// `release check` is the gate a release is cut through, and it had executed
// nothing. Both answers matter: a prepared snapshot is valid, and --strict is
// what turns a gap into a refusal rather than a note.
func TestReleaseCheckAnswersForAPreparedSnapshotAndRefusesUnderStrict(t *testing.T) {
	root, target, commit := preparedReleaseFixture(t)

	out, errOut, code := runReleaseCmd(t, root, "check", "--version", target, "--json")
	if code != 0 {
		t.Fatalf("code=%d %s", code, errOut)
	}
	var result struct {
		Valid     bool     `json:"valid"`
		Tagged    bool     `json:"tagged"`
		TagCommit string   `json:"tag_commit"`
		Gaps      []string `json:"gaps"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("check is not JSON: %v\n%s", err, out)
	}
	if !result.Valid || !result.Tagged || result.TagCommit != commit {
		t.Fatalf("a prepared, tagged release did not check out: %+v", result)
	}

	// An unprepared version is the gap case, and --strict is what makes it an
	// exit code instead of a line on stderr.
	if _, _, code := runReleaseCmd(t, root, "check", "--version", "v99.98.97"); code != 0 {
		t.Error("without --strict, gaps must report rather than fail")
	}
	if _, errOut, code := runReleaseCmd(t, root, "check", "--version", "v99.98.97", "--strict"); code == 0 {
		t.Errorf("--strict accepted an unprepared release: %s", errOut)
	}
}

// `release notes` exists to emit the snapshot that was frozen, and to refuse
// anything else. The refusal is the half that matters: notes read from a stale
// file are notes that describe a different release.
func TestReleaseNotesEmitsTheFrozenSnapshotAndRefusesAStaleOne(t *testing.T) {
	root, target, _ := preparedReleaseFixture(t)

	out, errOut, code := runReleaseCmd(t, root, "notes", "--version", target)
	if code != 0 {
		t.Fatalf("code=%d %s", code, errOut)
	}
	if !strings.Contains(out, "Something.") {
		t.Errorf("the notes do not carry the fragment body: %q", out)
	}

	writeReleaseFixture(t, root, ".pose/changelogs/"+target+".md", "tampered\n")
	if _, errOut, code := runReleaseCmd(t, root, "notes", "--version", target); code == 0 {
		t.Error("an altered snapshot was emitted as the frozen one")
	} else if !strings.Contains(errOut, "stale or invalid snapshot") {
		t.Errorf("the refusal does not say why: %q", errOut)
	}
}

// The rule that keeps a release record honest: evidence has to name the commit
// the tag actually points at. Nothing had ever executed it.
func TestReleaseRecordRefusesEvidenceThatDoesNotMatchTheTag(t *testing.T) {
	root, target, commit := preparedReleaseFixture(t)

	evidence := func(c string) string {
		return `{"schema_version":1,"version":"` + target + `","tag":"` + target +
			`","provider":"github","repository":"owner/repo","commit":"` + c + `"}`
	}
	writeReleaseFixture(t, root, ".pose/evidence-good.json", evidence(commit))
	writeReleaseFixture(t, root, ".pose/evidence-wrong.json", evidence("0000000000000000000000000000000000000000"))

	if _, errOut, code := runReleaseCmd(t, root, "record", "--version", target, "--event", "tagged", "--evidence", ".pose/evidence-good.json"); code != 0 {
		t.Fatalf("evidence naming the tag's own commit was refused: %s", errOut)
	}
	_, errOut, code := runReleaseCmd(t, root, "record", "--version", target, "--event", "tagged", "--evidence", ".pose/evidence-wrong.json")
	if code == 0 {
		t.Fatal("evidence naming a different commit was recorded")
	}
	if !strings.Contains(errOut, "does not match the immutable Git tag") {
		t.Errorf("the refusal does not say what is wrong: %q", errOut)
	}

	// And the evidence path is confined, so a record cannot read outside the
	// project it is recording for.
	if _, errOut, code := runReleaseCmd(t, root, "record", "--version", target, "--event", "tagged", "--evidence", "../outside.json"); code == 0 {
		t.Error("an evidence path outside the project was accepted")
	} else if !strings.Contains(errOut, "inside project") {
		t.Errorf("the refusal does not name the confinement: %q", errOut)
	}
}

// `release open-next` is what says the cycle may move on, and it must not say
// so while the last release is unverified. That refusal was unexecuted, which
// makes it the one worth pinning.
func TestReleaseOpenNextRefusesWhileTheLastReleaseIsUnverified(t *testing.T) {
	root, target, _ := preparedReleaseFixture(t)

	_, errOut, code := runReleaseCmd(t, root, "open-next", "--version", "v99.98.97")
	if code == 0 {
		t.Fatal("the next cycle opened over an unverified release")
	}
	if !strings.Contains(errOut, "not verified") {
		t.Errorf("the refusal does not name the reason: %q", errOut)
	}

	if _, errOut, code := runReleaseCmd(t, root, "open-next", "--version", "not-a-version"); code == 0 {
		t.Errorf("an invalid next version was accepted: %s", errOut)
	}
	_ = target
}
