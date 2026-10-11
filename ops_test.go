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
		{"2+3", Scalar(2).Add(3), "5", Int},
		{`"10"+"5"`, Scalar("10").Add("5"), "15", Int},
		{`"3 apples"+2`, Scalar("3 apples").Add(2), "5", Int},
		{`"abc"+1`, Scalar("abc").Add(1), "1", Int},
		{"undef+1", Scalar(nil).Add(1), "1", Int},
		{"true+true", Scalar(true).Add(true), "2", Int},
		{"1.5+1", Scalar(1.5).Add(1), "2.5", Float},
		{`"1.5"+"1.5"`, Scalar("1.5").Add("1.5"), "3", Float},
		{"0.1+0.2", Scalar(0.1).Add(0.2), "0.3", Float},
		{"MaxInt64+1", Scalar(math.MaxInt64).Add(1), "9.22337203685478e+18", Float},
		{"MinInt64-1", Scalar(math.MinInt64).Sub(1), "-9.22337203685478e+18", Float},
		{"5-8", Scalar(5).Sub(8), "-3", Int},
		{"6*7", Scalar(6).Mul(7), "42", Int},
		{"MaxInt64*2", Scalar(math.MaxInt64).Mul(2), "1.84467440737096e+19", Float},
		{"MinInt64*-1", Scalar(math.MinInt64).Mul(-1), "9.22337203685478e+18", Float},
		{"6/3", Scalar(6).Div(3), "2", Int},
		{"7/2", Scalar(7).Div(2), "3.5", Float},
		{"1/3", Scalar(1).Div(3), "0.333333333333333", Float},
		{"MinInt64/-1", Scalar(math.MinInt64).Div(-1), "9.22337203685478e+18", Float},
		{"7%3", Scalar(7).Mod(3), "1", Int},
		{"-7%3", Scalar(-7).Mod(3), "2", Int},
		{"7%-3", Scalar(7).Mod(-3), "-2", Int},
		{"-7%-3", Scalar(-7).Mod(-3), "-1", Int},
		{"7.9%3", Scalar(7.9).Mod(3), "1", Int},
		{"2**10", Scalar(2).Pow(10), "1024", Int},
		{"2**-1", Scalar(2).Pow(-1), "0.5", Float},
		{"0**0", Scalar(0).Pow(0), "1", Int},
		{"2**64", Scalar(2).Pow(64), "1.84467440737096e+19", Float},
		{"-2**63", Scalar(-2).Pow(63), "-9223372036854775808", Int},
		{"4**0.5", Scalar(4).Pow(0.5), "2", Float},
		{"operand is *SV", Scalar(2).Add(Scalar("40")), "42", Int},
	})
}

func TestNeg(t *testing.T) {
	checkOps(t, []opCase{
		{"-5", Scalar(5).Neg(), "-5", Int},
		{"-(-5)", Scalar(-5).Neg(), "5", Int},
		{"-MinInt64", Scalar(math.MinInt64).Neg(), "9.22337203685478e+18", Float},
		{"-1.5", Scalar(1.5).Neg(), "-1.5", Float},
		{"-0.0", Scalar(0.0).Neg(), "-0", Float},
		{`-"foo"`, Scalar("foo").Neg(), "-foo", String},
		{`-"_x"`, Scalar("_x").Neg(), "-_x", String},
		{`-"-foo"`, Scalar("-foo").Neg(), "+foo", String},
		{`-"+foo"`, Scalar("+foo").Neg(), "-foo", String},
		{`-"+5"`, Scalar("+5").Neg(), "-5", String},
		{`-"-5"`, Scalar("-5").Neg(), "5", Int},
		{`-"10"`, Scalar("10").Neg(), "-10", Int},
		{`-"2.5"`, Scalar("2.5").Neg(), "-2.5", Float},
		{`-"10 green bottles"`, Scalar("10 green bottles").Neg(), "-10", Int},
		{"-undef", Scalar(nil).Neg(), "0", Int},
	})
}

func TestStringOps(t *testing.T) {
	checkOps(t, []opCase{
		{`"foo"."bar"`, Scalar("foo").Concat("bar"), "foobar", String},
		{`1 . 2`, Scalar(1).Concat(2), "12", String},
		{`undef . "x"`, Scalar(nil).Concat("x"), "x", String},
		{`1.50 . ""`, Scalar(1.50).Concat(""), "1.5", String},
		{`"ab" x 3`, Scalar("ab").Repeat(3), "ababab", String},
		{`"ab" x "2"`, Scalar("ab").Repeat("2"), "abab", String},
		{`"ab" x 0`, Scalar("ab").Repeat(0), "", String},
		{`"ab" x -1`, Scalar("ab").Repeat(-1), "", String},
		{`5 x 3`, Scalar(5).Repeat(3), "555", String},
	})
}

func TestIncDec(t *testing.T) {
	inc := func(x any) *SV { return Scalar(x).Inc() }
	dec := func(x any) *SV { return Scalar(x).Dec() }
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
	if h.Get("words").Path("perl").Int() != 1 {
		t.Error("$h{words}{perl}++ didn't autovivify")
	}
}

func TestComparisons(t *testing.T) {
	nan := Scalar(math.NaN())
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{`"10" == 10.0`, Scalar("10").NumEq(10.0), true},
		{`"1e1" == 10`, Scalar("1e1").NumEq(10), true},
		{`"abc" == 0`, Scalar("abc").NumEq(0), true},
		{`"10" eq "10.0"`, Scalar("10").Eq("10.0"), false},
		{`10 eq "10"`, Scalar(10).Eq("10"), true},
		{`"abc" ne "abd"`, Scalar("abc").Ne("abd"), true},
		{`2 < 10`, Scalar(2).NumLt(10), true},
		{`"2" lt "10"`, Scalar("2").Lt("10"), false},
		{`"10" le "10"`, Scalar("10").Le("10"), true},
		{`"b" gt "a"`, Scalar("b").Gt("a"), true},
		{`"a" ge "b"`, Scalar("a").Ge("b"), false},
		{`3 >= 3`, Scalar(3).NumGe(3), true},
		{`3 > 3`, Scalar(3).NumGt(3), false},
		{`2.5 <= 3`, Scalar(2.5).NumLe(3), true},
		{`undef == 0`, Scalar(nil).NumEq(0), true},
		{`undef eq ""`, Scalar(nil).Eq(""), true},
		{"large ints compare exactly", Scalar("9223372036854775807").NumGt("9223372036854775806"), true},
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
	expectPanic(t, "1/0", div, func() { Scalar(1).Div(0) })
	expectPanic(t, `1/"0.0"`, div, func() { Scalar(1).Div("0.0") })
	expectPanic(t, `1/"abc"`, div, func() { Scalar(1).Div("abc") })
	expectPanic(t, "1%0", mod, func() { Scalar(1).Mod(0) })
	expectPanic(t, "1%0.5", mod, func() { Scalar(1).Mod(0.5) })
}

func TestOperandsUnchanged(t *testing.T) {
	a, b := Scalar(2), Scalar(3)
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
		{"0++", Scalar(0).Inc(), "1", Int},
		{"-1++", Scalar(-1).Inc(), "0", Int},
		{"MaxInt64-1 ++", Scalar(int64(math.MaxInt64 - 1)).Inc(), "9223372036854775807", Int},
		{"MaxInt64++ overflows to float", Scalar(int64(math.MaxInt64)).Inc(), "9.22337203685478e+18", Float},
		{"MinInt64-- overflows to float", Scalar(int64(math.MinInt64)).Dec(), "-9.22337203685478e+18", Float},
		{"MinInt64+1 --", Scalar(int64(math.MinInt64 + 1)).Dec(), "-9223372036854775808", Int},
		{"undef--", Scalar(nil).Dec(), "-1", Int},
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
