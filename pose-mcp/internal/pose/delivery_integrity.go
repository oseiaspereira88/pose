package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const DeliveryIntegritySchemaVersion = 1

type ArtifactPolicy struct {
	SchemaVersion int               `json:"schema_version"`
	Enabled       bool              `json:"enabled"`
	AdoptedAt     string            `json:"adopted_at,omitempty"`
	GovernedRoots []string          `json:"governed_roots"`
	Exclusions    []string          `json:"exclusions,omitempty"`
	NoneReasons   []string          `json:"none_reasons,omitempty"`
	Severities    map[string]string `json:"severities,omitempty"`
}

type ArtifactClaim struct {
	Spec    string `json:"spec"`
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type ObservedPath struct {
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
}

type ChangeSet struct {
	ID           string         `json:"id"`
	Spec         string         `json:"spec,omitempty"`
	Selector     string         `json:"selector"`
	Base         string         `json:"base"`
	Head         string         `json:"head"`
	ResolvedBase string         `json:"resolved_base"`
	ResolvedHead string         `json:"resolved_head"`
	Commits      []string       `json:"commits,omitempty"`
	Paths        []ObservedPath `json:"paths"`
	DiffDigest   string         `json:"diff_digest"`
}

type DeliveryIntegrityFinding struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Spec        string `json:"spec,omitempty"`
	Path        string `json:"path,omitempty"`
	ChangeSet   string `json:"change_set,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

type DeliveryIntegrityNode struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type DeliveryIntegrityEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type DeliveryIntegrityGraph struct {
	SchemaVersion     int                        `json:"schema_version"`
	InputDigest       string                     `json:"input_digest"`
	ProvenanceDigest  string                     `json:"provenance_digest,omitempty"`
	Nodes             []DeliveryIntegrityNode    `json:"nodes"`
	Edges             []DeliveryIntegrityEdge    `json:"edges"`
	Claims            []ArtifactClaim            `json:"claims"`
	ChangeSets        []ChangeSet                `json:"change_sets"`
	Reverse           map[string][]string        `json:"reverse"`
	Findings          []DeliveryIntegrityFinding `json:"findings"`
	Deliveries        []DeliveryTarget           `json:"deliveries,omitempty"`
	ValidationResults []DeliveryValidationResult `json:"validation_results,omitempty"`
	RoadmapCriteria   []RoadmapCriterion         `json:"roadmap_criteria,omitempty"`
	Paths             map[string][]string        `json:"paths,omitempty"`
}

func LoadArtifactPolicy(root string) (ArtifactPolicy, error) {
	path := filepath.Join(root, ".pose", "policy", "artifacts.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ArtifactPolicy{}, nil
	}
	if err != nil {
		return ArtifactPolicy{}, err
	}
	var policy ArtifactPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return ArtifactPolicy{}, fmt.Errorf("pose: invalid artifact policy: %w", err)
	}
	if policy.SchemaVersion != DeliveryIntegritySchemaVersion {
		return ArtifactPolicy{}, fmt.Errorf("pose: unsupported artifact policy schema %d", policy.SchemaVersion)
	}
	for _, path := range append(append([]string{}, policy.GovernedRoots...), policy.Exclusions...) {
		if err := ValidateArtifactPath(root, path, true); err != nil {
			return ArtifactPolicy{}, fmt.Errorf("pose: invalid artifact policy path %q: %w", path, err)
		}
	}
	return policy, nil
}

func ValidateArtifactPath(root, value string, allowDirectory bool) error {
	clean, err := validateArtifactPathSyntax(value)
	if err != nil {
		return err
	}
	joined := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes project root")
	}
	if info, err := os.Lstat(joined); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(joined)
			if err != nil {
				return fmt.Errorf("cannot resolve symlink")
			}
			targetRel, err := filepath.Rel(root, target)
			if err != nil || targetRel == ".." || strings.HasPrefix(targetRel, ".."+string(os.PathSeparator)) {
				return fmt.Errorf("symlink escapes project root")
			}
		}
		if info.IsDir() && !allowDirectory {
			return fmt.Errorf("directories are not valid artifact claims")
		}
	}
	return nil
}

func validateArtifactPathSyntax(value string) (string, error) {
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "*?[]{}") {
		return "", fmt.Errorf("path must be an exact project-relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes project root")
	}
	return clean, nil
}

func ParseArtifactClaims(spec Spec, policy ArtifactPolicy) ([]ArtifactClaim, bool, error) {
	lines := strings.Split(strings.ReplaceAll(spec.Body, "\r\n", "\n"), "\n")
	inSection, found := false, false
	claims := []ArtifactClaim{}
	allowedReasons := map[string]bool{"analysis": true, "decision": true, "documentation": true, "governance": true, "spike": true}
	for _, reason := range policy.NoneReasons {
		allowedReasons[reason] = true
	}
	seen := map[string]string{}
	hasNone := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "### Artifacts" {
			inSection, found = true, true
			continue
		}
		if inSection && strings.HasPrefix(line, "### ") {
			break
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(line, "- ")), ":", 2)
		if len(parts) != 2 {
			return nil, found, fmt.Errorf("pose: spec %s has malformed artifact claim %q", spec.Slug, line)
		}
		action, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		claim := ArtifactClaim{Spec: spec.Slug, Action: action}
		switch action {
		case "created", "modified", "removed":
			if hasNone {
				return nil, found, fmt.Errorf("pose: spec %s mixes none with artifact actions", spec.Slug)
			}
			if _, err := validateArtifactPathSyntax(value); err != nil {
				return nil, found, fmt.Errorf("pose: spec %s artifact %q: %w", spec.Slug, value, err)
			}
			claim.Path = filepath.ToSlash(filepath.Clean(value))
		case "renamed":
			if hasNone {
				return nil, found, fmt.Errorf("pose: spec %s mixes none with artifact actions", spec.Slug)
			}
			pair := strings.Split(value, "->")
			if len(pair) != 2 {
				return nil, found, fmt.Errorf("pose: spec %s rename requires old/path -> new/path", spec.Slug)
			}
			claim.OldPath, claim.NewPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(pair[0]))), filepath.ToSlash(filepath.Clean(strings.TrimSpace(pair[1])))
			_, oldErr := validateArtifactPathSyntax(claim.OldPath)
			_, newErr := validateArtifactPathSyntax(claim.NewPath)
			if oldErr != nil || newErr != nil || claim.OldPath == claim.NewPath {
				return nil, found, fmt.Errorf("pose: spec %s has invalid rename paths", spec.Slug)
			}
		case "none":
			if len(claims) > 0 || hasNone || !allowedReasons[value] {
				return nil, found, fmt.Errorf("pose: spec %s has invalid none artifact reason %q", spec.Slug, value)
			}
			hasNone = true
			claim.Reason = value
		default:
			return nil, found, fmt.Errorf("pose: spec %s has unknown artifact action %q", spec.Slug, action)
		}
		key := claim.Path
		if action == "renamed" {
			key = claim.OldPath + "->" + claim.NewPath
		}
		if previous, duplicate := seen[key]; duplicate {
			return nil, found, fmt.Errorf("pose: spec %s has duplicate or contradictory artifact actions %s/%s for %s", spec.Slug, previous, action, key)
		}
		seen[key] = action
		claims = append(claims, claim)
	}
	return claims, found, nil
}

func BuildDeliveryIntegrity(specs []Spec, claims []ArtifactClaim, changeSets []ChangeSet, tracked []string, policy ArtifactPolicy) DeliveryIntegrityGraph {
	graph := DeliveryIntegrityGraph{SchemaVersion: DeliveryIntegritySchemaVersion, Claims: append([]ArtifactClaim{}, claims...), ChangeSets: append([]ChangeSet{}, changeSets...), Reverse: map[string][]string{}, Nodes: []DeliveryIntegrityNode{}, Edges: []DeliveryIntegrityEdge{}, Findings: []DeliveryIntegrityFinding{}}
	sort.Slice(graph.Claims, func(i, j int) bool { return claimKey(graph.Claims[i]) < claimKey(graph.Claims[j]) })
	sort.Slice(graph.ChangeSets, func(i, j int) bool { return graph.ChangeSets[i].ID < graph.ChangeSets[j].ID })
	claimBySpec := map[string][]ArtifactClaim{}
	claimedPaths := map[string]bool{}
	trackedSet := map[string]bool{}
	for _, path := range tracked {
		trackedSet[filepath.ToSlash(path)] = true
	}
	for _, claim := range graph.Claims {
		claimBySpec[claim.Spec] = append(claimBySpec[claim.Spec], claim)
		if claim.Action == "none" {
			continue
		}
		paths := []string{claim.Path}
		if claim.Action == "renamed" {
			paths = []string{claim.OldPath, claim.NewPath}
		}
		for _, path := range paths {
			claimedPaths[path] = true
			graph.Reverse[path] = appendUnique(graph.Reverse[path], claim.Spec)
			graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "artifact:" + path, Type: "artifact", Attributes: map[string]string{"path": path}})
			graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "spec:" + claim.Spec, To: "artifact:" + path, Type: "declares"})
		}
		existencePath := claim.Path
		if claim.Action == "renamed" {
			existencePath = claim.NewPath
		}
		if len(tracked) > 0 && (claim.Action == "created" || claim.Action == "modified" || claim.Action == "renamed") && !trackedSet[existencePath] {
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("existence", severity(policy, "existence"), claim.Spec, existencePath, "", "declared current artifact is not tracked at the selected head", "correct the exact path or commit the artifact before closeout"))
		}
	}
	for _, spec := range specs {
		graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "spec:" + spec.Slug, Type: "spec", Attributes: map[string]string{"status": spec.Status}})
		if _, ok := claimBySpec[spec.Slug]; !ok {
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("legacy-narrative", severity(policy, "legacy-narrative"), spec.Slug, "", "", "spec has no structured Artifacts section", "add a machine-parseable ### Artifacts section"))
		}
	}
	observedBySpec := map[string]map[string]bool{}
	for _, set := range graph.ChangeSets {
		if observedBySpec[set.Spec] == nil {
			observedBySpec[set.Spec] = map[string]bool{}
		}
		for _, path := range set.Paths {
			observedBySpec[set.Spec][observedKey(path)] = true
		}
	}
	for spec, declared := range claimBySpec {
		for _, claim := range declared {
			if claim.Action == "none" || observedBySpec[spec][claimKeyNoSpec(claim)] {
				continue
			}
			path := claim.Path
			if claim.Action == "renamed" {
				path = claim.OldPath + " -> " + claim.NewPath
			}
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("action-mismatch", severity(policy, "action-mismatch"), spec, path, "", "declared artifact action is absent from the attributed Git change sets", "correct the action or record the exact attributed revisions"))
		}
	}
	for _, set := range graph.ChangeSets {
		graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "change-set:" + set.ID, Type: "change-set", Attributes: map[string]string{"selector": set.Selector, "diff_digest": set.DiffDigest}})
		if set.Spec != "" {
			graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "spec:" + set.Spec, To: "change-set:" + set.ID, Type: "observed-by"})
		}
		observed := map[string]ObservedPath{}
		for _, path := range set.Paths {
			key := observedKey(path)
			observed[key] = path
			for _, value := range observedPaths(path) {
				graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "artifact:" + value, Type: "artifact", Attributes: map[string]string{"path": value}})
				graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "change-set:" + set.ID, To: "artifact:" + value, Type: "changes"})
			}
		}
		declared := claimBySpec[set.Spec]
		for _, path := range set.Paths {
			if !claimMatchesObserved(declared, path) && governedPath(firstObservedPath(path), policy) {
				graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("undeclared", severity(policy, "undeclared"), set.Spec, firstObservedPath(path), set.ID, "Git observed a governed path not declared by the spec", "declare the exact action or narrow the attributed change set"))
			}
		}
	}
	for _, path := range tracked {
		path = filepath.ToSlash(path)
		if governedPath(path, policy) && !claimedPaths[path] {
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("orphan", severity(policy, "orphan"), "", path, "", "governed tracked path has no spec provenance claim", "backfill an explicit claim or add an exact reviewed exclusion"))
		}
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })
	sort.Slice(graph.Edges, func(i, j int) bool {
		a, b := graph.Edges[i], graph.Edges[j]
		return a.From+"\x00"+a.Type+"\x00"+a.To < b.From+"\x00"+b.Type+"\x00"+b.To
	})
	sort.Slice(graph.Findings, func(i, j int) bool { return graph.Findings[i].ID < graph.Findings[j].ID })
	for path := range graph.Reverse {
		sort.Strings(graph.Reverse[path])
	}
	input := struct {
		Claims     []ArtifactClaim `json:"claims"`
		ChangeSets []ChangeSet     `json:"change_sets"`
		Tracked    []string        `json:"tracked"`
		Policy     ArtifactPolicy  `json:"policy"`
	}{graph.Claims, graph.ChangeSets, append([]string{}, tracked...), policy}
	sort.Strings(input.Tracked)
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	graph.InputDigest = "sha256:" + hex.EncodeToString(sum[:])
	provenanceRaw, _ := json.Marshal(struct {
		Claims     []ArtifactClaim `json:"claims"`
		ChangeSets []ChangeSet     `json:"change_sets"`
	}{graph.Claims, graph.ChangeSets})
	provenanceSum := sha256.Sum256(provenanceRaw)
	graph.ProvenanceDigest = "sha256:" + hex.EncodeToString(provenanceSum[:])
	return graph
}

func (s Store) GetDeliveryIntegrity(path string) (DeliveryIntegrityGraph, error) {
	indexPath := filepath.Join(s.Root, ".pose", "indexes", "delivery-integrity.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return DeliveryIntegrityGraph{}, fmt.Errorf("pose: delivery-integrity index not found; run `pose index`")
	}
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil || graph.SchemaVersion != DeliveryIntegritySchemaVersion {
		return DeliveryIntegrityGraph{}, fmt.Errorf("pose: invalid delivery-integrity index")
	}
	if path != "" {
		if err := ValidateArtifactPath(s.Root, path, false); err != nil {
			return DeliveryIntegrityGraph{}, err
		}
		wanted := filepath.ToSlash(filepath.Clean(path))
		graph.Reverse = map[string][]string{wanted: graph.Reverse[wanted]}
		filtered := graph.Findings[:0]
		for _, finding := range graph.Findings {
			if finding.Path == wanted {
				filtered = append(filtered, finding)
			}
		}
		graph.Findings = filtered
	}
	return graph, nil
}

func claimKey(c ArtifactClaim) string { return c.Spec + "\x00" + claimKeyNoSpec(c) }
func claimKeyNoSpec(c ArtifactClaim) string {
	if c.Action == "renamed" {
		return c.Action + "\x00" + c.OldPath + "\x00" + c.NewPath
	}
	return c.Action + "\x00" + c.Path
}
func observedKey(p ObservedPath) string {
	if p.Action == "renamed" {
		return p.Action + "\x00" + p.OldPath + "\x00" + p.NewPath
	}
	return p.Action + "\x00" + p.Path
}
func observedPaths(p ObservedPath) []string {
	if p.Action == "renamed" {
		return []string{p.OldPath, p.NewPath}
	}
	return []string{p.Path}
}
func firstObservedPath(p ObservedPath) string {
	if p.Action == "renamed" {
		return p.NewPath
	}
	return p.Path
}
func claimMatchesObserved(claims []ArtifactClaim, observed ObservedPath) bool {
	for _, claim := range claims {
		if claimKeyNoSpec(claim) == observedKey(observed) {
			return true
		}
	}
	return false
}
func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
func appendNode(nodes []DeliveryIntegrityNode, node DeliveryIntegrityNode) []DeliveryIntegrityNode {
	for _, existing := range nodes {
		if existing.ID == node.ID {
			return nodes
		}
	}
	return append(nodes, node)
}
func severity(policy ArtifactPolicy, code string) string {
	if value := policy.Severities[code]; value != "" {
		return value
	}
	return "warning"
}
func NewDeliveryIntegrityFinding(code, severity, spec, path, set, message, remediation string) DeliveryIntegrityFinding {
	sum := sha256.Sum256([]byte(code + "\x00" + spec + "\x00" + path + "\x00" + set))
	return DeliveryIntegrityFinding{ID: "dif-" + hex.EncodeToString(sum[:6]), Code: code, Severity: severity, Spec: spec, Path: path, ChangeSet: set, Message: message, Remediation: remediation}
}
func governedPath(path string, policy ArtifactPolicy) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	for _, exclusion := range policy.Exclusions {
		exclusion = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(exclusion)), "/")
		if path == exclusion || strings.HasPrefix(path, exclusion+"/") {
			return false
		}
	}
	if len(policy.GovernedRoots) == 0 {
		return false
	}
	for _, root := range policy.GovernedRoots {
		root = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(root)), "/")
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}
