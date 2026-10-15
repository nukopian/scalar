package scalar_test

// Examples of scalars: making them, reading them in each context, and
// following references. They're run as tests, and they make up the
// Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
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
	doc, _ := scalar.FromJSON([]byte(`{"skills": ["Go", "SQL"], "salary": {"max": 55000}}`))
	fmt.Println(doc.Equal(map[string]any{ // is_deeply($doc, {...})
		"skills": []string{"Go", "SQL"},
		"salary": map[string]any{"max": "55000"}, // as strings, as eq compares
	}))
	fmt.Println(doc.At("salary.max").Equal(55000.0), scalar.Value(nil).Equal("")) // undef equals only undef
	// Output:
	// true
	// true false
}

func ExampleSV_Diff() {
	rules, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
	model, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "Docker"]}`))
	fmt.Println(rules.Diff(model)) // where they first differ, as a path for At
	fmt.Printf("%q\n", rules.Diff(rules.Clone()))
	// Output:
	// skills[1]: got "SQL", want "Docker"
	// ""
}

func ExampleSV_Apply() {
	doc, _ := scalar.FromJSON([]byte(`{"title": " Go Developer ", "skills": [" Go", "SQL "], "salary": 50000}`))
	doc.Apply(func(v *scalar.SV) any { // rmap { $_ = trim $_ } $doc
		if v.Kind() == scalar.String {
			return v.Trim()
		}
		return v // anything else as it is
	})
	out, _ := doc.MarshalJSON()
	fmt.Println(string(out))
	// Output:
	// {"salary":50000,"skills":["Go","SQL"],"title":"Go Developer"}
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
	figure := func(v *scalar.SV) any { return v.M(`£([\d,]+)`).Get(1).Then(digits) } // undef if no figure
	annual := func(v *scalar.SV) any { return v.Mul(12) }
	for _, text := range []string{"£4,000 a month", "Competitive"} {
		fmt.Println(scalar.Value(text).Then(figure).Then(annual).Or("unknown")) // stops at the first undef
	}
	// Output:
	// 48000
	// unknown
}

func ExampleSV_When() {
	positive := func(v *scalar.SV) bool { return v.Num() > 0 }
	doc, _ := scalar.FromJSON([]byte(`{"min": 45000, "max": 0}`))
	fmt.Println(doc.At("min").When(positive).Or("none"))
	fmt.Println(doc.At("max").When(positive).Or("none"))   // fails the test: undef
	fmt.Println(doc.At("bonus").When(positive).Or("none")) // missing: positive isn't called
	// Output:
	// 45000
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
	doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"], "salary": {"min": 45000, "max": null}}`))
	fmt.Println(doc.Dump()) // print Dumper($doc), as Data::Dumper::Concise prints it
	// Output:
	// {
	//   salary => {
	//     max => undef,
	//     min => 45000,
	//   },
	//   skills => [
	//     "Go",
	//     "SQL",
	//   ],
	//   title => "Go Developer",
	// }
}

func ExampleFromDump() {
	tags := scalar.Value(scalar.Array("go"))
	doc := scalar.Hash("title", "Go Developer", "tags", tags, "again", tags)         // the same array twice
	back, err := scalar.FromDump(scalar.Value(doc).Dump())                          // a round trip
	fmt.Println(err, back.Equal(doc), back.At("tags").AV() == back.At("again").AV()) // still one array

	perl, err := scalar.FromDump(`bless( {'name' => 'Ada', 'langs' => ['Perl', 'Go']}, 'Person' )`) // Data::Dumper's output
	fmt.Println(err, perl.At("name"), perl.At("langs[1]"))
	// Output:
	// <nil> true true
	// <nil> Ada Go
}

func ExampleMustFromDump() {
	want := scalar.MustFromDump(`{
		title  => "Go Developer",
		skills => ["Go", "SQL"],   # test data, as Perl would write it
	}`)
	got, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
	fmt.Printf("%q\n", got.Diff(want))
	// Output:
	// ""
}

func ExampleMustFromDump_testData() {
	// The expected value in a test: shorter than JSON, with comments where
	// they help.
	want := scalar.MustFromDump(`{
		title     => "Senior Go Developer",
		seniority => "Senior",
		salary    => { min => 60000, max => undef },   # max: not stated
		skills    => ["Go", "PostgreSQL", "Docker"],
		remote    => !!1,
	}`)
	got, _ := scalar.FromJSON([]byte(`{"title": "Senior Go Developer", "seniority": "Senior",
		"salary": {"min": 60000, "max": null}, "skills": ["Go", "PostgreSQL", "Docker"], "remote": true}`))
	fmt.Printf("%q\n", got.Diff(want))
	got.At("skills[2]").Set("Kubernetes")
	fmt.Println(got.Diff(want)) // where they differ, as a path for At
	// Output:
	// ""
	// skills[2]: got "Kubernetes", want "Docker"
}

func ExampleMustFromDump_toStruct() {
	// Test data straight into the struct a function under test takes.
	type Salary struct{ Min, Max int }
	type Posting struct {
		Title  string
		Salary Salary
		Skills []string
	}
	p, err := scalar.To[Posting](scalar.MustFromDump(`{
		Title  => 'Data Engineer',
		Salary => { Min => 45_000, Max => 55_000 },   # underscores, as in Perl
		Skills => ['Python', 'SQL'],
	}`))
	fmt.Printf("%+v %v\n", p, err)
	// Output:
	// {Title:Data Engineer Salary:{Min:45000 Max:55000} Skills:[Python SQL]} <nil>
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
		title  => "Go Developer",
		skills => ["Go" "SQL"],
	}`)
	fmt.Println(err) // a line and column to look at
	_, err = scalar.FromDump(`{ handler => sub { "DUMMY" } }`)
	fmt.Println(err) // data only: no Perl is ever run
	// Output:
	// scalar: bad dump at line 3, column 19: expected a comma or "]", found '"'
	// scalar: bad dump at line 1, column 14: sub can't be read: FromDump reads data, not Perl code
}

func ExampleSV_Bytes() {
	png := scalar.Value([]byte("\x89PNG\r\n\x1a\n..."))  // a []byte makes a byte string
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
	doc, _ := scalar.FromJSON([]byte(`{"salary": null}`))
	_, ok := doc.PathOK("salary") // present, but null
	fmt.Println(ok)
	_, ok = doc.PathOK("bonus") // missing: and nothing is created
	fmt.Println(ok, doc.HV().Exists("bonus"))
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
	doc, _ := scalar.FromJSON([]byte(`{"salary": {"min": 45000, "max": null}}`))
	_, ok := doc.AtOK("salary.max") // present, but null
	fmt.Println(ok)
	_, ok = doc.AtOK("salary.bonus") // missing: and nothing is created
	fmt.Println(ok, doc.At("salary").HV().Exists("bonus"))
	// Output:
	// true
	// false false
}

func ExampleSV_DeleteAt() {
	doc, _ := scalar.FromJSON([]byte(`{"salary": {"min": 45000, "bonus": "TBC"}, "skills": ["Go", "SQL"]}`))
	fmt.Println(doc.DeleteAt("salary.bonus")) // delete $doc->{salary}{bonus}: its value
	fmt.Println(doc.DeleteAt("skills[-1]"))   // the last element: the array shrinks
	fmt.Println(doc.DeleteAt("benefits.car")) // nothing there: undef, and nothing created
	fmt.Println(doc.Dump())
	// Output:
	// TBC
	// SQL
	//
	// {
	//   salary => {
	//     min => 45000,
	//   },
	//   skills => [
	//     "Go",
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
