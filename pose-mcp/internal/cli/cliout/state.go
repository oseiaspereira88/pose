package cliout

// One severity vocabulary (spec pose-cli-output-rendering-system R3).
//
// Before this, three dialects coexisted — `[ERROR]/[WARN]/[INFO]`, `Error:` and
// one command's private `[pose-install]` prefix — and a state's look was decided
// at each call site. Here a state has exactly one word, one symbol, one ASCII
// fallback and one colour, and the word is always printed: colour is never the
// only carrier of meaning.

import "strings"

type State int

const (
	StatePass State = iota
	StateFail
	StateError
	StateWarning
	StateSkipped
	StateInfo
	StateHint
)

// Key is the machine-facing name, shared with the JSON channel and with the
// evidence vocabularies POSE already closes over.
func (s State) Key() string {
	switch s {
	case StatePass:
		return "pass"
	case StateFail:
		return "fail"
	case StateError:
		return "error"
	case StateWarning:
		return "warning"
	case StateSkipped:
		return "skipped"
	case StateInfo:
		return "info"
	case StateHint:
		return "hint"
	}
	return "info"
}

// StateFromKey maps a stored severity back to a state. An unknown severity
// renders as info rather than inventing a look for it.
func StateFromKey(key string) State {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "pass", "passed", "ok", "success":
		return StatePass
	case "fail", "failed", "failure":
		return StateFail
	case "error", "critical":
		return StateError
	case "warn", "warning":
		return StateWarning
	case "skip", "skipped":
		return StateSkipped
	case "hint":
		return StateHint
	default:
		return StateInfo
	}
}

// Word is the state in the reader's language. It is printed in every profile.
func (s State) Word(locale string) string {
	return Msg(stateWord(s), locale)
}

func stateWord(s State) MessageID {
	switch s {
	case StatePass:
		return MsgStatePass
	case StateFail:
		return MsgStateFail
	case StateError:
		return MsgStateError
	case StateWarning:
		return MsgStateWarning
	case StateSkipped:
		return MsgStateSkipped
	case StateHint:
		return MsgStateHint
	default:
		return MsgStateInfo
	}
}

// Symbol is the glyph for the state, or its ASCII fallback when the terminal
// cannot be trusted with more.
func (s State) Symbol(p Profile) string {
	if p.Unicode {
		switch s {
		case StatePass:
			return "✔"
		case StateFail:
			return "✖"
		case StateError:
			return "✖"
		case StateWarning:
			return "⚠"
		case StateSkipped:
			return "–"
		case StateHint:
			return "💡"
		default:
			return "ℹ"
		}
	}
	switch s {
	case StatePass:
		return "[ok]"
	case StateFail:
		return "[x]"
	case StateError:
		return "[err]"
	case StateWarning:
		return "[!]"
	case StateSkipped:
		return "[-]"
	case StateHint:
		return "[hint]"
	default:
		return "[i]"
	}
}

// SGR codes stay in the 4-bit range: CI log viewers and plain terminals render
// them, and 256-colour would buy nothing a gate needs.
const (
	sgrReset  = "\x1b[0m"
	sgrRed    = "\x1b[31m"
	sgrGreen  = "\x1b[32m"
	sgrYellow = "\x1b[33m"
	sgrBlue   = "\x1b[34m"
	sgrCyan   = "\x1b[36m"
	sgrDim    = "\x1b[2m"
	sgrBold   = "\x1b[1m"
)

func (s State) color() string {
	switch s {
	case StatePass:
		return sgrGreen
	case StateFail, StateError:
		return sgrRed
	case StateWarning:
		return sgrYellow
	case StateSkipped:
		return sgrDim
	case StateHint:
		return sgrCyan
	default:
		return sgrBlue
	}
}

// paint applies an SGR sequence only when the destination is a terminal that
// asked for colour. Every other caller gets the string unchanged, which is what
// keeps escapes out of files and evidence.
func paint(p Profile, sgr, text string) string {
	if !p.Color || sgr == "" || text == "" {
		return text
	}
	return sgr + text + sgrReset
}
