package scalar

import (
	"strconv"
	"strings"
)

// numericPrefix returns the longest numeric prefix of t Perl would accept,
// or "": a sign, then digits with a fraction or a fraction alone, then an
// exponent if digits follow it; or inf, infinity or nan, in any case. It's
// the regex ^[+-]?(?:(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?|(?i:inf(?:inity)?|nan)),
// written out, as the regex was most of a numeric read's time
// (TestNumericPrefix and FuzzNumericPrefix check them against each other).
func numericPrefix(t string) string {
	i := 0
	if i < len(t) && (t[i] == '+' || t[i] == '-') {
		i++
	}
	start := i
	for i < len(t) && isASCIIDigit(t[i]) {
		i++
	}
	whole := i > start
	if i < len(t) && t[i] == '.' {
		j := i + 1
		for j < len(t) && isASCIIDigit(t[j]) {
			j++
		}
		if whole || j > i+1 { // "4." and ".5", not "."
			i, whole = j, true
		}
	}
	if !whole {
		rest := t[start:]
		switch {
		case hasPrefixFold(rest, "infinity"):
			return t[:start+8]
		case hasPrefixFold(rest, "inf"), hasPrefixFold(rest, "nan"):
			return t[:start+3]
		}
		return ""
	}
	if i < len(t) && (t[i] == 'e' || t[i] == 'E') {
		j := i + 1
		if j < len(t) && (t[j] == '+' || t[j] == '-') {
			j++
		}
		k := j
		for k < len(t) && isASCIIDigit(t[k]) {
			k++
		}
		if k > j {
			i = k
		}
	}
	return t[:i]
}

func isASCIIDigit(b byte) bool { return '0' <= b && b <= '9' }

// hasPrefixFold reports whether t starts with prefix, a lower-case ASCII
// word, in any case. strings.ToLower would allocate for every capital.
func hasPrefixFold(t, prefix string) bool {
	if len(t) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if t[i]|0x20 != prefix[i] {
			return false
		}
	}
	return true
}

// isIntText reports whether m, a numeric prefix, is a sign and digits
// only, so that strconv.ParseInt is worth trying: a failed one allocates.
func isIntText(m string) bool {
	if m != "" && (m[0] == '+' || m[0] == '-') {
		m = m[1:]
	}
	for i := 0; i < len(m); i++ {
		if !isASCIIDigit(m[i]) {
			return false
		}
	}
	return m != ""
}

const perlSpace = " \t\n\r\f\v"

// parseNumber numifies a string the way Perl does. Leading and trailing
// whitespace is allowed; anything else after the number makes ok false but
// still yields the prefix's value. isInt reports whether iv is exact.
func parseNumber(s string) (iv int64, nv float64, isInt, ok bool) {
	if s == "0 but true" {
		return 0, 0, true, true
	}
	t := strings.TrimLeft(s, perlSpace)
	m := numericPrefix(t)
	if m == "" {
		return 0, 0, true, false
	}
	ok = strings.TrimRight(t[len(m):], perlSpace) == ""

	if isIntText(m) {
		if i, err := strconv.ParseInt(m, 10, 64); err == nil {
			return i, float64(i), true, ok
		}
	}
	f, _ := strconv.ParseFloat(m, 64) // out-of-range yields ±Inf, as in Perl
	return 0, f, false, ok
}
