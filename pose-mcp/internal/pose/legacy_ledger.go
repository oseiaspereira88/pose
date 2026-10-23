package pose

// Signed legacy attestation ledgers (spec pose-signed-legacy-attestation-ledger).
//
// Requiring signed attestations is deliberately live for every bundle, old
// ones included (spec pose-reuse-is-sealed-signing-stays-live): a cutoff date
// would exempt an attestation forged later on an old bundle. Attestations are
// immutable and content-addressed, so the ones recorded before a project
// adopted signing can never carry an envelope. A ledger closes that gap
// without a date: a pinned issuer signs, before adoption, the exact set of
// unsigned attestations that exist, each by its full content digest. An
// unsigned attestation then counts only if a trusted ledger names it,
// unchanged; anything recorded or altered afterwards is outside every ledger.

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LegacyLedgerSchemaVersion versions the ledger document.
const LegacyLedgerSchemaVersion = 1

const legacyLedgerDir = ".pose/review-ledgers"

// LegacyLedgerEntry names one attestation by id and full content digest.
type LegacyLedgerEntry struct {
	AttestationID string `json:"attestation_id"`
	ContentDigest string `json:"content_digest"`
}

// LegacyLedger is the signed set of attestations recorded before signing was
// required.
type LegacyLedger struct {
	SchemaVersion int                 `json:"schema_version"`
	Kind          string              `json:"kind"`
	Project       string              `json:"project"`
	Issuer        string              `json:"issuer"`
	PublicKey     string              `json:"public_key"`
	SealedAt      string              `json:"sealed_at"`
	Entries       []LegacyLedgerEntry `json:"entries"`
	Signature     string              `json:"signature,omitempty"`
	Path          string              `json:"-"`
}

// attestationContentDigest is the full digest the attestation id truncates.
func attestationContentDigest(att ReviewAttestation) string {
	identity := att
	identity.AttestationID = ""
	identity.Path = ""
	identity.Envelope = nil
	digest, _ := digestJSON(identity)
	return digest
}

func (l LegacyLedger) signable() []byte {
	unsigned := l
	unsigned.Signature = ""
	raw, _ := json.Marshal(unsigned)
	return raw
}

// ledgerProject is the project id a ledger is bound to: the review policy's
// authority_project, else the instance's declared project id.
func (s Store) ledgerProject() string {
	if policy, _, err := s.loadReviewPolicy(); err == nil && policy.AuthorityProject != "" {
		return policy.AuthorityProject
	}
	if id, ok, _ := ReadProjectFileIn(s.fsys(), s.Root); ok {
		return id
	}
	return ""
}

// UnsignedAttestations lists recorded attestations without an envelope.
func (s Store) UnsignedAttestations() ([]ReviewAttestation, error) {
	read, err := s.ReadReviewAttestations("")
	if err != nil {
		return nil, err
	}
	// The set of unsigned attestations is what a ledger declares and what adopting
	// signatures re-judges: a set read with a hole in it is not that set.
	if !read.Coverage.Complete() {
		return nil, fmt.Errorf("pose: the attestation history cannot be read completely, so the unsigned attestations are not known: %s", read.Coverage.Describe(""))
	}
	out := []ReviewAttestation{}
	for _, att := range read.Attestations {
		if att.Envelope == nil {
			out = append(out, att)
		}
	}
	return out, nil
}

// PlanLegacyLedger builds the ledger key would sign for the attestations that
// exist now. It refuses once signing is required: a ledger sealed after
// adoption could cover attestations recorded unsigned in breach of policy.
func (s Store) PlanLegacyLedger(key IssuerKey, now time.Time) (LegacyLedger, error) {
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return LegacyLedger{}, err
	}
	if policy.RequireSignedAttestations {
		return LegacyLedger{}, fmt.Errorf("pose: review policy already requires signed attestations; a legacy ledger is sealed before adoption, never after")
	}
	unsigned, err := s.UnsignedAttestations()
	if err != nil {
		return LegacyLedger{}, err
	}
	if len(unsigned) == 0 {
		return LegacyLedger{}, fmt.Errorf("pose: no unsigned attestation to seal")
	}
	ledger := LegacyLedger{SchemaVersion: LegacyLedgerSchemaVersion, Kind: "legacy-attestations", Project: s.ledgerProject(),
		Issuer: key.Name, PublicKey: key.PublicKey(), SealedAt: now.UTC().Truncate(time.Second).Format(time.RFC3339)}
	for _, att := range unsigned {
		ledger.Entries = append(ledger.Entries, LegacyLedgerEntry{AttestationID: att.AttestationID, ContentDigest: attestationContentDigest(att)})
	}
	sort.Slice(ledger.Entries, func(i, j int) bool { return ledger.Entries[i].AttestationID < ledger.Entries[j].AttestationID })
	return ledger, nil
}

// SealLegacyLedger signs and writes ledger. It re-checks, at the moment of
// writing, everything the plan was computed from — the policy still does not
// require signing and the unsigned history is exactly the planned entries —
// so a plan kept past adoption, or past a new unsigned attestation, seals
// nothing (found in review by agent:gpt-6.1-sol). It never replaces a file
// and never writes through a symlinked directory.
func (s Store) SealLegacyLedger(ledger LegacyLedger, key IssuerKey) (LegacyLedger, error) {
	if key.private == nil || key.Name != ledger.Issuer || key.PublicKey() != ledger.PublicKey {
		return LegacyLedger{}, fmt.Errorf("pose: the ledger was planned for another issuer key")
	}
	current, err := s.PlanLegacyLedger(key, time.Now())
	if err != nil {
		return LegacyLedger{}, err
	}
	if current.Project != ledger.Project || !sameLedgerEntries(current.Entries, ledger.Entries) {
		return LegacyLedger{}, fmt.Errorf("pose: the unsigned history changed since the ledger was planned; plan the adoption again")
	}
	ledger.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key.private, ledger.signable()))
	dir, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, legacyLedgerDir, true)
	if err != nil {
		return LegacyLedger{}, err
	}
	stamp, _ := time.Parse(time.RFC3339, ledger.SealedAt)
	name := fmt.Sprintf("legacy-%s-%s.json", ledger.Issuer, stamp.UTC().Format("20060102T150405Z"))
	raw, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return LegacyLedger{}, err
	}
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return LegacyLedger{}, err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		file.Close()
		os.Remove(path)
		return LegacyLedger{}, err
	}
	if err := file.Close(); err != nil {
		return LegacyLedger{}, err
	}
	ledger.Path = filepath.ToSlash(filepath.Join(legacyLedgerDir, name))
	return ledger, nil
}

func sameLedgerEntries(a, b []LegacyLedgerEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// trustedLegacyEntries returns the entries of every ledger whose signature
// verifies under a pinned attestation issuer and whose project is this one,
// keyed by attestation id. Ledgers that fail are skipped, never trusted.
func (s Store) trustedLegacyEntries() (map[string]string, error) {
	entries, _, err := s.trustedLegacyEntriesRead()
	return entries, err
}

// trustedLegacyEntriesRead is trustedLegacyEntries and the ledgers it could not use for a
// reason other than not being this project's or not being pinned: a file it could not
// read, a ledger that does not parse, one of another version, one whose key or
// signature does not verify. None of them is trusted, and none is mistaken for absence
// either (spec pose-source-coverage-integrity R3): a ledger that was meant to cover an
// attestation and cannot be read leaves that attestation uncovered for a reason the
// operator can act on. A ledger of another project, or by an issuer this project does not
// pin, is valid evidence that it does not apply, and is only counted.
func (s Store) trustedLegacyEntriesRead() (map[string]string, SourceCoverage, error) {
	entries := map[string]string{}
	coverage := SourceCoverage{Losses: []SourceLoss{}}
	if _, err := s.fsys().Lstat(filepath.Join(s.Root, filepath.FromSlash(legacyLedgerDir))); errors.Is(err, os.ErrNotExist) {
		return entries, coverage, nil
	}
	// A symlinked ledger directory is refused, so a ledger outside the
	// project is never trusted.
	dir, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, legacyLedgerDir, false)
	if err != nil {
		return nil, coverage, err
	}
	files, err := s.fsys().ReadDir(dir)
	if err != nil {
		return nil, coverage, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return nil, coverage, err
	}
	project := s.ledgerProject()
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(legacyLedgerDir, f.Name()))
		lose := func(kind, detail string) {
			coverage.Add(SourceLoss{Producer: "legacy-ledger", Artifact: rel, Kind: kind, Detail: detail})
		}
		if !f.Type().IsRegular() {
			lose(LossUnsupported, "not a regular file")
			continue
		}
		raw, err := s.fsys().ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			lose(LossReadError, err.Error())
			continue
		}
		var ledger LegacyLedger
		if err := json.Unmarshal(raw, &ledger); err != nil {
			lose(LossParseError, "the ledger is not valid JSON")
			continue
		}
		if ledger.SchemaVersion != LegacyLedgerSchemaVersion || ledger.Kind != "legacy-attestations" {
			lose(LossUnsupported, "not a legacy attestations ledger of a version this engine reads")
			continue
		}
		if ledger.Project != project {
			coverage.addExclusion("legacy-ledger", "other-project", "ledgers bound to another project do not apply here", rel)
			continue
		}
		public, err := base64.StdEncoding.DecodeString(ledger.PublicKey)
		if err != nil || len(public) != ed25519.PublicKeySize {
			lose(LossInvalid, "the ledger's public key is not a valid key")
			continue
		}
		signature, err := base64.StdEncoding.DecodeString(ledger.Signature)
		if err != nil || !ed25519.Verify(ed25519.PublicKey(public), ledger.signable(), signature) {
			lose(LossInvalid, "the ledger's signature does not verify")
			continue
		}
		pin := IssuerPin(ledger.Issuer, public)
		trusted := false
		for _, candidate := range policy.TrustedAttestationIssuers {
			trusted = trusted || candidate == pin
		}
		if !trusted {
			coverage.addExclusion("legacy-ledger", "issuer-not-pinned", "ledgers by an issuer this project does not pin are not trusted", rel)
			continue
		}
		for _, entry := range ledger.Entries {
			entries[entry.AttestationID] = entry.ContentDigest
		}
	}
	coverage.Normalize()
	return entries, coverage, nil
}

// LegacyLedgerCoverage says which legacy ledgers could not be read or verified. It
// changes nothing about trust: they are not trusted either way.
func (s Store) LegacyLedgerCoverage() SourceCoverage {
	_, coverage, err := s.trustedLegacyEntriesRead()
	if err != nil {
		// The whole directory could not be used: that is the loss, not an empty one.
		coverage.Add(SourceLoss{Producer: "legacy-ledger", Artifact: legacyLedgerDir, Kind: LossReadError, Detail: err.Error()})
		coverage.Normalize()
	}
	return coverage
}

// legacyLedgerCovers reports whether a trusted ledger names att, unchanged.
func (s Store) legacyLedgerCovers(att ReviewAttestation) bool {
	entries, err := s.trustedLegacyEntries()
	if err != nil {
		return false
	}
	digest, ok := entries[att.AttestationID]
	return ok && digest == attestationContentDigest(att)
}

// UncoveredUnsignedAttestations counts unsigned attestations no trusted ledger
// covers: what adopting signed attestations would re-judge.
func (s Store) UncoveredUnsignedAttestations() (int, error) {
	unsigned, err := s.UnsignedAttestations()
	if err != nil {
		return 0, err
	}
	entries, err := s.trustedLegacyEntries()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, att := range unsigned {
		if digest, ok := entries[att.AttestationID]; !ok || digest != attestationContentDigest(att) {
			count++
		}
	}
	return count, nil
}

// AdoptionEffect is work an adoption performs besides writing policy keys.
// It is planned and shown in the preview, and applied before the policy is
// written, so `pose adopt` stays the one contract an agent sees.
type AdoptionEffect struct {
	Description string
	apply       func() error
}

// Apply performs the effect.
func (e AdoptionEffect) Apply() error { return e.apply() }

// localAttestationIssuers are the current local keys pinned for attestations.
func localAttestationIssuers(files Files, root string, docs PolicyDocs) []IssuerKeySummary {
	keys, err := ListIssuerKeys(root)
	if err != nil {
		return nil
	}
	pinned := map[string]bool{}
	for _, pin := range governedStringList(docs.Review["trusted_attestation_issuers"]) {
		pinned[pin] = true
	}
	out := []IssuerKeySummary{}
	for _, k := range keys {
		if !k.Retired && pinned[k.Pin] {
			out = append(out, k)
		}
	}
	return out
}

// signedHistoryBlocker explains why adopting signed attestations cannot seal
// the project's unsigned history, or returns "" when it can (or need not).
// With checkAmbiguity false (the catalog prerequisite, which knows no
// --issuer) only a missing issuer blocks; the choice among several is the
// planner's, which receives --issuer.
func signedHistoryBlocker(files Files, root string, docs PolicyDocs, issuer string, checkAmbiguity bool) string {
	uncovered, err := (Store{Root: root, FS: files}).UncoveredUnsignedAttestations()
	if err != nil {
		return "the attestation history cannot be inspected, so the unsigned attestations adopting would seal are not known: " + err.Error()
	}
	if uncovered == 0 {
		return ""
	}
	if issuer != "" {
		return ""
	}
	switch n := len(localAttestationIssuers(files, root, docs)); {
	case n == 0:
		return fmt.Sprintf("%d attestation(s) were recorded without a signature; adopting seals them into a ledger signed by a local issuer pinned for attestations, and none is on this machine — create one with pose issuer init <name> and pose issuer pin <name> --apply", uncovered)
	case n > 1 && checkAmbiguity:
		return fmt.Sprintf("%d attestation(s) were recorded without a signature and %d local issuers could seal them; choose one with --issuer <name>", uncovered, n)
	}
	return ""
}

// PlanAdoptionEffects returns the work adopting capability performs besides
// its policy keys. For signed-attestations in a project with history, that is
// sealing every unsigned attestation into a legacy ledger signed by the chosen
// (or only) local issuer pinned for attestations (spec
// pose-signed-legacy-attestation-ledger).
func PlanAdoptionEffects(root, capability, issuer string, now time.Time) ([]AdoptionEffect, error) {
	if capability != "signed-attestations" {
		return nil, nil
	}
	docs, err := LoadPolicyDocs(root)
	if err != nil {
		return nil, err
	}
	if blocker := signedHistoryBlocker(NativeFiles(), root, docs, issuer, true); blocker != "" {
		return nil, fmt.Errorf("pose: %s", blocker)
	}
	store := Store{Root: root}
	uncovered, err := store.UncoveredUnsignedAttestations()
	if err != nil || uncovered == 0 {
		return nil, err
	}
	name := issuer
	if name == "" {
		name = localAttestationIssuers(NativeFiles(), root, docs)[0].Issuer
	}
	key, err := LoadIssuerKey(root, name)
	if err != nil {
		return nil, err
	}
	pinned := false
	for _, pin := range governedStringList(docs.Review["trusted_attestation_issuers"]) {
		pinned = pinned || pin == key.Pin()
	}
	if !pinned {
		return nil, fmt.Errorf("pose: issuer %s is not pinned for attestations; pin it with pose issuer pin %s --apply", name, name)
	}
	ledger, err := store.PlanLegacyLedger(key, now)
	if err != nil {
		return nil, err
	}
	return []AdoptionEffect{{
		Description: fmt.Sprintf("seal %d attestation(s) recorded without a signature into a legacy ledger signed by %s; only those, unchanged, keep counting", len(ledger.Entries), key.Pin()),
		apply: func() error {
			_, err := store.SealLegacyLedger(ledger, key)
			return err
		},
	}}, nil
}
