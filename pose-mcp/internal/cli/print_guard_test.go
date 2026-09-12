package cli

// The ratchet (spec pose-cli-output-rendering-system R2).
//
// Output was formatted at 1138 call sites with nothing in between, which is why
// three severity dialects and two channel habits coexisted. A convention would
// drift back within a release, so the constraint is a test: every direct
// fmt.Fprint* in this package is counted, and the counts may only go down.
//
// Migrating a command to the renderer lowers its count; the baseline is then
// updated to match, and the number can never climb again. A new file may not
// print directly at all.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const printBaselinePath = "testdata/direct-print-sites.json"

func countDirectPrints(t *testing.T) map[string]int {
	t.Helper()
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	fset := token.NewFileSet()
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" {
				return true
			}
			switch selector.Sel.Name {
			case "Fprint", "Fprintf", "Fprintln":
				counts[path]++
			}
			return true
		})
	}
	return counts
}

func TestDirectPrintSitesOnlyShrink(t *testing.T) {
	raw, err := os.ReadFile(printBaselinePath)
	if err != nil {
		t.Fatalf("%s: %v", printBaselinePath, err)
	}
	var baseline map[string]int
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	counts := countDirectPrints(t)

	files := map[string]bool{}
	for name := range baseline {
		files[name] = true
	}
	for name := range counts {
		files[name] = true
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		got, want := counts[name], baseline[name]
		switch {
		case got > want:
			t.Errorf("%s prints directly %d time(s), baseline %d: emit through cliout instead, or the layer erodes", name, got, want)
		case got < want:
			t.Errorf("%s prints directly %d time(s), baseline %d: lower the baseline in %s so the ratchet holds", name, got, want, printBaselinePath)
		}
	}

	total := 0
	for _, count := range counts {
		total += count
	}
	t.Logf("direct print sites remaining: %d across %d file(s)", total, len(counts))
}
