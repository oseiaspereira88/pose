package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func writeAgentProject(t *testing.T, root, slug, status string) {
	t.Helper()
	for _, dir := range []string{".pose/templates", ".pose/policy", ".pose/specs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".pose/templates/spec.md"), []byte("---\nslug: <feature-slug>\nstatus: draft\ncreated_at: <created_at>\n---\n# Spec: <feature-slug>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1}` + "\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	if slug != "" {
		body := "---\nslug: " + slug + "\nstatus: " + status + "\ncreated_at: 2026-09-24\n---\n# Spec: " + slug + "\n"
		if err := os.WriteFile(filepath.Join(root, ".pose/specs/2026-09-24-"+slug+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commitAgentFixture(t, root, "initial fixture")
}

func commitAgentFixture(t *testing.T, root, message string) {
	t.Helper()
	for _, args := range [][]string{{"config", "user.email", "pose-agent-test@example.invalid"}, {"config", "user.name", "POSE agent fixture"}, {"add", "-A"}, {"commit", "-qm", message}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
}

func agentRootsJSON(t *testing.T, roots map[string]string) string {
	t.Helper()
	raw, err := json.Marshal(roots)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestMultiRepoAgentContextResolvesOneQualifiedTaskFromEveryCheckout(t *testing.T) {
	parent := newGitRepo(t)
	executor := newGitRepo(t)
	sibling := newGitRepo(t)
	writeAgentProject(t, parent, "checkout", "draft")
	writeAgentProject(t, executor, "checkout", "in-progress")
	writeAgentProject(t, sibling, "checkout", "draft")
	roots := agentRootsJSON(t, map[string]string{"proj.parent": parent, "proj.executor": executor, "proj.sibling": sibling})
	t.Setenv("POSE_PROJECT_ROOTS", roots)
	t.Setenv("HARNE8_PROJECTS_DIR", "")

	taskRef := "xref:proj.executor/spec:checkout"
	var authority string
	var taskRevision string
	for _, workspace := range []struct{ id, root string }{{"proj.parent", parent}, {"proj.executor", executor}, {"proj.sibling", sibling}} {
		t.Setenv("POSE_DEFAULT_PROJECT_ID", workspace.id)
		var stdout, stderr bytes.Buffer
		if code := cmdProjectContext(workspace.root, []string{"--task", taskRef, "--json"}, &stdout, &stderr); code != 0 {
			t.Fatalf("context from %s failed: code=%d stderr=%s", workspace.id, code, stderr.String())
		}
		var got posemodel.AgentProjectContext
		if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
			t.Fatalf("decode context from %s: %v", workspace.id, err)
		}
		if got.SelectedProjectID != workspace.id || got.Authority == nil || got.Authority.String() != taskRef {
			t.Fatalf("wrong authority from %s: %+v", workspace.id, got)
		}
		if got.TaskResolution == nil || !got.TaskResolution.Resolved || got.TaskResolution.Status != "in-progress" {
			t.Fatalf("task did not resolve from %s: %+v", workspace.id, got.TaskResolution)
		}
		if got.ContextRevision == "" || got.AuthorityRevision == "" {
			t.Fatalf("context from %s is not revision-bound: %+v", workspace.id, got)
		}
		if authority == "" {
			authority, taskRevision = got.Authority.String(), got.AuthorityRevision
		} else if authority != got.Authority.String() || taskRevision != got.AuthorityRevision {
			t.Fatalf("workspace changed canonical task: authority=%s revision=%s context=%+v", authority, taskRevision, got)
		}
		raw, _ := json.Marshal(got)
		for _, path := range []string{parent, executor, sibling} {
			if strings.Contains(string(raw), path) {
				t.Fatalf("project context leaked root %q: %s", path, raw)
			}
		}
	}
}

func TestMultiRepoAgentRoutingCreatesAtQualifiedAuthorityAndReusesIt(t *testing.T) {
	parent := newGitRepo(t)
	projectsDir := t.TempDir()
	executor := filepath.Join(projectsDir, "proj.executor")
	if err := os.MkdirAll(executor, 0o755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", executor, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init executor: %v: %s", err, output)
	}
	sibling := newGitRepo(t)
	writeAgentProject(t, parent, "", "")
	writeAgentProject(t, executor, "", "")
	writeAgentProject(t, sibling, "remote-task", "draft")
	taskRef := "xref:proj.executor/spec:remote-task"
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.parent")
	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.executor": executor, "proj.sibling": sibling}))
	t.Setenv("HARNE8_PROJECTS_DIR", "")

	context, _, _, err := cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	stale := context.ContextRevision
	if err := os.WriteFile(filepath.Join(executor, "REVISION.md"), []byte("changed after discovery\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAgentFixture(t, executor, "advance executor revision")
	var stdout, stderr bytes.Buffer
	if code := cmdNewSpec(parent, []string{"remote-task", "--task", taskRef, "--expect-context", stale}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "stale-context") {
		t.Fatalf("stale context must block the write: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	today := time.Now().UTC().Format("2006-01-02")
	target := filepath.Join(executor, ".pose", "specs", today+"-remote-task.md")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("stale attempt wrote target spec: stat err=%v", err)
	}

	context, _, _, err = cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	// Discovery of a scanned sibling root does not grant write permission.
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("HARNE8_PROJECTS_DIR", projectsDir)
	stdout.Reset()
	stderr.Reset()
	if code := cmdNewSpec(parent, []string{"remote-task", "--task", taskRef, "--expect-context", context.ContextRevision}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "cross-project-write-unauthorized") {
		t.Fatalf("scanned root must not authorize cross-project write: code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("unauthorized attempt wrote target spec: stat err=%v", err)
	}

	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.executor": executor, "proj.sibling": sibling}))
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	context, _, _, err = cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdNewSpec(parent, []string{"remote-task", "--task", taskRef, "--expect-context", context.ContextRevision}, &stdout, &stderr); code != 0 {
		t.Fatalf("authorized qualified create failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("qualified target was not created in authority project: %v", err)
	}
	localShadow := filepath.Join(parent, ".pose", "specs", today+"-remote-task.md")
	if _, err := os.Stat(localShadow); !os.IsNotExist(err) {
		t.Fatalf("qualified task created a local shadow: stat err=%v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdNewSpec(parent, []string{"remote-task", "--task", taskRef}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Canonical spec") {
		t.Fatalf("existing qualified task was not reused: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestMultiRepoAgentNegativeContextRejectsMalformedAndUnregisteredAuthority(t *testing.T) {
	parent := newGitRepo(t)
	writeAgentProject(t, parent, "", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.parent")
	t.Setenv("POSE_PROJECT_ROOTS", "{}")
	t.Setenv("HARNE8_PROJECTS_DIR", "")

	var stdout, stderr bytes.Buffer
	if code := cmdProjectContext(parent, []string{"--task", "../ambiguous"}, &stdout, &stderr); code != 1 {
		t.Fatalf("malformed task ref should fail: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdNewSpec(parent, []string{"remote-task", "--task", "xref:proj.ghost/spec:remote-task", "--expect-context", strings.Repeat("0", 64)}, &stdout, &stderr); code != 1 {
		t.Fatalf("unregistered authority should fail closed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(parent, ".pose", "specs", time.Now().UTC().Format("2006-01-02")+"-remote-task.md")); !os.IsNotExist(err) {
		t.Fatalf("failed external lookup created a local shadow: stat err=%v", err)
	}
}

func TestMultiRepoAgentNegativeBindingChangeInvalidatesContext(t *testing.T) {
	parent, executor := newGitRepo(t), newGitRepo(t)
	writeAgentProject(t, parent, "", "")
	writeAgentProject(t, executor, "checkout", "in-progress")
	clone := filepath.Join(t.TempDir(), "proj.executor")
	if output, err := exec.Command("git", "clone", "-q", executor, clone).CombinedOutput(); err != nil {
		t.Fatalf("clone executor binding: %v: %s", err, output)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.parent")
	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.executor": executor}))
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	taskRef := "xref:proj.executor/spec:checkout"
	first, _, _, err := cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.executor": clone}))
	second, _, _, err := cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	if first.ContextRevision == second.ContextRevision || first.AuthorityRevision != second.AuthorityRevision || first.TaskResolution.Digest != second.TaskResolution.Digest {
		t.Fatalf("context token did not bind project-id to its configured checkout: first=%+v second=%+v", first, second)
	}
	raw, _ := json.Marshal(second)
	if strings.Contains(string(raw), executor) || strings.Contains(string(raw), clone) {
		t.Fatalf("binding identity leaked a filesystem path: %s", raw)
	}
}

func TestMultiRepoAgentNegativePolicyChangeInvalidatesContext(t *testing.T) {
	parent := newGitRepo(t)
	writeAgentProject(t, parent, "", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.parent")
	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.parent": parent}))
	t.Setenv("HARNE8_PROJECTS_DIR", "")

	first, _, _, err := cliAgentContext(parent, "")
	if err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(parent, ".pose", "policy", "review.json")
	updatedPolicy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"contract_adoptions":{"review-bundles":"2026-09-24"}}` + "\n"
	if err := os.WriteFile(policyPath, []byte(updatedPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	second, _, _, err := cliAgentContext(parent, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.SelectedRevision != second.SelectedRevision || first.ContextRevision == second.ContextRevision {
		t.Fatalf("context token did not bind an uncommitted policy change: first=%+v second=%+v", first, second)
	}
}

func TestMultiRepoAgentNegativeTransferBlocksUntilCanonicalAuthorityIsActive(t *testing.T) {
	sourceRoot, destinationRoot := t.TempDir(), t.TempDir()
	initCLITransferProject(t, sourceRoot, "proj.source", "source-work")
	initCLITransferProject(t, destinationRoot, "proj.destination", "destination-work")
	roots := posemodel.NewRoots(posemodel.RootsConfig{
		DefaultRoot:      sourceRoot,
		DefaultProjectID: "proj.source",
		Explicit:         map[string]string{"proj.destination": destinationRoot},
	})
	resolver := posemodel.ArtifactResolver{Roots: roots}
	request := posemodel.SpecTransferRequest{
		Source:      posemodel.ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "source-work"},
		Destination: posemodel.ArtifactRef{Project: "proj.destination", Kind: "spec", Slug: "destination-work"},
		Mappings: []posemodel.SpecTransferRequirementMapping{{
			SourceRequirement: "R1", DestinationRequirement: "R1", Disposition: "equivalent",
		}},
	}
	plan, err := posemodel.PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true}
	if _, err := posemodel.ApplySpecTransfer(resolver, plan, plan.Digest, permissions, func(phase string) error {
		if phase == "prepared" {
			return errors.New("simulated interruption")
		}
		return nil
	}); err == nil {
		t.Fatal("transfer fixture should remain interrupted")
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.source")
	rootsJSON := agentRootsJSON(t, map[string]string{"proj.destination": destinationRoot})
	t.Setenv("POSE_PROJECT_ROOTS", rootsJSON)
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	taskRef := "xref:proj.source/spec:source-work"
	context, _, _, err := cliAgentContext(sourceRoot, taskRef)
	if err != nil || context.TaskResolution == nil || context.TaskResolution.State != "transfer-in-progress" {
		t.Fatalf("context did not stop during interrupted authority transfer: context=%+v err=%v", context, err)
	}
	var stdout, stderr bytes.Buffer
	if code := cmdNewSpec(sourceRoot, []string{"source-work", "--task", taskRef}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "transfer-in-progress") {
		t.Fatalf("new-spec ignored transfer barrier: code=%d stderr=%s", code, stderr.String())
	}
	status, err := posemodel.ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil)
	if err != nil || status.Phase != "activated" {
		t.Fatalf("resume did not activate one authority: status=%+v err=%v", status, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdNewSpec(sourceRoot, []string{"source-work"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "xref:proj.destination/spec:destination-work") {
		t.Fatalf("redirect did not reuse the canonical spec: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestMultiRepoAgentRoutingClosesOnlyQualifiedAuthorityWithFreshContext(t *testing.T) {
	parent, executor := newGitRepo(t), newGitRepo(t)
	writeAgentProject(t, parent, "checkout", "draft")
	writeAgentProject(t, executor, "checkout", "in-progress")
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":true,"adopted_at":"2026-09-24","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	if err := os.WriteFile(filepath.Join(executor, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}` + "\n"
	if err := os.MkdirAll(filepath.Join(executor, ".pose/review-profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(executor, ".pose/review-profiles/spec-closeout.json"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAgentFixture(t, executor, "configure review for closeout")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.parent")
	t.Setenv("POSE_PROJECT_ROOTS", agentRootsJSON(t, map[string]string{"proj.executor": executor}))
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	taskRef := "xref:proj.executor/spec:checkout"
	context, _, _, err := cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(executor, "ADVANCE.md"), []byte("new revision\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAgentFixture(t, executor, "advance before close")
	var stdout, stderr bytes.Buffer
	if code := cmdClose(parent, []string{taskRef, "--expect-context", context.ContextRevision}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "stale-context") {
		t.Fatalf("stale close context should fail: code=%d stderr=%s", code, stderr.String())
	}
	parentSpec, err := (posemodel.Store{Root: parent}).GetSpec("checkout")
	if err != nil || parentSpec.Status != "draft" {
		t.Fatalf("parent same-slug spec changed: spec=%+v err=%v", parentSpec, err)
	}
	executorSpec, err := (posemodel.Store{Root: executor}).GetSpec("checkout")
	if err != nil || executorSpec.Status != "in-progress" {
		t.Fatalf("stale close mutated executor: spec=%+v err=%v", executorSpec, err)
	}
	context, _, _, err = cliAgentContext(parent, taskRef)
	if err != nil {
		t.Fatal(err)
	}
	reviewArgs := []string{"record", taskRef, "--reviewer", "agent:executor-review", "--decision", "approved", "--evidence", "check:unit", "--apply", "--expect-context", context.ContextRevision}
	if code := cmdReview(parent, reviewArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("authority review setup failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdReviewCheck(parent, []string{taskRef, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("qualified review check failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var reviewState posemodel.ReviewEvaluation
	if err := json.Unmarshal(stdout.Bytes(), &reviewState); err != nil || !reviewState.Approved {
		t.Fatalf("review check did not read the authority project: state=%+v err=%v raw=%s", reviewState, err, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdCloseoutCheck(parent, []string{taskRef, "--json"}, &stdout, &stderr); code != 1 {
		t.Fatalf("open authority must remain pending before close: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var closeoutState posemodel.CloseoutState
	if err := json.Unmarshal(stdout.Bytes(), &closeoutState); err != nil || !closeoutState.Review.Approved || closeoutState.LifecycleDone {
		t.Fatalf("closeout check read a shadow spec: state=%+v err=%v raw=%s", closeoutState, err, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cmdClose(parent, []string{taskRef, "--expect-context", context.ContextRevision}, &stdout, &stderr); code != 0 {
		t.Fatalf("qualified close failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	executorSpec, err = (posemodel.Store{Root: executor}).GetSpec("checkout")
	if err != nil || executorSpec.Status != "done" {
		t.Fatalf("qualified close did not update authority: spec=%+v err=%v", executorSpec, err)
	}
	parentSpec, err = (posemodel.Store{Root: parent}).GetSpec("checkout")
	if err != nil || parentSpec.Status != "draft" {
		t.Fatalf("qualified close changed local same-slug spec: spec=%+v err=%v", parentSpec, err)
	}
	coordinatorState, err := (posemodel.Store{Root: parent}).GetCloseoutState("spec:checkout")
	if err != nil || coordinatorState.LifecycleDone || coordinatorState.Terminal {
		t.Fatalf("executor close incorrectly completed the coordinator: state=%+v err=%v", coordinatorState, err)
	}
}

func TestMultiRepoAgentSurfaceExposesPathFreeCLIContext(t *testing.T) {
	root, err := projectRootAt(".")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.engine")
	t.Setenv("POSE_PROJECT_ROOTS", "{}")
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	var stdout, stderr bytes.Buffer
	if code := cmdProjectContext(root, []string{"--task", "spec:pose-agent-project-context", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("CLI context command failed: code=%d stderr=%s", code, stderr.String())
	}
	var got posemodel.AgentProjectContext
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("CLI context is not JSON: %v: %s", err, stdout.String())
	}
	if got.SelectedProjectID != "proj.engine" || got.Authority == nil || got.Authority.String() != "xref:proj.engine/spec:pose-agent-project-context" || got.ContextRevision == "" {
		t.Fatalf("CLI context omitted selected authority or freshness token: %+v", got)
	}
	if strings.Contains(stdout.String(), root) {
		t.Fatalf("CLI context leaked a filesystem root: %s", stdout.String())
	}
}

func TestMultiRepoAgentInstalledJourneyUsesInstalledCLIAndRejectsStaleBinding(t *testing.T) {
	parent := newGitRepo(t)
	executor := newGitRepo(t)
	writeAgentProject(t, parent, "", "")
	writeAgentProject(t, executor, "", "")
	projects := agentRootsJSON(t, map[string]string{"proj.executor": executor})
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../.."))
	binary := filepath.Join(t.TempDir(), "pose")
	build := exec.Command("go", "build", "-o", binary, "./cmd/pose")
	build.Dir = moduleRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build installed CLI: %v: %s", err, output)
	}
	baseEnv := append(os.Environ(), "POSE_DEFAULT_PROJECT_ID=proj.parent", "POSE_PROJECT_ROOTS="+projects, "HARNE8_PROJECTS_DIR=")
	taskRef := "xref:proj.executor/spec:installed-task"
	run := func(root string, args ...string) ([]byte, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir, cmd.Env = root, baseEnv
		return cmd.CombinedOutput()
	}
	for _, args := range [][]string{
		{"new-spec-qualified", "installed-task", "--task", taskRef},
		{"new-spec-qualified", "installed-task", "--task", "spec:installed-task", "--expect-context", "digest"},
	} {
		output, err := run(parent, args...)
		if err == nil || !strings.Contains(string(output), "new-spec-qualified requires") {
			t.Fatalf("qualified verb accepted missing context or local task: args=%v err=%v output=%s", args, err, output)
		}
	}
	contextRaw, err := run(parent, "context", "--task", taskRef, "--json")
	if err != nil {
		t.Fatalf("installed context: %v: %s", err, contextRaw)
	}
	var context posemodel.AgentProjectContext
	if err := json.Unmarshal(contextRaw, &context); err != nil || context.Authority == nil || context.Authority.String() != taskRef {
		t.Fatalf("installed context mismatch: context=%+v err=%v raw=%s", context, err, contextRaw)
	}
	if err := os.WriteFile(filepath.Join(executor, "ADVANCE.md"), []byte("advance\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAgentFixture(t, executor, "advance executor binding")
	staleOutput, err := run(parent, "new-spec-qualified", "installed-task", "--task", taskRef, "--expect-context", context.ContextRevision)
	if err == nil || !strings.Contains(string(staleOutput), "stale-context") {
		t.Fatalf("installed CLI applied stale context: err=%v output=%s", err, staleOutput)
	}
	freshRaw, err := run(parent, "context", "--task", taskRef, "--json")
	if err != nil {
		t.Fatalf("refresh installed context: %v: %s", err, freshRaw)
	}
	if err := json.Unmarshal(freshRaw, &context); err != nil {
		t.Fatal(err)
	}
	created, err := run(parent, "new-spec-qualified", "installed-task", "--task", taskRef, "--expect-context", context.ContextRevision)
	if err != nil {
		t.Fatalf("installed cross-project create: %v: %s", err, created)
	}
	if _, err := os.Stat(filepath.Join(executor, ".pose", "specs", time.Now().UTC().Format("2006-01-02")+"-installed-task.md")); err != nil {
		t.Fatalf("installed CLI did not write the canonical project: %v", err)
	}
}
