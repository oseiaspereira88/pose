package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-effective-governance-projection.

func governanceFixture(t *testing.T, review, dor string) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/policy/review.json", review)
	if dor != "" {
		write(".pose/policy/dor.json", dor)
	}
	return Store{Root: root}
}

func governanceEntry(t *testing.T, p GovernanceProjection, id string) GovernanceEntry {
	t.Helper()
	for _, e := range p.Entries {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("no governance entry %s", id)
	return GovernanceEntry{}
}

func hasReason(e GovernanceEntry, code string) bool {
	for _, r := range e.Reasons {
		if r == code {
			return true
		}
	}
	return false
}

const governanceReviewPolicy = `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14","explicit_judgment_adopted_at":"2026-09-20"}`

func TestUnadoptedCapabilitiesAreSupportedAndNotInForce(t *testing.T) {
	s := governanceFixture(t, governanceReviewPolicy, `{"schema_version":1,"adopted_at":""}`)
	p, err := s.EffectiveGovernance("")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"atomic-start", "contract-nodes", "causality-closeout"} {
		e := governanceEntry(t, p, id)
		if !e.Supported || e.Configured || e.Effective || !hasReason(e, "not-adopted") {
			t.Errorf("%s: %+v", id, e)
		}
	}
	dor := governanceEntry(t, p, "definition-of-ready")
	if dor.Effective || !hasReason(dor, "no-readiness-cutoff") {
		t.Errorf("an empty DoR adopted_at was reported as an active cutoff: %+v", dor)
	}
	verified := governanceEntry(t, p, "verified-identity")
	if verified.Effective || !hasReason(verified, "identity-assurance-declared") {
		t.Errorf("declared assurance reported as verified: %+v", verified)
	}
	judgment := governanceEntry(t, p, "explicit-judgment")
	if !judgment.Effective || judgment.Source != ContractAdoptionLegacy || judgment.Value != "2026-09-20" {
		t.Errorf("explicit-judgment legacy cutoff not projected: %+v", judgment)
	}
	causality := governanceEntry(t, p, "structural-causality")
	if !causality.Effective || !hasReason(causality, "no-adoption-recorded") {
		t.Errorf("structural-causality without a date: %+v", causality)
	}
}

func TestTheProjectionReadsWhatTheGatesRead(t *testing.T) {
	s := governanceFixture(t, `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14","atomic_start_version":1,"contract_nodes_version":1}`, `{"schema_version":1,"adopted_at":"2026-09-01"}`)
	p, _ := s.EffectiveGovernance("")
	gateAtomic, _ := s.AtomicStartAdopted()
	gateNodes, _ := s.ContractNodesAdopted()
	if governanceEntry(t, p, "atomic-start").Effective != gateAtomic || governanceEntry(t, p, "contract-nodes").Effective != gateNodes || !gateAtomic {
		t.Fatal("the projection and the gates disagree about adoption")
	}
	if !governanceEntry(t, p, "definition-of-ready").Effective {
		t.Fatal("a set DoR cutoff is not projected as in force")
	}
}

func TestScopedProjectionReportsTheBundlesStampedContracts(t *testing.T) {
	_, store := reviewBundleFixture(t)
	if _, err := store.SealReviewBundle("spec:backend", time.Now()); err != nil {
		t.Fatal(err)
	}
	p, err := store.EffectiveGovernance("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	ctx := p.ScopeBundle
	if ctx == nil || ctx.BundleID == "" || ctx.Unstamped || !ctx.Sealed {
		t.Fatalf("scope context: %+v", ctx)
	}
	if strings.Join(ctx.StampedContracts, ",") != strings.Join(governingContractsAtSeal(), ",") || len(ctx.NotStamped) != 0 {
		t.Fatalf("a bundle sealed now stamps every registry contract: %+v", ctx)
	}
	none, _ := store.EffectiveGovernance("spec:does-not-exist")
	if none.ScopeBundle == nil || none.ScopeBundle.BundleID != "" {
		t.Fatalf("a scope without bundles: %+v", none.ScopeBundle)
	}
}

func TestTheProjectionWritesNothing(t *testing.T) {
	s := governanceFixture(t, governanceReviewPolicy, `{"schema_version":1,"adopted_at":""}`)
	hash := func() string {
		h := sha256.New()
		_ = filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				raw, _ := os.ReadFile(path)
				h.Write([]byte(path))
				h.Write(raw)
			}
			return nil
		})
		return hex.EncodeToString(h.Sum(nil))
	}
	before := hash()
	if _, err := s.EffectiveGovernance("spec:alpha"); err != nil {
		t.Fatal(err)
	}
	if hash() != before {
		t.Fatal("reading effective governance changed the tree")
	}
}
