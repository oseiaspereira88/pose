package pose

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func containmentPolicy() *ReviewerContainment {
	return &ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}
}
func containmentInt(n int64) *int64 { return &n }

func TestAdapterContainmentConfiguration(t *testing.T) {
	base := func() ReviewerAdapter {
		return ReviewerAdapter{Command: []string{"sh", "-c", "cat >/dev/null"}, Vendor: "fixture", Model: "fixture", Containment: containmentPolicy()}
	}
	cases := map[string]func(*ReviewerAdapter){
		"empty-command": func(a *ReviewerAdapter) { a.Command = nil }, "blank-command": func(a *ReviewerAdapter) { a.Command[0] = " " }, "NUL-argv": func(a *ReviewerAdapter) { a.Command = append(a.Command, "a\x00b") }, "blank-vendor": func(a *ReviewerAdapter) { a.Vendor = "  " }, "blank-model": func(a *ReviewerAdapter) { a.Model = " " }, "blank-executor": func(a *ReviewerAdapter) { a.Executor = " " },
		"negative-timeout": func(a *ReviewerAdapter) { a.Timeout = "-1s" }, "zero-timeout": func(a *ReviewerAdapter) { a.Timeout = "0" }, "overflow-timeout": func(a *ReviewerAdapter) { a.Timeout = "999999999999999h" }, "whitespace-timeout": func(a *ReviewerAdapter) { a.Timeout = " 1s" }, "sub-ms": func(a *ReviewerAdapter) { a.Timeout = "1ns" }, "too-long": func(a *ReviewerAdapter) { a.Timeout = "25h" }, "negative-runs": func(a *ReviewerAdapter) { a.MaxRunsPerBundle = -1 }, "too-many-runs": func(a *ReviewerAdapter) { a.MaxRunsPerBundle = 101 },
		"zero-stream": func(a *ReviewerAdapter) { a.Containment.StreamBytes = containmentInt(0) }, "negative-output": func(a *ReviewerAdapter) { a.Containment.OutputBytes = containmentInt(-1) }, "huge-stream": func(a *ReviewerAdapter) { a.Containment.StreamBytes = containmentInt(1 << 62) }, "tiny-retention": func(a *ReviewerAdapter) { a.Containment.RetainedBytes = containmentInt(1) }, "negative-ttl": func(a *ReviewerAdapter) { a.Containment.RetentionTTLSeconds = -1 }, "expiry": func(a *ReviewerAdapter) { a.Containment.RetentionTTLSeconds = 1 }, "memory": func(a *ReviewerAdapter) { a.Containment.MemoryBytes = containmentInt(0) }, "cpu": func(a *ReviewerAdapter) { a.Containment.CPUTimeMS = containmentInt(1000) },
		"wrong-profile": func(a *ReviewerAdapter) { a.Containment.Profile = "sandbox" }, "wrong-schema": func(a *ReviewerAdapter) { a.Containment.SchemaVersion = 2 }, "unknown-capability": func(a *ReviewerAdapter) { a.Containment.Required = []string{"sandboxes"} }, "loader": func(a *ReviewerAdapter) { a.Containment.Environment = []string{"LD_PRELOAD"} }, "ambient-secret": func(a *ReviewerAdapter) { a.Containment.Environment = []string{"FIXTURE_TOKEN"} }, "bad-env": func(a *ReviewerAdapter) { a.Containment.Environment = []string{"X=Y"} }, "duplicate-env": func(a *ReviewerAdapter) { a.Containment.Environment = []string{"X", "X"} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			a := base()
			change(&a)
			if _, err := ResolveReviewerContainment(a); err == nil {
				t.Fatal("invalid policy accepted")
			}
		})
	}
	a := base()
	r, err := ResolveReviewerContainment(a)
	if err != nil || r.MaxRuns != 3 || r.StreamBytes != 8<<20 || r.Retention == "" {
		t.Fatal(r, err)
	}
	for _, capability := range []string{"filesystem-isolation", "origin-isolation", "shared-git-isolation", "network-isolation", "credential-filesystem-isolation", "escaped-descendant-termination", "memory-limit", "cpu-limit", "automatic-expiry"} {
		a := base()
		a.Containment.Required = []string{capability}
		if _, err := ResolveReviewerContainment(a); err == nil {
			t.Fatal("unavailable guarantee accepted", capability)
		}
	}
	root := t.TempDir()
	s := Store{Root: root}
	for _, value := range []string{"0", "-1", "null", "999999999999999999999999", "\"3\""} {
		writeReviewFixture(t, root, ".pose/policy/reviewers.json", `{"schema_version":1,"adapters":{"fake":{"command":["sh"],"vendor":"fixture","model":"fixture","max_runs_per_bundle":`+value+`}}}`)
		if _, _, err := s.LoadReviewersPolicy(); err == nil {
			t.Fatal("explicit bad budget", value)
		}
	}
	writeReviewFixture(t, root, ".pose/policy/reviewers.json", `{"schema_version":1,"adapters":{"fake":{"command":["sh"],"vendor":"fixture","model":"fixture","containment":{"schema_version":1,"profile":"trusted-process","memroy_bytes":1}}}}`)
	if _, _, err := s.LoadReviewersPolicy(); err == nil {
		t.Fatal("unknown enforcement field dropped")
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-attempts")); !os.IsNotExist(err) {
		t.Fatal("validation produced attempts")
	}
}

func TestAdapterContainmentEnvironmentNamesCannotBypassReservedKeys(t *testing.T) {
	for _, name := range []string{"home", "Home", "userprofile", "tmp", "Temp", "tmpdir", "bash_env", "Env", "shellopts", "git_dir", "ld_preload", "dyld_library_path"} {
		a := ReviewerAdapter{Command: []string{"sh"}, Vendor: "fixture", Model: "fixture", Containment: containmentPolicy()}
		a.Containment.Environment = []string{name}
		if _, err := ResolveReviewerContainment(a); err == nil {
			t.Fatal("reserved case variant accepted", name)
		}
	}
	a := ReviewerAdapter{Command: []string{"sh"}, Vendor: "fixture", Model: "fixture", Containment: containmentPolicy()}
	a.Containment.Environment = []string{"FIXTURE_ALLOWED", "fixture_allowed"}
	if _, err := ResolveReviewerContainment(a); err == nil {
		t.Fatal("case-fold duplicate accepted")
	}
}

func TestAdapterContainmentConcurrentCapture(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &reviewBoundedCapture{limit: 1024, cancel: cancel}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = w.Write(bytes.Repeat([]byte("x"), 4096))
			}
		}()
	}
	wg.Wait()
	raw, overflow := w.result()
	if len(raw) != 1024 || !overflow || ctx.Err() != context.Canceled {
		t.Fatal(len(raw), overflow, ctx.Err())
	}
}

func TestAdapterContainmentEnvironmentAndSignedRetention(t *testing.T) {
	const secret = "synthetic-authorized-token-123456"
	t.Setenv("FIXTURE_TOKEN", secret)
	t.Setenv("UNAUTHORIZED_TOKEN", "synthetic-ambient-secret")
	t.Setenv("FIXTURE_ALLOWED", "visible")
	f := provenanceFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Containment.Environment = []string{"FIXTURE_ALLOWED"}
		a.Containment.Credentials = []string{"FIXTURE_TOKEN"}
		a.Command = []string{"sh", "-c", `cat >/dev/null; test "$FIXTURE_ALLOWED" = visible && test -z "$UNAUTHORIZED_TOKEN" && test "$HOME" = "$TMPDIR" || exit 1; printf 'decision: approved\n%s\n' "$FIXTURE_TOKEN"`}
	})
	if f.run.Status != ReviewRunCompleted || f.run.Safety == nil || f.run.Safety.Profile != "trusted-process" || f.run.Safety.Operation.Outcome != OutcomeCompleted {
		t.Fatal(f.run)
	}
	raw, err := os.ReadFile(filepath.Join(f.store.Root, filepath.FromSlash(f.run.TranscriptPath)))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secret)) || strings.Contains(f.run.Output, secret) || !strings.Contains(f.run.Output, "[credential]") {
		t.Fatal("credential leaked or not captured")
	}
	if f.run.Provenance.RawTranscriptDigest == f.run.TranscriptDigest {
		t.Fatal("raw/redacted digests were conflated")
	}
	for _, path := range []string{f.run.Path, f.run.TranscriptPath} {
		stat, err := os.Stat(filepath.Join(f.store.Root, filepath.FromSlash(path)))
		if err != nil || stat.Mode().Perm()&0077 != 0 {
			t.Fatal("artifact permissions", stat, err)
		}
	}
	if audit, err := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); err != nil || !audit.CompleteReview {
		t.Fatal(audit, err)
	}
	if _, _, err := reviewAdapterEnvironment(ReviewerAdapter{Containment: &ReviewerContainment{Credentials: []string{"FIXTURE_TOKEN"}}}, "", secret, ""); err == nil {
		t.Fatal("credential in brief accepted")
	}
	a := ReviewerAdapter{Command: []string{"sh"}, Vendor: "fixture", Model: "fixture"}
	r, err := ResolveReviewerContainment(a)
	if err != nil || r.Profile != "legacy-trusted" {
		t.Fatal(r, err)
	}
	env, _, err := reviewAdapterEnvironment(a, "/synthetic", "", "")
	if err != nil || strings.Contains(strings.Join(env, "\n"), "UNAUTHORIZED_TOKEN=") {
		t.Fatal("legacy ambient token inherited")
	}
}

func TestAdapterContainmentUnsafeAndOversizedOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix fixtures; Windows group containment not qualified")
	}
	for name, script := range map[string]string{
		"parallel-streams": `cat >/dev/null; (head -c 65536 /dev/zero >&1)& (head -c 65536 /dev/zero >&2)& wait`,
		"oversized-file":   `cat >/dev/null; head -c 65536 /dev/zero > "$1"; printf 'decision: approved\n'`,
		"output-symlink":   `cat >/dev/null; ln -s /dev/zero "$1"; printf 'decision: approved\n'`,
		"output-fifo":      `cat >/dev/null; mkfifo "$1"; printf 'decision: approved\n'`,
	} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
				a.Containment = containmentPolicy()
				a.Containment.StreamBytes = containmentInt(1024)
				a.Containment.OutputBytes = containmentInt(1024)
				a.Command = []string{"sh", "-c", script, "sh", "{output}"}
			})
			if f.run.Status != ReviewRunFailed || f.run.Safety.CaptureComplete || len(f.run.Output) > 1024 || time.Since(start) > 10*time.Second {
				t.Fatal(f.run.Status, f.run.Failure, time.Since(start))
			}
			if why := f.store.ReviewRunStillGoverns(f.run); why == "" {
				t.Fatal("incomplete capture governed review")
			}
			if len(f.run.TranscriptPath) > 0 {
				raw, err := os.ReadFile(filepath.Join(f.store.Root, filepath.FromSlash(f.run.TranscriptPath)))
				if err != nil || len(raw) > 1024 {
					t.Fatal(len(raw), err)
				}
			}
		})
	}
}

func TestAdapterContainmentTimeoutAndChildren(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process-group qualification only")
	}
	start := time.Now()
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Containment.Required = []string{"process-group-termination"}
		a.Timeout = "100ms"
		a.Command = []string{"sh", "-c", `cat >/dev/null; sleep 60 & wait`}
	})
	if f.run.Status != ReviewRunInterrupted || time.Since(start) > 8*time.Second || f.run.Safety.Operation.Outcome != OutcomeCompleted {
		t.Fatal(f.run.Status, f.run.Failure, time.Since(start), f.run.Safety)
	}
}

func TestAdapterContainmentCleanupConfinedAndIdempotent(t *testing.T) {
	tmp := t.TempDir()
	owned, err := os.Lstat(tmp)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "keep")
	if err := os.WriteFile(sentinel, []byte("live reader content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(tmp, "external")); err != nil {
		t.Skip(err)
	}
	reader, err := os.Open(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	s := Store{}
	if err := s.cleanupReviewResources(tmp, owned, "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.cleanupReviewResources(tmp, owned, "", nil, nil); err != nil {
		t.Fatal("nonidempotent", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("external data deleted", err)
	}
	// Replacing the operation root cannot authorize cleanup of its replacement.
	tmp = t.TempDir()
	owned, _ = os.Lstat(tmp)
	if err := os.Rename(tmp, tmp+"-original"); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp + "-original")
	if err := os.Symlink(outside, tmp); err != nil {
		t.Fatal(err)
	}
	if err := s.cleanupReviewResources(tmp, owned, "", nil, nil); err == nil {
		t.Fatal("replacement root accepted")
	}
}

func TestAdapterContainmentControlledOutsideWriteIsNotIsolation(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "observed-write")
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf controlled > "$1"; test -d "$(git rev-parse --git-common-dir)" || exit 1; printf 'decision: approved\n'`, "sh", outside}
	})
	if _, err := os.Stat(outside); err != nil || f.run.Status != ReviewRunCompleted {
		t.Fatal("fixture not exercised", f.run.Failure, err)
	}
	if f.run.Safety.Capabilities["filesystem-isolation"] != "unavailable" || f.run.Safety.Capabilities["shared-git-isolation"] != "unavailable" {
		t.Fatal("false isolation claim")
	}
}

func TestAdapterContainmentNotesAndArgv(t *testing.T) {
	for _, notes := range []string{strings.Repeat("x", ReviewNotesMaxBytes+1), "bad\x00note", string([]byte{255})} {
		if validateReviewNotes(notes) == nil {
			t.Fatal("invalid notes accepted")
		}
	}
	sentinel := filepath.Join(t.TempDir(), "should-not-exist")
	arg := "$(touch " + sentinel + ")"
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `cat >/dev/null; test "$1" = "$2" || exit 1; printf 'decision: approved\n'`, "sh", arg, arg}
	})
	if f.run.Status != ReviewRunCompleted {
		t.Fatal(f.run.Failure)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("argv was evaluated")
	}
	// Marshal/unmarshal preserves profile and finite limits without secret values.
	raw, _ := json.Marshal(f.run.Safety)
	if !bytes.Contains(raw, []byte(`"capabilities"`)) {
		t.Fatal(string(raw))
	}
}

func TestAdapterContainmentOverflowWithholdsCredentialPrefix(t *testing.T) {
	const secret = "synthetic-private-token-longer-than-capture"
	t.Setenv("FIXTURE_TOKEN", secret)
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Containment.Credentials = []string{"FIXTURE_TOKEN"}
		a.Containment.StreamBytes = containmentInt(10)
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf '%s\n' "$FIXTURE_TOKEN"`}
	})
	raw, err := os.ReadFile(filepath.Join(f.store.Root, f.run.TranscriptPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 0 || f.run.Output != "" || f.run.Status != ReviewRunFailed {
		t.Fatal("partial credential retained", f.run.Status, len(raw), len(f.run.Output))
	}
}

func TestAdapterContainmentCallerCancellation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux covered group only")
	}
	marker := filepath.Join(t.TempDir(), "second-run")
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `cat >/dev/null; if test -e "$1"; then : > "$1.started"; sleep 60 & wait; else printf 'decision: approved\n'; fi`, "sh", marker}
	})
	if err := os.WriteFile(marker, []byte("cancel"), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "cancellation-fixture")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := os.Stat(marker + ".started"); err == nil {
					cancel()
					return
				}
			}
		}
	}()
	started := time.Now()
	run, err := f.store.RunReviewDispatchContext(ctx, plan, time.Now)
	if err != nil || run.Status != ReviewRunInterrupted || run.Interruption == nil || run.Safety.ExecutionCause != ExecutionCauseCanceled || time.Since(started) > 8*time.Second {
		t.Fatal(run.Status, run.Failure, run.Interruption, err, time.Since(started))
	}
}

func TestAdapterContainmentSetupDoesNotRunCheckoutHook(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "second-run")
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'`}
	})
	// Repository-local hooks are ambient executable inputs, not sealed review
	// inputs. Setup must neither execute them nor capture their unbounded output.
	hook := filepath.Join(f.store.Root, ".git/hooks/post-checkout")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nprintf unexpected > '"+marker+"'\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "hook-fixture")
	if err != nil {
		t.Fatal(err)
	}
	run, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != ReviewRunCompleted {
		t.Fatal(run.Failure, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("unsealed checkout hook executed")
	}
}
