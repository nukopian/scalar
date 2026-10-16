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
	items   []*SV // nil entries are holes
	eachPos int   // how far Each has got
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
		if plainValue(v) {
			out = append(out, Value(v))
			return
		}
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
				out = append(out, &SV{v: mapKey(it.Key())}, Value(it.Value().Interface()))
			}
		default:
			out = append(out, Value(v))
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

// Defined is Perl's defined $a[i]: whether element i exists and isn't
// undef. Negative indices count from the end, as in Exists.
func (a *AV) Defined(i int) bool {
	idx, ok := a.index(i)
	return ok && idx < a.Length() && a.items[idx].Defined() // a hole is nil
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

// Undef is Perl's undef @a: it empties a in place, so everything referring
// to a sees it empty, and returns a, now empty. Each starts again from the
// beginning. A nil a panics, as Perl dies on undef @$x for a non-array.
func (a *AV) Undef() *AV {
	a.mustExist()
	a.items, a.eachPos = nil, 0
	return a
}

// Push is Perl's push: it appends vals and returns a. As in Perl, an
// array among vals adds its elements, and a hash adds its key/value pairs,
// so a.Push(b) is push @a, @b. Go slices and maps flatten the
// same way. To add a nested array or hash, push a scalar holding a
// reference to it: a.Push(Value(b)) is push @a, \@b. Values are copied.
func (a *AV) Push(vals ...any) *AV {
	a.mustExist()
	if plainValues(vals) {
		for _, v := range vals {
			a.items = append(a.items, Value(v))
		}
		return a
	}
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

// PopN removes the last n elements and returns them, in array order, as
// splice(@a, -n) does: Array(1, 2, 3, 4).PopN(2) gives 3, 4 and leaves 1,
// 2. Calling Pop n times would give them the other way round. Unlike
// splice, which dies when n is more than the array holds, PopN removes and
// returns what there is. As with Pop, the elements are the array's own,
// not copies. n must not be negative.
func (a *AV) PopN(n int) *AV {
	n = countN("PopN", n, a.Length())
	if n == 0 {
		return &AV{}
	}
	return a.Splice(a.Length()-n, n)
}

// ShiftN removes the first n elements and returns them, in array order, as
// splice(@a, 0, n) does: Array(1, 2, 3, 4).ShiftN(2) gives 1, 2 and leaves
// 3, 4. If n is more than the array holds, it removes and returns them all.
// As with Shift, the elements are the array's own, not copies. n must not
// be negative.
func (a *AV) ShiftN(n int) *AV {
	n = countN("ShiftN", n, a.Length())
	if n == 0 {
		return &AV{}
	}
	return a.Splice(0, n)
}

// countN checks the count given to PopN or ShiftN, and limits it to the
// array's length.
func countN(op string, n, length int) int {
	if n < 0 {
		panic(fmt.Sprintf("scalar: %s count %d is negative", op, n))
	}
	return min(n, length)
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

// Values is Perl's values @a: the elements. Given indices, it's the array
// slice @a[i, j, ...] instead: the elements at those indices, in that
// order. Negative indices count from the end, and the indices may be
// numbers, scalars or arrays of them, flattened as a list is (see Push), so
// a.Values(picks) is @a[@picks].
//
// As in Perl, the elements are the array's own, not copies: changing one
// changes a. A missing element reads as undef and isn't created unless you
// write to it, as with Get. Use Copy on an element for a value of your own.
func (a *AV) Values(indices ...any) *AV {
	if len(indices) == 0 {
		a.resetEach() // as values @a does
		return &AV{items: a.Items()}
	}
	picks := flatten(indices)
	out := &AV{items: make([]*SV, len(picks))}
	for i, x := range picks {
		out.items[i] = a.Get(int(x.Int()))
	}
	return out
}

// Pairs is Perl's index/value slice %a[i, j, ...]: a new array of
// index/value pairs for those indices, in that order, given as for Values.
// Without indices, it's every index and element. As in Perl, each index is
// given as written, so -1 stays -1 while its value is the last element's;
// the values are the array's own, and a missing element is undef and isn't
// created unless you write to it.
func (a *AV) Pairs(indices ...any) *AV {
	var picks []int
	if len(indices) == 0 {
		a.resetEach()
		for i := range a.Length() {
			picks = append(picks, i)
		}
	} else {
		for _, x := range flatten(indices) {
			picks = append(picks, int(x.Int()))
		}
	}
	out := &AV{items: make([]*SV, 0, 2*len(picks))}
	for _, i := range picks {
		out.items = append(out.items, &SV{v: int64(i)}, a.Get(i))
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

// convertSlice applies f to every element of a, holes included. The result
// is never nil, so it encodes as [] in JSON.
func convertSlice[T any](a *AV, f func(*SV) T) []T {
	out := make([]T, a.Length())
	for i := range out {
		out[i] = f(a.items[i]) // a hole is a nil *SV, which reads as undef
	}
	return out
}

// Each is Perl's each @a (Perl 5.12 and later): the next index and element,
// with ok true, or ok false once every element has been returned, after
// which it starts again from the beginning. The element is the array's own,
// and a hole is undef:
//
//	for i, v, ok := a.Each(); ok; i, v, ok = a.Each() {   // while (my ($i, $v) = each @a)
//		...
//	}
//
// As in Perl, the array remembers how far Each has got, and Values and
// Pairs (without indices) reset it, as keys @a and values @a do. For a loop
// without any of this, use All.
func (a *AV) Each() (index int, value *SV, ok bool) {
	if a == nil {
		return 0, nil, false
	}
	if a.eachPos < len(a.items) {
		i := a.eachPos
		a.eachPos++
		return i, a.Get(i), true
	}
	a.resetEach()
	return 0, nil, false
}

// resetEach starts Each again from the beginning.
func (a *AV) resetEach() {
	if a != nil {
		a.eachPos = 0
	}
}

// All iterates over index/element pairs. Holes are yielded as undef. Unlike
// Each, every loop starts from the beginning, and it doesn't change where
// Each has got to.
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
	out := a.copy()
	slices.SortStableFunc(out.items, orStrCmp(cmp))
	return out
}

// SortInPlace is Perl's @a = sort @a: it sorts a itself, as Sort would, and
// returns a. As in Perl, holes become undef elements that exist.
func (a *AV) SortInPlace(cmp func(x, y *SV) int) *AV {
	if a == nil {
		return nil
	}
	a.fillHoles()
	slices.SortStableFunc(a.items, orStrCmp(cmp))
	return a
}

// Reverse is Perl's reverse: a reversed copy of a.
func (a *AV) Reverse() *AV {
	out := a.copy()
	slices.Reverse(out.items)
	return out
}

// ReverseInPlace is Perl's @a = reverse @a: it reverses a itself and
// returns a. As in Perl, holes become undef elements that exist.
func (a *AV) ReverseInPlace() *AV {
	if a == nil {
		return nil
	}
	a.fillHoles()
	slices.Reverse(a.items)
	return a
}

func orStrCmp(cmp func(x, y *SV) int) func(x, y *SV) int {
	if cmp == nil {
		return StrCmp
	}
	return cmp
}

// fillHoles makes every hole an existing undef element, as assigning a list
// back to an array does in Perl.
func (a *AV) fillHoles() {
	for i, s := range a.items {
		if s == nil {
			a.items[i] = &SV{}
		}
	}
}

func (a *AV) copy() *AV {
	out := &AV{items: make([]*SV, a.Length())}
	for i, s := range a.All() {
		out.items[i] = Value(s)
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
