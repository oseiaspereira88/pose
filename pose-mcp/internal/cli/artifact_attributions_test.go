package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestRetroactiveAttribution(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen unavailable")
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	root, base, _ := artifactGitFixture(t)
	specPath := ".pose/specs/alpha/spec.md"
	specBytes, err := os.ReadFile(filepath.Join(root, specPath))
	if err != nil {
		t.Fatal(err)
	}
	writeArtifactTestFile(t, root, specPath, strings.Replace(string(specBytes), "status: in-progress", "status: done", 1))
	writeArtifactTestFile(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"proj.attribution-fixture"}`)
	const path = "internal/undeclared.go"
	writeArtifactTestFile(t, root, path, "package internal\n")
	artifactGit(t, root, "add", ".pose/project.json", specPath, path)
	artifactGit(t, root, "commit", "-qm", "historical change", "-m", "POSE-Spec: alpha")
	head := artifactGit(t, root, "rev-parse", "HEAD")
	_, claims, sets, tracked, policy, _, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
	if err != nil {
		t.Fatal(err)
	}
	specs := []posemodel.Spec{{Slug: "alpha", Status: "done"}}
	original := posemodel.BuildDeliveryIntegrity(specs, claims, sets, tracked, policy)
	key := filepath.Join(t.TempDir(), "maintainer")
	if out, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("test key: %s: %v", out, err)
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	register := func(roles, principal string) {
		t.Helper()
		writeArtifactTestFile(t, root, ".pose/policy/actions.json",
			`{"schema_version":1,"roles":{"maintainer":[`+roles+`]},"keys":{"`+principal+`":[{"key":`+string(mustAttributionJSON(t, strings.TrimSpace(string(pub))))+`}]}}`)
	}
	register(`"human:maintainer"`, "human:maintainer")
	manifest := attributionManifest{1, "proj.attribution-fixture", "human:maintainer", head, []attributionRecord{
		{Spec: "alpha", Base: base, Head: head, Witness: head, Observed: posemodel.ObservedPath{Action: "created", Path: path}, Reason: "Review of the historical range"},
	}}
	signed := func(m attributionManifest, namespace string) {
		t.Helper()
		raw := mustAttributionJSON(t, m)
		writeArtifactTestFile(t, root, posemodel.ArtifactAttributionManifest, string(raw))
		cmd := exec.Command(keygen, "-Y", "sign", "-f", key, "-n", namespace)
		cmd.Stdin = bytes.NewReader(raw)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		writeArtifactTestFile(t, root, posemodel.ArtifactAttributionManifest+".sig", string(out))
	}
	apply := func() (posemodel.DeliveryIntegrityGraph, error) {
		g := original
		g.Findings = append([]posemodel.DeliveryIntegrityFinding(nil), original.Findings...)
		err := applyArtifactAttributions(posemodel.NativeFiles(), root, specs, &g)
		return g, err
	}
	signed(manifest, attributionNamespace)
	g, err := apply()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range g.Findings {
		if f.Code == "undeclared" {
			t.Fatal("exact signed record failed to satisfy finding")
		}
		if f.Code == "retroactive-attribution" && strings.Contains(f.Message, "human:maintainer") && strings.Contains(f.Message, manifest.Records[0].Reason) {
			found = true
		}
	}
	if !found || g.ProvenanceDigest != original.ProvenanceDigest || g.InputDigest == original.InputDigest {
		t.Fatal("wrong attribution disposition or digest binding")
	}

	var output, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--strict", "--json"}, &output, &errOut); code != 0 || !strings.Contains(output.String(), "retroactive-attribution") {
		t.Fatalf("artifact-check did not consume signed finding: %d: %s", code, errOut.String())
	}
	artifactGit(t, root, "commit", "--allow-empty", "-qm", "new attributed range", "-m", "POSE-Spec: alpha")
	output.Reset()
	errOut.Reset()
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--strict", "--json"}, &output, &errOut); code == 0 || !strings.Contains(output.String(), "undeclared") {
		t.Fatalf("record authorized a new range: %d: %s %s", code, output.String(), errOut.String())
	}

	for _, tc := range []struct {
		name string
		edit func(*attributionManifest)
		want string
	}{
		{"wrong project", func(m *attributionManifest) { m.Project = "proj.somewhere" }, "invalid project"},
		{"unknown spec", func(m *attributionManifest) { m.Records[0].Spec = "unknown" }, "invalid or duplicate"},
		{"wrong path", func(m *attributionManifest) { m.Records[0].Observed.Path = "internal/elsewhere.go" }, "did not make the exact action and path"},
		{"wrong action", func(m *attributionManifest) { m.Records[0].Observed.Action = "modified" }, "did not make the exact action and path"},
		{"outside baseline", func(m *attributionManifest) { m.Baseline = base }, "outside baseline"},
		{"outside range", func(m *attributionManifest) { m.Records[0].Witness = base }, "outside range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := manifest
			m.Records = append([]attributionRecord(nil), manifest.Records...)
			tc.edit(&m)
			signed(m, attributionNamespace)
			if _, err := apply(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}

	specs[0].Status = "in-progress"
	signed(manifest, attributionNamespace)
	if _, err := apply(); err == nil || !strings.Contains(err.Error(), "invalid or duplicate") {
		t.Fatalf("open scope accepted retroactive attribution: %v", err)
	}
	specs[0].Status = "done"

	signed(manifest, "other-namespace")
	if _, err := apply(); err == nil || !strings.Contains(err.Error(), "namespace") {
		t.Fatalf("foreign namespace accepted: %v", err)
	}
	signed(manifest, attributionNamespace)
	register(`"human:maintainer"`, "human:other")
	if _, err := apply(); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("unauthorized key accepted: %v", err)
	}
	register(`"human:other"`, "human:maintainer")
	if _, err := apply(); err == nil || !strings.Contains(err.Error(), "human maintainer") {
		t.Fatalf("unassigned signer accepted: %v", err)
	}
	register(`"human:maintainer"`, "human:maintainer")
	if err := os.Remove(filepath.Join(root, posemodel.ArtifactAttributionManifest+".sig")); err != nil {
		t.Fatal(err)
	}
	if _, err := apply(); err == nil || !strings.Contains(err.Error(), "read signature") {
		t.Fatalf("unsigned record accepted: %v", err)
	}
}

func mustAttributionJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
