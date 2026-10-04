package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Spec pose-review-assurance-disclosure R5: no manual, doc or help text says
// "independent review" for a fact the engine cannot observe.
func TestNoSurfaceClaimsAnIndependentReview(t *testing.T) {
	phrase := regexp.MustCompile(`(?i)\bindependent(ly)?\s+review(ed|er|s)?\b|\breviewed\s+independently\b`)
	root := filepath.Join("..", "..", "..")
	paths := []string{"POSE.md", "locales/pt-BR/POSE.md", "pose-mcp/internal/scaffold/dist/POSE.md"}
	docs, _ := filepath.Glob(filepath.Join(root, "docs-site", "docs", "*.md"))
	for _, doc := range docs {
		rel, _ := filepath.Rel(root, doc)
		paths = append(paths, rel)
	}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if loc := phrase.FindIndex(raw); loc != nil {
			t.Errorf("%s claims an independent review: %q", rel, string(raw[loc[0]:loc[1]]))
		}
	}
	var help strings.Builder
	for _, entry := range commandHelpCatalog {
		help.WriteString(entry.SummaryEN + "\n" + entry.DescriptionEN + "\n")
		for _, flag := range entry.Flags {
			help.WriteString(flag.DescriptionEN + "\n")
		}
		for _, sub := range entry.Subcommands {
			help.WriteString(sub.SummaryEN + "\n")
		}
	}
	if phrase.MatchString(help.String()) {
		t.Error("the help catalog claims an independent review")
	}
}
