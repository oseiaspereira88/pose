package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func writeDocsSyncManifest(t *testing.T, root string) {
	t.Helper()
	writeCLIDocsFixture(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))
}

func TestDocsSyncExport_WritesBundleToStdout(t *testing.T) {
	root := t.TempDir()
	writeDocsSyncManifest(t, root)
	var stdout, stderr bytes.Buffer
	if code := docsSyncExport(root, []string{"--project", "proj.demo"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	var bundle pose.DocsSyncBundle
	if err := json.Unmarshal(stdout.Bytes(), &bundle); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, stdout.String())
	}
	if len(bundle.Docs) != 1 || bundle.ProjectID != "proj.demo" {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
}

func TestDocsSyncExport_NoManifestErrors(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := docsSyncExport(root, nil, &stdout, &stderr, localeEN); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestDocsSyncExport_WritesToFile(t *testing.T) {
	root := t.TempDir()
	writeDocsSyncManifest(t, root)
	out := filepath.Join(root, "bundle.json")
	var stdout, stderr bytes.Buffer
	if code := docsSyncExport(root, []string{"--out", out}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected bundle file to exist: %v", err)
	}
}

func TestDocsSyncPush_PutsBundleToConductorEndpoint(t *testing.T) {
	root := t.TempDir()
	writeDocsSyncManifest(t, root)

	var gotMethod, gotPath, gotAuth string
	var gotBody pose.DocsSyncBundle
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := docsSyncPush(root, []string{"--url", srv.URL, "--project", "proj.demo", "--token", "secret"}, &stdout, &stderr, localeEN)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	if gotMethod != http.MethodPut || gotPath != "/api/v1/projects/proj.demo/wiki/bundle" {
		t.Fatalf("unexpected request: method=%s path=%s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("expected bearer token, got %q", gotAuth)
	}
	if len(gotBody.Docs) != 1 {
		t.Fatalf("expected bundle body with 1 doc, got %+v", gotBody)
	}

	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "refresh-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"consumer":"docs-sync"`)) || !bytes.Contains(raw, []byte(`"result":"ok"`)) {
		t.Fatalf("expected a successful docs-sync refresh-log entry, got %s", raw)
	}
}

func TestDocsSyncPush_FailureIsVisibleInRefreshLog(t *testing.T) {
	root := t.TempDir()
	writeDocsSyncManifest(t, root)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := docsSyncPush(root, []string{"--url", srv.URL, "--project", "proj.demo"}, &stdout, &stderr, localeEN)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "refresh-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"consumer":"docs-sync"`)) || !bytes.Contains(raw, []byte(`"result":"failed"`)) {
		t.Fatalf("expected a failed docs-sync refresh-log entry, got %s", raw)
	}
}

func TestDocsSyncPush_RequiresURLAndProject(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := docsSyncPush(root, nil, &stdout, &stderr, localeEN); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}
