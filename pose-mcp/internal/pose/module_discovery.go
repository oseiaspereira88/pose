package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	moduleDiscoveryMaxEntries = 10000
	moduleDiscoveryMaxBytes   = 8 << 20
)

// ModuleManifest records the manifest path and the stack selected by its
// filename. Paths are relative to the project root and use slash separators.
type ModuleManifest struct {
	Path  string `json:"path"`
	Stack string `json:"stack"`
}

// ModuleDiscoveryInventory is the complete bounded decision input consumed by
// `pose index` when it builds repo-map.json. Keeping the scanner's decisions in
// one value lets the generation descriptor bind discovery without hashing
// unrelated source contents.
type ModuleDiscoveryInventory struct {
	Manifests     []ModuleManifest `json:"manifests"`
	Dockerfiles   []string         `json:"dockerfiles"`
	HelmCharts    []string         `json:"helm_charts"`
	Readmes       []string         `json:"readmes"`
	AndroidModule []string         `json:"android_modules"`
}

// StackForManifestFile returns the stack a manifest or marker filename
// signals, or the empty string when the name is not recognized.
func StackForManifestFile(name string) string {
	switch name {
	case "package.json":
		return "node"
	case "go.mod":
		return "go"
	case "Cargo.toml":
		return "rust"
	case "pom.xml", "build.gradle", "build.gradle.kts":
		return "java"
	case "pyproject.toml", "requirements.txt", "Pipfile", "poetry.lock", "setup.py":
		return "python"
	case "wrangler.toml", "wrangler.json", "wrangler.jsonc":
		return "cloudflare-workers"
	}
	switch filepath.Ext(name) {
	case ".sln", ".csproj", ".fsproj", ".vbproj":
		return "dotnet"
	}
	return ""
}

// ScanModuleDiscoveryIn applies the same static, gitignore, and submodule
// exclusions as the module scanners. It reports traversal, Git, and metadata
// budget failures so callers cannot publish a partial inventory as current.
func ScanModuleDiscoveryIn(files Files, root string) (ModuleDiscoveryInventory, error) {
	files = FilesOrNative(files)
	ignoredPaths, err := GitIgnoredPathsBoundedIn(files, root, "module-discovery")
	if err != nil {
		return ModuleDiscoveryInventory{}, fmt.Errorf("pose: module discovery ignore inventory: %w", err)
	}
	staticIgnored := map[string]bool{
		// .pose is the governance record (specs, review bundles and attestations, results): it
		// holds no module, and it grows with every closeout, so counting it against the bound of
		// the walk would stop an instance from ever refreshing its indexes once it is large (spec
		// pose-index-recovery-and-close-scope-consistency).
		".git": true, ".pose": true, ".qwen": true, "node_modules": true, ".gradle": true,
		"build": true, "dist": true, "target": true, "vendor": true,
		"__pycache__": true, "testdata": true, "fixture": true, "fixtures": true,
	}
	inventory := ModuleDiscoveryInventory{
		Manifests: []ModuleManifest{}, Dockerfiles: []string{}, HelmCharts: []string{},
		Readmes: []string{}, AndroidModule: []string{},
	}
	visitedEntries := 0
	traversalBytes := 0
	retainedBytes := 0
	add := func(path string) error {
		retainedBytes += len(path)*2 + 128
		if len(inventory.Manifests)+len(inventory.Dockerfiles)+len(inventory.HelmCharts)+len(inventory.Readmes)+len(inventory.AndroidModule) >= moduleDiscoveryMaxEntries || retainedBytes > moduleDiscoveryMaxBytes {
			return errGovernanceReadViewBudget
		}
		return nil
	}
	err = WalkDirIn(files, root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		visitedEntries++
		traversalBytes += len(filepath.ToSlash(path)) + len(entry.Name())*2 + len(fmt.Sprintf("%o", entry.Type())) + 128
		if visitedEntries > moduleDiscoveryMaxEntries || traversalBytes > moduleDiscoveryMaxBytes {
			return fmt.Errorf("%w: module discovery visited more than %d entries or %d bytes of paths; ignore generated or vendored directories in .gitignore", errGovernanceReadViewBudget, moduleDiscoveryMaxEntries, moduleDiscoveryMaxBytes)
		}
		if entry.IsDir() {
			if path != root && (staticIgnored[entry.Name()] || strings.HasPrefix(entry.Name(), ".venv")) {
				return filepath.SkipDir
			}
			if path != root {
				if rel, relErr := filepath.Rel(root, path); relErr != nil {
					return relErr
				} else if ignoredPaths[filepath.ToSlash(rel)+"/"] || ignoredPaths[filepath.ToSlash(rel)] {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		name := entry.Name()
		if stack := StackForManifestFile(name); stack != "" {
			if err := add(rel); err != nil {
				return err
			}
			inventory.Manifests = append(inventory.Manifests, ModuleManifest{Path: rel, Stack: stack})
		}
		if name == "Dockerfile" || strings.HasPrefix(name, "Dockerfile.") {
			if err := add(rel); err != nil {
				return err
			}
			inventory.Dockerfiles = append(inventory.Dockerfiles, rel)
		}
		if name == "Chart.yaml" {
			if err := add(rel); err != nil {
				return err
			}
			inventory.HelmCharts = append(inventory.HelmCharts, rel)
		}
		if strings.HasPrefix(strings.ToLower(name), "readme") {
			if err := add(rel); err != nil {
				return err
			}
			inventory.Readmes = append(inventory.Readmes, rel)
		}
		return nil
	})
	if err != nil {
		return ModuleDiscoveryInventory{}, fmt.Errorf("pose: module discovery walk: %w", err)
	}
	sort.Slice(inventory.Manifests, func(i, j int) bool { return inventory.Manifests[i].Path < inventory.Manifests[j].Path })
	for _, paths := range [][]string{inventory.Dockerfiles, inventory.HelmCharts, inventory.Readmes} {
		sort.Strings(paths)
	}
	javaDirs := map[string]bool{}
	for _, manifest := range inventory.Manifests {
		if manifest.Stack == "java" {
			javaDirs[filepath.ToSlash(filepath.Dir(filepath.FromSlash(manifest.Path)))] = true
		}
	}
	for dir := range javaDirs {
		if dir == "." {
			dir = ""
		}
		found := false
		for _, candidate := range []string{
			filepath.Join(root, filepath.FromSlash(dir), "src", "main", "AndroidManifest.xml"),
			filepath.Join(root, filepath.FromSlash(dir), "AndroidManifest.xml"),
		} {
			if _, statErr := files.Stat(candidate); statErr == nil {
				found = true
				break
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return ModuleDiscoveryInventory{}, fmt.Errorf("pose: module discovery Android marker %s: %w", filepath.ToSlash(candidate), statErr)
			}
		}
		if found {
			if err := add(filepath.ToSlash(dir)); err != nil {
				return ModuleDiscoveryInventory{}, fmt.Errorf("pose: module discovery inventory: %w", err)
			}
			inventory.AndroidModule = append(inventory.AndroidModule, filepath.ToSlash(dir))
		}
	}
	sort.Strings(inventory.AndroidModule)
	return inventory, nil
}

// ModuleDiscoveryBindingIn returns the digest of exactly the scanner's
// decision inventory. An unbounded or unreadable inventory is unknown.
func ModuleDiscoveryBindingIn(files Files, root string) (string, bool) {
	inventory, err := ScanModuleDiscoveryIn(files, root)
	if err != nil {
		return "", false
	}
	raw, err := json.Marshal(inventory)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), true
}

// GitIgnoredPathsBoundedIn returns Git's ignored-path decision set through
// the operation's captured Git view. It shares the same commands and recursive
// submodule policy as GitIgnoredPathsIn, and fails closed on unavailable or
// over-budget Git data in a repository.
func GitIgnoredPathsBoundedIn(files Files, root, producer string) (map[string]bool, error) {
	files = FilesOrNative(files)
	ignored := map[string]bool{}
	visited := map[string]bool{}
	ignoredMetadata := 0
	if err := walkGitIgnoredPathsBounded(files, root, "", producer, ignored, visited, &ignoredMetadata); err != nil {
		if preview := files.Preview(); preview != nil {
			preview.Incomplete("module discovery ignore inventory: " + err.Error())
		}
		return nil, err
	}
	if preview := files.Preview(); preview != nil {
		projectGitIgnored(preview, root, ignored)
	}
	return ignored, nil
}

func walkGitIgnoredPathsBounded(files Files, dir, prefix, producer string, ignored, visited map[string]bool, ignoredMetadata *int) error {
	identity := physicalIdentityIn(files, dir)
	if visited[identity] {
		return nil
	}
	visited[identity] = true
	ignoredOutput, err := captureModuleDiscoveryGit(files, dir, producer, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
	if err != nil {
		if !hasGitMetadataIn(files, dir) {
			return nil
		}
		return err
	}
	for _, entry := range strings.Split(string(ignoredOutput), "\x00") {
		if entry == "" {
			continue
		}
		normalized := prefix + filepath.ToSlash(entry)
		addIgnoredPath(ignored, ignoredMetadata, normalized)
		if strings.HasSuffix(normalized, "/") {
			addIgnoredPath(ignored, ignoredMetadata, strings.TrimSuffix(normalized, "/"))
		} else {
			addIgnoredPath(ignored, ignoredMetadata, normalized+"/")
		}
		if len(ignored) > moduleDiscoveryMaxEntries || *ignoredMetadata > moduleDiscoveryMaxBytes {
			return errGovernanceReadViewBudget
		}
	}
	stageOutput, err := captureModuleDiscoveryGit(files, dir, producer, "ls-files", "--stage", "-z")
	if err != nil {
		if !hasGitMetadataIn(files, dir) {
			return nil
		}
		return err
	}
	for _, record := range strings.Split(string(stageOutput), "\x00") {
		meta, path, ok := strings.Cut(record, "\t")
		if !ok || !strings.HasPrefix(meta, "160000 ") {
			continue
		}
		submodule := filepath.ToSlash(path)
		pathOnDisk := filepath.Join(dir, filepath.FromSlash(submodule))
		if _, statErr := files.Stat(filepath.Join(pathOnDisk, ".git")); statErr != nil {
			continue
		}
		enclosing, rootErr := resolvesToEnclosingRepositoryIn(files, pathOnDisk, producer)
		if rootErr != nil {
			return rootErr
		}
		if enclosing {
			continue
		}
		if err := walkGitIgnoredPathsBounded(files, pathOnDisk, prefix+submodule+"/", producer, ignored, visited, ignoredMetadata); err != nil {
			return err
		}
	}
	return nil
}

func addIgnoredPath(paths map[string]bool, retainedBytes *int, path string) {
	if paths[path] {
		return
	}
	paths[path] = true
	*retainedBytes += len(path)*2 + 128
}

func resolvesToEnclosingRepositoryIn(files Files, dir, producer string) (bool, error) {
	top, err := captureModuleDiscoveryGit(files, dir, producer, "rev-parse", "--show-toplevel")
	if err != nil {
		if !hasGitMetadataIn(files, dir) {
			return false, nil
		}
		return false, err
	}
	asked, err := files.EvalSymlinks(dir)
	if err != nil {
		return false, err
	}
	answered, err := files.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil {
		return false, err
	}
	return filepath.Clean(asked) != filepath.Clean(answered), nil
}

// Submodule commands belong to the containing authorized root's read set.
// Keep that root as the capture identity and select the bounded child with -C.
func captureModuleDiscoveryGit(files Files, dir, producer string, args ...string) ([]byte, error) {
	if bounded, ok := files.(interface{ GovernanceReadViewRootFor(string) string }); ok {
		root := bounded.GovernanceReadViewRootFor(dir)
		realRoot, err := files.EvalSymlinks(root)
		if err != nil { return nil, err }
		realDir, err := files.EvalSymlinks(dir)
		if err != nil { return nil, err }
		if !pathWithin(realRoot, realDir) { return nil, errGovernanceReadViewOutsideRoot }
		rel, err := filepath.Rel(root, dir)
		if err != nil { return nil, err }
		if rel != "." {
			return captureGovernanceGit(files, root, producer, append([]string{"-C", rel}, args...)...)
		}
	}
	return captureGovernanceGit(files, dir, producer, args...)
}

func physicalIdentityIn(files Files, dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	if physical, err := files.EvalSymlinks(abs); err == nil {
		return physical
	}
	return abs
}

func hasGitMetadataIn(files Files, root string) bool {
	files = FilesOrNative(files)
	if view, _ := governanceReadViewFromFiles(files); view != nil {
		return hasGitMetadataWithinReadViewIn(files, root, view)
	}
	dir, err := filepath.Abs(root)
	if err != nil {
		return true
	}
	if gitMetadataAboveIn(files, dir) {
		return true
	}
	physical, err := files.EvalSymlinks(dir)
	return err == nil && physical != dir && gitMetadataAboveIn(files, physical)
}

func hasGitMetadataWithinReadViewIn(files Files, root string, view *governanceReadView) bool {
	producer := governanceProducerFromFiles(files, "module-discovery")
	dir, err := filepath.Abs(root)
	if err != nil {
		view.markUnknown(producer, "Git metadata root could not be resolved")
		return true
	}
	boundary, err := filepath.Abs(view.root)
	if err != nil {
		view.markUnknown(producer, "project read-view root could not be resolved for Git metadata")
		return true
	}
	dir, boundary = filepath.Clean(dir), filepath.Clean(boundary)
	if !pathWithin(boundary, dir) {
		view.markUnknown(producer, "Git metadata lookup is outside the project read-view root")
		return true
	}
	for {
		found, metadataErr := gitMetadataAtBoundedIn(files, dir)
		if metadataErr != nil {
			view.markUnknown(producer, "Git metadata could not be read within the project read-view root")
			return true
		}
		if found {
			return true
		}
		if dir == boundary {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir || !pathWithin(boundary, parent) {
			break
		}
		dir = parent
	}

	// No repository marker was found in the authorized project root. A
	// captured Git query distinguishes a repository rooted within the view
	// from a valid enclosing repository outside the view; the latter is
	// unknown because its worktree is outside the operation's authorized
	// input boundary.
	top, gitErr := CaptureGovernanceGitIn(files, root, producer, 1024, "rev-parse", "--show-toplevel")
	if gitErr != nil {
		var exit *exec.ExitError
		if errors.As(gitErr, &exit) && strings.Contains(strings.ToLower(string(exit.Stderr)), "not a git repository") {
			return false
		}
		view.markUnknown(producer, "Git context could not be proved at the project root")
		return true
	}
	topPath := filepath.Clean(strings.TrimSpace(string(top)))
	if !pathWithin(boundary, topPath) {
		view.markUnknown(producer, "Git repository root is outside the project read-view root")
	}
	return true
}

func gitMetadataAtBoundedIn(files Files, dir string) (bool, error) {
	gitPath := filepath.Join(dir, ".git")
	if _, err := files.Lstat(gitPath); err == nil {
		// A directory or gitfile is enough to prove that this path intended to
		// carry Git metadata. Git commands will report corruption instead of
		// letting the index treat it as a non-Git checkout.
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	head, headErr := files.Lstat(filepath.Join(dir, "HEAD"))
	objects, objectsErr := files.Stat(filepath.Join(dir, "objects"))
	refs, refsErr := files.Stat(filepath.Join(dir, "refs"))
	for _, err := range []error{headErr, objectsErr, refsErr} {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	hasHead := headErr == nil && !head.IsDir()
	hasObjects := objectsErr == nil && objects.IsDir()
	hasRefs := refsErr == nil && refs.IsDir()
	if hasHead && (hasObjects || hasRefs) || hasObjects && hasRefs {
		return true, nil
	}
	return false, nil
}

func gitMetadataAboveIn(files Files, dir string) bool {
	for {
		if gitMetadataAtIn(files, dir) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func gitMetadataAtIn(files Files, dir string) bool {
	if isBareRepositoryIn(files, dir) {
		return true
	}
	path := filepath.Join(dir, ".git")
	info, err := files.Lstat(path)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	if !info.IsDir() {
		return true
	}
	entries, err := files.ReadDir(path)
	return err != nil || len(entries) > 0
}

func isBareRepositoryIn(files Files, dir string) bool {
	head, err := files.Lstat(filepath.Join(dir, "HEAD"))
	if err != nil || head.IsDir() {
		return false
	}
	for _, name := range []string{"objects", "refs"} {
		info, err := files.Stat(filepath.Join(dir, name))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}
