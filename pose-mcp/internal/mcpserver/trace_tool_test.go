package mcpserver

// Behavior tests for pose_requirement_trace (spec
// pose-requirement-evidence-traceability R3: bidirectional traversal via MCP).

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

const tracedSpec = `---
slug: traced
status: done
created_at: 2026-07-01
completed_at: 2026-07-02
---

# Spec: traced

## 2. Requirements

### Functional
- R1: behave correctly.
- R2: stay traced.

## 6. Validation

### Requirement trace
- R1 [satisfied] unit suite; check:test report:2026-07-02-report.md
- R2 [waived: covered upstream] check:test

## 7. Final Report
`

func newTraceServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".pose", "specs", "traced", "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(tracedSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func TestRequirementTraceTool(t *testing.T) {
	ts := newTraceServer(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_requirement_trace","arguments":{"slug":"traced"}}}`)
	if isErr, _ := out.Result["isError"].(bool); isErr {
		t.Fatalf("unexpected tool error: %v", out.Result)
	}
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc == nil {
		// content fallback: parse text payload
		content, _ := out.Result["content"].([]any)
		if len(content) == 0 {
			t.Fatal("no content in result")
		}
		text, _ := content[0].(map[string]any)["text"].(string)
		if err := json.Unmarshal([]byte(text), &sc); err != nil {
			t.Fatalf("parsing tool payload: %v", err)
		}
	}
	trace, _ := sc["trace"].(map[string]any)
	if trace == nil {
		t.Fatalf("missing trace in payload: %v", sc)
	}
	reqs, _ := trace["requirements"].([]any)
	if len(reqs) != 2 {
		t.Fatalf("requirements = %d, want 2", len(reqs))
	}
	byEvidence, _ := trace["by_evidence"].(map[string]any)
	ids, _ := byEvidence["check:test"].([]any)
	if len(ids) != 2 {
		t.Errorf("by_evidence[check:test] = %v, want both R1 and R2", ids)
	}
	if has, _ := trace["has_section"].(bool); !has {
		t.Error("has_section should be true")
	}
}

func TestRequirementTraceToolMissingSlug(t *testing.T) {
	ts := newTraceServer(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_requirement_trace","arguments":{}}}`)
	if isErr, _ := out.Result["isError"].(bool); !isErr {
		t.Fatal("expected isError for missing slug")
	}
}
