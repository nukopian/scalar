package scalar

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
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
	// Bytes is a byte string: binary data, such as a blob or Pack's result.
	// See SV.Bytes.
	Bytes
)

// SV is a Perl-style scalar. The zero value is undef.
type SV struct {
	v any // nil (undef), bool, int64, float64, string, []byte, *AV, *HV

	// viv attaches a detached SV to the container it was read from. It is
	// set when Get misses on an AV or HV, and fires on the first write, so
	// reading a missing element doesn't create it but writing through it does.
	viv func(*SV)

	// noCreate marks an undef that Perl couldn't create, such as an array
	// element before index 0 or an element of a non-reference. Writing to it
	// panics with this message; dereferencing it yields nil, so reads
	// through it stay undef.
	noCreate string

	// ro marks a scalar Lock has made read-only: writing to it panics, and
	// as an undef it doesn't autovivify.
	ro bool
}

// Value returns a scalar holding x. See Set for how Go values are converted.
func Value(x any) *SV {
	return &SV{v: normalize(x)}
}

// Set is Perl's $s = x: it assigns x to s and returns s. To assign to what
// s refers to, $s->{key} = x or $s->[i] = x, subscript first, as Perl
// does: s.Get(key).Set(x), as HV.Set and AV.Set do. Go values are
// converted as follows:
// integers become IVs, floats become NVs, slices become array references,
// maps become hash references (with their keys in string context, as Perl
// stores them), structs become hash references too, keyed as encoding/json
// would write them (Decode's way back), a type with its own MarshalJSON or
// MarshalText is what that gives (time.Time is its RFC 3339 string), and
// *SV is copied by value (a copied reference still points at the same
// AV/HV, as in Perl); a nil *AV or *HV is undef, as a nil slice or map
// is. Anything else is stringified with fmt.Sprint. A Go value that
// contains itself panics, as encoding/json refuses one.
func (s *SV) Set(x any) *SV {
	s.vivify()
	s.v = normalize(x)
	return s
}

// Undef is Perl's undef $s: s becomes undef, and Undef returns it, now
// undef. A reference s held is dropped, and what it referred to is left
// alone. An element of an array or hash keeps its place, as in Perl: after
// h.Get("k").Undef(), "k" still exists, with an undef value (and a missing
// key is created, as undef $h{k} creates it). To remove an element, use
// Delete.
func (s *SV) Undef() *SV { return s.Set(nil) }

// Copy returns a new scalar holding s's value, as Perl's my $x = $s does.
// Elements returned by Get, Path, All and Values are the stored values
// themselves; Copy gives you one you can change without affecting the
// container. Like Perl's assignment, the copy is shallow: a copied
// reference points at the same array or hash. For a deep copy, use Clone.
func (s *SV) Copy() *SV {
	return &SV{v: s.val()}
}

func (s *SV) vivify() {
	if s.ro {
		panic("scalar: " + readOnlyMsg)
	}
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
	case []byte:
		return Bytes
	}
	return Undef
}

// Defined is Perl's defined().
func (s *SV) Defined() bool { return s.val() != nil }

// Any returns s as a plain Go value, in the shapes encoding/json uses:
// undef is nil; booleans, integers, floats and strings are bool, int64,
// float64 and string; array references become []any and hash references
// map[string]any, converted all the way down. The result shares nothing
// with s. It panics if a reference leads back to itself, which plain Go
// values can't represent.
func (s *SV) Any() any {
	v, err := s.toAny(map[any]bool{})
	if err != nil {
		panic("scalar: Any on a " + err.Error())
	}
	return v
}

// toAny is Any, returning an error for a reference that leads back to
// itself rather than panicking. inProgress holds the arrays and hashes
// being converted, so the same one can appear twice without being a cycle.
func (s *SV) toAny(inProgress map[any]bool) (any, error) {
	switch v := s.val().(type) {
	case *AV, *HV:
		if inProgress[v] {
			return nil, fmt.Errorf("self-referencing %s", s.Ref())
		}
		inProgress[v] = true
		defer delete(inProgress, v)
	}

	switch v := s.val().(type) {
	case []byte:
		return slices.Clone(v), nil
	case *AV:
		out := make([]any, v.Length())
		for i, e := range v.items {
			var err error
			if out[i], err = e.toAny(inProgress); err != nil {
				return nil, err
			}
		}
		return out, nil
	case *HV:
		out := make(map[string]any, v.Length())
		for k, e := range v.m {
			var err error
			if out[k], err = e.toAny(inProgress); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	return s.val(), nil
}

// Or is Perl's defined-or, $s // $def: a new scalar holding s's value if s
// is defined, otherwise def's. Unlike ||, a defined but false value such as
// 0 or "" is kept.
func (s *SV) Or(def any) *SV {
	if s.Defined() {
		return s.Copy()
	}
	return operand(def).Copy()
}

// OrSet is Perl's $s //= $def: if s is undef, it sets s to def. It returns
// s. Like Set, it writes through to a missing array or hash element,
// creating it.
func (s *SV) OrSet(def any) *SV {
	if !s.Defined() {
		s.Set(def)
	}
	return s
}

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
	case []byte:
		return string(v) // the bytes as they are
	case *AV:
		return fmt.Sprintf("ARRAY(%p)", v)
	case *HV:
		return fmt.Sprintf("HASH(%p)", v)
	}
	return ""
}

// String implements fmt.Stringer.
func (s *SV) String() string { return s.Str() }

// Bytes returns s as a Go []byte: a byte string's bytes, or anything else's
// bytes in string context (see Str), so text gives its UTF-8 encoding. The
// slice is new, so changing it doesn't change s.
//
// A byte string is a scalar of kind Bytes: one made from a Go []byte, or by
// Pack. Its Str is the bytes as they are, for writing out or Unpack. The
// character functions (Length, Substr, Index, Lc, the regex methods and so
// on) treat each byte as one character from 0 to 255, as Perl treats a byte
// string under use v5.12 or later: lengths and positions count bytes, \w
// and case rules apply to bytes as Latin-1 letters. A result that's all
// characters from 0 to 255 is a byte string again; one with characters
// beyond, such as Uc of "\xff", is text.
func (s *SV) Bytes() []byte {
	if b, ok := s.val().([]byte); ok {
		return slices.Clone(b)
	}
	return []byte(s.Str())
}

// text is s as characters, for the string functions: its string, or for a
// byte string, its bytes as Latin-1 characters, one per byte, so that a
// byte 0xe9 is the character é. isBytes reports which.
func (s *SV) text() (str string, isBytes bool) {
	b, ok := s.val().([]byte)
	if !ok {
		return s.Str(), false
	}
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r), true
}

// textResult is the result of a string function: str as text, or, if the
// input was a byte string and every character of str is from 0 to 255, a
// byte string again. A character beyond 255 makes it text, as Perl upgrades
// a byte string.
func textResult(str string, isBytes bool) *SV {
	if !isBytes {
		return &SV{v: str}
	}
	b := make([]byte, 0, len(str))
	for _, r := range str {
		if r > 0xff {
			return &SV{v: str}
		}
		b = append(b, byte(r))
	}
	return &SV{v: b}
}

// Format implements fmt.Formatter, so that Go's format verbs treat s as
// Perl's sprintf does, in the context each verb implies: %d, %b, %o, %x, %X
// and %c in integer context (see Int), %e, %E, %f, %F, %g and %G in numeric
// context (see Num), %t in boolean context, and %s, %q, %v and anything
// else in string context (see Str). Widths, precisions and flags work as
// usual, so fmt.Sprintf("%05.2f", Value("3.14159")) is "03.14".
//
// A byte string prints as its bytes, as Perl's print does, and formats as Go
// formats a []byte for %q, %x and %X: %x gives its bytes in hex, which is
// what you want for binary data, where Perl's %x would read it as a number.
func (s *SV) Format(f fmt.State, verb rune) {
	if b, ok := s.val().([]byte); ok && strings.ContainsRune("qxX", verb) {
		fmt.Fprintf(f, fmt.FormatString(f, verb), b)
		return
	}
	var arg any
	switch verb {
	case 'd', 'b', 'o', 'x', 'X', 'c':
		arg = s.Int()
	case 'e', 'E', 'f', 'F', 'g', 'G':
		arg = s.Num()
	case 't':
		arg = s.Bool()
	default:
		arg = s.Str()
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), arg)
}

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
	case []byte:
		_, nv, _, ok := parseNumber(string(v))
		return nv, ok
	case *AV, *HV:
		return float64(refAddr(v)), true
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
	case string, []byte:
		iv, nv, isInt, ok := parseNumber(Value(v).Str())
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
	case []byte:
		return len(v) > 0 && string(v) != "0"
	}
	return true
}

// AV dereferences s as an array, as Perl's @$s does, for what works on the
// whole array, such as Push or Length; for one element, s.Get(i) needs no
// AV. An undef s is autovivified into a new array reference. Returns nil if
// s holds anything else, or is an undef that can't be created (see
// AV.Get); reads through a nil *AV yield undef, writes panic (Perl's "Not
// an ARRAY reference").
func (s *SV) AV() *AV {
	if s == nil {
		return nil
	}
	switch v := s.v.(type) {
	case *AV:
		return v
	case nil:
		if s.noCreate != "" || s.ro {
			return nil
		}
		a := &AV{}
		s.vivify()
		s.v = a
		return a
	}
	return nil
}

// HV dereferences s as a hash, as Perl's %$s does, for what works on the
// whole hash, such as Keys or Length; for one key, s.Get(key) needs no HV.
// It autovivifies undef, and returns nil for anything else, as AV does.
func (s *SV) HV() *HV {
	if s == nil {
		return nil
	}
	switch v := s.v.(type) {
	case *HV:
		return v
	case nil:
		if s.noCreate != "" || s.ro {
			return nil
		}
		h := &HV{}
		s.vivify()
		s.v = h
		return h
	}
	return nil
}

// Get is Perl's $s->{key} or $s->[i]: a string key looks up the hash s
// refers to, and an int the array.
//
//	s.Get("content").Get(0).Get("text")  // $s->{content}[0]{text}
//
// It needs no AV or HV between the steps, as Perl needs no arrow between
// subscripts, and s.Get(key).Set(x) is $s->{key} = x, as HV.Set is
// h.Get(key).Set(x). It's HV().Get or AV().Get in one step, with the same rules:
// an undef s autovivifies, as Perl's $s->{key} does even when only read,
// and a missing key or index reads as undef without being created, until
// something is written to it. If s holds anything else, Get returns undef,
// and writing to it panics. Any other type of key panics.
func (s *SV) Get(key any) *SV {
	switch k := key.(type) {
	case string:
		return s.HV().Get(k)
	case int:
		return s.AV().Get(k)
	}
	panic(badSubscript("Get", key))
}

// Exists is Perl's exists $s->{key} or exists $s->[i]: whether the hash or
// array s refers to has that key or element, even if its value is undef. A
// string key is a hash key and an int an array index, as for Get.
//
// As in Perl, an undef s autovivifies into the hash or array it looks in,
// though the key itself is never created; PathOK asks without creating
// anything. If s is anything else, such as a number, or a hash asked for
// an index, Exists panics, as Perl dies; PathOK says false instead.
func (s *SV) Exists(key any) bool {
	st := subscript("Exists", key)
	st.deref(s)
	return st.peek(s) != nil
}

// Delete is Perl's delete $s->{key} or delete $s->[i]: it removes the key
// or element from the hash or array s refers to, and returns its value, or
// undef if there was nothing there. A string key is a hash key and an int
// an array index, as for Get, and an array keeps its holes as AV.Delete
// says.
//
// As in Perl, an undef s autovivifies into the hash or array it deletes
// from; DeletePath deletes without creating anything. If s is anything
// else, Delete panics, as Perl dies; DeletePath deletes nothing instead.
func (s *SV) Delete(key any) *SV {
	st := subscript("Delete", key)
	st.deref(s)
	return s.deleteSteps([]atStep{st})
}

// deref autovivifies an undef s into the hash or array that st looks in, as
// Perl's exists and delete do, and panics, as Perl dies, if s is anything
// else, or an undef that can't be created.
func (st atStep) deref(s *SV) {
	msg := "Not a HASH reference"
	if st.kind == stepKey {
		if s.HV() != nil {
			return
		}
	} else if msg = notArrayRef; s.AV() != nil {
		return
	}
	if s != nil && s.v == nil {
		s.vivify() // read-only, or can't be created: panics with why
	}
	panic("scalar: " + msg)
}

// subscript is key as one step of a path, for Exists and Delete, whose
// name method is.
func subscript(method string, key any) atStep {
	switch k := key.(type) {
	case string:
		return atStep{kind: stepKey, key: k}
	case int:
		return atStep{kind: stepIndex, index: k}
	}
	panic(badSubscript(method, key))
}

// badSubscript is the panic for a key to Get, Exists or Delete that's
// neither a string nor an int.
func badSubscript(method string, key any) string {
	return fmt.Sprintf("scalar: %s key must be a string or an int, got %T", method, key)
}

// Path walks a chain of hash keys (string) and array indices (int), so
// s.Path("content", 0, "text") is $s->{content}[0]{text}, as is
// s.Get("content").Get(0).Get("text"). Intermediate levels autovivify, as
// they do in Perl; the final element does not. A key may also be a compiled
// path (see MustPath), whose steps are taken in turn.
func (s *SV) Path(keys ...any) *SV {
	cur := s
	for _, k := range keys {
		switch k := k.(type) {
		case string:
			cur = cur.HV().Get(k)
		case int:
			cur = cur.AV().Get(k)
		case *PathExpr:
			for _, st := range k.steps {
				cur = st.walk(cur)
			}
		default:
			panic(fmt.Sprintf("scalar: Path key must be a string, an int or a *PathExpr, got %T", k))
		}
	}
	return cur
}

// PathOK is Path with Go's comma-ok: ok reports whether every step exists,
// so a value that's present but null (undef) gives ok true, while a missing
// key or index, or a step through something that isn't an array or hash,
// gives ok false and an undef scalar. Unlike Path, it never creates
// anything. When ok is true, v is the stored value itself, as with Path.
func (s *SV) PathOK(keys ...any) (v *SV, ok bool) {
	cur := s
	if cur == nil {
		return &SV{}, false
	}
	for _, st := range pathSteps(keys) {
		if cur = st.peek(cur); cur == nil {
			return &SV{}, false
		}
	}
	return cur, true
}

// pathSteps turns Path's keys into the steps At takes: a string is a hash
// key, taken whole, an int an array index, and a *PathExpr its own steps.
func pathSteps(keys []any) []atStep {
	var steps []atStep
	for _, k := range keys {
		switch k := k.(type) {
		case string:
			steps = append(steps, atStep{kind: stepKey, key: k})
		case int:
			steps = append(steps, atStep{kind: stepIndex, index: k})
		case *PathExpr:
			steps = append(steps, k.steps...)
		default:
			panic(fmt.Sprintf("scalar: Path key must be a string, an int or a *PathExpr, got %T", k))
		}
	}
	return steps
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

// refAddr is a reference's address, which is what Perl gives for a reference
// in numeric context.
func refAddr(ref any) int64 {
	return int64(reflect.ValueOf(ref).Pointer())
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

// maxValueDepth is how deeply a Go value may nest on its way into a scalar,
// as maxDumpDepth limits a dump. A value that contains itself would recurse
// until the stack overflowed, which no recover catches; encoding/json
// refuses one with an error, and Value, with no error to return, panics.
const maxValueDepth = 10000

func normalize(x any) any { return normalizeAt(x, 0) }

// valueAt is Value for what's depth levels inside the value being converted.
func valueAt(x any, depth int) *SV { return &SV{v: normalizeAt(x, depth)} }

// enter checks the depth as a container is entered, so the panic names it.
func enter(x any, depth int) {
	if depth >= maxValueDepth {
		panic(fmt.Sprintf("scalar: Value: a %T refers to itself, or is nested more than %d levels deep", x, maxValueDepth))
	}
}

func normalizeAt(x any, depth int) any {
	switch v := x.(type) {
	case nil:
		return nil
	case *SV:
		return v.val()
	case SV:
		return v.v
	case bool, string:
		return v
	case *AV:
		if v == nil {
			return nil // a nil array is undef, as a nil slice is
		}
		return v
	case *HV:
		if v == nil {
			return nil
		}
		return v
	case []byte:
		if v == nil {
			return nil // a nil slice is undef
		}
		return slices.Clone(v) // the scalar's own copy
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
	case json.Number:
		return normalizeJSONNumber(v)
	case []any:
		if v == nil {
			return nil // a nil slice is undef, not an empty array
		}
		enter(x, depth)
		a := &AV{items: make([]*SV, len(v))}
		for i, e := range v {
			a.items[i] = valueAt(e, depth+1) // nested arrays stay nested
		}
		return a
	case []string:
		if v == nil {
			return nil
		}
		a := &AV{items: make([]*SV, len(v))}
		for i, e := range v {
			a.items[i] = &SV{v: e}
		}
		return a
	case map[string]any:
		enter(x, depth)
		h := newHV(len(v))
		for k, e := range v {
			h.m[k] = valueAt(e, depth+1)
		}
		return h
	}

	// A type that says how it's written as JSON, or as text, is that:
	// time.Time is its RFC 3339 string.
	switch m := x.(type) {
	case json.Marshaler:
		if data, err := m.MarshalJSON(); err == nil {
			if v, err := FromJSON(data); err == nil {
				return v.v
			}
		}
	case encoding.TextMarshaler:
		if text, err := m.MarshalText(); err == nil {
			return string(text)
		}
	}

	rv := reflect.ValueOf(x)
	switch rv.Kind() {
	case reflect.Struct:
		enter(x, depth)
		return structHash(rv, depth)
	// Named types of the kinds above, such as type Kind string: by kind,
	// as encoding/json writes them, not stringified.
	case reflect.Bool:
		return rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintToNum(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.String:
		return rv.String()
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return nil
		}
		enter(x, depth)
		a := &AV{items: make([]*SV, rv.Len())}
		for i := range rv.Len() {
			a.items[i] = valueAt(rv.Index(i).Interface(), depth+1)
		}
		return a
	case reflect.Map:
		if rv.IsNil() {
			return nil
		}
		enter(x, depth)
		h := newHV(rv.Len())
		for it := rv.MapRange(); it.Next(); {
			h.m[mapKey(it.Key())] = valueAt(it.Value().Interface(), depth+1)
		}
		return h
	case reflect.Pointer:
		if rv.IsNil() {
			return nil
		}
		enter(x, depth)
		return normalizeAt(rv.Elem().Interface(), depth+1)
	}
	return fmt.Sprint(x)
}

// structHash turns a struct into a hash, as encoding/json would encode it:
// a key for each exported field, named by its json tag or else the field;
// the fields of embedded structs promoted; none for a field tagged "-", or
// tagged omitempty and empty. It's the way Decode, the way back, reads a
// struct.
func structHash(rv reflect.Value, depth int) *HV {
	fs := structFields(rv.Type())
	h := newHV(len(fs.byName))
	for name, f := range fs.byName {
		v, ok := fieldIn(rv, f.index)
		if !ok || f.omitEmpty && isEmpty(v) {
			continue
		}
		h.m[name] = valueAt(v.Interface(), depth+1)
	}
	return h
}

// fieldIn is rv.FieldByIndex, but false where an embedded pointer on the
// way is nil, rather than a panic.
func fieldIn(rv reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 && rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return reflect.Value{}, false
			}
			rv = rv.Elem()
		}
		rv = rv.Field(x)
	}
	return rv, true
}

// isEmpty is what omitempty leaves out, as encoding/json has it: false, 0,
// a nil pointer or interface, and an empty array, slice, map or string.
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Interface, reflect.Pointer:
		return v.IsZero()
	}
	return false
}

// mapKey turns a Go map key into a hash key. As in Perl, hash keys are
// strings, so other keys are used in string context: 1 becomes "1".
func mapKey(k reflect.Value) string {
	return Value(k.Interface()).Str()
}

func uintToNum(u uint64) any {
	if u <= math.MaxInt64 {
		return int64(u)
	}
	return float64(u)
}
