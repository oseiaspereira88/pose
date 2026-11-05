package cli

import (
	"encoding/json"
	"errors"
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

func cmdIndexIn(files posepkg.Files, root string, args []string, stdout, stderr io.Writer) int {
	return cmdIndexInGuarded(files, root, args, stdout, stderr)
}

func cmdIndexInGuarded(files posepkg.Files, root string, args []string, stdout, stderr io.Writer) int {
	rest, flags, flagErr := splitOutputFlags(args)
	verify := len(rest) == 1 && rest[0] == "--verify"
	if flagErr != "" || (len(rest) > 0 && !verify) {
		return usageError(stderr, "Usage: pose index [--verify] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]")
	}
	gate := newGateOutput("index", flags, stdout, stderr)
	gate.root = root
	defer gate.Close()
	out := gate.r
	fail := func(code, message string) int {
		if out.Recording() {
			out.Finding(cliout.Finding{State: cliout.StateError, Code: code, Message: message})
			out.RecordVerdict(cliout.Verdict{State: cliout.StateFail, Text: message})
		}
		render(stdout, stderr).Failure("pose index: " + message)
		return 1
	}
	if verify {
		return verifyIndexGeneration(files, root, gate, out, fail)
	}
	return generateIndexGeneration(files, root, gate, out, fail)
}

// cmdIndex reads and writes the filesystem directly; cmdIndexIn takes the
// operation's files.
func cmdIndex(root string, args []string, stdout, stderr io.Writer) int {
	return cmdIndexIn(posepkg.NativeFiles(), root, args, stdout, stderr)
}

func scanModulesBoundedIn(files posepkg.Files, root string) ([]indexedModule, []string, []string, []string, []string, error) {
	inventory, err := posepkg.ScanModuleDiscoveryIn(files, root)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return indexedModulesFromInventory(root, inventory), moduleManifestPaths(inventory), inventory.Dockerfiles, inventory.HelmCharts, inventory.Readmes, nil
}

func scanModulesIn(files posepkg.Files, root string) ([]indexedModule, []string, []string, []string, []string) {
	modules, manifests, dockers, charts, readmes, err := scanModulesBoundedIn(files, root)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	return modules, manifests, dockers, charts, readmes
}

func moduleManifestPaths(inventory posepkg.ModuleDiscoveryInventory) []string {
	manifests := make([]string, 0, len(inventory.Manifests))
	for _, manifest := range inventory.Manifests {
		manifests = append(manifests, manifest.Path)
	}
	return manifests
}

func indexedModulesFromInventory(root string, inventory posepkg.ModuleDiscoveryInventory) []indexedModule {
	byDir := map[string]string{}
	for _, manifest := range inventory.Manifests {
		moduleDir := filepath.Join(root, filepath.FromSlash(filepath.Dir(manifest.Path)))
		lang := manifest.Stack
		// Preserve repo-map's existing Javascript label for package.json.
		if lang == "node" {
			lang = "javascript"
		}
		// Inventory manifests are path-sorted, matching WalkDirIn's previous
		// lexical traversal and its last-recognized-manifest-wins behavior.
		byDir[moduleDir] = lang
	}
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	mods := make([]indexedModule, 0, len(dirs))
	androidModules := make(map[string]bool, len(inventory.AndroidModule))
	for _, module := range inventory.AndroidModule {
		androidModules[module] = true
	}
	for _, d := range dirs {
		rel := relativePath(root, d)
		if rel == "." {
			rel = ""
		}
		lang := byDir[d]
		if lang == "java" && androidModules[filepath.ToSlash(rel)] {
			lang = "android"
		}
		mods = append(mods, indexedModule{Name: filepath.Base(d), Path: filepath.ToSlash(rel), Language: lang, HasDockerfile: containsParent(inventory.Dockerfiles, rel), HasHelmChart: containsParent(inventory.HelmCharts, rel)})
	}
	return mods
}

// scanModules reads and writes the filesystem directly; scanModulesIn takes the
// operation's files.
func scanModules(root string) ([]indexedModule, []string, []string, []string, []string) {
	return scanModulesIn(posepkg.NativeFiles(), root)
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
func loadModuleMetadataIn(files posepkg.Files, root string) (map[string]string, map[string]map[string]string) {
	raw, e := files.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json"))
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

// loadModuleMetadata reads and writes the filesystem directly; loadModuleMetadataIn takes the
// operation's files.
func loadModuleMetadata(root string) (map[string]string, map[string]map[string]string) {
	return loadModuleMetadataIn(posepkg.NativeFiles(), root)
}

// indexPublishCode names the failure of a generation publication for the
// machine-readable document.
func indexPublishCode(err error) string {
	switch {
	case errors.Is(err, posepkg.ErrIndexGenerationBusy), errors.Is(err, posepkg.ErrGovernanceWriteGuardBusy):
		return "index-generation-busy"
	case errors.Is(err, posepkg.ErrGovernanceWriteGuardUnsupported), errors.Is(err, posepkg.ErrGovernanceWriteGuardUnavailable),
		errors.Is(err, posepkg.ErrGovernanceWriteGuardRootBinding), errors.Is(err, posepkg.ErrGovernanceWriteGuardTooManyRoots),
		errors.Is(err, posepkg.ErrGovernanceWriteGuardInvalid), errors.Is(err, posepkg.ErrGovernanceWriteGuardReleased),
		errors.Is(err, posepkg.ErrGovernanceWriteGuardReentrant):
		return "governance-write-guard"
	case errors.Is(err, posepkg.ErrPublishConflict):
		return "publish-conflict"
	case errors.Is(err, posepkg.ErrPublishUnsynced):
		return "publish-unsynced"
	}
	return "write"
}

// verifyIndexGeneration reads the published set without writing anything. It
// passes only for a current generation whose bindings still hold; every other
// state is named, never reported as a complete set.
func verifyIndexGeneration(files posepkg.Files, root string, gate *gateOutput, out *cliout.Renderer, fail func(code, message string) int) int {
	var view posepkg.IndexGenerationView
	var stale, unknown []string
	store := posepkg.Store{Root: root, FS: files}
	captureErr := store.WithGovernanceReadViewForIndexBuild("index.verify", func(captured posepkg.Store) (bool, error) {
		view = captured.ReadIndexGeneration()
		if view.Status != posepkg.IndexGenerationCurrent {
			return false, nil
		}
		bindings, unknownInputs, requiresRevision := posepkg.IndexInputBindingsForStoreIn(captured)
		staleSet := map[string]bool{}
		for _, name := range posepkg.IndexInputNames {
			want := strings.TrimSpace(view.Descriptor.Inputs[name])
			actual := strings.TrimSpace(bindings[name])
			if want == "" {
				staleSet[name] = true
			}
			if actual == "" {
				unknown = append(unknown, name)
			} else if want != "" && want != actual {
				staleSet[name] = true
			}
		}
		unknown = append(unknown, unknownInputs...)
		for name := range staleSet {
			stale = append(stale, name)
		}
		sort.Strings(stale)
		unknown = uniqueSortedStrings(unknown)
		return requiresRevision, nil
	})
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
	if captureErr != nil {
		if errors.Is(captureErr, posepkg.ErrGovernanceReadViewChanged) {
			return fail("index-generation-read-view-changed", "index inputs changed during verification; run `pose index --verify` again")
		}
		return fail("index-generation-read-view-unknown", "index input freshness could not be proved: "+captureErr.Error()+"; run `pose index`")
	}
	if len(unknown) > 0 {
		out.RecordField("index_generation_unknown_inputs", strings.Join(unknown, ","))
		return fail("index-generation-input-unknown", "current index inputs could not be proved ("+strings.Join(unknown, ", ")+" ); run `pose index`")
	}
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
