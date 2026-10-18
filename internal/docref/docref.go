// Package docref keeps a package's examples and the Reference section of
// its README, or of another Markdown file with CheckDoc, in step. Each package's tests call it:
//
//	func TestEveryFeatureHasExample(t *testing.T) { docref.CheckExamples(t, groups) }
//	func TestReference(t *testing.T)              { docref.CheckReadme(t, groups) }
//
// CheckExamples fails if an exported function or method of the package has
// no example. CheckReadme fails if the README's Reference section, between
// the markers <!-- reference:start --> and <!-- reference:end -->, isn't
// exactly what the examples generate; run the test with -update to rewrite
// it:
//
//	go test -run TestReference -update
package docref

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

var update = flag.Bool("update", false, "rewrite the Reference section of the docs from the examples")

// Group is a file of examples and the heading they appear under in the
// Reference. A package with a single group can leave Title empty, for a
// Reference without sub-headings.
type Group struct {
	File  string
	Title string
}

const (
	startMarker = "<!-- reference:start -->"
	endMarker   = "<!-- reference:end -->"
)

// CheckExamples fails t for every exported function and method of the
// package in the current directory that has no example in groups.
func CheckExamples(t *testing.T, groups []Group) {
	t.Helper()
	have := map[string]bool{}
	for _, g := range groups {
		exs, _, _ := examples(t, g.File)
		for _, ex := range exs {
			have[ex.Name] = true
		}
	}
	for name := range api(t) {
		if !have[name] {
			t.Errorf("%s has no example: add Example%s to %s", strings.Replace(name, "_", ".", 1), name, groupFiles(groups))
		}
	}
}

// CheckReadme is CheckDoc for the package's README.md.
func CheckReadme(t *testing.T, groups []Group) {
	t.Helper()
	CheckDoc(t, "README.md", groups)
}

// CheckDoc fails t if the Reference section of the Markdown file at path,
// relative to the package directory, isn't what groups' examples generate,
// or rewrites it if the test was run with -update.
func CheckDoc(t *testing.T, path string, groups []Group) {
	t.Helper()
	want := reference(t, groups)
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	start, end := bytes.Index(text, []byte(startMarker)), bytes.Index(text, []byte(endMarker))
	if start < 0 || end < start {
		t.Fatalf("%s needs %s and %s around the Reference section", path, startMarker, endMarker)
	}
	if string(text[start+len(startMarker):end]) == want {
		return
	}
	if !*update {
		t.Fatalf("%s's Reference section is out of date with the examples; run: go test %s -run TestReference -update",
			path, packageArg(t))
	}
	updated := slices.Concat(text[:start+len(startMarker)], []byte(want), text[end:])
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

// api is the exported functions and methods of the package in the current
// directory, as example names spell them ("Atomise", "SV_Substr"), with the
// first sentence of each one's doc comment.
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
				ident, ok := typ.(*ast.Ident)
				if !ok || !ident.IsExported() {
					continue // a method of an unexported type isn't part of the API
				}
				name = ident.Name + "_" + name
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

// reference builds the Reference section from groups' examples.
func reference(t *testing.T, groups []Group) string {
	docs := api(t)
	var b strings.Builder
	fmt.Fprintf(&b, "\n\nGenerated from the examples in %s, which `go test` runs, so every\n"+
		"example here works and prints what it says. Edit the examples, not this section, then run\n"+
		"`go test %s -run TestReference -update`.\n", groupFiles(groups), packageArg(t))

	titled := len(groups) > 1
	if titled {
		// The headings start "Reference:", so their anchors don't clash with
		// sections of the same names elsewhere in the document.
		b.WriteString("\n")
		for _, g := range groups {
			fmt.Fprintf(&b, "- [%s](#reference-%s)\n", g.Title, strings.ReplaceAll(strings.ToLower(g.Title), " ", "-"))
		}
	}
	level := "###"
	for _, g := range groups {
		if titled {
			fmt.Fprintf(&b, "\n### Reference: %s\n", g.Title)
			level = "####"
		}
		exs, fset, src := examples(t, g.File)
		for _, ex := range exs {
			// In code formatting: some renderers link bare names that look
			// like domains, such as SV.Lc (.lc is Saint Lucia's).
			name := "`" + strings.Replace(ex.Name, "_", ".", 1) + "`"
			if ex.Suffix != "" {
				name += " (" + ex.Suffix + ")"
			}
			// A package-level example, Example or Example_matchTree, names
			// no function: head it in words, "Overview" or "Match tree".
			if ex.Name == "" || strings.HasPrefix(ex.Name, "_") {
				name = words(strings.TrimPrefix(ex.Name, "_"))
			}
			fmt.Fprintf(&b, "\n%s %s\n\n", level, name)
			if synopsis := docs[ex.Name]; synopsis != "" {
				b.WriteString(goName.ReplaceAllString(escapeProse(synopsis), "`$0`") + "\n\n")
			}
			// The example's own doc comment, if it has one, explains it.
			if ex.Doc != "" {
				b.WriteString(escapeProse(string(exampleDoc(ex.Doc))))
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "```go\n%s```\n", body(fset, src, ex))
			switch out := strings.TrimSpace(ex.Output); {
			case out != "":
				fmt.Fprintf(&b, "\n```\n%s\n```\n", out)
			case !ex.EmptyOutput:
				// No "// Output:" at all: go test compiles it but doesn't
				// run it, so say so; the example's comments say why.
				b.WriteString("\nCompiled with the tests, but not run.\n")
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

// exampleDoc renders an example's doc comment as Markdown: paragraphs, and
// indented lines as code.
func exampleDoc(text string) []byte {
	var p comment.Parser
	return (&comment.Printer{}).Markdown(p.Parse(text))
}

// words turns an example suffix such as matchTree into a heading, "Match
// tree"; an empty one is "Overview".
func words(suffix string) string {
	if suffix == "" {
		return "Overview"
	}
	var b strings.Builder
	for i, r := range suffix {
		switch {
		case i == 0:
			b.WriteString(strings.ToUpper(string(r)))
		case unicode.IsUpper(r):
			b.WriteString(" " + strings.ToLower(string(r)))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeProse escapes each $ and # in Markdown prose as \$ and \#, leaving
// code spans, fenced blocks, indented code and headings alone. Perl in a
// doc comment is full of both: GitHub, VS Code and others read the text
// between two dollars, as in "my $x = $s", as a LaTeX formula, which
// renders as maths or fails to parse; and some read the #a of $#a as a
// tag. Escaped, each renders as itself.
func escapeProse(md string) string {
	lines := strings.Split(md, "\n")
	fenced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if fenced || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "#") {
			continue
		}
		var b strings.Builder
		inCode := false
		for _, r := range line {
			switch {
			case r == '`':
				inCode = !inCode
			case (r == '$' || r == '#') && !inCode:
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

// goName finds a qualified Go name in a doc comment, such as SV.Int or
// json.Marshaler, to put in code formatting, as for the headings.
var goName = regexp.MustCompile(`\b[A-Za-z]\w*\.[A-Z]\w*\b`)

// body is the source of an example's body, as written (comments included),
// without its braces, its "// Output:" block or a level of indentation.
func body(fset *token.FileSet, src []byte, ex *doc.Example) string {
	block := ex.Code.(*ast.BlockStmt)
	text := string(src[fset.Position(block.Lbrace).Offset+1 : fset.Position(block.Rbrace).Offset])
	if i := strings.Index(text, "\t// Output:"); i >= 0 {
		text = text[:i]
	}
	var out strings.Builder
	for _, line := range strings.Split(strings.Trim(text, "\n"), "\n") {
		out.WriteString(strings.TrimPrefix(line, "\t") + "\n")
	}
	return out.String()
}

// groupFiles names groups' files for a message, such as "example_test.go"
// or "example_sv_test.go, example_av_test.go".
func groupFiles(groups []Group) string {
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = "`" + g.File + "`"
	}
	if len(names) > 3 {
		return "`example_*_test.go`"
	}
	return strings.Join(names, ", ")
}

// packageArg is the go test argument for the package in the current
// directory, relative to the module root: "." for the root package, or a
// path such as ./internal/docref.
func packageArg(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for root := dir; ; root = filepath.Dir(root) {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			rel, err := filepath.Rel(root, dir)
			if err != nil {
				t.Fatal(err)
			}
			if rel == "." {
				return "."
			}
			return "./" + filepath.ToSlash(rel)
		}
		if filepath.Dir(root) == root {
			return "."
		}
	}
}
