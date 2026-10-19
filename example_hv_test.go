package scalar_test

// Examples of hashes. Hash order is random, as in Perl, so the examples sort
// keys before printing them. They're run as tests, and they make up
// docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
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
	h := scalar.Hash("place", "Cobb, CA", "log level", "debug")
	fmt.Println(h) // keys sorted, and quoted only when they need to be
	// Output:
	// {"log level" => "debug",place => "Cobb, CA"}
}

func ExampleHV_Dump() {
	tags := scalar.Value(scalar.Array("go"))
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
	h := scalar.Hash("id", "ci40123456")
	h.Push("place", "Ridgecrest, CA", "felt", 1204) // $h{place} = "Ridgecrest, CA"; $h{felt} = 1204
	defaults := scalar.Hash("tsunami", false, "felt", 0)
	h.Push(defaults, "felt", 1500) // merge, later keys winning
	fmt.Println(h.Keys().Sort(nil).Join(","))
	fmt.Println(h.Get("felt"))
	// Output:
	// felt,id,place,tsunami
	// 1500
}

func ExampleHV_Exists() {
	h := scalar.Hash("felt", nil)
	fmt.Println(h.Exists("felt"), h.Exists("cdi")) // there, though undef
	// Output:
	// true false
}

func ExampleHV_Defined() {
	h := scalar.Hash("felt", nil, "place", "Cobb, CA")
	fmt.Println(h.Defined("felt"), h.Defined("place"))
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
	defaults := scalar.Hash("tsunami", false, "stations", scalar.Value([]string{"PAS"}))
	row := defaults.Clone() // dclone(\%defaults): nested arrays too
	row.Get("stations").AV().Push("GSC")
	merged := scalar.Hash(defaults) // %row = %defaults: a new hash, but the same stations array
	merged.Get("stations").AV().Push("ISA")
	fmt.Println(defaults.Get("stations").AV().Join(","), "/", row.Get("stations").AV().Join(","))
	// Output:
	// PAS,ISA / PAS,GSC
}

func ExampleHV_Equal() {
	h := scalar.Hash("tsunami", true, "felt", 2)
	fmt.Println(h.Equal(map[string]any{"tsunami": 1, "felt": "2"})) // true is 1, as in Perl
	// Output:
	// true
}

func ExampleHV_Diff() {
	h := scalar.Hash("felt", nil)
	fmt.Println(h.Diff(map[string]any{})) // a key whose value is undef isn't a missing key
	// Output:
	// felt: got undef, want does not exist
}

func ExampleHV_Apply() {
	row := scalar.Hash("place", "ridgecrest, CA", "tsunami", true)
	row.Apply(func(v *scalar.SV) any {
		if v.Kind() == scalar.String {
			return v.UcFirst()
		}
		return v
	})
	fmt.Println(row.Get("place"), row.Get("tsunami"))
	// Output:
	// Ridgecrest, CA 1
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

func ExampleHV_At() {
	tallies := scalar.Hash()
	tallies.At("ci.reviewed").Inc() // $tallies{ci}{reviewed}++
	tallies.At("ci.reviewed").Inc()
	tallies.At("ci.automatic").Inc()
	fmt.Println(tallies.At("ci.reviewed"), tallies.At("ci.automatic"), tallies.At("us.reviewed").Int())
	// Output:
	// 2 1 0
}

func ExampleHV_Find() {
	doc, _ := scalar.FromJSON([]byte(`{"ci": ["PAS", "GSC"], "nc": ["MCB"]}`))
	networks := doc.HV()
	fmt.Println(networks.Find("$.*[*]").Length()) // every station, in any network
	fmt.Println(networks.Find("$.ci[-1]").Get(0)) // $ is the hash itself
	// Output:
	// 3
	// GSC
}

func ExampleHV_AtOK() {
	h := scalar.Hash("depth", scalar.Value(scalar.Hash("error", nil)))
	_, ok := h.AtOK("depth.error") // exists, though undef
	_, missing := h.AtOK("depth.km")
	fmt.Println(ok, missing)
	// Output:
	// true false
}

func ExampleHV_DeleteAt() {
	h := scalar.Hash("depth", scalar.Value(scalar.Hash("km", 8, "error", 2)))
	fmt.Println(h.DeleteAt("depth.error")) // delete $h{depth}{error}
	fmt.Println(h.At("depth").HV().KeysSlice())
	// Output:
	// 2
	// [km]
}

func ExampleHV_Path() {
	h := scalar.Hash()
	h.Path("a.b", "c").Set(1) // $h{"a.b"}{c} = 1: Path keys are taken whole, dots and all
	fmt.Println(h.Path("a.b", "c"), h.At(`["a.b"].c`))
	// Output:
	// 1 1
}

func ExampleHV_PathOK() {
	h := scalar.Hash("tags", scalar.Value(scalar.Array("go")))
	v, ok := h.PathOK("tags", 0)
	_, past := h.PathOK("tags", 5)
	fmt.Println(v, ok, past)
	// Output:
	// go true false
}

func ExampleHV_DeletePath() {
	section, key := "depth", "error" // from data
	h := scalar.Hash("depth", scalar.Value(scalar.Hash("km", 8, "error", 2)))
	fmt.Println(h.DeletePath(section, key)) // delete $h{$section}{$key}
	// Output:
	// 2
}

func ExampleHV_Undef() {
	seen := scalar.Hash("go", 1, "perl", 1)
	seen.Undef() // undef %seen: emptied, ready for the next batch
	seen.Set("raku", 1)
	fmt.Println(seen.KeysSlice())
	// Output:
	// [raku]
}

// Map builds a new hash, renaming keys or changing values or both. The old
// hash is unchanged.
func ExampleHV_Map() {
	depth := scalar.Hash("min", 2000, "max", 12500) // metres
	km := depth.Map(func(key string, v *scalar.SV) (string, any) {
		return "depth_" + key, v.Div(1000).Str() + " km"
	})
	fmt.Println(km.Get("depth_min"), "-", km.Get("depth_max"), depth.Get("min"))
	// Output:
	// 2 km - 12.5 km 2000
}

// A locked hash keeps the keys it has: reading, writing or deleting one it
// hasn't got panics, as in a Perl restricted hash; Exists still answers.
func ExampleHV_Lock() {
	h := scalar.Hash("min", 25000).Lock()
	fmt.Println(h.Get("min"), h.Exists("max"))
	fmt.Println(catch(func() { h.Get("max") }))
	fmt.Println(catch(func() { h.Delete("min") }))
	// Output:
	// 25000 false
	// scalar: Attempt to access disallowed key 'max' in a restricted hash
	// scalar: Attempt to delete readonly key 'min' from a restricted hash
}

func ExampleHV_Unlock() {
	h := scalar.Hash("min", 25000).Lock().Unlock()
	h.Set("max", 30000)
	fmt.Println(h.Get("max"))
	// Output:
	// 30000
}

func ExampleHV_Locked() {
	h := scalar.Hash("min", 25000)
	fmt.Println(h.Locked(), h.Lock().Locked())
	// Output:
	// false true
}
