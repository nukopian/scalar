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
number: 3.

When an item ships, it moves from here to the CHANGELOG, whose line
for it ends with its number, such as "(roadmap 1)", and links to its
record. An idea that's dropped is deleted, and the commit that drops
it names its number.

## Now

Nothing yet.

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

### 2. JSON numbers that keep their digits

Added 2026-10-26. When a JSON number can't be held exactly, an
`int64` or `float64`, keep its text beside the float, so `Str()`,
`MarshalJSON` and `scalar.To` into an exact type, such as
shopspring/decimal's, give every digit, while `Num()` still gives the
float. Big ids are common in API JSON, and today they lose digits
silently, where Perl keeps a whole number up to 2⁶⁴−1 exact and JSON::PP
keeps the digits of a bigger one. It changes behaviour, so it would be
measured for cost and tested against Perl.
