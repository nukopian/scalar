package scalar

import (
	"cmp"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Operators take their right-hand operand as any: a *SV, or a Go value
// converted as by SV.Set. Except for Inc and Dec, they return a new SV and
// leave both operands alone.
//
// Arithmetic stays in int64 while both operands are integers and the result
// fits, and falls back to float64 otherwise, as Perl's IV/NV handling does.
// Non-numeric strings numify silently; check NumOK first if that matters.

func operand(x any) *SV {
	if s, ok := x.(*SV); ok {
		return s
	}
	return Value(x)
}

// numify returns s in numeric context; isInt reports whether iv is exact.
func (s *SV) numify() (iv int64, nv float64, isInt bool) {
	switch v := s.val().(type) {
	case nil:
		return 0, 0, true
	case bool:
		if v {
			return 1, 1, true
		}
		return 0, 0, true
	case int64:
		return v, float64(v), true
	case float64:
		return 0, v, false
	case string:
		iv, nv, isInt, _ = parseNumber(v)
		return iv, nv, isInt
	case []byte:
		iv, nv, isInt, _ = parseNumber(string(v))
		return iv, nv, isInt
	}
	p := refAddr(s.v)
	return p, float64(p), true
}

// Add is Perl's +.
func (s *SV) Add(x any) *SV {
	ai, af, aInt := s.numify()
	bi, bf, bInt := operand(x).numify()
	if aInt && bInt {
		if r, ok := addInt(ai, bi); ok {
			return &SV{v: r}
		}
	}
	return &SV{v: af + bf}
}

// Sub is Perl's -.
func (s *SV) Sub(x any) *SV {
	ai, af, aInt := s.numify()
	bi, bf, bInt := operand(x).numify()
	if aInt && bInt {
		if r, ok := subInt(ai, bi); ok {
			return &SV{v: r}
		}
	}
	return &SV{v: af - bf}
}

// Mul is Perl's *.
func (s *SV) Mul(x any) *SV {
	ai, af, aInt := s.numify()
	bi, bf, bInt := operand(x).numify()
	if aInt && bInt {
		if r, ok := mulInt(ai, bi); ok {
			return &SV{v: r}
		}
	}
	return &SV{v: af * bf}
}

// Div is Perl's /. The result is an integer only when the division is
// exact (7/2 is 3.5, not 3). Dividing by zero panics, as Perl dies.
func (s *SV) Div(x any) *SV {
	ai, af, aInt := s.numify()
	bi, bf, bInt := operand(x).numify()
	if bf == 0 {
		panic("scalar: Illegal division by zero")
	}
	if aInt && bInt && ai%bi == 0 && !(ai == math.MinInt64 && bi == -1) {
		return &SV{v: ai / bi}
	}
	return &SV{v: af / bf}
}

// Mod is Perl's %: both operands are truncated to integers, and a nonzero
// result takes the sign of the right operand (-7 % 3 is 2). As in Perl, a
// whole number up to 2⁶⁴−1 is exact, a float is truncated while it's
// below 2⁶⁴, and once either side is 2⁶⁴ or more the remainder is a
// float's, of the operands rounded to whole numbers. A zero right operand,
// or one that truncates to zero, panics, as Perl dies.
func (s *SV) Mod(x any) *SV { return perlMod(s, operand(x)) }

// modOperand is an operand of % as Perl's pp_modulo takes it: a whole
// number's magnitude and sign, exactly, if it's an integer, or keeps the
// digits of one up to 2⁶⁴−1; otherwise whole is false, and f is its float.
func modOperand(s *SV) (mag uint64, neg, whole bool, f float64) {
	iv, nv, isInt := s.numify()
	switch {
	case isInt && iv < 0:
		return -uint64(iv), true, true, nv
	case isInt:
		return uint64(iv), false, true, nv
	}
	if u, ok := s.keptWhole(); ok {
		return u, false, true, nv
	}
	return 0, false, false, nv
}

// perlMod is Perl's %, as pp_modulo works it out.
func perlMod(a, b *SV) *SV {
	right, rightNeg, rightWhole, dright := modOperand(b)
	useDouble, drightValid := false, false
	if !rightWhole {
		rightNeg = dright < 0
		if rightNeg {
			dright = -dright
		}
		if dright < 0x1p64 {
			right, drightValid = uint64(dright), true
		} else {
			useDouble = true // beyond a uint64, or NaN
		}
	}
	left, leftNeg, leftWhole, dleft := modOperand(a)
	if useDouble || !leftWhole {
		leftNeg = dleft < 0
		if leftNeg {
			dleft = -dleft
		}
		if !useDouble {
			if dleft < 0x1p64 {
				left = uint64(dleft)
			} else {
				// Perl rounds both, then, as 5.6 did.
				dleft = math.Floor(dleft + 0.5)
				useDouble = true
				if drightValid {
					dright = math.Floor(dright + 0.5)
				} else {
					dright = float64(right)
				}
			}
		}
	}
	if useDouble {
		if dright == 0 {
			panic("scalar: Illegal modulus zero")
		}
		ans := math.Mod(dleft, dright)
		if leftNeg != rightNeg && ans != 0 {
			ans = dright - ans
		}
		if rightNeg {
			ans = -ans
		}
		if ans == 0 {
			ans = 0 // not -0, which Perl prints as 0
		}
		return &SV{v: ans}
	}
	if right == 0 {
		panic("scalar: Illegal modulus zero")
	}
	ans := left % right
	if leftNeg != rightNeg && ans != 0 {
		ans = right - ans
	}
	switch {
	case rightNeg && ans <= 1<<63:
		return &SV{v: -int64(ans-1) - 1} // -2⁶³ doesn't negate in int64
	case rightNeg:
		return &SV{v: -float64(ans)}
	case ans <= math.MaxInt64:
		return &SV{v: int64(ans)}
	}
	return svOf(keepNumber(strconv.FormatUint(ans, 10), float64(ans)))
}

// Pow is Perl's **.
func (s *SV) Pow(x any) *SV {
	ai, af, aInt := s.numify()
	bi, bf, bInt := operand(x).numify()
	if aInt && bInt && bi >= 0 {
		if r, ok := powInt(ai, bi); ok {
			return &SV{v: r}
		}
	}
	return &SV{v: math.Pow(af, bf)}
}

// Neg is Perl's unary minus, string madness included: -"foo" is "-foo",
// -"-foo" is "+foo", and -"+foo" is "-foo". Numeric strings negate as numbers.
func (s *SV) Neg() *SV {
	switch v := s.val().(type) {
	case float64:
		return &SV{v: -v} // keeps -0.0, which 0 - v would not
	case string:
		if v == "" {
			break
		}
		switch c := v[0]; {
		case c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
			return &SV{v: "-" + v}
		case c == '+':
			return &SV{v: "-" + v[1:]}
		case c == '-':
			if _, _, _, numeric := parseNumber(v); !numeric {
				return &SV{v: "+" + v[1:]}
			}
		}
	}
	return Value(0).Sub(s)
}

// Concat is Perl's . operator. If either side is a byte string, so is the
// result, joined as characters, as Perl joins them: text characters from 0 to
// 255 become one byte each, and any beyond make the result text (see
// SV.Bytes). To append text's UTF-8 bytes to a byte string instead, append
// Value([]byte(text)).
func (s *SV) Concat(x any) *SV {
	a, aBytes := s.text()
	b, bBytes := operand(x).text()
	return textResult(a+b, aBytes || bBytes)
}

// Repeat is Perl's x operator on a scalar. A count below 1 gives "".
func (s *SV) Repeat(n any) *SV {
	str, isBytes := s.text()
	count := operand(n).Int()
	if count < 1 {
		return textResult("", isBytes)
	}
	return textResult(strings.Repeat(str, int(count)), isBytes)
}

var magicIncRE = regexp.MustCompile(`^[a-zA-Z]*[0-9]*$`)

// Inc is Perl's ++: it modifies s in place and returns it. A non-numeric
// string matching /^[a-zA-Z]*[0-9]*$/ increments as a string, carrying
// within each character class: "aa" → "ab", "Az" → "Ba", "zz" → "aaa",
// "a9" → "b0", "Zz99" → "AAa00". So does a string of digits that a number
// wouldn't give back, as Perl increments every string of digits as a
// string: "007" → "008", and "12345678901234567890" → "12345678901234567891",
// beyond int64. Anything else increments as a number, so "9" → 10, where
// Perl's is the string "10", which prints the same.
func (s *SV) Inc() *SV {
	switch v := s.val().(type) {
	case nil:
		return s.setInt(1) // undef++ is 1
	case int64:
		if v < math.MaxInt64 {
			return s.setInt(v + 1)
		}
	case string:
		if v != "" && magicIncRE.MatchString(v) {
			if _, _, _, numeric := parseNumber(v); !numeric || !incAsNumber(v) {
				return s.Set(magicInc(v))
			}
		}
	}
	return s.Set(s.Add(1))
}

// incAsNumber reports whether digits, incremented as a number, give the
// text Perl's string increment gives: not with a leading zero, which the
// number drops, nor at int64's maximum or beyond, where it's a float.
func incAsNumber(digits string) bool {
	if len(digits) > 1 && digits[0] == '0' {
		return false
	}
	i, err := strconv.ParseInt(digits, 10, 64)
	return err == nil && i < math.MaxInt64
}

// Dec is Perl's --: it modifies s in place and returns it. Unlike Inc it is
// never magical, so "aa" decrements to -1.
func (s *SV) Dec() *SV {
	switch v := s.val().(type) {
	case nil:
		return s.setInt(-1)
	case int64:
		if v > math.MinInt64 {
			return s.setInt(v - 1)
		}
	}
	return s.Set(s.Sub(1))
}

// setInt is Set for the integer fast paths of Inc and Dec, which avoid
// building a temporary result. Like Set, it writes through autovivification.
func (s *SV) setInt(i int64) *SV {
	s.vivify()
	s.v = i
	return s
}

// Assignment operators. Each is s = s OP x, following OP's rules, and
// returns s. Like Set, they write through to a missing array or hash
// element, creating it, so $total{$k} += $n is total.Get(k).AddInPlace(n).
// If OP panics (division by zero), s is left unchanged.

// AddInPlace is Perl's +=.
func (s *SV) AddInPlace(x any) *SV { return s.Set(s.Add(x)) }

// SubInPlace is Perl's -=.
func (s *SV) SubInPlace(x any) *SV { return s.Set(s.Sub(x)) }

// MulInPlace is Perl's *=.
func (s *SV) MulInPlace(x any) *SV { return s.Set(s.Mul(x)) }

// DivInPlace is Perl's /=.
func (s *SV) DivInPlace(x any) *SV { return s.Set(s.Div(x)) }

// ModInPlace is Perl's %=.
func (s *SV) ModInPlace(x any) *SV { return s.Set(s.Mod(x)) }

// PowInPlace is Perl's **=.
func (s *SV) PowInPlace(x any) *SV { return s.Set(s.Pow(x)) }

// ConcatInPlace is Perl's .=.
func (s *SV) ConcatInPlace(x any) *SV { return s.Set(s.Concat(x)) }

// RepeatInPlace is Perl's x=.
func (s *SV) RepeatInPlace(n any) *SV { return s.Set(s.Repeat(n)) }

func magicInc(str string) string {
	b := []byte(str)
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case '9':
			b[i] = '0'
		case 'z':
			b[i] = 'a'
		case 'Z':
			b[i] = 'A'
		default:
			b[i]++
			return string(b)
		}
	}
	// Carried out of the leftmost character: grow by one of its class.
	switch first := str[0]; {
	case first >= '0' && first <= '9':
		return "1" + string(b)
	case first >= 'a' && first <= 'z':
		return "a" + string(b)
	}
	return "A" + string(b)
}

// strPair is s and x as characters, for the string comparisons: a byte
// string compares as its bytes, as Latin-1 characters (see SV.Bytes), so
// "\xe9" as bytes equals "é", as in Perl.
func strPair(s *SV, x any) (string, string) {
	a, _ := s.text()
	b, _ := operand(x).text()
	return a, b
}

// Eq is Perl's eq.
func (s *SV) Eq(x any) bool { a, b := strPair(s, x); return a == b }

// Ne is Perl's ne.
func (s *SV) Ne(x any) bool { a, b := strPair(s, x); return a != b }

// Lt is Perl's lt.
func (s *SV) Lt(x any) bool { a, b := strPair(s, x); return a < b }

// Le is Perl's le.
func (s *SV) Le(x any) bool { a, b := strPair(s, x); return a <= b }

// Gt is Perl's gt.
func (s *SV) Gt(x any) bool { a, b := strPair(s, x); return a > b }

// Ge is Perl's ge.
func (s *SV) Ge(x any) bool { a, b := strPair(s, x); return a >= b }

// NumEq is Perl's ==. NaN equals nothing, itself included.
func (s *SV) NumEq(x any) bool { c, ok := numCompare(s, operand(x)); return ok && c == 0 }

// NumNe is Perl's !=. NaN is unequal to everything.
func (s *SV) NumNe(x any) bool { c, ok := numCompare(s, operand(x)); return !ok || c != 0 }

// NumLt is Perl's <.
func (s *SV) NumLt(x any) bool { c, ok := numCompare(s, operand(x)); return ok && c < 0 }

// NumLe is Perl's <=.
func (s *SV) NumLe(x any) bool { c, ok := numCompare(s, operand(x)); return ok && c <= 0 }

// NumGt is Perl's >.
func (s *SV) NumGt(x any) bool { c, ok := numCompare(s, operand(x)); return ok && c > 0 }

// NumGe is Perl's >=.
func (s *SV) NumGe(x any) bool { c, ok := numCompare(s, operand(x)); return ok && c >= 0 }

// NumCmp is Perl's <=>, for use with AV.Sort. Comparisons involving NaN,
// which Perl leaves undefined, return 0.
func NumCmp(a, b *SV) int {
	c, _ := numCompare(a, b)
	return c
}

// numCompare compares a and b numerically, exactly when both are integers.
// ok is false if either is NaN.
func numCompare(a, b *SV) (c int, ok bool) {
	ai, af, aInt := a.numify()
	bi, bf, bInt := b.numify()
	if aInt && bInt {
		return cmp.Compare(ai, bi), true
	}
	if math.IsNaN(af) || math.IsNaN(bf) {
		return 0, false
	}
	if c := cmp.Compare(af, bf); c != 0 {
		return c, true // exact even for kept digits: rounding keeps order
	}
	if c, ok := exactCompare(a, b); ok {
		return c, true // equal floats: the digits decide
	}
	return 0, true
}

func addInt(a, b int64) (int64, bool) {
	r := a + b
	return r, (a >= 0) != (b >= 0) || (r >= 0) == (a >= 0)
}

func subInt(a, b int64) (int64, bool) {
	r := a - b
	return r, (a >= 0) == (b >= 0) || (r >= 0) == (a >= 0)
}

func mulInt(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	if (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return 0, false
	}
	r := a * b
	return r, r/b == a
}

func powInt(base, exp int64) (int64, bool) {
	r := int64(1)
	for exp > 0 {
		if exp&1 == 1 {
			var ok bool
			if r, ok = mulInt(r, base); !ok {
				return 0, false
			}
		}
		exp >>= 1
		if exp > 0 {
			var ok bool
			if base, ok = mulInt(base, base); !ok {
				return 0, false
			}
		}
	}
	return r, true
}
