package scalar

// Every exported function and method has an example, and the examples are
// the Reference section of README.md. The tests here keep both true:
//
//	go test ./internal/scalar                                  # checks
//	go test ./internal/scalar -run TestReference -update       # rewrites the README section

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the Reference section of README.md from the examples")

// referenceGroups are the example files, in the order and under the
// headings they appear in the Reference.
var referenceGroups = []struct{ file, title string }{
	{"example_sv_test.go", "Scalars"},
	{"example_str_test.go", "Strings"},
	{"example_ops_test.go", "Operators"},
	{"example_av_test.go", "Arrays"},
	{"example_hv_test.go", "Hashes"},
	{"example_list_test.go", "List utilities"},
	{"example_regex_test.go", "Regular expressions"},
	{"example_json_test.go", "JSON and JSONPath"},
}

const (
	referenceStart = "<!-- reference:start -->"
	referenceEnd   = "<!-- reference:end -->"
)

// api is the package's exported functions and methods, as example names
// spell them ("Scalar", "SV_Substr"), with the first sentence of each one's
// doc comment.
func api(t *testing.T) map[string]string {
	t.Helper()
	paths, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
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
				name = typ.(*ast.Ident).Name + "_" + name
			}
			out[name] = new(doc.Package).Synopsis(fn.Doc.Text())
		}
	}
	return out
}

// examples reads the examples in file, in the order they're written.
func examples(t *testing.T, file string) ([]*doc.Example, *token.FileSet, []byte) {
	t.Helper()
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	exs := doc.Examples(f)
	slices.SortStableFunc(exs, func(a, b *doc.Example) int { return int(a.Code.Pos() - b.Code.Pos()) })
	return exs, fset, src
}

// Every exported function and method has at least one example, in one of
// the reference files.
func TestEveryFeatureHasExample(t *testing.T) {
	have := map[string]bool{}
	for _, g := range referenceGroups {
		exs, _, _ := examples(t, g.file)
		for _, ex := range exs {
			have[ex.Name] = true
		}
	}
	for name := range api(t) {
		if !have[name] {
			t.Errorf("%s has no example: add Example%s to one of the example_*_test.go files", strings.Replace(name, "_", ".", 1), name)
		}
	}
}

// The Reference section of README.md is exactly what the examples generate.
func TestReference(t *testing.T) {
	want := reference(t)
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	start, end := bytes.Index(readme, []byte(referenceStart)), bytes.Index(readme, []byte(referenceEnd))
	if start < 0 || end < start {
		t.Fatalf("README.md needs %s and %s around the Reference section", referenceStart, referenceEnd)
	}
	got := string(readme[start+len(referenceStart) : end])
	if got == want {
		return
	}
	if !*update {
		t.Fatal("README.md's Reference section is out of date with the examples; " +
			"run: go test ./internal/scalar -run TestReference -update")
	}
	updated := slices.Concat(readme[:start+len(referenceStart)], []byte(want), readme[end:])
	if err := os.WriteFile("README.md", updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

// reference builds the Reference section from the examples.
func reference(t *testing.T) string {
	docs := api(t)
	var b strings.Builder
	b.WriteString("\n\nGenerated from the examples in `example_*_test.go`, which `go test` runs, so every\n" +
		"example here works and prints what it says. Edit the examples, not this section, then run\n" +
		"`go test ./internal/scalar -run TestReference -update`.\n\n")
	// The headings start "Reference:", so their anchors don't clash with the
	// sections of the same names earlier in the README.
	for _, g := range referenceGroups {
		fmt.Fprintf(&b, "- [%s](#reference-%s)\n", g.title, strings.ReplaceAll(strings.ToLower(g.title), " ", "-"))
	}
	for _, g := range referenceGroups {
		fmt.Fprintf(&b, "\n### Reference: %s\n", g.title)
		exs, fset, src := examples(t, g.file)
		for _, ex := range exs {
			name := strings.Replace(ex.Name, "_", ".", 1)
			if ex.Suffix != "" {
				name += " (" + ex.Suffix + ")"
			}
			fmt.Fprintf(&b, "\n#### %s\n\n", name)
			if synopsis := docs[ex.Name]; synopsis != "" {
				b.WriteString(synopsis + "\n\n")
			}
			fmt.Fprintf(&b, "```go\n%s```\n", exampleBody(fset, src, ex))
			if out := strings.TrimSpace(ex.Output); out != "" {
				fmt.Fprintf(&b, "\n```\n%s\n```\n", out)
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

// exampleBody is the source of an example's body, as written (comments
// included), without its braces, its "// Output:" block or a level of
// indentation.
func exampleBody(fset *token.FileSet, src []byte, ex *doc.Example) string {
	block := ex.Code.(*ast.BlockStmt)
	body := string(src[fset.Position(block.Lbrace).Offset+1 : fset.Position(block.Rbrace).Offset])
	if i := strings.Index(body, "\t// Output:"); i >= 0 {
		body = body[:i]
	}
	var out strings.Builder
	for _, line := range strings.Split(strings.Trim(body, "\n"), "\n") {
		out.WriteString(strings.TrimPrefix(line, "\t") + "\n")
	}
	return out.String()
}
