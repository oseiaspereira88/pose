package cli

// Supported package-manager distribution (spec pose-package-manager-distribution):
// generate a Homebrew formula and a WinGet manifest set deterministically
// from the same verified release metadata every other channel uses —
// compatibility.json's engine_version and the release's checksums.txt.
// Never runs before the release compatibility gate: this generator is a
// release-pipeline step wired in after `tests/release/compat.sh` succeeds
// (R2 — metadata updates only once verification passes).

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const winGetManifestVersion = "1.6.0"
const winGetPackageIdentifier = "Harne8.Pose"
const homebrewClass = "Pose"
const releaseRepo = "oseiaspereira88/pose"

// releaseChecksums maps release archive filename -> lowercase hex sha256,
// parsed from GoReleaser's checksums.txt ("<sha256>  <filename>" per line,
// the same format tests/install/run.sh already verifies against).
type releaseChecksums map[string]string

var checksumLineRE = regexp.MustCompile(`^([0-9a-f]{64})\s+(\S+)$`)

func parseChecksums(raw string) (releaseChecksums, error) {
	out := releaseChecksums{}
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := checksumLineRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("checksums.txt line %d: expected \"<sha256>  <filename>\", got %q", i+1, line)
		}
		out[m[2]] = m[1]
	}
	return out, nil
}

func (c releaseChecksums) require(name string) (string, error) {
	sum, ok := c[name]
	if !ok {
		return "", fmt.Errorf("checksums.txt has no entry for %s", name)
	}
	return sum, nil
}

// homebrewFormula renders a formula that installs the exact signed release
// archive for the running platform — the same code path Homebrew exercises
// from a real tap, testable here without hosting one (spec's clean-host
// matrix runs `brew install --formula` directly against this file in CI).
func homebrewFormula(version string, sums releaseChecksums) (string, error) {
	type platArch struct{ goos, arch, brewArchCond string }
	targets := []platArch{
		{"darwin", "arm64", "Hardware::CPU.arm?"},
		{"darwin", "amd64", "Hardware::CPU.intel?"},
		{"linux", "arm64", "Hardware::CPU.arm?"},
		{"linux", "amd64", "Hardware::CPU.intel?"},
	}
	type resolved struct {
		cond, url, sha string
	}
	var darwinEntries, linuxEntries []resolved
	for _, t := range targets {
		asset := fmt.Sprintf("pose_%s_%s_%s.tar.gz", version, t.goos, t.arch)
		sum, err := sums.require(asset)
		if err != nil {
			return "", err
		}
		url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", releaseRepo, version, asset)
		r := resolved{t.brewArchCond, url, sum}
		if t.goos == "darwin" {
			darwinEntries = append(darwinEntries, r)
		} else {
			linuxEntries = append(linuxEntries, r)
		}
	}
	renderBlock := func(blockName string, entries []resolved) string {
		var b strings.Builder
		fmt.Fprintf(&b, "  %s do\n", blockName)
		for _, e := range entries {
			fmt.Fprintf(&b, "    if %s\n", e.cond)
			fmt.Fprintf(&b, "      url %q\n", e.url)
			fmt.Fprintf(&b, "      sha256 %q\n", e.sha)
			fmt.Fprintln(&b, "    end")
		}
		fmt.Fprintln(&b, "  end")
		return b.String()
	}
	var out strings.Builder
	fmt.Fprintf(&out, "class %s < Formula\n", homebrewClass)
	fmt.Fprintln(&out, "  desc \"Repository-owned governance engine for agentic software delivery\"")
	fmt.Fprintf(&out, "  homepage \"https://github.com/%s\"\n", releaseRepo)
	fmt.Fprintf(&out, "  version %q\n", version)
	fmt.Fprintln(&out, "  license \"Apache-2.0\"")
	fmt.Fprintln(&out)
	out.WriteString(renderBlock("on_macos", darwinEntries))
	out.WriteString(renderBlock("on_linux", linuxEntries))
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "  def install")
	fmt.Fprintln(&out, "    bin.install \"pose\"")
	fmt.Fprintln(&out, "  end")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "  test do")
	fmt.Fprintln(&out, "    assert_match version.to_s, shell_output(\"#{bin}/pose version\")")
	fmt.Fprintln(&out, "  end")
	fmt.Fprintln(&out, "end")
	return out.String(), nil
}

// winGetManifests renders the three-file WinGet manifest set (version,
// installer, default locale) per the current multi-file manifest schema.
func winGetManifests(version string, sums releaseChecksums) (map[string]string, error) {
	type arch struct{ id, goarch string }
	archs := []arch{{"x64", "amd64"}, {"arm64", "arm64"}}
	var installerEntries strings.Builder
	for _, a := range archs {
		asset := fmt.Sprintf("pose_%s_windows_%s.zip", version, a.goarch)
		sum, err := sums.require(asset)
		if err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", releaseRepo, version, asset)
		fmt.Fprintf(&installerEntries, "  - Architecture: %s\n", a.id)
		fmt.Fprintf(&installerEntries, "    InstallerUrl: %s\n", url)
		fmt.Fprintf(&installerEntries, "    InstallerSha256: %s\n", strings.ToUpper(sum))
	}

	versionYAML := fmt.Sprintf(`PackageIdentifier: %s
PackageVersion: %s
DefaultLocale: en-US
ManifestType: version
ManifestVersion: %s
`, winGetPackageIdentifier, version, winGetManifestVersion)

	installerYAML := fmt.Sprintf(`PackageIdentifier: %s
PackageVersion: %s
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
  - RelativeFilePath: pose.exe
    PortableCommandAlias: pose
Installers:
%sManifestType: installer
ManifestVersion: %s
`, winGetPackageIdentifier, version, installerEntries.String(), winGetManifestVersion)

	localeYAML := fmt.Sprintf(`PackageIdentifier: %s
PackageVersion: %s
PackageLocale: en-US
Publisher: Harne8
PackageName: POSE
License: Apache-2.0
ShortDescription: Repository-owned governance engine for agentic software delivery
PackageUrl: https://github.com/%s
ManifestType: defaultLocale
ManifestVersion: %s
`, winGetPackageIdentifier, version, releaseRepo, winGetManifestVersion)

	return map[string]string{
		fmt.Sprintf("%s.yaml", winGetPackageIdentifier):              versionYAML,
		fmt.Sprintf("%s.installer.yaml", winGetPackageIdentifier):    installerYAML,
		fmt.Sprintf("%s.locale.en-US.yaml", winGetPackageIdentifier): localeYAML,
	}, nil
}

func cmdReleasePackageManifests(args []string, stdout, stderr io.Writer) int {
	version, checksumsPath, outDir := "", "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "--checksums", "--out":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose release-package-manifests --version X.Y.Z --checksums <checksums.txt> --out <dir>")
			}
			i++
			switch args[i-1] {
			case "--version":
				version = args[i]
			case "--checksums":
				checksumsPath = args[i]
			case "--out":
				outDir = args[i]
			}
		default:
			return usageError(stderr, "Usage: pose release-package-manifests --version X.Y.Z --checksums <checksums.txt> --out <dir>")
		}
	}
	if version == "" || checksumsPath == "" || outDir == "" {
		return usageError(stderr, "Usage: pose release-package-manifests --version X.Y.Z --checksums <checksums.txt> --out <dir>")
	}
	if strings.Count(version, ".") != 2 {
		fmt.Fprintf(stderr, "pose release-package-manifests: --version must be X.Y.Z, got %q\n", version)
		return 2
	}
	raw, err := os.ReadFile(checksumsPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	sums, err := parseChecksums(string(raw))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	formula, err := homebrewFormula(version, sums)
	if err != nil {
		fmt.Fprintf(stderr, "homebrew: %v\n", err)
		return 1
	}
	winget, err := winGetManifests(version, sums)
	if err != nil {
		fmt.Fprintf(stderr, "winget: %v\n", err)
		return 1
	}

	brewDir := filepath.Join(outDir, "homebrew")
	wingetDir := filepath.Join(outDir, "winget")
	if err := os.MkdirAll(brewDir, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(wingetDir, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	brewPath := filepath.Join(brewDir, "pose.rb")
	if err := os.WriteFile(brewPath, []byte(formula), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "homebrew: %s\n", brewPath)

	names := make([]string, 0, len(winget))
	for name := range winget {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(wingetDir, name)
		if err := os.WriteFile(p, []byte(winget[name]), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "winget: %s\n", p)
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}
