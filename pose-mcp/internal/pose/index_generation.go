package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// An index set is published and read as one generation (spec
// pose-index-generation-integrity, ADR 2026-10-17 "Index sets are published as
// verified generations").
//
// `pose index` writes seven derived files. Each file is published whole by
// PublishFile, but the seven are separate names, so a failure or a crash
// between two of them used to leave a mixture that every reader accepted: the
// repository map of one run next to the delivery graph of another.
//
// A generation is the seven files plus a descriptor, .pose/indexes/generation.json,
// that names each file by SHA-256 and binds the inputs it was computed from.
// The descriptor is published last, over the one the writer read, so the
// generation is committed exactly when it is complete; a reader that finds
// the descriptor reads each file, checks it against the descriptor and either
// has a whole, identified set or says why it has none.
//
// The bytes of a generation also live under .pose/indexes/generations/<id>/,
// immutable once written. A reader prefers that copy to the legacy name, so a
// publication in progress, which rewrites the legacy names before it commits,
// never changes what a reader that already fixed the generation sees. The
// legacy names stay current for tools that read them directly; they are what
// a clone without the (git-ignored) generations directory verifies against
// the descriptor.
const IndexGenerationSchemaVersion = 1

const (
	indexDescriptorName   = "generation.json"
	indexGenerationsDir   = "generations"
	indexGenerationPrefix = "gen-"
	// A reader that finds the descriptor replaced while it reads starts again,
	// this many times at most, then reports the set as changing.
	indexReadAttempts = 3
)

// Retention: the current generation and the newest others up to this many stay,
// and nothing younger than the grace period is removed, so a reader that is still
// copying a generation it fixed finds its files. Variables so tests can set them.
var (
	indexGenerationKeep  = 3
	indexGenerationGrace = 10 * time.Minute
)

// IndexGenerationFiles are the files of a generation, in the order `pose index`
// has always listed them.
var IndexGenerationFiles = []string{
	"repo-map.json", "services.json", "packages.json", "spec-graph.json",
	"roadmaps.json", "delivery-integrity.json", "releases.json",
}

// IndexGenerationStatus says what a reader may conclude from the indexes.
type IndexGenerationStatus string

const (
	// IndexGenerationCurrent: the descriptor is valid and every file read
	// matches it. The set is whole and identified.
	IndexGenerationCurrent IndexGenerationStatus = "current"
	// IndexGenerationLegacy: no descriptor exists. The files are whatever
	// they are; nothing here claims they belong together.
	IndexGenerationLegacy IndexGenerationStatus = "legacy-unversioned"
	// IndexGenerationUnavailable: a descriptor exists and the files do not
	// match it, or it cannot be read. Reasons are in IndexGenerationView.Reason.
	IndexGenerationUnavailable IndexGenerationStatus = "unavailable"
)

// ErrIndexGenerationUnavailable is wrapped by every error a reader returns for
// an unavailable set, so a caller can tell it from a missing file.
var ErrIndexGenerationUnavailable = errors.New("index-generation-unavailable")

// ErrIndexGenerationLegacy is returned by ReplaceIndex when there is no
// generation to replace a file in; the caller may keep writing the legacy file.
var ErrIndexGenerationLegacy = errors.New("index-generation-legacy")

// IndexGenerationFile names one file of a generation.
type IndexGenerationFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// IndexGenerationDescriptor is the commit record of a generation.
type IndexGenerationDescriptor struct {
	SchemaVersion int                   `json:"schema_version"`
	GenerationID  string                `json:"generation_id"`
	Files         []IndexGenerationFile `json:"files"`
	// Inputs binds the generation to what it was computed from: name to
	// "sha256:<hex>". Bindings are compared by IndexGenerationView.StaleInputs;
	// a different input does not make the set unavailable, it makes it stale.
	Inputs map[string]string `json:"inputs"`
}

// IndexGenerationView is what a reader got: the status, the descriptor it
// fixed and, for a current set, the bytes it read.
type IndexGenerationView struct {
	Status     IndexGenerationStatus
	Reason     string
	Descriptor IndexGenerationDescriptor
	Files      map[string][]byte
	// Diverged lists the legacy names (.pose/indexes/<file>) whose bytes are not
	// those of the generation although the generation is whole: the file was
	// edited by hand, or a publication is running and has not committed yet.
	// Engine readers read the generation, so they do not see the difference;
	// `pose index --verify` does, and fails on it.
	Diverged []string
}

// IndexGenerationReceipt is the outcome of one publication.
type IndexGenerationReceipt struct {
	GenerationID string
	Previous     string
	// Unchanged: the descriptor already named this generation and every file,
	// under its legacy name too, matched it, so nothing was written. A legacy
	// name that diverged (edited by hand, a publication that stopped before the
	// commit) is written again.
	Unchanged bool
	// CleanupError is the retention step failing after the generation was
	// committed; it never makes the publication a failure.
	CleanupError error
}

// IndexPublishFault and IndexPublishPhase let tests stop a publication between
// phases. They are unset in production.
var IndexPublishFault func(phase string) error

// IndexReadFault lets tests run something between a reader's read of the
// descriptor and its read of the files. It is unset in production.
var IndexReadFault func(phase string)

func indexFault(phase string) error {
	if IndexPublishFault != nil {
		return IndexPublishFault(phase)
	}
	return nil
}

func indexDir(root string) string { return filepath.Join(root, ".pose", "indexes") }

func indexGenerationDir(root, id string) string {
	return filepath.Join(indexDir(root), indexGenerationsDir, id)
}

var indexGenerationIDRE = regexp.MustCompile(`^gen-[0-9a-f]{16}$`)

func sha256Prefixed(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// NewIndexGenerationDescriptor builds the descriptor for these outputs. The
// identity is a digest of the file digests and the bindings, so two runs over
// the same inputs name the same generation and a retry converges on it.
func NewIndexGenerationDescriptor(outputs map[string][]byte, inputs map[string]string) (IndexGenerationDescriptor, error) {
	if len(outputs) != len(IndexGenerationFiles) {
		return IndexGenerationDescriptor{}, fmt.Errorf("pose: an index generation has exactly %d files, got %d", len(IndexGenerationFiles), len(outputs))
	}
	files := make([]IndexGenerationFile, 0, len(IndexGenerationFiles))
	for _, name := range IndexGenerationFiles {
		raw, ok := outputs[name]
		if !ok {
			return IndexGenerationDescriptor{}, fmt.Errorf("pose: index generation is missing %s", name)
		}
		files = append(files, IndexGenerationFile{Name: name, SHA256: sha256Prefixed(raw), Bytes: len(raw)})
	}
	bound := map[string]string{}
	for key, value := range inputs {
		bound[key] = value
	}
	identity, err := json.Marshal(struct {
		Files  []IndexGenerationFile `json:"files"`
		Inputs map[string]string     `json:"inputs"`
	}{files, bound})
	if err != nil {
		return IndexGenerationDescriptor{}, err
	}
	sum := sha256.Sum256(identity)
	return IndexGenerationDescriptor{
		SchemaVersion: IndexGenerationSchemaVersion,
		GenerationID:  indexGenerationPrefix + hex.EncodeToString(sum[:8]),
		Files:         files,
		Inputs:        bound,
	}, nil
}

func (d IndexGenerationDescriptor) bytes() ([]byte, error) {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func (d IndexGenerationDescriptor) file(name string) (IndexGenerationFile, bool) {
	for _, f := range d.Files {
		if f.Name == name {
			return f, true
		}
	}
	return IndexGenerationFile{}, false
}

// validate refuses a descriptor that is not the one this engine writes: another
// schema, another file set, a digest that is not a SHA-256, an id that is not
// the digest of the content.
func (d IndexGenerationDescriptor) validate() error {
	if d.SchemaVersion != IndexGenerationSchemaVersion {
		return fmt.Errorf("schema_version %d is not supported (this engine reads %d)", d.SchemaVersion, IndexGenerationSchemaVersion)
	}
	if len(d.Files) != len(IndexGenerationFiles) {
		return fmt.Errorf("the descriptor lists %d files, a generation has %d", len(d.Files), len(IndexGenerationFiles))
	}
	for i, name := range IndexGenerationFiles {
		f := d.Files[i]
		if f.Name != name {
			return fmt.Errorf("file %d is %q, expected %q", i, f.Name, name)
		}
		if !strings.HasPrefix(f.SHA256, "sha256:") || len(f.SHA256) != len("sha256:")+64 {
			return fmt.Errorf("%s has no SHA-256 digest", name)
		}
	}
	if !indexGenerationIDRE.MatchString(d.GenerationID) {
		return fmt.Errorf("generation_id %q is malformed", d.GenerationID)
	}
	identity, err := json.Marshal(struct {
		Files  []IndexGenerationFile `json:"files"`
		Inputs map[string]string     `json:"inputs"`
	}{d.Files, nilToEmpty(d.Inputs)})
	if err != nil {
		return err
	}
	sum := sha256.Sum256(identity)
	if want := indexGenerationPrefix + hex.EncodeToString(sum[:8]); want != d.GenerationID {
		return fmt.Errorf("generation_id %s does not match the descriptor content (%s)", d.GenerationID, want)
	}
	return nil
}

func nilToEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// readDescriptor reads the commit record. exists is false when there is none.
func readIndexDescriptor(root string) (desc IndexGenerationDescriptor, raw []byte, exists bool, err error) {
	raw, readErr := os.ReadFile(filepath.Join(indexDir(root), indexDescriptorName))
	if errors.Is(readErr, os.ErrNotExist) {
		return IndexGenerationDescriptor{}, nil, false, nil
	}
	if readErr != nil {
		return IndexGenerationDescriptor{}, nil, true, fmt.Errorf("the generation descriptor cannot be read: %w", readErr)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&desc); err != nil {
		return IndexGenerationDescriptor{}, raw, true, fmt.Errorf("the generation descriptor is not valid JSON of the expected shape: %w", err)
	}
	if dec.More() {
		return IndexGenerationDescriptor{}, raw, true, errors.New("the generation descriptor has data after its value")
	}
	if err := desc.validate(); err != nil {
		return IndexGenerationDescriptor{}, raw, true, fmt.Errorf("the generation descriptor is invalid: %w", err)
	}
	return desc, raw, true, nil
}

// readGenerationFile returns the bytes of one file of the generation that match
// its digest: the immutable copy when it exists and matches, the legacy name
// when that matches. When neither matches it returns the bytes it found, so the
// caller can say what differed, and ok false.
func readGenerationFile(root, id, name, want string) (data []byte, ok bool, err error) {
	var lastErr error
	for _, path := range []string{filepath.Join(indexGenerationDir(root, id), name), filepath.Join(indexDir(root), name)} {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			if lastErr == nil || !errors.Is(readErr, os.ErrNotExist) {
				lastErr = readErr
			}
			continue
		}
		if sha256Prefixed(raw) == want {
			return raw, true, nil
		}
		data, lastErr = raw, nil
	}
	return data, false, lastErr
}

// ReadIndexGeneration fixes the generation named by the descriptor and returns
// every file of it, each checked against the descriptor. It starts again if the
// descriptor is replaced while it reads, a bounded number of times.
func (s Store) ReadIndexGeneration() IndexGenerationView {
	for attempt := 0; attempt < indexReadAttempts; attempt++ {
		desc, raw, exists, err := readIndexDescriptor(s.Root)
		if err != nil {
			return IndexGenerationView{Status: IndexGenerationUnavailable, Reason: err.Error()}
		}
		if !exists {
			return IndexGenerationView{Status: IndexGenerationLegacy, Reason: "no generation descriptor: the indexes were not published as a set"}
		}
		if IndexReadFault != nil {
			IndexReadFault("after-descriptor")
		}
		files := map[string][]byte{}
		diverged := []string{}
		var problem string
		for _, f := range desc.Files {
			data, matched, readErr := readGenerationFile(s.Root, desc.GenerationID, f.Name, f.SHA256)
			if readErr != nil {
				problem = fmt.Sprintf("%s of generation %s cannot be read: %v", f.Name, desc.GenerationID, readErr)
				break
			}
			if !matched {
				problem = fmt.Sprintf("%s does not match generation %s (descriptor %s, file %s)", f.Name, desc.GenerationID, f.SHA256, sha256Prefixed(data))
				break
			}
			files[f.Name] = data
			if legacy, legacyErr := os.ReadFile(filepath.Join(indexDir(s.Root), f.Name)); legacyErr != nil || sha256Prefixed(legacy) != f.SHA256 {
				diverged = append(diverged, f.Name)
			}
		}
		again, againRaw, _, againErr := readIndexDescriptor(s.Root)
		if againErr == nil && again.GenerationID == desc.GenerationID && string(againRaw) == string(raw) {
			if problem != "" {
				return IndexGenerationView{Status: IndexGenerationUnavailable, Reason: problem, Descriptor: desc}
			}
			return IndexGenerationView{Status: IndexGenerationCurrent, Descriptor: desc, Files: files, Diverged: diverged}
		}
	}
	return IndexGenerationView{Status: IndexGenerationUnavailable, Reason: "the generation descriptor kept changing while it was read"}
}

// ReadIndex returns the bytes of one index and what may be concluded from them.
// For a published set the file is checked against the descriptor, so a file that
// belongs to another generation is an error that wraps ErrIndexGenerationUnavailable.
// For a repository that never published a set the file is returned as before,
// with status IndexGenerationLegacy and no claim of coherence. A missing file is
// os.ErrNotExist in both cases.
func (s Store) ReadIndex(name string) ([]byte, IndexGenerationStatus, error) {
	known := false
	for _, n := range IndexGenerationFiles {
		known = known || n == name
	}
	if !known {
		return nil, "", fmt.Errorf("pose: %s is not an index of a generation", name)
	}
	for attempt := 0; attempt < indexReadAttempts; attempt++ {
		desc, descRaw, exists, err := readIndexDescriptor(s.Root)
		if err != nil {
			return nil, IndexGenerationUnavailable, fmt.Errorf("pose: %w: %s", ErrIndexGenerationUnavailable, err)
		}
		if !exists {
			raw, readErr := os.ReadFile(filepath.Join(indexDir(s.Root), name))
			return raw, IndexGenerationLegacy, readErr
		}
		f, _ := desc.file(name)
		if IndexReadFault != nil {
			IndexReadFault("after-descriptor")
		}
		data, matched, readErr := readGenerationFile(s.Root, desc.GenerationID, name, f.SHA256)
		again, againRaw, _, againErr := readIndexDescriptor(s.Root)
		if againErr != nil || again.GenerationID != desc.GenerationID || string(againRaw) != string(descRaw) {
			continue
		}
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				return nil, IndexGenerationUnavailable, fmt.Errorf("pose: %w: %s of generation %s is missing (%w)", ErrIndexGenerationUnavailable, name, desc.GenerationID, readErr)
			}
			return nil, IndexGenerationUnavailable, fmt.Errorf("pose: %w: %s: %v", ErrIndexGenerationUnavailable, name, readErr)
		}
		if !matched {
			return nil, IndexGenerationUnavailable, fmt.Errorf("pose: %w: %s does not match generation %s; run `pose index`", ErrIndexGenerationUnavailable, name, desc.GenerationID)
		}
		return data, IndexGenerationCurrent, nil
	}
	return nil, IndexGenerationUnavailable, fmt.Errorf("pose: %w: the generation descriptor kept changing while %s was read", ErrIndexGenerationUnavailable, name)
}

// PublishIndexGeneration writes the seven outputs as one generation. Phases:
//  1. the files are written under generations/<id>/, each exclusively, and read
//     back against the descriptor;
//  2. the legacy names are published from the same bytes;
//  3. the descriptor is published over the one the writer read: this is the commit.
//
// A failure before step 3 leaves the previous descriptor, and so the previous
// generation, in force; the legacy names may already differ from it, which a
// reader that falls back to them reports as unavailable instead of mixing.
// Retention runs after the commit and cannot fail it.
func (s Store) PublishIndexGeneration(outputs map[string][]byte, inputs map[string]string) (IndexGenerationReceipt, error) {
	desc, err := NewIndexGenerationDescriptor(outputs, inputs)
	if err != nil {
		return IndexGenerationReceipt{}, err
	}
	descBytes, err := desc.bytes()
	if err != nil {
		return IndexGenerationReceipt{}, err
	}
	prev, prevRaw, prevExists, prevErr := readIndexDescriptor(s.Root)
	receipt := IndexGenerationReceipt{GenerationID: desc.GenerationID}
	if prevExists && prevErr == nil {
		receipt.Previous = prev.GenerationID
	}
	// A descriptor that cannot be read is replaced like any other record the
	// engine owns, over the bytes that were found.
	expect := ExpectBytes(prevRaw, prevExists)

	if view := s.ReadIndexGeneration(); prevExists && prevErr == nil && prev.GenerationID == desc.GenerationID && view.Status == IndexGenerationCurrent && len(view.Diverged) == 0 {
		receipt.Unchanged = true
		receipt.CleanupError = s.pruneIndexGenerations(desc.GenerationID)
		return receipt, nil
	}

	if err := indexFault("before-generation"); err != nil {
		return receipt, err
	}
	dir := indexGenerationDir(s.Root, desc.GenerationID)
	for _, f := range desc.Files {
		if err := PublishFile(filepath.Join(dir, f.Name), outputs[f.Name], 0o644, PublishDerived, PublishExpectation{Mode: ExpectAbsent}); err != nil {
			return receipt, fmt.Errorf("pose: publishing %s of generation %s: %w", f.Name, desc.GenerationID, err)
		}
		if err := indexFault("after-generation-file:" + f.Name); err != nil {
			return receipt, err
		}
	}
	for _, f := range desc.Files {
		back, err := os.ReadFile(filepath.Join(dir, f.Name))
		if err != nil || sha256Prefixed(back) != f.SHA256 {
			return receipt, fmt.Errorf("pose: %s of generation %s does not read back as published", f.Name, desc.GenerationID)
		}
	}
	if err := indexFault("before-legacy"); err != nil {
		return receipt, err
	}
	for _, f := range desc.Files {
		if err := PublishFile(filepath.Join(indexDir(s.Root), f.Name), outputs[f.Name], 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny}); err != nil {
			return receipt, fmt.Errorf("pose: publishing %s: %w", f.Name, err)
		}
		if err := indexFault("after-legacy-file:" + f.Name); err != nil {
			return receipt, err
		}
	}
	if err := indexFault("before-commit"); err != nil {
		return receipt, err
	}
	if err := PublishFile(filepath.Join(indexDir(s.Root), indexDescriptorName), descBytes, 0o644, PublishAuthoritative, expect); err != nil {
		return receipt, fmt.Errorf("pose: committing generation %s: %w", desc.GenerationID, err)
	}
	receipt.CleanupError = s.pruneIndexGenerations(desc.GenerationID)
	return receipt, nil
}

// ReplaceIndex publishes a new generation in which one file is replaced and the
// others are those of the current generation, all verified. It refuses to build
// on a set that is not current. When there is no generation yet it returns
// ErrIndexGenerationLegacy and writes nothing, so the caller may keep writing the
// file as it did before sets existed.
func (s Store) ReplaceIndex(name string, data []byte) (IndexGenerationReceipt, error) {
	view := s.ReadIndexGeneration()
	switch view.Status {
	case IndexGenerationLegacy:
		return IndexGenerationReceipt{}, ErrIndexGenerationLegacy
	case IndexGenerationUnavailable:
		return IndexGenerationReceipt{}, fmt.Errorf("pose: %w: %s", ErrIndexGenerationUnavailable, view.Reason)
	}
	if _, ok := view.Files[name]; !ok {
		return IndexGenerationReceipt{}, fmt.Errorf("pose: %s is not an index of a generation", name)
	}
	outputs := map[string][]byte{}
	for key, value := range view.Files {
		outputs[key] = value
	}
	outputs[name] = data
	return s.PublishIndexGeneration(outputs, view.Descriptor.Inputs)
}

// StaleInputs lists the bindings of the generation whose current value differs
// from the one it was computed from, and the bindings it cannot recompute. The
// list is empty for a set computed from the inputs as they are. Compute maps a
// binding name to its current digest.
func (v IndexGenerationView) StaleInputs(compute func(name string) (string, bool)) []string {
	stale := []string{}
	names := make([]string, 0, len(v.Descriptor.Inputs))
	for name := range v.Descriptor.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		now, ok := compute(name)
		if !ok || now != v.Descriptor.Inputs[name] {
			stale = append(stale, name)
		}
	}
	return stale
}

// pruneIndexGenerations removes generations beyond the retention window. It
// touches only directories of the form gen-<16 hex> directly under the
// generations directory, never follows a link, never removes the current
// generation, and leaves anything younger than the grace period.
func (s Store) pruneIndexGenerations(current string) error {
	base := filepath.Join(indexDir(s.Root), indexGenerationsDir)
	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	type generation struct {
		name string
		mod  time.Time
	}
	others := []generation{}
	for _, e := range entries {
		if !indexGenerationIDRE.MatchString(e.Name()) || e.Name() == current {
			continue
		}
		info, err := os.Lstat(filepath.Join(base, e.Name()))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		others = append(others, generation{e.Name(), info.ModTime()})
	}
	sort.Slice(others, func(i, j int) bool { return others[i].mod.After(others[j].mod) })
	var firstErr error
	for i, g := range others {
		if i < indexGenerationKeep-1 || time.Since(g.mod) < indexGenerationGrace {
			continue
		}
		if err := os.RemoveAll(filepath.Join(base, g.name)); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// IndexInputNames are the bindings of a generation: the governed inputs `pose
// index` computes the seven files from, besides the Git history that the
// delivery graph reads and that a descriptor cannot pin without changing on every
// commit.
var IndexInputNames = []string{"specs", "roadmaps", "policy", "module-metadata", "validation-matrix"}

// IndexInputBinding returns the current digest of one binding. A directory is
// digested by the paths and bytes of the regular files under it, in path order;
// a missing input has the digest of absence, so its appearance is a change.
func IndexInputBinding(root, name string) (string, bool) {
	switch name {
	case "specs":
		return treeDigest(root, ".pose/specs"), true
	case "roadmaps":
		return treeDigest(root, ".pose/roadmaps"), true
	case "policy":
		return treeDigest(root, ".pose/policy"), true
	case "module-metadata":
		return fileDigest(root, ".pose/indexes/module-metadata.json"), true
	case "validation-matrix":
		return fileDigest(root, ".pose/indexes/validation-matrix.json"), true
	}
	return "", false
}

// IndexInputBindings returns every binding a generation carries.
func IndexInputBindings(root string) map[string]string {
	out := map[string]string{}
	for _, name := range IndexInputNames {
		out[name], _ = IndexInputBinding(root, name)
	}
	return out
}

func fileDigest(root, rel string) string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "absent"
	}
	return sha256Prefixed(raw)
}

func treeDigest(root, rel string) string {
	base := filepath.Join(root, filepath.FromSlash(rel))
	paths := []string{}
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && !IsPublisherStaging(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if len(paths) == 0 {
		return "absent"
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		relPath, _ := filepath.Rel(base, path)
		h.Write([]byte(filepath.ToSlash(relPath)))
		h.Write([]byte{0})
		h.Write(raw)
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
