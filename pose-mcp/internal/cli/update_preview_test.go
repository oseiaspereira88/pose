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
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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

// Fixed-time parity exercises the production execution entry, including the
// install pre/post gates and generation publication, on the same original tree.
func TestUpdatePreviewParityAcrossUpdateBranches(t *testing.T) {
	for _, tc := range []struct {
		name    string
		force   bool
		locale  string
		prepare func(*testing.T, string)
	}{
		{name: "plain-stale"},
		{name: "force-stale", force: true},
		{name: "plain-pt", locale: "pt-BR"},
		{name: "force-pt", force: true, locale: "pt-BR"},
		{name: "plain-current", prepare: func(t *testing.T, root string) { updateOnce(t, root) }},
		{name: "force-current", force: true, prepare: func(t *testing.T, root string) { updateOnce(t, root) }},
		{name: "automatic-portuguese", prepare: func(t *testing.T, root string) {
			if code, out := runPose(t, root, "update", "--locale", "pt-BR", "--no-self"); code != 0 {
				t.Fatalf("fixture: %s", out)
			}
		}},
		{name: "explicit-english-switch", locale: "en", prepare: func(t *testing.T, root string) {
			if code, out := runPose(t, root, "update", "--locale", "pt-BR", "--no-self"); code != 0 {
				t.Fatalf("fixture: %s", out)
			}
		}},
		{name: "answered-unapplied-and-open-requests", prepare: func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:ada"]},"identity_assurance":"declared"}`)
			updateOnce(t, root)
			requests, err := configurationReviewRequests(root)
			if err != nil {
				t.Fatal(err)
			}
			answerReview(t, root, requestFor(t, requests, "contract-nodes"), "adopt", "")
		}},
		{name: "v1-policy-and-profile-migrations", prepare: func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`)
			mustWrite(t, filepath.Join(root, ".pose/review-profiles/spec-closeout.json"), `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
		}},
		{name: "empty-adoption-and-curated-metadata", prepare: func(t *testing.T, root string) {
			path := filepath.Join(root, ".pose/policy/review.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			doc["adopted_at"] = ""
			doc["contract_adoptions"] = map[string]any{posemodel.ReviewContracts()[0].ID: ""}
			raw, _ = json.Marshal(doc)
			mustWrite(t, path, string(raw))
			mustWrite(t, filepath.Join(root, "go.mod"), "module root\n")
			mustWrite(t, filepath.Join(root, ".pose/indexes/module-metadata.json"), `{"schemaVersion":1,"defaults":{"owner":"curated"},"modules":{"`+filepath.Base(root)+`":{"owner":"team","domain":"curated","custom":"keep"}}}`)
		}},
		{name: "schema-zero", prepare: func(t *testing.T, root string) { mustWrite(t, filepath.Join(root, ".pose/schema-version"), "0\n") }},
		{name: "future-schema", prepare: func(t *testing.T, root string) { mustWrite(t, filepath.Join(root, ".pose/schema-version"), "999\n") }},
		{name: "force-missing-policy-onboarding", force: true, prepare: func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, ".pose/policy/review.json")); err != nil {
				t.Fatal(err)
			}
			paths, _ := filepath.Glob(filepath.Join(root, ".pose/specs/*-pose-onboarding.md"))
			for _, path := range paths {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{name: "missing-computed-index", prepare: func(t *testing.T, root string) {
			if err := os.RemoveAll(filepath.Join(root, ".pose/indexes/generations")); err != nil {
				t.Fatal(err)
			}
			for _, name := range append(append([]string{}, posemodel.IndexGenerationFiles...), "generation.json") {
				_ = os.Remove(filepath.Join(root, ".pose/indexes", name))
			}
			mustWrite(t, filepath.Join(root, "services/billing/go.mod"), "module billing\n")
		}},
		{name: "final-gate-refusal", force: true, prepare: func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-01-invalid.md"), "---\nslug: invalid\nstatus: broken\n---\n# Invalid\n")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := staleInstance(t)
			if tc.prepare != nil {
				tc.prepare(t, root)
			}
			now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
			u := instanceUpdate{root: root, force: tc.force, localeFlag: tc.locale, locale: localeEN}
			if tc.locale == "pt-BR" {
				u.locale = localePtBR
			}
			before := instanceState(t, root)
			p := posemodel.NewPreviewFiles(now)
			var predicted, applied bytes.Buffer
			pc := u.apply(p, &predicted, &predicted)
			sameEffects(t, "preview must not write", nil, appliedEffects(before, instanceState(t, root)))
			if gaps := p.Gaps(); len(gaps) > 0 {
				t.Fatalf("preview gaps: %v", gaps)
			}
			changes := []string{}
			for _, c := range p.Changes(root) {
				changes = append(changes, c.Verb+": "+c.Path)
			}
			sort.Strings(changes)

			ac := u.apply(posemodel.NativeFilesAt(now), &applied, &applied)
			if pc != ac {
				t.Fatalf("preview %d, apply %d:\n%s\n---\n%s", pc, ac, &predicted, &applied)
			}
			if predicted.String() != applied.String() {
				t.Fatalf("shared operation wording diverged:\n%s\n---\n%s", &predicted, &applied)
			}
			sameEffects(t, "effects", changes, appliedEffects(before, instanceState(t, root)))
			// A matching path count is insufficient: compare every projected byte,
			// mode and link target to the actual result, including requests and indexes.
			for path := range instanceState(t, root) {
				physical := filepath.Join(root, filepath.FromSlash(path))
				want, err := os.Lstat(physical)
				if err != nil {
					t.Fatal(err)
				}
				got, err := p.Lstat(physical)
				if err != nil {
					t.Fatal(err)
				}
				if got.Mode() != want.Mode() {
					t.Errorf("%s: projected mode %v, actual %v", path, got.Mode(), want.Mode())
				}
				if want.Mode().IsRegular() {
					a, e := os.ReadFile(physical)
					if e != nil {
						t.Fatal(e)
					}
					b, e := p.ReadFile(physical)
					if e != nil {
						t.Fatal(e)
					}
					if !bytes.Equal(a, b) {
						t.Errorf("%s projected content differs", path)
					}
				}
				if want.Mode()&fs.ModeSymlink != 0 {
					a, _ := os.Readlink(physical)
					b, _ := p.Readlink(physical)
					if a != b {
						t.Errorf("%s projected link %s, actual %s", path, b, a)
					}
				}
			}
		})
	}
}

// Failed relevant reads and a refused configuration-review evaluation cannot
// turn an otherwise current instance into a no-op verdict.
func TestUpdatePreviewDoesNotCallLostEvaluationNoChange(t *testing.T) {
	for _, kind := range []string{"malformed-review-policy", "malformed-review-profile", "malformed-module-metadata", "malformed-changelog-policy", "corrupt-git-index", "unreadable-review-policy", "unlistable-source"} {
		t.Run(kind, func(t *testing.T) {
			root := staleInstance(t)
			if code, out := runPose(t, root, "update", "--no-self"); code != 0 {
				t.Fatalf("fixture: %d %s", code, out)
			}
			policy := filepath.Join(root, ".pose/policy/review.json")
			switch kind {
			case "malformed-review-policy":
				mustWrite(t, policy, "{broken")
			case "malformed-review-profile":
				mustWrite(t, filepath.Join(root, ".pose/review-profiles/spec-closeout.json"), "{broken")
			case "malformed-module-metadata":
				mustWrite(t, filepath.Join(root, ".pose/indexes/module-metadata.json"), "{broken")
			case "malformed-changelog-policy":
				mustWrite(t, filepath.Join(root, ".pose/policy/changelog.json"), "{broken")
			case "corrupt-git-index":
				mustWrite(t, filepath.Join(root, ".git/index"), "invalid-index")
			case "unreadable-review-policy":
				if err := os.Chmod(policy, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(policy, 0o644) })
			case "unlistable-source":
				dir := filepath.Join(root, "services/private")
				mustWrite(t, filepath.Join(dir, "go.mod"), "module private\n")
				if err := os.Chmod(dir, 0o100); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			}
			code, out := runPose(t, root, "update", "--dry-run")
			if code == 0 || strings.Contains(out, "would change nothing") || !strings.Contains(out, "INCOMPLETE") {
				t.Fatalf("lost %s evaluated as complete: %d %s", kind, code, out)
			}
		})
	}
}

// Preserve the scanner's submodule-local ignore, Android and curated-root-alias
// behavior while projecting the governance index that discovery updates.
func TestUpdatePreviewDiscoversSubmodulesAndroidAndCuratedMetadata(t *testing.T) {
	root := submoduleWithIgnoredCache(t)
	var out bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &out, &out); code != 0 {
		t.Fatalf("install fixture: %d %s", code, &out)
	}
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	mustWrite(t, filepath.Join(root, "mod/new-tool/go.mod"), "module example.com/new\n")
	mustWrite(t, filepath.Join(root, "apps/mobile/build.gradle.kts"), "plugins {}\n")
	mustWrite(t, filepath.Join(root, "apps/mobile/src/main/AndroidManifest.xml"), "<manifest/>\n")
	mustWrite(t, filepath.Join(root, ".pose/indexes/module-metadata.json"), `{"schemaVersion":1,"modules":{"`+filepath.Base(root)+`":{"owner":"curated","domain":"root","custom":"keep"}}}`)
	now := time.Now()
	u := instanceUpdate{root: root, force: true, locale: localeEN}
	p := posemodel.NewPreviewFiles(now)
	before := instanceState(t, root)
	out.Reset()
	if code := u.apply(p, &out, &out); code != 0 {
		t.Fatalf("projected update: %d %s", code, &out)
	}
	sameEffects(t, "preview wrote", nil, appliedEffects(before, instanceState(t, root)))
	if len(p.Gaps()) != 0 {
		t.Fatalf("incomplete discovery: %v", p.Gaps())
	}
	path := filepath.Join(root, ".pose/indexes/module-metadata.json")
	predicted, err := p.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	if err := json.Unmarshal(predicted, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Modules["mod/new-tool"] == nil || metadata.Modules["mod/tool"] == nil || metadata.Modules["apps/mobile"]["domain"] != "android" {
		t.Fatalf("lost real source modules: %s", predicted)
	}
	if metadata.Modules["mod/cache"] != nil || metadata.Modules["."] != nil || metadata.Modules[filepath.Base(root)]["custom"] != "keep" {
		t.Fatalf("lost ignore/alias/curation: %s", predicted)
	}
	out.Reset()
	if code := u.apply(posemodel.NativeFilesAt(now), &out, &out); code != 0 {
		t.Fatalf("actual update: %d %s", code, &out)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(predicted, actual) {
		t.Fatalf("discovery differs:\n%s\n---\n%s", predicted, actual)
	}
}

// Inject the same provider failure at the shared operation boundary in both
// modes. Compare partial delivery and diagnostics, including failures after an
// index generation has files but before its descriptor commits.
type updateFaultFiles struct {
	posemodel.Files
	op, match string
	fired     bool
}

func (f *updateFaultFiles) deny(op, path string) error {
	if !f.fired && f.op == op && strings.Contains(filepath.ToSlash(path), f.match) {
		f.fired = true
		return &fs.PathError{Op: op, Path: path, Err: fs.ErrPermission}
	}
	return nil
}
func (f *updateFaultFiles) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if e := f.deny("write", path); e != nil {
		return e
	}
	return f.Files.WriteFile(path, data, perm)
}
func (f *updateFaultFiles) CreateExclusive(path string, data []byte, perm fs.FileMode, sync bool) error {
	if e := f.deny("exclusive", path); e != nil {
		return e
	}
	return f.Files.CreateExclusive(path, data, perm, sync)
}
func (f *updateFaultFiles) Publish(path string, data []byte, perm fs.FileMode, class string, expect posemodel.PublishExpectation) error {
	if e := f.deny("publish", path); e != nil {
		return e
	}
	return f.Files.Publish(path, data, perm, class, expect)
}
func (f *updateFaultFiles) RemoveAll(path string) error {
	if e := f.deny("remove-all", path); e != nil {
		return e
	}
	return f.Files.RemoveAll(path)
}

func TestUpdatePreviewPreservesPartialDeliveryAtFailureBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, op, match string
		force           bool
	}{
		{"manual-backup", "write", "AGENTS.md.pose-backup", false},
		{"manual-publish", "publish", "/AGENTS.md", false},
		{"machinery-backup", "write", "templates/spec.md.pose-backup", false},
		{"request-exclusive", "exclusive", "/actions/act-", false},
		{"generation-file", "publish", "/indexes/generations/", true},
		{"generation-commit", "publish", "/indexes/generation.json", true},
		{"generation-retention", "remove-all", "/indexes/generations/", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := staleInstance(t)
			// An engine-owned section edit requires a manual backup and publication.
			path := filepath.Join(root, "AGENTS.md")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mustWrite(t, path, strings.Replace(string(raw), "## Verification", "local edit\n\n## Verification", 1))
			if tc.name == "generation-retention" {
				for _, id := range []string{"0000000000000001", "0000000000000002", "0000000000000003", "0000000000000004", "0000000000000005"} {
					dir := filepath.Join(root, ".pose/indexes/generations/gen-"+id)
					mustWrite(t, filepath.Join(dir, "sentinel"), "old\n")
					old := time.Now().Add(-24 * time.Hour)
					if err := os.Chtimes(dir, old, old); err != nil {
						t.Fatal(err)
					}
				}
			}
			now := time.Now()
			u := instanceUpdate{root: root, force: tc.force, locale: localeEN}
			before := instanceState(t, root)
			p := posemodel.NewPreviewFiles(now)
			pf := &updateFaultFiles{Files: p, op: tc.op, match: tc.match}
			var predicted, actual bytes.Buffer
			pc := u.apply(pf, &predicted, &predicted)
			sameEffects(t, "preview wrote", nil, appliedEffects(before, instanceState(t, root)))
			changes := []string{}
			for _, c := range p.Changes(root) {
				changes = append(changes, c.Verb+": "+c.Path)
			}
			sort.Strings(changes)
			af := &updateFaultFiles{Files: posemodel.NativeFilesAt(now), op: tc.op, match: tc.match}
			ac := u.apply(af, &actual, &actual)
			if !pf.fired || !af.fired {
				t.Fatalf("failure boundary not reached: preview %v apply %v", pf.fired, af.fired)
			}
			if pc != ac || predicted.String() != actual.String() {
				t.Fatalf("partial delivery differs: preview %d, apply %d\n%s\n---\n%s", pc, ac, &predicted, &actual)
			}
			sameEffects(t, "partial delivery", changes, appliedEffects(before, instanceState(t, root)))
		})
	}
}

// A tolerated link replacement failure is a known refusal, not evidence that
// the operation successfully evaluated as a no-op. Native best-effort behavior
// remains unchanged, and the preview explains the refusal without inventing a
// fatal native outcome.
func TestUpdatePreviewDistinguishesRefusedOperationsFromNoOp(t *testing.T) {
	root := staleInstance(t)
	for i := 0; i < 2; i++ {
		if code, out := runPose(t, root, "update", "--no-self", "--force"); code != 0 {
			t.Fatalf("fixture: %d %s", code, out)
		}
	}
	dir := filepath.Join(root, ".claude/skills")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	before := instanceState(t, root)
	code, out := runPose(t, root, "update", "--dry-run", "--force")
	sameEffects(t, "preview wrote", nil, appliedEffects(before, instanceState(t, root)))
	if code == 0 || !strings.Contains(out, "operation(s) refused") || strings.Contains(out, "would change nothing") || strings.Contains(out, "would fail") {
		t.Fatalf("refusal misreported: %d %s", code, out)
	}
	if code, out := runPose(t, root, "update", "--no-self", "--force"); code != 0 {
		t.Fatalf("native best-effort behavior changed: %d %s", code, out)
	}
}

// Suppression is local to the parsed preview invocation. A subsequent apply
// still records ordinary usage; another preview changes neither that log nor
// its salt/metadata, even when usage is enabled.
func TestUpdatePreviewLeavesUsageRecordingForApply(t *testing.T) {
	root := staleInstance(t)
	usageDir := t.TempDir()
	t.Setenv("POSE_USAGE_DIR", usageDir)
	t.Setenv("POSE_USAGE_DISABLED", "")
	t.Setenv("POSE_USAGE_SUPPRESS_CLI", "")
	if code, out := runPose(t, root, "update", "--dry-run"); code != 0 {
		t.Fatalf("preview: %d %s", code, out)
	}
	if entries, err := os.ReadDir(usageDir); err != nil || len(entries) != 0 {
		t.Fatalf("preview recorded usage: %v %v", entries, err)
	}
	if code, out := runPose(t, root, "update", "--no-self"); code != 0 {
		t.Fatalf("apply: %d %s", code, out)
	}
	before := instanceState(t, usageDir)
	if len(before) == 0 {
		t.Fatal("apply failed to record usage after preview")
	}
	if code, out := runPose(t, root, "update", "--dry-run"); code != 0 {
		t.Fatalf("preview after apply: %d %s", code, out)
	}
	sameEffects(t, "preview changed usage storage", nil, appliedEffects(before, instanceState(t, usageDir)))
}

// The real update writer follows absolute link components; a lost alias model
// must also remain incomplete through the shared command's best-effort paths.
func TestUpdatePreviewSourceReviewFilesystemReachability(t *testing.T) {
	for _, kind := range []string{"missing-dotdot", "symlink-dotdot", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			root := staleInstance(t)
			outside := t.TempDir()
			path := filepath.Join(root, ".pose/rules/security.md")
			victim := filepath.Join(outside, "victim")
			mustWrite(t, victim, "external original\n")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "hardlink":
				if err := os.Link(victim, path); err != nil {
					t.Skip(err)
				}
			case "missing-dotdot":
				if err := os.Symlink(outside+"/missing/../victim", path); err != nil {
					t.Skip(err)
				}
			case "symlink-dotdot":
				target := t.TempDir()
				if err := os.Mkdir(filepath.Join(target, "child"), 0700); err != nil {
					t.Fatal(err)
				}
				mustWrite(t, filepath.Join(target, "victim"), "resolved original\n")
				if err := os.Symlink(filepath.Join(target, "child"), filepath.Join(outside, "component")); err != nil {
					t.Skip(err)
				}
				if err := os.Symlink(outside+"/component/../victim", path); err != nil {
					t.Skip(err)
				}
			}
			before := instanceState(t, root, path, victim)
			code, out := runPose(t, root, "update", "--dry-run", "--force")
			if after := instanceState(t, root, path, victim); !reflect.DeepEqual(before, after) {
				t.Fatal("preview changed physical state")
			}
			if kind == "hardlink" {
				if code == 0 || !strings.Contains(out, "INCOMPLETE") || !strings.Contains(out, "inode aliases") {
					t.Fatalf("hardlink must stay incomplete: %d %s", code, out)
				}
			} else {
				applyCode, applyOut := runPose(t, root, "update", "--no-self", "--force")
				if (code == 0) != (applyCode == 0) {
					t.Fatalf("preview=%d %s\napply=%d %s", code, out, applyCode, applyOut)
				}
				raw, err := os.ReadFile(victim)
				if err != nil || string(raw) != "external original\n" {
					t.Fatalf("lexically cleaned victim changed: %q %v", raw, err)
				}
			}
		})
	}
}
