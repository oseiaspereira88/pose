package pose

// Requirement-to-evidence traceability (spec pose-requirement-evidence-traceability).
// The trace is declared in the spec itself — a "### Requirement trace"
// subsection under "## 6. Validation" — so links stay explicit, reviewable
// and valid offline. Compliance is never inferred from file proximity.
//
// Trace bullet grammar (additive to the spec contract):
//
//	- R<N> [satisfied] <verification case and evidence refs>
//	- R<N> [waived: <reason>] <optional context>
//	- R<N> [withdrawn: <reason>] <optional context>
//
// Evidence refs are structured tokens inside the free text — check:<name>,
// test:<id>, report:<file>, commit:<sha> — and feed the reverse traversal.

import (
	"regexp"
	"sort"
	"strings"
)

// TraceRequirement joins a declared requirement with its trace entry (nil
// when the requirement has no trace line yet).
type TraceRequirement struct {
	ID          string      `json:"id"`
	Criticality string      `json:"criticality,omitempty"`
	Text        string      `json:"text"`
	Entry       *TraceEntry `json:"entry,omitempty"`
}

// TraceEntry is one parsed trace bullet.
type TraceEntry struct {
	ID          string   `json:"id"`
	Disposition string   `json:"disposition"` // satisfied | waived | withdrawn
	Reason      string   `json:"reason,omitempty"`
	Evidence    string   `json:"evidence,omitempty"`
	Refs        []string `json:"refs,omitempty"`
}

// RequirementTrace is the bidirectional projection of one spec.
type RequirementTrace struct {
	HasSection   bool               `json:"has_section"`
	Requirements []TraceRequirement `json:"requirements"`
	// ByEvidence maps a structured evidence ref to the requirement IDs it
	// supports (result → requirement traversal).
	ByEvidence map[string][]string `json:"by_evidence,omitempty"`
	Missing    []string            `json:"missing,omitempty"` // declared, untraced
	Orphans    []string            `json:"orphans,omitempty"` // traced, undeclared
	Errors     []string            `json:"errors,omitempty"`  // malformed entries
}

var (
	traceReqLineRE   = regexp.MustCompile(`^\s*-\s*(R\d+)\s*(?:\[(\w+)\])?\s*[:—-]\s*(.*\S)?\s*$`)
	traceEntryRE     = regexp.MustCompile(`^\s*-\s*(R\d+)\s*\[\s*([a-z-]+)\s*(?::\s*([^\]]+?)\s*)?\]\s*(.*?)\s*$`)
	traceBulletRE    = regexp.MustCompile(`^\s*-\s*R\d+`)
	traceRefRE       = regexp.MustCompile(`\b(check|test|report|commit):[^\s,;)\]]+`)
	traceHeadingRE   = regexp.MustCompile(`^##\s+\d+\.\s+(.+?)\s*$`)
	traceSubheadRE   = regexp.MustCompile(`^###\s+(.+?)\s*$`)
	traceCommentRE   = regexp.MustCompile(`(?s)<!--.*?-->`)
	validTraceStatus = map[string]bool{"satisfied": true, "waived": true, "withdrawn": true}
)

// ParseRequirementTrace extracts requirements (section 2) and trace entries
// (section 6, "Requirement trace" subsection) from a spec body.
func ParseRequirementTrace(body string) RequirementTrace {
	body = traceCommentRE.ReplaceAllString(body, "")
	trace := RequirementTrace{ByEvidence: map[string][]string{}}

	section, subsection := "", ""
	declared := []TraceRequirement{}
	declaredIdx := map[string]int{}
	entries := map[string]*TraceEntry{}

	for _, line := range strings.Split(body, "\n") {
		if m := traceHeadingRE.FindStringSubmatch(line); m != nil {
			section = strings.ToLower(strings.TrimSpace(m[1]))
			subsection = ""
			continue
		}
		if m := traceSubheadRE.FindStringSubmatch(line); m != nil {
			subsection = strings.ToLower(strings.TrimSpace(m[1]))
			if strings.HasPrefix(section, "validation") && strings.HasPrefix(subsection, "requirement trace") {
				trace.HasSection = true
			}
			continue
		}
		switch {
		case strings.HasPrefix(section, "requirements"):
			if m := traceReqLineRE.FindStringSubmatch(line); m != nil {
				if _, dup := declaredIdx[m[1]]; !dup {
					declaredIdx[m[1]] = len(declared)
					declared = append(declared, TraceRequirement{ID: m[1], Criticality: m[2], Text: strings.TrimSpace(m[3])})
				}
			}
		case strings.HasPrefix(section, "validation") && strings.HasPrefix(subsection, "requirement trace"):
			if !traceBulletRE.MatchString(line) {
				continue
			}
			m := traceEntryRE.FindStringSubmatch(line)
			if m == nil {
				trace.Errors = append(trace.Errors, "malformed trace bullet: "+strings.TrimSpace(line))
				continue
			}
			id, disposition, reason, evidence := m[1], m[2], strings.TrimSpace(m[3]), strings.TrimSpace(m[4])
			if !validTraceStatus[disposition] {
				trace.Errors = append(trace.Errors, id+": invalid trace disposition ["+disposition+"] (use satisfied|waived|withdrawn)")
				continue
			}
			if disposition != "satisfied" && reason == "" {
				trace.Errors = append(trace.Errors, id+": ["+disposition+"] requires a reason (use ["+disposition+": <reason>])")
				continue
			}
			if disposition == "satisfied" && evidence == "" {
				trace.Errors = append(trace.Errors, id+": [satisfied] requires evidence (verification case, check:, test:, report: or commit: refs)")
				continue
			}
			if _, dup := entries[id]; dup {
				trace.Errors = append(trace.Errors, id+": duplicate trace entry")
				continue
			}
			entry := &TraceEntry{ID: id, Disposition: disposition, Reason: reason, Evidence: evidence}
			for _, ref := range traceRefRE.FindAllString(evidence, -1) {
				entry.Refs = append(entry.Refs, ref)
				trace.ByEvidence[ref] = append(trace.ByEvidence[ref], id)
			}
			entries[id] = entry
		}
	}

	for i := range declared {
		if entry, ok := entries[declared[i].ID]; ok {
			declared[i].Entry = entry
			delete(entries, declared[i].ID)
		} else if trace.HasSection {
			trace.Missing = append(trace.Missing, declared[i].ID)
		}
	}
	for id := range entries {
		trace.Orphans = append(trace.Orphans, id)
	}
	sort.Strings(trace.Missing)
	sort.Strings(trace.Orphans)
	if len(trace.ByEvidence) == 0 {
		trace.ByEvidence = nil
	}
	trace.Requirements = declared
	return trace
}
