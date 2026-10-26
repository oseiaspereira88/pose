package pose

// The identity of whoever produced a subject (spec pose-delegation-routing
// R1–R3, R10). A bundle records it when it is sealed, so a reviewer is compared
// with the implementation and not with whoever happened to start the run. The
// identity is a declaration of the sealing session — detected from the
// variables its agent host sets, or declared with --lead — never a vendor
// receipt, and it says which.

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Sources of an implementation identity.
const (
	ImplementationDetected = "detected"
	ImplementationDeclared = "declared"
)

// ImplementationUnknown stands for a vendor or a model nobody declared. It
// never establishes a difference (D2).
const ImplementationUnknown = "unknown"

// Independence a run achieved against the implementation (R8).
const (
	IndependenceDifferentVendor = "different-vendor"
	IndependenceDifferentModel  = "different-model"
	IndependenceSameModel       = "same-model"
)

// ReviewBundleImplementation is the identity recorded in a sealed bundle.
// ExecutionDigest binds the sealing session's reference without storing it.
type ReviewBundleImplementation struct {
	Principal       string `json:"principal,omitempty"`
	Vendor          string `json:"vendor"`
	Model           string `json:"model"`
	ExecutionDigest string `json:"execution_digest,omitempty"`
	Source          string `json:"source"`
	// Replaces names the bundle of the same content whose identity this seal
	// replaced, so the replacement governs whatever the seal instants say
	// (spec pose-delegation-routing D3).
	Replaces string `json:"replaces,omitempty"`
}

// sessionVendorMarkers are the variables by which an agent host tells its
// children they run inside one of its sessions, and the variable among them
// that names the session.
var sessionVendorMarkers = []struct {
	vendor  string
	markers []string
	session []string
}{
	{"anthropic", []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SESSION_ID"}, []string{"CLAUDE_CODE_SESSION_ID"}},
	{"openai", []string{"CODEX_THREAD_ID", "CODEX_SESSION_ID", "CODEX_SANDBOX", "CODEX_CI"}, []string{"CODEX_THREAD_ID", "CODEX_SESSION_ID"}},
}

// ParseLead reads --lead <vendor>[/<model>].
func ParseLead(value string) (vendor, model string, err error) {
	vendor, model, _ = strings.Cut(strings.TrimSpace(value), "/")
	vendor, model = strings.ToLower(strings.TrimSpace(vendor)), strings.ToLower(strings.TrimSpace(model))
	if !runIdentityComponent.MatchString(vendor) || vendor == ImplementationUnknown || strings.Contains(vendor, "/") {
		return "", "", fmt.Errorf("pose: --lead takes <vendor>[/<model>], such as openai or anthropic/claude-opus-5-5")
	}
	if model != "" && (!runIdentityComponent.MatchString(model) || model == ImplementationUnknown || strings.Contains(model, "/")) {
		return "", "", fmt.Errorf("pose: --lead model %q is not a model name", model)
	}
	return vendor, model, nil
}

// SessionIdentity determines the identity of the session that runs the engine:
// declared with lead, or detected from getenv; the model comes from lead or
// from leadModels, and stays unknown otherwise. Markers of more than one vendor
// leave the vendor unknown rather than pick one, and say so in the warning.
func SessionIdentity(getenv func(string) string, lead string, leadModels map[string]string) (ReviewBundleImplementation, string, error) {
	id := ReviewBundleImplementation{Vendor: ImplementationUnknown, Model: ImplementationUnknown, Source: ImplementationDetected}
	warning := ""
	session := ""
	if lead != "" {
		vendor, model, err := ParseLead(lead)
		if err != nil {
			return id, "", err
		}
		id.Vendor, id.Source = vendor, ImplementationDeclared
		if model != "" {
			id.Model = model
		}
		for _, m := range sessionVendorMarkers {
			if m.vendor == vendor {
				session = firstSet(getenv, m.session)
			}
		}
	} else {
		vendors := []string{}
		for _, m := range sessionVendorMarkers {
			if firstSet(getenv, m.markers) != "" {
				vendors = append(vendors, m.vendor)
				session = firstSet(getenv, m.session)
			}
		}
		switch len(vendors) {
		case 1:
			id.Vendor = vendors[0]
		case 0:
			session = ""
		default:
			sort.Strings(vendors)
			session = ""
			warning = "the session carries the markers of " + strings.Join(vendors, " and ") + "; the implementation vendor is recorded as unknown (declare it with --lead)"
		}
	}
	if id.Model == ImplementationUnknown && id.Vendor != ImplementationUnknown {
		if model := strings.ToLower(strings.TrimSpace(leadModels[id.Vendor])); model != "" {
			id.Model = model
		}
	}
	if id.Vendor != ImplementationUnknown {
		id.Principal = "agent:" + id.Vendor + "/" + id.Model
	}
	if session != "" {
		id.ExecutionDigest = digestBytes([]byte(session))
	}
	return id, warning, nil
}

// EnvironmentSessionIdentity is SessionIdentity over the process environment.
func EnvironmentSessionIdentity(lead string, leadModels map[string]string) (ReviewBundleImplementation, string, error) {
	return SessionIdentity(os.Getenv, lead, leadModels)
}

func firstSet(getenv func(string) string, names []string) string {
	for _, name := range names {
		if value := strings.TrimSpace(getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

// normalizeIdentityPart reads a vendor or a model the way provenance does.
func normalizeIdentityPart(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

// AchievedIndependence compares an agent with the implementation. An unknown
// vendor or model on either side establishes no difference (D2).
func AchievedIndependence(implVendor, implModel, vendor, model string) string {
	known := func(v string) bool { return v != "" && v != ImplementationUnknown }
	implVendor, implModel, vendor, model = normalizeIdentityPart(implVendor), normalizeIdentityPart(implModel), normalizeIdentityPart(vendor), normalizeIdentityPart(model)
	switch {
	case known(implVendor) && known(vendor) && implVendor != vendor:
		return IndependenceDifferentVendor
	case known(implModel) && known(model) && implModel != model:
		return IndependenceDifferentModel
	}
	return IndependenceSameModel
}

// validateBundleImplementation checks a recorded identity's shape.
func validateBundleImplementation(id ReviewBundleImplementation) error {
	if id.Source != ImplementationDetected && id.Source != ImplementationDeclared {
		return fmt.Errorf("implementation source %q is not detected or declared", id.Source)
	}
	for _, v := range []string{id.Vendor, id.Model} {
		if !runIdentityComponent.MatchString(v) {
			return fmt.Errorf("implementation vendor or model %q is malformed", v)
		}
	}
	if id.Principal != "" && !validReviewPrincipal(id.Principal) {
		return fmt.Errorf("implementation principal %q is malformed", id.Principal)
	}
	if id.ExecutionDigest != "" && !validProofDigest(id.ExecutionDigest) {
		return fmt.Errorf("implementation execution digest is malformed")
	}
	if id.Replaces != "" && (!strings.HasPrefix(id.Replaces, "rvb-") || len(id.Replaces) != 20) {
		return fmt.Errorf("implementation replaces %q, which is not a bundle id", id.Replaces)
	}
	return nil
}

// DescribeImplementation renders an identity for disclosure, labelling a
// declared one as such (security of the spec).
func DescribeImplementation(id ReviewBundleImplementation) string {
	return id.Vendor + "/" + id.Model + " (" + id.Source + ")"
}
