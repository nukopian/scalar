package scalar

import (
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// Decode stores s in the Go value v points to, as json.Unmarshal would
// store the same data, but converting each value as Perl would, so a
// number that came as a string, "45000", fills an int field. It's the way
// out of scalar once the shape of the data is known:
//
//	var p struct {
//		Title  string   `json:"title"`
//		Skills []string `json:"skills"`
//		Salary struct{ Min, Max int } `json:"salary"`
//	}
//	err := doc.Decode(&p)
//
// Hashes fill structs, matching keys to fields as encoding/json does (the
// json tag's name, or the field's, exactly or else ignoring case), and maps
// with string or integer keys. Arrays fill slices and arrays. A *SV, SV,
// *AV or *HV field takes a deep copy (see Clone) of whatever is there, for
// the parts that stay loose. A type with its own UnmarshalJSON or
// UnmarshalText, such as time.Time, decodes itself. Keys with no field are
// ignored, and fields with no key are left as they are.
//
// Numbers and strings convert as in Perl, but only where Perl wouldn't
// warn: "42" and " 42 " fill an int, and "42abc", "45,000" and "" don't.
// An integer field takes only a whole number that fits it. A boolean
// doesn't fill a number or string field, where Perl would make it 1 or "":
// it's a sign the data has the wrong shape. A bool field takes a boolean, 0
// or 1, or "true" or "false", but not Perl's other true and false values,
// since "false" would be true. A []byte field takes a byte string's bytes,
// or decodes text as base64, as encoding/json does. Undef sets a pointer,
// slice, map or interface to nil, and leaves anything else as it is, as
// JSON's null does.
//
// Data that refers back to itself, decoded into a type that can hold it
// only by recursing, such as a struct with a pointer to its own type, gives
// an error rather than recursing for ever.
//
// v must be a non-nil pointer. Decode stops at the first value it can't
// store, and returns a *DecodeError saying where, as a path for At. It takes
// a hash's keys in sorted order, so when several values don't fit, the
// error is always about the same one.
func (s *SV) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("scalar: Decode needs a non-nil pointer, not %T", v)
	}
	d := decoder{inProgress: map[decodeStep]bool{}}
	return d.value(s, rv.Elem(), "")
}

// decoder decodes one value, remembering which arrays and hashes it's in
// the middle of decoding into which Go types, so that data that refers back
// to itself gives an error rather than recursing until the stack runs out.
type decoder struct {
	inProgress map[decodeStep]bool
}

// decodeStep is an array or hash being decoded into a Go type.
type decodeStep struct {
	container any
	typ       reflect.Type
}

// DecodeError is the error Decode returns for a value it can't store: where
// it is, as a path for At, and why.
type DecodeError struct {
	Path string // as At takes it, such as "salary.max" or "skills[2]"; "" for s itself
	Msg  string // such as `want int, got "competitive"`
}

func (e *DecodeError) Error() string {
	if e.Path == "" {
		return "scalar: Decode: " + e.Msg
	}
	return "scalar: Decode: " + e.Path + ": " + e.Msg
}

var (
	svType = reflect.TypeFor[SV]()
	avType = reflect.TypeFor[AV]()
	hvType = reflect.TypeFor[HV]()
)

// plainKey is a hash key At can take after a dot without brackets or braces.
var plainKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// braceEscaper escapes a key for At's braces.
var braceEscaper = strings.NewReplacer(`\`, `\\`, `}`, `\}`)

// keyPath adds a hash key to an At path: .name, or {key} for any other key.
func keyPath(path, key string) string {
	if !plainKey.MatchString(key) {
		return path + "{" + braceEscaper.Replace(key) + "}"
	}
	if path == "" {
		return key
	}
	return path + "." + key
}

// describe is a short account of s for an error message: its value, if it's
// plain, or what it is.
func describe(s *SV) string {
	switch s.Kind() {
	case ArrayRef:
		return "an array"
	case HashRef:
		return "a hash"
	case Undef:
		return "undef"
	case Bool:
		return strconv.FormatBool(s.Bool())
	case Int, Float:
		return s.Str()
	}
	d := s.Dump() // a string, quoted, with any odd characters escaped
	if len(d) > 40 {
		d = d[:37] + "..."
	}
	return d
}

func (d decoder) value(s *SV, v reflect.Value, path string) error {
	fail := func(want string) error {
		return &DecodeError{Path: path, Msg: fmt.Sprintf("want %s, got %s", want, describe(s))}
	}

	// The loose parts: a deep copy of whatever is there. Undef leaves an
	// array or hash as it is.
	switch v.Type() {
	case svType:
		v.Set(reflect.ValueOf(*s.Clone()))
		return nil
	case avType:
		if !s.Defined() {
			return nil
		}
		if s.Kind() != ArrayRef {
			return fail("an array")
		}
		v.Set(reflect.ValueOf(*s.AV().Clone()))
		return nil
	case hvType:
		if !s.Defined() {
			return nil
		}
		if s.Kind() != HashRef {
			return fail("a hash")
		}
		v.Set(reflect.ValueOf(*s.HV().Clone()))
		return nil
	}

	// A type that decodes itself: from JSON, or from text.
	if v.CanAddr() && v.Kind() != reflect.Interface {
		if u, ok := v.Addr().Interface().(json.Unmarshaler); ok {
			data, err := s.MarshalJSON()
			if err == nil {
				err = u.UnmarshalJSON(data)
			}
			if err != nil {
				return &DecodeError{Path: path, Msg: err.Error()}
			}
			return nil
		}
		if u, ok := v.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if k := s.Kind(); k != String && k != Bytes {
				return fail("text for " + v.Type().String())
			}
			if err := u.UnmarshalText([]byte(s.Str())); err != nil {
				return &DecodeError{Path: path, Msg: err.Error()}
			}
			return nil
		}
	}

	if !s.Defined() {
		switch v.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			v.SetZero()
		}
		return nil
	}

	// An array or hash met again on the way down, into the same type: the
	// data refers back to itself, and decoding it would never end.
	if c := s.val(); isRef(c) {
		step := decodeStep{c, v.Type()}
		if d.inProgress[step] {
			return &DecodeError{Path: path, Msg: "the data refers back to itself here, so it can't be decoded"}
		}
		d.inProgress[step] = true
		defer delete(d.inProgress, step)
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return d.value(s, v.Elem(), path)

	case reflect.Interface:
		if v.NumMethod() > 0 {
			return fail(v.Type().String() + ", which Decode can't fill")
		}
		x, err := s.toAny(map[any]bool{})
		if err != nil {
			return &DecodeError{Path: path, Msg: err.Error()}
		}
		if x != nil {
			v.Set(reflect.ValueOf(x))
		}
		return nil

	case reflect.Bool:
		b, ok := decodeBool(s)
		if !ok {
			return fail("bool")
		}
		v.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := decodeInt(s)
		if !ok || v.OverflowInt(i) {
			return fail(v.Type().String())
		}
		v.SetInt(i)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, ok := decodeInt(s)
		if !ok || i < 0 || v.OverflowUint(uint64(i)) {
			return fail(v.Type().String())
		}
		v.SetUint(uint64(i))
		return nil

	case reflect.Float32, reflect.Float64:
		f, ok := s.NumOK()
		if k := s.Kind(); !ok || k == Bool || k == ArrayRef || k == HashRef || v.OverflowFloat(f) {
			return fail(v.Type().String())
		}
		v.SetFloat(f)
		return nil

	case reflect.String:
		if k := s.Kind(); k == Bool || k == ArrayRef || k == HashRef {
			return fail("string")
		}
		v.SetString(s.Str())
		return nil

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 && s.Kind() != ArrayRef {
			switch s.Kind() {
			case Bytes:
				v.SetBytes(s.Bytes())
			case String:
				b, err := base64.StdEncoding.DecodeString(s.Str())
				if err != nil {
					return fail("base64 text for []byte")
				}
				v.SetBytes(b)
			default:
				return fail("a byte string or base64 text")
			}
			return nil
		}
		if s.Kind() != ArrayRef {
			return fail("an array")
		}
		items := s.AV().items
		out := reflect.MakeSlice(v.Type(), len(items), len(items))
		for i, e := range items {
			if err := d.value(e, out.Index(i), path+"["+strconv.Itoa(i)+"]"); err != nil {
				return err
			}
		}
		v.Set(out)
		return nil

	case reflect.Array:
		if s.Kind() != ArrayRef {
			return fail("an array")
		}
		items := s.AV().items
		for i := range v.Len() {
			if i >= len(items) {
				v.Index(i).SetZero()
				continue
			}
			if err := d.value(items[i], v.Index(i), path+"["+strconv.Itoa(i)+"]"); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		if s.Kind() != HashRef {
			return fail("a hash")
		}
		t := v.Type()
		if v.IsNil() {
			v.Set(reflect.MakeMap(t))
		}
		for _, k := range sortedKeys(s.HV()) {
			e := s.HV().m[k]
			key := reflect.New(t.Key()).Elem()
			switch t.Key().Kind() {
			case reflect.String:
				key.SetString(k)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(k, 10, 64)
				if err != nil || key.OverflowInt(n) {
					return &DecodeError{Path: path, Msg: fmt.Sprintf("want %s keys, got %q", t.Key(), k)}
				}
				key.SetInt(n)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				n, err := strconv.ParseUint(k, 10, 64)
				if err != nil || key.OverflowUint(n) {
					return &DecodeError{Path: path, Msg: fmt.Sprintf("want %s keys, got %q", t.Key(), k)}
				}
				key.SetUint(n)
			default:
				return &DecodeError{Path: path, Msg: fmt.Sprintf("can't decode into a map with %s keys", t.Key())}
			}
			elem := reflect.New(t.Elem()).Elem()
			if err := d.value(e, elem, keyPath(path, k)); err != nil {
				return err
			}
			v.SetMapIndex(key, elem)
		}
		return nil

	case reflect.Struct:
		if s.Kind() != HashRef {
			return fail("a hash for " + v.Type().String())
		}
		fields := structFields(v.Type())
		for _, k := range sortedKeys(s.HV()) {
			f, ok := fields.byName[k]
			if !ok {
				f, ok = fields.byFold[foldKey(k)]
			}
			if !ok {
				continue // a key with no field
			}
			fv, err := fieldByIndex(v, f.index)
			if err != nil {
				return &DecodeError{Path: keyPath(path, k), Msg: err.Error()}
			}
			if err := d.value(s.HV().m[k], fv, keyPath(path, k)); err != nil {
				return err
			}
		}
		return nil
	}
	return fail(v.Type().String() + ", which Decode can't fill")
}

// sortedKeys is h's keys in sorted order, so that Decode takes them in the
// same order every time: when several values don't fit, the error is always
// about the same one.
func sortedKeys(h *HV) []string {
	return slices.Sorted(maps.Keys(h.m))
}

// decodeBool is s as a bool field takes it: a boolean, 0 or 1, or "true"
// or "false".
func decodeBool(s *SV) (bool, bool) {
	switch v := s.val().(type) {
	case bool:
		return v, true
	case string, []byte:
		switch strings.ToLower(strings.TrimSpace(s.Str())) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	case *AV, *HV:
		return false, false
	}
	switch n, ok := s.NumOK(); {
	case ok && n == 1:
		return true, true
	case ok && n == 0:
		return false, true
	}
	return false, false
}

// decodeInt is s as an integer field takes it: a whole number, from a
// number or a numeric string, exactly however large.
func decodeInt(s *SV) (int64, bool) {
	switch v := s.val().(type) {
	case int64:
		return v, true
	case bool, *AV, *HV:
		return 0, false
	case string, []byte:
		if iv, _, isInt, ok := parseNumber(s.Str()); isInt {
			return iv, ok
		}
	}
	f, ok := s.NumOK()
	if !ok || f != math.Trunc(f) || f < math.MinInt64 || f >= math.MaxInt64 {
		return 0, false
	}
	return int64(f), true
}

// fieldInfo is a struct field Decode can fill: where it is, through any
// embedded structs.
type fieldInfo struct {
	index []int
}

// fieldSet is the fields of a struct type, by the name a key must have, and
// by that name folded, for a key that matches only ignoring case.
type fieldSet struct {
	byName map[string]fieldInfo
	byFold map[string]fieldInfo
}

var fieldCache sync.Map // reflect.Type → fieldSet

// structFields lists the fields of t that Decode fills, by the names
// encoding/json gives them and by its rules: the json tag's name, or the
// field's own; the fields of embedded structs promoted; and where several
// fields have a name, the shallowest wins, then the one that's tagged, and
// if that leaves more than one, none does. A name that matches only ignoring
// case goes to the first such field in the struct's order.
func structFields(t reflect.Type) fieldSet {
	if fs, ok := fieldCache.Load(t); ok {
		return fs.(fieldSet)
	}
	type candidate struct {
		index  []int
		depth  int
		tagged bool
	}
	byName := map[string][]candidate{}
	var names []string // in the order first seen
	walking := map[reflect.Type]bool{}
	var walk func(t reflect.Type, index []int, depth int)
	walk = func(t reflect.Type, index []int, depth int) {
		if walking[t] {
			return // a struct that embeds itself: its fields are already here, shallower
		}
		walking[t] = true
		defer delete(walking, t)
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, _, _ := strings.Cut(tag, ",")
			idx := append(append([]int(nil), index...), i)
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if f.Anonymous && name == "" && ft.Kind() == reflect.Struct {
				walk(ft, idx, depth+1) // an embedded struct: promote its fields
				continue
			}
			if !f.IsExported() {
				continue
			}
			tagged := name != ""
			if !tagged {
				name = f.Name
			}
			if _, seen := byName[name]; !seen {
				names = append(names, name)
			}
			byName[name] = append(byName[name], candidate{idx, depth, tagged})
		}
	}
	walk(t, nil, 0)

	// The field each name goes to, if any, and those fields in the struct's
	// order, for the case-insensitive matches.
	type field struct {
		name  string
		index []int
	}
	var chosen []field
	for _, name := range names {
		cs := byName[name]
		depth := slices.MinFunc(cs, func(a, b candidate) int { return a.depth - b.depth }).depth
		var best []candidate
		for _, c := range cs {
			if c.depth == depth {
				best = append(best, c)
			}
		}
		if len(best) > 1 {
			var tagged []candidate
			for _, c := range best {
				if c.tagged {
					tagged = append(tagged, c)
				}
			}
			best = tagged
		}
		if len(best) == 1 {
			chosen = append(chosen, field{name, best[0].index})
		}
	}
	slices.SortFunc(chosen, func(a, b field) int { return slices.Compare(a.index, b.index) })
	fs := fieldSet{byName: map[string]fieldInfo{}, byFold: map[string]fieldInfo{}}
	for _, c := range chosen {
		fs.byName[c.name] = fieldInfo{c.index}
		if k := foldKey(c.name); fs.byFold[k].index == nil {
			fs.byFold[k] = fieldInfo{c.index} // the first in the struct's order
		}
	}
	fieldCache.Store(t, fs)
	return fs
}

// foldKey is s with each character replaced by the smallest that equals it
// ignoring case, so that two names that strings.EqualFold says are equal
// have the same foldKey.
func foldKey(s string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			least = min(least, f)
		}
		return least
	}, s)
}

// fieldByIndex is v.FieldByIndex, allocating any nil embedded pointers on
// the way.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Pointer {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}, fmt.Errorf("can't set the embedded %s, which isn't exported", v.Type().Elem())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v, nil
}
