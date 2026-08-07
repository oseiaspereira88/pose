package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The command reference in POSE.md is what an agent reads to decide which
// command to invoke, so a command missing from it is a shipped capability
// nothing will use. It drifted to 29 of 57 commands undetected until a consumer
// repository hand-edited its own copy (spec pose-command-reference-parity).
//
// `pose help` is the authoritative public surface: this gate asserts the manual
// documents every command the help advertises. Commands deliberately absent
// from the help are internal and equally absent from the manual, so the two
// stay consistent by construction.

var (
	commandBashBlockRE = regexp.MustCompile("(?s)```bash\n(.*?)\n```")
	commandTokenRE     = regexp.MustCompile(`^[a-z][a-z-]{2,}$`)
	commandBracketRE   = regexp.MustCompile(`<[^>]*>|\[[^\]]*\]`)
	// A reference line may group alternatives — `pose a <x> | pose b <y>` — so
	// every `pose <cmd>` on the line counts, not just the one at its start.
	documentedTokenRE = regexp.MustCompile(`(?m)(?:^|\|\s*)pose ([a-z][a-z-]*)`)
)

// helpAdvertisedCommands extracts the command names `pose help` lists. Entries
// appear either one per line with a trailing description or grouped on a single
// line separated by pipes.
func helpAdvertisedCommands(help string) []string {
	seen := map[string]bool{}
	for _, line := range strings.Split(help, "\n") {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(strings.TrimSpace(line), "-") {
			continue
		}
		entry := strings.TrimSpace(line)
		// Drop the description: it starts at the first run of two or more
		// spaces after the command form.
		if idx := strings.Index(entry, "  "); idx > 0 {
			entry = entry[:idx]
		}
		// Placeholders can themselves contain pipes (`<plan|prepare|...>`),
		// so remove them before splitting alternatives.
		entry = commandBracketRE.ReplaceAllString(entry, " ")
		for _, alternative := range strings.Split(entry, "|") {
			fields := strings.Fields(alternative)
			if len(fields) == 0 {
				continue
			}
			if name := fields[0]; commandTokenRE.MatchString(name) {
				seen[name] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// documentedCommands returns the command names the manual's bash reference
// block lists.
func documentedCommands(manual string) map[string]bool {
	documented := map[string]bool{}
	for _, block := range commandBashBlockRE.FindAllStringSubmatch(manual, -1) {
		for _, match := range documentedTokenRE.FindAllStringSubmatch(block[1], -1) {
			documented[match[1]] = true
		}
	}
	return documented
}

// CommandReferenceGaps lists the commands `pose help` advertises that the
// manual's reference block does not document, sorted for stable reporting.
func CommandReferenceGaps(manual, help string) []string {
	documented := documentedCommands(manual)
	var missing []string
	for _, name := range helpAdvertisedCommands(help) {
		if !documented[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// checkCommandReference fails when POSE.md stops documenting a command the CLI
// advertises. Localized manuals are checked against the same help surface: a
// translation that lags is as unusable to an agent as a missing entry.
func (checker *nativeChecker) checkCommandReference() {
	manuals := []string{"POSE.md"}
	if entries, err := os.ReadDir(filepath.Join(checker.root, "locales")); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				manuals = append(manuals, filepath.Join("locales", entry.Name(), "POSE.md"))
			}
		}
	}
	for _, manual := range manuals {
		raw, err := os.ReadFile(filepath.Join(checker.root, manual))
		if err != nil {
			continue
		}
		// A manual with no reference block at all is a different contract
		// (an instance may trim it); only drift within an existing block is
		// reported.
		if !commandBashBlockRE.MatchString(string(raw)) {
			continue
		}
		missing := CommandReferenceGaps(string(raw), helpTextEN)
		if len(missing) == 0 {
			continue
		}
		checker.failOrWarn(fmt.Sprintf(checker.message(
			"command reference: %s omits commands the CLI advertises: %s",
			"referência de comandos: %s omite comandos que a CLI anuncia: %s",
		), filepath.ToSlash(manual), strings.Join(missing, ", ")))
	}
}
