package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assessTestRoot builds a minimal POSE root in a temp dir (no git — commit
// staleness must degrade to "unknown", never fail).
func assessTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".pose/specs", ".pose/policy"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runAssess(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cmdAssess(root, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestAssessEndToEnd(t *testing.T) {
	root := assessTestRoot(t)

	// Bare assess without an artifact: nominal error pointing at init.
	code, _, errOut := runAssess(t, root)
	if code == 0 || !strings.Contains(errOut, "pose assess init") {
		t.Fatalf("missing artifact must fail nominally, got code=%d stderr=%q", code, errOut)
	}

	// init scaffolds 16 mechanisms; re-init refuses.
	if code, out, _ := runAssess(t, root, "init"); code != 0 || !strings.Contains(out, "16") {
		t.Fatalf("init failed: code=%d out=%q", code, out)
	}
	if code, _, errOut := runAssess(t, root, "init"); code == 0 || !strings.Contains(errOut, "already exists") {
		t.Fatalf("re-init must refuse, got code=%d stderr=%q", code, errOut)
	}

	// Scaffold validates clean (no git → commits lag unknown, no error).
	code, out, errOut := runAssess(t, root)
	if code != 0 {
		t.Fatalf("scaffold must validate: code=%d out=%q err=%q", code, out, errOut)
	}
	if !strings.Contains(out, "16") || !strings.Contains(out, "unknown") {
		t.Fatalf("summary must report 16 mechanisms and unknown lag: %q", out)
	}

	// First snapshot appends; identical second run is a no-op.
	if code, out, _ := runAssess(t, root, "snapshot"); code != 0 || !strings.Contains(out, "16") {
		t.Fatalf("snapshot failed: code=%d out=%q", code, out)
	}
	if code, out, _ := runAssess(t, root, "snapshot"); code != 0 || !strings.Contains(out, "No change") {
		t.Fatalf("identical snapshot must be a no-op: code=%d out=%q", code, out)
	}

	// Edit a score, snapshot again, diff sees the raise.
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), "- score: 0", "- score: 4", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runAssess(t, root, "snapshot"); code != 0 {
		t.Fatalf("second snapshot failed: %q", errOut)
	}
	code, out, _ = runAssess(t, root, "diff")
	if code != 0 || !strings.Contains(out, "0 -> 4") {
		t.Fatalf("diff must show the raise: code=%d out=%q", code, out)
	}
	code, out, _ = runAssess(t, root, "diff", "--json")
	if code != 0 || !strings.Contains(out, `"raised"`) {
		t.Fatalf("diff --json must emit structured output: %q", out)
	}
}

func TestAssessDanglingEvidenceFailsNominally(t *testing.T) {
	root := assessTestRoot(t)
	if code, _, _ := runAssess(t, root, "init"); code != 0 {
		t.Fatal("init failed")
	}
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, _ := os.ReadFile(path)
	edited := strings.Replace(string(raw), "- evidence:\n", "- evidence: spec:ghost-spec\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runAssess(t, root)
	if code == 0 || !strings.Contains(out, "ghost-spec") {
		t.Fatalf("dangling spec ref must fail with the slug named: code=%d out=%q", code, out)
	}
	// Snapshot refuses while validation fails.
	if code, _, errOut := runAssess(t, root, "snapshot"); code == 0 || !strings.Contains(errOut, "validation") {
		t.Fatalf("snapshot must refuse on errors: code=%d stderr=%q", code, errOut)
	}
}

func TestAssessStableIDContractViaHistory(t *testing.T) {
	root := assessTestRoot(t)
	if code, _, _ := runAssess(t, root, "init"); code != 0 {
		t.Fatal("init failed")
	}
	if code, _, _ := runAssess(t, root, "snapshot"); code != 0 {
		t.Fatal("snapshot failed")
	}
	// Remove a published mechanism instead of retiring it.
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, _ := os.ReadFile(path)
	content := string(raw)
	start := strings.Index(content, "## Mechanism: multi-repo-enterprise")
	if start < 0 {
		t.Fatal("fixture drift: mechanism not found")
	}
	if err := os.WriteFile(path, []byte(content[:start]), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runAssess(t, root)
	if code == 0 || !strings.Contains(out, "multi-repo-enterprise") || !strings.Contains(out, "retire") {
		t.Fatalf("removing a published mechanism must fail pointing at retirement: code=%d out=%q", code, out)
	}
}

func TestAssessStalenessWarnsByPolicy(t *testing.T) {
	root := assessTestRoot(t)
	if code, _, _ := runAssess(t, root, "init"); code != 0 {
		t.Fatal("init failed")
	}
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, _ := os.ReadFile(path)
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "assessed_at: ") {
			lines[i] = "assessed_at: 2020-01-01"
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runAssess(t, root)
	if code != 0 {
		t.Fatalf("staleness is a warning, not an error: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "[AVISO]") || !strings.Contains(out, "days old") {
		t.Fatalf("old assessment must warn by policy: %q", out)
	}

	// Tighter policy stays configurable.
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "capabilities.json"),
		[]byte(`{"stale_after_days": 100000, "stale_after_commits": 200}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ = runAssess(t, root)
	if strings.Contains(out, "days old") {
		t.Fatalf("policy override must silence the day warning: %q", out)
	}
}

func TestCheckCapabilitiesOptIn(t *testing.T) {
	// nativeChecker with no artifact: no issues added.
	root := assessTestRoot(t)
	var out bytes.Buffer
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
	checker.checkCapabilities()
	if checker.errors != 0 || checker.warnings != 0 {
		t.Fatalf("absent artifact must be a no-op: errors=%d warnings=%d", checker.errors, checker.warnings)
	}

	// Planted dangling ref: check fails nominally.
	if code, _, _ := runAssess(t, root, "init"); code != 0 {
		t.Fatal("init failed")
	}
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, _ := os.ReadFile(path)
	edited := strings.Replace(string(raw), "- evidence:\n", "- evidence: adr:ghost.md\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	checker = &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
	checker.checkCapabilities()
	if checker.errors == 0 || !strings.Contains(out.String(), "ghost.md") {
		t.Fatalf("dangling ref must fail the check: errors=%d out=%q", checker.errors, out.String())
	}
}

func TestAssessDiffAgainstAuthorizationBoundary(t *testing.T) {
	local := assessTestRoot(t)
	other := assessTestRoot(t)
	if code, _, _ := runAssess(t, local, "init"); code != 0 {
		t.Fatal("init local failed")
	}
	if code, _, _ := runAssess(t, other, "init"); code != 0 {
		t.Fatal("init other failed")
	}

	// Unauthorized project id: nominal refusal, no read.
	code, _, errOut := runAssess(t, local, "diff", "--against", "proj.ghost")
	if code == 0 || !strings.Contains(errOut, "authorized") {
		t.Fatalf("unauthorized root must refuse: code=%d stderr=%q", code, errOut)
	}

	// Authorized via POSE_PROJECT_ROOTS: matrix renders both columns.
	t.Setenv("POSE_PROJECT_ROOTS", `{"proj.other":"`+strings.ReplaceAll(other, `\`, `\\`)+`"}`)
	code, out, errOut := runAssess(t, local, "diff", "--against", "proj.other")
	if code != 0 {
		t.Fatalf("authorized --against failed: %q", errOut)
	}
	if !strings.Contains(out, "proj.other") || !strings.Contains(out, "0/3") {
		t.Fatalf("matrix must show the other project's scores: %q", out)
	}
}
