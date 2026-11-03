package cli

import (
	"bytes"
	"context"
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
	// Exceeding the preparation budget is still a valid input, not a truncated
	// or rejected catalogue. One member needs no pairwise comparison.
	large := []followup{{Spec: "large", Text: strings.Repeat("x", 1024*1024+1)}}
	if got := clusterFollowups(large, 0.6); len(got) != 0 {
		t.Fatal("invented candidate")
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

func TestGovernanceOperationPerformanceBenchmark(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	_, source, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(source), "../../..", "scripts", "bench_governance.py")
	for _, outcome := range []string{"success", "failure", "timeout", "version-timeout", "setup-timeout", "setup-error"} {
		t.Run(outcome, func(t *testing.T) {
			root := t.TempDir()
			artifactGit(t, root, "init", "-q")
			artifactGit(t, root, "config", "user.name", "Fixture")
			artifactGit(t, root, "config", "user.email", "fixture@example.invalid")
			artifactGit(t, root, "config", "core.hooksPath", "/dev/null")
			writeArtifactTestFile(t, root, ".pose/indexes/validation-matrix.json", "{}\n")
			writeArtifactTestFile(t, root, ".pose/state/owned", "committed\n")
			artifactGit(t, root, "add", ".")
			artifactGit(t, root, "commit", "-q", "-m", "synthetic corpus")
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
			if (runErr == nil) != (outcome == "success") {
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
			if strings.HasPrefix(outcome, "setup-") {
				if len(report.Samples) != 0 || len(report.Setup) < 1 {
					t.Fatalf("invalid setup became measured operation: %s", data)
				}
				setup := report.Setup[len(report.Setup)-1]
				if setup.Phase != "clone" || setup.Timeout != (outcome == "setup-timeout") || (outcome == "setup-error" && setup.ExitCode != 9) {
					t.Fatalf("lost setup error: %s", data)
				}
				raw, err := os.ReadFile(setup.Stderr)
				if err != nil || !bytes.Contains(raw, []byte("clone")) {
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
			if sample.Valid != (outcome == "success") || sample.Timeout != (outcome == "timeout") {
				t.Fatalf("wrong sample disposition: %s", data)
			}
			if outcome == "failure" && sample.ExitCode != 7 {
				t.Fatalf("lost exit code: %s", data)
			}
			if (report.Summary["index"].Median != nil) != (outcome == "success") {
				t.Fatalf("invalid sample included in median: %s", data)
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
