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

func TestDelegateTaskDiagnosesRedactionGrowthWithoutKeepingTheCapture(t *testing.T) {
	root, _, prompt := taskInstance(t)
	stream := "session: X\n" + strings.Repeat("X", 80)
	agent := map[string]any{
		"vendor": "test", "model": "synthetic", "events": "plain",
		"commands":    map[string][]string{"read": {"sh", "-c", "cat >/dev/null; printf '%s' \"$1\"", "worker", stream}},
		"session":     map[string]string{"pattern": "session: (X)"},
		"containment": map[string]any{"schema_version": 1, "profile": "trusted-process", "stream_bytes": 128},
	}
	raw, err := json.Marshal(agent)
	if err != nil {
		t.Fatal(err)
	}
	writeDelegation(t, root, taskPolicy(`"default_access":"read"`, `"budget-profile":`+string(raw)))
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "budget-profile", "--record", "--json")
	if code == 0 {
		t.Fatalf("capture growth accepted: %s %s", out, errOut)
	}
	files, _ := filepath.Glob(filepath.Join(root, ".pose", "delegate-runs", "trn-*.json"))
	if len(files) != 1 {
		t.Fatalf("record: %v; %s %s", files, out, errOut)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Status, Failure, Output string
		Safety                  struct {
			CaptureComplete bool `json:"capture_complete"`
		}
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.Status != "failed" || record.Output != "" || record.Safety.CaptureComplete {
		t.Fatalf("unsafe capture promoted: %s", data)
	}
	for _, want := range []string{"budget-profile", `events="plain"`, `stage="stdout"`, "redacted capture exceeds byte budget"} {
		if !strings.Contains(record.Failure, want) {
			t.Errorf("failure lacks %q: %s", want, record.Failure)
		}
	}
	base := strings.TrimSuffix(files[0], ".json")
	for _, suffix := range []string{".transcript", ".stderr"} {
		retained, err := os.ReadFile(base + suffix)
		if err != nil || len(retained) != 0 {
			t.Fatalf("unsafe %s retained: %v", suffix, err)
		}
	}
	if _, err := os.Stat(base + ".events.ndjson"); !os.IsNotExist(err) {
		t.Fatalf("unsafe events retained: %v", err)
	}
}
