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

// strictJSONWithoutTheEnd names the functions that refuse unknown fields
// without requiring the end of the input, with the reason it holds there.
var strictJSONWithoutTheEnd = map[string]string{
	"internal/mcpserver/server.go#dispatch": "tool arguments are one value the JSON-RPC transport already parsed",
}

// Spec pose-strict-json-readers-require-the-end, R2: a function of the
// engine that refuses unknown fields also refuses content after the value,
// or names here why it need not.
func TestStrictJSONReadersRequireTheEnd(t *testing.T) {
	base := filepath.Join("..", "..")
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(base, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
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
			strict, ends := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				// io.EOF compared after a decode, or json.Unmarshal of the
				// same input, which refuses anything after the value.
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if pkg, ok := sel.X.(*ast.Ident); ok && ((pkg.Name == "io" && sel.Sel.Name == "EOF") || (pkg.Name == "json" && sel.Sel.Name == "Unmarshal")) {
						ends = true
					}
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch sel := call.Fun.(type) {
				case *ast.SelectorExpr:
					switch sel.Sel.Name {
					case "DisallowUnknownFields":
						strict = true
					case "Token", "More", "DecodeStrictJSON":
						ends = true
					case "Decode":
						if len(call.Args) == 1 {
							if inner, ok := call.Args[0].(*ast.CallExpr); ok {
								if id, ok := inner.Fun.(*ast.Ident); ok && id.Name == "new" {
									ends = true
								}
							}
						}
					}
				case *ast.Ident:
					if sel.Name == "DecodeStrictJSON" {
						ends = true
					}
				}
				return true
			})
			if !strict {
				continue
			}
			site := rel + "#" + fn.Name.Name
			seen[site] = true
			if !ends && strictJSONWithoutTheEnd[site] == "" {
				t.Errorf("%s refuses unknown fields but reads past the value: use DecodeStrictJSON, or name here why the end need not be required", site)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for site := range strictJSONWithoutTheEnd {
		if !seen[site] {
			t.Errorf("%s is listed but no longer refuses unknown fields", site)
		}
	}
}
