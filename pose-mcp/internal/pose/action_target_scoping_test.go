package pose

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func actionTargetSpec(t *testing.T, s Store, slug, status, text string) {
	t.Helper()
	p := filepath.Join(s.Root, ".pose/specs/2026-10-21-"+slug+".md")
	if err := os.WriteFile(p, []byte("---\nslug: "+slug+"\nstatus: "+status+"\n---\n# Spec: "+slug+"\n## 2. Requirements\n- R4: "+text+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
func actionTargetItems(t *testing.T, s Store, scope string) []Obligation {
	t.Helper()
	r, e := s.ProjectObligations(ObligationQuery{Scope: scope})
	if e != nil {
		t.Fatal(e)
	}
	var out []Obligation
	for _, o := range r.Obligations {
		if o.Source.Producer == "action-requests" {
			out = append(out, o)
		}
	}
	return out
}
func TestActionTargetCrossArtifactEffectsAndLegacyEmptyScope(t *testing.T) {
	s := actionFixture(t)
	actionTargetSpec(t, s, "beta", "draft", "Beta material")
	project := s.CurrentObligationSnapshot().Project
	r := decisionRequest()
	beta := NodeRef{Artifact: "xref:" + project + "/spec:beta"}
	r.Targets = []NodeRef{beta}
	r.Effects = nil
	for _, phase := range []string{PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease} {
		r.Effects = append(r.Effects, ObligationEffect{Phase: phase, Mode: EffectBlock, Scope: []NodeRef{beta}})
	}
	if _, e := s.OpenActionRequest(r, time.Now()); e != nil {
		t.Fatal(e)
	}
	for _, phase := range []string{PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease} {
		origin, e := s.GovernedEffectRestrictions(phase, []string{"storage"})
		if e != nil || len(origin) != 0 {
			t.Fatalf("origin blocked for %s: %+v %v", phase, origin, e)
		}
		target, e := s.GovernedEffectRestrictions(phase, []string{"beta"})
		if e != nil || len(target) != 1 {
			t.Fatalf("destination missing for %s: %+v %v", phase, target, e)
		}
	}
	if got := actionTargetItems(t, s, "spec:beta"); len(got) != 1 {
		t.Fatalf("target query: %+v", got)
	}
	phases, e := s.SpecPhaseReadiness("beta")
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range phases.Phases {
		if len(p.Restricting) > 0 && p.State != PhaseRestricted {
			t.Fatalf("whole artifact reported partial: %+v", p)
		}
	}
	r.Effects = []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock}}
	if _, e := s.OpenActionRequest(r, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, e := s.GovernedEffectRestrictions(PhaseExecution, []string{"storage"})
	if e != nil || len(got) != 1 {
		t.Fatalf("legacy empty scope: %+v %v", got, e)
	}
	got, e = s.GovernedEffectRestrictions(PhaseExecution, []string{"beta"})
	if e != nil || len(got) != 1 {
		t.Fatalf("target inferred a second blocker: %+v %v", got, e)
	}
}
func TestActionTargetSubjectUsesExactArtifactKindAndID(t *testing.T) {
	s := actionFixture(t)
	actionTargetSpec(t, s, "beta", "draft", "Distinct beta material")
	project := s.CurrentObligationSnapshot().Project
	r := decisionRequest()
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "xref:" + project + "/spec:beta", Kind: "requirement", ID: "R4"}}
	prepared, e := s.PrepareActionRequest(r, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	beta, _ := s.GetSpec("beta")
	want := ProjectContractNodes("beta", beta.Body).Nodes[0].Hash
	if prepared.Subject.Digest != want {
		t.Fatalf("subject bound origin: got %s want %s", prepared.Subject.Digest, want)
	}
	if got := s.currentSubjectDigest(prepared); got != want {
		t.Fatalf("recomputation lost identity: %s", got)
	}
	for _, kind := range []string{"decision", "assumption"} {
		r.Subject.Node.Kind = kind
		if _, e := s.PrepareActionRequest(r, time.Now()); e == nil {
			t.Errorf("accepted %s:R4", kind)
		}
	}
	r.Subject.Node.Kind = "requirement"
	r.Subject.Node.ID = "R404"
	if _, e := s.PrepareActionRequest(r, time.Now()); e == nil {
		t.Fatal("missing subject accepted")
	}
	r.Subject = nil
	r.Targets = []NodeRef{{Artifact: "self", Kind: "requirement", ID: "R404"}}
	if _, e := s.PrepareActionRequest(r, time.Now()); e == nil {
		t.Fatal("missing material target accepted")
	}
}
func TestActionTargetTerminalOriginStillVisible(t *testing.T) {
	s := actionFixture(t)
	if _, e := s.OpenActionRequest(decisionRequest(), time.Now()); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(s.Root, ".pose/specs/2026-10-04-storage.md")
	raw, _ := os.ReadFile(p)
	if e := os.WriteFile(p, []byte(replaceOnce(string(raw), "status: in-progress", "status: done")), 0644); e != nil {
		t.Fatal(e)
	}
	if got := actionTargetItems(t, s, ""); len(got) != 1 {
		t.Fatalf("terminal origin hid open request: %+v", got)
	}
}

func actionTargetRoadmap(t *testing.T, s Store, slug, members string) {
	t.Helper()
	qualifiedFile(t, s.Root, "roadmaps/"+slug+".md", "---\nslug: "+slug+"\nstatus: in-progress\n---\n# Roadmap\n## Milestone: first\n- specs: "+members+"\n")
}
func TestActionTargetRoadmapMembershipIsCurrentAndNeverGlobal(t *testing.T) {
	for _, kind := range []string{"roadmap:", "milestone:"} {
		t.Run(kind, func(t *testing.T) {
			s := actionFixture(t)
			actionTargetSpec(t, s, "beta", "draft", "Beta")
			actionTargetRoadmap(t, s, "one", "storage")
			actionTargetRoadmap(t, s, "two", "beta")
			r := decisionRequest()
			r.Origin = kind + "one"
			if kind == "milestone:" {
				r.Origin += "/first"
			}
			r.Targets = []NodeRef{{Artifact: "self"}}
			r.Effects = []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}}
			if _, e := s.OpenActionRequest(r, time.Now()); e != nil {
				t.Fatal(e)
			}
			for _, tc := range []struct {
				slug string
				want int
			}{{"storage", 1}, {"beta", 0}} {
				items, e := s.GovernedEffectRestrictions(PhaseStart, []string{tc.slug})
				if e != nil || len(items) != tc.want {
					t.Fatalf("%s: %+v %v", tc.slug, items, e)
				}
			}
			if got := actionTargetItems(t, s, "roadmap:two"); len(got) != 0 {
				t.Fatalf("disjoint roadmap leaked: %+v", got)
			}
			actionTargetRoadmap(t, s, "one", "beta")
			if got, e := s.GovernedEffectRestrictions(PhaseStart, []string{"storage"}); e != nil || len(got) != 0 {
				t.Fatalf("stale membership: %+v %v", got, e)
			}
			if got, e := s.GovernedEffectRestrictions(PhaseStart, []string{"beta"}); e != nil || len(got) != 1 {
				t.Fatalf("new membership: %+v %v", got, e)
			}
			if e := os.Remove(filepath.Join(s.Root, ".pose/roadmaps/one.md")); e != nil {
				t.Fatal(e)
			}
			report, e := s.ProjectObligations(ObligationQuery{})
			if e != nil {
				t.Fatal(e)
			}
			if coverageOf(report, "action-requests").State != CoverageStateIncomplete {
				t.Fatalf("missing membership hidden: %+v", report.Coverage)
			}
			if got, e := s.GovernedEffectRestrictions(PhaseStart, []string{"beta"}); e != nil || len(got) != 0 {
				t.Fatalf("unknown membership invented blocker: %+v %v", got, e)
			}
		})
	}
}
func TestActionTargetMaterialBindingAndAdoptionBoundary(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind = ActionApproval
	r.Options = nil
	r.Recommend = ""
	if _, e := s.PrepareActionRequest(r, time.Now()); e != nil {
		t.Fatalf("policy-off legacy behavior changed: %v", e)
	}
	adoptAgencyReadiness(t, s, "")
	for _, kind := range []string{ActionApproval, ActionAcceptance} {
		r.Kind = kind
		if _, e := s.PrepareActionRequest(r, time.Now()); e == nil {
			t.Fatalf("adopted %s omitted material subject", kind)
		}
	}
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "self", Kind: "requirement", ID: "R4"}}
	v, e := s.OpenActionRequest(r, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ResolveActionRequest(answer(v, "human:maintainer", "accept", "binding"), time.Now()); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(s.Root, ".pose/specs/2026-10-04-storage.md")
	raw, _ := os.ReadFile(p)
	if e = os.WriteFile(p, []byte(replaceOnce(string(raw), "status: in-progress", "status: done")), 0644); e != nil {
		t.Fatal(e)
	}
	raw, _ = os.ReadFile(p)
	if e = os.WriteFile(p, []byte(replaceOnce(string(raw), "Keep reading schema v1 records.", "Changed material.")), 0644); e != nil {
		t.Fatal(e)
	}
	items := actionTargetItems(t, s, "")
	if len(items) != 1 || items[0].Satisfaction == SatisfactionSatisfied || items[0].ReasonCode != "action-invalidated" {
		t.Fatalf("terminal changed subject satisfied: %+v", items)
	}
	r.Subject.Path = "README.md"
	if _, e = s.PrepareActionRequest(r, time.Now()); e == nil {
		t.Fatal("ambiguous subject accepted")
	}
}
func TestActionTargetForeignSubjectUsesAuthorizedResolver(t *testing.T) {
	s := actionFixture(t)
	foreign := actionFixture(t)
	actionTargetSpec(t, foreign, "beta", "draft", "Foreign beta content")
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"remote": foreign.Root}}), Authorize: func(project string) bool { return project == "remote" }}
	s.FederatedResolver = &resolver
	r := decisionRequest()
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "xref:remote/spec:beta", Kind: "requirement", ID: "R4"}}
	p, e := s.PrepareActionRequest(r, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if s.currentSubjectDigest(p) != p.Subject.Digest {
		t.Fatal("foreign identity lost on recomputation")
	}
	resolver.Authorize = func(string) bool { return false }
	if _, e := s.PrepareActionRequest(r, time.Now()); e == nil {
		t.Fatal("foreign authorization bypassed")
	}
	if got := s.currentSubjectDigest(p); got != "" {
		t.Fatal("unreadable foreign subject fell back to local content")
	}
}

func TestActionTargetActualGatesRecomposeCrossDestinationAtWrite(t *testing.T) {
	s := actionFixture(t)
	actionTargetSpec(t, s, "beta", "draft", "Beta")
	adoptAgencyReadiness(t, s, `,"atomic_start_version":1`)
	plan, e := s.PreviewStart("beta")
	if e != nil {
		t.Fatal(e)
	}
	project := s.CurrentObligationSnapshot().Project
	r := closeoutRequest()
	r.Targets = []NodeRef{{Artifact: "xref:" + project + "/spec:beta"}}
	for i := range r.Effects {
		r.Effects[i].Scope = append([]NodeRef{}, r.Targets...)
	}
	if _, e = s.OpenActionRequest(r, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ApplyStart(plan, plan.Digest, nil); e == nil {
		t.Fatal("stale preview authorized cross-destination start")
	}
	fresh, e := s.PreviewStart("beta")
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, w := range fresh.WaitingOn {
		found = found || w.Code == "action-request-pending"
	}
	if !found || fresh.Ready {
		t.Fatalf("start not restricted: %+v", fresh)
	}
	for _, tc := range []struct {
		slug string
		want bool
	}{{"storage", false}, {"beta", true}} {
		state, e := s.GetCloseoutState("spec:" + tc.slug)
		if e != nil {
			t.Fatal(e)
		}
		if hasDiagnostic(state, "action-request-pending") != tc.want {
			t.Fatalf("closeout %s: %+v", tc.slug, state.Diagnostics)
		}
		diagnostics, e := s.ReleaseGovernedRefusal([]string{tc.slug})
		if e != nil || (len(diagnostics) > 0) != tc.want {
			t.Fatalf("release freeze %s: %+v %v", tc.slug, diagnostics, e)
		}
	}
}
func TestActionTargetMixedMembershipKeepsKnownLocalRestriction(t *testing.T) {
	s := actionFixture(t)
	actionTargetSpec(t, s, "beta", "draft", "Beta")
	actionTargetRoadmap(t, s, "mixed", "storage, xref:unknown/spec:foreign")
	r := decisionRequest()
	r.Origin = "roadmap:mixed"
	r.Targets = []NodeRef{{Artifact: "self"}}
	r.Effects = []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}}
	if _, e := s.OpenActionRequest(r, time.Now()); e != nil {
		t.Fatal(e)
	}
	report, e := s.ProjectObligations(ObligationQuery{Scope: "spec:storage"})
	if e != nil {
		t.Fatal(e)
	}
	if coverageOf(report, "action-requests").State != CoverageStateIncomplete {
		t.Fatal("foreign member's authority limit hidden")
	}
	for _, tc := range []struct {
		slug string
		want int
	}{{"storage", 1}, {"beta", 0}} {
		got, e := s.GovernedEffectRestrictions(PhaseStart, []string{tc.slug})
		if e != nil || len(got) != tc.want {
			t.Fatalf("%s: %+v %v", tc.slug, got, e)
		}
	}
}
func TestActionTargetEffectParsingRejectsMalformedOrAmbiguousInputs(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `[]`, `"start:bogus"`, `{"phase":"start","mode":"block","scope":[17]}`, `{"phase":"start","mode":"block","extra":true}`, `{"phase":"start","mode":"block"} {}`} {
		if _, e := ParseActionEffect([]byte(raw)); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
