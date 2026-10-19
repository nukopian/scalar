package scalar

import (
	"math"
	"testing"
)

// opCase checks both the printed result and whether it stayed an integer.
type opCase struct {
	name string
	got  *SV
	want string
	kind Kind
}

func checkOps(t *testing.T, tests []opCase) {
	t.Helper()
	for _, tt := range tests {
		if tt.got.Str() != tt.want || tt.got.Kind() != tt.kind {
			t.Errorf("%s = %q (kind %d), want %q (kind %d)", tt.name, tt.got.Str(), tt.got.Kind(), tt.want, tt.kind)
		}
	}
}

func TestArithmetic(t *testing.T) {
	checkOps(t, []opCase{
		{"2+3", Value(2).Add(3), "5", Int},
		{`"10"+"5"`, Value("10").Add("5"), "15", Int},
		{`"3 apples"+2`, Value("3 apples").Add(2), "5", Int},
		{`"abc"+1`, Value("abc").Add(1), "1", Int},
		{"undef+1", Value(nil).Add(1), "1", Int},
		{"true+true", Value(true).Add(true), "2", Int},
		{"1.5+1", Value(1.5).Add(1), "2.5", Float},
		{`"1.5"+"1.5"`, Value("1.5").Add("1.5"), "3", Float},
		{"0.1+0.2", Value(0.1).Add(0.2), "0.3", Float},
		{"MaxInt64+1", Value(math.MaxInt64).Add(1), "9.22337203685478e+18", Float},
		{"MinInt64-1", Value(math.MinInt64).Sub(1), "-9.22337203685478e+18", Float},
		{"5-8", Value(5).Sub(8), "-3", Int},
		{"6*7", Value(6).Mul(7), "42", Int},
		{"MaxInt64*2", Value(math.MaxInt64).Mul(2), "1.84467440737096e+19", Float},
		{"MinInt64*-1", Value(math.MinInt64).Mul(-1), "9.22337203685478e+18", Float},
		{"6/3", Value(6).Div(3), "2", Int},
		{"7/2", Value(7).Div(2), "3.5", Float},
		{"1/3", Value(1).Div(3), "0.333333333333333", Float},
		{"MinInt64/-1", Value(math.MinInt64).Div(-1), "9.22337203685478e+18", Float},
		{"7%3", Value(7).Mod(3), "1", Int},
		{"-7%3", Value(-7).Mod(3), "2", Int},
		{"7%-3", Value(7).Mod(-3), "-2", Int},
		{"-7%-3", Value(-7).Mod(-3), "-1", Int},
		{"7.9%3", Value(7.9).Mod(3), "1", Int},
		{"2**10", Value(2).Pow(10), "1024", Int},
		{"2**-1", Value(2).Pow(-1), "0.5", Float},
		{"0**0", Value(0).Pow(0), "1", Int},
		{"2**64", Value(2).Pow(64), "1.84467440737096e+19", Float},
		{"-2**63", Value(-2).Pow(63), "-9223372036854775808", Int},
		{"4**0.5", Value(4).Pow(0.5), "2", Float},
		{"operand is *SV", Value(2).Add(Value("40")), "42", Int},
	})
}

func TestNeg(t *testing.T) {
	checkOps(t, []opCase{
		{"-5", Value(5).Neg(), "-5", Int},
		{"-(-5)", Value(-5).Neg(), "5", Int},
		{"-MinInt64", Value(math.MinInt64).Neg(), "9.22337203685478e+18", Float},
		{"-1.5", Value(1.5).Neg(), "-1.5", Float},
		{"-0.0", Value(0.0).Neg(), "-0", Float},
		{`-"foo"`, Value("foo").Neg(), "-foo", String},
		{`-"_x"`, Value("_x").Neg(), "-_x", String},
		{`-"-foo"`, Value("-foo").Neg(), "+foo", String},
		{`-"+foo"`, Value("+foo").Neg(), "-foo", String},
		{`-"+5"`, Value("+5").Neg(), "-5", String},
		{`-"-5"`, Value("-5").Neg(), "5", Int},
		{`-"10"`, Value("10").Neg(), "-10", Int},
		{`-"2.5"`, Value("2.5").Neg(), "-2.5", Float},
		{`-"10 green bottles"`, Value("10 green bottles").Neg(), "-10", Int},
		{"-undef", Value(nil).Neg(), "0", Int},
	})
}

func TestStringOps(t *testing.T) {
	checkOps(t, []opCase{
		{`"foo"."bar"`, Value("foo").Concat("bar"), "foobar", String},
		{`1 . 2`, Value(1).Concat(2), "12", String},
		{`undef . "x"`, Value(nil).Concat("x"), "x", String},
		{`1.50 . ""`, Value(1.50).Concat(""), "1.5", String},
		{`"ab" x 3`, Value("ab").Repeat(3), "ababab", String},
		{`"ab" x "2"`, Value("ab").Repeat("2"), "abab", String},
		{`"ab" x 0`, Value("ab").Repeat(0), "", String},
		{`"ab" x -1`, Value("ab").Repeat(-1), "", String},
		{`5 x 3`, Value(5).Repeat(3), "555", String},
	})
}

func TestIncDec(t *testing.T) {
	inc := func(x any) *SV { return Value(x).Inc() }
	dec := func(x any) *SV { return Value(x).Dec() }
	checkOps(t, []opCase{
		{`"aa"++`, inc("aa"), "ab", String},
		{`"az"++`, inc("az"), "ba", String},
		{`"Az"++`, inc("Az"), "Ba", String},
		{`"zz"++`, inc("zz"), "aaa", String},
		{`"Zz"++`, inc("Zz"), "AAa", String},
		{`"ZZ"++`, inc("ZZ"), "AAA", String},
		{`"a9"++`, inc("a9"), "b0", String},
		{`"Zz99"++`, inc("Zz99"), "AAa00", String},
		{`"zz99"++`, inc("zz99"), "aaa00", String},
		{`"9"++`, inc("9"), "10", Int},
		{`"09"++`, inc("09"), "10", Int},
		{`"1.5"++`, inc("1.5"), "2.5", Float},
		{`"a-b"++`, inc("a-b"), "1", Int},
		{`"9a"++`, inc("9a"), "10", Int},
		{`""++`, inc(""), "1", Int},
		{"undef++", inc(nil), "1", Int},
		{"41++", inc(41), "42", Int},
		{"MaxInt64++", inc(math.MaxInt64), "9.22337203685478e+18", Float},
		{`"aa"--`, dec("aa"), "-1", Int},
		{"undef--", dec(nil), "-1", Int},
		{"43--", dec(43), "42", Int},
		{"1.5--", dec(1.5), "0.5", Float},
	})

	// Inc and Dec modify in place, and write through autovivification.
	h := Hash()
	h.Get("count").Inc()
	h.Get("count").Inc()
	if h.Get("count").Int() != 2 {
		t.Errorf("$h{count}++ twice = %s", h.Get("count"))
	}
	h.Get("words").HV().Get("perl").Inc()
	if h.Get("words").Get("perl").Int() != 1 {
		t.Error("$h{words}{perl}++ didn't autovivify")
	}
}

func TestComparisons(t *testing.T) {
	nan := Value(math.NaN())
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{`"10" == 10.0`, Value("10").NumEq(10.0), true},
		{`"1e1" == 10`, Value("1e1").NumEq(10), true},
		{`"abc" == 0`, Value("abc").NumEq(0), true},
		{`"10" eq "10.0"`, Value("10").Eq("10.0"), false},
		{`10 eq "10"`, Value(10).Eq("10"), true},
		{`"abc" ne "abd"`, Value("abc").Ne("abd"), true},
		{`2 < 10`, Value(2).NumLt(10), true},
		{`"2" lt "10"`, Value("2").Lt("10"), false},
		{`"10" le "10"`, Value("10").Le("10"), true},
		{`"b" gt "a"`, Value("b").Gt("a"), true},
		{`"a" ge "b"`, Value("a").Ge("b"), false},
		{`3 >= 3`, Value(3).NumGe(3), true},
		{`3 > 3`, Value(3).NumGt(3), false},
		{`2.5 <= 3`, Value(2.5).NumLe(3), true},
		{`undef == 0`, Value(nil).NumEq(0), true},
		{`undef eq ""`, Value(nil).Eq(""), true},
		{"large ints compare exactly", Value("9223372036854775807").NumGt("9223372036854775806"), true},
		{"NaN == NaN", nan.NumEq(nan), false},
		{"NaN != NaN", nan.NumNe(nan), true},
		{"NaN < 1", nan.NumLt(1), false},
		{"NaN >= 1", nan.NumGe(1), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestDivModByZeroPanic(t *testing.T) {
	const div, mod = "Illegal division by zero", "Illegal modulus zero"
	expectPanic(t, "1/0", div, func() { Value(1).Div(0) })
	expectPanic(t, `1/"0.0"`, div, func() { Value(1).Div("0.0") })
	expectPanic(t, `1/"abc"`, div, func() { Value(1).Div("abc") })
	expectPanic(t, "1%0", mod, func() { Value(1).Mod(0) })
	expectPanic(t, "1%0.5", mod, func() { Value(1).Mod(0.5) })
}

func TestOperandsUnchanged(t *testing.T) {
	a, b := Value(2), Value(3)
	a.Add(b)
	a.Concat(b)
	b.Neg()
	if a.Int() != 2 || b.Int() != 3 {
		t.Error("operators modified their operands")
	}
}

// The integer fast paths of Inc and Dec behave exactly like the general ones.
func TestIncDecFastPaths(t *testing.T) {
	checkOps(t, []opCase{
		{"0++", Value(0).Inc(), "1", Int},
		{"-1++", Value(-1).Inc(), "0", Int},
		{"MaxInt64-1 ++", Value(int64(math.MaxInt64 - 1)).Inc(), "9223372036854775807", Int},
		{"MaxInt64++ overflows to float", Value(int64(math.MaxInt64)).Inc(), "9.22337203685478e+18", Float},
		{"MinInt64-- overflows to float", Value(int64(math.MinInt64)).Dec(), "-9.22337203685478e+18", Float},
		{"MinInt64+1 --", Value(int64(math.MinInt64 + 1)).Dec(), "-9223372036854775808", Int},
		{"undef--", Value(nil).Dec(), "-1", Int},
	})

	// $h{$k}++ still writes through to a missing key, and counts.
	h := Hash()
	for range 3 {
		h.Get("k").Inc()
	}
	if h.Get("k").Int() != 3 {
		t.Errorf("$h{k}++ x3 = %s", h.Get("k"))
	}
	h.Get("d").Dec()
	if !h.Exists("d") || h.Get("d").Int() != -1 {
		t.Error("$h{d}-- should create d")
	}
}

func TestAssignmentOperators(t *testing.T) {
	inPlace := func(start any, op func(*SV) *SV) *SV {
		s := Value(start)
		if got := op(s); got != s {
			t.Errorf("should return the same scalar")
		}
		return s
	}
	checkOps(t, []opCase{
		{"10 += 5", inPlace(10, func(s *SV) *SV { return s.AddInPlace(5) }), "15", Int},
		{`"10" += "5"`, inPlace("10", func(s *SV) *SV { return s.AddInPlace("5") }), "15", Int},
		{"10 -= 15", inPlace(10, func(s *SV) *SV { return s.SubInPlace(15) }), "-5", Int},
		{"6 *= 7", inPlace(6, func(s *SV) *SV { return s.MulInPlace(7) }), "42", Int},
		{"7 /= 2", inPlace(7, func(s *SV) *SV { return s.DivInPlace(2) }), "3.5", Float},
		{"-7 %= 3", inPlace(-7, func(s *SV) *SV { return s.ModInPlace(3) }), "2", Int},
		{"2 **= 10", inPlace(2, func(s *SV) *SV { return s.PowInPlace(10) }), "1024", Int},
		{`"foo" .= "bar"`, inPlace("foo", func(s *SV) *SV { return s.ConcatInPlace("bar") }), "foobar", String},
		{`"ab" x= 3`, inPlace("ab", func(s *SV) *SV { return s.RepeatInPlace(3) }), "ababab", String},
		{"undef += 3", inPlace(nil, func(s *SV) *SV { return s.AddInPlace(3) }), "3", Int},
		{`undef .= "x"`, inPlace(nil, func(s *SV) *SV { return s.ConcatInPlace("x") }), "x", String},
		{"MaxInt64 += 1 overflows to float", inPlace(int64(math.MaxInt64), func(s *SV) *SV { return s.AddInPlace(1) }), "9.22337203685478e+18", Float},
	})

	// $total{$k} += $n creates the key, then accumulates.
	totals := Hash()
	for _, n := range []int{5, 10, 20} {
		totals.Get("felt").AddInPlace(n)
	}
	totals.Get("log").ConcatInPlace("a").ConcatInPlace("b") // chains
	if totals.Get("felt").Int() != 35 || totals.Get("log").Str() != "ab" {
		t.Errorf("accumulating = %s", mustJSON(t, totals))
	}

	// The right-hand side is left alone.
	y := Value(5)
	Value(1).AddInPlace(y)
	if y.Int() != 5 {
		t.Error("the operand changed")
	}

	// A panicking operation leaves s unchanged.
	s := Value(10)
	expectPanic(t, "/= 0", "Illegal division by zero", func() { s.DivInPlace(0) })
	if s.Int() != 10 {
		t.Errorf("s changed to %s after a failed /=", s)
	}

	// Writes before the start of an array panic, as with Set.
	expectPanic(t, "+= before the start", "non-creatable", func() { Array().Get(-1).AddInPlace(1) })
}
