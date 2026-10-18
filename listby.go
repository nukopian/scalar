package scalar

import "slices"

// The analogues of CPAN's List::UtilsBy: each takes a function that gives a
// key for an element, called once for each element, and works on the keys.
// The function may return a scalar or any Go value.

// keyed is an element with its key.
type keyed struct {
	elem *SV
	key  *SV
}

// keyed calls f once for each element of a, holes as undef.
func (a *AV) keyed(f func(*SV) any) []keyed {
	out := make([]keyed, 0, a.Length())
	for _, s := range a.All() {
		out = append(out, keyed{s, Value(f(s))})
	}
	return out
}

// sortBy is a sorted copy of a by f's keys, compared by cmp, stable, so
// equal keys keep their order whichever way the sort goes.
func (a *AV) sortBy(f func(*SV) any, cmp func(x, y *SV) int, reverse bool) *AV {
	ks := a.keyed(f)
	slices.SortStableFunc(ks, func(x, y keyed) int {
		if reverse {
			return cmp(y.key, x.key)
		}
		return cmp(x.key, y.key)
	})
	out := &AV{items: make([]*SV, len(ks))}
	for i, k := range ks {
		out.items[i] = Value(k.elem)
	}
	return out
}

// SortByStr is List::UtilsBy's sort_by: a sorted copy of a, ordered by the
// string keys f gives, as sort { $a cmp $b } would order them. f is called
// once for each element, not once for each comparison, and elements with
// equal keys keep their order.
//
//	quakes.SortByStr(func(q *scalar.SV) any { return q.At("place").Fc() })   // sort_by { fc $_->{place} }
func (a *AV) SortByStr(f func(*SV) any) *AV { return a.sortBy(f, StrCmp, false) }

// SortByNum is List::UtilsBy's nsort_by: as SortByStr, but comparing the
// keys as numbers.
//
//	quakes.SortByNum(func(q *scalar.SV) any { return q.At("mag.value") })   // nsort_by { $_->{mag}{value} }
func (a *AV) SortByNum(f func(*SV) any) *AV { return a.sortBy(f, NumCmp, false) }

// RevSortByStr is List::UtilsBy's rev_sort_by: SortByStr, in descending
// order. Elements with equal keys still keep their order, as they do in
// Perl, so it isn't the same as SortByStr(f).Reverse().
func (a *AV) RevSortByStr(f func(*SV) any) *AV { return a.sortBy(f, StrCmp, true) }

// RevSortByNum is List::UtilsBy's rev_nsort_by: SortByNum, in descending
// order, with elements of equal keys keeping their order.
//
//	quakes.RevSortByNum(func(q *scalar.SV) any { return q.At("mag.value") })   // the strongest first
func (a *AV) RevSortByNum(f func(*SV) any) *AV { return a.sortBy(f, NumCmp, true) }

// MaxBy is List::UtilsBy's max_by: the element with the largest numeric key,
// the first of them if several share it, or undef for an empty array. Like
// MaxNum, it returns the element itself, not a copy.
func (a *AV) MaxBy(f func(*SV) any) *SV {
	return a.extremeBy(f, func(x, best *SV) bool { return x.NumGt(best) })
}

// MinBy is List::UtilsBy's min_by: the element with the smallest numeric
// key; see MaxBy.
func (a *AV) MinBy(f func(*SV) any) *SV {
	return a.extremeBy(f, func(x, best *SV) bool { return x.NumLt(best) })
}

func (a *AV) extremeBy(f func(*SV) any, beats func(x, best *SV) bool) *SV {
	var best keyed
	for i, k := range a.keyed(f) {
		if i == 0 || beats(k.key, best.key) {
			best = k
		}
	}
	return orUndef(best.elem)
}

// UniqueBy is List::UtilsBy's uniq_by: a new array of copies of the
// elements, keeping only the first with each key, in order. Keys compare as
// strings, undef as "".
//
//	quakes.UniqueBy(func(q *scalar.SV) any { return q.At("id") })   // one of each event
func (a *AV) UniqueBy(f func(*SV) any) *AV {
	seen := map[string]bool{}
	out := &AV{}
	for _, k := range a.keyed(f) {
		if key := k.key.Str(); !seen[key] {
			seen[key] = true
			out.items = append(out.items, Value(k.elem))
		}
	}
	return out
}

// PartitionBy is List::UtilsBy's partition_by: a hash from each key f gives
// to an array of copies of the elements with that key, in order.
//
//	byNetwork := quakes.PartitionBy(func(q *scalar.SV) any { return q.At("net") })
//	byNetwork.Get("ci").AV()   // the quakes Southern California's network recorded
func (a *AV) PartitionBy(f func(*SV) any) *HV {
	out := newHV(0)
	for _, k := range a.keyed(f) {
		key := k.key.Str()
		group, ok := out.m[key]
		if !ok {
			group = &SV{v: &AV{}}
			out.m[key] = group
			out.order = nil
		}
		g := group.v.(*AV)
		g.items = append(g.items, Value(k.elem))
	}
	return out
}

// CountBy is List::UtilsBy's count_by: how many elements have each key f
// gives, as a Bag, so MostCommon lists the commonest first. count_by gives
// a hash, which the bag's Counts method gives too.
//
//	quakes.CountBy(func(q *scalar.SV) any { return q.At("region") }).MostCommon(5)
func (a *AV) CountBy(f func(*SV) any) *Bag {
	b := &Bag{}
	for _, k := range a.keyed(f) {
		b.c.add(k.key, 1)
	}
	return b
}

// ExtractBy is List::UtilsBy's extract_by: it removes the elements for
// which f returns true from a, and returns them, in order, as a new array.
// a keeps the rest, in order, and any holes in it become undef elements, as
// with SortInPlace. Unlike the other functions here, f returns a bool.
//
//	reviewed := quakes.ExtractBy(func(q *scalar.SV) bool { return q.At("status").Eq("reviewed") })
func (a *AV) ExtractBy(f func(*SV) bool) *AV {
	a.mustWrite()
	out := &AV{}
	if a == nil {
		return out
	}
	a.fillHoles()
	kept := a.items[:0]
	for _, s := range a.items {
		if f(s) {
			out.items = append(out.items, s)
		} else {
			kept = append(kept, s)
		}
	}
	clear(a.items[len(kept):])
	a.items = kept
	return out
}
