package scalar_test

// Examples of JSON and JSONPath. They're run as tests, and they make up the
// Reference section of README.md (see TestReference).

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
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
	h := scalar.Hash("b", 1, "a", scalar.Scalar(scalar.Array("x", nil)))
	out, _ := json.Marshal(scalar.Scalar(h)) // keys sorted, undef as null
	fmt.Println(string(out))
	h.Set("self", scalar.Scalar(h)) // a loop can't be JSON: an error, not a crash
	_, err := json.Marshal(scalar.Scalar(h))
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
