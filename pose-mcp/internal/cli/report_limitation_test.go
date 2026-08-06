package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportLimitationUsesStandardUpstreamLabels(t *testing.T) {
	cases := map[string]string{
		"limitation": "enhancement",
		"suggestion": "enhancement",
		"bug":        "bug",
	}
	for kind, want := range cases {
		if got := feedbackIssueLabel(kind); got != want {
			t.Errorf("feedbackIssueLabel(%q) = %q, want %q", kind, got, want)
		}
	}

	// Both community intake paths — the CLI above and the GitHub issue forms
	// below — must agree on labels the upstream repository actually defines.
	// The templates live outside this Go module, so a standalone module
	// checkout skips instead of failing.
	templates := filepath.Join("..", "..", "..", ".github", "ISSUE_TEMPLATE")
	if _, err := os.Stat(templates); err != nil {
		t.Skipf("issue templates not present in this checkout: %v", err)
	}
	for _, template := range []string{"feature_suggestion.yml", "engine_limitation.yml"} {
		raw, err := os.ReadFile(filepath.Join(templates, template))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, `labels: ["enhancement"]`) {
			t.Errorf("%s does not use the enhancement label: %s", template, text)
		}
		for _, removed := range []string{"feature-proposal", "engine-limitation"} {
			if strings.Contains(text, removed) {
				t.Errorf("%s still references undefined label %q", template, removed)
			}
		}
	}
}
