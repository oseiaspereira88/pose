package pose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGovernanceReadViewProbeIdentityTracksObservedBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.json")
	firstBytes := []byte(`{"value":"a"}`)
	secondBytes := []byte(`{"value":"b"}`)
	if len(firstBytes) != len(secondBytes) {
		t.Fatal("probe inputs must have equal lengths")
	}

	readID := func(data []byte) (ReadViewMetadata, error) {
		t.Helper()
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return ReadViewMetadata{}, err
		}
		view := newGovernanceReadView(root, NativeFiles())
		got, err := view.filesFor("probe").ReadFile(path)
		if err != nil {
			return ReadViewMetadata{}, err
		}
		if !bytes.Equal(got, data) {
			return ReadViewMetadata{}, errors.New("view did not return the captured bytes")
		}
		metadata, _ := view.metadata("project", "same-revision", "same-index", view.audit())
		return metadata, nil
	}

	first, err := readID(firstBytes)
	if err != nil {
		t.Fatal(err)
	}
	second, err := readID(secondBytes)
	if err != nil {
		t.Fatal(err)
	}
	if first.ObservedInputs != second.ObservedInputs {
		t.Fatalf("probe did not hold the observed-input count constant: first=%d second=%d", first.ObservedInputs, second.ObservedInputs)
	}
	if first.InputBytes != second.InputBytes {
		t.Fatalf("probe did not hold the observed-byte count constant: first=%d second=%d", first.InputBytes, second.InputBytes)
	}
	if first.ID == second.ID {
		t.Fatalf("read-view identity ignored changed bytes with identical context and counts: %s", first.ID)
	}
}

func TestGovernanceReadViewProbeReadFileAbsenceReplayPreservesMeaning(t *testing.T) {
	root := t.TempDir()
	view := newGovernanceReadView(root, NativeFiles())
	path := filepath.Join(root, "absent.json")

	first, firstErr := view.filesFor("producer-a").ReadFile(path)
	second, secondErr := view.filesFor("producer-b").ReadFile(path)
	if len(first) != 0 || len(second) != 0 {
		t.Fatalf("missing reads returned bytes: first=%q second=%q", first, second)
	}
	if !errors.Is(firstErr, os.ErrNotExist) {
		t.Fatalf("first producer lost the absence sentinel: %v", firstErr)
	}
	if !errors.Is(secondErr, os.ErrNotExist) {
		t.Fatalf("replayed absence lost errors.Is(os.ErrNotExist): %v", secondErr)
	}
}

type governanceReadViewProbePermissionErrorFiles struct{ Files }

func (governanceReadViewProbePermissionErrorFiles) Open(string) (fs.File, error) {
	return nil, os.ErrPermission
}

func TestGovernanceReadViewProbeOpenErrorReplayPreservesErrorsIs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "restricted.json")
	if err := os.WriteFile(path, []byte("present but unreadable"), 0o600); err != nil {
		t.Fatal(err)
	}
	view := newGovernanceReadView(root, governanceReadViewProbePermissionErrorFiles{Files: NativeFiles()})
	first, firstErr := view.filesFor("producer-a").ReadFile(path)
	second, secondErr := view.filesFor("producer-b").Open(path)
	if len(first) != 0 || second != nil {
		t.Fatalf("permission failures returned content or a file: first=%q second=%v", first, second)
	}
	if !errors.Is(firstErr, os.ErrPermission) {
		t.Fatalf("initial read lost permission sentinel: %v", firstErr)
	}
	if !errors.Is(secondErr, os.ErrPermission) {
		t.Fatalf("replayed Open failure lost errors.Is(os.ErrPermission): %v", secondErr)
	}
	if len(view.unknown["producer-b"]) == 0 {
		t.Fatalf("replayed Open failure was not attributed as incomplete to its producer: %+v", view.unknown)
	}
}

func TestGovernanceReadViewProbeStableMetadataAndCallerIsolation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.json")
	wantBytes := []byte("captured input")
	if err := os.WriteFile(path, wantBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	view := newGovernanceReadView(root, NativeFiles())
	a := view.filesFor("producer-a")
	b := view.filesFor("producer-b")

	firstBytes, err := a.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes[0] = 'X'
	opened, err := b.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	openedBytes, readErr := io.ReadAll(opened)
	closeErr := opened.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if !bytes.Equal(openedBytes, wantBytes) {
		t.Fatalf("mutating one producer's ReadFile bytes contaminated another producer: got %q, want %q", openedBytes, wantBytes)
	}

	firstInfo, err := a.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	field, original, changed := governanceReadViewProbeMutateSys(t, firstInfo)
	secondInfo, err := b.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := governanceReadViewProbeSysUint(t, secondInfo, field); got != original {
		t.Fatalf("mutating Stat().Sys() contaminated a second producer: got %d, want %d", got, original)
	}
	if changed == original {
		t.Fatal("probe failed to mutate the first Stat().Sys() value")
	}

	opened, err = a.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	openInfo, err := opened.Stat()
	if err != nil {
		t.Fatal(err)
	}
	_, openOriginal, openChanged := governanceReadViewProbeMutateSys(t, openInfo)
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	opened, err = b.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	secondOpenInfo, err := opened.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if got := governanceReadViewProbeSysUint(t, secondOpenInfo, field); got != openOriginal {
		t.Fatalf("mutating Open().Stat().Sys() contaminated another producer: got %d, want %d", got, openOriginal)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if openChanged == openOriginal {
		t.Fatal("probe failed to mutate the first Open().Stat().Sys() value")
	}

	firstEntries, err := a.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	firstEntryInfo, err := firstEntries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	_, dirOriginal, dirChanged := governanceReadViewProbeMutateSys(t, firstEntryInfo)
	secondEntries, err := b.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	secondEntryInfo, err := secondEntries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if got := governanceReadViewProbeSysUint(t, secondEntryInfo, field); got != dirOriginal {
		t.Fatalf("mutating DirEntry.Info().Sys() contaminated another producer: got %d, want %d", got, dirOriginal)
	}
	if dirChanged == dirOriginal {
		t.Fatal("probe failed to mutate the first DirEntry.Info().Sys() value")
	}

	if audit := view.audit(); len(audit.ChangedPaths) != 0 {
		t.Fatalf("stable Stat/Open/DirEntry.Info reads produced a false closure conflict: %v", audit.ChangedPaths)
	}
	metadata, proof := view.metadata("project", "revision", "index", view.audit())
	if changed := proof.changes(NativeFiles()); len(changed) != 0 {
		t.Fatalf("stable read proof produced a false conflict after caller mutations: %v", changed)
	}
	if metadata.InputBytes != int64(len(wantBytes)) {
		t.Fatalf("caller mutation changed captured byte accounting: got %d, want %d", metadata.InputBytes, len(wantBytes))
	}
}

type governanceReadViewProbePermissionFiles struct {
	Files
	reads int
}

func (f *governanceReadViewProbePermissionFiles) ReadFile(string) ([]byte, error) {
	f.reads++
	return nil, os.ErrPermission
}

func (f *governanceReadViewProbePermissionFiles) Open(string) (fs.File, error) {
	f.reads++
	return nil, os.ErrPermission
}

func TestGovernanceReadViewProbeStablePermissionFailureIsIncompleteForItsProducer(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "restricted.json")
	if err := os.WriteFile(path, []byte("present but unreadable"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := &governanceReadViewProbePermissionFiles{Files: NativeFiles()}
	view := newGovernanceReadView(root, base)
	_, err := view.filesFor("permission-producer").ReadFile(path)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("probe did not observe the intended permission failure: %v", err)
	}

	metadata, _ := view.metadata("project", "revision", "index", view.audit())
	if metadata.Freshness == ReadViewFreshnessCurrent {
		t.Fatal("a stable permission failure was treated as a complete current observation")
	}
	if len(view.unknown["permission-producer"]) == 0 {
		t.Fatalf("permission failure was not attributed as incomplete to its producer: %+v", view.unknown)
	}
	if len(view.unknown) != 1 {
		t.Fatalf("stable read failure contaminated unrelated producer coverage: %+v", view.unknown)
	}
	if base.reads == 0 {
		t.Fatal("permission provider was not observed")
	}
}

func governanceReadViewProbeMutateSys(t *testing.T, info fs.FileInfo) (field string, original, changed uint64) {
	t.Helper()
	sys := reflect.ValueOf(info.Sys())
	if !sys.IsValid() || sys.Kind() != reflect.Pointer || sys.IsNil() {
		t.Skip("platform FileInfo.Sys does not expose a mutable pointer for this isolation probe")
	}
	value := sys.Elem()
	for _, name := range []string{"Uid", "Nlink", "Blocks"} {
		candidate := value.FieldByName(name)
		if !candidate.IsValid() || !candidate.CanSet() || candidate.Kind() < reflect.Uint || candidate.Kind() > reflect.Uint64 {
			continue
		}
		original := candidate.Uint()
		changed := original + 1
		candidate.SetUint(changed)
		return name, original, changed
	}
	t.Skip("platform FileInfo.Sys has no exported mutable unsigned field")
	return "", 0, 0
}

func governanceReadViewProbeSysUint(t *testing.T, info fs.FileInfo, field string) uint64 {
	t.Helper()
	sys := reflect.ValueOf(info.Sys())
	if !sys.IsValid() || sys.Kind() != reflect.Pointer || sys.IsNil() {
		t.Fatalf("FileInfo.Sys no longer exposes a pointer: %T", info.Sys())
	}
	value := sys.Elem().FieldByName(field)
	if !value.IsValid() || value.Kind() < reflect.Uint || value.Kind() > reflect.Uint64 {
		t.Fatalf("FileInfo.Sys has no unsigned %s field: %T", field, info.Sys())
	}
	return value.Uint()
}

type governanceReadViewProbeOversizedInfo struct{}

func (governanceReadViewProbeOversizedInfo) Name() string       { return "oversized.bin" }
func (governanceReadViewProbeOversizedInfo) Size() int64        { return governanceReadViewMaxInputBytes + 1 }
func (governanceReadViewProbeOversizedInfo) Mode() fs.FileMode  { return 0o600 }
func (governanceReadViewProbeOversizedInfo) ModTime() time.Time { return time.Unix(1, 0) }
func (governanceReadViewProbeOversizedInfo) IsDir() bool        { return false }
func (governanceReadViewProbeOversizedInfo) Sys() any           { return nil }

type governanceReadViewProbeGuardFile struct {
	info  fs.FileInfo
	reads *int
}

func (f *governanceReadViewProbeGuardFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *governanceReadViewProbeGuardFile) Read([]byte) (int, error) {
	*f.reads++
	return 0, errors.New("probe: oversized file was read")
}
func (*governanceReadViewProbeGuardFile) Close() error { return nil }

type governanceReadViewProbeBudgetFiles struct {
	Files
	readFileCalls int
	openCalls     int
	statCalls     int
	readCalls     int
}

func (f *governanceReadViewProbeBudgetFiles) ReadFile(string) ([]byte, error) {
	f.readFileCalls++
	return nil, errors.New("probe: unbounded ReadFile fallback")
}
func (f *governanceReadViewProbeBudgetFiles) Open(name string) (fs.File, error) {
	f.openCalls++
	file, err := f.Files.Open(name)
	if err != nil {
		return nil, err
	}
	return governanceReadViewProbeTrackedFile{File: file, reads: &f.readCalls}, nil
}
func (f *governanceReadViewProbeBudgetFiles) Stat(name string) (fs.FileInfo, error) {
	f.statCalls++
	return f.Files.Stat(name)
}

type governanceReadViewProbeTrackedFile struct {
	fs.File
	reads *int
}

func (f governanceReadViewProbeTrackedFile) Read(p []byte) (int, error) {
	*f.reads++
	return f.File.Read(p)
}

type governanceReadViewProbeOversizedFiles struct {
	Files
	readCalls     int
	readFileCalls int
	openCalls     int
}

func (f *governanceReadViewProbeOversizedFiles) ReadFile(string) ([]byte, error) {
	f.readFileCalls++
	return nil, errors.New("probe: proof attempted unbounded ReadFile")
}
func (f *governanceReadViewProbeOversizedFiles) Open(string) (fs.File, error) {
	f.openCalls++
	return &governanceReadViewProbeGuardFile{info: governanceReadViewProbeOversizedInfo{}, reads: &f.readCalls}, nil
}

func (f *governanceReadViewProbeOversizedFiles) Stat(string) (fs.FileInfo, error) {
	return governanceReadViewProbeOversizedInfo{}, nil
}

func TestGovernanceReadViewProbeBoundsBeforeReadingAuditProofAndFallback(t *testing.T) {
	t.Run("oversized capture is refused without reading", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "oversized.bin")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(path, governanceReadViewMaxInputBytes+1); err != nil {
			t.Fatal(err)
		}
		base := &governanceReadViewProbeBudgetFiles{Files: NativeFiles()}
		view := newGovernanceReadView(root, base)

		data, err := view.filesFor("capture").ReadFile(path)
		if err == nil || len(data) != 0 {
			t.Fatalf("oversized capture was returned instead of refused: len=%d err=%v", len(data), err)
		}
		if !errors.Is(err, errGovernanceReadViewBudget) {
			t.Fatalf("oversized capture was refused for a reason other than the byte budget: %v", err)
		}
		audit := view.audit()
		metadata, proof := view.metadata("project", "revision", "index", audit)
		_ = proof.changes(base)
		if metadata.Freshness != ReadViewFreshnessUnknown {
			t.Fatalf("budget refusal did not make freshness unknown: %s", metadata.Freshness)
		}
		if metadata.InputBytes > governanceReadViewMaxInputBytes {
			t.Fatalf("view exceeded byte budget: %d > %d", metadata.InputBytes, governanceReadViewMaxInputBytes)
		}
		if base.readCalls != 0 {
			t.Fatalf("oversized file was read during capture, audit, or proof: reads=%d", base.readCalls)
		}
		if base.openCalls == 0 && base.statCalls == 0 {
			t.Fatal("oversized capture was refused without consulting bounded file metadata")
		}
		if base.readFileCalls != 0 {
			t.Fatalf("capture fell back to unbounded Files.ReadFile: calls=%d", base.readFileCalls)
		}
	})

	t.Run("exhausted budget does not fall back to the base reader", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "oversized.bin")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(path, governanceReadViewMaxInputBytes+1); err != nil {
			t.Fatal(err)
		}
		base := &governanceReadViewProbeBudgetFiles{Files: NativeFiles()}
		view := newGovernanceReadView(root, base)
		view.inputBytes = governanceReadViewMaxInputBytes

		data, err := view.filesFor("fallback").ReadFile(path)
		if err == nil || len(data) != 0 {
			t.Fatalf("exhausted view budget returned uncaptured data: len=%d err=%v", len(data), err)
		}
		if !errors.Is(err, errGovernanceReadViewBudget) {
			t.Fatalf("exhausted view did not return a budget error: %v", err)
		}
		metadata, proof := view.metadata("project", "revision", "index", view.audit())
		_ = proof.changes(base)
		if metadata.Freshness != ReadViewFreshnessUnknown {
			t.Fatalf("exhausted budget did not make freshness unknown: %s", metadata.Freshness)
		}
		if base.readFileCalls != 0 || base.readCalls != 0 {
			t.Fatalf("exhausted budget reached an unbounded reader: ReadFile=%d Read=%d", base.readFileCalls, base.readCalls)
		}
	})

	t.Run("Open refuses oversized input before reading it", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "oversized.bin")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(path, governanceReadViewMaxInputBytes+1); err != nil {
			t.Fatal(err)
		}
		base := &governanceReadViewProbeBudgetFiles{Files: NativeFiles()}
		view := newGovernanceReadView(root, base)

		file, err := view.filesFor("open-capture").Open(path)
		if err == nil || file != nil {
			t.Fatalf("oversized Open capture was returned instead of refused: file=%v err=%v", file, err)
		}
		if !errors.Is(err, errGovernanceReadViewBudget) {
			t.Fatalf("oversized Open was refused for a reason other than the byte budget: %v", err)
		}
		metadata, proof := view.metadata("project", "revision", "index", view.audit())
		_ = proof.changes(base)
		if metadata.Freshness != ReadViewFreshnessUnknown {
			t.Fatalf("oversized Open did not make freshness unknown: %s", metadata.Freshness)
		}
		if base.readCalls != 0 {
			t.Fatalf("oversized Open was read during capture, audit, or proof: reads=%d", base.readCalls)
		}
		if base.openCalls != 0 {
			t.Fatalf("oversized Open reached the underlying file after metadata proved it exceeds the budget: calls=%d", base.openCalls)
		}
		if base.statCalls == 0 {
			t.Fatal("oversized Open was refused without inspecting its file size")
		}
	})

	t.Run("proof replay bounds an Open before reading a replacement", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "input.bin")
		if err := os.WriteFile(path, []byte("small captured file"), 0o600); err != nil {
			t.Fatal(err)
		}
		view := newGovernanceReadView(root, NativeFiles())
		file, err := view.filesFor("capture").Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadAll(file); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		_, proof := view.metadata("project", "revision", "index", view.audit())

		base := &governanceReadViewProbeOversizedFiles{Files: NativeFiles()}
		changed := proof.changes(base)
		if len(changed) == 0 {
			t.Fatal("proof replay did not report the replacement input as changed")
		}
		if base.readCalls != 0 {
			t.Fatalf("proof replay read an oversized replacement: reads=%d", base.readCalls)
		}
		if base.readFileCalls != 0 {
			t.Fatalf("proof replay used unbounded ReadFile for an oversized replacement: calls=%d", base.readFileCalls)
		}
	})

	t.Run("proof replay bounds a ReadFile capture before reading a replacement", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "input.bin")
		if err := os.WriteFile(path, []byte("small captured file"), 0o600); err != nil {
			t.Fatal(err)
		}
		view := newGovernanceReadView(root, NativeFiles())
		if _, err := view.filesFor("capture").ReadFile(path); err != nil {
			t.Fatal(err)
		}
		_, proof := view.metadata("project", "revision", "index", view.audit())

		base := &governanceReadViewProbeOversizedFiles{Files: NativeFiles()}
		changed := proof.changes(base)
		if len(changed) == 0 {
			t.Fatal("proof replay did not report the replacement input as changed")
		}
		if base.readCalls != 0 || base.readFileCalls != 0 {
			t.Fatalf("proof replay read an oversized replacement: Open reads=%d ReadFile calls=%d", base.readCalls, base.readFileCalls)
		}
	})

	t.Run("oversized directory is refused before inspecting every entry", func(t *testing.T) {
		root := t.TempDir()
		entries := make([]fs.DirEntry, governanceReadViewMaxRecords+1)
		infoCalls := 0
		for i := range entries {
			entries[i] = governanceReadViewProbeCountedDirEntry{name: fmt.Sprintf("entry-%05d", i), infoCalls: &infoCalls}
		}
		base := &governanceReadViewProbeDirectoryFiles{Files: NativeFiles(), root: root, entries: entries}
		view := newGovernanceReadView(root, base)

		observed, err := view.filesFor("directory-reader").ReadDir(root)
		if err == nil || len(observed) != 0 {
			t.Fatalf("oversized directory was silently truncated or accepted: entries=%d err=%v", len(observed), err)
		}
		metadata, _ := view.metadata("project", "revision", "index", view.audit())
		if metadata.Freshness != ReadViewFreshnessUnknown {
			t.Fatalf("directory budget refusal did not make freshness unknown: %s", metadata.Freshness)
		}
		if infoCalls != 0 {
			t.Fatalf("read view inspected DirEntry.Info for rejected directory entries: calls=%d", infoCalls)
		}
	})

	t.Run("DirEntry.Info failure makes only its producer unknown", func(t *testing.T) {
		root := t.TempDir()
		entries := []fs.DirEntry{
			governanceReadViewProbeCountedDirEntry{
				name:      "restricted.json",
				infoCalls: new(int),
				infoErr:   os.ErrPermission,
			},
		}
		view := newGovernanceReadView(root, &governanceReadViewProbeDirectoryFiles{Files: NativeFiles(), root: root, entries: entries})
		observed, err := view.filesFor("directory-reader").ReadDir(root)
		if err != nil || len(observed) != 1 {
			t.Fatalf("directory names were lost with one Info failure: entries=%d err=%v", len(observed), err)
		}
		if _, err := observed[0].Info(); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("DirEntry.Info replay lost the permission sentinel: %v", err)
		}
		replayed, err := view.filesFor("directory-consumer").ReadDir(root)
		if err != nil || len(replayed) != 1 {
			t.Fatalf("cached directory listing was unavailable to a second producer: entries=%d err=%v", len(replayed), err)
		}
		if _, err := replayed[0].Info(); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("cached DirEntry.Info replay lost the permission sentinel: %v", err)
		}
		metadata, _ := view.metadata("project", "revision", "index", view.audit())
		if metadata.Freshness != ReadViewFreshnessUnknown || len(view.unknown["directory-reader"]) == 0 {
			t.Fatalf("DirEntry.Info failure was certified current: metadata=%+v unknown=%+v", metadata, view.unknown)
		}
		if len(view.unknown["directory-consumer"]) == 0 {
			t.Fatalf("cached DirEntry.Info failure was not attributed to its second producer: %+v", view.unknown)
		}
		if len(view.unknown) != 2 {
			t.Fatalf("DirEntry.Info failure contaminated unrelated producer coverage: %+v", view.unknown)
		}
	})
}

type governanceReadViewProbeCountedDirEntry struct {
	name      string
	infoCalls *int
	infoErr   error
}

func (e governanceReadViewProbeCountedDirEntry) Name() string    { return e.name }
func (governanceReadViewProbeCountedDirEntry) IsDir() bool       { return false }
func (governanceReadViewProbeCountedDirEntry) Type() fs.FileMode { return 0 }
func (e governanceReadViewProbeCountedDirEntry) Info() (fs.FileInfo, error) {
	*e.infoCalls++
	if e.infoErr != nil {
		return nil, e.infoErr
	}
	return governanceReadViewProbeOversizedInfo{}, nil
}

type governanceReadViewProbeDirectoryFiles struct {
	Files
	root    string
	entries []fs.DirEntry
}

func (f *governanceReadViewProbeDirectoryFiles) Open(name string) (fs.File, error) {
	if filepath.Clean(name) == filepath.Clean(f.root) {
		info, err := os.Stat(f.root)
		if err != nil {
			return nil, err
		}
		return &governanceReadViewProbeDirectoryFile{info: info, entries: f.entries}, nil
	}
	return f.Files.Open(name)
}

type governanceReadViewProbeDirectoryFile struct {
	info    fs.FileInfo
	entries []fs.DirEntry
	cursor  int
}

func (f *governanceReadViewProbeDirectoryFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (*governanceReadViewProbeDirectoryFile) Read([]byte) (int, error) {
	return 0, errors.New("probe: directory file cannot be read as bytes")
}
func (*governanceReadViewProbeDirectoryFile) Close() error { return nil }
func (f *governanceReadViewProbeDirectoryFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.cursor >= len(f.entries) {
		return nil, io.EOF
	}
	end := len(f.entries)
	if n > 0 && end-f.cursor > n {
		end = f.cursor + n
	}
	result := append([]fs.DirEntry(nil), f.entries[f.cursor:end]...)
	f.cursor = end
	return result, nil
}

func TestGovernanceReadViewProbeDetectsInterleavedMutationBeforeFinalAudit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "subject.json")
	first := []byte(`{"state":"a"}`)
	second := []byte(`{"state":"b"}`)
	if err := os.WriteFile(path, first, 0o600); err != nil {
		t.Fatal(err)
	}
	firstInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	view := newGovernanceReadView(root, NativeFiles())
	captured, err := view.filesFor("subject-producer").ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(captured, first) {
		t.Fatalf("initial producer read = %q, want %q", captured, first)
	}
	if err := os.WriteFile(path, second, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, firstInfo.ModTime(), firstInfo.ModTime()); err != nil {
		t.Fatal(err)
	}

	audit := view.audit()
	if len(audit.ChangedPaths) == 0 || !audit.ChangedProducers["subject-producer"] {
		t.Fatalf("interleaved same-size mutation escaped final audit attribution: %+v", audit)
	}
	metadata, _ := view.metadata("project", "same-revision", "same-index", audit)
	if metadata.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("interleaved mutation was certified current: %+v", metadata)
	}
}

func TestGovernanceReadViewProbeSymlinkIdentityIsPartOfObservedClosure(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	link := filepath.Join(root, "subject.json")
	if err := os.WriteFile(target, []byte(`{"state":"same"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.json", link); err != nil {
		t.Skipf("create initial symlink: %v", err)
	}
	firstView := newGovernanceReadView(root, NativeFiles())
	if _, err := firstView.filesFor("subject-producer").ReadFile(link); err != nil {
		t.Fatal(err)
	}
	first, _ := firstView.metadata("project", "same-revision", "same-index", governanceReadViewAudit{})
	if first.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("initial symlink observation was not current: %+v", first)
	}

	replacement := filepath.Join(root, "subject-replacement.json")
	if err := os.Symlink("target.json", replacement); err != nil {
		t.Skipf("create replacement symlink with the same target: %v", err)
	}
	if err := os.Rename(replacement, link); err != nil {
		t.Skipf("atomically replace symlink with the same target: %v", err)
	}
	firstAudit := firstView.audit()
	firstAfterAudit, _ := firstView.metadata("project", "same-revision", "same-index", firstAudit)
	if len(firstAudit.ChangedPaths) == 0 || !firstAudit.ChangedProducers["subject-producer"] {
		t.Fatalf("replacing an observed symlink with an identical target escaped audit: %+v", firstAudit)
	}
	if firstAfterAudit.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("replaced observed symlink remained current: %+v", firstAfterAudit)
	}
	secondView := newGovernanceReadView(root, NativeFiles())
	if _, err := secondView.filesFor("subject-producer").ReadFile(link); err != nil {
		t.Fatal(err)
	}
	secondAudit := secondView.audit()
	second, _ := secondView.metadata("project", "same-revision", "same-index", secondAudit)
	if second.ID == first.ID {
		t.Fatalf("read-view identity ignored replacing a symlink inode with identical target text: %s", first.ID)
	}
	if second.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("replacement symlink was not current in a new operation: %+v", second)
	}
}

func governanceReadViewProbeUseKnownEngineVersion(t *testing.T) {
	t.Helper()
	previous := EngineVersion
	EngineVersion = "probe-engine-v1"
	t.Cleanup(func() { EngineVersion = previous })
}

func TestGovernanceReadViewProbeRejectsNonregularMetadataBeforeOpen(t *testing.T) {
	for _, operation := range []string{"ReadFile", "Open"} {
		t.Run(operation, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "blocked.fifo")
			if err := os.WriteFile(path, []byte("regular backing file"), 0o600); err != nil {
				t.Fatal(err)
			}
			base := &governanceReadViewProbeNonregularFiles{Files: NativeFiles(), path: path, info: governanceReadViewProbeNonregularInfo{}}
			view := newGovernanceReadView(root, base)
			files := view.filesFor("fifo-reader")
			if operation == "ReadFile" {
				if _, err := files.ReadFile(path); err == nil {
					t.Fatal("ReadFile accepted metadata identifying a nonregular input")
				}
			} else {
				if file, err := files.Open(path); err == nil || file != nil {
					t.Fatalf("Open accepted metadata identifying a nonregular input: file=%v err=%v", file, err)
				}
			}
			if base.openCalls != 0 {
				t.Fatalf("nonregular input was opened before rejection: calls=%d", base.openCalls)
			}
			metadata, _ := view.metadata("project", "revision", "index", view.audit())
			if metadata.Freshness != ReadViewFreshnessUnknown {
				t.Fatalf("nonregular input was certified current: %+v", metadata)
			}
		})
	}
}

type governanceReadViewProbeNonregularInfo struct{}

func (governanceReadViewProbeNonregularInfo) Name() string       { return "blocked.fifo" }
func (governanceReadViewProbeNonregularInfo) Size() int64        { return 0 }
func (governanceReadViewProbeNonregularInfo) Mode() fs.FileMode  { return fs.ModeNamedPipe | 0o600 }
func (governanceReadViewProbeNonregularInfo) ModTime() time.Time { return time.Unix(1, 0) }
func (governanceReadViewProbeNonregularInfo) IsDir() bool        { return false }
func (governanceReadViewProbeNonregularInfo) Sys() any           { return nil }

type governanceReadViewProbeNonregularFiles struct {
	Files
	path      string
	info      fs.FileInfo
	openCalls int
}

func (f *governanceReadViewProbeNonregularFiles) Stat(name string) (fs.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(f.path) {
		return f.info, nil
	}
	return f.Files.Stat(name)
}

func (f *governanceReadViewProbeNonregularFiles) Lstat(name string) (fs.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(f.path) {
		return f.info, nil
	}
	return f.Files.Lstat(name)
}

func (f *governanceReadViewProbeNonregularFiles) Open(string) (fs.File, error) {
	f.openCalls++
	return nil, errors.New("probe: unexpected Open of nonregular input")
}

func TestGovernanceReadViewProbeRejectsUnboundedReadDirFallback(t *testing.T) {
	root := t.TempDir()
	entries := make([]fs.DirEntry, 11)
	infoCalls := 0
	for i := range entries {
		entries[i] = governanceReadViewProbeCountedDirEntry{name: fmt.Sprintf("entry-%02d", i), infoCalls: &infoCalls}
	}
	base := &governanceReadViewProbeNonStreamingDirFiles{Files: NativeFiles(), root: root, entries: entries}
	observed, err := readDirGovernanceBounded(base, root, 10)
	if err == nil || len(observed) != 0 {
		t.Fatalf("non-streaming directory fallback returned an unbounded listing: entries=%d err=%v", len(observed), err)
	}
	if base.readDirCalls != 0 {
		t.Fatalf("bounded directory reader fell back to unbounded Files.ReadDir: calls=%d", base.readDirCalls)
	}
	if infoCalls != 0 {
		t.Fatalf("non-streaming directory fallback inspected entry metadata: calls=%d", infoCalls)
	}
}

type governanceReadViewProbeNonStreamingDirFiles struct {
	Files
	root         string
	entries      []fs.DirEntry
	readDirCalls int
}

func (f *governanceReadViewProbeNonStreamingDirFiles) Open(name string) (fs.File, error) {
	if filepath.Clean(name) == filepath.Clean(f.root) {
		info, err := os.Stat(f.root)
		if err != nil {
			return nil, err
		}
		return &governanceReadViewProbeNonStreamingDirFile{info: info}, nil
	}
	return f.Files.Open(name)
}

func (f *governanceReadViewProbeNonStreamingDirFiles) ReadDir(string) ([]fs.DirEntry, error) {
	f.readDirCalls++
	return append([]fs.DirEntry(nil), f.entries...), nil
}

type governanceReadViewProbeNonStreamingDirFile struct{ info fs.FileInfo }

func (f *governanceReadViewProbeNonStreamingDirFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (*governanceReadViewProbeNonStreamingDirFile) Read([]byte) (int, error) {
	return 0, errors.New("probe: directory handle cannot be read as bytes")
}
func (*governanceReadViewProbeNonStreamingDirFile) Close() error { return nil }

func TestGovernanceReadViewProbeIdentityIndependentOfReadFileOpenOrder(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	root := t.TempDir()
	path := filepath.Join(root, "shared-input.json")
	want := []byte(`{"value":"same generation"}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	modTime := time.Unix(1_800_000_000, 123)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	observe := func(readFileFirst bool) ReadViewMetadata {
		t.Helper()
		view := newGovernanceReadView(root, NativeFiles())
		files := view.filesFor("probe")
		readFile := func() {
			t.Helper()
			got, err := files.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("ReadFile returned %q, want %q", got, want)
			}
		}
		open := func() {
			t.Helper()
			file, err := files.Open(path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			got, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil {
				t.Fatalf("read Open result: %v", readErr)
			}
			if closeErr != nil {
				t.Fatalf("close Open result: %v", closeErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("Open returned %q, want %q", got, want)
			}
		}
		if readFileFirst {
			readFile()
			open()
		} else {
			open()
			readFile()
		}
		metadata, _ := view.metadata("project", "revision", "index", view.audit())
		return metadata
	}

	readFileFirst := observe(true)
	openFirst := observe(false)
	if readFileFirst.ID != openFirst.ID {
		t.Fatalf("equivalent ReadFile+Open observations got order-dependent identities: ReadFile-first=%s Open-first=%s", readFileFirst.ID, openFirst.ID)
	}
}

func TestGovernanceReadViewProbeDirectoryOrderMatchesNativeFiles(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	root := t.TempDir()
	dir := filepath.Join(root, "entries")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"charlie", "bravo", "alpha"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := NativeFiles().ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	streamed := append([]fs.DirEntry(nil), want...)
	for left, right := 0, len(streamed)-1; left < right; left, right = left+1, right-1 {
		streamed[left], streamed[right] = streamed[right], streamed[left]
	}

	base := &governanceReadViewProbeDirectoryFiles{Files: NativeFiles(), root: dir, entries: streamed}
	view := newGovernanceReadView(root, base)
	got, err := view.filesFor("probe").ReadDir(dir)
	if err != nil {
		t.Fatalf("read bounded directory: %v", err)
	}
	wantNames := make([]string, len(want))
	gotNames := make([]string, len(got))
	for i := range want {
		wantNames[i] = want[i].Name()
	}
	for i := range got {
		gotNames[i] = got[i].Name()
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("governance ReadDir order = %v, NativeFiles order = %v", gotNames, wantNames)
	}
}

func TestGovernanceReadViewProbeOpenDirectoryReplaysBoundedSortedInventory(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	root := t.TempDir()
	dir := filepath.Join(root, "entries")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"charlie", "bravo", "alpha"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	nativeEntries, err := NativeFiles().ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	streamed := append([]fs.DirEntry(nil), nativeEntries...)
	for left, right := 0, len(streamed)-1; left < right; left, right = left+1, right-1 {
		streamed[left], streamed[right] = streamed[right], streamed[left]
	}
	base := &governanceReadViewProbeOpenDirectoryFiles{Files: NativeFiles(), dir: dir, entries: streamed}
	view := newGovernanceReadView(root, base)
	first, err := view.filesFor("directory-open-a").Open(dir)
	if err != nil {
		t.Fatalf("open directory through the captured view: %v", err)
	}
	firstInfo, err := first.Stat()
	if err != nil || !firstInfo.IsDir() {
		t.Fatalf("directory Open().Stat() = %+v, %v", firstInfo, err)
	}
	firstReader, ok := first.(interface {
		ReadDir(int) ([]fs.DirEntry, error)
	})
	if !ok {
		t.Fatal("captured directory does not implement ReadDir")
	}
	firstBatch, err := firstReader.ReadDir(1)
	if err != nil || len(firstBatch) != 1 || firstBatch[0].Name() != "alpha" {
		t.Fatalf("first bounded directory batch = %v, %v", dirEntryNames(firstBatch), err)
	}
	remaining, err := firstReader.ReadDir(-1)
	if err != nil || !reflect.DeepEqual(dirEntryNames(remaining), []string{"bravo", "charlie"}) {
		t.Fatalf("remaining directory batch = %v, %v", dirEntryNames(remaining), err)
	}
	if _, err := firstReader.ReadDir(1); !errors.Is(err, io.EOF) {
		t.Fatalf("exhausted directory did not report EOF: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := view.filesFor("directory-open-b").Open(dir)
	if err != nil {
		t.Fatalf("replay the captured directory to a second producer: %v", err)
	}
	secondReader := second.(interface {
		ReadDir(int) ([]fs.DirEntry, error)
	})
	all, err := secondReader.ReadDir(0)
	if err != nil || !reflect.DeepEqual(dirEntryNames(all), []string{"alpha", "bravo", "charlie"}) {
		t.Fatalf("replayed directory inventory = %v, %v", dirEntryNames(all), err)
	}
	if base.openCalls != 1 {
		t.Fatalf("directory replay reopened the underlying directory: calls=%d", base.openCalls)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}

	audit := view.audit()
	if len(audit.ChangedPaths) != 0 {
		t.Fatalf("stable captured directory was reported changed: %+v", audit)
	}
	if err := os.WriteFile(filepath.Join(dir, "delta"), []byte("delta"), 0o600); err != nil {
		t.Fatal(err)
	}
	audit = view.audit()
	if len(audit.ChangedPaths) == 0 || !audit.ChangedProducers["directory-open-a"] || !audit.ChangedProducers["directory-open-b"] {
		t.Fatalf("directory inventory/stat change was not audited for both readers: %+v", audit)
	}
}

type governanceReadViewProbeOpenDirectoryFiles struct {
	Files
	dir       string
	entries   []fs.DirEntry
	openCalls int
}

func (f *governanceReadViewProbeOpenDirectoryFiles) Open(name string) (fs.File, error) {
	if filepath.Clean(name) == filepath.Clean(f.dir) {
		f.openCalls++
		info, err := f.Files.Stat(name)
		if err != nil {
			return nil, err
		}
		return &governanceReadViewProbeDirectoryFile{info: info, entries: f.entries}, nil
	}
	return f.Files.Open(name)
}

func dirEntryNames(entries []fs.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

type governanceReadViewProbeZeroSizeInfo struct{ fs.FileInfo }

func (governanceReadViewProbeZeroSizeInfo) Size() int64 { return 0 }

type governanceReadViewProbeStaleSmallFiles struct {
	Files
	path      string
	statCalls int
	openCalls int
}

func (f *governanceReadViewProbeStaleSmallFiles) Stat(name string) (fs.FileInfo, error) {
	f.statCalls++
	info, err := f.Files.Stat(name)
	if err == nil && filepath.Clean(name) == filepath.Clean(f.path) && f.statCalls == 1 {
		return governanceReadViewProbeZeroSizeInfo{FileInfo: info}, nil
	}
	return info, err
}

func (f *governanceReadViewProbeStaleSmallFiles) Open(name string) (fs.File, error) {
	f.openCalls++
	return f.Files.Open(name)
}

func TestGovernanceReadViewProbeStaleSmallPreflightCannotOverrunReservation(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	root := t.TempDir()
	path := filepath.Join(root, "input.json")
	want := []byte(`{"value":"larger than stale stat"}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	base := &governanceReadViewProbeStaleSmallFiles{Files: NativeFiles(), path: path}
	view := newGovernanceReadView(root, base)

	got, err := view.filesFor("stale-small").ReadFile(path)
	if !errors.Is(err, errGovernanceReadViewBudget) || len(got) != 0 {
		t.Fatalf("stale small preflight returned captured bytes: len=%d err=%v", len(got), err)
	}
	metadata, proof := view.metadata("project", "revision", "index", view.audit())
	_ = proof.changes(base)
	if metadata.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("stale size did not make the read view unknown: %+v", metadata)
	}
	if metadata.InputBytes != 0 {
		t.Fatalf("refused input retained/accounted %d bytes, want zero", metadata.InputBytes)
	}
	if base.openCalls != 0 {
		t.Fatalf("stale size reached Open before the actual size was bounded: calls=%d", base.openCalls)
	}
}

func TestGovernanceReadViewProbeRecordCapacityAndOverflow(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	// Keep absence error strings short: this test isolates the record cap
	// while retaining the unchanged, independently enforced metadata cap.
	root, err := os.MkdirTemp("", "b34-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	view := newGovernanceReadView(root, NativeFiles())
	files := view.filesFor("capacity")
	// Lstat retains real absence observations and stable parent witnesses. Fill
	// the new boundary without creating 40,000 files or retaining file bodies.
	for i := 0; len(view.observed) < governanceReadViewMaxRecords; i++ {
		_, err := files.Lstat(filepath.Join(root, fmt.Sprintf("absent-%05d", i)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("observation %d refused before the new boundary: %v", i, err)
		}
	}
	if len(view.observed) != governanceReadViewMaxRecords {
		t.Fatalf("capture retained %d records, want %d", len(view.observed), governanceReadViewMaxRecords)
	}
	// Auditors and replay must accept the same boundary as capture.
	audit := view.audit()
	if governanceReviewAuditChanged(audit) || len(audit.UnknownProducers) != 0 {
		t.Fatalf("stable boundary failed final audit: %+v", audit)
	}
	metadata, proof := view.metadata("project", "revision", "index", audit)
	if metadata.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("stable capacity was not current: %+v", metadata)
	}
	if changed := proof.changes(NativeFiles()); len(changed) != 0 {
		t.Fatalf("proof replay rejected the new capacity: %v", changed)
	}

	// A separate capture reaches the same boundary and rejects one more real
	// observation. Finalized views above cannot be reused to exercise overflow.
	overflow := newGovernanceReadView(root, NativeFiles())
	for i := 0; len(overflow.observed) < governanceReadViewMaxRecords; i++ {
		if _, err := overflow.filesFor("capacity").Lstat(filepath.Join(root, fmt.Sprintf("absent-%05d", i))); !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	if _, err := overflow.filesFor("overflow").Lstat(filepath.Join(root, "one-too-many")); !errors.Is(err, errGovernanceReadViewBudget) {
		t.Fatalf("overflow did not reject the next observation: %v", err)
	}
	if audit := overflow.audit(); len(audit.UnknownProducers["overflow"]) == 0 || len(audit.UnknownProducers["capacity"]) != 0 {
		t.Fatalf("overflow was not localized: %+v", audit.UnknownProducers)
	}
}

// Versioned synthetic directory snapshots keep corpus-shaped capacity tests
// small. Listings stream through fs.ReadDirFile; file content is never read.
type governanceReadViewCapacityDirectoryFiles struct {
	Files
	directories map[string][]fs.DirEntry
	opens       int
}

func (f *governanceReadViewCapacityDirectoryFiles) Open(name string) (fs.File, error) {
	if entries, ok := f.directories[filepath.Clean(name)]; ok {
		info, err := f.Files.Stat(name)
		if err != nil {
			return nil, err
		}
		f.opens++
		return &governanceReadViewProbeDirectoryFile{info: info, entries: entries}, nil
	}
	return f.Files.Open(name)
}

type governanceReadViewCapacityDirEntry struct {
	name string
	info fs.FileInfo
}

func (e governanceReadViewCapacityDirEntry) Name() string               { return e.name }
func (e governanceReadViewCapacityDirEntry) IsDir() bool                { return false }
func (e governanceReadViewCapacityDirEntry) Type() fs.FileMode          { return 0 }
func (e governanceReadViewCapacityDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

func governanceReadViewCapacityEntries(info fs.FileInfo, count int) []fs.DirEntry {
	entries := make([]fs.DirEntry, count)
	for i := range entries {
		entries[i] = governanceReadViewCapacityDirEntry{name: fmt.Sprintf("input-%05d", i), info: info}
	}
	return entries
}

func TestGovernanceReadViewProbeAuditAndReplayShareInputByteCapacity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.bin")
	payload := []byte("small bounded payload")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, overflow := range []bool{false, true} {
		t.Run(fmt.Sprintf("overflow=%t", overflow), func(t *testing.T) {
			base := &governanceReadViewProbeBudgetFiles{Files: NativeFiles()}
			view := newGovernanceReadView(root, base)
			if got, err := view.filesFor("capacity").ReadFile(path); err != nil || !bytes.Equal(got, payload) {
				t.Fatalf("capture failed: %q %v", got, err)
			}
			prior := int64(governanceReadViewMaxInputBytes - len(payload))
			if overflow {
				prior++
			}
			// Represent earlier roots' consumption, preserving real bounded I/O
			// for this root at the exact aggregate boundary and one byte beyond.
			before := base.readCalls
			audit := view.auditSingleWithBudget(&governanceReadViewAuditBudget{inputBytes: prior})
			if (len(audit.UnknownProducers["capacity"]) > 0) != overflow {
				t.Fatalf("audit disagreed with aggregate byte boundary: %+v", audit)
			}
			if overflow && base.readCalls != before || !overflow && base.readCalls == before {
				t.Fatalf("audit did not enforce budget before I/O: reads=%d/%d", base.readCalls, before)
			}
			_, proof := view.metadata("project", "revision", "index", audit)
			before = base.readCalls
			changed := proof.changesWithBudget(base, &governanceReadProofBudget{inputBytes: prior})
			if (len(changed) > 0) != overflow {
				t.Fatalf("proof replay disagreed with aggregate byte boundary: %v", changed)
			}
			if overflow && base.readCalls != before || !overflow && base.readCalls == before {
				t.Fatalf("proof replay did not enforce budget before I/O: reads=%d/%d", base.readCalls, before)
			}
			if base.readFileCalls != 0 {
				t.Fatal("audit or replay fell back to unbounded ReadFile")
			}
		})
	}
}
