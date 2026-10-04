package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-state-attention.
func attentionFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-01-base.md"), "---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n")
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-02-consumer.md"), "---\nslug: consumer\nstatus: draft\ndepends_on: base\n---\n\n# Spec: consumer\n")
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-03-shipped.md"), "---\nslug: shipped\nstatus: done\n---\n\n# Spec: shipped\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Revisit later (owner:@core crit:low review:2027-01-01)\n")
	return root
}

func TestStateAttentionShowsCoverageFirstAndSeparatesKinds(t *testing.T) {
	root := attentionFixture(t)
	var out, errB bytes.Buffer
	inDir(t, root, func() { Main([]string{"state", "--attention"}, &out, &errB) })
	text := out.String()
	coverage := strings.Index(text, "attention.coverage=INCOMPLETE")
	firstGroup := strings.Index(text, "attention.for_actor=")
	if coverage < 0 || firstGroup < 0 || coverage > firstGroup {
		t.Fatalf("an incomplete answer must say so before any group:\n%s%s", text, errB.String())
	}
	for _, want := range []string{"attention.blocks.start=1 obligation(s)", "[dependency dependency-not-done] consumer waits on base", "attention.residual=1 advisory follow-up(s); they restrict no phase", "attention.blocks.closeout=0 obligation(s)"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

func TestStateAttentionAndMCPShareIDs(t *testing.T) {
	root := attentionFixture(t)
	var out, errB bytes.Buffer
	inDir(t, root, func() { Main([]string{"state", "--attention", "--json"}, &out, &errB) })
	var payload struct {
		Obligations []pose.Obligation `json:"obligations"`
		Attention   pose.Attention    `json:"attention"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("%v %s", err, out.String())
	}
	direct, _ := pose.Store{Root: root}.ProjectObligations(pose.ObligationQuery{})
	if len(payload.Obligations) != len(direct.Obligations) {
		t.Fatalf("CLI %d vs domain %d", len(payload.Obligations), len(direct.Obligations))
	}
	for i := range direct.Obligations {
		if payload.Obligations[i].ID != direct.Obligations[i].ID {
			t.Fatalf("ids differ at %d", i)
		}
	}
	if len(payload.Attention.Residual) != 1 || len(payload.Attention.Blocking["start"]) != 1 {
		t.Fatalf("attention grouping: %+v", payload.Attention)
	}
}
