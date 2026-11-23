package cli

// Carried checks (spec pose-validate-carries-unchanged-checks): validate
// --carry refreshes the canonical result by running only what a change can
// affect and carrying the rest with the head it really ran at.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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
	if !strings.Contains(out, "+ b: declared dependency vendor/lib changed: vendor") || !strings.Contains(out, "- a: not affected") {
		t.Fatalf("a gitlink above b's dependency must select b and leave a: %s", out)
	}
}

// A dependency beneath a surviving module keeps its dependents selected after
// it is deleted (review round 4, N7).
func TestCarryDeletedNestedDependencySelectsItsDependents(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["a/nested"]}}}`)
	f.write("a/nested/go.mod", "module nested\n")
	f.commit("nested module")
	f.baseline()
	f.git("rm", "-rq", "a/nested")
	f.commit("delete the nested dependency")
	_, _, run := f.validate("--carry")
	if !strings.Contains(run.Carry.Selected["b"], "a/nested") {
		t.Fatalf("selected = %v, want b selected by its deleted nested dependency", run.Carry.Selected)
	}
}

// A dependency on the project root is changed by any file once the root
// module is gone (review round 4, N8).
func TestCarryRootDependencyIsChangedByRootFiles(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"a":{"selfContained":true},"b":{"selfContained":true,"dependsOn":["."]}}}`)
	f.baseline()
	f.write("root.txt", "edited\n")
	f.commit("root change")
	_, _, run := f.validate("--carry")
	if _, ok := run.Carry.Selected["b"]; !ok {
		t.Fatalf("selected = %v, want b selected through its root dependency", run.Carry.Selected)
	}
	if _, ok := run.Carry.Selected["a"]; ok {
		t.Fatalf("selected = %v, want a carried", run.Carry.Selected)
	}
}

// A module reached through a symlink is never carried (review round 4, N9).
func TestCarryRefusesModulesReachedThroughASymlink(t *testing.T) {
	f := newCarryFixture(t, "")
	if err := os.Symlink("a", filepath.Join(f.root, "alias")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks"`, `"moduleOverrides":{"alias":{"stack":"go"}},"stacks"`, 1))
	f.commit("alias")
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "symlink") {
		t.Fatalf("carry = %+v, want an aliased module refused", run.Carry)
	}
}

// A run whose HEAD moved while checks ran did not validate its recorded head
// (review round 4, N10).
func TestValidateHeadMovingDuringTheRunIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	commit := `git -C .. -c user.name=t -c user.email=t@t -c core.hooksPath=/dev/null commit -q --allow-empty -m moved`
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks":{"go":{"checks":[`, `"moduleOverrides":{"a":{"checks":[{"name":"moves","program":"sh","args":["-c","`+commit+`"],"severity":"required","evidenceClass":"unit"}]}},"stacks":{"go":{"checks":[`, 1))
	f.commit("a check that commits")
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true although HEAD moved during the run: %s", out)
	}
}

// Every eligible dependency counts, whatever its order (review round 5, N11).
func TestChangedScopeEvaluatesEveryDeclaredDependency(t *testing.T) {
	for _, deps := range []string{`["a","."]`, `[".","a"]`} {
		f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":`+deps+`}}}`)
		f.write("a/nested/go.mod", "module nested\n")
		f.commit("nested module")
		f.write("a/nested/changed.go", "package nested\n")
		f.commit("change nested")
		_, out := runValidate(t, f.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
		if !strings.Contains(out, "+ b: declared dependency ") {
			t.Fatalf("dependsOn %s: b must be selected whatever the order: %s", deps, out)
		}
	}
}

// A run whose HEAD moved binds no material (review round 5, N12).
func TestValidateHeadMovingDuringTheRunBindsNoMaterial(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write(".pose/specs/2026-11-23-carry-fixture.md", "---\nslug: carry-fixture\nstatus: in-progress\ncreated_at: 2026-11-23\n---\n\n# Spec: carry-fixture\n\n## 3. Technical Plan\n\n### Artifacts\n- created: a/feature.go\n")
	f.write("a/feature.go", "package a\n")
	f.git("add", "-A")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "feature\n\nPOSE-Spec: carry-fixture")
	if _, out, run := f.validate(); len(run.MaterialProvenance) == 0 {
		t.Fatalf("the fixture must bind material for this test to mean anything: %s", out)
	}
	f.commit("record")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	commit := `git -C .. -c user.name=t -c user.email=t@t -c core.hooksPath=/dev/null commit -q --allow-empty -m moved`
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks":{"go":{"checks":[`, `"moduleOverrides":{"a":{"checks":[{"name":"moves","program":"sh","args":["-c","`+commit+`"],"severity":"required","evidenceClass":"unit"}]}},"stacks":{"go":{"checks":[`, 1))
	f.commit("a check that commits")
	_, _, run := f.validate()
	if run.MaterialClean {
		t.Fatal("material_clean must be false")
	}
	for spec, binding := range run.MaterialProvenance {
		if binding != posemodel.MaterialBindingUnavailable {
			t.Fatalf("binding of %s = %s, want unavailable when HEAD moved", spec, binding)
		}
	}
}

// Two modules on one physical directory alias each other (review round 5,
// N13); a case alias cannot be made on this filesystem, so stat is faked.
func TestCarryRefusesModulesSharingADirectory(t *testing.T) {
	dir := t.TempDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	modules := []validationModule{{Rel: "a", Abs: "/repo/a"}, {Rel: "A", Abs: "/repo/A"}}
	same := func(string) (os.FileInfo, error) { return info, nil }
	identity := func(p string) (string, error) { return p, nil }
	if reason := aliasedModule(modules, identity, same); !strings.Contains(reason, "shares its directory") {
		t.Fatalf("reason = %q, want modules sharing a directory refused", reason)
	}
	distinct := func(p string) (os.FileInfo, error) { return os.Stat(map[string]string{"/repo/a": dir, "/repo/A": t.TempDir()}[p]) }
	if reason := aliasedModule(modules, identity, distinct); reason != "" {
		t.Fatalf("distinct directories refused: %q", reason)
	}
}

// Selection uses the head pinned before planning, not whatever HEAD is later
// (review round 5, N14).
func TestCarryPlansAgainstThePinnedHead(t *testing.T) {
	f := newCarryFixture(t, "")
	f.baseline()
	pinned := f.git("rev-parse", "HEAD")
	f.write("a/changed.go", "package a\n")
	f.commit("a change after the pin")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		t.Fatal(err)
	}
	modules, err := discoverValidationModules(f.root)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	plan := planCarry(f.root, carryResults, "strict", "sha256:"+hex.EncodeToString(digest[:]), pinned, matrix, modules)
	if plan.record.Used && len(plan.record.Selected) != 0 {
		t.Fatalf("selected = %v, want nothing changed up to the pinned head", plan.record.Selected)
	}
	if !plan.record.Used && !strings.Contains(plan.record.Reason, "engine") {
		t.Fatalf("carry unused for an unexpected reason: %q", plan.record.Reason)
	}
}

// A discovered dependency above the changed module selects its dependents
// (review round 6, N15), and so does a discovered root dependency.
func TestChangedScopeDiscoveredAncestorDependencySelectsItsDependents(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["a"]}}}`)
	f.write("a/nested/go.mod", "module nested\n")
	f.commit("nested module")
	f.write("a/nested/input.txt", "x\n")
	f.commit("change nested")
	_, out := runValidate(t, f.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
	if !strings.Contains(out, "+ b: declared dependency a changed: a/nested/input.txt") {
		t.Fatalf("b must be selected through its discovered ancestor dependency: %s", out)
	}

	g := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["."]}}}`)
	g.write("go.mod", "module root\n")
	g.commit("root module")
	g.write("a/changed.go", "package a\n")
	g.commit("change a")
	_, out = runValidate(t, g.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
	if !strings.Contains(out, "+ b: declared dependency . changed: a/changed.go") {
		t.Fatalf("b must be selected through its discovered root dependency: %s", out)
	}
}

// A declared dependency reached through a symlink refuses the carry (review
// round 6, N16).
func TestCarryRefusesDependenciesReachedThroughASymlink(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["link"]}}}`)
	if err := os.Symlink("a", filepath.Join(f.root, "link")); err != nil {
		t.Fatal(err)
	}
	f.commit("link")
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "symlink link") {
		t.Fatalf("carry = %+v, want an aliased dependency refused", run.Carry)
	}
}

// A symlink inside a module refuses the carry even when the dependency it
// leads to is absent (review round 7, N17).
func TestCarryRefusesASymlinkInsideAModule(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["b/link/input"]}}}`)
	f.write("a/input", "x\n")
	if err := os.Symlink("../a", filepath.Join(f.root, "b", "link")); err != nil {
		t.Fatal(err)
	}
	f.commit("link from b into a")
	f.baseline()
	f.git("rm", "-q", "a/input")
	f.commit("delete the input")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "symlink b/link") {
		t.Fatalf("carry = %+v, want a symlink inside a module refused", run.Carry)
	}
}

// Symlinks between paths no module owns do not block a carry.
func TestCarryAllowsSymlinksBetweenUnownedPaths(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write(".agents/skills/x/SKILL.md", "skill\n")
	if err := os.MkdirAll(filepath.Join(f.root, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../.agents/skills/x", filepath.Join(f.root, ".claude", "skills", "x")); err != nil {
		t.Fatal(err)
	}
	f.commit("skills link")
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || !run.Carry.Used || checkOf(run, "b").CarriedFrom == "" {
		t.Fatalf("carry = %+v, want b carried despite an unowned symlink", run.Carry)
	}
}

// A case-folding filesystem refuses the carry (review round 7, N18); faked,
// since this filesystem is case-sensitive.
func TestCarryDetectsACaseInsensitiveFilesystem(t *testing.T) {
	dir := t.TempDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	folding := func(string) (os.FileInfo, error) { return info, nil }
	if !caseInsensitiveFilesystem("/repo", folding) {
		t.Fatal("a filesystem where .GIT names .git must read as case-insensitive")
	}
	sensitive := func(p string) (os.FileInfo, error) {
		if strings.HasSuffix(p, ".GIT") {
			return nil, os.ErrNotExist
		}
		return info, nil
	}
	if caseInsensitiveFilesystem("/repo", sensitive) {
		t.Fatal("a case-sensitive filesystem refused")
	}
}

// Two checks with one id never share a base result (review round 7, N19).
func TestCarryRefusesDuplicateCheckIDs(t *testing.T) {
	f := newCarryFixture(t, "")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks"`, `"moduleOverrides":{"b":{"checks":[{"name":"ok","program":"true","severity":"required","evidenceClass":"unit"}]}},"stacks"`, 1))
	f.commit("duplicate check")
	f.validate()
	f.commit("record")
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "more than once") {
		t.Fatalf("carry = %+v, want duplicate check ids refused", run.Carry)
	}
}

// A symlink inside a self-contained module that leads to a path no module owns
// would hide that path's changes from the module (review round 7, N17).
func TestCarryRefusesASymlinkFromAModuleToAnUnownedPath(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true}}}`)
	f.write("shared/input", "x\n")
	if err := os.Symlink("../shared", filepath.Join(f.root, "b", "shared")); err != nil {
		t.Fatal(err)
	}
	f.commit("link from b to an unowned path")
	f.baseline()
	f.write("shared/input", "y\n")
	f.commit("change the shared input")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "symlink b/shared") {
		t.Fatalf("carry = %+v, want a module's outward symlink refused", run.Carry)
	}
}

// Configuration never hides a changed gitlink (review round 8, N23).
func TestChangedScopeIgnoresSubmoduleHidingConfiguration(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "diff.ignoreSubmodules")
	t.Setenv("GIT_CONFIG_VALUE_0", "all")
	TestChangedScopeGitlinkAboveADependencySelectsItsDependents(t)
}

// Untracked material created while checks run means the run did not validate
// its recorded head (review round 8, N21).
func TestValidateUntrackedMaterialCreatedDuringTheRunIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks"`, `"moduleOverrides":{"a":{"checks":[{"name":"writes","program":"sh","args":["-c","echo x > created.go"],"severity":"required","evidenceClass":"unit"}]}},"stacks"`, 1))
	f.commit("a check that writes an input")
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true although the run created untracked material: %s", out)
	}
}

// Symlinks inside a submodule are scanned at its recorded commit (review
// round 8, N20).
func TestCarryRefusesAnAliasingSymlinkInsideASubmodule(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["sub/link/input"]}}}`)
	sub := filepath.Join(f.root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run(sub, "init", "-q")
	if err := os.Symlink("../a", filepath.Join(sub, "link")); err != nil {
		t.Fatal(err)
	}
	run(sub, "add", "-A")
	run(sub, "-c", "core.hooksPath=/dev/null", "commit", "-qm", "link out of the submodule")
	f.commit("sub as a gitlink")
	if links, _ := gitlinks(f.root, "HEAD"); links["sub"] == "" {
		t.Fatalf("the fixture must track sub as a gitlink: %v", links)
	}
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, result := f.validate("--carry")
	if result.Carry == nil || result.Carry.Used || !strings.Contains(result.Carry.Reason, "symlink sub/link") {
		t.Fatalf("carry = %+v, want the submodule's aliasing symlink refused", result.Carry)
	}
}

// A case-folded directory is detected inside a case-sensitive filesystem
// (review round 8, N22); faked, since this filesystem does not fold.
func TestCarryDetectsACaseFoldedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "input"))
	if err != nil {
		t.Fatal(err)
	}
	folding := func(p string) (os.FileInfo, error) {
		if strings.EqualFold(filepath.Base(p), "input") {
			return info, nil
		}
		return os.Stat(p)
	}
	if !casefoldedDirectory(dir, os.ReadDir, folding) {
		t.Fatal("a directory where INPUT names input must read as case-folded")
	}
	if casefoldedDirectory(dir, os.ReadDir, os.Stat) {
		t.Fatal("a case-sensitive directory read as case-folded")
	}
}

// An enclosing module is selected when a module nested in it changes (review
// round 9, N25).
func TestChangedScopeSelectsTheEnclosingModule(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/n/go.mod", "module n\n")
	f.commit("nested module in b")
	f.write("b/n/input", "x\n")
	f.commit("change inside the nested module")
	_, out := runValidate(t, f.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
	if !strings.Contains(out, "+ b/n: contains changed file: b/n/input") || !strings.Contains(out, "+ b: changed file in nested module b/n: b/n/input") {
		t.Fatalf("both the owner and its enclosing module must be selected: %s", out)
	}
}

// A gitlink inside a module is scanned too (review round 9, N24).
func TestCarryScansAGitlinkInsideAModule(t *testing.T) {
	f := newCarryFixture(t, "")
	sub := filepath.Join(f.root, "b", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git(sub, "init", "-q")
	if err := os.Symlink("../../a", filepath.Join(sub, "link")); err != nil {
		t.Fatal(err)
	}
	git(sub, "add", "-A")
	git(sub, "-c", "core.hooksPath=/dev/null", "commit", "-qm", "link out")
	f.commit("gitlink inside b")
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "symlink b/sub/link") {
		t.Fatalf("carry = %+v, want the aliasing symlink in the module's gitlink refused", run.Carry)
	}
}

// Case behaviour is probed on every directory up to the root (review round 9,
// N26).
func TestCarryRelevantDirsIncludeEveryAncestor(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shared", "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := scopeMetadata{Modules: map[string]scopeModuleMeta{"b": {DependsOn: []string{"shared/A/input"}}}}
	dirs := carryRelevantDirs(root, nil, meta)
	want := map[string]bool{root: false, filepath.Join(root, "shared"): false}
	for _, d := range dirs {
		if _, ok := want[d]; ok {
			want[d] = true
		}
	}
	for d, found := range want {
		if !found {
			t.Fatalf("dirs = %v, missing ancestor %s", dirs, d)
		}
	}
}

// The root module encloses every other module.
func TestChangedScopeRootModuleEnclosesEveryModule(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("go.mod", "module root\n")
	f.commit("root module")
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, out := runValidate(t, f.root, "--changed-from", "HEAD~1", "--changed-to", "HEAD", "--explain")
	if !strings.Contains(out, "+ .: changed file in nested module a: a/changed.go") || !strings.Contains(out, "- b: not affected") {
		t.Fatalf("the root module must be selected as the enclosing module, b left: %s", out)
	}
}

// An unusable entry does not end the case probe (review round 10, N27).
func TestCarryCaseProbeSkipsUnusableEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("../missing", filepath.Join(dir, "0-dangling")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "input"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dir, "input"))
	if err != nil {
		t.Fatal(err)
	}
	folding := func(p string) (os.FileInfo, error) {
		switch filepath.Base(p) {
		case "0-dangling", "0-DANGLING":
			return nil, os.ErrNotExist
		}
		if strings.EqualFold(filepath.Base(p), "input") {
			return info, nil
		}
		return os.Lstat(p)
	}
	if !casefoldedDirectory(dir, os.ReadDir, folding) {
		t.Fatal("a folding directory must be detected past an unusable first entry")
	}
	unreadable := func(string) ([]os.DirEntry, error) { return nil, os.ErrPermission }
	if !casefoldedDirectory(dir, unreadable, os.Lstat) {
		t.Fatal("an unreadable directory must not count as case-sensitive")
	}
}

// A dependency outside the repository refuses the carry (review round 10,
// N28).
func TestCarryRefusesDependenciesOutsideTheRepository(t *testing.T) {
	for _, dep := range []string{"../shared/input", "/etc/shared"} {
		f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["`+dep+`"]}}}`)
		f.baseline()
		f.write("a/changed.go", "package a\n")
		f.commit("change a")
		_, _, run := f.validate("--carry")
		if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "outside the repository") {
			t.Fatalf("dependsOn %s: carry = %+v, want an external dependency refused", dep, run.Carry)
		}
	}
}

// A dependency that is not already clean refuses the carry (review round 11,
// N30).
func TestCarryRefusesUncleanDependencies(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["unowned/link/../input"]}}}`)
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "not a clean path") {
		t.Fatalf("carry = %+v, want an unclean dependency refused", run.Carry)
	}
}

// Non-ASCII module or dependency paths refuse the carry (review round 11,
// N32).
func TestCarryRefusesNonASCIIPaths(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["shared/café.txt"]}}}`)
	f.baseline()
	f.write("a/changed.go", "package a\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "not an ASCII path") {
		t.Fatalf("carry = %+v, want a non-ASCII dependency refused", run.Carry)
	}
}

// An index entry hiding edits from status means the run validated no known
// tree, and refuses the carry (review round 11, N31).
func TestCarryRefusesHiddenIndexEntries(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/input", "x\n")
	f.commit("b input")
	f.git("update-index", "--skip-worktree", "b/input")
	if _, out, run := f.validate(); run.MaterialClean {
		t.Fatalf("material_clean = true with a skip-worktree entry: %s", out)
	}
	f.git("update-index", "--no-skip-worktree", "b/input")
	f.baseline()
	f.git("update-index", "--assume-unchanged", "b/input")
	f.write("a/changed.go", "package a\n")
	f.git("add", "a/changed.go")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "assume-unchanged or skip-worktree") {
		t.Fatalf("carry = %+v, want a hidden index entry refused", run.Carry)
	}
}

// A dependency reached through an ignored symlink refuses the carry (review
// round 12, N33).
func TestCarryRefusesDependenciesThroughAnIgnoredSymlink(t *testing.T) {
	f := newCarryFixture(t, `{"schemaVersion":1,"modules":{"b":{"selfContained":true,"dependsOn":["link/input"]}}}`)
	f.write(".gitignore", ".pose/transfers/.authority-transfer.lock\nlink\n")
	f.commit("ignore link")
	if err := os.Symlink("a", filepath.Join(f.root, "link")); err != nil {
		t.Fatal(err)
	}
	f.baseline()
	f.write("a/input", "x\n")
	f.commit("change a")
	_, _, run := f.validate("--carry")
	if run.Carry == nil || run.Carry.Used || !strings.Contains(run.Carry.Reason, "resolves through a symlink") {
		t.Fatalf("carry = %+v, want an ignored-symlink dependency refused", run.Carry)
	}
}

// HEAD moving away and back during a run is still movement (review round 12,
// N34).
func TestValidateHeadMovingAndReturningIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	script := `git -C .. -c user.name=t -c user.email=t@t -c core.hooksPath=/dev/null commit -q --allow-empty -m moved && git -C .. reset -q --hard HEAD~1`
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks"`, `"moduleOverrides":{"a":{"checks":[{"name":"moves","program":"sh","args":["-c","`+script+`"],"severity":"required","evidenceClass":"unit"}]}},"stacks"`, 1))
	f.commit("a check that moves HEAD and back")
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true although HEAD moved and came back: %s", out)
	}
}

// The case probe ignores entries whose case swap is not a plain ASCII fold
// (review round 12, N35).
func TestCarryCaseProbeUsesOnlyASCIILetters(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"0bı", "a"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Lstat(filepath.Join(dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	folding := func(p string) (os.FileInfo, error) {
		if base := filepath.Base(p); base == "a" || base == "A" {
			return info, nil
		}
		return os.Lstat(p)
	}
	if !casefoldedDirectory(dir, os.ReadDir, folding) {
		t.Fatal("a folding directory must be detected despite a non-ASCII first entry")
	}
	empty := t.TempDir()
	if !casefoldedDirectory(empty, os.ReadDir, os.Lstat) {
		t.Fatal("a directory without an ASCII letter entry cannot be shown not to fold")
	}
}

// A mode change hidden by core.fileMode=false still makes the run unclean
// (review round 13, N36).
func TestValidateModeChangeHiddenByConfigurationIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/check.sh", "#!/bin/sh\nexit 0\n")
	f.commit("non-executable script")
	f.git("config", "core.fileMode", "false")
	if err := os.Chmod(filepath.Join(f.root, "b", "check.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true although b/check.sh differs in mode from the commit: %s", out)
	}
}

// An uninitialized submodule is not a clean base, and refuses the carry
// (review round 13, N38).
func TestValidateUninitializedSubmoduleIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	first := f.git("rev-parse", "HEAD")
	f.git("update-index", "--add", "--cacheinfo", "160000,"+first+",b/data")
	f.git("-c", "core.hooksPath=/dev/null", "commit", "-qm", "uninitialized gitlink in b")
	// An uninitialized submodule is an empty directory, which status reports
	// as clean.
	if err := os.MkdirAll(filepath.Join(f.root, "b", "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !materialWorktreeClean(f.root) {
		t.Fatal("the fixture must look clean to status for this test to mean anything")
	}
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true with an uninitialized submodule: %s", out)
	}
	if sub := submodulesCheckedOut(f.root, "HEAD"); sub != "b/data" {
		t.Fatalf("submodulesCheckedOut = %q, want b/data", sub)
	}
}

// A condition that holds only at the end of the run does not make it clean:
// a hidden index entry cleared during the run (review round 14, N40).
func TestValidateConditionClearedDuringTheRunIsNotClean(t *testing.T) {
	f := newCarryFixture(t, "")
	f.write("b/input", "x\n")
	f.commit("b input")
	raw, _ := os.ReadFile(filepath.Join(f.root, ".pose/indexes/validation-matrix.json"))
	script := `git -C .. update-index --no-skip-worktree b/input`
	f.write(".pose/indexes/validation-matrix.json", strings.Replace(string(raw), `"stacks"`, `"moduleOverrides":{"a":{"checks":[{"name":"clears","program":"sh","args":["-c","`+script+`"],"severity":"required","evidenceClass":"unit"}]}},"stacks"`, 1))
	f.commit("a check that clears the flag")
	f.git("update-index", "--skip-worktree", "b/input")
	_, out, run := f.validate()
	if run.MaterialClean {
		t.Fatalf("material_clean = true although the run started with a hidden index entry: %s", out)
	}
}
