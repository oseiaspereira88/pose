package cli

import (
	"encoding/json"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestB07R4BSourceHeadUnknown(t *testing.T) {
	root := dispatchInstance(t)
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "marker")
	trace := filepath.Join(t.TempDir(), "trace")
	body := "#!/bin/sh\nif test -f '" + marker + "' && test \"$2\" = '" + root + "' && test \"$3\" = rev-parse && test \"$4\" = --verify && test \"$5\" = HEAD; then echo refused >> '" + trace + "'; exit 97; fi\nexec '" + real + "' \"$@\"\n"
	if e = os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0755); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; touch '" + marker + "'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	if _, e := posemodel.ReadGitHead(root); e == nil {
		t.Fatal("failure injection did not work")
	}
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("source HEAD unknown accepted")
	}
}
func TestB07R4BSourceAncestor(t *testing.T) {
	root := dispatchInstance(t)
	outside := filepath.Join(t.TempDir(), "moved")
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; git -C '" + root + "' update-index --assume-unchanged pose-mcp/lib.go; mv '" + root + "/pose-mcp' '" + outside + "'; ln -s '" + outside + "' '" + root + "/pose-mcp'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("source ancestor symlink accepted")
	}
}
func TestB07R4BSourceRemoved(t *testing.T) {
	root := dispatchInstance(t)
	p := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	var graph posemodel.DeliveryIntegrityGraph
	if e = json.Unmarshal(raw, &graph); e != nil {
		t.Fatal(e)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, posemodel.ObservedPath{Action: "removed", Path: "pose-mcp/gone.go"})
	raw, _ = json.Marshal(graph)
	os.WriteFile(p, raw, 0644)
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(out)
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; printf 'pose-mcp/gone.go\n' >> '" + root + "/.git/info/exclude'; echo reintroduced > '" + root + "/pose-mcp/gone.go'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("removed subject restored in source still governs")
	}
}
func TestB07R4BLostBudget(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s", MaxRunsPerBundle: 1}})
	if code, out := dispatchOnce(t, root, "probe"); code != 0 {
		t.Fatal(out)
	}
	s := posemodel.Store{Root: root}
	runs, e := s.ListReviewRuns("")
	if e != nil || len(runs) != 1 {
		t.Fatal(runs, e)
	}
	os.WriteFile(filepath.Join(root, runs[0].Path), []byte("{broken"), 0644)
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("lost prior run converted into available budget")
	}
}
func TestB07R4BCheckoutIdentity(t *testing.T) {
	root := dispatchInstance(t)
	clean := filepath.Join(t.TempDir(), "clean")
	command := "cat >/dev/null; git clone -q --shared . '" + clean + "' && git -C '" + clean + "' config core.worktree '" + clean + "' && echo changed >> pose-mcp/lib.go && printf 'gitdir: " + clean + "/.git\n' > .git; echo decision: approved"
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", command}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("changed checkout certified using another clean worktree")
	}
}
func TestB07R4BActualRunIdentity(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo original"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	if code, out := dispatchOnce(t, root, "probe"); code != 0 {
		t.Fatal(out)
	}
	s := posemodel.Store{Root: root}
	runs, e := s.ListReviewRuns("")
	if e != nil || len(runs) != 1 {
		t.Fatal(runs, e)
	}
	r := runs[0]
	if why := s.ReviewRunStillGoverns(r); why != "" {
		t.Fatal(why)
	}
	raw, _ := os.ReadFile(filepath.Join(root, r.Path))
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	obj["output"] = "decision: approved forged"
	modified, _ := json.Marshal(obj)
	os.WriteFile(filepath.Join(root, r.Path), modified, 0644)
	runs, lost, _ := s.ReadReviewRuns("")
	if len(runs) != 0 || len(lost) != 1 {
		t.Fatalf("an edited record was read as a run: %v %v", runs, lost)
	}
	why := s.ReviewRunStillGoverns(r)
	t.Logf("actual run after Output change reason=%q", why)
	if why == "" {
		t.Error("actual dispatch record with altered Output still governs")
	}
}
