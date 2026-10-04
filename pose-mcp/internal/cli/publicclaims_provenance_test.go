package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-public-claims-publication-provenance.

func writeReleaseRecord(t *testing.T, root, version string, events ...string) {
	t.Helper()
	dir := filepath.Join(root, ".pose", "releases", version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "manifest.json"), `{"schema_version":1,"version":"`+version+`","prepared_at":"2026-10-02T00:00:00Z","specs":[],"categories":{},"fragments":[],"breaking":false,"notes_digest":"","release_input_digest":"","policy_digest":"","version_evidence":{}}`)
	if len(events) > 0 {
		lines := []string{}
		for _, state := range events {
			lines = append(lines, `{"schema_version":1,"version":"`+version+`","state":"`+state+`","recorded_at":"2026-10-02T00:00:00Z","evidence":{"schema_version":1,"provider":"github","repository":"o/r","version":"`+version+`","tag":"`+version+`","commit":"c","published_at":"2026-10-02T00:00:00Z","url":"u"},"evidence_digest":"sha256:x"}`)
		}
		mustWrite(t, filepath.Join(dir, "events.jsonl"), strings.Join(lines, "\n")+"\n")
	}
}

func claimsProvenance(t *testing.T, root string) publicVersionProvenance {
	t.Helper()
	code, out := runClaims(t, root, "--json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	var payload struct {
		Released   string                  `json:"released_version"`
		Provenance publicVersionProvenance `json:"version_provenance"`
		Deprecated map[string]string       `json:"deprecated_fields"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if payload.Released != payload.Provenance.CandidateVersion || payload.Deprecated["released_version"] == "" {
		t.Fatalf("legacy field must keep the candidate and be marked deprecated: %+v", payload)
	}
	return payload.Provenance
}

func TestPublicClaimsReportsAPreparedCandidateAsPreparedNeverPublished(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"}`)
	writeSurface(t, root, "README.md", "POSE 1.7.10\n")
	writeReleaseRecord(t, root, "v1.7.10")
	writeReleaseRecord(t, root, "v1.7.9", "tagged", "published")

	p := claimsProvenance(t, root)
	if p.CandidateVersion != "1.7.10" || p.CandidateState != "prepared" || p.PreparedVersion != "1.7.10" {
		t.Fatalf("prepared candidate misreported: %+v", p)
	}
	if p.PublishedVersion != "1.7.9" || p.PublishedVersionState != "published" || !strings.HasSuffix(p.PublishedSource, "v1.7.9/events.jsonl") {
		t.Fatalf("the latest proven publication is the older release: %+v", p)
	}

	_, out := runClaims(t, root, "--strict")
	if !strings.Contains(out, "public-claims.candidate_state=prepared") || !strings.Contains(out, "public-claims.published_version=1.7.9 (published") {
		t.Fatalf("terminal does not separate prepared from published: %s", out)
	}
}

func TestPublicClaimsWithoutRetainedPublicationEvidenceSaysUnproven(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"}`)
	writeSurface(t, root, "README.md", "POSE 1.7.10\n")

	p := claimsProvenance(t, root)
	if p.CandidateState != "unprepared" || p.PublishedVersion != "" || p.PublishedVersionState != "unproven" {
		t.Fatalf("local metadata alone was read as more than a candidate: %+v", p)
	}
	_, out := runClaims(t, root, "--strict")
	if !strings.Contains(out, "public-claims.published_version=unproven") {
		t.Fatalf("missing publication evidence not rendered as unproven: %s", out)
	}
}

func TestPublicClaimsRejectsAPublicationWithLifecycleGaps(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"README.md","version_claims":"current-only"}`)
	writeSurface(t, root, "README.md", "POSE 1.7.10\n")
	// published without a tagged predecessor is a gap, not proof.
	writeReleaseRecord(t, root, "v1.7.10", "published")
	p := claimsProvenance(t, root)
	if p.PublishedVersion != "" {
		t.Fatalf("a publication record with lifecycle gaps was accepted as proof: %+v", p)
	}
}
