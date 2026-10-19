package scalar

import (
	"fmt"
	"slices"
	"testing"
)

// At, AtOK, DeleteAt and GetOK on a hash or array give what the same
// call on a scalar holding it gives, and write to the container itself.
func TestContainerPaths(t *testing.T) {
	newDoc := func() *SV {
		doc, _ := FromJSON([]byte(`{"a":{"b":[10,{"c":"x"}]},"list":[1,[2,3]],"0":"zero"}`))
		return doc
	}
	for _, path := range []string{"a.b[1].c", "a.b.0", "list[1][0]", "a.missing", "0", `["0"]`, "list[-1].1", ""} {
		doc := newDoc()
		want, wantOK := doc.AtOK(path)
		got, gotOK := doc.HV().AtOK(path)
		if d := got.Diff(want); d != "" || gotOK != wantOK || doc.HV().At(path).Diff(doc.At(path)) != "" {
			t.Errorf("HV.At(%q): %s, ok %v vs %v", path, d, gotOK, wantOK)
		}
	}
	for _, keys := range [][]any{{"a", "b", 1, "c"}, {"list", 1, 0}, {"a", "nope"}} {
		doc := newDoc()
		want, wantOK := doc.GetOK(keys...)
		got, gotOK := doc.HV().GetOK(keys...)
		if d := got.Diff(want); d != "" || gotOK != wantOK || doc.HV().Get(keys[0].(string)).Get(keys[1:]...).Diff(doc.Get(keys...)) != "" {
			t.Errorf("HV.Get(%v): %s", keys, d)
		}
	}
	arr := newDoc().At("list").AV()
	if arr.At("[1][1]").Int() != 3 || arr.Get(1).Get(0).Int() != 2 || arr.At("[0]").Int() != 1 {
		t.Errorf("AV paths: %v %v", arr.At("[1][1]"), arr.Get(1).Get(0))
	}
	if v, ok := arr.AtOK("[5]"); ok || v.Defined() {
		t.Error("AV.AtOK past the end")
	}
	if _, ok := arr.GetOK(1, 0); !ok {
		t.Error("AV.GetOK")
	}

	// Writing through creates in the container, as autovivification does.
	h := Hash()
	h.At("tally.right").Inc()
	h.At("tally.right").Inc()
	h.Get("tally").Get("wrong").Set(1)
	if h.Get("tally").HV().Get("right").Int() != 2 || h.At("tally.wrong").Int() != 1 {
		t.Errorf("writing through a hash: %s", h.Dump())
	}
	a := Array()
	a.At("[2].name").Set("x")
	if a.Length() != 3 || a.Get(2).HV().Get("name").Str() != "x" {
		t.Errorf("writing through an array: %s", a.Dump())
	}

	// DeleteAt removes from the container.
	doc := newDoc()
	if got := doc.HV().DeleteAt("a.b[1].c"); got.Str() != "x" || doc.At("a.b[1]").HV().Exists("c") {
		t.Errorf("HV.DeleteAt: %v", got)
	}
	list := newDoc().At("list").AV()
	// Deleting an element before the last leaves a hole, as delete $a[0]
	// does in Perl: the length stays, and the element no longer exists.
	if got := list.DeleteAt("[1][0]"); got.Int() != 2 || list.At("[1]").AV().Exists(0) || list.At("[1]").AV().Length() != 2 {
		t.Errorf("AV.DeleteAt: %v", got)
	}

	// Reading creates the levels it walks through, but not the last, as
	// reading $h{x}{y}{z} does in Perl: the same as reading from a scalar.
	viaHash, viaScalar := Hash(), Value(Hash())
	viaHash.At("x.y.z")
	viaScalar.At("x.y.z")
	if viaHash.Dump() != viaScalar.HV().Dump() || viaHash.At("x.y").HV().Exists("z") {
		t.Errorf("reading through: %s vs %s", viaHash.Dump(), viaScalar.HV().Dump())
	}
}

// A nil hash or array reads as undef and panics on writing, as Get does.
func TestContainerPathsNil(t *testing.T) {
	var h *HV
	var a *AV
	if h.At("a").Defined() || a.At("[0]").Defined() || h.Get("a").Defined() || a.Get(0).Defined() {
		t.Error("nil read wasn't undef")
	}
	if _, ok := h.AtOK("a"); ok {
		t.Error("nil AtOK")
	}
	if _, ok := a.GetOK(0); ok {
		t.Error("nil GetOK")
	}
	if h.DeleteAt("a").Defined() || a.DeleteAt("[0]").Defined() {
		t.Error("nil DeleteAt")
	}
	expectPanic(t, "nil hash write", "Not a HASH reference", func() { h.At("a").Set(1) })
	expectPanic(t, "nil array write", "Not an ARRAY reference", func() { a.At("[0]").Set(1) })
	// A first step of the wrong kind dies as it does on a scalar.
	expectPanic(t, "index into a hash", "Not an ARRAY reference", func() { Hash("k", 1).At("[0]").Set(1) })
}

// Matches gives each match's captures, as M with "g" does, without the
// array of references.
func TestMatches(t *testing.T) {
	for _, c := range []struct {
		text, re string
		flags    []string
	}{
		{"Depth: ≈12,500 - ≈15,000 m", `≈([\d,]+)`, nil},
		{"a1b22c333", `(\d)(\d)?`, nil},
		{"no digits", `\d`, nil},
		{"Aa aA", `a`, []string{"i"}},
		{"", `x*`, nil},
		{"ab\ncd", `^(\w)`, []string{"m"}},
	} {
		var got []string
		for m := range Value(c.text).Matches(c.re, c.flags...) {
			got = append(got, m.Dump())
		}
		var want []string
		for _, m := range Value(c.text).M(c.re, append(c.flags, "g")...).All() {
			want = append(want, m.AV().Dump())
		}
		if !slices.Equal(got, want) {
			t.Errorf("Matches(%q) on %q: %v, want %v", c.re, c.text, got, want)
		}
	}
	// Byte strings give byte strings, as M does.
	for m := range Value([]byte("\x01\xff")).Matches(`.`) {
		if m.Get(0).Kind() != Bytes {
			t.Errorf("byte string match is %v", m.Get(0).Kind())
		}
	}
	// Stopping early is fine.
	n := 0
	for range Value("aaaa").Matches(`a`) {
		n++
		break
	}
	if n != 1 {
		t.Error("break")
	}
	expectPanic(t, "g flag", "Matches", func() {
		for range Value("a").Matches(`a`, "g") {
		}
	})
}

// Delete, DeleteAt and DeleteOK agree: on every index of an array with a
// hole, from either end and past them; on keys present, missing, empty,
// numeric and dotted; and on nil arrays and hashes.
func TestDeleteConsistency(t *testing.T) {
	mk := func() *AV { return Array("a", "b", nil, "d") }
	for _, i := range []int{0, 1, 2, 3, -1, -4, 4, -5, 99} {
		a1, a2, a3 := mk(), mk(), mk()
		r1 := a1.Delete(i).Dump() + a1.Dump()
		r2 := a2.DeleteAt(fmt.Sprintf("[%d]", i)).Dump() + a2.Dump()
		r3 := first(a3.DeleteOK(i)).Dump() + a3.Dump()
		if r1 != r2 || r1 != r3 {
			t.Errorf("index %d: Delete %s, DeleteAt %s, DeleteOK %s", i, r1, r2, r3)
		}
	}
	mkh := func() *HV { return Hash("x", 1, "a.b", 2, "0", 3, "", 4) }
	for _, k := range []string{"x", "missing", "a.b", "0", ""} {
		h1, h2, h3 := mkh(), mkh(), mkh()
		r1 := h1.Delete(k).Dump() + h1.Dump()
		r2 := h2.DeleteAt(fmt.Sprintf("[%q]", k)).Dump() + h2.Dump()
		r3 := first(h3.DeleteOK(k)).Dump() + h3.Dump()
		if r1 != r2 || r1 != r3 {
			t.Errorf("key %q: Delete %s, DeleteAt %s, DeleteOK %s", k, r1, r2, r3)
		}
	}
	var na *AV
	var nh *HV
	if na.Delete(0).Defined() || na.DeleteAt("[0]").Defined() || second(na.DeleteOK(0)) ||
		nh.Delete("k").Defined() || nh.DeleteAt("k").Defined() || second(nh.DeleteOK("k")) {
		t.Error("deleting from nil wasn't undef")
	}
}

// DeleteOK takes keys as Get does: whole, dots and all, with ints for
// indices. It creates nothing, never panics on the shape of the data, and
// its ok says whether there was something there to remove.
func TestDeleteOK(t *testing.T) {
	doc, _ := FromJSON([]byte(`{"a.b":{"c":1},"a":{"b":{"c":2}},"list":[{"k":"v"},{"k":"w"}]}`))
	if got := first(doc.DeleteOK("a.b", "c")); got.Int() != 1 || doc.At(`["a.b"]`).HV().Exists("c") {
		t.Errorf("dotted key: %v", got)
	}
	if got := first(doc.DeleteOK("list", -1, "k")); got.Str() != "w" || doc.At("list[1]").HV().Exists("k") {
		t.Errorf("index: %v", got)
	}
	if got := first(doc.DeleteOK("a", "b", "c")); got.Int() != 2 {
		t.Errorf("undotted path: %v", got)
	}
	before := doc.Dump()
	if second(doc.DeleteOK("nope", "deeper", 0)) || second(doc.DeleteOK("list", 0, 5)) || doc.Dump() != before {
		t.Error("deleting a missing path created or removed something")
	}
	if second(doc.DeleteOK()) {
		t.Error("no keys: the scalar itself can't be deleted")
	}
	expectPanic(t, "bad key", "DeleteOK key must be a string or an int", func() { doc.DeleteOK(1.5) })
	expectPanic(t, "bad key after a missing step", "DeleteOK key must be", func() { doc.DeleteOK("nope", true) })

	// ok tells a removed null from nothing there, which the value can't.
	quake, _ := FromJSON([]byte(`{"felt": null, "mag": 4.6, "tsunami": 0, "stations": ["PAS", "GSC", "ISA"]}`))
	if v, ok := quake.DeleteOK("felt"); !ok || v.Defined() || quake.HV().Exists("felt") {
		t.Errorf("a null: %v, %v", v, ok)
	}
	if v, ok := quake.DeleteOK("felt"); ok || v.Defined() {
		t.Errorf("gone already: %v, %v", v, ok)
	}
	if _, ok := quake.DeleteOK("stations", 1); !ok {
		t.Error("an element in the middle")
	}
	if _, ok := quake.DeleteOK("stations", 1); ok {
		t.Error("a hole isn't there to remove")
	}
	if v, ok := quake.DeleteOK("stations", -1); !ok || v.Str() != "ISA" {
		t.Errorf("the last: %v, %v", v, ok)
	}

	// The wrong shape is nothing to delete, never a panic, and leaves the
	// data as it was.
	before = quake.Dump()
	for _, keys := range [][]any{{"tsunami", "x"}, {"tsunami", 0}, {"mag", "x", "y"}, {0}, {"stations", "0"}, {"stations", 0, "x"}, {"nope", 0}} {
		if _, ok := quake.DeleteOK(keys...); ok {
			t.Errorf("DeleteOK%v found something", keys)
		}
	}
	if quake.Dump() != before {
		t.Errorf("a DeleteOK of the wrong shape changed the data: %s", quake.Dump())
	}

	// On a hash or an array, the first key is typed, as for Get.
	h := quake.HV()
	if _, ok := h.DeleteOK("mag"); !ok || h.Exists("mag") {
		t.Error("HV.DeleteOK")
	}
	st := quake.Get("stations").AV()
	if v, ok := st.DeleteOK(0); !ok || v.Str() != "PAS" {
		t.Errorf("AV.DeleteOK: %v, %v", v, ok)
	}

	// A locked hash panics, as HV.Delete and Perl do.
	locked := Value(Hash("mag", 4.6))
	locked.Lock()
	expectPanic(t, "locked", "Attempt to delete readonly key 'mag'", func() { locked.DeleteOK("mag") })
}

// A nil hash or array isn't there, so the probes say so, as they do for a
// nil scalar: GetOK and AtOK with no steps are false, and Find matches
// nothing.
func TestContainerProbesNil(t *testing.T) {
	var nh *HV
	var na *AV
	for name, ok := range map[string]bool{
		"HV.GetOK()":   second(nh.GetOK()),
		"AV.GetOK()":   second(na.GetOK()),
		`HV.AtOK("")`:  second(nh.AtOK("")),
		`AV.AtOK("$")`: second(na.AtOK("$")),
		"HV.DeleteOK":  second(nh.DeleteOK("k")),
		"AV.DeleteOK":  second(na.DeleteOK(0)),
	} {
		if ok {
			t.Errorf("%s on nil: ok is true", name)
		}
	}
	if m := nh.Find("$"); m != nil {
		t.Errorf("nil HV.Find($) = %s", m.Dump())
	}
	if m := na.Find("$"); m != nil {
		t.Errorf("nil AV.Find($) = %s", m.Dump())
	}
	// A hash or array that's there is found, as before.
	if !second(Hash().GetOK()) || !second(Array().AtOK("")) || Hash().Find("$").Length() != 1 {
		t.Error("an empty hash or array is still there")
	}
}

func first(v *SV, _ bool) *SV    { return v }
func second(_ *SV, ok bool) bool { return ok }
