package pose

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type confinementRecordingFiles struct {
	Files
	reads    int
	metadata []string
}

func (f *confinementRecordingFiles) Lstat(path string) (os.FileInfo, error) {
	f.metadata = append(f.metadata, path)
	return f.Files.Lstat(path)
}

func (f *confinementRecordingFiles) Stat(path string) (os.FileInfo, error) {
	f.metadata = append(f.metadata, path)
	return f.Files.Stat(path)
}

func (f *confinementRecordingFiles) forbidMetadata(t *testing.T, paths ...string) {
	t.Helper()
	for _, got := range f.metadata {
		for _, path := range paths {
			if got == path {
				t.Errorf("inspected external metadata through %s", path)
			}
		}
	}
}

func (f *confinementRecordingFiles) ReadFile(path string) ([]byte, error) {
	f.reads++
	return f.Files.ReadFile(path)
}

func TestUpdatePreviewProjectConfinement(t *testing.T) {
	for _, projected := range []bool{false, true} {
		name := "native"
		if projected {
			name = "projected"
		}
		t.Run(name, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".pose/reports"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("external"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
				t.Skip(err)
			}
			var files Files = NativeFiles()
			if projected {
				files = NewPreviewFiles(time.Now())
			}
			if err := ValidateArtifactPathIn(files, root, "escape/secret", false); err == nil {
				t.Error("ancestor escape accepted")
			}
			if err := ValidateArtifactPathIn(files, root, "escape/missing/claim", false); err == nil {
				t.Error("missing claim through external ancestor accepted")
			}
			recording := &confinementRecordingFiles{Files: files}
			s := Store{Root: root, FS: recording}
			if _, err := s.resolveActionSubject(ArtifactRef{}, ActionSubject{Path: "escape/secret"}); err == nil {
				t.Error("external action subject accepted")
			}
			if recording.reads != 0 {
				t.Fatal("subject refusal read content")
			}
			recording.forbidMetadata(t, filepath.Join(root, "escape/secret"))
			if err := files.MkdirAll(filepath.Join(root, ".pose/reports"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, ".pose/reports/escape")); err != nil {
				t.Fatal(err)
			}
			if err := validateReviewEvidenceRef(recording, root, "report:escape/secret"); err == nil {
				t.Error("external evidence accepted")
			}
			recording.forbidMetadata(t, filepath.Join(root, ".pose/reports/escape/secret"))
			if err := files.WriteFile(filepath.Join(root, ".pose/reports/local.md"), []byte("local"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := validateReviewEvidenceRef(files, root, "report:local.md"); err != nil {
				t.Fatal(err)
			}
			if err := ValidateArtifactPathIn(files, root, "future/claim", false); err != nil {
				t.Fatalf("missing local claim: %v", err)
			}
			if err := files.WriteFile(filepath.Join(root, "local"), []byte("local"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(root, filepath.Join(root, "inside")); err != nil {
				t.Fatal(err)
			}
			if err := ValidateArtifactPathIn(files, root, "inside/local", false); err != nil {
				t.Fatalf("internal link: %v", err)
			}
			if got, err := s.resolveActionSubject(ArtifactRef{}, ActionSubject{Path: "inside/local"}); err != nil || got.Digest != digestBytes([]byte("local")) {
				t.Fatalf("local subject: %+v %v", got, err)
			}
			recording.reads = 0
			if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "final")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, "absent"), filepath.Join(root, "dangling")); err != nil {
				t.Fatal(err)
			}
			for _, rel := range []string{"final", "dangling", "../escape", outside, ".", ""} {
				if err := ValidateArtifactPathIn(files, root, rel, false); err == nil {
					t.Errorf("unsafe claim accepted: %q", rel)
				}
			}
			linkedRoot := filepath.Join(t.TempDir(), "root")
			if err := os.Symlink(root, linkedRoot); err != nil {
				t.Fatal(err)
			}
			if err := ValidateArtifactPathIn(files, linkedRoot, "inside/local", false); err != nil {
				t.Fatalf("linked root: %v", err)
			}
			if err := os.Symlink(outside, filepath.Join(root, ".pose/actions")); err != nil {
				t.Fatal(err)
			}
			id := "act-0123456789abcdef"
			journal := filepath.Join(outside, id+".jsonl")
			if err := os.WriteFile(journal, []byte("unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := s.LoadActionRequest(id); err == nil {
				t.Error("linked journal load accepted")
			}
			if _, err := s.appendActionEvent(id, ActionEvent{}, -1, time.Now()); err == nil {
				t.Error("linked journal append accepted")
			}
			if recording.reads != 0 {
				t.Fatal("journal refusal read content")
			}
			if got, err := os.ReadFile(journal); err != nil || string(got) != "unchanged" {
				t.Fatalf("outside journal changed: %q %v", got, err)
			}
			if err := os.Symlink(outside, filepath.Join(root, ".pose/review-attestations")); err != nil {
				t.Fatal(err)
			}
			s.reviewReuse = newReviewReuseTraversal()
			if err := os.WriteFile(filepath.Join(outside, "rva-0123456789abcdef.json"), []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.reserveReuseArtifact("review-attestations", "rva-0123456789abcdef"); err == nil {
				t.Error("linked reuse directory accepted")
			}
			recording.forbidMetadata(t, filepath.Join(root, ".pose/review-attestations/rva-0123456789abcdef.json"))
			if err := os.WriteFile(filepath.Join(outside, "program.md"), []byte("---\nslug: program\nstatus: done\n---\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, ".pose/roadmaps")); err != nil {
				t.Fatal(err)
			}
			resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"proj.fixture": root}}), FS: recording}
			if got := resolver.Resolve("proj.fixture", "roadmap:program"); got.State != "unavailable-artifact" {
				t.Fatalf("external roadmap: %+v", got)
			}
			recording.forbidMetadata(t, filepath.Join(root, ".pose/roadmaps/program.md"))
		})
	}
}

func TestUpdatePreviewLocalBoundarySyntax(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"", "../escape", root, "act-0123456789abc/ef"} {
		if err := (Store{Root: root}).checkActionJournal(id); err == nil {
			t.Errorf("invalid journal id accepted: %q", id)
		}
	}
	if runtime.GOOS == "windows" {
		for _, rel := range []string{`C:escape`, `C:\escape`, `\escape`, `\\server\share`, "CON", "NUL.txt"} {
			if _, err := validateArtifactPathSyntax(rel); err == nil {
				t.Errorf("nonlocal Windows claim accepted: %q", rel)
			}
		}
	}
}
