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

### 2. JSON whole numbers that keep their digits

An investigation, not decided, and it may be dropped. Experiments on
two branches show what keeping the digits of a whole number from 2⁶³ to
2⁶⁴−1 would cost, and a build holds one as a `uint64`, as Perl holds a
UV. Whether it lands depends on idea 3, keeping decimals' digits, which
is the outcome that matters most: a design for whole numbers alone could
leave decimals needing a second mechanism. The
[change record](docs/changes/2026-10-26-exact-whole-numbers.md) has the
plan and what's happened.

## Next

Nothing yet.

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

Added 2026-10-26. A JSON decimal with more digits than a `float64`
holds, such as `1.23456789012345678901`, keeps its text beside the
float, so money and measurements survive a round trip and `scalar.To`
into an exact decimal type. Perl doesn't do this by default: JSON::PP
reads such a number as a float, `1.23456789012346`, unless
`allow_bignum` makes it a Math::BigFloat. So it would be a deviation
from Perl, documented in the reference, or an option like
`allow_bignum`. Item 2's build holds whole numbers as a `uint64`, so this
would need storage of its own; the
[`experiment/exact-integers`](https://github.com/nukopian/scalar/tree/experiment/exact-integers)
branch built and measured text beside the float, for whole numbers only:
decimals weren't tried.
