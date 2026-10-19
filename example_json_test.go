package scalar_test

// Examples of JSON and JSONPath. They're run as tests, and they make up
// docs/EXAMPLES.md (see TestReference).

import (
	"encoding/json"
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleFromJSON() {
	doc, err := scalar.FromJSON([]byte(`{"id": 9007199254740993, "depth": 8.25, "stations": ["PAS"], "felt": null}`))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(doc.Get("id")) // whole numbers stay exact
	fmt.Println(doc.Get("depth"), doc.Get("stations").Ref())
	fmt.Println(doc.Exists("felt"), doc.Get("felt").Defined()) // null is undef
	// Output:
	// 9007199254740993
	// 8.25 ARRAY
	// true false
}

func ExampleSV_MarshalJSON() {
	h := scalar.Hash("b", 1, "a", scalar.Value(scalar.Array("x", nil)))
	out, _ := json.Marshal(scalar.Value(h)) // keys sorted, undef as null
	fmt.Println(string(out))
	h.Set("self", scalar.Value(h)) // a loop can't be JSON: an error, not a crash
	_, err := json.Marshal(scalar.Value(h))
	fmt.Println(err != nil)
	// Output:
	// {"a":["x",null],"b":1}
	// true
}

func ExampleAV_MarshalJSON() {
	out, _ := json.Marshal(scalar.Array(1, "two", 3.5))
	fmt.Println(string(out))
	// Output:
	// [1,"two",3.5]
}

func ExampleHV_MarshalJSON() {
	out, _ := json.Marshal(scalar.Hash("z", 1, "a", 2))
	fmt.Println(string(out))
	// Output:
	// {"a":2,"z":1}
}

func ExampleSV_UnmarshalJSON() {
	var record struct {
		ID   int        `json:"id"`
		Data *scalar.SV `json:"data"` // any shape at all
	}
	_ = json.Unmarshal([]byte(`{"id": 1, "data": {"tags": ["a", "b"]}}`), &record)
	fmt.Println(record.ID, record.Data.Get("tags").AV().Join(","))
	// Output:
	// 1 a,b
}

func ExampleSV_Find() {
	doc, _ := scalar.FromJSON([]byte(`{
		"network": {"station": [
			{"code": "PAS", "elevation": 295},
			{"code": "GSC", "elevation": 990}
		]}
	}`))
	fmt.Println(doc.Find("$.network.station[*].code").Join(","))  // every code
	fmt.Println(doc.Find("$..elevation").Join(","))               // every elevation, at any depth
	fmt.Println(doc.Find("$.network.station[-1].code").Join(",")) // the last station's code
	fmt.Println(doc.Find("$.network.sensor") == nil)              // nothing matched
	// Output:
	// PAS,GSC
	// 295,990
	// GSC
	// true
}

func ExampleSV_Find_filter() {
	doc, _ := scalar.FromJSON([]byte(`{"quakes": [
		{"place": "Northern Alaska", "mag": {"value": 3.2}, "stations": ["FAIR"]},
		{"place": "Ridgecrest, CA", "mag": {"value": 5.1, "error": 0.2}, "stations": ["PAS", "GSC"]},
		{"place": "Golfe du Saint-Laurent", "stations": []}
	]}`))
	fmt.Println(doc.Find("$.quakes[?@.mag.value >= 4.5].place").Join("; "))       // a comparison
	fmt.Println(doc.Find("$.quakes[?!@.mag].place").Join("; "))                   // no magnitude at all
	fmt.Println(doc.Find("$.quakes[?count(@.stations[*]) > 1].place").Join("; ")) // a function
	fmt.Println(doc.Find("$.quakes[?match(@.place, 'N.*')].place").Join("; "))    // I-Regexp, whole string
	fmt.Println(doc.Find("$..[?@ == 'GSC']").Join("; "))                          // at any depth
	// Output:
	// Ridgecrest, CA
	// Golfe du Saint-Laurent
	// Ridgecrest, CA
	// Northern Alaska
	// GSC
}

func ExampleQuoteJSONPath() {
	doc, _ := scalar.FromJSON([]byte(`{"regions": [
		{"name": "Alaska", "stations": ["FAIR"]},
		{"name": "Hawai'i", "stations": ["HVO", "KHU"]}
	]}`))
	name := "Hawai'i" // from Go, with a quote in it
	fmt.Println(scalar.QuoteJSONPath(name))
	fmt.Println(doc.Find("$.regions[?@.name == " + scalar.QuoteJSONPath(name) + "].stations[*]").Join(", "))
	// Output:
	// 'Hawai\'i'
	// HVO, KHU
}
