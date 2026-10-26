package pose

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const legacyReviewersFixture = `{
  "schema_version": 1,
  "adapters": {
    "codex": {
      "command": ["codex", "exec", "-m", "gpt-6.1-sol", "-s", "read-only", "-o", "{output}", "-"],
      "vendor": "openai", "model": "gpt-6.1-sol", "timeout": "60m", "max_runs_per_bundle": 3,
      "limit": {"output_patterns": ["usage limit"]},
      "session": {"pattern": "session id: ([0-9a-f-]+)"},
      "resume": {"command": ["codex", "exec", "resume", "{session}", "-o", "{output}", "-"], "max_resumes": 2}
    },
    "claude": {
      "command": ["claude", "-p", "--model", "claude-opus-5-5", "--permission-mode", "plan"],
      "vendor": "anthropic", "model": "claude-opus-5-5", "timeout": "45m", "max_runs_per_bundle": 3
    }
  }
}
`

func writePolicy(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// R7: every migrated agent projects the adapter it came from, byte for byte,
// so runs recorded under schema 1 keep verifying their configuration digest.
func TestDelegationMigrationProjectsTheSameAdapters(t *testing.T) {
	legacy, err := parseReviewersPolicy([]byte(legacyReviewersFixture))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := MigrateReviewersPolicy(legacy)
	if err != nil {
		t.Fatal(err)
	}
	view := policy.reviewersView(DelegationPurposeReview)
	for name, adapter := range legacy.Adapters {
		a, _ := json.Marshal(adapter)
		b, _ := json.Marshal(view.Adapters[name])
		if string(a) != string(b) {
			t.Fatalf("%s projects %s, want %s", name, b, a)
		}
	}
	if got := policy.Purposes[DelegationPurposeAdjudicate].MaxRunsPerBundle; got != 3 {
		t.Fatalf("a shared budget moves to every review-path purpose, got %d", got)
	}
}

// R7: the conversion is deterministic and its output is a valid schema 2 policy.
func TestDelegationMigrationIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writePolicy(t, root, reviewersPolicyPath, legacyReviewersFixture)
	first, ok, err := MigrateReviewersPolicyFile(NativeFiles(), root)
	if err != nil || !ok {
		t.Fatalf("migration: ok=%v err=%v", ok, err)
	}
	for i := 0; i < 5; i++ {
		again, _, _ := MigrateReviewersPolicyFile(NativeFiles(), root)
		if string(again) != string(first) {
			t.Fatalf("the conversion is not byte-stable:\n%s\n%s", first, again)
		}
	}
	if !strings.Contains(string(first), `"schema_version": 2`) || !strings.Contains(string(first), `"read": [`) {
		t.Fatalf("unexpected schema 2 bytes:\n%s", first)
	}
	writePolicy(t, root, delegationPolicyPath, string(first))
	if _, ok, _ := MigrateReviewersPolicyFile(NativeFiles(), root); ok {
		t.Fatal("an instance that has delegation.json has nothing to convert")
	}
}

// R7: budgets the adapters do not share stay on each agent.
func TestDelegationMigrationKeepsUnsharedBudgetsPerAgent(t *testing.T) {
	raw := strings.Replace(legacyReviewersFixture, `"timeout": "45m", "max_runs_per_bundle": 3`, `"timeout": "45m", "max_runs_per_bundle": 5`, 1)
	legacy, err := parseReviewersPolicy([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := MigrateReviewersPolicy(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Purposes != nil || policy.Agents["claude"].MaxRunsPerBundle != 5 || policy.Agents["codex"].MaxRunsPerBundle != 3 {
		t.Fatalf("unshared budgets: %+v %+v", policy.Purposes, policy.Agents)
	}
}

// R7: the engine reads schema 1 while schema 2 is absent and refuses both.
func TestDelegationPolicyFilesAreExclusive(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	if _, _, found, err := s.LoadDelegationPolicy(); found || err != nil {
		t.Fatalf("no policy: found=%v err=%v", found, err)
	}
	writePolicy(t, root, reviewersPolicyPath, legacyReviewersFixture)
	policy, path, found, err := s.LoadDelegationPolicy()
	if err != nil || !found || path != reviewersPolicyPath || len(policy.Agents) != 2 {
		t.Fatalf("schema 1 alone: path=%s found=%v err=%v", path, found, err)
	}
	converted, _, _ := MigrateReviewersPolicyFile(NativeFiles(), root)
	writePolicy(t, root, delegationPolicyPath, string(converted))
	if _, _, _, err := s.LoadDelegationPolicy(); !errors.Is(err, ErrDelegationPolicyConflict) {
		t.Fatalf("both files must be refused, got %v", err)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(reviewersPolicyPath))); err != nil {
		t.Fatal(err)
	}
	policy2, path, found, err := s.LoadDelegationPolicy()
	if err != nil || !found || path != delegationPolicyPath || !reflect.DeepEqual(policy2.reviewersView("review"), policy.reviewersView("review")) {
		t.Fatalf("schema 2 must read as the converted schema 1: path=%s err=%v", path, err)
	}
}

const delegationFixture = `{
  "schema_version": 2,
  "agents": {
    "codex": {
      "vendor": "openai", "model": "gpt-6.1-sol", "cli": "codex", "events": "codex-jsonl",
      "commands": {
        "read": ["codex", "exec", "-m", "gpt-6.1-sol", "{effort}", "-s", "read-only", "--json", "-o", "{output}", "-"],
        "write": ["codex", "exec", "-m", "gpt-6.1-sol", "{effort}", "--json", "-o", "{output}", "-"]
      },
      "effort": {"default": "medium", "levels": {"medium": ["-c", "model_reasoning_effort=\"medium\""], "high": ["-c", "model_reasoning_effort=\"high\""]}},
      "timeout": "60m"
    },
    "writer": {
      "vendor": "anthropic", "model": "claude-opus-5-5",
      "commands": {"write": ["claude", "-p"]}
    }
  },
  "routing": {"prefer": "opposite-vendor-of-implementation", "on_unavailable": ["same-vendor-other-model", "wait", "same-model"], "rounds": "continue-thread"},
  "purposes": {"review": {"max_runs_per_bundle": 3}, "adjudicate": {"max_runs_per_bundle": 1, "timeout": "20m"}},
  "env_unset": ["MY_PARENT_SESSION"]
}
`

// R6: a schema 2 policy loads, and each purpose projects its own budget; an
// agent that cannot read is not a reviewer.
func TestDelegationPolicyProjectsPerPurposeBudgets(t *testing.T) {
	policy, err := ParseDelegationPolicy([]byte(delegationFixture))
	if err != nil {
		t.Fatal(err)
	}
	review := policy.reviewersView(DelegationPurposeReview).Adapters["codex"]
	adjudicate := policy.reviewersView(DelegationPurposeAdjudicate).Adapters["codex"]
	if review.MaxRunsPerBundle != 3 || review.Timeout != "60m" || adjudicate.MaxRunsPerBundle != 1 || adjudicate.Timeout != "20m" {
		t.Fatalf("review %+v adjudicate %+v", review, adjudicate)
	}
	if _, ok := policy.reviewersView(DelegationPurposeReview).Adapters["writer"]; ok {
		t.Fatal("an agent without a read command was projected as a reviewer")
	}
}

// R6 and the security rule: the schema is strict and placeholders are whole
// argv elements.
func TestDelegationPolicyRefusesWhatItCannotStateSafely(t *testing.T) {
	cases := map[string]string{
		"unknown field": strings.Replace(delegationFixture, `"cli": "codex",`, `"cli": "codex", "comand": [],`, 1),
		"interpolated output": strings.Replace(delegationFixture, `"-o", "{output}", "-"],
        "write"`, `"--out={output}", "-"],
        "write"`, 1),
		"placeholder as argv0":  strings.Replace(delegationFixture, `"commands": {"write": ["claude", "-p"]}`, `"commands": {"write": ["{output}", "-p"]}`, 1),
		"unknown access":        strings.Replace(delegationFixture, `"commands": {"write": ["claude", "-p"]}`, `"commands": {"admin": ["claude", "-p"]}`, 1),
		"unknown events":        strings.Replace(delegationFixture, `"events": "codex-jsonl"`, `"events": "xml"`, 1),
		"effort without levels": strings.Replace(delegationFixture, `"commands": {"write": ["claude", "-p"]}`, `"commands": {"write": ["claude", "{effort}"]}`, 1),
		"unknown default":       strings.Replace(delegationFixture, `"default": "medium"`, `"default": "max"`, 1),
		"unknown purpose":       strings.Replace(delegationFixture, `"adjudicate": {`, `"adjudication": {`, 1),
		"budget out of range":   strings.Replace(delegationFixture, `"review": {"max_runs_per_bundle": 3}`, `"review": {"max_runs_per_bundle": 101}`, 1),
		"unknown routing step":  strings.Replace(delegationFixture, `"wait", "same-model"`, `"wait", "any-vendor"`, 1),
		"bad env name":          strings.Replace(delegationFixture, `"MY_PARENT_SESSION"`, `"A=B"`, 1),
		"schema 1 number":       strings.Replace(delegationFixture, `"schema_version": 2`, `"schema_version": 1`, 1),
	}
	for name, raw := range cases {
		if raw == delegationFixture {
			t.Fatalf("%s: the fixture edit did not apply", name)
		}
		if _, err := ParseDelegationPolicy([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// B06 of the first review: the schema is strict about explicit budgets and
// about anything after the policy object.
func TestDelegationPolicyRefusesExplicitZeroNullAndTrailingData(t *testing.T) {
	cases := map[string]string{
		"purpose budget zero": strings.Replace(delegationFixture, `"review": {"max_runs_per_bundle": 3}`, `"review": {"max_runs_per_bundle": 0}`, 1),
		"purpose budget null": strings.Replace(delegationFixture, `"review": {"max_runs_per_bundle": 3}`, `"review": {"max_runs_per_bundle": null}`, 1),
		"agent budget null":   strings.Replace(delegationFixture, `"timeout": "60m"`, `"timeout": "60m", "max_runs_per_bundle": null`, 1),
		"empty timeout":       strings.Replace(delegationFixture, `"timeout": "60m"`, `"timeout": ""`, 1),
		"trailing brace":      delegationFixture + "}\n",
		"second object":       delegationFixture + "{}\n",
	}
	for name, raw := range cases {
		if raw == delegationFixture {
			t.Fatalf("%s: the fixture edit did not apply", name)
		}
		if _, err := ParseDelegationPolicy([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := ParseDelegationPolicy([]byte(delegationFixture)); err != nil {
		t.Fatalf("the valid fixture: %v", err)
	}
}

// B03 of the first review: the resume command keeps placeholders whole too.
func TestDelegationResumeCommandKeepsPlaceholdersWhole(t *testing.T) {
	raw := strings.Replace(delegationFixture, `"timeout": "60m"`, `"timeout": "60m",
      "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ["sh", "-c", "cat > {output}", "{session}"]}`, 1)
	if _, err := ParseDelegationPolicy([]byte(raw)); err == nil || !strings.Contains(err.Error(), "resume command") {
		t.Fatalf("an interpolated resume command was accepted: %v", err)
	}
	ok := strings.Replace(raw, `"cat > {output}", "{session}"`, `"resume-reviewer", "{session}", "{output}"`, 1)
	if _, err := ParseDelegationPolicy([]byte(ok)); err != nil {
		t.Fatalf("a whole-element resume command was refused: %v", err)
	}
}

// B01 of the first review and D7: after pose update converts reviewers.json,
// a run recorded before the conversion keeps its policy baseline, and only
// while delegation.json is exactly that conversion.
func TestDelegationMigrationKeepsThePolicyBaselineOfEarlierRuns(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	writePolicy(t, root, ".pose/policy/review.json", `{"schema_version":2}`)
	writePolicy(t, root, reviewersPolicyPath, legacyReviewersFixture)
	before := directoryDigestIn(s.fsys(), filepath.Join(root, ".pose", "policy"))
	converted, ok, err := MigrateReviewersPolicyFile(NativeFiles(), root)
	if err != nil || !ok {
		t.Fatalf("migration: %v %v", ok, err)
	}
	writePolicy(t, root, delegationArchivePath, legacyReviewersFixture)
	writePolicy(t, root, delegationPolicyPath, string(converted))
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(reviewersPolicyPath))); err != nil {
		t.Fatal(err)
	}
	if now := directoryDigestIn(s.fsys(), filepath.Join(root, ".pose", "policy")); now == before {
		t.Fatal("fixture: the conversion did not change the policy directory")
	}
	if got, ok := s.migratedPolicyBaseline(); !ok || got != before {
		t.Fatalf("the earlier baseline is lost: %v %s want %s", ok, got, before)
	}
	writePolicy(t, root, delegationPolicyPath, strings.Replace(string(converted), `"timeout": "45m"`, `"timeout": "50m"`, 1))
	if _, ok := s.migratedPolicyBaseline(); ok {
		t.Fatal("an edited delegation.json inherited the baseline of the policy it replaced")
	}
	writePolicy(t, root, delegationPolicyPath, string(converted))
	writePolicy(t, root, delegationArchivePath, legacyReviewersFixture+" ")
	if _, ok := s.migratedPolicyBaseline(); ok {
		t.Fatal("an archive that is not the converted file restored the baseline")
	}
}
