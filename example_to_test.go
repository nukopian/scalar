package scalar_test

// Examples of leaving scalar for ordinary Go values: To and Decode. They're
// run as tests, and they make up docs/EXAMPLES.md (see TestReference).

import (
	"errors"
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleTo() {
	doc, _ := scalar.FromJSON([]byte(`{
		"id": "ci40123456",
		"stations": ["PAS", "GSC"],
		"depth": {"km": "8", "error": 2}
	}`))
	type Event struct {
		ID       string   `json:"id"`
		Stations []string `json:"stations"`
		Depth    struct {
			Km    int `json:"km"` // "8" fills it, as Perl would numify it
			Error int `json:"error"`
		} `json:"depth"`
	}
	e, err := scalar.To[Event](doc) // the type you want, in brackets
	if err != nil {
		panic(err)
	}
	fmt.Println(e.ID, e.Stations, e.Depth.Km, e.Depth.Error)
	// Output:
	// ci40123456 [PAS GSC] 8 2
}

func ExampleTo_slice() {
	doc, _ := scalar.FromJSON([]byte(`{"stations": ["PAS", "GSC"], "felt": [3, "5"]}`))
	stations, _ := scalar.To[[]string](doc.At("stations")) // from a scalar holding an array
	felt, _ := scalar.To[[]int](doc.At("felt").AV())       // or from the array itself
	fmt.Printf("%q %v\n", stations, felt)
	// Output:
	// ["PAS" "GSC"] [3 5]
}

func ExampleTo_number() {
	felt, err := scalar.To[int](scalar.Value(" 1204 ")) // a numeric string, as Perl reads it
	fmt.Println(felt, err)
	mag, err := scalar.To[float64](scalar.Value("4.5"))
	fmt.Println(mag, err)
	// Output:
	// 1204 <nil>
	// 4.5 <nil>
}

func ExampleTo_map() {
	doc, _ := scalar.FromJSON([]byte(`{"ci": 3, "us": "2"}`))
	counts, _ := scalar.To[map[string]int](doc) // a hash fills a map
	fmt.Println(counts["ci"], counts["us"])
	// Output:
	// 3 2
}

func ExampleTo_set() {
	networks := scalar.NewSet("ci", "us", 5)
	members, _ := scalar.To[[]string](networks) // a set gives its members, in insertion order
	fmt.Printf("%q\n", members)
	// Output:
	// ["ci" "us" "5"]
}

func ExampleTo_bag() {
	regions := scalar.NewBag("Alaska", "Alaska", "Nevada")
	counts, _ := scalar.To[map[string]int](regions)    // a bag gives its counts
	names, _ := scalar.To[[]string](regions.Members()) // and Members its distinct members
	fmt.Println(counts["Alaska"], counts["Nevada"], names)
	// Output:
	// 2 1 [Alaska Nevada]
}

func ExampleTo_any() {
	doc, _ := scalar.FromJSON([]byte(`{"types": ["origin", 1]}`))
	plain, _ := scalar.To[any](doc) // the plain Go form, as Any gives it
	fmt.Printf("%v\n", plain)
	// Output:
	// map[types:[origin 1]]
}

func ExampleTo_missing() {
	doc, _ := scalar.FromJSON([]byte(`{"felt": null}`))
	felt, err := scalar.To[*int](doc.At("felt")) // null, or missing: the zero value, here nil
	fmt.Println(felt, err)
	products, err := scalar.To[[]string](doc.At("products"))
	fmt.Println(products == nil, err)
	// Output:
	// <nil> <nil>
	// true <nil>
}

// An optional value is a pointer: nil if the value is undef, missing or
// null, and otherwise a pointer to it, converted and checked. There's no
// StrPtr or NumPtr method, as To[*T] does it for any type, and refuses a
// value that doesn't fit, where Num would quietly make "abc" 0.
func ExampleTo_optional() {
	doc, _ := scalar.FromJSON([]byte(`{"value": 4.5, "error": null, "type": "ml", "felt": "widely"}`))

	value, _ := scalar.To[*float64](doc.At("value"))
	magErr, _ := scalar.To[*float64](doc.At("error")) // null: nil
	none, _ := scalar.To[*string](doc.At("network"))  // missing: nil
	fmt.Println(*value, magErr, none)

	_, err := scalar.To[*float64](doc.At("felt")) // "widely" isn't a number
	fmt.Println(err)

	// The same, for a struct's optional fields, as a feed's often are.
	type Mag struct {
		Value, Error *float64
		Type         *string
	}
	m, _ := scalar.To[Mag](doc)
	fmt.Println(*m.Value, m.Error, *m.Type)
	// Output:
	// 4.5 <nil> <nil>
	// scalar: Decode: want float64, got "widely"
	// 4.5 <nil> ml
}

func ExampleTo_error() {
	doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": "unknown"}}`))
	type Event struct {
		Depth struct {
			Km int `json:"km"`
		} `json:"depth"`
	}
	e, err := scalar.To[Event](doc)
	fmt.Println(err)
	fmt.Println(e.Depth.Km) // on an error, the zero value
	var de *scalar.DecodeError
	if errors.As(err, &de) {
		fmt.Println(doc.At(de.Path)) // the path leads At to the value
	}
	// Output:
	// scalar: Decode: depth.km: want int, got "unknown"
	// 0
	// unknown
}

func ExampleTo_strict() {
	felt := scalar.Array("3", "five", 7)
	fmt.Println(felt.IntSlice())     // Perl's numeric context: never fails, "five" is 0
	_, err := scalar.To[[]int](felt) // To checks
	fmt.Println(err)
	// Output:
	// [3 0 7]
	// scalar: Decode: [1]: want int, got "five"
}

func ExampleTo_loose() {
	doc, _ := scalar.FromJSON([]byte(`{"id": "ci40123456", "extra": {"products": ["shakemap"]}}`))
	type Event struct {
		ID    string     `json:"id"`
		Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar: a deep copy
	}
	e, _ := scalar.To[Event](doc)
	fmt.Println(e.ID, e.Extra.At("products[0]"))
	// Output:
	// ci40123456 shakemap
}

func ExampleSV_Decode() {
	doc, _ := scalar.FromJSON([]byte(`{
		"id": "ci40123456",
		"stations": ["PAS", "GSC"],
		"depth": {"km": "8", "error": 2},
		"extra": {"products": ["shakemap"]}
	}`))
	var event struct {
		ID       string   `json:"id"`
		Stations []string `json:"stations"`
		Depth    struct {
			Km    int `json:"km"` // "8" fills it, as Perl would numify it
			Error int `json:"error"`
		} `json:"depth"`
		Extra *scalar.SV `json:"extra"` // a part that stays loose
	}
	if err := doc.Decode(&event); err != nil {
		panic(err)
	}
	fmt.Println(event.ID, event.Stations, event.Depth.Km, event.Depth.Error)
	fmt.Println(event.Extra.At("products[0]"))
	// Output:
	// ci40123456 [PAS GSC] 8 2
	// shakemap
}

func ExampleDecodeError_Error() {
	doc, _ := scalar.FromJSON([]byte(`{"depth": {"km": "unknown"}}`))
	var event struct {
		Depth struct {
			Km int `json:"km"`
		} `json:"depth"`
	}
	err := doc.Decode(&event)
	fmt.Println(err)
	var de *scalar.DecodeError
	if errors.As(err, &de) {
		fmt.Println(doc.At(de.Path)) // the path leads At to the value
	}
	// Output:
	// scalar: Decode: depth.km: want int, got "unknown"
	// unknown
}
