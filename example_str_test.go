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
	s := scalar.Value("12 km NNE of Ridgecrest, CA")
	fmt.Println(s.Substr(13))           // substr($s, 13)
	fmt.Println(s.Substr(13, 10))       // substr($s, 13, 10)
	fmt.Println(s.Substr(-2))           // from the end
	fmt.Println(s.Substr(0, -15))       // all but the last 15
	fmt.Println(s.Substr(99).Defined()) // entirely outside: undef
	// Output:
	// Ridgecrest, CA
	// Ridgecrest
	// CA
	// 12 km NNE of
	// false
}

func ExampleSV_SubstrReplace() {
	s := scalar.Value("12 km NNE of Ridgecrest, CA")
	old := s.SubstrReplace(0, 2, "15") // substr($s, 0, 2) = "15"
	fmt.Println(s, "/", old)
	// Output:
	// 15 km NNE of Ridgecrest, CA / 12
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
	fmt.Println(scalar.QuoteMeta("M4.5+ (USGS)"))
	alert := scalar.Value("Alerts for M4.5+ quakes")
	fmt.Println(alert.M(scalar.QuoteMeta("M4.5+")) != nil) // a literal match
	// Output:
	// M4\.5\+\ \(USGS\)
	// true
}

func ExampleSV_Interpolate() {
	doc, _ := scalar.FromJSON([]byte(`{"mag": 5.1, "place": "Ridgecrest, CA",
		"stations": ["PAS", "GSC"], "depth": {"km": 8}}`))
	fmt.Println(doc.Interpolate("M$mag near $place: @stations"))   // doc's keys are the variables
	fmt.Println(doc.Interpolate("Depth $depth{km} km"))            // subscripts, as in Perl
	fmt.Println(doc.Interpolate("Depth ${depth.km} km"))           // any At path in ${...}
	fmt.Println(doc.Interpolate(`[$cdi] \$0 alerts\@example.com`)) // missing is ""; \$ and \@ are literal
	// Output:
	// M5.1 near Ridgecrest, CA: PAS GSC
	// Depth 8 km
	// Depth 8 km
	// [] $0 alerts@example.com
}

func ExampleHV_Interpolate() {
	row := scalar.Hash("name", "Ada", "langs", scalar.Value([]string{"Go", "Perl"}))
	fmt.Println(row.Interpolate("$name writes @langs"))
	// Output:
	// Ada writes Go Perl
}

func ExampleMustTemplate() {
	var summary = scalar.MustTemplate("$place. Stations: @stations.") // parsed once
	doc, _ := scalar.FromJSON([]byte(`{"place": "Ridgecrest, CA", "stations": ["PAS", "GSC"]}`))
	fmt.Println(doc.Interpolate(summary))
	// Output:
	// Ridgecrest, CA. Stations: PAS GSC.
}

func ExampleCompileTemplate() {
	t, err := scalar.CompileTemplate("Depth ${depth.km") // a template from outside the program: check it
	fmt.Println(t == nil, err)
	// Output:
	// true scalar: bad template "Depth ${depth.km" at offset 7: no } to match the {
}

func ExampleTemplate_ListSeparator() {
	t := scalar.MustTemplate("Stations: @stations").ListSeparator(", ") // local $" = ", "
	fmt.Println(scalar.Hash("stations", scalar.Value([]string{"PAS", "GSC"})).Interpolate(t))
	// Output:
	// Stations: PAS, GSC
}

func ExampleTemplate_String() {
	fmt.Println(scalar.MustTemplate("M$mag near $place"))
	// Output:
	// M$mag near $place
}
