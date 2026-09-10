// The bundle validator reads exactly one thing from the live policy
// (spec pose-only-the-signing-gate-is-read-live).
//
// Everything a sealed bundle is judged by is sealed with it, so that a setting
// flipped today cannot re-judge a review recorded years ago. One gate is
// deliberately not: `require_signed_attestations` is a bar rather than a
// permission, and a bundle sealed before a project started requiring signatures
// must not be permanently exempt.
//
// That exception is one line of code and one paragraph of ADR. Sealing it by
// symmetry with the others would look like tidying and would quietly exempt
// every existing bundle from a security requirement — so the count is asserted
// rather than remembered.

package pose

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var livePolicyFieldRe = regexp.MustCompile(`\bpolicy\.([A-Z][A-Za-z0-9]*)`)

// validateBundleAttestationWithBody returns the source of the function that
// judges an attestation against a sealed bundle.
func validateBundleAttestationWithBody(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("review_bundle.go")
	if err != nil {
		t.Fatalf("reading review_bundle.go: %v", err)
	}
	source := string(raw)
	start := strings.Index(source, "func (s Store) validateBundleAttestationWith(")
	if start < 0 {
		t.Fatal("validateBundleAttestationWith not found — this test is reading the wrong function")
	}
	rest := source[start:]
	// The next top-level declaration ends it.
	if end := strings.Index(rest[1:], "\nfunc "); end >= 0 {
		return rest[:end+1]
	}
	return rest
}

func TestOnlyTheSigningGateIsReadFromLivePolicy(t *testing.T) {
	body := validateBundleAttestationWithBody(t)
	seen := map[string]bool{}
	for _, match := range livePolicyFieldRe.FindAllStringSubmatch(body, -1) {
		seen[match[1]] = true
	}
	if !seen["RequireSignedAttestations"] {
		t.Error("the signing requirement is no longer read from the live policy — a bundle sealed before a project started requiring signatures is now permanently exempt from it")
	}
	delete(seen, "RequireSignedAttestations")
	for field := range seen {
		t.Errorf("policy.%s is read live while judging a sealed bundle: seal it into the bundle's gates, or record here why it is the second exception", field)
	}
}

// The other side of the same contract: the gates a bundle carries are the ones
// the validator consults. A gate added to the struct and never read is a
// setting that seals and does nothing.
func TestEverySealedGateIsConsulted(t *testing.T) {
	body := validateBundleAttestationWithBody(t)
	for _, gate := range []string{"AllowApprovedWithReservations", "AcceptedRiskSeverities", "AllowCriterionReuse"} {
		if !strings.Contains(body, "SealedGates()."+gate) {
			t.Errorf("%s is sealed into the bundle and never consulted while judging one", gate)
		}
	}
}
