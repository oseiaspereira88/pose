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

// spec pose-public-install-contract: the published quickstart and CI docs pin
// the current release base, reference real release coordinates (no
// placeholders) and agree with the GoReleaser asset naming template.
func TestPublicInstallContract(t *testing.T) {
	base := version.ReleaseBase()
	readme, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	rd := string(readme)
	for _, want := range []string{
		"V=" + base + "\n",
		`$V = "` + base + `"`,
		"https://github.com/oseiaspereira88/pose/releases/download/v${V}/pose_${V}_linux_amd64.tar.gz",
		"checksums.txt",
	} {
		if !strings.Contains(rd, want) {
			t.Errorf("README quickstart missing %q (asset naming or pinned version drifted)", want)
		}
	}
	gorel, err := os.ReadFile("../../../.goreleaser.yaml")
	if err != nil {
		t.Fatalf("reading .goreleaser.yaml: %v", err)
	}
	gr := string(gorel)
	for _, want := range []string{
		`name_template: "pose_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`,
		"goos: windows",
		"formats: [zip]",
	} {
		if !strings.Contains(gr, want) {
			t.Errorf(".goreleaser.yaml missing %q (README asset commands would break)", want)
		}
	}
	for _, doc := range []string{"../../../docs-site/docs/ci.md", "../../../pose-action/README.md"} {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("reading %s: %v", doc, err)
		}
		if strings.Contains(string(raw), "<owner>") || strings.Contains(string(raw), "<repo>") {
			t.Errorf("%s still contains owner/repo placeholders", doc)
		}
	}
	ci, _ := os.ReadFile("../../../docs-site/docs/ci.md")
	if !strings.Contains(string(ci), "rev: v"+base) {
		t.Errorf("docs-site/docs/ci.md pre-commit rev is not pinned to v%s", base)
	}
}

// specs pose-release-signing / pose-cyclonedx-sbom: the release pipeline
// signs every artifact with offline-verifiable Sigstore bundles, publishes a
// CycloneDX SBOM per archive, and the workflow verifies both against the
// pinned identity before the run may succeed.
func TestArtifactIdentityContract(t *testing.T) {
	gorel, err := os.ReadFile("../../../.goreleaser.yaml")
	if err != nil {
		t.Fatalf("reading .goreleaser.yaml: %v", err)
	}
	gr := string(gorel)
	for _, want := range []string{
		`signature: "${artifact}.sigstore.json"`,
		`"sign-blob", "--yes", "--bundle", "${signature}"`,
		"artifacts: all",
		`documents: ["${artifact}.cdx.json"]`,
		`"cyclonedx-json=${document}"`,
	} {
		if !strings.Contains(gr, want) {
			t.Errorf(".goreleaser.yaml missing %q (signing/SBOM contract)", want)
		}
	}
	rel, err := os.ReadFile("../../../.github/workflows/release.yml")
	if err != nil {
		t.Fatalf("reading release.yml: %v", err)
	}
	rw := string(rel)
	for _, want := range []string{
		"id-token: write",
		"sigstore/cosign-installer@",
		"anchore/sbom-action/download-syft@",
		"tests/release/verify.sh",
	} {
		if !strings.Contains(rw, want) {
			t.Errorf("release.yml missing %q (identity verification gate)", want)
		}
	}
	sec, err := os.ReadFile("../../../SECURITY.md")
	if err != nil {
		t.Fatalf("reading SECURITY.md: %v", err)
	}
	sm := string(sec)
	for _, want := range []string{
		"cosign verify-blob",
		"--certificate-oidc-issuer https://token.actions.githubusercontent.com",
		`--certificate-identity-regexp`,
		"release\\.yml@refs/tags/v",
	} {
		if !strings.Contains(sm, want) {
			t.Errorf("SECURITY.md missing %q (pinned verification instructions)", want)
		}
	}
}

// specs pose-slsa-provenance / pose-reproducible-release-verification: every
// archive is a provenance subject, the build is reproducible by
// configuration, and an independent consumer-side workflow verifies
// signature, provenance, checksum and SBOM before executing anything.
func TestAttestedReleaseContract(t *testing.T) {
	rel, err := os.ReadFile("../../../.github/workflows/release.yml")
	if err != nil {
		t.Fatalf("reading release.yml: %v", err)
	}
	rw := string(rel)
	for _, want := range []string{
		"attestations: write",
		"actions/attest-build-provenance@",
		"dist-release/*.tar.gz",
		"dist-release/checksums.txt",
	} {
		if !strings.Contains(rw, want) {
			t.Errorf("release.yml missing %q (provenance contract)", want)
		}
	}
	gorel, _ := os.ReadFile("../../../.goreleaser.yaml")
	for _, want := range []string{
		"flags: [-trimpath]",
		`mod_timestamp: "{{ .CommitTimestamp }}"`,
	} {
		if !strings.Contains(string(gorel), want) {
			t.Errorf(".goreleaser.yaml missing %q (reproducible build inputs)", want)
		}
	}
	ver, err := os.ReadFile("../../../.github/workflows/verify-release.yml")
	if err != nil {
		t.Fatalf("reading verify-release.yml (independent verifier): %v", err)
	}
	vw := string(ver)
	for _, want := range []string{
		"release:",
		"workflow_dispatch:",
		"cache: false",
		"tests/release/independent-verify.sh",
		"permissions: { contents: read }",
	} {
		if !strings.Contains(vw, want) {
			t.Errorf("verify-release.yml missing %q (verifier isolation contract)", want)
		}
	}
	sec, _ := os.ReadFile("../../../SECURITY.md")
	for _, want := range []string{
		"gh attestation verify",
		"--signer-workflow",
	} {
		if !strings.Contains(string(sec), want) {
			t.Errorf("SECURITY.md missing %q (provenance verification instructions)", want)
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
