package cli

// Harne8 control-plane composition (spec pose-harne8-control-plane-integration):
// Harness results reconcile into evidence identity-bound to the
// Execution Identity RunID, and a prior reconciliation for the same
// request is never silently overwritten (R2) — superseding requires an
// explicit flag and always appends a new record referencing the old one,
// never editing or deleting it. Evidence storage is per-project (tenant
// isolation) and supports retention housekeeping.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recordEvidenceArgs(runID, requestID, execID, digest, status, source string) []string {
	return []string{
		"record",
		"--run-id", runID, "--request-id", requestID, "--execution-id", execID,
		"--plan-digest", digest, "--status", status, "--source", source,
	}
}

func TestReconcileEvidenceValidation(t *testing.T) {
	root := newGitRepo(t)
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"missing-required", []string{"record", "--status", "success", "--source", "harness"}, 2},
		{"invalid-status", []string{"record", "--run-id", "r1", "--request-id", "req1", "--execution-id", "e1", "--plan-digest", "d1", "--status", "maybe", "--source", "harness"}, 2},
		{"invalid-source", []string{"record", "--run-id", "r1", "--request-id", "req1", "--execution-id", "e1", "--plan-digest", "d1", "--status", "success", "--source", "vibes"}, 2},
		{"valid", recordEvidenceArgs("r1", "req1", "e1", "d1", "success", "harness"), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errB bytes.Buffer
			if code := cmdReconcileEvidence(root, c.args, &out, &errB); code != c.want {
				t.Fatalf("exit=%d want=%d out=%s err=%s", code, c.want, out.String(), errB.String())
			}
		})
	}
}

func TestReconcileEvidenceRejectsSilentMutation(t *testing.T) {
	root := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdReconcileEvidence(root, recordEvidenceArgs("r1", "req1", "e1", "d1", "success", "harness"), &out, &errB); code != 0 {
		t.Fatalf("first record exit=%d err=%s", code, errB.String())
	}

	out.Reset()
	errB.Reset()
	if code := cmdReconcileEvidence(root, recordEvidenceArgs("r1", "req1", "e2", "d1", "failure", "harness"), &out, &errB); code == 0 {
		t.Fatal("a second record for the same request_id without --allow-supersede must be rejected")
	} else if !strings.Contains(errB.String(), "already exists") {
		t.Errorf("expected an already-exists diagnostic: %s", errB.String())
	}

	out.Reset()
	errB.Reset()
	args := append(recordEvidenceArgs("r1", "req1", "e2", "d1", "failure", "harness"), "--allow-supersede")
	if code := cmdReconcileEvidence(root, args, &out, &errB); code != 0 {
		t.Fatalf("supersede exit=%d err=%s", code, errB.String())
	}

	out.Reset()
	errB.Reset()
	if code := cmdReconcileEvidence(root, []string{"list", "--request-id", "req1", "--json"}, &out, &errB); code != 0 {
		t.Fatalf("list exit=%d err=%s", code, errB.String())
	}
	var records []harnessEvidence
	if err := json.Unmarshal(out.Bytes(), &records); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if len(records) != 2 {
		t.Fatalf("expected both records preserved (append-only), got %d: %+v", len(records), records)
	}
	var superseding *harnessEvidence
	for i := range records {
		if records[i].SupersedesRecordedAt != "" {
			superseding = &records[i]
		}
	}
	if superseding == nil {
		t.Fatal("expected one record to explicitly reference what it supersedes")
	}
	if superseding.Status != "failure" || superseding.ExecutionID != "e2" {
		t.Errorf("unexpected superseding record: %+v", superseding)
	}
}

func TestReconcileEvidenceIsIdentityBound(t *testing.T) {
	root := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdReconcileEvidence(root, recordEvidenceArgs("run-abc123", "req1", "e1", "d1", "success", "harness"), &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	records := readHarnessEvidence(root)
	if len(records) != 1 || records[0].RunID != "run-abc123" {
		t.Fatalf("expected evidence bound to the submitting RunID: %+v", records)
	}
}

func TestReconcileEvidenceHousekeeping(t *testing.T) {
	root := newGitRepo(t)
	oldFile := filepath.Join(harnessEvidenceDir(root), "harness-evidence-2020-01.jsonl")
	recentFile := filepath.Join(harnessEvidenceDir(root), "harness-evidence-"+time.Now().UTC().Format("2006-01")+".jsonl")
	for _, p := range []string{oldFile, recentFile} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(`{"request_id":"x"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var out, errB bytes.Buffer
	if code := cmdReconcileEvidence(root, []string{"housekeeping", "list-expired", "--older-than-days", "400"}, &out, &errB); code != 0 {
		t.Fatalf("list-expired exit=%d err=%s", code, errB.String())
	}
	if !strings.Contains(out.String(), "2020-01") {
		t.Errorf("expected the old file listed: %s", out.String())
	}

	out.Reset()
	if code := cmdReconcileEvidence(root, []string{"housekeeping", "purge", "--older-than-days", "400", "--apply"}, &out, &errB); code != 0 {
		t.Fatalf("purge exit=%d err=%s", code, errB.String())
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("expected the expired evidence file to be removed")
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Error("recent evidence file must be preserved")
	}
}

func TestReconcileEvidenceTenantIsolation(t *testing.T) {
	rootA := newGitRepo(t)
	rootB := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdReconcileEvidence(rootA, recordEvidenceArgs("r1", "req-a", "e1", "d1", "success", "harness"), &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	if got := readHarnessEvidence(rootB); len(got) != 0 {
		t.Errorf("evidence recorded in project A leaked into project B: %+v", got)
	}
	if got := readHarnessEvidence(rootA); len(got) != 1 {
		t.Errorf("expected exactly 1 record in project A, got %+v", got)
	}
}

func TestReconcileEvidenceCLIEndToEnd(t *testing.T) {
	root := newGitRepo(t)
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main(append([]string{"reconcile-evidence"}, recordEvidenceArgs("r1", "req1", "e1", "d1", "success", "harness")...), &out, &errB); code != 0 {
			t.Fatalf("exit=%d err=%s", code, errB.String())
		}
		out.Reset()
		if code := Main([]string{"reconcile-evidence", "list"}, &out, &errB); code != 0 {
			t.Fatalf("list exit=%d err=%s", code, errB.String())
		}
		if !strings.Contains(out.String(), "req1") {
			t.Errorf("expected the recorded evidence to be listed: %s", out.String())
		}
	})
}
