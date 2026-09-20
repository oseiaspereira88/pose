package cli

import (
	"reflect"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// focusSurfaceGraph takes the graph by value, which reads as a copy and is not
// one: filtering into `graph.Deliveries[:0]` writes through the shared backing
// array, so the caller's graph is silently truncated and a second focus on the
// same graph sees whatever the first left behind.
//
// That aliasing is why the graph could not be built once and reused, and
// rebuilding it per spec is what made `check --strict` spend 600s of its 635s in
// checkDeliveryContracts.
func twoSpecGraph() posemodel.DeliveryIntegrityGraph {
	return posemodel.DeliveryIntegrityGraph{
		Deliveries: []posemodel.DeliveryTarget{
			{Spec: "alpha", Ref: "surface:a", Kind: "surface", ID: "a"},
			{Spec: "beta", Ref: "surface:b", Kind: "surface", ID: "b"},
		},
		Findings: []posemodel.DeliveryIntegrityFinding{
			{Spec: "alpha", Code: "alpha-finding", Severity: "error", Message: "a"},
			{Spec: "beta", Code: "beta-finding", Severity: "error", Message: "b"},
		},
		Paths: map[string][]string{"surface:a": {"a.go"}, "surface:b": {"b.go"}},
	}
}

func TestFocusSurfaceGraphDoesNotConsumeItsInput(t *testing.T) {
	graph := twoSpecGraph()
	before := append([]posemodel.DeliveryTarget{}, graph.Deliveries...)
	beforeFindings := append([]posemodel.DeliveryIntegrityFinding{}, graph.Findings...)

	alpha := focusSurfaceGraph(graph, "alpha")
	if len(alpha.Deliveries) != 1 || alpha.Deliveries[0].Spec != "alpha" {
		t.Fatalf("alpha focus = %+v", alpha.Deliveries)
	}
	if !reflect.DeepEqual(graph.Deliveries, before) {
		t.Errorf("focusing consumed the caller's deliveries: %+v, want %+v", graph.Deliveries, before)
	}
	if !reflect.DeepEqual(graph.Findings, beforeFindings) {
		t.Errorf("focusing consumed the caller's findings: %+v, want %+v", graph.Findings, beforeFindings)
	}

	// The reason the reuse matters: a second focus on the same graph has to see
	// its own spec, not the residue of the first.
	beta := focusSurfaceGraph(graph, "beta")
	if len(beta.Deliveries) != 1 || beta.Deliveries[0].Spec != "beta" {
		t.Fatalf("beta focus after alpha = %+v — the graph was consumed", beta.Deliveries)
	}
	if len(beta.Findings) != 1 || beta.Findings[0].Code != "beta-finding" {
		t.Fatalf("beta findings after alpha = %+v", beta.Findings)
	}
	if len(alpha.Deliveries) != 1 || alpha.Deliveries[0].Spec != "alpha" {
		t.Fatalf("the first focus was rewritten by the second: %+v", alpha.Deliveries)
	}
}

// The blockers of one spec are the same whether the graph is built for it alone
// or shared across specs. This is the property the hoist rests on.
func TestDeliverySpecBlockersFromSharedGraph(t *testing.T) {
	graph := twoSpecGraph()
	for _, slug := range []string{"alpha", "beta", "alpha"} {
		shared := deliverySpecBlockersFromGraph(graph, slug)
		if len(shared) != 1 {
			t.Fatalf("%s: blockers from shared graph = %v, want exactly one", slug, shared)
		}
		want := slug + "-finding"
		if !reflect.DeepEqual(shared, []string{want + ": " + map[string]string{"alpha": "a", "beta": "b"}[slug]}) {
			t.Fatalf("%s: blockers = %v, want the %s finding", slug, shared, slug)
		}
	}
}
