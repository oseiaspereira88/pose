package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// nextReviewRound changes the scope, commits it and seals the next bundle, as
// an implementer does after a rejection.
func nextReviewRound(t *testing.T, root, change string) {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	lib := filepath.Join(root, "pose-mcp", "lib.go")
	raw, _ := os.ReadFile(lib)
	if err := os.WriteFile(lib, append(raw, []byte("// "+change+"\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", change)
	head := git("rev-parse", "HEAD")
	graphPath := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	var graph posemodel.DeliveryIntegrityGraph
	rawGraph, _ := os.ReadFile(graphPath)
	if err := json.Unmarshal(rawGraph, &graph); err != nil {
		t.Fatal(err)
	}
	cs := &graph.ChangeSets[0]
	cs.Selector, cs.Head, cs.ResolvedHead = "range:"+cs.Base+".."+head, head, head
	graph.ValidationResults[0].GitHead = head
	rawGraph, _ = json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(rawGraph))
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatalf("seal: %s", out)
	}
}

// roundsAgent answers round 1 with two findings and, from round 2, accounts
// for them; it marks whether it ran its resume command, so a continued
// thread shows. sloppy omits a prior finding.
func roundsAgent(name, vendor string, sloppy bool) string {
	prior := `{"id":"R1-B1","status":"fixed","evidence":"lib.go:2"},{"id":"R1-B2","status":"partial","evidence":"half done"}`
	if sloppy {
		prior = `{"id":"R1-B1","status":"fixed","evidence":"lib.go:2"}`
	}
	// The drafts sit in single-quoted shell words and are printed with %s:
	// no printf escape, so every sh prints them alike.
	second := `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[` + prior + `],"sweep":"the whole change"}`
	first := `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[{"id":"B1","severity":"high","disposition":"blocker","file":"pose-mcp/lib.go","line":1,"summary":"first defect"},{"id":"B2","severity":"medium","disposition":"blocker","summary":"second defect"}]}`
	fence := "```"
	script := `in=$(cat); echo "session: rounds-sess-01"; [ -n "$1" ] && echo RESUMED-$1; ` +
		`if printf '%s' "$in" | grep -q 'This is round 2'; then printf '%s\n' 'decision: approved' '` + fence + `pose-draft' '` + second + `' '` + fence + `'; ` +
		`else printf '%s\n' 'decision: changes-requested' '` + fence + `pose-draft' '` + first + `' '` + fence + `'; fi`
	argv, _ := json.Marshal([]string{"sh", "-c", script, "x"})
	resume, _ := json.Marshal([]string{"sh", "-c", script, "x", "{session}"})
	return `"` + name + `": {"vendor": "` + vendor + `", "model": "` + name + `-1", "timeout": "30s",
      "commands": {"read": ` + string(argv) + `},
      "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ` + string(resume) + `}}`
}

func roundRun(t *testing.T, root, agent string) map[string]any {
	t.Helper()
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", agent, "--json", "--reason", "next round")
	var doc map[string]any
	if json.Unmarshal([]byte(out), &doc) != nil || doc["status"] != "completed" {
		t.Fatalf("round run: %d %s %s", code, out, errOut)
	}
	return doc
}

func runRecord(t *testing.T, root, id string) posemodel.ReviewRun {
	t.Helper()
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	for _, r := range runs {
		if r.RunID == id {
			return r
		}
	}
	t.Fatalf("no run %s", id)
	return posemodel.ReviewRun{}
}

// R1–R7, R9: round 2 is briefed with round 1's findings, continues the
// thread's session, and its draft accounts for each; verify discloses the
// round and the lineage; insights measure convergence.
func TestReviewRoundsContinueTheThreadAndAccountForFindings(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(roundsAgent("rounder", "other", false)))
	first := roundRun(t, root, "rounder")
	if r := runRecord(t, root, first["run"].(string)); r.Round != 1 || r.Thread != nil || len(r.DraftProblems) != 0 {
		t.Fatalf("round 1: %+v", r)
	}
	nextReviewRound(t, root, "fix B1")
	brief, err := posemodel.Store{Root: root}.RenderReviewBrief("spec:alpha", "review", "")
	if err != nil || !strings.Contains(brief.Text, "This is round 2") || !strings.Contains(brief.Text, "> R1-B1 (high, blocker, round 1) at pose-mcp/lib.go:1 — first defect") {
		t.Fatalf("round 2 brief: %v\n%s", err, brief.Text)
	}
	second := roundRun(t, root, "rounder")
	r2 := runRecord(t, root, second["run"].(string))
	if r2.Round != 2 || !strings.Contains(r2.Output, "RESUMED-") || r2.Thread == nil || r2.Thread.Mode != posemodel.ReviewThreadContinued || r2.Thread.From != first["run"] || len(r2.DraftProblems) != 0 {
		t.Fatalf("round 2: round=%d thread=%+v problems=%v output=%q", r2.Round, r2.Thread, r2.DraftProblems, r2.Output)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "wait", second["run"].(string), "--json"); code != 0 || !strings.Contains(out, `"decision":"approved"`) {
		t.Fatalf("wait: %d %s", code, out)
	}
	_, out := runPose(t, root, "review", "verify", "spec:alpha")
	if !strings.Contains(out, "review_verify.round=2") || !strings.Contains(out, "review_verify.thread."+second["run"].(string)+"=round 2: continued session from "+first["run"].(string)) {
		t.Fatalf("verify: %s", out)
	}
	code, out := runPose(t, root, "insights", "--json")
	if code != 0 || !strings.Contains(out, `"rounds"`) || !strings.Contains(out, `"scope":"spec:alpha"`) || !strings.Contains(out, `"rounds_to_approval":2`) || !strings.Contains(out, `"continued":{"rounds":1,"invalid_rounds":0,"findings":0,"approvals":1,"accounted":2,"fixed":1,"partial":1`) {
		t.Fatalf("insights: %d %s", code, out)
	}
}

// R4, R6, R9: a draft that omits a prior finding is invalid and never an
// approval, continued session or not; a vendor change opens a new session
// with its reason; cold-session opens one every round.
func TestReviewRoundsRefuseAnIncompleteAccount(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(roundsAgent("rounder", "other", false), roundsAgent("sloppy", "other", true), roundsAgent("elsewhere", "third", false)))
	first := roundRun(t, root, "rounder")
	nextReviewRound(t, root, "fix B1")
	second := roundRun(t, root, "sloppy")
	r2 := runRecord(t, root, second["run"].(string))
	if len(r2.DraftProblems) == 0 || !strings.Contains(strings.Join(r2.DraftProblems, ";"), "R1-B2") {
		t.Fatalf("an incomplete account passed: round=%d thread=%+v problems=%v output=%q", r2.Round, r2.Thread, r2.DraftProblems, r2.Output)
	}
	if r2.Thread == nil || r2.Thread.Mode != posemodel.ReviewThreadNew || !strings.Contains(r2.Thread.Reason, "agent changed") {
		t.Fatalf("another agent continued the session: %+v", r2.Thread)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "wait", second["run"].(string), "--json"); code == 0 || !strings.Contains(out, `"decision":"invalid"`) {
		t.Fatalf("an invalid draft decided: %d %s", code, out)
	}
	third := roundRun(t, root, "elsewhere")
	if r3 := runRecord(t, root, third["run"].(string)); r3.Thread == nil || r3.Thread.Mode != posemodel.ReviewThreadNew || !strings.Contains(r3.Thread.Reason, "vendor changed") {
		t.Fatalf("a vendor change continued the session: %+v", r3.Thread)
	}
	_ = first
	root = dispatchInstance(t)
	writeDelegation(t, root, `{"schema_version": 2, "agents": {`+roundsAgent("rounder", "other", false)+`}, "routing": {"rounds": "cold-session"}, "purposes": {"review": {"max_runs_per_bundle": 5}}}`)
	roundRun(t, root, "rounder")
	nextReviewRound(t, root, "fix B1")
	cold := roundRun(t, root, "rounder")
	if r := runRecord(t, root, cold["run"].(string)); r.Thread == nil || r.Thread.Mode != posemodel.ReviewThreadNew || !strings.Contains(r.Thread.Reason, "cold-session") || strings.Contains(r.Output, "RESUMED") {
		t.Fatalf("cold-session continued: %+v %s", r.Thread, r.Output)
	}
	// R6, R7: at the round threshold a new session opens and verify warns.
	root = dispatchInstance(t)
	writeDelegation(t, root, `{"schema_version": 2, "agents": {`+roundsAgent("rounder", "other", false)+`}, "routing": {"round_threshold": 2}, "purposes": {"review": {"max_runs_per_bundle": 5}}}`)
	roundRun(t, root, "rounder")
	nextReviewRound(t, root, "fix B1")
	late := roundRun(t, root, "rounder")
	if r := runRecord(t, root, late["run"].(string)); r.Thread == nil || r.Thread.Mode != posemodel.ReviewThreadNew || !strings.Contains(r.Thread.Reason, "threshold") {
		t.Fatalf("the threshold continued the session: %+v", r.Thread)
	}
	if _, out := runPose(t, root, "review", "verify", "spec:alpha"); !strings.Contains(out, "round threshold 2") {
		t.Fatalf("verify did not warn at the threshold: %s", out)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "ls", late["run"].(string), "--json"); code != 0 || !strings.Contains(out, `"round": 2`) {
		t.Fatalf("ls does not show the round: %d %s", code, out)
	}
}

// Review round 1, B9, B10, B11: a bundle of an earlier round that cannot be
// read makes the brief refuse rather than fall back to round 1; ls and
// insights show the round in text; a resume in round 2 keeps its thread.
func TestReviewRoundsSurviveLossesAndResumes(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(roundsAgent("rounder", "other", false)))
	first := roundRun(t, root, "rounder")
	nextReviewRound(t, root, "fix B1")
	if code, out, _ := runPoseSplit(t, root, "delegate", "ls", first["run"].(string)); code != 0 || !strings.Contains(out, "round=1") {
		t.Fatalf("ls text: %d %s", code, out)
	}
	if code, out := runPose(t, root, "insights"); code != 0 || !strings.Contains(out, "stats.delegated.rounds.spec:alpha=rounds=1") {
		t.Fatalf("insights text: %d %s", code, out)
	}
	bundle := runRecord(t, root, first["run"].(string)).BundleID
	path := filepath.Join(root, ".pose", "review-bundles", bundle+".json")
	saved, _ := os.ReadFile(path)
	_ = os.WriteFile(path, []byte("{"), 0o644)
	if _, err := (posemodel.Store{Root: root}).RenderReviewBrief("spec:alpha", "review", ""); err == nil {
		t.Fatal("a lost bundle of round 1 let round 2 be briefed as round 1")
	}
	_ = os.WriteFile(path, saved, 0o644)
	marker := filepath.Join(t.TempDir(), "stopped")
	limited := strings.Replace(roundsAgent("rounder", "other", false), `"session": {`, `"limit": {"exit_codes": [75]}, "session": {`, 1)
	limited = strings.ReplaceAll(limited, `[ -n \"$1\" ] \u0026\u0026 echo RESUMED-$1;`, `[ -n \"$1\" ] \u0026\u0026 [ ! -e `+marker+` ] \u0026\u0026 touch `+marker+` \u0026\u0026 echo partial \u0026\u0026 exit 75;`)
	if !strings.Contains(limited, marker) {
		t.Fatal("the limited agent was not built")
	}
	writeDelegation(t, root, delegationPolicy(limited))
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "rounder", "--json", "--reason", "round 2")
	var stopped map[string]any
	if json.Unmarshal([]byte(out), &stopped) != nil || stopped["status"] != "interrupted" {
		if run, ok := stopped["run"].(string); ok {
			r := runRecord(t, root, run)
			t.Logf("thread=%+v output=%q", r.Thread, r.Output)
		}
		t.Fatalf("round 2 did not stop: %d %s %s", code, out, errOut)
	}
	if r := runRecord(t, root, stopped["run"].(string)); r.Round != 2 || r.Thread == nil || r.Thread.Mode != posemodel.ReviewThreadContinued {
		t.Fatalf("the stopped round 2: round=%d thread=%+v", r.Round, r.Thread)
	}
	code, out, errOut = runPoseSplit(t, root, "delegate", "resume", stopped["run"].(string), "--json")
	var resumed map[string]any
	if json.Unmarshal([]byte(out), &resumed) != nil || resumed["status"] != "completed" {
		t.Fatalf("resume: %d %s %s", code, out, errOut)
	}
	if r := runRecord(t, root, resumed["run"].(string)); r.Round != 2 || r.Thread == nil || r.Thread.Mode != posemodel.ReviewThreadContinued || len(r.DraftProblems) != 0 {
		t.Fatalf("the resumed round 2 lost its lineage: round=%d thread=%+v problems=%v", r.Round, r.Thread, r.DraftProblems)
	}
	if _, out := runPose(t, root, "insights", "--json"); !strings.Contains(out, `"continued":{"rounds":1`) {
		t.Fatalf("a resumed continued round counted as cold: %s", out)
	}
}
