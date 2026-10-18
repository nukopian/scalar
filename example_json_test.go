package scalar_test

// Examples of JSON and JSONPath. They're run as tests, and they make up
// docs/EXAMPLES.md (see TestReference).

import (
	"encoding/json"
	"fmt"

	"github.com/nukopian/scalar"
)

func ExampleFromJSON() {
	doc, err := scalar.FromJSON([]byte(`{"id": 9007199254740993, "pay": 45000.5, "skills": ["Go"], "bonus": null}`))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(doc.Path("id")) // whole numbers stay exact
	fmt.Println(doc.Path("pay"), doc.Path("skills").Ref())
	fmt.Println(doc.HV().Exists("bonus"), doc.Path("bonus").Defined()) // null is undef
	// Output:
	// 9007199254740993
	// 45000.5 ARRAY
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
	fmt.Println(record.ID, record.Data.Path("tags").AV().Join(","))
	// Output:
	// 1 a,b
}

func ExampleSV_Find() {
	doc, _ := scalar.FromJSON([]byte(`{
		"store": {"book": [
			{"title": "Go", "price": 30},
			{"title": "Perl", "price": 45}
		]}
	}`))
	fmt.Println(doc.Find("$.store.book[*].title").Join(","))  // every title
	fmt.Println(doc.Find("$..price").Join(","))               // every price, at any depth
	fmt.Println(doc.Find("$.store.book[-1].title").Join(",")) // the last book's title
	fmt.Println(doc.Find("$.store.magazine") == nil)          // nothing matched
	// Output:
	// Go,Perl
	// 30,45
	// Perl
	// true
}

func ExampleSV_Find_filter() {
	doc, _ := scalar.FromJSON([]byte(`{"jobs": [
		{"title": "Nurse", "salary": {"min": 32000}, "skills": ["care"]},
		{"title": "Developer", "salary": {"min": 55000, "max": 70000}, "skills": ["Go", "SQL"]},
		{"title": "Café manager", "skills": []}
	]}`))
	fmt.Println(doc.Find("$.jobs[?@.salary.min >= 40000].title").Join(","))  // a comparison
	fmt.Println(doc.Find("$.jobs[?!@.salary].title").Join(","))              // no salary at all
	fmt.Println(doc.Find("$.jobs[?count(@.skills[*]) > 1].title").Join(",")) // a function
	fmt.Println(doc.Find("$.jobs[?match(@.title, 'N.*')].title").Join(","))  // I-Regexp, whole string
	fmt.Println(doc.Find("$..[?@ == 'SQL']").Join(","))                      // at any depth
	// Output:
	// Developer
	// Café manager
	// Developer
	// Nurse
	// SQL
}

func ExampleQuoteJSONPath() {
	doc, _ := scalar.FromJSON([]byte(`{"sections": [
		{"kind": "summary", "lines": ["We're hiring."]},
		{"kind": "it's benefits", "lines": ["Pension", "Gym"]}
	]}`))
	kind := "it's benefits" // from Go, with a quote in it
	fmt.Println(scalar.QuoteJSONPath(kind))
	fmt.Println(doc.Find("$.sections[?@.kind == " + scalar.QuoteJSONPath(kind) + "].lines[*]").Join(", "))
	// Output:
	// 'it\'s benefits'
	// Pension, Gym
}
