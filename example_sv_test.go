package scalar_test

// Examples of scalars: making them, reading them in each context, and
// following references. They're run as tests, and they make up
// docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleValue() {
	fmt.Println(scalar.Value(42).Kind() == scalar.Int)
	fmt.Println(scalar.Value(3.5).Kind() == scalar.Float)
	fmt.Println(scalar.Value("text").Kind() == scalar.String)
	fmt.Println(scalar.Value([]string{"a", "b"}).Ref()) // a slice becomes an array reference
	fmt.Println(scalar.Value(nil).Defined())            // nil is undef
	// Output:
	// true
	// true
	// true
	// ARRAY
	// false
}

// A struct becomes a hash, keyed as encoding/json would write it: by its
// json tags, leaving out what's omitempty and empty. Decode is the way back.
func ExampleValue_struct() {
	type station struct {
		Code string `json:"code"`
		Name string `json:"name,omitempty"`
	}
	st := scalar.Value(station{Code: "PAS"})
	fmt.Println(st.Ref(), st.At("code"), st.At("name").Defined())
	// Output:
	// HASH PAS false
}

func ExampleSV_Set() {
	s := scalar.Value("old")
	s.Set(10)                     // $s = 10
	fmt.Println(s.Add(1))         // it's a number now
	s.Set(map[string]any{"a": 1}) // a map becomes a hash reference
	fmt.Println(s.Ref())
	// Output:
	// 11
	// HASH
}

func ExampleSV_Copy() {
	h := scalar.Hash("count", 1)
	alias := h.Get("count")      // the stored value itself
	own := h.Get("count").Copy() // my $own = $h{count}
	alias.Set(2)
	own.Set(99)
	fmt.Println(h.Get("count"))
	// Output:
	// 2
}

func ExampleSV_Clone() {
	h := scalar.Hash("defaults", scalar.Value(map[string]any{"tags": []any{"go"}}))
	shallow := h.Get("defaults").Copy() // my $row = $h{defaults}: the same hash
	deep := h.Get("defaults").Clone()   // my $row = dclone($h{defaults}): its own
	deep.At("tags").AV().Push("perl")
	fmt.Println(h.Get("defaults").At("tags").AV().Join(","), deep.At("tags").AV().Join(","))
	shallow.At("tags").AV().Push("sql") // changes the original
	fmt.Println(h.Get("defaults").At("tags").AV().Join(","))
	// Output:
	// go go,perl
	// go,sql
}

func ExampleSV_Equal() {
	doc, _ := scalar.FromJSON([]byte(`{"stations": ["PAS", "GSC"], "depth": {"max": 12}}`))
	fmt.Println(doc.Equal(map[string]any{ // is_deeply($doc, {...})
		"stations": []string{"PAS", "GSC"},
		"depth":    map[string]any{"max": "12"}, // as strings, as eq compares
	}))
	fmt.Println(doc.At("depth.max").Equal(12.0), scalar.Value(nil).Equal("")) // undef equals only undef
	// Output:
	// true
	// true false
}

func ExampleSV_Diff() {
	ours, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"]}`))
	theirs, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "ISA"]}`))
	fmt.Println(ours.Diff(theirs)) // where they first differ, as a path for At
	fmt.Printf("%q\n", ours.Diff(ours.Clone()))
	// Output:
	// stations[1]: got "GSC", want "ISA"
	// ""
}

func ExampleSV_Apply() {
	doc, _ := scalar.FromJSON([]byte(`{"place": " 5 km W of Cobb, CA ", "stations": [" PAS", "GSC "], "depth": 2}`))
	doc.Apply(func(v *scalar.SV) any { // rmap { $_ = trim $_ } $doc
		if v.Kind() == scalar.String {
			return v.Trim()
		}
		return v // anything else as it is
	})
	out, _ := doc.MarshalJSON()
	fmt.Println(string(out))
	// Output:
	// {"depth":2,"place":"5 km W of Cobb, CA","stations":["PAS","GSC"]}
}

func ExampleSV_Kind() {
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
	// Output:
	// undef
	// bool
	// int
	// float
	// string
	// bytes
	// array
	// hash
}

func ExampleSV_Defined() {
	fmt.Println(scalar.Value(nil).Defined())
	fmt.Println(scalar.Value("").Defined()) // empty, but defined
	fmt.Println(scalar.Hash().Get("missing").Defined())
	// Output:
	// false
	// true
	// false
}

func ExampleSV_Any() {
	s := scalar.Value(map[string]any{"tags": []any{"go", 1}})
	plain := s.Any().(map[string]any) // plain Go, sharing nothing with s
	fmt.Println(plain["tags"])
	// Output:
	// [go 1]
}

func ExampleSV_Or() {
	fmt.Println(scalar.Value(nil).Or("default")) // $x // "default"
	fmt.Println(scalar.Value(0).Or("default"))   // 0 is defined, so it's kept
	// Output:
	// default
	// 0
}

func ExampleSV_OrSet() {
	h := scalar.Hash()
	h.Get("retries").OrSet(3) // $h{retries} //= 3
	h.Get("retries").OrSet(5) // already set, so unchanged
	fmt.Println(h.Get("retries"))
	// Output:
	// 3
}

func ExampleSV_Then() {
	digits := func(v *scalar.SV) any { return v.S(`,`, "", "gr") }
	figure := func(v *scalar.SV) any { return v.M(`([\d,]+) m\b`).Get(1).Then(digits) } // undef if no figure
	km := func(v *scalar.SV) any { return v.Div(1000) }
	for _, text := range []string{"Depth: 12,500 m", "Depth not determined"} {
		fmt.Println(scalar.Value(text).Then(figure).Then(km).Or("unknown")) // stops at the first undef
	}
	// Output:
	// 12.5
	// unknown
}

func ExampleSV_When() {
	positive := func(v *scalar.SV) bool { return v.Num() > 0 }
	doc, _ := scalar.FromJSON([]byte(`{"felt": 1204, "tsunami": 0}`))
	fmt.Println(doc.At("felt").When(positive).Or("none"))
	fmt.Println(doc.At("tsunami").When(positive).Or("none")) // fails the test: undef
	fmt.Println(doc.At("cdi").When(positive).Or("none"))     // missing: positive isn't called
	// Output:
	// 1204
	// none
	// none
}

func ExampleSV_Ref() {
	fmt.Println(scalar.Value(scalar.Array(1)).Ref())
	fmt.Println(scalar.Value(scalar.Hash()).Ref())
	fmt.Printf("%q\n", scalar.Value("text").Ref())
	// Output:
	// ARRAY
	// HASH
	// ""
}

func ExampleSV_Str() {
	fmt.Println(scalar.Value(42).Str())
	fmt.Println(scalar.Value(0.1 + 0.2).Str()) // Perl's 15 significant digits
	fmt.Println(scalar.Value(true).Str())
	fmt.Printf("%q\n", scalar.Value(nil).Str())
	// Output:
	// 42
	// 0.3
	// 1
	// ""
}

func ExampleSV_String() {
	fmt.Println(scalar.Value(1e21)) // fmt uses String, as Perl's print would
	// Output:
	// 1e+21
}

func ExampleSV_Dump() {
	doc, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"], "mag": {"value": 5.1, "error": null}}`))
	fmt.Println(doc.Dump()) // print Dumper($doc), as Data::Dumper::Concise prints it
	// Output:
	// {
	//   id => "ci40123456",
	//   mag => {
	//     error => undef,
	//     value => "5.1",
	//   },
	//   stations => [
	//     "PAS",
	//     "GSC",
	//   ],
	// }
}

func ExampleFromDump() {
	stations := scalar.Value(scalar.Array("PAS"))
	doc := scalar.Hash("id", "ci40123456", "stations", stations, "again", stations)      // the same array twice
	back, err := scalar.FromDump(scalar.Value(doc).Dump())                               // a round trip
	fmt.Println(err, back.Equal(doc), back.At("stations").AV() == back.At("again").AV()) // still one array

	perl, err := scalar.FromDump(`bless( {'code' => 'PAS', 'channels' => ['BHZ', 'BHN']}, 'Station' )`) // Data::Dumper's output
	fmt.Println(err, perl.At("code"), perl.At("channels[1]"))
	// Output:
	// <nil> true true
	// <nil> PAS BHN
}

func ExampleMustFromDump() {
	want := scalar.MustFromDump(`{
		id       => "ci40123456",
		stations => ["PAS", "GSC"],   # test data, as Perl would write it
	}`)
	got, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "stations": ["PAS", "GSC"]}`))
	fmt.Printf("%q\n", got.Diff(want))
	// Output:
	// ""
}

func ExampleMustFromDump_testData() {
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
	// Output:
	// ""
	// stations[2]: got "SVD", want "ISA"
}

func ExampleMustFromDump_toStruct() {
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
	// Output:
	// {ID:ci40123456 Origin:{Lat:35.7 Lon:-117.6 Depth:8200} Stations:[PAS GSC]} <nil>
}

func ExampleFromDump_keysAndStrings() {
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
	// Output:
	// -1 42 bare_word double email log.level single
	// "it's \\n literal"
	// "tab\there☺"
	// a number as a key | so is a negative one | debug | jo@example.com
}

func ExampleFromDump_bytes() {
	doc := scalar.MustFromDump(`["\211PNG\r\n", "caf\x{e9}", "caf\351"]`)
	for _, s := range doc.AV().All() {
		fmt.Println(s.Kind() == scalar.Bytes, s.Length()) // octal above 127: bytes; \x{...}: text
	}
	fmt.Println(doc.At("[1]").Eq(doc.At("[2]"))) // the same characters, so eq
	// Output:
	// true 6
	// false 4
	// true 4
	// true
}

func ExampleFromDump_shared() {
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
	// Output:
	// 1 1
	// 1
}

func ExampleFromDump_fromPerl() {
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
	// Output:
	// <nil> Ada Perl/Go 2026
}

func ExampleFromDump_types() {
	// A dump doesn't record types: each value comes back as its text says,
	// as Perl reads a dump back.
	doc := scalar.MustFromDump(`[42, "42", "2.5", 2.5, "12345678901", !!1, undef]`)
	names := map[scalar.Kind]string{scalar.Int: "int", scalar.String: "string", scalar.Float: "float",
		scalar.Bool: "bool", scalar.Undef: "undef"}
	fmt.Println(doc.AV().Map(func(v *scalar.SV) any { return names[v.Kind()] }).Join(" "))
	fmt.Println(scalar.Value("42").Dump(), scalar.Value(2.5).Dump()) // so Dump's "42" reads back as 42, and 2.5 as "2.5"
	// Output:
	// int string string float string bool undef
	// 42 "2.5"
}

func ExampleFromDump_errors() {
	_, err := scalar.FromDump(`{
		id       => "ci40123456",
		stations => ["PAS" "GSC"],
	}`)
	fmt.Println(err) // a line and column to look at
	_, err = scalar.FromDump(`{ handler => sub { "DUMMY" } }`)
	fmt.Println(err) // data only: no Perl is ever run
	// Output:
	// scalar: bad dump at line 3, column 22: expected a comma or "]", found '"'
	// scalar: bad dump at line 1, column 14: sub can't be read: FromDump reads data, not Perl code
}

func ExampleSV_Bytes() {
	png := scalar.Value([]byte("\x89PNG\r\n\x1a\n..."))   // a []byte makes a byte string
	fmt.Println(png.Kind() == scalar.Bytes, png.Length()) // Length counts bytes
	fmt.Println(png.M(`^\x89PNG`) != nil)                 // patterns match bytes
	fmt.Printf("%x\n", png.Substr(0, 4))                  // %x gives the bytes in hex
	blob := png.Bytes()                                   // back to a Go []byte (a copy)
	fmt.Println(len(blob))
	fmt.Println(scalar.Value("café").Bytes()) // text gives its UTF-8 bytes
	// Output:
	// true 11
	// true
	// 89504e47
	// 11
	// [99 97 102 195 169]
}

func ExampleSV_Format() {
	fmt.Printf("%d|%05.2f|%x|%5s|%t\n",
		scalar.Value("42 apples"), scalar.Value("3.14159"), scalar.Value(255),
		scalar.Value("ab"), scalar.Value("0"))
	// Output:
	// 42|03.14|ff|   ab|false
}

func ExampleSV_Num() {
	fmt.Println(scalar.Value("3.5kg").Num()) // the numeric prefix, as Perl does
	fmt.Println(scalar.Value("abc").Num())
	// Output:
	// 3.5
	// 0
}

func ExampleSV_NumOK() {
	for _, x := range []any{"42", " 42 ", "42abc", "", nil} {
		n, ok := scalar.Value(x).NumOK()
		fmt.Println(n, ok) // ok is false wherever Perl would warn
	}
	// Output:
	// 42 true
	// 42 true
	// 42 false
	// 0 false
	// 0 false
}

func ExampleSV_Int() {
	fmt.Println(scalar.Value(-3.9).Int()) // truncates toward zero
	fmt.Println(scalar.Value("12 items").Int())
	// Output:
	// -3
	// 12
}

func ExampleSV_IntOK() {
	i, ok := scalar.Value("9223372036854775807").IntOK() // exact, however large
	fmt.Println(i, ok)
	i, ok = scalar.Value(1e30).IntOK() // out of range: clamped
	fmt.Println(i, ok)
	// Output:
	// 9223372036854775807 true
	// 9223372036854775807 false
}

func ExampleSV_Bool() {
	for _, x := range []any{nil, "", "0", 0, 0.0, "0.0", "00", " ", "a"} {
		fmt.Print(scalar.Value(x).Bool(), " ")
	}
	fmt.Println()
	// Output:
	// false false false false false true true true true
}

func ExampleSV_AV() {
	var s scalar.SV   // undef
	s.AV().Push(1, 2) // push @$s, 1, 2: autovivifies an array
	fmt.Println(s.AV().Join(","))
	fmt.Println(scalar.Value("text").AV() == nil) // not an array
	// Output:
	// 1,2
	// true
}

func ExampleSV_HV() {
	var s scalar.SV
	s.HV().Set("a", 1) // $s->{a} = 1
	fmt.Println(s.Ref(), s.HV().Get("a"))
	// Output:
	// HASH 1
}

func ExampleSV_Path() {
	doc, _ := scalar.FromJSON([]byte(`{"items": [{"name": "first"}]}`))
	fmt.Println(doc.Path("items", 0, "name"))  // $doc->{items}[0]{name}
	fmt.Println(doc.Path("items", -1, "name")) // negative indices count from the end
	doc.Path("meta", "count").Set(1)           // writing autovivifies
	fmt.Println(doc.Path("meta", "count"))
	// Output:
	// first
	// first
	// 1
}

func ExampleSV_PathOK() {
	doc, _ := scalar.FromJSON([]byte(`{"alert": null}`))
	_, ok := doc.PathOK("alert") // present, but null
	fmt.Println(ok)
	_, ok = doc.PathOK("felt") // missing: and nothing is created
	fmt.Println(ok, doc.HV().Exists("felt"))
	// Output:
	// true
	// false false
}

func ExampleSV_At() {
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
	// Output:
	// A
	// B
	// B
	// debug
	// zero zero
	// B
	// debug
	// {"rules":[{"name":"first"}]}
}

func ExampleSV_AtOK() {
	doc, _ := scalar.FromJSON([]byte(`{"mag": {"value": 5.1, "error": null}}`))
	_, ok := doc.AtOK("mag.error") // present, but null
	fmt.Println(ok)
	_, ok = doc.AtOK("mag.type") // missing: and nothing is created
	fmt.Println(ok, doc.At("mag").HV().Exists("type"))
	// Output:
	// true
	// false false
}

func ExampleSV_DeleteAt() {
	doc, _ := scalar.FromJSON([]byte(`{"felt": {"reports": 1204, "source": "DYFI"}, "stations": ["PAS", "GSC"]}`))
	fmt.Println(doc.DeleteAt("felt.source"))  // delete $doc->{felt}{source}: its value
	fmt.Println(doc.DeleteAt("stations[-1]")) // the last element: the array shrinks
	fmt.Println(doc.DeleteAt("origin.depth")) // nothing there: undef, and nothing created
	fmt.Println(doc.Dump())
	// Output:
	// DYFI
	// GSC
	//
	// {
	//   felt => {
	//     reports => 1204,
	//   },
	//   stations => [
	//     "PAS",
	//   ],
	// }
}

func ExampleMustPath() {
	rule := scalar.MustPath("content[0].rule_set") // parsed once, like a compiled regex
	doc, _ := scalar.FromJSON([]byte(`{"content": [{"rule_set": {"name": "A", "ids": [7, 8]}}]}`))
	fmt.Println(doc.Path(rule, "name"))   // a compiled path, then more keys
	fmt.Println(doc.Path(rule, "ids", 1)) // ...{ids}[1]
	_, ok := doc.PathOK(rule, "missing")
	fmt.Println(ok)
	// Output:
	// A
	// 8
	// false
}

func ExampleCompilePath() {
	p, err := scalar.CompilePath("content..rule_set") // a path from outside the program: check it
	fmt.Println(p == nil, err)
	// Output:
	// true scalar: bad path "content..rule_set" at offset 8: expected a name
}

func ExamplePathExpr_String() {
	fmt.Println(scalar.MustPath(`meta["log.level"]`))
	// Output:
	// meta["log.level"]
}

func ExampleStrCmp() {
	fmt.Println(scalar.Array("b", "10", "a", "9").Sort(scalar.StrCmp).Join(" ")) // sort { $a cmp $b }
	// Output:
	// 10 9 a b
}

func ExampleNumCmp() {
	fmt.Println(scalar.Array("10", 9, "2.5").Sort(scalar.NumCmp).Join(" ")) // sort { $a <=> $b }
	// Output:
	// 2.5 9 10
}

func ExampleSV_DeletePath() {
	doc, _ := scalar.FromJSON([]byte(`{"depth":{"km":8,"error":2},"v1.2":{"note":"x"}}`))
	fmt.Println(doc.DeletePath("depth", "error")) // delete $doc->{depth}{error}
	fmt.Println(doc.DeletePath("v1.2", "note"))   // keys are taken whole, dots and all
	fmt.Println(doc.At("depth").HV().KeysSlice(), doc.DeletePath("depth", "type").Defined())
	// Output:
	// 2
	// x
	// [km] false
}

func ExampleSV_Undef() {
	h := scalar.Hash("felt", 1204)
	fmt.Println(h.Get("felt").Undef().Defined()) // undef $h{felt}: returns it, now undef
	fmt.Println(h.Exists("felt"))                // the key stays, as in Perl; Delete removes it
	// Output:
	// false
	// true
}

// Lock makes a structure read-only all the way down, as Perl's Hash::Util
// does a hash: its keys are fixed, so even reading one it hasn't got
// panics, where it would have autovivified, and a misspelled key is caught
// at once.
func ExampleSV_Lock() {
	doc, _ := scalar.FromJSON([]byte(`{"place": "5 km W of Cobb, CA", "depth": {"km": 2}}`))
	doc.Lock()
	fmt.Println(doc.At("depth.km"))
	fmt.Println(catch(func() { doc.At("depth.error").Set(0.5) }))
	fmt.Println(catch(func() { doc.At("plcae") })) // misspelled
	// Output:
	// 2
	// scalar: Attempt to access disallowed key 'error' in a restricted hash
	// scalar: Attempt to access disallowed key 'plcae' in a restricted hash
}

// Unlock undoes Lock, all the way down.
func ExampleSV_Unlock() {
	doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": 2}}`))
	doc.Lock().Unlock()
	doc.At("depth.error").Set(0.5)
	fmt.Println(doc.At("depth.error"))
	// Output:
	// 0.5
}

// Locked reports whether Lock has made a value read-only; a Clone isn't.
func ExampleSV_Locked() {
	doc := scalar.Value(map[string]any{"a": 1}).Lock()
	fmt.Println(doc.Locked(), doc.Clone().Locked())
	// Output:
	// true false
}

// catch runs f and returns what it panicked with.
func catch(f func()) (msg any) {
	defer func() { msg = recover() }()
	f()
	return nil
}
