package cli

// pose delegate watch (spec pose-delegated-review-completion-signal R9–R12).
// An attempt is shown as one task: the runs it recorded are replayed first,
// with a marker where a run resumed another, and a running attempt is then
// followed through the live events its process writes, until the record is
// published. Colours follow the terminal profile: none when the output is not
// a terminal or NO_COLOR is set.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const (
	delegateWatchSchema = "pose.delegate.watch/v1"
	delegateWatchStep   = 300 * time.Millisecond
	delegateWatchLines  = 12
)

// Seams the tests replace: whether a person is at a terminal, what they type,
// and the picker command.
var (
	delegateInteractive           = func() bool { return cliout.IsTerminal(os.Stdout) && cliout.IsTerminal(os.Stdin) }
	delegateStdin       io.Reader = os.Stdin
	delegatePicker                = "fzf"
)

func cmdDelegateWatch(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "-v", "--lines", "--no-follow", "--refresh")
	if problem == "" && len(f.positional) > 1 {
		problem = tr("one run reference at most", "no máximo uma referência de run")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "watch", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate watch [<run>|<label>] [-v] [--lines <n>] [--no-follow] [--refresh <d>] [--json]")
	}
	if f.lines == 0 {
		f.lines = delegateWatchLines
	}
	if f.refresh == 0 && !containsFlag(args, "--refresh") {
		f.refresh = 2 * time.Second
	}
	store := posemodel.Store{Root: root}
	attempts, lost, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "watch", 1, err)
	}
	var attempt posemodel.DelegationAttempt
	if len(f.positional) == 1 || f.label != "" {
		a, code, err := namedAttempt(attempts, f)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "watch", code, err)
		}
		attempt = a
	} else {
		a, code, err := chooseWatchedAttempt(root, attempts, f, stderr)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "watch", code, err)
		}
		attempt = a
	}
	w := &watchPrinter{out: stdout, profile: render(stdout, stderr).OutProfile(), f: f, losses: map[string]bool{}}
	w.lose(lost)
	code := w.follow(store, attempt)
	// What could not be read is said, and the watch does not end as if the
	// history it showed were whole.
	if len(w.losses) > 0 {
		out := render(stdout, stderr)
		for loss := range w.losses {
			out.Failure(tr("delegated run history incomplete: ", "histórico de runs delegados incompleto: ") + loss)
		}
		return 1
	}
	return code
}

func (w *watchPrinter) lose(lost []string) {
	for _, loss := range lost {
		w.losses[loss] = true
	}
}

func containsFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

// chooseWatchedAttempt follows the only active run, or lets a person pick one
// at a terminal; elsewhere an absent reference exits 2 with the candidates
// (R11).
func chooseWatchedAttempt(root string, attempts []posemodel.DelegationAttempt, f delegateFlags, stderr io.Writer) (posemodel.DelegationAttempt, int, error) {
	var active, candidates []posemodel.DelegationAttempt
	for _, a := range attempts {
		if f.mismatch(a) != "" {
			continue
		}
		candidates = append(candidates, a)
		if a.Active {
			active = append(active, a)
		}
	}
	if len(active) == 1 {
		return active[0], 0, nil
	}
	if len(active) > 1 {
		candidates = active
	}
	if len(candidates) == 0 {
		return posemodel.DelegationAttempt{}, 2, errors.New(tr("no delegated run to watch", "nenhum run delegado para acompanhar"))
	}
	if !delegateInteractive() {
		ids := make([]string, 0, len(candidates))
		for _, a := range candidates {
			ids = append(ids, a.AttemptID)
		}
		return posemodel.DelegationAttempt{}, 2, &posemodel.ErrDelegationRefAmbiguous{Candidates: ids}
	}
	chosen, err := pickAttempt(root, candidates, f, stderr)
	if err != nil {
		return posemodel.DelegationAttempt{}, 2, err
	}
	for _, a := range candidates {
		if a.AttemptID == chosen {
			return a, 0, nil
		}
	}
	return posemodel.DelegationAttempt{}, 2, errors.New(tr("nothing was picked", "nada foi escolhido"))
}

// pickAttempt opens fzf with a live preview of the run under the cursor, or a
// numbered menu without it.
func pickAttempt(root string, candidates []posemodel.DelegationAttempt, f delegateFlags, stderr io.Writer) (string, error) {
	lines := make([]string, 0, len(candidates))
	for _, a := range candidates {
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s", a.AttemptID, a.State, a.Agent, a.Label, delegateAge(a.StartedAt)))
	}
	if path, err := exec.LookPath(delegatePicker); err == nil {
		self, err := os.Executable()
		if err != nil {
			return "", err
		}
		preview := shellQuote(self) + " delegate watch {1} --no-follow --lines " + strconv.Itoa(f.lines)
		if f.verbose {
			preview += " -v"
		}
		if f.refresh > 0 {
			// fzf replaces the preview when the cursor moves; the loop keeps
			// the one in view current.
			preview = "while :; do clear; " + preview + "; sleep " + strconv.FormatFloat(f.refresh.Seconds(), 'f', -1, 64) + "; done"
		}
		cmd := exec.Command(path, "--delimiter", "\t", "--with-nth", "1..", "--preview", preview, "--preview-window", "right,60%")
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
		cmd.Stderr = stderr
		out, err := cmd.Output()
		if err != nil {
			return "", errors.New(tr("nothing was picked", "nada foi escolhido"))
		}
		id, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
		return id, nil
	}
	menu := render(io.Discard, stderr)
	for i, line := range lines {
		menu.Usage(fmt.Sprintf("%d) %s", i+1, strings.ReplaceAll(line, "\t", "  ")))
	}
	menu.Usage(tr("run to watch (number): ", "run a acompanhar (número): "))
	answer, _ := bufio.NewReader(delegateStdin).ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(answer))
	if err != nil || n < 1 || n > len(candidates) {
		return "", errors.New(tr("nothing was picked", "nada foi escolhido"))
	}
	return candidates[n-1].AttemptID, nil
}

// shellQuote quotes a path for the shell fzf runs its preview with.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

type watchPrinter struct {
	out     io.Writer
	profile cliout.Profile
	f       delegateFlags
	// liveSeq is the last live event shown; the record of the run followed
	// live then adds only what the live file did not carry.
	liveSeq int
	losses  map[string]bool
}

// follow replays the attempt's recorded runs, then its live events until the
// record is published (R9, R10).
func (w *watchPrinter) follow(store posemodel.Store, attempt posemodel.DelegationAttempt) int {
	shown := map[string]bool{}
	w.replay(store, attempt, shown)
	if attempt.State == posemodel.DelegationStalled {
		w.marker("■", tr("stalled: its process is gone and left no record (since ", "travado: o processo não existe mais e não deixou registro (desde ")+attempt.StalledAt+")", cliout.ToneBad, map[string]any{"attempt": attempt.AttemptID, "status": posemodel.DelegationStalled})
		return 0
	}
	if !attempt.Active || w.f.noFollow {
		if attempt.Active {
			w.live(store, attempt, 0)
		}
		return 0
	}
	w.marker("▶", tr("live: ", "ao vivo: ")+attempt.Agent+" ("+attempt.Purpose+")", cliout.ToneInfo, map[string]any{"attempt": attempt.AttemptID})
	var offset int64
	for {
		offset = w.live(store, attempt, offset)
		attempts, lost, err := store.ListDelegationAttempts()
		if err != nil {
			return 1
		}
		w.lose(lost)
		current, found := attempt, false
		for _, a := range attempts {
			if a.AttemptID == attempt.AttemptID {
				current, found = a, true
			}
		}
		if !found || !current.Active {
			// The record is published: what the live file did not carry —
			// the final message, the end state — comes from it.
			if found {
				w.replay(store, current, shown)
			}
			return 0
		}
		time.Sleep(delegateWatchStep)
	}
}

// replay prints the recorded runs of the attempt not printed yet.
func (w *watchPrinter) replay(store posemodel.Store, attempt posemodel.DelegationAttempt, shown map[string]bool) {
	runs, lost, err := store.ReadReviewRuns("")
	if err != nil {
		w.lose([]string{err.Error()})
		return
	}
	w.lose(lost)
	byID := map[string]posemodel.ReviewRun{}
	continuedBy := map[string]string{}
	for _, r := range runs {
		byID[r.RunID] = r
		if r.Resumes != "" {
			continuedBy[r.Resumes] = r.RunID
		} else if r.HandsOffFrom != "" {
			continuedBy[r.HandsOffFrom] = r.RunID
		}
	}
	for _, id := range attempt.Runs {
		run, ok := byID[id]
		if !ok || shown[id] {
			continue
		}
		shown[id] = true
		followed := w.liveSeq > 0
		if followed {
			// The run just followed live: its header and the events already
			// shown are not repeated.
		} else if run.Resumes != "" {
			w.marker("↻", tr("resumed ", "retomado ")+run.Resumes+" → "+run.RunID+" ("+run.Adapter+")", cliout.ToneInfo, map[string]any{"run": run.RunID, "resumes": run.Resumes})
		} else if run.HandsOffFrom != "" {
			w.marker("⇄", tr("handed off ", "repassado ")+run.HandsOffFrom+" → "+run.RunID+" ("+run.Adapter+")", cliout.ToneInfo, map[string]any{"run": run.RunID, "hands_off_from": run.HandsOffFrom})
		} else {
			w.marker("▶", tr("run ", "run ")+run.RunID+" ("+run.Adapter+", "+run.Purpose+")", cliout.ToneBold, map[string]any{"run": run.RunID})
		}
		events, err := store.ReadDelegationEvents(run)
		if err != nil {
			w.marker("✗", tr("the events of this run could not be read: ", "os eventos deste run não puderam ser lidos: ")+err.Error(), cliout.ToneBad, map[string]any{"run": run.RunID})
		}
		for _, e := range events {
			if followed && e.Seq <= w.liveSeq && e.Kind != posemodel.DelegationEventFinal {
				continue
			}
			w.event(run.RunID, e)
		}
		end := run.Status
		if run.Interruption != nil {
			end += " (" + run.Interruption.Cause + ")"
		}
		if next := continuedBy[run.RunID]; next != "" {
			end += tr(", continued by ", ", continuado por ") + next
		}
		tone := cliout.ToneOK
		if run.Status != posemodel.ReviewRunCompleted {
			tone = cliout.ToneWarn
		}
		w.marker("■", end, tone, map[string]any{"run": run.RunID, "status": run.Status})
	}
}

// live prints the live events from offset and returns the offset after them.
func (w *watchPrinter) live(store posemodel.Store, attempt posemodel.DelegationAttempt, offset int64) int64 {
	events, next, err := store.ReadDelegationLiveEvents(attempt.AttemptID, offset)
	if err != nil {
		w.lose([]string{"the live events of attempt " + attempt.AttemptID + " could not be read: " + err.Error()})
		return offset
	}
	for _, e := range events {
		w.event("", e)
		if e.Seq > w.liveSeq {
			w.liveSeq = e.Seq
		}
	}
	return next
}

// marker prints a hand-over or a run boundary.
func (w *watchPrinter) marker(symbol, text, tone string, fields map[string]any) {
	if w.f.jsonOut {
		doc := map[string]any{"schema": delegateWatchSchema, "kind": "marker", "text": text}
		for k, v := range fields {
			doc[k] = v
		}
		raw, _ := json.Marshal(doc)
		render(w.out, io.Discard).ContractLine(string(raw))
		return
	}
	render(w.out, io.Discard).ContractLine(cliout.Tint(w.profile, tone, symbol+" "+text))
}

// event prints one normalized event; -v adds the tail of a tool's output.
func (w *watchPrinter) event(run string, e posemodel.DelegationEvent) {
	out := render(w.out, io.Discard)
	if w.f.jsonOut {
		raw, _ := json.Marshal(struct {
			Run string `json:"run,omitempty"`
			posemodel.DelegationEvent
		}{run, e})
		out.ContractLine(string(raw))
		return
	}
	text := strings.Join(strings.Fields(e.Text), " ")
	if len(text) > 160 {
		cut := 157
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut] + "..."
	}
	switch e.Kind {
	case posemodel.DelegationEventTurn:
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneDim, "── turn ──"))
	case posemodel.DelegationEventMessage:
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneBold, "agent: ") + text)
	case posemodel.DelegationEventToolCall:
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneInfo, "▶ ") + text)
	case posemodel.DelegationEventToolOutput:
		code := 0
		if e.ExitCode != nil {
			code = *e.ExitCode
		}
		if code == 0 {
			out.ContractLine(cliout.Tint(w.profile, cliout.ToneOK, "  ✓ exit 0"))
		} else {
			out.ContractLine(cliout.Tint(w.profile, cliout.ToneBad, fmt.Sprintf("  ✗ exit %d", code)))
		}
		if w.f.verbose && strings.TrimSpace(e.Text) != "" {
			lines := strings.Split(strings.TrimRight(e.Text, "\n"), "\n")
			if len(lines) > w.f.lines {
				lines = lines[len(lines)-w.f.lines:]
			}
			for _, line := range lines {
				out.ContractLine(cliout.Tint(w.profile, cliout.ToneDim, "    "+line))
			}
		}
	case posemodel.DelegationEventUsageLimit:
		line := "⏸ usage limit"
		if e.ResetAt != "" {
			line += " (reset " + e.ResetAt + ")"
		}
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneWarn, line))
	case posemodel.DelegationEventFinal:
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneBold, "final: ") + text)
	case posemodel.DelegationEventError:
		out.ContractLine(cliout.Tint(w.profile, cliout.ToneBad, "✗ error: ") + text)
	}
}
