package cli

// Signed extension catalog and lifecycle (spec pose-extension-catalog-lifecycle):
// install, update, conflict, removal and provenance for skills, workflows,
// rules and import-adapter extensions — without a hosted marketplace and
// without ever executing an installer script. An extension is data only
// (skill/workflow/rule markdown, or an import-adapter manifest); the
// lifecycle only ever reads, hashes, verifies and copies files.
//
// Package layout (a directory, not an archive — mirrors the existing
// Spec Kit/OpenSpec import adapter's "already-materialized, symlink-free
// directory" trust model instead of reinventing tar-extraction safety):
//
//	<package-dir>/
//	  extension.json         manifest (R1)
//	  files/<repo-relative>  every file the manifest lists, staged verbatim
//	<package-dir>.sigstore.json   optional Sigstore bundle (sibling file)

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const extensionSchemaVersion = 1

// extensionWhitelist bounds where any extension may ever write (architecture
// principle: extend POSE safely through repository data). Nothing outside
// these prefixes is ever installable, regardless of what a manifest claims.
var extensionWhitelist = []string{
	".agents/skills/",
	".pose/workflows/",
	".pose/rules/",
	".pose/templates/",
}

type extensionProvenance struct {
	Source string `json:"source,omitempty"` // e.g. a git remote URL the operator trusts
	Commit string `json:"commit,omitempty"`
	Signer string `json:"signer,omitempty"` // expected Sigstore certificate identity (regexp)
	Issuer string `json:"issuer,omitempty"` // expected OIDC issuer; defaults to Sigstore public good if empty
}

type extensionManifest struct {
	SchemaVersion   int                 `json:"schema_version"`
	ID              string              `json:"id"`
	Version         string              `json:"version"`
	Kind            string              `json:"kind"` // skill | workflow | rule | import-adapter
	Description     string              `json:"description"`
	PoseSchemaRange string              `json:"pose_schema_range"`
	Files           []string            `json:"files"`       // repo-relative targets
	Permissions     []string            `json:"permissions"` // repo-relative prefixes this package may write
	ConflictsWith   []string            `json:"conflicts_with,omitempty"`
	Provenance      extensionProvenance `json:"provenance"`
	Revoked         bool                `json:"revoked,omitempty"`
	RevokedReason   string              `json:"revoked_reason,omitempty"`
}

var validExtensionKinds = map[string]bool{"skill": true, "workflow": true, "rule": true, "import-adapter": true}

func loadExtensionManifest(pkgDir string) (*extensionManifest, error) {
	raw, err := os.ReadFile(filepath.Join(pkgDir, "extension.json"))
	if err != nil {
		return nil, fmt.Errorf("extension.json not found: %w", err)
	}
	var m extensionManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("extension.json: invalid JSON: %w", err)
	}
	if m.SchemaVersion != extensionSchemaVersion {
		return nil, fmt.Errorf("extension.json: unsupported schema_version %d", m.SchemaVersion)
	}
	if m.ID == "" || !slugPatternRE.MatchString(m.ID) {
		return nil, fmt.Errorf("extension.json: invalid or missing id")
	}
	if m.Version == "" {
		return nil, fmt.Errorf("extension.json: version is required")
	}
	if !validExtensionKinds[m.Kind] {
		return nil, fmt.Errorf("extension.json: invalid kind %q (use skill|workflow|rule|import-adapter)", m.Kind)
	}
	if _, _, err := parseSchemaRange(m.PoseSchemaRange); err != nil {
		return nil, fmt.Errorf("extension.json: invalid pose_schema_range: %w", err)
	}
	if len(m.Files) == 0 {
		return nil, fmt.Errorf("extension.json: files must declare at least one target")
	}
	if len(m.Permissions) == 0 {
		return nil, fmt.Errorf("extension.json: permissions must declare at least one prefix")
	}
	if m.Revoked {
		reason := m.RevokedReason
		if reason == "" {
			reason = "no reason given"
		}
		return nil, fmt.Errorf("extension %s@%s is revoked: %s", m.ID, m.Version, reason)
	}
	for _, target := range m.Files {
		clean := filepath.ToSlash(filepath.Clean(target))
		if !confinedRelativePath(clean) {
			return nil, fmt.Errorf("extension.json: file target escapes the repository: %s", target)
		}
		if !withinAnyPrefix(clean, m.Permissions) {
			return nil, fmt.Errorf("extension.json: file target %q is outside its own declared permissions", target)
		}
		if !withinAnyPrefix(clean, extensionWhitelist) {
			return nil, fmt.Errorf("extension.json: file target %q is outside the extension-manageable directories (%s)", target, strings.Join(extensionWhitelist, ", "))
		}
		if _, err := os.Stat(filepath.Join(pkgDir, "files", filepath.FromSlash(target))); err != nil {
			return nil, fmt.Errorf("extension.json: declared file missing from package: %s", target)
		}
	}
	return &m, nil
}

var slugPatternRE = depSlugRE // reuse the existing slug grammar (a-z0-9._-)

func withinAnyPrefix(target string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(target, strings.TrimSuffix(p, "/")+"/") {
			return true
		}
	}
	return false
}

// packageDigest is a deterministic, content-addressable digest over every
// declared file's path and bytes (non-functional: deterministic install
// from a digest) — independent of directory-walk order.
func packageDigest(pkgDir string, files []string) (string, error) {
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	h := sha256.New()
	for _, f := range sorted {
		b, err := os.ReadFile(filepath.Join(pkgDir, "files", filepath.FromSlash(f)))
		if err != nil {
			return "", err
		}
		fh := sha256.Sum256(b)
		fmt.Fprintf(h, "%s %s\n", f, hex.EncodeToString(fh[:]))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileDigest(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// verifyExtensionSignature is a seam: production shells out to cosign
// against the identity the manifest itself declares (an extension is
// operator-curated, not this repo's own release pipeline, so there is no
// single pinned identity — the manifest's provenance carries it). Tests
// substitute a fake to avoid depending on a real cosign binary/network.
var verifyExtensionSignature = func(pkgDir string, m *extensionManifest) error {
	bundle := pkgDir + ".sigstore.json"
	if _, err := os.Stat(bundle); err != nil {
		return fmt.Errorf("no signature bundle found at %s (unsigned extensions are rejected by default; see --allow-unsigned)", bundle)
	}
	if m.Provenance.Signer == "" {
		return fmt.Errorf("extension.json: provenance.signer is required to verify a signed extension")
	}
	issuer := m.Provenance.Issuer
	if issuer == "" {
		issuer = "https://oauth2.sigstore.dev/auth"
	}
	// The package itself is the signed blob: hash it deterministically via
	// packageDigest is NOT what was signed (that would require the signer
	// to reproduce our exact encoding); instead the convention is that the
	// publisher signs a manifest-listed "signed_blob" file — for a
	// directory package that is extension.json itself.
	cmd := exec.Command("cosign", "verify-blob",
		"--bundle", bundle,
		"--certificate-identity-regexp", m.Provenance.Signer,
		"--certificate-oidc-issuer", issuer,
		filepath.Join(pkgDir, "extension.json"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature verification failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// --- lock file ---------------------------------------------------------

type extensionLockEntry struct {
	Version           string              `json:"version"`
	Kind              string              `json:"kind"`
	InstalledAt       string              `json:"installed_at"`
	Digest            string              `json:"digest"`
	Files             map[string]string   `json:"files"` // repo-relative -> sha256 at install time
	Provenance        extensionProvenance `json:"provenance"`
	SignatureVerified bool                `json:"signature_verified"`
}

type extensionLock struct {
	SchemaVersion int                           `json:"schema_version"`
	Extensions    map[string]extensionLockEntry `json:"extensions"`
}

func extensionLockPath(root string) string {
	return filepath.Join(root, ".pose", "indexes", "extensions.lock.json")
}

func loadExtensionLock(root string) (*extensionLock, error) {
	raw, err := os.ReadFile(extensionLockPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return &extensionLock{SchemaVersion: extensionSchemaVersion, Extensions: map[string]extensionLockEntry{}}, nil
		}
		return nil, err
	}
	var lock extensionLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, fmt.Errorf("extensions.lock.json: invalid JSON: %w", err)
	}
	if lock.Extensions == nil {
		lock.Extensions = map[string]extensionLockEntry{}
	}
	return &lock, nil
}

func writeExtensionLock(root string, lock *extensionLock) error {
	raw, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(extensionLockPath(root)), 0o755); err != nil {
		return err
	}
	return writeAtomic(extensionLockPath(root), append(raw, '\n'), 0o644)
}

// --- conflict detection --------------------------------------------------

type extensionConflict struct {
	Target string
	Reason string
}

// detectConflicts finds every file target that cannot be safely written:
// owned by a different extension, or present on disk without being tracked
// by any extension (an existing repository or user file) — R2's "preserve
// user modifications".
func detectConflicts(root string, m *extensionManifest, lock *extensionLock) []extensionConflict {
	var conflicts []extensionConflict
	ownedByThis := map[string]bool{}
	if existing, ok := lock.Extensions[m.ID]; ok {
		for f := range existing.Files {
			ownedByThis[f] = true
		}
	}
	for _, target := range m.Files {
		abs := filepath.Join(root, filepath.FromSlash(target))
		info, err := os.Stat(abs)
		if err != nil {
			continue // does not exist yet — safe to create
		}
		if info.Mode()&os.ModeSymlink != 0 {
			conflicts = append(conflicts, extensionConflict{target, "existing path is a symlink"})
			continue
		}
		owner := ""
		for extID, entry := range lock.Extensions {
			if _, tracked := entry.Files[target]; tracked {
				owner = extID
				break
			}
		}
		switch {
		case owner == m.ID:
			// Our own file from a prior install — an upgrade, not a conflict.
		case owner != "":
			conflicts = append(conflicts, extensionConflict{target, fmt.Sprintf("already installed by extension %q", owner)})
		default:
			conflicts = append(conflicts, extensionConflict{target, "existing untracked file (not installed by any extension)"})
		}
	}
	for _, other := range m.ConflictsWith {
		if _, installed := lock.Extensions[other]; installed {
			conflicts = append(conflicts, extensionConflict{other, fmt.Sprintf("declared incompatible with installed extension %q", other)})
		}
	}
	return conflicts
}

// --- install / remove (transactional) ------------------------------------

// installPlan is what --dry-run prints: exactly what would happen, nothing
// more, nothing executed.
type installPlan struct {
	ID          string              `json:"id"`
	Version     string              `json:"version"`
	Kind        string              `json:"kind"`
	Digest      string              `json:"digest"`
	Files       []string            `json:"files"`
	Conflicts   []extensionConflict `json:"conflicts,omitempty"`
	SignatureOK bool                `json:"signature_verified"`
}

func planExtensionInstall(root, pkgDir string, m *extensionManifest, lock *extensionLock, allowUnsigned bool) (*installPlan, error) {
	digest, err := packageDigest(pkgDir, m.Files)
	if err != nil {
		return nil, err
	}
	sigOK := false
	if allowUnsigned {
		sigOK = false
	} else if err := verifyExtensionSignature(pkgDir, m); err != nil {
		return nil, fmt.Errorf("consent to install cannot be granted: %w", err)
	} else {
		sigOK = true
	}
	conflicts := detectConflicts(root, m, lock)
	return &installPlan{ID: m.ID, Version: m.Version, Kind: m.Kind, Digest: digest, Files: append([]string(nil), m.Files...), Conflicts: conflicts, SignatureOK: sigOK}, nil
}

// applyExtensionInstall performs the transaction: pre-images of any files
// being overwritten are captured before any write; on any failure every
// change made so far is rolled back (deleted-if-new, restored-if-overwritten)
// so the repository never sits in a partially-applied state.
func applyExtensionInstall(root, pkgDir string, m *extensionManifest, plan *installPlan) (err error) {
	type undo struct {
		path     string
		existed  bool
		preImage []byte
	}
	var applied []undo
	rollback := func() {
		for i := len(applied) - 1; i >= 0; i-- {
			u := applied[i]
			if u.existed {
				_ = os.WriteFile(u.path, u.preImage, 0o644)
			} else {
				_ = os.Remove(u.path)
			}
		}
	}
	defer func() {
		if err != nil {
			rollback()
		}
	}()
	for _, target := range m.Files {
		abs := filepath.Join(root, filepath.FromSlash(target))
		var pre []byte
		existed := false
		if b, statErr := os.ReadFile(abs); statErr == nil {
			pre, existed = b, true
		}
		if mkErr := os.MkdirAll(filepath.Dir(abs), 0o755); mkErr != nil {
			return fmt.Errorf("creating directory for %s: %w", target, mkErr)
		}
		src := filepath.Join(pkgDir, "files", filepath.FromSlash(target))
		content, readErr := os.ReadFile(src)
		if readErr != nil {
			return fmt.Errorf("reading package file %s: %w", target, readErr)
		}
		if writeErr := writeAtomic(abs, content, 0o644); writeErr != nil {
			return fmt.Errorf("writing %s: %w", target, writeErr)
		}
		applied = append(applied, undo{abs, existed, pre})
	}
	return nil
}

func cmdExtension(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose extension <install|list|remove|verify> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "install":
		return cmdExtensionInstall(root, rest, stdout, stderr)
	case "list":
		return cmdExtensionList(root, rest, stdout, stderr)
	case "remove":
		return cmdExtensionRemove(root, rest, stdout, stderr)
	case "verify":
		return cmdExtensionVerify(root, rest, stdout, stderr)
	default:
		return usageError(stderr, "Usage: pose extension <install|list|remove|verify> ...")
	}
}

func cmdExtensionVerify(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		return usageError(stderr, "Usage: pose extension verify <package-dir> [--allow-unsigned]")
	}
	pkgDir, allowUnsigned := args[0], hasFlag(args[1:], "--allow-unsigned")
	m, err := loadExtensionManifest(pkgDir)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension verify: %v\n", err)
		return 1
	}
	digest, err := packageDigest(pkgDir, m.Files)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension verify: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "id=%s version=%s kind=%s digest=%s\n", m.ID, m.Version, m.Kind, digest)
	if allowUnsigned {
		fmt.Fprintln(stdout, "signature: SKIPPED (--allow-unsigned)")
		fmt.Fprintln(stdout, "Result: SUCCESS (unsigned)")
		return 0
	}
	if err := verifyExtensionSignature(pkgDir, m); err != nil {
		fmt.Fprintf(stdout, "signature: FAILED (%v)\n", err)
		fmt.Fprintln(stdout, "Result: FAILURE")
		return 1
	}
	fmt.Fprintln(stdout, "signature: OK")
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}

func cmdExtensionInstall(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		return usageError(stderr, "Usage: pose extension install <package-dir> [--dry-run] [--yes] [--force] [--allow-unsigned]")
	}
	pkgDir := args[0]
	flags := args[1:]
	dryRun := hasFlag(flags, "--dry-run")
	yes := hasFlag(flags, "--yes")
	force := hasFlag(flags, "--force")
	allowUnsigned := hasFlag(flags, "--allow-unsigned")

	m, err := loadExtensionManifest(pkgDir)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension install: %v\n", err)
		return 1
	}
	lock, err := loadExtensionLock(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension install: %v\n", err)
		return 1
	}
	plan, err := planExtensionInstall(root, pkgDir, m, lock, allowUnsigned)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension install: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "[plan] %s@%s (%s) digest=%s\n", plan.ID, plan.Version, plan.Kind, plan.Digest)
	for _, f := range plan.Files {
		fmt.Fprintf(stdout, "  + %s\n", f)
	}
	blocking := 0
	for _, c := range plan.Conflicts {
		fmt.Fprintf(stdout, "  ! conflict: %s (%s)\n", c.Target, c.Reason)
		if !force {
			blocking++
		}
	}
	if blocking > 0 {
		fmt.Fprintln(stdout, "Result: FAILURE (conflicts present; rerun with --force to override, at your own risk)")
		return 1
	}
	if dryRun {
		fmt.Fprintln(stdout, "Result: DRY-RUN — no changes applied.")
		return 0
	}
	if !yes {
		fmt.Fprintln(stderr, "pose extension install: consent required — rerun with --yes to apply, or --dry-run to preview")
		return 2
	}
	if err := applyExtensionInstall(root, pkgDir, m, plan); err != nil {
		fmt.Fprintf(stderr, "pose extension install: %v (rolled back)\n", err)
		return 1
	}
	entry := extensionLockEntry{
		Version: m.Version, Kind: m.Kind, InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Digest: plan.Digest, Files: map[string]string{}, Provenance: m.Provenance, SignatureVerified: plan.SignatureOK,
	}
	for _, f := range m.Files {
		d, ferr := fileDigest(filepath.Join(root, filepath.FromSlash(f)))
		if ferr != nil {
			fmt.Fprintf(stderr, "pose extension install: recording lock digest for %s: %v\n", f, ferr)
			return 1
		}
		entry.Files[f] = d
	}
	lock.Extensions[m.ID] = entry
	if err := writeExtensionLock(root, lock); err != nil {
		fmt.Fprintf(stderr, "pose extension install: writing lock file: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}

func cmdExtensionList(root string, args []string, stdout, stderr io.Writer) int {
	jsonOut := hasFlag(args, "--json")
	lock, err := loadExtensionLock(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension list: %v\n", err)
		return 1
	}
	if jsonOut {
		enc := json.NewEncoder(stdout)
		return boolToExit(enc.Encode(lock) == nil)
	}
	ids := make([]string, 0, len(lock.Extensions))
	for id := range lock.Extensions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := lock.Extensions[id]
		fmt.Fprintf(stdout, "%s@%s (%s) installed=%s files=%d signed=%v\n", id, e.Version, e.Kind, e.InstalledAt, len(e.Files), e.SignatureVerified)
	}
	fmt.Fprintf(stdout, "extensions.count=%d\n", len(ids))
	return 0
}

func cmdExtensionRemove(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		return usageError(stderr, "Usage: pose extension remove <id> [--dry-run] [--yes] [--force]")
	}
	id := args[0]
	flags := args[1:]
	dryRun := hasFlag(flags, "--dry-run")
	yes := hasFlag(flags, "--yes")
	force := hasFlag(flags, "--force")

	lock, err := loadExtensionLock(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose extension remove: %v\n", err)
		return 1
	}
	entry, ok := lock.Extensions[id]
	if !ok {
		fmt.Fprintf(stderr, "pose extension remove: %q is not installed\n", id)
		return 1
	}
	modified := 0
	targets := make([]string, 0, len(entry.Files))
	for f := range entry.Files {
		targets = append(targets, f)
	}
	sort.Strings(targets)
	for _, f := range targets {
		abs := filepath.Join(root, filepath.FromSlash(f))
		d, ferr := fileDigest(abs)
		if ferr == nil && d != entry.Files[f] {
			modified++
			fmt.Fprintf(stdout, "  ! modified since install, will not remove without --force: %s\n", f)
		} else {
			fmt.Fprintf(stdout, "  - %s\n", f)
		}
	}
	if modified > 0 && !force {
		fmt.Fprintln(stdout, "Result: FAILURE (user-modified files present; rerun with --force to remove anyway)")
		return 1
	}
	if dryRun {
		fmt.Fprintln(stdout, "Result: DRY-RUN — no changes applied.")
		return 0
	}
	if !yes {
		fmt.Fprintln(stderr, "pose extension remove: consent required — rerun with --yes to apply, or --dry-run to preview")
		return 2
	}
	for _, f := range targets {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(f)))
	}
	delete(lock.Extensions, id)
	if err := writeExtensionLock(root, lock); err != nil {
		fmt.Fprintf(stderr, "pose extension remove: writing lock file: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func boolToExit(ok bool) int {
	if ok {
		return 0
	}
	return 1
}
