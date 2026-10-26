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

### 3. JSON decimals that keep their digits

An experiment. A number read as text, from JSON, YAML or a dump, keeps
its text beside its float when the float can't give the number back: a
decimal such as `1.23456789012345678901`, or a whole number beyond
`int64`. It's on for every number, with no option to ask for it, and
arithmetic stays on the float; money stays with plain Go at the edges.
It's being built and measured on the `experiment/exact-decimals` branch,
off develop; the
[change record](docs/changes/2026-10-26-exact-decimals.md) has the plan
and what's happened.

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
