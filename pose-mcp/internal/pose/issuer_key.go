package pose

// Native attestation issuers (spec pose-native-attestation-issuer).
//
// The engine already verified Ed25519 envelopes from issuers pinned in review
// policy, but only an external emitter (the Harne8 Conductor) could produce
// them, so a project with POSE alone could never require signed attestations.
// A native issuer is the same thing the Conductor is to the verifier: a name
// and an Ed25519 key, pinned as `<name>#sha256:<public-key-digest>`. Nothing
// about verification changes, which is what lets the two coexist: any pinned
// issuer's envelope counts, and pinning one never removes another.
//
// The private key never enters the project. It lives under the user's config
// directory with owner-only permissions, and no function here returns or
// prints it; callers get the public key, the pin and signatures.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// IssuerKeySchemaVersion versions the key file.
const IssuerKeySchemaVersion = 1

var issuerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// IssuerKey is a native issuer's key pair. Its private half is unexported and
// never serialized outside the key file.
type IssuerKey struct {
	Name      string
	Path      string
	CreatedAt string
	public    ed25519.PublicKey
	private   ed25519.PrivateKey
}

// PublicKey returns the base64 public key, as envelopes carry it.
func (k IssuerKey) PublicKey() string { return base64.StdEncoding.EncodeToString(k.public) }

// Pin is what review policy lists to trust this issuer.
func (k IssuerKey) Pin() string { return IssuerPin(k.Name, k.public) }

// IssuerPin computes `<name>#sha256:<digest>` for a public key.
func IssuerPin(name string, public ed25519.PublicKey) string {
	return name + "#" + digestBytes(public)
}

type issuerKeyFile struct {
	SchemaVersion int    `json:"schema_version"`
	Issuer        string `json:"issuer"`
	Algorithm     string `json:"algorithm"`
	CreatedAt     string `json:"created_at"`
	Seed          string `json:"seed"`
}

// IssuerKeyDir is where native issuer keys live: $POSE_ISSUER_HOME, or
// <user config dir>/pose/issuers.
func IssuerKeyDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("POSE_ISSUER_HOME")); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("pose: no user config directory for issuer keys (set POSE_ISSUER_HOME): %w", err)
	}
	return filepath.Join(base, "pose", "issuers"), nil
}

// ValidIssuerName reports whether name can be a pinned issuer: lowercase,
// no `#` (the pin separator), no path separators.
func ValidIssuerName(name string) error {
	if !issuerNamePattern.MatchString(name) {
		return fmt.Errorf("pose: issuer name %q must be lowercase letters, digits, '.', '_' or '-' (at most 64)", name)
	}
	return nil
}

func issuerKeyPath(dir, name string) string { return filepath.Join(dir, name+".key") }

// issuerKeyDirOutside returns the key directory after checking it is not
// inside the project at root, symlinks resolved: a key under the project is a
// key one `git add -A` away from being published. An empty root skips the
// check, for callers with no project.
func issuerKeyDirOutside(root string) (string, error) {
	dir, err := IssuerKeyDir()
	if err != nil {
		return "", err
	}
	if root == "" {
		return dir, nil
	}
	realDir, err := resolveExisting(dir)
	if err != nil {
		return "", err
	}
	realRoot, err := resolveExisting(root)
	if err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(realRoot, realDir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("pose: issuer keys must live outside the project, and %s is inside %s; point POSE_ISSUER_HOME elsewhere", dir, root)
	}
	return dir, nil
}

// resolveExisting resolves symlinks in the longest existing prefix of path
// and appends the rest, so a directory that does not exist yet is judged by
// where it would be created.
func resolveExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rest := ""
	for current := abs; ; {
		if real, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(real, rest), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs, nil
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
	}
}

// CreateIssuerKey creates a new key for name. It refuses to replace an
// existing key: losing a key would orphan its pin, and overwriting it would
// silently change what the pin trusts.
func CreateIssuerKey(root, name string, now time.Time) (IssuerKey, error) {
	if err := ValidIssuerName(name); err != nil {
		return IssuerKey{}, err
	}
	dir, err := issuerKeyDirOutside(root)
	if err != nil {
		return IssuerKey{}, err
	}
	if err := ensurePrivateDir(dir); err != nil {
		return IssuerKey{}, err
	}
	return writeIssuerKey(name, issuerKeyPath(dir, name), now)
}

func writeIssuerKey(name, path string, now time.Time) (IssuerKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return IssuerKey{}, err
	}
	created := now.UTC().Truncate(time.Second).Format(time.RFC3339)
	raw, err := json.MarshalIndent(issuerKeyFile{SchemaVersion: IssuerKeySchemaVersion, Issuer: name, Algorithm: "ed25519", CreatedAt: created, Seed: base64.StdEncoding.EncodeToString(private.Seed())}, "", "  ")
	if err != nil {
		return IssuerKey{}, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return IssuerKey{}, fmt.Errorf("pose: issuer %s already has a key at %s; use pose issuer rotate to replace it", name, path)
	}
	if err != nil {
		return IssuerKey{}, err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		file.Close()
		os.Remove(path)
		return IssuerKey{}, err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return IssuerKey{}, err
	}
	return IssuerKey{Name: name, Path: path, CreatedAt: created, public: public, private: private}, nil
}

// RotateIssuerKey retires the current key of name (kept beside it, so the old
// pin can still be inspected) and creates a new one. Pins are policy and are
// not touched here: the caller pins the new key next to the old one.
func RotateIssuerKey(root, name string, now time.Time) (retired, current IssuerKey, err error) {
	retired, err = LoadIssuerKey(root, name)
	if err != nil {
		return IssuerKey{}, IssuerKey{}, err
	}
	dir := filepath.Dir(retired.Path)
	// A hard link fails instead of replacing an existing file, so a second
	// rotation in the same second can never overwrite an earlier retired key.
	stamp := now.UTC().Format("20060102T150405Z")
	archive := ""
	for i := 0; archive == ""; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s.retired-%s.key", name, stamp))
		if i > 0 {
			candidate = filepath.Join(dir, fmt.Sprintf("%s.retired-%s-%d.key", name, stamp, i))
		}
		switch err := os.Link(retired.Path, candidate); {
		case err == nil:
			archive = candidate
		case errors.Is(err, os.ErrExist) && i < 1000:
		default:
			return IssuerKey{}, IssuerKey{}, err
		}
	}
	if err := os.Remove(retired.Path); err != nil {
		os.Remove(archive)
		return IssuerKey{}, IssuerKey{}, err
	}
	retired.Path = archive
	current, err = writeIssuerKey(name, issuerKeyPath(dir, name), now)
	if err != nil {
		// Put the old key back so the issuer is never left without one.
		_ = os.Rename(archive, issuerKeyPath(dir, name))
		return IssuerKey{}, IssuerKey{}, err
	}
	return retired, current, nil
}

// LoadIssuerKey reads name's current key, refusing a key file or directory
// that other users can read or that lies inside the project at root.
func LoadIssuerKey(root, name string) (IssuerKey, error) {
	if err := ValidIssuerName(name); err != nil {
		return IssuerKey{}, err
	}
	dir, err := issuerKeyDirOutside(root)
	if err != nil {
		return IssuerKey{}, err
	}
	return loadIssuerKeyFile(name, issuerKeyPath(dir, name))
}

func loadIssuerKeyFile(name, path string) (IssuerKey, error) {
	if err := requirePrivate(filepath.Dir(path), true); err != nil {
		return IssuerKey{}, err
	}
	if err := requirePrivate(path, false); err != nil {
		return IssuerKey{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return IssuerKey{}, fmt.Errorf("pose: no key for issuer %s at %s; create one with pose issuer init %s", name, path, name)
	}
	if err != nil {
		return IssuerKey{}, err
	}
	var doc issuerKeyFile
	if err := json.Unmarshal(raw, &doc); err != nil || doc.SchemaVersion != IssuerKeySchemaVersion || doc.Algorithm != "ed25519" {
		return IssuerKey{}, fmt.Errorf("pose: %s is not a POSE issuer key", path)
	}
	if doc.Issuer != name {
		return IssuerKey{}, fmt.Errorf("pose: %s belongs to issuer %s, not %s", path, doc.Issuer, name)
	}
	seed, err := base64.StdEncoding.DecodeString(doc.Seed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return IssuerKey{}, fmt.Errorf("pose: %s holds a malformed key", path)
	}
	private := ed25519.NewKeyFromSeed(seed)
	return IssuerKey{Name: name, Path: path, CreatedAt: doc.CreatedAt, public: private.Public().(ed25519.PublicKey), private: private}, nil
}

// IssuerKeySummary describes a key without its private half.
type IssuerKeySummary struct {
	Issuer    string `json:"issuer"`
	Pin       string `json:"pin"`
	PublicKey string `json:"public_key"`
	CreatedAt string `json:"created_at"`
	Retired   bool   `json:"retired"`
	Path      string `json:"path"`
}

// ListIssuerKeys lists the native issuer keys on this machine, current and
// retired. Unreadable or foreign files are skipped, not reported as keys.
func ListIssuerKeys(root string) ([]IssuerKeySummary, error) {
	dir, err := issuerKeyDirOutside(root)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []IssuerKeySummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []IssuerKeySummary{}
	for _, entry := range entries {
		file := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(file, ".key") {
			continue
		}
		stem := strings.TrimSuffix(file, ".key")
		name, retired := stem, false
		if i := strings.Index(stem, ".retired-"); i > 0 {
			name, retired = stem[:i], true
		}
		key, err := loadIssuerKeyFile(name, filepath.Join(dir, file))
		if err != nil {
			continue
		}
		out = append(out, IssuerKeySummary{Issuer: name, Pin: key.Pin(), PublicKey: key.PublicKey(), CreatedAt: key.CreatedAt, Retired: retired, Path: key.Path})
	}
	slices.SortFunc(out, func(a, b IssuerKeySummary) int {
		if c := strings.Compare(a.Issuer, b.Issuer); c != 0 {
			return c
		}
		return strings.Compare(a.CreatedAt, b.CreatedAt)
	})
	return out, nil
}

func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return requirePrivate(dir, true)
}

// requirePrivate refuses a path group or others can access. A key readable by
// another account is a key that account may have copied.
func requirePrivate(path string, isDir bool) error {
	stat := os.Stat
	if !isDir {
		// A key file is read where it is, never through a link: a symlink or
		// a second hard link can place the key inside a project even when the
		// key directory is outside it.
		stat = os.Lstat
	}
	info, err := stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isDir && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("pose: %s is a symlink; an issuer key must be a regular file in the key directory", path)
	}
	if !isDir && extraHardLinks(info) {
		return fmt.Errorf("pose: %s has another hard link, so a copy of the key may live elsewhere; keep a single link", path)
	}
	if isDir != info.IsDir() {
		return fmt.Errorf("pose: %s is not a %s", path, map[bool]string{true: "directory", false: "file"}[isDir])
	}
	if info.Mode().Perm()&0o077 != 0 {
		want := "0600"
		if isDir {
			want = "0700"
		}
		return fmt.Errorf("pose: %s is accessible to other users (mode %04o); restrict it to %s", path, info.Mode().Perm(), want)
	}
	return nil
}

// IssuerPinPlan is the policy change that trusts a native issuer.
type IssuerPinPlan struct {
	Pin            string   `json:"pin"`
	Attestations   bool     `json:"attestations"`
	HumanAuthority bool     `json:"human_authority"`
	Changes        []string `json:"changes"`
}

// PlanIssuerPin adds pin to the requested policy lists of docs, keeping every
// pin already there, and sets authority_audience and authority_project when
// they are absent and the caller supplied them. It never replaces a different
// audience or project: those name what every issuer's claims are addressed to.
func PlanIssuerPin(docs PolicyDocs, pin string, attestations, humanAuthority bool, audience, project string) (IssuerPinPlan, error) {
	plan := IssuerPinPlan{Pin: pin, Attestations: attestations, HumanAuthority: humanAuthority}
	if !attestations && !humanAuthority {
		return plan, fmt.Errorf("pose: choose --attestations, --human-authority or both")
	}
	if _, _, ok := strings.Cut(pin, "#sha256:"); !ok || !validHumanAuthorityIssuerPin(pin) {
		return plan, fmt.Errorf("pose: %q is not an issuer pin (<issuer>#sha256:<64 hex>)", pin)
	}
	add := func(key string) {
		list := governedStringList(docs.Review[key])
		if slices.Contains(list, pin) {
			return
		}
		docs.Review[key] = toAnyList(append(list, pin))
		plan.Changes = append(plan.Changes, "add "+key+" "+pin)
	}
	if attestations {
		add("trusted_attestation_issuers")
	}
	if humanAuthority {
		add("human_authority_issuers")
	}
	setIfAbsent := func(key, value string) error {
		if value == "" {
			return nil
		}
		current, _ := docs.Review[key].(string)
		switch {
		case current == "":
			docs.Review[key] = value
			plan.Changes = append(plan.Changes, "set "+key+"="+value)
		case current != value:
			return fmt.Errorf("pose: %s is already %q; every issuer's claims answer to it, so pinning an issuer does not change it", key, current)
		}
		return nil
	}
	if err := setIfAbsent("authority_audience", audience); err != nil {
		return plan, err
	}
	if err := setIfAbsent("authority_project", project); err != nil {
		return plan, err
	}
	return plan, nil
}

// NativeAuthority is what a native issuer asserts about the reviewer.
type NativeAuthority struct {
	ReviewExecution         string
	ImplementationPrincipal string
	ImplementationExecution string
	TTL                     time.Duration
}

// SignReviewAttestation completes att for its sealed bundle, optionally binds
// a reviewer authority claim issued by key, signs the canonical bytes and
// returns the envelope the verifier accepts. It records nothing.
func (s Store) SignReviewAttestation(att ReviewAttestation, key IssuerKey, authority *NativeAuthority, now time.Time) (ReviewAttestationEnvelope, error) {
	if key.private == nil {
		return ReviewAttestationEnvelope{}, fmt.Errorf("pose: issuer %s has no loaded key", key.Name)
	}
	if authority != nil {
		bundle, err := s.LoadReviewBundle(att.BundleID)
		if err != nil {
			return ReviewAttestationEnvelope{}, err
		}
		policy, _, err := s.loadReviewPolicy()
		if err != nil {
			return ReviewAttestationEnvelope{}, err
		}
		role := ""
		switch {
		case strings.HasPrefix(att.Reviewer, "human:"):
			role = "human"
		case strings.HasPrefix(att.Reviewer, "agent:"):
			role = "agent"
		default:
			return ReviewAttestationEnvelope{}, fmt.Errorf("pose: an authority claim needs a human: or agent: reviewer, got %q", att.Reviewer)
		}
		issued := now.UTC().Truncate(time.Second)
		claim := &ReviewAuthorityClaim{
			SchemaVersion: ReviewBundleSchemaVersion, Project: policy.AuthorityProject, BundleDigest: bundle.BundleDigest,
			Principal: att.Reviewer, Role: role, ReviewExecution: authority.ReviewExecution,
			ImplementationPrincipal: authority.ImplementationPrincipal, ImplementationExecution: authority.ImplementationExecution,
			Issuer: key.Name, IssuedAt: issued.Format(time.RFC3339), Audience: policy.AuthorityAudience,
		}
		if authority.TTL > 0 {
			claim.ExpiresAt = issued.Add(authority.TTL).Format(time.RFC3339)
		}
		att.Authority = claim
	}
	completed, canonical, err := s.SignableReviewAttestation(att, now)
	if err != nil {
		return ReviewAttestationEnvelope{}, err
	}
	signature := ed25519.Sign(key.private, canonical)
	return ReviewAttestationEnvelope{
		SchemaVersion: ReviewBundleSchemaVersion, Issuer: key.Name, Subject: completed.BundleID, Algorithm: "ed25519",
		PublicKey: key.PublicKey(), Signature: base64.StdEncoding.EncodeToString(signature), Attestation: completed,
	}, nil
}
