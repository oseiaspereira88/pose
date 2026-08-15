package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildTarGz packages entries (top-level-dir-relative path -> content) as a
// gzipped tar the same shape the release workflow produces:
// `tar -czf <id>.tar.gz -C extensions <id>` — every entry prefixed with
// topDir.
func buildTarGz(t *testing.T, topDir string, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for path, content := range entries {
		name := topDir + "/" + path
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildMaliciousTarGz(t *testing.T, name string, typeflag byte, linkname string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0o644, Typeflag: typeflag, Linkname: linkname}
	if typeflag == tar.TypeReg {
		hdr.Size = 0
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractExtensionTarballExtractsRealPackage(t *testing.T) {
	data := buildTarGz(t, "pose-rule-x", map[string]string{
		"extension.json":         `{"id":"pose-rule-x"}`,
		"files/.pose/rules/x.md": "rule content",
	})
	dest := t.TempDir()
	if err := extractExtensionTarball(data, dest); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dest, "extension.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "pose-rule-x") {
		t.Errorf("unexpected manifest content: %s", manifest)
	}
	rule, err := os.ReadFile(filepath.Join(dest, "files", ".pose", "rules", "x.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(rule) != "rule content" {
		t.Errorf("unexpected rule content: %s", rule)
	}
}

func TestExtractExtensionTarballRejectsPathTraversal(t *testing.T) {
	data := buildMaliciousTarGz(t, "pose-rule-x/../../../etc/passwd", tar.TypeReg, "")
	dest := t.TempDir()
	if err := extractExtensionTarball(data, dest); err == nil {
		t.Fatal("expected an error for a path-traversal tar entry, got nil")
	}
}

func TestExtractExtensionTarballRejectsSymlinks(t *testing.T) {
	data := buildMaliciousTarGz(t, "pose-rule-x/files/evil", tar.TypeSymlink, "/etc/passwd")
	dest := t.TempDir()
	if err := extractExtensionTarball(data, dest); err == nil {
		t.Fatal("expected an error for a symlink tar entry, got nil")
	}
}

func TestResolveCatalogAssetsFindsMatchingAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/oseiaspereira88/pose/releases/latest" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(catalogRelease{
			TagName: "v9.9.9",
			Assets: []catalogAsset{
				{Name: "pose-rule-x.tar.gz", BrowserDownloadURL: "https://example.invalid/pose-rule-x.tar.gz"},
				{Name: "pose-rule-x.sigstore.json", BrowserDownloadURL: "https://example.invalid/pose-rule-x.sigstore.json"},
				{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/checksums.txt"},
			},
		})
	}))
	defer srv.Close()
	origBase := catalogAPIBase
	catalogAPIBase = srv.URL
	t.Cleanup(func() { catalogAPIBase = origBase })

	tarURL, sigURL, tag, err := resolveCatalogAssets("pose-rule-x")
	if err != nil {
		t.Fatal(err)
	}
	if tarURL != "https://example.invalid/pose-rule-x.tar.gz" {
		t.Errorf("tarballURL = %q", tarURL)
	}
	if sigURL != "https://example.invalid/pose-rule-x.sigstore.json" {
		t.Errorf("sigstoreURL = %q", sigURL)
	}
	if tag != "v9.9.9" {
		t.Errorf("tag = %q", tag)
	}

	if _, _, _, err := resolveCatalogAssets("pose-rule-does-not-exist"); err == nil {
		t.Error("expected an error for an unpublished extension id, got nil")
	}
}

func TestFetchCatalogPackageDownloadsExtractsAndWritesSigstoreSibling(t *testing.T) {
	tarballData := buildTarGz(t, "pose-rule-x", map[string]string{
		"extension.json":         `{"id":"pose-rule-x"}`,
		"files/.pose/rules/x.md": "rule content",
	})
	sigstoreData := []byte(`{"fake":"sigstore-bundle"}`)

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/oseiaspereira88/pose/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		json.NewEncoder(w).Encode(catalogRelease{
			TagName: "v9.9.9",
			Assets: []catalogAsset{
				{Name: "pose-rule-x.tar.gz", BrowserDownloadURL: host + "/assets/pose-rule-x.tar.gz"},
				{Name: "pose-rule-x.sigstore.json", BrowserDownloadURL: host + "/assets/pose-rule-x.sigstore.json"},
			},
		})
	})
	mux.HandleFunc("/assets/pose-rule-x.tar.gz", func(w http.ResponseWriter, r *http.Request) { w.Write(tarballData) })
	mux.HandleFunc("/assets/pose-rule-x.sigstore.json", func(w http.ResponseWriter, r *http.Request) { w.Write(sigstoreData) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	origBase := catalogAPIBase
	catalogAPIBase = srv.URL
	t.Cleanup(func() { catalogAPIBase = origBase })

	pkgDir, cleanup, err := fetchCatalogPackage("pose-rule-x")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, err := os.Stat(filepath.Join(pkgDir, "extension.json")); err != nil {
		t.Errorf("extension.json not extracted: %v", err)
	}
	sig, err := os.ReadFile(pkgDir + ".sigstore.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(sig) != string(sigstoreData) {
		t.Errorf("sigstore bundle content = %s, want %s", sig, sigstoreData)
	}
}

func TestExtensionInstallByIDResolvesFromCatalogEndToEnd(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)

	tarballData := buildTarGz(t, "pose-rule-catalog", map[string]string{
		"extension.json":               `{"schema_version":1,"id":"pose-rule-catalog","version":"1.0.0","kind":"rule","description":"test","pose_schema_range":"1-1","files":[".pose/rules/catalog.md"],"permissions":[".pose/rules/"],"provenance":{"source":"https://example.com"}}`,
		"files/.pose/rules/catalog.md": "catalog-installed content",
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/oseiaspereira88/pose/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		json.NewEncoder(w).Encode(catalogRelease{
			TagName: "v9.9.9",
			Assets: []catalogAsset{
				{Name: "pose-rule-catalog.tar.gz", BrowserDownloadURL: host + "/assets/pose-rule-catalog.tar.gz"},
				{Name: "pose-rule-catalog.sigstore.json", BrowserDownloadURL: host + "/assets/pose-rule-catalog.sigstore.json"},
			},
		})
	})
	mux.HandleFunc("/assets/pose-rule-catalog.tar.gz", func(w http.ResponseWriter, r *http.Request) { w.Write(tarballData) })
	mux.HandleFunc("/assets/pose-rule-catalog.sigstore.json", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	origBase := catalogAPIBase
	catalogAPIBase = srv.URL
	t.Cleanup(func() { catalogAPIBase = origBase })

	code, out := runExt(t, root, "install", "pose-rule-catalog", "--yes")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".pose", "rules", "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "catalog-installed content" {
		t.Errorf("installed content = %q", installed)
	}
}

func TestExtensionInstallByIDReportsMismatchedManifestID(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)

	// Asset published under one name, manifest declares a different id —
	// must be rejected before any file is written.
	tarballData := buildTarGz(t, "pose-rule-mismatch", map[string]string{
		"extension.json":         `{"schema_version":1,"id":"pose-rule-something-else","version":"1.0.0","kind":"rule","description":"test","pose_schema_range":"1-1","files":[".pose/rules/m.md"],"permissions":[".pose/rules/"],"provenance":{"source":"https://example.com"}}`,
		"files/.pose/rules/m.md": "content",
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/oseiaspereira88/pose/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		json.NewEncoder(w).Encode(catalogRelease{
			TagName: "v9.9.9",
			Assets: []catalogAsset{
				{Name: "pose-rule-mismatch.tar.gz", BrowserDownloadURL: host + "/assets/pose-rule-mismatch.tar.gz"},
				{Name: "pose-rule-mismatch.sigstore.json", BrowserDownloadURL: host + "/assets/pose-rule-mismatch.sigstore.json"},
			},
		})
	})
	mux.HandleFunc("/assets/pose-rule-mismatch.tar.gz", func(w http.ResponseWriter, r *http.Request) { w.Write(tarballData) })
	mux.HandleFunc("/assets/pose-rule-mismatch.sigstore.json", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	origBase := catalogAPIBase
	catalogAPIBase = srv.URL
	t.Cleanup(func() { catalogAPIBase = origBase })

	code, out := runExt(t, root, "install", "pose-rule-mismatch", "--yes")
	if code == 0 {
		t.Fatalf("expected a non-zero exit for a manifest/id mismatch, got 0: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "rules", "m.md")); err == nil {
		t.Error("mismatched extension must not have written any file")
	}
}

func TestExtensionInstallLocalDirectoryPathNeverTouchesCatalog(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)
	dir := t.TempDir()
	pkg := writeExtPkg(t, dir, "pose-rule-local", "1.0.0", "rule",
		map[string]string{".pose/rules/local.md": "local content"}, nil)

	// Point the catalog at an address nothing listens on — if the local
	// directory path accidentally went through catalog resolution, this
	// install would fail with a connection error instead of succeeding.
	origBase := catalogAPIBase
	catalogAPIBase = "http://127.0.0.1:1"
	t.Cleanup(func() { catalogAPIBase = origBase })

	code, out := runExt(t, root, "install", pkg, "--yes")
	if code != 0 {
		t.Fatalf("local directory install must not touch the catalog: exit=%d out=%s", code, out)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".pose", "rules", "local.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "local content" {
		t.Errorf("installed content = %q", installed)
	}
}
