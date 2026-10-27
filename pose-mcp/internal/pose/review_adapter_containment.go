package pose

// The trusted-process backend bounds engine capture, not the adapter's address
// space or filesystem. Capabilities describe enforcement, never configuration.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const ReviewNotesMaxBytes = 16 << 10

// ReviewerContainment is optional schema-1 adapter policy. Pointers distinguish
// omitted finite defaults from explicit zero, which is invalid for byte limits.
type ReviewerContainment struct {
	SchemaVersion       int      `json:"schema_version"`
	Profile             string   `json:"profile"`
	Required            []string `json:"required,omitempty"`
	Environment         []string `json:"environment,omitempty"`
	Credentials         []string `json:"credentials,omitempty"`
	StreamBytes         *int64   `json:"stream_bytes,omitempty"`
	OutputBytes         *int64   `json:"output_bytes,omitempty"`
	RetainedBytes       *int64   `json:"retained_bytes,omitempty"`
	RetentionTTLSeconds int64    `json:"retention_ttl_seconds"`
	MemoryBytes         *int64   `json:"memory_bytes,omitempty"`
	CPUTimeMS           *int64   `json:"cpu_time_ms,omitempty"`
}

// UnmarshalJSON refuses misspelled enforcement fields instead of dropping them.
func (p *ReviewerContainment) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for name, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("containment field %s cannot be null; omit optional fields for defaults", name)
		}
	}
	type plain ReviewerContainment
	var decoded plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&decoded); err != nil {
		return err
	}
	*p = ReviewerContainment(decoded)
	return nil
}

// ReviewExecutionSafety is retained with the exact run and covered by its hash,
// witness and optional native signature. No environment values are exported.
type ReviewExecutionSafety struct {
	SchemaVersion   int               `json:"schema_version"`
	Profile         string            `json:"profile"`
	Platform        string            `json:"platform"`
	Capabilities    map[string]string `json:"capabilities"`
	Limitations     []string          `json:"limitations"`
	StreamBytes     int64             `json:"stream_bytes"`
	OutputBytes     int64             `json:"output_bytes"`
	RetainedBytes   int64             `json:"retained_bytes"`
	TimeoutMS       int64             `json:"timeout_ms"`
	MaxRuns         int               `json:"max_runs_per_bundle"`
	Retention       string            `json:"retention"`
	CaptureComplete bool              `json:"capture_complete"`
	RuntimeID       string            `json:"runtime_id,omitempty"`
	ExecutionCause  string            `json:"execution_cause,omitempty"`
	Operation       OperationResult   `json:"operation"`
}

var adapterEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func effectiveReviewMaxRuns(a ReviewerAdapter) int {
	if a.MaxRunsPerBundle == 0 {
		return 3
	}
	return a.MaxRunsPerBundle
}

// ResolveReviewerContainment is shared by preview, execution and consumers.
// It makes no changes, including attempt reservations.
func ResolveReviewerContainment(a ReviewerAdapter) (*ReviewExecutionSafety, error) {
	safety, err := resolveReviewerContainment(a)
	return safety, refusal(DelegationReasonContainment, err)
}

func resolveReviewerContainment(a ReviewerAdapter) (*ReviewExecutionSafety, error) {
	for _, identity := range []string{a.Vendor, a.Model} {
		if strings.TrimSpace(identity) == "" || strings.ContainsAny(identity, "\x00\r\n") {
			return nil, fmt.Errorf("adapter identity must be nonempty without surrounding whitespace")
		}
	}
	if len(a.Command) == 0 || strings.TrimSpace(a.Command[0]) == "" || strings.TrimSpace(a.Command[0]) != a.Command[0] {
		return nil, fmt.Errorf("adapter argv requires a nonempty executable")
	}
	for _, arg := range a.Command {
		if strings.ContainsRune(arg, 0) {
			return nil, fmt.Errorf("adapter argv contains NUL")
		}
	}
	if a.Executor != "" && (strings.TrimSpace(a.Executor) != a.Executor || !strings.HasPrefix(a.Executor, "agent:") || strings.TrimSpace(strings.TrimPrefix(a.Executor, "agent:")) == "") {
		return nil, fmt.Errorf("adapter executor identity is invalid")
	}
	if a.MaxRunsPerBundle < 0 || a.MaxRunsPerBundle > 100 {
		return nil, fmt.Errorf("max_runs_per_bundle must be 1..100 when present (omitted defaults to 3)")
	}
	timeout := time.Hour
	if a.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(a.Timeout)
		if err != nil || timeout < time.Millisecond || timeout > 24*time.Hour || strings.TrimSpace(a.Timeout) != a.Timeout {
			return nil, fmt.Errorf("adapter timeout must be a positive duration up to 24h")
		}
	}
	r := &ReviewExecutionSafety{SchemaVersion: 1, Profile: "legacy-trusted", Platform: runtime.GOOS + "/" + runtime.GOARCH, StreamBytes: 8 << 20, OutputBytes: 64 << 10, RetainedBytes: 16 << 20, TimeoutMS: timeout.Milliseconds(), MaxRuns: effectiveReviewMaxRuns(a), Retention: "retain-evidence; ttl=0; no automatic deletion", Capabilities: map[string]string{
		"bounded-capture": "available", "environment-allowlist": "available", "parent-termination": "available", "process-group-termination": "unsupported", "filesystem-isolation": "unavailable", "origin-isolation": "unavailable", "shared-git-isolation": "unavailable", "network-isolation": "unavailable", "credential-filesystem-isolation": "unavailable", "escaped-descendant-termination": "unavailable", "memory-limit": "unavailable", "cpu-limit": "unavailable", "automatic-expiry": "unavailable",
	}, Limitations: []string{"trusted local executable required; worktree is not a sandbox", "outside writes, transient writes, shared Git and credential files are not isolated", "no memory/CPU quota or control of descendants escaping the process group", "cleanup filesystem calls remain subject to operating-system I/O latency"}}
	if runtime.GOOS == "linux" {
		if err := reviewProcessGroupAvailable(); err == nil {
			r.Capabilities["process-group-termination"] = "available"
		} else {
			r.Capabilities["process-group-termination"] = "unavailable"
			r.Limitations = append(r.Limitations, "process inventory not observable on this host")
		}
	} else {
		r.Limitations = append(r.Limitations, "process-group termination not qualified on this platform")
	}
	p := a.Containment
	if p == nil {
		r.Limitations = append(r.Limitations, "legacy compatibility: HOME retained; other ambient credentials excluded")
		return r, nil
	}
	if p.SchemaVersion != 1 || p.Profile != "trusted-process" {
		return nil, fmt.Errorf("unsupported containment schema/profile; use schema_version 1 and trusted-process")
	}
	r.Profile = p.Profile
	for _, limit := range []struct {
		name   string
		value  *int64
		target *int64
		max    int64
	}{{"stream_bytes", p.StreamBytes, &r.StreamBytes, 64 << 20}, {"output_bytes", p.OutputBytes, &r.OutputBytes, 128 << 10}, {"retained_bytes", p.RetainedBytes, &r.RetainedBytes, 128 << 20}} {
		if limit.value != nil {
			if *limit.value <= 0 || *limit.value > limit.max {
				return nil, fmt.Errorf("%s must be 1..%d bytes", limit.name, limit.max)
			}
			*limit.target = *limit.value
		}
	}
	if r.RetainedBytes < r.StreamBytes+6*r.OutputBytes+(1<<20) {
		return nil, fmt.Errorf("retained_bytes must cover stream_bytes + 6*output_bytes + 1048576 bytes of record overhead")
	}
	if p.RetentionTTLSeconds != 0 {
		return nil, fmt.Errorf("automatic retention expiry unavailable; retention_ttl_seconds must be 0 (retain evidence)")
	}
	if p.MemoryBytes != nil || p.CPUTimeMS != nil {
		return nil, fmt.Errorf("memory/CPU enforcement unavailable in trusted-process; omit memory_bytes/cpu_time_ms")
	}
	for _, capability := range p.Required {
		state, ok := r.Capabilities[capability]
		if !ok || state != "available" {
			return nil, fmt.Errorf("required capability %q unsupported/unavailable on %s; select a qualified backend or remove that requirement", capability, r.Platform)
		}
	}
	if len(p.Environment) > 128 || len(p.Credentials) > 32 {
		return nil, fmt.Errorf("environment allowlist exceeds 128 names or 32 credential names")
	}
	seen := map[string]bool{}
	for _, name := range append(append([]string{}, p.Environment...), p.Credentials...) {
		if len(name) > 128 || !adapterEnvironmentName.MatchString(name) || seen[strings.ToUpper(name)] {
			return nil, fmt.Errorf("environment/credentials require unique valid variable names")
		}
		upper := strings.ToUpper(name)
		seen[upper] = true
		// Loader hooks can turn a trusted binary into an unrelated executable.
		if strings.HasPrefix(upper, "LD_") || strings.HasPrefix(upper, "DYLD_") || strings.HasPrefix(upper, "GIT_") || upper == "BASH_ENV" || upper == "ENV" || upper == "SHELLOPTS" || upper == "HOME" || upper == "USERPROFILE" || upper == "TMPDIR" || upper == "TMP" || upper == "TEMP" {
			return nil, fmt.Errorf("environment variable %s is reserved or unsafe", name)
		}
	}
	for _, name := range p.Environment {
		upper := strings.ToUpper(name)
		for _, mark := range []string{"TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "API_KEY", "PRIVATE_KEY"} {
			if strings.Contains(upper, mark) {
				return nil, fmt.Errorf("sensitive environment variable %s requires credentials authorization", name)
			}
		}
	}
	return r, nil
}

func validateExplicitReviewBudget(raw []byte) error {
	var policy struct {
		Adapters map[string]map[string]json.RawMessage `json:"adapters"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return err
	}
	for name, a := range policy.Adapters {
		if value, ok := a["max_runs_per_bundle"]; ok {
			var n int
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &n) != nil || n <= 0 || n > 100 {
				return fmt.Errorf("adapter %s: explicit max_runs_per_bundle must be 1..100; omit for default 3", name)
			}
		}
	}
	return nil
}

func reviewAdapterEnvironment(a ReviewerAdapter, tmp, brief, notes string) ([]string, []string, error) {
	names := []string{"PATH", "LANG", "LC_ALL", "LC_CTYPE", "SYSTEMROOT", "WINDIR"}
	if a.Containment == nil {
		names = append(names, "HOME", "USERPROFILE")
	} else {
		names = append(names, a.Containment.Environment...)
	}
	values := map[string]string{}
	put := func(name, value string) {
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		values[name] = value
	}
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			put(name, value)
		}
	}
	secrets := []string{}
	if a.Containment != nil {
		for _, name := range a.Containment.Credentials {
			if value, ok := os.LookupEnv(name); ok && value != "" {
				if len(value) < 8 || len(value) > 64<<10 {
					return nil, nil, fmt.Errorf("authorized credential value has unsupported length")
				}
				if strings.Contains(brief, value) || strings.Contains(notes, value) {
					return nil, nil, fmt.Errorf("brief/notes contain an authorized credential; refusing execution")
				}
				put(name, value)
				secrets = append(secrets, value)
			}
		}
		values["HOME"] = tmp
		values["USERPROFILE"] = tmp
	}
	values["TMPDIR"] = tmp
	values["TMP"] = tmp
	values["TEMP"] = tmp
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		env = append(env, k+"="+values[k])
	}
	// Replace overlapping credentials longest first so no residual suffix leaks.
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	return env, secrets, nil
}

// A single longest-first replacement pass handles overlapping session and
// credential values without rescanning placeholders. The writer refuses growth
// beyond the byte budget even for a one-byte session repeated across a stream.
func redactReviewCapture(raw []byte, secrets []string, session string, limit int64) ([]byte, error) {
	replacements := map[string]string{}
	for _, secret := range secrets {
		replacements[secret] = "[credential]"
	}
	if session != "" {
		if _, exists := replacements[session]; !exists {
			replacements[session] = "[private-session]"
		}
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, key, replacements[key])
	}
	w := &reviewRedactionWriter{limit: limit}
	if _, err := strings.NewReplacer(pairs...).WriteString(w, string(raw)); err != nil {
		return nil, err
	}
	return w.buf.Bytes(), nil
}

type reviewRedactionWriter struct {
	buf   bytes.Buffer
	limit int64
}

func (w *reviewRedactionWriter) Write(p []byte) (int, error) {
	if int64(w.buf.Len())+int64(len(p)) > w.limit {
		return 0, fmt.Errorf("redacted capture exceeds byte budget")
	}
	return w.buf.Write(p)
}

// Shared writer bounds concurrent stdout/stderr as one budget. Cancellation is
// triggered on the first excess byte; subsequent writes do not grow storage.
type reviewBoundedCapture struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	limit    int64
	overflow bool
	cancel   context.CancelFunc
}

func (w *reviewBoundedCapture) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	remaining := w.limit - int64(w.buf.Len())
	if int64(n) > remaining {
		w.buf.Write(p[:int(remaining)])
		w.overflow = true
		if w.cancel != nil {
			w.cancel()
		}
	} else {
		w.buf.Write(p)
	}
	return n, nil
}
func (w *reviewBoundedCapture) result() ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.buf.Bytes()...), w.overflow
}

// ReadReviewAdapterFile refuses links/special files before opening; the platform
// opener also uses nonblocking/no-follow semantics where available.
func ReadReviewAdapterFile(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > limit {
		return nil, fmt.Errorf("adapter file must be regular, non-symlink and at most %d bytes", limit)
	}
	f, err := openReviewAdapterFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("adapter file changed identity")
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("adapter file exceeds %d bytes", limit)
	}
	last, err := os.Lstat(path)
	if err != nil || !os.SameFile(after, last) || last.Size() != int64(len(raw)) {
		return nil, fmt.Errorf("adapter file changed while reading")
	}
	return raw, nil
}

func readReviewAdapterOutput(root *os.Root, limit int64) ([]byte, error) {
	return readReviewRootFile(root, "output.txt", limit)
}
func readReviewRootFile(root *os.Root, path string, limit int64) ([]byte, error) {
	before, err := root.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > limit {
		return nil, fmt.Errorf("unsafe or oversized output file")
	}
	f, err := openReviewRootFile(root, path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("output identity changed")
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("output exceeds byte budget")
	}
	last, err := root.Lstat(path)
	if err != nil || !os.SameFile(last, after) || last.Size() != int64(len(raw)) {
		return nil, fmt.Errorf("output changed while reading")
	}
	return raw, nil
}

func validateReviewNotes(notes string) error {
	if len(notes) > ReviewNotesMaxBytes || !utf8.ValidString(notes) || strings.ContainsRune(notes, 0) {
		return fmt.Errorf("review notes must be valid UTF-8, without NUL, at most %d bytes", ReviewNotesMaxBytes)
	}
	return nil
}

// reviewCleanupOwnership holds descriptors opened before the adapter starts.
// Git's recursive worktree-remove command takes a path, so it cannot safely
// delete after a hostile rename. Workspace and exact registration are instead
// removed independently through confined roots. No global prune is performed.
type reviewCleanupOwnership struct {
	removed          bool
	runtime          *os.Root
	registry         *os.Root
	registration     string
	registrationInfo os.FileInfo
	backlink         string
}

func (s Store) captureReviewCleanupOwnership(runtime *os.Root, worktree string, marker []byte) (*reviewCleanupOwnership, error) {
	raw, err := snapshotGit(s.Root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, err
	}
	parent := filepath.Join(strings.TrimSpace(string(raw)), "worktrees")
	path := strings.TrimSpace(strings.TrimPrefix(string(marker), "gitdir: "))
	if !strings.HasPrefix(string(marker), "gitdir: ") || !filepath.IsAbs(path) || filepath.Dir(path) != parent {
		return nil, fmt.Errorf("unexpected worktree registration")
	}
	registry, err := os.OpenRoot(parent)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(path)
	info, err := registry.Lstat(name)
	if err != nil || !info.IsDir() {
		registry.Close()
		return nil, fmt.Errorf("worktree registration unavailable")
	}
	o := &reviewCleanupOwnership{runtime: runtime, registry: registry, registration: name, registrationInfo: info, backlink: filepath.Join(worktree, ".git")}
	child, err := registry.OpenRoot(name)
	if err != nil {
		registry.Close()
		return nil, err
	}
	backlink, err := readReviewRootFile(child, "gitdir", 4096)
	child.Close()
	if err != nil || strings.TrimSpace(string(backlink)) != o.backlink {
		registry.Close()
		return nil, fmt.Errorf("worktree registration backlink differs")
	}
	return o, nil
}

func (s Store) cleanupReviewResources(tmp string, owned os.FileInfo, worktree string, worktreeOwned os.FileInfo, gitMarker []byte, ownership ...*reviewCleanupOwnership) error {
	var o *reviewCleanupOwnership
	if len(ownership) > 0 {
		o = ownership[0]
	}
	current, err := os.Lstat(tmp)
	if errors.Is(err, os.ErrNotExist) {
		if (o != nil && !o.removed) || (o == nil && worktreeOwned != nil) {
			return fmt.Errorf("runtime path disappeared without confirmed cleanup; resources may remain")
		}
		return nil
	}
	if err != nil || !current.IsDir() || !os.SameFile(owned, current) {
		return fmt.Errorf("cleanup refused: runtime directory identity changed")
	}

	var root *os.Root
	if o != nil {
		root = o.runtime
	} else {
		root, err = os.OpenRoot(tmp)
		if err != nil {
			return fmt.Errorf("runtime cleanup root unavailable")
		}
		defer root.Close()
	}
	stat, err := root.Stat(".")
	if err != nil || !os.SameFile(owned, stat) {
		return fmt.Errorf("runtime cleanup root identity changed")
	}
	if worktreeOwned != nil {
		info, err := root.Lstat("worktree")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("worktree cleanup unavailable")
		}
		if err == nil {
			marker, e := readReviewRootFile(root, "worktree/.git", 4096)
			if e != nil || !info.IsDir() || !os.SameFile(worktreeOwned, info) || !bytes.Equal(marker, gitMarker) {
				return fmt.Errorf("cleanup refused: worktree identity changed")
			}
		}
		if o == nil || o.registry == nil {
			return fmt.Errorf("cleanup registration ownership unavailable; resources retained")
		}
		info, e := o.registry.Lstat(o.registration)
		if e != nil || !info.IsDir() || !os.SameFile(info, o.registrationInfo) {
			return fmt.Errorf("cleanup registration identity changed")
		}
		registration, e := o.registry.OpenRoot(o.registration)
		if e != nil {
			return fmt.Errorf("cleanup registration unavailable")
		}
		defer registration.Close()
		info, e = registration.Stat(".")
		if e != nil || !os.SameFile(info, o.registrationInfo) {
			return fmt.Errorf("cleanup registration identity changed")
		}
		backlink, e := readReviewRootFile(registration, "gitdir", 4096)
		if e != nil || strings.TrimSpace(string(backlink)) != o.backlink {
			return fmt.Errorf("cleanup registration backlink changed")
		}
		for _, lock := range []string{"locked", "index.lock", "HEAD.lock"} {
			if _, e := registration.Lstat(lock); !errors.Is(e, os.ErrNotExist) {
				return fmt.Errorf("worktree registration has a live or unknown lock; resources retained")
			}
		}
		if err == nil {
			tree, e := root.OpenRoot("worktree")
			if e != nil {
				return fmt.Errorf("worktree cleanup root unavailable")
			}
			info, e := tree.Stat(".")
			if e != nil || !os.SameFile(info, worktreeOwned) {
				tree.Close()
				return fmt.Errorf("worktree cleanup identity changed")
			}
			e = removeReviewOwnedContents(tree, 100000, 256<<20)
			tree.Close()
			if e != nil {
				return e
			}
			if e := root.Remove("worktree"); e != nil && !errors.Is(e, os.ErrNotExist) {
				return fmt.Errorf("worktree cleanup incomplete")
			}
		}
		if e := removeReviewOwnedContents(registration, 1024, 256<<20); e != nil {
			return e
		}
		registration.Close()
		if e := o.registry.Remove(o.registration); e != nil && !errors.Is(e, os.ErrNotExist) {
			return fmt.Errorf("worktree registration cleanup incomplete")
		}
	} else if _, e := root.Lstat("worktree"); !errors.Is(e, os.ErrNotExist) {
		return fmt.Errorf("partial worktree setup has no verified ownership; resources retained")
	}
	if err := removeReviewOwnedContents(root, 4096, 128<<20); err != nil {
		return err
	}
	current, err = os.Lstat(tmp)
	if err != nil || !os.SameFile(owned, current) {
		return fmt.Errorf("runtime path changed during cleanup; replacement retained")
	}
	if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("runtime directory cleanup incomplete")
	}
	if o != nil {
		o.removed = true
	}
	return nil
}

func removeReviewOwnedContents(root *os.Root, maxFiles int, maxBytes int64) error {
	paths := []string{}
	var total int64
	err := fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if len(paths) >= maxFiles || strings.Count(path, "/") > 128 {
			return fmt.Errorf("cleanup inventory exceeds coverage")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
			if total > maxBytes {
				return fmt.Errorf("cleanup inventory exceeds bytes")
			}
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("runtime cleanup inventory incomplete; resources retained")
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if err := root.Remove(paths[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("runtime cleanup incomplete; resources retained")
		}
	}
	return nil
}

func reviewSafetyConflict(run ReviewRun) string {
	if run.Safety == nil {
		return ""
	} // Historical runs have no invented observations.
	if run.Safety.SchemaVersion != 1 || (run.Status == ReviewRunCompleted && !run.Safety.CaptureComplete) {
		return "adapter capture incomplete"
	}
	if run.Safety.Operation.Outcome != OutcomeCompleted {
		return "adapter cleanup incomplete"
	}
	return ""
}

func existingReviewDispatchResult(run ReviewRun) (ReviewRun, error) {
	if run.Safety != nil && run.Safety.Operation.Outcome != OutcomeCompleted {
		return run, fmt.Errorf("existing dispatch has incomplete cleanup; no new adapter was started")
	}
	return run, nil
}
