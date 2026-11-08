package pose

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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
const (
	// deliveryGraphCacheBytes and deliveryGraphOwnerBytes bound what the parsed graphs
	// retain, in estimated heap bytes (see estimateGraphBytes), globally and per
	// worktree. Alternating between projects or generations evicts the least recently
	// used graph instead of growing, and a graph larger than a budget is served
	// without being retained.
	deliveryGraphCacheBytes = 192 << 20
	deliveryGraphOwnerBytes = 96 << 20
)

// deliveryGraphCache holds parsed graphs keyed by the engine, the schema and the
// digest of the bytes: identical bytes cannot produce a different graph, so two
// worktrees that hold the same index share one entry and no worktree can be served
// another's graph. The owner is the worktree root and only meters the budget.
var deliveryGraphCache = newBoundedCache(deliveryGraphCacheBytes, deliveryGraphOwnerBytes, 0, copyDeliveryIntegrityGraph)

// DeliveryGraphCacheStats reports hits, coalesced waits, misses, evictions and the
// bytes retained by the parsed-graph cache (spec pose-bounded-isolated-graph-caches R7).
func DeliveryGraphCacheStats() CacheStats { return deliveryGraphCache.Stats() }

// parseDeliveryIntegrityGraph is parseDeliveryIntegrityGraphErr for callers that
// treat any failure as "no graph".
func parseDeliveryIntegrityGraph(owner string, raw []byte) (DeliveryIntegrityGraph, bool) {
	graph, err := parseDeliveryIntegrityGraphErr(owner, raw)
	return graph, err == nil
}

// parseDeliveryIntegrityGraphErr returns the graph for these bytes, parsing only
// when no graph for them is retained and, when several callers miss together, once.
// owner is the worktree root the bytes were read from. When the cache has no room
// for another production or waiter the error wraps ErrCacheBusy: overload is an
// error the caller sees, never an unaccounted production outside the cache.
func parseDeliveryIntegrityGraphErr(owner string, raw []byte) (DeliveryIntegrityGraph, error) {
	sum := sha256.Sum256(raw)
	key := fmt.Sprintf("%s|%d|%x", engineVersion, len(raw), sum)
	return deliveryGraphCache.Get(context.Background(), owner, key, graphParseReserve(raw), func() (DeliveryIntegrityGraph, int64, error) {
		parsed, ok := decodeDeliveryIntegrityIndex(raw)
		if !ok {
			return DeliveryIntegrityGraph{}, 0, errInvalidDeliveryIndex
		}
		return parsed, estimateGraphBytes(parsed), nil
	})
}

// viewDeliveryIntegrityGraph hands fn the retained graph without copying it. fn
// must neither modify nor keep the graph or anything it reaches; the callers that
// only read a few fields of a large graph (a digest, a lookup) use it, the others
// get a deep copy from parseDeliveryIntegrityGraph.
func viewDeliveryIntegrityGraph(owner string, raw []byte, fn func(DeliveryIntegrityGraph)) bool {
	return viewDeliveryIntegrityGraphErr(owner, raw, fn) == nil
}

func viewDeliveryIntegrityGraphErr(owner string, raw []byte, fn func(DeliveryIntegrityGraph)) error {
	sum := sha256.Sum256(raw)
	key := fmt.Sprintf("%s|%d|%x", engineVersion, len(raw), sum)
	return deliveryGraphCache.View(context.Background(), owner, key, graphParseReserve(raw), func() (DeliveryIntegrityGraph, int64, error) {
		parsed, ok := decodeDeliveryIntegrityIndex(raw)
		if !ok {
			return DeliveryIntegrityGraph{}, 0, errInvalidDeliveryIndex
		}
		return parsed, estimateGraphBytes(parsed), nil
	}, fn)
}

// ViewDeliveryIntegrity gives fn the delivery graph of the current index without a
// copy (spec pose-bounded-isolated-graph-caches R1). fn is read-only: it must not
// modify or retain the graph. The error is that of GetDeliveryIntegrity.
func (s Store) ViewDeliveryIntegrity(fn func(DeliveryIntegrityGraph)) error {
	raw, _, err := s.ReadIndex("delivery-integrity.json")
	if errors.Is(err, ErrIndexGenerationUnavailable) {
		return err
	}
	if err != nil {
		return fmt.Errorf("pose: delivery-integrity index not found; run `pose index`")
	}
	if err := viewDeliveryIntegrityGraphErr(s.Root, raw, fn); err != nil {
		return deliveryIndexError(err)
	}
	return nil
}

var errInvalidDeliveryIndex = errors.New("pose: invalid delivery-integrity index")

// deliveryIndexError names an overload as such and anything else as an invalid index.
func deliveryIndexError(err error) error {
	if errors.Is(err, ErrCacheBusy) {
		return fmt.Errorf("pose: the delivery-integrity graph cache is busy: %w", err)
	}
	return errInvalidDeliveryIndex
}

// estimateGraphBytes estimates the heap a parsed graph retains by walking every
// field of it (see estimateRetainedBytes), so no variable-sized field is charged a
// constant. TestGraphByteEstimateTracksTheMeasuredHeap calibrates it on the real index.
func estimateGraphBytes(g DeliveryIntegrityGraph) int64 { return estimateRetainedBytes(g) }

// graphParseReserve is what a parse is expected to hold while it runs: the bytes
// being decoded and the graph being built.
func graphParseReserve(raw []byte) int64 { return int64(len(raw)) * 4 }

// decodeDeliveryIntegrityIndex reads schema 1, written as the expanded graph, and
// schema 2, which stores validation runs and sets once. Any other schema is not
// read as either.
func decodeDeliveryIntegrityIndex(raw []byte) (DeliveryIntegrityGraph, bool) {
	var head struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return DeliveryIntegrityGraph{}, false
	}
	switch head.SchemaVersion {
	case DeliveryIntegritySchemaVersion:
		var graph DeliveryIntegrityGraph
		if err := json.Unmarshal(raw, &graph); err != nil {
			return DeliveryIntegrityGraph{}, false
		}
		return graph, true
	case DeliveryIntegrityIndexSchemaVersion:
		var compact compactDeliveryGraph
		if err := json.Unmarshal(raw, &compact); err != nil {
			return DeliveryIntegrityGraph{}, false
		}
		graph, err := expandDeliveryIndex(compact)
		if err != nil {
			return DeliveryIntegrityGraph{}, false
		}
		return graph, true
	}
	return DeliveryIntegrityGraph{}, false
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// copyDeliveryIntegrityGraph copies every field that a caller could write through.
// TestDeliveryGraphCopyCoversEveryField fails when a field is added to the struct
// without being handled here, so an omission cannot become a silently shared slice.
func copyDeliveryIntegrityGraph(in DeliveryIntegrityGraph) DeliveryIntegrityGraph {
	out := in
	out.Nodes = make([]DeliveryIntegrityNode, len(in.Nodes))
	for i, node := range in.Nodes {
		node.Attributes = copyStringMap(node.Attributes)
		out.Nodes[i] = node
	}
	out.Edges = append([]DeliveryIntegrityEdge{}, in.Edges...)
	out.Claims = append([]ArtifactClaim{}, in.Claims...)
	out.Findings = append([]DeliveryIntegrityFinding{}, in.Findings...)
	out.Deliveries = append([]DeliveryTarget{}, in.Deliveries...)
	out.ValidationResults = make([]DeliveryValidationResult, len(in.ValidationResults))
	for i, result := range in.ValidationResults {
		result.ScopeProvenance = copyStringMap(result.ScopeProvenance)
		result.MaterialProvenance = copyStringMap(result.MaterialProvenance)
		out.ValidationResults[i] = result
	}
	out.RoadmapCriteria = make([]RoadmapCriterion, len(in.RoadmapCriteria))
	for i, criterion := range in.RoadmapCriteria {
		criterion.Refs = append([]string(nil), criterion.Refs...)
		criterion.Reasons = append([]string(nil), criterion.Reasons...)
		out.RoadmapCriteria[i] = criterion
	}
	out.Archivals = append([]ArchivedFragment{}, in.Archivals...)
	out.Removals = make([]ArtifactRemoval, len(in.Removals))
	for i, r := range in.Removals {
		if r.Supersedes != nil {
			supersedes := make(map[string]string, len(r.Supersedes))
			for k, v := range r.Supersedes {
				supersedes[k] = v
			}
			r.Supersedes = supersedes
		}
		if r.Refusals != nil {
			refusals := make(map[string]string, len(r.Refusals))
			for k, v := range r.Refusals {
				refusals[k] = v
			}
			r.Refusals = refusals
		}
		out.Removals[i] = r
	}
	if in.Removals == nil {
		out.Removals = nil
	}

	// ChangeSet carries slices of its own, so copying the outer slice is not enough.
	out.ChangeSets = make([]ChangeSet, len(in.ChangeSets))
	for i, set := range in.ChangeSets {
		set.Commits = append([]string{}, set.Commits...)
		set.Paths = append([]ObservedPath{}, set.Paths...)
		if set.MaterialInputs != nil {
			inputs := MaterialInputs{Names: append([]string(nil), set.MaterialInputs.Names...), Moved: append([]string(nil), set.MaterialInputs.Moved...)}
			set.MaterialInputs = &inputs
		}
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
