# Roadmap

What scalar means to do, and why. The [CHANGELOG](CHANGELOG.md) says
what it has done, and the [README](README.md) what it is now.

Items come in four groups, the most useful first in each:

- **Now:** being worked on, each with a plan detailed enough to pick up
  cold, in its [change record](docs/changes/README.md); never more than
  three, so that it stays a queue;
- **Next:** decided, but not started: what, why, and when it's done;
- **Later:** meant to be done, but not yet planned, in a line or two;
- **Ideas:** might be done, or never; kept so they aren't lost, in a
  line or two, newest last.

Each item has a number, given when it's added, in whichever group. It
keeps it as it moves between groups, and a number is never used again,
so that 1 means the same item in a commit message months later. Next
number: 4.

When an item ships, it moves from here to the CHANGELOG, whose line
for it ends with its number, such as "(roadmap 1)", and links to its
record. An idea that's dropped is deleted, and the commit that drops
it names its number.

## Now

Nothing yet.

## Next

### 2. JSON whole numbers that keep their digits

**What:** a JSON whole number too big for an `int64`, such as the id
`12345678901234567890`, keeps its digits as text beside the `float64`
it becomes, so `Str()`, `MarshalJSON`, `Dump`, YAML output and
`scalar.To` into an exact type, such as shopspring/decimal's, give
every digit, while `Num()`, arithmetic and `Any()` still see the float.
Go `uint64` values above `int64`, which become floats too, keep theirs
the same way. Decimals stay as they are, as in Perl; idea 3 is keeping
theirs.

**Why:** big ids are common in API JSON, and today they lose their last
digits silently: `12345678901234567890` is written back as
`12345678901234567000`, and `To` can't win them back. Perl keeps them:
JSON::PP reads such a number as a plain string with every digit, and
Perl adds the float beside it when the number is first used as a
number, so both mean the same number (Perl's ordinary storage, not
Scalar::Util's `dualvar`, whose two can disagree). A number up to
2⁶⁴−1 is an exact UV in Perl, with exact arithmetic; scalar's arithmetic
there stays float, a deviation to document.

**How, as far as it's known:** the text goes in the scalar's `rare`
field, so the float stays in `v` and the code that switches on
`float64` is unchanged. Both JSON decoders end in
`normalizeJSONNumber`, and the fast path's integers of up to 18 digits
never reach it, so the common case costs nothing. Every write passes
through `vivify()`, which clears the text, so a changed value never
prints the old digits. `Copy` and `Clone` carry it.

**Done when** whole numbers that don't fit an `int64` round-trip through
`FromJSON` and `MarshalJSON` with every digit; a differential test
compares them with JSON::PP; a test shows that every mutator clears the
text; the JSON benchmarks show no real cost; the fast-paths comparison
and `FuzzReadJSON` cover it; and the reference's JSON section and
Caveats say what's kept, and what isn't: decimals, and exact arithmetic
above `int64`.

## Later

Nothing yet.

## Ideas

An idea is a numbered heading, the date it was added, and a line or two
on what and why. Each wash-up prunes them: each is kept, promoted to
Later or Next with its number, or dropped.

### 1. A regex engine chosen per pattern

Added 2026-10-26. Being considered, not decided.
`CompileWith(scalar.Stdlib, pattern)` would match with Go's regexp, 1.2
to 2.6 times as fast in the experiment's benchmarks and never
backtracking, at the cost of lookaround, backreferences and Unicode
`\w`; other engines, such as coregex, could plug in the same way. The
experiment, with its tests and measurements, is on the
[`experiment/regex-engine`](https://github.com/nukopian/scalar/tree/experiment/regex-engine)
branch.

### 3. JSON decimals that keep their digits

Added 2026-10-26. Item 2 done for decimals too: a JSON decimal with
more digits than a `float64` holds, such as `1.23456789012345678901`,
keeps its text beside the float, the same way, so money and
measurements survive a round trip and `scalar.To` into an exact decimal
type. Perl doesn't do this by default: JSON::PP reads such a number as a
float, `1.23456789012346`, unless `allow_bignum` makes it a
Math::BigFloat. So it would be a deviation from Perl, documented in the
reference, or an option like `allow_bignum`. Waits on item 2, which
builds the storage it would reuse.
