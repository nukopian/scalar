package scalar

import "testing"

func TestIndexRindex(t *testing.T) {
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
		{"negative position is 0", "hello", "h", []int{-5}, 0, []int{-5}, 0},
		{"position past the end", "hello", "o", []int{99}, -1, []int{99}, 4},
		{"empty substring", "abc", "", nil, 0, nil, 3},
		{"empty substring past the end", "abc", "", []int{9}, 3, []int{9}, 3},
		{"characters, not bytes", "£45,000 – £55,000", "£", nil, 0, nil, 10},
		{"after a multi-byte character", "£45,000", "4", nil, 1, nil, 1},
		{"numbers in string context", 12345, 34, nil, 2, nil, 2},
		{"undef", nil, "", nil, 0, nil, 0},
	}
	for _, tt := range tests {
		s := Scalar(tt.s)
		if got := s.Index(tt.sub, tt.pos...); got != tt.index {
			t.Errorf("%s: index = %d, want %d", tt.name, got, tt.index)
		}
		if got := s.Rindex(tt.sub, tt.rindexPos...); got != tt.rindex {
			t.Errorf("%s: rindex = %d, want %d", tt.name, got, tt.rindex)
		}
	}

	// Index's result is a character position, so it works with Substr.
	salary := Scalar("Salary: £45,000 – £55,000")
	if got := salary.Substr(salary.Index("£"), 7).Str(); got != "£45,000" {
		t.Errorf("Substr at Index = %q", got)
	}
	if got := salary.Substr(salary.Rindex("£")).Str(); got != "£55,000" {
		t.Errorf("Substr at Rindex = %q", got)
	}
}

func TestCase(t *testing.T) {
	tests := []struct {
		name string
		got  *SV
		want string
	}{
		{"lc", Scalar("Senior GO Developer").Lc(), "senior go developer"},
		{"lc accented", Scalar("ÉCOLE").Lc(), "école"},
		{"uc", Scalar("node.js").Uc(), "NODE.JS"},
		{"uc full mapping", Scalar("straße").Uc(), "STRASSE"},
		{"fc", Scalar("Straße").Fc(), "strasse"},
		{"ucfirst", Scalar("go developer").Ucfirst(), "Go developer"},
		{"ucfirst keeps the rest", Scalar("eBay").Ucfirst(), "EBay"},
		{"ucfirst titlecase", Scalar("ǆemal").Ucfirst(), "ǅemal"},
		{"lcfirst", Scalar("HELLO").Lcfirst(), "hELLO"},
		{"lcfirst unicode", Scalar("Élan").Lcfirst(), "élan"},
		{"empty", Scalar("").Ucfirst(), ""},
		{"undef", Scalar(nil).Uc(), ""},
		{"number", Scalar(1e21).Lc(), "1e+21"},
	}
	for _, tt := range tests {
		if tt.got.Str() != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}

	// Fc is for comparing regardless of case; Lc isn't enough for that.
	if !Scalar("MASSE").Fc().Eq(Scalar("Maße").Fc()) {
		t.Error("fc(MASSE) should equal fc(Maße)")
	}
	if Scalar("MASSE").Lc().Eq(Scalar("Maße").Lc()) {
		t.Error("lc forms differ, which is why Fc exists")
	}

	orig := Scalar("Mixed")
	orig.Lc()
	orig.Uc()
	orig.Ucfirst()
	if orig.Str() != "Mixed" {
		t.Error("case functions shouldn't change s")
	}
}

func TestTrim(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{"  Senior Developer \n\t", "Senior Developer"},
		{" no-break spaces ", "no-break spaces"},
		{"inner   spaces stay", "inner   spaces stay"},
		{"", ""},
		{nil, ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		if got := Scalar(tt.in).Trim().Str(); got != tt.want {
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
		if got := Quotemeta(tt.in); got != tt.want {
			t.Errorf("Quotemeta(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// Every ASCII character, quoted, matches only itself, with or without
	// the x flag (which would otherwise ignore spaces and treat # as a
	// comment).
	for c := 0; c < 128; c++ {
		lit := string(rune(c))
		for _, flags := range []string{"", "x"} {
			pattern := "^" + Quotemeta(lit) + "$"
			if Scalar(lit).M(pattern, flags) == nil {
				t.Errorf("quoted %q doesn't match itself (flags %q)", lit, flags)
			}
			if lit != "x" && Scalar("x").M(pattern, flags) != nil {
				t.Errorf("quoted %q matches something else (flags %q)", lit, flags)
			}
		}
	}

	// The point of it: patterns built from skill names.
	posting := Scalar("Experience with C++ and .NET; Node.js a plus")
	for _, skill := range []string{"C++", ".NET", "Node.js"} {
		if posting.M(Quotemeta(skill)) == nil {
			t.Errorf("%s not found", skill)
		}
	}
	if Scalar("xNET").M(Quotemeta(".NET")) != nil {
		t.Error(`".NET" shouldn't match "xNET"`)
	}
	if Quotemeta(Scalar("a.b")) != `a\.b` {
		t.Error("Quotemeta should accept a scalar")
	}
}
