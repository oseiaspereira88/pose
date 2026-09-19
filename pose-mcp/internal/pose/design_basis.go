package pose

// Design-basis parsing is deliberately narrow. Markdown remains the source of
// truth; this file only projects the explicitly structured part of a spec's
// Decisions section and validates relationships that can be checked without a
// model, a network call or executing repository content.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DesignBasisSchemaVersion = 1
	DesignBasisParserVersion = "1"
)

var (
	designTopHeadingRE  = regexp.MustCompile(`^##\s+(?:\d+\.\s*)?(.+?)\s*$`)
	designNodeHeadingRE = regexp.MustCompile(`(?i)^(assumption|premissa|decision|decisão)\s+([AD]\d+)(?:\s*[:—-]\s*.*)?$`)
	designIDRE          = regexp.MustCompile(`^[RACD]\d+$`)
	designNodeIDRE      = regexp.MustCompile(`^[AD]\d+$`)
	designRequirementRE = regexp.MustCompile(`^\s*-\s*([RC]\d+)\s*(?:[:—-])`)
	designHTMLCommentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// DesignReference is a typed pointer used by an assumption. Resolution is a
// projection detail and is intentionally excluded from the semantic digest:
// changing local availability must not silently rewrite the decision basis.
type DesignReference struct {
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Resolution string `json:"resolution"` // resolved | declared | unknown | unsupported
}

type DesignAssumption struct {
	ID           string            `json:"id"`
	Claim        string            `json:"claim,omitempty"`
	Status       string            `json:"status,omitempty"`
	Evidence     string            `json:"evidence,omitempty"`
	EvidenceRefs []DesignReference `json:"evidence_refs,omitempty"`
	Scope        string            `json:"scope,omitempty"`
	Affects      []string          `json:"affects,omitempty"`
	Line         int               `json:"line"`
}

type DesignDecision struct {
	ID             string   `json:"id"`
	Basis          []string `json:"basis,omitempty"`
	MinimalOption  string   `json:"minimal_option,omitempty"`
	SelectedOption string   `json:"selected_option,omitempty"`
	Rationale      string   `json:"rationale,omitempty"`
	Consequences   string   `json:"consequences,omitempty"`
	Falsifier      string   `json:"falsifier,omitempty"`
	Line           int      `json:"line"`
}

type DesignBasisDiagnostic struct {
	Severity string `json:"severity"` // error | warning
	Code     string `json:"code"`
	ID       string `json:"id,omitempty"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
}

// DesignBasisReport is the read-only projection consumed by CLI/MCP. A spec
// with no Decisions section is a valid legacy input and returns HasSection=false
// with an empty digest and no diagnostics.
type DesignBasisReport struct {
	SchemaVersion int                     `json:"schema_version"`
	ParserVersion string                  `json:"parser_version"`
	HasSection    bool                    `json:"has_section"`
	Assumptions   []DesignAssumption      `json:"assumptions,omitempty"`
	Decisions     []DesignDecision        `json:"decisions,omitempty"`
	Diagnostics   []DesignBasisDiagnostic `json:"diagnostics,omitempty"`
	Digest        string                  `json:"digest,omitempty"`
}

type designNode struct {
	kind   string
	id     string
	line   int
	fields map[string]string
}

// ParseDesignBasis parses the canonical Decisions section without resolving
// repository paths. Use ValidateDesignBasis when a project root is available.
func ParseDesignBasis(body string) DesignBasisReport {
	return parseDesignBasis(body, "")
}

// ValidateDesignBasis parses and validates the material decision basis. root is
// optional; when present it is used only to resolve local knowledge/report/ADR
// references. No file outside root is ever read.
func ValidateDesignBasis(body, root string) DesignBasisReport {
	return parseDesignBasis(body, root)
}

func parseDesignBasis(body, root string) DesignBasisReport {
	report := DesignBasisReport{
		SchemaVersion: DesignBasisSchemaVersion,
		ParserVersion: DesignBasisParserVersion,
	}
	nodes, hasSection := parseDesignNodes(body)
	report.HasSection = hasSection
	if !hasSection {
		return report
	}

	requirements, constraints := declaredDesignRequirements(body)
	assumptions := map[string]*DesignAssumption{}
	decisions := map[string]*DesignDecision{}
	for _, node := range nodes {
		switch node.kind {
		case "assumption":
			item := assumptionFromNode(node, root)
			report.Assumptions = append(report.Assumptions, item)
			if _, exists := assumptions[item.ID]; exists {
				report.addDiagnostic("error", "duplicate-id", item.ID, item.Line,
					fmt.Sprintf("design basis ID %s is declared more than once", item.ID))
			} else {
				copy := item
				assumptions[item.ID] = &copy
			}
		case "decision":
			item := decisionFromNode(node)
			report.Decisions = append(report.Decisions, item)
			if _, exists := decisions[item.ID]; exists {
				report.addDiagnostic("error", "duplicate-id", item.ID, item.Line,
					fmt.Sprintf("design basis ID %s is declared more than once", item.ID))
			} else {
				copy := item
				decisions[item.ID] = &copy
			}
		}
	}

	for _, item := range report.Assumptions {
		validateAssumption(&report, item, root, requirements, constraints)
	}
	for _, item := range report.Decisions {
		validateDecision(&report, item, assumptions, decisions, requirements, constraints)
	}
	sort.Slice(report.Assumptions, func(i, j int) bool { return report.Assumptions[i].ID < report.Assumptions[j].ID })
	sort.Slice(report.Decisions, func(i, j int) bool { return report.Decisions[i].ID < report.Decisions[j].ID })
	sort.SliceStable(report.Diagnostics, func(i, j int) bool {
		if report.Diagnostics[i].Line != report.Diagnostics[j].Line {
			return report.Diagnostics[i].Line < report.Diagnostics[j].Line
		}
		if report.Diagnostics[i].Code != report.Diagnostics[j].Code {
			return report.Diagnostics[i].Code < report.Diagnostics[j].Code
		}
		return report.Diagnostics[i].ID < report.Diagnostics[j].ID
	})
	report.Digest = designBasisDigest(report)
	return report
}

func (r *DesignBasisReport) addDiagnostic(severity, code, id string, line int, message string) {
	r.Diagnostics = append(r.Diagnostics, DesignBasisDiagnostic{
		Severity: severity,
		Code:     code,
		ID:       id,
		Line:     line,
		Message:  message,
	})
}

func parseDesignNodes(body string) ([]designNode, bool) {
	body = designHTMLCommentRE.ReplaceAllString(body, "")
	lines := strings.Split(body, "\n")
	inDecisions, inFence, hasSection := false, false, false
	var nodes []designNode
	var current *designNode
	field := ""
	finish := func() {
		if current != nil {
			copy := *current
			copy.fields = cloneFields(current.fields)
			nodes = append(nodes, copy)
			current = nil
			field = ""
		}
	}
	for i, raw := range lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := designTopHeadingRE.FindStringSubmatch(raw); m != nil {
			name := normalizeDesignHeading(m[1])
			if name == "decisions" || name == "decisoes" {
				finish()
				inDecisions = true
				hasSection = true
				continue
			}
			if inDecisions {
				finish()
				inDecisions = false
			}
			continue
		}
		if !inDecisions {
			continue
		}
		nodeHeading := strings.TrimSpace(raw)
		if strings.HasPrefix(nodeHeading, "###") {
			nodeHeading = strings.TrimSpace(strings.TrimPrefix(nodeHeading, "###"))
		}
		if m := designNodeHeadingRE.FindStringSubmatch(nodeHeading); m != nil {
			finish()
			kind := "assumption"
			if strings.EqualFold(m[1], "decision") || strings.EqualFold(m[1], "decisão") {
				kind = "decision"
			}
			current = &designNode{kind: kind, id: strings.ToUpper(m[2]), line: lineNumber, fields: map[string]string{}}
			continue
		}
		if current == nil {
			continue
		}
		if key, value, ok := designFieldLine(raw); ok {
			field = key
			current.fields[key] = value
			continue
		}
		if field != "" && trimmed != "" && (strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")) {
			current.fields[field] = strings.TrimSpace(current.fields[field] + " " + trimmed)
		}
	}
	finish()
	return nodes, hasSection
}

func cloneFields(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = strings.TrimSpace(value)
	}
	return out
}

func designFieldLine(raw string) (string, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "-") {
		return "", "", false
	}
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	idx := strings.IndexByte(trimmed, ':')
	if idx <= 0 {
		return "", "", false
	}
	key := normalizeDesignKey(trimmed[:idx])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(trimmed[idx+1:]), true
}

func normalizeDesignHeading(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, ":")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "ã", "a")
	value = strings.ReplaceAll(value, "õ", "o")
	value = strings.ReplaceAll(value, "ç", "c")
	value = strings.ReplaceAll(value, "â", "a")
	value = strings.ReplaceAll(value, "ê", "e")
	value = strings.ReplaceAll(value, "ô", "o")
	value = strings.ReplaceAll(value, "á", "a")
	value = strings.ReplaceAll(value, "é", "e")
	value = strings.ReplaceAll(value, "í", "i")
	value = strings.ReplaceAll(value, "ó", "o")
	value = strings.ReplaceAll(value, "ú", "u")
	return value
}

func normalizeDesignKey(value string) string {
	value = normalizeDesignHeading(value)
	value = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(value)
	return value
}

func assumptionFromNode(node designNode, root string) DesignAssumption {
	claim := firstDesignField(node.fields, "claim", "afirmacao")
	status := firstDesignField(node.fields, "status", "estado")
	evidence := firstDesignField(node.fields, "evidence", "evidencia")
	scope := firstDesignField(node.fields, "scope", "escopo")
	return DesignAssumption{
		ID:           node.id,
		Claim:        claim,
		Status:       strings.ToLower(strings.TrimSpace(status)),
		Evidence:     evidence,
		EvidenceRefs: designReferences(evidence, root),
		Scope:        scope,
		Affects:      designIDList(firstDesignField(node.fields, "affects", "afeta", "impacta")),
		Line:         node.line,
	}
}

func decisionFromNode(node designNode) DesignDecision {
	return DesignDecision{
		ID:             node.id,
		Basis:          designIDList(firstDesignField(node.fields, "basis", "base", "fundamento")),
		MinimalOption:  firstDesignField(node.fields, "minimaloption", "minimal", "opcaominima"),
		SelectedOption: firstDesignField(node.fields, "selectedoption", "selected", "decision", "decisao", "opcaoselecionada"),
		Rationale:      firstDesignField(node.fields, "rationale", "racional", "justificativa"),
		Consequences:   firstDesignField(node.fields, "consequences", "consequencias"),
		Falsifier:      firstDesignField(node.fields, "falsifier", "falsificador", "condicaodefalsificacao"),
		Line:           node.line,
	}
}

func firstDesignField(fields map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fields[key]); value != "" {
			return value
		}
	}
	return ""
}

func declaredDesignRequirements(body string) (map[string]bool, map[string]bool) {
	requirements, constraints := map[string]bool{}, map[string]bool{}
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := designRequirementRE.FindStringSubmatch(line); m != nil {
			if strings.HasPrefix(m[1], "R") {
				requirements[m[1]] = true
			} else {
				constraints[m[1]] = true
			}
		}
	}
	return requirements, constraints
}

func designIDList(value string) []string {
	value = strings.ReplaceAll(value, "`", "")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, ".:()[]{}")
		if part != "" {
			ids = append(ids, strings.ToUpper(part))
		}
	}
	return ids
}

func designReferences(value, root string) []DesignReference {
	var refs []DesignReference
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		part = strings.TrimSpace(strings.Trim(part, "`"))
		if part == "" {
			continue
		}
		kind, refValue, ok := strings.Cut(part, ":")
		kind = strings.ToLower(strings.TrimSpace(kind))
		refValue = strings.TrimSpace(refValue)
		ref := DesignReference{Kind: kind, Value: refValue, Resolution: "unsupported"}
		if !ok || kind == "" || refValue == "" || filepath.IsAbs(refValue) || strings.Contains(refValue, "..") {
			ref.Kind = "invalid"
			ref.Resolution = "unknown"
			refs = append(refs, ref)
			continue
		}
		if kind == "url" || kind == "uri" || kind == "remote" || strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			ref.Resolution = "unknown"
			refs = append(refs, ref)
			continue
		}
		if !knownDesignReferenceKind(kind) {
			ref.Resolution = "unsupported"
			refs = append(refs, ref)
			continue
		}
		if root != "" && designLocalReference(kind) {
			if designReferenceExists(root, kind, refValue) {
				ref.Resolution = "resolved"
			} else {
				ref.Resolution = "unknown"
			}
		} else {
			// Logical references are deliberately not content claims. Their
			// typed identifier plus a scope is the strongest offline state.
			ref.Resolution = "declared"
		}
		refs = append(refs, ref)
	}
	return refs
}

func knownDesignReferenceKind(kind string) bool {
	switch kind {
	case "contract", "check", "test", "report", "knowledge", "adr", "commit", "snapshot", "doc", "evidence", "url", "uri", "remote":
		return true
	default:
		return false
	}
}

func designLocalReference(kind string) bool {
	switch kind {
	case "knowledge", "report", "adr", "doc", "snapshot":
		return true
	default:
		return false
	}
}

func designReferenceExists(root, kind, value string) bool {
	if root == "" || filepath.IsAbs(value) || strings.Contains(value, "..") {
		return false
	}
	var candidates []string
	switch kind {
	case "knowledge":
		candidates = []string{filepath.Join(root, ".pose", "knowledge", value), filepath.Join(root, ".pose", "knowledge", value+".md")}
		candidates = append(candidates, filepath.Join(root, ".pose", "knowledge", "*-"+value+".md"))
	case "report":
		candidates = []string{filepath.Join(root, ".pose", "reports", value), filepath.Join(root, ".pose", "reports", value+".md"), filepath.Join(root, ".pose", "reports", value+".json")}
	case "adr":
		candidates = []string{filepath.Join(root, ".pose", "adr", value), filepath.Join(root, ".pose", "adr", value+".md"), filepath.Join(root, ".pose", "adr", "*-"+value+".md")}
	case "doc":
		candidates = []string{filepath.Join(root, filepath.FromSlash(value))}
	case "snapshot":
		candidates = []string{filepath.Join(root, ".pose", "results", value), filepath.Join(root, ".pose", "results", value+".json"), filepath.Join(root, ".pose", "assessments", value), filepath.Join(root, ".pose", "assessments", value+".json")}
	}
	for _, candidate := range candidates {
		if strings.Contains(candidate, "*") {
			matches, _ := filepath.Glob(candidate)
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && !info.IsDir() {
					return true
				}
			}
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func validateAssumption(report *DesignBasisReport, item DesignAssumption, root string, requirements, constraints map[string]bool) {
	if !designNodeIDRE.MatchString(item.ID) || !strings.HasPrefix(item.ID, "A") {
		report.addDiagnostic("error", "invalid-id", item.ID, item.Line, "assumption ID must match A<N>")
	}
	if item.Claim == "" {
		report.addDiagnostic("error", "missing-claim", item.ID, item.Line, "assumption requires a Claim field")
	}
	switch item.Status {
	case "unverified", "verified", "invalidated", "withdrawn":
	case "":
		report.addDiagnostic("error", "missing-status", item.ID, item.Line, "assumption requires Status: unverified|verified|invalidated|withdrawn")
	default:
		report.addDiagnostic("error", "invalid-status", item.ID, item.Line, fmt.Sprintf("unknown assumption status %q", item.Status))
	}
	for _, id := range item.Affects {
		if !designIDRE.MatchString(id) || (!strings.HasPrefix(id, "R") && !strings.HasPrefix(id, "C")) {
			report.addDiagnostic("error", "invalid-affects-ref", item.ID, item.Line, fmt.Sprintf("assumption affects ref %q is not R<N> or C<N>", id))
			continue
		}
		if strings.HasPrefix(id, "R") && !requirements[id] || strings.HasPrefix(id, "C") && !constraints[id] {
			report.addDiagnostic("error", "orphan-ref", item.ID, item.Line, fmt.Sprintf("assumption affects undeclared %s", id))
		}
	}
	if item.Status != "verified" {
		return
	}
	if strings.TrimSpace(item.Evidence) == "" {
		report.addDiagnostic("error", "verified-without-evidence", item.ID, item.Line, "verified assumption requires Evidence")
	}
	if strings.TrimSpace(item.Scope) == "" {
		report.addDiagnostic("error", "verified-without-scope", item.ID, item.Line, "verified assumption requires Scope")
	}
	if len(item.EvidenceRefs) == 0 {
		report.addDiagnostic("error", "invalid-evidence-ref", item.ID, item.Line, "Evidence must contain a typed reference such as contract:name or test:name")
	}
	for _, ref := range item.EvidenceRefs {
		if ref.Kind == "invalid" {
			report.addDiagnostic("error", "invalid-evidence-ref", item.ID, item.Line, "Evidence contains an invalid or unsafe reference")
			continue
		}
		switch ref.Resolution {
		case "unknown":
			report.addDiagnostic("error", "evidence-unknown-offline", item.ID, item.Line, fmt.Sprintf("verified evidence %s:%s is unknown offline", ref.Kind, ref.Value))
		case "unsupported":
			report.addDiagnostic("error", "unsupported-evidence-kind", item.ID, item.Line, fmt.Sprintf("unsupported evidence reference kind %q", ref.Kind))
		}
	}
}

func validateDecision(report *DesignBasisReport, item DesignDecision, assumptions map[string]*DesignAssumption, decisions map[string]*DesignDecision, requirements, constraints map[string]bool) {
	if !designNodeIDRE.MatchString(item.ID) || !strings.HasPrefix(item.ID, "D") {
		report.addDiagnostic("error", "invalid-id", item.ID, item.Line, "decision ID must match D<N>")
	}
	if len(item.Basis) == 0 {
		report.addDiagnostic("error", "missing-basis", item.ID, item.Line, "decision requires Basis refs")
	}
	hasRequirementPath := false
	for _, id := range item.Basis {
		if !designIDRE.MatchString(id) {
			report.addDiagnostic("error", "invalid-basis-ref", item.ID, item.Line, fmt.Sprintf("decision basis ref %q is not R<N>, A<N> or C<N>", id))
			continue
		}
		switch {
		case strings.HasPrefix(id, "R"):
			if !requirements[id] {
				report.addDiagnostic("error", "orphan-ref", item.ID, item.Line, fmt.Sprintf("decision basis references undeclared %s", id))
			} else {
				hasRequirementPath = true
			}
		case strings.HasPrefix(id, "C"):
			if !constraints[id] {
				report.addDiagnostic("error", "orphan-ref", item.ID, item.Line, fmt.Sprintf("decision basis references undeclared %s", id))
			} else {
				hasRequirementPath = true
			}
		case strings.HasPrefix(id, "A"):
			assumption, ok := assumptions[id]
			if !ok {
				report.addDiagnostic("error", "orphan-ref", item.ID, item.Line, fmt.Sprintf("decision basis references undeclared %s", id))
				continue
			}
			for _, affected := range assumption.Affects {
				if strings.HasPrefix(affected, "R") && requirements[affected] || strings.HasPrefix(affected, "C") && constraints[affected] {
					hasRequirementPath = true
				}
			}
		case strings.HasPrefix(id, "D"):
			// D→D would make a graph cycle/hidden dependency. Decisions may
			// depend on requirements, constraints or assumptions only.
			if _, exists := decisions[id]; exists {
				report.addDiagnostic("error", "decision-cycle", item.ID, item.Line, fmt.Sprintf("decision basis cannot reference decision %s", id))
			} else {
				report.addDiagnostic("error", "orphan-ref", item.ID, item.Line, fmt.Sprintf("decision basis references undeclared %s", id))
			}
		}
	}
	if len(item.Basis) > 0 && !hasRequirementPath {
		report.addDiagnostic("error", "no-requirement-path", item.ID, item.Line, "material decision basis must reach a declared requirement or constraint")
	}
	for key, value := range map[string]string{
		"minimal-option":  item.MinimalOption,
		"selected-option": item.SelectedOption,
		"rationale":       item.Rationale,
		"consequences":    item.Consequences,
		"falsifier":       item.Falsifier,
	} {
		if strings.TrimSpace(value) == "" {
			report.addDiagnostic("warning", "incomplete-decision", item.ID, item.Line, fmt.Sprintf("decision is missing %s; add it when the decision is material", key))
		} else if designPlaceholder(value) {
			report.addDiagnostic("warning", "placeholder", item.ID, item.Line, fmt.Sprintf("decision %s still contains a placeholder", key))
		}
	}
}

func designPlaceholder(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "<todo") || strings.Contains(value, "tbd") || strings.Contains(value, "<fill") || strings.Contains(value, "...")
}

func designBasisDigest(report DesignBasisReport) string {
	type digestView struct {
		SchemaVersion int                `json:"schema_version"`
		ParserVersion string             `json:"parser_version"`
		Assumptions   []DesignAssumption `json:"assumptions,omitempty"`
		Decisions     []DesignDecision   `json:"decisions,omitempty"`
	}
	view := digestView{
		SchemaVersion: report.SchemaVersion,
		ParserVersion: report.ParserVersion,
		Assumptions:   append([]DesignAssumption(nil), report.Assumptions...),
		Decisions:     append([]DesignDecision(nil), report.Decisions...),
	}
	for i := range view.Assumptions {
		view.Assumptions[i].Line = 0
		view.Assumptions[i].EvidenceRefs = append([]DesignReference(nil), view.Assumptions[i].EvidenceRefs...)
		for j := range view.Assumptions[i].EvidenceRefs {
			view.Assumptions[i].EvidenceRefs[j].Resolution = ""
		}
	}
	for i := range view.Decisions {
		view.Decisions[i].Line = 0
	}
	encoded, _ := json.Marshal(view)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
