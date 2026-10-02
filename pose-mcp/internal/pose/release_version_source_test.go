package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSourceFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseVersionSourceReadsTextAndJSON(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name   string
		source ReleaseVersionSource
		file   string
		body   string
		want   string
	}{
		{"bare text", ReleaseVersionSource{Path: "VERSION", Kind: "text"}, "VERSION", "0.2.0\n", "v0.2.0"},
		{"prefixed text", ReleaseVersionSource{Path: "VERSION", Kind: "text"}, "VERSION", "v1.4.10", "v1.4.10"},
		{"development suffix", ReleaseVersionSource{Path: "VERSION", Kind: "text"}, "VERSION", "0.3.0-dev\n", "v0.3.0"},
		{"json key", ReleaseVersionSource{Path: "app/package.json", Kind: "json", Key: "version"}, "app/package.json", `{"name":"x","version":"2.0.1","private":true}`, "v2.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeSourceFile(t, root, c.file, c.body)
			got, err := ReadReleaseVersionSource(root, c.source)
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestReleaseVersionSourceRejectsUnsafeOrMalformedSources(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeSourceFile(t, outside, "VERSION", "9.9.9\n")
	if err := os.Symlink(filepath.Join(outside, "VERSION"), filepath.Join(root, "LINK")); err != nil {
		t.Skip("symlinks unavailable")
	}
	writeSourceFile(t, root, "MULTI", "1.0.0\n2.0.0\n")
	writeSourceFile(t, root, "WORD", "next\n")
	writeSourceFile(t, root, "PRE", "1.0.0-rc.1\n")
	writeSourceFile(t, root, "BIG", strings.Repeat("1", releaseVersionTextLimit+1))
	writeSourceFile(t, root, "obj.json", `{"version":3,"other":"1.0.0"}`)
	writeSourceFile(t, root, "list.json", `[1]`)
	if err := os.MkdirAll(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		source ReleaseVersionSource
		want   string
	}{
		{"escape", ReleaseVersionSource{Path: "../VERSION", Kind: "text"}, "escape"},
		{"symlink out", ReleaseVersionSource{Path: "LINK", Kind: "text"}, "symlink escape"},
		{"absolute", ReleaseVersionSource{Path: filepath.Join(outside, "VERSION"), Kind: "text"}, "project-relative"},
		{"missing", ReleaseVersionSource{Path: "NOPE", Kind: "text"}, "file not found"},
		{"directory", ReleaseVersionSource{Path: "adir", Kind: "text"}, "not a regular file"},
		{"oversized", ReleaseVersionSource{Path: "BIG", Kind: "text"}, "exceeds"},
		{"two lines", ReleaseVersionSource{Path: "MULTI", Kind: "text"}, "single line"},
		{"not a version", ReleaseVersionSource{Path: "WORD", Kind: "text"}, "semantic version"},
		{"prerelease", ReleaseVersionSource{Path: "PRE", Kind: "text"}, "semantic version"},
		{"key absent", ReleaseVersionSource{Path: "obj.json", Kind: "json", Key: "missing"}, "is absent"},
		{"key not string", ReleaseVersionSource{Path: "obj.json", Kind: "json", Key: "version"}, "not a string"},
		{"not an object", ReleaseVersionSource{Path: "list.json", Kind: "json", Key: "version"}, "not a JSON object"},
		{"json without key", ReleaseVersionSource{Path: "obj.json", Kind: "json"}, "key is required"},
		{"text with key", ReleaseVersionSource{Path: "WORD", Kind: "text", Key: "version"}, "only valid for kind"},
		{"unknown kind", ReleaseVersionSource{Path: "WORD", Kind: "toml"}, "kind must be"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ReadReleaseVersionSource(root, c.source)
			if err == nil || got != "" || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %q, %v; want an error containing %q", got, err, c.want)
			}
			if strings.Contains(err.Error(), outside) || strings.Contains(err.Error(), root) {
				t.Fatalf("error leaks an absolute path: %v", err)
			}
		})
	}
}

func TestReleasePolicyOmitsAnUnsetVersionSource(t *testing.T) {
	// The policy digest recorded in every existing manifest is taken over this
	// serialization, so an unset source must not appear in it.
	policy := ReleasePolicy{SchemaVersion: 1, AdoptedAt: "2026-08-03", Provider: "github", Repository: "owner/repo"}
	if strings.Contains(string(CanonicalJSON(policy)), "version_source") {
		t.Fatal("unset version_source changed the serialized policy")
	}
	policy.VersionSource = &ReleaseVersionSource{Path: "VERSION", Kind: "text"}
	if !strings.Contains(string(CanonicalJSON(policy)), "version_source") {
		t.Fatal("declared version_source is not serialized")
	}
	if ReleaseDigest(ReleasePolicy{SchemaVersion: 1, AdoptedAt: "2026-08-03", Provider: "github", Repository: "owner/repo"}) == ReleaseDigest(policy) {
		t.Fatal("declaring a source must change the policy digest")
	}
}

func TestLoadReleasePolicyValidatesTheVersionSource(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"2026-10-02","provider":"github","repository":"o/r","version_source":{"path":"VERSION","kind":"yaml"}}`)
	if _, err := LoadReleasePolicy(root); err == nil || !strings.Contains(err.Error(), "version_source") {
		t.Fatalf("an invalid declaration must be refused by name: %v", err)
	}
	writeSourceFile(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"2026-10-02","provider":"github","repository":"o/r","version_source":{"path":"VERSION","kind":"text"}}`)
	policy, err := LoadReleasePolicy(root)
	if err != nil || policy.VersionSource == nil || policy.VersionSource.Path != "VERSION" {
		t.Fatalf("policy=%+v err=%v", policy, err)
	}
}
