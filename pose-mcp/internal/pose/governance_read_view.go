package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	GovernanceReadViewSchemaVersion = 1
	governanceReadViewParserVersion = "pose-governance-inputs-v1"
	governanceReadViewMaxRecords    = 25000
	governanceReadViewMaxInputBytes = 512 * 1024 * 1024
	governanceReadViewMaxMetadata   = 8 * 1024 * 1024
	ReadViewFreshnessCurrent        = "current"
	ReadViewFreshnessUnknown        = "unknown"
)

var errGovernanceReadViewWrite = errors.New("pose: governance read view is read-only")
var errGovernanceReadViewFinalized = errors.New("pose: governance read view is finalized")
var errGovernanceReadViewOutsideRoot = errors.New("pose: governance read view input escapes the project root")
var errGovernanceReadViewBudget = errors.New("pose: governance read view budget exceeded")
var errGovernanceReadViewIsDirectory = errors.New("is a directory")

// ReadViewMetadata identifies the inputs verified for one operation. Root
// paths are represented by digests to avoid disclosing host paths.
type ReadViewMetadata struct {
	SchemaVersion  int               `json:"schema_version"`
	ID             string            `json:"id"`
	Scope          string            `json:"scope,omitempty"`
	Freshness      string            `json:"freshness"`
	EngineVersion  string            `json:"engine_version"`
	ParserVersions map[string]string `json:"parser_versions"`
	Bindings       []ReadViewBinding `json:"bindings"`
	ObservedInputs int               `json:"observed_inputs"`
	InputBytes     int64             `json:"input_bytes"`
	Limitations    []string          `json:"limitations,omitempty"`
}

// ReadViewBinding is one authorized root in an operation's generation vector.
type ReadViewBinding struct {
	ProjectID       string `json:"project_id"`
	RootDigest      string `json:"root_digest"`
	SourceRevision  string `json:"source_revision"`
	IndexGeneration string `json:"index_generation,omitempty"`
}

type governanceReadView struct {
	root          string
	base          Files
	mu            sync.Mutex
	cond          *sync.Cond
	closed        bool
	active        int
	files         map[string]*governanceReadFileObservation
	observed      map[string]*governanceReadObservation
	unknown       map[string]map[string]bool
	inputBytes    int64
	metadataBytes int64
	operation     *governanceReadViewOperation
}

type governanceReadViewRoot struct {
	project         string
	root            string
	revision        string
	indexGeneration string
	indexPin        *IndexGenerationView
	store           Store
	view            *governanceReadView
	ready           chan struct{}
	producers       map[string]bool
}

type governanceReadViewOperation struct {
	primary                           *governanceReadView
	roots                             []*governanceReadViewRoot
	initialRoots                      map[string]string
	initialConflicts                  map[string]bool
	rootProjectByPath                 map[string]string
	rootRegistryID                    string
	allowMissingPrimarySourceRevision bool
	currentRoots                      func() (map[string]string, map[string]bool, bool)
	resolver                          ArtifactResolver
	baseStore                         Store
	mainProject                       string
	indexPins                         map[string]*IndexGenerationView
	indexInputChecks                  map[string]*governanceReadViewIndexInputCheck
	unknown                           map[string]map[string]bool
	rootOverflow                      bool
	registryOverflow                  bool
	finalAudits                       map[*governanceReadView]governanceReadViewAudit
	budgetMu                          sync.Mutex
	inputBytes                        int64
	metadataBytes                     int64
	records                           int
	directoryEntries                  int
	git                               *governanceReadViewGitState
	mu                                sync.Mutex
}

type governanceReadViewIndexInputCheck struct {
	ready   chan struct{}
	reasons []string
}

type governanceReadFileObservation struct {
	bytes     []byte
	readLimit int64
	err       error
	errText   string
	source    string
	baseName  string
	info      fs.FileInfo
	witness   fs.FileInfo
	rootReal  string
	resolved  string
	producers map[string]bool
}

type governanceReadObservation struct {
	method     string
	path       string
	baseName   string
	err        error
	errText    string
	info       fs.FileInfo
	witness    fs.FileInfo
	entries    []string
	dirEntries []fs.DirEntry
	value      string
	producers  map[string]bool
}

type governanceReadViewAudit struct {
	ChangedPaths     []string
	ChangedProducers map[string]bool
	UnknownProducers map[string][]string
}

type governanceReadViewAuditBudget struct {
	inputBytes int64
	metadata   int64
	records    int
}

func (b *governanceReadViewAuditBudget) reserve(metadata int64) bool {
	if b == nil || metadata < 0 || b.records+1 > governanceReadViewMaxRecords || metadata > governanceReadViewMaxMetadata-b.metadata {
		return false
	}
	b.records++
	b.metadata += metadata
	return true
}

func (b *governanceReadViewAuditBudget) reserveInput(bytes int64) bool {
	if b == nil || bytes < 0 || bytes > governanceReadViewMaxInputBytes-b.inputBytes {
		return false
	}
	b.inputBytes += bytes
	return true
}

func (b *governanceReadViewAuditBudget) reserveDirectory(entries []fs.DirEntry, maxEntries int, maxMetadata int64) bool {
	if b == nil || len(entries) > maxEntries || len(entries) > governanceReadViewMaxRecords-b.records {
		return false
	}
	metadata := readViewDirMetadataBytes(entries)
	if metadata > maxMetadata || metadata > governanceReadViewMaxMetadata-b.metadata {
		return false
	}
	b.records += len(entries)
	b.metadata += metadata
	return true
}

type governanceReadViewProof struct {
	root        string
	inputBudget int64
	entries     []governanceReadProofEntry
	base        Files
	children    []*governanceReadViewProof
}

type governanceReadProofEntry struct {
	method     string
	path       string
	baseName   string
	digest     string
	errText    string
	info       fs.FileInfo
	witness    fs.FileInfo
	rootReal   string
	resolved   string
	inputBytes int64
	readLimit  int64
	entries    []string
	value      string
	producers  []string
}

type canonicalReadViewEntry struct {
	Method         string   `json:"method"`
	Path           string   `json:"path"`
	Digest         string   `json:"digest,omitempty"`
	ErrorDigest    string   `json:"error_digest,omitempty"`
	Name           string   `json:"name,omitempty"`
	Size           int64    `json:"size,omitempty"`
	Mode           uint32   `json:"mode,omitempty"`
	ModTime        int64    `json:"mod_time,omitempty"`
	IdentityDigest string   `json:"identity_digest,omitempty"`
	Entries        []string `json:"entries,omitempty"`
	ValueDigest    string   `json:"value_digest,omitempty"`
	ResolvedDigest string   `json:"resolved_digest,omitempty"`
}

func newGovernanceReadView(root string, base Files) *governanceReadView {
	view := &governanceReadView{root: filepath.Clean(root), base: FilesOrNative(base), files: map[string]*governanceReadFileObservation{}, observed: map[string]*governanceReadObservation{}, unknown: map[string]map[string]bool{}}
	view.cond = sync.NewCond(&view.mu)
	return view
}

func (op *governanceReadViewOperation) route(name, producer, project string) *governanceReadView {
	if op == nil {
		return nil
	}
	path, err := filepath.Abs(name)
	if err != nil {
		return nil
	}
	path = filepath.Clean(path)
	op.mu.Lock()
	rootID, rootPath := op.rootForPathLocked(path, project)
	if rootID == "" {
		op.mu.Unlock()
		return nil
	}
	for _, candidate := range op.roots {
		if candidate.project == rootID && sameLexicalRoot(candidate.root, rootPath) {
			if producer != "" {
				candidate.producers[producer] = true
			}
			view, ready := candidate.view, candidate.ready
			op.mu.Unlock()
			if ready != nil {
				<-ready
			}
			return view
		}
	}
	if len(op.roots) >= 16 {
		op.rootOverflow = true
		primary := op.primary
		op.mu.Unlock()
		primary.markUnknown(producer, "operation exceeded the 16 authorized-root budget")
		return nil
	}
	rootStore, storeErr := op.resolver.Roots.StoreFor(rootID)
	if storeErr != nil {
		rootStore = Store{Root: rootPath, FS: op.resolver.fsys()}
	}
	rootStore = rootStore.withoutGovernanceReadView()
	rootStore.Root = rootPath
	view := newGovernanceReadView(rootPath, rootStore.fsys())
	ready := make(chan struct{})
	entry := &governanceReadViewRoot{project: rootID, root: rootPath, revision: "unknown", indexGeneration: "legacy", store: rootStore, view: view, ready: ready, producers: map[string]bool{producer: true}}
	op.roots = append(op.roots, entry)
	slot := op.indexPins[rootID]
	if slot == nil {
		slot = &IndexGenerationView{}
		op.indexPins[rootID] = slot
	}
	op.mu.Unlock()

	view.markUnknownIf(storeErr != nil, producer, "authorized project root could not be opened")
	revisionBytes, revisionErr := op.capture(rootPath, producer, "rev-parse", "--verify", "HEAD")
	entry.revision = strings.TrimSpace(string(revisionBytes))
	if revisionErr != nil || entry.revision == "" {
		entry.revision = "unknown"
	}
	if entry.revision == "unknown" {
		view.markUnknown(producer, "source revision for an authorized root is unavailable")
	}
	pinned := rootStore
	pinned.Root, pinned.FS = rootPath, view.filesFor(producer)
	pinned = pinned.PinIndexes()
	if pinned.indexPin != nil {
		*slot = *pinned.indexPin
		entry.indexPin = slot
		switch slot.Status {
		case IndexGenerationCurrent:
			entry.indexGeneration = slot.Descriptor.GenerationID
		case IndexGenerationUnavailable:
			entry.indexGeneration = "unknown"
			view.markUnknown(producer, "index generation could not be pinned: "+slot.Reason)
		}
	}
	if err := op.adoptCapturedView(view); err != nil {
		view.markUnknown(producer, err.Error())
	}
	view.operation = op
	close(ready)
	return view
}

func (op *governanceReadViewOperation) rootForPathLocked(path, project string) (string, string) {
	if project != "" {
		configured := op.initialRoots[project]
		if configured != "" && pathWithin(configured, path) {
			return project, configured
		}
	}
	// A Files wrapper may be rooted at the coordinator while a consumer passes
	// an absolute path in another already-authorized root. Route that path by
	// the frozen vector so all roots still share the operation budgets.
	for ancestor := path; ; ancestor = filepath.Dir(ancestor) {
		if project, ok := op.rootProjectByPath[ancestor]; ok {
			return project, ancestor
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
	}
	return "", ""
}

func (v *governanceReadView) markUnknownIf(condition bool, producer, reason string) {
	if condition {
		v.markUnknown(producer, reason)
	}
}

func (op *governanceReadViewOperation) audit() governanceReadViewAudit {
	result := governanceReadViewAudit{ChangedProducers: map[string]bool{}, UnknownProducers: map[string][]string{}}
	if op == nil {
		return result
	}
	op.mu.Lock()
	roots := append([]*governanceReadViewRoot(nil), op.roots...)
	initialRoots := op.initialRoots // Immutable after beginGovernanceReadView returns.
	initialConflicts := op.initialConflicts
	currentRoots := op.currentRoots
	op.mu.Unlock()
	perRoot := map[*governanceReadView]governanceReadViewAudit{}
	auditBudget := &governanceReadViewAuditBudget{}
	for _, root := range roots {
		if _, done := perRoot[root.view]; done {
			continue
		}
		perRoot[root.view] = root.view.auditSingleWithBudget(auditBudget)
	}
	changed := []string{}
	for _, audit := range perRoot {
		changed = append(changed, audit.ChangedPaths...)
		for producer, value := range audit.ChangedProducers {
			result.ChangedProducers[producer] = value
		}
		for producer, reasons := range audit.UnknownProducers {
			result.UnknownProducers[producer] = append(result.UnknownProducers[producer], reasons...)
		}
	}
	if op.git != nil {
		gitAudit := op.git.audit()
		primaryAudit := perRoot[op.primary]
		mergeGovernanceReadViewAudit(&primaryAudit, gitAudit)
		perRoot[op.primary] = primaryAudit
		changed = append(changed, gitAudit.ChangedPaths...)
		for producer, value := range gitAudit.ChangedProducers {
			result.ChangedProducers[producer] = value
		}
		for producer, reasons := range gitAudit.UnknownProducers {
			result.UnknownProducers[producer] = append(result.UnknownProducers[producer], reasons...)
		}
	}
	if currentRoots != nil {
		current, conflicts, overBudget := currentRoots()
		if overBudget || !sameStringMap(initialRoots, current) || !sameBoolMap(initialConflicts, conflicts) {
			changed = append(changed, "authorized-roots")
			result.ChangedProducers["projection"] = true
			reason := "authorized root registry changed during the operation"
			if overBudget {
				reason = "authorized root registry exceeded its bounded final-audit budget"
			}
			result.UnknownProducers["projection"] = append(result.UnknownProducers["projection"], reason)
		}
	}
	result.ChangedPaths = uniqueSorted(changed)
	for producer := range result.UnknownProducers {
		result.UnknownProducers[producer] = uniqueSorted(result.UnknownProducers[producer])
	}
	op.mu.Lock()
	op.finalAudits = perRoot
	op.mu.Unlock()
	return result
}

func mergeGovernanceReadViewAudit(target *governanceReadViewAudit, source governanceReadViewAudit) {
	if target.ChangedProducers == nil {
		target.ChangedProducers = map[string]bool{}
	}
	if target.UnknownProducers == nil {
		target.UnknownProducers = map[string][]string{}
	}
	target.ChangedPaths = append(target.ChangedPaths, source.ChangedPaths...)
	for producer, changed := range source.ChangedProducers {
		if changed {
			target.ChangedProducers[producer] = true
		}
	}
	for producer, reasons := range source.UnknownProducers {
		target.UnknownProducers[producer] = append(target.UnknownProducers[producer], reasons...)
	}
}

func (op *governanceReadViewOperation) metadata(project, revision, indexGeneration string) (ReadViewMetadata, *governanceReadViewProof) {
	if op == nil {
		return ReadViewMetadata{}, nil
	}
	op.mu.Lock()
	finalAudits := op.finalAudits
	op.mu.Unlock()
	if finalAudits == nil {
		_ = op.audit()
	}
	op.mu.Lock()
	finalAudits = op.finalAudits
	roots := append([]*governanceReadViewRoot(nil), op.roots...)
	rootOverflow := op.rootOverflow
	op.mu.Unlock()
	metadata := ReadViewMetadata{SchemaVersion: GovernanceReadViewSchemaVersion, Freshness: ReadViewFreshnessCurrent, ParserVersions: map[string]string{}, Bindings: []ReadViewBinding{}, Limitations: []string{}}
	proof := &governanceReadViewProof{inputBudget: governanceReadViewMaxInputBytes}
	type childIdentity struct {
		Project string `json:"project"`
		Root    string `json:"root"`
		ID      string `json:"id"`
	}
	identities := []childIdentity{}
	seenBindings := map[string]bool{}
	seenViews := map[*governanceReadView]bool{}
	for _, root := range roots {
		if root.view == nil {
			continue
		}
		childProject, childRevision, childIndex := root.project, root.revision, root.indexGeneration
		if root.view == op.primary {
			childProject, childRevision, childIndex = project, revision, indexGeneration
			if childRevision == "" {
				childRevision = root.revision
			}
			if childIndex == "" {
				childIndex = root.indexGeneration
			}
		}
		audit := finalAudits[root.view]
		child, childProof := root.view.metadataSingle(childProject, childRevision, childIndex, audit)
		metadata.ObservedInputs += child.ObservedInputs
		metadata.InputBytes += child.InputBytes
		metadata.Limitations = append(metadata.Limitations, child.Limitations...)
		metadata.ParserVersions = child.ParserVersions
		metadata.EngineVersion = child.EngineVersion
		if child.Freshness != ReadViewFreshnessCurrent {
			metadata.Freshness = ReadViewFreshnessUnknown
		}
		for _, binding := range child.Bindings {
			key := binding.ProjectID + "\x00" + binding.RootDigest
			if !seenBindings[key] {
				seenBindings[key] = true
				metadata.Bindings = append(metadata.Bindings, binding)
			}
		}
		identities = append(identities, childIdentity{Project: childProject, Root: root.root, ID: child.ID})
		if !seenViews[root.view] {
			seenViews[root.view] = true
			proof.children = append(proof.children, childProof)
		}
	}
	op.budgetMu.Lock()
	metadata.ObservedInputs = op.records + op.directoryEntries
	metadata.InputBytes = op.inputBytes
	op.budgetMu.Unlock()
	metadata.ParserVersions["raw-git"] = governanceReadViewGitParserVersion
	sort.Slice(metadata.Bindings, func(i, j int) bool {
		if metadata.Bindings[i].ProjectID != metadata.Bindings[j].ProjectID {
			return metadata.Bindings[i].ProjectID < metadata.Bindings[j].ProjectID
		}
		return metadata.Bindings[i].RootDigest < metadata.Bindings[j].RootDigest
	})
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].Project != identities[j].Project {
			return identities[i].Project < identities[j].Project
		}
		return identities[i].Root < identities[j].Root
	})
	if rootOverflow {
		metadata.Freshness = ReadViewFreshnessUnknown
		metadata.Limitations = append(metadata.Limitations, "authorized root vector exceeds the 16-root budget")
	}
	if op.registryOverflow {
		metadata.Freshness = ReadViewFreshnessUnknown
		metadata.Limitations = append(metadata.Limitations, "authorized root registry exceeds its bounded snapshot budget")
	}
	metadata.Limitations = uniqueSorted(metadata.Limitations)
	identity := struct {
		SchemaVersion  int               `json:"schema_version"`
		EngineVersion  string            `json:"engine_version"`
		ParserVersions map[string]string `json:"parser_versions"`
		Bindings       []ReadViewBinding `json:"bindings"`
		Children       []childIdentity   `json:"children"`
		GitDigest      string            `json:"git_digest,omitempty"`
		RootRegistryID string            `json:"root_registry_id"`
	}{metadata.SchemaVersion, metadata.EngineVersion, metadata.ParserVersions, metadata.Bindings, identities, op.git.identityDigest(), op.rootRegistryID}
	raw, _ := json.Marshal(identity)
	sum := sha256.Sum256(raw)
	metadata.ID = "sha256:" + hex.EncodeToString(sum[:])
	return metadata, proof
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func sameBoolMap(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func (op *governanceReadViewOperation) remainingInput() int64 {
	if op == nil {
		return governanceReadViewMaxInputBytes
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	return governanceReadViewMaxInputBytes - op.inputBytes
}

func (op *governanceReadViewOperation) reserveInput(size int64) bool {
	if op == nil {
		return size >= 0 && size <= governanceReadViewMaxInputBytes
	}
	if size < 0 {
		return false
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	if size > governanceReadViewMaxInputBytes-op.inputBytes {
		return false
	}
	op.inputBytes += size
	return true
}

func (op *governanceReadViewOperation) finishInput(reserved, actual int64) {
	if op == nil || actual >= reserved {
		return
	}
	op.budgetMu.Lock()
	op.inputBytes -= reserved - actual
	if op.inputBytes < 0 {
		op.inputBytes = 0
	}
	op.budgetMu.Unlock()
}

func (op *governanceReadViewOperation) reserveRecord(metadataBytes int64) bool {
	if op == nil {
		return true
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	if op.records+op.directoryEntries+1 > governanceReadViewMaxRecords || metadataBytes < 0 || metadataBytes > governanceReadViewMaxMetadata-op.metadataBytes {
		return false
	}
	op.records++
	op.metadataBytes += metadataBytes
	return true
}

// checkIndexInputs verifies the declared source bindings only when a producer
// actually consumes a derived index. Taking a pin for snapshot identity alone
// does not make the projection depend on that index, so the projection producer
// deliberately skips this check. The work is shared per root, generation and
// producer; each producer still records its own observations and limitation.
func (op *governanceReadViewOperation) checkIndexInputs(files Files, root, producer string, pin *IndexGenerationView) []string {
	if op == nil || pin == nil || pin.Status != IndexGenerationCurrent {
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return []string{"index input root could not be resolved"}
	}
	key := filepath.Clean(abs) + "\x00" + pin.Descriptor.GenerationID + "\x00" + producer
	op.mu.Lock()
	if op.indexInputChecks == nil {
		op.indexInputChecks = map[string]*governanceReadViewIndexInputCheck{}
	}
	if prior := op.indexInputChecks[key]; prior != nil {
		ready := prior.ready
		op.mu.Unlock()
		<-ready
		return append([]string(nil), prior.reasons...)
	}
	check := &governanceReadViewIndexInputCheck{ready: make(chan struct{})}
	op.indexInputChecks[key] = check
	op.mu.Unlock()

	reasons := []string{}
	for _, name := range IndexInputNames {
		bound, exists := pin.Descriptor.Inputs[name]
		if !exists || strings.TrimSpace(bound) == "" {
			reasons = append(reasons, "index generation does not bind input "+name)
			continue
		}
		current, known := op.indexInputBinding(files, root, producer, name)
		if !known || current == "" {
			reasons = append(reasons, "current index input "+name+" could not be bounded")
			continue
		}
		if current != bound {
			reasons = append(reasons, "index input "+name+" changed since generation "+pin.Descriptor.GenerationID)
		}
	}
	reasons = uniqueSorted(reasons)
	op.mu.Lock()
	check.reasons = append([]string(nil), reasons...)
	close(check.ready)
	op.mu.Unlock()
	return reasons
}

// Revalidation must use the same captured project/resolver context as the
// generation consumer, including when that consumer is a foreign Store.
func (op *governanceReadViewOperation) indexInputBinding(files Files, root, producer, name string) (string, bool) {
	if name != "federated-material" {
		return IndexInputBindingIn(files, root, name)
	}
	op.mu.Lock()
	project, selectedRoot := op.rootForPathLocked(governanceAbsoluteRoot(root), "")
	conflict := op.initialConflicts[project]
	op.mu.Unlock()
	if project == "" || conflict || !sameLexicalRoot(root, selectedRoot) {
		return "", false
	}
	resolver := op.resolver
	resolver.FS = files
	resolver.Roots = op.rootsForProducer(resolver.Roots, files, producer)
	resolver.snapshots = nil
	resolver.governanceDepthLimit = 8
	return IndexInputBindingForStoreIn(Store{Root: root, FS: files, FederatedProjectID: project, FederatedResolver: &resolver}, name)
}

func (s Store) governanceIndexConsumed(status IndexGenerationStatus, err error, pin *IndexGenerationView) {
	if err != nil {
		return
	}
	view, carrier := governanceReadViewFromFiles(s.fsys())
	if view == nil || carrier == nil || carrier.producer == "" || carrier.producer == "projection" {
		return
	}
	producer := carrier.producer
	project := carrier.project
	switch status {
	case IndexGenerationLegacy:
		markGovernanceReadUnknownForProject(s.fsys(), s.Root, project, producer, "a legacy index has no bound input generation")
		return
	case IndexGenerationCurrent:
	default:
		return
	}
	if pin == nil {
		pin = s.indexPin
	}
	if pin == nil {
		captured := s.ReadIndexGeneration()
		pin = &captured
	}
	if pin.Status != IndexGenerationCurrent {
		markGovernanceReadUnknownForProject(s.fsys(), s.Root, project, producer, "the consumed index generation could not be fixed")
		return
	}
	if view.operation == nil {
		markGovernanceReadUnknownForProject(s.fsys(), s.Root, project, producer, "the consumed index has no operation-scoped input bindings")
		return
	}
	for _, reason := range view.operation.checkIndexInputs(s.fsys(), s.Root, producer, pin) {
		markGovernanceReadUnknownForProject(s.fsys(), s.Root, project, producer, reason)
	}
}

// adoptCapturedView charges the bounded index-generation read made while a
// lazy root binding was being initialized. The view is not published to other
// producers until after this transfer, so the operation's aggregate budget
// still covers those first reads.
func (op *governanceReadViewOperation) adoptCapturedView(view *governanceReadView) error {
	if op == nil || view == nil {
		return nil
	}
	view.mu.Lock()
	defer view.mu.Unlock()
	var records, directories int
	var metadata int64 = view.metadataBytes
	for _, file := range view.files {
		records++
		metadata += int64(len("file") + len(view.displayPath(file.baseName)) + 64)
	}
	for _, observation := range view.observed {
		records++
		if observation.method == "directory" {
			directories += len(observation.entries)
		}
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	if view.inputBytes > governanceReadViewMaxInputBytes-op.inputBytes {
		return errGovernanceReadViewBudget
	}
	if records < 0 || op.records+op.directoryEntries+records+directories > governanceReadViewMaxRecords {
		return errGovernanceReadViewBudget
	}
	if metadata < 0 || metadata > governanceReadViewMaxMetadata-op.metadataBytes {
		return errGovernanceReadViewBudget
	}
	op.inputBytes += view.inputBytes
	op.records += records
	op.directoryEntries += directories
	op.metadataBytes += metadata
	return nil
}

func (op *governanceReadViewOperation) readDirectory(files Files, name string) ([]fs.DirEntry, error) {
	if op == nil {
		return readDirGovernanceBounded(files, name, governanceReadViewMaxRecords)
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	remainingEntries := governanceReadViewMaxRecords - op.records - op.directoryEntries
	remainingMetadata := governanceReadViewMaxMetadata - op.metadataBytes
	if remainingEntries < 0 || remainingMetadata < 0 {
		return nil, errGovernanceReadViewBudget
	}
	entries, err := readDirGovernanceBoundedWithBudget(files, name, remainingEntries, remainingMetadata)
	if err != nil {
		return nil, err
	}
	retained := readViewDirMetadataBytes(entries)
	if len(entries) > remainingEntries || retained > remainingMetadata {
		return nil, errGovernanceReadViewBudget
	}
	op.directoryEntries += len(entries)
	op.metadataBytes += retained
	return entries, nil
}

func readViewDirMetadataBytes(entries []fs.DirEntry) int64 {
	var total int64
	for _, entry := range entries {
		// Charge both the retained name and the canonical identity text, plus
		// conservative slice/string metadata overhead.
		total += int64(len(entry.Name())*2 + len(fmt.Sprintf("%s\x00%o", entry.Name(), entry.Type())) + 128)
	}
	return total
}

func (s Store) beginGovernanceReadView(allowMissingPrimaryRevision ...bool) (Store, *governanceReadView) {
	base := s.withoutGovernanceReadView()
	resolver := ArtifactResolver{}
	project := strings.TrimSpace(base.FederatedProjectID)
	consumerHasResolver := base.FederatedResolver != nil
	consumerHasRoots := consumerHasResolver && base.FederatedResolver.Roots != nil
	fromEnvironment := !consumerHasResolver
	var resolverErr error
	projectBindingAmbiguous := false
	projectBindingOverflow := false
	if base.FederatedResolver != nil {
		resolver = *base.FederatedResolver
		if resolver.Roots == nil {
			resolverErr = errors.New("authorized project roots are unavailable")
		} else if project == "" {
			var found bool
			project, found, projectBindingAmbiguous, projectBindingOverflow = uniqueProjectIDForRootIn(base.fsys(), resolver.Roots, base.Root, resolver.Authorize)
			if projectBindingAmbiguous {
				project = ""
				resolverErr = errors.New("project root has multiple authorized project identities")
			} else if projectBindingOverflow {
				project = ""
				resolverErr = errors.New("authorized project root registry exceeds its bounded snapshot budget")
			} else if !found {
				candidate, identityErr := ResolveDefaultProjectIDIn(base.fsys(), base.Root)
				if identityErr != nil {
					project = ""
					resolverErr = fmt.Errorf("primary project identity could not be resolved: %w", identityErr)
				} else if ValidateSlug(candidate) != nil || resolver.Authorize != nil && !resolver.Authorize(candidate) {
					project = ""
					resolverErr = errors.New("primary project identity is not authorized")
				} else {
					selected, selectionErr := resolver.Roots.StoreFor(candidate)
					if selectionErr == nil && sameProjectRoot(base.fsys(), base.Root, selected.Root) {
						project = candidate
					} else if isUnknownProjectSelection(selectionErr) && rootsHaveNoDefaultProjectID(resolver.Roots) {
						implicitDefault, defaultErr := resolver.Roots.StoreFor("")
						if defaultErr == nil && sameProjectRoot(base.fsys(), base.Root, implicitDefault.Root) {
							project = candidate
						} else {
							project = ""
							resolverErr = errors.New("primary root is not bound to its derived project identity")
						}
					} else {
						project = ""
						resolverErr = errors.New("primary root is not bound to its derived project identity")
					}
				}
			}
		}
	} else {
		resolver, project, resolverErr = EnvironmentArtifactResolverIn(base.fsys(), base.Root, "")
	}
	if project == "" && fromEnvironment && !projectBindingAmbiguous && !projectBindingOverflow {
		candidate, identityErr := ResolveDefaultProjectIDIn(base.fsys(), base.Root)
		if identityErr == nil {
			project = candidate
		} else if resolverErr == nil {
			resolverErr = identityErr
		}
	}
	if resolver.Roots == nil {
		resolver.Roots = NewRootsIn(base.fsys(), RootsConfig{DefaultRoot: base.Root, DefaultProjectID: project})
	}
	resolver.governanceDepthLimit = 8
	resolver.FS = base.fsys()
	base.FederatedProjectID = project
	if consumerHasResolver {
		consumerResolver := resolver
		if !consumerHasRoots {
			// An internal primary-root binding must not grant a caller a
			// federated root registry it did not supply.
			consumerResolver.Roots = nil
		}
		base.FederatedResolver = &consumerResolver
	} else {
		// The operation uses the environment resolver to bind its identity and
		// authorized root vector. Do not install that resolver on the consumer
		// Store: a plain local Store has not opted into federated semantics.
		base.FederatedResolver = nil
	}
	primary := newGovernanceReadView(base.Root, base.fsys())
	op := &governanceReadViewOperation{
		primary: primary, resolver: resolver, baseStore: base, mainProject: project,
		allowMissingPrimarySourceRevision: len(allowMissingPrimaryRevision) > 0 && allowMissingPrimaryRevision[0],
		indexPins:                         map[string]*IndexGenerationView{}, indexInputChecks: map[string]*governanceReadViewIndexInputCheck{}, unknown: map[string]map[string]bool{},
	}
	op.git = newGovernanceReadViewGitState(op)
	primary.operation = op
	if resolverErr != nil {
		primary.markUnknown("projection", "authorized roots could not be resolved: "+resolverErr.Error())
	}
	rootMap, rootConflicts, registryOverflow := authorizedRootRegistrySnapshot(resolver.Roots, base.Root, project, resolver.Authorize, op.reserveRecord)
	op.initialRoots = rootMap
	op.initialConflicts = rootConflicts
	op.rootProjectByPath = governanceRootPathIndex(rootMap, project)
	op.registryOverflow = registryOverflow
	op.rootRegistryID = governanceRootRegistryDigest(rootMap, rootConflicts, registryOverflow)
	if project != "" && rootConflicts[project] {
		primary.markUnknown("projection", "primary root conflicts with its authorized project identity")
	}
	if registryOverflow {
		primary.markUnknown("projection", "authorized root registry exceeded its bounded snapshot budget")
	}
	rootMapForAudit := func() (map[string]string, map[string]bool, bool) {
		currentResolver := resolver
		currentProject := project
		if fromEnvironment {
			var err error
			currentResolver, currentProject, err = EnvironmentArtifactResolverIn(base.fsys(), base.Root, "")
			if err != nil {
				return map[string]string{}, map[string]bool{}, true
			}
			currentResolver.governanceDepthLimit = 8
		}
		return authorizedRootRegistrySnapshot(currentResolver.Roots, base.Root, currentProject, currentResolver.Authorize, nil)
	}
	op.currentRoots = rootMapForAudit
	mainRoot, mainErr := filepath.Abs(base.Root)
	if mainErr == nil {
		mainRoot = filepath.Clean(mainRoot)
	} else {
		mainRoot = filepath.Clean(base.Root)
	}
	primaryPin := &IndexGenerationView{}
	if project != "" {
		op.indexPins[project] = primaryPin
	}
	op.roots = []*governanceReadViewRoot{{project: project, root: mainRoot, revision: "", indexGeneration: "legacy", indexPin: primaryPin, store: base, view: primary, producers: map[string]bool{"projection": true}}}
	primaryRevision, revisionErr := op.capture(mainRoot, "projection", "rev-parse", "--verify", "HEAD")
	op.roots[0].revision = strings.TrimSpace(string(primaryRevision))
	if revisionErr != nil || op.roots[0].revision == "" {
		if op.allowMissingPrimarySourceRevision {
			op.roots[0].revision = ""
		} else {
			op.roots[0].revision = "unknown"
			primary.markUnknown("projection", "source revision for the primary root is unavailable")
		}
	}
	return base, primary
}

func rootsHaveNoDefaultProjectID(roots *Roots) bool {
	if roots == nil {
		return false
	}
	roots.mu.RLock()
	defer roots.mu.RUnlock()
	return roots.cfg.DefaultProjectID == ""
}

func isUnknownProjectSelection(err error) bool {
	var unknown ProjectUnknownError
	return errors.As(err, &unknown)
}

// uniqueProjectIDForRootIn returns the sole authorized project identifier
// already bound to root. It reuses the bounded registry snapshot so aliases
// cannot force unbounded allocation or filesystem lookups. A second alias or
// an over-budget registry fails closed; callers must not substitute a
// directory-name identity for either case.
func uniqueProjectIDForRootIn(files Files, roots *Roots, root string, authorize func(string) bool) (projectID string, found, ambiguous, overflow bool) {
	candidates, conflicts, overflow := authorizedRootRegistrySnapshot(roots, "", "", authorize, nil)
	if overflow {
		return "", false, false, true
	}
	files = FilesOrNative(files)
	matched := []string{}
	for id, candidate := range candidates {
		if id == "" || candidate == "" || !sameProjectRoot(files, root, candidate) {
			continue
		}
		if conflicts[id] {
			return "", false, true, false
		}
		matched = append(matched, id)
	}
	sort.Strings(matched)
	switch len(matched) {
	case 0:
		return "", false, false, false
	case 1:
		return matched[0], true, false, false
	default:
		return "", false, true, false
	}
}

func (op *governanceReadViewOperation) mayOmitPrimaryRevision(root string) bool {
	if op == nil || !op.allowMissingPrimarySourceRevision {
		return false
	}
	op.mu.Lock()
	defer op.mu.Unlock()
	for _, entry := range op.roots {
		if entry.view == op.primary {
			return sameLexicalRoot(entry.root, root)
		}
	}
	return false
}

func (op *governanceReadViewOperation) requirePrimaryRevision(producer string) {
	if op == nil {
		return
	}
	op.mu.Lock()
	revision := "unknown"
	for _, entry := range op.roots {
		if entry.view == op.primary {
			revision = entry.revision
			break
		}
	}
	op.mu.Unlock()
	if revision == "" || revision == "unknown" {
		op.primary.markUnknown(producer, "source revision is required for this index build but is unavailable")
	}
}

func (op *governanceReadViewOperation) recordIndexPin(root string, pin *IndexGenerationView) {
	if op == nil || pin == nil {
		return
	}
	op.mu.Lock()
	defer op.mu.Unlock()
	for _, entry := range op.roots {
		if !sameLexicalRoot(entry.root, root) {
			continue
		}
		slot := op.indexPins[entry.project]
		if slot == nil {
			slot = &IndexGenerationView{}
			op.indexPins[entry.project] = slot
		}
		*slot = *pin
		entry.indexPin = slot
		switch pin.Status {
		case IndexGenerationCurrent:
			entry.indexGeneration = pin.Descriptor.GenerationID
		case IndexGenerationUnavailable:
			entry.indexGeneration = "unknown"
			entry.view.markUnknown("projection", "index generation could not be pinned: "+pin.Reason)
		}
	}
}

func authorizedRootRegistrySnapshot(roots *Roots, mainRoot, mainProject string, authorize func(string) bool, reserve func(int64) bool) (map[string]string, map[string]bool, bool) {
	result := map[string]string{}
	conflicts := map[string]bool{}
	retainedMetadata := int64(0)
	mainPath := governanceAbsoluteRoot(mainRoot)
	add := func(project, root string, conflict bool) bool {
		if project == "" || root == "" {
			return true
		}
		absolute := governanceAbsoluteRoot(root)
		if project == mainProject {
			if _, exists := result[project]; !exists {
				metadataCost := int64(len(project) + len(mainPath) + 128)
				if len(result) >= governanceReadViewMaxRecords || metadataCost > governanceReadViewMaxMetadata-retainedMetadata || reserve != nil && !reserve(metadataCost) {
					return false
				}
				retainedMetadata += metadataCost
			}
			if absolute != mainPath || conflict {
				conflicts[project] = true
			}
			result[project] = mainPath
			return true
		}
		if existing, ok := result[project]; ok {
			if existing != absolute || conflict {
				conflicts[project] = true
			}
			return true
		}
		metadataCost := int64(len(project) + len(absolute) + 128)
		if len(result) >= governanceReadViewMaxRecords || metadataCost > governanceReadViewMaxMetadata-retainedMetadata || reserve != nil && !reserve(metadataCost) {
			return false
		}
		result[project] = absolute
		retainedMetadata += metadataCost
		if conflict {
			conflicts[project] = true
		}
		return true
	}
	if mainProject != "" && !add(mainProject, mainPath, false) {
		return result, conflicts, true
	}
	if roots != nil {
		roots.mu.RLock()
		if len(roots.byProject) > governanceReadViewMaxRecords {
			roots.mu.RUnlock()
			return result, conflicts, true
		}
		for project, root := range roots.byProject {
			if authorize != nil && !authorize(project) {
				continue
			}
			if !add(project, root, roots.conflicts[project]) {
				roots.mu.RUnlock()
				return result, conflicts, true
			}
		}
		if roots.cfg.DefaultProjectID != "" && roots.cfg.DefaultRoot != "" && (authorize == nil || authorize(roots.cfg.DefaultProjectID)) {
			if !add(roots.cfg.DefaultProjectID, roots.cfg.DefaultRoot, roots.conflicts[roots.cfg.DefaultProjectID]) {
				roots.mu.RUnlock()
				return result, conflicts, true
			}
		}
		roots.mu.RUnlock()
	}
	return result, conflicts, false
}

func governanceAbsoluteRoot(root string) string {
	absolute, err := filepath.Abs(root)
	if err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(root)
}

func governanceRootPathIndex(roots map[string]string, preferredProject string) map[string]string {
	index := make(map[string]string, len(roots))
	for project, root := range roots {
		current, exists := index[root]
		if !exists || project == preferredProject || current != preferredProject && project < current {
			index[root] = project
		}
	}
	return index
}

func governanceRootRegistryDigest(roots map[string]string, conflicts map[string]bool, overflow bool) string {
	raw, _ := json.Marshal(struct {
		Roots     map[string]string `json:"roots"`
		Conflicts map[string]bool   `json:"conflicts,omitempty"`
		Overflow  bool              `json:"overflow,omitempty"`
	}{roots, conflicts, overflow})
	return digestBytes(raw)
}

func sameLexicalRoot(a, b string) bool {
	left, lerr := filepath.Abs(a)
	right, rerr := filepath.Abs(b)
	return lerr == nil && rerr == nil && filepath.Clean(left) == filepath.Clean(right)
}

func containsReadViewRoot(roots []*governanceReadViewRoot, view *governanceReadView) bool {
	for _, root := range roots {
		if root.view == view {
			return true
		}
	}
	return false
}

func (v *governanceReadView) filesFor(producer string) Files {
	return &governanceReadViewFiles{view: v, producer: producer}
}

func (v *governanceReadView) filesForProject(producer, project string) Files {
	return &governanceReadViewFiles{view: v, producer: producer, project: project}
}

func (v *governanceReadView) route(name, producer, project string) *governanceReadView {
	if v.operation == nil {
		return v
	}
	if target := v.operation.route(name, producer, project); target != nil {
		return target
	}
	v.markUnknown(producer, "input is outside the authorized root generation vector")
	return v
}

func (v *governanceReadView) pathKey(name string) string {
	path, err := filepath.Abs(name)
	if err == nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(name)
}

func (v *governanceReadView) begin() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return errGovernanceReadViewFinalized
	}
	v.active++
	return nil
}

func (v *governanceReadView) end() {
	v.mu.Lock()
	v.active--
	if v.active == 0 && v.cond != nil {
		v.cond.Broadcast()
	}
	v.mu.Unlock()
}

func (v *governanceReadView) displayPath(path string) string {
	rel, err := filepath.Rel(v.root, path)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel)
	}
	return "external:" + digestBytes([]byte(filepath.Clean(path)))
}

func (v *governanceReadView) markUnknownLocked(producer, reason string) {
	if producer == "" {
		producer = "projection"
	}
	if v.unknown[producer] == nil {
		v.unknown[producer] = map[string]bool{}
	}
	v.unknown[producer][reason] = true
}

func (v *governanceReadView) markUnknown(producer, reason string) {
	v.mu.Lock()
	v.markUnknownLocked(producer, reason)
	v.mu.Unlock()
}

func markGovernanceReadUnknown(files Files, root, producer, reason string) {
	viewFiles, ok := FilesOrNative(files).(*governanceReadViewFiles)
	if !ok || viewFiles.view == nil {
		return
	}
	view := viewFiles.view.route(root, producer, viewFiles.project)
	view.markUnknown(producer, reason)
}

func governanceProducerFromFiles(files Files, fallback string) string {
	_, viewFiles := governanceReadViewFromFiles(files)
	if viewFiles != nil && viewFiles.producer != "" {
		return viewFiles.producer
	}
	return fallback
}

func markGovernanceReadUnknownForProject(files Files, root, project, fallback, reason string) {
	view, viewFiles := governanceReadViewFromFiles(files)
	if view == nil {
		return
	}
	producer := governanceProducerFromFiles(files, fallback)
	if producer == "" {
		producer = "projection"
	}
	routeProject := project
	if viewFiles != nil && viewFiles.project != "" {
		routeProject = viewFiles.project
	}
	target := view.route(root, producer, routeProject)
	if target == nil {
		target = view
	}
	target.markUnknown(producer, reason)
}

func (v *governanceReadView) resolveForReadLocked(name, producer string) (string, string, error) {
	rootAbs, err := filepath.Abs(v.root)
	if err != nil {
		v.markUnknownLocked(producer, "authorized root identity could not be resolved")
		return "", "", err
	}
	path := v.pathKey(name)
	if !pathWithin(rootAbs, path) {
		v.markUnknownLocked(producer, "input is outside the bound project root")
		return "", "", errGovernanceReadViewOutsideRoot
	}
	rootReal, rootErr := v.readStringLocked("root-eval-symlinks", rootAbs, rootAbs, producer, v.base.EvalSymlinks)
	if rootErr != nil {
		v.markUnknownLocked(producer, "authorized root symlink could not be resolved")
		return "", "", rootErr
	}
	if filepath.Clean(path) == filepath.Clean(rootAbs) {
		return filepath.Clean(rootReal), filepath.Clean(rootReal), nil
	}
	parent := filepath.Dir(path)
	resolvedParent, parentErr := v.readStringLocked("eval-parent-symlinks", parent, parent, producer, v.base.EvalSymlinks)
	if parentErr != nil {
		if errors.Is(parentErr, os.ErrNotExist) {
			// A missing parent is a concrete absent input. The failed parent lookup
			// stays in the closure and will conflict if the directory appears.
			return filepath.Join(parent, filepath.Base(path)), filepath.Clean(rootReal), nil
		}
		v.markUnknownLocked(producer, "input parent symlink could not be resolved")
		return "", rootReal, parentErr
	}
	resolvedParent = filepath.Clean(resolvedParent)
	if !pathWithin(rootReal, resolvedParent) {
		v.markUnknownLocked(producer, "input parent resolves outside the bound project root")
		return "", rootReal, errGovernanceReadViewOutsideRoot
	}
	linkInfo, linkErr := v.readStatLocked("lstat", path, path, producer, v.base.Lstat)
	if linkErr != nil {
		if errors.Is(linkErr, os.ErrNotExist) {
			// The lexical leaf is absent beneath a proven parent. Preserve that
			// absence and allow ReadFile/Open to return the original ENOENT.
			return filepath.Join(resolvedParent, filepath.Base(path)), filepath.Clean(rootReal), nil
		}
		v.markUnknownLocked(producer, "input metadata could not be read")
		return "", rootReal, linkErr
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return filepath.Join(resolvedParent, filepath.Base(path)), filepath.Clean(rootReal), nil
	}
	_, readlinkErr := v.readStringLocked("readlink", path, path, producer, v.base.Readlink)
	if readlinkErr != nil && !errors.Is(readlinkErr, os.ErrNotExist) {
		v.markUnknownLocked(producer, "input symlink target could not be read")
		return "", rootReal, readlinkErr
	}
	resolved, resolveErr := v.readStringLocked("eval-symlinks", path, path, producer, v.base.EvalSymlinks)
	if resolveErr != nil {
		if errors.Is(resolveErr, os.ErrNotExist) {
			// Dangling links have a meaningful lstat and link target; a future
			// target creation will be found by the captured absence at audit.
			return filepath.Join(resolvedParent, filepath.Base(path)), filepath.Clean(rootReal), nil
		}
		v.markUnknownLocked(producer, "input symlink target could not be resolved")
		return "", rootReal, resolveErr
	}
	if !pathWithin(rootReal, resolved) {
		v.markUnknownLocked(producer, "input symlink resolves outside the bound project root")
		return "", rootReal, errGovernanceReadViewOutsideRoot
	}
	return filepath.Clean(resolved), filepath.Clean(rootReal), nil
}

func (v *governanceReadView) readStringLocked(method, baseName, path, producer string, read func(string) (string, error)) (string, error) {
	o := v.observeLocked(method, path, producer)
	if o == nil {
		return "", errGovernanceReadViewBudget
	}
	if o != nil && (o.value != "" || o.err != nil || o.baseName != "") {
		if o.err != nil && !errors.Is(o.err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "filesystem metadata could not be read")
		}
		return o.value, o.err
	}
	value, err := read(baseName)
	if o != nil {
		o.baseName, o.value, o.err, o.errText = baseName, value, err, errorText(err)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		v.markUnknownLocked(producer, "filesystem metadata could not be read")
	}
	return value, err
}

func (v *governanceReadView) readStatLocked(method, baseName, path, producer string, read func(string) (fs.FileInfo, error)) (fs.FileInfo, error) {
	o := v.observeLocked(method, path, producer)
	if o == nil {
		return nil, errGovernanceReadViewBudget
	}
	if o != nil && (o.info != nil || o.err != nil || o.baseName != "") {
		if o.err != nil && !errors.Is(o.err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "filesystem metadata could not be read")
		}
		return frozenInfo(o.info), o.err
	}
	info, err := read(baseName)
	if o != nil {
		o.baseName, o.info, o.witness, o.err, o.errText = baseName, frozenInfo(info), info, err, errorText(err)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		v.markUnknownLocked(producer, "filesystem metadata could not be read")
	}
	return frozenInfo(info), err
}

func (v *governanceReadView) observeLocked(method, path, producer string) *governanceReadObservation {
	key := method + "\x00" + path
	if o := v.observed[key]; o != nil {
		if producer != "" {
			o.producers[producer] = true
		}
		return o
	}
	if producer == "" {
		producer = "projection"
	}
	metadataCost := int64(len(method) + len(v.displayPath(path)) + 64)
	if v.operation != nil {
		if !v.operation.reserveRecord(metadataCost) {
			v.markUnknownLocked(producer, "read view exceeded its operation record or metadata budget")
			return nil
		}
	} else if len(v.observed)+1 > governanceReadViewMaxRecords || metadataCost > governanceReadViewMaxMetadata-v.metadataBytes {
		v.markUnknownLocked(producer, "read view exceeded its record or metadata budget")
		return nil
	}
	v.metadataBytes += metadataCost
	if v.metadataBytes > governanceReadViewMaxMetadata {
		v.markUnknownLocked(producer, "read view exceeded its metadata budget")
		return nil
	}
	o := &governanceReadObservation{method: method, path: path, producers: map[string]bool{producer: true}}
	v.observed[key] = o
	if !pathWithin(v.root, path) {
		v.markUnknownLocked(producer, "input is outside the bound project root")
	}
	return o
}

func (v *governanceReadView) readFile(name, producer string) ([]byte, error) {
	if err := v.begin(); err != nil {
		return nil, err
	}
	defer v.end()
	path := v.pathKey(name)
	v.mu.Lock()
	if cached := v.files[path]; cached != nil {
		cached.producers[producer] = true
		data := append([]byte(nil), cached.bytes...)
		err := cached.err
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input bytes could not be read")
		}
		v.mu.Unlock()
		return data, err
	}
	remaining := governanceReadViewMaxInputBytes - v.inputBytes
	if v.operation != nil && v.operation.remainingInput() < remaining {
		remaining = v.operation.remainingInput()
	}
	if remaining < 0 || v.operation == nil && int64(len(v.files)+len(v.observed)+1) > governanceReadViewMaxRecords {
		v.markUnknownLocked(producer, "file bytes were not retained within the read view budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	resolved, rootReal, err := v.resolveForReadLocked(path, producer)
	if err != nil {
		v.mu.Unlock()
		return nil, err
	}
	if v.operation != nil {
		metadataCost := int64(len("file") + len(v.displayPath(path)) + 64)
		if !v.operation.reserveRecord(metadataCost) {
			v.markUnknownLocked(producer, "read view exceeded its operation record or metadata budget")
			v.mu.Unlock()
			return nil, errGovernanceReadViewBudget
		}
	}
	preflight, preflightErr := v.base.Stat(path)
	if preflightErr != nil {
		if !errors.Is(preflightErr, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input metadata could not be read before opening")
		}
		v.files[path] = &governanceReadFileObservation{err: preflightErr, errText: errorText(preflightErr), source: "readfile", baseName: path, rootReal: rootReal, resolved: resolved, producers: map[string]bool{producer: true}}
		v.mu.Unlock()
		return nil, preflightErr
	}
	if !preflight.Mode().IsRegular() {
		err := fmt.Errorf("pose: read view cannot capture non-regular file")
		v.markUnknownLocked(producer, "input is not a regular file")
		v.mu.Unlock()
		return nil, err
	}
	if preflight.Size() < 0 {
		v.markUnknownLocked(producer, "input size is unavailable before opening")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	if preflight.Size() > remaining {
		v.markUnknownLocked(producer, "read view exceeded its input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	reserved := preflight.Size()
	if v.operation != nil && !v.operation.reserveInput(reserved) {
		v.markUnknownLocked(producer, "read view exceeded its operation input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	// The second stat/open sequence is constrained to the exact size whose
	// aggregate budget reservation succeeded above. A concurrent growth must
	// fail closed before retaining bytes that the operation did not reserve.
	raw, info, witness, err := readGovernanceFileLimited(v.base, path, reserved, rootReal, resolved)
	if v.operation != nil && reserved > 0 {
		v.operation.finishInput(reserved, int64(len(raw)))
	}
	if errors.Is(err, errGovernanceReadViewBudget) {
		v.markUnknownLocked(producer, "read view exceeded its input byte budget")
		v.mu.Unlock()
		return nil, err
	}
	if int64(len(raw)) > remaining {
		v.markUnknownLocked(producer, "read view exceeded its input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		v.markUnknownLocked(producer, "input bytes could not be read")
	}
	v.inputBytes += int64(len(raw))
	if err == nil {
		v.observeCapturedFileStatLocked(path, info, witness, producer)
	}
	v.files[path] = &governanceReadFileObservation{bytes: append([]byte(nil), raw...), readLimit: reserved, err: err, errText: errorText(err), source: "readfile", baseName: path, info: frozenInfo(info), witness: witness, rootReal: rootReal, resolved: resolved, producers: map[string]bool{producer: true}}
	v.mu.Unlock()
	return raw, err
}

func (v *governanceReadView) open(name, producer string) (fs.File, error) {
	if err := v.begin(); err != nil {
		return nil, err
	}
	defer v.end()
	path := v.pathKey(name)
	v.mu.Lock()
	if cached := v.files[path]; cached != nil {
		cached.producers[producer] = true
		data, err, info := append([]byte(nil), cached.bytes...), cached.err, cached.info
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input bytes could not be read")
		}
		v.mu.Unlock()
		if err != nil {
			return nil, err
		}
		if info == nil {
			var err error
			info, err = v.statInfo(path, producer, false)
			if err != nil {
				return nil, err
			}
		}
		return newGovernanceReadFile(data, info), nil
	}
	resolved, rootReal, resolveErr := v.resolveForReadLocked(path, producer)
	if resolveErr != nil {
		v.mu.Unlock()
		return nil, resolveErr
	}
	preflight, preflightErr := v.base.Stat(path)
	if preflightErr != nil {
		if !errors.Is(preflightErr, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input metadata could not be read before opening")
		}
		v.files[path] = &governanceReadFileObservation{err: preflightErr, errText: errorText(preflightErr), source: "open", baseName: path, rootReal: rootReal, resolved: resolved, producers: map[string]bool{producer: true}}
		v.mu.Unlock()
		return nil, preflightErr
	}
	if preflight.IsDir() {
		v.mu.Unlock()
		info, statErr := v.statInfo(path, producer, false)
		if statErr != nil {
			return nil, statErr
		}
		if info == nil || !info.IsDir() {
			v.markUnknown(producer, "directory changed while its metadata was captured")
			return nil, errGovernanceReadViewOutsideRoot
		}
		entries, dirErr := v.readDir(path, producer)
		if dirErr != nil {
			return nil, dirErr
		}
		return newGovernanceReadDirectoryFile(path, producer, entries, info), nil
	}
	if !preflight.Mode().IsRegular() {
		v.markUnknownLocked(producer, "input is not a regular file")
		v.mu.Unlock()
		return nil, fmt.Errorf("pose: read view cannot capture non-regular file %s", v.displayPath(path))
	}
	remaining := governanceReadViewMaxInputBytes - v.inputBytes
	if v.operation != nil && v.operation.remainingInput() < remaining {
		remaining = v.operation.remainingInput()
	}
	if remaining < 0 || v.operation == nil && int64(len(v.files)+len(v.observed)+1) > governanceReadViewMaxRecords {
		v.markUnknownLocked(producer, "file bytes were not retained within the read view budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	if v.operation != nil {
		metadataCost := int64(len("file") + len(v.displayPath(path)) + 64)
		if !v.operation.reserveRecord(metadataCost) {
			v.markUnknownLocked(producer, "read view exceeded its operation record or metadata budget")
			v.mu.Unlock()
			return nil, errGovernanceReadViewBudget
		}
	}
	if preflight.Size() < 0 {
		v.markUnknownLocked(producer, "input size is unavailable before opening")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	if preflight.Size() > remaining {
		v.markUnknownLocked(producer, "read view exceeded its input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	reserved := preflight.Size()
	if v.operation != nil && !v.operation.reserveInput(reserved) {
		v.markUnknownLocked(producer, "read view exceeded its operation input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	captured := int64(0)
	if v.operation != nil && reserved > 0 {
		defer func() { v.operation.finishInput(reserved, captured) }()
	}
	f, err := v.base.Open(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input bytes could not be read")
		}
		v.files[path] = &governanceReadFileObservation{readLimit: reserved, err: err, errText: errorText(err), source: "open", baseName: path, rootReal: rootReal, resolved: resolved, producers: map[string]bool{producer: true}}
		v.mu.Unlock()
		return nil, err
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() || info.Size() < 0 {
		_ = f.Close()
		v.markUnknownLocked(producer, "opened file is not a capturable regular file")
		v.mu.Unlock()
		if statErr != nil {
			return nil, statErr
		}
		return nil, fmt.Errorf("pose: read view cannot capture non-regular file %s", v.displayPath(path))
	}
	if !sameObservedFile(preflight, info) {
		_ = f.Close()
		v.markUnknownLocked(producer, "input changed between preflight and open")
		v.mu.Unlock()
		return nil, errGovernanceReadViewOutsideRoot
	}
	currentResolved, resolveErr := v.base.EvalSymlinks(path)
	currentInfo, pathStatErr := v.base.Stat(path)
	if resolveErr != nil || !pathWithin(rootReal, currentResolved) || filepath.Clean(currentResolved) != resolved || pathStatErr != nil || !sameObservedFile(info, currentInfo) {
		_ = f.Close()
		v.markUnknownLocked(producer, "input path changed while opening a captured file")
		v.mu.Unlock()
		return nil, errGovernanceReadViewOutsideRoot
	}
	if info.Size() > reserved {
		_ = f.Close()
		v.markUnknownLocked(producer, "input grew beyond its reserved byte budget before opening")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	raw, readErr := readLimited(f, preflight.Size())
	closeErr := f.Close()
	captured = int64(len(raw))
	if readErr == nil {
		readErr = closeErr
	}
	if errors.Is(readErr, errGovernanceReadViewBudget) || int64(len(raw)) > remaining {
		v.markUnknownLocked(producer, "read view exceeded its input byte budget")
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	v.inputBytes += int64(len(raw))
	if readErr != nil {
		v.markUnknownLocked(producer, "input bytes could not be read")
	}
	v.files[path] = &governanceReadFileObservation{bytes: append([]byte(nil), raw...), readLimit: reserved, err: readErr, errText: errorText(readErr), source: "open", baseName: path, info: frozenInfo(info), witness: info, rootReal: rootReal, resolved: resolved, producers: map[string]bool{producer: true}}
	v.observeCapturedFileStatLocked(path, frozenInfo(info), info, producer)
	v.mu.Unlock()
	return newGovernanceReadFile(raw, frozenInfo(info), readErr), nil
}

// observeCapturedFileStatLocked records the same successful file metadata
// whether the consumer first asked for bytes through ReadFile or Open. If a
// separate Stat already observed a different identity, retain that first
// witness and mark the producer unknown instead of overwriting it.
func (v *governanceReadView) observeCapturedFileStatLocked(path string, info, witness fs.FileInfo, producer string) {
	if info == nil || witness == nil {
		return
	}
	stat := v.observeLocked("stat", path, producer)
	if stat == nil {
		return
	}
	if stat.info == nil && stat.err == nil {
		stat.baseName, stat.info, stat.witness = path, frozenInfo(info), witness
		return
	}
	if stat.err != nil || stat.info == nil || !sameObservedFile(stat.witness, witness) {
		v.markUnknownLocked(producer, "file metadata changed during the read-view capture")
	}
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, errGovernanceReadViewBudget
	}
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(raw)) > limit {
		return nil, errGovernanceReadViewBudget
	}
	return raw, err
}

// readDirGovernanceBounded reads at most max entries from a directory. Native
// files implement fs.ReadDirFile, so the normal path is bounded while reading;
// Files implementations without streaming support are checked before their
// entries are retained.
func readDirGovernanceBounded(files Files, name string, maxEntries int) ([]fs.DirEntry, error) {
	return readDirGovernanceBoundedWithBudget(files, name, maxEntries, governanceReadViewMaxMetadata)
}

func readDirGovernanceBoundedWithBudget(files Files, name string, maxEntries int, maxMetadata int64) ([]fs.DirEntry, error) {
	if maxEntries < 0 {
		return nil, errGovernanceReadViewBudget
	}
	if maxMetadata < 0 {
		return nil, errGovernanceReadViewBudget
	}
	file, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader, ok := file.(interface {
		ReadDir(int) ([]fs.DirEntry, error)
	})
	if !ok {
		return nil, fmt.Errorf("%w: filesystem does not support bounded directory reads", errGovernanceReadViewBudget)
	}
	capacity := maxEntries
	if capacity > 256 {
		capacity = 256
	}
	entries := make([]fs.DirEntry, 0, capacity)
	var metadataBytes int64
	for len(entries) <= maxEntries {
		remaining := maxEntries + 1 - len(entries)
		batch := remaining
		if batch > 256 {
			batch = 256
		}
		if batch < 1 {
			return nil, errGovernanceReadViewBudget
		}
		part, readErr := reader.ReadDir(batch)
		batchMetadata := readViewDirMetadataBytes(part)
		if batchMetadata > maxMetadata-metadataBytes {
			return nil, errGovernanceReadViewBudget
		}
		metadataBytes += batchMetadata
		entries = append(entries, part...)
		if len(entries) > maxEntries {
			return nil, errGovernanceReadViewBudget
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
				return entries, nil
			}
			return nil, readErr
		}
	}
	return nil, errGovernanceReadViewBudget
}

func readGovernanceFileLimited(files Files, name string, limit int64, rootReal, expectedResolved string) ([]byte, fs.FileInfo, fs.FileInfo, error) {
	preflight, err := files.Stat(name)
	if err != nil {
		return nil, nil, nil, err
	}
	if !preflight.Mode().IsRegular() {
		return nil, nil, nil, fmt.Errorf("pose: read view cannot capture non-regular file")
	}
	if preflight.Size() < 0 || preflight.Size() > limit {
		return nil, nil, nil, errGovernanceReadViewBudget
	}
	file, err := files.Open(name)
	if err != nil {
		return nil, nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 {
		_ = file.Close()
		return nil, nil, nil, fmt.Errorf("pose: read view cannot capture non-regular file")
	}
	if info.Size() > limit {
		_ = file.Close()
		return nil, nil, nil, errGovernanceReadViewBudget
	}
	if !sameObservedFile(preflight, info) {
		_ = file.Close()
		return nil, nil, nil, errGovernanceReadViewOutsideRoot
	}
	resolved, resolveErr := files.EvalSymlinks(name)
	current, statErr := files.Stat(name)
	if resolveErr != nil || !pathWithin(rootReal, resolved) || filepath.Clean(resolved) != expectedResolved || statErr != nil || !sameObservedFile(info, current) {
		_ = file.Close()
		if resolveErr != nil {
			return nil, nil, nil, resolveErr
		}
		if statErr != nil {
			return nil, nil, nil, statErr
		}
		return nil, nil, nil, errGovernanceReadViewOutsideRoot
	}
	raw, readErr := readLimited(file, limit)
	closeErr := file.Close()
	if errors.Is(readErr, errGovernanceReadViewBudget) {
		return nil, nil, nil, readErr
	}
	if readErr == nil {
		readErr = closeErr
	}
	return raw, frozenInfo(info), info, readErr
}

// reobserveGovernanceFileFailure repeats the metadata/open portion of a read
// that failed before retaining any bytes. This compares a stable permission
// denial without asking the byte reader to replay it with a zero-byte limit.
func reobserveGovernanceFileFailure(files Files, name, rootReal, expectedResolved string) error {
	preflight, err := files.Stat(name)
	if err != nil {
		return err
	}
	if !preflight.Mode().IsRegular() {
		return fmt.Errorf("pose: read view cannot capture non-regular file")
	}
	resolved, err := files.EvalSymlinks(name)
	if err != nil {
		return err
	}
	if !pathWithin(rootReal, resolved) || filepath.Clean(resolved) != expectedResolved {
		return errGovernanceReadViewOutsideRoot
	}
	file, err := files.Open(name)
	if err != nil {
		return err
	}
	info, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return statErr
	}
	if !info.Mode().IsRegular() || !sameObservedFile(preflight, info) {
		_ = file.Close()
		return errGovernanceReadViewOutsideRoot
	}
	current, statErr := files.Stat(name)
	currentResolved, resolveErr := files.EvalSymlinks(name)
	if statErr != nil || resolveErr != nil || !sameObservedFile(info, current) || !pathWithin(rootReal, currentResolved) || filepath.Clean(currentResolved) != expectedResolved {
		_ = file.Close()
		if statErr != nil {
			return statErr
		}
		if resolveErr != nil {
			return resolveErr
		}
		return errGovernanceReadViewOutsideRoot
	}
	return file.Close()
}

func (v *governanceReadView) statInfo(name, producer string, link bool) (fs.FileInfo, error) {
	if err := v.begin(); err != nil {
		return nil, err
	}
	defer v.end()
	path := v.pathKey(name)
	method := "stat"
	if link {
		method = "lstat"
	}
	v.mu.Lock()
	if link {
		if err := v.checkParentBoundLocked(path, producer); err != nil {
			v.mu.Unlock()
			return nil, err
		}
	} else {
		if _, _, err := v.resolveForReadLocked(path, producer); err != nil {
			v.mu.Unlock()
			return nil, err
		}
	}
	read := v.base.Stat
	if link {
		read = v.base.Lstat
	}
	info, err := v.readStatLocked(method, path, path, producer, read)
	v.mu.Unlock()
	return info, err
}

func (v *governanceReadView) checkParentBoundLocked(name, producer string) error {
	rootAbs, err := filepath.Abs(v.root)
	if err != nil {
		v.markUnknownLocked(producer, "authorized root identity could not be resolved")
		return err
	}
	path := v.pathKey(name)
	if !pathWithin(rootAbs, path) {
		v.markUnknownLocked(producer, "input is outside the bound project root")
		return errGovernanceReadViewOutsideRoot
	}
	rootReal, err := v.readStringLocked("root-eval-symlinks", rootAbs, rootAbs, producer, v.base.EvalSymlinks)
	if err != nil {
		v.markUnknownLocked(producer, "authorized root symlink could not be resolved")
		return err
	}
	if filepath.Clean(path) == filepath.Clean(rootAbs) {
		return nil
	}
	parentPath := filepath.Dir(path)
	parent, err := v.readStringLocked("eval-parent-symlinks", parentPath, parentPath, producer, v.base.EvalSymlinks)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "input parent symlink could not be resolved")
		}
		return err
	}
	if !pathWithin(rootReal, parent) {
		v.markUnknownLocked(producer, "input parent resolves outside the bound project root")
		return errGovernanceReadViewOutsideRoot
	}
	return nil
}

func (v *governanceReadView) readDir(name, producer string) ([]fs.DirEntry, error) {
	if err := v.begin(); err != nil {
		return nil, err
	}
	defer v.end()
	path := v.pathKey(name)
	v.mu.Lock()
	key := "directory\x00" + path
	if o := v.observed[key]; o != nil {
		o.producers[producer] = true
		entries := cloneReadViewDirForProducer(oDirEntries(o), producer)
		err := o.err
		v.mu.Unlock()
		return entries, err
	}
	o := v.observeLocked("directory", path, producer)
	if o == nil {
		v.mu.Unlock()
		return nil, errGovernanceReadViewBudget
	}
	_, _, err := v.resolveForReadLocked(path, producer)
	var entries []fs.DirEntry
	if err == nil {
		if v.operation != nil {
			entries, err = v.operation.readDirectory(v.base, path)
		} else {
			entries, err = readDirGovernanceBounded(v.base, path, governanceReadViewMaxRecords-len(v.observed)-len(v.files))
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errGovernanceReadViewBudget) {
		v.markUnknownLocked(producer, "directory listing could not be read")
	}
	if errors.Is(err, errGovernanceReadViewBudget) {
		v.markUnknownLocked(producer, "read view exceeded its directory-entry budget")
	}
	wrapped := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		wrapped = append(wrapped, &readViewDirEntry{view: v, dir: path, entry: entry, producer: producer, name: entry.Name(), modeType: entry.Type()})
	}
	if o != nil {
		o.baseName = path
		o.entries = dirEntryIdentities(wrapped)
		o.err, o.errText = err, errorText(err)
		o.dirEntries = cloneReadViewDir(wrapped)
		v.metadataBytes += readViewDirMetadataBytes(wrapped)
		if v.metadataBytes > governanceReadViewMaxMetadata {
			v.markUnknownLocked(producer, "read view exceeded its metadata budget")
		}
	}
	v.mu.Unlock()
	return cloneReadViewDir(wrapped), err
}

func (v *governanceReadView) readDirEntryInfoLocked(dir, name string, entry fs.DirEntry, producer string) (fs.FileInfo, error) {
	path := filepath.Join(dir, name)
	o := v.observeLocked("direntry-info", path, producer)
	if o != nil && (o.info != nil || o.err != nil || o.baseName != "") {
		if o.err != nil && !errors.Is(o.err, os.ErrNotExist) {
			v.markUnknownLocked(producer, "directory entry metadata could not be read")
		}
		return frozenInfo(o.info), o.err
	}
	info, err := entry.Info()
	if o != nil {
		o.baseName, o.info, o.witness, o.err, o.errText = dir, frozenInfo(info), info, err, errorText(err)
	}
	if err != nil {
		v.markUnknownLocked(producer, "directory entry metadata could not be read")
	}
	return frozenInfo(info), err
}

func (v *governanceReadView) readDirEntryInfo(dir string, entry fs.DirEntry, producer string) (fs.FileInfo, error) {
	if err := v.begin(); err != nil {
		return nil, err
	}
	defer v.end()
	v.mu.Lock()
	info, err := v.readDirEntryInfoLocked(dir, entry.Name(), entry, producer)
	v.mu.Unlock()
	return info, err
}

func (v *governanceReadView) readString(name, producer, method string) (string, error) {
	if err := v.begin(); err != nil {
		return "", err
	}
	defer v.end()
	path := v.pathKey(name)
	v.mu.Lock()
	var value string
	var err error
	if method == "readlink" {
		err = v.checkParentBoundLocked(path, producer)
		if err == nil {
			value, err = v.readStringLocked(method, path, path, producer, v.base.Readlink)
		}
	} else {
		if _, _, err = v.resolveForReadLocked(path, producer); err == nil {
			value, err = v.readStringLocked(method, path, path, producer, v.base.EvalSymlinks)
			if err == nil && !pathWithin(v.root, value) {
				_, rootReal, rootErr := v.resolveForReadLocked(path, producer)
				if rootErr != nil || !pathWithin(rootReal, value) {
					v.markUnknownLocked(producer, "input symlink resolves outside the bound project root")
					err = errGovernanceReadViewOutsideRoot
				}
			}
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errGovernanceReadViewOutsideRoot) {
		v.markUnknownLocked(producer, "filesystem metadata could not be read")
	}
	v.mu.Unlock()
	return value, err
}

func (v *governanceReadView) audit() governanceReadViewAudit {
	if v.operation != nil && v.operation.primary == v {
		return v.operation.audit()
	}
	return v.auditSingle()
}

func (v *governanceReadView) auditSingle() governanceReadViewAudit {
	return v.auditSingleWithBudget(&governanceReadViewAuditBudget{})
}

func (v *governanceReadView) auditSingleWithBudget(budget *governanceReadViewAuditBudget) governanceReadViewAudit {
	v.mu.Lock()
	v.closed = true
	for v.active > 0 {
		v.cond.Wait()
	}
	observations := make([]governanceReadObservation, 0, len(v.observed))
	for _, o := range v.observed {
		copy := *o
		copy.producers = cloneBoolMap(o.producers)
		copy.entries = append([]string(nil), o.entries...)
		copy.dirEntries = cloneReadViewDir(o.dirEntries)
		observations = append(observations, copy)
	}
	files := make(map[string]governanceReadFileObservation, len(v.files))
	for path, item := range v.files {
		copy := *item
		copy.bytes = append([]byte(nil), item.bytes...)
		copy.producers = cloneBoolMap(item.producers)
		files[path] = copy
	}
	unknown := unknownReasons(v.unknown)
	v.mu.Unlock()

	result := governanceReadViewAudit{ChangedProducers: map[string]bool{}, UnknownProducers: unknown}
	markBudgetFailure := func(path string, producers map[string]bool, reason string) {
		result.ChangedPaths = append(result.ChangedPaths, v.displayPath(path))
		for producer := range producers {
			result.ChangedProducers[producer] = true
			result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], reason)
		}
	}
	for _, item := range files {
		if item.witness != nil && fileInfoIdentityDigest(item.witness) == "" {
			for producer := range item.producers {
				result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], "filesystem does not expose a stable file identity")
			}
		}
	}
	for _, item := range observations {
		if item.witness != nil && fileInfoIdentityDigest(item.witness) == "" {
			for producer := range item.producers {
				result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], "filesystem does not expose a stable file identity")
			}
		}
	}
	for path, item := range files {
		metadataCost := int64(len("file") + len(v.displayPath(path)) + len(item.errText) + 128)
		if !budget.reserve(metadataCost) {
			markBudgetFailure(path, item.producers, "final read-view audit exceeded its record or metadata budget")
			continue
		}
		var raw []byte
		var err error
		var witness fs.FileInfo
		if item.source == "open" || item.source == "readfile" {
			if item.errText != "" && len(item.bytes) == 0 {
				err = reobserveGovernanceFileFailure(v.base, item.baseName, item.rootReal, item.resolved)
			} else {
				if !budget.reserveInput(item.readLimit) {
					markBudgetFailure(path, item.producers, "final read-view audit exceeded its input byte budget")
					continue
				}
				raw, _, witness, err = readGovernanceFileLimited(v.base, item.baseName, item.readLimit, item.rootReal, item.resolved)
			}
		}
		if errorText(err) != item.errText || !bytes.Equal(raw, item.bytes) || item.witness != nil && !sameObservedFile(item.witness, witness) {
			for producer := range item.producers {
				result.ChangedProducers[producer] = true
			}
			result.ChangedPaths = append(result.ChangedPaths, v.displayPath(path))
		}
	}
	for _, item := range observations {
		metadataCost := int64(len(item.method) + len(v.displayPath(item.path)) + len(item.errText) + len(item.value) + 128)
		if !budget.reserve(metadataCost) {
			markBudgetFailure(item.path, item.producers, "final read-view audit exceeded its record or metadata budget")
			continue
		}
		changed := false
		switch item.method {
		case "stat":
			info, err := v.base.Stat(item.baseName)
			changed = errorText(err) != item.errText || err == nil && !sameObservedFile(item.witness, info)
		case "lstat":
			info, err := v.base.Lstat(item.baseName)
			changed = errorText(err) != item.errText || err == nil && !sameObservedFile(item.witness, info)
		case "directory":
			remainingRecords := governanceReadViewMaxRecords - budget.records
			remainingMetadata := governanceReadViewMaxMetadata - budget.metadata
			entries, err := readDirGovernanceBoundedWithBudget(v.base, item.baseName, remainingRecords, remainingMetadata)
			if errors.Is(err, errGovernanceReadViewBudget) {
				markBudgetFailure(item.path, item.producers, "final directory audit exceeded its bounded record or metadata budget")
				continue
			}
			if err == nil && !budget.reserveDirectory(entries, remainingRecords, remainingMetadata) {
				markBudgetFailure(item.path, item.producers, "final directory audit exceeded its bounded record or metadata budget")
				continue
			}
			changed = errorText(err) != item.errText || err == nil && strings.Join(item.entries, "\x00") != strings.Join(dirEntryIdentities(entries), "\x00")
		case "direntry-info":
			remainingRecords := governanceReadViewMaxRecords - budget.records
			remainingMetadata := governanceReadViewMaxMetadata - budget.metadata
			entries, err := readDirGovernanceBoundedWithBudget(v.base, item.baseName, remainingRecords, remainingMetadata)
			if errors.Is(err, errGovernanceReadViewBudget) {
				markBudgetFailure(item.path, item.producers, "final directory audit exceeded its bounded record or metadata budget")
				continue
			}
			if err == nil && !budget.reserveDirectory(entries, remainingRecords, remainingMetadata) {
				markBudgetFailure(item.path, item.producers, "final directory audit exceeded its bounded record or metadata budget")
				continue
			}
			if errorText(err) == item.errText && err == nil {
				found := false
				for _, entry := range entries {
					if entry.Name() == filepath.Base(item.path) {
						found = true
						info, infoErr := entry.Info()
						changed = errorText(infoErr) != item.errText || infoErr == nil && !sameObservedFile(item.witness, info)
						break
					}
				}
				changed = changed || !found
			} else {
				changed = errorText(err) != item.errText
			}
		case "readlink":
			value, err := v.base.Readlink(item.baseName)
			changed = errorText(err) != item.errText || err == nil && value != item.value
		case "eval-symlinks":
			value, err := v.base.EvalSymlinks(item.baseName)
			changed = errorText(err) != item.errText || err == nil && filepath.Clean(value) != filepath.Clean(item.value)
		case "eval-parent-symlinks":
			value, err := v.base.EvalSymlinks(item.baseName)
			changed = errorText(err) != item.errText || err == nil && filepath.Clean(value) != filepath.Clean(item.value)
		case "root-eval-symlinks":
			value, err := v.base.EvalSymlinks(item.baseName)
			changed = errorText(err) != item.errText || err == nil && filepath.Clean(value) != filepath.Clean(item.value)
		}
		if changed {
			result.ChangedPaths = append(result.ChangedPaths, v.displayPath(item.path))
			for producer := range item.producers {
				result.ChangedProducers[producer] = true
			}
		}
	}
	sort.Strings(result.ChangedPaths)
	return result
}

func (v *governanceReadView) metadata(project, revision, indexGeneration string, audit governanceReadViewAudit) (ReadViewMetadata, *governanceReadViewProof) {
	if v.operation != nil && v.operation.primary == v {
		return v.operation.metadata(project, revision, indexGeneration)
	}
	return v.metadataSingle(project, revision, indexGeneration, audit)
}

func (v *governanceReadView) metadataSingle(project, revision, indexGeneration string, audit governanceReadViewAudit) (ReadViewMetadata, *governanceReadViewProof) {
	v.mu.Lock()
	proof := &governanceReadViewProof{root: v.root, inputBudget: governanceReadViewMaxInputBytes, base: v.base}
	unknown := cloneUnknown(v.unknown)
	rootObservation := v.observed["root-eval-symlinks\x00"+v.pathKey(v.root)]
	rootReal := v.root
	if rootObservation != nil && rootObservation.err == nil && rootObservation.value != "" {
		rootReal = filepath.Clean(rootObservation.value)
	} else {
		unknown["projection"] = mergeReason(unknown["projection"], "authorized root binding is unavailable")
	}
	for path, file := range v.files {
		proof.entries = append(proof.entries, governanceReadProofEntry{method: file.source, path: path, baseName: file.baseName, digest: digestBytes(file.bytes), errText: file.errText, info: frozenInfo(file.info), witness: frozenInfo(file.witness), rootReal: file.rootReal, resolved: file.resolved, inputBytes: int64(len(file.bytes)), readLimit: file.readLimit, producers: sortedBoolKeys(file.producers)})
	}
	for _, observation := range v.observed {
		proof.entries = append(proof.entries, governanceReadProofEntry{method: observation.method, path: observation.path, baseName: observation.baseName, errText: observation.errText, info: frozenInfo(observation.info), witness: frozenInfo(observation.witness), entries: append([]string(nil), observation.entries...), value: observation.value, producers: sortedBoolKeys(observation.producers)})
	}
	for producer, reasons := range audit.UnknownProducers {
		for _, reason := range reasons {
			unknown[producer] = mergeReason(unknown[producer], reason)
		}
	}
	inputBytes := v.inputBytes
	observedInputs := len(v.observed) + len(v.files)
	v.mu.Unlock()

	rootRealAbs, err := filepath.Abs(rootReal)
	if err == nil {
		rootReal = rootRealAbs
	}
	rootDigest := digestBytes([]byte(rootReal))
	if err != nil {
		unknown["projection"] = mergeReason(unknown["projection"], "authorized root identity could not be resolved")
	}
	engineVersion := EngineVersion
	if engineVersion == "" || engineVersion == "unknown" {
		engineVersion = "unknown"
		unknown["projection"] = mergeReason(unknown["projection"], "engine version is unavailable")
	}
	parserVersions := map[string]string{"governance-inputs": governanceReadViewParserVersion}
	bindings := []ReadViewBinding{{ProjectID: project, RootDigest: rootDigest, SourceRevision: revision, IndexGeneration: indexGeneration}}
	if project == "" {
		unknown["projection"] = mergeReason(unknown["projection"], "project binding is unavailable")
	}
	allowMissingRevision := v.operation != nil && v.operation.primary == v && v.operation.allowMissingPrimarySourceRevision && revision == ""
	if (revision == "" || revision == "unknown") && !allowMissingRevision {
		unknown["projection"] = mergeReason(unknown["projection"], "source revision binding is unavailable")
	}
	identityEntries := make([]canonicalReadViewEntry, 0, len(proof.entries))
	for _, entry := range proof.entries {
		method := entry.method
		if method == "readfile" || method == "open" {
			method = "file"
		}
		canonical := canonicalReadViewEntry{Method: method, Path: v.displayPath(entry.path), ErrorDigest: errorDigest(entry.errText)}
		if entry.digest != "" {
			canonical.Digest = entry.digest
		}
		if entry.info != nil {
			canonical.Name, canonical.Size, canonical.Mode, canonical.IdentityDigest = entry.info.Name(), entry.info.Size(), uint32(entry.info.Mode()), fileInfoIdentityDigest(entry.info)
			if canonical.IdentityDigest == "" {
				for _, producer := range entry.producers {
					unknown[producer] = mergeReason(unknown[producer], "filesystem does not expose a stable file identity")
				}
			}
			if method == "file" || method == "stat" || method == "lstat" || method == "directory" {
				canonical.ModTime = entry.info.ModTime().UnixNano()
			}
		}
		canonical.Entries = append([]string(nil), entry.entries...)
		if entry.value != "" {
			canonical.ValueDigest = digestBytes([]byte(entry.value))
		}
		if entry.resolved != "" {
			canonical.ResolvedDigest = digestBytes([]byte(filepath.Clean(entry.resolved)))
		}
		identityEntries = append(identityEntries, canonical)
	}
	sort.Slice(identityEntries, func(i, j int) bool {
		if identityEntries[i].Method != identityEntries[j].Method {
			return identityEntries[i].Method < identityEntries[j].Method
		}
		return identityEntries[i].Path < identityEntries[j].Path
	})
	identity := struct {
		SchemaVersion  int                      `json:"schema_version"`
		EngineVersion  string                   `json:"engine_version"`
		ParserVersions map[string]string        `json:"parser_versions"`
		Bindings       []ReadViewBinding        `json:"bindings"`
		Entries        []canonicalReadViewEntry `json:"entries"`
		Unknown        map[string][]string      `json:"unknown,omitempty"`
		ChangedPaths   []string                 `json:"changed_paths,omitempty"`
	}{GovernanceReadViewSchemaVersion, engineVersion, parserVersions, bindings, identityEntries, unknownReasons(unknown), uniqueSorted(append([]string(nil), audit.ChangedPaths...))}
	raw, _ := json.Marshal(identity)
	sum := sha256.Sum256(raw)
	freshness := ReadViewFreshnessCurrent
	if len(unknown) != 0 || len(audit.ChangedPaths) != 0 || len(audit.ChangedProducers) != 0 || len(audit.UnknownProducers) != 0 || err != nil {
		freshness = ReadViewFreshnessUnknown
	}
	limitations := []string{}
	for producer, reasons := range unknownReasons(unknown) {
		for _, reason := range reasons {
			if producer == "projection" {
				limitations = append(limitations, "read view: "+reason)
			} else {
				limitations = append(limitations, "producer "+producer+": "+reason)
			}
		}
	}
	for _, path := range uniqueSorted(append([]string(nil), audit.ChangedPaths...)) {
		limitations = append(limitations, "observed input changed before final validation: "+path)
	}
	metadata := ReadViewMetadata{SchemaVersion: GovernanceReadViewSchemaVersion, ID: "sha256:" + hex.EncodeToString(sum[:]), Freshness: freshness, EngineVersion: engineVersion, ParserVersions: parserVersions, Bindings: bindings, ObservedInputs: observedInputs, InputBytes: inputBytes, Limitations: uniqueSorted(limitations)}
	return metadata, proof
}

func (p *governanceReadViewProof) changes(files Files) []string {
	if p == nil {
		return []string{"read-view-unverified"}
	}
	return p.changesWithBudget(files, &governanceReadProofBudget{})
}

type governanceReadProofBudget struct {
	inputBytes int64
	metadata   int64
	records    int
}

func (p *governanceReadViewProof) changesWithBudget(files Files, budget *governanceReadProofBudget) []string {
	if p == nil {
		return []string{"read-view-unverified"}
	}
	if len(p.children) > 0 {
		changed := []string{}
		for _, child := range p.children {
			changed = append(changed, child.changesWithBudget(child.base, budget)...)
			for _, item := range changed {
				if strings.HasPrefix(item, "read-view-unverified:") {
					return uniqueSorted(changed)
				}
			}
		}
		return uniqueSorted(changed)
	}
	if files == nil && p.base != nil {
		files = p.base
	}
	files = FilesOrNative(files)
	changed := []string{}
	for _, item := range p.entries {
		metadataCost := int64(len(item.method) + len(p.displayPath(item.path)) + len(item.errText) + len(item.value) + len(item.digest) + 128)
		if budget.records+1 > governanceReadViewMaxRecords || metadataCost > governanceReadViewMaxMetadata-budget.metadata {
			return []string{"read-view-unverified:record-budget"}
		}
		budget.records++
		budget.metadata += metadataCost
		var differs bool
		switch item.method {
		case "readfile", "open":
			var raw []byte
			var witness fs.FileInfo
			var err error
			if item.errText != "" && item.inputBytes == 0 {
				err = reobserveGovernanceFileFailure(files, item.baseName, item.rootReal, item.resolved)
			} else {
				limit := p.inputBudget
				if limit > governanceReadViewMaxInputBytes {
					limit = governanceReadViewMaxInputBytes
				}
				if item.readLimit < 0 || item.readLimit > limit-budget.inputBytes || item.readLimit > governanceReadViewMaxInputBytes-budget.inputBytes {
					return []string{"read-view-unverified:input-byte-budget"}
				}
				budget.inputBytes += item.readLimit
				raw, _, witness, err = readGovernanceFileLimited(files, item.baseName, item.readLimit, item.rootReal, item.resolved)
			}
			differs = errorText(err) != item.errText || digestBytes(raw) != item.digest || item.witness != nil && !sameObservedFile(item.witness, witness)
		case "stat", "lstat":
			var info fs.FileInfo
			var err error
			if item.method == "stat" {
				info, err = files.Stat(item.baseName)
			} else {
				info, err = files.Lstat(item.baseName)
			}
			differs = errorText(err) != item.errText || err == nil && !sameObservedFile(item.info, info)
		case "directory":
			remainingRecords := governanceReadViewMaxRecords - budget.records
			remainingMetadata := governanceReadViewMaxMetadata - budget.metadata
			entries, err := readDirGovernanceBoundedWithBudget(files, item.baseName, remainingRecords, remainingMetadata)
			if errors.Is(err, errGovernanceReadViewBudget) {
				return []string{"read-view-unverified:metadata-budget"}
			}
			if err == nil {
				if budget.records+len(entries) > governanceReadViewMaxRecords || readViewDirMetadataBytes(entries) > governanceReadViewMaxMetadata-budget.metadata {
					return []string{"read-view-unverified:metadata-budget"}
				}
				budget.records += len(entries)
				budget.metadata += readViewDirMetadataBytes(entries)
			}
			differs = errorText(err) != item.errText || err == nil && !sameDirIdentities(item.entries, entries)
		case "direntry-info":
			remainingRecords := governanceReadViewMaxRecords - budget.records
			remainingMetadata := governanceReadViewMaxMetadata - budget.metadata
			entries, err := readDirGovernanceBoundedWithBudget(files, item.baseName, remainingRecords, remainingMetadata)
			if errors.Is(err, errGovernanceReadViewBudget) {
				return []string{"read-view-unverified:metadata-budget"}
			}
			if errorText(err) != item.errText || err != nil {
				differs = true
				break
			}
			if budget.records+len(entries) > governanceReadViewMaxRecords || readViewDirMetadataBytes(entries) > governanceReadViewMaxMetadata-budget.metadata {
				return []string{"read-view-unverified:metadata-budget"}
			}
			budget.records += len(entries)
			budget.metadata += readViewDirMetadataBytes(entries)
			found := false
			for _, entry := range entries {
				if entry.Name() == filepath.Base(item.path) {
					found = true
					info, infoErr := entry.Info()
					differs = errorText(infoErr) != item.errText || infoErr == nil && !sameObservedFile(item.witness, info)
					break
				}
			}
			differs = differs || !found
		case "readlink":
			value, err := files.Readlink(item.baseName)
			differs = errorText(err) != item.errText || err == nil && value != item.value
		case "eval-symlinks", "eval-parent-symlinks", "root-eval-symlinks":
			value, err := files.EvalSymlinks(item.baseName)
			differs = errorText(err) != item.errText || err == nil && value != item.value
		}
		if differs {
			changed = append(changed, p.displayPath(item.path))
		}
	}
	sort.Strings(changed)
	return changed
}

func (p *governanceReadViewProof) displayPath(path string) string {
	rel, err := filepath.Rel(p.root, path)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel)
	}
	return "external:" + digestBytes([]byte(filepath.Clean(path)))
}

type governanceReadViewFiles struct {
	view     *governanceReadView
	producer string
	project  string
}

func (f *governanceReadViewFiles) ReadFile(name string) ([]byte, error) {
	return f.view.route(name, f.producer, f.project).readFile(name, f.producer)
}
func (f *governanceReadViewFiles) Open(name string) (fs.File, error) {
	return f.view.route(name, f.producer, f.project).open(name, f.producer)
}
func (f *governanceReadViewFiles) Stat(name string) (fs.FileInfo, error) {
	return f.view.route(name, f.producer, f.project).statInfo(name, f.producer, false)
}
func (f *governanceReadViewFiles) Lstat(name string) (fs.FileInfo, error) {
	return f.view.route(name, f.producer, f.project).statInfo(name, f.producer, true)
}
func (f *governanceReadViewFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.view.route(name, f.producer, f.project).readDir(name, f.producer)
}
func (f *governanceReadViewFiles) Readlink(name string) (string, error) {
	return f.view.route(name, f.producer, f.project).readString(name, f.producer, "readlink")
}
func (f *governanceReadViewFiles) EvalSymlinks(name string) (string, error) {
	return f.view.route(name, f.producer, f.project).readString(name, f.producer, "eval-symlinks")
}
func (*governanceReadViewFiles) WriteFile(string, []byte, fs.FileMode) error {
	return errGovernanceReadViewWrite
}
func (*governanceReadViewFiles) CreateExclusive(string, []byte, fs.FileMode, bool) error {
	return errGovernanceReadViewWrite
}
func (*governanceReadViewFiles) Publish(string, []byte, fs.FileMode, string, PublishExpectation) error {
	return errGovernanceReadViewWrite
}
func (*governanceReadViewFiles) MkdirAll(string, fs.FileMode) error {
	return errGovernanceReadViewWrite
}
func (*governanceReadViewFiles) Mkdir(string, fs.FileMode) error { return errGovernanceReadViewWrite }
func (*governanceReadViewFiles) Remove(string) error             { return errGovernanceReadViewWrite }
func (*governanceReadViewFiles) RemoveAll(string) error          { return errGovernanceReadViewWrite }
func (*governanceReadViewFiles) Symlink(string, string) error    { return errGovernanceReadViewWrite }
func (f *governanceReadViewFiles) Preview() *PreviewFiles        { return f.view.base.Preview() }
func (f *governanceReadViewFiles) Now() time.Time                { return f.view.base.Now() }
func (f *governanceReadViewFiles) UnderlyingFiles() Files        { return f.view.base }
func (f *governanceReadViewFiles) GovernanceReadView() *governanceReadView {
	return f.view
}
func (f *governanceReadViewFiles) GovernanceReadViewRootFor(path string) string {
	return f.view.route(path, f.producer, f.project).root
}

// withoutGovernanceReadView gives a new operation the original Files binding,
// never an earlier observation that has already been finalized. Operation
// caches tied to that observation are dropped with it.
func (s Store) withoutGovernanceReadView() Store {
	if base, unwrapped := unwrapGovernanceReadViewFiles(s.fsys()); unwrapped {
		s.FS = base
	}
	if s.FederatedResolver != nil {
		resolver := *s.FederatedResolver
		if base, unwrapped := unwrapGovernanceReadViewFiles(resolver.fsys()); unwrapped {
			resolver.FS = base
		}
		if resolver.Roots != nil {
			rootFiles := resolver.Roots.fsys()
			if base, unwrapped := unwrapGovernanceReadViewFiles(rootFiles); unwrapped {
				roots := cloneRootsWithFiles(resolver.Roots, base)
				roots.governanceStores = map[string]Store{}
				resolver.Roots.mu.RLock()
				stores := make(map[string]Store, len(resolver.Roots.governanceStores))
				for id, store := range resolver.Roots.governanceStores {
					stores[id] = store
				}
				resolver.Roots.mu.RUnlock()
				for id, store := range stores {
					roots.governanceStores[id] = store.withoutGovernanceReadView()
				}
				resolver.Roots = roots
			}
		}
		resolver.snapshots = nil
		s.FederatedResolver = &resolver
	}
	s.specs = nil
	s.reviewView = nil
	s.reviewReuse = nil
	s.indexPin = nil
	return s
}

func unwrapGovernanceReadViewFiles(files Files) (Files, bool) {
	viewFiles, ok := FilesOrNative(files).(*governanceReadViewFiles)
	if !ok {
		return files, false
	}
	return viewFiles.view.base, true
}

func (s Store) withGovernanceReadView(view *governanceReadView, producer string) Store {
	s.FS = view.filesFor(producer)
	s.reviewView = nil
	s.reviewReuse = nil
	if s.FederatedResolver != nil {
		resolver := *s.FederatedResolver
		resolver.FS = s.FS
		if view.operation != nil {
			resolver.Roots = view.operation.rootsForProducer(resolver.Roots, s.FS, producer)
		} else {
			resolver.Roots = cloneRootsWithFiles(resolver.Roots, s.FS)
		}
		resolver.governanceDepthLimit = 8
		resolver.snapshots = nil
		s.FederatedResolver = &resolver
	}
	return s
}

func cloneRootsWithFiles(roots *Roots, files Files) *Roots {
	if roots == nil {
		return nil
	}
	roots.mu.RLock()
	clone := &Roots{
		cfg:              roots.cfg,
		byProject:        make(map[string]string, len(roots.byProject)),
		conflicts:        make(map[string]bool, len(roots.conflicts)),
		governanceStores: make(map[string]Store, len(roots.governanceStores)),
		lastScan:         roots.lastScan,
		rescanWindow:     roots.rescanWindow,
		fs:               files,
	}
	for project, root := range roots.byProject {
		clone.byProject[project] = root
	}
	for project, conflict := range roots.conflicts {
		clone.conflicts[project] = conflict
	}
	for project, store := range roots.governanceStores {
		clone.governanceStores[project] = store
	}
	roots.mu.RUnlock()
	return clone
}

func (op *governanceReadViewOperation) rootsForProducer(roots *Roots, files Files, producer string) *Roots {
	if roots == nil || op == nil {
		return nil
	}
	roots.mu.RLock()
	cfg := roots.cfg
	roots.mu.RUnlock()
	op.mu.Lock()
	rootMap, conflicts := op.initialRoots, op.initialConflicts
	op.mu.Unlock()
	return &Roots{
		cfg: cfg, byProject: rootMap, conflicts: conflicts, governanceStores: map[string]Store{},
		lastScan: time.Now(), rescanWindow: time.Duration(1 << 62), fs: files,
	}
}

type readViewFile struct {
	reader  *bytes.Reader
	info    fs.FileInfo
	readErr error
}

func newGovernanceReadFile(data []byte, info fs.FileInfo, readErrors ...error) *readViewFile {
	var readErr error
	if len(readErrors) > 0 {
		readErr = readErrors[0]
	}
	return &readViewFile{reader: bytes.NewReader(data), info: frozenInfo(info), readErr: readErr}
}
func (f *readViewFile) Stat() (fs.FileInfo, error) {
	if f.info == nil {
		return nil, fs.ErrInvalid
	}
	return f.info, nil
}
func (f *readViewFile) Read(p []byte) (int, error) {
	if f.reader.Len() == 0 && f.readErr != nil {
		return 0, f.readErr
	}
	n, err := f.reader.Read(p)
	if err == nil && n > 0 && f.readErr != nil {
		return n, f.readErr
	}
	return n, err
}
func (*readViewFile) Close() error { return nil }

type readViewDirectoryFile struct {
	name     string
	producer string
	entries  []fs.DirEntry
	info     fs.FileInfo
	cursor   int
}

func newGovernanceReadDirectoryFile(name, producer string, entries []fs.DirEntry, info fs.FileInfo) *readViewDirectoryFile {
	return &readViewDirectoryFile{
		name: name, producer: producer, entries: cloneReadViewDirForProducer(entries, producer), info: frozenInfo(info),
	}
}

func (f *readViewDirectoryFile) Stat() (fs.FileInfo, error) {
	if f == nil || f.info == nil {
		return nil, fs.ErrInvalid
	}
	return frozenInfo(f.info), nil
}

func (f *readViewDirectoryFile) Read([]byte) (int, error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}
	return 0, &fs.PathError{Op: "read", Path: f.name, Err: errGovernanceReadViewIsDirectory}
}

func (*readViewDirectoryFile) Close() error { return nil }

func (f *readViewDirectoryFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}
	if n <= 0 {
		entries := cloneReadViewDirForProducer(f.entries[f.cursor:], f.producer)
		f.cursor = len(f.entries)
		return entries, nil
	}
	if f.cursor >= len(f.entries) {
		return nil, io.EOF
	}
	end := f.cursor + n
	if end > len(f.entries) {
		end = len(f.entries)
	}
	entries := cloneReadViewDirForProducer(f.entries[f.cursor:end], f.producer)
	f.cursor = end
	if len(entries) < n {
		return entries, io.EOF
	}
	return entries, nil
}

type readViewDirEntry struct {
	view     *governanceReadView
	dir      string
	entry    fs.DirEntry
	producer string
	name     string
	modeType fs.FileMode
}

func (e *readViewDirEntry) Name() string      { return e.name }
func (e *readViewDirEntry) IsDir() bool       { return e.modeType.IsDir() }
func (e *readViewDirEntry) Type() fs.FileMode { return e.modeType }
func (e *readViewDirEntry) Info() (fs.FileInfo, error) {
	if e == nil || e.view == nil || e.entry == nil {
		return nil, fs.ErrInvalid
	}
	return e.view.readDirEntryInfo(e.dir, e.entry, e.producer)
}

func frozenInfo(info fs.FileInfo) fs.FileInfo {
	if info == nil {
		return nil
	}
	return readViewFileInfo{name: info.Name(), size: info.Size(), mode: info.Mode(), mod: info.ModTime(), sys: cloneFileInfoSys(info.Sys())}
}

type readViewFileInfo struct {
	name string
	size int64
	mode fs.FileMode
	mod  time.Time
	sys  any
}

func (i readViewFileInfo) Name() string       { return i.name }
func (i readViewFileInfo) Size() int64        { return i.size }
func (i readViewFileInfo) Mode() fs.FileMode  { return i.mode }
func (i readViewFileInfo) ModTime() time.Time { return i.mod }
func (i readViewFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i readViewFileInfo) Sys() any           { return cloneFileInfoSys(i.sys) }

func cloneFileInfoSys(value any) any {
	if value == nil {
		return nil
	}
	cloned := cloneFileInfoValue(reflect.ValueOf(value))
	if !cloned.IsValid() || !cloned.CanInterface() {
		return value
	}
	return cloned.Interface()
}

func cloneFileInfoValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.New(value.Type()).Elem()
		copy.Set(cloneFileInfoValue(value.Elem()))
		return copy
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.New(value.Type().Elem())
		copy.Elem().Set(cloneFileInfoValue(value.Elem()))
		return copy
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		copy.Set(value)
		for i := 0; i < value.NumField(); i++ {
			if !copy.Field(i).CanSet() || !value.Field(i).CanInterface() {
				continue
			}
			copy.Field(i).Set(cloneFileInfoValue(value.Field(i)))
		}
		return copy
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneFileInfoValue(value.Index(i)))
		}
		return copy
	case reflect.Array:
		copy := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneFileInfoValue(value.Index(i)))
		}
		return copy
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			copy.SetMapIndex(cloneFileInfoValue(iterator.Key()), cloneFileInfoValue(iterator.Value()))
		}
		return copy
	default:
		return value
	}
}

func cloneReadViewDir(entries []fs.DirEntry) []fs.DirEntry {
	return cloneReadViewDirForProducer(entries, "")
}

func cloneReadViewDirForProducer(entries []fs.DirEntry, producer string) []fs.DirEntry {
	result := make([]fs.DirEntry, len(entries))
	for i, entry := range entries {
		if frozen, ok := entry.(*readViewDirEntry); ok {
			copy := *frozen
			if producer != "" {
				copy.producer = producer
			}
			result[i] = &copy
			continue
		}
		info, err := entry.Info()
		result[i] = frozenReadViewDirEntry{entry: entry, info: frozenInfo(info), infoErr: err}
	}
	return result
}

type frozenReadViewDirEntry struct {
	entry   fs.DirEntry
	info    fs.FileInfo
	infoErr error
}

func (e frozenReadViewDirEntry) Name() string                { return e.entry.Name() }
func (e frozenReadViewDirEntry) IsDir() bool                 { return e.entry.IsDir() }
func (e frozenReadViewDirEntry) Type() fs.FileMode           { return e.entry.Type() }
func (e frozenReadViewDirEntry) Info() (fs.FileInfo, error)  { return frozenInfo(e.info), e.infoErr }
func oDirEntries(o *governanceReadObservation) []fs.DirEntry { return o.dirEntries }

func dirEntryIdentities(entries []fs.DirEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, fmt.Sprintf("%s\x00%o", entry.Name(), entry.Type()))
	}
	sort.Strings(result)
	return result
}
func sameDirIdentities(before []string, after []fs.DirEntry) bool {
	return strings.Join(before, "\x00") == strings.Join(dirEntryIdentities(after), "\x00")
}
func sameObservedFile(before, after fs.FileInfo) bool {
	if before == nil || after == nil {
		return before == nil && after == nil
	}
	metadataMatches := before.Name() == after.Name() && before.Size() == after.Size() && before.Mode() == after.Mode() && before.ModTime().Equal(after.ModTime())
	if !metadataMatches {
		return false
	}
	beforeID, afterID := fileInfoIdentityDigest(before), fileInfoIdentityDigest(after)
	if beforeID != "" && afterID != "" {
		return beforeID == afterID
	}
	return os.SameFile(before, after) || metadataMatches
}

func fileInfoIdentityDigest(info fs.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	value := reflect.ValueOf(info.Sys())
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	fields := map[string]string{}
	for _, name := range []string{"Dev", "Ino", "VolumeSerialNumber", "FileIndexHigh", "FileIndexLow", "FileID"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.CanInterface() {
			fields[name] = fmt.Sprint(field.Interface())
		}
	}
	if len(fields) == 0 {
		return ""
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return digestBytes(raw)
}

func errorDigest(errText string) string {
	if errText == "" {
		return ""
	}
	return digestBytes([]byte(errText))
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if value {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func pathWithin(root, path string) bool {
	rootAbs, rootErr := filepath.Abs(root)
	pathAbs, pathErr := filepath.Abs(path)
	if rootErr != nil || pathErr != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func cloneUnknown(source map[string]map[string]bool) map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(source))
	for producer, reasons := range source {
		result[producer] = map[string]bool{}
		for reason, value := range reasons {
			result[producer][reason] = value
		}
	}
	return result
}

func unknownReasons(source map[string]map[string]bool) map[string][]string {
	result := make(map[string][]string, len(source))
	for producer, reasons := range source {
		for reason := range reasons {
			result[producer] = append(result[producer], reason)
		}
		sort.Strings(result[producer])
	}
	return result
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
func mergeReason(reasons map[string]bool, reason string) map[string]bool {
	if reasons == nil {
		reasons = map[string]bool{}
	}
	reasons[reason] = true
	return reasons
}
func (v *governanceReadView) pathKeyForFile(item *governanceReadFileObservation) string {
	for path, candidate := range v.files {
		if candidate == item {
			return path
		}
	}
	return ""
}
