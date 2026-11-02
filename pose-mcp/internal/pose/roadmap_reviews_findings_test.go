package pose

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPremiseOutlivesItsDelivery(t *testing.T) {
	probe, _ := premiseFixture(t, "doc:api/payments.proto")
	pin := digestPrefix(t, probe, "api/payments.proto")
	_, s := premiseFixture(t, "doc:api/payments.proto@"+pin)
	if err := os.WriteFile(filepath.Join(s.Root, "api/payments.proto"), []byte("syntax = \"proto3\";\nmessage Charge {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := premiseObligationsFor(t, s); len(got) != 1 {
		t.Fatalf("the fixture does not signal: %+v", got)
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range specs {
		raw, err := os.ReadFile(sp.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sp.Path, []byte(ReplaceFrontmatterValue(string(raw), "status", "done")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := premiseObligationsFor(t, s)
	if len(got) != 1 || got[0].Effects[0].Mode != EffectAdvisory {
		t.Fatalf("a premise stopped being watched at delivery: %+v", got)
	}
}

func TestAnEmptyGitDirectoryIsNoRepository(t *testing.T) {
	base := t.TempDir()
	below := filepath.Join(base, "fixture")
	if err := os.MkdirAll(below, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(base, ".git")
	if err := os.Mkdir(marker, 0o755); err != nil {
		t.Fatal(err)
	}
	if hasGitMetadata(below) || !GitNotARepository(below) {
		t.Fatalf("an empty directory read as a repository: metadata=%v", hasGitMetadata(below))
	}
	// Anything in it is a repository git may be failing to read, and stays unknown.
	if err := os.WriteFile(filepath.Join(marker, "HEAD"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasGitMetadata(below) || GitNotARepository(below) {
		t.Fatal("a broken repository read as none")
	}
	if err := os.RemoveAll(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasGitMetadata(below) || GitNotARepository(below) {
		t.Fatal("a gitfile read as no repository")
	}
}

func TestOnlyAMissingLinkFallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	linkFile = func(string, string) error {
		return &os.LinkError{Op: "link", Err: errors.New("no space left on device")}
	}
	t.Cleanup(func() { linkFile = os.Link })
	if err := PublishFile(path, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err == nil {
		t.Fatal("a failed link fell back to an in-place write")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a record was left by a failed link")
	}
}

func TestACutWriteLeavesNoRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	PublishSyncHook = func(kind, _ string) error {
		if kind == "exclusive" {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { PublishSyncHook = nil })
	if err := publishExclusive(path, []byte("whole\n"), 0o644, PublishAuthoritative); err == nil {
		t.Fatal("the fixture did not fail")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a cut write left a record that refuses every retry")
	}
	PublishSyncHook = nil
	if err := publishExclusive(path, []byte("whole\n"), 0o644, PublishAuthoritative); err != nil {
		t.Fatalf("a retry was refused: %v", err)
	}
}

func TestTheCleanupPlanHoldsBothCorpora(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "reports", "2026-11-02-pose-v7-legacy-cleanup-both-corpora.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"| Class | pose-dist", "Harne8", "before and after the run"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the report does not hold %q", want)
		}
	}
}
