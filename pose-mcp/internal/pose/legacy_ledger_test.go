package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-signed-legacy-attestation-ledger.

type ledgerFixture struct {
	root   string
	store  Store
	bundle ReviewBundle
	key    IssuerKey
	old    ReviewAttestation
	now    time.Time
}

// ledgerFixtureWithHistory seals a bundle, records an unsigned attestation as
// a project with history would have, and pins a native issuer.
func ledgerFixtureWithHistory(t *testing.T) ledgerFixture {
	t.Helper()
	useIssuerHome(t)
	root, store := reviewBundleFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	old, err := store.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:reviewer"), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	key, err := CreateIssuerKey(root, "maintainer", now)
	if err != nil {
		t.Fatal(err)
	}
	pinNative(t, root, key, true, false)
	return ledgerFixture{root: root, store: store, bundle: bundle, key: key, old: old, now: now}
}

func (f ledgerFixture) requireSigned(t *testing.T) {
	t.Helper()
	docs, err := LoadPolicyDocs(f.root)
	if err != nil {
		t.Fatal(err)
	}
	docs.Review["require_signed_attestations"] = true
	if err := docs.Write(f.root, Store{Root: f.root}); err != nil {
		t.Fatal(err)
	}
}

func (f ledgerFixture) seal(t *testing.T) LegacyLedger {
	t.Helper()
	ledger, err := f.store.PlanLegacyLedger(f.key, f.now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := f.store.SealLegacyLedger(ledger, f.key)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func signatureBlockers(f ledgerFixture, att ReviewAttestation) []string {
	out := []string{}
	for _, b := range f.store.validateBundleAttestationWith(f.bundle, att, true) {
		if strings.Contains(b, "signed attestation proof") {
			out = append(out, b)
		}
	}
	return out
}

// R1/R3: the attestation sealed into a trusted ledger keeps counting once
// signatures are required; without the ledger it does not.
func TestLegacyLedgerKeepsSealedAttestationsValid(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	f.requireSigned(t)
	if len(signatureBlockers(f, f.old)) == 0 {
		t.Fatal("an unsigned attestation counted with no ledger")
	}
	docs, _ := LoadPolicyDocs(f.root)
	docs.Review["require_signed_attestations"] = false
	if err := docs.Write(f.root, Store{Root: f.root}); err != nil {
		t.Fatal(err)
	}
	ledger := f.seal(t)
	if len(ledger.Entries) != 1 || ledger.Entries[0].AttestationID != f.old.AttestationID || !strings.HasPrefix(ledger.Entries[0].ContentDigest, "sha256:") || len(ledger.Entries[0].ContentDigest) != 71 {
		t.Fatalf("ledger entries: %+v", ledger.Entries)
	}
	f.requireSigned(t)
	if b := signatureBlockers(f, f.old); len(b) != 0 {
		t.Fatalf("a sealed legacy attestation stopped counting: %v", b)
	}
}

// R2: sealing is refused after adoption and when nothing is unsigned.
func TestLegacyLedgerIsSealedOnlyBeforeAdoption(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	f.requireSigned(t)
	if _, err := f.store.PlanLegacyLedger(f.key, f.now); err == nil || !strings.Contains(err.Error(), "before adoption") {
		t.Fatalf("a ledger was planned after adoption: %v", err)
	}
	g := ledgerFixtureWithHistory(t)
	os.RemoveAll(filepath.Join(g.root, ".pose/review-attestations"))
	if _, err := g.store.PlanLegacyLedger(g.key, g.now); err == nil || !strings.Contains(err.Error(), "no unsigned attestation") {
		t.Fatalf("an empty ledger was planned: %v", err)
	}
}

// R4: an attestation recorded after the ledger, a wrong digest, a tampered
// ledger, an unpinned issuer and another project are each refused.
func TestLegacyLedgerRefusesEverythingOutsideIt(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	ledger := f.seal(t)
	later, err := f.store.RecordReviewAttestation(approvedBundleAttestation(f.bundle, "agent:later"), f.now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	f.requireSigned(t)
	if len(signatureBlockers(f, later)) == 0 {
		t.Fatal("an attestation recorded after the ledger counted")
	}
	path := filepath.Join(f.root, filepath.FromSlash(ledger.Path))
	original, _ := os.ReadFile(path)
	rewrite := func(edit func(*LegacyLedger)) {
		t.Helper()
		var doc LegacyLedger
		if err := json.Unmarshal(original, &doc); err != nil {
			t.Fatal(err)
		}
		edit(&doc)
		raw, _ := json.Marshal(doc)
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]func(*LegacyLedger){
		"wrong digest": func(l *LegacyLedger) { l.Entries[0].ContentDigest = "sha256:" + strings.Repeat("0", 64) },
		"tampered ledger": func(l *LegacyLedger) {
			l.Entries = append(l.Entries, LegacyLedgerEntry{AttestationID: later.AttestationID, ContentDigest: attestationContentDigest(later)})
		},
		"other project": func(l *LegacyLedger) { l.Project = "proj.elsewhere" },
	}
	for name, edit := range cases {
		rewrite(edit)
		if len(signatureBlockers(f, f.old)) == 0 && name != "tampered ledger" {
			t.Fatalf("%s: the legacy attestation still counted", name)
		}
		if name == "tampered ledger" && (len(signatureBlockers(f, later)) == 0 || len(signatureBlockers(f, f.old)) == 0) {
			t.Fatal("tampered ledger: an entry added after signing was trusted")
		}
	}
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if len(signatureBlockers(f, f.old)) != 0 {
		t.Fatal("the restored ledger is not trusted")
	}
	// Unpinning the issuer removes the ledger's trust.
	docs, _ := LoadPolicyDocs(f.root)
	docs.Review["trusted_attestation_issuers"] = []any{}
	if err := docs.Write(f.root, Store{Root: f.root}); err != nil {
		t.Fatal(err)
	}
	if len(signatureBlockers(f, f.old)) == 0 {
		t.Fatal("a ledger from an unpinned issuer was trusted")
	}
}

// R5: with history, adoption is refused only when no local pinned issuer can
// seal it, and the adoption effect seals exactly the uncovered attestations.
func TestAdoptingSignedAttestationsSealsTheHistory(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	entry, _ := LookupCatalogEntry("signed-attestations")
	docs, _ := LoadPolicyDocs(f.root)
	if blocker := CatalogAdoptBlocker(f.root, docs, entry); blocker != "" {
		t.Fatalf("a local pinned issuer is present and adoption is blocked: %q", blocker)
	}
	effects, err := PlanAdoptionEffects(f.root, "signed-attestations", "", f.now)
	if err != nil || len(effects) != 1 || !strings.Contains(effects[0].Description, "seal 1 attestation") {
		t.Fatalf("effects: %+v %v", effects, err)
	}
	if err := effects[0].Apply(); err != nil {
		t.Fatal(err)
	}
	if effects, err := PlanAdoptionEffects(f.root, "signed-attestations", "", f.now); err != nil || len(effects) != 0 {
		t.Fatalf("a covered history is sealed again: %+v %v", effects, err)
	}
	// With no local key, adoption names how to create one.
	g := ledgerFixtureWithHistory(t)
	if err := os.Remove(g.key.Path); err != nil {
		t.Fatal(err)
	}
	docs, _ = LoadPolicyDocs(g.root)
	if blocker := CatalogAdoptBlocker(g.root, docs, entry); !strings.Contains(blocker, "pose issuer init") {
		t.Fatalf("no local issuer and the blocker does not say what to do: %q", blocker)
	}
}

// R1/R5: with two local issuers pinned, --issuer chooses the sealer and its
// absence names --issuer; the catalog prerequisite alone does not block
// (found in review by agent:gpt-6.1-sol).
func TestAdoptingWithSeveralIssuersHonoursTheChoice(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	second, err := CreateIssuerKey(f.root, "second", f.now)
	if err != nil {
		t.Fatal(err)
	}
	pinNative(t, f.root, second, true, false)
	entry, _ := LookupCatalogEntry("signed-attestations")
	docs, _ := LoadPolicyDocs(f.root)
	if blocker := CatalogAdoptBlocker(f.root, docs, entry); blocker != "" {
		t.Fatalf("the prerequisite blocks a choice it cannot see: %q", blocker)
	}
	if _, err := PlanAdoptionEffects(f.root, "signed-attestations", "", f.now); err == nil || !strings.Contains(err.Error(), "--issuer") {
		t.Fatalf("an ambiguous sealer was not named: %v", err)
	}
	effects, err := PlanAdoptionEffects(f.root, "signed-attestations", "second", f.now)
	if err != nil || len(effects) != 1 || !strings.Contains(effects[0].Description, second.Pin()) {
		t.Fatalf("--issuer second was not honoured: %+v %v", effects, err)
	}
}

// R2/R4: a plan kept past adoption or past a new unsigned attestation seals
// nothing, and a symlinked ledger directory is neither written nor trusted
// (found in review by agent:gpt-6.1-sol).
func TestLegacyLedgerSealingRechecksAndRefusesSymlinkedDir(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	plan, err := f.store.PlanLegacyLedger(f.key, f.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordReviewAttestation(approvedBundleAttestation(f.bundle, "agent:later"), f.now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.SealLegacyLedger(plan, f.key); err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("a stale plan sealed: %v", err)
	}
	plan, err = f.store.PlanLegacyLedger(f.key, f.now)
	if err != nil {
		t.Fatal(err)
	}
	f.requireSigned(t)
	if _, err := f.store.SealLegacyLedger(plan, f.key); err == nil || !strings.Contains(err.Error(), "before adoption") {
		t.Fatalf("a plan sealed after adoption: %v", err)
	}

	g := ledgerFixtureWithHistory(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(g.root, ".pose", "review-ledgers")); err != nil {
		t.Fatal(err)
	}
	plan, err = g.store.PlanLegacyLedger(g.key, g.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.store.SealLegacyLedger(plan, g.key); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("a ledger was written through a symlinked directory: %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatal("a ledger landed outside the project")
	}
	// A valid ledger placed in the symlink target is still not trusted.
	os.Remove(filepath.Join(g.root, ".pose", "review-ledgers"))
	sealed := g.seal(t)
	raw, _ := os.ReadFile(filepath.Join(g.root, filepath.FromSlash(sealed.Path)))
	os.RemoveAll(filepath.Join(g.root, ".pose", "review-ledgers"))
	if err := os.WriteFile(filepath.Join(outside, "legacy.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(g.root, ".pose", "review-ledgers")); err != nil {
		t.Fatal(err)
	}
	g.requireSigned(t)
	if len(signatureBlockers(g, g.old)) == 0 {
		t.Fatal("a ledger behind a symlinked directory was trusted")
	}
}
