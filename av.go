package scalar

import (
	"iter"
	"slices"
	"strings"
)

// AV is a Perl-style array of scalars. The zero value is an empty array.
type AV struct {
	items []*SV
}

// Array returns an array holding vals, converted as by SV.Set.
func Array(vals ...any) *AV {
	return (&AV{}).Push(vals...)
}

// Len is Perl's scalar(@a).
func (a *AV) Len() int {
	if a == nil {
		return 0
	}
	return len(a.items)
}

// Get returns element i; negative indices count from the end. A missing
// element reads as undef without being created. Writing to it (Set, or an
// autovivifying AV/HV deref) extends the array to fit, as $a[10] = 1 does.
func (a *AV) Get(i int) *SV {
	if a == nil {
		return &SV{}
	}
	if i < 0 {
		i += len(a.items)
		if i < 0 {
			return &SV{} // Perl can't create these either
		}
	}
	if i < len(a.items) {
		return a.items[i]
	}
	return &SV{viv: func(s *SV) { a.store(i, s) }}
}

func (a *AV) store(i int, s *SV) {
	for len(a.items) <= i {
		a.items = append(a.items, &SV{})
	}
	a.items[i] = s
}

// Set assigns v to element i, extending the array if needed.
func (a *AV) Set(i int, v any) *SV {
	a.mustExist("Set")
	return a.Get(i).Set(v)
}

// Push appends vals and returns a.
func (a *AV) Push(vals ...any) *AV {
	a.mustExist("Push")
	for _, v := range vals {
		a.items = append(a.items, Scalar(v))
	}
	return a
}

// Unshift prepends vals, keeping their order, and returns a.
func (a *AV) Unshift(vals ...any) *AV {
	a.mustExist("Unshift")
	front := make([]*SV, len(vals))
	for i, v := range vals {
		front[i] = Scalar(v)
	}
	a.items = append(front, a.items...)
	return a
}

// Pop removes and returns the last element, or undef if a is empty.
func (a *AV) Pop() *SV {
	if a.Len() == 0 {
		return &SV{}
	}
	s := a.items[len(a.items)-1]
	a.items = a.items[:len(a.items)-1]
	return s
}

// Shift removes and returns the first element, or undef if a is empty.
func (a *AV) Shift() *SV {
	if a.Len() == 0 {
		return &SV{}
	}
	s := a.items[0]
	a.items = a.items[1:]
	return s
}

// Items returns the elements. The slice is a copy; the SVs are not.
func (a *AV) Items() []*SV {
	if a == nil {
		return nil
	}
	return slices.Clone(a.items)
}

// All iterates over index/element pairs.
func (a *AV) All() iter.Seq2[int, *SV] {
	return func(yield func(int, *SV) bool) {
		for i := range a.Len() {
			if !yield(i, a.items[i]) {
				return
			}
		}
	}
}

// Join is Perl's join.
func (a *AV) Join(sep string) string {
	parts := make([]string, a.Len())
	for i, s := range a.All() {
		parts[i] = s.Str()
	}
	return strings.Join(parts, sep)
}

// Map is Perl's map: a new array of f applied to each element.
func (a *AV) Map(f func(*SV) any) *AV {
	out := &AV{}
	for _, s := range a.All() {
		out.Push(f(s))
	}
	return out
}

// Grep is Perl's grep: a new array of copies of the elements f accepts.
func (a *AV) Grep(f func(*SV) bool) *AV {
	out := &AV{}
	for _, s := range a.All() {
		if f(s) {
			out.Push(s)
		}
	}
	return out
}

// Sort is Perl's sort: a sorted copy of a. A nil cmp sorts by string
// comparison, as bare sort does; pass NumCmp for sort { $a <=> $b }.
func (a *AV) Sort(cmp func(x, y *SV) int) *AV {
	if cmp == nil {
		cmp = StrCmp
	}
	out := a.copy()
	slices.SortStableFunc(out.items, cmp)
	return out
}

// Reverse is Perl's reverse: a reversed copy of a.
func (a *AV) Reverse() *AV {
	out := a.copy()
	slices.Reverse(out.items)
	return out
}

func (a *AV) copy() *AV {
	out := &AV{items: make([]*SV, a.Len())}
	for i, s := range a.All() {
		out.items[i] = Scalar(s)
	}
	return out
}

func (a *AV) mustExist(op string) {
	if a == nil {
		panic("scalar: " + op + " on nil *AV (not an ARRAY reference)")
	}
}
