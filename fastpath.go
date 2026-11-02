package scalar

import "strings"

// fastPaths turns on shortcuts for the commonest calls, which give the same
// results as the general code more cheaply. Tests turn them off to check
// that.
var fastPaths = true

// plainHash builds Hash's hash directly, for the usual call, Hash("k", v,
// ...): string keys and values that don't flatten. The general code would
// make a scalar of every key only to turn it back into a string. It reports
// false for any other list.
func plainHash(kv []any) (*HV, bool) {
	if !fastPaths {
		return nil, false
	}
	copies := 0
	for i, x := range kv {
		if i%2 == 0 {
			if _, ok := x.(string); !ok {
				return nil, false
			}
			continue
		}
		if !plainValue(x) {
			return nil, false
		}
		if _, ok := x.(*SV); ok {
			copies++
		}
	}
	n := (len(kv) + 1) / 2
	h := newHV(n)
	block := make([]SV, n-copies)
	for i := 0; i < len(kv); i += 2 {
		var x any
		if i+1 < len(kv) {
			x = kv[i+1]
		}
		h.m[kv[i].(string)] = plainCopy(x, &block)
	}
	return h, true
}

// plainCopy is Value(x) for a plain value x, taking the new scalar from
// the front of block, so that a list's new scalars are made in one
// allocation, not one each. That keeps them all alive while any one is
// referenced: a hash or array built from a list is usually kept or
// dropped whole. A *SV is copied with its digits, as Value copies it,
// and takes nothing from block.
func plainCopy(x any, block *[]SV) *SV {
	if s, ok := x.(*SV); ok {
		return Value(s)
	}
	s := &(*block)[0]
	*block = (*block)[1:]
	s.setValue(normalize(x))
	return s
}

// appendPlain appends copies of vals to items, as flatten would, for vals
// that plainValues has passed, with the new scalars in one allocation.
func appendPlain(items []*SV, vals []any) []*SV {
	if len(vals) == 1 { // a block of one saves nothing, and costs a call
		return append(items, Value(vals[0]))
	}
	copies := 0
	for _, v := range vals {
		if _, ok := v.(*SV); ok {
			copies++
		}
	}
	block := make([]SV, len(vals)-copies)
	for _, v := range vals {
		items = append(items, plainCopy(v, &block))
	}
	return items
}

// setNew is Set for a key h doesn't have yet, storing a new scalar
// directly, without the placeholder Get makes for a missing key and the
// closure it vivifies through. As Set's general code does, it stores the
// scalar before converting v, so a conversion that panics leaves the key
// there, holding undef. It reports false, having done nothing, for a key h
// already has, which must keep its scalar, for a locked or nil hash,
// whose panics the general code gives, and for a hash with
// autovivification off, whose setting the general code passes on.
func (h *HV) setNew(key string, v any) (*SV, bool) {
	if !fastPaths || h == nil || h.locked || h.noViv || h.noVivDeep {
		return nil, false
	}
	if _, ok := h.m[key]; ok {
		return nil, false
	}
	s := &SV{}
	h.store(key, s)
	s.setValue(normalize(v))
	return s, true
}

// plainValues reports whether no value in vals flattens, so that Array and
// Push can append copies of them directly, as flatten would, without the
// list it builds.
func plainValues(vals []any) bool {
	if !fastPaths {
		return false
	}
	for _, v := range vals {
		if !plainValue(v) {
			return false
		}
	}
	return true
}

// plainValue reports whether flatten adds v as one value, a copy of it, as
// it does a scalar, a string, a bool, the commonest numbers and a []byte (one
// byte string). The fast paths rely on this, so it's the one list of them.
func plainValue(v any) bool {
	switch v.(type) {
	case nil, *SV, string, bool, int, int64, float64, []byte:
		return true
	}
	return false
}

// literalPattern reports whether pattern matches only itself, so that S can
// replace it with the strings package instead of a regex. The flags that
// would change that (i and x) are checked by the caller.
func literalPattern(pattern string) bool {
	return pattern != "" && !strings.ContainsAny(pattern, `\.+*?()|[]{}^$`)
}
