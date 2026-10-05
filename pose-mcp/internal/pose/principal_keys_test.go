package pose

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-signed-action-answers.

func writeActionPolicy(t *testing.T, s Store, doc map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(s.Root, ".pose", "policy", "actions.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func keysPolicy(assurance string, keys map[string][]string, extra map[string]any) map[string]any {
	registered := map[string][]PrincipalKey{}
	for principal, lines := range keys {
		for _, line := range lines {
			registered[principal] = append(registered[principal], PrincipalKey{Key: line, AddedAt: "2026-10-05"})
		}
	}
	doc := map[string]any{"schema_version": 1, "roles": map[string][]string{"maintainer": {"human:maintainer"}}, "identity_assurance": assurance, "keys": registered}
	for k, v := range extra {
		doc[k] = v
	}
	return doc
}

func signedAnswer(view ActionRequestView, key sshTestKey, actor, value, idem string, flags byte) ActionResolution {
	res := answer(view, actor, value, idem)
	res.Signature = key.sign(ActionAnswerNamespace, AnswerStatement(view.Request, actor, value, idem).Canonical(), flags)
	return res
}

func TestSignedAnswerByARegisteredKeyIsVerifiedWithoutAnIssuer(t *testing.T) {
	s := actionFixture(t)
	key := newSSHTestKey(t, true)
	writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {key.line("yubikey")}}, nil))
	view := openedDecision(t, s)
	after, err := s.ResolveActionRequest(signedAnswer(view, key, "human:maintainer", "preserve-v1", "k1", sshSKUserPresent), time.Now())
	if err != nil {
		t.Fatalf("a signed answer by a registered key was refused: %v", err)
	}
	if after.Assurance != ReviewIdentityAssuranceVerified || after.State != ActionStateAnswered {
		t.Fatalf("not recorded as verified: %+v", after)
	}
	// A retry with the same signature is a no-op.
	if _, err := s.ResolveActionRequest(signedAnswer(view, key, "human:maintainer", "preserve-v1", "k1", sshSKUserPresent), time.Now()); err != nil {
		t.Fatalf("an identical retry was refused: %v", err)
	}
	// R7: the journal alone re-verifies the answer, after the key is gone.
	writeActionPolicy(t, s, keysPolicy("verified", nil, nil))
	reloaded, err := s.LoadActionRequest(view.Request.ID)
	if err != nil {
		t.Fatal(err)
	}
	var answered *ActionEvent
	for i := range reloaded.Events {
		if reloaded.Events[i].Type == ActionEventAnswered {
			answered = &reloaded.Events[i]
		}
	}
	if answered == nil || answered.SSHSignature == nil || !answered.SSHSignature.UserPresent {
		t.Fatalf("the signature is not in the journal: %+v", answered)
	}
	if err := VerifyRecordedActionSignature(reloaded.Request, *answered); err != nil {
		t.Fatalf("the recorded answer does not re-verify: %v", err)
	}
	tampered := *answered
	tampered.Answer = "break-v1"
	if VerifyRecordedActionSignature(reloaded.Request, tampered) == nil {
		t.Fatal("a recorded signature verified another answer")
	}
}

func TestSignedAnswerRefusesEveryMismatch(t *testing.T) {
	key := newSSHTestKey(t, false)
	other := newSSHTestKey(t, false)
	cases := map[string]func(view ActionRequestView) ActionResolution{
		"unregistered key": func(view ActionRequestView) ActionResolution {
			return signedAnswer(view, other, "human:maintainer", "preserve-v1", "k1", 0)
		},
		"another answer signed": func(view ActionRequestView) ActionResolution {
			res := signedAnswer(view, key, "human:maintainer", "break-v1", "k1", 0)
			res.Answer = "preserve-v1"
			return res
		},
		"another request version": func(view ActionRequestView) ActionResolution {
			stale := view
			stale.Request.RequestDigest = "sha256:old"
			res := signedAnswer(stale, key, "human:maintainer", "preserve-v1", "k1", 0)
			res.RequestDigest = view.Request.RequestDigest
			return res
		},
		"another idempotency key": func(view ActionRequestView) ActionResolution {
			res := signedAnswer(view, key, "human:maintainer", "preserve-v1", "k1", 0)
			res.IdempotencyKey = "k2"
			return res
		},
		"another namespace": func(view ActionRequestView) ActionResolution {
			res := answer(view, "human:maintainer", "preserve-v1", "k1")
			res.Signature = key.sign("git", AnswerStatement(view.Request, "human:maintainer", "preserve-v1", "k1").Canonical(), 0)
			return res
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			s := actionFixture(t)
			writeActionPolicy(t, s, keysPolicy("declared", map[string][]string{"human:maintainer": {key.line("")}, "human:other": {other.line("")}}, nil))
			view := openedDecision(t, s)
			if _, err := s.ResolveActionRequest(build(view), time.Now()); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
	// A key registered to someone else is named as such.
	s := actionFixture(t)
	writeActionPolicy(t, s, keysPolicy("declared", map[string][]string{"human:maintainer": {key.line("")}, "human:other": {other.line("")}}, nil))
	view := openedDecision(t, s)
	_, err := s.ResolveActionRequest(signedAnswer(view, other, "human:maintainer", "preserve-v1", "k1", 0), time.Now())
	if err == nil || !strings.Contains(err.Error(), "registered to human:other") {
		t.Fatalf("a key of another principal was not named: %v", err)
	}
}

func TestSignedAnswerRequirePresenceRefusesAnUntouchedKey(t *testing.T) {
	plain := newSSHTestKey(t, false)
	sk := newSSHTestKey(t, true)
	s := actionFixture(t)
	writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {plain.line(""), sk.line("")}}, map[string]any{"require_presence": true}))
	view := openedDecision(t, s)
	for name, res := range map[string]ActionResolution{
		"plain key":              signedAnswer(view, plain, "human:maintainer", "preserve-v1", "k1", 0),
		"security key, no touch": signedAnswer(view, sk, "human:maintainer", "preserve-v1", "k2", 0),
	} {
		if _, err := s.ResolveActionRequest(res, time.Now()); err == nil || !strings.Contains(err.Error(), "presence") {
			t.Fatalf("%s passed require_presence: %v", name, err)
		}
	}
	if _, err := s.ResolveActionRequest(signedAnswer(view, sk, "human:maintainer", "preserve-v1", "k3", sshSKUserPresent), time.Now()); err != nil {
		t.Fatalf("a touched security key was refused: %v", err)
	}
}

func TestPrincipalKeyPolicyRefusesBadKeysAndSharedKeys(t *testing.T) {
	key := newSSHTestKey(t, false)
	for name, keys := range map[string]map[string][]string{
		"malformed":       {"human:a": {"ssh-ed25519 !!!"}},
		"unsupported":     {"human:a": {"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ=="}},
		"shared":          {"human:a": {key.line("")}, "human:b": {key.line("")}},
		"not a principal": {"maintainer": {key.line("")}},
	} {
		s := actionFixture(t)
		writeActionPolicy(t, s, keysPolicy("declared", keys, nil))
		if _, err := LoadActionPolicy(s.Root); err == nil {
			t.Errorf("%s: the policy was read", name)
		}
	}
	// Editing keeps keys the engine does not model and refuses a shared key.
	s := actionFixture(t)
	if err := os.WriteFile(filepath.Join(s.Root, ".pose/policy/actions.json"), []byte(`{"_comment":"kept","schema_version":1,"roles":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := LoadActionPolicyDocument(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := ParseAuthorizedKey(key.line(""))
	if added, err := doc.AddKey("human:a", parsed, "laptop", "2026-10-05"); !added || err != nil {
		t.Fatalf("add: %v %v", added, err)
	}
	if _, err := doc.AddKey("human:b", parsed, "", "2026-10-05"); err == nil {
		t.Fatal("a key was registered to a second principal")
	}
	doc.GrantRole("maintainer", "human:a")
	if err := doc.Write(s.Root); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(s.Root, ".pose/policy/actions.json"))
	if !strings.Contains(string(raw), `"_comment": "kept"`) || !strings.Contains(string(raw), "laptop") {
		t.Fatalf("editing lost content: %s", raw)
	}
	policy, _ := LoadActionPolicy(s.Root)
	if len(policy.KeysFor("human:a")) != 1 || !policy.holds("maintainer", "human:a") {
		t.Fatalf("written policy: %+v", policy)
	}
}
