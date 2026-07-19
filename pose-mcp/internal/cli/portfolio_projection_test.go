package cli

// Cross-repository portfolio projections (spec pose-cross-repo-portfolio):
// stable xref identities resolved only against authorized projects (R1,
// Security), blocked/stale/conflict paths explained explicitly (R2),
// ownership/criticality exposed without fabricating capacity (R3), and
// projections are revisioned with tombstones for disappeared artifacts.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// portfolioFixtureProject creates a project directory (real git repo +
// .pose/specs) under base/name and returns its absolute root. A real repo
// is needed so the CLI end-to-end test (which resolves root via `git
// rev-parse`) works the same as the white-box tests that pass root
// explicitly.
func portfolioFixtureProject(t *testing.T, base, name string) string {
	t.Helper()
	root := filepath.Join(base, name)
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func writePortfolioSpec(t *testing.T, projectRoot, slug, status, dependsOn string) {
	t.Helper()
	dir := filepath.Join(projectRoot, ".pose", "specs", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fm := "---\nslug: " + slug + "\nstatus: " + status + "\ncreated_at: 2026-06-01\n"
	if dependsOn != "" {
		fm += "depends_on: " + dependsOn + "\n"
	}
	fm += "---\n\n# Spec: " + slug + "\n"
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(fm), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePortfolioModuleMetadata(t *testing.T, projectRoot, owner, criticality string) {
	t.Helper()
	dir := filepath.Join(projectRoot, ".pose", "indexes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"defaults":{"owner":"` + owner + `","criticality":"` + criticality + `"},"modules":{}}`
	if err := os.WriteFile(filepath.Join(dir, "module-metadata.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupTwoProjectPortfolio(t *testing.T) (selfRoot, otherRoot, projectsDir string) {
	t.Helper()
	base := t.TempDir()
	selfRoot = portfolioFixtureProject(t, base, "self-project")
	otherRoot = portfolioFixtureProject(t, base, "other-project")
	t.Setenv("HARNE8_PROJECTS_DIR", base)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "self-project")
	return selfRoot, otherRoot, base
}

func TestPortfolioProjectionResolvesAuthorizedXref(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "needs-other", "draft", "xref:other-project/upstream-done")
	writePortfolioSpec(t, otherRoot, "upstream-done", "done", "")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	var projection portfolioProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	spec := findProjectedSpec(t, projection, "self-project", "needs-other")
	if len(spec.XrefsOut) != 1 {
		t.Fatalf("expected 1 xref, got %+v", spec.XrefsOut)
	}
	x := spec.XrefsOut[0]
	if !x.Resolved || x.TargetStatus != "done" || x.Blocking {
		t.Errorf("expected resolved, non-blocking xref to a done spec: %+v", x)
	}
}

func TestPortfolioProjectionExplainsBlockedXref(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "needs-other", "draft", "xref:other-project/upstream-wip")
	writePortfolioSpec(t, otherRoot, "upstream-wip", "in-progress", "")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	var projection portfolioProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	spec := findProjectedSpec(t, projection, "self-project", "needs-other")
	if !spec.XrefsOut[0].Resolved || !spec.XrefsOut[0].Blocking {
		t.Errorf("expected a resolved but blocking xref (target not done): %+v", spec.XrefsOut[0])
	}
}

func TestPortfolioProjectionRejectsUnauthorizedProject(t *testing.T) {
	selfRoot, _, _ := setupTwoProjectPortfolio(t)
	// A project that exists on disk but is NOT under the authorized
	// projects dir and NOT in POSE_PROJECT_ROOTS.
	rogue := portfolioFixtureProject(t, t.TempDir(), "rogue-project")
	writePortfolioSpec(t, rogue, "secret-spec", "done", "")
	writePortfolioSpec(t, selfRoot, "needs-rogue", "draft", "xref:rogue-project/secret-spec")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	var projection portfolioProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	spec := findProjectedSpec(t, projection, "self-project", "needs-rogue")
	x := spec.XrefsOut[0]
	if x.Resolved || x.Reason != "unauthorized-project" {
		t.Errorf("expected an unauthorized-project rejection, got %+v", x)
	}
	if strings.Contains(out.String(), "rogue-project") == false {
		t.Error("the ref itself should still be echoed back (it's the caller's own input)")
	}
}

func TestPortfolioProjectionExplainsUnknownSpec(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	_ = otherRoot
	writePortfolioSpec(t, selfRoot, "needs-ghost", "draft", "xref:other-project/does-not-exist")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	var projection portfolioProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	spec := findProjectedSpec(t, projection, "self-project", "needs-ghost")
	if spec.XrefsOut[0].Resolved || spec.XrefsOut[0].Reason != "unknown-spec" {
		t.Errorf("expected unknown-spec, got %+v", spec.XrefsOut[0])
	}
}

func TestPortfolioProjectionMarksStaleSource(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "needs-other", "draft", "xref:other-project/upstream-done")
	writePortfolioSpec(t, otherRoot, "upstream-done", "done", "")
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(otherRoot, ".pose", "specs", "upstream-done", "spec.md"), oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json", "--max-staleness-days", "7"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	var projection portfolioProjection
	if err := json.Unmarshal(out.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	other := findProjectedSpec(t, projection, "other-project", "upstream-done")
	if !other.Stale {
		t.Error("expected the other-project spec to be marked stale")
	}
	self := findProjectedSpec(t, projection, "self-project", "needs-other")
	if self.XrefsOut[0].Reason != "stale-source" {
		t.Errorf("expected the xref to note the stale source, got %+v", self.XrefsOut[0])
	}
}

func TestPortfolioProjectionOwnershipCriticalityNoFabricatedCapacity(t *testing.T) {
	selfRoot, _, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "owned-spec", "draft", "")
	writePortfolioModuleMetadata(t, selfRoot, "@platform-team", "high")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	spec := findProjectedSpec(t, unmarshalProjection(t, out.Bytes()), "self-project", "owned-spec")
	if spec.Owner != "@platform-team" || spec.Criticality != "high" {
		t.Errorf("expected owner/criticality from module-metadata.json, got %+v", spec)
	}
	for _, forbidden := range []string{"capacity", "velocity", "eta", "estimated"} {
		if strings.Contains(strings.ToLower(out.String()), forbidden) {
			t.Errorf("projection must never fabricate a capacity-shaped metric, found %q in output", forbidden)
		}
	}
}

func TestPortfolioProjectionNeverLeaksFilesystemPaths(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "s", "draft", "")
	writePortfolioSpec(t, otherRoot, "o", "draft", "")

	var out, errB bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d err=%s", code, errB.String())
	}
	if strings.Contains(out.String(), selfRoot) || strings.Contains(out.String(), otherRoot) {
		t.Error("projection output must never leak an absolute filesystem path, only logical project_id")
	}
}

func TestPortfolioProjectionTombstonesRemovedSpecs(t *testing.T) {
	selfRoot, _, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "temporary-spec", "draft", "")

	var out1, errB1 bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out1, &errB1); code != 0 {
		t.Fatalf("first run exit=%d err=%s", code, errB1.String())
	}

	if err := os.RemoveAll(filepath.Join(selfRoot, ".pose", "specs", "temporary-spec")); err != nil {
		t.Fatal(err)
	}
	var out2, errB2 bytes.Buffer
	if code := cmdPortfolioProjection(selfRoot, []string{"--json"}, &out2, &errB2); code != 0 {
		t.Fatalf("second run exit=%d err=%s", code, errB2.String())
	}
	projection := unmarshalProjection(t, out2.Bytes())
	found := false
	for _, tomb := range projection.Tombstones {
		if tomb.Project == "self-project" && tomb.Slug == "temporary-spec" {
			found = true
			if tomb.RemovedAt == "" {
				t.Error("tombstone missing removed_at")
			}
		}
	}
	if !found {
		t.Errorf("expected a tombstone for the removed spec, got %+v", projection.Tombstones)
	}
}

func TestXrefDependsOnPassesReadyCheck(t *testing.T) {
	root := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, installErr.String())
	}
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"new-spec", "xref-consumer"}, &out, &errB); code != 0 {
			t.Fatalf("new-spec exit=%d err=%s", code, errB.String())
		}
		specPath := filepath.Join(root, ".pose", "specs", "xref-consumer", "spec.md")
		raw, err := os.ReadFile(specPath)
		if err != nil {
			t.Fatal(err)
		}
		content := strings.Replace(string(raw), "depends_on:", "depends_on: xref:other-project/some-spec", 1)
		content = strings.Replace(content, "## 1. Intent", "## 1. Intent\n\nSomething concrete.\n", 1)
		content = strings.Replace(content, "- R1:", "- R1: Do the thing.", 1)
		content = strings.Replace(content, "## 3. Technical Plan", "## 3. Technical Plan\n\nSomething concrete.\n", 1)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		errB.Reset()
		code := Main([]string{"lint-spec", "xref-consumer", "--ready-check"}, &out, &errB)
		if strings.Contains(errB.String(), "invalid depends_on reference") {
			t.Fatalf("xref: reference must be accepted as valid depends_on syntax: exit=%d err=%s", code, errB.String())
		}
	})
}

func TestPortfolioProjectionCLIEndToEnd(t *testing.T) {
	selfRoot, otherRoot, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, selfRoot, "needs-other", "draft", "xref:other-project/upstream-done")
	writePortfolioSpec(t, otherRoot, "upstream-done", "done", "")
	if err := os.MkdirAll(filepath.Join(selfRoot, ".pose", "reports"), 0o755); err != nil {
		t.Fatal(err)
	}
	inDir(t, selfRoot, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"portfolio-projection"}, &out, &errB); code != 0 {
			t.Fatalf("exit=%d err=%s", code, errB.String())
		}
		if !strings.Contains(out.String(), "projects=2") {
			t.Errorf("expected 2 authorized projects in the summary: %s", out.String())
		}
	})
	if _, err := os.Stat(portfolioProjectionPath(selfRoot)); err != nil {
		t.Errorf("expected the projection to be persisted: %v", err)
	}
}

func findProjectedSpec(t *testing.T, projection portfolioProjection, project, slug string) projectedSpec {
	t.Helper()
	for _, s := range projection.Specs {
		if s.Project == project && s.Slug == slug {
			return s
		}
	}
	t.Fatalf("spec %s/%s not found in projection: %+v", project, slug, projection.Specs)
	return projectedSpec{}
}

func unmarshalProjection(t *testing.T, raw []byte) portfolioProjection {
	t.Helper()
	var p portfolioProjection
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, raw)
	}
	return p
}
