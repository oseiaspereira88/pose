package version_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseDocsParityDiscoversNestedDocument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux shell harness")
	}
	root := t.TempDir()
	docs := filepath.Join(root, "docs-site/docs/nested folder")
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(root, "README.md"):    "https://example.invalid/releases/download/vX.Y.Z/base.tar.gz\n",
		filepath.Join(docs, "new guide.md"): "https://example.invalid/releases/download/vX.Y.Z/new-asset.zip\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"git": "#!/bin/sh\nprintf '%s\\n' \"$PARITY_TEST_ROOT\"\n",
		"gh":  "#!/bin/sh\nprintf '%s\\n' \"$PARITY_TEST_ASSETS\"\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	script, err := filepath.Abs("../../../tests/release/docs-asset-parity.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		assets string
		pass   bool
	}{
		{"base.tar.gz", false}, {"base.tar.gz\nnew-asset.zip", true},
	} {
		cmd := exec.Command("bash", script, "v6.2.0")
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "PARITY_TEST_ROOT="+root, "PARITY_TEST_ASSETS="+tc.assets)
		out, err := cmd.CombinedOutput()
		if (err == nil) != tc.pass || !strings.Contains(string(out), "new-asset.zip") {
			t.Fatalf("pass=%v: %v %s", tc.pass, err, out)
		}
	}
}

func TestLocalVerifyCoversCurrentCIGates(t *testing.T) {
	ci, err := os.ReadFile("../../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	local, err := os.ReadFile("../../../scripts/verify.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(local)
	scripts := regexp.MustCompile(`(?m)^\s*run: bash ([^\s]+)([^\n]*)`).FindAllStringSubmatch(string(ci), -1)
	if len(scripts) == 0 {
		t.Fatal("no CI script gates detected")
	}
	for _, match := range scripts {
		if !strings.Contains(text, "bash "+match[1]+match[2]) {
			t.Errorf("local verifier omits CI gate: %s", match[0])
		}
	}
	commands := regexp.MustCompile(`(?m)^\s*run: '\"\$RUNNER_TEMP/pose\" ([^']+)'`).FindAllStringSubmatch(string(ci), -1)
	if len(commands) == 0 {
		t.Fatal("no CI POSE gates detected")
	}
	for _, match := range commands {
		if !strings.Contains(text, `"$BIN" `+match[1]) {
			t.Errorf("local verifier omits CI gate: %s", match[1])
		}
	}
}
