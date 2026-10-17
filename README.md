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
its ecosystem and the time it takes to learn them. Many of the shortcuts I
wanted already exist, proven over decades, in languages I've used.
`scalar` borrows them:

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

## Compared with plain Go

Two jobs are awkward in plain Go.

**1. Reading loosely shaped JSON.** LLM output and third-party APIs return
JSON whose shape changes: fields go missing, nesting varies, and numbers turn
up as strings. In plain Go, reading one nested value means a chain of type
assertions, and each one needs checking:

```go
var resp map[string]any
if err := json.Unmarshal(body, &resp); err != nil { ... }
content, ok := resp["content"].([]any)
if !ok || len(content) == 0 { ... }
first, ok := content[0].(map[string]any)
if !ok { ... }
text, ok := first["text"].(string)
if !ok { ... }
```

Skip a check and the program panics on the first unexpected response. With
`scalar`, the same read is:

```go
resp, err := scalar.FromJSON(body)
if err != nil { ... }
text := resp.At("content[0].text").Str()   // or resp.Path("content", 0, "text")
```

If anything along the path is missing or the wrong type, `text` is just `""`.

**2. Heavy text processing.** Cleaning up text, such as job postings, means a
lot of regex work. Go's standard `regexp` package deliberately leaves out
lookahead, lookbehind and backreferences. `scalar` uses an engine that has
them, with a timeout so a bad pattern can't hang the program. It also wraps
them in short helpers modelled on Perl's `s///`, `split` and `tr///`.

## What it costs

Dynamic values give up compile-time checking. A misspelled key compiles
without complaint and silently reads as undef. That's why the rule for this
package is:

> **Use `scalar` at the edges, where data comes in. Convert to ordinary Go
> structs as soon as you know the shape, and keep the core of the program
> typed.**

`To` is that conversion, in one step: `scalar.To[Posting](doc)` gives you a
`Posting`, or an error saying which value didn't fit. See
[Into Go values](#into-go-values-to-and-decode).

It's also what makes concurrent code safe. A struct of plain values can be
shared between goroutines for reading; a scalar can't, as some of its
reads write. See [Concurrency](#concurrency).

## Quick start

```go
import "github.com/nukopian/job-meta/internal/scalar"

resp, err := scalar.FromJSON(body)
if err != nil {
	return err
}

text := resp.Path("content", 0, "text").Str() // $resp->{content}[0]{text}
tokens := resp.At("usage.output_tokens").Int() // the same, as one string

usage, err := scalar.To[Usage](resp.At("usage")) // into your own struct, checked
```

If any step along the path is missing, you get undef back instead of a panic.
`Str()` on undef is `""` and `Int()` is `0`. `Path` takes keys and indices
as separate arguments; `At` takes the route as one string, in dot, bracket
or Perl's brace form (see [A path as one string](#a-path-as-one-string-at)).

## Why Perl?

Perl is better than Go at some jobs, and messy text and loosely shaped data
are among them. But for a whole program, Perl usually isn't the answer. Go's
types, concurrency, tooling and single-binary deployment are worth keeping.

Borrowing from Perl can be the answer. Go programmers write the same tedious
patterns over and over: a chain of type assertions to read one field, a
hand-written loop to find a substring by character, a fresh regex for every
small clean-up. Perl solved these problems decades ago. `scalar` borrows its
solutions and leaves the rest of the program as ordinary Go.

Go has done this itself. `slices.Sort`, `cmp.Or` and range-over-func
iterators arrived after years of every codebase writing the same loops by
hand. `scalar` takes the same approach to Perl's data model, which is a
well-tested design for exactly this kind of work.

## The types

`scalar` gives you three types from Perl itself:

| Type | Perl | What it is |
|------|------|------------|
| `SV` | `$x` | A scalar: undef, a boolean, an integer, a float, a string, a byte string (binary data), or a reference to an `AV` or `HV` |
| `AV` | `@a` | An array of scalars |
| `HV` | `%h` | A hash of scalars, keyed by string |

and two from CPAN, which take Go's usual names, as Perl has no built-in
names for them (see [Sets and bags](#sets-and-bags)):

| Type | Perl | What it is |
|------|------|------------|
| `Set` | `Set::Scalar` | A set of scalars: each member once |
| `Bag` | `Set::Bag` | A bag of scalars: each member with a count |

A scalar converts itself to whatever you ask for, the way Perl does. Ask for a
string and you get a string; ask for a number and you get a number.

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

For a worked example of every function and method, see the
[Reference](#reference) at the end.

## Recipes

Six short examples from this project's work, each combining several parts of
the package. The sections after this one explain each part in detail.

These recipes are also runnable examples in `example_test.go`, so `go test`
checks they still work. Keep the two in step when changing them.

### Reading a model's tool_use response

Find the `tool_use` block, then read its fields with defaults, telling a
missing field from a null one.

```go
resp, err := scalar.FromJSON(body)
if err != nil {
	return err
}

input := resp.Find("$.content[*]").FirstMatch(func(b *scalar.SV) bool {
	return b.Path("type").Eq("tool_use")
}).Path("input")

fmt.Println(input.Path("title"))                         // Senior Go Developer
fmt.Println(resp.At("usage.output_tokens").Or(0))        // 0, if the model didn't say

if v, ok := input.PathOK("salaryMax"); ok && !v.Defined() {
	fmt.Println("salaryMax: null")                         // present, but null
}
if _, ok := input.PathOK("location"); !ok {
	fmt.Println("location: missing")                       // left out entirely
}
input.HV().Get("location").OrSet("Unknown")              // $input->{location} //= "Unknown"
```

### Cleaning a list of skills

Trim each skill where it's stored, then drop duplicates regardless of case,
keeping the first spelling of each.

```go
skills := resp.Find("$..requiredSkills").Get(0).AV()   // [" Go ", "PostgreSQL", "go", "Docker "]

for _, s := range skills.All() {
	s.Set(s.Trim())                                      // changes the skills inside resp
}

// grep { !$seen{fc $_}++ } @skills
seen := scalar.Hash()
unique := skills.Grep(func(s *scalar.SV) bool {
	return seen.Get(s.Fc().Str()).Inc().Int() == 1
})
fmt.Println(unique.Join(", "))                         // Go, PostgreSQL, Docker
```

### Pulling salary figures out of text

The pattern is compiled once, when the program starts, as with Perl's
`qr//`.

```go
var salaryFigure = scalar.MustCompile(`£([\d,]+)`)

text := scalar.Value("Salary: £45,000 - £55,000 per annum")

figures := scalar.Array()
for m := range text.Matches(salaryFigure) {         // while ($text =~ /$salaryFigure/g)
	figures.Push(m.Get(1).S(`,`, "", "gr"))         // "45,000" → "45000"
}
fmt.Println(figures.MinNum(), figures.MaxNum())    // 45000 55000
```

### Counting skills across postings

Count each skill ignoring case in a `Bag`, then list the most common first.

```go
postings := [][]string{
	{"Go", "Docker"},
	{"go", "Kubernetes"},
	{"GO", "Docker", "Perl"},
}

var counts scalar.Bag                                // the zero value is an empty bag
for _, skills := range postings {
	for _, skill := range scalar.Array(skills).All() {
		counts.Insert(skill.Fc())                      // $counts{fc $skill}++
	}
}

for _, skill := range counts.MostCommon(3).All() {   // ties keep the order they were first seen in
	fmt.Println(skill, counts.Count(skill))            // go 3, docker 2, kubernetes 1
}
```

The same with a hash, `Inc` and a sort is in [Sorting](#sorting).

### Building records and writing JSON

Start each record from shared defaults, collect them in an array, and encode
the result.

```go
defaults := scalar.Hash("location", "Unknown", "remote", false)

rows := scalar.Array()
for _, title := range []string{"Go Developer", "Perl Developer"} {
	row := scalar.Hash(defaults, "title", title)   // %row = (%defaults, title => $title)
	rows.Push(scalar.Value(row))                  // push @rows, \%row
}
rows.Get(1).Path("remote").Set(true)             // changes only the second row

out, _ := json.Marshal(rows)
// [{"location":"Unknown","remote":false,"title":"Go Developer"},
//  {"location":"Unknown","remote":true,"title":"Perl Developer"}]
```

Each row is a separate copy of `defaults`, so changing one row changes neither
the others nor `defaults`.

### Storing an embedding vector as bytes

Pack a vector into 4 bytes per float32, for a blob column or a cache, and
unpack it again.

```go
vector := []float64{0.5, -0.25, 0.125}               // from the embedding model

blob := scalar.Pack("f<*", vector)                   // pack("f<*", @vector): little-endian float32s
blob.Length()                                        // 12: 4 bytes each
fmt.Printf("%x\n", blob)                             // 0000003f000080be0000003e, for a log

back := scalar.Unpack("f<*", blob.Bytes()).NumSlice()   // a []float64 again
```

`blob` is a [byte string](#byte-strings), and `blob.Bytes()` is the Go
`[]byte` to store.

## Scalars

Create a scalar with `Value`:

```go
s := scalar.Value("42")
s.Str()    // "42"
s.Int()    // 42
s.Num()    // 42.0
s.Bool()   // true
s.Length() // 2: length($s), counted in characters
```

Strings are turned into numbers the Perl way:

| Value | `Num()` | `NumOK()` |
|-------|---------|-----------|
| `"42"` | `42` | `42, true` |
| `"  42  "` | `42` | `42, true` |
| `"42abc"` | `42` | `42, false` |
| `"abc"` | `0` | `0, false` |
| `""` or undef | `0` | `0, false` |

`Num()` and `Int()` never fail. If you need to know whether a value really was
a number, use `NumOK()` or `IntOK()`.

For a value that may be missing, as Go's `*string` or `*float64` fields
are, use `scalar.To[*float64](s)`: it gives `nil` for undef and a checked
value otherwise. See [Optional values](#missing-null-and-optional-values).

### True and false

These are false: undef, `""`, `"0"`, `0` and `0.0`. Everything else is true,
including `"0.0"`, `"00"` and `" "`.

### Undef

The zero value of `SV` is undef, and so is `scalar.Value(nil)`. Check for it
with `Defined()`, and make a scalar undef with `Undef()`, Perl's `undef $s`,
which returns the scalar, now undef. As in Perl, undef'ing an element of an
array or hash keeps its place: after `h.Get("k").Undef()`, `"k"` still
exists, with an undef value. `Delete` removes it.

Undef is how `scalar` says "nothing here". A read that finds nothing returns
undef rather than failing, and undef passes along a chain of reads, so
`resp.Path("usage", "output_tokens")` is undef if any step is missing. Two
methods supply a default, using Perl's defined-or operators:

```go
tokens := resp.Path("usage", "output_tokens").Or(0).Int()   // ... // 0
meta.Get("location").OrSet("Unknown")                       // $meta{location} //= "Unknown"
```

- **`Or(def)`** is Perl's `//`. It returns a new scalar holding the value, or
  `def` if the value is undef. Unlike `||`, it keeps defined values that are
  false, such as `0` or `""`.
- **`OrSet(def)`** is Perl's `//=`. It sets the value to `def` if it's undef,
  creating a missing hash or array element if necessary, and returns the
  scalar.

### Chains that may come up empty: `Then` and `When`

`Then` and `When` extend the same idea to your own steps. Each passes undef
straight through, without calling its function, so a chain stops quietly at
the first step that finds nothing, and `Or` supplies the default at the end:

```go
annual := posting.At("salary.text").
	Then(parseFigure).   // undef if the text has no figure
	When(isPositive).    // undef unless the figure is above zero
	Then(toAnnual).      // skipped if anything before gave undef
	Or("unknown")
```

- **`s.Then(f)`** calls `f(s)` if `s` is defined, and returns its result as a
  new scalar. `f` may return a scalar or any Go value, and `nil` for undef.
- **`s.When(pred)`** returns a copy of `s` if it's defined and `pred(s)` is
  true, and undef otherwise.

Both return new scalars, so writing to the result never changes the
original or creates a missing element. For readers who know the term:
undef is Maybe's Nothing, `Value` is the unit and `Then` is the bind, and
`TestThenMonadLaws` checks that the monad laws hold.

### Length

`Length()` counts the characters in the string form of the value, like Perl's
`length`. It counts characters, not bytes, so `"café"` has length 4, and a
number is measured as it prints: `scalar.Value(12345).Length()` is 5. The one
difference from Perl is undef: Perl's `length(undef)` is undef, and here it's 0.

### Substrings

`Substr` and `SubstrReplace` are Perl's `substr`. Like `Length`, they count
characters, not bytes.

```go
s := scalar.Value("The black cat climbed the green tree")

s.Substr(4, 5)     // "black"                    substr($s, 4, 5)
s.Substr(14)       // "climbed the green tree"   substr($s, 14)
s.Substr(-4)       // "tree"                     substr($s, -4)
s.Substr(4, -11)   // "black cat climbed the"    substr($s, 4, -11)

s.SubstrReplace(14, 7, "jumped from")            // substr($s, 14, 7, "jumped from")
// returns "climbed"; s is now "The black cat jumped from the green tree"
```

The rules are Perl's:

- A negative offset counts from the end.
- Leaving out the length means "to the end". A negative length leaves that
  many characters off the end.
- If the substring is partly outside the string, you get the part that's
  inside. If it's entirely outside, `Substr` returns undef and
  `SubstrReplace` panics, just as Perl warns or dies.

`Substr` returns a new scalar and never changes the original.
`SubstrReplace` changes the scalar in place and returns the part it replaced,
like Perl's four-argument `substr`. It also does the job of assigning to
`substr`: Perl's `substr($s, 0, 1) = "X"` is `s.SubstrReplace(0, 1, "X")`.

### Searching, case and trimming

| Perl | Go | |
|------|-----|---|
| `index($s, "£")` | `s.Index("£")` | first position of `"£"`, or -1 |
| `index($s, "£", 5)` | `s.Index("£", 5)` | first position at or after 5 |
| `rindex($s, "£")` | `s.RIndex("£")` | last position, or -1 |
| `lc($s)`, `uc($s)` | `s.Lc()`, `s.Uc()` | lower or upper case |
| `ucfirst($s)`, `lcfirst($s)` | `s.UcFirst()`, `s.LcFirst()` | change just the first character |
| `fc($s)` | `s.Fc()` | case-fold, for comparing regardless of case |
| `trim($s)` | `s.Trim()` | remove leading and trailing whitespace |

They all return a new value and leave the scalar alone, so they chain:
`s.Trim().Lc()`.

`Index` and `RIndex` count characters, like Perl. Go's own `strings.Index`
counts bytes, which gives the wrong answer for `Substr` as soon as the text
contains `£`, `é` or `•`. The positions from `Index` can go straight into
`Substr`:

```go
salary := scalar.Value("Salary: £45,000 – £55,000")
salary.Substr(salary.Index("£"), 7)   // "£45,000"
salary.Substr(salary.RIndex("£"))     // "£55,000"
```

The case functions use Unicode's full rules, as Perl does, so
`Uc("straße")` is `"STRASSE"`. To compare text regardless of case, use `Fc`,
not `Lc`. Some words have more than one lower-case form, and `Fc` maps them
all to the same thing:

```go
seen := scalar.Hash()
for _, skill := range skills.All() {
	seen.Get(skill.Fc().Str()).Inc()   // "Go", "GO" and "go" count as one
}
```

`Trim` removes Unicode whitespace too, such as the no-break spaces that HTML
`&nbsp;` turns into.

### Other methods

- `Kind()` tells you what a scalar holds: `Undef`, `Bool`, `Int`, `Float`,
  `String`, `Bytes` (a [byte string](#byte-strings)), `ArrayRef` or
  `HashRef`.
- `Ref()` works like Perl's `ref()`: it returns `"ARRAY"`, `"HASH"` or `""`.
- `Set(x)` assigns a new value and returns the scalar.
- `Copy()` returns a copy of the scalar, and `Clone()` a deep copy, of the
  arrays and hashes it refers to as well. See
  [Copies, references and in-place changes](#copies-references-and-in-place-changes).
- `Any()` returns the scalar as a plain Go value (`nil`, `bool`, `int64`,
  `float64`, `string`, `[]any` or `map[string]any`), for code that doesn't use
  `scalar`. For a value of a type you choose, use `To`: see
  [Into Go values](#into-go-values-to-and-decode).
- `Bytes()` returns the scalar as a Go `[]byte`: a byte string's own bytes,
  or text's UTF-8. See [Byte strings](#byte-strings).
- `String()` is the same as `Str()`, so `fmt.Println(s)` and `%s` print a
  scalar the way Perl's `print` would. A reference prints as Perl's does,
  as `HASH(0xc000123456)`; to see what's inside, use `Dump()`.

### Looking inside: `Dump`

`Dump()` shows a whole structure as `Data::Dumper::Concise` prints it, for
debugging. Scalars, arrays and hashes all have it:

```go
fmt.Println(doc.Dump())   // print Dumper($doc)
```

```
{
  salary => {
    max => undef,
    min => 45000,
  },
  skills => [
    "Go",
    "SQL",
  ],
  title => "Go Developer",
}
```

Keys are sorted, and quoted only when they need to be. Strings are
double-quoted, with escapes for anything that isn't printable ASCII:
`"caf\x{e9}"` for text, and octal, `"\211PNG"`, for a byte string. A
boolean prints as `!!1` or `!!0`, as in Perl 5.36 and later. An array or
hash that appears twice prints in full the first time, then as its path,
such as `$VAR1->{tags}`, so a structure that loops back on itself is safe.
`TestDumpMatchesPerl` checks hundreds of random structures against Perl's
own `Data::Dumper` with Concise's settings. The one difference is that
there's no newline at the end, since `fmt.Println` adds one.

Arrays and hashes also have `String()`, which gives the same on one line, so
`fmt.Println(a)` shows `["Go",1,[2,3]]` and `fmt.Println(h)` shows
`{title => "Go Developer"}`.

### Reading a dump back: `FromDump`

`scalar.FromDump` reads what `Dump` writes, as `FromJSON` reads JSON, and
returns the structure. The round trip is exact: `FromDump(s.Dump())` is equal
to `s`, and dumps the same, with shared parts and loops rebuilt from their
`$VAR1` paths, which JSON can't carry. It also makes test data quick to
write, in the form you'd write it in Perl:

```go
want := scalar.MustFromDump(`{
	title  => "Go Developer",
	skills => ["Go", "SQL"],   # comments are fine
}`)
if d := got.Diff(want); d != "" {
	t.Error(d)
}
```

The [Reference](#reference-scalars) has more examples of `FromDump` and
`MustFromDump`: test data compared with `Diff` or turned into a struct with
`To`, every way of writing keys and strings, byte strings, shared parts and
loops, output pasted from Perl, the types values come back as, and errors.

It reads Perl's own `Data::Dumper` output too, under any of its layout
settings: any indentation, double or single quotes, keys quoted or bare,
with or without trailing commas. A blessed array or hash is read as a plain
one, since `scalar` has no classes. It reads data and never runs any Perl,
so code (`sub { … }`), compiled regexes (`qr/…/`) and references to plain
values (`\'x'`) give an error that says where they are, as does the
`$VAR1 = …;` that `Data::Dumper` writes without `Terse`. Arrays and hashes
may nest up to 10,000 levels deep, as `encoding/json` allows; deeper is an
error, not a crash.

A dump doesn't record types, so values come back with the types their text
gives them, exactly as Perl reads a dump back:

| Written in the dump | Comes back as | So, from `Dump` |
|---|---|---|
| a bare integer of up to 9 digits: `42` | an integer | the string `"42"` comes back as the integer 42 |
| a quoted value: `"2.5"`, `"12345678901"` | a string | a float, and a longer integer, come back as strings |
| a string with an octal or `\xHH` escape above 127, `"\211PNG"`, or raw bytes that aren't UTF-8, as `Data::Dumper` writes binary data without `Useqq` | a [byte string](#byte-strings) | a byte string of plain ASCII comes back as text |
| a string with characters written as they are, in UTF-8: `"café"` | text | the same |
| `!!1`, `!!0`, `undef` | a boolean, undef | the same |

Either way the value compares and calculates the same, as in Perl, so the
structure read back is `Equal` to the one dumped. Where the exact types
matter, use JSON or YAML, which record them.

`TestFromDumpRoundTrip` checks the round trip for hundreds of random
structures, shared parts and loops included, type by type. And
`TestFromDumpMatchesPerl` reads what Perl's `Data::Dumper` writes for random
structures under all 24 combinations of its layout settings, and checks that
each value comes back as what, and as the type, Perl gets by evaluating the
same dump. `TestFromDumpBytesMatchPerl` does the same for byte strings,
escaped or raw.

### Formatting, as with sprintf

Go's format verbs treat a scalar as Perl's `sprintf` does, in the context
each verb implies:

| Verbs | Context | Example |
|---|---|---|
| `%d`, `%b`, `%o`, `%x`, `%X`, `%c` | integer (see `Int`) | `fmt.Sprintf("%d", scalar.Value("42 apples"))` is `"42"` |
| `%e`, `%E`, `%f`, `%F`, `%g`, `%G` | number (see `Num`) | `fmt.Sprintf("%05.2f", scalar.Value("3.14159"))` is `"03.14"` |
| `%t` | boolean (see `Bool`) | `fmt.Sprintf("%t", scalar.Value("0"))` is `"false"` |
| `%s`, `%q`, `%v` and the rest | string (see `Str`) | `fmt.Sprintf("%5s", scalar.Value("ab"))` is `"   ab"` |

```go
fmt.Printf("%d items, £%.2f each\n", count, price)   // printf "%d items, £%.2f each\n", $count, $price
```

Widths, precisions and flags work as usual. As in Perl, `%x` is numeric:
`"ff"` formats as `0`, not as the hex of its bytes.

### Filling in templates: `Interpolate`

`Interpolate` fills in a template from a hash, as Perl interpolates a
double-quoted string, with the hash's keys as the variables:

```go
doc.Interpolate("$title in $location: @skills")   // "Go Developer in Leeds: Go SQL"
doc.Interpolate("Up to £$salary{max}")            // $doc->{salary}{max}
doc.Interpolate("Up to £${salary.max}")           // the same: any At path inside ${...}
```

| In the template | Gives |
|---|---|
| `$name` | the value of that key |
| `$name{key}`, `$name[0]`, `$name[0]{key}` | subscripts straight after a name are part of it, as in Perl, written as for `At` |
| `${path}` | any `At` path, dots included: `${salary.max}` |
| `@name`, `@{path}` | an array's elements, joined with a space, as Perl's `"@a"` joins them |
| `\$`, `\@` | a literal `$` or `@`, as in Perl |

As in Perl, `$title.x` is the title followed by `.x`, and `${title}s` puts
text straight after a value. A `$` or `@` not followed by a name or `{`, as
in `$5`, is literal, but `@example` in an email address is an array, so
write `me\@example.com`, as in Perl. A missing value gives `""`, and nothing
is created. A reference gives `ARRAY(0x…)`, as in Perl, so use `@` for an
array.

A template string is parsed the first time it's used, and a malformed one
panics, as a bad regex does. To parse one up front, use `scalar.MustTemplate`,
or `scalar.CompileTemplate` for one that comes from outside the program, which
returns an error instead. A compiled template can also change the separator
for lists, as `local $" = ", "` does:

```go
var summary = scalar.MustTemplate("$title. Skills: @skills.").ListSeparator(", ")

doc.Interpolate(summary)   // "Go Developer. Skills: Go, SQL."
```

There are no conditionals or loops: for text that leaves out lines with
nothing in them, as `embedding.Text` does, use Go's `text/template`, or
build the text in code.

## Arrays

```go
a := scalar.Array(3, "10", 2)  // @a = (3, "10", 2)

a.Push(1)                      // push @a, 1
a.Unshift(0)                   // unshift @a, 0
last := a.Pop()                // pop @a
first := a.Shift()             // shift @a

a.Get(0)                       // $a[0]
a.Get(-1)                      // $a[-1]
a.Set(5, "x")                  // $a[5] = "x"
a.Length()                     // scalar(@a)
a.Scalar()                     // scalar(@a), same as Length
a.LastIndex()                  // $#a
a.SetLastIndex(2)              // $#a = 2   (truncate or extend)
a.Exists(3)                    // exists $a[3]
a.Defined(3)                   // defined $a[3]
a.Delete(3)                    // delete $a[3]
a.Undef()                      // undef @a     (emptied in place: every reference to a sees it empty)
a.Values()                     // values @a
a.Values(0, -1)                // @a[0, -1]    (the elements themselves, as Perl aliases them)
a.Pairs(0, -1)                 // %a[0, -1]    (index/value pairs: 0, $a[0], -1, $a[-1])
a.Splice(1, 2, "x", "y")       // splice @a, 1, 2, "x", "y"
a.PopN(2)                      // splice(@a, -2)    (the last 2, in order; all of them if fewer)
a.ShiftN(2)                    // splice(@a, 0, 2)  (the first 2; all of them if fewer)
head, tail := a.SplitAt(2)     // my ($x, $y, @tail) = @a   (see below)

a.Join(", ")                   // join(", ", @a)
a.Sort(nil)                    // sort @a                  (as strings)
a.Sort(scalar.NumCmp)          // sort { $a <=> $b } @a    (as numbers)
a.Reverse()                    // reverse @a
a.SortInPlace(nil)             // @a = sort @a             (sorts a itself)
a.ReverseInPlace()             // @a = reverse @a          (reverses a itself)
a.Map(func(s *scalar.SV) any { return s.Int() * 2 })  // map  { $_ * 2 } @a
a.Grep(func(s *scalar.SV) bool { return s.Int() > 2 }) // grep { $_ > 2 } @a

for i, s := range a.All() {
	fmt.Println(i, s)
}
```

`Sort`, `Reverse`, `Map`, `Grep` and `SplitAt` return new arrays and leave
the original alone. `Values` and `Pairs` return new arrays too, but of the
elements themselves (see [Hashes](#hashes)). `Splice`, `PopN` and `ShiftN`
return the elements they removed.

### Splitting off the first few elements

Perl's `my ($one, $two, $three, @tail) = @a` splits a list into its first few
elements and the rest. Go can't return a different number of values depending
on an argument, so `SplitAt` returns the head as one array, and `SplitAtSlice`
returns it as a Go slice you can unpack. (The name comes from Haskell's
`splitAt`, which also returns the first `n` elements and the rest.)

```go
head, tail := a.SplitAt(3)          // head and tail are both arrays

h, tail := a.SplitAtSlice(3)        // h is a []*scalar.SV
one, two, three := h[0], h[1], h[2]
```

As with Perl's list assignment:

- **The head always has exactly `n` elements.** If the array is shorter, the
  missing ones are undef, so `h[2]` is always safe to read.
- **Everything is a copy,** and the original array isn't changed. To remove
  the elements as well, use `Shift` or `Splice(0, n)`.
- **A negative count panics,** as `make` does in Go.

### Back to plain Go

To turn an array into a Go slice of any type, checking each element, use
`scalar.To`: `scalar.To[[]int](a)`. See
[Into Go values](#into-go-values-to-and-decode).

Three methods convert every element as Perl's string or numeric context
would instead, and never fail:

| Method | Returns | Each element |
|---|---|---|
| `a.StrSlice()` | `[]string` | as a string: undef and holes become `""` |
| `a.IntSlice()` | `[]int64` | as an integer: anything that isn't a number becomes `0` |
| `a.NumSlice()` | `[]float64` | as a number |

```go
skills := scalar.Array(raw).Grep(notBlank).StrSlice()   // back to a []string
scalar.Array("3", "five").IntSlice()                     // [3 0]: "five" is 0, as in Perl
scalar.To[[]int](scalar.Array("3", "five"))              // an error: [1]: want int, got "five"
```

The results are copies, so changing them doesn't change the array.

### Lists flatten, as in Perl

Wherever Perl takes a list, arrays and hashes passed in are flattened:
arrays add their elements, and hashes add their key/value pairs. This applies
to `Push`, `Unshift`, the values inserted by `Splice`, and the `Array(...)`
and `Hash(...)` constructors. Go slices and maps flatten the same way, so you
don't need `...` to pass one.

```go
a.Push(b)                       // push @a, @b
a.Push(h)                       // push @a, %h        (key/value pairs)
a.Push(h.KeysSlice())           // push @a, keys %h
all := scalar.Array(a, b)       // my @all = (@a, @b)
```

A scalar always adds one element, even when it holds a reference. That's how
you nest an array or hash, as with Perl's `\@b` and `\%h`:

```go
a.Push(scalar.Value(b))        // push @a, \@b      (one element: a reference)
rows.Push(scalar.Value(row))   // push @rows, \%row (a list of records)
```

`Map` flattens too, as Perl's `map` does, so the function can return several
values for each element, or none:

```go
a.Map(func(s *scalar.SV) any { return scalar.Array(s, s.Mul(10)) })   // map { ($_, $_ * 10) } @a
```

Values are always copied in, as Perl copies list values.

### List utilities

These follow Perl's `List::Util`. The ones that test elements take a function,
where Perl takes a block:

| Perl (`List::Util`) | Go |
|------|-----|
| `any { ... } @a` | `a.AnyMatch(func(s *scalar.SV) bool { ... })` |
| `all { ... } @a` | `a.AllMatch(func(s *scalar.SV) bool { ... })` |
| `none { ... } @a` | `a.NoneMatch(func(s *scalar.SV) bool { ... })` |
| `first { ... } @a` | `a.FirstMatch(func(s *scalar.SV) bool { ... })` |
| `max @a`, `min @a` | `a.MaxNum()`, `a.MinNum()` |
| `maxstr @a`, `minstr @a` | `a.MaxStr()`, `a.MinStr()` |
| `uniqstr @a` | `a.UniqueStr()` |
| `uniqnum @a` | `a.UniqueNum()` |
| `sum @a`, `sum0 @a` | `a.Sum()`, `a.Sum0()` |
| `product @a` | `a.Product()` |
| `reduce { ... } @a` | `a.Reduce(func(acc, x *scalar.SV) any { ... })` |

```go
remote := func(s *scalar.SV) bool { return s.M(`(?i)\bremote\b`) != nil }
if bullets.AnyMatch(remote) { ... }

salary := chunks.FirstMatch(func(s *scalar.SV) bool { return s.Index("£") >= 0 })

skills.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr()   // ignoring case

total := salaries.Sum0()                                             // sum0 @salaries
longest := words.Reduce(func(a, b *scalar.SV) any {                  // reduce { length $b > length $a ? $b : $a } @words
	if b.Length() > a.Length() {
		return b
	}
	return a
})
```

As in Perl:

- **Empty arrays:** `AllMatch` and `NoneMatch` are true, and `AnyMatch` is false.
- **`AnyMatch` and `FirstMatch` stop at the first match.**
- **Nothing found:** `FirstMatch` and the max and min methods return undef.
  Otherwise, like `Get`, they return the element itself, not a copy.
- **`MaxNum` and `MinNum` compare as numbers,** so `"10"` beats `9`.
  **`MaxStr` and `MinStr` compare as strings,** so `"9"` beats `"10"`. All
  four keep the first of equal elements. As in `List::Util`, a NaN never wins
  a comparison, so `MaxNum` only returns one if it comes first.
- **`UniqueStr` and `UniqueNum` keep the first of each value, in order,** and
  return copies. `UniqueStr` compares as strings, with undef counting as `""`.
  `UniqueNum` compares as numbers, so `1`, `"1"` and `"1.0"` count as the same
  value. Large integers compare exactly.
- **`Sum` of an empty array is undef; `Sum0` is 0, and `Product` is 1.**
  They add and multiply as `Add` and `Mul` do, so integers stay exact while
  the result fits, and become floats when it doesn't.
- **`Reduce`** calls its function with the first two elements, then with
  the result so far and each following element, as Perl's `$a` and `$b`.
  With one element it returns a copy of it, without calling the function;
  with none, undef. The function may return a scalar or any Go value.

### By a key: `List::UtilsBy`

These follow CPAN's `List::UtilsBy`. Each takes a function that gives a key
for an element, calls it once for each element, and works on the keys:

```go
salary := func(p *scalar.SV) any { return p.At("salary.max") }

postings.SortByNum(salary)        // nsort_by { $_->{salary}{max} } @postings
postings.RevSortByNum(salary)     // rev_nsort_by: the best paid first
postings.MaxBy(salary)            // max_by: the best paid posting itself
postings.PartitionBy(func(p *scalar.SV) any { return p.At("seniority") })   // partition_by
postings.CountBy(func(p *scalar.SV) any { return p.At("location") }).MostCommon(5)
```

| `List::UtilsBy` | Go | Gives |
|---|---|---|
| `sort_by`, `nsort_by` | `SortByStr`, `SortByNum` | a sorted copy, comparing the keys as strings or as numbers |
| `rev_sort_by`, `rev_nsort_by` | `RevSortByStr`, `RevSortByNum` | the same, largest first |
| `max_by`, `min_by` | `MaxBy`, `MinBy` | the element with the largest or smallest numeric key, itself, or undef if the array is empty |
| `uniq_by` | `UniqueBy` | copies of the first element with each key, as a string |
| `partition_by` | `PartitionBy` | a hash from each key to an array of copies of the elements with it |
| `count_by` | `CountBy` | a [bag](#sets-and-bags) of the keys, so `MostCommon` lists the commonest first; `Counts()` gives Perl's hash |
| `extract_by` | `ExtractBy` | the elements for which a test is true, removed from the array |

As in Perl:

- **The key function is called once for each element,** not once for each
  comparison, so an expensive key costs one call per element.
- **Equal keys keep their order,** whichever way the sort goes. So
  `RevSortByNum(f)` isn't the same as `SortByNum(f).Reverse()`, which would
  reverse elements with equal keys too.
- **`MaxBy` and `MinBy` compare numbers,** and return the first of several
  elements with the same key.
- **`ExtractBy` changes the array,** and its function returns a bool, not a
  key. The elements it returns are the originals, moved out.

`TestListByMatchesPerl` checks all of these against `List::UtilsBy` itself,
for hundreds of random lists, when it's installed.

### Sorting

`Sort` takes any function of the form `func(x, y *scalar.SV) int`. Closures
work too. The function returns a negative number, zero or a positive number,
just as Perl's `<=>` and `cmp` do, and `x` and `y` play the parts of `$a` and
`$b`. `scalar.StrCmp` is `cmp` and `scalar.NumCmp` is `<=>`, so you can build
on them:

```go
// sort { $b <=> $a } @a   (descending: swap the arguments)
a.Sort(func(x, y *scalar.SV) int { return scalar.NumCmp(y, x) })

// sort { $scores{$b} <=> $scores{$a} } keys %scores   (highest score first)
scores := scalar.Hash("ann", 72, "bob", 95, "cat", 81)
byScore := scores.Keys().Sort(func(x, y *scalar.SV) int {
	return scalar.NumCmp(scores.Get(y.Str()), scores.Get(x.Str()))
})
// bob, cat, ann
```

To sort on more than one key, use `cmp.Or` from Go's standard library. It
returns the first comparison that isn't zero, just like chaining comparisons
with `or` in Perl:

```go
// sort { $a->{dept} cmp $b->{dept} or $b->{salary} <=> $a->{salary} } @people
sorted := people.AV().Sort(func(x, y *scalar.SV) int {
	return cmp.Or(
		scalar.StrCmp(x.Path("dept"), y.Path("dept")),     // by department,
		scalar.NumCmp(y.Path("salary"), x.Path("salary")), // then highest salary
	)
})
```

Some things to know:

- **A Go-style "less than" function won't compile.** One that returns `bool`
  doesn't fit. The function must return an `int`.
- **Equal elements keep their order.** The sort is stable, as Perl's default
  sort is in practice.
- **The comparison must be consistent.** As in Perl, if it contradicts itself,
  the order you get is unspecified. It won't crash, though.
- **`Sort` returns a sorted copy; `SortInPlace` sorts the array itself.**
  `SortInPlace` is Perl's `@a = sort @a`: it takes the same comparison
  function, returns the array so calls can chain, and saves copying a large
  array. As with assigning a list back in Perl, any holes become undef
  elements. `ReverseInPlace` is the same for `Reverse`.

### Negative indices

Negative indices count from the end, everywhere an index is accepted:
`Get(-1)` is the last element, and `Splice(-2, 1)` removes the second to last.

As in Perl, you can read before the start of an array, but you can't write
there. `a.Get(-10)` on a short array is undef, but `a.Set(-10, "x")` panics
with Perl's message: *Modification of non-creatable array value attempted,
subscript -10*.

### Holes

Setting an element past the end, as in `a.Set(5, "x")` on a short array, leaves
a gap. The elements in the gap read as undef, but they don't exist: `Exists`
is false for them, just as in Perl. `Delete` also leaves a hole, unless it
removes the last element, in which case the array shrinks.

`Splice` takes a count, not an end position. A count that runs past the end
stops at the end, so `a.Splice(2, a.Length())` removes everything from index 2.

## Hashes

```go
h := scalar.Hash("name", "Ada", "age", 36)  // %h = (name => "Ada", age => 36)
merged := scalar.Hash(defaults, "age", 37)   // %merged = (%defaults, age => 37)

h.Get("name")          // $h{name}
h.Set("lang", "Go")    // $h{lang} = "Go"
h.Push("a", 1, "b", 2) // $h{a} = 1; $h{b} = 2   (pairs, another hash or a Go map; later keys win)
h.Exists("age")        // exists $h{age}    (the key is there, even if its value is undef)
h.Defined("age")       // defined $h{age}   (the key is there and its value isn't undef)
h.Delete("age")        // delete $h{age}
h.Undef()              // undef %h          (emptied in place: every reference to h sees it empty)
h.Length()             // scalar(keys %h)
h.Scalar()             // scalar(%h), same as Length
h.Map(func(k string, v *scalar.SV) (string, any) { return k, v.Mul(2) })
                       // %new = map { $_ => $h{$_} * 2 } keys %h   (a new hash; h is unchanged)

h.Keys()            // keys %h
h.Values()          // values %h
h.Values("a", "b")  // @h{qw(a b)}   (a hash slice)
h.Pairs()           // %h            (key/value pairs, as a list)
h.Pairs("a", "b")   // %h{qw(a b)}   (a key/value slice)

h.KeysSlice()       // keys %h,   as a Go []string
h.ValuesSlice()     // values %h, as a Go []*scalar.SV
h.PairsMap()        // %h,        as a Go map[string]*scalar.SV

for k, v := range h.All() {
	fmt.Println(k, v)
}

for k, v, ok := h.Each(); ok; k, v, ok = h.Each() {   // while (my ($k, $v) = each %h)
	fmt.Println(k, v)
}
```

`Each` is Perl's `each`, gotchas and all: the hash remembers how far it has
got, so leaving the loop early means the next `Each` carries on from there,
and `Keys`, `Values` and `Pairs` (and their Go forms) reset it, as `keys`,
`values` and `%h` in list context do. `All` has none of that: each loop
starts afresh, and stopping early is clean. Prefer `All` unless you need
`each`'s behaviour. Arrays have both too: `a.Each()` is Perl 5.12's
`each @a`.

`Hash(...)` takes a list, so you can also build a hash from Go maps, from a
list of maps, or from other hashes. Later keys win, as in Perl:

```go
scalar.Hash(goMap)                      // %h = %$goMap
scalar.Hash([]map[string]any{m1, m2})   // m1 and m2 merged
scalar.Hash(defaults, overrides)        // %h = (%defaults, %overrides)
scalar.Hash(map[int]string{1: "one"})   // keys become strings: {"1": "one"}
```

Only the top level is merged. A value that is itself a slice or map is
stored as a reference, so it stays nested.

`Hash(h, ...)` makes a new merged hash. To merge into `h` itself, use
`h.Push(...)`, which takes the same kinds of list. It assigns each pair as
`$h{k} = v` does, so a key that's already there keeps its scalar, updated in
place.

`Values` and `Pairs` take keys, as Perl's slices do, and the keys can be
strings, scalars or arrays of them: `h.Values(names)` is `@h{@names}`. That's
an addition to Perl's `values`, which takes no keys, but it keeps one name
for "the values" whichever ones you want.

`Pairs` gives a list of pairs, so building a hash from some of another's
keys is one step:

```go
h1 := scalar.Hash(h2.Pairs("k1", "k2", "k3"))   // %h1 = %h2{qw(k1 k2 k3)}
```

As in Perl, every key asked for is included, in that order: a missing key
comes with an undef value, so `h1` has all three keys, though `h2` isn't
changed. Note that `List::Util`'s `pairs` is different: it gives `[key, value]`
array references, where `Pairs` gives one flat list.

`Keys` and `Values` return arrays, so you can sort, map, filter or join them
straight away:

```go
h.Keys().Sort(nil).Join(", ")   // join ", ", sort keys %h
```

`KeysSlice` and `ValuesSlice` return ordinary Go slices instead, for passing to
code that doesn't use `scalar`.

As in Perl, the order of keys is not fixed. `Keys`, `Values`, `KeysSlice`,
`ValuesSlice` and `All` always use the same order as each other, until you add
or delete a key.

Also as in Perl, keys are copies but values are not. Changing an element of
`Keys()` has no effect on the hash. Changing an element of `Values()` or
`Pairs()`, with keys or without, changes the value in the hash, because it
*is* that value. A missing key reads as undef, and is created only if you
write to it, as with `Get`:

```go
for _, v := range h.Values().All() {   // $_ *= 2 for values %h
	v.Set(v.Mul(2))
}
```

Adding or removing elements of the `Values()` array (with `Push`, `Pop` and so
on) doesn't add or remove anything in the hash. For a value of your own, call
`Copy()` on it. Arrays work the same way: `a.Values(0, -1)` gives the first and
last elements themselves.

## Sets and bags

Perl has no set type of its own: the usual idiom is a hash used only for its
keys. CPAN's `Set::Scalar` and `Set::Bag` give the familiar operations, and
`Set` and `Bag` follow them.

```go
mine := scalar.NewSet("Go", "Perl", "SQL")   // Set::Scalar->new(...)
wanted := scalar.NewSet("Go", "SQL", "Rust")
mine.Intersection(wanted)                    // (Go SQL): $mine->intersection($wanted)
wanted.Difference(mine)                      // (Rust)
mine.Has("Perl")                             // $mine->has("Perl")

var skills scalar.Bag                        // the zero value is empty, ready to use
for _, posting := range postings {
	skills.Insert(posting.Skills)            // count every skill across postings
}
skills.MostCommon(10)                        // the 10 most wanted, most common first
skills.Count("Go")                           // $skills->grab("Go")
```

| Set::Scalar | `Set` | | Set::Bag | `Bag` |
|---|---|---|---|---|
| `new(...)` | `NewSet(...)` | | `new(a => 3)` | `NewBag().InsertN("a", 3)` |
| `insert`, `delete` | `Insert`, `Delete` | | `insert(a => n)`, `delete(a => n)` | `InsertN`, `DeleteN` (or `Insert`, `Delete` for one each) |
| `has`, `size`, `members` | `Has`, `Length`, `Members` | | `grab(a)`, `elements` | `Count`, `Members` |
| `union`, `intersection`, `difference`, `symmetric_difference` | `Union`, `Intersection`, `Difference`, `SymmetricDifference` | | `sum`, `union`, `intersection`, `difference` | `Sum`, `Union`, `Intersection`, `Difference` |
| `is_subset`, `is_proper_subset`, `is_superset`, `is_equal`, `is_disjoint` | `IsSubset`, `IsProperSubset`, `IsSuperset`, `IsEqual`, `IsDisjoint` | | | `Total`, `Counts` (a hash), `MostCommon` |

- **Members are compared as strings,** as Perl's hash keys are, and both
  modules: `1`, `"1"` and `1.0` are one member, while `"1.0"`, `"Go"`
  and `"go"` are each different. The first value inserted is the one kept.
  For a case-insensitive set, insert `s.Fc()`.
- **Lists flatten,** as everywhere in `scalar`: `NewSet(array)`,
  `skills.Insert(goSlice)`.
- **Set operations return new sets and bags,** leaving the originals alone.
  `Insert` and `Delete` change the set or bag, and return it.
- **Members are copies,** so changing a value after inserting it, or
  changing an element of `Members()`, doesn't change the set.
- **A bag's counts never go below zero.** Removing more than there is removes
  what there is, as `Set::Bag` does with its `over_delete` option. Without
  that option `Set::Bag` dies, which would make counting code fragile.
- **`Union` and `Intersection` for bags** take the larger and smaller count
  of each member; `Sum` adds them.

Two additions to the modules:

- **Members keep the order they were first inserted in,** so `Members`,
  `All`, `MostCommon` ties and JSON are reproducible. Printing a set or bag
  (`String`, `fmt.Println`) sorts the members, as the modules print them:
  `(a b c)` and `(apples => 3, pears => 1)`.
- **Empty sets are disjoint.** `Set::Scalar` says two empty sets aren't;
  mathematically they are, since they have nothing in common, and `IsDisjoint`
  follows the mathematics.

A set encodes as a JSON array, and a bag as an object from member to count.
For Go code that doesn't use `scalar`, `To` converts them: a set gives its
members and a bag its counts, so `scalar.To[[]string](set)` is a `[]string` of
the members, in insertion order, and `scalar.To[map[string]int](bag)` a map of
the counts. A bag's members, without their counts, are
`scalar.To[[]string](bag.Members())`. The results are new, so changing them
doesn't change the set or bag. `Members()` keeps the original values instead,
so a number stays a number.

Sets and bags can't be stored inside a scalar, as arrays and hashes can,
because Perl has no references to them: to store one in a hash, store its
`Members()` (or a bag's `Counts()`).

## Nesting and autovivification

Arrays and hashes hold scalars, so you nest them through references, as in
Perl. `AV()` and `HV()` dereference a scalar:

```go
h.Get("tags").AV().Push("go", "perl")  // push @{ $h{tags} }, "go", "perl"
```

If the scalar is undef, `AV()` and `HV()` create a new array or hash on the
spot. This is called autovivification. It lets you build deep structures in
one line:

```go
h := scalar.Hash()
h.Get("x").HV().Get("y").AV().Get(2).HV().Set("z", 1)  // $h{x}{y}[2]{z} = 1
```

The rules are the same as Perl's:

- Reading a missing key or index gives undef and doesn't create anything.
- Writing to it creates it.
- Going *through* a missing key creates it, even if you are only reading.
  `h.Get("a").HV().Get("b")` creates `$h{a}` but not `$h{a}{b}`.

Reading a missing key does cost something: it gives a new undef scalar,
ready to create the key if you write to it. To ask only whether a key is
there, as in a loop over many records, `h.Exists(k)` costs nothing:
`m.HV().Exists("k")` rather than `m.At("k").Defined()`.

`Path(...)` walks a mix of hash keys (strings) and array indexes (ints), and
`At` takes the same route as one string. Both follow the same rules and
replace a chain of `AV()` and `HV()`. They're methods of scalars, hashes
and arrays alike, so `h.At("a.b")` is `$h{a}{b}` and `a.At("[0].name")` is
`$a[0]{name}`; writing through one changes that hash or array:

```go
var doc scalar.SV
doc.Path("x", "y", 2, "z").Set(1)   // $doc->{x}{y}[2]{z} = 1
doc.At("x.y[2].z").Set(1)           // the same
doc.At("{x}{y}[2]{z}").Set(1)       // the same, written as in Perl
```

### A path as one string: `At`

`At` takes the whole route as one string, in dot, bracket or Perl's brace
form, and walks it exactly as `Path` does:

```go
doc.At("content[0].rule_set")    // $doc->{content}[0]{rule_set}
doc.At("{content}[0]{rule_set}") // the same, written as in Perl
doc.At("content.0.rule_set")     // the same, when content is an array
doc.At("items[-1].name")         // negative indices count from the end
doc.At("meta{log.level}")        // a key with dots in it, in braces
doc.At(`meta["log.level"]`)      // or quoted
doc.At("rules[0].name").Set("x") // writing autovivifies
v, ok := doc.AtOK("salary.max")  // as PathOK: present? without creating anything
doc.DeleteAt("salary.bonus")     // delete $doc->{salary}{bonus}
```

`DeletePath` is `DeleteAt` with the route as `Path` takes it, for keys that
come from data and might contain dots or brackets:
`doc.DeletePath(section, key)` is `delete $doc->{$section}{$key}`. Both
remove only the last step, and create nothing on the way, so a missing
path deletes nothing.

| Step | Means |
|---|---|
| `name` | a hash key |
| `[0]`, `[-1]` | an array index |
| `{key}` | a hash key, always, as in Perl: everything up to the `}`, dots and spaces included, so `{0}` is the key `"0"` |
| `["key"]`, `['key']`, `{"key"}`, `{'key'}` | a hash key, however it's spelt |
| `.0` (a number after a dot) | an index if the container is an array, the key `"0"` if it's a hash; writing to nothing yet creates an array |

So `{0}` or `["0"]` is how to say "the hash key 0" unambiguously. A
backslash escapes the next character in a key, as in `["say \"hi\""]` or
`{a\}b}`, and the forms mix freely: `content[0]{tags}.1`. A leading `$` is
allowed, as in JSONPath (`$.content[0]`), and an empty path is the scalar
itself. A malformed path panics, as a bad regex does.

`DeleteAt` removes what a path leads to, and returns its value, or undef if
there was nothing there. As with `HV.Delete` and `AV.Delete`, an element in
the middle of an array leaves a hole, and deleting the last shrinks the
array. Unlike Perl, which creates the levels on the way to what it deletes,
`DeleteAt` never creates anything, so deleting a path that isn't there leaves
the data exactly as it was.

`Path` itself never splits strings: `doc.Path("log.level")` is the single
key `log.level`, as it should be, since real keys contain dots.

For a path used often, or one built from parts, compile it once with
`scalar.MustPath`, and pass it to `Path` or `PathOK` like any key, as
`qr//` is passed to a match:

```go
var rule = scalar.MustPath("content[0].rule_set")

doc.Path(rule)              // doc.At("content[0].rule_set")
doc.Path(rule, "name", 2)   // ...{name}[2]
```

`scalar.CompilePath` does the same but returns an error instead of
panicking, for paths that come from outside the program.

### Missing or null?

Both a missing key and a JSON `null` read as undef. To tell them apart, use
`PathOK`, which adds Go's comma-ok:

```go
v, ok := resp.PathOK("salary", "max")
switch {
case !ok:
	// the model left the field out (or something on the way is missing)
case !v.Defined():
	// the field is there, but null
default:
	// a real value
}
```

`ok` is false if any step doesn't exist, including stepping through something
that isn't an array or hash. Unlike `Path`, `PathOK` never creates anything.
When `ok` is true, `v` is the stored value itself, as with `Path`. For a
single key, `h.Exists(k)` does the same job.

There are two things Perl refuses to create, and `scalar` refuses them too:

- **An array element before the start of the array**, such as `$a[-10]` on a
  three-element array.
- **An element of something that isn't an array or hash**, such as using a
  string as if it were a hash.

For both, reading gives undef, even if you read further through them with
`AV()`, `HV()` or `Path`. Writing panics, just as Perl dies.

Strict Perl dies on some of these *reads* as well. `scalar` lets reads carry
on, because a read that can't panic is what makes `Path` safe on messy data.

## Finding values with JSONPath

`Path` reads one value at a location you know. `Find` searches with a
JSONPath query ([RFC 9535](https://www.rfc-editor.org/rfc/rfc9535)) and
returns every match as an array, or `nil` if nothing matched:

```go
resp.Find("$.content[*].text")    // the text of every content block
meta.Find("$..skills[*]")         // every skill, at any depth
doc.Find("$.items[-1].name")      // the last item's name
doc.Find("$.items[0:3]")          // the first three items
doc.Find("$['salary-min']")       // keys that aren't plain words need brackets
```

| JSONPath | Selects |
|----------|---------|
| `$` | the value you call `Find` on |
| `.name` or `['name']` | a key in a hash |
| `[0]`, `[-1]` | an element of an array (negative counts from the end) |
| `[start:end:step]` | a slice of an array, as in Python |
| `.*` or `[*]` | every element of an array or hash |
| `..name`, `..*`, `..[0]` | the same, at any depth |
| `[0,2]`, `['a','b']` | several at once |
| `[?@.price < 10]` | the elements (or a hash's values) the filter is true for; see below |

Each match is a scalar. Strings, numbers, booleans and nulls come back as plain
values. Objects and arrays come back as references, so you can continue from
them with `HV()`, `AV()`, `Path` or `Find`.

Like `Get` and `Path`, the matches are the stored values themselves, so you
can change data where it sits:

```go
for _, s := range meta.Find("$.requiredSkills[*]").All() {
	s.Set(s.Trim())   // trims each skill inside meta
}
```

### Filters

A filter, `[?...]`, selects the children of an array or hash for which an
expression is true. In it, `@` is the child, and `$` the value `Find` was
called on:

```go
jobs.Find("$[?@.salary.min >= 30000].title")              // a comparison
jobs.Find("$[?@.salary.max]")                             // has a maximum
jobs.Find("$[?!@.remote]")                                // hasn't "remote" at all
jobs.Find("$[?@.salary.min > $.threshold]")               // against the root
jobs.Find("$[?@.type == 'contract' && @.days <= 5]")      // && || ! and ( )
jobs.Find("$[?count(@.skills[*]) > 2]")                   // functions
resp.Find("$.content[?@.type == 'tool_use'].input")       // the tool_use block in a Claude response
```

| In a filter | Means |
|---|---|
| `@.a`, `$.a` on their own | the query selects something: `@.remote` is true even if it's `false` |
| `==`, `!=`, `<`, `<=`, `>`, `>=` | a comparison, of literals, queries that select one node at most (names and indexes only), or functions returning a value |
| `'text'`, `"text"`, `42`, `-1.5e3`, `true`, `false`, `null` | literals, written as in JSON |
| `&&`, `\|\|`, `!`, `( )` | and, or, not, grouping |
| `length(v)` | a string's characters, an array's elements or a hash's keys |
| `count(q)` | how many nodes a query selects |
| `value(q)` | the value a query selects, if it selects one |
| `match(s, 're')`, `search(s, 're')` | whether all of `s`, or some of it, matches an I-Regexp |

**Comparison goes by JSON type**, as RFC 9535 says, not by Perl's string and
numeric contexts: `1 == 1.0`, but `1 != "1"` and `1 != true`. Two integers
compare exactly, past float64's 2⁵³, as `NumEq` does. `<` and the like
compare two numbers, or two strings by their characters, and nothing else.
A byte string's characters are Latin-1, one a byte, as `Eq` and `Length`
have them, in comparisons, `length` and the regular expressions. A query that selects nothing is Nothing, which equals only Nothing:
`@.min == @.max` is true when neither is there. Arrays and hashes are equal
element by element, but can't be written as literals. Data built in Go
compares as it's stored: `Value("1")` is a string.

**Regular expressions are I-Regexp** (RFC 9485), a small, safe common
subset: no `\d` or `\w` (use `[0-9]`, `\p{L}`), no lazy quantifiers, no
`(?...)` groups or flags, no backreferences; braces only as `{n}`, `{n,}` or
`{n,m}`; `\p{..}` only for Unicode categories (`L`, `Lu`, `Nd`, …). `Find`
takes exactly what RFC 9485's grammar does, though Go's regexp would take
more. `.` is any character but a line break. A repeat over 1,000 (`a{1001}`)
is beyond Go's regexp, and so false. A pattern that isn't
I-Regexp makes `match` and `search` false, as the RFC says, rather than an
error. One point the RFC leaves open: its grammar makes `^` and `$`
ordinary characters, but its own translation to Go's regexp leaves them as
anchors, and the compliance suite expects them to anchor; `Find` follows the
translation and the suite.

**Mistakes in a filter are caught when it's parsed,** as the RFC's type rules
require, and panic like any malformed path: `@.* == 1` (a query that could
select several), `?length(@.a)` on its own (a value, not true or false),
`match(@, 'x') == true` (true or false, not a value to compare), an unknown
function, or the wrong number of arguments.

**A value from Go goes in through `QuoteJSONPath`,** which writes it as a
quoted, escaped literal, as `regexp.QuoteMeta` does for a pattern. Building
the quotes by hand breaks on an apostrophe, and a value such as
`']) || true || ('` would change the query:

```go
doc.Find("$.sections[?@.kind == " + scalar.QuoteJSONPath(kind) + "].lines[*]")
doc.Find("$[" + scalar.QuoteJSONPath(key) + "]")   // a key that isn't a plain word
```

For a condition the language can't say, the list utilities on the result are
plain Go:

```go
input := resp.Find("$.content[*]").FirstMatch(func(b *scalar.SV) bool {
	return b.Path("type").Eq("tool_use") && b.At("input.query").Defined()
}).Path("input")
```

### Things to know

- **`Find` never creates anything.** Unlike `Path`, a missing key or index
  simply matches nothing.
- **A hole in an array is undef,** as `MarshalJSON` writes it (`null`): `[*]`
  finds it, `?@ == null` matches it. The undef found is a fresh one, so the
  hole stays a hole, and writing to it panics.
- **Hash order is unspecified,** as in JSONPath and Perl. Matches from `[*]`
  or `..` on a hash come back in the hash's own order.
- **A malformed path panics,** like a bad regex. What's malformed is what
  RFC 9535 says is: an index with a leading zero (`[01]`) or `-0`, or past
  ±(2⁵³−1); a raw control character in a quoted name, or the other quote
  escaped (`['a\"b']`: write `['a"b']`); a space after the last segment.
  Nesting deeper than 1,000 is an error too, rather than a stack overflow.
- **It passes the [JSONPath Compliance Test Suite](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite),**
  all 715 cases (`TestCompliance`, with the suite in `testdata/cts`).
- **A part in two places is found in each,** as it would be after a JSON
  round trip: `..` goes down every branch. **A reference that loops back** to
  one of its ancestors isn't followed again, so `..` can't get stuck, and a
  filter comparing structures that loop compares them safely, as `Equal`
  does.

## Changing every value: `Apply`

`Apply` calls a function on every plain value in a structure, all the way
down, and stores what it returns in the value's place, as `Data::Rmap`'s
`rmap` does. Scalars, arrays and hashes all have it:

```go
doc.Apply(func(v *scalar.SV) any {   // rmap { $_ = trim $_ } $doc
	if v.Kind() == scalar.String {
		return v.Trim()
	}
	return v                         // anything else as it is
})
```

- **The function sees plain values:** strings, numbers, booleans, byte
  strings and undef. Arrays and hashes are walked through, not passed to it,
  and hash keys are left alone.
- **What it returns replaces the value.** Returning `v` leaves it as it was,
  and `nil` makes it undef. An array or hash returned is stored as a
  reference, and isn't walked.
- **It changes the structure in place,** and returns it. For a changed
  copy, use `doc.Clone().Apply(f)`.
- **Each array or hash is visited once,** as with `rmap`, so shared parts
  and references that loop back are safe.
- **Holes are skipped.** That's one difference from `rmap`, which visits an
  array's holes as undef and so fills them in.

## Comparing structures: `Equal` and `Diff`

`Equal` compares two structures all the way down, as `Test::More`'s
`is_deeply` does, and `Diff` says where they first differ, with a path for
`At`. Scalars, arrays and hashes all have both, and the other side can be a
scalar or a plain Go value:

```go
doc.Equal(map[string]any{"skills": []string{"Go", "SQL"}})   // is_deeply

rules.Diff(model)    // skills[1]: got "SQL", want "Docker"
```

The rules are `is_deeply`'s:

- **Plain values compare as strings,** as with `eq`. So `1` and `1.0` are
  equal, but `1` and `"1.0"` aren't. A boolean true equals `1`, and a byte
  string equals text of the same characters.
- **Undef equals only undef,** not `""`.
- **Arrays** must have the same length and equal elements. A hole counts as
  undef.
- **Hashes** must have the same keys and equal values, so a missing key isn't
  the same as one whose value is undef.
- **Structures that loop back on themselves** compare safely.

`Diff` returns `""` when the two are equal. Otherwise it gives the path,
then what's on each side, with `does not exist` for a missing element or key.
Keys are taken in sorted order, so the same difference is always the one
reported. That makes it handy in tests, and for comparing two extractions
of the same posting.

When the answer is all you need, `Equal` is the cheaper call: it builds no
report and stops at the first difference. On two copies of the ASHE pay
table (15,000 rows), it takes 28 ms where `Diff` takes 123 ms.

## Operators

Operators return a new scalar and don't change either side. The right-hand side
can be a `*SV` or a plain Go value.

| Perl | Go |
|------|-----|
| `$a + $b`, `-`, `*`, `/` | `a.Add(b)`, `a.Sub(b)`, `a.Mul(b)`, `a.Div(b)` |
| `$a % $b` | `a.Mod(b)` |
| `$a ** $b` | `a.Pow(b)` |
| `-$a` | `a.Neg()` |
| `$a . $b` | `a.Concat(b)` |
| `$a x 3` | `a.Repeat(3)` |
| `==`, `!=`, `<`, `<=`, `>`, `>=` | `NumEq`, `NumNe`, `NumLt`, `NumLe`, `NumGt`, `NumGe` |
| `eq`, `ne`, `lt`, `le`, `gt`, `ge` | `Eq`, `Ne`, `Lt`, `Le`, `Gt`, `Ge` |
| `$a++`, `$a--` | `a.Inc()`, `a.Dec()` |

```go
scalar.Value("10").Add(5)       // 15
scalar.Value("10").Concat(5)    // "105"
scalar.Value("10").NumEq(10.0)  // true
scalar.Value("10").Eq("10.0")   // false
```

Things to know:

- **Integers stay integers.** Integer results stay integers as long as they fit
  in an `int64`. Otherwise they become floats.
- **Division:** `/` gives an integer only when it divides evenly: `7/2` is `3.5`.
- **Modulus:** `%` takes the sign of the right-hand side: `-7 % 3` is `2`.
- **Dividing by zero:** `Div` and `Mod` by zero panic.
- **String increment:** `Inc` on a string like `"aa"` gives `"ab"`, `"Az"`
  gives `"Ba"`, and `"zz"` gives `"aaa"`. `Dec` always works on numbers.
- **Negating strings:** `Neg` on a word adds or flips its sign:
  `-"foo"` is `"-foo"`, and `-"-foo"` is `"+foo"`.
- **In place:** `Inc` and `Dec` change the scalar itself and autovivify, so
  counting words works the Perl way:

  ```go
  counts := scalar.Hash()
  for _, w := range words {
  	counts.Get(w).Inc()  // $counts{$w}++
  }
  ```

### Assignment operators

Perl's `+=` and friends change the scalar itself and return it, so they
chain. Each follows the matching operator's rules, so integers stay integers
and `/=` gives an integer only when it divides evenly.

| Perl | Go |
|------|-----|
| `$x += $y`, `-=`, `*=`, `/=` | `x.AddInPlace(y)`, `SubInPlace`, `MulInPlace`, `DivInPlace` |
| `$x %= $y`, `**=` | `x.ModInPlace(y)`, `PowInPlace` |
| `$x .= $y` | `x.ConcatInPlace(y)` |
| `$x x= 3` | `x.RepeatInPlace(3)` |
| `$x //= $y` | `x.OrSet(y)` (see [Undef](#undef)) |

Like `Set`, they write through to a missing hash or array element, so
accumulating works the Perl way:

```go
totals := scalar.Hash()
for _, job := range jobs {
	totals.Get(job.Dept).AddInPlace(job.Salary)   // $totals{$dept} += $salary
}
```

If the operation panics, as dividing by zero does, the scalar is left
unchanged.

## Regular expressions

Regexes use [regexp2](https://github.com/dlclark/regexp2) instead of Go's
standard `regexp`. Its syntax is close to Perl's. Unlike Go's standard engine,
it supports lookahead, lookbehind, backreferences and atomic groups.

One difference from Perl remains: case-insensitive matching pairs single
characters only. Perl also uses the folds where one character matches
several, so `/ss/i` matches `ß` and `/strasse/i` matches `straße`; regexp2
doesn't. To match those, fold both sides first with `Fc`.

You write patterns inline, as in Perl. Each pattern is compiled the first time
it's used and then reused. You can also compile a pattern up front, as with
Perl's `qr//`; see [Compiling patterns up front](#compiling-patterns-up-front).
The cache of compiled patterns holds 4,096, and starts again when it's full,
so patterns built from data, such as one for each skill name, can't make it
grow without end. `At` paths and `Interpolate` templates are cached the same
way.

| Perl | Go |
|------|-----|
| `if ($s =~ /re/)` | `if s.M(re) != nil` |
| `my @m = $s =~ /re/` | `s.M(re)` |
| `$s =~ /re/g` | `s.M(re, "g")` |
| `%+` | `s.Named(re)` |
| `$s =~ s/re/x/` | `s.S(re, "x")` |
| `$s =~ s/re/x/g` | `s.S(re, "x", "g")` |
| `$s =~ s/re/x/r` | `s.S(re, "x", "r")` |
| `$s =~ s/re/f($1)/e` | `s.S(re, func(m *scalar.AV) any { ... })` |
| `split /re/, $s` | `s.Split(re)` |
| `$s =~ tr/a-z/A-Z/` | `s.Tr("a-z", "A-Z")` |
| `my $re = qr/re/i` | `re := scalar.MustCompile(re, "i")` |
| `$s =~ $re` | `s.M(re)` |

`M`, `Named`, `S` and `Tr` take Perl's flags as an optional last argument,
written as one string: `s.M(re, "gi")`, `s.S(re, "x", "gi")`.

- **`M`, `Named` and `S`:** `i` (ignore case), `m` (multi-line `^`/`$`), `s`
  (`.` matches newline), `x` (allow spaces and comments in the pattern).
- **`M` and `S`:** `g` (every match, not just the first).
- **`S` only:** `r` (return a new string instead of changing the scalar).
- **`Tr` has its own flags:** `c`, `d`, `s` and `r`, the same as Perl's `tr///`.

`Split`'s optional argument is the limit instead, as in Perl. To give `Split`
a flag, put it inside the pattern: `` s.Split(`(?i)and`) ``. That works in
every pattern.

Use Go raw strings (backquotes) for patterns, so backslashes don't need
escaping.

### Compiling patterns up front

`scalar.MustCompile` compiles a pattern once and returns a `*scalar.Regexp`,
like Perl's `qr//`. `M`, `Matches`, `Named`, `S` and `Split` take one
wherever they take a pattern string:

```go
var salary = scalar.MustCompile(`£([\d,]+)`) // my $salary = qr/£([\d,]+)/;

for m := range s.Matches(salary) { ... }
s.S(salary, "£X", "g")
```

Declare the patterns your program relies on as package-level variables, as
`salary` is above. If one of them is invalid, the program then panics as soon
as it starts, rather than later, when the pattern is first used. Go's own
`regexp.MustCompile` works the same way.

For patterns that come from outside the program, such as user input or a
config file, use `scalar.Compile` instead. It returns an error rather than
panicking.

**Flags are fixed when compiling**, as with `qr//i`:
`scalar.MustCompile(re, "i")` ignores case wherever it's used. Giving `i`,
`m`, `s` or `x` again to `M`, `Named` or `S` with a `*Regexp` panics, rather
than being silently ignored. The flags that belong to the operation, `g` and
`r`, are given to it as usual: `s.M(re, "g")`.

Two more details:

- A compiled pattern keeps the `RegexTimeout` in force when it was compiled.
- `Split`'s awk mode is only for the string `" "`, as in Perl, where
  `split qr/ /` splits on single spaces.

The pattern parameter is a Go `any`, as methods can't be generic, so
passing anything other than a string or a `*Regexp` panics at run time.

### Building patterns from data

To use text from your data inside a pattern, pass it through
`scalar.QuoteMeta`, Perl's `quotemeta` (`\Q...\E`). Skill names are full of
characters that mean something in a regex: without it, `s.M("C++")` panics
with an invalid pattern, and `s.M(".NET")` also matches `"xNET"`.

```go
for _, skill := range []string{"C++", ".NET", "Node.js"} {
	if posting.M(`(?i)` + scalar.QuoteMeta(skill)) != nil { ... }
}
```

`QuoteMeta` takes a Go string or a scalar, and returns a Go string, ready to
add to a pattern.

### Matching

```go
s := scalar.Value("Salary: £45,000 - £55,000")

if s.M(`(?i)salary`) != nil { ... }

m := s.M(`£([\d,]+)\s*-\s*£([\d,]+)`)
m.Get(0) // "£45,000 - £55,000"   ($&)
m.Get(1) // "45,000"              ($1)
m.Get(2) // "55,000"              ($2)
```

Unlike Perl's list of captures, which starts at `$1`, `M` puts the whole
match at index 0. That way `Get(n)` is always `$n`.

`M` returns `nil` if nothing matched. It's still safe to read:
`s.M(re).Get(1).Str()` is just `""`. A group that didn't take part in the
match is undef.

To go through every match, as Perl's `while ($s =~ /re/g)` does, loop
over `Matches`. Each `m` is laid out as `M` lays out one match, so
`m.Get(1)` is `$1`, and breaking out of the loop stops the search:

```go
for m := range s.Matches(`£([\d,]+)`) {
	fmt.Println(m.Get(1))   // 45,000 then 55,000
}
```

With `g`, `M` finds every match at once, and returns an array with one
element per match: a reference to that match's array, as Perl's arrays of
arrays hold references. Reach into it with a path, `all.At("[1][1]")`, or
by dereferencing, `all.Get(1).AV().Get(1)`. If nothing matches, it returns
`nil` too, so `m != nil` means "matched" either way.

`Named` returns the named groups of the first match as a hash, like Perl's
`%+`, or `nil` if nothing matched. It doesn't take `g`, because `%+` only ever
describes one match:

```go
d := scalar.Value("2026-10-11").Named(`(?<year>\d{4})-(?<month>\d\d)`)
d.Get("year").Int() // 2026
```

**Group numbering differs from Perl.** Named groups are numbered *after* all
the unnamed ones. In `(?<y>\d+)-(\d+)`, the unnamed group is `$1` and `y` is
`$2`. If you name your groups, use `Named`.

### Substitution

`S` is Perl's `s///`. It changes the scalar you call it on and returns how
many substitutions it made, as a scalar (so use `.Int()` to get a Go `int64`).
If nothing matched, the scalar is left exactly as it was.

```go
s := scalar.Value("a  b   c")
s.S(`\s+`, " ", "g")       // s is now "a b c"; returns 2

name := scalar.Value("Smith, John")
name.S(`(\w+), (\w+)`, "$2 $1")   // "John Smith"
```

In a replacement string you can use `$1`, `${name}`, `$&` (the whole match)
and `$$` (a literal `$`).

With `r`, `s` isn't changed and you get the new string back:

```go
clean := s.S(`\s+`, " ", "gr").Str()
```

For computed replacements, as in `s///e`, pass a function. It gets the match
in the same layout as `M` returns:

```go
s := scalar.Value("10 apples and 3 pears")
s.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g")
// "20 apples and 6 pears"
```

### Split

```go
scalar.Value("a,b,,c,,").Split(`,`)         // a, b, "", c   (trailing empties dropped)
scalar.Value("a,b,,c,,").Split(`,`, -1)     // a, b, "", c, "", ""
scalar.Value("a,b,c,d").Split(`,`, 2)       // a, "b,c,d"
scalar.Value("a1b2c").Split(`(\d)`)         // a, 1, b, 2, c   (captures included)
scalar.Value("  some  words ").Split(" ")   // some, words     (awk mode)
scalar.Value("abc").Split(``)               // a, b, c
```

These follow Perl's `split` rules, including what happens with empty fields
and zero-width matches.

### Transliteration

`Tr` works like Perl's `tr///`. It changes `s` in place and returns how many
characters matched, as a scalar.

```go
s := scalar.Value("hello")
s.Tr("a-z", "A-Z")                                 // "HELLO"; returns 5

scalar.Value("banana").Tr("a", "")                // count only: returns 3
scalar.Value("h-e-l-l-o").Tr("-", "", "d")        // delete: "hello"
scalar.Value("a    b").Tr(" ", " ", "s")          // squeeze: "a b"
scalar.Value("hi, you!").Tr("a-zA-Z", " ", "cs")  // "hi you "
```

To use a literal hyphen, write `\-`.

With `c`, every character *not* in the list is replaced by the last character
of the replacement list. Perl matches them up position by position instead,
but that difference only matters if the replacement list has more than one
character.

### Timeouts

regexp2 backtracks, like Perl. That means a badly written pattern, such as
`^(a+)+$`, can take a very long time on some inputs. To stop one bad job
posting from hanging the whole program, every match has a time limit
(`scalar.RegexTimeout`, 100ms by default).

If a match runs past the limit, it panics. It may run a little past the limit
before stopping, because the engine only checks the clock every so often.
Change `RegexTimeout` at startup, before running any patterns.

An invalid pattern or an unknown flag also panics, just as Perl would die.

For a simple pattern on a hot path, Go's standard `regexp` package is faster
and can never backtrack.

## Byte strings

A scalar can hold binary data, such as a blob, as a **byte string**: one made
from a Go `[]byte`, or by `Pack`. Its kind is `scalar.Bytes`, and `s.Bytes()`
gives the bytes back as a `[]byte` (a copy).

```go
png := scalar.Value(data)       // data is a []byte
png.Length()                     // the number of bytes
png.M(`^\x89PNG`) != nil          // patterns match bytes
header := png.Substr(0, 8)       // still a byte string
blob := png.Bytes()              // a Go []byte again
```

A byte string behaves as Perl's does under `use v5.12` or later, which turns
on the `unicode_strings` feature: **each byte is one character, from 0 to
255**. So:

- `Length`, `Substr`, `Index` and `RIndex` count bytes, and their results
  are byte strings.
- Regexes see one character per byte, so `\x89PNG` finds those bytes, and
  captures, substitutions and splits of a byte string are byte strings.
- `\w`, `\s`, case-insensitive matching and `Lc`, `Uc` and friends treat the
  bytes from 128 to 255 as the Latin-1 letters and spaces they'd be as
  characters: `\xe9` is `é`, so `Uc` of it is `\xc9`.
- A result with a character beyond 255 becomes text, as Perl upgrades a
  string: `Uc` of `\xff` (`ÿ`) is `Ÿ`, which has no single byte.
- Joining text to a byte string joins characters: `é` (U+00E9) becomes the
  one byte `\xe9`, while `☺` makes the result text. To append text's UTF-8
  bytes, append `scalar.Value([]byte(text))`.
- `eq`, `lt` and the other comparisons compare characters, so the byte
  string `"\xe9"` equals the text `"é"`.

`Str()` gives the bytes as they are, for writing a blob out or for `Unpack`,
and printing a byte string prints its bytes, as Perl's `print` does.
`Pack`'s tests and these were checked against Perl itself: `TestBytesMatchPerl`
sends hundreds of random byte strings through every operation above in both.

Where byte strings differ from Perl:

- **JSON:** a byte string encodes as base64, as Go's `encoding/json` encodes a
  `[]byte`, so binary data survives the trip; Perl's JSON modules would write
  each byte as a character.
- **`%x` and `%X`** give a byte string's bytes in hex, as Go does for a
  `[]byte`, where Perl's `sprintf` would read it as a number.
- **Hash keys and set members** use a byte string's bytes as they are, so the
  byte string `"\xe9"` and the text `"é"` are different keys.

A Go `[]byte` in a list is one value, a byte string, not a list of numbers:
`scalar.Array(blob, "name")` has two elements.

## Pack and unpack

A subset of Perl's `pack` and `unpack`: hex (`H`, `h`), bits (`B`, `b`) and
floating-point numbers (`f` for float32, `d` and `F` for float64). That
covers turning bytes into hex or bits and back, and numbers into their
binary forms and back.

| Perl | Go |
|---|---|
| `pack("H*", "4a6f")` | `scalar.Pack("H*", "4a6f")`: `"Jo"` |
| `unpack("H*", $bytes)` | `scalar.Unpack("H*", bytes)`: an array holding `"4a6f"` |
| `pack("B8", "01001010")` | `scalar.Pack("B8", "01001010")`: `"J"` |
| `pack("f>", 1.5)` | `scalar.Pack("f>", 1.5)`: a big-endian float32's 4 bytes |
| `unpack("B32", pack("f>", $x))` | `scalar.Unpack("B32", scalar.Pack("f>", x))`: a float's bits |
| `pack("d<*", @nums)` | `scalar.Pack("d<*", nums)`: little-endian float64s |

Templates work as in Perl: a letter, then `<` or `>` for the byte order of a
float (without either, the machine's own), then a count or `*`, with spaces
and `#` comments allowed between items, as in `"H2 f> B8"`. For `H`, `h`,
`B` and `b` the count is a number of digits or bits, not bytes. `Unpack`
takes a scalar, a Go string or a `[]byte`.

Every detail follows Perl, including the odd ones: an odd number of hex
digits fills the last byte with a zero, characters that aren't hex digits or
bits are turned into them by Perl's own formula rather than rejected, a float32
comes back widened (`0.1` as `0.100000001490116`), and bytes left over at the
end are ignored. `TestPackMatchesPerl` checks hundreds of random templates
and values against Perl itself, when `perl` is installed.

**The result is a [byte string](#byte-strings),** as Perl's is, so `Length`,
`Substr` and `Index` count its bytes. Other template letters (`N`, `a`, `w`
and so on) panic.

## JSON

```go
s, err := scalar.FromJSON(data)  // decode
out, err := json.Marshal(s)      // encode
```

- JSON objects become hashes, arrays become arrays, and `null` becomes undef.
- Whole numbers stay exact `int64` values. Other numbers become `float64`.
- `SV`, `AV` and `HV` all work with `encoding/json`, so you can use them as
  fields in your own structs.
- Hashes are encoded with their keys sorted, so the same data always produces
  the same JSON, even though hash order is random.
- Decoding reads JSON straight into scalars, about three times faster than
  going through `encoding/json`'s maps and slices. Anything unusual (an
  error, bytes that aren't UTF-8, a lone `\ud800` escape, very deep
  nesting) is handed to `encoding/json` instead, so results and error
  messages are always exactly `encoding/json`'s.
- **Encoding can fail**, with an error rather than a panic, in two cases JSON
  can't represent: a reference that leads back to itself (a hash that
  contains itself, directly or further down), and a NaN or infinite number.
  The same array or hash appearing twice, without a loop, is fine: it's
  written out each time.

## YAML

`scalar.FromYAML` reads a YAML document, as `FromJSON` reads JSON, and
`ToYAML` writes one, for config files, Kubernetes manifests, Helm values and
the like:

```go
values, err := scalar.FromYAML(data)
replicas := values.At("production.replicas").Int()

out, err := values.ToYAML()
```

Reading follows YAML 1.2:

| In the YAML | Becomes |
|---|---|
| a mapping, a sequence | a hash, an array |
| `null`, `~`, or nothing | undef |
| `true`, `false` | a boolean |
| `42`, `0x1F`, `0o17` | an integer (a float if it's too big for an `int64`) |
| `2.5`, `1e3`, `.inf`, `.nan` | a float |
| `!!binary aGk=` | a [byte string](#byte-strings) |
| anything else, including `"42"`, `'007'`, `no`, `yes` and `2026-10-15` | a string, as written |
| `&base` … `*base` | the same array or hash at each place: shared parts, and loops |
| `<<: *base`, `<<: [*a, *b]` | merge keys: the keys the mapping doesn't have itself, the earlier mapping winning |

A few details:

- **`no` is the string `"no"`.** YAML 1.2 reads only `true` and `false` as
  booleans; YAML 1.1 also took `yes`, `no`, `on`, `off`, `y` and `n`, which is
  how Norway's country code once became `false`.
- **A key is its text as written,** so `1: one` gives the key `"1"`. A key
  that's a mapping or a sequence is an error.
- **Tags YAML doesn't define are dropped,** keeping the value they're on, as
  `FromDump` drops a class. For CloudFormation, that means `!Ref MyBucket`
  reads as `"MyBucket"`, and `!GetAtt Bucket.Arn` as `"Bucket.Arn"`: the
  values are there, but not that they were references.
- **Several documents,** separated by `---`, as in a Kubernetes manifest or
  Helm's output, need `FromYAMLAll`, which returns them as an array.
  `FromYAML` returns an error for more than one, rather than reading just the
  first. Empty documents, such as after a trailing `---`, are skipped, as
  `kubectl` skips them.

```go
docs, err := scalar.FromYAMLAll(manifest)
for _, d := range docs.All() {
	fmt.Println(d.At("kind"), d.At("metadata.name"))
}
```

Writing is careful to be read back the same way by every YAML reader, not
just YAML 1.2 ones:

- **Strings that any reader could take as something else are quoted:**
  `"1.0"`, `"007"` and `"true"`, and also `"no"`, `"on"`, `"y"`, `"1_000"`
  and `"1:30"`, which YAML 1.1 readers, such as Kubernetes' Go library and
  Python's PyYAML, read as booleans or numbers. That goes for keys as well.
- **Floats always look like floats,** so `1.0` is written `1.0`, not `1`,
  which would read back as an integer. Infinity and NaN are `.inf` and
  `.nan`, which JSON can't write at all.
- **A byte string is written as `!!binary`,** in base64.
- **An array or hash that appears more than once** is given an anchor, `&1`,
  where it first appears, and is an alias, `*1`, after that, so shared parts,
  and structures that refer back to themselves, come back as they were.
- **Keys are sorted,** so the same data always gives the same YAML, and the
  indent is two spaces, as Kubernetes and Helm use.

Scalars, arrays and hashes also work with the YAML package itself
(`go.yaml.in/yaml/v3`), as they do with `encoding/json`: a `*scalar.SV`
field in a struct you read with `yaml.Unmarshal` takes whatever part of the
document is there, for the parts whose shape varies.

`TestYAMLRoundTrip` checks that what `ToYAML` writes reads back the same,
type by type, for hundreds of random structures, shared parts and loops
included. `TestYAMLMatchesPerl` checks against Perl's `YAML::PP`, both ways:
`FromYAML` reads what `YAML::PP` writes, and `YAML::PP` reads what `ToYAML`
writes, with the same values and types. It runs when `YAML::PP` is
installed.

## Into Go values: `To` and `Decode`

Once you know the shape of some data, leave `scalar` for ordinary Go types,
which the compiler checks. `scalar.To` converts a scalar, array, hash, set or
bag into the Go type you name in brackets, and returns it:

```go
skills, err := scalar.To[[]string](doc.At("skills"))   // a []string
max, err := scalar.To[int](doc.At("salary.max"))       // an int, even from "55000"
posting, err := scalar.To[Posting](doc)                // your own struct
counts, err := scalar.To[map[string]int](bag)          // a bag's counts
```

`s.Decode(&v)` does the same into a value you already have. Both follow the
same rules, set out below. Use `To` for a new value, and `Decode` to fill in
one that's partly set already (see [Decode](#decode-filling-in-a-value-you-have)).

With a struct, `To` takes you from loosely shaped data to a typed value,
which [What it costs](#what-it-costs) recommends doing as early as you can:

```go
type Posting struct {
	Title  string   `json:"title"`
	Skills []string `json:"skills"`
	Salary struct {
		Min int `json:"min"`      // "45000" fills it, as Perl would numify it
		Max int `json:"max"`
	} `json:"salary"`
	Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar
}

p, err := scalar.To[Posting](doc)
if err != nil {
	return err   // scalar: Decode: salary.max: want int, got "competitive"
}
fmt.Println(p.Title, p.Salary.Max)
```

`To` is a function rather than a method, `scalar.To[T](x)` rather than
`x.To[T]()`, because Go's methods can't take a type in brackets.

### What can go in the brackets

| Type | Takes | Example |
|---|---|---|
| a struct | a hash: each key fills the field with that name (see below) | `scalar.To[Posting](doc)` |
| a slice, `[]T`, or an array, `[N]T` | an array, or a set's members | `scalar.To[[]string](doc.At("skills"))` |
| a map, `map[string]T` or `map[int]T` | a hash, or a bag's counts | `scalar.To[map[string]int](bag)` |
| `string` | any plain value, as its string | `scalar.To[string](doc.At("title"))` |
| `int`, `int64`, `uint`, … | a whole number, or a string holding one, that fits | `scalar.To[int](doc.At("salary.max"))` |
| `float64`, `float32` | a number, or a string holding one | `scalar.To[float64](doc.At("rate"))` |
| `bool` | a boolean, `0` or `1`, or `"true"` or `"false"` | `scalar.To[bool](doc.At("remote"))` |
| a pointer, `*T` | what `T` takes; undef gives nil | `scalar.To[*int](doc.At("bonus"))` |
| `any` | anything, as plain Go values, as `Any()` gives them | `scalar.To[any](doc)` |
| `*scalar.SV`, `*scalar.AV`, `*scalar.HV` | a deep copy (see `Clone`) of whatever is there | a struct field for a part that stays loose |
| a type with `UnmarshalJSON` or `UnmarshalText`, such as `time.Time` | whatever it accepts | `scalar.To[time.Time](doc.At("posted"))` |
| `[]byte` | a byte string's bytes, or text in base64, as `encoding/json` takes it | `scalar.To[[]byte](doc.At("blob"))` |

These nest as you'd expect: `[]Posting` takes an array of hashes,
`map[string][]string` a hash of arrays. The source can be a scalar holding an
array or hash, or the array or hash itself: `scalar.To[[]string](a)` works
for an `*AV`, as does `scalar.To[[]string]([]any{"Go", "SQL"})` for a Go value.

### How values convert

- **Numbers and strings convert as in Perl, but only where Perl wouldn't
  warn.** `"42"` and `" 42 "` fill an `int`; `"42abc"`, `"45,000"`, `""` and
  `2.5` don't, nor does a number too big for the type, such as `300` for an
  `int8`.
- **Booleans are stricter than Perl.** A `bool` takes a boolean, `0` or `1`,
  or `"true"` or `"false"`, but not Perl's other true and false values,
  since `"false"` would be true. A boolean doesn't fill a number or a
  string either, where Perl would make it `1` or `""`: it means the data has
  the wrong shape.
- **Struct fields match keys as `encoding/json` matches them:** by the json
  tag's name, or the field's own, exactly or else ignoring case, and then
  the first such field in the struct. Fields of embedded structs are
  promoted; where two have the same name at the same depth and neither is
  tagged, the name matches neither. Unexported fields and fields tagged
  `"-"` are skipped, keys with no field are ignored, and fields with no key
  are left as they are.
- **The result is a copy.** It shares nothing with the data it came from, so
  changing one never changes the other.

### Missing, null and optional values

Undef, whether it came from a missing key or a JSON `null`, isn't an error.
`To` gives the zero value: `nil` for a pointer, slice or map, `0`, `""` and
so on otherwise. Inside a struct, undef sets a pointer, slice, map or
interface field to nil and leaves any other field as it is, as `null` does
with `json.Unmarshal`.

So an optional value is a pointer. Asking for one tells a missing value
from a zero, which `Str`, `Int` and `Num` can't, as they give `""` and `0`
for undef:

```go
bonus, err := scalar.To[*int](doc.At("bonus"))     // nil if there's no bonus, or it's null
min, err := scalar.To[*float64](doc.At("min"))     // nil, or a pointer to the number
period, err := scalar.To[*string](doc.At("period"))
```

There's no `StrPtr` or `NumPtr` method, because `To[*T]` does this for any
type, and inside structs too, so a struct with `*string` and `*float64`
fields (as the model's metadata has) fills in one call. It's also checked:
`To[*float64]` refuses `"competitive"` with an error, where `Num` would make
it `0` and a pointer method would have to choose between the two.

| You want | Use |
|---|---|
| A value, with Perl's conversions, never failing | `Str()`, `Int()`, `Num()` |
| A value, and whether it really was one | `IntOK()`, `NumOK()` |
| An optional value: nil if missing, checked if not | `scalar.To[*int](s)`, `scalar.To[*string](s)` |

`To` never creates anything, so asking for a path that isn't there leaves
the data as it was.

### When a value doesn't fit

`To` and `Decode` return an error rather than panicking, and stop at the
first value that doesn't fit, taking a hash's keys in sorted order, so when
several values don't fit, the error is always about the same one. On an
error, `To` returns the zero value. The
error is a `*scalar.DecodeError`, whose `Path` is written as `At` takes it,
so it leads straight to the value:

```go
p, err := scalar.To[Posting](doc)
// scalar: Decode: salary.max: want int, got "competitive"

var de *scalar.DecodeError
if errors.As(err, &de) {
	log.Printf("%s is %s", de.Path, doc.At(de.Path).Dump())   // salary.max is "competitive"
}
```

Paths look like `salary.max`, `skills[2]` or `meta{log.level}`, the last for
a key that isn't a plain word. An empty path means the value itself.

Data that refers back to itself is an error too, where decoding it into a
type such as `type Node struct{ Next *Node }` would otherwise go round for
ever: the path says where the loop closes. Into a `*scalar.SV` or an `any`,
which can hold such data, it's fine.

### `Decode`: filling in a value you have

`Decode` takes a pointer to a value and fills it in, by the same rules. Since
fields with no key are left alone, it can fill in a struct whose defaults
are already set:

```go
p := Posting{Title: "Untitled"}          // the default
if err := doc.Decode(&p); err != nil {   // a title in doc replaces it; none leaves it
	return err
}
```

On an error, `Decode` leaves in place whatever it filled in before the value
that didn't fit, taking keys in sorted order, where `To` gives the zero
value. `Decode` needs a non-nil
pointer, and returns an error otherwise.

### And the way in: `Value` of a struct

`Value` and `Set` take a struct the other way: it becomes a hash, keyed as
encoding/json would write it, by the same rules `Decode` reads it with
(json tags, omitempty, promoted embedded fields), so a struct put in with
`Value` comes back out with `Decode` unchanged. A type with its own
`MarshalJSON` or `MarshalText` is what that gives: `time.Time` is its
RFC 3339 string.

A Go value that contains itself, such as a struct whose pointer field leads
back to it, or a map holding itself, panics, naming its type. encoding/json
refuses one with an error, but `Value` has no error to return, and
following the loop would overflow the stack, which kills the program.

```go
result.At("occupation").Set(occupations.Summarise(cd)) // a soc.Summary, as a hash
```

Perl has no structs, so this goes beyond it, as `Decode` does, for Go's
sake. Before, a struct was stringified with `fmt.Sprint`, quietly: found
writing `cmd/grammarrules`, where it took a JSON round trip each time.

### Which way out?

`To` and `Decode` check every value. The package's other ways out of
`scalar` do different jobs:

| You want | Use | A value that doesn't fit |
|---|---|---|
| a Go value of a type you choose, checked | `scalar.To[T](x)`, `x.Decode(&v)` | an error, with its path |
| each element in Perl's string or numeric context | `a.StrSlice()`, `a.IntSlice()`, `a.NumSlice()` | converted anyway, as Perl would: `"five"` is `0` |
| a Go slice or map of the stored scalars themselves, to change them in place | `h.ValuesSlice()`, `h.PairsMap()`, `a.SplitAtSlice(n)`, and `h.KeysSlice()` for the keys | no conversion, so nothing to fit |
| everything as plain Go values, for code that takes `any` | `x.Any()`, or `scalar.To[any](x)` | `Any` panics on a structure that loops back on itself, where `To` returns an error |
| a byte string's bytes | `x.Bytes()` | text gives its UTF-8 |

`TestDecodeMatchesJSON` checks that on data that's already the right shape,
`To` and `Decode` give what `encoding/json` gives.

## Copies, references and in-place changes

### Getting an element gives you the element itself

`a.Get(i)`, `h.Get(k)`, `Path(...)`, `At(...)`, `All()`, the matches from
`Find` and the elements of `Values()` give you the value that's actually stored, not a copy. Changing it changes
the array or hash:

```go
a.Get(0).Set("x")      // changes $a[0]
h.Get("count").Inc()   // changes $h{count}
```

**This is different from Perl's `my $x = $h{k}`**, which takes a copy. In Go,
`x := h.Get("k")` is more like Perl's `for` loop variable: it's another name
for the same value. To take a copy, call `Copy()`:

```go
x := h.Get("k").Copy()   // my $x = $h{k}
x.Set(1)                 // doesn't change $h{k}

for _, v := range a.All() {
	v := v.Copy()        // work on a copy inside the loop
	...
}
```

Perl doesn't need a `copy`, because its assignment copies. Go's `:=` can't be
made to copy, so `Copy()` makes it explicit. As with Perl's assignment, the
copy is shallow: copying a reference to an array or hash gives you a second
reference to the same data.

### Copying values and references

- Copying a scalar copies its value: `other.Copy()`, `scalar.Value(other)`
  or `s.Set(other)` all do this. So do `Push` and the other methods that take
  lists, for every value they add.
- Copying a reference doesn't copy the array or hash it points to. Both copies
  point at the same data, just as in Perl.

### Deep copies: `Clone`

`Clone` copies a scalar and everything it refers to, all the way down, as
Perl's `Storable::dclone` does. Changing the clone never changes the
original. Arrays and hashes have it too, as `dclone(\@a)` and `dclone(\%h)`:

```go
row := h.Get("defaults").Clone()   // my $row = dclone($h{defaults})
row.At("tags").AV().Push("new")    // $h{defaults}{tags} is unchanged

mine := rows.Clone()               // dclone(\@rows): the rows too, not just the list
```

Compare `Copy` and `scalar.Hash(h)`, which copy one level: a hash copied that
way still shares its nested arrays and hashes with the original.

As with `dclone`, the clone has the original's shape. An array or hash that
appears twice is copied once and appears twice in the clone, a reference
that loops back leads back into the clone, holes stay holes, and byte
strings stay byte strings. So `Clone` is safe on any structure, where `Any()`
panics on a loop.

### What changes in place

Methods either change the value you call them on, or leave it alone and
return something new:

| Changes in place | Returns something new, original untouched |
|---|---|
| `Set`, `Apply`, `Delete`, `DeleteAt`, `ExtractBy`, `Push` (arrays and hashes), `Pop`, `PopN`, `Shift`, `ShiftN`, `Unshift`, `Splice`, `SetLastIndex`, `SortInPlace`, `ReverseInPlace`, set and bag `Insert`, `InsertN`, `Delete` and `DeleteN` | `Copy`, `Clone`, `Then`, `When`, `Or`, `Sort`, `SortByStr` and the other `List::UtilsBy` methods, `Reverse`, `Map`, `Grep`, `Pairs` (whose values are the originals, as with `Values`), `Keys`, `SplitAt`, `SplitAtSlice`, `UniqueStr`, `UniqueNum`, `Sum`, `Sum0`, `Product`, `Reduce`, set and bag operations (`Union`, `Intersection`, …) |
| `SubstrReplace` | `Substr`, `Lc`, `Uc`, `Fc`, `UcFirst`, `LcFirst`, `Trim` |
| `Inc`, `Dec`, `AddInPlace` and the other assignment operators | Operators: `Add`, `Concat`, `Neg`, … |
| `S`, `Tr` | `S` and `Tr` with the `r` flag |

`Delete`, `Pop`, `PopN`, `Shift`, `ShiftN` and `Splice` return the elements they removed. These
are the actual values, no longer attached to the array or hash, so changing
them afterwards has no effect on it.

`Values` and `Pairs`, for arrays and hashes, sit between the two columns: the
array is new, but its elements (and in `Pairs`, the values) are the
original's own, as described under [Hashes](#hashes).

## Concurrency

**Scalars, arrays and hashes aren't safe for concurrent use, and that
includes reading them.** Several things that look like reads write:

| What you call | What it writes |
|---|---|
| `At("a.b")`, `Path(...)` on a missing route | the levels on the way, as Perl's autovivification does: reading `x.y` of `{"a": 1}` adds `x` |
| `Keys`, `Values()`, `Pairs`, `Each` on a hash | its cached key order, and `Each`'s position |

So two goroutines reading one structure can race, as two writing a Go map
do. The package's own caches (compiled regexes, paths, struct fields) are
safe to share; the values aren't.

**The pattern** is the one in [What it costs](#what-it-costs): scalar at the
edges, Go types in the core, and the edge belongs to one goroutine.

1. **Confine each structure to one goroutine.** A worker that reads a
   record keeps its scalars to itself, from parsing to writing.
2. **Hand over Go values, not scalars.** Once the shape is known, `To` or
   `Decode` it into a struct, and send that across: a struct of plain
   values is safe to share read-only, and the compiler checks it.
3. **If a scalar must cross, send a copy.** `Clone` makes a deep,
   independent one; the receiver owns it.
4. **Never share one for reading, unless it's locked.** Build data many
   goroutines read, such as lookup tables, as Go maps and slices before the
   goroutines start; or, if it must stay in scalar, `Lock` it first (see
   [Locking](#locking)): nothing a locked structure does when read writes.
5. **Check with the race detector.** `go test -race` and `go run -race`
   find what this list misses.

```go
// One worker per record: the scalars stay inside it, a struct comes out.
g.Go(func() error {
	doc, err := scalar.FromJSON(raw)
	if err != nil {
		return err
	}
	p, err := scalar.To[Posting](doc) // the shape is known: into Go
	results[i] = p                    // a plain struct, safe to share
	return err
})
```

`TestSharingAcrossGoroutines` checks these patterns under `-race`.

## Locking

`Lock` makes a structure read-only, all the way down, after Perl's
`Hash::Util`:

```go
doc, _ := scalar.FromJSON(body)
doc.Lock()
doc.At("pay.min")              // reading what's there is as usual
doc.At("titel")                // panics: Attempt to access disallowed key 'titel' in a restricted hash
doc.At("pay.max").Set(30000)   // panics: as it would add a key
doc.At("pay.min").Set(1)       // panics: Modification of a read-only value attempted
```

| Locked | Like Perl's | Reading | Writing |
|---|---|---|---|
| a hash | `lock_hash` | a key it has, as usual; one it hasn't got panics ("Attempt to access disallowed key 'x' in a restricted hash"), where `At` would have autovivified; `Exists`, `AtOK` and `Find` answer as usual | changing a value, adding a key or deleting one panics |
| an array | `Internals::SvREADONLY` | as usual, past the end too (undef) | push, pop, splice, an element, sorting or reversing in place: all panic |
| a scalar | `Readonly` (CPAN) | as usual; an undef doesn't autovivify | `Set`, `Inc`, `ConcatInPlace` and the rest panic |

It reaches everything the value refers to, nested arrays and their
elements included, which is a step beyond Perl, whose `lock_hash_recurse`
locks only hashes and whose read-only arrays still let their elements
change. A structure that refers back to itself locks too. `Unlock` undoes
it, as far; `Locked` says whether a value is locked; `Clone` gives an
unlocked copy. Locking twice, or unlocking what isn't locked, does nothing.

**What it's for.** Catching a misspelled key at the first read, rather
than reading undef and carrying on; keeping a lookup table or a config from
gaining keys; and sharing data between goroutines for reading. Nothing a
locked structure does when read writes anything (`Lock` works out each
hash's key order as it locks), so goroutines can read one at once; `Each`
is the exception, as it keeps a position. Lock before sharing, and
`TestLockedConcurrentReads` checks it under `-race`.

It isn't a substitute for Go types. A lock catches a mistake when the code
runs; a struct catches it when the code compiles. Locking is for the data
that has to stay in scalar.

**Does locking make reads faster?** No, and it doesn't need to. Data that
can't change could keep work it has done once, so we tried that, measured
on the ASHE pay table, 15,000 rows of strings (`lock_bench_test.go`). The
profiles found two costs that every value paid, locked or not, and those
were fixed for everyone instead:

| Read | Before | After | What it was |
|---|---|---|---|
| `Num` on 180,000 strings | 33 ms | 14.5 ms, no allocations | a regular expression; now a hand-written scanner |
| `Equal` on two copies of the table | 123 ms, 36 MB | 28 ms, 3 MB | `Equal` built `Diff`'s report and threw it away |

Four ideas that depend on a lock were measured, or thought through, and
then dropped:

- **Keep each value's number.** Now that `Num` is quick, this would save at
  most 30%. In exchange, every scalar would grow by 8 bytes, locked or not.
- **Remember what `At` found, per path.** A lookup would fall from about
  110 ns to 20 ns, but nothing does enough lookups for that to matter. Each
  hash would grow, and `Unlock` would have to throw the cache away.
- **Make `Clone` of a locked structure free.** A locked structure can
  already be shared as it is. `Clone`'s job is to give an unlocked copy,
  so this would change what it does.
- **Sort each hash's keys once.** Only `Dump` and `Diff` sort them, and
  neither is on a hot path.

## Caveats

- **It panics where Perl would die.** Most Go code returns errors instead of
  panicking, so be aware of these cases:
  - writing to an array element before the start of the array, or through
    something that isn't an array or hash (reads are always safe); this
    includes `Push` to a nil hash, and `Insert` into a nil set or bag
  - `Splice` with an offset before the start of the array
  - a negative count for `SplitAt`, `PopN`, `ShiftN` or `MostCommon`
  - `SubstrReplace` on a substring entirely outside the string
  - `Div` or `Mod` by zero
  - an invalid regex (use `scalar.Compile` to get an error instead), an
    unknown flag, a regex that runs past `RegexTimeout`, a pattern that's
    neither a string nor a `*Regexp`, a nil `*Regexp`, or the flags `i`,
    `m`, `s` or `x` given again with a `*Regexp`
  - a malformed JSONPath in `Find`
  - a template letter other than `H`, `h`, `B`, `b`, `f`, `d` and `F` in
    `Pack` or `Unpack`, or `<` or `>` after anything but a float
  - a malformed dump in `MustFromDump` (use `FromDump` to get an error)
  - a malformed path in `At`, `AtOK`, `DeleteAt` or `MustPath` (use
    `CompilePath` to get an error instead), or a malformed template in
    `Interpolate` or `MustTemplate` (use `CompileTemplate`), or a template
    that's neither a string nor a `*Template`
  - passing something other than a string, int or `*PathExpr` to `Path`, or
    something other than a string or function as an `S` replacement
  - `Any()` on a reference that leads back to itself (encoding it as JSON
    returns an error instead)

  If one bad record mustn't stop a batch, catch these with `recover` around
  the code that processes each record.
- **Not for concurrent use, not even reading:** none of these types are safe
  to share between goroutines without your own locking, as some reads write.
  See [Concurrency](#concurrency).
- **No compile-time checks:** a misspelled key won't fail to compile. It just
  reads as undef. Keep these types at the edges of your program, and use
  structs for anything you depend on.

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
October 2026. Commits up to 15 October mostly don't record Claude's part
in a `Co-Authored-By` trailer; later ones do.

## Reference

One example for every function and method, with the output it prints.
<!-- reference:start -->

Generated from the examples in `example_*_test.go`, which `go test` runs, so every
example here works and prints what it says. Edit the examples, not this section, then run
`go test ./internal/scalar -run TestReference -update`.

- [Scalars](#reference-scalars)
- [Strings](#reference-strings)
- [Operators](#reference-operators)
- [Arrays](#reference-arrays)
- [Hashes](#reference-hashes)
- [List utilities](#reference-list-utilities)
- [Sets and bags](#reference-sets-and-bags)
- [Regular expressions](#reference-regular-expressions)
- [Pack and unpack](#reference-pack-and-unpack)
- [JSON and JSONPath](#reference-json-and-jsonpath)
- [YAML](#reference-yaml)
- [Into Go values](#reference-into-go-values)

### Reference: Scalars

#### `Value`

Value returns a scalar holding x.

```go
fmt.Println(scalar.Value(42).Kind() == scalar.Int)
fmt.Println(scalar.Value(3.5).Kind() == scalar.Float)
fmt.Println(scalar.Value("text").Kind() == scalar.String)
fmt.Println(scalar.Value([]string{"a", "b"}).Ref()) // a slice becomes an array reference
fmt.Println(scalar.Value(nil).Defined())            // nil is undef
```

```
true
true
true
ARRAY
false
```

#### `Value.struct`

A struct becomes a hash, keyed as encoding/json would write it: by its json tags, leaving out what's omitempty and empty. Decode is the way back.

```go
type group struct {
	Code  string `json:"code"`
	Title string `json:"title,omitempty"`
}
g := scalar.Value(group{Code: "2134"})
fmt.Println(g.Ref(), g.At("code"), g.At("title").Defined())
```

```
HASH 2134 false
```

#### `SV.Set`

Set assigns x to s and returns s.

```go
s := scalar.Value("old")
s.Set(10)                     // $s = 10
fmt.Println(s.Add(1))         // it's a number now
s.Set(map[string]any{"a": 1}) // a map becomes a hash reference
fmt.Println(s.Ref())
```

```
11
HASH
```

#### `SV.Copy`

Copy returns a new scalar holding s's value, as Perl's my $x = $s does.

```go
h := scalar.Hash("count", 1)
alias := h.Get("count")      // the stored value itself
own := h.Get("count").Copy() // my $own = $h{count}
alias.Set(2)
own.Set(99)
fmt.Println(h.Get("count"))
```

```
2
```

#### `SV.Clone`

Clone returns a deep copy of s, as Storable's dclone does: arrays and hashes it refers to are copied too, all the way down, so changing the clone never changes the original.

```go
h := scalar.Hash("defaults", scalar.Value(map[string]any{"tags": []any{"go"}}))
shallow := h.Get("defaults").Copy() // my $row = $h{defaults}: the same hash
deep := h.Get("defaults").Clone()   // my $row = dclone($h{defaults}): its own
deep.At("tags").AV().Push("perl")
fmt.Println(h.Get("defaults").At("tags").AV().Join(","), deep.At("tags").AV().Join(","))
shallow.At("tags").AV().Push("sql") // changes the original
fmt.Println(h.Get("defaults").At("tags").AV().Join(","))
```

```
go go,perl
go,sql
```

#### `SV.Equal`

Equal reports whether s and other hold the same data, all the way down, as Test::More's is_deeply decides it.

```go
doc, _ := scalar.FromJSON([]byte(`{"skills": ["Go", "SQL"], "salary": {"max": 55000}}`))
fmt.Println(doc.Equal(map[string]any{ // is_deeply($doc, {...})
	"skills": []string{"Go", "SQL"},
	"salary": map[string]any{"max": "55000"}, // as strings, as eq compares
}))
fmt.Println(doc.At("salary.max").Equal(55000.0), scalar.Value(nil).Equal("")) // undef equals only undef
```

```
true
true false
```

#### `SV.Diff`

Diff says where s and other first differ, as is_deeply reports it, or "" if they're equal (see Equal).

```go
rules, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
model, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "Docker"]}`))
fmt.Println(rules.Diff(model)) // where they first differ, as a path for At
fmt.Printf("%q\n", rules.Diff(rules.Clone()))
```

```
skills[1]: got "SQL", want "Docker"
""
```

#### `SV.Apply`

Apply calls f on every plain value in s, all the way down, and stores what f returns in its place, as Data::Rmap's rmap { $_ = ...

```go
doc, _ := scalar.FromJSON([]byte(`{"title": " Go Developer ", "skills": [" Go", "SQL "], "salary": 50000}`))
doc.Apply(func(v *scalar.SV) any { // rmap { $_ = trim $_ } $doc
	if v.Kind() == scalar.String {
		return v.Trim()
	}
	return v // anything else as it is
})
out, _ := doc.MarshalJSON()
fmt.Println(string(out))
```

```
{"salary":50000,"skills":["Go","SQL"],"title":"Go Developer"}
```

#### `SV.Kind`

Kind reports what s currently holds.

```go
for _, x := range []any{nil, true, 1, 1.5, "s", []byte{0x89}, []any{}, map[string]any{}} {
	switch scalar.Value(x).Kind() {
	case scalar.Undef:
		fmt.Println("undef")
	case scalar.Bool:
		fmt.Println("bool")
	case scalar.Int:
		fmt.Println("int")
	case scalar.Float:
		fmt.Println("float")
	case scalar.String:
		fmt.Println("string")
	case scalar.Bytes:
		fmt.Println("bytes")
	case scalar.ArrayRef:
		fmt.Println("array")
	case scalar.HashRef:
		fmt.Println("hash")
	}
}
```

```
undef
bool
int
float
string
bytes
array
hash
```

#### `SV.Defined`

Defined is Perl's defined().

```go
fmt.Println(scalar.Value(nil).Defined())
fmt.Println(scalar.Value("").Defined()) // empty, but defined
fmt.Println(scalar.Hash().Get("missing").Defined())
```

```
false
true
false
```

#### `SV.Any`

Any returns s as a plain Go value, in the shapes encoding/json uses: undef is nil; booleans, integers, floats and strings are bool, int64, float64 and string; array references become []any and hash references map[string]any, converted all the way down.

```go
s := scalar.Value(map[string]any{"tags": []any{"go", 1}})
plain := s.Any().(map[string]any) // plain Go, sharing nothing with s
fmt.Println(plain["tags"])
```

```
[go 1]
```

#### `SV.Or`

Or is Perl's defined-or, $s // $def: a new scalar holding s's value if s is defined, otherwise def's.

```go
fmt.Println(scalar.Value(nil).Or("default")) // $x // "default"
fmt.Println(scalar.Value(0).Or("default"))   // 0 is defined, so it's kept
```

```
default
0
```

#### `SV.OrSet`

OrSet is Perl's $s //= $def: if s is undef, it sets s to def.

```go
h := scalar.Hash()
h.Get("retries").OrSet(3) // $h{retries} //= 3
h.Get("retries").OrSet(5) // already set, so unchanged
fmt.Println(h.Get("retries"))
```

```
3
```

#### `SV.Then`

Then calls f with s if s is defined, and returns what f returns as a new scalar; if s is undef, f isn't called, and Then returns undef.

```go
digits := func(v *scalar.SV) any { return v.S(`,`, "", "gr") }
figure := func(v *scalar.SV) any { return v.M(`£([\d,]+)`).Get(1).Then(digits) } // undef if no figure
annual := func(v *scalar.SV) any { return v.Mul(12) }
for _, text := range []string{"£4,000 a month", "Competitive"} {
	fmt.Println(scalar.Value(text).Then(figure).Then(annual).Or("unknown")) // stops at the first undef
}
```

```
48000
unknown
```

#### `SV.When`

When returns a copy of s if s is defined and pred(s) is true, and undef otherwise, without calling pred on undef.

```go
positive := func(v *scalar.SV) bool { return v.Num() > 0 }
doc, _ := scalar.FromJSON([]byte(`{"min": 45000, "max": 0}`))
fmt.Println(doc.At("min").When(positive).Or("none"))
fmt.Println(doc.At("max").When(positive).Or("none"))   // fails the test: undef
fmt.Println(doc.At("bonus").When(positive).Or("none")) // missing: positive isn't called
```

```
45000
none
none
```

#### `SV.Ref`

Ref is Perl's ref(): "ARRAY", "HASH", or "" for non-references.

```go
fmt.Println(scalar.Value(scalar.Array(1)).Ref())
fmt.Println(scalar.Value(scalar.Hash()).Ref())
fmt.Printf("%q\n", scalar.Value("text").Ref())
```

```
ARRAY
HASH
""
```

#### `SV.Str`

Str returns s in string context.

```go
fmt.Println(scalar.Value(42).Str())
fmt.Println(scalar.Value(0.1 + 0.2).Str()) // Perl's 15 significant digits
fmt.Println(scalar.Value(true).Str())
fmt.Printf("%q\n", scalar.Value(nil).Str())
```

```
42
0.3
1
""
```

#### `SV.String`

String implements `fmt.Stringer`.

```go
fmt.Println(scalar.Value(1e21)) // fmt uses String, as Perl's print would
```

```
1e+21
```

#### `SV.Dump`

Dump returns s as Data::Dumper::Concise prints it: Data::Dumper with Terse, Indent = 1, Useqq, Quotekeys = 0, Sortkeys and Trailingcomma, so nested arrays and hashes are laid out one element to a line, hash keys are sorted and quoted only when they need to be, and strings are double-quoted, with escapes for anything that isn't printable ASCII.

```go
doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"], "salary": {"min": 45000, "max": null}}`))
fmt.Println(doc.Dump()) // print Dumper($doc), as Data::Dumper::Concise prints it
```

```
{
  salary => {
    max => undef,
    min => 45000,
  },
  skills => [
    "Go",
    "SQL",
  ],
  title => "Go Developer",
}
```

#### `FromDump`

FromDump reads a structure written by Dump, or by Perl's Data::Dumper, and returns it, as FromJSON reads JSON.

```go
tags := scalar.Value(scalar.Array("go"))
doc := scalar.Hash("title", "Go Developer", "tags", tags, "again", tags)         // the same array twice
back, err := scalar.FromDump(scalar.Value(doc).Dump())                           // a round trip
fmt.Println(err, back.Equal(doc), back.At("tags").AV() == back.At("again").AV()) // still one array

perl, err := scalar.FromDump(`bless( {'name' => 'Ada', 'langs' => ['Perl', 'Go']}, 'Person' )`) // Data::Dumper's output
fmt.Println(err, perl.At("name"), perl.At("langs[1]"))
```

```
<nil> true true
<nil> Ada Go
```

#### `MustFromDump`

MustFromDump is FromDump, panicking if the dump is malformed.

```go
want := scalar.MustFromDump(`{
	title  => "Go Developer",
	skills => ["Go", "SQL"],   # test data, as Perl would write it
}`)
got, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
fmt.Printf("%q\n", got.Diff(want))
```

```
""
```

#### `MustFromDump.testData`

```go
// The expected value in a test: shorter than JSON, with comments where
// they help.
want := scalar.MustFromDump(`{
	title     => "Senior Go Developer",
	seniority => "Senior",
	salary    => { min => 60000, max => undef },   # max: not stated
	skills    => ["Go", "PostgreSQL", "Docker"],
	remote    => !!1,
}`)
got, _ := scalar.FromJSON([]byte(`{"title": "Senior Go Developer", "seniority": "Senior",
	"salary": {"min": 60000, "max": null}, "skills": ["Go", "PostgreSQL", "Docker"], "remote": true}`))
fmt.Printf("%q\n", got.Diff(want))
got.At("skills[2]").Set("Kubernetes")
fmt.Println(got.Diff(want)) // where they differ, as a path for At
```

```
""
skills[2]: got "Kubernetes", want "Docker"
```

#### `MustFromDump.toStruct`

```go
// Test data straight into the struct a function under test takes.
type Salary struct{ Min, Max int }
type Posting struct {
	Title  string
	Salary Salary
	Skills []string
}
p, err := scalar.To[Posting](scalar.MustFromDump(`{
	Title  => 'Data Engineer',
	Salary => { Min => 45_000, Max => 55_000 },   # underscores, as in Perl
	Skills => ['Python', 'SQL'],
}`))
fmt.Printf("%+v %v\n", p, err)
```

```
{Title:Data Engineer Salary:{Min:45000 Max:55000} Skills:[Python SQL]} <nil>
```

#### `FromDump.keysAndStrings`

```go
doc := scalar.MustFromDump(`{
	bare_word   => 1,
	'single'    => 'it\'s \n literal',   # only \\ and \' are escapes here
	"double"    => "tab\there\x{263a}",   # Perl's escapes
	42          => "a number as a key",
	-1          => "so is a negative one",
	"log.level" => "debug",
	email       => "jo@example.com",       # $ and @ aren't interpolated
}`)
fmt.Println(doc.HV().Keys().Sort(nil).Join(" "))
fmt.Printf("%q\n", doc.At("single").Str())
fmt.Printf("%q\n", doc.At("double").Str())
fmt.Println(doc.At("{42}"), "|", doc.At("{-1}"), "|", doc.At("{log.level}"), "|", doc.At("email"))
```

```
-1 42 bare_word double email log.level single
"it's \\n literal"
"tab\there☺"
a number as a key | so is a negative one | debug | jo@example.com
```

#### `FromDump.bytes`

```go
doc := scalar.MustFromDump(`["\211PNG\r\n", "caf\x{e9}", "caf\351"]`)
for _, s := range doc.AV().All() {
	fmt.Println(s.Kind() == scalar.Bytes, s.Length()) // octal above 127: bytes; \x{...}: text
}
fmt.Println(doc.At("[1]").Eq(doc.At("[2]"))) // the same characters, so eq
```

```
true 6
false 4
true 4
true
```

#### `FromDump.shared`

```go
// $VAR1 paths are the arrays and hashes already read: shared parts, and loops.
doc := scalar.MustFromDump(`{
	defaults => { cpu => "500m" },
	web      => { limits => $VAR1->{defaults} },
	worker   => { limits => $VAR1->{defaults} },
	parent   => $VAR1,
}`)
doc.At("defaults.cpu").Set("1")
fmt.Println(doc.At("web.limits.cpu"), doc.At("worker.limits.cpu")) // one hash, seen from two places
fmt.Println(doc.At("parent.parent.parent.web.limits.cpu"))         // a loop back to the top
```

```
1 1
1
```

#### `FromDump.fromPerl`

```go
// Data::Dumper's default layout, pasted from a Perl debugging session,
// with a blessed object: read as the plain hash it is.
dump := `bless( {
                 'name' => 'Ada',
                 'langs' => [
                              'Perl',
                              'Go'
                            ],
                 'since' => 1843
               }, 'Person' )`
p, err := scalar.FromDump(dump)
fmt.Println(err, p.At("name"), p.At("langs").AV().Join("/"), p.At("since").Add(183))
```

```
<nil> Ada Perl/Go 2026
```

#### `FromDump.types`

```go
// A dump doesn't record types: each value comes back as its text says,
// as Perl reads a dump back.
doc := scalar.MustFromDump(`[42, "42", "2.5", 2.5, "12345678901", !!1, undef]`)
names := map[scalar.Kind]string{scalar.Int: "int", scalar.String: "string", scalar.Float: "float",
	scalar.Bool: "bool", scalar.Undef: "undef"}
fmt.Println(doc.AV().Map(func(v *scalar.SV) any { return names[v.Kind()] }).Join(" "))
fmt.Println(scalar.Value("42").Dump(), scalar.Value(2.5).Dump()) // so Dump's "42" reads back as 42, and 2.5 as "2.5"
```

```
int string string float string bool undef
42 "2.5"
```

#### `FromDump.errors`

```go
_, err := scalar.FromDump(`{
	title  => "Go Developer",
	skills => ["Go" "SQL"],
}`)
fmt.Println(err) // a line and column to look at
_, err = scalar.FromDump(`{ handler => sub { "DUMMY" } }`)
fmt.Println(err) // data only: no Perl is ever run
```

```
scalar: bad dump at line 3, column 19: expected a comma or "]", found '"'
scalar: bad dump at line 1, column 14: sub can't be read: FromDump reads data, not Perl code
```

#### `SV.Bytes`

Bytes returns s as a Go []byte: a byte string's bytes, or anything else's bytes in string context (see Str), so text gives its UTF-8 encoding.

```go
png := scalar.Value([]byte("\x89PNG\r\n\x1a\n..."))   // a []byte makes a byte string
fmt.Println(png.Kind() == scalar.Bytes, png.Length()) // Length counts bytes
fmt.Println(png.M(`^\x89PNG`) != nil)                 // patterns match bytes
fmt.Printf("%x\n", png.Substr(0, 4))                  // %x gives the bytes in hex
blob := png.Bytes()                                   // back to a Go []byte (a copy)
fmt.Println(len(blob))
fmt.Println(scalar.Value("café").Bytes()) // text gives its UTF-8 bytes
```

```
true 11
true
89504e47
11
[99 97 102 195 169]
```

#### `SV.Format`

Format implements `fmt.Formatter`, so that Go's format verbs treat s as Perl's sprintf does, in the context each verb implies: %d, %b, %o, %x, %X and %c in integer context (see Int), %e, %E, %f, %F, %g and %G in numeric context (see Num), %t in boolean context, and %s, %q, %v and anything else in string context (see Str).

```go
fmt.Printf("%d|%05.2f|%x|%5s|%t\n",
	scalar.Value("42 apples"), scalar.Value("3.14159"), scalar.Value(255),
	scalar.Value("ab"), scalar.Value("0"))
```

```
42|03.14|ff|   ab|false
```

#### `SV.Num`

Num returns s in numeric context.

```go
fmt.Println(scalar.Value("3.5kg").Num()) // the numeric prefix, as Perl does
fmt.Println(scalar.Value("abc").Num())
```

```
3.5
0
```

#### `SV.NumOK`

NumOK is Num, plus false wherever Perl would warn: undef, empty or non-numeric strings, and strings with trailing garbage.

```go
for _, x := range []any{"42", " 42 ", "42abc", "", nil} {
	n, ok := scalar.Value(x).NumOK()
	fmt.Println(n, ok) // ok is false wherever Perl would warn
}
```

```
42 true
42 true
42 false
0 false
0 false
```

#### `SV.Int`

Int returns s in integer context, truncating toward zero.

```go
fmt.Println(scalar.Value(-3.9).Int()) // truncates toward zero
fmt.Println(scalar.Value("12 items").Int())
```

```
-3
12
```

#### `SV.IntOK`

IntOK is Int, plus false wherever NumOK would be false, and for NaN or floats outside the int64 range (which are clamped).

```go
i, ok := scalar.Value("9223372036854775807").IntOK() // exact, however large
fmt.Println(i, ok)
i, ok = scalar.Value(1e30).IntOK() // out of range: clamped
fmt.Println(i, ok)
```

```
9223372036854775807 true
9223372036854775807 false
```

#### `SV.Bool`

Bool returns s in boolean context: undef, "", "0", 0 and 0.0 are false; everything else, including "0.0" and "00", is true.

```go
for _, x := range []any{nil, "", "0", 0, 0.0, "0.0", "00", " ", "a"} {
	fmt.Print(scalar.Value(x).Bool(), " ")
}
fmt.Println()
```

```
false false false false false true true true true
```

#### `SV.AV`

AV dereferences s as an array.

```go
var s scalar.SV   // undef
s.AV().Push(1, 2) // push @$s, 1, 2: autovivifies an array
fmt.Println(s.AV().Join(","))
fmt.Println(scalar.Value("text").AV() == nil) // not an array
```

```
1,2
true
```

#### `SV.HV`

HV dereferences s as a hash, autovivifying undef like AV does.

```go
var s scalar.SV
s.HV().Set("a", 1) // $s->{a} = 1
fmt.Println(s.Ref(), s.HV().Get("a"))
```

```
HASH 1
```

#### `SV.Path`

Path walks a chain of hash keys (string) and array indices (int), so `s.Path`("content", 0, "text") is $s->{content}[0]{text}.

```go
doc, _ := scalar.FromJSON([]byte(`{"items": [{"name": "first"}]}`))
fmt.Println(doc.Path("items", 0, "name"))  // $doc->{items}[0]{name}
fmt.Println(doc.Path("items", -1, "name")) // negative indices count from the end
doc.Path("meta", "count").Set(1)           // writing autovivifies
fmt.Println(doc.Path("meta", "count"))
```

```
first
first
1
```

#### `SV.PathOK`

PathOK is Path with Go's comma-ok: ok reports whether every step exists, so a value that's present but null (undef) gives ok true, while a missing key or index, or a step through something that isn't an array or hash, gives ok false and an undef scalar.

```go
doc, _ := scalar.FromJSON([]byte(`{"salary": null}`))
_, ok := doc.PathOK("salary") // present, but null
fmt.Println(ok)
_, ok = doc.PathOK("bonus") // missing: and nothing is created
fmt.Println(ok, doc.HV().Exists("bonus"))
```

```
true
false false
```

#### `SV.At`

At is Path with the route as one string, in dot, bracket or Perl's brace form:

```go
doc, _ := scalar.FromJSON([]byte(`{
	"content": [{"rule_set": "A"}, {"rule_set": "B"}],
	"meta": {"log.level": "debug", "0": "zero"}
}`))
fmt.Println(doc.At("content[0].rule_set"))         // $doc->{content}[0]{rule_set}
fmt.Println(doc.At("content.1.rule_set"))          // a number after a dot: an index into an array
fmt.Println(doc.At("content[-1].rule_set"))        // from the end
fmt.Println(doc.At(`meta["log.level"]`))           // a key with a dot in it, quoted
fmt.Println(doc.At("meta.0"), doc.At(`meta["0"]`)) // into a hash, the number is a key
fmt.Println(doc.At("{content}[1]{rule_set}"))      // Perl's braces: always a hash key
fmt.Println(doc.At("meta{log.level}"))             // inside braces, dots are part of the key

var s scalar.SV
s.At("rules[0].name").Set("first") // writing autovivifies: [0] makes an array
out, _ := s.MarshalJSON()
fmt.Println(string(out))
```

```
A
B
B
debug
zero zero
B
debug
{"rules":[{"name":"first"}]}
```

#### `SV.AtOK`

AtOK is PathOK with the route as one string, written as for At: ok reports whether every step exists, so a value that's present but null gives ok true.

```go
doc, _ := scalar.FromJSON([]byte(`{"salary": {"min": 45000, "max": null}}`))
_, ok := doc.AtOK("salary.max") // present, but null
fmt.Println(ok)
_, ok = doc.AtOK("salary.bonus") // missing: and nothing is created
fmt.Println(ok, doc.At("salary").HV().Exists("bonus"))
```

```
true
false false
```

#### `SV.DeleteAt`

DeleteAt is Perl's delete $s->{content}[0]{rule_set}, with the route written as for At: it removes the last step's key or element, and returns its value, or undef if there was nothing there.

```go
doc, _ := scalar.FromJSON([]byte(`{"salary": {"min": 45000, "bonus": "TBC"}, "skills": ["Go", "SQL"]}`))
fmt.Println(doc.DeleteAt("salary.bonus")) // delete $doc->{salary}{bonus}: its value
fmt.Println(doc.DeleteAt("skills[-1]"))   // the last element: the array shrinks
fmt.Println(doc.DeleteAt("benefits.car")) // nothing there: undef, and nothing created
fmt.Println(doc.Dump())
```

```
TBC
SQL

{
  salary => {
    min => 45000,
  },
  skills => [
    "Go",
  ],
}
```

#### `MustPath`

MustPath is CompilePath, panicking if the path is malformed.

```go
rule := scalar.MustPath("content[0].rule_set") // parsed once, like a compiled regex
doc, _ := scalar.FromJSON([]byte(`{"content": [{"rule_set": {"name": "A", "ids": [7, 8]}}]}`))
fmt.Println(doc.Path(rule, "name"))   // a compiled path, then more keys
fmt.Println(doc.Path(rule, "ids", 1)) // ...{ids}[1]
_, ok := doc.PathOK(rule, "missing")
fmt.Println(ok)
```

```
A
8
false
```

#### `CompilePath`

CompilePath compiles an At path, or returns an error if it's malformed, for paths that come from outside the program.

```go
p, err := scalar.CompilePath("content..rule_set") // a path from outside the program: check it
fmt.Println(p == nil, err)
```

```
true scalar: bad path "content..rule_set" at offset 8: expected a name
```

#### `PathExpr.String`

String returns the path as written.

```go
fmt.Println(scalar.MustPath(`meta["log.level"]`))
```

```
meta["log.level"]
```

#### `StrCmp`

StrCmp is Perl's cmp, for use with `AV.Sort`.

```go
fmt.Println(scalar.Array("b", "10", "a", "9").Sort(scalar.StrCmp).Join(" ")) // sort { $a cmp $b }
```

```
10 9 a b
```

#### `NumCmp`

NumCmp is Perl's <=>, for use with `AV.Sort`.

```go
fmt.Println(scalar.Array("10", 9, "2.5").Sort(scalar.NumCmp).Join(" ")) // sort { $a <=> $b }
```

```
2.5 9 10
```

#### `SV.DeletePath`

DeletePath is DeleteAt with the route as Path takes it: keys taken whole, dots and all, so it suits keys that come from data, as delete $s->{$section}{$key} does in Perl.

```go
doc, _ := scalar.FromJSON([]byte(`{"pay":{"min":45000,"max":55000},"v1.2":{"note":"x"}}`))
fmt.Println(doc.DeletePath("pay", "max"))   // delete $doc->{pay}{max}
fmt.Println(doc.DeletePath("v1.2", "note")) // keys are taken whole, dots and all
fmt.Println(doc.At("pay").HV().KeysSlice(), doc.DeletePath("pay", "bonus").Defined())
```

```
55000
x
[min] false
```

#### `SV.Undef`

Undef is Perl's undef $s: s becomes undef, and Undef returns it, now undef.

```go
h := scalar.Hash("bonus", 5000)
fmt.Println(h.Get("bonus").Undef().Defined()) // undef $h{bonus}: returns it, now undef
fmt.Println(h.Exists("bonus"))                // the key stays, as in Perl; Delete removes it
```

```
false
true
```

#### `SV.Lock`

Lock makes s, and everything it refers to, read-only, all the way down, and returns s; see the README's Locking section.

Lock makes a structure read-only all the way down, as Perl's Hash::Util does a hash: its keys are fixed, so even reading one it hasn't got panics, where it would have autovivified, and a misspelled key is caught at once.

```go
doc, _ := scalar.FromJSON([]byte(`{"title": "Care worker", "pay": {"min": 25000}}`))
doc.Lock()
fmt.Println(doc.At("pay.min"))
fmt.Println(catch(func() { doc.At("pay.max").Set(30000) }))
fmt.Println(catch(func() { doc.At("titel") })) // misspelled
```

```
25000
scalar: Attempt to access disallowed key 'max' in a restricted hash
scalar: Attempt to access disallowed key 'titel' in a restricted hash
```

#### `SV.Unlock`

Unlock undoes Lock, all the way down, and returns s.

Unlock undoes Lock, all the way down.

```go
doc, _ := scalar.FromJSON([]byte(`{"pay": {"min": 25000}}`))
doc.Lock().Unlock()
doc.At("pay.max").Set(30000)
fmt.Println(doc.At("pay.max"))
```

```
30000
```

#### `SV.Locked`

Locked reports whether Lock has made s read-only.

Locked reports whether Lock has made a value read-only; a Clone isn't.

```go
doc := scalar.Value(map[string]any{"a": 1}).Lock()
fmt.Println(doc.Locked(), doc.Clone().Locked())
```

```
true false
```

### Reference: Strings

#### `SV.Length`

Length is Perl's length($s): the number of characters (not bytes) in s in string context, so Value(12345).Length() is 5 and "café" is 4.

```go
fmt.Println(scalar.Value("café").Length()) // characters, not bytes
fmt.Println(scalar.Value(12345).Length())  // in string context
```

```
4
5
```

#### `SV.Substr`

Substr is Perl's substr($s, offset) or substr($s, offset, length): a new scalar holding part of s in string context, counted in characters.

```go
s := scalar.Value("Senior Go Developer")
fmt.Println(s.Substr(7))            // substr($s, 7)
fmt.Println(s.Substr(7, 2))         // substr($s, 7, 2)
fmt.Println(s.Substr(-9))           // from the end
fmt.Println(s.Substr(0, -10))       // all but the last 10
fmt.Println(s.Substr(99).Defined()) // entirely outside: undef
```

```
Go Developer
Go
Developer
Senior Go
false
```

#### `SV.SubstrReplace`

SubstrReplace is Perl's 4-argument substr($s, offset, length, replacement), or assigning to substr: it replaces that part of s with replacement (in string context), changing s in place, and returns the part it replaced.

```go
s := scalar.Value("Junior Go Developer")
old := s.SubstrReplace(0, 6, "Senior") // substr($s, 0, 6) = "Senior"
fmt.Println(s, "/", old)
```

```
Senior Go Developer / Junior
```

#### `SV.Index`

Index is Perl's index($s, substr[, position]): the character position of the first occurrence of substr at or after position (default 0), or -1.

```go
s := scalar.Value("café au lait")
fmt.Println(s.Index("au"))   // a character position, ready for Substr
fmt.Println(s.Index("a", 2)) // starting from position 2
fmt.Println(s.Index("tea"))
```

```
5
5
-1
```

#### `SV.RIndex`

RIndex is Perl's rindex($s, substr[, position]): the character position of the last occurrence of substr starting at or before position (default the end of the string), or -1.

```go
s := scalar.Value("a/b/c")
fmt.Println(s.RIndex("/"))    // the last one
fmt.Println(s.RIndex("/", 2)) // the last one at or before position 2
```

```
3
1
```

#### `SV.Lc`

Lc is Perl's lc: s in lower case, using Unicode's full case mappings.

```go
fmt.Println(scalar.Value("ÉCOLE Go").Lc())
```

```
école go
```

#### `SV.Uc`

Uc is Perl's uc: s in upper case, using Unicode's full case mappings, so "straße" becomes "STRASSE".

```go
fmt.Println(scalar.Value("straße").Uc()) // full Unicode mappings
```

```
STRASSE
```

#### `SV.Fc`

Fc is Perl's fc: s case-folded, for comparing strings regardless of case.

```go
a, b := scalar.Value("Straße"), scalar.Value("STRASSE")
fmt.Println(a.Lc().Eq(b.Lc())) // lc isn't enough for this
fmt.Println(a.Fc().Eq(b.Fc())) // fc is: compare case-insensitively with it
```

```
false
true
```

#### `SV.UcFirst`

UcFirst is Perl's ucfirst: s with its first character in title case.

```go
fmt.Println(scalar.Value("élan vital").UcFirst())
```

```
Élan vital
```

#### `SV.LcFirst`

LcFirst is Perl's lcfirst: s with its first character in lower case.

```go
fmt.Println(scalar.Value("HELLO").LcFirst())
```

```
hELLO
```

#### `SV.Trim`

Trim is Perl's trim (builtin::trim, Perl 5.36): s without leading or trailing whitespace, including Unicode spaces such as no-break space.

```go
fmt.Printf("%q\n", scalar.Value("\u00a0 padded \t").Trim()) // Unicode spaces too
```

```
"padded"
```

#### `QuoteMeta`

QuoteMeta is Perl's quotemeta: x in string context with every character that could mean something in a pattern backslash-escaped, so that it matches literally.

```go
fmt.Println(scalar.QuoteMeta("C++ (.NET)"))
posting := scalar.Value("Skills: C++ and .NET")
fmt.Println(posting.M(scalar.QuoteMeta("C++")) != nil) // a literal match
```

```
C\+\+\ \(\.NET\)
true
```

#### `SV.Interpolate`

Interpolate fills in a template from s, as Perl interpolates variables in a double-quoted string, with the keys of the hash s holds as the variables:

```go
doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "location": "Leeds",
	"skills": ["Go", "SQL"], "salary": {"max": 55000}}`))
fmt.Println(doc.Interpolate("$title in $location: @skills"))   // doc's keys are the variables
fmt.Println(doc.Interpolate("Up to $salary{max}"))             // subscripts, as in Perl
fmt.Println(doc.Interpolate("Up to ${salary.max}"))            // any At path in ${...}
fmt.Println(doc.Interpolate(`[$bonus] \$5 jobs\@example.com`)) // missing is ""; \$ and \@ are literal
```

```
Go Developer in Leeds: Go SQL
Up to 55000
Up to 55000
[] $5 jobs@example.com
```

#### `HV.Interpolate`

Interpolate fills in a template from h; see `SV.Interpolate`.

```go
row := scalar.Hash("name", "Ada", "langs", scalar.Value([]string{"Go", "Perl"}))
fmt.Println(row.Interpolate("$name writes @langs"))
```

```
Ada writes Go Perl
```

#### `MustTemplate`

MustTemplate is CompileTemplate, panicking if the template is malformed.

```go
var summary = scalar.MustTemplate("$title. Skills: @skills.") // parsed once
doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "skills": ["Go", "SQL"]}`))
fmt.Println(doc.Interpolate(summary))
```

```
Go Developer. Skills: Go SQL.
```

#### `CompileTemplate`

CompileTemplate compiles a template for Interpolate, or returns an error if it's malformed, for templates that come from outside the program.

```go
t, err := scalar.CompileTemplate("Up to ${salary.max") // a template from outside the program: check it
fmt.Println(t == nil, err)
```

```
true scalar: bad template "Up to ${salary.max" at offset 7: no } to match the {
```

#### `Template.ListSeparator`

ListSeparator returns a copy of t that joins the elements of an array with sep, as local $" = sep does in Perl.

```go
t := scalar.MustTemplate("Skills: @skills").ListSeparator(", ") // local $" = ", "
fmt.Println(scalar.Hash("skills", scalar.Value([]string{"Go", "SQL"})).Interpolate(t))
```

```
Skills: Go, SQL
```

#### `Template.String`

String returns the template as written.

```go
fmt.Println(scalar.MustTemplate("$title in $location"))
```

```
$title in $location
```

### Reference: Operators

#### `SV.Add`

Add is Perl's +.

```go
fmt.Println(scalar.Value("40").Add(2))          // "40" + 2
fmt.Println(scalar.Value(1.5).Add("1.5"))       // floats when needed
fmt.Println(scalar.Value(math.MaxInt64).Add(1)) // an integer that would overflow becomes a float
```

```
42
3
9.22337203685478e+18
```

#### `SV.Sub`

Sub is Perl's -.

```go
fmt.Println(scalar.Value(10).Sub("2.5"))
```

```
7.5
```

#### `SV.Mul`

Mul is Perl's *.

```go
fmt.Println(scalar.Value("6").Mul(7))
```

```
42
```

#### `SV.Div`

Div is Perl's /.

```go
fmt.Println(scalar.Value(7).Div(2)) // not integer division
fmt.Println(scalar.Value(8).Div(2))
```

```
3.5
4
```

#### `SV.Mod`

Mod is Perl's %: both operands are truncated to integers, and a nonzero result takes the sign of the right operand (-7 % 3 is 2).

```go
fmt.Println(scalar.Value(7).Mod(3))
fmt.Println(scalar.Value(-7).Mod(3)) // takes the sign of the right operand
```

```
1
2
```

#### `SV.Pow`

Pow is Perl's **.

```go
fmt.Println(scalar.Value(2).Pow(10))
fmt.Println(scalar.Value(2).Pow(-1))
```

```
1024
0.5
```

#### `SV.Neg`

Neg is Perl's unary minus, string madness included: -"foo" is "-foo", -"-foo" is "+foo", and -"+foo" is "-foo".

```go
fmt.Println(scalar.Value(5).Neg())
fmt.Println(scalar.Value("foo").Neg()) // string madness, as in Perl
fmt.Println(scalar.Value("-foo").Neg())
```

```
-5
-foo
+foo
```

#### `SV.Concat`

Concat is Perl's .

```go
fmt.Println(scalar.Value("Go").Concat(1.5)) // "Go" . 1.5
```

```
Go1.5
```

#### `SV.Repeat`

Repeat is Perl's x operator on a scalar.

```go
fmt.Println(scalar.Value("ab").Repeat(3)) // "ab" x 3
fmt.Printf("%q\n", scalar.Value("ab").Repeat(0))
```

```
ababab
""
```

#### `SV.Inc`

Inc is Perl's ++: it modifies s in place and returns it.

```go
n := scalar.Value(9)
n.Inc()
id := scalar.Value("az9")
id.Inc() // Perl's magic string increment
fmt.Println(n, id)
```

```
10 ba0
```

#### `SV.Dec`

Dec is Perl's --: it modifies s in place and returns it.

```go
s := scalar.Value("aa")
s.Dec() // never magic: "aa" is 0 as a number
fmt.Println(s)
```

```
-1
```

#### `SV.AddInPlace`

AddInPlace is Perl's +=.

```go
totals := scalar.Hash()
for _, n := range []int{5, 10} {
	totals.Get("sum").AddInPlace(n) // $totals{sum} += $n
}
fmt.Println(totals.Get("sum"))
```

```
15
```

#### `SV.SubInPlace`

SubInPlace is Perl's -=.

```go
s := scalar.Value(10)
s.SubInPlace(3) // $s -= 3
fmt.Println(s)
```

```
7
```

#### `SV.MulInPlace`

MulInPlace is Perl's *=.

```go
s := scalar.Value(4)
s.MulInPlace(2.5) // $s *= 2.5
fmt.Println(s)
```

```
10
```

#### `SV.DivInPlace`

DivInPlace is Perl's /=.

```go
s := scalar.Value(9)
s.DivInPlace(2) // $s /= 2
fmt.Println(s)
```

```
4.5
```

#### `SV.ModInPlace`

ModInPlace is Perl's %=.

```go
s := scalar.Value(17)
s.ModInPlace(5) // $s %= 5
fmt.Println(s)
```

```
2
```

#### `SV.PowInPlace`

PowInPlace is Perl's **=.

```go
s := scalar.Value(3)
s.PowInPlace(2) // $s **= 2
fmt.Println(s)
```

```
9
```

#### `SV.ConcatInPlace`

ConcatInPlace is Perl's .=.

```go
s := scalar.Value("Go")
s.ConcatInPlace(" developer") // $s .= " developer"
fmt.Println(s)
```

```
Go developer
```

#### `SV.RepeatInPlace`

RepeatInPlace is Perl's x=.

```go
s := scalar.Value("-")
s.RepeatInPlace(5) // $s x= 5
fmt.Println(s)
```

```
-----
```

#### `SV.Eq`

Eq is Perl's eq.

```go
fmt.Println(scalar.Value(1.0).Eq("1")) // eq compares strings: 1.0 is "1"
fmt.Println(scalar.Value("1.0").Eq(1))
```

```
true
false
```

#### `SV.Ne`

Ne is Perl's ne.

```go
fmt.Println(scalar.Value("go").Ne("Go"))
```

```
true
```

#### `SV.Lt`

Lt is Perl's lt.

```go
fmt.Println(scalar.Value("10").Lt("9")) // as strings, "10" sorts first
```

```
true
```

#### `SV.Le`

Le is Perl's le.

```go
fmt.Println(scalar.Value("a").Le("a"))
```

```
true
```

#### `SV.Gt`

Gt is Perl's gt.

```go
fmt.Println(scalar.Value("b").Gt("abc"))
```

```
true
```

#### `SV.Ge`

Ge is Perl's ge.

```go
fmt.Println(scalar.Value("abc").Ge("abd"))
```

```
false
```

#### `SV.NumEq`

NumEq is Perl's ==.

```go
fmt.Println(scalar.Value("1.0").NumEq(1)) // == compares numbers
fmt.Println(scalar.Value("NaN").NumEq("NaN"))
```

```
true
false
```

#### `SV.NumNe`

NumNe is Perl's !=.

```go
fmt.Println(scalar.Value("10").NumNe(10.0))
```

```
false
```

#### `SV.NumLt`

NumLt is Perl's <.

```go
fmt.Println(scalar.Value("10").NumLt("9")) // as numbers, 9 is smaller
```

```
false
```

#### `SV.NumLe`

NumLe is Perl's <=.

```go
fmt.Println(scalar.Value(2).NumLe("2.0"))
```

```
true
```

#### `SV.NumGt`

NumGt is Perl's >.

```go
fmt.Println(scalar.Value("9007199254740993").NumGt("9007199254740992")) // integers compare exactly
```

```
true
```

#### `SV.NumGe`

NumGe is Perl's >=.

```go
fmt.Println(scalar.Value(-1).NumGe(0))
```

```
false
```

### Reference: Arrays

#### `Array`

Array returns an array of vals, as Perl's (...) list does: arrays and hashes among vals are flattened into it (see Push), and other values are converted as by `SV.Set`.

```go
a := scalar.Array("a", []string{"b", "c"}, scalar.Array("d")) // lists flatten, as in Perl
fmt.Println(a.Length(), a.Join(","))
nested := scalar.Array(1, scalar.Value(scalar.Array(2, 3))) // a reference stays one element
fmt.Println(nested.Length(), nested.Get(1).AV().Join(","))
```

```
4 a,b,c,d
2 2,3
```

#### `AV.String`

String returns a on one line, as Dump does but with Indent = 0, so `fmt.Println`(a) shows its elements, nested arrays and hashes included: [1,"a",[2]].

```go
a := scalar.Array("Go", 1, scalar.Value(scalar.Array(2, 3)))
fmt.Println(a) // fmt uses String: the elements, on one line
```

```
["Go",1,[2,3]]
```

#### `AV.Dump`

Dump returns a as Data::Dumper prints \@a; see `SV.Dump`.

```go
fmt.Println(scalar.Array("Go", nil, true).Dump())
```

```
[
  "Go",
  undef,
  !!1,
]
```

#### `AV.Length`

Length is Perl's scalar(@a).

```go
fmt.Println(scalar.Array(1, 2, 3).Length()) // scalar(@a)
```

```
3
```

#### `AV.Scalar`

Scalar is Perl's scalar(@a), the number of elements.

```go
fmt.Println(scalar.Array("x", "y").Scalar()) // scalar(@a), the same as Length
```

```
2
```

#### `AV.LastIndex`

LastIndex is Perl's $#a: Length() - 1, so -1 for an empty array.

```go
fmt.Println(scalar.Array(1, 2, 3).LastIndex()) // $#a
fmt.Println(scalar.Array().LastIndex())
```

```
2
-1
```

#### `AV.SetLastIndex`

SetLastIndex is Perl's $#a = n.

```go
a := scalar.Array(1, 2, 3, 4)
a.SetLastIndex(1) // $#a = 1: truncate
fmt.Println(a.Join(","))
a.SetLastIndex(3) // extend with holes
fmt.Println(a.Length(), a.Exists(3))
```

```
1,2
4 false
```

#### `AV.Get`

Get returns element i; negative indices count from the end, as in Perl.

```go
a := scalar.Array("first", "second", "last")
fmt.Println(a.Get(0), a.Get(-1))             // $a[0], $a[-1]
fmt.Println(a.Get(10).Defined(), a.Length()) // reading doesn't create
a.Get(4).Set("new")                          // writing does, as $a[4] = "new"
fmt.Println(a.Length())
```

```
first last
false 3
5
```

#### `AV.Set`

Set assigns v to element i, extending the array if needed.

```go
a := scalar.Array()
a.Set(2, "c") // $a[2] = "c": extends the array, leaving holes
fmt.Println(a.Length(), a.Exists(0), a.Get(2))
```

```
3 false c
```

#### `AV.Exists`

Exists is Perl's exists $a[i]: whether element i has been set.

```go
a := scalar.Array("a", nil)
fmt.Println(a.Exists(1)) // set, though undef
fmt.Println(a.Exists(5))
```

```
true
false
```

#### `AV.Defined`

Defined is Perl's defined $a[i]: whether element i exists and isn't undef.

```go
a := scalar.Array("a", nil)
fmt.Println(a.Defined(0), a.Defined(1), a.Defined(-2))
```

```
true false true
```

#### `AV.Delete`

Delete is Perl's delete $a[i]: it returns the element (undef if it didn't exist) and leaves a hole.

```go
a := scalar.Array("a", "b", "c")
fmt.Println(a.Delete(1)) // delete $a[1]: leaves a hole
fmt.Println(a.Length(), a.Exists(1))
a.Delete(-1) // deleting the last element shrinks the array
fmt.Println(a.Length())
```

```
b
3 false
1
```

#### `AV.Push`

Push is Perl's push: it appends vals and returns a.

```go
a := scalar.Array("a")
a.Push("b", []string{"c", "d"})          // push @a, "b", @list
a.Push(scalar.Value(scalar.Array(1, 2))) // push @a, [1, 2]: a reference, one element
fmt.Println(a.Length(), a.Get(-1).Ref())
```

```
5 ARRAY
```

#### `AV.Unshift`

Unshift is Perl's unshift: it prepends vals, keeping their order, and returns a.

```go
a := scalar.Array("c")
a.Unshift("a", "b") // unshift @a, "a", "b": order kept
fmt.Println(a.Join(","))
```

```
a,b,c
```

#### `AV.Pop`

Pop removes and returns the last element, or undef if a is empty.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Pop(), a.Join(","))
fmt.Println(scalar.Array().Pop().Defined()) // nothing to pop: undef
```

```
3 1,2
false
```

#### `AV.Shift`

Shift removes and returns the first element, or undef if a is empty.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Shift(), a.Join(","))
```

```
1 2,3
```

#### `AV.PopN`

PopN removes the last n elements and returns them, in array order, as splice(@a, -n) does: Array(1, 2, 3, 4).PopN(2) gives 3, 4 and leaves 1, 2.

```go
a := scalar.Array("a", "b", "c", "d")
last := a.PopN(2) // splice(@a, -2): in array order
fmt.Println(last.Join(","), "|", a.Join(","))
fmt.Println(a.PopN(10).Join(","), "|", a.Length()) // more than there is: all of it
```

```
c,d | a,b
a,b | 0
```

#### `AV.ShiftN`

ShiftN removes the first n elements and returns them, in array order, as splice(@a, 0, n) does: Array(1, 2, 3, 4).ShiftN(2) gives 1, 2 and leaves 3, 4.

```go
queue := scalar.Array("job1", "job2", "job3", "job4", "job5")
for queue.Length() > 0 {
	batch := queue.ShiftN(2) // splice(@queue, 0, 2): the next batch
	fmt.Println(batch.Join(","))
}
```

```
job1,job2
job3,job4
job5
```

#### `AV.Splice`

Splice is Perl's splice @a, offset, length, list: it removes length elements starting at offset, inserts list in their place, and returns the removed elements.

```go
a := scalar.Array("a", "b", "c", "d", "e")
removed := a.Splice(1, 2, "X") // splice @a, 1, 2, "X"
fmt.Println(removed.Join(","), "|", a.Join(","))
a.Splice(-1, 1) // remove the last element
fmt.Println(a.Join(","))
```

```
b,c | a,X,d,e
a,X,d
```

#### `AV.Values`

Values is Perl's values @a: the elements.

```go
a := scalar.Array("a", "b", "c", "d")
fmt.Println(a.Values().Join(","))         // values @a
fmt.Println(a.Values(0, -1, 1).Join(",")) // @a[0, -1, 1]
picks := scalar.Array(2, 3)
fmt.Println(a.Values(picks).Join(","))    // @a[@picks]
for _, s := range a.Values(0, -1).All() { // $_ = uc for @a[0, -1]: the elements themselves
	s.Set(s.Uc())
}
fmt.Println(a.Join(","))
```

```
a,b,c,d
a,d,b
c,d
A,b,c,D
```

#### `AV.Pairs`

Pairs is Perl's index/value slice %a[i, j, ...]: a new array of index/value pairs for those indices, in that order, given as for Values.

```go
a := scalar.Array("a", "b")
fmt.Println(a.Pairs(0, -1).Join(",")) // %a[0, -1]: indices as written
fmt.Println(a.Pairs().Join(","))      // every index and element
```

```
0,a,-1,b
0,a,1,b
```

#### `AV.Items`

Items returns the elements.

```go
for _, s := range scalar.Array("a", "b").Items() { // a Go []*scalar.SV
	fmt.Print(s, " ")
}
fmt.Println()
```

```
a b
```

#### `AV.StrSlice`

StrSlice returns the elements as a Go []string, each in string context (see `SV.Str`), so undef and holes become "".

```go
fmt.Printf("%q\n", scalar.Array("a", 1, nil).StrSlice())
```

```
["a" "1" ""]
```

#### `AV.IntSlice`

IntSlice returns the elements as a Go []int64, each in integer context (see `SV.Int`), so values that aren't numbers become 0.

```go
fmt.Println(scalar.Array("3", 4.9, "x").IntSlice())
```

```
[3 4 0]
```

#### `AV.NumSlice`

NumSlice returns the elements as a Go []float64, each in numeric context (see `SV.Num`).

```go
fmt.Println(scalar.Array("1.5", 2).NumSlice())
```

```
[1.5 2]
```

#### `AV.Each`

Each is Perl's each @a (Perl 5.12 and later): the next index and element, with ok true, or ok false once every element has been returned, after which it starts again from the beginning.

```go
a := scalar.Array("x", "y")
for i, v, ok := a.Each(); ok; i, v, ok = a.Each() { // while (my ($i, $v) = each @a)
	fmt.Println(i, v)
}
```

```
0 x
1 y
```

#### `AV.All`

All iterates over index/element pairs.

```go
for i, s := range scalar.Array("a", "b").All() { // while (my ($i, $s) = each @a)
	fmt.Println(i, s)
}
```

```
0 a
1 b
```

#### `AV.Join`

Join is Perl's join.

```go
fmt.Println(scalar.Array("Go", "Perl", 5).Join(", "))
```

```
Go, Perl, 5
```

#### `AV.Map`

Map is Perl's map: a new array of f applied to each element.

```go
a := scalar.Array("go", "perl")
fmt.Println(a.Map(func(s *scalar.SV) any { return s.Uc() }).Join(","))
pairs := a.Map(func(s *scalar.SV) any { return scalar.Array(s, s.Length()) }) // several values each
fmt.Println(pairs.Join(","))
```

```
GO,PERL
go,2,perl,4
```

#### `AV.Grep`

Grep is Perl's grep: a new array of copies of the elements f accepts.

```go
n := scalar.Array(1, 2, 3, 4).Grep(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })
fmt.Println(n.Join(","))
```

```
2,4
```

#### `AV.Sort`

Sort is Perl's sort: a sorted copy of a.

```go
a := scalar.Array("10", "9", "2")
fmt.Println(a.Sort(nil).Join(","))           // sort @a: as strings
fmt.Println(a.Sort(scalar.NumCmp).Join(",")) // sort { $a <=> $b } @a
byLength := func(x, y *scalar.SV) int { return x.Length() - y.Length() }
fmt.Println(scalar.Array("ccc", "a", "bb").Sort(byLength).Join(","))
fmt.Println(a.Join(",")) // unchanged
```

```
10,2,9
2,9,10
a,bb,ccc
10,9,2
```

#### `AV.SortInPlace`

SortInPlace is Perl's @a = sort @a: it sorts a itself, as Sort would, and returns a.

```go
a := scalar.Array(3, 1, 2)
a.SortInPlace(scalar.NumCmp) // @a = sort { $a <=> $b } @a
fmt.Println(a.Join(","))
```

```
1,2,3
```

#### `AV.Reverse`

Reverse is Perl's reverse: a reversed copy of a.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Reverse().Join(","), a.Join(","))
```

```
3,2,1 1,2,3
```

#### `AV.ReverseInPlace`

ReverseInPlace is Perl's @a = reverse @a: it reverses a itself and returns a.

```go
a := scalar.Array(1, 2, 3)
a.ReverseInPlace()
fmt.Println(a.Join(","))
```

```
3,2,1
```

#### `AV.SplitAt`

SplitAt splits a into its first n elements and the rest, as Perl's my ($x1, ..., $xn, @tail) = @a does.

```go
head, tail := scalar.Array("Ada", "Lovelace", "maths", "poetry").SplitAt(2) // my ($first, $last, @interests) = @a
fmt.Println(head.Join(" "), "|", tail.Join(","))
```

```
Ada Lovelace | maths,poetry
```

#### `AV.SplitAtSlice`

SplitAtSlice is SplitAt with the head as a Go slice of exactly n elements, so it can be unpacked: h, tail := `a.SplitAtSlice`(3); x, y, z := h[0], h[1], h[2].

```go
h, rest := scalar.Array("2026", "10", "13", "extra").SplitAtSlice(3)
year, month, day := h[0], h[1], h[2] // my ($year, $month, $day, @rest) = @a
fmt.Println(year, month, day, rest.Length())
```

```
2026 10 13 1
```

#### `AV.Clone`

Clone returns a deep copy of a, as dclone(\@a) does; see `SV.Clone`.

```go
rows := scalar.Array(scalar.Value(map[string]any{"title": "Go Developer"})) // one row: a hash reference
mine := rows.Clone()                                                        // dclone(\@rows): the rows too, not just the list
mine.Get(0).At("title").Set("Perl Developer")
fmt.Println(rows.Get(0).At("title"), "/", mine.Get(0).At("title"))
```

```
Go Developer / Perl Developer
```

#### `AV.Equal`

Equal reports whether a and other hold the same data; see `SV.Equal`.

```go
a := scalar.Array("Go", 1)
fmt.Println(a.Equal([]any{"Go", "1"}), a.Equal([]any{"Go"}))
```

```
true false
```

#### `AV.Diff`

Diff says where a and other first differ; see `SV.Diff`.

```go
fmt.Println(scalar.Array("Go", "SQL").Diff([]string{"Go"}))
```

```
[1]: got "SQL", want does not exist
```

#### `AV.Apply`

Apply calls f on every plain value in a, all the way down, as rmap { ...

```go
prices := scalar.Array(10, scalar.Value([]any{20, 30}))
prices.Apply(func(v *scalar.SV) any { return v.Mul(1.2) }) // every price, nested ones too
fmt.Println(prices.Get(0), prices.Get(1).AV().Join(" "))
```

```
12 24 36
```

#### `AV.At`

At is `SV.At`, starting at a.

```go
rows, _ := scalar.FromJSON([]byte(`[{"name":"Ada"},{"name":"Grace"}]`))
a := rows.AV()
fmt.Println(a.At("[1].name")) // $a[1]{name}
a.At("[2].name").Set("Alan")  // $a[2]{name} = "Alan": creates the row
fmt.Println(a.Length(), a.At("[-1].name"))
```

```
Grace
3 Alan
```

#### `AV.AtOK`

AtOK is `SV.AtOK`, starting at a.

```go
a := scalar.Array(1, nil)
_, ok := a.AtOK("[1]") // exists, though undef
_, past := a.AtOK("[2]")
fmt.Println(ok, past)
```

```
true false
```

#### `AV.DeleteAt`

DeleteAt is `SV.DeleteAt`, starting at a.

```go
a := scalar.Array("a", "b", "c")
fmt.Println(a.DeleteAt("[-1]"), a.Length()) // delete $a[-1]: the last goes, so it's shorter
fmt.Println(a.DeleteAt("[0]"), a.Length())  // delete $a[0]: a hole, so it isn't
```

```
c 2
a 2
```

#### `AV.Path`

Path is `SV.Path`, starting at a.

```go
a := scalar.Array()
a.Path(0, "tags", 1).Set("go") // $a[0]{tags}[1] = "go"
fmt.Println(a.Path(0, "tags", 1), a.At("[0].tags").AV().Length())
```

```
go 2
```

#### `AV.PathOK`

PathOK is `SV.PathOK`, starting at a.

```go
a := scalar.Array(scalar.Value(scalar.Hash("id", 7)))
v, ok := a.PathOK(0, "id")
_, missing := a.PathOK(0, "name")
fmt.Println(v, ok, missing)
```

```
7 true false
```

#### `AV.DeletePath`

DeletePath is `SV.DeletePath`, starting at a.

```go
rows, _ := scalar.FromJSON([]byte(`[{"id":1,"tmp":"x"},{"id":2,"tmp":"y"}]`))
a := rows.AV()
for i := range a.Length() {
	a.DeletePath(i, "tmp") // delete $a[$i]{tmp}
}
fmt.Println(a.At("[0]").HV().KeysSlice(), a.At("[1]").HV().KeysSlice())
```

```
[id] [id]
```

#### `AV.Undef`

Undef is Perl's undef @a: it empties a in place, so everything referring to a sees it empty, and returns a, now empty.

```go
a := scalar.Array("x", "y")
doc := scalar.Hash("tags", scalar.Value(a)) // another reference to a
a.Undef()                                   // undef @a
fmt.Println(a.Length(), doc.At("tags").AV().Length())
```

```
0 0
```

#### `AV.Lock`

Lock locks a and everything in it, as `SV.Lock` does, and returns a.

A locked array can't grow, shrink or change, nor can its elements.

```go
a := scalar.Array(1, 2, 3).Lock()
fmt.Println(a.Get(0), a.Length())
fmt.Println(catch(func() { a.Push(4) }))
fmt.Println(catch(func() { a.Get(0).Set(9) }))
```

```
1 3
scalar: Modification of a read-only value attempted
scalar: Modification of a read-only value attempted
```

#### `AV.Unlock`

Unlock undoes Lock, as `SV.Unlock` does, and returns a.

```go
a := scalar.Array(1, 2).Lock().Unlock()
fmt.Println(a.Push(3).Length())
```

```
3
```

#### `AV.Locked`

Locked reports whether a is locked.

```go
a := scalar.Array(1, 2)
fmt.Println(a.Locked(), a.Lock().Locked())
```

```
false true
```

### Reference: Hashes

#### `Hash`

Hash builds a hash from a key/value list, as %h = (k1 => v1, ...) does.

```go
h := scalar.Hash("name", "Ada", "age", 36) // %h = (name => "Ada", age => 36)
fmt.Println(h.Get("name"), h.Get("age"))
merged := scalar.Hash(h, "age", 37) // %merged = (%h, age => 37): later keys win
fmt.Println(merged.Get("age"))
fmt.Println(scalar.Hash(map[string]int{"a": 1}).Get("a")) // from a Go map
```

```
Ada 36
37
1
```

#### `HV.String`

String returns h on one line, as Dump does but with Indent = 0, so `fmt.Println`(h) shows its keys and values: {a => 1,b => [2]}.

```go
h := scalar.Hash("title", "Go Developer", "log level", "debug")
fmt.Println(h) // keys sorted, and quoted only when they need to be
```

```
{"log level" => "debug",title => "Go Developer"}
```

#### `HV.Dump`

Dump returns h as Data::Dumper prints \%h; see `SV.Dump`.

```go
tags := scalar.Value(scalar.Array("go"))
h := scalar.Hash("tags", tags, "again", tags) // the same array twice
fmt.Println(h.Dump())
```

```
{
  again => [
    "go",
  ],
  tags => $VAR1->{again},
}
```

#### `HV.Length`

Length is Perl's scalar(keys %h).

```go
fmt.Println(scalar.Hash("a", 1, "b", 2).Length()) // scalar(keys %h)
```

```
2
```

#### `HV.Scalar`

Scalar is Perl's scalar(%h), which since Perl 5.26 is the number of keys.

```go
fmt.Println(scalar.Hash("a", 1).Scalar()) // scalar(%h), the same as Length
```

```
1
```

#### `HV.Get`

Get returns the value for key.

```go
h := scalar.Hash("a", 1)
fmt.Println(h.Get("a"))
fmt.Println(h.Get("b").Defined(), h.Exists("b")) // reading doesn't create
h.Get("c").HV().Set("d", 1)                      // $h{c}{d} = 1 autovivifies
fmt.Println(h.Get("c").Ref())
```

```
1
false false
HASH
```

#### `HV.Set`

Set assigns v to key and returns the stored SV.

```go
h := scalar.Hash()
h.Set("tags", []string{"go", "perl"}) // $h{tags} = ["go", "perl"]
fmt.Println(h.Get("tags").AV().Join(","))
```

```
go,perl
```

#### `HV.Push`

Push adds key/value pairs to h and returns h.

```go
h := scalar.Hash("title", "Developer")
h.Push("location", "Leeds", "salary", 45000) // $h{location} = "Leeds"; $h{salary} = 45000
defaults := scalar.Hash("remote", false, "salary", 0)
h.Push(defaults, "salary", 50000) // merge, later keys winning
fmt.Println(h.Keys().Sort(nil).Join(","))
fmt.Println(h.Get("salary"))
```

```
location,remote,salary,title
50000
```

#### `HV.Exists`

Exists is Perl's exists.

```go
h := scalar.Hash("salary", nil)
fmt.Println(h.Exists("salary"), h.Exists("bonus")) // there, though undef
```

```
true false
```

#### `HV.Defined`

Defined is Perl's defined $h{key}: whether key exists and its value isn't undef.

```go
h := scalar.Hash("salary", nil, "title", "Developer")
fmt.Println(h.Defined("salary"), h.Defined("title"))
```

```
false true
```

#### `HV.Delete`

Delete is Perl's delete: removes key and returns its value, or undef.

```go
h := scalar.Hash("a", 1, "b", 2)
fmt.Println(h.Delete("a"), h.Exists("a")) // returns the value it removed
fmt.Println(h.Delete("missing").Defined())
```

```
1 false
false
```

#### `HV.Clone`

Clone returns a deep copy of h, as dclone(\%h) does; see `SV.Clone`.

```go
defaults := scalar.Hash("remote", false, "tags", scalar.Value([]string{"go"}))
row := defaults.Clone() // dclone(\%defaults): nested arrays too
row.Get("tags").AV().Push("perl")
merged := scalar.Hash(defaults) // %row = %defaults: a new hash, but the same tags array
merged.Get("tags").AV().Push("sql")
fmt.Println(defaults.Get("tags").AV().Join(","), "/", row.Get("tags").AV().Join(","))
```

```
go,sql / go,perl
```

#### `HV.Equal`

Equal reports whether h and other hold the same data; see `SV.Equal`.

```go
h := scalar.Hash("remote", true, "days", 2)
fmt.Println(h.Equal(map[string]any{"remote": 1, "days": "2"})) // true is 1, as in Perl
```

```
true
```

#### `HV.Diff`

Diff says where h and other first differ; see `SV.Diff`.

```go
h := scalar.Hash("bonus", nil)
fmt.Println(h.Diff(map[string]any{})) // a key whose value is undef isn't a missing key
```

```
bonus: got undef, want does not exist
```

#### `HV.Apply`

Apply calls f on every plain value in h, all the way down, as rmap { ...

```go
row := scalar.Hash("title", "go developer", "remote", true)
row.Apply(func(v *scalar.SV) any {
	if v.Kind() == scalar.String {
		return v.UcFirst()
	}
	return v
})
fmt.Println(row.Get("title"), row.Get("remote"))
```

```
Go developer 1
```

#### `HV.Pairs`

Pairs is Perl's %h in list context: a new array of key/value pairs, k1, v1, k2, v2 and so on, in the same order as Keys.

```go
h2 := scalar.Hash("k1", 1, "k2", 2, "k3", 3)
h1 := scalar.Hash(h2.Pairs("k1", "k3")) // %h1 = %h2{qw(k1 k3)}
fmt.Println(h1.Keys().Sort(nil).Join(","), h1.Get("k3"))
fmt.Println(h2.Pairs("k2", "missing").Join(","))      // a missing key comes with undef
fmt.Println(scalar.Hash("only", 1).Pairs().Join(",")) // every pair: %h in list context
```

```
k1,k3 3
k2,2,missing,
only,1
```

#### `HV.PairsMap`

PairsMap returns h as a Go map from its keys to its own values, as Pairs() would list them, for code that doesn't use this package.

```go
h := scalar.Hash("a", 1)
m := h.PairsMap() // a Go map[string]*scalar.SV of the hash's own values
m["a"].Set(2)
fmt.Println(len(m), h.Get("a"))
```

```
1 2
```

#### `HV.Keys`

Keys is Perl's keys %h: a new array of the keys.

```go
h := scalar.Hash("b", 2, "a", 1)
fmt.Println(h.Keys().Sort(nil).Join(",")) // sort keys %h
```

```
a,b
```

#### `HV.Values`

Values is Perl's values %h: the values, in the same order as Keys.

```go
h := scalar.Hash("a", 1, "b", 2, "c", 3)
fmt.Println(h.Values("c", "a").Join(","))                     // @h{qw(c a)}
fmt.Println(h.Values("missing").Get(0).Defined(), h.Length()) // undef, and not created
for _, v := range h.Values().All() {                          // $_ *= 10 for values %h
	v.MulInPlace(10)
}
fmt.Println(h.Values("a", "b", "c").Join(","))
```

```
3,1
false 3
10,20,30
```

#### `HV.KeysSlice`

KeysSlice is Keys as a Go []string.

```go
keys := scalar.Hash("a", 1).KeysSlice() // a Go []string
fmt.Printf("%q\n", keys)
```

```
["a"]
```

#### `HV.ValuesSlice`

ValuesSlice is Values() as a Go []*SV.

```go
values := scalar.Hash("a", "x").ValuesSlice() // a Go []*scalar.SV
fmt.Println(len(values), values[0])
```

```
1 x
```

#### `HV.Each`

Each is Perl's each %h: the next key/value pair, with ok true, or ok false once every pair has been returned, after which it starts again from the beginning.

```go
h := scalar.Hash("a", 1, "b", 2, "c", 3)
sum := 0
for _, v, ok := h.Each(); ok; _, v, ok = h.Each() { // while (my ($k, $v) = each %h)
	sum += int(v.Int())
}
fmt.Println(sum)

// As in Perl, the hash remembers where Each got to, until Keys and the
// like reset it.
first, _, _ := h.Each()
h.Keys() // keys %h resets the iterator
again, _, _ := h.Each()
fmt.Println(first == again)
```

```
6
true
```

#### `HV.All`

All iterates over key/value pairs in the same order as Keys.

```go
h := scalar.Hash("a", 1, "b", 2)
sum := 0
for _, v := range h.All() { // while (my ($k, $v) = each %h)
	sum += int(v.Int())
}
fmt.Println(sum)
```

```
3
```

#### `HV.At`

At is `SV.At`, starting at h.

```go
tallies := scalar.Hash()
tallies.At("benefits.right").Inc() // $tallies{benefits}{right}++
tallies.At("benefits.right").Inc()
tallies.At("benefits.missing").Inc()
fmt.Println(tallies.At("benefits.right"), tallies.At("benefits.missing"), tallies.At("summary.right").Int())
```

```
2 1 0
```

#### `HV.AtOK`

AtOK is `SV.AtOK`, starting at h.

```go
h := scalar.Hash("pay", scalar.Value(scalar.Hash("min", nil)))
_, ok := h.AtOK("pay.min") // exists, though undef
_, missing := h.AtOK("pay.max")
fmt.Println(ok, missing)
```

```
true false
```

#### `HV.DeleteAt`

DeleteAt is `SV.DeleteAt`, starting at h.

```go
h := scalar.Hash("pay", scalar.Value(scalar.Hash("min", 45000, "max", 55000)))
fmt.Println(h.DeleteAt("pay.max")) // delete $h{pay}{max}
fmt.Println(h.At("pay").HV().KeysSlice())
```

```
55000
[min]
```

#### `HV.Path`

Path is `SV.Path`, starting at h.

```go
h := scalar.Hash()
h.Path("a.b", "c").Set(1) // $h{"a.b"}{c} = 1: Path keys are taken whole, dots and all
fmt.Println(h.Path("a.b", "c"), h.At(`["a.b"].c`))
```

```
1 1
```

#### `HV.PathOK`

PathOK is `SV.PathOK`, starting at h.

```go
h := scalar.Hash("tags", scalar.Value(scalar.Array("go")))
v, ok := h.PathOK("tags", 0)
_, past := h.PathOK("tags", 5)
fmt.Println(v, ok, past)
```

```
go true false
```

#### `HV.DeletePath`

DeletePath is `SV.DeletePath`, starting at h.

```go
section, key := "pay", "max" // from data
h := scalar.Hash("pay", scalar.Value(scalar.Hash("min", 45000, "max", 55000)))
fmt.Println(h.DeletePath(section, key)) // delete $h{$section}{$key}
```

```
55000
```

#### `HV.Undef`

Undef is Perl's undef %h: it empties h in place, so everything referring to h sees it empty, and returns h, now empty.

```go
seen := scalar.Hash("go", 1, "perl", 1)
seen.Undef() // undef %seen: emptied, ready for the next batch
seen.Set("raku", 1)
fmt.Println(seen.KeysSlice())
```

```
[raku]
```

#### `HV.Map`

Map builds a new hash from h, as %new = map { f($_, $h{$_}) } keys %h does: f gets each key and value, and returns the new key and value.

Map builds a new hash, renaming keys or changing values or both. The old hash is unchanged.

```go
pay := scalar.Hash("min", 40000, "max", 50000)
k := pay.Map(func(key string, v *scalar.SV) (string, any) {
	return "pay_" + key, v.Div(1000).Str() + "k"
})
fmt.Println(k.Get("pay_min"), k.Get("pay_max"), pay.Get("min"))
```

```
40k 50k 40000
```

#### `HV.Lock`

Lock locks h and everything in it, as `SV.Lock` does, and returns h.

A locked hash keeps the keys it has: reading, writing or deleting one it hasn't got panics, as in a Perl restricted hash; Exists still answers.

```go
h := scalar.Hash("min", 25000).Lock()
fmt.Println(h.Get("min"), h.Exists("max"))
fmt.Println(catch(func() { h.Get("max") }))
fmt.Println(catch(func() { h.Delete("min") }))
```

```
25000 false
scalar: Attempt to access disallowed key 'max' in a restricted hash
scalar: Attempt to delete readonly key 'min' from a restricted hash
```

#### `HV.Unlock`

Unlock undoes Lock, as `SV.Unlock` does, and returns h.

```go
h := scalar.Hash("min", 25000).Lock().Unlock()
h.Set("max", 30000)
fmt.Println(h.Get("max"))
```

```
30000
```

#### `HV.Locked`

Locked reports whether h is locked.

```go
h := scalar.Hash("min", 25000)
fmt.Println(h.Locked(), h.Lock().Locked())
```

```
false true
```

### Reference: List utilities

#### `AV.AnyMatch`

AnyMatch is List::Util's any: whether f is true for at least one element.

```go
lines := scalar.Array("Office in Leeds", "Remote working available")
fmt.Println(lines.AnyMatch(isRemote)) // any { /remote/i } @lines
fmt.Println(scalar.Array().AnyMatch(isRemote))
```

```
true
false
```

#### `AV.AllMatch`

AllMatch is List::Util's all: whether f is true for every element.

```go
n := scalar.Array(2, 4, 6)
fmt.Println(n.AllMatch(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })) // all { $_ % 2 == 0 }
fmt.Println(scalar.Array().AllMatch(isRemote))                                  // true for nothing
```

```
true
true
```

#### `AV.NoneMatch`

NoneMatch is List::Util's none: whether f is false for every element.

```go
fmt.Println(scalar.Array("Leeds", "York").NoneMatch(isRemote))
```

```
true
```

#### `AV.FirstMatch`

FirstMatch is List::Util's first: the first element for which f is true, or undef if there's none.

```go
lines := scalar.Array("Leeds office", "Salary: £45k", "Salary: £50k")
salary := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("£") >= 0 })
fmt.Println(salary) // first { index($_, "£") >= 0 } @lines
fmt.Println(lines.FirstMatch(isRemote).Defined())
```

```
Salary: £45k
false
```

#### `AV.MaxNum`

MaxNum is List::Util's max: the element with the largest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MaxNum()) // max: as numbers
```

```
10
```

#### `AV.MinNum`

MinNum is List::Util's min: the element with the smallest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MinNum())
```

```
2.5
```

#### `AV.MaxStr`

MaxStr is List::Util's maxstr: the element that sorts last as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", "9", "100").MaxStr()) // maxstr: as strings
```

```
9
```

#### `AV.MinStr`

MinStr is List::Util's minstr: the element that sorts first as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("b", "a", "c").MinStr())
```

```
a
```

#### `AV.UniqueStr`

UniqueStr is List::Util's uniqstr: a new array of copies of the elements, keeping only the first of each distinct string, in order.

```go
fmt.Println(scalar.Array("Go", "go", "Go", "Perl").UniqueStr().Join(",")) // first of each, in order
skills := scalar.Array("Go", "go", "Perl")
fmt.Println(skills.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr().Join(",")) // ignoring case
```

```
Go,go,Perl
go,perl
```

#### `AV.UniqueNum`

UniqueNum is List::Util's uniqnum: a new array of copies of the elements, keeping only the first of each distinct number, in order.

```go
fmt.Println(scalar.Array(1, "1.0", "01", 2).UniqueNum().Join(",")) // the same number, however written
```

```
1,2
```

#### `AV.Sum`

Sum is List::Util's sum: the numeric sum of the elements, or undef for an empty array.

```go
fmt.Println(scalar.Array(1, "2", 3.5).Sum()) // sum @a
fmt.Println(scalar.Array().Sum().Defined())  // undef for nothing
```

```
6.5
false
```

#### `AV.Sum0`

Sum0 is List::Util's sum0: Sum, but 0 for an empty array.

```go
fmt.Println(scalar.Array().Sum0()) // sum0: 0 for nothing
```

```
0
```

#### `AV.Product`

Product is List::Util's product: the numeric product of the elements, or 1 for an empty array, as in List::Util.

```go
fmt.Println(scalar.Array(2, 3, 4).Product())
fmt.Println(scalar.Array().Product()) // 1 for nothing
```

```
24
1
```

#### `AV.Reduce`

Reduce is List::Util's reduce { ...

```go
words := scalar.Array("go", "perl", "c")
longest := words.Reduce(func(a, b *scalar.SV) any { // reduce { length $b > length $a ? $b : $a } @words
	if b.Length() > a.Length() {
		return b
	}
	return a
})
joined := words.Reduce(func(a, b *scalar.SV) any { return a.Str() + "+" + b.Str() })
fmt.Println(longest, joined)
```

```
perl go+perl+c
```

#### `AV.SortByStr`

SortByStr is List::UtilsBy's sort_by: a sorted copy of a, ordered by the string keys f gives, as sort { $a cmp $b } would order them.

```go
titles := scalar.Array("data engineer", "Go Developer", "Analyst")
byTitle := titles.SortByStr(func(t *scalar.SV) any { return t.Fc() }) // sort_by { fc } @titles
fmt.Println(byTitle.Join(", "))
```

```
Analyst, data engineer, Go Developer
```

#### `AV.SortByNum`

SortByNum is List::UtilsBy's nsort_by: as SortByStr, but comparing the keys as numbers.

```go
jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "salary": 60000}, {"title": "SQL", "salary": 30000}]`))
bySalary := jobs.AV().SortByNum(func(j *scalar.SV) any { return j.At("salary") }) // nsort_by { $_->{salary} }
fmt.Println(bySalary.Get(0).At("title"), bySalary.Get(1).At("title"))
```

```
SQL Go
```

#### `AV.RevSortByStr`

RevSortByStr is List::UtilsBy's rev_sort_by: SortByStr, in descending order.

```go
words := scalar.Array("bb", "a", "dd", "e")
fmt.Println(words.RevSortByStr(func(w *scalar.SV) any { return w.Length() }).Join(" ")) // equal keys keep their order
```

```
bb dd a e
```

#### `AV.RevSortByNum`

RevSortByNum is List::UtilsBy's rev_nsort_by: SortByNum, in descending order, with elements of equal keys keeping their order.

```go
jobs, _ := scalar.FromJSON([]byte(`[{"title": "SQL", "salary": 30000}, {"title": "Go", "salary": 60000}, {"title": "Perl", "salary": 60000}]`))
best := jobs.AV().RevSortByNum(func(j *scalar.SV) any { return j.At("salary") }) // the best paid first
fmt.Println(best.Get(0).At("title"), best.Get(1).At("title"), best.Get(2).At("title"))
```

```
Go Perl SQL
```

#### `AV.MaxBy`

MaxBy is List::UtilsBy's max_by: the element with the largest numeric key, the first of them if several share it, or undef for an empty array.

```go
jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "salary": 60000}, {"title": "Perl", "salary": 60000}, {"title": "SQL", "salary": 30000}]`))
best := jobs.AV().MaxBy(func(j *scalar.SV) any { return j.At("salary") }) // max_by { $_->{salary} }
fmt.Println(best.At("title"))                                             // the first of the best paid
```

```
Go
```

#### `AV.MinBy`

MinBy is List::UtilsBy's min_by: the element with the smallest numeric key; see MaxBy.

```go
words := scalar.Array("ccc", "a", "bb", "e")
fmt.Println(words.MinBy(func(w *scalar.SV) any { return w.Length() })) // min_by { length } @words
```

```
a
```

#### `AV.UniqueBy`

UniqueBy is List::UtilsBy's uniq_by: a new array of copies of the elements, keeping only the first with each key, in order.

```go
jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "level": "Senior"}, {"title": "SQL", "level": "Junior"}, {"title": "Perl", "level": "Senior"}]`))
oneEach := jobs.AV().UniqueBy(func(j *scalar.SV) any { return j.At("level") }) // the first job at each level
fmt.Println(oneEach.Length(), oneEach.Get(0).At("title"), oneEach.Get(1).At("title"))
```

```
2 Go SQL
```

#### `AV.PartitionBy`

PartitionBy is List::UtilsBy's partition_by: a hash from each key f gives to an array of copies of the elements with that key, in order.

```go
words := scalar.Array("bb", "a", "ccc", "dd", "e")
byLength := words.PartitionBy(func(w *scalar.SV) any { return w.Length() }) // partition_by { length } @words
fmt.Println(byLength)
```

```
{1 => ["a","e"],2 => ["bb","dd"],3 => ["ccc"]}
```

#### `AV.CountBy`

CountBy is List::UtilsBy's count_by: how many elements have each key f gives, as a Bag, so MostCommon lists the commonest first.

```go
places := scalar.Array("Leeds", "London", "leeds", "LEEDS")
where := places.CountBy(func(p *scalar.SV) any { return p.Fc() }) // a Bag: count_by's counts, with MostCommon
fmt.Println(where.MostCommon(1).Get(0), where.Count("leeds"), where.Counts())
```

```
leeds 3 {leeds => 3,london => 1}
```

#### `AV.ExtractBy`

ExtractBy is List::UtilsBy's extract_by: it removes the elements for which f returns true from a, and returns them, in order, as a new array.

```go
words := scalar.Array("bb", "a", "ccc", "e")
long := words.ExtractBy(func(w *scalar.SV) bool { return w.Length() > 1 }) // moved out of words
fmt.Println(long.Join(" "), "|", words.Join(" "))
```

```
bb ccc | a e
```

### Reference: Sets and bags

#### `NewSet`

NewSet is Set::Scalar->new(...): a set of vals, flattened as a list is (see `AV.Push`), so NewSet(array) and NewSet(goSlice) take their elements.

```go
s := scalar.NewSet("Go", "Perl", "Go") // Set::Scalar->new(...): each member once
fmt.Println(s, s.Length())
same := scalar.NewSet(1, "1", 1.0) // compared as strings: all "1"
fmt.Println(same.Length())
fromList := scalar.NewSet(scalar.Array("a", "b"), []string{"c"}) // lists flatten
fmt.Println(fromList)
```

```
(Go Perl) 2
1
(a b c)
```

#### `Set.Insert`

Insert is Set::Scalar's insert: it adds vals, flattened as by NewSet, and returns s.

```go
var s scalar.Set // the zero value is an empty set
s.Insert("Go").Insert("Perl", "Go")
fmt.Println(&s)
```

```
(Go Perl)
```

#### `Set.Delete`

Delete is Set::Scalar's delete: it removes vals, flattened as by NewSet, and returns s.

```go
s := scalar.NewSet("a", "b", "c")
s.Delete("b", "not a member") // non-members are ignored
fmt.Println(s)
```

```
(a c)
```

#### `Set.Has`

Has is Set::Scalar's has: whether v (compared as a string) is a member.

```go
s := scalar.NewSet("Go", 42)
fmt.Println(s.Has("Go"), s.Has("go"), s.Has("42")) // "42" is the same member as 42
```

```
true false true
```

#### `Set.Length`

Length is Set::Scalar's size: the number of members.

```go
fmt.Println(scalar.NewSet("a", "b", "a").Length())
```

```
2
```

#### `Set.Members`

Members is Set::Scalar's members: a new array of copies of the members, in the order they were first inserted.

```go
s := scalar.NewSet("c", "a", "b")
fmt.Println(s.Members().Join(","))           // in insertion order
fmt.Println(s.Members().Sort(nil).Join(",")) // sorted, with the array tools
```

```
c,a,b
a,b,c
```

#### `Set.All`

All iterates over copies of the members, in the order they were first inserted.

```go
for m := range scalar.NewSet("x", "y").All() {
	fmt.Println(m)
}
```

```
x
y
```

#### `Set.Copy`

Copy returns a new set with the same members.

```go
a := scalar.NewSet("a")
b := a.Copy()
b.Insert("b")
fmt.Println(a, b)
```

```
(a) (a b)
```

#### `Set.Union`

Union is Set::Scalar's union: a new set of every member of s or any of others, in order of first appearance.

```go
a, b := scalar.NewSet("a", "b"), scalar.NewSet("b", "c")
fmt.Println(a.Union(b))                     // $a->union($b)
fmt.Println(a.Union(b, scalar.NewSet("d"))) // several at once
```

```
(a b c)
(a b c d)
```

#### `Set.Intersection`

Intersection is Set::Scalar's intersection: a new set of the members of s that are also in every one of others.

```go
mine := scalar.NewSet("Go", "Perl", "SQL")
wanted := scalar.NewSet("Go", "SQL", "Rust")
fmt.Println(mine.Intersection(wanted)) // the skills I have that are wanted
```

```
(Go SQL)
```

#### `Set.Difference`

Difference is Set::Scalar's difference: a new set of the members of s that are in none of others.

```go
wanted := scalar.NewSet("Go", "SQL", "Rust")
mine := scalar.NewSet("Go", "Perl", "SQL")
fmt.Println(wanted.Difference(mine)) // what's wanted that I lack
```

```
(Rust)
```

#### `Set.SymmetricDifference`

SymmetricDifference is Set::Scalar's symmetric_difference: a new set of the members in exactly one of s and t.

```go
a, b := scalar.NewSet("a", "b", "c"), scalar.NewSet("b", "c", "d")
fmt.Println(a.SymmetricDifference(b)) // in one or the other, not both
```

```
(a d)
```

#### `Set.IsSubset`

IsSubset is Set::Scalar's is_subset: whether every member of s is in t.

```go
a := scalar.NewSet("Go")
fmt.Println(a.IsSubset(scalar.NewSet("Go", "Perl")), a.IsSubset(a), scalar.NewSet().IsSubset(a))
```

```
true true true
```

#### `Set.IsProperSubset`

IsProperSubset is Set::Scalar's is_proper_subset: a subset of t that isn't equal to it.

```go
a := scalar.NewSet("Go")
fmt.Println(a.IsProperSubset(scalar.NewSet("Go", "Perl")), a.IsProperSubset(a))
```

```
true false
```

#### `Set.IsSuperset`

IsSuperset is Set::Scalar's is_superset: whether s has every member of t.

```go
fmt.Println(scalar.NewSet("Go", "Perl").IsSuperset(scalar.NewSet("Perl")))
```

```
true
```

#### `Set.IsEqual`

IsEqual is Set::Scalar's is_equal: whether s and t have the same members, in any order.

```go
fmt.Println(scalar.NewSet("a", "b").IsEqual(scalar.NewSet("b", "a"))) // order doesn't matter
```

```
true
```

#### `Set.IsDisjoint`

IsDisjoint is Set::Scalar's is_disjoint: whether s and t have no members in common.

```go
fmt.Println(scalar.NewSet("a").IsDisjoint(scalar.NewSet("b")))
fmt.Println(scalar.NewSet().IsDisjoint(scalar.NewSet())) // true, unlike Set::Scalar
```

```
true
true
```

#### `Set.String`

String is the set as Set::Scalar prints it: its members sorted, between brackets, as in "(a b c)".

```go
fmt.Println(scalar.NewSet("c", "a", "b").String()) // sorted, as Set::Scalar prints a set
```

```
(a b c)
```

#### `Set.MarshalJSON`

MarshalJSON encodes the set as an array of its members, in the order they were first inserted.

```go
out, _ := json.Marshal(scalar.NewSet("c", "a", "b")) // an array, in insertion order
fmt.Println(string(out))
```

```
["c","a","b"]
```

#### `NewBag`

NewBag returns a bag of vals, flattened as a list is (see `AV.Push`), each counting once, so NewBag("Go", "Go", "Perl") has two Go and one Perl.

```go
skills := scalar.NewBag("Go", "Perl", "Go") // each value counts once
fmt.Println(skills)
apples := scalar.NewBag().InsertN("apples", 3) // Set::Bag->new(apples => 3)
fmt.Println(apples)
```

```
(Go => 2, Perl => 1)
(apples => 3)
```

#### `Bag.Insert`

Insert adds one of each of vals, flattened as by NewBag, and returns b.

```go
var counts scalar.Bag // the zero value is an empty bag
for _, posting := range [][]string{{"Go", "SQL"}, {"Go", "Perl"}, {"Go"}} {
	counts.Insert(posting) // count the skills across postings
}
fmt.Println(&counts)
```

```
(Go => 3, Perl => 1, SQL => 1)
```

#### `Bag.InsertN`

InsertN is Set::Bag's insert(v => n): it adds n of v and returns b.

```go
b := scalar.NewBag()
b.InsertN("apples", 3).InsertN("apples", 2) // $b->insert(apples => 3) and so on
fmt.Println(b)
```

```
(apples => 5)
```

#### `Bag.Delete`

Delete removes one of each of vals, flattened as by NewBag, and returns b.

```go
b := scalar.NewBag("a", "a", "b")
b.Delete("a", "b") // one of each
fmt.Println(b)
```

```
(a => 1)
```

#### `Bag.DeleteN`

DeleteN is Set::Bag's delete(v => n): it removes n of v, or all of it if there are fewer, and returns b.

```go
b := scalar.NewBag().InsertN("apples", 3)
b.DeleteN("apples", 2)
fmt.Println(b)
b.DeleteN("apples", 10) // more than there is: all of it, never below zero
fmt.Println(b)
```

```
(apples => 1)
()
```

#### `Bag.Count`

Count is Set::Bag's grab(v): how many of v there are, 0 if none.

```go
b := scalar.NewBag("Go", "Go", "Perl")
fmt.Println(b.Count("Go"), b.Count("Rust")) // $b->grab("Go")
```

```
2 0
```

#### `Bag.Length`

Length is the number of distinct members.

```go
fmt.Println(scalar.NewBag("a", "a", "b").Length()) // distinct members
```

```
2
```

#### `Bag.Total`

Total is the number of members counting repeats: the sum of the counts.

```go
fmt.Println(scalar.NewBag("a", "a", "b").Total()) // counting repeats
```

```
3
```

#### `Bag.Members`

Members is Set::Bag's elements: a new array of copies of the distinct members, in the order they were first inserted.

```go
fmt.Println(scalar.NewBag("b", "a", "b").Members().Join(",")) // distinct, in insertion order
```

```
b,a
```

#### `Bag.Counts`

Counts returns the bag as a new hash from each member (as a string) to its count, for the rest of the package's tools.

```go
counts := scalar.NewBag("Go", "Go", "Perl").Counts() // a hash, for the hash tools
fmt.Println(counts.Get("Go"), counts.Get("Perl"))
```

```
2 1
```

#### `Bag.MostCommon`

MostCommon returns copies of the members with the highest counts, most common first, and the first inserted first among equal counts: the top n, or every member if n is left out or there are fewer.

```go
skills := scalar.NewBag("SQL", "Go", "Perl", "Go", "SQL", "Go")
fmt.Println(skills.MostCommon(2).Join(",")) // the top 2
fmt.Println(skills.MostCommon().Join(","))  // all, most common first
```

```
Go,SQL
Go,SQL,Perl
```

#### `Bag.All`

All iterates over copies of the distinct members with their counts, in the order they were first inserted.

```go
for m, n := range scalar.NewBag("x", "y", "x").All() {
	fmt.Println(m, n)
}
```

```
x 2
y 1
```

#### `Bag.Copy`

Copy returns a new bag with the same members and counts.

```go
a := scalar.NewBag("a")
b := a.Copy()
b.Insert("a")
fmt.Println(a, b)
```

```
(a => 1) (a => 2)
```

#### `Bag.Sum`

Sum is Set::Bag's sum: a new bag in which each member's count is its counts in b and others added together.

```go
monday := scalar.NewBag().InsertN("Go", 3).InsertN("SQL", 1)
tuesday := scalar.NewBag().InsertN("Go", 1).InsertN("Perl", 2)
fmt.Println(monday.Sum(tuesday)) // counts added: $monday->sum($tuesday)
```

```
(Go => 4, Perl => 2, SQL => 1)
```

#### `Bag.Union`

Union is Set::Bag's union: a new bag in which each member's count is the largest of its counts in b and others.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
fmt.Println(x.Union(y)) // the larger count of each
```

```
(a => 3, b => 2)
```

#### `Bag.Intersection`

Intersection is Set::Bag's intersection: a new bag of the members in b and every one of others, each with the smallest of its counts.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("c", 2)
fmt.Println(x.Intersection(y)) // the smaller count of each, in both
```

```
(a => 1)
```

#### `Bag.Difference`

Difference is Set::Bag's difference: a new bag in which each member's count is its count in b less its counts in others, leaving out any that reach zero, as Set::Bag does with over_delete.

```go
x := scalar.NewBag().InsertN("a", 3).InsertN("b", 1)
y := scalar.NewBag().InsertN("a", 1).InsertN("b", 2)
fmt.Println(x.Difference(y)) // counts taken away, never below zero
```

```
(a => 2)
```

#### `Bag.String`

String is the bag as Set::Bag prints it: members sorted, each with its count, as in "(apples => 3, pears => 1)".

```go
fmt.Println(scalar.NewBag("pears", "apples", "apples").String()) // sorted, as Set::Bag prints a bag
```

```
(apples => 2, pears => 1)
```

#### `Bag.MarshalJSON`

MarshalJSON encodes the bag as an object from each member to its count, in the order the members were first inserted.

```go
out, _ := json.Marshal(scalar.NewBag("pears", "apples", "apples")) // in insertion order
fmt.Println(string(out))
```

```
{"pears":1,"apples":2}
```

### Reference: Regular expressions

#### `SV.M`

M is $s =~ /re/ with captures.

```go
s := scalar.Value("Salary: £45,000 - £55,000")
if s.M(`(?i)salary`) != nil { // if ($s =~ /salary/i)
	fmt.Println("matched")
}
m := s.M(`£([\d,]+)\s*-\s*£([\d,]+)`)
fmt.Println(m.Get(0)) // $&
fmt.Println(m.Get(1), m.Get(2))
all := s.M(`£([\d,]+)`, "g")                // @all = all the matches: one array of captures each
fmt.Println(all.Length(), all.At("[1][1]")) // $all[1][1]: the second match's $1
fmt.Println(s.M(`remote`).Get(1).Defined()) // no match: nil, still safe to read
```

```
matched
£45,000 - £55,000
45,000 55,000
2 55,000
false
```

#### `SV.Named`

Named is $s =~ /re/ followed by reading %+.

```go
d := scalar.Value("Closing 2026-11-30").Named(`(?<year>\d{4})-(?<month>\d\d)-(?<day>\d\d)`)
fmt.Println(d.Get("year"), d.Get("month"), d.Get("day")) // $+{year} and so on
```

```
2026 11 30
```

#### `SV.S`

S is Perl's s///.

```go
s := scalar.Value("a  b   c")
n := s.S(`\s+`, " ", "g") // $s =~ s/\s+/ /g
fmt.Println(s, n)
name := scalar.Value("Lovelace, Ada")
name.S(`(\w+), (\w+)`, "$2 $1") // $1 and $2 in the replacement
fmt.Println(name)
clean := scalar.Value("x1y2").S(`\d`, "", "gr") // s///r: a new string; the original is unchanged
fmt.Println(clean)
doubled := scalar.Value("3 apples, 4 pears")
doubled.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g") // s///e
fmt.Println(doubled)
```

```
a b c 2
Ada Lovelace
xy
6 apples, 8 pears
```

#### `SV.Split`

Split is Perl's split /re/, $s.

```go
fmt.Println(scalar.Value("a,b,,c,,").Split(`,`).Length())        // trailing empty fields dropped
fmt.Println(scalar.Value("a1b2c").Split(`(\d)`).Join("|"))       // captures included
fmt.Println(scalar.Value("a,b,c,d").Split(`,`, 2).Join("|"))     // a limit
fmt.Println(scalar.Value("  some  words ").Split(" ").Join("|")) // awk mode
```

```
4
a|1|b|2|c
a|b,c,d
some|words
```

#### `SV.Tr`

Tr is Perl's tr/// (y///).

```go
s := scalar.Value("hello")
s.Tr("a-z", "A-Z") // $s =~ tr/a-z/A-Z/
fmt.Println(s)
fmt.Println(scalar.Value("hello world").Tr("a-z", ""))  // counts: tr/a-z// returns how many
fmt.Println(scalar.Value("aabbcc").Tr("a-z", "", "sr")) // squeeze, returning a new string
```

```
HELLO
10
abc
```

#### `Compile`

Compile compiles pattern with flags, any of "i", "m", "s" and "x", or returns an error if the pattern is invalid.

```go
re, err := scalar.Compile(`£([\d,]+`) // a pattern from outside the program: check it
fmt.Println(re == nil, err != nil)
```

```
true true
```

#### `MustCompile`

MustCompile is Compile, panicking if the pattern is invalid.

```go
salary := scalar.MustCompile(`£([\d,]+)`) // my $salary = qr/£([\d,]+)/
s := scalar.Value("£45,000 to £55,000")
fmt.Println(s.M(salary, "g").Length()) // use it anywhere a pattern string goes
fmt.Println(s.S(salary, "£X", "gr"))
insensitive := scalar.MustCompile(`remote`, "i") // flags are fixed when compiling
fmt.Println(scalar.Value("REMOTE").M(insensitive) != nil)
```

```
2
£X to £X
true
```

#### `Regexp.String`

String returns the pattern as written.

```go
fmt.Println(scalar.MustCompile(`\d+`, "i"))
```

```
\d+
```

#### `SV.Matches`

Matches is Perl's while ($s =~ /re/g) { ...

```go
s := scalar.Value("Salary: £45,000 - £55,000")
for m := range s.Matches(`£([\d,]+)`) { // while ($s =~ /£([\d,]+)/g)
	fmt.Println(m.Get(1)) // $1
}
for m := range scalar.Value("a=1, b=2, c=3").Matches(`(\w)=(\d)`) {
	if m.Get(1).Str() == "b" {
		fmt.Println("found b:", m.Get(2))
		break // stops the search
	}
}
```

```
45,000
55,000
found b: 2
```

### Reference: Pack and unpack

#### `Pack`

Pack is Perl's pack for templates of H, h, B, b, f, d and F: it packs vals (flattened as a list is, see `AV.Push`) into bytes, returned as a byte string (see `SV.Bytes`).

```go
fmt.Println(scalar.Pack("H*", "4a6f"))     // hex digits to bytes: "Jo"
fmt.Println(scalar.Pack("B8", "01001010")) // bits to a byte: "J"

f := scalar.Pack("f>", 1.5) // a big-endian float32: 4 bytes
fmt.Println(len(f.Str()), hex.EncodeToString([]byte(f.Str())))
d := scalar.Pack("d<*", 1, 2.5) // two little-endian float64s
fmt.Println(len(d.Str()))
```

```
Jo
J
4 3fc00000
16
```

#### `Unpack`

Unpack is Perl's unpack for templates of H, h, B, b, f, d and F: it reads data, a string of bytes (a scalar, a Go string or a []byte), and returns what the template describes, as an array.

```go
fmt.Println(scalar.Unpack("H*", "Jo").Get(0))                       // bytes to hex
fmt.Println(scalar.Unpack("B8", "J").Get(0))                        // a byte to bits
fmt.Println(scalar.Unpack("H*", scalar.Pack("d>", 3.14159)).Get(0)) // a float64's bytes, in hex
fmt.Println(scalar.Unpack("B32", scalar.Pack("f>", -2)).Get(0))     // a float32's bits: sign, exponent, fraction
fmt.Println(scalar.Unpack("f<*", scalar.Pack("f<*", 1.5, 0.1)).Join(", "))
```

```
4a6f
01001010
400921f9f01b866e
11000000000000000000000000000000
1.5, 0.100000001490116
```

### Reference: JSON and JSONPath

#### `FromJSON`

FromJSON decodes data into a scalar.

```go
doc, err := scalar.FromJSON([]byte(`{"id": 9007199254740993, "pay": 45000.5, "skills": ["Go"], "bonus": null}`))
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(doc.Path("id")) // whole numbers stay exact
fmt.Println(doc.Path("pay"), doc.Path("skills").Ref())
fmt.Println(doc.HV().Exists("bonus"), doc.Path("bonus").Defined()) // null is undef
```

```
9007199254740993
45000.5 ARRAY
true false
```

#### `SV.MarshalJSON`

MarshalJSON implements `json.Marshaler`.

```go
h := scalar.Hash("b", 1, "a", scalar.Value(scalar.Array("x", nil)))
out, _ := json.Marshal(scalar.Value(h)) // keys sorted, undef as null
fmt.Println(string(out))
h.Set("self", scalar.Value(h)) // a loop can't be JSON: an error, not a crash
_, err := json.Marshal(scalar.Value(h))
fmt.Println(err != nil)
```

```
{"a":["x",null],"b":1}
true
```

#### `AV.MarshalJSON`

MarshalJSON implements `json.Marshaler`, as `SV.MarshalJSON` does.

```go
out, _ := json.Marshal(scalar.Array(1, "two", 3.5))
fmt.Println(string(out))
```

```
[1,"two",3.5]
```

#### `HV.MarshalJSON`

MarshalJSON implements `json.Marshaler`, as `SV.MarshalJSON` does.

```go
out, _ := json.Marshal(scalar.Hash("z", 1, "a", 2))
fmt.Println(string(out))
```

```
{"a":2,"z":1}
```

#### `SV.UnmarshalJSON`

UnmarshalJSON implements `json.Unmarshaler`.

```go
var record struct {
	ID   int        `json:"id"`
	Data *scalar.SV `json:"data"` // any shape at all
}
_ = json.Unmarshal([]byte(`{"id": 1, "data": {"tags": ["a", "b"]}}`), &record)
fmt.Println(record.ID, record.Data.Path("tags").AV().Join(","))
```

```
1 a,b
```

#### `SV.Find`

Find runs a JSONPath query (RFC 9535) on s and returns every match, or nil if nothing matched.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"store": {"book": [
		{"title": "Go", "price": 30},
		{"title": "Perl", "price": 45}
	]}
}`))
fmt.Println(doc.Find("$.store.book[*].title").Join(","))  // every title
fmt.Println(doc.Find("$..price").Join(","))               // every price, at any depth
fmt.Println(doc.Find("$.store.book[-1].title").Join(",")) // the last book's title
fmt.Println(doc.Find("$.store.magazine") == nil)          // nothing matched
```

```
Go,Perl
30,45
Perl
true
```

#### `SV.Find_filter`

```go
doc, _ := scalar.FromJSON([]byte(`{"jobs": [
	{"title": "Nurse", "salary": {"min": 32000}, "skills": ["care"]},
	{"title": "Developer", "salary": {"min": 55000, "max": 70000}, "skills": ["Go", "SQL"]},
	{"title": "Café manager", "skills": []}
]}`))
fmt.Println(doc.Find("$.jobs[?@.salary.min >= 40000].title").Join(","))  // a comparison
fmt.Println(doc.Find("$.jobs[?!@.salary].title").Join(","))              // no salary at all
fmt.Println(doc.Find("$.jobs[?count(@.skills[*]) > 1].title").Join(",")) // a function
fmt.Println(doc.Find("$.jobs[?match(@.title, 'N.*')].title").Join(","))  // I-Regexp, whole string
fmt.Println(doc.Find("$..[?@ == 'SQL']").Join(","))                      // at any depth
```

```
Developer
Café manager
Developer
Nurse
SQL
```

#### `QuoteJSONPath`

QuoteJSONPath returns s as a JSONPath string literal, quoted and escaped as RFC 9535 requires, to put a value or a name from Go into a query safely, as `regexp.QuoteMeta` does for a pattern:

```go
doc, _ := scalar.FromJSON([]byte(`{"sections": [
	{"kind": "summary", "lines": ["We're hiring."]},
	{"kind": "it's benefits", "lines": ["Pension", "Gym"]}
]}`))
kind := "it's benefits" // from Go, with a quote in it
fmt.Println(scalar.QuoteJSONPath(kind))
fmt.Println(doc.Find("$.sections[?@.kind == " + scalar.QuoteJSONPath(kind) + "].lines[*]").Join(", "))
```

```
'it\'s benefits'
Pension, Gym
```

### Reference: YAML

#### `FromYAML`

FromYAML reads a YAML document and returns it, as FromJSON reads JSON: mappings become hashes, sequences arrays, and plain values scalars.

```go
values, err := scalar.FromYAML([]byte(`
defaults: &defaults
  replicas: 2
  image: {repository: web, tag: "1.0"}
production:
  <<: *defaults        # a merge key: production gets the defaults
  replicas: 5
country: no            # YAML 1.2: the string "no", not false
`))
if err != nil {
	panic(err)
}
fmt.Println(values.At("production.replicas"), values.At("production.image.tag"))
fmt.Println(values.At("production.image").HV() == values.At("defaults.image").HV()) // one hash, shared
fmt.Println(values.At("country"))
```

```
5 1.0
true
no
```

#### `FromYAMLAll`

FromYAMLAll reads a stream of YAML documents, separated by ---, as in a Kubernetes manifest or Helm's output, and returns them as an array, in order.

```go
manifest := []byte(`
apiVersion: v1
kind: Service
metadata: {name: web}
---
apiVersion: apps/v1
kind: Deployment
metadata: {name: web}
spec: {replicas: 3}
---
`)
docs, err := scalar.FromYAMLAll(manifest) // several documents, as kubectl takes them
if err != nil {
	panic(err)
}
for _, d := range docs.All() {
	fmt.Println(d.At("kind"), d.At("metadata.name"), d.At("spec.replicas").Or("-"))
}
```

```
Service web -
Deployment web 3
```

#### `SV.ToYAML`

ToYAML writes s as a YAML document, with two-space indentation, as Kubernetes and Helm use:

```go
cfg := scalar.Hash("name", "web", "replicas", 3, "country", "no", "version", "1.0",
	"ports", scalar.Value([]int{80, 443}))
out, _ := scalar.Value(cfg).ToYAML() // keys sorted; strings that any reader could misread are quoted
fmt.Print(string(out))
```

```
country: "no"
name: web
ports:
  - 80
  - 443
replicas: 3
version: "1.0"
```

#### `AV.ToYAML`

ToYAML writes a as a YAML sequence; see `SV.ToYAML`.

```go
out, _ := scalar.Array("go", 1.0, nil, true).ToYAML()
fmt.Print(string(out))
```

```
- go
- 1.0
- null
- true
```

#### `HV.ToYAML`

ToYAML writes h as a YAML mapping; see `SV.ToYAML`.

```go
shared := scalar.Value(scalar.Hash("cpu", "500m"))
h := scalar.Hash("requests", shared, "limits", shared) // the same hash twice
out, _ := h.ToYAML()
fmt.Print(string(out)) // an anchor, then an alias
```

```
limits: &1
  cpu: 500m
requests: *1
```

#### `SV.UnmarshalYAML`

UnmarshalYAML lets a scalar be a field of a struct read with the YAML package's Unmarshal, for a part of a document whose shape varies; it reads as FromYAML does.

```go
var cfg struct {
	Name  string     `yaml:"name"`
	Extra *scalar.SV `yaml:"extra"` // a part whose shape varies
}
err := yaml.Unmarshal([]byte("name: web\nextra: {ports: [80, 443]}\n"), &cfg)
fmt.Println(err, cfg.Name, cfg.Extra.At("ports[1]"))
```

```
<nil> web 443
```

#### `SV.MarshalYAML`

MarshalYAML lets a scalar be a field of a struct written with the YAML package's Marshal; it writes as ToYAML does.

```go
out, _ := yaml.Marshal(struct {
	Name  string     `yaml:"name"`
	Extra *scalar.SV `yaml:"extra"`
}{"web", scalar.Value(map[string]any{"ports": []int{80}})})
fmt.Print(string(out))
```

```
name: web
extra:
    ports:
        - 80
```

#### `AV.MarshalYAML`

MarshalYAML lets an array be written with the YAML package's Marshal.

```go
out, _ := yaml.Marshal(map[string]any{"ports": scalar.Array(80, 443)})
fmt.Print(string(out))
```

```
ports:
    - 80
    - 443
```

#### `HV.MarshalYAML`

MarshalYAML lets a hash be written with the YAML package's Marshal.

```go
out, _ := yaml.Marshal(map[string]any{"labels": scalar.Hash("app", "web")})
fmt.Print(string(out))
```

```
labels:
    app: web
```

### Reference: Into Go values

#### `To`

To converts v into a Go value of type T, the type in brackets: the way out of scalar into ordinary Go, once you know the shape you want.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"title": "Go Developer",
	"skills": ["Go", "SQL"],
	"salary": {"min": "45000", "max": 55000}
}`))
type Posting struct {
	Title  string   `json:"title"`
	Skills []string `json:"skills"`
	Salary struct {
		Min int `json:"min"` // "45000" fills it, as Perl would numify it
		Max int `json:"max"`
	} `json:"salary"`
}
p, err := scalar.To[Posting](doc) // the type you want, in brackets
if err != nil {
	panic(err)
}
fmt.Println(p.Title, p.Skills, p.Salary.Min, p.Salary.Max)
```

```
Go Developer [Go SQL] 45000 55000
```

#### `To.slice`

```go
doc, _ := scalar.FromJSON([]byte(`{"skills": ["Go", "SQL"], "years": [3, "5"]}`))
skills, _ := scalar.To[[]string](doc.At("skills")) // from a scalar holding an array
years, _ := scalar.To[[]int](doc.At("years").AV()) // or from the array itself
fmt.Printf("%q %v\n", skills, years)
```

```
["Go" "SQL"] [3 5]
```

#### `To.number`

```go
salary, err := scalar.To[int](scalar.Value(" 55000 ")) // a numeric string, as Perl reads it
fmt.Println(salary, err)
rate, err := scalar.To[float64](scalar.Value("37.5"))
fmt.Println(rate, err)
```

```
55000 <nil>
37.5 <nil>
```

#### `To.map`

```go
doc, _ := scalar.FromJSON([]byte(`{"go": 3, "perl": "2"}`))
counts, _ := scalar.To[map[string]int](doc) // a hash fills a map
fmt.Println(counts["go"], counts["perl"])
```

```
3 2
```

#### `To.set`

```go
langs := scalar.NewSet("Go", "Perl", 5)
members, _ := scalar.To[[]string](langs) // a set gives its members, in insertion order
fmt.Printf("%q\n", members)
```

```
["Go" "Perl" "5"]
```

#### `To.bag`

```go
skills := scalar.NewBag("Go", "Go", "Perl")
counts, _ := scalar.To[map[string]int](skills)    // a bag gives its counts
names, _ := scalar.To[[]string](skills.Members()) // and Members its distinct members
fmt.Println(counts["Go"], counts["Perl"], names)
```

```
2 1 [Go Perl]
```

#### `To.any`

```go
doc, _ := scalar.FromJSON([]byte(`{"tags": ["go", 1]}`))
plain, _ := scalar.To[any](doc) // the plain Go form, as Any gives it
fmt.Printf("%v\n", plain)
```

```
map[tags:[go 1]]
```

#### `To.missing`

```go
doc, _ := scalar.FromJSON([]byte(`{"bonus": null}`))
bonus, err := scalar.To[*int](doc.At("bonus")) // null, or missing: the zero value, here nil
fmt.Println(bonus, err)
perks, err := scalar.To[[]string](doc.At("perks"))
fmt.Println(perks == nil, err)
```

```
<nil> <nil>
true <nil>
```

#### `To.optional`

An optional value is a pointer: nil if the value is undef, missing or null, and otherwise a pointer to it, converted and checked. There's no StrPtr or NumPtr method, as To\[\*T] does it for any type, and refuses a value that doesn't fit, where Num would quietly make "abc" 0.

```go
doc, _ := scalar.FromJSON([]byte(`{"min": 45000, "max": null, "period": "Year", "bonus": "competitive"}`))

min, _ := scalar.To[*float64](doc.At("min"))
max, _ := scalar.To[*float64](doc.At("max"))      // null: nil
none, _ := scalar.To[*string](doc.At("currency")) // missing: nil
fmt.Println(*min, max, none)

_, err := scalar.To[*float64](doc.At("bonus")) // "competitive" isn't a number
fmt.Println(err)

// The same, for a struct's optional fields, as the model's metadata has.
type Pay struct {
	Min, Max *float64
	Period   *string
}
p, _ := scalar.To[Pay](doc)
fmt.Println(*p.Min, p.Max, *p.Period)
```

```
45000 <nil> <nil>
scalar: Decode: want float64, got "competitive"
45000 <nil> Year
```

#### `To.error`

```go
doc, _ := scalar.FromJSON([]byte(`{"salary": {"max": "competitive"}}`))
type Posting struct {
	Salary struct {
		Max int `json:"max"`
	} `json:"salary"`
}
p, err := scalar.To[Posting](doc)
fmt.Println(err)
fmt.Println(p.Salary.Max) // on an error, the zero value
var de *scalar.DecodeError
if errors.As(err, &de) {
	fmt.Println(doc.At(de.Path)) // the path leads At to the value
}
```

```
scalar: Decode: salary.max: want int, got "competitive"
0
competitive
```

#### `To.strict`

```go
years := scalar.Array("3", "five", 7)
fmt.Println(years.IntSlice())     // Perl's numeric context: never fails, "five" is 0
_, err := scalar.To[[]int](years) // To checks
fmt.Println(err)
```

```
[3 0 7]
scalar: Decode: [1]: want int, got "five"
```

#### `To.loose`

```go
doc, _ := scalar.FromJSON([]byte(`{"title": "Go Developer", "extra": {"benefits": ["Pension"]}}`))
type Posting struct {
	Title string     `json:"title"`
	Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar: a deep copy
}
p, _ := scalar.To[Posting](doc)
fmt.Println(p.Title, p.Extra.At("benefits[0]"))
```

```
Go Developer Pension
```

#### `SV.Decode`

Decode stores s in the Go value v points to, as `json.Unmarshal` would store the same data, but converting each value as Perl would, so a number that came as a string, "45000", fills an int field.

```go
doc, _ := scalar.FromJSON([]byte(`{
	"title": "Go Developer",
	"skills": ["Go", "SQL"],
	"salary": {"min": "45000", "max": 55000},
	"extra": {"benefits": ["Pension"]}
}`))
var posting struct {
	Title  string   `json:"title"`
	Skills []string `json:"skills"`
	Salary struct {
		Min int `json:"min"` // "45000" fills it, as Perl would numify it
		Max int `json:"max"`
	} `json:"salary"`
	Extra *scalar.SV `json:"extra"` // a part that stays loose
}
if err := doc.Decode(&posting); err != nil {
	panic(err)
}
fmt.Println(posting.Title, posting.Skills, posting.Salary.Min, posting.Salary.Max)
fmt.Println(posting.Extra.At("benefits[0]"))
```

```
Go Developer [Go SQL] 45000 55000
Pension
```

#### `DecodeError.Error`

```go
doc, _ := scalar.FromJSON([]byte(`{"salary": {"max": "competitive"}}`))
var posting struct {
	Salary struct {
		Max int `json:"max"`
	} `json:"salary"`
}
err := doc.Decode(&posting)
fmt.Println(err)
var de *scalar.DecodeError
if errors.As(err, &de) {
	fmt.Println(doc.At(de.Path)) // the path leads At to the value
}
```

```
scalar: Decode: salary.max: want int, got "competitive"
competitive
```

<!-- reference:end -->
