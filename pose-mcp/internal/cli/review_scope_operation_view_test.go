package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

type d7CLICountFiles struct {
	pose.Files
	mu    sync.Mutex
	opens map[string]int
}

func (f *d7CLICountFiles) Open(path string) (fs.File, error) {
	f.mu.Lock()
	f.opens[filepath.Clean(path)]++
	f.mu.Unlock()
	return f.Files.Open(path)
}

func d7CLIApprovedScopes(t *testing.T) string {
	t.Helper()
	root := closeoutCLIFixture(t)
	original, err := os.ReadFile(filepath.Join(root, ".pose/specs/alpha/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeCloseoutCLIFile(t, root, ".pose/specs/beta/spec.md", strings.ReplaceAll(string(original), "alpha", "beta"))
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "two closeout subjects")
	for _, slug := range []string{"alpha", "beta"} {
		var out, errOut bytes.Buffer
		if code := cmdReviewRecord(root, []string{"spec:" + slug, "--reviewer", "agent:d7-fixture", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut); code != 0 {
			t.Fatalf("review %s: %d %s%s", slug, code, &out, &errOut)
		}
		out.Reset()
		errOut.Reset()
		if code := cmdClose(root, []string{"spec:" + slug}, &out, &errOut); code != 0 {
			t.Fatalf("close %s: %d %s%s", slug, code, &out, &errOut)
		}
	}
	return root
}

func d7CLIRunCloseoutGate(t *testing.T, root string, files pose.Files) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	checker := &nativeChecker{root: root, files: files, mode: "strict", stdout: &out, out: render(&out, &errOut)}
	checker.checkReviewCloseout()
	return checker.errors, out.String() + errOut.String()
}

func TestGovernanceReadViewScopeGlobalGateSerialAndParallelPreserveApprovals(t *testing.T) {
	for _, workers := range []string{"1", "4"} {
		t.Run(workers, func(t *testing.T) {
			t.Setenv("POSE_CHECK_WORKERS", workers)
			root := d7CLIApprovedScopes(t)
			// A valid unrelated header exercises shared immutable classification. It is
			// not used as an approval or a synthetic attestation of either real scope.
			header := filepath.Join(root, ".pose/review-bundles/unrelated.json")
			writeCloseoutCLIFile(t, root, ".pose/review-bundles/unrelated.json", `{"payload":{"scope":{"ref":"spec:unrelated"}}}`)
			files := &d7CLICountFiles{Files: pose.NativeFiles(), opens: map[string]int{}}
			count, output := d7CLIRunCloseoutGate(t, root, files)
			if count != 0 {
				t.Fatalf("approved scopes failed with %s workers: %s", workers, output)
			}
			files.mu.Lock()
			reads := files.opens[header]
			files.mu.Unlock()
			if reads != 2 {
				t.Fatalf("shared header opened %d times; want one capture plus one final audit", reads)
			}
		})
	}
}

func TestGovernanceReadViewScopeGlobalGateDoesNotHideUnreviewedSibling(t *testing.T) {
	root := d7CLIApprovedScopes(t)
	writeCloseoutCLIFile(t, root, ".pose/specs/unreviewed/spec.md", "---\nslug: unreviewed\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-13\n---\n# Unreviewed\n")
	count, output := d7CLIRunCloseoutGate(t, root, pose.NativeFiles())
	if count != 1 || !strings.Contains(output, "spec:unreviewed") || strings.Contains(output, "spec:alpha:") || strings.Contains(output, "spec:beta:") {
		t.Fatalf("loss was hidden or healthy approvals were poisoned: %d %s", count, output)
	}
}

type d7CLIPolicyInterleaveFiles struct {
	pose.Files
	path          string
	first, second []byte
	mu            sync.Mutex
	opens         int
}
type d7CLIReadFile struct {
	fs.File
	reader *bytes.Reader
}

func (f *d7CLIReadFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *d7CLIPolicyInterleaveFiles) Open(path string) (fs.File, error) {
	file, err := f.Files.Open(path)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(path) != f.path {
		return file, nil
	}
	f.mu.Lock()
	f.opens++
	served := f.first
	if f.opens > 1 {
		served = f.second
	}
	f.mu.Unlock()
	return &d7CLIReadFile{File: file, reader: bytes.NewReader(served)}, nil
}

func TestGovernanceReadViewScopeGlobalGateAuditsDisabledAndEmptyPolicyPaths(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			root := t.TempDir()
			for _, rel := range []string{".pose/specs", ".pose/roadmaps"} {
				if err := os.MkdirAll(filepath.Join(root, rel), 0700); err != nil {
					t.Fatal(err)
				}
			}
			policy := func(value bool) []byte {
				return []byte(fmt.Sprintf(`{"schema_version":1,"enabled":%-5t,"adopted_at":"2026-08-02"}`, value))
			}
			first, second := policy(enabled), policy(!enabled)
			path := filepath.Join(root, ".pose/policy/review.json")
			writeCloseoutCLIFile(t, root, ".pose/policy/review.json", string(first))
			files := &d7CLIPolicyInterleaveFiles{Files: pose.NativeFiles(), path: path, first: first, second: second}
			count, output := d7CLIRunCloseoutGate(t, root, files)
			if count == 0 || !strings.Contains(output, "review closeout") || (!strings.Contains(output, "changed") && !strings.Contains(output, "unknown")) {
				t.Fatalf("early path escaped policy audit: %d %s", count, output)
			}
		})
	}
}

type d7CLISelectionFiles struct {
	pose.Files
	trigger     string
	once        sync.Once
	mutate      func() error
	mutationErr error
}
type d7CLISelectionDir struct {
	fs.File
	owner *d7CLISelectionFiles
}

func (f *d7CLISelectionDir) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := f.File.(fs.ReadDirFile).ReadDir(n)
	f.owner.once.Do(func() { f.owner.mutationErr = f.owner.mutate() })
	return entries, err
}
func (f *d7CLISelectionFiles) Open(path string) (fs.File, error) {
	file, err := f.Files.Open(path)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(path) == f.trigger {
		return &d7CLISelectionDir{File: file, owner: f}, nil
	}
	return file, nil
}

func TestGovernanceReadViewScopeGlobalGateAuditsSpecAndRoadmapEnumeration(t *testing.T) {
	for _, kind := range []string{"add-spec", "completion-date", "add-roadmap"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			for _, rel := range []string{".pose/specs", ".pose/roadmaps"} {
				if err := os.MkdirAll(filepath.Join(root, rel), 0700); err != nil {
					t.Fatal(err)
				}
			}
			writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02"}`)
			original := "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-02\ncompleted_at: 2026-08-01\n---\nbody\n"
			writeCloseoutCLIFile(t, root, ".pose/specs/alpha.md", original)
			files := &d7CLISelectionFiles{Files: pose.NativeFiles(), trigger: filepath.Join(root, ".pose/roadmaps")}
			files.mutate = func() error {
				switch kind {
				case "add-spec":
					return os.WriteFile(filepath.Join(root, ".pose/specs/beta.md"), []byte(strings.ReplaceAll(original, "alpha", "beta")), 0600)
				case "completion-date":
					return os.WriteFile(filepath.Join(root, ".pose/specs/alpha.md"), []byte(strings.Replace(original, "completed_at: 2026-08-01", "completed_at: 2026-08-03", 1)), 0600)
				default:
					return os.WriteFile(filepath.Join(root, ".pose/roadmaps/added.md"), []byte("---\nslug: added\nstatus: done\ncreated_at: 2026-08-03\n---\n# Roadmap\n"), 0600)
				}
			}
			count, output := d7CLIRunCloseoutGate(t, root, files)
			if files.mutationErr != nil {
				t.Fatal(files.mutationErr)
			}
			if count == 0 || (!strings.Contains(output, "changed") && !strings.Contains(output, "unknown")) {
				t.Fatalf("selection mutation %s was certified: %d %s", kind, count, output)
			}
		})
	}
}

func TestGovernanceReadViewScopeGlobalGateKeepsDomainFailureWithSelectionLoss(t *testing.T) {
	root := closeoutCLIFixture(t)
	if err := os.MkdirAll(filepath.Join(root, ".pose/roadmaps"), 0700); err != nil {
		t.Fatal(err)
	}
	writeCloseoutCLIFile(t, root, ".pose/specs/unreviewed/spec.md", "---\nslug: unreviewed\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-13\n---\n# Unreviewed\n")
	files := &d7CLISelectionFiles{Files: pose.NativeFiles(), trigger: filepath.Join(root, ".pose/roadmaps")}
	files.mutate = func() error {
		return os.WriteFile(filepath.Join(root, ".pose/specs/later.md"), []byte("---\nslug: later\nstatus: draft\n---\n"), 0600)
	}
	count, output := d7CLIRunCloseoutGate(t, root, files)
	if files.mutationErr != nil {
		t.Fatal(files.mutationErr)
	}
	if count < 2 || !strings.Contains(output, "spec:unreviewed") || (!strings.Contains(output, "changed") && !strings.Contains(output, "unknown")) {
		t.Fatalf("domain failure or selection audit was discarded: %d %s", count, output)
	}
}
