package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func writeAssessmentCLIFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssessDiscoverUpdatesProjectStateFromObservedData(t *testing.T) {
	root := filepath.Join(t.TempDir(), "neutral-cli")
	writeAssessmentCLIFile(t, root, "service/main.go", "package main\n// TODO: tracked later\nfunc main() {}\n")
	writeAssessmentCLIFile(t, root, ".pose/state/components/removed.json", `{"schema_version":1,"component_slug":"removed","metrics":{"loc_production":999},"status":"verified","completeness_score":1}`)
	writeAssessmentCLIFile(t, root, ".pose/state/project-state.md", `---
schema_version: 1
generated_at: 2026-08-03T00:00:00Z
baseline_commit: abc123
staleness_policy: max_age_days=7,max_commits=20
refresh_pending:
---

# Project State

## Arquitetura
<!-- state:derived hash:000000000000 status:unavailable -->

pending

## Docs
<!-- state:curated -->

No docs.
`)
	var stdout, stderr bytes.Buffer
	if code := assessDiscover(root, []string{"--component", "service", "--json", "--update-state"}, &stdout, &stderr, cliLocale("en")); code != 0 {
		t.Fatalf("assessDiscover exit=%d stderr=%s", code, stderr.String())
	}
	var states []posemodel.ComponentDiscoveryState
	if err := json.Unmarshal(stdout.Bytes(), &states); err != nil || len(states) != 1 {
		t.Fatalf("JSON state = %#v err=%v output=%s", states, err, stdout.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	fm, body := posemodel.SplitFrontmatter(string(raw))
	state, err := posemodel.ParseProjectState(fm, body)
	if err != nil {
		t.Fatal(err)
	}
	if state.Tampered {
		t.Fatalf("updated derived hash is stale: %s", raw)
	}
	lower := strings.ToLower(string(raw))
	for _, expected := range []string{"linguagens: go", "todos=1", "componentes: total=1", "producao=3"} {
		if !strings.Contains(lower, expected) {
			t.Fatalf("project state missing %q: %s", expected, raw)
		}
	}
	for _, forbidden := range []string{"graphforge", "conductor", "harness", "portal"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("project state leaked %q: %s", forbidden, raw)
		}
	}
}

func TestAssessDiscoverRejectsEscapingComponent(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := assessDiscover(root, []string{"--component", "../outside"}, &stdout, &stderr, cliLocale("en")); code == 0 {
		t.Fatalf("escaping path accepted: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "escapes the project root") {
		t.Fatalf("unexpected error: %s", stderr.String())
	}
}
