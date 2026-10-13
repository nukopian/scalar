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

For a worked example of every function and method, see the
[Reference](#reference) at the end.

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
salary := scalar.Scalar("Salary: £45,000 – £55,000")
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
  `String`, `ArrayRef` or `HashRef`.
- `Ref()` works like Perl's `ref()`: it returns `"ARRAY"`, `"HASH"` or `""`.
- `Set(x)` assigns a new value and returns the scalar.
- `Copy()` returns a copy of the scalar. See
  [Copies, references and in-place changes](#copies-references-and-in-place-changes).
- `Any()` returns the scalar as a plain Go value (`nil`, `bool`, `int64`,
  `float64`, `string`, `[]any` or `map[string]any`), for code that doesn't use
  `scalar`. See [Back to plain Go](#back-to-plain-go).
- `String()` is the same as `Str()`, so `fmt.Println(s)` and `%s` print a
  scalar the way Perl's `print` would.

### Formatting, as with sprintf

Go's format verbs treat a scalar as Perl's `sprintf` does, in the context
each verb implies:

| Verbs | Context | Example |
|---|---|---|
| `%d`, `%b`, `%o`, `%x`, `%X`, `%c` | integer (see `Int`) | `fmt.Sprintf("%d", scalar.Scalar("42 apples"))` is `"42"` |
| `%e`, `%E`, `%f`, `%F`, `%g`, `%G` | number (see `Num`) | `fmt.Sprintf("%05.2f", scalar.Scalar("3.14159"))` is `"03.14"` |
| `%t` | boolean (see `Bool`) | `fmt.Sprintf("%t", scalar.Scalar("0"))` is `"false"` |
| `%s`, `%q`, `%v` and the rest | string (see `Str`) | `fmt.Sprintf("%5s", scalar.Scalar("ab"))` is `"   ab"` |

```go
fmt.Printf("%d items, £%.2f each\n", count, price)   // printf "%d items, £%.2f each\n", $count, $price
```

Widths, precisions and flags work as usual. As in Perl, `%x` is numeric:
`"ff"` formats as `0`, not as the hex of its bytes.

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
a.Slice(0, -1)                 // @a[0, -1]
a.KeyValueSlice(0, -1)         // %a[0, -1]   (index/value pairs: 0, $a[0], -1, $a[-1])
a.Splice(1, 2, "x", "y")       // splice @a, 1, 2, "x", "y"
head, tail := a.SplitAt(2)        // my ($x, $y, @tail) = @a   (see below)

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

`Sort`, `Reverse`, `Map`, `Grep`, `Slice` and `SplitAt` return new arrays and
leave the original alone. `Splice` returns the elements it removed.

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

When an array's contents need to go to code that doesn't use `scalar`, convert
it to a Go slice:

| Method | Returns | Each element |
|---|---|---|
| `a.StrSlice()` | `[]string` | as a string: undef and holes become `""` |
| `a.IntSlice()` | `[]int64` | as an integer: anything that isn't a number becomes `0` |
| `a.NumSlice()` | `[]float64` | as a number |
| `a.AnySlice()` | `[]any` | as a plain Go value, nested arrays and hashes included |

`AnySlice` uses the same shapes as `encoding/json`: `nil`, `bool`, `int64`,
`float64`, `string`, `[]any` and `map[string]any`. For a single value, use
`s.Any()`. The results are copies, so changing them doesn't change the array.

```go
skills := scalar.Array(raw).Grep(notBlank).StrSlice()   // back to a []string
```

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

h.Get("name")       // $h{name}
h.Set("lang", "Go") // $h{lang} = "Go"
h.Exists("age")     // exists $h{age}    (the key is there, even if its value is undef)
h.Defined("age")    // defined $h{age}   (the key is there and its value isn't undef)
h.Delete("age")     // delete $h{age}
h.Length()          // scalar(keys %h)
h.Scalar()          // scalar(%h), same as Length

h.Slice("a", "b")   // @h{qw(a b)}   (copies; missing keys are undef and aren't created)
h.KeyValueSlice("a", "b")   // %h{qw(a b)}   (key/value pairs, in that order)

h.Keys()            // keys %h
h.Values()          // values %h
h.KeysSlice()       // keys %h,   as a Go []string
h.ValuesSlice()     // values %h, as a Go []*scalar.SV

for k, v := range h.All() {
	fmt.Println(k, v)
}
```

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

A key/value slice gives a list of pairs, as in Perl, so building a hash
from some of another's keys is the same one step:

```go
h1 := scalar.Hash(h2.KeyValueSlice("k1", "k2", "k3"))   // %h1 = %h2{qw(k1 k2 k3)}
```

As in Perl, every key asked for is included, in that order: a missing key
comes with an undef value, so `h1` has all three keys, though `h2` isn't
changed. The values are copies.

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

You write patterns inline, as in Perl. Each pattern is compiled the first time
it's used and then reused. You can also compile a pattern up front, as with
Perl's `qr//`; see [Compiling patterns up front](#compiling-patterns-up-front).

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
like Perl's `qr//`. `M`, `Named`, `S` and `Split` take one wherever they
take a pattern string:

```go
var salary = scalar.MustCompile(`£([\d,]+)`) // my $salary = qr/£([\d,]+)/;

for _, m := range s.M(salary, "g").All() { ... }
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
- **Encoding can fail**, with an error rather than a panic, in two cases JSON
  can't represent: a reference that leads back to itself (a hash that
  contains itself, directly or further down), and a NaN or infinite number.
  The same array or hash appearing twice, without a loop, is fine: it's
  written out each time.

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
| `Set`, `Delete`, `Push`, `Pop`, `Shift`, `Unshift`, `Splice`, `SetLastIndex`, `SortInPlace`, `ReverseInPlace` | `Sort`, `Reverse`, `Map`, `Grep`, `Slice` and `KeyValueSlice` (arrays and hashes), `Keys`, `SplitAt`, `SplitAtSlice`, `UniqueStr`, `UniqueNum`, `Sum`, `Sum0`, `Product`, `Reduce` |
| `SubstrReplace` | `Substr`, `Lc`, `Uc`, `Fc`, `UcFirst`, `LcFirst`, `Trim` |
| `Inc`, `Dec`, `AddInPlace` and the other assignment operators | Operators: `Add`, `Concat`, `Neg`, … |
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
  - an invalid regex (use `scalar.Compile` to get an error instead), an
    unknown flag, a regex that runs past `RegexTimeout`, or a pattern that's
    neither a string nor a `*Regexp`
  - passing something other than a string or int to `Path`, or something
    other than a string or function as an `S` replacement
  - `Any()` on a reference that leads back to itself (encoding it as JSON
    returns an error instead)

  If one bad record mustn't stop a batch, catch these with `recover` around
  the code that processes each record.
- **Not for concurrent use:** none of these types are safe to share between
  goroutines without your own locking.
- **No compile-time checks:** a misspelled key won't fail to compile. It just
  reads as undef. Keep these types at the edges of your program, and use
  structs for anything you depend on.

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
- [Regular expressions](#reference-regular-expressions)
- [JSON and JSONPath](#reference-json-and-jsonpath)

### Reference: Scalars

#### Scalar

Scalar returns a scalar holding x.

```go
fmt.Println(scalar.Scalar(42).Kind() == scalar.Int)
fmt.Println(scalar.Scalar(3.5).Kind() == scalar.Float)
fmt.Println(scalar.Scalar("text").Kind() == scalar.String)
fmt.Println(scalar.Scalar([]string{"a", "b"}).Ref()) // a slice becomes an array reference
fmt.Println(scalar.Scalar(nil).Defined())            // nil is undef
```

```
true
true
true
ARRAY
false
```

#### SV.Set

Set assigns x to s and returns s.

```go
s := scalar.Scalar("old")
s.Set(10)                     // $s = 10
fmt.Println(s.Add(1))         // it's a number now
s.Set(map[string]any{"a": 1}) // a map becomes a hash reference
fmt.Println(s.Ref())
```

```
11
HASH
```

#### SV.Copy

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

#### SV.Kind

Kind reports what s currently holds.

```go
for _, x := range []any{nil, true, 1, 1.5, "s", []any{}, map[string]any{}} {
	switch scalar.Scalar(x).Kind() {
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
array
hash
```

#### SV.Defined

Defined is Perl's defined().

```go
fmt.Println(scalar.Scalar(nil).Defined())
fmt.Println(scalar.Scalar("").Defined()) // empty, but defined
fmt.Println(scalar.Hash().Get("missing").Defined())
```

```
false
true
false
```

#### SV.Any

Any returns s as a plain Go value, in the shapes encoding/json uses: undef is nil; booleans, integers, floats and strings are bool, int64, float64 and string; array references become []any and hash references map[string]any, converted all the way down.

```go
s := scalar.Scalar(map[string]any{"tags": []any{"go", 1}})
plain := s.Any().(map[string]any) // plain Go, sharing nothing with s
fmt.Println(plain["tags"])
```

```
[go 1]
```

#### SV.Or

Or is Perl's defined-or, $s // $def: a new scalar holding s's value if s is defined, otherwise def's.

```go
fmt.Println(scalar.Scalar(nil).Or("default")) // $x // "default"
fmt.Println(scalar.Scalar(0).Or("default"))   // 0 is defined, so it's kept
```

```
default
0
```

#### SV.OrSet

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

#### SV.Ref

Ref is Perl's ref(): "ARRAY", "HASH", or "" for non-references.

```go
fmt.Println(scalar.Scalar(scalar.Array(1)).Ref())
fmt.Println(scalar.Scalar(scalar.Hash()).Ref())
fmt.Printf("%q\n", scalar.Scalar("text").Ref())
```

```
ARRAY
HASH
""
```

#### SV.Str

Str returns s in string context.

```go
fmt.Println(scalar.Scalar(42).Str())
fmt.Println(scalar.Scalar(0.1 + 0.2).Str()) // Perl's 15 significant digits
fmt.Println(scalar.Scalar(true).Str())
fmt.Printf("%q\n", scalar.Scalar(nil).Str())
```

```
42
0.3
1
""
```

#### SV.String

String implements fmt.Stringer.

```go
fmt.Println(scalar.Scalar(1e21)) // fmt uses String, as Perl's print would
```

```
1e+21
```

#### SV.Format

Format implements fmt.Formatter, so that Go's format verbs treat s as Perl's sprintf does, in the context each verb implies: %d, %b, %o, %x, %X and %c in integer context (see Int), %e, %E, %f, %F, %g and %G in numeric context (see Num), %t in boolean context, and %s, %q, %v and anything else in string context (see Str).

```go
fmt.Printf("%d|%05.2f|%x|%5s|%t\n",
	scalar.Scalar("42 apples"), scalar.Scalar("3.14159"), scalar.Scalar(255),
	scalar.Scalar("ab"), scalar.Scalar("0"))
```

```
42|03.14|ff|   ab|false
```

#### SV.Num

Num returns s in numeric context.

```go
fmt.Println(scalar.Scalar("3.5kg").Num()) // the numeric prefix, as Perl does
fmt.Println(scalar.Scalar("abc").Num())
```

```
3.5
0
```

#### SV.NumOK

NumOK is Num, plus false wherever Perl would warn: undef, empty or non-numeric strings, and strings with trailing garbage.

```go
for _, x := range []any{"42", " 42 ", "42abc", "", nil} {
	n, ok := scalar.Scalar(x).NumOK()
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

#### SV.Int

Int returns s in integer context, truncating toward zero.

```go
fmt.Println(scalar.Scalar(-3.9).Int()) // truncates toward zero
fmt.Println(scalar.Scalar("12 items").Int())
```

```
-3
12
```

#### SV.IntOK

IntOK is Int, plus false wherever NumOK would be false, and for NaN or floats outside the int64 range (which are clamped).

```go
i, ok := scalar.Scalar("9223372036854775807").IntOK() // exact, however large
fmt.Println(i, ok)
i, ok = scalar.Scalar(1e30).IntOK() // out of range: clamped
fmt.Println(i, ok)
```

```
9223372036854775807 true
9223372036854775807 false
```

#### SV.Bool

Bool returns s in boolean context: undef, "", "0", 0 and 0.0 are false; everything else, including "0.0" and "00", is true.

```go
for _, x := range []any{nil, "", "0", 0, 0.0, "0.0", "00", " ", "a"} {
	fmt.Print(scalar.Scalar(x).Bool(), " ")
}
fmt.Println()
```

```
false false false false false true true true true
```

#### SV.AV

AV dereferences s as an array.

```go
var s scalar.SV   // undef
s.AV().Push(1, 2) // push @$s, 1, 2: autovivifies an array
fmt.Println(s.AV().Join(","))
fmt.Println(scalar.Scalar("text").AV() == nil) // not an array
```

```
1,2
true
```

#### SV.HV

HV dereferences s as a hash, autovivifying undef like AV does.

```go
var s scalar.SV
s.HV().Set("a", 1) // $s->{a} = 1
fmt.Println(s.Ref(), s.HV().Get("a"))
```

```
HASH 1
```

#### SV.Path

Path walks a chain of hash keys (string) and array indices (int), so s.Path("content", 0, "text") is $s->{content}[0]{text}.

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

#### SV.PathOK

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

#### StrCmp

StrCmp is Perl's cmp, for use with AV.Sort.

```go
fmt.Println(scalar.Array("b", "10", "a", "9").Sort(scalar.StrCmp).Join(" ")) // sort { $a cmp $b }
```

```
10 9 a b
```

#### NumCmp

NumCmp is Perl's <=>, for use with AV.Sort.

```go
fmt.Println(scalar.Array("10", 9, "2.5").Sort(scalar.NumCmp).Join(" ")) // sort { $a <=> $b }
```

```
2.5 9 10
```

### Reference: Strings

#### SV.Length

Length is Perl's length($s): the number of characters (not bytes) in s in string context, so Scalar(12345).Length() is 5 and "café" is 4.

```go
fmt.Println(scalar.Scalar("café").Length()) // characters, not bytes
fmt.Println(scalar.Scalar(12345).Length())  // in string context
```

```
4
5
```

#### SV.Substr

Substr is Perl's substr($s, offset) or substr($s, offset, length): a new scalar holding part of s in string context, counted in characters.

```go
s := scalar.Scalar("Senior Go Developer")
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

#### SV.SubstrReplace

SubstrReplace is Perl's 4-argument substr($s, offset, length, replacement), or assigning to substr: it replaces that part of s with replacement (in string context), changing s in place, and returns the part it replaced.

```go
s := scalar.Scalar("Junior Go Developer")
old := s.SubstrReplace(0, 6, "Senior") // substr($s, 0, 6) = "Senior"
fmt.Println(s, "/", old)
```

```
Senior Go Developer / Junior
```

#### SV.Index

Index is Perl's index($s, substr[, position]): the character position of the first occurrence of substr at or after position (default 0), or -1.

```go
s := scalar.Scalar("café au lait")
fmt.Println(s.Index("au"))   // a character position, ready for Substr
fmt.Println(s.Index("a", 2)) // starting from position 2
fmt.Println(s.Index("tea"))
```

```
5
5
-1
```

#### SV.RIndex

RIndex is Perl's rindex($s, substr[, position]): the character position of the last occurrence of substr starting at or before position (default the end of the string), or -1.

```go
s := scalar.Scalar("a/b/c")
fmt.Println(s.RIndex("/"))    // the last one
fmt.Println(s.RIndex("/", 2)) // the last one at or before position 2
```

```
3
1
```

#### SV.Lc

Lc is Perl's lc: s in lower case, using Unicode's full case mappings.

```go
fmt.Println(scalar.Scalar("ÉCOLE Go").Lc())
```

```
école go
```

#### SV.Uc

Uc is Perl's uc: s in upper case, using Unicode's full case mappings, so "straße" becomes "STRASSE".

```go
fmt.Println(scalar.Scalar("straße").Uc()) // full Unicode mappings
```

```
STRASSE
```

#### SV.Fc

Fc is Perl's fc: s case-folded, for comparing strings regardless of case.

```go
a, b := scalar.Scalar("Straße"), scalar.Scalar("STRASSE")
fmt.Println(a.Lc().Eq(b.Lc())) // lc isn't enough for this
fmt.Println(a.Fc().Eq(b.Fc())) // fc is: compare case-insensitively with it
```

```
false
true
```

#### SV.UcFirst

UcFirst is Perl's ucfirst: s with its first character in title case.

```go
fmt.Println(scalar.Scalar("élan vital").UcFirst())
```

```
Élan vital
```

#### SV.LcFirst

LcFirst is Perl's lcfirst: s with its first character in lower case.

```go
fmt.Println(scalar.Scalar("HELLO").LcFirst())
```

```
hELLO
```

#### SV.Trim

Trim is Perl's trim (builtin::trim, Perl 5.36): s without leading or trailing whitespace, including Unicode spaces such as no-break space.

```go
fmt.Printf("%q\n", scalar.Scalar("\u00a0 padded \t").Trim()) // Unicode spaces too
```

```
"padded"
```

#### QuoteMeta

QuoteMeta is Perl's quotemeta: x in string context with every character that could mean something in a pattern backslash-escaped, so that it matches literally.

```go
fmt.Println(scalar.QuoteMeta("C++ (.NET)"))
posting := scalar.Scalar("Skills: C++ and .NET")
fmt.Println(posting.M(scalar.QuoteMeta("C++")) != nil) // a literal match
```

```
C\+\+\ \(\.NET\)
true
```

### Reference: Operators

#### SV.Add

Add is Perl's +.

```go
fmt.Println(scalar.Scalar("40").Add(2))          // "40" + 2
fmt.Println(scalar.Scalar(1.5).Add("1.5"))       // floats when needed
fmt.Println(scalar.Scalar(math.MaxInt64).Add(1)) // an integer that would overflow becomes a float
```

```
42
3
9.22337203685478e+18
```

#### SV.Sub

Sub is Perl's -.

```go
fmt.Println(scalar.Scalar(10).Sub("2.5"))
```

```
7.5
```

#### SV.Mul

Mul is Perl's *.

```go
fmt.Println(scalar.Scalar("6").Mul(7))
```

```
42
```

#### SV.Div

Div is Perl's /.

```go
fmt.Println(scalar.Scalar(7).Div(2)) // not integer division
fmt.Println(scalar.Scalar(8).Div(2))
```

```
3.5
4
```

#### SV.Mod

Mod is Perl's %: both operands are truncated to integers, and a nonzero result takes the sign of the right operand (-7 % 3 is 2).

```go
fmt.Println(scalar.Scalar(7).Mod(3))
fmt.Println(scalar.Scalar(-7).Mod(3)) // takes the sign of the right operand
```

```
1
2
```

#### SV.Pow

Pow is Perl's **.

```go
fmt.Println(scalar.Scalar(2).Pow(10))
fmt.Println(scalar.Scalar(2).Pow(-1))
```

```
1024
0.5
```

#### SV.Neg

Neg is Perl's unary minus, string madness included: -"foo" is "-foo", -"-foo" is "+foo", and -"+foo" is "-foo".

```go
fmt.Println(scalar.Scalar(5).Neg())
fmt.Println(scalar.Scalar("foo").Neg()) // string madness, as in Perl
fmt.Println(scalar.Scalar("-foo").Neg())
```

```
-5
-foo
+foo
```

#### SV.Concat

Concat is Perl's .

```go
fmt.Println(scalar.Scalar("Go").Concat(1.5)) // "Go" . 1.5
```

```
Go1.5
```

#### SV.Repeat

Repeat is Perl's x operator on a scalar.

```go
fmt.Println(scalar.Scalar("ab").Repeat(3)) // "ab" x 3
fmt.Printf("%q\n", scalar.Scalar("ab").Repeat(0))
```

```
ababab
""
```

#### SV.Inc

Inc is Perl's ++: it modifies s in place and returns it.

```go
n := scalar.Scalar(9)
n.Inc()
id := scalar.Scalar("az9")
id.Inc() // Perl's magic string increment
fmt.Println(n, id)
```

```
10 ba0
```

#### SV.Dec

Dec is Perl's --: it modifies s in place and returns it.

```go
s := scalar.Scalar("aa")
s.Dec() // never magic: "aa" is 0 as a number
fmt.Println(s)
```

```
-1
```

#### SV.AddInPlace

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

#### SV.SubInPlace

SubInPlace is Perl's -=.

```go
s := scalar.Scalar(10)
s.SubInPlace(3) // $s -= 3
fmt.Println(s)
```

```
7
```

#### SV.MulInPlace

MulInPlace is Perl's *=.

```go
s := scalar.Scalar(4)
s.MulInPlace(2.5) // $s *= 2.5
fmt.Println(s)
```

```
10
```

#### SV.DivInPlace

DivInPlace is Perl's /=.

```go
s := scalar.Scalar(9)
s.DivInPlace(2) // $s /= 2
fmt.Println(s)
```

```
4.5
```

#### SV.ModInPlace

ModInPlace is Perl's %=.

```go
s := scalar.Scalar(17)
s.ModInPlace(5) // $s %= 5
fmt.Println(s)
```

```
2
```

#### SV.PowInPlace

PowInPlace is Perl's **=.

```go
s := scalar.Scalar(3)
s.PowInPlace(2) // $s **= 2
fmt.Println(s)
```

```
9
```

#### SV.ConcatInPlace

ConcatInPlace is Perl's .=.

```go
s := scalar.Scalar("Go")
s.ConcatInPlace(" developer") // $s .= " developer"
fmt.Println(s)
```

```
Go developer
```

#### SV.RepeatInPlace

RepeatInPlace is Perl's x=.

```go
s := scalar.Scalar("-")
s.RepeatInPlace(5) // $s x= 5
fmt.Println(s)
```

```
-----
```

#### SV.Eq

Eq is Perl's eq.

```go
fmt.Println(scalar.Scalar(1.0).Eq("1")) // eq compares strings: 1.0 is "1"
fmt.Println(scalar.Scalar("1.0").Eq(1))
```

```
true
false
```

#### SV.Ne

Ne is Perl's ne.

```go
fmt.Println(scalar.Scalar("go").Ne("Go"))
```

```
true
```

#### SV.Lt

Lt is Perl's lt.

```go
fmt.Println(scalar.Scalar("10").Lt("9")) // as strings, "10" sorts first
```

```
true
```

#### SV.Le

Le is Perl's le.

```go
fmt.Println(scalar.Scalar("a").Le("a"))
```

```
true
```

#### SV.Gt

Gt is Perl's gt.

```go
fmt.Println(scalar.Scalar("b").Gt("abc"))
```

```
true
```

#### SV.Ge

Ge is Perl's ge.

```go
fmt.Println(scalar.Scalar("abc").Ge("abd"))
```

```
false
```

#### SV.NumEq

NumEq is Perl's ==.

```go
fmt.Println(scalar.Scalar("1.0").NumEq(1)) // == compares numbers
fmt.Println(scalar.Scalar("NaN").NumEq("NaN"))
```

```
true
false
```

#### SV.NumNe

NumNe is Perl's !=.

```go
fmt.Println(scalar.Scalar("10").NumNe(10.0))
```

```
false
```

#### SV.NumLt

NumLt is Perl's <.

```go
fmt.Println(scalar.Scalar("10").NumLt("9")) // as numbers, 9 is smaller
```

```
false
```

#### SV.NumLe

NumLe is Perl's <=.

```go
fmt.Println(scalar.Scalar(2).NumLe("2.0"))
```

```
true
```

#### SV.NumGt

NumGt is Perl's >.

```go
fmt.Println(scalar.Scalar("9007199254740993").NumGt("9007199254740992")) // integers compare exactly
```

```
true
```

#### SV.NumGe

NumGe is Perl's >=.

```go
fmt.Println(scalar.Scalar(-1).NumGe(0))
```

```
false
```

### Reference: Arrays

#### Array

Array returns an array of vals, as Perl's (...) list does: arrays and hashes among vals are flattened into it (see Push), and other values are converted as by SV.Set.

```go
a := scalar.Array("a", []string{"b", "c"}, scalar.Array("d")) // lists flatten, as in Perl
fmt.Println(a.Length(), a.Join(","))
nested := scalar.Array(1, scalar.Scalar(scalar.Array(2, 3))) // a reference stays one element
fmt.Println(nested.Length(), nested.Get(1).AV().Join(","))
```

```
4 a,b,c,d
2 2,3
```

#### AV.Length

Length is Perl's scalar(@a).

```go
fmt.Println(scalar.Array(1, 2, 3).Length()) // scalar(@a)
```

```
3
```

#### AV.Scalar

Scalar is Perl's scalar(@a), the number of elements.

```go
fmt.Println(scalar.Array("x", "y").Scalar()) // scalar(@a), the same as Length
```

```
2
```

#### AV.LastIndex

LastIndex is Perl's $#a: Length() - 1, so -1 for an empty array.

```go
fmt.Println(scalar.Array(1, 2, 3).LastIndex()) // $#a
fmt.Println(scalar.Array().LastIndex())
```

```
2
-1
```

#### AV.SetLastIndex

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

#### AV.Get

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

#### AV.Set

Set assigns v to element i, extending the array if needed.

```go
a := scalar.Array()
a.Set(2, "c") // $a[2] = "c": extends the array, leaving holes
fmt.Println(a.Length(), a.Exists(0), a.Get(2))
```

```
3 false c
```

#### AV.Exists

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

#### AV.Defined

Defined is Perl's defined $a[i]: whether element i exists and isn't undef.

```go
a := scalar.Array("a", nil)
fmt.Println(a.Defined(0), a.Defined(1), a.Defined(-2))
```

```
true false true
```

#### AV.Delete

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

#### AV.Push

Push is Perl's push: it appends vals and returns a.

```go
a := scalar.Array("a")
a.Push("b", []string{"c", "d"})           // push @a, "b", @list
a.Push(scalar.Scalar(scalar.Array(1, 2))) // push @a, [1, 2]: a reference, one element
fmt.Println(a.Length(), a.Get(-1).Ref())
```

```
5 ARRAY
```

#### AV.Unshift

Unshift is Perl's unshift: it prepends vals, keeping their order, and returns a.

```go
a := scalar.Array("c")
a.Unshift("a", "b") // unshift @a, "a", "b": order kept
fmt.Println(a.Join(","))
```

```
a,b,c
```

#### AV.Pop

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

#### AV.Shift

Shift removes and returns the first element, or undef if a is empty.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Shift(), a.Join(","))
```

```
1 2,3
```

#### AV.Splice

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

#### AV.Slice

Slice is Perl's array slice @a[i, j, ...]: a new array of copies of the elements at those indices.

```go
a := scalar.Array("a", "b", "c", "d")
fmt.Println(a.Slice(0, -1, 1).Join(",")) // @a[0, -1, 1]
```

```
a,d,b
```

#### AV.KeyValueSlice

KeyValueSlice is Perl's index/value array slice %a[i, j, ...]: a new array of index/value pairs for those indices, in that order, with copies of the values.

```go
a := scalar.Array("a", "b")
fmt.Println(a.KeyValueSlice(0, -1).Join(",")) // %a[0, -1]: indices as written
```

```
0,a,-1,b
```

#### AV.Items

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

#### AV.StrSlice

StrSlice returns the elements as a Go []string, each in string context (see SV.Str), so undef and holes become "".

```go
fmt.Printf("%q\n", scalar.Array("a", 1, nil).StrSlice())
```

```
["a" "1" ""]
```

#### AV.IntSlice

IntSlice returns the elements as a Go []int64, each in integer context (see SV.Int), so values that aren't numbers become 0.

```go
fmt.Println(scalar.Array("3", 4.9, "x").IntSlice())
```

```
[3 4 0]
```

#### AV.NumSlice

NumSlice returns the elements as a Go []float64, each in numeric context (see SV.Num).

```go
fmt.Println(scalar.Array("1.5", 2).NumSlice())
```

```
[1.5 2]
```

#### AV.AnySlice

AnySlice returns the elements as plain Go values (see SV.Any), for code that doesn't use this package.

```go
fmt.Println(scalar.Array("a", 1, scalar.Scalar(scalar.Array(2))).AnySlice())
```

```
[a 1 [2]]
```

#### AV.All

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

#### AV.Join

Join is Perl's join.

```go
fmt.Println(scalar.Array("Go", "Perl", 5).Join(", "))
```

```
Go, Perl, 5
```

#### AV.Map

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

#### AV.Grep

Grep is Perl's grep: a new array of copies of the elements f accepts.

```go
n := scalar.Array(1, 2, 3, 4).Grep(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })
fmt.Println(n.Join(","))
```

```
2,4
```

#### AV.Sort

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

#### AV.SortInPlace

SortInPlace is Perl's @a = sort @a: it sorts a itself, as Sort would, and returns a.

```go
a := scalar.Array(3, 1, 2)
a.SortInPlace(scalar.NumCmp) // @a = sort { $a <=> $b } @a
fmt.Println(a.Join(","))
```

```
1,2,3
```

#### AV.Reverse

Reverse is Perl's reverse: a reversed copy of a.

```go
a := scalar.Array(1, 2, 3)
fmt.Println(a.Reverse().Join(","), a.Join(","))
```

```
3,2,1 1,2,3
```

#### AV.ReverseInPlace

ReverseInPlace is Perl's @a = reverse @a: it reverses a itself and returns a.

```go
a := scalar.Array(1, 2, 3)
a.ReverseInPlace()
fmt.Println(a.Join(","))
```

```
3,2,1
```

#### AV.SplitAt

SplitAt splits a into its first n elements and the rest, as Perl's my ($x1, ..., $xn, @tail) = @a does.

```go
head, tail := scalar.Array("Ada", "Lovelace", "maths", "poetry").SplitAt(2) // my ($first, $last, @interests) = @a
fmt.Println(head.Join(" "), "|", tail.Join(","))
```

```
Ada Lovelace | maths,poetry
```

#### AV.SplitAtSlice

SplitAtSlice is SplitAt with the head as a Go slice of exactly n elements, so it can be unpacked: h, tail := a.SplitAtSlice(3); x, y, z := h[0], h[1], h[2].

```go
h, rest := scalar.Array("2026", "10", "13", "extra").SplitAtSlice(3)
year, month, day := h[0], h[1], h[2] // my ($year, $month, $day, @rest) = @a
fmt.Println(year, month, day, rest.Length())
```

```
2026 10 13 1
```

### Reference: Hashes

#### Hash

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

#### HV.Length

Length is Perl's scalar(keys %h).

```go
fmt.Println(scalar.Hash("a", 1, "b", 2).Length()) // scalar(keys %h)
```

```
2
```

#### HV.Scalar

Scalar is Perl's scalar(%h), which since Perl 5.26 is the number of keys.

```go
fmt.Println(scalar.Hash("a", 1).Scalar()) // scalar(%h), the same as Length
```

```
1
```

#### HV.Get

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

#### HV.Set

Set assigns v to key and returns the stored SV.

```go
h := scalar.Hash()
h.Set("tags", []string{"go", "perl"}) // $h{tags} = ["go", "perl"]
fmt.Println(h.Get("tags").AV().Join(","))
```

```
go,perl
```

#### HV.Exists

Exists is Perl's exists.

```go
h := scalar.Hash("salary", nil)
fmt.Println(h.Exists("salary"), h.Exists("bonus")) // there, though undef
```

```
true false
```

#### HV.Defined

Defined is Perl's defined $h{key}: whether key exists and its value isn't undef.

```go
h := scalar.Hash("salary", nil, "title", "Developer")
fmt.Println(h.Defined("salary"), h.Defined("title"))
```

```
false true
```

#### HV.Delete

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

#### HV.Slice

Slice is Perl's hash slice @h{k1, k2, ...}: a new array of copies of the values for those keys, in that order.

```go
h := scalar.Hash("a", 1, "b", 2)
fmt.Println(h.Slice("b", "a").Join(","))                     // @h{qw(b a)}
fmt.Println(h.Slice("missing").Get(0).Defined(), h.Length()) // undef, and not created
```

```
2,1
false 2
```

#### HV.KeyValueSlice

KeyValueSlice is Perl's key/value hash slice %h{k1, k2, ...}: a new array of key/value pairs for those keys, in that order, with copies of the values.

```go
h2 := scalar.Hash("k1", 1, "k2", 2, "k3", 3)
h1 := scalar.Hash(h2.KeyValueSlice("k1", "k3")) // %h1 = %h2{qw(k1 k3)}
fmt.Println(h1.Keys().Sort(nil).Join(","), h1.Get("k3"))
fmt.Println(h2.KeyValueSlice("k2", "missing").Join(",")) // a missing key comes with undef
```

```
k1,k3 3
k2,2,missing,
```

#### HV.Keys

Keys is Perl's keys %h: a new array of the keys.

```go
h := scalar.Hash("b", 2, "a", 1)
fmt.Println(h.Keys().Sort(nil).Join(",")) // sort keys %h
```

```
a,b
```

#### HV.Values

Values is Perl's values %h, in the same order as Keys.

```go
h := scalar.Hash("a", 1, "b", 2)
for _, v := range h.Values().All() { // $_ *= 10 for values %h
	v.MulInPlace(10)
}
fmt.Println(h.Get("a"), h.Get("b"))
```

```
10 20
```

#### HV.KeysSlice

KeysSlice is Keys as a Go []string.

```go
keys := scalar.Hash("a", 1).KeysSlice() // a Go []string
fmt.Printf("%q\n", keys)
```

```
["a"]
```

#### HV.ValuesSlice

ValuesSlice is Values as a Go []*SV.

```go
values := scalar.Hash("a", "x").ValuesSlice() // a Go []*scalar.SV
fmt.Println(len(values), values[0])
```

```
1 x
```

#### HV.All

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

### Reference: List utilities

#### AV.AnyMatch

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

#### AV.AllMatch

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

#### AV.NoneMatch

NoneMatch is List::Util's none: whether f is false for every element.

```go
fmt.Println(scalar.Array("Leeds", "York").NoneMatch(isRemote))
```

```
true
```

#### AV.FirstMatch

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

#### AV.MaxNum

MaxNum is List::Util's max: the element with the largest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MaxNum()) // max: as numbers
```

```
10
```

#### AV.MinNum

MinNum is List::Util's min: the element with the smallest numeric value, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", 9, "2.5").MinNum())
```

```
2.5
```

#### AV.MaxStr

MaxStr is List::Util's maxstr: the element that sorts last as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("10", "9", "100").MaxStr()) // maxstr: as strings
```

```
9
```

#### AV.MinStr

MinStr is List::Util's minstr: the element that sorts first as a string, or undef for an empty array.

```go
fmt.Println(scalar.Array("b", "a", "c").MinStr())
```

```
a
```

#### AV.UniqueStr

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

#### AV.UniqueNum

UniqueNum is List::Util's uniqnum: a new array of copies of the elements, keeping only the first of each distinct number, in order.

```go
fmt.Println(scalar.Array(1, "1.0", "01", 2).UniqueNum().Join(",")) // the same number, however written
```

```
1,2
```

#### AV.Sum

Sum is List::Util's sum: the numeric sum of the elements, or undef for an empty array.

```go
fmt.Println(scalar.Array(1, "2", 3.5).Sum()) // sum @a
fmt.Println(scalar.Array().Sum().Defined())  // undef for nothing
```

```
6.5
false
```

#### AV.Sum0

Sum0 is List::Util's sum0: Sum, but 0 for an empty array.

```go
fmt.Println(scalar.Array().Sum0()) // sum0: 0 for nothing
```

```
0
```

#### AV.Product

Product is List::Util's product: the numeric product of the elements, or 1 for an empty array, as in List::Util.

```go
fmt.Println(scalar.Array(2, 3, 4).Product())
fmt.Println(scalar.Array().Product()) // 1 for nothing
```

```
24
1
```

#### AV.Reduce

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

### Reference: Regular expressions

#### SV.M

M is $s =~ /re/ with captures.

```go
s := scalar.Scalar("Salary: £45,000 - £55,000")
if s.M(`(?i)salary`) != nil { // if ($s =~ /salary/i)
	fmt.Println("matched")
}
m := s.M(`£([\d,]+)\s*-\s*£([\d,]+)`)
fmt.Println(m.Get(0)) // $&
fmt.Println(m.Get(1), m.Get(2))
for _, each := range s.M(`£([\d,]+)`, "g").All() { // $s =~ /£([\d,]+)/g
	fmt.Println(each.AV().Get(1))
}
fmt.Println(s.M(`remote`).Get(1).Defined()) // no match: nil, still safe to read
```

```
matched
£45,000 - £55,000
45,000 55,000
45,000
55,000
false
```

#### SV.Named

Named is $s =~ /re/ followed by reading %+.

```go
d := scalar.Scalar("Closing 2026-11-30").Named(`(?<year>\d{4})-(?<month>\d\d)-(?<day>\d\d)`)
fmt.Println(d.Get("year"), d.Get("month"), d.Get("day")) // $+{year} and so on
```

```
2026 11 30
```

#### SV.S

S is Perl's s///.

```go
s := scalar.Scalar("a  b   c")
n := s.S(`\s+`, " ", "g") // $s =~ s/\s+/ /g
fmt.Println(s, n)
name := scalar.Scalar("Lovelace, Ada")
name.S(`(\w+), (\w+)`, "$2 $1") // $1 and $2 in the replacement
fmt.Println(name)
clean := scalar.Scalar("x1y2").S(`\d`, "", "gr") // s///r: a new string; the original is unchanged
fmt.Println(clean)
doubled := scalar.Scalar("3 apples, 4 pears")
doubled.S(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g") // s///e
fmt.Println(doubled)
```

```
a b c 2
Ada Lovelace
xy
6 apples, 8 pears
```

#### SV.Split

Split is Perl's split /re/, $s.

```go
fmt.Println(scalar.Scalar("a,b,,c,,").Split(`,`).Length())        // trailing empty fields dropped
fmt.Println(scalar.Scalar("a1b2c").Split(`(\d)`).Join("|"))       // captures included
fmt.Println(scalar.Scalar("a,b,c,d").Split(`,`, 2).Join("|"))     // a limit
fmt.Println(scalar.Scalar("  some  words ").Split(" ").Join("|")) // awk mode
```

```
4
a|1|b|2|c
a|b,c,d
some|words
```

#### SV.Tr

Tr is Perl's tr/// (y///).

```go
s := scalar.Scalar("hello")
s.Tr("a-z", "A-Z") // $s =~ tr/a-z/A-Z/
fmt.Println(s)
fmt.Println(scalar.Scalar("hello world").Tr("a-z", ""))  // counts: tr/a-z// returns how many
fmt.Println(scalar.Scalar("aabbcc").Tr("a-z", "", "sr")) // squeeze, returning a new string
```

```
HELLO
10
abc
```

#### Compile

Compile compiles pattern with flags, any of "i", "m", "s" and "x", or returns an error if the pattern is invalid.

```go
re, err := scalar.Compile(`£([\d,]+`) // a pattern from outside the program: check it
fmt.Println(re == nil, err != nil)
```

```
true true
```

#### MustCompile

MustCompile is Compile, panicking if the pattern is invalid.

```go
salary := scalar.MustCompile(`£([\d,]+)`) // my $salary = qr/£([\d,]+)/
s := scalar.Scalar("£45,000 to £55,000")
fmt.Println(s.M(salary, "g").Length()) // use it anywhere a pattern string goes
fmt.Println(s.S(salary, "£X", "gr"))
insensitive := scalar.MustCompile(`remote`, "i") // flags are fixed when compiling
fmt.Println(scalar.Scalar("REMOTE").M(insensitive) != nil)
```

```
2
£X to £X
true
```

#### Regexp.String

String returns the pattern as written.

```go
fmt.Println(scalar.MustCompile(`\d+`, "i"))
```

```
\d+
```

### Reference: JSON and JSONPath

#### FromJSON

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

#### SV.MarshalJSON

MarshalJSON implements json.Marshaler.

```go
h := scalar.Hash("b", 1, "a", scalar.Scalar(scalar.Array("x", nil)))
out, _ := json.Marshal(scalar.Scalar(h)) // keys sorted, undef as null
fmt.Println(string(out))
h.Set("self", scalar.Scalar(h)) // a loop can't be JSON: an error, not a crash
_, err := json.Marshal(scalar.Scalar(h))
fmt.Println(err != nil)
```

```
{"a":["x",null],"b":1}
true
```

#### AV.MarshalJSON

MarshalJSON implements json.Marshaler, as SV.MarshalJSON does.

```go
out, _ := json.Marshal(scalar.Array(1, "two", 3.5))
fmt.Println(string(out))
```

```
[1,"two",3.5]
```

#### HV.MarshalJSON

MarshalJSON implements json.Marshaler, as SV.MarshalJSON does.

```go
out, _ := json.Marshal(scalar.Hash("z", 1, "a", 2))
fmt.Println(string(out))
```

```
{"a":2,"z":1}
```

#### SV.UnmarshalJSON

UnmarshalJSON implements json.Unmarshaler.

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

#### SV.Find

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

<!-- reference:end -->
