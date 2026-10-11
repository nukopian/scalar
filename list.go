package scalar

import (
	"fmt"
	"math"
)

// List utilities, after Perl's List::Util. The predicate functions play the
// part of List::Util's blocks: a.AnyMatch(f) is any { f($_) } @a.

// AnyMatch is List::Util's any: whether f is true for at least one element.
// It is false for an empty array.
func (a *AV) AnyMatch(f func(*SV) bool) bool {
	for _, s := range a.All() {
		if f(s) {
			return true
		}
	}
	return false
}

// AllMatch is List::Util's all: whether f is true for every element. It is
// true for an empty array.
func (a *AV) AllMatch(f func(*SV) bool) bool {
	for _, s := range a.All() {
		if !f(s) {
			return false
		}
	}
	return true
}

// NoneMatch is List::Util's none: whether f is false for every element. It
// is true for an empty array.
func (a *AV) NoneMatch(f func(*SV) bool) bool {
	return !a.AnyMatch(f)
}

// FirstMatch is List::Util's first: the first element for which f is true,
// or undef if there's none. Like Get, it returns the element itself, not a
// copy.
func (a *AV) FirstMatch(f func(*SV) bool) *SV {
	for _, s := range a.All() {
		if f(s) {
			return s
		}
	}
	return &SV{}
}

// MaxStr is List::Util's maxstr: the element that sorts last as a string,
// or undef for an empty array. Like Get, it returns the element itself.
func (a *AV) MaxStr() *SV {
	return a.extreme(func(x, best *SV) bool { return x.Gt(best) })
}

// MinStr is List::Util's minstr: the element that sorts first as a string,
// or undef for an empty array. Like Get, it returns the element itself.
func (a *AV) MinStr() *SV {
	return a.extreme(func(x, best *SV) bool { return x.Lt(best) })
}

// MaxNum is List::Util's max: the element with the largest numeric value,
// or undef for an empty array. Like Get, it returns the element itself.
// Values compare as numbers, so "10" beats 9, and integers compare exactly.
// As in List::Util, a NaN never wins a comparison, so it's only the result
// if it comes first.
func (a *AV) MaxNum() *SV {
	return a.extreme(func(x, best *SV) bool { return x.NumGt(best) })
}

// MinNum is List::Util's min: the element with the smallest numeric value,
// or undef for an empty array. It treats values as MaxNum does.
func (a *AV) MinNum() *SV {
	return a.extreme(func(x, best *SV) bool { return x.NumLt(best) })
}

// extreme returns the first element that no later element beats.
func (a *AV) extreme(beats func(x, best *SV) bool) *SV {
	var best *SV
	for _, s := range a.All() {
		if best == nil || beats(s, best) {
			best = s
		}
	}
	if best == nil {
		return &SV{}
	}
	return best
}

// SplitAt splits a into its first n elements and the rest, as Perl's
// my ($x1, ..., $xn, @tail) = @a does. head always has exactly n elements,
// padded with undef if a is shorter, so head.Get(i) for i < n is always
// meaningful. Both arrays hold copies, and a isn't changed; to remove the
// elements as well, use Shift or Splice. n must not be negative. The name
// comes from Haskell's splitAt, which also returns the head and the rest.
func (a *AV) SplitAt(n int) (head, tail *AV) {
	h, tail := a.SplitAtSlice(n)
	return &AV{items: h}, tail
}

// SplitAtSlice is SplitAt with the head as a Go slice of exactly n elements, so
// it can be unpacked: h, tail := a.SplitAtSlice(3); x, y, z := h[0], h[1], h[2].
func (a *AV) SplitAtSlice(n int) (head []*SV, tail *AV) {
	if n < 0 {
		panic(fmt.Sprintf("scalar: SplitAt count %d is negative", n))
	}
	var items []*SV // Copy turns holes (nil entries) into undef
	if a != nil {
		items = a.items
	}
	head = make([]*SV, n)
	for i := range head {
		if i < len(items) {
			head[i] = items[i].Copy()
		} else {
			head[i] = &SV{}
		}
	}
	tail = &AV{items: make([]*SV, 0, max(len(items)-n, 0))}
	for _, s := range items[min(n, len(items)):] {
		tail.items = append(tail.items, s.Copy())
	}
	return head, tail
}

// UniqueStr is List::Util's uniqstr: a new array of copies of the elements,
// keeping only the first of each distinct string, in order. Undef counts as
// "", as in uniqstr.
func (a *AV) UniqueStr() *AV {
	return a.unique(func(s *SV) any { return s.Str() })
}

// UniqueNum is List::Util's uniqnum: a new array of copies of the elements,
// keeping only the first of each distinct number, in order. Values compare
// as numbers, so "1", "1.0" and 1 are the same; integers compare exactly,
// however large; 0 and -0 are the same, and so are all NaNs.
func (a *AV) UniqueNum() *AV {
	return a.unique(func(s *SV) any {
		iv, nv, isInt := s.numify()
		switch {
		case isInt:
			return iv
		case math.IsNaN(nv):
			return "NaN"
		case nv == math.Trunc(nv) && nv >= math.MinInt64 && nv < math.MaxInt64:
			return int64(nv) // so 1.0 matches 1, and -0 matches 0
		}
		return nv
	})
}

func (a *AV) unique(key func(*SV) any) *AV {
	seen := make(map[any]bool, a.Length())
	out := &AV{}
	for _, s := range a.All() {
		k := key(s)
		if !seen[k] {
			seen[k] = true
			out.items = append(out.items, s.Copy())
		}
	}
	return out
}
