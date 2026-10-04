package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Spec pose-v7-legacy-cleanup-plan.

func treeListing(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			rel, _ := filepath.Rel(root, path)
			entries = append(entries, rel+"|"+info.ModTime().String()+"|"+strconv.FormatInt(info.Size(), 10))
		}
		return nil
	})
	sort.Strings(entries)
	return strings.Join(entries, "\n")
}

func TestMigrateDryRunInventoriesLegacyWithoutWriting(t *testing.T) {
	root := t.TempDir()
	spec := func(slug, status string) string {
		return "---\nslug: " + slug + "\nstatus: " + status + "\n---\n\n# Spec: " + slug + "\n"
	}
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-01-flat.md"), spec("flat", "done"))
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-01-dated/spec.md"), spec("dated", "done"))
	mustWrite(t, filepath.Join(root, ".pose/specs/undated/spec.md"), spec("undated", "in-progress"))
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-02-waiting.md"), spec("waiting", "blocked"))
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":1,"enabled":true,"profiles":{},"explicit_judgment_adopted_at":"2026-07-01","require_signed_attestion":true}`)
	before := treeListing(t, root)

	code, out := runPose(t, root, "migrate", "v7", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("dry-run: %d %s", code, out)
	}
	var inv struct {
		Counts  map[string]int      `json:"counts"`
		Details map[string][]string `json:"details"`
		Risks   []string            `json:"risks"`
		Writes  []string            `json:"writes"`
	}
	if err := json.Unmarshal([]byte(out), &inv); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	want := map[string]int{"spec-layout-flat": 2, "spec-layout-folder": 1, "spec-layout-legacy-folder": 1, "status-blocked": 1, "policy-legacy-adoption-keys": 1, "policy-unknown-keys": 1}
	for key, n := range want {
		if inv.Counts[key] != n {
			t.Errorf("%s = %d, want %d (%v)", key, inv.Counts[key], n, inv.Counts)
		}
	}
	// R3: a misspelled key that would have enabled a gate is surfaced, not ignored.
	if len(inv.Details["policy-unknown-keys"]) != 1 || inv.Details["policy-unknown-keys"][0] != "require_signed_attestion" || len(inv.Risks) == 0 {
		t.Fatalf("unknown key not reported as a risk: %+v %+v", inv.Details, inv.Risks)
	}
	if len(inv.Writes) != 0 {
		t.Fatalf("a dry-run declares writes: %v", inv.Writes)
	}
	if after := treeListing(t, root); after != before {
		t.Fatal("the dry-run changed the repository")
	}
	if code, out := runPose(t, root, "migrate", "v7", "--apply"); code != 2 || !strings.Contains(out, "only --dry-run") {
		t.Fatalf("6.x accepted an apply: %d %s", code, out)
	}
}
