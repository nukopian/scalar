package scalar_test

// These examples are the opening example and the recipes in README.md. Go
// runs them as tests and checks their output, so if one stops working, go
// test says so. Keep the README in step when changing them.

import (
	"encoding/json"
	"fmt"

	"github.com/nukopian/scalar"
)

// The example at the top of README.md: read, extract, clean, then convert
// to typed Go.
func Example_overview() {
	body := []byte(`{"posting": {
		"title": " Senior Go Developer ",
		"location": null,
		"description": "Salary: £45,000 - £55,000 per annum",
		"skills": [" Go", "PostgreSQL ", "Docker"]
	}}`)
	type Posting struct {
		Title  string   `json:"title"`
		Skills []string `json:"skills"`
	}

	doc, err := scalar.FromJSON(body) // whatever shape it turns out to be
	if err != nil {
		panic(err)
	}

	// Clean every string in the structure, at any depth.
	doc.Apply(func(v *scalar.SV) any {
		if v.Kind() == scalar.String {
			return v.Trim()
		}
		return v
	})

	// Read nested values: anything missing or null is undef, not a panic.
	title := doc.At("posting.title").Str()
	location := doc.At("posting.location").Or("Unknown").Str()
	remote := doc.At("posting.remote").Bool() // not there at all: false

	// Pull a figure out of text: no match gives -1, not an error to handle.
	salary := doc.At("posting.description").
		M(`£([\d,]+)`).Get(1).
		Then(func(v *scalar.SV) any { return v.S(`,`, "", "gr") }). // "45,000" → "45000"
		Or(-1).
		Int()

	// Then back to ordinary, typed Go.
	posting, err := scalar.To[Posting](doc.At("posting"))
	if err != nil {
		panic(err)
	}

	fmt.Println(title, "|", location, "|", remote, "|", salary)
	fmt.Printf("%+v\n", posting)
	// Output:
	// Senior Go Developer | Unknown | false | 45000
	// {Title:Senior Go Developer Skills:[Go PostgreSQL Docker]}
}

const toolUseResponse = `{
  "content": [
    {"type": "text", "text": "Here is the metadata."},
    {"type": "tool_use", "name": "job_metadata", "input": {
      "title": "Senior Go Developer",
      "salaryMin": 60000,
      "salaryMax": null,
      "requiredSkills": [" Go ", "PostgreSQL", "go", "Docker "]
    }}
  ],
  "usage": {"input_tokens": 1200}
}`

// Reading a model's tool_use response: find the block, then read fields
// with defaults, telling missing fields from null ones.
func Example_readToolUse() {
	resp, err := scalar.FromJSON([]byte(toolUseResponse))
	if err != nil {
		panic(err)
	}

	input := resp.Find("$.content[*]").FirstMatch(func(b *scalar.SV) bool {
		return b.Path("type").Eq("tool_use")
	}).Path("input")

	fmt.Println(input.Path("title"))
	fmt.Println(resp.At("usage.output_tokens").Or(0)) // $resp->{usage}{output_tokens} // 0

	if v, ok := input.PathOK("salaryMax"); ok && !v.Defined() {
		fmt.Println("salaryMax: null")
	}
	if _, ok := input.PathOK("location"); !ok {
		fmt.Println("location: missing")
	}
	input.HV().Get("location").OrSet("Unknown") // $input->{location} //= "Unknown"
	fmt.Println(input.Path("location"))

	// Output:
	// Senior Go Developer
	// 0
	// salaryMax: null
	// location: missing
	// Unknown
}

// Cleaning a list of skills: trim them in place, then drop duplicates
// regardless of case, keeping the first spelling of each.
func Example_cleanSkills() {
	resp, _ := scalar.FromJSON([]byte(toolUseResponse))
	skills := resp.Find("$..requiredSkills").Get(0).AV()

	for _, s := range skills.All() {
		s.Set(s.Trim()) // changes the skills inside resp
	}

	// grep { !$seen{fc $_}++ } @skills
	seen := scalar.Hash()
	unique := skills.Grep(func(s *scalar.SV) bool {
		return seen.Get(s.Fc().Str()).Inc().Int() == 1
	})
	fmt.Println(unique.Join(", "))

	// Output:
	// Go, PostgreSQL, Docker
}

// salaryFigure is compiled once, when the program starts, as with qr//.
var salaryFigure = scalar.MustCompile(`£([\d,]+)`)

// Pulling salary figures out of text.
func Example_salaryFromText() {
	text := scalar.Value("Salary: £45,000 - £55,000 per annum")

	figures := scalar.Array()
	for m := range text.Matches(salaryFigure) {
		figures.Push(m.Get(1).S(`,`, "", "gr")) // "45,000" → "45000"
	}
	fmt.Println(figures.MinNum(), figures.MaxNum())

	// Output:
	// 45000 55000
}

// Counting skills across postings, ignoring case, and listing the most
// common first.
func Example_countSkills() {
	postings := [][]string{
		{"Go", "Docker"},
		{"go", "Kubernetes"},
		{"GO", "Docker", "Perl"},
	}

	var counts scalar.Bag // the zero value is an empty bag
	for _, skills := range postings {
		for _, skill := range scalar.Array(skills).All() {
			counts.Insert(skill.Fc()) // $counts{fc $skill}++
		}
	}

	for _, skill := range counts.MostCommon(3).All() { // ties keep the order they were first seen in
		fmt.Println(skill, counts.Count(skill))
	}

	// Output:
	// go 3
	// docker 2
	// kubernetes 1
}

// Building records from defaults and writing them out as JSON.
func Example_buildRecords() {
	defaults := scalar.Hash("location", "Unknown", "remote", false)

	rows := scalar.Array()
	for _, title := range []string{"Go Developer", "Perl Developer"} {
		row := scalar.Hash(defaults, "title", title) // %row = (%defaults, title => $title)
		rows.Push(scalar.Value(row))                 // push @rows, \%row
	}
	rows.Get(1).Path("remote").Set(true) // changes only the second row

	out, _ := json.Marshal(rows)
	fmt.Println(string(out))

	// Output:
	// [{"location":"Unknown","remote":false,"title":"Go Developer"},{"location":"Unknown","remote":true,"title":"Perl Developer"}]
}

// Storing an embedding vector as bytes, 4 for each float32, and reading it
// back.
func Example_packEmbedding() {
	vector := []float64{0.5, -0.25, 0.125} // from the embedding model

	blob := scalar.Pack("f<*", vector) // pack("f<*", @vector): little-endian float32s
	fmt.Println(blob.Length())         // 4 bytes each
	fmt.Printf("%x\n", blob)           // in hex, for a log

	back := scalar.Unpack("f<*", blob.Bytes()).NumSlice() // a []float64 again
	fmt.Println(back)

	// Output:
	// 12
	// 0000003f000080be0000003e
	// [0.5 -0.25 0.125]
}
