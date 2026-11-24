package pose

import (
	"strings"
	"testing"
)

// Spec pose-assess-leaves-sealed-checkouts-alone R3: the review plan runs
// every assessment tool in the form that writes nothing.
func TestReviewPlanRunsAssessmentsWithoutPersisting(t *testing.T) {
	scope := ScopeRef{Kind: "spec", Slug: "fixed"}
	for _, id := range []string{"assess-discover", "assess-integrate", "assess-tech-debt"} {
		for _, component := range []string{"", "core"} {
			args := strings.Join(reviewToolArgs(id, scope, component), " ")
			if !strings.Contains(args, "--no-persist") {
				t.Fatalf("%s (%q) plans a persisting run: %s", id, component, args)
			}
		}
	}
}
