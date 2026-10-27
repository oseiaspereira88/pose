package pose

// The renderers of the commands a handoff section lists, as handoff runs
// recorded before handoff_renderer existed were sent them. They are kept only
// so those runs' proofs still verify (spec pose-delegation-handoff R4); a new
// run always renders with handoffCommands and records its version.

import (
	"fmt"
	"strings"
)

// handoffCommandsV0 paired each output with the last call (eb4ad73).
func handoffCommandsV0(events []DelegationEvent) []string {
	var commands []string
	pending := ""
	for _, e := range events {
		switch e.Kind {
		case DelegationEventToolCall:
			pending = strings.Join(strings.Fields(e.Text), " ")
		case DelegationEventToolOutput:
			code := 0
			if e.ExitCode != nil {
				code = *e.ExitCode
			}
			if pending != "" {
				commands = append(commands, fmt.Sprintf("> `%s` → exit %d", strings.ReplaceAll(pending, "`", "'"), code))
				pending = ""
			}
		}
	}
	return commands
}

// handoffCommandsV1 paired by identifier and by a lone anonymous call, without
// counting identified calls still waiting (8b2da51).
func handoffCommandsV1(events []DelegationEvent) []string {
	type call struct {
		text string
		exit string
	}
	var calls []*call
	byID := map[string]*call{}
	var waiting []*call // calls with no identifier and no output yet
	// unclaimed counts outputs still owed to calls that became ambiguous:
	// until they arrive, no anonymous output is paired by elimination.
	unclaimed := 0
	for _, e := range events {
		switch e.Kind {
		case DelegationEventToolCall:
			c := &call{text: strings.ReplaceAll(strings.Join(strings.Fields(e.Text), " "), "`", "'"), exit: "unknown"}
			calls = append(calls, c)
			if e.Call != "" {
				byID[e.Call] = c
			} else {
				waiting = append(waiting, c)
			}
		case DelegationEventToolOutput:
			code := "unknown"
			if e.ExitCode != nil {
				code = fmt.Sprint(*e.ExitCode)
			}
			switch {
			case e.Call != "" && byID[e.Call] != nil:
				byID[e.Call].exit = code
				delete(byID, e.Call)
			case e.Call == "" && len(waiting) == 1 && unclaimed == 0:
				waiting[0].exit = code
				waiting = nil
			default:
				// Several calls wait, or the call was never recorded: which
				// command this output belongs to is not known, and the calls
				// it may answer stay unknown.
				calls = append(calls, &call{text: "(an output whose command is not known)", exit: code})
				if e.Call == "" {
					unclaimed += len(waiting)
					waiting = nil
					if unclaimed > 0 {
						unclaimed--
					}
				}
			}
		}
	}
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, fmt.Sprintf("> `%s` → exit %s", c.text, c.exit))
	}
	return out
}
