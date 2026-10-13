package scalar_test

// Examples of Perl's operators. They're run as tests, and they make up the
// Reference section of README.md (see TestReference).

import (
	"fmt"
	"math"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleSV_Add() {
	fmt.Println(scalar.Scalar("40").Add(2))          // "40" + 2
	fmt.Println(scalar.Scalar(1.5).Add("1.5"))       // floats when needed
	fmt.Println(scalar.Scalar(math.MaxInt64).Add(1)) // an integer that would overflow becomes a float
	// Output:
	// 42
	// 3
	// 9.22337203685478e+18
}

func ExampleSV_Sub() {
	fmt.Println(scalar.Scalar(10).Sub("2.5"))
	// Output:
	// 7.5
}

func ExampleSV_Mul() {
	fmt.Println(scalar.Scalar("6").Mul(7))
	// Output:
	// 42
}

func ExampleSV_Div() {
	fmt.Println(scalar.Scalar(7).Div(2)) // not integer division
	fmt.Println(scalar.Scalar(8).Div(2))
	// Output:
	// 3.5
	// 4
}

func ExampleSV_Mod() {
	fmt.Println(scalar.Scalar(7).Mod(3))
	fmt.Println(scalar.Scalar(-7).Mod(3)) // takes the sign of the right operand
	// Output:
	// 1
	// 2
}

func ExampleSV_Pow() {
	fmt.Println(scalar.Scalar(2).Pow(10))
	fmt.Println(scalar.Scalar(2).Pow(-1))
	// Output:
	// 1024
	// 0.5
}

func ExampleSV_Neg() {
	fmt.Println(scalar.Scalar(5).Neg())
	fmt.Println(scalar.Scalar("foo").Neg()) // string madness, as in Perl
	fmt.Println(scalar.Scalar("-foo").Neg())
	// Output:
	// -5
	// -foo
	// +foo
}

func ExampleSV_Concat() {
	fmt.Println(scalar.Scalar("Go").Concat(1.5)) // "Go" . 1.5
	// Output:
	// Go1.5
}

func ExampleSV_Repeat() {
	fmt.Println(scalar.Scalar("ab").Repeat(3)) // "ab" x 3
	fmt.Printf("%q\n", scalar.Scalar("ab").Repeat(0))
	// Output:
	// ababab
	// ""
}

func ExampleSV_Inc() {
	n := scalar.Scalar(9)
	n.Inc()
	id := scalar.Scalar("az9")
	id.Inc() // Perl's magic string increment
	fmt.Println(n, id)
	// Output:
	// 10 ba0
}

func ExampleSV_Dec() {
	s := scalar.Scalar("aa")
	s.Dec() // never magic: "aa" is 0 as a number
	fmt.Println(s)
	// Output:
	// -1
}

func ExampleSV_AddInPlace() {
	totals := scalar.Hash()
	for _, n := range []int{5, 10} {
		totals.Get("sum").AddInPlace(n) // $totals{sum} += $n
	}
	fmt.Println(totals.Get("sum"))
	// Output:
	// 15
}

func ExampleSV_SubInPlace() {
	s := scalar.Scalar(10)
	s.SubInPlace(3) // $s -= 3
	fmt.Println(s)
	// Output:
	// 7
}

func ExampleSV_MulInPlace() {
	s := scalar.Scalar(4)
	s.MulInPlace(2.5) // $s *= 2.5
	fmt.Println(s)
	// Output:
	// 10
}

func ExampleSV_DivInPlace() {
	s := scalar.Scalar(9)
	s.DivInPlace(2) // $s /= 2
	fmt.Println(s)
	// Output:
	// 4.5
}

func ExampleSV_ModInPlace() {
	s := scalar.Scalar(17)
	s.ModInPlace(5) // $s %= 5
	fmt.Println(s)
	// Output:
	// 2
}

func ExampleSV_PowInPlace() {
	s := scalar.Scalar(3)
	s.PowInPlace(2) // $s **= 2
	fmt.Println(s)
	// Output:
	// 9
}

func ExampleSV_ConcatInPlace() {
	s := scalar.Scalar("Go")
	s.ConcatInPlace(" developer") // $s .= " developer"
	fmt.Println(s)
	// Output:
	// Go developer
}

func ExampleSV_RepeatInPlace() {
	s := scalar.Scalar("-")
	s.RepeatInPlace(5) // $s x= 5
	fmt.Println(s)
	// Output:
	// -----
}

func ExampleSV_Eq() {
	fmt.Println(scalar.Scalar(1.0).Eq("1")) // eq compares strings: 1.0 is "1"
	fmt.Println(scalar.Scalar("1.0").Eq(1))
	// Output:
	// true
	// false
}

func ExampleSV_Ne() {
	fmt.Println(scalar.Scalar("go").Ne("Go"))
	// Output:
	// true
}

func ExampleSV_Lt() {
	fmt.Println(scalar.Scalar("10").Lt("9")) // as strings, "10" sorts first
	// Output:
	// true
}

func ExampleSV_Le() {
	fmt.Println(scalar.Scalar("a").Le("a"))
	// Output:
	// true
}

func ExampleSV_Gt() {
	fmt.Println(scalar.Scalar("b").Gt("abc"))
	// Output:
	// true
}

func ExampleSV_Ge() {
	fmt.Println(scalar.Scalar("abc").Ge("abd"))
	// Output:
	// false
}

func ExampleSV_NumEq() {
	fmt.Println(scalar.Scalar("1.0").NumEq(1)) // == compares numbers
	fmt.Println(scalar.Scalar("NaN").NumEq("NaN"))
	// Output:
	// true
	// false
}

func ExampleSV_NumNe() {
	fmt.Println(scalar.Scalar("10").NumNe(10.0))
	// Output:
	// false
}

func ExampleSV_NumLt() {
	fmt.Println(scalar.Scalar("10").NumLt("9")) // as numbers, 9 is smaller
	// Output:
	// false
}

func ExampleSV_NumLe() {
	fmt.Println(scalar.Scalar(2).NumLe("2.0"))
	// Output:
	// true
}

func ExampleSV_NumGt() {
	fmt.Println(scalar.Scalar("9007199254740993").NumGt("9007199254740992")) // integers compare exactly
	// Output:
	// true
}

func ExampleSV_NumGe() {
	fmt.Println(scalar.Scalar(-1).NumGe(0))
	// Output:
	// false
}
