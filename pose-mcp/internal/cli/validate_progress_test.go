package cli

// Steps, capture and --verbose (spec pose-cli-output-rendering-system R6, R7).
//
// `pose validate` streamed each check's raw output for minutes and printed no
// per-check outcome, duration or counter, so a human watching a gate saw a wall
// of `go test` noise and then a verdict. These tests hold the replacement: a
// step per check on the progress stream, the output of a failing check shown as
// a tail, and streaming available on request.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// progressFixture writes a matrix with one passing and one failing check, both
// of which print, so capture and tailing are observable.
func progressFixture(t *testing.T) string {
	t.Helper()
	repo := newGitRepo(t)
	module := filepath.Join(repo, "service")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.test/service\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shell := "/bin/sh"
	if _, err := os.Stat(shell); err != nil {
		t.Skipf("no %s on this platform", shell)
	}
	matrix := fmt.Sprintf(`{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[
		{"name":"quiet","program":%q,"args":["-c","echo passing-check-noise"],"severity":"required"},
		{"name":"loud","program":%q,"args":["-c","echo failing-check-noise; exit 3"],"severity":"required"}
	]}},"moduleOverrides":{}}`, shell, shell)
	dir := filepath.Join(repo, ".pose", "indexes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestValidatePrintsAStepPerCheckAndCapturesTheirOutput(t *testing.T) {
	repo := progressFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--module", "service", "--json", ".pose/results/validate.json"}, &out, &errB); code != 1 {
			t.Fatalf("a failing required check must fail the gate: exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		progress, result := errB.String(), out.String()

		// Every check reports its own outcome and duration, which existed only
		// in the JSON before.
		step := regexp.MustCompile(`  <- quiet pass [0-9]+\.[0-9]s`)
		if !step.MatchString(progress) {
			t.Errorf("a passing check must resolve with its duration:\n%s", progress)
		}
		if !regexp.MustCompile(`  <- loud fail [0-9]+\.[0-9]s exit=3`).MatchString(progress) {
			t.Errorf("a failing check must resolve with its exit code:\n%s", progress)
		}
		if !strings.Contains(progress, "2 step(s) · 1 pass · 1 fail") {
			t.Errorf("the group must summarise what it ran:\n%s", progress)
		}

		// A passing check's noise is captured, not streamed; a failing one's is
		// shown as a tail, with the full output one path away.
		// The step names the command, so the string appears there; what must not
		// appear is the check's own output, which is printed indented.
		if strings.Contains(progress, "      passing-check-noise") || strings.Contains(result, "passing-check-noise") {
			t.Errorf("a passing check's output must stay captured:\n%s\n%s", progress, result)
		}
		if !strings.Contains(progress, "      failing-check-noise") {
			t.Errorf("a failing check's output must be shown under it:\n%s", progress)
		}
		if !strings.Contains(progress, "full output in .pose/results/validate.json") {
			t.Errorf("the tail must point at the whole output:\n%s", progress)
		}

		// The verdict is the result and stays on stdout, where a script reads
		// it; progress never lands there.
		if !strings.Contains(result, "Result: FAILURE — required check failed") {
			t.Errorf("the verdict must name the reason on stdout: %q", result)
		}
		if strings.Contains(result, "  <- ") || strings.Contains(result, "[module]") {
			t.Errorf("progress must not reach stdout: %q", result)
		}

		// The captured output still reaches the recorded evidence, redacted as
		// before: capture changed the terminal, not the result file.
		raw, err := os.ReadFile(filepath.Join(repo, ".pose", "results", "validate.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "failing-check-noise") {
			t.Error("the validation result must keep each check's captured output")
		}
	})
}

func TestValidateVerboseStreamsTheCheckOutputAgain(t *testing.T) {
	repo := progressFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--module", "service", "--verbose"}, &out, &errB); code != 1 {
			t.Fatalf("exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "passing-check-noise") {
			t.Errorf("--verbose must stream a check's output as it runs:\n%s", out.String())
		}
	})
}

// Nothing the run prints carries an escape sequence when the destination is not
// a terminal: these buffers are what CI and an agent see.
func TestValidateWritesNoEscapesOutsideATerminal(t *testing.T) {
	repo := progressFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		Main([]string{"validate", "--module", "service"}, &out, &errB)
		escapes := regexp.MustCompile(`\x1b\[`)
		if escapes.MatchString(out.String()) || escapes.MatchString(errB.String()) {
			t.Errorf("no terminal, no escapes:\nout=%q\nerr=%q", out.String(), errB.String())
		}
	})
}
