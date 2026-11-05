package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestQualifiedArtifactCLIProjectionDatedAndTyped(t *testing.T) {
	self, other, _ := setupTwoProjectPortfolio(t)
	t.Setenv("POSE_PROJECT_ROOTS", "")
	writePortfolioSpec(t, self, "consumer", "draft", "xref:other-project/spec:upstream")
	path := filepath.Join(other, ".pose", "specs", "2026-09-21-upstream.md")
	if err := os.WriteFile(path, []byte("---\nslug: upstream\nstatus: done\n---\n# Upstream\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// The external source needs a real revision before the baseline projection.
	// An unborn repository cannot supply the B34 authorized-root proof.
	for _, args := range [][]string{{"add", "--all"}, {"-c", "user.name=POSE fixture", "-c", "user.email=pose@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "upstream baseline"}} {
		if out, err := exec.Command("git", append([]string{"-C", other}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	known, err := discoverAuthorizedProjects(self, "")
	if err != nil {
		t.Fatal(err)
	}
	projection := buildPortfolioProjection(time.Now(), known, 7)
	if len(projection.Specs) != 2 || projection.SchemaVersion != 3 {
		t.Fatalf("projection: %+v", projection)
	}
	consumer := findProjectedSpec(t, projection, "self-project", "consumer")
	dep := consumer.XrefsOut[0]
	if !dep.Resolved || dep.Blocking || dep.Identity.String() != "xref:other-project/spec:upstream" || dep.SourceDigest == "" {
		t.Fatalf("dependency: %+v", dep)
	}
	ready, err := (pose.Store{Root: self}).SpecReadiness("consumer")
	if err != nil || !ready.Ready {
		t.Fatalf("readiness: %+v %v", ready, err)
	}
	var out bytes.Buffer
	checker := nativeChecker{root: self, mode: "strict", stdout: &out}
	checker.checkSpecs()
	if checker.errors != 0 {
		t.Fatalf("check disagrees: %s", out.String())
	}
	var indexOut, indexErr bytes.Buffer
	if code := cmdIndex(self, nil, &indexOut, &indexErr); code == 0 || !strings.Contains(indexErr.String(), "foreign writes require an explicit project root") {
		t.Fatalf("discovery granted index write scope: %d %s %s", code, indexOut.String(), indexErr.String())
	}
	if _, err := os.Stat(filepath.Join(self, ".pose", "indexes", "generation.json")); !os.IsNotExist(err) {
		t.Fatalf("index published without an explicit foreign binding: %v", err)
	}
	indexOut.Reset()
	indexErr.Reset()
	// Discovery authorizes reads only. Index publication also leases the
	// foreign material root, which this fixture must authorize explicitly.
	bindings, err := json.Marshal(map[string]string{"other-project": other})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_PROJECT_ROOTS", string(bindings))
	if code := cmdIndex(self, nil, &indexOut, &indexErr); code != 0 {
		t.Fatalf("index: %s %s", indexOut.String(), indexErr.String())
	}
	indexed, err := os.ReadFile(filepath.Join(self, ".pose", "indexes", "spec-graph.json"))
	if err != nil || !strings.Contains(string(indexed), `"dependency_resolutions"`) || !strings.Contains(string(indexed), dep.SourceDigest) {
		t.Fatalf("index disagrees: %s %v", indexed, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	// A same-slug local spec cannot satisfy an unavailable external source.
	writePortfolioSpec(t, self, "upstream", "done", "")
	ready, err = (pose.Store{Root: self}).SpecReadiness("consumer")
	if err != nil || ready.Ready {
		t.Fatalf("fallback: %+v %v", ready, err)
	}
	projection = buildPortfolioProjection(time.Now(), known, 7)
	consumer = findProjectedSpec(t, projection, "self-project", "consumer")
	if !consumer.XrefsOut[0].Blocking || consumer.XrefsOut[0].Reason != "unknown-spec" {
		t.Fatal(consumer)
	}
	raw, _ := json.Marshal(projection)
	if strings.Contains(string(raw), self) || strings.Contains(string(raw), other) {
		t.Fatal("projection disclosed roots")
	}
}
