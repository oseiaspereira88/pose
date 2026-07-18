package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readinessStore builds a hermetic tree exercising every readiness branch:
// done/draft deps, typed refs, missing refs and terminal/blocked statuses.
func readinessStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/base/spec.md", `---
slug: base
status: done
completed_at: 2026-07-01
---

# Spec: base
`)
	write(".pose/specs/pending/spec.md", `---
slug: pending
status: in-progress
---

# Spec: pending
`)
	write(".pose/specs/ready-one/spec.md", `---
slug: ready-one
status: draft
depends_on: base
priority: 2
---

# Spec: ready-one
`)
	write(".pose/specs/waiting/spec.md", `---
slug: waiting
status: draft
depends_on: base, pending, missing-spec, milestone:v2/c4, roadmap:v1
---

# Spec: waiting
`)
	write(".pose/specs/finished/spec.md", `---
slug: finished
status: done
completed_at: 2026-07-02
depends_on: base
---

# Spec: finished
`)
	write(".pose/specs/stuck/spec.md", `---
slug: stuck
status: blocked
---

# Spec: stuck
`)
	return Store{Root: root}
}

func TestSpecReadinessReady(t *testing.T) {
	s := readinessStore(t)
	r, err := s.SpecReadiness("ready-one")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	if !r.Ready {
		t.Fatalf("ready-one should be ready; waiting_on=%v reason=%q", r.WaitingOn, r.Reason)
	}
	if len(r.WaitingOn) != 0 {
		t.Errorf("waiting_on should be empty, got %v", r.WaitingOn)
	}
}

func TestSpecReadinessWaiting(t *testing.T) {
	s := readinessStore(t)
	r, err := s.SpecReadiness("waiting")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	if r.Ready {
		t.Fatal("waiting should not be ready")
	}
	reasons := map[string]string{}
	for _, w := range r.WaitingOn {
		reasons[w.Ref] = w.Reason
	}
	if _, ok := reasons["base"]; ok {
		t.Error("done dependency 'base' must not appear in waiting_on")
	}
	for _, ref := range []string{"pending", "missing-spec", "milestone:v2/c4", "roadmap:v1"} {
		if reasons[ref] == "" {
			t.Errorf("expected %q in waiting_on with a reason, got %v", ref, r.WaitingOn)
		}
	}
}

func TestSpecReadinessTerminalAndBlocked(t *testing.T) {
	s := readinessStore(t)
	for slug, wantReady := range map[string]bool{"finished": false, "stuck": false} {
		r, err := s.SpecReadiness(slug)
		if err != nil {
			t.Fatalf("SpecReadiness(%s): %v", slug, err)
		}
		if r.Ready != wantReady {
			t.Errorf("%s: ready=%v, want %v (reason=%q)", slug, r.Ready, wantReady, r.Reason)
		}
		if r.Reason == "" {
			t.Errorf("%s: terminal/blocked readiness must carry a reason", slug)
		}
	}
}

func TestSpecReadinessUnknownSpec(t *testing.T) {
	s := readinessStore(t)
	if _, err := s.SpecReadiness("nope"); err == nil {
		t.Fatal("expected error for unknown spec")
	}
}

func TestParseSpecDependsOnAndPriority(t *testing.T) {
	s := readinessStore(t)
	sp, err := s.GetSpec("ready-one")
	if err != nil {
		t.Fatalf("GetSpec: %v", err)
	}
	if len(sp.DependsOn) != 1 || sp.DependsOn[0] != "base" {
		t.Errorf("DependsOn = %v, want [base]", sp.DependsOn)
	}
	if sp.Priority == nil || *sp.Priority != 2 {
		t.Errorf("Priority = %v, want 2", sp.Priority)
	}
	multi, err := s.GetSpec("waiting")
	if err != nil {
		t.Fatalf("GetSpec: %v", err)
	}
	if len(multi.DependsOn) != 5 {
		t.Errorf("DependsOn = %v, want 5 refs preserved verbatim", multi.DependsOn)
	}
}

func TestParseDependsOnFormats(t *testing.T) {
	cases := map[string][]string{
		"":                       nil,
		"a":                      {"a"},
		"a, b":                   {"a", "b"},
		"[a, b]":                 {"a", "b"},
		" [ a , milestone:r/m ]": {"a", "milestone:r/m"},
	}
	for input, want := range cases {
		got := parseDependsOn(input)
		if len(got) != len(want) {
			t.Errorf("parseDependsOn(%q) = %v, want %v", input, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("parseDependsOn(%q)[%d] = %q, want %q", input, i, got[i], want[i])
			}
		}
	}
}

// --- Definition of Ready no readiness (pose-definition-of-ready) ---

func dorStore(t *testing.T, withPolicy bool) Store {
	t.Helper()
	s := readinessStore(t)
	write := func(rel, content string) {
		path := filepath.Join(s.Root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if withPolicy {
		write(".pose/policy/dor.json", `{"adopted_at":"2026-07-11"}`)
	}
	write(".pose/specs/new-no-criteria/spec.md", `---
slug: new-no-criteria
status: draft
created_at: 2026-07-12
---

# Spec: new-no-criteria

## 2. Requirements
- sem ids
`)
	write(".pose/specs/new-with-criteria/spec.md", `---
slug: new-with-criteria
status: draft
created_at: 2026-07-12
---

# Spec: new-with-criteria

## 2. Requirements
- R1: comportamento verificável.
`)
	write(".pose/specs/legacy-old/spec.md", `---
slug: legacy-old
status: draft
created_at: 2026-06-01
---

# Spec: legacy-old

## 2. Requirements
- sem ids, mas anterior ao cutoff
`)
	return s
}

func TestSpecReadinessDoRBlocksNewSpecWithoutCriteria(t *testing.T) {
	s := dorStore(t, true)
	r, err := s.SpecReadiness("new-no-criteria")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	if r.Ready {
		t.Fatal("new spec without acceptance criteria IDs must not be ready")
	}
	found := false
	for _, w := range r.WaitingOn {
		if w.Ref == "dor:acceptance-criteria" {
			found = true
		}
	}
	if !found {
		t.Fatalf("waiting_on = %v, want dor:acceptance-criteria", r.WaitingOn)
	}
}

func TestSpecReadinessDoRPassesWithCriteria(t *testing.T) {
	s := dorStore(t, true)
	r, err := s.SpecReadiness("new-with-criteria")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	if !r.Ready {
		t.Fatalf("spec with criteria IDs should be ready; waiting_on=%v", r.WaitingOn)
	}
}

func TestSpecReadinessDoRExemptsLegacyAndNoPolicy(t *testing.T) {
	s := dorStore(t, true)
	if r, _ := s.SpecReadiness("legacy-old"); !r.Ready {
		t.Fatalf("legacy spec (before cutoff) must be exempt; waiting_on=%v", r.WaitingOn)
	}
	noPolicy := dorStore(t, false)
	if r, _ := noPolicy.SpecReadiness("new-no-criteria"); !r.Ready {
		t.Fatalf("without dor.json the DoR condition must be disabled; waiting_on=%v", r.WaitingOn)
	}
}

// --- Refs tipadas resolvidas contra roadmaps (pose-roadmap-artifact R4) ---

func roadmapStore(t *testing.T) Store {
	t.Helper()
	s := readinessStore(t)
	write := func(rel, content string) {
		path := filepath.Join(s.Root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/roadmaps/serie.md", `---
slug: serie
status: active
depends_on:
---

# Roadmap: serie

## Milestone: m1
- after:
- target_start: 2026-07-01
- target_due: 2026-07-10
- specs: base

## Milestone: m2
- after: m1
- specs: pending
`)
	write(".pose/roadmaps/entregue.md", `---
slug: entregue
status: done
---

# Roadmap: entregue
`)
	write(".pose/specs/gated/spec.md", `---
slug: gated
status: draft
depends_on: milestone:serie/m1, milestone:serie/m2, roadmap:entregue, roadmap:serie
---

# Spec: gated
`)
	return s
}

func TestRoadmapParsing(t *testing.T) {
	s := roadmapStore(t)
	rm, err := s.GetRoadmap("serie")
	if err != nil {
		t.Fatalf("GetRoadmap: %v", err)
	}
	if rm.Status != "active" || len(rm.Milestones) != 2 {
		t.Fatalf("roadmap = %+v, want active with 2 milestones", rm)
	}
	m1 := rm.Milestones[0]
	if m1.ID != "m1" || m1.TargetDue != "2026-07-10" || len(m1.Specs) != 1 || m1.Specs[0] != "base" {
		t.Fatalf("m1 = %+v", m1)
	}
	if rm.Milestones[1].After[0] != "m1" {
		t.Fatalf("m2.after = %v, want [m1]", rm.Milestones[1].After)
	}
	list, err := s.ListRoadmaps()
	if err != nil || len(list) != 2 {
		t.Fatalf("ListRoadmaps = %v/%v, want 2", list, err)
	}
}

func TestSpecReadinessResolvesTypedRefs(t *testing.T) {
	s := roadmapStore(t)
	r, err := s.SpecReadiness("gated")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	reasons := map[string]string{}
	for _, w := range r.WaitingOn {
		reasons[w.Ref] = w.Reason
	}
	// m1: única spec (base) está done → satisfeito, não aparece.
	if _, ok := reasons["milestone:serie/m1"]; ok {
		t.Fatalf("m1 satisfeito não deveria aparecer: %v", r.WaitingOn)
	}
	// roadmap done → satisfeito.
	if _, ok := reasons["roadmap:entregue"]; ok {
		t.Fatalf("roadmap done não deveria aparecer: %v", r.WaitingOn)
	}
	// m2: spec pending está in-progress → insatisfeito com razão nominal.
	if !strings.Contains(reasons["milestone:serie/m2"], "pending (in-progress)") {
		t.Fatalf("m2 reason = %q, want pending listed", reasons["milestone:serie/m2"])
	}
	// roadmap ativo → insatisfeito.
	if !strings.Contains(reasons["roadmap:serie"], `status is "active"`) {
		t.Fatalf("roadmap:serie reason = %q", reasons["roadmap:serie"])
	}
	if r.Ready {
		t.Fatal("gated não deveria estar ready")
	}
}

func TestSpecReadinessTypedRefNotFoundFailsClosed(t *testing.T) {
	s := roadmapStore(t)
	write := func(rel, content string) {
		path := filepath.Join(s.Root, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	write(".pose/specs/broken-ref/spec.md", "---\nslug: broken-ref\nstatus: draft\ndepends_on: roadmap:ghost, milestone:serie/m9\n---\n\n# Spec: broken-ref\n")
	r, err := s.SpecReadiness("broken-ref")
	if err != nil {
		t.Fatalf("SpecReadiness: %v", err)
	}
	if r.Ready || len(r.WaitingOn) != 2 {
		t.Fatalf("refs quebradas devem manter fail-closed: %+v", r)
	}
}
