package cli

// A Go check that ran no test does not pass (spec
// pose-validate-refuses-empty-go-checks).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func goCheck(name string, args ...string) map[string]any {
	return map[string]any{"name": name, "program": "go", "args": append([]string{"test"}, args...), "severity": "required"}
}

func TestValidateRefusesAGoCheckThatRanNoTest(t *testing.T) {
	t.Setenv("GOFLAGS", "")
	cases := []struct {
		name    string
		check   map[string]any
		outcome string
		state   string
	}{
		{"selects nothing", goCheck("c", "./...", "-run", "TestGhost", "-count=1"), "fail", "no-tests"},
		{"inline flag", goCheck("c", "./...", "-run=TestGhost", "-count=1"), "fail", "no-tests"},
		{"verbose", goCheck("c", "./...", "-v", "-run", "TestGhost", "-count=1"), "fail", "no-tests"},
		{"package without test files alone", goCheck("c", "./notests", "-run", "TestReal", "-count=1"), "fail", "no-tests"},
		{"one package of two", goCheck("c", "./...", "-run", "TestReal", "-count=1"), "pass", ""},
		{"no pattern", goCheck("c", "./notests", "-count=1"), "pass", ""},
		{"two-dash spelling", goCheck("c", "./a", "--test.run=Ghost", "-count=1"), "fail", "no-tests"},
		{"after -args a test flag still counts", goCheck("c", "./a", "-count=1", "-args", "-test.run=Ghost"), "fail", "no-tests"},
		{"after -args another flag does not", goCheck("c", "./notests", "-count=1", "-args", "-run=Ghost"), "pass", ""},
		{"past -- nothing is a flag", goCheck("c", "./notests", "-count=1", "--", "-run=Ghost"), "pass", ""},
		{"the real pattern stands before --", goCheck("c", "./a", "-run=Ghost", "-count=1", "--", "-run="), "fail", "no-tests"},
		{"an empty pattern selects", goCheck("c", "./notests", "-run=", "-count=1"), "fail", "no-tests"},
		{"-skip removes every test", goCheck("c", "./a", "-skip=Test", "-count=1"), "fail", "no-tests"},
		{"coverage, nothing selected", goCheck("c", "./a", "-cover", "-run=Ghost", "-count=1"), "fail", "no-tests"},
		{"coverage, no test files", goCheck("c", "./notests", "-cover", "-run=Ghost", "-count=1"), "fail", "no-tests"},
		{"coverage, tests ran", goCheck("c", "./...", "-cover", "-run=TestReal", "-count=1"), "pass", ""},
		{"a look-alike line is not a result", goCheck("c", "./noisy", "-v", "-run=Ghost", "-count=1"), "fail", "no-tests"},
		{"benchmarks are not judged", goCheck("c", "./a", "-run=^$", "-bench=.", "-benchtime=1x"), "pass", ""},
		{"-list is not judged", goCheck("c", "./a", "-list=.", "-run=Ghost"), "pass", ""},
		{"GOFLAGS selects", map[string]any{"name": "c", "program": "go", "args": []string{"test", "./a", "-count=1"}, "severity": "required", "env": map[string]string{"GOFLAGS": "-run=Ghost"}}, "fail", "no-tests"},
		{"a failing test stays a failure", goCheck("c", "./...", "-run", "TestBroken", "-count=1"), "fail", ""},
		{"not go", map[string]any{"name": "c", "program": "sh", "args": []string{"-c", "echo 'ok  	x	0.1s [no tests to run]'", "-run", "x"}, "severity": "required"}, "pass", ""},
	}
	for _, c := range cases {
		root := checksFixture(t, c.check)
		dir := filepath.Join(root, "mod")
		for rel, body := range map[string]string{
			"a/a.go":          "package a\n",
			"a/a_test.go":     "package a\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestReal(t *testing.T) {}\nfunc TestBroken(t *testing.T) {\n\tif os.Getenv(\"NEVER\") == \"\" {\n\t\tt.Fatal(\"broken\")\n\t}\n}\n",
			"b/b.go":          "package b\n",
			"b/b_test.go":     "package b\n\nimport \"testing\"\n\nfunc TestOther(t *testing.T) {}\n",
			"notests/lib.go":  "package notests\n",
			"noisy/noisy.go":  "package noisy\n",
			"noisy/n_test.go": "package noisy\n\nimport (\n\t\"fmt\"\n\t\"testing\"\n)\n\nfunc init() { fmt.Println(\"ok  fake 0.001s\") }\n\nfunc TestReal(t *testing.T) {}\n",
		} {
			path := filepath.Join(dir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		code, out := runValidate(t, root, "--jobs", "1", "--json", "result.json")
		run := loadRun(t, root)
		got := run.Checks[0]
		if got.Outcome != c.outcome || got.LimitState != c.state {
			t.Fatalf("%s: %s/%s, want %s/%s (exit %d): %s", c.name, got.Outcome, got.LimitState, c.outcome, c.state, code, out)
		}
		if c.state == "no-tests" && (code != 1 || !strings.Contains(got.Output, "no test ran")) {
			t.Fatalf("%s: exit %d: %s", c.name, code, got.Output)
		}
	}

	// A cached result still says what it ran.
	root := checksFixture(t, goCheck("c", "./...", "-run", "TestGhost"))
	if err := os.WriteFile(filepath.Join(root, "mod", "x_test.go"), []byte("package x\n\nimport \"testing\"\n\nfunc TestReal(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if code, out := runValidate(t, root, "--jobs", "1", "--json", "result.json"); code != 1 {
			t.Fatalf("attempt %d: %d %s", attempt, code, out)
		}
	}

	// R3: an optional check is an optional failure.
	optional := goCheck("c", "./...", "-run", "TestGhost", "-count=1")
	optional["severity"] = "optional"
	root = checksFixture(t, optional)
	if err := os.WriteFile(filepath.Join(root, "mod", "x_test.go"), []byte("package x\n\nimport \"testing\"\n\nfunc TestReal(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runValidate(t, root, "--jobs", "1", "--json", "result.json"); code != 0 {
		t.Fatalf("optional: %d %s", code, out)
	}
	if run := loadRun(t, root); run.Outcome != "partial" || run.Counts.OptionalFailed != 1 {
		t.Fatalf("optional: %+v", run.Counts)
	}
}

// goTestSelection reads arguments as go test does; a mode whose success says
// nothing about tests is not judged.
func TestGoTestSelection(t *testing.T) {
	t.Setenv("GOFLAGS", "")
	for _, c := range []struct {
		args    []string
		pattern string
		ok      bool
	}{
		{[]string{"test", "./...", "-run", "X"}, "X", true},
		{[]string{"test", "./...", "--run=X"}, "X", true},
		{[]string{"test", "./...", "-test.run", "X"}, "X", true},
		{[]string{"test", "./...", "-run=A", "-run=B"}, "B", true},
		{[]string{"test", "./...", "-run="}, `""`, true},
		{[]string{"test", "./..."}, "", false},
		{[]string{"test", "./...", "--", "-run=X"}, "", false},
		{[]string{"test", "./...", "-args", "-run=X"}, "", false},
		{[]string{"test", "./...", "-run=X", "-c"}, "", false},
		{[]string{"test", "./...", "-run=X", "-exec=true"}, "", false},
		{[]string{"test", "-json", "./...", "-run=X"}, "", false},
		{[]string{"vet", "./...", "-run=X"}, "", false},
	} {
		pattern, ok := goTestSelection(validationCheck{Program: "go", Args: c.args})
		if pattern != c.pattern || ok != c.ok {
			t.Errorf("%v: %q %v, want %q %v", c.args, pattern, ok, c.pattern, c.ok)
		}
	}
	if _, ok := goTestSelection(validationCheck{Program: "sh", Args: []string{"test", "-run=X"}}); ok {
		t.Error("a non-Go check was read")
	}
}

// A line split across writes is read whole, and an endless line costs nothing.
func TestGoTestPackagesReadsLinesInBoundedMemory(t *testing.T) {
	var p goTestPackages
	for _, part := range []string{"ok  \tx\t0.1", "s [no tests", " to run]\r\n"} {
		p.Write([]byte(part))
	}
	if !p.ranNone() {
		t.Fatalf("a split line: %+v", p)
	}
	p.Write([]byte(strings.Repeat("x", 3*goTestLineLimit)))
	p.Write([]byte(strings.Repeat("y", 3*goTestLineLimit) + "\n"))
	if len(p.partial) != 0 {
		t.Fatalf("kept %d bytes", len(p.partial))
	}
	p.Write([]byte("ok  \ty\t0.2s\n"))
	if p.ranNone() {
		t.Fatalf("a package ran tests: %+v", p)
	}
}
