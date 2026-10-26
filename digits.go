package scalar

import (
	"math"
	"strconv"
	"strings"
)

// Numbers that keep their digits (roadmap 3): a number read as text, from
// JSON, YAML, a dump or a JSONPath literal, keeps its text beside its
// float when the float can't give the number back, so that it's written
// out with every digit: a decimal such as 1.23456789012345678901, or a
// whole number beyond int64. A number of up to 15 significant digits
// always comes back from its float, so ordinary numbers keep nothing and
// cost only a count of their digits. Arithmetic is on the float, and any
// write to the scalar drops the text. The reference's "Numbers that keep
// their digits" says where this differs from Perl.

// keepDigits turns the feature off for the tests that compare the output
// with and without it.
var keepDigits = true

// keptNumber carries a number's float and its text from normalize to the
// scalar that stores them.
type keptNumber struct {
	f    float64
	text string
}

// keepNumber is a number read as text, as a scalar stores it: f, or f
// with its text if f can't give the number back. A whole number written
// as digits always keeps them, as it's only a float here because it's
// beyond int64, and should print as the integer it is, as Perl prints
// one up to 2⁶⁴−1: 10000000000000000000, not 1e+19. text is in a form
// strconv.ParseFloat reads.
func keepNumber(text string, f float64) any {
	if keepDigits && (isIntText(text) || !faithful(text, f)) {
		if isJSONNumber(text) {
			// As written: JSON, YAML and Perl all read it. A copy, so that
			// text doesn't escape, and the readers' numbers that keep
			// nothing don't allocate it.
			return keptNumber{f, strings.Clone(text)}
		}
		if d, ok := parseDecimal(text); ok {
			return keptNumber{f, d.text()}
		}
	}
	return f
}

// isJSONNumber reports whether s is a number as JSON writes one, without
// the + or bare point YAML and Perl allow.
func isJSONNumber(s string) bool {
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	digits := func() int {
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return i - start
	}
	if n := digits(); n == 0 || (n > 1 && s[i-n] == '0') {
		return false // no digits, or a leading zero
	}
	if i < len(s) && s[i] == '.' {
		i++
		if digits() == 0 {
			return false
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		if digits() == 0 {
			return false
		}
	}
	return i == len(s)
}

// faithful reports whether f, read from text, gives the number back: its
// shortest form is the same number. Text that isn't a decimal number,
// such as "0x1F" or ".inf", is left to the float.
func faithful(text string, f float64) bool {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return sigDigits(text) < 0
	}
	n := sigDigits(text)
	if n < 0 {
		return true
	}
	// A float64 gives back any number of up to 15 significant digits in
	// its normal range: the guarantee C calls DBL_DIG.
	if a := math.Abs(f); n <= 15 && (a >= 0x1p-1022 || n == 0) {
		return true
	}
	// A float's shortest form has at most 17 significant digits, so a
	// number with more can't be one.
	if n > 17 {
		return false
	}
	d, ok := parseDecimal(text)
	if !ok {
		return true
	}
	back, _ := parseDecimal(strconv.FormatFloat(f, 'g', -1, 64))
	return d == back
}

// sigDigits counts the significant digits of a decimal number's text,
// from its first nonzero digit to its last, without allocating: 0 for
// zero, and -1 if it isn't a decimal number.
func sigDigits(s string) int {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	first, last, pos := -1, -1, 0
	seen := false
	for ; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			continue
		}
		if c == 'e' || c == 'E' {
			break
		}
		if c < '0' || c > '9' {
			return -1
		}
		seen = true
		if c != '0' {
			if first < 0 {
				first = pos
			}
			last = pos
		}
		pos++
	}
	switch {
	case !seen:
		return -1
	case first < 0:
		return 0
	}
	return last - first + 1
}

// decimal is a number's exact value: 0.digits × 10^point, with no leading
// or trailing zeros in digits, which is "" for zero.
type decimal struct {
	neg    bool
	digits string
	point  int
}

// parseDecimal reads a decimal number: a sign, digits with perhaps a
// point, and perhaps an exponent, as JSON, YAML and Perl write them.
func parseDecimal(s string) (decimal, bool) {
	var d decimal
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		d.neg = s[i] == '-'
		i++
	}
	var b strings.Builder
	b.Grow(len(s)) // one allocation, not one per doubling
	intDigits, digits, dot := 0, 0, false
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && !dot:
			dot = true
			continue
		case c >= '0' && c <= '9':
			b.WriteByte(c)
			digits++
			if !dot {
				intDigits++
			}
			continue
		}
		break
	}
	if digits == 0 {
		return decimal{}, false
	}
	exp := 0
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		eneg := false
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			eneg = s[i] == '-'
			i++
		}
		if i == len(s) {
			return decimal{}, false
		}
		for ; i < len(s); i++ {
			c := s[i]
			if c < '0' || c > '9' {
				return decimal{}, false
			}
			if exp < 1<<30 {
				exp = exp*10 + int(c-'0')
			}
		}
		if eneg {
			exp = -exp
		}
	}
	if i != len(s) {
		return decimal{}, false
	}
	all := b.String()
	lead := 0
	for lead < len(all) && all[lead] == '0' {
		lead++
	}
	all = strings.TrimRight(all[lead:], "0")
	if all == "" {
		return decimal{}, true // zero, whatever its sign
	}
	d.digits = all
	d.point = intDigits - lead + exp
	return d, true
}

// text writes d as a number JSON, YAML and Perl all read: plainly from
// 1e-6 to 1e21, as JavaScript does, and with an exponent beyond.
func (d decimal) text() string {
	if d.digits == "" {
		return "0"
	}
	var b strings.Builder
	if d.neg {
		b.WriteByte('-')
	}
	n, p := len(d.digits), d.point
	switch {
	case p > 0 && p <= 21:
		if n <= p {
			b.WriteString(d.digits)
			b.WriteString(strings.Repeat("0", p-n))
		} else {
			b.WriteString(d.digits[:p])
			b.WriteByte('.')
			b.WriteString(d.digits[p:])
		}
	case p <= 0 && p > -6:
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -p))
		b.WriteString(d.digits)
	default:
		b.WriteByte(d.digits[0])
		if n > 1 {
			b.WriteByte('.')
			b.WriteString(d.digits[1:])
		}
		b.WriteByte('e')
		b.WriteString(strconv.Itoa(p - 1))
	}
	return b.String()
}

// whole reports whether d is a whole number.
func (d decimal) whole() bool { return d.point >= len(d.digits) }

// cmp compares a and b as numbers.
func (a decimal) cmp(b decimal) int {
	an, bn := a.neg && a.digits != "", b.neg && b.digits != ""
	if an != bn {
		if an {
			return -1
		}
		return 1
	}
	c := a.cmpMagnitude(b)
	if an {
		return -c
	}
	return c
}

func (a decimal) cmpMagnitude(b decimal) int {
	switch {
	case a.digits == "" && b.digits == "":
		return 0
	case a.digits == "":
		return -1
	case b.digits == "":
		return 1
	case a.point != b.point:
		if a.point < b.point {
			return -1
		}
		return 1
	}
	return strings.Compare(a.digits, b.digits) // no trailing zeros, so a prefix is smaller
}

// digits is the text kept beside s's float, or "".
func (s *SV) digits() string {
	if s == nil || s.rare == nil {
		return ""
	}
	return s.rare.digits
}

// setValue stores x, normalize's result, in s, with a number's text if it
// carries one.
func (s *SV) setValue(x any) {
	if k, ok := x.(keptNumber); ok {
		s.v = k.f
		if s.rare == nil {
			s.rare = &svRare{}
		}
		s.rare.digits = k.text
		s.rare.dec = nil
		return
	}
	s.v = x
}

// svOf is a new scalar holding x, normalize's result.
func svOf(x any) *SV {
	s := &SV{}
	s.setValue(x)
	return s
}

// copyDigits gives c, a new copy of s's value, s's kept text directly,
// without the carrier normalize uses, and shares its parsed digits, which
// never change once parsed. c gets its own svRare, so a change to c
// leaves s as it was.
func (s *SV) copyDigits(c *SV) {
	if t := s.digits(); t != "" {
		c.rare = &svRare{digits: t, dec: s.rare.dec}
	}
}

// packed is s's value as normalize gives it: with its text, if it keeps
// one.
func (s *SV) packed() any {
	if t := s.digits(); t != "" {
		return keptNumber{s.v.(float64), t}
	}
	return s.val()
}

// exactDecimal is s as an exact decimal, if it's one that compares by its
// digits: an integer, a number that keeps its text, or a string of a
// number its float can't give back. ok is false for anything else,
// floats included, which compare as floats, as in Perl.
func (s *SV) exactDecimal() (d decimal, kept, ok bool) {
	switch v := s.val().(type) {
	case int64:
		d, _ = parseDecimal(strconv.FormatInt(v, 10))
		return d, false, true
	case float64:
		if t := s.digits(); t != "" {
			if s.rare.dec != nil {
				return *s.rare.dec, true, true
			}
			d, _ = parseDecimal(t)
			if !s.ro {
				// Parsed once, as a sort compares it many times; not when
				// locked, as a locked scalar may be read concurrently. A
				// new decimal, as &d would move d to the heap on every call.
				cached := d
				s.rare.dec = &cached
			}
			return d, true, true
		}
	case string:
		return digitString(v)
	case []byte:
		return digitString(string(v))
	}
	return decimal{}, false, false
}

// digitString is a string that's wholly a number its float can't give
// back, as an exact decimal.
func digitString(v string) (decimal, bool, bool) {
	if len(v) < 16 {
		return decimal{}, false, false // 15 digits or fewer: its float does
	}
	_, f, _, ok := parseNumber(v)
	t := strings.Trim(v, perlSpace)
	if !ok || faithful(t, f) {
		return decimal{}, false, false
	}
	d, ok := parseDecimal(t)
	return d, ok, ok
}

// exactCompare compares a and b by their digits when either keeps its
// text or is a string of digits its float can't hold, and both are exact
// decimals; ok is false otherwise. numCompare asks only when their floats
// are equal: rounding to a float never reverses an order, so floats that
// differ already give the exact answer.
func exactCompare(a, b *SV) (c int, ok bool) {
	if !mayKeep(a) && !mayKeep(b) {
		return 0, false
	}
	da, ak, aok := a.exactDecimal()
	db, bk, bok := b.exactDecimal()
	if !aok || !bok || (!ak && !bk) {
		return 0, false
	}
	return da.cmp(db), true
}

// mayKeep is a quick check that s could compare by its digits: it keeps
// text, or it's a string long enough to need it.
func mayKeep(s *SV) bool {
	switch v := s.val().(type) {
	case float64:
		return s.digits() != ""
	case string:
		return len(v) >= 16
	case []byte:
		return len(v) >= 16
	}
	return false
}

// numberText is a number as exact text: the digits it keeps, an integer's,
// or the shortest form that gives its float back, not the 15 digits Str
// gives, as Perl prints it.
func numberText(s *SV) string {
	switch v := s.val().(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if t := s.digits(); t != "" {
			return t
		}
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	return s.Str()
}

// keptWhole is s as a uint64, when it keeps a whole number's text, or is
// a string of digits, beyond int64.
func (s *SV) keptWhole() (uint64, bool) {
	d, kept, ok := s.exactDecimal()
	if !ok || !kept || d.neg || !d.whole() {
		return 0, false
	}
	u, err := strconv.ParseUint(d.text(), 10, 64)
	return u, err == nil
}
