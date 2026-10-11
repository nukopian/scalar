package scalar

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
)

// AV is a Perl-style array of scalars. The zero value is an empty array.
//
// As in Perl, an array can have holes: elements that have never been set
// (after $a[5] = 1 on an empty array, or delete $a[2]). Holes read as undef
// but don't exist; see Exists.
type AV struct {
	items []*SV // nil entries are holes
}

const notArrayRef = "Not an ARRAY reference"

func nonCreatable(i int) string {
	return fmt.Sprintf("Modification of non-creatable array value attempted, subscript %d", i)
}

// Array returns an array of vals, as Perl's (...) list does: arrays and
// hashes among vals are flattened into it (see Push), and other values are
// converted as by SV.Set.
func Array(vals ...any) *AV {
	return &AV{items: flatten(vals)}
}

// flatten expands vals as Perl expands a list. Arrays and Go slices add
// their elements, and hashes and Go maps add their key/value
// pairs, so (@a, %h) is flatten(a, h). Anything else adds one element,
// including a scalar holding a reference: that's how a list holds a nested
// array or hash, as \@a and \%h do in Perl. Every value is copied.
func flatten(vals []any) []*SV {
	out := make([]*SV, 0, len(vals))
	var add func(v any)
	add = func(v any) {
		switch v := v.(type) {
		case *AV:
			for _, s := range v.All() {
				out = append(out, s.Copy())
			}
			return
		case *HV:
			for k, s := range v.All() {
				out = append(out, &SV{v: k}, s.Copy())
			}
			return
		case nil, *SV, string, bool, int, int64, float64:
			out = append(out, Scalar(v))
			return
		// The most common slices skip reflection.
		case []string:
			for _, e := range v {
				out = append(out, &SV{v: e})
			}
			return
		case []any:
			for _, e := range v {
				add(e)
			}
			return
		}
		switch rv := reflect.ValueOf(v); rv.Kind() {
		case reflect.Slice, reflect.Array:
			for i := range rv.Len() {
				add(rv.Index(i).Interface())
			}
		case reflect.Map:
			for it := rv.MapRange(); it.Next(); {
				out = append(out, &SV{v: mapKey(it.Key())}, Scalar(it.Value().Interface()))
			}
		default:
			out = append(out, Scalar(v))
		}
	}
	for _, v := range vals {
		add(v)
	}
	return out
}

// Length is Perl's scalar(@a).
func (a *AV) Length() int {
	if a == nil {
		return 0
	}
	return len(a.items)
}

// Scalar is Perl's scalar(@a), the number of elements. It is the same as
// Length.
func (a *AV) Scalar() int {
	return a.Length()
}

// LastIndex is Perl's $#a: Length() - 1, so -1 for an empty array.
func (a *AV) LastIndex() int {
	return a.Length() - 1
}

// SetLastIndex is Perl's $#a = n. It truncates the array, or extends it
// with holes. Any n below -1 empties it.
func (a *AV) SetLastIndex(n int) *AV {
	a.mustExist()
	n = max(n, -1)
	if n+1 <= len(a.items) {
		clear(a.items[n+1:])
		a.items = a.items[:n+1]
	} else {
		a.items = append(a.items, make([]*SV, n+1-len(a.items))...)
	}
	return a
}

// index resolves a Perl index, counting negative ones from the end. ok is
// false if a negative index reaches before the start.
func (a *AV) index(i int) (int, bool) {
	if i < 0 {
		i += a.Length()
	}
	return i, i >= 0
}

// Get returns element i; negative indices count from the end, as in Perl.
//
// A missing element reads as undef without being created. Writing to it
// (Set, Inc, or an autovivifying AV/HV deref) creates it, extending the
// array if needed, as $a[10] = 1 does.
//
// Perl can't create an element before the start of the array ($a[-10] on
// a short array), or an element of something that isn't an array. Get
// still returns undef for those, but writing to it panics, as Perl dies.
func (a *AV) Get(i int) *SV {
	if a == nil {
		return &SV{noCreate: notArrayRef}
	}
	idx, ok := a.index(i)
	if !ok {
		return &SV{noCreate: nonCreatable(i)}
	}
	if idx < len(a.items) && a.items[idx] != nil {
		return a.items[idx]
	}
	return &SV{viv: func(s *SV) { a.store(idx, s) }}
}

func (a *AV) store(i int, s *SV) {
	if i >= len(a.items) {
		a.items = append(a.items, make([]*SV, i+1-len(a.items))...)
	}
	a.items[i] = s
}

// Set assigns v to element i, extending the array if needed. It panics
// where Get's result can't be written (see Get).
func (a *AV) Set(i int, v any) *SV {
	return a.Get(i).Set(v)
}

// Exists is Perl's exists $a[i]: whether element i has been set. Holes
// and indices out of range don't exist.
func (a *AV) Exists(i int) bool {
	idx, ok := a.index(i)
	return ok && idx < a.Length() && a.items[idx] != nil
}

// Delete is Perl's delete $a[i]: it returns the element (undef if it
// didn't exist) and leaves a hole. Deleting the last element shrinks the
// array, along with any holes before it.
func (a *AV) Delete(i int) *SV {
	idx, ok := a.index(i)
	if !ok || idx >= a.Length() || a.items[idx] == nil {
		return &SV{}
	}
	s := a.items[idx]
	a.items[idx] = nil
	if idx == len(a.items)-1 {
		n := len(a.items)
		for n > 0 && a.items[n-1] == nil {
			n--
		}
		a.items = a.items[:n]
	}
	return s
}

// Push is Perl's push: it appends vals and returns a. As in Perl, an
// array among vals adds its elements, and a hash adds its key/value pairs,
// so a.Push(b) is push @a, @b. Go slices and maps flatten the
// same way. To add a nested array or hash, push a scalar holding a
// reference to it: a.Push(Scalar(b)) is push @a, \@b. Values are copied.
func (a *AV) Push(vals ...any) *AV {
	a.mustExist()
	a.items = append(a.items, flatten(vals)...)
	return a
}

// Unshift is Perl's unshift: it prepends vals, keeping their order, and
// returns a. vals are flattened as by Push.
func (a *AV) Unshift(vals ...any) *AV {
	a.mustExist()
	a.items = append(flatten(vals), a.items...)
	return a
}

// Pop removes and returns the last element, or undef if a is empty.
func (a *AV) Pop() *SV {
	if a.Length() == 0 {
		return &SV{}
	}
	s := a.items[len(a.items)-1]
	a.items[len(a.items)-1] = nil
	a.items = a.items[:len(a.items)-1]
	return orUndef(s)
}

// Shift removes and returns the first element, or undef if a is empty.
func (a *AV) Shift() *SV {
	if a.Length() == 0 {
		return &SV{}
	}
	s := a.items[0]
	a.items[0] = nil
	a.items = a.items[1:]
	return orUndef(s)
}

// Splice is Perl's splice @a, offset, length, list: it removes length
// elements starting at offset, inserts list in their place, and returns
// the removed elements.
//
// As in Perl, a negative offset counts from the end, and a negative length
// leaves that many elements at the end. A length running past the end
// stops at the end, so Splice(i, a.Length()) removes everything from i
// on. An offset past the end splices at the end. An offset before the
// start panics, as Perl dies.
func (a *AV) Splice(offset, length int, list ...any) *AV {
	a.mustExist()
	n := len(a.items)
	start, ok := a.index(offset)
	if !ok {
		panic("scalar: " + nonCreatable(offset))
	}
	start = min(start, n)
	if length < 0 {
		length = max(n-start+length, 0)
	}
	end := min(start+length, n)

	removed := &AV{items: make([]*SV, end-start)}
	for i, s := range a.items[start:end] {
		removed.items[i] = orUndef(s)
	}
	a.items = slices.Replace(a.items, start, end, flatten(list)...)
	return removed
}

// Slice is Perl's array slice @a[i, j, ...]: a new array of copies of the
// elements at those indices. Negative indices count from the end, and
// missing elements are undef.
func (a *AV) Slice(indices ...int) *AV {
	out := &AV{items: make([]*SV, len(indices))}
	for i, idx := range indices {
		out.items[i] = Scalar(a.Get(idx))
	}
	return out
}

// Items returns the elements. The slice is a copy; the SVs are not.
func (a *AV) Items() []*SV {
	if a == nil {
		return nil
	}
	out := make([]*SV, len(a.items))
	for i := range out {
		out[i] = a.Get(i)
	}
	return out
}

// StrSlice returns the elements as a Go []string, each in string context
// (see SV.Str), so undef and holes become "".
func (a *AV) StrSlice() []string { return convertSlice(a, (*SV).Str) }

// IntSlice returns the elements as a Go []int64, each in integer context
// (see SV.Int), so values that aren't numbers become 0.
func (a *AV) IntSlice() []int64 { return convertSlice(a, (*SV).Int) }

// NumSlice returns the elements as a Go []float64, each in numeric context
// (see SV.Num).
func (a *AV) NumSlice() []float64 { return convertSlice(a, (*SV).Num) }

// AnySlice returns the elements as plain Go values (see SV.Any), for code
// that doesn't use this package.
func (a *AV) AnySlice() []any { return convertSlice(a, (*SV).Any) }

// convertSlice applies f to every element of a, holes included. The result
// is never nil, so it encodes as [] in JSON.
func convertSlice[T any](a *AV, f func(*SV) T) []T {
	out := make([]T, a.Length())
	for i := range out {
		out[i] = f(a.items[i]) // a hole is a nil *SV, which reads as undef
	}
	return out
}

// All iterates over index/element pairs. Holes are yielded as undef.
func (a *AV) All() iter.Seq2[int, *SV] {
	return func(yield func(int, *SV) bool) {
		for i := range a.Length() {
			if !yield(i, a.Get(i)) {
				return
			}
		}
	}
}

// Join is Perl's join.
func (a *AV) Join(sep string) string {
	parts := make([]string, a.Length())
	for i, s := range a.All() {
		parts[i] = s.Str()
	}
	return strings.Join(parts, sep)
}

// Map is Perl's map: a new array of f applied to each element. As in Perl,
// if f returns an array or hash it is flattened into the result (see Push),
// so f can return several values, or none.
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
	out := &AV{items: make([]*SV, a.Length())}
	for i, s := range a.All() {
		out.items[i] = Scalar(s)
	}
	return out
}

func (a *AV) mustExist() {
	if a == nil {
		panic("scalar: " + notArrayRef)
	}
}

func orUndef(s *SV) *SV {
	if s == nil {
		return &SV{}
	}
	return s
}
