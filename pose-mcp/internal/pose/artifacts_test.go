package pose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func artifactsFixture(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/workflows/feature.md", "# Workflow: Feature\n\n## Objetivo\nEntregar feature.\n")
	write(".pose/workflows/bugfix.md", "# Workflow: Bugfix\n\nCausa raiz primeiro.\n")
	write(".pose/rules/security.md", "# Rule: Security\n\nSem segredos.\n")
	write(".pose/rules/_base-recurrence.md", "# Base: Recorrência\n")
	return Store{Root: root}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"feature", "documentation-update", "_base-recurrence", "backend-go"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "..", "../etc", "a/b", "UPPER", "-lead"} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", bad)
		}
	}
}

func TestGetWorkflowAndRule(t *testing.T) {
	s := artifactsFixture(t)
	wf, err := s.GetWorkflow("feature")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if wf.Title != "Workflow: Feature" || !strings.Contains(wf.Body, "Entregar feature.") {
		t.Errorf("workflow mismatch: %+v", wf)
	}
	rule, err := s.GetRule("_base-recurrence") // leading underscore is valid
	if err != nil {
		t.Fatalf("GetRule(_base-recurrence): %v", err)
	}
	if rule.Title != "Base: Recorrência" {
		t.Errorf("rule title = %q", rule.Title)
	}
	if _, err := s.GetWorkflow("missing"); err == nil {
		t.Error("expected not-found error")
	}
	if _, err := s.GetRule("../escape"); err == nil {
		t.Error("expected validation error (traversal)")
	}
}

func TestListWorkflowsAndRules(t *testing.T) {
	s := artifactsFixture(t)
	wfs, err := s.ListWorkflows()
	if err != nil || len(wfs) != 2 {
		t.Fatalf("ListWorkflows: len=%d err=%v, want 2", len(wfs), err)
	}
	if wfs[0].Name != "bugfix" || wfs[1].Name != "feature" {
		t.Errorf("order mismatch: %+v", wfs)
	}
	if wfs[0].Body != "" {
		t.Error("listing must not carry body")
	}
	rules, err := s.ListRules()
	if err != nil || len(rules) != 2 {
		t.Fatalf("ListRules: len=%d err=%v, want 2", len(rules), err)
	}
}

// TestIntegration_SuggestDogfood wraps the real CLI of this repository.
func TestIntegration_SuggestDogfood(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "pose")); err != nil {
		t.Skip("repo pose wrapper not available")
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "pose-mcp", "spec.md")); err != nil {
		t.Skip("monorepo instance not available (standalone dist repo)")
	}
	s := Store{Root: root}
	out, err := s.Suggest(context.Background(), "feature", "", "")
	if err != nil {
		t.Fatalf("Suggest(feature): %v", err)
	}
	m, ok := out.(map[string]any)
	if !ok || m["workflow"] == nil || m["skill"] == nil {
		t.Errorf("suggest output missing canonical fields: %v", out)
	}
	if m["skill"] != "pose-feature" {
		t.Errorf("skill = %v, want pose-feature", m["skill"])
	}
	if _, err := s.Suggest(context.Background(), "tipo-inexistente", "", ""); err == nil {
		t.Error("expected CLI error for unknown task type")
	}
	if _, err := s.Suggest(context.Background(), "feature", "", "../escape"); err == nil {
		t.Error("expected path validation error")
	}
}

// TestIntegration_GatesDogfood evaluates the real read-only gates of this
// repository through the adapter (followups, check, lint-spec).
func TestIntegration_GatesDogfood(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "pose")); err != nil {
		t.Skip("repo pose wrapper not available")
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "pose-mcp", "spec.md")); err != nil {
		t.Skip("monorepo instance not available (standalone dist repo)")
	}
	s := Store{Root: root}
	ctx := context.Background()

	fu, err := s.Followups(ctx, false)
	if err != nil {
		t.Fatalf("Followups: %v", err)
	}
	if m, ok := fu.(map[string]any); !ok || m["total"] == nil {
		t.Errorf("followups shape unexpected: %v", fu)
	}

	check, err := s.Check(ctx, true)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !check.Passed || check.ExitCode != 0 {
		t.Errorf("check --strict should pass on this repo: %+v", check)
	}

	lint, err := s.LintSpec(ctx, "semql-entity-aliases", true)
	if err != nil {
		t.Fatalf("LintSpec: %v", err)
	}
	if !lint.Passed || !strings.Contains(lint.Output, "SUCESSO") {
		t.Errorf("lint of a closed spec should pass: %+v", lint)
	}
	if !strings.Contains(lint.Command, "lint-spec semql-entity-aliases --strict") {
		t.Errorf("command echo = %q", lint.Command)
	}

	if _, err := s.LintSpec(ctx, "../escape", true); err == nil {
		t.Error("expected slug validation error")
	}
}
