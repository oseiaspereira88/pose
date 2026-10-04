package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// Spec pose-obligation-contract.

func sampleObligation(category string) Obligation {
	src := NodeRef{Artifact: "xref:proj.pose-dist/spec:storage", Kind: "requirement", ID: "R4"}
	o := Obligation{
		SchemaVersion: ObligationSchemaVersion,
		Project:       "proj.pose-dist",
		Source:        ObligationSource{Producer: "readiness", Ref: src, Detail: "depends_on:schema-v2"},
		Category:      category,
		ReasonCode:    "dependency-not-done",
		Condition:     "spec:schema-v2 reaches done",
		Rule:          "depends_on",
		Recipient:     ObligationActor{Unassigned: true},
		Targets:       []NodeRef{src},
		Effects:       []ObligationEffect{{Phase: PhaseStart, Mode: EffectBlock}},
		Satisfaction:  SatisfactionPending,
		Knowledge:     KnowledgeKnown,
		Waiting:       WaitingArtifact,
		Observation:   ObligationObservation{Freshness: FreshnessCurrent, Coverage: CoverageComplete},
		Message:       "storage waits on schema-v2",
	}
	o.ID = ObligationID(o.Project, o.Source.Producer, o.Source.Ref, o.Rule, o.Source.Detail)
	return o
}

func TestObligationIDIsStableAcrossWordingOrderAndTime(t *testing.T) {
	a := sampleObligation(ObligationDependency)
	b := a
	b.Message = "completely different wording, another locale"
	b.Condition = "rephrased"
	b.Observation.SourceRevision = "a-later-revision"
	if ObligationID(b.Project, b.Source.Producer, b.Source.Ref, b.Rule, b.Source.Detail) != a.ID {
		t.Fatal("the id changed with wording or revision")
	}
	if ObligationID(a.Project, a.Source.Producer, a.Source.Ref, a.Rule, "depends_on:other") == a.ID {
		t.Fatal("two logically distinct obligations share an id")
	}
}

func TestSameLocalNodeInDifferentSpecsOrProjectsNeverCollides(t *testing.T) {
	r4 := func(project, slug string) NodeRef {
		return QualifyNodeRef(project, ArtifactRef{Kind: "spec", Slug: slug}, "requirement", "R4")
	}
	ids := map[string]string{}
	for _, ref := range []NodeRef{r4("proj.pose-dist", "storage"), r4("proj.pose-dist", "billing"), r4("proj.harne8", "storage")} {
		id := ObligationID("proj.x", "readiness", ref, "depends_on", "d")
		if prior, dup := ids[id]; dup {
			t.Fatalf("%s and %s collide on %s", prior, ref, id)
		}
		ids[id] = ref.String()
	}
	if _, err := ParseNodeRef("spec:storage#requirement:R4"); err == nil {
		t.Fatal("an unqualified node reference was accepted")
	}
	ref, err := ParseNodeRef("xref:proj.harne8/spec:storage#requirement:R4")
	if err != nil || ref.Artifact != "xref:proj.harne8/spec:storage" || ref.Kind != "requirement" || ref.ID != "R4" {
		t.Fatalf("qualified node reference: %+v %v", ref, err)
	}
	for _, bad := range []string{"xref:proj.x/spec:../etc#requirement:R4", "xref:proj.x/spec:s#bogus:R4", "xref:proj.x/spec:s#requirement:", "xref:proj.x/spec:s#requirement:a b"} {
		if _, err := ParseNodeRef(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestUnknownIsNeverAResolution(t *testing.T) {
	for _, satisfaction := range []string{SatisfactionSatisfied, SatisfactionWaived, SatisfactionCancelled} {
		o := sampleObligation(ObligationEvidence)
		o.Knowledge = KnowledgeUnknown
		o.Satisfaction = satisfaction
		if err := ValidateObligation(o); err == nil {
			t.Errorf("unknown knowledge carried satisfaction %q", satisfaction)
		}
	}
	o := sampleObligation(ObligationEvidence)
	o.Knowledge = KnowledgeUnknown
	if err := ValidateObligation(o); err != nil {
		t.Fatalf("an unknown pending obligation is valid: %v", err)
	}
	if !o.Restricts(PhaseStart) {
		t.Fatal("an unknown pending obligation must still restrict its phase")
	}
}

func TestEveryCategoryValidatesAndRestrictsOnlyItsPhase(t *testing.T) {
	for _, category := range ObligationEnums()["category"] {
		o := sampleObligation(category)
		o.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}, {Phase: PhaseRelease, Mode: EffectAdvisory}}
		if err := ValidateObligation(o); err != nil {
			t.Fatalf("%s: %v", category, err)
		}
		if o.Restricts(PhaseExecution) || o.Restricts(PhaseRelease) || !o.Restricts(PhaseCloseout) {
			t.Fatalf("%s: effect leaked to another phase or an advisory restricted", category)
		}
		o.Satisfaction = SatisfactionSatisfied
		if o.Restricts(PhaseCloseout) {
			t.Fatalf("%s: a satisfied obligation still restricts", category)
		}
	}
}

func TestLegacyOpaqueObligationInventsNoActorOrTarget(t *testing.T) {
	o := sampleObligation(ObligationReconciliation)
	o.Observation.Coverage = CoverageLegacyOpaque
	o.Recipient = ObligationActor{Role: "maintainer"}
	if err := ValidateObligation(o); err == nil {
		t.Fatal("a legacy-opaque blocker gained an actor")
	}
	o.Recipient = ObligationActor{Unassigned: true}
	if err := ValidateObligation(o); err == nil {
		t.Fatal("a legacy-opaque blocker kept a target it cannot know")
	}
	o.Targets = nil
	if err := ValidateObligation(o); err != nil {
		t.Fatalf("a legacy-opaque blocker with origin only is valid: %v", err)
	}
}

func TestObligationSchemaMatchesTheEngineVocabulary(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v1", "obligation.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	enumOf := func(raw json.RawMessage, path ...string) []string {
		var node map[string]json.RawMessage
		_ = json.Unmarshal(raw, &node)
		for _, key := range path {
			_ = json.Unmarshal(node[key], &node)
		}
		var values []string
		_ = json.Unmarshal(node["enum"], &values)
		sort.Strings(values)
		return values
	}
	engine := ObligationEnums()
	checks := map[string][]string{
		"category":     enumOf(schema.Properties["category"]),
		"satisfaction": enumOf(schema.Properties["satisfaction"]),
		"knowledge":    enumOf(schema.Properties["knowledge"]),
		"waiting":      enumOf(schema.Properties["waiting"]),
		"freshness":    enumOf(schema.Properties["observation"], "properties", "freshness"),
		"coverage":     enumOf(schema.Properties["observation"], "properties", "coverage"),
		"phase":        enumOf(schema.Properties["effects"], "items", "properties", "phase"),
		"mode":         enumOf(schema.Properties["effects"], "items", "properties", "mode"),
	}
	for name, schemaValues := range checks {
		if !reflect.DeepEqual(schemaValues, engine[name]) {
			t.Errorf("%s: schema %v, engine %v", name, schemaValues, engine[name])
		}
	}
}
