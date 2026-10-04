package pose

import (
	"sort"
	"strings"
)

// Diagnostic is a typed fact a producer emits about something still owed
// (spec pose-typed-producer-diagnostics). The code and refs are the contract;
// Message is a rendering for people and nothing parses it. Producers keep
// their legacy strings during the transition, rendered from these items where
// the producer builds them itself.
type Diagnostic struct {
	Code      string   `json:"code"`
	Domain    string   `json:"domain"`
	Refs      []string `json:"refs,omitempty"`
	Condition string   `json:"condition,omitempty"`
	// Opaque marks a legacy text blocker the producer cannot yet type: it
	// keeps its origin and message and claims nothing else.
	Opaque  bool   `json:"opaque,omitempty"`
	Message string `json:"message"`
}

// Diagnostic domains.
const (
	DiagnosticReadiness = "readiness"
	DiagnosticReview    = "review"
	DiagnosticCloseout  = "closeout"
	DiagnosticStart     = "start"
)

// DiagnosticCatalog lists every code a producer may emit, with the domain
// that owns it and what satisfies it. A code missing here is a defect the
// catalog test reports.
var DiagnosticCatalog = map[string]struct {
	Domain    string
	Condition string
}{
	"dor-acceptance-criteria-missing": {DiagnosticReadiness, "requirements carry acceptance criteria with stable ids (- R<N>: ...)"},
	"dependency-not-done":             {DiagnosticReadiness, "the referenced artifact reaches status done"},
	"dependency-unresolved":           {DiagnosticReadiness, "the reference resolves to an authorized artifact"},
	"dependency-graph-invalid":        {DiagnosticReadiness, "the dependency graph is valid for this reference"},
	"blocked-cause-unknown":           {DiagnosticReadiness, "the cause of the block is recorded or the spec leaves blocked"},
	"judgment-unanswered":             {DiagnosticReview, "a reviewer records a conclusion for the criterion"},
	"review-not-approved":             {DiagnosticCloseout, "a fresh approved attestation of the current sealed bundle"},
	"review-blocker":                  {DiagnosticCloseout, "the review blocker is resolved at its source"},
	"lifecycle-not-done":              {DiagnosticCloseout, "the guarded lifecycle transition sets status done"},
	"child-scope-open":                {DiagnosticCloseout, "the child scope's closeout is terminal"},
	"external-member-unaccepted":      {DiagnosticCloseout, "federated acceptance of the external member"},
	"federated-acceptance-blocker":    {DiagnosticCloseout, "the federated acceptance blocker is resolved at its source"},
	"review-criterion-owed":           {DiagnosticStart, "the criterion is answered at closeout review"},
	"action-request-pending":          {DiagnosticCloseout, "the action request receives a satisfying answer from its authority, or is waived"},
}

// NewDiagnostic builds a catalogued diagnostic; the condition comes from the
// catalog so a producer cannot describe the same code two ways.
func NewDiagnostic(code, message string, refs ...string) Diagnostic {
	entry := DiagnosticCatalog[code]
	return Diagnostic{Code: code, Domain: entry.Domain, Refs: refs, Condition: entry.Condition, Message: message}
}

// OpaqueDiagnostic wraps a legacy text blocker without inventing structure.
func OpaqueDiagnostic(code, domain, message string) Diagnostic {
	return Diagnostic{Code: code, Domain: domain, Opaque: true, Message: message, Condition: DiagnosticCatalog[code].Condition}
}

// dependencyWaitingCode classifies a readiness waiting reason by the
// resolver state it came from.
func dependencyWaitingCode(resolutionState, graphReason string, notDone bool) string {
	switch {
	case notDone:
		return "dependency-not-done"
	case graphReason != "":
		return "dependency-graph-invalid"
	case resolutionState != "":
		return "dependency-unresolved"
	}
	return "dependency-unresolved"
}

// SortDiagnostics orders diagnostics deterministically by code then refs.
func SortDiagnostics(items []Diagnostic) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Code != items[j].Code {
			return items[i].Code < items[j].Code
		}
		return strings.Join(items[i].Refs, ",") < strings.Join(items[j].Refs, ",")
	})
}
