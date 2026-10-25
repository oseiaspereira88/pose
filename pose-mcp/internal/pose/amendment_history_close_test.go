package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAmendmentHistorySeedValidatesStartAndKeepsRecordedNodes(t *testing.T) {
	for _, tc := range []string{"missing", "valid", "wrong-spec", "incomplete", "tampered", "malformed"} {
		t.Run(tc, func(t *testing.T) {
			store := Store{Root: t.TempDir()}
			record := StartRecord{SchemaVersion: StartSchemaVersion, Spec: "alpha", Phase: "started", Baseline: ProjectContractNodes("alpha", "## 2. Requirements\n- R1: original\n- R2: unchanged\n")}
			switch tc {
			case "wrong-spec":
				record.Spec = "other"
			case "incomplete":
				record.Phase = "baseline-recorded"
			case "tampered":
				record.Baseline.Nodes[0].Hash = "tampered"
			}
			if tc != "missing" {
				if err := writeStartRecord(startRecordPath(store.Root, "alpha"), record, true); err != nil {
					t.Fatal(err)
				}
			}
			if tc == "malformed" {
				if err := os.WriteFile(startRecordPath(store.Root, "alpha"), []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			seed, err := store.AmendmentBaselineFromStart("alpha", true, "@now", "2026-10-24T00:00:00Z")
			switch tc {
			case "missing":
				if err != nil || seed != nil {
					t.Fatalf("legacy fabricated: %+v %v", seed, err)
				}
			case "valid":
				if err != nil || seed == nil || len(seed.IDs) != 2 || seed.Author != "@now" || seed.At != "2026-10-24T00:00:00Z" || seed.Assurance != "declared" {
					t.Fatalf("seed: %+v %v", seed, err)
				}
			default:
				if err == nil || seed != nil {
					t.Fatalf("invalid start accepted: %+v %v", seed, err)
				}
			}
		})
	}
}

func TestAmendmentHistoryInspectionPreservesOptInAndRefusesCorruption(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	spec := Spec{Slug: "alpha", Path: filepath.Join(root, "alpha.md"), Body: "## 2. Requirements\n- R1: changed\n"}
	if _, findings, err := store.InspectAmendmentHistory(spec); err != nil || len(findings) != 0 {
		t.Fatalf("missing journal imposed gate: %v %v", findings, err)
	}
	path := AmendmentsPath(spec.Path)
	if err := os.WriteFile(path, []byte("{broken\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.InspectAmendmentHistory(spec); err == nil {
		t.Fatal("corruption ignored")
	}
	if err := os.WriteFile(path, []byte(`{"schema":1,"at":"2026-10-10T00:00:00Z","change":"baseline","ids":["R1"],"author":"@old","hashes":{"R1":"old"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, findings, err := store.InspectAmendmentHistory(spec)
	if err != nil || len(findings) != 1 || !strings.Contains(findings[0], "R1 changed") {
		t.Fatalf("gate lost: %v %v", findings, err)
	}
}
