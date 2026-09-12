package cliout

// The renderer (spec pose-cli-output-rendering-system R1, R2, R4, R9).
//
// Channel discipline is enforced here rather than trusted to each call site:
// the command's result — its verdict, findings and data — goes to Out, and its
// progress, usage and own failures go to Err. Before this, 27 severity-prefixed
// lines went to stdout and 43 to stderr, findings included, so `pose x
// 2>/dev/null` dropped real findings in some commands and not in others.

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

// Renderer owns every byte a command prints. It holds one profile per stream,
// because stdout and stderr are redirected independently.
type Renderer struct {
	out, err       io.Writer
	outP, errP     Profile
	locale         string
	activeStepSets []*StepSet
}

// New builds a renderer for two streams and their resolved profiles.
func New(out, err io.Writer, outProfile, errProfile Profile) *Renderer {
	locale := outProfile.Locale
	if locale == "" {
		locale = LocaleEN
	}
	return &Renderer{out: out, err: err, outP: outProfile, errP: errProfile, locale: locale}
}

// NewPlain is the renderer a test, a pipe or a file gets: every fact, no
// decoration.
func NewPlain(out, err io.Writer) *Renderer {
	return New(out, err, Plain(), Plain())
}

// Resolve builds a renderer from the environment, one profile per stream.
func Resolve(out, err io.Writer, mode ColorMode, env Env) *Renderer {
	return New(out, err, ResolveProfile(out, mode, env), ResolveProfile(err, mode, env))
}

func (r *Renderer) Locale() string      { return r.locale }
func (r *Renderer) OutProfile() Profile { return r.outP }
func (r *Renderer) ErrProfile() Profile { return r.errP }

// SetQuiet suppresses everything but the verdict, on both streams.
func (r *Renderer) SetQuiet(quiet bool) {
	r.outP.Quiet, r.errP.Quiet = quiet, quiet
}

// SetVerbose restores a child process's streamed output.
func (r *Renderer) SetVerbose(verbose bool) {
	r.outP.Verbose, r.errP.Verbose = verbose, verbose
}

func (r *Renderer) Quiet() bool   { return r.outP.Quiet }
func (r *Renderer) Verbose() bool { return r.outP.Verbose }

// Out and Err expose the raw streams for output this layer does not own yet —
// a JSON document, or a child process's stream under --verbose. Every use is
// counted by the guard test.
func (r *Renderer) Out() io.Writer { return r.out }
func (r *Renderer) Err() io.Writer { return r.err }

// Verdict is a command's one-line decision. It is a contract line: machines
// read it, so its shape is pinned by golden tests.
type Verdict struct {
	State State
	// Word overrides the default SUCCESS/FAILURE vocabulary for the gates
	// that have their own (COMPATIBLE, VERIFIED).
	Word string
	// Text is the already-composed detail after the em dash. Empty prints the
	// word alone.
	Text string
}

// Verdict writes the decision to Out, in every profile including --quiet.
func (r *Renderer) Verdict(v Verdict) {
	word := v.Word
	if word == "" {
		word = Msg(MsgResultOK, r.locale)
		if v.State == StateFail || v.State == StateError {
			word = Msg(MsgResultFail, r.locale)
		}
	}
	line := Msg(MsgResultLabel, r.locale) + ": " + paint(r.outP, v.State.color()+sgrBold, word)
	if v.Text != "" {
		line += " — " + v.Text
	}
	fmt.Fprintln(r.out, line)
}

// Finding is one governed diagnostic. The fields are the ones the delivery
// integrity graph already carries, so the human channel stops flattening them
// by hand and the machine channel stays identical.
type Finding struct {
	State       State
	Code        string
	Path        string
	Message     string
	Remediation string
	// ID is the catalog id when the message came from the catalog; it travels
	// to the JSON channel so a machine keys on it instead of on prose.
	ID string
}

// Finding writes a diagnostic to Out — it is part of the command's result, not
// of its progress. The remediation is shown rather than dropped.
func (r *Renderer) Finding(f Finding) {
	if r.outP.Quiet {
		return
	}
	head := f.State.Symbol(r.outP) + " " + paint(r.outP, f.State.color(), f.State.Word(r.locale))
	if f.Code != "" {
		head += " " + f.Code
	}
	if f.Path != "" {
		head += " " + paint(r.outP, sgrBold, f.Path)
	}
	fmt.Fprintln(r.out, head)
	for _, line := range r.wrap(f.Message, 4) {
		fmt.Fprintln(r.out, line)
	}
	if f.Remediation != "" {
		for i, line := range r.wrap(Msg(MsgFix, r.locale)+": "+f.Remediation, 4) {
			if i == 0 {
				line = strings.Replace(line, Msg(MsgFix, r.locale), paint(r.outP, sgrDim, Msg(MsgFix, r.locale)), 1)
			}
			fmt.Fprintln(r.out, line)
		}
	}
}

// Field is a machine-oriented `name.field=value` diagnostic. It is a contract
// line: never wrapped, never decorated.
func (r *Renderer) Field(name, value string) {
	fmt.Fprintf(r.out, "%s=%s\n", name, value)
}

// Section titles a block of result output.
func (r *Renderer) Section(title string) {
	if r.outP.Quiet {
		return
	}
	fmt.Fprintln(r.out, paint(r.outP, sgrBold, title))
}

// Hint is advice, never a finding: it goes to Err so a piped result stays
// machine-clean.
func (r *Renderer) Hint(text string) {
	if r.errP.Quiet {
		return
	}
	fmt.Fprintf(r.err, "%s %s\n", StateHint.Symbol(r.errP), text)
}

// Failure reports that the command itself could not run. It goes to Err, with
// the exit code left to the caller (2 for usage, 1 for a gate).
func (r *Renderer) Failure(text string) {
	fmt.Fprintf(r.err, "%s: %s\n", Msg(MsgError, r.locale), text)
}

// Usage prints the synopsis to Err.
func (r *Renderer) Usage(text string) {
	fmt.Fprintln(r.err, text)
}

// UnknownToken reports an unrecognised command or flag in one shape across
// commands, naming the offending token alone and suggesting the nearest valid
// name when one is close enough to be a typo.
func (r *Renderer) UnknownToken(kind, token string, candidates []string) {
	r.Failure(Msg(MsgUnknown, r.locale, kind, token))
	if best, ok := nearest(token, candidates); ok {
		r.Hint(Msg(MsgSuggest, r.locale, best))
	}
}

// Table renders aligned columns through text/tabwriter — stdlib, so no
// dependency, and every list command stops hand-spacing its own columns.
type Table struct {
	Header []string
	Rows   [][]string
}

func (r *Renderer) Table(t Table) {
	if r.outP.Quiet {
		return
	}
	w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
	if len(t.Header) > 0 {
		fmt.Fprintln(w, paint(r.outP, sgrBold, strings.Join(t.Header, "\t")))
	}
	for _, row := range t.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}

// wrap breaks prose at the profile's budget and indents it. Contract lines
// never pass through here: a wrapped path is a broken path.
func (r *Renderer) wrap(text string, indent int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	pad := strings.Repeat(" ", indent)
	budget := r.outP.ProseWidth() - indent
	if budget < minWidth {
		budget = minWidth
	}
	words := strings.Fields(text)
	lines := []string{}
	current := pad
	for _, word := range words {
		candidate := current
		if current != pad {
			candidate += " "
		}
		candidate += word
		if len(candidate) > budget+indent && current != pad {
			lines = append(lines, current)
			current = pad + word
			continue
		}
		current = candidate
	}
	if current != pad {
		lines = append(lines, current)
	}
	return lines
}

// nearest returns the closest candidate within an edit distance a typo can
// explain, so a suggestion is never a guess at an unrelated command.
func nearest(token string, candidates []string) (string, bool) {
	token = strings.TrimLeft(strings.ToLower(strings.TrimSpace(token)), "-")
	if token == "" || len(candidates) == 0 {
		return "", false
	}
	sorted := append([]string{}, candidates...)
	sort.Strings(sorted)
	best, bestDistance := "", 1<<30
	for _, candidate := range sorted {
		distance := editDistance(token, strings.TrimLeft(strings.ToLower(candidate), "-"))
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	limit := 2
	if len(token) <= 4 {
		limit = 1
	}
	if bestDistance > limit {
		return "", false
	}
	return best, true
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(prev[j]+1, min(current[j-1]+1, prev[j-1]+cost))
		}
		prev, current = current, prev
	}
	return prev[len(b)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
