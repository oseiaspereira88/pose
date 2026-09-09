// The command surfaces the audit found at zero
// (spec pose-doctor-fixtures-exercise-production-path, follow-up).
//
// The release surface came off zero in
// pose-policy-keys-and-release-surface-coverage. These are the rest of what the
// same measurement named: six commands and the spec-readiness helpers, none of
// which had executed a statement under the suite.
//
// They go through Main, so a command that stops being reachable fails here
// rather than in someone's terminal.

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := 0
	inDir(t, root, func() { code = Main(args, &out, &errOut) })
	return out.String(), errOut.String(), code
}

// Every one of these refused a bad invocation and nothing had ever seen it do
// so. A usage path that returns the wrong code is a script that keeps going.
func TestTheseCommandsRefuseAnInvocationTheyCannotServe(t *testing.T) {
	root := doctorTrailerFixture(t)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"docs-sync with no subcommand", []string{"docs-sync"}, "Usage"},
		{"docs-sync with an unknown one", []string{"docs-sync", "nope"}, "invalid argument"},
		{"docs-review with no subcommand", []string{"docs-review"}, "Usage"},
		{"docs-review with an unknown one", []string{"docs-review", "nope"}, "invalid argument"},
		{"history-check with an unknown flag", []string{"history-check", "--nope"}, "invalid argument"},
		{"roadmap-check with no slug", []string{"roadmap-check"}, "Usage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, errOut, code := runCLI(t, root, tc.args...)
			if code != 2 {
				t.Errorf("code = %d, want 2 for a usage error: %s", code, errOut)
			}
			if !strings.Contains(errOut, tc.want) {
				t.Errorf("the refusal does not say %q: %q", tc.want, errOut)
			}
		})
	}
}

// `history-check` reads the commit history for spec attribution, and its two
// modes differ only in whether a gap is fatal. Neither had run.
func TestHistoryCheckRunsInBothModes(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "alpha", "done")
	// The command answers about `.pose/reports/history/`, so its absence is a
	// missing precondition rather than a clean result — which is itself a branch
	// nothing had run.
	if _, errOut, code := runCLI(t, root, "history-check", "--tolerant"); code != 2 || !strings.Contains(errOut, "history") {
		t.Errorf("an absent history directory = %d %q", code, errOut)
	}

	mustWrite(t, filepath.Join(root, ".pose", "reports", "history", "standard.jsonl"), "{\"run\":1}\n")
	if _, errOut, code := runCLI(t, root, "history-check", "--tolerant"); code != 0 {
		t.Errorf("tolerant mode failed on an untracked history file (%d): %s", code, errOut)
	}
	// Strict is the same read with a different verdict: an untracked JSONL is
	// what it exists to refuse.
	if _, errOut, code := runCLI(t, root, "history-check", "--strict"); code == 0 {
		t.Errorf("strict accepted an untracked history file: %q", errOut)
	}
}

// `review-check` answers whether a scope is approved. On a repository with no
// review policy the answer is "not required", and nothing had asserted that the
// command reaches an answer at all.
func TestReviewCheckAnswersForAScope(t *testing.T) {
	root := doctorTrailerFixture(t)
	writeDoctorTrailerSpec(t, root, "alpha", "in-progress")

	out, errOut, code := runCLI(t, root, "review-check", "spec:alpha", "--json")
	if code != 0 {
		t.Fatalf("code=%d out=%s err=%s", code, out, errOut)
	}
	if !strings.Contains(out, "\"required\"") {
		t.Errorf("the JSON answer does not say whether review is required: %s", out)
	}

	// A scope that does not exist has to fail rather than answer for nothing.
	if _, errOut, code := runCLI(t, root, "review-check", "spec:absent"); code == 0 {
		t.Errorf("a scope that does not exist was answered for: %s", errOut)
	}
}

// `roadmap-check` gates a roadmap on its cut criteria. Both modes were
// unexecuted, so "strict fails and tolerant does not" was an assumption.
func TestRoadmapCheckIsStrictByDefaultAndTolerantOnRequest(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "roadmaps", "alpha.md"),
		"---\nslug: alpha\nstatus: in-progress\n---\n\n## Cut criteria\n"+
			"- C1: surface:dashboard check:web-reachability evidence:e2e\n")
	// Without delivery profiles the graph builder returns before it ever loads a
	// roadmap, so the command reports no criteria and passes. That early return
	// is why this fixture carries a profile index: the assertion below is about
	// the gate, and it has to reach the gate.
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{"web-ui":{"kind":"surface","requiredEvidenceClasses":["reachability"],"anyEvidenceClasses":["integration","e2e"]}},"stacks":{}}`)

	mustWrite(t, filepath.Join(root, ".pose", "specs", "alpha-spec", "spec.md"),
		"---\nslug: alpha-spec\nstatus: in-progress\ncreated_at: 2026-08-15\n---\n\n# Spec: alpha-spec\n\nwork\n")

	out, errOut, strictCode := runCLI(t, root, "roadmap-check", "alpha", "--json")
	if !strings.Contains(out, "C1") {
		t.Fatalf("the criterion never reached the gate: out=%q err=%q", out, errOut)
	}
	if !strings.Contains(out, "unknown delivery ref") {
		t.Errorf("the criterion passed while naming a delivery target that does not exist: %s", out)
	}
	_, _, tolerantCode := runCLI(t, root, "roadmap-check", "alpha", "--tolerant")
	if strictCode == 0 {
		t.Errorf("strict accepted an unmet cut criterion: %s%s", out, errOut)
	}
	if tolerantCode != 0 {
		t.Errorf("tolerant refused what it is meant to report (%d): %s", tolerantCode, errOut)
	}
	// A slug that is not one has to be refused before anything is read.
	if _, errOut, code := runCLI(t, root, "roadmap-check", "not a slug"); code != 2 || !strings.Contains(errOut, "invalid roadmap slug") {
		t.Errorf("an invalid slug = %d %q", code, errOut)
	}
}

// `contribute submit` reaches `gh issue create`. What is covered here is
// everything before that: the two refusals. The submit itself is an outward
// action and is deliberately not exercised.
func TestContributeSubmitRefusesBeforeItReachesTheNetwork(t *testing.T) {
	root := doctorTrailerFixture(t)

	_, errOut, code := runCLI(t, root, "contribute", "submit")
	if code != 2 || !strings.Contains(errOut, "Usage") {
		t.Errorf("no argument = %d %q", code, errOut)
	}
	_, errOut, code = runCLI(t, root, "contribute", "submit", "no-such-contribution")
	if code == 0 {
		t.Errorf("a contribution that does not exist was submitted: %q", errOut)
	}
	if strings.Contains(errOut, "Submitting contribution upstream") {
		t.Errorf("the refusal happened after announcing a submission: %q", errOut)
	}
}

// `docs-sync export` writes the docs manifest a Conductor consumes. `push` is
// the half that talks to one, and is not exercised.
func TestDocsSyncExportsWithoutAConductor(t *testing.T) {
	root := doctorTrailerFixture(t)
	// Without a manifest there is nothing to export, and saying so is a branch
	// of its own.
	if _, errOut, code := runCLI(t, root, "docs-sync", "export"); code == 0 || !strings.Contains(errOut, "docs manifest") {
		t.Errorf("an absent manifest = %d %q", code, errOut)
	}

	mustWrite(t, filepath.Join(root, ".pose", "docs.json"),
		`{"schema_version":1,"roots":["docs"],"docs":[]}`)
	out, errOut, code := runCLI(t, root, "docs-sync", "export")
	if code != 0 {
		t.Fatalf("export failed (%d): %s %s", code, out, errOut)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("export produced nothing at all")
	}
}

// The Definition of Ready decides whether a spec may leave draft, and every
// helper it is built from was unexecuted.
func TestSpecReadyReadsTheSectionsTheDoRRequires(t *testing.T) {
	root := doctorTrailerFixture(t)
	path := filepath.Join(root, ".pose", "specs", "alpha.md")

	// A feature spec needs Intent, Requirements and Technical Plan filled, and
	// a numbered requirement in Requirements.
	mustWrite(t, path, "---\nslug: alpha\nstatus: draft\n---\n\n"+
		"## 1. Intent\n\nA goal.\n\n## 2. Requirements\n\n- R1: something\n\n## 3. Technical Plan\n\nA plan.\n")
	if !specReady(root, path) {
		t.Error("a complete feature spec was reported not ready")
	}

	// An empty section is not a filled one, whatever it is called.
	mustWrite(t, path, "---\nslug: alpha\nstatus: draft\n---\n\n"+
		"## 1. Intent\n\nA goal.\n\n## 2. Requirements\n\n- R1: something\n\n## 3. Technical Plan\n\n")
	if specReady(root, path) {
		t.Error("a spec with an empty Technical Plan was reported ready")
	}

	// Requirements prose without a numbered criterion is the case the gate
	// exists for: a section that looks filled and states nothing testable.
	mustWrite(t, path, "---\nslug: alpha\nstatus: draft\n---\n\n"+
		"## 1. Intent\n\nA goal.\n\n## 2. Requirements\n\nWe want it to be good.\n\n## 3. Technical Plan\n\nA plan.\n")
	if specReady(root, path) {
		t.Error("Requirements with no numbered criterion was accepted")
	}

	// A task type the policy configures replaces the required set.
	mustWrite(t, filepath.Join(root, ".pose", "policy", "dor.json"),
		`{"schemaVersion":1,"defaultTaskType":"feature","taskTypes":{"bugfix":["Intent"]}}`)
	mustWrite(t, path, "---\nslug: alpha\nstatus: draft\ntask_type: bugfix\n---\n\n## 1. Intent\n\nA goal.\n")
	if !specReady(root, path) {
		t.Error("a bugfix spec was held to the feature sections")
	}
}

// A commented-out section is not a section. Stripping comments before reading
// them is what stops a spec from passing on text nobody renders.
func TestSpecSectionsIgnoreWhatIsCommentedOut(t *testing.T) {
	text := stripHTMLComments("## 1. Intent\n\n<!-- ## 2. Requirements\n\n- R1: hidden\n-->\n\n## 3. Technical Plan\n\nA plan.\n")
	sections := specSections(text)
	if _, ok := sections["Requirements"]; ok {
		t.Error("a commented-out section was read as present")
	}
	if !sectionFilled(sections["Technical Plan"]) {
		t.Error("a real section after a comment was lost")
	}
	if sectionFilled(sections["Intent"]) {
		t.Error("an empty Intent was reported filled")
	}
}

// sectionFilled skips the scaffold's own prompts, or every freshly created spec
// would report ready.
func TestSectionFilledSkipsScaffoldProse(t *testing.T) {
	if sectionFilled([]string{"", "- [ ]", "-", "> Definition of Ready: fill this in", "### A heading"}) {
		t.Error("scaffold placeholders were counted as content")
	}
	if !sectionFilled([]string{"", "- [ ]", "an actual sentence"}) {
		t.Error("real content after placeholders was not seen")
	}
}

// frontmatterBody returns what comes after the frontmatter, and the whole file
// when there is none — the case a spec written without frontmatter takes.
func TestFrontmatterBodyHandlesBothShapes(t *testing.T) {
	root := t.TempDir()
	withFM := filepath.Join(root, "a.md")
	if err := os.WriteFile(withFM, []byte("---\nslug: a\n---\n\nBody here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if body := frontmatterBody(withFM); strings.Contains(body, "slug: a") || !strings.Contains(body, "Body here.") {
		t.Errorf("frontmatter leaked into the body: %q", body)
	}
	without := filepath.Join(root, "b.md")
	if err := os.WriteFile(without, []byte("Just a body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if body := frontmatterBody(without); !strings.Contains(body, "Just a body.") {
		t.Errorf("a file with no frontmatter lost its body: %q", body)
	}
	if body := frontmatterBody(filepath.Join(root, "absent.md")); body != "" {
		t.Errorf("a missing file returned %q", body)
	}
}

// parseRoadmap reads the milestones a roadmap declares. It fell back to the
// filename when the frontmatter has no slug, and nothing had taken that path.
func TestParseRoadmapReadsMilestonesAndFallsBackToTheFilename(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "delivery-2026.md")
	if err := os.WriteFile(path, []byte(
		"---\nstatus: in-progress\ndepends_on: alpha, beta\n---\n\n"+
			"## Milestone: first\n- specs: alpha\n\n## Not a milestone\n- specs: ignored\n\n## Milestone: second\n- specs: beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	roadmap := parseRoadmap(path)
	if roadmap.slug != "delivery-2026" {
		t.Errorf("slug = %q, want the filename when the frontmatter has none", roadmap.slug)
	}
	if len(roadmap.milestones) != 2 || roadmap.milestones[0].id != "first" || roadmap.milestones[1].id != "second" {
		t.Errorf("milestones = %+v", roadmap.milestones)
	}
	if len(roadmap.dependsOn) != 2 {
		t.Errorf("dependsOn = %v", roadmap.dependsOn)
	}
	if parseRoadmap(filepath.Join(root, "absent.md")).slug != "absent" {
		t.Error("a missing roadmap did not fall back to its filename")
	}
}
