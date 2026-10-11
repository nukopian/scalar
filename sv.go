// Package scalar brings Perl-style dynamic values to Go.
//
// SV is a scalar: undef, a boolean, an integer (IV), a float (NV), a string,
// or a reference to an AV (array) or HV (hash). Scalars coerce between
// string, numeric and boolean contexts the way Perl does, and dereferencing
// an undef scalar as an array or hash autovivifies it.
//
// Nothing here is safe for concurrent use.
package scalar

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind reports what an SV currently holds.
type Kind int

const (
	Undef Kind = iota
	Bool
	Int
	Float
	String
	ArrayRef
	HashRef
)

// SV is a Perl-style scalar. The zero value is undef.
type SV struct {
	v any // nil (undef), bool, int64, float64, string, *AV, *HV

	// viv attaches a detached SV to the container it was read from. It is
	// set when Get misses on an AV or HV, and fires on the first write, so
	// reading a missing element doesn't create it but writing through it does.
	viv func(*SV)

	// noCreate marks an undef that Perl couldn't create, such as an array
	// element before index 0 or an element of a non-reference. Writing to it
	// panics with this message; dereferencing it yields nil, so reads
	// through it stay undef.
	noCreate string
}

// Scalar returns a scalar holding x. See Set for how Go values are converted.
func Scalar(x any) *SV {
	s := &SV{}
	s.v = normalize(x)
	return s
}

// Set assigns x to s and returns s. Go values are converted as follows:
// integers become IVs, floats become NVs, slices become array references,
// string-keyed maps become hash references, and *SV is copied by value (a
// copied reference still points at the same AV/HV, as in Perl). Anything
// else is stringified with fmt.Sprint.
func (s *SV) Set(x any) *SV {
	s.vivify()
	s.v = normalize(x)
	return s
}

// Copy returns a new scalar holding s's value, as Perl's my $x = $s does.
// Elements returned by Get, Path, All and Values are the stored values
// themselves; Copy gives you one you can change without affecting the
// container. Like Perl's assignment, the copy is shallow: a copied
// reference points at the same array or hash.
func (s *SV) Copy() *SV {
	return &SV{v: s.val()}
}

func (s *SV) vivify() {
	if s.noCreate != "" {
		panic("scalar: " + s.noCreate)
	}
	if s.viv != nil {
		f := s.viv
		s.viv = nil
		f(s)
	}
}

func (s *SV) val() any {
	if s == nil {
		return nil
	}
	return s.v
}

// Kind reports what s currently holds.
func (s *SV) Kind() Kind {
	switch s.val().(type) {
	case bool:
		return Bool
	case int64:
		return Int
	case float64:
		return Float
	case string:
		return String
	case *AV:
		return ArrayRef
	case *HV:
		return HashRef
	}
	return Undef
}

// Defined is Perl's defined().
func (s *SV) Defined() bool { return s.val() != nil }

// Ref is Perl's ref(): "ARRAY", "HASH", or "" for non-references.
func (s *SV) Ref() string {
	switch s.val().(type) {
	case *AV:
		return "ARRAY"
	case *HV:
		return "HASH"
	}
	return ""
}

// Str returns s in string context.
func (s *SV) Str() string {
	switch v := s.val().(type) {
	case bool:
		if v {
			return "1"
		}
		return ""
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return formatFloat(v)
	case string:
		return v
	case *AV:
		return fmt.Sprintf("ARRAY(%p)", v)
	case *HV:
		return fmt.Sprintf("HASH(%p)", v)
	}
	return ""
}

// String implements fmt.Stringer.
func (s *SV) String() string { return s.Str() }

// Length is Perl's length($s): the number of characters (not bytes) in s
// in string context, so Scalar(12345).Length() is 5 and "café" is 4. Perl
// returns undef for undef; this returns 0.
func (s *SV) Length() int { return utf8.RuneCountInString(s.Str()) }

// Num returns s in numeric context. Values that don't look like numbers
// numify as Perl does ("42abc" is 42, "abc" is 0); use NumOK to detect that.
func (s *SV) Num() float64 {
	f, _ := s.NumOK()
	return f
}

// NumOK is Num, plus false wherever Perl would warn: undef, empty or
// non-numeric strings, and strings with trailing garbage.
func (s *SV) NumOK() (float64, bool) {
	switch v := s.val().(type) {
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		_, nv, _, ok := parseNumber(v)
		return nv, ok
	case *AV, *HV:
		return float64(reflect.ValueOf(v).Pointer()), true
	}
	return 0, false
}

// Int returns s in integer context, truncating toward zero.
func (s *SV) Int() int64 {
	i, _ := s.IntOK()
	return i
}

// IntOK is Int, plus false wherever NumOK would be false, and for NaN or
// floats outside the int64 range (which are clamped).
func (s *SV) IntOK() (int64, bool) {
	switch v := s.val().(type) {
	case int64:
		return v, true
	case float64:
		return floatToInt(v)
	case string:
		iv, nv, isInt, ok := parseNumber(v)
		if isInt {
			return iv, ok
		}
		i, inRange := floatToInt(nv)
		return i, ok && inRange
	}
	f, ok := s.NumOK()
	return int64(f), ok
}

// Bool returns s in boolean context: undef, "", "0", 0 and 0.0 are false;
// everything else, including "0.0" and "00", is true.
func (s *SV) Bool() bool {
	switch v := s.val().(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != "" && v != "0"
	}
	return true
}

// AV dereferences s as an array. An undef s is autovivified into a new
// array reference. Returns nil if s holds anything else, or is an undef
// that can't be created (see AV.Get); reads through a nil *AV yield undef,
// writes panic (Perl's "Not an ARRAY reference").
func (s *SV) AV() *AV {
	if s == nil {
		return nil
	}
	switch v := s.v.(type) {
	case *AV:
		return v
	case nil:
		if s.noCreate != "" {
			return nil
		}
		a := &AV{}
		s.vivify()
		s.v = a
		return a
	}
	return nil
}

// HV dereferences s as a hash, autovivifying undef like AV does.
func (s *SV) HV() *HV {
	if s == nil {
		return nil
	}
	switch v := s.v.(type) {
	case *HV:
		return v
	case nil:
		if s.noCreate != "" {
			return nil
		}
		h := &HV{}
		s.vivify()
		s.v = h
		return h
	}
	return nil
}

// Path walks a chain of hash keys (string) and array indices (int), so
// s.Path("content", 0, "text") is $s->{content}[0]{text}. Intermediate
// levels autovivify, as they do in Perl; the final element does not.
func (s *SV) Path(keys ...any) *SV {
	cur := s
	for _, k := range keys {
		switch k := k.(type) {
		case string:
			cur = cur.HV().Get(k)
		case int:
			cur = cur.AV().Get(k)
		default:
			panic(fmt.Sprintf("scalar: Path key must be string or int, got %T", k))
		}
	}
	return cur
}

// StrCmp is Perl's cmp, for use with AV.Sort.
func StrCmp(a, b *SV) int { return strings.Compare(a.Str(), b.Str()) }

func formatFloat(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Inf"
	case math.IsInf(f, -1):
		return "-Inf"
	case math.IsNaN(f):
		return "NaN"
	}
	// Perl prints NVs with %.15g.
	return strconv.FormatFloat(f, 'g', 15, 64)
}

func floatToInt(f float64) (int64, bool) {
	switch {
	case math.IsNaN(f):
		return 0, false
	case f >= math.MaxInt64:
		return math.MaxInt64, false
	case f <= math.MinInt64:
		return math.MinInt64, false
	}
	return int64(f), true
}

func normalize(x any) any {
	switch v := x.(type) {
	case nil:
		return nil
	case *SV:
		return v.val()
	case SV:
		return v.v
	case bool, string, *AV, *HV:
		return v
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return uintToNum(uint64(v))
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return uintToNum(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case jsonNumber:
		return normalizeJSONNumber(v)
	case []any:
		return Array(v...)
	case map[string]any:
		h := &HV{}
		for k, e := range v {
			h.Set(k, e)
		}
		return h
	}

	rv := reflect.ValueOf(x)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return nil
		}
		a := &AV{}
		for i := range rv.Len() {
			a.Push(rv.Index(i).Interface())
		}
		return a
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			if rv.IsNil() {
				return nil
			}
			h := &HV{}
			for it := rv.MapRange(); it.Next(); {
				h.Set(it.Key().String(), it.Value().Interface())
			}
			return h
		}
	case reflect.Pointer:
		if rv.IsNil() {
			return nil
		}
		return normalize(rv.Elem().Interface())
	}
	return fmt.Sprint(x)
}

func uintToNum(u uint64) any {
	if u <= math.MaxInt64 {
		return int64(u)
	}
	return float64(u)
}
