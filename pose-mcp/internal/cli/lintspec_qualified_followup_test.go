package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-followup-dispositions-accept-qualified-refs.

func qualifiedFollowupFixture(t *testing.T, target string) (string, string, string) {
	t.Helper()
	local, other := t.TempDir(), t.TempDir()
	write := func(root, rel, body string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(other, ".pose/specs/2026-10-01-covering.md", "---\nslug: covering\nstatus: draft\ncreated_at: 2026-10-01\n---\n\n# Spec: covering\n")
	source := ".pose/specs/2026-10-01-source.md"
	write(local, source, "---\nslug: source\nstatus: done\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\n---\n\n# Spec: source\n\n## 7. Final Report\n\n### Follow-ups\n\n- ["+target+"] Owned by the other project.\n")
	return local, other, filepath.Join(local, filepath.FromSlash(source))
}

func lintQualifiedFollowup(t *testing.T, target string, bind bool) string {
	t.Helper()
	local, other, _ := qualifiedFollowupFixture(t, target)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.local")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	if bind {
		t.Setenv("POSE_PROJECT_ROOTS", `{"proj.other":"`+strings.ReplaceAll(other, `\`, `\\`)+`"}`)
	}
	var out, errB bytes.Buffer
	inDir(t, local, func() {
		Main([]string{"lint-spec", "source"}, &out, &errB)
	})
	output := out.String() + errB.String()
	if !strings.Contains(output, "lint.specs.checked=1") {
		t.Fatalf("the fixture spec was not linted, so nothing below is evidence:\n%s", output)
	}
	return output
}

func TestQualifiedFollowupIsAcceptedWhenTheProjectIsNotBound(t *testing.T) {
	if output := lintQualifiedFollowup(t, "covered: xref:proj.other/spec:covering", false); strings.Contains(output, "follow-up lacks a valid disposition") {
		t.Fatalf("a qualified disposition for an unbound project was refused:\n%s", output)
	}
}

func TestQualifiedFollowupIsVerifiedWhenTheProjectIsBound(t *testing.T) {
	if output := lintQualifiedFollowup(t, "covered: xref:proj.other/spec:covering", true); strings.Contains(output, "follow-up lacks a valid disposition") {
		t.Fatalf("a qualified disposition naming an existing spec was refused:\n%s", output)
	}
	output := lintQualifiedFollowup(t, "duplicate: xref:proj.other/spec:absent", true)
	if !strings.Contains(output, "follow-up lacks a valid disposition") || !strings.Contains(output, "unknown-spec") {
		t.Fatalf("a qualified disposition naming a spec missing in a bound project was accepted:\n%s", output)
	}
}

func TestQualifiedFollowupRefusesMalformedTargets(t *testing.T) {
	for _, target := range []string{"covered: xref:proj.other/roadmap:covering", "spawned: xref:not a ref"} {
		output := lintQualifiedFollowup(t, target, false)
		if !strings.Contains(output, "follow-up lacks a valid disposition") {
			t.Fatalf("%q was accepted:\n%s", target, output)
		}
	}
}

func TestQualifiedFollowupIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "[covered: xref:") {
			t.Fatalf("%s does not document qualified follow-up dispositions", rel)
		}
	}
}
