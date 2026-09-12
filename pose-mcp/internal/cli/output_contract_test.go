package cli

// Regressions from the review of pose#111 (spec pose-cli-output-rendering-system).

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
)

// The shipped binary wraps stdout to record command usage, and capability
// detection used to stop at the wrapper: every real invocation looked like a
// pipe, so the colour path was alive only in tests. A character device seen
// through a wrapper must still be recognised.
func TestTerminalDetectionSeesThroughTheUsageWrapper(t *testing.T) {
	device, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no character device to test with: %v", err)
	}
	t.Cleanup(func() { _ = device.Close() })
	if !cliout.IsTerminal(device) {
		t.Skip("the null device is not a character device on this platform")
	}
	if !cliout.IsTerminal(&usageOutput{Writer: device}) {
		t.Fatal("the usage wrapper must not hide the stream underneath")
	}
	if cliout.IsTerminal(&usageOutput{Writer: &bytes.Buffer{}}) {
		t.Fatal("a wrapped buffer is still not a terminal")
	}
}

// NO_COLOR and --color=never suppress the repainting line too: it is an escape
// sequence like any other, and R6 lists them. The same facts still arrive, one
// line per event.
func TestNoColourMeansNoRepainting(t *testing.T) {
	errOut := &bytes.Buffer{}
	profile := cliout.Plain()
	profile.TTY = true // a terminal, but colour was declined
	r := cliout.New(&bytes.Buffer{}, errOut, cliout.Plain(), profile)
	steps := r.Steps(1)
	steps.Start("test", "go test ./...").Resolve(cliout.StatePass, "")
	steps.Summary()
	got := errOut.String()
	if strings.Contains(got, "\r") || strings.Contains(got, "\x1b[") {
		t.Fatalf("NO_COLOR must not repaint: %q", got)
	}
	if !strings.Contains(got, "  -> test go test ./...") || !strings.Contains(got, "  <- test pass") {
		t.Fatalf("the facts must still arrive: %q", got)
	}
}

// --quiet is the verdict alone, so a contributor hint does not follow it.
func TestQuietCheckPrintsTheVerdictAlone(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	state, err := json.Marshal(map[string]any{"active": true, "upstream": "https://example.invalid/pose"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "state", "contributor.json"), state, 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	if code := cmdCheck(root, []string{"--strict", "--quiet"}, &out, &errB); code != 1 {
		t.Fatalf("an empty directory must fail the gate: exit=%d out=%q", code, out.String())
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "Result: FAILURE") {
		t.Fatalf("--quiet must print the verdict alone, got %q", out.String())
	}
	// Without --quiet the same run does print the hint, so the test is about the
	// flag and not about contributor mode being inactive.
	out.Reset()
	if code := cmdCheck(root, []string{"--strict"}, &out, &errB); code != 1 || !strings.Contains(out.String(), "Contributor Mode ACTIVE") {
		t.Fatalf("contributor mode must be active in this fixture: exit=%d out=%q", code, out.String())
	}
}
