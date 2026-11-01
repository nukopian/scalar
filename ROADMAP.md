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
size and its story points: XS 1, S 2, M 3 and L 5. Sizes compare items
with each other; they aren't a measure of time. An item bigger than L
is broken into items that aren't, each saying what it was split from.
Items that can't be done one without another say so ("Moves with:"),
and move between groups together.

When an item ships, it moves from here to the CHANGELOG, whose line
for it ends with its number, such as "(roadmap 1)", and links to its
record. An idea that's dropped is deleted, and the commit that drops
it names its number.

## Now

### 1. A regex engine chosen per pattern

**Serves:** speed

**Size:** L (5)

`scalar.Linear`, Go's regexp, beside `scalar.Backtracking`, regexp2 and
the default, chosen for one pattern with `MustCompileWith` or for the
whole program with `DefaultEngine`. Nothing changes unless a program
asks. `Linear` never gives a different answer from `Backtracking`, only
a faster one: a pattern it can't run is an error when chosen for one
pattern, and falls back with a warning when chosen for the program, and
a match on text where the engines could differ goes to `Backtracking`.
The [change record](docs/changes/2026-11-01-r1-regex-engines.md) has the
plan and what's happened.

## Next

Nothing yet.

## Later

Nothing yet.

## Ideas

An idea is a numbered heading, the date it was added, its size, and a
line or two on what and why. Each wash-up prunes them: each is kept,
promoted to Later or Next with its number, or dropped.

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

**First profile, 2026-11-01:** scalar isn't a cost yet. jdc-lab's
`BenchmarkAdOfSamples`, chunking 2,000 ads (Apple M2, Go 1.26.10,
`-benchtime 3x`; jdc-lab 199ef5b42ff8679f2c6033167c8abcc4a443cd79
through its `go.work`, with scalar
c18c75e434dfea89e2bb79bdfc537a2ad49314ff and grammar
690b8ee3bd0939ca640a802b393b90139554354b, which has v0.4.3's matcher
changes), took 1.38 s, 1.30 GB and 24.5 million allocations an
iteration. Scalar's share:

| Measure | Scalar | Mostly |
|---|---:|---|
| CPU | 2.3% | allocation, through grammar's `FindAll` |
| bytes allocated | 5.3% | `Hash`, 3.5%: grammar builds a hash for each match |
| allocations | 5.5% | `Hash` and `svOf` |

grammar's matcher (`once`, `seq.from`, `step`) is still about a
quarter of the CPU. None of the benchmarks in the first table shows
up, so scalar's own figures would have pointed the wrong way. Removing
every one of scalar's allocations would save at most about 5% of the
lab's, so nothing is worth cutting yet. As grammar's matcher gets
cheaper, scalar's share will grow: profile again after grammar's next
matcher change, starting with `Hash` as grammar calls it.
