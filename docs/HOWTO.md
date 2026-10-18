# scalar: how-to guide

How to get things done with [scalar](../README.md): getting started, then
recipes for common tasks, then two jobs done in plain Go and with scalar,
side by side. For how each feature behaves, see the
[reference](REFERENCE.md); for one example of every function and method,
see the [examples](EXAMPLES.md).

- [Quick start](#quick-start)
- [Recipes](#recipes)
  - [Reading one event](#reading-one-event)
  - [Cleaning a list of station codes](#cleaning-a-list-of-station-codes)
  - [Pulling figures out of text](#pulling-figures-out-of-text)
  - [Counting quakes by region](#counting-quakes-by-region)
  - [Building records and writing JSON](#building-records-and-writing-json)
  - [Storing waveform samples as bytes](#storing-waveform-samples-as-bytes)
- [Compared with plain Go](#compared-with-plain-go)

## Quick start

```go
import "github.com/nukopian/scalar"

feed, err := scalar.FromJSON(body)   // the USGS earthquake feed
if err != nil {
	return err
}

place := feed.Path("features", 0, "properties", "place").Str() // $feed->{features}[0]{properties}{place}
mag := feed.At("features[0].properties.mag").Num()            // the same idea, as one string

meta, err := scalar.To[Metadata](feed.At("metadata")) // into your own struct, checked
```

If any step along the path is missing, you get undef back instead of a panic.
`Str()` on undef is `""` and `Num()` is `0`. `Path` takes keys and indices
as separate arguments; `At` takes the route as one string, in dot, bracket
or Perl's brace form (see [A path as one string](REFERENCE.md#a-path-as-one-string-at)).

## Recipes

Six short examples, each combining several parts of the package, on data
like the USGS earthquake feed. The [reference](REFERENCE.md) explains each
part in detail.

These recipes are also runnable examples in the repository's
`example_test.go`, so `go test` checks they still work. Keep the two in
step when changing them.

### Reading one event

Find an event in the feed, then read its fields with defaults, telling a
missing field from a null one.

```go
feed, err := scalar.FromJSON(body)
if err != nil {
	return err
}

event := feed.Find("$.features[*]").FirstMatch(func(f *scalar.SV) bool {
	return f.Path("id").Eq("ci40123456")
}).Path("properties")

fmt.Println(event.Path("place"))                  // 12 km NNE of Ridgecrest, CA
fmt.Println(event.At("cdi").Or("no reports"))     // $event->{cdi} // "no reports"

if v, ok := event.PathOK("alert"); ok && !v.Defined() {
	fmt.Println("alert: null")                      // present, but null
}
if _, ok := event.PathOK("felt"); !ok {
	fmt.Println("felt: missing")                    // left out entirely
}
event.HV().Get("felt").OrSet(0)                   // $event->{felt} //= 0
```

### Cleaning a list of station codes

Station codes typed into a config by hand: trim each where it's stored,
then drop duplicates regardless of case, keeping the first spelling of
each.

```go
stations := config.Find("$..stations").Get(0).AV()   // [" PAS ", "GSC", "pas", "ISA "]

for _, s := range stations.All() {
	s.Set(s.Trim())                                    // changes the stations inside config
}

// grep { !$seen{fc $_}++ } @stations
seen := scalar.Hash()
unique := stations.Grep(func(s *scalar.SV) bool {
	return seen.Get(s.Fc().Str()).Inc().Int() == 1
})
fmt.Println(unique.Join(", "))                       // PAS, GSC, ISA
```

### Pulling figures out of text

The pattern is compiled once, when the program starts, as with Perl's
`qr//`.

```go
var figure = scalar.MustCompile(`(\d[\d,]*)`)

text := scalar.Value("Felt by 12,431 people in 1,204 ZIP codes")

figures := scalar.Array()
for m := range text.Matches(figure) {           // while ($text =~ /$figure/g)
	figures.Push(m.Get(1).S(`,`, "", "gr"))     // "12,431" → "12431"
}
fmt.Println(figures.MinNum(), figures.MaxNum())  // 1204 12431
```

### Counting quakes by region

Count each region ignoring case in a `Bag`, then list the most common
first.

```go
days := [][]string{
	{"California", "Alaska"},
	{"alaska", "Nevada"},
	{"ALASKA", "California", "Hawaii"},
}

var counts scalar.Bag                                  // the zero value is an empty bag
for _, regions := range days {
	for _, region := range scalar.Array(regions).All() {
		counts.Insert(region.Fc())                       // $counts{fc $region}++
	}
}

for _, region := range counts.MostCommon(3).All() {    // ties keep the order they were first seen in
	fmt.Println(region, counts.Count(region))            // alaska 3, california 2, nevada 1
}
```

The same with a hash, `Inc` and a sort is in [Sorting](REFERENCE.md#sorting).

### Building records and writing JSON

Start each record from shared defaults, collect them in an array, and encode
the result.

```go
defaults := scalar.Hash("status", "automatic", "tsunami", false)

rows := scalar.Array()
for _, id := range []string{"ci40123456", "nc75012345"} {
	row := scalar.Hash(defaults, "id", id)     // %row = (%defaults, id => $id)
	rows.Push(scalar.Value(row))               // push @rows, \%row
}
rows.Get(1).Path("status").Set("reviewed")   // changes only the second row

out, _ := json.Marshal(rows)
// [{"id":"ci40123456","status":"automatic","tsunami":false},
//  {"id":"nc75012345","status":"reviewed","tsunami":false}]
```

Each row is a separate copy of `defaults`, so changing one row changes neither
the others nor `defaults`.

### Storing waveform samples as bytes

Pack a seismometer's samples into 4 bytes per float32, for a blob column or
a cache, and unpack them again.

```go
samples := []float64{0.5, -0.25, 0.125}              // ground velocity, from a seismometer

blob := scalar.Pack("f<*", samples)                  // pack("f<*", @samples): little-endian float32s
blob.Length()                                        // 12: 4 bytes each
fmt.Printf("%x\n", blob)                             // 0000003f000080be0000003e, for a log

back := scalar.Unpack("f<*", blob.Bytes()).NumSlice()   // a []float64 again
```

`blob` is a [byte string](REFERENCE.md#byte-strings), and `blob.Bytes()` is the Go
`[]byte` to store.

## Compared with plain Go

Two jobs are awkward in plain Go.

**1. Reading loosely shaped JSON.** Feeds and third-party APIs return JSON
whose shape varies: fields go missing or turn up null, nesting varies, and
numbers turn up as strings. In plain Go, reading one nested value means a
chain of type assertions, and each one needs checking:

```go
var feed map[string]any
if err := json.Unmarshal(body, &feed); err != nil { ... }
features, ok := feed["features"].([]any)
if !ok || len(features) == 0 { ... }
first, ok := features[0].(map[string]any)
if !ok { ... }
props, ok := first["properties"].(map[string]any)
if !ok { ... }
place, ok := props["place"].(string)
if !ok { ... }   // and a null place lands here too
```

Skip a check and the program panics on the first unexpected response. With
`scalar`, the same read is:

```go
feed, err := scalar.FromJSON(body)
if err != nil { ... }
place := feed.At("features[0].properties.place").Str()   // or feed.Path("features", 0, "properties", "place")
```

If anything along the path is missing, null or the wrong type, `place` is
just `""`.

**2. Heavy text processing.** Pulling facts out of free text, such as place
descriptions and bulletins, means a lot of regex work. Go's standard
`regexp` package deliberately leaves out lookahead, lookbehind and
backreferences. `scalar` uses an engine that has them, with a timeout so a
bad pattern can't hang the program. It also wraps them in short helpers
modelled on Perl's `s///`, `split` and `tr///`.
