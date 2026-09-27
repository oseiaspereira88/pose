package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func initCLITransferProject(t *testing.T, root, projectID, slug string) {
	t.Helper()
	for _, dir := range []string{".pose/policy", ".pose/specs", ".pose/roadmaps", ".pose/review-bundles"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":true,"adopted_at":"2026-09-24","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	body := "---\nslug: " + slug + "\nstatus: in-progress\ncreated_at: 2026-09-24\n---\n# Spec: " + slug + "\n\n## 2 Requirements\n\n- R1: Keep authority explicit.\n\n## 3 Technical Plan\n\nPreserve the operation record.\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose/specs/2026-09-24-"+slug+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "pose-transfer@example.invalid")
	git("config", "user.name", "POSE transfer CLI fixture")
	git("add", ".pose")
	git("commit", "-q", "-m", "fixture "+projectID)
}

func TestSpecTransferCLIEndToEndAndAuthorization(t *testing.T) {
	sourceRoot, destinationRoot := t.TempDir(), t.TempDir()
	initCLITransferProject(t, sourceRoot, "proj.source", "source-work")
	initCLITransferProject(t, destinationRoot, "proj.destination", "destination-work")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.source")
	rootsJSON, _ := json.Marshal(map[string]string{"proj.destination": destinationRoot})
	t.Setenv("POSE_PROJECT_ROOTS", string(rootsJSON))
	t.Setenv("HARNE8_PROJECTS_DIR", "")

	args := []string{"spec-transfer", "preview", "--source", "xref:proj.source/spec:source-work", "--destination", "xref:proj.destination/spec:destination-work", "--map", "R1=equivalent:R1", "--date", "2026-09-24", "--json"}
	out, stderr, code := runCLI(t, sourceRoot, args...)
	if code != 0 {
		t.Fatalf("preview exit=%d stderr=%s", code, stderr)
	}
	var plan posemodel.SpecTransferPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("preview did not return a JSON plan: %v\n%s", err, out)
	}
	if plan.OperationID == "" || plan.Digest == "" || plan.Source.Project != "proj.source" || plan.Destination.Project != "proj.destination" {
		t.Fatalf("incomplete preview plan: %+v", plan)
	}
	planPath := filepath.Join(sourceRoot, "transfer-plan.json")
	if err := os.WriteFile(planPath, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	baseApply := []string{"spec-transfer", "apply", "--plan", planPath, "--digest", plan.Digest, "--authorize-project", "proj.source"}
	if _, stderr, code := runCLI(t, sourceRoot, baseApply...); code == 0 || !strings.Contains(stderr, "write-authorization-required") {
		t.Fatalf("partial authorization accepted: exit=%d stderr=%s", code, stderr)
	}
	apply := append(append([]string{}, baseApply...), "--authorize-project", "proj.destination")
	if _, stderr, code := runCLI(t, sourceRoot, apply...); code != 0 {
		t.Fatalf("authorized apply exit=%d stderr=%s", code, stderr)
	}
	statusArgs := []string{"spec-transfer", "status", "--operation", plan.OperationID, "--project", "proj.destination"}
	out, stderr, code = runCLI(t, sourceRoot, statusArgs...)
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, stderr)
	}
	var status posemodel.SpecTransferStatus
	if err := json.Unmarshal([]byte(out), &status); err != nil || status.Phase != "activated" || status.ProjectID != "proj.destination" {
		t.Fatalf("status = %+v, err=%v, raw=%s", status, err, out)
	}
}

func TestSpecTransferNegativeCLIRejectsMalformedInvocation(t *testing.T) {
	root := newGitRepo(t)
	if _, stderr, code := runCLI(t, root, "spec-transfer"); code != 2 || !strings.Contains(stderr, "subcommand is required") {
		t.Fatalf("missing subcommand: code=%d stderr=%q", code, stderr)
	}
	if _, stderr, code := runCLI(t, root, "spec-transfer", "preview", "--source", "work"); code != 2 || !strings.Contains(stderr, "qualified") {
		t.Fatalf("unqualified source: code=%d stderr=%q", code, stderr)
	}
}

func TestSpecTransferCLIReconcileTerminalReadsModeAndMapFile(t *testing.T) {
	sourceRoot, destinationRoot := t.TempDir(), t.TempDir()
	initCLITransferProject(t, sourceRoot, "proj.source", "source-work")
	initCLITransferProject(t, destinationRoot, "proj.destination", "destination-work")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.source")
	rootsJSON, _ := json.Marshal(map[string]string{"proj.destination": destinationRoot})
	t.Setenv("POSE_PROJECT_ROOTS", string(rootsJSON))
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	mapFile := filepath.Join(t.TempDir(), "map.json")
	if err := os.WriteFile(mapFile, []byte(`[{"source_requirement":"R1","destination_requirement":"R1","disposition":"equivalent"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	base := []string{"spec-transfer", "preview", "--source", "xref:proj.source/spec:source-work", "--destination", "xref:proj.destination/spec:destination-work", "--map-file", mapFile, "--date", "2026-09-26"}
	_, stderr, code := runCLI(t, sourceRoot, append(base, "--mode", "reconcile-terminal")...)
	if code == 0 || !strings.Contains(stderr, "destination-spec-not-terminal") {
		t.Fatalf("reconcile-terminal accepted an open executor: code=%d stderr=%s", code, stderr)
	}
	out, stderr, code := runCLI(t, sourceRoot, base...)
	if code != 0 {
		t.Fatalf("map file was not read for an ordinary transfer: code=%d stderr=%s", code, stderr)
	}
	var plan posemodel.SpecTransferPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil || len(plan.Mappings) != 1 || plan.SchemaVersion != posemodel.SpecTransferSchemaVersion {
		t.Fatalf("plan from map file = %+v err=%v", plan, err)
	}
	if err := os.WriteFile(mapFile, []byte(`[{"source_requirement":"R1","disposition":"equivalent","unknown":true}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := runCLI(t, sourceRoot, base...); code == 0 || !strings.Contains(stderr, "--map-file") {
		t.Fatalf("unknown map field accepted: code=%d stderr=%s", code, stderr)
	}
}

func TestSpecTransferCheckValidatesQualifiedRoadmapMembers(t *testing.T) {
	coordinatorRoot, sourceRoot := t.TempDir(), t.TempDir()
	initCLITransferProject(t, sourceRoot, "proj.source", "source-work")
	initCLITransferProject(t, coordinatorRoot, "proj.coordinator", "local-work")
	roadmap := "---\nslug: program\nstatus: active\n---\n\n## Milestone: core\n- specs: xref:proj.source/spec:source-work, local-work\n\n## Milestone: next\n- after: core, xref:proj.source/spec:source-work\n- specs: xref:proj.unknown/spec:ghost\n"
	if err := os.WriteFile(filepath.Join(coordinatorRoot, ".pose/roadmaps/program.md"), []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.coordinator")
	rootsJSON, _ := json.Marshal(map[string]string{"proj.source": sourceRoot})
	t.Setenv("POSE_PROJECT_ROOTS", string(rootsJSON))
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	out, stderr, _ := runCLI(t, coordinatorRoot, "check")
	all := out + stderr
	for _, wrong := range []string{"missing spec: xref:proj.source/spec:source-work", "after references a missing milestone: xref:proj.source/spec:source-work"} {
		if strings.Contains(all, wrong) {
			t.Fatalf("a resolvable qualified member was reported missing (%s):\n%s", wrong, all)
		}
	}
	if !strings.Contains(all, "roadmap program/next: xref:proj.unknown/spec:ghost:") {
		t.Fatalf("an unresolvable qualified member was not reported:\n%s", all)
	}
}
