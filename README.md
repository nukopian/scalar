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

Perl's data model is a well-tested design for exactly this kind of work, so
`scalar` copies it rather than inventing a new one. You don't need to know
Perl to use it. The comparisons with Perl in this README are just there to
help readers who do.

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

### Length

`Length()` counts the characters in the string form of the value, like Perl's
`length`. It counts characters, not bytes, so `"café"` has length 4, and a
number is measured as it prints: `scalar.Scalar(12345).Length()` is 5. The one
difference from Perl is undef: Perl's `length(undef)` is undef, and here it's 0.

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

There are two things Perl refuses to create, and `scalar` refuses them too:

- **An array element before the start of the array**, such as `$a[-10]` on a
  three-element array.
- **An element of something that isn't an array or hash**, such as using a
  string as if it were a hash.

For both, reading gives undef, even if you read further through them with
`AV()`, `HV()` or `Path`. Writing panics, just as Perl dies.

Strict Perl dies on some of these *reads* as well. `scalar` lets reads carry
on, because a read that can't panic is what makes `Path` safe on messy data.

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
| `$s =~ /re/` | `s.Matches(re)` |
| `my @m = $s =~ /re/` | `s.Match(re)` |
| `$s =~ /re/g` | `s.MatchAll(re)` |
| `%+` | `s.MatchNamed(re)` |
| `$s =~ s/re/x/` | `s.Subst(re, "x")` |
| `$s =~ s/re/x/g` | `s.Subst(re, "x", "g")` |
| `$s =~ s/re/x/r` | `s.Subst(re, "x", "r")` |
| `$s =~ s/re/f($1)/e` | `s.Subst(re, func(m *scalar.AV) any { ... })` |
| `split /re/, $s` | `s.Split(re)` |
| `$s =~ tr/a-z/A-Z/` | `s.Tr("a-z", "A-Z")` |

Flags go in an optional last argument, as one string: `s.Subst(re, "x", "gi")`.

- **Everywhere:** `i` (ignore case), `m` (multi-line `^`/`$`), `s` (`.` matches
  newline), `x` (allow spaces and comments in the pattern).
- **`Subst` only:** `g` (replace every match) and `r` (return a new string
  instead of changing `s`).
- **`Tr` has its own flags:** `c`, `d`, `s` and `r`, the same as Perl's `tr///`.

Use Go raw strings (backquotes) for patterns, so backslashes don't need
escaping.

### Matching

```go
s := scalar.Scalar("Salary: £45,000 - £55,000")

if s.Matches(`(?i)salary`) { ... }

m := s.Match(`£([\d,]+)\s*-\s*£([\d,]+)`)
m.Get(0) // "£45,000 - £55,000"   ($&)
m.Get(1) // "45,000"              ($1)
m.Get(2) // "55,000"              ($2)
```

Unlike Perl's list of captures, which starts at `$1`, `Match` puts the whole
match at index 0. That way `Get(n)` is always `$n`.

`Match` returns `nil` if nothing matched. It's still safe to read:
`s.Match(re).Get(1).Str()` is just `""`. A group that didn't take part in the
match is undef.

`MatchAll` returns one array per match:

```go
for _, m := range s.MatchAll(`£([\d,]+)`).All() {
	fmt.Println(m.AV().Get(1))
}
```

`MatchNamed` returns named groups as a hash:

```go
d := scalar.Scalar("2026-10-11").MatchNamed(`(?<year>\d{4})-(?<month>\d\d)`)
d.Get("year").Int() // 2026
```

**Group numbering differs from Perl.** Named groups are numbered *after* all
the unnamed ones. In `(?<y>\d+)-(\d+)`, the unnamed group is `$1` and `y` is
`$2`. If you name your groups, use `MatchNamed`.

### Substitution

`Subst` changes `s` in place and returns how many substitutions it made, as a
scalar (so use `.Int()` to get a Go `int64`). If nothing matched, `s` is left
exactly as it was.

```go
s := scalar.Scalar("a  b   c")
s.Subst(`\s+`, " ", "g")       // s is now "a b c"; returns 2

name := scalar.Scalar("Smith, John")
name.Subst(`(\w+), (\w+)`, "$2 $1")   // "John Smith"
```

In a replacement string you can use `$1`, `${name}`, `$&` (the whole match)
and `$$` (a literal `$`).

With `r`, `s` isn't changed and you get the new string back:

```go
clean := s.Subst(`\s+`, " ", "gr").Str()
```

For computed replacements, as in `s///e`, pass a function. It gets the match
in the same layout as `Match` returns:

```go
s := scalar.Scalar("10 apples and 3 pears")
s.Subst(`(\d+)`, func(m *scalar.AV) any { return m.Get(1).Mul(2) }, "g")
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
  or `s.Set(other)` all do this.
- Copying a reference doesn't copy the array or hash it points to. Both copies
  point at the same data, just as in Perl.

### What changes in place

Methods either change the value you call them on, or leave it alone and
return something new:

| Changes in place | Returns something new, original untouched |
|---|---|
| `Set`, `Delete`, `Push`, `Pop`, `Shift`, `Unshift`, `Splice`, `SetLastIndex` | `Sort`, `Reverse`, `Map`, `Grep`, `Slice`, `Keys` |
| `Inc`, `Dec` | Operators: `Add`, `Concat`, `Neg`, … |
| `Subst`, `Tr` | `Subst` and `Tr` with the `r` flag |

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
  - `Div` or `Mod` by zero
  - an invalid regex, an unknown flag, or a regex that runs past
    `RegexTimeout`
  - passing something other than a string or int to `Path`, or something
    other than a string or function as a `Subst` replacement

  If one bad record mustn't stop a batch, catch these with `recover` around
  the code that processes each record.
- **Not for concurrent use:** none of these types are safe to share between
  goroutines without your own locking.
- **No compile-time checks:** a misspelled key won't fail to compile. It just
  reads as undef. Keep these types at the edges of your program, and use
  structs for anything you depend on.
