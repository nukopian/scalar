package scalar

import (
	"cmp"
	"encoding/json"
	"math/big"
	"strconv"
)

// An experiment, not for develop: roadmap 2, JSON whole numbers that keep
// their digits, built two ways behind one switch so that the same tests
// and benchmarks compare them.
//
//   - exactText is the roadmap's plan: a whole number too big for int64
//     is a float64 in v, with its digits in rare beside it.
//   - exactUV is Perl's own way: a whole number up to 2⁶⁴−1 is a uint64
//     in v, as Perl's UV, exact; beyond that, a float, as in Perl.
//
// exactOff is develop's behaviour.
type exactMode int

const (
	exactOff exactMode = iota
	exactText
	exactUV
)

var exactInts = exactOff

// wholeText carries a whole number's float and digits from normalize to
// the scalar that stores them, in exactText.
type wholeText struct {
	f    float64
	text string
}

// isWhole reports whether a JSON number's text is a whole number, with no
// fraction or exponent.
func isWhole(n string) bool {
	for i := range len(n) {
		if c := n[i]; c == '.' || c == 'e' || c == 'E' {
			return false
		}
	}
	return true
}

// bigWhole is a whole number that doesn't fit int64, n its digits and f
// its float, as exactInts stores it.
func bigWhole(n string, f float64) any {
	switch exactInts {
	case exactUV:
		if u, err := strconv.ParseUint(n, 10, 64); err == nil {
			return u
		}
	case exactText:
		return wholeText{f, n}
	}
	return f
}

// digits is the text kept beside s's float, or "".
func (s *SV) digits() string {
	if s == nil || s.rare == nil {
		return ""
	}
	return s.rare.digits
}

// setValue stores x, normalize's result, in s: a whole number's digits,
// if it carries them, go beside its float.
func (s *SV) setValue(x any) {
	if w, ok := x.(wholeText); ok {
		s.v = w.f
		if s.rare == nil {
			s.rare = &svRare{}
		}
		s.rare.digits = w.text
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

// packed is s's value as normalize gives it: with its digits, if it has
// them.
func (s *SV) packed() any {
	if d := s.digits(); d != "" {
		return wholeText{s.v.(float64), d}
	}
	return s.val()
}

// exactJSON is s's value as JSON writes it: a whole number's digits
// rather than its float.
func (s *SV) exactJSON() any {
	switch v := s.val().(type) {
	case uint64:
		return v
	case float64:
		if d := s.digits(); d != "" {
			return json.Number(d)
		}
	}
	return s.val()
}

// exactCompare compares a and b exactly where either is a whole number
// beyond int64, as Perl compares IVs and UVs; ok is false where it can't.
func exactCompare(a, b *SV) (c int, ok bool) {
	if !a.beyondInt64() && !b.beyondInt64() {
		return 0, false
	}
	if au, ok := a.val().(uint64); ok {
		switch bv := b.val().(type) {
		case uint64:
			return cmp.Compare(au, bv), true
		case int64:
			return 1, true
		}
	}
	if _, ok := b.val().(uint64); ok {
		if _, ok := a.val().(int64); ok {
			return -1, true
		}
	}
	ab, aok := a.exactBig()
	bb, bok := b.exactBig()
	if !aok || !bok {
		return 0, false
	}
	return ab.Cmp(bb), true
}

// beyondInt64 reports whether s is a whole number exactInts keeps exactly
// beyond int64.
func (s *SV) beyondInt64() bool {
	if _, ok := s.val().(uint64); ok {
		return true
	}
	return s.digits() != ""
}

// exactBig is s as an exact integer, if it is one.
func (s *SV) exactBig() (*big.Int, bool) {
	switch v := s.val().(type) {
	case int64:
		return big.NewInt(v), true
	case uint64:
		return new(big.Int).SetUint64(v), true
	case float64:
		if d := s.digits(); d != "" {
			return new(big.Int).SetString(d, 10)
		}
	}
	return nil, false
}

// exactUint is s as a uint64 field takes it, exactly, if it's a whole
// number beyond int64.
func (s *SV) exactUint() (uint64, bool) {
	switch v := s.val().(type) {
	case uint64:
		return v, true
	case float64:
		if d := s.digits(); d != "" {
			u, err := strconv.ParseUint(d, 10, 64)
			return u, err == nil
		}
	}
	return 0, false
}
