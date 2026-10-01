package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReportsConflictingAdoptionDates(t *testing.T) {
	for _, tc := range []struct {
		mapped, legacy string
		want           bool
	}{
		{"2026-10-01", "2026-09-01", true}, {"", "2026-09-01", true},
		{"2026-10-01", "2026-10-01", false},
	} {
		root := doctorTrailerFixture(t)
		mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), fmt.Sprintf(`{"schema_version":2,"contract_adoptions":{"component-aware":%q},"component_aware_adopted_at":%q}`, tc.mapped, tc.legacy))
		finding, found := findDoctorFinding(runDoctorJSON(t, root), "review.adoption-conflict")
		if found != tc.want || found && (finding.Level != "warn" || !strings.Contains(finding.Hint, "precedence")) {
			t.Fatalf("conflict: %+v", finding)
		}
	}
}

func TestDoctorCountsLegacySealedBundleFields(t *testing.T) {
	root := doctorTrailerFixture(t)
	if _, found := findDoctorFinding(runDoctorJSON(t, root), "review.legacy-bundles"); found {
		t.Fatal("empty repository reported as legacy")
	}
	mustWrite(t, filepath.Join(root, ".pose/review-bundles/old.json"), `{"state":"sealed","payload":{}}`)
	mustWrite(t, filepath.Join(root, ".pose/review-bundles/current.json"), `{"state":"sealed","payload":{"governing_contracts":["explicit-judgment"],"gates":{}}}`)
	finding, found := findDoctorFinding(runDoctorJSON(t, root), "review.legacy-bundles")
	if !found || finding.Level != "warn" || !strings.Contains(finding.Message, "2 sealed bundle(s): 1 without governing contracts, 1 without sealed gates") {
		t.Fatalf("counts: %+v", finding)
	}
}

func TestDoctorNamesStaleProfileDeclarationWithoutRewriting(t *testing.T) {
	root := doctorTrailerFixture(t)
	path := filepath.Join(root, ".pose/review-profiles/legacy.json")
	body := `{"schema_version":1,"id":"legacy","version":1,"scope":"spec","criteria":[{"id":"c","evidence_classes":["invented-class"]}]}`
	mustWrite(t, path, body)
	finding, found := findDoctorFinding(runDoctorJSON(t, root), "review.profile-schema")
	if !found || !strings.Contains(finding.Message, "invented-class") {
		t.Fatalf("invalid declaration missing: %+v", finding)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != body {
		t.Fatalf("diagnostic rewrote profile: %v", err)
	}
}
