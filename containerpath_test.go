package scalar

import (
	"fmt"
	"slices"
	"testing"
)

// At, AtOK, DeleteAt, Path and PathOK on a hash or array give what the same
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
		want, wantOK := doc.PathOK(keys...)
		got, gotOK := doc.HV().PathOK(keys...)
		if d := got.Diff(want); d != "" || gotOK != wantOK || doc.HV().Path(keys...).Diff(doc.Path(keys...)) != "" {
			t.Errorf("HV.Path(%v): %s", keys, d)
		}
	}
	arr := newDoc().At("list").AV()
	if arr.At("[1][1]").Int() != 3 || arr.Path(1, 0).Int() != 2 || arr.At("[0]").Int() != 1 {
		t.Errorf("AV paths: %v %v", arr.At("[1][1]"), arr.Path(1, 0))
	}
	if v, ok := arr.AtOK("[5]"); ok || v.Defined() {
		t.Error("AV.AtOK past the end")
	}
	if _, ok := arr.PathOK(1, 0); !ok {
		t.Error("AV.PathOK")
	}

	// Writing through creates in the container, as autovivification does.
	h := Hash()
	h.At("tally.right").Inc()
	h.At("tally.right").Inc()
	h.Path("tally", "wrong").Set(1)
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
	if h.At("a").Defined() || a.At("[0]").Defined() || h.Path("a").Defined() || a.Path(0).Defined() {
		t.Error("nil read wasn't undef")
	}
	if _, ok := h.AtOK("a"); ok {
		t.Error("nil AtOK")
	}
	if _, ok := a.PathOK(0); ok {
		t.Error("nil PathOK")
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

// Delete, DeleteAt and DeletePath agree: on every index of an array with a
// hole, from either end and past them; on keys present, missing, empty,
// numeric and dotted; and on nil arrays and hashes.
func TestDeleteConsistency(t *testing.T) {
	mk := func() *AV { return Array("a", "b", nil, "d") }
	for _, i := range []int{0, 1, 2, 3, -1, -4, 4, -5, 99} {
		a1, a2, a3 := mk(), mk(), mk()
		r1 := a1.Delete(i).Dump() + a1.Dump()
		r2 := a2.DeleteAt(fmt.Sprintf("[%d]", i)).Dump() + a2.Dump()
		r3 := a3.DeletePath(i).Dump() + a3.Dump()
		if r1 != r2 || r1 != r3 {
			t.Errorf("index %d: Delete %s, DeleteAt %s, DeletePath %s", i, r1, r2, r3)
		}
	}
	mkh := func() *HV { return Hash("x", 1, "a.b", 2, "0", 3, "", 4) }
	for _, k := range []string{"x", "missing", "a.b", "0", ""} {
		h1, h2, h3 := mkh(), mkh(), mkh()
		r1 := h1.Delete(k).Dump() + h1.Dump()
		r2 := h2.DeleteAt(fmt.Sprintf("[%q]", k)).Dump() + h2.Dump()
		r3 := h3.DeletePath(k).Dump() + h3.Dump()
		if r1 != r2 || r1 != r3 {
			t.Errorf("key %q: Delete %s, DeleteAt %s, DeletePath %s", k, r1, r2, r3)
		}
	}
	var na *AV
	var nh *HV
	if na.Delete(0).Defined() || na.DeleteAt("[0]").Defined() || na.DeletePath(0).Defined() ||
		nh.Delete("k").Defined() || nh.DeleteAt("k").Defined() || nh.DeletePath("k").Defined() {
		t.Error("deleting from nil wasn't undef")
	}
}

// DeletePath takes keys as Path does: whole, dots and all, with ints for
// indices and compiled paths for several steps. It creates nothing.
func TestDeletePath(t *testing.T) {
	doc, _ := FromJSON([]byte(`{"a.b":{"c":1},"a":{"b":{"c":2}},"list":[{"k":"v"},{"k":"w"}]}`))
	if got := doc.DeletePath("a.b", "c"); got.Int() != 1 || doc.At(`["a.b"]`).HV().Exists("c") {
		t.Errorf("dotted key: %v", got)
	}
	if got := doc.DeletePath("list", -1, "k"); got.Str() != "w" || doc.At("list[1]").HV().Exists("k") {
		t.Errorf("index: %v", got)
	}
	if got := doc.DeletePath(MustPath("a.b"), "c"); got.Int() != 2 {
		t.Errorf("compiled path: %v", got)
	}
	before := doc.Dump()
	if doc.DeletePath("nope", "deeper", 0).Defined() || doc.DeletePath("list", 0, 5).Defined() || doc.Dump() != before {
		t.Error("deleting a missing path created or removed something")
	}
	if doc.DeletePath().Defined() {
		t.Error("no keys: the scalar itself can't be deleted")
	}
	expectPanic(t, "bad key", "Path key must be", func() { doc.DeletePath(1.5) })
}
