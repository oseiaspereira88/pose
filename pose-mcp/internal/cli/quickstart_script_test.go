package cli

import (
	"os/exec"
	"strings"
	"testing"
)

// Spec pose-quickstart-real-lifecycle R4: the quickstart page is executed as
// written, every chapter, so the validation matrix carries its evidence.
func TestQuickstartScriptRunsEveryChapter(t *testing.T) {
	if testing.Short() {
		t.Skip("the quickstart builds the binary and walks a full delivery")
	}
	for _, tool := range []string{"bash", "python3", "ssh-keygen", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	out, err := exec.Command("bash", "../../../tests/quickstart/first-governed-loop.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("the quickstart diverged from its page: %v\n%s", err, out)
	}
	for _, chapter := range []string{"chapter 1:", "chapter 2:", "chapter 3:", "chapter 4:", "chapter 5:"} {
		if !strings.Contains(string(out), chapter) {
			t.Fatalf("the quickstart did not run %s\n%s", chapter, out)
		}
	}
}
