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

// Hash builds a hash from a flat key/value list, as %h = (k1 => v1, ...)
// does. Keys are stringified; a trailing key without a value maps to undef.
func Hash(kv ...any) *HV {
	h := &HV{}
	for i := 0; i < len(kv); i += 2 {
		var v any
		if i+1 < len(kv) {
			v = kv[i+1]
		}
		h.Set(Scalar(kv[i]).Str(), v)
	}
	return h
}

// Len is Perl's scalar(keys %h).
func (h *HV) Len() int {
	if h == nil {
		return 0
	}
	return len(h.m)
}

// Get returns the value for key. A missing key reads as undef without being
// created; writing through the returned SV (Set, or an autovivifying AV/HV
// deref) creates it, so h.Get("a").HV().Set("b", 1) is $h{a}{b} = 1.
func (h *HV) Get(key string) *SV {
	if h == nil {
		return &SV{}
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
	if h == nil {
		panic("scalar: Set on nil *HV (not a HASH reference)")
	}
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

// Keys is Perl's keys.
func (h *HV) Keys() []string {
	return slices.Clone(h.keys())
}

// Values is Perl's values, in the same order as Keys.
func (h *HV) Values() []*SV {
	ks := h.keys()
	vs := make([]*SV, len(ks))
	for i, k := range ks {
		vs[i] = h.m[k]
	}
	return vs
}

// All iterates over key/value pairs in the same order as Keys.
func (h *HV) All() iter.Seq2[string, *SV] {
	return func(yield func(string, *SV) bool) {
		for _, k := range slices.Clone(h.keys()) {
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
