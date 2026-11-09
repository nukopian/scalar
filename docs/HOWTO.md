# scalar: how-to guide

How to get things done with [scalar](../README.md): getting started, then
recipes for common tasks, each with the same job done in plain Go beside
it, then when each is the better choice. For how each feature behaves,
see the [reference](REFERENCE.md); for one example of every function and
method, see the [examples](EXAMPLES.md). Coming from Perl? The
[phrasebook](PERL.md) shows each task in Perl, with scalar and in plain Go.

Every recipe here is generated from an example in the repository's
`example_howto_test.go`, and so is its plain Go, so `go test` checks that
both still work and print what this guide says they print.

- [Quick start](#quick-start)
- [Recipes](#recipes)
  - [Reading one event](#reading-one-event)
  - [Cleaning a list of station codes](#cleaning-a-list-of-station-codes)
  - [Pulling figures out of text](#pulling-figures-out-of-text)
  - [Making regex-heavy code faster](#making-regex-heavy-code-faster)
  - [Counting quakes by region](#counting-quakes-by-region)
  - [Building records and writing JSON](#building-records-and-writing-json)
  - [Storing waveform samples as bytes](#storing-waveform-samples-as-bytes)
- [Compared with plain Go](#compared-with-plain-go)

## Quick start

Read a feed, pick out two fields, then convert a part whose shape you
know into your own struct.

<!-- recipe:quickStart -->

```go
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
```

It prints:

```
5 km W of Cobb, CA 2.3
USGS All Earthquakes, Past Day 2
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Get`](EXAMPLES.md#svget), [`SV.Str`](EXAMPLES.md#svstr), [`SV.At`](EXAMPLES.md#svat), [`SV.Num`](EXAMPLES.md#svnum), [`To`](EXAMPLES.md#to).

<!-- /recipe -->

If any step along the path is missing, you get undef back instead of a
panic, though, as in Perl, a read creates the levels it goes through (see
[What you might assume](REFERENCE.md#what-you-might-assume)). `Str()` on
undef is `""` and `Num()` is `0`. `Get` takes keys and indices as
separate arguments, one key or index for each level; for several fields
at once, Perl's hash slice, use `HV().Values`. `At` takes the route as one
string, in dot, bracket or Perl's brace form (see
[A path as one string](REFERENCE.md#a-path-as-one-string-at)).

Where the shape is known, as here, the plain Go is as short and is
checked by the compiler, so prefer it; scalar earns its place where the
shape isn't known yet. In Perl: [a nested field, with a
default](PERL.md#a-nested-field-with-a-default).

## Recipes

Seven short examples, each combining several parts of the package, on data
like the USGS earthquake feed. The [reference](REFERENCE.md) explains each
part in detail.

### Reading one event

Find an event in the feed, then read its fields with defaults, telling a
missing field from a null one.

<!-- recipe:readEvent -->

```go
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
```

It prints:

```
12 km NNE of Ridgecrest, CA
no reports
alert: null
felt: missing
0
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Find`](EXAMPLES.md#svfind), [`AV.FirstMatch`](EXAMPLES.md#avfirstmatch), [`SV.Get`](EXAMPLES.md#svget), [`SV.Eq`](EXAMPLES.md#sveq), [`SV.At`](EXAMPLES.md#svat), [`SV.Or`](EXAMPLES.md#svor), [`SV.GetOK`](EXAMPLES.md#svgetok), [`SV.Defined`](EXAMPLES.md#svdefined), [`SV.OrSet`](EXAMPLES.md#svorset).

<!-- /recipe -->

The plain Go can't use a struct here, which would read a missing field
and a null one alike, so it reads into `map[string]any`, and each step
needs a type assertion. In Perl: [missing, null or a
value](PERL.md#missing-null-or-a-value) and [probe without
creating](PERL.md#probe-without-creating).

### Cleaning a list of station codes

Station codes typed into a config by hand: trim each where it's stored,
then drop duplicates regardless of case, keeping the first spelling of
each.

<!-- recipe:cleanStations -->

```go
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
```

It prints:

```
PAS, GSC, ISA
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Get`](EXAMPLES.md#svget), [`SV.AV`](EXAMPLES.md#svav), [`AV.All`](EXAMPLES.md#avall), [`SV.Set`](EXAMPLES.md#svset), [`SV.Trim`](EXAMPLES.md#svtrim), [`Hash`](EXAMPLES.md#hash), [`AV.Grep`](EXAMPLES.md#avgrep), [`HV.Get`](EXAMPLES.md#hvget), [`SV.Fc`](EXAMPLES.md#svfc), [`SV.Str`](EXAMPLES.md#svstr), [`SV.Inc`](EXAMPLES.md#svinc), [`SV.Int`](EXAMPLES.md#svint), [`AV.Join`](EXAMPLES.md#avjoin).

<!-- /recipe -->

`Fc` folds case as Perl's `fc` does, and the plain Go needs
`golang.org/x/text/cases` for the same: `strings.ToLower` misses letters
such as `ß`. In Perl: [split, trim and drop
duplicates](PERL.md#split-trim-and-drop-duplicates), and [a whole array
or hash](PERL.md#a-whole-array-or-hash) for `AV()`.

### Pulling figures out of text

The pattern is compiled once, when the program starts, as with Perl's
`qr//`.

<!-- recipe:figuresFromText -->

```go
// figure is compiled once, when the program starts, as with Perl's qr//.
var figure = scalar.MustCompile(`(\d[\d,]*)`)

text := scalar.Value("Felt by 12,431 people in 1,204 ZIP codes")

figures := scalar.Array()
for m := range text.Matches(figure) { // while ($text =~ /$figure/g)
	figures.Push(m.Get(1).S(`,`, "", "gr")) // "12,431" → "12431"
}
fmt.Println(figures.MinNum(), figures.MaxNum())
```

It prints:

```
1204 12431
```

In plain Go, printing the same:

```go
// figureRE is compiled once, when the program starts.
var figureRE = regexp.MustCompile(`\d[\d,]*`)

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
```

**Uses:** [`MustCompile`](EXAMPLES.md#mustcompile), [`Value`](EXAMPLES.md#value), [`Array`](EXAMPLES.md#array), [`SV.Matches`](EXAMPLES.md#svmatches), [`AV.Push`](EXAMPLES.md#avpush), [`AV.Get`](EXAMPLES.md#avget), [`SV.S`](EXAMPLES.md#svs), [`AV.MinNum`](EXAMPLES.md#avminnum), [`AV.MaxNum`](EXAMPLES.md#avmaxnum).

<!-- /recipe -->

In Perl: [every figure in a text](PERL.md#every-figure-in-a-text).

### Making regex-heavy code faster

`scalar.Linear` runs patterns on Go's own engine, which gave the same
answers 1.2 to 3.4 times as fast in scalar's benchmarks. Choose it for
the patterns that matter, compiled once, or for every pattern in the
program, at the start of `main`, with `scalar.DefaultEngine`:

<!-- recipe:fasterRegex -->

```go
// place runs on Linear, Go's own engine, compiled once.
var place = scalar.MustCompileWith(scalar.Linear, `(\d+) km (\w+) of ([^,]+), (.+)$`)

p := scalar.Value("4 km NW of Ōtaki, New Zealand").M(place)
fmt.Println(p.Get(1), p.Get(2), p.Get(3))

scalar.DefaultEngine = scalar.Linear                                    // or every pattern, from here on
fmt.Println(scalar.Value("M4.8, depth 10 km").M(`(?<=M)\d\.\d`).Get(0)) // lookbehind: runs on Backtracking
```

It prints:

```
4 NW Ōtaki
4.8
```

In plain Go, printing the same:

```go
// placeRE is compiled once, when the program starts.
var placeRE = regexp.MustCompile(`(\d+) km (\w+) of ([^,]+), (.+)$`)

if p := placeRE.FindStringSubmatch("4 km NW of Ōtaki, New Zealand"); p != nil {
	fmt.Println(p[1], p[2], p[3])
}

// Go's regexp has no lookbehind, so capture what follows the M instead.
if m := regexp.MustCompile(`M(\d\.\d)`).FindStringSubmatch("M4.8, depth 10 km"); m != nil {
	fmt.Println(m[1])
}
```

**Uses:** [`MustCompileWith`](EXAMPLES.md#mustcompilewith), [`Value`](EXAMPLES.md#value), [`SV.M`](EXAMPLES.md#svm), [`AV.Get`](EXAMPLES.md#avget).

<!-- /recipe -->

If `Linear` can't run a pattern, such as one with lookbehind,
`MustCompileWith` panics as the program starts, saying what it can't run.
With `DefaultEngine`, such a pattern runs on `Backtracking`, as before,
and scalar logs a warning, once for that pattern:

```
scalar: regex "(?<=M)\\d\\.\\d" uses lookbehind, which Linear can't run, so it runs on Backtracking
```

Either way, the answers are the ones `Backtracking` gives, even for text
beyond ASCII such as `Ōtaki`. [Choosing an
engine](REFERENCE.md#choosing-an-engine-backtracking-and-linear) says what
`Linear` can't run, and which matches it still hands to `Backtracking`.
Go's own `regexp`, in the plain Go, has no lookbehind, so the pattern
has to be written another way.

### Counting quakes by region

Count each region ignoring case in a `Bag`, then list the most common
first.

<!-- recipe:countRegions -->

```go
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
```

It prints:

```
alaska 3
california 2
nevada 1
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`Array`](EXAMPLES.md#array), [`AV.All`](EXAMPLES.md#avall), [`Bag.Insert`](EXAMPLES.md#baginsert), [`SV.Fc`](EXAMPLES.md#svfc), [`Bag.MostCommon`](EXAMPLES.md#bagmostcommon), [`Bag.Count`](EXAMPLES.md#bagcount).

<!-- /recipe -->

The same with a hash, `Inc` and a sort is in [Sorting](REFERENCE.md#sorting).
In Perl: [count by a key](PERL.md#count-by-a-key).

### Building records and writing JSON

Start each record from shared defaults, collect them in an array, and encode
the result.

<!-- recipe:buildRecords -->

```go
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
```

It prints:

```
[{"id":"ci40123456","status":"automatic","tsunami":false},{"id":"nc75012345","status":"reviewed","tsunami":false}]
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`Hash`](EXAMPLES.md#hash), [`Array`](EXAMPLES.md#array), [`AV.Push`](EXAMPLES.md#avpush), [`Value`](EXAMPLES.md#value), [`AV.Get`](EXAMPLES.md#avget), [`SV.Get`](EXAMPLES.md#svget), [`SV.Set`](EXAMPLES.md#svset).

<!-- /recipe -->

Each row is a separate copy of `defaults`, so changing one row changes
neither the others nor `defaults`. In the plain Go, a struct does the same
by being copied when it's assigned; with the fields known, it's the better
choice.

### Storing waveform samples as bytes

Pack a seismometer's samples into 4 bytes per float32, for a blob column or
a cache, and unpack them again.

<!-- recipe:packSamples -->

```go
samples := []float64{0.5, -0.25, 0.125} // ground velocity, from a seismometer

blob := scalar.Pack("f<*", samples) // pack("f<*", @samples): little-endian float32s
fmt.Println(blob.Length())          // 4 bytes each
fmt.Printf("%x\n", blob)            // in hex, for a log

back := scalar.Unpack("f<*", blob.Bytes()).NumSlice() // a []float64 again
fmt.Println(back)
```

It prints:

```
12
0000003f000080be0000003e
[0.5 -0.25 0.125]
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`Pack`](EXAMPLES.md#pack), [`SV.Length`](EXAMPLES.md#svlength), [`Unpack`](EXAMPLES.md#unpack), [`SV.Bytes`](EXAMPLES.md#svbytes), [`AV.NumSlice`](EXAMPLES.md#avnumslice).

<!-- /recipe -->

`blob` is a [byte string](REFERENCE.md#byte-strings), and `blob.Bytes()` is the Go
`[]byte` to store.

## Compared with plain Go

Two jobs are awkward in plain Go.

**1. Reading loosely shaped JSON.** Feeds and third-party APIs return JSON
whose shape varies: fields go missing or turn up null, nesting varies, and
numbers turn up as strings. Where the shape is known, a struct reads it
well, as the [quick start](#quick-start)'s plain Go shows. Where it
isn't, plain Go reads into `map[string]any`, and reading one nested value
means a chain of type assertions, each one needing a check, as in
[Reading one event](#reading-one-event). Skip a check and the program
panics on the first unexpected response. With scalar, the same read is a
path, and if anything along it is missing, null or the wrong type, the
answer is undef.

**2. Heavy text processing.** Pulling facts out of free text, such as place
descriptions and bulletins, means a lot of regex work. Go's standard
`regexp` package deliberately leaves out lookahead, lookbehind and
backreferences. `scalar` uses an engine that has them, with a timeout so a
bad pattern can't hang the program. It also wraps them in short helpers
modelled on Perl's `s///`, `split` and `tr///`. When the regex work is a
cost, `scalar.Linear` runs the same patterns on Go's engine and gives the
same answers faster (see [Making regex-heavy code faster](#making-regex-heavy-code-faster)).
