package scalar

import "testing"

// same reports whether two scalars hold the same value: the same kind and
// string form, and for references, the same array or hash.
func same(x, y *SV) bool {
	if x.Kind() != y.Kind() || x.Str() != y.Str() {
		return false
	}
	switch x.val().(type) {
	case *AV, *HV:
		return x.val() == y.val()
	}
	return true
}

func TestThen(t *testing.T) {
	called := false
	if r := Value(nil).Then(func(v *SV) any { called = true; return 1 }); called || r.Defined() {
		t.Error("Then on undef shouldn't call f, and gives undef")
	}
	if r := (*SV)(nil).Then(func(v *SV) any { return 1 }); r.Defined() {
		t.Error("Then on nil gives undef")
	}
	if r := Value("0").Then(func(v *SV) any { return v.Add(1) }); r.Int() != 1 {
		t.Error(`Then on "0", defined though false, should call f`)
	}
	if r := Value(1).Then(func(v *SV) any { return nil }); r.Defined() {
		t.Error("f returning nil gives undef")
	}

	// A new scalar: writing to it doesn't change the original, or create a
	// missing element.
	h := Hash("a", 1)
	h.Get("a").Then(func(v *SV) any { return v }).Set(99)
	h.Get("missing").Then(func(v *SV) any { return v }).Set(99)
	if h.Get("a").Int() != 1 || h.Exists("missing") {
		t.Error("Then should return a new scalar")
	}

	// When: a guard.
	positive := func(v *SV) bool { return v.Num() > 0 }
	if Value(5).When(positive).Int() != 5 || Value(-5).When(positive).Defined() {
		t.Error("When should keep a value that passes and drop one that doesn't")
	}
	called = false
	if Value(nil).When(func(v *SV) bool { called = true; return true }).Defined() || called {
		t.Error("When on undef shouldn't call pred")
	}
	if h.Get("missing").When(func(v *SV) bool { return true }).Set(1); h.Exists("missing") {
		t.Error("When should return a new scalar")
	}

	// A chain, as in the documentation.
	parse := func(v *SV) any { return v.M(`£([\d,]+)`).Get(1).S(`,`, "", "gr").When(positive) }
	annual := func(v *SV) any { return v.Mul(12) }
	for text, want := range map[string]string{"£4,000 a month": "48000", "competitive": "none", "£0": "none"} {
		if got := Value(text).Then(parse).Then(annual).Or("none").Str(); got != want {
			t.Errorf("%q: %s, want %s", text, got, want)
		}
	}
}

// Then is the bind of a Maybe monad, with Value as the unit (over defined
// values) and undef as Nothing.
func TestThenMonadLaws(t *testing.T) {
	arr, hash := Value(Array(1)), Value(Hash("k", "v"))
	values := []*SV{Value(nil), Value(0), Value(""), Value("x"), Value(2.5), Value([]byte{0xff}), arr, hash}
	fns := map[string]func(*SV) any{
		"double":    func(v *SV) any { return v.Mul(2) },
		"undef":     func(v *SV) any { return nil },
		"len or no": func(v *SV) any { return v.When(func(v *SV) bool { return v.Length() > 1 }) },
		"ref":       func(v *SV) any { return arr },
		"plain go":  func(v *SV) any { return v.Str() + "!" },
	}
	unit := func(v *SV) any { return v }

	for _, m := range values {
		// Right identity: m.Then(unit) is m.
		if got := m.Then(unit); !same(got, m) {
			t.Errorf("right identity for %q: got %q", m.Str(), got.Str())
		}
		for fname, f := range fns {
			// Left identity: Value(x).Then(f) is f(x), for a defined x.
			if m.Defined() {
				if got, want := m.Then(f), Value(f(m)); !same(got, want) {
					t.Errorf("left identity, %s of %q: %q, want %q", fname, m.Str(), got.Str(), want.Str())
				}
			}
			// Associativity: m.Then(f).Then(g) is m.Then(x => f(x).Then(g)).
			for gname, g := range fns {
				lhs := m.Then(f).Then(g)
				rhs := m.Then(func(x *SV) any { return Value(f(x)).Then(g) })
				if !same(lhs, rhs) {
					t.Errorf("associativity, %s then %s of %q: %q vs %q", fname, gname, m.Str(), lhs.Str(), rhs.Str())
				}
			}
		}
	}
}
