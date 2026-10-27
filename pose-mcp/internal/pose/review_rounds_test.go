package pose

import (
	"strings"
	"testing"
)

func draftOutput(decision, body string) string {
	return "decision: " + decision + "\nprose the engine does not read\n```pose-draft\n" + body + "\n```\n"
}

// R4: a pose-draft block is read strictly; prose around it is not read.
func TestParseReviewDraftIsStrict(t *testing.T) {
	ok := draftOutput("changes-requested", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[{"id":"B1","severity":"high","disposition":"blocker","file":"a.go","line":3,"summary":"a defect"}]}`)
	if d, err := ParseReviewDraft(ok); err != nil || d == nil || len(d.Findings) != 1 {
		t.Fatalf("a valid draft: %v %+v", err, d)
	}
	if d, err := ParseReviewDraft("decision: approved\n"); err != nil || d != nil {
		t.Fatalf("no block is no draft: %v %+v", err, d)
	}
	for name, out := range map[string]string{
		"schema":    draftOutput("approved", `{"schema":"other","decision":"approved","findings":[]}`),
		"unknown":   draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"extra":1}`),
		"decision":  draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[]}`),
		"duplicate": draftOutput("changes-requested", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[{"id":"B1","severity":"low","disposition":"follow-up","summary":"x"},{"id":"B1","severity":"low","disposition":"follow-up","summary":"y"}]}`),
		"severity":  draftOutput("changes-requested", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[{"id":"B1","severity":"critical","disposition":"blocker","summary":"x"}]}`),
		"status":    draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[{"id":"R1-B1","status":"done","evidence":"e"}],"sweep":"all"}`),
		"unclosed":  "decision: approved\n```pose-draft\n{\"schema\":\"pose.review.draft/v1\"}",
	} {
		if _, err := ParseReviewDraft(out); err == nil {
			t.Fatalf("%s: an invalid draft was read", name)
		}
	}
}

// R1, R2, R5: the round counts the bundles a completed review concluded on
// before this one; open findings keep their identity when carried partial or
// not-fixed, a fixed one is not carried, and the section is byte-stable.
func TestReviewRoundsCarryOpenFindings(t *testing.T) {
	bundle := func(id, at string) ReviewBundle {
		return ReviewBundle{BundleID: id, BundleDigest: "sha256:" + id, SealedAt: at, Payload: ReviewBundlePayload{Subject: ReviewBundleSubject{Head: "commit-" + id}}}
	}
	run := func(id, bundle, output string) ReviewRun {
		ended := map[string]string{"rvb-1": "2026-10-27T01:30:00.5Z", "rvb-2": "2026-10-27T02:30:00.5Z", "rvb-3": "2026-10-27T03:30:00.5Z"}[bundle]
		return ReviewRun{RunID: id, BundleID: bundle, Kind: ReviewBriefKindReview, Status: ReviewRunCompleted, Output: output, EndedAt: ended}
	}
	b1, b2, b3, b4 := bundle("rvb-1", "2026-10-27T01:00:00Z"), bundle("rvb-2", "2026-10-27T02:00:00Z"), bundle("rvb-3", "2026-10-27T03:00:00Z"), bundle("rvb-4", "2026-10-27T04:00:00Z")
	r1 := run("rrn-1", "rvb-1", draftOutput("changes-requested", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[
		{"id":"B1","severity":"high","disposition":"blocker","file":"a.go","line":3,"summary":"first"},
		{"id":"B2","severity":"medium","disposition":"blocker","summary":"second"},
		{"id":"F1","severity":"low","disposition":"follow-up","summary":"third"}]}`))
	r2 := run("rrn-2", "rvb-2", draftOutput("changes-requested", `{"schema":"pose.review.draft/v1","decision":"changes-requested","findings":[{"id":"B9","severity":"low","disposition":"blocker","summary":"new"}],
		"prior_findings":[{"id":"R1-B1","status":"fixed","evidence":"a.go:3"},{"id":"R1-B2","status":"partial","evidence":"half"},{"id":"R1-F1","status":"not-fixed","evidence":"open"}],"sweep":"all"}`))
	intact := func(ReviewRun) string { return "" }
	bundles := []ReviewBundle{b3, b1, b2, b4}
	runs := []ReviewRun{r1, r2, run("rrn-x", "rvb-2", "decision: approved\n")}
	runs[2].SupersededBy = "rrn-y"
	round, err := reviewRoundFrom(b3, bundles, runs, intact)
	if err != nil || round.Number != 3 || round.Prior == nil || round.Prior.RunID != "rrn-2" {
		t.Fatalf("round: %v %+v", err, round)
	}
	ids := []string{}
	for _, f := range round.Open {
		ids = append(ids, f.ID+"/"+f.Carried)
	}
	if strings.Join(ids, ",") != "R1-B2/partial,R1-F1/not-fixed,R2-B9/" {
		t.Fatalf("open findings: %v", ids)
	}
	section := renderPriorRound(round)
	if section != renderPriorRound(round) || !strings.Contains(section, "- R1-B2 (medium, blocker, round 1) [reported partial in a later round] — second") || strings.Contains(section, "R1-B1") {
		t.Fatalf("section:\n%s", section)
	}
	if first, _ := reviewRoundFrom(b1, bundles, runs, intact); first.Number != 1 || first.Prior != nil {
		t.Fatalf("round 1: %+v", first)
	}
	// A tampered prior run refuses; an unreadable draft breaks the chain.
	if _, err := reviewRoundFrom(b3, bundles, runs, func(r ReviewRun) string { return "tampered" }); err == nil {
		t.Fatal("a tampered prior run was carried")
	}
	runs[1].Output = "decision: changes-requested\n"
	if broken, err := reviewRoundFrom(b3, bundles, runs, intact); err != nil || broken.Broken == "" || len(broken.Open) != 0 {
		t.Fatalf("a broken chain: %v %+v", err, broken)
	}
	_ = b4
}

// R3, R4, R9: from round 2 a draft accounts for every open finding, a fixed
// one with evidence, and states its sweep; otherwise it is invalid.
func TestValidateRoundDraftRequiresAnAccount(t *testing.T) {
	round := ReviewRound{Number: 2, Open: []ReviewOpenFinding{{ID: "R1-B1"}, {ID: "R1-B2"}}}
	valid := draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[{"id":"R1-B1","status":"fixed","evidence":"a.go:3"},{"id":"R1-B2","status":"superseded","evidence":"removed"}],"sweep":"the whole change"}`)
	if problems := ValidateRoundDraft(round, valid); len(problems) != 0 {
		t.Fatalf("a valid account: %v", problems)
	}
	for name, out := range map[string]string{
		"omitted":  draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[{"id":"R1-B1","status":"fixed","evidence":"a.go:3"}],"sweep":"all"}`),
		"evidence": draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[{"id":"R1-B1","status":"fixed","evidence":""},{"id":"R1-B2","status":"fixed","evidence":"x"}],"sweep":"all"}`),
		"sweep":    draftOutput("approved", `{"schema":"pose.review.draft/v1","decision":"approved","findings":[],"prior_findings":[{"id":"R1-B1","status":"fixed","evidence":"x"},{"id":"R1-B2","status":"fixed","evidence":"x"}]}`),
		"no block": "decision: approved\n",
	} {
		if problems := ValidateRoundDraft(round, out); len(problems) == 0 {
			t.Fatalf("%s: an invalid account passed", name)
		}
	}
	if problems := ValidateRoundDraft(ReviewRound{Number: 1}, "decision: approved\n"); len(problems) != 0 {
		t.Fatalf("round 1 needs no block: %v", problems)
	}
}
