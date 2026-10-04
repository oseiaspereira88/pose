package pose

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec pose-blocked-semantics-alignment: blocked is a non-terminal
// operational condition whose cause is reported when the engine can tell.
func TestBlockedReadinessIsNonTerminalAndExplainsItsCause(t *testing.T) {
	s := readinessStore(t)
	write := func(rel, content string) {
		path := filepath.Join(s.Root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/2026-10-04-waits.md", "---\nslug: waits\nstatus: blocked\ndepends_on: pending\n---\n\n# Spec: waits\n")
	write(".pose/specs/2026-10-04-unexplained.md", "---\nslug: unexplained\nstatus: blocked\n---\n\n# Spec: unexplained\n")
	write(".pose/specs/2026-10-04-unblockable.md", "---\nslug: unblockable\nstatus: blocked\ndepends_on: base\n---\n\n# Spec: unblockable\n")

	waits, err := s.SpecReadiness("waits")
	if err != nil {
		t.Fatal(err)
	}
	if waits.Ready || waits.Terminal || waits.Cause != ReadinessCauseDependency || len(waits.WaitingOn) != 1 || waits.WaitingOn[0].Ref != "pending" {
		t.Fatalf("blocked spec with an unmet prerequisite: %+v", waits)
	}
	unexplained, _ := s.SpecReadiness("unexplained")
	if unexplained.Ready || unexplained.Terminal || unexplained.Cause != ReadinessCauseUnknown {
		t.Fatalf("blocked spec without a recorded cause: %+v", unexplained)
	}
	// Prerequisites satisfied does not unblock it: the legacy Ready meaning
	// of a blocked spec is unchanged, and the cause is honestly unknown.
	unblockable, _ := s.SpecReadiness("unblockable")
	if unblockable.Ready || unblockable.Cause != ReadinessCauseUnknown {
		t.Fatalf("a blocked spec with met prerequisites became ready or invented a cause: %+v", unblockable)
	}
	done, _ := s.SpecReadiness("base")
	if !done.Terminal || done.Cause != "" {
		t.Fatalf("a done spec is terminal and has no blocked cause: %+v", done)
	}
}
