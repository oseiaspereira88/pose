package pose

// The capability catalog describes everything a project can adopt — what it
// changes day to day, since which version, whether a new instance gets it,
// what it requires — and how to turn it on and off (spec
// pose-capability-catalog). `pose adopt`, `pose setup`, install and the update
// review all read it, so a capability is described once and toggled the same
// way everywhere. A decision not to adopt is recorded too, in
// .pose/policy/adoption-decisions.json, so nothing decided is asked again.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PolicyDocs are the raw policy documents a capability writes. Raw maps keep
// every key the engine does not model; the reader validates them on Write.
type PolicyDocs struct {
	Review map[string]any
	DoR    map[string]any
	// Root is the instance the documents were loaded from, for an adoption
	// that copies a declared value (the project id) into policy.
	Root string

	reviewOriginal, dorOriginal []byte
}

func LoadPolicyDocs(root string) (PolicyDocs, error) {
	docs := PolicyDocs{Root: root}
	load := func(rel string) (map[string]any, []byte, error) {
		raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", rel))
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil, nil
		}
		if err != nil {
			return nil, nil, err
		}
		doc := map[string]any{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, nil, fmt.Errorf("pose: invalid .pose/policy/%s: %w", rel, err)
		}
		canonical, _ := json.Marshal(doc)
		return doc, canonical, nil
	}
	var err error
	if docs.Review, docs.reviewOriginal, err = load("review.json"); err != nil {
		return docs, err
	}
	if docs.DoR, docs.dorOriginal, err = load("dor.json"); err != nil {
		return docs, err
	}
	return docs, nil
}

// Rendered returns the documents as they would be written, validated by the
// readers, and which of them changed.
func (d PolicyDocs) Rendered(store Store) (review, dor []byte, reviewChanged, dorChanged bool, err error) {
	reviewCanonical, _ := json.Marshal(d.Review)
	dorCanonical, _ := json.Marshal(d.DoR)
	reviewChanged = !bytes.Equal(reviewCanonical, d.reviewOriginal)
	dorChanged = !bytes.Equal(dorCanonical, d.dorOriginal)
	if review, err = json.MarshalIndent(d.Review, "", "  "); err != nil {
		return
	}
	review = append(review, '\n')
	if dor, err = json.MarshalIndent(d.DoR, "", "  "); err != nil {
		return
	}
	dor = append(dor, '\n')
	if reviewChanged {
		if _, err = store.ParseReviewPolicyDocument(review); err != nil {
			return
		}
	}
	if dorChanged {
		var policy DoRPolicy
		if err = json.Unmarshal(dor, &policy); err != nil {
			return
		}
		if policy.AdoptedAt != "" {
			if _, perr := time.Parse("2006-01-02", policy.AdoptedAt); perr != nil {
				err = fmt.Errorf("pose: dor.json adopted_at must be YYYY-MM-DD, got %q", policy.AdoptedAt)
				return
			}
		}
	}
	return
}

// Write persists the changed documents after the readers accept them.
func (d PolicyDocs) Write(root string, store Store) error {
	review, dor, reviewChanged, dorChanged, err := d.Rendered(store)
	if err != nil {
		return err
	}
	if reviewChanged {
		if err := writePolicyFile(filepath.Join(root, ".pose", "policy", "review.json"), review); err != nil {
			return err
		}
	}
	if dorChanged {
		if err := writePolicyFile(filepath.Join(root, ".pose", "policy", "dor.json"), dor); err != nil {
			return err
		}
	}
	return nil
}

func writePolicyFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// CatalogEntry is one adoptable capability.
type CatalogEntry struct {
	ID      string
	Summary string
	// Effect says what changes in day-to-day work once it is on.
	Effect        string
	IntroducedIn  string
	DefaultForNew bool
	// Requires are catalog capabilities that must be on first. Implies are
	// capabilities this one turns on with itself, which therefore cannot be
	// turned off while it is on.
	Requires []string
	Implies  []string
	// Prerequisite reports what outside the catalog is missing ("" when met).
	Prerequisite func(root string, docs PolicyDocs) string
	Adopt        func(docs PolicyDocs, date string) []string
	Retire       func(docs PolicyDocs) []string
	Adopted      func(docs PolicyDocs) bool
}

func governedEntry(c GovernedCapability, introduced, effect string, implies ...string) CatalogEntry {
	return CatalogEntry{
		ID: c.ID, Summary: c.Summary, Effect: effect, IntroducedIn: introduced, DefaultForNew: true, Implies: implies,
		Adopt:   func(docs PolicyDocs, date string) []string { return AdoptGovernedCapability(docs.Review, c, date) },
		Retire:  func(docs PolicyDocs) []string { return RetireGovernedCapability(docs.Review, c) },
		Adopted: func(docs PolicyDocs) bool { return GovernedCapabilityAdopted(docs.Review, c) },
	}
}

func overlayEntry(name, introduced, effect string) CatalogEntry {
	ref := name + "@1"
	return CatalogEntry{
		ID: "overlay:" + name, Summary: "review overlay " + ref, Effect: effect, IntroducedIn: introduced,
		Prerequisite: func(root string, _ PolicyDocs) string {
			if _, err := os.Stat(filepath.Join(root, ".pose", "review-profiles", name+".json")); err != nil {
				return "the review profile " + ref + " is not installed; run `pose update` to install the shipped profiles"
			}
			return ""
		},
		Adopt: func(docs PolicyDocs, date string) []string {
			overlays := governedStringList(docs.Review["overlay_profiles"])
			if containsString(overlays, ref) {
				return nil
			}
			docs.Review["overlay_profiles"] = toAnyList(append(overlays, ref))
			dates, _ := docs.Review["overlay_adopted_at"].(map[string]any)
			if dates == nil {
				dates = map[string]any{}
			}
			dates[ref] = date
			docs.Review["overlay_adopted_at"] = dates
			return []string{"add overlay_profiles " + ref, "set overlay_adopted_at." + ref + "=" + date}
		},
		Retire: func(docs PolicyDocs) []string {
			return RetireGovernedCapability(docs.Review, GovernedCapability{Overlay: ref})
		},
		Adopted: func(docs PolicyDocs) bool {
			return containsString(governedStringList(docs.Review["overlay_profiles"]), ref)
		},
	}
}

func schemaVersionOf(docs PolicyDocs) int {
	if v, ok := docs.Review["schema_version"].(float64); ok {
		return int(v)
	}
	if v, ok := docs.Review["schema_version"].(int); ok {
		return v
	}
	return 0
}

func intKeyIs(docs PolicyDocs, key string, want int) bool {
	switch v := docs.Review[key].(type) {
	case float64:
		return int(v) == want
	case int:
		return v == want
	}
	return false
}

func catalogEntries() []CatalogEntry {
	byID := map[string]GovernedCapability{}
	for _, c := range governedCapabilities {
		byID[c.ID] = c
	}
	trustedIssuer := func(_ string, docs PolicyDocs) string {
		if len(governedStringList(docs.Review["trusted_attestation_issuers"])) == 0 {
			return "no trusted issuer is pinned in trusted_attestation_issuers; create a native one with pose issuer init <name> and pose issuer pin <name> --attestations --human-authority --apply, or pin an external issuer such as a Harne8 installation"
		}
		return ""
	}
	entries := []CatalogEntry{
		governedEntry(byID["agency-readiness"], "7.0.0", "an action request that is not satisfied refuses the start, close or release it restricts"),
		governedEntry(byID["atomic-start"], "6.0.0", "a spec moves from draft to in-progress with `pose start --apply`, which records the baseline it is reconciled against"),
		governedEntry(byID["causality-closeout"], "6.0.0", "a review that answers for observed structure maps each material fact to a requirement or decision, with its own reason", "overlay:structural-materiality"),
		governedEntry(byID["contract-nodes"], "6.0.0", "a change to a started spec's requirements, assumptions or decisions is recorded with `pose amend`"),
		{
			ID: "criterion-reuse", Summary: "unchanged review criteria are carried forward from a superseded bundle", IntroducedIn: "1.1.0", DefaultForNew: true,
			Effect:  "a resealed bundle keeps the conclusions of criteria whose inputs did not change",
			Adopt:   func(docs PolicyDocs, _ string) []string { return setBool(docs.Review, "allow_criterion_reuse", true) },
			Retire:  func(docs PolicyDocs) []string { return setBool(docs.Review, "allow_criterion_reuse", false) },
			Adopted: func(docs PolicyDocs) bool { v, _ := docs.Review["allow_criterion_reuse"].(bool); return v },
		},
		{
			ID: "definition-of-ready", Summary: "the entry gate for specs", IntroducedIn: "4.0.0", DefaultForNew: true,
			Effect: "a spec created on or after the date cannot move to in-progress until Intent, Requirements with stable ids and Technical Plan are filled",
			Adopt: func(docs PolicyDocs, date string) []string {
				if v, _ := docs.DoR["adopted_at"].(string); v != "" {
					return nil
				}
				docs.DoR["adopted_at"] = date
				if _, ok := docs.DoR["schemaVersion"]; !ok {
					docs.DoR["schemaVersion"] = 1
				}
				return []string{"set dor.json adopted_at=" + date}
			},
			Retire: func(docs PolicyDocs) []string {
				if v, _ := docs.DoR["adopted_at"].(string); v == "" {
					return nil
				}
				docs.DoR["adopted_at"] = ""
				return []string{"clear dor.json adopted_at"}
			},
			Adopted: func(docs PolicyDocs) bool { v, _ := docs.DoR["adopted_at"].(string); return v != "" },
		},
		overlayEntry("backend-review", "1.1.0", "a spec touching a backend component is also reviewed for contracts, input errors, concurrency, observability and integration impact"),
		overlayEntry("engineering-judgment", "6.0.0", "a scope selected by its declared risk is also reviewed for engineering judgment"),
		overlayEntry("frontend-review", "1.1.0", "a spec touching a frontend component is also reviewed for accessibility, state and rendering"),
		overlayEntry("high-criticality-review", "6.0.0", "a scope touching a critical component carries the stricter review its criticality asks for"),
		overlayEntry("structural-materiality", "6.0.0", "a spec that changes dependencies, components, public or governance contracts owes a structural review"),
		{
			ID: "qualified-artifact-refs", Summary: "references across projects", IntroducedIn: "6.0.0",
			Effect: "a spec, roadmap or milestone of another project is referenced as `xref:<project>/…` and resolved through the bound project",
			Adopt: func(docs PolicyDocs, _ string) []string {
				if intKeyIs(docs, "qualified_artifact_refs_version", 1) {
					return nil
				}
				changes := []string{"set qualified_artifact_refs_version=1"}
				docs.Review["qualified_artifact_refs_version"] = 1
				if schemaVersionOf(docs) < QualifiedArtifactPolicySchemaVersion {
					docs.Review["schema_version"] = QualifiedArtifactPolicySchemaVersion
					changes = append(changes, fmt.Sprintf("set schema_version=%d", QualifiedArtifactPolicySchemaVersion))
				}
				return changes
			},
			Retire: func(docs PolicyDocs) []string {
				if !intKeyIs(docs, "qualified_artifact_refs_version", 1) {
					return nil
				}
				delete(docs.Review, "qualified_artifact_refs_version")
				docs.Review["schema_version"] = ReviewPolicySchemaVersion
				return []string{"remove qualified_artifact_refs_version", fmt.Sprintf("set schema_version=%d", ReviewPolicySchemaVersion)}
			},
			Adopted: func(docs PolicyDocs) bool { return intKeyIs(docs, "qualified_artifact_refs_version", 1) },
		},
		{
			ID: "spec-authority-transfer", Summary: "moving a spec's authority between projects", IntroducedIn: "6.0.0", Requires: []string{"qualified-artifact-refs"},
			Effect: "a spec's authority moves to another project with `pose spec-transfer`, leaving a redirect every reference follows",
			Adopt: func(docs PolicyDocs, _ string) []string {
				if intKeyIs(docs, "spec_authority_transfer_version", 1) {
					return nil
				}
				docs.Review["spec_authority_transfer_version"] = 1
				docs.Review["schema_version"] = SpecAuthorityTransferPolicySchemaVersion
				return []string{"set spec_authority_transfer_version=1", fmt.Sprintf("set schema_version=%d", SpecAuthorityTransferPolicySchemaVersion)}
			},
			Retire: func(docs PolicyDocs) []string {
				if !intKeyIs(docs, "spec_authority_transfer_version", 1) {
					return nil
				}
				delete(docs.Review, "spec_authority_transfer_version")
				docs.Review["schema_version"] = QualifiedArtifactPolicySchemaVersion
				return []string{"remove spec_authority_transfer_version", fmt.Sprintf("set schema_version=%d", QualifiedArtifactPolicySchemaVersion)}
			},
			Adopted: func(docs PolicyDocs) bool { return intKeyIs(docs, "spec_authority_transfer_version", 1) },
		},
		{
			ID: "signed-attestations", Summary: "signatures on review attestations", IntroducedIn: "5.0.0",
			Prerequisite: func(root string, docs PolicyDocs) string {
				if missing := trustedIssuer(root, docs); missing != "" {
					return missing
				}
				// Signing stays live for old bundles, so adopting seals the
				// unsigned history with a local pinned issuer (spec
				// pose-signed-legacy-attestation-ledger).
				if blocker := signedHistoryBlocker(root, docs, "", false); blocker != "" {
					return blocker
				}
				return ""
			},
			Effect: "a review attestation counts only when a trusted issuer signed it",
			Adopt: func(docs PolicyDocs, _ string) []string {
				return setBool(docs.Review, "require_signed_attestations", true)
			},
			Retire:  func(docs PolicyDocs) []string { return setBool(docs.Review, "require_signed_attestations", false) },
			Adopted: func(docs PolicyDocs) bool { v, _ := docs.Review["require_signed_attestations"].(bool); return v },
		},
		{
			ID: "verified-identity", Summary: "verified reviewer identity", IntroducedIn: "6.0.0",
			Effect: "a reviewer's identity is read from a signed authority claim, not from the name the reviewer writes",
			Prerequisite: func(root string, docs PolicyDocs) string {
				if missing := trustedIssuer(root, docs); missing != "" {
					return missing
				}
				if audience, _ := docs.Review["authority_audience"].(string); strings.TrimSpace(audience) == "" {
					return "authority_audience is not set: name the verifier installation claims are addressed to"
				}
				if project, _ := docs.Review["authority_project"].(string); project == "" {
					if _, ok, err := ReadProjectFile(root); !ok || err != nil {
						return "no project identity: declare it in .pose/project.json so authority_project can name it"
					}
				}
				return ""
			},
			Adopt: func(docs PolicyDocs, _ string) []string {
				changes := []string{}
				// Verified assurance needs authority_project; the prerequisite
				// accepts the declared identity for it, so adoption writes it.
				if project, _ := docs.Review["authority_project"].(string); project == "" && docs.Root != "" {
					if id, ok, err := ReadProjectFile(docs.Root); ok && err == nil {
						docs.Review["authority_project"] = id
						changes = append(changes, "set authority_project="+id+" (from .pose/project.json)")
					}
				}
				levels, _ := docs.Review["identity_assurance"].(map[string]any)
				if levels == nil {
					levels = map[string]any{}
				}
				for _, scope := range []string{"spec", "milestone", "roadmap"} {
					if levels[scope] != ReviewIdentityAssuranceVerified {
						levels[scope] = ReviewIdentityAssuranceVerified
						changes = append(changes, "set identity_assurance."+scope+"=verified")
					}
				}
				docs.Review["identity_assurance"] = levels
				return changes
			},
			Retire: func(docs PolicyDocs) []string {
				if _, ok := docs.Review["identity_assurance"]; !ok {
					return nil
				}
				delete(docs.Review, "identity_assurance")
				return []string{"remove identity_assurance"}
			},
			Adopted: func(docs PolicyDocs) bool {
				levels, _ := docs.Review["identity_assurance"].(map[string]any)
				for _, level := range levels {
					if level == ReviewIdentityAssuranceVerified {
						return true
					}
				}
				return false
			},
		},
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

func setBool(doc map[string]any, key string, value bool) []string {
	if current, ok := doc[key].(bool); ok && current == value {
		return nil
	}
	if _, present := doc[key]; !present && !value {
		return nil
	}
	doc[key] = value
	return []string{fmt.Sprintf("set %s=%t", key, value)}
}

// CapabilityCatalog returns every adoptable capability, sorted by id.
func CapabilityCatalog() []CatalogEntry { return catalogEntries() }

// LookupCatalogEntry finds one capability by id.
func LookupCatalogEntry(id string) (CatalogEntry, bool) {
	for _, entry := range catalogEntries() {
		if entry.ID == id {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}

// CatalogIDs lists every catalog id.
func CatalogIDs() []string {
	ids := []string{}
	for _, entry := range catalogEntries() {
		ids = append(ids, entry.ID)
	}
	return ids
}

// CatalogAdoptBlocker names what must happen before entry can be adopted.
func CatalogAdoptBlocker(root string, docs PolicyDocs, entry CatalogEntry) string {
	for _, required := range entry.Requires {
		if dependency, ok := LookupCatalogEntry(required); ok && !dependency.Adopted(docs) {
			return entry.ID + " requires " + required + "; run `pose adopt " + required + " --apply` first"
		}
	}
	if entry.Prerequisite != nil {
		return entry.Prerequisite(root, docs)
	}
	return ""
}

// CatalogRetireBlocker names the adopted capability that depends on entry.
func CatalogRetireBlocker(docs PolicyDocs, entry CatalogEntry) string {
	for _, other := range catalogEntries() {
		if other.ID == entry.ID || !other.Adopted(docs) {
			continue
		}
		for _, dependency := range append(append([]string{}, other.Requires...), other.Implies...) {
			if dependency == entry.ID {
				return other.ID + " depends on " + entry.ID + "; turn " + other.ID + " off first"
			}
		}
	}
	return ""
}

// Adoption decisions record that a project looked at a capability and chose
// not to adopt it, now or for the time being.
const (
	AdoptionDeclined = "declined"
	AdoptionDeferred = "deferred"
)

type AdoptionDecision struct {
	Decision string `json:"decision"`
	Date     string `json:"date"`
	Reason   string `json:"reason"`
	// Request is the action request whose answer recorded it, when one did.
	Request string `json:"request,omitempty"`
	// Version is the engine release the decision was taken under; a
	// deferral is asked again once the engine moves past it (spec
	// pose-setup-command).
	Version string `json:"version,omitempty"`
}

type AdoptionDecisions struct {
	SchemaVersion int `json:"schema_version"`
	// ReviewedVersion is the engine release whose capabilities this project
	// last reviewed: a capability introduced after it, and undecided, is new
	// (spec pose-setup-command). A fresh install records its own version.
	ReviewedVersion string                      `json:"reviewed_version,omitempty"`
	Decisions       map[string]AdoptionDecision `json:"decisions"`
}

func adoptionDecisionsPath(root string) string {
	return filepath.Join(root, ".pose", "policy", "adoption-decisions.json")
}

func ReadAdoptionDecisions(root string) (AdoptionDecisions, error) {
	out := AdoptionDecisions{SchemaVersion: 1, Decisions: map[string]AdoptionDecision{}}
	raw, err := os.ReadFile(adoptionDecisionsPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("pose: invalid .pose/policy/adoption-decisions.json: %w", err)
	}
	if out.SchemaVersion != 1 {
		return out, fmt.Errorf("pose: unsupported adoption-decisions schema %d", out.SchemaVersion)
	}
	if out.Decisions == nil {
		out.Decisions = map[string]AdoptionDecision{}
	}
	return out, nil
}

func writeAdoptionDecisions(root string, doc AdoptionDecisions) error {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writePolicyFile(adoptionDecisionsPath(root), append(raw, '\n'))
}

// RecordAdoptionDecision records a decline or a deferral.
func RecordAdoptionDecision(root, id, decision, reason, date string) error {
	return RecordAdoptionDecisionFor(root, id, AdoptionDecision{Decision: decision, Reason: reason, Date: date})
}

// RecordAdoptionDecisionFor records a full decision, including the request
// that carried it.
func RecordAdoptionDecisionFor(root, id string, decision AdoptionDecision) error {
	if _, ok := LookupCatalogEntry(id); !ok {
		return fmt.Errorf("pose: unknown capability %s; governed capabilities: %s", id, strings.Join(CatalogIDs(), ", "))
	}
	if decision.Decision != AdoptionDeclined && decision.Decision != AdoptionDeferred {
		return fmt.Errorf("pose: an adoption decision is declined or deferred, got %q", decision.Decision)
	}
	if strings.TrimSpace(decision.Reason) == "" {
		return fmt.Errorf("pose: a %s capability needs a reason, so the decision can be revisited", decision.Decision)
	}
	if _, err := time.Parse("2006-01-02", decision.Date); err != nil {
		return fmt.Errorf("pose: decision date must be YYYY-MM-DD, got %q", decision.Date)
	}
	doc, err := ReadAdoptionDecisions(root)
	if err != nil {
		return err
	}
	doc.Decisions[id] = decision
	return writeAdoptionDecisions(root, doc)
}

// SetReviewedVersion records the engine release whose capabilities the
// project has reviewed.
func SetReviewedVersion(root, version string) error {
	doc, err := ReadAdoptionDecisions(root)
	if err != nil {
		return err
	}
	if doc.ReviewedVersion == version {
		return nil
	}
	doc.ReviewedVersion = version
	return writeAdoptionDecisions(root, doc)
}

// CompareReleaseVersions orders two `MAJOR.MINOR.PATCH` versions (a suffix
// after `-` is ignored); an unreadable part counts as zero.
func CompareReleaseVersions(a, b string) int {
	parse := func(v string) [3]int {
		var out [3]int
		v = strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v")
		for i, part := range strings.SplitN(v, ".", 3) {
			n := 0
			for _, r := range part {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			out[i] = n
		}
		return out
	}
	pa, pb := parse(a), parse(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// CapabilitiesToReview returns the capabilities this project has not decided
// for the engine release it runs: off, undecided and introduced after the
// reviewed version (every off, undecided one when none is recorded), and
// deferrals taken under an older release. Capabilities that need setup are
// not decisions yet and are not returned.
func CapabilitiesToReview(root, engine string) ([]CapabilityState, error) {
	states, err := CapabilityStates(root)
	if err != nil {
		return nil, err
	}
	decisions, err := ReadAdoptionDecisions(root)
	if err != nil {
		return nil, err
	}
	out := []CapabilityState{}
	for _, state := range states {
		switch state.State {
		case CapabilityOff:
			if decisions.ReviewedVersion == "" || CompareReleaseVersions(state.IntroducedIn, decisions.ReviewedVersion) > 0 {
				out = append(out, state)
			}
		case CapabilityDeferred:
			if state.Decision != nil && (state.Decision.Version == "" || CompareReleaseVersions(state.Decision.Version, engine) < 0) {
				out = append(out, state)
			}
		}
	}
	return out, nil
}

// ClearAdoptionDecision forgets a recorded decision, as adopting does.
func ClearAdoptionDecision(root, id string) error {
	doc, err := ReadAdoptionDecisions(root)
	if err != nil {
		return err
	}
	if _, ok := doc.Decisions[id]; !ok {
		return nil
	}
	delete(doc.Decisions, id)
	return writeAdoptionDecisions(root, doc)
}

// Capability states.
const (
	CapabilityOn         = "on"
	CapabilityOff        = "off"
	CapabilityDeclined   = "declined"
	CapabilityDeferred   = "deferred"
	CapabilityNeedsSetup = "needs-setup"
)

type CapabilityState struct {
	ID            string            `json:"id"`
	State         string            `json:"state"`
	Summary       string            `json:"summary"`
	Effect        string            `json:"effect"`
	IntroducedIn  string            `json:"introduced_in"`
	DefaultForNew bool              `json:"default_for_new"`
	Requires      []string          `json:"requires,omitempty"`
	Missing       string            `json:"missing,omitempty"`
	Decision      *AdoptionDecision `json:"decision,omitempty"`
}

// CapabilityStates reads every catalog entry's state in this instance.
func CapabilityStates(root string) ([]CapabilityState, error) {
	docs, err := LoadPolicyDocs(root)
	if err != nil {
		return nil, err
	}
	decisions, err := ReadAdoptionDecisions(root)
	if err != nil {
		return nil, err
	}
	out := []CapabilityState{}
	for _, entry := range catalogEntries() {
		state := CapabilityState{ID: entry.ID, Summary: entry.Summary, Effect: entry.Effect, IntroducedIn: entry.IntroducedIn, DefaultForNew: entry.DefaultForNew, Requires: entry.Requires}
		switch {
		case entry.Adopted(docs):
			state.State = CapabilityOn
		default:
			if decision, ok := decisions.Decisions[entry.ID]; ok {
				copy := decision
				state.Decision = &copy
				state.State = decision.Decision
			} else if entry.Prerequisite != nil && entry.Prerequisite(root, docs) != "" {
				state.State, state.Missing = CapabilityNeedsSetup, entry.Prerequisite(root, docs)
			} else {
				state.State = CapabilityOff
				state.Missing = CatalogAdoptBlocker(root, docs, entry)
			}
		}
		out = append(out, state)
	}
	return out, nil
}
