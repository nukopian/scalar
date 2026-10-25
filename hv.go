package scalar

import (
	"fmt"
	"iter"
	"slices"
)

// HV is a Perl-style hash of scalars. The zero value is an empty hash.
//
// Like Perl, iteration order is unspecified and varies between hashes, but
// Keys, Values and All agree with each other until a key is added or deleted.
type HV struct {
	m     map[string]*SV
	order []string // cached iteration order; nil when stale

	// Each's position: the keys it's going through, and how far it's got.
	eachKeys []string
	eachPos  int

	locked bool // by Lock: its keys fixed, and reading one it hasn't got panics

	// Autovivify's settings for the values and below, off when true, and
	// the undef h was made for, until h attaches to it (see attach).
	noViv, noVivDeep bool
	owner            *SV
}

// Hash builds a hash from a key/value list, as %h = (k1 => v1, ...) does.
// The list is flattened first, as by AV.Push, so Hash(defaults, "k", "v")
// merges like %h = (%defaults, k => "v"), with later keys winning. Keys are
// stringified; a trailing key without a value maps to undef.
func Hash(kv ...any) *HV {
	if h, ok := plainHash(kv); ok {
		return h
	}
	flat := flatten(kv)
	h := newHV((len(flat) + 1) / 2)
	for i := 0; i < len(flat); i += 2 {
		v := &SV{}
		if i+1 < len(flat) {
			v = flat[i+1]
		}
		h.m[flat[i].Str()] = v
	}
	return h
}

// newHV returns an empty hash with room for n keys. Constructors fill h.m
// directly, skipping the autovivification machinery in Get and Set.
func newHV(n int) *HV {
	return &HV{m: make(map[string]*SV, n)}
}

// Length is Perl's scalar(keys %h).
func (h *HV) Length() int {
	if h == nil {
		return 0
	}
	return len(h.m)
}

// Scalar is Perl's scalar(%h), which since Perl 5.26 is the number of keys.
// It is the same as Length.
func (h *HV) Scalar() int {
	return h.Length()
}

// Get is Perl's $h{key}: it returns the value for key. A missing key reads
// as undef without being created; writing through the returned SV (Set, or
// an autovivifying AV/HV deref) creates it, so h.Get("a").Set(1) is
// $h{a} = 1.
//
// More keys go on from there as a scalar's Get takes them, so
// h.Get("a", 0, "b") is $h{a}[0]{b}, the same as h.Get("a").Get(0, "b").
// That's a route, not a slice: Perl's @h{"a", "b"} is h.Values("a", "b").
//
// Get on a nil *HV (something that wasn't a hash) returns undef, but
// writing to it panics, as Perl dies with "Not a HASH reference".
func (h *HV) Get(key string, more ...any) *SV {
	if len(more) > 0 {
		return h.get(key).Get(more...)
	}
	return h.get(key)
}

// get is Get with one key, for the package's own use on hot paths.
func (h *HV) get(key string) *SV {
	if h == nil {
		return noCreateSV("Not a HASH reference")
	}
	if s, ok := h.m[key]; ok {
		return s
	}
	if h.locked && !h.noViv { // with autovivification off, a missing key reads as undef
		panic(fmt.Sprintf("scalar: Attempt to access disallowed key '%s' in a restricted hash", key))
	}
	return &SV{viv: func(s *SV) { h.store(key, s) }, noViv: h.noViv, noVivDeep: h.noVivDeep}
}

func (h *HV) store(key string, s *SV) {
	if h.locked {
		if _, ok := h.m[key]; ok {
			panic("scalar: " + readOnlyMsg)
		}
		panic(fmt.Sprintf("scalar: Attempt to access disallowed key '%s' in a restricted hash", key))
	}
	if h.owner != nil {
		h.attach()
	}
	adopt(s, h.noViv, h.noVivDeep)
	if h.m == nil {
		h.m = map[string]*SV{}
	}
	if _, ok := h.m[key]; !ok {
		h.order = nil
	}
	h.m[key] = s
}

// Set is Perl's $h{key} = v: it assigns v to key and returns the stored
// SV. It's h.Get(key).Set(v) in one step, as a scalar's s.Get(key).Set(v)
// is $s->{key} = v.
func (h *HV) Set(key string, v any) *SV {
	if s, ok := h.setNew(key, v); ok {
		return s
	}
	return h.Get(key).Set(v)
}

// Push adds key/value pairs to h and returns h. The list is flattened as
// Hash and AV.Push flatten theirs, so it can be pairs, another hash, a Go
// map, or the result of Pairs:
//
//	h.Push("a", 1, "b", 2)        // $h{a} = 1; $h{b} = 2
//	h.Push(defaults)              // merge another hash in
//	h.Push(other.Pairs("x", "y")) // copy just those keys over
//
// Each pair is assigned as $h{k} = v is, so an existing key keeps its
// scalar, updated in place, and anything holding it (from Get or Values)
// sees the change. Later keys win, and a trailing key without a value is
// set to undef, as in Hash. Values are copied in. Perl has no single
// operator for this; %h = (%h, ...) is the nearest, though it rebuilds the
// hash. Hash(h, ...) makes a new merged hash instead. Push on a nil *HV
// (something that isn't a hash) panics, as Perl dies.
func (h *HV) Push(kv ...any) *HV {
	if h == nil {
		panic("scalar: Not a HASH reference")
	}
	flat := flatten(kv)
	for i := 0; i < len(flat); i += 2 {
		var v *SV
		if i+1 < len(flat) {
			v = flat[i+1]
		}
		h.Get(flat[i].Str()).Set(v)
	}
	return h
}

// Exists is Perl's exists $h{key}: whether key is there, even if its value
// is undef. More keys go on from there as a scalar's Exists takes them, so
// h.Exists("a", "b") is exists $h{a}{b}, which creates $h{a} on the way, as
// Perl does.
func (h *HV) Exists(key string, more ...any) bool {
	if len(more) > 0 {
		return h.get(key).Exists(more...)
	}
	if h == nil {
		return false
	}
	_, ok := h.m[key]
	return ok
}

// Defined is Perl's defined $h{key}: whether key exists and its value isn't
// undef. Unlike h.Get(key).Defined(), it never creates a placeholder.
func (h *HV) Defined(key string) bool {
	if h == nil {
		return false
	}
	return h.m[key].Defined() // a missing key is a nil *SV, which isn't defined
}

// Delete is Perl's delete $h{key}: it removes key and returns its value, or
// undef. More keys go on from there as a scalar's Delete takes them, so
// h.Delete("a", "b") is delete $h{a}{b}, which creates $h{a} on the way,
// as Perl does.
func (h *HV) Delete(key string, more ...any) *SV {
	if len(more) > 0 {
		return h.get(key).Delete(more...)
	}
	if h != nil && h.locked {
		what := "disallowed"
		if _, ok := h.m[key]; ok {
			what = "readonly"
		}
		panic(fmt.Sprintf("scalar: Attempt to delete %s key '%s' from a restricted hash", what, key))
	}
	if h == nil {
		return &SV{}
	}
	s, ok := h.m[key]
	if !ok {
		return &SV{}
	}
	delete(h.m, key)
	h.order = nil
	return s
}

// Undef is Perl's undef %h: it empties h in place, so everything referring
// to h sees it empty, and returns h, now empty. Each starts again from the
// beginning. A nil h panics, as Perl dies on undef %$x for a non-hash.
func (h *HV) Undef() *HV {
	if h != nil && h.locked && len(h.m) > 0 {
		// Perl names whichever key it meets first; the first in order is
		// the same every time.
		panic(fmt.Sprintf("scalar: Attempt to delete readonly key '%s' from a restricted hash", slices.Min(h.keys())))
	}
	if h == nil {
		panic("scalar: Not a HASH reference")
	}
	if h.owner != nil {
		h.attach() // as undef %{$s->{a}} creates the hash, even under no autovivification
	}
	h.m, h.order, h.eachKeys, h.eachPos = map[string]*SV{}, nil, nil, 0
	return h
}

func (h *HV) keys() []string {
	if h == nil {
		return nil
	}
	if h.order == nil && len(h.m) > 0 {
		h.order = make([]string, 0, len(h.m))
		for k := range h.m {
			h.order = append(h.order, k)
		}
	}
	return h.order
}

// Keys is Perl's keys %h: a new array of the keys. Changing it doesn't
// change h. As in Perl, it resets Each.
func (h *HV) Keys() *AV {
	h.resetEach()
	ks := h.keys()
	out := &AV{items: make([]*SV, len(ks))}
	for i, k := range ks {
		out.items[i] = &SV{v: k}
	}
	return out
}

// Values is Perl's values %h: the values, in the same order as Keys. Given
// keys, it's the hash slice @h{k1, k2, ...} instead: the values for those
// keys, in that order. The keys may be strings, scalars or arrays of them,
// flattened as a list is (see AV.Push), so h.Values(names) is @h{@names}.
//
// As in Perl, the elements are the hash's own values, not copies: changing
// one (with Set, Inc and so on) changes h. A missing key reads as undef and
// isn't created unless you write to it, as with Get. Adding or removing
// elements of the array doesn't change h; use Copy on an element for a
// value of your own.
func (h *HV) Values(keys ...any) *AV {
	if len(keys) == 0 {
		return &AV{items: h.ValuesSlice()} // which resets Each, as values %h does
	}
	names := flatten(keys)
	out := &AV{items: make([]*SV, len(names))}
	for i, k := range names {
		out.items[i] = h.Get(k.Str())
	}
	return out
}

// Pairs is Perl's %h in list context: a new array of key/value pairs, k1,
// v1, k2, v2 and so on, in the same order as Keys. Given keys, it's the
// key/value slice %h{k1, k2, ...} instead: the pairs for those keys, in that
// order, with keys given as for Values. A missing key is included, with an
// undef value. To make a hash of them, as %h1 = %h2{...} does, pass the
// result to Hash:
//
//	h1 := scalar.Hash(h2.Pairs("k1", "k2", "k3"))
//
// As in Perl, the keys are new strings and the values are the hash's own,
// as with Values. Note that List::Util's pairs is different: it returns
// [key, value] array references, where this returns one flat list.
func (h *HV) Pairs(keys ...any) *AV {
	var names []string
	if len(keys) == 0 {
		h.resetEach() // as %h in list context does
		names = h.keys()
	} else {
		for _, k := range flatten(keys) {
			names = append(names, k.Str())
		}
	}
	out := &AV{items: make([]*SV, 0, 2*len(names))}
	for _, k := range names {
		out.items = append(out.items, &SV{v: k}, h.Get(k))
	}
	return out
}

// KeysSlice is Keys as a Go []string. Like Keys, it resets Each.
func (h *HV) KeysSlice() []string {
	h.resetEach()
	return slices.Clone(h.keys())
}

// ValuesSlice is Values() as a Go []*SV. Like Values, the elements are the
// hash's own values, and it resets Each.
func (h *HV) ValuesSlice() []*SV {
	h.resetEach()
	ks := h.keys()
	vs := make([]*SV, len(ks))
	for i, k := range ks {
		vs[i] = h.m[k]
	}
	return vs
}

// Each is Perl's each %h: the next key/value pair, with ok true, or ok false
// once every pair has been returned, after which it starts again from the
// beginning. The value is the hash's own, as in Perl:
//
//	for k, v, ok := h.Each(); ok; k, v, ok = h.Each() {   // while (my ($k, $v) = each %h)
//		...
//	}
//
// As in Perl, the hash remembers how far Each has got, so leaving such a
// loop early means the next Each carries on where it stopped; and Keys,
// Values, Pairs (without keys), KeysSlice, ValuesSlice and PairsMap reset
// it, as keys, values and %h in list context do. Deleting keys along the
// way is safe: they're skipped. Keys added along the way aren't returned
// until the next pass. For a loop without any of this, use All.
func (h *HV) Each() (key string, value *SV, ok bool) {
	if h == nil {
		return "", nil, false
	}
	if h.eachKeys == nil {
		h.eachKeys = h.keys() // replaced, never changed, when keys come and go
		h.eachPos = 0
	}
	for h.eachPos < len(h.eachKeys) {
		k := h.eachKeys[h.eachPos]
		h.eachPos++
		if s, exists := h.m[k]; exists {
			return k, s, true
		}
	}
	h.resetEach()
	return "", nil, false
}

// resetEach starts Each again from the beginning.
func (h *HV) resetEach() {
	// Only when Each has begun, so that reading a locked hash writes
	// nothing, and goroutines can read it at once.
	if h != nil && (h.eachKeys != nil || h.eachPos != 0) {
		h.eachKeys, h.eachPos = nil, 0
	}
}

// All iterates over key/value pairs in the same order as Keys. Unlike Each,
// every loop starts from the beginning, stopping early is clean, and it
// doesn't change where Each has got to. Keys added
// during iteration aren't visited; keys deleted during iteration are skipped.
func (h *HV) All() iter.Seq2[string, *SV] {
	return func(yield func(string, *SV) bool) {
		// Adding or deleting a key replaces h.order rather than changing it,
		// so ranging over the current slice is safe without a copy.
		for _, k := range h.keys() {
			s, ok := h.m[k]
			if !ok {
				continue // deleted during iteration
			}
			if !yield(k, s) {
				return
			}
		}
	}
}

// Map builds a new hash from h, as %new = map { f($_, $h{$_}) } keys %h
// does: f gets each key and value, and returns the new key and value. Values
// are copied in, as by Set, so h is left alone. If f gives two keys the same
// new key, one value survives, but which one depends on hash order, as in
// Perl. A nil h reads as empty. AV.Map is the array version.
func (h *HV) Map(f func(k string, v *SV) (string, any)) *HV {
	out := newHV(h.Length())
	for k, v := range h.All() {
		out.Set(f(k, v))
	}
	return out
}

// PairsMap returns h as a Go map from its keys to its own values, as
// Pairs() would list them, for code that doesn't use this package. The map
// is new, so adding or removing keys doesn't change h, but changing a value
// does. For plain Go values throughout, use Any on a scalar holding h.
func (h *HV) PairsMap() map[string]*SV {
	h.resetEach() // as %h in list context does
	out := make(map[string]*SV, h.Length())
	for k, v := range h.All() {
		out[k] = v
	}
	return out
}
