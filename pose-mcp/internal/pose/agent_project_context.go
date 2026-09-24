package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AgentProjectContext is a path-free snapshot shared by the native CLI and MCP
// entrypoints. ContextRevision binds the selected project, qualified task,
// observed project revisions and artifact digest; it is a freshness token, not
// authorization by itself.
type AgentProjectContext struct {
	SchemaVersion       int                 `json:"schema_version"`
	SelectedProjectID   string              `json:"selected_project_id"`
	SelectedRevision    string              `json:"selected_revision,omitempty"`
	TaskRef             string              `json:"task_ref,omitempty"`
	Authority           *ArtifactRef        `json:"authority,omitempty"`
	CoordinatorRelation string              `json:"coordinator_relation,omitempty"`
	TaskResolution      *ArtifactResolution `json:"task_resolution,omitempty"`
	AuthorityRevision   string              `json:"authority_revision,omitempty"`
	SupportedContracts  []string            `json:"supported_contracts"`
	ContextRevision     string              `json:"context_revision"`
}

// SupportedProjectContextContracts lists the multi-project contracts this
// engine can read. A project still has to adopt a contract before the resolver
// will accept its metadata.
func SupportedProjectContextContracts() []string {
	return []string{
		"qualified-artifact-refs@1",
		"spec-authority-transfer@1",
		"federated-roadmap-acceptance@1",
	}
}

// ResolveAgentProjectContext resolves a selected project and optional task
// through the same bounded, authorization-aware resolver used by governed
// artifact reads. It never returns a filesystem path or artifact body.
func ResolveAgentProjectContext(resolver ArtifactResolver, selectedProjectID, taskRef string) (AgentProjectContext, error) {
	if selectedProjectID == "" || ValidateSlug(selectedProjectID) != nil {
		return AgentProjectContext{}, fmt.Errorf("invalid-project-selection")
	}
	if resolver.Authorize != nil && !resolver.Authorize(selectedProjectID) {
		return AgentProjectContext{}, fmt.Errorf("unauthorized-project")
	}
	if resolver.Roots == nil {
		return AgentProjectContext{}, fmt.Errorf("project-registry-unavailable")
	}
	selectedStore, err := resolver.Roots.StoreFor(selectedProjectID)
	if err != nil {
		return AgentProjectContext{}, fmt.Errorf("project-selection-unavailable")
	}

	result := AgentProjectContext{
		SchemaVersion:      1,
		SelectedProjectID:  selectedProjectID,
		SelectedRevision:   gitRevisionOrEmpty(selectedStore.Root),
		SupportedContracts: SupportedProjectContextContracts(),
	}
	if taskRef != "" {
		if !strings.Contains(taskRef, ":") {
			return AgentProjectContext{}, fmt.Errorf("unqualified-task-reference")
		}
		ref, err := ParseArtifactRef(taskRef)
		if err != nil {
			return AgentProjectContext{}, fmt.Errorf("invalid-artifact-reference")
		}
		if ref.Project == "" {
			ref.Project = selectedProjectID
		}
		result.TaskRef = ref.String()
		resolved := resolver.Resolve(selectedProjectID, result.TaskRef)
		if resolved.State == "unauthorized-project" {
			return AgentProjectContext{}, fmt.Errorf("unauthorized-project")
		}
		result.TaskResolution = &resolved
		authority := resolved.Identity
		if authority.Project == "" {
			authority.Project = selectedProjectID
		}
		if resolved.CanonicalIdentity != nil {
			authority = *resolved.CanonicalIdentity
		}
		result.Authority = &authority
		result.AuthorityRevision = resolved.Revision
		if result.AuthorityRevision == "" && authority.Project != selectedProjectID {
			if authorityStore, storeErr := resolver.Roots.StoreFor(authority.Project); storeErr == nil {
				result.AuthorityRevision = gitRevisionOrEmpty(authorityStore.Root)
			}
		}
		if authority.Project == selectedProjectID {
			result.CoordinatorRelation = "same-project-authority"
		} else {
			result.CoordinatorRelation = "selected-project-to-qualified-authority"
		}
	}

	identity := struct {
		SelectedProjectID       string       `json:"selected_project_id"`
		SelectedRevision        string       `json:"selected_revision"`
		SelectedBindingDigest   string       `json:"selected_binding_digest"`
		SelectedContractDigest  string       `json:"selected_contract_digest"`
		TaskRef                 string       `json:"task_ref"`
		Authority               *ArtifactRef `json:"authority,omitempty"`
		AuthorityRevision       string       `json:"authority_revision"`
		AuthorityBindingDigest  string       `json:"authority_binding_digest"`
		AuthorityContractDigest string       `json:"authority_contract_digest"`
		ResolutionState         string       `json:"resolution_state"`
		ArtifactRevision        string       `json:"artifact_revision"`
		ArtifactDigest          string       `json:"artifact_digest"`
	}{
		SelectedProjectID:      result.SelectedProjectID,
		SelectedRevision:       result.SelectedRevision,
		SelectedBindingDigest:  projectBindingDigest(selectedProjectID, selectedStore.Root),
		SelectedContractDigest: projectContractDigest(selectedStore.Root),
		TaskRef:                result.TaskRef,
		Authority:              result.Authority,
		AuthorityRevision:      result.AuthorityRevision,
	}
	if result.TaskResolution != nil {
		identity.ResolutionState = result.TaskResolution.State
		identity.ArtifactRevision = result.TaskResolution.Revision
		identity.ArtifactDigest = result.TaskResolution.Digest
	}
	if result.Authority != nil {
		if authorityStore, storeErr := resolver.Roots.StoreFor(result.Authority.Project); storeErr == nil {
			identity.AuthorityBindingDigest = projectBindingDigest(result.Authority.Project, authorityStore.Root)
			identity.AuthorityContractDigest = projectContractDigest(authorityStore.Root)
		}
	}
	canonical, err := json.Marshal(identity)
	if err != nil {
		return AgentProjectContext{}, fmt.Errorf("context-identity-unavailable")
	}
	digest := sha256.Sum256(canonical)
	result.ContextRevision = hex.EncodeToString(digest[:])
	return result, nil
}

func gitRevisionOrEmpty(root string) string {
	revision, err := transferGitRevision(root)
	if err != nil {
		return ""
	}
	return revision
}

func projectBindingDigest(projectID, root string) string {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if absolute, err := filepath.Abs(root); err == nil {
		root = filepath.Clean(absolute)
	}
	digest := sha256.Sum256([]byte("pose-project-binding-v1\x00" + projectID + "\x00" + root))
	return hex.EncodeToString(digest[:])
}

func projectContractDigest(root string) string {
	reviewDigest := projectReviewPolicyDigest(root)
	_, federationDigest, federationErr := LoadFederatedRoadmapPolicy(root)
	state := struct {
		Review     string `json:"review"`
		Federation string `json:"federation"`
	}{
		Review:     reviewDigest,
		Federation: federationDigest,
	}
	if federationErr != nil {
		state.Federation = "federation-contract-unavailable"
	}
	canonical, err := json.Marshal(state)
	if err != nil {
		return "project-contract-unavailable"
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}

func projectReviewPolicyDigest(root string) string {
	const policyPath = ".pose/policy/review.json"
	if err := ValidateArtifactPath(root, policyPath, false); err != nil {
		return "review-contract-unavailable"
	}
	path := filepath.Join(root, filepath.FromSlash(policyPath))
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		digest := sha256.Sum256(nil)
		return hex.EncodeToString(digest[:])
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > reviewBaselineMaxBytes {
		return "review-contract-unavailable"
	}
	file, err := os.Open(path)
	if err != nil {
		return "review-contract-unavailable"
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, reviewBaselineMaxBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(raw) > reviewBaselineMaxBytes {
		return "review-contract-unavailable"
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
