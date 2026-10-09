package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// An invented test: ref blocks a spec that can still change and only warns on
// a closed one (spec pose-trace-test-refs-resolve).
func TestTraceTestRefsBlockAnOpenSpecAndWarnOnAClosedOne(t *testing.T) {
	root := closeoutCLIFixture(t)
	writeCloseoutCLIFile(t, root, "api/digest_test.go", "package api\n\nfunc TestDigestStable(t *testing.T) {}\n")
	body := "# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestDigestStableForever\n"
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n"+body)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "fixture"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	spec := filepath.Join(root, ".pose/specs/alpha/spec.md")
	var out, errOut bytes.Buffer
	if code := cmdLintSpecInRoot(root, []string{"alpha"}, &out, &errOut); code == 0 || !strings.Contains(out.String()+errOut.String(), "test:TestDigestStableForever names no test") {
		t.Fatalf("lint accepted an invented test ref on an open spec: code=%d %s%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	_ = cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut)
	out.Reset()
	errOut.Reset()
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "names no test") {
		t.Fatalf("close accepted an invented test ref: code=%d %s", code, errOut.String())
	}

	// The same ref on a closed spec is reported, not failed.
	raw, _ := os.ReadFile(spec)
	closed := strings.Replace(strings.Replace(string(raw), "status: in-progress", "status: done", 1), "completed_at:", "completed_at: 2026-08-03", 1)
	if err := os.WriteFile(spec, []byte(closed), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	cmdLintSpecInRoot(root, []string{"alpha"}, &out, &errOut)
	text := out.String() + errOut.String()
	if !strings.Contains(text, "names no test") || strings.Contains(text, "✖ error requirement-trace") {
		t.Fatalf("a closed spec's unresolved ref is not a warning: %s", text)
	}

	// A resolvable ref closes.
	fixed := strings.Replace(string(raw), "TestDigestStableForever", "TestDigestStable", 1)
	if err := os.WriteFile(spec, []byte(fixed), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdLintSpecInRoot(root, []string{"alpha"}, &out, &errOut); strings.Contains(out.String()+errOut.String(), "names no test") {
		t.Fatalf("a resolvable ref was reported: code=%d %s%s", code, out.String(), errOut.String())
	}
}
