package scalar_test

// These examples are the recipes in docs/HOWTO.md, which TestHowTo
// generates from them: each Example_name is a recipe, and
// Example_namePlainGo does the same job in plain Go, printing the same.
// Edit them here, then run:
//
//	go test -run TestHowTo -update
//
// The guide shows each example's body, and the declarations of this file
// it refers to, the first time a recipe needs them. A line ending in
// "// test only" is left out of the guide: it keeps the example from
// changing what other tests see, and isn't part of the recipe.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nukopian/scalar"
	"golang.org/x/text/cases"
)

// quakeFeed is a small part of the USGS earthquake feed.
var quakeFeed = []byte(`{
  "type": "FeatureCollection",
  "metadata": {"title": "USGS All Earthquakes, Past Day", "count": 2},
  "features": [
    {"type": "Feature", "id": "nc75012345",
     "properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "felt": null, "alert": null}},
    {"type": "Feature", "id": "ci40123456",
     "properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "alert": null}}
  ]
}`)

func Example_quickStart() {
	feed, err := scalar.FromJSON(quakeFeed)
	if err != nil {
		panic(err)
	}

	place := feed.Get("features", 0, "properties", "place").Str() // $feed->{features}[0]{properties}{place}
	mag := feed.At("features[0].properties.mag").Num()            // the same idea, as one string
	fmt.Println(place, mag)

	type Metadata struct {
		Title string `json:"title"`
		Count int    `json:"count"`
	}
	meta, err := scalar.To[Metadata](feed.At("metadata")) // into your own struct, checked
	if err != nil {
		panic(err)
	}
	fmt.Println(meta.Title, meta.Count)

	// Output:
	// 5 km W of Cobb, CA 2.3
	// USGS All Earthquakes, Past Day 2
}

func Example_quickStartPlainGo() {
	var feed struct {
		Metadata struct {
			Title string `json:"title"`
			Count int    `json:"count"`
		} `json:"metadata"`
		Features []struct {
			Properties struct {
				Place string  `json:"place"`
				Mag   float64 `json:"mag"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(quakeFeed, &feed); err != nil {
		panic(err)
	}

	if len(feed.Features) > 0 { // an index out of range would panic
		p := feed.Features[0].Properties
		fmt.Println(p.Place, p.Mag)
	}
	fmt.Println(feed.Metadata.Title, feed.Metadata.Count)

	// Output:
	// 5 km W of Cobb, CA 2.3
	// USGS All Earthquakes, Past Day 2
}

func Example_readEvent() {
	feed, err := scalar.FromJSON(quakeFeed)
	if err != nil {
		panic(err)
	}

	event := feed.Find("$.features[*]").FirstMatch(func(f *scalar.SV) bool {
		return f.Get("id").Eq("ci40123456")
	}).Get("properties")

	fmt.Println(event.Get("place"))
	fmt.Println(event.At("cdi").Or("no reports")) // $event->{cdi} // "no reports"

	if v, ok := event.GetOK("alert"); ok && !v.Defined() {
		fmt.Println("alert: null") // present, but null
	}
	if _, ok := event.GetOK("felt"); !ok {
		fmt.Println("felt: missing") // left out entirely
	}
	event.Get("felt").OrSet(0) // $event->{felt} //= 0
	fmt.Println(event.Get("felt"))

	// Output:
	// 12 km NNE of Ridgecrest, CA
	// no reports
	// alert: null
	// felt: missing
	// 0
}

func Example_readEventPlainGo() {
	var feed map[string]any // a struct can't tell a missing field from a null one
	if err := json.Unmarshal(quakeFeed, &feed); err != nil {
		panic(err)
	}

	event := map[string]any{}
	features, _ := feed["features"].([]any)
	for _, f := range features {
		f, _ := f.(map[string]any)
		if id, _ := f["id"].(string); id == "ci40123456" {
			if props, ok := f["properties"].(map[string]any); ok {
				event = props
			}
			break
		}
	}

	place, _ := event["place"].(string)
	fmt.Println(place)
	cdi, ok := event["cdi"]
	if !ok || cdi == nil {
		cdi = "no reports"
	}
	fmt.Println(cdi)

	if v, ok := event["alert"]; ok && v == nil {
		fmt.Println("alert: null")
	}
	if _, ok := event["felt"]; !ok {
		fmt.Println("felt: missing")
	}
	if event["felt"] == nil {
		event["felt"] = 0
	}
	fmt.Println(event["felt"])

	// Output:
	// 12 km NNE of Ridgecrest, CA
	// no reports
	// alert: null
	// felt: missing
	// 0
}

func Example_cleanStations() {
	config, err := scalar.FromJSON([]byte(`{"stations": [" PAS ", "GSC", "pas", "ISA "]}`))
	if err != nil {
		panic(err)
	}
	stations := config.Get("stations").AV()

	for _, s := range stations.All() {
		s.Set(s.Trim()) // changes the stations inside config
	}

	// grep { !$seen{fc $_}++ } @stations
	seen := scalar.Hash()
	unique := stations.Grep(func(s *scalar.SV) bool {
		return seen.Get(s.Fc().Str()).Inc().Int() == 1
	})
	fmt.Println(unique.Join(", "))

	// Output:
	// PAS, GSC, ISA
}

func Example_cleanStationsPlainGo() {
	var config struct {
		Stations []string `json:"stations"`
	}
	if err := json.Unmarshal([]byte(`{"stations": [" PAS ", "GSC", "pas", "ISA "]}`), &config); err != nil {
		panic(err)
	}

	fold := cases.Fold() // as Perl's fc; strings.ToLower misses some letters
	seen := map[string]bool{}
	var unique []string
	for i, s := range config.Stations {
		s = strings.TrimSpace(s)
		config.Stations[i] = s
		if key := fold.String(s); !seen[key] {
			seen[key] = true
			unique = append(unique, s)
		}
	}
	fmt.Println(strings.Join(unique, ", "))

	// Output:
	// PAS, GSC, ISA
}

// figure is compiled once, when the program starts, as with Perl's qr//.
var figure = scalar.MustCompile(`(\d[\d,]*)`)

func Example_figuresFromText() {
	text := scalar.Value("Felt by 12,431 people in 1,204 ZIP codes")

	figures := scalar.Array()
	for m := range text.Matches(figure) { // while ($text =~ /$figure/g)
		figures.Push(m.Get(1).S(`,`, "", "gr")) // "12,431" → "12431"
	}
	fmt.Println(figures.MinNum(), figures.MaxNum())

	// Output:
	// 1204 12431
}

// figureRE is compiled once, when the program starts.
var figureRE = regexp.MustCompile(`\d[\d,]*`)

func Example_figuresFromTextPlainGo() {
	text := "Felt by 12,431 people in 1,204 ZIP codes"

	var figures []int
	for _, m := range figureRE.FindAllString(text, -1) {
		n, err := strconv.Atoi(strings.ReplaceAll(m, ",", "")) // "12,431" → 12431
		if err != nil {
			panic(err)
		}
		figures = append(figures, n)
	}
	fmt.Println(slices.Min(figures), slices.Max(figures))

	// Output:
	// 1204 12431
}

// place runs on Linear, Go's own engine, compiled once.
var place = scalar.MustCompileWith(scalar.Linear, `(\d+) km (\w+) of ([^,]+), (.+)$`)

func Example_fasterRegex() {
	p := scalar.Value("4 km NW of Ōtaki, New Zealand").M(place)
	fmt.Println(p.Get(1), p.Get(2), p.Get(3))

	defer func(e scalar.Engine, w func(string)) { scalar.DefaultEngine, scalar.RegexWarn = e, w }(scalar.DefaultEngine, scalar.RegexWarn) // test only

	scalar.RegexWarn = nil // test only

	scalar.DefaultEngine = scalar.Linear                                    // or every pattern, from here on
	fmt.Println(scalar.Value("M4.8, depth 10 km").M(`(?<=M)\d\.\d`).Get(0)) // lookbehind: runs on Backtracking

	// Output:
	// 4 NW Ōtaki
	// 4.8
}

// placeRE is compiled once, when the program starts.
var placeRE = regexp.MustCompile(`(\d+) km (\w+) of ([^,]+), (.+)$`)

func Example_fasterRegexPlainGo() {
	if p := placeRE.FindStringSubmatch("4 km NW of Ōtaki, New Zealand"); p != nil {
		fmt.Println(p[1], p[2], p[3])
	}

	// Go's regexp has no lookbehind, so capture what follows the M instead.
	if m := regexp.MustCompile(`M(\d\.\d)`).FindStringSubmatch("M4.8, depth 10 km"); m != nil {
		fmt.Println(m[1])
	}

	// Output:
	// 4 NW Ōtaki
	// 4.8
}

func Example_countRegions() {
	days := [][]string{
		{"California", "Alaska"},
		{"alaska", "Nevada"},
		{"ALASKA", "California", "Hawaii"},
	}

	var counts scalar.Bag // the zero value is an empty bag
	for _, regions := range days {
		for _, region := range scalar.Array(regions).All() {
			counts.Insert(region.Fc()) // $counts{fc $region}++
		}
	}

	for _, region := range counts.MostCommon(3).All() { // ties keep the order they were first seen in
		fmt.Println(region, counts.Count(region))
	}

	// Output:
	// alaska 3
	// california 2
	// nevada 1
}

func Example_countRegionsPlainGo() {
	days := [][]string{
		{"California", "Alaska"},
		{"alaska", "Nevada"},
		{"ALASKA", "California", "Hawaii"},
	}

	fold := cases.Fold()
	counts := map[string]int{}
	var order []string // a map has no order, so keep the order first seen
	for _, regions := range days {
		for _, region := range regions {
			key := fold.String(region)
			if counts[key] == 0 {
				order = append(order, key)
			}
			counts[key]++
		}
	}

	slices.SortStableFunc(order, func(a, b string) int { return counts[b] - counts[a] }) // most common first
	for _, region := range order[:min(3, len(order))] {
		fmt.Println(region, counts[region])
	}

	// Output:
	// alaska 3
	// california 2
	// nevada 1
}

func Example_buildRecords() {
	defaults := scalar.Hash("status", "automatic", "tsunami", false)

	rows := scalar.Array()
	for _, id := range []string{"ci40123456", "nc75012345"} {
		row := scalar.Hash(defaults, "id", id) // %row = (%defaults, id => $id)
		rows.Push(scalar.Value(row))           // push @rows, \%row
	}
	rows.Get(1).Get("status").Set("reviewed") // changes only the second row

	out, err := json.Marshal(rows)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Output:
	// [{"id":"ci40123456","status":"automatic","tsunami":false},{"id":"nc75012345","status":"reviewed","tsunami":false}]
}

func Example_buildRecordsPlainGo() {
	type Row struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Tsunami bool   `json:"tsunami"`
	}
	defaults := Row{Status: "automatic"}

	var rows []Row
	for _, id := range []string{"ci40123456", "nc75012345"} {
		row := defaults // a struct is copied by value
		row.ID = id
		rows = append(rows, row)
	}
	rows[1].Status = "reviewed"

	out, err := json.Marshal(rows)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Output:
	// [{"id":"ci40123456","status":"automatic","tsunami":false},{"id":"nc75012345","status":"reviewed","tsunami":false}]
}

func Example_packSamples() {
	samples := []float64{0.5, -0.25, 0.125} // ground velocity, from a seismometer

	blob := scalar.Pack("f<*", samples) // pack("f<*", @samples): little-endian float32s
	fmt.Println(blob.Length())          // 4 bytes each
	fmt.Printf("%x\n", blob)            // in hex, for a log

	back := scalar.Unpack("f<*", blob.Bytes()).NumSlice() // a []float64 again
	fmt.Println(back)

	// Output:
	// 12
	// 0000003f000080be0000003e
	// [0.5 -0.25 0.125]
}

func Example_packSamplesPlainGo() {
	samples := []float64{0.5, -0.25, 0.125}

	var blob []byte
	for _, s := range samples {
		blob = binary.LittleEndian.AppendUint32(blob, math.Float32bits(float32(s)))
	}
	fmt.Println(len(blob))
	fmt.Printf("%x\n", blob)

	back := make([]float64, len(blob)/4)
	for i := range back {
		back[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(blob[4*i:])))
	}
	fmt.Println(back)

	// Output:
	// 12
	// 0000003f000080be0000003e
	// [0.5 -0.25 0.125]
}
