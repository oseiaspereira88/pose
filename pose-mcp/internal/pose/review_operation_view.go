package pose

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

type reviewHeaderObservation struct {
	owner      string
	loss       *SourceLoss
	digest     [32]byte
	readError  string
	inputBytes int
}

// readReviewBundleHeader is the same classification for cached and live reads.
// A known different owner is skippable only after whole-JSON and ambiguity checks.
func readReviewBundleHeader(files Files, dir, name string) reviewHeaderObservation {
	rel := filepath.ToSlash(filepath.Join(".pose", "review-bundles", name))
	raw, err := files.ReadFile(filepath.Join(dir, name))
	if err != nil {
		loss := SourceLoss{Producer: "review-bundles", Artifact: rel, Kind: LossReadError, Detail: err.Error()}
		return reviewHeaderObservation{loss: &loss, readError: err.Error()}
	}
	result := reviewHeaderObservation{digest: sha256.Sum256(raw), inputBytes: len(raw)}
	var header struct {
		Payload struct {
			Scope struct {
				Ref string `json:"ref"`
			} `json:"scope"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		loss := attributeLoss(SourceLoss{Producer: "review-bundles", Artifact: rel, Kind: LossParseError, Detail: "the record is not valid JSON"}, raw, "payload", "scope", "ref")
		result.loss = &loss
	} else if _, _, ambiguous := sniffJSONString(raw, "payload", "scope", "ref"); ambiguous {
		loss := SourceLoss{Producer: "review-bundles", Artifact: rel, Kind: LossInvalid, Detail: "the record names its scope more than once"}
		result.loss = &loss
	} else if header.Payload.Scope.Ref == "" {
		loss := SourceLoss{Producer: "review-bundles", Artifact: rel, Kind: LossParseError, Detail: "the record names no scope"}
		result.loss = &loss
	} else {
		result.owner = header.Payload.Scope.Ref
	}
	return result
}

type reviewOperationView struct {
	specOnce        sync.Once
	specUsed        bool
	specErr         error
	specFallback    bool
	specEntries     []Spec
	specBySlug      map[string]int
	specBodyMu      sync.Mutex
	specBodySeen    []bool
	specBodyDigests [][32]byte
	specConflict    bool
	root            string
	files           Files
	project         string
	resolver        *ArtifactResolver
	once            sync.Once
	closed          atomic.Bool
	used            bool // published by once; consumers call once before inspecting it
	directoryError  string
	inventory       [32]byte
	headers         map[string]reviewHeaderObservation
	fallback        bool
	maxRecords      int
	maxInputBytes   int
	maxRetainedText int
}

// WithReviewOperationView shares immutable headers and spec metadata for this
// callback. Strict loading, ordering, trust and each worker's traversal remain
// live. Finalization rereads observed bytes before the read can be green.
// This is not an atomic filesystem snapshot or a write authorization cache.
func (s Store) WithReviewOperationView(run func(Store)) error {
	s.specs = nil
	s.reviewReuse = nil
	s.reviewView = nil
	files := s.fsys()
	if !reflect.ValueOf(files).Comparable() {
		// Unknown identity cannot safely key an optimization. Keep live semantics.
		run(s)
		return nil
	}
	view := &reviewOperationView{root: filepath.Clean(s.Root), files: files,
		project: s.FederatedProjectID, resolver: s.FederatedResolver,
		maxRecords: 10000, maxInputBytes: 128 * 1024 * 1024, maxRetainedText: 8 * 1024 * 1024}
	return s.withReviewOperationView(view, run)
}

func (s Store) withReviewOperationView(view *reviewOperationView, run func(Store)) error {
	s.reviewView = view
	s.specs = nil       // an earlier advisory listing is not this operation's capture
	s.reviewReuse = nil // copied worker stores each start their own traversal
	defer view.closed.Store(true)
	run(s)
	// Join lazy initialization, if any; finalization must follow joined workers.
	view.once.Do(func() {})
	view.specOnce.Do(func() {})
	if err := view.validateSpecs(); err != nil {
		return err
	}
	return view.validate()
}

func (v *reviewOperationView) captureSpecs(s Store) {
	v.specOnce.Do(func() {
		v.specUsed = true
		entries, err := s.listSpecs("", "")
		v.specErr = err
		if err != nil {
			return
		}
		if len(entries) > v.maxRecords {
			v.specFallback = true
			return
		}
		retained := 0
		for _, sp := range entries {
			raw, _ := json.Marshal(sp)
			retained += len(raw)
			if retained > v.maxRetainedText {
				v.specFallback = true
				return
			}
		}
		v.specEntries = entries
		v.specBySlug = make(map[string]int, len(entries))
		for i, sp := range entries {
			if _, exists := v.specBySlug[sp.Slug]; exists {
				v.specBySlug[sp.Slug] = -1
			} else {
				v.specBySlug[sp.Slug] = i
			}
		}
		v.specBodySeen = make([]bool, len(entries))
		v.specBodyDigests = make([][32]byte, len(entries))
	})
}

func (s Store) reviewViewSpec(slug string) (*Spec, bool, error) {
	v := s.reviewView
	if v == nil || !v.boundTo(s) {
		return nil, false, nil
	}
	v.captureSpecs(s)
	if v.specFallback {
		return nil, false, nil
	}
	if v.specErr != nil {
		return nil, true, v.specErr
	}
	index, exists := v.specBySlug[slug]
	if !exists {
		return nil, true, fmt.Errorf("pose: spec %q not found", slug)
	}
	if index < 0 {
		return nil, true, SpecIdentityConflictError{Slug: slug}
	}
	full, err := s.readListedSpec(cloneSpec(v.specEntries[index]))
	return full, true, err
}

func (s Store) reviewViewSpecs(status, components string) ([]Spec, bool, error) {
	v := s.reviewView
	if v == nil || !v.boundTo(s) {
		return nil, false, nil
	}
	v.captureSpecs(s)
	if v.specFallback {
		return nil, false, nil
	}
	if v.specErr != nil {
		return nil, true, v.specErr
	}
	wantedStatus, wantedComponents := splitStatusFilter(status), splitCommaFilter(components)
	result := []Spec{}
	for _, sp := range v.specEntries {
		if len(wantedStatus) > 0 && !statusMatchesAny(sp.Status, wantedStatus) {
			continue
		}
		if len(wantedComponents) > 0 && !componentsMatchAny(sp.Components, wantedComponents) {
			continue
		}
		result = append(result, cloneSpec(sp))
	}
	return result, true, nil
}

func (s Store) observeReviewSpecBody(listed Spec, full *Spec) error {
	v := s.reviewView
	if v == nil || !v.boundTo(s) {
		return nil
	}
	v.captureSpecs(s)
	if v.specFallback {
		return nil
	}
	if v.specErr != nil {
		return v.specErr
	}
	index, exists := v.specBySlug[listed.Slug]
	metadata := *full
	metadata.Body = ""
	digest := sha256.Sum256([]byte(full.Body))
	v.specBodyMu.Lock()
	defer v.specBodyMu.Unlock()
	if !exists || index < 0 || !reflect.DeepEqual(metadata, v.specEntries[index]) || (v.specBodySeen[index] && v.specBodyDigests[index] != digest) {
		v.specConflict = true
		return fmt.Errorf("pose: review inputs changed during operation: spec %s", listed.Slug)
	}
	v.specBodySeen[index] = true
	v.specBodyDigests[index] = digest
	return nil
}

func (v *reviewOperationView) validateSpecs() error {
	if !v.specUsed || v.specFallback {
		return nil
	}
	store := Store{Root: v.root, FS: v.files}
	current, err := store.listSpecs("", "")
	if v.specErr != nil {
		if err != nil && err.Error() == v.specErr.Error() {
			return nil
		}
		return fmt.Errorf("pose: review inputs changed during operation: spec catalogue availability")
	}
	if err != nil {
		return fmt.Errorf("pose: review inputs changed during operation: %w", err)
	}
	if !reflect.DeepEqual(current, v.specEntries) {
		return fmt.Errorf("pose: review inputs changed during operation: spec catalogue")
	}
	v.specBodyMu.Lock()
	conflict := v.specConflict
	seen := append([]bool(nil), v.specBodySeen...)
	digests := append([][32]byte(nil), v.specBodyDigests...)
	v.specBodyMu.Unlock()
	if conflict {
		return fmt.Errorf("pose: review inputs changed during operation: consumed spec body")
	}
	for i, listed := range current {
		if !seen[i] {
			continue
		}
		full, err := store.readListedSpec(listed)
		if err != nil {
			return fmt.Errorf("pose: review inputs changed during operation: %w", err)
		}
		digest := sha256.Sum256([]byte(full.Body))
		full.Body = ""
		if digest != digests[i] || !reflect.DeepEqual(*full, listed) {
			return fmt.Errorf("pose: review inputs changed during operation: spec %s", listed.Slug)
		}
	}
	current, err = store.listSpecs("", "")
	if err != nil || !reflect.DeepEqual(current, v.specEntries) {
		return fmt.Errorf("pose: review inputs changed during operation: spec catalogue after body audit")
	}
	return nil
}

func (v *reviewOperationView) boundTo(s Store) bool {
	files := s.fsys()
	return !v.closed.Load() && filepath.Clean(s.Root) == v.root &&
		s.FederatedProjectID == v.project && s.FederatedResolver == v.resolver &&
		reflect.TypeOf(files) == reflect.TypeOf(v.files) && reflect.ValueOf(files).Comparable() && reflect.ValueOf(v.files).Comparable() && files == v.files
}

func reviewHeaderInventory(entries []os.DirEntry) [32]byte {
	h := sha256.New()
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			fmt.Fprintf(h, "%s\x00%d\x00", entry.Name(), entry.Type())
		}
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func (v *reviewOperationView) observeDirectoryError(s Store, err error) {
	if v.boundTo(s) {
		v.once.Do(func() { v.used = true; v.directoryError = err.Error() })
	}
}

func (s Store) reviewBundleHeaders(dir string, entries []os.DirEntry) (map[string]reviewHeaderObservation, error) {
	v := s.reviewView
	if v == nil || !v.boundTo(s) {
		return nil, nil
	}
	v.once.Do(func() {
		v.used = true
		v.inventory = reviewHeaderInventory(entries)
		headers := map[string]reviewHeaderObservation{}
		inputBytes, textBytes := 0, 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			if len(headers) >= v.maxRecords {
				v.fallback = true
				return
			}
			observation := readReviewBundleHeader(v.files, dir, entry.Name())
			inputBytes += observation.inputBytes
			textBytes += len(entry.Name()) + len(observation.owner) + len(observation.readError)
			if observation.loss != nil {
				raw, _ := json.Marshal(observation.loss)
				textBytes += len(raw)
			}
			if inputBytes > v.maxInputBytes || textBytes > v.maxRetainedText {
				v.fallback = true
				return
			}
			headers[entry.Name()] = observation
		}
		v.headers = headers // publish only the complete immutable catalogue
	})
	if v.directoryError != "" || v.inventory != reviewHeaderInventory(entries) {
		return nil, fmt.Errorf("pose: review inputs changed during operation: bundle directory")
	}
	if v.fallback {
		return nil, nil
	}
	return v.headers, nil
}

func (v *reviewOperationView) validate() error {
	if !v.used {
		return nil
	}
	dir, err := ensureReviewArtifactDirIn(v.files, v.root, ".pose/review-bundles", false)
	var entries []os.DirEntry
	if err == nil {
		entries, err = v.files.ReadDir(dir)
	}
	if v.directoryError != "" {
		if err != nil && err.Error() == v.directoryError {
			return nil
		}
		return fmt.Errorf("pose: review inputs changed during operation: bundle directory availability")
	}
	if err != nil {
		return fmt.Errorf("pose: review inputs changed during operation: %w", err)
	}
	if reviewHeaderInventory(entries) != v.inventory {
		return fmt.Errorf("pose: review inputs changed during operation: bundle inventory")
	}
	if v.fallback {
		return nil
	} // no header observations were reused
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		before := v.headers[entry.Name()]
		raw, err := v.files.ReadFile(filepath.Join(dir, entry.Name()))
		if before.readError != "" {
			if err != nil && err.Error() == before.readError {
				continue
			}
			return fmt.Errorf("pose: review inputs changed during operation: %s availability", entry.Name())
		}
		if err != nil {
			return fmt.Errorf("pose: review inputs changed during operation: %s: %w", entry.Name(), err)
		}
		if sha256.Sum256(raw) != before.digest {
			return fmt.Errorf("pose: review inputs changed during operation: %s bytes", entry.Name())
		}
	}
	return nil
}
