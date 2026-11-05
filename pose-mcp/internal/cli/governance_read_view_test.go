package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

const governanceReadViewParityProjectID = "proj.b34-read-view"

type governanceReadViewParityBinding struct {
	ProjectID       string `json:"project_id"`
	RootDigest      string `json:"root_digest"`
	SourceRevision  string `json:"source_revision"`
	IndexGeneration string `json:"index_generation,omitempty"`
}

type governanceReadViewParityMetadata struct {
	SchemaVersion int                               `json:"schema_version"`
	ID            string                            `json:"id"`
	Bindings      []governanceReadViewParityBinding `json:"bindings"`
	Freshness     string                            `json:"freshness"`
}

type governanceReadViewParitySnapshot struct {
	ProjectID      string                           `json:"project_id"`
	SourceRevision string                           `json:"source_revision"`
	WorktreeDirty  bool                             `json:"worktree_dirty"`
	ReadView       governanceReadViewParityMetadata `json:"read_view"`
}

type governanceReadViewParityPayload struct {
	SchemaVersion int                              `json:"schema_version"`
	Snapshot      governanceReadViewParitySnapshot `json:"snapshot"`
	Coverage      []pose.ProducerCoverage          `json:"coverage"`
	Obligations   []pose.Obligation                `json:"obligations"`
	Attention     pose.Attention                   `json:"attention"`
}

func TestGovernanceReadViewParityCLIStableDirtyToDirtyAndReadOnly(t *testing.T) {
	previousEngineVersion := pose.EngineVersion
	pose.EngineVersion = "probe-engine-v1"
	t.Cleanup(func() { pose.EngineVersion = previousEngineVersion })
	root := governanceReadViewParityCLIFixture(t)
	governanceReadViewParitySelectCLIProject(t, root)

	path := filepath.Join(root, ".pose/specs/2026-10-01-base.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes := bytes.Replace(original, []byte("R1: x"), []byte("R1: z"), 1)
	if bytes.Equal(firstBytes, original) {
		t.Fatal("fixture spec did not contain the expected requirement")
	}
	if len(firstBytes) != len(original) {
		t.Fatalf("fixture edit changed file size: original=%d edited=%d", len(original), len(firstBytes))
	}
	if err := os.WriteFile(path, firstBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	firstInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	beforeRead := governanceReadViewParityTreeDigest(t, root)
	first := governanceReadViewParityRunCLI(t, root)
	firstAgain := governanceReadViewParityRunCLI(t, root)
	if got := governanceReadViewParityTreeDigest(t, root); got != beforeRead {
		t.Fatalf("CLI read changed the worktree: before=%s after=%s", beforeRead, got)
	}
	governanceReadViewParityAssertBoundReport(t, first)
	governanceReadViewParityAssertBoundReport(t, firstAgain)
	if first.Snapshot.ReadView.ID != firstAgain.Snapshot.ReadView.ID {
		t.Fatalf("stable CLI reads changed identity: first=%q repeated=%q", first.Snapshot.ReadView.ID, firstAgain.Snapshot.ReadView.ID)
	}
	if !reflect.DeepEqual(first.Snapshot.ReadView.Bindings, firstAgain.Snapshot.ReadView.Bindings) {
		t.Fatalf("stable CLI reads changed bindings: first=%+v repeated=%+v", first.Snapshot.ReadView.Bindings, firstAgain.Snapshot.ReadView.Bindings)
	}
	if !reflect.DeepEqual(first.Coverage, firstAgain.Coverage) {
		t.Fatalf("stable CLI reads changed producer coverage: first=%+v repeated=%+v", first.Coverage, firstAgain.Coverage)
	}
	if !first.Snapshot.WorktreeDirty || !firstAgain.Snapshot.WorktreeDirty {
		t.Fatalf("fixture should be dirty for both stable reads: first=%+v repeated=%+v", first.Snapshot, firstAgain.Snapshot)
	}

	secondBytes := bytes.Replace(firstBytes, []byte("R1: z"), []byte("R1: y"), 1)
	if len(secondBytes) != len(firstBytes) || reflect.DeepEqual(secondBytes, firstBytes) {
		t.Fatal("second same-size edit is not materially distinct")
	}
	if err := os.WriteFile(path, secondBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, firstInfo.ModTime(), firstInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if secondInfo.Size() != firstInfo.Size() || !secondInfo.ModTime().Equal(firstInfo.ModTime()) {
		t.Fatalf("dirty-to-dirty edit did not preserve size/mtime: first=%v second=%v", firstInfo, secondInfo)
	}

	beforeSecondRead := governanceReadViewParityTreeDigest(t, root)
	second := governanceReadViewParityRunCLI(t, root)
	secondAgain := governanceReadViewParityRunCLI(t, root)
	if got := governanceReadViewParityTreeDigest(t, root); got != beforeSecondRead {
		t.Fatalf("CLI read after input edit changed the worktree: before=%s after=%s", beforeSecondRead, got)
	}
	governanceReadViewParityAssertBoundReport(t, second)
	governanceReadViewParityAssertBoundReport(t, secondAgain)
	if !second.Snapshot.WorktreeDirty || !secondAgain.Snapshot.WorktreeDirty {
		t.Fatalf("fixture should remain dirty after both edits: second=%+v repeated=%+v", second.Snapshot, secondAgain.Snapshot)
	}
	if second.Snapshot.SourceRevision != first.Snapshot.SourceRevision {
		t.Fatalf("dirty-to-dirty edit changed HEAD binding: first=%q second=%q", first.Snapshot.SourceRevision, second.Snapshot.SourceRevision)
	}
	if second.Snapshot.ReadView.ID == first.Snapshot.ReadView.ID {
		t.Fatalf("same-size input edit in an already-dirty tree kept read-view identity %q", second.Snapshot.ReadView.ID)
	}
	if second.Snapshot.ReadView.ID != secondAgain.Snapshot.ReadView.ID {
		t.Fatalf("stable second CLI reads changed identity: second=%q repeated=%q", second.Snapshot.ReadView.ID, secondAgain.Snapshot.ReadView.ID)
	}
	if !reflect.DeepEqual(second.Snapshot.ReadView.Bindings, secondAgain.Snapshot.ReadView.Bindings) || !reflect.DeepEqual(second.Coverage, secondAgain.Coverage) {
		t.Fatalf("stable second reads changed bindings or coverage: second=%+v repeated=%+v", second, secondAgain)
	}
}

func governanceReadViewParityCLIFixture(t *testing.T) string {
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
	git("commit", "-q", "-m", "B34 governance read-view fixture")
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

func governanceReadViewParityRunCLI(t *testing.T, root string) governanceReadViewParityPayload {
	t.Helper()
	var stdout, stderr strings.Builder
	inDir(t, root, func() {
		if code := Main([]string{"state", "--attention", "--json"}, &stdout, &stderr); code != 0 {
			t.Fatalf("pose state --attention --json exited %d: stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
	})
	var report governanceReadViewParityPayload
	if err := json.Unmarshal([]byte(stdout.String()), &report); err != nil {
		t.Fatalf("decode CLI report: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("pose state --attention --json wrote stderr: %s", stderr.String())
	}
	return report
}

func governanceReadViewParityAssertBoundReport(t *testing.T, report governanceReadViewParityPayload) {
	t.Helper()
	if report.Snapshot.ProjectID != governanceReadViewParityProjectID {
		t.Fatalf("snapshot project = %q, want explicitly selected %q", report.Snapshot.ProjectID, governanceReadViewParityProjectID)
	}
	if report.SchemaVersion != 1 || report.Snapshot.ReadView.SchemaVersion != 1 {
		t.Fatalf("report/read-view schemas = %d/%d, want 1/1", report.SchemaVersion, report.Snapshot.ReadView.SchemaVersion)
	}
	view := report.Snapshot.ReadView
	if view.ID == "" {
		t.Fatal("CLI report omitted the operation read-view digest")
	}
	if view.Freshness != "current" {
		t.Fatalf("bound, stable fixture should have current read-view freshness, got %+v", view)
	}
	found := false
	for _, binding := range view.Bindings {
		if binding.ProjectID == governanceReadViewParityProjectID {
			found = true
			if binding.RootDigest == "" || binding.SourceRevision == "" {
				t.Errorf("project binding lacks root/source identity: %+v", binding)
			}
		}
	}
	if !found {
		t.Fatalf("read-view bindings omit explicitly selected project %q: %+v", governanceReadViewParityProjectID, view.Bindings)
	}
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
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write([]byte(info.Mode().String()))
		h.Write([]byte{0})
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
