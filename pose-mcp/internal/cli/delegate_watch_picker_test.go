package cli

import (
	"os/exec"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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

// R2/R3: the rows and the arguments of the picker. The state column is as wide as the widest
// "glyph state" in code points, so no state shifts the columns after it; the id leads and is
// hidden; the layout is reversed with a border and a header over the same columns.
func TestThePickerRowsAlignEveryStateAndTheArgumentsHideTheID(t *testing.T) {
	candidates := []posemodel.DelegationAttempt{
		{AttemptID: "a1", State: "completed", Agent: "x", Label: "one"},
		{AttemptID: "a2", State: "interrupted", Agent: "y", Label: "two"},
		{AttemptID: "a3", State: "running", Agent: "z", Label: "three"},
	}
	var agentAt []int
	for i, line := range pickerLines(candidates) {
		if !strings.HasPrefix(line, candidates[i].AttemptID+"\t") {
			t.Errorf("the id does not lead row %d: %q", i, line)
		}
		_, rest, _ := strings.Cut(line, "\t")
		at := utf8.RuneCountInString(rest[:strings.Index(rest, string(candidates[i].Agent[0])+" ")])
		agentAt = append(agentAt, at)
	}
	for _, at := range agentAt {
		if at != agentAt[0] {
			t.Fatalf("the agent column shifts between states: %v", agentAt)
		}
	}
	if want := utf8.RuneCountInString("◐ interrupted"); agentAt[0] < want {
		t.Errorf("the state column (%d) is narrower than the widest state (%d)", agentAt[0], want)
	}
	args := strings.Join(pickerArgs("PREVIEW", candidates), "\x00")
	for _, want := range []string{"--with-nth\x002..", "--layout\x00reverse", "--border", "--preview\x00PREVIEW", "--delimiter\x00\t"} {
		if !strings.Contains(args, want) {
			t.Errorf("the picker arguments lack %q", want)
		}
	}
	header := pickerArgs("P", candidates)
	hdr := header[len(header)-1]
	if got := utf8.RuneCountInString(hdr[:strings.Index(hdr, "AGENT")]); got != agentAt[0] {
		t.Errorf("the header does not line up with the rows: %q", hdr)
	}
}
