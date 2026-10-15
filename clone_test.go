package scalar

import "testing"

func TestClone(t *testing.T) {
	orig, err := FromJSON([]byte(`{"tags": ["go"], "meta": {"n": 1}, "none": null}`))
	if err != nil {
		t.Fatal(err)
	}
	orig.At("blob").Set([]byte{0x89, 'P'})
	shared := orig.At("tags").AV()
	orig.HV().Set("again", Value(shared)) // the same array twice
	orig.HV().Set("self", orig)           // and a loop back to the top

	c := orig.Clone()
	if c == orig || c.HV() == orig.HV() {
		t.Fatal("Clone should make a new hash")
	}
	c.At("tags").AV().Push("perl")
	c.At("meta.n").Set(2)
	c.At("blob").SubstrReplace(0, 1, "x")
	if got := orig.At("tags").AV().Join(","); got != "go" {
		t.Errorf("the original's tags changed: %s", got)
	}
	if orig.At("meta.n").Int() != 1 || orig.At("blob").Str() != "\x89P" {
		t.Error("the original's values changed")
	}

	// The clone has the original's shape: shared once, loops kept.
	if c.At("again").AV() != c.At("tags").AV() {
		t.Error("an array appearing twice should be copied once, as dclone does")
	}
	if c.At("again").AV() == shared {
		t.Error("the shared array should be copied")
	}
	if c.At("self").HV() != c.HV() {
		t.Error("a loop should lead back into the clone")
	}
	if c.At("blob").Kind() != Bytes {
		t.Error("a byte string should stay one")
	}
	if v, ok := c.AtOK("none"); !ok || v.Defined() {
		t.Error("a null should stay present and undef")
	}

	// Holes stay holes, as with dclone.
	a := Array(1)
	a.Set(3, 2)
	ca := a.Clone()
	if ca.Length() != 4 || ca.Exists(1) || !ca.Exists(3) {
		t.Errorf("holes: length %d, exists(1) %v", ca.Length(), ca.Exists(1))
	}

	// Plain values, undef and nil receivers.
	if Value(42).Clone().Int() != 42 || (*SV)(nil).Clone().Defined() {
		t.Error("cloning a plain value, or nil, should copy it")
	}
	if (*AV)(nil).Clone() != nil || (*HV)(nil).Clone() != nil {
		t.Error("cloning a nil array or hash should give nil")
	}
	h := Hash("a", Value(Array(1, 2)))
	if ch := h.Clone(); ch.Get("a").AV() == h.Get("a").AV() || ch.Length() != 1 {
		t.Error("HV.Clone should copy nested arrays")
	}
}

// A clone shares byte strings, which nothing changes in place: changing
// either side replaces its value, leaving the other alone.
func TestCloneSharesByteStrings(t *testing.T) {
	orig := Value(Hash("b", []byte{1, 2, 3}))
	clone := orig.Clone()
	if clone.At("b").Str() != "\x01\x02\x03" || clone.At("b").Kind() != Bytes {
		t.Fatalf("clone: %v", clone.At("b"))
	}
	clone.At("b").ConcatInPlace("x")
	clone.At("b").S(`\x01`, "z")
	if orig.At("b").Str() != "\x01\x02\x03" {
		t.Errorf("original changed: %q", orig.At("b").Str())
	}
	b := orig.At("b").Bytes()
	b[0] = 9
	if orig.At("b").Str() != "\x01\x02\x03" || clone.At("b").Str() == "" {
		t.Error("Bytes didn't return a copy")
	}
}
