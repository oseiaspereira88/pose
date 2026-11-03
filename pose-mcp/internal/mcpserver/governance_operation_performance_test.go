package mcpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli"
	"github.com/harne8/pose-mcp/internal/mcpserver"
	"github.com/harne8/pose-mcp/internal/pose"
)

// Produce the index through the real CLI collector and consume it over MCP,
// rather than injecting a graph whose attribution was never calculated.
func TestGovernanceOperationPerformanceCLIToMCP(t *testing.T) {
	root := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/project.json", "{\"schema_version\":1,\"project_id\":\"fixture\",\"name\":\"Fixture\"}\n")
	write(".pose/specs/sample.md", "---\nslug: sample\nstatus: in-progress\n---\n\n# Sample\n\n## 3. Technical Plan\n\n### Artifacts\n- created: internal/core.go\n")
	write("internal/core.go", "package core\n")
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "synthetic\n\nPOSE-Spec: sample")
	t.Chdir(root)
	t.Setenv("POSE_PROJECT_ROOT", root)
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	var stdout, stderr bytes.Buffer
	if code := cli.Main([]string{"index"}, &stdout, &stderr); code != 0 {
		t.Fatalf("index failed: %d %s %s", code, stdout.String(), stderr.String())
	}
	store := pose.Store{Root: root}
	want, err := store.GetDeliveryIntegrity("")
	if err != nil || len(want.ChangeSets) != 1 || len(want.ChangeSets[0].Commits) != 1 {
		t.Fatalf("real attribution missing: %+v %v", want, err)
	}
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: root, DefaultProjectID: "fixture", Explicit: map[string]string{"fixture": root}})
	server := httptest.NewServer(mcpserver.NewWithRoots(roots).Handler("", ""))
	t.Cleanup(server.Close)
	post := func(body string) struct {
		Result map[string]any
		Error  any
	} {
		t.Helper()
		response, err := server.Client().Post(server.URL+"/mcp", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var result struct {
			Result map[string]any
			Error  any
		}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_delivery_integrity","arguments":{"project_id":"fixture"}}}`
	result := post(call)
	if result.Error != nil || result.Result["isError"] != false {
		t.Fatalf("MCP rejected generation: %+v", result)
	}
	raw, err := json.Marshal(result.Result["structuredContent"])
	if err != nil {
		t.Fatal(err)
	}
	var got pose.DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MCP changed graph identity/coverage: got=%+v want=%+v", got, want)
	}
	// Mixing a generated file behind its descriptor must still be refused.
	write(".pose/indexes/delivery-integrity.json", "{\"schema_version\":1}\n")
	if err := os.RemoveAll(filepath.Join(root, ".pose/indexes/generations")); err != nil {
		t.Fatal(err)
	}
	result = post(call)
	if result.Error != nil || result.Result["isError"] != true {
		t.Fatalf("MCP accepted mixed generation: %+v", result)
	}
}
