package mcpenforce

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// IdentityHeader is the HTTP header carrying the signed Execution Identity token
// (ADR-007) on MCP tool calls.
const IdentityHeader = "X-MCP-Execution-Identity"

// Identity is the conductor-issued Execution Identity (ADR-007): run-bound,
// least-privilege and time-boxed. It is the source of the scope fields applied
// to a PolicyInput.
type Identity struct {
	RunID         string    `json:"run_id"`
	ProjectID     string    `json:"project_id"`
	Scopes        []string  `json:"scopes"`
	PolicyVersion string    `json:"policy_version,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// MintToken produces a compact "<payload>.<sig>" token, where payload is the
// base64url JSON identity and sig is base64url(HMAC-SHA256(secret, payload)).
// The conductor (issuer) and the gate's consumer (verifier) share secret — the
// "backend leve" of ADR-007; an asymmetric/SPIFFE scheme is the drop-in upgrade.
func MintToken(id Identity, secret []byte) (string, error) {
	raw, err := json.Marshal(id)
	if err != nil {
		return "", fmt.Errorf("identity: marshal: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + signPayload(payload, secret), nil
}

// ParseToken verifies a token's HMAC signature and returns the Identity. It does
// NOT check expiry — the PolicyGate enforces the time-box so a single clock
// governs expiry decisions.
func ParseToken(token string, secret []byte) (Identity, error) {
	var id Identity
	payload, sig, ok := strings.Cut(token, ".")
	if !ok {
		return id, fmt.Errorf("identity: malformed token")
	}
	want := signPayload(payload, secret)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(want)) != 1 {
		return id, fmt.Errorf("identity: signature mismatch")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return id, fmt.Errorf("identity: decode payload: %w", err)
	}
	if err := json.Unmarshal(raw, &id); err != nil {
		return id, fmt.Errorf("identity: unmarshal: %w", err)
	}
	return id, nil
}

func signPayload(payload string, secret []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// IdentityFromHeader extracts and verifies the Execution Identity token from h.
// It returns (nil, nil) when no token is present (anonymous/dev), and an error
// when a token is present but fails verification.
func IdentityFromHeader(h http.Header, secret []byte) (*Identity, error) {
	tok := strings.TrimSpace(h.Get(IdentityHeader))
	if tok == "" {
		return nil, nil
	}
	id, err := ParseToken(tok, secret)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// Apply copies the identity's scope fields onto a PolicyInput. ProjectID is
// filled only when the input does not already carry one (argument/header wins).
func (id Identity) Apply(in PolicyInput) PolicyInput {
	in.RunID = id.RunID
	in.Scopes = id.Scopes
	in.ExpiresAt = id.ExpiresAt
	if in.ProjectID == "" {
		in.ProjectID = id.ProjectID
	}
	return in
}
