# scalar

Perl-style dynamic values for Go.

## Why this exists

Go's type system is a real strength, and most of this project should use it.
But two jobs in this project are awkward in plain Go:

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
text := resp.Path("content", 0, "text").Str()
```

If anything along the path is missing or the wrong type, `text` is just `""`.

**2. Heavy text processing.** Cleaning up job postings means a lot of regex
work. Go's standard `regexp` package deliberately leaves out lookahead,
lookbehind and backreferences. `scalar` uses an engine that has them, with a
timeout so a bad pattern can't hang the program. It also wraps them in short
helpers modelled on Perl's `s///`, `split` and `tr///`.

### Borrowing from Perl, not switching to it

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

You don't need to know Perl to use it. The comparisons with Perl in this
README are there to help readers who do.

### What it costs

Dynamic values give up compile-time checking. A misspelled key compiles
without complaint and silently reads as undef. That's why the rule for this
package is:

> **Use `scalar` at the edges, where data comes in. Convert to ordinary Go
> structs as soon as you know the shape, and keep the core of the program
> typed.**

## The types

`scalar` gives you three types:

| Type | Perl | What it is |
|------|------|------------|
| `SV` | `$x` | A scalar: undef, a boolean, an integer, a float, a string, or a reference to an `AV` or `HV` |
| `AV` | `@a` | An array of scalars |
| `HV` | `%h` | A hash of scalars, keyed by string |

A scalar converts itself to whatever you ask for, the way Perl does. Ask for a
string and you get a string; ask for a number and you get a number.

## Quick start

```go
import "github.com/nukopian/job-meta/internal/scalar"

resp, err := scalar.FromJSON(body)
if err != nil {
	return err
}

text := resp.Path("content", 0, "text").Str() // $resp->{content}[0]{text}
tokens := resp.Path("usage", "output_tokens").Int()
```

If any step along the path is missing, you get undef back instead of a panic.
`Str()` on undef is `""` and `Int()` is `0`.

## Recipes

Five short examples from this project's work, each combining several parts of
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
fmt.Println(resp.Path("usage", "output_tokens").Or(0))   // 0, if the model didn't say

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

```go
text := scalar.Scalar("Salary: £45,000 - £55,000 per annum")

figures := scalar.Array()
for _, m := range text.M(`£([\d,]+)`, "g").All() {
	figures.Push(m.AV().Get(1).S(`,`, "", "gr"))   // "45,000" → "45000"
}
fmt.Println(figures.MinNum(), figures.MaxNum())    // 45000 55000
```

### Counting skills across postings

Count each skill ignoring case, then list the most common first.

```go
postings := [][]string{
	{"Go", "Docker"},
	{"go", "Kubernetes"},
	{"GO", "Docker", "Perl"},
}

counts := scalar.Hash()
for _, skills := range postings {
	for _, skill := range scalar.Array(skills).All() {
		counts.Get(skill.Fc().Str()).Inc()             // $counts{fc $skill}++
	}
}

top := counts.Keys().Sort(func(x, y *scalar.SV) int {
	return cmp.Or(
		scalar.NumCmp(counts.Get(y.Str()), counts.Get(x.Str())),   // most common first,
		scalar.StrCmp(x, y),                                       // then alphabetically
	)
})
for _, skill := range top.Slice(0, 1, 2).All() {
	fmt.Println(skill, counts.Get(skill.Str()))        // go 3, docker 2, kubernetes 1
}
```

### Building records and writing JSON

Start each record from shared defaults, collect them in an array, and encode
the result.

```go
defaults := scalar.Hash("location", "Unknown", "remote", false)

rows := scalar.Array()
for _, title := range []string{"Go Developer", "Perl Developer"} {
	row := scalar.Hash(defaults, "title", title)   // %row = (%defaults, title => $title)
	rows.Push(scalar.Scalar(row))                  // push @rows, \%row
}
rows.Get(1).Path("remote").Set(true)             // changes only the second row

out, _ := json.Marshal(rows)
// [{"location":"Unknown","remote":false,"title":"Go Developer"},
//  {"location":"Unknown","remote":true,"title":"Perl Developer"}]
```

Each row is a separate copy of `defaults`, so changing one row changes neither
the others nor `defaults`.

## Scalars

Create a scalar with `Scalar`:

```go
s := scalar.Scalar("42")
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

### True and false

These are false: undef, `""`, `"0"`, `0` and `0.0`. Everything else is true,
including `"0.0"`, `"00"` and `" "`.

### Undef

The zero value of `SV` is undef, and so is `scalar.Scalar(nil)`. Check for it
with `Defined()`.

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

### Length

`Length()` counts the characters in the string form of the value, like Perl's
`length`. It counts characters, not bytes, so `"café"` has length 4, and a
number is measured as it prints: `scalar.Scalar(12345).Length()` is 5. The one
difference from Perl is undef: Perl's `length(undef)` is undef, and here it's 0.

### Substrings

`Substr` and `SubstrReplace` are Perl's `substr`. Like `Length`, they count
characters, not bytes.

```go
s := scalar.Scalar("The black cat climbed the green tree")

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
| `rindex($s, "£")` | `s.Rindex("£")` | last position, or -1 |
| `lc($s)`, `uc($s)` | `s.Lc()`, `s.Uc()` | lower or upper case |
| `ucfirst($s)`, `lcfirst($s)` | `s.Ucfirst()`, `s.Lcfirst()` | change just the first character |
| `fc($s)` | `s.Fc()` | case-fold, for comparing regardless of case |
| `trim($s)` | `s.Trim()` | remove leading and trailing whitespace |

They all return a new value and leave the scalar alone, so they chain:
`s.Trim().Lc()`.

`Index` and `Rindex` count characters, like Perl. Go's own `strings.Index`
counts bytes, which gives the wrong answer for `Substr` as soon as the text
contains `£`, `é` or `•`. The positions from `Index` can go straight into
`Substr`:

```go
salary := scalar.Scalar("Salary: £45,000 – £55,000")
salary.Substr(salary.Index("£"), 7)   // "£45,000"
salary.Substr(salary.Rindex("£"))     // "£55,000"
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
  `String`, `ArrayRef` or `HashRef`.
- `Ref()` works like Perl's `ref()`: it returns `"ARRAY"`, `"HASH"` or `""`.
- `Set(x)` assigns a new value and returns the scalar.
- `Copy()` returns a copy of the scalar. See
  [Copies, references and in-place changes](#copies-references-and-in-place-changes).
- `String()` is the same as `Str()`, so `fmt.Println(s)` and `%s` print a
  scalar the way Perl's `print` would.

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
a.Delete(3)                    // delete $a[3]
a.Slice(0, -1)                 // @a[0, -1]
a.Splice(1, 2, "x", "y")       // splice @a, 1, 2, "x", "y"

a.Join(", ")                   // join(", ", @a)
a.Sort(nil)                    // sort @a                  (as strings)
a.Sort(scalar.NumCmp)          // sort { $a <=> $b } @a    (as numbers)
a.Reverse()                    // reverse @a
a.Map(func(s *scalar.SV) any { return s.Int() * 2 })  // map  { $_ * 2 } @a
a.Grep(func(s *scalar.SV) bool { return s.Int() > 2 }) // grep { $_ > 2 } @a

for i, s := range a.All() {
	fmt.Println(i, s)
}
```

`Sort`, `Reverse`, `Map`, `Grep` and `Slice` return a new array and leave the
original alone. `Splice` returns the elements it removed.

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
a.Push(scalar.Scalar(b))        // push @a, \@b      (one element: a reference)
rows.Push(scalar.Scalar(row))   // push @rows, \%row (a list of records)
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

```go
remote := func(s *scalar.SV) bool { return s.M(`(?i)\bremote\b`) != nil }
if bullets.AnyMatch(remote) { ... }

salary := chunks.FirstMatch(func(s *scalar.SV) bool { return s.Index("£") >= 0 })

skills.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr()   // ignoring case
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

h.Get("name")       // $h{name}
h.Set("lang", "Go") // $h{lang} = "Go"
h.Exists("age")     // exists $h{age}
h.Delete("age")     // delete $h{age}
h.Length()          // scalar(keys %h)
h.Scalar()          // scalar(%h), same as Length

h.Keys()            // keys %h
h.Values()          // values %h
h.KeysSlice()       // keys %h,   as a Go []string
h.ValuesSlice()     // values %h, as a Go []*scalar.SV

for k, v := range h.All() {
	fmt.Println(k, v)
}
```

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
`Keys()` has no effect on the hash. Changing an element of `Values()` changes
the value in the hash, because it *is* that value:

```go
for _, v := range h.Values().All() {   // $_ *= 2 for values %h
	v.Set(v.Mul(2))
}
```

Adding or removing elements of the `Values()` array (with `Push`, `Pop` and so
on) doesn't add or remove anything in the hash.

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

`Path(...)` walks a mix of hash keys (strings) and array indexes (ints).
It follows the same rules.

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

`Find` doesn't support JSONPath's filter expressions (`[?...]`). Use the list
utilities on the result instead, which are plain Go:

```go
// The tool_use block in a Bedrock/Claude response.
input := resp.Find("$.content[*]").FirstMatch(func(b *scalar.SV) bool {
	return b.Path("type").Eq("tool_use")
}).Path("input")
```

Things to know:

- **`Find` never creates anything.** Unlike `Path`, a missing key or index
  simply matches nothing.
- **Hash order is unspecified,** as in JSONPath and Perl. Matches from `[*]`
  or `..` on a hash come back in the hash's own order.
- **A malformed path panics,** like a bad regex.
- **References that loop back on themselves are visited once,** so `..`
  can't get stuck.

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
scalar.Scalar("10").Add(5)       // 15
scalar.Scalar("10").Concat(5)    // "105"
scalar.Scalar("10").NumEq(10.0)  // true
scalar.Scalar("10").Eq("10.0")   // false
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

There is no `+=`. Use `s.Set(s.Add(x))` instead.

## Regular expressions

Regexes use [regexp2](https://github.com/dlclark/regexp2) instead of Go's
standard `regexp`. Its syntax is close to Perl's. Unlike Go's standard engine,
it supports lookahead, lookbehind, backreferences and atomic groups.

You write patterns inline, as in Perl. Each pattern is compiled the first time
it's used and then reused.

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

### Building patterns from data

To use text from your data inside a pattern, pass it through
`scalar.Quotemeta`, Perl's `quotemeta` (`\Q...\E`). Skill names are full of
characters that mean something in a regex: without it, `s.M("C++")` panics
with an invalid pattern, and `s.M(".NET")` also matches `"xNET"`.

```go
for _, skill := range []string{"C++", ".NET", "Node.js"} {
	if posting.M(`(?i)` + scalar.Quotemeta(skill)) != nil { ... }
}
```

`Quotemeta` takes a Go string or a scalar, and returns a Go string, ready to
add to a pattern.

### Matching

```go
s := scalar.Scalar("Salary: £45,000 - £55,000")

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

With `g`, `M` finds every match and returns one array per match, each
laid out the same way. If nothing matches, it also returns `nil`, so
`m != nil` means "matched" either way:

```go
for _, m := range s.M(`£([\d,]+)`, "g").All() {
	fmt.Println(m.AV().Get(1))   // 45,000 then 55,000
}
```

`Named` returns the named groups of the first match as a hash, like Perl's
`%+`, or `nil` if nothing matched. It doesn't take `g`, because `%+` only ever
describes one match:

```go
d := scalar.Scalar("2026-10-11").Named(`(?<year>\d{4})-(?<month>\d\d)`)
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
s := scalar.Scalar("a  b   c")
s.S(`\s+`, " ", "g")       // s is now "a b c"; returns 2

name := scalar.Scalar("Smith, John")
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
s := scalar.Scalar("10 apples and 3 pears")
s.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g")
// "20 apples and 6 pears"
```

### Split

```go
scalar.Scalar("a,b,,c,,").Split(`,`)         // a, b, "", c   (trailing empties dropped)
scalar.Scalar("a,b,,c,,").Split(`,`, -1)     // a, b, "", c, "", ""
scalar.Scalar("a,b,c,d").Split(`,`, 2)       // a, "b,c,d"
scalar.Scalar("a1b2c").Split(`(\d)`)         // a, 1, b, 2, c   (captures included)
scalar.Scalar("  some  words ").Split(" ")   // some, words     (awk mode)
scalar.Scalar("abc").Split(``)               // a, b, c
```

These follow Perl's `split` rules, including what happens with empty fields
and zero-width matches.

### Transliteration

`Tr` works like Perl's `tr///`. It changes `s` in place and returns how many
characters matched, as a scalar.

```go
s := scalar.Scalar("hello")
s.Tr("a-z", "A-Z")                                 // "HELLO"; returns 5

scalar.Scalar("banana").Tr("a", "")                // count only: returns 3
scalar.Scalar("h-e-l-l-o").Tr("-", "", "d")        // delete: "hello"
scalar.Scalar("a    b").Tr(" ", " ", "s")          // squeeze: "a b"
scalar.Scalar("hi, you!").Tr("a-zA-Z", " ", "cs")  // "hi you "
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

## Copies, references and in-place changes

### Getting an element gives you the element itself

`a.Get(i)`, `h.Get(k)`, `Path(...)`, `All()` and the elements of `Values()`
give you the value that's actually stored, not a copy. Changing it changes
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

- Copying a scalar copies its value: `other.Copy()`, `scalar.Scalar(other)`
  or `s.Set(other)` all do this. So do `Push` and the other methods that take
  lists, for every value they add.
- Copying a reference doesn't copy the array or hash it points to. Both copies
  point at the same data, just as in Perl.

### What changes in place

Methods either change the value you call them on, or leave it alone and
return something new:

| Changes in place | Returns something new, original untouched |
|---|---|
| `Set`, `Delete`, `Push`, `Pop`, `Shift`, `Unshift`, `Splice`, `SetLastIndex` | `Sort`, `Reverse`, `Map`, `Grep`, `Slice`, `Keys`, `UniqueStr`, `UniqueNum` |
| `SubstrReplace` | `Substr`, `Lc`, `Uc`, `Fc`, `Ucfirst`, `Lcfirst`, `Trim` |
| `Inc`, `Dec` | Operators: `Add`, `Concat`, `Neg`, … |
| `S`, `Tr` | `S` and `Tr` with the `r` flag |

`Delete`, `Pop`, `Shift` and `Splice` return the elements they removed. These
are the actual values, no longer attached to the array or hash, so changing
them afterwards has no effect on it.

`Values()` sits between the two columns: the array is new, but its elements
are the hash's own values, as described under [Hashes](#hashes).

## Caveats

- **It panics where Perl would die.** Most Go code returns errors instead of
  panicking, so be aware of these cases:
  - writing to an array element before the start of the array, or through
    something that isn't an array or hash (reads are always safe)
  - `SubstrReplace` on a substring entirely outside the string
  - `Div` or `Mod` by zero
  - an invalid regex, an unknown flag, or a regex that runs past
    `RegexTimeout`
  - passing something other than a string or int to `Path`, or something
    other than a string or function as an `S` replacement

  If one bad record mustn't stop a batch, catch these with `recover` around
  the code that processes each record.
- **Not for concurrent use:** none of these types are safe to share between
  goroutines without your own locking.
- **No compile-time checks:** a misspelled key won't fail to compile. It just
  reads as undef. Keep these types at the edges of your program, and use
  structs for anything you depend on.
