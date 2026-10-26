package pose

// The child of a delegated run never believes it is nested in its parent
// (spec pose-delegation-surface R8). The adapter environment is already an
// allowlist; this removes, even from that allowlist, the session variables a
// known agent host sets for its own children, plus the policy's env_unset, and
// reports which of them the parent actually had. Values are never recorded.

import (
	"os"
	"runtime"
	"sort"
	"strings"
)

// delegationSessionVariables are the variables Claude Code and Codex set to
// tell a process it runs inside one of their sessions.
var delegationSessionVariables = []string{
	"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_CHILD_SESSION",
	"CLAUDE_CODE_BRIDGE_SESSION_ID", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN",
	"CLAUDE_CODE_SESSION_ATTENDED", "CLAUDE_CODE_EXECPATH", "CLAUDE_CODE_SSE_PORT", "CLAUDE_EFFORT", "CLAUDE_PID",
	"CODEX_THREAD_ID", "CODEX_SESSION_ID", "CODEX_CI", "CODEX_SANDBOX", "CODEX_SANDBOX_NETWORK_DISABLED",
}

// DelegationSessionVariables lists the built-in names, sorted.
func DelegationSessionVariables() []string {
	out := append([]string(nil), delegationSessionVariables...)
	sort.Strings(out)
	return out
}

// withoutParentSession drops every name of the built-in list and of unset from
// env, and returns the names among them that the parent process had.
func withoutParentSession(env, unset []string) ([]string, []string) {
	deny := map[string]bool{}
	for _, name := range append(append([]string(nil), delegationSessionVariables...), unset...) {
		deny[envKey(name)] = true
	}
	kept := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		if !deny[envKey(name)] {
			kept = append(kept, entry)
		}
	}
	removed := []string{}
	seen := map[string]bool{}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if deny[envKey(name)] && !seen[envKey(name)] {
			seen[envKey(name)] = true
			removed = append(removed, name)
		}
	}
	sort.Strings(removed)
	return kept, removed
}

func envKey(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(name)
	}
	return name
}
