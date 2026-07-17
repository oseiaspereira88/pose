package mcpenforce

import (
	"net/http"
	"testing"
	"time"
)

var testSecret = []byte("shared-hmac-secret")

func sampleIdentity() Identity {
	return Identity{
		RunID:         "run-123",
		ProjectID:     "proj.a",
		Scopes:        []string{"repo:read", "graph:read"},
		PolicyVersion: "rego.bundle.v1",
		ExpiresAt:     time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
	}
}

func TestMintParse_RoundTrip(t *testing.T) {
	tok, err := MintToken(sampleIdentity(), testSecret)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	got, err := ParseToken(tok, testSecret)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := sampleIdentity()
	if got.RunID != want.RunID || got.ProjectID != want.ProjectID || got.PolicyVersion != want.PolicyVersion {
		t.Errorf("scalar fields mismatch: %+v", got)
	}
	if len(got.Scopes) != 2 || got.Scopes[0] != "repo:read" || got.Scopes[1] != "graph:read" {
		t.Errorf("scopes = %v", got.Scopes)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("expires_at = %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
}

func TestParseToken_TamperedPayload_Rejected(t *testing.T) {
	tok, _ := MintToken(sampleIdentity(), testSecret)
	// Flip the first payload byte; signature must no longer match.
	tampered := "X" + tok[1:]
	if _, err := ParseToken(tampered, testSecret); err == nil {
		t.Error("tampered token accepted, want signature mismatch")
	}
}

func TestParseToken_WrongSecret_Rejected(t *testing.T) {
	tok, _ := MintToken(sampleIdentity(), testSecret)
	if _, err := ParseToken(tok, []byte("other-secret")); err == nil {
		t.Error("token verified under wrong secret, want error")
	}
}

func TestParseToken_Malformed_Rejected(t *testing.T) {
	if _, err := ParseToken("no-dot-here", testSecret); err == nil {
		t.Error("malformed token (no separator) accepted")
	}
}

func TestIdentityFromHeader(t *testing.T) {
	h := http.Header{}
	// Absent → (nil, nil): anonymous/dev, not an error.
	id, err := IdentityFromHeader(h, testSecret)
	if err != nil || id != nil {
		t.Fatalf("absent header = (%v, %v), want (nil, nil)", id, err)
	}
	tok, _ := MintToken(sampleIdentity(), testSecret)
	h.Set(IdentityHeader, tok)
	id, err = IdentityFromHeader(h, testSecret)
	if err != nil || id == nil || id.RunID != "run-123" {
		t.Fatalf("present header = (%+v, %v), want valid identity", id, err)
	}
	// Present but invalid → error.
	h.Set(IdentityHeader, "garbage.sig")
	if _, err := IdentityFromHeader(h, testSecret); err == nil {
		t.Error("invalid token in header accepted")
	}
}

func TestIdentity_Apply(t *testing.T) {
	in := PolicyInput{Principal: "svc.worker", Method: "tools/call", ToolName: "graph_query"}
	out := sampleIdentity().Apply(in)
	if out.RunID != "run-123" || len(out.Scopes) != 2 || out.ProjectID != "proj.a" {
		t.Errorf("apply did not bind scope fields: %+v", out)
	}
	if out.ExpiresAt.IsZero() {
		t.Error("apply did not set ExpiresAt")
	}
	// An input that already carries a project_id keeps it.
	in2 := PolicyInput{ProjectID: "proj.explicit"}
	if got := sampleIdentity().Apply(in2).ProjectID; got != "proj.explicit" {
		t.Errorf("project_id = %q, want explicit argument to win", got)
	}
}
