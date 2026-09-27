package pose

// Contract nodes (spec pose-abm-contract-nodes): one versioned projection of
// a spec's requirements (R), assumptions (A) and decisions (D), with stable
// IDs, a namespace, a content hash kept apart from the node's state, and the
// relations between them. The projection is rebuilt from the spec body —
// Requirements and Decisions stay the canonical sources — so nothing here is
// persisted except the amendment events that acknowledge a change.
//
// Adoption is explicit. Without `contract_nodes_version: 1` in the review
// policy the amendment gate keeps its requirement-only behaviour, and a log
// holding a schema-2 event is refused rather than read permissively.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ContractNodesSchemaVersion versions the projection. It is deliberately
// independent of the product release number.
const ContractNodesSchemaVersion = 1

// ContractNodesPolicyVersion is the only adopted value of the review policy's
// contract_nodes_version capability.
const ContractNodesPolicyVersion = 1

var contractNodeIDRE = regexp.MustCompile(`^[RAD]\d+$`)

// ContractNode is one requirement, assumption or decision.
type ContractNode struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"` // requirement | assumption | decision
	Namespace string   `json:"namespace"`
	Hash      string   `json:"hash"`
	State     string   `json:"state"`
	Relations []string `json:"relations,omitempty"` // A: affects; D: basis
}

// ContractNodesProjection is the versioned view of a spec's nodes.
type ContractNodesProjection struct {
	SchemaVersion int                     `json:"schema_version"`
	Namespace     string                  `json:"namespace"`
	Nodes         []ContractNode          `json:"nodes"`
	Diagnostics   []DesignBasisDiagnostic `json:"diagnostics,omitempty"`
	Digest        string                  `json:"digest"`
}

// NodeState is what an amendment event acknowledges about one node.
type NodeState struct {
	Hash      string   `json:"hash"`
	State     string   `json:"state"`
	Relations []string `json:"relations,omitempty"`
}

// contractNodeTransitions lists the state changes a transition event may
// acknowledge. A withdrawn node never comes back under the same ID.
var contractNodeTransitions = map[string]map[string]bool{
	"assumption:unverified":  {"verified": true, "invalidated": true, "withdrawn": true},
	"assumption:verified":    {"invalidated": true, "withdrawn": true},
	"assumption:invalidated": {"verified": true, "withdrawn": true},
	"decision:active":        {"withdrawn": true},
}

// ContractNodeTransitionAllowed reports whether kind may move from one state
// to another.
func ContractNodeTransitionAllowed(kind, from, to string) bool {
	return contractNodeTransitions[kind+":"+from][to]
}

// ProjectContractNodes builds the projection of one spec body.
func ProjectContractNodes(slug, body string) ContractNodesProjection {
	namespace := "spec:" + slug
	projection := ContractNodesProjection{SchemaVersion: ContractNodesSchemaVersion, Namespace: namespace, Nodes: []ContractNode{}}
	for _, r := range ParseRequirementTrace(body).Requirements {
		projection.Nodes = append(projection.Nodes, ContractNode{ID: r.ID, Kind: "requirement", Namespace: namespace, Hash: RequirementHash(r.Text), State: "active"})
	}
	basis := ParseDesignBasis(body)
	states := map[string]string{}
	for _, a := range basis.Assumptions {
		affects := sortedCopy(a.Affects)
		projection.Nodes = append(projection.Nodes, ContractNode{
			ID: a.ID, Kind: "assumption", Namespace: namespace, State: a.Status, Relations: affects,
			Hash: RequirementHash(strings.Join([]string{a.Claim, a.Scope, a.Evidence, "affects:" + strings.Join(affects, ",")}, "\x1f")),
		})
		states[a.ID] = a.Status
	}
	for _, d := range basis.Decisions {
		state := d.Status
		if state == "" {
			state = "active"
		}
		refs := sortedCopy(d.Basis)
		projection.Nodes = append(projection.Nodes, ContractNode{
			ID: d.ID, Kind: "decision", Namespace: namespace, State: state, Relations: refs,
			Hash: RequirementHash(strings.Join([]string{"basis:" + strings.Join(refs, ","), d.MinimalOption, d.SelectedOption, d.Rationale, d.Consequences, d.Falsifier}, "\x1f")),
		})
		if state != "active" {
			continue
		}
		for _, ref := range refs {
			switch states[ref] {
			case "invalidated":
				projection.Diagnostics = append(projection.Diagnostics, DesignBasisDiagnostic{Severity: "error", Code: "invalidated-basis", ID: d.ID, Line: d.Line,
					Message: fmt.Sprintf("active decision %s rests on invalidated assumption %s; change its basis or withdraw it", d.ID, ref)})
			case "withdrawn":
				projection.Diagnostics = append(projection.Diagnostics, DesignBasisDiagnostic{Severity: "error", Code: "withdrawn-basis", ID: d.ID, Line: d.Line,
					Message: fmt.Sprintf("active decision %s still uses withdrawn assumption %s", d.ID, ref)})
			}
		}
	}
	for _, diagnostic := range basis.Diagnostics {
		if diagnostic.Severity == "error" {
			projection.Diagnostics = append(projection.Diagnostics, diagnostic)
		}
	}
	sort.Slice(projection.Nodes, func(i, j int) bool { return contractNodeLess(projection.Nodes[i].ID, projection.Nodes[j].ID) })
	sort.SliceStable(projection.Diagnostics, func(i, j int) bool {
		if projection.Diagnostics[i].ID != projection.Diagnostics[j].ID {
			return projection.Diagnostics[i].ID < projection.Diagnostics[j].ID
		}
		return projection.Diagnostics[i].Code < projection.Diagnostics[j].Code
	})
	projection.Digest = contractNodesDigest(projection)
	return projection
}

// contractNodesDigest covers identity, content, state and relations; line
// numbers and diagnostics are presentation and stay out of it.
func contractNodesDigest(projection ContractNodesProjection) string {
	view := struct {
		SchemaVersion int            `json:"schema_version"`
		Namespace     string         `json:"namespace"`
		Nodes         []ContractNode `json:"nodes"`
	}{projection.SchemaVersion, projection.Namespace, projection.Nodes}
	encoded, _ := json.Marshal(view)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func contractNodeLess(a, b string) bool {
	if a[0] != b[0] {
		return strings.Index("RAD", a[:1]) < strings.Index("RAD", b[:1])
	}
	var na, nb int
	fmt.Sscanf(a[1:], "%d", &na)
	fmt.Sscanf(b[1:], "%d", &nb)
	return na < nb
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

// CurrentNodeStates indexes a projection by ID.
func CurrentNodeStates(projection ContractNodesProjection) map[string]ContractNode {
	out := map[string]ContractNode{}
	for _, node := range projection.Nodes {
		out[node.ID] = node
	}
	return out
}

// AcknowledgedNodeStates overlays the history in order. A schema-1 event is
// read as a requirement-only snapshot: a non-empty hash is an active
// requirement, an empty one a withdrawn requirement.
func AcknowledgedNodeStates(events []Amendment) map[string]NodeState {
	latest := map[string]NodeState{}
	for _, e := range events {
		if e.Schema == AmendmentSchemaV2 {
			for id, state := range e.After {
				latest[id] = state
			}
			continue
		}
		for id, hash := range e.Hashes {
			state := "active"
			if hash == "" {
				state = "withdrawn"
			}
			latest[id] = NodeState{Hash: hash, State: state}
		}
	}
	return latest
}

// UnacknowledgedNodeChanges is the amendment gate. Without the adopted
// capability it keeps the requirement-only contract and refuses schema-2
// events; with it, it compares every node's content, state and relations
// against the acknowledged history and reports the decision-basis rules.
func UnacknowledgedNodeChanges(slug, body string, events []Amendment, adopted bool) []string {
	if !adopted {
		for _, e := range events {
			if e.Schema == AmendmentSchemaV2 {
				return []string{"schema-2 amendment events need the contract_nodes_version capability; they are refused, not read as requirement history"}
			}
		}
		return UnacknowledgedChanges(body, events)
	}
	projection := ProjectContractNodes(slug, body)
	acknowledged := AcknowledgedNodeStates(events)
	var findings []string
	for _, node := range projection.Nodes {
		ack, ok := acknowledged[node.ID]
		switch {
		case !ok:
			findings = append(findings, fmt.Sprintf("%s was added without an amendment event (pose amend --change added)", node.ID))
		case node.Kind == "requirement" && ack.State == "withdrawn":
			findings = append(findings, node.ID+" is acknowledged as withdrawn but still declared in Requirements")
		case ack.State != node.State:
			findings = append(findings, fmt.Sprintf("%s moved from %s to %s without a transition event (pose amend --change transition)", node.ID, ack.State, node.State))
		case ack.Hash != node.Hash:
			findings = append(findings, node.ID+" changed after its last acknowledged amendment (pose amend --change semantic|editorial)")
		}
	}
	current := CurrentNodeStates(projection)
	for id, ack := range acknowledged {
		if _, exists := current[id]; !exists && ack.State != "withdrawn" {
			findings = append(findings, id+" was removed without a withdrawn amendment event")
		}
	}
	for _, diagnostic := range projection.Diagnostics {
		findings = append(findings, fmt.Sprintf("%s %s: %s", diagnostic.ID, diagnostic.Code, diagnostic.Message))
	}
	sort.Strings(findings)
	return findings
}

// EditorialAllowed reports whether a node's change may be acknowledged as
// editorial: rewording only. A changed state or relation set is semantic by
// structure and cannot be masked as editorial.
func EditorialAllowed(node ContractNode, acknowledged NodeState) bool {
	return node.State == acknowledged.State && strings.Join(node.Relations, ",") == strings.Join(acknowledged.Relations, ",")
}

// ContractNodeIDValid reports whether id is an R/A/D node ID.
func ContractNodeIDValid(id string) bool { return contractNodeIDRE.MatchString(id) }

// ContractNodesAdopted reads the review policy's capability.
func (s Store) ContractNodesAdopted() (bool, error) {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return false, err
	}
	return policy.ContractNodesVersion == ContractNodesPolicyVersion, nil
}
