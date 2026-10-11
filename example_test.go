package scalar_test

// These examples are the recipes in README.md. Go runs them as tests and
// checks their output, so if one stops working, go test says so. Keep the
// README in step when changing them.

import (
	"cmp"
	"encoding/json"
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
)

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
	fmt.Println(resp.Path("usage", "output_tokens").Or(0)) // $resp->{usage}{output_tokens} // 0

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

// Pulling salary figures out of text.
func Example_salaryFromText() {
	text := scalar.Scalar("Salary: £45,000 - £55,000 per annum")

	figures := scalar.Array()
	for _, m := range text.M(`£([\d,]+)`, "g").All() {
		figures.Push(m.AV().Get(1).S(`,`, "", "gr")) // "45,000" → "45000"
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

	counts := scalar.Hash()
	for _, skills := range postings {
		for _, skill := range scalar.Array(skills).All() {
			counts.Get(skill.Fc().Str()).Inc() // $counts{fc $skill}++
		}
	}

	top := counts.Keys().Sort(func(x, y *scalar.SV) int {
		return cmp.Or(
			scalar.NumCmp(counts.Get(y.Str()), counts.Get(x.Str())), // most common first,
			scalar.StrCmp(x, y), // then alphabetically
		)
	})
	for _, skill := range top.Slice(0, 1, 2).All() {
		fmt.Println(skill, counts.Get(skill.Str()))
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
		rows.Push(scalar.Scalar(row))                // push @rows, \%row
	}
	rows.Get(1).Path("remote").Set(true) // changes only the second row

	out, _ := json.Marshal(rows)
	fmt.Println(string(out))

	// Output:
	// [{"location":"Unknown","remote":false,"title":"Go Developer"},{"location":"Unknown","remote":true,"title":"Perl Developer"}]
}
