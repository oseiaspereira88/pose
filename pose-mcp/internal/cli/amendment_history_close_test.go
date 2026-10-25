package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func amendmentHistoryStart(t *testing.T, root, slug, body string) {
	t.Helper()
	record := pose.StartRecord{SchemaVersion: pose.StartSchemaVersion, Spec: slug, Phase: "started", Baseline: pose.ProjectContractNodes(slug, body), RecordedAt: "2026-10-10T00:00:00Z"}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose", "starts", slug+".json"), string(raw))
}

func TestAmendmentHistoryFirstChangeSeedsRecordedStart(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		t.Run(map[bool]string{false: "requirements", true: "nodes"}[adopted], func(t *testing.T) {
			root := contractNodesCLIFixture(t, adopted)
			amendmentHistoryStart(t, root, "nodes", contractNodesCLISpec)
			specPath := filepath.Join(root, ".pose/specs/nodes/spec.md")
			mustWrite(t, specPath, strings.Replace(contractNodesCLISpec, "Keep legacy logs readable.", "Keep verified history readable.", 1))
			if code, out := runAmend(t, root, "nodes", "--ids", "R1", "--change", "semantic", "--rationale", "correct contract", "--author", "@core"); code != 0 {
				t.Fatalf("amend %d: %s", code, out)
			}
			events, err := pose.LoadAmendments(pose.AmendmentsPath(specPath))
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 2 || events[0].Change != "baseline" || events[0].Hashes["R1"] != pose.CurrentRequirementHashes(contractNodesCLISpec)["R1"] {
				t.Fatalf("missing recorded baseline: %+v", events)
			}
			raw, _ := os.ReadFile(specPath)
			if findings := pose.UnacknowledgedNodeChanges("nodes", string(raw), events, adopted); len(findings) > 0 {
				t.Fatalf("history: %v", findings)
			}
		})
	}
}

func TestAmendmentHistoryInProgressWarnsBeforeDone(t *testing.T) {
	root, path := amendFixture(t)
	if code, out := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatal(out)
	}
	body := strings.Replace(amendSpec, "status: done", "status: in-progress", 1)
	body = strings.Replace(body, "keep behaving.", "new behavior.", 1)
	mustWrite(t, path, body)
	var out, errOut bytes.Buffer
	if code := lintOneSpec(path, false, false, &out, &errOut); code != 0 {
		t.Fatalf("warning became fatal %d: %s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "R2 changed after its last acknowledged amendment") {
		t.Fatalf("no warning: %s%s", out.String(), errOut.String())
	}
}

func TestAmendmentHistoryCloseRefusesBeforeTransition(t *testing.T) {
	root := closeoutCLIFixture(t)
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	if code, out := runAmend(t, root, "alpha", "--baseline", "--author", "@core"); code != 0 {
		t.Fatal(out)
	}
	raw, _ := os.ReadFile(path)
	mustWrite(t, path, strings.Replace(string(raw), "R1: works", "R1: changed", 1))
	var out, errOut bytes.Buffer
	if code := cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut); code != 0 {
		t.Fatal(errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code == 0 || !strings.Contains(out.String()+errOut.String(), "amendment history") {
		t.Fatalf("invalid history closed: %d %s%s", code, out.String(), errOut.String())
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, []byte(strings.Replace(string(raw), "R1: works", "R1: changed", 1))) {
		t.Fatal("refusal changed lifecycle")
	}
}

func TestAmendmentHistoryInvalidInputAndStartNeverSeedJournal(t *testing.T) {
	for _, tc := range []string{"invalid-option", "invalid-transition", "corrupt-start", "incomplete-start"} {
		t.Run(tc, func(t *testing.T) {
			root := contractNodesCLIFixture(t, true)
			amendmentHistoryStart(t, root, "nodes", contractNodesCLISpec)
			startPath := filepath.Join(root, ".pose/starts/nodes.json")
			if tc == "corrupt-start" {
				mustWrite(t, startPath, "{")
			}
			if tc == "incomplete-start" {
				raw, _ := os.ReadFile(startPath)
				mustWrite(t, startPath, strings.Replace(string(raw), `"started"`, `"baseline-recorded"`, 1))
			}
			id, change := "R1", "semantic"
			if tc == "invalid-option" {
				id = "R99"
			}
			if tc == "invalid-transition" {
				id = "A1"
				change = "transition"
			}
			if code, out := runAmend(t, root, "nodes", "--ids", id, "--change", change, "--rationale", "synthetic change", "--author", "@core"); code == 0 {
				t.Fatalf("invalid accepted %s", out)
			}
			if _, err := os.Stat(filepath.Join(root, ".pose/specs/nodes/amendments.jsonl")); !os.IsNotExist(err) {
				t.Fatalf("invalid attempt wrote journal %v", err)
			}
		})
	}
}

func TestAmendmentHistoryLegacyAndExistingJournalStayAppendOnly(t *testing.T) {
	root, path := amendFixture(t)
	if code, out := runAmend(t, root, "amended", "--ids", "R1", "--change", "semantic", "--rationale", "legacy ack", "--author", "@core"); code != 0 {
		t.Fatal(out)
	}
	log := pose.AmendmentsPath(path)
	before, _ := os.ReadFile(log)
	events, _ := pose.LoadAmendments(log)
	if len(events) != 1 {
		t.Fatal("legacy was automatically seeded")
	}
	amendmentHistoryStart(t, root, "amended", amendSpec)
	if code, out := runAmend(t, root, "amended", "--ids", "R2", "--change", "semantic", "--rationale", "explicit ack", "--author", "@core"); code != 0 {
		t.Fatal(out)
	}
	after, _ := os.ReadFile(log)
	if !bytes.HasPrefix(after, before) {
		t.Fatal("old event rewritten")
	}
	events, _ = pose.LoadAmendments(log)
	if len(events) != 2 || events[1].Change == "baseline" {
		t.Fatal("existing journal silently repaired")
	}
}

func TestAmendmentHistoryFlatAddedAndTransitionKeepEarlierNodes(t *testing.T) {
	for _, transition := range []bool{false, true} {
		t.Run(map[bool]string{false: "added", true: "transition"}[transition], func(t *testing.T) {
			root := contractNodesCLIFixture(t, true)
			folder := filepath.Join(root, ".pose/specs/nodes/spec.md")
			flat := filepath.Join(root, ".pose/specs/2026-10-10-nodes.md")
			if err := os.Rename(folder, flat); err != nil {
				t.Fatal(err)
			}
			amendmentHistoryStart(t, root, "nodes", contractNodesCLISpec)
			id, change := "R2", "added"
			body := strings.Replace(contractNodesCLISpec, "- R1: Keep legacy logs readable.", "- R1: Keep legacy logs readable.\n- R2: additional behavior.", 1)
			if transition {
				id = "A1"
				change = "transition"
				body = strings.Replace(contractNodesCLISpec, "- Status: unverified", "- Status: verified\n- Evidence: test:TestAmendmentHistoryFlatAddedAndTransitionKeepEarlierNodes\n- Scope: this spec", 1)
			}
			mustWrite(t, flat, body)
			if code, out := runAmend(t, root, "nodes", "--ids", id, "--change", change, "--rationale", "explicit change", "--author", "@core"); code != 0 {
				t.Fatal(out)
			}
			events, err := pose.LoadAmendments(pose.AmendmentsPath(flat))
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 2 {
				t.Fatalf("events %d", len(events))
			}
			if findings := pose.UnacknowledgedNodeChanges("nodes", body, events, true); len(findings) > 0 {
				t.Fatal(findings)
			}
		})
	}
}

func TestAmendmentHistoryCloseRechecksUnderLockAndValidControl(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "changed-after-gates"}[corrupt], func(t *testing.T) {
			root := closeoutCLIFixture(t)
			path := filepath.Join(root, ".pose/specs/alpha/spec.md")
			if code, out := runAmend(t, root, "alpha", "--baseline", "--author", "@core"); code != 0 {
				t.Fatal(out)
			}
			var out, errOut bytes.Buffer
			if code := cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut); code != 0 {
				t.Fatal(errOut.String())
			}
			if corrupt {
				closeGatesJudged = func() { mustWrite(t, pose.AmendmentsPath(path), "{broken\n") }
				t.Cleanup(func() { closeGatesJudged = nil })
			}
			out.Reset()
			errOut.Reset()
			code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut)
			if corrupt && (code == 0 || !strings.Contains(out.String()+errOut.String(), "amendment history")) {
				t.Fatalf("changed history accepted %d %s%s", code, out.String(), errOut.String())
			}
			if !corrupt && code != 0 {
				t.Fatalf("valid refused %s%s", out.String(), errOut.String())
			}
			raw, _ := os.ReadFile(path)
			want := "status: done"
			if corrupt {
				want = "status: in-progress"
			}
			if !strings.Contains(string(raw), want) {
				t.Fatalf("wrong lifecycle %s", raw)
			}
		})
	}
}
