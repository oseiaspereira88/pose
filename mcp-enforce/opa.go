package mcpenforce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// opaQueryBody is the payload sent to POST /v1/data/<path>.
type opaQueryBody struct {
	Input opaInputDoc `json:"input"`
}

// opaInputDoc is the OPA input document. The four base fields are always present
// (preserving the established wire contract); the Execution Identity scope fields
// are omitted when unset so existing Rego policies see an unchanged document.
type opaInputDoc struct {
	Principal  string   `json:"principal"`
	ProjectID  string   `json:"project_id"`
	ProjectIDs []string `json:"project_ids,omitempty"`
	Method     string   `json:"method"`
	Tool       string   `json:"tool"`

	Scopes    []string `json:"scopes,omitempty"`
	RunID     string   `json:"run_id,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"` // RFC3339 UTC, only when set
}

// opaDataResponse wraps the /v1/data envelope. Result is null when the policy
// path is undefined.
type opaDataResponse struct {
	Result *opaDecisionDoc `json:"result"`
}

// opaDecisionDoc is the expected shape of the Rego decision document.
//
//	default allow = false
//	violations contains msg if { ... }
type opaDecisionDoc struct {
	Allow      bool     `json:"allow"`
	Violations []string `json:"violations"`
}

// newOPAInputDoc projects a PolicyInput onto the OPA wire document.
func newOPAInputDoc(input PolicyInput) opaInputDoc {
	doc := opaInputDoc{
		Principal:  input.Principal,
		ProjectID:  input.ProjectID,
		ProjectIDs: input.ProjectIDs,
		Method:     input.Method,
		Tool:       input.ToolName,
		Scopes:     input.Scopes,
		RunID:      input.RunID,
	}
	if !input.ExpiresAt.IsZero() {
		doc.ExpiresAt = input.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return doc
}

func (g *PolicyGate) queryOPA(ctx context.Context, input PolicyInput) (allow bool, violations []string, err error) {
	payload, err := json.Marshal(opaQueryBody{Input: newOPAInputDoc(input)})
	if err != nil {
		return false, nil, fmt.Errorf("policy: marshal OPA input: %w", err)
	}

	url := g.cfg.OPAURL + "/v1/data/" + g.cfg.OPAPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false, nil, fmt.Errorf("policy: build OPA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.hc.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("policy: OPA query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("policy: OPA returned HTTP %d", resp.StatusCode)
	}

	var out opaDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, nil, fmt.Errorf("policy: decode OPA response: %w", err)
	}

	// OPA returns null result when the policy path is undefined → treat as deny.
	if out.Result == nil {
		return false, []string{"policy_path_undefined"}, nil
	}
	return out.Result.Allow, out.Result.Violations, nil
}
