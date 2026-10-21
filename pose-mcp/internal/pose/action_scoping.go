package pose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParseActionEffect accepts the legacy phase:mode spelling or an additive
// structured effect. Both transports use this parser and PrepareActionRequest
// qualifies and validates its material references.
func ParseActionEffect(raw json.RawMessage) (ObligationEffect, error) {
	var legacy string
	if json.Unmarshal(raw, &legacy) == nil {
		phase, mode, ok := strings.Cut(legacy, ":")
		if !ok || !obligationPhases[phase] || !obligationModes[mode] {
			return ObligationEffect{}, fmt.Errorf("effect must be <phase>:<block|advisory>")
		}
		return ObligationEffect{Phase: phase, Mode: mode}, nil
	}
	var input struct {
		Phase string   `json:"phase"`
		Mode  string   `json:"mode"`
		Scope []string `json:"scope"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return ObligationEffect{}, fmt.Errorf("invalid structured effect: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return ObligationEffect{}, fmt.Errorf("structured effect must be one JSON object")
	}
	if !obligationPhases[input.Phase] || !obligationModes[input.Mode] {
		return ObligationEffect{}, fmt.Errorf("unknown effect phase or mode")
	}
	out := ObligationEffect{Phase: input.Phase, Mode: input.Mode}
	for _, raw := range input.Scope {
		n, err := ParseActionTarget(raw)
		if err != nil {
			return ObligationEffect{}, err
		}
		out.Scope = append(out.Scope, n)
	}
	return out, nil
}

// actionMaterial resolves exactly the named artifact/node. Foreign reads must
// pass the caller's existing resolver authorization; no origin fallback exists.
func (s Store) actionMaterial(project string, node NodeRef) (Store, ArtifactRef, string, error) {
	ref, err := ParseArtifactRef(node.Artifact)
	if err != nil {
		return Store{}, ref, "", err
	}
	source := s
	if ref.Project != project {
		var resolver ArtifactResolver
		if s.FederatedResolver != nil {
			resolver = *s.FederatedResolver
		} else {
			resolver, _, err = EnvironmentArtifactResolver(s.Root, "")
			if err != nil {
				return Store{}, ref, "", err
			}
		}
		resolved := resolver.Resolve(project, ref.String())
		if !resolved.Resolved {
			return Store{}, ref, "", fmt.Errorf("material %s: %s", ref.String(), resolved.State)
		}
		if resolved.CanonicalIdentity != nil {
			ref = *resolved.CanonicalIdentity
		}
		source, err = resolver.Roots.StoreFor(ref.Project)
		if err != nil {
			return Store{}, ref, "", err
		}
	}
	if ref.Kind != "spec" {
		if node.Kind != "" || node.ID != "" {
			return Store{}, ref, "", fmt.Errorf("material node needs a spec artifact")
		}
		if err := source.actionOriginExists(ref); err != nil {
			return Store{}, ref, "", err
		}
		return source, ref, "", nil
	}
	spec, err := source.GetSpec(ref.Slug)
	if err != nil {
		return Store{}, ref, "", err
	}
	if node.Kind == "" && node.ID == "" {
		return source, ref, digestBytes([]byte(spec.Body)), nil
	}
	for _, n := range ProjectContractNodes(spec.Slug, spec.Body).Nodes {
		if n.Kind == node.Kind && n.ID == node.ID {
			return source, ref, n.Hash, nil
		}
	}
	return Store{}, ref, "", fmt.Errorf("material node %s:%s not found in %s", node.Kind, node.ID, ref.String())
}

// actionDestinations expands only the named artifact. Unknown membership never
// becomes all local specs. Foreign destinations remain qualified observations;
// this local operation does not infer authority over their members.
func (s Store) actionDestinations(project string, node NodeRef) ([]NodeRef, error) {
	ref, err := ParseArtifactRef(node.Artifact)
	if err != nil {
		return nil, err
	}
	_, _, _, err = s.actionMaterial(project, node)
	if err != nil {
		// A persisted restriction still names this local artifact even if its node
		// disappeared. Keep that named restriction and disclose the missing material.
		if ref.Project == project && ref.Kind == "spec" {
			return []NodeRef{node}, err
		}
		return nil, err
	}
	if ref.Project != project || ref.Kind == "spec" {
		return []NodeRef{node}, nil
	}
	if ref.Kind != "roadmap" && ref.Kind != "milestone" {
		return nil, fmt.Errorf("unsupported action destination %s", node.Artifact)
	}
	rm, err := s.GetRoadmap(ref.Slug)
	if err != nil {
		return nil, err
	}
	found := ref.Kind == "roadmap"
	var out []NodeRef
	var limitations []string
	for _, m := range rm.Milestones {
		if ref.Kind == "milestone" && m.ID != ref.Milestone {
			continue
		}
		found = true
		for _, raw := range m.Specs {
			member, e := ParseArtifactRef(raw)
			if e != nil {
				return nil, e
			}
			if member.Project == "" {
				member.Project = project
			}
			if member.Kind != "spec" {
				return nil, fmt.Errorf("invalid action member %s", raw)
			}
			n := NodeRef{Artifact: member.String()}
			if _, _, _, e = s.actionMaterial(project, n); e != nil {
				limitations = append(limitations, e.Error())
				if member.Project != project {
					continue
				}
			}
			out = append(out, n)
		}
	}
	if !found {
		return nil, fmt.Errorf("milestone %s not found", ref.String())
	}
	if len(limitations) > 0 {
		return out, fmt.Errorf("%s", strings.Join(limitations, "; "))
	}
	return out, nil
}

// scopedActionObligations is shared by queries and transition gates. It keeps
// provenance and targets intact but returns only effects applicable to the
// selected specs. Global queries also retain terminal-origin requests.
func (s Store) scopedActionObligations(project string, specs []Spec, global bool, coverage *Coverage) ([]Obligation, error) {
	views, err := s.ListActionRequests()
	if err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	for _, sp := range specs {
		selected[specNode(project, sp.Slug).Artifact] = true
	}
	var out []Obligation
	for _, v := range views {
		r := v.Request
		if r.Project != project || v.State == ActionStateSuperseded {
			continue
		}
		o := actionObligation(project, v)
		o.Effects = nil
		relevant := global
		var limits []string
		unknownMembership := false
		expand := func(n NodeRef) []NodeRef {
			nodes, e := s.actionDestinations(project, n)
			if e != nil {
				limits = append(limits, e.Error())
				ref, parseErr := ParseArtifactRef(n.Artifact)
				if parseErr == nil && ref.Project == project && (ref.Kind == "roadmap" || ref.Kind == "milestone") {
					unknownMembership = true
				}
			}
			return nodes
		}
		for _, target := range r.Targets {
			for _, n := range expand(target) {
				if selected[n.Artifact] {
					relevant = true
				}
			}
		}
		for _, effect := range r.Effects {
			scopes := effect.Scope
			if len(scopes) == 0 {
				scopes = []NodeRef{{Artifact: r.Origin}}
			}
			var applicable []NodeRef
			for _, scope := range scopes {
				for _, n := range expand(scope) {
					if global || selected[n.Artifact] {
						applicable = append(applicable, n)
						relevant = true
					}
				}
			}
			if len(applicable) > 0 {
				effect.Scope = applicable
				o.Effects = append(o.Effects, effect)
			}
		}
		if !relevant {
			if unknownMembership && coverage != nil {
				markCoverageIncomplete(coverage, strings.Join(uniqueSorted(limits), "; "))
			}
			continue
		}
		if len(limits) > 0 {
			o.Observation.Coverage = CoveragePartial
			o.Observation.Limitations = append(o.Observation.Limitations, uniqueSorted(limits)...)
			if coverage != nil {
				markCoverageIncomplete(coverage, strings.Join(uniqueSorted(limits), "; "))
			}
		}
		out = append(out, o)
	}
	return out, nil
}
