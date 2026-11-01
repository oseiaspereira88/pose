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
// the native CLI indexer.
type Roadmap struct {
	Slug       string      `json:"slug"`
	Status     string      `json:"status"`
	CreatedAt  string      `json:"created_at,omitempty"`
	DependsOn  []string    `json:"depends_on,omitempty"`
	Consumes   []string    `json:"consumes,omitempty"`
	Milestones []Milestone `json:"milestones"`
	Path       string      `json:"path"`
	Body       string      `json:"body,omitempty"`
}

type Milestone struct {
	ID          string   `json:"id"`
	After       []string `json:"after,omitempty"`
	Consumes    []string `json:"consumes,omitempty"`
	TargetStart string   `json:"target_start,omitempty"`
	TargetDue   string   `json:"target_due,omitempty"`
	Specs       []string `json:"specs"`
}

var milestoneHeadingRE = regexp.MustCompile(`^##\s+Milestone:\s*(.+?)\s*$`)

// ReachedMilestones lists the scopes of the milestones of roadmaps not yet done
// whose members are all terminal and which hold no approval: the milestones
// `pose close` can be asked for (spec pose-milestone-closes-when-reached). A spec
// keeps the ones it is a member of.
func (s Store) ReachedMilestones(spec string) ([]string, error) {
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return nil, err
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	statuses := map[string]string{}
	for _, sp := range specs {
		statuses[sp.Slug] = sp.Status
	}
	reached := []string{}
	for _, rm := range roadmaps {
		if rm.Status == "done" {
			continue
		}
		for _, milestone := range rm.Milestones {
			if !s.milestoneMembersTerminal(milestone, statuses, spec) {
				continue
			}
			if ref := "milestone:" + rm.Slug + "/" + milestone.ID; !s.scopeApproved(ref) {
				reached = append(reached, ref)
			}
		}
	}
	return reached, nil
}

// milestoneMembersTerminal also reports that a milestone has spec as a member,
// when one is named.
func (s Store) milestoneMembersTerminal(milestone Milestone, statuses map[string]string, spec string) bool {
	member := spec == ""
	for _, raw := range milestone.Specs {
		slug, external := s.milestoneMember(raw)
		if external && !s.externalMemberDone(raw) {
			return false
		}
		if !external {
			slug = strings.TrimPrefix(slug, "spec:")
			if !terminalStatuses[statuses[slug]] {
				return false
			}
			member = member || slug == spec
		}
	}
	return member && len(milestone.Specs) > 0
}

func (s Store) scopeApproved(ref string) bool {
	eval, err := s.ReviewCheck(ref)
	return err == nil && (eval.Approved || !eval.Required)
}

// RoadmapReached reports that every milestone of a roadmap not yet done is
// approved, so `pose close roadmap:` is the next step.
func (s Store) RoadmapReached(slug string) bool {
	rm, err := s.GetRoadmap(slug)
	if err != nil || rm.Status == "done" || len(rm.Milestones) == 0 {
		return false
	}
	for _, milestone := range rm.Milestones {
		if !s.scopeApproved("milestone:" + slug + "/" + milestone.ID) {
			return false
		}
	}
	return true
}

// unapprovedPredecessors lists the milestones named in the after: of the
// milestones a spec belongs to that hold no approval. Other forms of after: are
// not milestones of this roadmap, and readiness answers for them.
func (s Store) unapprovedPredecessors(spec string) ([]string, error) {
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return nil, err
	}
	found := map[string]bool{}
	for _, rm := range roadmaps {
		for _, milestone := range rm.Milestones {
			member := false
			for _, raw := range milestone.Specs {
				slug, external := s.milestoneMember(raw)
				member = member || !external && strings.TrimPrefix(slug, "spec:") == spec
			}
			if !member {
				continue
			}
			for _, after := range milestone.After {
				if !strings.Contains(after, ":") {
					after = "milestone:" + rm.Slug + "/" + after
				}
				if ref, err := ParseArtifactRef(after); err == nil && ref.Kind == "milestone" && ref.Project == "" && !s.scopeApproved(after) {
					found[after] = true
				}
			}
		}
	}
	out := make([]string, 0, len(found))
	for ref := range found {
		out = append(out, ref)
	}
	sort.Strings(out)
	return out, nil
}

func (s Store) roadmapsDir() string { return filepath.Join(s.Root, ".pose", "roadmaps") }

// GetRoadmap returns one roadmap by slug, body included.
func (s Store) GetRoadmap(slug string) (*Roadmap, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, fmt.Errorf("pose: invalid roadmap slug %q", slug)
	}
	path := filepath.Join(s.roadmapsDir(), slug+".md")
	rm, err := parseRoadmapFile(s.fsys(), path, slug, true)
	if err != nil {
		return nil, fmt.Errorf("pose: roadmap %q not found", slug)
	}
	return rm, nil
}

// ListRoadmaps returns every roadmap (no body), sorted by slug.
func (s Store) ListRoadmaps() ([]Roadmap, error) {
	entries, err := s.fsys().ReadDir(s.roadmapsDir())
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
		rm, err := parseRoadmapFile(s.fsys(), filepath.Join(s.roadmapsDir(), e.Name()), slug, false)
		if err != nil {
			continue // um artefato quebrado não derruba a listagem
		}
		roadmaps = append(roadmaps, *rm)
	}
	sort.Slice(roadmaps, func(i, j int) bool { return roadmaps[i].Slug < roadmaps[j].Slug })
	return roadmaps, nil
}

func parseRoadmapFile(files Files, path, slug string, includeBody bool) (*Roadmap, error) {
	raw, err := files.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm, body := SplitFrontmatter(string(raw))
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
			rm.DependsOn = parseInlineList(value)
		case "consumes":
			rm.Consumes = parseInlineList(value)
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
			current.After = parseInlineList(value)
		case "consumes":
			current.Consumes = parseInlineList(value)
		case "target_start":
			current.TargetStart = value
		case "target_due":
			current.TargetDue = value
		case "specs":
			current.Specs = parseInlineList(value)
		}
	}
	if includeBody {
		rm.Body = body
	}
	return rm, nil
}
