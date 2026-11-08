package mcpserver

// Generation-bound query cursors over MCP (spec pose-generation-bound-query-cursors).

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func cursorServer(t *testing.T, n int) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < n; i++ {
		writeCursorSpec(t, root, fmt.Sprintf("spec-%02d", i))
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts, root
}

func writeCursorSpec(t *testing.T, root, slug string) {
	t.Helper()
	path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("---\nslug: %s\nstatus: draft\ncreated_at: 2026-06-01\n---\n\n# Spec: %s\n", slug, slug)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// call returns the structured result and whether the tool reported an error and its text.
func cursorCall(t *testing.T, ts *httptest.Server, tool string, args map[string]any) (map[string]any, bool, string) {
	t.Helper()
	raw, _ := json.Marshal(args)
	_, out := post(t, ts, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, tool, raw))
	isErr, _ := out.Result["isError"].(bool)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	text := ""
	if content, ok := out.Result["content"].([]any); ok && len(content) > 0 {
		if m, ok := content[0].(map[string]any); ok {
			text, _ = m["text"].(string)
		}
	}
	return structured, isErr, text
}

func TestEveryPageDisclosesItsGenerationAndCollection(t *testing.T) {
	ts, _ := cursorServer(t, 7)
	result, isErr, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	if isErr {
		t.Fatal("first page failed")
	}
	page, _ := result["page"].(map[string]any)
	if page["collection"] == "" || page["collection"] == nil || page["disclosure"] == nil || page["total"] != float64(7) {
		t.Fatalf("the page does not disclose what it was read from: %+v", page)
	}
	if result["next_cursor"] == "" {
		t.Fatal("no cursor for a collection larger than the page")
	}
}

func TestACursorIsRefusedAfterTheCollectionChangesBetweenPages(t *testing.T) {
	ts, root := cursorServer(t, 7)
	first, _, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	cursor, _ := first["next_cursor"].(string)

	// An unchanged collection continues.
	if _, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor}); isErr {
		t.Fatalf("an unchanged collection must continue: %s", text)
	}
	// A new spec arrives: the cursor is stale, not continued over the changed collection.
	writeCursorSpec(t, root, "spec-new")
	_, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor})
	if !isErr || !strings.Contains(text, pose.CursorStale) || !strings.Contains(text, "restart") {
		t.Fatalf("a changed collection was continued or the refusal is not actionable: err=%v %q", isErr, text)
	}
	// Restarting from the first page works and sees the new spec.
	again, isErr, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	if isErr || again["total"] != float64(8) {
		t.Fatalf("restart did not see the changed collection: %+v", again)
	}
}

func TestACursorStaysWithItsToolAndFilters(t *testing.T) {
	ts, _ := cursorServer(t, 7)
	first, _, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	cursor, _ := first["next_cursor"].(string)
	if _, isErr, text := cursorCall(t, ts, "pose_list_roadmaps", map[string]any{"limit": 3, "cursor": cursor}); !isErr || !strings.Contains(text, pose.CursorBinding) {
		t.Fatalf("a cursor crossed to another tool: %v %q", isErr, text)
	}
	if _, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor, "status": "done"}); !isErr || !strings.Contains(text, pose.CursorBinding) {
		t.Fatalf("a cursor survived a changed filter: %v %q", isErr, text)
	}
}

func TestACursorDoesNotSurviveAnotherServerProcess(t *testing.T) {
	ts, root := cursorServer(t, 7)
	first, _, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	cursor, _ := first["next_cursor"].(string)
	other := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(other.Close)
	_, isErr, text := cursorCall(t, other, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor})
	if !isErr || !strings.Contains(text, pose.CursorSignature) {
		t.Fatalf("a cursor crossed a process boundary: %v %q", isErr, text)
	}
}

func TestAHostileCursorIsRefusedOverMCP(t *testing.T) {
	ts, _ := cursorServer(t, 7)
	for _, token := range []string{"AAAA", "eyJ2IjoxLCJhZnRlciI6M30", strings.Repeat("x", pose.GenerationCursorMaxBytes+1)} {
		if _, isErr, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"cursor": token}); !isErr {
			t.Errorf("hostile cursor %.12q was accepted", token)
		}
	}
}

func TestPageLimitIsBoundedAndDisclosed(t *testing.T) {
	ts, _ := cursorServer(t, 3)
	result, isErr, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 100000})
	page, _ := result["page"].(map[string]any)
	if isErr || page["limit_applied"] != float64(pose.GenerationPageMax) {
		t.Fatalf("an oversized limit was not bounded and disclosed: %v %+v", isErr, page)
	}
}

func TestACursorIsRefusedWhenAReturnedFieldChanges(t *testing.T) {
	ts, root := cursorServer(t, 7)
	first, _, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3})
	cursor, _ := first["next_cursor"].(string)
	// A field the listing returns changes while the slug, status and date do not.
	path := filepath.Join(root, ".pose", "specs", "spec-05", "spec.md")
	body := "---\nslug: spec-05\nstatus: draft\ncreated_at: 2026-06-01\npriority: 3\n---\n\n# Spec: spec-05\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor})
	if !isErr || !strings.Contains(text, pose.CursorStale) {
		t.Fatalf("a cursor continued over changed results: %v %q", isErr, text)
	}
}

func TestEquivalentFiltersShareACursor(t *testing.T) {
	ts, _ := cursorServer(t, 7)
	first, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "status": "Draft, draft"})
	if isErr {
		t.Fatal(text)
	}
	cursor, _ := first["next_cursor"].(string)
	if _, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 3, "cursor": cursor, "status": "draft"}); isErr {
		t.Fatalf("equivalent filters were treated as another query: %s", text)
	}
}

func TestAPageStatesItsCostAndGenerationStatus(t *testing.T) {
	ts, _ := cursorServer(t, 3)
	result, isErr, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 2})
	page, _ := result["page"].(map[string]any)
	if isErr || page["cost"] == nil || !strings.Contains(page["cost"].(string), "full-scan") || page["generation_status"] == nil {
		t.Fatalf("a page must state its cost and the status of its generation: %v %+v", isErr, page)
	}
}

func TestAContinuationWithoutALimitIsBoundedOverMCP(t *testing.T) {
	ts, _ := cursorServer(t, pose.GenerationPageMax+25)
	first, _, _ := cursorCall(t, ts, "pose_list_specs", map[string]any{"limit": 1})
	cursor, _ := first["next_cursor"].(string)
	second, isErr, text := cursorCall(t, ts, "pose_list_specs", map[string]any{"cursor": cursor})
	if isErr {
		t.Fatal(text)
	}
	if got := second["count"]; got != float64(pose.GenerationPageMax) {
		t.Fatalf("a continuation without limit returned %v items, want %d", got, pose.GenerationPageMax)
	}
}
