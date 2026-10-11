package scalar

import (
	"strings"
	"testing"
)

// expectPanic fails unless f panics with a message containing want.
func expectPanic(t *testing.T, name, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Errorf("%s: didn't panic", name)
		} else if !strings.Contains(r.(string), want) {
			t.Errorf("%s: panic %q, want it to contain %q", name, r, want)
		}
	}()
	f()
}

// show renders an array as Perl would print it, with holes and undef as "-".
func show(a *AV) string {
	parts := make([]string, a.Length())
	for i, s := range a.All() {
		if s.Defined() {
			parts[i] = s.Str()
		} else {
			parts[i] = "-"
		}
	}
	return strings.Join(parts, ",")
}

func TestNegativeIndexReads(t *testing.T) {
	a := Array("a", "b", "c")
	for i, want := range map[int]string{-1: "c", -2: "b", -3: "a"} {
		if got := a.Get(i).Str(); got != want {
			t.Errorf("$a[%d] = %q, want %q", i, got, want)
		}
	}
	if a.Get(-4).Defined() || Array().Get(-1).Defined() {
		t.Error("indices before the start should read as undef")
	}
	// Reading through them, even with an autovivifying deref, stays safe.
	if a.Get(-4).HV().Get("x").Defined() || Scalar(a).Path(-10, "x", 0).Defined() {
		t.Error("reads through a non-creatable element should be undef")
	}
	if a.Length() != 3 {
		t.Errorf("reads changed the array: %s", show(a))
	}
}

func TestNegativeIndexWrites(t *testing.T) {
	a := Array("a", "b", "c")
	a.Get(-1).Set("C")
	a.Set(-3, "A")
	a.Get(-2).Inc()
	if show(a) != "A,c,C" {
		t.Errorf("in-range negative writes: %s", show(a))
	}

	const msg = "Modification of non-creatable array value attempted, subscript -4"
	expectPanic(t, "Set(-4)", msg, func() { a.Set(-4, "x") })
	expectPanic(t, "Get(-4).Set", msg, func() { a.Get(-4).Set("x") })
	expectPanic(t, "Get(-4).Inc", msg, func() { a.Get(-4).Inc() })
	expectPanic(t, "Get(-1) on empty", "subscript -1", func() { Array().Set(-1, "x") })
	if show(a) != "A,c,C" {
		t.Errorf("failed writes changed the array: %s", show(a))
	}
}

func TestWritesThroughNonReferences(t *testing.T) {
	s := Scalar("str")
	expectPanic(t, "array element of a string", "Not an ARRAY reference", func() { s.AV().Get(0).Set(1) })
	expectPanic(t, "AV.Set on a string", "Not an ARRAY reference", func() { s.AV().Set(0, 1) })
	expectPanic(t, "Push on a string", "Not an ARRAY reference", func() { s.AV().Push(1) })
	expectPanic(t, "hash element of a string", "Not a HASH reference", func() { s.HV().Get("k").Set(1) })
	expectPanic(t, "HV.Set on a string", "Not a HASH reference", func() { s.HV().Set("k", 1) })
	expectPanic(t, "write via Path", "Not a HASH reference", func() { s.Path("k").Set(1) })
	if s.Str() != "str" {
		t.Errorf("s changed to %q", s)
	}
	// Reads are still fine.
	if s.Path("k", 0).Defined() || s.AV().Get(0).Defined() || s.AV().Length() != 0 {
		t.Error("reads through a non-reference should be undef")
	}
}

func TestHolesAndExists(t *testing.T) {
	a := Array()
	a.Set(3, "d")
	if a.Length() != 4 || show(a) != "-,-,-,d" {
		t.Fatalf("$a[3] = 'd' on empty: %s", show(a))
	}
	if a.Exists(0) || a.Exists(-2) || !a.Exists(3) || !a.Exists(-1) || a.Exists(4) || a.Exists(-5) {
		t.Error("Exists should be true only for elements that were set")
	}

	// An element explicitly set to undef exists.
	a.Set(1, nil)
	if !a.Exists(1) {
		t.Error("$a[1] = undef should exist")
	}

	// Writing to a hole fills it.
	a.Get(0).Set("a")
	if !a.Exists(0) || a.Get(0).Str() != "a" {
		t.Error("writing to a hole should create it")
	}
}

func TestDelete(t *testing.T) {
	a := Array("a", "b", "c", "d")
	if got := a.Delete(1).Str(); got != "b" {
		t.Errorf("delete $a[1] = %q", got)
	}
	if a.Length() != 4 || a.Exists(1) || show(a) != "a,-,c,d" {
		t.Errorf("deleting from the middle should leave a hole: %s", show(a))
	}

	// Deleting the last element shrinks the array past any holes before it.
	a.Delete(2)
	if got := a.Delete(-1).Str(); got != "d" {
		t.Errorf("delete $a[-1] = %q", got)
	}
	if a.Length() != 1 || show(a) != "a" {
		t.Errorf("deleting the last element should trim trailing holes: %s", show(a))
	}

	if a.Delete(5).Defined() || a.Delete(-5).Defined() || a.Length() != 1 {
		t.Error("deleting out of range should return undef and change nothing")
	}
}

func TestLastIndex(t *testing.T) {
	a := Array("a", "b", "c")
	if a.LastIndex() != 2 || Array().LastIndex() != -1 {
		t.Error("$#a is Length - 1")
	}
	a.SetLastIndex(0)
	if show(a) != "a" {
		t.Errorf("$#a = 0: %s", show(a))
	}
	a.SetLastIndex(2)
	if a.Length() != 3 || a.Exists(1) || a.Exists(2) {
		t.Errorf("extending $#a should add holes: %s", show(a))
	}
	a.SetLastIndex(-5)
	if a.Length() != 0 {
		t.Error("$#a = -5 should empty the array")
	}
}

func TestSplice(t *testing.T) {
	tests := []struct {
		name           string
		offset, length int
		list           []any
		wantRemoved    string
		wantArrayAfter string
	}{
		{"remove middle", 1, 2, nil, "b,c", "a,d,e"},
		{"replace middle", 1, 2, []any{"X", "Y", "Z"}, "b,c", "a,X,Y,Z,d,e"},
		{"insert only", 2, 0, []any{"X"}, "", "a,b,X,c,d,e"},
		{"negative offset", -2, 1, nil, "d", "a,b,c,e"},
		{"negative length", 1, -1, nil, "b,c,d", "a,e"},
		{"length past end", 3, 100, nil, "d,e", "a,b,c"},
		{"everything from 2", 2, 5, nil, "c,d,e", "a,b"},
		{"offset past end appends", 10, 0, []any{"X"}, "", "a,b,c,d,e,X"},
		{"offset at end appends", 5, 0, []any{"X"}, "", "a,b,c,d,e,X"},
		{"negative length beyond", 1, -10, nil, "", "a,b,c,d,e"},
	}
	for _, tt := range tests {
		a := Array("a", "b", "c", "d", "e")
		removed := a.Splice(tt.offset, tt.length, tt.list...)
		if show(removed) != tt.wantRemoved || show(a) != tt.wantArrayAfter {
			t.Errorf("%s: removed %q, left %q; want %q, %q",
				tt.name, show(removed), show(a), tt.wantRemoved, tt.wantArrayAfter)
		}
	}

	expectPanic(t, "offset before start", "subscript -6", func() { Array(1, 2, 3, 4, 5).Splice(-6, 1) })

	// Removed holes come back as undef.
	a := Array()
	a.Set(2, "c")
	if r := a.Splice(0, 3); r.Length() != 3 || r.Get(0).Defined() || r.Get(2).Str() != "c" {
		t.Errorf("splice over holes: %s", show(r))
	}
}

func TestSlice(t *testing.T) {
	a := Array("a", "b", "c", "d")
	if got := show(a.Slice(0, -1, 1, 10, -10)); got != "a,d,b,-,-" {
		t.Errorf("@a[0,-1,1,10,-10] = %s", got)
	}
	s := a.Slice(0)
	s.Set(0, "changed")
	if a.Get(0).Str() != "a" {
		t.Error("Slice should copy")
	}
	if a.Length() != 4 {
		t.Error("Slice shouldn't extend the array")
	}
}

func TestPopShiftHoles(t *testing.T) {
	a := Array()
	a.Set(2, "c")
	if a.Shift().Defined() || a.Length() != 2 {
		t.Error("shifting a hole should give undef")
	}
	if a.Pop().Str() != "c" || a.Pop().Defined() || a.Length() != 0 {
		t.Error("pop should return elements then holes as undef")
	}
}

func TestHolesInJSON(t *testing.T) {
	a := Array()
	a.Set(2, 3)
	if got := mustJSON(t, a); got != "[null,null,3]" {
		t.Errorf("JSON with holes = %s", got)
	}
}
