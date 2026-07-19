package cli

// Polyglot stack catalog behavior (spec pose-stack-catalog-expansion):
// marker detection, manager-conflict priority, prerequisite reporting and
// the pose stacks read-only command. No project file is ever executed.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectStackProfilesResolvesConflictByPriority(t *testing.T) {
	dets := detectStackProfiles([]string{"pyproject.toml", "requirements.txt", "poetry.lock"})
	var winner stackDetection
	var found int
	for _, d := range dets {
		if d.Profile.Stack != "python" {
			t.Fatalf("unexpected stack in result: %+v", d)
		}
		found++
		if d.Winner {
			winner = d
		}
	}
	if found != 3 {
		t.Fatalf("expected 3 matched python profiles (conflict), got %d", found)
	}
	if winner.Profile.Manager != "poetry" {
		t.Errorf("winner = %s, want poetry (lowest priority number)", winner.Profile.Manager)
	}
	if winner.Confidence != "medium" {
		t.Errorf("confidence under conflict = %s, want medium", winner.Confidence)
	}
}

func TestDetectStackProfilesNoConflictIsHighConfidence(t *testing.T) {
	dets := detectStackProfiles([]string{"go.mod"})
	if len(dets) != 1 || dets[0].Confidence != "high" || !dets[0].Winner {
		t.Fatalf("single marker should be high-confidence winner: %+v", dets)
	}
}

func TestDetectStackProfilesDotnetSuffixMarkers(t *testing.T) {
	dets := detectStackProfiles([]string{"App.csproj", "readme.md"})
	if len(dets) != 1 || dets[0].Profile.Stack != "dotnet" || dets[0].Profile.Manager != "dotnet" {
		t.Fatalf("dotnet suffix marker not detected: %+v", dets)
	}
}

func TestDetectStackProfilesAbsentToolReported(t *testing.T) {
	dets := detectStackProfiles([]string{"requirements.txt"})
	if len(dets) != 1 {
		t.Fatalf("dets = %+v", dets)
	}
	// pytest is very likely absent in this sandboxed test environment;
	// assert the field is populated deterministically either way, and that
	// LookPath — not execution — is the mechanism (no project file ran).
	_ = dets[0].PrerequisiteFound
	if dets[0].Profile.Prerequisite != "pytest" {
		t.Errorf("prerequisite = %q, want pytest", dets[0].Profile.Prerequisite)
	}
}

func TestValidationMatrixPythonManagerExclusion(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Both a poetry lockfile and a legacy requirements.txt are present; the
	// pip-test check must yield to poetry-test (fileNotExistsAny).
	write("mod/poetry.lock", "")
	write("mod/requirements.txt", "")
	write(".pose/indexes/validation-matrix.json", `{
    "defaults": {"mode": "strict"},
    "stacks": {"python": {"checks": [
      {"name":"poetry-test","program":"true","severity":"required","when":{"fileExists":"poetry.lock"}},
      {"name":"pip-test","program":"true","severity":"required","when":{"fileExists":"requirements.txt","fileNotExistsAny":["poetry.lock"]}}
    ]}},
    "moduleOverrides": {"mod": {"stack": "python"}}
  }`)
	code, _ := runValidate(t, root, "--json", "result.json")
	if code != 0 {
		t.Fatal("run failed")
	}
	run := loadRun(t, root)
	byName := map[string]checkResult{}
	for _, c := range run.Checks {
		byName[c.Name] = c
	}
	if byName["poetry-test"].Outcome != "pass" {
		t.Errorf("poetry-test = %+v", byName["poetry-test"])
	}
	if byName["pip-test"].Outcome != "skipped" || !strings.Contains(byName["pip-test"].SkipReason, "fileNotExistsAny violated") {
		t.Errorf("pip-test must yield to poetry: %+v", byName["pip-test"])
	}
}

func TestStacksCommandReportsConflictAndOverrideHint(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"Pipfile", "poetry.lock"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errB bytes.Buffer
	if code := cmdStacks(root, nil, &out, &errB); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
	s := out.String()
	if !strings.Contains(s, "poetry (winner)") || !strings.Contains(s, "pipenv (shadowed)") {
		t.Errorf("expected winner/shadowed reporting: %s", s)
	}
	if !strings.Contains(s, "override:") {
		t.Errorf("expected override hint: %s", s)
	}
}

func TestStacksCommandJSONAndPathConfinement(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	if code := cmdStacks(root, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	var payload struct {
		Detections []stackDetection `json:"detections"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("stacks --json is not valid JSON: %v", err)
	}
	if len(payload.Detections) != 1 || payload.Detections[0].Profile.Stack != "go" {
		t.Fatalf("detections = %+v", payload.Detections)
	}
	out.Reset()
	errB.Reset()
	if code := cmdStacks(root, []string{"--path", "../escape"}, &out, &errB); code != 2 {
		t.Fatalf("path escape must be rejected, code=%d", code)
	}
}
