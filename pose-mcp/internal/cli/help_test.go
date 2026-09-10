package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUniversalHelpFlagsAllCommands(t *testing.T) {
	commands := []string{
		"init", "version", "doctor", "validate", "check", "lint-spec",
		"new-spec", "new-roadmap", "new-adr", "new-knowledge",
		"review", "close", "extension", "contribute", "release",
		"state", "followups", "index", "update", "hooks",
		"suggest", "assess", "serve-mcp", "telemetry", "import",
		"report-limitation", "artifact-check", "surface-check",
		"roadmap-check", "stats", "usage", "dora-metrics",
		"adoption-metrics", "history-check", "knowledge-check",
		"recurrence-check", "skills-check", "report",
	}

	for _, cmd := range commands {
		for _, flag := range []string{"-h", "--help"} {
			var stdout, stderr bytes.Buffer
			exitCode := Main([]string{cmd, flag}, &stdout, &stderr)
			if exitCode != 0 {
				t.Errorf("pose %s %s failed with exit code %d (stderr=%s)", cmd, flag, exitCode, stderr.String())
			}
			out := stdout.String()
			if out == "" {
				t.Errorf("pose %s %s returned empty stdout", cmd, flag)
			}
			if !strings.Contains(out, "Usage:") && !strings.Contains(out, "Uso:") {
				t.Errorf("pose %s %s output missing Usage header: %s", cmd, flag, out)
			}
		}
	}
}

func TestHierarchicalHelpCommand(t *testing.T) {
	testCases := []struct {
		args        []string
		wantSnippet string
	}{
		{[]string{"help"}, "Usage:"},
		{[]string{"help", "validate"}, "validation-matrix.json"},
		{[]string{"help", "review"}, "bundle"},
		{[]string{"help", "review", "bundle"}, "pose review bundle <scope>"},
		{[]string{"help", "extension", "install"}, "pose extension install"},
		{[]string{"help", "contribute", "stage"}, "pose contribute stage"},
		{[]string{"help", "release", "plan"}, "pose release plan"},
		{[]string{"help", "hooks", "install"}, "pose hooks install"},
	}

	for _, tc := range testCases {
		var stdout, stderr bytes.Buffer
		exitCode := Main(tc.args, &stdout, &stderr)
		if exitCode != 0 {
			t.Errorf("pose %v failed with exit code %d", tc.args, exitCode)
		}
		if !strings.Contains(stdout.String(), tc.wantSnippet) {
			t.Errorf("pose %v output missing expected snippet %q, got:\n%s", tc.args, tc.wantSnippet, stdout.String())
		}
	}
}

func TestHelpCatalogCompleteness(t *testing.T) {
	if len(commandHelpCatalog) < 30 {
		t.Fatalf("expected at least 30 commands cataloged, got %d", len(commandHelpCatalog))
	}
	for name, h := range commandHelpCatalog {
		if h.Usage == "" {
			t.Errorf("command %s has empty Usage", name)
		}
		if h.SummaryEN == "" || h.SummaryPtBR == "" {
			t.Errorf("command %s missing bilingual summary", name)
		}
	}
}

func TestBilingualHelpParity(t *testing.T) {
	// English output
	var stdoutEN bytes.Buffer
	dispatchCommandHelp("validate", nil, &stdoutEN, "en")
	if !strings.Contains(stdoutEN.String(), "Execute the deterministic validation matrix") {
		t.Errorf("expected English text in validate help, got: %s", stdoutEN.String())
	}

	// Portuguese output
	var stdoutPT bytes.Buffer
	dispatchCommandHelp("validate", nil, &stdoutPT, "pt-BR")
	if !strings.Contains(stdoutPT.String(), "Executa a matriz determinística de validação") {
		t.Errorf("expected Portuguese text in validate help, got: %s", stdoutPT.String())
	}
}

// TestReportHelpNamesEveryFlag holds `pose report --help` to the parser: four
// of its sixteen flags were documented, and `--validate-output` — which decides
// the report's derived outcome — was in no help, no manual and no reference.
func TestReportHelpNamesEveryFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"report", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pose report --help exit=%d stderr=%s", code, stderr.String())
	}
	help := stdout.String()
	flags := []string{"git-stage"}
	for flag := range reportValueFlags {
		flags = append(flags, flag)
	}
	for _, flag := range flags {
		if !strings.Contains(help, "--"+flag+" ") {
			t.Errorf("pose report accepts --%s and its help does not name it", flag)
		}
	}
}
