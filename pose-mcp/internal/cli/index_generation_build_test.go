package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

type indexMutationFiles struct {
	posepkg.Files
	mu          sync.Mutex
	target      string
	triggerOpen int
	opens       int
	replacement []byte
	mutated     bool
	mutationErr error
}

type indexMutationOnCloseFile struct {
	fs.File
	owner         *indexMutationFiles
	path          string
	mutateOnClose bool
}

func (f *indexMutationOnCloseFile) Close() error {
	err := f.File.Close()
	if !f.mutateOnClose {
		return err
	}
	writeErr := f.owner.Files.WriteFile(f.path, f.owner.replacement, 0o644)
	f.owner.mu.Lock()
	f.owner.mutated = true
	f.owner.mutationErr = writeErr
	f.owner.mu.Unlock()
	return err
}

func (f *indexMutationFiles) Open(name string) (fs.File, error) {
	file, err := f.Files.Open(name)
	if err != nil || filepath.Clean(name) != f.target {
		return file, err
	}
	f.mu.Lock()
	f.opens++
	mutateOnClose := !f.mutated && f.opens == f.triggerOpen
	f.mu.Unlock()
	return &indexMutationOnCloseFile{File: file, owner: f, path: name, mutateOnClose: mutateOnClose}, nil
}

func indexBuildFixtureRoot(t *testing.T, project string) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{
		".pose/specs", ".pose/roadmaps", ".pose/policy", ".pose/changelogs/unreleased", ".pose/releases",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(relative)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	projectFile := fmt.Sprintf(`{"schema_version":1,"project_id":%q}`, project)
	if err := os.WriteFile(filepath.Join(root, ".pose", "project.json"), []byte(projectFile), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func indexBuildWriteFixture(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexGenerationRetriesOutputAndBindingsAsOneReadView(t *testing.T) {
	root := indexBuildFixtureRoot(t, "proj.index-build")
	first := "---\nslug: sample\nstatus: in-progress\n---\n# Version A\n"
	second := "---\nslug: sample\nstatus: done\n---\n# Version B\n"
	indexBuildWriteFixture(t, root, ".pose/specs/sample.md", first)
	inputPath := filepath.Join(root, ".pose", "specs", "sample.md")
	files := &indexMutationFiles{
		Files: posepkg.NativeFiles(), target: filepath.Clean(inputPath), triggerOpen: 3, replacement: []byte(second),
	}
	var stdout, stderr bytes.Buffer
	if code := cmdIndexIn(files, root, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("pose index returned %d: stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	files.mu.Lock()
	mutated, opens, mutationErr := files.mutated, files.opens, files.mutationErr
	files.mu.Unlock()
	if !mutated || mutationErr != nil || opens < 6 {
		t.Fatalf("interleaving did not force a full retry: mutated=%v opens=%d err=%v", mutated, opens, mutationErr)
	}

	generation := (posepkg.Store{Root: root, FS: posepkg.NativeFiles()}).ReadIndexGeneration()
	if generation.Status != posepkg.IndexGenerationCurrent {
		t.Fatalf("published generation status = %s (%s)", generation.Status, generation.Reason)
	}
	var graph struct {
		Specs map[string]struct {
			Status string `json:"status"`
		} `json:"specs"`
	}
	if err := json.Unmarshal(generation.Files["spec-graph.json"], &graph); err != nil {
		t.Fatal(err)
	}
	if got := graph.Specs["sample"].Status; got != "done" {
		t.Fatalf("published output retained attempt A after binding moved to B: sample status=%q", got)
	}
	current, known := posepkg.IndexInputBindingIn(posepkg.NativeFiles(), root, "specs")
	if !known || generation.Descriptor.Inputs["specs"] != current {
		t.Fatalf("published inputs are not from the retried view: descriptor=%q current=%q known=%v", generation.Descriptor.Inputs["specs"], current, known)
	}
}

func TestIndexGenerationAllowsLocalGitColdStartWithoutFederatedInputs(t *testing.T) {
	root := indexBuildFixtureRoot(t, "proj.index-cold-start")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", out, err)
	}
	var stdout, stderr bytes.Buffer
	if code := cmdIndexIn(posepkg.NativeFiles(), root, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("pose index rejected an unborn local repository without federated inputs: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	generation := (posepkg.Store{Root: root, FS: posepkg.NativeFiles()}).ReadIndexGeneration()
	if generation.Status != posepkg.IndexGenerationCurrent {
		t.Fatalf("cold-start generation status = %s (%s)", generation.Status, generation.Reason)
	}
}

func TestIndexGenerationLocksExactFederatedMaterialRoots(t *testing.T) {
	local := indexBuildFixtureRoot(t, "proj.index-local")
	foreign := indexBuildFixtureRoot(t, "proj.index-foreign")
	unrelated := indexBuildFixtureRoot(t, "proj.index-unrelated")
	indexBuildWriteFixture(t, local, ".pose/specs/consumer.md", "---\nslug: consumer\nstatus: in-progress\ndepends_on: xref:proj.index-foreign/spec:producer\n---\n")
	indexBuildWriteFixture(t, foreign, ".pose/specs/producer.md", "---\nslug: producer\nstatus: done\n---\n")
	if out, err := exec.Command("git", "-C", foreign, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init foreign: %s: %v", out, err)
	}
	for _, args := range [][]string{
		{"-c", "user.name=POSE fixture", "-c", "user.email=pose@example.invalid", "add", "--all"},
		{"-c", "user.name=POSE fixture", "-c", "user.email=pose@example.invalid", "commit", "-q", "-m", "foreign fixture"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", foreign}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	projectRoots, err := json.Marshal(map[string]string{
		"proj.index-local": local, "proj.index-foreign": foreign, "proj.index-unrelated": unrelated,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", string(projectRoots))

	t.Run("foreign material contention refuses publication", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := posepkg.WithGovernanceWriteGuard("test.hold-foreign-index-material", []posepkg.Store{{Root: foreign, FS: posepkg.NativeFiles(), FederatedProjectID: "proj.index-foreign"}}, func([]posepkg.Store) error {
			if code := cmdIndexIn(posepkg.NativeFiles(), local, nil, &stdout, &stderr); code != 1 {
				return fmt.Errorf("index with locked foreign material returned %d", code)
			}
			if !strings.Contains(stderr.String(), "governance-write-guard-busy") {
				return fmt.Errorf("foreign material contention was not reported: %q", stderr.String())
			}
			if _, statErr := os.Lstat(filepath.Join(local, ".pose", "indexes", "generation.json")); !errors.Is(statErr, os.ErrNotExist) {
				return fmt.Errorf("index generation was published despite foreign contention: %v", statErr)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unrelated root contention does not block publication", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := posepkg.WithGovernanceWriteGuard("test.hold-unrelated-index-root", []posepkg.Store{{Root: unrelated, FS: posepkg.NativeFiles(), FederatedProjectID: "proj.index-unrelated"}}, func([]posepkg.Store) error {
			if code := cmdIndexIn(posepkg.NativeFiles(), local, nil, &stdout, &stderr); code != 0 {
				return fmt.Errorf("index with locked unrelated root returned %d: %q", code, stderr.String())
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
