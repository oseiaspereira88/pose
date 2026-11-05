package mcpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
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

const governanceReviewReadViewProjectID = "proj.b34-review-view"

func TestGovernanceReadViewReviewAndCloseoutParityAcrossCLIAndMCP(t *testing.T) {
	previousEngineVersion := pose.EngineVersion
	pose.EngineVersion = "review-read-view-probe-v1"
	t.Cleanup(func() { pose.EngineVersion = previousEngineVersion })
	root := governanceReviewReadViewFixture(t)
	roots, err := json.Marshal(map[string]string{governanceReviewReadViewProjectID: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", governanceReviewReadViewProjectID)
	t.Setenv("POSE_PROJECT_ROOTS", string(roots))
	t.Setenv("POSE_PROJECT_ROOT", root)
	t.Setenv("HARNE8_PROJECTS_DIR", "")
	t.Chdir(root)

	var stdout, stderr bytes.Buffer
	if code := cli.Main([]string{"review-check", "spec:consumer", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("CLI review-check exited %d: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var cliReview pose.ReviewEvaluation
	if err := json.Unmarshal(stdout.Bytes(), &cliReview); err != nil {
		t.Fatalf("decode CLI review evaluation: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("CLI review-check wrote stderr: %s", stderr.String())
	}

	serverRoots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot: root, DefaultProjectID: governanceReviewReadViewProjectID,
		Explicit: map[string]string{governanceReviewReadViewProjectID: root},
	})
	ts := httptest.NewServer(mcpserver.NewWithRoots(serverRoots).Handler("", ""))
	t.Cleanup(ts.Close)
	body := `{"jsonrpc":"2.0","id":412,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"project_id":"` + governanceReviewReadViewProjectID + `","scope":"spec:consumer"}}}`
	response, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("MCP HTTP status %s", response.Status)
	}
	var envelope governanceReadViewJSONRPCResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error != nil || envelope.Result.IsError || len(envelope.Result.StructuredContent) == 0 {
		t.Fatalf("pose_closeout_state: error=%+v result=%+v", envelope.Error, envelope.Result)
	}
	var mcpState pose.CloseoutState
	if err := json.Unmarshal(envelope.Result.StructuredContent, &mcpState); err != nil {
		t.Fatalf("decode MCP closeout state: %v\n%s", err, envelope.Result.StructuredContent)
	}
	if cliReview.ReadView == nil || mcpState.ReadView == nil || mcpState.Review.ReadView == nil {
		t.Fatalf("review/closeout output omitted read-view metadata: CLI=%+v MCP=%+v", cliReview.ReadView, mcpState.ReadView)
	}
	if cliReview.ReadView.Freshness != pose.ReadViewFreshnessCurrent || mcpState.ReadView.Freshness != pose.ReadViewFreshnessCurrent {
		t.Fatalf("stable Git-backed inputs were not current: CLI=%+v MCP=%+v", cliReview.ReadView, mcpState.ReadView)
	}
	if !reflect.DeepEqual(cliReview.ReadView.Bindings, mcpState.ReadView.Bindings) {
		t.Fatalf("CLI/MCP review bindings differ: CLI=%+v MCP=%+v", cliReview.ReadView.Bindings, mcpState.ReadView.Bindings)
	}
	if cliReview.Scope != mcpState.Review.Scope || cliReview.ScopeDigest != mcpState.Review.ScopeDigest || cliReview.Approved != mcpState.Review.Approved {
		t.Fatalf("CLI review and MCP closeout review differ: CLI=%+v MCP=%+v", cliReview, mcpState.Review)
	}
}

func governanceReviewReadViewFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/project.json", `{"schema_version":1,"project_id":"`+governanceReviewReadViewProjectID+`","name":"review read view fixture"}`+"\n")
	write(".pose/specs/2026-10-01-consumer.md", "---\nslug: consumer\nstatus: draft\n---\n\n# Spec: consumer\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "review read-view fixture"}} {
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	return root
}
