// A bundle names the contracts that govern it
// (spec pose-bundles-seal-the-contracts-that-govern-them).
//
// The exemption that keeps a finished scope's approval when a contract arrives
// after it was compared `reviewed_at` against a date in the live policy. The
// date is editable and re-read on every verification, so an immutable bundle was
// judged by today's configuration — the thing sealing exists to prevent, and the
// argument already used to seal `selected_profiles`.
//
// Sealing the contracts in force settles it permanently. The date survives only
// as the reading of a bundle sealed before this field existed, and of the legacy
// attempt path, which has no bundle at all.

package pose

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// What is sealed has to be the registry, or a contract added to the registry
// would not govern the bundles sealed after it — the drift this replaces a date
// to avoid.
func TestSealedContractsAreTheRegistry(t *testing.T) {
	sealed := governingContractsAtSeal()
	want := []string{}
	for _, contract := range ReviewContracts() {
		want = append(want, contract.ID)
	}
	sort.Strings(want)
	if !reflect.DeepEqual(sealed, want) {
		t.Errorf("sealed = %v, registry = %v — a contract in one and not the other governs nothing", sealed, want)
	}
	if len(sealed) == 0 {
		t.Error("no contract is sealed at all, so every bundle reads as pre-contract")
	}
}

func TestBundleGovernedBySeparatesUnstampedFromUngoverned(t *testing.T) {
	unstamped := ReviewBundle{}
	if governed, stamped := BundleGovernedBy(unstamped, "evidence-vocabulary"); governed || stamped {
		t.Errorf("an unstamped bundle answered governed=%v stamped=%v", governed, stamped)
	}

	// The distinction is the whole design: "this bundle says the contract did
	// not govern it" is a permanent fact, and "this bundle says nothing" is a
	// question for the dated rule. Collapsing them into one boolean would make
	// every legacy bundle read as ungoverned.
	stampedBundle := ReviewBundle{Payload: ReviewBundlePayload{GoverningContracts: []string{"component-aware"}}}
	if governed, stamped := BundleGovernedBy(stampedBundle, "component-aware"); !governed || !stamped {
		t.Errorf("a listed contract answered governed=%v stamped=%v", governed, stamped)
	}
	if governed, stamped := BundleGovernedBy(stampedBundle, "evidence-vocabulary"); governed || !stamped {
		t.Errorf("an unlisted contract on a stamped bundle answered governed=%v stamped=%v", governed, stamped)
	}
}

// doneScopeFixture is an instance whose spec is done, which is what the dated
// exemption requires. Without it every assertion below passes for the wrong
// reason — the first version of this test did, and the mutation that should have
// failed it did not.
func doneScopeFixture(t *testing.T) (Store, ScopeRef, ReviewPolicy) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "specs", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"),
		[]byte("---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-15\n---\n\n# Spec: alpha\n\nwork\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	scope := ScopeRef{Kind: "spec", Slug: "alpha"}
	// The adoption date is in the future, so the dated rule would exempt every
	// review recorded so far. That is the edit the old mechanism could not
	// resist, and the control for everything below.
	policy := ReviewPolicy{ContractAdoptions: map[string]string{"evidence-vocabulary": "2099-01-01"}}
	if done, err := store.scopeLifecycleDone(scope); err != nil || !done {
		t.Fatalf("the fixture scope is not done (%v, %v), so the dated rule cannot fire and nothing below means anything", done, err)
	}
	if !store.reviewCompletedBeforeContract(scope, policy, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Fatal("the dated rule does not exempt this fixture, so a stamped bundle refusing the exemption proves nothing")
	}
	return store, scope, policy
}

// A bundle sealed by this engine is held to every contract, whatever the policy
// date says — including a date moved forward afterwards.
func TestAStampedBundleIgnoresTheAdoptionDate(t *testing.T) {
	store, scope, policy := doneScopeFixture(t)
	bundle := ReviewBundle{Payload: ReviewBundlePayload{GoverningContracts: governingContractsAtSeal()}}

	if store.bundleContractExempt(scope, policy, bundle, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Error("a bundle that names the contract was exempted by a future adoption date")
	}
}

// A stamped bundle that does not name the contract is exempt permanently, from
// the bundle rather than from the date — and still only for a finished scope.
func TestAStampedBundleWithoutTheContractIsExemptOnItsOwnTerms(t *testing.T) {
	store, scope, policy := doneScopeFixture(t)
	bundle := ReviewBundle{Payload: ReviewBundlePayload{GoverningContracts: []string{"component-aware"}}}

	if !store.bundleContractExempt(scope, policy, bundle, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Error("a bundle sealed before the contract existed was held to it")
	}
	// And the date is now irrelevant in that direction too: removing the
	// adoption entirely does not withdraw the exemption, because the bundle is
	// what grants it.
	if !store.bundleContractExempt(scope, ReviewPolicy{}, bundle, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Error("the exemption depended on the policy date after all")
	}
}

// And an unstamped bundle still reads by the date, because that is all it says.
func TestAnUnstampedBundleStillReadsByTheDate(t *testing.T) {
	store, scope, policy := doneScopeFixture(t)
	bundle := ReviewBundle{}

	if !store.bundleContractExempt(scope, policy, bundle, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Error("an unstamped bundle stopped taking the dated reading")
	}
	// With no adoption recorded the dated rule exempts nothing, which is the
	// existing "judge my whole history by the current contract" default.
	if store.bundleContractExempt(scope, ReviewPolicy{}, bundle, "evidence-vocabulary", "2026-09-09T12:00:00Z") {
		t.Error("an unstamped bundle with no adoption recorded was exempted")
	}
}

// Sealing adds a payload field, so a bundle sealed by this engine does not
// digest the same as one sealed before it. That is the same consequence the
// component-scoping change accepted, and it is worth pinning: the digest is the
// bundle's identity, and a field that did not change it would not be sealed.
func TestGoverningContractsAreInsideTheDigest(t *testing.T) {
	without := ReviewBundle{Payload: ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}}}
	with := without
	with.Payload.GoverningContracts = governingContractsAtSeal()
	before, err := reviewBundlePayloadDigest(without.Payload)
	if err != nil {
		t.Fatal(err)
	}
	after, err := reviewBundlePayloadDigest(with.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Error("the sealed contracts do not change the digest, so they are not sealed")
	}
}
