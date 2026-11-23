package cli

// Carried checks (spec pose-validate-carries-unchanged-checks): validate
// --carry refreshes the canonical result by running only what a change can
// affect and carrying the rest with the head it really ran at.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const carryResults = ".pose/results/delivery-validation.json"

type carryFixture struct {
	t       *testing.T
	root    string
	counter string
}

// newCarryFixture builds a committed repository with Go modules a and b and
// a root file. Every check appends its module to a counter file outside the
// tree, so a test can tell an executed check from a carried one.
func newCarryFixture(t *testing.T, metadata string) *carryFixture {
	t.Helper()
	f := &carryFixture{t: t, root: t.TempDir(), counter: filepath.Join(t.TempDir(), "runs")}
	f.write("a/go.mod", "module a\n")
	f.write("b/go.mod", "module b\n")
	f.write("root.txt", "root\n")
	f.write(".gitignore", ".pose/transfers/.authority-transfer.lock\n")
	f.write(".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"results_path":"`+carryResults+`"}`)
	matrix := `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[` +
		`{"name":"ok","program":"sh","args":["-c","basename \"$PWD\" >> ` + f.counter + `; test ! -f fail"],"severity":"required","evidenceClass":"unit"}]}}}`
	f.write(".pose/indexes/validation-matrix.json", matrix)
	if metadata != "" {
		f.write(".pose/indexes/module-metadata.json", metadata)
	}
	f.git("init", "-q")
	f.commit("base")
	return f
}

func (f *carryFixture) write(rel, content string) {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *carryFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (f *carryFixture) commit(message string) string {
	f.t.Helper()
	f.git("add", "-A")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", message, "--allow-empty")
	return f.git("rev-parse", "HEAD")
}

// validate runs pose validate and returns its exit code, output and the
// canonical record it wrote. The counter is reset first.
func (f *carryFixture) validate(args ...string) (int, string, validationRunResult) {
	f.t.Helper()
	_ = os.Remove(f.counter)
	code, out := runValidate(f.t, f.root, append(args, "--json-out", carryResults)...)
	var run validationRunResult
	if raw, err := os.ReadFile(filepath.Join(f.root, carryResults)); err == nil {
		if err := json.Unmarshal(raw, &run); err != nil {
			f.t.Fatal(err)
		}
	}
	return code, out, run
}

func (f *carryFixture) ran() []string {
	raw, err := os.ReadFile(f.counter)
	if err != nil {
		return nil
	}
	runs := strings.Fields(string(raw))
	sort.Strings(runs)
	return runs
}

func checkOf(run validationRunResult, module string) checkResult {
	for _, c := range run.Checks {
		if c.Module == module {
			return c
		}
	}
	return checkResult{}
}

// baseline runs the full matrix, commits the canonical record and returns the
// head the record validated.
func (f *carryFixture) baseline() string {
	f.t.Helper()
	validated := f.git("rev-parse", "HEAD")
	code, out, _ := f.validate()
	if code != 0 {
		f.t.Fatalf("baseline exit=%d %s", code, out)
	}
	f.commit("record")
	return validated
}

func TestCarryRunsOnlyTheChangedModule(t *testing.T) {
	f := newCarryFixture(t, "")
	base := f.baseline()
	f.write("a/changed.go", "package a\n")
	head := f.commit("change a")
	code, out, run := f.validate("--carry")
	if code != 0 {
		t.Fatalf("exit=%d %s", code, out)
	}
	if got := strings.Join(f.ran(), ","); got != "a" {
		t.Fatalf("executed modules = %q, want only a", got)
	}
	if run.Carry == nil || !run.Carry.Used || run.Carry.BaseHead != base {
		t.Fatalf("carry = %+v, want used from %s", run.Carry, base)
	}
	if b := checkOf(run, "b"); b.Outcome != "pass" || b.CarriedFrom != base || b.CarriedGeneratedAt == "" {
		t.Fatalf("b = %+v, want a pass carried from %s", b, base)
	}
	if a := checkOf(run, "a"); a.CarriedFrom != "" || a.Outcome != "pass" {
		t.Fatalf("a = %+v, want executed", a)
	}
	if run.GitHead != head || run.Counts.Carried != 1 || run.Counts.Executed != 1 || run.Counts.Passed != 2 {
		t.Fatalf("head=%s counts=%+v", run.GitHead, run.Counts)
	}
	if run.EngineVersion == "" {
		t.Fatal("engine_version must be recorded")
	}
}

func TestCarryKeepsTheOriginalHeadAcrossChainedCarries(t *testing.T) {
	f := newCarryFixture(t, "")
	base := f.baseline()
	f.write("a/one.go", "package a\n")
	f.commit("change a")
	if code, out, _ := f.validate("--carry"); code != 0 {
		t.Fatalf("exit=%d %s", code, out)
	}
	f.commit("record again")
	f.write("a/two.go", "package a\n")
	f.commit("change a again")
	_, _, run := f.validate("--carry")
	if b := checkOf(run, "b"); b.CarriedFrom != base {
		t.Fatalf("b carried from %q, want the head it ran at %s", b.CarriedFrom, base)
	}
}

func TestCarryKeepsACarriedFailureAFailure(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/fail", "x\n")
	f.commit("b fails")
	if code, _, _ := f.validate(); code == 0 {
		t.Fatal("baseline with a failing b must fail")
	}
	f.commit("record")
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	code, _, run := f.validate("--carry")
	if code == 0 || run.Outcome != "fail" || checkOf(run, "b").Outcome != "fail" || checkOf(run, "b").CarriedFrom == "" {
		t.Fatalf("exit=%d outcome=%s b=%+v, want the carried failure to fail the run", code, run.Outcome, checkOf(run, "b"))
	}
}

func TestCarryFallsBackToAFullRun(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(f *carryFixture)
		reason string
	}{
		{"no base", func(f *carryFixture) {}, "no canonical result"},
		{"matrix changed", func(f *carryFixture) {
			f.baseline()
			raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
			f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"strict"`, `"strict" `, 1))
			f.commit("matrix")
		}, "matrix"},
		{"dirty worktree", func(f *carryFixture) {
			f.baseline()
			f.write("a/dirty.go", "package a\n")
		}, "working copy"},
		{"base not an ancestor", func(f *carryFixture) {
			f.baseline()
			raw, _ := os.ReadFile(filepath.Join(f.root, carryResults))
			var run map[string]any
			_ = json.Unmarshal(raw, &run)
			run["git_head"] = strings.Repeat("0", 40)
			out, _ := json.Marshal(run)
			f.write(carryResults, string(out))
			f.commit("forged base")
		}, "ancestor"},
		{"base without engine version", func(f *carryFixture) {
			f.baseline()
			raw, _ := os.ReadFile(filepath.Join(f.root, carryResults))
			var run map[string]any
			_ = json.Unmarshal(raw, &run)
			delete(run, "engine_version")
			out, _ := json.Marshal(run)
			f.write(carryResults, string(out))
			f.commit("old base")
		}, "engine"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCarryFixture(t, "")
			tc.mutate(f)
			_, out, run := f.validate("--carry")
			if got := strings.Join(f.ran(), ","); got != "a,b" {
				t.Fatalf("executed = %q, want a full run; out=%s", got, out)
			}
			if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, tc.reason) {
				t.Fatalf("carry = %+v, want unused with a reason naming %q", run.Carry, tc.reason)
			}
		})
	}
}

func TestCarryRootChangeRunsEverythingUnlessSelfContained(t *testing.T) {
	f := newCarryFixture(t, "")
	f.baseline()
	f.write("root.txt", "edited\n")
	f.commit("root")
	f.validate("--carry")
	if got := strings.Join(f.ran(), ","); got != "a,b" {
		t.Fatalf("root change executed %q, want everything", got)
	}

	g := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true}}}`)
	g.baseline()
	g.write("root.txt", "edited\n")
	g.commit("root")
	_, _, run := g.validate("--carry")
	if got := strings.Join(g.ran(), ","); got != "a" {
		t.Fatalf("root change executed %q, want only the module that is not self-contained", got)
	}
	if checkOf(run, "b").CarriedFrom == "" {
		t.Fatal("the self-contained module must be carried")
	}
}

func TestChangedScopeHonoursSelfContainedModules(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true}}}`)
	f.write("root.txt", "edited\n")
	_, out := runValidate(t, f.root, "--changed-from", "HEAD", "--explain")
	if !strings.Contains(out, "+ a: root-level change outside any module: root.txt") || !strings.Contains(out, "- b: not affected") {
		t.Fatalf("self-contained module must not be widened by a root change: %s", out)
	}
}

func TestCarryRecordFileIsNotAChange(t *testing.T) {
	f := newCarryFixture(t, "")
	f.baseline()
	_, _, run := f.validate("--carry")
	if len(f.ran()) != 0 || run.Counts.Carried != 2 {
		t.Fatalf("executed=%v counts=%+v carry=%+v, want everything carried when only the record changed", f.ran(), run.Counts, run.Carry)
	}
}

func TestCarrySelectsAModuleWithoutAPriorResult(t *testing.T) {
	f := newCarryFixture(t, "")
	f.baseline()
	raw, _ := os.ReadFile(filepath.Join(f.root, carryResults))
	var run map[string]any
	_ = json.Unmarshal(raw, &run)
	checks := run["checks"].([]any)
	kept := []any{}
	for _, c := range checks {
		if c.(map[string]any)["module"] != "b" {
			kept = append(kept, c)
		}
	}
	run["checks"] = kept
	out, _ := json.Marshal(run)
	f.write(carryResults, string(out))
	f.commit("base without b")
	_, _, result := f.validate("--carry")
	if got := strings.Join(f.ran(), ","); got != "b" {
		t.Fatalf("executed = %q, want only the module without a prior result", got)
	}
	if !strings.Contains(result.Carry.Selected["b"], "no prior result") {
		t.Fatalf("selection reason = %q", result.Carry.Selected["b"])
	}
}

func TestCarryRefusesAFilteredBase(t *testing.T) {
	f := newCarryFixture(t, "")
	if code, out, _ := f.validate("--module", "a"); code != 0 {
		t.Fatalf("exit=%d %s", code, out)
	}
	f.commit("partial record")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "filter") {
		t.Fatalf("carry = %+v, want a filtered base refused", run.Carry)
	}
}

func TestCarryRefusesIncompatibleFlags(t *testing.T) {
	f := newCarryFixture(t, "")
	for _, extra := range [][]string{{"--module", "a"}, {"--stack", "go"}, {"--changed-from", "HEAD"}} {
		args := append([]string{"--carry"}, extra...)
		if code, out := runValidate(t, f.root, append(args, "--json-out", carryResults)...); code != 2 {
			t.Fatalf("%v: exit=%d, want 2: %s", extra, code, out)
		}
	}
	if code, out := runValidate(t, f.root, "--carry", "--json-out", "other.json"); code != 2 {
		t.Fatalf("--carry with a non-canonical output: exit=%d, want 2: %s", code, out)
	}
}

func TestCarryIgnoresDerivedAdministrativeRecords(t *testing.T) {
	f := newCarryFixture(t, "")
	f.baseline()
	f.write(".pose/indexes/generation.json", "{\"generation\":2}\n")
	f.write(".pose/state/project-state.md", "state\n")
	f.commit("index and state")
	_, _, run := f.validate("--carry")
	if len(f.ran()) != 0 || run.Counts.Carried != 2 {
		t.Fatalf("executed=%v counts=%+v carry=%+v, want derived records not to count as changes", f.ran(), run.Counts, run.Carry)
	}
	// A script stored in a class, and the validation matrix, stay material.
	f.write(".pose/indexes/helper.sh", "echo\n")
	f.commit("script in a class")
	f.validate("--carry")
	if got := strings.Join(f.ran(), ","); got != "a,b" {
		t.Fatalf("a non-record file in a class executed %q, want everything", got)
	}
}

// A pass recorded over an uncommitted fix proves nothing about the committed
// head; once the fix is dropped it must not be carried (review round 1, F1).
func TestCarryRefusesABaseThatValidatedADirtyWorkingCopy(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/fail", "x\n")
	f.commit("b fails")
	if err := os.Remove(filepath.Join(f.root, "b", "fail")); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := f.validate(); code != 0 {
		t.Fatalf("the uncommitted fix should pass: exit=%d %s", code, out)
	}
	f.git("add", "--", carryResults)
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "record over a dirty copy")
	f.git("checkout", "--", "b/fail")
	code, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "clean working copy") {
		t.Fatalf("carry = %+v, want a dirty base refused", run.Carry)
	}
	if code == 0 || checkOf(run, "b").Outcome != "fail" {
		t.Fatalf("exit=%d b=%+v, want b re-run and failing", code, checkOf(run, "b"))
	}
}

// A rename names its source module too (review round 2, N1).
func TestCarrySelectsBothSidesOfARename(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("a/data.txt", "data\n")
	f.commit("a data")
	f.baseline()
	f.git("mv", "a/data.txt", "b/data.txt")
	f.commit("move data from a to b")
	_, _, run := f.validate("--carry")
	if got := strings.Join(f.ran(), ","); got != "a,b" {
		t.Fatalf("a rename executed %q, want both its source and destination modules", got)
	}
	if !strings.Contains(run.Carry.Selected["a"], "a/data.txt") {
		t.Fatalf("source module reason = %q", run.Carry.Selected["a"])
	}
}

// Names Git would quote keep their owner (review round 2, N3).
func TestCarryKeepsTheOwnerOfAQuotedName(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"a":{"selfContained":true},"b":{"selfContained":true}}}`)
	f.baseline()
	f.write("a/naïve \"name\".txt", "x\n")
	f.commit("quoted name in a")
	_, _, run := f.validate("--carry")
	if got := strings.Join(f.ran(), ","); got != "a" {
		t.Fatalf("a quoted name in a executed %q, want a", got)
	}
	if !strings.HasPrefix(run.Carry.Selected["a"], "contains changed file: a/") {
		t.Fatalf("reason = %q, want ownership by a", run.Carry.Selected["a"])
	}
}

// A root module owns root files even when it declares itself self-contained
// (review round 2, N2).
func TestCarryRootModuleOwnsRootFiles(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{".":{"selfContained":true},"a":{"selfContained":true},"b":{"selfContained":true}}}`)
	f.write("go.mod", "module root\n")
	f.commit("root module")
	f.baseline()
	f.write("main.go", "package main\n")
	f.commit("root change")
	_, _, run := f.validate("--carry")
	if _, ok := run.Carry.Selected["."]; !ok {
		t.Fatalf("selected = %v, want the root module selected by its own file", run.Carry.Selected)
	}
	if got := strings.Join(f.ran(), ","); strings.Contains(got, "a") || strings.Contains(got, "b") || got == "" {
		t.Fatalf("executed %q, want only the root module", got)
	}
}

// A deleted dependency still selects its self-contained dependents (review
// round 3, N5).
func TestCarryDeletedDependencySelectsItsDependents(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["a"]}}}`)
	f.baseline()
	f.git("rm", "-rq", "a")
	f.commit("delete a")
	_, _, run := f.validate("--carry")
	if !strings.Contains(run.Carry.Selected["b"], "a/") {
		t.Fatalf("selected = %v, want b selected by its deleted dependency", run.Carry.Selected)
	}
	if got := strings.Join(f.ran(), ","); got != "b" {
		t.Fatalf("executed %q, want b", got)
	}
}

// Metadata that does not decode never narrows the selection (review round 3,
// N6).
func TestCarryRejectsUndecodableMetadata(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":"a"}}}`)
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "metadata") {
		t.Fatalf("carry = %+v, want a full run over undecodable metadata", run.Carry)
	}
	if code, out := runValidate(t, f.root, "--changed-from", "HEAD~1"); code != 2 || !strings.Contains(out, "metadata") {
		t.Fatalf("--changed-from over undecodable metadata: exit=%d %s", code, out)
	}
}

// A gitlink above a declared dependency changes what lies under it. The
// selection is shared by --changed-from and --carry.
func TestChangedScopeGitlinkAboveADependencySelectsItsDependents(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"a":{"selfContained":true},"b":{"selfContained":true,"dependsOn":["vendor/lib"]}}}`)
	first := f.git("rev-parse", "HEAD")
	f.git("update-index", "--add", "--cacheinfo", "160000,"+first+",vendor")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "gitlink")
	second := f.git("rev-parse", "HEAD")
	f.git("update-index", "--add", "--cacheinfo", "160000,"+second+",vendor")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "move the gitlink")
	_, out := runValidate(t, f.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
	if !strings.Contains(out, "+ b: root-level change outside any module: vendor") || !strings.Contains(out, "- a: not affected") {
		t.Fatalf("a gitlink above b's dependency must select b and leave a: %s", out)
	}
}
