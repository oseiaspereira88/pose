package pose

// Principal keys: the SSH public keys a project registers for each principal
// in .pose/policy/actions.json, and the statement an answer is signed over
// (spec pose-signed-action-answers). A principal string is a declaration;
// a signature by a key the project registered for it is a proof, and needs
// no service beyond the user's own ssh-keygen.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ActionAnswerNamespace is the SSHSIG namespace of an answer signature, so a
// signature made for anything else (a commit, a file) is never accepted.
const ActionAnswerNamespace = "pose-action-answer"

// PrincipalKey is one registered key.
type PrincipalKey struct {
	Key     string `json:"key"`
	AddedAt string `json:"added_at,omitempty"`
}

// ValidatePrincipalKeys checks every registered key and that no key is
// registered to two principals.
func ValidatePrincipalKeys(keys map[string][]PrincipalKey) error {
	owner := map[string]string{}
	for principal, list := range keys {
		if !validReviewPrincipal(principal) {
			return fmt.Errorf("keys: %q is not an agent: or human: principal", principal)
		}
		for _, entry := range list {
			key, err := ParseAuthorizedKey(entry.Key)
			if err != nil {
				return fmt.Errorf("keys: %s: %v", principal, err)
			}
			fp := key.Fingerprint()
			if prior, ok := owner[fp]; ok && prior != principal {
				return fmt.Errorf("keys: %s is registered to both %s and %s; a key proves one principal", fp, prior, principal)
			}
			owner[fp] = principal
		}
	}
	return nil
}

// KeysFor returns the parsed keys registered to principal.
func (p ActionPolicy) KeysFor(principal string) []SSHPublicKey {
	var out []SSHPublicKey
	for _, entry := range p.Keys[principal] {
		if key, err := ParseAuthorizedKey(entry.Key); err == nil {
			out = append(out, key)
		}
	}
	return out
}

// KeyOwner returns the principal a key is registered to.
func (p ActionPolicy) KeyOwner(fingerprint string) (string, bool) {
	for principal := range p.Keys {
		for _, key := range p.KeysFor(principal) {
			if key.Fingerprint() == fingerprint {
				return principal, true
			}
		}
	}
	return "", false
}

// Principals lists every principal named by a role or a key.
func (p ActionPolicy) Principals() []string {
	seen := map[string]bool{}
	for _, list := range p.Roles {
		for _, principal := range list {
			seen[principal] = true
		}
	}
	for principal := range p.Keys {
		seen[principal] = true
	}
	out := make([]string, 0, len(seen))
	for principal := range seen {
		out = append(out, principal)
	}
	sort.Strings(out)
	return out
}

// RolesOf lists the roles that name principal exactly.
func (p ActionPolicy) RolesOf(principal string) []string {
	var out []string
	for role, list := range p.Roles {
		for _, candidate := range list {
			if candidate == principal {
				out = append(out, role)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ActionAnswerStatement is what an answer signature covers. It binds the
// project, the request and its version, the principal, the answer and the
// idempotency key, so a signature never fits another request, version,
// project or answer.
type ActionAnswerStatement struct {
	SchemaVersion  int    `json:"schema_version"`
	Namespace      string `json:"namespace"`
	Project        string `json:"project"`
	RequestID      string `json:"request_id"`
	RequestDigest  string `json:"request_digest"`
	Principal      string `json:"principal"`
	Answer         string `json:"answer"`
	IdempotencyKey string `json:"idempotency_key"`
	// Reason is covered by the signature when the answer carries one, so a
	// declined or deferred capability's recorded reason cannot be edited
	// after signing (spec pose-adopt-request-keeps-the-reason). Omitted when
	// empty, so statements without a reason keep their bytes.
	Reason string `json:"reason,omitempty"`
}

// Canonical is the exact byte sequence signed: compact JSON and a newline,
// which is what `pose action statement` prints for ssh-keygen to read.
func (st ActionAnswerStatement) Canonical() []byte {
	raw, _ := json.Marshal(st)
	return append(raw, '\n')
}

// AnswerStatement builds the statement for an answer to r.
func AnswerStatement(r ActionRequest, principal, answer, idempotencyKey, reason string) ActionAnswerStatement {
	return ActionAnswerStatement{SchemaVersion: 1, Namespace: ActionAnswerNamespace, Project: r.Project, RequestID: r.ID, RequestDigest: r.RequestDigest,
		Principal: principal, Answer: answer, IdempotencyKey: idempotencyKey, Reason: strings.TrimSpace(reason)}
}

// ActionSSHSignature is the proof recorded on an answer: enough to verify
// it again from the journal alone, after the key has been rotated away.
type ActionSSHSignature struct {
	Namespace    string `json:"namespace"`
	Statement    string `json:"statement"`
	Signature    string `json:"signature"`
	Key          string `json:"key"`
	Fingerprint  string `json:"fingerprint"`
	UserPresent  bool   `json:"user_present,omitempty"`
	UserVerified bool   `json:"user_verified,omitempty"`
}

// VerifyRecordedActionSignature re-verifies a recorded signature against the
// statement and key it carries, and that the statement is the event's.
func VerifyRecordedActionSignature(r ActionRequest, event ActionEvent) error {
	sig := event.SSHSignature
	if sig == nil {
		return errors.New("the event carries no signature")
	}
	want := AnswerStatement(r, event.Actor, event.Answer, event.IdempotencyKey, event.Reason)
	want.RequestDigest = event.RequestDigest
	if sig.Statement != string(want.Canonical()) {
		return errors.New("the recorded statement is not this answer's")
	}
	check, err := VerifySSHSignature(sig.Signature, ActionAnswerNamespace, []byte(sig.Statement))
	if err != nil {
		return err
	}
	if check.Key.AuthorizedKey() != sig.Key {
		return errors.New("the signature was made by another key than the one recorded")
	}
	return nil
}

// verifyActionSignature checks an answer's signature against the keys the
// policy registers for the actor.
func verifyActionSignature(policy ActionPolicy, r ActionRequest, res ActionResolution) (*ActionSSHSignature, error) {
	fail := func(reason string) error { return fmt.Errorf("%w: %s", ErrActionVerificationFailed, reason) }
	statement := AnswerStatement(r, res.Actor, res.Answer, res.IdempotencyKey, res.Reason)
	check, err := VerifySSHSignature(res.Signature, ActionAnswerNamespace, statement.Canonical())
	if err != nil {
		return nil, fail(err.Error() + " (it must sign `pose action statement` for this request version, actor, answer and idempotency key)")
	}
	registered := false
	for _, key := range policy.KeysFor(res.Actor) {
		if key.Fingerprint() == check.Key.Fingerprint() {
			registered = true
			break
		}
	}
	if !registered {
		if owner, ok := policy.KeyOwner(check.Key.Fingerprint()); ok {
			return nil, fail("the signing key " + check.Key.Fingerprint() + " is registered to " + owner + ", not " + res.Actor)
		}
		return nil, fail("the signing key " + check.Key.Fingerprint() + " is not registered to " + res.Actor + " in .pose/policy/actions.json (`pose identity add`)")
	}
	if policy.RequirePresence && strings.HasPrefix(res.Actor, "human:") && !check.UserPresent {
		return nil, fail("require_presence is set and the signature asserts no user presence; sign with a security key (ed25519-sk) and touch it")
	}
	return &ActionSSHSignature{Namespace: ActionAnswerNamespace, Statement: string(statement.Canonical()), Signature: strings.TrimSpace(res.Signature) + "\n",
		Key: check.Key.AuthorizedKey(), Fingerprint: check.Key.Fingerprint(), UserPresent: check.UserPresent, UserVerified: check.UserVerified}, nil
}

// ActionPolicyDocument is actions.json as written, keeping keys this engine
// does not model (a `_comment`) so editing a role or a key loses nothing.
type ActionPolicyDocument struct {
	raw map[string]json.RawMessage
	ActionPolicy
}

func actionPolicyPath(root string) string {
	return filepath.Join(root, ".pose", "policy", "actions.json")
}

// LoadActionPolicyDocument reads actions.json for editing; an absent file is
// the empty declared policy.
func LoadActionPolicyDocument(root string) (ActionPolicyDocument, error) {
	policy, err := LoadActionPolicy(root)
	if err != nil {
		return ActionPolicyDocument{}, err
	}
	doc := ActionPolicyDocument{raw: map[string]json.RawMessage{}, ActionPolicy: policy}
	if raw, err := os.ReadFile(actionPolicyPath(root)); err == nil {
		if err := json.Unmarshal(raw, &doc.raw); err != nil {
			return ActionPolicyDocument{}, err
		}
	}
	if doc.Keys == nil {
		doc.Keys = map[string][]PrincipalKey{}
	}
	return doc, nil
}

// AddKey registers key for principal, refusing a key another principal holds.
// It reports false when the key was already registered to principal.
func (d *ActionPolicyDocument) AddKey(principal string, key SSHPublicKey, comment, date string) (bool, error) {
	if !validReviewPrincipal(principal) {
		return false, fmt.Errorf("%q is not an agent: or human: principal", principal)
	}
	if owner, ok := d.KeyOwner(key.Fingerprint()); ok {
		if owner == principal {
			return false, nil
		}
		return false, fmt.Errorf("key %s is already registered to %s; a key proves one principal", key.Fingerprint(), owner)
	}
	line := key.AuthorizedKey()
	if comment = strings.TrimSpace(comment); comment != "" {
		line += " " + comment
	}
	d.Keys[principal] = append(d.Keys[principal], PrincipalKey{Key: line, AddedAt: date})
	return true, nil
}

// RemoveKeys removes principal's keys, or only the one with fingerprint, and
// returns the fingerprints removed.
func (d *ActionPolicyDocument) RemoveKeys(principal, fingerprint string) []string {
	var kept []PrincipalKey
	var removed []string
	for _, entry := range d.Keys[principal] {
		key, err := ParseAuthorizedKey(entry.Key)
		if err == nil && (fingerprint == "" || key.Fingerprint() == fingerprint) {
			removed = append(removed, key.Fingerprint())
			continue
		}
		kept = append(kept, entry)
	}
	if len(kept) == 0 {
		delete(d.Keys, principal)
	} else {
		d.Keys[principal] = kept
	}
	return removed
}

// GrantRole adds principal to role; false when it already held it.
func (d *ActionPolicyDocument) GrantRole(role, principal string) bool {
	for _, candidate := range d.Roles[role] {
		if candidate == principal {
			return false
		}
	}
	d.Roles[role] = append(d.Roles[role], principal)
	return true
}

// Write validates the result through the reader and writes it.
func (d ActionPolicyDocument) Write(root string) error {
	if err := ValidatePrincipalKeys(d.Keys); err != nil {
		return err
	}
	set := func(key string, value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		d.raw[key] = raw
		return nil
	}
	if _, ok := d.raw["schema_version"]; !ok {
		_ = set("schema_version", 1)
	}
	if _, ok := d.raw["identity_assurance"]; !ok {
		_ = set("identity_assurance", d.IdentityAssurance)
	}
	if err := set("roles", d.Roles); err != nil {
		return err
	}
	if len(d.Keys) > 0 {
		if err := set("keys", d.Keys); err != nil {
			return err
		}
	} else {
		delete(d.raw, "keys")
	}
	out, err := json.MarshalIndent(d.raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(actionPolicyPath(root)), 0o755); err != nil {
		return err
	}
	tmp := actionPolicyPath(root) + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, actionPolicyPath(root)); err != nil {
		return err
	}
	if _, err := LoadActionPolicy(root); err != nil {
		return fmt.Errorf("the written action policy is refused by its reader: %w", err)
	}
	return nil
}
