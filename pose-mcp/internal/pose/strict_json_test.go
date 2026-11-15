package pose

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-strict-json-readers-require-the-end, R1: one value and nothing
// after it.
func TestDecodeStrictJSONReadsOneValueAndNothingElse(t *testing.T) {
	type policy struct {
		Name string `json:"name"`
	}
	for name, raw := range map[string]string{
		"object":             `{"name":"a"}`,
		"surrounding spaces": " \n{\"name\":\"a\"}\n\t ",
	} {
		var p policy
		if err := DecodeStrictJSON([]byte(raw), &p); err != nil || p.Name != "a" {
			t.Fatalf("%s: %v %+v", name, err, p)
		}
	}
	for name, raw := range map[string]string{
		"garbage after":  `{"name":"a"} garbage`,
		"second object":  `{"name":"a"}{"name":"b"}`,
		"second value":   `{"name":"a"} 1`,
		"unknown field":  `{"name":"a","shell":true}`,
		"empty":          ``,
		"truncated":      `{"name":"a"`,
		"closing excess": `{"name":"a"}}`,
	} {
		var p policy
		if err := DecodeStrictJSON([]byte(raw), &p); err == nil {
			t.Fatalf("%s was read: %+v", name, p)
		}
	}
}

// strictJSONReadersOutsideTheHelper names the functions that refuse unknown
// JSON fields without DecodeStrictJSON, each with how it refuses content
// after the value, or why it need not. A syntactic test of the end check was
// tried and bypassed twice (review rounds 1 and 2: F1, N1); a reviewed list
// is the predicate this guard can measure exactly (spec
// pose-strict-json-readers-require-the-end, D2).
var strictJSONReadersOutsideTheHelper = map[string]string{
	"internal/cli/spec_transfer.go#decodeTransferPlan":                         "a second decode of the input must return io.EOF",
	"internal/cli/spec_transfer.go#readTransferMapFile":                        "a second decode of the input must return io.EOF",
	"internal/mcpserver/server.go#dispatch":                                    "tool arguments are one value the JSON-RPC transport already parsed",
	"internal/pose/action_scoping.go#ParseActionEffect":                        "a second decode of the input must return io.EOF",
	"internal/pose/closeout_receipt.go#parseCloseoutReceipt":                   "a second decode of the input must return io.EOF",
	"internal/pose/delegation_policy.go#ParseDelegationPolicy":                 "a second decode of the input must return io.EOF",
	"internal/pose/federated_acceptance.go#strictJSONBytes":                    "a second decode of the input must return io.EOF",
	"internal/pose/index_generation.go#readIndexDescriptor":                    "a Token read after the value must return io.EOF",
	"internal/pose/journal_integrity.go#decodeActionEvent":                     "a second decode of the input must return io.EOF",
	"internal/pose/release_manifest_integrity.go#ValidateReleaseManifestBytes": "the field decode of the same bytes before it must end in io.EOF",
	"internal/pose/review_adapter_containment.go#UnmarshalJSON":                "json.Unmarshal of the same bytes before it refuses content after the value",
	"internal/pose/review_bundle.go#strictJSONFileIn":                          "a second decode of the input must return io.EOF",
	"internal/pose/review_closeout.go#parseReviewProfile":                      "json.Unmarshal of the same bytes before it refuses content after the value",
	"internal/pose/review_rounds.go#ParseReviewDraft":                          "a Token read after the value must return io.EOF",
	"internal/pose/spec_transfer.go#readSpecTransferPlan":                      "a second decode of the input must return io.EOF",
	"internal/pose/spec_transfer.go#readSpecTransferRedirect":                  "a second decode of the input must return io.EOF",
}

// Spec pose-strict-json-readers-require-the-end, R2: a function of the
// engine refuses unknown fields only through DecodeStrictJSON, or is named
// here with how it requires the end.
func TestStrictJSONReadersRequireTheEnd(t *testing.T) {
	base := filepath.Join("..", "..")
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(base, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(base, path)
		rel = filepath.ToSlash(rel)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			strict := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "DisallowUnknownFields" {
						strict = true
					}
				}
				return true
			})
			site := rel + "#" + fn.Name.Name
			if !strict || site == "internal/pose/strict_json.go#DecodeStrictJSON" {
				continue
			}
			seen[site] = true
			if strictJSONReadersOutsideTheHelper[site] == "" {
				t.Errorf("%s refuses unknown fields outside DecodeStrictJSON: use it, or name here how the end is required", site)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for site := range strictJSONReadersOutsideTheHelper {
		if !seen[site] {
			t.Errorf("%s is listed but no longer refuses unknown fields", site)
		}
	}
}
