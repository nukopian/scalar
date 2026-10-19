# scalar: reference

What each part of [scalar](../README.md) does, feature by feature: how it
behaves, where it follows Perl or another source, and where it doesn't.
For tasks, see the [how-to guide](HOWTO.md); for one example of every
function and method, see the [examples](EXAMPLES.md); coming from Perl,
see the [phrasebook](PERL.md).

- [The types](#the-types)
- [Scalars](#scalars)
- [Arrays](#arrays)
- [Hashes](#hashes)
- [Sets and bags](#sets-and-bags)
- [Nesting and autovivification](#nesting-and-autovivification)
- [Finding values with JSONPath](#finding-values-with-jsonpath)
- [Changing every value: `Apply`](#changing-every-value-apply)
- [Comparing structures: `Equal` and `Diff`](#comparing-structures-equal-and-diff)
- [Operators](#operators)
- [Regular expressions](#regular-expressions)
- [Byte strings](#byte-strings)
- [Pack and unpack](#pack-and-unpack)
- [JSON](#json)
- [YAML](#yaml)
- [Into Go values: `To` and `Decode`](#into-go-values-to-and-decode)
- [Copies, references and in-place changes](#copies-references-and-in-place-changes)
- [Concurrency](#concurrency)
- [Locking](#locking)
- [Caveats](#caveats)
- [Why Perl?](#why-perl)
- [What it costs](#what-it-costs)

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
`resp.Get("usage", "output_tokens")` is undef if any step is missing. Two
methods supply a default, using Perl's defined-or operators:

```go
tokens := resp.Get("usage", "output_tokens").Or(0).Int()    // ... // 0
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
depthKm := event.At("depth.text").
	Then(parseFigure).   // undef if the text has no figure
	When(isPositive).    // undef unless the figure is above zero
	Then(metresToKm).    // skipped if anything before gave undef
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
| `index($s, "±")` | `s.Index("±")` | first position of `"±"`, or -1 |
| `index($s, "±", 5)` | `s.Index("±", 5)` | first position at or after 5 |
| `rindex($s, "±")` | `s.RIndex("±")` | last position, or -1 |
| `lc($s)`, `uc($s)` | `s.Lc()`, `s.Uc()` | lower or upper case |
| `ucfirst($s)`, `lcfirst($s)` | `s.UcFirst()`, `s.LcFirst()` | change just the first character |
| `fc($s)` | `s.Fc()` | case-fold, for comparing regardless of case |
| `trim($s)` | `s.Trim()` | remove leading and trailing whitespace |

They all return a new value and leave the scalar alone, so they chain:
`s.Trim().Lc()`.

`Index` and `RIndex` count characters, like Perl. Go's own `strings.Index`
counts bytes, which gives the wrong answer for `Substr` as soon as the text
contains `±`, `é` or `•`. The positions from `Index` can go straight into
`Substr`:

```go
s := scalar.Value("Magnitude 5.1 ± 0.2 · Depth 8 km ± 1.5 km")
s.Substr(s.Index("±"), 5)   // "± 0.2"
s.Substr(s.RIndex("±"))     // "± 1.5 km"
```

The case functions use Unicode's full rules, as Perl does, so
`Uc("straße")` is `"STRASSE"`. To compare text regardless of case, use `Fc`,
not `Lc`. Some words have more than one lower-case form, and `Fc` maps them
all to the same thing:

```go
seen := scalar.Hash()
for _, station := range stations.All() {
	seen.Get(station.Fc().Str()).Inc()   // "PAS", "Pas" and "pas" count as one
}
```

`Trim` removes Unicode whitespace too, such as the no-break spaces that HTML
`&nbsp;` turns into.

### Other methods

- `Kind()` tells you what a scalar holds: `Undef`, `Bool`, `Int`, `Float`,
  `String`, `Bytes` (a [byte string](#byte-strings)), `ArrayRef` or
  `HashRef`.
- `Ref()` works like Perl's `ref()`: it returns `"ARRAY"`, `"HASH"` or `""`.
- `Set(x)` is `$s = x`: it assigns a new value and returns the scalar.
- `Get(k)`, `Exists(k)` and `Delete(k)` subscript the array or hash a
  scalar refers to, a string `k` being a hash key and an int an index, and
  `Get(k).Set(x)` assigns to it. See
  [Nesting and autovivification](#nesting-and-autovivification).
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
  id => "ci40123456",
  mag => {
    error => undef,
    value => "5.1",
  },
  stations => [
    "PAS",
    "GSC",
  ],
}
```

Keys are sorted, and quoted only when they need to be. Strings are
double-quoted, with escapes for anything that isn't printable ASCII:
`"caf\x{e9}"` for text, and octal, `"\211PNG"`, for a byte string. A
boolean prints as `!!1` or `!!0`, as in Perl 5.36 and later. An array or
hash that appears twice prints in full the first time, then as its path,
such as `$VAR1->{stations}`, so a structure that loops back on itself is safe.
`TestDumpMatchesPerl` checks hundreds of random structures against Perl's
own `Data::Dumper` with Concise's settings. The one difference is that
there's no newline at the end, since `fmt.Println` adds one.

Arrays and hashes also have `String()`, which gives the same on one line, so
`fmt.Println(a)` shows `["PAS",1,[2,3]]` and `fmt.Println(h)` shows
`{place => "Cobb, CA"}`.

### Reading a dump back: `FromDump`

`scalar.FromDump` reads what `Dump` writes, as `FromJSON` reads JSON, and
returns the structure. The round trip is exact: `FromDump(s.Dump())` is equal
to `s`, and dumps the same, with shared parts and loops rebuilt from their
`$VAR1` paths, which JSON can't carry. It also makes test data quick to
write, in the form you'd write it in Perl:

```go
want := scalar.MustFromDump(`{
	id       => "ci40123456",
	stations => ["PAS", "GSC"],   # comments are fine
}`)
if d := got.Diff(want); d != "" {
	t.Error(d)
}
```

The [Reference](EXAMPLES.md#reference-scalars) has more examples of `FromDump` and
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
fmt.Printf("%d quakes, the largest M%.1f\n", count, mag)   // printf "%d quakes, the largest M%.1f\n", $count, $mag
```

Widths, precisions and flags work as usual. As in Perl, `%x` is numeric:
`"ff"` formats as `0`, not as the hex of its bytes.

### Filling in templates: `Interpolate`

`Interpolate` fills in a template from a hash, as Perl interpolates a
double-quoted string, with the hash's keys as the variables:

```go
doc.Interpolate("M$mag near $place: @stations")   // "M5.1 near Ridgecrest, CA: PAS GSC"
doc.Interpolate("Depth $depth{km} km")            // $doc->{depth}{km}
doc.Interpolate("Depth ${depth.km} km")           // the same: any At path inside ${...}
```

| In the template | Gives |
|---|---|
| `$name` | the value of that key |
| `$name{key}`, `$name[0]`, `$name[0]{key}` | subscripts straight after a name are part of it, as in Perl, written as for `At` |
| `${path}` | any `At` path, dots included: `${depth.km}` |
| `@name`, `@{path}` | an array's elements, joined with a space, as Perl's `"@a"` joins them |
| `\$`, `\@` | a literal `$` or `@`, as in Perl |

As in Perl, `$place.x` is the place followed by `.x`, and `${place}s` puts
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
var summary = scalar.MustTemplate("$place. Stations: @stations.").ListSeparator(", ")

doc.Interpolate(summary)   // "Ridgecrest, CA. Stations: PAS, GSC."
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
a.Get(0, "tags", 1)            // $a[0]{tags}[1]   (a route, not a slice: @a[0, 1] is a.Values(0, 1))
a.Set(5, "x")                  // $a[5] = "x"
a.Length()                     // scalar(@a)
a.Scalar()                     // scalar(@a), same as Length
a.LastIndex()                  // $#a
a.SetLastIndex(2)              // $#a = 2   (truncate or extend)
a.Exists(3)                    // exists $a[3]
a.Exists(0, "id")              // exists $a[0]{id}   (creates $a[0] on the way, as Perl does)
a.Defined(3)                   // defined $a[3]
a.Delete(3)                    // delete $a[3]
a.Delete(0, "tmp")             // delete $a[0]{tmp}  (creates $a[0] on the way, as Perl does)
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
stations := scalar.Array(raw).Grep(notBlank).StrSlice() // back to a []string
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
offshore := func(s *scalar.SV) bool { return s.M(`(?i)\boffshore\b`) != nil }
if places.AnyMatch(offshore) { ... }

mag := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("Magnitude") >= 0 })

stations.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr() // ignoring case

total := felt.Sum0()                                                 // sum0 @felt
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
mag := func(q *scalar.SV) any { return q.At("mag.value") }

quakes.SortByNum(mag)        // nsort_by { $_->{mag}{value} } @quakes
quakes.RevSortByNum(mag)     // rev_nsort_by: the strongest first
quakes.MaxBy(mag)            // max_by: the strongest quake itself
quakes.PartitionBy(func(q *scalar.SV) any { return q.At("net") })   // partition_by
quakes.CountBy(func(q *scalar.SV) any { return q.At("region") }).MostCommon(5)
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
// sort { $a->{net} cmp $b->{net} or $b->{mag} <=> $a->{mag} } @quakes
sorted := quakes.AV().Sort(func(x, y *scalar.SV) int {
	return cmp.Or(
		scalar.StrCmp(x.Get("net"), y.Get("net")), // by network,
		scalar.NumCmp(y.Get("mag"), x.Get("mag")), // then strongest first
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
h.Get("depth", "km")   // $h{depth}{km}     (a route, not a slice: @h{qw(a b)} is h.Values("a", "b"))
h.Set("lang", "Go")    // $h{lang} = "Go"
h.Push("a", 1, "b", 2) // $h{a} = 1; $h{b} = 2   (pairs, another hash or a Go map; later keys win)
h.Exists("age")        // exists $h{age}    (the key is there, even if its value is undef)
h.Exists("depth", "km") // exists $h{depth}{km}  (creates $h{depth} on the way, as Perl does)
h.Defined("age")       // defined $h{age}   (the key is there and its value isn't undef)
h.Delete("age")        // delete $h{age}
h.Delete("depth", "km") // delete $h{depth}{km}  (creates $h{depth} on the way, as Perl does)
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
recorded := scalar.NewSet("PAS", "GSC", "ISA")   // Set::Scalar->new(...)
expected := scalar.NewSet("PAS", "ISA", "SVD")
recorded.Intersection(expected)                  // (ISA PAS): $recorded->intersection($expected)
expected.Difference(recorded)                    // (SVD): expected, but silent
recorded.Has("GSC")                              // $recorded->has("GSC")

var networks scalar.Bag                          // the zero value is empty, ready to use
for _, day := range days {
	networks.Insert(day.Networks)                // count every network across days
}
networks.MostCommon(10)                          // the 10 busiest, most common first
networks.Count("ci")                             // $networks->grab("ci")
```

| Set::Scalar | `Set` | | Set::Bag | `Bag` |
|---|---|---|---|---|
| `new(...)` | `NewSet(...)` | | `new(a => 3)` | `NewBag().InsertN("a", 3)` |
| `insert`, `delete` | `Insert`, `Delete` | | `insert(a => n)`, `delete(a => n)` | `InsertN`, `DeleteN` (or `Insert`, `Delete` for one each) |
| `has`, `size`, `members` | `Has`, `Length`, `Members` | | `grab(a)`, `elements` | `Count`, `Members` |
| `union`, `intersection`, `difference`, `symmetric_difference` | `Union`, `Intersection`, `Difference`, `SymmetricDifference` | | `sum`, `union`, `intersection`, `difference` | `Sum`, `Union`, `Intersection`, `Difference` |
| `is_subset`, `is_proper_subset`, `is_superset`, `is_equal`, `is_disjoint` | `IsSubset`, `IsProperSubset`, `IsSuperset`, `IsEqual`, `IsDisjoint` | | | `Total`, `Counts` (a hash), `MostCommon` |

- **Members are compared as strings,** as Perl's hash keys are, and both
  modules: `1`, `"1"` and `1.0` are one member, while `"1.0"`, `"PAS"`
  and `"pas"` are each different. The first value inserted is the one kept.
  For a case-insensitive set, insert `s.Fc()`.
- **Lists flatten,** as everywhere in `scalar`: `NewSet(array)`,
  `networks.Insert(goSlice)`.
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
Perl. There are two ways to reach into nested data, and which to use
depends on how sure you are of its shape:

- **As Perl does.** `Get`, `Exists` and `Delete` take one key or a
  whole route of them, and `At` a route as one string. They create what
  they go through, as Perl does, and panic where Perl dies. Use them as
  you'd write the Perl.
- **Looking without creating anything.** `GetOK` and `AtOK` read,
  `DeletePath` and `DeleteAt` delete, and `Find` searches. They never
  create anything, and data of the wrong shape just means there's nothing
  there. Use them on data you aren't sure of.

| | As Perl does | Looking without creating anything |
|---|---|---|
| read | `s.Get(k1, k2)`, `s.At("a.b")` | `s.GetOK(k1, k2)`, `s.AtOK("a.b")`, which tell missing from null |
| assign | `s.Get(k1, k2).Set(x)`, `s.At("a.b").Set(x)` | |
| is it there? | `s.Exists(k1, k2)` | the `ok` of `GetOK` or `AtOK` |
| delete | `s.Delete(k1, k2)` | `s.DeletePath(k1, k2)`, `s.DeleteAt("a.b")` |
| search | | `s.Find("$..mag")`: see [Finding values with JSONPath](#finding-values-with-jsonpath) |

In either mode, the route is spelt one of two ways: as keys, each taken
whole, with `Get`, `Exists`, `Delete`, `GetOK` and `DeletePath`, which
suits keys that come from data; or as one string with `At`, `AtOK` and
`DeleteAt`, which suits routes written in the program. Keys can go in one
call, `s.Get("a", 0, "b")`, or a step at a time,
`s.Get("a").Get(0).Get("b")`, which is the same thing.

### What you might assume

This part of `scalar` follows Perl rather than Go, so some of what a Go
programmer, or a Perl programmer reading Go, would expect isn't so:

| You might expect | What happens | For what you expected |
|---|---|---|
| `h.Get("mag", "net")` fetches two keys, like Perl's hash slice `@h{qw(mag net)}` | It's a route, one key inside the other: `$h{mag}{net}`. Several keys to `Get`, `Exists` and `Delete` always mean a route | `h.Values("mag", "net")` is the slice; `a.Values(0, -1)` is `@a[0, -1]` |
| A read never changes the data | Going *through* a missing level creates it, as Perl's autovivification does: `s.Get("origin", "depth")` adds `"origin"` as an empty hash. The last key is never created by a read | `s.GetOK("origin", "depth")` or `s.AtOK("origin.depth")`, which create nothing |
| `Exists` only looks | It creates the levels on the way as `Get` does, an undef `s` becomes a hash or array, and it panics if what it looks in is something else, all as Perl's `exists` does | `_, ok := s.GetOK(...)`, which only looks |
| Deleting something that isn't there does nothing | `Delete` creates and panics as `Exists` does | `s.DeletePath(...)` or `s.DeleteAt(...)`, which do nothing |
| Any number can be an index | Only an `int`. A JSON number is a `float64`, and a `float64`, `int64` or `uint` key panics, as does any type but `string` and `int` | Convert first: `s.Get(int(n.Int()))` |
| `"0"` is index 0 | A string is always a hash key, as `$h{0}` is in Perl | `s.Get(0)` for the index |
| `s.Get()` with no keys is a mistake | It's `s` itself, so a route that might be empty works: `s.Get(route...)`. `Exists()` and `Delete()` with no keys panic | |
| `h.Get(0)` on a hash | Doesn't compile: a hash's first key is a `string` and an array's an `int`. After that, keys are as for a scalar: `h.Get("stations", 0)` | Start from a scalar to choose at run time |

Reading through creates, as in Perl:

```go
doc, _ := scalar.FromJSON([]byte(`{"properties": {"mag": 4.6}}`))
doc.Get("origin", "depth")                 // undef, but doc now has "origin": {}
_, ok := doc.GetOK("products", "shakemap") // false, and doc is unchanged
```

### Reaching in as Perl does

A scalar holding a reference takes subscripts itself, with `Get`,
`Exists` and `Delete`: a string is a hash key and an int an array index,
as `{...}` and `[...]` are in Perl. As Perl needs no arrow between
subscripts, the keys simply follow one another:

```go
s.Get("properties", "place")           // $s->{properties}{place}
s.Get("stations", -1)                  // $s->{stations}[-1]
s.Get("stations", 0, "code")           // $s->{stations}[0]{code}
s.Get("properties", "felt").Set(1204)  // $s->{properties}{felt} = 1204
s.Exists("properties", "felt")         // exists $s->{properties}{felt}
s.Delete("properties", "felt")         // delete $s->{properties}{felt}
s.Get("properties").Get("place")       // a step at a time: the same as the first
```

Arrays, hashes and scalars share four operations on one element:

| Perl | Array | Hash | Scalar holding a reference |
|---|---|---|---|
| read | `a.Get(i)` | `h.Get(k)` | `s.Get(k)` |
| assign | `a.Set(i, x)` | `h.Set(k, x)` | `s.Get(k).Set(x)` |
| `exists` | `a.Exists(i)` | `h.Exists(k)` | `s.Exists(k)` |
| `delete` | `a.Delete(i)` | `h.Delete(k)` | `s.Delete(k)` |

A scalar's `Set(x)` is already `$s = x`, so assigning to an element
subscripts first, as Perl does: `$s->{k} = x` is `$s->{k}`, then `= x`.
`a.Set(i, x)` and `h.Set(k, x)` are the same two steps, `a.Get(i).Set(x)`
and `h.Get(k).Set(x)`, written as one, so all three behave alike:
autovivifying, extending arrays, and panicking where Perl dies.

For what Perl writes with `@{...}` and `%{...}`, the whole array or hash,
`AV()` and `HV()` dereference a scalar and give you the `AV` or `HV`
itself:

```go
h.Get("tags").AV().Push("go", "perl")  // push @{ $h{tags} }, "go", "perl"
s.Get("properties").HV().Keys()        // keys %{ $s->{properties} }
s.Get("stations").AV().Length()        // scalar @{ $s->{stations} }
```

If the scalar is undef, `Get`, `AV()` and `HV()` create a new array or
hash on the spot. This is called autovivification. It lets you build deep
structures in one line:

```go
var s scalar.SV
s.Get("x", "y", 2, "z").Set(1)  // $s->{x}{y}[2]{z} = 1
```

The rules are the same as Perl's:

- Reading a missing key or index gives undef and doesn't create anything.
- Writing to it creates it.
- Going *through* a missing key creates it, even if you are only reading.
  `s.Get("a", "b")` creates `$s->{a}` but not `$s->{a}{b}`.

`Exists` and `Delete` autovivify as Perl's `exists` and `delete` do: an
undef `s` becomes the hash or array they look in, and
`s.Exists("a", "b")` creates `$s->{a}`, though never `$s->{a}{b}`.
Where `s` is something else, such as a number, or a hash given an index,
Perl dies and they panic. To look or delete without either, see
[Looking without creating anything](#looking-without-creating-anything).

Reading a missing key does cost something: it gives a new undef scalar,
ready to create the key if you write to it. To ask only whether a key is
there, as in a loop over many records, `Exists` and `GetOK` cost
nothing: `m.Exists("k")` rather than `m.Get("k").Defined()`, or
`m.GetOK("k")` on records that might not be hashes at all.

`Get` with several keys walks a mix of hash keys (strings) and array
indexes (ints) in turn, as a chain of one-key `Get`s does, and `At` takes
the same route as one string. Both follow the same rules. A route built
from data goes in whole, as `s.Get(route...)`, and `s.Get()` with no keys
is `s` itself. `At` is a method of hashes and arrays as well as scalars,
so `h.At("a.b")` is `$h{a}{b}` and `a.At("[0].name")` is `$a[0]{name}`;
writing through one changes that hash or array:

```go
var doc scalar.SV
doc.Get("x", "y", 2, "z").Set(1)             // $doc->{x}{y}[2]{z} = 1
doc.Get("x").Get("y").Get(2).Get("z").Set(1) // the same, a step at a time
doc.At("x.y[2].z").Set(1)                    // the same
doc.At("{x}{y}[2]{z}").Set(1)                // the same, written as in Perl
```

### A path as one string: `At`

`At` takes the whole route as one string, in dot, bracket or Perl's brace
form, and walks it exactly as `Get` with several keys does:

```go
doc.At("content[0].rule_set")    // $doc->{content}[0]{rule_set}
doc.At("{content}[0]{rule_set}") // the same, written as in Perl
doc.At("content.0.rule_set")     // the same, when content is an array
doc.At("items[-1].name")         // negative indices count from the end
doc.At("meta{log.level}")        // a key with dots in it, in braces
doc.At(`meta["log.level"]`)      // or quoted
doc.At("rules[0].name").Set("x") // writing autovivifies
v, ok := doc.AtOK("depth.km")    // as GetOK: present? without creating anything
doc.DeleteAt("depth.error")      // delete $doc->{depth}{error}
```

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

`Get` never splits strings: `doc.Get("log.level")` is the single
key `log.level`, as it should be, since real keys contain dots.

A path that comes from outside the program, such as a config file or a
flag, can be checked first: `scalar.CheckPath(path)` returns the error
`At` would panic with, or nil.

`At` parses each path once and keeps it, for up to 4,096 different paths,
so a path written in the program costs nothing to parse after its first
use. A route built from data, such as one for each station, is better
built with `Get` than formatted into a string for `At`. Fix the part that
doesn't vary with `At` and add the rest with `Get`:

```go
week := doc.At("week.ci")
week.Get(station, field)  // not doc.At(fmt.Sprintf("week.ci{%s}.%s", station, field))
```

**Compiled paths were measured, then dropped.** `MustPath` compiled an
`At` path once, for `Path` (now `Get`) to take among its keys, as `qr//`
is taken by a match. Once `At` cached its parsing, that no longer paid. On the generated
station table the lock benchmarks use, with an Apple M2, in ns per lookup:

| Lookup | `At` | `MustPath` | `At(prefix).Get` | `Get` chain |
|---|---|---|---|---|
| One path, again and again | 79 | 67 | | 79 |
| 1,000 routes built from data | 116 | 127 | 123 | 101 |
| About 6,600 routes, more than `At` keeps | 435 | 126 | 121 | 101 |

The 12 ns on one path was within the spread of the runs. Past what the
cache keeps, `At` parses every path again, which is the case to avoid,
and a `Get` chain does better than a compiled path there. So `MustPath`,
`CompilePath` and their `PathExpr` were removed in v0.4.0, leaving
`CheckPath` for checking a path. `Path` went too, as `Get` came to take
several keys, and `PathOK` became `GetOK`.

### Looking without creating anything

`GetOK` and `AtOK` read without creating anything, and add Go's comma-ok,
so they also tell a missing key from a JSON `null`, which both read as
undef:

```go
v, ok := event.GetOK("depth", "km")
switch {
case !ok:
	// the feed left the field out (or something on the way is missing)
case !v.Defined():
	// the field is there, but null
default:
	// a real value
}
```

`ok` is false if any step doesn't exist, including stepping through
something that isn't an array or hash. When `ok` is true, `v` is the
stored value itself, as with `Get`. With one key, `s.GetOK(k)` is the
forgiving form of `s.Exists(k)`: it creates nothing, and it doesn't panic
when `s` turns out not to be a hash.

`DeleteAt` removes what a path leads to, and returns its value, or undef if
there was nothing there. `DeletePath` is the same with the route as `Get`
takes it, for keys that come from data and might contain dots or brackets:
`doc.DeletePath(section, key)` is `delete $doc->{$section}{$key}`. As with
`HV.Delete` and `AV.Delete`, an element in the middle of an array leaves a
hole, and deleting the last shrinks the array. Unlike Perl, which creates
the levels on the way to what it deletes, neither creates anything, so
deleting a path that isn't there leaves the data exactly as it was.

`Find` searches with JSONPath, and creates nothing either: see
[Finding values with JSONPath](#finding-values-with-jsonpath).

### What Perl won't create

There are two things Perl refuses to create, and `scalar` refuses them too:

- **An array element before the start of the array**, such as `$a[-10]` on a
  three-element array.
- **An element of something that isn't an array or hash**, such as using a
  string as if it were a hash.

For both, reading gives undef, even if you read further through them with
`Get`, `AV()` or `HV()`. Writing panics, just as Perl dies.

Strict Perl dies on some of these *reads* as well. `scalar` lets reads
carry on, because a read that can't panic is what makes `Get` and `At`
safe on messy data. `Exists` and `Delete` are the exception: they say
which kind of container they expect, so they die where Perl does.

## Finding values with JSONPath

`Get` and `At` read one value at a location you know. `Find` searches with a
JSONPath query ([RFC 9535](https://www.rfc-editor.org/rfc/rfc9535)) and
returns every match as an array, or `nil` if nothing matched:

```go
feed.Find("$.features[*].properties.place") // every quake's place
feed.Find("$..stations[*]")       // every station, at any depth
doc.Find("$.items[-1].name")      // the last item's name
doc.Find("$.items[0:3]")          // the first three items
doc.Find("$['mag-type']")         // keys that aren't plain words need brackets
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

Arrays and hashes have `Find` too, with `$` standing for the array or
hash itself, so `h.Find("$.ci[*]")` needs no scalar around `h`.

Each match is a scalar. Strings, numbers, booleans and nulls come back as plain
values. Objects and arrays come back as references, so you can continue from
them with `Get`, `HV()`, `AV()` or `Find`.

Like `Get` and `At`, the matches are the stored values themselves, so you
can change data where it sits:

```go
for _, s := range config.Find("$.stations[*]").All() {
	s.Set(s.Trim())   // trims each station code inside config
}
```

### Filters

A filter, `[?...]`, selects the children of an array or hash for which an
expression is true. In it, `@` is the child, and `$` the value `Find` was
called on:

```go
quakes.Find("$[?@.mag >= 4.5].place")                     // a comparison
quakes.Find("$[?@.felt]")                                  // has felt reports
quakes.Find("$[?!@.tsunami]")                              // hasn't "tsunami" at all
quakes.Find("$[?@.mag > $.threshold]")                     // against the root
quakes.Find("$[?@.net == 'ci' && @.depth <= 5]")           // && || ! and ( )
quakes.Find("$[?count(@.stations[*]) > 2]")                // functions
feed.Find("$.features[?@.properties.alert == 'red'].id")  // events with a red alert
```

| In a filter | Means |
|---|---|
| `@.a`, `$.a` on their own | the query selects something: `@.tsunami` is true even if it's `false` |
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
doc.Find("$.regions[?@.name == " + scalar.QuoteJSONPath(name) + "].stations[*]")   // name: "Hawai'i"
doc.Find("$[" + scalar.QuoteJSONPath(key) + "]")   // a key that isn't a plain word
```

For a condition the language can't say, the list utilities on the result are
plain Go:

```go
event := feed.Find("$.features[*]").FirstMatch(func(f *scalar.SV) bool {
	return f.At("properties.place").M(`(?i)\boffshore\b`) != nil && f.At("properties.felt").Defined()
}).Get("properties")
```

### Things to know

- **`Find` never creates anything.** Unlike `Get` and `At`, a missing key or index
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
doc.Equal(map[string]any{"stations": []string{"PAS", "GSC"}})   // is_deeply

ours.Diff(theirs)    // stations[1]: got "GSC", want "ISA"
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
reported. That makes it handy in tests, and for comparing two catalogues'
versions of the same event.

When the answer is all you need, `Equal` is the cheaper call: it builds no
report and stops at the first difference. On two copies of a table of
15,000 rows, it takes 28 ms where `Diff` takes 123 ms.

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
for _, q := range quakes {
	totals.Get(q.Region).AddInPlace(q.Felt)   // $totals{$region} += $felt
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
so patterns built from data, such as one for each station name, can't make it
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
var mag = scalar.MustCompile(`M(\d\.\d)`) // my $mag = qr/M(\d\.\d)/;

for m := range s.Matches(mag) { ... }
s.S(mag, "M?", "g")
```

Declare the patterns your program relies on as package-level variables, as
`mag` is above. If one of them is invalid, the program then panics as soon
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
`scalar.QuoteMeta`, Perl's `quotemeta` (`\Q...\E`). Labels in data are
often full of characters that mean something in a regex: without it,
`s.M("(USGS")` panics with an invalid pattern, and `s.M("M4.5")` also
matches `"M405"`.

```go
for _, label := range []string{"M4.5+", "(USGS)", "Mw 6.0"} {
	if bulletin.M(`(?i)` + scalar.QuoteMeta(label)) != nil { ... }
}
```

`QuoteMeta` takes a Go string or a scalar, and returns a Go string, ready to
add to a pattern.

### Matching

```go
s := scalar.Value("Magnitude range: M4.8 - M5.3")

if s.M(`(?i)magnitude`) != nil { ... }

m := s.M(`M(\d\.\d)\s*-\s*M(\d\.\d)`)
m.Get(0) // "M4.8 - M5.3"   ($&)
m.Get(1) // "4.8"           ($1)
m.Get(2) // "5.3"           ($2)
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
for m := range s.Matches(`M(\d\.\d)`) {
	fmt.Println(m.Get(1))   // 4.8 then 5.3
}
```

With `g`, `M` finds every match at once, and returns an array with one
element per match: a reference to that match's array, as Perl's arrays of
arrays hold references. Reach into it with a path, `all.At("[1][1]")`, or
by subscripting, `all.Get(1).Get(1)`. If nothing matches, it returns
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
`^(a+)+$`, can take a very long time on some inputs. To stop one bad record
from hanging the whole program, every match has a time limit
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
stations, err := scalar.To[[]string](doc.At("stations"))   // a []string
km, err := scalar.To[int](doc.At("depth.km"))               // an int, even from "8"
event, err := scalar.To[Event](doc)                         // your own struct
counts, err := scalar.To[map[string]int](bag)               // a bag's counts
```

`s.Decode(&v)` does the same into a value you already have. Both follow the
same rules, set out below. Use `To` for a new value, and `Decode` to fill in
one that's partly set already (see [Decode](#decode-filling-in-a-value-you-have)).

With a struct, `To` takes you from loosely shaped data to a typed value,
which [What it costs](#what-it-costs) recommends doing as early as you can:

```go
type Event struct {
	Place    string   `json:"place"`
	Stations []string `json:"stations"`
	Depth    struct {
		Km    int `json:"km"`      // "8" fills it, as Perl would numify it
		Error int `json:"error"`
	} `json:"depth"`
	Extra *scalar.SV `json:"extra"` // a part whose shape varies stays a scalar
}

e, err := scalar.To[Event](doc)
if err != nil {
	return err   // scalar: Decode: depth.km: want int, got "unknown"
}
fmt.Println(e.Place, e.Depth.Km)
```

`To` is a function rather than a method, `scalar.To[T](x)` rather than
`x.To[T]()`, because Go's methods can't take a type in brackets.

### What can go in the brackets

| Type | Takes | Example |
|---|---|---|
| a struct | a hash: each key fills the field with that name (see below) | `scalar.To[Event](doc)` |
| a slice, `[]T`, or an array, `[N]T` | an array, or a set's members | `scalar.To[[]string](doc.At("stations"))` |
| a map, `map[string]T` or `map[int]T` | a hash, or a bag's counts | `scalar.To[map[string]int](bag)` |
| `string` | any plain value, as its string | `scalar.To[string](doc.At("place"))` |
| `int`, `int64`, `uint`, … | a whole number, or a string holding one, that fits | `scalar.To[int](doc.At("depth.km"))` |
| `float64`, `float32` | a number, or a string holding one | `scalar.To[float64](doc.At("mag"))` |
| `bool` | a boolean, `0` or `1`, or `"true"` or `"false"` | `scalar.To[bool](doc.At("tsunami"))` |
| a pointer, `*T` | what `T` takes; undef gives nil | `scalar.To[*int](doc.At("felt"))` |
| `any` | anything, as plain Go values, as `Any()` gives them | `scalar.To[any](doc)` |
| `*scalar.SV`, `*scalar.AV`, `*scalar.HV` | a deep copy (see `Clone`) of whatever is there | a struct field for a part that stays loose |
| a type with `UnmarshalJSON` or `UnmarshalText`, such as `time.Time` | whatever it accepts | `scalar.To[time.Time](doc.At("posted"))` |
| `[]byte` | a byte string's bytes, or text in base64, as `encoding/json` takes it | `scalar.To[[]byte](doc.At("samples"))` |

These nest as you'd expect: `[]Event` takes an array of hashes,
`map[string][]string` a hash of arrays. The source can be a scalar holding an
array or hash, or the array or hash itself: `scalar.To[[]string](a)` works
for an `*AV`, as does `scalar.To[[]string]([]any{"PAS", "GSC"})` for a Go value.

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
felt, err := scalar.To[*int](doc.At("felt"))           // nil if no one reported it, or it's null
magErr, err := scalar.To[*float64](doc.At("error"))    // nil, or a pointer to the number
magType, err := scalar.To[*string](doc.At("magType"))
```

There's no `StrPtr` or `NumPtr` method, because `To[*T]` does this for any
type, and inside structs too, so a struct with `*string` and `*float64`
fields (as a feed's optional fields often are) fills in one call. It's also
checked: `To[*float64]` refuses `"unknown"` with an error, where `Num` would make
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
e, err := scalar.To[Event](doc)
// scalar: Decode: depth.km: want int, got "unknown"

var de *scalar.DecodeError
if errors.As(err, &de) {
	log.Printf("%s is %s", de.Path, doc.At(de.Path).Dump())   // depth.km is "unknown"
}
```

Paths look like `depth.km`, `stations[2]` or `meta{log.level}`, the last for
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
e := Event{Place: "unknown"}             // the default
if err := doc.Decode(&e); err != nil {   // a place in doc replaces it; none leaves it
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
doc.At("origin").Set(Origin{Lat: 35.7, Lon: -117.6})   // a struct, as a hash
```

Perl has no structs, so this goes beyond it, as `Decode` does, for Go's
sake.

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

`a.Get(i)`, `h.Get(k)`, `s.Get(k, ...)`, `At(...)`, `All()`, the
matches from `Find` and the elements of `Values()` give you the value that's
actually stored, not a copy. Changing it changes the array or hash:

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
| `At("a.b")`, `Get(k1, k2)` on a missing route | the levels on the way, as Perl's autovivification does: reading `x.y` of `{"a": 1}` adds `x` |
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
	e, err := scalar.To[Event](doc) // the shape is known: into Go
	results[i] = e                  // a plain struct, safe to share
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
doc.At("depth.km")              // reading what's there is as usual
doc.At("plcae")                 // panics: Attempt to access disallowed key 'plcae' in a restricted hash
doc.At("depth.error").Set(0.5)  // panics: as it would add a key
doc.At("depth.km").Set(1)       // panics: Modification of a read-only value attempted
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
on a table of 15,000 rows of strings: at first the ONS's ASHE pay table,
and now a generated table of seismic statistics in the same shape, which
anyone can run and which gives the same results (`lock_bench_test.go`).
The profiles found two costs that every value paid, locked or not, and
those were fixed for everyone instead:

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
  - a malformed path in `At`, `AtOK` or `DeleteAt` (use `CheckPath` to get
    an error instead), or a malformed template in
    `Interpolate` or `MustTemplate` (use `CompileTemplate`), or a template
    that's neither a string nor a `*Template`
  - passing something other than a string or int to `GetOK`,
    `DeletePath`, or a scalar's `Get`, `Exists` or `Delete`, or something other than a string or function as an `S`
    replacement
  - a scalar's `Exists` or `Delete` on something other than the hash (for
    a string key) or array (for an int) it refers to, or an undef that
    can create one
  - `Any()` on a reference that leads back to itself (encoding it as JSON
    returns an error instead)

  If one bad record mustn't stop a batch, catch these with `recover` around
  the code that processes each record.
- **Not for concurrent use, not even reading:** none of these types are safe
  to share between goroutines without your own locking, as some reads write,
  unless the structure is [locked](#locking). See [Concurrency](#concurrency).
- **No compile-time checks:** a misspelled key won't fail to compile. It just
  reads as undef. Keep these types at the edges of your program, and use
  structs for anything you depend on.

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

## What it costs

Dynamic values give up compile-time checking. A misspelled key compiles
without complaint and silently reads as undef. That's why the rule for this
package is:

> **Use `scalar` at the edges, where data comes in. Convert to ordinary Go
> structs as soon as you know the shape, and keep the core of the program
> typed.**

`To` is that conversion, in one step: `scalar.To[Event](doc)` gives you an
`Event`, or an error saying which value didn't fit. See
[Into Go values](#into-go-values-to-and-decode).

It's also what makes concurrent code safe. A struct of plain values can be
shared between goroutines for reading; a scalar can't, as some of its
reads write. See [Concurrency](#concurrency).
