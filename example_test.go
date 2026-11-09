package scalar_test

// This example is the opening example in README.md. Go runs it as a test
// and checks its output, so if it stops working, go test says so. Keep the
// README in step when changing it. The how-to guide's recipes are in
// example_howto_test.go.

import (
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
