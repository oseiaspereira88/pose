package pose

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

// What may start git without the raw reader, each with its reason.
var directGitExceptions = map[string]string{
	"internal/pose/git_snapshot_integrity.go": "the raw reader itself",
	"internal/pose/review_attempt_ledger.go":  "runs the arguments and environment of gitCommand under its own deadline",
	"internal/usage/usage.go":                 "asks for the common directory to place telemetry, and reads no object; the package stays free of this one",
	"internal/cli/report.go":                  "a git add, which writes to the index the operator selected",
	"internal/cli/doctor.go":                  "looks git up to report whether it is installed, and starts nothing",
}

// What starts a program the source does not name, each with what that program is. A new
// one has to say here that it is not git.
var dynamicLaunches = map[string]string{
	"internal/cli/action_resolve.go":           "ssh-keygen",
	"internal/cli/delegate_wait.go":            "this binary",
	"internal/cli/delegate_watch.go":           "fzf",
	"internal/cli/maintenance.go":              "this binary",
	"internal/cli/stack_catalog.go":            "looks a tool of the stack up, and starts nothing",
	"internal/cli/validate.go":                 "a declared check of the validation matrix, an external tool of the instance",
	"internal/mcpserver/delegate_tools.go":     "this binary",
	"internal/pose/cli.go":                     "the pose executable",
	"internal/pose/delegation_hook.go":         "a declared delegation hook",
	"internal/pose/delegation_routing.go":      "a declared routing probe",
	"internal/pose/delegation_task.go":         "a declared delegated agent",
	"internal/pose/review_dispatch.go":         "a declared reviewer adapter",
	"internal/pose/review_dispatch_unix.go":    "a declared reviewer adapter",
	"internal/pose/review_dispatch_windows.go": "a declared reviewer adapter",
	"internal/pose/review_run_provenance.go":   "looks a declared reviewer adapter up, and starts nothing",
}

// launchesOf classifies each process start of a Go source: "git" when it starts git by
// name, through a shell or by looking it up, "dynamic" when the source does not name the
// program.
func launchesOf(t *testing.T, name, source string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := []string{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok || pkg.Name != "exec" {
			return true
		}
		args := call.Args
		switch selector.Sel.Name {
		case "LookPath":
		case "Command":
		case "CommandContext":
			if len(args) > 0 {
				args = args[1:]
			}
		default:
			return true
		}
		if len(args) == 0 {
			return true
		}
		literal, ok := args[0].(*ast.BasicLit)
		if !ok {
			found = append(found, "dynamic")
			return true
		}
		program, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		if program == "git" || filepath.Base(program) == "git" {
			found = append(found, "git")
			return true
		}
		for _, arg := range args[1:] {
			if text, ok := arg.(*ast.BasicLit); ok && gitInShell.MatchString(text.Value) && shells[program] {
				found = append(found, "git")
			}
		}
		return true
	})
	return found
}

var (
	gitInShell = regexp.MustCompile(`(^|[\s;&|("'` + "`" + `])git\s`)
	shells     = map[string]bool{"sh": true, "bash": true, "zsh": true, "dash": true}
)

func TestTheLaunchScanSeesEveryForm(t *testing.T) {
	for body, want := range map[string]string{
		`exec.Command("git", "status")`:                               "git",
		`exec.CommandContext(context.Background(), "git", "status")`:  "git",
		"exec.Command(\n\t\"git\",\n\t\"status\",\n)":                 "git",
		`exec.LookPath("git")`:                                        "git",
		`exec.Command("bash", "-c", "git log")`:                       "git",
		`exec.Command("sh", "-c", "cd x && git log")`:                 "git",
		`exec.Command("/usr/bin/git", "status")`:                      "git",
		`exec.Command(gitBin, "status")`:                              "dynamic",
		`exec.Command(command[0], command[1:]...)`:                    "dynamic",
		`exec.CommandContext(ctx, program, "status")`:                 "dynamic",
		`exec.Command("gh", "issue", "create", "--title", "git log")`: "",
		`exec.Command("cosign", "verify-blob")`:                       "",
	} {
		got := launchesOf(t, "probe.go", "package p\nfunc f() { _ = 0; "+strings.ReplaceAll(body, "exec.", "exec.")+" }\n")
		if want == "" && len(got) != 0 || want != "" && (len(got) != 1 || got[0] != want) {
			t.Errorf("%s: classified %v, want %q", body, got, want)
		}
	}
}

func TestEngineStartsGitOnlyThroughTheRawReader(t *testing.T) {
	root := filepath.Join("..", "..")
	seenGit, seenDynamic := map[string]bool{}, map[string]bool{}
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(filepath.ToSlash(path), "/scaffold/dist/") {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			for _, kind := range launchesOf(t, path, string(raw)) {
				switch {
				case kind == "git" && directGitExceptions[rel] == "":
					t.Errorf("%s starts git directly: use RawHistoryGit, or name the exception and its reason here", rel)
				case kind == "dynamic" && dynamicLaunches[rel] == "" && directGitExceptions[rel] == "":
					t.Errorf("%s starts a program the source does not name: say here that it is not git", rel)
				case kind == "git":
					seenGit[rel] = true
				case kind == "dynamic":
					seenDynamic[rel] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for rel := range directGitExceptions {
		if !seenGit[rel] {
			t.Errorf("the git exception for %s no longer applies", rel)
		}
	}
	for rel := range dynamicLaunches {
		if !seenDynamic[rel] {
			t.Errorf("the dynamic launch of %s no longer applies", rel)
		}
	}
}
