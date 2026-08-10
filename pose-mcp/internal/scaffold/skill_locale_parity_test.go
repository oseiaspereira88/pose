package scaffold

// Skill locale parity (spec pose-skill-command-parity).
//
// `skills-check` validates a translated skill's contract — metadata, links,
// unsafe instructions — and passed for the whole life of a defect where the
// pt-BR `pose-spec-closeout` never mentioned `pose review record`,
// `pose review-check` or `pose close`. A skill can be perfectly conformant and
// still teach a workflow the engine no longer has.
//
// The manual parity check cannot be reused here. English skills were rewritten
// into a terse Codex-native shape while the translations kept an earlier,
// example-rich one, so comparing every technical token would report that
// deliberate difference as drift and drown the real signal. What must match is
// narrower and sharper: the POSE commands and MCP tools each side tells an
// agent to run.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// `pose <command>`, including the two-word forms (`review record`,
	// `assess snapshot`, `docs-review resolve`, `extension install`).
	poseCommandRe = regexp.MustCompile(`\bpose ([a-z][a-z0-9-]{2,})(?: ([a-z][a-z0-9-]{2,}))?`)
	// MCP tools are named directly: pose_project_state, pose_closeout_state.
	poseToolRe = regexp.MustCompile(`\bpose_([a-z][a-z0-9_]{2,})`)
	// Second words that are arguments or prose, never a subcommand.
	notSubcommands = map[string]bool{
		"spec": true, "specs": true, "the": true, "and": true, "for": true,
		"with": true, "from": true, "que": true, "com": true, "para": true,
		"que a": true, "the spec": true, "roadmap": true, "milestone": true,
	}
)

// taughtCommands returns the POSE commands and MCP tools a skill tells an agent
// to run. Flags and arguments are deliberately excluded: a translation is free
// to show more or fewer options, but not a different set of commands.
func taughtCommands(content string) map[string]bool {
	// A wrapped code span puts `pose` and its subcommand on different lines, so
	// matching line by line reports a command as untaught because a paragraph was
	// reflowed. That is a false negative in the gate, and it disarms it silently:
	// the translation still teaches the command, and the check still passes.
	content = strings.Join(strings.Fields(content), " ")

	out := map[string]bool{}
	for _, m := range poseCommandRe.FindAllStringSubmatch(content, -1) {
		cmd := m[1]
		if m[2] != "" && !notSubcommands[m[2]] && !strings.HasPrefix(m[2], "--") {
			// Only a known multi-word command keeps its second word; anything
			// else is an argument and would fragment the comparison.
			if multiWordCommands[cmd] {
				cmd = cmd + " " + m[2]
			}
		}
		out[cmd] = true
	}
	for _, m := range poseToolRe.FindAllStringSubmatch(content, -1) {
		out["pose_"+m[1]] = true
	}
	return out
}

// multiWordCommands are the CLI verbs whose subcommand changes what runs.
var multiWordCommands = map[string]bool{
	"review":      true,
	"assess":      true,
	"extension":   true,
	"release":     true,
	"docs-review": true,
	"state":       true,
	"telemetry":   true,
}

func TestSkillLocaleParity(t *testing.T) {
	root := poseDistDir(t)
	skillsDir := filepath.Join(root, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("reading .agents/skills: %v", err)
	}
	localesDir := filepath.Join(root, "locales")
	locales, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("reading locales/: %v", err)
	}
	compared := 0
	for _, loc := range locales {
		if !loc.IsDir() {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			slug := e.Name()
			source := filepath.Join(skillsDir, slug, "SKILL.md")
			translated := filepath.Join(localesDir, loc.Name(), ".agents", "skills", slug, "SKILL.md")
			if _, err := os.Stat(translated); err != nil {
				continue // a locale need not translate every skill
			}
			if _, err := os.Stat(source); err != nil {
				t.Errorf("%s/%s: translated skill has no English source", loc.Name(), slug)
				continue
			}
			compared++
			src := taughtCommands(readManual(t, source))
			tgt := taughtCommands(readManual(t, translated))
			if missing := diffTokens(src, tgt); len(missing) > 0 {
				t.Errorf("%s/%s: teaches %d POSE command(s) the translation does not — an agent following the translation runs a different workflow: %s",
					loc.Name(), slug, len(missing), strings.Join(capTokens(missing), ", "))
			}
			if extra := diffTokens(tgt, src); len(extra) > 0 {
				t.Errorf("%s/%s: the translation teaches %d POSE command(s) the English skill does not — either the English skill lost them or the translation is stale: %s",
					loc.Name(), slug, len(extra), strings.Join(capTokens(extra), ", "))
			}
		}
	}
	if compared == 0 {
		t.Error("no translated skill was compared — the discovery is broken, not the skills")
	}
}

// TestSkillParityRejectsAndAllows pins the contract's reach: a dropped command
// must fail, and the format difference that motivated this check must not.
func TestSkillParityRejectsAndAllows(t *testing.T) {
	english := "Run `pose review record spec:<slug> --apply`, then `pose review-check spec:<slug>` and `pose close spec:<slug>`."

	dropped := "Rode `pose review-check spec:<slug>` e edite o frontmatter."
	if missing := diffTokens(taughtCommands(english), taughtCommands(dropped)); len(missing) == 0 {
		t.Error("a translation that drops review record and close was accepted — that is the exact defect this check exists for")
	}

	// Same commands, opposite format: terse English against example-rich pt-BR.
	verbose := "Registre a review:\n```bash\npose review record spec:<slug> --reviewer <x> --apply\n```\nExija o gate:\n```bash\npose review-check spec:<slug>\n```\nAplique:\n```bash\npose close spec:<slug>\n```"
	if missing := diffTokens(taughtCommands(english), taughtCommands(verbose)); len(missing) > 0 {
		t.Errorf("the format difference was reported as drift (%v) — this check must compare commands, not shape", missing)
	}
	if extra := diffTokens(taughtCommands(verbose), taughtCommands(english)); len(extra) > 0 {
		t.Errorf("extra flags in the verbose form were reported as drift (%v) — flags are deliberately out of scope", extra)
	}
}

// TestSkillIndexParity covers .agents/skills/README.md, the routing table that
// lists every skill. TestSkillLocaleParity compares SKILL.md files and walked
// straight past it: the pt-BR index was missing pose-surface-closeout and
// pose-release-closeout entirely, so an agent reading the translated index
// could not discover two skills that exist. A gate that checks the entries and
// not the index is checking the wrong half.
func TestSkillIndexParity(t *testing.T) {
	root := poseDistDir(t)
	entryRe := regexp.MustCompile(`(?m)^\|\s*\[(pose-[a-z0-9-]+)\]`)
	source := filepath.Join(root, ".agents", "skills", "README.md")
	localesDir := filepath.Join(root, "locales")
	locales, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("reading locales/: %v", err)
	}
	listed := func(path string) map[string]bool {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		out := map[string]bool{}
		for _, m := range entryRe.FindAllStringSubmatch(string(raw), -1) {
			out[m[1]] = true
		}
		return out
	}
	src := listed(source)
	if len(src) == 0 {
		t.Fatal("no skills listed in .agents/skills/README.md — the extraction is broken, not the index")
	}
	compared := 0
	for _, loc := range locales {
		translated := filepath.Join(localesDir, loc.Name(), ".agents", "skills", "README.md")
		if _, err := os.Stat(translated); err != nil {
			continue
		}
		compared++
		tgt := listed(translated)
		if missing := diffTokens(src, tgt); len(missing) > 0 {
			t.Errorf("%s skill index omits %d skill(s) the English index lists — an agent reading it cannot discover them: %s",
				loc.Name(), len(missing), strings.Join(capTokens(missing), ", "))
		}
		if extra := diffTokens(tgt, src); len(extra) > 0 {
			t.Errorf("%s skill index lists %d skill(s) absent from the English index: %s",
				loc.Name(), len(extra), strings.Join(capTokens(extra), ", "))
		}
	}
	if compared == 0 {
		t.Error("no translated skill index was compared — the discovery is broken, not the indexes")
	}
}
