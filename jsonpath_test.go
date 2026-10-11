package scalar

import (
	"slices"
	"testing"
)

// The example document from RFC 9535, section 1.5.
const bookstore = `{ "store": {
    "book": [
      { "category": "reference", "author": "Nigel Rees",
        "title": "Sayings of the Century", "price": 8.95 },
      { "category": "fiction", "author": "Evelyn Waugh",
        "title": "Sword of Honour", "price": 12.99 },
      { "category": "fiction", "author": "Herman Melville",
        "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99 },
      { "category": "fiction", "author": "J. R. R. Tolkien",
        "title": "The Lord of the Rings", "isbn": "0-395-19395-8",
        "price": 22.99 }
    ],
    "bicycle": { "color": "red", "price": 399 }
  }
}`

func mustFromJSON(t *testing.T, s string) *SV {
	t.Helper()
	sv, err := FromJSON([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return sv
}

// strs renders matches as strings, in order, with references as their kind.
func strs(a *AV) []string {
	var out []string
	for _, s := range a.All() {
		if r := s.Ref(); r != "" {
			out = append(out, r)
		} else {
			out = append(out, s.Str())
		}
	}
	return out
}

func TestFindRFCExamples(t *testing.T) {
	doc := mustFromJSON(t, bookstore)
	tests := []struct {
		path   string
		want   []string
		sorted bool // hash order is unspecified, so compare sorted
	}{
		{"$.store.book[*].author", []string{"Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"}, false},
		{"$..author", []string{"Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"}, false},
		{"$.store.*", []string{"ARRAY", "HASH"}, true},
		{"$.store..price", []string{"12.99", "22.99", "399", "8.95", "8.99"}, true},
		{"$..book[2].title", []string{"Moby Dick"}, false},
		{"$..book[-1].title", []string{"The Lord of the Rings"}, false},
		{"$..book[0,1].title", []string{"Sayings of the Century", "Sword of Honour"}, false},
		{"$..book[:2].title", []string{"Sayings of the Century", "Sword of Honour"}, false},
	}
	for _, tt := range tests {
		got := strs(doc.Find(tt.path))
		if tt.sorted {
			slices.Sort(got)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("Find(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFindSelectors(t *testing.T) {
	doc := mustFromJSON(t, `{
		"a": [0, 1, 2, 3, 4, 5],
		"odd key": "yes",
		"it's": "quoted",
		"café": "unicode",
		"n": {"x": {"x": 1}},
		"k1": "v1"
	}`)
	tests := []struct {
		path string
		want []string
	}{
		{"$", []string{"HASH"}},
		{"$.a[0]", []string{"0"}},
		{"$.a[-1]", []string{"5"}},
		{"$.a[9]", nil},
		{"$.a[-9]", nil},
		{"$.a[1:3]", []string{"1", "2"}},
		{"$.a[4:]", []string{"4", "5"}},
		{"$.a[:2]", []string{"0", "1"}},
		{"$.a[-2:]", []string{"4", "5"}},
		{"$.a[::2]", []string{"0", "2", "4"}},
		{"$.a[::-1]", []string{"5", "4", "3", "2", "1", "0"}},
		{"$.a[3:0:-1]", []string{"3", "2", "1"}},
		{"$.a[::0]", nil},
		{"$.a[0, -1, 0]", []string{"0", "5", "0"}}, // unions keep duplicates
		{"$.a[ 1 : 3 ]", []string{"1", "2"}},       // spaces are allowed
		{"$['odd key']", []string{"yes"}},
		{`$["odd key"]`, []string{"yes"}},
		{`$['it\'s']`, []string{"quoted"}},
		{"$['caf\\u00e9']", []string{"unicode"}},
		{"$.café", []string{"unicode"}},
		{"$.k1", []string{"v1"}},
		{"$['k1','odd key']", []string{"v1", "yes"}},
		{"$..x", []string{"HASH", "1"}},
		{"$.n..x", []string{"HASH", "1"}},
		{"$.missing", nil},
		{"$.missing.deeper[0]", nil},
		{"$.a.x", nil},   // name on an array
		{"$.n[0]", nil},  // index on a hash
		{"$.k1.x", nil},  // name on a string
		{"$.k1[*]", nil}, // wildcard on a string
		{"$ .a [0]", []string{"0"}},
	}
	for _, tt := range tests {
		if got := strs(doc.Find(tt.path)); !slices.Equal(got, tt.want) {
			t.Errorf("Find(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFindReturnsStoredValues(t *testing.T) {
	meta := mustFromJSON(t, `{"requiredSkills": ["  Go ", "Perl  "], "n": 1}`)
	for _, s := range meta.Find("$.requiredSkills[*]").All() {
		s.Set(s.Trim())
	}
	if got := strs(meta.Path("requiredSkills").AV()); !slices.Equal(got, []string{"Go", "Perl"}) {
		t.Errorf("changes through Find didn't reach meta: %q", got)
	}

	// Nested references can be followed further.
	resp := mustFromJSON(t, `{"content": [
		{"type": "text", "text": "Here you go"},
		{"type": "tool_use", "input": {"title": "Senior Go Developer"}}
	]}`)
	input := resp.Find("$.content[*]").FirstMatch(func(b *SV) bool {
		return b.Path("type").Eq("tool_use")
	}).Path("input")
	if input.Path("title").Str() != "Senior Go Developer" {
		t.Errorf("tool_use input = %s", mustJSON(t, input))
	}
}

func TestFindDoesNotCreate(t *testing.T) {
	h := Scalar(Hash())
	if h.Find("$.a.b[0]") != nil {
		t.Error("nothing should match")
	}
	if h.HV().Length() != 0 {
		t.Error("Find shouldn't autovivify")
	}
	var nilSV *SV
	if nilSV.Find("$") != nil {
		t.Error("Find on a nil *SV should be nil")
	}
	if Scalar(nil).Find("$").Length() != 1 {
		t.Error("$ always matches the root, even undef")
	}
}

func TestFindHolesAndCycles(t *testing.T) {
	a := Array()
	a.Set(2, "c")
	root := Scalar(a)
	if got := strs(root.Find("$[*]")); !slices.Equal(got, []string{"c"}) {
		t.Errorf("holes should be skipped: %q", got)
	}
	if root.Find("$[0]") != nil {
		t.Error("a hole doesn't exist")
	}

	// A hash that contains a reference to itself.
	h := Hash("name", "loop")
	h.Set("self", h)
	if got := strs(Scalar(h).Find("$..name")); !slices.Equal(got, []string{"loop"}) {
		t.Errorf("cycle: %q", got)
	}
}

func TestFindEscapesAndDescendantBrackets(t *testing.T) {
	keys := []string{"a\nb", "a\tb", "a/b", "a\\b", `a"b`, "a\bb", "a\fb", "a\rb", "😀"}
	h := Hash()
	for _, k := range keys {
		h.Set(k, k)
	}
	doc := Scalar(h)
	paths := map[string]string{
		"$['a\\nb']":          "a\nb",
		"$['a\\tb']":          "a\tb",
		"$['a\\/b']":          "a/b",
		"$['a\\\\b']":         "a\\b",
		"$['a\\\"b']":         `a"b`,
		"$['a\\bb']":          "a\bb",
		"$['a\\fb']":          "a\fb",
		"$['a\\rb']":          "a\rb",
		"$['\\ud83d\\ude00']": "😀", // surrogate pair
	}
	for path, want := range paths {
		if got := strs(doc.Find(path)); !slices.Equal(got, []string{want}) {
			t.Errorf("Find(%q) = %q, want %q", path, got, want)
		}
	}

	// .. followed by brackets.
	nested := mustFromJSON(t, `{"a": [1, [2, 3]], "b": {"c": [4]}}`)
	got := strs(nested.Find("$..[0]"))
	slices.Sort(got)
	if want := []string{"1", "2", "4"}; !slices.Equal(got, want) {
		t.Errorf("Find($..[0]) = %q, want %q", got, want)
	}
}

func TestFindErrors(t *testing.T) {
	doc := Scalar(Hash())
	tests := map[string]string{
		"":                        "must start with $",
		"store.book":              "must start with $",
		"$.":                      "expected a member name",
		"$.[0]":                   "expected a name or * after .",
		"$.a-b":                   "expected ., .. or [",
		"$[":                      "expected a name, index, slice or *",
		"$[0":                     "expected , or ]",
		"$['a":                    "unterminated string",
		`$['\q']`:                 `unknown escape \q`,
		"$[?@.isbn]":              "filter expressions aren't supported",
		"$[99999999999999999999]": "bad integer",
		"$['\\u12":                "short \\u escape",
		"$['\\uzzzz']":            "bad \\u escape",
		"$['\\":                   "unterminated escape",
	}
	for path, want := range tests {
		expectPanic(t, path, want, func() { doc.Find(path) })
	}
}
