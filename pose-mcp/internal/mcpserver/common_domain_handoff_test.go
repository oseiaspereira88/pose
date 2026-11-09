package mcpserver

// Common domain and CLI/MCP handoff (spec pose-common-domain-and-cli-mcp-handoff).

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestEveryMCPRouteHasADeclaredClass(t *testing.T) {
	advertised := map[string]bool{}
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		advertised[name] = true
		if _, ok := mcpRouteInventory[name]; !ok {
			t.Errorf("tool %q has no declared route class", name)
		}
	}
	for _, name := range mcpRouteInventory.SortedTools() {
		if !advertised[name] {
			t.Errorf("the inventory names %q, which is not advertised", name)
		}
	}
	counts := map[pose.RouteClass]int{}
	for _, class := range mcpRouteInventory {
		counts[class]++
	}
	t.Logf("route classes over %d tools: %v", len(mcpRouteInventory), counts)
}

// The tools that may write are exactly the admitted write points, and only the
// governance-write catalog entries are classified that way.
func TestOnlyAdmittedWritesAreGovernanceWrites(t *testing.T) {
	for name, class := range mcpRouteInventory {
		risk := catalogGovernance[name].Risk
		if risk == "governance-write" && class != pose.RouteAdmittedWrite {
			t.Errorf("%s writes governance state but is classified %s", name, class)
		}
		if class == pose.RouteAdmittedWrite && risk != "governance-write" {
			t.Errorf("%s is classified an admitted write but its catalog risk is %s", name, risk)
		}
	}
}

func TestTheReadOnlyWrappersNoLongerStartTheBinary(t *testing.T) {
	ts := newTestServer(t, "")
	for _, tc := range []struct {
		tool string
		args string
	}{{"pose_suggest", `{"task_type":"feature"}`}, {"pose_get_followups", `{}`}} {
		before, runsBefore := pose.CLISpawnAttempts(), pose.CLIInProcessRuns()
		_, out := post(t, ts, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, tc.tool, tc.args))
		// The fixture may lack what the command needs (a task map): then the CLI's own
		// error comes back through the wrapper, which is the parity this proves too.
		if out.Result["isError"] == true {
			content, _ := out.Result["content"].([]any)
			text := ""
			if len(content) > 0 {
				m, _ := content[0].(map[string]any)
				text, _ = m["text"].(string)
			}
			if !strings.Contains(text, "pose "+strings.TrimPrefix(tc.tool, "pose_get_")) && !strings.Contains(text, "pose "+strings.TrimPrefix(tc.tool, "pose_")) {
				t.Errorf("%s failed with something that is not the command's own error: %q", tc.tool, text)
			}
		}
		if got := pose.CLISpawnAttempts() - before; got != 0 {
			t.Errorf("%s tried to start the binary %d times", tc.tool, got)
		}
		if pose.CLIInProcessRuns()-runsBefore != 1 {
			t.Errorf("%s did not run in process", tc.tool)
		}
	}
}

// The gates keep their process boundary: a call is an attempt to start the binary
// (refused in a test process, which is an error, not a verdict).
func TestTheGatesKeepTheProcessBoundary(t *testing.T) {
	ts := newTestServer(t, "")
	for _, tc := range []struct{ tool, args string }{{"pose_check", `{}`}, {"pose_skills_check", `{}`}, {"pose_lint_spec", `{"slug":"alpha"}`}} {
		before := pose.CLISpawnAttempts()
		post(t, ts, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, tc.tool, tc.args))
		if pose.CLISpawnAttempts()-before < 1 {
			t.Errorf("%s no longer goes through the process boundary", tc.tool)
		}
	}
}

// planFixture is a committed repository with an open spec, so the closeout planner
// has a Git revision to read and steps left to apply.
func planFixture(t *testing.T) *httptest.Server {
	ts, _ := planFixtureRoot(t)
	return ts
}

func planFixtureRoot(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n\n# Spec: alpha\n\nBody.\n")
	write(".pose/specs/beta/spec.md", "---\nslug: beta\nstatus: in-progress\ncreated_at: 2026-06-03\n---\n\n# Spec: beta\n\n## 2. Requirements\n\n- R1: Something.\n\n## 6. Validation\n\n### Requirement trace\n- R1 [waived: the fixture has no implementation]\n")
	write(".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`)
	write(".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts, root
}

func TestACloseoutPlanWithStepsLeftCarriesATypedHandoffThatExecutesNothing(t *testing.T) {
	ts := planFixture(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:beta","include_plan":true}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("closeout state failed: %+v %v", out.Error, out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	handoff, _ := structured["handoff"].(map[string]any)
	plan, _ := structured["execution_plan"].(map[string]any)
	if plan == nil {
		t.Fatalf("no execution plan: %v", structured)
	}
	if handoff == nil {
		t.Fatalf("a plan with steps left carries no handoff: terminal=%v steps=%v", plan["terminal"], plan["steps"])
	}
	if handoff["executes"] != false || handoff["operation"] != "close" || handoff["scope"] != "spec:beta" {
		t.Fatalf("handoff = %v", handoff)
	}
	argv, _ := handoff["argv"].([]any)
	if len(argv) < 4 || argv[0] != "pose" || argv[1] != "close" {
		t.Fatalf("the handoff is not structured arguments: %v", argv)
	}
	for _, a := range argv {
		if s, _ := a.(string); strings.ContainsAny(s, " \t\n;|&$`") {
			t.Errorf("an argument is shell prose: %q", s)
		}
	}
	inputs, _ := handoff["inputs"].(map[string]any)
	if inputs["plan_digest"] != plan["digest"] {
		t.Fatalf("the handoff is not bound to the plan it came from: %v vs %v", inputs, plan["digest"])
	}
	raw, _ := json.Marshal(handoff)
	if strings.Contains(strings.ToLower(string(raw)), "token") || strings.Contains(strings.ToLower(string(raw)), "secret") {
		t.Fatalf("the handoff carries something secret-shaped: %s", raw)
	}
}

// Planning is read-only and stable: asking twice for the same state returns the same
// terminal, next action and digest (P15), and a done spec yields no handoff.
func TestThePlanAndItsHandoffAreStableAndADoneSpecHasNone(t *testing.T) {
	ts := planFixture(t)
	call := func(scope string) map[string]any {
		_, out := post(t, ts, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":%q,"include_plan":true}}}`, scope))
		sc, _ := out.Result["structuredContent"].(map[string]any)
		return sc
	}
	a, b := call("spec:beta"), call("spec:beta")
	pa, _ := a["execution_plan"].(map[string]any)
	pb, _ := b["execution_plan"].(map[string]any)
	if pa == nil || pb == nil {
		t.Fatalf("no plan to compare: %v %v", a, b)
	}
	if pa["digest"] != pb["digest"] || pa["terminal"] != pb["terminal"] || pa["next_action"] != pb["next_action"] {
		t.Fatalf("the same state planned twice differs: %v vs %v", pa, pb)
	}
	if ha, hb := a["handoff"], b["handoff"]; fmt.Sprint(ha) != fmt.Sprint(hb) {
		t.Fatalf("the handoff is not deterministic: %v vs %v", ha, hb)
	}
	if done := call("spec:alpha"); done["handoff"] != nil {
		t.Fatalf("a done spec has a handoff: %v", done["handoff"])
	}
}

// The MCP plan is the plan the CLI computes: both read PlanCloseout, so for one
// state they agree on digest, terminal, next action and steps (P15), and the
// handoff derived from it names that digest.
func TestTheMCPPlanIsTheDomainPlan(t *testing.T) {
	ts, root := planFixtureRoot(t)
	direct, err := (pose.Store{Root: root}).PlanCloseout("spec:beta")
	if err != nil {
		t.Fatal(err)
	}
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:beta","include_plan":true}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	plan, _ := sc["execution_plan"].(map[string]any)
	next, _ := plan["next_action"].(string)
	if plan["digest"] != direct.Digest || plan["terminal"] != direct.Terminal || next != direct.NextAction {
		t.Fatalf("the MCP plan differs from the domain plan: %v vs %+v", plan, direct)
	}
	steps, _ := plan["steps"].([]any)
	if len(steps) != len(direct.Steps) {
		t.Fatalf("steps differ: %d vs %d", len(steps), len(direct.Steps))
	}
	if h := pose.CloseoutHandoff(direct, pose.HandoffTarget{}); h == nil || h.Inputs.PlanDigest != direct.Digest {
		t.Fatalf("the handoff is not bound to the domain plan: %+v", h)
	}
}

// The handoff returned over MCP names the project by its logical id and carries no
// filesystem path: the MCP contract exposes no root (spec pose-common-domain-and-cli-mcp-handoff).
func TestTheHandoffOverMCPCarriesNoFilesystemPath(t *testing.T) {
	ts, root := planFixtureRoot(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:beta","include_plan":true}}}`)
	raw, err := json.Marshal(out.Result)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["handoff"] == nil {
		t.Fatalf("the fixture's plan has steps left and must carry a handoff: %s", raw)
	}
	if strings.Contains(string(raw), root) {
		t.Fatalf("the MCP response carries the filesystem root %q: %s", root, raw)
	}
}

// The write points an MCP caller can reach are exactly the four admitted ones, and
// no tool is named for applying a closeout: the plan hands that operation off.
func TestTheAdmittedWritesAreExactlyTheKnownFour(t *testing.T) {
	var got []string
	for _, name := range mcpRouteInventory.SortedTools() {
		if mcpRouteInventory[name] == pose.RouteAdmittedWrite {
			got = append(got, name)
		}
		// The Conductor reporter closes a run, not a spec: that is the external boundary.
		if strings.Contains(name, "close") && !strings.HasPrefix(name, "conductor_") && mcpRouteInventory[name] != pose.RouteReadOnlyPlan {
			t.Errorf("%s looks like a closeout operation but is %s", name, mcpRouteInventory[name])
		}
	}
	want := []string{"pose_action_fulfil", "pose_action_open", "pose_action_resolve", "pose_review_attest"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("admitted writes = %v, want %v", got, want)
	}
}
