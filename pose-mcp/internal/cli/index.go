package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	posepkg "github.com/crisol/pose-mcp/internal/pose"
)

type indexedModule struct {
	Name              string            `json:"name"`
	Path              string            `json:"path"`
	Language          string            `json:"language"`
	HasDockerfile     bool              `json:"hasDockerfile"`
	HasHelmChart      bool              `json:"hasHelmChart"`
	Owner             string            `json:"owner"`
	Criticality       string            `json:"criticality"`
	Domain            string            `json:"domain"`
	ValidationProfile string            `json:"validationProfile"`
	Metadata          map[string]string `json:"metadata"`
	MetadataStatus    map[string]any    `json:"metadataStatus"`
}

func cmdIndex(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		return usageError(stderr, "Usage: pose index")
	}
	modules, manifests, dockers, charts, readmes := scanModules(root)
	metadataDefaults, metadata := loadModuleMetadata(root)
	apps, services, packages := []indexedModule{}, []indexedModule{}, []indexedModule{}
	for _, m := range modules {
		decl := metadata[m.Path]
		defaults := map[string]string{"owner": "unknown", "criticality": "medium", "domain": "unknown", "validationProfile": "baseline"}
		for key, value := range metadataDefaults {
			if value != "" {
				defaults[key] = value
			}
		}
		missing := []string{}
		for k := range defaults {
			if v := decl[k]; v != "" {
				defaults[k] = v
			} else {
				missing = append(missing, k)
			}
		}
		sort.Strings(missing)
		m.Owner = defaults["owner"]
		m.Criticality = defaults["criticality"]
		m.Domain = defaults["domain"]
		m.ValidationProfile = defaults["validationProfile"]
		m.Metadata = defaults
		source := "declared"
		if len(decl) == 0 {
			source = "defaulted"
		} else if len(missing) > 0 {
			source = "partial"
		}
		m.MetadataStatus = map[string]any{"isComplete": len(missing) == 0 && len(decl) > 0, "source": source, "missingFields": missing}
		lower := strings.ToLower(m.Path + "/" + m.Name)
		if strings.Contains(lower, "service") || strings.Contains(lower, "/services/") {
			services = append(services, m)
		} else if strings.Contains(lower, "/app") || strings.Contains(lower, "web") || strings.Contains(lower, "portal") || strings.HasSuffix(lower, "-ui") {
			apps = append(apps, m)
		} else {
			packages = append(packages, m)
		}
	}
	repo := map[string]any{"root": ".", "apps": apps, "services": services, "packages": packages, "manifests": manifests, "dockerfiles": dockers, "helmCharts": charts, "readmes": readmes, "moduleMetadata": map[string]any{"schemaVersion": 1, "source": ".pose/indexes/module-metadata.json"}}
	store := posepkg.Store{Root: root}
	specs, _ := store.ListSpecs("")
	specMap := map[string]any{}
	edges := []map[string]string{}
	for _, s := range specs {
		specMap[s.Slug] = map[string]any{"status": s.Status, "depends_on": s.DependsOn, "priority": s.Priority, "path": relativePath(root, s.Path)}
		for _, d := range s.DependsOn {
			if !strings.HasPrefix(d, "milestone:") && !strings.HasPrefix(d, "roadmap:") {
				edges = append(edges, map[string]string{"from": s.Slug, "to": d})
			}
		}
	}
	roadmaps, _ := store.ListRoadmaps()
	roadmapMap := map[string]any{}
	for _, r := range roadmaps {
		roadmapMap[r.Slug] = map[string]any{"status": r.Status, "created_at": r.CreatedAt, "depends_on": r.DependsOn, "milestones": r.Milestones, "path": relativePath(root, r.Path)}
	}
	outputs := map[string]any{"repo-map.json": repo, "services.json": services, "packages.json": packages, "spec-graph.json": map[string]any{"schemaVersion": 1, "specs": specMap, "edges": edges}, "roadmaps.json": map[string]any{"schemaVersion": 1, "roadmaps": roadmapMap}}
	dir := filepath.Join(root, ".pose", "indexes")
	for name, value := range outputs {
		b, e := json.MarshalIndent(value, "", "  ")
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		b = append(b, '\n')
		if e = writeAtomic(filepath.Join(dir, name), b, 0o644); e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
	}
	fmt.Fprintf(stdout, "POSE indexes updated at %s\n", dir)
	return 0
}

func scanModules(root string) ([]indexedModule, []string, []string, []string, []string) {
	ignored := map[string]bool{".git": true, "node_modules": true, ".gradle": true, "build": true, "dist": true, "target": true, "vendor": true, "__pycache__": true}
	byDir := map[string]string{}
	manifests, dockers, charts, readmes := []string{}, []string{}, []string{}, []string{}
	_ = filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if path != root && (ignored[e.Name()] || strings.HasPrefix(e.Name(), ".venv")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := relativePath(root, path)
		name := e.Name()
		lang := ""
		switch name {
		case "go.mod":
			lang = "go"
		case "Cargo.toml":
			lang = "rust"
		case "pom.xml":
			lang = "java"
		case "package.json":
			lang = "javascript"
		}
		if lang != "" {
			byDir[filepath.Dir(path)] = lang
			manifests = append(manifests, rel)
		}
		if name == "Dockerfile" || strings.HasPrefix(name, "Dockerfile.") {
			dockers = append(dockers, rel)
		}
		if name == "Chart.yaml" {
			charts = append(charts, rel)
		}
		if strings.HasPrefix(strings.ToLower(name), "readme") {
			readmes = append(readmes, rel)
		}
		return nil
	})
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	mods := make([]indexedModule, 0, len(dirs))
	for _, d := range dirs {
		rel := relativePath(root, d)
		if rel == "." {
			rel = ""
		}
		mods = append(mods, indexedModule{Name: filepath.Base(d), Path: filepath.ToSlash(rel), Language: byDir[d], HasDockerfile: containsParent(dockers, rel), HasHelmChart: containsParent(charts, rel)})
	}
	for _, s := range [][]string{manifests, dockers, charts, readmes} {
		sort.Strings(s)
	}
	return mods, manifests, dockers, charts, readmes
}
func containsParent(paths []string, parent string) bool {
	for _, p := range paths {
		if filepath.ToSlash(filepath.Dir(p)) == parent {
			return true
		}
	}
	return false
}
func relativePath(root, path string) string {
	r, e := filepath.Rel(root, path)
	if e != nil {
		return path
	}
	return filepath.ToSlash(r)
}
func loadModuleMetadata(root string) (map[string]string, map[string]map[string]string) {
	raw, e := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json"))
	if e != nil {
		return map[string]string{}, map[string]map[string]string{}
	}
	var payload struct {
		Defaults map[string]string            `json:"defaults"`
		Modules  map[string]map[string]string `json:"modules"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return map[string]string{}, map[string]map[string]string{}
	}
	if payload.Defaults == nil {
		payload.Defaults = map[string]string{}
	}
	if payload.Modules == nil {
		payload.Modules = map[string]map[string]string{}
	}
	return payload.Defaults, payload.Modules
}
