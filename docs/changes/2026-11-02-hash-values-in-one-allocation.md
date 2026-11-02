# A hash's or an array's values in one allocation

**Status:** shipped in v0.6.4

**Origin:** a TODO task from grammar's session (grammar f163c99fba29ca7e6e1fdc6176e2e83e27ef28ee)

**Started:** 2026-11-02, in 26c2ac39f47a7d255f917fdfbb8431bba867109f

**Finished:** 2026-11-02

## The plan

The plan as it was when work started, from `TODO.md`, word for word.

**What:** in `plainHash` (`fastpath.go`), make the value scalars of the
hash in one allocation, a `[]SV` of the hash's size, taking a pointer to
each element, rather than one `&SV{}` (through `Value`, then `svOf`) for
each value. A value that's already a `*SV` still goes through `Value`,
as it's a copy with its digits.

**Why:** grammar's roadmap 25 made its matcher fast enough that building
a match's scalar data is now about two thirds of a parse, and nine
tenths of its allocations. Each match is a `scalar.Hash("~", text,
"from", n, "to", n)`, so three of its scalars come from `plainHash`.
Tried in this working copy through grammar's `go.work`, then reverted:

| grammar benchmark | before | after |
|---|---|---|
| `BenchmarkParse` | 5.44 µs, 126 allocs | 4.92 µs (−9.6%), 86 allocs (−32%) |
| `BenchmarkFindLines` | 523 µs, 4,082 allocs | 512 µs (−2.3%), 2,982 (−27%) |
| `BenchmarkFindAll` | 26.7 µs, 260 allocs | 25.9 µs (−2.7%), 195 (−25%) |
| `BenchmarkAssert` | 1.21 ms | 1.15 ms (−5.4%), 21% fewer allocs |

The experiment was:

```go
h := newHV((len(kv) + 1) / 2)
block := make([]SV, (len(kv)+1)/2)
for i := 0; i < len(kv); i += 2 {
	v := &block[i/2]
	if i+1 < len(kv) {
		if s, ok := kv[i+1].(*SV); ok {
			v = Value(s)
		} else {
			v.setValue(normalize(kv[i+1]))
		}
	}
	h.m[kv[i].(string)] = v
}
```

**What's known:** it keeps every value of the hash alive while any one is
referenced, so a program holding one value of a big hash and dropping
the rest keeps more memory than now; `Hash`'s fast path is for building,
so that's probably fine, but say so. `svOf` sets a value through
`setValue`, as the loop does, so the scalars are the same; check
nothing else `svOf` or `Value` does is skipped. The same could apply to
`Array`'s fast path, if it has one, and to `HV.Set` making many keys,
though that can't know how many in advance.

**Steps:** behind `fastPaths`, as the rest of `plainHash` is;
`TestHashFastPathSame` already compares `Hash` with the fast paths on
and off, so check it covers a `*SV` value and an undef trailing key, and
plant a bug (a value stored in the wrong element) to show it notices.
Benchmark scalar's own `Hash` benchmarks, if any, and grammar's with
`go.work`. Then a scalar release, and a task in grammar's `TODO.md` to
take it up.

**Done when:** `Hash` makes its plain values in one allocation, the
differential test covers it, and grammar's `BenchmarkParse` shows the
gain against the released scalar.

## What happened

- **Outcome:** done. `Hash` makes its plain values in one allocation,
  and `TestHashFastPathSame` covers it, with a `*SV` value and an undef
  trailing key as before, and now a check that setting two values of a
  hash made together leaves the rest as they were. A planted bug, every
  value stored in the block's first element, failed it: each value read
  as undef. Grammar's `BenchmarkParse` against the released scalar
  shows the gain, as below.
- **Half the gain wasn't the block.** A profile of the old code showed
  two allocations for each value, not one: `v := &SV{}` made an empty
  scalar for every key, which `Value`'s scalar then replaced, and as it
  went into the map, the compiler couldn't drop it. Grammar's experiment
  rewrote the loop and so lost both. `Hash`'s general code, for a list
  that flattens, had the same spare scalar, and loses it too: a
  three-key hash there makes 11 allocations, not 14.
- **Changes from the plan:** the block is sized for the values that need
  a new scalar, so a value that's already a `*SV`, copied through
  `Value`, leaves no unused element in it.
- **What `svOf` and `Value` did:** `Value` copies a `*SV` with its
  digits, and for anything else is `svOf(normalize(x))`, which only
  allocates and calls `setValue`. The loop calls `Value` for a `*SV` and
  `setValue(normalize(x))` on the block's element for the rest, so
  nothing is skipped.
- **Measurements in scalar:** two new benchmarks, `BenchmarkHash`, a
  hash shaped like grammar's match, and `BenchmarkHashWide`, 20 values;
  interleaved, 8 runs each, against the code before:

  | benchmark | time | bytes | allocations |
  |---|---|---|---|
  | `BenchmarkHash` | 234.3 ns → 158.1 ns, −32.5% | 624 → 464, −25.6% | 9 → 4 |
  | `BenchmarkHashWide` | 1240.5 ns → 586.1 ns, −52.8% | 2.914 KiB → 1.914 KiB, −34.3% | 45 → 6 |

  `ReadJSON` and `Clone`, which don't build through `Hash`, didn't move.
- **Measurements in grammar,** at f163c99fba29ca7e6e1fdc6176e2e83e27ef28ee,
  interleaved, 8 runs each, against scalar v0.6.3 (`GOWORK=off`) and
  this working copy through grammar's `go.work`:

  | benchmark | time | bytes | allocations |
  |---|---|---|---|
  | `BenchmarkParse` | 5.549 µs → 4.960 µs, −10.6% | −17.4% | 126 → 86, −31.7% |
  | `BenchmarkFindLines` | 393.9 µs → 377.1 µs, −4.3% | −13.3% | 4.083k → 2.983k, −26.9% |
  | `BenchmarkFindAll` | 28.25 µs → 27.15 µs, −3.9% | −6.8% | 260 → 195, −25.0% |
  | `BenchmarkAssert` | 1.184 ms → 1.106 ms, −6.6% | −13.9% | 23.87k → 18.76k, −21.4% |

  These match grammar's experiment.
- **The cost:** a hash's new values are freed together, so a program
  that keeps one value of a big hash after dropping the hash keeps them
  all. The reference's [What it costs](../REFERENCE.md#what-it-costs)
  says so, and how to let the rest go.
- **Taken further, the same day, to `Array` and `Push`,** which made
  one scalar for each value, `Array` through `flatten`. `plainCopy`
  now makes a plain value's copy from a block for all three, and
  `appendPlain` appends a block's worth for `Array` and `Push`, behind
  `fastPaths` and after `plainValues`, as `Push`'s fast path was.
  `TestPushFastPathSame` now compares `Array` too, its dump and its
  JSON, with an empty list and `*SV` copies among new scalars, and
  checks that setting two values leaves the rest. Two planted bugs were
  caught: the block not advancing failed it, and counting every value
  as a copy, leaving the block too small, panicked.
- **A float or an `int64` was boxed twice.** `normalize`'s cases for
  `int64` and `float64` returned the typed value, which Go boxed again
  into a new interface, where its cases for `bool` and `string` gave
  back the one passed in. They now do the same, so `Value` of a float
  or a large integer makes one allocation fewer, everywhere. In
  `BenchmarkArray`, of 20 floats, that alone took 40 allocations to 21.
- **Measurements, `Array` and `Push`:** two new benchmarks,
  `BenchmarkArray`, 20 floats, and `BenchmarkPush`, ten pushes of three
  values; interleaved, 8 runs each, against
  4d630bb090a48da47206190f0940f8b3f5015a9d:

  | benchmark | time | allocations |
  |---|---|---|
  | `BenchmarkArray` | 650.2 ns → 290.3 ns, −55.4% | 40 → 3 |
  | `BenchmarkPush` | 1066.0 ns → 796.2 ns, −25.3% | 47 → 16 |
  | `BenchmarkHash` | 165.3 ns → 167.3 ns, +1.2% | 4 → 4 |
  | `BenchmarkHashWide` | 631.9 ns → 642.3 ns, +1.7% | 6 → 6 |

  `ReadJSON`, `Clone`, `SumMag`, `SortNum` and the arithmetic didn't
  move. `Hash` pays for sharing `plainCopy`, which doesn't inline (cost
  201, as `Value` and `setValue` inline into it), about 2 ns a call; in
  grammar it doesn't show: against the same commit, `BenchmarkParse`,
  `FindLines` and `Assert` didn't move, and `FindAll` was 4.7% faster.
  One helper for the three was kept over a copy of the loop in `Hash`.
- **The whole suite, against the commit before both changes,**
  26c2ac39f47a7d255f917fdfbb8431bba867109f, interleaved, 6 runs each,
  found one slowdown: `Regex/Split`, 1.2–1.3% slower on both engines
  (10 runs each), with the same allocations. `Split` pushes one piece at
  a time, and `appendPlain` made a block of one and called `plainCopy`
  where `Push` had called `Value`. A list of one now goes straight to
  `Value`, and `Split` is level again (2.802 µs → 2.809 µs, Linear;
  3.429 µs → 3.420 µs, Backtracking; neither significant). Nothing else
  moved but `NumLtInt`, 3.6% faster. No benchmark run on both commits
  changed its bytes or allocations.
- **`BenchmarkPush` with that guard:** 680 ns, 1832 bytes and 17
  allocations, against 725 ns, 2072 bytes and 16 without it, though a
  push of three never reaches it. One push into an array with room
  costs one allocation either way; the difference is in how the
  array's slice grows over ten pushes (capacities 4, 8, 16, 32 with
  it, 3, 6, 12, 24, 48 without), which the change doesn't touch.
- **The wash-up before the release found nothing to fix.** With Perl
  5.42.3 and the race detector, the whole suite gives the same 779
  results at v0.6.3 and at d39d510e89e3aa55d2e5fbd1db2c72706eee5d7f,
  none failing or skipped. grammar's 526 tests and jdc-lab's 278 give
  the same results against v0.6.3 and against this scalar, swapped in
  alone through a workspace. Nothing in scalar relies on a scalar
  being its own allocation (no finalizers, weak pointers or `unsafe`).
  Every relative link and anchor in the documents changed since v0.6.3
  resolves, by a script that caught a planted bad anchor and a missing
  file. gitleaks finds nothing in 239 commits.
- **Roadmap idea 4 had predicted it.** Profiling scalar under a
  consumer's workload was dropped on 2026-11-01, as jdc-lab's profile
  found scalar at about 5% of its allocations, but it ended: "grammar
  makes new state for every text `FindAll` searches, and builds scalar
  values for each match, so the lab's remaining cost may include
  scalar's." Once grammar's roadmap 25 made its matcher fast, that
  cost was most of a parse.
- **Grammar against the released scalar,** at
  f29165bc1895eb5f2ba5bb19d7733c80e6a9cc91, with `GOWORK=off`, v0.6.3
  against v0.6.4, 8 runs each, interleaved, on an Apple M2 (grammar's
  numbers): `BenchmarkParse` 5.78 µs → 5.02 µs, −13%, 126 → 86
  allocations, −32%; `BenchmarkFindLines` −4.7%, `BenchmarkFindAll`
  −10% and `BenchmarkAssert` −8%, with 21–27% fewer allocations in
  each. Every grammar test passed, Raku's included. Filling a struct
  from a pay text through grammar's actions and `To`, in job-meta's
  comparison with Participle, went from 14.3 µs to 13.9 µs.
- **What's left:** `HV.Set` makes one scalar for each new key, and can't
  know how many keys are coming. Grammar has taken up v0.6.4.
