package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Roadmap is the parsed view of a governed roadmap artifact
// (pose-roadmap-artifact): flat frontmatter + `## Milestone: <id>` body
// sections with flat bullets. Keep the parser in sync with
// .pose/scripts/pose-spec-graph.py (parse_roadmap).
type Roadmap struct {
	Slug       string      `json:"slug"`
	Status     string      `json:"status"`
	CreatedAt  string      `json:"created_at,omitempty"`
	DependsOn  []string    `json:"depends_on,omitempty"`
	Milestones []Milestone `json:"milestones"`
	Path       string      `json:"path"`
	Body       string      `json:"body,omitempty"`
}

type Milestone struct {
	ID          string   `json:"id"`
	After       []string `json:"after,omitempty"`
	TargetStart string   `json:"target_start,omitempty"`
	TargetDue   string   `json:"target_due,omitempty"`
	Specs       []string `json:"specs"`
}

var milestoneHeadingRE = regexp.MustCompile(`^##\s+Milestone:\s*(.+?)\s*$`)

func (s Store) roadmapsDir() string { return filepath.Join(s.Root, ".pose", "roadmaps") }

// GetRoadmap returns one roadmap by slug, body included.
func (s Store) GetRoadmap(slug string) (*Roadmap, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, fmt.Errorf("pose: invalid roadmap slug %q", slug)
	}
	path := filepath.Join(s.roadmapsDir(), slug+".md")
	rm, err := parseRoadmapFile(path, slug, true)
	if err != nil {
		return nil, fmt.Errorf("pose: roadmap %q not found", slug)
	}
	return rm, nil
}

// ListRoadmaps returns every roadmap (no body), sorted by slug.
func (s Store) ListRoadmaps() ([]Roadmap, error) {
	entries, err := os.ReadDir(s.roadmapsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []Roadmap{}, nil
		}
		return nil, fmt.Errorf("pose: reading roadmaps dir: %w", err)
	}
	roadmaps := []Roadmap{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		rm, err := parseRoadmapFile(filepath.Join(s.roadmapsDir(), e.Name()), slug, false)
		if err != nil {
			continue // um artefato quebrado não derruba a listagem
		}
		roadmaps = append(roadmaps, *rm)
	}
	sort.Slice(roadmaps, func(i, j int) bool { return roadmaps[i].Slug < roadmaps[j].Slug })
	return roadmaps, nil
}

func parseRoadmapFile(path, slug string, includeBody bool) (*Roadmap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm, body := splitFrontmatter(string(raw))
	rm := &Roadmap{Slug: slug, Path: path, Milestones: []Milestone{}}
	for key, value := range fm {
		switch key {
		case "slug":
			if value != "" {
				rm.Slug = value
			}
		case "status":
			rm.Status = value
		case "created_at":
			rm.CreatedAt = value
		case "depends_on":
			rm.DependsOn = parseDependsOn(value)
		}
	}
	var current *Milestone
	for _, line := range strings.Split(body, "\n") {
		stripped := strings.TrimSpace(line)
		if m := milestoneHeadingRE.FindStringSubmatch(stripped); m != nil {
			rm.Milestones = append(rm.Milestones, Milestone{ID: m[1]})
			current = &rm.Milestones[len(rm.Milestones)-1]
			continue
		}
		if strings.HasPrefix(stripped, "## ") {
			current = nil
			continue
		}
		if current == nil || !strings.HasPrefix(stripped, "- ") {
			continue
		}
		key, value, ok := strings.Cut(stripped[2:], ":")
		if !ok {
			continue
		}
		value = cleanValue(value)
		switch strings.TrimSpace(key) {
		case "after":
			current.After = parseDependsOn(value)
		case "target_start":
			current.TargetStart = value
		case "target_due":
			current.TargetDue = value
		case "specs":
			current.Specs = parseDependsOn(value)
		}
	}
	if includeBody {
		rm.Body = body
	}
	return rm, nil
}
