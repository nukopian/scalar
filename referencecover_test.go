package scalar

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// coveredElsewhere are exported functions and methods the reference
// describes under another's name, as Type.Method or Func, each with
// where it's covered. The test fails on one listed that the reference
// names after all, so the list doesn't outlive its reasons.
var coveredElsewhere = map[string]string{}

// The reference names every exported function and method in code, so
// that none goes undocumented there (roadmap 2026-0009). A method counts
// once its name appears on any type, as the reference describes Get once
// for scalars, arrays and hashes.
func TestReferenceNamesEveryFunction(t *testing.T) {
	text, err := os.ReadFile("docs/REFERENCE.md")
	if err != nil {
		t.Fatal(err)
	}
	code := markdownCode(string(text))
	for _, name := range exportedFuncs(t, ".") {
		_, listed := coveredElsewhere[name]
		named := namesIn(code, name)
		switch {
		case named && listed:
			t.Errorf("%s is in coveredElsewhere, but the reference names it: take it out of the list", name)
		case !named && !listed:
			t.Errorf("docs/REFERENCE.md doesn't name %s in code: describe it in its feature's section, "+
				"or add it to coveredElsewhere saying where it's covered", name)
		}
	}
}

// The test catches a function the reference doesn't name: a planted one
// fails, and one it names passes.
func TestReferenceNamesEveryFunctionCatches(t *testing.T) {
	dir := t.TempDir()
	src := "package p\n\nfunc NeverDocumentedAnywhere() {}\n\ntype T struct{}\n\nfunc (T) Get() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "p.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile("docs/REFERENCE.md")
	if err != nil {
		t.Fatal(err)
	}
	code := markdownCode(string(text))
	var missing []string
	for _, name := range exportedFuncs(t, dir) {
		if !namesIn(code, name) {
			missing = append(missing, name)
		}
	}
	if !slices.Equal(missing, []string{"NeverDocumentedAnywhere"}) {
		t.Errorf("missing = %v, want only the planted NeverDocumentedAnywhere", missing)
	}
}

// namesIn says whether code names the function or method name, given as
// Func or Type.Method, as a whole word; a method counts on any type.
func namesIn(code, name string) bool {
	name = name[strings.LastIndex(name, ".")+1:]
	return regexp.MustCompile(`\b` + name + `\b`).MatchString(code)
}

// exportedFuncs is the exported functions and methods of the package in
// dir, as Func or Type.Method, sorted; methods of unexported types are
// left out, as they aren't part of the API.
func exportedFuncs(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var out []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() {
				continue
			}
			name := fn.Name.Name
			if fn.Recv != nil {
				typ := fn.Recv.List[0].Type
				if star, ok := typ.(*ast.StarExpr); ok {
					typ = star.X
				}
				if index, ok := typ.(*ast.IndexExpr); ok {
					typ = index.X
				}
				ident, ok := typ.(*ast.Ident)
				if !ok || !ident.IsExported() {
					continue
				}
				name = ident.Name + "." + name
			}
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

var (
	fencedCode = regexp.MustCompile("(?ms)^\\s*```[^\\n]*\\n.*?^\\s*```")
	codeSpan   = regexp.MustCompile("`[^`\\n]+`")
)

// markdownCode is the text of md's code blocks and code spans, one per
// line, so that a word in prose doesn't count as a name.
func markdownCode(md string) string {
	var b strings.Builder
	for _, block := range fencedCode.FindAllString(md, -1) {
		b.WriteString(block)
		b.WriteByte('\n')
	}
	for _, span := range codeSpan.FindAllString(fencedCode.ReplaceAllString(md, ""), -1) {
		b.WriteString(span)
		b.WriteByte('\n')
	}
	return b.String()
}
