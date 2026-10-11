package scalar

import (
	"strings"
	"testing"
	"time"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		s, re, flags string
		want         bool
	}{
		{"Senior Developer", `^senior`, "i", true},
		{"Senior Developer", `^senior`, "", false},
		{"a\nb", `^b$`, "", false},
		{"a\nb", `^b$`, "m", true},
		{"a\nb", `a.b`, "", false},
		{"a\nb", `a.b`, "s", true},
		{"abc123", `[a-z]+ \d+  # letters then digits`, "x", true},
		{"price: $100", `(?<=\$)\d+`, "", true},   // lookbehind
		{"foo.txt", `^(?!.*\.exe$).*$`, "", true}, // negative lookahead
		{"hello hello", `\b(\w+) \1\b`, "", true}, // backreference
		{"hello world", `\b(\w+) \1\b`, "", false},
	}
	for _, tt := range tests {
		if got := Scalar(tt.s).Matches(tt.re, tt.flags); got != tt.want {
			t.Errorf("%q =~ /%s/%s = %v, want %v", tt.s, tt.re, tt.flags, got, tt.want)
		}
	}
}

func TestMatch(t *testing.T) {
	m := Scalar("Salary: £45,000 - £55,000").Match(`£([\d,]+)\s*-\s*£([\d,]+)`)
	if m.Length() != 3 || m.Get(0).Str() != "£45,000 - £55,000" || m.Get(1).Str() != "45,000" || m.Get(2).Str() != "55,000" {
		t.Errorf("Match = %s", mustJSON(t, m))
	}

	// Groups that don't participate are undef.
	m = Scalar("ab").Match(`(a)(x)?(b)`)
	if m.Get(2).Defined() || m.Get(3).Str() != "b" {
		t.Errorf("optional group: %s", mustJSON(t, m))
	}

	// No match gives nil, which is still safe to read.
	m = Scalar("nothing here").Match(`(\d+)`)
	if m != nil || m.Get(1).Str() != "" || m.Length() != 0 {
		t.Error("no match should be a readable nil")
	}
}

func TestMatchGlobal(t *testing.T) {
	all := Scalar("Go 1.27, Perl 5.40, Python 3.13").Match(`(\w+) (\d+)\.(\d+)`, "g")
	var got []string
	for _, m := range all.All() {
		got = append(got, m.AV().Get(1).Str()+"="+m.AV().Get(2).Str())
	}
	if strings.Join(got, ",") != "Go=1,Perl=5,Python=3" {
		t.Errorf(`Match(re, "g") = %v`, got)
	}

	// Each match is laid out as Match lays out one: $& at 0, then groups.
	first := all.Get(0).AV()
	if first.Get(0).Str() != "Go 1.27" || first.Get(3).Str() != "27" {
		t.Errorf("first match = %s", show(first))
	}

	// Flags combine.
	if got := Scalar("a A a").Match(`a`, "gi").Length(); got != 3 {
		t.Errorf(`Match(re, "gi") found %d matches, want 3`, got)
	}

	// No match is nil either way, so m != nil means "matched".
	if Scalar("none").Match(`\d`, "g") != nil || Scalar("none").Match(`\d`) != nil {
		t.Error("no match should be nil, with or without g")
	}
	if Scalar("none").Match(`\d`, "g").Length() != 0 {
		t.Error("a nil result should still be safe to read")
	}

	// g doesn't make sense for these, as in Perl.
	expectPanic(t, "Matches with g", `doesn't support flag 'g'`, func() { Scalar("x").Matches(`x`, "g") })
	expectPanic(t, "MatchNamed with g", `doesn't support flag 'g'`, func() { Scalar("x").MatchNamed(`(?<n>x)`, "g") })
}

func TestMatchNamed(t *testing.T) {
	h := Scalar("2026-10-11").MatchNamed(`(?<year>\d{4})-(?<month>\d\d)-(\d\d)(?<time>T.*)?`)
	if h.Get("year").Int() != 2026 || h.Get("month").Str() != "10" {
		t.Errorf("MatchNamed = %s", mustJSON(t, h))
	}
	if !h.Exists("time") || h.Get("time").Defined() {
		t.Error("non-participating named group should exist as undef")
	}
	if h.Length() != 3 {
		t.Errorf("unnamed groups leaked into MatchNamed: %s", mustJSON(t, h))
	}
	if Scalar("x").MatchNamed(`(?<d>\d)`) != nil {
		t.Error("no match should be nil")
	}
}

func TestSubst(t *testing.T) {
	s := Scalar("a  b   c")
	if n := s.S(`\s+`, " ", "g"); n.Int() != 2 || s.Str() != "a b c" {
		t.Errorf("s/\\s+/ /g: n=%s s=%q", n, s.Str())
	}

	s = Scalar("a  b   c")
	if n := s.S(`\s+`, " "); n.Int() != 1 || s.Str() != "a b   c" {
		t.Errorf("s/\\s+/ /: n=%s s=%q", n, s.Str())
	}

	// Captures in the replacement string.
	s = Scalar("Smith, John")
	s.S(`(\w+), (\w+)`, "$2 $1")
	if s.Str() != "John Smith" {
		t.Errorf("swap = %q", s.Str())
	}
	s = Scalar("2026-10-11")
	s.S(`(?<y>\d+)-(?<m>\d+)-(?<d>\d+)`, "${d}/${m}/${y}")
	if s.Str() != "11/10/2026" {
		t.Errorf("named swap = %q", s.Str())
	}

	// s///r leaves the original alone.
	orig := Scalar("Hello")
	if got := orig.S(`l`, "L", "gr").Str(); got != "HeLLo" || orig.Str() != "Hello" {
		t.Errorf("s///gr = %q, original now %q", got, orig.Str())
	}

	// s///e with a function.
	s = Scalar("10 apples and 3 pears")
	n := s.S(`(\d+)`, func(m *AV) any { return m.Get(1).Mul(2) }, "g")
	if n.Int() != 2 || s.Str() != "20 apples and 6 pears" {
		t.Errorf("s///ge: n=%s s=%q", n, s.Str())
	}
	upper := Scalar("go and perl").S(`\b(\w)`, func(m *AV) string { return strings.ToUpper(m.Get(1).Str()) }, "gr")
	if upper.Str() != "Go And Perl" {
		t.Errorf("ucfirst each word = %q", upper)
	}

	// No match: count 0, untouched, and an undef isn't vivified.
	h := Hash()
	if n := h.Get("missing").S(`x`, "y"); n.Int() != 0 || h.Exists("missing") {
		t.Error("failed S should not write")
	}

	if got := Scalar("ABC").S(`b`, "x", "ir").Str(); got != "AxC" {
		t.Errorf("case-insensitive S = %q", got)
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name, s, re string
		limit       []int
		want        string
	}{
		{"simple", "a,b,c", `,`, nil, "a|b|c"},
		{"trailing empties dropped", "a,b,,,", `,`, nil, "a|b"},
		{"leading empty kept", ",a,b", `,`, nil, "|a|b"},
		{"negative limit keeps trailing", "a,b,,", `,`, []int{-1}, "a|b||"},
		{"positive limit", "a,b,c,d", `,`, []int{2}, "a|b,c,d"},
		{"captures included", "a1b2c", `(\d)`, nil, "a|1|b|2|c"},
		{"unmatched capture is undef", "a-b", `(,)?-`, nil, "a|-|b"},
		{"empty pattern splits chars", "abc", ``, nil, "a|b|c"},
		{"empty pattern, negative limit", "abc", ``, []int{-1}, "a|b|c|"},
		{"zero-width after match", "a,b", `,*`, nil, "a|b"},
		{"awk mode", "  alpha  beta\tgamma\n", " ", nil, "alpha|beta|gamma"},
		{"unicode", "café→naïve→ok", `→`, nil, "café|naïve|ok"},
		{"empty string", "", `,`, nil, ""},
	}
	for _, tt := range tests {
		if got := showSep(Scalar(tt.s).Split(tt.re, tt.limit...), "|"); got != tt.want {
			t.Errorf("%s: split /%s/, %q = %q, want %q", tt.name, tt.re, tt.s, got, tt.want)
		}
	}
}

func TestTr(t *testing.T) {
	tests := []struct {
		name, s, from, to, flags string
		want                     string
		count                    int64
	}{
		{"uppercase", "hello", "a-z", "A-Z", "", "HELLO", 5},
		{"rot13", "Hello", "A-Za-z", "N-ZA-Mn-za-m", "", "Uryyb", 5},
		{"count only", "banana", "a", "", "", "banana", 3},
		{"short to repeats last", "abcd", "a-d", "xy", "", "xyyy", 4},
		{"delete", "h-e-l-l-o", "-", "", "d", "hello", 4},
		{"delete beyond to", "abcabc", "abc", "x", "d", "xx", 6},
		{"squeeze", "aabbccdd", "a-c", "a-c", "s", "abcdd", 6},
		{"squeeze spaces", "a    b  c", " ", " ", "s", "a b c", 6},
		{"complement", "a1b2!", "a-z", "_", "c", "a_b__", 3},
		{"complement delete", "a1b2!", "a-z", "", "cd", "ab", 3},
		{"complement squeeze", "hello,   world!", "a-zA-Z", " ", "cs", "hello world ", 5},
		{"escaped hyphen", "a-b", `\-`, "+", "", "a+b", 1},
		{"unicode", "naïve", "ï", "i", "", "naive", 1},
	}
	for _, tt := range tests {
		s := Scalar(tt.s)
		n := s.Tr(tt.from, tt.to, tt.flags)
		if s.Str() != tt.want || n.Int() != tt.count {
			t.Errorf("%s: tr/%s/%s/%s on %q = %q (count %s), want %q (count %d)",
				tt.name, tt.from, tt.to, tt.flags, tt.s, s.Str(), n, tt.want, tt.count)
		}
	}

	orig := Scalar("abc")
	if got := orig.Tr("a-c", "A-C", "r").Str(); got != "ABC" || orig.Str() != "abc" {
		t.Errorf("tr///r = %q, original now %q", got, orig.Str())
	}
}

func TestRegexPanics(t *testing.T) {
	expectPanic(t, "bad pattern", "bad regex", func() { Scalar("x").Matches(`(`) })
	expectPanic(t, "bad flag", `doesn't support flag 'g'`, func() { Scalar("x").Matches(`x`, "g") })
	expectPanic(t, "bad tr flag", `doesn't support flag 'g'`, func() { Scalar("x").Tr("x", "y", "g") })
	expectPanic(t, "bad repl type", "S replacement", func() { Scalar("x").S(`x`, 42) })
	expectPanic(t, "bad Path key", "Path key must be string or int", func() { Scalar(nil).Path(1.5) })
}

func TestRegexTimeout(t *testing.T) {
	old := RegexTimeout
	RegexTimeout = 50 * time.Millisecond
	defer func() { RegexTimeout = old }()

	evil := Scalar(strings.Repeat("a", 40) + "!")
	start := time.Now()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("catastrophic backtracking should time out")
		}
		if !strings.Contains(r.(string), "timeout") {
			t.Errorf("panic = %v", r)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("timeout took %v", elapsed)
		}
	}()
	evil.Matches(`^(a+)+$`)
}
