package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestGovernanceOperationPerformanceCapturesTrailersOnce(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{1, 3, 6} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root := t.TempDir()
			artifactGit(t, root, "init", "-q")
			artifactGit(t, root, "config", "user.name", "Fixture")
			artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
			artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
			message := "one commit, several owners\n"
			for i := 0; i < count; i++ {
				slug := fmt.Sprintf("sample-%d", i)
				writeArtifactTestFile(t, root, ".pose/specs/"+slug+".md", fmt.Sprintf("---\nslug: %s\nstatus: draft\n---\n\n# Spec\n\n## 3. Technical Plan\n\n### Artifacts\n- none: analysis\n", slug))
				message += "\nPOSE-Spec: " + slug
			}
			artifactGit(t, root, "add", ".pose")
			artifactGit(t, root, "commit", "-q", "-m", message)
			bin, events := t.TempDir(), filepath.Join(t.TempDir(), "git-events")
			// A tiny real-Git wrapper records only trailer-log launches. These
			// synthetic paths/arguments contain no newlines; no timing is inferred.
			wrapper := "#!/bin/sh\nfor arg do\n  if [ \"$arg\" = '--grep=POSE-Spec:' ]; then printf 'log\\n' >> " + shellQuote(events) + "; fi\ndone\nexec " + shellQuote(realGit) + " \"$@\"\n"
			if err := os.WriteFile(filepath.Join(bin, "git"), []byte(wrapper), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			specs, _, sets, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
			if err != nil || len(gaps) != 0 || len(specs) != count || len(sets) != count {
				t.Fatalf("lost collector inputs: specs=%d sets=%d gaps=%v err=%v", len(specs), len(sets), gaps, err)
			}
			raw, err := os.ReadFile(events)
			if err != nil {
				t.Fatal(err)
			}
			if got := bytes.Count(raw, []byte("log\n")); got != 1 {
				t.Fatalf("full trailer log read %d times for %d specs, want one bounded capture", got, count)
			}
		})
	}
}

// The pre-optimization scoring rule, retained as a test oracle. It normalizes
// each pair and does not call either prepared production helper.
func governanceLegacySimilarity(left, right string) float64 {
	left, right = normalizeFollowup(left), normalizeFollowup(right)
	leftTokens, rightTokens := followupTokens(left), followupTokens(right)
	intersection, union := 0, len(leftTokens)
	for token := range rightTokens {
		if leftTokens[token] {
			intersection++
		} else {
			union++
		}
	}
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(intersection) / float64(union)
	}
	sequence := ratcliffRatio([]rune(left), []rune(right))
	if sequence > jaccard {
		return sequence
	}
	return jaccard
}

func governanceLegacyClusters(entries []followup, threshold float64) []nearDuplicateCandidate {
	parents := make([]int, len(entries))
	for i := range parents {
		parents[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parents[i] != i {
			parents[i] = find(parents[i])
		}
		return parents[i]
	}
	for left := range entries {
		for right := left + 1; right < len(entries); right++ {
			if entries[left].Spec != entries[right].Spec && governanceLegacySimilarity(entries[left].Text, entries[right].Text) >= threshold {
				parents[find(left)] = find(right)
			}
		}
	}
	groups := map[int][]followup{}
	for i, entry := range entries {
		groups[find(i)] = append(groups[find(i)], entry)
	}
	keys := []int{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	result := []nearDuplicateCandidate{}
	for _, key := range keys {
		specs := map[string]bool{}
		for _, entry := range groups[key] {
			specs[entry.Spec] = true
		}
		if len(specs) < 2 {
			continue
		}
		candidate := nearDuplicateCandidate{}
		for _, entry := range groups[key] {
			disposition := entry.RawDisposition
			if disposition == "" {
				disposition = "open"
			}
			candidate.Members = append(candidate.Members, nearDuplicateMember{entry.Spec, entry.Text, disposition})
		}
		for spec := range specs {
			candidate.Specs = append(candidate.Specs, spec)
		}
		sort.Strings(candidate.Specs)
		result = append(result, candidate)
	}
	return result
}

func TestGovernanceOperationPerformanceClustering(t *testing.T) {
	texts := []string{"", "!!!", "the and para", "Árvore `ação` 123", "árvore ação 123", "audit the release gate", "audit release gate", "release audit before publication", "aabcc abcab"}
	entries := []followup{}
	for i, text := range texts {
		entries = append(entries, followup{Spec: fmt.Sprintf("sample-%d", i%4), Text: text})
		for _, right := range texts {
			got := preparedFollowupSimilarity(prepareFollowupSimilarity(text), prepareFollowupSimilarity(right))
			if got != governanceLegacySimilarity(text, right) {
				t.Fatalf("changed score %q %q: %g", text, right, got)
			}
		}
	}
	for _, threshold := range []float64{0, 0.2, 0.6, 0.8, 1} {
		got, want := clusterFollowups(entries, threshold), governanceLegacyClusters(entries, threshold)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("changed clusters/order at %g:\n%+v\n%+v", threshold, got, want)
		}
	}
	// Punctuation exceeds the budget while normalizing to an empty sequence,
	// keeping the real fallback pair comparisons fast. Ordinary members still
	// produce candidates and must not be truncated or reordered.
	large := []followup{{Spec: "padding", Text: strings.Repeat("!", 1024*1024+1)},
		{Spec: "one", Text: "audit release gate"}, {Spec: "two", Text: "audit the release gate"},
		{Spec: "two", Text: "audit release gate"}}
	for _, threshold := range []float64{0, 0.2, 0.6, 0.8, 1} {
		if got, want := clusterFollowups(large, threshold), governanceLegacyClusters(large, threshold); !reflect.DeepEqual(got, want) {
			t.Fatalf("fallback changed candidates at %g: %+v %+v", threshold, got, want)
		}
	}
}

func BenchmarkGovernanceOperationPerformanceClustering(b *testing.B) {
	for _, count := range []int{8, 32} {
		for _, length := range []int{32, 128} {
			entries := make([]followup, count)
			for i := range entries {
				entries[i] = followup{Spec: fmt.Sprintf("spec-%d", i), Text: strings.Repeat("audit gate ", length/11) + fmt.Sprint(i)}
			}
			for _, implementation := range []string{"legacy", "prepared"} {
				b.Run(fmt.Sprintf("N%d-L%d/%s", count, length, implementation), func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						if implementation == "legacy" {
							governanceLegacyClusters(entries, 0.6)
						} else {
							clusterFollowups(entries, 0.6)
						}
					}
				})
			}
		}
	}
}

func TestGovernanceOperationPerformanceCapturedChangeSetParity(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.name", "Fixture")
	artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
	artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
	writeArtifactTestFile(t, root, "original.txt", "first\n")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "initial\n\nPOSE-Spec: sample")
	artifactGit(t, root, "mv", "original.txt", "renamed.txt")
	artifactGit(t, root, "commit", "-q", "-m", "rename\n\nPOSE-Spec: sample")
	captured, err := commitsWithSpecTrailer(root, "sample")
	if err != nil {
		t.Fatal(err)
	}
	before, err := resolveGitChangeSet(root, "sample", "", "")
	if err != nil {
		t.Fatal(err)
	}
	after, err := resolveGitChangeSetWithTrailers(root, "sample", "", "", captured)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("changed attribution:\n%+v\n%+v", before, after)
	}
	after.Commits[0] = "mutated"
	if captured[0] == "mutated" {
		t.Fatal("mutable capture aliases returned change set")
	}
	// A later operation observes a new trailer rather than retaining old refs.
	writeArtifactTestFile(t, root, "later.txt", "later\n")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "later\n\nPOSE-Spec: sample")
	later, err := resolveGitChangeSet(root, "sample", "", "")
	if err != nil || len(later.Commits) != 3 || later.ID == before.ID {
		t.Fatalf("later operation retained stale attribution: %+v %v", later, err)
	}
}

func TestGovernanceOperationPerformanceChangeSetIndependentGolden(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.name", "Fixture")
	artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
	artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
	writeArtifactTestFile(t, root, "base.txt", "before\n")
	writeArtifactTestFile(t, root, "remove.txt", "remove\n")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")
	base := artifactGit(t, root, "rev-parse", "HEAD")
	writeArtifactTestFile(t, root, "base.txt", "after\n")
	writeArtifactTestFile(t, root, "old.txt", "renamed later\n")
	artifactGit(t, root, "rm", "-q", "remove.txt")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "owned\n\nPOSE-Spec: sample")
	first := artifactGit(t, root, "rev-parse", "HEAD")
	writeArtifactTestFile(t, root, "noise.txt", "unrelated\n")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "unrelated\n\nPOSE-Spec: other")
	artifactGit(t, root, "mv", "old.txt", "new.txt")
	artifactGit(t, root, "commit", "-q", "-m", "rename\n\nPOSE-Spec: sample")
	last := artifactGit(t, root, "rev-parse", "HEAD")
	// Use independent raw Git bytes, not either change-set resolver. Preserve
	// newlines: the fixture helper normally trims command output.
	hash := sha256.New()
	for _, commit := range []string{first, last} {
		raw, err := exec.Command("git", "-C", root, "show", "--binary", "--no-ext-diff", "--format=", commit, "--").Output()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = hash.Write(raw)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	id := sha256.Sum256([]byte("sample\x00" + base + "\x00" + last + "\x00" + digest))
	want := posemodel.ChangeSet{ID: "cs-" + hex.EncodeToString(id[:6]), Spec: "sample", Selector: "trailers:sample", Base: first + "^", Head: last, ResolvedBase: base, ResolvedHead: last,
		Commits: []string{first, last}, Paths: []posemodel.ObservedPath{{Action: "created", Path: "new.txt"}, {Action: "modified", Path: "base.txt"}, {Action: "removed", Path: "remove.txt"}}, DiffDigest: "sha256:" + digest}
	got, err := resolveGitChangeSet(root, "sample", "", "")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("historical attribution changed:\n got=%+v\nwant=%+v\nerr=%v", got, want, err)
	}
}

func TestGovernanceOperationPerformanceGitPathBytes(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.name", "Fixture")
	artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
	artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
	names := []string{"tab\tname.txt", "line\nname.txt", "café space.txt", "back\\slash.txt"}
	want := []posemodel.ObservedPath{}
	for _, name := range names {
		writeArtifactTestFile(t, root, name, "literal path\n")
		want = append(want, posemodel.ObservedPath{Action: "created", Path: name})
	}
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "paths\n\nPOSE-Spec: sample")
	got, err := resolveGitChangeSet(root, "sample", "", "")
	sort.Slice(want, func(i, j int) bool { return observedSortKey(want[i]) < observedSortKey(want[j]) })
	if err != nil || !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("Git path bytes changed: got=%+v want=%+v err=%v", got.Paths, want, err)
	}
	base := artifactGit(t, root, "rev-parse", "HEAD")
	artifactGit(t, root, "mv", names[0], "renamed\nnew.txt")
	artifactGit(t, root, "commit", "-q", "-m", "rename")
	head := artifactGit(t, root, "rev-parse", "HEAD")
	renamed, err := resolveGitChangeSet(root, "sample", base, head)
	wantRename := []posemodel.ObservedPath{{Action: "renamed", OldPath: names[0], NewPath: "renamed\nnew.txt"}}
	if err != nil || !reflect.DeepEqual(renamed.Paths, wantRename) {
		t.Fatalf("range rename lost literal paths: %+v %v", renamed, err)
	}
}

func TestGovernanceOperationPerformanceRejectsMalformedPathFraming(t *testing.T) {
	for _, raw := range [][]byte{[]byte("A\x00x"), []byte("\x00x\x00"), []byte("A\x00\x00"), []byte("R100\x00old\x00"), []byte("R101\x00old\x00new\x00"), []byte("R+1\x00old\x00new\x00"), []byte("R0000\x00old\x00new\x00"), []byte("C100\x00old\x00new\x00"), []byte("A\x00x\x00junk\x00"), {'A', 0, 0xff, 0}} {
		if paths, err := parseGitNameStatusNUL(raw); err == nil || paths != nil {
			t.Fatalf("malformed/unsupported record accepted: %q %+v %v", raw, paths, err)
		}
	}
	if paths, err := parseGitNameStatusNUL(nil); err != nil || len(paths) != 0 {
		t.Fatalf("empty diff rejected: %+v %v", paths, err)
	}
}

func TestGovernanceOperationPerformanceBenchmark(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	_, source, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(source), "../../..", "scripts", "bench_governance.py")
	for _, outcome := range []string{"success", "failure", "timeout", "version-timeout", "setup-timeout", "setup-error", "missing-matrix", "shared"} {
		t.Run(outcome, func(t *testing.T) {
			wantValid := outcome == "success" || outcome == "shared"
			root := t.TempDir()
			artifactGit(t, root, "init", "-q")
			artifactGit(t, root, "config", "user.name", "Fixture")
			artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
			artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
			if outcome != "missing-matrix" {
				writeArtifactTestFile(t, root, ".pose/indexes/validation-matrix.json", "{}\n")
			}
			writeArtifactTestFile(t, root, ".pose/state/owned", "committed\n")
			artifactGit(t, root, "add", ".")
			artifactGit(t, root, "commit", "-q", "-m", "synthetic corpus")
			provider := ""
			if outcome == "shared" {
				provider = root
				root = filepath.Join(t.TempDir(), "shared-source")
				artifactGit(t, provider, "clone", "--shared", "-q", provider, root)
			}
			writeArtifactTestFile(t, root, ".pose/state/owned", "operator change\n")
			// An ambient global post-checkout hook must not escape isolation.
			xdg, hooks := t.TempDir(), t.TempDir()
			if err := os.MkdirAll(filepath.Join(xdg, "git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(xdg, "git/config"), []byte("[core]\n hooksPath = "+hooks+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			hook := "#!/bin/sh\nprintf 'hook changed source\\n' > " + shellQuote(filepath.Join(root, ".pose/state/owned")) + "\n"
			if err := os.WriteFile(filepath.Join(hooks, "post-checkout"), []byte(hook), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_CONFIG_HOME", xdg)
			if strings.HasPrefix(outcome, "setup-") {
				realGit, err := exec.LookPath("git")
				if err != nil {
					t.Fatal(err)
				}
				bin := t.TempDir()
				failure := "echo clone failure >&2; exit 9"
				if outcome == "setup-timeout" {
					failure = "echo clone started >&2; exec sleep 30"
				}
				wrapper := "#!/bin/sh\nfor arg do\n if [ \"$arg\" = clone ]; then " + failure + "; fi\ndone\nexec " + shellQuote(realGit) + " \"$@\"\n"
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(wrapper), 0o700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			binary := filepath.Join(t.TempDir(), "fake-pose")
			behavior := "exit 0\n"
			if outcome == "failure" {
				behavior = "echo rejected >&2\nexit 7\n"
			}
			if outcome == "timeout" {
				behavior = "sleep 30\n"
			}
			version := "echo fixture; exit 0"
			if outcome == "version-timeout" {
				version = "sleep 30; exit 0"
			}
			fake := "#!/bin/sh\nif [ \"$1\" = version ]; then " + version + "; fi\nprintf 'generated\\n' > .pose/state/owned\n" + behavior
			if err := os.WriteFile(binary, []byte(fake), 0o700); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(t.TempDir(), "report")
			cmd := exec.Command("python3", script, "--root", root, "--binary", binary, "--runs", "1", "--operation", "index", "--timeout", "2", "--output-dir", output)
			log, runErr := cmd.CombinedOutput()
			if (runErr == nil) != wantValid {
				t.Fatalf("unexpected exit %v: %s", runErr, log)
			}
			data, err := os.ReadFile(filepath.Join(output, "report.json"))
			if err != nil {
				t.Fatalf("missing report: %v: %s", err, log)
			}
			var report struct {
				VersionTimeout bool `json:"engine_version_timeout"`
				Setup          []struct {
					Phase    string
					ExitCode int `json:"exit_code"`
					Timeout  bool
					Stderr   string
				}
				Samples []struct {
					Valid    bool
					Timeout  bool
					ExitCode int `json:"exit_code"`
				}
				Summary map[string]struct {
					ValidSamples int      `json:"valid_samples"`
					Median       *float64 `json:"median_ms"`
				}
			}
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			owned, err := os.ReadFile(filepath.Join(root, ".pose/state/owned"))
			if err != nil || string(owned) != "operator change\n" {
				t.Fatalf("operator files changed: %q %v", owned, err)
			}
			if strings.HasPrefix(outcome, "setup-") || outcome == "missing-matrix" {
				if len(report.Samples) != 0 || len(report.Setup) < 1 {
					t.Fatalf("invalid setup became measured operation: %s", data)
				}
				setup := report.Setup[len(report.Setup)-1]
				phase := "clone"
				if outcome == "missing-matrix" {
					phase = "matrix"
				}
				if setup.Phase != phase || setup.Timeout != (outcome == "setup-timeout") || (outcome == "setup-error" && setup.ExitCode != 9) || (outcome == "missing-matrix" && setup.ExitCode != 128) {
					t.Fatalf("lost setup error: %s", data)
				}
				raw, err := os.ReadFile(setup.Stderr)
				if err != nil || len(raw) == 0 {
					t.Fatalf("lost raw setup stderr: %q %v", raw, err)
				}
				return
			}
			if len(report.Samples) != 1 {
				t.Fatalf("missing raw sample: %s", data)
			}
			sample := report.Samples[0]
			if report.VersionTimeout != (outcome == "version-timeout") {
				t.Fatalf("lost version timeout: %s", data)
			}
			if sample.Valid != wantValid || sample.Timeout != (outcome == "timeout") {
				t.Fatalf("wrong sample disposition: %s", data)
			}
			if outcome == "failure" && sample.ExitCode != 7 {
				t.Fatalf("lost exit code: %s", data)
			}
			if (report.Summary["index"].Median != nil) != wantValid {
				t.Fatalf("invalid sample included in median: %s", data)
			}
			if provider != "" {
				if _, err := os.Stat(filepath.Join(output, "corpus.git/objects/info/alternates")); !os.IsNotExist(err) {
					t.Fatalf("mirror still borrows objects: %v", err)
				}
				if err := os.Rename(provider, filepath.Join(t.TempDir(), "moved-provider")); err != nil {
					t.Fatal(err)
				}
				artifactGit(t, filepath.Join(output, "corpus.git"), "cat-file", "-e", "HEAD")
			}
		})
	}
}

func TestGovernanceOperationPerformanceGitFailuresAndRefChanges(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"head", "other-ref", "partial-log"} {
		t.Run(outcome, func(t *testing.T) {
			root := t.TempDir()
			artifactGit(t, root, "init", "-q")
			artifactGit(t, root, "config", "user.name", "Fixture")
			artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
			artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
			writeArtifactTestFile(t, root, ".pose/specs/sample.md", "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n\n## 3. Technical Plan\n\n### Artifacts\n- none: analysis\n")
			artifactGit(t, root, "add", ".")
			artifactGit(t, root, "commit", "-q", "-m", "initial\n\nPOSE-Spec: sample")
			post := "exit 7\n"
			if outcome == "head" {
				post = shellQuote(realGit) + " -C " + shellQuote(root) + " -c core.hooksPath=/dev/null commit --allow-empty -q -m 'later\n\nPOSE-Spec: sample'\nexit 0\n"
			}
			if outcome == "other-ref" {
				post = shellQuote(realGit) + " -C " + shellQuote(root) + " update-ref refs/heads/later HEAD\nexit 0\n"
			}
			wrapper := "#!/bin/sh\nfor arg do\n if [ \"$arg\" = '--grep=POSE-Spec:' ]; then\n" + shellQuote(realGit) + " \"$@\"\n" + post + " fi\ndone\nexec " + shellQuote(realGit) + " \"$@\"\n"
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "git"), []byte(wrapper), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			_, _, sets, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
			if outcome == "partial-log" {
				if err != nil || gaps[""] == "" || len(sets) != 0 {
					t.Fatalf("partial Git output accepted: sets=%v gaps=%v err=%v", sets, gaps, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "Git inputs changed") {
				t.Fatalf("changed refs accepted: %v %v", gaps, err)
			}
		})
	}
}

func TestGovernanceOperationPerformanceGitTimeoutAndByteLimit(t *testing.T) {
	bin := t.TempDir()
	path := filepath.Join(bin, "git")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	out, err := gitOutputBoundedContext(ctx, t.TempDir(), 10, "log")
	if !errors.Is(err, context.DeadlineExceeded) || out != nil {
		t.Fatalf("Git timeout became output: %q %v", out, err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf 'too many bytes for this limit'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	out, err = gitOutputBounded(t.TempDir(), 5, "log")
	if err == nil || out != nil {
		t.Fatalf("bounded partial output became success: %q %v", out, err)
	}
}

func TestGovernanceOperationPerformanceTrackedPathFraming(t *testing.T) {
	bin := t.TempDir()
	path := filepath.Join(bin, "git")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, fixture := range []struct {
		name, program string
		want          []string
		valid         bool
	}{
		{"empty", "exit 0", []string{}, true},
		{"complete", "printf 'one\\000two\\000'", []string{"one", "two"}, true},
		{"partial", "printf 'complete\\000partial'", nil, false},
		{"empty-record", "printf 'complete\\000\\000'", nil, false},
		{"invalid-utf8", "printf '\\377\\000'", nil, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+fixture.program+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			got, err := gitTrackedPaths(t.TempDir())
			if (err == nil) != fixture.valid || !reflect.DeepEqual(got, fixture.want) {
				t.Fatalf("framing disposition changed: got=%q err=%v want=%q valid=%v", got, err, fixture.want, fixture.valid)
			}
		})
	}
}

type governanceDisappearingCatalogueFiles struct {
	posemodel.Files
	root  string
	reads int
}

func (f *governanceDisappearingCatalogueFiles) ReadFile(path string) ([]byte, error) {
	raw, err := f.Files.ReadFile(path)
	if filepath.ToSlash(path) == filepath.ToSlash(filepath.Join(f.root, ".pose/specs/sample.md")) {
		f.reads++
		if f.reads == 2 {
			if removeErr := os.RemoveAll(filepath.Join(f.root, ".pose/specs")); removeErr != nil {
				return nil, removeErr
			}
		}
	}
	return raw, err
}

func TestGovernanceOperationPerformanceDisappearingCatalogue(t *testing.T) {
	root := t.TempDir()
	writeArtifactTestFile(t, root, ".pose/specs/sample.md", "---\nslug: sample\nstatus: draft\n---\n\n# Sample\n\n## 3. Technical Plan\n\n### Artifacts\n- created: sample.txt\n")
	files := &governanceDisappearingCatalogueFiles{Files: posemodel.NativeFiles(), root: root}
	specs, claims, sets, _, _, _, err := collectArtifactGraphInputsWithGapsIn(files, root)
	if err == nil || len(specs) != 0 || len(claims) != 0 || len(sets) != 0 {
		t.Fatalf("partial catalogue accepted: specs=%v claims=%v sets=%v err=%v", specs, claims, sets, err)
	}
}

func TestGovernanceOperationPerformanceInitiallyAbsentCatalogue(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.name", "Fixture")
	artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
	artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
	writeArtifactTestFile(t, root, "readme.txt", "no spec catalogue\n")
	artifactGit(t, root, "add", ".")
	artifactGit(t, root, "commit", "-q", "-m", "fixture")
	specs, claims, sets, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
	if err != nil || len(specs) != 0 || len(claims) != 0 || len(sets) != 0 || len(gaps) != 0 {
		t.Fatalf("initial absence became conflict: %v %v %v %v %v", specs, claims, sets, gaps, err)
	}
}

type governanceMutatingHeaderFiles struct {
	posemodel.Files
	path    string
	once    sync.Once
	changed bool
}

func (f *governanceMutatingHeaderFiles) ReadFile(path string) ([]byte, error) {
	raw, err := f.Files.ReadFile(path)
	if err == nil && path == f.path {
		f.once.Do(func() {
			if writeErr := f.Files.WriteFile(path, bytes.ReplaceAll(raw, []byte("other"), []byte("owned")), 0o644); writeErr == nil {
				f.changed = true
			}
		})
	}
	return raw, err
}

func TestGovernanceOperationPerformanceCheckFinalizesReviewView(t *testing.T) {
	root := closeoutCLIFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","review_bundles":true,"review_bundles_adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`)
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte("status: in-progress"), []byte("status: done"))
	raw = bytes.ReplaceAll(raw, []byte("completed_at:\n"), []byte("completed_at: 2026-11-03\n"))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	header := filepath.Join(root, ".pose/review-bundles/rvb-0000000000000000.json")
	writeCloseoutCLIFile(t, root, ".pose/review-bundles/rvb-0000000000000000.json", `{"payload":{"scope":{"ref":"spec:other"}}}`)
	files := &governanceMutatingHeaderFiles{Files: posemodel.NativeFiles(), path: header}
	var stdout, stderr bytes.Buffer
	checker := &nativeChecker{root: root, files: files, mode: "strict", stdout: &stdout, out: render(&stdout, &stderr)}
	checker.checkReviewCloseout()
	if !files.changed || checker.errors == 0 || !strings.Contains(stdout.String()+stderr.String(), "review inputs changed during operation") {
		t.Fatalf("native check ignored final view audit: changed=%v errors=%d out=%s err=%s", files.changed, checker.errors, stdout.String(), stderr.String())
	}
}
