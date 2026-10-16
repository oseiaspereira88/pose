package pose

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Spec pose-writer-coordination-and-atomic-publication: writers stage in
// their own file, publish whole and only over what they read.

func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	var tmp []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			tmp = append(tmp, e.Name())
		}
	}
	return tmp
}

// V1 — concurrent writers never share staging: every call succeeds, the
// published file is one writer's whole payload, and no staging file is left.
func TestConcurrentPublishersNeverShareStaging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")
	payloads := map[string]bool{}
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		payload := fmt.Sprintf(`{"writer":%d,"pad":"%s"}`+"\n", i, strings.Repeat("x", 4096+i))
		payloads[payload] = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- PublishFile(path, []byte(payload), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("a concurrent publication failed: %v", err)
		}
	}
	got, _ := os.ReadFile(path)
	if !payloads[string(got)] {
		t.Fatalf("the published file is not one writer's whole payload: %d bytes", len(got))
	}
	if tmp := leftovers(t, dir); len(tmp) != 0 {
		t.Fatalf("staging files were left: %v", tmp)
	}
}

// V2 — a mutable record is published only over the bytes its writer read; an
// immutable one only where nothing is, and a retry of the same bytes converges.
func TestPublicationsKeepTheirPreconditions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := ExpectBytes([]byte("v1\n"), true)
	if err := os.WriteFile(path, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PublishFile(path, []byte("v2\n"), 0o644, PublishAuthoritative, read)
	var conflict *PublishConflictError
	if !errors.As(err, &conflict) || !errors.Is(err, ErrPublishConflict) || conflict.Expected != read.Digest {
		t.Fatalf("a lost update was published: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "edited\n" {
		t.Fatalf("the concurrent edit was overwritten: %q", got)
	}
	if err := PublishFile(path, []byte("v2\n"), 0o644, PublishAuthoritative, ExpectBytes([]byte("edited\n"), true)); err != nil {
		t.Fatal(err)
	}
	if err := PublishFile(filepath.Join(dir, "gone.json"), []byte("x"), 0o644, PublishAuthoritative, read); !errors.Is(err, ErrPublishConflict) {
		t.Fatalf("a record that disappeared was recreated: %v", err)
	}

	immutable := filepath.Join(dir, "sealed.json")
	if err := PublishFile(immutable, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		t.Fatal(err)
	}
	if err := PublishFile(immutable, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		t.Fatalf("a retry of the same immutable bytes did not converge: %v", err)
	}
	if err := PublishFile(immutable, []byte("other\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); !errors.Is(err, ErrPublishConflict) {
		t.Fatalf("an immutable record was overwritten: %v", err)
	}
	if got, _ := os.ReadFile(immutable); string(got) != "sealed\n" {
		t.Fatalf("the immutable record changed: %q", got)
	}
	if tmp := leftovers(t, dir); len(tmp) != 0 {
		t.Fatalf("staging files were left: %v", tmp)
	}
}

// V6 — a failure at the publication point publishes nothing, removes only
// this call's staging file and leaves another writer's in place.
func TestAFailedPublicationCleansOnlyItsOwnStaging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.md")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, ".state.md.999.tmp")
	if err := os.WriteFile(foreign, []byte("another writer"), 0o644); err != nil {
		t.Fatal(err)
	}
	PublishFault = func(string) error { return errors.New("injected") }
	t.Cleanup(func() { PublishFault = nil })
	if err := PublishFile(path, []byte("after\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAny}); err == nil {
		t.Fatal("the injected failure was not reported")
	}
	PublishFault = nil
	if got, _ := os.ReadFile(path); string(got) != "before\n" {
		t.Fatalf("a failed publication changed the file: %q", got)
	}
	if tmp := leftovers(t, dir); len(tmp) != 1 || tmp[0] != ".state.md.999.tmp" {
		t.Fatalf("cleanup touched another writer's staging or left its own: %v", tmp)
	}
	if os.Getuid() != 0 {
		readOnly := filepath.Join(dir, "ro")
		if err := os.Mkdir(readOnly, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(readOnly, 0o755) })
		if err := PublishFile(filepath.Join(readOnly, "x.json"), []byte("x"), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny}); err == nil {
			t.Fatal("a publication into a read-only directory was reported as done")
		}
	}
}

// V3 — start and transfer share the per-spec lifecycle lock: a transfer of a
// spec another lifecycle call holds is refused before it writes anything.
func TestATransferWaitsForNoLifecycleWriterOfItsSpecs(t *testing.T) {
	resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(sourceRoot, ".pose", "specs", "2026-09-24-source-task.md"))
	for _, held := range []struct{ root, slug string }{{sourceRoot, plan.Source.Slug}, {destinationRoot, plan.Destination.Slug}} {
		unlock, err := LockSpecLifecycle(held.root, held.slug, "start.apply")
		if err != nil {
			t.Fatal(err)
		}
		_, err = ApplySpecTransfer(resolver, plan, plan.Digest, map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}, nil)
		unlock()
		if err == nil || !strings.Contains(err.Error(), "concurrent-lifecycle-writer") || !strings.Contains(err.Error(), LockOwnerActive) {
			t.Fatalf("a transfer interleaved with a lifecycle writer of %s: %v", held.slug, err)
		}
	}
	if after, _ := os.ReadFile(filepath.Join(sourceRoot, ".pose", "specs", "2026-09-24-source-task.md")); !bytes.Equal(before, after) {
		t.Fatal("the refused transfer changed the source spec")
	}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}, nil); err != nil {
		t.Fatalf("the transfer did not apply once the lock was free: %v", err)
	}
}

func TestStartRefusesWhileAnotherLifecycleWriterHoldsItsSpec(t *testing.T) {
	store, _ := startFixture(t, true, "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: One.\n")
	plan, err := store.PreviewStart("work")
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := LockSpecLifecycle(store.Root, "work", "close.transition")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ApplyStart(plan, plan.Digest, nil)
	unlock()
	var held *StartLockError
	if !errors.As(err, &held) || held.Owner == nil || held.Owner.Operation != "close.transition" {
		t.Fatalf("a start interleaved with a close: %v", err)
	}
	if record, err := store.ApplyStart(plan, plan.Digest, nil); err != nil || record.Phase != "started" {
		t.Fatalf("the start after the close released: %+v %v", record, err)
	}
}

// V2 — the action policy is a read-modify-write: a write over a policy that
// changed after it was read is a conflict, and the other edit stays.
func TestTheActionPolicyIsWrittenOnlyOverWhatWasRead(t *testing.T) {
	store := actionFixture(t)
	doc, err := LoadActionPolicyDocument(store.Root)
	if err != nil {
		t.Fatal(err)
	}
	doc.GrantRole("maintainer", "human:second")
	path := actionPolicyPath(store.Root)
	other := `{"schema_version":1,"roles":{"maintainer":["human:maintainer","human:other"]}}` + "\n"
	if err := os.WriteFile(path, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := doc.Write(store.Root); !errors.Is(err, ErrPublishConflict) {
		t.Fatalf("a policy edit was overwritten: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != other {
		t.Fatalf("the policy changed: %s", got)
	}
	again, _ := LoadActionPolicyDocument(store.Root)
	again.GrantRole("maintainer", "human:second")
	if err := again.Write(store.Root); err != nil {
		t.Fatal(err)
	}
}
