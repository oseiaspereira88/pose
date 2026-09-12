package cliout

// Steps and progress (spec pose-cli-output-rendering-system R6).
//
// `pose validate` streamed each check's raw output for minutes and printed no
// per-check outcome, duration or counter at all: a human watching a gate could
// not tell what was running or how far it had got. A step is the unit of work
// that fixes that — started, then resolved with an outcome and a duration.
//
// The renderer, not the command, decides how a step looks. On a terminal the
// active step is one line that repaints — spinner, label, counter, elapsed —
// replaced in place by its resolved line. Anywhere else the same facts arrive as
// one line per event, with no escape sequence, which is what a CI log and an
// agent need. Animation is an affordance of the terminal, never a data channel.

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// spinnerFrames are braille cells on a capable terminal and ASCII elsewhere.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var spinnerFramesASCII = []string{"|", "/", "-", "\\"}

const spinnerInterval = 120 * time.Millisecond

// StepSet is one run of work: a group of steps with a shared counter and a
// shared elapsed clock.
type StepSet struct {
	r     *Renderer
	w     io.Writer
	p     Profile
	total int

	mu       sync.Mutex
	now      func() time.Time
	tick     time.Duration
	started  time.Time
	done     int
	counts   map[State]int
	active   *Step
	frame    int
	painted  bool
	stopping chan struct{}
	stopped  chan struct{}
}

// Step is one unit of work in progress.
type Step struct {
	set     *StepSet
	label   string
	detail  string
	started time.Time
	ended   bool
}

// Steps starts a group of steps. Progress is written to Err: it is not the
// command's result, and keeping it off stdout is what lets `pose x --json | jq`
// stay safe while a human still sees movement.
func (r *Renderer) Steps(total int) *StepSet {
	// The group's clock starts with its first step, not with its creation: a
	// caller that resolves arguments before starting work should not have that
	// time counted as the run's.
	return &StepSet{r: r, w: r.err, p: r.errP, total: total, now: time.Now, tick: spinnerInterval, counts: map[State]int{}}
}

// animated reports whether this destination may repaint. A redirected stream,
// a dumb terminal and --quiet all get plain lines instead.
func (s *StepSet) animated() bool {
	return s.p.TTY && !s.p.Quiet && !s.p.Verbose
}

// Start opens a step. label is the unit's name; detail is what it runs.
func (s *StepSet) Start(label, detail string) *Step {
	s.mu.Lock()
	if s.started.IsZero() {
		s.started = s.now()
	}
	step := &Step{set: s, label: label, detail: detail, started: s.now()}
	s.active = step
	s.mu.Unlock()
	if s.p.Quiet {
		return step
	}
	if !s.animated() {
		fmt.Fprintf(s.w, "  -> %s\n", strings.TrimSpace(label+" "+detail))
		return step
	}
	s.paint()
	s.startTicker()
	return step
}

// Resolve closes a step with its outcome. note is an optional short reason —
// an exit code, a skip reason — and never a wall of output.
func (st *Step) Resolve(state State, note string) {
	if st == nil || st.ended {
		return
	}
	st.ended = true
	s := st.set
	elapsed := s.now().Sub(st.started)
	s.stopTicker()
	s.mu.Lock()
	s.done++
	s.counts[state]++
	s.active = nil
	s.mu.Unlock()
	if s.p.Quiet {
		return
	}
	if s.animated() {
		s.clear()
		line := fmt.Sprintf("  %s %s %s", state.Symbol(s.p), paint(s.p, state.color(), st.label), paint(s.p, sgrDim, st.detail))
		line = strings.TrimRight(line, " ")
		line += "  " + paint(s.p, sgrDim, Msg(MsgElapsed, s.r.locale, elapsed.Seconds()))
		if note != "" {
			line += "  " + paint(s.p, sgrDim, note)
		}
		fmt.Fprintln(s.w, line)
		return
	}
	line := fmt.Sprintf("  <- %s %s %s", st.label, state.Key(), Msg(MsgElapsed, s.r.locale, elapsed.Seconds()))
	if note != "" {
		line += " " + note
	}
	fmt.Fprintln(s.w, line)
}

// Detail prints output a step produced — the tail of a failing check, for
// instance — indented under it, with a pointer to where the whole of it lives.
func (st *Step) Detail(text, fullOutputPath string) {
	if st == nil || st.set.p.Quiet || strings.TrimSpace(text) == "" {
		return
	}
	s := st.set
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		fmt.Fprintf(s.w, "      %s\n", line)
	}
	if fullOutputPath != "" {
		fmt.Fprintf(s.w, "      %s\n", paint(s.p, sgrDim, Msg(MsgFullOut, s.r.locale, fullOutputPath)))
	}
}

// Note prints a progress line that belongs to the group rather than to one
// step — a module header, for instance. Any painted status line is cleared
// first, so the two never overwrite each other.
func (s *StepSet) Note(text string) {
	if s.p.Quiet || strings.TrimSpace(text) == "" {
		return
	}
	s.clear()
	fmt.Fprintln(s.w, text)
	s.mu.Lock()
	active := s.active != nil
	s.mu.Unlock()
	if active {
		s.paint()
	}
}

// Summary closes the group with the counts and the elapsed time.
func (s *StepSet) Summary() {
	s.stopTicker()
	if s.p.Quiet {
		return
	}
	s.mu.Lock()
	done, counts := s.done, s.counts
	elapsed := time.Duration(0)
	if !s.started.IsZero() {
		elapsed = s.now().Sub(s.started)
	}
	s.mu.Unlock()
	parts := []string{}
	for _, state := range []State{StatePass, StateFail, StateError, StateWarning, StateSkipped} {
		if counts[state] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[state], state.Word(s.r.locale)))
		}
	}
	fmt.Fprintln(s.w, Msg(MsgStepsDone, s.r.locale, done, strings.Join(parts, " · "), Msg(MsgElapsed, s.r.locale, elapsed.Seconds())))
}

// Counts reports what the group resolved, for a caller that needs the same
// numbers in its machine channel.
func (s *StepSet) Counts() map[State]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[State]int{}
	for state, count := range s.counts {
		out[state] = count
	}
	return out
}

// paint writes the active step's status line in place. Callers hold no lock.
func (s *StepSet) paint() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paintLocked()
}

func (s *StepSet) paintLocked() {
	if s.active == nil || !s.animated() {
		return
	}
	frames := spinnerFrames
	if !s.p.Unicode {
		frames = spinnerFramesASCII
	}
	spin := frames[s.frame%len(frames)]
	elapsed := Msg(MsgElapsed, s.r.locale, s.now().Sub(s.active.started).Seconds())
	counter := ""
	if s.total > 0 {
		counter = fmt.Sprintf(" (%d/%d)", s.done+1, s.total)
	}
	line := fmt.Sprintf("  %s %s%s %s %s", paint(s.p, sgrCyan, spin), s.active.label, counter, paint(s.p, sgrDim, s.active.detail), paint(s.p, sgrDim, elapsed))
	line = truncate(line, s.p.Width)
	fmt.Fprintf(s.w, "\r\x1b[K%s", line)
	s.painted = true
}

// clear erases the status line so a resolved line or another writer can take
// the terminal cleanly.
func (s *StepSet) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.painted {
		fmt.Fprint(s.w, "\r\x1b[K")
		s.painted = false
	}
}

// Release hands the terminal back — used before a child process streams under
// --verbose, so two writers never fight over one line.
func (s *StepSet) Release() { s.stopTicker(); s.clear() }

func (s *StepSet) startTicker() {
	if !s.animated() || s.stopping != nil {
		return
	}
	s.stopping, s.stopped = make(chan struct{}), make(chan struct{})
	go func(stop <-chan struct{}, done chan<- struct{}, interval time.Duration) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.frame++
				s.paintLocked()
				s.mu.Unlock()
			}
		}
	}(s.stopping, s.stopped, s.tick)
}

func (s *StepSet) stopTicker() {
	if s.stopping == nil {
		return
	}
	close(s.stopping)
	<-s.stopped
	s.stopping, s.stopped = nil, nil
}

func truncate(line string, width int) string {
	if width <= 0 {
		width = defaultWidth
	}
	runes := []rune(line)
	// Escape sequences are not printable width; the budget is generous enough
	// that a rune count keeps the line inside the terminal in practice.
	if len(runes) <= width {
		return line
	}
	return string(runes[:width-1]) + "…"
}
