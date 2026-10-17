package scalar

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
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
	ok := "$[?" + strings.Repeat("(", 999) + "@ == 1" + strings.Repeat(")", 999) + "]"
	if msg := panicOf(func() { doc.Find(ok) }); msg != "" || doc.Find(ok).Length() != 1 {
		t.Errorf("999 levels of parentheses: %s", msg) // the limit is 1,000, as the message says
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

// The wash-up's findings, for data built in Go rather than read from JSON.

// A filter comparing a structure with one that contains itself compares
// safely, as Equal does, where it overflowed the stack. In a child
// process, as a stack overflow kills the process.
func TestFindFilterLoops(t *testing.T) {
	if os.Getenv("SCALAR_FILTER_LOOP") != "" {
		h := Hash("a", 1)
		h.Set("self", h)
		if got := Value(h).Find("$[?@ == $]").Length(); got != 1 { // self is h, so equal to $
			t.Errorf("$[?@ == $] matched %d, want 1", got)
		}
		other := Hash("a", 1)
		other.Set("self", other)
		if !(value{sv: Value(h)}).equal(value{sv: Value(other)}) {
			t.Error("two structures that loop alike aren't equal")
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFindFilterLoops$", "-test.v")
	cmd.Env = append(os.Environ(), "SCALAR_FILTER_LOOP=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := string(out)
		if len(msg) > 600 {
			msg = msg[:600] + "..."
		}
		t.Fatalf("the child process failed (%v):\n%s", err, msg)
	}
}

// A nil *AV or *HV is undef, as a nil slice or map is, not a reference
// that panics when followed.
func TestNilContainers(t *testing.T) {
	for name, f := range map[string]func(){
		"Find $[*] on a nil array":     func() { Value((*AV)(nil)).Find("$[*]") },
		"Find $.a on a nil hash":       func() { Value((*HV)(nil)).Find("$.a") },
		"length() of a nil array":      func() { Value([]any{(*AV)(nil)}).Find("$[?length(@) == 0]") },
		"MarshalJSON of a nil array":   func() { Value((*AV)(nil)).MarshalJSON() },
		"Set to a nil hash, then Dump": func() { Value(1).Set((*HV)(nil)).Dump() },
	} {
		if msg := panicOf(f); msg != "" {
			t.Errorf("%s: panicked: %s", name, msg)
		}
	}
	if Value((*AV)(nil)).Defined() || Value((*HV)(nil)).Defined() {
		t.Error("a nil *AV or *HV isn't undef")
	}
}

// Numbers compare exactly when both are integers, as NumEq does, beyond
// float64's 2^53.
func TestFindFilterBigIntegers(t *testing.T) {
	doc := Value([]any{int64(1<<53 + 1), int64(1 << 53)})
	for path, want := range map[string]int{
		"$[?@ == 9007199254740993]": 1,
		"$[?@ > 9007199254740992]":  1,
		"$[?@ == 9007199254740992]": 1,
		"$[?@ < 1e300]":             2,
	} {
		if got := doc.Find(path).Length(); got != want {
			t.Errorf("Find(%q) matched %d, want %d", path, got, want)
		}
	}
}

// A byte string is a string of Latin-1 characters, as Eq and Length have
// it, in filters too.
func TestFindFilterBytes(t *testing.T) {
	doc := Value([]any{[]byte{0xe9}})
	for path, want := range map[string]int{
		"$[?@ == 'é']":            1,
		"$[?@ < 'ê']":             1,
		"$[?length(@) == 1]":      1,
		"$[?match(@, 'é')]":       1,
		`$[?search(@, '\\p{L}')]`: 1,
	} {
		if got := doc.Find(path).Length(); got != want {
			t.Errorf("Find(%q) matched %d, want %d", path, got, want)
		}
	}
}

// A hole in an array is null, as MarshalJSON writes it, to Find too.
func TestFindHoles(t *testing.T) {
	a := Array()
	a.Set(0, 1)
	a.Set(2, 3)
	doc := Value(a)
	for path, want := range map[string]int{
		"$[1]":                 1,
		"$[*]":                 3,
		"$[0:3]":               3,
		"$[?@ == null]":        1,
		"$[?count($[*]) == 3]": 3,
	} {
		if got := doc.Find(path).Length(); got != want {
			t.Errorf("Find(%q) matched %d, want %d", path, got, want)
		}
	}
	if a.Get(1).Defined() || a.Exists(1) {
		t.Error("Find filled the hole in")
	}
}

// A part shared by two branches is found under each, as it would be after
// a JSON round trip; a loop is still followed once.
func TestFindShared(t *testing.T) {
	x := Hash("c", 1)
	h := Hash() // not Hash("a", x, ...), which flattens x into the list, as Perl does
	h.Set("a", x)
	h.Set("b", x)
	doc := Value(h)
	if got := doc.Find("$..c").Length(); got != 2 {
		t.Errorf("$..c matched %d, want 2", got)
	}
	if got := doc.Find("$..*").Length(); got != 4 {
		t.Errorf("$..* matched %d, want 4", got)
	}
	loop := Hash("n", 1)
	loop.Set("self", loop)
	if got := Value(loop).Find("$..n").Length(); got != 1 {
		t.Errorf("$..n in a loop matched %d, want 1", got)
	}
}

// Patterns that aren't I-Regexp, by RFC 9485's grammar, though Go's regexp
// would take them.
func TestIRegexpStrict(t *testing.T) {
	for _, pattern := range []string{
		"(?i)abc", "(?:a)", "(?P<x>a)", "(?s).", // groups with ?
		"a{,3}", "a{", "a}", "a]", "{1}", // braces and brackets outside a class
		`\p{Greek}`, `\p{Any}`, `\p{^L}`, // not a category
		"[a-b-c]", // - after a range
	} {
		if got, ok := translateIRegexp(pattern); ok {
			t.Errorf("translateIRegexp(%q) = %q, want it rejected", pattern, got)
		}
	}
	for _, pattern := range []string{"a{2}", "a{2,}", "a{2,3}", `\p{L}`, `\p{Lu}`, `\P{Nd}`, "[a-c]", "[a-]", "(ab)+"} {
		if _, ok := translateIRegexp(pattern); !ok {
			t.Errorf("translateIRegexp(%q) rejected, want it taken", pattern)
		}
	}
}

// The caches keyed by what a program is given (paths, and patterns that
// may come from the data) are bounded.
func TestFindCachesBounded(t *testing.T) {
	doc := Value([]any{1})
	for i := range cacheLimit + 10 {
		doc.Find("$[" + strconv.Itoa(i) + "]")
		iRegexp("x"+strconv.Itoa(i), true)
	}
	if n := pathCache.len(); n > cacheLimit {
		t.Errorf("pathCache has %d entries, over %d", n, cacheLimit)
	}
	if n := iRegexps.len(); n > cacheLimit {
		t.Errorf("iRegexps has %d entries, over %d", n, cacheLimit)
	}
}

// QuoteJSONPath's literal finds exactly what it quotes, as a value in a
// filter and as a name, however awkward.
func TestQuoteJSONPath(t *testing.T) {
	for _, v := range []string{"plain", "it's", `back\slash`, `"double"`, "tab\there", "line\nbreak", "\x01ctrl", "café ✓", "", "]) || true || ("} {
		doc := Value([]any{map[string]any{"k": v}, map[string]any{"k": v + "x"}})
		if got := doc.Find("$[?@.k == " + QuoteJSONPath(v) + "].k"); got.Length() != 1 || got.Get(0).Str() != v {
			t.Errorf("filter on %q: %v", v, got)
		}
		h := Hash()
		h.Set(v, 1)
		h.Set(v+"x", 2)
		if got := Value(h).Find("$[" + QuoteJSONPath(v) + "]"); got.Length() != 1 || got.Get(0).Int() != 1 {
			t.Errorf("name %q: %v", v, got)
		}
	}
}
