package scalar

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
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
		if got := Value(tt.in).Str(); got != tt.want {
			t.Errorf("Value(%#v).Str() = %q, want %q", tt.in, got, tt.want)
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
		got, ok := Value(tt.in).NumOK()
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Value(%#v).NumOK() = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
	if f, ok := Value("nan").NumOK(); !math.IsNaN(f) || !ok {
		t.Errorf(`Value("nan").NumOK() = (%v, %v), want (NaN, true)`, f, ok)
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
		got, ok := Value(tt.in).IntOK()
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Value(%#v).IntOK() = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
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
		if got := Value(tt.in).Bool(); got != tt.want {
			t.Errorf("Value(%#v).Bool() = %v, want %v", tt.in, got, tt.want)
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
	if got.Length() != 3 || got.Get(0).Defined() || got.Get(2).HV().Get("z").Int() != 1 {
		t.Fatalf("deep autoviv produced %s", mustJSON(t, h))
	}

	// Writing through a detached read attaches it.
	s := h.Get("late")
	s.Set("now")
	if h.Get("late").Str() != "now" {
		t.Fatal("Set on a missing key didn't store it")
	}
}

func TestGetRoute(t *testing.T) {
	resp, err := FromJSON([]byte(`{"content":[{"type":"text","text":"hi"}],"usage":{"output_tokens":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Get("content", 0, "text").Str(); got != "hi" {
		t.Errorf("text = %q", got)
	}
	if got := resp.Get("content", -1, "type").Str(); got != "text" {
		t.Errorf("type = %q", got)
	}
	if got := resp.Get("usage", "output_tokens").Int(); got != 12 {
		t.Errorf("output_tokens = %d", got)
	}
	// A string can't be dereferenced: reads yield undef rather than panicking.
	if resp.Get("content", 0, "text", "nope").Defined() {
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
	if a.Get(-10).Defined() || a.Get(99).Defined() || a.Length() != 3 {
		t.Error("out-of-range reads should be undef and not extend the array")
	}
	a.Set(5, "x")
	if a.Length() != 6 || a.Get(4).Defined() {
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
	ref := Value(a)
	Value(ref).AV().Push(3)
	if a.Length() != 3 {
		t.Error("copied reference should share the array")
	}
}

func TestHashOrder(t *testing.T) {
	h := Hash("a", 1, "b", 2, "c", 3, "d", 4, "e", 5, "orphan")
	if h.Length() != 6 || h.Get("orphan").Defined() || !h.Exists("orphan") {
		t.Fatal("odd-length Hash should map the last key to undef")
	}

	check := func() {
		t.Helper()
		keys, vals := h.KeysSlice(), h.ValuesSlice()
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
		if !slices.Equal(keys, h.KeysSlice()) {
			t.Error("Keys order changed without modification")
		}
	}

	check()
	h.Set("a", 100) // overwriting keeps the order
	check()
	h.Set("f", 6)
	h.Delete("b")
	check()
	if h.Length() != 6 || h.Exists("b") {
		t.Error("add/delete bookkeeping is off")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := `{"big":9007199254740993,"f":1.5,"n":null,"ok":true,"s":"x","xs":[1,"two",[3]]}`
	s, err := FromJSON([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if s.Get("big").Kind() != Int || s.Get("big").Int() != 9007199254740993 {
		t.Error("large integer lost precision")
	}
	if s.Get("f").Kind() != Float {
		t.Error("1.5 should be a float")
	}
	if s.Get("n").Defined() || !s.HV().Exists("n") {
		t.Error("null should be an existing undef")
	}
	if got := mustJSON(t, s); got != in {
		t.Errorf("round trip:\n got %s\nwant %s", got, in)
	}
	if _, err := FromJSON([]byte(`{} {}`)); err == nil {
		t.Error("expected error for trailing data")
	}
}

func TestLengthAndScalar(t *testing.T) {
	tests := []struct {
		in   any
		want int
	}{
		{nil, 0},
		{"", 0},
		{"hello", 5},
		{"café", 4},
		{"→•", 2},
		{12345, 5},
		{-1.5, 4},
		{true, 1},
		{false, 0},
		{"0E0", 3},
	}
	for _, tt := range tests {
		if got := Value(tt.in).Length(); got != tt.want {
			t.Errorf("length(%#v) = %d, want %d", tt.in, got, tt.want)
		}
	}
	var nilSV *SV
	if nilSV.Length() != 0 {
		t.Error("nil *SV should have length 0")
	}

	a := Array(1, 2, 3)
	if a.Scalar() != 3 || a.Scalar() != a.Length() || Array().Scalar() != 0 {
		t.Error("scalar(@a) should be the element count")
	}
	var nilAV *AV
	if nilAV.Scalar() != 0 {
		t.Error("nil *AV should have scalar 0")
	}

	h := Hash("a", 1, "b", 2)
	if h.Scalar() != 2 || h.Scalar() != h.Length() || Hash().Scalar() != 0 {
		t.Error("scalar(%h) should be the key count")
	}
	var nilHV *HV
	if nilHV.Scalar() != 0 {
		t.Error("nil *HV should have scalar 0")
	}
}

func TestKeysValuesArrays(t *testing.T) {
	h := Hash("ann", 72, "bob", 95, "cat", 81)
	keys, vals := h.Keys(), h.Values()
	if keys.Length() != 3 || vals.Length() != 3 {
		t.Fatalf("lengths %d, %d", keys.Length(), vals.Length())
	}
	for i, k := range h.KeysSlice() {
		if keys.Get(i).Str() != k || vals.Get(i) != h.Get(k) {
			t.Errorf("index %d: Keys/Values disagree with KeysSlice", i)
		}
	}

	byScore := h.Keys().Sort(func(x, y *SV) int {
		return NumCmp(h.Get(y.Str()), h.Get(x.Str()))
	})
	if got := byScore.Join(","); got != "bob,cat,ann" {
		t.Errorf("sorted keys = %s", got)
	}

	// keys are copies: changing them doesn't rename anything.
	keys.Set(0, "changed")
	if h.Exists("changed") || h.Length() != 3 {
		t.Error("changing Keys shouldn't change the hash")
	}

	// values are aliases, as in Perl: $_ *= 2 for values %h
	for _, v := range h.Values().All() {
		v.Set(v.Mul(2))
	}
	if h.Get("ann").Int() != 144 || h.Get("bob").Int() != 190 {
		t.Errorf("changing Values should change the hash: %s", mustJSON(t, h))
	}
	// ...but reshaping the array doesn't touch the hash.
	vals = h.Values()
	vals.Pop()
	vals.Push(1)
	if h.Length() != 3 {
		t.Error("Pop/Push on Values shouldn't change the hash")
	}

	var nilHV *HV
	if nilHV.Keys().Length() != 0 || nilHV.Values().Length() != 0 || nilHV.KeysSlice() != nil {
		t.Error("nil *HV should give empty results")
	}
}

func TestCopy(t *testing.T) {
	h := Hash("k", 5)
	x := h.Get("k").Copy()
	x.Set(9)
	if h.Get("k").Int() != 5 || x.Int() != 9 {
		t.Error("changing a copy shouldn't change the hash")
	}

	a := Array(1, 2)
	for _, v := range a.All() {
		v.Copy().Inc()
	}
	if a.Join(",") != "1,2" {
		t.Error("Copy inside All shouldn't change the array")
	}

	// Copying a missing element gives a plain undef that can be written,
	// without creating the element.
	m := h.Get("missing").Copy()
	m.Set(1)
	if h.Exists("missing") {
		t.Error("Copy of a missing element shouldn't attach to the hash")
	}
	c := a.Get(-10).Copy()
	c.Set(1) // would panic on the original
	if c.Int() != 1 {
		t.Error("Copy of a non-creatable element should be writable")
	}

	// Shallow, as in Perl: a copied reference shares the array.
	ref := Value(a)
	ref.Copy().AV().Push(3)
	if a.Length() != 3 {
		t.Error("a copied reference should point at the same array")
	}

	var nilSV *SV
	if nilSV.Copy().Defined() {
		t.Error("Copy of a nil *SV should be undef")
	}
}

func TestConversions(t *testing.T) {
	n := 7
	var nilPtr *int
	var nilSlice []string
	var nilMap map[string]int
	tests := []struct {
		name string
		in   any
		kind Kind
		json string
	}{
		{"nil", nil, Undef, "null"},
		{"int8", int8(-8), Int, "-8"},
		{"int16", int16(16), Int, "16"},
		{"int32", int32(32), Int, "32"},
		{"uint", uint(1), Int, "1"},
		{"uint8", uint8(8), Int, "8"},
		{"uint16", uint16(16), Int, "16"},
		{"uint32", uint32(32), Int, "32"},
		{"uint64 fits", uint64(64), Int, "64"},
		{"uint64 too big", uint64(math.MaxUint64), Float, "18446744073709552000"},
		{"float32", float32(1.5), Float, "1.5"},
		{"SV value", *Value("v"), String, `"v"`},
		{"pointer", &n, Int, "7"},
		{"nil pointer", nilPtr, Undef, "null"},
		{"[]string", []string{"a", "b"}, ArrayRef, `["a","b"]`},
		{"array", [2]int{1, 2}, ArrayRef, "[1,2]"},
		{"nil slice", nilSlice, Undef, "null"},
		{"map[string]int", map[string]int{"a": 1}, HashRef, `{"a":1}`},
		{"nil map", nilMap, Undef, "null"},
		{"map with int keys", map[int]string{1: "x"}, HashRef, `{"1":"x"}`},
		{"struct", struct{ A int }{1}, HashRef, `{"A":1}`}, // as encoding/json writes it; see TestValueStruct
		{"nested", map[string]any{"xs": []any{1, map[string]any{"y": true}}}, HashRef, `{"xs":[1,{"y":true}]}`},
	}
	for _, tt := range tests {
		s := Value(tt.in)
		if s.Kind() != tt.kind {
			t.Errorf("%s: kind %d, want %d", tt.name, s.Kind(), tt.kind)
		}
		if got := mustJSON(t, s); got != tt.json {
			t.Errorf("%s: JSON %s, want %s", tt.name, got, tt.json)
		}
	}
}

func TestKindRefAndPrinting(t *testing.T) {
	tests := []struct {
		in   any
		kind Kind
		ref  string
	}{
		{nil, Undef, ""},
		{true, Bool, ""},
		{1, Int, ""},
		{1.5, Float, ""},
		{"s", String, ""},
		{Array(), ArrayRef, "ARRAY"},
		{Hash(), HashRef, "HASH"},
	}
	for _, tt := range tests {
		s := Value(tt.in)
		if s.Kind() != tt.kind || s.Ref() != tt.ref {
			t.Errorf("%#v: Kind %d Ref %q, want %d %q", tt.in, s.Kind(), s.Ref(), tt.kind, tt.ref)
		}
	}

	a := Array()
	ref := Value(a)
	if got, want := ref.String(), fmt.Sprintf("ARRAY(%p)", a); got != want {
		t.Errorf("array ref prints as %q, want %q", got, want)
	}
	if !strings.HasPrefix(Value(Hash()).Str(), "HASH(0x") {
		t.Error("hash ref should print as HASH(0x...)")
	}
	if ref.Num() == 0 || ref.Int() == 0 || !ref.Bool() {
		t.Error("a reference is a non-zero address and true, as in Perl")
	}
	if Value("2.5").Num() != 2.5 || Value(math.Inf(-1)).Str() != "-Inf" {
		t.Error("Num/Str basics")
	}
}

func TestJSONEdges(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{Array(), "[]"},
		{Hash(), "{}"},
		{&AV{}, "[]"},
		{&HV{}, "{}"},
		{(*AV)(nil), "null"},
		{(*HV)(nil), "null"},
		{(*SV)(nil), "null"},
		{struct{ S *SV }{Value("x")}, `{"S":"x"}`},
	}
	for _, tt := range tests {
		if got := mustJSON(t, tt.v); got != tt.want {
			t.Errorf("%#v encodes as %s, want %s", tt.v, got, tt.want)
		}
	}

	for _, bad := range []string{`{} }`, `{} ]`, `[] x`, `1 2`, `{`, ``} {
		if _, err := FromJSON([]byte(bad)); err == nil {
			t.Errorf("FromJSON(%q) should fail", bad)
		}
	}
	if s, err := FromJSON([]byte("  {\"a\":1}\n\t ")); err != nil || s.Get("a").Int() != 1 {
		t.Errorf("surrounding whitespace should be fine: %v", err)
	}

	var holder struct{ S *SV }
	if err := json.Unmarshal([]byte(`{"S":{"x":[1,2]}}`), &holder); err != nil || holder.S.Get("x", 1).Int() != 2 {
		t.Errorf("SV as a struct field: %v", err)
	}
}

func TestIteration(t *testing.T) {
	a := Array(1, 2, 3)
	var seen []int64
	for _, s := range a.All() {
		seen = append(seen, s.Int())
		if len(seen) == 2 {
			break
		}
	}
	if !slices.Equal(seen, []int64{1, 2}) {
		t.Errorf("breaking out of AV.All: %v", seen)
	}
	items := a.Items()
	items[0] = Value(99)
	if a.Get(0).Int() != 1 || len(items) != 3 {
		t.Error("Items should return a new slice")
	}
	items[1].Set(20)
	if a.Get(1).Int() != 20 {
		t.Error("Items' elements are the stored values")
	}

	h := Hash("a", 1, "b", 2, "c", 3, "d", 4)
	visited := 0
	for k := range h.All() {
		visited++
		// Delete everything else and add new keys mid-iteration.
		for _, other := range []string{"a", "b", "c", "d"} {
			if other != k {
				h.Delete(other)
			}
		}
		h.Set("new-"+k, 1)
	}
	if visited != 1 {
		t.Errorf("keys deleted during All should be skipped, and added ones not visited; visited %d", visited)
	}

	h = Hash("a", 1, "b", 2)
	visited = 0
	for range h.All() {
		visited++
		break
	}
	if visited != 1 {
		t.Error("breaking out of HV.All")
	}
}

func TestSubstr(t *testing.T) {
	const cat = "The black cat climbed the green tree"
	undef := "<undef>"
	tests := []struct {
		name   string
		s      any
		offset int
		length []int
		want   string
	}{
		// The examples from perlfunc's substr.
		{"perlfunc color", cat, 4, []int{5}, "black"},
		{"perlfunc middle", cat, 4, []int{-11}, "black cat climbed the"},
		{"perlfunc end", cat, 14, nil, "climbed the green tree"},
		{"perlfunc tail", cat, -4, nil, "tree"},
		{"perlfunc z", cat, -4, []int{2}, "tr"},
		{"perlfunc null: start at the end", "freddy", 6, []int{2}, ""},
		{"perlfunc oops: start past the end", "freddy", 7, nil, undef},

		// Partly outside: the part that's inside.
		{"length past the end", "abc", 1, []int{10}, "bc"},
		{"offset before the start", "abc", -5, nil, "abc"},
		{"offset before the start, length reaching in", "abc", -5, []int{3}, "a"},
		{"length leaves nothing", "abc", 2, []int{-5}, ""},
		{"zero length", "abc", 1, []int{0}, ""},

		// Entirely outside: undef.
		{"entirely before the start", "abc", -5, []int{1}, undef},

		// Characters, not bytes, and string context.
		{"unicode", "naïve café", 6, []int{4}, "café"},
		{"number in string context", 12345, 1, []int{3}, "234"},
		{"undef is empty", nil, 0, nil, ""},
		{"empty string, negative offset", "", -1, nil, ""},
	}
	for _, tt := range tests {
		got := Value(tt.s).Substr(tt.offset, tt.length...)
		gotStr := got.Str()
		if !got.Defined() {
			gotStr = undef
		}
		if gotStr != tt.want {
			t.Errorf("%s: substr(%q, %d, %v) = %q, want %q", tt.name, Value(tt.s).Str(), tt.offset, tt.length, gotStr, tt.want)
		}
	}
}

func TestSubstrReplace(t *testing.T) {
	// perlfunc: my $z = substr $s, 14, 7, "jumped from";
	s := Value("The black cat climbed the green tree")
	if got := s.SubstrReplace(14, 7, "jumped from").Str(); got != "climbed" {
		t.Errorf("returned %q, want the replaced part", got)
	}
	if want := "The black cat jumped from the green tree"; s.Str() != want {
		t.Errorf("s = %q, want %q", s, want)
	}

	// perlfunc: substr($name, 4) = 'dy'; and substr($name, 7) = 'gap' dies.
	name := Value("fred")
	name.SubstrReplace(4, 0, "dy")
	if name.Str() != "freddy" {
		t.Errorf("appending at the end: %q", name)
	}
	expectPanic(t, "start past the end", "substr outside of string", func() { name.SubstrReplace(7, 0, "gap") })
	expectPanic(t, "entirely before the start", "substr outside of string", func() { name.SubstrReplace(-10, 1, "x") })
	if name.Str() != "freddy" {
		t.Errorf("a failed replace changed s to %q", name)
	}

	tests := []struct {
		s              string
		offset, length int
		repl           any
		want, removed  string
	}{
		{"abcdef", 0, 1, "X", "Xbcdef", "a"},
		{"abcdef", -2, 2, "YZ", "abcdYZ", "ef"},
		{"abcdef", 1, -1, "-", "a-f", "bcde"},
		{"abcdef", 2, 0, "+++", "ab+++cdef", ""},
		{"abcdef", 1, 2, "", "adef", "bc"},
		{"naïve", 2, 1, "i", "naive", "ï"},
		{"price: 0", -1, 1, 42, "price: 42", "0"},
	}
	for _, tt := range tests {
		s := Value(tt.s)
		removed := s.SubstrReplace(tt.offset, tt.length, tt.repl)
		if s.Str() != tt.want || removed.Str() != tt.removed {
			t.Errorf("substr(%q, %d, %d, %v): s = %q, returned %q; want %q, %q",
				tt.s, tt.offset, tt.length, tt.repl, s, removed, tt.want, tt.removed)
		}
	}

	// Writing through a missing hash element creates it, as in Perl.
	h := Hash()
	h.Get("k").SubstrReplace(0, 0, "new")
	if h.Get("k").Str() != "new" {
		t.Error("SubstrReplace should write through to a missing element")
	}

	// Substr itself never changes s.
	orig := Value("abc")
	orig.Substr(0, 1)
	if orig.Str() != "abc" {
		t.Error("Substr changed s")
	}
}

func TestOr(t *testing.T) {
	tests := []struct {
		name string
		s    *SV
		def  any
		want string
	}{
		{"undef takes the default", Value(nil), "n/a", "n/a"},
		{"defined keeps its value", Value("x"), "n/a", "x"},
		{"0 is defined", Value(0), 42, "0"},
		{`"" is defined`, Value(""), "n/a", ""},
		{"nil *SV", nil, 7, "7"},
		{"scalar default", Value(nil), Value("d"), "d"},
	}
	for _, tt := range tests {
		if got := tt.s.Or(tt.def).Str(); got != tt.want {
			t.Errorf("%s: Or = %q, want %q", tt.name, got, tt.want)
		}
	}

	// Reading a missing path with a default.
	resp := mustFromJSON(t, `{"usage": {"input_tokens": 10}}`)
	if resp.Get("usage", "output_tokens").Or(0).Int() != 0 || resp.Get("usage", "input_tokens").Or(0).Int() != 10 {
		t.Error("Or on a path")
	}

	// Or returns a new scalar, so changing it can't change the original.
	h := Hash("k", "v")
	h.Get("k").Or("x").Set("changed")
	def := Value("d")
	Value(nil).Or(def).Set("changed")
	if h.Get("k").Str() != "v" || def.Str() != "d" {
		t.Error("Or should return a new scalar")
	}
}

func TestOrSet(t *testing.T) {
	h := Hash("present", 0)
	h.Get("present").OrSet(99)
	h.Get("missing").OrSet("Unknown")
	if h.Get("present").Int() != 0 {
		t.Error("OrSet shouldn't replace a defined value, even 0")
	}
	if !h.Exists("missing") || h.Get("missing").Str() != "Unknown" {
		t.Error("OrSet should create a missing element")
	}
	// It returns s, for chaining: $h{n} //= 0; $h{n}++
	if h.Get("n").OrSet(0).Inc().Int() != 1 {
		t.Error("OrSet should return s")
	}
	expectPanic(t, "write before the start", "non-creatable", func() { Array().Get(-1).OrSet(1) })
}

func TestGetOK(t *testing.T) {
	doc := mustFromJSON(t, `{"usage": {"output_tokens": 12, "stop": null}, "content": [{"text": "hi"}], "s": "str"}`)
	tests := []struct {
		name    string
		keys    []any
		want    string
		defined bool
		ok      bool
	}{
		{"present", []any{"usage", "output_tokens"}, "12", true, true},
		{"present but null", []any{"usage", "stop"}, "", false, true},
		{"missing key", []any{"usage", "nope"}, "", false, false},
		{"missing parent", []any{"nope", "deeper"}, "", false, false},
		{"array index", []any{"content", 0, "text"}, "hi", true, true},
		{"negative index", []any{"content", -1, "text"}, "hi", true, true},
		{"index out of range", []any{"content", 5}, "", false, false},
		{"index before the start", []any{"content", -5}, "", false, false},
		{"through a string", []any{"s", "x"}, "", false, false},
		{"key on an array", []any{"content", "x"}, "", false, false},
		{"index on a hash", []any{"usage", 0}, "", false, false},
		{"no keys", nil, "HASH", true, true},
	}
	for _, tt := range tests {
		v, ok := doc.GetOK(tt.keys...)
		got := v.Str()
		if v.Ref() != "" {
			got = v.Ref()
		}
		if got != tt.want || v.Defined() != tt.defined || ok != tt.ok {
			t.Errorf("%s: PathOK = (%q, defined %v, %v), want (%q, %v, %v)", tt.name, got, v.Defined(), ok, tt.want, tt.defined, tt.ok)
		}
	}

	// It never creates anything, unlike Path.
	h := Value(Hash())
	h.GetOK("a", "b", 0)
	if h.HV().Length() != 0 {
		t.Error("PathOK shouldn't autovivify")
	}

	// Holes don't exist.
	a := Array()
	a.Set(2, "c")
	if _, ok := Value(a).GetOK(0); ok {
		t.Error("a hole doesn't exist")
	}

	// The value is the stored one.
	v, _ := doc.GetOK("usage", "output_tokens")
	v.Inc()
	if doc.Get("usage", "output_tokens").Int() != 13 {
		t.Error("PathOK should return the stored value")
	}

	var nilSV *SV
	if v, ok := nilSV.GetOK(); ok || v == nil {
		t.Error("nil *SV: not ok, but a usable undef")
	}
	expectPanic(t, "bad key type", "GetOK key must be a string or an int", func() { doc.GetOK(1.5) })
}

func TestHashFlattening(t *testing.T) {
	defaults := Hash("network", "ci", "status", "automatic")

	// %h = (%defaults, network => "us"): later keys win.
	h := Hash(defaults, "network", "us")
	if h.Get("network").Str() != "us" || h.Get("status").Str() != "automatic" || h.Length() != 2 {
		t.Errorf("merge = %s", mustJSON(t, h))
	}
	// Copying a hash: %copy = %h
	cp := Hash(h)
	cp.Set("status", "reviewed")
	if h.Get("status").Str() != "automatic" {
		t.Error("Hash(h) should copy")
	}
	// A hash from an array of pairs.
	if got := mustJSON(t, Hash(Array("a", 1, "b", 2))); got != `{"a":1,"b":2}` {
		t.Errorf("Hash(array of pairs) = %s", got)
	}
	// Values that are references stay references.
	nested := Hash("list", Value(Array(1, 2)))
	if nested.Get("list").AV().Length() != 2 {
		t.Error("a reference value should stay a reference")
	}
	// Odd lists still give the last key undef.
	if odd := Hash(Array("k")); !odd.Exists("k") || odd.Get("k").Defined() {
		t.Error("odd list")
	}
}

func TestHashFromMaps(t *testing.T) {
	m1 := map[string]any{"a": 1, "b": 2}
	tests := []struct {
		name string
		got  *HV
		want string
	}{
		{"Go map", Hash(m1), `{"a":1,"b":2}`},
		{"list of maps, later keys win", Hash([]map[string]any{m1, {"b": 99}}), `{"a":1,"b":99}`},
		{"several maps", Hash(m1, map[string]int{"c": 3}), `{"a":1,"b":2,"c":3}`},
		{"map plus pairs", Hash(m1, "k", "v"), `{"a":1,"b":2,"k":"v"}`},
		{"array of hashes", Hash(Array(Hash("x", 1), Hash("y", 2))), `{"x":1,"y":2}`},
		{"int keys", Hash(map[int]string{1: "one", 2: "two"}), `{"1":"one","2":"two"}`},
		{"float and bool keys", Hash(map[float64]int{1.5: 1}, map[bool]int{true: 2}), `{"1":2,"1.5":1}`},
		{"nested values stay nested", Hash(map[string]any{"list": []any{1, 2}}), `{"list":[1,2]}`},
		{"nil map adds nothing", Hash(map[string]int(nil), "k", 1), `{"k":1}`},
	}
	for _, tt := range tests {
		if got := mustJSON(t, tt.got); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
	if got := mustJSON(t, Value(map[int]string{7: "x"})); got != `{"7":"x"}` {
		t.Errorf("Value(map[int]string) = %s", got)
	}
	if Value(map[int]string(nil)).Defined() {
		t.Error("a nil map should be undef")
	}
}

// exists says the key or element is there; defined also needs a non-undef
// value.
func TestExistsVersusDefined(t *testing.T) {
	h := Hash("name", "Ada", "middle", nil, "zero", 0, "empty", "")
	tests := []struct {
		key             string
		exists, defined bool
	}{
		{"name", true, true},
		{"middle", true, false}, // present, but undef
		{"zero", true, true},    // false values are still defined
		{"empty", true, true},
		{"missing", false, false},
	}
	for _, tt := range tests {
		if h.Exists(tt.key) != tt.exists || h.Defined(tt.key) != tt.defined {
			t.Errorf("%q: exists %v defined %v, want %v %v",
				tt.key, h.Exists(tt.key), h.Defined(tt.key), tt.exists, tt.defined)
		}
	}
	h.Defined("missing")
	if h.Exists("missing") {
		t.Error("Defined shouldn't create the key")
	}
	if (*HV)(nil).Defined("k") {
		t.Error("nil hash")
	}

	a := Array("a", nil)
	a.Set(3, 0) // hole at 2
	for i, want := range map[int]bool{0: true, 1: false, 2: false, 3: true, 4: false, -1: true, -4: true, -9: false} {
		if a.Defined(i) != want {
			t.Errorf("defined $a[%d] = %v, want %v", i, a.Defined(i), want)
		}
	}
	if !a.Exists(1) || a.Exists(2) {
		t.Error("undef element exists; hole doesn't")
	}
	if (*AV)(nil).Defined(0) {
		t.Error("nil array")
	}
}

// A reference that leads back to itself can't be JSON: encoding it is an
// error, not a crash. The same hash appearing twice isn't a cycle.
func TestMarshalJSONCycles(t *testing.T) {
	h := Hash("a", 1)
	h.Set("self", Value(h))
	for name, v := range map[string]any{"SV": Value(h), "HV": h, "AV": Array(Value(h))} {
		if _, err := json.Marshal(v); err == nil || !strings.Contains(err.Error(), "self-referencing HASH") {
			t.Errorf("%s: err = %v", name, err)
		}
	}

	shared := Hash("x", 1)
	twice := Hash("first", Value(shared), "second", Value(shared))
	got, err := json.Marshal(Value(twice))
	if err != nil || string(got) != `{"first":{"x":1},"second":{"x":1}}` {
		t.Errorf("shared hash: %s, %v", got, err)
	}

	expectPanic(t, "Any on a cycle", "self-referencing HASH", func() { Value(h).Any() })
}

// The format verbs treat a scalar as Perl's sprintf does.
func TestFormat(t *testing.T) {
	for _, tt := range []struct {
		format string
		arg    any
		want   string
	}{
		{"%d", "42 apples", "42"},
		{"%d", 3.7, "3"},
		{"%05.2f", "3.14159", "03.14"},
		{"%x", 255, "ff"},
		{"%x", "ff", "0"}, // numeric context, as in Perl
		{"%c", 65, "A"},
		{"%e", "1500", "1.500000e+03"},
		{"%s", 12.5, "12.5"},
		{"%5s|", "ab", "   ab|"},
		{"%q", "a\"b", `"a\"b"`},
		{"%v", nil, ""},
		{"%t", "0", "false"},
		{"%t", "0.0", "true"},
	} {
		if got := fmt.Sprintf(tt.format, Value(tt.arg)); got != tt.want {
			t.Errorf("Sprintf(%q, %v) = %q, want %q", tt.format, tt.arg, got, tt.want)
		}
	}
	var undef *SV
	if got := fmt.Sprintf("%d|%s", undef, undef); got != "0|" {
		t.Errorf("nil *SV = %q", got)
	}
}

// Values with keys is Perl's hash slice @h{...}: the hash's own values, as
// Perl aliases them (checked against perl -E).
func TestHashValues(t *testing.T) {
	h := Hash("a", 1, "b", 2)
	got := h.Values("b", "missing", Array("a"))
	if got.Join(",") != "2,,1" || got.Get(1).Defined() {
		t.Errorf("@h{qw(b missing a)} = %q", got.StrSlice())
	}
	if h.Exists("missing") {
		t.Error("reading a slice shouldn't create missing keys")
	}
	for _, v := range h.Values("a", "b").All() { // $_ *= 10 for @h{qw(a b)}
		v.MulInPlace(10)
	}
	if h.Get("a").Int() != 10 || h.Get("b").Int() != 20 {
		t.Errorf("writing through a slice: a=%v b=%v", h.Get("a"), h.Get("b"))
	}
	h.Values("new").Get(0).Set(3) // writing to a missing key creates it
	if h.Get("new").Int() != 3 {
		t.Error("writing through a slice should create a missing key")
	}
	var nilHash *HV
	if nilHash.Values("a").Length() != 1 || nilHash.Values().Length() != 0 {
		t.Error("Values of a nil hash should give undef for each key, or nothing")
	}
}

func TestPairsMap(t *testing.T) {
	h := Hash("a", 1, "b", 2)
	m := h.PairsMap()
	if len(m) != 2 || m["a"].Int() != 1 {
		t.Errorf("PairsMap = %v", m)
	}
	m["a"].Set(10) // the hash's own values
	delete(m, "b") // but its own map
	if h.Get("a").Int() != 10 || !h.Exists("b") {
		t.Error("PairsMap should share values but not keys")
	}
	var nilHash *HV
	if m := nilHash.PairsMap(); m == nil || len(m) != 0 {
		t.Errorf("PairsMap of a nil hash = %#v, want an empty map", m)
	}
}

// Pairs behave as Perl 5.42's key/value slices and %h in list context do
// (checked against perl -E).
func TestPairs(t *testing.T) {
	undefStr := func(a *AV) string {
		parts := make([]string, a.Length())
		for i, s := range a.All() {
			parts[i] = s.Or("undef").Str()
		}
		return strings.Join(parts, ",")
	}

	// my @l = %h2{qw(k1 k2 k3)}   gives   k1,1,k2,undef,k3,undef
	h2 := Hash("k1", 1, "k2", nil)
	if got := undefStr(h2.Pairs("k1", "k2", "k3")); got != "k1,1,k2,undef,k3,undef" {
		t.Errorf("hash = %s", got)
	}
	// my %h1 = %h2{qw(k1 k3)}: k3 exists in h1 but isn't created in h2, and
	// h1 is a hash of its own.
	h1 := Hash(h2.Pairs("k1", "k3"))
	if !h1.Exists("k3") || h1.Length() != 2 || h2.Exists("k3") {
		t.Errorf("h1 has k3: %v (length %d); h2 has k3: %v", h1.Exists("k3"), h1.Length(), h2.Exists("k3"))
	}
	h1.Get("k1").Set(99)
	if h2.Get("k1").Int() != 1 {
		t.Error("a hash made from Pairs should have its own values")
	}
	// for (%h{k1}) { $_ .= "!" } changes the value, as the values are aliased.
	for _, s := range h2.Pairs("k1").All() {
		s.ConcatInPlace("!")
	}
	if h2.Get("k1").Str() != "1!" || !h2.Exists("k1") {
		t.Errorf("after writing through Pairs: k1=%v", h2.Get("k1"))
	}
	// Without keys, every pair: %h in list context.
	if got := Hash(Hash("a", 1, "b", 2).Pairs()); got.Length() != 2 || got.Get("b").Int() != 2 {
		t.Errorf("all pairs = %v", got.Pairs())
	}

	// my @p = %a[0, 5, -1]   gives   0,a,5,undef,-1,b, and @a stays at 2.
	a := Array("a", "b")
	if got := undefStr(a.Pairs(0, 5, -1)); got != "0,a,5,undef,-1,b" || a.Length() != 2 {
		t.Errorf("array = %s, length %d", got, a.Length())
	}
	if got := undefStr(a.Pairs()); got != "0,a,1,b" {
		t.Errorf("all array pairs = %s", got)
	}
	var nilHash *HV
	if got := undefStr(nilHash.Pairs("a")); got != "a,undef" {
		t.Errorf("nil hash = %s", got)
	}
}

// Each behaves as Perl's each does (checked against perl -E).
func TestHashEach(t *testing.T) {
	h := Hash("a", 1, "b", 2, "c", 3)
	pass := func() []string {
		var ks []string
		for k, _, ok := h.Each(); ok; k, _, ok = h.Each() {
			ks = append(ks, k)
		}
		return ks
	}
	first := pass()
	if len(first) != 3 {
		t.Fatalf("a full pass gave %q", first)
	}
	if !slices.Equal(pass(), first) {
		t.Error("after the end, Each should start again, in the same order")
	}

	// Stopping early, then carrying on where it stopped.
	k1, _, _ := h.Each()
	k2, _, _ := h.Each()
	if k1 != first[0] || k2 != first[1] {
		t.Errorf("early pairs %s, %s", k1, k2)
	}
	// Keys, Values, Pairs, KeysSlice, ValuesSlice and PairsMap reset it; a
	// slice of named keys doesn't, and nor does All.
	for name, reset := range map[string]func(){
		"Keys": func() { h.Keys() }, "Values": func() { h.Values() }, "Pairs": func() { h.Pairs() },
		"KeysSlice": func() { h.KeysSlice() }, "ValuesSlice": func() { h.ValuesSlice() },
		"PairsMap": func() { h.PairsMap() },
	} {
		h.Each()
		reset()
		if k, _, _ := h.Each(); k != first[0] {
			t.Errorf("%s should reset Each, but it carried on at %s", name, k)
		}
	}
	h.Keys() // back to the start
	h.Each()
	h.Values("a", "b")
	for range h.All() {
	}
	if k, _, _ := h.Each(); k != first[1] {
		t.Errorf("a named slice or All shouldn't reset Each, but the next key was %s", k)
	}

	// The values are the hash's own.
	h.Keys()
	_, v, _ := h.Each()
	v.Set(99)
	if h.Get(first[0]).Int() != 99 {
		t.Error("Each should give the hash's own values")
	}

	// Deleting a key along the way skips it; one added isn't returned until
	// the next pass.
	h.Keys()
	h.Each()
	h.Delete(first[1])
	h.Set("d", 4)
	if rest := pass(); !slices.Equal(rest, first[2:]) {
		t.Errorf("after deleting and adding mid-pass: %q, want %q", rest, first[2:])
	}
	if next := pass(); len(next) != 3 || !slices.Contains(next, "d") {
		t.Errorf("the next pass = %q, want 3 keys including d", next)
	}

	var nilHash *HV
	if _, _, ok := nilHash.Each(); ok {
		t.Error("Each of a nil hash should give nothing")
	}
}

func TestArrayEach(t *testing.T) {
	a := Array("x", "y")
	a.Set(3, "z") // a hole at 2
	var got []string
	for i, v, ok := a.Each(); ok; i, v, ok = a.Each() {
		got = append(got, fmt.Sprintf("%d=%s", i, v.Or("undef")))
	}
	if want := []string{"0=x", "1=y", "2=undef", "3=z"}; !slices.Equal(got, want) {
		t.Errorf("each @a = %q, want %q", got, want)
	}
	if i, _, _ := a.Each(); i != 0 {
		t.Error("after the end, Each should start again")
	}
	a.Each()
	a.Values() // as values @a does
	if i, _, _ := a.Each(); i != 0 {
		t.Error("Values should reset Each")
	}
	a.Each()
	a.Pairs()
	if i, _, _ := a.Each(); i != 0 {
		t.Error("Pairs should reset Each")
	}
	var nilArray *AV
	if _, _, ok := nilArray.Each(); ok {
		t.Error("Each of a nil array should give nothing")
	}
}

func TestHashPush(t *testing.T) {
	h := Hash("a", 1)
	held := h.Get("a")
	if got := h.Push("a", 2, "b", 3); got != h {
		t.Error("Push should return the hash, for chaining")
	}
	if h.Get("a").Int() != 2 || h.Get("b").Int() != 3 || held.Int() != 2 {
		t.Errorf("after Push: a=%v b=%v held=%v; an existing key's scalar should be updated in place",
			h.Get("a"), h.Get("b"), held)
	}

	// Another hash, a Go map, Pairs, later keys winning, and a trailing key.
	h.Push(Hash("c", 4), map[string]int{"d": 5}, Hash("x", 6, "y", 7).Pairs("y"), "e", 1, "e", 8, "f")
	want := map[string]string{"a": "2", "b": "3", "c": "4", "d": "5", "y": "7", "e": "8", "f": ""}
	if h.Length() != len(want) || h.Defined("f") || !h.Exists("f") {
		t.Errorf("after merging: %v", h.Pairs())
	}
	for k, v := range want {
		if h.Get(k).Str() != v {
			t.Errorf("%s = %q, want %q", k, h.Get(k).Str(), v)
		}
	}

	// Values are copied in.
	src := Value("orig")
	h.Push("g", src)
	src.Set("changed")
	if h.Get("g").Str() != "orig" {
		t.Error("Push should copy values in")
	}
	// A reference stays a reference to the same data, as in Perl.
	list := Array(1)
	h.Push("list", Value(list))
	list.Push(2)
	if h.Get("list").AV().Length() != 2 {
		t.Error("a pushed reference should point at the same array")
	}

	if got := h.Push(); got != h {
		t.Error("Push with nothing should return h unchanged")
	}
	var nilHash *HV
	expectPanic(t, "Push to a nil hash", "Not a HASH reference", func() { nilHash.Push("a", 1) })
}
