package cli

// Harne8 control-plane composition (spec pose-harne8-control-plane-integration):
// reconciles a Harness execution result into local evidence, identity-bound
// to the Execution Identity RunID that authorized the original submission
// (R2). Never mutates or overwrites a prior reconciliation for the same
// request — a second record for the same request_id is rejected unless
// --allow-supersede is passed, and even then a fresh, append-only record
// is written that explicitly references what it supersedes; nothing is
// ever edited or deleted in place. POSE governs (owns this evidence
// contract); Harness executes and reports back through it; Conductor is
// the durable run-state owner this local record composes with, never
// replaces (Constraint: POSE governs, Conductor orchestrates, Harness
// executes).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type harnessEvidence struct {
	RecordedAt           string `json:"recorded_at"`
	RunID                string `json:"run_id"`
	RequestID            string `json:"request_id"`
	ExecutionID          string `json:"execution_id"`
	PlanDigest           string `json:"plan_digest"`
	Status               string `json:"status"` // success | failure
	ResultDigest         string `json:"result_digest,omitempty"`
	Source               string `json:"source"` // harness | manual
	SupersedesRecordedAt string `json:"supersedes_recorded_at,omitempty"`
}

var validEvidenceStatus = map[string]bool{"success": true, "failure": true}
var validEvidenceSource = map[string]bool{"harness": true, "manual": true}

func harnessEvidenceDir(root string) string {
	return filepath.Join(root, ".pose", "reports", "history")
}

func harnessEvidencePath(root string, t time.Time) string {
	return filepath.Join(harnessEvidenceDir(root), "harness-evidence-"+t.UTC().Format("2006-01")+".jsonl")
}

func readHarnessEvidence(root string) []harnessEvidence {
	entries, err := os.ReadDir(harnessEvidenceDir(root))
	if err != nil {
		return nil
	}
	var out []harnessEvidence
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "harness-evidence-") || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(harnessEvidenceDir(root), e.Name()))
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			if strings.TrimSpace(s.Text()) == "" {
				continue
			}
			var ev harnessEvidence
			if json.Unmarshal([]byte(s.Text()), &ev) == nil {
				out = append(out, ev)
			}
		}
		_ = f.Close()
	}
	return out
}

// latestEvidenceFor returns the most recently recorded evidence for
// request_id, if any.
func latestEvidenceFor(root, requestID string) (harnessEvidence, bool) {
	var latest harnessEvidence
	found := false
	for _, ev := range readHarnessEvidence(root) {
		if ev.RequestID != requestID {
			continue
		}
		if !found || ev.RecordedAt > latest.RecordedAt {
			latest, found = ev, true
		}
	}
	return latest, found
}

func cmdReconcileEvidence(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose reconcile-evidence <record|list|housekeeping> ...")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "record":
		return cmdReconcileEvidenceRecord(root, args, stdout, stderr)
	case "list":
		return cmdReconcileEvidenceList(root, args, stdout, stderr)
	case "housekeeping":
		return cmdReconcileEvidenceHousekeeping(root, args, stdout, stderr)
	default:
		return usageError(stderr, "pose reconcile-evidence: unknown subcommand "+sub)
	}
}

func cmdReconcileEvidenceRecord(root string, args []string, stdout, stderr io.Writer) int {
	var ev harnessEvidence
	allowSupersede := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--allow-supersede" {
			allowSupersede = true
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, "Usage: pose reconcile-evidence record --run-id ID --request-id ID --execution-id ID --plan-digest SHA --status success|failure --source harness|manual [--result-digest SHA] [--allow-supersede]")
		}
		v := args[i+1]
		switch args[i] {
		case "--run-id":
			ev.RunID = v
		case "--request-id":
			ev.RequestID = v
		case "--execution-id":
			ev.ExecutionID = v
		case "--plan-digest":
			ev.PlanDigest = v
		case "--status":
			ev.Status = v
		case "--result-digest":
			ev.ResultDigest = v
		case "--source":
			ev.Source = v
		default:
			return usageError(stderr, "pose reconcile-evidence record: unknown flag "+args[i])
		}
		i++
	}
	if ev.RunID == "" || ev.RequestID == "" || ev.ExecutionID == "" || ev.PlanDigest == "" {
		fmt.Fprintln(stderr, "pose reconcile-evidence record: --run-id, --request-id, --execution-id and --plan-digest are required (identity-bound evidence, R2)")
		return 2
	}
	if !validEvidenceStatus[ev.Status] {
		fmt.Fprintln(stderr, "pose reconcile-evidence record: --status must be success|failure")
		return 2
	}
	if !validEvidenceSource[ev.Source] {
		fmt.Fprintln(stderr, "pose reconcile-evidence record: --source must be harness|manual")
		return 2
	}

	if prior, exists := latestEvidenceFor(root, ev.RequestID); exists {
		if !allowSupersede {
			fmt.Fprintf(stderr, "pose reconcile-evidence record: evidence for request_id %q already exists (recorded_at=%s) — pass --allow-supersede to add a newer record; the prior one is never edited or removed\n", ev.RequestID, prior.RecordedAt)
			return 1
		}
		ev.SupersedesRecordedAt = prior.RecordedAt
	}

	ev.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	line, _ := json.Marshal(ev)
	if err := appendEvent(harnessEvidencePath(root, time.Now().UTC()), line); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "evidence recorded: request_id=%s run_id=%s status=%s\n", ev.RequestID, ev.RunID, ev.Status)
	return 0
}

func cmdReconcileEvidenceList(root string, args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	requestID := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--request-id":
			if i+1 >= len(args) {
				return usageError(stderr, "pose reconcile-evidence list: --request-id requires a value")
			}
			i++
			requestID = args[i]
		default:
			return usageError(stderr, "Usage: pose reconcile-evidence list [--request-id ID] [--json]")
		}
	}
	var records []harnessEvidence
	for _, ev := range readHarnessEvidence(root) {
		if requestID != "" && ev.RequestID != requestID {
			continue
		}
		records = append(records, ev)
	}
	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(records)
		return 0
	}
	for _, ev := range records {
		superseded := ""
		if ev.SupersedesRecordedAt != "" {
			superseded = " supersedes:" + ev.SupersedesRecordedAt
		}
		fmt.Fprintf(stdout, "%s request_id=%s run_id=%s status=%s source=%s%s\n",
			ev.RecordedAt, ev.RequestID, ev.RunID, ev.Status, ev.Source, superseded)
	}
	return 0
}

func cmdReconcileEvidenceHousekeeping(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose reconcile-evidence housekeeping <list-expired|purge> [--older-than-days N] [--apply]")
	}
	mode := args[0]
	args = args[1:]
	days := 400
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--older-than-days":
			if i+1 >= len(args) {
				return 2
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return 2
			}
			days = n
		case "--apply":
			apply = true
		default:
			return usageError(stderr, "pose reconcile-evidence housekeeping: invalid argument")
		}
	}
	if mode != "list-expired" && mode != "purge" {
		return usageError(stderr, "pose reconcile-evidence housekeeping: invalid command")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	entries, _ := os.ReadDir(harnessEvidenceDir(root))
	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "harness-evidence-") || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		month, err := time.Parse("2006-01", strings.TrimSuffix(strings.TrimPrefix(e.Name(), "harness-evidence-"), ".jsonl"))
		if err != nil || !month.Before(cutoff) {
			continue
		}
		path := filepath.Join(harnessEvidenceDir(root), e.Name())
		if mode == "list-expired" {
			fmt.Fprintf(stdout, "%s\n", path)
			continue
		}
		if apply {
			if err := os.Remove(path); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			removed++
		} else {
			fmt.Fprintf(stdout, "[DRY-RUN] would remove: %s\n", path)
		}
	}
	if mode == "purge" {
		if apply {
			fmt.Fprintf(stdout, "Result: SUCCESS — %d evidence file(s) removed.\n", removed)
		} else {
			fmt.Fprintln(stdout, "Result: DRY-RUN — re-run with --apply to remove.")
		}
	}
	return 0
}
