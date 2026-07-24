package pose

// Docs governance manifest (spec pose-docs-governance-contract): an opt-in,
// per-project contract declaring the expected documentation structure
// (Diátaxis doc_type by default, customizable) plus a deterministic,
// offline checker (`pose docs-check`) validating structure, cross-
// references, staleness and a security scan — the same shape of gate the
// pose-dist repo already runs on its own docs-site, offered here as a
// generic, opt-in-by-presence capability (same mechanic as
// pose-capability-mechanism: absent manifest, fully valid project).
//
// Read-only per ADR-003: CheckDocs never writes anything. `docs-init`
// (the only mutation — scaffolding the manifest) lives in internal/cli.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DocsManifestSchema is the highest schema_version this build understands.
const DocsManifestSchema = 1

// DocsEntry declares one governed doc (R1).
type DocsEntry struct {
	Path    string `json:"path"`
	DocType string `json:"doc_type"` // tutorial | howto | reference | explanation | custom
	// Topics are free-text tags; Owns names code areas/components the doc
	// describes (a plain path today — "component:<id>" once a GraphForge
	// export producer exists locally, same degrade-by-absence contract as
	// the rest of this portfolio).
	Topics []string `json:"topics,omitempty"`
	Owns   []string `json:"owns,omitempty"`
	// Owner is the doc's dono (spec pose-docs-assessment-followups R2):
	// the alias a docs-review demand is assigned to when this entry
	// declares one; empty defers to the docs-review policy default.
	Owner string `json:"owner,omitempty"`
	// AppliesTo is a free-text version/product range, informational only —
	// V1 does not gate on it (see spec Decisions).
	AppliesTo string `json:"applies_to,omitempty"`
	// ReviewAfter is an absolute YYYY-MM-DD deadline. Empty defers to the
	// manifest's DefaultReviewDays counted from the doc's last touching
	// commit (same two-tier shape as the capability/project-state policies).
	ReviewAfter string `json:"review_after,omitempty"`
	// Sensitive excludes this doc from `pose docs-sync export` by default
	// (spec harne8-platform-wiki-sync Restrições) — the wiki bundle never
	// carries a sensitive doc's content unless a project explicitly
	// allowlists it (future extension); V1 ships the exclusion half.
	Sensitive bool `json:"sensitive,omitempty"`
}

// DocsManifest is `.pose/docs.json` (R1).
type DocsManifest struct {
	SchemaVersion int      `json:"schema_version"`
	Profile       string   `json:"profile,omitempty"`
	Roots         []string `json:"roots"`
	// DefaultReviewDays is the staleness fallback when an entry has no
	// ReviewAfter. Zero disables age-based staleness for such entries.
	DefaultReviewDays int `json:"default_review_days,omitempty"`
	// Severities overrides the default severity ("error"|"warning"|"off")
	// per rule name (missing, undeclared, missing_frontmatter, broken_link,
	// broken_reference, stale, security). Unset rules use the default.
	Severities map[string]string `json:"severities,omitempty"`
	Entries    []DocsEntry       `json:"entries"`
	Path       string            `json:"-"`
}

func (s Store) DocsManifestPath() string { return filepath.Join(s.Root, ".pose", "docs.json") }

func (s Store) HasDocsManifest() bool {
	_, err := os.Stat(s.DocsManifestPath())
	return err == nil
}

func (s Store) LoadDocsManifest() (*DocsManifest, error) {
	path := s.DocsManifestPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pose: docs manifest not found at %s (run `pose docs-init`)", path)
	}
	manifest, err := ParseDocsManifest(raw)
	if err != nil {
		return nil, err
	}
	manifest.Path = path
	return manifest, nil
}

// ParseDocsManifest validates structural integrity (schema, duplicate
// paths) — existence/content checks are CheckDocs's job, which is
// root-aware and needs git.
func ParseDocsManifest(raw []byte) (*DocsManifest, error) {
	var manifest DocsManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("pose: invalid docs manifest JSON: %w", err)
	}
	if manifest.SchemaVersion < 1 {
		return nil, fmt.Errorf("pose: docs manifest missing schema_version")
	}
	if manifest.SchemaVersion > DocsManifestSchema {
		return nil, fmt.Errorf("pose: docs manifest schema_version %d is newer than supported %d", manifest.SchemaVersion, DocsManifestSchema)
	}
	seen := map[string]bool{}
	for _, e := range manifest.Entries {
		if e.Path == "" {
			return nil, fmt.Errorf("pose: docs manifest has an entry with an empty path")
		}
		if seen[e.Path] {
			return nil, fmt.Errorf("pose: docs manifest has a duplicate entry path %q", e.Path)
		}
		seen[e.Path] = true
	}
	return &manifest, nil
}

// docsDefaultSeverities are conservative: structural breakage (missing
// file, broken link/reference, secret-shaped content) errors; drift
// signals that need a human read (undeclared file, thin frontmatter, age)
// warn. Rationale mirrors the capability-mechanism's evidence-vs-score
// split — a warning never blocks by itself.
var docsDefaultSeverities = map[string]string{
	"missing":             "error",
	"undeclared":          "warning",
	"missing_frontmatter": "warning",
	"broken_link":         "error",
	"broken_reference":    "error",
	"stale":               "warning",
	"security":            "error",
}

func (m *DocsManifest) severity(rule string) string {
	if m.Severities != nil {
		if v, ok := m.Severities[rule]; ok {
			switch v {
			case "error", "warning", "off":
				return v
			}
		}
	}
	return docsDefaultSeverities[rule]
}

// DocsIssue is one nominal finding: which doc, which rule, how severe, why.
type DocsIssue struct {
	Path     string `json:"path"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// DocsTotals summarizes a check run for quick reporting (CLI header,
// project-state provider, MCP tool) without re-walking Issues.
type DocsTotals struct {
	Declared   int            `json:"declared"`
	Undeclared int            `json:"undeclared"`
	Stale      int            `json:"stale"`
	ByType     map[string]int `json:"by_type"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
}

// DocsCheckResult is the versioned, JSON-stable output of CheckDocs (R5/R6).
type DocsCheckResult struct {
	SchemaVersion int         `json:"schema_version"`
	GeneratedAt   string      `json:"generated_at"`
	ManifestPath  string      `json:"manifest_path"`
	Totals        DocsTotals  `json:"totals"`
	Issues        []DocsIssue `json:"issues"`
	// ReviewPending projects .pose/docs-review.jsonl's current state
	// (spec pose-docs-assessment-followups R5) — additive, always
	// populated when the log exists, empty when it doesn't.
	ReviewPending []DocsReviewSummary `json:"review_pending,omitempty"`
}

// DocsReviewSummary is one doc's currently open review-pending triggers.
type DocsReviewSummary struct {
	Doc      string              `json:"doc"`
	Triggers []DocsReviewTrigger `json:"triggers"`
}

var docsLinkRE = regexp.MustCompile(`\]\(([^)]+)\)`)
var docsTypedRefRE = regexp.MustCompile(`\b(spec|adr|knowledge|doc|check|url|commit|component):[^\s)\]"'` + "`" + `]+`)

// CheckDocs runs every rule over the manifest against this root's current
// files and git history. Deterministic and offline except for git log
// calls (staleness), which degrade to "unknown" (never an error) when git
// is unavailable — same posture as the rest of this portfolio's git-backed
// facts (state.go's commitsSince).
func (s Store) CheckDocs(ctx context.Context, manifest *DocsManifest) DocsCheckResult {
	result := DocsCheckResult{
		SchemaVersion: DocsManifestSchema,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ManifestPath:  manifest.Path,
		Totals:        DocsTotals{ByType: map[string]int{}, Declared: len(manifest.Entries)},
	}

	add := func(rule, path, message string) {
		sev := manifest.severity(rule)
		if sev == "off" {
			return
		}
		result.Issues = append(result.Issues, DocsIssue{Path: path, Rule: rule, Severity: sev, Message: message})
		if sev == "error" {
			result.Totals.Errors++
		} else {
			result.Totals.Warnings++
		}
	}

	declared := map[string]bool{}
	for _, entry := range manifest.Entries {
		declared[filepath.ToSlash(entry.Path)] = true
		result.Totals.ByType[entry.DocType]++

		if !localArtifactExists(s.Root, ".", entry.Path) {
			add("missing", entry.Path, fmt.Sprintf("declared doc %q not found under the project root", entry.Path))
			continue
		}
		full := filepath.Join(s.Root, filepath.FromSlash(entry.Path))
		raw, err := os.ReadFile(full)
		if err != nil {
			add("missing", entry.Path, fmt.Sprintf("declared doc %q could not be read: %v", entry.Path, err))
			continue
		}

		fm, _ := SplitFrontmatter(string(raw))
		if strings.TrimSpace(fm["title"]) == "" || strings.TrimSpace(fm["doc_type"]) == "" {
			add("missing_frontmatter", entry.Path, "missing required frontmatter field(s): title, doc_type")
		}

		for _, m := range docsLinkRE.FindAllStringSubmatch(string(raw), -1) {
			if brokenRelativeLink(s.Root, entry.Path, m[1]) {
				add("broken_link", entry.Path, fmt.Sprintf("relative link target %q does not resolve", m[1]))
			}
		}

		for _, m := range docsTypedRefRE.FindAllString(string(raw), -1) {
			if ok, reason := s.ResolvePointer(m); !ok {
				add("broken_reference", entry.Path, reason)
			}
		}

		for _, issue := range ScanContentSecurity(raw) {
			add("security", entry.Path, issue)
		}

		if stale, reason := s.docStaleness(ctx, entry, manifest.DefaultReviewDays); stale {
			result.Totals.Stale++
			add("stale", entry.Path, reason)
		}
	}

	for _, root := range manifest.Roots {
		for _, path := range s.walkMarkdownFiles(root) {
			if !declared[path] {
				result.Totals.Undeclared++
				add("undeclared", path, fmt.Sprintf("doc present under root %q but not declared in the manifest", root))
			}
		}
	}

	if events, err := LoadDocsReviewEvents(s.DocsReviewPath()); err == nil {
		pending := PendingDocsReviews(events)
		docs := make([]string, 0, len(pending))
		for doc := range pending {
			docs = append(docs, doc)
		}
		sort.Strings(docs)
		for _, doc := range docs {
			result.ReviewPending = append(result.ReviewPending, DocsReviewSummary{Doc: doc, Triggers: pending[doc]})
		}
	}

	return result
}

// brokenRelativeLink reports true only for same-repo relative links that
// fail to resolve. External (http/https/mailto), anchor-only (#...) and
// typed-pointer (spec:/adr:/...) targets are out of scope here — those are
// either not locally checkable or already covered by docsTypedRefRE.
func brokenRelativeLink(root, docPath, target string) bool {
	target = strings.SplitN(target, "#", 2)[0]
	if target == "" {
		return false
	}
	if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
		return false
	}
	for _, kind := range PointerKinds {
		if strings.HasPrefix(target, kind+":") {
			return false
		}
	}
	resolved := filepath.Join(filepath.Dir(docPath), filepath.FromSlash(target))
	rel, err := filepath.Rel(".", resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return true
	}
	return !localArtifactExists(root, ".", rel)
}

// docStaleness resolves an entry's deadline (its own ReviewAfter, or
// defaultDays counted from the last commit that touched it) and reports
// whether today is past it. A missing git history or an unset deadline is
// "not stale" — never a false positive from absent data.
func (s Store) docStaleness(ctx context.Context, entry DocsEntry, defaultDays int) (stale bool, reason string) {
	today := time.Now().UTC().Format("2006-01-02")
	if entry.ReviewAfter != "" {
		if _, err := time.Parse("2006-01-02", entry.ReviewAfter); err != nil {
			return false, ""
		}
		if entry.ReviewAfter < today {
			return true, fmt.Sprintf("past its review_after deadline (%s)", entry.ReviewAfter)
		}
		return false, ""
	}
	if defaultDays <= 0 {
		return false, ""
	}
	lastTouched, ok := s.lastCommitDate(ctx, entry.Path)
	if !ok {
		return false, ""
	}
	deadline := lastTouched.AddDate(0, 0, defaultDays)
	if time.Now().UTC().After(deadline) {
		return true, fmt.Sprintf("last touched %s, past the %d-day default review window", lastTouched.Format("2006-01-02"), defaultDays)
	}
	return false, ""
}

func (s Store) lastCommitDate(ctx context.Context, relPath string) (time.Time, bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", s.Root, "log", "-1", "--format=%cI", "--", relPath)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	ts := strings.TrimSpace(string(out))
	if ts == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// walkMarkdownFiles lists *.md files under root/dir (repo-relative,
// slash-separated), confined to the project root. A root that escapes the
// project (absolute path, "..") yields no files rather than an error —
// same fail-closed posture as localArtifactExists.
func (s Store) walkMarkdownFiles(dir string) []string {
	if dir == "" || strings.Contains(dir, "..") || filepath.IsAbs(dir) {
		return nil
	}
	base := filepath.Join(s.Root, filepath.FromSlash(dir))
	cleanRoot := filepath.Clean(s.Root) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(base)+string(filepath.Separator), cleanRoot) {
		return nil
	}
	var out []string
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

// docsProfileNames are the recommended `docs-init --profile` starting
// points (R4) — each just seeds a sensible default `roots` list; none is
// mandatory (spec Restrições: "recomendação pronta, nunca obrigatória").
var docsProfileNames = []string{"library", "service", "cli", "monorepo"}

func DocsProfileRoots(profile string) []string {
	switch profile {
	case "library", "service", "cli":
		return []string{"docs"}
	case "monorepo":
		return []string{"docs", "packages"}
	default:
		return []string{"docs"}
	}
}

func ValidDocsProfile(profile string) bool {
	for _, p := range docsProfileNames {
		if p == profile {
			return true
		}
	}
	return false
}
