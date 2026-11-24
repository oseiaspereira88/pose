package pose

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// Spec pose-assess-leaves-sealed-checkouts-alone R3: the review plan runs
// every assessment tool in the form that writes nothing.
func TestReviewPlanRunsAssessmentsWithoutPersisting(t *testing.T) {
	scope := ScopeRef{Kind: "spec", Slug: "fixed"}
	for _, id := range []string{"assess-discover", "assess-integrate", "assess-tech-debt"} {
		for _, component := range []string{"", "core"} {
			args := strings.Join(reviewToolArgs(id, scope, component), " ")
			if !strings.Contains(args, "--no-persist") {
				t.Fatalf("%s (%q) plans a persisting run: %s", id, component, args)
			}
		}
	}
}

// Review round 1 (N1): a writer without the guard could still publish into a
// sealed checkout. Every function in the assessment files that writes a file
// must call the guard first.
func TestEveryAssessmentWriterRefusesASealedCheckout(t *testing.T) {
	for _, file := range []string{"discovery.go", "integration.go", "techdebt.go"} {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			writes, guarded := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch f := call.Fun.(type) {
				case *ast.SelectorExpr:
					if id, ok := f.X.(*ast.Ident); ok && id.Name == "os" && (f.Sel.Name == "WriteFile" || f.Sel.Name == "Create" || f.Sel.Name == "MkdirAll" || f.Sel.Name == "OpenFile") {
						writes = true
					}
				case *ast.Ident:
					if f.Name == "refuseSealedAssessmentWrite" {
						guarded = true
					}
				}
				return true
			})
			if writes && !guarded {
				t.Errorf("%s: %s writes without refusing a sealed checkout", file, fn.Name.Name)
			}
		}
	}
}
