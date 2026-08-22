package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestContributorModeHints(t *testing.T) {
	temp := t.TempDir()

	// Initially inactive
	if IsContributorModeActive(temp) {
		t.Fatal("expected contributor mode to be inactive initially")
	}

	var buf bytes.Buffer
	PrintContributorFailureHint(temp, &buf, "en")
	if buf.Len() > 0 {
		t.Fatalf("expected no failure hint when inactive, got %q", buf.String())
	}

	buf.Reset()
	PrintContributorDoctorHint(temp, &buf, "en")
	if buf.Len() > 0 {
		t.Fatalf("expected no doctor hint when inactive, got %q", buf.String())
	}

	// Enable contributor mode
	var out, errBuf bytes.Buffer
	code := cmdContributeEnable(temp, nil, &out, &errBuf, "en")
	if code != 0 {
		t.Fatalf("cmdContributeEnable failed: %s", errBuf.String())
	}

	if !IsContributorModeActive(temp) {
		t.Fatal("expected contributor mode to be active after enable")
	}

	// Test failure hint in English
	buf.Reset()
	PrintContributorFailureHint(temp, &buf, "en")
	if !strings.Contains(buf.String(), "[Contributor Mode ACTIVE]") || !strings.Contains(buf.String(), "pose contribute stage --type bug") {
		t.Fatalf("unexpected English failure hint: %s", buf.String())
	}

	// Test failure hint in Portuguese
	buf.Reset()
	PrintContributorFailureHint(temp, &buf, "pt-BR")
	if !strings.Contains(buf.String(), "[Modo Contribuidor ATIVO]") || !strings.Contains(buf.String(), "pose contribute stage --type bug") {
		t.Fatalf("unexpected Portuguese failure hint: %s", buf.String())
	}

	// Test doctor hint in English
	buf.Reset()
	PrintContributorDoctorHint(temp, &buf, "en")
	if !strings.Contains(buf.String(), "[INFO] Contributor Mode: ACTIVE") {
		t.Fatalf("unexpected English doctor hint: %s", buf.String())
	}

	// Test doctor hint in Portuguese
	buf.Reset()
	PrintContributorDoctorHint(temp, &buf, "pt-BR")
	if !strings.Contains(buf.String(), "[INFO] Modo Contribuidor: ATIVO") {
		t.Fatalf("unexpected Portuguese doctor hint: %s", buf.String())
	}
}

func TestReportLimitationStagesContributionWhenActive(t *testing.T) {
	temp := t.TempDir()

	// Enable contributor mode
	var out, errBuf bytes.Buffer
	if code := cmdContributeEnable(temp, nil, &out, &errBuf, "en"); code != 0 {
		t.Fatalf("enable failed: %s", errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	args := []string{"--title", "Missing Zig Stack Rules", "--kind", "suggestion", "--body", "Add rules for Zig toolchain"}
	if code := cmdReportLimitation(temp, args, &out, &errBuf); code != 0 {
		t.Fatalf("cmdReportLimitation failed: %s", errBuf.String())
	}

	// Check that feedback was saved in .pose/feedback and .pose/contributions
	contribs, err := listStagedContributions(temp)
	if err != nil {
		t.Fatalf("listStagedContributions: %v", err)
	}
	if len(contribs) != 1 {
		t.Fatalf("expected 1 staged contribution, got %d", len(contribs))
	}
	if contribs[0].Title != "Missing Zig Stack Rules" {
		t.Fatalf("expected title 'Missing Zig Stack Rules', got %q", contribs[0].Title)
	}
}

func TestMergeManagedDocPreservesContributorMode(t *testing.T) {
	canonical := `# POSE.md — Project Operating Standard

## Project context
<!-- pose:instance-owned -->
Canonical context.

## 6) The pose CLI
Canonical CLI info.
`
	local := `# POSE.md — Project Operating Standard

## Project context
<!-- pose:instance-owned -->
My custom project context.

## 6) The pose CLI
Canonical CLI info.

## Open-Source POSE Contributor Mode
<!-- pose:contributor-mode -->
**Contributor Mode is ACTIVE.**
`
	merged, preserved := MergeManagedDoc(canonical, local)
	if !preserved {
		t.Fatal("expected preserved to be true")
	}
	if !strings.Contains(merged, "## Open-Source POSE Contributor Mode") {
		t.Fatalf("expected merged doc to contain contributor mode section, got:\n%s", merged)
	}
	if !strings.Contains(merged, "My custom project context.") {
		t.Fatalf("expected merged doc to contain custom instance context, got:\n%s", merged)
	}
}

