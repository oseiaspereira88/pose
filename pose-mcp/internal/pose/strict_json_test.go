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
			strict, ends := strictJSONReaderEnds(fn)
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

// strictJSONReaderEnds reports whether fn refuses unknown JSON fields, and
// whether it checks the end of its input: a comparison with io.EOF (==, !=
// or errors.Is), a call of DecodeStrictJSON, or json.Unmarshal of the input,
// which refuses anything after the value. A decode or a More/Token call
// that is not compared with io.EOF is no such check (review round 1, F1).
func strictJSONReaderEnds(fn *ast.FuncDecl) (strict, ends bool) {
	isEOF := func(e ast.Expr) bool {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return ok && pkg.Name == "io" && sel.Sel.Name == "EOF"
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (isEOF(n.X) || isEOF(n.Y)) {
				ends = true
			}
		case *ast.CallExpr:
			switch fun := n.Fun.(type) {
			case *ast.SelectorExpr:
				pkg, _ := fun.X.(*ast.Ident)
				switch {
				case fun.Sel.Name == "DisallowUnknownFields":
					strict = true
				case fun.Sel.Name == "DecodeStrictJSON":
					ends = true
				case pkg != nil && pkg.Name == "json" && fun.Sel.Name == "Unmarshal":
					ends = true
				case pkg != nil && pkg.Name == "errors" && fun.Sel.Name == "Is" && len(n.Args) == 2 && isEOF(n.Args[1]):
					ends = true
				}
			case *ast.Ident:
				if fun.Name == "DecodeStrictJSON" {
					ends = true
				}
			}
		}
		return true
	})
	return strict, ends
}

// Review round 1 (F1): the guard's predicate rejects readers that decode
// without checking the end, and accepts each form that checks it.
func TestStrictJSONReaderEndsJudgesTheCheckNotItsShape(t *testing.T) {
	const header = "package p\nimport (\"encoding/json\"; \"errors\"; \"io\"; \"bytes\")\nvar _ = errors.Is\nvar _ = bytes.NewReader\nvar _ = io.EOF\nvar _ = json.Unmarshal\n"
	for name, c := range map[string]struct {
		body string
		ends bool
	}{
		"decode of a new value only": {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); return dec.Decode(new(map[string]any))", false},
		"More as the end check":      {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); var v any; if err := dec.Decode(&v); err != nil { return err }; if dec.More() { return errors.New(\"x\") }; return nil", false},
		"Token not compared":         {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); var v any; if err := dec.Decode(&v); err != nil { return err }; _, err := dec.Token(); return err", false},
		"io.EOF mentioned only":      {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); var v any; e := io.EOF; _ = e; return dec.Decode(&v)", false},
		"second decode compared":     {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); var v any; if err := dec.Decode(&v); err != nil { return err }; if err := dec.Decode(new(any)); err != io.EOF { return errors.New(\"x\") }; return nil", true},
		"Token with errors.Is":       {"dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); var v any; if err := dec.Decode(&v); err != nil { return err }; if _, err := dec.Token(); !errors.Is(err, io.EOF) { return errors.New(\"x\") }; return nil", true},
		"Unmarshal first":            {"var v any; if err := json.Unmarshal(raw, &v); err != nil { return err }; dec := json.NewDecoder(bytes.NewReader(raw)); dec.DisallowUnknownFields(); return dec.Decode(&v)", true},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), "p.go", header+"func read(raw []byte) error { "+c.body+" }\n", 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var fn *ast.FuncDecl
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok {
				fn = f
			}
		}
		strict, ends := strictJSONReaderEnds(fn)
		if !strict || ends != c.ends {
			t.Errorf("%s: strict=%v ends=%v, want ends=%v", name, strict, ends, c.ends)
		}
	}
}
