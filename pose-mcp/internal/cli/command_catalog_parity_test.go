package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Spec pose-help-lists-every-command.

// helpAliases are dispatched names that are another command spelled differently:
// they are not a command of their own and `pose help` does not list them twice.
var helpAliases = map[string]string{"feedback": "report-limitation"}

// mainSwitchCommands returns the command names mainCommand dispatches, read from
// the switch itself so the catalog cannot be a second copy that drifts.
func mainSwitchCommands(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "cli.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		fn, ok := node.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "mainCommand" {
			return true
		}
		for _, stmt := range fn.Body.List {
			sw, ok := stmt.(*ast.SwitchStmt)
			if !ok {
				continue
			}
			tag, ok := sw.Tag.(*ast.Ident)
			if !ok || tag.Name != "cmd" {
				continue
			}
			for _, clause := range sw.Body.List {
				for _, expr := range clause.(*ast.CaseClause).List {
					if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						name, _ := strconv.Unquote(lit.Value)
						if !strings.HasPrefix(name, "-") {
							seen[name] = true
						}
					}
				}
			}
		}
		return false
	})
	names := make([]string, 0, len(seen))
	for name := range seen {
		if _, alias := helpAliases[name]; !alias {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) < 50 {
		t.Fatalf("read only %d commands from the dispatch switch: the parser lost it", len(names))
	}
	return names
}

func setOf(names []string) map[string]bool {
	set := map[string]bool{}
	for _, name := range names {
		set[name] = true
	}
	return set
}

// R1, R2: `pose help`, in each language, lists every command the binary dispatches.
func TestPoseHelpListsEveryDispatchedCommand(t *testing.T) {
	for name, help := range map[string]string{"en": helpTextEN, "pt-BR": helpTextPtBR} {
		advertised := setOf(helpAdvertisedCommands(help))
		var missing []string
		for _, command := range mainSwitchCommands(t) {
			if !advertised[command] {
				missing = append(missing, command)
			}
		}
		if len(missing) > 0 {
			t.Errorf("pose help (%s) omits dispatched commands: %s", name, strings.Join(missing, ", "))
		}
	}
}

// R3: the two languages advertise the same commands, so a translation that lags is caught.
func TestPoseHelpAdvertisesTheSameCommandsInEveryLanguage(t *testing.T) {
	en, pt := setOf(helpAdvertisedCommands(helpTextEN)), setOf(helpAdvertisedCommands(helpTextPtBR))
	for command := range en {
		if !pt[command] {
			t.Errorf("pose help in pt-BR omits %s, which the English help lists", command)
		}
	}
	for command := range pt {
		if !en[command] {
			t.Errorf("pose help in English omits %s, which the pt-BR help lists", command)
		}
	}
}

// R4: everything the help advertises is dispatched, so the help does not promise what the binary lacks.
func TestPoseHelpAdvertisesNothingTheBinaryDoesNotDispatch(t *testing.T) {
	dispatched := setOf(mainSwitchCommands(t))
	for _, command := range helpAdvertisedCommands(helpTextEN) {
		if !dispatched[command] && command != "help" {
			t.Errorf("pose help advertises %s, which mainCommand does not dispatch", command)
		}
	}
}

// R4: every copy of the manual documents every command the help advertises. The runtime gate
// of `pose check` does the same for an instance; this holds the copies this repository ships.
func TestTheManualsDocumentEveryCommandTheHelpAdvertises(t *testing.T) {
	for _, path := range []string{
		"../../../POSE.md", "../../../locales/pt-BR/POSE.md",
		"../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if missing := CommandReferenceGaps(string(raw), helpTextEN); len(missing) > 0 {
			t.Errorf("%s does not document: %s", path, strings.Join(missing, ", "))
		}
	}
}

// R5: a command is read from the margin of its group; the continuation of a description
// that begins with a word is not a command.
func TestTheHelpReaderIgnoresDescriptionContinuations(t *testing.T) {
	help := "Group:\n  first <x>                           Does a thing and\n                                      sem continuation words here\n  second | third                      Others\n"
	got := strings.Join(helpAdvertisedCommands(help), ",")
	if got != "first,second,third" {
		t.Fatalf("commands read = %s", got)
	}
}
