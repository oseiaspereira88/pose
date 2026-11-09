package cli

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Spec pose-delegate-picker-and-claude-runs.

// R1: the preview command fzf hands to the login shell is one `sh -c` word, so it parses in
// fish as in any shell; the loop is inside the quoted script.
func TestThePickerPreviewIsOnePOSIXWordTheLoginShellCanParse(t *testing.T) {
	cmd := pickerPreviewCommand("/home/go/.local/bin/pose", delegateFlags{lines: 12, refresh: 2 * time.Second, verbose: true})
	if !strings.HasPrefix(cmd, "sh -c '") || !strings.Contains(cmd, `"$0" delegate watch "$1"`) || !strings.HasSuffix(cmd, " {1}") {
		t.Fatalf("the preview command is not sh -c '<script>' <pose> {1}: %s", cmd)
	}
	// The loop sits inside the single quotes, so the login shell sees no `while` of its own.
	outside := cmd[:strings.Index(cmd, "'")] + cmd[strings.LastIndex(cmd, "'")+1:]
	if strings.Contains(outside, "while") || strings.Contains(outside, "do ") || strings.Contains(outside, "done") {
		t.Fatalf("a shell loop is outside the quoted script: %s", outside)
	}
	// It runs: with `{1}` replaced by an id, sh executes the script (the fake pose is `echo`).
	run := strings.Replace(pickerPreviewCommand("/bin/echo", delegateFlags{lines: 3}), "{1}", "'tat-1'", 1)
	out, err := exec.Command("sh", "-c", run).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "delegate watch tat-1 --no-follow --lines 3") {
		t.Fatalf("the preview did not run with the id as an argument: %q %v", out, err)
	}
	// fish, when present, parses it too.
	if fish, err := exec.LookPath("fish"); err == nil {
		if out, err := exec.Command(fish, "-n", "-c", pickerPreviewCommand("/bin/echo", delegateFlags{lines: 3, refresh: time.Second})).CombinedOutput(); err != nil {
			t.Fatalf("fish refuses the preview command: %s", out)
		}
	}
}

// R2: a list line carries the state, the agent and the age in columns, the label last.
func TestTheStateGlyphIsOneCellAndEveryStateHasOne(t *testing.T) {
	for _, state := range []string{"running", "completed", "failed", "stalled", "interrupted", "other"} {
		if len([]rune(delegateStateGlyph(state))) != 1 {
			t.Errorf("state %s has no one-cell glyph", state)
		}
	}
}
