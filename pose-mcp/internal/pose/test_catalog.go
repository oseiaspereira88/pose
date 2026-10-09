package pose

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// TestCatalog is the set of test names a requirement trace's `test:` refs can
// resolve to (spec pose-trace-test-refs-resolve).
//
// lint-spec used to count trace refs without resolving them, so an invented
// test name passed the gate: pose-abm-design-basis cited
// TestABMDesignBasisDigestStable, which never existed. The catalog reads the
// tracked files, submodules and nested module repositories included, and records what each ecosystem names a
// test: Go test, benchmark, fuzz and example functions and t.Run subtests;
// JavaScript and TypeScript it/test/describe titles; Python test_ functions;
// Rust #[test] functions. A ref that names a tracked file also resolves, and so does a `go test -run`
// pattern that selects at least one Go test.
type TestCatalog struct {
	goTests []string
	names   map[string]bool
	slugs   map[string]bool
	files   map[string]bool
}

var (
	goTestFuncRE  = regexp.MustCompile(`(?m)^func (Test\w*|Benchmark\w+|Fuzz\w+|Example\w*)\(`)
	goSubtestRE   = regexp.MustCompile(`\bt\.Run\(\s*"([^"]+)"`)
	jsTestTitleRE = regexp.MustCompile("\\b(?:it|test|describe)(?:\\.(?:only|skip|each\\([^)]*\\)))?\\(\\s*['\"`]([^'\"`]+)['\"`]")
	pyTestFuncRE  = regexp.MustCompile(`(?m)^\s*(?:async\s+)?def (test_\w+)`)
	rustTestFnRE  = regexp.MustCompile(`#\[(?:tokio::)?test\]\s*(?:#\[[^\]]*\]\s*)*(?:pub\s+)?(?:async\s+)?fn (\w+)`)
	slugSepRE     = regexp.MustCompile(`[^a-z0-9]+`)

	testCatalogCache = struct {
		sync.Mutex
		byRoot map[string]*TestCatalog
	}{byRoot: map[string]*TestCatalog{}}
)

const testCatalogMaxFileBytes = 2 << 20

func testRefSlug(value string) string {
	return strings.Trim(slugSepRE.ReplaceAllString(strings.ToLower(value), "-"), "-")
}

// LoadTestCatalog builds the catalog for root once per process. It returns nil
// when Git cannot list the tree, so callers skip resolution rather than report
// every ref as unresolved.
func LoadTestCatalog(root string) *TestCatalog {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	testCatalogCache.Lock()
	defer testCatalogCache.Unlock()
	if catalog, ok := testCatalogCache.byRoot[abs]; ok {
		return catalog
	}
	catalog := buildTestCatalog(abs)
	testCatalogCache.byRoot[abs] = catalog
	return catalog
}

func gitTrackedFiles(dir string) ([]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "ls-files", "-z", "--recurse-submodules")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, false
	}
	files := []string{}
	for _, rel := range strings.Split(out.String(), "\x00") {
		if rel != "" {
			files = append(files, rel)
		}
	}
	return files, true
}

// nestedModuleRepositories returns, relative to root, the top level of every
// Git repository nested inside root that the validation matrix registers as a
// module. Such a repository is neither tracked nor a submodule of root (Harne8
// keeps graphforge this way), yet its tests are the ones root's checks run.
func nestedModuleRepositories(root string) []string {
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		return nil
	}
	var matrix struct {
		ModuleOverrides map[string]json.RawMessage `json:"moduleOverrides"`
	}
	if json.Unmarshal(raw, &matrix) != nil {
		return nil
	}
	seen := map[string]bool{}
	nested := []string{}
	for module := range matrix.ModuleOverrides {
		dir := filepath.Join(root, filepath.FromSlash(module))
		out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
		if err != nil {
			continue
		}
		top := strings.TrimSpace(string(out))
		rel, err := filepath.Rel(root, top)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || seen[rel] {
			continue
		}
		seen[rel] = true
		nested = append(nested, filepath.ToSlash(rel))
	}
	sort.Strings(nested)
	return nested
}

func buildTestCatalog(root string) *TestCatalog {
	files, ok := gitTrackedFiles(root)
	if !ok {
		return nil
	}
	for _, repo := range nestedModuleRepositories(root) {
		inner, ok := gitTrackedFiles(filepath.Join(root, filepath.FromSlash(repo)))
		if !ok {
			continue
		}
		for _, rel := range inner {
			files = append(files, repo+"/"+rel)
		}
	}
	catalog := &TestCatalog{names: map[string]bool{}, slugs: map[string]bool{}, files: map[string]bool{}}
	for _, rel := range files {
		catalog.files[rel] = true
		pattern := testFilePattern(rel)
		if pattern == nil {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > testCatalogMaxFileBytes {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, re := range pattern {
			for _, match := range re.FindAllStringSubmatch(string(raw), -1) {
				catalog.add(match[1])
				if re == goTestFuncRE {
					catalog.goTests = append(catalog.goTests, match[1])
				}
			}
		}
	}
	return catalog
}

// testFilePattern returns the expressions that name tests in rel, or nil when
// rel is not a test file.
func testFilePattern(rel string) []*regexp.Regexp {
	base := strings.ToLower(filepath.Base(rel))
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return []*regexp.Regexp{goTestFuncRE, goSubtestRE}
	case strings.HasSuffix(base, ".rs"):
		return []*regexp.Regexp{rustTestFnRE}
	case strings.HasSuffix(base, ".py") && (strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")):
		return []*regexp.Regexp{pyTestFuncRE}
	}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".mts"} {
		if !strings.HasSuffix(base, ext) {
			continue
		}
		stem := strings.TrimSuffix(base, ext)
		if strings.HasSuffix(stem, ".test") || strings.HasSuffix(stem, ".spec") || strings.Contains("/"+rel, "/test/") || strings.Contains("/"+rel, "/tests/") || strings.Contains("/"+rel, "/e2e/") {
			return []*regexp.Regexp{jsTestTitleRE}
		}
	}
	return nil
}

func (c *TestCatalog) add(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	c.names[name] = true
	if slug := testRefSlug(name); slug != "" {
		c.slugs[slug] = true
	}
}

// Resolves reports whether ref, the part after `test:`, names a known test or
// a tracked file. A Go subtest path resolves through its top-level test, and a
// JavaScript title may be cited in slug form.
func (c *TestCatalog) Resolves(ref string) bool {
	ref = strings.TrimRight(strings.TrimSpace(ref), ".`'\"")
	if ref == "" {
		return false
	}
	if c.names[ref] || c.files[strings.TrimPrefix(ref, "./")] {
		return true
	}
	if head, _, ok := strings.Cut(ref, "/"); ok && c.names[head] && strings.HasPrefix(head, "Test") {
		return true
	}
	if c.slugs[testRefSlug(ref)] {
		return true
	}
	// A trace often cites the -run pattern its check uses. It resolves when
	// `go test -run <ref>` would select a test, the way the check itself runs.
	if pattern, err := regexp.Compile(strings.SplitN(ref, "/", 2)[0]); err == nil {
		for _, name := range c.goTests {
			if pattern.MatchString(name) {
				return true
			}
		}
	}
	return false
}

// UnresolvedTestRefs returns, in trace order, the `test:` refs of trace that
// resolve to nothing in catalog.
func UnresolvedTestRefs(trace RequirementTrace, catalog *TestCatalog) []string {
	if catalog == nil {
		return nil
	}
	unresolved := []string{}
	seen := map[string]bool{}
	for _, requirement := range trace.Requirements {
		if requirement.Entry == nil {
			continue
		}
		for _, ref := range requirement.Entry.Refs {
			name, ok := strings.CutPrefix(ref, "test:")
			if !ok || seen[name] || catalog.Resolves(name) {
				continue
			}
			seen[name] = true
			unresolved = append(unresolved, requirement.ID+" test:"+name)
		}
	}
	return unresolved
}
