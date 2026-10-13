package scalar_test

// Examples of the List::Util functions. They're run as tests, and they make
// up the Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

var isRemote = func(s *scalar.SV) bool { return s.M(`(?i)\bremote\b`) != nil }

func ExampleAV_AnyMatch() {
	lines := scalar.Array("Office in Leeds", "Remote working available")
	fmt.Println(lines.AnyMatch(isRemote)) // any { /remote/i } @lines
	fmt.Println(scalar.Array().AnyMatch(isRemote))
	// Output:
	// true
	// false
}

func ExampleAV_AllMatch() {
	n := scalar.Array(2, 4, 6)
	fmt.Println(n.AllMatch(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })) // all { $_ % 2 == 0 }
	fmt.Println(scalar.Array().AllMatch(isRemote))                                  // true for nothing
	// Output:
	// true
	// true
}

func ExampleAV_NoneMatch() {
	fmt.Println(scalar.Array("Leeds", "York").NoneMatch(isRemote))
	// Output:
	// true
}

func ExampleAV_FirstMatch() {
	lines := scalar.Array("Leeds office", "Salary: £45k", "Salary: £50k")
	salary := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("£") >= 0 })
	fmt.Println(salary) // first { index($_, "£") >= 0 } @lines
	fmt.Println(lines.FirstMatch(isRemote).Defined())
	// Output:
	// Salary: £45k
	// false
}

func ExampleAV_MaxNum() {
	fmt.Println(scalar.Array("10", 9, "2.5").MaxNum()) // max: as numbers
	// Output:
	// 10
}

func ExampleAV_MinNum() {
	fmt.Println(scalar.Array("10", 9, "2.5").MinNum())
	// Output:
	// 2.5
}

func ExampleAV_MaxStr() {
	fmt.Println(scalar.Array("10", "9", "100").MaxStr()) // maxstr: as strings
	// Output:
	// 9
}

func ExampleAV_MinStr() {
	fmt.Println(scalar.Array("b", "a", "c").MinStr())
	// Output:
	// a
}

func ExampleAV_UniqueStr() {
	fmt.Println(scalar.Array("Go", "go", "Go", "Perl").UniqueStr().Join(",")) // first of each, in order
	skills := scalar.Array("Go", "go", "Perl")
	fmt.Println(skills.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr().Join(",")) // ignoring case
	// Output:
	// Go,go,Perl
	// go,perl
}

func ExampleAV_UniqueNum() {
	fmt.Println(scalar.Array(1, "1.0", "01", 2).UniqueNum().Join(",")) // the same number, however written
	// Output:
	// 1,2
}

func ExampleAV_Sum() {
	fmt.Println(scalar.Array(1, "2", 3.5).Sum()) // sum @a
	fmt.Println(scalar.Array().Sum().Defined())  // undef for nothing
	// Output:
	// 6.5
	// false
}

func ExampleAV_Sum0() {
	fmt.Println(scalar.Array().Sum0()) // sum0: 0 for nothing
	// Output:
	// 0
}

func ExampleAV_Product() {
	fmt.Println(scalar.Array(2, 3, 4).Product())
	fmt.Println(scalar.Array().Product()) // 1 for nothing
	// Output:
	// 24
	// 1
}

func ExampleAV_Reduce() {
	words := scalar.Array("go", "perl", "c")
	longest := words.Reduce(func(a, b *scalar.SV) any { // reduce { length $b > length $a ? $b : $a } @words
		if b.Length() > a.Length() {
			return b
		}
		return a
	})
	joined := words.Reduce(func(a, b *scalar.SV) any { return a.Str() + "+" + b.Str() })
	fmt.Println(longest, joined)
	// Output:
	// perl go+perl+c
}
