// Every shipped review profile is at the current schema
// (spec pose-shipped-review-profiles-are-schema-v2).
//
// `milestone-integration.json` and `roadmap-outcome.json` shipped at
// `schema_version: 1`, and the v1-to-v2 migration copies the distribution's own
// file over the instance's. So it copied a v1 file over a v1 file and logged
// `review-profile (migrated): ... (v1 -> v2)` — on every `pose update`, forever,
// telling the operator a migration had succeeded that never happened.
//
// It surfaced while adopting the release in a consumer repository, not from a
// report: the log said migrated twice per run and the files never changed.

package cli

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/scaffold"
)

func TestEveryShippedReviewProfileIsAtTheCurrentSchema(t *testing.T) {
	entries, err := fs.ReadDir(scaffold.Dist(), ".pose/review-profiles")
	if err != nil {
		t.Fatalf("reading the shipped review profiles: %v", err)
	}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(scaffold.Dist(), ".pose/review-profiles/"+entry.Name())
		if err != nil {
			t.Errorf("%s: %v", entry.Name(), err)
			continue
		}
		var profile struct {
			SchemaVersion int `json:"schema_version"`
		}
		if err := json.Unmarshal(raw, &profile); err != nil {
			t.Errorf("%s is not valid JSON: %v", entry.Name(), err)
			continue
		}
		checked++
		if profile.SchemaVersion != posemodel.ReviewPolicySchemaVersion {
			t.Errorf("%s ships at schema_version %d, not %d — the v1-to-v2 migration copies this file over an instance's and would report a migration it did not perform",
				entry.Name(), profile.SchemaVersion, posemodel.ReviewPolicySchemaVersion)
		}
	}
	if checked == 0 {
		t.Fatal("no shipped review profile was examined — the scaffold lookup is broken, not the profiles")
	}
}

// And the migration itself no longer claims what the distribution cannot
// deliver: a shipped profile that is not current takes the explicit-rewrite
// branch instead of the copy-and-log one.
func TestTheMigrationOnlyClaimsACopyThatMigrates(t *testing.T) {
	current := []byte(`{"schema_version":2,"id":"x"}`)
	stale := []byte(`{"schema_version":1,"id":"x"}`)
	if !shippedProfileIsCurrentSchema(current) {
		t.Error("a current shipped profile was treated as stale")
	}
	if shippedProfileIsCurrentSchema(stale) {
		t.Error("a stale shipped profile would be copied over an instance's and logged as a migration")
	}
	if shippedProfileIsCurrentSchema([]byte("not json")) {
		t.Error("unreadable content was treated as a current profile")
	}
}
