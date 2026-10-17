package pose

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// outputsFor returns a complete set whose every file carries the same tag, so a
// reader that mixes generations shows files with different tags.
func outputsFor(tag string) map[string][]byte {
	out := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		out[name] = []byte(fmt.Sprintf("{\"generation_tag\":%q,\"file\":%q}\n", tag, name))
	}
	return out
}

func tagOf(t *testing.T, raw []byte) string {
	t.Helper()
	var v struct {
		Tag string `json:"generation_tag"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unreadable index bytes %q: %v", raw, err)
	}
	return v.Tag
}

func newIndexStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Store{Root: root}
}

func publishTag(t *testing.T, s Store, tag string) IndexGenerationReceipt {
	t.Helper()
	receipt, err := s.PublishIndexGeneration(outputsFor(tag), map[string]string{"specs": "sha256:" + tag})
	if err != nil {
		t.Fatalf("publishing %s: %v", tag, err)
	}
	return receipt
}

func assertWholeSet(t *testing.T, view IndexGenerationView, tag string) {
	t.Helper()
	if view.Status != IndexGenerationCurrent {
		t.Fatalf("status %s (%s), want current", view.Status, view.Reason)
	}
	if len(view.Files) != len(IndexGenerationFiles) {
		t.Fatalf("a current set has %d files, want %d", len(view.Files), len(IndexGenerationFiles))
	}
	for name, raw := range view.Files {
		if got := tagOf(t, raw); got != tag {
			t.Fatalf("%s carries %q, want %q: the set mixes generations", name, got, tag)
		}
	}
}

func TestIndexGenerationPublishesAndReadsOneIdentifiedSet(t *testing.T) {
	s := newIndexStore(t)
	receipt := publishTag(t, s, "a")
	if !strings.HasPrefix(receipt.GenerationID, "gen-") || receipt.Previous != "" || receipt.Unchanged {
		t.Fatalf("first receipt = %+v", receipt)
	}
	view := s.ReadIndexGeneration()
	assertWholeSet(t, view, "a")
	if view.Descriptor.GenerationID != receipt.GenerationID {
		t.Fatalf("descriptor %s, receipt %s", view.Descriptor.GenerationID, receipt.GenerationID)
	}
	for _, name := range IndexGenerationFiles {
		raw, status, err := s.ReadIndex(name)
		if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "a" {
			t.Fatalf("ReadIndex(%s) = %q, %s, %v", name, raw, status, err)
		}
		// Both the legacy name and the immutable copy hold the bytes.
		legacy, _ := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", name))
		copyRaw, _ := os.ReadFile(filepath.Join(indexGenerationDir(s.Root, receipt.GenerationID), name))
		if string(legacy) != string(copyRaw) {
			t.Fatalf("%s: the legacy name and the generation copy differ", name)
		}
	}
	// The same intent again is a no-op, not a second generation.
	again := publishTag(t, s, "a")
	if !again.Unchanged || again.GenerationID != receipt.GenerationID {
		t.Fatalf("republishing the same set = %+v, want unchanged %s", again, receipt.GenerationID)
	}
}

// V1: an output that cannot be published leaves no mixed set current.
func TestIndexGenerationBlockedOutputNeverLeavesAMixedCurrentSet(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "old")
	defer func() { PublishFault = nil }()
	PublishFault = func(path string) error {
		if filepath.Base(path) == "services.json" && filepath.Base(filepath.Dir(path)) == "indexes" {
			return errors.New("services.json is blocked")
		}
		return nil
	}
	if _, err := s.PublishIndexGeneration(outputsFor("new"), map[string]string{"specs": "sha256:new"}); err == nil {
		t.Fatal("a blocked output was reported as a published generation")
	}
	PublishFault = nil

	// The descriptor was not replaced, so the previous generation is in force and
	// complete, although repo-map.json already holds the new bytes under its legacy name.
	assertWholeSet(t, s.ReadIndexGeneration(), "old")
	raw, status, err := s.ReadIndex("repo-map.json")
	if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "old" {
		t.Fatalf("repo-map.json = %q, %s, %v: a reader got the half-published name", raw, status, err)
	}

	// A clone has no generations directory: it verifies the legacy names against
	// the descriptor and finds the mixture, which it reports instead of using.
	if err := os.RemoveAll(filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)); err != nil {
		t.Fatal(err)
	}
	view := s.ReadIndexGeneration()
	if view.Status != IndexGenerationUnavailable || view.Files != nil {
		t.Fatalf("a mixed legacy set read as %s with %d files", view.Status, len(view.Files))
	}
	if _, status, err := s.ReadIndex("repo-map.json"); status != IndexGenerationUnavailable || !errors.Is(err, ErrIndexGenerationUnavailable) {
		t.Fatalf("ReadIndex over a mixed set = %s, %v", status, err)
	}
	// The retry converges.
	publishTag(t, s, "new")
	assertWholeSet(t, s.ReadIndexGeneration(), "new")
}

// V2: readers racing a publisher see one whole generation or an explicit status.
func TestIndexGenerationConcurrentReadersNeverSeeAMixture(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "t0")
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []string
	record := func(msg string) {
		mu.Lock()
		failures = append(failures, msg)
		mu.Unlock()
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				view := s.ReadIndexGeneration()
				if view.Status != IndexGenerationCurrent {
					continue
				}
				tag := ""
				for name, raw := range view.Files {
					var v struct {
						Tag string `json:"generation_tag"`
					}
					if err := json.Unmarshal(raw, &v); err != nil {
						record(name + ": " + err.Error())
						return
					}
					if tag == "" {
						tag = v.Tag
					} else if tag != v.Tag {
						record(fmt.Sprintf("mixed generation: %s carries %s, others %s", name, v.Tag, tag))
						return
					}
				}
			}
		}()
	}
	for i := 1; i <= 25; i++ {
		if _, err := s.PublishIndexGeneration(outputsFor(fmt.Sprintf("t%d", i)), map[string]string{"specs": fmt.Sprintf("sha256:t%d", i)}); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("publication %d: %v", i, err)
		}
	}
	close(stop)
	wg.Wait()
	if len(failures) > 0 {
		t.Fatal(failures[0])
	}
	assertWholeSet(t, s.ReadIndexGeneration(), "t25")
}

// V3: a stop between any two phases leaves the previous generation or the new one,
// and the same intent run again converges once.
func TestIndexGenerationRetryAfterAStopAtEveryPhaseConverges(t *testing.T) {
	phases := []string{"before-generation", "after-generation-file:repo-map.json", "after-generation-file:releases.json",
		"before-legacy", "after-legacy-file:repo-map.json", "after-legacy-file:services.json", "before-commit"}
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			s := newIndexStore(t)
			publishTag(t, s, "old")
			defer func() { IndexPublishFault = nil }()
			IndexPublishFault = func(p string) error {
				if p == phase {
					return errors.New("stopped at " + p)
				}
				return nil
			}
			if _, err := s.PublishIndexGeneration(outputsFor("new"), map[string]string{"specs": "sha256:new"}); err == nil {
				t.Fatalf("a stop at %s was reported as published", phase)
			}
			IndexPublishFault = nil
			// The descriptor is the commit: until it is replaced the old set is the one.
			assertWholeSet(t, s.ReadIndexGeneration(), "old")
			if _, err := os.Stat(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName)); err != nil {
				t.Fatal("no descriptor after a stop: the commit record must stay whole")
			}

			receipt := publishTag(t, s, "new")
			assertWholeSet(t, s.ReadIndexGeneration(), "new")
			if receipt.Previous == "" || receipt.Unchanged {
				t.Fatalf("the retry receipt = %+v", receipt)
			}
			// Exactly one directory holds the new generation, however many attempts wrote it.
			entries, _ := os.ReadDir(filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir))
			count := 0
			for _, e := range entries {
				if e.Name() == receipt.GenerationID {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("%d directories for %s", count, receipt.GenerationID)
			}
		})
	}
}

func TestIndexGenerationStopAtTheDescriptorKeepsTheOldRecordWhole(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "old")
	before, _ := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName))
	defer func() { PublishFault = nil }()
	PublishFault = func(path string) error {
		if filepath.Base(path) == indexDescriptorName {
			return errors.New("descriptor cannot be staged")
		}
		return nil
	}
	if _, err := s.PublishIndexGeneration(outputsFor("new"), map[string]string{"specs": "sha256:new"}); err == nil {
		t.Fatal("a descriptor that could not be published was reported as committed")
	}
	PublishFault = nil
	after, _ := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName))
	if string(before) != string(after) {
		t.Fatal("the descriptor changed although its publication failed")
	}
	assertWholeSet(t, s.ReadIndexGeneration(), "old")
}

// V4: what a reader cannot verify is unavailable, with its reason, never an empty set.
func TestIndexGenerationUnverifiableSetsAreUnavailableWithAReason(t *testing.T) {
	descriptorPath := func(s Store) string { return filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName) }
	mutate := func(s Store, f func(map[string]any)) {
		raw, _ := os.ReadFile(descriptorPath(s))
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			panic(err)
		}
		f(doc)
		out, _ := json.Marshal(doc)
		_ = os.WriteFile(descriptorPath(s), out, 0o644)
	}
	cases := map[string]struct {
		damage func(s Store)
		reason string
	}{
		"a file missing everywhere": {func(s Store) {
			v := s.ReadIndexGeneration()
			_ = os.Remove(filepath.Join(indexGenerationDir(s.Root, v.Descriptor.GenerationID), "roadmaps.json"))
			_ = os.Remove(filepath.Join(s.Root, ".pose", "indexes", "roadmaps.json"))
		}, "roadmaps.json"},
		"a file corrupt in both places": {func(s Store) {
			v := s.ReadIndexGeneration()
			_ = os.WriteFile(filepath.Join(indexGenerationDir(s.Root, v.Descriptor.GenerationID), "packages.json"), []byte("{}\n"), 0o644)
			_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "packages.json"), []byte("{}\n"), 0o644)
		}, "packages.json"},
		"another schema":                {func(s Store) { mutate(s, func(d map[string]any) { d["schema_version"] = 2 }) }, "schema_version"},
		"an unknown field":              {func(s Store) { mutate(s, func(d map[string]any) { d["extra"] = true }) }, "descriptor"},
		"an id that is not the content": {func(s Store) { mutate(s, func(d map[string]any) { d["generation_id"] = "gen-0000000000000000" }) }, "generation_id"},
		"a file dropped from the list": {func(s Store) {
			mutate(s, func(d map[string]any) { d["files"] = d["files"].([]any)[:6] })
		}, "files"},
		"a truncated descriptor": {func(s Store) {
			raw, _ := os.ReadFile(descriptorPath(s))
			_ = os.WriteFile(descriptorPath(s), raw[:len(raw)/2], 0o644)
		}, "descriptor"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newIndexStore(t)
			publishTag(t, s, "a")
			tc.damage(s)
			view := s.ReadIndexGeneration()
			if view.Status != IndexGenerationUnavailable || view.Files != nil || !strings.Contains(view.Reason, tc.reason) {
				t.Fatalf("view = %s %q (%d files), want unavailable naming %q", view.Status, view.Reason, len(view.Files), tc.reason)
			}
			if _, status, err := s.ReadIndex("services.json"); status == IndexGenerationCurrent && err == nil && tc.reason != "roadmaps.json" && tc.reason != "packages.json" {
				t.Fatalf("a single file of an invalid set was served as current")
			}
		})
	}
}

func TestIndexGenerationDivergentInputBindingIsStaleNotUnavailable(t *testing.T) {
	s := newIndexStore(t)
	if _, err := s.PublishIndexGeneration(outputsFor("a"), map[string]string{"specs": "sha256:one", "roadmaps": "sha256:two"}); err != nil {
		t.Fatal(err)
	}
	view := s.ReadIndexGeneration()
	stale := view.StaleInputs(func(name string) (string, bool) {
		switch name {
		case "specs":
			return "sha256:changed", true
		case "roadmaps":
			return "sha256:two", true
		}
		return "", false
	})
	if view.Status != IndexGenerationCurrent || len(stale) != 1 || stale[0] != "specs" {
		t.Fatalf("status %s, stale %v: a changed input makes the set stale, still whole", view.Status, stale)
	}
	unknown := view.StaleInputs(func(string) (string, bool) { return "", false })
	if len(unknown) != 2 {
		t.Fatalf("bindings that cannot be recomputed must be reported, got %v", unknown)
	}
}

// V5: a repository that never published a set keeps working, without a claim.
func TestIndexGenerationLegacyFilesAreReadAsBeforeWithoutAClaim(t *testing.T) {
	s := newIndexStore(t)
	for _, name := range IndexGenerationFiles {
		if err := os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", name), outputsFor("legacy")[name], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, status, err := s.ReadIndex("repo-map.json")
	if err != nil || status != IndexGenerationLegacy || tagOf(t, raw) != "legacy" {
		t.Fatalf("legacy read = %q, %s, %v", raw, status, err)
	}
	if view := s.ReadIndexGeneration(); view.Status != IndexGenerationLegacy || view.Files != nil {
		t.Fatalf("legacy view = %s with %d files: nothing here claims a set", view.Status, len(view.Files))
	}
	if _, err := s.ReplaceIndex("delivery-integrity.json", []byte("{}\n")); !errors.Is(err, ErrIndexGenerationLegacy) {
		t.Fatalf("ReplaceIndex without a generation = %v, want ErrIndexGenerationLegacy", err)
	}
	if _, _, err := s.ReadIndex("task-map.json"); err == nil {
		t.Fatal("task-map.json is not an index of a generation")
	}
	empty := newIndexStore(t)
	if _, status, err := empty.ReadIndex("repo-map.json"); !errors.Is(err, os.ErrNotExist) || status != IndexGenerationLegacy {
		t.Fatalf("a missing legacy file = %s, %v, want not-exist", status, err)
	}
}

func TestIndexGenerationReplaceIndexKeepsTheSetWhole(t *testing.T) {
	s := newIndexStore(t)
	first := publishTag(t, s, "a")
	receipt, err := s.ReplaceIndex("delivery-integrity.json", []byte("{\"generation_tag\":\"a\",\"file\":\"changed\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.GenerationID == first.GenerationID || receipt.Previous != first.GenerationID {
		t.Fatalf("receipt %+v: a replaced file is a new generation", receipt)
	}
	view := s.ReadIndexGeneration()
	if view.Status != IndexGenerationCurrent || string(view.Files["delivery-integrity.json"]) != "{\"generation_tag\":\"a\",\"file\":\"changed\"}\n" {
		t.Fatalf("replace view = %s %s", view.Status, view.Reason)
	}
	if string(view.Files["repo-map.json"]) != string(outputsFor("a")["repo-map.json"]) {
		t.Fatal("the other files changed")
	}
	// A set that does not verify is not built upon.
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "repo-map.json"), []byte("x\n"), 0o644)
	_ = os.RemoveAll(filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir))
	if _, err := s.ReplaceIndex("delivery-integrity.json", []byte("{}\n")); !errors.Is(err, ErrIndexGenerationUnavailable) {
		t.Fatalf("ReplaceIndex over a mixed set = %v", err)
	}
}

func TestIndexGenerationCompactDeliveryIndexReadsThroughTheGeneration(t *testing.T) {
	s := newIndexStore(t)
	graph := DeliveryIntegrityGraph{SchemaVersion: DeliveryIntegritySchemaVersion, InputDigest: "sha256:in", ProvenanceDigest: "sha256:prov"}
	compact, err := graph.IndexJSON()
	if err != nil {
		t.Fatal(err)
	}
	outputs := outputsFor("a")
	outputs["delivery-integrity.json"] = append(compact, '\n')
	if _, err := s.PublishIndexGeneration(outputs, nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDeliveryIntegrity("")
	if err != nil || got.InputDigest != "sha256:in" || got.ProvenanceDigest != "sha256:prov" {
		t.Fatalf("GetDeliveryIntegrity = %+v, %v", got, err)
	}
	// An unavailable set is an error, not an empty graph.
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "delivery-integrity.json"), []byte("{}\n"), 0o644)
	_ = os.RemoveAll(filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir))
	if graph, err := s.GetDeliveryIntegrity(""); !errors.Is(err, ErrIndexGenerationUnavailable) || graph.InputDigest != "" {
		t.Fatalf("GetDeliveryIntegrity over a mismatched file = %+v, %v", graph, err)
	}
}

// V6: retention is bounded and leaves the live generation alone.
func TestIndexGenerationRetentionIsBoundedAndSparesTheLiveGeneration(t *testing.T) {
	s := newIndexStore(t)
	ids := []string{}
	for i := 0; i < 6; i++ {
		ids = append(ids, publishTag(t, s, fmt.Sprintf("g%d", i)).GenerationID)
	}
	base := filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)
	// Age all but the newest two beyond the grace period, oldest first.
	for i, id := range ids {
		when := time.Now().Add(-time.Duration(len(ids)-i) * time.Hour)
		if i >= 4 {
			when = time.Now()
		}
		_ = os.Chtimes(filepath.Join(base, id), when, when)
	}
	// A generation younger than the grace period that ranks beyond the kept ones.
	young := time.Now().Add(-time.Minute)
	_ = os.Chtimes(filepath.Join(base, ids[2]), young, young)
	soon := time.Now().Add(time.Minute)
	_ = os.Chtimes(filepath.Join(base, ids[4]), soon, soon)
	_ = os.Chtimes(filepath.Join(base, ids[5]), soon, soon)
	receipt := publishTag(t, s, "g6")
	if receipt.CleanupError != nil {
		t.Fatalf("cleanup: %v", receipt.CleanupError)
	}
	entries, _ := os.ReadDir(base)
	present := map[string]bool{}
	for _, e := range entries {
		present[e.Name()] = true
	}
	if !present[receipt.GenerationID] {
		t.Fatalf("the live generation was removed: %v", present)
	}
	kept := 0
	for _, id := range ids {
		if present[id] {
			kept++
		}
	}
	if present[ids[0]] || present[ids[1]] || kept > indexGenerationKeep {
		t.Fatalf("old generations survived: %v", present)
	}
	if !present[ids[4]] || !present[ids[5]] || !present[ids[2]] {
		t.Fatalf("a kept generation, or one younger than the grace period, was removed: %v", present)
	}
}

func TestIndexGenerationRetentionNeverRemovesTheLiveGenerationNorForeignEntries(t *testing.T) {
	old := indexGenerationGrace
	indexGenerationGrace = 0
	defer func() { indexGenerationGrace = old }()
	s := newIndexStore(t)
	live := publishTag(t, s, "live").GenerationID
	base := filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)
	for i := 0; i < 5; i++ {
		publishTag(t, s, fmt.Sprintf("other%d", i))
	}
	// The live generation is the oldest on disk: ranking alone would remove it.
	ancient := time.Now().Add(-72 * time.Hour)
	_ = os.Chtimes(filepath.Join(base, live), ancient, ancient)
	current := publishTag(t, s, "current").GenerationID
	_ = os.Chtimes(filepath.Join(base, current), ancient, ancient)
	// Entries that are not generations, however old, are never touched.
	notes := filepath.Join(base, "notes")
	if err := os.MkdirAll(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(notes, ancient, ancient)
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(base, "gen-0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	// Four newer directories that look like generations outrank it.
	for i := 1; i <= 4; i++ {
		fake := filepath.Join(base, fmt.Sprintf("gen-000000000000000%d", i))
		if err := os.MkdirAll(fake, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	publishTag(t, s, "current") // same intent: nothing new is written, retention runs
	for _, name := range []string{current, "notes", "gen-0123456789abcdef"} {
		if _, err := os.Lstat(filepath.Join(base, name)); err != nil {
			t.Fatalf("retention removed %s: %v", name, err)
		}
	}
	if _, err := os.Stat(elsewhere); err != nil {
		t.Fatal("retention followed a link out of the generations directory")
	}
}

func TestIndexGenerationRefusesToOverwriteACopyOfItsOwnIdentity(t *testing.T) {
	s := newIndexStore(t)
	first := publishTag(t, s, "a")
	dir := filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir, first.GenerationID)
	_ = os.Remove(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName))
	if err := os.WriteFile(filepath.Join(dir, "releases.json"), []byte("{\"corrupt\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := s.PublishIndexGeneration(outputsFor("a"), map[string]string{"specs": "sha256:a"})
	if !errors.Is(err, ErrPublishConflict) {
		t.Fatalf("a different copy under the same identity was overwritten or ignored: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName)); statErr == nil {
		t.Fatal("a descriptor was committed over a generation that does not verify")
	}
}

func TestIndexGenerationCommitSyncsTheDescriptorAndItsDirectory(t *testing.T) {
	s := newIndexStore(t)
	defer func() { PublishSyncHook = nil }()
	synced := map[string]bool{}
	PublishSyncHook = func(kind, path string) error {
		synced[kind+":"+filepath.Base(path)] = true
		return nil
	}
	publishTag(t, s, "a")
	if !synced["file:"+indexDescriptorName] || !synced["dir:"+indexDescriptorName] {
		t.Fatalf("the commit record was not synced (file and directory): %v", synced)
	}
	if synced["file:repo-map.json"] {
		t.Fatalf("a derived index was synced like a commit record: %v", synced)
	}
}

func TestIndexGenerationReaderThatFixedAGenerationKeepsItsBytes(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "pinned")
	view := s.ReadIndexGeneration()
	for i := 0; i < 8; i++ {
		publishTag(t, s, fmt.Sprintf("later%d", i))
	}
	assertWholeSet(t, view, "pinned")
	assertWholeSet(t, s.ReadIndexGeneration(), "later7")
}

func TestIndexGenerationFailedCleanupIsNotAFailedPublication(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root removes read-only directories")
	}
	s := newIndexStore(t)
	first := publishTag(t, s, "a")
	for i := 0; i < 4; i++ {
		publishTag(t, s, fmt.Sprintf("b%d", i))
	}
	base := filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)
	old := filepath.Join(base, first.GenerationID)
	// An entry that cannot be unlinked makes the removal of this generation fail.
	locked := filepath.Join(old, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(locked, 0o755) }()
	past := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(old, past, past)

	receipt, err := s.PublishIndexGeneration(outputsFor("c"), map[string]string{"specs": "sha256:c"})
	if err != nil {
		t.Fatalf("a failed cleanup was reported as a failed publication: %v", err)
	}
	if receipt.CleanupError == nil {
		t.Fatal("the cleanup failure is not reported")
	}
	assertWholeSet(t, s.ReadIndexGeneration(), "c")
}

// R5/R6: every engine reader of a published set goes through the same check.
func TestIndexGenerationReadersRefuseAnUnavailableSet(t *testing.T) {
	s := newIndexStore(t)
	outputs := outputsFor("a")
	outputs["repo-map.json"] = []byte("{\"packages\":[{\"name\":\"api\",\"path\":\"api\"}]}\n")
	outputs["delivery-integrity.json"] = []byte("{\"reverse\":{}}\n")
	if _, err := s.PublishIndexGeneration(outputs, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.loadReviewRepoEntries(); err != nil {
		t.Fatalf("component map of a current set: %v", err)
	}
	if snap := s.CurrentObligationSnapshot(); !strings.HasPrefix(snap.IndexDigest, "sha256:") {
		t.Fatalf("index digest of a current set = %q", snap.IndexDigest)
	}
	// Another writer replaces the delivery graph behind the descriptor.
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "delivery-integrity.json"), []byte("{\"reverse\":{\"x\":[\"y\"]}}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "repo-map.json"), []byte("{\"packages\":[]}\n"), 0o644)
	_ = os.RemoveAll(filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir))

	if _, err := s.loadReviewRepoEntries(); !errors.Is(err, ErrIndexGenerationUnavailable) {
		t.Fatalf("the review plan used a component map from an unavailable set: %v", err)
	}
	snap := s.CurrentObligationSnapshot()
	if snap.IndexDigest != "unavailable" {
		t.Fatalf("the snapshot binds an index that does not verify: %q", snap.IndexDigest)
	}
	limited := false
	for _, l := range snap.Limitations {
		limited = limited || strings.Contains(l, "index set is unavailable")
	}
	if !limited {
		t.Fatalf("the snapshot does not say the set is unavailable: %v", snap.Limitations)
	}
	var plan reviewPlanContext
	s.loadObservedReviewPaths([]string{"y"}, nil, func(reviewRepoEntry, string) {}, map[string]bool{}, &plan)
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "observed paths were not used") {
		t.Fatalf("a plan built without observed paths says nothing: %v", plan.Warnings)
	}
}

func TestIndexGenerationNamesTheLegacyNamesThatDivergeFromAWholeSet(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "a")
	if view := s.ReadIndexGeneration(); len(view.Diverged) != 0 {
		t.Fatalf("a freshly published set diverges: %v", view.Diverged)
	}
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "packages.json"), []byte("[]\n"), 0o644)
	_ = os.Remove(filepath.Join(s.Root, ".pose", "indexes", "releases.json"))
	view := s.ReadIndexGeneration()
	if view.Status != IndexGenerationCurrent || len(view.Diverged) != 2 || view.Diverged[0] != "packages.json" || view.Diverged[1] != "releases.json" {
		t.Fatalf("view = %s, diverged %v", view.Status, view.Diverged)
	}
}

func TestIndexGenerationRepublishingRepairsADivergedLegacyName(t *testing.T) {
	s := newIndexStore(t)
	first := publishTag(t, s, "a")
	_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", "services.json"), []byte("[]\n"), 0o644)
	again := publishTag(t, s, "a")
	if again.Unchanged || again.GenerationID != first.GenerationID {
		t.Fatalf("receipt %+v: a diverged legacy name must be written again under the same identity", again)
	}
	if view := s.ReadIndexGeneration(); view.Status != IndexGenerationCurrent || len(view.Diverged) != 0 {
		t.Fatalf("after the repair: %s diverged %v", view.Status, view.Diverged)
	}
}

// A writer that does not take the generation lock still cannot be overwritten
// by a commit that was computed from an older descriptor.
func TestIndexGenerationConcurrentPublishersCommitOverWhatTheyRead(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "base")
	rivalDesc, err := NewIndexGenerationDescriptor(outputsFor("rival"), map[string]string{"specs": "sha256:rival"})
	if err != nil {
		t.Fatal(err)
	}
	rivalBytes, _ := rivalDesc.bytes()
	rivalOutputs := outputsFor("rival")
	for _, f := range rivalDesc.Files {
		if err := PublishFile(filepath.Join(indexGenerationDir(s.Root, rivalDesc.GenerationID), f.Name), rivalOutputs[f.Name], 0o644, PublishDerived, PublishExpectation{Mode: ExpectAbsent}); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { IndexPublishFault = nil }()
	IndexPublishFault = func(phase string) error {
		if phase == "before-commit" {
			// A writer outside the lock commits between this one's read and its commit.
			prevRaw, _ := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName))
			if err := PublishFile(filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName), rivalBytes, 0o644, PublishAuthoritative, ExpectBytes(prevRaw, true)); err != nil {
				t.Errorf("the rival commit: %v", err)
			}
		}
		return nil
	}
	_, err = s.PublishIndexGeneration(outputsFor("mine"), map[string]string{"specs": "sha256:mine"})
	IndexPublishFault = nil
	if !errors.Is(err, ErrPublishConflict) {
		t.Fatalf("a commit over a descriptor that changed was accepted: %v", err)
	}
	if got := s.ReadIndexGeneration().Descriptor.GenerationID; got != rivalDesc.GenerationID {
		t.Fatalf("the current generation is %s, want the rival's %s", got, rivalDesc.GenerationID)
	}
}

// F01: publishers that race for the lock never both accept the same predecessor.
func TestIndexGenerationPublishersNeverAcceptTheSamePredecessor(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "base")
	for round := 0; round < 20; round++ {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var done []IndexGenerationReceipt
		var busy int
		for w := 0; w < 4; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				receipt, err := s.PublishIndexGeneration(outputsFor(fmt.Sprintf("r%dw%d", round, w)), map[string]string{"specs": fmt.Sprintf("sha256:r%dw%d", round, w)})
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					done = append(done, receipt)
				case errors.Is(err, ErrIndexGenerationBusy):
					busy++
				default:
					t.Errorf("unexpected error: %v", err)
				}
			}(w)
		}
		wg.Wait()
		if len(done) == 0 {
			t.Fatalf("round %d: nobody published", round)
		}
		seen := map[string]bool{}
		for _, r := range done {
			if seen[r.Previous] {
				t.Fatalf("round %d: two publications accepted predecessor %s", round, r.Previous)
			}
			seen[r.Previous] = true
		}
		if view := s.ReadIndexGeneration(); view.Status != IndexGenerationCurrent {
			t.Fatalf("round %d: %s %s", round, view.Status, view.Reason)
		}
	}
}

func TestIndexGenerationLockIsNeverBrokenWhileItsOwnerMayBeAlive(t *testing.T) {
	s := newIndexStore(t)
	unlock, err := lockIndexGeneration(s.Root, "test.hold")
	if err != nil {
		t.Fatal(err)
	}
	var busy *IndexGenerationBusyError
	if _, err := s.PublishIndexGeneration(outputsFor("a"), nil); !errors.As(err, &busy) || busy.State != string(LockOwnerActive) {
		t.Fatalf("a live owner's lock was not reported busy: %v", err)
	}
	unlock()
	// An owner this engine cannot read is not proven dead and is not broken.
	if err := os.WriteFile(indexLockPath(s.Root), []byte("not an owner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublishIndexGeneration(outputsFor("a"), nil); !errors.Is(err, ErrIndexGenerationBusy) {
		t.Fatalf("an unreadable lock was broken: %v", err)
	}
	if _, err := os.Stat(indexLockPath(s.Root)); err != nil {
		t.Fatal("the unreadable lock was removed")
	}
}

func TestIndexGenerationLockOfADeadOwnerIsRecovered(t *testing.T) {
	s := newIndexStore(t)
	child := exec.Command(os.Args[0], "-test.run=NONE")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	owner := currentStartLockOwner(s.Root, "indexes", "index.publish")
	owner.PID = child.ProcessState.Pid()
	raw, _ := json.Marshal(owner)
	if err := os.WriteFile(indexLockPath(s.Root), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if view := inspectIndexLock(s.Root); view.State != string(LockOwnerDead) {
		t.Skipf("the platform cannot prove a process dead (%s): %s", view.State, view.Reason)
	}
	publishTag(t, s, "after-the-crash")
	assertWholeSet(t, s.ReadIndexGeneration(), "after-the-crash")
	if _, err := os.Stat(indexLockPath(s.Root)); err == nil {
		t.Fatal("the lock outlived the publication")
	}
}

// F02: a replacement is committed over the descriptor it was built from; a rival
// cannot publish between the read and the commit, and nothing is reverted.
func TestIndexGenerationReplaceIndexNeverRevertsAConcurrentGeneration(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "a")
	defer func() { IndexPublishFault = nil }()
	var rivalErr error
	IndexPublishFault = func(phase string) error {
		if phase == "replace-after-read" {
			_, rivalErr = s.PublishIndexGeneration(outputsFor("b"), map[string]string{"specs": "sha256:b"})
		}
		return nil
	}
	if _, err := s.ReplaceIndex("delivery-integrity.json", []byte("{\"generation_tag\":\"a\",\"file\":\"replaced\"}\n")); err != nil {
		t.Fatal(err)
	}
	IndexPublishFault = nil
	if !errors.Is(rivalErr, ErrIndexGenerationBusy) {
		t.Fatalf("a rival published while a replacement was being built: %v", rivalErr)
	}
	view := s.ReadIndexGeneration()
	if view.Status != IndexGenerationCurrent || tagOf(t, view.Files["repo-map.json"]) != "a" {
		t.Fatalf("view %s: the replacement lost or mixed files", view.Status)
	}
}

// F03: one generation for the whole operation.
func TestIndexGenerationPinnedStoreReadsOneGenerationEvenIfAnotherCommits(t *testing.T) {
	s := newIndexStore(t)
	publishTag(t, s, "a")
	pinned := s.PinIndexes()
	publishTag(t, s, "b")
	for _, name := range []string{"repo-map.json", "delivery-integrity.json"} {
		raw, status, err := pinned.ReadIndex(name)
		if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "a" {
			t.Fatalf("pinned %s = %q %s %v, want the pinned generation a", name, raw, status, err)
		}
		fresh, _, _ := s.ReadIndex(name)
		if tagOf(t, fresh) != "b" {
			t.Fatalf("an unpinned store did not see the new generation")
		}
	}
}

func TestIndexGenerationAReviewPlanReadsTheSetOnce(t *testing.T) {
	_, store := componentReviewFixture(t)
	outputs := outputsFor("plan")
	outputs["repo-map.json"] = []byte("{\"packages\":[]}\n")
	outputs["delivery-integrity.json"] = []byte("{\"reverse\":{}}\n")
	if _, err := store.PublishIndexGeneration(outputs, nil); err != nil {
		t.Fatal(err)
	}
	reads := 0
	defer func() { IndexReadFault = nil }()
	IndexReadFault = func(string) { reads++ }
	if _, err := store.ReviewPlan("spec:backend"); err != nil {
		t.Fatal(err)
	}
	if reads != 1 {
		t.Fatalf("a review plan fixed the generation %d times, want once: its files may come from two generations", reads)
	}
}

// F04: a first publication that did not commit is not a set.
func TestIndexGenerationInterruptedFirstPublicationReadsAsUnavailable(t *testing.T) {
	s := newIndexStore(t)
	for _, name := range IndexGenerationFiles {
		_ = os.WriteFile(filepath.Join(s.Root, ".pose", "indexes", name), outputsFor("legacy")[name], 0o644)
	}
	defer func() { IndexPublishFault = nil }()
	IndexPublishFault = func(phase string) error {
		if phase == "after-legacy-file:services.json" {
			return errors.New("stopped")
		}
		return nil
	}
	if _, err := s.PublishIndexGeneration(outputsFor("new"), map[string]string{"specs": "sha256:new"}); err == nil {
		t.Fatal("a stopped first publication was reported as published")
	}
	IndexPublishFault = nil
	if _, status, err := s.ReadIndex("repo-map.json"); status != IndexGenerationUnavailable || !errors.Is(err, ErrIndexGenerationUnavailable) {
		t.Fatalf("a mixture of two runs was served: %s %v", status, err)
	}
	if view := s.ReadIndexGeneration(); view.Status != IndexGenerationUnavailable {
		t.Fatalf("view = %s", view.Status)
	}
	if _, err := s.GetDeliveryIntegrity(""); !errors.Is(err, ErrIndexGenerationUnavailable) {
		t.Fatalf("the delivery graph consumed the mixture: %v", err)
	}
	// The retry commits and clears the marker.
	publishTag(t, s, "new")
	assertWholeSet(t, s.ReadIndexGeneration(), "new")
	if indexPending(s.Root) {
		t.Fatal("the pending marker outlived the commit")
	}
}

// F06: retention protects the generation the descriptor names now.
func TestIndexGenerationRetentionProtectsTheDescriptorsGenerationNotJustItsArgument(t *testing.T) {
	old := indexGenerationGrace
	indexGenerationGrace = 0
	defer func() { indexGenerationGrace = old }()
	s := newIndexStore(t)
	a := publishTag(t, s, "a")
	b := publishTag(t, s, "b")
	base := filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)
	for i := 1; i <= 4; i++ {
		_ = os.MkdirAll(filepath.Join(base, fmt.Sprintf("gen-000000000000000%d", i)), 0o755)
	}
	ancient := time.Now().Add(-72 * time.Hour)
	_ = os.Chtimes(filepath.Join(base, b.GenerationID), ancient, ancient)
	// A late retention step of the older publication.
	if err := s.pruneIndexGenerations(a.GenerationID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, b.GenerationID)); err != nil {
		t.Fatalf("a late retention removed the live generation: %v", err)
	}
	assertWholeSet(t, s.ReadIndexGeneration(), "b")
}

// F07: a link at the generations directory is never followed.
func TestIndexGenerationRetentionNeverFollowsALinkedGenerationsDirectory(t *testing.T) {
	old := indexGenerationGrace
	indexGenerationGrace = 0
	defer func() { indexGenerationGrace = old }()
	s := newIndexStore(t)
	outside := t.TempDir()
	victims := []string{}
	for i := 1; i <= 5; i++ {
		d := filepath.Join(outside, fmt.Sprintf("gen-000000000000000%d", i))
		_ = os.MkdirAll(d, 0o755)
		_ = os.WriteFile(filepath.Join(d, "data"), []byte("x"), 0o644)
		past := time.Now().Add(-time.Duration(i) * 24 * time.Hour)
		_ = os.Chtimes(d, past, past)
		victims = append(victims, d)
	}
	if err := os.Symlink(outside, filepath.Join(s.Root, ".pose", "indexes", indexGenerationsDir)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublishIndexGeneration(outputsFor("a"), nil); err == nil {
		t.Fatal("a generation was published through a linked generations directory")
	}
	if err := s.pruneIndexGenerations("gen-ffffffffffffffff"); err == nil {
		t.Fatal("retention accepted a linked generations directory")
	}
	for _, d := range victims {
		if _, err := os.Stat(filepath.Join(d, "data")); err != nil {
			t.Fatalf("retention removed %s outside the project", d)
		}
	}
}

// F08: nothing after the one JSON value.
func TestIndexGenerationDescriptorWithTrailingBytesIsUnavailable(t *testing.T) {
	for _, tail := range []string{"]", "}", " x", "\n{}", ",", "\x00"} {
		s := newIndexStore(t)
		publishTag(t, s, "a")
		path := filepath.Join(s.Root, ".pose", "indexes", indexDescriptorName)
		raw, _ := os.ReadFile(path)
		_ = os.WriteFile(path, append(raw, []byte(tail)...), 0o644)
		if view := s.ReadIndexGeneration(); view.Status != IndexGenerationUnavailable {
			t.Fatalf("tail %q: a descriptor with data after its value read as %s", tail, view.Status)
		}
	}
}

// A generation that is replaced and pruned while a reader reads it: the reader
// starts over on the new descriptor instead of reporting the set unavailable.
func TestIndexGenerationReaderStartsOverWhenTheDescriptorMovesUnderIt(t *testing.T) {
	for _, mode := range []string{"set", "file"} {
		t.Run(mode, func(t *testing.T) {
			s := newIndexStore(t)
			first := publishTag(t, s, "a")
			defer func() { IndexReadFault = nil }()
			moved := false
			IndexReadFault = func(phase string) {
				if moved {
					return
				}
				moved = true
				publishTag(t, s, "b")
				_ = os.RemoveAll(indexGenerationDir(s.Root, first.GenerationID))
			}
			if mode == "set" {
				assertWholeSet(t, s.ReadIndexGeneration(), "b")
				return
			}
			raw, status, err := s.ReadIndex("services.json")
			if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "b" {
				t.Fatalf("ReadIndex = %q %s %v: it did not start over on the new descriptor", raw, status, err)
			}
		})
	}
}
