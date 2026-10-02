package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Compact delivery-integrity index (spec pose-delivery-integrity-index-compaction).
//
// The in-memory graph is unchanged: every delivery is linked to every passing
// validation result that gates it, which is the shape each gate and MCP tool
// reads. Written out, that product grows with deliveries times results, so the
// index file stores the same facts once:
//
//   - results that share one validation run reference it, and the run's
//     scope_provenance map is stored once;
//   - the results that validate a delivery are one shared validation-set node per
//     distinct set, linked from each delivery with a single validated-by edge;
//   - a delivery path names its validation set instead of listing each result.
//
// Reading expands this back to the exact pairs the writer started from, so no gate
// reaches a different conclusion. The provenance digest and the scoped digests
// hash claims and change sets only, so none of them moves.

// DeliveryIntegrityIndexSchemaVersion is the schema of the index file. The graph a
// caller holds in memory keeps DeliveryIntegritySchemaVersion.
const DeliveryIntegrityIndexSchemaVersion = 2

const (
	impliedChangesEdges    = "changes"
	validationSetPrefix    = "validation-set:"
	validationResultPrefix = "validation-result:"
)

type compactValidationRun struct {
	ID               string            `json:"id"`
	GitHead          string            `json:"git_head,omitempty"`
	GeneratedAt      string            `json:"generated_at,omitempty"`
	ProvenanceDigest string            `json:"provenance_digest,omitempty"`
	Report           string            `json:"report,omitempty"`
	ScopeProvenance  map[string]string `json:"scope_provenance,omitempty"`
}

type compactValidationResult struct {
	ID            string `json:"id"`
	Module        string `json:"module"`
	Check         string `json:"check"`
	EvidenceClass string `json:"evidence_class"`
	Severity      string `json:"severity,omitempty"`
	Outcome       string `json:"outcome"`
	Run           string `json:"run,omitempty"`
}

type compactDeliveryGraph struct {
	SchemaVersion    int    `json:"schema_version"`
	InputDigest      string `json:"input_digest"`
	ProvenanceDigest string `json:"provenance_digest,omitempty"`
	// ImpliedEdges names edge types left out of Edges because they follow from
	// other fields. Today only "changes", which is each change set's observed paths.
	ImpliedEdges      []string                   `json:"implied_edges,omitempty"`
	Nodes             []DeliveryIntegrityNode    `json:"nodes"`
	Edges             []DeliveryIntegrityEdge    `json:"edges"`
	Claims            []ArtifactClaim            `json:"claims"`
	ChangeSets        []ChangeSet                `json:"change_sets"`
	Reverse           map[string][]string        `json:"reverse"`
	Findings          []DeliveryIntegrityFinding `json:"findings"`
	Deliveries        []DeliveryTarget           `json:"deliveries,omitempty"`
	ValidationRuns    []compactValidationRun     `json:"validation_runs,omitempty"`
	ValidationResults []compactValidationResult  `json:"validation_results,omitempty"`
	RoadmapCriteria   []RoadmapCriterion         `json:"roadmap_criteria,omitempty"`
	Paths             map[string][]string        `json:"paths,omitempty"`
	Archivals         []ArchivedFragment         `json:"archivals,omitempty"`
}

// IndexJSON encodes the graph as the schema 2 index file, indented like every
// other index.
func (g DeliveryIntegrityGraph) IndexJSON() ([]byte, error) {
	return json.MarshalIndent(compactDeliveryIndex(g), "", "  ")
}

func edgeLess(a, b DeliveryIntegrityEdge) bool {
	return a.From+"\x00"+a.Type+"\x00"+a.To < b.From+"\x00"+b.Type+"\x00"+b.To
}

// dedupeSortedEdges drops exact repeats from an edge list sorted by edgeLess.
func dedupeSortedEdges(edges []DeliveryIntegrityEdge) []DeliveryIntegrityEdge {
	out := edges[:0]
	for i, edge := range edges {
		if i > 0 && edge == edges[i-1] {
			continue
		}
		out = append(out, edge)
	}
	return out
}

// derivedChangeEdges is every change-set-to-artifact edge the change sets imply.
func derivedChangeEdges(sets []ChangeSet) []DeliveryIntegrityEdge {
	var edges []DeliveryIntegrityEdge
	for _, set := range sets {
		for _, observed := range set.Paths {
			for _, value := range observedPaths(observed) {
				edges = append(edges, DeliveryIntegrityEdge{From: "change-set:" + set.ID, To: "artifact:" + value, Type: impliedChangesEdges})
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edgeLess(edges[i], edges[j]) })
	return dedupeSortedEdges(edges)
}

// changesEdgesAreImplied reports whether the graph's changes edges are exactly the
// ones its change sets imply, which is what makes leaving them out lossless.
func changesEdgesAreImplied(g DeliveryIntegrityGraph) bool {
	actual := []DeliveryIntegrityEdge{}
	for _, edge := range g.Edges {
		if edge.Type == impliedChangesEdges {
			actual = append(actual, edge)
		}
	}
	sort.Slice(actual, func(i, j int) bool { return edgeLess(actual[i], actual[j]) })
	actual = dedupeSortedEdges(actual)
	derived := derivedChangeEdges(g.ChangeSets)
	if len(actual) != len(derived) {
		return false
	}
	for i := range actual {
		if actual[i] != derived[i] {
			return false
		}
	}
	return true
}

func validationSetID(members []string) string {
	sum := sha256.Sum256([]byte(strings.Join(members, "\n")))
	return validationSetPrefix + hex.EncodeToString(sum[:8])
}

func compactValidationRunID(run compactValidationRun) string {
	run.ID = ""
	raw, _ := json.Marshal(run)
	sum := sha256.Sum256(raw)
	return "run-" + hex.EncodeToString(sum[:8])
}

func compactDeliveryIndex(g DeliveryIntegrityGraph) compactDeliveryGraph {
	out := compactDeliveryGraph{
		SchemaVersion: DeliveryIntegrityIndexSchemaVersion, InputDigest: g.InputDigest, ProvenanceDigest: g.ProvenanceDigest,
		Claims: g.Claims, ChangeSets: g.ChangeSets, Reverse: g.Reverse, Findings: g.Findings, Deliveries: g.Deliveries,
		RoadmapCriteria: g.RoadmapCriteria, Archivals: g.Archivals,
	}

	out.Edges = []DeliveryIntegrityEdge{}

	// Results reference one run per distinct run-level tuple.
	runs := map[string]compactValidationRun{}
	for _, result := range g.ValidationResults {
		compact := compactValidationResult{ID: result.ID, Module: result.Module, Check: result.Check, EvidenceClass: result.EvidenceClass, Severity: result.Severity, Outcome: result.Outcome}
		run := compactValidationRun{GitHead: result.GitHead, GeneratedAt: result.GeneratedAt, ProvenanceDigest: result.ProvenanceDigest, Report: result.Report, ScopeProvenance: result.ScopeProvenance}
		if run.GitHead != "" || run.GeneratedAt != "" || run.ProvenanceDigest != "" || run.Report != "" || len(run.ScopeProvenance) > 0 {
			run.ID = compactValidationRunID(run)
			compact.Run = run.ID
			runs[run.ID] = run
		}
		out.ValidationResults = append(out.ValidationResults, compact)
	}
	for _, run := range runs {
		out.ValidationRuns = append(out.ValidationRuns, run)
	}
	sort.Slice(out.ValidationRuns, func(i, j int) bool { return out.ValidationRuns[i].ID < out.ValidationRuns[j].ID })

	// Validation sets: one per distinct sorted membership.
	sets := map[string][]string{}
	register := func(members []string) string {
		sorted := append([]string{}, members...)
		sort.Strings(sorted)
		id := validationSetID(sorted)
		sets[id] = sorted
		return id
	}
	byFrom := map[string][]string{}
	implied := changesEdgesAreImplied(g)
	if implied {
		out.ImpliedEdges = []string{impliedChangesEdges}
	}
	for _, edge := range g.Edges {
		if implied && edge.Type == impliedChangesEdges {
			continue
		}
		if edge.Type == "validated-by" && strings.HasPrefix(edge.To, validationResultPrefix) {
			byFrom[edge.From] = append(byFrom[edge.From], edge.To)
			continue
		}
		out.Edges = append(out.Edges, edge)
	}
	for from, members := range byFrom {
		out.Edges = append(out.Edges, DeliveryIntegrityEdge{From: from, To: register(members), Type: "validated-by"})
	}
	out.Paths = map[string][]string{}
	for key, path := range g.Paths {
		tail := len(path)
		for tail > 0 && strings.HasPrefix(path[tail-1], validationResultPrefix) {
			tail--
		}
		if tail == len(path) {
			out.Paths[key] = path
			continue
		}
		compact := append(append([]string{}, path[:tail]...), register(path[tail:]))
		out.Paths[key] = compact
	}
	if len(g.Paths) == 0 {
		out.Paths = g.Paths
	}

	// The graph's own nodes keep their order; set nodes follow, sorted among
	// themselves, so expanding the index gives back the order it was written from.
	out.Nodes = append([]DeliveryIntegrityNode{}, g.Nodes...)
	setIDs := make([]string, 0, len(sets))
	for id := range sets {
		setIDs = append(setIDs, id)
	}
	sort.Strings(setIDs)
	for _, id := range setIDs {
		out.Nodes = append(out.Nodes, DeliveryIntegrityNode{ID: id, Type: "validation-set", Attributes: map[string]string{"size": fmt.Sprint(len(sets[id]))}})
		for _, member := range sets[id] {
			out.Edges = append(out.Edges, DeliveryIntegrityEdge{From: id, To: member, Type: "contains"})
		}
	}
	sort.Slice(out.Edges, func(i, j int) bool { return edgeLess(out.Edges[i], out.Edges[j]) })
	out.Edges = dedupeSortedEdges(out.Edges)
	return out
}

// expandDeliveryIndex rebuilds the in-memory graph from a schema 2 index.
func expandDeliveryIndex(c compactDeliveryGraph) (DeliveryIntegrityGraph, error) {
	g := DeliveryIntegrityGraph{
		SchemaVersion: DeliveryIntegritySchemaVersion, InputDigest: c.InputDigest, ProvenanceDigest: c.ProvenanceDigest,
		Claims: c.Claims, ChangeSets: c.ChangeSets, Reverse: c.Reverse, Findings: c.Findings, Deliveries: c.Deliveries,
		RoadmapCriteria: c.RoadmapCriteria, Archivals: c.Archivals,
	}
	runs := map[string]compactValidationRun{}
	for _, run := range c.ValidationRuns {
		runs[run.ID] = run
	}
	for _, result := range c.ValidationResults {
		expanded := DeliveryValidationResult{ID: result.ID, Module: result.Module, Check: result.Check, EvidenceClass: result.EvidenceClass, Severity: result.Severity, Outcome: result.Outcome}
		if result.Run != "" {
			run, ok := runs[result.Run]
			if !ok {
				return DeliveryIntegrityGraph{}, fmt.Errorf("validation result %s references unknown run %s", result.ID, result.Run)
			}
			expanded.GitHead, expanded.GeneratedAt, expanded.ProvenanceDigest, expanded.Report = run.GitHead, run.GeneratedAt, run.ProvenanceDigest, run.Report
			if len(run.ScopeProvenance) > 0 {
				// A copy per result, so a caller that writes one result's map cannot
				// change every result that shares the run.
				expanded.ScopeProvenance = make(map[string]string, len(run.ScopeProvenance))
				for spec, digest := range run.ScopeProvenance {
					expanded.ScopeProvenance[spec] = digest
				}
			}
		}
		g.ValidationResults = append(g.ValidationResults, expanded)
	}

	members := map[string][]string{}
	for _, edge := range c.Edges {
		if edge.Type == "contains" && strings.HasPrefix(edge.From, validationSetPrefix) {
			members[edge.From] = append(members[edge.From], edge.To)
		}
	}
	for id := range members {
		sort.Strings(members[id])
	}
	for _, node := range c.Nodes {
		if node.Type != "validation-set" {
			g.Nodes = append(g.Nodes, node)
		}
	}
	for _, edge := range c.Edges {
		switch {
		case edge.Type == "contains" && strings.HasPrefix(edge.From, validationSetPrefix):
			continue
		case edge.Type == "validated-by" && strings.HasPrefix(edge.To, validationSetPrefix):
			list, ok := members[edge.To]
			if !ok {
				return DeliveryIntegrityGraph{}, fmt.Errorf("edge from %s references unknown validation set %s", edge.From, edge.To)
			}
			for _, member := range list {
				g.Edges = append(g.Edges, DeliveryIntegrityEdge{From: edge.From, To: member, Type: "validated-by"})
			}
		default:
			g.Edges = append(g.Edges, edge)
		}
	}
	for _, implied := range c.ImpliedEdges {
		if implied != impliedChangesEdges {
			return DeliveryIntegrityGraph{}, fmt.Errorf("unknown implied edge type %q", implied)
		}
		g.Edges = append(g.Edges, derivedChangeEdges(g.ChangeSets)...)
	}
	sort.Slice(g.Edges, func(i, j int) bool { return edgeLess(g.Edges[i], g.Edges[j]) })
	g.Edges = dedupeSortedEdges(g.Edges)

	if c.Paths != nil {
		g.Paths = make(map[string][]string, len(c.Paths))
		for key, path := range c.Paths {
			expanded := make([]string, 0, len(path))
			for _, element := range path {
				if !strings.HasPrefix(element, validationSetPrefix) {
					expanded = append(expanded, element)
					continue
				}
				list, ok := members[element]
				if !ok {
					return DeliveryIntegrityGraph{}, fmt.Errorf("path %s references unknown validation set %s", key, element)
				}
				expanded = append(expanded, list...)
			}
			g.Paths[key] = expanded
		}
	}
	return g, nil
}
