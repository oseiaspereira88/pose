package cli

// The personal launchers retire only at proven parity (spec
// pose-delegation-parity-gate): a versioned matrix maps every capability the
// delegation ADR enumerates to a pose delegate command and the tests that
// cover it, and this check fails while any row is uncovered (R1–R3).

import (
	"encoding/json"
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
	if end := strings.Index(section, "\n#"); end >= 0 {
		section = section[:end]
	}
	opener := regexp.MustCompile("^`([a-z-]+)`: ")
	out := map[string][]string{}
	var unrecognized []string
	for _, paragraph := range strings.Split(section, "\n\n") {
		text := strings.Join(strings.Fields(paragraph), " ")
		if text == "" || strings.HasPrefix(text, "- Retire the personal launchers only when ") && !strings.Contains(text, "\n- ") && strings.Count(text, " - ") == 0 {
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

// removalNote marks a sentence that names a replaced command only to say it
// is gone; stillThere marks one that says it is not (review round 1,
// PG-R1-05): "remain as deprecated aliases" is not a removal note.
var (
	removalNote = regexp.MustCompile(`(?i)\b(was|were|is|are|been) removed\b|\bremoved in\b|\b(foi|foram|é|são) removid[oa]s?\b`)
	stillThere  = regexp.MustCompile(`(?i)\b(remain|remains|still|available|keep|keeps|ficam|fica|continua|continuam|permanece|permanecem|disponíve(l|is))\b`)
	sentenceEnd = regexp.MustCompile(`[.;!?](\s+|$)`)
)

// replacedNameProblems are the sentences of text that name a replaced
// command or a launcher outside a removal note.
func replacedNameProblems(name, text string) []string {
	var problems []string
	line := 1
	for _, paragraph := range strings.Split(text, "\n\n") {
		joined := strings.Join(strings.Fields(paragraph), " ")
		for _, sentence := range sentenceEnd.Split(joined, -1) {
			if replacedNames.MatchString(sentence) && (!removalNote.MatchString(sentence) || stillThere.MatchString(sentence)) {
				problems = append(problems, name+":"+strconv.Itoa(line)+": "+strings.TrimSpace(sentence))
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

// goStringLiteral is a Go interpreted or raw string literal.
var goStringLiteral = regexp.MustCompile("\"(?:[^\"\\\\\\n]|\\\\.)*\"|`[^`]*`")

// sourceStringProblems are the string literals of the Go sources under dir,
// tests aside, that name a replaced command or a launcher outside a removal
// note: usage strings, errors, MCP tool descriptions (PG-R1-04).
func sourceStringProblems(t *testing.T, root, dir string) []string {
	t.Helper()
	var problems []string
	err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, literal := range goStringLiteral.FindAllString(string(raw), -1) {
			problems = append(problems, replacedNameProblems(filepath.ToSlash(rel), literal)...)
		}
		return nil
	})
	if err != nil {
		problems = append(problems, dir+": "+err.Error())
	}
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
	write("POSE.md", "Run `pose review dispatch spec:x --via codex`.\n`pose review status` was removed; use `pose delegate ls`.\n\n`pose review dispatch`, `resume` and `status` remain as deprecated aliases.\n")
	write("locales/pt-BR/POSE.md", "`pose review dispatch`, `resume` e `status` foram removidos.\n\n`pose review dispatch`, `resume` e `status` ficam como aliases depreciados.\n\n`pose review resume` foi removido, mas continua disponível como alias.\n")
	write("docs-site/docs/guide.mdx", "The old alias was removed in 7.5. Start it with secondary-agent --read-only prompt.md\n")
	write(".pose/adr/old.md", "The launchers secondary-agent and pose review resume were replaced.\n")
	write("pose-mcp/internal/cli/x.go", "package cli\nconst usage = \"Usage: pose review dispatch <scope>\"\nconst gone = \"pose review dispatch was removed\"\n")
	problems := append(activeDocumentProblems(t, root), sourceStringProblems(t, root, "pose-mcp")...)
	want := []string{
		"POSE.md:1: Run `pose review dispatch spec:x --via codex`",
		"POSE.md:4: `pose review dispatch`, `resume` and `status` remain as deprecated aliases",
		"locales/pt-BR/POSE.md:3: `pose review dispatch`, `resume` e `status` ficam como aliases depreciados",
		"locales/pt-BR/POSE.md:5: `pose review resume` foi removido, mas continua disponível como alias",
		"docs-site/docs/guide.mdx:1: Start it with secondary-agent --read-only prompt.md",
		".pose/rules: stat " + filepath.Join(root, ".pose", "rules") + ": no such file or directory",
		"pose-mcp/internal/cli/x.go:1: \"Usage: pose review dispatch <scope>\"",
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
	if _, unrecognized := adrLauncherCapabilities(adr); len(unrecognized) != 0 {
		t.Fatalf("the ADR as written: %v", unrecognized)
	}
}
