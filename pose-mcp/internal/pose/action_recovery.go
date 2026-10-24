package pose

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const maxActionIntentBytes = 64 * 1024

type ActionRecovery struct {
	Phase        string `json:"phase"`
	Predecessor  string `json:"predecessor"`
	Successor    string `json:"successor"`
	IntentDigest string `json:"intent_digest,omitempty"`
	Reason       string `json:"reason,omitempty"`
}
type actionSupersessionIntent struct {
	SchemaVersion     int           `json:"schema_version"`
	Predecessor       string        `json:"predecessor"`
	PredecessorDigest string        `json:"predecessor_digest"`
	Revision          int           `json:"revision"`
	Successor         ActionRequest `json:"successor"`
	Retirement        ActionEvent   `json:"retirement"`
	Digest            string        `json:"digest"`
}

func actionIntentPath(root, id string) string {
	return filepath.Join(root, actionDir, "supersessions", id+".json")
}
func intentEvent(e ActionEvent) ActionEvent {
	e.At, e.Channel, e.Assurance = "", "", ""
	e.Claim, e.Envelope, e.SSHSignature = nil, nil, nil
	return e
}
func intentDigest(i actionSupersessionIntent) string { i.Digest = ""; d, _ := digestJSON(i); return d }
func (s Store) readActionIntent(id string) (*actionSupersessionIntent, error) {
	if err := s.checkActionJournal(id); err != nil {
		return nil, err
	}
	path := actionIntentPath(s.Root, id)
	if err := validateProjectPathIn(s.fsys(), s.Root, filepath.Join(actionDir, "supersessions", id+".json")); err != nil {
		return nil, err
	}
	st, err := s.fsys().Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > maxActionIntentBytes {
		return nil, fmt.Errorf("%w: intent must be a regular file within 64 KiB", ErrActionRecoveryUnknown)
	}
	file, err := s.fsys().Open(path)
	if err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maxActionIntentBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("%w: reading intent: %v", ErrActionRecoveryUnknown, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%w: closing intent: %v", ErrActionRecoveryUnknown, closeErr)
	}
	if len(raw) > maxActionIntentBytes {
		return nil, fmt.Errorf("%w: intent exceeds 64 KiB", ErrActionRecoveryUnknown)
	}
	var i actionSupersessionIntent
	if err := json.Unmarshal(raw, &i); err != nil {
		return nil, fmt.Errorf("%w: malformed intent: %v", ErrActionRecoveryUnknown, err)
	}
	if i.SchemaVersion != 1 || i.Predecessor != id || i.Revision < 1 || i.Digest != intentDigest(i) || i.Successor.Supersedes != id || i.Successor.RequestDigest != ActionRequestDigest(i.Successor) || i.Retirement.Type != ActionEventSuperseded || i.Retirement.SupersededBy != i.Successor.ID || i.Retirement.SuccessorDigest != i.Successor.RequestDigest || i.Retirement.ExpectedRevision != i.Revision || i.Retirement.RequestDigest != i.PredecessorDigest {
		return nil, fmt.Errorf("%w: inconsistent intent identity, revision or lineage", ErrActionRecoveryUnknown)
	}
	identity, _ := digestJSON([]string{"action-request/v1", i.Successor.RequestDigest, i.Successor.RequestedAt, i.Successor.RequestedBy.Principal, id})
	if i.Successor.ID != "act-"+identity[len("sha256:"):len("sha256:")+16] {
		return nil, fmt.Errorf("%w: inconsistent successor identity", ErrActionRecoveryUnknown)
	}
	return &i, nil
}
func (s Store) withActionRecovery(v ActionRequestView) ActionRequestView {
	id := v.Request.ID
	i, err := s.readActionIntent(id)
	// An outgoing operation takes precedence over a completed incoming link.
	if os.IsNotExist(err) && v.Request.Supersedes != "" {
		id = v.Request.Supersedes
		i, err = s.readActionIntent(id)
	}
	if err != nil && !os.IsNotExist(err) {
		v.Recovery = &ActionRecovery{Phase: "repair-required", Predecessor: id, Successor: v.Request.ID, Reason: err.Error()}
	}
	if i != nil {
		rec := &ActionRecovery{Phase: "reserved", Predecessor: id, Successor: i.Successor.ID, IntentDigest: i.Digest}
		prior, e := s.loadActionRequestBase(id)
		child, ce := s.loadActionRequestBase(i.Successor.ID)
		switch {
		case e != nil || prior.Request.RequestDigest != i.PredecessorDigest:
			rec.Phase, rec.Reason = "repair-required", "predecessor is unreadable or changed"
		case ce != nil && !os.IsNotExist(ce):
			rec.Phase, rec.Reason = "repair-required", "successor journal needs explicit repair: "+ce.Error()
		case ce == nil && (child.Request.RequestDigest != i.Successor.RequestDigest || child.Request.Supersedes != id):
			rec.Phase, rec.Reason = "repair-required", "successor lineage differs from durable intent"
		case prior.State == ActionStateSuperseded:
			last := prior.Events[len(prior.Events)-1]
			if ce == nil && last.SupersededBy == i.Successor.ID && last.SuccessorDigest == i.Successor.RequestDigest && sameActionIntent(intentEvent(last), i.Retirement) {
				rec.Phase = "complete"
			} else {
				rec.Phase, rec.Reason = "repair-required", "retirement does not match durable intent"
			}
		case prior.Revision != i.Revision:
			rec.Phase, rec.Reason = "repair-required", "predecessor revision changed after reservation"
		case ce == nil:
			rec.Phase = "successor-opened"
		}
		chain := prior
		if v.Request.ID == id {
			chain = v
		}
		if err := s.actionIncomingStatus(chain); err != nil {
			rec.Phase, rec.Reason = "repair-required", err.Error()
		}
		if v.Request.ID != id && v.Request.ID != i.Successor.ID {
			rec.Phase, rec.Reason = "repair-required", "this legacy successor conflicts with the active durable intent"
		}
		v.Recovery = rec
	} else if os.IsNotExist(err) && v.Request.Supersedes != "" {
		prior, e := s.loadActionRequestBase(id)
		rec := &ActionRecovery{Phase: "successor-opened", Predecessor: id, Successor: v.Request.ID, Reason: "legacy replacement: retry the same prepared open with current authority"}
		if e != nil {
			rec.Phase, rec.Reason = "repair-required", "predecessor is unreadable"
		} else if prior.State != ActionStateSuperseded && prior.Request.RequestDigest == v.Request.RequestDigest {
			rec.Phase, rec.Reason = "repair-required", "legacy replacement repeats material; obtain an authorized waiver or explicit repair"
		} else if prior.State == ActionStateSuperseded {
			last := prior.Events[len(prior.Events)-1]
			if last.SupersededBy == v.Request.ID {
				rec.Phase = "complete"
			} else {
				rec.Phase, rec.Reason = "repair-required", "predecessor retired to another successor"
			}
		}
		if e == nil {
			if err := s.actionIncomingStatus(prior); err != nil {
				rec.Phase, rec.Reason = "repair-required", err.Error()
			}
		}
		v.Limitations = append(v.Limitations, "legacy supersession has no durable intent; lineage is limited to recorded retirement")
		v.Recovery = rec
	}
	if v.Recovery != nil && v.Recovery.Phase != "complete" {
		v.Satisfaction = SatisfactionPending
		v.Limitations = append(v.Limitations, "supersession is "+v.Recovery.Phase+"; partial progress does not satisfy the condition")
	}
	v.NextStep = nextActionStep(v)
	return v
}

// One lock, one bounded attempt. The durable intent fences competing writers;
// neither successor existence nor an earlier preview authorizes retirement.
func (s Store) openSupersession(prepared ActionRequest, authority *ActionResolution, now time.Time) (ActionRequestView, error) {
	id := prepared.Supersedes
	file, events, release, err := s.lockedActionJournal(id)
	if err != nil {
		return ActionRequestView{}, err
	}
	defer release()
	prior := s.deriveActionView(events)
	i, err := s.readActionIntent(id)
	if err != nil && !os.IsNotExist(err) {
		return ActionRequestView{}, err
	}
	candidate, e, complete, err := s.prepareActionSupersession(prior, prepared, authority, i, now, false)
	if err != nil {
		return ActionRequestView{}, err
	}
	if complete {
		return s.LoadActionRequest(prepared.ID)
	}
	if i == nil {
		if _, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, actionDir+"/supersessions", true); err != nil {
			return ActionRequestView{}, err
		}
		raw, err := json.Marshal(candidate)
		if err != nil {
			return ActionRequestView{}, err
		}
		if len(raw) > maxActionIntentBytes {
			return ActionRequestView{}, fmt.Errorf("%w: intent exceeds 64 KiB", ErrActionRecoveryUnknown)
		}
		if err := s.fsys().CreateExclusive(actionIntentPath(s.Root, id), append(raw, '\n'), 0644, true); err != nil {
			return ActionRequestView{}, fmt.Errorf("%w: reserving supersession: %v", ErrActionRecoveryUnknown, err)
		}
	}
	if _, err := s.openPreparedAction(prepared); err != nil {
		return ActionRequestView{}, fmt.Errorf("%w: successor %s pending after reservation: %v", ErrActionRecoveryPending, prepared.ID, err)
	}
	if _, err := s.appendActionEventLocked(file, events, id, e, candidate.Revision, now); err != nil {
		return ActionRequestView{}, fmt.Errorf("%w: successor %s opened; predecessor retirement pending: %w", ErrActionRecoveryPending, prepared.ID, err)
	}
	return s.LoadActionRequest(prepared.ID)
}

func (s Store) prepareActionSupersession(prior ActionRequestView, prepared ActionRequest, authority *ActionResolution, i *actionSupersessionIntent, now time.Time, preview bool) (actionSupersessionIntent, ActionEvent, bool, error) {
	id := prepared.Supersedes
	if err := s.actionIncomingStatus(prior); err != nil {
		return actionSupersessionIntent{}, ActionEvent{}, false, err
	}
	if prior.State != ActionStateSuperseded && prior.Request.RequestDigest == prepared.RequestDigest {
		return actionSupersessionIntent{}, ActionEvent{}, false, fmt.Errorf("%w: change the material question/subject or obtain an authorized waiver", ErrActionMaterialUnchanged)
	}
	res := ActionResolution{Type: ActionEventSuperseded, RequestID: id, Actor: prepared.RequestedBy.Principal, RequestDigest: prior.Request.RequestDigest, ExpectedRevision: prior.Revision, IdempotencyKey: "supersede:" + prepared.ID, Reason: "superseded by " + prepared.ID}
	if i != nil {
		res.ExpectedRevision = i.Revision
	}
	if authority != nil {
		if authority.RequestID != "" && authority.RequestID != id {
			return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionIntentConflict
		}
		if authority.Type != "" && authority.Type != ActionEventSuperseded {
			return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionIntentConflict
		}
		res = *authority
	}
	if authority != nil && (res.SuccessorID != prepared.ID || res.SuccessorDigest != prepared.RequestDigest) {
		return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionIntentConflict
	}
	res.Type, res.RequestID, res.SuccessorID, res.SuccessorDigest = ActionEventSuperseded, id, prepared.ID, prepared.RequestDigest
	e := lifecycleEvent(res, now)
	// Retirement role normalization is material; use the same policy authority.
	policy, err := LoadActionPolicyIn(s.fsys(), s.Root)
	if err != nil {
		return actionSupersessionIntent{}, ActionEvent{}, false, err
	}
	if e.Actor != prior.Request.RequestedBy.Principal {
		role, err := authorizeActionActor(policy, prior.Request, e.Actor, e.Role)
		if err != nil {
			return actionSupersessionIntent{}, ActionEvent{}, false, err
		}
		e.Role = role
	}
	candidate := actionSupersessionIntent{SchemaVersion: 1, Predecessor: id, PredecessorDigest: res.RequestDigest, Revision: res.ExpectedRevision, Successor: prepared, Retirement: intentEvent(e)}
	candidate.Digest = intentDigest(candidate)
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return actionSupersessionIntent{}, ActionEvent{}, false, err
	}
	if len(encoded) > maxActionIntentBytes {
		return actionSupersessionIntent{}, ActionEvent{}, false, fmt.Errorf("%w: intent exceeds 64 KiB", ErrActionRecoveryUnknown)
	}
	if i != nil && i.Digest != candidate.Digest {
		return actionSupersessionIntent{}, ActionEvent{}, false, fmt.Errorf("%w: predecessor reserved for %s", ErrActionIntentConflict, i.Successor.ID)
	}
	if prior.State == ActionStateSuperseded {
		last := prior.Events[len(prior.Events)-1]
		if last.SupersededBy != prepared.ID || last.SuccessorDigest != prepared.RequestDigest {
			return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionIntentConflict
		}
		if authority != nil && !sameActionIntent(intentEvent(last), intentEvent(e)) {
			return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionIdempotencyConflict
		}
		child, err := s.loadActionRequestBase(prepared.ID)
		if err != nil || child.Request.RequestDigest != prepared.RequestDigest {
			return actionSupersessionIntent{}, ActionEvent{}, false, fmt.Errorf("%w: completed successor is missing or changed", ErrActionRecoveryUnknown)
		}
		if authority != nil && (res.Signature != "" || res.Claim != nil || res.Envelope != nil) {
			if err := s.admitLifecycleMode(prior.Request, &e, res.Signature, now, now, preview); err != nil {
				return actionSupersessionIntent{}, ActionEvent{}, false, err
			}
		}
		return candidate, e, true, nil
	}
	if res.ExpectedRevision != prior.Revision {
		return actionSupersessionIntent{}, ActionEvent{}, false, ErrActionRevisionConflict
	}
	if err := s.admitLifecycleMode(prior.Request, &e, res.Signature, now, now, preview); err != nil {
		return actionSupersessionIntent{}, ActionEvent{}, false, err
	}
	return candidate, e, false, nil
}

// PreflightActionOpen prepares identity and retirement admission without taking
// a lock, creating an intent, touching a key or promising future apply.
func (s Store) PreflightActionOpen(r ActionRequest, authority *ActionResolution, now time.Time) (ActionOpenPreflight, error) {
	out := ActionOpenPreflight{Preview: true, Acceptance: "not-certified", AuditPersisted: false}
	prepared, err := s.PrepareActionRequest(r, now)
	out.Request = prepared
	if err == nil && prepared.Supersedes != "" {
		prior, e := s.loadActionRequestBase(prepared.Supersedes)
		err = e
		if err == nil {
			i, e := s.readActionIntent(prepared.Supersedes)
			if e != nil && !os.IsNotExist(e) {
				err = e
			} else {
				_, event, _, e := s.prepareActionSupersession(prior, prepared, authority, i, now, true)
				err = e
				if e == nil {
					p, e := LoadActionPolicyIn(s.fsys(), s.Root)
					err = e
					res := lifecycleResolution(event)
					if authority != nil {
						res.Signature = authority.Signature
					}
					out.ProofPending = actionProofPending(p, res)
				}
			}
		}
	}
	out.Admissible = err == nil
	out.Refusal = actionRefusal(err)
	return out, err
}

type ActionOpenPreflight struct {
	Request        ActionRequest  `json:"request"`
	Preview        bool           `json:"preview"`
	Admissible     bool           `json:"admissible"`
	Acceptance     string         `json:"acceptance"`
	ProofPending   bool           `json:"proof_pending"`
	AuditPersisted bool           `json:"audit_persisted"`
	Refusal        *ActionRefusal `json:"refusal,omitempty"`
}

// Bound transitive lineage checks without recursively decorating views.
func (s Store) actionIncomingStatus(v ActionRequestView) error {
	seen := map[string]bool{v.Request.ID: true}
	for depth := 0; v.Request.Supersedes != ""; depth++ {
		id := v.Request.Supersedes
		if depth >= 64 || seen[id] {
			return fmt.Errorf("%w: incoming lineage %s is cyclic or exceeds 64 links", ErrActionRecoveryUnknown, id)
		}
		seen[id] = true
		prior, err := s.loadActionRequestBase(id)
		if err != nil {
			return fmt.Errorf("%w: incoming predecessor %s is unreadable: %v", ErrActionRecoveryUnknown, id, err)
		}
		i, err := s.readActionIntent(id)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("%w: incoming intent %s: %v", ErrActionRecoveryUnknown, id, err)
		}
		if prior.State != ActionStateSuperseded || len(prior.Events) < 2 {
			return fmt.Errorf("%w: incoming predecessor %s has not retired", ErrActionRecoveryPending, id)
		}
		e := prior.Events[len(prior.Events)-1]
		if prior.Request.Project != v.Request.Project || e.Type != ActionEventSuperseded || e.SupersededBy != v.Request.ID || (e.SuccessorDigest != "" && e.SuccessorDigest != v.Request.RequestDigest) {
			return fmt.Errorf("%w: incoming retirement %s does not bind this successor", ErrActionRecoveryUnknown, id)
		}
		if i != nil && (i.Successor.ID != v.Request.ID || i.Successor.RequestDigest != v.Request.RequestDigest || i.PredecessorDigest != prior.Request.RequestDigest || i.Revision != e.ExpectedRevision || !sameActionIntent(intentEvent(e), i.Retirement)) {
			return fmt.Errorf("%w: incoming intent %s does not match recorded retirement", ErrActionRecoveryUnknown, id)
		}
		v = prior
	}
	return nil
}
