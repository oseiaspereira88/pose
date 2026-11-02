package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// reachedMilestoneFixture puts spec bundle in milestone first of program, with
// milestone second after it.
func reachedMilestoneFixture(t *testing.T) string {
	t.Helper()
	root := reviewBundleCLIFixture(t)
	policy := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, policy, strings.Replace(string(raw), `"profiles":{"spec":"spec-closeout@2"},`, `"profiles":{"spec":"spec-closeout@2","milestone":"milestone-integration@1"},`, 1))
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/milestone-integration.json", `{"schema_version":2,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"correctness","description":"reviewed","evidence_classes":["unit"]}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/waiting/spec.md", "---\nslug: waiting\nstatus: draft\ncreated_at: 2026-08-13\n---\n\n# Spec: waiting\n")
	writeCloseoutCLIFile(t, root, ".pose/specs/extra/spec.md", "---\nslug: extra\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-14\n---\n\n# Spec: extra\n")
	writeCloseoutCLIFile(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: first\n- specs: bundle, extra\n\n## Milestone: second\n- after: first\n- specs: waiting\n")
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "program")
	return root
}

func TestCloseNamesTheNextClose(t *testing.T) {
	root := reachedMilestoneFixture(t)
	code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer")
	if code != 0 {
		t.Fatalf("close %d %s", code, out)
	}
	if !strings.Contains(out, "close.next=pose close milestone:program/first") {
		t.Fatalf("closing the last member did not name the milestone: %s", out)
	}
	if strings.Contains(out, "milestone:program/second") {
		t.Fatalf("closing a member named a milestone it does not finish: %s", out)
	}
	if code, again := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 || !strings.Contains(again, "close.next=pose close milestone:program/first") {
		t.Fatalf("a repeated close lost the milestone: %d %s", code, again)
	}
}

func TestCloseOfAnInheritingMilestonesSpecNamesTheRoadmap(t *testing.T) {
	root := reachedMilestoneFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: first\n- specs: bundle\n")
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "one member")
	code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer")
	if code != 0 {
		t.Fatalf("close %d %s", code, out)
	}
	if !strings.Contains(out, "close.next=pose close roadmap:program") || strings.Contains(out, "close.next=pose close milestone") {
		t.Fatalf("closing the only member of the last milestone did not name the roadmap: %s", out)
	}
}

func TestCloseOfTheLastMilestoneNamesTheRoadmap(t *testing.T) {
	root := reachedMilestoneFixture(t)
	commit := func(message string) {
		artifactGit(t, root, "add", "-A")
		artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", message)
	}
	writeCloseoutCLIFile(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: first\n- specs: bundle\n")
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"api-contract":{"kind":"contract","requiredEvidenceClasses":["integration","unit"]}}}`)
	commit("one milestone")
	if code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 {
		t.Fatalf("close spec %d %s", code, out)
	}
	commit("closeout")
	var sealed posemodel.ReviewBundle
	if code, out := runPoseJSON(t, root, &sealed, "review", "bundle", "milestone:program/first", "--seal", "--json"); code != 0 {
		t.Fatalf("seal %d %s", code, out)
	}
	attest := []string{"review", "attest", sealed.BundleID, "--reviewer", "agent:bundle-review", "--decision", "approved", "--tool", "validate|pose-mcp|passed|validation:module|", "--apply"}
	for _, ev := range sealed.Payload.Evidence {
		attest = append(attest, "--evidence", ev.EvidenceClass+":"+ev.ID)
	}
	if code, out := runPose(t, root, attest...); code != 0 {
		t.Fatalf("attest %d %s", code, out)
	}
	code, out := runPose(t, root, "close", "milestone:program/first")
	if code != 0 {
		t.Fatalf("close milestone %d %s", code, out)
	}
	if !strings.Contains(out, "close.next=pose close roadmap:program") {
		t.Fatalf("closing the last milestone did not name the roadmap: %s", out)
	}
}

func TestStartPrintsItsWarning(t *testing.T) {
	root := reachedMilestoneFixture(t)
	code, out := runPose(t, root, "start", "spec:waiting")
	if code != 0 || !strings.Contains(out, "start.warning=milestone:program/first comes before this spec and is not closed") {
		t.Fatalf("start did not warn: %d %s", code, out)
	}
}

func TestStrictLintFailsOnAnUnacknowledgedChange(t *testing.T) {
	const finding = "R2 changed after its last acknowledged amendment"
	for status, fails := range map[string]bool{"in-progress": true, "draft": false} {
		root, path := amendFixture(t)
		if code, out := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
			t.Fatalf("baseline %d %s", code, out)
		}
		mutated := strings.Replace(amendSpec, "- R2: keep behaving.", "- R2: do something else entirely.", 1)
		lines := strings.SplitAfter(mutated, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "status: ") {
				lines[i] = "status: " + status + "\n"
				break
			}
		}
		mustWrite(t, path, strings.Join(lines, ""))
		var o, e bytes.Buffer
		rc := lintOneSpec(path, false, false, &o, &e)
		if got := strings.Contains(o.String(), finding); got != fails {
			t.Fatalf("%s: finding reported=%v: %s", status, got, o.String())
		}
		if fails && (rc == 0 || !strings.Contains(o.String(), "amendments.failures=1")) {
			t.Fatalf("%s: an unacknowledged change passed: rc=%d %s", status, rc, o.String()+e.String())
		}
	}
}

func TestCloseoutSkillNamesTheMilestoneClose(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, rel := range []string{
		".agents/skills/pose-spec-closeout/SKILL.md",
		"locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md",
		"pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md",
		"pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"close.next", "pose close milestone:<roadmap>/<id>", "pose close roadmap:<roadmap>", "milestone-reached"} {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s does not say %q", rel, want)
			}
		}
	}
}
