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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nukopian/scalar"
	"go.yaml.in/yaml/v3"
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

// stationsYAML is two YAML documents, one for each seismic network.
var stationsYAML = []byte(`
network: ci
stations:
  - {code: PAS, elevation: 295}
  - {code: GSC}
---
network: nc
stations:
  - {code: CMB}
`)

func Example_readYAML() {
	docs, err := scalar.FromYAMLAll(stationsYAML) // FromYAML reads just one
	if err != nil {
		panic(err)
	}
	for _, d := range docs.All() {
		fmt.Println(d.At("network"), d.Find("$.stations[*].code").Join(","), d.At("stations[0].elevation").Or("-"))
	}

	// Output:
	// ci PAS,GSC 295
	// nc CMB -
}

func Example_readYAMLPlainGo() {
	dec := yaml.NewDecoder(bytes.NewReader(stationsYAML))
	for {
		var d struct {
			Network  string `yaml:"network"`
			Stations []struct {
				Code      string `yaml:"code"`
				Elevation *int   `yaml:"elevation"`
			} `yaml:"stations"`
		}
		if err := dec.Decode(&d); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			panic(err)
		}

		var codes []string
		for _, s := range d.Stations {
			codes = append(codes, s.Code)
		}
		elevation := "-"
		if len(d.Stations) > 0 && d.Stations[0].Elevation != nil {
			elevation = strconv.Itoa(*d.Stations[0].Elevation)
		}
		fmt.Println(d.Network, strings.Join(codes, ","), elevation)
	}

	// Output:
	// ci PAS,GSC 295
	// nc CMB -
}

// weekFeed is four quakes from the USGS feed: one with no place, one with
// a null felt count, and one with none at all.
var weekFeed = []byte(`{"type": "FeatureCollection", "features": [
  {"id": "ci40123456", "properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "net": "ci", "felt": 1204},
   "geometry": {"coordinates": [-117.6, 35.7, 8.2]}},
  {"id": "nc75012345", "properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "net": "nc", "felt": null},
   "geometry": {"coordinates": [-122.8, 38.8, 1.9]}},
  {"id": "us7000n7v2", "properties": {"mag": 4.7, "place": null, "net": "us"},
   "geometry": {"coordinates": [-125.9, 40.3, 10.0]}},
  {"id": "us7000n8x4", "properties": {"mag": 4.9, "place": "Mid-Atlantic Ridge", "net": "us", "felt": 3},
   "geometry": {"coordinates": [-29.4, 0.9, 10.0]}}
]}`)

func Example_findAnywhere() {
	feed, err := scalar.FromJSON(weekFeed)
	if err != nil {
		panic(err)
	}

	mags := feed.Find("$..mag") // every "mag", at any depth
	fmt.Println(mags.Length(), mags.MaxNum())
	fmt.Println(feed.Find("$..net").UniqueStr().Join(","))

	// Output:
	// 4 5.1
	// ci,nc,us
}

func Example_findAnywherePlainGo() {
	var feed any
	if err := json.Unmarshal(weekFeed, &feed); err != nil {
		panic(err)
	}

	// find collects the values under key, at any depth, taking a map's
	// keys in sorted order, as the JSON's order is lost.
	var find func(v any, key string) []any
	find = func(v any, key string) []any {
		var found []any
		switch v := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				if k == key {
					found = append(found, v[k])
				}
				found = append(found, find(v[k], key)...)
			}
		case []any:
			for _, e := range v {
				found = append(found, find(e, key)...)
			}
		}
		return found
	}

	mags := find(feed, "mag")
	most := math.Inf(-1)
	for _, m := range mags {
		if f, ok := m.(float64); ok {
			most = max(most, f)
		}
	}
	fmt.Println(len(mags), most)

	var nets []string
	for _, n := range find(feed, "net") {
		if s, ok := n.(string); ok && !slices.Contains(nets, s) {
			nets = append(nets, s)
		}
	}
	fmt.Println(strings.Join(nets, ","))

	// Output:
	// 4 5.1
	// ci,nc,us
}

func Example_filterQuakes() {
	feed, err := scalar.FromJSON(weekFeed)
	if err != nil {
		panic(err)
	}

	strong := feed.Find("$.features[?@.properties.mag >= 4.5].properties.place")
	for _, place := range strong.All() {
		fmt.Println(place.Or("(no place)"))
	}

	// Output:
	// 12 km NNE of Ridgecrest, CA
	// (no place)
	// Mid-Atlantic Ridge
}

func Example_filterQuakesPlainGo() {
	var feed struct {
		Features []struct {
			Properties struct {
				Mag   float64 `json:"mag"`
				Place *string `json:"place"` // null is nil
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(weekFeed, &feed); err != nil {
		panic(err)
	}

	for _, f := range feed.Features {
		if p := f.Properties; p.Mag >= 4.5 {
			if p.Place != nil {
				fmt.Println(*p.Place)
			} else {
				fmt.Println("(no place)")
			}
		}
	}

	// Output:
	// 12 km NNE of Ridgecrest, CA
	// (no place)
	// Mid-Atlantic Ridge
}

func Example_routeFromData() {
	feed, err := scalar.FromJSON(weekFeed)
	if err != nil {
		panic(err)
	}

	columns := [][]any{ // the fields to print, as from a config
		{"id"},
		{"properties", "mag"},
		{"geometry", "coordinates", 2}, // depth, in km
	}
	for i := range 2 {
		var row []string
		for _, route := range columns {
			row = append(row, feed.Get("features", i).Get(route...).Str())
		}
		fmt.Println(strings.Join(row, "\t"))
	}

	// Output:
	// ci40123456	5.1	8.2
	// nc75012345	2.3	1.9
}

func Example_routeFromDataPlainGo() {
	var feed any
	if err := json.Unmarshal(weekFeed, &feed); err != nil {
		panic(err)
	}

	// get follows route, a key for a map and an index for a slice, and
	// returns nil if a step isn't there.
	get := func(v any, route ...any) any {
		for _, step := range route {
			switch s := step.(type) {
			case string:
				m, _ := v.(map[string]any)
				v = m[s]
			case int:
				a, _ := v.([]any)
				if s < 0 || s >= len(a) {
					return nil
				}
				v = a[s]
			}
		}
		return v
	}

	columns := [][]any{
		{"id"},
		{"properties", "mag"},
		{"geometry", "coordinates", 2},
	}
	for i := range 2 {
		var row []string
		for _, route := range columns {
			v := get(feed, append([]any{"features", i}, route...)...)
			if v == nil {
				v = ""
			}
			row = append(row, fmt.Sprint(v))
		}
		fmt.Println(strings.Join(row, "\t"))
	}

	// Output:
	// ci40123456	5.1	8.2
	// nc75012345	2.3	1.9
}

func Example_defaultsAndSteps() {
	feed, err := scalar.FromJSON(weekFeed)
	if err != nil {
		panic(err)
	}

	positive := func(v *scalar.SV) bool { return v.Num() > 0 }
	reports := func(v *scalar.SV) any { return v.Str() + " felt it" }

	for _, q := range feed.Find("$.features[*].properties").All() {
		// "12 km NNE of Ridgecrest, CA" → "Ridgecrest, CA"; with no "of", the
		// place itself; with no place at all, a default.
		place := q.Get("place")
		near := place.M(`of (.+)$`).Get(1).Or(place).Or("an unknown place")

		// A count, if there's one above zero, as text; or a default.
		felt := q.Get("felt").When(positive).Then(reports).Or("no reports")

		fmt.Printf("%s: %s\n", near, felt)
	}

	// Output:
	// Ridgecrest, CA: 1204 felt it
	// Cobb, CA: no reports
	// an unknown place: no reports
	// Mid-Atlantic Ridge: 3 felt it
}

// nearRE finds the place a quake was near.
var nearRE = regexp.MustCompile(`of (.+)$`)

func Example_defaultsAndStepsPlainGo() {
	var feed struct {
		Features []struct {
			Properties struct {
				Place *string `json:"place"`
				Felt  *int    `json:"felt"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(weekFeed, &feed); err != nil {
		panic(err)
	}

	for _, f := range feed.Features {
		p := f.Properties
		near := "an unknown place"
		if p.Place != nil {
			near = *p.Place
			if m := nearRE.FindStringSubmatch(near); m != nil {
				near = m[1]
			}
		}

		felt := "no reports"
		if p.Felt != nil && *p.Felt > 0 {
			felt = strconv.Itoa(*p.Felt) + " felt it"
		}

		fmt.Printf("%s: %s\n", near, felt)
	}

	// Output:
	// Ridgecrest, CA: 1204 felt it
	// Cobb, CA: no reports
	// an unknown place: no reports
	// Mid-Atlantic Ridge: 3 felt it
}

func Example_readWithoutChanging() {
	event, err := scalar.FromJSON([]byte(`{"properties": {"mag": 4.6}}`))
	if err != nil {
		panic(err)
	}

	_, ok := event.GetOK("properties", "products", "shakemap") // creates nothing
	fmt.Println(ok)

	event.Autovivify(false) // nor, from here on, does reading through what's missing
	fmt.Println(event.At("properties.products.shakemap").Defined())

	out, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out)) // as it was read

	// Output:
	// false
	// false
	// {"properties":{"mag":4.6}}
}

func Example_readWithoutChangingPlainGo() {
	var event map[string]any
	if err := json.Unmarshal([]byte(`{"properties": {"mag": 4.6}}`), &event); err != nil {
		panic(err)
	}

	props, _ := event["properties"].(map[string]any)
	products, _ := props["products"].(map[string]any) // a nil map: reading it creates nothing
	_, ok := products["shakemap"]
	fmt.Println(ok)
	fmt.Println(products["shakemap"] != nil)

	out, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Output:
	// false
	// false
	// {"properties":{"mag":4.6}}
}

func Example_numbersAsStrings() {
	data := []byte(`[{"mag": 4.6, "depth": "8.2"}, {"mag": "2.3", "depth": 1.9}]`) // quoted, some of the time
	feed, err := scalar.FromJSON(data)
	if err != nil {
		panic(err)
	}

	type Quake struct {
		Mag   float64 `json:"mag"`
		Depth float64 `json:"depth"`
	}
	quakes, err := scalar.To[[]Quake](feed) // "8.2" fills a float64, as Perl would read it
	if err != nil {
		panic(err)
	}
	fmt.Println(quakes)

	// Output:
	// [{4.6 8.2} {2.3 1.9}]
}

// number reads a JSON number, or a string holding one.
type number float64

func (n *number) UnmarshalJSON(b []byte) error {
	f, err := strconv.ParseFloat(strings.Trim(string(b), `"`), 64)
	*n = number(f)
	return err
}

func Example_numbersAsStringsPlainGo() {
	type Quake struct {
		Mag   number `json:"mag"`
		Depth number `json:"depth"`
	}
	data := []byte(`[{"mag": 4.6, "depth": "8.2"}, {"mag": "2.3", "depth": 1.9}]`)
	var quakes []Quake
	if err := json.Unmarshal(data, &quakes); err != nil {
		panic(err)
	}
	fmt.Println(quakes)

	// Output:
	// [{4.6 8.2} {2.3 1.9}]
}

func Example_whatDoesntFit() {
	doc, err := scalar.FromJSON([]byte(`{"id": "ci40123456", "depth": {"km": "unknown"}}`))
	if err != nil {
		panic(err)
	}

	type Event struct {
		ID    string `json:"id"`
		Depth struct {
			Km float64 `json:"km"`
		} `json:"depth"`
	}
	_, err = scalar.To[Event](doc)
	var de *scalar.DecodeError
	if errors.As(err, &de) {
		fmt.Println(de.Path, doc.At(de.Path)) // where, and what's there
	}

	// Output:
	// depth.km unknown
}

func Example_whatDoesntFitPlainGo() {
	data := []byte(`{"id": "ci40123456", "depth": {"km": "unknown"}}`)

	var event struct {
		ID    string `json:"id"`
		Depth struct {
			Km float64 `json:"km"`
		} `json:"depth"`
	}
	err := json.Unmarshal(data, &event)
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		// The error names the field, but not what was there: read it again.
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			panic(err)
		}
		for _, key := range strings.Split(te.Field, ".") {
			m, _ := raw.(map[string]any)
			raw = m[key]
		}
		fmt.Println(te.Field, raw)
	}

	// Output:
	// depth.km unknown
}

func Example_loosePart() {
	var event struct {
		ID       string     `json:"id"`
		Products *scalar.SV `json:"products"` // its shape differs from one event to the next
	}
	data := []byte(`{"id": "ci40123456", "products": {"shakemap": [{"status": "reviewed"}]}}`)
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	shakemap := event.Products.At("shakemap[0].status").Or("none")
	dyfi := event.Products.At("dyfi[0].status").Or("none") // this event has none
	fmt.Println(event.ID, shakemap, dyfi)

	// Output:
	// ci40123456 reviewed none
}

func Example_loosePartPlainGo() {
	var event struct {
		ID       string          `json:"id"`
		Products json.RawMessage `json:"products"` // left as JSON, to read when its shape is known
	}
	data := []byte(`{"id": "ci40123456", "products": {"shakemap": [{"status": "reviewed"}]}}`)
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	var products map[string][]struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(event.Products, &products); err != nil {
		panic(err)
	}
	status := func(kind string) string {
		if p := products[kind]; len(p) > 0 {
			return p[0].Status
		}
		return "none"
	}
	fmt.Println(event.ID, status("shakemap"), status("dyfi"))

	// Output:
	// ci40123456 reviewed none
}
