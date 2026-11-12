package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure(t *testing.T) {
	const session = "synthetic-private-session-001"
	stream := `{"type":"system","subtype":"init","session_id":"` + session + `"}` + "\n" +
		`{"type":"result","session_id":"` + session + `","result":"task finished"}` + "\n"
	for _, tc := range []struct {
		name, pattern string
		ok            bool
	}{
		{"legacy", `"session_id"\s*:\s*"([^"]+)"`, false},
		{"init-only", `(?m)^\{"type"\s*:\s*"system"\s*,\s*"subtype"\s*:\s*"init"[^\n]*"session_id"\s*:\s*"([^"\r\n]+)"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, prompt := taskInstance(t)
			agent := map[string]any{"vendor": "test", "model": "synthetic", "events": "claude-stream-json", "commands": map[string][]string{"read": {"sh", "-c", "cat >/dev/null; printf '%s' \"$1\"", "worker", stream}}, "session": map[string]string{"pattern": tc.pattern}}
			raw, err := json.Marshal(agent)
			if err != nil {
				t.Fatal(err)
			}
			writeDelegation(t, root, taskPolicy(`"default_access":"read"`, `"legacy-claude":`+string(raw)))
			code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "legacy-claude", "--record", "--json")
			files, _ := filepath.Glob(filepath.Join(root, ".pose", "delegate-runs", "trn-*.json"))
			if len(files) != 1 {
				t.Fatalf("record: %v; %d %s %s", files, code, out, errOut)
			}
			var record struct{ Status, Failure, Output string }
			data, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &record); err != nil {
				t.Fatal(err)
			}
			if tc.ok {
				if code != 0 || record.Status != "completed" || record.Output != "task finished" {
					t.Fatalf("init-only: %d %s %s", code, out, errOut)
				}
			} else {
				if code == 0 || record.Status != "failed" || record.Output != "" {
					t.Fatalf("ambiguous capture accepted: %d %s", code, out)
				}
				for _, want := range []string{"legacy-claude", "claude-stream-json", "session/interrupt recognition", "ambiguous adapter capture", "system/init"} {
					if !strings.Contains(record.Failure, want) {
						t.Errorf("failure lacks %q: %s", want, record.Failure)
					}
				}
				for _, suffix := range []string{".transcript", ".stderr"} {
					retained, err := os.ReadFile(strings.TrimSuffix(files[0], ".json") + suffix)
					if err != nil || len(retained) != 0 {
						t.Fatalf("unsafe capture retained: %v", err)
					}
				}
			}
			if strings.Contains(string(data)+out+errOut, session) {
				t.Fatal("private session leaked")
			}
		})
	}
}
