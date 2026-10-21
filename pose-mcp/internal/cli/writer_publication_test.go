package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-writer-coordination-and-atomic-publication: the CLI writers
// coordinate on the spec's lifecycle lock and publish only over what their
// gates judged.

// V4 — an edit of the spec made while the close gates ran is a conflict:
// the transition is not published over it, and the edit stays.
func TestACloseNeverPublishesOverAnEditMadeWhileItsGatesRan(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	spec := filepath.Join(root, ".pose", "specs", "bundle", "spec.md")
	edited := false
	posemodel.PublishFault = func(path string) error {
		if path == spec && !edited {
			edited = true
			raw, _ := os.ReadFile(spec)
			return os.WriteFile(spec, append(raw, []byte("\nA concurrent edit.\n")...), 0o644)
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishFault = nil })
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer", "--json")
	posemodel.PublishFault = nil
	raw, _ := os.ReadFile(spec)
	if !edited || code != 1 || result.Step != posemodel.CloseoutStepTransition || !strings.Contains(result.Error.Message, "publish-conflict") ||
		!strings.Contains(string(raw), "A concurrent edit.") || !strings.Contains(string(raw), "status: in-progress") {
		t.Fatalf("the close published over a concurrent edit: edited=%v code=%d %+v\n%s", edited, code, result.OperationResult, raw)
	}
	commitSnapshotFixture(t, root)
	current, err := (posemodel.Store{Root: root}).PlanCloseout("spec:bundle")
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--digest", current.Digest, "--reviewer", "agent:closeout-reviewer", "--json"); code != 0 {
		t.Fatalf("the close of the current bytes: %d %+v", code, result.OperationResult)
	}
}

// V3 — a close does not interleave with a start, a cancel or a transfer of
// the same spec: it is refused with the lock's owner while one holds it.
func TestACloseWaitsForNoOtherLifecycleWriter(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	unlock, err := posemodel.LockSpecLifecycle(root, "bundle", "transfer.apply")
	if err != nil {
		t.Fatal(err)
	}
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer", "--json")
	unlock()
	raw, _ := os.ReadFile(filepath.Join(root, ".pose", "specs", "bundle", "spec.md"))
	if code != 1 || result.Step != posemodel.CloseoutStepTransition || !strings.Contains(result.Error.Message, "spec-lifecycle-in-progress") || !strings.Contains(string(raw), "status: in-progress") {
		t.Fatalf("a close interleaved with another lifecycle writer: %d %+v", code, result.OperationResult)
	}
	if code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer", "--json"); code != 0 || result.Outcome != posemodel.OutcomeCompleted {
		t.Fatalf("the close after the lock was released: %d %+v", code, result.OperationResult)
	}
}

func stateFixture(t *testing.T) string {
	t.Helper()
	root := startCLIFixture(t, true)
	if code, out := runPose(t, root, "state", "init"); code != 0 {
		t.Fatalf("state init: %d %s", code, out)
	}
	return root
}

func refreshLog(t *testing.T, root string) []refreshLogEntry {
	t.Helper()
	entries, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

// V5 — a refresh that publishes the state and cannot append its history
// says so: the state is published, the log records a partial refresh, and
// the next refresh appends its own entry without making one up.
func TestARefreshThatPublishedTheStateSaysSoWhenItsHistoryFails(t *testing.T) {
	root := stateFixture(t)
	store := posemodel.Store{Root: root}
	history := store.StateHistoryPath()
	before, _ := os.ReadFile(history)
	if err := os.Remove(history); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(history, 0o755); err != nil {
		t.Fatal(err)
	}
	// A derived section changes, so the publication is observable.
	specPath := filepath.Join(root, ".pose", "specs", "2026-09-27-work.md")
	spec, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, bytes.Replace(spec, []byte("status: draft"), []byte("status: in-progress"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	stateBefore, _ := os.ReadFile(store.StatePath())
	code, out := runPose(t, root, "state", "refresh")
	stateAfter, _ := os.ReadFile(store.StatePath())
	log := refreshLog(t, root)
	if code != 1 || !strings.Contains(out, "state.published=") || !strings.Contains(out, "its history entry was not appended") ||
		bytes.Equal(stateBefore, stateAfter) || log[len(log)-1].Result != "partial" {
		t.Fatalf("a published state was reported as nothing done: %d %s %+v", code, out, log[len(log)-1])
	}
	if err := os.Remove(history); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "state", "refresh"); code != 0 {
		t.Fatalf("the retry: %d %s", code, out)
	}
	entries, _ := readStateHistory(history)
	if lines := strings.Count(string(before), "\n"); len(entries) != lines+1 {
		t.Fatalf("the retry appended %d entries over %d", len(entries), lines)
	}
}

// V5 — an edit of a curated section made while a refresh renders is never
// overwritten: the refresh conflicts and the edit stays.
func TestARefreshNeverOverwritesACuratedEditMadeWhileItRendered(t *testing.T) {
	root := stateFixture(t)
	store := posemodel.Store{Root: root}
	edit := "Curated by a person while the refresh ran."
	posemodel.PublishFault = func(path string) error {
		if path != store.StatePath() {
			return nil
		}
		raw, _ := os.ReadFile(path)
		return os.WriteFile(path, []byte(strings.Replace(string(raw), "<!-- state:curated -->\n\n", "<!-- state:curated -->\n\n"+edit+"\n", 1)), 0o644)
	}
	t.Cleanup(func() { posemodel.PublishFault = nil })
	code, out := runPose(t, root, "state", "refresh")
	posemodel.PublishFault = nil
	raw, _ := os.ReadFile(store.StatePath())
	if code != 1 || !strings.Contains(out, "publish-conflict") || !strings.Contains(string(raw), edit) {
		t.Fatalf("the refresh overwrote a curated edit: %d %s", code, out)
	}
	if code, _ := runPose(t, root, "state", "refresh"); code != 0 {
		t.Fatal("the refresh of the current state")
	}
	if raw, _ := os.ReadFile(store.StatePath()); !strings.Contains(string(raw), edit) {
		t.Fatal("the next refresh lost the curated edit")
	}
}

// V6/R7 — an append cut short never glues the next history entry to its
// bytes: they stay as they were, and the new entry reads on its own line.
func TestAHistoryEntryIsNeverGluedToATornTail(t *testing.T) {
	root := stateFixture(t)
	history := posemodel.Store{Root: root}.StateHistoryPath()
	torn := []byte(`{"generated_at":"2026-10-16T00:00:00Z","baseline_`)
	f, err := os.OpenFile(history, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(torn)
	f.Close()
	before, _ := readStateHistory(history)
	if code, out := runPose(t, root, "state", "refresh"); code != 0 {
		t.Fatalf("refresh: %d %s", code, out)
	}
	raw, _ := os.ReadFile(history)
	after, _ := readStateHistory(history)
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	var last stateHistoryEntry
	if len(after) != len(before)+1 || !bytes.Contains(raw, append(append([]byte{}, torn...), '\n')) || json.Unmarshal([]byte(lines[len(lines)-1]), &last) != nil {
		t.Fatalf("the new entry was glued to the torn tail:\n%s", raw)
	}
}

// V4 — an edit made after the gates judged the spec and before the
// transition is read is a conflict too: the transition is bound to the
// bytes the gates judged, not to whatever is there when it writes.
func TestACloseIsBoundToTheBytesItsGatesJudged(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	spec := filepath.Join(root, ".pose", "specs", "bundle", "spec.md")
	// Bring the scope to an approved review with the transition not yet
	// published.
	posemodel.PublishFault = func(path string) error {
		if path == spec {
			return os.ErrPermission
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishFault = nil })
	if code, _ := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 1 {
		t.Fatal("the fixture did not stop at the transition")
	}
	posemodel.PublishFault = nil
	edit := func() error {
		raw, _ := os.ReadFile(spec)
		return os.WriteFile(spec, append(raw, []byte("\nEdited after the gates.\n")...), 0o644)
	}
	var out, errOut bytes.Buffer
	code := 0
	inDir(t, root, func() { code = cmdCloseLocal(root, "spec:bundle", edit, &out, &errOut) })
	raw, _ := os.ReadFile(spec)
	if code != 1 || !strings.Contains(errOut.String(), "publish-conflict") || !strings.Contains(string(raw), "status: in-progress") || !strings.Contains(string(raw), "Edited after the gates.") {
		t.Fatalf("the transition was published over bytes the gates did not judge: %d %s\n%s", code, errOut.String(), raw)
	}
}

// V4/R4 — the authorization is judged again inside the protected region: a
// review that stopped being approved after the gates ran does not govern the
// transition.
func TestACloseRevalidatesItsAuthorizationInsideTheLock(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	spec := filepath.Join(root, ".pose", "specs", "bundle", "spec.md")
	posemodel.PublishFault = func(path string) error {
		if path == spec {
			return os.ErrPermission
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishFault = nil })
	if code, _ := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 1 {
		t.Fatal("the fixture did not stop at the transition")
	}
	posemodel.PublishFault = nil
	// The delivered artifact changes after the gates judged it: the sealed
	// bundle no longer represents the current semantic inputs.
	closeGatesJudged = func() {
		_ = os.WriteFile(filepath.Join(root, "pose-mcp", "main.go"), []byte("package main\n\nfunc changed() {}\n"), 0o644)
	}
	t.Cleanup(func() { closeGatesJudged = nil })
	var out, errOut bytes.Buffer
	code := 0
	inDir(t, root, func() { code = cmdCloseLocal(root, "spec:bundle", nil, &out, &errOut) })
	raw, _ := os.ReadFile(spec)
	if code != 1 || !strings.Contains(errOut.String(), "authorization changed while its gates ran") || !strings.Contains(string(raw), "status: in-progress") {
		t.Fatalf("a stale authorization governed the transition: %d %s\n%s", code, errOut.String(), raw)
	}
}

// F1/R6/R7 — a hook-triggered refresh that published the state and failed
// its history neither claims a pending refresh nor hides the partial
// effect: `pose state` reports it, and the next run of the event
// appends the entry.
func TestAHookRefreshThatPublishedTheStateDoesNotClaimItIsPending(t *testing.T) {
	root := stateFixture(t)
	store := posemodel.Store{Root: root}
	history := store.StateHistoryPath()
	before, _ := os.ReadFile(history)
	if err := os.Remove(history); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(history, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stateRefreshConsumer(root, HookEvent{Kind: "lint_spec_strict", Target: "spec:work"}); err != nil {
		t.Fatalf("a best-effort hook failed its caller: %v", err)
	}
	raw, _ := os.ReadFile(store.StatePath())
	if strings.Contains(string(raw), "refresh_pending: lint") {
		t.Fatalf("a current state claims a pending refresh:\n%s", raw)
	}
	if code, out := runPose(t, root, "state"); !strings.Contains(out, "history-not-appended") {
		t.Fatalf("`pose state` hid the partial refresh: %d %s", code, out)
	}
	if err := os.Remove(history); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stateRefreshConsumer(root, HookEvent{Kind: "lint_spec_strict", Target: "spec:work"}); err != nil {
		t.Fatal(err)
	}
	if entries, _ := readStateHistory(history); len(entries) != strings.Count(string(before), "\n")+1 {
		t.Fatalf("the retry of the event did not append its entry: %d", len(entries))
	}
}

// F2/R2 — the other writers of the state publish over what they read.
func TestEveryStateWriterKeepsACuratedEdit(t *testing.T) {
	root := stateFixture(t)
	store := posemodel.Store{Root: root}
	edit := "Curated while a writer ran."
	curate := func(path string) error {
		if path != store.StatePath() {
			return nil
		}
		raw, _ := os.ReadFile(path)
		return os.WriteFile(path, []byte(strings.Replace(string(raw), "<!-- state:curated -->\n\n", "<!-- state:curated -->\n\n"+edit+"\n", 1)), 0o644)
	}
	posemodel.PublishFault = curate
	t.Cleanup(func() { posemodel.PublishFault = nil })
	if err := markRefreshPending(root, "lint_spec_strict"); !errors.Is(err, posemodel.ErrPublishConflict) {
		t.Fatalf("markRefreshPending overwrote a curated edit: %v", err)
	}
	if raw, _ := os.ReadFile(store.StatePath()); !strings.Contains(string(raw), edit) {
		t.Fatal("the curated edit was lost")
	}
	posemodel.PublishFault = nil
	if code, _ := runPose(t, root, "state", "refresh"); code != 0 {
		t.Fatal("refresh")
	}
	posemodel.PublishFault = func(path string) error {
		if path != store.StatePath() {
			return nil
		}
		raw, _ := os.ReadFile(path)
		return os.WriteFile(path, append(raw, []byte("\n"+edit+" (again)\n")...), 0o644)
	}
	if err := updateProjectStateArchitecture(root, nil); !errors.Is(err, posemodel.ErrPublishConflict) {
		t.Fatalf("the assess writer overwrote a concurrent edit: %v", err)
	}
	if raw, _ := os.ReadFile(store.StatePath()); !strings.Contains(string(raw), edit+" (again)") {
		t.Fatal("the concurrent edit was lost")
	}
}

// F4/R2/R3 — the artifact backfill rewrites specs under their lifecycle lock
// and only over the bytes it read: it never reverts an edit or a
// transition made meanwhile.
func TestTheBackfillNeverOverwritesAConcurrentSpecEdit(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	raw, _ := os.ReadFile(path)
	stripped := strings.Replace(string(raw), "### Artifacts\n- modified: internal/core.go\n- created: internal/new.go\n\n", "", 1)
	if err := os.WriteFile(path, []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	posemodel.PublishFault = func(p string) error {
		if p != path {
			return nil
		}
		return os.WriteFile(p, []byte(strings.Replace(stripped, "status: in-progress", "status: done", 1)), 0o644)
	}
	t.Cleanup(func() { posemodel.PublishFault = nil })
	var out, errOut bytes.Buffer
	code := cmdArtifactBackfill(root, []string{"--from-git", "--apply", "--confirm-spec-edits"}, &out, &errOut)
	posemodel.PublishFault = nil
	after, _ := os.ReadFile(path)
	if code != 1 || !strings.Contains(errOut.String(), "publish-conflict") || !strings.Contains(string(after), "status: done") {
		t.Fatalf("the backfill reverted a concurrent transition: %d %s\n%s", code, errOut.String(), after)
	}
	unlock, err := posemodel.LockSpecLifecycle(root, "alpha", "close.transition")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	out.Reset()
	errOut.Reset()
	if code := cmdArtifactBackfill(root, []string{"--from-git", "--apply", "--confirm-spec-edits"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "spec-lifecycle-in-progress") {
		t.Fatalf("the backfill interleaved with a close: %d %s", code, errOut.String())
	}
}
