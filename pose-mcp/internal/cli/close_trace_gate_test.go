package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-quickstart-real-lifecycle: `pose close` holds the exit gate it
// documents — a done spec points every promise at evidence.

func TestCloseTraceGateRefusesAnIncompleteTrace(t *testing.T) {
	cases := map[string]struct {
		body string
		want string
	}{
		"no trace section": {"## 2. Requirements\n- R1: works\n", "no `### Requirement trace` subsection"},
		"untraced R2":      {"## 2. Requirements\n- R1: works\n- R2: also works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n", "R2 has no trace entry"},
		"orphan entry":     {"## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n- R9 [satisfied] test:TestOther\n", "R9 is traced but not declared"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := closeoutCLIFixture(t)
			writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n"+tc.body)
			var out, errOut bytes.Buffer
			if code := cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut); code != 0 {
				t.Fatalf("review: %s", errOut.String())
			}
			out.Reset()
			errOut.Reset()
			code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut)
			if code == 0 || !strings.Contains(errOut.String(), "requirement trace gate failed") || !strings.Contains(errOut.String(), tc.want) || !strings.Contains(errOut.String(), "[satisfied] test:<TestName>") {
				t.Fatalf("close accepted an incomplete trace: code=%d err=%s", code, errOut.String())
			}
			raw, _ := os.ReadFile(filepath.Join(root, ".pose/specs/alpha/spec.md"))
			if !strings.Contains(string(raw), "status: in-progress") {
				t.Fatal("a refused close changed the spec")
			}
		})
	}
	// A complete trace closes.
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	_ = cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut)
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("a traced spec was refused: %s", errOut.String())
	}
	if state, _ := (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha"); !state.Terminal {
		t.Fatal("a traced spec did not close")
	}
}

func TestCloseTraceGateSpecTemplateDeclaresNoExampleTarget(t *testing.T) {
	for _, rel := range []string{"../../../.pose/templates/spec.md", "../../../locales/pt-BR/.pose/templates/spec.md", "../scaffold/dist/.pose/templates/spec.md", "../scaffold/dist/locales/pt-BR/.pose/templates/spec.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		section := string(raw)
		start := strings.Index(section, "### Delivery targets")
		end := strings.Index(section[start+1:], "\n### ")
		body := section[start : start+1+end]
		// No line is a list item: the delivery-target parser reads "- "
		// lines even inside an HTML comment.
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				t.Fatalf("%s carries a delivery-target list item a scaffolded spec would declare:\n%s", rel, body)
			}
		}
		// Outside the comment, the section declares nothing.
		withoutComment := body
		for {
			i := strings.Index(withoutComment, "<!--")
			if i < 0 {
				break
			}
			j := strings.Index(withoutComment[i:], "-->")
			if j < 0 {
				break
			}
			withoutComment = withoutComment[:i] + withoutComment[i+j+3:]
		}
		if strings.Contains(withoutComment, "surface:") {
			t.Fatalf("%s declares an example delivery target a scaffolded spec would carry into closeout:\n%s", rel, body)
		}
	}
}

func TestCloseTraceGateBlocksThePlanBeforeSealing(t *testing.T) {
	root := newGitRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n")
	gitCommitAll(t, root, "alpha")
	plan, err := posemodel.Store{Root: root}.PlanCloseout("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	next, _ := plan.NextCloseoutStep()
	if next.ID != posemodel.CloseoutStepTrace || next.State != posemodel.CloseoutStateBlocked || !strings.Contains(next.Reason, "R1 has no trace entry") {
		t.Fatalf("the plan does not stop at the trace before sealing: %+v", plan.Steps)
	}
	for _, step := range plan.Steps {
		if step.ID == posemodel.CloseoutStepSeal {
			t.Fatalf("a plan with an incomplete trace reaches sealing: %+v", plan.Steps)
		}
	}
}
