package cli

// Parity tests for the native gates (spec pose-cli-native-gates): the bash
// engine is the source of truth; the native port must produce the same
// verdicts and machine metrics over the ENTIRE spec corpus of this repo.
// Any divergence is a test failure, not a warning.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func repoRootForTest(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not in a git repo")
	}
	return strings.TrimSpace(string(out))
}

// metricLines filters the machine-readable "key=value" lines from output.
func metricLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "[") &&
			!strings.HasPrefix(strings.TrimSpace(line), "-") {
			out = append(out, strings.TrimSpace(line))
		}
	}
	sort.Strings(out)
	return out
}

func TestLintSpecParityCorpus(t *testing.T) {
	root := repoRootForTest(t)
	linter := filepath.Join(root, ".pose", "scripts", "pose-lint-spec.py")
	if _, err := os.Stat(linter); err != nil {
		t.Skip("bash engine not available")
	}
	specsDir := filepath.Join(root, ".pose", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		specMD := filepath.Join(specsDir, e.Name(), "spec.md")
		if _, err := os.Stat(specMD); err != nil {
			continue
		}
		for _, ready := range []bool{false, true} {
			// Python engine.
			args := []string{linter, "--spec", specMD}
			if ready {
				args = append(args, "--ready-check")
			}
			pyCmd := exec.Command("python3", args...)
			var pyOut, pyErr bytes.Buffer
			pyCmd.Stdout, pyCmd.Stderr = &pyOut, &pyErr
			pyRC := 0
			if err := pyCmd.Run(); err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					pyRC = ee.ExitCode()
				} else {
					t.Fatalf("python engine: %v", err)
				}
			}
			// Native port.
			var goOut, goErr bytes.Buffer
			goRC := lintOneSpec(specMD, false, ready, &goOut, &goErr)

			mode := "full"
			if ready {
				mode = "ready"
			}
			if pyRC != goRC {
				t.Errorf("%s [%s]: exit divergence py=%d go=%d\npy-stderr: %s\ngo-stderr: %s",
					e.Name(), mode, pyRC, goRC, pyErr.String(), goErr.String())
				continue
			}
			pyMetrics := metricLines(pyOut.String())
			goMetrics := metricLines(goOut.String())
			// Normalize spec.path (absolute in both, should match exactly).
			if strings.Join(pyMetrics, "\n") != strings.Join(goMetrics, "\n") {
				t.Errorf("%s [%s]: metric divergence\npy: %v\ngo: %v",
					e.Name(), mode, pyMetrics, goMetrics)
			}
		}
		checked++
	}
	if checked < 50 {
		t.Skipf("corpus too small (%d specs) — full-corpus parity runs in the monorepo", checked)
	}
	t.Logf("parity verified over %d specs (full + ready-check)", checked)
}

func TestHistoryCheckParity(t *testing.T) {
	root := repoRootForTest(t)
	script := filepath.Join(root, ".pose", "scripts", "pose-history-check.sh")
	if _, err := os.Stat(script); err != nil {
		t.Skip("bash engine not available")
	}
	for _, mode := range []string{"--tolerant", "--strict"} {
		bashCmd := exec.Command("bash", script, mode)
		bashCmd.Dir = root
		var bashOut, bashErr bytes.Buffer
		bashCmd.Stdout, bashCmd.Stderr = &bashOut, &bashErr
		bashRC := 0
		if err := bashCmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				bashRC = ee.ExitCode()
			}
		}
		var goOut, goErr bytes.Buffer
		goRC := 0
		inDir(t, root, func() {
			goRC = cmdHistoryCheck([]string{mode}, &goOut, &goErr)
		})
		if bashRC != goRC {
			t.Errorf("history-check %s: exit divergence bash=%d go=%d", mode, bashRC, goRC)
		}
		if strings.Join(metricLines(bashOut.String()), "\n") != strings.Join(metricLines(goOut.String()), "\n") {
			t.Errorf("history-check %s: metric divergence\nbash: %v\ngo: %v",
				mode, metricLines(bashOut.String()), metricLines(goOut.String()))
		}
	}
}
