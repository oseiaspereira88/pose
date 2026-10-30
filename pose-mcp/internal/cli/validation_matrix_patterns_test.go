package cli

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// plainRunPattern is a -run pattern made only of test-name alternatives,
// each optionally anchored; other patterns pass only when listed in
// unjudgedRunPatterns.
var plainRunPattern = regexp.MustCompile(`^\^?[A-Za-z0-9_]+\$?(\|\^?[A-Za-z0-9_]+\$?)*$`)

// unjudgedRunPatterns are the checks whose -run pattern is not a plain
// alternation, with why; a new one fails until it is listed here (review
// round 1, F2/F3).
var unjudgedRunPatterns = map[string]string{
	"legacy-contract-cutoffs-integration":  "grouped alternation",
	"legacy-contract-cutoffs-reachability": "grouped alternation",
	"update-preview-integration":           "grouped alternation",
}

// goTestValueFlags are the go test flags this guard models that take a
// value; any other flag makes the check unsupported.
var goTestValueFlags = map[string]bool{"-run": true, "-count": true, "-timeout": true, "-tags": true}

// goTestBoolFlags are the go test flags this guard models without a value.
var goTestBoolFlags = map[string]bool{"-race": true, "-v": true, "-short": true}

type matrixGoTest struct {
	pattern  string
	packages []string
	tags     []string
}

// buildEnvKeys are the environment keys that change which files a check
// builds; a check setting one is not judged (review round 2, R2-B1).
var buildEnvKeys = []string{"GOFLAGS", "GOOS", "GOARCH", "CGO_ENABLED", "GOEXPERIMENT"}

// parseMatrixGoTest reads a go test check the way go test does, for the
// flags it models: -flag value and -flag=value, packages as relative paths,
// import paths or the module's own directory by default (review round 1, B1).
func parseMatrixGoTest(args []string) (matrixGoTest, error) {
	var check matrixGoTest
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			check.packages = append(check.packages, arg)
			continue
		}
		flag, value, inline := strings.Cut(strings.Replace(arg, "-test.", "-", 1), "=")
		switch {
		case goTestBoolFlags[flag]:
			// -race builds with the race tag (review round 2, R2-B1).
			if flag == "-race" && (!inline || value == "true") {
				check.tags = append(check.tags, "race")
			}
		case goTestValueFlags[flag]:
			if !inline {
				if i+1 >= len(args) {
					return check, errors.New("flag " + flag + " has no value")
				}
				i++
				value = args[i]
			}
			switch flag {
			case "-run":
				check.pattern = value
			case "-tags":
				check.tags = append(check.tags, strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })...)
			}
		default:
			return check, errors.New("flag " + arg + " is not modelled")
		}
	}
	if len(check.packages) == 0 {
		check.packages = []string{"."}
	}
	return check, nil
}

// packageDirs resolves a package argument to directories the way go test
// selects them: ./... stops at testdata, vendor, _ and . directories and at
// nested modules (review round 1, B2).
func packageDirs(moduleDir, modulePath, pkg string) ([]string, error) {
	switch {
	case modulePath != "" && pkg == modulePath:
		pkg = "."
	case modulePath != "" && strings.HasPrefix(pkg, modulePath+"/"):
		pkg = "./" + strings.TrimPrefix(pkg, modulePath+"/")
	case !strings.HasPrefix(pkg, "."):
		return nil, errors.New("package " + pkg + " is outside the module")
	}
	recursive := pkg == "./..." || strings.HasSuffix(pkg, "/...")
	dir := filepath.Join(moduleDir, filepath.FromSlash(strings.TrimSuffix(strings.TrimSuffix(pkg, "..."), "/")))
	// The directory must be inside the module and not inside a nested one
	// (review round 2, R2-B3).
	rel, err := filepath.Rel(moduleDir, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("package " + pkg + " is outside the module")
	}
	for inner := dir; inner != filepath.Clean(moduleDir); inner = filepath.Dir(inner) {
		if _, err := os.Stat(filepath.Join(inner, "go.mod")); err == nil {
			return nil, errors.New("package " + pkg + " is in a nested module")
		}
		if filepath.Dir(inner) == inner {
			break
		}
	}
	if !recursive {
		return []string{dir}, nil
	}
	var dirs []string
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != dir {
			base := entry.Name()
			if base == "testdata" || base == "vendor" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}
		dirs = append(dirs, path)
		return nil
	})
	return dirs, err
}

// runnableTests returns the names go test -run can run in dir under tags:
// in the test files go/build selects, the functions Go's isTestFunc accepts
// as tests and fuzz targets, and the examples go/doc finds with an output
// comment (review rounds 1–2, B2, F1, R2-B2). Benchmarks need -bench, and
// TestMain(*testing.M) is the hook, not a test.
func runnableTests(dir string, tags []string) ([]string, error) {
	ctx := build.Default
	ctx.BuildTags = tags
	pkg, err := ctx.ImportDir(dir, 0)
	var noGo *build.NoGoError
	if errors.As(err, &noGo) {
		return nil, nil
	}
	if err != nil && pkg == nil {
		return nil, err
	}
	var names []string
	for _, group := range [][]string{pkg.TestGoFiles, pkg.XTestGoFiles} {
		var files []*ast.File
		for _, name := range group {
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				n := fn.Name.Name
				if n == "TestMain" && isTestFunc(fn, "M") {
					continue
				}
				if goTestName(n, "Test") && isTestFunc(fn, "T") || goTestName(n, "Fuzz") && isTestFunc(fn, "F") {
					names = append(names, n)
				}
			}
		}
		for _, example := range doc.Examples(files...) {
			if example.Output != "" || example.EmptyOutput {
				names = append(names, "Example"+example.Name)
			}
		}
	}
	return names, nil
}

// goTestName is Go's rule: the prefix, then nothing or a rune that is not a
// lower-case letter.
func goTestName(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

// isTestFunc mirrors cmd/go's: no results, one parameter of type *T —
// written testing.T, or T under a dot import — for the given kind.
func isTestFunc(fn *ast.FuncDecl, kind string) bool {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 || fn.Type.Params.List == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}
	star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	switch x := star.X.(type) {
	case *ast.Ident:
		return x.Name == kind
	case *ast.SelectorExpr:
		return x.Sel.Name == kind
	}
	return false
}

type matrixCheck struct {
	Name    string            `json:"name"`
	Program string            `json:"program"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// deadRunPatterns returns, sorted and prefixed with the module, each
// alternative of a check's -run pattern that matches no runnable test in its
// packages, and each check the guard cannot judge.
func deadRunPatterns(root string, raw []byte) ([]string, error) {
	var matrix struct {
		Stacks          map[string]struct{ Checks []matrixCheck } `json:"stacks"`
		ModuleOverrides map[string]struct {
			Checks []matrixCheck `json:"checks"`
		} `json:"moduleOverrides"`
	}
	if err := json.Unmarshal(raw, &matrix); err != nil {
		return nil, err
	}
	var problems []string
	// Stack checks apply to modules this guard does not resolve; one that
	// selects tests by name is refused rather than left unread (B1).
	for stack, def := range matrix.Stacks {
		for _, check := range def.Checks {
			if check.Program != "go" || len(check.Args) == 0 || check.Args[0] != "test" {
				continue
			}
			if parsed, err := parseMatrixGoTest(check.Args); err != nil || parsed.pattern != "" {
				problems = append(problems, "stack "+stack+": "+check.Name+": a stack check selecting tests by name is not judged")
			}
		}
	}
	runnable := map[string][]string{}
	for module, override := range matrix.ModuleOverrides {
		moduleDir := filepath.Join(root, filepath.FromSlash(module))
		modulePath := ""
		if gomod, err := os.ReadFile(filepath.Join(moduleDir, "go.mod")); err == nil {
			for _, line := range strings.Split(string(gomod), "\n") {
				if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					modulePath = strings.TrimSpace(path)
				}
			}
		}
		for _, raw := range override.Checks {
			if raw.Program != "go" || len(raw.Args) == 0 || raw.Args[0] != "test" {
				continue
			}
			label := module + ": " + raw.Name
			check, err := parseMatrixGoTest(raw.Args)
			if err != nil {
				problems = append(problems, label+": "+err.Error())
				continue
			}
			if check.pattern != "" {
				for _, key := range buildEnvKeys {
					if _, set := raw.Env[key]; set {
						err = errors.New("its environment sets " + key + ", which changes the files it builds")
					}
				}
			}
			if err != nil {
				problems = append(problems, label+": "+err.Error())
				continue
			}
			if check.pattern == "" {
				continue
			}
			if !plainRunPattern.MatchString(check.pattern) {
				if _, listed := unjudgedRunPatterns[raw.Name]; !listed {
					problems = append(problems, label+": pattern "+check.pattern+" is not a plain alternation and is not listed as unjudged")
				}
				continue
			}
			var names []string
			for _, pkg := range check.packages {
				dirs, err := packageDirs(moduleDir, modulePath, pkg)
				if err != nil {
					problems = append(problems, label+": "+err.Error())
					continue
				}
				for _, dir := range dirs {
					key := dir + "\x00" + strings.Join(check.tags, ",")
					if _, ok := runnable[key]; !ok {
						found, err := runnableTests(dir, check.tags)
						if err != nil {
							return nil, err
						}
						runnable[key] = found
					}
					names = append(names, runnable[key]...)
				}
			}
			for _, alternative := range strings.Split(check.pattern, "|") {
				re := regexp.MustCompile(alternative)
				found := false
				for _, name := range names {
					if re.MatchString(name) {
						found = true
						break
					}
				}
				if !found {
					problems = append(problems, label+": "+alternative)
				}
			}
		}
	}
	sort.Strings(problems)
	return problems, nil
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
	problems, err := deadRunPatterns(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Errorf("validation checks name tests that do not exist, or cannot be judged:\n%s", strings.Join(problems, "\n"))
	}
}

// TestDeadRunPatternGuardFollowsGoSelection pins the guard to what go test
// selects (review round 1): both flag spellings, build constraints, ignored
// files, vendor and nested modules, names Go does not run, stack checks and
// unlisted non-plain patterns.
func TestDeadRunPatternGuardFollowsGoSelection(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("m/go.mod", "module example.com/m\n\ngo 1.22\n")
	write("m/a/a.go", "package a\n")
	write("m/a/a_test.go", "package a\n\nimport \"testing\"\n\nfunc TestReal(t *testing.T) {}\nfunc Testhelper(t *testing.T) {}\nfunc BenchmarkBench(b *testing.B) {}\nfunc ExampleNoOutput() {}\nfunc ExampleWithOutput() {\n\t// Output:\n}\n")
	write("m/a/tagged_test.go", "//go:build reviewnever\n\npackage a\n\nimport \"testing\"\n\nfunc TestTagged(t *testing.T) {}\n")
	write("m/a/_ignored_test.go", "package a\n\nimport \"testing\"\n\nfunc TestIgnored(t *testing.T) {}\n")
	write("m/vendor/v/v_test.go", "package v\n\nimport \"testing\"\n\nfunc TestVendored(t *testing.T) {}\n")
	write("m/nested/go.mod", "module example.com/nested\n")
	write("m/nested/n_test.go", "package nested\n\nimport \"testing\"\n\nfunc TestNested(t *testing.T) {}\n")
	write("other/go.mod", "module example.com/other\n")
	write("other/o_test.go", "package other\n\nimport \"testing\"\n\nfunc TestOther(t *testing.T) {}\n")
	write("plain/p_test.go", "package plain\n\nimport \"testing\"\n\nfunc TestPlain(t *testing.T) {}\n")
	// Round 2 (R2-B1, R2-B2): the race tag, and valid tests the first
	// version missed.
	write("m/r/norace_test.go", "//go:build !race\n\npackage r\n\nimport \"testing\"\n\nfunc TestNoRace(t *testing.T) {}\n")
	write("m/r/race_test.go", "//go:build race\n\npackage r\n\nimport \"testing\"\n\nfunc TestRaceOnly(t *testing.T) {}\n")
	write("m/d/d_test.go", "package d\n\nimport . \"testing\"\n\nfunc TestDot(t *T) {}\nfunc TestEmpty(t *T) () {}\nfunc TestMain(t *T) {}\nfunc Example_lower() {\n\t// output:\n}\n")
	write("m/h/h_test.go", "package h\n\nimport \"testing\"\n\nfunc TestMain(m *testing.M) {}\n")
	cases := []struct {
		name string
		args string
		dead bool
		env  string
	}{
		{"live", `"./a", "-run", "TestReal|WithOutput"`, false, ""},
		{"live recursive", `"./...", "-run", "TestReal"`, false, ""},
		{"live import path", `"example.com/m/a", "-run=TestReal"`, false, ""},
		{"default package is the module directory", `"-tags", "reviewnever", "-run", "TestTagged"`, true, ""},
		{"live tagged", `"./a", "-tags=reviewnever", "-run", "TestTagged"`, false, ""},
		{"equals spelling", `"./a", "-run=TestReal|Ghost"`, true, ""},
		{"excluded by build tag", `"./a", "-run", "TestTagged"`, true, ""},
		{"ignored file", `"./a", "-run", "TestIgnored"`, true, ""},
		{"vendor", `"./...", "-run", "TestVendored"`, true, ""},
		{"nested module", `"./...", "-run", "TestNested"`, true, ""},
		{"not a test name", `"./a", "-run", "Testhelper"`, true, ""},
		{"benchmark", `"./a", "-run", "BenchmarkBench"`, true, ""},
		{"example without output", `"./a", "-run", "ExampleNoOutput"`, true, ""},
		{"unmodelled flag", `"./a", "-bench", ".", "-run", "TestReal"`, true, ""},
		{"unlisted non-plain pattern", `"./a", "-run", "^TestReal$|Ghost()"`, true, ""},
		{"race excludes a !race test", `"./r", "-race", "-run", "TestNoRace"`, true, ""},
		{"race selects a race test", `"./r", "-race", "-run", "TestRaceOnly"`, false, ""},
		{"no race selects the !race test", `"./r", "-run", "TestNoRace"`, false, ""},
		{"dot-imported T, empty results, TestMain as a test, lower-case output", `"./d", "-run", "TestDot|TestEmpty|TestMain|Example_lower"`, false, ""},
		{"TestMain hook is not a test", `"./h", "-run", "TestMain"`, true, ""},
		{"build environment", `"./a", "-run", "TestReal"`, true, `"GOFLAGS": "-tags=x"`},
		{"sibling module", `"../other", "-run", "TestOther"`, true, ""},
		{"sibling directory outside the module", `"../plain", "-run", "TestPlain"`, true, ""},
		{"explicit nested module", `"./nested", "-run", "TestNested"`, true, ""},
	}
	for _, c := range cases {
		matrix := `{"moduleOverrides":{"m":{"checks":[{"name":"probe","program":"go","args":["test", ` + c.args + `],"env":{` + c.env + `}}]}}}`
		problems, err := deadRunPatterns(root, []byte(matrix))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if (len(problems) > 0) != c.dead {
			t.Errorf("%s: problems %v, want dead=%v", c.name, problems, c.dead)
		}
	}
	stack := `{"stacks":{"go":{"checks":[{"name":"probe","program":"go","args":["test","./...","-run","TestReal"]}]}}}`
	if problems, err := deadRunPatterns(root, []byte(stack)); err != nil || len(problems) == 0 {
		t.Fatalf("a stack check selecting tests by name was not refused: %v %v", problems, err)
	}
}
