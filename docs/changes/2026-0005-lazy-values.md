# 2026-0005. Lazy values

**Status:** dropped

**Origin:** roadmap 2026-0005

**Started:** 2026-11-02, in a97896c011e1244a5e478dfd2b2cadd6722002bd

**Finished:** 2026-11-02

## The plan

The plan as it was when work started, from the roadmap's Next, word for
word.

**Size:** M (3)

**What:** a value built by a function the first time it's read, so a
structure built for a caller who reads little of it costs only what's
read. Every way of reading a value resolves it first: `Lock`, `Clone`,
`Equal`, `Dump`, JSON and YAML included.

**Why:** grammar builds the whole match so far for each code
assertion, a hash for every capture, though the function may read one
or two (grammar's roadmap 24). In grammar's `BenchmarkAssert`, that's
39% of a search's allocations.

**Done when** a lazy value reads, locks, clones, compares and encodes
as the value it builds, with a test of each; it's built once, however
it's read, and safely from several goroutines; and grammar's
`BenchmarkAssert`, with grammar building its match so far lazily,
shows what it saves.

## What happened

- **Outcome:** dropped. `Lazy` was built and worked: it met Done when,
  apart from shipping. But grammar, its one user, did better without
  it, by naming the captures an assertion reads (grammar's roadmap 24),
  and kept that way. With no other user, a feature that touches every
  read in scalar isn't worth its cost.
- **What was built:** `Lazy(f func() any) *SV`, after CPAN's
  Scalar::Defer `lazy { ... }`: built once, on the first read, however
  it's read (`Clone`, `Equal`, `Dump`, JSON, YAML, `Find` and `Any`
  included), and safely from several goroutines, with `sync.Once`. If
  `f` panicked, every read panicked. Tests covered each way of reading,
  building once from 8 goroutines under `-race`, numbers keeping their
  digits, and the panic.
- **Changes from the plan:** `Lock` and `Autovivify` didn't build the
  value: what they set reached it when it was built. Grammar locks the
  match before the assertion reads it, so a `Lock` that builds would
  have saved nothing. Copying and storing it didn't build it either: a
  copy shared what the original built, as a copied reference shares
  what it refers to.
- **Two designs were tried.** The first held the lazy value in the
  scalar's `rare` field. `val()`, which every read goes through, then
  needed a second load and no longer inlined, and numeric compares and
  sorts were 5–7% slower; and every copy made a new lazy value and
  closure, which ate most of what grammar's lazy captures saved. The
  second held it as the scalar's value, a `*lazyVal` in `v`: `val()`
  needed one type check, and a copy was one scalar. Against develop,
  8 runs each, interleaved, scalar's benchmarks then moved −5.5% to
  +4.2%, +0.5% geomean (`SumMag` +4.2%, `Clone` +3.6%, `NumLtInt`
  −5.2%).
- **Measurements in grammar:** `BenchmarkAssert`, interleaved, 10 runs
  each, all on the second design, against eager captures on the same
  scalar (grammar's numbers, from its
  [record](https://github.com/nukopian/grammar/blob/develop/docs/changes/2026-0024-building-only-what-a-code-assertion-reads.md)):

  | `BenchmarkAssert` | time | bytes | allocations |
  |---|---|---|---|
  | eager, every capture | 1.879 ms | 2.401 MiB | 44.25k |
  | lazy captures | 1.639 ms, −12.7% | 2.192 MiB, −8.7% | 39.92k, −9.8% |
  | named reads | 1.439 ms, −23.4% | 1.875 MiB, −21.9% | 31.95k, −27.8% |

  Each lazy capture cost a few allocations of its own, the scalar, its
  lazy value and its closure, nearly what a small capture costs to
  build; naming the reads builds nothing for the rest. Named reads also
  cover parses with actions, which the lazy prototype didn't.
- **Why it isn't kept for other uses:** scalar's data mostly arrives
  built, from JSON, YAML and text, with nothing to defer; every read in
  scalar would have to go through `val()`, or see the `*lazyVal`, a
  mistake only tests would catch; and plain Go has `sync.OnceValue`. The
  reference's [What it costs](../REFERENCE.md#what-it-costs) says so.
- **What's left:** nothing. The prototypes were on local branches, never
  pushed, and are deleted: scalar's `r5-lazy-values`, last at
  c0719949675e824b08aa29b2abe2cfeda1e5afb4, and grammar's
  `r24-lazy-sofar`, at 6e74c9dd16a5627cd89b37373e5cb1e5c1f46b50.
