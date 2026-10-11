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
	ref := Scalar(a)
	Scalar(ref).AV().Push(3)
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
		if got := Scalar(tt.in).Length(); got != tt.want {
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
	ref := Scalar(a)
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
		{"SV value", *Scalar("v"), String, `"v"`},
		{"pointer", &n, Int, "7"},
		{"nil pointer", nilPtr, Undef, "null"},
		{"[]string", []string{"a", "b"}, ArrayRef, `["a","b"]`},
		{"array", [2]int{1, 2}, ArrayRef, "[1,2]"},
		{"nil slice", nilSlice, Undef, "null"},
		{"map[string]int", map[string]int{"a": 1}, HashRef, `{"a":1}`},
		{"nil map", nilMap, Undef, "null"},
		{"map with int keys", map[int]string{1: "x"}, String, `"map[1:x]"`},
		{"struct", struct{ A int }{1}, String, `"{1}"`},
		{"nested", map[string]any{"xs": []any{1, map[string]any{"y": true}}}, HashRef, `{"xs":[1,{"y":true}]}`},
	}
	for _, tt := range tests {
		s := Scalar(tt.in)
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
		s := Scalar(tt.in)
		if s.Kind() != tt.kind || s.Ref() != tt.ref {
			t.Errorf("%#v: Kind %d Ref %q, want %d %q", tt.in, s.Kind(), s.Ref(), tt.kind, tt.ref)
		}
	}

	a := Array()
	ref := Scalar(a)
	if got, want := ref.String(), fmt.Sprintf("ARRAY(%p)", a); got != want {
		t.Errorf("array ref prints as %q, want %q", got, want)
	}
	if !strings.HasPrefix(Scalar(Hash()).Str(), "HASH(0x") {
		t.Error("hash ref should print as HASH(0x...)")
	}
	if ref.Num() == 0 || ref.Int() == 0 || !ref.Bool() {
		t.Error("a reference is a non-zero address and true, as in Perl")
	}
	if Scalar("2.5").Num() != 2.5 || Scalar(math.Inf(-1)).Str() != "-Inf" {
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
		{struct{ S *SV }{Scalar("x")}, `{"S":"x"}`},
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
	if s, err := FromJSON([]byte("  {\"a\":1}\n\t ")); err != nil || s.Path("a").Int() != 1 {
		t.Errorf("surrounding whitespace should be fine: %v", err)
	}

	var holder struct{ S *SV }
	if err := json.Unmarshal([]byte(`{"S":{"x":[1,2]}}`), &holder); err != nil || holder.S.Path("x", 1).Int() != 2 {
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
	items[0] = Scalar(99)
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
		got := Scalar(tt.s).Substr(tt.offset, tt.length...)
		gotStr := got.Str()
		if !got.Defined() {
			gotStr = undef
		}
		if gotStr != tt.want {
			t.Errorf("%s: substr(%q, %d, %v) = %q, want %q", tt.name, Scalar(tt.s).Str(), tt.offset, tt.length, gotStr, tt.want)
		}
	}
}

func TestSubstrReplace(t *testing.T) {
	// perlfunc: my $z = substr $s, 14, 7, "jumped from";
	s := Scalar("The black cat climbed the green tree")
	if got := s.SubstrReplace(14, 7, "jumped from").Str(); got != "climbed" {
		t.Errorf("returned %q, want the replaced part", got)
	}
	if want := "The black cat jumped from the green tree"; s.Str() != want {
		t.Errorf("s = %q, want %q", s, want)
	}

	// perlfunc: substr($name, 4) = 'dy'; and substr($name, 7) = 'gap' dies.
	name := Scalar("fred")
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
		s := Scalar(tt.s)
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
	orig := Scalar("abc")
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
		{"undef takes the default", Scalar(nil), "n/a", "n/a"},
		{"defined keeps its value", Scalar("x"), "n/a", "x"},
		{"0 is defined", Scalar(0), 42, "0"},
		{`"" is defined`, Scalar(""), "n/a", ""},
		{"nil *SV", nil, 7, "7"},
		{"scalar default", Scalar(nil), Scalar("d"), "d"},
	}
	for _, tt := range tests {
		if got := tt.s.Or(tt.def).Str(); got != tt.want {
			t.Errorf("%s: Or = %q, want %q", tt.name, got, tt.want)
		}
	}

	// Reading a missing path with a default.
	resp := mustFromJSON(t, `{"usage": {"input_tokens": 10}}`)
	if resp.Path("usage", "output_tokens").Or(0).Int() != 0 || resp.Path("usage", "input_tokens").Or(0).Int() != 10 {
		t.Error("Or on a path")
	}

	// Or returns a new scalar, so changing it can't change the original.
	h := Hash("k", "v")
	h.Get("k").Or("x").Set("changed")
	def := Scalar("d")
	Scalar(nil).Or(def).Set("changed")
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

func TestPathOK(t *testing.T) {
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
		v, ok := doc.PathOK(tt.keys...)
		got := v.Str()
		if v.Ref() != "" {
			got = v.Ref()
		}
		if got != tt.want || v.Defined() != tt.defined || ok != tt.ok {
			t.Errorf("%s: PathOK = (%q, defined %v, %v), want (%q, %v, %v)", tt.name, got, v.Defined(), ok, tt.want, tt.defined, tt.ok)
		}
	}

	// It never creates anything, unlike Path.
	h := Scalar(Hash())
	h.PathOK("a", "b", 0)
	if h.HV().Length() != 0 {
		t.Error("PathOK shouldn't autovivify")
	}

	// Holes don't exist.
	a := Array()
	a.Set(2, "c")
	if _, ok := Scalar(a).PathOK(0); ok {
		t.Error("a hole doesn't exist")
	}

	// The value is the stored one.
	v, _ := doc.PathOK("usage", "output_tokens")
	v.Inc()
	if doc.Path("usage", "output_tokens").Int() != 13 {
		t.Error("PathOK should return the stored value")
	}

	var nilSV *SV
	if v, ok := nilSV.PathOK(); ok || v == nil {
		t.Error("nil *SV: not ok, but a usable undef")
	}
	expectPanic(t, "bad key type", "Path key must be string or int", func() { doc.PathOK(1.5) })
}
