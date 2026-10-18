# scalar

Read, clean and reshape messy data in Go, then get back to typed Go.

Go is at its best when data has a known shape. Data from APIs, language
models, scraped pages and other people's systems often doesn't: fields go
missing, nesting varies, numbers turn up as strings, and the value you need
is buried in a sentence. In plain Go, each of those costs a type assertion,
a nil check or a hand-written loop.

`scalar` takes that ceremony away. Here it reads an earthquake feed from
the US Geological Survey:

```go
feed, err := scalar.FromJSON(body)   // a USGS earthquake feed, in GeoJSON
if err != nil {
	return err
}

// Search the whole tree with JSONPath: the strongest magnitude anywhere.
fmt.Println("Strongest:", feed.Find("$..mag").MaxNum())

// Filter with JSONPath, then read nested values: anything missing or
// null is undef, not a panic, and Or supplies a default.
for _, q := range feed.Find("$.features[?@.properties.mag >= 4.5]").All() {
	mag := q.At("properties.mag").Num()
	depth := q.At("geometry.coordinates[2]").Num()   // [lon, lat, depth]
	near := q.At("properties.place").
		M(`of (.+)$`).Get(1).   // "12 km NNE of Ridgecrest, CA" → "Ridgecrest, CA"
		Or("an unknown place").Str()
	when := time.UnixMilli(q.At("properties.time").Int()).UTC()
	fmt.Printf("M%.1f near %s, %.0f km deep, %s\n", mag, near, depth, when.Format("2 Jan 15:04 MST"))
}

// Then back to ordinary, typed Go.
quakes, err := scalar.To[[]Quake](feed.Find("$.features[*].properties"))
```

```
Strongest: 5.1
M5.1 near Ridgecrest, CA, 8 km deep, 18 Oct 10:00 UTC
M4.7 near an unknown place, 10 km deep, 18 Oct 11:00 UTC
```

There isn't a type assertion or a nil check in that code, nor a loop to find
the big ones. `$..mag` searches the whole tree, and the filter picks out the
quakes of 4.5 and over. A `null` place, a missing field and a pattern that
didn't match are all undef, an ordinary value that passes along the chain,
and `Or` supplies a default wherever you want one. Once the data is in
shape, `To` converts it to your own types, or returns an error saying which
value didn't fit, and where. The example runs as a test, on a sample of the
feed: `Example_overview`, in `example_test.go`.

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

It needs Go 1.26.9 or later, and brings in three modules: `regexp2` for
Perl-compatible regular expressions, `yaml/v3`, and `x/text`.

## Rationale

Though, I'm relatively new to Go, I've been programming for nearly over
three decades: C, REXX, Perl, Python, JavaScript, XSLT and Scala came
first. Go's concepts weren't the hard part. What struck me was how painful
and tediously repetitive even basic tasks can be: a struct to write before
you can read a field, a check for every value that might be missing, a
loop for what is a one-liner elsewhere. I understand why Go's idioms
matter. But sometimes a job needs doing and doing quickly, and
brains *do* prefer shortcuts.

The barrier between an idea and its implementation is often the language,
its ecosystem, and the time it takes to learn them. Many of the shortcuts
I've wanted already exist, proven over decades, in other languages and
paradigms I've used, and `scalar` borrows from them:

- Perl's way with messy data: nested paths, a regex at every turn, `//`,
  `map` and `grep`.
- Functional programming's Maybe, Scala's `Option`, in which a missing
  value flows quietly through a chain of `Then` and `Or`.
- XPath's way of querying a tree, as JSONPath, with Python's slices.
- The set theory of sets and bags.
- Go's own iterators and generics, to get back to typed Go at the end.

Then solving a problem quickly and well enough doesn't mean torturing
yourself for linguistic purity, or giving up on Go for another language
and its ecosystem. Be practical, and pragmatic.

Primarily, this is for my own use, when I'm in a hurry, but also for
anyone who wants a helping hand turning ideas, experiments and proofs of
concept into something that works: especially people learning Go, and Go
programmers tired of schlepping. It isn't for Go purists, and doesn't try
to be. Think of it as a half-way house. It gets you to a working result
quickly, and leaves room to improve. When production performance begins
to bite and the data's shape has settled, `To` hands the data back as
ordinary typed Go, and the code that stays can become as idiomatic as it
needs to be.

## What's in it

| For | Use | Perl |
|---|---|---|
| Reading nested data | `Path("content", 0, "text")`, or the whole route as one string: `At("content[0].text")`, `At("{content}[0]{text}")`; `PathOK` and `AtOK` to tell missing from null; `DeleteAt` to remove what a route leads to; `MustPath` to compile a route once | `$r->{content}[0]{text}` |
| Searching nested data | `Find("$..skills[*]")`, `Find("$.jobs[?@.salary.min > 30000]")`, JSONPath (RFC 9535, all of it); `QuoteJSONPath` to put a Go value into a query | |
| Changing every value | `Apply(f)`, all the way down | `Data::Rmap`'s `rmap` |
| Comparing structures | `Equal`, and `Diff` to say where they differ | `Test::More`'s `is_deeply` |
| Filling in text | `Interpolate("$title in $location: @skills")`, with `MustTemplate` to compile one and set the list separator | `"$title: @skills"`, `$"` |
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

Each has a PDF beside it. [CLAUDE.md](CLAUDE.md) sets out how `scalar` is
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
The grammar package, in [grammar/](grammar/), has a licence of its own.

## Acknowledgements

The ideas here aren't new. `scalar` borrows Perl's way with data: undef
that passes harmlessly through a chain, autovivification, context, list
utilities, and a regex at every turn. It's tested against Perl itself to
keep its behaviour faithful. Thanks to the Perl community for decades of
thought about handling messy data well.

`scalar` was conceived and directed by Iain Campbell (nukopian). He set
its rules: Perl as the baseline, checked against Perl itself, with any
difference from Perl only where Go can't follow, each one documented;
tests first; and speed changes kept only when measured. He decided what
to build and what not to, and reviewed and committed each change. Most of
the code, tests and documentation were written by Claude, Anthropic's AI
model, working to that direction, from the package's first commit on 11
October 2026, and every commit credits Claude in a `Co-Authored-By`
trailer.

`scalar` was extracted, with its history, from a larger project, so some
commit messages mention packages that stayed behind.
