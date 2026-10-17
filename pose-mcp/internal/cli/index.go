package cli

import (
	"encoding/json"
	"errors"
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
	rest, flags, flagErr := splitOutputFlags(args)
	verify := len(rest) == 1 && rest[0] == "--verify"
	if flagErr != "" || (len(rest) > 0 && !verify) {
		return usageError(stderr, "Usage: pose index [--verify] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]")
	}
	gate := newGateOutput("index", flags, stdout, stderr)
	gate.root = root
	defer gate.Close()
	out := gate.r
	// fail reports why the indexes could not be written. The human line keeps
	// going to stderr as before; the machine document names the same failure.
	fail := func(code, message string) int {
		if out.Recording() {
			out.Finding(cliout.Finding{State: cliout.StateError, Code: code, Message: message})
			out.RecordVerdict(cliout.Verdict{State: cliout.StateFail, Text: message})
		}
		render(stdout, stderr).Failure("pose index: " + message)
		return 1
	}
	if verify {
		return verifyIndexGeneration(root, gate, out, fail)
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
	encoded := map[string][]byte{}
	for name, value := range outputs {
		var b []byte
		var e error
		if graph, ok := value.(posepkg.DeliveryIntegrityGraph); ok {
			b, e = graph.IndexJSON()
		} else {
			b, e = json.MarshalIndent(value, "", "  ")
		}
		if e != nil {
			return fail("encode", fmt.Sprintf("encoding %s: %v", name, e))
		}
		encoded[name] = append(b, '\n')
	}
	// The seven files are published as one generation (spec
	// pose-index-generation-integrity): the set becomes current only when the
	// descriptor is committed after every file.
	receipt, e := store.PublishIndexGeneration(encoded, posepkg.IndexInputBindings(root))
	if e != nil {
		switch {
		case receipt.Committed:
			// The descriptor was renamed into place: the new generation is the
			// current one, and what could not be confirmed is its durability.
			out.RecordField("index_generation", receipt.GenerationID)
			return fail(indexPublishCode(e), fmt.Sprintf("generation %s was committed but not confirmed durable: %v; run `pose index --verify`", receipt.GenerationID, e))
		case errors.Is(e, posepkg.ErrIndexGenerationBusy):
			return fail(indexPublishCode(e), fmt.Sprintf("%v; nothing was written, run `pose index` again when it finishes", e))
		}
		return fail(indexPublishCode(e), fmt.Sprintf("publishing the index generation: %v (the previous generation stays in force; run `pose index` again)", e))
	}
	out.RecordField("index_generation", receipt.GenerationID)
	if receipt.Previous != "" {
		out.RecordField("index_generation_previous", receipt.Previous)
	}
	if receipt.CleanupError != nil {
		// The generation is committed; only the retention step failed.
		out.Finding(cliout.Finding{State: cliout.StateWarning, Code: "index-generation-cleanup", Message: "retention of old index generations failed: " + receipt.CleanupError.Error()})
	}
	out.RecordField("indexes_dir", dir)
	out.RecordCount("specs", len(specMap))
	out.RecordCount("roadmaps", len(roadmapMap))
	out.RecordCount("modules", len(modules))
	out.RecordCount("delivery_findings", len(deliveryGraph.Findings))
	// A pinned contract line; --quiet keeps it, since it is the verdict.
	gate.Verdict(cliout.StatePass, "POSE indexes updated at "+dir, "indexes updated")
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

// indexPublishCode names the failure of a generation publication for the
// machine-readable document.
func indexPublishCode(err error) string {
	switch {
	case errors.Is(err, posepkg.ErrPublishConflict):
		return "publish-conflict"
	case errors.Is(err, posepkg.ErrPublishUnsynced):
		return "publish-unsynced"
	case errors.Is(err, posepkg.ErrIndexGenerationBusy):
		return "index-generation-busy"
	}
	return "write"
}

// verifyIndexGeneration reads the published set without writing anything. It
// passes only for a current generation whose bindings still hold; every other
// state is named, never reported as a complete set.
func verifyIndexGeneration(root string, gate *gateOutput, out *cliout.Renderer, fail func(code, message string) int) int {
	view := posepkg.Store{Root: root}.ReadIndexGeneration()
	out.RecordField("index_generation_status", string(view.Status))
	switch view.Status {
	case posepkg.IndexGenerationLegacy:
		return fail("index-generation-legacy", view.Reason+"; run `pose index`")
	case posepkg.IndexGenerationUnavailable:
		return fail("index-generation-unavailable", "the index set is unavailable: "+view.Reason+"; run `pose index`")
	}
	out.RecordField("index_generation", view.Descriptor.GenerationID)
	if len(view.Diverged) > 0 {
		out.RecordField("index_generation_diverged", strings.Join(view.Diverged, ","))
		return fail("index-generation-diverged", "generation "+view.Descriptor.GenerationID+" is whole, but "+strings.Join(view.Diverged, ", ")+" under .pose/indexes differ from it (edited by hand, or a publication is running); run `pose index`")
	}
	stale := view.StaleInputs(func(name string) (string, bool) { return posepkg.IndexInputBinding(root, name) })
	if len(stale) > 0 {
		out.RecordField("index_generation_stale_inputs", strings.Join(stale, ","))
		return fail("index-generation-stale", "generation "+view.Descriptor.GenerationID+" was computed from other inputs ("+strings.Join(stale, ", ")+" changed); run `pose index`")
	}
	gate.Verdict(cliout.StatePass, "index generation "+view.Descriptor.GenerationID+" is current", "index generation current")
	return 0
}

// refreshDeliveryIndex publishes a recomputed delivery graph. When the indexes were
// published as a generation it replaces that file in a new generation, so the
// descriptor keeps describing the set; before the first `pose index` there is no
// generation and the file is written as it always was. A set that cannot be
// verified is not written over: the failure is said, not hidden.
func refreshDeliveryIndex(root string, data []byte, stderr io.Writer) {
	_ = os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755)
	receipt, _, err := posepkg.Store{Root: root}.ReplaceIndexOrLegacy("delivery-integrity.json", data)
	switch {
	case err == nil:
		if receipt.CleanupError != nil {
			render(io.Discard, stderr).Failure("pose: retention of old index generations failed: " + receipt.CleanupError.Error())
		}
	case receipt.Committed:
		render(io.Discard, stderr).Failure("pose: the delivery index was published as generation " + receipt.GenerationID + ", but its durability could not be confirmed: " + err.Error() + "; run `pose index --verify`")
	default:
		render(io.Discard, stderr).Failure("pose: the delivery index was not refreshed: " + err.Error() + "; run `pose index`")
	}
}
