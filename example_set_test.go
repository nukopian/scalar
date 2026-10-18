package scalar_test

// Examples of sets and bags, after CPAN's Set::Scalar and Set::Bag. They're
// run as tests, and they make up docs/EXAMPLES.md (see TestReference).

import (
	"encoding/json"
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleNewSet() {
	s := scalar.NewSet("Go", "Perl", "Go") // Set::Scalar->new(...): each member once
	fmt.Println(s, s.Length())
	same := scalar.NewSet(1, "1", 1.0) // compared as strings: all "1"
	fmt.Println(same.Length())
	fromList := scalar.NewSet(scalar.Array("a", "b"), []string{"c"}) // lists flatten
	fmt.Println(fromList)
	// Output:
	// (Go Perl) 2
	// 1
	// (a b c)
}

func ExampleSet_Insert() {
	var s scalar.Set // the zero value is an empty set
	s.Insert("Go").Insert("Perl", "Go")
	fmt.Println(&s)
	// Output:
	// (Go Perl)
}

func ExampleSet_Delete() {
	s := scalar.NewSet("a", "b", "c")
	s.Delete("b", "not a member") // non-members are ignored
	fmt.Println(s)
	// Output:
	// (a c)
}

func ExampleSet_Has() {
	s := scalar.NewSet("Go", 42)
	fmt.Println(s.Has("Go"), s.Has("go"), s.Has("42")) // "42" is the same member as 42
	// Output:
	// true false true
}

func ExampleSet_Length() {
	fmt.Println(scalar.NewSet("a", "b", "a").Length())
	// Output:
	// 2
}

func ExampleSet_Members() {
	s := scalar.NewSet("c", "a", "b")
	fmt.Println(s.Members().Join(","))           // in insertion order
	fmt.Println(s.Members().Sort(nil).Join(",")) // sorted, with the array tools
	// Output:
	// c,a,b
	// a,b,c
}

func ExampleSet_All() {
	for m := range scalar.NewSet("x", "y").All() {
		fmt.Println(m)
	}
	// Output:
	// x
	// y
}

func ExampleSet_Copy() {
	a := scalar.NewSet("a")
	b := a.Copy()
	b.Insert("b")
	fmt.Println(a, b)
	// Output:
	// (a) (a b)
}

func ExampleSet_Union() {
	a, b := scalar.NewSet("a", "b"), scalar.NewSet("b", "c")
	fmt.Println(a.Union(b))                     // $a->union($b)
	fmt.Println(a.Union(b, scalar.NewSet("d"))) // several at once
	// Output:
	// (a b c)
	// (a b c d)
}

func ExampleSet_Intersection() {
	recorded := scalar.NewSet("PAS", "GSC", "ISA")
	expected := scalar.NewSet("PAS", "ISA", "SVD")
	fmt.Println(recorded.Intersection(expected)) // the expected stations that recorded the quake
	// Output:
	// (ISA PAS)
}

func ExampleSet_Difference() {
	expected := scalar.NewSet("PAS", "ISA", "SVD")
	recorded := scalar.NewSet("PAS", "GSC", "ISA")
	fmt.Println(expected.Difference(recorded)) // expected, but silent
	// Output:
	// (SVD)
}

func ExampleSet_SymmetricDifference() {
	a, b := scalar.NewSet("a", "b", "c"), scalar.NewSet("b", "c", "d")
	fmt.Println(a.SymmetricDifference(b)) // in one or the other, not both
	// Output:
	// (a d)
}

func ExampleSet_IsSubset() {
	a := scalar.NewSet("Go")
	fmt.Println(a.IsSubset(scalar.NewSet("Go", "Perl")), a.IsSubset(a), scalar.NewSet().IsSubset(a))
	// Output:
	// true true true
}

func ExampleSet_IsProperSubset() {
	a := scalar.NewSet("Go")
	fmt.Println(a.IsProperSubset(scalar.NewSet("Go", "Perl")), a.IsProperSubset(a))
	// Output:
	// true false
}

func ExampleSet_IsSuperset() {
	fmt.Println(scalar.NewSet("Go", "Perl").IsSuperset(scalar.NewSet("Perl")))
	// Output:
	// true
}

func ExampleSet_IsEqual() {
	fmt.Println(scalar.NewSet("a", "b").IsEqual(scalar.NewSet("b", "a"))) // order doesn't matter
	// Output:
	// true
}

func ExampleSet_IsDisjoint() {
	fmt.Println(scalar.NewSet("a").IsDisjoint(scalar.NewSet("b")))
	fmt.Println(scalar.NewSet().IsDisjoint(scalar.NewSet())) // true, unlike Set::Scalar
	// Output:
	// true
	// true
}

func ExampleSet_String() {
	fmt.Println(scalar.NewSet("c", "a", "b").String()) // sorted, as Set::Scalar prints a set
	// Output:
	// (a b c)
}

func ExampleSet_MarshalJSON() {
	out, _ := json.Marshal(scalar.NewSet("c", "a", "b")) // an array, in insertion order
	fmt.Println(string(out))
	// Output:
	// ["c","a","b"]
}

func ExampleNewBag() {
	networks := scalar.NewBag("ci", "us", "ci") // each value counts once
	fmt.Println(networks)
	apples := scalar.NewBag().InsertN("apples", 3) // Set::Bag->new(apples => 3)
	fmt.Println(apples)
	// Output:
	// (ci => 2, us => 1)
	// (apples => 3)
}

func ExampleBag_Insert() {
	var counts scalar.Bag // the zero value is an empty bag
	for _, day := range [][]string{{"ci", "us"}, {"ci", "nc"}, {"ci"}} {
		counts.Insert(day) // count the networks across days
	}
	fmt.Println(&counts)
	// Output:
	// (ci => 3, nc => 1, us => 1)
}

func ExampleBag_InsertN() {
	b := scalar.NewBag()
	b.InsertN("apples", 3).InsertN("apples", 2) // $b->insert(apples => 3) and so on
	fmt.Println(b)
	// Output:
	// (apples => 5)
}

func ExampleBag_Delete() {
	b := scalar.NewBag("a", "a", "b")
	b.Delete("a", "b") // one of each
	fmt.Println(b)
	// Output:
	// (a => 1)
}

func ExampleBag_DeleteN() {
	b := scalar.NewBag().InsertN("apples", 3)
	b.DeleteN("apples", 2)
	fmt.Println(b)
	b.DeleteN("apples", 10) // more than there is: all of it, never below zero
	fmt.Println(b)
	// Output:
	// (apples => 1)
	// ()
}

func ExampleBag_Count() {
	b := scalar.NewBag("Go", "Go", "Perl")
	fmt.Println(b.Count("Go"), b.Count("Rust")) // $b->grab("Go")
	// Output:
	// 2 0
}

func ExampleBag_Length() {
	fmt.Println(scalar.NewBag("a", "a", "b").Length()) // distinct members
	// Output:
	// 2
}

func ExampleBag_Total() {
	fmt.Println(scalar.NewBag("a", "a", "b").Total()) // counting repeats
	// Output:
	// 3
}

func ExampleBag_Members() {
	fmt.Println(scalar.NewBag("b", "a", "b").Members().Join(",")) // distinct, in insertion order
	// Output:
	// b,a
}

func ExampleBag_Counts() {
	counts := scalar.NewBag("Go", "Go", "Perl").Counts() // a hash, for the hash tools
	fmt.Println(counts.Get("Go"), counts.Get("Perl"))
	// Output:
	// 2 1
}

func ExampleBag_MostCommon() {
	networks := scalar.NewBag("us", "ci", "nc", "ci", "us", "ci")
	fmt.Println(networks.MostCommon(2).Join(",")) // the top 2
	fmt.Println(networks.MostCommon().Join(","))  // all, most common first
	// Output:
	// ci,us
	// ci,us,nc
}

func ExampleBag_All() {
	for m, n := range scalar.NewBag("x", "y", "x").All() {
		fmt.Println(m, n)
	}
	// Output:
	// x 2
	// y 1
}

func ExampleBag_Copy() {
	a := scalar.NewBag("a")
	b := a.Copy()
	b.Insert("a")
	fmt.Println(a, b)
	// Output:
	// (a => 1) (a => 2)
}

func ExampleBag_Sum() {
	monday := scalar.NewBag().InsertN("ci", 3).InsertN("us", 1)
	tuesday := scalar.NewBag().InsertN("ci", 1).InsertN("nc", 2)
	fmt.Println(monday.Sum(tuesday)) // counts added: $monday->sum($tuesday)
	// Output:
	// (ci => 4, nc => 2, us => 1)
}

func ExampleBag_Union() {
	x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
	y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
	fmt.Println(x.Union(y)) // the larger count of each
	// Output:
	// (a => 3, b => 2)
}

func ExampleBag_Intersection() {
	x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
	y := scalar.NewBag().InsertN("a", 1).InsertN("c", 2)
	fmt.Println(x.Intersection(y)) // the smaller count of each, in both
	// Output:
	// (a => 1)
}

func ExampleBag_Difference() {
	x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
	y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
	fmt.Println(x.Difference(y)) // counts taken away, never below zero
	// Output:
	// (a => 2)
}

func ExampleBag_String() {
	fmt.Println(scalar.NewBag("pears", "apples", "apples").String()) // sorted, as Set::Bag prints a bag
	// Output:
	// (apples => 2, pears => 1)
}

func ExampleBag_MarshalJSON() {
	out, _ := json.Marshal(scalar.NewBag("pears", "apples", "apples")) // in insertion order
	fmt.Println(string(out))
	// Output:
	// {"pears":1,"apples":2}
}
