package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Spec pose-claude-reviewer-runs-tests-in-sandbox.
func sandboxedReviewFixture(t *testing.T, profile string) []byte {
	t.Helper()
	legacy, err := parseReviewersPolicy([]byte(legacyReviewersFixture))
	if err != nil {
		t.Fatal(err)
	}
	p, err := MigrateReviewersPolicy(legacy)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if profile != "" {
		var value any
		if err := json.Unmarshal([]byte(profile), &value); err != nil {
			t.Fatal(err)
		}
		doc["agents"].(map[string]any)["codex"].(map[string]any)["review_profile"] = value
	}
	raw, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSandboxedReviewSelectsOnlyReviewAndKeepsResumeConfiguration(t *testing.T) {
	p, err := ParseDelegationPolicy(sandboxedReviewFixture(t, `{"command":["sandbox","fresh","{output}"],"resume_command":["sandbox","resume","{session}","{output}"]}`))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(p.Agents["codex"])
	for _, purpose := range []string{DelegationPurposeReview, DelegationPurposeAdjudicate, DelegationPurposeSmoke, DelegationPurposeTask} {
		a, ok := p.agentAdapter("codex", purpose)
		if !ok {
			t.Fatal("missing agent")
		}
		want := "codex"
		if purpose == DelegationPurposeReview {
			want = "sandbox"
		}
		if a.Command[0] != want || a.Resume.Command[0] != want {
			t.Fatalf("%s: wrong command/resume: %+v", purpose, a)
		}
		if a.Resume.MaxResumes != 2 || a.Session.Pattern == "" {
			t.Fatalf("continuity lost: %+v", a)
		}
	}
	a, _ := p.agentAdapter("codex", DelegationPurposeReview)
	a.Resume.Command[0] = "changed"
	a.Command[0] = "changed"
	after, _ := json.Marshal(p.Agents["codex"])
	if string(before) != string(after) {
		t.Fatal("projecting/mutating review modified shared task/read configuration")
	}
}

func TestSandboxedReviewOmissionPreservesLegacyAdapter(t *testing.T) {
	legacy, err := parseReviewersPolicy([]byte(legacyReviewersFixture))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseDelegationPolicy(sandboxedReviewFixture(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range legacy.Adapters {
		got, _ := p.agentAdapter(name, DelegationPurposeReview)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s adapter changed: %+v != %+v", name, got, want)
		}
	}
}

func TestSandboxedReviewRefusesIncompleteOrUnsafeCommands(t *testing.T) {
	for _, profile := range []string{
		`{}`, `{"command":[]}`, `{"command":["sandbox"]}`,
		`{"command":["sandbox"],"resume_command":["sandbox","resume"]}`,
		`{"command":["{output}"],"resume_command":["sandbox","{session}"]}`,
		`{"command":["sandbox","--output={output}"],"resume_command":["sandbox","{session}"]}`,
		`{"command":["sandbox","{outside_git}"],"resume_command":["sandbox","{session}"]}`,
		`{"command":["sandbox"],"resume_command":["sh","-c","{session}"]}`,
	} {
		if _, err := ParseDelegationPolicy(sandboxedReviewFixture(t, profile)); err == nil {
			t.Errorf("accepted %s", profile)
		}
	}
}

func sandboxedClaudeReviewCommands(t *testing.T) [][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", delegationPolicyPath))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseDelegationPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := p.agentAdapter("claude-opus-5-5", DelegationPurposeReview)
	return [][]string{a.Command, a.Resume.Command}
}

func TestSandboxedReviewClaudeEnforcesFreshAndResumedShellBoundaries(t *testing.T) {
	for i, argv := range sandboxedClaudeReviewCommands(t) {
		joined := strings.Join(argv, "\x00")
		for _, want := range []string{`"enabled":true`, `"failIfUnavailable":true`, `"allowUnsandboxedCommands":false`, `"denyWrite":["."]`, "--safe-mode", "--restricted", "--strict-mcp-config", "Read,Grep,Glob,Bash", "GOCACHE", "GOTMPDIR", "CLAUDE_CODE_TMPDIR", "{output}", "--append-system-prompt", "go clean -cache", "inside the mandatory sandbox"} {
			if !strings.Contains(joined, want) {
				t.Errorf("command %d lacks %s", i, want)
			}
		}
		if strings.Contains(joined, "Edit,Write") || strings.Contains(joined, "bypassPermissions") {
			t.Fatalf("writing tools/permission bypass in %v", argv)
		}
		if i == 1 && !containsString(argv, "{session}") {
			t.Fatal("resume lacks session")
		}
	}
}

func TestSandboxedReviewDependencyFailuresNeverLaunchClaude(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, argv := range sandboxedClaudeReviewCommands(t) {
		if len(argv) < 5 || argv[0] != "sh" || argv[1] != "-c" {
			t.Fatalf("no dependency wrapper: %v", argv)
		}
		for _, missing := range []string{"bwrap", "socat", "jq", "both"} {
			dir := t.TempDir()
			for _, name := range []string{"uname", "bwrap", "socat", "jq"} {
				if name == missing || missing == "both" && name != "uname" {
					continue
				}
				body := "#!/bin/sh\nexit 0\n"
				if name == "uname" {
					body = "#!/bin/sh\necho Linux\n"
				}
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			marker := filepath.Join(dir, "launched")
			cmd := exec.Command(sh, "-c", argv[2], "sh", filepath.Join(dir, "output"), sh, "-c", "echo yes > \"$1\"", "sh", marker)
			cmd.Env = []string{"PATH=" + dir}
			out, err := cmd.CombinedOutput()
			if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 127 {
				t.Fatalf("missing %s: %v %s", missing, err, out)
			}
			for _, want := range []string{"bubblewrap", "socat", "Install", "apt", "dnf", "pacman"} {
				if !strings.Contains(string(out), want) {
					t.Errorf("missing %s diagnostic: %s", want, out)
				}
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("Claude started without dependencies")
			}
		}
	}
}

func TestSandboxedReviewWrapperKeepsScratchPrivateAndPropagatesStartupFailure(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("no jq for the qualified Claude profile")
	}
	for _, argv := range sandboxedClaudeReviewCommands(t) {
		dir := t.TempDir()
		bin := filepath.Join(dir, "bin")
		if err := os.Mkdir(bin, 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"uname", "bwrap", "socat"} {
			body := "#!/bin/sh\nexit 0\n"
			if name == "uname" {
				body = "#!/bin/sh\necho Linux\n"
			}
			if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		fake := filepath.Join(bin, "claude")
		body := `#!/bin/sh
test -d "$GOCACHE" && test -d "$GOTMPDIR" && test "$TMPDIR" = "$GOTMPDIR" || exit 90
test "$1" = --settings || exit 91
printf '%s' "$2" | jq -e --arg scratch "$CLAUDE_CODE_TMPDIR" --arg cwd "$PWD" '.sandbox.failIfUnavailable == true and .sandbox.allowUnsandboxedCommands == false and .sandbox.filesystem.allowWrite == [$scratch] and .sandbox.filesystem.denyWrite == [$cwd] and .sandbox.filesystem.disabled == false and .sandbox.allowNestedUserNamespaces == false' >/dev/null || exit 92
echo sandbox-initialization-refused >&2
exit 78
`
		if err := os.WriteFile(fake, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(sh, "-c", argv[2], "sh", filepath.Join(dir, "output"), argv[5], fake)
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 78 || !strings.Contains(string(out), "sandbox-initialization-refused") {
			t.Fatalf("startup failure not propagated: %v %s", err, out)
		}
		info, err := os.Stat(filepath.Join(dir, "claude-review"))
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("scratch not private: %v %v", info, err)
		}
	}
}

func TestSandboxedReviewCaptureFailureNamesTheSafeStageAndKeepsOutputWithheld(t *testing.T) {
	f := provenanceRunFixture(t, "anthropic", "review-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", "cat >/dev/null; printf 'session: synthetic-a\\nsession: synthetic-b\\n'"}
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
	})
	if f.run.Status != ReviewRunFailed || f.run.Output != "" || f.run.Safety.CaptureComplete {
		t.Fatalf("unsafe capture was not refused: %+v", f.run)
	}
	for _, want := range []string{"adapter capture failed", "session/interrupt recognition", "ambiguous adapter capture"} {
		if !strings.Contains(f.run.Failure, want) {
			t.Errorf("missing safe diagnostic %q: %s", want, f.run.Failure)
		}
	}
	if strings.Contains(f.run.Failure, "synthetic-a") || strings.Contains(f.run.Failure, "synthetic-b") {
		t.Fatal("raw capture leaked")
	}
}

func TestSandboxedReviewSessionPatternCapturesAnActualInitEvent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", delegationPolicyPath))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseDelegationPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	value, err := captureReviewValue(p.Agents["claude-opus-5-5"].Session.Pattern, `{"type":"system","subtype":"init","session_id":"synthetic-session"}`)
	if err != nil || value != "synthetic-session" {
		t.Fatalf("init capture: %q %v", value, err)
	}
}
