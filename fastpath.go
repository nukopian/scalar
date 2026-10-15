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
	}
	h := newHV((len(kv) + 1) / 2)
	for i := 0; i < len(kv); i += 2 {
		v := &SV{}
		if i+1 < len(kv) {
			v = Value(kv[i+1])
		}
		h.m[kv[i].(string)] = v
	}
	return h, true
}

// plainValues reports whether no value in vals flattens, so that Push can
// append copies of them directly, as flatten would, without the list it
// builds.
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
