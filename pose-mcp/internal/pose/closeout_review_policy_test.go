package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests of spec pose-closeout-review-policy-by-cost-benefit. Every test name
// carries the TestCloseoutReviewPolicy prefix the permanent integration check runs.

func setCloseoutPolicy(t *testing.T, root string, milestone, roadmap string) {
	t.Helper()
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["schema_version"] = CloseoutReviewPolicySchemaVersion
	policy[MilestoneReviewRequiredKey], policy[RoadmapReviewRequiredKey] = milestone, roadmap
	out, _ := json.Marshal(policy)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

// closeoutMembersFixture is an instance with specs a..e and roadmaps whose
// milestones have one, two and three members, without reviews: enough for the
// evaluator, which reads composition and associations only.
func closeoutMembersFixture(t *testing.T, milestone, roadmap string) (string, Store) {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/project.json", `{"schema_version":1,"project_id":"proj.fixture","name":"fixture"}`)
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1","milestone":"milestone-integration@1","roadmap":"roadmap-outcome@1"}}`)
	for _, slug := range []string{"a", "b", "c", "d", "e"} {
		write(".pose/specs/"+slug+".md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-11-01\n---\n\n# Spec: "+slug+"\n")
	}
	write(".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: one\n- specs: a\n\n## Milestone: two\n- specs: a, b\n\n## Milestone: three\n- specs: a, b, c\n\n## Milestone: aliases\n- specs: a, spec:a, xref:proj.fixture/spec:a\n")
	write(".pose/roadmaps/single.md", "---\nslug: single\nstatus: active\nadrs: [adr:decision.md]\n---\n\n## Milestone: only\n- specs: d\n\n## Milestone: again\n- specs: spec:d\n")
	write(".pose/roadmaps/pair.md", "---\nslug: pair\nstatus: active\n---\n\nThe pair roadmap mentions decision.md in prose only.\n\n## Milestone: m1\n- specs: d\n\n## Milestone: m2\n- specs: e\n")
	write(".pose/adr/decision.md", "# ADR: decision\n")
	if milestone != "" {
		setCloseoutPolicy(t, root, milestone, roadmap)
	}
	return root, Store{Root: root}
}

func evaluate(t *testing.T, s Store, ref string) CloseoutReviewDecision {
	t.Helper()
	decision, err := s.EvaluateCloseoutReview(ref)
	if err != nil {
		t.Fatalf("%s: %v", ref, err)
	}
	return decision
}

func TestCloseoutReviewPolicySchema(t *testing.T) {
	s := Store{Root: t.TempDir()}
	cases := []struct {
		name, policy string
		valid        bool
	}{
		{"legacy schema 2 without the pair", `{"schema_version":2,"enabled":true}`, true},
		{"legacy schema 4 without the pair", `{"schema_version":4,"enabled":true,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1}`, true},
		{"schema 5 with the pair", `{"schema_version":5,"enabled":true,"milestone_review_required":"never","roadmap_review_required":"when-justified"}`, true},
		{"schema 5 with refs and transfer", `{"schema_version":5,"enabled":true,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"milestone_review_required":"always","roadmap_review_required":"always"}`, true},
		{"schema 5 with refs only", `{"schema_version":5,"enabled":true,"qualified_artifact_refs_version":1,"milestone_review_required":"never","roadmap_review_required":"never"}`, true},
		{"schema 5 transfer without refs", `{"schema_version":5,"enabled":true,"spec_authority_transfer_version":1,"milestone_review_required":"never","roadmap_review_required":"never"}`, false},
		{"schema 5 without the pair", `{"schema_version":5,"enabled":true}`, false},
		{"schema 5 with one key", `{"schema_version":5,"enabled":true,"milestone_review_required":"never"}`, false},
		{"null value", `{"schema_version":5,"enabled":true,"milestone_review_required":null,"roadmap_review_required":"never"}`, false},
		{"empty value", `{"schema_version":5,"enabled":true,"milestone_review_required":"","roadmap_review_required":"never"}`, false},
		{"wrong type", `{"schema_version":5,"enabled":true,"milestone_review_required":1,"roadmap_review_required":"never"}`, false},
		{"unknown enum", `{"schema_version":5,"enabled":true,"milestone_review_required":"sometimes","roadmap_review_required":"never"}`, false},
		{"duplicated key", `{"schema_version":5,"enabled":true,"milestone_review_required":"always","milestone_review_required":"never","roadmap_review_required":"never"}`, false},
		{"new key under an old schema", `{"schema_version":2,"enabled":true,"milestone_review_required":"never","roadmap_review_required":"never"}`, false},
		{"downgrade keeps the pair", `{"schema_version":4,"enabled":true,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"milestone_review_required":"never","roadmap_review_required":"never"}`, false},
		{"schema 6 is unknown", `{"schema_version":6,"enabled":true,"milestone_review_required":"never","roadmap_review_required":"never"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := s.parseReviewPolicy([]byte(tc.policy))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
			if err == nil && !strings.Contains(tc.policy, `"schema_version":5`) && policy.CloseoutReviewAdopted() {
				t.Fatal("a legacy policy reads as adopted")
			}
		})
	}
	// The resolver's own probe accepts schema 5 and no later one.
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755)
	os.WriteFile(filepath.Join(root, ".pose", "policy", "review.json"), []byte(`{"schema_version":5,"enabled":true,"milestone_review_required":"never","roadmap_review_required":"never"}`), 0o644)
	if err := requireSpecTransferCapability(NativeFiles(), Store{Root: root}); err == nil {
		t.Fatal("transfer reads as adopted under schema 5 without transfer_version")
	}
}

func TestCloseoutReviewPolicyThresholds(t *testing.T) {
	type want struct {
		ref      string
		required bool
	}
	cases := map[string][]want{
		"never": {{"milestone:program/one", false}, {"milestone:program/three", false}, {"roadmap:program", false}, {"roadmap:pair", false}},
		"when-justified": {
			{"milestone:program/one", false}, {"milestone:program/two", false}, {"milestone:program/three", true},
			{"milestone:program/aliases", false},
			// three distinct members, no ADR association
			{"roadmap:program", false},
			// one distinct member (d twice) with an ADR
			{"roadmap:single", false},
			// two members, and an ADR only named in prose
			{"roadmap:pair", false},
		},
		"always": {{"milestone:program/one", true}, {"milestone:program/aliases", true}, {"roadmap:single", true}},
	}
	for mode, wants := range cases {
		t.Run(mode, func(t *testing.T) {
			_, s := closeoutMembersFixture(t, mode, mode)
			for _, w := range wants {
				d := evaluate(t, s, w.ref)
				if !d.Adopted || d.Required != w.required || !d.Known || d.EvaluatorVersion != CloseoutReviewEvaluatorVersion || d.PolicyValue != mode {
					t.Fatalf("%s under %s: %+v", w.ref, mode, d)
				}
				if d.Reason == "" {
					t.Fatalf("%s has no reason", w.ref)
				}
			}
		})
	}
	t.Run("roadmap with two members and an ADR", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "status: active\n", "status: active\nadrs: [adr:decision.md]\n")
		d := evaluate(t, s, "roadmap:pair")
		if !d.Required || len(d.ADRs) != 1 || d.Reason != "members=2, threshold>1, adrs=1" {
			t.Fatalf("%+v", d)
		}
	})
	t.Run("one-member milestone reason", func(t *testing.T) {
		_, s := closeoutMembersFixture(t, "when-justified", "never")
		if d := evaluate(t, s, "milestone:program/one"); d.Reason != "members=1, threshold>2" || d.Review() != "not-required" {
			t.Fatalf("%+v", d)
		}
	})
	t.Run("legacy policy is not adopted", func(t *testing.T) {
		_, s := closeoutMembersFixture(t, "", "")
		if d := evaluate(t, s, "milestone:program/three"); d.Adopted {
			t.Fatalf("%+v", d)
		}
	})
}

func editFile(t *testing.T, path, from, to string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), from) {
		t.Fatalf("%s has no %q", path, from)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCloseoutReviewPolicyMembers(t *testing.T) {
	root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
	d := evaluate(t, s, "milestone:program/aliases")
	if len(d.Members) != 1 || d.Members[0] != "xref:proj.fixture/spec:a" {
		t.Fatalf("aliases were counted apart: %+v", d.Members)
	}
	d = evaluate(t, s, "roadmap:program")
	if strings.Join(d.Members, ",") != "xref:proj.fixture/spec:a,xref:proj.fixture/spec:b,xref:proj.fixture/spec:c" {
		t.Fatalf("roadmap union: %+v", d.Members)
	}
	t.Run("missing spec is unknown, never zero", func(t *testing.T) {
		editFile(t, filepath.Join(root, ".pose/roadmaps/program.md"), "- specs: a\n", "- specs: ghost\n")
		d := evaluate(t, s, "milestone:program/one")
		if d.Known || !d.Required || len(d.Diagnostics) == 0 {
			t.Fatalf("an unknown member granted an exemption: %+v", d)
		}
	})
	t.Run("external member without a resolver is unknown", func(t *testing.T) {
		editFile(t, filepath.Join(root, ".pose/roadmaps/program.md"), "- specs: a, b\n", "- specs: a, xref:other/spec:b\n")
		d := evaluate(t, s, "milestone:program/two")
		if d.Known || !d.Required {
			t.Fatalf("%+v", d)
		}
	})
	t.Run("never still reports unknown members and does not exempt", func(t *testing.T) {
		setCloseoutPolicy(t, root, "never", "never")
		d := evaluate(t, s, "milestone:program/one")
		if d.Required || d.Known {
			t.Fatalf("%+v", d)
		}
		os.MkdirAll(filepath.Join(root, ".pose", "review-profiles"), 0o755)
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "milestone-integration.json"), []byte(`{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"x","kind":"judgment","evidence_classes":["integration"]}]}`), 0o644)
		eval, err := s.ReviewCheck("milestone:program/one")
		// Failing closed with an error is as good as a blocked evaluation.
		if err == nil && (!eval.Required || eval.Approved || len(eval.Blockers) == 0) {
			t.Fatalf("an unknown member read as an exemption: %+v", eval)
		}
	})
}

func TestCloseoutReviewPolicyAssociations(t *testing.T) {
	cases := []struct {
		name, roadmapFrontmatter, roadmapBody, adr string
		adrs                                       int
		known                                      bool
	}{
		{"forward metadata", "adrs: [adr:decision.md]\n", "", "# ADR\n", 1, true},
		{"backward metadata", "", "", "---\nroadmaps: [roadmap:pair]\n---\n# ADR\n", 1, true},
		{"qualified backward metadata", "", "", "---\nroadmaps: [xref:proj.fixture/roadmap:pair]\n---\n# ADR\n", 1, true},
		{"backward metadata for another project", "", "", "---\nroadmaps: [xref:proj.other/roadmap:pair]\n---\n# ADR\n", 0, true},
		{"inline link from the roadmap", "", "See [the ADR](../adr/decision.md#context).\n", "# ADR\n", 1, true},
		{"angle bracket link", "", "See [the ADR](<../adr/decision.md>).\n", "# ADR\n", 1, true},
		{"reference-style link", "", "See [the ADR][d].\n\n[d]: ../adr/decision.md \"title\"\n", "# ADR\n", 1, true},
		{"link back from the ADR", "", "", "Governs [the roadmap](../roadmaps/pair.md).\n", 1, true},
		{"both directions count once", "adrs: [adr:decision.md]\n", "", "---\nroadmaps: [roadmap:pair]\n---\n", 1, true},
		{"word in prose", "", "This follows decision.md and the decision ADR.\n", "Mentions pair and roadmap pair.\n", 0, true},
		{"external URL", "", "See [x](https://example.invalid/.pose/adr/decision.md).\n", "# ADR\n", 0, true},
		{"traversal declaration", "adrs: [adr:../decision.md]\n", "", "# ADR\n", 0, false},
		{"missing declared ADR", "adrs: [adr:absent.md]\n", "", "# ADR\n", 0, false},
		{"link to a missing ADR", "", "See [x](../adr/absent.md).\n", "# ADR\n", 0, false},
		{"malformed declaration", "adrs: adr:decision.md\n", "", "# ADR\n", 0, false},
		{"malformed backward declaration", "", "", "---\nroadmaps: [spec:pair]\n---\n", 0, false},
		{"forward association does not hide a malformed ADR", "adrs: [adr:decision.md]\n", "", "---\nroadmaps: [spec:pair]\n---\n", 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
			os.WriteFile(filepath.Join(root, ".pose/roadmaps/pair.md"), []byte("---\nslug: pair\nstatus: active\n"+tc.roadmapFrontmatter+"---\n\n"+tc.roadmapBody+"\n## Milestone: m1\n- specs: d\n\n## Milestone: m2\n- specs: e\n"), 0o644)
			os.WriteFile(filepath.Join(root, ".pose/adr/decision.md"), []byte(tc.adr), 0o644)
			d := evaluate(t, s, "roadmap:pair")
			if len(d.ADRs) != tc.adrs || d.Known != tc.known {
				t.Fatalf("adrs=%v known=%v diagnostics=%v", d.ADRs, d.Known, d.Diagnostics)
			}
			if d.Required != (tc.adrs > 0 || !tc.known) {
				t.Fatalf("required=%v for %+v", d.Required, d)
			}
		})
	}
	t.Run("symlinked ADR is not an association", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		outside := filepath.Join(t.TempDir(), "outside.md")
		os.WriteFile(outside, []byte("# outside\n"), 0o644)
		if err := os.Symlink(outside, filepath.Join(root, ".pose/adr/linked.md")); err != nil {
			t.Skip(err)
		}
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "status: active\n", "status: active\nadrs: [adr:linked.md]\n")
		if d := evaluate(t, s, "roadmap:pair"); d.Known || len(d.ADRs) != 0 {
			t.Fatalf("%+v", d)
		}
	})
	t.Run("the shared runtime roadmap link form survives", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.WriteFile(filepath.Join(root, ".pose/adr/2026-10-14-shared-domain.md"), []byte("# ADR\n"), 0o644)
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "mentions decision.md in prose only.", "follows the [ADR de domínio/runtime Proposed](../adr/2026-10-14-shared-domain.md).")
		if d := evaluate(t, s, "roadmap:pair"); !d.Required || strings.Join(d.ADRs, ",") != "2026-10-14-shared-domain.md" {
			t.Fatalf("%+v", d)
		}
	})
}

// closeoutReviewedFixture is a bundle instance whose milestone program/core has
// the one done, approved member backend.
func closeoutReviewedFixture(t *testing.T, milestone, roadmap string) (string, Store) {
	t.Helper()
	root, s := inheritedFixture(t)
	setCloseoutPolicy(t, root, milestone, roadmap)
	return root, s
}

func TestCloseoutReviewPolicyProtection(t *testing.T) {
	t.Run("always does not inherit the sole member", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "always", "always")
		verification, err := s.VerifyReviewBundle("milestone:program/core")
		if err != nil {
			t.Fatal(err)
		}
		if verification.Approved {
			t.Fatalf("always inherited a member approval: %+v", verification)
		}
		eval, err := s.ReviewCheck("milestone:program/core")
		if err != nil || !eval.Required || eval.Approved || eval.CloseoutReview == nil || !eval.CloseoutReview.Required {
			t.Fatalf("%+v %v", eval, err)
		}
	})
	t.Run("always ignores unreviewed_milestones", func(t *testing.T) {
		root, s := closeoutReviewedFixture(t, "always", "always")
		editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: the team accepts it\n")
		if verification, err := s.VerifyReviewBundle("milestone:program/core"); err != nil || verification.Approved {
			t.Fatalf("a declaration weakened always: %+v %v", verification, err)
		}
	})
	t.Run("never dispenses only the aggregate", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		verification, err := s.VerifyReviewBundle("milestone:program/core")
		if err != nil || !verification.Approved || verification.ApprovalOrigin != "not-required:milestone_review_required=never" {
			t.Fatalf("%+v %v", verification, err)
		}
		member, err := s.ReviewCheck("spec:backend")
		if err != nil || !member.Required {
			t.Fatalf("the member review was dispensed too: %+v %v", member, err)
		}
	})
	t.Run("enabled=false does not weaken always", func(t *testing.T) {
		root, s := closeoutReviewedFixture(t, "always", "always")
		path := filepath.Join(root, ".pose", "policy", "review.json")
		raw, _ := os.ReadFile(path)
		os.WriteFile(path, []byte(strings.Replace(string(raw), `"enabled":true`, `"enabled":false`, 1)), 0o644)
		eval, err := s.ReviewCheck("milestone:program/core")
		if !eval.Required || eval.Approved || eval.CloseoutReview == nil {
			t.Fatalf("a disabled policy dispensed an always review: %+v %v", eval, err)
		}
		if state, err := s.GetCloseoutState("milestone:program/core"); err == nil && state.Terminal {
			t.Fatalf("%+v", state)
		}
		owed, err := s.CloseoutReviewsOwed()
		if err != nil || len(owed) == 0 {
			t.Fatalf("doctor lost the owed review under enabled=false: %+v %v", owed, err)
		}
	})
	t.Run("mandatory-human is never dispensed", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "never", "never")
		path := filepath.Join(root, ".pose", "policy", "review.json")
		raw, _ := os.ReadFile(path)
		var policy map[string]any
		json.Unmarshal(raw, &policy)
		policy["reviewer_independence"] = map[string]any{"milestone": "mandatory-human"}
		os.MkdirAll(filepath.Join(root, ".pose", "review-profiles"), 0o755)
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "roadmap-outcome.json"), []byte(`{"schema_version":1,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"x","kind":"judgment","evidence_classes":["integration"]}]}`), 0o644)
		out, _ := json.Marshal(policy)
		os.WriteFile(path, out, 0o644)
		d, err := s.CurrentCloseoutDecision("milestone:program/one")
		if err != nil || !d.Required || !strings.Contains(d.Reason, "mandatory-human") {
			t.Fatalf("never dispensed a mandatory-human review: %+v %v", d, err)
		}
		if d, _ := s.CurrentCloseoutDecision("roadmap:program"); d.Required {
			t.Fatalf("the roadmap kind inherited the milestone's human requirement: %+v", d)
		}
	})
}

func TestCloseoutReviewPolicyVeto(t *testing.T) {
	_, s := closeoutReviewedFixture(t, "never", "never")
	bundle, err := s.SealReviewBundle("milestone:program/core", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, s, bundle, reviewFixtureNow.Add(3*time.Minute), "rejected")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil || verification.Approved {
		t.Fatalf("a rejection made by choice was laundered by never: %+v %v", verification, err)
	}
	eval, err := s.ReviewCheck("milestone:program/core")
	if err != nil || eval.Approved || !eval.Required {
		t.Fatalf("%+v %v", eval, err)
	}
	state, err := s.GetCloseoutState("milestone:program/core")
	if err != nil || state.Terminal {
		t.Fatalf("%+v %v", state, err)
	}
}

func TestCloseoutReviewPolicyReceipt(t *testing.T) {
	_, s := closeoutReviewedFixture(t, "never", "never")
	ref := "milestone:program/core"
	state, err := s.GetCloseoutState(ref)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's legacy index leaves the read view unknown, which may name
	// itself as the next step; the close is still what the scope owes.
	if state.Terminal || !state.CloseRequired || state.NextStep == nil || state.NextStep.Code != "close-scope" && state.NextStep.Code != "verify-governance-read-view" {
		t.Fatalf("an eligible milestone read as terminal or not closable: %+v", state)
	}
	if state.Review.Required || state.Review.CloseoutReview == nil || state.Review.CloseoutReview.Review() != "not-required" {
		t.Fatalf("%+v", state.Review)
	}
	reached, err := s.ReachedMilestones("backend")
	if err != nil || len(reached) != 1 || reached[0] != ref {
		t.Fatalf("reached: %v %v", reached, err)
	}
	receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Review != "not-required" || receipt.Decision.PolicyKey != MilestoneReviewRequiredKey || receipt.Decision.Reason != "members=1, never required" {
		t.Fatalf("%+v", receipt)
	}
	if err := s.PublishCloseoutReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	if err := s.PublishCloseoutReceipt(receipt); err != ErrCloseoutReceiptExists {
		t.Fatalf("a receipt was replaced: %v", err)
	}
	state, err = s.GetCloseoutState(ref)
	if err != nil || !state.Terminal || state.CloseoutReceipt == nil || state.CloseRequired {
		t.Fatalf("the receipt did not close the milestone: %+v %v", state, err)
	}
	if reached, _ := s.ReachedMilestones("backend"); len(reached) != 0 {
		t.Fatalf("a closed milestone is still reached: %v", reached)
	}
	t.Run("a policy change does not reinterpret the receipt", func(t *testing.T) {
		root := s.Root
		setCloseoutPolicy(t, root, "always", "always")
		state, err := s.GetCloseoutState(ref)
		if err != nil || !state.Terminal || state.Review.Required {
			t.Fatalf("%+v %v", state, err)
		}
	})
	t.Run("a corrupt receipt blocks and is not overwritten", func(t *testing.T) {
		path, _ := s.CloseoutReceiptPath(ref)
		os.WriteFile(path, []byte(`{"schema_version":1}`), 0o644)
		state, err := s.GetCloseoutState(ref)
		if err == nil && state.Terminal {
			t.Fatalf("a corrupt receipt read as closed: %+v", state)
		}
		if err := s.PublishCloseoutReceipt(receipt); err == nil {
			t.Fatal("a corrupt receipt was overwritten")
		}
	})
}

func TestCloseoutReviewPolicyHistory(t *testing.T) {
	root, s := inheritedFixture(t)
	ref := "milestone:program/core"
	before, err := s.VerifyReviewBundle(ref)
	if err != nil || !before.Approved {
		t.Fatalf("legacy inheritance: %+v %v", before, err)
	}
	// A done roadmap closed under the legacy rule keeps its sealed reading
	// after adoption, under any value, and gets no backfilled receipt. Its
	// legacy closure is positive evidence: here, the legacy rule exempted it
	// as done before review adoption.
	editRoadmapFixture(t, root, "status: active\n", "status: done\ncreated_at: 2026-01-01\n")
	setPolicyField(t, root, "adopted_at", "2026-09-01")
	for _, mode := range []string{"never", "when-justified", "always"} {
		setCloseoutPolicy(t, root, mode, mode)
		after, err := s.VerifyReviewBundle(ref)
		if err != nil || after.Approved != before.Approved || after.ApprovalOrigin != before.ApprovalOrigin {
			t.Fatalf("%s reinterpreted history: %+v %v", mode, after, err)
		}
		if receipt, err := s.ReadCloseoutReceipt(ref); err != nil || receipt != nil {
			t.Fatalf("a receipt appeared: %+v %v", receipt, err)
		}
		if eval, err := s.ReviewCheck(ref); err != nil || eval.CloseoutReview != nil {
			t.Fatalf("history was evaluated by the current policy: %+v %v", eval, err)
		}
	}
}

func TestCloseoutReviewPolicyUnavailable(t *testing.T) {
	root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
	t.Run("unreadable ADR inventory", func(t *testing.T) {
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "status: active\n", "status: active\n")
		dir := filepath.Join(root, ".pose", "adr")
		if err := os.Chmod(dir, 0o000); err != nil {
			t.Skip(err)
		}
		defer os.Chmod(dir, 0o755)
		if _, err := os.ReadDir(dir); err == nil {
			t.Skip("running with privileges that read a 0000 directory")
		}
		d := evaluate(t, s, "roadmap:pair")
		if d.Known || !d.Required {
			t.Fatalf("an unreadable inventory proved absence: %+v", d)
		}
	})
	t.Run("unreadable project identity", func(t *testing.T) {
		os.WriteFile(filepath.Join(root, ".pose", "project.json"), []byte(`{`), 0o644)
		d := evaluate(t, s, "milestone:program/one")
		if d.Known || !d.Required {
			t.Fatalf("%+v", d)
		}
	})
	t.Run("invalid policy is refused before any decision", func(t *testing.T) {
		path := filepath.Join(root, ".pose", "policy", "review.json")
		editFile(t, path, `"milestone_review_required":"when-justified"`, `"milestone_review_required":"maybe"`)
		if _, err := s.EvaluateCloseoutReview("milestone:program/one"); err == nil {
			t.Fatal("an invalid value decided")
		}
	})
}

func TestCloseoutReviewPolicyFederation(t *testing.T) {
	root, s, _, authorized := federatedTestMilestoneConsumer(t)
	approveMember(t, s, reviewFixtureNow)
	setCloseoutPolicy(t, root, "never", "never")
	ref := "milestone:program/core"
	d := evaluate(t, s, ref)
	if !d.Known || len(d.Members) != 2 || d.Required {
		t.Fatalf("an authorized external member did not resolve to its own identity: %+v", d)
	}
	for _, member := range d.Members {
		if !strings.HasPrefix(member, "xref:") {
			t.Fatalf("member %s is not canonical", member)
		}
	}
	*authorized = false
	d = evaluate(t, s, ref)
	if d.Known || len(d.Diagnostics) == 0 {
		t.Fatalf("an unauthorized source read as resolved: %+v", d)
	}
	if verification, err := s.VerifyReviewBundle(ref); err == nil && verification.Approved {
		t.Fatalf("the consumer's never exempted a milestone with an unknown producer member: %+v", verification)
	}
	if state, err := s.GetCloseoutState(ref); err == nil && state.Terminal {
		t.Fatalf("%+v", state)
	}
}

func TestCloseoutReviewPolicyReceiptIntegrity(t *testing.T) {
	_, s := closeoutReviewedFixture(t, "never", "never")
	ref := "milestone:program/core"
	state, err := s.GetCloseoutState(ref)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("always cannot record not-required", func(t *testing.T) {
		forged := receipt
		forged.Decision.PolicyValue = CloseoutReviewAlways
		if _, err := parseCloseoutReceipt(CanonicalJSON(forged)); err == nil {
			t.Fatal("an always receipt without a required review parsed")
		}
	})
	if err := s.PublishCloseoutReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReviewPlan(ref); err != nil {
		t.Fatalf("a standing receipt failed the plan: %v", err)
	}
	t.Run("a changed composition does not inherit the receipt", func(t *testing.T) {
		editRoadmapFixture(t, s.Root, "- specs: backend\n", "- specs: backend, pending\n")
		if _, err := s.ReviewCheck(ref); err == nil || !strings.Contains(err.Error(), "no longer matches its scope") {
			t.Fatalf("a receipt carried over to another composition: %v", err)
		}
		editRoadmapFixture(t, s.Root, "- specs: backend, pending\n", "- specs: backend\n")
	})
	t.Run("a corrupt receipt fails the plan as it fails closeout", func(t *testing.T) {
		path, _ := s.CloseoutReceiptPath(ref)
		os.WriteFile(path, []byte(`{"schema_version":1}`), 0o644)
		if _, err := s.ReviewPlan(ref); err == nil {
			t.Fatal("the plan hid a corrupt receipt")
		}
	})
}

func TestCloseoutReviewPolicyInterruptedRoadmapClose(t *testing.T) {
	root, s := closeoutReviewedFixture(t, "never", "never")
	ref := "roadmap:program"
	decision, err := s.CurrentCloseoutDecision(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PublishCloseoutIntent(ref, decision, reviewFixtureNow); err != nil {
		t.Fatal(err)
	}
	editRoadmapFixture(t, root, "status: active\n", "status: done\n")
	eval, err := s.ReviewCheck("milestone:program/core")
	if err != nil || eval.CloseoutReview == nil {
		t.Fatalf("a close interrupted before its receipt read as history: %+v %v", eval, err)
	}
	if err := s.ClearCloseoutIntent(ref); err != nil {
		t.Fatal(err)
	}
	// Without the intent, status alone is still no evidence of a legacy close.
	if eval, err := s.ReviewCheck("milestone:program/core"); err != nil || eval.CloseoutReview == nil {
		t.Fatalf("a hand-marked roadmap read as history: %+v %v", eval, err)
	}
	// With evidence of the legacy closure it is history.
	editRoadmapFixture(t, root, "status: done\n", "status: done\ncreated_at: 2026-01-01\n")
	setPolicyField(t, root, "adopted_at", "2026-09-01")
	if eval, err := s.ReviewCheck("milestone:program/core"); err != nil || eval.CloseoutReview != nil {
		t.Fatalf("a roadmap done before adoption was evaluated: %+v %v", eval, err)
	}
}

func TestCloseoutReviewPolicyFederatedReceiptProof(t *testing.T) {
	_, s := closeoutReviewedFixture(t, "never", "never")
	ref := "milestone:program/core"
	identity := ArtifactRef{Kind: "milestone", Slug: "program", Milestone: "core", Project: "coordinator"}
	if _, _, _, _, _, _, blockers := federatedSourceProof(NativeFiles(), s, identity, gitHeadAtRoot(s.Root)); len(blockers) == 0 {
		t.Fatal("an exempt milestone without its receipt was federated as closed")
	}
	state, err := s.GetCloseoutState(ref)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PublishCloseoutReceipt(receipt); err != nil {
		t.Fatal(err)
	}
	id, digest, _, _, _, _, blockers := federatedSourceProof(NativeFiles(), s, identity, gitHeadAtRoot(s.Root))
	if len(blockers) != 0 || id != "closeout-receipt:"+ref || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("the receipt of an exempt close did not prove it to a consumer: %s %s %v", id, digest, blockers)
	}
}

func setPolicyField(t *testing.T, root, key string, value any) {
	t.Helper()
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy[key] = value
	out, _ := json.Marshal(policy)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCloseoutReviewPolicyStatusAloneIsNotHistory(t *testing.T) {
	root, s := closeoutReviewedFixture(t, "always", "always")
	// Marked done by hand, with no receipt, no intent and no evidence of a
	// legacy closure: the adopted rule still governs its milestones.
	editRoadmapFixture(t, root, "status: active\n", "status: done\n")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil || verification.Approved {
		t.Fatalf("a hand-edited status re-enabled singleton inheritance under always: %+v %v", verification, err)
	}
	if eval, err := s.ReviewCheck("milestone:program/core"); err != nil || eval.CloseoutReview == nil || !eval.Required {
		t.Fatalf("%+v %v", eval, err)
	}
}

func TestCloseoutReviewPolicyRoundTwo(t *testing.T) {
	t.Run("the effective plan decides the human requirement", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "never", "never")
		os.MkdirAll(filepath.Join(root, ".pose", "review-profiles"), 0o755)
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "human-gate.json"), []byte(`{"schema_version":2,"id":"human-gate","version":1,"scope":"milestone","independence":"mandatory-human","criteria":[{"id":"human","description":"x","evidence_classes":["integration"]}]}`), 0o644)
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "milestone-integration.json"), []byte(`{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"x","kind":"judgment","evidence_classes":["integration"]}]}`), 0o644)
		setPolicyField(t, root, "overlay_profiles", []string{"human-gate@1"})
		// The plan selects no overlay here, so it asks for no human either.
		if d, err := s.CurrentCloseoutDecision("milestone:program/one"); err != nil || d.Required {
			t.Fatalf("an overlay the plan does not select forced a review: %+v %v", d, err)
		}
		// A base profile asking for a human is in the effective plan.
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "milestone-integration.json"), []byte(`{"schema_version":2,"id":"milestone-integration","version":1,"scope":"milestone","independence":"mandatory-human","criteria":[{"id":"integration","description":"x","evidence_classes":["integration"]}]}`), 0o644)
		if d, err := s.CurrentCloseoutDecision("milestone:program/one"); err != nil || !d.Required || !strings.Contains(d.Reason, "mandatory-human") {
			t.Fatalf("never dispensed the plan's mandatory-human review: %+v %v", d, err)
		}
	})
	t.Run("a single-member roadmap with a malformed association is unknown", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		editFile(t, filepath.Join(root, ".pose/roadmaps/single.md"), "adrs: [adr:decision.md]", "adrs: [adr:../decision.md]")
		if d := evaluate(t, s, "roadmap:single"); d.Known || !d.Required {
			t.Fatalf("an unknown association granted an exemption below the threshold: %+v", d)
		}
	})
	t.Run("an unreadable unassociated ADR candidate is unknown", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "status: active\n", "status: active\n")
		outside := filepath.Join(t.TempDir(), "elsewhere.md")
		os.WriteFile(outside, []byte("---\nroadmaps: [roadmap:pair]\n---\n"), 0o644)
		if err := os.Symlink(outside, filepath.Join(root, ".pose/adr/elsewhere.md")); err != nil {
			t.Skip(err)
		}
		if d := evaluate(t, s, "roadmap:pair"); d.Known {
			t.Fatalf("an unreadable candidate became absence: %+v", d)
		}
	})
	t.Run("an associated README is read too", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.WriteFile(filepath.Join(root, ".pose/adr/README.md"), []byte("---\nroadmaps: [spec:pair]\n---\n"), 0o644)
		editFile(t, filepath.Join(root, ".pose/roadmaps/pair.md"), "status: active\n", "status: active\nadrs: [adr:README.md]\n")
		if d := evaluate(t, s, "roadmap:pair"); d.Known {
			t.Fatalf("an associated README was not validated: %+v", d)
		}
	})
	t.Run("an exit gate change does not inherit the receipt", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		ref := "milestone:program/core"
		state, err := s.GetCloseoutState(ref)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil || s.PublishCloseoutReceipt(receipt) != nil {
			t.Fatal(err)
		}
		editRoadmapFixture(t, s.Root, "- specs: backend\n", "- specs: backend\n\n**Exit gate:** a gate written after the close\n")
		if _, err := s.ReviewCheck(ref); err == nil || !strings.Contains(err.Error(), "exit gate") {
			t.Fatalf("a receipt carried over a changed exit gate: %v", err)
		}
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 1 {
			t.Fatalf("a milestone whose receipt no longer stands is not reached again: %v %v", reached, err)
		}
	})
	t.Run("a refusal recorded after the receipt reopens the milestone", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		ref := "milestone:program/core"
		state, _ := s.GetCloseoutState(ref)
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil || s.PublishCloseoutReceipt(receipt) != nil {
			t.Fatal(err)
		}
		bundle, err := s.SealReviewBundle(ref, reviewFixtureNow.Add(2*time.Hour))
		if err != nil {
			t.Fatal(err, bundle.Blockers)
		}
		attestChildReview(t, s, bundle, reviewFixtureNow.Add(3*time.Hour), "rejected")
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 1 {
			t.Fatalf("attention hid a milestone rejected after its receipt: %v %v", reached, err)
		}
		if state, err := s.GetCloseoutState(ref); err == nil && state.Terminal {
			t.Fatalf("%+v", state)
		}
	})
}

func TestCloseoutReviewPolicyRoundThree(t *testing.T) {
	t.Run("a backward declaration of a missing roadmap is unknown", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.WriteFile(filepath.Join(root, ".pose/adr/decision.md"), []byte("---\nroadmaps: [roadmap:missing]\n---\n"), 0o644)
		if d := evaluate(t, s, "roadmap:pair"); d.Known {
			t.Fatalf("an unresolved backward target proved absence: %+v", d)
		}
	})
	t.Run("a README backlink counts", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.WriteFile(filepath.Join(root, ".pose/adr/README.md"), []byte("---\nroadmaps: [roadmap:pair]\n---\n"), 0o644)
		if d := evaluate(t, s, "roadmap:pair"); !d.Known || !d.Required || strings.Join(d.ADRs, ",") != "README.md" {
			t.Fatalf("a README backlink was ignored: %+v", d)
		}
	})
	t.Run("a sealed but unattested review after the receipt reopens reached", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		ref := "milestone:program/core"
		state, _ := s.GetCloseoutState(ref)
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil || s.PublishCloseoutReceipt(receipt) != nil {
			t.Fatal(err)
		}
		if bundle, err := s.SealReviewBundle(ref, reviewFixtureNow.Add(2*time.Hour)); err != nil {
			t.Fatal(err, bundle.Blockers)
		}
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 1 {
			t.Fatalf("a pending review after the receipt was hidden: %v %v", reached, err)
		}
	})
	t.Run("a roadmap receipt binds its cut criteria", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		ref := "roadmap:program"
		decision, err := s.CurrentCloseoutDecision(ref)
		if err != nil {
			t.Fatal(err)
		}
		state, _ := s.GetCloseoutState(ref)
		receipt, err := s.NewCloseoutReceipt(ref, decision, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil || s.PublishCloseoutReceipt(receipt) != nil {
			t.Fatal(err)
		}
		editRoadmapFixture(t, s.Root, "## Milestone: core\n", "## Cut criteria\n\n- C1: manual-review:docs/missing.md\n\n## Milestone: core\n")
		if _, err := s.ReviewCheck(ref); err == nil || !strings.Contains(err.Error(), "exit gate") {
			t.Fatalf("a roadmap receipt survived a new cut criterion: %v", err)
		}
	})
	t.Run("emitted receipts conform to the published schema", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		ref := "milestone:program/core"
		state, _ := s.GetCloseoutState(ref)
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(receipt)
		var emitted map[string]any
		json.Unmarshal(raw, &emitted)
		schemaRaw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v1", "closeout-receipt.schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Required   []string                  `json:"required"`
			Properties map[string]map[string]any `json:"properties"`
			Defs       map[string]struct {
				Properties map[string]any `json:"properties"`
			} `json:"$defs"`
		}
		json.Unmarshal(schemaRaw, &schema)
		for key := range emitted {
			if _, ok := schema.Properties[key]; !ok {
				t.Fatalf("receipt field %q is not in the published schema", key)
			}
		}
		for _, key := range schema.Required {
			if _, ok := emitted[key]; !ok {
				t.Fatalf("required field %q is not emitted", key)
			}
		}
		for key := range emitted["decision"].(map[string]any) {
			if _, ok := schema.Defs["decision"].Properties[key]; !ok {
				t.Fatalf("decision field %q is not in the published schema", key)
			}
		}
	})
}

func TestCloseoutReviewPolicyRoundFour(t *testing.T) {
	t.Run("a dispensed multi-member milestone with an exit gate keeps its mode", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.MkdirAll(filepath.Join(root, ".pose", "review-profiles"), 0o755)
		os.WriteFile(filepath.Join(root, ".pose", "review-profiles", "milestone-integration.json"), []byte(`{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"x","kind":"judgment","evidence_classes":["integration"]}]}`), 0o644)
		editFile(t, filepath.Join(root, ".pose/roadmaps/program.md"), "- specs: a, b\n", "- specs: a, b\n\n**Exit gate:** both parts integrate\n")
		// Round 5 N19: the threshold decides, the gate adds no review.
		d, err := s.CurrentCloseoutDecision("milestone:program/two")
		if err != nil || d.Required {
			t.Fatalf("an exit gate forced the aggregate review under when-justified: %+v %v", d, err)
		}
	})
	t.Run("a backward link to a missing roadmap is unknown", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		os.WriteFile(filepath.Join(root, ".pose/adr/decision.md"), []byte("Governs [the program](../roadmaps/missing.md).\n"), 0o644)
		if d := evaluate(t, s, "roadmap:pair"); d.Known {
			t.Fatalf("an unresolved backward link proved absence: %+v", d)
		}
	})
	t.Run("a milestone approved before adoption stays closed", func(t *testing.T) {
		root, s := inheritedFixture(t)
		ref := "milestone:program/core"
		approveCore(t, s, reviewFixtureNow.Add(2*time.Minute))
		setCloseoutPolicy(t, root, "always", "always")
		eval, err := s.ReviewCheck(ref)
		if err != nil || eval.CloseoutReview != nil || !eval.Approved {
			t.Fatalf("a pre-adoption approval was re-decided: %+v %v", eval, err)
		}
		if state, err := s.GetCloseoutState(ref); err != nil || state.CloseRequired {
			t.Fatalf("a historically approved milestone owes a receipt: %+v %v", state, err)
		}
	})
}

// withDistProfiles gives a members fixture the distribution's review profiles,
// so plans and closeout states can be built on it.
func withDistProfiles(t *testing.T, root string) {
	t.Helper()
	os.MkdirAll(filepath.Join(root, ".pose", "review-profiles"), 0o755)
	for _, id := range []string{"spec-closeout", "milestone-integration", "roadmap-outcome"} {
		if err := os.WriteFile(filepath.Join(root, ".pose", "review-profiles", id+".json"), []byte(mustReadDistProfile(t, id)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCloseoutReviewPolicyRoundFive(t *testing.T) {
	t.Run("a milestone approved before adoption is not reached again", func(t *testing.T) {
		root, s := inheritedFixture(t)
		approveCore(t, s, reviewFixtureNow.Add(2*time.Minute))
		setCloseoutPolicy(t, root, "always", "always")
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 0 {
			t.Fatalf("a historically approved milestone is offered for close again: %v %v", reached, err)
		}
		if closed, err := s.closedByReceipt("milestone:program/core"); err != nil || !closed {
			t.Fatalf("the shared terminal projection does not recognise the historical closure: %v %v", closed, err)
		}
	})
	t.Run("a merely reached milestone without a receipt is not closed", func(t *testing.T) {
		_, s := closeoutReviewedFixture(t, "never", "never")
		if closed, err := s.closedByReceipt("milestone:program/core"); err != nil || closed {
			t.Fatalf("a milestone with neither receipt nor historical approval reads closed: %v %v", closed, err)
		}
	})
	t.Run("a receipt does not close a milestone whose consumed outcome is unavailable", func(t *testing.T) {
		root, s := closeoutReviewedFixture(t, "never", "never")
		ref := "milestone:program/core"
		state, err := s.GetCloseoutState(ref)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := s.NewCloseoutReceipt(ref, *state.Review.CloseoutReview, state.Review, state.ScopeDigest, reviewFixtureNow.Add(time.Hour))
		if err != nil || s.PublishCloseoutReceipt(receipt) != nil {
			t.Fatal(err)
		}
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 0 {
			t.Fatalf("the closed milestone is still reached: %v %v", reached, err)
		}
		editRoadmapFixture(t, root, "- specs: backend\n", "- specs: backend\n- consumes: xref:source/roadmap:absent\n")
		if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 1 || reached[0] != ref {
			t.Fatalf("a receipt with an unavailable dependency still reads closed: %v %v", reached, err)
		}
		unbound := Store{Root: root}
		if closed, err := unbound.closedByReceipt(ref); err != nil || closed {
			t.Fatalf("without a bound resolver the dependency edges were not judged: %v %v", closed, err)
		}
	})
	t.Run("the last open member carries a multi-member exit gate", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
		withDistProfiles(t, root)
		editFile(t, filepath.Join(root, ".pose/roadmaps/program.md"), "- specs: a, b\n", "- specs: a, b\n\n**Exit gate:** both parts integrate\n")
		editFile(t, filepath.Join(root, ".pose/specs/b.md"), "status: done", "status: in-progress")
		carries := func(s Store, slug string) bool {
			t.Helper()
			plan, err := s.ReviewPlan("spec:" + slug)
			if err != nil {
				t.Fatal(err)
			}
			for _, criterion := range plan.Criteria {
				if criterion.ID == "milestone-exit-gate-program-two" {
					return criterion.Description == "The exit gate of milestone:program/two holds: both parts integrate" && criterion.Required
				}
			}
			return false
		}
		if !carries(s, "b") {
			t.Fatal("the last open member does not carry the milestone's exit gate")
		}
		editFile(t, filepath.Join(root, ".pose/specs/a.md"), "status: done", "status: in-progress")
		if carries(s, "b") {
			t.Fatal("a member carried the gate while another member is still open")
		}
		legacyRoot, legacy := closeoutMembersFixture(t, "", "")
		withDistProfiles(t, legacyRoot)
		editFile(t, filepath.Join(legacyRoot, ".pose/roadmaps/program.md"), "- specs: a, b\n", "- specs: a, b\n\n**Exit gate:** both parts integrate\n")
		editFile(t, filepath.Join(legacyRoot, ".pose/specs/b.md"), "status: done", "status: in-progress")
		if carries(legacy, "b") {
			t.Fatal("the legacy policy moved a multi-member gate out of the milestone's own review")
		}
	})
	t.Run("an exempt multi-member milestone with an unjudged exit gate does not close", func(t *testing.T) {
		root, s := closeoutMembersFixture(t, "never", "never")
		withDistProfiles(t, root)
		editFile(t, filepath.Join(root, ".pose/roadmaps/program.md"), "- specs: a, b\n", "- specs: a, b\n\n**Exit gate:** both parts integrate\n")
		state, err := s.GetCloseoutState("milestone:program/two")
		if err != nil {
			t.Fatal(err)
		}
		if state.Review.Required || !strings.Contains(strings.Join(state.Blockers, "\n"), "is judged by no approved review") || state.Terminal {
			t.Fatalf("an unjudged exit gate did not block the exempt close: required=%v blockers=%v", state.Review.Required, state.Blockers)
		}
		gateless, err := s.GetCloseoutState("milestone:program/three")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.Join(gateless.Blockers, "\n"), "is judged by no approved review") {
			t.Fatalf("a gateless milestone was blocked for its gate: %v", gateless.Blockers)
		}
	})
	t.Run("an unreadable backward link target is unknown", func(t *testing.T) {
		for name, prepare := range map[string]func(t *testing.T, path string){
			"unreadable": func(t *testing.T, path string) {
				os.WriteFile(path, []byte("---\nslug: other\nstatus: active\n---\n"), 0o644)
				if err := os.Chmod(path, 0); err != nil {
					t.Skip(err)
				}
				if _, err := os.ReadFile(path); err == nil {
					t.Skip("the file stays readable to this user")
				}
			},
		} {
			t.Run(name, func(t *testing.T) {
				root, s := closeoutMembersFixture(t, "when-justified", "when-justified")
				path := filepath.Join(root, ".pose/roadmaps/other.md")
				prepare(t, path)
				defer os.Chmod(path, 0o644)
				os.WriteFile(filepath.Join(root, ".pose/adr/decision.md"), []byte("Governs [the other](../roadmaps/other.md).\n"), 0o644)
				d, err := s.EvaluateCloseoutReview("roadmap:pair")
				if err == nil && d.Known {
					t.Fatalf("an unresolvable backward target proved absence: %+v", d)
				}
			})
		}
	})
}
