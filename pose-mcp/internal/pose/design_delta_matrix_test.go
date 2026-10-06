package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validationMatrixBase = `{
  "defaults": {"mode": "strict"},
  "stacks": {"go": {"checks": [
    {"name": "build", "program": "go", "args": ["build", "./..."], "severity": "required"},
    {"name": "test", "program": "go", "args": ["test", "./..."], "severity": "required"}
  ]}},
  "moduleOverrides": {"svc": {"stack": "go", "checks": [
    {"name": "svc-integration", "program": "go", "args": ["test", "./...", "-run", "Integration"], "severity": "required", "evidenceClass": "integration"}
  ]}}
}
`

// matrixDeltas commits the base matrix, then the head (or removes the file when
// head is empty and action says so), and returns the observed deltas.
func matrixDeltas(t *testing.T, base, head, action string) []StructuralDelta {
	t.Helper()
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeDesignDeltaFile(t, root, "README.md", "demo\n")
	if base != "" {
		writeDesignDeltaFile(t, root, ".pose/indexes/validation-matrix.json", base)
	}
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "base")
	baseRev := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	writeDesignDeltaFile(t, root, ".pose/indexes/validation-matrix.json", head)
	designDeltaGit(t, root, "add", "--", ".")
	headRev := designDeltaGit(t, root, "commit", "-q", "-m", "head")
	subject := ReviewBundleSubject{Base: baseRev, Head: headRev, Entries: []ReviewBundleSubjectEntry{
		// A real subject carries the entry digest, and the delta cache is keyed
		// on it; without one every fixture here would share a cache entry.
		{Action: action, Path: ".pose/indexes/validation-matrix.json", Class: "governance", Digest: digestBytes([]byte(head))},
	}}
	report, err := AssessDesignDelta(root, subject, "spec:demo", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return report.Deltas
}

func materialMatrixFacts(deltas []StructuralDelta) (material []StructuralDelta, additions []StructuralDelta) {
	for _, delta := range deltas {
		if StructuralDeltaIsMaterial(delta) {
			material = append(material, delta)
		}
		if delta.Kind == "validation-check" {
			additions = append(additions, delta)
		}
	}
	return material, additions
}

func TestValidationCheckMaterialityReportsNewChecksWithoutCharging(t *testing.T) {
	head := strings.Replace(validationMatrixBase,
		`"severity": "required", "evidenceClass": "integration"}`,
		`"severity": "required", "evidenceClass": "integration"},
    {"name": "svc-cutoff-integration", "program": "go", "args": ["test", "./...", "-run", "Cutoff"], "severity": "required", "evidenceClass": "integration"}`, 1)
	head = strings.Replace(head,
		`{"name": "test", "program": "go", "args": ["test", "./..."], "severity": "required"}`,
		`{"name": "test", "program": "go", "args": ["test", "./..."], "severity": "required"},
    {"name": "vet", "program": "go", "args": ["vet", "./..."], "severity": "required"}`, 1)
	material, additions := materialMatrixFacts(matrixDeltas(t, validationMatrixBase, head, "modified"))
	if len(material) != 0 {
		t.Fatalf("adding checks produced material facts: %+v", material)
	}
	subjects := map[string]bool{}
	for _, delta := range additions {
		if delta.Action != "added" || delta.State != "observed" || delta.Path != ".pose/indexes/validation-matrix.json" {
			t.Fatalf("unexpected addition shape: %+v", delta)
		}
		subjects[delta.Subject] = true
	}
	if len(additions) != 2 || !subjects["moduleOverrides.svc:svc-cutoff-integration"] || !subjects["stacks.go:vet"] {
		t.Fatalf("want one reported fact per added check, got %+v", additions)
	}
}

func TestValidationCheckMaterialityKeepsEveryOtherChangeMaterial(t *testing.T) {
	addCheck := func(name string) string {
		return strings.Replace(validationMatrixBase,
			`"severity": "required", "evidenceClass": "integration"}`,
			`"severity": "required", "evidenceClass": "integration"},
    {"name": "`+name+`", "program": "true", "severity": "required", "evidenceClass": "integration"}`, 1)
	}
	cases := map[string]struct {
		base, head, action string
	}{
		"added check reuses an existing name": {validationMatrixBase, addCheck("test"), "modified"},
		"removed check": {validationMatrixBase, strings.Replace(validationMatrixBase,
			`,
    {"name": "test", "program": "go", "args": ["test", "./..."], "severity": "required"}`, "", 1), "modified"},
		"edited check":                      {validationMatrixBase, strings.Replace(validationMatrixBase, `"-run", "Integration"`, `"-run", "Nothing"`, 1), "modified"},
		"mode downgrade beside an addition": {validationMatrixBase, strings.Replace(addCheck("svc-new"), `"mode": "strict"`, `"mode": "tolerant"`, 1), "modified"},
		"added module override": {validationMatrixBase, strings.Replace(validationMatrixBase, `"moduleOverrides": {`,
			`"moduleOverrides": {"web": {"stack": "go", "replaceDefaultChecks": true, "checks": []}, `, 1), "modified"},
		"created matrix":      {"", validationMatrixBase, "created"},
		"unparsable new side": {validationMatrixBase, "{not json", "modified"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			material, additions := materialMatrixFacts(matrixDeltas(t, tc.base, tc.head, tc.action))
			if len(additions) != 0 {
				t.Fatalf("non-additive change reported as additions: %+v", additions)
			}
			if len(material) != 1 || material[0].Kind != "delivery-metadata" || material[0].Subject != ".pose/indexes/validation-matrix.json" {
				t.Fatalf("want the matrix as one material delivery-metadata fact, got %+v", material)
			}
		})
	}
}

func TestValidationCheckMaterialityChangesTheParserVersion(t *testing.T) {
	if DesignDeltaParserVersion == "pose-design-delta/v1" {
		t.Fatal("the refined matrix reading must not reuse the v1 parser version")
	}
}

func TestValidationCheckMaterialityIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "`validation-check`") {
			t.Fatalf("%s does not document that an added validation check is reported and not charged", rel)
		}
	}
}
