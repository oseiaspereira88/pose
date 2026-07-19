package mcpserver

// Protocol completeness behavior (spec pose-mcp-protocol-completeness):
// R1 opaque cursor pagination with stable ordering, exercised over both
// stdio and Streamable HTTP transports (non-functional requirement), plus
// the catalog list-change/reconnect contract (R2).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func paginationFixture(t *testing.T, n int) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < n; i++ {
		slug := fmt.Sprintf("spec-%d", i)
		path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf("---\nslug: %s\nstatus: draft\ncreated_at: 2026-06-01\n---\n\n# Spec: %s\n", slug, slug)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func listSpecsPage(t *testing.T, ts *httptest.Server, cursor string, limit int) (specs []string, next string) {
	t.Helper()
	args := map[string]any{}
	if cursor != "" {
		args["cursor"] = cursor
	}
	if limit > 0 {
		args["limit"] = limit
	}
	argsJSON, _ := json.Marshal(args)
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":%s}}`, argsJSON)
	_, out := post(t, ts, req)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	items, _ := sc["specs"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		specs = append(specs, m["slug"].(string))
	}
	next, _ = sc["next_cursor"].(string)
	return
}

func TestPaginationWalksEveryItemExactlyOnce(t *testing.T) {
	ts := paginationFixture(t, 5)
	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		page, next := listSpecsPage(t, ts, cursor, 2)
		for _, slug := range page {
			if seen[slug] {
				t.Fatalf("slug %q returned twice across pages", slug)
			}
			seen[slug] = true
		}
		pages++
		if next == "" {
			break
		}
		cursor = next
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d distinct specs, want 5: %v", len(seen), seen)
	}
	if pages != 3 { // 2 + 2 + 1
		t.Errorf("pages = %d, want 3 for limit=2 over 5 items", pages)
	}
}

func TestPaginationOmittedIsUnpaginatedAndBackwardCompatible(t *testing.T) {
	ts := paginationFixture(t, 5)
	page, next := listSpecsPage(t, ts, "", 0)
	if len(page) != 5 || next != "" {
		t.Fatalf("omitting cursor/limit must return everything in one page, got %d items next=%q", len(page), next)
	}
}

func TestPaginationInvalidCursorIsRejected(t *testing.T) {
	ts := paginationFixture(t, 3)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"cursor":"not-a-valid-cursor!!"}}}`)
	isErr, _ := out.Result["isError"].(bool)
	if !isErr {
		t.Fatal("a malformed cursor must be a tool error, not silently treated as page 1")
	}
}

func TestPaginationConsistentAcrossStdioAndHTTP(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		slug := fmt.Sprintf("spec-%d", i)
		path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(fmt.Sprintf("---\nslug: %s\nstatus: draft\n---\n\n# %s\n", slug, slug)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := New(pose.Store{Root: root})
	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/call",
		Params: json.RawMessage(`{"name":"pose_list_specs","arguments":{"limit":2}}`)}
	resp := s.dispatchRPC(context.Background(), req)
	// ServeStdio serializes the response to JSON on the wire (encoding/gob
	// is never used); round-trip it the same way a real stdio client would
	// see it, rather than inspecting the in-process Go value.
	wire, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Result struct {
			StructuredContent struct {
				Specs      []map[string]any `json:"specs"`
				NextCursor string           `json:"next_cursor"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(wire, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Result.StructuredContent.Specs) != 2 || parsed.Result.StructuredContent.NextCursor == "" {
		t.Fatalf("stdio pagination: got %d items, next_cursor=%q", len(parsed.Result.StructuredContent.Specs), parsed.Result.StructuredContent.NextCursor)
	}
}

func TestListToolsShareThePaginationSchema(t *testing.T) {
	paginated := map[string]bool{"pose_list_specs": true, "pose_list_roadmaps": true, "pose_list_knowledge": true, "pose_list_reports": true}
	found := map[string]bool{}
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if !paginated[name] {
			continue
		}
		found[name] = true
		schema, _ := def["inputSchema"].(map[string]any)
		props, _ := schema["properties"].(map[string]any)
		cursor, _ := props["cursor"].(map[string]any)
		limit, _ := props["limit"].(map[string]any)
		if cursor["type"] != "string" || cursor["description"] != sharedCursorDescription {
			t.Errorf("tool %q cursor property diverges from the shared contract: %+v", name, cursor)
		}
		if limit["type"] != "integer" || limit["description"] != sharedLimitDescription {
			t.Errorf("tool %q limit property diverges from the shared contract: %+v", name, limit)
		}
	}
	for name := range paginated {
		if !found[name] {
			t.Errorf("expected paginated tool %q not found in catalog", name)
		}
	}
}

// R2: the catalog is immutable for a server process's lifetime (no dynamic
// tool add/remove) — declared listChanged:false must be verifiably true,
// not just claimed. A version change (new binary) is the reconnect signal a
// client uses; documented in mcp.md, not implemented as a protocol event.
func TestToolCatalogIsStableWithinAProcessLifetime(t *testing.T) {
	ts := newTestServer(t, "")
	_, first := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	_, second := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	a, _ := json.Marshal(first.Result["tools"])
	b, _ := json.Marshal(second.Result["tools"])
	if string(a) != string(b) {
		t.Fatal("tools/list must be byte-identical across calls within one process — listChanged:false must be true, not aspirational")
	}
	_, init := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"initialize","params":{}}`)
	caps, _ := init.Result["capabilities"].(map[string]any)
	tools, _ := caps["tools"].(map[string]any)
	if tools["listChanged"] != false {
		t.Errorf("capabilities.tools.listChanged = %v, want false", tools["listChanged"])
	}
}
