# scalar

[![test](https://github.com/nukopian/scalar/actions/workflows/test.yml/badge.svg)](https://github.com/nukopian/scalar/actions/workflows/test.yml)

Read, clean and reshape messy data in Go, then get back to typed Go.

Go is at its best when data has a known shape. Data from APIs, language
models, scraped pages and other people's systems often doesn't: fields go
missing, nesting varies, numbers turn up as strings, and the value you need
is buried in a sentence. In plain Go, each of those costs a type assertion,
a nil check or a hand-written loop.

`scalar` takes that ceremony away. Here it reads the last day's
earthquakes from the US Geological Survey's live feed:

```go
resp, err := http.Get("https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_day.geojson")
if err != nil {
	return err
}
body, err := io.ReadAll(resp.Body)
resp.Body.Close()
if err != nil {
	return err
}

feed, err := scalar.FromJSON(body)   // GeoJSON, whatever shape it turns out to be
if err != nil {
	return err
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
		"depth", q.At("geometry.coordinates[2]"),   // [lon, lat, depth]
		"time", q.At("properties.time"),
	)))
}

// The shape is known: into ordinary, typed Go.
type Quake struct {
	Mag   float64 `json:"mag"`
	Near  string  `json:"near"`
	Depth float64 `json:"depth"`   // km
	Time  int64   `json:"time"`    // ms since 1970
}
quakes, err := scalar.To[[]Quake](big)
if err != nil {
	return err
}

// From here on, it's plain Go.
fmt.Printf("Strongest: M%.1f\n", strongest)
for _, q := range quakes {
	when := time.UnixMilli(q.Time).UTC().Format("2 Jan 15:04 GMT")
	fmt.Printf("M%.1f near %s, %.0f km deep, %s\n", q.Mag, q.Near, q.Depth, when)
}
```

One day's run printed:

```
Strongest: M5.2
M4.7 near Sandy Point Town, Saint Kitts and Nevis, 150 km deep, 18 Oct 15:35 GMT
M4.5 near Kermadec Islands, New Zealand, 10 km deep, 18 Oct 12:42 GMT
M5.2 near Ruteng, Indonesia, 10 km deep, 18 Oct 08:00 GMT
M4.8 near Kainantu, Papua New Guinea, 124 km deep, 18 Oct 00:28 GMT
```

There isn't a type assertion or a nil check in the `scalar` part, nor a loop
to find the big ones. `$..mag` searches the whole tree, and the filter picks
out the quakes of 4.5 and over. A `null` place, a missing field and a
pattern that didn't match are all undef, an ordinary value that passes along
the chain, so one `Or` after another tries each fallback in turn. Once the
data has the shape you want, `To` converts it to your own types, or returns
an error saying which value didn't fit, and where, and the rest is ordinary
Go. The same code runs as a test on a fixed sample of the feed, with every
case above: `Example_overview`, in `example_test.go`.

Most of the tools come from Perl, which has handled exactly this kind of
data for decades: its nested paths, regular expressions, `//` operator,
`map`, `grep` and list utilities, carried over faithfully and checked
against Perl itself. Searching comes from JSONPath, the RFC 9535 standard,
checked against its compliance suite.
You don't need to know Perl to use them. The comparisons with Perl in these
docs are there for readers who do.

## Install

```sh
go get github.com/nukopian/scalar
```

It needs Go 1.25 or later, and brings in three modules: `regexp2` for
Perl-compatible regular expressions, `yaml/v3`, and `x/text`.

## Rationale

I am relatively new to Go, but I've written code for well over thirty
years, encountering C/C++, REXX, XSLT, JavaScript (and friends), Scala,
Python, and, to a much greater extent, Perl at various points throughout
that journey. While I may be learning on the job, this is definitely not
my first rodeo, and I'm looking forward to mastering this new skill.

Learning on the job often requires flexibility, adaptability and a healthy
dose of pragmatism when dealing with obstacles. When learning a new
language while trying to be productive, the barriers between an idea or
concept and its eventual expression can often be the language itself,
its ecosystem, and the time it takes to learn both, especially when time
is in short supply.

For newcomers to any language, even basic tasks can be frustratingly
tedious, and Go is not immune to this: a struct to write before you can
read a field, or checking for every value that might be missing, or a loop
for what is a one-liner elsewhere. I get why Go's idioms matter, but
sometimes a job needs doing fast, and brains are hard-wired to prefer
shortcuts!

Many of the shortcuts I've wanted already exist, proven over decades,
in other languages and paradigms, and this package borrows from them:

- Perl: handling messy data, nested paths, a regex at every turn, `//`,
  `map` and `grep`.
- Functional programming: Maybe, Scala's `Option`, in which a missing
  value flows quietly through a chain of `Then` and `Or`.
- XPath: its way of querying a tree, as JSONPath, with Python's slices.
- Set theory: sets and bags.
- Go: its own iterators and generics, to get back to typed Go at the
  end.

Solving a problem quickly and well enough doesn't always mean torturing
oneself in service to linguistic purity, nor does it mean giving up on
one language for another language and its ecosystem. We can choose to
be practical.

I created `scalar` for my own use, when I'm in a hurry. I also created
it the way I did to help me learn. I am sharing it because it might be
helpful to others wanting to turn an idea, experiment or proof-of-concept
into working Go code, especially those like me who are learning Go. The
work isn't aimed at Go purists. It should be thought of as a halfway
house that gets you to a working solution to a problem quickly, leaving
room for improvement. When performance requirements bite and the shape of
the data has settled, `To` hands that data back as ordinary typed Go.

## What's in it

| For | Use | Perl |
|---|---|---|
| Reading nested data | As Perl does: `Get("content", 0, "text")`, or a step at a time, with `Get(k).Set(x)`, `Exists` and `Delete` taking keys the same way, or the whole route as one string, `At("content[0].text")`. Without creating anything, for data you aren't sure of: `GetOK` and `AtOK`, which tell missing from null, and `DeletePath` and `DeleteAt`. `CheckPath` checks a route from outside the program | `$r->{content}[0]{text}` |
| Searching nested data | `Find("$..stations[*]")`, `Find("$.features[?@.properties.mag >= 4.5]")`, JSONPath (RFC 9535, all of it); `QuoteJSONPath` to put a Go value into a query | |
| Changing every value | `Apply(f)`, all the way down | `Data::Rmap`'s `rmap` |
| Comparing structures | `Equal`, and `Diff` to say where they differ | `Test::More`'s `is_deeply` |
| Filling in text | `Interpolate("M$mag near $place: @stations")`, with `MustTemplate` to compile one and set the list separator | `"$place: @stations"`, `$"` |
| Text | `Substr`, `Index`, `Lc`, `Fc`, `Trim`, counting characters, not bytes | `substr`, `index`, `lc`, `fc`, `trim` |
| Regular expressions | `M`, `Named`, `S`, `Split`, `Tr`, with lookarounds and backreferences; `MustCompile` to compile a pattern once | `m//`, `%+`, `s///`, `split`, `tr///`, `qr//` |
| Arrays | `Push`, `Pop`, `PopN`, `Splice`, `Sort`, `Map`, `Grep`, `Values` and `Pairs` slices, `Each` | `push`, `splice`, `@a[...]`, `%a[...]`, `each` |
| List utilities | `AnyMatch`, `FirstMatch`, `MaxNum`, `UniqueStr`, `Sum0`, `Reduce`, … | `List::Util` |
| Sorting, grouping and counting by a key | `SortByNum`, `RevSortByNum`, `MaxBy`, `UniqueBy`, `PartitionBy`, `CountBy`, `ExtractBy`, … | `List::UtilsBy` |
| Hashes | `Get`, `Set`, `Push` to merge, `Values` and `Pairs` slices, `Each`, autovivification | `$h{k}`, `@h{...}`, `%h{...}`, `each` |
| Sets and counts | `NewSet` with `Union`, `Intersection`, …; `Bag` with `Insert`, `Count`, `MostCommon` | `Set::Scalar`, `Set::Bag` |
| Numbers and operators | `Add`, `Concat`, `Inc` (`"aa"` to `"ab"`), `AddInPlace`, `NumCmp`; `fmt` verbs as `sprintf` | `+`, `.`, `++`, `+=`, `<=>`, `sprintf` |
| Binary data | byte strings, from a `[]byte`; `Pack` and `Unpack` for hex, bits and floats | `pack`, `unpack` |
| Steps that may find nothing | `Then(f)` and `When(pred)`, which pass undef straight through, then `Or(default)` | `//` |
| JSON | `FromJSON`, and `encoding/json` for writing | |
| YAML: config, Kubernetes, Helm, CloudFormation | `FromYAML`, `FromYAMLAll` for several documents, `ToYAML` | |
| Copies | `Copy`, shallow, as assignment is; `Clone`, deep, for a structure of your own | `my $x = $y`, `dclone` |
| Locking | `Lock`, all the way down: no new keys, no changes, no autovivifying, and reading a key it hasn't got panics; `Unlock`, `Locked` | `Hash::Util`'s `lock_hash_recurse`, `Internals::SvREADONLY` |
| Looking at a structure | `Dump()`, as `Data::Dumper::Concise` prints it, and `FromDump` to read it back; `fmt.Println` of an array or hash, on one line | `Dumper` |
| Back to plain Go | `To[T]` for any Go type you name, checked; `Decode(&v)` to fill one you have; `StrSlice`, `IntSlice` and `NumSlice` in Perl's contexts; `Any`, `Bytes` | |

For how each feature behaves, see the [reference](docs/REFERENCE.md); for
a worked example of every function and method, see the
[examples](docs/EXAMPLES.md).

## When not to use it

- **When you know the data's shape.** Declare a struct and use
  `encoding/json`; `scalar` would only add cost. It belongs at the edges,
  where data comes in, and `To` turns it into your own types as soon as
  the shape is clear.
- **When a misspelled key must fail to compile.** Dynamic values give up
  compile-time checks: a wrong key just reads as undef.
- **In hot paths.** Every value is boxed, so `scalar` is slower than plain
  Go and allocates more. Measure before you use it in a tight loop.
- **When a panic is unacceptable.** Like Perl, it stops on things such as an
  invalid regex or division by zero, by panicking; the
  [caveats](docs/REFERENCE.md#caveats) list every case, and the error-returning
  alternatives.
- **For data shared between goroutines,** unless it's
  [locked](docs/REFERENCE.md#locking): some reads write, so share plain Go
  values or a locked structure. See
  [Concurrency](docs/REFERENCE.md#concurrency).

[What it costs](docs/REFERENCE.md#what-it-costs) has the reasoning.

## Documentation

- [How-to guide](docs/HOWTO.md): a quick start, recipes for common tasks,
  and two jobs done both in plain Go and with `scalar`.
- [Reference](docs/REFERENCE.md): each feature in depth, how it behaves,
  and where it follows Perl or another source.
- [Examples](docs/EXAMPLES.md): every function and method, each with a
  working example and its output, generated from the tests.
- [From Perl to Go](docs/PERL.md): a phrasebook of everyday tasks in Perl,
  with `scalar` and in plain Go, all three checked against each other.

To read them offline, `scripts/readme-pdf.sh` builds a PDF of each, with
pandoc and Prince. [CLAUDE.md](CLAUDE.md) sets out how `scalar` is
developed: the rules, and the reasons for them.

## Status

Version 0: it works, and it's tested against Perl itself, the JSONPath
compliance suite and fuzzing, but the API may still change before 1.0.

## Licence

Copyright 2026 Iain Campbell. `scalar` is free software; you can
redistribute it and/or modify it under the same terms as Perl itself:
either the GNU General Public License, version 1 or (at your option) any
later version, or the Artistic License, as they come with Perl
(`Artistic-1.0-Perl OR GPL-1.0-or-later`). Both are in [LICENSE](LICENSE).

## Acknowledgements

The ideas here aren't new. Most come from Perl, and the rest from
JSONPath, functional programming and a few CPAN modules. Thanks to the
Perl community, whose thinking about messy data I've leaned on for years,
and to the authors of the JSONPath Compliance Test Suite, which keeps
`Find` honest.

Claude, Anthropic's AI model, wrote most of the code, tests and
documentation, and every commit says so in a `Co-Authored-By` trailer. My
part was deciding what `scalar` should do and what it shouldn't, setting
the rules it's built by (they're in [CLAUDE.md](CLAUDE.md)), and reviewing
the work. It has been a partnership, and I've learned a good deal of Go
along the way.

The `scalar` package started inside a larger proof-of-concept and was
extracted with its history, so some commit messages mention packages
that stayed behind.
