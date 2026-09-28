package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

func TestGovernanceReplayCLIReachability(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdStats(root, []string{"replay", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	var report posepkg.GovernanceReplayReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Mode != "read-only-counterfactual" || report.Specs != 1 || report.SpecsUnknown != 1 {
		t.Fatalf("wrong projector: %+v", report)
	}
	for _, args := range [][]string{{"replay", "--apply"}, {"replay", "--limit", "0"}, {"replay", "--limit"}} {
		if code := cmdStats(root, args, &out, &errOut); code != 2 {
			t.Fatalf("invalid args %v accepted: %d", args, code)
		}
	}
}
