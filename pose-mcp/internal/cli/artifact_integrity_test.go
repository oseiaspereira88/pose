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

func artifactGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeArtifactTestFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func artifactGitFixture(t *testing.T) (root, base, head string) {
	t.Helper()
	root = t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error","orphan":"warning"}}`)
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-03\n---\n\n# Spec: alpha\n\n## 3. Technical Plan\n\n### Artifacts\n- modified: internal/core.go\n- created: internal/new.go\n\n## 4. Tasks\nwork\n")
	writeArtifactTestFile(t, root, "internal/core.go", "package internal\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")
	base = artifactGit(t, root, "rev-parse", "HEAD")
	writeArtifactTestFile(t, root, "internal/core.go", "package internal\n// changed\n")
	writeArtifactTestFile(t, root, "internal/new.go", "package internal\n")
	artifactGit(t, root, "add", "--", "internal/core.go", "internal/new.go")
	artifactGit(t, root, "commit", "-q", "-m", "implement alpha", "-m", "POSE-Spec: alpha")
	head = artifactGit(t, root, "rev-parse", "HEAD")
	return root, base, head
}

func TestArtifactCheckMatchesExplicitGitChangeSetAndRejectsUnsafeRevision(t *testing.T) {
	root, base, head := artifactGitFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--from", base, "--to", head, "--strict", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("artifact-check code=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	var graph posemodel.DeliveryIntegrityGraph
	if err := json.Unmarshal(out.Bytes(), &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.ChangeSets) != 1 || len(graph.ChangeSets[0].Paths) != 2 || graph.ChangeSets[0].DiffDigest == "" {
		t.Fatalf("unexpected graph: %+v", graph)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--from", "--upload-pack=evil", "--to", head}, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "unsafe Git revision") {
		t.Fatalf("unsafe revision accepted: code=%d err=%s", code, errOut.String())
	}
}

func TestArtifactCheckFindsUndeclaredAndActionMismatch(t *testing.T) {
	root, base, head := artifactGitFixture(t)
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	raw, _ := os.ReadFile(path)
	text := strings.Replace(string(raw), "- created: internal/new.go", "- removed: internal/missing.go", 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--from", base, "--to", head, "--strict"}, &out, &errOut); code != 1 {
		t.Fatalf("mismatch should block: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "action-mismatch") || !strings.Contains(out.String(), "undeclared") {
		t.Fatalf("missing findings: %s", out.String())
	}
}

func TestArtifactCheckMatchesRenameWithEditAndRemoval(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error"}}`)
	writeArtifactTestFile(t, root, ".pose/specs/rename/spec.md", "---\nslug: rename\nstatus: in-progress\ncreated_at: 2026-08-03\n---\n\n# Spec: rename\n\n## 3. Technical Plan\n\n### Artifacts\n- renamed: internal/old.go -> internal/new.go\n- removed: internal/obsolete.go\n\n## 4. Tasks\nwork\n")
	writeArtifactTestFile(t, root, "internal/old.go", "package internal\n\nfunc preservedOne() {}\nfunc preservedTwo() {}\nfunc preservedThree() {}\n")
	writeArtifactTestFile(t, root, "internal/obsolete.go", "package internal\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")
	base := artifactGit(t, root, "rev-parse", "HEAD")
	artifactGit(t, root, "mv", "--", "internal/old.go", "internal/new.go")
	writeArtifactTestFile(t, root, "internal/new.go", "package internal\n\nfunc preservedOne() {}\nfunc preservedTwo() {}\nfunc preservedThree() {}\n// edited after rename\n")
	artifactGit(t, root, "rm", "-q", "--", "internal/obsolete.go")
	artifactGit(t, root, "add", "--", "internal/new.go")
	artifactGit(t, root, "commit", "-q", "-m", "rename and remove", "-m", "POSE-Spec: rename")
	head := artifactGit(t, root, "rev-parse", "HEAD")
	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "rename", "--from", base, "--to", head, "--strict", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("rename/remove check code=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	var graph posemodel.DeliveryIntegrityGraph
	if err := json.Unmarshal(out.Bytes(), &graph); err != nil {
		t.Fatal(err)
	}
	if got := graph.ChangeSets[0].Paths; len(got) != 2 || got[0].Action != "removed" || got[1].Action != "renamed" {
		t.Fatalf("unexpected rename/remove observations: %+v", got)
	}
}

func TestArtifactCheckReconcilesRecordedReleaseRename(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, root, "README.md", "fixture\n")
	artifactGit(t, root, "add", "--", "README.md")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["release"],"severities":{"action-mismatch":"error","undeclared":"warning"}}`)
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-03\n---\n\n# Spec: alpha\n\n## 3. Technical Plan\n\n### Artifacts\n- created: release/unreleased/alpha.md\n")
	writeArtifactTestFile(t, root, "release/unreleased/alpha.md", "alpha\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "implement alpha", "-m", "POSE-Spec: alpha")

	base := artifactGit(t, root, "rev-parse", "HEAD")
	if err := os.MkdirAll(filepath.Join(root, "release/v1.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactGit(t, root, "mv", "--", "release/unreleased/alpha.md", "release/v1.1.0/alpha.md")
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-03\n---\n\n# Spec: alpha\n\n## 3. Technical Plan\n\n### Artifacts\n- renamed: release/unreleased/alpha.md -> release/v1.1.0/alpha.md\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "prepare release")
	head := artifactGit(t, root, "rev-parse", "HEAD")

	var reportOut, reportErr bytes.Buffer
	args := []string{"--task", "alpha release attribution", "--spec", "alpha", "--outcome", "pass", "--change-from", base, "--change-to", head}
	if code := cmdReport(root, args, &reportOut, &reportErr); code != 0 {
		t.Fatalf("report code=%d err=%s", code, reportErr.String())
	}

	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--strict", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("artifact-check code=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	var graph posemodel.DeliveryIntegrityGraph
	if err := json.Unmarshal(out.Bytes(), &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.ChangeSets) != 2 {
		t.Fatalf("recorded release change set was not reconciled: %+v", graph.ChangeSets)
	}
	for _, finding := range graph.Findings {
		if finding.Code == "action-mismatch" {
			t.Fatalf("recorded release rename did not satisfy the final claim: %+v", finding)
		}
	}
}

func TestArtifactBackfillDryRunDoesNotMutateSpecs(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	before, _ := os.ReadFile(path)
	var out, errOut bytes.Buffer
	if code := cmdArtifactBackfill(root, []string{"--from-git"}, &out, &errOut); code != 0 {
		t.Fatalf("backfill code=%d err=%s", code, errOut.String())
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) || !strings.Contains(out.String(), `"dry_run": true`) || !strings.Contains(out.String(), `"confidence"`) {
		t.Fatalf("dry run mutated spec or omitted marker")
	}
	out.Reset()
	errOut.Reset()
	if code := cmdArtifactBackfill(root, nil, &out, &errOut); code != 2 {
		t.Fatalf("backfill without selector should be usage error: code=%d err=%s", code, errOut.String())
	}
}

func TestReportChangeSetPersistsImmutableGitEvidence(t *testing.T) {
	root, base, head := artifactGitFixture(t)
	var out, errOut bytes.Buffer
	args := []string{"--task", "alpha delivery", "--spec", "alpha", "--outcome", "pass", "--change-from", base, "--change-to", head}
	if code := cmdReport(root, args, &out, &errOut); code != 0 {
		t.Fatalf("report code=%d err=%s", code, errOut.String())
	}
	history := filepath.Join(root, ".pose/reports/history/standard-alpha-delivery.jsonl")
	raw, err := os.ReadFile(history)
	if err != nil {
		t.Fatal(err)
	}
	var record reportRecord
	if err := json.Unmarshal(bytes.TrimSpace(raw), &record); err != nil {
		t.Fatal(err)
	}
	if record.ChangeSet == nil || record.ChangeSet.ResolvedBase != base || record.ChangeSet.ResolvedHead != head || record.ChangeSet.DiffDigest == "" {
		t.Fatalf("missing change-set evidence: %+v", record)
	}
	if record.StableHash == "" {
		t.Fatal("change-set did not participate in stable record")
	}
}

func TestMalformedReportHistoryCannotFabricateChangeSet(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	writeArtifactTestFile(t, root, ".pose/reports/history/broken.jsonl", "{not-json}\n")
	if sets := loadRecordedChangeSets(root); len(sets) != 0 {
		t.Fatalf("malformed history produced evidence: %+v", sets)
	}
}

func TestIndexWritesDeliveryIntegrityGraphAndReverseLookup(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdIndex(root, nil, &out, &errOut); code != 0 {
		t.Fatalf("index code=%d err=%s", code, errOut.String())
	}
	graph, err := (posemodel.Store{Root: root}).GetDeliveryIntegrity("internal/core.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.Reverse["internal/core.go"]; len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("reverse=%v", graph.Reverse)
	}
}
