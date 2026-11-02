# To do

## Close the record of Hash's values in one allocation

**From:** grammar's session, 2026-11-02 (grammar f29165bc1895eb5f2ba5bb19d7733c80e6a9cc91)
**Waiting on:** nothing

**What:** the last check in
`docs/changes/2026-11-02-hash-values-in-one-allocation.md` is done:
grammar now requires scalar v0.6.4, and its `BenchmarkParse` shows the
gain against the released scalar. Add it to the record's What happened,
and change the Outcome from "done, but for its last check" to done.

**Why:** the record's Done when names this check, and its What's left
waits on it.

**The numbers:** grammar at f29165bc1895eb5f2ba5bb19d7733c80e6a9cc91,
with `GOWORK=off`, scalar v0.6.3 against v0.6.4, 8 runs each,
interleaved, on an Apple M2: `BenchmarkParse` 5.78 µs to 5.02 µs
(−13%), 126 to 86 allocations (−32%); `BenchmarkFindLines` −4.7%,
`BenchmarkFindAll` −10%, `BenchmarkAssert` −8%, with 21–27% fewer
allocations in each. Every grammar test passed, Raku's included. In
job-meta's Participle comparison, filling a struct from pay through
actions and `To` went from 14.3 µs to 13.9 µs.

**Steps:** update the record with the numbers above, and say in its
What's left that grammar has taken it up.

**Done when:** the record's Outcome no longer waits on grammar.
