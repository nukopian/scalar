package scalar

import (
	"sort"
	"strings"
	"testing"
)

var kindNames = map[Kind]string{Undef: "Undef", Bool: "Bool", Int: "Int", Float: "Float",
	String: "String", Bytes: "Bytes", ArrayRef: "ArrayRef", HashRef: "HashRef"}

func TestApply(t *testing.T) {
	doc, err := FromJSON([]byte(`{" k ": " v ", "n": null, "num": 42, "ok": true,
		"deep": {"x": [" y ", [" z "]]}}`))
	if err != nil {
		t.Fatal(err)
	}
	tags := Array(" a ")
	doc.HV().Set("list", Value(tags))
	doc.HV().Set("again", Value(tags)) // the same array twice
	doc.HV().Set("self", doc)          // and a loop back to the top
	doc.At("blob").Set([]byte(" b "))

	var seen []string
	got := doc.Apply(func(v *SV) any {
		seen = append(seen, kindNames[v.Kind()]+":"+v.Str())
		if v.Kind() == String {
			return v.Trim()
		}
		return v
	})
	if got != doc {
		t.Error("Apply should return the scalar it was called on")
	}
	sort.Strings(seen)
	want := "Bool:1|Bytes: b |Int:42|String: a |String: v |String: y |String: z |Undef:"
	if strings.Join(seen, "|") != want {
		t.Errorf("f saw %s,\nwant %s", strings.Join(seen, "|"), want)
	}
	for path, want := range map[string]string{
		`{ k }`: "v", "list[0]": "a", "again[0]": "a", "deep.x[0]": "y", "deep.x[1][0]": "z",
		"num": "42", "blob": " b ",
	} {
		if got := doc.At(path).Str(); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if !doc.HV().Exists(" k ") || doc.HV().Exists("k") {
		t.Error("keys should be left as they are")
	}
	if doc.At("num").Kind() != Int || doc.At("blob").Kind() != Bytes {
		t.Error("returning v should leave it exactly as it was")
	}
	if v, ok := doc.AtOK("n"); !ok || v.Defined() {
		t.Error("returning undef as it is should keep it present and undef")
	}

	// Returning nil makes a value undef; returning an array stores a
	// reference, which isn't walked.
	calls := 0
	h := Hash("a", 1, "b", 2)
	h.Apply(func(v *SV) any {
		calls++
		if v.Int() == 1 {
			return nil
		}
		return Array("x", "y")
	})
	if calls != 2 || h.Get("a").Defined() || h.Get("b").Ref() != "ARRAY" {
		t.Errorf("calls %d, a %v, b %s", calls, h.Get("a"), h.Get("b").Ref())
	}

	// Holes are skipped, and stay holes; a plain scalar is f's only value.
	a := Array(1)
	a.Set(3, 2)
	calls = 0
	a.Apply(func(v *SV) any { calls++; return v.Mul(10) })
	if calls != 2 || a.Exists(1) || a.Get(0).Int() != 10 || a.Get(3).Int() != 20 {
		t.Errorf("holes: calls %d, exists(1) %v, %v %v", calls, a.Exists(1), a.Get(0), a.Get(3))
	}
	if s := Value("x").Apply(func(v *SV) any { return v.Uc() }); s.Str() != "X" {
		t.Errorf("a plain scalar: %q", s.Str())
	}
	if (*SV)(nil).Apply(func(v *SV) any { return v }) != nil {
		t.Error("Apply on nil should do nothing")
	}

	// Returning v from a missing element doesn't create it.
	m := Hash()
	m.Get("missing").Apply(func(v *SV) any { return v })
	if m.Exists("missing") {
		t.Error("returning v should write nothing")
	}
}
