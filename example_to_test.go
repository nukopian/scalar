package scalar_test

// Examples of leaving scalar for ordinary Go values: To and Decode. They're
// run as tests, and they make up the Reference section of README.md (see
// TestReference).

import (
	"errors"
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

func ExampleTo() {
	doc, _ := scalar.FromJSON([]byte(`{
		"title": "Go Developer",
		"skills": ["Go", "SQL"],
		"salary": {"min": "45000", "max": 55000}
	}`))
	type Posting struct {
		Title  string   `json:"title"`
		Skills []string `json:"skills"`
		Salary struct {
			Min int `json:"min"` // "45000" fills it, as Perl would numify it
			Max int `json:"max"`
		} `json:"salary"`
	}
	p, err := scalar.To[Posting](doc) // the type you want, in brackets
	if err != nil {
		panic(err)
	}
	fmt.Println(p.Title, p.Skills, p.Salary.Min, p.Salary.Max)
	// Output:
	// Go Developer [Go SQL] 45000 55000
}

func ExampleTo_slice() {
	doc, _ := scalar.FromJSON([]byte(`{"skills": ["Go", "SQL"], "years": [3, "5"]}`))
	skills, _ := scalar.To[[]string](doc.At("skills")) // from a scalar holding an array
	years, _ := scalar.To[[]int](doc.At("years").AV()) // or from the array itself
	fmt.Printf("%q %v\n", skills, years)
	// Output:
	// ["Go" "SQL"] [3 5]
}

func ExampleTo_number() {
	salary, err := scalar.To[int](scalar.Value(" 55000 ")) // a numeric string, as Perl reads it
	fmt.Println(salary, err)
	rate, err := scalar.To[float64](scalar.Value("37.5"))
	fmt.Println(rate, err)
	// Output:
	// 55000 <nil>
	// 37.5 <nil>
}

func ExampleTo_map() {
	doc, _ := scalar.FromJSON([]byte(`{"go": 3, "perl": "2"}`))
	counts, _ := scalar.To[map[string]int](doc) // a hash fills a map
	fmt.Println(counts["go"], counts["perl"])
	// Output:
	// 3 2
}

func ExampleTo_set() {
	langs := scalar.NewSet("Go", "Perl", 5)
	members, _ := scalar.To[[]string](langs) // a set gives its members, in insertion order
	fmt.Printf("%q\n", members)
	// Output:
	// ["Go" "Perl" "5"]
}

func ExampleTo_bag() {
	skills := scalar.NewBag("Go", "Go", "Perl")
	counts, _ := scalar.To[map[string]int](skills)    // a bag gives its counts
	names, _ := scalar.To[[]string](skills.Members()) // and Members its distinct members
	fmt.Println(counts["Go"], counts["Perl"], names)
	// Output:
	// 2 1 [Go Perl]
}

func ExampleTo_any() {
	doc, _ := scalar.FromJSON([]byte(`{"tags": ["go", 1]}`))
	plain, _ := scalar.To[any](doc) // the plain Go form, as Any gives it
	fmt.Printf("%v\n", plain)
	// Output:
	// map[tags:[go 1]]
}

func ExampleTo_missing() {
	doc, _ := scalar.FromJSON([]byte(`{"bonus": null}`))
	bonus, err := scalar.To[*int](doc.At("bonus")) // null, or missing: the zero value, here nil
	fmt.Println(bonus, err)
	perks, err := scalar.To[[]string](doc.At("perks"))
	fmt.Println(perks == nil, err)
	// Output:
	// <nil> <nil>
	// true <nil>
}

func ExampleTo_error() {
	doc, _ := scalar.FromJSON([]byte(`{"salary": {"max": "competitive"}}`))
	type Posting struct {
		Salary struct {
			Max int `json:"max"`
		} `json:"salary"`
	}
	p, err := scalar.To[Posting](doc)
	fmt.Println(err)
	fmt.Println(p.Salary.Max) // on an error, the zero value
	var de *scalar.DecodeError
	if errors.As(err, &de) {
		fmt.Println(doc.At(de.Path)) // the path leads At to the value
	}
	// Output:
	// scalar: Decode: salary.max: want int, got "competitive"
	// 0
	// competitive
}

func ExampleTo_strict() {
	years := scalar.Array("3", "five", 7)
	fmt.Println(years.IntSlice())     // Perl's numeric context: never fails, "five" is 0
	_, err := scalar.To[[]int](years) // To checks
	fmt.Println(err)
	// Output:
	// [3 0 7]
	// scalar: Decode: [1]: want int, got "five"
}

func ExampleTo_loose() {
	doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "extra": {"benefits": ["Pension"]}}`))
	type Posting struct {
		Title string     `json:"title"`
		Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar: a deep copy
	}
	p, _ := scalar.To[Posting](doc)
	fmt.Println(p.Title, p.Extra.At("benefits[0]"))
	// Output:
	// Go Developer Pension
}

func ExampleSV_Decode() {
	doc, _ := scalar.FromJSON([]byte(`{
		"title": "Go Developer",
		"skills": ["Go", "SQL"],
		"salary": {"min": "45000", "max": 55000},
		"extra": {"benefits": ["Pension"]}
	}`))
	var posting struct {
		Title  string   `json:"title"`
		Skills []string `json:"skills"`
		Salary struct {
			Min int `json:"min"` // "45000" fills it, as Perl would numify it
			Max int `json:"max"`
		} `json:"salary"`
		Extra *scalar.SV `json:"extra"` // a part that stays loose
	}
	if err := doc.Decode(&posting); err != nil {
		panic(err)
	}
	fmt.Println(posting.Title, posting.Skills, posting.Salary.Min, posting.Salary.Max)
	fmt.Println(posting.Extra.At("benefits[0]"))
	// Output:
	// Go Developer [Go SQL] 45000 55000
	// Pension
}

func ExampleDecodeError_Error() {
	doc, _ := scalar.FromJSON([]byte(`{"salary": {"max": "competitive"}}`))
	var posting struct {
		Salary struct {
			Max int `json:"max"`
		} `json:"salary"`
	}
	err := doc.Decode(&posting)
	fmt.Println(err)
	var de *scalar.DecodeError
	if errors.As(err, &de) {
		fmt.Println(doc.At(de.Path)) // the path leads At to the value
	}
	// Output:
	// scalar: Decode: salary.max: want int, got "competitive"
	// competitive
}
