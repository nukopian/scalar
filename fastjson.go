package scalar

import (
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// readJSON reads data as JSON straight into a scalar, for FromJSON and
// UnmarshalJSON. encoding/json would decode it into Go's maps and slices
// first, which Set then copies into hashes and arrays: twice the work, and
// most of the allocations.
//
// It takes only the JSON it can be sure of, and reports false for anything
// else: every error, a string with bytes that aren't UTF-8 or a lone
// surrogate escape, and nesting deeper than encoding/json allows. The
// caller then uses encoding/json, so results and error messages are
// encoding/json's either way.
func readJSON(data []byte) (*SV, bool) {
	if !fastPaths {
		return nil, false
	}
	r := &jsonReader{data: data}
	r.space()
	v, ok := r.value()
	if !ok {
		return nil, false
	}
	r.space()
	if r.i != len(data) {
		return nil, false // trailing data: encoding/json says what's wrong
	}
	return svOf(v), true
}

// jsonMaxDepth is encoding/json's limit on nesting, past which it fails.
const jsonMaxDepth = 10000

type jsonReader struct {
	data  []byte
	i     int
	depth int
}

func (r *jsonReader) space() {
	for r.i < len(r.data) {
		switch r.data[r.i] {
		case ' ', '\t', '\n', '\r':
			r.i++
		default:
			return
		}
	}
}

// value reads one value, as Set would store it: nil, a bool, an int64 or
// float64, a string, an *AV or an *HV.
func (r *jsonReader) value() (any, bool) {
	if r.i >= len(r.data) {
		return nil, false
	}
	switch c := r.data[r.i]; {
	case c == '{':
		return r.object()
	case c == '[':
		return r.array()
	case c == '"':
		return r.str()
	case c == 't':
		return true, r.literal("true")
	case c == 'f':
		return false, r.literal("false")
	case c == 'n':
		return nil, r.literal("null")
	case c == '-' || c >= '0' && c <= '9':
		return r.number()
	}
	return nil, false
}

func (r *jsonReader) literal(word string) bool {
	end := r.i + len(word)
	if end > len(r.data) || string(r.data[r.i:end]) != word { // compared without a copy
		return false
	}
	r.i += len(word)
	return true
}

func (r *jsonReader) object() (any, bool) {
	if r.depth++; r.depth >= jsonMaxDepth {
		return nil, false
	}
	defer func() { r.depth-- }()
	r.i++ // {
	h := newHV(0)
	r.space()
	if r.i < len(r.data) && r.data[r.i] == '}' {
		r.i++
		return h, true
	}
	for {
		r.space()
		if r.i >= len(r.data) || r.data[r.i] != '"' {
			return nil, false
		}
		k, ok := r.str()
		if !ok {
			return nil, false
		}
		r.space()
		if r.i >= len(r.data) || r.data[r.i] != ':' {
			return nil, false
		}
		r.i++
		r.space()
		v, ok := r.value()
		if !ok {
			return nil, false
		}
		h.m[k.(string)] = svOf(v) // a repeated key: the last wins, as in encoding/json
		r.space()
		if r.i >= len(r.data) {
			return nil, false
		}
		switch r.data[r.i] {
		case ',':
			r.i++
		case '}':
			r.i++
			return h, true
		default:
			return nil, false
		}
	}
}

func (r *jsonReader) array() (any, bool) {
	if r.depth++; r.depth >= jsonMaxDepth {
		return nil, false
	}
	defer func() { r.depth-- }()
	r.i++ // [
	a := &AV{items: []*SV{}}
	r.space()
	if r.i < len(r.data) && r.data[r.i] == ']' {
		r.i++
		return a, true
	}
	for {
		r.space()
		v, ok := r.value()
		if !ok {
			return nil, false
		}
		a.items = append(a.items, svOf(v))
		r.space()
		if r.i >= len(r.data) {
			return nil, false
		}
		switch r.data[r.i] {
		case ',':
			r.i++
		case ']':
			r.i++
			return a, true
		default:
			return nil, false
		}
	}
}

// str reads a string. One with no escapes is a single copy of its bytes;
// escapes are decoded as encoding/json decodes them, except a lone
// surrogate, which it replaces and this leaves to it.
func (r *jsonReader) str() (any, bool) {
	start := r.i + 1
	for j := start; j < len(r.data); j++ {
		switch c := r.data[j]; {
		case c == '"':
			raw := r.data[start:j]
			if !utf8.Valid(raw) {
				return nil, false // encoding/json replaces bad bytes; let it
			}
			r.i = j + 1
			return string(raw), true
		case c == '\\':
			return r.escapedStr(start)
		case c < 0x20:
			return nil, false
		}
	}
	return nil, false // unterminated
}

func (r *jsonReader) escapedStr(start int) (any, bool) {
	var b strings.Builder
	j := start
	for j < len(r.data) {
		c := r.data[j]
		switch {
		case c == '"':
			if !utf8.ValidString(b.String()) {
				return nil, false
			}
			r.i = j + 1
			return b.String(), true
		case c < 0x20:
			return nil, false
		case c != '\\':
			b.WriteByte(c)
			j++
			continue
		}
		if j+1 >= len(r.data) {
			return nil, false
		}
		switch e := r.data[j+1]; e {
		case '"', '\\', '/':
			b.WriteByte(e)
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			r1, ok := hex4(r.data, j+2)
			if !ok {
				return nil, false
			}
			j += 6
			if utf16.IsSurrogate(r1) {
				// A pair: 😀. Anything else is a lone surrogate.
				if j+1 >= len(r.data) || r.data[j] != '\\' || r.data[j+1] != 'u' {
					return nil, false
				}
				r2, ok := hex4(r.data, j+2)
				if !ok {
					return nil, false
				}
				combined := utf16.DecodeRune(r1, r2)
				if combined == utf8.RuneError {
					return nil, false
				}
				r1 = combined
				j += 6
			}
			b.WriteRune(r1)
			continue
		default:
			return nil, false
		}
		j += 2
	}
	return nil, false // unterminated
}

func hex4(data []byte, at int) (rune, bool) {
	if at+4 > len(data) {
		return 0, false
	}
	n, err := strconv.ParseUint(string(data[at:at+4]), 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(n), true
}

// number reads a number as JSON's grammar allows, and converts it as
// FromJSON always has: an integer that fits in int64 as an IV, anything
// else as an NV.
func (r *jsonReader) number() (any, bool) {
	start := r.i
	d := r.data
	i := r.i
	if i < len(d) && d[i] == '-' {
		i++
	}
	switch {
	case i < len(d) && d[i] == '0':
		i++
	case i < len(d) && d[i] >= '1' && d[i] <= '9':
		for i < len(d) && d[i] >= '0' && d[i] <= '9' {
			i++
		}
	default:
		return nil, false
	}
	integer := true
	if i < len(d) && d[i] == '.' {
		integer = false
		i++
		if i >= len(d) || d[i] < '0' || d[i] > '9' {
			return nil, false
		}
		for i < len(d) && d[i] >= '0' && d[i] <= '9' {
			i++
		}
	}
	if i < len(d) && (d[i] == 'e' || d[i] == 'E') {
		integer = false
		i++
		if i < len(d) && (d[i] == '+' || d[i] == '-') {
			i++
		}
		if i >= len(d) || d[i] < '0' || d[i] > '9' {
			return nil, false
		}
		for i < len(d) && d[i] >= '0' && d[i] <= '9' {
			i++
		}
	}
	r.i = i
	text := d[start:i]
	if integer && len(text) <= 18 {
		// Fits in int64 for certain: no string, no parsing library.
		var n int64
		neg := text[0] == '-'
		for _, c := range text {
			if c != '-' {
				n = n*10 + int64(c-'0')
			}
		}
		if neg {
			n = -n
		}
		return n, true
	}
	return normalizeJSONNumber(jsonNumber(text)), true
}
