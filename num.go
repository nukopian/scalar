package scalar

import (
	"regexp"
	"strconv"
	"strings"
)

// numPrefix matches the longest numeric prefix Perl would accept.
var numPrefix = regexp.MustCompile(`^[+-]?(?:(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?|(?i:inf(?:inity)?|nan))`)

const perlSpace = " \t\n\r\f\v"

// parseNumber numifies a string the way Perl does. Leading and trailing
// whitespace is allowed; anything else after the number makes ok false but
// still yields the prefix's value. isInt reports whether iv is exact.
func parseNumber(s string) (iv int64, nv float64, isInt, ok bool) {
	if s == "0 but true" {
		return 0, 0, true, true
	}
	t := strings.TrimLeft(s, perlSpace)
	m := numPrefix.FindString(t)
	if m == "" {
		return 0, 0, true, false
	}
	ok = strings.TrimRight(t[len(m):], perlSpace) == ""

	if i, err := strconv.ParseInt(m, 10, 64); err == nil {
		return i, float64(i), true, ok
	}
	f, _ := strconv.ParseFloat(m, 64) // out-of-range yields ±Inf, as in Perl
	return 0, f, false, ok
}
