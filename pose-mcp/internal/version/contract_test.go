// Contract tests for the authoritative version source (spec
// pose-version-contract): every public version surface must derive from
// internal/version, development builds must be explicit, and release
// metadata checked into the repository must agree with the release base.
package version_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli"
	"github.com/harne8/pose-mcp/internal/version"
)

var semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.-]+)?$`)

// R2: a development build exposes an explicit development identifier and
// never impersonates a release.
func TestDevelopmentVersionPolicy(t *testing.T) {
	if !semverRe.MatchString(version.Version) {
		t.Fatalf("Version %q is not SemVer", version.Version)
	}
	if version.IsDevelopment() {
		if !strings.HasSuffix(version.Version, "-dev") {
			t.Fatalf("development build must end in -dev, got %q", version.Version)
		}
		if version.ReleaseBase() == version.Version {
			t.Fatalf("ReleaseBase() must strip the -dev suffix, got %q", version.ReleaseBase())
		}
	}
	if !semverRe.MatchString(version.ReleaseBase()) {
		t.Fatalf("ReleaseBase() %q is not SemVer", version.ReleaseBase())
	}
}

// R1/R3: the CLI surface reports exactly the authoritative version.
func TestCLIVersionMatchesAuthority(t *testing.T) {
	if cli.Version != version.Version {
		t.Fatalf("cli.Version = %q, authority = %q", cli.Version, version.Version)
	}
}

// R3: the checked-in MCP registry metadata (server.json) carries the release
// base of the authoritative version on every version field.
func TestRegistryMetadataMatchesReleaseBase(t *testing.T) {
	raw, err := os.ReadFile("../../server.json")
	if err != nil {
		t.Fatalf("reading server.json: %v", err)
	}
	var doc struct {
		VersionDetail struct {
			Version string `json:"version"`
		} `json:"version_detail"`
		Packages []struct {
			Identifier string `json:"identifier"`
			Version    string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing server.json: %v", err)
	}
	base := version.ReleaseBase()
	if doc.VersionDetail.Version != base {
		t.Errorf("server.json version_detail.version = %q, want %q", doc.VersionDetail.Version, base)
	}
	if len(doc.Packages) == 0 {
		t.Fatal("server.json declares no packages")
	}
	for _, p := range doc.Packages {
		if p.Version != base {
			t.Errorf("server.json package %q version = %q, want %q", p.Identifier, p.Version, base)
		}
	}
}

// R1/R3: the release pipeline stamps the authoritative symbol — and only it.
func TestReleasePipelineStampsAuthority(t *testing.T) {
	raw, err := os.ReadFile("../../../.goreleaser.yaml")
	if err != nil {
		t.Fatalf("reading .goreleaser.yaml: %v", err)
	}
	cfg := string(raw)
	const want = "-X github.com/harne8/pose-mcp/internal/version.Version={{ .Version }}"
	if !strings.Contains(cfg, want) {
		t.Fatalf(".goreleaser.yaml does not stamp the authoritative symbol %q", want)
	}
	for _, line := range strings.Split(cfg, "\n") {
		if strings.Contains(line, "-X ") && !strings.Contains(line, "internal/version.Version") {
			t.Errorf("ldflags stamps a non-authoritative symbol: %s", strings.TrimSpace(line))
		}
	}
}
