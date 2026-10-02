package cliout

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

type profileGolden struct {
	Out string
	Err string
}

func profileScenario(p, errP Profile, machine bool) profileGolden {
	var out, errors bytes.Buffer
	r := New(&out, &errors, p, errP)
	if machine {
		r.RecordJSON("check")
	}
	message, fix := "The delivery has no requirement evidence yet.", "Record the verification evidence before closing the delivery."
	if p.Locale == LocalePtBR {
		message, fix = "A entrega ainda não tem evidência dos requisitos.", "Registre a evidência de verificação antes de concluir a entrega."
	}
	r.Finding(Finding{State: StateWarning, Code: "trace.missing", Path: "spec.md", Message: message, Remediation: fix, ID: "trace.missing"})
	r.Field("check.findings", "1")
	r.Table(Table{Header: []string{"ID", "STATE"}, Rows: [][]string{{"requirement-one", "warning"}}})
	r.Hint(Msg(MsgSuggest, p.Locale, "check"))
	r.Verdict(Verdict{State: StateFail})
	if machine {
		_ = r.FlushJSON()
	}
	return profileGolden{out.String(), errors.String()}
}

func TestLifecycleRendererProfileGoldens(t *testing.T) {
	fixtures := map[string]profileGolden{}
	for _, locale := range []string{LocaleEN, LocalePtBR} {
		for _, name := range []string{"plain", "ascii", "unicode", "colour", "narrow", "quiet", "mixed"} {
			p := Plain()
			p.Locale = locale
			switch name {
			case "ascii":
				p.Width = 60
			case "unicode":
				p.Unicode = true
			case "colour":
				p.TTY, p.Color, p.Unicode = true, true, true
			case "narrow":
				p.Width = 32
			case "quiet":
				p.Quiet = true
			}
			errP := p
			if name == "mixed" {
				errP.TTY, errP.Color, errP.Unicode = true, true, true
			}
			key := locale + "/" + name
			fixtures[key] = profileScenario(p, errP, false)
			machine := profileScenario(p, errP, true)
			var document Record
			if err := json.Unmarshal([]byte(machine.Out), &document); err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			if len(document.Findings) != 1 || document.Findings[0].Code != "trace.missing" || document.Fields["check.findings"] != "1" || document.Outcome != "fail" || strings.Contains(machine.Out, "\\u001b") {
				t.Fatalf("%s loses or decorates machine facts: %s", key, machine.Out)
			}
			if !p.Color && strings.Contains(fixtures[key].Out, "\x1b") {
				t.Fatalf("%s decorates the result stream", key)
			}
			if p.Quiet && fixtures[key].Err != "" {
				t.Fatalf("%s leaks quiet progress", key)
			}
		}
	}
	path := filepath.Join("testdata", "profile-contract.json")
	if os.Getenv("POSE_UPDATE_PROFILE_GOLDENS") == "1" {
		raw, _ := json.MarshalIndent(fixtures, "", "  ")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]profileGolden
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	if len(expected) != len(fixtures) {
		t.Fatal("profile coverage changed")
	}
	for key, actual := range fixtures {
		if actual != expected[key] {
			t.Errorf("%s profile differs\nout=%q\nerr=%q", key, actual.Out, actual.Err)
		}
	}
}

func TestColouredTablesKeepPlainAlignment(t *testing.T) {
	plain := Plain()
	colour := plain
	colour.Color = true
	want := profileScenario(plain, plain, false).Out
	got := profileScenario(colour, plain, false).Out
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	if ansi.ReplaceAllString(got, "") != want {
		t.Fatalf("colour changes alignment\n%s\n%s", got, want)
	}
}

func TestUTF8ProseWrapsByCharacters(t *testing.T) {
	p := Plain()
	p.Width = 32
	r := New(&bytes.Buffer{}, &bytes.Buffer{}, p, p)
	lines := r.wrap(strings.Repeat("ação ", 12), 4)
	if len(lines) != 3 || strings.TrimSpace(lines[0]) != "ação ação ação ação ação" {
		t.Fatalf("UTF-8 wrapping still uses bytes: %q", lines)
	}
	for _, line := range lines {
		if utf8.RuneCountInString(line) > p.Width || !utf8.ValidString(line) {
			t.Fatalf("invalid width or UTF-8: %q", line)
		}
	}
}

func TestUnknownKindsAreFullyLocalised(t *testing.T) {
	for _, kind := range []string{"command", "flag"} {
		var errors bytes.Buffer
		p := Plain()
		p.Locale = LocalePtBR
		r := New(&bytes.Buffer{}, &errors, p, p)
		r.UnknownToken(kind, "--jsonn", []string{"--json"})
		if strings.Contains(errors.String(), kind) || !strings.Contains(errors.String(), "você quis dizer") {
			t.Fatalf("untranslated token kind or missing suggestion: %s", errors.String())
		}
	}
}
