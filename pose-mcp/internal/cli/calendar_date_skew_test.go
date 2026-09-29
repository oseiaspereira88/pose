package cli

// POSE stamps created_at and completed_at in UTC while a person filling one in
// writes their local date (spec calendar-dates-tolerate-utc-stamping). West of
// UTC, a spec created in the evening is stamped with tomorrow's date, and
// closing it by hand the same evening wrote a completion "earlier" than its
// creation — found by the quickstart script at 23:33 in UTC-3.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lintDates(t *testing.T, created, completed string) string {
	t.Helper()
	t.Setenv("POSE_LOCALE", "en")
	path := filepath.Join(t.TempDir(), "spec.md")
	body := "---\nslug: dated\nstatus: done\ncreated_at: " + created + "\ncompleted_at: " + completed + "\n---\n\n# Spec: dated\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	_ = lintOneSpec(path, true, false, &out, &errB)
	return out.String() + errB.String()
}

func TestCompletionOneDayBeforeUTCCreationIsZoneSkew(t *testing.T) {
	const refusal = "completed_at is earlier than created_at"
	for _, tc := range []struct {
		name, created, completed string
		refused                  bool
	}{
		{"same day", "2026-09-29", "2026-09-29", false},
		{"local date one day behind the UTC stamp", "2026-09-29", "2026-09-28", false},
		{"two days earlier is not skew", "2026-09-29", "2026-09-27", true},
		{"instants compare exactly", "2026-09-29T02:33:00Z", "2026-09-28T23:59:00Z", true},
		{"instant and date compare exactly", "2026-09-29T02:33:00Z", "2026-09-28", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Contains(lintDates(t, tc.created, tc.completed), refusal)
			if got != tc.refused {
				t.Fatalf("created %s, completed %s: refused=%v, want %v", tc.created, tc.completed, got, tc.refused)
			}
		})
	}
}
