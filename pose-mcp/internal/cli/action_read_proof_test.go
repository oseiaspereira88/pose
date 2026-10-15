package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-action-answers-reverify-on-read R5: exercise the public CLI,
// including JSON and Attention, rather than just its domain reader.
func TestActionCLIRejectsPersistedProoflessVerifiedAnswer(t *testing.T) {
	root := actionCLIFixture(t)
	_, raw := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	var opened struct {
		Request struct {
			ID string `json:"id"`
		} `json:"request"`
	}
	if err := json.Unmarshal([]byte(raw), &opened); err != nil || opened.Request.ID == "" {
		t.Fatalf("open: %v %s", err, raw)
	}
	id := opened.Request.ID
	// Resolve through the CLI using the unmodified declared contract.
	_, shown := runPose(t, root, "action", "show", id, "--json")
	var view map[string]any
	if err := json.Unmarshal([]byte(shown), &view); err != nil {
		t.Fatal(err)
	}
	request := view["request"].(map[string]any)
	code, out := runPose(t, root, "action", "resolve", id, "--actor", "human:maintainer", "--answer", "preserve-v1",
		"--request-digest", request["request_digest"].(string), "--expected-revision", "1", "--idempotency-key", "read-proof", "--apply")
	if code != 0 {
		t.Fatalf("resolve: %d %s", code, out)
	}
	if code, out := runPose(t, root, "action", "show", id, "--json"); code != 0 || !strings.Contains(out, `"assurance": "declared"`) {
		t.Fatalf("declared history: %d %s", code, out)
	}
	path := filepath.Join(root, ".pose/actions", id+".jsonl")
	journal, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(journal), `"assurance":"declared"`) {
		t.Fatalf("fixture has no declared answer: %s", journal)
	}
	journal = []byte(strings.ReplaceAll(string(journal), `"assurance":"declared"`, `"assurance":"verified"`))
	if err := os.WriteFile(path, journal, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"action", "show", id, "--json"}, {"action", "show", id}, {"action", "list", "--json"}} {
		code, out := runPose(t, root, args...)
		if code == 0 || !strings.Contains(out, "action-claim-not-verified") {
			t.Fatalf("%v consumed forged assurance: %d %s", args, code, out)
		}
	}
	_, attention := runPose(t, root, "state", "--attention", "--json")
	if !strings.Contains(attention, `"complete": false`) || !strings.Contains(attention, `"unavailable"`) {
		t.Fatalf("Attention hid incomplete action coverage: %s", attention)
	}
}
