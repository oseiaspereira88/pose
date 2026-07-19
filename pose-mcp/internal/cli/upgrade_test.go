package cli

// Upgrade compatibility lab (spec pose-upgrade-compatibility-lab): proves
// dry-run accuracy, apply idempotency and preservation of a populated,
// locale-installed, user-modified instance across the schema upgrade, plus
// explicit remediation for unsupported (newer) instances and rejection of a
// symlinked managed directory (security: block path/symlink escapes).
// Real N-minus engine/schema pairs against authenticated prior release
// binaries are exercised by tests/release/compat.sh, which is
// network-dependent and therefore not runnable in this sandbox.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const upgradeLabUserMarker = "\n<!-- upgrade-lab: user customization preserved across upgrade -->\n"

// newPopulatedUpgradeFixture installs a full instance in pt-BR (locale
// coverage), seeds a spec and a knowledge note (populated-artifact
// coverage), appends a marker to a managed file (user-modification
// coverage) and rewinds schema-version to simulate a pre-upgrade instance —
// exactly the shape R2 requires.
func newPopulatedUpgradeFixture(t *testing.T) string {
	t.Helper()
	repo := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{repo, "--locale", "pt-BR", "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install fixture exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"new-spec", "upgrade-lab-fixture"}, &out, &errB); code != 0 {
			t.Fatalf("new-spec exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"new-knowledge", "handoff", "upgrade-lab-fixture", "--owner", "@pose-maintainers"}, &out, &errB); code != 0 {
			t.Fatalf("new-knowledge exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
	})
	agentsPath := filepath.Join(repo, "AGENTS.md")
	agents, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("reading AGENTS.md fixture: %v", err)
	}
	if err := os.WriteFile(agentsPath, append(agents, []byte(upgradeLabUserMarker)...), 0o644); err != nil {
		t.Fatalf("appending user marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".pose", "schema-version"), []byte("0\n"), 0o644); err != nil {
		t.Fatalf("rewinding schema-version: %v", err)
	}
	return repo
}

// snapshotTree hashes every regular file under root (excluding .git),
// keyed by slash-separated relative path, so before/after comparisons are
// exact and independent of OS path separators.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotting %s: %v", root, err)
	}
	return out
}

func diffSnapshots(before, after map[string]string) (added, removed, changed []string) {
	for p, h := range after {
		if bh, ok := before[p]; !ok {
			added = append(added, p)
		} else if bh != h {
			changed = append(changed, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			removed = append(removed, p)
		}
	}
	return
}

func TestUpgradeDryRunIsAccurateAndNonMutating(t *testing.T) {
	fixture := newPopulatedUpgradeFixture(t)
	before := snapshotTree(t, fixture)
	inDir(t, fixture, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade", "--dry-run"}, &out, &errB); code != 0 {
			t.Fatalf("dry-run exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "v0 -> v1") || !strings.Contains(out.String(), "001-baseline") || !strings.Contains(out.String(), "DRY-RUN") {
			t.Errorf("dry-run output missing plan details: %s", out.String())
		}
	})
	after := snapshotTree(t, fixture)
	added, removed, changed := diffSnapshots(before, after)
	if len(added) != 0 || len(removed) != 0 || len(changed) != 0 {
		t.Errorf("dry-run must not mutate the tree: added=%v removed=%v changed=%v", added, removed, changed)
	}
}

func TestUpgradeApplyIsIdempotentAndPreservesInstanceContent(t *testing.T) {
	fixture := newPopulatedUpgradeFixture(t)
	before := snapshotTree(t, fixture)

	inDir(t, fixture, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code != 0 {
			t.Fatalf("apply exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "Result: SUCCESS") || !strings.Contains(out.String(), "schema v1") {
			t.Errorf("apply output missing success marker: %s", out.String())
		}
	})

	sv, err := os.ReadFile(filepath.Join(fixture, ".pose", "schema-version"))
	if err != nil || strings.TrimSpace(string(sv)) != "1" {
		t.Fatalf("schema-version not bumped to 1: content=%q err=%v", sv, err)
	}

	afterApply := snapshotTree(t, fixture)
	added, removed, changed := diffSnapshots(before, afterApply)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("apply must not add/remove files on an already-populated instance: added=%v removed=%v", added, removed)
	}
	if len(changed) != 1 || changed[0] != ".pose/schema-version" {
		t.Errorf("apply on a fully-populated instance must change only schema-version, got: %v", changed)
	}

	agents, err := os.ReadFile(filepath.Join(fixture, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "upgrade-lab: user customization") {
		t.Error("user-modified AGENTS.md content was not preserved across upgrade")
	}
	if _, err := os.Stat(filepath.Join(fixture, ".pose", "specs", "upgrade-lab-fixture", "spec.md")); err != nil {
		t.Errorf("populated spec artifact was not preserved: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(fixture, ".pose", "knowledge", "*upgrade-lab-fixture*.md"))
	if err != nil || len(matches) != 1 {
		t.Errorf("populated knowledge artifact was not preserved: matches=%v err=%v", matches, err)
	}

	// Reapply must be idempotent: no further mutation, explicit no-op message.
	inDir(t, fixture, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code != 0 {
			t.Fatalf("reapply exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "already at schema v1") {
			t.Errorf("reapply did not report a no-op: %s", out.String())
		}
	})
	afterReapply := snapshotTree(t, fixture)
	_, _, changedAgain := diffSnapshots(afterApply, afterReapply)
	if len(changedAgain) != 0 {
		t.Errorf("reapply must be a strict no-op, tree changed: %v", changedAgain)
	}
}

func TestUpgradeRejectsNewerInstanceWithExplicitRemediation(t *testing.T) {
	fixture := newPopulatedUpgradeFixture(t)
	if err := os.WriteFile(filepath.Join(fixture, ".pose", "schema-version"), []byte("99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, fixture)
	inDir(t, fixture, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code == 0 {
			t.Fatalf("expected non-zero exit for a newer-than-engine instance, out=%s", out.String())
		} else if !strings.Contains(errB.String(), "newer than engine") || !strings.Contains(errB.String(), "downgrade is unsupported") {
			t.Errorf("expected explicit remediation diagnostic, got: %s", errB.String())
		}
	})
	after := snapshotTree(t, fixture)
	added, removed, changed := diffSnapshots(before, after)
	if len(added) != 0 || len(removed) != 0 || len(changed) != 0 {
		t.Errorf("rejected upgrade must not partially mutate the instance: added=%v removed=%v changed=%v", added, removed, changed)
	}
}

func TestUpgradeBlocksManagedDirSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows CI runners")
	}
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".pose", "schema-version"), []byte("0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo, ".pose", "roadmaps")); err != nil {
		t.Fatal(err)
	}

	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"upgrade"}, &out, &errB); code == 0 {
			t.Fatalf("expected non-zero exit when a managed directory is a symlink, out=%s", out.String())
		} else if !strings.Contains(errB.String(), "refusing to follow symlink") {
			t.Errorf("expected a symlink-refusal diagnostic, got: %s", errB.String())
		}
	})

	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("upgrade must not write through the symlinked managed directory, found: %v", entries)
	}
	sv, err := os.ReadFile(filepath.Join(repo, ".pose", "schema-version"))
	if err != nil || strings.TrimSpace(string(sv)) != "0" {
		t.Errorf("schema-version must not advance on a blocked (partial) upgrade: content=%q err=%v", sv, err)
	}
}
