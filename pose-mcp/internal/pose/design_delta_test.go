package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func designDeltaGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	output, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func mustDesignDeltaGitOutput(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return output
}

func writeDesignDeltaFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestABMStructuralDeltaObservesBothSidesAndStableIDs(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeDesignDeltaFile(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n\nrequire (\n\tgithub.com/old/dependency v1.0.0\n)\n")
	writeDesignDeltaFile(t, root, "package.json", "{\n  \"name\": \"app\",\n  \"dependencies\": {\"old-package\": \"^1.0.0\"},\n  \"devDependencies\": {\"test-runner\": \"^1.0.0\"}\n}\n")
	writeDesignDeltaFile(t, root, "old.go", "package app\n")
	base := designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "base")
	base = strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))

	writeDesignDeltaFile(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n\nrequire (\n\tgithub.com/old/dependency v1.1.0\n\tgithub.com/new/dependency v2.0.0\n\tgithub.com/transitive v1.0.0 // indirect\n)\n")
	writeDesignDeltaFile(t, root, "package.json", "{\n  \"name\": \"app\",\n  \"dependencies\": {\"old-package\": \"^2.0.0\", \"new-package\": \"^1.0.0\"},\n  \"devDependencies\": {\"test-runner\": \"^1.1.0\"},\n  \"optionalDependencies\": {\"optional-package\": \"^1.0.0\"}\n}\n")
	if err := os.Rename(filepath.Join(root, "old.go"), filepath.Join(root, "new.go")); err != nil {
		t.Fatal(err)
	}
	writeDesignDeltaFile(t, root, ".pose/policy/delivery.json", "{\"schema_version\":1}\n")
	designDeltaGit(t, root, "add", "--", ".")
	head := designDeltaGit(t, root, "commit", "-q", "-m", "head")

	subject := ReviewBundleSubject{Base: base, Head: head, Entries: []ReviewBundleSubjectEntry{
		{Action: "modified", Path: "go.mod", Class: "implementation"},
		{Action: "modified", Path: "package.json", Class: "implementation"},
		{Action: "renamed", OldPath: "old.go", NewPath: "new.go", Class: "implementation"},
		{Action: "created", Path: ".pose/policy/delivery.json", Class: "governance"},
	}}
	report, err := AssessDesignDelta(root, subject, "spec:demo", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "observed" {
		t.Fatalf("status = %q, warnings=%v, coverage=%+v", report.Status, report.Warnings, report.Coverage)
	}
	if report.InputDigest == "" || report.CacheKey != report.InputDigest {
		t.Fatalf("missing stable input/cache digest: %+v", report)
	}
	if len(report.Deltas) < 8 {
		t.Fatalf("expected component, metadata and dependency deltas, got %+v", report.Deltas)
	}
	var sawNew, sawTransitive, sawRename, sawDelivery bool
	for _, delta := range report.Deltas {
		switch {
		case delta.Subject == "go:github.com/new/dependency" && delta.Action == "added":
			sawNew = delta.BeforeDigest == "" && delta.AfterDigest != ""
		case delta.Subject == "go:github.com/transitive" && delta.Runtime == "runtime-transitive":
			sawTransitive = true
		case delta.Kind == "path" && delta.Action == "renamed":
			sawRename = true
		case delta.Kind == "delivery-metadata" && delta.Subject == ".pose/policy/delivery.json":
			sawDelivery = true
		}
		if len(delta.ID) < 12 || len(delta.DisplayID) < 4 {
			t.Fatalf("delta identity is too short: %+v", delta)
		}
	}
	if !sawNew || !sawTransitive || !sawRename || !sawDelivery {
		t.Fatalf("missing expected deltas new=%v transitive=%v rename=%v delivery=%v: %+v", sawNew, sawTransitive, sawRename, sawDelivery, report.Deltas)
	}
	second, err := AssessDesignDelta(root, subject, "spec:demo", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(report)
	right, _ := json.Marshal(second)
	if string(left) != string(right) {
		t.Fatalf("same subject was not deterministic:\n%s\n%s", left, right)
	}
}

func TestABMStructuralDeltaSubjectActionsAndUnsupportedCoverage(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeDesignDeltaFile(t, root, "Cargo.toml", "[package]\nname = \"demo\"\n")
	designDeltaGit(t, root, "add", "--", ".")
	base := designDeltaGit(t, root, "commit", "-q", "-m", "base")
	writeDesignDeltaFile(t, root, "Cargo.toml", "[package]\nname = \"demo2\"\n")
	designDeltaGit(t, root, "add", "--", ".")
	head := designDeltaGit(t, root, "commit", "-q", "-m", "head")
	report, err := AssessDesignDelta(root, ReviewBundleSubject{Base: base, Head: head, Entries: []ReviewBundleSubjectEntry{
		{Action: "modified", Path: "Cargo.toml", Class: "implementation"},
		{Action: "modified", Path: "vendor/dep", Class: "submodule"},
	}}, "spec:demo", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "partial" {
		t.Fatalf("unsupported subject should be partial, got %q (%+v)", report.Status, report)
	}
	var unsupported, submodule bool
	for _, delta := range report.Deltas {
		unsupported = unsupported || delta.Kind == "unsupported-manifest" && delta.State == "unsupported"
		submodule = submodule || delta.Kind == "submodule" && delta.State == "observed"
	}
	if !unsupported || !submodule {
		t.Fatalf("missing unsupported/submodule observations: %+v", report.Deltas)
	}
}

func TestABMDesignDeltaBoundsAndUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeDesignDeltaFile(t, root, "go.mod", "module example.test/app\n")
	designDeltaGit(t, root, "add", "--", ".")
	base := designDeltaGit(t, root, "commit", "-q", "-m", "base")
	writeDesignDeltaFile(t, root, "go.mod", "module example.test/app\n\nrequire example.test/large v1.0.0\n")
	designDeltaGit(t, root, "add", "--", ".")
	head := designDeltaGit(t, root, "commit", "-q", "-m", "head")
	entries := []ReviewBundleSubjectEntry{{Action: "modified", Path: "go.mod", Class: "implementation"}, {Action: "modified", Path: "../escape", Class: "implementation"}}
	for i := 0; i < 4; i++ {
		entries = append(entries, ReviewBundleSubjectEntry{Action: "modified", Path: "go.mod", Class: "implementation", Digest: string(rune('a' + i))})
	}
	report, err := AssessDesignDelta(root, ReviewBundleSubject{Base: base, Head: head, Entries: entries}, "spec:demo", DesignDeltaOptions{MaxFiles: 1, MaxBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Coverage.Truncated || report.Coverage.FilesSkipped == 0 {
		t.Fatalf("file bound not visible: %+v", report.Coverage)
	}
	if report.Status != "partial" && report.Status != "unknown" {
		t.Fatalf("bounded assessment status = %q", report.Status)
	}
	serialized, _ := json.Marshal(report)
	if strings.Contains(string(serialized), "../") || strings.Contains(string(serialized), root) {
		t.Fatalf("unsafe path leaked into report: %s", serialized)
	}
	unknown, err := AssessDesignDelta(root, ReviewBundleSubject{Base: "not-a-revision", Head: head, Entries: entries[:1]}, "spec:demo", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Status != "unknown" || len(unknown.Warnings) == 0 {
		t.Fatalf("unsafe revision should remain unknown: %+v", unknown)
	}
}
