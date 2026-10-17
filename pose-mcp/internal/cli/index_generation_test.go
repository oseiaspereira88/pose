package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func runIndex(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cmdIndex(root, args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func indexBytes(t *testing.T, root, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestIndexPublishesOneIdentifiedGenerationAndRepeatsIt(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index code=%d err=%s", code, errOut)
	}
	store := posemodel.Store{Root: root}
	view := store.ReadIndexGeneration()
	if view.Status != posemodel.IndexGenerationCurrent || len(view.Files) != len(posemodel.IndexGenerationFiles) {
		t.Fatalf("after pose index: %s %s", view.Status, view.Reason)
	}
	before := map[string]string{}
	for _, name := range posemodel.IndexGenerationFiles {
		before[name] = string(indexBytes(t, root, name))
	}
	descriptor := string(indexBytes(t, root, "generation.json"))

	// Same inputs: same generation, byte for byte (provenance and digests equal).
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("second index code=%d err=%s", code, errOut)
	}
	if string(indexBytes(t, root, "generation.json")) != descriptor {
		t.Fatal("pose index over unchanged inputs named another generation")
	}
	for _, name := range posemodel.IndexGenerationFiles {
		if string(indexBytes(t, root, name)) != before[name] {
			t.Fatalf("%s changed although the inputs did not", name)
		}
	}
}

func TestIndexJSONReportsTheGeneration(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	code, out, errOut := runIndex(t, root, "--json")
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut)
	}
	if !strings.Contains(out, `"index_generation": "gen-`) && !strings.Contains(out, `"index_generation":"gen-`) {
		t.Fatalf("the machine document does not name the generation:\n%s", out)
	}
}

func TestIndexVerifyNamesEveryStateAndPassesOnlyForACurrentSet(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	// Before the first pose index there is no set to verify.
	_ = os.Remove(filepath.Join(root, ".pose", "indexes", "generation.json"))
	if code, _, errOut := runIndex(t, root, "--verify"); code == 0 || !strings.Contains(errOut, "not published as a set") {
		t.Fatalf("verify without a generation: code=%d err=%s", code, errOut)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	if code, out, errOut := runIndex(t, root, "--verify"); code != 0 || !strings.Contains(out, "is current") {
		t.Fatalf("verify of a current set: code=%d out=%s err=%s", code, out, errOut)
	}

	// An input moved on: the set is whole but stale.
	writeCloseoutCLIFile(t, root, ".pose/specs/added-after-index.md", "---\nslug: added-after-index\nstatus: draft\ncreated_at: 2026-10-17\n---\n# Spec\n")
	if code, _, errOut := runIndex(t, root, "--verify"); code == 0 || !strings.Contains(errOut, "specs") || !strings.Contains(errOut, "stale") && !strings.Contains(errOut, "other inputs") {
		t.Fatalf("verify after a spec was added: code=%d err=%s", code, errOut)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("reindex: %d %s", code, errOut)
	}
	if code, _, errOut := runIndex(t, root, "--verify"); code != 0 {
		t.Fatalf("verify after reindex: code=%d err=%s", code, errOut)
	}

	// A file edited by hand next to the descriptor is not a set.
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "repo-map.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(root, ".pose", "indexes", "generations"))
	if code, _, errOut := runIndex(t, root, "--verify"); code == 0 || !strings.Contains(errOut, "unavailable") {
		t.Fatalf("verify of a hand-edited set: code=%d err=%s", code, errOut)
	}
	if code, _, errOut := runIndex(t, root, "--verify", "--bogus"); code != 2 {
		t.Fatalf("unknown flag: code=%d err=%s", code, errOut)
	}
}

func TestIndexWithABlockedOutputKeepsThePreviousGenerationInForce(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	previous := string(indexBytes(t, root, "generation.json"))
	writeCloseoutCLIFile(t, root, ".pose/specs/changes-the-graph.md", "---\nslug: changes-the-graph\nstatus: draft\ncreated_at: 2026-10-17\n---\n# Spec\n")
	defer func() { posemodel.PublishFault = nil }()
	posemodel.PublishFault = func(path string) error {
		if filepath.Base(path) == "packages.json" && filepath.Base(filepath.Dir(path)) == "indexes" {
			return errors.New("packages.json is blocked")
		}
		return nil
	}
	code, _, errOut := runIndex(t, root)
	posemodel.PublishFault = nil
	if code == 0 || !strings.Contains(errOut, "previous generation stays in force") {
		t.Fatalf("a blocked output: code=%d err=%s", code, errOut)
	}
	if string(indexBytes(t, root, "generation.json")) != previous {
		t.Fatal("the descriptor moved although the publication failed")
	}
	store := posemodel.Store{Root: root}
	if view := store.ReadIndexGeneration(); view.Status != posemodel.IndexGenerationCurrent {
		t.Fatalf("the previous generation is no longer whole: %s %s", view.Status, view.Reason)
	}
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("retry: %d %s", code, errOut)
	}
	if string(indexBytes(t, root, "generation.json")) == previous {
		t.Fatal("the retry did not publish the new generation")
	}
}

func TestRefreshDeliveryIndexKeepsTheDescriptorTruthful(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	before := posemodel.Store{Root: root}.ReadIndexGeneration().Descriptor.GenerationID
	var errOut bytes.Buffer
	refreshDeliveryIndex(root, []byte("{\"schema_version\":1}\n"), &errOut)
	view := posemodel.Store{Root: root}.ReadIndexGeneration()
	if view.Status != posemodel.IndexGenerationCurrent || view.Descriptor.GenerationID == before {
		t.Fatalf("after a refresh: %s %s (id %s, before %s): a replaced file is a new, whole generation", view.Status, view.Reason, view.Descriptor.GenerationID, before)
	}
	if errOut.Len() != 0 {
		t.Fatalf("a refresh over a current set said: %s", errOut.String())
	}

	// Over a set that does not verify it does not write, and says so.
	_ = os.WriteFile(filepath.Join(root, ".pose", "indexes", "services.json"), []byte("[1]\n"), 0o644)
	_ = os.RemoveAll(filepath.Join(root, ".pose", "indexes", "generations"))
	errOut.Reset()
	refreshDeliveryIndex(root, []byte("{\"schema_version\":1,\"x\":1}\n"), &errOut)
	if !strings.Contains(errOut.String(), "not refreshed") {
		t.Fatalf("a refresh over an unavailable set was silent: %q", errOut.String())
	}

	// Before the first set, the file is written as it always was.
	legacy, _, _ := artifactGitFixture(t)
	_ = os.Remove(filepath.Join(legacy, ".pose", "indexes", "generation.json"))
	refreshDeliveryIndex(legacy, []byte("{\"schema_version\":1,\"legacy\":true}\n"), &errOut)
	raw, err := os.ReadFile(filepath.Join(legacy, ".pose", "indexes", "delivery-integrity.json"))
	if err != nil || !json.Valid(raw) || !strings.Contains(string(raw), "legacy") {
		t.Fatalf("the legacy write did not happen: %q %v", raw, err)
	}
}

func TestSuggestDoesNotInferADomainFromAnUnavailableComponentMap(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	writeIndexFixture(t, root, "repo-map.json", `{"packages":[{"name":"api","path":"api","language":"go"}]}`)
	if domain, source := inferSuggestDomain(root, "api/server.go"); domain == "" || source == "undefined" {
		t.Fatalf("a current map was not used: %q %q", domain, source)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "repo-map.json"), []byte(`{"packages":[{"name":"api","path":"api","language":"javascript"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(root, ".pose", "indexes", "generations"))
	if domain, source := inferSuggestDomain(root, "api/server.go"); domain != "" || source != "undefined" {
		t.Fatalf("a domain was inferred from a map that does not verify: %q %q", domain, source)
	}
}

func TestIndexVerifyReportsAHandEditedIndexWhileTheGenerationIsWhole(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "services.json"), []byte("[{\"edited\":true}]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Engine readers read the generation, so the edit has no effect on them ...
	raw, status, err := posemodel.Store{Root: root}.ReadIndex("services.json")
	if err != nil || status != posemodel.IndexGenerationCurrent || strings.Contains(string(raw), "edited") {
		t.Fatalf("ReadIndex served the edited file: %q %s %v", raw, status, err)
	}
	// ... and the verifier names the file.
	if code, _, errOut := runIndex(t, root, "--verify"); code == 0 || !strings.Contains(errOut, "services.json") || !strings.Contains(errOut, "differ from it") {
		t.Fatalf("verify of a diverged set: code=%d err=%s", code, errOut)
	}
}

// F09: a commit whose durability is unconfirmed is a commit.
func TestIndexAnUnsyncedCommitSaysTheNewGenerationIsCurrent(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	previous := posemodel.Store{Root: root}.ReadIndexGeneration().Descriptor.GenerationID
	writeCloseoutCLIFile(t, root, ".pose/specs/moves-the-graph.md", "---\nslug: moves-the-graph\nstatus: draft\ncreated_at: 2026-10-17\n---\n# Spec\n")
	defer func() { posemodel.PublishSyncHook = nil }()
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && filepath.Base(path) == "generation.json" {
			return errors.New("directory sync refused")
		}
		return nil
	}
	code, out, errOut := runIndex(t, root, "--json")
	posemodel.PublishSyncHook = nil
	current := posemodel.Store{Root: root}.ReadIndexGeneration().Descriptor.GenerationID
	if code == 0 || current == previous {
		t.Fatalf("code=%d current=%s previous=%s: the descriptor was renamed, so the new generation is current", code, current, previous)
	}
	if strings.Contains(errOut, "previous generation stays") || !strings.Contains(errOut, "committed but not confirmed durable") || !strings.Contains(errOut, current) {
		t.Fatalf("the failure does not say what happened: %s", errOut)
	}
	if !strings.Contains(out, current) {
		t.Fatalf("the machine document does not name the effective generation:\n%s", out)
	}
}

func TestIndexWithAnotherPublicationRunningWritesNothingAndSaysSo(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	before := string(indexBytes(t, root, "generation.json"))
	owner := posemodel.CurrentStartLockOwner(root, "indexes", "index.publish")
	raw, _ := json.Marshal(owner)
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", ".generation.lock"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	writeCloseoutCLIFile(t, root, ".pose/specs/waits-for-nobody.md", "---\nslug: waits-for-nobody\nstatus: draft\ncreated_at: 2026-10-17\n---\n# Spec\n")
	code, _, errOut := runIndex(t, root)
	if code == 0 || !strings.Contains(errOut, "another index publication holds") || !strings.Contains(errOut, "nothing was written") {
		t.Fatalf("code=%d err=%s", code, errOut)
	}
	if string(indexBytes(t, root, "generation.json")) != before {
		t.Fatal("a refused publication changed the descriptor")
	}
}

// F10: the refresh reports a retention failure.
func TestRefreshDeliveryIndexReportsAFailedRetention(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root removes read-only directories")
	}
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	base := filepath.Join(root, ".pose", "indexes", "generations")
	for i := 1; i <= 4; i++ {
		d := filepath.Join(base, "gen-00000000000000"+strings.Repeat("0", 1)+string(rune('0'+i)))
		locked := filepath.Join(d, "locked")
		if err := os.MkdirAll(locked, 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(locked, "f"), []byte("x"), 0o644)
		_ = os.Chmod(locked, 0o500)
		past := time.Now().Add(-48 * time.Hour)
		_ = os.Chtimes(d, past, past)
		defer func(p string) { _ = os.Chmod(p, 0o755) }(locked)
	}
	var errOut bytes.Buffer
	refreshDeliveryIndex(root, []byte("{\"schema_version\":1,\"x\":2}\n"), &errOut)
	if !strings.Contains(errOut.String(), "retention of old index generations failed") {
		t.Fatalf("a failed retention was dropped: %q", errOut.String())
	}
	if view := (posemodel.Store{Root: root}).ReadIndexGeneration(); view.Status != posemodel.IndexGenerationCurrent {
		t.Fatalf("the refresh did not publish: %s %s", view.Status, view.Reason)
	}
}

// F13: the refresh names the generation it committed.
func TestRefreshDeliveryIndexNamesACommittedGenerationWhoseDurabilityIsUnconfirmed(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	if code, _, errOut := runIndex(t, root); code != 0 {
		t.Fatalf("index: %d %s", code, errOut)
	}
	previous := posemodel.Store{Root: root}.ReadIndexGeneration().Descriptor.GenerationID
	defer func() { posemodel.PublishSyncHook = nil }()
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && filepath.Base(path) == "generation.json" {
			return errors.New("directory sync refused")
		}
		return nil
	}
	var errOut bytes.Buffer
	refreshDeliveryIndex(root, []byte("{\"schema_version\":1,\"y\":3}\n"), &errOut)
	posemodel.PublishSyncHook = nil
	current := posemodel.Store{Root: root}.ReadIndexGeneration().Descriptor.GenerationID
	if current == previous {
		t.Fatal("the descriptor did not move")
	}
	if strings.Contains(errOut.String(), "not refreshed") || !strings.Contains(errOut.String(), current) || !strings.Contains(errOut.String(), "durability could not be confirmed") {
		t.Fatalf("the refresh denied or hid its committed effect: %q", errOut.String())
	}
}
