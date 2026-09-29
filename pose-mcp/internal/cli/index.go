package cli

import (
	"encoding/json"
	"fmt"
	"github.com/harne8/pose-mcp/internal/cli/cliout"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
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
	asJSON, quiet := false, false
	colorMode := cliout.ColorAuto
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--quiet":
			quiet = true
		case "--color":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose index [--json] [--quiet] [--color auto|always|never]")
			}
			i++
			parsed, ok := cliout.ParseColorMode(args[i])
			if !ok {
				render(stdout, stderr).UnknownToken("value", args[i], []string{"auto", "always", "never"})
				return 2
			}
			colorMode = parsed
		default:
			return usageError(stderr, "Usage: pose index [--json] [--quiet] [--color auto|always|never]")
		}
	}
	out := renderWithColor(stdout, stderr, colorMode)
	out.SetQuiet(quiet)
	if asJSON {
		out.RecordJSON("index")
	}
	defer func() { _ = out.FlushJSON() }()
	// fail reports why the indexes could not be written. The human line keeps
	// going to stderr as before; the machine document names the same failure.
	fail := func(code, message string) int {
		if asJSON {
			out.Finding(cliout.Finding{State: cliout.StateError, Code: code, Message: message})
			out.Verdict(cliout.Verdict{State: cliout.StateFail, Text: message})
		}
		render(stdout, stderr).Failure("pose index: " + message)
		return 1
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
	specs, _ := store.ListSpecs("", "")
	resolver, project, resolveErr := posepkg.EnvironmentArtifactResolver(root, "")
	if resolveErr != nil {
		return fail("invalid-project-configuration", "invalid-project-configuration: "+resolveErr.Error())
	}
	specMap := map[string]any{}
	edges := []map[string]string{}
	for _, s := range specs {
		if _, exists := specMap[s.Slug]; exists {
			return fail("conflicting-artifact-identity", "conflicting-artifact-identity")
		}
		identity, _ := posepkg.ParseArtifactRef("spec:" + s.Slug)
		identity.Project = project
		resolvedDeps := []posepkg.ArtifactResolution{}
		for _, raw := range s.DependsOn {
			resolvedDeps = append(resolvedDeps, resolver.Resolve(project, raw))
		}
		specMap[s.Slug] = map[string]any{"identity": identity, "dependency_resolutions": resolvedDeps, "status": s.Status, "depends_on": s.DependsOn, "priority": s.Priority, "path": relativePath(root, s.Path)}
		for _, d := range s.DependsOn {
			if ref, err := posepkg.ParseArtifactRef(d); err == nil && ref.Kind == "spec" {
				edges = append(edges, map[string]string{"from": s.Slug, "to": d})
			}
		}
	}
	roadmaps, _ := store.ListRoadmaps()
	roadmapMap := map[string]any{}
	for _, r := range roadmaps {
		federated, err := store.FederatedRoadmapAcceptance(project, r.Slug, resolver)
		if err != nil {
			return fail("federated-roadmap", fmt.Sprintf("federated roadmap %s: %v", r.Slug, err))
		}
		roadmapMap[r.Slug] = map[string]any{"status": r.Status, "created_at": r.CreatedAt, "depends_on": r.DependsOn, "consumes": r.Consumes, "milestones": r.Milestones, "federated_acceptance": federated, "path": relativePath(root, r.Path)}
	}
	deliveryGraph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		return fail("delivery-integrity", fmt.Sprintf("delivery integrity: %v", err))
	}
	releaseStatus, err := store.GetReleaseStatus("")
	if err != nil {
		return fail("release-lifecycle", fmt.Sprintf("release lifecycle: %v", err))
	}
	outputs := map[string]any{"repo-map.json": repo, "services.json": services, "packages.json": packages, "spec-graph.json": map[string]any{"schemaVersion": 1, "specs": specMap, "edges": edges}, "roadmaps.json": map[string]any{"schemaVersion": 1, "roadmaps": roadmapMap}, "delivery-integrity.json": deliveryGraph, "releases.json": releaseStatus}
	dir := filepath.Join(root, ".pose", "indexes")
	for name, value := range outputs {
		b, e := json.MarshalIndent(value, "", "  ")
		if e != nil {
			return fail("encode", fmt.Sprintf("encoding %s: %v", name, e))
		}
		b = append(b, '\n')
		if e = writeAtomic(filepath.Join(dir, name), b, 0o644); e != nil {
			return fail("write", fmt.Sprintf("writing %s: %v", name, e))
		}
	}
	out.RecordField("indexes_dir", dir)
	out.RecordCount("specs", len(specMap))
	out.RecordCount("roadmaps", len(roadmapMap))
	out.RecordCount("modules", len(modules))
	out.RecordCount("delivery_findings", len(deliveryGraph.Findings))
	if asJSON {
		out.Verdict(cliout.Verdict{State: cliout.StatePass, Text: "indexes updated"})
		return 0
	}
	// A pinned contract line; --quiet keeps it, since it is the verdict.
	fmt.Fprintf(stdout, "POSE indexes updated at %s\n", dir)
	return 0
}

func scanModules(root string) ([]indexedModule, []string, []string, []string, []string) {
	// testdata/fixture(s) hold synthetic content by convention (testdata is
	// the Go toolchain's own name for it), never a repository's real
	// deliverable modules — a fixture go.mod under an adoption-kit example
	// was otherwise discovered as a real module and invalidated closed
	// specs' review evidence on every reindex (spec
	// pose-upgrade-path-audit-fixes).
	ignored := map[string]bool{".git": true, ".qwen": true, "node_modules": true, ".gradle": true, "build": true, "dist": true, "target": true, "vendor": true, "__pycache__": true, "testdata": true, "fixture": true, "fixtures": true}
	// See discoverValidationModules (validate.go) for why a gitignored
	// subtree is excluded the same way (spec
	// pose-discovery-gitignore-and-root-alias-fix).
	gitIgnored := posepkg.GitIgnoredPaths(root)
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
			if path != root {
				if rel, relErr := filepath.Rel(root, path); relErr == nil && (gitIgnored[filepath.ToSlash(rel)+"/"] || gitIgnored[filepath.ToSlash(rel)]) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel := relativePath(root, path)
		name := e.Name()
		// Shared with discoverValidationModules (spec
		// pose-validation-scanner-consolidation) so a stack-detection fix
		// applies to both `pose index` and `pose validate`/install/init
		// instead of only whichever scanner happened to get updated.
		// "node" is renamed to this index's pre-existing "javascript" label
		// for output compatibility; every other stack name is new coverage
		// (java via build.gradle(.kts), python, dotnet, cloudflare-workers)
		// this repo-map previously silently skipped.
		lang := stackForManifestFile(name)
		if lang == "node" {
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
		lang := byDir[d]
		if lang == "java" && isAndroidModule(d) {
			lang = "android"
		}
		mods = append(mods, indexedModule{Name: filepath.Base(d), Path: filepath.ToSlash(rel), Language: lang, HasDockerfile: containsParent(dockers, rel), HasHelmChart: containsParent(charts, rel)})
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
