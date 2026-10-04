package pose

import (
	"fmt"
	"sort"
	"strings"
)

// GovernanceProjection says which governance this instance actually applies,
// as opposed to which the engine can apply (spec
// pose-effective-governance-projection). It is introspection over the same
// readers the gates call — the registry, the capability fields, the DoR and
// delivery policies — and writes nothing. A capability shipped in the engine
// and a capability in force are different facts.
type GovernanceProjection struct {
	SchemaVersion int               `json:"schema_version"`
	EngineVersion string            `json:"engine_version,omitempty"`
	Entries       []GovernanceEntry `json:"entries"`
	// Scope and ScopeBundle are set when the projection is read for one
	// scope: the contracts its newest sealed bundle stamped are compared with
	// what the policy would seal today.
	Scope       string                  `json:"scope,omitempty"`
	ScopeBundle *GovernanceScopeContext `json:"scope_bundle,omitempty"`
}

// GovernanceEntry is one contract, capability or gate.
type GovernanceEntry struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Supported  bool   `json:"supported"`
	Configured bool   `json:"configured"`
	Applicable bool   `json:"applicable"`
	Effective  bool   `json:"effective"`
	// Source names where the configuration was read from.
	Source string `json:"source,omitempty"`
	// Value is the configured value as read (a date, a version, a mode).
	Value string `json:"value,omitempty"`
	// Reasons are stable codes for every false above, plus notes about how
	// the entry behaves (a cutoff, a dependency).
	Reasons []string `json:"reasons,omitempty"`
	// Explanation is a rendering for people.
	Explanation string `json:"explanation"`
}

// GovernanceScopeContext compares one scope's sealed bundle with today.
type GovernanceScopeContext struct {
	BundleID         string   `json:"bundle_id,omitempty"`
	Sealed           bool     `json:"sealed"`
	StampedContracts []string `json:"stamped_contracts,omitempty"`
	Unstamped        bool     `json:"unstamped,omitempty"`
	// NotStamped are registry contracts the bundle does not carry: it was
	// sealed before they existed and is never held to them.
	NotStamped        []string `json:"not_stamped,omitempty"`
	IdentityAssurance string   `json:"identity_assurance,omitempty"`
}

const GovernanceProjectionSchemaVersion = 1

// Governance entry kinds.
const (
	GovernanceContract   = "contract"
	GovernanceCapability = "capability"
	GovernanceGate       = "gate"
)

// EffectiveGovernance projects the instance's governance. scope may be empty.
func (s Store) EffectiveGovernance(scope string) (GovernanceProjection, error) {
	out := GovernanceProjection{SchemaVersion: GovernanceProjectionSchemaVersion}
	policy, raw, err := s.loadReviewPolicy()
	if err != nil {
		return out, err
	}
	reviewOn := policy.Enabled && policy.SchemaVersion >= ReviewPolicySchemaVersion
	bundlesOn := reviewOn && policy.ReviewBundles

	for _, contract := range ReviewContracts() {
		entry := GovernanceEntry{ID: contract.ID, Kind: GovernanceContract, Supported: true}
		source := ContractAdoptionSource{Source: ContractAdoptionAbsent}
		if raw != nil {
			if parsed, parseErr := ContractAdoptionSourceOf(raw, contract.ID); parseErr == nil {
				source = parsed
			}
		}
		entry.Source = source.Source
		entry.Value = source.Date
		entry.Configured = source.Source != ContractAdoptionAbsent
		entry.Applicable = bundlesOn
		entry.Effective = bundlesOn
		if !reviewOn {
			entry.Reasons = append(entry.Reasons, "review-policy-disabled")
		} else if !policy.ReviewBundles {
			entry.Reasons = append(entry.Reasons, "review-bundles-not-adopted")
		}
		switch {
		case source.Date != "":
			entry.Reasons = append(entry.Reasons, "legacy-cutoff:"+source.Date)
		case entry.Configured:
			entry.Reasons = append(entry.Reasons, "explicit-no-cutoff")
		default:
			entry.Reasons = append(entry.Reasons, "no-adoption-recorded")
		}
		if source.LegacyShadowed {
			entry.Reasons = append(entry.Reasons, "legacy-key-shadowed")
		}
		cutoff := "no adoption date is recorded, so reviews recorded before the contract are judged by it"
		if source.Date != "" {
			cutoff = "unstamped history completed by " + source.Date + " is read under the dated exemption"
		} else if entry.Configured {
			cutoff = "an explicit empty adoption judges all history by the current contract"
		}
		if entry.Effective {
			entry.Explanation = fmt.Sprintf("every new bundle seals %s (%s); %s", contract.ID, contract.Summary, cutoff)
		} else {
			entry.Explanation = fmt.Sprintf("%s is known to the engine but no bundle is sealed under it here; %s", contract.ID, cutoff)
		}
		out.Entries = append(out.Entries, entry)
	}

	capability := func(id, key string, configured bool, value string, effective bool, reasons []string, explanation string) {
		entry := GovernanceEntry{ID: id, Kind: GovernanceCapability, Supported: true, Configured: configured, Value: value, Source: key, Applicable: configured, Effective: effective, Explanation: explanation}
		if !configured {
			entry.Reasons = append(entry.Reasons, "not-adopted")
		}
		entry.Reasons = append(entry.Reasons, reasons...)
		out.Entries = append(out.Entries, entry)
	}
	version := func(v int) string {
		if v == 0 {
			return ""
		}
		return fmt.Sprint(v)
	}
	capability("atomic-start", "review.atomic_start_version", policy.AtomicStartVersion == AtomicStartPolicyVersion, version(policy.AtomicStartVersion),
		policy.AtomicStartVersion == AtomicStartPolicyVersion, nil,
		pick(policy.AtomicStartVersion == AtomicStartPolicyVersion, "`pose start --apply` records the R/A/D baseline and moves draft to in-progress under a lock", "supported and not adopted: `pose start` previews, `--apply` is refused"))
	capability("contract-nodes", "review.contract_nodes_version", policy.ContractNodesVersion == ContractNodesPolicyVersion, version(policy.ContractNodesVersion),
		policy.ContractNodesVersion == ContractNodesPolicyVersion, nil,
		pick(policy.ContractNodesVersion == ContractNodesPolicyVersion, "amendments track requirements, assumptions and decisions as schema-2 nodes", "supported and not adopted: amendments stay requirement-only"))
	causality := policy.CausalityCloseoutVersion == CausalityCloseoutPolicyVersion
	var causalityReasons []string
	if causality && !bundlesOn {
		causalityReasons = append(causalityReasons, "requires-review-bundles")
	}
	capability("causality-closeout", "review.causality_closeout_version", causality, version(policy.CausalityCloseoutVersion),
		causality && bundlesOn, causalityReasons,
		pick(causality && bundlesOn, "new bundles stamp the causality-closeout contract", "supported and not in force: no bundle stamps causality-closeout"))
	capability("qualified-artifact-refs", "review.qualified_artifact_refs_version", policy.QualifiedArtifactRefsVersion == 1, version(policy.QualifiedArtifactRefsVersion),
		policy.QualifiedArtifactRefsVersion == 1, nil, pick(policy.QualifiedArtifactRefsVersion == 1, "dependencies and members may name another project with xref:", "supported and not adopted: references stay local"))
	capability("spec-authority-transfer", "review.spec_authority_transfer_version", policy.SpecAuthorityTransferVersion == 1, version(policy.SpecAuthorityTransferVersion),
		policy.SpecAuthorityTransferVersion == 1, nil, pick(policy.SpecAuthorityTransferVersion == 1, "specs may move between projects through preview/apply/resume", "supported and not adopted"))
	capability("criterion-reuse", "review.allow_criterion_reuse", policy.AllowCriterionReuse, pick(policy.AllowCriterionReuse, "true", ""),
		policy.AllowCriterionReuse && bundlesOn, nil, pick(policy.AllowCriterionReuse, "unchanged criteria may be carried forward from a superseded bundle", "every criterion is answered afresh"))
	capability("signed-attestations", "review.require_signed_attestations", policy.RequireSignedAttestations, pick(policy.RequireSignedAttestations, "true", ""),
		policy.RequireSignedAttestations && bundlesOn, nil, pick(policy.RequireSignedAttestations, "an attestation needs a trusted signed envelope", "attestations need no signature"))

	kinds := []string{}
	for kind, assurance := range policy.IdentityAssurance {
		if assurance == ReviewIdentityAssuranceVerified {
			kinds = append(kinds, kind)
		}
	}
	sort.Strings(kinds)
	verified := len(kinds) > 0
	var verifiedReasons []string
	verifiedEffective := verified && bundlesOn
	if verified && (policy.AuthorityAudience == "" || len(policy.TrustedAttestationIssuers) == 0) {
		verifiedReasons = append(verifiedReasons, "incomplete-verified-configuration")
		verifiedEffective = false
	}
	if !verified {
		verifiedReasons = append(verifiedReasons, "identity-assurance-declared")
	}
	capability("verified-identity", "review.identity_assurance", verified, strings.Join(kinds, ","), verifiedEffective, verifiedReasons,
		pick(verified, "reviewer identity and separation for "+strings.Join(kinds, ", ")+" come from signed claims", "identity assurance is declared: reviewer identity is the string the reviewer writes and separation is not authenticated"))

	dor := LoadDoRPolicy(s.Root)
	dorEntry := GovernanceEntry{ID: "definition-of-ready", Kind: GovernanceGate, Supported: true, Source: "dor.adopted_at", Value: dor.AdoptedAt,
		Configured: dor.AdoptedAt != "", Applicable: dor.AdoptedAt != "", Effective: dor.AdoptedAt != ""}
	if dor.AdoptedAt == "" {
		dorEntry.Reasons = []string{"no-readiness-cutoff"}
		dorEntry.Explanation = "adopted_at is empty, so no spec is held to the Definition of Ready by readiness; the gate exists and is not in force"
	} else {
		dorEntry.Explanation = "specs created on or after " + dor.AdoptedAt + " need acceptance criteria with stable ids before they are ready"
	}
	out.Entries = append(out.Entries, dorEntry)

	delivery, deliveryErr := LoadDeliveryPolicy(s.Root)
	deliveryEntry := GovernanceEntry{ID: "delivery-integrity", Kind: GovernanceGate, Supported: true, Source: "delivery.enabled"}
	if deliveryErr == nil {
		deliveryEntry.Configured = delivery.Enabled
		deliveryEntry.Value = delivery.AdoptedAt
		deliveryEntry.Applicable = delivery.Enabled
		deliveryEntry.Effective = delivery.Enabled && delivery.AdoptedAt != ""
		if !delivery.Enabled {
			deliveryEntry.Reasons = append(deliveryEntry.Reasons, "not-adopted")
		} else if delivery.AdoptedAt == "" {
			deliveryEntry.Reasons = append(deliveryEntry.Reasons, "no-adoption-date")
		}
	} else {
		deliveryEntry.Reasons = []string{"policy-unreadable"}
	}
	deliveryEntry.Explanation = pick(deliveryEntry.Effective, "declared delivery targets must be proven reachable with current evidence before close", "delivery targets are not gated here")
	out.Entries = append(out.Entries, deliveryEntry)

	if scope != "" {
		out.Scope = scope
		ctx := &GovernanceScopeContext{}
		bundles, listErr := s.ListReviewBundles(scope)
		if listErr != nil {
			return out, listErr
		}
		if len(bundles) > 0 {
			latest := bundles[len(bundles)-1]
			ctx.BundleID, ctx.Sealed = latest.BundleID, latest.State == "sealed"
			ctx.StampedContracts = append([]string(nil), latest.Payload.GoverningContracts...)
			ctx.Unstamped = len(ctx.StampedContracts) == 0
			stamped := map[string]bool{}
			for _, id := range ctx.StampedContracts {
				stamped[id] = true
			}
			if !ctx.Unstamped {
				for _, contract := range ReviewContracts() {
					if !stamped[contract.ID] {
						ctx.NotStamped = append(ctx.NotStamped, contract.ID)
					}
				}
			}
			ctx.IdentityAssurance = latest.Payload.SealedGates().IdentityAssurance
		}
		out.ScopeBundle = ctx
	}
	return out, nil
}

func pick(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}
