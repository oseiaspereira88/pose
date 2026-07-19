package pose

// Read-only extension discovery (spec pose-extension-catalog-lifecycle R3).
// Only the read side lives in the shared domain — install/remove are
// filesystem-mutating and stay CLI-only (architectural principle: keep file
// mutations in the execution sandbox, never expose general-purpose write
// tools over MCP). This mirrors internal/cli's lock schema without
// depending on that package (no cross-package coupling for one read path).

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ExtensionInfo is the read-only projection of one installed extension.
type ExtensionInfo struct {
	ID                string   `json:"id"`
	Version           string   `json:"version"`
	Kind              string   `json:"kind"`
	InstalledAt       string   `json:"installed_at"`
	Digest            string   `json:"digest"`
	Files             []string `json:"files"`
	SignatureVerified bool     `json:"signature_verified"`
}

// ListExtensions reads .pose/indexes/extensions.lock.json. A missing lock
// file means no extensions are installed (empty slice, not an error).
func (s Store) ListExtensions() ([]ExtensionInfo, error) {
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "extensions.lock.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return []ExtensionInfo{}, nil
		}
		return nil, err
	}
	var lock struct {
		Extensions map[string]struct {
			Version           string            `json:"version"`
			Kind              string            `json:"kind"`
			InstalledAt       string            `json:"installed_at"`
			Digest            string            `json:"digest"`
			Files             map[string]string `json:"files"`
			SignatureVerified bool              `json:"signature_verified"`
		} `json:"extensions"`
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, err
	}
	out := make([]ExtensionInfo, 0, len(lock.Extensions))
	for id, e := range lock.Extensions {
		files := make([]string, 0, len(e.Files))
		for f := range e.Files {
			files = append(files, f)
		}
		out = append(out, ExtensionInfo{
			ID: id, Version: e.Version, Kind: e.Kind, InstalledAt: e.InstalledAt,
			Digest: e.Digest, Files: files, SignatureVerified: e.SignatureVerified,
		})
	}
	return out, nil
}
