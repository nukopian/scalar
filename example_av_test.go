package scalar_test

// Examples of arrays. They're run as tests, and they make up the Reference
// section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
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
	// Output:
	// first last
	// false 3
	// 5
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
	// Output:
	// true
	// false
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
	// Output:
	// b
	// 3 false
	// 1
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
	queue := scalar.Array("job1", "job2", "job3", "job4", "job5")
	for queue.Length() > 0 {
		batch := queue.ShiftN(2) // splice(@queue, 0, 2): the next batch
		fmt.Println(batch.Join(","))
	}
	// Output:
	// job1,job2
	// job3,job4
	// job5
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
	rows := scalar.Array(scalar.Value(map[string]any{"title": "Go Developer"})) // one row: a hash reference
	mine := rows.Clone()                                                        // dclone(\@rows): the rows too, not just the list
	mine.Get(0).At("title").Set("Perl Developer")
	fmt.Println(rows.Get(0).At("title"), "/", mine.Get(0).At("title"))
	// Output:
	// Go Developer / Perl Developer
}

func ExampleAV_Equal() {
	a := scalar.Array("Go", 1)
	fmt.Println(a.Equal([]any{"Go", "1"}), a.Equal([]any{"Go"}))
	// Output:
	// true false
}

func ExampleAV_Diff() {
	fmt.Println(scalar.Array("Go", "SQL").Diff([]string{"Go"}))
	// Output:
	// [1]: got "SQL", want does not exist
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

func ExampleAV_Path() {
	a := scalar.Array()
	a.Path(0, "tags", 1).Set("go") // $a[0]{tags}[1] = "go"
	fmt.Println(a.Path(0, "tags", 1), a.At("[0].tags").AV().Length())
	// Output:
	// go 2
}

func ExampleAV_PathOK() {
	a := scalar.Array(scalar.Value(scalar.Hash("id", 7)))
	v, ok := a.PathOK(0, "id")
	_, missing := a.PathOK(0, "name")
	fmt.Println(v, ok, missing)
	// Output:
	// 7 true false
}

func ExampleAV_DeletePath() {
	rows, _ := scalar.FromJSON([]byte(`[{"id":1,"tmp":"x"},{"id":2,"tmp":"y"}]`))
	a := rows.AV()
	for i := range a.Length() {
		a.DeletePath(i, "tmp") // delete $a[$i]{tmp}
	}
	fmt.Println(a.At("[0]").HV().KeysSlice(), a.At("[1]").HV().KeysSlice())
	// Output:
	// [id] [id]
}
