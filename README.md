# scalar

Read, clean and reshape messy data in Go, then get back to typed Go.

Go is at its best when data has a known shape. Data from APIs, language
models, scraped pages and other people's systems often doesn't: fields go
missing, nesting varies, numbers turn up as strings, and the value you need
is buried in a sentence. In plain Go, each of those costs a type assertion,
a nil check or a hand-written loop.

`scalar` takes that ceremony away:

```go
doc, err := scalar.FromJSON(body)   // whatever shape it turns out to be
if err != nil {
	return err
}

// Clean every string in the structure, at any depth.
doc.Apply(func(v *scalar.SV) any {
	if v.Kind() == scalar.String {
		return v.Trim()
	}
	return v
})

// Read nested values: anything missing or null is undef, not a panic.
title := doc.At("posting.title").Str()
location := doc.At("posting.location").Or("Unknown").Str()
remote := doc.At("posting.remote").Bool()   // not there at all: false

// Pull a figure out of text: no match gives -1, not an error to handle.
salary := doc.At("posting.description").
	M(`£([\d,]+)`).Get(1).
	Then(func(v *scalar.SV) any { return v.S(`,`, "", "gr") }).   // "45,000" → "45000"
	Or(-1).
	Int()

// Then back to ordinary, typed Go.
posting, err := scalar.To[Posting](doc.At("posting"))
```

There isn't a type assertion or a nil check in that code. A missing key, a
JSON `null` and a pattern that didn't match are all undef, an ordinary value
that passes along the chain, and `Or` supplies a default wherever you want
one. Once the data is in shape, `To` converts it to your own struct, or
returns an error saying which value didn't fit, and where. The example runs
as a test: `Example_overview`, in `example_test.go`.

The tools come from Perl, which has handled exactly this kind of data for
decades: its nested paths, regular expressions, `//` operator, `map`, `grep`
and list utilities, carried over faithfully and checked against Perl itself.
You don't need to know Perl to use them. The comparisons with Perl in this
README are there for readers who do.

## Why scalar exists

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
