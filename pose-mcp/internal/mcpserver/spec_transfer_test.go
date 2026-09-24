package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func initMCPTransferProject(t *testing.T, root, projectID, slug string) {
	t.Helper()
	for _, dir := range []string{".pose/policy", ".pose/specs", ".pose/roadmaps", ".pose/review-bundles"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":true,"adopted_at":"2026-09-24","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "---\nslug: " + slug + "\nstatus: in-progress\ncreated_at: 2026-09-24\n---\n# Spec: " + slug + "\n\n## 2 Requirements\n\n- R1: Keep authority explicit.\n\n## 3 Technical Plan\n\nPreserve the operation record.\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/specs/2026-09-24-"+slug+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "pose-transfer@example.invalid")
	git("config", "user.name", "POSE transfer fixture")
	git("add", ".pose")
	git("commit", "-q", "-m", "fixture "+projectID)
}

func makeMCPTransferOperation(t *testing.T) (string, string, string, *pose.Roots) {
	t.Helper()
	sourceRoot, destinationRoot := t.TempDir(), t.TempDir()
	initMCPTransferProject(t, sourceRoot, "proj.source", "source-work")
	initMCPTransferProject(t, destinationRoot, "proj.destination", "destination-work")
	unrelatedRoot := t.TempDir()
	initMCPTransferProject(t, unrelatedRoot, "proj.unrelated", "unrelated-work")
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: sourceRoot, DefaultProjectID: "proj.source", Explicit: map[string]string{"proj.destination": destinationRoot, "proj.unrelated": unrelatedRoot}})
	resolver := pose.ArtifactResolver{Roots: roots}
	request := pose.SpecTransferRequest{
		Source:      pose.ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "source-work"},
		Destination: pose.ArtifactRef{Project: "proj.destination", Kind: "spec", Slug: "destination-work"},
		Mappings:    []pose.SpecTransferRequirementMapping{{SourceRequirement: "R1", DestinationRequirement: "R1", Disposition: "equivalent"}},
	}
	plan, err := pose.PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true}
	if _, err := pose.ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err != nil {
		t.Fatal(err)
	}
	return sourceRoot, destinationRoot, plan.OperationID, roots
}

func TestSpecTransferStatusAuthorization(t *testing.T) {
	sourceRoot, destinationRoot, operationID, roots := makeMCPTransferOperation(t)
	var destinationAllowed atomic.Bool
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Input struct {
				ProjectID string `json:"project_id"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		allowed := input.Input.ProjectID == "proj.source" || input.Input.ProjectID == "proj.destination" && destinationAllowed.Load()
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]bool{"allow": allowed}})
	}))
	t.Cleanup(opa.Close)
	server := NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()}))
	ts := httptest.NewServer(server.Handler("", ""))
	t.Cleanup(ts.Close)
	call := func(projectID string) rpcResult {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": "pose_spec_transfer_status", "arguments": map[string]string{"operation_id": operationID, "project_id": projectID}},
		})
		_, result := post(t, ts, string(body))
		return result
	}
	denied := call("proj.destination")
	if denied.Error == nil || denied.Error.Code != -32004 {
		t.Fatalf("unapproved project status was not denied: %+v", denied)
	}
	destinationAllowed.Store(true)
	allowed := call("proj.destination")
	if allowed.Result["isError"] != false {
		t.Fatalf("authorized status failed: %+v", allowed.Result)
	}
	status, ok := allowed.Result["structuredContent"].(map[string]any)
	if !ok || status["phase"] != "activated" || status["project_id"] != "proj.destination" {
		t.Fatalf("status projection = %+v", status)
	}
	raw, _ := json.Marshal(status)
	if strings.Contains(string(raw), sourceRoot) || strings.Contains(string(raw), destinationRoot) {
		t.Fatalf("status disclosed a filesystem path: %s", raw)
	}
	_, specResult := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"project_id":"proj.source","slug":"source-work"}}}`)
	canonical, ok := specResult.Result["structuredContent"].(map[string]any)
	if specResult.Result["isError"] != false || !ok || canonical["slug"] != "destination-work" || canonical["status"] != "in-progress" {
		t.Fatalf("MCP source redirect did not return canonical spec: %+v", specResult.Result)
	}
}

func TestSpecTransferStatusRejectsUnrelatedProject(t *testing.T) {
	sourceRoot, _, operationID, roots := makeMCPTransferOperation(t)
	server := NewWithRoots(roots)
	out, err := server.dispatch(context.Background(), "pose_spec_transfer_status", json.RawMessage(`{"operation_id":"`+operationID+`","project_id":"proj.unrelated"}`))
	status, _ := out.(pose.SpecTransferStatus)
	if err == nil || !strings.Contains(err.Error(), "operation-not-found") || status.Phase != "" {
		t.Fatalf("unrelated project binding = %#v, %v (source root %s)", out, err, sourceRoot)
	}
}
