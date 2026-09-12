// Package cliout renders CLI output.
//
// Spec pose-cli-output-rendering-system; ADR
// 2026-09-11-the-cli-has-one-rendering-layer-and-its-printed-lines-are-a-contract.
//
// Two rules shape this package. Commands emit *semantic events* — a verdict, a
// finding, a step, a field — and never format severities, symbols, colours or
// alignment themselves. And decoration happens at the edge: the capability
// profile decides what a terminal gets, so nothing written to a file, a JSON
// document, a report or an evidence record can carry an escape sequence.
package cliout

import (
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ColorMode is the operator's explicit choice, from --color.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// ParseColorMode reads a --color value. An unknown value is a usage error for
// the caller to report; it never silently becomes a mode.
func ParseColorMode(value string) (ColorMode, bool) {
	switch ColorMode(value) {
	case ColorAuto, ColorAlways, ColorNever:
		return ColorMode(value), true
	case "":
		return ColorAuto, true
	default:
		return ColorAuto, false
	}
}

// Profile is what one destination can do, resolved per stream: stdout and
// stderr are redirected independently, so a terminal on one says nothing about
// the other.
type Profile struct {
	// TTY reports whether the stream is a terminal. Animation exists only
	// here: a repainted line is noise in a log and unreadable to an agent.
	TTY bool
	// Color and Unicode are capabilities, not preferences: a plain profile
	// must carry every fact the decorated one does.
	Color   bool
	Unicode bool
	// Width is the wrapping budget for prose. Contract lines are never
	// wrapped — a wrapped path is a broken path.
	Width int
	// Locale selects the message catalog.
	Locale string
	// Quiet suppresses everything but the verdict; Verbose restores a child
	// process's streamed output.
	Quiet   bool
	Verbose bool
}

// Env reads the environment. Injected so tests never mutate the process.
type Env func(string) string

const (
	defaultWidth = 80
	minWidth     = 20
	maxWidth     = 500
	maxProse     = 100
)

// ResolveProfile derives the profile for one stream.
//
// Colour is off unless a terminal is present, and the precedence is: --color
// (explicit), then POSE_COLOR, then NO_COLOR, then TERM. NO_COLOR is honoured
// whenever it is set to a non-empty value, which is the published convention.
func ResolveProfile(w io.Writer, mode ColorMode, env Env) Profile {
	if env == nil {
		env = os.Getenv
	}
	tty := IsTerminal(w)
	dumb := env("TERM") == "dumb"
	color := false
	switch mode {
	case ColorAlways:
		color = true
	case ColorNever:
		color = false
	default:
		switch env("POSE_COLOR") {
		case "always":
			color = true
		case "never":
			color = false
		default:
			color = tty && !dumb && env("NO_COLOR") == ""
		}
	}
	return Profile{
		TTY:     tty && !dumb,
		Color:   color,
		Unicode: unicodeCapable(env),
		Width:   widthFrom(env),
		Locale:  localeFrom(env),
	}
}

// Plain is the profile a file, a pipe and a test get: every fact, no
// decoration.
func Plain() Profile {
	return Profile{Width: defaultWidth, Locale: LocaleEN}
}

// IsTerminal reports whether w is a character device. Stdlib only: a Stat on
// the file answers this without a dependency.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// unicodeCapable keeps box drawing and symbols away from consoles that would
// render them as noise. A legacy Windows console is assumed unless the
// environment names a terminal that handles UTF-8.
func unicodeCapable(env Env) bool {
	if runtime.GOOS == "windows" {
		return env("WT_SESSION") != "" || strings.Contains(strings.ToUpper(env("TERM_PROGRAM")), "VSCODE")
	}
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := strings.ToUpper(env(key))
		if value == "" {
			continue
		}
		return strings.Contains(value, "UTF-8") || strings.Contains(value, "UTF8")
	}
	// A POSIX shell with no locale set is still overwhelmingly UTF-8 today;
	// the ASCII profile stays one flag away.
	return true
}

func widthFrom(env Env) int {
	if value, err := strconv.Atoi(strings.TrimSpace(env("COLUMNS"))); err == nil && value >= minWidth && value <= maxWidth {
		return value
	}
	return defaultWidth
}

func localeFrom(env Env) string {
	if env("POSE_LOCALE") == LocalePtBR {
		return LocalePtBR
	}
	return LocaleEN
}

// ProseWidth is the budget for wrapped prose: never wider than the reader's
// terminal, never wider than a comfortable measure.
func (p Profile) ProseWidth() int {
	width := p.Width
	if width <= 0 {
		width = defaultWidth
	}
	if width > maxProse {
		return maxProse
	}
	return width
}
