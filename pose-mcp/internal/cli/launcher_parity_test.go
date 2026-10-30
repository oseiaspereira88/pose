package cli

// The personal launchers retire only at proven parity (spec
// pose-delegation-parity-gate): a versioned matrix maps every capability the
// delegation ADR enumerates to a pose delegate command and the tests that
// cover it, and this check fails while any row is uncovered (R1–R3).

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const launcherParityMatrix = "../../../.pose/parity/delegation-launchers.json"

type launcherParityRow struct {
	ID         string   `json:"id"`
	Launcher   string   `json:"launcher"`
	Capability string   `json:"capability"`
	Command    string   `json:"command"`
	Tests      []string `json:"tests"`
	Note       string   `json:"note,omitempty"`
}

type launcherParity struct {
	SchemaVersion int                 `json:"schema_version"`
	ADR           string              `json:"adr"`
	Rows          []launcherParityRow `json:"rows"`
}

// adrRetirementRule is the one paragraph of the section that is not an
// inventory, word for word: an edit to it, or a sentence added to it, is read
// as an unrecognized paragraph (review round 2, PG-R2-03).
const adrRetirementRule = "- Retire the personal launchers only when a versioned parity matrix maps every capability below to a native command and a test, and the matrix check fails while any row is uncovered."

// adrLauncherCapabilities reads the capabilities the ADR's "Retirement by
// parity" section enumerates: one paragraph per launcher, opened by its name
// in backticks and a colon, its capabilities separated by semicolons. It
// fails closed: any other paragraph of the section but the retirement rule
// is returned as unrecognized, so a capability added in another form is not
// silently left out (review round 1, PG-R1-03).
func adrLauncherCapabilities(adr string) (map[string][]string, []string) {
	start := strings.Index(adr, "### Retirement by parity")
	if start < 0 {
		return nil, []string{"the ADR has no \"Retirement by parity\" section"}
	}
	section := adr[start+len("### Retirement by parity"):]
	// The section ends at the next heading of its level or above; a deeper
	// heading stays inside it and, not being an inventory paragraph, is read
	// as unrecognized (review round 3, PG-R3-02).
	if end := regexp.MustCompile(`\n#{1,3} `).FindStringIndex(section); end != nil {
		section = section[:end[0]]
	}
	opener := regexp.MustCompile("^`([a-z-]+)`: ")
	out := map[string][]string{}
	var unrecognized []string
	for _, paragraph := range strings.Split(section, "\n\n") {
		text := strings.Join(strings.Fields(paragraph), " ")
		if text == "" || text == adrRetirementRule {
			continue
		}
		m := opener.FindStringSubmatch(text)
		if m == nil {
			unrecognized = append(unrecognized, text)
			continue
		}
		for _, item := range strings.Split(strings.TrimSuffix(text[len(m[0]):], "."), "; ") {
			out[m[1]] = append(out[m[1]], strings.TrimSpace(item))
		}
	}
	return out, unrecognized
}

// repositoryTestNames are the Go test functions of the module rooted at dir.
func repositoryTestNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	decl := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
	names := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range decl.FindAllStringSubmatch(string(raw), -1) {
			names[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// delegateCommandProblem says why command is not a pose delegate command the
// help catalogue documents with every flag it names, or "" when it is.
func delegateCommandProblem(command string) string {
	fields := strings.Fields(command)
	if len(fields) < 3 || fields[0] != "pose" || fields[1] != "delegate" {
		return "is not a pose delegate command"
	}
	for _, sub := range commandHelpCatalog["delegate"].Subcommands {
		if sub.Name != fields[2] {
			continue
		}
		for _, field := range fields[3:] {
			if strings.HasPrefix(field, "-") && !regexp.MustCompile(`(^|[\s\[])`+regexp.QuoteMeta(field)+`([\s\]=\[]|$)`).MatchString(sub.Usage) {
				return "names " + field + ", which `pose delegate " + sub.Name + "` does not document"
			}
		}
		return ""
	}
	return "names `pose delegate " + fields[2] + "`, which does not exist"
}

// launcherParityProblems is every reason the matrix does not prove parity
// with the capabilities adr enumerates (R2).
func launcherParityProblems(m launcherParity, adr map[string][]string, tests map[string]bool) []string {
	var problems []string
	carried := map[string]bool{}
	ids := map[string]bool{}
	for _, row := range m.Rows {
		carried[row.Launcher+"\x00"+row.Capability] = true
		switch {
		case row.ID == "" || ids[row.ID]:
			problems = append(problems, "row "+row.ID+": missing or repeated id")
		case row.Command == "":
			problems = append(problems, row.ID+": no native command")
		case len(row.Tests) == 0:
			problems = append(problems, row.ID+": no covering test")
		}
		ids[row.ID] = true
		if row.Command != "" {
			if why := delegateCommandProblem(row.Command); why != "" {
				problems = append(problems, row.ID+": command "+why)
			}
		}
		for _, name := range row.Tests {
			if !tests[name] {
				problems = append(problems, row.ID+": test "+name+" does not exist")
			}
		}
	}
	launchers := make([]string, 0, len(adr))
	for launcher := range adr {
		launchers = append(launchers, launcher)
	}
	sort.Strings(launchers)
	for _, launcher := range launchers {
		for _, capability := range adr[launcher] {
			if !carried[launcher+"\x00"+capability] {
				problems = append(problems, launcher+": the ADR's capability \""+capability+"\" has no row")
			}
		}
	}
	return problems
}

func loadLauncherParity(t *testing.T) (launcherParity, map[string][]string, []string) {
	t.Helper()
	raw, err := os.ReadFile(launcherParityMatrix)
	if err != nil {
		t.Fatal(err)
	}
	var m launcherParity
	if err := json.Unmarshal(raw, &m); err != nil || m.SchemaVersion != 1 {
		t.Fatalf("the parity matrix is not schema 1: %v", err)
	}
	adr, err := os.ReadFile(filepath.Join("../../..", m.ADR))
	if err != nil {
		t.Fatal(err)
	}
	inventory, unrecognized := adrLauncherCapabilities(string(adr))
	return m, inventory, unrecognized
}

// R1, R2: the matrix proves parity: every capability the ADR enumerates has
// a row, and every row a documented pose delegate command and tests that
// exist.
func TestLauncherParityMatrixCoversEveryLauncherCapability(t *testing.T) {
	m, adr, unrecognized := loadLauncherParity(t)
	if len(adr["secondary-agent"]) == 0 || len(adr["secondary-agent-watch"]) == 0 {
		t.Fatalf("the ADR's parity inventory could not be read: %v", adr)
	}
	problems := launcherParityProblems(m, adr, repositoryTestNames(t, "../.."))
	for _, text := range unrecognized {
		problems = append(problems, "the ADR's parity section has a paragraph the inventory cannot read: "+text)
	}
	if len(problems) > 0 {
		t.Fatalf("the launchers are not at parity:\n  %s", strings.Join(problems, "\n  "))
	}
}

// R2, R3: the check fails on a row with no test, a test that does not
// exist, a command or a flag the help does not document, and a capability
// the ADR enumerates that no row carries.
func TestLauncherParityCheckFailsOnEveryUncoveredRow(t *testing.T) {
	m, adr, _ := loadLauncherParity(t)
	tests := repositoryTestNames(t, "../..")
	fixtures := map[string]func(*launcherParity) string{
		"no covering test": func(p *launcherParity) string {
			p.Rows[0].Tests = nil
			return p.Rows[0].ID + ": no covering test"
		},
		"a test that does not exist": func(p *launcherParity) string {
			p.Rows[1].Tests = []string{"TestNothingByThisName"}
			return p.Rows[1].ID + ": test TestNothingByThisName does not exist"
		},
		"no native command": func(p *launcherParity) string {
			p.Rows[2].Command = ""
			return p.Rows[2].ID + ": no native command"
		},
		"a command that does not exist": func(p *launcherParity) string {
			p.Rows[3].Command = "pose delegate launch"
			return p.Rows[3].ID + ": command names `pose delegate launch`, which does not exist"
		},
		"a flag the command does not document": func(p *launcherParity) string {
			p.Rows[4].Command = "pose delegate ls --follow"
			return p.Rows[4].ID + ": command names --follow, which `pose delegate ls` does not document"
		},
		"an ADR capability with no row": func(p *launcherParity) string {
			missing := p.Rows[len(p.Rows)-1]
			p.Rows = p.Rows[:len(p.Rows)-1]
			return missing.Launcher + ": the ADR's capability \"" + missing.Capability + "\" has no row"
		},
	}
	for name, mutate := range fixtures {
		t.Run(name, func(t *testing.T) {
			fixture := m
			fixture.Rows = append([]launcherParityRow(nil), m.Rows...)
			want := mutate(&fixture)
			problems := launcherParityProblems(fixture, adr, tests)
			if !containsString(problems, want) {
				t.Fatalf("the check accepted %s; problems: %v", name, problems)
			}
		})
	}
}

// replacedNames are what an active document may no longer name: the removed
// review aliases and the personal launchers (R4).
var replacedNames = regexp.MustCompile("pose review (dispatch|resume|status)\\b|secondary-agent")

// allowedRemovalNotes are the only sentences an active document may name a
// replaced command in: each says the aliases were removed and names their
// replacement. A note is allowed by being listed here, word for word, never
// by containing a removal word (review rounds 1 and 2, PG-R1-05, PG-R2-01):
// a new mention of a replaced command must be added here on purpose.
var allowedRemovalNotes = map[string]bool{
	"`pose review dispatch`, `resume` and `status`, the aliases this surface deprecated, were removed: each exits 2 and names its `pose delegate` replacement":           true,
	"`pose review dispatch`, `resume` e `status`, os aliases que esta superfície depreciou, foram removidos: cada um sai com 2 e nomeia o substituto em `pose delegate`": true,
	"`pose review dispatch`, `resume` and `status` were removed": true,
	"pose review dispatch, resume and status were removed":       true,
	"pose review dispatch, resume e status foram removidos":      true,
}

var sentenceEnd = regexp.MustCompile(`[.;!?](\s+|$)`)

// replacedNameProblems are the sentences of text that name a replaced
// command or a launcher and are not an allowed removal note.
func replacedNameProblems(name, text string) []string {
	var problems []string
	line := 1
	for _, paragraph := range strings.Split(text, "\n\n") {
		joined := strings.Join(strings.Fields(paragraph), " ")
		for _, sentence := range sentenceEnd.Split(joined, -1) {
			sentence = strings.TrimSpace(sentence)
			if replacedNames.MatchString(sentence) && !allowedRemovalNotes[sentence] {
				problems = append(problems, name+":"+strconv.Itoa(line)+": "+sentence)
			}
		}
		line += strings.Count(paragraph, "\n") + 2
	}
	return problems
}

// activeDocumentRoots are the places a person or an agent reads to operate
// POSE; the historical records under .pose (specs, ADRs, reports, roadmaps,
// changelogs, knowledge, releases) are not among them. A required root must
// exist.
var activeDocumentRoots = []struct {
	path     string
	required bool
}{
	{"POSE.md", true}, {"AGENTS.md", true}, {"README.md", true}, {"README.pt-BR.md", false}, {"CONTRIBUTING.md", false}, {"SECURITY.md", false},
	{"locales", true}, {"docs-site/docs", true}, {"docs", false}, {".agents/skills", true}, {".pose/workflows", true}, {".pose/rules", true},
	{".pose/templates", false}, {"pose-mcp/internal/scaffold", true},
}

// activeDocumentProblems are the sentences of the active documents under
// root that name a replaced command or a launcher outside a removal note
// (R4; review round 1, PG-R1-04).
func activeDocumentProblems(t *testing.T, root string) []string {
	t.Helper()
	var problems []string
	for _, top := range activeDocumentRoots {
		base := filepath.Join(root, filepath.FromSlash(top.path))
		if _, err := os.Stat(base); err != nil {
			if top.required || !os.IsNotExist(err) {
				problems = append(problems, top.path+": "+err.Error())
			}
			continue
		}
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !(strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdx")) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			problems = append(problems, replacedNameProblems(filepath.ToSlash(rel), string(raw))...)
			return nil
		})
		if err != nil {
			problems = append(problems, top.path+": "+err.Error())
		}
	}
	return problems
}

// goStringFolder folds the string expressions of one Go package with the
// constant values the type checker gives them: literals unquoted, constants
// resolved through the declarations their names denote — shadowing, groups,
// local constants and any declaration order included — and + chains folded,
// an operand with no constant value standing as a gap (review rounds 2–4,
// PG-R2-02, PG-R3-01, PG-R3-03, PG-R4-01, PG-R4-02).
type goStringFolder struct {
	info   *types.Info
	pkg    *types.Package
	folded map[ast.Node]bool
}

func (g *goStringFolder) fold(e ast.Expr) (string, bool) {
	if tv, ok := g.info.Types[e]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			v, err := strconv.Unquote(e.Value)
			return v, err == nil
		}
	case *ast.ParenExpr:
		return g.fold(e.X)
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			left, lok := g.fold(e.X)
			right, rok := g.fold(e.Y)
			if lok || rok {
				if !lok {
					left = "\x00"
				}
				if !rok {
					right = "\x00"
				}
				return left + right, true
			}
		}
	}
	return "", false
}

// mark records the nodes a folded value already accounts for, so a literal
// is not read twice; an operand that did not fold is left to be visited.
func (g *goStringFolder) mark(e ast.Expr) {
	switch e := e.(type) {
	case *ast.BasicLit:
		g.folded[e] = true
	case *ast.ParenExpr:
		g.folded[e] = true
		g.mark(e.X)
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			g.folded[e] = true
			if _, ok := g.fold(e.X); ok {
				g.mark(e.X)
			}
			if _, ok := g.fold(e.Y); ok {
				g.mark(e.Y)
			}
		}
	}
}

// noImports lets a package be type-checked alone: what comes from another
// package has no constant value, so it stays a gap.
type noImports struct{}

func (noImports) Import(path string) (*types.Package, error) {
	return nil, fmt.Errorf("not imported: %s", path)
}

// parityBuildTargets are the configurations the distribution ships; a
// package is read once per configuration, with the files Go would build for
// it, so declarations of mutually exclusive files never meet in one check
// (review round 5, PG-R5-01).
var parityBuildTargets = []string{"linux", "darwin", "windows"}

// goPackageStringValues are the string values of the Go files of one
// package directory, by the position of the expression that holds them: for
// each shipped configuration, the files it builds are type-checked alone
// and each expression folded once, at its widest, with every literal inside
// an operand that did not fold — a call's argument, a function's body —
// read as well; comments are not strings. A position whose value differs
// between configurations keeps every value.
func goPackageStringValues(t *testing.T, dir string, names []string) map[string][]string {
	t.Helper()
	byPosition := map[string]map[string]bool{}
	for _, goos := range parityBuildTargets {
		ctx := build.Default
		ctx.GOOS, ctx.GOARCH, ctx.CgoEnabled = goos, "amd64", true
		fset := token.NewFileSet()
		var parsed []*ast.File
		for _, name := range names {
			if ok, err := ctx.MatchFile(dir, name); err != nil {
				t.Fatalf("%s: %v", filepath.Join(dir, name), err)
			} else if !ok {
				continue
			}
			f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
			if err != nil {
				t.Fatalf("%s: %v", filepath.Join(dir, name), err)
			}
			parsed = append(parsed, f)
		}
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		conf := types.Config{Importer: noImports{}, Error: func(error) {}}
		pkg, _ := conf.Check("parity", fset, parsed, info)
		g := &goStringFolder{info: info, pkg: pkg, folded: map[ast.Node]bool{}}
		for _, f := range parsed {
			ast.Inspect(f, func(n ast.Node) bool {
				if g.folded[n] {
					return true
				}
				switch n := n.(type) {
				case *ast.BinaryExpr, *ast.BasicLit:
					if v, ok := g.fold(n.(ast.Expr)); ok {
						position := fset.Position(n.Pos())
						key := position.Filename + "\x00" + strconv.Itoa(position.Offset)
						if byPosition[key] == nil {
							byPosition[key] = map[string]bool{}
						}
						byPosition[key][v] = true
						g.mark(n.(ast.Expr))
					}
				}
				return true
			})
		}
	}
	values := map[string][]string{}
	keys := make([]string, 0, len(byPosition))
	for key := range byPosition {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		path := key[:strings.IndexByte(key, 0)]
		for v := range byPosition[key] {
			values[path] = append(values[path], v)
		}
	}
	return values
}

// sourceStringProblems are the string values of the Go sources under dir,
// tests aside, that name a replaced command or a launcher outside an allowed
// removal note: usage strings, errors, MCP tool descriptions (PG-R1-04).
func sourceStringProblems(t *testing.T, root, dir string) []string {
	t.Helper()
	packages := map[string][]string{}
	var problems []string
	err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], filepath.Base(path))
		return nil
	})
	if err != nil {
		problems = append(problems, dir+": "+err.Error())
	}
	for pkg, names := range packages {
		sort.Strings(names)
		for path, values := range goPackageStringValues(t, pkg, names) {
			rel, _ := filepath.Rel(root, path)
			for _, value := range values {
				problems = append(problems, replacedNameProblems(filepath.ToSlash(rel), value)...)
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// helpProblems are the help texts, flags included, that name a replaced
// command outside a removal note.
func helpProblems() []string {
	var problems []string
	for name, h := range commandHelpCatalog {
		texts := []string{h.Usage, h.SummaryEN, h.SummaryPtBR, h.DescriptionEN, h.DescriptionPtBR}
		for _, flag := range h.Flags {
			texts = append(texts, flag.Flag, flag.DescriptionEN, flag.DescriptionPtBR)
		}
		for _, sub := range h.Subcommands {
			texts = append(texts, sub.Usage, sub.SummaryEN, sub.SummaryPtBR)
		}
		texts = append(texts, h.Examples...)
		for _, text := range texts {
			problems = append(problems, replacedNameProblems("help "+name, text)...)
		}
	}
	sort.Strings(problems)
	return problems
}

// R4: documentation, skills and help name only pose delegate; the removed
// aliases and the launchers appear only in removal notes and in historical
// records.
func TestActiveDocumentsNameOnlyPoseDelegate(t *testing.T) {
	problems := append(activeDocumentProblems(t, "../../.."), helpProblems()...)
	problems = append(problems, sourceStringProblems(t, "../../..", "pose-mcp")...)
	problems = append(problems, sourceStringProblems(t, "../../..", "mcp-enforce")...)
	if len(problems) > 0 {
		t.Fatalf("active documents name replaced commands:\n  %s", strings.Join(problems, "\n  "))
	}
}

// R4 (review round 1, PG-R1-04, PG-R1-05): the documentation check fails on
// an active document, a Go string or a missing required place that names a
// replaced command or a launcher, sentence by sentence; a removal note
// passes, a sentence that says the command stays does not, and the
// historical records are not read.
func TestActiveDocumentCheckFailsOnAReplacedCommand(t *testing.T) {
	root := t.TempDir()
	write := func(rel, text string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, top := range activeDocumentRoots {
		switch {
		case !top.required || top.path == ".pose/rules":
		case strings.HasSuffix(top.path, ".md"):
			write(top.path, "Use pose delegate review.\n")
		default:
			write(top.path+"/keep.md", "Use pose delegate review.\n")
		}
	}
	write("POSE.md", "Run `pose review dispatch spec:x --via codex`.\n`pose review dispatch`, `resume` and `status` were removed; `pose delegate` replaces them.\n\n`pose review dispatch`, `resume` and `status` remain as deprecated aliases.\n\nThe old transport was removed, so run pose review dispatch spec:x.\n\npose review dispatch was removed from the retirement list and works as before.\n")
	write("locales/pt-BR/POSE.md", "`pose review dispatch`, `resume` e `status`, os aliases que esta superfície depreciou, foram removidos: cada um sai com 2 e nomeia o substituto em `pose delegate`.\n\n`pose review dispatch`, `resume` e `status` ficam como aliases depreciados.\n\n`pose review resume` foi removido, mas continua disponível como alias.\n")
	write("docs-site/docs/guide.mdx", "Start it with secondary-agent --read-only prompt.md\n")
	write(".pose/adr/old.md", "The launchers secondary-agent and pose review resume were replaced.\n")
	write("pose-mcp/internal/cli/x.go", "package cli\n\n// \"secondary-agent\" was an old launcher.\nconst usage = \"Usage: pose review dispatch <scope>\"\nconst escaped = \"Run pose review \\x64ispatch spec:x\"\nconst joined = \"Run pose review \" + \"dispatch spec:x\"\nconst gap = \"pose review \" + verb + \" was removed\"\nconst gone = \"pose review dispatch, resume and status were removed.\"\n\nvar verb = \"x\"\n\nfunc a() string { return \"Usage: \" + fmt.Sprint(\"Run pose review dispatch spec:a\") }\n\nfunc b() string { return \"Usage: \" + func() string { return \"Run pose review dispatch spec:b\" }() }\n\nconst r3Verb = \"dispatch\"\n")
	write("pose-mcp/internal/cli/y.go", "package cli\n\nconst r3Usage = \"Run pose review \" + r3Verb + \" spec:c\"\n")
	write("pose-mcp/internal/cli/w_unix.go", "//go:build !windows\n\npackage cli\n\nconst r5Suffix = \" spec:r5\"\nconst r5Verb = \"dispatch\"\n")
	write("pose-mcp/internal/cli/w_windows.go", "package cli\n\nconst r5Suffix = \", resume and status were removed\"\nconst r5Verb = \"bundle\"\n")
	write("pose-mcp/internal/cli/w.go", "package cli\n\nconst r5Usage = \"pose review dispatch\" + r5Suffix\nconst r5Split = \"pose review \" + r5Verb + \" spec:s\"\n")
	// The same, its Windows file first in name order.
	write("pose-mcp/internal/cli/u_windows.go", "package cli\n\nconst r6Suffix = \", resume and status were removed\"\n")
	write("pose-mcp/internal/cli/v_unix.go", "//go:build !windows\n\npackage cli\n\nconst r6Suffix = \" spec:r6\"\n")
	write("pose-mcp/internal/cli/v.go", "package cli\n\nconst r6Usage = \"pose review dispatch\" + r6Suffix\n")
	write("pose-mcp/internal/cli/z.go", "package cli\n\nconst r4Suffix = \", resume and status were removed\"\n\nfunc r4Usage() string {\n\tr4Suffix := \" spec:x\"\n\treturn \"pose review dispatch\" + r4Suffix\n}\n\nconst r4Verb = \"dispatch\"\n\nfunc r4Harmless() string {\n\tr4Verb := \"bundle\"\n\treturn \"pose review \" + r4Verb\n}\n\nconst r5Usage = \"pose review \" + r5A + \" spec:d\"\nconst r5A = r5B\nconst r5B = r5C\nconst r5C = r5D\nconst r5D = r5E\nconst r5E = \"dispatch\"\n\nconst (\n\tg1 = \"dispatch\"\n\tg2\n)\n\nconst gUsage = \"pose review \" + g2 + \" spec:e\"\n\nfunc h() string {\n\tconst lv = \"dispatch\"\n\treturn \"pose review \" + lv + \" spec:f\"\n}\n")
	problems := append(activeDocumentProblems(t, root), sourceStringProblems(t, root, "pose-mcp")...)
	want := []string{
		"POSE.md:1: Run `pose review dispatch spec:x --via codex`",
		"POSE.md:4: `pose review dispatch`, `resume` and `status` remain as deprecated aliases",
		"POSE.md:6: The old transport was removed, so run pose review dispatch spec:x",
		"POSE.md:8: pose review dispatch was removed from the retirement list and works as before",
		"locales/pt-BR/POSE.md:3: `pose review dispatch`, `resume` e `status` ficam como aliases depreciados",
		"locales/pt-BR/POSE.md:5: `pose review resume` foi removido, mas continua disponível como alias",
		"docs-site/docs/guide.mdx:1: Start it with secondary-agent --read-only prompt.md",
		".pose/rules: stat " + filepath.Join(root, ".pose", "rules") + ": no such file or directory",
		"pose-mcp/internal/cli/x.go:1: Usage: pose review dispatch <scope>",
		"pose-mcp/internal/cli/x.go:1: Run pose review dispatch spec:x", // the escaped literal
		"pose-mcp/internal/cli/x.go:1: Run pose review dispatch spec:x", // the folded concatenation
		"pose-mcp/internal/cli/x.go:1: Run pose review dispatch spec:a", // a call inside a partial fold
		"pose-mcp/internal/cli/x.go:1: Run pose review dispatch spec:b", // a function inside a partial fold
		"pose-mcp/internal/cli/y.go:1: Run pose review dispatch spec:c", // a constant of another file
		"pose-mcp/internal/cli/z.go:1: pose review dispatch\x00",        // a shadowed suffix is a gap, not a removal note
		"pose-mcp/internal/cli/z.go:1: pose review dispatch spec:d",     // a chain of constants
		"pose-mcp/internal/cli/z.go:1: pose review dispatch spec:e",     // an omitted value in a const group
		"pose-mcp/internal/cli/z.go:1: pose review dispatch spec:f",     // a local constant
		"pose-mcp/internal/cli/w.go:1: pose review dispatch spec:r5",    // a suffix of the build that ships it, not of another
		"pose-mcp/internal/cli/w.go:1: pose review dispatch spec:s",     // a verb of one build
		"pose-mcp/internal/cli/v.go:1: pose review dispatch spec:r6",    // whichever build file sorts first
	}
	for _, w := range want {
		if !containsString(problems, w) {
			t.Fatalf("the check missed %q; problems:\n  %s", w, strings.Join(problems, "\n  "))
		}
	}
	if len(problems) != len(want) {
		t.Fatalf("the check reported more than the fixture plants:\n  %s", strings.Join(problems, "\n  "))
	}
}

// R2 (review round 1, PG-R1-03): the check follows the ADR's text: a
// capability added to a launcher's paragraph has no row, and one added in
// any other form is not read and fails closed.
func TestLauncherParityFollowsTheADRsText(t *testing.T) {
	m, _, _ := loadLauncherParity(t)
	raw, err := os.ReadFile(filepath.Join("../../..", m.ADR))
	if err != nil {
		t.Fatal(err)
	}
	adr := string(raw)
	tests := repositoryTestNames(t, "../..")
	appended := strings.Replace(adr, "removal of the parent's session variables.", "removal of the parent's session variables; a capability the matrix lacks.", 1)
	inventory, unrecognized := adrLauncherCapabilities(appended)
	if problems := launcherParityProblems(m, inventory, tests); len(unrecognized) != 0 || !containsString(problems, "secondary-agent: the ADR's capability \"a capability the matrix lacks\" has no row") {
		t.Fatalf("an appended capability: %v %v", problems, unrecognized)
	}
	for _, added := range []string{"- `secondary-agent`: a capability the matrix lacks.", "`secondary-agent` also keeps a capability the matrix lacks.", "* secondary-agent-watch: another one."} {
		bulleted := strings.Replace(adr, "`secondary-agent-watch`: ", added+"\n\n`secondary-agent-watch`: ", 1)
		if _, unrecognized := adrLauncherCapabilities(bulleted); len(unrecognized) != 1 {
			t.Fatalf("an inventory entry in another form was read as nothing: %q -> %v", added, unrecognized)
		}
	}
	extended := strings.Replace(adr, "while any row is uncovered.", "while any row is uncovered. The secondary-agent launcher also supports automatic credential rotation.", 1)
	if extended == adr {
		t.Fatal("the fixture did not find the retirement rule")
	}
	if _, unrecognized := adrLauncherCapabilities(extended); len(unrecognized) != 1 {
		t.Fatalf("a capability added to the retirement rule was read as nothing: %v", unrecognized)
	}
	// A capability under a deeper heading is still inside the section, and
	// is not read as nothing (PG-R3-02).
	sub := strings.Replace(adr, "\n## Maintainer's decisions", "\n#### Additional launcher capabilities\n\n`secondary-agent`: automatic credential rotation.\n\n## Maintainer's decisions", 1)
	if sub == adr {
		t.Fatal("the fixture did not find the next section")
	}
	if inventory, unrecognized := adrLauncherCapabilities(sub); len(unrecognized) != 1 || !containsString(launcherParityProblems(m, inventory, tests), "secondary-agent: the ADR's capability \"automatic credential rotation\" has no row") {
		t.Fatalf("a capability under a deeper heading: %v %v", unrecognized, launcherParityProblems(m, inventory, tests))
	}
	if _, unrecognized := adrLauncherCapabilities(adr); len(unrecognized) != 0 {
		t.Fatalf("the ADR as written: %v", unrecognized)
	}
}

// The Go reader resolves the constants of one package; a fragment imported
// from another package of the module is a gap, so a command assembled from
// it is not read (review round 5, PG-R5-02). A string in a cgo preamble is C,
// a comment to the Go parser, and is not read either (PG-R5-03). Review of
// the messages a change assembles is the defence; the corpus keeps both
// limits visible.
func init() {
	adversarialCorpus = append(adversarialCorpus,
		adversarialCase{"imported-fragment-completes-a-replaced-command", "known-gap", "pose-delegation-parity-gate", func(t *testing.T) (bool, string) {
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "pose-mcp/internal/pose/verb.go"), "package pose\n\nconst Verb = \"dispatch\"\n")
			mustWrite(t, filepath.Join(root, "pose-mcp/internal/cli/usage.go"), "package cli\n\nimport posemodel \"example/pose\"\n\nconst usage = \"pose review \" + posemodel.Verb + \" spec:x\"\n")
			problems := sourceStringProblems(t, root, "pose-mcp")
			return len(problems) == 0, strings.Join(problems, "; ")
		}},
		adversarialCase{"cgo-preamble-text-reaches-usage", "known-gap", "pose-delegation-parity-gate", func(t *testing.T) (bool, string) {
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "pose-mcp/internal/cli/c.go"), "package cli\n\n/*\nstatic const char* usage(void) { return \"pose review dispatch spec:x\"; }\n*/\nimport \"C\"\n\nfunc cUsage() string { return C.GoString(C.usage()) }\n")
			problems := sourceStringProblems(t, root, "pose-mcp")
			return len(problems) == 0, strings.Join(problems, "; ")
		}},
	)
}

// The parity check proves that each row names a documented command and tests
// that exist, not that they exercise the capability: a row citing an
// unrelated existing test passes. Semantic review of the cited tests is the
// defence; the corpus keeps the gap visible (review round 2, PG-R2-07).
func init() {
	adversarialCorpus = append(adversarialCorpus, adversarialCase{"parity-row-cites-an-unrelated-test", "known-gap", "pose-delegation-parity-gate", func(t *testing.T) (bool, string) {
		m, adr, _ := loadLauncherParity(t)
		fixture := m
		fixture.Rows = append([]launcherParityRow(nil), m.Rows...)
		fixture.Rows[0].Command = "pose delegate agents"
		fixture.Rows[0].Tests = []string{"TestActiveDocumentCheckFailsOnAReplacedCommand"}
		problems := launcherParityProblems(fixture, adr, repositoryTestNames(t, "../.."))
		return len(problems) == 0, strings.Join(problems, "; ")
	}})
}
