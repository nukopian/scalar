package scalar

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// The how-to guide's recipes are generated from example_howto_test.go
// (roadmap 2026-0008). To update them:
//
//	go test -run TestHowTo -update
func TestHowTo(t *testing.T) {
	const guidePath, examplesPath = "docs/HOWTO.md", "example_howto_test.go"
	guide, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatal(err)
	}
	want, problems := howto(guide, examplesPath, src)
	for _, p := range problems {
		t.Error(p)
	}
	if len(problems) > 0 || bytes.Equal(want, guide) {
		return
	}
	if !updating() {
		t.Fatalf("%s's recipes are out of date with %s: edit the examples, not the guide, then run: go test -run TestHowTo -update",
			guidePath, examplesPath)
	}
	if err := os.WriteFile(guidePath, want, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHowToCatches plants each failure TestHowTo promises to catch.
func TestHowToCatches(t *testing.T) {
	src := []byte(`package scalar_test

import (
	"fmt"

	"github.com/nukopian/scalar"
)

// two is shown once, with the first recipe that uses it.
var two = scalar.Value(2)

func Example_good() {
	fmt.Println(scalar.Value(1).Add(two))
	// Output: 3
}

func Example_goodPlainGo() {
	fmt.Println(1 + 2)
	// Output: 3
}

func Example_alone() {
	fmt.Println(two)
	// Output: 2
}

func Example_differs() {
	fmt.Println(two)
	// Output: 2
}

func Example_differsPlainGo() {
	fmt.Println("two")
	// Output: two
}

func Example_unused() {
	fmt.Println(two)
	// Output: 2
}
`)
	guide := []byte("# Guide\n\n<!-- recipe:good -->\n<!-- /recipe -->\n\n" +
		"<!-- recipe:alone -->\n<!-- /recipe -->\n\n" +
		"<!-- recipe:differs -->\n<!-- /recipe -->\n\n" +
		"<!-- recipe:missing -->\n<!-- /recipe -->\n")
	got, problems := howto(guide, "planted_example_test.go", src)
	for _, want := range []string{
		"recipe alone has no plain Go twin",
		"recipe differs prints differently from its plain Go twin",
		"recipe missing has no Example_missing",
		"Example_unused has no recipe",
	} {
		if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, want) }) {
			t.Errorf("no problem saying %q; got %q", want, problems)
		}
	}
	if len(problems) != 4 {
		t.Errorf("got %d problems, want 4: %q", len(problems), problems)
	}
	if !bytes.Contains(got, []byte("var two = scalar.Value(2)")) ||
		bytes.Count(got, []byte("var two = scalar.Value(2)")) != 1 {
		t.Errorf("the declaration two should be shown once, with the first recipe:\n%s", got)
	}
	if !bytes.Contains(got, []byte("[`SV.Add`](EXAMPLES.md#svadd)")) {
		t.Errorf("the good recipe should link SV.Add:\n%s", got)
	}

	// A recipe edited in the guide alone is regenerated, so TestHowTo
	// sees the guide out of date.
	edited := bytes.Replace(got, []byte("Add(two)"), []byte("Add(two) // edited"), 1)
	if again, _ := howto(edited, "planted_example_test.go", src); bytes.Equal(again, edited) {
		t.Error("a recipe edited in the guide alone wasn't noticed")
	}
}

// updating says whether the tests were run with -update, the flag docref
// defines for the examples.
func updating() bool {
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

var recipeRegion = regexp.MustCompile(`(?s)<!-- recipe:(\w+) -->\n.*?<!-- /recipe -->`)

// howto returns guide with each recipe region generated from the examples
// in src, read as the file filename, and the problems found.
func howto(guide []byte, filename string, src []byte) ([]byte, []string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return guide, []string{err.Error()}
	}
	info := &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check("scalar_test", fset, []*ast.File{f}, info)
	if err != nil {
		return guide, []string{fmt.Sprintf("%s doesn't type-check: %v", filename, err)}
	}
	r := &recipes{fset: fset, src: src, info: info, pkg: pkg, funcs: map[string]*ast.FuncDecl{},
		outputs: map[string]string{}, decls: map[types.Object]ast.Decl{}, methods: map[ast.Decl][]ast.Decl{},
		shown: map[ast.Decl]bool{}}
	for _, ex := range doc.Examples(f) {
		r.outputs[strings.TrimPrefix(ex.Name, "_")] = ex.Output // Example_name is named "_name"
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Example_") {
			r.funcs[strings.TrimPrefix(fn.Name.Name, "Example_")] = fn
		}
		ast.Inspect(d, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if obj := info.Defs[id]; obj != nil && obj.Parent() == pkg.Scope() {
					r.decls[obj] = d
				}
			}
			return true
		})
	}
	// A type's methods go with it, as a method such as UnmarshalJSON is
	// called without being named.
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil {
			typ := fn.Recv.List[0].Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			if id, ok := typ.(*ast.Ident); ok {
				if t := r.decls[info.Uses[id]]; t != nil {
					r.methods[t] = append(r.methods[t], d)
				}
			}
		}
	}

	var problems []string
	seen := map[string]bool{}
	out := recipeRegion.ReplaceAllFunc(guide, func(region []byte) []byte {
		name := string(recipeRegion.FindSubmatch(region)[1])
		seen[name] = true
		text, problem := r.region(name)
		if problem != "" {
			problems = append(problems, problem)
			return region
		}
		return []byte(text)
	})
	for name := range r.funcs {
		if !strings.HasSuffix(name, "PlainGo") && !seen[name] {
			problems = append(problems, fmt.Sprintf("Example_%s has no recipe: add <!-- recipe:%s --> and <!-- /recipe --> to the guide", name, name))
		}
	}
	slices.Sort(problems)
	return out, problems
}

// recipes is what's needed to write the guide's recipes from a file of
// examples.
type recipes struct {
	fset    *token.FileSet
	src     []byte
	info    *types.Info
	pkg     *types.Package
	funcs   map[string]*ast.FuncDecl // examples, by suffix
	outputs map[string]string        // what each prints, by suffix
	decls   map[types.Object]ast.Decl
	methods map[ast.Decl][]ast.Decl // a type's methods, by the type's declaration
	shown   map[ast.Decl]bool       // declarations already in the guide
}

// region is the recipe region for the example with suffix name, or a
// problem.
func (r *recipes) region(name string) (string, string) {
	fn, twin := r.funcs[name], r.funcs[name+"PlainGo"]
	switch {
	case fn == nil:
		return "", fmt.Sprintf("recipe %s has no Example_%s", name, name)
	case twin == nil:
		return "", fmt.Sprintf("recipe %s has no plain Go twin: add Example_%sPlainGo, printing the same", name, name)
	case r.outputs[name] != r.outputs[name+"PlainGo"]:
		return "", fmt.Sprintf("recipe %s prints differently from its plain Go twin:\n%s\nagainst\n%s",
			name, r.outputs[name], r.outputs[name+"PlainGo"])
	}
	decls := r.used(fn)
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- recipe:%s -->\n\n```go\n%s```\n\nIt prints:\n\n```\n%s```\n\n", name, r.code(fn), r.outputs[name])
	fmt.Fprintf(&b, "In plain Go, printing the same:\n\n```go\n%s```\n\n", r.code(twin))
	if links := r.links(fn, decls); links != "" {
		fmt.Fprintf(&b, "**Uses:** %s.\n\n", links)
	}
	b.WriteString("<!-- /recipe -->")
	return b.String(), ""
}

// code is the source of fn's body, after the declarations it uses that
// the guide hasn't shown yet, without its braces, its "// Output:", a
// level of indentation, or lines ending "// test only".
func (r *recipes) code(fn *ast.FuncDecl) string {
	var b strings.Builder
	for _, d := range r.used(fn) {
		if r.shown[d] {
			continue
		}
		r.shown[d] = true
		start := d.Pos()
		if g, ok := d.(*ast.GenDecl); ok && g.Doc != nil {
			start = g.Doc.Pos()
		}
		b.WriteString(r.text(start, d.End()) + "\n\n")
	}
	body := r.text(fn.Body.Lbrace+1, fn.Body.Rbrace)
	if i := strings.Index(body, "\t// Output:"); i >= 0 {
		body = body[:i]
	}
	blank := true
	for _, line := range strings.Split(strings.Trim(body, "\n"), "\n") {
		line = strings.TrimPrefix(line, "\t")
		if strings.HasSuffix(line, "// test only") || (line == "" && blank) {
			continue
		}
		blank = line == ""
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func (r *recipes) text(start, end token.Pos) string {
	return string(r.src[r.fset.Position(start).Offset:r.fset.Position(end).Offset])
}

// used is the package-level declarations fn refers to, directly or
// through another, in the order they're written, leaving out what's
// referred to only on lines ending "// test only".
func (r *recipes) used(fn *ast.FuncDecl) []ast.Decl {
	found := map[ast.Decl]bool{}
	var visit func(n ast.Node)
	visit = func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok || r.testOnly(id.Pos()) {
				return true
			}
			if d := r.decls[r.info.Uses[id]]; d != nil && !found[d] {
				found[d] = true
				visit(d)
				for _, m := range r.methods[d] {
					found[m] = true
					visit(m)
				}
			}
			return true
		})
	}
	visit(fn.Body)
	out := make([]ast.Decl, 0, len(found))
	for d := range found {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b ast.Decl) int { return int(a.Pos() - b.Pos()) })
	return out
}

func (r *recipes) testOnly(pos token.Pos) bool {
	p := r.fset.Position(pos)
	line := r.src[p.Offset-p.Column+1:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return bytes.HasSuffix(bytes.TrimSpace(line), []byte("// test only"))
}

// links links each scalar function and method fn and decls call to its
// entry in the examples, in the order they're first written.
func (r *recipes) links(fn *ast.FuncDecl, decls []ast.Decl) string {
	type call struct {
		pos  token.Pos
		name string
	}
	var calls []call
	inspect := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			se, ok := n.(*ast.SelectorExpr)
			if !ok || r.testOnly(se.Pos()) {
				return true
			}
			obj := r.info.Uses[se.Sel]
			if sel := r.info.Selections[se]; sel != nil && sel.Kind() == types.MethodVal {
				obj = sel.Obj()
			}
			if name := scalarFunc(obj); name != "" {
				calls = append(calls, call{se.Sel.Pos(), name}) // a chain's outer call is found first
			}
			return true
		})
	}
	for _, d := range decls {
		inspect(d)
	}
	inspect(fn.Body)
	slices.SortFunc(calls, func(a, b call) int { return int(a.pos - b.pos) })
	var names []string
	for _, c := range calls {
		if !slices.Contains(names, c.name) {
			names = append(names, c.name)
		}
	}
	links := make([]string, len(names))
	for i, name := range names {
		links[i] = fmt.Sprintf("[`%s`](EXAMPLES.md#%s)", name, strings.ToLower(strings.ReplaceAll(name, ".", "")))
	}
	return strings.Join(links, ", ")
}

// scalarFunc names obj as the examples' headings do, "Hash" or "SV.Get",
// if it's an exported function or method of scalar, or returns "".
func scalarFunc(obj types.Object) string {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "github.com/nukopian/scalar" || !fn.Exported() {
		return ""
	}
	recv := fn.Type().(*types.Signature).Recv()
	if recv == nil {
		return fn.Name()
	}
	typ := recv.Type()
	if p, ok := typ.(*types.Pointer); ok {
		typ = p.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok || !named.Obj().Exported() {
		return ""
	}
	return named.Obj().Name() + "." + fn.Name()
}

// Every relative link in the documents resolves, #anchors included.
func TestDocLinksResolve(t *testing.T) {
	var docs []string
	for _, pattern := range []string{"*.md", "docs/*.md", "docs/changes/*.md"} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, paths...)
	}
	for _, path := range docs {
		for _, problem := range brokenLinks(t, path) {
			t.Error(problem)
		}
	}
}

// TestDocLinksResolveCatches plants a broken link of each kind.
func TestDocLinksResolveCatches(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.md", "# A `title`\n\n## Same\n\n## Same\n\n"+
		"[ok](#a-title) [ok](#same-1) [ok](b.md#reading-one-event) [ok](https://example.com/x#y) "+
		"`[code](missing.md)`\n\n```\n[block](missing.md)\n```\n\n"+
		"[bad](#nowhere) [bad](missing.md) [bad](b.md#gone)\n")
	write("b.md", "### Reading one event\n")
	got := brokenLinks(t, filepath.Join(dir, "a.md"))
	if len(got) != 3 {
		t.Errorf("want 3 broken links, got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
	for _, want := range []string{"#nowhere", "missing.md", "b.md#gone"} {
		if !slices.ContainsFunc(got, func(p string) bool { return strings.Contains(p, "-> "+want) }) {
			t.Errorf("%s wasn't reported: %q", want, got)
		}
	}
}

var (
	mdFence   = regexp.MustCompile("(?ms)^\\s*```.*?^\\s*```[^\\n]*$")
	mdSpan    = regexp.MustCompile("`[^`\\n]*`")
	mdLink    = regexp.MustCompile(`\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	mdRefLink = regexp.MustCompile(`(?m)^\[[^\]]+\]:\s*(\S+)`)
	mdHeading = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*#*$`)
	mdAnchor  = regexp.MustCompile(`<a\s+(?:name|id)="([^"]+)"`)
	mdInline  = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// brokenLinks is the relative links in the Markdown file at path, outside
// code, whose file or #anchor doesn't exist.
func brokenLinks(t *testing.T, path string) []string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	prose := mdSpan.ReplaceAllString(mdFence.ReplaceAllString(string(text), ""), "")
	var targets []string
	for _, m := range mdLink.FindAllStringSubmatch(prose, -1) {
		targets = append(targets, m[1])
	}
	for _, m := range mdRefLink.FindAllStringSubmatch(prose, -1) {
		targets = append(targets, m[1])
	}
	var broken []string
	for _, target := range targets {
		if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		file, anchor, _ := strings.Cut(target, "#")
		dest := path
		if file != "" {
			dest = filepath.Join(filepath.Dir(path), file)
			if _, err := os.Stat(dest); err != nil {
				broken = append(broken, fmt.Sprintf("%s -> %s: no such file", path, target))
				continue
			}
		}
		if anchor != "" && strings.HasSuffix(dest, ".md") && !anchors(t, dest)[anchor] {
			broken = append(broken, fmt.Sprintf("%s -> %s: no such heading or anchor", path, target))
		}
	}
	return broken
}

// anchors is the #anchors of the Markdown file at path: its headings, as
// GitHub makes them, and its <a id> and <a name> tags.
func anchors(t *testing.T, path string) map[string]bool {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	md := mdFence.ReplaceAllString(string(text), "")
	out := map[string]bool{}
	for _, m := range mdAnchor.FindAllStringSubmatch(md, -1) {
		out[m[1]] = true
	}
	count := map[string]int{}
	for _, m := range mdHeading.FindAllStringSubmatch(md, -1) {
		slug := strings.Map(func(r rune) rune {
			switch {
			case r == ' ':
				return '-'
			case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
				return unicode.ToLower(r)
			}
			return -1
		}, mdInline.ReplaceAllString(m[1], "$1"))
		if n := count[slug]; n > 0 {
			out[fmt.Sprintf("%s-%d", slug, n)] = true
		} else {
			out[slug] = true
		}
		count[slug]++
	}
	return out
}
