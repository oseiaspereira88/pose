package pose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// keptIndexMaterial holds, per process, the digest of the federated input
// material of an index together with the closure of reads that built it.
// Every review operation checks that the index it reads is current, and that
// check rebuilt the federated material (about 90 s of a 140 s closeout on
// harne8 with pose-dist as a dependency), so a long-running MCP server spent
// minutes on every call. A kept digest is reused only after every read of its
// closure is made again through the current operation's read view and gives
// the same bytes: the reads enter the current closure and its final audit,
// and anything the material read is covered, whatever it is (spec
// pose-index-material-is-kept-per-repository-state, ADR
// governance-reads-use-bounded-operation-scoped-views D13).
var keptIndexMaterial = struct {
	sync.Mutex
	entries map[string]keptMaterialEntry
}{entries: map[string]keptMaterialEntry{}}

// keptIndexMaterialBound bounds the kept entries of a long-running process.
const keptIndexMaterialBound = 64

type keptMaterialEntry struct {
	digest  string
	closure []materialRead
	// authorized holds each authorization decision the build consulted.
	authorized map[string]bool
}

// materialRead is one read the material's build made: a file's bytes, a
// stat, a directory listing, a link, or a Git command, with the digest of
// what it returned.
type materialRead struct {
	root   string
	kind   string
	path   string
	args   []string
	limit  int64
	digest string
}

// indexMaterialCompute, when set, replaces the build of the material; tests
// use it to count calls.
var indexMaterialCompute func(Store) (IndexFederatedMaterial, error)

const keptMaterialProducer = "index-material"

// keptIndexMaterialDigest returns the federated material digest of store.
// Inside a read-view operation it reuses a kept digest whose closure reads
// the same through that operation; otherwise it builds the material in an
// isolated operation over the same roots, whose observations are then its
// whole closure, keeps it when that operation saw nothing unknown or changed,
// and replays the closure into the current operation so its final audit
// covers every input.
func keptIndexMaterialDigest(store Store) (string, bool) {
	build := func(s Store) (string, bool) {
		run := BuildIndexFederatedMaterialIn
		if indexMaterialCompute != nil {
			run = indexMaterialCompute
		}
		material, err := run(s)
		if err != nil {
			return "", false
		}
		return IndexFederatedMaterialDigest(material.Projection)
	}
	view, _ := governanceReadViewFromFiles(store.fsys())
	if view == nil || view.operation == nil || store.FederatedResolver == nil {
		return build(store)
	}
	op := view.operation
	key, keyable := keptMaterialKey(store)
	if !keyable {
		return build(store)
	}
	keptIndexMaterial.Lock()
	entry, found := keptIndexMaterial.entries[key]
	keptIndexMaterial.Unlock()
	if found && sameAuthorization(store.FederatedResolver, entry.authorized) && replayMaterialClosure(op, store.fsys(), entry.closure) {
		return entry.digest, true
	}
	digest, closure, authorized, ok := buildIsolatedMaterial(op, store, build)
	if !ok || !sameAuthorization(store.FederatedResolver, authorized) || !replayMaterialClosure(op, store.fsys(), closure) {
		// The isolated build could not be proved, or its inputs moved before
		// the replay: the current operation answers on its own reads.
		return build(store)
	}
	keptIndexMaterial.Lock()
	if len(keptIndexMaterial.entries) >= keptIndexMaterialBound {
		keptIndexMaterial.entries = map[string]keptMaterialEntry{}
	}
	keptIndexMaterial.entries[key] = keptMaterialEntry{digest: digest, closure: closure, authorized: authorized}
	keptIndexMaterial.Unlock()
	return digest, true
}

// buildIsolatedMaterial builds the material in a fresh read-view operation
// over the current operation's base binding, and returns its digest and the
// closure of everything that operation read.
func buildIsolatedMaterial(op *governanceReadViewOperation, store Store, build func(Store) (string, bool)) (string, []materialRead, map[string]bool, bool) {
	// The isolated operation reads the requested store's root through the
	// coordinator's base binding; withGovernanceReadView binds the resolver's
	// files and roots to the isolated operation, so every read the material
	// makes, a dependency's included, is in its closure.
	base := op.baseStore
	base.Root = store.Root
	resolver, authorized := recordingAuthorization(*store.FederatedResolver)
	base.FederatedResolver, base.FederatedProjectID = &resolver, store.FederatedProjectID
	isolatedBase, isolated := base.beginGovernanceReadView()
	scoped := isolatedBase.withGovernanceReadView(isolated, keptMaterialProducer)
	scoped.FederatedProjectID = store.FederatedProjectID
	digest, known := build(scoped)
	if !known || isolated.operation == nil {
		return "", nil, nil, false
	}
	closure, ok := materialReadsSince(isolated.operation, materialReadSnapshot{})
	if !ok || len(closure) == 0 {
		return "", nil, nil, false
	}
	if audit := isolated.audit(); len(audit.ChangedPaths) > 0 || len(audit.UnknownProducers) > 0 {
		return "", nil, nil, false
	}
	return digest, closure, authorized.decisions(), true
}

// keptMaterialKey names the material's question: the engine, the project,
// the store's root, and the resolver's whole registry: its configuration, each
// project's root or lookup error, and its conflicting bindings. Those decide
// how the material resolves a reference without reading a file, so they are
// part of the question, not of the closure. Authorization decisions are
// per-call policy: they are recorded with the entry and asked again on a hit.
func keptMaterialKey(store Store) (string, bool) {
	resolver := store.FederatedResolver
	if resolver == nil {
		return "", false
	}
	parts := []string{engineVersion, store.FederatedProjectID, filepath.Clean(store.Root), fmt.Sprintf("authorize=%v", resolver.Authorize != nil)}
	if roots := resolver.Roots; roots != nil {
		context := roots.Context()
		parts = append(parts, fmt.Sprintf("default=%s configured=%v strict=%v", context.DefaultProjectID, context.DefaultConfigured, context.StrictSelection))
		for _, project := range context.ProjectIDs {
			if source, err := roots.StoreFor(project); err == nil {
				parts = append(parts, "root:"+project+"="+filepath.Clean(source.Root))
			} else {
				parts = append(parts, "error:"+project+"="+err.Error())
			}
		}
		roots.mu.RLock()
		conflicts := make([]string, 0, len(roots.conflicts))
		for project, conflict := range roots.conflicts {
			if conflict {
				conflicts = append(conflicts, "conflict:"+project)
			}
		}
		roots.mu.RUnlock()
		sort.Strings(conflicts)
		parts = append(parts, conflicts...)
	}
	return strings.Join(parts, "\x00"), true
}

// materialReadSnapshot is what an operation had observed before a build.
type materialReadSnapshot struct {
	files    map[*governanceReadView]map[string]bool
	observed map[*governanceReadView]map[string]bool
	unknown  int
	git      map[string]bool
	streams  int
}

func operationViews(op *governanceReadViewOperation) []*governanceReadView {
	op.mu.Lock()
	defer op.mu.Unlock()
	views := []*governanceReadView{op.primary}
	for _, entry := range op.roots {
		if entry.view != nil && entry.view != op.primary {
			views = append(views, entry.view)
		}
	}
	return views
}

func unknownCount(op *governanceReadViewOperation, views []*governanceReadView) int {
	count := 0
	for _, v := range views {
		v.mu.Lock()
		for _, reasons := range v.unknown {
			count += len(reasons)
		}
		v.mu.Unlock()
	}
	op.mu.Lock()
	for _, reasons := range op.unknown {
		count += len(reasons)
	}
	op.mu.Unlock()
	return count
}

func snapshotMaterialReads(op *governanceReadViewOperation) materialReadSnapshot {
	views := operationViews(op)
	snap := materialReadSnapshot{files: map[*governanceReadView]map[string]bool{}, observed: map[*governanceReadView]map[string]bool{}, git: map[string]bool{}}
	for _, v := range views {
		v.mu.Lock()
		files, observed := map[string]bool{}, map[string]bool{}
		for path := range v.files {
			files[path] = true
		}
		for key := range v.observed {
			observed[key] = true
		}
		v.mu.Unlock()
		snap.files[v], snap.observed[v] = files, observed
	}
	snap.unknown = unknownCount(op, views)
	if state := op.git; state != nil {
		state.mu.Lock()
		for key := range state.observations {
			snap.git[key] = true
		}
		snap.streams = len(state.streams)
		state.mu.Unlock()
	}
	return snap
}

// materialReadsSince is the closure a build added to the operation. A build
// that left anything unknown, or streamed a Git object it did not retain,
// has no closure that can be replayed, and is not kept.
func materialReadsSince(op *governanceReadViewOperation, before materialReadSnapshot) ([]materialRead, bool) {
	views := operationViews(op)
	if unknownCount(op, views) != before.unknown {
		return nil, false
	}
	reads := []materialRead{}
	for _, v := range views {
		v.mu.Lock()
		for path, observation := range v.files {
			if before.files[v][path] {
				continue
			}
			if observation.err != nil {
				reads = append(reads, materialRead{root: v.root, kind: "file", path: path, digest: "error\x00" + observation.errText})
				continue
			}
			sum := sha256.Sum256(observation.bytes)
			reads = append(reads, materialRead{root: v.root, kind: "file", path: path, digest: hex.EncodeToString(sum[:])})
		}
		for key, observation := range v.observed {
			if before.observed[v][key] {
				continue
			}
			reads = append(reads, materialRead{root: v.root, kind: observation.method, path: observation.path, digest: observationDigest(observation.method, observation.info, observation.entries, observation.value, observation.errText)})
		}
		v.mu.Unlock()
	}
	if state := op.git; state != nil {
		state.mu.Lock()
		defer state.mu.Unlock()
		if len(state.streams) != before.streams {
			return nil, false
		}
		for key, observation := range state.observations {
			if before.git[key] {
				continue
			}
			reads = append(reads, materialRead{root: observation.root, kind: "git", args: append([]string(nil), observation.args...), limit: observation.maxOutputBytes, digest: gitReadDigest(observation.stdout, observation.stderr, observation.errText)})
		}
	}
	return reads, true
}

// replayMaterialClosure makes every read of a kept closure again through the
// current operation, which records it for the final audit, and reports
// whether each returned what the build read.
func replayMaterialClosure(op *governanceReadViewOperation, files Files, closure []materialRead) bool {
	for _, read := range closure {
		if read.kind == "git" {
			out, err := op.captureFilesBoundedContext(files, context.Background(), read.root, keptMaterialProducer, read.limit, read.args...)
			if gitReadDigest(out, nil, errorText(err)) != read.digest {
				return false
			}
			continue
		}
		// A read is routed as the build's was: the operation opens the view of
		// an authorized root on its first read, and the read must land in the
		// root it was made in.
		v := op.route(read.path, keptMaterialProducer, "")
		if v == nil || filepath.Clean(v.root) != filepath.Clean(read.root) {
			return false
		}
		var digest string
		switch read.kind {
		case "file":
			raw, err := v.readFile(read.path, keptMaterialProducer)
			if err != nil {
				digest = "error\x00" + errorText(err)
			} else {
				sum := sha256.Sum256(raw)
				digest = hex.EncodeToString(sum[:])
			}
		case "stat", "lstat":
			info, err := v.statInfo(read.path, keptMaterialProducer, read.kind == "lstat")
			digest = observationDigest(read.kind, info, nil, "", errorText(err))
		case "directory":
			entries, err := v.readDir(read.path, keptMaterialProducer)
			digest = observationDigest(read.kind, nil, dirEntryIdentities(entries), "", errorText(err))
		case "direntry-info":
			digest = replayDirEntryInfo(v, read.path)
		case "readlink", "eval-symlinks", "eval-parent-symlinks", "root-eval-symlinks":
			readString := v.base.EvalSymlinks
			if read.kind == "readlink" {
				readString = v.base.Readlink
			}
			v.mu.Lock()
			value, err := v.readStringLocked(read.kind, read.path, read.path, keptMaterialProducer, readString)
			v.mu.Unlock()
			digest = observationDigest(read.kind, nil, nil, value, errorText(err))
		default:
			return false
		}
		if digest != read.digest {
			return false
		}
	}
	return true
}

// observationDigest names what a metadata read returned: mode, size and
// modification time of a stat, the names of a listing, the value of a link,
// or the error.
func observationDigest(method string, info fs.FileInfo, entries []string, value, errText string) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s\x00%s\x00", method, errText)
	if info != nil {
		fmt.Fprintf(hash, "%v\x00%d\x00%d\x00", info.Mode(), info.Size(), info.ModTime().UnixNano())
	}
	for _, name := range entries {
		fmt.Fprintf(hash, "%s\x00", name)
	}
	fmt.Fprintf(hash, "%s\x00", value)
	return hex.EncodeToString(hash.Sum(nil))
}

func gitReadDigest(stdout, stderr []byte, errText string) string {
	hash := sha256.New()
	hash.Write(stdout)
	hash.Write([]byte{0})
	hash.Write(stderr)
	hash.Write([]byte{0})
	hash.Write([]byte(errText))
	return hex.EncodeToString(hash.Sum(nil))
}

// replayDirEntryInfo reads the information of one entry of a directory as
// the build read it, through the directory listing.
func replayDirEntryInfo(v *governanceReadView, path string) string {
	entries, err := v.readDir(filepath.Dir(path), keptMaterialProducer)
	if err != nil {
		return observationDigest("direntry-info", nil, nil, "", errorText(err))
	}
	for _, entry := range entries {
		if entry.Name() == filepath.Base(path) {
			info, infoErr := v.readDirEntryInfo(filepath.Dir(path), entry, keptMaterialProducer)
			return observationDigest("direntry-info", info, nil, "", errorText(infoErr))
		}
	}
	return observationDigest("direntry-info", nil, nil, "", "missing")
}

// authorizationRecord collects the decisions of a resolver's authorization
// function during one build.
type authorizationRecord struct {
	mu        sync.Mutex
	byProject map[string]bool
}

func (r *authorizationRecord) decisions() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]bool, len(r.byProject))
	for project, allowed := range r.byProject {
		out[project] = allowed
	}
	return out
}

// recordingAuthorization returns a copy of resolver whose authorization
// function records every decision it makes.
func recordingAuthorization(resolver ArtifactResolver) (ArtifactResolver, *authorizationRecord) {
	record := &authorizationRecord{byProject: map[string]bool{}}
	if authorize := resolver.Authorize; authorize != nil {
		resolver.Authorize = func(project string) bool {
			allowed := authorize(project)
			record.mu.Lock()
			record.byProject[project] = allowed
			record.mu.Unlock()
			return allowed
		}
	}
	return resolver, record
}

// sameAuthorization asks the current resolver again for each decision a kept
// build consulted; the material is reused only when every answer is the same.
func sameAuthorization(resolver *ArtifactResolver, decisions map[string]bool) bool {
	if len(decisions) == 0 {
		return true
	}
	if resolver == nil || resolver.Authorize == nil {
		return false
	}
	projects := make([]string, 0, len(decisions))
	for project := range decisions {
		projects = append(projects, project)
	}
	sort.Strings(projects)
	for _, project := range projects {
		if resolver.Authorize(project) != decisions[project] {
			return false
		}
	}
	return true
}
