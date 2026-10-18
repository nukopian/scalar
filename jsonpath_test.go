package scalar

import (
	"slices"
	"strings"
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
	config := mustFromJSON(t, `{"stations": ["  PAS ", "GSC  "], "n": 1}`)
	for _, s := range config.Find("$.stations[*]").All() {
		s.Set(s.Trim())
	}
	if got := strs(config.Path("stations").AV()); !slices.Equal(got, []string{"PAS", "GSC"}) {
		t.Errorf("changes through Find didn't reach config: %q", got)
	}

	// Nested references can be followed further.
	feed := mustFromJSON(t, `{"features": [
		{"type": "Summary", "text": "Two events"},
		{"type": "Feature", "properties": {"place": "12 km NNE of Ridgecrest, CA"}}
	]}`)
	props := feed.Find("$.features[*]").FirstMatch(func(f *SV) bool {
		return f.Path("type").Eq("Feature")
	}).Path("properties")
	if props.Path("place").Str() != "12 km NNE of Ridgecrest, CA" {
		t.Errorf("feature properties = %s", mustJSON(t, props))
	}
}

func TestFindDoesNotCreate(t *testing.T) {
	h := Value(Hash())
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
	if Value(nil).Find("$").Length() != 1 {
		t.Error("$ always matches the root, even undef")
	}
}

func TestFindHolesAndCycles(t *testing.T) {
	a := Array()
	a.Set(2, "c")
	root := Value(a)
	// A hole is undef, as MarshalJSON writes it (null), and stays a hole.
	if got := root.Find("$[*]"); got.Length() != 3 || got.Get(0).Defined() || got.Get(2).Str() != "c" {
		t.Errorf("holes should be undef: %s", got.Dump())
	}
	if got := root.Find("$[0]"); got.Length() != 1 || got.Get(0).Defined() || a.Exists(0) {
		t.Error("a hole should be found as undef, and not filled in")
	}
	if msg := panicOf(func() { root.Find("$[0]").Get(0).Set(1) }); !strings.Contains(msg, "a hole in an array") {
		t.Errorf("writing to a hole found by Find: %q", msg)
	}

	// A hash that contains a reference to itself.
	h := Hash("name", "loop")
	h.Set("self", h)
	if got := strs(Value(h).Find("$..name")); !slices.Equal(got, []string{"loop"}) {
		t.Errorf("cycle: %q", got)
	}
}

func TestFindEscapesAndDescendantBrackets(t *testing.T) {
	keys := []string{"a\nb", "a\tb", "a/b", "a\\b", `a"b`, "a\bb", "a\fb", "a\rb", "😀"}
	h := Hash()
	for _, k := range keys {
		h.Set(k, k)
	}
	doc := Value(h)
	paths := map[string]string{
		"$['a\\nb']":          "a\nb",
		"$['a\\tb']":          "a\tb",
		"$['a\\/b']":          "a/b",
		"$['a\\\\b']":         "a\\b",
		`$['a"b']`:            `a"b`, // the other quote needs no escape
		`$["a\"b"]`:           `a"b`, // its own does
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
	doc := Value(Hash())
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
		"$[99999999999999999999]": "must be within",
		"$[9007199254740992]":     "must be within", // 2^53
		"$[01]":                   "can't start with 0",
		"$[-0]":                   "can't start with 0",
		`$['a\"b']`:               `unknown escape \"`,   // only "\'" in single quotes
		"$['a\tb']":               "a control character", // a raw tab
		"$ ":                      "space after the last segment",
		"$['\\u12":                "short \\u escape",
		"$['\\uzzzz']":            "bad \\u escape",
		"$['\\":                   "unterminated escape",
		"$['\\ud83d']":            "unpaired high surrogate",
		"$['\\ud83dx']":           "unpaired high surrogate",
		"$['\\ud83d\\u0041']":     "not a low surrogate",
		"$['\\ude00']":            "unpaired low surrogate",
	}
	for path, want := range tests {
		expectPanic(t, path, want, func() { doc.Find(path) })
	}
}
