// Package pose provides read-only access to the POSE artifacts of a single
// project root (its .pose/ directory). It never writes: mutations to POSE
// files belong to the Harness + approval flow (ADR-003), not to this adapter.
package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Spec is the parsed view of a POSE spec: the lifecycle frontmatter plus the
// markdown body. Body is only populated by GetSpec; listings stay light.
type Spec struct {
	Slug        string   `json:"slug"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at,omitempty"`
	CompletedAt string   `json:"completed_at,omitempty"`
	Supersedes  string   `json:"supersedes,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Title       string   `json:"title,omitempty"`
	Path        string   `json:"path"`
	Body        string   `json:"body,omitempty"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// ValidateSlug rejects anything that could escape .pose/specs (path
// traversal) or that does not look like a POSE slug.
func ValidateSlug(slug string) error {
	if slug == "" || strings.Contains(slug, "..") || !slugPattern.MatchString(slug) {
		return fmt.Errorf("pose: invalid spec slug %q", slug)
	}
	return nil
}

// Store reads POSE artifacts from one project root (ADR-013: project-scoped).
type Store struct {
	Root string // project root containing .pose/
}

func (s Store) specsDir() string { return filepath.Join(s.Root, ".pose", "specs") }

// GetSpec returns a spec by slug, body included. The canonical layout is
// .pose/specs/<slug>/spec.md; flat legacy files (.pose/specs/<slug>.md) and
// section-split directories are also supported.
func (s Store) GetSpec(slug string) (*Spec, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}
	canonical := filepath.Join(s.specsDir(), slug, "spec.md")
	if _, err := os.Stat(canonical); err == nil {
		return parseSpecFile(canonical, slug, true)
	}
	legacy := filepath.Join(s.specsDir(), slug+".md")
	if _, err := os.Stat(legacy); err == nil {
		return parseSpecFile(legacy, slug, true)
	}
	splitDir := filepath.Join(s.specsDir(), slug)
	if files := splitSpecFiles(splitDir); len(files) > 0 {
		return parseSplitSpec(splitDir, slug, files, true)
	}
	return nil, fmt.Errorf("pose: spec %q not found", slug)
}

// ListSpecs returns the frontmatter of every spec (no body), sorted by slug.
// status, when non-empty, filters case-insensitively on the lifecycle state.
func (s Store) ListSpecs(status string) ([]Spec, error) {
	entries, err := os.ReadDir(s.specsDir())
	if err != nil {
		return nil, fmt.Errorf("pose: reading specs dir: %w", err)
	}
	specs := []Spec{}
	for _, e := range entries {
		var sp *Spec
		var err error
		switch {
		case e.IsDir():
			slug := e.Name()
			dir := filepath.Join(s.specsDir(), slug)
			path := filepath.Join(dir, "spec.md")
			if _, statErr := os.Stat(path); statErr == nil {
				sp, err = parseSpecFile(path, slug, false)
			} else if files := splitSpecFiles(dir); len(files) > 0 {
				sp, err = parseSplitSpec(dir, slug, files, false)
			} else {
				continue // directory without a recognizable spec artifact
			}
		case strings.HasSuffix(e.Name(), ".md") && !strings.EqualFold(e.Name(), "README.md"):
			slug := strings.TrimSuffix(e.Name(), ".md")
			path := filepath.Join(s.specsDir(), e.Name())
			sp, err = parseSpecFile(path, slug, false)
		default:
			continue
		}
		if err != nil {
			continue // one unparseable artifact must not break the listing
		}
		if status != "" && !strings.EqualFold(sp.Status, status) {
			continue
		}
		specs = append(specs, *sp)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Slug < specs[j].Slug })
	return specs, nil
}

var splitSpecSectionFiles = []string{
	"intent.md",
	"requirements.md",
	"technical-plan.md",
	"tasks.md",
	"decisions.md",
	"validation.md",
	"final-report.md",
	"STATUS.md",
}

func splitSpecFiles(dir string) []string {
	var files []string
	for _, name := range splitSpecSectionFiles {
		path := filepath.Join(dir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			files = append(files, path)
		}
	}
	return files
}

func parseSplitSpec(dir, slug string, files []string, includeBody bool) (*Spec, error) {
	body, err := readSplitSpecBody(files)
	if err != nil {
		return nil, err
	}
	sp := &Spec{
		Slug:   slug,
		Status: splitSpecStatus(files),
		Title:  firstHeading(body),
		Path:   dir,
	}
	if includeBody {
		sp.Body = body
	}
	return sp, nil
}

func readSplitSpecBody(files []string) (string, error) {
	sections := make([]string, 0, len(files))
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("pose: reading split spec section: %w", err)
		}
		section := strings.TrimSpace(string(raw))
		if section != "" {
			sections = append(sections, section)
		}
	}
	return strings.Join(sections, "\n\n"), nil
}

func splitSpecStatus(files []string) string {
	for _, path := range files {
		if !strings.EqualFold(filepath.Base(path), "STATUS.md") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		content := strings.ToLower(string(raw))
		for _, status := range []string{"done", "in-progress", "draft", "blocked"} {
			if strings.Contains(content, status) {
				return status
			}
		}
	}
	return ""
}

func parseSpecFile(path, slug string, includeBody bool) (*Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: reading spec: %w", err)
	}
	fm, body := splitFrontmatter(string(raw))
	sp := &Spec{Slug: slug, Path: path}
	for key, value := range fm {
		switch key {
		case "slug":
			if value != "" {
				sp.Slug = value
			}
		case "status":
			sp.Status = value
		case "created_at":
			sp.CreatedAt = value
		case "completed_at":
			sp.CompletedAt = value
		case "supersedes":
			sp.Supersedes = value
		case "depends_on":
			sp.DependsOn = parseDependsOn(value)
		case "priority":
			if n, err := strconv.Atoi(value); err == nil && n >= 0 {
				sp.Priority = &n
			}
		}
	}
	sp.Title = firstHeading(body)
	if includeBody {
		sp.Body = body
	}
	return sp, nil
}

// splitFrontmatter separates the leading `--- … ---` block from the markdown
// body. POSE frontmatter is deliberately flat (key: value per line) — that is
// the whole contract — and template files carry trailing `# comments` on the
// value, which are stripped.
func splitFrontmatter(content string) (map[string]string, string) {
	fm := map[string]string{}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return fm, strings.Join(lines[i+1:], "\n")
		}
		key, value, ok := strings.Cut(lines[i], ":")
		if !ok {
			continue
		}
		fm[strings.TrimSpace(key)] = cleanValue(value)
	}
	return fm, "" // unterminated frontmatter: no body
}

func cleanValue(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "#") {
		return "" // value is only a template comment
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i] // strip trailing template comment
	}
	return strings.TrimSpace(v)
}

// parseDependsOn parses the flat inline list of dependency refs ("a, b" or
// "[a, b]"). Refs keep their typed prefixes (milestone:/roadmap:) verbatim —
// resolution semantics live in SpecReadiness, not in parsing.
func parseDependsOn(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = value[1 : len(value)-1]
	}
	var refs []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			refs = append(refs, item)
		}
	}
	return refs
}

func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
