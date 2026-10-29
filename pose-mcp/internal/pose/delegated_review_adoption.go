package pose

// Adoption composes the existing signed-run, conclusion and action contracts.
// It does not confer human authority or turn task runs into review evidence.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ReviewLimitationDelegationDisabled = "delegated-review-disabled"

func delegatedReviewCatalogEntry() CatalogEntry {
	return CatalogEntry{
		ID: "delegated-review", Summary: "independent review through a verified delegated run", IntroducedIn: "7.4.1",
		Requires: []string{"review-bundles"},
		Effect:   "a completed independent review records its verified conclusion; unavailable agents escalate to a person",
		Prerequisite: func(root string, docs PolicyDocs) string {
			if err := ValidateDelegationPaths(root); err != nil {
				return err.Error()
			}
			policy, _, found, err := (Store{Root: root, FS: docs.FS}).LoadDelegationPolicy()
			if err != nil {
				return "delegation policy is invalid: " + err.Error()
			}
			if !found || len(policy.Agents) == 0 {
				return "configure at least one agent in .pose/policy/delegation.json before adopting delegated-review"
			}
			if err := (Store{Root: root, FS: docs.FS}).ValidateDelegatedReviewActivation(policy, docs); err != nil {
				return err.Error()
			}
			return ""
		},
		Adopt: func(docs PolicyDocs, date string) []string {
			if enabled, _ := docs.Review["delegated_review"].(bool); enabled {
				return nil
			}
			docs.Review["delegated_review"] = true
			docs.Review["delegated_review_adopted_at"] = date
			return []string{"set delegated_review=true", "set delegated_review_adopted_at=" + date}
		},
		Retire: func(docs PolicyDocs) []string {
			if value, exists := docs.Review["delegated_review"].(bool); exists && !value {
				return nil
			}
			docs.Review["delegated_review"] = false
			docs.Review["delegated_review_disabled_at"] = time.Now().UTC().Format(time.DateOnly)
			return []string{"set delegated_review=false", "record delegated_review_disabled_at"}
		},
		Adopted: func(docs PolicyDocs) bool { on, _ := docs.Review["delegated_review"].(bool); return on },
	}
}

// ValidateDelegatedReviewActivation validates the actual executor and exact pin,
// not merely the presence of some issuer or a syntactically valid agent map.
func (s Store) ValidateDelegatedReviewActivation(policy DelegationPolicy, docs PolicyDocs) error {
	usable := false
	for _, agent := range policy.Agents {
		usable = usable || len(agent.Commands[DelegationAccessRead]) > 0
	}
	if !usable {
		return fmt.Errorf("configure at least one review-capable agent")
	}
	if policy.Provenance == nil || policy.Provenance.Trust != "executor-attested" || policy.ConclusionsIssuer() == "" {
		return fmt.Errorf("configure executor-attested provenance and conclusions.issuer")
	}
	if err := validateDelegationPolicy(policy); err != nil {
		return err
	}
	project, found, err := ReadProjectFileIn(s.fsys(), s.Root)
	if err != nil || !found {
		return fmt.Errorf("a declared project identity is required")
	}
	if id, _ := docs.Review["authority_project"].(string); id != project {
		return fmt.Errorf("authority_project must match the declared project")
	}
	if audience, _ := docs.Review["authority_audience"].(string); strings.TrimSpace(audience) == "" {
		return fmt.Errorf("authority_audience is required")
	}
	key, err := LoadIssuerKey(s.Root, policy.ConclusionsIssuer())
	if err != nil {
		return fmt.Errorf("load conclusions issuer %s: %w", policy.ConclusionsIssuer(), err)
	}
	if !containsString(governedStringList(docs.Review["trusted_attestation_issuers"]), key.Pin()) {
		return fmt.Errorf("pin the exact conclusions issuer for attestations before adopting delegated-review")
	}
	return nil
}

func delegationActivation(d PolicyDocs) bool {
	on, _ := d.Review["delegated_review"].(bool)
	var before map[string]any
	_ = json.Unmarshal(d.reviewOriginal, &before)
	wasOn, _ := before["delegated_review"].(bool)
	return on && !wasOn
}

func delegationDecisionChanged(d PolicyDocs) bool {
	var before map[string]any
	_ = json.Unmarshal(d.reviewOriginal, &before)
	old, had := before["delegated_review"]
	current, has := d.Review["delegated_review"]
	return had != has || old != current
}

// publishDelegationActivation serializes cooperative activations, validates the
// configuration at apply, then checks the same bytes after publication. An
// uncooperative filesystem writer cannot be locked out; an observed replacement
// fails closed and disables only our own still-unchanged published decision.
func (d PolicyDocs) publishDelegationActivation(store Store, review []byte) error {
	return d.publishDelegationActivationReading(store, review, ReadReviewAdapterFile)
}

func (d PolicyDocs) publishDelegationActivationReading(store Store, review []byte, read func(string, int64) ([]byte, error)) error {
	unlock, err := store.lockPrivateName("adoption-locks", "delegated-review")
	if err != nil {
		return err
	}
	defer unlock()
	if err := ValidateDelegationPaths(store.Root); err != nil {
		return err
	}
	_, path, found, err := store.LoadDelegationPolicy()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("delegation policy disappeared before activation")
	}
	before, err := read(filepath.Join(store.Root, path), 1<<20)
	if err != nil {
		return err
	}
	// Parse the captured bytes themselves. A prior parsed policy and a later
	// byte snapshot may belong to different uncooperative filesystem writers.
	var policy DelegationPolicy
	if path == delegationPolicyPath {
		policy, err = ParseDelegationPolicy(before)
	} else {
		var legacy ReviewersPolicy
		legacy, err = parseReviewersPolicy(before)
		if err == nil {
			policy, err = MigrateReviewersPolicy(legacy)
		}
	}
	if err != nil {
		return err
	}
	if err := store.ValidateDelegatedReviewActivation(policy, d); err != nil {
		return err
	}
	target := filepath.Join(store.Root, ".pose/policy/review.json")
	if err := store.fsys().Publish(target, review, 0o644, PublishAuthoritative, d.reviewRead); err != nil {
		return fmt.Errorf("activation publication failed; inspect the current decision before retrying: %w", err)
	}
	after, readErr := read(filepath.Join(store.Root, path), 1<<20)
	_, currentPath, currentFound, policyErr := store.LoadDelegationPolicy()
	if readErr == nil && bytes.Equal(before, after) && policyErr == nil && currentFound && currentPath == path {
		return nil
	}
	var disabled map[string]any
	if err := json.Unmarshal(review, &disabled); err != nil {
		return err
	}
	disabled["delegated_review"] = false
	raw, err := json.MarshalIndent(disabled, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := store.fsys().Publish(target, raw, 0o644, PublishAuthoritative, ExpectBytes(review, true)); err != nil {
		return fmt.Errorf("delegation policy changed during activation; decision status is unknown, inspect policy before retrying: %w", err)
	}
	return fmt.Errorf("delegation policy changed during activation; delegation was left disabled; inspect policy before adopting again")
}

// ValidateDelegationPaths rejects links before new trust is provisioned. This
// is a preflight check; arbitrary external filesystem replacement is outside
// the cooperative-writer guarantee and is detected again at activation.
func ValidateDelegationPaths(root string) error {
	for _, rel := range []string{".pose", ".pose/policy", ".pose/project.json", ".pose/policy/review.json", ".pose/policy/delegation.json", ".pose/policy/reviewers.json"} {
		info, err := os.Lstat(filepath.Join(root, rel))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing linked governance path %s", rel)
		}
	}
	return nil
}

// EscalateUnavailableDelegation raises the existing maintainer decision without
// opening a run, consuming a slot or silently weakening an adopted contract.
// Preview callers must not call it.
func (s Store) EscalateUnavailableDelegation(target, purpose, reason string, now time.Time) (string, error) {
	p, err := s.GetReviewPolicy()
	if err != nil || !p.DelegatedReview {
		return "", err
	}
	m, err := s.PrepareDelegatedReviewMaterial(target, purpose, "", nil)
	if err != nil {
		return "", err
	}
	r, err := s.PrepareActionRequest(ActionRequest{
		Origin: m.Scope, Kind: ActionDelegatedReview, Review: &m,
		Question:    "Obtain the delegated " + purpose + " of " + m.Scope,
		RequestedBy: ActionPrincipal{Principal: "agent:pose-engine"},
		Targets:     []NodeRef{{}}, Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}},
	}, now)
	if err != nil {
		return "", err
	}
	if s.PersonRequiredByPlan(m) {
		return s.EscalateDelegatedReview(r, EscalationPersonRequired, "The plan of "+m.Scope+" reserves its review for a person: who reviews it?", "Bundle "+m.BundleID+" requires mandatory-human review; a delegated run cannot satisfy it.", nil, now)
	}
	return s.EscalateDelegatedReview(r, "agents-unavailable", "No agent is available for "+purpose+" of "+m.Scope+": how should the review proceed?", strings.TrimSpace(reason), defaultEscalationOptions(), now)
}

// ValidateDelegationAdoptionDate is shared by policy readers, including reads
// of a protected Git baseline. A malformed cutoff never means "not adopted".
func ValidateDelegationAdoptionDate(date string) error {
	if date != "" {
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			return fmt.Errorf("pose: delegated_review_adopted_at must be YYYY-MM-DD")
		}
	}
	return nil
}
