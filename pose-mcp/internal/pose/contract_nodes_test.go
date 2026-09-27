package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contractNodesBody(assumptionStatus, decisionStatus, decisionBasis string) string {
	decision := "### Decision D1\n- Basis: " + decisionBasis + "\n- Minimal option: keep one reader\n- Selected option: dual reader\n- Rationale: legacy logs stay readable\n- Consequences: two schemas\n- Falsifier: a v1 log misread\n"
	if decisionStatus != "" {
		decision += "- Status: " + decisionStatus + "\n"
	}
	return "# Spec: nodes\n\n## 2. Requirements\n\n- R1: Keep legacy logs readable.\n- R2: Record node state.\n\n## 4. Tasks\n\n- [ ] Implement.\n\n## 5. Decisions\n\n### Assumption A1\n- Claim: v1 logs hold only requirements\n- Status: " + assumptionStatus + "\n- Evidence: test:TestABMNodeTransitionsRequireAcknowledgement\n- Scope: this spec's amendment log\n- Affects: R1\n\n" + decision + "\n## 7. Final Report\n\nPending.\n"
}

func findingsContain(findings []string, fragment string) bool {
	return strings.Contains(strings.Join(findings, "\n"), fragment)
}

func baselineV2(projection ContractNodesProjection) Amendment {
	event := Amendment{Schema: AmendmentSchemaV2, Change: "baseline", Author: "@agent", Assurance: "declared", Hashes: map[string]string{}, After: map[string]NodeState{}}
	for _, node := range projection.Nodes {
		event.IDs = append(event.IDs, node.ID)
		event.After[node.ID] = NodeState{Hash: node.Hash, State: node.State, Relations: node.Relations}
		event.Hashes[node.ID] = node.Hash
	}
	return event
}

func TestABMContractNodesProjectionCoversRADWithStableDigest(t *testing.T) {
	projection := ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1, A1"))
	var ids []string
	for _, node := range projection.Nodes {
		ids = append(ids, node.ID+":"+node.Kind+":"+node.State+":"+strings.Join(node.Relations, ","))
		if node.Namespace != "spec:nodes" || node.Hash == "" {
			t.Fatalf("node without namespace or hash: %+v", node)
		}
	}
	want := "R1:requirement:active:,R2:requirement:active:,A1:assumption:unverified:R1,D1:decision:active:A1,R1"
	if strings.Join(ids, ",") != want {
		t.Fatalf("projection = %s, want %s", strings.Join(ids, ","), want)
	}
	if projection.SchemaVersion != ContractNodesSchemaVersion || len(projection.Diagnostics) != 0 {
		t.Fatalf("projection header or diagnostics: %+v", projection)
	}
	reflowed := strings.Replace(contractNodesBody("unverified", "", "R1, A1"), "Keep legacy logs readable.", "Keep   legacy  logs readable. ", 1)
	if ProjectContractNodes("nodes", reflowed).Digest != projection.Digest {
		t.Fatal("whitespace-only reflow changed the projection digest")
	}
	requirementsOnly := ProjectContractNodes("nodes", "# Spec\n\n## 2. Requirements\n\n- R1: One.\n")
	if len(requirementsOnly.Nodes) != 1 || requirementsOnly.Nodes[0].Kind != "requirement" {
		t.Fatalf("an R-only spec did not project as R-only: %+v", requirementsOnly.Nodes)
	}
}

func TestABMContractNodesV1HistoryStaysRequirementHistory(t *testing.T) {
	body := contractNodesBody("unverified", "", "R1, A1")
	v1 := []Amendment{{Schema: AmendmentSchema, Change: "baseline", Author: "@agent", IDs: []string{"R1", "R2"}, Hashes: CurrentRequirementHashes(body)}}
	legacy := UnacknowledgedChanges(body, v1)
	if got := UnacknowledgedNodeChanges("nodes", body, v1, false); strings.Join(got, "|") != strings.Join(legacy, "|") {
		t.Fatalf("without the capability the gate changed: %v vs %v", got, legacy)
	}
	acknowledged := AcknowledgedNodeStates(v1)
	if acknowledged["R1"].State != "active" || acknowledged["R1"].Hash == "" {
		t.Fatalf("v1 baseline was not read as active requirements: %+v", acknowledged)
	}
	adopted := UnacknowledgedNodeChanges("nodes", body, v1, true)
	if findingsContain(adopted, "R1") || !findingsContain(adopted, "A1 was added without an amendment event") || !findingsContain(adopted, "D1 was added") {
		t.Fatalf("adoption must keep v1 requirement history and oblige only the new nodes: %v", adopted)
	}
}

func TestABMContractNodesRefusesSchema2WithoutCapability(t *testing.T) {
	body := contractNodesBody("unverified", "", "R1, A1")
	events := []Amendment{baselineV2(ProjectContractNodes("nodes", body))}
	if got := UnacknowledgedNodeChanges("nodes", body, events, false); len(got) != 1 || !strings.Contains(got[0], "need the contract_nodes_version capability") {
		t.Fatalf("a schema-2 log was read permissively without the capability: %v", got)
	}
	if got := UnacknowledgedNodeChanges("nodes", body, events, true); len(got) != 0 {
		t.Fatalf("an adopted v2 baseline does not acknowledge its own state: %v", got)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "amendments.jsonl")
	for name, line := range map[string]string{
		"no after":      `{"schema":2,"at":"2026-09-27T00:00:00Z","change":"baseline","ids":["A1"],"author":"@a","assurance":"declared","hashes":{}}`,
		"no assurance":  `{"schema":2,"at":"2026-09-27T00:00:00Z","change":"baseline","ids":["A1"],"author":"@a","hashes":{},"after":{"A1":{"hash":"x","state":"unverified"}}}`,
		"bad id":        `{"schema":2,"at":"2026-09-27T00:00:00Z","change":"baseline","ids":["X1"],"author":"@a","assurance":"declared","hashes":{},"after":{"X1":{"hash":"x","state":"s"}}}`,
		"schema 3":      `{"schema":3,"at":"2026-09-27T00:00:00Z","change":"baseline","ids":["R1"],"author":"@a","hashes":{}}`,
		"v1 transition": `{"schema":1,"at":"2026-09-27T00:00:00Z","change":"transition","ids":["R1"],"author":"@a","rationale":"r","hashes":{"R1":"x"}}`,
	} {
		if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadAmendments(path); err == nil {
			t.Errorf("%s: malformed amendment line was accepted", name)
		}
	}
}

func TestABMNodeTransitionsRequireAcknowledgement(t *testing.T) {
	baseline := baselineV2(ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1, A1")))
	verified := contractNodesBody("verified", "", "R1, A1")
	findings := UnacknowledgedNodeChanges("nodes", verified, []Amendment{baseline}, true)
	if !findingsContain(findings, "A1 moved from unverified to verified without a transition event") {
		t.Fatalf("a silent state change was accepted: %v", findings)
	}
	current := CurrentNodeStates(ProjectContractNodes("nodes", verified))["A1"]
	transition := Amendment{Schema: AmendmentSchemaV2, Change: "transition", Author: "@agent", Rationale: "evidence attached", Assurance: "declared",
		IDs: []string{"A1"}, Before: map[string]NodeState{"A1": baseline.After["A1"]}, After: map[string]NodeState{"A1": {Hash: current.Hash, State: current.State, Relations: current.Relations}}}
	if findings := UnacknowledgedNodeChanges("nodes", verified, []Amendment{baseline, transition}, true); findingsContain(findings, "A1") {
		t.Fatalf("an acknowledged transition still reported: %v", findings)
	}
	for _, c := range []struct {
		kind, from, to string
		ok             bool
	}{
		{"assumption", "unverified", "verified", true},
		{"assumption", "verified", "invalidated", true},
		{"assumption", "withdrawn", "verified", false},
		{"decision", "active", "withdrawn", true},
		{"decision", "withdrawn", "active", false},
	} {
		if ContractNodeTransitionAllowed(c.kind, c.from, c.to) != c.ok {
			t.Errorf("transition %s %s→%s allowed=%v", c.kind, c.from, c.to, !c.ok)
		}
	}
}

func TestABMNodeTransitionsInvalidatedAssumptionBlocksActiveDecision(t *testing.T) {
	invalidated := ProjectContractNodes("nodes", contractNodesBody("invalidated", "", "R1, A1"))
	if !diagnosticCodes(invalidated)["invalidated-basis"] {
		t.Fatalf("an active decision on an invalidated assumption passed: %+v", invalidated.Diagnostics)
	}
	if findings := UnacknowledgedNodeChanges("nodes", contractNodesBody("invalidated", "", "R1, A1"), []Amendment{baselineV2(invalidated)}, true); !findingsContain(findings, "D1 invalidated-basis") {
		t.Fatalf("the gate did not report the invalidated basis: %v", findings)
	}
	if withdrawnDecision := ProjectContractNodes("nodes", contractNodesBody("invalidated", "withdrawn", "R1, A1")); diagnosticCodes(withdrawnDecision)["invalidated-basis"] {
		t.Fatal("a withdrawn decision still counted as resting on the invalidated assumption")
	}
	if rebased := ProjectContractNodes("nodes", contractNodesBody("invalidated", "", "R1")); diagnosticCodes(rebased)["invalidated-basis"] {
		t.Fatal("a decision reconciled off the invalidated assumption still failed")
	}
	if withdrawn := ProjectContractNodes("nodes", contractNodesBody("withdrawn", "", "R1, A1")); !diagnosticCodes(withdrawn)["withdrawn-basis"] {
		t.Fatal("an active decision still using a withdrawn assumption passed")
	}
	if orphan := ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1, A7")); !diagnosticCodes(orphan)["orphan-ref"] {
		t.Fatal("an orphan basis ref was not reported")
	}
	if cycle := ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1, D1")); !diagnosticCodes(cycle)["decision-cycle"] {
		t.Fatal("a decision referencing a decision was not reported")
	}
	if bad := ProjectContractNodes("nodes", contractNodesBody("unverified", "retired", "R1")); !diagnosticCodes(bad)["invalid-status"] {
		t.Fatal("an unknown decision status was accepted")
	}
}

func diagnosticCodes(projection ContractNodesProjection) map[string]bool {
	codes := map[string]bool{}
	for _, diagnostic := range projection.Diagnostics {
		codes[diagnostic.Code] = true
	}
	return codes
}

func TestABMNodeTransitionsEditorialCannotMaskSemantic(t *testing.T) {
	base := CurrentNodeStates(ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1, A1")))["D1"]
	acknowledged := NodeState{Hash: base.Hash, State: base.State, Relations: base.Relations}
	reworded := CurrentNodeStates(ProjectContractNodes("nodes", strings.Replace(contractNodesBody("unverified", "", "R1, A1"), "legacy logs stay readable", "old logs remain readable", 1)))["D1"]
	if reworded.Hash == base.Hash || !EditorialAllowed(reworded, acknowledged) {
		t.Fatal("pure rewording was not recognised as editorial")
	}
	rebased := CurrentNodeStates(ProjectContractNodes("nodes", contractNodesBody("unverified", "", "R1")))["D1"]
	if EditorialAllowed(rebased, acknowledged) {
		t.Fatal("a changed decision basis was allowed as editorial")
	}
	withdrawn := CurrentNodeStates(ProjectContractNodes("nodes", contractNodesBody("unverified", "withdrawn", "R1, A1")))["D1"]
	if EditorialAllowed(withdrawn, acknowledged) {
		t.Fatal("a state change was allowed as editorial")
	}
}

func TestABMNodeDigestIgnoresDerivedSections(t *testing.T) {
	body := contractNodesBody("unverified", "", "R1, A1")
	base := ProjectContractNodes("nodes", body).Digest
	derived := strings.Replace(strings.Replace(body, "- [ ] Implement.", "- [x] Implement.\n- [x] Validate.", 1), "Pending.", "Delivered with metrics 42/42.", 1)
	if ProjectContractNodes("nodes", derived).Digest != base {
		t.Fatal("Tasks or Final Report edits changed the contract-node digest")
	}
	if ProjectContractNodes("nodes", strings.Replace(body, "dual reader", "single reader", 1)).Digest == base {
		t.Fatal("a Decisions change did not change the contract-node digest")
	}
}

// A real schema-1 log from this repository must keep its meaning under the
// new reader, with and without the capability.
func TestABMContractNodesMigratesARealV1Log(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "contract-nodes", "v1-spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := LoadAmendments(filepath.Join("testdata", "contract-nodes", "v1-amendments.jsonl"))
	if err != nil || len(events) == 0 {
		t.Fatalf("real v1 log unreadable: %v", err)
	}
	legacy := UnacknowledgedChanges(string(body), events)
	if got := UnacknowledgedNodeChanges("pose-governance-gate-activation", string(body), events, false); strings.Join(got, "|") != strings.Join(legacy, "|") {
		t.Fatalf("the new reader changed a v1 verdict: %v vs %v", got, legacy)
	}
	projection := ProjectContractNodes("pose-governance-gate-activation", string(body))
	adopted := UnacknowledgedNodeChanges("pose-governance-gate-activation", string(body), events, true)
	for _, node := range projection.Nodes {
		if node.Kind == "requirement" && findingsContain(adopted, node.ID+" ") && !findingsContain(legacy, node.ID+" ") {
			t.Fatalf("adoption invented a requirement finding for %s: %v", node.ID, adopted)
		}
	}
	for _, e := range events {
		if e.Change != "baseline" && e.Rationale == "" {
			t.Fatalf("a legacy event without rationale was read; none may be fabricated: %+v", e)
		}
	}
}
