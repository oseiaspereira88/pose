package pose

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Release version sources (spec pose-release-version-source). The release
// lifecycle needs one authoritative version to compare a cut against. The
// engine's own version is that evidence only inside the engine repository;
// any other project declares the file that holds its own.

const (
	ReleaseVersionSourceText = "text"
	ReleaseVersionSourceJSON = "json"

	// EngineVersionFile is where the engine repository keeps its version. Its
	// presence in a project root is what makes the compiled engine version the
	// authoritative evidence for that project.
	EngineVersionFile = "pose-mcp/internal/version/version.go"

	releaseVersionTextLimit = 4 << 10
	releaseVersionJSONLimit = 1 << 20
	releaseVersionDevSuffix = "-dev"
)

var releaseSourceVersionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ReleaseVersionSource declares where a project's authoritative version lives.
type ReleaseVersionSource struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Key  string `json:"key,omitempty"`
}

// Validate checks the declaration without touching the file system.
func (s ReleaseVersionSource) Validate() error {
	if strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(s.Path) || strings.Contains(s.Path, "\\") {
		return fmt.Errorf("path must be project-relative")
	}
	switch s.Kind {
	case ReleaseVersionSourceText:
		if s.Key != "" {
			return fmt.Errorf("key is only valid for kind %q", ReleaseVersionSourceJSON)
		}
	case ReleaseVersionSourceJSON:
		if s.Key == "" {
			return fmt.Errorf("key is required for kind %q", ReleaseVersionSourceJSON)
		}
	default:
		return fmt.Errorf("kind must be %q or %q", ReleaseVersionSourceText, ReleaseVersionSourceJSON)
	}
	return nil
}

// ConfinedProjectPath resolves a project-relative path and refuses absolute
// paths and any path or symlink that leaves the project root.
func ConfinedProjectPath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute path")
	}
	full := filepath.Join(root, filepath.Clean(path))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("escape")
	}
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = root
	}
	resolvedRel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("symlink escape")
	}
	return full, nil
}

// ReadReleaseVersionSource returns the declared project version as vX.Y.Z. It
// performs one bounded read of a regular file inside the project, and shows
// nothing from the file other than the version it validated.
func ReadReleaseVersionSource(root string, source ReleaseVersionSource) (string, error) {
	if err := source.Validate(); err != nil {
		return "", fmt.Errorf("release policy version_source: %w", err)
	}
	full, err := ConfinedProjectPath(root, source.Path)
	if err != nil {
		// Resolution errors from the file system quote absolute paths; report
		// only the cause.
		cause := "cannot resolve file"
		switch {
		case err.Error() == "escape" || err.Error() == "symlink escape":
			cause = err.Error()
		case os.IsNotExist(err):
			cause = "file not found"
		}
		return "", fmt.Errorf("release policy version_source %q: %s", source.Path, cause)
	}
	limit := int64(releaseVersionTextLimit)
	if source.Kind == ReleaseVersionSourceJSON {
		limit = releaseVersionJSONLimit
	}
	file, err := os.Open(full)
	if err != nil {
		return "", fmt.Errorf("release policy version_source %q: cannot read file", source.Path)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("release policy version_source %q: not a regular file", source.Path)
	}
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return "", fmt.Errorf("release policy version_source %q: cannot read file", source.Path)
	}
	if int64(len(raw)) > limit {
		return "", fmt.Errorf("release policy version_source %q: file exceeds %d bytes", source.Path, limit)
	}
	value := ""
	switch source.Kind {
	case ReleaseVersionSourceText:
		value = strings.TrimSpace(string(raw))
		if strings.ContainsAny(value, "\r\n") {
			return "", fmt.Errorf("release policy version_source %q: expected a single line", source.Path)
		}
	case ReleaseVersionSourceJSON:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return "", fmt.Errorf("release policy version_source %q: not a JSON object", source.Path)
		}
		field, ok := fields[source.Key]
		if !ok {
			return "", fmt.Errorf("release policy version_source %q: key %q is absent", source.Path, source.Key)
		}
		if err := json.Unmarshal(field, &value); err != nil {
			return "", fmt.Errorf("release policy version_source %q: key %q is not a string", source.Path, source.Key)
		}
	}
	value = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(value), "v"), releaseVersionDevSuffix)
	if !releaseSourceVersionRE.MatchString(value) {
		return "", fmt.Errorf("release policy version_source %q: value is not a semantic version X.Y.Z", source.Path)
	}
	return "v" + value, nil
}

// ProjectHasEngineVersionFile reports whether the project is the engine
// repository, the only place where the compiled engine version is authoritative.
func ProjectHasEngineVersionFile(root string) bool {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(EngineVersionFile)))
	return err == nil && info.Mode().IsRegular()
}
