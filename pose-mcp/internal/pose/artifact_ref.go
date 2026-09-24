package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ArtifactRef is identity, not evidence. Project is empty only for an unbound
// local reference. Milestone identities retain their owning roadmap.
type ArtifactRef struct {
	Project   string `json:"project_id,omitempty"`
	Kind      string `json:"kind"`
	Slug      string `json:"slug"`
	Milestone string `json:"milestone,omitempty"`
}

func (r ArtifactRef) String() string {
	local := r.Kind + ":" + r.Slug
	if r.Milestone != "" {
		local += "/" + r.Milestone
	}
	if r.Project != "" {
		return "xref:" + r.Project + "/" + local
	}
	return local
}

// ParseArtifactRef is the shared grammar for dependency and projection readers.
// Never clean a path-like input: rejecting traversal is part of the contract.
func ParseArtifactRef(raw string) (ArtifactRef, error) {
	r := ArtifactRef{Kind: "spec"}
	bad := func() (ArtifactRef, error) { return ArtifactRef{}, errors.New("invalid-artifact-reference") }
	if len(raw) == 0 || len(raw) > 1024 || strings.TrimSpace(raw) != raw {
		return bad()
	}
	if strings.HasPrefix(raw, "xref:") {
		var ok bool
		r.Project, raw, ok = strings.Cut(strings.TrimPrefix(raw, "xref:"), "/")
		if !ok || ValidateSlug(r.Project) != nil {
			return bad()
		}
	}
	if kind, rest, ok := strings.Cut(raw, ":"); ok {
		r.Kind, raw = kind, rest
	}
	switch r.Kind {
	case "spec", "roadmap":
		r.Slug = raw
	case "milestone":
		var ok bool
		r.Slug, r.Milestone, ok = strings.Cut(raw, "/")
		if !ok || ValidateSlug(r.Milestone) != nil {
			return bad()
		}
	default:
		return bad()
	}
	if ValidateSlug(r.Slug) != nil {
		return bad()
	}
	return r, nil
}

// ArtifactResolution intentionally contains no paths or artifact bodies. Digest
// describes observed source content; it does not certify a review or closeout.
type ArtifactResolution struct {
	SchemaVersion     int           `json:"schema_version"`
	Identity          ArtifactRef   `json:"identity"`
	CanonicalIdentity *ArtifactRef  `json:"canonical_identity,omitempty"`
	Redirected        bool          `json:"redirected,omitempty"`
	Resolved          bool          `json:"resolved"`
	State             string        `json:"resolution_state"`
	OperationID       string        `json:"operation_id,omitempty"`
	Status            string        `json:"status,omitempty"`
	Revision          string        `json:"source_revision,omitempty"`
	Digest            string        `json:"source_digest,omitempty"`
	Dependencies      []ArtifactRef `json:"dependencies,omitempty"`
}

type ArtifactResolver struct {
	Roots *Roots
	// Authorize runs BEFORE registry lookup, including for transitive refs.
	// nil is the trusted local CLI boundary: explicitly configured roots only.
	Authorize func(projectID string) bool
}

func (r ArtifactResolver) Resolve(localProject, raw string) ArtifactResolution {
	return r.resolve(localProject, raw, map[string]bool{}, 0)
}

func (r ArtifactResolver) resolve(localProject, raw string, redirectPath map[string]bool, redirectDepth int) ArtifactResolution {
	out := ArtifactResolution{SchemaVersion: 1}
	ref, err := ParseArtifactRef(raw)
	if err != nil {
		out.State = "invalid-artifact-reference"
		return out
	}
	if ref.Project == "" {
		ref.Project = localProject
	}
	out.Identity = ref
	if r.Authorize != nil && !r.Authorize(ref.Project) {
		out.State = "unauthorized-project"
		return out
	}
	if r.Roots == nil {
		out.State = "unknown-project"
		return out
	}
	s, err := r.Roots.StoreFor(ref.Project)
	if err != nil {
		out.State = "unknown-project"
		var conflict ProjectBindingConflictError
		var ambiguous ProjectAmbiguousError
		if errors.As(err, &conflict) || errors.As(err, &ambiguous) {
			out.State = "conflicting-project-binding"
		}
		return out
	}
	if info, err := os.Stat(filepath.Join(s.Root, ".pose")); err != nil || !info.IsDir() {
		out.State = "unavailable-project"
		return out
	}
	if operationID, pending := incompleteTransferFor(s.Root, ref); pending {
		out.State = "transfer-in-progress"
		out.OperationID = operationID
		return out
	}
	policyPath := filepath.Join(s.Root, ".pose", "policy", "review.json")
	if _, err := os.Lstat(policyPath); err == nil && !artifactPathWithin(s.Root, policyPath) {
		out.State = "unsupported-artifact-contract"
		return out
	}
	if raw, err := os.ReadFile(policyPath); err == nil {
		var policy struct {
			SchemaVersion   int `json:"schema_version"`
			RefsVersion     int `json:"qualified_artifact_refs_version"`
			TransferVersion int `json:"spec_authority_transfer_version"`
		}
		if json.Unmarshal(raw, &policy) != nil {
			out.State = "unsupported-artifact-contract"
			return out
		}
		refsRequired := policy.SchemaVersion == QualifiedArtifactPolicySchemaVersion || policy.SchemaVersion == SpecAuthorityTransferPolicySchemaVersion
		transferValid := policy.SchemaVersion == SpecAuthorityTransferPolicySchemaVersion && policy.TransferVersion == 1 || policy.SchemaVersion != SpecAuthorityTransferPolicySchemaVersion && policy.TransferVersion == 0
		refsCompatible := policy.RefsVersion == 0 || policy.RefsVersion == 1 && refsRequired
		if !refsCompatible || refsRequired && policy.RefsVersion != 1 || policy.SchemaVersion > SpecAuthorityTransferPolicySchemaVersion || !transferValid {
			out.State = "unsupported-artifact-contract"
			return out
		}
	} else if !os.IsNotExist(err) {
		out.State = "unavailable-project"
		return out
	}
	var body string
	var deps []string
	switch ref.Kind {
	case "spec":
		redirect, found, err := readSpecTransferRedirect(s, ref)
		if err != nil {
			out.State = "invalid-spec-redirect"
			return out
		}
		if found {
			key := ref.String()
			if redirectDepth >= 16 || redirectPath[key] {
				out.State = "redirect-cycle"
				return out
			}
			redirectPath[key] = true
			target := r.resolve(redirect.Destination.Project, redirect.Destination.String(), redirectPath, redirectDepth+1)
			delete(redirectPath, key)
			if !target.Resolved {
				out.State = "redirect-target-" + target.State
				return out
			}
			canonical := target.Identity
			if target.CanonicalIdentity != nil {
				canonical = *target.CanonicalIdentity
			}
			out.Resolved, out.State, out.Redirected = true, "redirected", true
			out.CanonicalIdentity = &canonical
			out.Status, out.Revision, out.Digest = target.Status, target.Revision, target.Digest
			out.Dependencies = append([]ArtifactRef{}, target.Dependencies...)
			return out
		}
		sp, err := s.GetSpec(ref.Slug)
		if err != nil {
			out.State = "unknown-spec"
			var duplicate SpecIdentityConflictError
			if errors.As(err, &duplicate) {
				out.State = "conflicting-artifact-identity"
			}
			return out
		}
		out.Status, body, deps = sp.Status, sp.Body, sp.DependsOn
		if info, err := os.Stat(sp.Path); err == nil && !info.IsDir() {
			raw, err := os.ReadFile(sp.Path)
			if err != nil {
				out.State = "unavailable-artifact"
				return out
			}
			body = string(raw)
		}
	case "roadmap", "milestone":
		path := filepath.Join(s.roadmapsDir(), ref.Slug+".md")
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			out.State = "unknown-roadmap"
			return out
		}
		if !artifactPathWithin(s.Root, path) {
			out.State = "unavailable-artifact"
			return out
		}
		rm, err := s.GetRoadmap(ref.Slug)
		if err != nil {
			out.State = "unknown-roadmap"
			return out
		}
		out.Status, body = rm.Status, rm.Body
		if raw, err := os.ReadFile(path); err == nil {
			body = string(raw)
		} else {
			out.State = "unavailable-artifact"
			return out
		}
		for _, dep := range rm.DependsOn {
			if !strings.Contains(dep, ":") {
				dep = "roadmap:" + dep
			}
			deps = append(deps, dep)
		}
		for _, consumed := range rm.Consumes {
			if !strings.Contains(consumed, ":") {
				consumed = "roadmap:" + consumed
			}
			deps = append(deps, consumed)
		}
		if ref.Kind == "roadmap" {
			for _, ms := range rm.Milestones {
				deps = append(deps, "milestone:"+rm.Slug+"/"+ms.ID)
			}
		}
		if ref.Kind == "milestone" {
			found := false
			for _, ms := range rm.Milestones {
				if ms.ID != ref.Milestone {
					continue
				}
				if found {
					out.State = "conflicting-artifact-identity"
					return out
				}
				found = true
				deps = append([]string{}, ms.Specs...)
				for _, consumed := range ms.Consumes {
					if !strings.Contains(consumed, ":") {
						consumed = "roadmap:" + consumed
					}
					deps = append(deps, consumed)
				}
				for _, after := range ms.After {
					if !strings.Contains(after, ":") {
						after = "milestone:" + rm.Slug + "/" + after
					}
					deps = append(deps, after)
				}
				out.Status = "done"
				for _, rawSpec := range ms.Specs {
					if !strings.Contains(rawSpec, ":") {
						rawSpec = "spec:" + rawSpec
					}
					sp := r.resolve(ref.Project, rawSpec, redirectPath, redirectDepth)
					if !sp.Resolved || sp.Status != "done" {
						out.Status = "pending"
					}
				}
			}
			if !found {
				out.State = "unknown-milestone"
				return out
			}
		}
	}
	if len(deps) > 256 {
		out.State = "resolution-limit"
		return out
	}
	for _, dep := range deps {
		parsed, err := ParseArtifactRef(dep)
		if err != nil {
			out.State = "invalid-artifact-reference"
			return out
		}
		if parsed.Project == "" {
			parsed.Project = ref.Project
		}
		out.Dependencies = append(out.Dependencies, parsed)
	}
	out.Resolved, out.State = true, "resolved"
	digest := sha256.Sum256([]byte(body))
	out.Digest = hex.EncodeToString(digest[:])
	// rev-parse can walk to an ancestor repository: require this root to be its
	// own Git top level before reporting a revision for a registered project.
	if top, err := exec.Command("git", "-C", s.Root, "rev-parse", "--show-toplevel").Output(); err == nil && sameProjectRoot(s.Root, strings.TrimSpace(string(top))) {
		if revision, err := exec.Command("git", "-C", s.Root, "rev-parse", "--verify", "HEAD").Output(); err == nil {
			out.Revision = strings.TrimSpace(string(revision))
		}
	}
	return out
}

// ValidateGraph checks existence/authorization/cycles, not lifecycle acceptance.
// It is bounded even for adversarial cross-project graphs and produces path-free
// diagnostics. A done node does not conceal a broken or cyclic dependency.
func (r ArtifactResolver) ValidateGraph(project, raw string) string {
	visiting, visited := map[string]bool{}, map[string]bool{}
	nodes := 0
	var visit func(string, string, int) string
	visit = func(project, raw string, depth int) string {
		if depth > 64 || nodes >= 256 {
			return "resolution-limit"
		}
		ref, err := ParseArtifactRef(raw)
		if err != nil {
			return "invalid-artifact-reference"
		}
		if ref.Project == "" {
			ref.Project = project
		}
		key := ref.String()
		if visiting[key] {
			return "dependency-cycle"
		}
		if visited[key] {
			return ""
		}
		nodes++
		resolved := r.Resolve(project, raw)
		if !resolved.Resolved {
			return resolved.State
		}
		visiting[key] = true
		for _, dep := range resolved.Dependencies {
			if reason := visit(dep.Project, dep.String(), depth+1); reason != "" {
				return reason
			}
		}
		delete(visiting, key)
		visited[key] = true
		return ""
	}
	return visit(project, raw, 0)
}

func sameProjectRoot(a, b string) bool {
	clean := func(p string) string {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		abs, _ := filepath.Abs(p)
		return filepath.Clean(abs)
	}
	return clean(a) == clean(b)
}

func artifactPathWithin(root, path string) bool {
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(base, real)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// EnvironmentArtifactResolver reuses the CLI's existing project configuration.
// Merely placing a checkout inside another repository never registers it.
func EnvironmentArtifactResolver(root, projectsDir string) (ArtifactResolver, string, error) {
	explicit, err := ParseRootsJSON(os.Getenv("POSE_PROJECT_ROOTS"))
	if err != nil {
		return ArtifactResolver{}, "", err
	}
	id := os.Getenv("POSE_DEFAULT_PROJECT_ID")
	if id == "" {
		for candidate, path := range explicit {
			if !sameProjectRoot(root, path) {
				continue
			}
			if id != "" && id != candidate {
				return ArtifactResolver{}, "", fmt.Errorf("conflicting-project-binding")
			}
			id = candidate
		}
	}
	if id == "" {
		id = "proj." + filepath.Base(root)
	}
	if ValidateSlug(id) != nil {
		return ArtifactResolver{}, "", fmt.Errorf("invalid-project-id")
	}
	if selected, ok := explicit[id]; ok && !sameProjectRoot(selected, root) {
		return ArtifactResolver{}, "", fmt.Errorf("conflicting-project-binding")
	}
	if projectsDir == "" {
		projectsDir = os.Getenv("HARNE8_PROJECTS_DIR")
	}
	roots := NewRoots(RootsConfig{DefaultRoot: root, DefaultProjectID: id, ProjectsDir: projectsDir, Explicit: explicit})
	return ArtifactResolver{Roots: roots}, id, nil
}
