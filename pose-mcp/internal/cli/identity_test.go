package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-signed-action-answers, through Main.

func identityFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.storage-test")
	root := newGitRepo(t)
	if out, err := exec.Command("git", "-C", root, "config", "user.email", "Ada.Lovelace@example.com").CombinedOutput(); err != nil {
		t.Fatalf("git config: %v %s", err, out)
	}
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-storage.md"), "---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n## 2. Requirements\n\n- R4: Keep reading schema v1 records.\n")
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"_comment":"kept","schema_version":1,"roles":{}}`)
	return root
}

func sshKeygenOrSkip(t *testing.T) string {
	t.Helper()
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not installed")
	}
	return keygen
}

func newSSHKeyFile(t *testing.T, keygen string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-C", "ada@laptop", "-f", path).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v %s", err, out)
	}
	return path
}

func TestIdentityAddSuggestsTheGitNameAndRegistersTheKey(t *testing.T) {
	keygen := sshKeygenOrSkip(t)
	root := identityFixture(t)
	key := newSSHKeyFile(t, keygen)

	code, out := runPose(t, root, "identity", "add", "--key", key+".pub", "--role", "maintainer")
	if code != 0 || !strings.Contains(out, "identity.suggested=human:ada.lovelace (from git user.email; a name, not a proof") ||
		!strings.Contains(out, "does not prove presence") || !strings.Contains(out, "identity.apply=false") {
		t.Fatalf("preview: %d %s", code, out)
	}
	if raw, _ := os.ReadFile(filepath.Join(root, ".pose/policy/actions.json")); strings.Contains(string(raw), "keys") {
		t.Fatal("a preview wrote the policy")
	}
	if code, out = runPose(t, root, "identity", "add", "--key", key, "--apply"); code == 0 || !strings.Contains(out, "private key") {
		t.Fatalf("a private key file was accepted: %d %s", code, out)
	}
	if code, out = runPose(t, root, "identity", "add", "--key", key+".pub", "--role", "maintainer", "--apply"); code != 0 {
		t.Fatalf("apply: %s", out)
	}
	if code, out = runPose(t, root, "identity", "add", "human:mallory", "--key", key+".pub", "--apply"); code == 0 || !strings.Contains(out, "already registered to human:ada.lovelace") {
		t.Fatalf("a key was registered to a second principal: %d %s", code, out)
	}
	_, listed := runPose(t, root, "identity", "list", "--json")
	var doc struct {
		Principals []struct {
			Principal string   `json:"principal"`
			Roles     []string `json:"roles"`
			Keys      []struct {
				Fingerprint string `json:"fingerprint"`
				Comment     string `json:"comment"`
			} `json:"keys"`
		} `json:"principals"`
	}
	if err := json.Unmarshal([]byte(listed), &doc); err != nil || len(doc.Principals) != 1 || doc.Principals[0].Principal != "human:ada.lovelace" ||
		len(doc.Principals[0].Roles) != 1 || len(doc.Principals[0].Keys) != 1 || doc.Principals[0].Keys[0].Comment != "ada@laptop" {
		t.Fatalf("list: %v %s", err, listed)
	}
	fingerprint, _ := exec.Command(keygen, "-l", "-E", "sha256", "-f", key+".pub").Output()
	if !strings.Contains(string(fingerprint), doc.Principals[0].Keys[0].Fingerprint) {
		t.Fatalf("the listed fingerprint is not ssh-keygen's: %s", fingerprint)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".pose/policy/actions.json"))
	if !strings.Contains(string(raw), `"_comment": "kept"`) {
		t.Fatalf("editing lost the comment: %s", raw)
	}
	if code, out = runPose(t, root, "identity", "remove", "human:ada.lovelace", "--apply"); code != 0 || !strings.Contains(out, "stay verified") {
		t.Fatalf("remove: %d %s", code, out)
	}
	policy, _ := posemodel.LoadActionPolicy(root)
	if len(policy.KeysFor("human:ada.lovelace")) != 0 || len(policy.RolesOf("human:ada.lovelace")) != 1 {
		t.Fatalf("remove touched more than the key: %+v", policy)
	}
}

func TestSignedAnswerThroughTheCLIIsVerifiedAndReverified(t *testing.T) {
	keygen := sshKeygenOrSkip(t)
	root := identityFixture(t)
	key := newSSHKeyFile(t, keygen)
	if code, out := runPose(t, root, "identity", "add", "human:maintainer", "--key", key+".pub", "--role", "maintainer", "--apply"); code != 0 {
		t.Fatalf("identity add: %s", out)
	}
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), strings.Replace(readFileString(t, filepath.Join(root, ".pose/policy/actions.json")), `"identity_assurance": "declared"`, `"identity_assurance": "verified"`, 1))
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatalf("open: %s", out)
	}
	var view posemodel.ActionRequestView
	if err := json.Unmarshal([]byte(out), &view); err != nil {
		t.Fatal(err)
	}
	resolve := []string{"action", "resolve", view.Request.ID, "--actor", "human:maintainer", "--answer", "preserve-v1", "--request-digest", view.Request.RequestDigest, "--expected-revision", "1", "--idempotency-key", "k1"}
	if code, out = runPose(t, root, append(append([]string{}, resolve...), "--apply")...); code == 0 || !strings.Contains(out, "--sign") {
		t.Fatalf("an unsigned answer passed verified assurance: %d %s", code, out)
	}

	// A signature made elsewhere over `pose action statement`.
	_, statement := runPose(t, root, "action", "statement", view.Request.ID, "--actor", "human:maintainer", "--answer", "break-v1", "--idempotency-key", "k1")
	sign := exec.Command(keygen, "-Y", "sign", "-f", key, "-n", posemodel.ActionAnswerNamespace)
	sign.Stdin = strings.NewReader(statement)
	signature, err := sign.Output()
	if err != nil {
		t.Fatal(err)
	}
	sigPath := filepath.Join(t.TempDir(), "answer.sig")
	if err := os.WriteFile(sigPath, signature, 0o644); err != nil {
		t.Fatal(err)
	}
	// It signs break-v1, so it cannot carry preserve-v1.
	if code, out = runPose(t, root, append(append([]string{}, resolve...), "--signature", sigPath, "--apply")...); code == 0 {
		t.Fatalf("a signature for another answer was accepted: %s", out)
	}
	if code, out = runPose(t, root, append(append([]string{}, resolve...), "--sign", key, "--apply")...); code != 0 || !strings.Contains(out, "(verified)") {
		t.Fatalf("a signed answer was refused: %d %s", code, out)
	}
	_, shown := runPose(t, root, "action", "show", view.Request.ID)
	if !strings.Contains(shown, "re-verified from the journal") {
		t.Fatalf("show does not re-verify the signature:\n%s", shown)
	}
}

func TestDoctorReportsHumanRolesThatCannotProveAnAnswer(t *testing.T) {
	keygen := sshKeygenOrSkip(t)
	repo, _ := installedInstance(t)
	actions := filepath.Join(repo, ".pose/policy/actions.json")
	mustWrite(t, actions, `{"schema_version":1,"roles":{"maintainer":["human:ada"]},"identity_assurance":"verified"}`)
	f, _ := findDoctorFinding(runDoctorJSON(t, repo), "actions.keys")
	if f.Level != "warn" || !strings.Contains(f.Message, "human:ada") || !strings.Contains(f.Hint, "pose identity add") {
		t.Fatalf("an unprovable human role: %+v", f)
	}
	key := newSSHKeyFile(t, keygen)
	if code, out := runPose(t, repo, "identity", "add", "human:ada", "--key", key+".pub", "--apply"); code != 0 {
		t.Fatalf("identity add: %s", out)
	}
	f, _ = findDoctorFinding(runDoctorJSON(t, repo), "actions.keys")
	if f.Level != "warn" || !strings.Contains(f.Message, "cannot prove presence") {
		t.Fatalf("a key without presence: %+v", f)
	}
	mustWrite(t, actions, strings.Replace(readFileString(t, actions), `"schema_version": 1`, `"require_presence": true, "schema_version": 1`, 1))
	var report struct {
		Findings []doctorFinding `json:"findings"`
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		Main([]string{"doctor", "--json"}, &out, &errB)
		_ = json.Unmarshal(out.Bytes(), &report)
	})
	f, _ = findDoctorFinding(report.Findings, "actions.keys")
	if f.Level != "error" {
		t.Fatalf("require_presence with no presence key is not an error: %+v", f)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
