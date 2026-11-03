package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type reviewViewCountingFiles struct {
	Files
	mu    sync.Mutex
	reads int
	fail  string
}

func (f *reviewViewCountingFiles) ReadFile(path string) ([]byte, error) {
	if strings.Contains(filepath.ToSlash(path), "/.pose/review-bundles/") {
		f.mu.Lock()
		f.reads++
		f.mu.Unlock()
	}
	if filepath.Base(path) == f.fail {
		return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrPermission}
	}
	return f.Files.ReadFile(path)
}

func viewWrite(t *testing.T, root, name, text string) {
	t.Helper()
	writeReviewFixture(t, root, ".pose/review-bundles/"+name, text)
}

func viewError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func TestReviewOperationViewCoverageAndStrictLoadParity(t *testing.T) {
	root := t.TempDir()
	records := []string{
		`{"schema_version":999,"payload":{"scope":{"ref":"spec:other"}},"unknown":"ignored only by another scope"}`,
		`{"payload":{"scope":{"ref":"spec:owned"}},"broken":`,
		`{"payload":{"scope":{"ref":"spec:owned","ref":"spec:owned"}}}`,
		`{"payload":{}}`,
		`{"payload":{"scope":{"ref":"spec:owned"}}} {}`,
		`{"payload":{"scope":{"ref":"spec:owned"},"scope":{"ref":"spec:other"}}}`,
		`{"payload":{"scope":{"re\u0066":"spec:other"}}}`,
		`{"payload":{"scope":{"ref":"spec:other"}}}`,
	}
	for i, raw := range records {
		viewWrite(t, root, fmt.Sprintf("rvb-%016x.json", i), raw)
	}
	fail := "rvb-ffffffffffffffff.json"
	viewWrite(t, root, fail, `{}`)
	files := &reviewViewCountingFiles{Files: NativeFiles(), fail: fail}
	store := Store{Root: root, FS: files}
	scopes := []string{"spec:absent", "spec:owned", "spec:other", ""}
	wants := make([]ReviewBundlesRead, len(scopes))
	errors := make([]string, len(scopes))
	for i, scope := range scopes {
		var err error
		wants[i], err = store.ReadReviewBundles(scope)
		errors[i] = viewError(err)
	}
	err := store.WithReviewOperationView(func(shared Store) {
		for repetition := 0; repetition < 2; repetition++ {
			for i, scope := range scopes {
				got, err := shared.ReadReviewBundles(scope)
				if viewError(err) != errors[i] || !reflect.DeepEqual(got, wants[i]) {
					t.Fatalf("scope %q changed read: got=%+v err=%v want=%+v err=%s", scope, got, err, wants[i], errors[i])
				}
				if len(got.Coverage.Losses) > 0 {
					got.Coverage.Losses[0].Detail = "caller mutation"
				}
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReviewOperationViewCapturesEachHeaderOnce(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 6; i++ {
		viewWrite(t, root, fmt.Sprintf("rvb-%016x.json", i), `{"payload":{"scope":{"ref":"spec:other"}}}`)
	}
	files := &reviewViewCountingFiles{Files: NativeFiles()}
	store := Store{Root: root, FS: files}
	err := store.WithReviewOperationView(func(shared Store) {
		for i := 0; i < 4; i++ {
			got, err := shared.ReadReviewBundles("spec:absent")
			if err != nil || len(got.Bundles) != 0 || len(got.Coverage.Losses) != 0 {
				t.Fatalf("lost unrelated coverage: %+v %v", got, err)
			}
		}
		if files.reads != 6 {
			t.Fatalf("headers read %d times, want 6", files.reads)
		}
	})
	if err != nil || files.reads != 12 {
		t.Fatalf("capture/final audit reads=%d err=%v", files.reads, err)
	}
}

func TestReviewOperationViewBudgetFallback(t *testing.T) {
	root := t.TempDir()
	viewWrite(t, root, "rvb-0000000000000000.json", `{"payload":{"scope":{"ref":"spec:other"}}}`)
	viewWrite(t, root, "rvb-ffffffffffffffff.json", `{"payload":{"scope":{"ref":"spec:owned"}},"broken":`)
	store := Store{Root: root}
	want, err := store.ReadReviewBundles("spec:owned")
	if err != nil || len(want.Coverage.Losses) != 1 {
		t.Fatalf("bad fixture: %+v %v", want, err)
	}
	for _, budget := range []string{"records", "input", "text"} {
		t.Run(budget, func(t *testing.T) {
			view := &reviewOperationView{root: root, files: NativeFiles(), maxRecords: 100, maxInputBytes: 100000, maxRetainedText: 100000}
			switch budget {
			case "records":
				view.maxRecords = 1
			case "input":
				view.maxInputBytes = 1
			case "text":
				view.maxRetainedText = 1
			}
			err := store.withReviewOperationView(view, func(shared Store) {
				for i := 0; i < 2; i++ {
					got, err := shared.ReadReviewBundles("spec:owned")
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("budget dropped later loss: %+v %v", got, err)
					}
				}
			})
			if err != nil || !view.fallback || view.headers != nil {
				t.Fatalf("partial cache retained: fallback=%v headers=%v err=%v", view.fallback, view.headers, err)
			}
		})
	}
}

func TestReviewOperationViewChangedInputs(t *testing.T) {
	for _, mutation := range []string{"rewrite", "append", "delete", "directory", "initially-absent"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			name := "rvb-0000000000000000.json"
			original := `{"payload":{"scope":{"ref":"spec:other"}}}`
			if mutation != "initially-absent" {
				viewWrite(t, root, name, original)
			}
			store := Store{Root: root}
			err := store.WithReviewOperationView(func(shared Store) {
				if _, err := shared.ReadReviewBundles("spec:absent"); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "rewrite":
					path := filepath.Join(root, ".pose/review-bundles", name)
					info, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					viewWrite(t, root, name, strings.ReplaceAll(original, "other", "owned"))
					if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
						t.Fatal(err)
					}
				case "append", "initially-absent":
					viewWrite(t, root, "rvb-ffffffffffffffff.json", original)
				case "delete":
					if err := os.Remove(filepath.Join(root, ".pose/review-bundles", name)); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.RemoveAll(filepath.Join(root, ".pose/review-bundles")); err != nil {
						t.Fatal(err)
					}
				}
			})
			if err == nil || !strings.Contains(err.Error(), "review inputs changed") {
				t.Fatalf("mutation accepted: %v", err)
			}
		})
	}
}

func TestReviewOperationViewBindingAndRetirement(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	name := "rvb-0000000000000000.json"
	viewWrite(t, root, name, `{"payload":{"scope":{"ref":"spec:other"}}}`)
	viewWrite(t, other, name, `{"payload":{"scope":{"ref":"spec:owned"}},"broken":`)
	store := Store{Root: root}
	var escaped Store
	err := store.WithReviewOperationView(func(shared Store) {
		escaped = shared
		if _, err := shared.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
		foreign := shared
		foreign.Root = other
		got, err := foreign.ReadReviewBundles("spec:owned")
		if err != nil || len(got.Coverage.Losses) != 1 {
			t.Fatalf("borrowed another root: %+v %v", got, err)
		}
		preview := NewPreviewFiles(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
		if err := preview.WriteFile(filepath.Join(root, ".pose/review-bundles", name), []byte(`{"payload":{}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		staged := shared
		staged.FS = preview
		got, err = staged.ReadReviewBundles("spec:absent")
		if err != nil || len(got.Coverage.Losses) != 1 {
			t.Fatalf("borrowed another filesystem: %+v %v", got, err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	viewWrite(t, root, name, `{"payload":{}}`)
	got, err := escaped.ReadReviewBundles("spec:absent")
	if err != nil || len(got.Coverage.Losses) != 1 {
		t.Fatalf("escaped store retained observations: %+v %v", got, err)
	}
}

func TestReviewOperationViewParallelTraversalIsolation(t *testing.T) {
	_, store := reviewBundleFixture(t)
	if _, err := store.SealReviewBundle("spec:backend", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	want, err := store.ReadReviewBundles("spec:backend")
	if err != nil || len(want.Bundles) != 1 {
		t.Fatalf("invalid fixture: %+v %v", want, err)
	}
	store.reviewReuse = newReviewReuseTraversal()
	err = store.WithReviewOperationView(func(shared Store) {
		var wg sync.WaitGroup
		traversals := make(chan *reviewReuseTraversal, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker := shared.reviewOperation()
				traversals <- worker.reviewReuse
				got, err := worker.ReadReviewBundles("spec:backend")
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("parallel read differed: %+v %v", got, err)
				}
			}()
		}
		wg.Wait()
		close(traversals)
		seen := map[*reviewReuseTraversal]bool{}
		for traversal := range traversals {
			if traversal == nil || seen[traversal] {
				t.Error("workers shared traversal state")
			}
			seen[traversal] = true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

type nonComparableReviewFiles struct {
	Files
	marker []string
}

type reviewFilesEnvelope struct{ Files }

func TestReviewOperationViewContextBindingAndUnkeyedFiles(t *testing.T) {
	root := t.TempDir()
	viewWrite(t, root, "rvb-0000000000000000.json", `{"payload":{"scope":{"ref":"spec:other"}}}`)
	files := &reviewViewCountingFiles{Files: NativeFiles()}
	store := Store{Root: root, FS: files}
	err := store.WithReviewOperationView(func(shared Store) {
		if _, err := shared.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
		if files.reads != 1 {
			t.Fatalf("initial capture reads=%d", files.reads)
		}
		project := shared
		project.FederatedProjectID = "different"
		if _, err := project.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
		resolver := shared
		resolver.FederatedResolver = &ArtifactResolver{}
		if _, err := resolver.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
		if files.reads != 3 {
			t.Fatalf("changed context reused header: reads=%d", files.reads)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	store.FS = nonComparableReviewFiles{Files: files, marker: []string{"unkeyed"}}
	err = store.WithReviewOperationView(func(shared Store) {
		if shared.reviewView != nil {
			t.Fatal("noncomparable filesystem was assigned a cache identity")
		}
		if _, err := shared.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	// The struct type is comparable, but its embedded interface holds a value
	// with a slice. Comparing interfaces must not panic or borrow a cache key.
	store.FS = reviewFilesEnvelope{Files: nonComparableReviewFiles{Files: files, marker: []string{"nested"}}}
	err = store.WithReviewOperationView(func(shared Store) {
		if shared.reviewView != nil {
			t.Fatal("nested noncomparable filesystem assigned a cache key")
		}
		if _, err := shared.ReadReviewBundles("spec:absent"); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}
