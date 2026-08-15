package cli

// Extension catalog resolution (spec pose-extension-catalog-resolution):
// `pose extension install <id>` resolves an extension ID to the current
// published GitHub release's matching signed asset pair
// (<id>.tar.gz + <id>.sigstore.json), downloads and safely extracts the
// tarball into a fresh temp directory, and hands off to the exact same
// local-package install pipeline `pose extension install <dir>` already
// used. Signature verification (verifyExtensionSignature), permission and
// whitelist checks are entirely unchanged — this only produces a local,
// extracted package directory the existing pipeline already knows how to
// consume.
//
// Only the latest published release is consulted; an extension not present
// there is reported as not found rather than searched for across release
// history (documented scope limit).

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// catalogAPIBase and catalogRepo are package-level vars so tests point them
// at a local httptest server / fixture repo instead of the real network.
var catalogAPIBase = "https://api.github.com"
var catalogRepo = "oseiaspereira88/pose"

// catalogHTTPClient is used for every catalog network call.
var catalogHTTPClient = &http.Client{Timeout: 30 * time.Second}

type catalogAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type catalogRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []catalogAsset `json:"assets"`
}

// resolveCatalogAssets finds the tarball and Sigstore bundle download URLs
// for id in the latest published release.
func resolveCatalogAssets(id string) (tarballURL, sigstoreURL, tag string, err error) {
	url := strings.TrimRight(catalogAPIBase, "/") + "/repos/" + catalogRepo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := catalogHTTPClient.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("resolving catalog release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("resolving catalog release: unexpected status %s", resp.Status)
	}
	var rel catalogRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", "", fmt.Errorf("decoding catalog release: %w", err)
	}
	byName := map[string]string{}
	for _, a := range rel.Assets {
		byName[a.Name] = a.BrowserDownloadURL
	}
	tarballURL, ok := byName[id+".tar.gz"]
	if !ok {
		return "", "", "", fmt.Errorf("extension %q not found in the latest published release (%s)", id, rel.TagName)
	}
	sigstoreURL, ok = byName[id+".sigstore.json"]
	if !ok {
		return "", "", "", fmt.Errorf("extension %q has no published signature bundle in release %s", id, rel.TagName)
	}
	return tarballURL, sigstoreURL, rel.TagName, nil
}

func downloadCatalogAsset(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := catalogHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: unexpected status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// extractExtensionTarball safely extracts a .tar.gz whose entries are all
// prefixed with one top-level directory name (matching the release
// workflow's `tar -czf <id>.tar.gz -C extensions <id>` layout) into destDir,
// stripping that one leading path component. Rejects any entry that would
// escape destDir (path traversal) or that is a symlink/hardlink — an
// actively enforced version of the package format's existing
// "symlink-free directory" trust model, since this content crossed a
// network boundary instead of already sitting reviewed on disk.
func extractExtensionTarball(data []byte, destDir string) error {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("not a valid gzip archive: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(header.Name)))
		parts := strings.SplitN(clean, "/", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue // the top-level directory entry itself
		}
		rel := parts[1]
		if !confinedRelativePath(rel) {
			return fmt.Errorf("tar entry escapes the extraction directory: %s", header.Name)
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("tar entry is a symlink, rejected: %s", header.Name)
		default:
			// Skip anything else (char/block devices, fifos, ...) rather
			// than fail the whole install over an entry no legitimate
			// extension package would ever contain.
		}
	}
	return nil
}

// fetchCatalogPackage resolves id to its published release assets,
// downloads both, and extracts the tarball into a fresh temp directory
// alongside its Sigstore bundle at the sibling path
// verifyExtensionSignature already expects (pkgDir + ".sigstore.json").
// The returned cleanup removes the temp directory; callers must defer it.
func fetchCatalogPackage(id string) (pkgDir string, cleanup func(), err error) {
	tarballURL, sigstoreURL, _, err := resolveCatalogAssets(id)
	if err != nil {
		return "", nil, err
	}
	tmpRoot, err := os.MkdirTemp("", "pose-extension-catalog-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpRoot) }
	pkgDir = filepath.Join(tmpRoot, id)
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		cleanup()
		return "", nil, err
	}
	tarballData, err := downloadCatalogAsset(tarballURL)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if err := extractExtensionTarball(tarballData, pkgDir); err != nil {
		cleanup()
		return "", nil, err
	}
	sigstoreData, err := downloadCatalogAsset(sigstoreURL)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if err := os.WriteFile(pkgDir+".sigstore.json", sigstoreData, 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return pkgDir, cleanup, nil
}
