# To do

## Decide whether lazy values stay (roadmap 5)

**From:** scalar's session, 2026-11-02 (scalar a97896c011e1244a5e478dfd2b2cadd6722002bd)
**Waiting on:** grammar's roadmap 24, measured both ways

**What:** keep `scalar.Lazy` and ship roadmap 5, or drop it, once
grammar has measured its roadmap 24 both ways: lazy captures, using
`Lazy`, and an `Assertion` that names the captures it reads.

**Why it waits:** grammar was the reason for 5, and an experiment found
the other way pays more there. Without a user that needs it, `Lazy`
costs every program a little (see below) and makes every read in scalar
go through `val()`, so the decision is Iain's once grammar's numbers
are in.

**What's known:** a prototype is on the local branch `r5-lazy-values`
(not pushed), at c0719949675e824b08aa29b2abe2cfeda1e5afb4.

- `Lazy(f func() any) *SV`, after CPAN's Scalar::Defer `lazy { ... }`:
  built once, on the first read, however it's read (Clone, Equal, Dump,
  JSON, YAML, Find and Any included), and safely from several goroutines.
  Copying, storing, `Lock` and `Autovivify` don't build it; what they set
  reaches the value when it's built. A copy shares what it builds, as a
  copied reference does. This changes the roadmap's plan, where `Lock`
  builds: grammar locks the match before the assertion reads it, so that
  would save nothing.
- The lazy value is held as the scalar's value (`*lazyVal` in `s.v`), so
  `val()` needs one type check and still inlines. An earlier commit on
  the branch, 1497b19089987e9efaba1eeedf81b23ddf4a98e0, held it in
  `svRare`: that cost 5–7% on numeric compares and sorts, and made every
  copy allocate a new lazy value and closure.
- Tests in `lazy_test.go` and `ExampleLazy`; the full suite passes with
  `-race` and Perl 5.42.3, nothing skipped.
- scalar's benchmarks, against develop (8 runs each, interleaved):
  −5.5% to +4.2%, +0.5% geomean; SumMag +4.2%, Clone +3.6%,
  EqualAlikeUnlocked +3.3%, NumLtInt −5.2%.
- grammar's `BenchmarkAssert`, against eager captures on the same
  scalar: lazy captures −13% time, −9% bytes, −10% allocations; named
  captures, a quick estimate, −21% time, −17% bytes, −21% allocations.
  `sofar` made 45% of the allocations, the ceiling.

**Steps:**

1. Read grammar's change record for its roadmap 24.
2. Put both choices to Iain, with these numbers and grammar's.
3. To keep it: rebase `r5-lazy-values` on develop, open the change
   record `docs/changes/2026-11-02-r5-lazy-values.md` with the roadmap's
   plan word for word, and add the reference section, a README row and
   a CHANGELOG line. To drop it: open the record as dropped, saying why,
   put the numbers in the reference's reverted ideas, remove item 5 from
   the roadmap, and delete the branch.

**Done when** item 5 has shipped or been dropped, with its record.
