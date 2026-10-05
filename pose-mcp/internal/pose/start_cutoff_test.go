package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-atomic-start-adoption-cutoff: a spec already in progress before
// the adoption date is reported as legacy-unbaselined without blocking; one
// that entered progress on or after it without a start still needs
// reconciliation, and an unparsable date is refused.
func TestAtomicStartCutoffReportsOlderSpecsWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec := func(slug, created string) string {
		return "---\nslug: " + slug + "\nstatus: in-progress\ncreated_at: " + created + "\n---\n\n# Spec: " + slug + "\n\n## 2. Requirements\n\n- R1: x.\n"
	}
	write(".pose/specs/2026-09-01-old.md", spec("old", "2026-09-01"))
	write(".pose/specs/2026-10-06-new.md", spec("new", "2026-10-06"))
	policy := func(extra string) {
		write(".pose/policy/review.json", `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"},"atomic_start_version":1`+extra+`}`)
	}
	s := Store{Root: root}

	policy(`,"atomic_start_adopted_at":"2026-10-05"`)
	old, err := s.GetStartStatus("old")
	if err != nil {
		t.Fatal(err)
	}
	if len(old.Reconciliation) != 0 || !strings.Contains(strings.Join(old.Notes, " "), "before atomic start was adopted") {
		t.Fatalf("a spec older than the cutoff is blocked or unexplained: %+v", old)
	}
	if len(old.Nodes) == 0 || old.Nodes[0].Origin != "legacy-unbaselined" {
		t.Fatalf("the older spec gained a baseline it never had: %+v", old.Nodes)
	}
	fresh, _ := s.GetStartStatus("new")
	if len(fresh.Reconciliation) == 0 {
		t.Fatal("a spec created after the cutoff without a start escaped reconciliation")
	}

	policy(``)
	legacy, _ := s.GetStartStatus("old")
	if len(legacy.Reconciliation) == 0 {
		t.Fatal("without a cutoff the adopted capability must keep applying to every spec")
	}

	policy(`,"atomic_start_adopted_at":"05/10/2026"`)
	if _, err := s.GetReviewPolicy(); err == nil {
		t.Fatal("an unparsable cutoff was accepted")
	}
}
