# To do

## Decide whether lazy values stay (roadmap 5)

**From:** scalar's session, 2026-11-02 (scalar a97896c011e1244a5e478dfd2b2cadd6722002bd)
**Waiting on:** nothing: grammar has measured both ways (below)

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

**grammar's answer** (grammar f827a363542da63d9712a8ab24b85614092d99e8):
grammar kept the named way, and doesn't need `Lazy`. Interleaved, 10
runs each, all on `r5-lazy-values`, against eager captures on the same
scalar: lazy captures −12.7% time, −8.7% bytes, −9.8% allocations;
named reads −23.4%, −21.9%, −27.8%. On scalar v0.6.3 as released, named
is −24.2% (1.88 ms to 1.43 ms). Named reads also cover parses with
actions, where the lazy prototype didn't, and jdc-lab's chunker, naming
its two assertions' reads, gave identical output over 28,000 ads about
2% faster. Lazy captures are reverted in grammar, with their numbers in
its record,
`docs/changes/2026-11-02-r24-building-only-what-a-code-assertion-reads.md`,
and its reference's Engineering gaps. With no other user waiting on it,
grammar's numbers point to dropping roadmap 5; the decision is Iain's.

**Steps:**

1. Read grammar's change record for its roadmap 24.
2. Put both choices to Iain, with these numbers and grammar's.
3. To keep it: rebase `r5-lazy-values` on develop, open the change
   record `docs/changes/2026-11-02-r5-lazy-values.md` with the roadmap's
   plan word for word, and add the reference section, a README row and
   a CHANGELOG line. To drop it: open the record as dropped, saying why,
   put the numbers in the reference's reverted ideas, remove item 5 from
   the roadmap, and delete the branch.
4. Either way, delete grammar's local branch `r24-lazy-sofar`, the lazy
   prototype grammar measured and reverted: it builds only against
   `r5-lazy-values`, and grammar's record has its numbers. Check it's
   still at 6e74c9dd16a5627cd89b37373e5cb1e5c1f46b50 and not pushed
   (`git -C /Users/icampbell/Git/github.com/nukopian/grammar branch -r --contains r24-lazy-sofar`
   prints nothing), then
   `git -C /Users/icampbell/Git/github.com/nukopian/grammar branch -D r24-lazy-sofar`.
   If item 5 is kept, its record should name the branch's last commit
   first, in case anyone wants grammar's side again.

**Done when** item 5 has shipped or been dropped, with its record, and
grammar's `r24-lazy-sofar` is gone.
