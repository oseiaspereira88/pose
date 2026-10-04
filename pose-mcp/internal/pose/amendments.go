package pose

// Spec amendment history (spec pose-spec-amendment-history): material changes
// to published requirements are append-only, reviewed events stored in
// .pose/specs/<slug>/amendments.jsonl. Detection is deterministic — each
// event records the normalized-text hash of the affected R-IDs, and the
// closeout gate compares current hashes against the latest acknowledged
// state. Published IDs are never renumbered; a withdrawn ID stays
// addressable with an empty hash.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AmendmentSchema is the requirement-only amendments.jsonl schema version.
const AmendmentSchema = 1

// AmendmentSchemaV2 records requirement, assumption and decision nodes with
// their state (spec pose-abm-contract-nodes). Engines that know only schema 1
// refuse these lines instead of reading them as requirement history.
const AmendmentSchemaV2 = 2

// ValidAmendmentChanges is the material-change taxonomy. "baseline" snapshots
// every requirement; "editorial" acknowledges non-semantic rewording.
var ValidAmendmentChanges = map[string]bool{
	"baseline": true, "added": true, "withdrawn": true, "semantic": true, "editorial": true,
}

// transition acknowledges a state change of an assumption or decision; it
// exists only in schema-2 events.
var validAmendmentChangesV2 = map[string]bool{
	"baseline": true, "added": true, "withdrawn": true, "semantic": true, "editorial": true, "transition": true,
}

// Amendment is one append-only event.
type Amendment struct {
	Schema    int               `json:"schema"`
	At        string            `json:"at"` // RFC3339 UTC
	Change    string            `json:"change"`
	IDs       []string          `json:"ids"`
	Rationale string            `json:"rationale,omitempty"`
	Author    string            `json:"author"`
	Reviewer  string            `json:"reviewer,omitempty"`
	Hashes    map[string]string `json:"hashes"` // R-ID → hash after the change ("" = withdrawn)
	// Schema 2 only: the node state before and after the event, and how the
	// origin is assured. Aliases are declared, not verified.
	Before    map[string]NodeState `json:"before,omitempty"`
	After     map[string]NodeState `json:"after,omitempty"`
	Assurance string               `json:"assurance,omitempty"`
}

// RequirementHash fingerprints one requirement's normalized text (whitespace
// collapsed): editorial formatting does not change the hash, wording does.
func RequirementHash(text string) string {
	normalized := strings.Join(strings.Fields(text), " ")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])[:12]
}

// CurrentRequirementHashes returns the hash of every declared requirement in
// a spec body.
func CurrentRequirementHashes(body string) map[string]string {
	hashes := map[string]string{}
	for _, r := range ParseRequirementTrace(body).Requirements {
		hashes[r.ID] = RequirementHash(r.Text)
	}
	return hashes
}

// AmendmentsPath returns the event log of a spec, derived from its identity
// path so no two specs share one (spec pose-flat-spec-amendments).
//
// A folder spec (`<dir>/spec.md`, or the directory of a split spec) keeps its
// sibling `amendments.jsonl`, which every existing journal uses. A flat dated
// spec (`.pose/specs/YYYY-MM-DD-<slug>.md`) gets `YYYY-MM-DD-<slug>.amendments.jsonl`
// beside it; the sibling rule would have put every flat spec of a directory
// in one shared `.pose/specs/amendments.jsonl`.
func AmendmentsPath(specPath string) string {
	if info, err := os.Stat(specPath); err == nil && info.IsDir() {
		return filepath.Join(specPath, "amendments.jsonl")
	}
	if filepath.Base(specPath) == "spec.md" || !strings.HasSuffix(specPath, ".md") {
		return filepath.Join(filepath.Dir(specPath), "amendments.jsonl")
	}
	return strings.TrimSuffix(specPath, ".md") + ".amendments.jsonl"
}

// LoadAmendments parses an amendments.jsonl. A missing file yields an empty
// history; a malformed line is an error (append-only logs are never partially
// trusted).
func LoadAmendments(path string) ([]Amendment, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var events []Amendment
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var e Amendment
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		switch e.Schema {
		case AmendmentSchema:
			if !ValidAmendmentChanges[e.Change] {
				return nil, fmt.Errorf("line %d: invalid change %q", line, e.Change)
			}
		case AmendmentSchemaV2:
			if !validAmendmentChangesV2[e.Change] {
				return nil, fmt.Errorf("line %d: invalid change %q", line, e.Change)
			}
			if len(e.IDs) == 0 || len(e.After) != len(e.IDs) {
				return nil, fmt.Errorf("line %d: schema-2 event needs an after state for every id", line)
			}
			for _, id := range e.IDs {
				if !ContractNodeIDValid(id) {
					return nil, fmt.Errorf("line %d: invalid node id %q", line, id)
				}
				if _, ok := e.After[id]; !ok {
					return nil, fmt.Errorf("line %d: node %s has no after state", line, id)
				}
			}
			if e.Assurance != "declared" {
				return nil, fmt.Errorf("line %d: schema-2 event must declare assurance \"declared\"", line)
			}
		default:
			return nil, fmt.Errorf("line %d: unsupported schema %d (engine supports %d and %d)", line, e.Schema, AmendmentSchema, AmendmentSchemaV2)
		}
		if e.Author == "" {
			return nil, fmt.Errorf("line %d: author is required", line)
		}
		if e.Change != "baseline" && e.Rationale == "" {
			return nil, fmt.Errorf("line %d: rationale is required for change %q", line, e.Change)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// AcknowledgedHashes overlays events in order and returns the latest
// acknowledged hash per R-ID ("" means acknowledged as withdrawn).
func AcknowledgedHashes(events []Amendment) map[string]string {
	latest := map[string]string{}
	for _, e := range events {
		for id, h := range e.Hashes {
			latest[id] = h
		}
	}
	return latest
}

// UnacknowledgedChanges compares the spec's current requirements against the
// acknowledged history. Returns human-readable findings; empty means the
// history acknowledges the current state.
func UnacknowledgedChanges(body string, events []Amendment) []string {
	current := CurrentRequirementHashes(body)
	acknowledged := AcknowledgedHashes(events)
	var findings []string
	for _, r := range ParseRequirementTrace(body).Requirements {
		ack, ok := acknowledged[r.ID]
		switch {
		case !ok:
			findings = append(findings, r.ID+" was added without an amendment event (pose amend --change added)")
		case ack == "":
			findings = append(findings, r.ID+" is acknowledged as withdrawn but still declared in Requirements")
		case ack != current[r.ID]:
			findings = append(findings, r.ID+" changed after its last acknowledged amendment (pose amend --change semantic|editorial)")
		}
	}
	for id, ack := range acknowledged {
		if _, exists := current[id]; !exists && ack != "" {
			findings = append(findings, id+" was removed without a withdrawn amendment event")
		}
	}
	return findings
}
