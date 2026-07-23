package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func hookTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runStateGit(t, root, "init")
	runStateGit(t, root, "config", "user.email", "test@example.com")
	runStateGit(t, root, "config", "user.name", "Hooks Test")
	writeStateTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\n---\n\n# Spec: alpha\n")
	runStateGit(t, root, "add", "-A")
	runStateGit(t, root, "commit", "-m", "seed")
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	return root
}

func TestRegisterAndEmitHook_BestEffortSwallowsPanic(t *testing.T) {
	kind := "test_kind_" + t.Name()
	called := 0
	RegisterHook(kind, func(root string, ev HookEvent) error {
		called++
		panic("boom")
	})
	EmitHook(t.TempDir(), HookEvent{Kind: kind})
	if called != 1 {
		t.Fatalf("consumer called %d times, want 1", called)
	}
	// The panic must not have propagated past EmitHook (test reaching here proves it).
}

func TestEmitHook_NoOpWhenNoProjectState(t *testing.T) {
	root := t.TempDir() // no `pose state init` here
	runStateGit(t, root, "init")
	// Registered consumers for real event kinds must not error/panic on a
	// project without .pose/state/ — Compatibilidade (aditivo).
	EmitHook(root, HookEvent{Kind: "spec_amend", Target: "whatever", Commit: "abc1234"})
	if _, err := os.Stat(filepath.Join(root, ".pose", "state")); !os.IsNotExist(err) {
		t.Fatalf(".pose/state must not be created as a side effect of a no-op hook: %v", err)
	}
}

func TestStateRefreshConsumer_PartialRefreshOnlyTouchesMappedSections(t *testing.T) {
	root := hookTestRoot(t)
	before, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}

	writeStateTestFile(t, root, ".pose/specs/beta/spec.md", "---\nslug: beta\nstatus: draft\n---\n\n# Spec: beta\n")
	EmitHook(root, HookEvent{Kind: "assessment_snapshot", Commit: gitHeadCommit(root)})

	after, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	// assessment_snapshot only maps to Capabilities — the new spec must NOT
	// show up in Specs & Roadmaps, because that section was not requested.
	if strings.Contains(string(after), "total=2") {
		t.Fatalf("assessment_snapshot must not have refreshed Specs & Roadmaps:\n%s", after)
	}
	_ = before // the artifact may be byte-identical when nothing mapped changed within the same second; the log below is the precise assertion

	entries := readHistoryForTest(t, root)
	if len(entries) != 2 {
		t.Fatalf("history entries = %d, want 2 (init + hook refresh)", len(entries))
	}
	log, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var hookEntries []refreshLogEntry
	for _, e := range log {
		if e.Trigger == "assessment_snapshot" {
			hookEntries = append(hookEntries, e)
		}
	}
	if len(hookEntries) != 1 || hookEntries[0].Result != "ok" {
		t.Fatalf("refresh-log assessment_snapshot entries = %+v, want one ok entry", hookEntries)
	}
}

func TestStateRefreshConsumer_EventDedup(t *testing.T) {
	root := hookTestRoot(t)
	ev := HookEvent{Kind: "evidence_reconciled", Target: "req-1", Commit: gitHeadCommit(root)}
	EmitHook(root, ev)
	EmitHook(root, ev) // same event, same commit — must be deduped (R6)

	log, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	for _, e := range log {
		if e.Trigger == "evidence_reconciled" {
			results = append(results, e.Result)
		}
	}
	if len(results) != 2 || results[0] != "ok" || results[1] != "skipped" {
		t.Fatalf("refresh-log results for the repeated event = %v, want [ok skipped]", results)
	}
}

// blockNewEntriesInStateDir removes write permission from .pose/state/ so
// writeAtomic's temp-file-then-rename (used by the main refresh write path)
// reliably fails, while direct in-place edits to files that already exist
// keep working (relied on by markRefreshPending — see its own comment).
// Restores permissions in cleanup so t.TempDir() can remove the tree.
func blockNewEntriesInStateDir(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".pose", "state")
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

func TestStateRefreshConsumer_FailureMarksRefreshPendingNonStrict(t *testing.T) {
	root := hookTestRoot(t)
	blockNewEntriesInStateDir(t, root)

	EmitHook(root, HookEvent{Kind: "spec_amend", Target: "alpha", Commit: gitHeadCommit(root)})

	path := filepath.Join(root, ".pose", "state", "project-state.md")
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "refresh_pending: spec_amend") {
		t.Fatalf("expected refresh_pending to be set after a failed hook refresh, got:\n%s", after)
	}

	state, err := pose.Store{Root: root}.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if state.RefreshPending != "spec_amend" {
		t.Fatalf("ProjectState.RefreshPending = %q, want spec_amend", state.RefreshPending)
	}
}

func TestStateRefreshConsumer_StrictModePropagatesError(t *testing.T) {
	root := hookTestRoot(t)
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "state.json"), []byte(`{"strict_refresh":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	blockNewEntriesInStateDir(t, root)

	err := stateRefreshConsumer(root, HookEvent{Kind: "spec_amend", Target: "alpha", Commit: gitHeadCommit(root)})
	if err == nil {
		t.Fatal("strict_refresh must propagate the refresh failure instead of swallowing it")
	}
}

func TestRefreshPendingClearedByNextSuccessfulRefresh(t *testing.T) {
	root := hookTestRoot(t)
	if err := markRefreshPending(root, "spec_amend"); err != nil {
		t.Fatal(err)
	}
	state, err := pose.Store{Root: root}.ProjectState(context.Background(), "")
	if err != nil || state.RefreshPending != "spec_amend" {
		t.Fatalf("setup: RefreshPending = %q err=%v, want spec_amend", state.RefreshPending, err)
	}

	if code, _, errOut := runState(t, root, "refresh"); code != 0 {
		t.Fatalf("refresh: %v", errOut)
	}
	state, err = pose.Store{Root: root}.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if state.RefreshPending != "" {
		t.Fatalf("RefreshPending = %q after a successful refresh, want cleared", state.RefreshPending)
	}
}

func TestCmdStateRefresh_IfStaleSkipsWhenFresh(t *testing.T) {
	root := hookTestRoot(t)
	before := readHistoryForTest(t, root)

	code, out, _ := runState(t, root, "refresh", "--if-stale")
	if code != 0 || !strings.Contains(out, "skipped") {
		t.Fatalf("--if-stale on a fresh artifact should skip: code=%d out=%q", code, out)
	}
	after := readHistoryForTest(t, root)
	if len(after) != len(before) {
		t.Fatalf("--if-stale performed a refresh anyway: history %d -> %d", len(before), len(after))
	}
}

func TestSectionsForEvent_UnknownKindDefaultsToFullRefresh(t *testing.T) {
	got := sectionsForEvent("some_future_event_nobody_mapped_yet")
	want := 0
	for _, def := range stateSectionOrder {
		if !def.curated {
			want++
		}
	}
	if len(got) != want {
		t.Fatalf("unmapped event kind sections = %d, want all %d derived sections (never silently skip)", len(got), want)
	}
}

func TestRenderDirectedHitSummary(t *testing.T) {
	empty := renderDirectedHitSummary(&graphForgeHitResult{})
	if !strings.Contains(empty, "nenhum componente") {
		t.Errorf("empty hit summary = %q", empty)
	}
	withHits := renderDirectedHitSummary(&graphForgeHitResult{Hits: []map[string]any{
		{"component_id": "conductor-internal-policy", "level": "direct"},
	}})
	if !strings.Contains(withHits, "component:conductor-internal-policy") || !strings.Contains(withHits, "[direct]") {
		t.Errorf("hit summary = %q", withHits)
	}
}

func TestResolveComponentsHitCaller_UnconfiguredIsNil(t *testing.T) {
	t.Setenv("POSE_GRAPHFORGE_MCP_URL", "")
	if caller := resolveComponentsHitCaller(t.TempDir()); caller != nil {
		t.Fatalf("caller = %v, want nil when POSE_GRAPHFORGE_MCP_URL is unset", caller)
	}
}

func TestHTTPComponentsHitCaller_CallsToolsCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "tools/call" {
			t.Fatalf("method = %q, want tools/call", req.Method)
		}
		params, _ := req.Params.(map[string]any)
		if params["name"] != "components_hit" {
			t.Fatalf("tool = %v, want components_hit", params["name"])
		}
		structured, _ := json.Marshal(graphForgeHitResult{Hits: []map[string]any{{"component_id": "x", "level": "direct"}}})
		resp := jsonRPCToolCallResult{Result: &struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
			IsError           bool            `json:"isError"`
		}{StructuredContent: structured}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	caller := httpComponentsHitCaller{url: srv.URL, projectID: "demo"}
	result, ok, err := caller.ComponentsHit(context.Background(), "abc1234", "def5678")
	if err != nil || !ok {
		t.Fatalf("ComponentsHit: ok=%v err=%v", ok, err)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("hits = %v", result.Hits)
	}
}

func TestHTTPComponentsHitCaller_UnreachableDegradesNotErrors(t *testing.T) {
	caller := httpComponentsHitCaller{url: "http://127.0.0.1:1", projectID: "demo"}
	_, ok, err := caller.ComponentsHit(context.Background(), "abc1234", "def5678")
	if ok || err != nil {
		t.Fatalf("unreachable GraphForge must degrade (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}
}
