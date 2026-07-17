package mcpenforce

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// HeaderValue returns the first non-empty, trimmed value among the named headers.
func HeaderValue(h http.Header, names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(h.Get(name)); v != "" {
			return v
		}
	}
	return ""
}

// PrincipalFromHeader extracts the authenticated principal, preferring
// X-MCP-Principal and falling back to X-Principal.
func PrincipalFromHeader(h http.Header) string {
	return HeaderValue(h, "X-MCP-Principal", "X-Principal")
}

// ProjectScopeFromArguments reads and normalizes project_id/project_ids from a
// tools/call arguments object. Malformed input yields an empty scope.
func ProjectScopeFromArguments(args json.RawMessage) (string, []string, bool) {
	if len(bytes.TrimSpace(args)) == 0 || bytes.Equal(bytes.TrimSpace(args), []byte("null")) {
		return "", nil, false
	}
	var sel struct {
		ProjectID  string   `json:"project_id"`
		ProjectIDs []string `json:"project_ids"`
	}
	if json.Unmarshal(args, &sel) != nil {
		return "", nil, true
	}
	projectID := strings.TrimSpace(sel.ProjectID)
	seen := make(map[string]struct{}, len(sel.ProjectIDs))
	projectIDs := make([]string, 0, len(sel.ProjectIDs))
	for _, raw := range sel.ProjectIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		projectIDs = append(projectIDs, id)
	}
	return projectID, projectIDs, false
}

// ProjectIDFromArguments preserves the legacy scalar extraction API.
func ProjectIDFromArguments(args json.RawMessage) string {
	projectID, _, _ := ProjectScopeFromArguments(args)
	return projectID
}

// ConfigFromEnv reads OPA integration settings from environment variables under
// the given prefix. For prefix "POSE_MCP_" it reads:
//
//   - <prefix>OPA_URL            base URL of the OPA server (empty = dev allow-all)
//   - <prefix>OPA_PATH           OPA policy path (falls back to defaultOPAPath)
//   - <prefix>OPA_TIMEOUT        evaluation timeout in seconds
//   - <prefix>REQUIRE_PRINCIPAL  "1"/"true"/"yes"/"on" → deny anonymous callers
//   - <prefix>REQUIRE_IDENTITY   "1"/"true"/"yes"/"on" → deny calls without a
//     run-bound Execution Identity (ADR-007)
func ConfigFromEnv(prefix, defaultOPAPath string) PolicyConfig {
	cfg := PolicyConfig{
		OPAURL:           os.Getenv(prefix + "OPA_URL"),
		OPAPath:          os.Getenv(prefix + "OPA_PATH"),
		RequirePrincipal: isTruthy(os.Getenv(prefix + "REQUIRE_PRINCIPAL")),
		RequireIdentity:  isTruthy(os.Getenv(prefix + "REQUIRE_IDENTITY")),
	}
	if cfg.OPAPath == "" {
		cfg.OPAPath = defaultOPAPath
	}
	if t := os.Getenv(prefix + "OPA_TIMEOUT"); t != "" {
		if secs, err := strconv.ParseFloat(t, 64); err == nil && secs > 0 {
			cfg.Timeout = time.Duration(secs * float64(time.Second))
		}
	}
	return cfg
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
