package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReleaseManifestAndNotesAreDeterministic(t *testing.T) {
	fragments := []ReleaseFragment{{Spec: "alpha", Category: "added", Body: "Adds alpha.", Path: "alpha.md", Digest: "sha256:a"}, {Spec: "beta", Category: "fixed", Body: "Fixes beta.", Path: "beta.md", Digest: "sha256:b"}}
	policy := ReleasePolicy{SchemaVersion: 1, AdoptedAt: "2026-08-03", Provider: "github", Repository: "owner/repo"}
	evidence := map[string]string{"source": "version.go", "value": "v1.2.0"}
	a := NewReleaseManifest("v1.2.0", "v1.1.0", "2026-08-03T00:00:00Z", fragments, policy, evidence)
	b := NewReleaseManifest("v1.2.0", "v1.1.0", "2026-08-03T00:00:00Z", fragments, policy, evidence)
	if ReleaseDigest(a) != ReleaseDigest(b) || a.NotesDigest != ReleaseDigest(RenderReleaseNotes("v1.2.0", fragments)) {
		t.Fatalf("release snapshot is not deterministic: %+v %+v", a, b)
	}
}

func TestReleaseFragmentsRejectMalformedDuplicateAndSymlink(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte("missing frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseFragments(dir); err == nil {
		t.Fatal("malformed fragment accepted")
	}
	if err := os.Remove(filepath.Join(dir, "bad.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(dir, "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseFragments(dir); err == nil {
		t.Fatal("symlink release fragment escaped confinement")
	}
}

func TestMissingReleasePolicyIsActionable(t *testing.T) {
	if _, err := LoadReleasePolicy(t.TempDir()); err == nil || !strings.Contains(err.Error(), ".pose/release-policy.json") {
		t.Fatalf("missing policy error=%v", err)
	}
}

func TestReleaseProjectionRequiresPublicationBoundVerification(t *testing.T) {
	manifest := &ReleaseManifest{Version: "v1.2.0"}
	evidence := ReleaseEvidence{SchemaVersion: 1, Provider: "github", Repository: "owner/repo", Version: "v1.2.0", Tag: "v1.2.0", Commit: "0123456789012345678901234567890123456789"}
	tagged := NewReleaseEvent("v1.2.0", "tagged", evidence, time.Unix(1, 0))
	published := NewReleaseEvent("v1.2.0", "published", evidence, time.Unix(2, 0))
	verifiedEvidence := evidence
	verifiedEvidence.Publication = "sha256:not-the-publication"
	verified := NewReleaseEvent("v1.2.0", "verified", verifiedEvidence, time.Unix(3, 0))
	projection := ProjectRelease(manifest, []ReleaseEvent{tagged, published, verified})
	if projection.State == "verified" || len(projection.Gaps) == 0 {
		t.Fatalf("forged verification advanced release: %+v", projection)
	}
	verified.Evidence.Publication = published.EvidenceDigest
	projection = ProjectRelease(manifest, []ReleaseEvent{tagged, published, verified})
	if projection.State != "verified" || len(projection.Gaps) != 0 {
		t.Fatalf("bound verification did not advance release: %+v", projection)
	}
}
