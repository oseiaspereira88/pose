package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type KnowledgeEntry struct {
	Slug        string `json:"slug"`
	Type        string `json:"type"`         // handoff | decision-log | note
	Owner       string `json:"owner"`
	Sensitivity string `json:"sensitivity"` // public | public-internal | restricted
	CreatedAt   string `json:"created_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	Body        string `json:"body,omitempty"` // omitted on list, present on get
}

func (s Store) knowledgeDir() string { return filepath.Join(s.Root, ".pose", "knowledge") }

// ListKnowledge returns all knowledge entries that are not sensitivity: restricted.
// Body is omitted — use GetKnowledge to fetch the full entry.
func (s Store) ListKnowledge() ([]KnowledgeEntry, error) {
	entries, err := os.ReadDir(s.knowledgeDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []KnowledgeEntry{}, nil
		}
		return nil, fmt.Errorf("pose: reading knowledge dir: %w", err)
	}

	var knowledge []KnowledgeEntry
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		path := filepath.Join(s.knowledgeDir(), e.Name())
		ke, err := parseKnowledgeFile(path, false)
		if err != nil {
			continue // skip unparseable entries
		}
		if ke.Sensitivity == "restricted" {
			continue // exclude restricted entries
		}
		knowledge = append(knowledge, *ke)
	}
	sort.Slice(knowledge, func(i, j int) bool { return knowledge[i].Slug < knowledge[j].Slug })
	return knowledge, nil
}

// GetKnowledge returns one entry by slug including body.
// Returns error if sensitivity == "restricted".
func (s Store) GetKnowledge(slug string) (*KnowledgeEntry, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.knowledgeDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pose: knowledge entry %q not found", slug)
		}
		return nil, fmt.Errorf("pose: reading knowledge dir: %w", err)
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		path := filepath.Join(s.knowledgeDir(), e.Name())
		ke, err := parseKnowledgeFile(path, true)
		if err != nil {
			continue
		}
		if ke.Slug == slug {
			if ke.Sensitivity == "restricted" {
				return nil, fmt.Errorf("pose: knowledge entry %q has restricted sensitivity", slug)
			}
			return ke, nil
		}
	}
	return nil, fmt.Errorf("pose: knowledge entry %q not found", slug)
}

func parseKnowledgeFile(path string, includeBody bool) (*KnowledgeEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: reading knowledge entry: %w", err)
	}
	fm, body := splitFrontmatter(string(raw))
	ke := &KnowledgeEntry{}
	for key, value := range fm {
		switch key {
		case "slug":
			ke.Slug = value
		case "type":
			ke.Type = value
		case "owner":
			ke.Owner = value
		case "sensitivity":
			ke.Sensitivity = value
		case "created_at":
			ke.CreatedAt = value
		case "expires_at":
			ke.ExpiresAt = value
		}
	}
	if includeBody {
		ke.Body = body
	}
	return ke, nil
}
