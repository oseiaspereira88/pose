package pose

// Docs sync bundle (spec harne8-platform-wiki-sync): a deterministic,
// versioned snapshot of the governed docs eligible for the platform wiki
// projection — content + per-doc provenance (hash, last commit) + the
// live docs-check/review-pending signal, so the Conductor never has to
// re-derive anything POSE already computed. Read-only per ADR-003:
// building a bundle never writes; only `pose docs-sync push` (CLI) sends
// it anywhere.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DocsSyncBundleSchema is the highest schema_version this build produces.
const DocsSyncBundleSchema = 1

// DocsSyncDoc is one doc's content plus everything the wiki needs to
// render it with mandatory provenance (R5) without a second round trip.
type DocsSyncDoc struct {
	Path          string   `json:"path"`
	DocType       string   `json:"doc_type"`
	Topics        []string `json:"topics,omitempty"`
	Content       string   `json:"content"`
	ContentHash   string   `json:"content_hash"`
	LastCommit    string   `json:"last_commit,omitempty"`
	LastCommitAt  string   `json:"last_commit_at,omitempty"`
	CheckErrors   int      `json:"check_errors"`
	CheckWarnings int      `json:"check_warnings"`
	ReviewPending bool     `json:"review_pending"`
}

// DocsSyncBundle is the deterministic export `pose docs-sync export`
// produces and `pose docs-sync push` sends to the Conductor's ingestion
// endpoint (R1). BundleHash is computed over the serialized Docs slice so
// ingestion can be idempotent by content (R2) — never over GeneratedAt,
// which would make every export look "new".
type DocsSyncBundle struct {
	SchemaVersion     int           `json:"schema_version"`
	ProjectID         string        `json:"project_id,omitempty"`
	GeneratedAt       string        `json:"generated_at"`
	BaselineCommit    string        `json:"baseline_commit,omitempty"`
	Docs              []DocsSyncDoc `json:"docs"`
	ExcludedSensitive int           `json:"excluded_sensitive"`
	BundleHash        string        `json:"bundle_hash"`
}

// BuildDocsSyncBundle assembles a bundle from the current manifest, docs-
// check result and git history. Sensitive entries (Restrições) are
// counted but never included. A doc that fails the "missing" rule (file
// gone) is silently excluded from the bundle too — docs-check already
// reports that as an error; the bundle just never carries content that
// doesn't exist.
func (s Store) BuildDocsSyncBundle(ctx context.Context, projectID string) (*DocsSyncBundle, error) {
	if !s.HasDocsManifest() {
		return nil, fmt.Errorf("pose: docs manifest not found at %s (run `pose docs-init`)", s.DocsManifestPath())
	}
	manifest, err := s.LoadDocsManifest()
	if err != nil {
		return nil, err
	}
	result := s.CheckDocs(ctx, manifest)

	issueCounts := map[string]struct{ errors, warnings int }{}
	for _, issue := range result.Issues {
		c := issueCounts[issue.Path]
		if issue.Severity == "error" {
			c.errors++
		} else if issue.Severity == "warning" {
			c.warnings++
		}
		issueCounts[issue.Path] = c
	}
	pending := map[string]bool{}
	for _, rp := range result.ReviewPending {
		pending[rp.Doc] = len(rp.Triggers) > 0
	}

	bundle := &DocsSyncBundle{
		SchemaVersion:  DocsSyncBundleSchema,
		ProjectID:      projectID,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		BaselineCommit: gitHeadCommitForSync(s.Root),
	}
	for _, entry := range manifest.Entries {
		if entry.Sensitive {
			bundle.ExcludedSensitive++
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(entry.Path)))
		if err != nil {
			continue
		}
		commit, commitAt := s.lastCommitInfo(ctx, entry.Path)
		counts := issueCounts[entry.Path]
		bundle.Docs = append(bundle.Docs, DocsSyncDoc{
			Path: entry.Path, DocType: entry.DocType, Topics: entry.Topics,
			Content: string(raw), ContentHash: ContentHash12(string(raw)),
			LastCommit: commit, LastCommitAt: commitAt,
			CheckErrors: counts.errors, CheckWarnings: counts.warnings,
			ReviewPending: pending[entry.Path],
		})
	}
	bundle.BundleHash = hashDocsSyncDocs(bundle.Docs)
	return bundle, nil
}

// lastCommitInfo returns the hash and ISO8601 date of the last commit
// that touched relPath. Empty strings when git history is unavailable —
// never an error (same posture as lastCommitDate/commitsSince).
func (s Store) lastCommitInfo(ctx context.Context, relPath string) (hash, at string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", s.Root, "log", "-1", "--format=%H%x1f%cI", "--", relPath)
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	fields := strings.SplitN(strings.TrimSpace(string(out)), "\x1f", 2)
	if len(fields) != 2 {
		return "", ""
	}
	return fields[0], fields[1]
}

func gitHeadCommitForSync(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// hashDocsSyncDocs fingerprints the bundle's content deterministically:
// same docs, same hash, regardless of GeneratedAt — the idempotency key
// the Conductor's ingestion endpoint dedupes on (R2).
func hashDocsSyncDocs(docs []DocsSyncDoc) string {
	var b strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&b, "%s\x1f%s\x1f%s\x1e", d.Path, d.ContentHash, d.LastCommit)
	}
	return ContentHash12(b.String())
}
