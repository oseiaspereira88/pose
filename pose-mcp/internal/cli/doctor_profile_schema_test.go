// A review profile left behind is visible
// (spec pose-doctor-reports-a-profile-left-behind).
//
// Two profiles shipped at `schema_version: 1` for long enough that every
// instance installed in that window still carries them, and the migration that
// should have moved them copied a v1 file over a v1 file and reported success.
// The shipped files are v2 now, so `pose update` migrates them — but an instance
// that has not updated since is carrying profiles exempt from the closed rule
// and evidence catalogs schema v2 enforces, and nothing said so.

package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReportsAReviewProfileBelowTheCurrentSchema(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "legacy.json"),
		`{"schema_version":1,"id":"legacy","version":1,"scope":"spec","criteria":[{"id":"c","description":"d"}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "current.json"),
		`{"schema_version":2,"id":"current","version":1,"scope":"spec","criteria":[{"id":"c","description":"d"}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.profile-schema")
	if !ok {
		t.Fatal("no review.profile-schema finding")
	}
	if f.Level != "warn" {
		t.Fatalf("level = %q, want warn: %s", f.Level, f.Message)
	}
	if !strings.Contains(f.Message, "legacy.json") {
		t.Errorf("the finding does not name the profile left behind: %q", f.Message)
	}
	// Naming the current one would be the finding inverted: it would tell an
	// operator to migrate a profile that is already migrated.
	if strings.Contains(f.Message, "current.json") {
		t.Errorf("the finding names a profile that is already current: %q", f.Message)
	}
	if !strings.Contains(f.Hint, "pose update") {
		t.Errorf("the finding names a problem with no remedy: %q", f.Hint)
	}
}

// An instance whose profiles are all current gets the ok, so the check is known
// to be reading them rather than reporting on nothing.
func TestDoctorAcceptsProfilesAtTheCurrentSchema(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "current.json"),
		`{"schema_version":2,"id":"current","version":1,"scope":"spec","criteria":[{"id":"c","description":"d"}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.profile-schema")
	if !ok || f.Level != "ok" {
		t.Errorf("profiles at the current schema were reported: %+v", f)
	}
}

// An instance with no profiles at all is not a finding: there is nothing left
// behind, and saying so on every fresh repository is how a check is learned to
// be ignored.
func TestDoctorSaysNothingWhenThereAreNoProfiles(t *testing.T) {
	root := doctorTrailerFixture(t)
	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.profile-schema"); ok {
		t.Errorf("reported on an instance with no review profiles: %+v", f)
	}
}
