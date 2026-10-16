package scalar

import "testing"

// HV.Map is %new = map { f($_, $h{$_}) } keys %h.
func TestHVMap(t *testing.T) {
	h := Hash("a", 1, "b", 2)
	out := h.Map(func(k string, v *SV) (string, any) { return k + k, v.Mul(10) })
	if got := out.Keys().Sort(StrCmp).Join(","); got != "aa,bb" {
		t.Errorf("keys = %s, want aa,bb", got)
	}
	if out.Get("aa").Int() != 10 || out.Get("bb").Int() != 20 {
		t.Errorf("values = %v %v, want 10 20", out.Get("aa"), out.Get("bb"))
	}
	if got := h.Keys().Sort(StrCmp).Join(","); got != "a,b" || h.Get("a").Int() != 1 {
		t.Errorf("original changed: keys %s, a = %v", got, h.Get("a"))
	}

	// Returning v itself copies it: the new hash doesn't share the old one's scalars.
	same := h.Map(func(k string, v *SV) (string, any) { return k, v })
	same.Set("a", 99)
	if h.Get("a").Int() != 1 {
		t.Errorf("new hash shares a scalar with the old: a = %v", h.Get("a"))
	}

	// Keys that collide leave one of them.
	one := h.Map(func(string, *SV) (string, any) { return "k", 0 })
	if one.Length() != 1 || !one.Exists("k") {
		t.Errorf("collided keys = %v", one.Keys())
	}

	// A nil hash reads as empty, as Length and Keys treat it.
	var nilh *HV
	if n := nilh.Map(func(k string, v *SV) (string, any) { return k, v }).Length(); n != 0 {
		t.Errorf("nil Map has %d keys", n)
	}
}
