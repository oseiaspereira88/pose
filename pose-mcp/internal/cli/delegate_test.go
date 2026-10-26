package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-delegation-surface: pose delegate is the one surface for delegated
// executions. These tests drive the command as a person or an agent does.

func writeDelegation(t *testing.T, root, body string) {
	t.Helper()
	writeCloseoutCLIFile(t, root, ".pose/policy/delegation.json", body)
}

const approvingAgent = `"ok": {"vendor": "test", "model": "ok-1", "cli": "sh", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "grep -q 'Independent review of spec:alpha' && echo decision: approved"]}}`

func delegationPolicy(agents ...string) string {
	return `{"schema_version": 2, "agents": {` + strings.Join(agents, ",") + `}, "purposes": {"review": {"max_runs_per_bundle": 5}}}`
}

func runPoseSplit(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := 0
	inDir(t, root, func() { code = Main(args, &out, &errB) })
	return code, out.String(), errB.String()
}

// R1, R2, R3: one tree, one vocabulary; what is not delivered yet says who
// delivers it; the review path reads only.
func TestDelegateTreeParsesAndRefusesAsSpecified(t *testing.T) {
	root := dispatchInstance(t)
	if code, out, _ := runPoseSplit(t, root, "delegate"); code != 0 || !strings.Contains(out, "pose delegate") || !strings.Contains(out, "attempt") {
		t.Fatalf("no argument prints the help with the vocabulary: %d %s", code, out)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "reveiw", "spec:alpha"); code != 2 {
		t.Fatalf("an unknown subcommand exits 2, got %d", code)
	}
	for sub, spec := range map[string]string{"watch": "completion-signal", "wait": "completion-signal", "handoff": "pose-delegation-handoff", "task": "pose-delegation-task"} {
		if code, _, errOut := runPoseSplit(t, root, "delegate", sub, "x"); code != 2 || !strings.Contains(errOut, spec) {
			t.Fatalf("%s: %d %s", sub, code, errOut)
		}
	}
	for _, args := range [][]string{
		{"delegate", "review", "spec:alpha", "--access", "write"},
		{"delegate", "review", "spec:alpha", "--detach"},
		{"delegate", "review", "spec:alpha", "--label", "bad label"},
		{"delegate", "review", "spec:alpha", "--lead", "Not A Vendor"},
		{"delegate", "review"},
		{"delegate", "ls", "--wait-until-reset"},
	} {
		if code, _, _ := runPoseSplit(t, root, args...); code != 2 {
			t.Fatalf("%v exits 2, got %d", args, code)
		}
	}
	// R10: with --json even a usage error is one versioned document.
	for _, args := range [][]string{{"delegate", "watch", "x", "--json"}, {"delegate", "review", "--json"}, {"delegate", "nope", "--json"}} {
		code, out, _ := runPoseSplit(t, root, args...)
		var doc map[string]any
		if code != 2 || json.Unmarshal([]byte(out), &doc) != nil || doc["schema"] != delegateErrorSchema {
			t.Fatalf("%v: %d %s", args, code, out)
		}
	}
	if code, out, _ := runPoseSplit(t, root, "help", "delegate"); code != 0 || !strings.Contains(out, "a run is one execution") {
		t.Fatalf("the help defines the terms: %s", out)
	}
}

// R4, R12: auto is the only agent of the purpose; with several it refuses and
// lists them; --dry-run resolves everything and reserves nothing.
func TestDelegateAutoAgentAndDryRun(t *testing.T) {
	root := dispatchInstance(t)
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha"); code != 1 || !strings.Contains(out+errOut, "delegation.json") {
		t.Fatalf("no agent: %d %s %s", code, out, errOut)
	}
	t.Setenv("REVIEWER_TOKEN", "tok-1234567890")
	writeDelegation(t, root, delegationPolicy(approvingAgent, `"other": {"vendor": "test", "model": "o-1", "commands": {"read": ["sh", "-c", "echo tok-1234567890 >/dev/null"]},
      "containment": {"schema_version": 1, "profile": "trusted-process", "credentials": ["REVIEWER_TOKEN"], "retention_ttl_seconds": 0}}`))
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--json")
	if code != 2 || !strings.Contains(out, "ok, other") || !strings.Contains(out, delegateErrorSchema) {
		t.Fatalf("several agents: %d %s %s", code, out, errOut)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "other", "--dry-run", "--json", "--label", "alpha-r1")
	var plan map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &plan) != nil {
		t.Fatalf("dry run: %d %s", code, out)
	}
	argv, _ := json.Marshal(plan["argv"])
	if plan["schema"] != delegatePlanSchema || plan["reserves_attempt"] != false || plan["access"] != "read" || strings.Contains(string(argv), "tok-1234567890") || !strings.Contains(string(argv), "[credential]") {
		t.Fatalf("plan: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "review-attempts")); !os.IsNotExist(err) {
		t.Fatal("a dry run reserved an attempt")
	}
}

// R5 and the validation strategy: the alias and pose delegate record the same
// run apart from label and purpose; the alias names its replacement and is
// absent from the help listing.
func TestDelegateReviewRecordsWhatTheAliasRecords(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	code, out, errOut := runPoseSplit(t, root, "review", "dispatch", "spec:alpha", "--via", "ok", "--apply")
	if code != 0 || !strings.Contains(out, "review_dispatch.status=completed") || !strings.Contains(errOut, "pose delegate review") {
		t.Fatalf("alias: %d %s %s", code, out, errOut)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--reason", "same subject again", "--label", "alpha-r1", "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc["status"] != "completed" || doc["schema"] != delegateRunSchema {
		t.Fatalf("delegate review: %d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 2 {
		t.Fatalf("runs: %+v", runs)
	}
	project := func(r posemodel.ReviewRun) [8]string {
		return [8]string{r.Scope, r.BundleID, r.BundleDigest, r.Kind, r.BriefDigest, r.Adapter, r.Status, r.Output}
	}
	if project(runs[0]) != project(runs[1]) || runs[0].Label != "" || runs[1].Label != "alpha-r1" || runs[1].Purpose != "review" {
		t.Fatalf("records differ beyond label and purpose:\n%+v\n%+v", runs[0], runs[1])
	}
	for _, sub := range []string{"dispatch", "resume", "status"} {
		if _, help, _ := runPoseSplit(t, root, "help", "review"); strings.Contains(help, "pose review "+sub) {
			t.Fatalf("the help still lists pose review %s", sub)
		}
	}
	if code, _, errOut := runPoseSplit(t, root, "review", "status"); code != 0 || !strings.Contains(errOut, "pose delegate ls") {
		t.Fatalf("status alias: %d %s", code, errOut)
	}
}

// R8: the child receives none of the parent's session variables, even when
// the containment allowlist names them, and the run names what was removed.
func TestDelegateChildReceivesNoParentSession(t *testing.T) {
	root := dispatchInstance(t)
	t.Setenv("CODEX_THREAD_ID", "parent-thread")
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("MY_PARENT_SESSION", "parent-session")
	t.Setenv("KEEP_ME", "kept")
	writeDelegation(t, root, `{"schema_version": 2, "env_unset": ["MY_PARENT_SESSION"], "agents": {"envy": {"vendor": "test", "model": "env-1",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; env; echo decision: approved"]},
      "containment": {"schema_version": 1, "profile": "trusted-process", "environment": ["CODEX_THREAD_ID", "CLAUDECODE", "MY_PARENT_SESSION", "KEEP_ME"], "retention_ttl_seconds": 0}}}}`)
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha"); code != 0 {
		t.Fatalf("run: %d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	run := runs[len(runs)-1]
	for _, name := range []string{"CODEX_THREAD_ID", "CLAUDECODE", "MY_PARENT_SESSION"} {
		if strings.Contains(run.Output, name+"=") {
			t.Fatalf("the child received %s:\n%s", name, run.Output)
		}
		if !strings.Contains(" "+strings.Join(run.EnvRemoved, " ")+" ", " "+name+" ") {
			t.Fatalf("env_removed does not name %s: %v", name, run.EnvRemoved)
		}
	}
	if !strings.Contains(run.Output, "KEEP_ME=kept") {
		t.Fatalf("an allowlisted variable was removed:\n%s", run.Output)
	}
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(run.Path)))
	if strings.Contains(string(raw), "parent-thread") || strings.Contains(string(raw), "parent-session") {
		t.Fatal("the record holds a removed variable's value")
	}
}

// R9, R11, R12: a streaming agent's run keeps normalized events; ls shows its
// turns and last step; a run is named by label, by id or by attempt, and an
// ambiguous label exits 2 with the candidates.
func TestDelegateListsAndNamesRuns(t *testing.T) {
	root := dispatchInstance(t)
	stream := `{"type":"turn.started"}
{"type":"item.started","item":{"type":"command_execution","command":"git status"}}
{"type":"item.completed","item":{"type":"command_execution","command":"git status","aggregated_output":"","exit_code":0}}
{"type":"item.completed","item":{"type":"agent_message","text":"decision: approved"}}`
	streamFile := filepath.Join(t.TempDir(), "stream.jsonl")
	if err := os.WriteFile(streamFile, []byte(stream+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDelegation(t, root, delegationPolicy(`"streamer": {"vendor": "test", "model": "s-1", "events": "codex-jsonl",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; cat `+streamFile+`"]}}`))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "twice"); code != 0 {
		t.Fatalf("first: %d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if runs[0].EventsDigest == "" || strings.TrimSpace(runs[0].Output) != "decision: approved" {
		t.Fatalf("the stream's final message is the draft and its events are bound: %+v", runs[0])
	}
	code, out, _ := runPoseSplit(t, root, "delegate", "ls", "twice", "--json")
	var listing struct {
		Schema   string                        `json:"schema"`
		Attempts []posemodel.DelegationAttempt `json:"attempts"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &listing) != nil || len(listing.Attempts) != 1 {
		t.Fatalf("ls by label: %d %s", code, out)
	}
	a := listing.Attempts[0]
	if listing.Schema != delegateListSchema || a.Turns != 1 || a.State != "completed" || a.Purpose != "review" || a.Agent != "streamer" || !strings.Contains(a.LastStep, "decision: approved") {
		t.Fatalf("row: %+v", a)
	}
	for _, ref := range []string{a.AttemptID, a.LatestRun} {
		if code, _, _ := runPoseSplit(t, root, "delegate", "ls", ref); code != 0 {
			t.Fatalf("ls %s exits %d", ref, code)
		}
	}
	runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "twice", "--reason", "second look")
	code, out, _ = runPoseSplit(t, root, "delegate", "ls", "twice", "--json")
	var ambiguous struct {
		Candidates []string `json:"candidates"`
	}
	if code != 2 || json.Unmarshal([]byte(out), &ambiguous) != nil || len(ambiguous.Candidates) != 2 {
		t.Fatalf("an ambiguous label: %d %s", code, out)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "ls", "nothing-named-so"); code != 2 {
		t.Fatalf("an unknown reference exits 2, got %d", code)
	}
}

// R13 and D2: cancelling a running run records it failed with the cause
// cancelled, never interrupted, and its attempt stays consumed.
func TestDelegateCancelRecordsAFailedCancelledRun(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, `{"schema_version": 2, "agents": {"sleepy": {"vendor": "test", "model": "z-1", "timeout": "60s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; sleep 30"]}}}, "purposes": {"review": {"max_runs_per_bundle": 1}}}`)
	t.Chdir(root)
	done := make(chan int, 1)
	go func() {
		var out, errB bytes.Buffer
		done <- Main([]string{"delegate", "review", "spec:alpha", "--label", "sleepy-run"}, &out, &errB)
	}()
	deadline := time.Now().Add(20 * time.Second)
	for {
		var out, errB bytes.Buffer
		Main([]string{"delegate", "ls", "--active", "--json"}, &out, &errB)
		if strings.Contains(out.String(), `"state": "running"`) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run never showed as running: %s", out.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
	var out, errB bytes.Buffer
	if code := Main([]string{"delegate", "cancel", "sleepy-run", "--json"}, &out, &errB); code != 0 || !strings.Contains(out.String(), `"stopped": true`) {
		t.Fatalf("cancel: %s %s", out.String(), errB.String())
	}
	select {
	case code := <-done:
		if code != 1 {
			t.Fatalf("a cancelled run exits 1, got %d", code)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the run did not stop")
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Status != posemodel.ReviewRunFailed || runs[0].Interruption != nil || !strings.Contains(runs[0].Failure, "cancelled") || runs[0].Safety == nil || runs[0].Safety.ExecutionCause != "cancelled" {
		t.Fatalf("record: %+v", runs)
	}
	out.Reset()
	if code := Main([]string{"delegate", "review", "spec:alpha", "--reason", "after a cancel"}, &out, &errB); code == 0 || !strings.Contains(out.String()+errB.String(), "budget") {
		t.Fatalf("the cancelled attempt returned its budget: %d %s %s", code, out.String(), errB.String())
	}
}

// R14: agents shows vendor, model, whether the CLI is on the path and the
// reset instant of the last recorded usage limit.
func TestDelegateAgentsShowsAvailability(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(`"limited": {"vendor": "test", "model": "l-1", "cli": "sh",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'LIMIT reset: 2099-01-01T00:00:00Z'; exit 1"]},
      "limit": {"output_patterns": ["LIMIT"], "reset_pattern": "reset: (\\S+)"}}`,
		`"absent": {"vendor": "test2", "model": "a-1", "cli": "pose-no-such-cli", "commands": {"read": ["pose-no-such-cli"]}}`))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "limited"); code != 1 || !strings.Contains(out, "interrupted") {
		t.Fatalf("limited run: %d %s %s", code, out, errOut)
	}
	code, out, _ := runPoseSplit(t, root, "delegate", "agents", "--json")
	var doc struct {
		Schema string             `json:"schema"`
		Agents []delegateAgentRow `json:"agents"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || len(doc.Agents) != 2 || doc.Schema != delegateAgentsSchema {
		t.Fatalf("agents: %d %s", code, out)
	}
	byName := map[string]delegateAgentRow{}
	for _, row := range doc.Agents {
		byName[row.Name] = row
	}
	if !byName["limited"].OnPath || byName["absent"].OnPath || byName["limited"].LimitResetAt != "2099-01-01T00:00:00Z" || !byName["limited"].Limited || byName["limited"].Events != "plain" {
		t.Fatalf("rows: %+v", byName)
	}
}

// R7: pose update converts reviewers.json into delegation.json, shows it in
// its preview, and the instance reads the same adapters afterwards.
func TestUpdateConvertsTheReviewersPolicy(t *testing.T) {
	repo := freshInstall(t)
	writeReviewers(t, repo, map[string]posemodel.ReviewerAdapter{
		"codex": {Command: []string{"codex", "exec", "-o", "{output}", "-"}, Vendor: "openai", Model: "gpt-6.1-sol", Timeout: "60m", MaxRunsPerBundle: 3},
	})
	before, _, _ := posemodel.Store{Root: repo}.LoadReviewersPolicyFor("review")
	want, ok, err := posemodel.MigrateReviewersPolicyFile(posemodel.NativeFiles(), repo)
	if err != nil || !ok {
		t.Fatalf("fixture: %v %v", ok, err)
	}
	code, out := runPose(t, repo, "update", "--dry-run")
	if code != 0 || !strings.Contains(out, "delegation.json") || !strings.Contains(out, "reviewers.json") {
		t.Fatalf("the preview does not show the conversion: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "policy", "delegation.json")); !os.IsNotExist(err) {
		t.Fatal("the preview wrote")
	}
	if code, out := runPose(t, repo, "update", "--no-self"); code != 0 {
		t.Fatalf("update: %s", out)
	}
	got, err := os.ReadFile(filepath.Join(repo, ".pose", "policy", "delegation.json"))
	if err != nil || string(got) != string(want) {
		t.Fatalf("delegation.json:\n%s\nwant:\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "policy", "reviewers.json")); !os.IsNotExist(err) {
		t.Fatal("reviewers.json was kept beside delegation.json")
	}
	after, _, _ := posemodel.Store{Root: repo}.LoadReviewersPolicyFor("review")
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatalf("the instance reads other adapters after the conversion:\n%s\n%s", a, b)
	}
}

// R7 and the security rule: a schema 1 adapter that interpolates a
// placeholder into code is not converted; the update warns and keeps it.
func TestUpdateKeepsAReviewersPolicyItCannotConvertSafely(t *testing.T) {
	repo := freshInstall(t)
	writeReviewers(t, repo, map[string]posemodel.ReviewerAdapter{
		"shell": {Command: []string{"sh", "-c", "cat > {output}"}, Vendor: "test", Model: "s-1"},
	})
	code, out := runPose(t, repo, "update", "--no-self")
	if !strings.Contains(out, "delegation policy migration") || !strings.Contains(out, "whole argv element") {
		t.Fatalf("the update did not warn: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "policy", "delegation.json")); !os.IsNotExist(err) {
		t.Fatal("an unsafe template was converted")
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "policy", "reviewers.json")); err != nil {
		t.Fatal("reviewers.json was removed without a conversion")
	}
	if policy, found, err := (posemodel.Store{Root: repo}).LoadReviewersPolicyFor("review"); err != nil || !found || len(policy.Adapters) != 1 {
		t.Fatalf("the legacy policy no longer reads: %v %v", found, err)
	}
}
