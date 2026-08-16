package cli

// Instance config completeness and module-metadata orphan detection (specs
// pose-update-instance-config-completeness, pose-fixture-directory-discovery-exclusion):
// an old instance whose schema-version claims it is current but is missing
// subsystems its own manuals reference used to pass `pose doctor` silently.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findingLevel(t *testing.T, out []byte, check string) string {
	t.Helper()
	var report doctorJSON
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("invalid doctor JSON: %v\n%s", err, out)
	}
	for _, f := range report.Findings {
		if f.Check == check {
			return f.Level
		}
	}
	t.Fatalf("no finding for check %q in report: %s", check, out)
	return ""
}

func TestDoctorFlagsIncompleteInstanceConfig(t *testing.T) {
	repo := doctorFixture(t)
	// Simulate an instance a plain `pose update` (pre-fix) left with docs
	// already referencing subsystems it never seeded.
	if err := os.RemoveAll(filepath.Join(repo, ".pose", "policy")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, ".pose", "indexes", "spec-graph.json")); err != nil {
		t.Fatal(err)
	}

	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("doctor --json exit=%d err=%s", code, errB.String())
		}
		if level := findingLevel(t, out.Bytes(), "instance.config-completeness"); level != "warn" {
			t.Errorf("instance.config-completeness level = %q, want \"warn\"", level)
		}
	})

	// The suggested remediation ("run pose update") must actually resolve
	// it: this is the other half of the fix — seeding must not be gated
	// behind --force.
	var updateOut, updateErr bytes.Buffer
	if code := cmdUpdate(repo, []string{"--no-self"}, &updateOut, &updateErr); code != 0 {
		t.Fatalf("pose update exit=%d err=%s out=%s", code, updateErr.String(), updateOut.String())
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &out, &errB); code != 0 {
			t.Fatalf("doctor --json exit=%d err=%s", code, errB.String())
		}
		if level := findingLevel(t, out.Bytes(), "instance.config-completeness"); level != "ok" {
			t.Errorf("instance.config-completeness level after `pose update` = %q, want \"ok\"", level)
		}
	})
}

func TestDoctorFlagsOrphanModuleMetadataEntries(t *testing.T) {
	repo := doctorFixture(t)
	metaPath := filepath.Join(repo, ".pose", "indexes", "module-metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		SchemaVersion int                          `json:"schemaVersion"`
		Defaults      map[string]string            `json:"defaults"`
		Modules       map[string]map[string]string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Modules == nil {
		doc.Modules = map[string]map[string]string{}
	}
	// A self-referential ghost entry (the exact shape of the residual issue
	// #21-adjacent contamination) and a mis-cased path that resolves to
	// nothing real.
	doc.Modules["pose-mcp"] = map[string]string{"criticality": "medium", "domain": "go", "validationProfile": "baseline"}
	doc.Modules["Nonexistent/Module"] = map[string]string{"criticality": "medium", "domain": "go", "validationProfile": "baseline"}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, out, 0o644); err != nil {
		t.Fatal(err)
	}

	inDir(t, repo, func() {
		var stdout, stderr bytes.Buffer
		if code := Main([]string{"doctor", "--json"}, &stdout, &stderr); code != 0 {
			t.Fatalf("doctor --json exit=%d err=%s", code, stderr.String())
		}
		var report doctorJSON
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("invalid doctor JSON: %v\n%s", err, stdout.String())
		}
		found := false
		for _, f := range report.Findings {
			if f.Check != "module-metadata.orphan-entries" {
				continue
			}
			found = true
			if f.Level != "warn" {
				t.Errorf("module-metadata.orphan-entries level = %q, want \"warn\"", f.Level)
			}
			if !strings.Contains(f.Message, "pose-mcp") || !strings.Contains(f.Message, "Nonexistent/Module") {
				t.Errorf("finding message does not name both orphans: %q", f.Message)
			}
		}
		if !found {
			t.Fatal("no module-metadata.orphan-entries finding reported")
		}
	})
}
