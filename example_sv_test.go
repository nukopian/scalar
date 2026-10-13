package scalar_test

// Examples of scalars: making them, reading them in each context, and
// following references. They're run as tests, and they make up the
// Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleScalar() {
	fmt.Println(scalar.Scalar(42).Kind() == scalar.Int)
	fmt.Println(scalar.Scalar(3.5).Kind() == scalar.Float)
	fmt.Println(scalar.Scalar("text").Kind() == scalar.String)
	fmt.Println(scalar.Scalar([]string{"a", "b"}).Ref()) // a slice becomes an array reference
	fmt.Println(scalar.Scalar(nil).Defined())            // nil is undef
	// Output:
	// true
	// true
	// true
	// ARRAY
	// false
}

func ExampleSV_Set() {
	s := scalar.Scalar("old")
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

func ExampleSV_Kind() {
	for _, x := range []any{nil, true, 1, 1.5, "s", []any{}, map[string]any{}} {
		switch scalar.Scalar(x).Kind() {
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
	// array
	// hash
}

func ExampleSV_Defined() {
	fmt.Println(scalar.Scalar(nil).Defined())
	fmt.Println(scalar.Scalar("").Defined()) // empty, but defined
	fmt.Println(scalar.Hash().Get("missing").Defined())
	// Output:
	// false
	// true
	// false
}

func ExampleSV_Any() {
	s := scalar.Scalar(map[string]any{"tags": []any{"go", 1}})
	plain := s.Any().(map[string]any) // plain Go, sharing nothing with s
	fmt.Println(plain["tags"])
	// Output:
	// [go 1]
}

func ExampleSV_Or() {
	fmt.Println(scalar.Scalar(nil).Or("default")) // $x // "default"
	fmt.Println(scalar.Scalar(0).Or("default"))   // 0 is defined, so it's kept
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

func ExampleSV_Ref() {
	fmt.Println(scalar.Scalar(scalar.Array(1)).Ref())
	fmt.Println(scalar.Scalar(scalar.Hash()).Ref())
	fmt.Printf("%q\n", scalar.Scalar("text").Ref())
	// Output:
	// ARRAY
	// HASH
	// ""
}

func ExampleSV_Str() {
	fmt.Println(scalar.Scalar(42).Str())
	fmt.Println(scalar.Scalar(0.1 + 0.2).Str()) // Perl's 15 significant digits
	fmt.Println(scalar.Scalar(true).Str())
	fmt.Printf("%q\n", scalar.Scalar(nil).Str())
	// Output:
	// 42
	// 0.3
	// 1
	// ""
}

func ExampleSV_String() {
	fmt.Println(scalar.Scalar(1e21)) // fmt uses String, as Perl's print would
	// Output:
	// 1e+21
}

func ExampleSV_Bytes() {
	png := scalar.Scalar([]byte("\x89PNG\r\n\x1a\n..."))  // a []byte makes a byte string
	fmt.Println(png.Kind() == scalar.Bytes, png.Length()) // Length counts bytes
	fmt.Println(png.M(`^\x89PNG`) != nil)                 // patterns match bytes
	fmt.Printf("%x\n", png.Substr(0, 4))                  // %x gives the bytes in hex
	blob := png.Bytes()                                   // back to a Go []byte (a copy)
	fmt.Println(len(blob))
	fmt.Println(scalar.Scalar("café").Bytes()) // text gives its UTF-8 bytes
	// Output:
	// true 11
	// true
	// 89504e47
	// 11
	// [99 97 102 195 169]
}

func ExampleSV_Format() {
	fmt.Printf("%d|%05.2f|%x|%5s|%t\n",
		scalar.Scalar("42 apples"), scalar.Scalar("3.14159"), scalar.Scalar(255),
		scalar.Scalar("ab"), scalar.Scalar("0"))
	// Output:
	// 42|03.14|ff|   ab|false
}

func ExampleSV_Num() {
	fmt.Println(scalar.Scalar("3.5kg").Num()) // the numeric prefix, as Perl does
	fmt.Println(scalar.Scalar("abc").Num())
	// Output:
	// 3.5
	// 0
}

func ExampleSV_NumOK() {
	for _, x := range []any{"42", " 42 ", "42abc", "", nil} {
		n, ok := scalar.Scalar(x).NumOK()
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
	fmt.Println(scalar.Scalar(-3.9).Int()) // truncates toward zero
	fmt.Println(scalar.Scalar("12 items").Int())
	// Output:
	// -3
	// 12
}

func ExampleSV_IntOK() {
	i, ok := scalar.Scalar("9223372036854775807").IntOK() // exact, however large
	fmt.Println(i, ok)
	i, ok = scalar.Scalar(1e30).IntOK() // out of range: clamped
	fmt.Println(i, ok)
	// Output:
	// 9223372036854775807 true
	// 9223372036854775807 false
}

func ExampleSV_Bool() {
	for _, x := range []any{nil, "", "0", 0, 0.0, "0.0", "00", " ", "a"} {
		fmt.Print(scalar.Scalar(x).Bool(), " ")
	}
	fmt.Println()
	// Output:
	// false false false false false true true true true
}

func ExampleSV_AV() {
	var s scalar.SV   // undef
	s.AV().Push(1, 2) // push @$s, 1, 2: autovivifies an array
	fmt.Println(s.AV().Join(","))
	fmt.Println(scalar.Scalar("text").AV() == nil) // not an array
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
