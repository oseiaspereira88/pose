package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestQualifiedArtifactCLIProjectionDatedAndTyped(t *testing.T) {
	self, other, _ := setupTwoProjectPortfolio(t)
	writePortfolioSpec(t, self, "consumer", "draft", "xref:other-project/spec:upstream")
	path := filepath.Join(other, ".pose", "specs", "2026-09-21-upstream.md")
	if err := os.WriteFile(path, []byte("---\nslug: upstream\nstatus: done\n---\n# Upstream\n"), 0644); err != nil {
		t.Fatal(err)
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
