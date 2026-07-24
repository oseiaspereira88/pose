package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func capabilityTriggerTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runStateGit(t, root, "init")
	runStateGit(t, root, "config", "user.email", "test@example.com")
	runStateGit(t, root, "config", "user.name", "Capability Trigger Test")
	return root
}

const triggerAssessmentFixture = `---
schema_version: 1
assessed_at: 2026-07-21
baseline_commit: 38a248d
---

# Capability assessment

## Mechanism: cli-parity
- title: CLI parity
- score: 4
- target: 5
- evidence: check:go test ./...
- paths: internal/cli/*.go

## Mechanism: docs-quality
- title: Docs quality
- score: 3
- target: 5
- evidence: check:manual review
`

func writeTriggerAssessment(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(triggerAssessmentFixture), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssessmentStalenessConsumer_NoAssessmentIsNoOp(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	if err := assessmentStalenessConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo", Commit: "0000000", At: time.Now()}); err != nil {
		t.Fatalf("no-assessment case must never error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "capabilities")); !os.IsNotExist(err) {
		t.Fatal(".pose/capabilities must not be created as a side effect")
	}
}

func TestAssessmentStalenessConsumer_PathsFallbackMarksCapability(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	// A commit touching internal/cli/state.go matches "internal/cli/*.go".
	if err := os.MkdirAll(filepath.Join(root, "internal", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "cli", "state.go"), []byte("package cli\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateGit(t, root, "add", "-A")
	runStateGit(t, root, "commit", "-m", "touch cli")
	commit := gitHeadCommit(root)

	if err := assessmentStalenessConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo-spec", Commit: commit, At: time.Now().UTC()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assessment, err := pose.Store{Root: root}.LoadCapabilityAssessment()
	if err != nil {
		t.Fatal(err)
	}
	cliMechanism := assessment.Mechanisms[0]
	if len(cliMechanism.StaleTriggers) != 1 || cliMechanism.StaleTriggers[0].Trigger != "spec:demo-spec" {
		t.Fatalf("cli-parity stale triggers = %+v", cliMechanism.StaleTriggers)
	}
	if docsMechanism := assessment.Mechanisms[1]; len(docsMechanism.StaleTriggers) != 0 {
		t.Fatalf("docs-quality must be untouched (no paths declared), got %+v", docsMechanism.StaleTriggers)
	}

	log, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, e := range log {
		if e.Consumer == "assessment-staleness" && e.Result == "ok" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an ok assessment-staleness log entry, got %+v", log)
	}
}

func TestAssessmentStalenessConsumer_NoMappingLogsUnavailable(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	// Assessment with no `paths:` declared anywhere, no GraphForge configured.
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	noPaths := `---
schema_version: 1
assessed_at: 2026-07-21
baseline_commit: 38a248d
---

## Mechanism: alpha
- title: Alpha
- score: 3
- target: 5
- evidence: check:x
`
	if err := os.WriteFile(path, []byte(noPaths), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateGit(t, root, "add", "-A")
	runStateGit(t, root, "commit", "-m", "seed")

	if err := assessmentStalenessConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo", Commit: gitHeadCommit(root), At: time.Now()}); err != nil {
		t.Fatalf("must never error: %v", err)
	}
	log, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 1 || log[0].Consumer != "assessment-staleness" || log[0].Result != "skipped" || log[0].Error != "capability_mapping_unavailable" {
		t.Fatalf("expected a visible capability_mapping_unavailable signal, got %+v", log)
	}
}

func TestFilterByMinHits(t *testing.T) {
	hits := map[string][]string{"a": {"x"}, "b": {"x", "y"}}
	got := filterByMinHits(hits, 2)
	if len(got) != 1 {
		t.Fatalf("filterByMinHits(2) = %+v, want only mechanism b", got)
	}
	if _, ok := got["b"]; !ok {
		t.Fatalf("expected mechanism b to survive, got %+v", got)
	}
	if same := filterByMinHits(hits, 1); len(same) != 2 {
		t.Fatalf("filterByMinHits(1) must keep everything, got %+v", same)
	}
}

func TestCollectCapabilityStaleFollowups_AppearsInAggregate(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	content, err := os.ReadFile(filepath.Join(root, ".pose", "capabilities", "assessment.md"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := addStaleMark(string(content), "cli-parity", pose.StaleTrigger{
		Since: "2026-07-23T10:00:00Z", Trigger: "spec:demo-spec", Hits: []string{"component:x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "capabilities", "assessment.md"), []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := collectFollowups(root)
	var found *followup
	for i := range entries {
		if entries[i].Spec == "capability:cli-parity" {
			found = &entries[i]
		}
	}
	if found == nil {
		t.Fatalf("expected a synthetic capability follow-up, got %+v", entries)
	}
	if found.RawDisposition != "open" || found.Target != "spec:demo-spec" || found.Owner != "unowned" {
		t.Fatalf("synthetic follow-up = %+v", found)
	}
	if !strings.Contains(found.Text, "cli-parity") || !strings.Contains(found.Text, "component:x") {
		t.Fatalf("follow-up text missing detail: %q", found.Text)
	}
	if found.Review != "2026-08-06" { // since + 14 default review days
		t.Fatalf("review date = %q, want 2026-08-06", found.Review)
	}
}

func TestCollectCapabilityStaleFollowups_NoMarksIsEmpty(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	if got := collectCapabilityStaleFollowups(root); len(got) != 0 {
		t.Fatalf("expected no synthetic follow-ups, got %+v", got)
	}
}

func TestAssessSnapshot_ClearsStaleMarksAndLinksHistory(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	if code, _, errOut := runAssess(t, root, "init"); code != 0 {
		t.Fatalf("assess init: %v", errOut)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "capabilities", "assessment.md"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := pose.ParseCapabilityAssessment(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	firstMechanism := assessment.Mechanisms[0].ID
	marked, err := addStaleMark(string(raw), firstMechanism, pose.StaleTrigger{
		Since: "2026-07-23T10:00:00Z", Trigger: "spec:demo-spec",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "capabilities", "assessment.md"), []byte(marked), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := runAssess(t, root, "snapshot")
	if code != 0 {
		t.Fatalf("snapshot: %s", errOut)
	}
	if !strings.Contains(out, "Cleared stale marks") || !strings.Contains(out, firstMechanism) {
		t.Fatalf("snapshot output missing clear confirmation: %q", out)
	}

	after, err := pose.Store{Root: root}.LoadCapabilityAssessment()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range after.Mechanisms {
		if m.ID == firstMechanism && len(m.StaleTriggers) != 0 {
			t.Fatalf("mechanism %q still has stale triggers: %+v", firstMechanism, m.StaleTriggers)
		}
	}

	history, err := pose.LoadCapabilityHistory(pose.Store{Root: root}.CapabilityHistoryPath())
	if err != nil {
		t.Fatal(err)
	}
	last := history[len(history)-1]
	if len(last.ClearedStale) != 1 || last.ClearedStale[0] != firstMechanism {
		t.Fatalf("history entry ClearedStale = %+v, want [%s]", last.ClearedStale, firstMechanism)
	}

	// The synthetic follow-up must be gone now that the mark is cleared.
	for _, f := range collectFollowups(root) {
		if f.Spec == "capability:"+firstMechanism {
			t.Fatalf("follow-up for %q should have disappeared after snapshot cleared it", firstMechanism)
		}
	}
}

func TestResolveViaComponentsHit_UsesSpecSlugModeAndDirectLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		params, _ := req.Params.(map[string]any)
		args, _ := params["arguments"].(map[string]any)
		if args["spec_slug"] != "demo-spec" {
			t.Fatalf("expected spec_slug mode, got args=%v", args)
		}
		structured, _ := json.Marshal(graphForgeHitResult{Hits: []map[string]any{
			{"component_id": "component:a", "level": "direct", "capabilities": []any{
				map[string]any{"mechanism_id": "cli-parity"},
			}},
			{"component_id": "component:b", "level": "transitive", "capabilities": []any{
				map[string]any{"mechanism_id": "docs-quality"},
			}},
		}})
		resp := jsonRPCToolCallResult{Result: &struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
			IsError           bool            `json:"isError"`
		}{StructuredContent: structured}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	caller := httpComponentsHitCaller{url: srv.URL, projectID: "demo"}
	hits, ok := resolveViaComponentsHit("", caller, HookEvent{Target: "demo-spec"}, capabilityTriggerPolicy{MinHits: 1, Level: "direct"})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if _, present := hits["cli-parity"]; !present {
		t.Fatalf("direct hit must be included, got %+v", hits)
	}
	if _, present := hits["docs-quality"]; present {
		t.Fatalf("transitive hit must be excluded under direct-only policy, got %+v", hits)
	}

	anyLevel, ok := resolveViaComponentsHit("", caller, HookEvent{Target: "demo-spec"}, capabilityTriggerPolicy{MinHits: 1, Level: "any"})
	if !ok || len(anyLevel) != 2 {
		t.Fatalf("any-level policy must include both mechanisms, got %+v", anyLevel)
	}
}

func TestAssessStale_NoMarksReportsEmpty(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	var stdout, stderr bytes.Buffer
	if code := assessStale(root, nil, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No stale capabilities.") {
		t.Fatalf("expected empty-state message, got %q", stdout.String())
	}
}

func TestAssessStale_NoAssessmentErrors(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	var stdout, stderr bytes.Buffer
	if code := assessStale(root, nil, &stdout, &stderr, localeEN); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestAssessStale_ListsMarkedMechanisms(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	path := filepath.Join(root, ".pose", "capabilities", "assessment.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := addStaleMark(string(raw), "cli-parity", pose.StaleTrigger{
		Since: "2026-07-21T10:00:00Z", Trigger: "spec:demo", Hits: []string{"component:a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := assessStale(root, nil, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cli-parity") || !strings.Contains(stdout.String(), "component:a") {
		t.Fatalf("expected human report to mention mechanism and hit, got %q", stdout.String())
	}

	stdout.Reset()
	if code := assessStale(root, []string{"--json"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var report []staleMechanismReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, stdout.String())
	}
	if len(report) != 1 || report[0].Mechanism != "cli-parity" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestAssessStale_RejectsUnknownFlag(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	var stdout, stderr bytes.Buffer
	if code := assessStale(root, []string{"--bogus"}, &stdout, &stderr, localeEN); code != 2 {
		t.Fatalf("expected exit 2 for unknown flag, got %d", code)
	}
}

func TestAssessRequest_MarksMechanismStale(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	var stdout, stderr bytes.Buffer
	code := assessRequest(root, []string{"--mechanism", "cli-parity", "--reason", "regressao suspeita"}, &stdout, &stderr, localeEN)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}

	store := pose.Store{Root: root}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		t.Fatal(err)
	}
	var mechanism *pose.CapabilityMechanism
	for i := range assessment.Mechanisms {
		if assessment.Mechanisms[i].ID == "cli-parity" {
			mechanism = &assessment.Mechanisms[i]
		}
	}
	if mechanism == nil || len(mechanism.StaleTriggers) != 1 {
		t.Fatalf("expected exactly one stale trigger on cli-parity, got %+v", mechanism)
	}
	if mechanism.StaleTriggers[0].Trigger != "manual:regressao suspeita" {
		t.Fatalf("unexpected trigger text: %q", mechanism.StaleTriggers[0].Trigger)
	}
}

func TestAssessRequest_RequiresMechanismFlag(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	var stdout, stderr bytes.Buffer
	if code := assessRequest(root, nil, &stdout, &stderr, localeEN); code != 2 {
		t.Fatalf("expected exit 2 when --mechanism is missing, got %d", code)
	}
}

func TestAssessRequest_UnknownMechanismErrors(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	writeTriggerAssessment(t, root)
	var stdout, stderr bytes.Buffer
	code := assessRequest(root, []string{"--mechanism", "does-not-exist"}, &stdout, &stderr, localeEN)
	if code != 1 {
		t.Fatalf("expected exit 1 for unknown mechanism, got %d, stderr=%s", code, stderr.String())
	}
}

func TestAssessRequest_NoAssessmentErrors(t *testing.T) {
	root := capabilityTriggerTestRoot(t)
	var stdout, stderr bytes.Buffer
	if code := assessRequest(root, []string{"--mechanism", "cli-parity"}, &stdout, &stderr, localeEN); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}
