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
		{"properties": {"mag": 4.9, "place": "Mid-Atlantic Ridge", "time": 1760786100000},
		 "geometry": {"coordinates": [-29.4, 0.9, 10.0]}},
		{"properties": {"mag": null, "place": "Alaska Peninsula", "time": 1760787000000},
		 "geometry": {"coordinates": [-157.1, 56.2, 35.4]}}
	]}`)
	feed, err := scalar.FromJSON(body) // a USGS earthquake feed, in GeoJSON
	if err != nil {
		panic(err)
	}

	// Search the whole tree with JSONPath: the strongest magnitude anywhere.
	strongest := feed.Find("$..mag").MaxNum().Num()

	// Filter with JSONPath, then reshape each quake. Anything missing or null
	// is undef, not a panic, and Or supplies a default.
	big := scalar.Array()
	for _, q := range feed.Find("$.features[?@.properties.mag >= 4.5]").All() {
		// "12 km NNE of Ridgecrest, CA" → "Ridgecrest, CA"; with no "of", the
		// place itself; with no place at all, a default.
		place := q.At("properties.place")
		near := place.M(`of (.+)$`).Get(1).Or(place).Or("an unknown place")

		big.Push(scalar.Value(scalar.Hash(
			"mag", q.At("properties.mag"),
			"near", near,
			"depth", q.At("geometry.coordinates[2]"), // [lon, lat, depth]
			"time", q.At("properties.time"),
		)))
	}

	// The shape is known: into ordinary, typed Go.
	type Quake struct {
		Mag   float64 `json:"mag"`
		Near  string  `json:"near"`
		Depth float64 `json:"depth"` // km
		Time  int64   `json:"time"`  // ms since 1970
	}
	quakes, err := scalar.To[[]Quake](big)
	if err != nil {
		panic(err)
	}

	// From here on, it's plain Go.
	fmt.Printf("Strongest: M%.1f\n", strongest)
	for _, q := range quakes {
		when := time.UnixMilli(q.Time).UTC().Format("2 Jan 15:04 GMT")
		fmt.Printf("M%.1f near %s, %.0f km deep, %s\n", q.Mag, q.Near, q.Depth, when)
	}
	// Output:
	// Strongest: M5.1
	// M5.1 near Ridgecrest, CA, 8 km deep, 18 Oct 10:00 GMT
	// M4.7 near an unknown place, 10 km deep, 18 Oct 11:00 GMT
	// M4.9 near Mid-Atlantic Ridge, 10 km deep, 18 Oct 11:15 GMT
}

const quakeFeed = `{
  "type": "FeatureCollection",
  "metadata": {"title": "USGS All Earthquakes, Past Day", "count": 2},
  "features": [
    {"type": "Feature", "id": "nc75012345",
     "properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "felt": null, "alert": null}},
    {"type": "Feature", "id": "ci40123456",
     "properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "alert": null}}
  ]
}`

// Reading one event from a feed: find it, then read fields with defaults,
// telling missing fields from null ones.
func Example_readEvent() {
	feed, err := scalar.FromJSON([]byte(quakeFeed))
	if err != nil {
		panic(err)
	}

	event := feed.Find("$.features[*]").FirstMatch(func(f *scalar.SV) bool {
		return f.Path("id").Eq("ci40123456")
	}).Path("properties")

	fmt.Println(event.Path("place"))
	fmt.Println(event.At("cdi").Or("no reports")) // $event->{cdi} // "no reports"

	if v, ok := event.PathOK("alert"); ok && !v.Defined() {
		fmt.Println("alert: null")
	}
	if _, ok := event.PathOK("felt"); !ok {
		fmt.Println("felt: missing")
	}
	event.HV().Get("felt").OrSet(0) // $event->{felt} //= 0
	fmt.Println(event.Path("felt"))

	// Output:
	// 12 km NNE of Ridgecrest, CA
	// no reports
	// alert: null
	// felt: missing
	// 0
}

// Cleaning a list of station codes typed into a config by hand: trim them
// in place, then drop duplicates regardless of case, keeping the first
// spelling of each.
func Example_cleanStations() {
	config, _ := scalar.FromJSON([]byte(`{"stations": [" PAS ", "GSC", "pas", "ISA "]}`))
	stations := config.Find("$..stations").Get(0).AV()

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

// figure is compiled once, when the program starts, as with qr//.
var figure = scalar.MustCompile(`(\d[\d,]*)`)

// Pulling figures out of text.
func Example_figuresFromText() {
	text := scalar.Value("Felt by 12,431 people in 1,204 ZIP codes")

	figures := scalar.Array()
	for m := range text.Matches(figure) {
		figures.Push(m.Get(1).S(`,`, "", "gr")) // "12,431" → "12431"
	}
	fmt.Println(figures.MinNum(), figures.MaxNum())

	// Output:
	// 1204 12431
}

// Counting quakes by region over several days, ignoring case, and listing
// the most common first.
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

// Building records from defaults and writing them out as JSON.
func Example_buildRecords() {
	defaults := scalar.Hash("status", "automatic", "tsunami", false)

	rows := scalar.Array()
	for _, id := range []string{"ci40123456", "nc75012345"} {
		row := scalar.Hash(defaults, "id", id) // %row = (%defaults, id => $id)
		rows.Push(scalar.Value(row))           // push @rows, \%row
	}
	rows.Get(1).Path("status").Set("reviewed") // changes only the second row

	out, _ := json.Marshal(rows)
	fmt.Println(string(out))

	// Output:
	// [{"id":"ci40123456","status":"automatic","tsunami":false},{"id":"nc75012345","status":"reviewed","tsunami":false}]
}

// Storing waveform samples as bytes, 4 for each float32, and reading them
// back.
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
