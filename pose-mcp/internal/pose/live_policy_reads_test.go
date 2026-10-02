// The bundle validator reads exactly one thing from the live policy
// (spec pose-only-the-signing-gate-is-read-live).
//
// Everything a sealed bundle is judged by is sealed with it, so that a setting
// flipped today cannot re-judge a review recorded years ago. One gate is
// deliberately not: `require_signed_attestations` is a bar rather than a
// permission, and a bundle sealed before a project started requiring signatures
// must not be permanently exempt.
//
// That exception is one line of code and one paragraph of ADR. Sealing it by
// symmetry with the others would look like tidying and would quietly exempt
// every existing bundle from a security requirement — so the count is asserted
// rather than remembered.

package pose

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// livePolicyFields follows identifier bindings, so aliases cannot hide reads
// and a shadowed variable with the same name cannot invent them.
func livePolicyFields(source string) (map[string]bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "validator.go", "package pose\n"+source, 0)
	if err != nil {
		return nil, err
	}
	aliases := map[*ast.Object]bool{}
	var baseObject func(ast.Expr) *ast.Object
	baseObject = func(expr ast.Expr) *ast.Object {
		switch value := expr.(type) {
		case *ast.Ident:
			return value.Obj
		case *ast.ParenExpr:
			return baseObject(value.X)
		case *ast.StarExpr:
			return baseObject(value.X)
		case *ast.UnaryExpr:
			if value.Op == token.AND {
				return baseObject(value.X)
			}
		}
		return nil
	}
	isPolicy := func(expr ast.Expr) bool {
		if call, ok := expr.(*ast.CallExpr); ok {
			if method, ok := call.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "loadReviewPolicy" {
				return true
			}
		}
		object := baseObject(expr)
		return object != nil && aliases[object]
	}
	changed := true
	for changed {
		changed = false
		mark := func(left ast.Expr, right ast.Expr) {
			object := baseObject(left)
			if object != nil && !aliases[object] && isPolicy(right) {
				aliases[object] = true
				changed = true
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.AssignStmt:
				for i, right := range value.Rhs {
					if i < len(value.Lhs) {
						mark(value.Lhs[i], right)
					}
				}
			case *ast.ValueSpec:
				for i, right := range value.Values {
					if i < len(value.Names) {
						mark(value.Names[i], right)
					}
				}
			}
			return true
		})
	}
	seen := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && isPolicy(selector.X) {
			seen[selector.Sel.Name] = true
		}
		return true
	})
	return seen, nil
}

// validateBundleAttestationWithBody returns the source of the function that
// judges an attestation against a sealed bundle.
func validateBundleAttestationWithBody(t *testing.T) string {
	t.Helper()
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "review_bundle.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	found := false
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || (fn.Name.Name != "validateBundleAttestationWith" && fn.Name.Name != "reviewFindingDecisionBlockers") {
			continue
		}
		if fn.Name.Name == "validateBundleAttestationWith" {
			found = true
		}
		if err := format.Node(&body, set, fn); err != nil {
			t.Fatal(err)
		}
		body.WriteString("\n")
	}
	if !found {
		t.Fatal("validateBundleAttestationWith not found")
	}
	return body.String()
}

func TestOnlyTheSigningGateIsReadFromLivePolicy(t *testing.T) {
	body := validateBundleAttestationWithBody(t)
	seen, err := livePolicyFields(body)
	if err != nil {
		t.Fatal(err)
	}
	if !seen["RequireSignedAttestations"] {
		t.Error("the signing requirement is no longer read from the live policy — a bundle sealed before a project started requiring signatures is now permanently exempt from it")
	}
	delete(seen, "RequireSignedAttestations")
	for field := range seen {
		t.Errorf("policy.%s is read live while judging a sealed bundle: seal it into the bundle's gates, or record here why it is the second exception", field)
	}
}

// The other side of the same contract: the gates a bundle carries are the ones
// the validator consults. A gate added to the struct and never read is a
// setting that seals and does nothing.
func TestEverySealedGateIsConsulted(t *testing.T) {
	body := validateBundleAttestationWithBody(t)
	for _, gate := range []string{"AllowApprovedWithReservations", "AcceptedRiskSeverities", "AllowCriterionReuse"} {
		if !strings.Contains(body, "SealedGates()."+gate) {
			t.Errorf("%s is sealed into the bundle and never consulted while judging one", gate)
		}
	}
}

func TestLivePolicyReadsFollowAliasesAndIgnoreText(t *testing.T) {
	source := `func validate(s Store) {
 policy, _, _ := s.loadReviewPolicy()
 first := policy
 var second = &first
 _ = second.AllowCriterionReuse
 _ = policy.RequireSignedAttestations
 _ = "policy.FakeString"
 // policy.FakeComment
 { policy := unrelated{}; _ = policy.Shadowed }
 }`
	fields, err := livePolicyFields(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || !fields["AllowCriterionReuse"] || !fields["RequireSignedAttestations"] {
		t.Fatalf("fields: %v", fields)
	}
	if _, err := livePolicyFields("func broken("); err == nil {
		t.Fatal("malformed source accepted")
	}
}
