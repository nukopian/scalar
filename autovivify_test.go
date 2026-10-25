package scalar

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// quietDoc is subscriptDoc with autovivification off, and a clone taken
// before, to compare with after a read.
func quietDoc(t *testing.T) (doc, before *SV) {
	t.Helper()
	doc = newSubscriptDoc(t)
	before = doc.Clone()
	return doc.Autovivify(false), before
}

// unchanged fails if doc isn't what it was: Equal to the clone taken
// before, and the same JSON.
func unchanged(t *testing.T, name string, doc, before *SV) {
	t.Helper()
	if !doc.Equal(before) || mustJSON(t, doc) != mustJSON(t, before) {
		t.Errorf("%s changed the document:\n%s", name, doc.Dump())
	}
}

// With autovivification off, as under Perl's no autovivification, reading
// creates nothing, through any accessor and at any depth, and what's read
// is undef.
func TestAutovivifyOffReadsCreateNothing(t *testing.T) {
	for name, read := range map[string]func(doc *SV) *SV{
		"Get":                       func(doc *SV) *SV { return doc.Get("nope") },
		"Get, two levels":           func(doc *SV) *SV { return doc.Get("nope", "deeper") },
		"Get, a chain":              func(doc *SV) *SV { return doc.Get("nope").Get(0).Get("x") },
		"Get through null":          func(doc *SV) *SV { return doc.Get("properties", "felt", "count") },
		"Get past an array's end":   func(doc *SV) *SV { return doc.Get("stations", 5, 0) },
		"At":                        func(doc *SV) *SV { return doc.At("nope.deeper[2].x") },
		"At through null":           func(doc *SV) *SV { return doc.At("properties.felt.count") },
		"At, either kind":           func(doc *SV) *SV { return doc.At("nope.0.x") },
		"HV of a missing key":       func(doc *SV) *SV { return Value(doc.Get("nope").HV().Keys().Length()) },
		"AV of a missing key":       func(doc *SV) *SV { return Value(doc.Get("nope").AV().Length()) },
		"AV of null, All":           func(doc *SV) *SV { return Value(len(doc.At("properties.felt").AV().StrSlice())) },
		"HV of null, Values":        func(doc *SV) *SV { return doc.At("properties.felt").HV().Values("a", "b").Get(0) },
		"HV of a missing key, Each": func(doc *SV) *SV { k, _, _ := doc.Get("nope").HV().Each(); return Value(k) },
		"Exists":                    func(doc *SV) *SV { return Value(doc.Exists("nope", "deeper")) },
		"Exists through null":       func(doc *SV) *SV { return Value(doc.Exists("properties", "felt", 0)) },
		"Delete":                    func(doc *SV) *SV { return doc.Delete("nope", "deeper") },
		"Delete through null":       func(doc *SV) *SV { return doc.Delete("properties", "felt", "x") },
		"DeleteAt":                  func(doc *SV) *SV { return doc.DeleteAt("nope.deeper") },
		"an array's Get":            func(doc *SV) *SV { return doc.Get("stations").AV().Get(4, 1) },
		"a hash's Get":              func(doc *SV) *SV { return doc.HV().Get("nope", "deeper") },
		"a hash's Exists":           func(doc *SV) *SV { return Value(doc.HV().Exists("nope", "deeper")) },
	} {
		doc, before := quietDoc(t)
		got := read(doc)
		if got.Defined() && got.Ref() != "" || got.Bool() {
			t.Errorf("%s read %s", name, got.Dump())
		}
		unchanged(t, name, doc, before)
	}
}

// The undef that grammar returns for no match reads through as undef,
// rather than turning into a hash.
func TestAutovivifyOffUndefStaysUndef(t *testing.T) {
	m := (&SV{}).Autovivify(false)
	if m.At("made").Defined() || m.Defined() || m.String() != "" || m.Ref() != "" {
		t.Errorf("reading through undef made %s", m.Dump())
	}
}

// A store still creates the levels it needs, as under the pragma, and
// creates nothing else.
func TestAutovivifyOffStoresCreate(t *testing.T) {
	for name, tc := range map[string]struct {
		write func(doc *SV)
		want  string
	}{
		"At, three levels": {func(doc *SV) { doc.At("a.b[1].c").Set(1) }, `{"a":{"b":[null,{"c":1}]}}`},
		"Get, then Set":    {func(doc *SV) { doc.Get("a", "b").Set("x") }, `{"a":{"b":"x"}}`},
		"a chain of Get":   {func(doc *SV) { doc.Get("a").Get(0).Get("b").Set(2) }, `{"a":[{"b":2}]}`},
		"Push":             {func(doc *SV) { doc.Get("a").AV().Push(1, 2) }, `{"a":[1,2]}`},
		"HV().Set":         {func(doc *SV) { doc.Get("a").HV().Set("b", 1) }, `{"a":{"b":1}}`},
		"HV() twice":       {func(doc *SV) { a := doc.Get("a"); h1, h2 := a.HV(), a.HV(); h1.Set("b", 1); h2.Set("c", 2) }, `{"a":{"b":1,"c":2}}`},
		"HV(), one by one": {func(doc *SV) { doc.Get("a").HV().Set("b", 1); doc.Get("a").HV().Set("c", 2) }, `{"a":{"b":1,"c":2}}`},
		"AV() twice":       {func(doc *SV) { a := doc.Get("a"); a1, a2 := a.AV(), a.AV(); a1.Push(1); a2.Push(2) }, `{"a":[1,2]}`},
		"read, then write": {func(doc *SV) { s := doc.At("a.b"); doc.At("x.y"); s.Set(1) }, `{"a":{"b":1}}`},
		"Unshift":          {func(doc *SV) { doc.Get("a").AV().Unshift(1) }, `{"a":[1]}`},
		"Splice":           {func(doc *SV) { doc.Get("a").AV().Splice(0, 0, 1) }, `{"a":[1]}`},
		"SetLastIndex":     {func(doc *SV) { doc.Get("a").AV().SetLastIndex(1) }, `{"a":[null,null]}`},
	} {
		doc := Value(map[string]any{}).Autovivify(false)
		tc.write(doc)
		if got := mustJSON(t, doc); got != tc.want {
			t.Errorf("%s: %s, want %s", name, got, tc.want)
		}
	}
}

// Values stored later take the setting of the container they go into, so
// the whole tree stays as it was set, however it grows.
func TestAutovivifyOffInherited(t *testing.T) {
	quake := map[string]any{"properties": map[string]any{"mag": 4.6}, "stations": []any{map[string]any{"net": "CI"}}}
	for name, store := range map[string]func(doc *SV) *SV{
		"HV.Set":        func(doc *SV) *SV { doc.HV().Set("q", quake); return doc.Get("q") },
		"Get().Set":     func(doc *SV) *SV { doc.Get("q").Set(quake); return doc.Get("q") },
		"At().Set":      func(doc *SV) *SV { doc.At("x.q").Set(quake); return doc.At("x.q") },
		"AV.Set":        func(doc *SV) *SV { doc.Get("list").AV().Set(0, quake); return doc.Get("list", 0) },
		"Push":          func(doc *SV) *SV { doc.Get("list").AV().Push(Value(quake)); return doc.Get("list", 0) },
		"Unshift":       func(doc *SV) *SV { doc.Get("list").AV().Unshift(Value(quake)); return doc.Get("list", 0) },
		"Splice":        func(doc *SV) *SV { doc.Get("list").AV().Splice(0, 0, Value(quake)); return doc.Get("list", 0) },
		"UnmarshalJSON": func(doc *SV) *SV { mustUnmarshal(t, doc.Get("q"), mustJSON(t, quake)); return doc.Get("q") },
		"FromJSON, Set": func(doc *SV) *SV {
			q, _ := FromJSON([]byte(mustJSON(t, quake)))
			doc.HV().Set("q", q)
			return doc.Get("q")
		},
	} {
		doc := Value(map[string]any{}).Autovivify(false)
		q := store(doc)
		before := doc.Clone()
		for _, path := range []string{"properties.depth.km", "stations[0].sta.code", "stations[3].net", "nope.x"} {
			q.At(path)
		}
		q.Get("properties").Exists("nope", "x")
		if !doc.Equal(before) {
			t.Errorf("%s: reading the stored value changed it:\n%s", name, doc.Dump())
		}
		if q.Autovivifies() || q.At("stations[0]").Autovivifies() || q.Get("properties").HV().Autovivifies() {
			t.Errorf("%s: the stored value doesn't have the setting", name)
		}
	}
}

func mustUnmarshal(t *testing.T, s *SV, data string) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), s); err != nil {
		t.Fatal(err)
	}
}

// Autovivify takes the receiver's setting and the setting below it apart,
// and with one argument sets both.
func TestAutovivifyLevels(t *testing.T) {
	newDoc := func() *SV { return Value(map[string]any{"a": map[string]any{"b": nil}}) }
	created := func(doc *SV, path string) bool {
		before := doc.Clone()
		doc.At(path)
		return !doc.Equal(before)
	}
	for _, tc := range []struct {
		args          []bool
		self, deeply  bool
		top, belowTop bool // whether reading through each creates
	}{
		{nil, true, true, true, true},
		{[]bool{true}, true, true, true, true},
		{[]bool{false}, false, false, false, false},
		{[]bool{true, false}, true, false, true, false},
		{[]bool{false, true}, false, true, false, true},
	} {
		doc := newDoc().Autovivify(false).Autovivify(tc.args...)
		if doc.Autovivifies() != tc.self || doc.AutovivifiesDeeply() != tc.deeply {
			t.Errorf("Autovivify%v: Autovivifies %v, AutovivifiesDeeply %v", tc.args, doc.Autovivifies(), doc.AutovivifiesDeeply())
		}
		// Through the receiver: an undef receiver dereferenced.
		u := (&SV{}).Autovivify(tc.args...)
		u.Get("x")
		if got := u.Defined(); got != tc.top {
			t.Errorf("Autovivify%v: reading through an undef receiver created %v", tc.args, got)
		}
		// Below it: a level inside the hash it refers to.
		if got := created(doc, "a.b.c"); got != tc.belowTop {
			t.Errorf("Autovivify%v: reading below created %v", tc.args, got)
		}
	}
	// A hash's or an array's own level is its elements.
	h := newDoc().HV().Autovivify(false, true)
	if h.Autovivifies() || !h.AutovivifiesDeeply() || h.Get("a").Autovivifies() || !h.Get("a").AutovivifiesDeeply() {
		t.Error("HV.Autovivify(false, true): the elements' settings")
	}
	h.Get("x", "y")
	h.Get("a", "b", "c")
	if h.Exists("x") || h.At("a.b").Ref() != "HASH" {
		t.Errorf("HV.Autovivify(false, true): %s", mustJSON(t, h))
	}
	a := Array(nil, 1).Autovivify(false)
	a.Get(0, "x")
	a.Get(5, 0)
	if mustJSON(t, a) != `[null,1]` || a.Autovivifies() || a.AutovivifiesDeeply() {
		t.Errorf("AV.Autovivify(false): %s", mustJSON(t, a))
	}
	expectPanic(t, "three arguments", "Autovivify takes at most two", func() { doc := newDoc(); doc.Autovivify(true, false, true) })
	var nilSV *SV
	var nilAV *AV
	var nilHV *HV
	if !nilSV.Autovivifies() || !nilAV.AutovivifiesDeeply() || !nilHV.Autovivifies() {
		t.Error("nil autovivifies, as the default")
	}
}

// Autovivify(true) puts back the default, all the way down.
func TestAutovivifyBackOn(t *testing.T) {
	doc, _ := quietDoc(t)
	doc.Autovivify()
	doc.At("properties.x.y")
	if !doc.At("properties").Exists("x") {
		t.Errorf("Autovivify() didn't turn it back on:\n%s", doc.Dump())
	}
}

// A structure that refers back to itself is set once, and the receiver
// keeps its own setting when the walk comes back to it.
func TestAutovivifyCycle(t *testing.T) {
	h := Hash("mag", 4.6)
	h.Set("self", Value(h))
	h.Autovivify(false, true)
	if h.Autovivifies() || !h.AutovivifiesDeeply() {
		t.Error("the cycle back to the receiver changed its setting")
	}
	doc := Value(h).Autovivify(false)
	doc.At("self.self.nope.x")
	if h.Exists("nope") {
		t.Error("reading round a cycle created something")
	}
}

// Clone drops the setting, as it drops a lock; Copy is shallow, so a copied
// reference shares the hash and its setting.
func TestAutovivifyCloneAndCopy(t *testing.T) {
	doc, _ := quietDoc(t)
	c := doc.Clone()
	if !c.Autovivifies() || !c.Get("properties").Autovivifies() {
		t.Error("Clone kept the setting")
	}
	c.At("properties.x.y")
	if !c.At("properties").Exists("x") {
		t.Error("a clone doesn't autovivify")
	}
	cp := doc.Copy()
	cp.At("properties.z.y")
	if doc.At("properties").HV().Exists("z") {
		t.Error("a copy's shared hash autovivified")
	}
}

// With Lock as well, a missing key reads as undef rather than panicking,
// and writes still panic.
func TestAutovivifyOffLocked(t *testing.T) {
	doc, before := quietDoc(t)
	doc.Lock()
	if doc.Get("nope").Defined() || doc.At("nope.x[0]").Defined() || doc.Exists("nope", "x") || doc.Get("properties", "nope").Defined() {
		t.Error("reading a locked structure with autovivification off")
	}
	unchanged(t, "reading it locked", doc, before)
	expectPanic(t, "a new key", "Attempt to access disallowed key 'nope'", func() { doc.Get("nope").Set(1) })
	expectPanic(t, "a new level", "Attempt to access disallowed key 'nope'", func() { doc.At("nope.x").Set(1) })
	expectPanic(t, "a value", readOnly, func() { doc.Get("tsunami").Set(1) })
}

// Reading JSON or YAML into a locked scalar is an error, not a panic, as
// Unmarshal's callers expect; the scalar is left as it was.
func TestUnmarshalIntoLocked(t *testing.T) {
	for name, read := range map[string]func(s *SV) error{
		"json.Unmarshal":   func(s *SV) error { return json.Unmarshal([]byte(`{"mag":4.6}`), s) },
		"UnmarshalJSON":    func(s *SV) error { return s.UnmarshalJSON([]byte(`[1, 2]`)) },
		"the slow path":    func(s *SV) error { return s.UnmarshalJSON([]byte(`{"mag": 1e400}`)) },
		"yaml.Unmarshal":   func(s *SV) error { return yaml.Unmarshal([]byte("mag: 4.6\n"), s) },
		"a locked element": func(s *SV) error { return json.Unmarshal([]byte(`1`), Array(1).Lock().Get(3)) },
	} {
		s := Value("CI").Lock()
		err := read(s)
		if err == nil || !strings.Contains(err.Error(), readOnly) && !strings.Contains(err.Error(), "read-only") {
			t.Errorf("%s: error %v", name, err)
		}
		if s.Str() != "CI" {
			t.Errorf("%s: changed the scalar to %s", name, s.Dump())
		}
	}
}

// YAML read into a missing element is stored, as JSON is.
func TestUnmarshalYAMLIntoElement(t *testing.T) {
	h := Hash()
	if err := yaml.Unmarshal([]byte("mag: 4.6\n"), h.Get("q")); err != nil {
		t.Fatal(err)
	}
	if got := mustJSON(t, h); got != `{"q":{"mag":4.6}}` {
		t.Errorf("got %s", got)
	}
}

// With autovivification off, Get, Exists and Delete, and the Set that
// creates, do what the same subscripts do in Perl under the CPAN pragma's
// no autovivification, to the value and to the document. It needs perl
// with autovivification, and is skipped without it.
func TestAutovivifyOffMatchesPerl(t *testing.T) {
	needPerl(t)
	if exec.Command("perl", "-Mautovivification", "-e1").Run() != nil {
		t.Skip("autovivification isn't installed (cpanm autovivification, or set PERL5LIB)")
	}
	type testCase struct {
		Code string `json:"code"`
	}
	ops, paths := subscriptCases(t)
	cases := make([]testCase, len(ops))
	for n, op := range ops {
		expr := "$s"
		for _, k := range paths[n] {
			switch k := k.(type) {
			case string:
				expr += "->{'" + k + "'}"
			case int:
				expr += fmt.Sprintf("->[%d]", k)
			}
		}
		code := map[string]string{
			"get":    "$got = " + expr,
			"set":    "$got = (" + expr + ` = "set")`,
			"exists": "$got = (exists " + expr + ") ? JSON::PP::true : JSON::PP::false",
			"delete": "$got = delete " + expr,
		}[op]
		cases[n] = testCase{code}
	}
	in, _ := json.Marshal(map[string]any{"doc": json.RawMessage(subscriptDoc), "cases": cases})
	const script = `
use v5.36; use JSON::PP;
my $json = JSON::PP->new->canonical->allow_nonref;
my $in = decode_json(do { local $/; <STDIN> });
my @out;
for my $c (@{ $in->{cases} }) {
	my $s = $json->decode($json->encode($in->{doc})); # a fresh copy
	my $got;
	my $lived = eval "no autovivification; $c->{code}; 1";
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
	counts := map[string]int{}
	for n, op := range ops {
		path := paths[n]
		counts[op]++
		doc := newSubscriptDoc(t).Autovivify(false)
		var got any
		died := func() (died bool) {
			defer func() { died = recover() != nil }()
			switch op {
			case "get":
				got = doc.Get(path...)
			case "set":
				got = doc.Get(path...).Set("set")
			case "exists":
				got = doc.Exists(path...)
			case "delete":
				got = doc.Delete(path...)
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
			t.Errorf("%s: panicked %v, Perl died %v", cases[n].Code, died, perlDied)
		} else if !reflect.DeepEqual(asAny(gotJSON), want[n][0]) || !reflect.DeepEqual(asAny(docJSON), want[n][1]) {
			t.Errorf("%s:\n got %s, %s\nwant %v, %v", cases[n].Code, gotJSON, docJSON, want[n][0], want[n][1])
		}
		if want[n][2] == 1.0 {
			counts[op+" dies"]++
		}
	}
	t.Logf("%d cases: %v", len(cases), counts)
}

// Setting a structure the size of a grammar match, a dozen hashes,
// allocates nothing, where a map of the containers seen allocates past
// eight.
func TestAutovivifySmallDoesntAllocate(t *testing.T) {
	h := Hash("~", "2026-10-25", "from", 0, "to", 10)
	for _, k := range []string{"year", "month", "day", "zone", "a", "b", "c", "d", "e", "f", "g"} {
		h.Set(k, map[string]any{"~": "2026", "from": 0, "to": 4})
	}
	s := Value(h)
	if n := testing.AllocsPerRun(100, func() { s.Autovivify(false) }); n != 0 {
		t.Errorf("%v allocations", n)
	}
}

// Past the containers the walk keeps in an array, it keeps them in a
// map: a large structure, with a shared part and a loop, is set all the
// way down, each part once.
func TestAutovivifyLarge(t *testing.T) {
	shared := Hash("net", "CI")
	a := Array()
	for i := 0; i < 100; i++ {
		a.Push(Value(map[string]any{"mag": i, "station": shared, "depth": map[string]any{"km": 2}}))
	}
	doc := Value(map[string]any{"features": a})
	shared.Set("doc", doc) // a loop back to the top
	doc.Autovivify(false, true)
	if doc.Autovivifies() || !doc.AutovivifiesDeeply() {
		t.Error("the receiver's own setting changed")
	}
	doc.Autovivify(false)
	before := doc.Get("features", 99, "depth").Clone()
	for i := range 100 {
		doc.At(fmt.Sprintf("features[%d].depth.error.x", i))
		doc.At(fmt.Sprintf("features[%d].station.doc.nope.x", i))
	}
	if doc.Get("features").AV().Length() != 100 || !doc.Get("features", 99, "depth").Equal(before) || shared.Exists("nope") || doc.Exists("nope") {
		t.Error("reading a large structure with autovivification off created something")
	}
}
