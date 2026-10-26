package scalar

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sameJSON reports how a and b differ, kinds included: 1 and 1.0 differ.
func sameJSON(a, b *SV) string {
	if a.Kind() != b.Kind() {
		return fmt.Sprintf("kind %v vs %v", a.Kind(), b.Kind())
	}
	switch a.Kind() {
	case ArrayRef:
		x, y := a.AV(), b.AV()
		if x.Length() != y.Length() {
			return fmt.Sprintf("length %d vs %d", x.Length(), y.Length())
		}
		for i := range x.Length() {
			if d := sameJSON(x.Get(i), y.Get(i)); d != "" {
				return fmt.Sprintf("[%d] %s", i, d)
			}
		}
	case HashRef:
		x, y := a.HV(), b.HV()
		if x.Length() != y.Length() {
			return fmt.Sprintf("keys %v vs %v", x.KeysSlice(), y.KeysSlice())
		}
		for k, v := range x.All() {
			if !y.Exists(k) {
				return fmt.Sprintf("key %q missing", k)
			}
			if d := sameJSON(v, y.Get(k)); d != "" {
				return fmt.Sprintf("{%q} %s", k, d)
			}
		}
	case Float:
		if a.Num() != b.Num() || a.Str() != b.Str() {
			return fmt.Sprintf("%v vs %v", a.Num(), b.Num())
		}
	default:
		if a.Str() != b.Str() {
			return fmt.Sprintf("%q vs %q", a.Str(), b.Str())
		}
	}
	return ""
}

// fromJSONBoth decodes data with the JSON reader and with encoding/json.
func fromJSONBoth(data []byte) (fast *SV, fastOK bool, slow *SV, slowErr error) {
	fast, fastOK = readJSON(data)
	slow = &SV{}
	slowErr = slow.unmarshalJSONSlow(data)
	return
}

// checkJSON fails t if the reader accepted data and gave anything other
// than what encoding/json gives.
func checkJSON(t *testing.T, data []byte) (fastOK bool) {
	t.Helper()
	fast, ok, slow, err := fromJSONBoth(data)
	if !ok {
		return false
	}
	if err != nil {
		t.Fatalf("%q: reader accepted it, encoding/json said %v", data, err)
	}
	if d := sameJSON(fast, slow); d != "" {
		t.Fatalf("%q: %s", data, d)
	}
	return true
}

func TestReadJSONSame(t *testing.T) {
	for _, js := range []string{
		`null`, `true`, `false`, `0`, `-0`, `1`, `-17`, `1.5`, `-0.25`, `1e3`, `1E-2`, `2.5e+10`,
		`9223372036854775807`, `9223372036854775808`, `-9223372036854775808`, `-9223372036854775809`,
		`123456789012345678901234567890`, `0.1`, `1.0`, `100`,
		`""`, `"abc"`, `"≈12.5 km – ≈15 km"`, `"tab\there"`, `"q\"uote"`, `"back\\slash"`, `"\/"`,
		`"\b\f\n\r\t"`, `"£"`, `"€"`, `"😀"`, `"mixed é é"`,
		`[]`, `{}`, `[1,"a",null,true,[],{}]`, `{"a":1,"b":[1,2,{"c":null}],"d":{"e":"f"}}`,
		`{"a":1,"a":2}`, ` { "sp" : [ 1 , 2 ] } `, "\n\t[1]\r\n",
		`{"":"empty key"}`, `{"k":"v","n":{"m":{"o":[[[1]]]}}}`,
		// Not accepted by the reader, or not JSON at all: encoding/json decides.
		`"\ud800"`, `"\udc00x"`, "\"bad\xffutf8\"", `[1,]`, `{"a":1,}`, `01`, `1.`, `.5`, `-`, `+1`,
		`tru`, `nul`, `[1] [2]`, `{"a"}`, `{a:1}`, `"unterminated`, `[`, ``, `   `, "\"ctl\x01\"",
		`"\x"`, `"\u12"`, `1e`, `1e+`, `[1 2]`, `{"a":1 "b":2}`, `truex`, `nullnull`,
	} {
		checkJSON(t, []byte(js))
	}
}

// Ordinary documents take the fast path: a real feed, in testdata/usgs.
func TestReadJSONTakesFeeds(t *testing.T) {
	paths, _ := filepath.Glob("testdata/usgs/*.json")
	if len(paths) == 0 {
		t.Fatal("no feeds in testdata/usgs")
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !checkJSON(t, data) {
			t.Errorf("%s: the reader handed it to encoding/json", p)
		}
	}
}

// Random documents, valid and broken, agree.
func TestReadJSONRandom(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 11))
	pieces := []string{`{`, `}`, `[`, `]`, `,`, `:`, `"a"`, `"é\n"`, `"é"`, `1`, `-2.5e3`, `true`, `null`, ` `, `"k"`, `0`, `"\\"`}
	var gen func(depth int) string
	gen = func(depth int) string {
		switch r := rng.IntN(10); {
		case depth > 4 || r < 3:
			return []string{`1`, `-0.5`, `"s"`, `true`, `null`, `"€"`, `12345678901234567890`}[rng.IntN(7)]
		case r < 6:
			var items []string
			for range rng.IntN(4) {
				items = append(items, gen(depth+1))
			}
			return "[" + strings.Join(items, ",") + "]"
		default:
			var items []string
			for i := range rng.IntN(4) {
				items = append(items, fmt.Sprintf(`"k%d":%s`, i%3, gen(depth+1)))
			}
			return "{" + strings.Join(items, ",") + "}"
		}
	}
	for range 2000 {
		checkJSON(t, []byte(gen(0)))
		var b strings.Builder
		for range rng.IntN(8) {
			b.WriteString(pieces[rng.IntN(len(pieces))])
		}
		checkJSON(t, []byte(b.String()))
	}
}

// Deep nesting is left to encoding/json, which limits it.
func TestReadJSONDeep(t *testing.T) {
	deep := strings.Repeat("[", 20000) + strings.Repeat("]", 20000)
	if _, ok := readJSON([]byte(deep)); ok {
		t.Error("the reader took 20,000 levels of nesting")
	}
	if _, err := FromJSON([]byte(deep)); err == nil {
		t.Error("FromJSON accepted 20,000 levels of nesting")
	}
}

func FuzzReadJSON(f *testing.F) {
	for _, s := range []string{`{"a":[1,2.5,"x",null,true]}`, `"😀"`, `[1,]`, "\"\xff\"", `-0e0`, `[12345678901234567890,-123456789012345678901234]`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		checkJSON(t, data)
	})
}

// Push gives the same array with or without its fast path.
func TestPushFastPathSame(t *testing.T) {
	ref := Value(Array(1))
	for _, vals := range [][]any{
		{1}, {"a", 2, 3.5, true, nil}, {ref}, {Value("x"), &SV{}}, {[]byte("b")}, {int64(9)},
		{Array(1, 2)}, {Hash("k", 1)}, {[]string{"p"}}, {1, Array(2)}, // these flatten: no fast path
	} {
		fast, slow := withFastPaths(func() string { return Array("start").Push(vals...).Dump() })
		if fast != slow {
			t.Errorf("Push(%v):\nfast %s\nslow %s", vals, fast, slow)
		}
	}
	v := Value("orig")
	a := Array().Push(v)
	v.Set("changed")
	if a.Get(0).Str() != "orig" {
		t.Error("Push kept the caller's scalar instead of a copy")
	}
}
