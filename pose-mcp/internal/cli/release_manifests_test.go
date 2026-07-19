package cli

// Deterministic package-manager manifest generation (spec
// pose-package-manager-distribution): checksums.txt parsing and exact
// Homebrew formula / WinGet manifest content, runnable locally without
// brew/winget (the clean-host install matrix itself runs in CI — see
// .github/workflows/package-channels.yml).

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleChecksums() string {
	var b strings.Builder
	for _, asset := range []string{
		"pose_1.2.3_darwin_arm64.tar.gz",
		"pose_1.2.3_darwin_amd64.tar.gz",
		"pose_1.2.3_linux_arm64.tar.gz",
		"pose_1.2.3_linux_amd64.tar.gz",
		"pose_1.2.3_windows_amd64.zip",
		"pose_1.2.3_windows_arm64.zip",
	} {
		fakeSHA := strings.Repeat("a", 63) + string(rune('0'+len(asset)%10))
		b.WriteString(fakeSHA + "  " + asset + "\n")
	}
	return b.String()
}

func TestParseChecksumsValid(t *testing.T) {
	sums, err := parseChecksums(sampleChecksums())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sums) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(sums))
	}
	if _, ok := sums["pose_1.2.3_darwin_arm64.tar.gz"]; !ok {
		t.Errorf("missing darwin/arm64 entry")
	}
}

func TestParseChecksumsIgnoresBlankLines(t *testing.T) {
	raw := "\n" + sampleChecksums() + "\n\n"
	sums, err := parseChecksums(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sums) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(sums))
	}
}

func TestParseChecksumsRejectsMalformedLine(t *testing.T) {
	_, err := parseChecksums("not-a-valid-line\n")
	if err == nil {
		t.Fatal("expected error for malformed checksums line")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Errorf("expected line-numbered diagnostic: %v", err)
	}
}

func TestHomebrewFormulaDeterministicContent(t *testing.T) {
	sums, err := parseChecksums(sampleChecksums())
	if err != nil {
		t.Fatal(err)
	}
	formula, err := homebrewFormula("1.2.3", sums)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		`class Pose < Formula`,
		`version "1.2.3"`,
		`on_macos do`,
		`on_linux do`,
		`Hardware::CPU.arm?`,
		`Hardware::CPU.intel?`,
		`https://github.com/oseiaspereira88/pose/releases/download/v1.2.3/pose_1.2.3_darwin_arm64.tar.gz`,
		`bin.install "pose"`,
		`assert_match version.to_s, shell_output("#{bin}/pose version")`,
	} {
		if !strings.Contains(formula, want) {
			t.Errorf("formula missing %q:\n%s", want, formula)
		}
	}

	again, err := homebrewFormula("1.2.3", sums)
	if err != nil {
		t.Fatal(err)
	}
	if formula != again {
		t.Error("homebrewFormula must be deterministic across repeated calls")
	}
}

func TestHomebrewFormulaMissingChecksumFails(t *testing.T) {
	sums, err := parseChecksums("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := homebrewFormula("1.2.3", sums); err == nil {
		t.Fatal("expected error when release asset checksums are missing")
	}
}

func TestWinGetManifestsDeterministicContent(t *testing.T) {
	sums, err := parseChecksums(sampleChecksums())
	if err != nil {
		t.Fatal(err)
	}
	manifests, err := winGetManifests("1.2.3", sums)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 3 {
		t.Fatalf("expected 3 manifest files, got %d", len(manifests))
	}
	version := manifests["Harne8.Pose.yaml"]
	for _, want := range []string{"PackageIdentifier: Harne8.Pose", "PackageVersion: 1.2.3", "ManifestType: version"} {
		if !strings.Contains(version, want) {
			t.Errorf("version manifest missing %q:\n%s", want, version)
		}
	}
	installer := manifests["Harne8.Pose.installer.yaml"]
	for _, want := range []string{"Architecture: x64", "Architecture: arm64", "InstallerType: zip", "ManifestType: installer"} {
		if !strings.Contains(installer, want) {
			t.Errorf("installer manifest missing %q:\n%s", want, installer)
		}
	}
	locale := manifests["Harne8.Pose.locale.en-US.yaml"]
	if !strings.Contains(locale, "ManifestType: defaultLocale") {
		t.Errorf("locale manifest missing ManifestType: %s", locale)
	}

	again, err := winGetManifests("1.2.3", sums)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range manifests {
		if again[name] != content {
			t.Errorf("winGetManifests must be deterministic for %s", name)
		}
	}
}

func TestWinGetManifestsMissingChecksumFails(t *testing.T) {
	sums, err := parseChecksums("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := winGetManifests("1.2.3", sums); err == nil {
		t.Fatal("expected error when Windows asset checksums are missing")
	}
}

func TestCmdReleasePackageManifestsWritesFiles(t *testing.T) {
	dir := t.TempDir()
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(sampleChecksums()), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	var out, errB bytes.Buffer
	code := cmdReleasePackageManifests([]string{"--version", "1.2.3", "--checksums", checksumsPath, "--out", outDir}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	if !strings.Contains(out.String(), "Result: SUCCESS") {
		t.Errorf("expected success marker: %s", out.String())
	}
	for _, p := range []string{
		filepath.Join(outDir, "homebrew", "pose.rb"),
		filepath.Join(outDir, "winget", "Harne8.Pose.yaml"),
		filepath.Join(outDir, "winget", "Harne8.Pose.installer.yaml"),
		filepath.Join(outDir, "winget", "Harne8.Pose.locale.en-US.yaml"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected generated file %s: %v", p, err)
		}
	}
}

func TestCmdReleasePackageManifestsRejectsMalformedVersion(t *testing.T) {
	dir := t.TempDir()
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(sampleChecksums()), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	code := cmdReleasePackageManifests([]string{"--version", "not-semver", "--checksums", checksumsPath, "--out", filepath.Join(dir, "out")}, &out, &errB)
	if code == 0 {
		t.Fatal("expected non-zero exit for malformed version")
	}
	if !strings.Contains(errB.String(), "X.Y.Z") {
		t.Errorf("expected version-format diagnostic: %s", errB.String())
	}
}

func TestCmdReleasePackageManifestsMissingChecksumsFile(t *testing.T) {
	dir := t.TempDir()
	var out, errB bytes.Buffer
	code := cmdReleasePackageManifests([]string{"--version", "1.2.3", "--checksums", filepath.Join(dir, "missing.txt"), "--out", filepath.Join(dir, "out")}, &out, &errB)
	if code == 0 {
		t.Fatal("expected non-zero exit for missing checksums file")
	}
}

func TestCmdReleasePackageManifestsUsageError(t *testing.T) {
	var out, errB bytes.Buffer
	code := cmdReleasePackageManifests(nil, &out, &errB)
	if code != 2 {
		t.Fatalf("expected usage error exit=2, got %d", code)
	}
	if !strings.Contains(errB.String(), "Usage:") {
		t.Errorf("expected usage message: %s", errB.String())
	}
}
