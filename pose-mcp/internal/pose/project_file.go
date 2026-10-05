package pose

// .pose/project.json declares a repository's project id once, so the CLI, the
// MCP server and any checkout under any directory name resolve the same
// project (spec pose-project-identity-file). It sits after the explicit
// environment bindings and before the directory name, and an environment
// binding that disagrees with it is refused rather than silently preferred.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ProjectFileSchemaVersion = 1

type ProjectFile struct {
	SchemaVersion int    `json:"schema_version"`
	ProjectID     string `json:"project_id"`
	Name          string `json:"name,omitempty"`
}

func projectFilePath(root string) string {
	return filepath.Join(root, ".pose", "project.json")
}

// ReadProjectFile returns the declared id. ok is false when the file is
// absent; err is set when it exists and does not declare a valid id, which a
// caller resolving an identity must treat as a failure, never as absence.
func ReadProjectFile(root string) (string, bool, error) {
	raw, err := os.ReadFile(projectFilePath(root))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var doc ProjectFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", true, fmt.Errorf("invalid-project-id: .pose/project.json is not valid JSON: %v", err)
	}
	if doc.SchemaVersion != ProjectFileSchemaVersion {
		return "", true, fmt.Errorf("invalid-project-id: .pose/project.json schema_version %d is not supported (engine reads %d)", doc.SchemaVersion, ProjectFileSchemaVersion)
	}
	if ValidateSlug(doc.ProjectID) != nil {
		return "", true, fmt.Errorf("invalid-project-id: .pose/project.json declares %q, which is not a valid project id", doc.ProjectID)
	}
	return doc.ProjectID, true, nil
}

// WriteProjectFile declares id unless a file already exists, and reports
// whether it wrote. A declared identity is never rewritten: other
// repositories may already reference the project by it.
func WriteProjectFile(root, id, name string) (bool, error) {
	if ValidateSlug(id) != nil {
		return false, fmt.Errorf("pose: %q is not a valid project id", id)
	}
	if _, err := os.Stat(projectFilePath(root)); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	raw, err := json.MarshalIndent(ProjectFile{SchemaVersion: ProjectFileSchemaVersion, ProjectID: id, Name: name}, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(projectFilePath(root)), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(projectFilePath(root), append(raw, '\n'), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
