package mcpserver_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli"
	"github.com/harne8/pose-mcp/internal/mcpserver"
	"github.com/harne8/pose-mcp/internal/pose"
)

const governanceReadViewParityProjectID = "proj.b34-read-view"

type governanceReadViewParitySnapshot struct {
	ProjectID      string                `json:"project_id"`
	SourceRevision string                `json:"source_revision"`
	WorktreeDirty  bool                  `json:"worktree_dirty"`
	Coherent       bool                  `json:"coherent"`
	Limitations    []string              `json:"limitations,omitempty"`
	ReadView       pose.ReadViewMetadata `json:"read_view"`
}

type governanceReadViewParityReport struct {
	SchemaVersion int                              `json:"schema_version"`
	Snapshot      governanceReadViewParitySnapshot `json:"snapshot"`
	Coverage      []pose.ProducerCoverage          `json:"coverage"`
	Complete      bool                             `json:"complete"`
	Obligations   []pose.Obligation                `json:"obligations"`
	Limitations   []string                         `json:"limitations,omitempty"`
}

type governanceReadViewJSONRPCResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Result struct {
		IsError           bool            `json:"isError"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	} `json:"result"`
}

// Spec pose-governance-read-view-generation V5: real CLI and MCP entrypoints
// disclose the same operation-bound view and preserve the domain projection.
func TestGovernanceReadViewParityAcrossCLIAndMCPEntryPoints(t *testing.T) {
	previousEngineVersion := pose.EngineVersion
	pose.EngineVersion = "probe-engine-v1"
	t.Cleanup(func() { pose.EngineVersion = previousEngineVersion })
	root := governanceReadViewParityFixture(t)
	governanceReadViewParitySelectCLIProject(t, root)
	worktreeBefore := governanceReadViewParityTreeDigest(t, root)

	cliReport := governanceReadViewParityCallCLI(t, root)
	cliAgain := governanceReadViewParityCallCLI(t, root)
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      root,
		DefaultProjectID: governanceReadViewParityProjectID,
		Explicit:         map[string]string{governanceReadViewParityProjectID: root},
	})
	ts := httptest.NewServer(mcpserver.NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)
	mcpReport := governanceReadViewParityCallTool(t, ts)
	mcpAgain := governanceReadViewParityCallTool(t, ts)

	resolver, selectedProject, err := pose.EnvironmentArtifactResolver(root, "")
	if err != nil {
		t.Fatalf("resolve explicit domain project context: %v", err)
	}
	if selectedProject != governanceReadViewParityProjectID {
		t.Fatalf("domain selected project %q, want %q", selectedProject, governanceReadViewParityProjectID)
	}
	domainStore := pose.Store{Root: root, FederatedProjectID: selectedProject, FederatedResolver: &resolver}
	domainReport, err := domainStore.ProjectObligations(pose.ObligationQuery{})
	if err != nil {
		t.Fatalf("domain ProjectObligations: %v", err)
	}
	domainJSON, err := json.Marshal(domainReport)
	if err != nil {
		t.Fatal(err)
	}
	var domain governanceReadViewParityReport
	if err := json.Unmarshal(domainJSON, &domain); err != nil {
		t.Fatalf("decode domain report: %v\n%s", err, domainJSON)
	}

	reports := map[string]governanceReadViewParityReport{
		"CLI": cliReport, "repeated CLI": cliAgain, "MCP": mcpReport,
		"repeated MCP": mcpAgain, "domain": domain,
	}
	for name, report := range reports {
		governanceReadViewParityAssertReport(t, name, report)
	}
	if !reflect.DeepEqual(cliReport, cliAgain) {
		t.Fatalf("stable CLI reads diverged:\nfirst=%+v\nrepeated=%+v", cliReport, cliAgain)
	}
	if !reflect.DeepEqual(mcpReport, mcpAgain) {
		t.Fatalf("stable MCP reads diverged:\nfirst=%+v\nrepeated=%+v", mcpReport, mcpAgain)
	}
	for name, report := range reports {
		if !reflect.DeepEqual(cliReport.Snapshot.ReadView, report.Snapshot.ReadView) {
			t.Errorf("%s read-view identity/bindings differ from CLI: got=%+v want=%+v", name, report.Snapshot.ReadView, cliReport.Snapshot.ReadView)
		}
		if !reflect.DeepEqual(cliReport.Coverage, report.Coverage) {
			t.Errorf("%s producer coverage differs from CLI: got=%+v want=%+v", name, report.Coverage, cliReport.Coverage)
		}
		if !reflect.DeepEqual(governanceReadViewParityObligationIDs(cliReport.Obligations), governanceReadViewParityObligationIDs(report.Obligations)) {
			t.Errorf("%s obligation IDs differ from CLI: got=%v want=%v", name, governanceReadViewParityObligationIDs(report.Obligations), governanceReadViewParityObligationIDs(cliReport.Obligations))
		}
		if report.Snapshot.ProjectID != cliReport.Snapshot.ProjectID ||
			report.Snapshot.SourceRevision != cliReport.Snapshot.SourceRevision ||
			report.Snapshot.WorktreeDirty != cliReport.Snapshot.WorktreeDirty {
			t.Errorf("%s project/revision/worktree bindings differ from CLI: got=%+v want=%+v", name, report.Snapshot, cliReport.Snapshot)
		}
		if report.Complete != cliReport.Complete || report.Snapshot.Coherent != cliReport.Snapshot.Coherent ||
			!reflect.DeepEqual(report.Limitations, cliReport.Limitations) ||
			!reflect.DeepEqual(report.Snapshot.Limitations, cliReport.Snapshot.Limitations) {
			t.Errorf("%s completion/coherence/limitations differ from CLI: complete=%t coherent=%t limits=%v snapshot limits=%v", name, report.Complete, report.Snapshot.Coherent, report.Limitations, report.Snapshot.Limitations)
		}
	}
	if cliReport.Snapshot.ReadView.ID == "" || cliReport.Snapshot.ReadView.ID != mcpReport.Snapshot.ReadView.ID {
		t.Fatalf("CLI/MCP operation IDs differ: CLI=%q MCP=%q", cliReport.Snapshot.ReadView.ID, mcpReport.Snapshot.ReadView.ID)
	}
	if worktreeAfter := governanceReadViewParityTreeDigest(t, root); worktreeAfter != worktreeBefore {
		t.Fatalf("governance read entrypoints changed the project tree: before=%s after=%s", worktreeBefore, worktreeAfter)
	}
}

func governanceReadViewParityFixture(t *testing.T) string {
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
	write(".pose/project.json", "{\"schema_version\":1,\"project_id\":\""+governanceReadViewParityProjectID+"\",\"name\":\"B34 fixture\"}\n")
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"}}`)
	write(".pose/specs/2026-10-01-base.md", "---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n\n## 2. Requirements\n- R1: x\n")
	write(".pose/specs/2026-10-02-consumer.md", "---\nslug: consumer\nstatus: draft\ndepends_on: base\n---\n\n# Spec: consumer\n")
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "B34 governance read-view parity fixture")
	return root
}

func governanceReadViewParitySelectCLIProject(t *testing.T, root string) {
	t.Helper()
	roots, err := json.Marshal(map[string]string{governanceReadViewParityProjectID: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", governanceReadViewParityProjectID)
	t.Setenv("POSE_PROJECT_ROOTS", string(roots))
	t.Setenv("POSE_PROJECT_ROOT", root)
	t.Setenv("HARNE8_PROJECTS_DIR", "")
}

func governanceReadViewParityCallCLI(t *testing.T, root string) governanceReadViewParityReport {
	t.Helper()
	var stdout, stderr bytes.Buffer
	t.Chdir(root)
	if code := cli.Main([]string{"state", "--attention", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pose state --attention --json exited %d: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report governanceReadViewParityReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode CLI report: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("pose state --attention --json wrote stderr: %s", stderr.String())
	}
	return report
}

func governanceReadViewParityCallTool(t *testing.T, ts *httptest.Server) governanceReadViewParityReport {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":301,"method":"tools/call","params":{"name":"pose_obligations","arguments":{"project_id":"` + governanceReadViewParityProjectID + `"}}}`
	response, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(response.Body)
		t.Fatalf("MCP HTTP status %s: %s", response.Status, raw)
	}
	var envelope governanceReadViewJSONRPCResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode MCP response: %v", err)
	}
	if envelope.Error != nil || envelope.Result.IsError || len(envelope.Result.StructuredContent) == 0 {
		t.Fatalf("pose_obligations: error=%+v result=%+v", envelope.Error, envelope.Result)
	}
	var report governanceReadViewParityReport
	if err := json.Unmarshal(envelope.Result.StructuredContent, &report); err != nil {
		t.Fatalf("decode MCP report: %v\n%s", err, envelope.Result.StructuredContent)
	}
	return report
}

func governanceReadViewParityAssertReport(t *testing.T, name string, report governanceReadViewParityReport) {
	t.Helper()
	view := report.Snapshot.ReadView
	if report.SchemaVersion != 1 || view.SchemaVersion != 1 {
		t.Errorf("%s schema versions = report:%d read_view:%d, want 1/1", name, report.SchemaVersion, view.SchemaVersion)
	}
	if view.ID == "" || view.Freshness != pose.ReadViewFreshnessCurrent {
		t.Errorf("%s read_view = %+v, want a current identity on the bound fixture", name, view)
	}
	found := false
	for _, binding := range view.Bindings {
		if binding.ProjectID == governanceReadViewParityProjectID {
			found = true
			if binding.RootDigest == "" || binding.SourceRevision == "" {
				t.Errorf("%s project binding lacks root/source identity: %+v", name, binding)
			}
		}
	}
	if !found {
		t.Errorf("%s read-view bindings omit project %q: %+v", name, governanceReadViewParityProjectID, view.Bindings)
	}
	foundDependency := false
	for _, obligation := range report.Obligations {
		foundDependency = foundDependency || obligation.Category == pose.ObligationDependency && obligation.ReasonCode == "dependency-not-done"
	}
	if !foundDependency {
		t.Errorf("%s omitted the fixture's dependency obligation: %+v", name, report.Obligations)
	}
}

func governanceReadViewParityObligationIDs(obligations []pose.Obligation) []string {
	ids := make([]string, 0, len(obligations))
	for _, obligation := range obligations {
		ids = append(ids, obligation.ID)
	}
	return ids
}

func governanceReadViewParityTreeDigest(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" {
			return filepath.SkipDir
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write([]byte(info.Mode().String()))
		h.Write([]byte{0})
		h.Write([]byte(strconv.FormatInt(info.Size(), 10)))
		h.Write([]byte{0})
		h.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
		h.Write([]byte{0})
		h.Write([]byte(governanceReadViewParityFileIdentity(info)))
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			h.Write([]byte(target))
			return nil
		}
		if !info.IsDir() {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(body)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func governanceReadViewParityFileIdentity(info os.FileInfo) string {
	value := reflect.ValueOf(info.Sys())
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	fields := []string{}
	for _, name := range []string{"Dev", "Ino", "VolumeSerialNumber", "FileIndexHigh", "FileIndexLow", "FileID", "Ctim", "Ctime", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.CanInterface() {
			fields = append(fields, name+"="+fmt.Sprint(field.Interface()))
		}
	}
	return strings.Join(fields, "\x00")
}
