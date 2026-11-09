# scalar: how-to guide

How to get things done with [scalar](../README.md), in chapters that
follow a feed of unknown shape on its way to typed Go: reading it,
finding things in it, handling what's missing, and moving into structs,
then working with text, counting, writing out and making it fast. Each
recipe has the same job done in plain Go beside it, and each chapter
says when each is the better choice. For how each feature behaves, see
the [reference](REFERENCE.md); for one example of every function and
method, see the [examples](EXAMPLES.md). Coming from Perl? The
[phrasebook](PERL.md) shows each task in Perl, with scalar and in plain Go.

Every recipe here is generated from an example in the repository's
`example_howto_test.go`, and so is its plain Go, so `go test` checks that
both still work and print what this guide says they print.

- [Quick start](#quick-start)
- [Reading loose data](#reading-loose-data)
  - [YAML, one document or several](#yaml-one-document-or-several)
- [Finding things](#finding-things)
  - [Every value at any depth](#every-value-at-any-depth)
  - [Filtering with JSONPath](#filtering-with-jsonpath)
  - [A route built from data](#a-route-built-from-data)
- [When something's missing](#when-somethings-missing)
  - [Reading one event](#reading-one-event)
  - [Defaults, and steps that may come up empty](#defaults-and-steps-that-may-come-up-empty)
  - [Reading without changing the data](#reading-without-changing-the-data)
- [Into typed Go](#into-typed-go)
  - [Numbers that arrive as strings](#numbers-that-arrive-as-strings)
  - [Finding what doesn't fit](#finding-what-doesnt-fit)
  - [A part that stays loose](#a-part-that-stays-loose)
- [Working with text](#working-with-text)
  - [Cleaning a list of station codes](#cleaning-a-list-of-station-codes)
  - [Pulling figures out of text](#pulling-figures-out-of-text)
- [Counting and grouping](#counting-and-grouping)
  - [Counting quakes by region](#counting-quakes-by-region)
- [Writing it out](#writing-it-out)
  - [Building records and writing JSON](#building-records-and-writing-json)
  - [Storing waveform samples as bytes](#storing-waveform-samples-as-bytes)
- [Making it faster](#making-it-faster)
  - [Making regex-heavy code faster](#making-regex-heavy-code-faster)
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

## Reading loose data

`FromJSON` and `FromYAML` read any document into a scalar, whatever its
shape, and fail only on text that isn't JSON or YAML. Nothing about the
shape is promised until you ask for it, so a field that's missing, null
or of another type than you expected reads as undef, or as the value it
is, rather than failing the whole read. Plain Go's decoders fill a type
you declare first, which suits a document whose shape you know, and
reject one that doesn't fit it. The quick start reads JSON; the
reference's [JSON](REFERENCE.md#json) and [YAML](REFERENCE.md#yaml)
sections say what each reads as what.

### YAML, one document or several

A file of YAML documents, one for each seismic network, as a config or a
manifest often is.

<!-- recipe:readYAML -->

```go
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

docs, err := scalar.FromYAMLAll(stationsYAML) // FromYAML reads just one
if err != nil {
	panic(err)
}
for _, d := range docs.All() {
	fmt.Println(d.At("network"), d.Find("$.stations[*].code").Join(","), d.At("stations[0].elevation").Or("-"))
}
```

It prints:

```
ci PAS,GSC 295
nc CMB -
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromYAMLAll`](EXAMPLES.md#fromyamlall), [`AV.All`](EXAMPLES.md#avall), [`SV.At`](EXAMPLES.md#svat), [`SV.Find`](EXAMPLES.md#svfind), [`AV.Join`](EXAMPLES.md#avjoin), [`SV.Or`](EXAMPLES.md#svor).

<!-- /recipe -->

The plain Go declares the shape, and a pointer for the elevation that
may be left out; the scalar reads whatever is there.

## Finding things

Three ways to reach into a document: `Get` takes a route, one key or
index for each level, as Perl's `$x->{a}[0]` does; `At` takes the route as
one string; and `Find` takes a JSONPath query, which can search at any
depth and filter, and returns an array of every match. Plain Go walks a
struct with its fields, which is clearest when the shape is fixed, and
needs a function of its own to search or to follow a route it's given.
See the reference's [Reaching in as Perl
does](REFERENCE.md#reaching-in-as-perl-does), [A path as one
string](REFERENCE.md#a-path-as-one-string-at) and [Finding values with
JSONPath](REFERENCE.md#finding-values-with-jsonpath).

### Every value at any depth

`..` searches every level, for a field whose place varies, or to survey a
feed you haven't seen.

<!-- recipe:findAnywhere -->

```go
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

feed, err := scalar.FromJSON(weekFeed)
if err != nil {
	panic(err)
}

mags := feed.Find("$..mag") // every "mag", at any depth
fmt.Println(mags.Length(), mags.MaxNum())
fmt.Println(feed.Find("$..net").UniqueStr().Join(","))
```

It prints:

```
4 5.1
ci,nc,us
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Find`](EXAMPLES.md#svfind), [`AV.Length`](EXAMPLES.md#avlength), [`AV.MaxNum`](EXAMPLES.md#avmaxnum), [`AV.UniqueStr`](EXAMPLES.md#avuniquestr), [`AV.Join`](EXAMPLES.md#avjoin).

<!-- /recipe -->

JSONPath's results come in the document's order. The plain Go's
`map[string]any` keeps no order, so it sorts the keys, which happens to
give the same answer here.

### Filtering with JSONPath

A filter, `[?…]`, keeps what passes a test, and the path after it picks
out what you want from each.

<!-- recipe:filterQuakes -->

```go
feed, err := scalar.FromJSON(weekFeed)
if err != nil {
	panic(err)
}

strong := feed.Find("$.features[?@.properties.mag >= 4.5].properties.place")
for _, place := range strong.All() {
	fmt.Println(place.Or("(no place)"))
}
```

It prints:

```
12 km NNE of Ridgecrest, CA
(no place)
Mid-Atlantic Ridge
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Find`](EXAMPLES.md#svfind), [`AV.All`](EXAMPLES.md#avall), [`SV.Or`](EXAMPLES.md#svor).

<!-- /recipe -->

A null place comes back as undef, and `Or` gives it a default. The
reference's [Filters](REFERENCE.md#filters) has what a filter can say.
In Perl: [filter, then map](PERL.md#filter-then-map).

### A route built from data

When the fields to read come from a config or a user, each one a route,
`Get` takes the route as it is.

<!-- recipe:routeFromData -->

```go
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
```

It prints:

```
ci40123456	5.1	8.2
nc75012345	2.3	1.9
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Get`](EXAMPLES.md#svget), [`SV.Str`](EXAMPLES.md#svstr).

<!-- /recipe -->

A route that doesn't lead anywhere reads as undef, and `Str()` gives
`""`. For a route as one string, such as `"geometry.coordinates[2]"`,
use `At` instead.

## When something's missing

Feeds leave fields out and send them as null, and code reading them
spends much of its length on what to do then. In scalar, a missing or
null value is undef: `Or` gives a default, `OrSet` stores one, and `Then`
and `When` carry your own steps along a chain, skipping the rest at the
first undef, so the code reads in the order of the work. When the
difference between missing and null matters, `GetOK` tells you. Plain Go
answers with a pointer or a `map` lookup's `ok` for each field, and an
`if` for each default, which is explicit, and longer. See the reference's
[Undef](REFERENCE.md#undef) and [Chains that may come up
empty](REFERENCE.md#chains-that-may-come-up-empty-then-and-when).

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

### Defaults, and steps that may come up empty

A place shortened to what it's near, and a felt count turned into words,
each with a default for when there's nothing to work with.

<!-- recipe:defaultsAndSteps -->

```go
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
```

It prints:

```
Ridgecrest, CA: 1204 felt it
Cobb, CA: no reports
an unknown place: no reports
Mid-Atlantic Ridge: 3 felt it
```

In plain Go, printing the same:

```go
// nearRE finds the place a quake was near.
var nearRE = regexp.MustCompile(`of (.+)$`)

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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.Num`](EXAMPLES.md#svnum), [`SV.Str`](EXAMPLES.md#svstr), [`SV.Find`](EXAMPLES.md#svfind), [`AV.All`](EXAMPLES.md#avall), [`SV.Get`](EXAMPLES.md#svget), [`SV.M`](EXAMPLES.md#svm), [`AV.Get`](EXAMPLES.md#avget), [`SV.Or`](EXAMPLES.md#svor), [`SV.When`](EXAMPLES.md#svwhen), [`SV.Then`](EXAMPLES.md#svthen).

<!-- /recipe -->

`When` keeps a value only if it passes the test, and `Then` applies a
step to a value that's there; either gives undef otherwise, and `Or`
ends the chain with the default.

### Reading without changing the data

As in Perl, reading through a level that isn't there creates it, which
writing the data back out would show. `GetOK` and `AtOK` look without
creating anything, and `Autovivify(false)` stops reads creating anything
in the whole structure.

<!-- recipe:readWithoutChanging -->

```go
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
```

It prints:

```
false
false
{"properties":{"mag":4.6}}
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`SV.GetOK`](EXAMPLES.md#svgetok), [`SV.Autovivify`](EXAMPLES.md#svautovivify), [`SV.At`](EXAMPLES.md#svat), [`SV.Defined`](EXAMPLES.md#svdefined).

<!-- /recipe -->

Plain Go never creates anything by reading: a missing key in a `nil`
map reads as the zero value. The reference's [Looking without creating
anything](REFERENCE.md#looking-without-creating-anything) and [Turning
autovivification off](REFERENCE.md#turning-autovivification-off-autovivify)
have the rest.

## Into typed Go

Once you know the shape, move into structs, which the compiler checks:
`scalar.To[T]` converts a scalar, or any part of one, into a Go type you
choose, and says where a value doesn't fit. It reads values as Perl
would, so a number in a string fills a number field, which plain Go's
decoder refuses. From there on, it's plain Go. A part whose shape still
varies can stay a scalar inside the struct. The reference's [Which way
out?](REFERENCE.md#which-way-out) compares the ways into Go, and [Into Go
values](REFERENCE.md#into-go-values-to-and-decode) has the rest.

### Numbers that arrive as strings

Some feeds quote their numbers, some of the time.

<!-- recipe:numbersAsStrings -->

```go
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
```

It prints:

```
[{4.6 8.2} {2.3 1.9}]
```

In plain Go, printing the same:

```go
// number reads a JSON number, or a string holding one.
type number float64

func (n *number) UnmarshalJSON(b []byte) error {
	f, err := strconv.ParseFloat(strings.Trim(string(b), `"`), 64)
	*n = number(f)
	return err
}

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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`To`](EXAMPLES.md#to).

<!-- /recipe -->

The plain Go needs a type of its own, with an `UnmarshalJSON` method, for
each kind of field that might come quoted.

### Finding what doesn't fit

When a value can't fill its field, the error's path leads `At` to it.

<!-- recipe:whatDoesntFit -->

```go
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
```

It prints:

```
depth.km unknown
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`FromJSON`](EXAMPLES.md#fromjson), [`To`](EXAMPLES.md#to), [`SV.At`](EXAMPLES.md#svat).

<!-- /recipe -->

The reference's [When a value doesn't
fit](REFERENCE.md#when-a-value-doesnt-fit) has what the error holds.

### A part that stays loose

A `*scalar.SV` field takes whatever part of the document is there, for
the part whose shape differs from one record to the next, while the rest
of the struct is typed. `encoding/json` fills it through its
[`UnmarshalJSON`](EXAMPLES.md#svunmarshaljson) method.

<!-- recipe:loosePart -->

```go
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
```

It prints:

```
ci40123456 reviewed none
```

In plain Go, printing the same:

```go
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
```

**Uses:** [`SV.At`](EXAMPLES.md#svat), [`SV.Or`](EXAMPLES.md#svor).

<!-- /recipe -->

The plain Go keeps the part as `json.RawMessage`, and decodes it into a
type once it knows which.

## Working with text

Text gets the regular expressions and helpers Perl has: `M` matches, `S`
substitutes, `Split`, `Tr`, `Trim` and the case functions, on an engine
with lookaround and backreferences. Plain Go's `strings` and `regexp`
packages do most of this, without lookaround, in more lines. See the
reference's [Regular expressions](REFERENCE.md#regular-expressions).

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

## Counting and grouping

Arrays come with Perl's list utilities, `List::Util` and
`List::UtilsBy`, and scalar has sets and bags, from CPAN's
`Set::Scalar` and `Set::Bag`, which keep the order members were first
added. Plain Go has `slices` and `maps`, and a map for counting, with no
order of its own. See the reference's [List
utilities](REFERENCE.md#list-utilities) and [Sets and
bags](REFERENCE.md#sets-and-bags).

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

## Writing it out

Scalars, arrays and hashes work with `encoding/json` and the YAML
package as they are, and `Pack` writes binary records, as Perl's `pack`
does. In plain Go, a struct encodes as well, once its fields are known.
See the reference's [JSON](REFERENCE.md#json) and [Pack and
unpack](REFERENCE.md#pack-and-unpack).

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

## Making it faster

scalar puts ease ahead of speed, and is fast enough for most work at the
edges of a program. Where it's a cost, the regex engine can be changed
for Go's own, and anything whose shape is settled can move into plain
Go. See the reference's [What it costs](REFERENCE.md#what-it-costs).

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
