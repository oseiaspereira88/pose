package pose

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var lifecycleNow = time.Date(2026, 10, 24, 10, 0, 0, 123, time.UTC)

func lifecycleFixture(t *testing.T) (Store, ActionRequestView, ed25519.PrivateKey) {
	t.Helper()
	s := actionFixture(t)
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{31}, ed25519.SeedSize))
	public := private.Public().(ed25519.PublicKey)
	pin := "fixture:lifecycle#" + digestBytes(public)
	writeLifecycleFile(t, s, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"authority_project":"proj.fixture","authority_audience":"fixture-verifier","trusted_attestation_issuers":["`+pin+`"],"human_authority_issuers":["`+pin+`"]}`)
	writeLifecycleFile(t, s, ".pose/policy/actions.json", `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:maintainer"]}}`)
	v, err := s.OpenActionRequest(decisionRequest(), lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	return s, v, private
}
func writeLifecycleFile(t *testing.T, s Store, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.Root, path), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
func lifecycleRequest(v ActionRequestView, typ, actor string) ActionResolution {
	return ActionResolution{RequestID: v.Request.ID, Type: typ, Actor: actor, RequestDigest: v.Request.RequestDigest, ExpectedRevision: v.Revision, IdempotencyKey: "lifecycle-1", Reason: "condition deliberately retired"}
}
func signLifecycle(t *testing.T, v ActionRequestView, res ActionResolution, private ed25519.PrivateKey) ActionResolution {
	t.Helper()
	c := ActionAuthorityClaim{SchemaVersion: 2, Project: "proj.fixture", Audience: "fixture-verifier", RequestID: v.Request.ID, RequestDigest: res.RequestDigest,
		Principal: res.Actor, Role: "human", Issuer: "fixture:lifecycle", IssuedAt: lifecycleNow.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: lifecycleNow.Add(time.Minute).Format(time.RFC3339Nano),
		Reason: res.Reason, Operation: res.Type, ExpectedRevision: res.ExpectedRevision, IdempotencyKey: res.IdempotencyKey, SuccessorID: res.SuccessorID, SuccessorDigest: res.SuccessorDigest}
	if res.Actor[:6] == "agent:" {
		c.Role = "agent"
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	res.Claim = &c
	res.Envelope = &ActionClaimEnvelope{Issuer: c.Issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey)), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))}
	return res
}
func assertLifecycleUnchanged(t *testing.T, s Store, id string, before []byte) {
	t.Helper()
	after, err := os.ReadFile(actionJournalPath(s.Root, id))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("refusal altered journal: %v", err)
	}
}

func TestActionLifecycleWaiverProofAndHistoricalRead(t *testing.T) {
	for _, tc := range []string{"missing", "altered-operation", "altered-principal", "altered-reason", "altered-project", "altered-audience", "altered-revision", "altered-key", "unsupported-schema", "expired", "future", "valid"} {
		t.Run(tc, func(t *testing.T) {
			s, v, key := lifecycleFixture(t)
			res := signLifecycle(t, v, lifecycleRequest(v, ActionEventWaived, "human:maintainer"), key)
			switch tc {
			case "missing":
				res.Claim, res.Envelope = nil, nil
			case "altered-operation":
				res.Type = ActionEventCancelled
			case "altered-principal":
				res.Claim.Principal = "human:impostor"
			case "altered-reason":
				res.Reason = "different risk"
			case "altered-project":
				res.Claim.Project = "elsewhere"
			case "altered-audience":
				res.Claim.Audience = "elsewhere"
			case "altered-revision":
				res.ExpectedRevision = 2
			case "altered-key":
				res.IdempotencyKey = "different-key"
			case "unsupported-schema":
				res.Claim.SchemaVersion = 9
			case "expired":
				res.Claim.ExpiresAt = lifecycleNow.Add(-time.Second).Format(time.RFC3339Nano)
			case "future":
				res.Claim.IssuedAt = lifecycleNow.Add(time.Second).Format(time.RFC3339Nano)
			}
			before, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
			if err != nil {
				t.Fatal(err)
			}
			after, err := s.ResolveActionRequest(res, lifecycleNow)
			if tc != "valid" {
				if err == nil {
					t.Fatal("invalid proof waived condition")
				}
				assertLifecycleUnchanged(t, s, v.Request.ID, before)
				return
			}
			if err != nil || after.Satisfaction != SatisfactionWaived || after.Assurance != ReviewIdentityAssuranceVerified {
				t.Fatalf("valid waiver: %+v %v", after, err)
			}
			// Expiry now is irrelevant to the accepted historical instant.
			reread, err := s.LoadActionRequest(v.Request.ID)
			if err != nil || reread.Assurance != ReviewIdentityAssuranceVerified {
				t.Fatalf("historical verification: %+v %v", reread, err)
			}
			raw, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
			if err != nil {
				t.Fatal(err)
			}
			events, err := parseActionJournal(raw)
			if err != nil {
				t.Fatal(err)
			}
			events[1].Type = ActionEventCancelled
			data, _ := json.Marshal(events[0])
			other, _ := json.Marshal(events[1])
			writeLifecycleFile(t, s, filepath.Join(actionDir, v.Request.ID+".jsonl"), string(data)+"\n"+string(other)+"\n")
			if _, err := s.LoadActionRequest(v.Request.ID); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("persisted operation mutation accepted: %v", err)
			}
		})
	}
}
func TestActionLifecycleSignedBindingsRefuseValidlySignedWrongClaims(t *testing.T) {
	for _, field := range []string{"operation", "schema", "project", "audience", "revision", "successor", "expired", "future"} {
		t.Run(field, func(t *testing.T) {
			s, v, key := lifecycleFixture(t)
			res := signLifecycle(t, v, lifecycleRequest(v, ActionEventWaived, "human:maintainer"), key)
			switch field {
			case "operation":
				res.Claim.Operation = ActionEventAnswered
			case "schema":
				res.Claim.SchemaVersion = 1
			case "project":
				res.Claim.Project = "other"
			case "audience":
				res.Claim.Audience = "other"
			case "revision":
				res.Claim.ExpectedRevision++
			case "successor":
				res.Claim.SuccessorID = "act-0000000000000000"
			case "expired":
				res.Claim.ExpiresAt = lifecycleNow.Add(-time.Second).Format(time.RFC3339Nano)
			case "future":
				res.Claim.IssuedAt = lifecycleNow.Add(time.Second).Format(time.RFC3339Nano)
			}
			raw, _ := json.Marshal(res.Claim)
			res.Envelope.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, raw))
			if _, err := s.ResolveActionRequest(res, lifecycleNow); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("signed mismatched binding accepted: %v", err)
			}
		})
	}
}
func TestActionLifecycleCancellationDoesNotWaiveAnotherProducer(t *testing.T) {
	s, v, key := lifecycleFixture(t)
	own := signLifecycle(t, v, lifecycleRequest(v, ActionEventCancelled, "agent:impl"), key)
	missing := own
	missing.Claim, missing.Envelope = nil, nil
	if _, err := s.ResolveActionRequest(missing, lifecycleNow); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("requester string spoof accepted: %v", err)
	}
	third := signLifecycle(t, v, lifecycleRequest(v, ActionEventCancelled, "human:maintainer"), key)
	if _, err := s.ResolveActionRequest(third, lifecycleNow); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("recipient cancelled requester question: %v", err)
	}
	// A second independent request remains pending.
	other := decisionRequest()
	other.Question = "Independent approval?"
	otherV, err := s.OpenActionRequest(other, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.ResolveActionRequest(own, lifecycleNow)
	if err != nil || after.State != ActionStateCancelled || after.Satisfaction == SatisfactionWaived {
		t.Fatalf("own cancellation: %+v %v", after, err)
	}
	still, err := s.LoadActionRequest(otherV.Request.ID)
	if err != nil || still.Satisfaction != SatisfactionPending {
		t.Fatalf("independent producer changed: %+v %v", still, err)
	}
}
func TestActionLifecycleSupersessionAuthorizesPredecessor(t *testing.T) {
	for _, tc := range []string{"third-party", "missing-proof", "stale-revision", "wrong-successor", "valid"} {
		t.Run(tc, func(t *testing.T) {
			s, v, key := lifecycleFixture(t)
			next := decisionRequest()
			next.Question = "Replacement scope?"
			next.Supersedes = v.Request.ID
			if tc == "third-party" {
				next.RequestedBy.Principal = "agent:intruder"
			}
			prepared, err := s.PrepareActionRequest(next, lifecycleNow)
			if err != nil {
				t.Fatal(err)
			}
			res := lifecycleRequest(v, ActionEventSuperseded, next.RequestedBy.Principal)
			res.SuccessorID, res.SuccessorDigest = prepared.ID, prepared.RequestDigest
			if tc == "stale-revision" {
				res.ExpectedRevision++
			}
			res = signLifecycle(t, v, res, key)
			if tc == "missing-proof" {
				res.Claim, res.Envelope = nil, nil
			}
			if tc == "wrong-successor" {
				res.SuccessorDigest = "other"
			}
			before, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.OpenActionRequestAuthorized(prepared, &res, lifecycleNow)
			if tc != "valid" {
				if err == nil {
					t.Fatal("unauthorized retirement accepted")
				}
				assertLifecycleUnchanged(t, s, v.Request.ID, before)
				if _, err := os.Stat(actionJournalPath(s.Root, prepared.ID)); !os.IsNotExist(err) {
					t.Fatalf("refusal created successor: %v", err)
				}
				return
			}
			if err != nil || got.Request.ID != prepared.ID {
				t.Fatalf("legitimate supersession: %+v %v", got, err)
			}
			old, err := s.LoadActionRequest(v.Request.ID)
			if err != nil || old.State != ActionStateSuperseded || old.Events[1].SupersededBy != got.Request.ID || old.Events[1].SuccessorDigest != got.Request.RequestDigest {
				t.Fatalf("lineage: %+v %v", old, err)
			}
		})
	}
}
func TestActionLifecycleInvalidationIsObservedNotActorDeclared(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "self", Kind: "requirement", ID: "R4"}}
	v, err := s.OpenActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []string{"agent:intruder", "agent:pose-spec-transfer", "agent:impl", "human:maintainer"} {
		res := lifecycleRequest(v, ActionEventInvalidated, actor)
		if _, err := s.ResolveActionRequest(res, lifecycleNow); !errors.Is(err, ErrActionNotAuthorized) {
			t.Fatalf("actor simulates engine: %s %v", actor, err)
		}
	}
	if _, err := s.ResolveActionRequest(answer(v, "human:maintainer", "preserve-v1", "a1"), lifecycleNow); err != nil {
		t.Fatal(err)
	}
	writeLifecycleFile(t, s, ".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R4: Material changed.\n")
	got, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || got.State != ActionStateInvalidated || got.Revision != 2 || len(got.Limitations) == 0 {
		t.Fatalf("engine observation: %+v %v", got, err)
	}
}
func TestActionLifecycleProtectedAppendRechecksPolicy(t *testing.T) {
	s, v, key := lifecycleFixture(t)
	res := signLifecycle(t, v, lifecycleRequest(v, ActionEventWaived, "human:maintainer"), key)
	e := lifecycleEvent(res, lifecycleNow)
	if err := s.admitLifecycle(v.Request, &e, "", lifecycleNow, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleFile(t, s, ".pose/policy/actions.json", `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:someone-else"]}}`)
	if _, err := s.appendActionEvent(v.Request.ID, e, 1, lifecycleNow); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("stale preflight authorized write: %v", err)
	}
	assertLifecycleUnchanged(t, s, v.Request.ID, before)
}
func TestActionLifecycleLegacyIsDisclosedAndNotUpgraded(t *testing.T) {
	s := actionFixture(t)
	v := openedDecision(t, s)
	e := ActionEvent{Type: ActionEventWaived, Actor: "human:maintainer", Reason: "legacy", IdempotencyKey: "legacy"}
	if _, err := s.appendActionEvent(v.Request.ID, e, 1, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || got.Assurance != ReviewIdentityAssuranceDeclared || len(got.Limitations) == 0 {
		t.Fatalf("legacy not disclosed: %+v %v", got, err)
	}
	writeLifecycleFile(t, s, ".pose/policy/actions.json", `{"schema_version":2,"identity_assurance":"verified"}`)
	if _, err := s.LoadActionRequest(v.Request.ID); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("legacy waiver dispensed verified condition: %v", err)
	}
}

func TestActionLifecycleSSHSIGSeparatesOperationsAndReverifies(t *testing.T) {
	for _, tc := range []string{"answer-replay", "wrong-key", "valid", "presence-missing", "presence-valid"} {
		t.Run(tc, func(t *testing.T) {
			s := actionFixture(t)
			key := newSSHTestKey(t, true)
			extra := map[string]any{"schema_version": 2}
			if tc == "presence-missing" || tc == "presence-valid" {
				extra["require_presence"] = true
				extra["presence_scope"] = "universal"
			}
			writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {key.line("fixture")}}, extra))
			v := openedDecision(t, s)
			res := lifecycleRequest(v, ActionEventWaived, "human:maintainer")
			flags := byte(sshSKUserPresent)
			if tc == "presence-missing" {
				flags = 0
			}
			namespace := ActionLifecycleNamespace
			payload := LifecycleStatement(v.Request, res).Canonical()
			if tc == "answer-replay" {
				namespace = ActionAnswerNamespace
				payload = AnswerStatement(v.Request, res.Actor, "preserve-v1", res.IdempotencyKey, res.Reason).Canonical()
			}
			signer := key
			if tc == "wrong-key" {
				signer = newSSHTestKey(t, true)
			}
			res.Signature = signer.sign(namespace, payload, flags)
			got, err := s.ResolveActionRequest(res, lifecycleNow)
			if tc != "valid" && tc != "presence-valid" {
				if err == nil {
					t.Fatal("invalid signature/presence dispensed condition")
				}
				return
			}
			if err != nil || got.Assurance != ReviewIdentityAssuranceVerified {
				t.Fatalf("valid SSH lifecycle: %+v %v", got, err)
			}
			// Key rotation does not rewrite historical proof (B01).
			writeActionPolicy(t, s, keysPolicy("verified", nil, extra))
			if got, err := s.LoadActionRequest(v.Request.ID); err != nil || got.Assurance != ReviewIdentityAssuranceVerified {
				t.Fatalf("historical SSH rotation: %+v %v", got, err)
			}
		})
	}
}
