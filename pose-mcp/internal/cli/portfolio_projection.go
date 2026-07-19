package cli

// Cross-repository portfolio projections (spec pose-cross-repo-portfolio):
// dependencies, readiness, ownership and criticality across repositories
// — reconciled locally, authority never leaves the source repository
// (Constraint). Reuses the same project-authorization allowlist the MCP
// server already uses (pose.ScanProjectsDir / POSE_PROJECT_ROOTS) so a
// projection can never silently walk the filesystem beyond registered
// projects (Security: enforce tenant/project authorization). No filesystem
// path of any project ever appears in the projection's output — only the
// logical project_id.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

type xrefResolution struct {
	Ref          string `json:"ref"`
	Resolved     bool   `json:"resolved"`
	TargetStatus string `json:"target_status,omitempty"`
	Blocking     bool   `json:"blocking"`
	Reason       string `json:"reason,omitempty"` // unauthorized-project | unknown-spec | stale-source
}

type projectedSpec struct {
	Project     string           `json:"project"`
	Slug        string           `json:"slug"`
	Status      string           `json:"status"`
	Owner       string           `json:"owner,omitempty"`
	Criticality string           `json:"criticality,omitempty"`
	Stale       bool             `json:"stale"`
	XrefsOut    []xrefResolution `json:"xrefs_out,omitempty"`
}

type portfolioTombstone struct {
	Project   string `json:"project"`
	Slug      string `json:"slug"`
	RemovedAt string `json:"removed_at"`
}

type portfolioProjection struct {
	GeneratedAt string               `json:"generated_at"`
	Projects    []string             `json:"projects"`
	Specs       []projectedSpec      `json:"specs"`
	Tombstones  []portfolioTombstone `json:"tombstones,omitempty"`
}

// discoverAuthorizedProjects mirrors the MCP server's own project
// authorization: the current repository (self), anything under a scanned
// projects dir, and any explicit POSE_PROJECT_ROOTS override — never an
// unrestricted filesystem walk.
func discoverAuthorizedProjects(root, projectsDir string) (map[string]string, error) {
	selfID := os.Getenv("POSE_DEFAULT_PROJECT_ID")
	if selfID == "" {
		selfID = "proj." + filepath.Base(root)
	}
	known := map[string]string{selfID: root}

	if projectsDir == "" {
		projectsDir = os.Getenv("HARNE8_PROJECTS_DIR")
	}
	scanned, err := posepkg.ScanProjectsDir(projectsDir, "")
	if err != nil {
		return nil, err
	}
	for id, r := range scanned {
		known[id] = r
	}

	explicit, err := posepkg.ParseRootsJSON(os.Getenv("POSE_PROJECT_ROOTS"))
	if err != nil {
		return nil, err
	}
	for id, r := range explicit {
		known[id] = r
	}
	return known, nil
}

type localSpecSummary struct {
	Slug       string
	Status     string
	DependsOn  []string
	ModifiedAt time.Time
}

func loadLocalSpecSummaries(root string) []localSpecSummary {
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*", "spec.md"))
	sort.Strings(paths)
	var out []localSpecSummary
	for _, path := range paths {
		fm, err := readFlatFrontmatter(path)
		if err != nil {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		out = append(out, localSpecSummary{
			Slug: fm["slug"], Status: fm["status"],
			DependsOn:  lintParseDependsOn(fm["depends_on"]),
			ModifiedAt: info.ModTime(),
		})
	}
	return out
}

func specStatusIn(root, slug string) (string, bool) {
	fm, err := readFlatFrontmatter(filepath.Join(root, ".pose", "specs", slug, "spec.md"))
	if err != nil {
		return "", false
	}
	return fm["status"], true
}

func buildPortfolioProjection(now time.Time, knownProjects map[string]string, maxStalenessDays int) portfolioProjection {
	projection := portfolioProjection{GeneratedAt: now.UTC().Format(time.RFC3339)}
	for id := range knownProjects {
		projection.Projects = append(projection.Projects, id)
	}
	sort.Strings(projection.Projects)

	staleness := map[string]bool{}
	for id, root := range knownProjects {
		newest := time.Time{}
		for _, s := range loadLocalSpecSummaries(root) {
			if s.ModifiedAt.After(newest) {
				newest = s.ModifiedAt
			}
		}
		staleness[id] = !newest.IsZero() && now.Sub(newest) > time.Duration(maxStalenessDays)*24*time.Hour
	}

	for _, project := range projection.Projects {
		root := knownProjects[project]
		defaults, _ := loadModuleMetadata(root)
		for _, s := range loadLocalSpecSummaries(root) {
			ps := projectedSpec{
				Project: project, Slug: s.Slug, Status: s.Status,
				Owner: defaults["owner"], Criticality: defaults["criticality"],
				Stale: staleness[project],
			}
			for _, dep := range s.DependsOn {
				if !depXrefRE.MatchString(dep) {
					continue
				}
				ref := strings.TrimPrefix(dep, "xref:")
				targetProject, targetSlug, ok := strings.Cut(ref, "/")
				res := xrefResolution{Ref: dep}
				targetRoot, authorized := knownProjects[targetProject]
				switch {
				case !ok:
					res.Reason = "unknown-spec"
				case !authorized:
					res.Reason = "unauthorized-project"
				default:
					status, found := specStatusIn(targetRoot, targetSlug)
					if !found {
						res.Reason = "unknown-spec"
					} else {
						res.Resolved = true
						res.TargetStatus = status
						res.Blocking = status != "done"
						if staleness[targetProject] {
							res.Reason = "stale-source"
						}
					}
				}
				ps.XrefsOut = append(ps.XrefsOut, res)
			}
			projection.Specs = append(projection.Specs, ps)
		}
	}
	sort.Slice(projection.Specs, func(i, j int) bool {
		if projection.Specs[i].Project != projection.Specs[j].Project {
			return projection.Specs[i].Project < projection.Specs[j].Project
		}
		return projection.Specs[i].Slug < projection.Specs[j].Slug
	})
	return projection
}

func portfolioProjectionPath(root string) string {
	return filepath.Join(root, ".pose", "reports", "portfolio-projection.json")
}

// reconcileTombstones loads the previous projection (if any) and marks
// every (project, slug) pair present before but absent now — Data/storage
// changes: revisioned projections with timestamps and tombstones, so a
// disappearance is explicit, never a silent gap.
func reconcileTombstones(root string, next portfolioProjection, now time.Time) portfolioProjection {
	raw, err := os.ReadFile(portfolioProjectionPath(root))
	if err != nil {
		return next
	}
	var previous portfolioProjection
	if json.Unmarshal(raw, &previous) != nil {
		return next
	}
	present := map[string]bool{}
	for _, s := range next.Specs {
		present[s.Project+"\x00"+s.Slug] = true
	}
	carried := map[string]portfolioTombstone{}
	for _, tomb := range previous.Tombstones {
		carried[tomb.Project+"\x00"+tomb.Slug] = tomb
	}
	for _, s := range previous.Specs {
		key := s.Project + "\x00" + s.Slug
		if present[key] {
			continue
		}
		if _, already := carried[key]; already {
			continue
		}
		carried[key] = portfolioTombstone{Project: s.Project, Slug: s.Slug, RemovedAt: now.UTC().Format(time.RFC3339)}
	}
	next.Tombstones = nil
	for _, tomb := range carried {
		if !present[tomb.Project+"\x00"+tomb.Slug] {
			next.Tombstones = append(next.Tombstones, tomb)
		}
	}
	sort.Slice(next.Tombstones, func(i, j int) bool {
		if next.Tombstones[i].Project != next.Tombstones[j].Project {
			return next.Tombstones[i].Project < next.Tombstones[j].Project
		}
		return next.Tombstones[i].Slug < next.Tombstones[j].Slug
	})
	return next
}

func cmdPortfolioProjection(root string, args []string, stdout, stderr io.Writer) int {
	projectsDir := ""
	maxStaleness := 7
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--projects-dir":
			if i+1 >= len(args) {
				return usageError(stderr, "pose portfolio-projection: --projects-dir requires a value")
			}
			i++
			projectsDir = args[i]
		case "--max-staleness-days":
			if i+1 >= len(args) {
				return usageError(stderr, "pose portfolio-projection: --max-staleness-days requires a value")
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return usageError(stderr, "pose portfolio-projection: --max-staleness-days must be a positive integer")
			}
			maxStaleness = n
		case "--json":
			jsonOut = true
		default:
			return usageError(stderr, "Usage: pose portfolio-projection [--projects-dir DIR] [--max-staleness-days N] [--json]")
		}
	}

	known, err := discoverAuthorizedProjects(root, projectsDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	now := time.Now().UTC()
	projection := buildPortfolioProjection(now, known, maxStaleness)
	projection = reconcileTombstones(root, projection, now)

	out, merr := json.MarshalIndent(projection, "", "  ")
	if merr != nil {
		fmt.Fprintln(stderr, merr)
		return 1
	}
	if err := writeAtomic(portfolioProjectionPath(root), append(out, '\n'), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if jsonOut {
		fmt.Fprintln(stdout, string(out))
		return 0
	}
	blocked, stale, unauthorized, unknown := 0, 0, 0, 0
	for _, s := range projection.Specs {
		if s.Stale {
			stale++
		}
		for _, x := range s.XrefsOut {
			if x.Blocking {
				blocked++
			}
			switch x.Reason {
			case "unauthorized-project":
				unauthorized++
			case "unknown-spec":
				unknown++
			}
		}
	}
	fmt.Fprintf(stdout, "# Cross-repository portfolio projection\n\n")
	fmt.Fprintf(stdout, "projects=%d specs=%d blocked_by_cross_repo=%d stale_sources=%d unauthorized_xrefs=%d unknown_xrefs=%d tombstones=%d\n",
		len(projection.Projects), len(projection.Specs), blocked, stale, unauthorized, unknown, len(projection.Tombstones))
	fmt.Fprintf(stdout, "\nprojection written to: %s\n", portfolioProjectionPath(root))
	return 0
}
