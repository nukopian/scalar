package scalar_test

// Examples of regular expressions. They're run as tests, and they make up
// the Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleSV_M() {
	s := scalar.Value("Salary: £45,000 - £55,000")
	if s.M(`(?i)salary`) != nil { // if ($s =~ /salary/i)
		fmt.Println("matched")
	}
	m := s.M(`£([\d,]+)\s*-\s*£([\d,]+)`)
	fmt.Println(m.Get(0)) // $&
	fmt.Println(m.Get(1), m.Get(2))
	for _, each := range s.M(`£([\d,]+)`, "g").All() { // $s =~ /£([\d,]+)/g
		fmt.Println(each.AV().Get(1))
	}
	fmt.Println(s.M(`remote`).Get(1).Defined()) // no match: nil, still safe to read
	// Output:
	// matched
	// £45,000 - £55,000
	// 45,000 55,000
	// 45,000
	// 55,000
	// false
}

func ExampleSV_Named() {
	d := scalar.Value("Closing 2026-11-30").Named(`(?<year>\d{4})-(?<month>\d\d)-(?<day>\d\d)`)
	fmt.Println(d.Get("year"), d.Get("month"), d.Get("day")) // $+{year} and so on
	// Output:
	// 2026 11 30
}

func ExampleSV_S() {
	s := scalar.Value("a  b   c")
	n := s.S(`\s+`, " ", "g") // $s =~ s/\s+/ /g
	fmt.Println(s, n)
	name := scalar.Value("Lovelace, Ada")
	name.S(`(\w+), (\w+)`, "$2 $1") // $1 and $2 in the replacement
	fmt.Println(name)
	clean := scalar.Value("x1y2").S(`\d`, "", "gr") // s///r: a new string; the original is unchanged
	fmt.Println(clean)
	doubled := scalar.Value("3 apples, 4 pears")
	doubled.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g") // s///e
	fmt.Println(doubled)
	// Output:
	// a b c 2
	// Ada Lovelace
	// xy
	// 6 apples, 8 pears
}

func ExampleSV_Split() {
	fmt.Println(scalar.Value("a,b,,c,,").Split(`,`).Length())        // trailing empty fields dropped
	fmt.Println(scalar.Value("a1b2c").Split(`(\d)`).Join("|"))       // captures included
	fmt.Println(scalar.Value("a,b,c,d").Split(`,`, 2).Join("|"))     // a limit
	fmt.Println(scalar.Value("  some  words ").Split(" ").Join("|")) // awk mode
	// Output:
	// 4
	// a|1|b|2|c
	// a|b,c,d
	// some|words
}

func ExampleSV_Tr() {
	s := scalar.Value("hello")
	s.Tr("a-z", "A-Z") // $s =~ tr/a-z/A-Z/
	fmt.Println(s)
	fmt.Println(scalar.Value("hello world").Tr("a-z", ""))  // counts: tr/a-z// returns how many
	fmt.Println(scalar.Value("aabbcc").Tr("a-z", "", "sr")) // squeeze, returning a new string
	// Output:
	// HELLO
	// 10
	// abc
}

func ExampleCompile() {
	re, err := scalar.Compile(`£([\d,]+`) // a pattern from outside the program: check it
	fmt.Println(re == nil, err != nil)
	// Output:
	// true true
}

func ExampleMustCompile() {
	salary := scalar.MustCompile(`£([\d,]+)`) // my $salary = qr/£([\d,]+)/
	s := scalar.Value("£45,000 to £55,000")
	fmt.Println(s.M(salary, "g").Length()) // use it anywhere a pattern string goes
	fmt.Println(s.S(salary, "£X", "gr"))
	insensitive := scalar.MustCompile(`remote`, "i") // flags are fixed when compiling
	fmt.Println(scalar.Value("REMOTE").M(insensitive) != nil)
	// Output:
	// 2
	// £X to £X
	// true
}

func ExampleRegexp_String() {
	fmt.Println(scalar.MustCompile(`\d+`, "i"))
	// Output:
	// \d+
}
