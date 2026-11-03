package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type governanceCountingFiles struct {
	Files
	reads int
}

func (f *governanceCountingFiles) ReadFile(path string) ([]byte, error) {
	if strings.Contains(filepath.ToSlash(path), "/.pose/specs/") {
		f.reads++
	}
	return f.Files.ReadFile(path)
}

func governanceWrite(t *testing.T, root, path, content string) {
	t.Helper()
	path = filepath.Join(root, ".pose", "specs", path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceOperationPerformanceCatalogue(t *testing.T) {
	for _, count := range []int{1, 3, 6} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root := t.TempDir()
			for i := 0; i < count; i++ {
				governanceWrite(t, root, fmt.Sprintf("sample-%d.md", i), fmt.Sprintf("---\nslug: sample-%d\nstatus: draft\ndepends_on: other\n---\n\n# Sample\n\nbody %d\n", i, i))
			}
			files := &governanceCountingFiles{Files: NativeFiles()}
			store := Store{Root: root, FS: files}
			visited := []Spec{}
			listed, err := store.VisitSpecs(func(sp Spec) error { visited = append(visited, sp); return nil })
			if err != nil || len(visited) != count || len(listed) != count {
				t.Fatalf("missing specs: %v %v", visited, err)
			}
			if files.reads != 4*count {
				t.Fatalf("catalogue read %d files, want linear %d", files.reads, 4*count)
			}
			for i, sp := range visited {
				legacy, err := (Store{Root: root}).GetSpec(sp.Slug)
				if err != nil || !reflect.DeepEqual(sp, *legacy) {
					t.Fatalf("body differs from canonical reader: %v", err)
				}
				if listed[i].Body != "" {
					t.Fatal("listing retained full bodies")
				}
				sp.DependsOn[0] = "mutated"
				if listed[i].DependsOn[0] != "other" {
					t.Fatal("callback aliases listing")
				}
			}
		})
	}
}

func TestGovernanceOperationPerformanceLayoutsAndConflicts(t *testing.T) {
	for _, layout := range []string{"flat", "nested", "split", "duplicate", "foreign"} {
		t.Run(layout, func(t *testing.T) {
			root := t.TempDir()
			text := "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n\nbody\n"
			switch layout {
			case "flat":
				governanceWrite(t, root, "sample.md", text)
			case "nested":
				governanceWrite(t, root, "2026-11-03-sample/spec.md", text)
			case "split":
				governanceWrite(t, root, "sample/INTENT.md", "# Sample\n\nintent\n")
				governanceWrite(t, root, "sample/STATUS.md", "draft\n")
			case "duplicate":
				governanceWrite(t, root, "sample.md", text)
				governanceWrite(t, root, "2026-11-03-sample.md", text)
			case "foreign":
				governanceWrite(t, root, "sample.md", text)
				outside := filepath.Join(t.TempDir(), "foreign.md")
				if err := os.WriteFile(outside, []byte(strings.ReplaceAll(text, "sample", "foreign")), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(root, ".pose/specs/foreign.md")); err != nil {
					t.Fatal(err)
				}
			}
			store := Store{Root: root}
			var got []Spec
			_, err := store.VisitSpecs(func(sp Spec) error { got = append(got, sp); return nil })
			legacy, oldErr := store.GetSpec("sample")
			if layout == "duplicate" {
				if err == nil || oldErr == nil || !strings.Contains(err.Error(), "conflicting") || len(got) != 0 {
					t.Fatalf("duplicate selected: %v %v %v", got, err, oldErr)
				}
				return
			}
			if err != nil || oldErr != nil || len(got) != 1 || !reflect.DeepEqual(got[0], *legacy) {
				t.Fatalf("layout parity failed: %v %v %v", got, err, oldErr)
			}
		})
	}
}

func TestGovernanceOperationPerformanceChangedInputs(t *testing.T) {
	for _, mutation := range []string{"body", "metadata", "duplicate"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			original := "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n\nbody\n"
			governanceWrite(t, root, "sample.md", original)
			store := Store{Root: root}
			_, err := store.VisitSpecs(func(sp Spec) error {
				switch mutation {
				case "body":
					governanceWrite(t, root, "sample.md", original+"changed\n")
				case "metadata":
					governanceWrite(t, root, "sample.md", strings.ReplaceAll(original, "draft", "done"))
				case "duplicate":
					governanceWrite(t, root, "2026-11-03-sample.md", original)
				}
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), "catalogue changed") {
				t.Fatalf("changed input accepted: %v", err)
			}
			if mutation != "duplicate" {
				var next Spec
				_, err := store.VisitSpecs(func(sp Spec) error { next = sp; return nil })
				if err != nil || (mutation == "metadata" && next.Status != "done") || (mutation == "body" && !strings.Contains(next.Body, "changed")) {
					t.Fatalf("new operation retained stale input: %+v %v", next, err)
				}
			}
		})
	}
}
