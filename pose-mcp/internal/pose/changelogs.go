package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChangelogFragment is one unreleased news fragment
// (pose-release-changelog): the user-facing record a spec earns at closeout,
// consolidated per release at cut time (conductor-release-cut).
type ChangelogFragment struct {
	Spec     string `json:"spec"`
	Category string `json:"category"`
	Breaking bool   `json:"breaking"`
	Refs     string `json:"refs,omitempty"`
	Body     string `json:"body"`
	Path     string `json:"path"`
}

// Changelog is the aggregate view: pending fragments plus consolidated
// release files (.pose/changelogs/<version>.md).
type Changelog struct {
	Unreleased []ChangelogFragment `json:"unreleased"`
	Releases   []string            `json:"releases"`
	// Version content when a specific version was requested.
	Version     string `json:"version,omitempty"`
	VersionBody string `json:"version_body,omitempty"`
}

func (s Store) changelogsDir() string { return filepath.Join(s.Root, ".pose", "changelogs") }

// GetChangelog returns the unreleased fragments and known releases; a
// non-empty version also loads that release's consolidated body.
func (s Store) GetChangelog(version string) (*Changelog, error) {
	out := &Changelog{Unreleased: []ChangelogFragment{}, Releases: []string{}}
	unreleasedDir := filepath.Join(s.changelogsDir(), "unreleased")
	if entries, err := os.ReadDir(unreleasedDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
				continue
			}
			path := filepath.Join(unreleasedDir, e.Name())
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				continue
			}
			fm, body := splitFrontmatter(string(raw))
			out.Unreleased = append(out.Unreleased, ChangelogFragment{
				Spec:     fm["spec"],
				Category: fm["category"],
				Breaking: fm["breaking"] == "true",
				Refs:     fm["refs"],
				Body:     strings.TrimSpace(stripHTMLComments(body)),
				Path:     path,
			})
		}
	}
	sort.Slice(out.Unreleased, func(i, j int) bool { return out.Unreleased[i].Spec < out.Unreleased[j].Spec })
	if entries, err := os.ReadDir(s.changelogsDir()); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
				continue
			}
			out.Releases = append(out.Releases, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	sort.Strings(out.Releases)
	if version != "" {
		if err := ValidateSlug(strings.TrimPrefix(version, "v")); err != nil && !validVersionName(version) {
			return nil, fmt.Errorf("pose: invalid changelog version %q", version)
		}
		// Defense in depth against a path-traversal-shaped version value:
		// require that the requested name is exactly one path component
		// (no separator survives Base unless the input already had none),
		// confining the read to changelogsDir() by construction rather
		// than relying solely on the substring checks above.
		if filepath.Base(version) != version {
			return nil, fmt.Errorf("pose: invalid changelog version %q", version)
		}
		raw, err := os.ReadFile(filepath.Join(s.changelogsDir(), version+".md"))
		if err != nil {
			return nil, fmt.Errorf("pose: changelog version %q not found", version)
		}
		out.Version = version
		out.VersionBody = string(raw)
	}
	return out, nil
}

// validVersionName accepts vX.Y.Z-ish names without path separators.
func validVersionName(v string) bool {
	if v == "" || strings.ContainsAny(v, "/\\") || strings.Contains(v, "..") {
		return false
	}
	return true
}

func stripHTMLComments(s string) string {
	for {
		start := strings.Index(s, "<!--")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "-->")
		if end < 0 {
			return s[:start]
		}
		s = s[:start] + s[start+end+3:]
	}
}
