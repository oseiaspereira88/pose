package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContributeLifecycleCommands(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	// Check initial status (inactive)
	code := cmdContribute(root, []string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("initial status failed: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "INACTIVE") {
		t.Errorf("expected INACTIVE status, got: %s", stdout.String())
	}

	// Enable contributor mode
	stdout.Reset()
	stderr.Reset()
	code = cmdContribute(root, []string{"enable"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("enable failed: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ENABLED") {
		t.Errorf("expected ENABLED in output, got: %s", stdout.String())
	}

	// Check status (active)
	stdout.Reset()
	code = cmdContribute(root, []string{"status", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status --json failed: %d", code)
	}
	if !strings.Contains(stdout.String(), `"active": true`) {
		t.Errorf("expected active: true in json, got: %s", stdout.String())
	}

	// Stage a contribution
	stdout.Reset()
	stderr.Reset()
	code = cmdContribute(root, []string{"stage", "--title", "Sample Rule Limitation", "--type", "limitation", "--body", "Details"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("stage failed: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Staged contribution recorded") {
		t.Errorf("expected staged contribution output, got: %s", stdout.String())
	}

	// List contributions
	stdout.Reset()
	code = cmdContribute(root, []string{"list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("list failed: %d", code)
	}
	if !strings.Contains(stdout.String(), "Sample Rule Limitation") {
		t.Errorf("expected staged item in list, got: %s", stdout.String())
	}

	// Disable contributor mode
	stdout.Reset()
	stderr.Reset()
	code = cmdContribute(root, []string{"disable"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("disable failed: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "DISABLED") {
		t.Errorf("expected DISABLED in output, got: %s", stdout.String())
	}
}

func TestContributeEnableInjectsDocSections(t *testing.T) {
	root := t.TempDir()
	agentsFile := filepath.Join(root, "AGENTS.md")
	poseFile := filepath.Join(root, "POSE.md")
	_ = os.WriteFile(agentsFile, []byte("# AGENTS.md\n\n## Context\nSome context.\n"), 0o644)
	_ = os.WriteFile(poseFile, []byte("# POSE.md\n\n## Overview\nSome overview.\n"), 0o644)

	var stdout, stderr bytes.Buffer
	code := cmdContribute(root, []string{"enable"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("enable failed: %d", code)
	}

	rawAgents, _ := os.ReadFile(agentsFile)
	if !strings.Contains(string(rawAgents), contributorModeMarker) {
		t.Error("AGENTS.md must contain contributorModeMarker after enable")
	}

	rawPose, _ := os.ReadFile(poseFile)
	if !strings.Contains(string(rawPose), contributorModeMarker) {
		t.Error("POSE.md must contain contributorModeMarker after enable")
	}

	// Disable removes the sections
	stdout.Reset()
	code = cmdContribute(root, []string{"disable"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("disable failed: %d", code)
	}

	rawAgentsCleaned, _ := os.ReadFile(agentsFile)
	if strings.Contains(string(rawAgentsCleaned), contributorModeMarker) {
		t.Error("AGENTS.md must not contain contributorModeMarker after disable")
	}
}

func TestManagedDocsPreservesContributorMode(t *testing.T) {
	canonical := `# AGENTS.md

## Context
<!-- pose:instance-owned -->
Instance context.

## Workflow
Engine workflow.
`
	local := `# AGENTS.md

## Context
<!-- pose:instance-owned -->
Instance context custom.

## Open-Source POSE Contributor Mode
<!-- pose:contributor-mode -->
Contributor mode active instructions.
`

	merged, preserved := MergeManagedDoc(canonical, local)
	if !preserved {
		t.Fatal("expected preserved to be true")
	}
	if !strings.Contains(merged, contributorModeMarker) {
		t.Errorf("merged doc must preserve contributorModeMarker, got:\n%s", merged)
	}
	if !strings.Contains(merged, "Contributor mode active instructions.") {
		t.Errorf("merged doc must preserve contributor body, got:\n%s", merged)
	}
}

func TestContributeStageAndList(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := cmdContribute(root, []string{"stage", "--title", "Missing Zig Rule Extension", "--type", "enhancement", "--body", "Zig is not detected in discovery."}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("stage returned code %d, stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	code = cmdContribute(root, []string{"list", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("list returned code %d", code)
	}
	if !strings.Contains(stdout.String(), "Missing Zig Rule Extension") {
		t.Errorf("expected staged item in json list, got: %s", stdout.String())
	}
}

func TestContributePrivacyEnforcement(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := cmdContribute(root, []string{"status", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status failed: %d", code)
	}
	if !strings.Contains(stdout.String(), "No proprietary code") {
		t.Errorf("expected privacy rule in status output, got: %s", stdout.String())
	}
}
