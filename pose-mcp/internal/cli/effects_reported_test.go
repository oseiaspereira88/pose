package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestUpdateRefusesAnEmptyGitDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose", "manifest.json"), "{}")
	code, out := runPose(t, root, "update", "--no-self", "--dry-run")
	if code == 0 || !strings.Contains(out, "git repository is required") {
		t.Fatalf("an empty .git passed for a repository: %d %s", code, out)
	}
}

func TestHooksInstallWhereGitRunsThem(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "config", "user.name", "Fixture")
	gitIn(t, repo, "config", "user.email", "fixture@example.invalid")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "base")
	if err := os.MkdirAll(filepath.Join(repo, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(base, "linked")
	gitIn(t, repo, "worktree", "add", "-q", linked)
	code, out := runPose(t, linked, "hooks", "install")
	if code != 0 {
		t.Fatalf("a linked worktree: %d %s", code, out)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("the hook is not where git runs it: %v", err)
	}
	plan, err := buildSetupPlan(linked)
	if err != nil || !plan.Hook {
		t.Fatalf("setup does not find the hook git runs: %+v %v", plan, err)
	}
	if code, out := runPose(t, linked, "init"); code != 0 {
		t.Fatalf("init: %d %s", code, out)
	}
	if _, out := runPose(t, linked, "doctor"); !strings.Contains(out, "pre-commit hook installed") {
		t.Fatalf("doctor does not find the hook git runs: %s", out)
	}
}

func failPolicySync(t *testing.T) {
	t.Helper()
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && strings.HasSuffix(path, "review.json") {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishSyncHook = nil })
}

func TestAnUnsyncedInstallAdoptionSaysItAdopted(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), `{"schema_version":1}`)
	failPolicySync(t)
	var said []string
	adoptGovernedCapabilitiesAtInstallIn(posemodel.NativeFiles(), root, time.Now(), func(en, _ string, v ...any) {
		said = append(said, en)
	})
	all := strings.Join(said, "\n")
	if strings.Contains(all, "not adopted") || !strings.Contains(all, "adopted, with durability unconfirmed") {
		t.Fatalf("the adoption was misreported: %s", all)
	}
	if !strings.Contains(all, "capability (adoption)") {
		t.Fatalf("what was adopted went unsaid: %s", all)
	}
}

func TestAnUnsyncedPolicyOutcomeIsAWarning(t *testing.T) {
	var warned strings.Builder
	durability := &posemodel.PublishDurabilityError{Path: "review.json", Err: errors.New("injected")}
	if err := policyPublished(durability, &warned); err != nil {
		t.Fatalf("a published policy read as a failure: %v", err)
	}
	if !strings.Contains(strings.Join(strings.Fields(warned.String()), " "), "durability is not confirmed") {
		t.Fatalf("the warning went unsaid: %s", warned.String())
	}
	failed := errors.New("refused")
	if err := policyPublished(failed, &warned); !errors.Is(err, failed) {
		t.Fatalf("a failure read as published: %v", err)
	}
}

func TestAnUnsyncedPolicyIsPublishedForEveryCaller(t *testing.T) {
	for _, caller := range []string{"adopt", "configuration-review", "issuer", "init-delegation"} {
		t.Run(caller, func(t *testing.T) {
			if caller == "init-delegation" {
				delegationInitInput(t, "yes\n", true, true)
			}
			root, _ := installedInstance(t)
			if caller == "issuer" {
				t.Setenv("POSE_ISSUER_HOME", filepath.Join(t.TempDir(), "issuers"))
				if code, out := runPose(t, root, "issuer", "init", "fixture"); code != 0 {
					t.Fatalf("issuer fixture: %d %s", code, out)
				}
			}
			failPolicySync(t)
			var out, errOut strings.Builder
			var code int
			switch caller {
			case "adopt":
				code = cmdAdopt(root, []string{"overlay:high-criticality-review", "--apply"}, &out, &errOut)
			case "configuration-review":
				req := reviewRequest{Capability: "overlay:high-criticality-review"}
				req.View.Answer = reviewAnswerAdopt
				message, err := applyReviewAnswer(root, req, "2026-11-02", "", "")
				if err != nil {
					t.Fatal(err)
				}
				out.WriteString(message)
			case "issuer":
				code = cmdIssuer(root, []string{"pin", "fixture", "--attestations", "--apply"}, &out, &errOut)
			case "init-delegation":
				code = initDelegation(root, &out, &errOut)
			}
			if code != 0 || !strings.Contains(strings.Join(strings.Fields(out.String()+errOut.String()), " "), "durability is not confirmed") {
				t.Fatalf("published policy misreported: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			docs, err := posemodel.LoadPolicyDocs(root)
			if err != nil {
				t.Fatal(err)
			}
			if caller == "adopt" || caller == "configuration-review" {
				entry, _ := posemodel.LookupCatalogEntry("overlay:high-criticality-review")
				if !entry.Adopted(docs) {
					t.Fatal("success did not publish the adoption")
				}
			}
		})
	}
}

func TestAPartialRefreshReportsItsPublicationDurability(t *testing.T) {
	for _, unsynced := range []bool{false, true} {
		t.Run(map[bool]string{false: "durable", true: "unsynced"}[unsynced], func(t *testing.T) {
			root := unsyncedRefreshFixture(t, "")
			if !unsynced {
				posemodel.PublishSyncHook = nil
			}
			if err := os.MkdirAll(filepath.Join(root, ".pose/state/history.jsonl"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "partial"}); err != nil {
				t.Fatal(err)
			}
			last, ok := lastStateRefresh(root)
			if !ok || last.Result != "partial" || last.DurabilityUnconfirmed != unsynced {
				t.Fatalf("partial not recorded: %+v", last)
			}
			if _, open := unresolvedDurability(root); open != unsynced {
				t.Fatalf("durability open=%v, want %v", open, unsynced)
			}
		})
	}
}

func TestAReplayKeepsAnUnresolvedDurabilityWarning(t *testing.T) {
	event := HookEvent{Kind: "spec_closeout", Target: "x"}
	root := unsyncedRefreshFixture(t, "")
	for i := 0; i < 2; i++ {
		if err := stateRefreshConsumer(root, event); err != nil {
			t.Fatal(err)
		}
	}
	if last, _ := lastStateRefresh(root); last.Result != "skipped" {
		t.Fatalf("the fixture did not replay: %+v", last)
	}
	if open, ok := unresolvedDurability(root); !ok || open.Result != "unsynced" {
		t.Fatalf("the replay hid the warning: %+v", open)
	}
	posemodel.PublishSyncHook = nil
	var out, errOut strings.Builder
	if code := cmdStateRefresh(root, nil, &out, &errOut); code != 0 {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	if open, ok := unresolvedDurability(root); ok {
		t.Fatalf("a durable publication did not answer it: %+v", open)
	}
}

func TestALostLogReadsAsAnUnfinishedEffect(t *testing.T) {
	kind, unfinished, detail := refreshPendingMeaning("spec_closeout published, refresh log not appended, durability unconfirmed")
	if !unfinished || kind != "spec_closeout" || detail != "refresh log not appended, durability unconfirmed" {
		t.Fatalf("%q %v %q", kind, unfinished, detail)
	}
	if _, unfinished, _ := refreshPendingMeaning("spec_closeout"); unfinished {
		t.Fatal("a failed refresh read as an unfinished one")
	}
	root := unsyncedRefreshFixture(t, "")
	if err := os.MkdirAll(stateRefreshLogPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "x"}); err != nil {
		t.Fatal(err)
	}
	posemodel.PublishSyncHook = nil
	_, out := runPose(t, root, "state")
	if !strings.Contains(out, "refresh-effect-unfinished") || strings.Contains(out, "refresh failed") {
		t.Fatalf("the marker read as a failed refresh: %s", out)
	}
}

func TestDoctorReadsALostLogAsAnUnfinishedEffect(t *testing.T) {
	root := stateTestRoot(t)
	writeStateTestFile(t, root, ".pose/state/project-state.md", "---\nschema_version: 1\ngenerated_at: 2026-11-02T00:00:00Z\nbaseline_commit: "+strings.TrimSpace(runStateGit(t, root, "rev-parse", "HEAD"))+"\nrefresh_pending: spec_closeout published, refresh log not appended\n---\n")
	f, ok := findDoctorFinding(doctorStateFindings(t, root), "state.freshness")
	if !ok || f.Level != "warn" || !strings.Contains(f.Message, "left an effect to finish") {
		t.Fatalf("doctor read: %+v", f)
	}
}

func TestAFailedRefreshKeepsAnUnresolvedDurabilityWarning(t *testing.T) {
	root := unsyncedRefreshFixture(t, "")
	if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "x"}); err != nil {
		t.Fatal(err)
	}
	// File sync fails before publication; directory sync failed after it.
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "file" && strings.HasSuffix(path, "project-state.md") {
			return errors.New("injected before publication")
		}
		return nil
	}
	var out, errOut strings.Builder
	if code := cmdStateRefresh(root, nil, &out, &errOut); code == 0 {
		t.Fatalf("the fixture did not fail: %s %s", &out, &errOut)
	}
	if last, _ := lastStateRefresh(root); last.Result != "failed" {
		t.Fatalf("the fixture did not record a failed refresh: %+v", last)
	}
	if _, ok := unresolvedDurability(root); !ok {
		t.Fatal("a failed refresh hid the unconfirmed publication")
	}
	_, rendered := runPose(t, root, "state")
	if !strings.Contains(rendered, "durability-unconfirmed") {
		t.Fatalf("state validate hid the unconfirmed publication: %s", rendered)
	}
}
