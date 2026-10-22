package cli

// Spec pose-update-dry-run-reports-the-whole-update, decision D2: the
// preview runs the update's own operations against an operation-scoped view
// of the instance, so what it reports is what the same run applies. These
// tests compare a preview with the update that follows it on the same
// fixture, by the effects on disk and by the lines both print — not by an
// implementation's internal fields.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// staleInstance is an instance last updated by POSE 6.1.0, with a drifted
// managed template and three capabilities it has not decided: an update has
// manuals, machinery, a stamp and decision requests to deliver.
func staleInstance(t *testing.T) string {
	t.Helper()
	repo := olderInstance(t)
	manifest := filepath.Join(repo, ".pose", "state", "machinery-manifest.json")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["engine_version"] = "6.1.0"
	stamped, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(manifest, append(stamped, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(repo, ".pose", "templates", "spec.md")
	original, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte(strings.Replace(string(original), "## 1.", "## 1 (drifted).", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// entryState is what an update can change about one directory entry.
type entryState struct {
	kind    string // file, dir, link, unlistable
	digest  string
	perm    fs.FileMode
	target  string
	missing bool
}

// instanceState records every entry under root without following links,
// .git excluded. A directory that cannot be listed is recorded as such: the
// update cannot enumerate it either, and the entries the test names in probes
// are read by name inside it.
func instanceState(t *testing.T, root string, probes ...string) map[string]entryState {
	t.Helper()
	out := map[string]entryState{}
	var record func(path, name string) (bool, string)
	var walk func(dir, rel string)
	walk = func(dir, rel string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if rel != "" {
				state := out[rel]
				state.kind = "unlistable"
				out[rel] = state
			}
			return
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			name := entry.Name()
			if rel != "" {
				name = rel + "/" + name
			}
			if name == ".git" {
				continue
			}
			if isDir, _ := record(path, name); isDir {
				walk(path, name)
			}
		}
	}
	record = func(path, name string) (bool, string) {
		info, err := os.Lstat(path)
		if err != nil {
			return false, ""
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, _ := os.Readlink(path)
			out[name] = entryState{kind: "link", target: target}
		case info.IsDir():
			out[name] = entryState{kind: "dir", perm: info.Mode().Perm()}
			return true, "dir"
		case info.Mode().IsRegular():
			raw, err := os.ReadFile(path)
			digest := "unreadable"
			if err == nil {
				sum := sha256.Sum256(raw)
				digest = hex.EncodeToString(sum[:])
			}
			out[name] = entryState{kind: "file", digest: digest, perm: info.Mode().Perm()}
		}
		return false, ""
	}
	walk(root, "")
	for _, probe := range probes {
		record(filepath.Join(root, filepath.FromSlash(probe)), probe)
	}
	return out
}

// appliedEffects names each change between two states the way the preview
// reports them.
func appliedEffects(before, after map[string]entryState) []string {
	effects := []string{}
	for path, now := range after {
		was, existed := before[path]
		switch {
		case !existed:
			effects = append(effects, createVerb(now.kind)+": "+path)
		case was.kind != now.kind:
			effects = append(effects, "replace "+was.kind+" with "+now.kind+": "+path)
		case now.kind == "file" && was.digest != now.digest:
			effects = append(effects, "modify: "+path)
		case now.kind == "link" && was.target != now.target:
			effects = append(effects, "replace link: "+path)
		case was.perm != now.perm:
			effects = append(effects, "change mode: "+path)
		}
	}
	for path, was := range before {
		if _, ok := after[path]; !ok {
			if was.kind == "dir" {
				effects = append(effects, "remove directory: "+path)
			} else {
				effects = append(effects, "remove: "+path)
			}
		}
	}
	sort.Strings(effects)
	return effects
}

func createVerb(kind string) string {
	switch kind {
	case "dir":
		return "create directory"
	case "link":
		return "create link"
	}
	return "create"
}

var predictedEffect = regexp.MustCompile(`^\[DRY-RUN\] would ([a-z ]+): (.+)$`)

// predictedEffects returns the effects a preview's output names, request ids
// normalized: an id hashes the instant its request is opened.
func predictedEffects(out string) []string {
	effects := []string{}
	for _, line := range strings.Split(out, "\n") {
		if m := predictedEffect.FindStringSubmatch(strings.TrimSpace(line)); m != nil && m[1] != "stamp engine_version" {
			effects = append(effects, actionID.ReplaceAllString(m[1]+": "+m[2], "act-<id>"))
		}
	}
	sort.Strings(effects)
	return effects
}

// normalizedEffects is appliedEffects with request ids normalized.
func normalizedEffects(before, after map[string]entryState) []string {
	effects := appliedEffects(before, after)
	for i := range effects {
		effects[i] = actionID.ReplaceAllString(effects[i], "act-<id>")
	}
	sort.Strings(effects)
	return effects
}

var actionID = regexp.MustCompile(`act-[0-9a-f]{16}`)

// requestLines returns the decision-request lines an update printed, in
// order, with the preview prefix and request ids removed: an id hashes the
// instant the request is opened.
func requestLines(out string) []string {
	lines := []string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "[DRY-RUN] ")
		if strings.HasPrefix(line, "[INFO] decision request ") {
			lines = append(lines, actionID.ReplaceAllString(line, "act-<id>"))
		}
	}
	return lines
}

func sameEffects(t *testing.T, what string, predicted, applied []string) {
	t.Helper()
	if strings.Join(predicted, "\n") != strings.Join(applied, "\n") {
		t.Fatalf("%s: the preview predicted\n%s\nand the update did\n%s", what, strings.Join(predicted, "\n"), strings.Join(applied, "\n"))
	}
}

// A directory that can be searched but not listed (mode 0100) is read by
// name by the update; the preview must read it the same way rather than
// reproduce it empty (the defect that stopped the copy-based dry-run). The
// drifted rule inside it is rewritten in place — the file is writable — while
// its backup cannot be created in the read-only directory, which the update
// tolerates: a direct overwrite and a new entry need different permissions.
func TestUpdatePreviewReadsAnExecuteOnlyDirectoryByName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root lists every directory")
	}
	repo := staleInstance(t)
	rule := filepath.Join(repo, ".pose", "rules", "security.md")
	original, err := os.ReadFile(rule)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rule, append(original, []byte("\nlocal drift\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	rules := filepath.Dir(rule)
	if err := os.Chmod(rules, 0o100); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(rules, 0o755) })
	probes := []string{".pose/rules/security.md", ".pose/rules/security.md.pose-backup"}
	before := instanceState(t, repo, probes...)
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	if after := instanceState(t, repo, probes...); len(appliedEffects(before, after)) != 0 {
		t.Fatalf("the preview wrote: %v", appliedEffects(before, after))
	}
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != realCode || realCode != 0 {
		t.Fatalf("preview %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
	applied := normalizedEffects(before, instanceState(t, repo, probes...))
	if !strings.Contains(strings.Join(applied, "\n"), "modify: .pose/rules/security.md") {
		t.Fatalf("fixture: the update did not rewrite the drifted rule: %v", applied)
	}
	sameEffects(t, "execute-only .pose/rules", predictedEffects(dryOut), applied)
}

// Module discovery reads the instance's real source tree, ignore rules
// included: a module added since the last update is merged into
// module-metadata by both, and an ignored one by neither.
func TestUpdatePreviewDiscoversTheSourceTreeTheUpdateDiscovers(t *testing.T) {
	repo := staleInstance(t)
	mustWrite(t, filepath.Join(repo, "services", "billing", "go.mod"), "module billing\n")
	mustWrite(t, filepath.Join(repo, ".gitignore"), "scratch/\n")
	mustWrite(t, filepath.Join(repo, "scratch", "tool", "go.mod"), "module tool\n")
	before := instanceState(t, repo)
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != 0 || realCode != 0 {
		t.Fatalf("preview %d, update %d:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
	if !strings.Contains(realOut, "module-metadata (discovered): services/billing") || strings.Contains(realOut, "scratch/tool") {
		t.Fatalf("fixture: the update's discovery: %s", realOut)
	}
	if !strings.Contains(dryOut, "[DRY-RUN] [INFO] module-metadata (discovered): services/billing") || strings.Contains(dryOut, "scratch/tool") {
		t.Fatalf("the preview did not discover what the update discovers:\n%s", dryOut)
	}
	sameEffects(t, "discovery", predictedEffects(dryOut), normalizedEffects(before, instanceState(t, repo)))
}

// Each decision request is named with what it asks, of whom and with which
// recommendation — the same line the update prints when it opens it — not
// counted.
func TestUpdatePreviewNamesEachDecisionRequestAsTheUpdateDoes(t *testing.T) {
	repo := staleInstance(t)
	_, dryOut := runPose(t, repo, "update", "--dry-run")
	_, realOut := runPose(t, repo, "update", "--no-self")
	applied := requestLines(realOut)
	if len(applied) < 2 {
		t.Fatalf("the update names %d request(s):\n%s", len(applied), realOut)
	}
	for _, line := range applied {
		for _, want := range []string{"capability ", "role maintainer", "spec:pose-configuration-review-"} {
			if !strings.Contains(line, want) {
				t.Fatalf("request line without %q: %s", want, line)
			}
		}
	}
	if got := requestLines(dryOut); strings.Join(got, "\n") != strings.Join(applied, "\n") {
		t.Fatalf("the preview names\n%s\nand the update opened\n%s", strings.Join(got, "\n"), strings.Join(applied, "\n"))
	}
}

// The preview needs no temporary directory: with TMPDIR unwritable it still
// runs, and leaves TMPDIR as it found it.
func TestUpdatePreviewNeedsNoTemporaryCopy(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes into read-only directories")
	}
	repo := staleInstance(t)
	tmp := filepath.Join(t.TempDir(), "tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmp, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tmp, 0o755) })
	t.Setenv("TMPDIR", tmp)
	code, out := runPose(t, repo, "update", "--dry-run")
	if code != 0 || !strings.Contains(out, "Result: DRY-RUN") {
		t.Fatalf("preview with an unwritable TMPDIR: %d %s", code, out)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Fatalf("the preview left %d entries in TMPDIR", len(entries))
	}
}

var _ = bytes.NewBuffer
var _ = errors.New
