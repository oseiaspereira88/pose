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
	"strconv"
	"strings"
	"testing"
	"time"
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
// Each is bound to its exact pattern: changing the pattern needs a new look
// (review round 3).
var unjudgedRunPatterns = map[string]string{
	"legacy-contract-cutoffs-integration":  "TestContractAdopt|TestLegacyKeyCutoffs|TestLoadReviewPolicyRejectsAnUnparsableRegisteredAdoptionDate|TestDoctorReports(Legacy|AShadowed)",
	"legacy-contract-cutoffs-reachability": "TestDoctorReports(Legacy|AShadowed)",
	"update-preview-integration":           "^(TestUpdatePreview|TestUpdateDryRun|TestUpdateOperation|TestStaleInstanceEngine)",
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

var positiveDecimal = regexp.MustCompile(`^[1-9][0-9]*$`)

// plainTagList is the -tags syntax judged here: comma-separated names. A
// quoted or space-separated list, which Go unquotes, is refused rather than
// modelled (review round 4, R4-B1).
var plainTagList = regexp.MustCompile(`^[A-Za-z0-9_.]*(,[A-Za-z0-9_.]+)*$`)

// buildEnvKey reports whether an environment key is one cmd/go reads — GO*
// or CGO_* — and so may change which files a check builds (GOFLAGS, GOOS,
// GOARCH, GOAMD64 and the other architecture levels, GOEXPERIMENT,
// CGO_ENABLED); a check setting one is not judged (review rounds 2 and 4).
func buildEnvKey(key string) bool {
	return strings.HasPrefix(key, "GO") || strings.HasPrefix(key, "CGO_")
}

// parseMatrixGoTest reads a go test check the way go test does, for the
// flags it models: -flag value and -flag=value, packages as relative paths,
// import paths or the module's own directory by default (review round 1, B1).
func parseMatrixGoTest(args []string) (matrixGoTest, error) {
	var check matrixGoTest
	var tags []string
	race := false
	count, counted := "", false
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			check.packages = append(check.packages, arg)
			continue
		}
		flag, value, inline := strings.Cut(arg, "=")
		// -test. names a test-binary flag; the build flags -tags and -race
		// have no such spelling (review round 6).
		if name, ok := strings.CutPrefix(flag, "-test."); ok {
			if name == "tags" || name == "race" {
				return check, errors.New("flag " + arg + " is not a go test flag")
			}
			flag = "-" + name
		}
		switch {
		case goTestBoolFlags[flag]:
			// A boolean flag takes Go's spellings and its last value counts;
			// -race builds with the race tag (review rounds 2–3).
			on := true
			if inline {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return check, errors.New("flag " + arg + " has no boolean value")
				}
				on = parsed
			}
			if flag == "-race" {
				race = on
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
			case "-count":
				// Every value must parse as Go's flag does, and only a
				// positive decimal last value is judged: zero runs no test
				// (review rounds 5–6, R5-B1, R6-B1, F6).
				// The test binary reads every occurrence again as unsigned,
				// so a negative one fails the run (review round 7, R7-B1).
				if n, err := strconv.ParseInt(value, 0, strconv.IntSize); err != nil || n < 0 {
					return check, errors.New("flag -count=" + value + " is not a count go test accepts")
				}
				count, counted = value, true
			case "-timeout":
				if _, err := time.ParseDuration(value); err != nil {
					return check, errors.New("flag -timeout=" + value + " is not a duration")
				}
			case "-run":
				check.pattern = value
			case "-tags":
				// Each -tags replaces the list before it, as go's flag does.
				if !plainTagList.MatchString(value) {
					return check, errors.New("flag -tags=" + value + " is not a plain comma-separated list")
				}
				tags = strings.FieldsFunc(value, func(r rune) bool { return r == ',' })
			}
		default:
			return check, errors.New("flag " + arg + " is not modelled")
		}
	}
	if counted && !positiveDecimal.MatchString(count) {
		return check, errors.New("flag -count=" + count + " is not a positive decimal count")
	}
	check.tags = append([]string{}, tags...)
	if race {
		check.tags = append(check.tags, "race")
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
	// A package path through a directory go test does not match —
	// testdata, vendor, _ or . names — is refused (review round 5, F7).
	for _, element := range strings.Split(pkg, "/") {
		if element == "testdata" || element == "vendor" || element != "." && element != ".." && element != "..." && (strings.HasPrefix(element, ".") || strings.HasPrefix(element, "_")) {
			return nil, errors.New("package " + pkg + " is in a directory go test does not match")
		}
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
	if err != nil {
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
				for key := range raw.Env {
					if buildEnvKey(key) {
						err = errors.New("its environment sets " + key + ", which can change the files it builds")
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
				if unjudgedRunPatterns[raw.Name] != check.pattern {
					problems = append(problems, label+": pattern "+check.pattern+" is not a plain alternation and is not listed, with this exact pattern, as unjudged")
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
// matches nothing (spec pose-validation-matrix-dead-test-patterns R2). It
// judges declarations under the matrix's own arguments and go/build's
// default context; an operator's inherited GOFLAGS or GOOS is outside it,
// and pose validate itself fails a check of which Go reports no test ran (spec
// pose-validate-refuses-empty-go-checks).
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
	write("m/a/testdata/td/td_test.go", "package td\n\nimport \"testing\"\n\nfunc TestInTestdata(t *testing.T) {}\n")
	write("m/.hidden/h_test.go", "package hidden\n\nimport \"testing\"\n\nfunc TestHidden(t *testing.T) {}\n")
	write("m/_under/u_test.go", "package under\n\nimport \"testing\"\n\nfunc TestUnder(t *testing.T) {}\n")
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
		{"race last value false", `"./r", "-race", "-race=false", "-run", "TestRaceOnly"`, true, ""},
		{"race last value false selects !race", `"./r", "-race", "-race=false", "-run", "TestNoRace"`, false, ""},
		{"race spelled 1", `"./r", "-race=1", "-run", "TestRaceOnly"`, false, ""},
		{"race with tags keeps race", `"./r", "-race", "-tags=x", "-run", "TestRaceOnly"`, false, ""},
		{"explicit race tag survives -race=false", `"./r", "-tags=race", "-race=false", "-run", "TestRaceOnly"`, false, ""},
		{"race bad boolean", `"./r", "-race=maybe", "-run", "TestRaceOnly"`, true, ""},
		{"tags last value wins", `"./a", "-tags=reviewnever", "-tags=other", "-run", "TestTagged"`, true, ""},
		{"tags last value selects", `"./a", "-tags=other", "-tags=reviewnever", "-run", "TestTagged"`, false, ""},
		{"race selects a race test", `"./r", "-race", "-run", "TestRaceOnly"`, false, ""},
		{"no race selects the !race test", `"./r", "-run", "TestNoRace"`, false, ""},
		{"dot-imported T, empty results, TestMain as a test, lower-case output", `"./d", "-run", "TestDot|TestEmpty|TestMain|Example_lower"`, false, ""},
		{"TestMain hook is not a test", `"./h", "-run", "TestMain"`, true, ""},
		{"build environment", `"./a", "-run", "TestReal"`, true, `"GOFLAGS": "-tags=x"`},
		{"architecture level", `"./a", "-run", "TestReal"`, true, `"GOAMD64": "v3"`},
		{"cgo setting", `"./a", "-run", "TestReal"`, true, `"CGO_CFLAGS": "-O0"`},
		{"other environment", `"./a", "-run", "TestReal"`, false, `"POSE_X": "1"`},
		{"zero count", `"./a", "-run", "TestReal", "-count=0"`, true, ""},
		{"zero count split and prefixed", `"./a", "-run", "TestReal", "-test.count", "0"`, true, ""},
		{"zero count last", `"./a", "-run", "TestReal", "-count=1", "-count=0"`, true, ""},
		{"positive count last", `"./a", "-run", "TestReal", "-count=0", "-count=1"`, false, ""},
		{"count spelled in hex", `"./a", "-run", "TestReal", "-count=0x1"`, true, ""},
		{"count out of range", `"./a", "-run", "TestReal", "-count=9223372036854775808"`, true, ""},
		{"earlier count out of range", `"./a", "-run", "TestReal", "-count=9223372036854775808", "-count=1"`, true, ""},
		{"empty count", `"./a", "-run", "TestReal", "-count="`, true, ""},
		{"earlier negative count", `"./a", "-run", "TestReal", "-count=-1", "-count=1"`, true, ""},
		{"earlier negative count prefixed and split", `"./a", "-run", "TestReal", "-test.count", "-0x1", "-count=1"`, true, ""},
		{"earlier negative zero", `"./a", "-run", "TestReal", "-count=-0", "-count=1"`, false, ""},
		{"zero then positive count", `"./a", "-run", "TestReal", "-count=0", "-count=2"`, false, ""},
		{"test-prefixed tags", `"./a", "-test.tags=x", "-run", "TestReal"`, true, ""},
		{"test-prefixed race", `"./a", "-test.race=true", "-run", "TestReal"`, true, ""},
		{"test-prefixed run", `"./a", "-test.run=TestReal"`, false, ""},
		{"bad timeout", `"./a", "-run", "TestReal", "-timeout=bogus"`, true, ""},
		{"good timeout", `"./a", "-run", "TestReal", "-timeout=30s"`, false, ""},
		{"rooted in testdata", `"./a/testdata/...", "-run", "TestInTestdata"`, true, ""},
		{"rooted in a hidden directory", `"./.hidden/...", "-run", "TestHidden"`, true, ""},
		{"rooted in an underscore directory", `"./_under", "-run", "TestUnder"`, true, ""},
		{"rooted in vendor", `"./vendor/v", "-run", "TestVendored"`, true, ""},
		{"quoted tags", `"./a", "-tags='other'", "-run", "TestReal"`, true, ""},
		{"space-separated tags", `"./a", "-tags", "one other", "-run", "TestReal"`, true, ""},
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
	// A listed non-plain check whose pattern changed is judged anew.
	changed := `{"moduleOverrides":{"m":{"checks":[{"name":"update-preview-integration","program":"go","args":["test","./a","-run","^(TestReal|TestGhost)"]}]}}}`
	if problems, err := deadRunPatterns(root, []byte(changed)); err != nil || len(problems) == 0 {
		t.Fatalf("a listed check whose pattern changed was not refused: %v %v", problems, err)
	}
	// A directory go/build cannot load is an error, not an empty package.
	write("m/mixed/x_test.go", "package x\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n")
	write("m/mixed/y_test.go", "package y\n\nimport \"testing\"\n\nfunc TestY(t *testing.T) {}\n")
	mixed := `{"moduleOverrides":{"m":{"checks":[{"name":"probe","program":"go","args":["test","./mixed","-run","TestX"]}]}}}`
	if _, err := deadRunPatterns(root, []byte(mixed)); err == nil {
		t.Fatal("a directory go/build cannot load was read as a package")
	}
	stack := `{"stacks":{"go":{"checks":[{"name":"probe","program":"go","args":["test","./...","-run","TestReal"]}]}}}`
	if problems, err := deadRunPatterns(root, []byte(stack)); err != nil || len(problems) == 0 {
		t.Fatalf("a stack check selecting tests by name was not refused: %v %v", problems, err)
	}
}
