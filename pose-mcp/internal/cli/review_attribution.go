package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdReviewAttributionSupplement clarifies who prepared, concluded and
// applied an existing attestation without altering it (spec
// pose-review-attribution-roles). It cannot add a confirmation.
func cmdReviewAttributionSupplement(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose review attribution-supplement <attestation-id> --recorded-by <principal> --note <text> [--prepared-by <p>] [--concluded-by <p>] [--applied-by <p>] [--evidence <ref>]... [--apply]"
	out := render(stdout, stderr)
	var sup posemodel.ReviewAttributionSupplement
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--recorded-by", "--note", "--prepared-by", "--concluded-by", "--applied-by", "--evidence":
			if i+1 >= len(args) {
				out.Failure(usage)
				return 2
			}
			i++
			switch args[i-1] {
			case "--recorded-by":
				sup.RecordedBy = args[i]
			case "--note":
				sup.Note = args[i]
			case "--prepared-by":
				sup.Attribution.PreparedBy = args[i]
			case "--concluded-by":
				sup.Attribution.ConcludedBy = args[i]
			case "--applied-by":
				sup.Attribution.AppliedBy = args[i]
			case "--evidence":
				sup.Evidence = append(sup.Evidence, args[i])
			}
		case "--confirmed-by", "--confirmation-mode":
			out.Failure("pose review attribution-supplement: a supplement cannot add a confirmation after the fact; record a new review instead")
			return 2
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || sup.AttestationID != "" {
				out.Failure(usage)
				return 2
			}
			sup.AttestationID = args[i]
		}
	}
	if sup.AttestationID == "" {
		out.Failure(usage)
		return 2
	}
	store := posemodel.Store{Root: root}
	if !apply {
		if _, err := store.LoadReviewAttestation(sup.AttestationID); err != nil {
			out.Failure(fmt.Sprintf("pose review attribution-supplement: unknown attestation %s", sup.AttestationID))
			return 1
		}
		out.Field("review_attribution_supplement.attestation_id", sup.AttestationID)
		out.Field("review_attribution_supplement.apply", "false")
		return 0
	}
	recorded, err := store.RecordReviewAttributionSupplement(sup, time.Now())
	if err != nil {
		out.Failure(fmt.Sprintf("pose review attribution-supplement: %v", err))
		return 1
	}
	out.Field("review_attribution_supplement.id", recorded.SupplementID)
	out.Field("review_attribution_supplement.attestation_id", recorded.AttestationID)
	out.Field("review_attribution_supplement.apply", "true")
	return 0
}
