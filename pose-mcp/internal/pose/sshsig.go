package pose

// A native verifier for OpenSSH signatures (PROTOCOL.sshsig), so a project
// can check who answered without an external service or a new dependency
// (spec pose-signed-action-answers). Only Ed25519 keys are accepted: plain
// `ssh-ed25519` and the FIDO security-key form `sk-ssh-ed25519@openssh.com`,
// whose signature also carries the flags the authenticator asserted.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	SSHKeyEd25519   = "ssh-ed25519"
	SSHKeySKEd25519 = "sk-ssh-ed25519@openssh.com"

	sshsigMagic      = "SSHSIG"
	sshsigArmorBegin = "-----BEGIN SSH SIGNATURE-----"
	sshsigArmorEnd   = "-----END SSH SIGNATURE-----"

	// Authenticator flags (PROTOCOL.u2f).
	sshSKUserPresent  = 0x01
	sshSKUserVerified = 0x04
)

// SSHPublicKey is a parsed Ed25519 OpenSSH public key.
type SSHPublicKey struct {
	Type        string
	Key         ed25519.PublicKey
	Application string // security keys only, e.g. "ssh:"
	Blob        []byte
	Comment     string
}

// Fingerprint is the OpenSSH SHA256 fingerprint of the key.
func (k SSHPublicKey) Fingerprint() string {
	sum := sha256.Sum256(k.Blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

// AuthorizedKey renders the key in authorized_keys form, without comment.
func (k SSHPublicKey) AuthorizedKey() string {
	return k.Type + " " + base64.StdEncoding.EncodeToString(k.Blob)
}

// ProvesPresence is true for a security key: every signature it makes
// carries the authenticator's flags, so a verifier can require a touch.
func (k SSHPublicKey) ProvesPresence() bool { return k.Type == SSHKeySKEd25519 }

// SSHSignatureCheck is what a valid signature established.
type SSHSignatureCheck struct {
	Key           SSHPublicKey
	UserPresent   bool
	UserVerified  bool
	Counter       uint32
	HashAlgorithm string
}

type sshReader struct{ b []byte }

func (r *sshReader) bytes() ([]byte, error) {
	if len(r.b) < 4 {
		return nil, errors.New("truncated length")
	}
	n := binary.BigEndian.Uint32(r.b)
	if uint64(n) > uint64(len(r.b)-4) {
		return nil, errors.New("truncated field")
	}
	v := r.b[4 : 4+n]
	r.b = r.b[4+n:]
	return v, nil
}

func (r *sshReader) uint32() (uint32, error) {
	if len(r.b) < 4 {
		return 0, errors.New("truncated integer")
	}
	v := binary.BigEndian.Uint32(r.b)
	r.b = r.b[4:]
	return v, nil
}

func (r *sshReader) byte() (byte, error) {
	if len(r.b) < 1 {
		return 0, errors.New("truncated byte")
	}
	v := r.b[0]
	r.b = r.b[1:]
	return v, nil
}

func sshString(v []byte) []byte {
	out := make([]byte, 4+len(v))
	binary.BigEndian.PutUint32(out, uint32(len(v)))
	copy(out[4:], v)
	return out
}

// ParseSSHPublicKeyBlob parses the wire form of an Ed25519 public key.
func ParseSSHPublicKeyBlob(blob []byte) (SSHPublicKey, error) {
	r := &sshReader{b: blob}
	keyType, err := r.bytes()
	if err != nil {
		return SSHPublicKey{}, fmt.Errorf("public key: %v", err)
	}
	key := SSHPublicKey{Type: string(keyType), Blob: append([]byte(nil), blob...)}
	switch key.Type {
	case SSHKeyEd25519, SSHKeySKEd25519:
	default:
		return SSHPublicKey{}, fmt.Errorf("public key type %q is not supported; use ssh-ed25519 or sk-ssh-ed25519@openssh.com", key.Type)
	}
	raw, err := r.bytes()
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return SSHPublicKey{}, errors.New("public key: invalid Ed25519 key")
	}
	key.Key = ed25519.PublicKey(append([]byte(nil), raw...))
	if key.Type == SSHKeySKEd25519 {
		application, err := r.bytes()
		if err != nil {
			return SSHPublicKey{}, fmt.Errorf("public key: %v", err)
		}
		key.Application = string(application)
	}
	if len(r.b) != 0 {
		return SSHPublicKey{}, errors.New("public key: trailing data")
	}
	return key, nil
}

// ParseAuthorizedKey parses one authorized_keys line: `<type> <base64>
// [comment]`. Options before the type are not accepted.
func ParseAuthorizedKey(line string) (SSHPublicKey, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return SSHPublicKey{}, errors.New("a public key is `<type> <base64> [comment]`")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return SSHPublicKey{}, errors.New("public key: invalid base64")
	}
	key, err := ParseSSHPublicKeyBlob(blob)
	if err != nil {
		return SSHPublicKey{}, err
	}
	if key.Type != fields[0] {
		return SSHPublicKey{}, fmt.Errorf("public key: the line says %s and the key is %s", fields[0], key.Type)
	}
	key.Comment = strings.Join(fields[2:], " ")
	return key, nil
}

// decodeSSHSigArmor returns the binary blob of an armored signature.
func decodeSSHSigArmor(armored string) ([]byte, error) {
	text := strings.TrimSpace(armored)
	if !strings.HasPrefix(text, sshsigArmorBegin) || !strings.HasSuffix(text, sshsigArmorEnd) {
		return nil, errors.New("not an armored SSH signature")
	}
	body := strings.TrimSuffix(strings.TrimPrefix(text, sshsigArmorBegin), sshsigArmorEnd)
	body = strings.Join(strings.Fields(body), "")
	blob, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, errors.New("signature: invalid base64")
	}
	return blob, nil
}

// VerifySSHSignature checks an armored SSHSIG over message in namespace and
// returns the key that made it. It does not decide whether that key is
// trusted: the caller compares it with the keys it registered.
func VerifySSHSignature(armored string, namespace string, message []byte) (SSHSignatureCheck, error) {
	fail := func(format string, a ...any) (SSHSignatureCheck, error) {
		return SSHSignatureCheck{}, fmt.Errorf("signature: "+format, a...)
	}
	blob, err := decodeSSHSigArmor(armored)
	if err != nil {
		return SSHSignatureCheck{}, err
	}
	if !bytes.HasPrefix(blob, []byte(sshsigMagic)) {
		return fail("missing SSHSIG preamble")
	}
	r := &sshReader{b: blob[len(sshsigMagic):]}
	version, err := r.uint32()
	if err != nil || version != 1 {
		return fail("unsupported version")
	}
	publicBlob, err := r.bytes()
	if err != nil {
		return fail("%v", err)
	}
	key, err := ParseSSHPublicKeyBlob(publicBlob)
	if err != nil {
		return SSHSignatureCheck{}, err
	}
	signedNamespace, err := r.bytes()
	if err != nil {
		return fail("%v", err)
	}
	if string(signedNamespace) != namespace {
		return fail("made for namespace %q, not %q", signedNamespace, namespace)
	}
	reserved, err := r.bytes()
	if err != nil {
		return fail("%v", err)
	}
	hashAlgorithm, err := r.bytes()
	if err != nil {
		return fail("%v", err)
	}
	signatureBlob, err := r.bytes()
	if err != nil {
		return fail("%v", err)
	}
	if len(r.b) != 0 {
		return fail("trailing data")
	}
	var digest []byte
	switch string(hashAlgorithm) {
	case "sha512":
		sum := sha512.Sum512(message)
		digest = sum[:]
	case "sha256":
		sum := sha256.Sum256(message)
		digest = sum[:]
	default:
		return fail("hash algorithm %q is not supported", hashAlgorithm)
	}
	signed := append([]byte(sshsigMagic), sshString(signedNamespace)...)
	signed = append(signed, sshString(reserved)...)
	signed = append(signed, sshString(hashAlgorithm)...)
	signed = append(signed, sshString(digest)...)

	sr := &sshReader{b: signatureBlob}
	format, err := sr.bytes()
	if err != nil {
		return fail("%v", err)
	}
	if string(format) != key.Type {
		return fail("a %s signature cannot come from a %s key", format, key.Type)
	}
	raw, err := sr.bytes()
	if err != nil || len(raw) != ed25519.SignatureSize {
		return fail("invalid Ed25519 signature")
	}
	check := SSHSignatureCheck{Key: key, HashAlgorithm: string(hashAlgorithm)}
	if key.Type == SSHKeySKEd25519 {
		flags, err := sr.byte()
		if err != nil {
			return fail("%v", err)
		}
		counter, err := sr.uint32()
		if err != nil {
			return fail("%v", err)
		}
		applicationSum := sha256.Sum256([]byte(key.Application))
		messageSum := sha256.Sum256(signed)
		counterBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(counterBytes, counter)
		signed = append(append(append(append([]byte{}, applicationSum[:]...), flags), counterBytes...), messageSum[:]...)
		check.UserPresent = flags&sshSKUserPresent != 0
		check.UserVerified = flags&sshSKUserVerified != 0
		check.Counter = counter
	}
	if len(sr.b) != 0 {
		return fail("trailing data in the signature")
	}
	if !ed25519.Verify(key.Key, signed, raw) {
		return fail("does not verify for this message and key")
	}
	return check, nil
}
