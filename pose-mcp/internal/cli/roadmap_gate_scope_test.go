package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// roadmapGateScopeFixture holds a roadmap whose first milestone owns a done
// spec, whose second owns an open one, and whose cut criterion cannot pass.
func roadmapGateScopeFixture(t *testing.T, extraMember string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "coordinator")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	mustWrite(t, filepath.Join(root, ".pose", "specs", "finished", "spec.md"), "---\nslug: finished\nstatus: done\ncreated_at: 2026-09-26\n---\n\n# Spec: finished\n")
	mustWrite(t, filepath.Join(root, ".pose", "specs", "pending", "spec.md"), "---\nslug: pending\nstatus: draft\ncreated_at: 2026-09-26\n---\n\n# Spec: pending\n")
	later := "pending"
	if extraMember != "" {
		later += ", " + extraMember
	}
	mustWrite(t, filepath.Join(root, ".pose", "roadmaps", "program.md"), `---
slug: program
status: active
---

## Milestone: core
- specs: finished

## Milestone: later
- after: core
- specs: `+later+`

## Cut criteria
- C1: check:never-produced
`)
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.name", "POSE fixture")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "add", "--all")
	artifactGit(t, root, "commit", "-q", "-m", "roadmap gate fixture")
	return root
}

func roadmapGateBlockers(t *testing.T, root string) []string {
	t.Helper()
	out, errOut, _ := runCLI(t, root, "roadmap-check", "program", "--json")
	var report struct {
		Blockers []string `json:"blockers"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("roadmap-check returned invalid JSON: %v\n%s%s", err, out, errOut)
	}
	return report.Blockers
}

func TestRoadmapCheckGateMilestoneIgnoresLaterMembersAndCutCriteria(t *testing.T) {
	root := roadmapGateScopeFixture(t, "")
	blockers := strings.Join(roadmapGateBlockers(t, root), " ")
	if !strings.Contains(blockers, "member spec is not terminal: pending") || !strings.Contains(blockers, "C1:") {
		t.Fatalf("roadmap gate lost its later member or its cut criterion: %s", blockers)
	}
	core, err := roadmapGate(root, "program", "core")
	if err != nil || len(core.blockers) != 0 || core.federated.Manifest.Coordinator.Kind != "milestone" {
		t.Fatalf("earlier milestone gate was not scoped to its own edges: %+v err=%v", core.blockers, err)
	}
	later, err := roadmapGate(root, "program", "later")
	joined := strings.Join(later.blockers, " ")
	if err != nil || !strings.Contains(joined, "member spec is not terminal: pending") {
		t.Fatalf("later milestone gate ignored its own open member: %v err=%v", later.blockers, err)
	}
	if strings.Contains(joined, "C1:") {
		t.Fatalf("milestone gate applied the roadmap cut criterion: %s", joined)
	}
	if _, err := roadmapGate(root, "program", "missing"); err == nil {
		t.Fatal("unknown milestone passed the gate")
	}
}

func TestRoadmapCheckGateJudgesExternalMembersByFederation(t *testing.T) {
	root := roadmapGateScopeFixture(t, "xref:source/spec:backend, xref:coordinator/spec:finished")
	blockers := strings.Join(roadmapGateBlockers(t, root), " ")
	if strings.Contains(blockers, "member spec is not terminal: xref:source/spec:backend") {
		t.Fatalf("external member was judged by local closeout: %s", blockers)
	}
	if !strings.Contains(blockers, "xref:source/spec:backend:") {
		t.Fatalf("external member left the federated gate: %s", blockers)
	}
	if strings.Contains(blockers, "member spec is not terminal: xref:coordinator/spec:finished") {
		t.Fatalf("a locally qualified done member was not resolved by its slug: %s", blockers)
	}
}
