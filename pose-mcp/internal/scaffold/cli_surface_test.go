package scaffold

// The pose CLI's command surface, read from its own dispatch (spec
// pose-parity-reads-the-cli-surface).
//
// Both locale parity checks need to know what a pose command is: the manual
// check to tell `check` (a command) from `when` (prose), the skill check to
// tell `review record` (a subcommand) from `review-plan spec:x` (an argument).
// Each kept its own answer — one read the manuals' list entries, the other a
// hand-written list of eight groups — and both under-reported in the quiet
// direction: `contribute` takes a subcommand and was not on the list, while
// `roadmap`, which is not a command at all, was.
//
// The CLI is the authority, and this package cannot import it (cli imports
// scaffold). So the source is parsed instead: the commands are the case labels
// of the dispatch in mainCommand, and a command is a group when its handler
// switches on its first argument.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type cliSurface struct {
	// commands are the words `pose <word>` dispatches on.
	commands map[string]bool
	// subcommands maps a group command to the words its handler accepts first.
	subcommands map[string]map[string]bool
}

func loadCLISurface(t *testing.T) cliSurface {
	t.Helper()
	dir := filepath.Join("..", "cli")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the cli package: %v", err)
	}
	fset := token.NewFileSet()
	funcs := map[string]*ast.FuncDecl{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
				funcs[fn.Name.Name] = fn
			}
		}
	}
	dispatch, ok := funcs["mainCommand"]
	if !ok {
		t.Fatal("cli.mainCommand not found: the dispatch moved, and the surface would read as empty")
	}

	surface := cliSurface{commands: map[string]bool{}, subcommands: map[string]map[string]bool{}}
	ast.Inspect(dispatch.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || !isCommandSwitch(sw) {
			return true
		}
		for _, stmt := range sw.Body.List {
			clause := stmt.(*ast.CaseClause)
			handlers := calledFunctions(clause)
			for _, label := range stringLabels(clause) {
				if strings.HasPrefix(label, "-") {
					continue
				}
				surface.commands[label] = true
				for _, h := range handlers {
					if fn, ok := funcs[h]; ok {
						for sub := range firstArgumentCases(fn) {
							if surface.subcommands[label] == nil {
								surface.subcommands[label] = map[string]bool{}
							}
							surface.subcommands[label][sub] = true
						}
					}
				}
			}
		}
		return true
	})
	// A floor, not a list: an extraction that silently stopped matching would
	// otherwise disarm both parity checks while they kept passing.
	for _, anchor := range []string{"check", "review", "release"} {
		if !surface.commands[anchor] {
			t.Fatalf("the dispatch no longer yields %q: the surface extraction is broken, not the CLI", anchor)
		}
	}
	if len(surface.subcommands["review"]) == 0 || len(surface.subcommands["release"]) == 0 {
		t.Fatal("review and release yielded no subcommands: the handler extraction is broken, not the CLI")
	}
	return surface
}

// isSubcommand reports whether `pose <cmd> <word>` names a subcommand.
func (s cliSurface) isSubcommand(cmd, word string) bool {
	return s.subcommands[cmd][word]
}

func stringLabels(clause *ast.CaseClause) []string {
	var out []string
	for _, e := range clause.List {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

func isCommandSwitch(sw *ast.SwitchStmt) bool {
	tag, ok := sw.Tag.(*ast.Ident)
	return ok && tag.Name == "cmd"
}

// calledFunctions returns the functions a dispatch case calls. A case that
// groups several commands and switches on cmd again is left to that inner
// switch, which says which handler each of them reaches — otherwise every
// command in the group would inherit every sibling's subcommands.
func calledFunctions(clause *ast.CaseClause) []string {
	var out []string
	for _, stmt := range clause.Body {
		ast.Inspect(stmt, func(n ast.Node) bool {
			if sw, ok := n.(*ast.SwitchStmt); ok && isCommandSwitch(sw) {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
			return true
		})
	}
	return out
}

// firstArgumentCases returns the word labels of every switch in fn on its
// first argument — `switch args[0]`, or on a variable assigned from it, as in
// `sub := args[0]; switch sub`. Flag labels are not subcommands.
func firstArgumentCases(fn *ast.FuncDecl) map[string]bool {
	isFirstArg := func(e ast.Expr) bool {
		idx, ok := e.(*ast.IndexExpr)
		if !ok {
			return false
		}
		lit, ok := idx.Index.(*ast.BasicLit)
		return ok && lit.Value == "0"
	}
	fromFirstArg := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
			for i, rhs := range as.Rhs {
				if id, ok := as.Lhs[i].(*ast.Ident); ok && isFirstArg(rhs) {
					fromFirstArg[id.Name] = true
				}
			}
		}
		return true
	})
	out := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || sw.Tag == nil {
			return true
		}
		id, isIdent := sw.Tag.(*ast.Ident)
		if !isFirstArg(sw.Tag) && (!isIdent || !fromFirstArg[id.Name]) {
			return true
		}
		for _, stmt := range sw.Body.List {
			for _, label := range stringLabels(stmt.(*ast.CaseClause)) {
				if label != "" && !strings.HasPrefix(label, "-") {
					out[label] = true
				}
			}
		}
		return true
	})
	return out
}
