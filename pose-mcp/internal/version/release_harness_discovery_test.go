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
	for _, omitted := range localGateOmissions(string(ci), text) {
		t.Errorf("local verifier omits CI gate: %s", omitted)
	}
}

func localGateOmissions(ci, local string) []string {
	var omitted []string
	scripts := regexp.MustCompile(`(?m)^\s*run: bash ([^\s]+)([^\n]*)`).FindAllStringSubmatch(ci, -1)
	for _, match := range scripts {
		if !strings.Contains(local, "bash "+match[1]+match[2]) {
			omitted = append(omitted, match[0])
		}
	}
	commands := regexp.MustCompile(`(?m)^\s*run: '\"\$RUNNER_TEMP/pose\" ([^']+)'`).FindAllStringSubmatch(ci, -1)
	for _, match := range commands {
		if !strings.Contains(local, `"$BIN" `+match[1]) {
			omitted = append(omitted, match[1])
		}
	}
	goCommands := regexp.MustCompile(`(?m)^\s*run: (go [^\n]+)`).FindAllStringSubmatch(ci, -1)
	for _, match := range goCommands {
		if !strings.Contains(local, match[1]) {
			omitted = append(omitted, match[1])
		}
	}
	if len(scripts) == 0 || len(commands) == 0 || len(goCommands) == 0 {
		omitted = append(omitted, "CI gate discovery lost a command family")
	}
	return omitted
}

func TestLocalVerifyDetectsNewCIGates(t *testing.T) {
	ci, _ := os.ReadFile("../../../.github/workflows/ci.yml")
	local, _ := os.ReadFile("../../../scripts/verify.sh")
	for _, gate := range []string{"bash tests/release/new-gate.sh --strict", `'"$RUNNER_TEMP/pose" new-gate --strict'`, "go -C pose-mcp test ./new-package -count=1"} {
		if len(localGateOmissions(string(ci)+"\n        run: "+gate+"\n", string(local))) != 1 {
			t.Fatalf("new CI gate was not detected: %s", gate)
		}
	}
}

func TestReleaseObligationsAndNegativeControls(t *testing.T) {
	command := exec.Command("python3", "../../../tests/release/release-obligations.py")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("release obligations: %v\n%s", err, output)
	}
}

func TestActivationRecordingContract(t *testing.T) {
	command := exec.Command("python3", "../../../examples/demo/capture.py", "--check")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("activation recording: %v\n%s", err, output)
	}
}
