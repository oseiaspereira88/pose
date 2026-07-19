package cli

// Release compatibility matrix contract (spec pose-release-compatibility-matrix):
// R1 machine-readable matrix of engine/schema/upgrade pairs, R2 fresh-install
// and prior-version upgrade fixtures, with actionable diagnostics for
// unsupported pairs. SemVer and repository schema compatibility are separate
// axes and are tested separately.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/version"
)

type compatMatrix struct {
	MatrixVersion     int               `json:"matrix_version"`
	EngineVersion     string            `json:"engine_version"`
	SchemaVersion     int               `json:"schema_version"`
	SupportPolicy     map[string]string `json:"support_policy"`
	SupportedUpgrades []struct {
		From            string `json:"from"`
		ChecksumsSHA256 string `json:"checksums_sha256"`
	} `json:"supported_upgrades"`
}

func loadMatrix(t *testing.T) compatMatrix {
	t.Helper()
	raw, err := os.ReadFile("../../../compatibility.json")
	if err != nil {
		t.Fatalf("reading compatibility.json: %v", err)
	}
	var m compatMatrix
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parsing compatibility.json: %v", err)
	}
	return m
}

// R1: the published matrix must agree with the candidate engine and schema.
func TestCompatibilityMatrixContract(t *testing.T) {
	m := loadMatrix(t)
	if m.MatrixVersion != 1 {
		t.Errorf("matrix_version = %d, want 1", m.MatrixVersion)
	}
	if m.EngineVersion != version.ReleaseBase() {
		t.Errorf("engine_version = %q, want authoritative release base %q", m.EngineVersion, version.ReleaseBase())
	}
	if m.SchemaVersion != nativeSchemaVersion {
		t.Errorf("schema_version = %d, want engine schema %d", m.SchemaVersion, nativeSchemaVersion)
	}
	for _, key := range []string{"window", "downgrade", "schema", "prior_artifact_authentication"} {
		if strings.TrimSpace(m.SupportPolicy[key]) == "" {
			t.Errorf("support_policy.%s is empty", key)
		}
	}
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	shaHex := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, u := range m.SupportedUpgrades {
		if !semver.MatchString(u.From) {
			t.Errorf("supported_upgrades entry %q is not a release version", u.From)
		}
		if u.From == m.EngineVersion {
			t.Errorf("supported_upgrades must list prior releases, found the candidate itself (%s)", u.From)
		}
		if !shaHex.MatchString(u.ChecksumsSHA256) {
			t.Errorf("supported_upgrades %q must pin the SHA-256 of its checksums.txt (got %q)", u.From, u.ChecksumsSHA256)
		}
	}
}

// compatInstance builds a minimal governed instance at the given schema
// version (0 = unversioned legacy layout).
func compatInstance(t *testing.T, schema string) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{".git", ".pose"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if schema != "" {
		if err := os.WriteFile(filepath.Join(dir, ".pose", "schema-version"), []byte(schema+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// R2: a legacy (unversioned) instance upgrades to the engine schema, and the
// operation is idempotent.
func TestCompatibilityUpgradeFromLegacyInstance(t *testing.T) {
	dir := compatInstance(t, "")
	inDir(t, dir, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code != 0 {
			t.Fatalf("upgrade exit=%d stderr=%s", code, errB.String())
		}
		b, err := os.ReadFile(filepath.Join(dir, ".pose", "schema-version"))
		if err != nil || strings.TrimSpace(string(b)) != "1" {
			t.Fatalf("schema-version after upgrade = %q (err=%v), want 1", b, err)
		}
		out.Reset()
		if code := Main([]string{"upgrade"}, &out, &errB); code != 0 {
			t.Fatalf("second upgrade exit=%d", code)
		}
		if !strings.Contains(out.String(), "Nothing to do") {
			t.Errorf("upgrade is not idempotent: %s", out.String())
		}
	})
}

// Unsupported pair: an instance newer than the engine fails with an
// actionable diagnostic instead of corrupting state (downgrade unsupported).
func TestCompatibilityDowngradeRejected(t *testing.T) {
	dir := compatInstance(t, "999")
	inDir(t, dir, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code == 0 {
			t.Fatal("upgrade accepted an instance newer than the engine")
		}
		if !strings.Contains(errB.String(), "downgrade is unsupported") {
			t.Errorf("diagnostic should state downgrade is unsupported, got: %s", errB.String())
		}
	})
}
