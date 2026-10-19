package scalar

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// subscriptDoc is a small earthquake event, with a hash, an array, an
// array of arrays, a null and a key that looks like a number.
const subscriptDoc = `{
	"properties": {"mag": 4.6, "place": "12 km SSW of Ridgecrest, CA", "felt": null, "0": "zero"},
	"geometry": {"type": "Point", "coordinates": [-117.6, 35.7, 8.2]},
	"stations": [["CI", "CLC"], ["NC", "MCB"]],
	"tsunami": 0
}`

func newSubscriptDoc(t *testing.T) *SV {
	t.Helper()
	doc, err := FromJSON([]byte(subscriptDoc))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// Get is HV().Get or AV().Get in one step, chosen by the type of the key.
func TestSVGet(t *testing.T) {
	doc := newSubscriptDoc(t)
	if got := doc.Get("properties").Get("place").Str(); got != "12 km SSW of Ridgecrest, CA" {
		t.Errorf("hash then hash: %q", got)
	}
	if got := doc.Get("geometry").Get("coordinates").Get(-1).Num(); got != 8.2 {
		t.Errorf("negative index: %v", got)
	}
	if got := doc.Get("stations").Get(1).Get(0).Str(); got != "NC" {
		t.Errorf("array of arrays: %q", got)
	}
	if got := doc.Get("properties").Get("0").Str(); got != "zero" {
		t.Errorf(`the key "0": %q`, got)
	}

	// The element itself, not a copy: the same *SV as HV().Get and Path.
	if doc.Get("properties") != doc.HV().Get("properties") || doc.Get("stations").Get(0) != doc.Path("stations", 0) {
		t.Error("Get should give the stored element")
	}
	doc.Get("properties").Get("mag").Set(4.7)
	if got := doc.At("properties.mag").Num(); got != 4.7 {
		t.Errorf("writing through Get: %v", got)
	}

	// A missing key reads as undef without being created...
	if v := doc.Get("properties").Get("cdi"); v.Defined() || doc.At("properties").HV().Exists("cdi") {
		t.Error("reading a missing key created it")
	}
	// ...until it's written to, and levels on the way autovivify.
	doc.Get("properties").Get("cdi").Set(3.4)
	doc.Get("products").Get("shakemap").Get(0).Get("status").Set("reviewed")
	if doc.At("properties.cdi").Num() != 3.4 || doc.At("products.shakemap[0].status").Str() != "reviewed" {
		t.Errorf("writing autovivifies: %s", doc.Dump())
	}

	// An undef s autovivifies, as $s->{key} does in Perl even when read.
	var h, a SV
	h.Get("felt")
	a.Get(0)
	if h.Ref() != "HASH" || a.Ref() != "ARRAY" || h.HV().Length() != 0 || a.AV().Length() != 0 {
		t.Errorf("undef s: %s %s", h.Dump(), a.Dump())
	}

	// Reading through the wrong kind of value gives undef; writing panics.
	if doc.Get("tsunami").Get("x").Defined() || doc.Get("properties").Get(0).Defined() || doc.Get("stations").Get("x").Defined() {
		t.Error("reading through the wrong type should give undef")
	}
	expectPanic(t, "write through a number", "Not a HASH reference", func() { doc.Get("tsunami").Get("x").Set(1) })
	expectPanic(t, "write through a hash by index", notArrayRef, func() { doc.Get("properties").Get(0).Set(1) })
	expectPanic(t, "write before the start", "non-creatable", func() { doc.Get("stations").Get(-10).Set(1) })

	// A nil *SV reads as undef, as Path does.
	var nilSV *SV
	if nilSV.Get("x").Defined() || nilSV.Get(0).Defined() {
		t.Error("nil SV")
	}

	// A locked hash panics on a disallowed key, as HV.Get does.
	locked := Value(Hash("mag", 4.6))
	locked.HV().Lock()
	if locked.Get("mag").Num() != 4.6 {
		t.Error("locked hash, allowed key")
	}
	expectPanic(t, "locked hash", "disallowed key 'depth'", func() { locked.Get("depth") })

	for _, key := range []any{4.5, int64(1), nil, []string{"a"}, MustPath("a")} {
		expectPanic(t, fmt.Sprintf("Get(%T)", key), fmt.Sprintf("Get key must be a string or an int, got %T", key), func() { doc.Get(key) })
	}
}

// Exists is Perl's exists, autovivifying and dying where Perl does.
func TestSVExists(t *testing.T) {
	doc := newSubscriptDoc(t)
	props, coords := doc.Get("properties"), doc.At("geometry.coordinates")
	for _, c := range []struct {
		s    *SV
		key  any
		want bool
	}{
		{props, "mag", true},
		{props, "felt", true}, // null is there, but undef
		{props, "cdi", false},
		{props, "0", true},
		{coords, 0, true},
		{coords, -1, true},
		{coords, -3, true},
		{coords, -4, false}, // before the start
		{coords, 3, false},  // past the end
	} {
		if got := c.s.Exists(c.key); got != c.want {
			t.Errorf("Exists(%#v) on %s: %v, want %v", c.key, c.s.Dump(), got, c.want)
		}
	}

	// A hole doesn't exist.
	holes := Value(Array(1, 2, 3))
	holes.AV().Delete(1)
	if holes.Exists(1) || !holes.Exists(2) {
		t.Errorf("holes: %s", holes.Dump())
	}

	// An undef s autovivifies, as in Perl, into the kind the key asks for.
	var h, a SV
	if h.Exists("felt") || a.Exists(0) || h.Ref() != "HASH" || a.Ref() != "ARRAY" {
		t.Errorf("undef s became %s %s", h.Dump(), a.Dump())
	}
	// So do the levels on the way, but never the key itself.
	before := doc.Dump()
	props.Exists("cdi")
	if doc.Dump() != before {
		t.Error("Exists created the key")
	}
	if doc.Get("products").Exists("shakemap") || doc.At("products").Ref() != "HASH" || doc.At("products").HV().Length() != 0 {
		t.Errorf("exists $doc->{products}{shakemap}: %s", doc.Dump())
	}

	// Anything else panics, as Perl dies, and is left as it was.
	before = doc.Dump()
	for _, c := range []struct {
		name string
		s    *SV
		key  any
		want string
	}{
		{"a number", doc.Get("tsunami"), "x", "Not a HASH reference"},
		{"a number by index", doc.Get("tsunami"), 0, notArrayRef},
		{"a hash by index", props, 0, notArrayRef},
		{"an array by key", coords, "0", "Not a HASH reference"},
		{"through a number", doc.Get("tsunami").Get("x"), "y", "Not a HASH reference"},
		{"before the start", doc.Get("stations").Get(-10), 0, "non-creatable"},
		{"a nil SV", nil, "x", "Not a HASH reference"},
	} {
		expectPanic(t, "Exists on "+c.name, c.want, func() { c.s.Exists(c.key) })
	}
	if doc.Dump() != before {
		t.Errorf("a panicking Exists changed the document: %s", doc.Dump())
	}

	// A locked hash answers without panicking, as Perl's exists does.
	locked := Value(Hash("mag", 4.6))
	locked.HV().Lock()
	if !locked.Exists("mag") || locked.Exists("depth") {
		t.Error("locked hash")
	}

	expectPanic(t, "Exists(float64)", "Exists key must be a string or an int, got float64", func() { doc.Exists(1.0) })
	expectPanic(t, "nil SV, bad key", "Exists key must be", func() { (*SV)(nil).Exists(1.0) })
}

// Delete is Perl's delete, autovivifying and dying where Perl does.
func TestSVDelete(t *testing.T) {
	doc := newSubscriptDoc(t)
	props := doc.Get("properties")
	if got := props.Delete("mag"); got.Num() != 4.6 || props.Exists("mag") {
		t.Errorf("delete a key: %v", got)
	}
	if got := props.Delete("cdi"); got.Defined() || props.Exists("cdi") {
		t.Errorf("delete a missing key: %v", got)
	}

	// In the middle, a hole; at the end, the array shrinks past the holes.
	coords := doc.At("geometry.coordinates")
	if got := coords.Delete(1); got.Num() != 35.7 || coords.Exists(1) || coords.AV().Length() != 3 {
		t.Errorf("delete in the middle: %v, %s", got, coords.Dump())
	}
	if got := coords.Delete(-1); got.Num() != 8.2 || coords.AV().Length() != 1 {
		t.Errorf("delete the last: %v, %s", got, coords.Dump())
	}
	if coords.Delete(-5).Defined() || coords.Delete(7).Defined() || coords.AV().Length() != 1 {
		t.Error("deleting out of range")
	}

	// The deleted value is the element itself, as HV.Delete gives.
	st := doc.Get("stations").Get(0)
	if doc.Get("stations").Delete(0) != st {
		t.Error("Delete should give the element that was stored")
	}

	// An undef s autovivifies, as in Perl, even for an index before the start.
	var h, a, neg SV
	if h.Delete("felt").Defined() || a.Delete(0).Defined() || neg.Delete(-1).Defined() {
		t.Error("deleting from undef")
	}
	if h.Ref() != "HASH" || a.Ref() != "ARRAY" || neg.Ref() != "ARRAY" {
		t.Errorf("undef s became %s %s %s", h.Dump(), a.Dump(), neg.Dump())
	}
	doc.Get("products").Delete("shakemap")
	if doc.At("products").Ref() != "HASH" {
		t.Errorf("delete $doc->{products}{shakemap}: %s", doc.Dump())
	}

	// Anything else panics, as Perl dies, and is left as it was.
	before := doc.Dump()
	expectPanic(t, "a number", "Not a HASH reference", func() { doc.Get("tsunami").Delete("x") })
	expectPanic(t, "a hash by index", notArrayRef, func() { props.Delete(0) })
	expectPanic(t, "an array by key", "Not a HASH reference", func() { coords.Delete("0") })
	expectPanic(t, "before the start", "non-creatable", func() { doc.Get("stations").Get(-10).Delete("x") })
	var nilSV *SV
	expectPanic(t, "a nil SV", notArrayRef, func() { nilSV.Delete(0) })
	if doc.Dump() != before || !props.Exists("0") || doc.Get("tsunami").Int() != 0 {
		t.Errorf("a panicking Delete changed the document: %s", doc.Dump())
	}

	// A locked hash panics, as HV.Delete and Perl do.
	locked := Value(Hash("mag", 4.6))
	locked.HV().Lock()
	expectPanic(t, "locked hash", "Attempt to delete readonly key 'mag'", func() { locked.Delete("mag") })

	expectPanic(t, "Delete(bool)", "Delete key must be a string or an int, got bool", func() { doc.Delete(true) })
}

// Get, Exists and Delete are the one-step forms of Path, PathOK and
// DeletePath, on every kind of value and key, once Get has autovivified s
// as Exists and Delete would. Where s is the wrong kind, Exists and Delete
// panic, and PathOK and DeletePath find nothing.
func TestSVSubscriptMatchesPath(t *testing.T) {
	keys := []any{"properties", "mag", "felt", "0", "nope", 0, 1, -1, -2, 2, 5, -9}
	for _, at := range []string{"", "properties", "properties.felt", "geometry.coordinates", "stations", "stations[1]", "tsunami", "missing"} {
		for _, k := range keys {
			doc1, doc2 := newSubscriptDoc(t), newSubscriptDoc(t)
			s1, s2 := doc1.At(at), doc2.At(at)
			if d := s1.Get(k).Diff(s2.Path(k)); d != "" || doc1.Dump() != doc2.Dump() {
				t.Errorf("%q Get(%v) vs Path: %s", at, k, d)
			}
			_, isIndex := k.(int)
			if s1.Ref() != map[bool]string{false: "HASH", true: "ARRAY"}[isIndex] {
				expectPanic(t, fmt.Sprintf("%q Exists(%v)", at, k), "Not a", func() { s1.Exists(k) })
				expectPanic(t, fmt.Sprintf("%q Delete(%v)", at, k), "Not a", func() { s1.Delete(k) })
				if _, ok := s2.PathOK(k); ok || s2.DeletePath(k).Defined() || doc1.Dump() != doc2.Dump() {
					t.Errorf("%q PathOK or DeletePath(%v) found something", at, k)
				}
				continue
			}
			_, ok := s2.PathOK(k)
			if s1.Exists(k) != ok {
				t.Errorf("%q Exists(%v) = %v, PathOK %v", at, k, !ok, ok)
			}
			if d := s1.Delete(k).Diff(s2.DeletePath(k)); d != "" || doc1.Dump() != doc2.Dump() {
				t.Errorf("%q Delete(%v) vs DeletePath: %s\n%s\n%s", at, k, d, doc1.Dump(), doc2.Dump())
			}
		}
	}
}

// A chain of Get, ending in a read, a write, Exists or Delete, does what
// the same subscripts do in Perl, to the value and to the document, and
// panics where Perl dies. It leaves out reads that go through the wrong
// kind of value, where Perl dies and scalar reads undef, as documented. It
// needs perl, and is skipped without it.
func TestSVSubscriptMatchesPerl(t *testing.T) {
	needPerl(t)
	type step struct {
		Key   string `json:"k"`
		Index bool   `json:"i,omitempty"`
	}
	type testCase struct {
		Op   string `json:"op"`
		Path []step `json:"path"`
	}
	keys := []any{"properties", "mag", "felt", "0", "nope", "coordinates", "geometry", 0, 1, -1, -3, 2, 4, -9}
	toKeys := func(path []any) []step {
		var st []step
		for _, k := range path {
			switch k := k.(type) {
			case string:
				st = append(st, step{Key: k})
			case int:
				st = append(st, step{Key: fmt.Sprint(k), Index: true})
			}
		}
		return st
	}
	// fits says whether a get is one Perl does without dying; scalar's
	// reads carry on where Perl dies, as documented. Anything else fits.
	fits := func(op string, path []any) bool {
		if op != "get" {
			return true
		}
		cur, curOK := newSubscriptDoc(t), true // cur exists so far
		for i, k := range path {
			last := i == len(path)-1
			if !curOK || !cur.Defined() {
				if n, isInt := k.(int); isInt && n < 0 && !last {
					return false // $a[-1] of an empty array can't be created
				}
				curOK = false
				continue
			}
			switch k := k.(type) {
			case string:
				if cur.Ref() != "HASH" {
					return false
				}
			case int:
				if cur.Ref() != "ARRAY" {
					return false
				}
				if !last && (k < -cur.AV().Length()) {
					return false // going through an element Perl can't create
				}
			}
			cur, curOK = cur.PathOK(k)
		}
		return true
	}
	var cases []testCase
	var paths [][]any
	for _, op := range []string{"get", "set", "exists", "delete"} {
		for _, a := range keys {
			for _, b := range append([]any{nil}, keys...) {
				for _, c := range append([]any{nil}, keys...) {
					path := []any{a}
					if b != nil {
						path = append(path, b)
						if c != nil {
							path = append(path, c)
						}
					} else if c != nil {
						continue
					}
					if fits(op, path) {
						cases = append(cases, testCase{op, toKeys(path)})
						paths = append(paths, path)
					}
				}
			}
		}
	}
	if len(cases) < 1000 {
		t.Fatalf("only %d cases", len(cases))
	}
	in, _ := json.Marshal(map[string]any{"doc": json.RawMessage(subscriptDoc), "cases": cases})
	const script = `
use v5.36; use JSON::PP;
my $json = JSON::PP->new->canonical->allow_nonref;
my $in = decode_json(do { local $/; <STDIN> });
my @out;
for my $c (@{ $in->{cases} }) {
	my $s = $json->decode($json->encode($in->{doc})); # a fresh copy
	my @path = @{ $c->{path} };
	my $last = pop @path;
	my ($k, $i) = ($last->{k}, $last->{i});
	my $got;
	my $lived = eval {
		my $ref = \$s;
		# \ on an element creates it, as going through it would anyway
		$ref = $_->{i} ? \ $$ref->[$_->{k}] : \ $$ref->{$_->{k}} for @path;
		if    ($c->{op} eq 'get')    { $got = $i ? $$ref->[$k] : $$ref->{$k} }
		elsif ($c->{op} eq 'set')    { $got = ($i ? $$ref->[$k] : $$ref->{$k}) = "set" }
		elsif ($c->{op} eq 'exists') { $got = ($i ? exists $$ref->[$k] : exists $$ref->{$k}) ? JSON::PP::true : JSON::PP::false }
		else                         { $got = $i ? delete $$ref->[$k] : delete $$ref->{$k} }
		1;
	};
	push @out, $lived ? [$got, $s, 0] : [undef, $s, 1];
}
print $json->encode(\@out);
`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(string(in))
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
		t.Fatalf("perl: %v: %s", err, stderr)
	}
	var want [][3]any
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(cases) {
		t.Fatalf("perl gave %d results for %d cases", len(want), len(cases))
	}
	asAny := func(b []byte) any {
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	for n, c := range cases {
		path := paths[n]
		doc := newSubscriptDoc(t)
		cur := doc
		for _, k := range path[:len(path)-1] {
			cur = cur.Get(k)
		}
		k := path[len(path)-1]
		var got any
		died := func() (died bool) {
			defer func() { died = recover() != nil }()
			switch c.Op {
			case "get":
				got = cur.Get(k)
			case "set":
				got = cur.Get(k).Set("set")
			case "exists":
				got = cur.Exists(k)
			case "delete":
				got = cur.Delete(k)
			}
			return false
		}()
		gotJSON, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		docJSON, err := doc.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if perlDied := want[n][2] == 1.0; died != perlDied {
			t.Errorf("%s %v: panicked %v, Perl died %v", c.Op, path, died, perlDied)
		} else if !reflect.DeepEqual(asAny(gotJSON), want[n][0]) || !reflect.DeepEqual(asAny(docJSON), want[n][1]) {
			t.Errorf("%s %v:\n got %s, %s\nwant %v, %v", c.Op, path, gotJSON, docJSON, want[n][0], want[n][1])
		}
	}
	counts := map[string]int{}
	for n, c := range cases {
		counts[c.Op]++
		if want[n][2] == 1.0 {
			counts[c.Op+" dies"]++
		}
	}
	t.Logf("%d cases: %v", len(cases), counts)
}
