package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
}

// Roots resolves a project_id to a project-scoped Store. The registry is rebuilt
// from the projects dir on demand: a project cloned at runtime resolves on the
// next miss (rescan), so freshly onboarded repos appear without a restart.
type Roots struct {
	cfg          RootsConfig
	mu           sync.RWMutex
	byProject    map[string]string
	lastScan     time.Time
	rescanWindow time.Duration
}

// NewRoots builds and primes the registry.
func NewRoots(cfg RootsConfig) *Roots {
	r := &Roots{cfg: cfg, rescanWindow: 2 * time.Second}
	r.rebuild()
	return r
}

func (r *Roots) rebuild() {
	m := map[string]string{}
	if r.cfg.DefaultProjectID != "" && r.cfg.DefaultRoot != "" {
		m[r.cfg.DefaultProjectID] = r.cfg.DefaultRoot
	}
	if scan, err := ScanProjectsDir(r.cfg.ProjectsDir, r.cfg.ProjectIDPrefix); err == nil {
		for k, v := range scan {
			m[k] = v
		}
	}
	for k, v := range r.cfg.Explicit {
		if k != "" && v != "" {
			m[k] = v
		}
	}
	r.mu.Lock()
	r.byProject = m
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

// StoreFor resolves the Store for a project_id. Empty -> default root. Known
// project_id -> its root. An unknown id triggers one throttled rescan (a project
// may have just been cloned) before erroring; we never silently fall back.
func (r *Roots) StoreFor(projectID string) (Store, error) {
	if projectID == "" {
		if r.cfg.DefaultRoot == "" {
			return Store{}, fmt.Errorf("no default project root configured")
		}
		return Store{Root: r.cfg.DefaultRoot}, nil
	}
	r.mu.RLock()
	root, ok := r.byProject[projectID]
	r.mu.RUnlock()
	if !ok {
		r.maybeRescan()
		r.mu.RLock()
		root, ok = r.byProject[projectID]
		r.mu.RUnlock()
	}
	if !ok {
		return Store{}, fmt.Errorf("unknown project_id %q", projectID)
	}
	return Store{Root: root}, nil
}

// Refresh forces an immediate rebuild of the registry, bypassing the rescan
// throttle (portal-workspace-scale: an explicit, authenticated push signal from
// the Conductor after onboarding, not a passive on-miss rescan).
func (r *Roots) Refresh() {
	r.rebuild()
}

// Projects returns the currently known project_ids (for diagnostics/logging).
func (r *Roots) Projects() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.byProject))
	for id := range r.byProject {
		ids = append(ids, id)
	}
	return ids
}

// ScanProjectsDir registers each immediate subdirectory of base that contains a
// .pose/ directory, under the convention project_id = prefix + <dirname>. With an
// empty prefix the dirname IS the project_id (the canonical onboarding convention:
// clones land at CRISOL_PROJECTS_DIR/<project_id>).
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
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse project roots json: %w", err)
	}
	return out, nil
}
