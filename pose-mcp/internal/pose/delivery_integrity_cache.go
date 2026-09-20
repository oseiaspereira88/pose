package pose

import (
	"encoding/json"
	"sync"
)

// The delivery-integrity index is parsed once per distinct content, not once per
// caller.
//
// In this repository the file is 4.5 MB and a single `pose check --strict` read and
// unmarshalled it 528 times: 2.4 GB of JSON for one gate. Measured, the read costs
// 0.44 ms and the parse 10.7 ms, so the parse is 96% of it and the file is the only
// input — identical bytes cannot produce a different graph.
//
// The cache key is the raw bytes themselves rather than a modification time. An
// mtime has second or nanosecond granularity depending on the filesystem, and two
// writes inside one tick would serve a stale graph; comparing the bytes cannot be
// wrong. Reading still happens on every call, which is the 4% this deliberately
// keeps.
//
// Every hit returns a defensive copy. Callers filter this graph in place — one of
// them, `GetDeliveryIntegrity`'s own path filter, wrote through `Findings[:0]` — so
// handing out the cached value would let the first caller truncate the graph for
// every caller after it. That is the same aliasing defect `focusSurfaceGraph` had,
// and a cache is exactly what turns it from a local surprise into a shared one.
var deliveryGraphCache = struct {
	sync.Mutex
	raw   []byte
	graph DeliveryIntegrityGraph
	valid bool
}{}

// parseDeliveryIntegrityGraph returns the graph for these bytes, parsing only when
// the bytes differ from the cached ones.
func parseDeliveryIntegrityGraph(raw []byte) (DeliveryIntegrityGraph, bool) {
	deliveryGraphCache.Lock()
	if deliveryGraphCache.valid && bytesEqual(deliveryGraphCache.raw, raw) {
		graph := copyDeliveryIntegrityGraph(deliveryGraphCache.graph)
		deliveryGraphCache.Unlock()
		return graph, true
	}
	deliveryGraphCache.Unlock()

	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil || graph.SchemaVersion != DeliveryIntegritySchemaVersion {
		return DeliveryIntegrityGraph{}, false
	}
	stored := copyDeliveryIntegrityGraph(graph)
	deliveryGraphCache.Lock()
	deliveryGraphCache.raw = append([]byte{}, raw...)
	deliveryGraphCache.graph = stored
	deliveryGraphCache.valid = true
	deliveryGraphCache.Unlock()
	return graph, true
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// copyDeliveryIntegrityGraph copies every field that a caller could write through.
// TestDeliveryGraphCopyCoversEveryField fails when a field is added to the struct
// without being handled here, so an omission cannot become a silently shared slice.
func copyDeliveryIntegrityGraph(in DeliveryIntegrityGraph) DeliveryIntegrityGraph {
	out := in
	out.Nodes = append([]DeliveryIntegrityNode{}, in.Nodes...)
	out.Edges = append([]DeliveryIntegrityEdge{}, in.Edges...)
	out.Claims = append([]ArtifactClaim{}, in.Claims...)
	out.Findings = append([]DeliveryIntegrityFinding{}, in.Findings...)
	out.Deliveries = append([]DeliveryTarget{}, in.Deliveries...)
	out.ValidationResults = append([]DeliveryValidationResult{}, in.ValidationResults...)
	out.RoadmapCriteria = append([]RoadmapCriterion{}, in.RoadmapCriteria...)
	out.Archivals = append([]ArchivedFragment{}, in.Archivals...)

	// ChangeSet carries slices of its own, so copying the outer slice is not enough.
	out.ChangeSets = make([]ChangeSet, len(in.ChangeSets))
	for i, set := range in.ChangeSets {
		set.Commits = append([]string{}, set.Commits...)
		set.Paths = append([]ObservedPath{}, set.Paths...)
		out.ChangeSets[i] = set
	}
	out.Reverse = copyStringSliceMap(in.Reverse)
	out.Paths = copyStringSliceMap(in.Paths)
	return out
}

func copyStringSliceMap(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = append([]string{}, values...)
	}
	return out
}
