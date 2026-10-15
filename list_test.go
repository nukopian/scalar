package scalar

import (
	"math"
	"testing"
)

func TestMatchPredicates(t *testing.T) {
	bullets := Array("Write Go services", "Remote-first team", "Mentor juniors")
	remote := func(s *SV) bool { return s.M(`(?i)\bremote\b`) != nil }
	hasLetters := func(s *SV) bool { return s.M(`\p{L}`) != nil }
	salary := func(s *SV) bool { return s.Index("£") >= 0 }

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"any: one matches", bullets.AnyMatch(remote), true},
		{"any: none match", bullets.AnyMatch(salary), false},
		{"all: every one matches", bullets.AllMatch(hasLetters), true},
		{"all: one doesn't", bullets.AllMatch(remote), false},
		{"none: none match", bullets.NoneMatch(salary), true},
		{"none: one does", bullets.NoneMatch(remote), false},

		// Empty arrays, as in List::Util.
		{"any on empty", Array().AnyMatch(remote), false},
		{"all on empty", Array().AllMatch(remote), true},
		{"none on empty", Array().NoneMatch(remote), true},
		{"nil array", (*AV)(nil).AllMatch(remote), true},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	// AnyMatch stops at the first match.
	calls := 0
	Array(1, 2, 3).AnyMatch(func(s *SV) bool { calls++; return s.Int() == 1 })
	if calls != 1 {
		t.Errorf("AnyMatch made %d calls, want 1", calls)
	}
}

func TestFirstMatch(t *testing.T) {
	chunks := Array("About us", "Salary: £45,000", "Salary: £50,000")
	first := chunks.FirstMatch(func(s *SV) bool { return s.Index("£") >= 0 })
	if first.Str() != "Salary: £45,000" {
		t.Errorf("FirstMatch = %q", first)
	}

	// It's the element itself, like Get.
	first.Set("changed")
	if chunks.Get(1).Str() != "changed" {
		t.Error("FirstMatch should return the stored element")
	}

	if Array(1, 2).FirstMatch(func(s *SV) bool { return s.Int() > 5 }).Defined() {
		t.Error("no match should be undef")
	}
}

func TestMaxMinStr(t *testing.T) {
	a := Array("pear", "apple", "fig", "10", "9")
	if a.MaxStr().Str() != "pear" || a.MinStr().Str() != "10" {
		t.Errorf("MaxStr %q, MinStr %q", a.MaxStr(), a.MinStr())
	}
	// String order, not numeric: "9" sorts after "10".
	if Array(10, 9).MaxStr().Int() != 9 {
		t.Error("MaxStr compares as strings")
	}
	// Ties keep the first.
	x, y := Value("same"), Value("same")
	tied := Array(x, y)
	if tied.MaxStr() != tied.Get(0) || tied.MinStr() != tied.Get(0) {
		t.Error("ties should keep the first element")
	}
	if Array().MaxStr().Defined() || Array().MinStr().Defined() {
		t.Error("empty array should give undef")
	}
}

func TestUniqueStr(t *testing.T) {
	skills := Array("Go", "Perl", "Go", "go", "Perl", "SQL")
	if got := show(skills.UniqueStr()); got != "Go,Perl,go,SQL" {
		t.Errorf("UniqueStr = %s", got)
	}

	// Ignoring case: fold first, then remove duplicates.
	folded := skills.Map(func(s *SV) any { return s.Fc() }).UniqueStr()
	if got := show(folded); got != "go,perl,sql" {
		t.Errorf("UniqueStr of Fc = %s", got)
	}

	// Strings, so 1 and "1.0" differ; undef counts as "".
	if got := showSep(Array(1, "1", "1.0", nil, "").UniqueStr(), "|"); got != "1|1.0|-" {
		t.Errorf("UniqueStr = %s", got)
	}

	// Copies.
	a := Array("x")
	a.UniqueStr().Get(0).Set("changed")
	if a.Get(0).Str() != "x" {
		t.Error("UniqueStr should copy")
	}
}

func TestUniqueNum(t *testing.T) {
	tests := []struct {
		name string
		in   *AV
		want string
	}{
		{"numbers and numeric strings", Array(1, "1", "1.0", 2, " 2 ", 3.5, "3.50"), "1,2,3.5"},
		{"order kept", Array(3, 1, 3, 2, 1), "3,1,2"},
		{"0 and -0", Array(0, math.Copysign(0, -1), "0", "-0"), "0"},
		{"large integers exact", Array("9007199254740993", "9007199254740992"), "9007199254740993,9007199254740992"},
		{"NaNs are one", Array(math.NaN(), "nan", math.NaN()), "NaN"},
		{"non-numeric strings are 0", Array("abc", 0, "xyz"), "abc"},
		{"empty", Array(), ""},
	}
	for _, tt := range tests {
		if got := show(tt.in.UniqueNum()); got != tt.want {
			t.Errorf("%s: UniqueNum = %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestMaxMinNum(t *testing.T) {
	a := Array(3, "10", 2.5, " 7 ")
	if a.MaxNum() != a.Get(1) || a.MinNum() != a.Get(2) {
		t.Errorf("MaxNum %q, MinNum %q", a.MaxNum(), a.MinNum())
	}
	// Numeric, not string, order: the opposite of MaxStr here.
	if Array(10, 9).MaxNum().Int() != 10 || Array(10, 9).MaxStr().Int() != 9 {
		t.Error("MaxNum compares as numbers")
	}
	// Large integers compare exactly.
	big := Array("9007199254740992", "9007199254740993")
	if big.MaxNum().Str() != "9007199254740993" {
		t.Errorf("MaxNum of large integers = %q", big.MaxNum())
	}
	// Ties keep the first.
	tied := Array(1, "1.0")
	if tied.MaxNum() != tied.Get(0) || tied.MinNum() != tied.Get(0) {
		t.Error("ties should keep the first element")
	}
	// NaN never wins a comparison, so it only results if it comes first.
	if !math.IsNaN(Array(math.NaN(), 5).MaxNum().Num()) {
		t.Error("a leading NaN should be the result, as in List::Util")
	}
	if Array(5, math.NaN(), 7).MaxNum().Int() != 7 || Array(5, math.NaN(), 3).MinNum().Int() != 3 {
		t.Error("a later NaN should be ignored")
	}
	if Array().MaxNum().Defined() || Array().MinNum().Defined() {
		t.Error("empty array should give undef")
	}
}

func TestSplitAt(t *testing.T) {
	list := Array("one", "two", "three", "four", "five")

	// my ($one, $two, $three, @tail) = @list
	h, tail := list.SplitAtSlice(3)
	one, two, three := h[0], h[1], h[2]
	if one.Str() != "one" || two.Str() != "two" || three.Str() != "three" || show(tail) != "four,five" {
		t.Errorf("SplitAtSlice(3) = %v, %s", h, show(tail))
	}

	head, tail := list.SplitAt(3)
	if show(head) != "one,two,three" || show(tail) != "four,five" {
		t.Errorf("SplitAt(3) = %s, %s", show(head), show(tail))
	}

	// Shorter than n: the head is padded with undef, the tail is empty.
	h, tail = Array("a").SplitAtSlice(3)
	if len(h) != 3 || h[0].Str() != "a" || h[1].Defined() || h[2].Defined() || tail.Length() != 0 {
		t.Errorf("short SplitAtSlice = %v, %s", h, show(tail))
	}
	head, tail = Array("a").SplitAt(3)
	if head.Length() != 3 || show(head) != "a,-,-" || tail.Length() != 0 {
		t.Errorf("short SplitAt = %s, %s", show(head), show(tail))
	}

	// Edge cases.
	if head, tail := list.SplitAt(0); head.Length() != 0 || tail.Length() != 5 {
		t.Error("SplitAt(0) should give everything as the tail")
	}
	if h, tail := (*AV)(nil).SplitAtSlice(2); len(h) != 2 || h[0].Defined() || tail.Length() != 0 {
		t.Error("nil array")
	}
	holey := Array()
	holey.Set(2, "c")
	if head, _ := holey.SplitAt(3); show(head) != "-,-,c" {
		t.Errorf("holes = %s", show(head))
	}

	// Copies, and the list itself is unchanged.
	h, tail = list.SplitAtSlice(1)
	h[0].Set("changed")
	tail.Get(0).Set("changed")
	tail.Push("extra")
	if show(list) != "one,two,three,four,five" {
		t.Errorf("SplitAt changed the list: %s", show(list))
	}
	// Writing to a padding undef doesn't add to the list either.
	h, _ = Array().SplitAtSlice(1)
	h[0].Set("x")
	if list.Length() != 5 {
		t.Error("padding should be detached")
	}

	expectPanic(t, "negative count", "SplitAt count -1 is negative", func() { list.SplitAt(-1) })
}

func TestSumAndProduct(t *testing.T) {
	a := Array(1, "2", 3.5)
	if got := a.Sum(); got.Num() != 6.5 || got.Kind() != Float {
		t.Errorf("Sum = %v (%v)", got, got.Kind())
	}
	if got := Array(1, 2, 3).Sum(); got.Int() != 6 || got.Kind() != Int {
		t.Errorf("integer Sum = %v (%v), want an exact integer", got, got.Kind())
	}
	if got := Array().Sum(); got.Defined() {
		t.Errorf("Sum of nothing = %v, want undef", got)
	}
	if got := Array().Sum0(); got.Int() != 0 || !got.Defined() {
		t.Errorf("Sum0 of nothing = %v, want 0", got)
	}
	if got := Array(2, 3, 4).Product(); got.Int() != 24 {
		t.Errorf("Product = %v", got)
	}
	if got := Array().Product(); got.Int() != 1 {
		t.Errorf("Product of nothing = %v, want 1", got)
	}
	big := Array(math.MaxInt64, 1).Sum()
	if big.Kind() != Float {
		t.Errorf("an overflowing Sum should become a float, got %v", big.Kind())
	}
}

func TestReduce(t *testing.T) {
	words := Array("go", "perl", "c")
	longest := words.Reduce(func(a, b *SV) any {
		if b.Length() > a.Length() {
			return b
		}
		return a
	})
	if longest.Str() != "perl" {
		t.Errorf("longest = %v", longest)
	}
	concat := Array("a", "b", "c").Reduce(func(a, b *SV) any { return a.Str() + b.Str() })
	if concat.Str() != "abc" {
		t.Errorf("concat = %v", concat)
	}
	if got := Array().Reduce(nil); got.Defined() {
		t.Errorf("Reduce of nothing = %v, want undef", got)
	}
	one := Array("only")
	got := one.Reduce(nil) // f isn't called for one element
	got.Set("changed")
	if got.Str() != "changed" || one.Get(0).Str() != "only" {
		t.Error("Reduce of one element should return a copy of it")
	}
}
