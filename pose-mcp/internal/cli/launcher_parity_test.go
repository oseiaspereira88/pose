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
// in backticks and a colon, its capabilities separated by semicolons.
func adrLauncherCapabilities(adr string) map[string][]string {
	start := strings.Index(adr, "### Retirement by parity")
	if start < 0 {
		return nil
	}
	section := adr[start+len("### Retirement by parity"):]
	if end := strings.Index(section, "\n#"); end >= 0 {
		section = section[:end]
	}
	opener := regexp.MustCompile("^`([a-z-]+)`: ")
	out := map[string][]string{}
	for _, paragraph := range strings.Split(section, "\n\n") {
		text := strings.Join(strings.Fields(paragraph), " ")
		m := opener.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		for _, item := range strings.Split(strings.TrimSuffix(text[len(m[0]):], "."), "; ") {
			out[m[1]] = append(out[m[1]], strings.TrimSpace(item))
		}
	}
	return out
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

func loadLauncherParity(t *testing.T) (launcherParity, map[string][]string) {
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
	return m, adrLauncherCapabilities(string(adr))
}

// R1, R2: the matrix proves parity: every capability the ADR enumerates has
// a row, and every row a documented pose delegate command and tests that
// exist.
func TestLauncherParityMatrixCoversEveryLauncherCapability(t *testing.T) {
	m, adr := loadLauncherParity(t)
	if len(adr["secondary-agent"]) == 0 || len(adr["secondary-agent-watch"]) == 0 {
		t.Fatalf("the ADR's parity inventory could not be read: %v", adr)
	}
	if problems := launcherParityProblems(m, adr, repositoryTestNames(t, "../..")); len(problems) > 0 {
		t.Fatalf("the launchers are not at parity:\n  %s", strings.Join(problems, "\n  "))
	}
}

// R2, R3: the check fails on a row with no test, a test that does not
// exist, a command or a flag the help does not document, and a capability
// the ADR enumerates that no row carries.
func TestLauncherParityCheckFailsOnEveryUncoveredRow(t *testing.T) {
	m, adr := loadLauncherParity(t)
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

// removalNote marks a line that names a replaced command only to say it is
// gone or deprecated, which an active document may keep.
var removalNote = regexp.MustCompile(`(?i)\b(removed|deprecated|removid[oa]s?|depreciad[oa]s?)\b`)

// activeDocumentProblems are the lines of the active documents under root
// that name a replaced command or a launcher outside a removal note; the
// historical records under .pose are not read (R4).
func activeDocumentProblems(t *testing.T, root string) []string {
	t.Helper()
	var problems []string
	scan := func(name, text string) {
		for i, line := range strings.Split(text, "\n") {
			if replacedNames.MatchString(line) && !removalNote.MatchString(line) {
				problems = append(problems, name+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
	}
	roots := []string{"POSE.md", "AGENTS.md", "README.md", "README.pt-BR.md", "CONTRIBUTING.md", "SECURITY.md", "locales", "docs-site/docs", ".agents/skills", "pose-mcp/internal/scaffold"}
	for _, top := range roots {
		_ = filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err == nil {
				rel, _ := filepath.Rel(root, path)
				scan(filepath.ToSlash(rel), string(raw))
			}
			return nil
		})
	}
	return problems
}

// helpProblems are the help texts that name a replaced command outside a
// removal note.
func helpProblems() []string {
	var problems []string
	for name, h := range commandHelpCatalog {
		texts := []string{h.Usage, h.SummaryEN, h.SummaryPtBR, h.DescriptionEN, h.DescriptionPtBR}
		for _, sub := range h.Subcommands {
			texts = append(texts, sub.Usage, sub.SummaryEN, sub.SummaryPtBR)
		}
		texts = append(texts, h.Examples...)
		for _, text := range texts {
			if replacedNames.MatchString(text) && !removalNote.MatchString(text) {
				problems = append(problems, "help "+name+": "+text)
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// R4: documentation, skills and help name only pose delegate; the removed
// aliases and the launchers appear only in removal notes and in historical
// records.
func TestActiveDocumentsNameOnlyPoseDelegate(t *testing.T) {
	if problems := append(activeDocumentProblems(t, "../../.."), helpProblems()...); len(problems) > 0 {
		t.Fatalf("active documents name replaced commands:\n  %s", strings.Join(problems, "\n  "))
	}
}

// R4: the documentation check fails on an active document that names a
// replaced command or a launcher, and accepts a removal note and a
// historical record.
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
	write("POSE.md", "Run `pose review dispatch spec:x --via codex`.\n`pose review status` was removed; use `pose delegate ls`.\n")
	write("docs-site/docs/guide.md", "Start it with secondary-agent --read-only prompt.md\n")
	write(".agents/skills/x/SKILL.md", "Use pose delegate review.\n")
	write(".pose/adr/old.md", "The launchers secondary-agent and pose review resume were replaced.\n")
	problems := activeDocumentProblems(t, root)
	want := []string{"POSE.md:1: Run `pose review dispatch spec:x --via codex`.", "docs-site/docs/guide.md:1: Start it with secondary-agent --read-only prompt.md"}
	if strings.Join(problems, "\n") != strings.Join(want, "\n") {
		t.Fatalf("problems:\n  %s\nwant:\n  %s", strings.Join(problems, "\n  "), strings.Join(want, "\n  "))
	}
}
