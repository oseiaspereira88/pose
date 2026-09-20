package pose

// Design delta is a bounded observation of durable structure in the immutable
// review subject. It deliberately does not decide whether the observed
// structure is good: that conclusion belongs to a review criterion and its
// human/agentic judgment. (Spec pose-abm-structural-delta.)

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	DesignDeltaSchemaVersion = 1
	DesignDeltaParserVersion = "pose-design-delta/v1"
	defaultDesignDeltaFiles  = 4096
	defaultDesignDeltaBytes  = 16 << 20
)

var designDeltaRevisionRE = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

// DesignDeltaOptions bounds all reads performed by AssessDesignDelta.
// Defaults are applied when a field is zero or negative.
type DesignDeltaOptions struct {
	MaxFiles int `json:"max_files,omitempty"`
	MaxBytes int `json:"max_bytes,omitempty"`
}

func (o DesignDeltaOptions) normalized() DesignDeltaOptions {
	if o.MaxFiles <= 0 {
		o.MaxFiles = defaultDesignDeltaFiles
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = defaultDesignDeltaBytes
	}
	return o
}

// DesignDeltaSubjectSummary identifies the subject without copying its path
// manifest into the public report.
type DesignDeltaSubjectSummary struct {
	Base                 string `json:"base,omitempty"`
	Head                 string `json:"head,omitempty"`
	PatchDigest          string `json:"patch_digest,omitempty"`
	TreeDigest           string `json:"tree_digest,omitempty"`
	ImplementationDigest string `json:"implementation_digest,omitempty"`
	Entries              int    `json:"entries"`
}

type DesignDeltaDetector struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Observed    int    `json:"observed,omitempty"`
	Unknown     int    `json:"unknown,omitempty"`
	Unsupported int    `json:"unsupported,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type DesignDeltaCoverage struct {
	FilesInSubject  int                   `json:"files_in_subject"`
	FilesObserved   int                   `json:"files_observed"`
	FilesSkipped    int                   `json:"files_skipped,omitempty"`
	BytesRead       int                   `json:"bytes_read"`
	MaxFiles        int                   `json:"max_files"`
	MaxBytes        int                   `json:"max_bytes"`
	Truncated       bool                  `json:"truncated,omitempty"`
	SymlinksSkipped int                   `json:"symlinks_skipped,omitempty"`
	Detectors       []DesignDeltaDetector `json:"detectors"`
}

// StructuralDelta is a fact observed from both sides of the sealed subject.
// Values are represented by digests so a manifest cannot leak source or a
// secret-looking registry URL into a review artifact.
type StructuralDelta struct {
	ID           string `json:"id"`
	DisplayID    string `json:"display_id"`
	Kind         string `json:"kind"`
	Action       string `json:"action"`
	State        string `json:"state"`
	Subject      string `json:"subject"`
	Path         string `json:"path,omitempty"`
	OldPath      string `json:"old_path,omitempty"`
	NewPath      string `json:"new_path,omitempty"`
	Runtime      string `json:"runtime,omitempty"`
	BeforeDigest string `json:"before_digest,omitempty"`
	AfterDigest  string `json:"after_digest,omitempty"`
}

type DesignDeltaReport struct {
	SchemaVersion int                       `json:"schema_version"`
	ParserVersion string                    `json:"parser_version"`
	Scope         string                    `json:"scope,omitempty"`
	Status        string                    `json:"status"`
	InputDigest   string                    `json:"input_digest"`
	CacheKey      string                    `json:"cache_key"`
	Subject       DesignDeltaSubjectSummary `json:"subject"`
	Coverage      DesignDeltaCoverage       `json:"coverage"`
	Deltas        []StructuralDelta         `json:"deltas"`
	Warnings      []string                  `json:"warnings,omitempty"`
}

type designDeltaDependency struct {
	Value   string
	Runtime string
}

type designDeltaDetectorState struct {
	Observed    int
	Unknown     int
	Unsupported int
	Reasons     []string
}

type designDeltaCollector struct {
	root       string
	options    DesignDeltaOptions
	coverage   DesignDeltaCoverage
	detectors  map[string]*designDeltaDetectorState
	deltas     []StructuralDelta
	deltaKeys  map[string]bool
	displayIDs map[string]string
	warnings   []string
	bytesLeft  int
}

var errDesignDeltaTooLarge = errors.New("design delta read exceeds configured byte limit")
var errDesignDeltaSymlink = errors.New("design delta refuses symlink content")

func defaultDesignDeltaDetectors() []string {
	return []string{
		"dependency-go",
		"dependency-npm",
		"component",
		"delivery-metadata",
		"governance-contract",
		"public-contract",
		"git-actions",
	}
}

// AssessDesignDelta projects the subject without writing files or invoking a
// package manager. Subject entries are already canonicalized by review bundle
// preparation, so the same implementation cannot be replaced by a working-tree
// diff between preparation and review.
func AssessDesignDelta(root string, subject ReviewBundleSubject, scope string, options DesignDeltaOptions) (DesignDeltaReport, error) {
	options = options.normalized()
	if strings.TrimSpace(root) == "" {
		return DesignDeltaReport{}, fmt.Errorf("pose: design delta root is empty")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return DesignDeltaReport{}, fmt.Errorf("pose: design delta root: %w", err)
	}
	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		return DesignDeltaReport{}, fmt.Errorf("pose: design delta root is unavailable")
	}

	summary := DesignDeltaSubjectSummary{
		Base: subject.Base, Head: subject.Head, PatchDigest: subject.PatchDigest,
		TreeDigest: subject.TreeDigest, ImplementationDigest: subject.ImplementationDigest,
		Entries: len(subject.Entries),
	}
	input, err := digestJSON(struct {
		Parser  string
		Scope   string
		Subject ReviewBundleSubject
		Options DesignDeltaOptions
	}{DesignDeltaParserVersion, scope, canonicalDesignDeltaSubject(subject), options})
	if err != nil {
		return DesignDeltaReport{}, err
	}
	// The report is a pure function of this digest: it covers the parser version,
	// the scope, the canonical subject with its per-entry digests, and the read
	// bounds. Identical inputs read the same Git blobs and produce the same
	// report, which is why the existing CacheKey field was already this value —
	// it named a cache nothing had implemented.
	//
	// One `pose check --strict` on this repository made 152 of these calls for 116
	// distinct digests: 36 repeats at about 62 ms each. Measured before caching,
	// because a cache for a ratio nobody counted is a guess.
	if cached, ok := cachedDesignDelta(input); ok {
		return cached, nil
	}
	report := DesignDeltaReport{
		SchemaVersion: DesignDeltaSchemaVersion,
		ParserVersion: DesignDeltaParserVersion,
		Scope:         scope,
		Status:        "unknown",
		InputDigest:   input,
		CacheKey:      input,
		Subject:       summary,
		Coverage: DesignDeltaCoverage{
			FilesInSubject: len(subject.Entries), MaxFiles: options.MaxFiles, MaxBytes: options.MaxBytes,
			Detectors: []DesignDeltaDetector{},
		},
		Deltas: []StructuralDelta{},
	}

	collector := &designDeltaCollector{
		root: rootAbs, options: options, coverage: report.Coverage,
		detectors: map[string]*designDeltaDetectorState{}, deltaKeys: map[string]bool{},
		displayIDs: map[string]string{}, bytesLeft: options.MaxBytes,
	}
	for _, id := range defaultDesignDeltaDetectors() {
		collector.detectors[id] = &designDeltaDetectorState{}
	}

	entries := append([]ReviewBundleSubjectEntry{}, subject.Entries...)
	sort.Slice(entries, func(i, j int) bool { return designDeltaEntryKey(entries[i]) < designDeltaEntryKey(entries[j]) })
	if len(entries) > options.MaxFiles {
		collector.coverage.Truncated = true
		collector.coverage.FilesSkipped = len(entries) - options.MaxFiles
		collector.warn(fmt.Sprintf("subject has %d entries; only the bounded first %d were assessed", len(entries), options.MaxFiles))
		entries = entries[:options.MaxFiles]
	}

	if subject.Base == "" || subject.Head == "" {
		collector.warn("subject has no immutable base/head; structural comparison is unknown")
	} else if !designDeltaRevisionRE.MatchString(subject.Base) || !designDeltaRevisionRE.MatchString(subject.Head) {
		collector.warn("subject base/head is not a safe immutable Git revision")
	} else {
		for _, entry := range entries {
			collector.observeEntry(subject.Base, subject.Head, entry)
		}
	}

	collector.coverage.Detectors = collector.detectorReports()
	report.Coverage = collector.coverage
	report.Deltas = append([]StructuralDelta{}, collector.deltas...)
	report.Warnings = uniqueSorted(collector.warnings)
	if len(report.Deltas) > 0 || collector.coverage.FilesObserved > 0 {
		report.Status = "observed"
	}
	if collector.coverage.Truncated || collector.hasUnknown() || collector.hasUnsupported() {
		if report.Status == "unknown" {
			report.Status = "unknown"
		} else {
			report.Status = "partial"
		}
	}
	if len(report.Deltas) == 0 && collector.coverage.FilesObserved == 0 && len(report.Warnings) == 0 {
		report.Warnings = []string{"subject contains no structurally assessable entries"}
	}
	storeDesignDelta(input, report)
	return report, nil
}

// designDeltaCache memoizes by input digest. Every hit and every store goes
// through a copy: the report carries three slices, and handing the cached value
// out would let one caller's edit reach the next — the aliasing class this
// repository has now found three times, in focusSurfaceGraph, in the delivery
// index filter and here.
var designDeltaCache = struct {
	sync.Mutex
	reports map[string]DesignDeltaReport
}{reports: map[string]DesignDeltaReport{}}

func cachedDesignDelta(key string) (DesignDeltaReport, bool) {
	designDeltaCache.Lock()
	defer designDeltaCache.Unlock()
	report, ok := designDeltaCache.reports[key]
	if !ok {
		return DesignDeltaReport{}, false
	}
	return copyDesignDeltaReport(report), true
}

func storeDesignDelta(key string, report DesignDeltaReport) {
	designDeltaCache.Lock()
	defer designDeltaCache.Unlock()
	designDeltaCache.reports[key] = copyDesignDeltaReport(report)
}

// copyDesignDeltaReport copies every slice the report owns.
// TestDesignDeltaCopyCoversEveryField fails when a field is added without being
// handled here.
func copyDesignDeltaReport(in DesignDeltaReport) DesignDeltaReport {
	out := in
	out.Deltas = append([]StructuralDelta{}, in.Deltas...)
	out.Warnings = append([]string{}, in.Warnings...)
	out.Coverage.Detectors = append([]DesignDeltaDetector{}, in.Coverage.Detectors...)
	return out
}

func canonicalDesignDeltaSubject(subject ReviewBundleSubject) ReviewBundleSubject {
	copy := subject
	copy.ChangeSets = nil
	// Base/head are retained in the report for audit context, but they are
	// provider refs rather than implementation identity. A derived-only commit
	// or rebase must not create a new structural projection when the sealed
	// patch/tree manifest is unchanged.
	copy.Base, copy.Head = "", ""
	copy.Entries = append([]ReviewBundleSubjectEntry{}, subject.Entries...)
	sort.Slice(copy.Entries, func(i, j int) bool {
		return designDeltaEntryKey(copy.Entries[i]) < designDeltaEntryKey(copy.Entries[j])
	})
	return copy
}

func designDeltaEntryKey(entry ReviewBundleSubjectEntry) string {
	return entry.Action + "\x00" + entry.Path + "\x00" + entry.OldPath + "\x00" + entry.NewPath + "\x00" + entry.Class + "\x00" + entry.Digest
}

func (c *designDeltaCollector) observeEntry(base, head string, entry ReviewBundleSubjectEntry) {
	path := entry.Path
	if entry.NewPath != "" {
		path = entry.NewPath
	}
	if _, err := validateArtifactPathSyntax(path); err != nil {
		c.coverage.FilesSkipped++
		c.markUnknown("git-actions", err.Error())
		return
	}
	if entry.OldPath != "" {
		if _, err := validateArtifactPathSyntax(entry.OldPath); err != nil {
			c.coverage.FilesSkipped++
			c.markUnknown("git-actions", err.Error())
			return
		}
	}
	c.markObserved("git-actions")
	// A rename/delete is itself a durable structural fact even when neither
	// side is a manifest understood by a detector. Keep it in the report rather
	// than silently reducing the subject to the declaration's supported files.
	if entry.Action == "renamed" || entry.Action == "removed" {
		c.addDelta(StructuralDelta{
			Kind: "path", Action: normalizeDesignAction(entry.Action), State: "observed",
			Subject: path, Path: path, OldPath: entry.OldPath, NewPath: entry.NewPath,
		})
	}

	if entry.Class == "submodule" {
		c.addDelta(StructuralDelta{Kind: "submodule", Action: normalizeDesignAction(entry.Action), State: "observed", Subject: path, Path: path, OldPath: entry.OldPath, NewPath: entry.NewPath})
		c.coverage.FilesObserved++
		return
	}

	basePath := entry.OldPath
	if basePath == "" {
		basePath = entry.Path
	}
	afterPath := path
	before, beforeState := c.readGitFile(base, basePath)
	after, afterState := c.readGitFile(head, afterPath)
	if beforeState == "symlink" || afterState == "symlink" {
		c.coverage.FilesSkipped++
		c.markUnknown(detectorForPath(path), "symlink content is intentionally not followed")
		return
	}
	if beforeState == "too-large" || afterState == "too-large" {
		c.coverage.FilesSkipped++
		c.coverage.Truncated = true
		c.markUnknown(detectorForPath(path), "manifest exceeds the configured byte limit")
		return
	}
	if beforeState == "invalid" || afterState == "invalid" {
		c.coverage.FilesSkipped++
		c.markUnknown(detectorForPath(path), "Git revision or path is not safely readable")
		return
	}
	c.coverage.FilesObserved++

	if kind := designMetadataKind(path); kind != "" {
		c.markObserved(detectorForKind(kind))
		c.addDelta(StructuralDelta{Kind: kind, Action: normalizeDesignAction(entry.Action), State: "observed", Subject: path, Path: path, OldPath: entry.OldPath, NewPath: entry.NewPath, BeforeDigest: optionalDigest(before, beforeState), AfterDigest: optionalDigest(after, afterState)})
		return
	}
	if kind := designPublicContractKind(path); kind != "" {
		c.markObserved("public-contract")
		c.addDelta(StructuralDelta{Kind: kind, Action: normalizeDesignAction(entry.Action), State: "observed", Subject: path, Path: path, OldPath: entry.OldPath, NewPath: entry.NewPath, BeforeDigest: optionalDigest(before, beforeState), AfterDigest: optionalDigest(after, afterState)})
		return
	}

	baseName := strings.ToLower(filepath.Base(path))
	switch baseName {
	case "go.mod":
		c.observeGoManifest(path, entry, before, beforeState, after, afterState)
	case "package.json":
		c.observeNPMManifest(path, entry, before, beforeState, after, afterState)
	case "go.sum", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml":
		kind := "dependency-lock"
		detector := "dependency-go"
		if baseName != "go.sum" {
			detector = "dependency-npm"
		}
		c.markObserved(detector)
		c.addDelta(StructuralDelta{Kind: kind, Action: normalizeDesignAction(entry.Action), State: "observed", Subject: path, Path: path, Runtime: "transitive", BeforeDigest: optionalDigest(before, beforeState), AfterDigest: optionalDigest(after, afterState)})
	case "cargo.toml", "pyproject.toml", "pom.xml", "build.gradle", "gemfile":
		c.markUnsupported("component", "manifest format is not supported by this detector")
		c.addDelta(StructuralDelta{Kind: "unsupported-manifest", Action: normalizeDesignAction(entry.Action), State: "unsupported", Subject: path, Path: path})
	default:
		// Ordinary implementation paths are deliberately not interpreted as
		// persistence/network/architecture facts. Their enclosing component is
		// still observed when a supported manifest changed.
	}
}

func (c *designDeltaCollector) observeGoManifest(path string, entry ReviewBundleSubjectEntry, before []byte, beforeState string, after []byte, afterState string) {
	c.markObserved("dependency-go")
	c.observeComponentManifest(path, entry, "go")
	beforeDeps, beforeErr := parseGoDependencies(before, beforeState == "absent")
	afterDeps, afterErr := parseGoDependencies(after, afterState == "absent")
	if beforeErr != nil || afterErr != nil {
		c.markUnknown("dependency-go", "go.mod could not be parsed deterministically")
		c.addDelta(StructuralDelta{Kind: "dependency-manifest", Action: normalizeDesignAction(entry.Action), State: "unknown", Subject: path, Path: path, BeforeDigest: optionalDigest(before, beforeState), AfterDigest: optionalDigest(after, afterState)})
		return
	}
	c.compareDependencies("dependency-go", path, beforeDeps, afterDeps)
	if len(beforeDeps) == len(afterDeps) && dependencyMapsEqual(beforeDeps, afterDeps) && beforeState != "absent" && afterState != "absent" && !bytes.Equal(before, after) {
		c.addDelta(StructuralDelta{Kind: "dependency-manifest", Action: "changed", State: "observed", Subject: path, Path: path, BeforeDigest: digestBytes(before), AfterDigest: digestBytes(after)})
	}
}

func (c *designDeltaCollector) observeNPMManifest(path string, entry ReviewBundleSubjectEntry, before []byte, beforeState string, after []byte, afterState string) {
	c.markObserved("dependency-npm")
	c.observeComponentManifest(path, entry, "npm")
	beforeDeps, beforeErr := parseNPMDependencies(before, beforeState == "absent")
	afterDeps, afterErr := parseNPMDependencies(after, afterState == "absent")
	if beforeErr != nil || afterErr != nil {
		c.markUnknown("dependency-npm", "package.json could not be parsed deterministically")
		c.addDelta(StructuralDelta{Kind: "dependency-manifest", Action: normalizeDesignAction(entry.Action), State: "unknown", Subject: path, Path: path, BeforeDigest: optionalDigest(before, beforeState), AfterDigest: optionalDigest(after, afterState)})
		return
	}
	c.compareDependencies("dependency-npm", path, beforeDeps, afterDeps)
	if len(beforeDeps) == len(afterDeps) && dependencyMapsEqual(beforeDeps, afterDeps) && beforeState != "absent" && afterState != "absent" && !bytes.Equal(before, after) {
		c.addDelta(StructuralDelta{Kind: "dependency-manifest", Action: "changed", State: "observed", Subject: path, Path: path, BeforeDigest: digestBytes(before), AfterDigest: digestBytes(after)})
	}
}

func (c *designDeltaCollector) observeComponentManifest(path string, entry ReviewBundleSubjectEntry, ecosystem string) {
	component := filepath.ToSlash(filepath.Dir(path))
	if component == "." {
		component = "root"
	}
	c.markObserved("component")
	c.addDelta(StructuralDelta{Kind: "component", Action: normalizeDesignAction(entry.Action), State: "observed", Subject: ecosystem + ":" + component, Path: path, OldPath: entry.OldPath, NewPath: entry.NewPath})
}

func (c *designDeltaCollector) compareDependencies(detector, path string, before, after map[string]designDeltaDependency) {
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	for _, key := range ordered {
		old, hadOld := before[key]
		newValue, hadNew := after[key]
		action := "changed"
		switch {
		case !hadOld:
			action = "added"
		case !hadNew:
			action = "removed"
		}
		if hadOld && hadNew && old.Value == newValue.Value && old.Runtime == newValue.Runtime {
			continue
		}
		runtime := newValue.Runtime
		if runtime == "" {
			runtime = old.Runtime
		}
		c.addDelta(StructuralDelta{Kind: "dependency", Action: action, State: "observed", Subject: detectorName(detector) + ":" + key, Path: path, Runtime: runtime, BeforeDigest: dependencyDigest(old), AfterDigest: dependencyDigest(newValue)})
	}
}

func parseGoDependencies(raw []byte, absent bool) (map[string]designDeltaDependency, error) {
	result := map[string]designDeltaDependency{}
	if absent {
		return result, nil
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty go.mod")
	}
	inRequire := false
	for _, original := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(original)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if !inRequire && strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		} else if !inRequire {
			continue
		}
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		runtime := "runtime"
		if strings.Contains(original, "// indirect") {
			runtime = "runtime-transitive"
		}
		result[fields[0]] = designDeltaDependency{Value: fields[1], Runtime: runtime}
	}
	return result, nil
}

func parseNPMDependencies(raw []byte, absent bool) (map[string]designDeltaDependency, error) {
	result := map[string]designDeltaDependency{}
	if absent {
		return result, nil
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty package.json")
	}
	var doc struct {
		Dependencies         map[string]json.RawMessage `json:"dependencies"`
		DevDependencies      map[string]json.RawMessage `json:"devDependencies"`
		OptionalDependencies map[string]json.RawMessage `json:"optionalDependencies"`
		PeerDependencies     map[string]json.RawMessage `json:"peerDependencies"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for name, value := range doc.Dependencies {
		result[name] = designDeltaDependency{Value: strings.TrimSpace(string(value)), Runtime: "runtime"}
	}
	for name, value := range doc.DevDependencies {
		result[name] = designDeltaDependency{Value: strings.TrimSpace(string(value)), Runtime: "dev"}
	}
	for name, value := range doc.OptionalDependencies {
		result[name] = designDeltaDependency{Value: strings.TrimSpace(string(value)), Runtime: "optional"}
	}
	for name, value := range doc.PeerDependencies {
		result[name] = designDeltaDependency{Value: strings.TrimSpace(string(value)), Runtime: "peer"}
	}
	return result, nil
}

func dependencyMapsEqual(a, b map[string]designDeltaDependency) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if value != b[key] {
			return false
		}
	}
	return true
}

func dependencyDigest(value designDeltaDependency) string {
	if value.Value == "" {
		return ""
	}
	return digestBytes([]byte(value.Runtime + "\x00" + value.Value))
}

func (c *designDeltaCollector) readGitFile(revision, path string) ([]byte, string) {
	if path == "" {
		return nil, "absent"
	}
	if _, err := validateArtifactPathSyntax(path); err != nil || !designDeltaRevisionRE.MatchString(revision) {
		return nil, "invalid"
	}
	joined := filepath.Join(c.root, filepath.FromSlash(path))
	if info, err := os.Lstat(joined); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, resolveErr := filepath.EvalSymlinks(joined)
		if resolveErr != nil {
			return nil, "symlink"
		}
		rel, relErr := filepath.Rel(c.root, resolved)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, "symlink"
		}
		return nil, "symlink"
	}
	exists, err := gitObjectExists(c.root, revision, path)
	if err != nil {
		return nil, "invalid"
	}
	if !exists {
		return nil, "absent"
	}
	remaining := c.bytesLeft
	if remaining <= 0 {
		return nil, "too-large"
	}
	raw, err := gitShowBounded(c.root, revision, path, remaining)
	if err != nil {
		if errors.Is(err, errDesignDeltaTooLarge) {
			c.bytesLeft = 0
			return nil, "too-large"
		}
		return nil, "invalid"
	}
	c.bytesLeft -= len(raw)
	c.coverage.BytesRead += len(raw)
	return raw, "observed"
}

func gitObjectExists(root, revision, path string) (bool, error) {
	cmd := exec.Command("git", "-C", root, "cat-file", "-e", revision+":"+path)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false, nil
	}
	return false, err
}

type designDeltaBoundedWriter struct {
	buffer bytes.Buffer
	max    int
}

func (w *designDeltaBoundedWriter) Write(p []byte) (int, error) {
	if len(p) > w.max-w.buffer.Len() {
		return 0, errDesignDeltaTooLarge
	}
	return w.buffer.Write(p)
}

func gitShowBounded(root, revision, path string, max int) ([]byte, error) {
	writer := &designDeltaBoundedWriter{max: max}
	cmd := exec.Command("git", "-C", root, "show", "--no-ext-diff", "--format=", revision+":"+path)
	cmd.Stdout, cmd.Stderr = writer, io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return writer.buffer.Bytes(), nil
}

func (c *designDeltaCollector) addDelta(delta StructuralDelta) {
	delta.Path = normalizeDesignPath(delta.Path)
	delta.OldPath = normalizeDesignPath(delta.OldPath)
	delta.NewPath = normalizeDesignPath(delta.NewPath)
	key, _ := json.Marshal(struct {
		Kind, Action, State, Subject, Path, OldPath, NewPath, Runtime, BeforeDigest, AfterDigest string
	}{delta.Kind, delta.Action, delta.State, delta.Subject, delta.Path, delta.OldPath, delta.NewPath, delta.Runtime, delta.BeforeDigest, delta.AfterDigest})
	if c.deltaKeys[string(key)] {
		return
	}
	c.deltaKeys[string(key)] = true
	sum := sha256.Sum256(key)
	full := "sd-" + hex.EncodeToString(sum[:16])
	short := "SD-" + hex.EncodeToString(sum[:4])
	if prior, ok := c.displayIDs[short]; ok && prior != full {
		c.warn("display ID collision detected for " + short)
		delta.DisplayID = "SD-" + hex.EncodeToString(sum[:6])
	} else {
		delta.DisplayID = short
	}
	c.displayIDs[short] = full
	delta.ID = full
	c.deltas = append(c.deltas, delta)
}

func (c *designDeltaCollector) detectorReports() []DesignDeltaDetector {
	result := make([]DesignDeltaDetector, 0, len(c.detectors))
	for _, id := range defaultDesignDeltaDetectors() {
		state := c.detectors[id]
		detector := DesignDeltaDetector{ID: id, Observed: state.Observed, Unknown: state.Unknown, Unsupported: state.Unsupported}
		switch {
		case state.Unknown > 0:
			detector.State = "unknown"
		case state.Observed > 0:
			detector.State = "observed"
		case state.Unsupported > 0:
			detector.State = "unsupported"
		default:
			detector.State = "not-applicable"
		}
		if len(state.Reasons) > 0 {
			detector.Reason = strings.Join(uniqueSorted(state.Reasons), "; ")
		}
		result = append(result, detector)
	}
	return result
}

func (c *designDeltaCollector) markObserved(id string) {
	if state := c.detectors[id]; state != nil {
		state.Observed++
	}
}
func (c *designDeltaCollector) markUnknown(id, reason string) {
	if state := c.detectors[id]; state != nil {
		state.Unknown++
		state.Reasons = append(state.Reasons, reason)
	}
}
func (c *designDeltaCollector) markUnsupported(id, reason string) {
	if state := c.detectors[id]; state != nil {
		state.Unsupported++
		state.Reasons = append(state.Reasons, reason)
	}
}
func (c *designDeltaCollector) warn(value string) { c.warnings = append(c.warnings, value) }
func (c *designDeltaCollector) hasUnknown() bool {
	for _, state := range c.detectors {
		if state.Unknown > 0 {
			return true
		}
	}
	return false
}
func (c *designDeltaCollector) hasUnsupported() bool {
	for _, state := range c.detectors {
		if state.Unsupported > 0 {
			return true
		}
	}
	return false
}

func normalizeDesignAction(action string) string {
	switch action {
	case "created", "added":
		return "added"
	case "removed", "deleted":
		return "removed"
	case "renamed":
		return "renamed"
	default:
		return "changed"
	}
}

func normalizeDesignPath(value string) string {
	if value == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(value))
}

func optionalDigest(value []byte, state string) string {
	if state != "observed" {
		return ""
	}
	return digestBytes(value)
}

func detectorForPath(path string) string {
	switch strings.ToLower(filepath.Base(path)) {
	case "go.mod", "go.sum":
		return "dependency-go"
	case "package.json", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml":
		return "dependency-npm"
	default:
		return "git-actions"
	}
}

func detectorForKind(kind string) string {
	switch kind {
	case "delivery-metadata":
		return "delivery-metadata"
	case "governance-contract":
		return "governance-contract"
	case "public-contract":
		return "public-contract"
	default:
		return "git-actions"
	}
}

func detectorName(detector string) string {
	if detector == "dependency-go" {
		return "go"
	}
	return "npm"
}

func designMetadataKind(path string) string {
	path = normalizeDesignPath(path)
	if path == ".pose/indexes/validation-matrix.json" || path == ".pose/indexes/module-metadata.json" || path == ".pose/policy/delivery.json" || path == ".pose/project.json" {
		return "delivery-metadata"
	}
	for _, prefix := range []string{".pose/policy/", ".pose/review-profiles/", ".pose/workflows/", ".pose/rules/", ".pose/templates/", ".pose/adr/", ".agents/skills/"} {
		if strings.HasPrefix(path, prefix) {
			return "governance-contract"
		}
	}
	return ""
}

// designContractBearingExtensions are the file shapes that carry a contract.
// The directory rules below need them because a directory name is a hint, not a
// fact: everything under a top-level `api/` is not a public contract, and a
// README that lives there is a document. Observing it as a contract change
// overstated the subject for every consumer of the report, and it would make a
// documentation edit owe a causal mapping once a profile answers for structure.
var designContractBearingExtensions = []string{".json", ".yaml", ".yml", ".proto", ".graphql", ".graphqls", ".avsc", ".thrift", ".sql", ".wsdl", ".xsd"}

func designContractBearingFile(lower string) bool {
	for _, extension := range designContractBearingExtensions {
		if strings.HasSuffix(lower, extension) {
			return true
		}
	}
	return false
}

func designPublicContractKind(path string) string {
	path = normalizeDesignPath(path)
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".proto") || strings.HasSuffix(lower, ".graphql") || strings.HasSuffix(lower, "openapi.yaml") || strings.HasSuffix(lower, "openapi.json") || lower == "pose-mcp/server.json" {
		return "public-contract"
	}
	if (strings.HasPrefix(lower, "schemas/") || strings.HasPrefix(lower, "api/")) && designContractBearingFile(lower) {
		return "public-contract"
	}
	return ""
}
