package scalar

import (
	"math/rand/v2"
	"testing"
)

// Equal walks the structures itself, for speed, rather than asking Diff;
// it must agree with Diff on every pair. These are random structures and
// random changes to them, with the cases Equal's rules turn on: 1 and
// 1.0, "1.0", true, undef and "", holes, a missing key and an undef one,
// loops, shared parts, and arrays against hashes.
func TestEqualAgreesWithDiff(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 11))
	leaves := []any{nil, "", 0, 1, 1.0, "1", "1.0", true, false, "x", []byte("x"), "0 but true"}
	var build func(depth int) *SV
	build = func(depth int) *SV {
		switch k := r.IntN(6); {
		case depth > 3 || k < 3:
			return Value(leaves[r.IntN(len(leaves))])
		case k == 3:
			a := Array()
			for i := range r.IntN(4) {
				if r.IntN(5) == 0 {
					a.Get(i + 2).Set(1) // leaves holes
				} else {
					a.Push(build(depth + 1))
				}
			}
			return Value(a)
		default:
			h := Hash()
			for range r.IntN(4) {
				h.Set(string(rune('a'+r.IntN(4))), build(depth+1))
			}
			return Value(h)
		}
	}
	// change returns a copy of s with one random change, or none.
	change := func(s *SV) *SV {
		c := s.Clone()
		switch v := c.val().(type) {
		case *HV:
			if keys := v.KeysSlice(); len(keys) > 0 && r.IntN(2) == 0 {
				k := keys[r.IntN(len(keys))]
				switch r.IntN(3) {
				case 0:
					v.Delete(k)
				case 1:
					v.Set(k, leaves[r.IntN(len(leaves))])
				default:
					v.Set(k, nil)
				}
			} else if r.IntN(2) == 0 {
				v.Set("z", leaves[r.IntN(len(leaves))])
			}
		case *AV:
			if v.Length() > 0 && r.IntN(2) == 0 {
				v.Set(r.IntN(v.Length()), leaves[r.IntN(len(leaves))])
			} else if r.IntN(2) == 0 {
				v.Push(nil)
			}
		default:
			if r.IntN(2) == 0 {
				c.Set(leaves[r.IntN(len(leaves))])
			}
		}
		return c
	}
	agree := func(x, y *SV) {
		t.Helper()
		if eq, diff := x.Equal(y), x.Diff(y); eq != (diff == "") {
			t.Fatalf("Equal = %v but Diff = %q\nx: %s\ny: %s", eq, diff, x.Dump(), y.Dump())
		}
	}
	for range 20000 {
		x := build(0)
		agree(x, change(x))
		agree(x, build(0))
		agree(x, x.Clone())
	}
	// Loops and shared parts.
	loop := Hash("a", 1)
	loop.Set("self", loop)
	other := Hash("a", 1)
	other.Set("self", other)
	agree(Value(loop), Value(other))
	other.Set("a", 2)
	agree(Value(loop), Value(other))
	shared := Array(1, 2)
	agree(Value(Array(shared, shared)), Value(Array(Array(1, 2), Array(1, 2))))
}
