package scalar_test

// Examples of hashes. Hash order is random, as in Perl, so the examples
// sort keys before printing them. They're run as tests, and they make up
// the Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleHash() {
	h := scalar.Hash("name", "Ada", "age", 36) // %h = (name => "Ada", age => 36)
	fmt.Println(h.Get("name"), h.Get("age"))
	merged := scalar.Hash(h, "age", 37) // %merged = (%h, age => 37): later keys win
	fmt.Println(merged.Get("age"))
	fmt.Println(scalar.Hash(map[string]int{"a": 1}).Get("a")) // from a Go map
	// Output:
	// Ada 36
	// 37
	// 1
}

func ExampleHV_Length() {
	fmt.Println(scalar.Hash("a", 1, "b", 2).Length()) // scalar(keys %h)
	// Output:
	// 2
}

func ExampleHV_Scalar() {
	fmt.Println(scalar.Hash("a", 1).Scalar()) // scalar(%h), the same as Length
	// Output:
	// 1
}

func ExampleHV_Get() {
	h := scalar.Hash("a", 1)
	fmt.Println(h.Get("a"))
	fmt.Println(h.Get("b").Defined(), h.Exists("b")) // reading doesn't create
	h.Get("c").HV().Set("d", 1)                      // $h{c}{d} = 1 autovivifies
	fmt.Println(h.Get("c").Ref())
	// Output:
	// 1
	// false false
	// HASH
}

func ExampleHV_Set() {
	h := scalar.Hash()
	h.Set("tags", []string{"go", "perl"}) // $h{tags} = ["go", "perl"]
	fmt.Println(h.Get("tags").AV().Join(","))
	// Output:
	// go,perl
}

func ExampleHV_Exists() {
	h := scalar.Hash("salary", nil)
	fmt.Println(h.Exists("salary"), h.Exists("bonus")) // there, though undef
	// Output:
	// true false
}

func ExampleHV_Defined() {
	h := scalar.Hash("salary", nil, "title", "Developer")
	fmt.Println(h.Defined("salary"), h.Defined("title"))
	// Output:
	// false true
}

func ExampleHV_Delete() {
	h := scalar.Hash("a", 1, "b", 2)
	fmt.Println(h.Delete("a"), h.Exists("a")) // returns the value it removed
	fmt.Println(h.Delete("missing").Defined())
	// Output:
	// 1 false
	// false
}

func ExampleHV_Slice() {
	h := scalar.Hash("a", 1, "b", 2)
	fmt.Println(h.Slice("b", "a").Join(","))                     // @h{qw(b a)}
	fmt.Println(h.Slice("missing").Get(0).Defined(), h.Length()) // undef, and not created
	// Output:
	// 2,1
	// false 2
}

func ExampleHV_KeyValueSlice() {
	h2 := scalar.Hash("k1", 1, "k2", 2, "k3", 3)
	h1 := scalar.Hash(h2.KeyValueSlice("k1", "k3")) // %h1 = %h2{qw(k1 k3)}
	fmt.Println(h1.Keys().Sort(nil).Join(","), h1.Get("k3"))
	fmt.Println(h2.KeyValueSlice("k2", "missing").Join(",")) // a missing key comes with undef
	// Output:
	// k1,k3 3
	// k2,2,missing,
}

func ExampleHV_Keys() {
	h := scalar.Hash("b", 2, "a", 1)
	fmt.Println(h.Keys().Sort(nil).Join(",")) // sort keys %h
	// Output:
	// a,b
}

func ExampleHV_Values() {
	h := scalar.Hash("a", 1, "b", 2)
	for _, v := range h.Values().All() { // $_ *= 10 for values %h
		v.MulInPlace(10)
	}
	fmt.Println(h.Get("a"), h.Get("b"))
	// Output:
	// 10 20
}

func ExampleHV_KeysSlice() {
	keys := scalar.Hash("a", 1).KeysSlice() // a Go []string
	fmt.Printf("%q\n", keys)
	// Output:
	// ["a"]
}

func ExampleHV_ValuesSlice() {
	values := scalar.Hash("a", "x").ValuesSlice() // a Go []*scalar.SV
	fmt.Println(len(values), values[0])
	// Output:
	// 1 x
}

func ExampleHV_All() {
	h := scalar.Hash("a", 1, "b", 2)
	sum := 0
	for _, v := range h.All() { // while (my ($k, $v) = each %h)
		sum += int(v.Int())
	}
	fmt.Println(sum)
	// Output:
	// 3
}
