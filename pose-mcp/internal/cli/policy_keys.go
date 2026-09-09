package cli

import (
	"path/filepath"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// policyKeyCheck names one policy file, the doctor check that reports its
// unread keys, and the keys the engine actually models for it.
type policyKeyCheck struct {
	file  string
	check string
	label string
	known []string
	// legacy is the older location the engine still falls back to. Checking
	// only the current one would say nothing at all about an instance that has
	// not migrated — the silence reading as approval.
	legacy string
}

// paths returns the file locations to examine, in the order the engine reads
// them.
func (c policyKeyCheck) paths() []string {
	if c.legacy == "" {
		return []string{c.file}
	}
	return []string{c.file, c.legacy}
}

// policyKeyChecks is the set of policy files whose keys the doctor holds to
// what the engine reads. `known` comes from the structs themselves, so adding a
// field to a policy is enough — the list here never has to be updated with it.
//
// A file read by more than one struct passes all of them: capabilities.json is
// decoded once for the staleness thresholds and again for the trigger
// thresholds, and either half alone would report the other's keys as unread.
//
// policyFilesWithoutAModelledStruct, checked in policy_keys_test.go, is what
// keeps this from silently falling behind the policies the engine ships.
func policyKeyChecks() []policyKeyCheck {
	return []policyKeyCheck{
		{
			file: "review.json", check: "review.policy-keys", label: "review",
			known: posemodel.ReviewPolicyKnownKeys(),
		},
		{
			file: "delivery.json", check: "delivery.policy-keys", label: "delivery",
			known: posemodel.PolicyKnownKeys(posemodel.DeliveryPolicy{}),
		},
		{
			file: "artifacts.json", check: "artifact.policy-keys", label: "artifact",
			known: posemodel.PolicyKnownKeys(posemodel.ArtifactPolicy{}),
		},
		{
			file: "capabilities.json", check: "capability.policy-keys", label: "capability",
			known: posemodel.PolicyKnownKeys(capabilityPolicy{}, capabilityTriggerPolicy{}),
		},
		{
			file: "docs.json", check: "docs.policy-keys", label: "docs",
			known: posemodel.PolicyKnownKeys(docsReviewPolicy{}),
		},
		{
			file: "release.json", check: "release.policy-keys", label: "release",
			known: posemodel.PolicyKnownKeys(posemodel.ReleasePolicy{}),
			// LoadReleasePolicy falls back to the pre-.pose/policy location.
			legacy: filepath.Join("..", "release-policy.json"),
		},
		{
			file: "state.json", check: "state.policy-keys", label: "state",
			known: posemodel.PolicyKnownKeys(posemodel.StatePolicy{}),
		},
	}
}
