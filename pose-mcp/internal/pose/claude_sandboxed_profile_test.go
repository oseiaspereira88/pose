package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Spec pose-claude-sandboxed-profile.

func claudeSandboxedCommand(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "policy", "delegation.json"))
	if err != nil {
		t.Skipf("no delegation policy at the repository root: %v", err)
	}
	var policy struct {
		Agents map[string]struct {
			Commands map[string][]string `json:"commands"`
			Resume   struct {
				Commands map[string][]string `json:"commands"`
			} `json:"resume"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	agent, ok := policy.Agents["claude-opus-5-5"]
	if !ok {
		t.Skip("no Claude agent")
	}
	for _, command := range [][]string{agent.Commands["sandboxed"], agent.Resume.Commands["sandboxed"]} {
		if len(command) == 0 {
			t.Fatal("the Claude agent has no sandboxed command, fresh or resumed")
		}
	}
	return agent.Commands["sandboxed"]
}

// R1: the profile turns the Claude Code sandbox on and forbids unsandboxed commands.
func TestTheClaudeAgentDeclaresASandboxedProfileThatTurnsTheSandboxOn(t *testing.T) {
	command := strings.Join(claudeSandboxedCommand(t), "\x00")
	for _, want := range []string{`"enabled":true`, `"allowUnsandboxedCommands":false`, "--settings", "claude"} {
		if !strings.Contains(command, want) {
			t.Errorf("the sandboxed command lacks %q", want)
		}
	}
}

// R2: without bwrap and socat the command stops with 127, names what is missing and how to install it,
// and never starts the program it was to run.
func TestTheSandboxedProfileRefusesToStartWithoutItsDependenciesAndSaysHowToInstallThem(t *testing.T) {
	command := claudeSandboxedCommand(t)
	if len(command) < 4 || command[0] != "sh" || command[1] != "-c" {
		t.Fatalf("the sandboxed command is not a checked sh -c: %v", command[:min(4, len(command))])
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	marker := filepath.Join(t.TempDir(), "started")
	// The script runs with an empty PATH, so neither dependency is found, and its program is a
	// shell builtin-free path that would leave a marker if it were started.
	cmd := exec.Command(sh, "-c", command[2], "sh", "/bin/sh", "-c", "echo started > "+marker)
	cmd.Env = []string{"PATH=/nonexistent"}
	out, runErr := cmd.CombinedOutput()
	exit := -1
	if ee, ok := runErr.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	}
	if runtimeIsDarwin() {
		t.Skip("macOS has no such dependency")
	}
	if exit != 127 {
		t.Fatalf("exit = %d, want 127: %s", exit, out)
	}
	for _, want := range []string{"bubblewrap and socat", "is missing", "sudo apt install bubblewrap socat", "sudo dnf install bubblewrap socat", "sudo pacman -S bubblewrap socat"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the refusal does not say %q: %s", want, out)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the program was started although the dependencies are missing")
	}
}

// R3 (the check half): with both present the command hands over to its program.
func TestTheSandboxedProfileStartsWhenTheDependenciesArePresent(t *testing.T) {
	command := claudeSandboxedCommand(t)
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	for _, name := range []string{"bwrap", "socat"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	marker := filepath.Join(dir, "started")
	cmd := exec.Command(sh, "-c", command[2], "sh", "/bin/sh", "-c", "echo started > "+marker)
	cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the command failed with both dependencies present: %v %s", err, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the program was not started although both dependencies are present")
	}
}

func runtimeIsDarwin() bool {
	out, err := exec.Command("uname").Output()
	return err == nil && strings.TrimSpace(string(out)) == "Darwin"
}

// R4: an agent that declares requirements for a profile reports which executables are missing,
// and nothing when they are present or the platform is exempt.
func TestTheDelegationPolicyReportsTheMissingRequirementsOfAProfile(t *testing.T) {
	agent := DelegationAgent{Requires: map[string]DelegationRequirement{
		"sandboxed": {Executables: []string{"definitely-not-installed-xyz"}, Install: "install it"},
	}}
	missing := agent.MissingRequirements()
	if got := missing["sandboxed"]; len(got) != 1 || got[0] != "definitely-not-installed-xyz" {
		t.Fatalf("missing = %v", missing)
	}
	agent.Requires["sandboxed"] = DelegationRequirement{Executables: []string{"definitely-not-installed-xyz"}, SkipOn: []string{runtime.GOOS}}
	if len(agent.MissingRequirements()) != 0 {
		t.Fatal("a platform in skip_on still reports missing executables")
	}
	agent.Requires["sandboxed"] = DelegationRequirement{Executables: []string{"sh"}}
	if len(agent.MissingRequirements()) != 0 {
		t.Fatal("an installed executable is reported missing")
	}
}
