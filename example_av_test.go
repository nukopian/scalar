package scalar_test

// Examples of arrays. They're run as tests, and they make up
// docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleArray() {
	a := scalar.Array("a", []string{"b", "c"}, scalar.Array("d")) // lists flatten, as in Perl
	fmt.Println(a.Length(), a.Join(","))
	nested := scalar.Array(1, scalar.Value(scalar.Array(2, 3))) // a reference stays one element
	fmt.Println(nested.Length(), nested.Get(1).AV().Join(","))
	// Output:
	// 4 a,b,c,d
	// 2 2,3
}

func ExampleAV_String() {
	a := scalar.Array("Go", 1, scalar.Value(scalar.Array(2, 3)))
	fmt.Println(a) // fmt uses String: the elements, on one line
	// Output:
	// ["Go",1,[2,3]]
}

func ExampleAV_Dump() {
	fmt.Println(scalar.Array("Go", nil, true).Dump())
	// Output:
	// [
	//   "Go",
	//   undef,
	//   !!1,
	// ]
}

func ExampleAV_Length() {
	fmt.Println(scalar.Array(1, 2, 3).Length()) // scalar(@a)
	// Output:
	// 3
}

func ExampleAV_Scalar() {
	fmt.Println(scalar.Array("x", "y").Scalar()) // scalar(@a), the same as Length
	// Output:
	// 2
}

func ExampleAV_LastIndex() {
	fmt.Println(scalar.Array(1, 2, 3).LastIndex()) // $#a
	fmt.Println(scalar.Array().LastIndex())
	// Output:
	// 2
	// -1
}

func ExampleAV_SetLastIndex() {
	a := scalar.Array(1, 2, 3, 4)
	a.SetLastIndex(1) // $#a = 1: truncate
	fmt.Println(a.Join(","))
	a.SetLastIndex(3) // extend with holes
	fmt.Println(a.Length(), a.Exists(3))
	// Output:
	// 1,2
	// 4 false
}

func ExampleAV_Get() {
	a := scalar.Array("first", "second", "last")
	fmt.Println(a.Get(0), a.Get(-1))             // $a[0], $a[-1]
	fmt.Println(a.Get(10).Defined(), a.Length()) // reading doesn't create
	a.Get(4).Set("new")                          // writing does, as $a[4] = "new"
	fmt.Println(a.Length())

	rows, _ := scalar.FromJSON([]byte(`[{"net": "ci", "stations": ["PAS", "GSC"]}]`))
	r := rows.AV()
	fmt.Println(r.Get(0, "stations", -1)) // $r[0]{stations}[-1]: a route
	// Output:
	// first last
	// false 3
	// 5
	// GSC
}

func ExampleAV_Set() {
	a := scalar.Array()
	a.Set(2, "c") // $a[2] = "c": extends the array, leaving holes
	fmt.Println(a.Length(), a.Exists(0), a.Get(2))
	// Output:
	// 3 false c
}

func ExampleAV_Exists() {
	a := scalar.Array("a", nil)
	fmt.Println(a.Exists(1)) // set, though undef
	fmt.Println(a.Exists(5))
	fmt.Println(a.Exists(3, "id"), a.Length()) // exists $a[3]{id}: creates $a[3], as in Perl
	// Output:
	// true
	// false
	// false 4
}

func ExampleAV_Defined() {
	a := scalar.Array("a", nil)
	fmt.Println(a.Defined(0), a.Defined(1), a.Defined(-2))
	// Output:
	// true false true
}

func ExampleAV_Delete() {
	a := scalar.Array("a", "b", "c")
	fmt.Println(a.Delete(1)) // delete $a[1]: leaves a hole
	fmt.Println(a.Length(), a.Exists(1))
	a.Delete(-1) // deleting the last element shrinks the array
	fmt.Println(a.Length())
	rows, _ := scalar.FromJSON([]byte(`[{"id": "ci40123456", "tmp": 1}]`))
	fmt.Println(rows.AV().Delete(0, "tmp"), rows.At("[0]").HV().KeysSlice()) // delete $rows[0]{tmp}
	// Output:
	// b
	// 3 false
	// 1
	// 1 [id]
}

func ExampleAV_Push() {
	a := scalar.Array("a")
	a.Push("b", []string{"c", "d"})          // push @a, "b", @list
	a.Push(scalar.Value(scalar.Array(1, 2))) // push @a, [1, 2]: a reference, one element
	fmt.Println(a.Length(), a.Get(-1).Ref())
	// Output:
	// 5 ARRAY
}

func ExampleAV_Unshift() {
	a := scalar.Array("c")
	a.Unshift("a", "b") // unshift @a, "a", "b": order kept
	fmt.Println(a.Join(","))
	// Output:
	// a,b,c
}

func ExampleAV_Pop() {
	a := scalar.Array(1, 2, 3)
	fmt.Println(a.Pop(), a.Join(","))
	fmt.Println(scalar.Array().Pop().Defined()) // nothing to pop: undef
	// Output:
	// 3 1,2
	// false
}

func ExampleAV_Shift() {
	a := scalar.Array(1, 2, 3)
	fmt.Println(a.Shift(), a.Join(","))
	// Output:
	// 1 2,3
}

func ExampleAV_PopN() {
	a := scalar.Array("a", "b", "c", "d")
	last := a.PopN(2) // splice(@a, -2): in array order
	fmt.Println(last.Join(","), "|", a.Join(","))
	fmt.Println(a.PopN(10).Join(","), "|", a.Length()) // more than there is: all of it
	// Output:
	// c,d | a,b
	// a,b | 0
}

func ExampleAV_ShiftN() {
	queue := scalar.Array("PAS", "GSC", "ISA", "SVD", "BBR")
	for queue.Length() > 0 {
		batch := queue.ShiftN(2) // splice(@queue, 0, 2): the next batch
		fmt.Println(batch.Join(","))
	}
	// Output:
	// PAS,GSC
	// ISA,SVD
	// BBR
}

func ExampleAV_Splice() {
	a := scalar.Array("a", "b", "c", "d", "e")
	removed := a.Splice(1, 2, "X") // splice @a, 1, 2, "X"
	fmt.Println(removed.Join(","), "|", a.Join(","))
	a.Splice(-1, 1) // remove the last element
	fmt.Println(a.Join(","))
	// Output:
	// b,c | a,X,d,e
	// a,X,d
}

func ExampleAV_Values() {
	a := scalar.Array("a", "b", "c", "d")
	fmt.Println(a.Values().Join(","))         // values @a
	fmt.Println(a.Values(0, -1, 1).Join(",")) // @a[0, -1, 1]
	picks := scalar.Array(2, 3)
	fmt.Println(a.Values(picks).Join(","))    // @a[@picks]
	for _, s := range a.Values(0, -1).All() { // $_ = uc for @a[0, -1]: the elements themselves
		s.Set(s.Uc())
	}
	fmt.Println(a.Join(","))
	// Output:
	// a,b,c,d
	// a,d,b
	// c,d
	// A,b,c,D
}

func ExampleAV_Pairs() {
	a := scalar.Array("a", "b")
	fmt.Println(a.Pairs(0, -1).Join(",")) // %a[0, -1]: indices as written
	fmt.Println(a.Pairs().Join(","))      // every index and element
	// Output:
	// 0,a,-1,b
	// 0,a,1,b
}

func ExampleAV_Items() {
	for _, s := range scalar.Array("a", "b").Items() { // a Go []*scalar.SV
		fmt.Print(s, " ")
	}
	fmt.Println()
	// Output:
	// a b
}

func ExampleAV_StrSlice() {
	fmt.Printf("%q\n", scalar.Array("a", 1, nil).StrSlice())
	// Output:
	// ["a" "1" ""]
}

func ExampleAV_IntSlice() {
	fmt.Println(scalar.Array("3", 4.9, "x").IntSlice())
	// Output:
	// [3 4 0]
}

func ExampleAV_NumSlice() {
	fmt.Println(scalar.Array("1.5", 2).NumSlice())
	// Output:
	// [1.5 2]
}

func ExampleAV_Each() {
	a := scalar.Array("x", "y")
	for i, v, ok := a.Each(); ok; i, v, ok = a.Each() { // while (my ($i, $v) = each @a)
		fmt.Println(i, v)
	}
	// Output:
	// 0 x
	// 1 y
}

func ExampleAV_All() {
	for i, s := range scalar.Array("a", "b").All() { // while (my ($i, $s) = each @a)
		fmt.Println(i, s)
	}
	// Output:
	// 0 a
	// 1 b
}

func ExampleAV_Join() {
	fmt.Println(scalar.Array("Go", "Perl", 5).Join(", "))
	// Output:
	// Go, Perl, 5
}

func ExampleAV_Map() {
	a := scalar.Array("go", "perl")
	fmt.Println(a.Map(func(s *scalar.SV) any { return s.Uc() }).Join(","))
	pairs := a.Map(func(s *scalar.SV) any { return scalar.Array(s, s.Length()) }) // several values each
	fmt.Println(pairs.Join(","))
	// Output:
	// GO,PERL
	// go,2,perl,4
}

func ExampleAV_Grep() {
	n := scalar.Array(1, 2, 3, 4).Grep(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })
	fmt.Println(n.Join(","))
	// Output:
	// 2,4
}

func ExampleAV_Sort() {
	a := scalar.Array("10", "9", "2")
	fmt.Println(a.Sort(nil).Join(","))           // sort @a: as strings
	fmt.Println(a.Sort(scalar.NumCmp).Join(",")) // sort { $a <=> $b } @a
	byLength := func(x, y *scalar.SV) int { return x.Length() - y.Length() }
	fmt.Println(scalar.Array("ccc", "a", "bb").Sort(byLength).Join(","))
	fmt.Println(a.Join(",")) // unchanged
	// Output:
	// 10,2,9
	// 2,9,10
	// a,bb,ccc
	// 10,9,2
}

func ExampleAV_SortInPlace() {
	a := scalar.Array(3, 1, 2)
	a.SortInPlace(scalar.NumCmp) // @a = sort { $a <=> $b } @a
	fmt.Println(a.Join(","))
	// Output:
	// 1,2,3
}

func ExampleAV_Reverse() {
	a := scalar.Array(1, 2, 3)
	fmt.Println(a.Reverse().Join(","), a.Join(","))
	// Output:
	// 3,2,1 1,2,3
}

func ExampleAV_ReverseInPlace() {
	a := scalar.Array(1, 2, 3)
	a.ReverseInPlace()
	fmt.Println(a.Join(","))
	// Output:
	// 3,2,1
}

func ExampleAV_SplitAt() {
	head, tail := scalar.Array("Ada", "Lovelace", "maths", "poetry").SplitAt(2) // my ($first, $last, @interests) = @a
	fmt.Println(head.Join(" "), "|", tail.Join(","))
	// Output:
	// Ada Lovelace | maths,poetry
}

func ExampleAV_SplitAtSlice() {
	h, rest := scalar.Array("2026", "10", "13", "extra").SplitAtSlice(3)
	year, month, day := h[0], h[1], h[2] // my ($year, $month, $day, @rest) = @a
	fmt.Println(year, month, day, rest.Length())
	// Output:
	// 2026 10 13 1
}

func ExampleAV_Clone() {
	rows := scalar.Array(scalar.Value(map[string]any{"place": "Cobb, CA"})) // one row: a hash reference
	mine := rows.Clone()                                                    // dclone(\@rows): the rows too, not just the list
	mine.Get(0).At("place").Set("Anza, CA")
	fmt.Println(rows.Get(0).At("place"), "/", mine.Get(0).At("place"))
	// Output:
	// Cobb, CA / Anza, CA
}

func ExampleAV_Equal() {
	a := scalar.Array("PAS", 1)
	fmt.Println(a.Equal([]any{"PAS", "1"}), a.Equal([]any{"PAS"}))
	// Output:
	// true false
}

func ExampleAV_Diff() {
	fmt.Println(scalar.Array("PAS", "GSC").Diff([]string{"PAS"}))
	// Output:
	// [1]: got "GSC", want does not exist
}

func ExampleAV_Apply() {
	prices := scalar.Array(10, scalar.Value([]any{20, 30}))
	prices.Apply(func(v *scalar.SV) any { return v.Mul(1.2) }) // every price, nested ones too
	fmt.Println(prices.Get(0), prices.Get(1).AV().Join(" "))
	// Output:
	// 12 24 36
}

func ExampleAV_At() {
	rows, _ := scalar.FromJSON([]byte(`[{"name":"Ada"},{"name":"Grace"}]`))
	a := rows.AV()
	fmt.Println(a.At("[1].name")) // $a[1]{name}
	a.At("[2].name").Set("Alan")  // $a[2]{name} = "Alan": creates the row
	fmt.Println(a.Length(), a.At("[-1].name"))
	// Output:
	// Grace
	// 3 Alan
}

func ExampleAV_Find() {
	quakes, _ := scalar.FromJSON([]byte(`[
		{"place": "12 km SSW of Ridgecrest, CA", "mag": 4.6},
		{"place": "5 km N of Parkfield, CA", "mag": 2.1},
		{"place": "8 km E of Anza, CA", "mag": 3.8}
	]`))
	big := quakes.AV().Find("$[?@.mag >= 3].place") // $ is the array itself
	fmt.Println(big.Join("; "))
	// Output:
	// 12 km SSW of Ridgecrest, CA; 8 km E of Anza, CA
}

func ExampleAV_AtOK() {
	a := scalar.Array(1, nil)
	_, ok := a.AtOK("[1]") // exists, though undef
	_, past := a.AtOK("[2]")
	fmt.Println(ok, past)
	// Output:
	// true false
}

func ExampleAV_DeleteAt() {
	a := scalar.Array("a", "b", "c")
	fmt.Println(a.DeleteAt("[-1]"), a.Length()) // delete $a[-1]: the last goes, so it's shorter
	fmt.Println(a.DeleteAt("[0]"), a.Length())  // delete $a[0]: a hole, so it isn't
	// Output:
	// c 2
	// a 2
}

func ExampleAV_GetOK() {
	a := scalar.Array(scalar.Value(scalar.Hash("id", 7)))
	v, ok := a.GetOK(0, "id")
	_, missing := a.GetOK(0, "name")
	fmt.Println(v, ok, missing)
	// Output:
	// 7 true false
}

func ExampleAV_DeleteOK() {
	rows, _ := scalar.FromJSON([]byte(`[{"id": "ci40123456", "tmp": 1}, {"id": "nc75012345"}, "a stray string"]`))
	a := rows.AV()
	for i := range a.Length() {
		_, ok := a.DeleteOK(i, "tmp") // delete $a[$i]{tmp}, if it's there
		fmt.Println(i, ok)
	}
	// Output:
	// 0 true
	// 1 false
	// 2 false
}

func ExampleAV_Undef() {
	a := scalar.Array("x", "y")
	doc := scalar.Hash("tags", scalar.Value(a)) // another reference to a
	a.Undef()                                   // undef @a
	fmt.Println(a.Length(), doc.At("tags").AV().Length())
	// Output:
	// 0 0
}

// A locked array can't grow, shrink or change, nor can its elements.
func ExampleAV_Lock() {
	a := scalar.Array(1, 2, 3).Lock()
	fmt.Println(a.Get(0), a.Length())
	fmt.Println(catch(func() { a.Push(4) }))
	fmt.Println(catch(func() { a.Get(0).Set(9) }))
	// Output:
	// 1 3
	// scalar: Modification of a read-only value attempted
	// scalar: Modification of a read-only value attempted
}

func ExampleAV_Unlock() {
	a := scalar.Array(1, 2).Lock().Unlock()
	fmt.Println(a.Push(3).Length())
	// Output:
	// 3
}

func ExampleAV_Locked() {
	a := scalar.Array(1, 2)
	fmt.Println(a.Locked(), a.Lock().Locked())
	// Output:
	// false true
}

// With autovivification off, an array's missing elements, and what would
// be below them, read as undef and create nothing; writing still creates.
func ExampleAV_Autovivify() {
	quakes := scalar.Array(scalar.Value(map[string]any{"mag": 4.6})).Autovivify(false)
	fmt.Println(quakes.Get(0, "felt", "count").Defined(), quakes.Get(3, "mag").Defined(), quakes.Length())
	quakes.Get(1, "mag").Set(2.1)
	fmt.Println(quakes.Length(), quakes.Get(1, "mag"))
	// Output:
	// false false 1
	// 2 2.1
}

func ExampleAV_Autovivifies() {
	quakes := scalar.Array(4.6, 2.1)
	fmt.Println(quakes.Autovivifies(), quakes.Autovivify(false).Autovivifies())
	// Output:
	// true false
}

func ExampleAV_AutovivifiesDeeply() {
	quakes := scalar.Array(4.6, 2.1).Autovivify(false, true)
	fmt.Println(quakes.Autovivifies(), quakes.AutovivifiesDeeply())
	// Output:
	// false true
}
