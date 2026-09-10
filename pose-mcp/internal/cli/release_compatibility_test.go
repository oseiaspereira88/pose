// Cross-version policy compatibility (spec pose-release-compatibility-rehearsal).
//
// v2.0.0 wrote `contract_adoptions` into an instance's review policy and the
// v1.8.1 binary could no longer read it: its decoder used
// DisallowUnknownFields, so one added key made the whole policy invalid. The
// engine that introduced the key had no way to notice — it reads its own
// output fine. What was missing was the other half of the pair.
//
// This runs it: the previous release is built from its own tag and pointed at
// a policy this engine writes. It is a real binary, not a recorded key list,
// because a recorded list is a second source of truth whose only failure mode
// is being out of date on exactly the release that needed it.
//
// The probe is `review-plan`, not `check --strict`. `check` reads the policy
// through a four-field struct of its own and never reaches the loader that
// enforces the schema — against v1.8.1 it reports SUCCESS on a policy carrying
// an outright unknown key. `review-plan` goes through loadReviewPolicy, which
// is where the regression lived, and there v1.8.1 does reject what this engine
// writes: `unknown field "evidence_vocabulary_reconciled_at"`.

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/version"
)

func TestPreviousReleaseReadsThisEnginesReviewPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the previous release from source")
	}

	root := distributionRoot(t)
	tag := previousReleaseTag(t, root)

	prev := buildReleaseFromTag(t, root, tag)
	instance := instanceWrittenByThisEngine(t)
	policy := filepath.Join(instance, ".pose", "policy", "review.json")

	// The control comes first. If the probe did not reach the policy loader,
	// the real assertion below would pass for the wrong reason — which is how
	// `check --strict` fooled this comparison once already, and how the same
	// compatibility claim got measured in an instance that never loaded the
	// file at all.
	original, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	writePolicyWith(t, policy, func(p map[string]any) { p["enabled"] = "yes" })
	if out := runRelease(t, prev, instance); !mentionsPolicy(out) {
		t.Fatalf("%s accepted a policy whose `enabled` is a string, so this probe is not reaching its policy loader:\n%s", tag, out)
	}
	if err := os.WriteFile(policy, original, 0o644); err != nil {
		t.Fatal(err)
	}

	if out := runRelease(t, prev, instance); mentionsPolicy(out) {
		t.Fatalf("%s cannot read the review policy v%s writes, so an instance updated to v%s stops working for anyone still on %s:\n%s",
			tag, version.ReleaseBase(), version.ReleaseBase(), tag, out)
	}
}

// mentionsPolicy reports whether the older binary failed on the policy itself,
// as opposed to on the deliberately absent scope the probe names.
func mentionsPolicy(out string) bool {
	return strings.Contains(out, "review policy")
}

// ReleaseHistoryPromised is the environment variable a workflow sets to say the
// checkout it prepared carries this engine's release history — that is, that it
// asked for full history and the tags are there.
//
// It is declared by the workflow rather than inferred from the provider. The
// first version of this guard keyed on `CI`, which broke every repository that
// vendors the engine and runs its suite: those are CI too, their submodule
// checkouts have no tags, and there was nothing for them to configure. The
// second keyed on `GITHUB_REPOSITORY`, which is right but is the provider's to
// set — on a provider that does not, this repository's own CI would skip in
// silence, which is the hole the guard exists to close.
//
// A variable this repository's workflows declare has neither problem, and a
// workflow contract test asserts every job that runs the suite declares it
// (spec pose-the-guard-signal-is-declared-not-inherited).
const ReleaseHistoryPromised = "POSE_RELEASE_HISTORY_AVAILABLE"

// skipUnlessCI fails where the release history was promised, and skips
// everywhere else.
func skipUnlessCI(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv(ReleaseHistoryPromised) != "" {
		t.Fatalf("this workflow declares "+ReleaseHistoryPromised+", so a missing tag is a configuration failure rather than a missing precondition: "+format, args...)
	}
	t.Skipf(format, args...)
}

// distributionRoot is the repository root, three levels up from this package.
func distributionRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "compatibility.json")); err != nil {
		t.Skipf("distribution root not found at %s", root)
	}
	return root
}

// previousReleaseTag is the highest tag strictly below the version this binary
// carries — the release an instance is most likely still running when it meets
// a policy written by this one.
func previousReleaseTag(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "tag", "--list", "v*").Output()
	if err != nil {
		t.Skipf("git tags unavailable: %v", err)
	}
	current := parseSemver(version.ReleaseBase())
	if current == nil {
		t.Skipf("this binary's version is not a release version: %s", version.Version)
	}
	best, bestParsed := "", []int(nil)
	for _, line := range strings.Fields(string(out)) {
		parsed := parseSemver(strings.TrimPrefix(line, "v"))
		if parsed == nil || !semverLess(parsed, current) {
			continue
		}
		if bestParsed == nil || semverLess(bestParsed, parsed) {
			best, bestParsed = line, parsed
		}
	}
	if best == "" {
		skipUnlessCI(t, "no tag below v%s is present; a shallow clone has none", version.ReleaseBase())
	}
	return best
}

func parseSemver(s string) []int {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

func semverLess(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// buildReleaseFromTag builds the pose binary from a tag's own sources. It uses
// `git archive` rather than a worktree so the test never touches the checkout
// it is running in, and it extracts the whole distribution because pose-mcp
// resolves mcp-enforce through a relative replace directive.
func buildReleaseFromTag(t *testing.T, root, tag string) string {
	t.Helper()
	src := t.TempDir()
	archive := exec.Command("git", "-C", root, "archive", tag)
	var archiveErr bytes.Buffer
	archive.Stderr = &archiveErr
	extract := exec.Command("tar", "-x", "-C", src)
	pipe, err := archive.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	extract.Stdin = pipe
	var extractErr bytes.Buffer
	extract.Stderr = &extractErr
	if err := extract.Start(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Start(); err != nil {
		t.Fatal(err)
	}
	archiveWait := archive.Wait()
	extractWait := extract.Wait()
	if archiveWait != nil {
		// A shallow clone has no tag objects; that is a missing precondition,
		// not a failure of the contract under test.
		skipUnlessCI(t, "git archive %s unavailable (%v): %s", tag, archiveWait, archiveErr.String())
	}
	if extractWait != nil {
		t.Fatalf("extracting %s: %v\n%s", tag, extractWait, extractErr.String())
	}

	module := filepath.Join(src, "pose-mcp")
	if _, err := os.Stat(filepath.Join(module, "go.mod")); err != nil {
		t.Skipf("%s predates the current module layout", tag)
	}
	bin := filepath.Join(t.TempDir(), "pose-"+strings.TrimPrefix(tag, "v"))
	build := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "./cmd/pose")
	build.Dir = module
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building %s: %v\n%s", tag, err, out)
	}
	return bin
}

// instanceWrittenByThisEngine installs a fresh instance with the engine under
// test, in process, so the policy under examination is the one this build
// actually produces.
func instanceWrittenByThisEngine(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if code := cmdInstall([]string{dir}, io_Discard{}, io_Discard{}); code != 0 {
		t.Fatalf("pose install exited %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, ".pose", "policy", "review.json")); err != nil {
		t.Fatalf("this engine wrote no review policy to compare: %v", err)
	}
	return dir
}

type io_Discard struct{}

func (io_Discard) Write(p []byte) (int, error) { return len(p), nil }

func writePolicyWith(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	mutate(policy)
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// runRelease asks the older binary to plan a review for a scope that does not
// exist. Resolving the scope is downstream of loading the policy, so the
// command reaches the loader on an instance with no specs and then fails on the
// missing spec — a failure with a distinct message, which is what lets the
// assertion tell "cannot read your policy" from "no such scope".
func runRelease(t *testing.T, bin, instance string) string {
	t.Helper()
	cmd := exec.Command(bin, "review-plan", "spec:no-such-scope-for-compatibility-probe")
	cmd.Dir = instance
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	if _, ok := err.(*exec.ExitError); !ok && err != nil {
		t.Fatalf("running %s: %v", bin, err)
	}
	return string(out)
}
