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
number: 5.

Each item has a size too, a guess at how much work it is, as a T-shirt
size and its story points: XS 1, S 2, M 3 and L 5. About a day is S, a
few days M, and a week or two L. An item bigger than L is broken into
items that aren't, each saying what it was split from. Items that can't
be done one without another say so ("Moves with:"), and move between
groups together.

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

An idea is a numbered heading, the date it was added, its size, and a
line or two on what and why. Each wash-up prunes them: each is kept,
promoted to Later or Next with its number, or dropped.

### 1. A regex engine chosen per pattern

Added 2026-10-26. **Size:** M (3). Being considered, not decided.
`CompileWith(scalar.Stdlib, pattern)` would match with Go's regexp, 1.2
to 2.6 times as fast in the experiment's benchmarks and never
backtracking, at the cost of lookaround, backreferences and Unicode
`\w`; other engines, such as coregex, could plug in the same way. The
experiment, with its tests and measurements, is on the
[`experiment/regex-engine`](https://github.com/nukopian/scalar/tree/experiment/regex-engine)
branch.

### 4. Profile scalar's allocations under a consumer's workload

Added 2026-11-01. **Size:** M (3). Run a program that uses scalar,
such as grammar's matches or jdc-lab's pipeline, profile it, and if
scalar shows up, find its cost and cut it, as grammar's roadmap 9 did
for its matcher.

That item shows the method works. jdc-lab found the cost: chunking its
job ads was its slowest stage, and most of that time was the matcher
allocating. grammar then added a benchmark shaped like the lab's use
(`FindAll` over many short lines), profiled it (`mallocgc` was 60% of
the CPU), and counted calls with temporary counters to find the one
site making most of the allocations. Reusing per-match scratch space
behind a fast-path switch cut the lab's allocations by 74% and its time
by 39%, with every output unchanged
([record](https://github.com/nukopian/grammar/blob/v0.4.3/docs/changes/2026-10-31-r9-fewer-closures-in-the-matcher.md)).
grammar's own `FindAll` benchmark improved by only 7%: the lab's
workload found what grammar's benchmarks missed. That fix doesn't carry
over, as scalar has no matcher of its own (it uses regexp2), but the
method does.

A brief run of scalar's benchmarks with `-benchmem` (Apple M2, Go
1.26.10, at c28d87cfa6e6a758d0c93597c84bc8b3c7565474) suggests where to
look. None of these has been profiled, and no consumer has yet reported
scalar as a cost:

| Benchmark | Allocations | Possible cause, unchecked |
|---|---:|---|
| `SortNum` | 2,001 for 1,000 elements | about two per element, perhaps a conversion |
| `AtUnlocked` | 6 per lookup, none when locked | perhaps the path parsed again on every call |
| `AddInt`, `MulInt` and the like | 2 per operation, 56 B, about 33 ns | the result and one more |
| `SExpand/g/long` | about 2,400 | the result built anew for each match |

The reference's "Does locking make reads faster?" already measured that
caching `At`'s lookups wouldn't pay, so a consumer's profile, not these
figures, should decide what's worth doing. One place to start: grammar
makes new state for every text `FindAll` searches, and builds scalar
values for each match, so the lab's remaining cost may include
scalar's.
