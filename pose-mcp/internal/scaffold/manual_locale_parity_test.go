package scaffold

// Manual locale parity (spec pose-manual-locale-parity).
//
// POSE.md and AGENTS.md are shipped machinery: `pose upgrade` delivers the
// manual matching the instance's own locale, so a stale translation reaches
// every pt-BR instance on every release while the English one stays green.
// That is exactly what happened — thirteen feature commits updated POSE.md and
// never reached locales/pt-BR/POSE.md, leaving translated instances describing
// a POSE without the release lifecycle, `pose state`, docs governance or the
// extension ecosystem.
//
// The same class was already closed for translated skills
// (pose-machinery-distribution-contract R3, see skills_check.go); the manuals
// were simply outside that guard.
//
// Prose cannot be compared across languages, so the contract is structural:
// the heading tree must match, and every technical token — command names,
// flags, paths, config keys — must appear on both sides. Those are identifiers,
// not prose: they are the same word in every language.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold/distpolicy"
)

var (
	headingRe = regexp.MustCompile(`(?m)^(#{1,4}) `)
	tickRe    = regexp.MustCompile("`([^`]+)`")
	// A token is structurally technical when it carries a separator a prose
	// word never has: a flag, a file extension, or an internal -, _, . or /.
	// A bare lowercase word is deliberately NOT matched here — `check` is a
	// command but `when` and `unless` are prose, and no pattern separates them.
	// Single-word commands are recovered from the manuals themselves below.
	technicalRe = regexp.MustCompile(`^(--?[a-z][\w-]*|[\w./-]+\.(json|md|jsonl|yaml|yml|go|sh)|[a-z][a-z0-9]*([-_./][a-z0-9]+)+)$`)
	// A command entry looks like "- `name` — description" in either manual.
	commandEntryRe = regexp.MustCompile("(?m)^- `([a-z][a-z0-9-]*)")
)

// translatedPlaceholders are the `<...>` slots in usage strings. Their contents
// are prose inside syntax — `<reason>` is written `<motivo>`, `<execution-id>`
// as `<execução>` — so they are excluded from the comparison on both sides.
// Excluding is deliberate rather than mapping each pair: a mapping table would
// have to be extended for every new placeholder and every new locale, and a
// missing entry there fails as a false positive that reads like real drift.
var translatedPlaceholders = map[string]bool{
	"reason": true, "motivo": true,
	"name": true, "nome": true,
	"text": true, "texto": true,
	"scope": true, "escopo": true,
	"decision": true, "decisão": true,
	"execution-id": true, "execução": true,
	"repo-relative-target": true, "alvo-relativo-ao-repo": true,
	"doc-path": true, "caminho-do-doc": true,
	"other-spec": true, "outra-spec": true,
}

// commandNames collects the single-word commands each manual documents as a
// list entry, so `check` counts as technical while `when` does not — derived
// from the manuals rather than from a hand-maintained list that would drift.
func commandNames(sources ...string) map[string]bool {
	out := map[string]bool{}
	for _, s := range sources {
		for _, m := range commandEntryRe.FindAllStringSubmatch(s, -1) {
			out[m[1]] = true
		}
	}
	return out
}

// splitFences separates fenced code blocks from the surrounding prose. Fences
// must be removed before pairing inline backticks: each ``` is three backticks,
// so a manual with a different number of fenced blocks pairs its inline spans
// at a different offset and the comparison silently comes out wrong — which is
// exactly how an early version of this check passed a manual it should have
// failed.
func splitFences(content string) (code []string, prose string) {
	var proseB strings.Builder
	inFence := false
	var block strings.Builder
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inFence {
				code = append(code, block.String())
				block.Reset()
			}
			inFence = !inFence
			continue
		}
		if inFence {
			// A trailing comment inside a usage block is prose in the manual's
			// own language ("# outcome auto-derivado de ..."), not part of the
			// command being documented.
			if i := strings.Index(line, "#"); i >= 0 {
				line = line[:i]
			}
			block.WriteString(line)
			block.WriteString("\n")
		} else {
			proseB.WriteString(line)
			proseB.WriteString("\n")
		}
	}
	if block.Len() > 0 {
		code = append(code, block.String())
	}
	return code, proseB.String()
}

// technicalTokens returns the identifier-shaped tokens in a manual's code:
// fenced blocks in full, plus the backticked spans in the prose. Identifiers
// are extracted from *within* each span rather than by requiring the whole span
// to be one: the interesting ones almost always appear inside a usage string
// like `pose docs-check [--json] [--explain <rule>]`, and treating those spans
// as prose is what would let a stale translation pass.
func technicalTokens(content string, commands map[string]bool) map[string]bool {
	out := map[string]bool{}
	fences, prose := splitFences(content)
	spans := append([]string{}, fences...)
	for _, m := range tickRe.FindAllStringSubmatch(prose, -1) {
		spans = append(spans, m[1])
	}
	for _, raw := range spans {
		span := strings.TrimSpace(raw)
		if span == "" {
			continue
		}
		for _, word := range strings.FieldsFunc(span, func(r rune) bool {
			return r == ' ' || r == '|' || r == ',' || r == ';' || r == '[' || r == ']' ||
				r == '(' || r == ')' || r == '<' || r == '>' || r == '"' || r == '\'' ||
				r == '\n' || r == '\t' || r == '=' || r == '`'
		}) {
			word = strings.Trim(word, ".:")
			if word == "" {
				continue
			}
			if translatedPlaceholders[word] {
				continue
			}
			if technicalRe.MatchString(word) || commands[word] {
				out[word] = true
			}
		}
	}
	return out
}

func headingShape(content string) []string {
	m := headingRe.FindAllStringSubmatch(content, -1)
	out := make([]string, 0, len(m))
	for _, h := range m {
		out = append(out, h[1])
	}
	return out
}

func readManual(t *testing.T, parts ...string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("reading manual: %v", err)
	}
	return distpolicy.StripDynamicContributorSection(string(raw))
}

func TestManualLocaleParity(t *testing.T) {
	root := poseDistDir(t)
	localesDir := filepath.Join(root, "locales")
	locales, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("reading locales/: %v", err)
	}
	checked := 0
	for _, loc := range locales {
		if !loc.IsDir() {
			continue
		}
		for _, doc := range []string{"POSE.md", "AGENTS.md"} {
			translated := filepath.Join(localesDir, loc.Name(), doc)
			if _, err := os.Stat(translated); err != nil {
				continue // a locale need not translate every manual
			}
			checked++
			source := readManual(t, root, doc)
			target := readManual(t, translated)

			if a, b := headingShape(source), headingShape(target); !equalShape(a, b) {
				t.Errorf("%s/%s: heading tree differs from %s (%d vs %d headings) — the translation has drifted structurally, not just in wording",
					loc.Name(), doc, doc, len(a), len(b))
			}

			commands := commandNames(source, target)
			src, tgt := technicalTokens(source, commands), technicalTokens(target, commands)
			if missing := diffTokens(src, tgt); len(missing) > 0 {
				t.Errorf("%s/%s: %d technical token(s) documented in %s and absent from the translation — the translated manual is stale and `pose upgrade` ships it to every %s instance: %s",
					loc.Name(), doc, len(missing), doc, loc.Name(), strings.Join(capTokens(missing), ", "))
			}
			if extra := diffTokens(tgt, src); len(extra) > 0 {
				t.Errorf("%s/%s: %d technical token(s) present only in the translation — either %s is missing them or the translation documents something that does not exist: %s",
					loc.Name(), doc, len(extra), doc, strings.Join(capTokens(extra), ", "))
			}
		}
	}
	if checked == 0 {
		t.Error("no translated manual was compared — the locale discovery is broken, not the manuals")
	}
}

func equalShape(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffTokens(from, to map[string]bool) []string {
	var out []string
	for tok := range from {
		if !to[tok] {
			out = append(out, tok)
		}
	}
	sort.Strings(out)
	return out
}

// capTokens keeps the failure readable when a manual has drifted badly.
func capTokens(tokens []string) []string {
	const max = 12
	if len(tokens) <= max {
		return tokens
	}
	out := append([]string{}, tokens[:max]...)
	return append(out, "… and "+itoa(len(tokens)-max)+" more")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
