package cli

// Public claims contract behavior (spec pose-public-claims-contract).
//
// The cases that matter here are the negative ones. A gate that only proves
// it passes on clean input is exactly the failure mode this command exists to
// correct: the site's previous version gate passed for weeks while pinning a
// version the product had left behind.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeClaimsFixture(t *testing.T, root string, surfaces string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{
  "schema_version": 1,
  "product": {"name": "POSE", "docs_canonical": "https://docs.harne8.com/POSE/"},
  "version_source": "compatibility.json",
  "docs_hosts": {
    "canonical": "https://docs.harne8.com/POSE/",
    "deprecated": ["oseiaspereira88.github.io/pose"]
  },
  "surfaces": [` + surfaces + `]
}`
	if err := os.WriteFile(filepath.Join(root, ".pose", "public", "claims.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "compatibility.json"), []byte(`{"engine_version":"1.7.10"}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSurface(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runClaims(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdPublicClaims(root, args, &out, &errB)
	return code, out.String() + errB.String()
}

func TestPublicClaimsPassesOnCompliantSurfaces(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"},
	 {"path":"docs.md","version_claims":"none"}`)
	writeSurface(t, root, "README.md", "Install POSE:\n\nV=1.7.10\ncurl .../releases/download/v1.7.10/pose.tar.gz\n\nDocs: https://docs.harne8.com/POSE/\n")
	writeSurface(t, root, "docs.md", "POSE is an SDD framework. See https://docs.harne8.com/POSE/ for more.\n")

	code, out := runClaims(t, root, "--strict")
	if code != 0 {
		t.Fatalf("expected pass, got exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "public-claims.released_version=1.7.10") {
		t.Errorf("released version not reported: %s", out)
	}
}

func TestPublicClaimsFailsOnStaleVersion(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"}`)
	// The exact defect found in README.pt-BR.md: an install block pinning a
	// release four minors behind, so the reader installs the wrong binary.
	writeSurface(t, root, "README.md", "V=1.4.3\ncurl .../releases/download/v1.4.3/pose.tar.gz\n")

	code, out := runClaims(t, root, "--strict")
	if code == 0 {
		t.Fatalf("stale version must fail the gate; out=%s", out)
	}
	if !strings.Contains(out, "1.4.3") || !strings.Contains(out, "1.7.10") {
		t.Errorf("message should name both the claim and the released version: %s", out)
	}
}

func TestPublicClaimsFailsWhenEvergreenSurfaceGainsAVersion(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"landing.md","version_claims":"none"}`)
	// Even the *current* version is a failure here. The point of declaring a
	// surface evergreen is that the claim must not exist at all — otherwise it
	// silently becomes stale on the next release.
	writeSurface(t, root, "landing.md", "POSE 1.7.10 · Apache-2.0\n")

	code, out := runClaims(t, root, "--strict")
	if code == 0 {
		t.Fatalf("an evergreen surface naming any version must fail; out=%s", out)
	}
	if !strings.Contains(out, "evergreen") {
		t.Errorf("message should explain the evergreen contract: %s", out)
	}
}

func TestPublicClaimsFailsOnSchemaOrgVersionAndNonCanonicalDocs(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"landing.md","version_claims":"none"}`)
	// Both real defects from the audit in one surface: a hardcoded
	// softwareVersion in structured metadata, and a link to the second live
	// documentation site.
	writeSurface(t, root, "landing.md", `{"softwareVersion": "1.4.3"}
<a href="https://oseiaspereira88.github.io/pose/">Docs</a>
`)

	code, out := runClaims(t, root, "--strict")
	if code == 0 {
		t.Fatalf("expected failure; out=%s", out)
	}
	if !strings.Contains(out, "docs-route") {
		t.Errorf("non-canonical docs host must be reported: %s", out)
	}
	if !strings.Contains(out, "version") {
		t.Errorf("softwareVersion must be reported: %s", out)
	}
}

func TestPublicClaimsAcceptsReleaseLineButNotAnotherPatch(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"a.md","version_claims":"current-only"}`)
	// "POSE v1.7" names the release line and is satisfied by 1.7.10; dropping
	// a digit is a deliberate choice, not drift.
	writeSurface(t, root, "a.md", "POSE v1.7 is current.\n")
	if code, out := runClaims(t, root, "--strict"); code != 0 {
		t.Fatalf("release-line claim should pass; exit=%d out=%s", code, out)
	}

	writeSurface(t, root, "a.md", "POSE v1.7.9 is current.\n")
	if code, out := runClaims(t, root, "--strict"); code == 0 {
		t.Fatalf("a different patch must fail; out=%s", out)
	}
}

func TestPublicClaimsTolerantModeReportsWithoutFailing(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"a.md","version_claims":"none"}`)
	writeSurface(t, root, "a.md", "POSE 1.4.3\n")

	code, out := runClaims(t, root, "--tolerant")
	if code != 0 {
		t.Fatalf("tolerant mode must not fail the build; exit=%d", code)
	}
	if !strings.Contains(out, "TOLERATED_FAILURE") {
		t.Errorf("tolerant mode should still report the finding: %s", out)
	}
}

func TestPublicClaimsJSONOutputIsMachineReadable(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"a.md","version_claims":"none"}`)
	writeSurface(t, root, "a.md", "POSE 1.4.3\n")

	var out, errB bytes.Buffer
	code := cmdPublicClaims(root, []string{"--strict", "--json"}, &out, &errB)
	if code == 0 {
		t.Fatalf("expected failure exit")
	}
	var payload struct {
		Release  string `json:"released_version"`
		Surfaces int    `json:"surfaces_checked"`
		Findings []struct {
			Surface string `json:"surface"`
			Claim   string `json:"claim"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if payload.Release != "1.7.10" || payload.Surfaces != 1 || len(payload.Findings) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Findings[0].Surface != "a.md" || payload.Findings[0].Claim != "version" {
		t.Errorf("finding not attributed: %+v", payload.Findings[0])
	}
}

func TestPublicClaimsFailsOnMissingSurfaceAndUnknownMode(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"gone.md","version_claims":"none"},
	 {"path":"b.md","version_claims":"whatever"}`)
	writeSurface(t, root, "b.md", "no version here\n")

	code, out := runClaims(t, root, "--strict")
	if code == 0 {
		t.Fatalf("expected failure; out=%s", out)
	}
	// A declared surface that no longer exists is drift too: the contract
	// claims coverage it does not have.
	if !strings.Contains(out, "unreadable") {
		t.Errorf("missing surface must be reported: %s", out)
	}
	if !strings.Contains(out, "unknown version_claims") {
		t.Errorf("an unknown mode must not silently pass: %s", out)
	}
}

// The command must fail loudly rather than pass vacuously when its own inputs
// are absent — a gate that silently succeeds without a contract is worse than
// no gate.
func TestPublicClaimsFailsWithoutContractOrVersionSource(t *testing.T) {
	root := t.TempDir()
	if code, _ := runClaims(t, root, "--strict"); code != 2 {
		t.Errorf("missing contract should exit 2, got %d", code)
	}

	root2 := t.TempDir()
	writeClaimsFixture(t, root2, `{"path":"a.md","version_claims":"none"}`)
	writeSurface(t, root2, "a.md", "ok\n")
	if err := os.Remove(filepath.Join(root2, "compatibility.json")); err != nil {
		t.Fatal(err)
	}
	if code, _ := runClaims(t, root2, "--strict"); code != 2 {
		t.Errorf("missing version source should exit 2, got %d", code)
	}
}
