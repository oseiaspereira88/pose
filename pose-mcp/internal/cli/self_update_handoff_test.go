package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The handoff must run the binary that was just written, at the path it was
// written to. Re-resolving with os.Executable() looks equivalent and is not:
// this process renames itself out of the way before the copy, and on Linux
// /proc/self/exe then resolves to that removed `.old` name. CI caught it as
// `fork/exec .../cli.test.old: no such file or directory`, and no local run
// could — the replacement path needs network, so offline the update returns
// before ever handing off.
func TestSelfUpdateHandoffRunsThePathItWasGiven(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub is a shell script")
	}
	dir := t.TempDir()
	replaced := filepath.Join(dir, "pose-new")
	script := "#!/bin/sh\necho handed-off \"$@\"\nexit 0\n"
	if err := os.WriteFile(replaced, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	text := func(english, _ string) string { return english }
	if code := runSelfUpdatedBinary(replaced, []string{"--locale", "pt-BR"}, &out, &errOut, text); code != 0 {
		t.Fatalf("code = %d, err = %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "handed-off") {
		t.Fatalf("the given path was not run: %q %q", got, errOut.String())
	}
	// `--no-self` is what makes the handoff terminate rather than recurse, and
	// the original arguments have to survive it.
	if !strings.Contains(got, "update --no-self --locale pt-BR") {
		t.Errorf("forwarded arguments = %q", got)
	}
}

func TestSelfUpdateHandoffPropagatesTheExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub is a shell script")
	}
	dir := t.TempDir()
	replaced := filepath.Join(dir, "pose-new")
	if err := os.WriteFile(replaced, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := runSelfUpdatedBinary(replaced, nil, &out, &errOut, func(e, _ string) string { return e }); code != 3 {
		t.Errorf("code = %d, want the child's 3 — an update that fails after the handoff must not report success", code)
	}
}
