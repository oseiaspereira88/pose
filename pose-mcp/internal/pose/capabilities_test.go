package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validAssessment = `---
schema_version: 1
assessed_at: 2026-07-21
baseline_commit: 38a248d
method: local source inspection
---

# Capability assessment

## Mechanism: spec-lifecycle-closeout
- title: Spec lifecycle and closeout
- score: 5
- target: 5
- evidence: spec:demo-spec, doc:README.md, commit:38a248d, check:go test ./..., url:https://example.com
- gaps: none named

Prose commentary that the parser must ignore.

## Mechanism: operational-knowledge
- title: Operational knowledge
- score: 3
- target: 5
- evidence: knowledge:demo-note
- gaps: RBAC mapping open; retrieval is lexical
`

func capabilityFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(".pose/specs/demo-spec/spec.md", "---\nslug: demo-spec\nstatus: done\n---\n\n# Spec: demo-spec\n")
	mustWrite(".pose/knowledge/2026-07-21-note-demo-note.md", "---\nslug: demo-note\ntype: note\nstatus: active\n---\n\nbody\n")
	mustWrite("README.md", "# demo\n")
	mustWrite(".pose/capabilities/assessment.md", validAssessment)
	return root
}

func TestParseCapabilityAssessmentValid(t *testing.T) {
	assessment, err := ParseCapabilityAssessment(validAssessment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assessment.Mechanisms) != 2 {
		t.Fatalf("want 2 mechanisms, got %d", len(assessment.Mechanisms))
	}
	first := assessment.Mechanisms[0]
	if first.ID != "spec-lifecycle-closeout" || first.Score != 5 || first.Target != 5 {
		t.Fatalf("first mechanism parsed wrong: %+v", first)
	}
	if len(first.Evidence) != 5 {
		t.Fatalf("want 5 evidence refs, got %d: %v", len(first.Evidence), first.Evidence)
	}
	second := assessment.Mechanisms[1]
	if len(second.Gaps) != 2 {
		t.Fatalf("gaps must split on semicolons: %v", second.Gaps)
	}
}

func TestParseCapabilityAssessmentStructuralErrors(t *testing.T) {
	cases := map[string]string{
		"missing schema": strings.Replace(validAssessment, "schema_version: 1\n", "", 1),
		"future schema":  strings.Replace(validAssessment, "schema_version: 1", "schema_version: 99", 1),
		"bad date":       strings.Replace(validAssessment, "2026-07-21", "21/07/2026", 1),
		"bad commit":     strings.Replace(validAssessment, "baseline_commit: 38a248d", "baseline_commit: ZZZ", 1),
		"score range":    strings.Replace(validAssessment, "- score: 5", "- score: 9", 1),
		"missing title":  strings.Replace(validAssessment, "- title: Spec lifecycle and closeout\n", "", 1),
		"duplicate id":   strings.Replace(validAssessment, "Mechanism: operational-knowledge", "Mechanism: spec-lifecycle-closeout", 1),
	}
	for name, content := range cases {
		if _, err := ParseCapabilityAssessment(content); err == nil {
			t.Errorf("%s: expected a parse error, got none", name)
		}
	}
}

func TestValidateCapabilityEvidenceResolvesAndFails(t *testing.T) {
	root := capabilityFixtureRoot(t)
	store := Store{Root: root}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		t.Fatal(err)
	}
	if issues := store.ValidateCapabilityEvidence(assessment); len(issues) != 0 {
		t.Fatalf("valid fixture must resolve cleanly, got: %v", issues)
	}

	broken := strings.Replace(validAssessment, "spec:demo-spec", "spec:ghost-spec", 1)
	broken = strings.Replace(broken, "knowledge:demo-note", "knowledge:ghost-note", 1)
	broken = strings.Replace(broken, "url:https://example.com", "url:http://insecure", 1)
	assessmentBroken, err := ParseCapabilityAssessment(broken)
	if err != nil {
		t.Fatal(err)
	}
	issues := store.ValidateCapabilityEvidence(assessmentBroken)
	if len(issues) != 3 {
		t.Fatalf("want 3 nominal issues, got %d: %v", len(issues), issues)
	}
	for _, fragment := range []string{"ghost-spec", "ghost-note", "https://"} {
		found := false
		for _, issue := range issues {
			if strings.Contains(issue, fragment) {
				found = true
			}
		}
		if !found {
			t.Errorf("no issue mentions %q: %v", fragment, issues)
		}
	}
}

func TestValidateCapabilityEvidenceConfinesDocPaths(t *testing.T) {
	root := capabilityFixtureRoot(t)
	store := Store{Root: root}
	traversal := strings.Replace(validAssessment, "doc:README.md", "doc:../outside.md", 1)
	assessment, err := ParseCapabilityAssessment(traversal)
	if err != nil {
		t.Fatal(err)
	}
	issues := store.ValidateCapabilityEvidence(assessment)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "../outside.md") {
			found = true
		}
	}
	if !found {
		t.Fatalf("traversal doc ref must fail resolution, got: %v", issues)
	}
}

func TestCapabilityHistoryLoadAndSupersede(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	lines := []string{
		`{"schema":1,"at":"2026-07-20T10:00:00Z","baseline_commit":"38a248d","content_hash":"aaa","scores":{"m1":{"score":3,"target":5}}}`,
		`{"schema":1,"at":"2026-07-21T10:00:00Z","baseline_commit":"38a248d","content_hash":"bbb","scores":{"m1":{"score":4,"target":5}}}`,
		`{"schema":1,"at":"2026-07-21T11:00:00Z","baseline_commit":"38a248d","content_hash":"ccc","scores":{"m1":{"score":3,"target":5}},"supersedes_ts":"2026-07-21T10:00:00Z"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	events, err := LoadCapabilityHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 raw events, got %d", len(events))
	}
	effective := EffectiveSnapshots(events)
	if len(effective) != 2 {
		t.Fatalf("superseded entry must drop out: got %d effective", len(effective))
	}
	if effective[1].ContentHash != "ccc" {
		t.Fatalf("correction entry must survive, got %q", effective[1].ContentHash)
	}

	if _, err := LoadCapabilityHistory(filepath.Join(dir, "missing.jsonl")); err != nil {
		t.Fatalf("missing history must be empty, not an error: %v", err)
	}
	newer := `{"schema":99,"at":"2026-07-21T12:00:00Z","baseline_commit":"38a248d","content_hash":"ddd","scores":{}}`
	if err := os.WriteFile(path, []byte(newer+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCapabilityHistory(path); err == nil {
		t.Fatal("future snapshot schema must fail loudly")
	}
}

func TestDiffCapabilitySnapshots(t *testing.T) {
	from := CapabilitySnapshot{At: "t1", Scores: map[string]CapabilityScore{
		"up":      {Score: 3, Target: 5},
		"down":    {Score: 4, Target: 5},
		"same":    {Score: 2, Target: 4},
		"gone":    {Score: 1, Target: 3},
		"retires": {Score: 2, Target: 3},
	}}
	to := CapabilitySnapshot{At: "t2", Scores: map[string]CapabilityScore{
		"up":      {Score: 5, Target: 5},
		"down":    {Score: 2, Target: 5},
		"same":    {Score: 2, Target: 4},
		"new":     {Score: 0, Target: 3},
		"retires": {Score: 2, Target: 3, Retired: true},
	}}
	diff := DiffCapabilitySnapshots(from, to)
	if len(diff.Raised) != 1 || diff.Raised[0].ID != "up" || diff.Raised[0].To != 5 {
		t.Fatalf("raised wrong: %+v", diff.Raised)
	}
	if len(diff.Lowered) != 1 || diff.Lowered[0].ID != "down" {
		t.Fatalf("lowered wrong: %+v", diff.Lowered)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "new" {
		t.Fatalf("added wrong: %v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "gone" {
		t.Fatalf("removed wrong: %v", diff.Removed)
	}
	if len(diff.Retired) != 1 || diff.Retired[0] != "retires" {
		t.Fatalf("retired wrong: %v", diff.Retired)
	}
	if len(diff.Stable) != 1 || diff.Stable[0] != "same" {
		t.Fatalf("stable wrong: %v", diff.Stable)
	}
}

func TestRenumberedMechanisms(t *testing.T) {
	latest := CapabilitySnapshot{Scores: map[string]CapabilityScore{
		"kept":         {Score: 3, Target: 5},
		"vanished":     {Score: 2, Target: 4},
		"retired-fine": {Score: 1, Target: 3, Retired: true},
	}}
	current, err := ParseCapabilityAssessment(strings.Replace(validAssessment,
		"Mechanism: spec-lifecycle-closeout", "Mechanism: kept", 1))
	if err != nil {
		t.Fatal(err)
	}
	missing := RenumberedMechanisms(latest, current)
	if len(missing) != 1 || missing[0] != "vanished" {
		t.Fatalf("stable-id contract: want [vanished], got %v", missing)
	}
}
