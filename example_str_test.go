package scalar_test

// Examples of Perl's string functions. They're run as tests, and they make
// up docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleSV_Length() {
	fmt.Println(scalar.Value("café").Length()) // characters, not bytes
	fmt.Println(scalar.Value(12345).Length())  // in string context
	// Output:
	// 4
	// 5
}

func ExampleSV_Substr() {
	s := scalar.Value("Senior Go Developer")
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
	s := scalar.Value("Junior Go Developer")
	old := s.SubstrReplace(0, 6, "Senior") // substr($s, 0, 6) = "Senior"
	fmt.Println(s, "/", old)
	// Output:
	// Senior Go Developer / Junior
}

func ExampleSV_Index() {
	s := scalar.Value("café au lait")
	fmt.Println(s.Index("au"))   // a character position, ready for Substr
	fmt.Println(s.Index("a", 2)) // starting from position 2
	fmt.Println(s.Index("tea"))
	// Output:
	// 5
	// 5
	// -1
}

func ExampleSV_RIndex() {
	s := scalar.Value("a/b/c")
	fmt.Println(s.RIndex("/"))    // the last one
	fmt.Println(s.RIndex("/", 2)) // the last one at or before position 2
	// Output:
	// 3
	// 1
}

func ExampleSV_Lc() {
	fmt.Println(scalar.Value("ÉCOLE Go").Lc())
	// Output:
	// école go
}

func ExampleSV_Uc() {
	fmt.Println(scalar.Value("straße").Uc()) // full Unicode mappings
	// Output:
	// STRASSE
}

func ExampleSV_Fc() {
	a, b := scalar.Value("Straße"), scalar.Value("STRASSE")
	fmt.Println(a.Lc().Eq(b.Lc())) // lc isn't enough for this
	fmt.Println(a.Fc().Eq(b.Fc())) // fc is: compare case-insensitively with it
	// Output:
	// false
	// true
}

func ExampleSV_UcFirst() {
	fmt.Println(scalar.Value("élan vital").UcFirst())
	// Output:
	// Élan vital
}

func ExampleSV_LcFirst() {
	fmt.Println(scalar.Value("HELLO").LcFirst())
	// Output:
	// hELLO
}

func ExampleSV_Trim() {
	fmt.Printf("%q\n", scalar.Value("\u00a0 padded \t").Trim()) // Unicode spaces too
	// Output:
	// "padded"
}

func ExampleQuoteMeta() {
	fmt.Println(scalar.QuoteMeta("C++ (.NET)"))
	posting := scalar.Value("Skills: C++ and .NET")
	fmt.Println(posting.M(scalar.QuoteMeta("C++")) != nil) // a literal match
	// Output:
	// C\+\+\ \(\.NET\)
	// true
}

func ExampleSV_Interpolate() {
	doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "location": "Leeds",
		"skills": ["Go", "SQL"], "salary": {"max": 55000}}`))
	fmt.Println(doc.Interpolate("$title in $location: @skills"))   // doc's keys are the variables
	fmt.Println(doc.Interpolate("Up to $salary{max}"))             // subscripts, as in Perl
	fmt.Println(doc.Interpolate("Up to ${salary.max}"))            // any At path in ${...}
	fmt.Println(doc.Interpolate(`[$bonus] \$5 jobs\@example.com`)) // missing is ""; \$ and \@ are literal
	// Output:
	// Go Developer in Leeds: Go SQL
	// Up to 55000
	// Up to 55000
	// [] $5 jobs@example.com
}

func ExampleHV_Interpolate() {
	row := scalar.Hash("name", "Ada", "langs", scalar.Value([]string{"Go", "Perl"}))
	fmt.Println(row.Interpolate("$name writes @langs"))
	// Output:
	// Ada writes Go Perl
}

func ExampleMustTemplate() {
	var summary = scalar.MustTemplate("$title. Skills: @skills.") // parsed once
	doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
	fmt.Println(doc.Interpolate(summary))
	// Output:
	// Go Developer. Skills: Go SQL.
}

func ExampleCompileTemplate() {
	t, err := scalar.CompileTemplate("Up to ${salary.max") // a template from outside the program: check it
	fmt.Println(t == nil, err)
	// Output:
	// true scalar: bad template "Up to ${salary.max" at offset 7: no } to match the {
}

func ExampleTemplate_ListSeparator() {
	t := scalar.MustTemplate("Skills: @skills").ListSeparator(", ") // local $" = ", "
	fmt.Println(scalar.Hash("skills", scalar.Value([]string{"Go", "SQL"})).Interpolate(t))
	// Output:
	// Skills: Go, SQL
}

func ExampleTemplate_String() {
	fmt.Println(scalar.MustTemplate("$title in $location"))
	// Output:
	// $title in $location
}
