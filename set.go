package scalar

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"strings"
)

// Sets and bags of scalars, after CPAN's Set::Scalar and Set::Bag. Perl has
// neither built in, so they take Go's usual names, a type and a New...
// constructor, rather than the SV/AV/HV family's.
//
// Members are compared as strings, as Perl hash keys (and both modules)
// compare them: 1, "1" and 1.0 are the same member, "1.0" and "Go" and "go"
// are each different. The first value inserted is the one kept. Members
// keep the order they were first inserted in, so output is reproducible;
// String sorts them, as the modules print them.

// counts is the core of Set and Bag: members by their string form, in the
// order first inserted, with how many of each there are. Its zero value is
// empty and ready to use.
type counts struct {
	order []string
	first map[string]*SV // the value first inserted for each member
	n     map[string]int
}

// add adds n of v. A new member is copied in, so changing v afterwards
// doesn't change the member.
func (c *counts) add(v *SV, n int) {
	if n <= 0 {
		return
	}
	if c.n == nil {
		c.first, c.n = map[string]*SV{}, map[string]int{}
	}
	key := v.Str()
	if c.n[key] == 0 {
		c.order = append(c.order, key)
		c.first[key] = v.Copy()
	}
	c.n[key] += n
}

// remove takes away up to n of the member key, removing it at zero.
func (c *counts) remove(key string, n int) {
	if n <= 0 || c.n[key] == 0 {
		return
	}
	if c.n[key] -= n; c.n[key] > 0 {
		return
	}
	delete(c.n, key)
	delete(c.first, key)
	c.order = slices.DeleteFunc(c.order, func(k string) bool { return k == key })
}

// count is how many of key there are.
func (c *counts) count(key string) int {
	if c == nil {
		return 0
	}
	return c.n[key]
}

// keys is the members' string forms, in order. Nil-safe.
func (c *counts) keys() []string {
	if c == nil {
		return nil
	}
	return c.order
}

// members is a new array of copies of the members, in order.
func (c *counts) members() *AV {
	ks := c.keys()
	out := &AV{items: make([]*SV, len(ks))}
	for i, k := range ks {
		out.items[i] = c.first[k].Copy()
	}
	return out
}

// sortedKeys is the members' string forms, sorted, as the modules print them.
func (c *counts) sortedKeys() []string {
	return slices.Sorted(slices.Values(c.keys()))
}

// Set is a set of scalars, after Set::Scalar: each member at most once. The
// zero value is an empty set, ready to use. Insert and Delete change the set;
// Union, Intersection and the other set operations return a new one.
type Set struct {
	c counts
}

// cs is s's counts, or nil for a nil *Set, which then reads as empty.
func (s *Set) cs() *counts {
	if s == nil {
		return nil
	}
	return &s.c
}

// NewSet is Set::Scalar->new(...): a set of vals, flattened as a list is
// (see AV.Push), so NewSet(array) and NewSet(goSlice) take their elements.
func NewSet(vals ...any) *Set {
	return new(Set).Insert(vals...)
}

// Insert is Set::Scalar's insert: it adds vals, flattened as by NewSet, and
// returns s. Values already in the set are ignored. Insert on a nil *Set
// panics.
func (s *Set) Insert(vals ...any) *Set {
	if s == nil {
		panic("scalar: Insert on a nil *Set")
	}
	for _, v := range flatten(vals) {
		if s.c.count(v.Str()) == 0 {
			s.c.add(v, 1)
		}
	}
	return s
}

// Delete is Set::Scalar's delete: it removes vals, flattened as by NewSet,
// and returns s. Values that aren't members are ignored.
func (s *Set) Delete(vals ...any) *Set {
	if s == nil {
		return s
	}
	for _, v := range flatten(vals) {
		s.c.remove(v.Str(), 1)
	}
	return s
}

// Has is Set::Scalar's has: whether v (compared as a string) is a member.
func (s *Set) Has(v any) bool {
	return s.cs().count(operand(v).Str()) > 0
}

// Length is Set::Scalar's size: the number of members.
func (s *Set) Length() int {
	return len(s.cs().keys())
}

// Members is Set::Scalar's members: a new array of copies of the members,
// in the order they were first inserted. Changing it doesn't change s.
func (s *Set) Members() *AV {
	if s == nil {
		return &AV{}
	}
	return s.c.members()
}

// All iterates over copies of the members, in the order they were first
// inserted.
func (s *Set) All() iter.Seq[*SV] {
	return func(yield func(*SV) bool) {
		for _, m := range s.Members().items {
			if !yield(m) {
				return
			}
		}
	}
}

// Copy returns a new set with the same members.
func (s *Set) Copy() *Set {
	return NewSet(s.Members())
}

// Union is Set::Scalar's union: a new set of every member of s or any of
// others, in order of first appearance.
func (s *Set) Union(others ...*Set) *Set {
	out := s.Copy()
	for _, o := range others {
		out.Insert(o.Members())
	}
	return out
}

// Intersection is Set::Scalar's intersection: a new set of the members of s
// that are also in every one of others.
func (s *Set) Intersection(others ...*Set) *Set {
	return s.filter(func(key string) bool {
		return !slices.ContainsFunc(others, func(o *Set) bool { return o.cs().count(key) == 0 })
	})
}

// Difference is Set::Scalar's difference: a new set of the members of s
// that are in none of others.
func (s *Set) Difference(others ...*Set) *Set {
	return s.filter(func(key string) bool {
		return !slices.ContainsFunc(others, func(o *Set) bool { return o.cs().count(key) > 0 })
	})
}

// SymmetricDifference is Set::Scalar's symmetric_difference: a new set of
// the members in exactly one of s and t.
func (s *Set) SymmetricDifference(t *Set) *Set {
	return s.Difference(t).Union(t.Difference(s))
}

// filter is a new set of the members of s whose string forms keep accepts.
func (s *Set) filter(keep func(key string) bool) *Set {
	out := new(Set)
	if s == nil {
		return out
	}
	for _, k := range s.c.order {
		if keep(k) {
			out.c.add(s.c.first[k], 1)
		}
	}
	return out
}

// IsSubset is Set::Scalar's is_subset: whether every member of s is in t.
// A set is a subset of itself, and the empty set of every set.
func (s *Set) IsSubset(t *Set) bool {
	return !slices.ContainsFunc(s.cs().keys(), func(k string) bool { return t.cs().count(k) == 0 })
}

// IsProperSubset is Set::Scalar's is_proper_subset: a subset of t that
// isn't equal to it.
func (s *Set) IsProperSubset(t *Set) bool {
	return s.IsSubset(t) && s.Length() < t.Length()
}

// IsSuperset is Set::Scalar's is_superset: whether s has every member of t.
func (s *Set) IsSuperset(t *Set) bool { return t.IsSubset(s) }

// IsEqual is Set::Scalar's is_equal: whether s and t have the same members,
// in any order.
func (s *Set) IsEqual(t *Set) bool { return s.Length() == t.Length() && s.IsSubset(t) }

// IsDisjoint is Set::Scalar's is_disjoint: whether s and t have no members
// in common. Unlike Set::Scalar, which says two empty sets aren't disjoint,
// it follows the mathematics: sets with nothing in common are disjoint,
// empty ones included.
func (s *Set) IsDisjoint(t *Set) bool {
	return !slices.ContainsFunc(s.cs().keys(), func(k string) bool { return t.cs().count(k) > 0 })
}

// String is the set as Set::Scalar prints it: its members sorted, between
// brackets, as in "(a b c)".
func (s *Set) String() string {
	if s == nil {
		return "()"
	}
	return "(" + strings.Join(s.c.sortedKeys(), " ") + ")"
}

// MarshalJSON encodes the set as an array of its members, in the order they
// were first inserted.
func (s *Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Members())
}

// Bag is a bag (multiset) of scalars, after Set::Bag: each member with a
// count of how many there are. The zero value is an empty bag, ready to use.
// Counts never go below zero: removing more than there is removes what there
// is, as Set::Bag does with its over_delete option (without it, Set::Bag
// dies). Insert and Delete change the bag; Sum, Union, Intersection and
// Difference return a new one.
type Bag struct {
	c counts
}

// NewBag returns a bag of vals, flattened as a list is (see AV.Push), each
// counting once, so NewBag("Go", "Go", "Perl") has two Go and one Perl. For
// Set::Bag->new(apples => 3), use NewBag().InsertN("apples", 3).
func NewBag(vals ...any) *Bag {
	return new(Bag).Insert(vals...)
}

// Insert adds one of each of vals, flattened as by NewBag, and returns b.
// Insert on a nil *Bag panics.
func (b *Bag) Insert(vals ...any) *Bag {
	if b == nil {
		panic("scalar: Insert on a nil *Bag")
	}
	for _, v := range flatten(vals) {
		b.c.add(v, 1)
	}
	return b
}

// InsertN is Set::Bag's insert(v => n): it adds n of v and returns b. A
// count of zero or less adds nothing.
func (b *Bag) InsertN(v any, n int) *Bag {
	if b == nil {
		panic("scalar: InsertN on a nil *Bag")
	}
	b.c.add(operand(v), n)
	return b
}

// Delete removes one of each of vals, flattened as by NewBag, and returns b.
// A member whose count reaches zero is removed; one that isn't there is
// ignored.
func (b *Bag) Delete(vals ...any) *Bag {
	if b == nil {
		return b
	}
	for _, v := range flatten(vals) {
		b.c.remove(v.Str(), 1)
	}
	return b
}

// DeleteN is Set::Bag's delete(v => n): it removes n of v, or all of it if
// there are fewer, and returns b.
func (b *Bag) DeleteN(v any, n int) *Bag {
	if b != nil {
		b.c.remove(operand(v).Str(), n)
	}
	return b
}

// Count is Set::Bag's grab(v): how many of v there are, 0 if none.
func (b *Bag) Count(v any) int {
	return b.cs().count(operand(v).Str())
}

// Length is the number of distinct members.
func (b *Bag) Length() int {
	return len(b.cs().keys())
}

// Total is the number of members counting repeats: the sum of the counts.
func (b *Bag) Total() int {
	total := 0
	for _, k := range b.cs().keys() {
		total += b.c.n[k]
	}
	return total
}

// cs is b's counts, or nil for a nil *Bag, which then reads as empty.
func (b *Bag) cs() *counts {
	if b == nil {
		return nil
	}
	return &b.c
}

// Members is Set::Bag's elements: a new array of copies of the distinct
// members, in the order they were first inserted.
func (b *Bag) Members() *AV {
	if b == nil {
		return &AV{}
	}
	return b.c.members()
}

// Counts returns the bag as a new hash from each member (as a string) to its
// count, for the rest of the package's tools.
func (b *Bag) Counts() *HV {
	h := newHV(b.Length())
	for _, k := range b.cs().keys() {
		h.m[k] = &SV{v: int64(b.c.n[k])}
	}
	return h
}

// MostCommon returns copies of the members with the highest counts, most
// common first, and the first inserted first among equal counts: the top n,
// or every member if n is left out or there are fewer.
func (b *Bag) MostCommon(n ...int) *AV {
	ks := slices.Clone(b.cs().keys())
	slices.SortStableFunc(ks, func(x, y string) int { return cmp.Compare(b.c.n[y], b.c.n[x]) })
	if len(n) > 0 {
		if n[0] < 0 {
			panic(fmt.Sprintf("scalar: MostCommon count %d is negative", n[0]))
		}
		ks = ks[:min(n[0], len(ks))]
	}
	out := &AV{items: make([]*SV, len(ks))}
	for i, k := range ks {
		out.items[i] = b.c.first[k].Copy()
	}
	return out
}

// All iterates over copies of the distinct members with their counts, in
// the order they were first inserted.
func (b *Bag) All() iter.Seq2[*SV, int] {
	return func(yield func(*SV, int) bool) {
		for _, k := range slices.Clone(b.cs().keys()) {
			if !yield(b.c.first[k].Copy(), b.c.n[k]) {
				return
			}
		}
	}
}

// Copy returns a new bag with the same members and counts.
func (b *Bag) Copy() *Bag {
	out := new(Bag)
	for m, n := range b.All() {
		out.c.add(m, n)
	}
	return out
}

// Sum is Set::Bag's sum: a new bag in which each member's count is its
// counts in b and others added together.
func (b *Bag) Sum(others ...*Bag) *Bag {
	out := b.Copy()
	for _, o := range others {
		for m, n := range o.All() {
			out.c.add(m, n)
		}
	}
	return out
}

// Union is Set::Bag's union: a new bag in which each member's count is the
// largest of its counts in b and others.
func (b *Bag) Union(others ...*Bag) *Bag {
	out := b.Copy()
	for _, o := range others {
		for m, n := range o.All() {
			out.c.add(m, n-out.c.count(m.Str()))
		}
	}
	return out
}

// Intersection is Set::Bag's intersection: a new bag of the members in b and
// every one of others, each with the smallest of its counts.
func (b *Bag) Intersection(others ...*Bag) *Bag {
	out := new(Bag)
	for m, n := range b.All() {
		for _, o := range others {
			n = min(n, o.Count(m))
		}
		out.c.add(m, n)
	}
	return out
}

// Difference is Set::Bag's difference: a new bag in which each member's
// count is its count in b less its counts in others, leaving out any that
// reach zero, as Set::Bag does with over_delete.
func (b *Bag) Difference(others ...*Bag) *Bag {
	out := b.Copy()
	for _, o := range others {
		for m, n := range o.All() {
			out.c.remove(m.Str(), n)
		}
	}
	return out
}

// String is the bag as Set::Bag prints it: members sorted, each with its
// count, as in "(apples => 3, pears => 1)".
func (b *Bag) String() string {
	if b == nil {
		return "()"
	}
	parts := make([]string, 0, b.Length())
	for _, k := range b.c.sortedKeys() {
		parts = append(parts, fmt.Sprintf("%s => %d", k, b.c.n[k]))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// MarshalJSON encodes the bag as an object from each member to its count,
// in the order the members were first inserted.
func (b *Bag) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("{")
	for i, k := range b.cs().keys() {
		if i > 0 {
			buf.WriteString(",")
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "%s:%d", key, b.c.n[k])
	}
	buf.WriteString("}")
	return []byte(buf.String()), nil
}
