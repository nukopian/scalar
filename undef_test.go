package scalar

import "testing"

// SV.Undef is Perl's undef $s: s becomes undef, and Undef returns it.
func TestSVUndef(t *testing.T) {
	s := Value("x")
	if got := s.Undef(); got != s || s.Defined() {
		t.Errorf("Undef returned %p for %p, defined %v", got, s, s.Defined())
	}
	// A reference is dropped; what it referred to is untouched.
	a := Array(1, 2)
	r := Value(a)
	r.Undef()
	if r.Defined() || a.Length() != 2 {
		t.Errorf("reference: %v, array length %d", r.Defined(), a.Length())
	}
	// undef $h{k} leaves the key, with an undef value, as Perl does.
	h := Hash("k", 1)
	h.Get("k").Undef()
	if !h.Exists("k") || h.Defined("k") {
		t.Errorf("hash element: exists %v, defined %v", h.Exists("k"), h.Defined("k"))
	}
	// ...and creates a missing one, as undef $h{new} does.
	h.Get("new").Undef()
	if !h.Exists("new") {
		t.Error("undef $h{new} didn't create the key")
	}
	expectPanic(t, "element of a string", "Not an ARRAY reference", func() { Value("str").AV().Get(0).Undef() })
}

// AV.Undef and HV.Undef are Perl's undef @a and undef %h: emptied in place,
// so everything referring to them sees them empty, and returned.
func TestContainerUndef(t *testing.T) {
	a := Array(1, 2, 3)
	ref := Value(a)
	a.Each() // part-way through
	if got := a.Undef(); got != a || a.Length() != 0 || ref.AV().Length() != 0 {
		t.Errorf("array: returned %p for %p, length %d, via reference %d", got, a, a.Length(), ref.AV().Length())
	}
	if a.Push(9); a.Length() != 1 {
		t.Error("array unusable after Undef")
	}
	if i, v, ok := a.Each(); !ok || i != 0 || v.Int() != 9 {
		t.Errorf("Each didn't start again: %d %v %v", i, v, ok)
	}

	h := Hash("a", 1, "b", 2)
	href := Value(h)
	h.Each()
	if got := h.Undef(); got != h || h.Length() != 0 || href.HV().Length() != 0 || len(h.KeysSlice()) != 0 {
		t.Errorf("hash: returned %p for %p, length %d", got, h, h.Length())
	}
	h.Set("c", 3)
	if k, v, ok := h.Each(); !ok || k != "c" || v.Int() != 3 {
		t.Errorf("hash unusable or Each didn't start again: %q %v %v", k, v, ok)
	}

	var na *AV
	var nh *HV
	expectPanic(t, "nil array", "Not an ARRAY reference", func() { na.Undef() })
	expectPanic(t, "nil hash", "Not a HASH reference", func() { nh.Undef() })
}
