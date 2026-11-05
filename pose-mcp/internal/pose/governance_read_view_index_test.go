package pose

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func governanceReadViewIndexConsumer(t *testing.T, store Store, inputs map[string]string, publish bool, legacy bool) (Store, *governanceReadView, string) {
	t.Helper()
	project := DefaultProjectID(store.Root)
	roots := NewRootsIn(NativeFiles(), RootsConfig{DefaultRoot: store.Root, DefaultProjectID: project})
	resolver := ArtifactResolver{Roots: roots, FS: NativeFiles()}
	store.FederatedProjectID = project
	store.FederatedResolver = &resolver
	if publish {
		if _, err := store.PublishIndexGeneration(outputsFor("current"), inputs); err != nil {
			t.Fatalf("publish fixture index generation: %v", err)
		}
	}
	if legacy {
		path := filepath.Join(indexDir(store.Root), "repo-map.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, outputsFor("legacy")["repo-map.json"], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base, view := store.beginGovernanceReadView()
	return base.withGovernanceReadView(view, "index-consumer").PinIndexes(), view, project
}

func governanceReadViewIndexInputs(t *testing.T, store Store) map[string]string {
	t.Helper()
	inputs := IndexInputBindingsIn(NativeFiles(), store.Root)
	for _, name := range IndexInputNames {
		if strings.TrimSpace(inputs[name]) == "" {
			t.Fatalf("fixture could not bind required index input %q", name)
		}
	}
	return inputs
}

func governanceReadViewIndexUnknown(t *testing.T, view *governanceReadView, producer string) []string {
	t.Helper()
	if view == nil || view.operation == nil {
		t.Fatal("read view has no operation")
	}
	return view.operation.audit().UnknownProducers[producer]
}

func TestGovernanceReadViewIndexConsumptionChecksBoundInputsOnlyWhenUsed(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)

	t.Run("current bindings remain current", func(t *testing.T) {
		store := obligationFixture(t)
		inputs := governanceReadViewIndexInputs(t, store)
		reader, view, _ := governanceReadViewIndexConsumer(t, store, inputs, true, false)
		raw, status, err := reader.ReadIndex("repo-map.json")
		if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "current" {
			t.Fatalf("ReadIndex = %q, %s, %v", raw, status, err)
		}
		if reasons := governanceReadViewIndexUnknown(t, view, "index-consumer"); len(reasons) != 0 {
			t.Fatalf("current bound index input set made the consumer unknown: %v", reasons)
		}
	})

	t.Run("stale binding is localized to the consumer", func(t *testing.T) {
		store := obligationFixture(t)
		inputs := governanceReadViewIndexInputs(t, store)
		if _, err := store.PublishIndexGeneration(outputsFor("current"), inputs); err != nil {
			t.Fatal(err)
		}
		reader, view, _ := governanceReadViewIndexConsumer(t, store, nil, false, false)
		path := filepath.Join(reader.Root, ".pose", "specs", "2026-10-02-consumer.md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		body = append(body, []byte("\n<!-- changed after index publication -->\n")...)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
		raw, status, err := reader.ReadIndex("repo-map.json")
		if err != nil || status != IndexGenerationCurrent || tagOf(t, raw) != "current" {
			t.Fatalf("stale inputs made the pinned index unavailable: %q, %s, %v", raw, status, err)
		}
		reasons := governanceReadViewIndexUnknown(t, view, "index-consumer")
		if len(reasons) == 0 {
			t.Fatal("a consumed index with stale source bindings left its consumer current")
		}
		if !strings.Contains(strings.Join(reasons, " "), "index input specs changed") {
			t.Fatalf("stale specs input was not disclosed: %v", reasons)
		}
		if len(governanceReadViewIndexUnknown(t, view, "projection")) != 0 {
			t.Fatalf("a stale consumed index blocked the whole projection: %v", governanceReadViewIndexUnknown(t, view, "projection"))
		}
	})

	t.Run("missing required binding is unknown", func(t *testing.T) {
		store := obligationFixture(t)
		inputs := governanceReadViewIndexInputs(t, store)
		delete(inputs, "module-discovery")
		reader, view, _ := governanceReadViewIndexConsumer(t, store, inputs, true, false)
		if _, status, err := reader.ReadIndex("repo-map.json"); err != nil || status != IndexGenerationCurrent {
			t.Fatalf("index bytes were not retained for diagnosis: status=%s err=%v", status, err)
		}
		reasons := governanceReadViewIndexUnknown(t, view, "index-consumer")
		if !strings.Contains(strings.Join(reasons, " "), "does not bind input module-discovery") {
			t.Fatalf("missing module-discovery binding was not disclosed: %v", reasons)
		}
	})

	t.Run("legacy index is unknown when consumed", func(t *testing.T) {
		reader, view, _ := governanceReadViewIndexConsumer(t, obligationFixture(t), nil, false, true)
		if _, status, err := reader.ReadIndex("repo-map.json"); err != nil || status != IndexGenerationLegacy {
			t.Fatalf("legacy bytes were not retained for diagnosis: status=%s err=%v", status, err)
		}
		if reasons := governanceReadViewIndexUnknown(t, view, "index-consumer"); len(reasons) == 0 || !strings.Contains(strings.Join(reasons, " "), "legacy index") {
			t.Fatalf("legacy index use did not make its consumer unknown: %v", reasons)
		}
	})

	t.Run("unused indexes do not poison coverage", func(t *testing.T) {
		store := obligationFixture(t)
		inputs := governanceReadViewIndexInputs(t, store)
		if _, err := store.PublishIndexGeneration(outputsFor("current"), inputs); err != nil {
			t.Fatal(err)
		}
		_, view := store.beginGovernanceReadView()
		if reasons := governanceReadViewIndexUnknown(t, view, "index-consumer"); len(reasons) != 0 {
			t.Fatalf("unused index inputs made the consumer unknown: %v", reasons)
		}
	})

	t.Run("missing legacy output remains a not-exist result", func(t *testing.T) {
		reader, view, _ := governanceReadViewIndexConsumer(t, obligationFixture(t), nil, false, false)
		_, status, err := reader.ReadIndex("repo-map.json")
		if status != IndexGenerationLegacy || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing legacy index = status %s, err %v", status, err)
		}
		if reasons := governanceReadViewIndexUnknown(t, view, "index-consumer"); len(reasons) != 0 {
			t.Fatalf("proven missing legacy index made its consumer unknown: %v", reasons)
		}
	})
}
