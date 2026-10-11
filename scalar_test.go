package scalar

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestStr(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{true, "1"},
		{false, ""},
		{42, "42"},
		{-7, "-7"},
		{1.0, "1"},
		{0.1 + 0.2, "0.3"},
		{1e21, "1e+21"},
		{1234567.5, "1234567.5"},
		{math.Inf(1), "Inf"},
		{math.NaN(), "NaN"},
		{"hello", "hello"},
		{uint64(math.MaxUint64), "1.84467440737096e+19"},
	}
	for _, tt := range tests {
		if got := Scalar(tt.in).Str(); got != tt.want {
			t.Errorf("Scalar(%#v).Str() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNumOK(t *testing.T) {
	tests := []struct {
		in     any
		want   float64
		wantOK bool
	}{
		{nil, 0, false},
		{true, 1, true},
		{3, 3, true},
		{2.5, 2.5, true},
		{"42", 42, true},
		{"  42  ", 42, true},
		{"\n-3.5\t", -3.5, true},
		{"+7", 7, true},
		{"1e3", 1000, true},
		{".5", 0.5, true},
		{"5.", 5, true},
		{"42abc", 42, false},
		{"1e", 1, false},
		{"abc", 0, false},
		{"", 0, false},
		{"0x1A", 0, false},
		{"0 but true", 0, true},
		{"0E0", 0, true},
		{"inf", math.Inf(1), true},
		{"-Infinity", math.Inf(-1), true},
		{"1e999", math.Inf(1), true},
	}
	for _, tt := range tests {
		got, ok := Scalar(tt.in).NumOK()
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Scalar(%#v).NumOK() = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
	if f, ok := Scalar("nan").NumOK(); !math.IsNaN(f) || !ok {
		t.Errorf(`Scalar("nan").NumOK() = (%v, %v), want (NaN, true)`, f, ok)
	}
}

func TestIntOK(t *testing.T) {
	tests := []struct {
		in     any
		want   int64
		wantOK bool
	}{
		{nil, 0, false},
		{false, 0, true},
		{3.9, 3, true},
		{-3.9, -3, true},
		{"9223372036854775807", math.MaxInt64, true},
		{"3.7", 3, true},
		{"12 monkeys", 12, false},
		{"nope", 0, false},
		{1e30, math.MaxInt64, false},
		{math.NaN(), 0, false},
	}
	for _, tt := range tests {
		got, ok := Scalar(tt.in).IntOK()
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Scalar(%#v).IntOK() = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		in   any
		want bool
	}{
		{nil, false},
		{"", false},
		{"0", false},
		{0, false},
		{0.0, false},
		{false, false},
		{"0.0", true},
		{"0E0", true},
		{"00", true},
		{" ", true},
		{"a", true},
		{1, true},
		{math.NaN(), true},
		{Array(), true},
		{Hash(), true},
	}
	for _, tt := range tests {
		if got := Scalar(tt.in).Bool(); got != tt.want {
			t.Errorf("Scalar(%#v).Bool() = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestHashAutoviv(t *testing.T) {
	h := Hash()

	// Reading a missing leaf doesn't create it.
	if h.Get("missing").Defined() || h.Exists("missing") {
		t.Fatal("reading a missing key created it")
	}

	// Dereferencing through a missing key creates the intermediate hash,
	// but not the leaf: $h{a}{b} in rvalue context.
	if h.Get("a").HV().Get("b").Defined() {
		t.Fatal("leaf should be undef")
	}
	if !h.Exists("a") || h.Get("a").Ref() != "HASH" {
		t.Fatal("intermediate hash was not autovivified")
	}
	if h.Get("a").HV().Exists("b") {
		t.Fatal("leaf was autovivified by a read")
	}

	// $h{x}{y}[2]{z} = 1
	h.Get("x").HV().Get("y").AV().Get(2).HV().Set("z", 1)
	got := h.Get("x").HV().Get("y").AV()
	if got.Len() != 3 || got.Get(0).Defined() || got.Get(2).HV().Get("z").Int() != 1 {
		t.Fatalf("deep autoviv produced %s", mustJSON(t, h))
	}

	// Writing through a detached read attaches it.
	s := h.Get("late")
	s.Set("now")
	if h.Get("late").Str() != "now" {
		t.Fatal("Set on a missing key didn't store it")
	}
}

func TestPath(t *testing.T) {
	resp, err := FromJSON([]byte(`{"content":[{"type":"text","text":"hi"}],"usage":{"output_tokens":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Path("content", 0, "text").Str(); got != "hi" {
		t.Errorf("text = %q", got)
	}
	if got := resp.Path("content", -1, "type").Str(); got != "text" {
		t.Errorf("type = %q", got)
	}
	if got := resp.Path("usage", "output_tokens").Int(); got != 12 {
		t.Errorf("output_tokens = %d", got)
	}
	// A string can't be dereferenced: reads yield undef rather than panicking.
	if resp.Path("content", 0, "text", "nope").Defined() {
		t.Error("deref of a string should be undef")
	}
}

func TestArray(t *testing.T) {
	a := Array(3, "10", 2)
	a.Push(1).Unshift("first")
	if got := a.Join(","); got != "first,3,10,2,1" {
		t.Fatalf("Join = %q", got)
	}
	if got := a.Shift().Str(); got != "first" {
		t.Errorf("Shift = %q", got)
	}
	if got := a.Pop().Int(); got != 1 {
		t.Errorf("Pop = %d", got)
	}
	if got := a.Sort(nil).Join(","); got != "10,2,3" {
		t.Errorf("string Sort = %q", got)
	}
	if got := a.Sort(NumCmp).Join(","); got != "2,3,10" {
		t.Errorf("numeric Sort = %q", got)
	}
	if got := a.Reverse().Join(","); got != "2,10,3" {
		t.Errorf("Reverse = %q", got)
	}
	if got := a.Map(func(s *SV) any { return s.Int() * 2 }).Join(","); got != "6,20,4" {
		t.Errorf("Map = %q", got)
	}
	if got := a.Grep(func(s *SV) bool { return s.Int() > 2 }).Join(","); got != "3,10" {
		t.Errorf("Grep = %q", got)
	}
	if got := a.Get(-1).Int(); got != 2 {
		t.Errorf("Get(-1) = %d", got)
	}
	if a.Get(-10).Defined() || a.Get(99).Defined() || a.Len() != 3 {
		t.Error("out-of-range reads should be undef and not extend the array")
	}
	a.Set(5, "x")
	if a.Len() != 6 || a.Get(4).Defined() {
		t.Errorf("Set(5) should extend with undef, got %s", mustJSON(t, a))
	}
	if Array().Pop().Defined() {
		t.Error("Pop on empty should be undef")
	}
}

func TestCopiesAreValues(t *testing.T) {
	a := Array(1, 2)
	b := a.Sort(nil)
	b.Get(0).Set(99)
	if a.Get(0).Int() != 1 {
		t.Error("Sort result aliases the original elements")
	}

	// Copying a reference shares the referent, as in Perl.
	ref := Scalar(a)
	Scalar(ref).AV().Push(3)
	if a.Len() != 3 {
		t.Error("copied reference should share the array")
	}
}

func TestHashOrder(t *testing.T) {
	h := Hash("a", 1, "b", 2, "c", 3, "d", 4, "e", 5, "orphan")
	if h.Len() != 6 || h.Get("orphan").Defined() || !h.Exists("orphan") {
		t.Fatal("odd-length Hash should map the last key to undef")
	}

	check := func() {
		t.Helper()
		keys, vals := h.Keys(), h.Values()
		var iterKeys []string
		for k, v := range h.All() {
			iterKeys = append(iterKeys, k)
			if v != h.Get(k) {
				t.Errorf("All yielded wrong value for %q", k)
			}
		}
		if !slices.Equal(keys, iterKeys) {
			t.Errorf("Keys %v and All %v disagree", keys, iterKeys)
		}
		for i, k := range keys {
			if vals[i] != h.Get(k) {
				t.Errorf("Values[%d] doesn't belong to key %q", i, k)
			}
		}
		if !slices.Equal(keys, h.Keys()) {
			t.Error("Keys order changed without modification")
		}
	}

	check()
	h.Set("a", 100) // overwriting keeps the order
	check()
	h.Set("f", 6)
	h.Delete("b")
	check()
	if h.Len() != 6 || h.Exists("b") {
		t.Error("add/delete bookkeeping is off")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := `{"big":9007199254740993,"f":1.5,"n":null,"ok":true,"s":"x","xs":[1,"two",[3]]}`
	s, err := FromJSON([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if s.Path("big").Kind() != Int || s.Path("big").Int() != 9007199254740993 {
		t.Error("large integer lost precision")
	}
	if s.Path("f").Kind() != Float {
		t.Error("1.5 should be a float")
	}
	if s.Path("n").Defined() || !s.HV().Exists("n") {
		t.Error("null should be an existing undef")
	}
	if got := mustJSON(t, s); got != in {
		t.Errorf("round trip:\n got %s\nwant %s", got, in)
	}
	if _, err := FromJSON([]byte(`{} {}`)); err == nil {
		t.Error("expected error for trailing data")
	}
}

func TestNilContainerWritesPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic pushing onto a non-array")
		}
	}()
	Scalar("str").AV().Push(1)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
