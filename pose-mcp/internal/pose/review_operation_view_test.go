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
	mu        sync.Mutex
	reads     int
	specReads int
	fail      string
}

func (f *reviewViewCountingFiles) ReadFile(path string) ([]byte, error) {
	if strings.Contains(filepath.ToSlash(path), "/.pose/specs/") {
		f.mu.Lock()
		f.specReads++
		f.mu.Unlock()
	}
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

func TestReviewOperationViewSpecCatalogueReuseAndBodiesStayLive(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 6; i++ {
		governanceWrite(t, root, fmt.Sprintf("sample-%d.md", i), fmt.Sprintf("---\nslug: sample-%d\nstatus: draft\ncomponents: api\ndepends_on: other\n---\n\n# Sample\n\nbody %d\n", i, i))
	}
	files := &reviewViewCountingFiles{Files: NativeFiles()}
	store := Store{Root: root, FS: files}
	err := store.WithReviewOperationView(func(shared Store) {
		for repetition := 0; repetition < 2; repetition++ {
			for i := 0; i < 6; i++ {
				sp, err := shared.GetSpec(fmt.Sprintf("sample-%d", i))
				if err != nil || !strings.Contains(sp.Body, fmt.Sprintf("body %d", i)) {
					t.Fatalf("lost body: %+v %v", sp, err)
				}
			}
		}
		if files.specReads != 18 {
			t.Fatalf("spec reads=%d want initial6+12 live bodies", files.specReads)
		}
		listed, err := shared.ListSpecs("draft", "api")
		if err != nil || len(listed) != 6 {
			t.Fatalf("filter changed: %+v %v", listed, err)
		}
		listed[0].DependsOn[0] = "mutated"
		again, err := shared.ListSpecs("draft", "api")
		if err != nil || again[0].DependsOn[0] != "other" {
			t.Fatalf("metadata alias: %+v %v", again, err)
		}
		if empty, err := shared.ListSpecs("done", ""); err != nil || len(empty) != 0 {
			t.Fatalf("status filter changed: %+v %v", empty, err)
		}
		if _, err := shared.GetSpec("missing"); err == nil {
			t.Fatal("missing slug selected")
		}
		if files.specReads != 18 {
			t.Fatalf("metadata lookup reread corpus: %d", files.specReads)
		}
	})
	if err != nil || files.specReads != 36 {
		t.Fatalf("capture/live/final reads=%d want36 err=%v", files.specReads, err)
	}
}

func TestReviewOperationViewSpecCatalogueChanges(t *testing.T) {
	for _, mutation := range []string{"body", "metadata", "duplicate", "delete", "foreign", "body-ABA"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			original := "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n\nbody\n"
			governanceWrite(t, root, "sample.md", original)
			err := (Store{Root: root}).WithReviewOperationView(func(shared Store) {
				if _, err := shared.GetSpec("sample"); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "body", "body-ABA":
					governanceWrite(t, root, "sample.md", original+"changed\n")
				case "metadata":
					governanceWrite(t, root, "sample.md", strings.ReplaceAll(original, "draft", "done"))
				case "duplicate":
					governanceWrite(t, root, "2026-11-03-sample.md", original)
				case "delete":
					if err := os.Remove(filepath.Join(root, ".pose/specs/sample.md")); err != nil {
						t.Fatal(err)
					}
				case "foreign":
					outside := filepath.Join(t.TempDir(), "outside.md")
					if err := os.WriteFile(outside, []byte(original+"external sentinel\n"), 0o644); err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(root, ".pose/specs/sample.md")
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, path); err != nil {
						t.Fatal(err)
					}
					if sp, err := shared.GetSpec("sample"); err == nil || sp != nil {
						t.Fatalf("foreign spec body returned: %+v %v", sp, err)
					}
				}
				if mutation == "body-ABA" {
					if sp, err := shared.GetSpec("sample"); err == nil || sp != nil {
						t.Fatalf("changed live body accepted: %+v %v", sp, err)
					}
					governanceWrite(t, root, "sample.md", original)
				}
			})
			if err == nil || !strings.Contains(err.Error(), "review inputs changed") {
				t.Fatalf("mutation %s accepted: %v", mutation, err)
			}
		})
	}
}

func TestReviewOperationViewSpecCatalogueFallbackAndRetirement(t *testing.T) {
	root := t.TempDir()
	first := "---\nslug: first\nstatus: draft\n---\n\n# First\n\nbody\n"
	late := strings.ReplaceAll(first, "first", "late")
	governanceWrite(t, root, "first.md", first)
	governanceWrite(t, root, "late.md", late)
	store := Store{Root: root}
	for _, budget := range []string{"records", "text"} {
		t.Run(budget, func(t *testing.T) {
			view := &reviewOperationView{root: root, files: NativeFiles(), maxRecords: 100, maxRetainedText: 10000}
			if budget == "records" {
				view.maxRecords = 1
			} else {
				view.maxRetainedText = 1
			}
			err := store.withReviewOperationView(view, func(shared Store) {
				sp, err := shared.GetSpec("late")
				want, oldErr := store.GetSpec("late")
				if err != nil || oldErr != nil || !reflect.DeepEqual(sp, want) {
					t.Fatalf("fallback lost later spec: %+v %v", sp, err)
				}
			})
			if err != nil || !view.specFallback || view.specEntries != nil || view.specBodySeen != nil {
				t.Fatalf("partial metadata retained: %+v %v", view, err)
			}
		})
	}
	prior := store.withSpecSnapshot()
	if _, err := prior.ListSpecs("", ""); err != nil {
		t.Fatal(err)
	}
	var escaped Store
	err := prior.WithReviewOperationView(func(shared Store) {
		escaped = shared
		if _, err := shared.GetSpec("first"); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	governanceWrite(t, root, "first.md", strings.ReplaceAll(first, "draft", "done"))
	sp, err := escaped.GetSpec("first")
	if err != nil || sp.Status != "done" {
		t.Fatalf("retired view retained old metadata: %+v %v", sp, err)
	}
}

func TestReviewOperationViewSpecCatalogueLayoutsAndDuplicates(t *testing.T) {
	for _, layout := range []string{"flat", "nested", "split", "duplicate"} {
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
			}
			store := Store{Root: root}
			want, oldErr := store.GetSpec("sample")
			err := store.WithReviewOperationView(func(shared Store) {
				got, err := shared.GetSpec("sample")
				if viewError(err) != viewError(oldErr) || !reflect.DeepEqual(got, want) {
					t.Fatalf("layout %s changed result: %+v %v vs %+v %v", layout, got, err, want, oldErr)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
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
	wantSpec, err := store.GetSpec("backend")
	if err != nil {
		t.Fatal(err)
	}
	err = store.WithReviewOperationView(func(shared Store) {
		var wg sync.WaitGroup
		traversals := make(chan *reviewReuseTraversal, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker := shared.reviewOperation()
				traversals <- worker.reviewReuse
				sp, specErr := worker.GetSpec("backend")
				if specErr != nil || !reflect.DeepEqual(sp, wantSpec) {
					t.Errorf("parallel spec read differed: %+v %v", sp, specErr)
				}
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

func TestReviewOperationViewSpecCatalogueAvailability(t *testing.T) {
	for _, create := range []bool{false, true} {
		t.Run(fmt.Sprint(create), func(t *testing.T) {
			root := t.TempDir()
			err := (Store{Root: root}).WithReviewOperationView(func(shared Store) {
				if _, err := shared.ListSpecs("", ""); err == nil {
					t.Fatal("missing catalogue accepted")
				}
				if create {
					governanceWrite(t, root, "sample.md", "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n")
				}
			})
			if (err != nil) != create {
				t.Fatalf("availability disposition changed: create=%v err=%v", create, err)
			}
		})
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
