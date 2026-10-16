package pose

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Harne8's Harness vendors this package's production files as generated
// copies (scripts/sync-pose-sources.sh there). Go forbids importing another
// module's internal packages, so a production import of a sibling
// pose-mcp/internal package makes that copy uncompilable; the engine version
// reaches this package through EngineVersion instead
// (spec pose-engine-package-stays-mirrorable).
func TestProductionFilesImportNoSiblingInternalPackage(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	const sibling = "github.com/harne8/pose-mcp/internal/"
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		checked++
		for _, spec := range file.Imports {
			if path := strings.Trim(spec.Path.Value, `"`); strings.HasPrefix(path, sibling) {
				t.Errorf("%s imports %s; internal/pose must stay importable on its own", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no production files were checked")
	}
}
