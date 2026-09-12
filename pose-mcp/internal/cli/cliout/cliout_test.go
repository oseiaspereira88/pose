package cliout

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Spec pose-cli-output-rendering-system.

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func env(pairs map[string]string) Env {
	return func(key string) string { return pairs[key] }
}

// A stream that is not a terminal gets no decoration, whatever the other stream
// is: stdout and stderr are redirected independently, so one profile per
// process could write control sequences into a redirected log (review of #110).
func TestProfileIsResolvedPerStream(t *testing.T) {
	buffer := &bytes.Buffer{}
	p := ResolveProfile(buffer, ColorAuto, env(nil))
	if p.TTY || p.Color {
		t.Fatalf("a buffer is not a terminal: %+v", p)
	}
	// --color=always is the operator's explicit choice and overrides detection;
	// it is the only way a non-terminal gets colour.
	if p := ResolveProfile(buffer, ColorAlways, env(nil)); !p.Color {
		t.Fatal("--color=always must colour a redirected stream when asked")
	}
	if p := ResolveProfile(os.Stdout, ColorNever, env(nil)); p.Color {
		t.Fatal("--color=never must win over a terminal")
	}
}

func TestColourPrecedenceAndCapabilities(t *testing.T) {
	cases := map[string]struct {
		mode    ColorMode
		vars    map[string]string
		color   bool
		unicode bool
		width   int
		locale  string
	}{
		"NO_COLOR disables":       {ColorAuto, map[string]string{"NO_COLOR": "1", "LANG": "en_US.UTF-8"}, false, true, 80, LocaleEN},
		"POSE_COLOR always":       {ColorAuto, map[string]string{"POSE_COLOR": "always"}, true, true, 80, LocaleEN},
		"POSE_COLOR never":        {ColorAuto, map[string]string{"POSE_COLOR": "never"}, false, true, 80, LocaleEN},
		"flag beats POSE_COLOR":   {ColorNever, map[string]string{"POSE_COLOR": "always"}, false, true, 80, LocaleEN},
		"TERM=dumb disables":      {ColorAuto, map[string]string{"TERM": "dumb"}, false, true, 80, LocaleEN},
		"non-UTF-8 locale":        {ColorAuto, map[string]string{"LC_ALL": "C"}, false, false, 80, LocaleEN},
		"COLUMNS is honoured":     {ColorAuto, map[string]string{"COLUMNS": "120"}, false, true, 120, LocaleEN},
		"absurd COLUMNS ignored":  {ColorAuto, map[string]string{"COLUMNS": "4"}, false, true, 80, LocaleEN},
		"POSE_LOCALE selects the": {ColorAuto, map[string]string{"POSE_LOCALE": LocalePtBR}, false, true, 80, LocalePtBR},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := ResolveProfile(&bytes.Buffer{}, c.mode, env(c.vars))
			if p.Color != c.color || p.Unicode != c.unicode || p.Width != c.width || p.Locale != c.locale {
				t.Fatalf("got %+v, want color=%v unicode=%v width=%d locale=%s", p, c.color, c.unicode, c.width, c.locale)
			}
		})
	}
	if got := (Profile{Width: 200}).ProseWidth(); got != 100 {
		t.Fatalf("prose stays readable: got %d", got)
	}
}

// The verdict is a contract line: machines read it, so both locales are pinned
// byte for byte.
func TestVerdictIsAContractLine(t *testing.T) {
	for locale, want := range map[string]string{
		LocaleEN:   "Result: SUCCESS — valid POSE structure (strict mode).\n",
		LocalePtBR: "Resultado: SUCESSO — estrutura POSE válida (modo strict).\n",
	} {
		out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
		p := Plain()
		p.Locale = locale
		r := New(out, errOut, p, p)
		text := "valid POSE structure (strict mode)."
		if locale == LocalePtBR {
			text = "estrutura POSE válida (modo strict)."
		}
		r.Verdict(Verdict{State: StatePass, Text: text})
		if out.String() != want {
			t.Fatalf("locale %s: got %q want %q", locale, out.String(), want)
		}
		if errOut.Len() != 0 {
			t.Fatalf("the verdict is the result: it belongs on stdout, got %q on stderr", errOut.String())
		}
	}
	out := &bytes.Buffer{}
	r := NewPlain(out, &bytes.Buffer{})
	r.Verdict(Verdict{State: StateFail, Word: "INCOMPATIBLE", Text: "do not release this candidate."})
	if out.String() != "Result: INCOMPATIBLE — do not release this candidate.\n" {
		t.Fatalf("a gate with its own vocabulary keeps it: %q", out.String())
	}
}

// A finding is the command's result and carries its remediation; before this the
// human channel flattened both by hand and dropped the remediation in places.
func TestFindingCarriesItsRemediationAndStaysOnStdout(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewPlain(out, errOut)
	r.Finding(Finding{
		State: StateError, Code: "action-mismatch", Path: ".pose/changelogs/unreleased/x.md",
		Message:     "declared artifact action is absent from the attributed Git change sets",
		Remediation: "correct the action or record the exact attributed revisions",
	})
	got := out.String()
	for _, want := range []string{"[err] error action-mismatch .pose/changelogs/unreleased/x.md", "    declared artifact action", "    fix: correct the action"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if errOut.Len() != 0 {
		t.Fatalf("a finding is a result, not progress: %q", errOut.String())
	}
	if ansi.MatchString(got) {
		t.Fatalf("the plain profile must carry no escape sequence: %q", got)
	}
}

// Quiet keeps the verdict and drops everything else; the exit code and the
// decision are what a script needs.
func TestQuietKeepsOnlyTheVerdict(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewPlain(out, errOut)
	r.SetQuiet(true)
	r.Finding(Finding{State: StateWarning, Message: "a warning"})
	r.Section("a section")
	r.Hint("a hint")
	r.Table(Table{Header: []string{"a"}, Rows: [][]string{{"b"}}})
	r.Verdict(Verdict{State: StatePass})
	if out.String() != "Result: SUCCESS\n" || errOut.Len() != 0 {
		t.Fatalf("quiet: out=%q err=%q", out.String(), errOut.String())
	}
}

// Colour is applied only where it was asked for, and never carries meaning
// alone: the state's word is printed in every profile.
func TestColourNeverCarriesMeaningAlone(t *testing.T) {
	out := &bytes.Buffer{}
	coloured := Plain()
	coloured.Color, coloured.Unicode = true, true
	r := New(out, &bytes.Buffer{}, coloured, coloured)
	r.Finding(Finding{State: StateFail, Code: "existence", Message: "missing"})
	got := out.String()
	if !ansi.MatchString(got) {
		t.Fatal("a colour profile should decorate")
	}
	if !strings.Contains(ansi.ReplaceAllString(got, ""), "✖ fail existence") {
		t.Fatalf("the word must survive stripping the colour: %q", got)
	}
}

func TestHintAndFailureGoToStderr(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewPlain(out, errOut)
	r.Hint("try pose doctor")
	r.Failure("invalid argument: --jsonn")
	r.Usage("Usage: pose check [--strict|--tolerant]")
	if out.Len() != 0 {
		t.Fatalf("stdout must stay clean for the result: %q", out.String())
	}
	for _, want := range []string{"[hint] try pose doctor", "Error: invalid argument: --jsonn", "Usage: pose check"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("missing %q in %q", want, errOut.String())
		}
	}
}

func TestUnknownTokenSuggestsOnlyANeighbour(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewPlain(out, errOut)
	r.UnknownToken("flag", "--jsonn", []string{"--json", "--strict", "--tolerant"})
	if !strings.Contains(errOut.String(), `unknown flag: "--jsonn"`) || !strings.Contains(errOut.String(), `did you mean "--json"?`) {
		t.Fatalf("want the token and its neighbour: %q", errOut.String())
	}
	errOut.Reset()
	r.UnknownToken("command", "frobnicate", []string{"check", "validate"})
	if strings.Contains(errOut.String(), "did you mean") {
		t.Fatalf("a distant token must not produce a guess: %q", errOut.String())
	}
}

func TestFieldAndTableShapes(t *testing.T) {
	out := &bytes.Buffer{}
	r := NewPlain(out, &bytes.Buffer{})
	r.Field("artifact.claims", "14")
	r.Table(Table{Header: []string{"SLUG", "STATUS"}, Rows: [][]string{{"a", "done"}, {"longer-slug", "draft"}}})
	got := out.String()
	if !strings.HasPrefix(got, "artifact.claims=14\n") {
		t.Fatalf("the field line is a contract line: %q", got)
	}
	if !strings.Contains(got, "SLUG         STATUS") {
		t.Fatalf("columns must align: %q", got)
	}
}

// Outside a terminal a step is one line per event, greppable, with no escape
// sequence — the profile agents and CI see.
func TestStepsArePlainLinesWithoutATerminal(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := NewPlain(out, errOut)
	set := r.Steps(2)
	clock := time.Unix(0, 0)
	set.now = func() time.Time { return clock }
	step := set.Start("test", "go test ./...")
	clock = clock.Add(1500 * time.Millisecond)
	step.Resolve(StatePass, "")
	second := set.Start("typecheck", "go vet ./...")
	clock = clock.Add(2 * time.Second)
	second.Resolve(StateFail, "exit=1")
	second.Detail("internal/x.go:42: unreachable", ".pose/results/validate.json")
	set.Summary()
	got := errOut.String()
	for _, want := range []string{
		"  -> test go test ./...\n",
		"  <- test pass 1.5s\n",
		"  <- typecheck fail 2.0s exit=1\n",
		"      internal/x.go:42: unreachable\n",
		"      full output in .pose/results/validate.json\n",
		"2 step(s) · 1 pass · 1 fail · 3.5s\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if ansi.MatchString(got) {
		t.Fatalf("no terminal, no escapes: %q", got)
	}
	if out.Len() != 0 {
		t.Fatalf("progress is not the result: %q", out.String())
	}
}

// On a terminal the active step repaints one line and the resolved line replaces
// it; the facts are the same ones the plain profile prints.
func TestStepsRepaintOnlyOnATerminal(t *testing.T) {
	errOut := &bytes.Buffer{}
	tty := Plain()
	tty.TTY, tty.Color, tty.Unicode = true, true, true
	r := New(&bytes.Buffer{}, errOut, Plain(), tty)
	set := r.Steps(1)
	clock := time.Unix(0, 0)
	set.now = func() time.Time { return clock }
	step := set.Start("test", "go test ./...")
	clock = clock.Add(3 * time.Second)
	step.Resolve(StatePass, "")
	got := errOut.String()
	if !strings.Contains(got, "\r\x1b[K") {
		t.Fatalf("a terminal step must repaint in place: %q", got)
	}
	stripped := ansi.ReplaceAllString(strings.ReplaceAll(got, "\r", ""), "")
	if !strings.Contains(stripped, "✔ test") || !strings.Contains(stripped, "3.0s") {
		t.Fatalf("the resolved line must carry the state and the duration: %q", stripped)
	}
}

func TestQuietSuppressesProgressEntirely(t *testing.T) {
	errOut := &bytes.Buffer{}
	r := NewPlain(&bytes.Buffer{}, errOut)
	r.SetQuiet(true)
	set := r.Steps(1)
	set.now = func() time.Time { return time.Unix(0, 0) }
	set.Start("test", "go test ./...").Resolve(StatePass, "")
	set.Summary()
	if errOut.Len() != 0 {
		t.Fatalf("quiet must print no progress: %q", errOut.String())
	}
}

// Every message exists in both languages, with the same format verbs, or a
// pt-BR reader gets a mixed interface — which is what 28% localisation already
// produced.
func TestCatalogParity(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+ #0-9.]*[a-zA-Z]`)
	for _, id := range CatalogIDs() {
		en, ptBR, ok := CatalogEntry(id)
		if !ok || strings.TrimSpace(en) == "" || strings.TrimSpace(ptBR) == "" {
			t.Errorf("%s: both languages are required, got en=%q pt-BR=%q", id, en, ptBR)
			continue
		}
		if a, b := verbs.FindAllString(en, -1), verbs.FindAllString(ptBR, -1); len(a) != len(b) {
			t.Errorf("%s: format verbs differ: en=%v pt-BR=%v", id, a, b)
		}
	}
	if got := Msg("no.such.id", LocaleEN); got != "no.such.id" {
		t.Fatalf("an unknown id must name itself rather than print nothing: %q", got)
	}
}

func TestStateVocabularyIsClosedAndAsciiCapable(t *testing.T) {
	ascii := Plain()
	for _, state := range []State{StatePass, StateFail, StateError, StateWarning, StateSkipped, StateInfo, StateHint} {
		if state.Key() == "" || state.Word(LocaleEN) == "" || state.Word(LocalePtBR) == "" {
			t.Errorf("state %d is not fully described", state)
		}
		symbol := state.Symbol(ascii)
		for _, r := range symbol {
			if r > 127 {
				t.Errorf("state %s: the ASCII profile must stay ASCII, got %q", state.Key(), symbol)
				break
			}
		}
		if StateFromKey(state.Key()) != state {
			t.Errorf("state %s does not round-trip through its key", state.Key())
		}
	}
	if StateFromKey("something-new") != StateInfo {
		t.Fatal("an unknown severity renders as info rather than inventing a look")
	}
}
