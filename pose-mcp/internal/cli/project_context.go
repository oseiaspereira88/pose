package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func cliProjectResolver(root string) (posemodel.ArtifactResolver, string, error) {
	return posemodel.EnvironmentArtifactResolver(root, "")
}

func cliAgentContext(root, taskRef string) (posemodel.AgentProjectContext, posemodel.ArtifactResolver, string, error) {
	resolver, currentProject, err := cliProjectResolver(root)
	if err != nil {
		return posemodel.AgentProjectContext{}, posemodel.ArtifactResolver{}, "", err
	}
	context, err := posemodel.ResolveAgentProjectContext(resolver, currentProject, taskRef)
	return context, resolver, currentProject, err
}

func cliAuthorizedWriteStore(root, projectID string, resolver posemodel.ArtifactResolver, currentProject string) (posemodel.Store, bool, error) {
	store, err := resolver.Roots.StoreFor(projectID)
	if err != nil {
		return posemodel.Store{}, false, fmt.Errorf("project-selection-unavailable")
	}
	if sameCLIProjectRoot(root, store.Root) {
		if projectID != currentProject {
			return posemodel.Store{}, false, fmt.Errorf("conflicting-project-binding")
		}
		return store, false, nil
	}
	explicit, err := posemodel.ParseRootsJSON(os.Getenv("POSE_PROJECT_ROOTS"))
	if err != nil || explicit[projectID] == "" || !sameCLIProjectRoot(explicit[projectID], store.Root) {
		return posemodel.Store{}, true, fmt.Errorf("cross-project-write-unauthorized")
	}
	return store, true, nil
}

type cliArtifactRoute struct {
	Context        posemodel.AgentProjectContext
	Resolver       posemodel.ArtifactResolver
	CurrentProject string
	Store          posemodel.Store
	LocalRef       string
	CrossProject   bool
}

func cliResolveArtifactRoute(root, taskRef string) (cliArtifactRoute, error) {
	context, resolver, currentProject, err := cliAgentContext(root, taskRef)
	if err != nil {
		return cliArtifactRoute{}, err
	}
	if context.TaskResolution == nil || !context.TaskResolution.Resolved || context.Authority == nil {
		state := "unknown-artifact"
		if context.TaskResolution != nil {
			state = context.TaskResolution.State
		}
		return cliArtifactRoute{}, fmt.Errorf("task-authority-unavailable: %s", state)
	}
	store, err := resolver.Roots.StoreFor(context.Authority.Project)
	if err != nil {
		return cliArtifactRoute{}, fmt.Errorf("task-authority-unavailable")
	}
	return cliArtifactRoute{
		Context:        context,
		Resolver:       resolver,
		CurrentProject: currentProject,
		Store:          store,
		LocalRef:       scopeRefForArtifact(*context.Authority),
		CrossProject:   !sameCLIProjectRoot(root, store.Root) || context.Authority.Project != currentProject,
	}, nil
}

func cliMaybeRouteScope(root, taskRef string) (string, string, bool, error) {
	qualified := strings.HasPrefix(taskRef, "xref:")
	context, resolver, _, err := cliAgentContext(root, taskRef)
	if err != nil {
		if qualified {
			return "", "", false, err
		}
		return root, taskRef, false, nil
	}
	if context.TaskResolution == nil {
		return root, taskRef, false, nil
	}
	if context.TaskResolution.State == "transfer-in-progress" {
		return "", "", false, fmt.Errorf("transfer-in-progress")
	}
	if !context.TaskResolution.Resolved || context.Authority == nil {
		if qualified {
			return "", "", false, fmt.Errorf("task-authority-unavailable: %s", context.TaskResolution.State)
		}
		return root, taskRef, false, nil
	}
	if !qualified && !context.TaskResolution.Redirected {
		return root, taskRef, false, nil
	}
	store, err := resolver.Roots.StoreFor(context.Authority.Project)
	if err != nil {
		return "", "", false, fmt.Errorf("task-authority-unavailable")
	}
	return store.Root, scopeRefForArtifact(*context.Authority), true, nil
}

func sameCLIProjectRoot(left, right string) bool {
	canonical := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return filepath.Clean(path)
		}
		return filepath.Clean(absolute)
	}
	return canonical(left) == canonical(right)
}

func cmdProjectContext(root string, args []string, stdout, stderr io.Writer) int {
	selectedProject, taskRef, jsonOutput := "", "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project-id", "--task":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				render(io.Discard, stderr).Usage("Usage: pose context [--project-id <id>] [--task <artifact-ref>] [--json]")
				return 2
			}
			i++
			if args[i-1] == "--project-id" {
				if selectedProject != "" {
					render(io.Discard, stderr).Failure("pose context: duplicate --project-id")
					return 2
				}
				selectedProject = args[i]
			} else {
				if taskRef != "" {
					render(io.Discard, stderr).Failure("pose context: duplicate --task")
					return 2
				}
				taskRef = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			render(io.Discard, stderr).Usage("Usage: pose context [--project-id <id>] [--task <artifact-ref>] [--json]")
			return 2
		}
	}

	resolver, currentProject, err := posemodel.EnvironmentArtifactResolver(root, "")
	if err != nil {
		render(io.Discard, stderr).Failure("pose context: " + err.Error())
		return 1
	}
	if selectedProject == "" {
		selectedProject = currentProject
	}
	context, err := posemodel.ResolveAgentProjectContext(resolver, selectedProject, taskRef)
	if err != nil {
		render(io.Discard, stderr).Failure("pose context: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, context)
	}
	rows := [][]string{{"selected_project_id", context.SelectedProjectID}, {"selected_revision", context.SelectedRevision}}
	if context.TaskRef != "" {
		rows = append(rows, []string{"task_ref", context.TaskRef})
		if context.Authority != nil {
			rows = append(rows, []string{"authority", context.Authority.String()})
		}
		rows = append(rows, []string{"coordinator_relation", context.CoordinatorRelation})
		if context.TaskResolution != nil {
			rows = append(rows,
				[]string{"resolution_state", context.TaskResolution.State},
				[]string{"task_revision", context.TaskResolution.Revision},
				[]string{"task_digest", context.TaskResolution.Digest},
			)
		}
	}
	rows = append(rows,
		[]string{"context_revision", context.ContextRevision},
		[]string{"supported_contracts", strings.Join(context.SupportedContracts, ",")},
	)
	render(stdout, stderr).Table(cliout.Table{Header: []string{"Context", "Value"}, Rows: rows})
	return 0
}
