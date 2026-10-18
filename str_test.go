package scalar

import "testing"

func TestIndexRIndex(t *testing.T) {
	tests := []struct {
		name      string
		s, sub    any
		pos       []int
		index     int
		rindexPos []int
		rindex    int
	}{
		{"first and last", "hello world", "o", nil, 4, nil, 7},
		{"from a position", "hello world", "o", []int{5}, 7, []int{6}, 4},
		{"match exactly at the position", "hello world", "o", []int{4}, 4, []int{4}, 4},
		{"not found", "hello world", "z", nil, -1, nil, -1},
		{"nothing after the position", "hello world", "h", []int{1}, -1, []int{0}, 0},
		{"negative position (index: 0; rindex: searches nothing)", "hello", "h", []int{-5}, 0, []int{-5}, -1},
		{"position past the end", "hello", "o", []int{99}, -1, []int{99}, 4},
		{"empty substring", "abc", "", nil, 0, nil, 3},
		{"empty substring past the end", "abc", "", []int{9}, 3, []int{9}, 3},
		{"characters, not bytes", "±0.2 – ±1.5", "±", nil, 0, nil, 7},
		{"after a multi-byte character", "±0.2", "0", nil, 1, nil, 1},
		{"numbers in string context", 12345, 34, nil, 2, nil, 2},
		{"undef", nil, "", nil, 0, nil, 0},
	}
	for _, tt := range tests {
		s := Value(tt.s)
		if got := s.Index(tt.sub, tt.pos...); got != tt.index {
			t.Errorf("%s: index = %d, want %d", tt.name, got, tt.index)
		}
		if got := s.RIndex(tt.sub, tt.rindexPos...); got != tt.rindex {
			t.Errorf("%s: rindex = %d, want %d", tt.name, got, tt.rindex)
		}
	}

	// Index's result is a character position, so it works with Substr.
	errs := Value("Error: ±0.25 – ±1.50")
	if got := errs.Substr(errs.Index("±"), 5).Str(); got != "±0.25" {
		t.Errorf("Substr at Index = %q", got)
	}
	if got := errs.Substr(errs.RIndex("±")).Str(); got != "±1.50" {
		t.Errorf("Substr at RIndex = %q", got)
	}
}

func TestCase(t *testing.T) {
	tests := []struct {
		name string
		got  *SV
		want string
	}{
		{"lc", Value("Offshore NORTHERN California").Lc(), "offshore northern california"},
		{"lc accented", Value("ÉCOLE").Lc(), "école"},
		{"uc", Value("ci.pas").Uc(), "CI.PAS"},
		{"uc full mapping", Value("straße").Uc(), "STRASSE"},
		{"fc", Value("Straße").Fc(), "strasse"},
		{"ucfirst", Value("ridgecrest, CA").UcFirst(), "Ridgecrest, CA"},
		{"ucfirst keeps the rest", Value("eBay").UcFirst(), "EBay"},
		{"ucfirst titlecase", Value("ǆemal").UcFirst(), "ǅemal"},
		{"lcfirst", Value("HELLO").LcFirst(), "hELLO"},
		{"lcfirst unicode", Value("Élan").LcFirst(), "élan"},
		{"empty", Value("").UcFirst(), ""},
		{"undef", Value(nil).Uc(), ""},
		{"number", Value(1e21).Lc(), "1e+21"},
	}
	for _, tt := range tests {
		if tt.got.Str() != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}

	// Fc is for comparing regardless of case; Lc isn't enough for that.
	if !Value("MASSE").Fc().Eq(Value("Maße").Fc()) {
		t.Error("fc(MASSE) should equal fc(Maße)")
	}
	if Value("MASSE").Lc().Eq(Value("Maße").Lc()) {
		t.Error("lc forms differ, which is why Fc exists")
	}

	orig := Value("Mixed")
	orig.Lc()
	orig.Uc()
	orig.UcFirst()
	if orig.Str() != "Mixed" {
		t.Error("case functions shouldn't change s")
	}
}

func TestTrim(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{"  Ridgecrest, CA \n\t", "Ridgecrest, CA"},
		{" no-break spaces ", "no-break spaces"},
		{"inner   spaces stay", "inner   spaces stay"},
		{"", ""},
		{nil, ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		if got := Value(tt.in).Trim().Str(); got != tt.want {
			t.Errorf("trim(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestQuotemeta(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"C++", `C\+\+`},
		{".NET", `\.NET`},
		{"C#", `C\#`},
		{"snake_case99", "snake_case99"},
		{"a b", `a\ b`},
		{"café", "café"},
		{"no break", "no\\ break"},
	}
	for _, tt := range tests {
		if got := QuoteMeta(tt.in); got != tt.want {
			t.Errorf("QuoteMeta(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// Every ASCII character, quoted, matches only itself, with or without
	// the x flag (which would otherwise ignore spaces and treat # as a
	// comment).
	for c := 0; c < 128; c++ {
		lit := string(rune(c))
		for _, flags := range []string{"", "x"} {
			pattern := "^" + QuoteMeta(lit) + "$"
			if Value(lit).M(pattern, flags) == nil {
				t.Errorf("quoted %q doesn't match itself (flags %q)", lit, flags)
			}
			if lit != "x" && Value("x").M(pattern, flags) != nil {
				t.Errorf("quoted %q matches something else (flags %q)", lit, flags)
			}
		}
	}

	// The point of it: patterns built from labels in data.
	bulletin := Value("Alerts for M4.5+ (USGS); Mw 6.0 expected")
	for _, label := range []string{"M4.5+", "(USGS)", "Mw 6.0"} {
		if bulletin.M(QuoteMeta(label)) == nil {
			t.Errorf("%s not found", label)
		}
	}
	if Value("M405").M(QuoteMeta("M4.5")) != nil {
		t.Error(`"M4.5" shouldn't match "M405"`)
	}
	if QuoteMeta(Value("a.b")) != `a\.b` {
		t.Error("QuoteMeta should accept a scalar")
	}
}

// RIndex's position rule, checked against perl -E.
func TestRIndexPosition(t *testing.T) {
	s := Value("abc")
	for _, tt := range []struct {
		sub  string
		pos  int
		want int
	}{
		{"a", -1, -1}, // searches nothing: only an empty substring is found
		{"a", 0, 0},
		{"", -1, 0},
		{"", 9, 3},
		{"c", 1, -1},
		{"bc", 1, 1},
	} {
		if got := s.RIndex(tt.sub, tt.pos); got != tt.want {
			t.Errorf("rindex(abc, %q, %d) = %d, want %d", tt.sub, tt.pos, got, tt.want)
		}
	}
}
