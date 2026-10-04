package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReviewAttribution records who did what to produce a review, kept apart
// because identity alone cannot tell four observed cases from each other: a
// person wrote the conclusion; an agent wrote it and a person adopted it; a
// person authorized a run generically; an agent wrote and applied everything
// under an authorized identity (spec pose-review-attribution-roles, the 6.3.0
// attestation cycle).
//
// Every role is optional and roles may coincide. An absent role stays absent:
// nothing here is ever filled from Reviewer.
type ReviewAttribution struct {
	SchemaVersion int    `json:"schema_version"`
	PreparedBy    string `json:"prepared_by,omitempty"`
	ConcludedBy   string `json:"concluded_by,omitempty"`
	ConfirmedBy   string `json:"confirmed_by,omitempty"`
	AppliedBy     string `json:"applied_by,omitempty"`
	// ConfirmationMode says what a confirmation meant. Adopting conclusions
	// and authorizing an operation are different acts, and a generic
	// authorization to run a cycle is never a review confirmation.
	ConfirmationMode string `json:"confirmation_mode,omitempty"`
	// ConfirmationDigest binds a confirmation to the exact decision,
	// criteria, tools and findings it confirmed. Changing any of them makes
	// the confirmation not apply to the new content.
	ConfirmationDigest string `json:"confirmation_digest,omitempty"`
}

const (
	ReviewAttributionSchemaVersion = 1

	ReviewConfirmationAdoptedConclusions  = "adopted-conclusions"
	ReviewConfirmationAuthorizedOperation = "authorized-operation"
	ReviewConfirmationNone                = "none"

	ReviewAttributionLegacy          = "legacy-undifferentiated"
	ReviewAttributionDifferentiated  = "differentiated"
	reviewConfirmationAssuranceNone  = "none"
	reviewConfirmationAssuranceStale = "unbound"
)

// ReviewConfirmationDigest is the content a confirmation binds to.
func ReviewConfirmationDigest(att ReviewAttestation) string {
	content := struct {
		BundleDigest string                  `json:"bundle_digest"`
		Decision     string                  `json:"decision"`
		Criteria     []ReviewCriterion       `json:"criteria"`
		Tools        []ReviewToolDisposition `json:"tools,omitempty"`
		Findings     []ReviewFinding         `json:"findings"`
	}{att.BundleDigest, att.Decision, att.Criteria, att.Tools, att.Findings}
	digest, _ := digestJSON(content)
	return digest
}

func validReviewPrincipal(value string) bool {
	return (strings.HasPrefix(value, "agent:") || strings.HasPrefix(value, "human:")) && len(value) > len("agent:") && !strings.ContainsAny(value, "\r\n")
}

// ValidateReviewAttribution checks the block's own consistency. It does not
// and cannot prove that a declared confirmation happened; that is what the
// assurance of the confirming principal reports.
func ValidateReviewAttribution(att ReviewAttestation) error {
	a := att.Attribution
	if a == nil {
		return nil
	}
	if a.SchemaVersion != ReviewAttributionSchemaVersion {
		return fmt.Errorf("pose: unsupported review attribution schema %d", a.SchemaVersion)
	}
	for name, value := range map[string]string{"prepared_by": a.PreparedBy, "concluded_by": a.ConcludedBy, "confirmed_by": a.ConfirmedBy, "applied_by": a.AppliedBy} {
		if value != "" && !validReviewPrincipal(value) {
			return fmt.Errorf("pose: attribution %s must be an agent: or human: principal", name)
		}
	}
	if a.PreparedBy == "" && a.ConcludedBy == "" && a.ConfirmedBy == "" && a.AppliedBy == "" {
		return fmt.Errorf("pose: an attribution block must name at least one role")
	}
	switch a.ConfirmationMode {
	case "", ReviewConfirmationNone:
		if a.ConfirmedBy != "" {
			return fmt.Errorf("pose: confirmed_by requires confirmation_mode adopted-conclusions or authorized-operation")
		}
		if a.ConfirmationDigest != "" {
			return fmt.Errorf("pose: a confirmation digest without a confirming principal confirms nothing")
		}
	case ReviewConfirmationAdoptedConclusions, ReviewConfirmationAuthorizedOperation:
		if a.ConfirmedBy == "" {
			return fmt.Errorf("pose: confirmation_mode %s requires confirmed_by", a.ConfirmationMode)
		}
		if a.ConfirmationDigest != ReviewConfirmationDigest(att) {
			return fmt.Errorf("pose: the confirmation is bound to different content than this attestation records")
		}
	default:
		return fmt.Errorf("pose: unknown confirmation_mode %q", a.ConfirmationMode)
	}
	return nil
}

// ReviewAttributionDisclosure is what review surfaces render.
type ReviewAttributionDisclosure struct {
	State                 string                        `json:"state"`
	PreparedBy            string                        `json:"prepared_by,omitempty"`
	ConcludedBy           string                        `json:"concluded_by,omitempty"`
	ConfirmedBy           string                        `json:"confirmed_by,omitempty"`
	AppliedBy             string                        `json:"applied_by,omitempty"`
	ConfirmationMode      string                        `json:"confirmation_mode"`
	ConfirmationAssurance string                        `json:"confirmation_assurance"`
	Supplements           []ReviewAttributionSupplement `json:"supplements,omitempty"`
}

// DescribeReviewAttribution discloses an attestation's attribution given the
// assurance already established for its reviewer. A confirmation is verified
// only when a verified authority claim names the confirming principal as a
// human; anything else a record says about confirmation is a declaration.
func DescribeReviewAttribution(att ReviewAttestation, assurance ReviewAssurance, supplements []ReviewAttributionSupplement) ReviewAttributionDisclosure {
	out := ReviewAttributionDisclosure{State: ReviewAttributionLegacy, ConfirmationMode: ReviewConfirmationNone, ConfirmationAssurance: reviewConfirmationAssuranceNone, Supplements: supplements}
	a := att.Attribution
	if a == nil {
		return out
	}
	out.State = ReviewAttributionDifferentiated
	out.PreparedBy, out.ConcludedBy, out.ConfirmedBy, out.AppliedBy = a.PreparedBy, a.ConcludedBy, a.ConfirmedBy, a.AppliedBy
	if a.ConfirmedBy == "" {
		return out
	}
	out.ConfirmationMode = a.ConfirmationMode
	switch {
	case a.ConfirmationDigest != ReviewConfirmationDigest(att):
		out.ConfirmationAssurance = reviewConfirmationAssuranceStale
	case assurance.VerifiedClaim != nil && assurance.VerifiedClaim.Principal == a.ConfirmedBy && assurance.VerifiedClaim.Role == "human" && strings.HasPrefix(a.ConfirmedBy, "human:"):
		out.ConfirmationAssurance = ReviewIdentityAssuranceVerified
	default:
		out.ConfirmationAssurance = ReviewIdentityAssuranceDeclared
	}
	return out
}

// RenderReviewAttribution is the one-line human form.
func RenderReviewAttribution(d ReviewAttributionDisclosure) string {
	text := ""
	if d.State == ReviewAttributionLegacy {
		text = "attribution legacy-undifferentiated (preparation, conclusion, confirmation and application were not recorded apart)"
	} else {
		parts := []string{}
		add := func(label, value string) {
			if value != "" {
				parts = append(parts, label+" "+value)
			}
		}
		add("prepared by", d.PreparedBy)
		add("concluded by", d.ConcludedBy)
		if d.ConfirmedBy != "" {
			meaning := "adopted the conclusions"
			if d.ConfirmationMode == ReviewConfirmationAuthorizedOperation {
				meaning = "authorized the operation, did not adopt the conclusions"
			}
			parts = append(parts, fmt.Sprintf("confirmed by %s (%s, %s)", d.ConfirmedBy, d.ConfirmationAssurance, meaning))
		} else {
			parts = append(parts, "no confirmation recorded")
		}
		add("applied by", d.AppliedBy)
		text = "attribution " + strings.Join(parts, ", ")
	}
	if len(d.Supplements) > 0 {
		latest := d.Supplements[len(d.Supplements)-1]
		roles := []string{}
		for _, role := range [][2]string{{"prepared by", latest.Attribution.PreparedBy}, {"concluded by", latest.Attribution.ConcludedBy}, {"applied by", latest.Attribution.AppliedBy}} {
			if role[1] != "" {
				roles = append(roles, role[0]+" "+role[1])
			}
		}
		text += fmt.Sprintf("; %d attribution supplement(s) clarify this record (declared); latest %s by %s: %s", len(d.Supplements), latest.SupplementID, latest.RecordedBy, strings.Join(roles, ", "))
	}
	return text
}

// ReviewAttributionSupplement clarifies the attribution of an existing
// attestation without altering it: the attestation's id is its content
// digest, so editing it is not an option, and history is not rewritten.
type ReviewAttributionSupplement struct {
	SchemaVersion int               `json:"schema_version"`
	SupplementID  string            `json:"supplement_id"`
	AttestationID string            `json:"attestation_id"`
	Attribution   ReviewAttribution `json:"attribution"`
	Note          string            `json:"note"`
	Evidence      []string          `json:"evidence,omitempty"`
	RecordedBy    string            `json:"recorded_by"`
	RecordedAt    string            `json:"recorded_at"`
}

const reviewAttributionSupplementDir = ".pose/review-attribution-supplements"

// RecordReviewAttributionSupplement appends a supplement. A supplement never
// carries a confirmation: confirming after the fact is a new review, not a
// clarification of an old one.
func (s Store) RecordReviewAttributionSupplement(sup ReviewAttributionSupplement, now time.Time) (ReviewAttributionSupplement, error) {
	if _, err := s.LoadReviewAttestation(sup.AttestationID); err != nil {
		return ReviewAttributionSupplement{}, fmt.Errorf("pose: attribution supplement names an unknown attestation %s", sup.AttestationID)
	}
	if !validReviewPrincipal(sup.RecordedBy) {
		return ReviewAttributionSupplement{}, fmt.Errorf("pose: recorded_by must be an agent: or human: principal")
	}
	if strings.TrimSpace(sup.Note) == "" {
		return ReviewAttributionSupplement{}, fmt.Errorf("pose: an attribution supplement needs a note saying what it clarifies and from which source")
	}
	sup.Attribution.SchemaVersion = ReviewAttributionSchemaVersion
	if sup.Attribution.ConfirmedBy != "" || sup.Attribution.ConfirmationDigest != "" || (sup.Attribution.ConfirmationMode != "" && sup.Attribution.ConfirmationMode != ReviewConfirmationNone) {
		return ReviewAttributionSupplement{}, fmt.Errorf("pose: a supplement cannot add a confirmation after the fact; record a new review instead")
	}
	probe := ReviewAttestation{Attribution: &sup.Attribution}
	if err := ValidateReviewAttribution(probe); err != nil {
		return ReviewAttributionSupplement{}, err
	}
	sup.SchemaVersion = 1
	sup.RecordedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	sort.Strings(sup.Evidence)
	identity := sup
	identity.SupplementID = ""
	digest, err := digestJSON(identity)
	if err != nil {
		return ReviewAttributionSupplement{}, err
	}
	sup.SupplementID = "ras-" + strings.TrimPrefix(digest, "sha256:")[:16]
	dir, err := ensureReviewArtifactDir(s.Root, reviewAttributionSupplementDir, true)
	if err != nil {
		return ReviewAttributionSupplement{}, err
	}
	raw, err := json.MarshalIndent(sup, "", "  ")
	if err != nil {
		return ReviewAttributionSupplement{}, err
	}
	if err := writeImmutableJSON(filepath.Join(dir, sup.SupplementID+".json"), append(raw, '\n')); err != nil {
		return ReviewAttributionSupplement{}, err
	}
	return sup, nil
}

// ListReviewAttributionSupplements returns the supplements of one
// attestation, oldest first.
func (s Store) ListReviewAttributionSupplements(attestationID string) ([]ReviewAttributionSupplement, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, filepath.FromSlash(reviewAttributionSupplementDir)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := []ReviewAttributionSupplement{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(reviewAttributionSupplementDir), entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		var sup ReviewAttributionSupplement
		if json.Unmarshal(raw, &sup) != nil || (attestationID != "" && sup.AttestationID != attestationID) {
			continue
		}
		out = append(out, sup)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RecordedAt != out[j].RecordedAt {
			return out[i].RecordedAt < out[j].RecordedAt
		}
		return out[i].SupplementID < out[j].SupplementID
	})
	return out, nil
}
