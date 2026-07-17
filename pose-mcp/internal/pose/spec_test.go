package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureStore builds a hermetic .pose/specs tree: two canonical specs, one
// section-split directory and one legacy flat file carrying template-style
// trailing comments.
func fixtureStore(t *testing.T) Store {
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
	write(".pose/specs/alpha/spec.md", `---
slug: alpha
status: done
created_at: 2026-06-01
completed_at: 2026-06-02
---

# Spec: alpha

## 1. Intent
Alpha body text.
`)
	write(".pose/specs/beta/spec.md", `---
slug: beta
status: draft
created_at: 2026-06-03
---

# Spec: beta
`)
	write(".pose/specs/legacy-flat.md", `---
slug: legacy-flat
status: done        # draft | in-progress | done
created_at: 2026-05-01
completed_at: 2026-05-02   # stamped on done
supersedes:          # slug da spec substituída
---

# Spec: legacy-flat
`)
	write(".pose/specs/split-spec/intent.md", `# Intent: split-spec

Split spec intent.
`)
	write(".pose/specs/split-spec/requirements.md", `# Requirements: split-spec

- Keep section files visible to the board.
`)
	write(".pose/specs/split-spec/tasks.md", `# Tasks: split-spec

- [ ] Exercise split layout parsing.
`)
	write(".pose/specs/split-spec/STATUS.md", `# Status

- **Estado:** in-progress
`)
	write(".pose/specs/.gitkeep", "")
	return Store{Root: root}
}

func TestValidateSlug(t *testing.T) {
	for _, ok := range []string{"alpha", "semql-entity-aliases", "a.b_c-1"} {
		if err := ValidateSlug(ok); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "..", "../etc", "a/b", "a\\b", "UPPER", "-lead", ".hidden"} {
		if err := ValidateSlug(bad); err == nil {
			t.Errorf("ValidateSlug(%q) = nil, want error (traversal guard)", bad)
		}
	}
}

func TestGetSpec_Canonical(t *testing.T) {
	s := fixtureStore(t)
	sp, err := s.GetSpec("alpha")
	if err != nil {
		t.Fatalf("GetSpec(alpha): %v", err)
	}
	if sp.Status != "done" || sp.CreatedAt != "2026-06-01" || sp.CompletedAt != "2026-06-02" {
		t.Errorf("frontmatter mismatch: %+v", sp)
	}
	if sp.Title != "Spec: alpha" {
		t.Errorf("Title = %q, want %q", sp.Title, "Spec: alpha")
	}
	if !strings.Contains(sp.Body, "Alpha body text.") {
		t.Errorf("body not included: %q", sp.Body)
	}
}

func TestGetSpec_LegacyFlatWithComments(t *testing.T) {
	s := fixtureStore(t)
	sp, err := s.GetSpec("legacy-flat")
	if err != nil {
		t.Fatalf("GetSpec(legacy-flat): %v", err)
	}
	if sp.Status != "done" {
		t.Errorf("Status = %q, want done (template comment must be stripped)", sp.Status)
	}
	if sp.CompletedAt != "2026-05-02" {
		t.Errorf("CompletedAt = %q, want 2026-05-02", sp.CompletedAt)
	}
	if sp.Supersedes != "" {
		t.Errorf("Supersedes = %q, want empty (comment-only value)", sp.Supersedes)
	}
}

func TestGetSpec_SplitDirectory(t *testing.T) {
	s := fixtureStore(t)
	sp, err := s.GetSpec("split-spec")
	if err != nil {
		t.Fatalf("GetSpec(split-spec): %v", err)
	}
	if sp.Status != "in-progress" {
		t.Errorf("Status = %q, want in-progress", sp.Status)
	}
	if sp.Title != "Intent: split-spec" {
		t.Errorf("Title = %q, want %q", sp.Title, "Intent: split-spec")
	}
	for _, want := range []string{"Split spec intent.", "Keep section files visible", "Exercise split layout parsing"} {
		if !strings.Contains(sp.Body, want) {
			t.Errorf("body missing %q: %q", want, sp.Body)
		}
	}
}

func TestGetSpec_NotFoundAndInvalid(t *testing.T) {
	s := fixtureStore(t)
	if _, err := s.GetSpec("missing"); err == nil {
		t.Error("expected not-found error")
	}
	if _, err := s.GetSpec("../escape"); err == nil {
		t.Error("expected slug validation error")
	}
}

func TestListSpecs_SortedNoBody(t *testing.T) {
	s := fixtureStore(t)
	specs, err := s.ListSpecs("")
	if err != nil {
		t.Fatalf("ListSpecs: %v", err)
	}
	if len(specs) != 4 {
		t.Fatalf("len = %d, want 4 (%+v)", len(specs), specs)
	}
	want := []string{"alpha", "beta", "legacy-flat", "split-spec"}
	for i, w := range want {
		if specs[i].Slug != w {
			t.Errorf("specs[%d].Slug = %q, want %q (sorted)", i, specs[i].Slug, w)
		}
		if specs[i].Body != "" {
			t.Errorf("listing must not carry body (slug %s)", specs[i].Slug)
		}
	}
}

func TestListSpecs_StatusFilter(t *testing.T) {
	s := fixtureStore(t)
	done, err := s.ListSpecs("DONE") // case-insensitive
	if err != nil {
		t.Fatalf("ListSpecs(done): %v", err)
	}
	if len(done) != 2 {
		t.Errorf("done count = %d, want 2", len(done))
	}
	draft, _ := s.ListSpecs("draft")
	if len(draft) != 1 || draft[0].Slug != "beta" {
		t.Errorf("draft filter mismatch: %+v", draft)
	}
	inProgress, _ := s.ListSpecs("in-progress")
	if len(inProgress) != 1 || inProgress[0].Slug != "split-spec" {
		t.Errorf("in-progress filter mismatch: %+v", inProgress)
	}
}

// TestIntegration_DogfoodRepo exercises the store against the real .pose/ of
// this repository (the platform reading its own governance).
func TestIntegration_DogfoodRepo(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, ".pose", "specs", "semql-entity-aliases")); err != nil {
		t.Skip("monorepo instance not available (standalone dist repo)")
	}
	s := Store{Root: root}
	sp, err := s.GetSpec("semql-entity-aliases")
	if err != nil {
		t.Fatalf("GetSpec(semql-entity-aliases): %v", err)
	}
	if sp.Status != "done" {
		t.Errorf("semql-entity-aliases status = %q, want done", sp.Status)
	}
	all, err := s.ListSpecs("")
	if err != nil || len(all) < 2 {
		t.Errorf("ListSpecs on repo: len=%d err=%v, want >=2 specs", len(all), err)
	}
}
