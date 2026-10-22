package pose

import (
	"encoding/json"
	"fmt"
	"sort"
)

// DeliveryValidationResultSchema is the schema of the structured result of `pose validate`.
const DeliveryValidationResultSchema = 1

// ParseDeliveryValidationRun reads the structured result of a validation run into the results
// the delivery index holds: one per check that declares an evidence class, with the run's head,
// time, provenance and scope provenance. relative is the path the results were read from. It is
// the one reading of that file: the index is built with it and checked against it.
func ParseDeliveryValidationRun(raw []byte, relative string) ([]DeliveryValidationResult, error) {
	var run struct {
		SchemaVersion    int               `json:"schema_version"`
		GeneratedAt      string            `json:"generated_at"`
		GitHead          string            `json:"git_head,omitempty"`
		ProvenanceDigest string            `json:"provenance_digest,omitempty"`
		ScopeProvenance  map[string]string `json:"scope_provenance,omitempty"`
		Checks           []struct {
			ID            string `json:"id"`
			Module        string `json:"module"`
			Name          string `json:"name"`
			Severity      string `json:"severity"`
			EvidenceClass string `json:"evidence_class,omitempty"`
			Outcome       string `json:"outcome"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(raw, &run); err != nil || run.SchemaVersion != DeliveryValidationResultSchema {
		return nil, fmt.Errorf("invalid structured validation result")
	}
	results := make([]DeliveryValidationResult, 0, len(run.Checks))
	for _, check := range run.Checks {
		if check.EvidenceClass == "" {
			continue
		}
		results = append(results, DeliveryValidationResult{ID: check.ID, Module: check.Module, Check: check.Name, EvidenceClass: check.EvidenceClass, Severity: check.Severity, Outcome: check.Outcome, GitHead: run.GitHead, GeneratedAt: run.GeneratedAt, ProvenanceDigest: run.ProvenanceDigest, ScopeProvenance: run.ScopeProvenance, Report: relative})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, nil
}
