package scalar_test

// These examples are the opening example in README.md and the recipes in
// docs/HOWTO.md. Go runs them as tests and checks their output, so if one
// stops working, go test says so. Keep the documents in step when changing
// them.

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nukopian/scalar"
)

// The example at the top of README.md: read an earthquake feed, search
// and filter it, read nested values, then convert to typed Go.
func Example_overview() {
	body := []byte(`{"type": "FeatureCollection", "features": [
		{"properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "time": 1760781600000},
		 "geometry": {"coordinates": [-117.6, 35.7, 8.2]}},
		{"properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "time": 1760783400000},
		 "geometry": {"coordinates": [-122.8, 38.8, 1.9]}},
		{"properties": {"mag": 4.7, "place": null, "time": 1760785200000},
		 "geometry": {"coordinates": [-125.9, 40.3, 10.0]}},
		{"properties": {"mag": null, "place": "Alaska Peninsula", "time": 1760787000000},
		 "geometry": {"coordinates": [-157.1, 56.2, 35.4]}}
	]}`)
	type Quake struct {
		Mag   float64 `json:"mag"`
		Place string  `json:"place"`
	}

	feed, err := scalar.FromJSON(body) // a USGS earthquake feed, in GeoJSON
	if err != nil {
		panic(err)
	}

	// Search the whole tree with JSONPath: the strongest magnitude anywhere.
	fmt.Println("Strongest:", feed.Find("$..mag").MaxNum())

	// Filter with JSONPath, then read nested values: anything missing or
	// null is undef, not a panic, and Or supplies a default.
	for _, q := range feed.Find("$.features[?@.properties.mag >= 4.5]").All() {
		mag := q.At("properties.mag").Num()
		depth := q.At("geometry.coordinates[2]").Num() // [lon, lat, depth]
		near := q.At("properties.place").
			M(`of (.+)$`).Get(1). // "12 km NNE of Ridgecrest, CA" → "Ridgecrest, CA"
			Or("an unknown place").Str()
		when := time.UnixMilli(q.At("properties.time").Int()).UTC()
		fmt.Printf("M%.1f near %s, %.0f km deep, %s\n", mag, near, depth, when.Format("2 Jan 15:04 MST"))
	}

	// Then back to ordinary, typed Go.
	quakes, err := scalar.To[[]Quake](feed.Find("$.features[*].properties"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", quakes)
	// Output:
	// Strongest: 5.1
	// M5.1 near Ridgecrest, CA, 8 km deep, 18 Oct 10:00 UTC
	// M4.7 near an unknown place, 10 km deep, 18 Oct 11:00 UTC
	// [{Mag:5.1 Place:12 km NNE of Ridgecrest, CA} {Mag:2.3 Place:5 km W of Cobb, CA} {Mag:4.7 Place:} {Mag:0 Place:Alaska Peninsula}]
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
