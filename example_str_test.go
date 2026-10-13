package scalar_test

// Examples of Perl's string functions. They're run as tests, and they make
// up the Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleSV_Length() {
	fmt.Println(scalar.Scalar("café").Length()) // characters, not bytes
	fmt.Println(scalar.Scalar(12345).Length())  // in string context
	// Output:
	// 4
	// 5
}

func ExampleSV_Substr() {
	s := scalar.Scalar("Senior Go Developer")
	fmt.Println(s.Substr(7))            // substr($s, 7)
	fmt.Println(s.Substr(7, 2))         // substr($s, 7, 2)
	fmt.Println(s.Substr(-9))           // from the end
	fmt.Println(s.Substr(0, -10))       // all but the last 10
	fmt.Println(s.Substr(99).Defined()) // entirely outside: undef
	// Output:
	// Go Developer
	// Go
	// Developer
	// Senior Go
	// false
}

func ExampleSV_SubstrReplace() {
	s := scalar.Scalar("Junior Go Developer")
	old := s.SubstrReplace(0, 6, "Senior") // substr($s, 0, 6) = "Senior"
	fmt.Println(s, "/", old)
	// Output:
	// Senior Go Developer / Junior
}

func ExampleSV_Index() {
	s := scalar.Scalar("café au lait")
	fmt.Println(s.Index("au"))   // a character position, ready for Substr
	fmt.Println(s.Index("a", 2)) // starting from position 2
	fmt.Println(s.Index("tea"))
	// Output:
	// 5
	// 5
	// -1
}

func ExampleSV_RIndex() {
	s := scalar.Scalar("a/b/c")
	fmt.Println(s.RIndex("/"))    // the last one
	fmt.Println(s.RIndex("/", 2)) // the last one at or before position 2
	// Output:
	// 3
	// 1
}

func ExampleSV_Lc() {
	fmt.Println(scalar.Scalar("ÉCOLE Go").Lc())
	// Output:
	// école go
}

func ExampleSV_Uc() {
	fmt.Println(scalar.Scalar("straße").Uc()) // full Unicode mappings
	// Output:
	// STRASSE
}

func ExampleSV_Fc() {
	a, b := scalar.Scalar("Straße"), scalar.Scalar("STRASSE")
	fmt.Println(a.Lc().Eq(b.Lc())) // lc isn't enough for this
	fmt.Println(a.Fc().Eq(b.Fc())) // fc is: compare case-insensitively with it
	// Output:
	// false
	// true
}

func ExampleSV_UcFirst() {
	fmt.Println(scalar.Scalar("élan vital").UcFirst())
	// Output:
	// Élan vital
}

func ExampleSV_LcFirst() {
	fmt.Println(scalar.Scalar("HELLO").LcFirst())
	// Output:
	// hELLO
}

func ExampleSV_Trim() {
	fmt.Printf("%q\n", scalar.Scalar("\u00a0 padded \t").Trim()) // Unicode spaces too
	// Output:
	// "padded"
}

func ExampleQuoteMeta() {
	fmt.Println(scalar.QuoteMeta("C++ (.NET)"))
	posting := scalar.Scalar("Skills: C++ and .NET")
	fmt.Println(posting.M(scalar.QuoteMeta("C++")) != nil) // a literal match
	// Output:
	// C\+\+\ \(\.NET\)
	// true
}
