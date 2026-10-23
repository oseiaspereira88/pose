package pose

import (
	"fmt"
	"strings"
)

const (
	ActionPresenceUniversal = "universal"
	ActionPresenceSSHSIG    = "sshsig"
)

// ActionPresenceError distinguishes admission failures from an invalid proof.
// It preserves the existing verification error contract for callers.
type ActionPresenceError struct{ Code, Reason string }

func (e *ActionPresenceError) Error() string {
	return fmt.Sprintf("%s: presence %s: %s", ErrActionVerificationFailed, e.Code, e.Reason)
}
func (e *ActionPresenceError) Unwrap() error { return ErrActionVerificationFailed }

// ActionPresencePolicyView describes current write policy, never historical
// compliance or a successful preview evaluation.
type ActionPresencePolicyView struct {
	SchemaVersion     int    `json:"schema_version"`
	Required          bool   `json:"required"`
	Scope             string `json:"scope"`
	ConfiguredScope   string `json:"configured_scope,omitempty"`
	MigrationRequired bool   `json:"migration_required"`
	Evaluation        string `json:"evaluation"`
}

func (p ActionPolicy) PresencePolicy() ActionPresencePolicyView {
	v := ActionPresencePolicyView{SchemaVersion: p.SchemaVersion, Required: p.RequirePresence, Scope: "disabled", ConfiguredScope: p.PresenceScope, Evaluation: "not-evaluated"}
	if p.RequirePresence {
		v.Scope = p.PresenceScope
		if p.SchemaVersion < 2 {
			v.Scope = "legacy-sshsig-only"
			v.MigrationRequired = true
		}
	}
	return v
}

// CheckPresenceMigration is also usable before invoking an external signer.
func (p ActionPolicy) CheckPresenceMigration(actor string) error {
	if p.RequirePresence && p.SchemaVersion < 2 && strings.HasPrefix(actor, "human:") {
		return &ActionPresenceError{Code: "migration-required", Reason: "legacy require_presence enforced SSHSIG only; explicitly migrate actions.json to schema_version 2 and presence_scope universal or sshsig before a new human answer"}
	}
	return nil
}

// admitActionPresence runs only AFTER all supplied proofs verified. No channel
// label, issuer grant, or identity assurance may substitute for a presence proof.
func admitActionPresence(policy ActionPolicy, actor string, signature *ActionSSHSignature, issuer bool) error {
	if !policy.RequirePresence || !strings.HasPrefix(actor, "human:") {
		return nil
	}
	if err := policy.CheckPresenceMigration(actor); err != nil {
		return err
	}
	if policy.PresenceScope != ActionPresenceUniversal && policy.PresenceScope != ActionPresenceSSHSIG {
		return &ActionPresenceError{Code: "unsupported", Reason: "the action policy has no admitted presence scope"}
	}
	if policy.PresenceScope == ActionPresenceSSHSIG && signature == nil {
		return nil
	}
	if signature == nil {
		if issuer {
			return &ActionPresenceError{Code: "unsupported", Reason: "issuer claims prove identity and authority, but contain no admitted presence proof; supply a registered SK signature asserting user presence as well"}
		}
		return &ActionPresenceError{Code: "missing", Reason: "a human answer requires a registered SK signature asserting user presence"}
	}
	if !signature.UserPresent {
		return &ActionPresenceError{Code: "missing", Reason: "the verified SSHSIG asserts no user presence; a plain key or user-verification flag alone does not satisfy require_presence"}
	}
	return nil
}

// ActionPresenceObservation is recomputed from proof, not persisted labels or
// today's policy. It does not assert an authenticated historical policy receipt.
type ActionPresenceObservation struct {
	State                 string   `json:"state"`
	Mechanism             string   `json:"mechanism"`
	HistoricalRequirement string   `json:"historical_requirement"`
	Limitations           []string `json:"limitations"`
}

func observeActionPresence(r ActionRequest, event ActionEvent) *ActionPresenceObservation {
	v := &ActionPresenceObservation{State: "not-established", Mechanism: "declared-no-proof", HistoricalRequirement: "not-recorded", Limitations: []string{"No authenticated historical presence-policy receipt is recorded; current policy is not applied retrospectively."}}
	if !strings.HasPrefix(event.Actor, "human:") {
		v.State = "not-applicable"
		v.Mechanism = "agent-is-not-human-presence"
		return v
	}
	if event.SSHSignature != nil {
		v.Mechanism = "sshsig-no-user-presence"
		if VerifyRecordedActionSignature(r, event) == nil && event.SSHSignature.UserPresent {
			v.State = "verified-key-assertion"
			v.Mechanism = "sshsig-sk-user-present"
			v.Limitations = append(v.Limitations, "This authenticates the registered key's presence assertion, not hardware provenance or the identity of the person who touched it.")
		}
	} else if event.Claim != nil || event.Envelope != nil {
		v.Mechanism = "issuer-no-presence-contract"
	}
	return v
}
