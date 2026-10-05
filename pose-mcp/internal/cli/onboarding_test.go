package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-onboarding-spec.

func onboardingInstall(t *testing.T, locale string) string {
	t.Helper()
	isolateHome(t)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--project-id", "proj.onboard", "--project-name", "Onboard", "--locale", locale}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	if !strings.Contains(out.String(), "onboarding spec: .pose/specs/") && !strings.Contains(out.String(), "spec de onboarding: .pose/specs/") {
		t.Fatalf("install did not say it scaffolded the onboarding spec:\n%s", out.String())
	}
	return repo
}

func gitCommitAll(t *testing.T, repo, message string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", message, "--no-verify"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func TestOnboardingSpecIsScaffoldedReadyToStartInEitherLocale(t *testing.T) {
	for locale, heading := range map[string]string{"en": "# Spec: Adopt POSE in Onboard", "pt-BR": "# Spec: Adotar o POSE em Onboard"} {
		t.Run(locale, func(t *testing.T) {
			repo := onboardingInstall(t, locale)
			rel := findOnboardingSpec(repo)
			raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
			if err != nil || !strings.Contains(string(raw), heading) || !strings.Contains(string(raw), "status: draft") {
				t.Fatalf("onboarding spec %q: %v\n%s", rel, err, raw)
			}
			for _, want := range []string{".pose/project.json", "pose identity add", "pose hooks install", "pose setup", "pose close spec:pose-onboarding"} {
				if !strings.Contains(string(raw), want) {
					t.Errorf("the spec does not name %q", want)
				}
			}
			if code, out := runPose(t, repo, "lint-spec", "pose-onboarding", "--design-check"); code != 0 || strings.Contains(out, "✖") || strings.Contains(out, "⚠") {
				t.Fatalf("the scaffolded spec does not lint clean: %d %s", code, out)
			}
			if code, out := runPose(t, repo, "check", "--strict"); code != 0 {
				t.Fatalf("the scaffolded spec breaks the strict check: %s", out)
			}
			if code, out := runPose(t, repo, "start", "spec:pose-onboarding"); code != 0 || !strings.Contains(out, "start.ready=true") {
				t.Fatalf("the scaffolded spec is not ready to start: %d %s", code, out)
			}
		})
	}
}

func TestOnboardingSpecIsNeverCreatedByUpdateNorOverwritten(t *testing.T) {
	repo := onboardingInstall(t, "en")
	rel := findOnboardingSpec(repo)
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.WriteFile(path, append(mustRead(t, path), []byte("\n<!-- edited by the project -->\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--force"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall: %s", errB.String())
	}
	if !strings.Contains(string(mustRead(t, path)), "edited by the project") {
		t.Fatal("an install overwrote the onboarding spec")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update: %s", errB.String())
	}
	if code := cmdInstall([]string{repo, "--force"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall: %s", errB.String())
	}
	if findOnboardingSpec(repo) != "" {
		t.Fatal("update or a reinstall over an existing instance created the onboarding spec")
	}
}

func TestSetupDrivesTheOnboardingSpecToClose(t *testing.T) {
	keygen := sshKeygenOrSkip(t)
	repo := onboardingInstall(t, "en")
	if next := setupJSON(t, repo).Next; next == nil || next.Command != "pose start spec:pose-onboarding" {
		t.Fatalf("a draft onboarding spec is not the first step: %+v", next)
	}
	_, preview := runPose(t, repo, "start", "spec:pose-onboarding")
	digest := ""
	for _, line := range strings.Split(preview, "\n") {
		if strings.HasPrefix(line, "start.digest=") {
			digest = strings.TrimPrefix(line, "start.digest=")
		}
	}
	if code, out := runPose(t, repo, "start", "spec:pose-onboarding", "--apply", "--digest", digest); code != 0 {
		t.Fatalf("start: %s", out)
	}
	if step := stepOf(setupJSON(t, repo), "onboarding"); step.State != "optional" {
		t.Fatalf("an in-progress onboarding spec with open steps: %+v", step)
	}
	key := newSSHKeyFile(t, keygen)
	if code, out := runPose(t, repo, "identity", "add", "human:ada", "--key", key+".pub", "--role", "maintainer", "--apply"); code != 0 {
		t.Fatalf("identity: %s", out)
	}
	if code, out := runPose(t, repo, "hooks", "install"); code != 0 {
		t.Fatalf("hooks: %s", out)
	}
	gitCommitAll(t, repo, "Adopt POSE\n\nPOSE-Spec: pose-onboarding")
	plan := setupJSON(t, repo)
	if plan.Next == nil || plan.Next.ID != "onboarding" || plan.Next.Command != "pose close spec:pose-onboarding" {
		t.Fatalf("with every step done, closing the onboarding spec is not next: %+v\n%+v", plan.Next, plan.Steps)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
