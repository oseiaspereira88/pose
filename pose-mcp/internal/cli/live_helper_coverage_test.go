package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestAssessIntegrationAndDebtPersistJSONArtifacts(t *testing.T) {
	for _, tc := range []struct{ command, report, state string }{
		{"integrate", "integrations.md", "integrations.json"},
		{"tech-debt", "technical-debt.md", "technical-debt.json"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			root := newGitRepo(t)
			writeArtifactTestFile(t, root, "service/go.mod", "module example.test/service\n\ngo 1.22\n")
			writeArtifactTestFile(t, root, "service/main.go", "package service\n// TODO: synthetic uncovered debt\n")
			var out, errOut bytes.Buffer
			if code := cmdAssess(root, []string{tc.command, "--json"}, &out, &errOut); code != 0 {
				t.Fatalf("code=%d: %s", code, &errOut)
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload) == 0 {
				t.Fatal("empty assessment")
			}
			for _, path := range []string{filepath.Join(".pose/assessments", tc.report), filepath.Join(".pose/state", tc.state)} {
				raw, err := os.ReadFile(filepath.Join(root, path))
				if err != nil || len(raw) == 0 {
					t.Fatalf("missing artifact %s: %v", path, err)
				}
			}
		})
	}
}

func TestSinceDateParserCalendarRelativeAndInvalid(t *testing.T) {
	got, err := parseSinceDate(" 2026-10-01 ")
	if err != nil || got.Format(time.DateOnly) != "2026-10-01" {
		t.Fatalf("calendar: %v %v", got, err)
	}
	before := time.Now().UTC().AddDate(0, 0, -7)
	got, err = parseSinceDate("7d")
	after := time.Now().UTC().AddDate(0, 0, -7)
	if err != nil || got.Before(before) || got.After(after) {
		t.Fatalf("relative: %v %v", got, err)
	}
	for _, value := range []string{"", "yesterday", "2026-02-30", "sevend"} {
		if _, err := parseSinceDate(value); err == nil {
			t.Errorf("accepted invalid date %q", value)
		}
	}
}

func TestSuggestDomainUsesMostSpecificModuleAndSafeFallback(t *testing.T) {
	root := t.TempDir()
	writeArtifactTestFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"service","domain":"backend"},{"path":"service/web","language":"typescript"}]}`)
	for _, tc := range []struct{ path, domain, source string }{
		{"service/main.go", "backend-go", "repo-map"},
		{"service/web/main.ts", "frontend", "repo-map"},
		{"service-other/main.go", "", "undefined"},
		{"deploy/charts/app", "k8s", "hint-path"},
	} {
		domain, source := inferSuggestDomain(root, tc.path)
		if domain != tc.domain || source != tc.source {
			t.Errorf("%s: %s/%s", tc.path, domain, source)
		}
	}
	writeArtifactTestFile(t, root, ".pose/indexes/repo-map.json", "{")
	if domain, source := inferSuggestDomain(root, "service"); domain != "" || source != "undefined" {
		t.Fatal("malformed index invented domain")
	}
	if domain, source := inferSuggestDomain(t.TempDir(), "service"); domain != "" || source != "undefined" {
		t.Fatal("missing index invented domain")
	}
}

func TestDependencyStatusAndChildrenDistinguishTerminalStates(t *testing.T) {
	root := t.TempDir()
	writeArtifactTestFile(t, root, ".pose/specs/2026-10-01-flat.md", "---\nslug: flat\nstatus: done\n---\n# Flat\n")
	writeArtifactTestFile(t, root, ".pose/specs/folder/spec.md", "---\nslug: folder\nstatus: in-progress\n---\n# Folder\n")
	specs := filepath.Join(root, ".pose/specs")
	for slug, want := range map[string]string{"flat": "done", "folder": "in-progress", "absent": ""} {
		if got := siblingSpecStatus(specs, slug); got != want {
			t.Errorf("%s: %s, want %s", slug, got, want)
		}
	}
	if hasOpenChild(nil) || hasOpenChild([]posemodel.CloseoutState{{Terminal: true}, {Terminal: true}}) {
		t.Fatal("terminal children considered open")
	}
	if !hasOpenChild([]posemodel.CloseoutState{{Terminal: true}, {Terminal: false}}) {
		t.Fatal("open child considered terminal")
	}
}
