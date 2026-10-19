# scalar: examples

Every function and method in [scalar](../README.md), each with a working
example and the output it prints. For how the features fit together, see
the [reference](REFERENCE.md); for tasks, see the [how-to guide](HOWTO.md);
coming from Perl, see the [phrasebook](PERL.md).

<!-- reference:start -->

Generated from the examples in `example_*_test.go`, which `go test` runs, so every
example here works and prints what it says. Edit the examples, not this section, then run
`go test . -run TestReference -update`.

- [Scalars](#reference-scalars)
- [Strings](#reference-strings)
- [Operators](#reference-operators)
- [Arrays](#reference-arrays)
- [Hashes](#reference-hashes)
- [List utilities](#reference-list-utilities)
- [Sets and bags](#reference-sets-and-bags)
- [Regular expressions](#reference-regular-expressions)
- [Pack and unpack](#reference-pack-and-unpack)
- [JSON and JSONPath](#reference-json-and-jsonpath)
- [YAML](#reference-yaml)
- [Into Go values](#reference-into-go-values)

### Reference: Scalars

#### `Value`

Value returns a scalar holding x.

```go
fmt.Println(scalar.Value(42).Kind() == scalar.Int)
fmt.Println(scalar.Value(3.5).Kind() == scalar.Float)
fmt.Println(scalar.Value("text").Kind() == scalar.String)
fmt.Println(scalar.Value([]string{"a", "b"}).Ref()) // a slice becomes an array reference
fmt.Println(scalar.Value(nil).Defined())            // nil is undef
```

```
true
true
true
ARRAY
false
```

#### `Value.struct`

A struct becomes a hash, keyed as encoding/json would write it: by its json tags, leaving out what's omitempty and empty. Decode is the way back.

```go
type station struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}
st := scalar.Value(station{Code: "PAS"})
fmt.Println(st.Ref(), st.At("code"), st.At("name").Defined())
```

```
HASH PAS false
```

#### `SV.Set`

Set is Perl's \$s = x: it assigns x to s and returns s.

```go
s := scalar.Value("old")
s.Set(10)                     // $s = 10
fmt.Println(s.Add(1))         // it's a number now
s.Set(map[string]any{"a": 1}) // a map becomes a hash reference
fmt.Println(s.Ref())
s.Get("a").Set(2) // $s->{a} = 2: subscript, then assign
fmt.Println(s.Get("a"))
```

```
11
HASH
2
```

#### `SV.Copy`

Copy returns a new scalar holding s's value, as Perl's my \$x = \$s does.

```go
h := scalar.Hash("count", 1)
alias := h.Get("count")      // the stored value itself
own := h.Get("count").Copy() // my $own = $h{count}
alias.Set(2)
own.Set(99)
fmt.Println(h.Get("count"))
```

```
2
```

#### `SV.Clone`

Clone returns a deep copy of s, as Storable's dclone does: arrays and hashes it refers to are copied too, all the way down, so changing the clone never changes the original.

```go
h := scalar.Hash("defaults", scalar.Value(map[string]any{"tags": []any{"go"}}))
shallow := h.Get("defaults").Copy() // my $row = $h{defaults}: the same hash
deep := h.Get("defaults").Clone()   // my $row = dclone($h{defaults}): its own
deep.At("tags").AV().Push("perl")
fmt.Println(h.Get("defaults").At("tags").AV().Join(","), deep.At("tags").AV().Join(","))
shallow.At("tags").AV().Push("sql") // changes the original
fmt.Println(h.Get("defaults").At("tags").AV().Join(","))
```

```
go go,perl
go,sql
```

#### `SV.Equal`

Equal reports whether s and other hold the same data, all the way down, as Test::More's is_deeply decides it.

```go
doc, _ := scalar.FromJSON([]byte(`{"stations": ["PAS", "GSC"], "depth": {"max": 12}}`))
fmt.Println(doc.Equal(map[string]any{ // is_deeply($doc, {...})
	"stations": []string{"PAS", "GSC"},
	"depth":    map[string]any{"max": "12"}, // as strings, as eq compares
}))
fmt.Println(doc.At("depth.max").Equal(12.0), scalar.Value(nil).Equal("")) // undef equals only undef
```

```
true
true false
```

#### `SV.Diff`

Diff says where s and other first differ, as is_deeply reports it, or "" if they're equal (see Equal).

```go
ours, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"]}`))
theirs, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "ISA"]}`))
fmt.Println(ours.Diff(theirs)) // where they first differ, as a path for At
fmt.Printf("%q\n", ours.Diff(ours.Clone()))
```

```
stations[1]: got "GSC", want "ISA"
""
```

#### `SV.Apply`

Apply calls f on every plain value in s, all the way down, and stores what f returns in its place, as Data::Rmap's rmap { \$_ = ...

```go
doc, _ := scalar.FromJSON([]byte(`{"place": " 5 km W of Cobb, CA ", "stations": [" PAS", "GSC "], "depth": 2}`))
doc.Apply(func(v *scalar.SV) any { // rmap { $_ = trim $_ } $doc
	if v.Kind() == scalar.String {
		return v.Trim()
	}
	return v // anything else as it is
})
out, _ := doc.MarshalJSON()
fmt.Println(string(out))
```

```
{"depth":2,"place":"5 km W of Cobb, CA","stations":["PAS","GSC"]}
```

#### `SV.Kind`

Kind reports what s currently holds.

```go
for _, x := range []any{nil, true, 1, 1.5, "s", []byte{0x89}, []any{}, map[string]any{}} {
	switch scalar.Value(x).Kind() {
	case scalar.Undef:
		fmt.Println("undef")
	case scalar.Bool:
		fmt.Println("bool")
	case scalar.Int:
		fmt.Println("int")
	case scalar.Float:
		fmt.Println("float")
	case scalar.String:
		fmt.Println("string")
	case scalar.Bytes:
		fmt.Println("bytes")
	case scalar.ArrayRef:
		fmt.Println("array")
	case scalar.HashRef:
		fmt.Println("hash")
	}
}
```

```
undef
bool
int
float
string
bytes
array
hash
```

#### `SV.Defined`

Defined is Perl's defined().

```go
fmt.Println(scalar.Value(nil).Defined())
fmt.Println(scalar.Value("").Defined()) // empty, but defined
fmt.Println(scalar.Hash().Get("missing").Defined())
```

```
false
true
false
```

#### `SV.Any`

Any returns s as a plain Go value, in the shapes encoding/json uses: undef is nil; booleans, integers, floats and strings are bool, int64, float64 and string; array references become []any and hash references map[string]any, converted all the way down.

```go
s := scalar.Value(map[string]any{"tags": []any{"go", 1}})
plain := s.Any().(map[string]any) // plain Go, sharing nothing with s
fmt.Println(plain["tags"])
```

```
[go 1]
```

#### `SV.Or`

Or is Perl's defined-or, \$s // \$def: a new scalar holding s's value if s is defined, otherwise def's.

```go
fmt.Println(scalar.Value(nil).Or("default")) // $x // "default"
fmt.Println(scalar.Value(0).Or("default"))   // 0 is defined, so it's kept
```

```
default
0
```

#### `SV.OrSet`

OrSet is Perl's \$s //= \$def: if s is undef, it sets s to def.

```go
h := scalar.Hash()
h.Get("retries").OrSet(3) // $h{retries} //= 3
h.Get("retries").OrSet(5) // already set, so unchanged
fmt.Println(h.Get("retries"))
```

```
3
```

#### `SV.Then`

Then calls f with s if s is defined, and returns what f returns as a new scalar; if s is undef, f isn't called, and Then returns undef.

```go
digits := func(v *scalar.SV) any { return v.S(`,`, "", "gr") }
figure := func(v *scalar.SV) any { return v.M(`([\d,]+) m\b`).Get(1).Then(digits) } // undef if no figure
km := func(v *scalar.SV) any { return v.Div(1000) }
for _, text := range []string{"Depth: 12,500 m", "Depth not determined"} {
	fmt.Println(scalar.Value(text).Then(figure).Then(km).Or("unknown")) // stops at the first undef
}
```

```
12.5
unknown
```

#### `SV.When`

When returns a copy of s if s is defined and pred(s) is true, and undef otherwise, without calling pred on undef.

```go
positive := func(v *scalar.SV) bool { return v.Num() > 0 }
doc, _ := scalar.FromJSON([]byte(`{"felt": 1204, "tsunami": 0}`))
fmt.Println(doc.At("felt").When(positive).Or("none"))
fmt.Println(doc.At("tsunami").When(positive).Or("none")) // fails the test: undef
fmt.Println(doc.At("cdi").When(positive).Or("none"))     // missing: positive isn't called
```

```
1204
none
none
```

#### `SV.Ref`

Ref is Perl's ref(): "ARRAY", "HASH", or "" for non-references.

```go
fmt.Println(scalar.Value(scalar.Array(1)).Ref())
fmt.Println(scalar.Value(scalar.Hash()).Ref())
fmt.Printf("%q\n", scalar.Value("text").Ref())
```

```
ARRAY
HASH
""
```

#### `SV.Str`

Str returns s in string context.

```go
fmt.Println(scalar.Value(42).Str())
fmt.Println(scalar.Value(0.1 + 0.2).Str()) // Perl's 15 significant digits
fmt.Println(scalar.Value(true).Str())
fmt.Printf("%q\n", scalar.Value(nil).Str())
```

```
42
0.3
1
""
```

#### `SV.String`

String implements `fmt.Stringer`.

```go
fmt.Println(scalar.Value(1e21)) // fmt uses String, as Perl's print would
```

```
1e+21
```

#### `SV.Dump`

Dump returns s as Data::Dumper::Concise prints it: Data::Dumper with Terse, Indent = 1, Useqq, Quotekeys = 0, Sortkeys and Trailingcomma, so nested arrays and hashes are laid out one element to a line, hash keys are sorted and quoted only when they need to be, and strings are double-quoted, with escapes for anything that isn't printable ASCII.

```go
doc, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"], "mag": {"value": 5.1, "error": null}}`))
fmt.Println(doc.Dump()) // print Dumper($doc), as Data::Dumper::Concise prints it
```

```
{
  id => "ci40123456",
  mag => {
    error => undef,
    value => "5.1",
  },
  stations => [
    "PAS",
    "GSC",
  ],
}
```

#### `FromDump`

FromDump reads a structure written by Dump, or by Perl's Data::Dumper, and returns it, as FromJSON reads JSON.

```go
stations := scalar.Value(scalar.Array("PAS"))
doc := scalar.Hash("id", "ci40123456", "stations", stations, "again", stations)      // the same array twice
back, err := scalar.FromDump(scalar.Value(doc).Dump())                               // a round trip
fmt.Println(err, back.Equal(doc), back.At("stations").AV() == back.At("again").AV()) // still one array

perl, err := scalar.FromDump(`bless( {'code' => 'PAS', 'channels' => ['BHZ', 'BHN']}, 'Station' )`) // Data::Dumper's output
fmt.Println(err, perl.At("code"), perl.At("channels[1]"))
```

```
<nil> true true
<nil> PAS BHN
```

#### `MustFromDump`

MustFromDump is FromDump, panicking if the dump is malformed.

```go
want := scalar.MustFromDump(`{
	id       => "ci40123456",
	stations => ["PAS", "GSC"],   # test data, as Perl would write it
}`)
got, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"]}`))
fmt.Printf("%q\n", got.Diff(want))
```

```
""
```

#### `MustFromDump.testData`

```go
// The expected value in a test: shorter than JSON, with comments where
// they help.
want := scalar.MustFromDump(`{
	id       => "ci40123456",
	place    => "12 km NNE of Ridgecrest, CA",
	mag      => { value => 5.1, error => undef },   # error: not reported
	stations => ["PAS", "GSC", "ISA"],
	reviewed => !!1,
}`)
got, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "place": "12 km NNE of Ridgecrest, CA",
	"mag": {"value": 5.1, "error": null}, "stations": ["PAS", "GSC", "ISA"], "reviewed": true}`))
fmt.Printf("%q\n", got.Diff(want))
got.At("stations[2]").Set("SVD")
fmt.Println(got.Diff(want)) // where they differ, as a path for At
```

```
""
stations[2]: got "SVD", want "ISA"
```

#### `MustFromDump.toStruct`

```go
// Test data straight into the struct a function under test takes.
type Origin struct {
	Lat, Lon float64
	Depth    int // metres
}
type Event struct {
	ID       string
	Origin   Origin
	Stations []string
}
e, err := scalar.To[Event](scalar.MustFromDump(`{
	ID       => 'ci40123456',
	Origin   => { Lat => 35.7, Lon => -117.6, Depth => 8_200 },   # underscores, as in Perl
	Stations => ['PAS', 'GSC'],
}`))
fmt.Printf("%+v %v\n", e, err)
```

```
{ID:ci40123456 Origin:{Lat:35.7 Lon:-117.6 Depth:8200} Stations:[PAS GSC]} <nil>
```

#### `FromDump.keysAndStrings`

```go
doc := scalar.MustFromDump(`{
	bare_word   => 1,
	'single'    => 'it\'s \n literal',   # only \\ and \' are escapes here
	"double"    => "tab\there\x{263a}",   # Perl's escapes
	42          => "a number as a key",
	-1          => "so is a negative one",
	"log.level" => "debug",
	email       => "jo@example.com",       # $ and @ aren't interpolated
}`)
fmt.Println(doc.HV().Keys().Sort(nil).Join(" "))
fmt.Printf("%q\n", doc.At("single").Str())
fmt.Printf("%q\n", doc.At("double").Str())
fmt.Println(doc.At("{42}"), "|", doc.At("{-1}"), "|", doc.At("{log.level}"), "|", doc.At("email"))
```

```
-1 42 bare_word double email log.level single
"it's \\n literal"
"tab\there☺"
a number as a key | so is a negative one | debug | jo@example.com
```

#### `FromDump.bytes`

```go
doc := scalar.MustFromDump(`["\211PNG\r\n", "caf\x{e9}", "caf\351"]`)
for _, s := range doc.AV().All() {
	fmt.Println(s.Kind() == scalar.Bytes, s.Length()) // octal above 127: bytes; \x{...}: text
}
fmt.Println(doc.At("[1]").Eq(doc.At("[2]"))) // the same characters, so eq
```

```
true 6
false 4
true 4
true
```

#### `FromDump.shared`

```go
// $VAR1 paths are the arrays and hashes already read: shared parts, and loops.
doc := scalar.MustFromDump(`{
	defaults => { cpu => "500m" },
	web      => { limits => $VAR1->{defaults} },
	worker   => { limits => $VAR1->{defaults} },
	parent   => $VAR1,
}`)
doc.At("defaults.cpu").Set("1")
fmt.Println(doc.At("web.limits.cpu"), doc.At("worker.limits.cpu")) // one hash, seen from two places
fmt.Println(doc.At("parent.parent.parent.web.limits.cpu"))         // a loop back to the top
```

```
1 1
1
```

#### `FromDump.fromPerl`

```go
// Data::Dumper's default layout, pasted from a Perl debugging session,
// with a blessed object: read as the plain hash it is.
dump := `bless( {
                 'name' => 'Ada',
                 'langs' => [
                              'Perl',
                              'Go'
                            ],
                 'since' => 1843
               }, 'Person' )`
p, err := scalar.FromDump(dump)
fmt.Println(err, p.At("name"), p.At("langs").AV().Join("/"), p.At("since").Add(183))
```

```
<nil> Ada Perl/Go 2026
```

#### `FromDump.types`

```go
// A dump doesn't record types: each value comes back as its text says,
// as Perl reads a dump back.
doc := scalar.MustFromDump(`[42, "42", "2.5", 2.5, "12345678901", !!1, undef]`)
names := map[scalar.Kind]string{scalar.Int: "int", scalar.String: "string", scalar.Float: "float",
	scalar.Bool: "bool", scalar.Undef: "undef"}
fmt.Println(doc.AV().Map(func(v *scalar.SV) any { return names[v.Kind()] }).Join(" "))
fmt.Println(scalar.Value("42").Dump(), scalar.Value(2.5).Dump()) // so Dump's "42" reads back as 42, and 2.5 as "2.5"
```

```
int string string float string bool undef
42 "2.5"
```

#### `FromDump.errors`

```go
_, err := scalar.FromDump(`{
	id       => "ci40123456",
	stations => ["PAS" "GSC"],
}`)
fmt.Println(err) // a line and column to look at
_, err = scalar.FromDump(`{ handler => sub { "DUMMY" } }`)
fmt.Println(err) // data only: no Perl is ever run
```

```
scalar: bad dump at line 3, column 22: expected a comma or "]", found '"'
scalar: bad dump at line 1, column 14: sub can't be read: FromDump reads data, not Perl code
```

#### `SV.Bytes`

Bytes returns s as a Go []byte: a byte string's bytes, or anything else's bytes in string context (see Str), so text gives its UTF-8 encoding.

```go
png := scalar.Value([]byte("\x89PNG\r\n\x1a\n..."))   // a []byte makes a byte string
fmt.Println(png.Kind() == scalar.Bytes, png.Length()) // Length counts bytes
fmt.Println(png.M(`^\x89PNG`) != nil)                 // patterns match bytes
fmt.Printf("%x\n", png.Substr(0, 4))                  // %x gives the bytes in hex
blob := png.Bytes()                                   // back to a Go []byte (a copy)
fmt.Println(len(blob))
fmt.Println(scalar.Value("café").Bytes()) // text gives its UTF-8 bytes
```

```
true 11
true
89504e47
11
[99 97 102 195 169]
```

#### `SV.Format`

Format implements `fmt.Formatter`, so that Go's format verbs treat s as Perl's sprintf does, in the context each verb implies: %d, %b, %o, %x, %X and %c in integer context (see Int), %e, %E, %f, %F, %g and %G in numeric context (see Num), %t in boolean context, and %s, %q, %v and anything else in string context (see Str).

```go
fmt.Printf("%d|%05.2f|%x|%5s|%t\n",
	scalar.Value("42 apples"), scalar.Value("3.14159"), scalar.Value(255),
	scalar.Value("ab"), scalar.Value("0"))
```

```
42|03.14|ff|   ab|false
```

#### `SV.Num`

Num returns s in numeric context.

```go
fmt.Println(scalar.Value("3.5kg").Num()) // the numeric prefix, as Perl does
fmt.Println(scalar.Value("abc").Num())
```

```
3.5
0
```

#### `SV.NumOK`

NumOK is Num, plus false wherever Perl would warn: undef, empty or non-numeric strings, and strings with trailing garbage.

```go
for _, x := range []any{"42", " 42 ", "42abc", "", nil} {
	n, ok := scalar.Value(x).NumOK()
	fmt.Println(n, ok) // ok is false wherever Perl would warn
}
```

```
42 true
42 true
42 false
0 false
0 false
```

#### `SV.Int`

Int returns s in integer context, truncating toward zero.

```go
fmt.Println(scalar.Value(-3.9).Int()) // truncates toward zero
fmt.Println(scalar.Value("12 items").Int())
```

```
-3
12
```

#### `SV.IntOK`

IntOK is Int, plus false wherever NumOK would be false, and for NaN or floats outside the int64 range (which are clamped).

```go
i, ok := scalar.Value("9223372036854775807").IntOK() // exact, however large
fmt.Println(i, ok)
i, ok = scalar.Value(1e30).IntOK() // out of range: clamped
fmt.Println(i, ok)
```

```
9223372036854775807 true
9223372036854775807 false
```

#### `SV.Bool`

Bool returns s in boolean context: undef, "", "0", 0 and 0.0 are false; everything else, including "0.0" and "00", is true.

```go
for _, x := range []any{nil, "", "0", 0, 0.0, "0.0", "00", " ", "a"} {
	fmt.Print(scalar.Value(x).Bool(), " ")
}
fmt.Println()
```

```
false false false false false true true true true
```

#### `SV.AV`

AV dereferences s as an array, as Perl's @\$s does, for what works on the whole array, such as Push or Length; for one element, `s.Get`(i) needs no AV.

```go
var s scalar.SV   // undef
s.AV().Push(1, 2) // push @$s, 1, 2: autovivifies an array
fmt.Println(s.AV().Join(","))
fmt.Println(scalar.Value("text").AV() == nil) // not an array
```

```
1,2
true
```

#### `SV.HV`

HV dereferences s as a hash, as Perl's %\$s does, for what works on the whole hash, such as Keys or Length; for one key, `s.Get`(key) needs no HV.

```go
var s scalar.SV
s.HV().Set("a", 1) // $s->{a} = 1
fmt.Println(s.Ref(), s.Get("a"))
```

```
HASH 1
```

#### `SV.Get`

Get is Perl's \$s->{key} or \$s->[i]: a string key looks up the hash s refers to, and an int the array.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"properties": {"place": "12 km SSW of Ridgecrest, CA", "mag": 4.6},
	"geometry": {"coordinates": [-117.6, 35.7, 8.2]}
}`))
fmt.Println(doc.Get("properties", "place"))         // $doc->{properties}{place}
fmt.Println(doc.Get("geometry", "coordinates", -1)) // $doc->{geometry}{coordinates}[-1]: depth
fmt.Println(doc.Get("properties").Get("mag"))       // the same, a step at a time
route := []any{"geometry", "coordinates", 0}        // a route built from data
fmt.Println(doc.Get(route...))
doc.Get("properties", "mag").Set(4.7)              // the element itself, so this changes doc
doc.Get("products", "shakemap", 0).Set("reviewed") // writing autovivifies
fmt.Println(doc.At("properties.mag"), doc.At("products.shakemap[0]"))
```

```
12 km SSW of Ridgecrest, CA
8.2
4.6
-117.6
4.7 reviewed
```

#### `SV.Exists`

Exists is Perl's exists \$s->{key} or exists \$s->[i]: whether the hash or array s refers to has that key or element, even if its value is undef.

```go
doc, _ := scalar.FromJSON([]byte(`{"felt": null, "stations": ["PAS", "GSC"]}`))
fmt.Println(doc.Exists("felt"), doc.Exists("cdi"))                // null is there; cdi isn't
fmt.Println(doc.Exists("stations", 1), doc.Exists("stations", 2)) // exists $doc->{stations}[2]
var none scalar.SV
fmt.Println(none.Exists("felt"), none.Ref()) // as in Perl, an undef becomes a hash
```

```
true false
true false
false HASH
```

#### `SV.Delete`

Delete is Perl's delete \$s->{key} or delete \$s->[i]: it removes the key or element from the hash or array s refers to, and returns its value, or undef if there was nothing there.

```go
doc, _ := scalar.FromJSON([]byte(`{"mag": 4.6, "magError": 0.1, "stations": ["PAS", "GSC", "ISA"]}`))
fmt.Println(doc.Delete("magError"))      // delete $doc->{magError}: its value
fmt.Println(doc.Delete("stations", -1))  // delete $doc->{stations}[-1]: the array shrinks
fmt.Println(doc.Delete("cdi").Defined()) // nothing there: undef
out, _ := doc.MarshalJSON()
fmt.Println(string(out))
```

```
0.1
ISA
false
{"mag":4.6,"stations":["PAS","GSC"]}
```

#### `SV.GetOK`

GetOK is Get with Go's comma-ok: ok reports whether every step exists, so a value that's present but null (undef) gives ok true, while a missing key or index, or a step through something that isn't an array or hash, gives ok false and an undef scalar.

```go
doc, _ := scalar.FromJSON([]byte(`{"alert": null}`))
_, ok := doc.GetOK("alert") // present, but null
fmt.Println(ok)
_, ok = doc.GetOK("felt") // missing: and nothing is created
fmt.Println(ok, doc.Exists("felt"))
```

```
true
false false
```

#### `SV.At`

At is Get with the route as one string, in dot, bracket or Perl's brace form:

```go
doc, _ := scalar.FromJSON([]byte(`{
	"content": [{"rule_set": "A"}, {"rule_set": "B"}],
	"meta": {"log.level": "debug", "0": "zero"}
}`))
fmt.Println(doc.At("content[0].rule_set"))         // $doc->{content}[0]{rule_set}
fmt.Println(doc.At("content.1.rule_set"))          // a number after a dot: an index into an array
fmt.Println(doc.At("content[-1].rule_set"))        // from the end
fmt.Println(doc.At(`meta["log.level"]`))           // a key with a dot in it, quoted
fmt.Println(doc.At("meta.0"), doc.At(`meta["0"]`)) // into a hash, the number is a key
fmt.Println(doc.At("{content}[1]{rule_set}"))      // Perl's braces: always a hash key
fmt.Println(doc.At("meta{log.level}"))             // inside braces, dots are part of the key

var s scalar.SV
s.At("rules[0].name").Set("first") // writing autovivifies: [0] makes an array
out, _ := s.MarshalJSON()
fmt.Println(string(out))
```

```
A
B
B
debug
zero zero
B
debug
{"rules":[{"name":"first"}]}
```

#### `SV.AtOK`

AtOK is GetOK with the route as one string, written as for At: ok reports whether every step exists, so a value that's present but null gives ok true.

```go
doc, _ := scalar.FromJSON([]byte(`{"mag": {"value": 5.1, "error": null}}`))
_, ok := doc.AtOK("mag.error") // present, but null
fmt.Println(ok)
_, ok = doc.AtOK("mag.type") // missing: and nothing is created
fmt.Println(ok, doc.At("mag").Exists("type"))
```

```
true
false false
```

#### `SV.DeleteAt`

DeleteAt is Perl's delete \$s->{content}[0]{rule_set}, with the route written as for At: it removes the last step's key or element, and returns its value, or undef if there was nothing there.

```go
doc, _ := scalar.FromJSON([]byte(`{"felt": {"reports": 1204, "source": "DYFI"}, "stations": ["PAS", "GSC"]}`))
fmt.Println(doc.DeleteAt("felt.source"))  // delete $doc->{felt}{source}: its value
fmt.Println(doc.DeleteAt("stations[-1]")) // the last element: the array shrinks
fmt.Println(doc.DeleteAt("origin.depth")) // nothing there: undef, and nothing created
fmt.Println(doc.Dump())
```

```
DYFI
GSC

{
  felt => {
    reports => 1204,
  },
  stations => [
    "PAS",
  ],
}
```

#### `CheckPath`

CheckPath returns an error if path is malformed, and nil if At, AtOK and DeleteAt can take it, for paths that come from outside the program, which would otherwise panic as a bad regex does.

```go
fmt.Println(scalar.CheckPath("properties.mag"))    // a path from outside the program: check it
fmt.Println(scalar.CheckPath("properties..mag"))   // At would panic on this
fmt.Println(scalar.CheckPath(`properties["mag"]`)) // quoted keys are fine
```

```
<nil>
scalar: bad path "properties..mag" at offset 11: expected a name
<nil>
```

#### `StrCmp`

StrCmp is Perl's cmp, for use with `AV.Sort`.

```go
fmt.Println(scalar.Array("b", "10", "a", "9").Sort(scalar.StrCmp).Join(" ")) // sort { $a cmp $b }
```

```
10 9 a b
```

#### `NumCmp`

NumCmp is Perl's <=>, for use with `AV.Sort`.

```go
fmt.Println(scalar.Array("10", 9, "2.5").Sort(scalar.NumCmp).Join(" ")) // sort { $a <=> $b }
```

```
2.5 9 10
```

#### `SV.DeletePath`

DeletePath is DeleteAt with the route as Get takes it: keys taken whole, dots and all, so it suits keys that come from data, as delete \$s->{\$section}{\$key} does in Perl.

```go
doc, _ := scalar.FromJSON([]byte(`{"depth":{"km":8,"error":2},"v1.2":{"note":"x"}}`))
fmt.Println(doc.DeletePath("depth", "error")) // delete $doc->{depth}{error}
fmt.Println(doc.DeletePath("v1.2", "note"))   // keys are taken whole, dots and all
fmt.Println(doc.At("depth").HV().KeysSlice(), doc.DeletePath("depth", "type").Defined())
```

```
2
x
[km] false
```

#### `SV.Undef`

Undef is Perl's undef \$s: s becomes undef, and Undef returns it, now undef.

```go
h := scalar.Hash("felt", 1204)
fmt.Println(h.Get("felt").Undef().Defined()) // undef $h{felt}: returns it, now undef
fmt.Println(h.Exists("felt"))                // the key stays, as in Perl; Delete removes it
```

```
false
true
```

#### `SV.Lock`

Lock makes s, and everything it refers to, read-only, all the way down, and returns s; see the reference's Locking section.

Lock makes a structure read-only all the way down, as Perl's Hash::Util does a hash: its keys are fixed, so even reading one it hasn't got panics, where it would have autovivified, and a misspelled key is caught at once.

```go
doc, _ := scalar.FromJSON([]byte(`{"place": "5 km W of Cobb, CA", "depth": {"km": 2}}`))
doc.Lock()
fmt.Println(doc.At("depth.km"))
fmt.Println(catch(func() { doc.At("depth.error").Set(0.5) }))
fmt.Println(catch(func() { doc.At("plcae") })) // misspelled
```

```
2
scalar: Attempt to access disallowed key 'error' in a restricted hash
scalar: Attempt to access disallowed key 'plcae' in a restricted hash
```

#### `SV.Unlock`

Unlock undoes Lock, all the way down, and returns s.

Unlock undoes Lock, all the way down.

```go
doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": 2}}`))
doc.Lock().Unlock()
doc.At("depth.error").Set(0.5)
fmt.Println(doc.At("depth.error"))
```

```
0.5
```

#### `SV.Locked`

Locked reports whether Lock has made s read-only.

Locked reports whether Lock has made a value read-only; a Clone isn't.

```go
doc := scalar.Value(map[string]any{"a": 1}).Lock()
fmt.Println(doc.Locked(), doc.Clone().Locked())
```

```
true false
```

### Reference: Strings

#### `SV.Length`

Length is Perl's length(\$s): the number of characters (not bytes) in s in string context, so Value(12345).Length() is 5 and "café" is 4.

```go
fmt.Println(scalar.Value("café").Length()) // characters, not bytes
fmt.Println(scalar.Value(12345).Length())  // in string context
```

```
4
5
```

#### `SV.Substr`

Substr is Perl's substr(\$s, offset) or substr(\$s, offset, length): a new scalar holding part of s in string context, counted in characters.

```go
s := scalar.Value("12 km NNE of Ridgecrest, CA")
fmt.Println(s.Substr(13))           // substr($s, 13)
fmt.Println(s.Substr(13, 10))       // substr($s, 13, 10)
fmt.Println(s.Substr(-2))           // from the end
fmt.Println(s.Substr(0, -15))       // all but the last 15
fmt.Println(s.Substr(99).Defined()) // entirely outside: undef
```

```
Ridgecrest, CA
Ridgecrest
CA
12 km NNE of
false
```

#### `SV.SubstrReplace`

SubstrReplace is Perl's 4-argument substr(\$s, offset, length, replacement), or assigning to substr: it replaces that part of s with replacement (in string context), changing s in place, and returns the part it replaced.

```go
s := scalar.Value("12 km NNE of Ridgecrest, CA")
old := s.SubstrReplace(0, 2, "15") // substr($s, 0, 2) = "15"
fmt.Println(s, "/", old)
```

```
15 km NNE of Ridgecrest, CA / 12
```

#### `SV.Index`

Index is Perl's index(\$s, substr[, position]): the character position of the first occurrence of substr at or after position (default 0), or -1.

```go
s := scalar.Value("café au lait")
fmt.Println(s.Index("au"))   // a character position, ready for Substr
fmt.Println(s.Index("a", 2)) // starting from position 2
fmt.Println(s.Index("tea"))
```

```
5
5
-1
```

#### `SV.RIndex`

RIndex is Perl's rindex(\$s, substr[, position]): the character position of the last occurrence of substr starting at or before position (default the end of the string), or -1.

```go
s := scalar.Value("a/b/c")
fmt.Println(s.RIndex("/"))    // the last one
fmt.Println(s.RIndex("/", 2)) // the last one at or before position 2
```

```
3
1
```

#### `SV.Lc`

Lc is Perl's lc: s in lower case, using Unicode's full case mappings.

```go
fmt.Println(scalar.Value("ÉCOLE Go").Lc())
```

```
école go
```

#### `SV.Uc`

Uc is Perl's uc: s in upper case, using Unicode's full case mappings, so "straße" becomes "STRASSE".

```go
fmt.Println(scalar.Value("straße").Uc()) // full Unicode mappings
```

```
STRASSE
```

#### `SV.Fc`

Fc is Perl's fc: s case-folded, for comparing strings regardless of case.

```go
a, b := scalar.Value("Straße"), scalar.Value("STRASSE")
fmt.Println(a.Lc().Eq(b.Lc())) // lc isn't enough for this
fmt.Println(a.Fc().Eq(b.Fc())) // fc is: compare case-insensitively with it
```

```
false
true
```

#### `SV.UcFirst`

UcFirst is Perl's ucfirst: s with its first character in title case.

```go
fmt.Println(scalar.Value("élan vital").UcFirst())
```

```
Élan vital
```

#### `SV.LcFirst`

LcFirst is Perl's lcfirst: s with its first character in lower case.

```go
fmt.Println(scalar.Value("HELLO").LcFirst())
```

```
hELLO
```

#### `SV.Trim`

Trim is Perl's trim (builtin::trim, Perl 5.36): s without leading or trailing whitespace, including Unicode spaces such as no-break space.

```go
fmt.Printf("%q\n", scalar.Value("\u00a0 padded \t").Trim()) // Unicode spaces too
```

```
"padded"
```

#### `QuoteMeta`

QuoteMeta is Perl's quotemeta: x in string context with every character that could mean something in a pattern backslash-escaped, so that it matches literally.

```go
fmt.Println(scalar.QuoteMeta("M4.5+ (USGS)"))
alert := scalar.Value("Alerts for M4.5+ quakes")
fmt.Println(alert.M(scalar.QuoteMeta("M4.5+")) != nil) // a literal match
```

```
M4\.5\+\ \(USGS\)
true
```

#### `SV.Interpolate`

Interpolate fills in a template from s, as Perl interpolates variables in a double-quoted string, with the keys of the hash s holds as the variables:

```go
doc, _ := scalar.FromJSON([]byte(`{"mag": 5.1, "place": "Ridgecrest, CA",
	"stations": ["PAS", "GSC"], "depth": {"km": 8}}`))
fmt.Println(doc.Interpolate("M$mag near $place: @stations"))   // doc's keys are the variables
fmt.Println(doc.Interpolate("Depth $depth{km} km"))            // subscripts, as in Perl
fmt.Println(doc.Interpolate("Depth ${depth.km} km"))           // any At path in ${...}
fmt.Println(doc.Interpolate(`[$cdi] \$0 alerts\@example.com`)) // missing is ""; \$ and \@ are literal
```

```
M5.1 near Ridgecrest, CA: PAS GSC
Depth 8 km
Depth 8 km
[] $0 alerts@example.com
```

#### `HV.Interpolate`

Interpolate fills in a template from h; see `SV.Interpolate`.

```go
row := scalar.Hash("name", "Ada", "langs", scalar.Value([]string{"Go", "Perl"}))
fmt.Println(row.Interpolate("$name writes @langs"))
```

```
Ada writes Go Perl
```

#### `MustTemplate`

MustTemplate is CompileTemplate, panicking if the template is malformed.

```go
var summary = scalar.MustTemplate("$place. Stations: @stations.") // parsed once
doc, _ := scalar.FromJSON([]byte(`{"place": "Ridgecrest, CA", "stations": ["PAS", "GSC"]}`))
fmt.Println(doc.Interpolate(summary))
```

```
Ridgecrest, CA. Stations: PAS GSC.
```

#### `CompileTemplate`

CompileTemplate compiles a template for Interpolate, or returns an error if it's malformed, for templates that come from outside the program.

```go
t, err := scalar.CompileTemplate("Depth ${depth.km") // a template from outside the program: check it
fmt.Println(t == nil, err)
```

```
true scalar: bad template "Depth ${depth.km" at offset 7: no } to match the {
```

#### `Template.ListSeparator`

ListSeparator returns a copy of t that joins the elements of an array with sep, as local \$" = sep does in Perl.

```go
t := scalar.MustTemplate("Stations: @stations").ListSeparator(", ") // local $" = ", "
fmt.Println(scalar.Hash("stations", scalar.Value([]string{"PAS", "GSC"})).Interpolate(t))
```

```
Stations: PAS, GSC
```

#### `Template.String`

String returns the template as written.

```go
fmt.Println(scalar.MustTemplate("M$mag near $place"))
```

```
M$mag near $place
```

### Reference: Operators

#### `SV.Add`

Add is Perl's +.

```go
fmt.Println(scalar.Value("40").Add(2))          // "40" + 2
fmt.Println(scalar.Value(1.5).Add("1.5"))       // floats when needed
fmt.Println(scalar.Value(math.MaxInt64).Add(1)) // an integer that would overflow becomes a float
```

```
42
3
9.22337203685478e+18
```

#### `SV.Sub`

Sub is Perl's -.

```go
fmt.Println(scalar.Value(10).Sub("2.5"))
```

```
7.5
```

#### `SV.Mul`

Mul is Perl's *.

```go
fmt.Println(scalar.Value("6").Mul(7))
```

```
42
```

#### `SV.Div`

Div is Perl's /.

```go
fmt.Println(scalar.Value(7).Div(2)) // not integer division
fmt.Println(scalar.Value(8).Div(2))
```

```
3.5
4
```

#### `SV.Mod`

Mod is Perl's %: both operands are truncated to integers, and a nonzero result takes the sign of the right operand (-7 % 3 is 2).

```go
fmt.Println(scalar.Value(7).Mod(3))
fmt.Println(scalar.Value(-7).Mod(3)) // takes the sign of the right operand
```

```
1
2
```

#### `SV.Pow`

Pow is Perl's **.

```go
fmt.Println(scalar.Value(2).Pow(10))
fmt.Println(scalar.Value(2).Pow(-1))
```

```
1024
0.5
```

#### `SV.Neg`

Neg is Perl's unary minus, string madness included: -"foo" is "-foo", -"-foo" is "+foo", and -"+foo" is "-foo".

```go
fmt.Println(scalar.Value(5).Neg())
fmt.Println(scalar.Value("foo").Neg()) // string madness, as in Perl
fmt.Println(scalar.Value("-foo").Neg())
```

```
-5
-foo
+foo
```

#### `SV.Concat`

Concat is Perl's .

```go
fmt.Println(scalar.Value("Go").Concat(1.5)) // "Go" . 1.5
```

```
Go1.5
```

#### `SV.Repeat`

Repeat is Perl's x operator on a scalar.

```go
fmt.Println(scalar.Value("ab").Repeat(3)) // "ab" x 3
fmt.Printf("%q\n", scalar.Value("ab").Repeat(0))
```

```
ababab
""
```

#### `SV.Inc`

Inc is Perl's ++: it modifies s in place and returns it.

```go
n := scalar.Value(9)
n.Inc()
id := scalar.Value("az9")
id.Inc() // Perl's magic string increment
fmt.Println(n, id)
```

```
10 ba0
```

#### `SV.Dec`

Dec is Perl's --: it modifies s in place and returns it.

```go
s := scalar.Value("aa")
s.Dec() // never magic: "aa" is 0 as a number
fmt.Println(s)
```

```
-1
```

#### `SV.AddInPlace`

AddInPlace is Perl's +=.

```go
totals := scalar.Hash()
for _, n := range []int{5, 10} {
	totals.Get("sum").AddInPlace(n) // $totals{sum} += $n
}
fmt.Println(totals.Get("sum"))
```

```
15
```

#### `SV.SubInPlace`

SubInPlace is Perl's -=.

```go
s := scalar.Value(10)
s.SubInPlace(3) // $s -= 3
fmt.Println(s)
```

```
7
```

#### `SV.MulInPlace`

MulInPlace is Perl's *=.

```go
s := scalar.Value(4)
s.MulInPlace(2.5) // $s *= 2.5
fmt.Println(s)
```

```
10
```

#### `SV.DivInPlace`

DivInPlace is Perl's /=.

```go
s := scalar.Value(9)
s.DivInPlace(2) // $s /= 2
fmt.Println(s)
```

```
4.5
```

#### `SV.ModInPlace`

ModInPlace is Perl's %=.

```go
s := scalar.Value(17)
s.ModInPlace(5) // $s %= 5
fmt.Println(s)
```

```
2
```

#### `SV.PowInPlace`

PowInPlace is Perl's **=.

```go
s := scalar.Value(3)
s.PowInPlace(2) // $s **= 2
fmt.Println(s)
```

```
9
```

#### `SV.ConcatInPlace`

ConcatInPlace is Perl's .=.

```go
s := scalar.Value("M5.1")
s.ConcatInPlace(" near Ridgecrest") // $s .= " near Ridgecrest"
fmt.Println(s)
```

```
M5.1 near Ridgecrest
```

#### `SV.RepeatInPlace`

RepeatInPlace is Perl's x=.

```go
s := scalar.Value("-")
s.RepeatInPlace(5) // $s x= 5
fmt.Println(s)
```

```
-----
```

#### `SV.Eq`

Eq is Perl's eq.

```go
fmt.Println(scalar.Value(1.0).Eq("1")) // eq compares strings: 1.0 is "1"
fmt.Println(scalar.Value("1.0").Eq(1))
```

```
true
false
```

#### `SV.Ne`

Ne is Perl's ne.

```go
fmt.Println(scalar.Value("go").Ne("Go"))
```

```
true
```

#### `SV.Lt`

Lt is Perl's lt.

```go
fmt.Println(scalar.Value("10").Lt("9")) // as strings, "10" sorts first
```

```
true
```

#### `SV.Le`

Le is Perl's le.

```go
fmt.Println(scalar.Value("a").Le("a"))
```

```
true
```

#### `SV.Gt`

Gt is Perl's gt.

```go
fmt.Println(scalar.Value("b").Gt("abc"))
```

```
true
```

#### `SV.Ge`

Ge is Perl's ge.

```go
fmt.Println(scalar.Value("abc").Ge("abd"))
```

```
false
```

#### `SV.NumEq`

NumEq is Perl's ==.

```go
fmt.Println(scalar.Value("1.0").NumEq(1)) // == compares numbers
fmt.Println(scalar.Value("NaN").NumEq("NaN"))
```

```
true
false
```

#### `SV.NumNe`

NumNe is Perl's !=.

```go
fmt.Println(scalar.Value("10").NumNe(10.0))
```

```
false
```

#### `SV.NumLt`

NumLt is Perl's <.

```go
fmt.Println(scalar.Value("10").NumLt("9")) // as numbers, 9 is smaller
```

```
false
```

#### `SV.NumLe`

NumLe is Perl's <=.

```go
fmt.Println(scalar.Value(2).NumLe("2.0"))
```

```
true
```

#### `SV.NumGt`

NumGt is Perl's >.

```go
fmt.Println(scalar.Value("9007199254740993").NumGt("9007199254740992")) // integers compare exactly
```

```
true
```

#### `SV.NumGe`

NumGe is Perl's >=.

```go
fmt.Println(scalar.Value(-1).NumGe(0))
```

```
false
```

### Reference: Arrays

#### `Array`

Array returns an array of vals, as Perl's (...) list does: arrays and hashes among vals are flattened into it (see Push), and other values are converted as by `SV.Set`.

```go
a := scalar.Array("a", []string{"b", "c"}, scalar.Array("d")) // lists flatten, as in Perl
fmt.Println(a.Length(), a.Join(","))
nested := scalar.Array(1, scalar.Value(scalar.Array(2, 3))) // a reference stays one element
fmt.Println(nested.Length(), nested.Get(1).AV().Join(","))
```

```
4 a,b,c,d
2 2,3
```

#### `AV.String`

String returns a on one line, as Dump does but with Indent = 0, so `fmt.Println`(a) shows its elements, nested arrays and hashes included: [1,"a",[2]].

```go
a := scalar.Array("Go", 1, scalar.Value(scalar.Array(2, 3)))
fmt.Println(a) // fmt uses String: the elements, on one line
```

```
["Go",1,[2,3]]
```

#### `AV.Dump`

Dump returns a as Data::Dumper prints \@a; see `SV.Dump`.

```go
fmt.Println(scalar.Array("Go", nil, true).Dump())
```

```
[
  "Go",
  undef,
  !!1,
]
```

#### `AV.Length`

Length is Perl's scalar(@a).

```go
fmt.Println(scalar.Array(1, 2, 3).Length()) // scalar(@a)
```

```
3
```

#### `AV.Scalar`

Scalar is Perl's scalar(@a), the number of elements.

```go
fmt.Println(scalar.Array("x", "y").Scalar()) // scalar(@a), the same as Length
```

```
2
```

#### `AV.LastIndex`

LastIndex is Perl's \$\#a: Length() - 1, so -1 for an empty array.

```go
fmt.Println(scalar.Array(1, 2, 3).LastIndex()) // $#a
fmt.Println(scalar.Array().LastIndex())
```

```
2
-1
```

#### `AV.SetLastIndex`

SetLastIndex is Perl's \$\#a = n.

```go
a := scalar.Array(1, 2, 3, 4)
a.SetLastIndex(1) // $#a = 1: truncate
fmt.Println(a.Join(","))
a.SetLastIndex(3) // extend with holes
fmt.Println(a.Length(), a.Exists(3))
```

```
1,2
4 false
```

#### `AV.Get`

Get is Perl's \$a[i]: it returns element i; negative indices count from the end, as in Perl.

```go
a := scalar.Array("first", "second", "last")
fmt.Println(a.Get(0), a.Get(-1))             // $a[0], $a[-1]
fmt.Println(a.Get(10).Defined(), a.Length()) // reading doesn't create
a.Get(4).Set("new")                          // writing does, as $a[4] = "new"
fmt.Println(a.Length())

rows, _ := scalar.FromJSON([]byte(`[{"net": "ci", "stations": ["PAS", "GSC"]}]`))
r := rows.AV()
fmt.Println(r.Get(0, "stations", -1)) // $r[0]{stations}[-1]: a route
```

```
first last
false 3
5
GSC
```

#### `AV.Set`

Set is Perl's \$a[i] = v: it assigns v to element i, extending the array if needed, and returns the stored SV.

```go
a := scalar.Array()
a.Set(2, "c") // $a[2] = "c": extends the array, leaving holes
fmt.Println(a.Length(), a.Exists(0), a.Get(2))
```

```
3 false c
```

#### `AV.Exists`

Exists is Perl's exists \$a[i]: whether element i has been set.

```go
a := scalar.Array("a", nil)
fmt.Println(a.Exists(1)) // set, though undef
fmt.Println(a.Exists(5))
fmt.Println(a.Exists(3, "id"), a.Length()) // exists $a[3]{id}: creates $a[3], as in Perl
```

```
true
false
false 4
```

#### `AV.Defined`

Defined is Perl's defined \$a[i]: whether element i exists and isn't undef.

```go
a := scalar.Array("a", nil)
fmt.Println(a.Defined(0), a.Defined(1), a.Defined(-2))
```

```
true false true
```

#### `AV.Delete`

Delete is Perl's delete \$a[i]: it returns the element (undef if it didn't exist) and leaves a hole.

```go
a := scalar.Array("a", "b", "c")
fmt.Println(a.Delete(1)) // delete $a[1]: leaves a hole
fmt.Println(a.Length(), a.Exists(1))
a.Delete(-1) // deleting the last element shrinks the array
fmt.Println(a.Length())
rows, _ := scalar.FromJSON([]byte(`[{"id": "ci40123456", "tmp": 1}]`))
fmt.Println(rows.AV().Delete(0, "tmp"), rows.At("[0]").HV().KeysSlice()) // delete $rows[0]{tmp}
```

```
b
3 false
1
1 [id]
```

#### `AV.Push`

Push is Perl's push: it appends vals and returns a.

```go
a := scalar.Array("a")
a.Push("b", []string{"c", "d"})          // push @a, "b", @list
a.Push(scalar.Value(scalar.Array(1, 2))) // push @a, [1, 2]: a reference, one element
fmt.Println(a.Length(), a.Get(-1).Ref())
```

```
5 ARRAY
```

#### `AV.Unshift`

Unshift is Perl's unshift: it prepends vals, keeping their order, and returns a.

```go
a := scalar.Array("c")
a.Unshift("a", "b") // unshift @a, "a", "b": order kept
fmt.Println(a.Join(","))
```

```
a,b,c
```

#### `AV.Pop`

Pop removes and returns the last element, or undef if a is empty.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Pop(), a.Join(","))
fmt.Println(scalar.Array().Pop().Defined()) // nothing to pop: undef
```

```
3 1,2
false
```

#### `AV.Shift`

Shift removes and returns the first element, or undef if a is empty.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Shift(), a.Join(","))
```

```
1 2,3
```

#### `AV.PopN`

PopN removes the last n elements and returns them, in array order, as splice(@a, -n) does: Array(1, 2, 3, 4).PopN(2) gives 3, 4 and leaves 1, 2.

```go
a := scalar.Array("a", "b", "c", "d")
last := a.PopN(2) // splice(@a, -2): in array order
fmt.Println(last.Join(","), "|", a.Join(","))
fmt.Println(a.PopN(10).Join(","), "|", a.Length()) // more than there is: all of it
```

```
c,d | a,b
a,b | 0
```

#### `AV.ShiftN`

ShiftN removes the first n elements and returns them, in array order, as splice(@a, 0, n) does: Array(1, 2, 3, 4).ShiftN(2) gives 1, 2 and leaves 3, 4.

```go
queue := scalar.Array("PAS", "GSC", "ISA", "SVD", "BBR")
for queue.Length() > 0 {
	batch := queue.ShiftN(2) // splice(@queue, 0, 2): the next batch
	fmt.Println(batch.Join(","))
}
```

```
PAS,GSC
ISA,SVD
BBR
```

#### `AV.Splice`

Splice is Perl's splice @a, offset, length, list: it removes length elements starting at offset, inserts list in their place, and returns the removed elements.

```go
a := scalar.Array("a", "b", "c", "d", "e")
removed := a.Splice(1, 2, "X") // splice @a, 1, 2, "X"
fmt.Println(removed.Join(","), "|", a.Join(","))
a.Splice(-1, 1) // remove the last element
fmt.Println(a.Join(","))
```

```
b,c | a,X,d,e
a,X,d
```

#### `AV.Values`

Values is Perl's values @a: the elements.

```go
a := scalar.Array("a", "b", "c", "d")
fmt.Println(a.Values().Join(","))         // values @a
fmt.Println(a.Values(0, -1, 1).Join(",")) // @a[0, -1, 1]
picks := scalar.Array(2, 3)
fmt.Println(a.Values(picks).Join(","))    // @a[@picks]
for _, s := range a.Values(0, -1).All() { // $_ = uc for @a[0, -1]: the elements themselves
	s.Set(s.Uc())
}
fmt.Println(a.Join(","))
```

```
a,b,c,d
a,d,b
c,d
A,b,c,D
```

#### `AV.Pairs`

Pairs is Perl's index/value slice %a[i, j, ...]: a new array of index/value pairs for those indices, in that order, given as for Values.

```go
a := scalar.Array("a", "b")
fmt.Println(a.Pairs(0, -1).Join(",")) // %a[0, -1]: indices as written
fmt.Println(a.Pairs().Join(","))      // every index and element
```

```
0,a,-1,b
0,a,1,b
```

#### `AV.Items`

Items returns the elements.

```go
for _, s := range scalar.Array("a", "b").Items() { // a Go []*scalar.SV
	fmt.Print(s, " ")
}
fmt.Println()
```

```
a b
```

#### `AV.StrSlice`

StrSlice returns the elements as a Go []string, each in string context (see `SV.Str`), so undef and holes become "".

```go
fmt.Printf("%q\n", scalar.Array("a", 1, nil).StrSlice())
```

```
["a" "1" ""]
```

#### `AV.IntSlice`

IntSlice returns the elements as a Go []int64, each in integer context (see `SV.Int`), so values that aren't numbers become 0.

```go
fmt.Println(scalar.Array("3", 4.9, "x").IntSlice())
```

```
[3 4 0]
```

#### `AV.NumSlice`

NumSlice returns the elements as a Go []float64, each in numeric context (see `SV.Num`).

```go
fmt.Println(scalar.Array("1.5", 2).NumSlice())
```

```
[1.5 2]
```

#### `AV.Each`

Each is Perl's each @a (Perl 5.12 and later): the next index and element, with ok true, or ok false once every element has been returned, after which it starts again from the beginning.

```go
a := scalar.Array("x", "y")
for i, v, ok := a.Each(); ok; i, v, ok = a.Each() { // while (my ($i, $v) = each @a)
	fmt.Println(i, v)
}
```

```
0 x
1 y
```

#### `AV.All`

All iterates over index/element pairs.

```go
for i, s := range scalar.Array("a", "b").All() { // while (my ($i, $s) = each @a)
	fmt.Println(i, s)
}
```

```
0 a
1 b
```

#### `AV.Join`

Join is Perl's join.

```go
fmt.Println(scalar.Array("Go", "Perl", 5).Join(", "))
```

```
Go, Perl, 5
```

#### `AV.Map`

Map is Perl's map: a new array of f applied to each element.

```go
a := scalar.Array("go", "perl")
fmt.Println(a.Map(func(s *scalar.SV) any { return s.Uc() }).Join(","))
pairs := a.Map(func(s *scalar.SV) any { return scalar.Array(s, s.Length()) }) // several values each
fmt.Println(pairs.Join(","))
```

```
GO,PERL
go,2,perl,4
```

#### `AV.Grep`

Grep is Perl's grep: a new array of copies of the elements f accepts.

```go
n := scalar.Array(1, 2, 3, 4).Grep(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })
fmt.Println(n.Join(","))
```

```
2,4
```

#### `AV.Sort`

Sort is Perl's sort: a sorted copy of a.

```go
a := scalar.Array("10", "9", "2")
fmt.Println(a.Sort(nil).Join(","))           // sort @a: as strings
fmt.Println(a.Sort(scalar.NumCmp).Join(",")) // sort { $a <=> $b } @a
byLength := func(x, y *scalar.SV) int { return x.Length() - y.Length() }
fmt.Println(scalar.Array("ccc", "a", "bb").Sort(byLength).Join(","))
fmt.Println(a.Join(",")) // unchanged
```

```
10,2,9
2,9,10
a,bb,ccc
10,9,2
```

#### `AV.SortInPlace`

SortInPlace is Perl's @a = sort @a: it sorts a itself, as Sort would, and returns a.

```go
a := scalar.Array(3, 1, 2)
a.SortInPlace(scalar.NumCmp) // @a = sort { $a <=> $b } @a
fmt.Println(a.Join(","))
```

```
1,2,3
```

#### `AV.Reverse`

Reverse is Perl's reverse: a reversed copy of a.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Reverse().Join(","), a.Join(","))
```

```
3,2,1 1,2,3
```

#### `AV.ReverseInPlace`

ReverseInPlace is Perl's @a = reverse @a: it reverses a itself and returns a.

```go
a := scalar.Array(1, 2, 3)
a.ReverseInPlace()
fmt.Println(a.Join(","))
```

```
3,2,1
```

#### `AV.SplitAt`

SplitAt splits a into its first n elements and the rest, as Perl's my (\$x1, ..., \$xn, @tail) = @a does.

```go
head, tail := scalar.Array("Ada", "Lovelace", "maths", "poetry").SplitAt(2) // my ($first, $last, @interests) = @a
fmt.Println(head.Join(" "), "|", tail.Join(","))
```

```
Ada Lovelace | maths,poetry
```

#### `AV.SplitAtSlice`

SplitAtSlice is SplitAt with the head as a Go slice of exactly n elements, so it can be unpacked: h, tail := `a.SplitAtSlice`(3); x, y, z := h[0], h[1], h[2].

```go
h, rest := scalar.Array("2026", "10", "13", "extra").SplitAtSlice(3)
year, month, day := h[0], h[1], h[2] // my ($year, $month, $day, @rest) = @a
fmt.Println(year, month, day, rest.Length())
```

```
2026 10 13 1
```

#### `AV.Clone`

Clone returns a deep copy of a, as dclone(\@a) does; see `SV.Clone`.

```go
rows := scalar.Array(scalar.Value(map[string]any{"place": "Cobb, CA"})) // one row: a hash reference
mine := rows.Clone()                                                    // dclone(\@rows): the rows too, not just the list
mine.Get(0).At("place").Set("Anza, CA")
fmt.Println(rows.Get(0).At("place"), "/", mine.Get(0).At("place"))
```

```
Cobb, CA / Anza, CA
```

#### `AV.Equal`

Equal reports whether a and other hold the same data; see `SV.Equal`.

```go
a := scalar.Array("PAS", 1)
fmt.Println(a.Equal([]any{"PAS", "1"}), a.Equal([]any{"PAS"}))
```

```
true false
```

#### `AV.Diff`

Diff says where a and other first differ; see `SV.Diff`.

```go
fmt.Println(scalar.Array("PAS", "GSC").Diff([]string{"PAS"}))
```

```
[1]: got "GSC", want does not exist
```

#### `AV.Apply`

Apply calls f on every plain value in a, all the way down, as rmap { ...

```go
prices := scalar.Array(10, scalar.Value([]any{20, 30}))
prices.Apply(func(v *scalar.SV) any { return v.Mul(1.2) }) // every price, nested ones too
fmt.Println(prices.Get(0), prices.Get(1).AV().Join(" "))
```

```
12 24 36
```

#### `AV.At`

At is `SV.At`, starting at a.

```go
rows, _ := scalar.FromJSON([]byte(`[{"name":"Ada"},{"name":"Grace"}]`))
a := rows.AV()
fmt.Println(a.At("[1].name")) // $a[1]{name}
a.At("[2].name").Set("Alan")  // $a[2]{name} = "Alan": creates the row
fmt.Println(a.Length(), a.At("[-1].name"))
```

```
Grace
3 Alan
```

#### `AV.Find`

Find is `SV.Find`, with \$ standing for a.

```go
quakes, _ := scalar.FromJSON([]byte(`[
	{"place": "12 km SSW of Ridgecrest, CA", "mag": 4.6},
	{"place": "5 km N of Parkfield, CA", "mag": 2.1},
	{"place": "8 km E of Anza, CA", "mag": 3.8}
]`))
big := quakes.AV().Find("$[?@.mag >= 3].place") // $ is the array itself
fmt.Println(big.Join("; "))
```

```
12 km SSW of Ridgecrest, CA; 8 km E of Anza, CA
```

#### `AV.AtOK`

AtOK is `SV.AtOK`, starting at a.

```go
a := scalar.Array(1, nil)
_, ok := a.AtOK("[1]") // exists, though undef
_, past := a.AtOK("[2]")
fmt.Println(ok, past)
```

```
true false
```

#### `AV.DeleteAt`

DeleteAt is `SV.DeleteAt`, starting at a.

```go
a := scalar.Array("a", "b", "c")
fmt.Println(a.DeleteAt("[-1]"), a.Length()) // delete $a[-1]: the last goes, so it's shorter
fmt.Println(a.DeleteAt("[0]"), a.Length())  // delete $a[0]: a hole, so it isn't
```

```
c 2
a 2
```

#### `AV.GetOK`

GetOK is `SV.GetOK`, starting at a.

```go
a := scalar.Array(scalar.Value(scalar.Hash("id", 7)))
v, ok := a.GetOK(0, "id")
_, missing := a.GetOK(0, "name")
fmt.Println(v, ok, missing)
```

```
7 true false
```

#### `AV.DeletePath`

DeletePath is `SV.DeletePath`, starting at a.

```go
rows, _ := scalar.FromJSON([]byte(`[{"id":1,"tmp":"x"},{"id":2,"tmp":"y"}]`))
a := rows.AV()
for i := range a.Length() {
	a.DeletePath(i, "tmp") // delete $a[$i]{tmp}
}
fmt.Println(a.At("[0]").HV().KeysSlice(), a.At("[1]").HV().KeysSlice())
```

```
[id] [id]
```

#### `AV.Undef`

Undef is Perl's undef @a: it empties a in place, so everything referring to a sees it empty, and returns a, now empty.

```go
a := scalar.Array("x", "y")
doc := scalar.Hash("tags", scalar.Value(a)) // another reference to a
a.Undef()                                   // undef @a
fmt.Println(a.Length(), doc.At("tags").AV().Length())
```

```
0 0
```

#### `AV.Lock`

Lock locks a and everything in it, as `SV.Lock` does, and returns a.

A locked array can't grow, shrink or change, nor can its elements.

```go
a := scalar.Array(1, 2, 3).Lock()
fmt.Println(a.Get(0), a.Length())
fmt.Println(catch(func() { a.Push(4) }))
fmt.Println(catch(func() { a.Get(0).Set(9) }))
```

```
1 3
scalar: Modification of a read-only value attempted
scalar: Modification of a read-only value attempted
```

#### `AV.Unlock`

Unlock undoes Lock, as `SV.Unlock` does, and returns a.

```go
a := scalar.Array(1, 2).Lock().Unlock()
fmt.Println(a.Push(3).Length())
```

```
3
```

#### `AV.Locked`

Locked reports whether a is locked.

```go
a := scalar.Array(1, 2)
fmt.Println(a.Locked(), a.Lock().Locked())
```

```
false true
```

### Reference: Hashes

#### `Hash`

Hash builds a hash from a key/value list, as %h = (k1 => v1, ...) does.

```go
h := scalar.Hash("name", "Ada", "age", 36) // %h = (name => "Ada", age => 36)
fmt.Println(h.Get("name"), h.Get("age"))
merged := scalar.Hash(h, "age", 37) // %merged = (%h, age => 37): later keys win
fmt.Println(merged.Get("age"))
fmt.Println(scalar.Hash(map[string]int{"a": 1}).Get("a")) // from a Go map
```

```
Ada 36
37
1
```

#### `HV.String`

String returns h on one line, as Dump does but with Indent = 0, so `fmt.Println`(h) shows its keys and values: {a => 1,b => [2]}.

```go
h := scalar.Hash("place", "Cobb, CA", "log level", "debug")
fmt.Println(h) // keys sorted, and quoted only when they need to be
```

```
{"log level" => "debug",place => "Cobb, CA"}
```

#### `HV.Dump`

Dump returns h as Data::Dumper prints \%h; see `SV.Dump`.

```go
tags := scalar.Value(scalar.Array("go"))
h := scalar.Hash("tags", tags, "again", tags) // the same array twice
fmt.Println(h.Dump())
```

```
{
  again => [
    "go",
  ],
  tags => $VAR1->{again},
}
```

#### `HV.Length`

Length is Perl's scalar(keys %h).

```go
fmt.Println(scalar.Hash("a", 1, "b", 2).Length()) // scalar(keys %h)
```

```
2
```

#### `HV.Scalar`

Scalar is Perl's scalar(%h), which since Perl 5.26 is the number of keys.

```go
fmt.Println(scalar.Hash("a", 1).Scalar()) // scalar(%h), the same as Length
```

```
1
```

#### `HV.Get`

Get is Perl's \$h{key}: it returns the value for key.

```go
h := scalar.Hash("a", 1)
fmt.Println(h.Get("a"))
fmt.Println(h.Get("b").Defined(), h.Exists("b")) // reading doesn't create
h.Get("c").HV().Set("d", 1)                      // $h{c}{d} = 1 autovivifies
fmt.Println(h.Get("c").Ref())

quake, _ := scalar.FromJSON([]byte(`{"depth": {"km": 8.2}, "mag": 4.6, "net": "ci"}`))
q := quake.HV()
fmt.Println(q.Get("depth", "km"))             // $q{depth}{km}: a route
fmt.Println(q.Values("mag", "net").Join(" ")) // @q{qw(mag net)}: a slice is Values
```

```
1
false false
HASH
8.2
4.6 ci
```

#### `HV.Set`

Set is Perl's \$h{key} = v: it assigns v to key and returns the stored SV.

```go
h := scalar.Hash()
h.Set("tags", []string{"go", "perl"}) // $h{tags} = ["go", "perl"]
fmt.Println(h.Get("tags").AV().Join(","))
```

```
go,perl
```

#### `HV.Push`

Push adds key/value pairs to h and returns h.

```go
h := scalar.Hash("id", "ci40123456")
h.Push("place", "Ridgecrest, CA", "felt", 1204) // $h{place} = "Ridgecrest, CA"; $h{felt} = 1204
defaults := scalar.Hash("tsunami", false, "felt", 0)
h.Push(defaults, "felt", 1500) // merge, later keys winning
fmt.Println(h.Keys().Sort(nil).Join(","))
fmt.Println(h.Get("felt"))
```

```
felt,id,place,tsunami
1500
```

#### `HV.Exists`

Exists is Perl's exists \$h{key}: whether key is there, even if its value is undef.

```go
h := scalar.Hash("felt", nil)
fmt.Println(h.Exists("felt"), h.Exists("cdi")) // there, though undef
fmt.Println(h.Exists("origin", "depth"))       // exists $h{origin}{depth}
fmt.Println(h.Exists("origin"))                // which created $h{origin}, as in Perl
```

```
true false
false
true
```

#### `HV.Defined`

Defined is Perl's defined \$h{key}: whether key exists and its value isn't undef.

```go
h := scalar.Hash("felt", nil, "place", "Cobb, CA")
fmt.Println(h.Defined("felt"), h.Defined("place"))
```

```
false true
```

#### `HV.Delete`

Delete is Perl's delete \$h{key}: it removes key and returns its value, or undef.

```go
h := scalar.Hash("a", 1, "b", 2)
fmt.Println(h.Delete("a"), h.Exists("a")) // returns the value it removed
fmt.Println(h.Delete("missing").Defined())
quake, _ := scalar.FromJSON([]byte(`{"depth": {"km": 8.2, "error": 0.4}}`))
q := quake.HV()
fmt.Println(q.Delete("depth", "error"), q.Get("depth").HV().KeysSlice()) // delete $q{depth}{error}
```

```
1 false
false
0.4 [km]
```

#### `HV.Clone`

Clone returns a deep copy of h, as dclone(\%h) does; see `SV.Clone`.

```go
defaults := scalar.Hash("tsunami", false, "stations", scalar.Value([]string{"PAS"}))
row := defaults.Clone() // dclone(\%defaults): nested arrays too
row.Get("stations").AV().Push("GSC")
merged := scalar.Hash(defaults) // %row = %defaults: a new hash, but the same stations array
merged.Get("stations").AV().Push("ISA")
fmt.Println(defaults.Get("stations").AV().Join(","), "/", row.Get("stations").AV().Join(","))
```

```
PAS,ISA / PAS,GSC
```

#### `HV.Equal`

Equal reports whether h and other hold the same data; see `SV.Equal`.

```go
h := scalar.Hash("tsunami", true, "felt", 2)
fmt.Println(h.Equal(map[string]any{"tsunami": 1, "felt": "2"})) // true is 1, as in Perl
```

```
true
```

#### `HV.Diff`

Diff says where h and other first differ; see `SV.Diff`.

```go
h := scalar.Hash("felt", nil)
fmt.Println(h.Diff(map[string]any{})) // a key whose value is undef isn't a missing key
```

```
felt: got undef, want does not exist
```

#### `HV.Apply`

Apply calls f on every plain value in h, all the way down, as rmap { ...

```go
row := scalar.Hash("place", "ridgecrest, CA", "tsunami", true)
row.Apply(func(v *scalar.SV) any {
	if v.Kind() == scalar.String {
		return v.UcFirst()
	}
	return v
})
fmt.Println(row.Get("place"), row.Get("tsunami"))
```

```
Ridgecrest, CA 1
```

#### `HV.Pairs`

Pairs is Perl's %h in list context: a new array of key/value pairs, k1, v1, k2, v2 and so on, in the same order as Keys.

```go
h2 := scalar.Hash("k1", 1, "k2", 2, "k3", 3)
h1 := scalar.Hash(h2.Pairs("k1", "k3")) // %h1 = %h2{qw(k1 k3)}
fmt.Println(h1.Keys().Sort(nil).Join(","), h1.Get("k3"))
fmt.Println(h2.Pairs("k2", "missing").Join(","))      // a missing key comes with undef
fmt.Println(scalar.Hash("only", 1).Pairs().Join(",")) // every pair: %h in list context
```

```
k1,k3 3
k2,2,missing,
only,1
```

#### `HV.PairsMap`

PairsMap returns h as a Go map from its keys to its own values, as Pairs() would list them, for code that doesn't use this package.

```go
h := scalar.Hash("a", 1)
m := h.PairsMap() // a Go map[string]*scalar.SV of the hash's own values
m["a"].Set(2)
fmt.Println(len(m), h.Get("a"))
```

```
1 2
```

#### `HV.Keys`

Keys is Perl's keys %h: a new array of the keys.

```go
h := scalar.Hash("b", 2, "a", 1)
fmt.Println(h.Keys().Sort(nil).Join(",")) // sort keys %h
```

```
a,b
```

#### `HV.Values`

Values is Perl's values %h: the values, in the same order as Keys.

```go
h := scalar.Hash("a", 1, "b", 2, "c", 3)
fmt.Println(h.Values("c", "a").Join(","))                     // @h{qw(c a)}
fmt.Println(h.Values("missing").Get(0).Defined(), h.Length()) // undef, and not created
for _, v := range h.Values().All() {                          // $_ *= 10 for values %h
	v.MulInPlace(10)
}
fmt.Println(h.Values("a", "b", "c").Join(","))
```

```
3,1
false 3
10,20,30
```

#### `HV.KeysSlice`

KeysSlice is Keys as a Go []string.

```go
keys := scalar.Hash("a", 1).KeysSlice() // a Go []string
fmt.Printf("%q\n", keys)
```

```
["a"]
```

#### `HV.ValuesSlice`

ValuesSlice is Values() as a Go []*SV.

```go
values := scalar.Hash("a", "x").ValuesSlice() // a Go []*scalar.SV
fmt.Println(len(values), values[0])
```

```
1 x
```

#### `HV.Each`

Each is Perl's each %h: the next key/value pair, with ok true, or ok false once every pair has been returned, after which it starts again from the beginning.

```go
h := scalar.Hash("a", 1, "b", 2, "c", 3)
sum := 0
for _, v, ok := h.Each(); ok; _, v, ok = h.Each() { // while (my ($k, $v) = each %h)
	sum += int(v.Int())
}
fmt.Println(sum)

// As in Perl, the hash remembers where Each got to, until Keys and the
// like reset it.
first, _, _ := h.Each()
h.Keys() // keys %h resets the iterator
again, _, _ := h.Each()
fmt.Println(first == again)
```

```
6
true
```

#### `HV.All`

All iterates over key/value pairs in the same order as Keys.

```go
h := scalar.Hash("a", 1, "b", 2)
sum := 0
for _, v := range h.All() { // while (my ($k, $v) = each %h)
	sum += int(v.Int())
}
fmt.Println(sum)
```

```
3
```

#### `HV.At`

At is `SV.At`, starting at h.

```go
tallies := scalar.Hash()
tallies.At("ci.reviewed").Inc() // $tallies{ci}{reviewed}++
tallies.At("ci.reviewed").Inc()
tallies.At("ci.automatic").Inc()
fmt.Println(tallies.At("ci.reviewed"), tallies.At("ci.automatic"), tallies.At("us.reviewed").Int())
```

```
2 1 0
```

#### `HV.Find`

Find is `SV.Find`, with \$ standing for h.

```go
doc, _ := scalar.FromJSON([]byte(`{"ci": ["PAS", "GSC"], "nc": ["MCB"]}`))
networks := doc.HV()
fmt.Println(networks.Find("$.*[*]").Length()) // every station, in any network
fmt.Println(networks.Find("$.ci[-1]").Get(0)) // $ is the hash itself
```

```
3
GSC
```

#### `HV.AtOK`

AtOK is `SV.AtOK`, starting at h.

```go
h := scalar.Hash("depth", scalar.Value(scalar.Hash("error", nil)))
_, ok := h.AtOK("depth.error") // exists, though undef
_, missing := h.AtOK("depth.km")
fmt.Println(ok, missing)
```

```
true false
```

#### `HV.DeleteAt`

DeleteAt is `SV.DeleteAt`, starting at h.

```go
h := scalar.Hash("depth", scalar.Value(scalar.Hash("km", 8, "error", 2)))
fmt.Println(h.DeleteAt("depth.error")) // delete $h{depth}{error}
fmt.Println(h.At("depth").HV().KeysSlice())
```

```
2
[km]
```

#### `HV.GetOK`

GetOK is `SV.GetOK`, starting at h.

```go
h := scalar.Hash("tags", scalar.Value(scalar.Array("go")))
v, ok := h.GetOK("tags", 0)
_, past := h.GetOK("tags", 5)
fmt.Println(v, ok, past)
```

```
go true false
```

#### `HV.DeletePath`

DeletePath is `SV.DeletePath`, starting at h.

```go
section, key := "depth", "error" // from data
h := scalar.Hash("depth", scalar.Value(scalar.Hash("km", 8, "error", 2)))
fmt.Println(h.DeletePath(section, key)) // delete $h{$section}{$key}
```

```
2
```

#### `HV.Undef`

Undef is Perl's undef %h: it empties h in place, so everything referring to h sees it empty, and returns h, now empty.

```go
seen := scalar.Hash("go", 1, "perl", 1)
seen.Undef() // undef %seen: emptied, ready for the next batch
seen.Set("raku", 1)
fmt.Println(seen.KeysSlice())
```

```
[raku]
```

#### `HV.Map`

Map builds a new hash from h, as %new = map { f(\$_, \$h{\$_}) } keys %h does: f gets each key and value, and returns the new key and value.

Map builds a new hash, renaming keys or changing values or both. The old hash is unchanged.

```go
depth := scalar.Hash("min", 2000, "max", 12500) // metres
km := depth.Map(func(key string, v *scalar.SV) (string, any) {
	return "depth_" + key, v.Div(1000).Str() + " km"
})
fmt.Println(km.Get("depth_min"), "-", km.Get("depth_max"), depth.Get("min"))
```

```
2 km - 12.5 km 2000
```

#### `HV.Lock`

Lock locks h and everything in it, as `SV.Lock` does, and returns h.

A locked hash keeps the keys it has: reading, writing or deleting one it hasn't got panics, as in a Perl restricted hash; Exists still answers.

```go
h := scalar.Hash("min", 25000).Lock()
fmt.Println(h.Get("min"), h.Exists("max"))
fmt.Println(catch(func() { h.Get("max") }))
fmt.Println(catch(func() { h.Delete("min") }))
```

```
25000 false
scalar: Attempt to access disallowed key 'max' in a restricted hash
scalar: Attempt to delete readonly key 'min' from a restricted hash
```

#### `HV.Unlock`

Unlock undoes Lock, as `SV.Unlock` does, and returns h.

```go
h := scalar.Hash("min", 25000).Lock().Unlock()
h.Set("max", 30000)
fmt.Println(h.Get("max"))
```

```
30000
```

#### `HV.Locked`

Locked reports whether h is locked.

```go
h := scalar.Hash("min", 25000)
fmt.Println(h.Locked(), h.Lock().Locked())
```

```
false true
```

### Reference: List utilities

#### `AV.AnyMatch`

AnyMatch is List::Util's any: whether f is true for at least one element.

```go
places := scalar.Array("5 km W of Cobb, CA", "Offshore Northern California")
fmt.Println(places.AnyMatch(isOffshore)) // any { /offshore/i } @places
fmt.Println(scalar.Array().AnyMatch(isOffshore))
```

```
true
false
```

#### `AV.AllMatch`

AllMatch is List::Util's all: whether f is true for every element.

```go
n := scalar.Array(2, 4, 6)
fmt.Println(n.AllMatch(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })) // all { $_ % 2 == 0 }
fmt.Println(scalar.Array().AllMatch(isOffshore))                                // true for nothing
```

```
true
true
```

#### `AV.NoneMatch`

NoneMatch is List::Util's none: whether f is false for every element.

```go
fmt.Println(scalar.Array("Ridgecrest", "Cobb").NoneMatch(isOffshore))
```

```
true
```

#### `AV.FirstMatch`

FirstMatch is List::Util's first: the first element for which f is true, or undef if there's none.

```go
lines := scalar.Array("Origin: 35.7N 117.6W", "Magnitude: 5.1 ml", "Magnitude: 5.3 mw")
mag := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("Magnitude") >= 0 })
fmt.Println(mag) // first { index($_, "Magnitude") >= 0 } @lines
fmt.Println(lines.FirstMatch(isOffshore).Defined())
```

```
Magnitude: 5.1 ml
false
```

#### `AV.MaxNum`

MaxNum is List::Util's max: the element with the largest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MaxNum()) // max: as numbers
```

```
10
```

#### `AV.MinNum`

MinNum is List::Util's min: the element with the smallest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MinNum())
```

```
2.5
```

#### `AV.MaxStr`

MaxStr is List::Util's maxstr: the element that sorts last as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", "9", "100").MaxStr()) // maxstr: as strings
```

```
9
```

#### `AV.MinStr`

MinStr is List::Util's minstr: the element that sorts first as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("b", "a", "c").MinStr())
```

```
a
```

#### `AV.UniqueStr`

UniqueStr is List::Util's uniqstr: a new array of copies of the elements, keeping only the first of each distinct string, in order.

```go
fmt.Println(scalar.Array("PAS", "pas", "PAS", "GSC").UniqueStr().Join(",")) // first of each, in order
stations := scalar.Array("PAS", "pas", "GSC")
fmt.Println(stations.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr().Join(",")) // ignoring case
```

```
PAS,pas,GSC
pas,gsc
```

#### `AV.UniqueNum`

UniqueNum is List::Util's uniqnum: a new array of copies of the elements, keeping only the first of each distinct number, in order.

```go
fmt.Println(scalar.Array(1, "1.0", "01", 2).UniqueNum().Join(",")) // the same number, however written
```

```
1,2
```

#### `AV.Sum`

Sum is List::Util's sum: the numeric sum of the elements, or undef for an empty array.

```go
fmt.Println(scalar.Array(1, "2", 3.5).Sum()) // sum @a
fmt.Println(scalar.Array().Sum().Defined())  // undef for nothing
```

```
6.5
false
```

#### `AV.Sum0`

Sum0 is List::Util's sum0: Sum, but 0 for an empty array.

```go
fmt.Println(scalar.Array().Sum0()) // sum0: 0 for nothing
```

```
0
```

#### `AV.Product`

Product is List::Util's product: the numeric product of the elements, or 1 for an empty array, as in List::Util.

```go
fmt.Println(scalar.Array(2, 3, 4).Product())
fmt.Println(scalar.Array().Product()) // 1 for nothing
```

```
24
1
```

#### `AV.Reduce`

Reduce is List::Util's reduce { ...

```go
words := scalar.Array("go", "perl", "c")
longest := words.Reduce(func(a, b *scalar.SV) any { // reduce { length $b > length $a ? $b : $a } @words
	if b.Length() > a.Length() {
		return b
	}
	return a
})
joined := words.Reduce(func(a, b *scalar.SV) any { return a.Str() + "+" + b.Str() })
fmt.Println(longest, joined)
```

```
perl go+perl+c
```

#### `AV.SortByStr`

SortByStr is List::UtilsBy's sort_by: a sorted copy of a, ordered by the string keys f gives, as sort { \$a cmp \$b } would order them.

```go
places := scalar.Array("ridgecrest", "Cobb", "Anza")
byName := places.SortByStr(func(p *scalar.SV) any { return p.Fc() }) // sort_by { fc } @places
fmt.Println(byName.Join(", "))
```

```
Anza, Cobb, ridgecrest
```

#### `AV.SortByNum`

SortByNum is List::UtilsBy's nsort_by: as SortByStr, but comparing the keys as numbers.

```go
quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "mag": 5.1}, {"place": "Cobb", "mag": 2.3}]`))
byMag := quakes.AV().SortByNum(func(q *scalar.SV) any { return q.At("mag") }) // nsort_by { $_->{mag} }
fmt.Println(byMag.Get(0).At("place"), byMag.Get(1).At("place"))
```

```
Cobb Ridgecrest
```

#### `AV.RevSortByStr`

RevSortByStr is List::UtilsBy's rev_sort_by: SortByStr, in descending order.

```go
words := scalar.Array("bb", "a", "dd", "e")
fmt.Println(words.RevSortByStr(func(w *scalar.SV) any { return w.Length() }).Join(" ")) // equal keys keep their order
```

```
bb dd a e
```

#### `AV.RevSortByNum`

RevSortByNum is List::UtilsBy's rev_nsort_by: SortByNum, in descending order, with elements of equal keys keeping their order.

```go
quakes, _ := scalar.FromJSON([]byte(`[{"place": "Cobb", "mag": 2.3}, {"place": "Ridgecrest", "mag": 5.1}, {"place": "Anza", "mag": 5.1}]`))
strongest := quakes.AV().RevSortByNum(func(q *scalar.SV) any { return q.At("mag") }) // the strongest first
fmt.Println(strongest.Get(0).At("place"), strongest.Get(1).At("place"), strongest.Get(2).At("place"))
```

```
Ridgecrest Anza Cobb
```

#### `AV.MaxBy`

MaxBy is List::UtilsBy's max_by: the element with the largest numeric key, the first of them if several share it, or undef for an empty array.

```go
quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "mag": 5.1}, {"place": "Anza", "mag": 5.1}, {"place": "Cobb", "mag": 2.3}]`))
strongest := quakes.AV().MaxBy(func(q *scalar.SV) any { return q.At("mag") }) // max_by { $_->{mag} }
fmt.Println(strongest.At("place"))                                            // the first of the strongest
```

```
Ridgecrest
```

#### `AV.MinBy`

MinBy is List::UtilsBy's min_by: the element with the smallest numeric key; see MaxBy.

```go
words := scalar.Array("ccc", "a", "bb", "e")
fmt.Println(words.MinBy(func(w *scalar.SV) any { return w.Length() })) // min_by { length } @words
```

```
a
```

#### `AV.UniqueBy`

UniqueBy is List::UtilsBy's uniq_by: a new array of copies of the elements, keeping only the first with each key, in order.

```go
quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "net": "ci"}, {"place": "Cobb", "net": "nc"}, {"place": "Anza", "net": "ci"}]`))
oneEach := quakes.AV().UniqueBy(func(q *scalar.SV) any { return q.At("net") }) // the first quake from each network
fmt.Println(oneEach.Length(), oneEach.Get(0).At("place"), oneEach.Get(1).At("place"))
```

```
2 Ridgecrest Cobb
```

#### `AV.PartitionBy`

PartitionBy is List::UtilsBy's partition_by: a hash from each key f gives to an array of copies of the elements with that key, in order.

```go
words := scalar.Array("bb", "a", "ccc", "dd", "e")
byLength := words.PartitionBy(func(w *scalar.SV) any { return w.Length() }) // partition_by { length } @words
fmt.Println(byLength)
```

```
{1 => ["a","e"],2 => ["bb","dd"],3 => ["ccc"]}
```

#### `AV.CountBy`

CountBy is List::UtilsBy's count_by: how many elements have each key f gives, as a Bag, so MostCommon lists the commonest first.

```go
places := scalar.Array("Ridgecrest", "Cobb", "ridgecrest", "RIDGECREST")
where := places.CountBy(func(p *scalar.SV) any { return p.Fc() }) // a Bag: count_by's counts, with MostCommon
fmt.Println(where.MostCommon(1).Get(0), where.Count("ridgecrest"), where.Counts())
```

```
ridgecrest 3 {cobb => 1,ridgecrest => 3}
```

#### `AV.ExtractBy`

ExtractBy is List::UtilsBy's extract_by: it removes the elements for which f returns true from a, and returns them, in order, as a new array.

```go
words := scalar.Array("bb", "a", "ccc", "e")
long := words.ExtractBy(func(w *scalar.SV) bool { return w.Length() > 1 }) // moved out of words
fmt.Println(long.Join(" "), "|", words.Join(" "))
```

```
bb ccc | a e
```

### Reference: Sets and bags

#### `NewSet`

NewSet is Set::Scalar->new(...): a set of vals, flattened as a list is (see `AV.Push`), so NewSet(array) and NewSet(goSlice) take their elements.

```go
s := scalar.NewSet("Go", "Perl", "Go") // Set::Scalar->new(...): each member once
fmt.Println(s, s.Length())
same := scalar.NewSet(1, "1", 1.0) // compared as strings: all "1"
fmt.Println(same.Length())
fromList := scalar.NewSet(scalar.Array("a", "b"), []string{"c"}) // lists flatten
fmt.Println(fromList)
```

```
(Go Perl) 2
1
(a b c)
```

#### `Set.Insert`

Insert is Set::Scalar's insert: it adds vals, flattened as by NewSet, and returns s.

```go
var s scalar.Set // the zero value is an empty set
s.Insert("Go").Insert("Perl", "Go")
fmt.Println(&s)
```

```
(Go Perl)
```

#### `Set.Delete`

Delete is Set::Scalar's delete: it removes vals, flattened as by NewSet, and returns s.

```go
s := scalar.NewSet("a", "b", "c")
s.Delete("b", "not a member") // non-members are ignored
fmt.Println(s)
```

```
(a c)
```

#### `Set.Has`

Has is Set::Scalar's has: whether v (compared as a string) is a member.

```go
s := scalar.NewSet("Go", 42)
fmt.Println(s.Has("Go"), s.Has("go"), s.Has("42")) // "42" is the same member as 42
```

```
true false true
```

#### `Set.Length`

Length is Set::Scalar's size: the number of members.

```go
fmt.Println(scalar.NewSet("a", "b", "a").Length())
```

```
2
```

#### `Set.Members`

Members is Set::Scalar's members: a new array of copies of the members, in the order they were first inserted.

```go
s := scalar.NewSet("c", "a", "b")
fmt.Println(s.Members().Join(","))           // in insertion order
fmt.Println(s.Members().Sort(nil).Join(",")) // sorted, with the array tools
```

```
c,a,b
a,b,c
```

#### `Set.All`

All iterates over copies of the members, in the order they were first inserted.

```go
for m := range scalar.NewSet("x", "y").All() {
	fmt.Println(m)
}
```

```
x
y
```

#### `Set.Copy`

Copy returns a new set with the same members.

```go
a := scalar.NewSet("a")
b := a.Copy()
b.Insert("b")
fmt.Println(a, b)
```

```
(a) (a b)
```

#### `Set.Union`

Union is Set::Scalar's union: a new set of every member of s or any of others, in order of first appearance.

```go
a, b := scalar.NewSet("a", "b"), scalar.NewSet("b", "c")
fmt.Println(a.Union(b))                     // $a->union($b)
fmt.Println(a.Union(b, scalar.NewSet("d"))) // several at once
```

```
(a b c)
(a b c d)
```

#### `Set.Intersection`

Intersection is Set::Scalar's intersection: a new set of the members of s that are also in every one of others.

```go
recorded := scalar.NewSet("PAS", "GSC", "ISA")
expected := scalar.NewSet("PAS", "ISA", "SVD")
fmt.Println(recorded.Intersection(expected)) // the expected stations that recorded the quake
```

```
(ISA PAS)
```

#### `Set.Difference`

Difference is Set::Scalar's difference: a new set of the members of s that are in none of others.

```go
expected := scalar.NewSet("PAS", "ISA", "SVD")
recorded := scalar.NewSet("PAS", "GSC", "ISA")
fmt.Println(expected.Difference(recorded)) // expected, but silent
```

```
(SVD)
```

#### `Set.SymmetricDifference`

SymmetricDifference is Set::Scalar's symmetric_difference: a new set of the members in exactly one of s and t.

```go
a, b := scalar.NewSet("a", "b", "c"), scalar.NewSet("b", "c", "d")
fmt.Println(a.SymmetricDifference(b)) // in one or the other, not both
```

```
(a d)
```

#### `Set.IsSubset`

IsSubset is Set::Scalar's is_subset: whether every member of s is in t.

```go
a := scalar.NewSet("Go")
fmt.Println(a.IsSubset(scalar.NewSet("Go", "Perl")), a.IsSubset(a), scalar.NewSet().IsSubset(a))
```

```
true true true
```

#### `Set.IsProperSubset`

IsProperSubset is Set::Scalar's is_proper_subset: a subset of t that isn't equal to it.

```go
a := scalar.NewSet("Go")
fmt.Println(a.IsProperSubset(scalar.NewSet("Go", "Perl")), a.IsProperSubset(a))
```

```
true false
```

#### `Set.IsSuperset`

IsSuperset is Set::Scalar's is_superset: whether s has every member of t.

```go
fmt.Println(scalar.NewSet("Go", "Perl").IsSuperset(scalar.NewSet("Perl")))
```

```
true
```

#### `Set.IsEqual`

IsEqual is Set::Scalar's is_equal: whether s and t have the same members, in any order.

```go
fmt.Println(scalar.NewSet("a", "b").IsEqual(scalar.NewSet("b", "a"))) // order doesn't matter
```

```
true
```

#### `Set.IsDisjoint`

IsDisjoint is Set::Scalar's is_disjoint: whether s and t have no members in common.

```go
fmt.Println(scalar.NewSet("a").IsDisjoint(scalar.NewSet("b")))
fmt.Println(scalar.NewSet().IsDisjoint(scalar.NewSet())) // true, unlike Set::Scalar
```

```
true
true
```

#### `Set.String`

String is the set as Set::Scalar prints it: its members sorted, between brackets, as in "(a b c)".

```go
fmt.Println(scalar.NewSet("c", "a", "b").String()) // sorted, as Set::Scalar prints a set
```

```
(a b c)
```

#### `Set.MarshalJSON`

MarshalJSON encodes the set as an array of its members, in the order they were first inserted.

```go
out, _ := json.Marshal(scalar.NewSet("c", "a", "b")) // an array, in insertion order
fmt.Println(string(out))
```

```
["c","a","b"]
```

#### `NewBag`

NewBag returns a bag of vals, flattened as a list is (see `AV.Push`), each counting once, so NewBag("Go", "Go", "Perl") has two Go and one Perl.

```go
networks := scalar.NewBag("ci", "us", "ci") // each value counts once
fmt.Println(networks)
apples := scalar.NewBag().InsertN("apples", 3) // Set::Bag->new(apples => 3)
fmt.Println(apples)
```

```
(ci => 2, us => 1)
(apples => 3)
```

#### `Bag.Insert`

Insert adds one of each of vals, flattened as by NewBag, and returns b.

```go
var counts scalar.Bag // the zero value is an empty bag
for _, day := range [][]string{{"ci", "us"}, {"ci", "nc"}, {"ci"}} {
	counts.Insert(day) // count the networks across days
}
fmt.Println(&counts)
```

```
(ci => 3, nc => 1, us => 1)
```

#### `Bag.InsertN`

InsertN is Set::Bag's insert(v => n): it adds n of v and returns b.

```go
b := scalar.NewBag()
b.InsertN("apples", 3).InsertN("apples", 2) // $b->insert(apples => 3) and so on
fmt.Println(b)
```

```
(apples => 5)
```

#### `Bag.Delete`

Delete removes one of each of vals, flattened as by NewBag, and returns b.

```go
b := scalar.NewBag("a", "a", "b")
b.Delete("a", "b") // one of each
fmt.Println(b)
```

```
(a => 1)
```

#### `Bag.DeleteN`

DeleteN is Set::Bag's delete(v => n): it removes n of v, or all of it if there are fewer, and returns b.

```go
b := scalar.NewBag().InsertN("apples", 3)
b.DeleteN("apples", 2)
fmt.Println(b)
b.DeleteN("apples", 10) // more than there is: all of it, never below zero
fmt.Println(b)
```

```
(apples => 1)
()
```

#### `Bag.Count`

Count is Set::Bag's grab(v): how many of v there are, 0 if none.

```go
b := scalar.NewBag("Go", "Go", "Perl")
fmt.Println(b.Count("Go"), b.Count("Rust")) // $b->grab("Go")
```

```
2 0
```

#### `Bag.Length`

Length is the number of distinct members.

```go
fmt.Println(scalar.NewBag("a", "a", "b").Length()) // distinct members
```

```
2
```

#### `Bag.Total`

Total is the number of members counting repeats: the sum of the counts.

```go
fmt.Println(scalar.NewBag("a", "a", "b").Total()) // counting repeats
```

```
3
```

#### `Bag.Members`

Members is Set::Bag's elements: a new array of copies of the distinct members, in the order they were first inserted.

```go
fmt.Println(scalar.NewBag("b", "a", "b").Members().Join(",")) // distinct, in insertion order
```

```
b,a
```

#### `Bag.Counts`

Counts returns the bag as a new hash from each member (as a string) to its count, for the rest of the package's tools.

```go
counts := scalar.NewBag("Go", "Go", "Perl").Counts() // a hash, for the hash tools
fmt.Println(counts.Get("Go"), counts.Get("Perl"))
```

```
2 1
```

#### `Bag.MostCommon`

MostCommon returns copies of the members with the highest counts, most common first, and the first inserted first among equal counts: the top n, or every member if n is left out or there are fewer.

```go
networks := scalar.NewBag("us", "ci", "nc", "ci", "us", "ci")
fmt.Println(networks.MostCommon(2).Join(",")) // the top 2
fmt.Println(networks.MostCommon().Join(","))  // all, most common first
```

```
ci,us
ci,us,nc
```

#### `Bag.All`

All iterates over copies of the distinct members with their counts, in the order they were first inserted.

```go
for m, n := range scalar.NewBag("x", "y", "x").All() {
	fmt.Println(m, n)
}
```

```
x 2
y 1
```

#### `Bag.Copy`

Copy returns a new bag with the same members and counts.

```go
a := scalar.NewBag("a")
b := a.Copy()
b.Insert("a")
fmt.Println(a, b)
```

```
(a => 1) (a => 2)
```

#### `Bag.Sum`

Sum is Set::Bag's sum: a new bag in which each member's count is its counts in b and others added together.

```go
monday := scalar.NewBag().InsertN("ci", 3).InsertN("us", 1)
tuesday := scalar.NewBag().InsertN("ci", 1).InsertN("nc", 2)
fmt.Println(monday.Sum(tuesday)) // counts added: $monday->sum($tuesday)
```

```
(ci => 4, nc => 2, us => 1)
```

#### `Bag.Union`

Union is Set::Bag's union: a new bag in which each member's count is the largest of its counts in b and others.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
fmt.Println(x.Union(y)) // the larger count of each
```

```
(a => 3, b => 2)
```

#### `Bag.Intersection`

Intersection is Set::Bag's intersection: a new bag of the members in b and every one of others, each with the smallest of its counts.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("c", 2)
fmt.Println(x.Intersection(y)) // the smaller count of each, in both
```

```
(a => 1)
```

#### `Bag.Difference`

Difference is Set::Bag's difference: a new bag in which each member's count is its count in b less its counts in others, leaving out any that reach zero, as Set::Bag does with over_delete.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
fmt.Println(x.Difference(y)) // counts taken away, never below zero
```

```
(a => 2)
```

#### `Bag.String`

String is the bag as Set::Bag prints it: members sorted, each with its count, as in "(apples => 3, pears => 1)".

```go
fmt.Println(scalar.NewBag("pears", "apples", "apples").String()) // sorted, as Set::Bag prints a bag
```

```
(apples => 2, pears => 1)
```

#### `Bag.MarshalJSON`

MarshalJSON encodes the bag as an object from each member to its count, in the order the members were first inserted.

```go
out, _ := json.Marshal(scalar.NewBag("pears", "apples", "apples")) // in insertion order
fmt.Println(string(out))
```

```
{"pears":1,"apples":2}
```

### Reference: Regular expressions

#### `SV.M`

M is \$s =~ /re/ with captures.

```go
s := scalar.Value("Magnitude range: M4.8 - M5.3")
if s.M(`(?i)magnitude`) != nil { // if ($s =~ /magnitude/i)
	fmt.Println("matched")
}
m := s.M(`M(\d\.\d)\s*-\s*M(\d\.\d)`)
fmt.Println(m.Get(0)) // $&
fmt.Println(m.Get(1), m.Get(2))
all := s.M(`M(\d\.\d)`, "g")                  // @all = all the matches: one array of captures each
fmt.Println(all.Length(), all.At("[1][1]"))   // $all[1][1]: the second match's $1
fmt.Println(s.M(`offshore`).Get(1).Defined()) // no match: nil, still safe to read
```

```
matched
M4.8 - M5.3
4.8 5.3
2 5.3
false
```

#### `SV.Named`

Named is \$s =~ /re/ followed by reading %+.

```go
d := scalar.Value("Origin 2026-10-18 10:00 UTC").Named(`(?<year>\d{4})-(?<month>\d\d)-(?<day>\d\d)`)
fmt.Println(d.Get("year"), d.Get("month"), d.Get("day")) // $+{year} and so on
```

```
2026 10 18
```

#### `SV.S`

S is Perl's s///.

```go
s := scalar.Value("a  b   c")
n := s.S(`\s+`, " ", "g") // $s =~ s/\s+/ /g
fmt.Println(s, n)
name := scalar.Value("Lovelace, Ada")
name.S(`(\w+), (\w+)`, "$2 $1") // $1 and $2 in the replacement
fmt.Println(name)
clean := scalar.Value("x1y2").S(`\d`, "", "gr") // s///r: a new string; the original is unchanged
fmt.Println(clean)
doubled := scalar.Value("3 apples, 4 pears")
doubled.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g") // s///e
fmt.Println(doubled)
```

```
a b c 2
Ada Lovelace
xy
6 apples, 8 pears
```

#### `SV.Split`

Split is Perl's split /re/, \$s.

```go
fmt.Println(scalar.Value("a,b,,c,,").Split(`,`).Length())        // trailing empty fields dropped
fmt.Println(scalar.Value("a1b2c").Split(`(\d)`).Join("|"))       // captures included
fmt.Println(scalar.Value("a,b,c,d").Split(`,`, 2).Join("|"))     // a limit
fmt.Println(scalar.Value("  some  words ").Split(" ").Join("|")) // awk mode
```

```
4
a|1|b|2|c
a|b,c,d
some|words
```

#### `SV.Tr`

Tr is Perl's tr/// (y///).

```go
s := scalar.Value("hello")
s.Tr("a-z", "A-Z") // $s =~ tr/a-z/A-Z/
fmt.Println(s)
fmt.Println(scalar.Value("hello world").Tr("a-z", ""))  // counts: tr/a-z// returns how many
fmt.Println(scalar.Value("aabbcc").Tr("a-z", "", "sr")) // squeeze, returning a new string
```

```
HELLO
10
abc
```

#### `Compile`

Compile compiles pattern with flags, any of "i", "m", "s" and "x", or returns an error if the pattern is invalid.

```go
re, err := scalar.Compile(`M(\d\.\d`) // a pattern from outside the program: check it
fmt.Println(re == nil, err != nil)
```

```
true true
```

#### `MustCompile`

MustCompile is Compile, panicking if the pattern is invalid.

```go
mag := scalar.MustCompile(`M(\d\.\d)`) // my $mag = qr/M(\d\.\d)/
s := scalar.Value("M4.8 then M5.3")
fmt.Println(s.M(mag, "g").Length()) // use it anywhere a pattern string goes
fmt.Println(s.S(mag, "M?", "gr"))
insensitive := scalar.MustCompile(`offshore`, "i") // flags are fixed when compiling
fmt.Println(scalar.Value("OFFSHORE").M(insensitive) != nil)
```

```
2
M? then M?
true
```

#### `Regexp.String`

String returns the pattern as written.

```go
fmt.Println(scalar.MustCompile(`\d+`, "i"))
```

```
\d+
```

#### `SV.Matches`

Matches is Perl's while (\$s =~ /re/g) { ...

```go
s := scalar.Value("Felt reports: 1,204 - 12,431")
for m := range s.Matches(`(\d[\d,]*)`) { // while ($s =~ /(\d[\d,]*)/g)
	fmt.Println(m.Get(1)) // $1
}
for m := range scalar.Value("a=1, b=2, c=3").Matches(`(\w)=(\d)`) {
	if m.Get(1).Str() == "b" {
		fmt.Println("found b:", m.Get(2))
		break // stops the search
	}
}
```

```
1,204
12,431
found b: 2
```

### Reference: Pack and unpack

#### `Pack`

Pack is Perl's pack for templates of H, h, B, b, f, d and F: it packs vals (flattened as a list is, see `AV.Push`) into bytes, returned as a byte string (see `SV.Bytes`).

```go
fmt.Println(scalar.Pack("H*", "4a6f"))     // hex digits to bytes: "Jo"
fmt.Println(scalar.Pack("B8", "01001010")) // bits to a byte: "J"

f := scalar.Pack("f>", 1.5) // a big-endian float32: 4 bytes
fmt.Println(len(f.Str()), hex.EncodeToString([]byte(f.Str())))
d := scalar.Pack("d<*", 1, 2.5) // two little-endian float64s
fmt.Println(len(d.Str()))
```

```
Jo
J
4 3fc00000
16
```

#### `Unpack`

Unpack is Perl's unpack for templates of H, h, B, b, f, d and F: it reads data, a string of bytes (a scalar, a Go string or a []byte), and returns what the template describes, as an array.

```go
fmt.Println(scalar.Unpack("H*", "Jo").Get(0))                       // bytes to hex
fmt.Println(scalar.Unpack("B8", "J").Get(0))                        // a byte to bits
fmt.Println(scalar.Unpack("H*", scalar.Pack("d>", 3.14159)).Get(0)) // a float64's bytes, in hex
fmt.Println(scalar.Unpack("B32", scalar.Pack("f>", -2)).Get(0))     // a float32's bits: sign, exponent, fraction
fmt.Println(scalar.Unpack("f<*", scalar.Pack("f<*", 1.5, 0.1)).Join(", "))
```

```
4a6f
01001010
400921f9f01b866e
11000000000000000000000000000000
1.5, 0.100000001490116
```

### Reference: JSON and JSONPath

#### `FromJSON`

FromJSON decodes data into a scalar.

```go
doc, err := scalar.FromJSON([]byte(`{"id": 9007199254740993, "depth": 8.25, "stations": ["PAS"], "felt": null}`))
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(doc.Get("id")) // whole numbers stay exact
fmt.Println(doc.Get("depth"), doc.Get("stations").Ref())
fmt.Println(doc.Exists("felt"), doc.Get("felt").Defined()) // null is undef
```

```
9007199254740993
8.25 ARRAY
true false
```

#### `SV.MarshalJSON`

MarshalJSON implements `json.Marshaler`.

```go
h := scalar.Hash("b", 1, "a", scalar.Value(scalar.Array("x", nil)))
out, _ := json.Marshal(scalar.Value(h)) // keys sorted, undef as null
fmt.Println(string(out))
h.Set("self", scalar.Value(h)) // a loop can't be JSON: an error, not a crash
_, err := json.Marshal(scalar.Value(h))
fmt.Println(err != nil)
```

```
{"a":["x",null],"b":1}
true
```

#### `AV.MarshalJSON`

MarshalJSON implements `json.Marshaler`, as `SV.MarshalJSON` does.

```go
out, _ := json.Marshal(scalar.Array(1, "two", 3.5))
fmt.Println(string(out))
```

```
[1,"two",3.5]
```

#### `HV.MarshalJSON`

MarshalJSON implements `json.Marshaler`, as `SV.MarshalJSON` does.

```go
out, _ := json.Marshal(scalar.Hash("z", 1, "a", 2))
fmt.Println(string(out))
```

```
{"a":2,"z":1}
```

#### `SV.UnmarshalJSON`

UnmarshalJSON implements `json.Unmarshaler`.

```go
var record struct {
	ID   int        `json:"id"`
	Data *scalar.SV `json:"data"` // any shape at all
}
_ = json.Unmarshal([]byte(`{"id": 1, "data": {"tags": ["a", "b"]}}`), &record)
fmt.Println(record.ID, record.Data.Get("tags").AV().Join(","))
```

```
1 a,b
```

#### `SV.Find`

Find runs a JSONPath query (RFC 9535) on s and returns every match, or nil if nothing matched.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"network": {"station": [
		{"code": "PAS", "elevation": 295},
		{"code": "GSC", "elevation": 990}
	]}
}`))
fmt.Println(doc.Find("$.network.station[*].code").Join(","))  // every code
fmt.Println(doc.Find("$..elevation").Join(","))               // every elevation, at any depth
fmt.Println(doc.Find("$.network.station[-1].code").Join(",")) // the last station's code
fmt.Println(doc.Find("$.network.sensor") == nil)              // nothing matched
```

```
PAS,GSC
295,990
GSC
true
```

#### `SV.Find_filter`

```go
doc, _ := scalar.FromJSON([]byte(`{"quakes": [
	{"place": "Northern Alaska", "mag": {"value": 3.2}, "stations": ["FAIR"]},
	{"place": "Ridgecrest, CA", "mag": {"value": 5.1, "error": 0.2}, "stations": ["PAS", "GSC"]},
	{"place": "Golfe du Saint-Laurent", "stations": []}
]}`))
fmt.Println(doc.Find("$.quakes[?@.mag.value >= 4.5].place").Join("; "))       // a comparison
fmt.Println(doc.Find("$.quakes[?!@.mag].place").Join("; "))                   // no magnitude at all
fmt.Println(doc.Find("$.quakes[?count(@.stations[*]) > 1].place").Join("; ")) // a function
fmt.Println(doc.Find("$.quakes[?match(@.place, 'N.*')].place").Join("; "))    // I-Regexp, whole string
fmt.Println(doc.Find("$..[?@ == 'GSC']").Join("; "))                          // at any depth
```

```
Ridgecrest, CA
Golfe du Saint-Laurent
Ridgecrest, CA
Northern Alaska
GSC
```

#### `QuoteJSONPath`

QuoteJSONPath returns s as a JSONPath string literal, quoted and escaped as RFC 9535 requires, to put a value or a name from Go into a query safely, as `regexp.QuoteMeta` does for a pattern:

```go
doc, _ := scalar.FromJSON([]byte(`{"regions": [
	{"name": "Alaska", "stations": ["FAIR"]},
	{"name": "Hawai'i", "stations": ["HVO", "KHU"]}
]}`))
name := "Hawai'i" // from Go, with a quote in it
fmt.Println(scalar.QuoteJSONPath(name))
fmt.Println(doc.Find("$.regions[?@.name == " + scalar.QuoteJSONPath(name) + "].stations[*]").Join(", "))
```

```
'Hawai\'i'
HVO, KHU
```

### Reference: YAML

#### `FromYAML`

FromYAML reads a YAML document and returns it, as FromJSON reads JSON: mappings become hashes, sequences arrays, and plain values scalars.

```go
values, err := scalar.FromYAML([]byte(`
defaults: &defaults
  replicas: 2
  image: {repository: web, tag: "1.0"}
production:
  <<: *defaults        # a merge key: production gets the defaults
  replicas: 5
country: no            # YAML 1.2: the string "no", not false
`))
if err != nil {
	panic(err)
}
fmt.Println(values.At("production.replicas"), values.At("production.image.tag"))
fmt.Println(values.At("production.image").HV() == values.At("defaults.image").HV()) // one hash, shared
fmt.Println(values.At("country"))
```

```
5 1.0
true
no
```

#### `FromYAMLAll`

FromYAMLAll reads a stream of YAML documents, separated by ---, as in a Kubernetes manifest or Helm's output, and returns them as an array, in order.

```go
manifest := []byte(`
apiVersion: v1
kind: Service
metadata: {name: web}
---
apiVersion: apps/v1
kind: Deployment
metadata: {name: web}
spec: {replicas: 3}
---
`)
docs, err := scalar.FromYAMLAll(manifest) // several documents, as kubectl takes them
if err != nil {
	panic(err)
}
for _, d := range docs.All() {
	fmt.Println(d.At("kind"), d.At("metadata.name"), d.At("spec.replicas").Or("-"))
}
```

```
Service web -
Deployment web 3
```

#### `SV.ToYAML`

ToYAML writes s as a YAML document, with two-space indentation, as Kubernetes and Helm use:

```go
cfg := scalar.Hash("name", "web", "replicas", 3, "country", "no", "version", "1.0",
	"ports", scalar.Value([]int{80, 443}))
out, _ := scalar.Value(cfg).ToYAML() // keys sorted; strings that any reader could misread are quoted
fmt.Print(string(out))
```

```
country: "no"
name: web
ports:
  - 80
  - 443
replicas: 3
version: "1.0"
```

#### `AV.ToYAML`

ToYAML writes a as a YAML sequence; see `SV.ToYAML`.

```go
out, _ := scalar.Array("go", 1.0, nil, true).ToYAML()
fmt.Print(string(out))
```

```
- go
- 1.0
- null
- true
```

#### `HV.ToYAML`

ToYAML writes h as a YAML mapping; see `SV.ToYAML`.

```go
shared := scalar.Value(scalar.Hash("cpu", "500m"))
h := scalar.Hash("requests", shared, "limits", shared) // the same hash twice
out, _ := h.ToYAML()
fmt.Print(string(out)) // an anchor, then an alias
```

```
limits: &1
  cpu: 500m
requests: *1
```

#### `SV.UnmarshalYAML`

UnmarshalYAML lets a scalar be a field of a struct read with the YAML package's Unmarshal, for a part of a document whose shape varies; it reads as FromYAML does.

```go
var cfg struct {
	Name  string     `yaml:"name"`
	Extra *scalar.SV `yaml:"extra"` // a part whose shape varies
}
err := yaml.Unmarshal([]byte("name: web\nextra: {ports: [80, 443]}\n"), &cfg)
fmt.Println(err, cfg.Name, cfg.Extra.At("ports[1]"))
```

```
<nil> web 443
```

#### `SV.MarshalYAML`

MarshalYAML lets a scalar be a field of a struct written with the YAML package's Marshal; it writes as ToYAML does.

```go
out, _ := yaml.Marshal(struct {
	Name  string     `yaml:"name"`
	Extra *scalar.SV `yaml:"extra"`
}{"web", scalar.Value(map[string]any{"ports": []int{80}})})
fmt.Print(string(out))
```

```
name: web
extra:
    ports:
        - 80
```

#### `AV.MarshalYAML`

MarshalYAML lets an array be written with the YAML package's Marshal.

```go
out, _ := yaml.Marshal(map[string]any{"ports": scalar.Array(80, 443)})
fmt.Print(string(out))
```

```
ports:
    - 80
    - 443
```

#### `HV.MarshalYAML`

MarshalYAML lets a hash be written with the YAML package's Marshal.

```go
out, _ := yaml.Marshal(map[string]any{"labels": scalar.Hash("app", "web")})
fmt.Print(string(out))
```

```
labels:
    app: web
```

### Reference: Into Go values

#### `To`

To converts v into a Go value of type T, the type in brackets: the way out of scalar into ordinary Go, once you know the shape you want.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"id": "ci40123456",
	"stations": ["PAS", "GSC"],
	"depth": {"km": "8", "error": 2}
}`))
type Event struct {
	ID       string   `json:"id"`
	Stations []string `json:"stations"`
	Depth    struct {
		Km    int `json:"km"` // "8" fills it, as Perl would numify it
		Error int `json:"error"`
	} `json:"depth"`
}
e, err := scalar.To[Event](doc) // the type you want, in brackets
if err != nil {
	panic(err)
}
fmt.Println(e.ID, e.Stations, e.Depth.Km, e.Depth.Error)
```

```
ci40123456 [PAS GSC] 8 2
```

#### `To.slice`

```go
doc, _ := scalar.FromJSON([]byte(`{"stations": ["PAS", "GSC"], "felt": [3, "5"]}`))
stations, _ := scalar.To[[]string](doc.At("stations")) // from a scalar holding an array
felt, _ := scalar.To[[]int](doc.At("felt").AV())       // or from the array itself
fmt.Printf("%q %v\n", stations, felt)
```

```
["PAS" "GSC"] [3 5]
```

#### `To.number`

```go
felt, err := scalar.To[int](scalar.Value(" 1204 ")) // a numeric string, as Perl reads it
fmt.Println(felt, err)
mag, err := scalar.To[float64](scalar.Value("4.5"))
fmt.Println(mag, err)
```

```
1204 <nil>
4.5 <nil>
```

#### `To.map`

```go
doc, _ := scalar.FromJSON([]byte(`{"ci": 3, "us": "2"}`))
counts, _ := scalar.To[map[string]int](doc) // a hash fills a map
fmt.Println(counts["ci"], counts["us"])
```

```
3 2
```

#### `To.set`

```go
networks := scalar.NewSet("ci", "us", 5)
members, _ := scalar.To[[]string](networks) // a set gives its members, in insertion order
fmt.Printf("%q\n", members)
```

```
["ci" "us" "5"]
```

#### `To.bag`

```go
regions := scalar.NewBag("Alaska", "Alaska", "Nevada")
counts, _ := scalar.To[map[string]int](regions)    // a bag gives its counts
names, _ := scalar.To[[]string](regions.Members()) // and Members its distinct members
fmt.Println(counts["Alaska"], counts["Nevada"], names)
```

```
2 1 [Alaska Nevada]
```

#### `To.any`

```go
doc, _ := scalar.FromJSON([]byte(`{"types": ["origin", 1]}`))
plain, _ := scalar.To[any](doc) // the plain Go form, as Any gives it
fmt.Printf("%v\n", plain)
```

```
map[types:[origin 1]]
```

#### `To.missing`

```go
doc, _ := scalar.FromJSON([]byte(`{"felt": null}`))
felt, err := scalar.To[*int](doc.At("felt")) // null, or missing: the zero value, here nil
fmt.Println(felt, err)
products, err := scalar.To[[]string](doc.At("products"))
fmt.Println(products == nil, err)
```

```
<nil> <nil>
true <nil>
```

#### `To.optional`

An optional value is a pointer: nil if the value is undef, missing or null, and otherwise a pointer to it, converted and checked. There's no StrPtr or NumPtr method, as To\[\*T] does it for any type, and refuses a value that doesn't fit, where Num would quietly make "abc" 0.

```go
doc, _ := scalar.FromJSON([]byte(`{"value": 4.5, "error": null, "type": "ml", "felt": "widely"}`))

value, _ := scalar.To[*float64](doc.At("value"))
magErr, _ := scalar.To[*float64](doc.At("error")) // null: nil
none, _ := scalar.To[*string](doc.At("network"))  // missing: nil
fmt.Println(*value, magErr, none)

_, err := scalar.To[*float64](doc.At("felt")) // "widely" isn't a number
fmt.Println(err)

// The same, for a struct's optional fields, as a feed's often are.
type Mag struct {
	Value, Error *float64
	Type         *string
}
m, _ := scalar.To[Mag](doc)
fmt.Println(*m.Value, m.Error, *m.Type)
```

```
4.5 <nil> <nil>
scalar: Decode: want float64, got "widely"
4.5 <nil> ml
```

#### `To.error`

```go
doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": "unknown"}}`))
type Event struct {
	Depth struct {
		Km int `json:"km"`
	} `json:"depth"`
}
e, err := scalar.To[Event](doc)
fmt.Println(err)
fmt.Println(e.Depth.Km) // on an error, the zero value
var de *scalar.DecodeError
if errors.As(err, &de) {
	fmt.Println(doc.At(de.Path)) // the path leads At to the value
}
```

```
scalar: Decode: depth.km: want int, got "unknown"
0
unknown
```

#### `To.strict`

```go
felt := scalar.Array("3", "five", 7)
fmt.Println(felt.IntSlice())     // Perl's numeric context: never fails, "five" is 0
_, err := scalar.To[[]int](felt) // To checks
fmt.Println(err)
```

```
[3 0 7]
scalar: Decode: [1]: want int, got "five"
```

#### `To.loose`

```go
doc, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "extra": {"products": ["shakemap"]}}`))
type Event struct {
	ID    string     `json:"id"`
	Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar: a deep copy
}
e, _ := scalar.To[Event](doc)
fmt.Println(e.ID, e.Extra.At("products[0]"))
```

```
ci40123456 shakemap
```

#### `SV.Decode`

Decode stores s in the Go value v points to, as `json.Unmarshal` would store the same data, but converting each value as Perl would, so a number that came as a string, "8200", fills an int field.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"id": "ci40123456",
	"stations": ["PAS", "GSC"],
	"depth": {"km": "8", "error": 2},
	"extra": {"products": ["shakemap"]}
}`))
var event struct {
	ID       string   `json:"id"`
	Stations []string `json:"stations"`
	Depth    struct {
		Km    int `json:"km"` // "8" fills it, as Perl would numify it
		Error int `json:"error"`
	} `json:"depth"`
	Extra *scalar.SV `json:"extra"` // a part that stays loose
}
if err := doc.Decode(&event); err != nil {
	panic(err)
}
fmt.Println(event.ID, event.Stations, event.Depth.Km, event.Depth.Error)
fmt.Println(event.Extra.At("products[0]"))
```

```
ci40123456 [PAS GSC] 8 2
shakemap
```

#### `DecodeError.Error`

```go
doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": "unknown"}}`))
var event struct {
	Depth struct {
		Km int `json:"km"`
	} `json:"depth"`
}
err := doc.Decode(&event)
fmt.Println(err)
var de *scalar.DecodeError
if errors.As(err, &de) {
	fmt.Println(doc.At(de.Path)) // the path leads At to the value
}
```

```
scalar: Decode: depth.km: want int, got "unknown"
unknown
```

<!-- reference:end -->
