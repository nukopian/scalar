package scalar

import (
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
}

// Hash builds a hash from a key/value list, as %h = (k1 => v1, ...) does.
// The list is flattened first, as by AV.Push, so Hash(defaults, "k", "v")
// merges like %h = (%defaults, k => "v"), with later keys winning. Keys are
// stringified; a trailing key without a value maps to undef.
func Hash(kv ...any) *HV {
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

// Get returns the value for key. A missing key reads as undef without being
// created; writing through the returned SV (Set, or an autovivifying AV/HV
// deref) creates it, so h.Get("a").HV().Set("b", 1) is $h{a}{b} = 1.
//
// Get on a nil *HV (something that wasn't a hash) returns undef, but
// writing to it panics, as Perl dies with "Not a HASH reference".
func (h *HV) Get(key string) *SV {
	if h == nil {
		return &SV{noCreate: "Not a HASH reference"}
	}
	if s, ok := h.m[key]; ok {
		return s
	}
	return &SV{viv: func(s *SV) { h.store(key, s) }}
}

func (h *HV) store(key string, s *SV) {
	if h.m == nil {
		h.m = map[string]*SV{}
	}
	if _, ok := h.m[key]; !ok {
		h.order = nil
	}
	h.m[key] = s
}

// Set assigns v to key and returns the stored SV.
func (h *HV) Set(key string, v any) *SV {
	return h.Get(key).Set(v)
}

// Exists is Perl's exists.
func (h *HV) Exists(key string) bool {
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

// Delete is Perl's delete: removes key and returns its value, or undef.
func (h *HV) Delete(key string) *SV {
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
// change h.
func (h *HV) Keys() *AV {
	ks := h.keys()
	out := &AV{items: make([]*SV, len(ks))}
	for i, k := range ks {
		out.items[i] = &SV{v: k}
	}
	return out
}

// Values is Perl's values %h, in the same order as Keys. As in Perl, the
// elements are the hash's own values, not copies: changing one (with Set,
// Inc and so on) changes h. Adding or removing elements of the array
// doesn't change h.
func (h *HV) Values() *AV {
	return &AV{items: h.ValuesSlice()}
}

// KeysSlice is Keys as a Go []string.
func (h *HV) KeysSlice() []string {
	return slices.Clone(h.keys())
}

// ValuesSlice is Values as a Go []*SV. Like Values, the elements are the
// hash's own values.
func (h *HV) ValuesSlice() []*SV {
	ks := h.keys()
	vs := make([]*SV, len(ks))
	for i, k := range ks {
		vs[i] = h.m[k]
	}
	return vs
}

// All iterates over key/value pairs in the same order as Keys. Keys added
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
