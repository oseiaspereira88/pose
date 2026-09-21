package pose

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// RootsConfig configures project_id -> root resolution (pose-mcp-multi-project,
// pose-mcp-runtime-refresh).
type RootsConfig struct {
	DefaultRoot      string            // root for empty project_id (legacy single-root)
	DefaultProjectID string            // also mapped to DefaultRoot when non-empty
	ProjectsDir      string            // base scanned for materialized projects
	ProjectIDPrefix  string            // dirname -> prefix+dirname (default "": dirname IS the project_id)
	Explicit         map[string]string // env override, wins over scan/default
	// StrictSelection (spec pose-mcp-project-scope-contract): when true, an
	// empty project_id in a deployment that has onboarded more than one
	// project fails closed with ProjectAmbiguousError instead of silently
	// resolving to DefaultRoot. Default false preserves existing
	// single-project stdio ergonomics and gives multi-project deployments an
	// announced deprecation window before this becomes the default.
	StrictSelection bool
}

// ProjectUnknownError means project_id was provided but does not resolve to
// any registered root, even after a rescan. It never carries a filesystem
// path — only the caller-supplied logical identifier.
type ProjectUnknownError struct{ ProjectID string }

func (e ProjectUnknownError) Error() string {
	return fmt.Sprintf("unknown project_id %q", e.ProjectID)
}

// ProjectAmbiguousError means project_id was omitted and the server cannot
// (or, in strict mode, will not) pick one root unambiguously.
//
//   - "no-default": no default root is configured at all.
//   - "multi-project-implicit": a default root is configured but more than
//     one project is registered, so omitting project_id is a deprecated,
//     strict-mode-rejectable convenience rather than a real single-project
//     deployment.
type ProjectAmbiguousError struct{ Reason string }

func (e ProjectAmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous project selection: %s (pass project_id explicitly)", e.Reason)
}

// Roots resolves a project_id to a project-scoped Store. The registry is rebuilt
// from the projects dir on demand: a project cloned at runtime resolves on the
// next miss (rescan), so freshly onboarded repos appear without a restart.
type Roots struct {
	cfg          RootsConfig
	mu           sync.RWMutex
	byProject    map[string]string
	conflicts    map[string]bool
	lastScan     time.Time
	rescanWindow time.Duration
}

// RootsContext is the path-free runtime view exposed to MCP diagnostics. It
// deliberately contains only logical identifiers and selection behavior; the
// project_id -> filesystem-root map remains private to Roots.
type RootsContext struct {
	DefaultProjectID  string    `json:"default_project_id,omitempty"`
	DefaultConfigured bool      `json:"default_configured"`
	StrictSelection   bool      `json:"strict_project_selection"`
	ProjectIDs        []string  `json:"project_ids"`
	RefreshedAt       time.Time `json:"refreshed_at"`
}

// NewRoots builds and primes the registry.
func NewRoots(cfg RootsConfig) *Roots {
	r := &Roots{cfg: cfg, rescanWindow: 2 * time.Second}
	r.rebuild()
	return r
}

func (r *Roots) rebuild() {
	m := map[string]string{}
	conflicts := map[string]bool{}
	if r.cfg.DefaultProjectID != "" && r.cfg.DefaultRoot != "" {
		m[r.cfg.DefaultProjectID] = r.cfg.DefaultRoot
	}
	if scan, err := ScanProjectsDir(r.cfg.ProjectsDir, r.cfg.ProjectIDPrefix); err == nil {
		for k, v := range scan {
			if previous, ok := m[k]; ok && !sameProjectRoot(previous, v) {
				conflicts[k] = true
			}
			m[k] = v
		}
	}
	for k, v := range r.cfg.Explicit {
		if k != "" && v != "" {
			m[k] = v
			delete(conflicts, k)
		}
	}
	r.mu.Lock()
	r.byProject = m
	r.conflicts = conflicts
	r.lastScan = time.Now()
	r.mu.Unlock()
}

// maybeRescan rebuilds the registry if the throttle window has elapsed, so a
// stream of misses for an unknown id does not trigger a readdir storm.
func (r *Roots) maybeRescan() {
	r.mu.RLock()
	stale := time.Since(r.lastScan) >= r.rescanWindow
	r.mu.RUnlock()
	if stale {
		r.rebuild()
	}
}

// StoreFor resolves the Store for a project_id. Empty -> default root (or a
// distinct ProjectAmbiguousError when no default is configured, or, under
// StrictSelection, when more than one project is registered). Known
// project_id -> its root. An unknown id triggers one throttled rescan (a
// project may have just been cloned) before returning ProjectUnknownError;
// there is no silent fallback.
func (r *Roots) StoreFor(projectID string) (Store, error) {
	if projectID == "" {
		if r.cfg.DefaultRoot == "" {
			return Store{}, ProjectAmbiguousError{Reason: "no-default"}
		}
		if r.cfg.StrictSelection && r.registeredProjectCount() > 1 {
			return Store{}, ProjectAmbiguousError{Reason: "multi-project-implicit"}
		}
		return Store{Root: r.cfg.DefaultRoot}, nil
	}
	r.mu.RLock()
	root, ok := r.byProject[projectID]
	conflict := r.conflicts[projectID]
	r.mu.RUnlock()
	if conflict {
		return Store{}, ProjectBindingConflictError{ProjectID: projectID}
	}
	if !ok {
		r.maybeRescan()
		r.mu.RLock()
		root, ok = r.byProject[projectID]
		conflict = r.conflicts[projectID]
		r.mu.RUnlock()
		if conflict {
			return Store{}, ProjectBindingConflictError{ProjectID: projectID}
		}
	}
	if !ok {
		return Store{}, ProjectUnknownError{ProjectID: projectID}
	}
	return Store{Root: root}, nil
}

func (r *Roots) registeredProjectCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byProject)
}

// Refresh forces an immediate rebuild of the registry, bypassing the rescan
// throttle (portal-workspace-scale: an explicit, authenticated push signal from
// the Conductor after onboarding, not a passive on-miss rescan).
func (r *Roots) Refresh() {
	r.rebuild()
}

// Projects returns the currently known project_ids (for diagnostics/logging).
func (r *Roots) Projects() []string {
	return r.Context().ProjectIDs
}

// Context returns a deterministic, path-free snapshot of the active registry.
// It is safe for diagnostics but still requires caller authorization before
// any ProjectIDs are returned over a transport.
func (r *Roots) Context() RootsContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.byProject))
	for id := range r.byProject {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return RootsContext{
		DefaultProjectID:  r.cfg.DefaultProjectID,
		DefaultConfigured: r.cfg.DefaultRoot != "",
		StrictSelection:   r.cfg.StrictSelection,
		ProjectIDs:        ids,
		RefreshedAt:       r.lastScan,
	}
}

// ScanProjectsDir registers each immediate subdirectory of base that contains a
// .pose/ directory, under the convention project_id = prefix + <dirname>. With an
// empty prefix the dirname IS the project_id (the canonical onboarding convention:
// clones land at HARNE8_PROJECTS_DIR/<project_id>).
func ScanProjectsDir(base, projectIDPrefix string) (map[string]string, error) {
	out := map[string]string{}
	if base == "" {
		return out, nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("scan projects dir %q: %w", base, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		root := filepath.Join(base, e.Name())
		if fi, err := os.Stat(filepath.Join(root, ".pose")); err != nil || !fi.IsDir() {
			continue
		}
		out[projectIDPrefix+e.Name()] = root
	}
	return out, nil
}

// ParseRootsJSON parses an explicit {"project_id":"/abs/root"} map (env override).
func ParseRootsJSON(s string) (map[string]string, error) {
	out := map[string]string{}
	if s == "" {
		return out, nil
	}
	decoder := json.NewDecoder(strings.NewReader(s))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("invalid-project-roots")
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid-project-roots")
		}
		id, ok := token.(string)
		if !ok || ValidateSlug(id) != nil {
			return nil, fmt.Errorf("invalid-project-id")
		}
		if _, exists := out[id]; exists {
			return nil, ProjectBindingConflictError{ProjectID: id}
		}
		var root string
		if decoder.Decode(&root) != nil || root == "" || !filepath.IsAbs(root) {
			return nil, fmt.Errorf("invalid-project-root")
		}
		out[id] = root
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("invalid-project-roots")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("invalid-project-roots")
	}
	return out, nil
}

// ProjectBindingConflictError reports identity ambiguity without exposing roots.
type ProjectBindingConflictError struct{ ProjectID string }

func (e ProjectBindingConflictError) Error() string {
	return fmt.Sprintf("conflicting project binding: %s", e.ProjectID)
}
