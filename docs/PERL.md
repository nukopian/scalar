# scalar: from Perl to Go

A phrasebook for Perl programmers coming to Go. Each task below is shown
three ways: in Perl, with [scalar](../README.md), and in plain Go. Use
scalar to get going in Go with what you already know; use the plain Go
column to see what it looks like once the data's shape is settled, and
when it's worth switching.

All three versions of every task run as a test, `TestMappingMatchesPerl`
in `perl_mapping_test.go`, on the same feed, and must give the same
answer, Perl's included.

For how each scalar feature behaves, see the [reference](REFERENCE.md);
for tasks in more depth, the [how-to guide](HOWTO.md).

- [The feed](#the-feed)
- [A nested field, with a default](#a-nested-field-with-a-default)
- [Missing, null or a value](#missing-null-or-a-value)
- [Split, trim and drop duplicates](#split-trim-and-drop-duplicates)
- [Every figure in a text](#every-figure-in-a-text)
- [Filter, then map](#filter-then-map)
- [Sort by a field](#sort-by-a-field)
- [Count by a key](#count-by-a-key)
- [Max and sum](#max-and-sum)
- [Variables in a string](#variables-in-a-string)
- [Substitute](#substitute)
- [A hash slice](#a-hash-slice)
- [When to switch](#when-to-switch)

## The feed

Four quakes, in the shape of the USGS earthquake feed, with the usual mess:
a `null` place, a `null` felt count, one missing altogether, and a list of
stations typed by hand.

```json
{"features": [
  {"id": "ci40123456", "properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "felt": null, "net": "ci",
    "stations": " PAS, GSC ,pas,ISA"}},
  {"id": "nc75012345", "properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "felt": 12, "net": "nc"}},
  {"id": "us7000abcd", "properties": {"mag": 4.7, "place": null, "net": "us"}},
  {"id": "ak0251xyz", "properties": {"mag": 4.9, "place": "Kodiak Island region, Alaska", "felt": 3, "net": "ak"}}
]}
```

Each version reads it its own way first:

```perl
my $feed = decode_json($body);
my @quakes = @{ $feed->{features} };
my $p = $quakes[0]{properties};
```

```go
// scalar
feed, err := scalar.FromJSON(body)
p := feed.At("features[0].properties")
```

```go
// plain Go: a type for the shape, then encoding/json
type plainQuake struct {
	ID         string `json:"id"`
	Properties struct {
		Mag      float64 `json:"mag"`
		Place    *string `json:"place"` // a pointer, so null is nil
		Felt     *int    `json:"felt"`
		Net      string  `json:"net"`
		Stations string  `json:"stations"`
	} `json:"properties"`
}
type plainFeed struct {
	Features []plainQuake `json:"features"`
}

var feed plainFeed
err := json.Unmarshal(body, &feed)
p := feed.Features[0].Properties
```

That's the first difference, and the biggest: plain Go wants the shape
written down before it reads a field. scalar reads first, and you write the
type when you know it.

## A nested field, with a default

The third quake's place is `null`. Answer: `unknown`.

```perl
$feed->{features}[2]{properties}{place} // "unknown"
```

```go
// scalar
feed.At("features[2].properties.place").Or("unknown").Str()
```

```go
// plain Go
place := "unknown"
if pl := feed.Features[2].Properties.Place; pl != nil {
	place = *pl
}
```

## Missing, null or a value

The first quake's `felt` is `null`; the third has no `felt` at all.
Answer: `null missing`.

```perl
!exists $props->{felt} ? "missing" : !defined $props->{felt} ? "null" : $props->{felt}
```

```go
// scalar
switch v, ok := props.AtOK("felt"); {
case !ok:
	// missing
case !v.Defined():
	// null
default:
	// v
}
```

```go
// plain Go: a struct can't tell missing from null, so read it as a map
props := features[i].(map[string]any)["properties"].(map[string]any)
switch v, ok := props["felt"]; {
case !ok:
	// missing
case v == nil:
	// null
default:
	// v
}
```

## Split, trim and drop duplicates

The stations, `" PAS, GSC ,pas,ISA"`, regardless of case, keeping the first
spelling. Answer: `PAS,GSC,ISA`.

```perl
my %seen;
join ",", grep { !$seen{fc $_}++ } map { s/^\s+|\s+$//gr } split /,/, $p->{stations};
```

```go
// scalar
seen := scalar.Hash()
p.At("stations").Split(",").
	Map(func(s *scalar.SV) any { return s.Trim() }).
	Grep(func(s *scalar.SV) bool { return seen.Get(s.Fc().Str()).Inc().Int() == 1 }).
	Join(",")
```

```go
// plain Go
seen := map[string]bool{}
var unique []string
for _, s := range strings.Split(p.Stations, ",") {
	s = strings.TrimSpace(s)
	if key := strings.ToLower(s); !seen[key] {
		seen[key] = true
		unique = append(unique, s)
	}
}
strings.Join(unique, ",")
```

`strings.ToLower` is close to Perl's `fc`, but not the same: `fc` folds
`ß` to `ss`, and scalar's `Fc` does too.

## Every figure in a text

`"Felt by 12,431 people in 1,204 ZIP codes"`, commas removed. Answer:
`12431 1204`.

```perl
join " ", map { tr/,//dr } $text =~ /(\d[\d,]*)/g;
```

```go
// scalar
figures := scalar.Array()
for m := range text.Matches(`(\d[\d,]*)`) {
	figures.Push(m.Get(1).S(`,`, "", "gr"))
}
figures.Join(" ")
```

```go
// plain Go
var figureRe = regexp.MustCompile(`(\d[\d,]*)`)

var figures []string
for _, m := range figureRe.FindAllStringSubmatch(text, -1) {
	figures = append(figures, strings.ReplaceAll(m[1], ",", ""))
}
strings.Join(figures, " ")
```

## Filter, then map

The places of the quakes of 4.5 and over, with a default for a `null` one.
Answer: `12 km NNE of Ridgecrest, CA; unknown; Kodiak Island region, Alaska`.

```perl
join "; ", map { $_->{properties}{place} // "unknown" }
    grep { $_->{properties}{mag} >= 4.5 } @quakes;
```

```go
// scalar: the filter as JSONPath
feed.Find("$.features[?@.properties.mag >= 4.5]").
	Map(func(q *scalar.SV) any { return q.At("properties.place").Or("unknown") }).
	Join("; ")
```

```go
// plain Go
var places []string
for _, q := range feed.Features {
	if q.Properties.Mag >= 4.5 {
		place := "unknown"
		if q.Properties.Place != nil {
			place = *q.Properties.Place
		}
		places = append(places, place)
	}
}
strings.Join(places, "; ")
```

## Sort by a field

Event IDs, strongest first. Answer:
`ci40123456,ak0251xyz,us7000abcd,nc75012345`.

```perl
join ",", map { $_->{id} }
    sort { $b->{properties}{mag} <=> $a->{properties}{mag} } @quakes;
```

```go
// scalar: List::UtilsBy's rev_nsort_by
feed.At("features").AV().
	RevSortByNum(func(q *scalar.SV) any { return q.At("properties.mag") }).
	Map(func(q *scalar.SV) any { return q.At("id") }).
	Join(",")
```

```go
// plain Go
quakes := slices.Clone(feed.Features)
slices.SortStableFunc(quakes, func(a, b plainQuake) int {
	return cmp.Compare(b.Properties.Mag, a.Properties.Mag)
})
var ids []string
for _, q := range quakes {
	ids = append(ids, q.ID)
}
strings.Join(ids, ",")
```

## Count by a key

Quakes per whole magnitude, keys sorted. Answer: `2=1 4=2 5=1`.

```perl
my %count;
$count{int $_->{properties}{mag}}++ for @quakes;
join " ", map { "$_=$count{$_}" } sort keys %count;
```

```go
// scalar
count := scalar.Hash()
for _, q := range feed.At("features").AV().All() {
	count.Get(fmt.Sprint(q.At("properties.mag").Int())).Inc()
}
count.Keys().Sort(nil).
	Map(func(k *scalar.SV) any { return k.Str() + "=" + count.Get(k.Str()).Str() }).
	Join(" ")
```

```go
// plain Go
count := map[int]int{}
for _, q := range feed.Features {
	count[int(q.Properties.Mag)]++
}
var counts []string
for _, k := range slices.Sorted(maps.Keys(count)) {
	counts = append(counts, fmt.Sprintf("%d=%d", k, count[k]))
}
strings.Join(counts, " ")
```

Note that Go sorts the integer keys as numbers, and Perl's `sort keys` as
strings; they agree here only because every key is one digit.

## Max and sum

The strongest magnitude, and the felt reports in total, `null` and missing
counting as 0. Answer: `5.1 15`.

```perl
use List::Util qw(max sum0);
max(map { $_->{properties}{mag} } @quakes),
sum0(map { $_->{properties}{felt} // 0 } @quakes);
```

```go
// scalar: JSONPath gathers them from anywhere in the tree
feed.Find("$..mag").MaxNum()
feed.Find("$..felt").Sum0() // undef counts as 0
```

```go
// plain Go
maxMag, sumFelt := 0.0, 0
for _, q := range feed.Features {
	maxMag = max(maxMag, q.Properties.Mag)
	if q.Properties.Felt != nil {
		sumFelt += *q.Properties.Felt
	}
}
```

## Variables in a string

Answer: `M5.1 near 12 km NNE of Ridgecrest, CA`.

```perl
"M$p->{mag} near $p->{place}"
```

```go
// scalar: the hash's keys are the variables
p.Interpolate("M$mag near $place")
```

```go
// plain Go
fmt.Sprintf("M%v near %s", p.Mag, *p.Place)
```

## Substitute

The place without its distance and direction. Answer: `Ridgecrest, CA`.

```perl
$p->{place} =~ s/^\d+ km \w+ of //r
```

```go
// scalar: s///r, Perl's flags as the last argument
p.At("place").S(`^\d+ km \w+ of `, "", "r")
```

```go
// plain Go
var distanceRe = regexp.MustCompile(`^\d+ km \w+ of `)

distanceRe.ReplaceAllString(*p.Place, "")
```

## A hash slice

Answer: `5.1 ci`.

```perl
join " ", @{$p}{qw(mag net)}
```

```go
// scalar
p.HV().Values("mag", "net").Join(" ")
```

```go
// plain Go
fmt.Sprint(p.Mag, " ", p.Net)
```

## When to switch

The plain Go column is longer at the edges, where data arrives and its
shape is uncertain: missing and null values, mixed types, lists typed by
hand. There, scalar saves the most. It's shorter or level wherever the
shape is known: once `plainFeed` exists, `q.Properties.Mag` is checked by
the compiler, where a misspelt `q.At("properties.mga")` is quietly undef.

So the usual path is: start with scalar while the shape is unclear, then
convert once it's settled, with `scalar.To[plainFeed](feed)`, and carry on
in plain Go from there. The [README](../README.md#when-not-to-use-it) says
more about when not to use scalar at all.
