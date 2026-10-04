package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Cases for the gates added later in the agency-readiness program (spec
// pose-mechanization-adversarial-corpus R5): each is the cheapest formal
// satisfaction of one gate, added before the gate's spec closes.
func init() {
	adversarialCorpus = append(adversarialCorpus,
		adversarialCase{"stale-preview-authorizes-apply", "enforced", "pose-governed-effect-enforcement", func(t *testing.T) (bool, string) {
			// A start previewed before a restricting request is applied after it.
			root := actionCLIFixture(t)
			spec := filepath.Join(root, ".pose/specs/2026-10-04-storage.md")
			raw, _ := os.ReadFile(spec)
			mustWrite(t, spec, strings.Replace(string(raw), "status: in-progress", "status: draft", 1))
			mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"},"agency_readiness_version":1,"atomic_start_version":1}`)
			s := posemodel.Store{Root: root}
			plan, err := s.PreviewStart("storage")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.OpenActionRequest(posemodel.ActionRequest{Origin: "spec:storage", Kind: posemodel.ActionApproval, Question: "Start now?",
				RequestedBy: posemodel.ActionPrincipal{Principal: "agent:a"}, Recipient: posemodel.ObligationActor{Role: "maintainer"},
				Targets: []posemodel.NodeRef{{Artifact: "self"}}, Effects: []posemodel.ObligationEffect{{Phase: posemodel.PhaseStart, Mode: posemodel.EffectBlock}}}, time.Now()); err != nil {
				t.Fatal(err)
			}
			_, err = s.ApplyStart(plan, plan.Digest, nil)
			return err == nil, "apply after a newer restricting request"
		}},
		adversarialCase{"code-change-keeps-judgment", "enforced", "pose-material-equivalence-reuse", func(t *testing.T) (bool, string) {
			// After the reviewed code changes, every criterion still reads as
			// equivalent, so a sealed judgment would be reused unchanged.
			root := reviewBundleCLIFixture(t)
			s := posemodel.Store{Root: root}
			sealed, err := s.SealReviewBundle("spec:bundle", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			files, _ := filepath.Glob(filepath.Join(root, "pose-mcp", "*.go"))
			if len(files) == 0 {
				t.Fatal("fixture has no subject file")
			}
			raw, _ := os.ReadFile(files[0])
			_ = os.WriteFile(files[0], append(raw, []byte("\nfunc Changed() {}\n")...), 0o644)
			changed, err := s.PrepareReviewBundle("spec:bundle")
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range posemodel.ExplainCriterionReuse(sealed, changed) {
				if e.Decision == posemodel.ReuseChanged {
					return false, "criterion " + e.Criterion + " names the changed input"
				}
			}
			return true, "no criterion changed after a code change"
		}},
		adversarialCase{"minimal-surface-skips-trace", "enforced", "pose-progressive-spec-surface", func(t *testing.T) (bool, string) {
			// A done spec declares the minimal surface to drop its requirement trace.
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-small.md"), "---\nslug: small\nstatus: done\ncreated_at: 2026-10-04\ncompleted_at: 2026-10-04\nsurface: minimal\n---\n\n# Spec: small\n\n## 1. Intent\nFix a typo.\n\n## 2. Requirements\n- R1: The word is spelled right.\n\n## 3. Technical Plan\n### Artifacts\n- modified: README.md\n\n## 6. Validation\nRan it.\n\n## 7. Final Report\nDone.\n")
			code, out := runPose(t, root, "lint-spec", "small", "--strict")
			return code == 0, out
		}},
		adversarialCase{"stale-assessment-reused", "enforced", "pose-adaptive-assessment-freshness", func(t *testing.T) (bool, string) {
			// The assessed component changes and the old assessment still reads as fresh.
			root := t.TempDir()
			git := func(args ...string) {
				cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.email=c@x", "-c", "user.name=corpus"}, args...)...)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %s", args, out)
				}
			}
			git("init", "-q")
			mustWrite(t, filepath.Join(root, "svc/main.go"), "package main\n\nfunc main() {}\n")
			git("add", ".")
			git("commit", "-q", "-m", "init")
			s := posemodel.Store{Root: root}
			state, err := s.DiscoverComponent("svc")
			if err != nil {
				t.Fatal(err)
			}
			if err := s.SaveComponentState(state); err != nil {
				t.Fatal(err)
			}
			git("add", ".")
			git("commit", "-q", "-m", "assess")
			mustWrite(t, filepath.Join(root, "svc/main.go"), "package main\n\nfunc main() { println() }\n")
			git("commit", "-qam", "change")
			f := s.ComponentAssessmentFreshness("svc")
			return f.State == posemodel.AssessmentFresh, f.State
		}},
		adversarialCase{"unpinned-premise-trigger-reads-current", "enforced", "pose-assumption-validity-scope", func(t *testing.T) (bool, string) {
			// A material premise declares a trigger with no pin: it can never go
			// stale, and nothing says so.
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "api/x.proto"), "syntax = \"proto3\";\n")
			body := "## 2. Requirements\n\n- R1: x.\n\n## 5. Decisions\n\n### Assumption A1\n- Claim: the API is stable.\n- Status: verified\n- Stale trigger: doc:api/x.proto\n- Affects: R1\n"
			basis := posemodel.ValidateDesignBasis(body, root)
			for _, d := range basis.Diagnostics {
				if d.Code == "stale-trigger-unpinned" {
					return false, d.Message
				}
			}
			return true, "an unpinned trigger raised no diagnostic"
		}},
		adversarialCase{"calendar-ttl-as-premise-trigger", "enforced", "pose-assumption-validity-scope", func(t *testing.T) (bool, string) {
			// An arbitrary expiry date is offered as the material trigger.
			body := "## 2. Requirements\n\n- R1: x.\n\n## 5. Decisions\n\n### Assumption A1\n- Claim: the API is stable.\n- Status: verified\n- Stale trigger: date:2026-12-31\n- Affects: R1\n"
			for _, d := range posemodel.ValidateDesignBasis(body, t.TempDir()).Diagnostics {
				if d.Code == "calendar-trigger" {
					return false, d.Message
				}
			}
			return true, "a calendar expiry was accepted silently"
		}},
		adversarialCase{"unobserved-falsifier-reads-uncontradicted", "known-gap", "pose-falsifier-reconsideration", func(t *testing.T) (bool, string) {
			// A decision names a falsifier check that no validation run ever
			// produces; the projection raises nothing and nothing says the
			// falsifier was never observed.
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-d.md"), "---\nslug: d\nstatus: done\n---\n\n# Spec: d\n\n## 2. Requirements\n\n- R1: x.\n\n## 5. Decisions\n\n### Decision D1\n- Basis: R1\n- Selected option: y.\n- Falsifier: z.\n- Falsifier check: check:mod/never-run\n")
			mustWrite(t, filepath.Join(root, ".pose/indexes/delivery-integrity.json"), `{"schema_version":1,"input_digest":"sha256:x","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
			report, err := posemodel.Store{Root: root}.ProjectObligations(posemodel.ObligationQuery{Scope: "spec:d"})
			if err != nil {
				t.Fatal(err)
			}
			for _, o := range report.Obligations {
				if o.Source.Producer == "falsifiers" {
					return false, o.ReasonCode
				}
			}
			for _, l := range report.Limitations {
				if strings.Contains(l, "falsifier") {
					return false, l
				}
			}
			return true, "an unobserved falsifier check is silent"
		}},
	)
}

// delegatedCorpusCases are enforced by a test in another package because
// their fixture lives there; the corpus only checks the test still exists.
var delegatedCorpusCases = map[string]string{
	"request-opened-after-preview-transfers": "pose/spec_transfer_actions_test.go:TestTransferRefusesARequestOpenedAfterThePreview",
}

func TestDelegatedCorpusCasesStillExist(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "adversarial", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for name, ref := range delegatedCorpusCases {
		file, test, _ := strings.Cut(ref, ":")
		src, err := os.ReadFile(filepath.Join("..", file))
		if err != nil || !regexp.MustCompile(`(?m)^func `+test+`\(`).Match(src) {
			t.Errorf("delegated case %s: %s no longer exists", name, ref)
		}
		if !strings.Contains(string(raw), "| `"+name+"` | delegated |") {
			t.Errorf("README does not list delegated case %s", name)
		}
	}
}
