//go:build linux

package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// processReadBytes is the bytes this process has read so far (rchar of /proc/self/io),
// the measure of what a call parses: the list tools read their collection with plain
// file reads, so it sees them where a Files wrapper does not.
func processReadBytes(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile("/proc/self/io")
	if err != nil {
		t.Skipf("no /proc/self/io: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		var n int
		if _, err := fmt.Sscanf(line, "rchar: %d", &n); err == nil {
			return n
		}
	}
	t.Skip("no rchar in /proc/self/io")
	return 0
}

func writePaddedSpec(t *testing.T, root, slug string) {
	t.Helper()
	path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("---\nslug: %s\nstatus: draft\ncreated_at: 2026-06-01\n---\n\n# Spec: %s\n\n%s\n", slug, slug, strings.Repeat("filler line to give the file some weight\n", 100))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// R8 measurement for the non-indexed collections: a page costs one recomputation of
// the collection whatever its size (limit bounds what is returned, not what is read),
// the bytes read grow linearly with the collection, and the response follows the page.
// The numbers are logged; the assertions keep the relations.
func TestPageCostIsMeasuredAndBounded(t *testing.T) {
	type sample struct {
		read, response int
		elapsed        time.Duration
	}
	measure := func(n, limit int) sample {
		root := t.TempDir()
		for i := 0; i < n; i++ {
			writePaddedSpec(t, root, fmt.Sprintf("spec-%03d", i))
		}
		ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
		defer ts.Close()
		args := map[string]any{}
		if limit > 0 {
			args["limit"] = limit
		}
		raw, _ := json.Marshal(args)
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":%s}}`, raw)
		post(t, ts, body) // warm the process (first-use reads are not the collection's)
		before, start := processReadBytes(t), time.Now()
		_, out := post(t, ts, body)
		elapsed := time.Since(start)
		after := processReadBytes(t)
		resp, _ := json.Marshal(out.Result["structuredContent"])
		return sample{read: after - before, response: len(resp), elapsed: elapsed}
	}
	smallFull, smallPage := measure(20, 0), measure(20, 5)
	largeFull, largePage := measure(80, 0), measure(80, 5)
	for _, row := range []struct {
		name string
		s    sample
	}{{"n=20 full", smallFull}, {"n=20 limit=5", smallPage}, {"n=80 full", largeFull}, {"n=80 limit=5", largePage}} {
		t.Logf("%-14s read=%7dB response=%6dB latency=%v", row.name, row.s.read, row.s.response, row.s.elapsed)
	}
	if smallFull.read < 20*4000 {
		t.Fatalf("the measurement did not see the collection being read: %d bytes", smallFull.read)
	}
	// A page reads the whole collection, as the unpaginated call does (within noise).
	for _, p := range [][2]sample{{smallPage, smallFull}, {largePage, largeFull}} {
		lo, hi := p[1].read*8/10, p[1].read*12/10
		if p[0].read < lo || p[0].read > hi {
			t.Fatalf("a page changed the work done: read %d bytes against %d for the whole call", p[0].read, p[1].read)
		}
	}
	// The bytes read grow linearly with the collection (4x the specs, at most ~5x the bytes).
	if largeFull.read > smallFull.read*5 {
		t.Fatalf("the work grew faster than the collection: n=20 %d, n=80 %d", smallFull.read, largeFull.read)
	}
	// The response follows the page size, not the collection size.
	if largePage.response > smallPage.response*2 || largePage.response*3 > largeFull.response {
		t.Fatalf("the response did not follow the page: %d/%d against %d", smallPage.response, largePage.response, largeFull.response)
	}
}

// With a published generation, verifying it reads and hashes every index file.
// A page must not repeat that: the verification is reused while the descriptor and
// the files it names are unchanged, so the second page reads far less than the
// first call did (spec pose-generation-bound-query-cursors R5, R8; review finding).
func TestAPageDoesNotReverifyAnUnchangedGeneration(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 6; i++ {
		writePaddedSpec(t, root, fmt.Sprintf("spec-%03d", i))
	}
	outputs := map[string][]byte{}
	for _, name := range pose.IndexGenerationFiles {
		outputs[name] = []byte("[" + strings.Repeat(" ", 1<<20) + "]\n")
	}
	if _, err := (pose.Store{Root: root}).PublishIndexGeneration(outputs, map[string]string{"specs": "sha256:a"}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	defer ts.Close()
	call := func() int {
		before := processReadBytes(t)
		post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"limit":2}}}`)
		return processReadBytes(t) - before
	}
	first, second := call(), call()
	t.Logf("first call read %d B (verifies the %d-file generation); the next read %d B", first, len(pose.IndexGenerationFiles), second)
	if first < len(pose.IndexGenerationFiles)<<20 {
		t.Fatalf("the first call did not verify the generation: %d B", first)
	}
	if second > first/4 {
		t.Fatalf("a page re-verified an unchanged generation: %d B after %d B", second, first)
	}
	// A change to a legacy file beside the generation is seen again (status disclosed).
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "packages.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"limit":2}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	page, _ := sc["page"].(map[string]any)
	if diverged, _ := page["generation_diverged"].([]any); len(diverged) != 1 || diverged[0] != "packages.json" {
		t.Fatalf("a changed legacy file was not seen after the reuse: %+v", page)
	}
}

func TestAReusedGenerationSeesASymlinkTargetChange(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		writePaddedSpec(t, root, fmt.Sprintf("spec-%03d", i))
	}
	outputs := map[string][]byte{}
	for _, name := range pose.IndexGenerationFiles {
		outputs[name] = []byte("[]\n")
	}
	if _, err := (pose.Store{Root: root}).PublishIndexGeneration(outputs, map[string]string{"specs": "sha256:a"}); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(root, ".pose", "indexes")
	// A legacy file that is a link to a copy with the generation's bytes: not diverged.
	good := filepath.Join(idx, "good-copy.json")
	if err := os.WriteFile(good, outputs["packages.json"], 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(idx, "packages.json")
	_ = os.Remove(link)
	if err := os.Symlink("good-copy.json", link); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	defer ts.Close()
	diverged := func() []any {
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"limit":2}}}`)
		sc, _ := out.Result["structuredContent"].(map[string]any)
		page, _ := sc["page"].(map[string]any)
		d, _ := page["generation_diverged"].([]any)
		return d
	}
	if d := diverged(); len(d) != 0 {
		t.Fatalf("a link to identical bytes diverged: %v", d)
	}
	// Retarget the link at different bytes: the cached verification must not hide it.
	bad := filepath.Join(idx, "bad-copy.json")
	if err := os.WriteFile(bad, []byte("{\"changed\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(link)
	if err := os.Symlink("bad-copy.json", link); err != nil {
		t.Fatal(err)
	}
	if d := diverged(); len(d) != 1 || d[0] != "packages.json" {
		t.Fatalf("a retargeted link was not seen after the reuse: %v", d)
	}
}

func TestAReusedGenerationSeesAnEditThatPreservesSizeAndModTime(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		writePaddedSpec(t, root, fmt.Sprintf("spec-%03d", i))
	}
	outputs := map[string][]byte{}
	for _, name := range pose.IndexGenerationFiles {
		outputs[name] = []byte("[]\n")
	}
	if _, err := (pose.Store{Root: root}).PublishIndexGeneration(outputs, map[string]string{"specs": "sha256:a"}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	defer ts.Close()
	diverged := func() []any {
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"limit":2}}}`)
		sc, _ := out.Result["structuredContent"].(map[string]any)
		page, _ := sc["page"].(map[string]any)
		d, _ := page["generation_diverged"].([]any)
		return d
	}
	if d := diverged(); len(d) != 0 {
		t.Fatalf("a whole generation diverged: %v", d)
	}
	legacy := filepath.Join(root, ".pose", "indexes", "packages.json")
	info, err := os.Stat(legacy)
	if err != nil {
		t.Fatal(err)
	}
	// Same size, same modification time, different bytes.
	if err := os.WriteFile(legacy, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(legacy, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if d := diverged(); len(d) != 1 || d[0] != "packages.json" {
		t.Fatalf("an edit that restored size and mtime was hidden by the cache: %v", d)
	}
}
