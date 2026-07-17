package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Markdown is a named POSE markdown artifact (workflow or rule). Body is only
// populated when fetching a single artifact; listings stay light.
type Markdown struct {
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
	Path  string `json:"path"`
	Body  string `json:"body,omitempty"`
}

// namePattern is looser than spec slugs: internal artifacts may start with
// an underscore (e.g. rules/_base-recurrence.md). Traversal stays blocked.
var namePattern = regexp.MustCompile(`^[a-z0-9_][a-z0-9._-]*$`)

// ValidateName guards workflow/rule/domain names against path traversal.
func ValidateName(name string) error {
	if name == "" || strings.Contains(name, "..") || !namePattern.MatchString(name) {
		return fmt.Errorf("pose: invalid artifact name %q", name)
	}
	return nil
}

func (s Store) workflowsDir() string { return filepath.Join(s.Root, ".pose", "workflows") }
func (s Store) rulesDir() string     { return filepath.Join(s.Root, ".pose", "rules") }
func (s Store) skillsDir() string    { return filepath.Join(s.Root, ".agents", "skills") }

// GetWorkflow returns one workflow (.pose/workflows/<name>.md), body included.
func (s Store) GetWorkflow(name string) (*Markdown, error) {
	return s.getMarkdown(s.workflowsDir(), name, "workflow")
}

// GetRule returns one domain rule (.pose/rules/<name>.md), body included.
func (s Store) GetRule(name string) (*Markdown, error) {
	return s.getMarkdown(s.rulesDir(), name, "rule")
}

// ListWorkflows returns every workflow (no body), sorted by name.
func (s Store) ListWorkflows() ([]Markdown, error) { return s.listMarkdown(s.workflowsDir()) }

// ListRules returns every domain rule (no body), sorted by name.
func (s Store) ListRules() ([]Markdown, error) { return s.listMarkdown(s.rulesDir()) }

// GetSkill returns one agent skill (.agents/skills/<name>/SKILL.md), body included.
func (s Store) GetSkill(name string) (*Markdown, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(s.skillsDir(), name, "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: skill %q not found", name)
	}
	body := string(raw)
	return &Markdown{Name: name, Title: firstHeading(body), Path: path, Body: body}, nil
}

// ListSkills returns every skill (directory name + first heading), sorted by name.
func (s Store) ListSkills() ([]Markdown, error) {
	entries, err := os.ReadDir(s.skillsDir())
	if err != nil {
		return nil, fmt.Errorf("pose: reading skills dir: %w", err)
	}
	items := []Markdown{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(s.skillsDir(), name, "SKILL.md")
		title := ""
		if raw, err := os.ReadFile(path); err == nil {
			title = firstHeading(string(raw))
		} else {
			continue // no SKILL.md — skip
		}
		items = append(items, Markdown{Name: name, Title: title, Path: path})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s Store) getMarkdown(dir, name, kind string) (*Markdown, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: %s %q not found", kind, name)
	}
	body := string(raw)
	return &Markdown{Name: name, Title: firstHeading(body), Path: path, Body: body}, nil
}

func (s Store) listMarkdown(dir string) ([]Markdown, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("pose: reading %s: %w", dir, err)
	}
	items := []Markdown{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		path := filepath.Join(dir, e.Name())
		title := ""
		if raw, err := os.ReadFile(path); err == nil {
			title = firstHeading(string(raw))
		}
		items = append(items, Markdown{Name: name, Title: title, Path: path})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}
