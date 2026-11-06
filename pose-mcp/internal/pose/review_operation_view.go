package pose

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
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
	coordinator      *reviewOperationView
	coordinatorStore Store
	scopeBase        Store
	scopeRoots       map[string]string
	scopeConflicts   map[string]bool
	scopeInit        sync.Once
	scopeInitErr     error

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
	contextConflict atomic.Bool
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
	baseFiles := s.fsys()
	keyable := reflect.ValueOf(baseFiles).Comparable()
	base, readView := s.beginGovernanceReadView()
	shared := base.withGovernanceReadView(readView, "review-operation")
	view := &reviewOperationView{root: filepath.Clean(shared.Root), files: shared.fsys(),
		project: shared.FederatedProjectID, resolver: shared.FederatedResolver,
		maxRecords: 10000, maxInputBytes: 128 * 1024 * 1024, maxRetainedText: 8 * 1024 * 1024}
	var operationErr error
	if !keyable {
		// Unknown identity cannot safely key the bounded optimization. The
		// operation still reads through the governance view and receives its
		// final closure audit; it simply falls back to live catalogue traversal.
		shared.reviewView = nil
		shared.specs = nil
		shared.reviewReuse = nil
		run(shared)
	} else {
		operationErr = shared.withReviewOperationView(view, run)
	}
	audit := readView.audit()
	return errors.Join(operationErr, reviewOperationGovernanceAuditError(audit, readView, view))
}

func reviewOperationGovernanceAuditError(audit governanceReadViewAudit, readView *governanceReadView, operationView *reviewOperationView) error {
	var result error
	changed := audit.ChangedProducers["review-operation"] || audit.ChangedProducers["projection"]
	if changed {
		paths := append([]string(nil), audit.ChangedPaths...)
		if len(paths) == 0 {
			paths = []string{"input closure"}
		}
		result = fmt.Errorf("pose: review inputs changed during operation: %s: %w", strings.Join(uniqueSorted(paths), ", "), ErrGovernanceReadViewChanged)
	}
	// The operation view certifies the bytes and metadata its review producers
	// consumed. Projection-only context such as an unavailable HEAD does not
	// become a write authorization or a current-generation claim here; any
	// unprovable read attributed to this operation still fails closed.
	if reasons := audit.UnknownProducers["review-operation"]; len(reasons) > 0 && !reviewOperationReadLossIsCovered(audit, readView, operationView) {
		result = errors.Join(result, fmt.Errorf("pose: review input closure is unknown: %s: %w", strings.Join(uniqueSorted(reasons), "; "), ErrGovernanceReadViewUnknown))
	}
	return result
}

// reviewOperationReadLossIsCovered keeps a stable, unreadable bundle in the
// operation result as source-coverage loss. ReadReviewBundles reports that
// loss to its caller, and validate() proves the same file remains unreadable;
// this wrapper should not turn a localized record loss into a second API error.
// Other unknown inputs still make the operation result unavailable.
func reviewOperationReadLossIsCovered(audit governanceReadViewAudit, readView *governanceReadView, operationView *reviewOperationView) bool {
	if readView == nil || operationView == nil || !operationView.used || operationView.fallback {
		return false
	}
	for _, reason := range audit.UnknownProducers["review-operation"] {
		if reason != "input bytes could not be read" {
			return false
		}
	}
	readView.mu.Lock()
	defer readView.mu.Unlock()
	covered := map[string]string{}
	for name, observation := range operationView.headers {
		if observation.readError == "" {
			continue
		}
		if observation.loss == nil || observation.loss.Kind != LossReadError {
			return false
		}
		covered[filepath.Base(name)] = observation.readError
	}
	if len(covered) == 0 {
		return false
	}
	matched := 0
	for path, observation := range readView.files {
		if !observation.producers["review-operation"] || observation.err == nil || os.IsNotExist(observation.err) {
			continue
		}
		rel, err := filepath.Rel(operationView.root, path)
		if err != nil || !strings.HasPrefix(filepath.ToSlash(rel), ".pose/review-bundles/") {
			return false
		}
		name := filepath.Base(path)
		if covered[name] == "" || covered[name] != observation.errText {
			return false
		}
		delete(covered, name)
		matched++
	}
	return matched > 0 && len(covered) == 0
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
	return errors.Join(view.validateSpecs(), view.validate())
}

// prepareReviewOperationRead preserves Store's live-read behavior after an
// operation view retires, while refusing to borrow an active view under a
// different root, project, resolver, or Files binding.
func (s Store) prepareReviewOperationRead() (Store, error) {
	if s.reviewView != nil && s.reviewView.closed.Load() {
		s = s.withoutGovernanceReadView()
	}
	readView, viewFiles := governanceReadViewFromFiles(s.fsys())
	if readView != nil {
		readView.mu.Lock()
		finalized := readView.closed
		readView.mu.Unlock()
		if finalized {
			s = s.withoutGovernanceReadView()
			readView, viewFiles = governanceReadViewFromFiles(s.fsys())
		}
	}
	if s.reviewView != nil && !s.reviewView.closed.Load() && !s.reviewView.boundTo(s) {
		return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
	}
	if readView != nil && viewFiles != nil {
		if s.FederatedProjectID != "" && viewFiles.project != "" && s.FederatedProjectID != viewFiles.project {
			return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
		}
		if readView.operation == nil {
			if filepath.Clean(s.Root) != filepath.Clean(readView.root) {
				return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
			}
			// A same-root advisory view may be attached directly to a local Store
			// without an operation coordinator. Keep that local read path usable,
			// but do not infer a federated identity or borrow a resolver binding.
			if s.FederatedProjectID != viewFiles.project || s.FederatedResolver != nil || viewFiles.project != "" {
				return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
			}
		} else {
			// StoreFor(project) deliberately returns an unscoped Store whose Files
			// wrapper still carries the primary read view. Route by the captured
			// project/root vector first, then verify the selected root itself;
			// comparing against readView.root here would reject every valid child.
			routeProject := viewFiles.project
			if routeProject == "" {
				routeProject = s.FederatedProjectID
			}
			if routeProject == "" {
				routeProject = readView.operation.mainProject
			}
			if !sameLexicalRoot(readView.operation.initialRoots[routeProject], s.Root) {
				return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
			}
			target := readView.operation.route(s.Root, viewFiles.producer, routeProject)
			if target == nil || !sameLexicalRoot(target.root, s.Root) {
				return Store{}, s.refuseReviewOperationContext(readView, viewFiles)
			}
		}
	}
	return s, nil
}

func (s Store) refuseReviewOperationContext(readView *governanceReadView, viewFiles *governanceReadViewFiles) error {
	if s.reviewView != nil && !s.reviewView.closed.Load() {
		s.reviewView.contextConflict.Store(true)
	}
	if readView != nil {
		producer := "review-operation"
		if viewFiles != nil && viewFiles.producer != "" {
			producer = viewFiles.producer
		}
		readView.markUnknown(producer, "review Store context no longer matches the authorized operation binding")
	}
	return fmt.Errorf("pose: review Store context changed: %w", errGovernanceReadViewOutsideRoot)
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
	if v.coordinator != nil {
		if v.specConflict {
			return fmt.Errorf("pose: review inputs changed during operation: consumed spec body: %w", ErrGovernanceReadViewChanged)
		}
		return nil
	}
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
	if v.contextConflict.Load() {
		return fmt.Errorf("pose: review inputs changed during operation: Store context")
	}
	if v.coordinator != nil || !v.used {
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

// WithReviewCoordinator audits shared selection and catalogue inputs after all
// scope callbacks join. Findings must be buffered until this method returns.
// Scope operations certify their own closures, not a simultaneous batch snapshot.
func (s Store) WithReviewCoordinator(run func(Store) error) error {
	var callbackErr error
	auditErr := s.WithReviewOperationView(func(shared Store) {
		v := shared.reviewView
		if v == nil {
			callbackErr = fmt.Errorf("pose: review coordinator requires comparable Files binding")
			return
		}
		readView, _ := governanceReadViewFromFiles(shared.fsys())
		v.scopeBase = readView.operation.baseStore
		v.scopeRoots = readView.operation.initialRoots
		v.scopeConflicts = readView.operation.initialConflicts
		v.coordinatorStore = shared
		callbackErr = run(shared)
	})
	return errors.Join(callbackErr, auditErr)
}

// reviewScopeCatalogue authorizes reuse against the exact coordinator Store,
// and creates a separately bound catalogue consumer. It never unwraps Files.
type reviewScopeCatalogue interface {
	scopeCatalogue(Store) (Store, *reviewOperationView, error)
}

func (v *reviewOperationView) scopeCatalogue(s Store) (Store, *reviewOperationView, error) {
	if !v.boundTo(s) || v.coordinator != nil || v.coordinatorStore.reviewView != v {
		return Store{}, nil, s.refuseReviewOperationContext(nil, nil)
	}
	v.scopeInit.Do(func() {
		shared := v.coordinatorStore
		v.captureSpecs(shared)
		if v.specErr != nil {
			v.scopeInitErr = v.specErr
			return
		}
		dir, err := ensureReviewArtifactDirIn(v.files, v.root, ".pose/review-bundles", false)
		if err != nil {
			v.observeDirectoryError(shared, err)
			return
		}
		entries, err := v.files.ReadDir(dir)
		if err != nil {
			v.observeDirectoryError(shared, err)
			return
		}
		_, v.scopeInitErr = shared.reviewBundleHeaders(dir, entries)
	})
	if v.scopeInitErr != nil {
		return Store{}, nil, v.scopeInitErr
	}
	return v.scopeBase, v, nil
}

// WithReviewScopeOperationView evaluates one scope, including recursive children
// and authorized roots, with an independent bounded capture and final audit.
func (s Store) WithReviewScopeOperationView(run func(Store) error) error {
	var catalog reviewScopeCatalogue = s.reviewView
	if s.reviewView == nil {
		return fmt.Errorf("pose: review scope requires an active coordinator")
	}
	base, parent, err := catalog.scopeCatalogue(s)
	if err != nil {
		return err
	}
	base, readView := base.beginGovernanceReadView()
	shared := base.withGovernanceReadView(readView, "review-operation")
	if !sameStringMap(parent.scopeRoots, readView.operation.initialRoots) || !sameBoolMap(parent.scopeConflicts, readView.operation.initialConflicts) {
		readView.markUnknown("review-operation", "scope authorized root vector differs from coordinator")
		return errors.Join(errGovernanceReadViewOutsideRoot, reviewOperationGovernanceAuditError(readView.audit(), readView, nil))
	}
	view := &reviewOperationView{root: filepath.Clean(shared.Root), files: shared.fsys(), project: shared.FederatedProjectID, resolver: shared.FederatedResolver,
		coordinator: parent, specUsed: parent.specUsed, specErr: parent.specErr, specFallback: parent.specFallback, specEntries: parent.specEntries, specBySlug: parent.specBySlug,
		specBodySeen: make([]bool, len(parent.specEntries)), specBodyDigests: make([][32]byte, len(parent.specEntries)),
		used: parent.used, directoryError: parent.directoryError, inventory: parent.inventory, headers: parent.headers, fallback: parent.fallback,
		maxRecords: parent.maxRecords, maxInputBytes: parent.maxInputBytes, maxRetainedText: parent.maxRetainedText}
	view.specOnce.Do(func() {})
	view.once.Do(func() {})
	var callbackErr error
	operationErr := shared.withReviewOperationView(view, func(scoped Store) { callbackErr = run(scoped) })
	audit := readView.audit()
	return errors.Join(callbackErr, operationErr, reviewOperationGovernanceAuditError(audit, readView, view))
}
