package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

type recoveryCLIFileFault struct {
	pose.Files
	path  string
	fired bool
}

func (f *recoveryCLIFileFault) CreateExclusive(name string, b []byte, p os.FileMode, d bool) error {
	if err := f.Files.CreateExclusive(name, b, p, d); err != nil {
		return err
	}
	if name == f.path && !f.fired {
		f.fired = true
		return errors.New("synthetic successor Sync failure")
	}
	return nil
}
func TestActionRecoveryCLIPreviewRefusesWithoutSigningOrAuditing(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatal(out)
	}
	var v pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".pose/actions", v.Request.ID+".jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"action", "resolve", v.Request.ID, "--actor", "agent:intruder", "--answer", "preserve-v1", "--request-digest", v.Request.RequestDigest, "--expected-revision", "1", "--idempotency-key", "preview", "--json"}
	code, out = runPose(t, root, args...)
	if code == 0 || !strings.Contains(out, "action-actor-not-authorized") || !strings.Contains(out, `"audit_persisted": false`) {
		t.Fatalf("permissive preview: %d %s", code, out)
	}
	bin := t.TempDir()
	marker := filepath.Join(bin, "touched")
	mustWrite(t, filepath.Join(bin, "ssh-keygen"), "#!/bin/sh\nprintf touched > '"+marker+"'\nexit 1\n")
	if err := os.Chmod(filepath.Join(bin, "ssh-keygen"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	code, out = runPose(t, root, append(args, "--apply", "--sign", "nonexistent-private-key")...)
	if code == 0 || !strings.Contains(out, "action-actor-not-authorized") {
		t.Fatalf("invalid apply preflight: %d %s", code, out)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("invalid proposal invoked signer")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("refusal changed action journal")
	}
	// Valid preview under verified policy is still pending proof, never accepted.
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:maintainer"]}}`)
	args[4] = "human:maintainer"
	code, out = runPose(t, root, args...)
	if code != 0 || !strings.Contains(out, `"proof_pending": true`) || !strings.Contains(out, "not-certified") {
		t.Fatalf("proof was not pending: %d %s", code, out)
	}
}
func TestActionRecoveryCLIResumesSamePreparedSuccessor(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatal(out)
	}
	var first pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatal(err)
	}
	args := append([]string{}, openArgs...)
	for i := range args {
		if args[i] == "Keep reading schema v1?" {
			args[i] = "A materially changed question"
		}
	}
	args = append(args, "--supersedes", first.Request.ID, "--requested-at", "2026-10-24T12:00:00Z", "--json")
	code, out = runPose(t, root, args...)
	if code != 0 {
		t.Fatal(out)
	}
	var prepared pose.ActionRequest
	if err := json.Unmarshal([]byte(out), &prepared); err != nil {
		t.Fatal(err)
	}
	fault := &recoveryCLIFileFault{Files: pose.NativeFiles(), path: filepath.Join(root, ".pose/actions", prepared.ID+".jsonl")}
	if _, err := (pose.Store{Root: root, FS: fault}).OpenActionRequest(prepared, time.Now()); !errors.Is(err, pose.ErrActionRecoveryPending) {
		t.Fatalf("fault not reached: %v", err)
	}
	for n := 0; n < 2; n++ {
		code, out = runPose(t, root, append(args, "--apply")...)
		if code != 0 || !strings.Contains(out, `"phase": "complete"`) {
			t.Fatalf("CLI retry %d: %d %s", n, code, out)
		}
	}
	old, err := (pose.Store{Root: root}).LoadActionRequest(first.Request.ID)
	if err != nil || old.Revision != 2 || old.State != pose.ActionStateSuperseded {
		t.Fatalf("retirement duplicated: %+v %v", old, err)
	}
}

func TestActionRecoveryCLIStatementRefusesChangedSubject(t *testing.T) {
	root := actionCLIFixture(t)
	args := append(append([]string{}, openArgs...), "--subject", "requirement:R4", "--apply", "--json")
	code, out := runPose(t, root, args...)
	if code != 0 {
		t.Fatal(out)
	}
	var v pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-storage.md"), "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R4: New material.\n")
	code, out = runPose(t, root, "action", "statement", v.Request.ID, "--actor", "human:maintainer", "--answer", "preserve-v1", "--idempotency-key", "stale")
	if code == 0 || !strings.Contains(out, "action-request-not-open") {
		t.Fatalf("old material statement emitted: %d %s", code, out)
	}
}
