package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// The verdict line is what a reviewer, a CI log and a spec's execution log quote.
// A strict run that found only non-escalating warnings printed
// `(tolerant mode) with N warning(s)`, which invites two opposite errors: recording
// a strict pass that was never claimed, or re-running the gate believing --strict
// was dropped. Measured during the pose-abm-progressive-review closeout, where the
// first reading of the label was itself wrong.
func checkVerdictFixture(t *testing.T) string {
	t.Helper()
	root := newGitRepo(t)
	var out, errOut bytes.Buffer
	if code := Main([]string{"install", root, "--skip-mcp"}, &out, &errOut); code != 0 {
		t.Fatalf("install code=%d err=%s", code, errOut.String())
	}
	// A done spec with no changelog fragment is a warning in both modes: it is
	// raised through the warn path, not failOrWarn, so it does not escalate under
	// --strict. That is exactly the population whose verdict was mislabelled.
	//
	// completed_at is the install day on purpose. Install stamps the changelog
	// adoption with today, and a spec completed before it is exempt — a fixture
	// dated in the past produced no warning at all, which is how this generator
	// was chosen: by measuring it rather than assuming it.
	writeCloseoutCLIFile(t, root, ".pose/specs/warned.md",
		"---\nslug: warned\nstatus: done\ncreated_at: 2026-09-01\ncompleted_at: "+
			time.Now().UTC().Format(time.DateOnly)+"\n---\n"+
			"# Spec\n\n## 2. Requirements\n\n- R1: exist.\n")
	return root
}

func runCheckVerdict(t *testing.T, root string, args ...string) (string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := 0
	inDir(t, root, func() {
		code = Main(append([]string{"check"}, args...), &out, &errOut)
	})
	return out.String() + errOut.String(), code
}

func TestCheckVerdictModeNamesTheRunItWas(t *testing.T) {
	root := checkVerdictFixture(t)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"strict", []string{"--strict"}, "strict"},
		{"tolerant", []string{"--tolerant"}, "tolerant"},
		{"default", nil, "strict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text, _ := runCheckVerdict(t, root, tc.args...)
			if !strings.Contains(text, "warning(s)") {
				t.Skipf("fixture produced no warning-only verdict to label:\n%s", text)
			}
			if !strings.Contains(text, "("+tc.want+" mode)") {
				t.Errorf("a %s run does not name its own mode:\n%s", tc.name, text)
			}
			for _, other := range []string{"strict", "tolerant"} {
				if other != tc.want && strings.Contains(text, "("+other+" mode) with") {
					t.Errorf("a %s run reported itself as %s:\n%s", tc.name, other, text)
				}
			}
		})
	}
}

// The escalation contract is untouched: --strict still turns a failOrWarn finding
// into an error, and a warning-only run still exits 0 in every mode. Without this
// the label fix could quietly become a gate change.
func TestCheckVerdictModeKeepsEscalation(t *testing.T) {
	root := checkVerdictFixture(t)
	for _, args := range [][]string{{"--strict"}, {"--tolerant"}, nil} {
		text, code := runCheckVerdict(t, root, args...)
		if code != 0 {
			t.Fatalf("a warning-only run exited %d for args %v:\n%s", code, args, text)
		}
	}
	// A failOrWarn finding is the one that escalates: a malformed validation
	// matrix is a warning under --tolerant and an error under --strict. This is
	// the contract the label fix must not touch, and measuring it is also what
	// corrected my own first reading — that --strict was being ignored. It is
	// honored; only the verdict text was wrong.
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", "{ this is not json\n")
	strictText, strictCode := runCheckVerdict(t, root, "--strict")
	if strictCode != 1 || !strings.Contains(strictText, "error(s)") {
		t.Fatalf("--strict did not escalate a failOrWarn finding (exit=%d):\n%s", strictCode, strictText)
	}
	tolerantText, tolerantCode := runCheckVerdict(t, root, "--tolerant")
	if tolerantCode != 0 || strings.Contains(tolerantText, "error(s)") {
		t.Fatalf("--tolerant escalated a finding it should only warn about (exit=%d):\n%s", tolerantCode, tolerantText)
	}
}
