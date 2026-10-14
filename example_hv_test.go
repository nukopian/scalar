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

func ExampleHV_String() {
	h := scalar.Hash("title", "Go Developer", "log level", "debug")
	fmt.Println(h) // keys sorted, and quoted only when they need to be
	// Output:
	// {"log level" => "debug",title => "Go Developer"}
}

func ExampleHV_Dump() {
	tags := scalar.Scalar(scalar.Array("go"))
	h := scalar.Hash("tags", tags, "again", tags) // the same array twice
	fmt.Println(h.Dump())
	// Output:
	// {
	//   again => [
	//     "go",
	//   ],
	//   tags => $VAR1->{again},
	// }
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

func ExampleHV_Push() {
	h := scalar.Hash("title", "Developer")
	h.Push("location", "Leeds", "salary", 45000) // $h{location} = "Leeds"; $h{salary} = 45000
	defaults := scalar.Hash("remote", false, "salary", 0)
	h.Push(defaults, "salary", 50000) // merge, later keys winning
	fmt.Println(h.Keys().Sort(nil).Join(","))
	fmt.Println(h.Get("salary"))
	// Output:
	// location,remote,salary,title
	// 50000
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

func ExampleHV_Clone() {
	defaults := scalar.Hash("remote", false, "tags", scalar.Scalar([]string{"go"}))
	row := defaults.Clone() // dclone(\%defaults): nested arrays too
	row.Get("tags").AV().Push("perl")
	merged := scalar.Hash(defaults) // %row = %defaults: a new hash, but the same tags array
	merged.Get("tags").AV().Push("sql")
	fmt.Println(defaults.Get("tags").AV().Join(","), "/", row.Get("tags").AV().Join(","))
	// Output:
	// go,sql / go,perl
}

func ExampleHV_Equal() {
	h := scalar.Hash("remote", true, "days", 2)
	fmt.Println(h.Equal(map[string]any{"remote": 1, "days": "2"})) // true is 1, as in Perl
	// Output:
	// true
}

func ExampleHV_Diff() {
	h := scalar.Hash("bonus", nil)
	fmt.Println(h.Diff(map[string]any{})) // a key whose value is undef isn't a missing key
	// Output:
	// bonus: got undef, want does not exist
}

func ExampleHV_Apply() {
	row := scalar.Hash("title", "go developer", "remote", true)
	row.Apply(func(v *scalar.SV) any {
		if v.Kind() == scalar.String {
			return v.UcFirst()
		}
		return v
	})
	fmt.Println(row.Get("title"), row.Get("remote"))
	// Output:
	// Go developer 1
}

func ExampleHV_Pairs() {
	h2 := scalar.Hash("k1", 1, "k2", 2, "k3", 3)
	h1 := scalar.Hash(h2.Pairs("k1", "k3")) // %h1 = %h2{qw(k1 k3)}
	fmt.Println(h1.Keys().Sort(nil).Join(","), h1.Get("k3"))
	fmt.Println(h2.Pairs("k2", "missing").Join(","))      // a missing key comes with undef
	fmt.Println(scalar.Hash("only", 1).Pairs().Join(",")) // every pair: %h in list context
	// Output:
	// k1,k3 3
	// k2,2,missing,
	// only,1
}

func ExampleHV_PairsMap() {
	h := scalar.Hash("a", 1)
	m := h.PairsMap() // a Go map[string]*scalar.SV of the hash's own values
	m["a"].Set(2)
	fmt.Println(len(m), h.Get("a"))
	// Output:
	// 1 2
}

func ExampleHV_Keys() {
	h := scalar.Hash("b", 2, "a", 1)
	fmt.Println(h.Keys().Sort(nil).Join(",")) // sort keys %h
	// Output:
	// a,b
}

func ExampleHV_Values() {
	h := scalar.Hash("a", 1, "b", 2, "c", 3)
	fmt.Println(h.Values("c", "a").Join(","))                     // @h{qw(c a)}
	fmt.Println(h.Values("missing").Get(0).Defined(), h.Length()) // undef, and not created
	for _, v := range h.Values().All() {                          // $_ *= 10 for values %h
		v.MulInPlace(10)
	}
	fmt.Println(h.Values("a", "b", "c").Join(","))
	// Output:
	// 3,1
	// false 3
	// 10,20,30
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

func ExampleHV_Each() {
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
	// Output:
	// 6
	// true
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
