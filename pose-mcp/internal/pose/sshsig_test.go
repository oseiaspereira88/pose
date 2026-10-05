package pose

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-signed-action-answers.

// sshTestKey signs like ssh-keygen -Y sign, and like a FIDO authenticator for
// a security key, so the verifier is exercised without hardware.
type sshTestKey struct {
	private ed25519.PrivateKey
	sk      bool
	app     string
}

func newSSHTestKey(t *testing.T, sk bool) sshTestKey {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return sshTestKey{private: private, sk: sk, app: "ssh:"}
}

func (k sshTestKey) keyType() string {
	if k.sk {
		return SSHKeySKEd25519
	}
	return SSHKeyEd25519
}

func (k sshTestKey) blob() []byte {
	b := append(sshString([]byte(k.keyType())), sshString(k.private.Public().(ed25519.PublicKey))...)
	if k.sk {
		b = append(b, sshString([]byte(k.app))...)
	}
	return b
}

func (k sshTestKey) line(comment string) string {
	return strings.TrimSpace(k.keyType() + " " + base64.StdEncoding.EncodeToString(k.blob()) + " " + comment)
}

func (k sshTestKey) sign(namespace string, message []byte, flags byte) string {
	sum := sha512.Sum512(message)
	signed := append([]byte(sshsigMagic), sshString([]byte(namespace))...)
	signed = append(signed, sshString(nil)...)
	signed = append(signed, sshString([]byte("sha512"))...)
	signed = append(signed, sshString(sum[:])...)
	var sigBlob []byte
	if k.sk {
		counter := make([]byte, 4)
		binary.BigEndian.PutUint32(counter, 7)
		app := sha256.Sum256([]byte(k.app))
		msg := sha256.Sum256(signed)
		inner := append(append(append(append([]byte{}, app[:]...), flags), counter...), msg[:]...)
		sigBlob = append(sshString([]byte(k.keyType())), sshString(ed25519.Sign(k.private, inner))...)
		sigBlob = append(append(sigBlob, flags), counter...)
	} else {
		sigBlob = append(sshString([]byte(k.keyType())), sshString(ed25519.Sign(k.private, signed))...)
	}
	version := make([]byte, 4)
	binary.BigEndian.PutUint32(version, 1)
	blob := append([]byte(sshsigMagic), version...)
	blob = append(blob, sshString(k.blob())...)
	blob = append(blob, sshString([]byte(namespace))...)
	blob = append(blob, sshString(nil)...)
	blob = append(blob, sshString([]byte("sha512"))...)
	blob = append(blob, sshString(sigBlob)...)
	encoded := base64.StdEncoding.EncodeToString(blob)
	var out strings.Builder
	out.WriteString(sshsigArmorBegin + "\n")
	for len(encoded) > 70 {
		out.WriteString(encoded[:70] + "\n")
		encoded = encoded[70:]
	}
	out.WriteString(encoded + "\n" + sshsigArmorEnd + "\n")
	return out.String()
}

func TestSSHSigVerifiesPlainAndSecurityKeys(t *testing.T) {
	message := []byte("statement\n")
	plain := newSSHTestKey(t, false)
	check, err := VerifySSHSignature(plain.sign(ActionAnswerNamespace, message, 0), ActionAnswerNamespace, message)
	if err != nil || check.Key.Type != SSHKeyEd25519 || check.UserPresent {
		t.Fatalf("plain key: %+v %v", check, err)
	}
	sk := newSSHTestKey(t, true)
	check, err = VerifySSHSignature(sk.sign(ActionAnswerNamespace, message, sshSKUserPresent|sshSKUserVerified), ActionAnswerNamespace, message)
	if err != nil || !check.UserPresent || !check.UserVerified || check.Counter != 7 || check.Key.Application != "ssh:" {
		t.Fatalf("security key: %+v %v", check, err)
	}
	check, err = VerifySSHSignature(sk.sign(ActionAnswerNamespace, message, 0), ActionAnswerNamespace, message)
	if err != nil || check.UserPresent {
		t.Fatalf("a security-key signature without a touch must verify and say so: %+v %v", check, err)
	}
	parsed, err := ParseAuthorizedKey(sk.line("me@laptop"))
	if err != nil || parsed.Fingerprint() != check.Key.Fingerprint() || parsed.Comment != "me@laptop" || !parsed.ProvesPresence() {
		t.Fatalf("authorized key: %+v %v", parsed, err)
	}
}

func TestSSHSigRefusesWhatOpenSSHWouldRefuse(t *testing.T) {
	message := []byte("statement\n")
	key := newSSHTestKey(t, false)
	good := key.sign(ActionAnswerNamespace, message, 0)
	cases := map[string]func() (string, string, []byte){
		"tampered message":  func() (string, string, []byte) { return good, ActionAnswerNamespace, []byte("statement!\n") },
		"another namespace": func() (string, string, []byte) { return key.sign("git", message, 0), ActionAnswerNamespace, message },
		"not armored":       func() (string, string, []byte) { return "hello", ActionAnswerNamespace, message },
		"truncated": func() (string, string, []byte) {
			blob, _ := decodeSSHSigArmor(good)
			return sshsigArmorBegin + "\n" + base64.StdEncoding.EncodeToString(blob[:len(blob)-9]) + "\n" + sshsigArmorEnd, ActionAnswerNamespace, message
		},
		"flipped signature byte": func() (string, string, []byte) {
			blob, _ := decodeSSHSigArmor(good)
			blob[len(blob)-1] ^= 0xff
			return sshsigArmorBegin + "\n" + base64.StdEncoding.EncodeToString(blob) + "\n" + sshsigArmorEnd, ActionAnswerNamespace, message
		},
	}
	for name, build := range cases {
		armored, namespace, msg := build()
		if _, err := VerifySSHSignature(armored, namespace, msg); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := ParseAuthorizedKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ== x"); err == nil {
		t.Error("an RSA key was accepted")
	}
}

// The verifier agrees with OpenSSH in both directions when ssh-keygen is
// installed: it accepts what ssh-keygen signs, and ssh-keygen accepts what
// the test signer produces, so the test signer is not a private dialect.
func TestSSHSigInteroperatesWithSSHKeygen(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not installed")
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if out, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-C", "interop", "-f", keyPath).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v %s", err, out)
	}
	message := []byte(`{"schema_version":1}` + "\n")
	cmd := exec.Command(keygen, "-Y", "sign", "-f", keyPath, "-n", ActionAnswerNamespace)
	cmd.Stdin = bytes.NewReader(message)
	signature, err := cmd.Output()
	if err != nil {
		t.Fatalf("ssh-keygen -Y sign: %v", err)
	}
	check, err := VerifySSHSignature(string(signature), ActionAnswerNamespace, message)
	if err != nil {
		t.Fatalf("a signature made by ssh-keygen was refused: %v", err)
	}
	public, _ := os.ReadFile(keyPath + ".pub")
	parsed, err := ParseAuthorizedKey(string(public))
	if err != nil || parsed.Fingerprint() != check.Key.Fingerprint() {
		t.Fatalf("the signing key is not the generated one: %v", err)
	}
	fingerprint, _ := exec.Command(keygen, "-l", "-E", "sha256", "-f", keyPath+".pub").Output()
	if !strings.Contains(string(fingerprint), parsed.Fingerprint()) {
		t.Fatalf("fingerprint %s disagrees with ssh-keygen: %s", parsed.Fingerprint(), fingerprint)
	}

	ours := newSSHTestKey(t, false)
	sigPath := filepath.Join(dir, "ours.sig")
	if err := os.WriteFile(sigPath, []byte(ours.sign(ActionAnswerNamespace, message, 0)), 0o644); err != nil {
		t.Fatal(err)
	}
	check2 := exec.Command(keygen, "-Y", "check-novalidate", "-n", ActionAnswerNamespace, "-s", sigPath)
	check2.Stdin = bytes.NewReader(message)
	if out, err := check2.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen refuses the test signer: %v %s", err, out)
	}
}
