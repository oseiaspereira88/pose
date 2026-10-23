package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Force upgrade stages shipped governance before the final checker reads the task map.
// Missing physical files must not make the projected gate or availability reader fail.
func TestUpdatePreviewForceStagesInstructionsBeforeFinalGate(t *testing.T) {
	root := staleInstance(t)
	refs := []string{".pose/rules/security.md", ".pose/workflows/feature.md", ".agents/skills/pose-feature/SKILL.md"}
	for _, ref := range refs {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(ref))); err != nil {
			t.Fatal(err)
		}
	}
	before := instanceState(t, root)
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	p := posemodel.NewPreviewFiles(now)
	u := instanceUpdate{root: root, force: true, locale: localeEN}
	var preview, apply bytes.Buffer
	pc := u.apply(p, &preview, &preview)
	if pc != 0 {
		t.Fatalf("projected final gate: %d\n%s", pc, &preview)
	}
	sameEffects(t, "preview writes", nil, appliedEffects(before, instanceState(t, root)))
	for _, ref := range refs {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref))); !os.IsNotExist(err) {
			t.Fatalf("physical reference was created: %s: %v", ref, err)
		}
		if _, err := p.ReadFile(filepath.Join(root, filepath.FromSlash(ref))); err != nil {
			t.Fatal(err)
		}
	}
	var suggested bytes.Buffer
	if code := cmdSuggestIn(p, root, []string{"feature", "--json"}, &suggested, &suggested); code != 0 {
		t.Fatalf("projected suggest: %d %s", code, &suggested)
	}
	var doc struct {
		References []referenceStatus `json:"references_status"`
		Rules      []ruleStatus      `json:"rules_status"`
	}
	if err := json.Unmarshal(suggested.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, ref := range doc.References {
		if ref.Availability != "available" {
			t.Fatalf("projected reference: %+v", ref)
		}
	}
	for _, rule := range doc.Rules {
		if rule.Rule == "security" && rule.Availability != ruleInstalled {
			t.Fatalf("projected rule: %+v", rule)
		}
	}
	if ac := u.apply(posemodel.NativeFilesAt(now), &apply, &apply); ac != pc || apply.String() != preview.String() {
		t.Fatalf("force parity: preview=%d\n%s\napply:\n%s", pc, &preview, &apply)
	}
}

// Candidate judgment must consume staged policy, archive, notes, manifest, version and
// journal inputs. The committed tag is independently authoritative after it exists.
func TestUpdatePreviewReleaseCurrentTreeAndHistoricalTagStaySeparate(t *testing.T) {
	root := cliReleaseRepo(t)
	p := posemodel.NewPreviewFiles(time.Now())
	if gaps, m := checkRelease(root, "v1.0.0", p); m == nil || len(gaps) != 0 {
		t.Fatalf("fixture: %v", gaps)
	}
	version := filepath.Join(root, "VERSION")
	if err := p.WriteFile(version, []byte("2.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if gaps, _ := checkRelease(root, "v1.0.0", p); !strings.Contains(strings.Join(gaps, ";"), "version file says v2.0.0") {
		t.Fatalf("staged version lost: %v", gaps)
	}
	if gaps, _ := checkRelease(root, "v1.0.0"); len(gaps) != 0 {
		t.Fatalf("physical candidate changed: %v", gaps)
	}
	// Restore the staged version; changing policy is a candidate defect.
	if err := p.WriteFile(version, []byte("1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(root, ".pose/policy/release.json")
	raw, err := p.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte(`"allow_empty":false`), []byte(`"allow_empty":true`), 1)
	if err := p.WriteFile(policy, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if gaps, _ := checkRelease(root, "v1.0.0", p); !strings.Contains(strings.Join(gaps, ";"), "policy digest differs") {
		t.Fatalf("staged policy lost: %v", gaps)
	}
	if out, err := exec.Command("git", "-C", root, "tag", "v1.0.0").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	// A snapshot of a committed text source must parse without temporary writes.
	t.Setenv("TMPDIR", filepath.Join(root, "does-not-exist"))
	snap, err := posemodel.ReadReleaseSnapshot(root, "v1.0.0")
	if err != nil || !snap.Intact() {
		t.Fatalf("tag snapshot: %v %+v", err, snap)
	}
	if gaps, _ := checkRelease(root, "v1.0.0", p); len(gaps) != 0 {
		t.Fatalf("current policy contaminated historical cut: %v", gaps)
	}
	archive := filepath.Join(root, ".pose/changelogs/v1.0.0/alpha.md")
	if err := p.WriteFile(archive, []byte("corrupt staged fragment"), 0o644); err != nil {
		t.Fatal(err)
	}
	if gaps, _ := checkRelease(root, "v1.0.0", p); len(gaps) == 0 {
		t.Fatal("staged archive corruption ignored")
	}
	if snap, err := posemodel.ReadReleaseSnapshot(root, "v1.0.0"); err != nil || !snap.Intact() {
		t.Fatalf("staged bytes replaced immutable tag: %v %+v", err, snap)
	}
	manifest := filepath.Join(root, ".pose/releases/v1.0.0/manifest.json")
	if err := p.WriteFile(manifest, []byte(`{"version":"v1.0.0","version":"v1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if gaps, m := checkRelease(root, "v1.0.0", p); m != nil || !strings.Contains(strings.Join(gaps, ";"), "manifest missing or invalid") {
		t.Fatalf("strict staged manifest decoding lost: %v", gaps)
	}
}
