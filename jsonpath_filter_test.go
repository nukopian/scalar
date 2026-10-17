package scalar

import (
	"slices"
	"strings"
	"testing"
)

// Filters on the kind of data the package is for: postings.
func TestFindFilters(t *testing.T) {
	doc := mustFromJSON(t, `{"jobs": [
		{"title": "Nurse", "salary": {"min": 32000}, "remote": false, "skills": ["care"]},
		{"title": "Developer", "salary": {"min": 55000, "max": 70000}, "remote": true, "skills": ["Go", "SQL"]},
		{"title": "Café manager", "remote": false, "skills": []}
	], "threshold": 40000}`)
	for path, want := range map[string][]string{
		"$.jobs[?@.salary.min >= 30000].title":                      {"Nurse", "Developer"},
		"$.jobs[?@.salary.max].title":                               {"Developer"},    // exists
		"$.jobs[?!@.salary].title":                                  {"Café manager"}, // doesn't
		"$.jobs[?@.remote == true].title":                           {"Developer"},
		"$.jobs[?@.salary.min > $.threshold].title":                 {"Developer"}, // against the root
		"$.jobs[?length(@.skills) == 0].title":                      {"Café manager"},
		"$.jobs[?count(@.skills[*]) > 1].title":                     {"Developer"},
		"$.jobs[?match(@.title, 'N.*')].title":                      {"Nurse"},
		"$.jobs[?search(@.title, 'é')].title":                       {"Café manager"},
		"$.jobs[?length(@.title) == 12].title":                      {"Café manager"}, // characters, not bytes
		"$.jobs[?@.remote == false && @.salary].title":              {"Nurse"},
		"$.jobs[?(@.remote == true || @.salary.min < 35000)].title": {"Nurse", "Developer"},
		"$.jobs[?@.remote].title":                                   {"Nurse", "Developer", "Café manager"}, // exists, though false
		"$..[?@ == 'SQL']":                                          {"SQL"},
		"$.jobs[?value(@..min) == 55000].title":                     {"Developer"},
		"$.jobs[?@.salary.min == @.salary.max].title":               {"Café manager"}, // Nothing equals Nothing
	} {
		if got := strs(doc.Find(path)); !slices.Equal(got, want) {
			t.Errorf("Find(%q) = %q, want %q", path, got, want)
		}
	}
}

// Comparison is by JSON type, as RFC 9535 says, not by Perl's string and
// numeric contexts: 1 is 1.0, but not "1" or true.
func TestFindFilterTypes(t *testing.T) {
	// Not Array(...), which flattens arrays and hashes into the list, as
	// a Perl list does.
	doc := Value([]any{"1", 1, 1.0, true, nil, []any{1}, map[string]any{"a": 1}})
	for path, want := range map[string]int{
		"$[?@ == 1]":           2, // 1 and 1.0
		"$[?@ == '1']":         1,
		"$[?@ == true]":        1,
		"$[?@ == null]":        1,
		"$[?@ == [1]]":         -1, // not a literal: invalid
		"$[?@ < 2]":            2,  // numbers only
		"$[?@ > 'a']":          0,  // strings only: "1" < "a"
		"$[?@.a == 1]":         1,  // the hash
		"$[?@[0] == 1]":        1,  // the array
		"$[?@ != 1]":           5,
		"$[?@ >= 1 && @ <= 1]": 2,
	} {
		if want < 0 {
			expectPanic(t, path, "bad JSONPath", func() { doc.Find(path) })
			continue
		}
		if got := doc.Find(path).Length(); got != want {
			t.Errorf("Find(%q) matched %d, want %d", path, got, want)
		}
	}
}

// Queries the RFC's type rules make invalid, and why, in the panic.
func TestFindFilterErrors(t *testing.T) {
	doc := Value(Array(1))
	for path, want := range map[string]string{
		"$[?@.* == 1]":              "must select one node at most",
		"$[?@..a == 1]":             "must select one node at most",
		"$[?length(@.a)]":           "returns a value, not true or false",
		"$[?match(@, 'a') == true]": "doesn't return a value to compare",
		"$[?1]":                     "a literal isn't a test",
		"$[?!1 == 1]":               "a literal isn't a test",
		"$[?foo(@)]":                "no function foo()",
		"$[?count(1) == 1]":         "count() takes a query",
		"$[?length(@.*) == 1]":      "must select one node at most",
		"$[?match(@) == 1]":         "takes 2 arguments",
		"$[?@ == ]":                 "expected a literal, a query or a function call",
		"$[?(@ == 1]":               "expected )",
		"$[?@ == 01]":               "a bad number",
		"$[?@ == 1.]":               "digits must follow .",
		"$[?@ == True]":             "expected a literal, a query or a function call",
	} {
		expectPanic(t, path, want, func() { doc.Find(path) })
	}
}

// Deep nesting is an error, not a stack overflow, which would kill the
// program. 1,000 levels is fine.
func TestFindFilterDepth(t *testing.T) {
	doc := Value([]any{1})
	expectPanic(t, "a million (", "nested more than", func() { doc.Find("$[?" + strings.Repeat("(", 1_000_000) + "@") })
	expectPanic(t, "a million !(", "nested more than", func() { doc.Find("$[?" + strings.Repeat("!(", 1_000_000) + "@") })
	expectPanic(t, "!!", "expected a literal", func() { doc.Find("$[?!!@]") }) // one ! before a test, as the RFC's grammar has it
	ok := "$[?" + strings.Repeat("(", 300) + "@ == 1" + strings.Repeat(")", 300) + "]"
	if doc.Find(ok).Length() != 1 {
		t.Error("300 levels of parentheses")
	}
}

// I-Regexp patterns, translated to Go's regexp, or rejected.
func TestIRegexp(t *testing.T) {
	for pattern, want := range map[string]string{
		"a.c":     `a[^\n\r]c`,
		"[.]":     `[.]`,
		`\.`:      `\.`,
		`\p{Lu}+`: `\p{Lu}+`,
		"^ab$":    "^ab$", // anchors, as the RFC's mapping and the suite have them
		"a{2,3}":  "a{2,3}",
		"[^a-z]":  "[^a-z]",
		`\d`:      "", // not I-Regexp
		`\w`:      "",
		"a*?":     "", // lazy
		"a??":     "",
		`a\*?`:    `a\*?`, // an escaped *, then an optional one: not lazy
		"[]":      "",
		"[a[b]]":  "",
		"[a":      "",
		`\`:       "",
	} {
		got, ok := translateIRegexp(pattern)
		if want == "" && ok || want != "" && got != want {
			t.Errorf("translateIRegexp(%q) = %q, %v; want %q", pattern, got, ok, want)
		}
	}
	if re, ok := iRegexp("a.c", true); !ok || !re.MatchString("abc") || re.MatchString("xabc") || re.MatchString("a\nc") {
		t.Error("match's a.c")
	}
	if re, ok := iRegexp("b", false); !ok || !re.MatchString("abc") {
		t.Error("search's b")
	}
}

// Whatever the query, Find either works or panics with its own message.
func FuzzFind(f *testing.F) {
	for _, s := range []string{"$", "$.a", "$[?@.a == 1]", "$..[?length(@) > 2 && match(@.b, 'x.*')]", "$[1:3:2]",
		"$['a','b']", "$[?count(@.*) == value($..c)]", "$[?!(@.a || @.b)]", "$[?search(@, '[a-z]+')]"} {
		f.Add(s)
	}
	doc, err := FromJSON([]byte(`{"a": [1, "two", {"b": "xyz", "c": [3]}], "b": {"a": null, "c": 3.5}}`))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, path string) {
		defer func() {
			if r := recover(); r != nil {
				if msg, ok := r.(string); !ok || !strings.HasPrefix(msg, "scalar: bad JSONPath") {
					t.Fatalf("Find(%q) panicked with %v", path, r)
				}
			}
		}()
		doc.Find(path)
	})
}
