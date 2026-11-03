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

// WithReviewOperationView shares only immutable header observations for this
// callback. Strict loading, ordering, trust and each worker's traversal remain
// live. Finalization rereads observed bytes before the read can be green.
// This is not an atomic filesystem snapshot or a write authorization cache.
func (s Store) WithReviewOperationView(run func(Store)) error {
	files := s.fsys()
	if !reflect.TypeOf(files).Comparable() {
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
	s.reviewReuse = nil // copied worker stores each start their own traversal
	defer view.closed.Store(true)
	run(s)
	// Join lazy initialization, if any; finalization must follow joined workers.
	view.once.Do(func() {})
	return view.validate()
}

func (v *reviewOperationView) boundTo(s Store) bool {
	files := s.fsys()
	return !v.closed.Load() && filepath.Clean(s.Root) == v.root &&
		s.FederatedProjectID == v.project && s.FederatedResolver == v.resolver &&
		reflect.TypeOf(files) == reflect.TypeOf(v.files) && reflect.TypeOf(files).Comparable() && files == v.files
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
