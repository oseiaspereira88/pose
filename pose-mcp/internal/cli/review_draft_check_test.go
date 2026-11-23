package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-readiness-before-seal R3: draft-check reports what the
// conclusion step would refuse, before the run ends, and writes nothing.
func TestReviewDraftCheckMatchesTheConclusionStep(t *testing.T) {
	contextCopy := filepath.Join(t.TempDir(), "context.json")
	root, draftPath, bundleID := cliProvenanceInstance(t, func(root, draftPath string) {
		reviewers := posemodel.ReviewersPolicy{SchemaVersion: 1, Provenance: &posemodel.ReviewRunProvenancePolicy{Trust: "executor-attested", Implementation: posemodel.ModelIdentity{Principal: "agent:implementer", Execution: "implementation-fixture", Vendor: "openai", Model: "implementation-model"}}, Adapters: map[string]posemodel.ReviewerAdapter{"fake": {Executor: "agent:reviewer", Command: []string{"sh", "-c", "cat >/dev/null; cp \"$POSE_REVIEW_DRAFT_CONTEXT\" '" + contextCopy + "'; cat '" + draftPath + "'"}, Vendor: "openai", Model: "review-model"}}}
		raw, _ := json.Marshal(reviewers)
		writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", string(raw))
	})
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundleID, draftPath)
	if code, out := runPose(t, root, "review", "draft-check", bundleID, "--draft", draftPath); code != 0 || !strings.Contains(out, "would accept") {
		t.Fatalf("a draft the conclusion accepts was refused: %d %s", code, out)
	}

	// A criterion citing evidence the bundle did not seal, and a new finding
	// carrying a round prefix: both refused, by the same words conclude uses.
	good, _ := os.ReadFile(draftPath)
	bad := strings.Replace(string(good), "integration:val-alpha", "integration:val-missing", 1)
	badPath := filepath.Join(t.TempDir(), "bad.txt")
	if err := os.WriteFile(badPath, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, root, "review", "draft-check", bundleID, "--draft", badPath, "--json")
	var report struct {
		OK      bool     `json:"ok"`
		Reasons []string `json:"reasons"`
	}
	if code == 0 || json.Unmarshal([]byte(out), &report) != nil || report.OK || len(report.Reasons) == 0 || !strings.Contains(strings.Join(report.Reasons, " "), "the attestation preflight refuses it") {
		t.Fatalf("a draft the preflight refuses passed the draft check: %d %s", code, out)
	}

	// Inside a delegated review, the run provides the bundle and the round.
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 0 {
		t.Fatal(code, out)
	}
	if _, err := os.Stat(contextCopy); err != nil {
		t.Fatalf("the dispatch named no draft context: %v", err)
	}
	t.Setenv(posemodel.ReviewDraftContextEnv, contextCopy)
	if code, out := runPose(t, root, "review", "draft-check", "--draft", badPath); code == 0 || !strings.Contains(out, "would refuse") {
		t.Fatalf("the run's context did not judge the draft: %d %s", code, out)
	}
	if code, out := runPose(t, root, "review", "draft-check", "--draft", draftPath); code != 0 {
		t.Fatalf("the run's context refused a good draft: %d %s", code, out)
	}
	// From standard input, as a read-only sandbox can only offer it.
	t.Setenv(posemodel.ReviewDraftContextEnv, "")
	stdin, _ := os.Open(badPath)
	defer stdin.Close()
	saved := os.Stdin
	os.Stdin = stdin
	code, out = runPose(t, root, "review", "draft-check", bundleID, "--draft", "-")
	os.Stdin = saved
	if code == 0 || !strings.Contains(out, "would refuse") {
		t.Fatalf("a draft on standard input was not judged: %d %s", code, out)
	}
	_ = time.Now
}

// The same refusal at the conclusion step, so the check cannot drift from it.
func TestReviewDraftCheckRefusalIsTheConclusionRefusal(t *testing.T) {
	root, draftPath, bundleID := cliProvenanceInstance(t)
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundleID, draftPath)
	good, _ := os.ReadFile(draftPath)
	if err := os.WriteFile(draftPath, []byte(strings.Replace(string(good), "integration:val-alpha", "integration:val-missing", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, checked := runPose(t, root, "review", "draft-check", bundleID, "--draft", draftPath, "--json")
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ := s.ReadReviewRuns("")
	code, concluded := runPose(t, root, "review", "conclude", runs[0].RunID, "--issuer", "fixture-executor", "--apply")
	if code == 0 {
		t.Fatalf("conclude accepted the draft: %s", concluded)
	}
	var report struct{ Reasons []string }
	_ = json.Unmarshal([]byte(checked), &report)
	for _, r := range report.Reasons {
		for _, part := range strings.Split(strings.TrimPrefix(r, "the attestation preflight refuses it: "), "; ") {
			if !strings.Contains(concluded, part) {
				t.Fatalf("draft-check reported %q, which conclude did not: %s", part, concluded)
			}
		}
	}
}

// Spec pose-review-readiness-before-seal R5, R6: an oversized note is named
// with its size and the limit; a completed run nothing concluded names the
// command that concludes it.
func TestDelegateNamesTheNoteLimitAndTheConclusionStep(t *testing.T) {
	root, draftPath, bundleID := cliProvenanceInstance(t)
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundleID, draftPath)
	note := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(note, []byte(strings.Repeat("n", int(posemodel.ReviewNotesMaxBytes)+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake", "--note-file", note)
	if code == 0 || !strings.Contains(out, "implementer notes must be at most") || !strings.Contains(out, note) {
		t.Fatalf("an oversized note was not named: %d %s", code, out)
	}
	code, out = runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake")
	runs, _, _ := s.ReadReviewRuns("")
	if code != 0 || len(runs) != 1 || !strings.Contains(out, "pose review conclude "+runs[0].RunID+" --issuer <name> --apply") {
		t.Fatalf("the conclusion step was not named: %d %s", code, out)
	}
}
