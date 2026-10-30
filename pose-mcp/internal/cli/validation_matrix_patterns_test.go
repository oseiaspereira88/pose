package cli

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// plainRunPattern is a -run pattern made only of test-name alternatives,
// each optionally anchored; other patterns are not judged here.
var plainRunPattern = regexp.MustCompile(`^\^?[A-Za-z0-9_]+\$?(\|\^?[A-Za-z0-9_]+\$?)*$`)

// declaredTests returns the test, example, benchmark and fuzz functions
// declared in the _test.go files of dir, or of every directory under it for
// a pattern ending in "/...".
func declaredTests(t *testing.T, dir string) []string {
	t.Helper()
	recursive := strings.HasSuffix(dir, "/...")
	dir = strings.TrimSuffix(dir, "/...")
	var names []string
	readDir := func(d string) {
		entries, err := os.ReadDir(d)
		if err != nil {
			t.Fatalf("reading %s: %v", d, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(d, entry.Name()), nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parsing %s: %v", entry.Name(), err)
			}
			names = append(names, testFuncNames(file)...)
		}
	}
	if !recursive {
		readDir(dir)
		return names
	}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if base := entry.Name(); path != dir && (base == "testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_")) {
				return filepath.SkipDir
			}
			readDir(path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// TestValidationMatrixRunPatternsNameExistingTests keeps a check from naming
// a test that does not exist: go test -run exits 0 when an alternative
// matches nothing (spec pose-validation-matrix-dead-test-patterns R2).
func TestValidationMatrixRunPatternsNameExistingTests(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		ModuleOverrides map[string]struct {
			Checks []struct {
				Name    string   `json:"name"`
				Program string   `json:"program"`
				Args    []string `json:"args"`
			} `json:"checks"`
		} `json:"moduleOverrides"`
	}
	if err := json.Unmarshal(raw, &matrix); err != nil {
		t.Fatal(err)
	}
	judged := 0
	var dead []string
	declared := map[string][]string{}
	for module, override := range matrix.ModuleOverrides {
		for _, check := range override.Checks {
			if check.Program != "go" || len(check.Args) == 0 || check.Args[0] != "test" {
				continue
			}
			pattern := ""
			var packages []string
			for i, arg := range check.Args {
				if arg == "-run" && i+1 < len(check.Args) {
					pattern = check.Args[i+1]
				}
				if strings.HasPrefix(arg, "./") {
					packages = append(packages, arg)
				}
			}
			if pattern == "" || !plainRunPattern.MatchString(pattern) {
				continue
			}
			judged++
			var names []string
			for _, pkg := range packages {
				dir := filepath.Join(root, module, filepath.FromSlash(pkg))
				if _, ok := declared[dir]; !ok {
					declared[dir] = declaredTests(t, dir)
				}
				names = append(names, declared[dir]...)
			}
			for _, alternative := range strings.Split(pattern, "|") {
				re := regexp.MustCompile(alternative)
				found := false
				for _, name := range names {
					if re.MatchString(name) {
						found = true
						break
					}
				}
				if !found {
					dead = append(dead, check.Name+": "+alternative)
				}
			}
		}
	}
	sort.Strings(dead)
	if len(dead) > 0 {
		t.Errorf("validation checks name tests that do not exist:\n%s", strings.Join(dead, "\n"))
	}
	if judged < 100 {
		t.Fatalf("only %d checks were judged; the matrix was not read as expected", judged)
	}
}

func testFuncNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		for _, prefix := range []string{"Test", "Example", "Benchmark", "Fuzz"} {
			if strings.HasPrefix(fn.Name.Name, prefix) {
				names = append(names, fn.Name.Name)
				break
			}
		}
	}
	return names
}
