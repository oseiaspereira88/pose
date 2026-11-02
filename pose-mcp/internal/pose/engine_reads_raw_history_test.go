package pose

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestAReplacedTagTreeIsRefused(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	good := relGit(t, r.root, "rev-parse", "HEAD")
	relGit(t, r.root, "rm", "-q", "-r", ".pose/releases", ".pose/changelogs")
	r.commit(t, "tag without release snapshot")
	bad := relGit(t, r.root, "rev-parse", "HEAD")
	relGit(t, r.root, "tag", "v1.0.0")
	r.writeAll(t)
	relGit(t, r.root, "replace", bad, good)
	if _, err := ReadReleaseSnapshot(r.root, "v1.0.0"); !errors.Is(err, ErrReleaseSnapshotMissing) {
		t.Fatalf("a replace ref supplied the snapshot of a tag: %v", err)
	}
}

// What may start git without the raw reader, by file and function, each with its reason.
// The rule guards against a call written by habit: reflection or a generated program
// would pass it, and it does not read shell scripts or declared external checks.
var directGitExceptions = map[string]string{
	"internal/pose/git_snapshot_integrity.go#gitCommandContext": "the raw reader itself",
	"internal/pose/review_attempt_ledger.go#reviewHistoryGit":   "runs the arguments and environment of gitCommand under its own deadline",
	"internal/usage/usage.go#storageDir:rev-parse":              "asks for the common directory to place telemetry, and reads no object; the package stays free of this one",
	"internal/cli/report.go#runReportWith:add":                  "a git add, which writes to the index the operator selected",
	"internal/cli/doctor.go#runDoctorDiagnostics":               "looks git up to report whether it is installed, and starts nothing",
}

// What starts a program the source does not name, by file and function, each with what
// that program is. A new one has to say here that it is not git.
var dynamicLaunches = map[string]string{
	"internal/cli/action_resolve.go#sshSignNamespace":                      "ssh-keygen",
	"internal/cli/delegate_wait.go#detachDelegateRun":                      "this binary",
	"internal/cli/delegate_watch.go#pickAttempt":                           "fzf",
	"internal/cli/init_delegation.go#":                                     "the lookup of an agent CLI, as a value tests replace",
	"internal/cli/maintenance.go#runSelfUpdatedBinary":                     "this binary",
	"internal/cli/stack_catalog.go#detectStackProfiles":                    "looks a tool of the stack up, and starts nothing",
	"internal/cli/validate.go#execute":                                     "a declared check of the validation matrix, an external tool of the instance",
	"internal/mcpserver/delegate_tools.go#defaultDelegateRunner":           "this binary",
	"internal/pose/cli.go#runCLI":                                          "the pose executable",
	"internal/pose/cli.go#runCLIExit":                                      "the pose executable",
	"internal/pose/delegation_hook.go#RunDelegationTerminalHook":           "a declared delegation hook",
	"internal/pose/delegation_routing.go#DelegationAvailabilityOf":         "looks a declared agent up, and starts nothing",
	"internal/pose/delegation_routing.go#runDelegationProbe":               "a declared routing probe",
	"internal/pose/delegation_task.go#RunDelegationTask":                   "a declared delegated agent",
	"internal/pose/review_dispatch.go#RunReviewDispatchContext":            "a declared reviewer adapter",
	"internal/pose/review_dispatch_unix.go#reviewSetupCommand":             "the setup of a declared reviewer adapter",
	"internal/pose/review_dispatch_windows.go#reviewSetupCommand":          "the setup of a declared reviewer adapter",
	"internal/pose/review_run_provenance.go#provenanceExecutableSelection": "looks a declared reviewer adapter up, and starts nothing",
}

// operation is the git subcommand a literal call names, so that an exception for one
// operation does not cover another in the same function.
type processLaunch struct{ kind, function, operation string }

var (
	gitInShell = regexp.MustCompile(`(^|[\s;&|("'` + "`" + `])git\s`)
	shells     = map[string]bool{"sh": true, "bash": true, "zsh": true, "dash": true}
)

func importNames(file *ast.File, path string) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, spec := range file.Imports {
		imported, _ := strconv.Unquote(spec.Path.Value)
		if imported != path {
			continue
		}
		switch {
		case spec.Name == nil:
			names[filepath.Base(path)] = true
		case spec.Name.Name == ".":
			dot = true
		case spec.Name.Name != "_":
			names[spec.Name.Name] = true
		}
	}
	return names, dot
}

func classifyLaunch(t *testing.T, args []ast.Expr) string {
	t.Helper()
	if len(args) == 0 {
		return ""
	}
	literal, ok := args[0].(*ast.BasicLit)
	if !ok {
		return "dynamic"
	}
	program, err := strconv.Unquote(literal.Value)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(program)
	if base == "git" {
		return "git"
	}
	if !shells[base] {
		return ""
	}
	kind := ""
	for _, arg := range args[1:] {
		text, ok := arg.(*ast.BasicLit)
		switch {
		case !ok:
			kind = "dynamic"
		case gitInShell.MatchString(text.Value):
			return "git"
		}
	}
	return kind
}

// gitOperation is the first literal after the program that is not an option or its value.
func gitOperation(args []ast.Expr) string {
	skip := false
	for _, arg := range args[1:] {
		if skip {
			skip = false
			continue
		}
		literal, ok := arg.(*ast.BasicLit)
		if !ok {
			continue
		}
		text, err := strconv.Unquote(literal.Value)
		if err != nil {
			continue
		}
		if text == "-C" || text == "-c" {
			skip = true
			continue
		}
		if !strings.HasPrefix(text, "-") {
			return text
		}
	}
	return ""
}

// launchesOf lists each process start of a Go source: "git" when it starts git by name,
// through a shell or by looking it up, "dynamic" when the source does not name the
// program or takes the starter as a value.
func launchesOf(t *testing.T, name, source string) []processLaunch {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	execNames, execDot := importNames(file, "os/exec")
	osNames, _ := importNames(file, "os")
	syscallNames, _ := importNames(file, "syscall")
	found := []processLaunch{}
	walk := func(function string, root ast.Node) {
		called := map[ast.Node]bool{}
		ast.Inspect(root, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CallExpr:
				switch fun := n.Fun.(type) {
				case *ast.SelectorExpr:
					called[fun] = true
					pkg, ok := fun.X.(*ast.Ident)
					if !ok || !execNames[pkg.Name] {
						return true
					}
					args := n.Args
					switch fun.Sel.Name {
					case "CommandContext":
						if len(args) > 0 {
							args = args[1:]
						}
					case "Command", "LookPath":
					default:
						return true
					}
					if kind := classifyLaunch(t, args); kind != "" {
						found = append(found, processLaunch{kind, function, gitOperation(args)})
					}
				case *ast.Ident:
					if execDot && (fun.Name == "Command" || fun.Name == "CommandContext" || fun.Name == "LookPath") {
						found = append(found, processLaunch{"dynamic", function, ""})
					}
				}
			case *ast.CompositeLit:
				if selector, ok := n.Type.(*ast.SelectorExpr); ok && selector.Sel.Name == "Cmd" {
					if pkg, ok := selector.X.(*ast.Ident); ok && execNames[pkg.Name] {
						found = append(found, processLaunch{"dynamic", function, ""})
					}
				}
			case *ast.SelectorExpr:
				pkg, ok := n.X.(*ast.Ident)
				if !ok {
					return true
				}
				name := n.Sel.Name
				switch {
				case execNames[pkg.Name] && !called[n] && (name == "Command" || name == "CommandContext" || name == "LookPath"):
					found = append(found, processLaunch{"dynamic", function, ""})
				case osNames[pkg.Name] && name == "StartProcess", syscallNames[pkg.Name] && (name == "Exec" || name == "ForkExec" || name == "StartProcess"):
					found = append(found, processLaunch{"dynamic", function, ""})
				}
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok {
			walk(function.Name.Name, function)
			continue
		}
		walk("", decl)
	}
	return found
}

func TestTheLaunchScanSeesEveryForm(t *testing.T) {
	const plain = "import (\n\"context\"\n\"os\"\n\"os/exec\"\n\"syscall\"\n)\nvar _, _, _, _ = context.Background, os.Args, exec.ErrDot, syscall.EINVAL\n"
	for body, want := range map[string]string{
		`func f() { exec.Command("git", "status") }`:                               "git",
		`func f() { exec.CommandContext(context.Background(), "git", "status") }`:  "git",
		"func f() { exec.Command(\n\t\"git\",\n\t\"status\",\n) }":                 "git",
		`func f() { exec.LookPath("git") }`:                                        "git",
		`func f() { exec.Command("bash", "-c", "git log") }`:                       "git",
		`func f() { exec.Command("/bin/sh", "-c", "cd x && git log") }`:            "git",
		`func f() { exec.Command("/usr/bin/git", "status") }`:                      "git",
		`func f() { exec.Command(gitBin, "status") }`:                              "dynamic",
		`func f() { exec.Command(command[0], command[1:]...) }`:                    "dynamic",
		`func f() { exec.CommandContext(ctx, program, "status") }`:                 "dynamic",
		`func f() { exec.Command("sh", "-c", script) }`:                            "dynamic",
		`func f() { launch := exec.Command; launch("git") }`:                       "dynamic",
		`func f() { _ = (&exec.Cmd{Path: "/usr/bin/git"}).Run() }`:                 "dynamic",
		`func f() { syscall.Exec("/usr/bin/git", nil, nil) }`:                      "dynamic",
		`func f() { os.StartProcess("/usr/bin/git", nil, nil) }`:                   "dynamic",
		`func f() { exec.Command("gh", "issue", "create", "--title", "git log") }`: "",
		`func f() { exec.Command("cosign", "verify-blob") }`:                       "",
	} {
		got := launchesOf(t, "probe.go", "package p\n"+plain+body+"\n")
		if want == "" && len(got) != 0 || want != "" && (len(got) != 1 || got[0].kind != want) {
			t.Errorf("%s: classified %v, want %q", body, got, want)
		}
	}
	for body, want := range map[string]string{
		`func f(root string) { exec.Command("git", "-C", root, "add", "--", "x") }`: "add",
		`func f(root string) { exec.Command("git", "-C", root, "log", "-1") }`:      "log",
		`func f() { exec.Command("git", "-c", "a=b", "status") }`:                   "status",
		`func f(args []string) { exec.Command("git", args...) }`:                    "",
	} {
		got := launchesOf(t, "probe.go", "package p\n"+plain+body+"\n")
		if len(got) != 1 || got[0].operation != want {
			t.Errorf("%s: operation %v, want %q", body, got, want)
		}
	}
	aliased := launchesOf(t, "probe.go", "package p\nimport process \"os/exec\"\nfunc f() { process.Command(\"git\") }\n")
	if len(aliased) != 1 || aliased[0].kind != "git" {
		t.Errorf("an alias of os/exec hid a launch: %v", aliased)
	}
	dotted := launchesOf(t, "probe.go", "package p\nimport . \"os/exec\"\nfunc f() { Command(\"git\") }\n")
	if len(dotted) != 1 || dotted[0].kind != "dynamic" {
		t.Errorf("a dot import of os/exec hid a launch: %v", dotted)
	}
}

func TestEngineStartsGitOnlyThroughTheRawReader(t *testing.T) {
	root := filepath.Join("..", "..")
	seen := map[string]int{}
	unlisted := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); path != root && (name == "testdata" || name == "dist" || name == "vendor" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		for _, launch := range launchesOf(t, path, string(raw)) {
			site := rel + "#" + launch.function
			if launch.operation != "" && launch.kind == "git" {
				site += ":" + launch.operation
			}
			seen[site]++
			switch {
			case directGitExceptions[site] != "":
			case launch.kind == "git":
				t.Errorf("%s starts git directly: use RawHistoryGit, or name the exception and its reason here", site)
			case dynamicLaunches[site] == "":
				unlisted = append(unlisted, site)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(unlisted)
	for _, site := range unlisted {
		t.Errorf("%s starts a program the source does not name: say here that it is not git", site)
	}
	for _, listed := range []map[string]string{directGitExceptions, dynamicLaunches} {
		for site := range listed {
			if seen[site] == 0 {
				t.Errorf("the entry for %s no longer applies", site)
			}
		}
	}
	// An exception covers the one start it names: a second start in the function is
	// a new one, which the reason does not cover.
	for site := range directGitExceptions {
		if seen[site] > 1 {
			t.Errorf("%s starts git %d times under one exception", site, seen[site])
		}
	}
}
