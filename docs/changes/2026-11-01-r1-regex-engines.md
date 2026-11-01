# A regex engine chosen per pattern

**Status:** shipped in v0.6.2

**Origin:** roadmap 1

**Started:** 2026-11-01, on `develop`, from
55165748fece092f29a023dcd9815df7867fd813

**Finished:** 2026-11-01

## The plan

The idea as the roadmap had it when work started, word for word:

> Added 2026-10-26. **Size:** M (3). Being considered, not decided.
> `CompileWith(scalar.Stdlib, pattern)` would match with Go's regexp, 1.2
> to 2.6 times as fast in the experiment's benchmarks and never
> backtracking, at the cost of lookaround, backreferences and Unicode
> `\w`; other engines, such as coregex, could plug in the same way. The
> experiment, with its tests and measurements, is on the
> [`experiment/regex-engine`](https://github.com/nukopian/scalar/tree/experiment/regex-engine)
> branch.

Then the design, as agreed on 2026-11-01.

**What:** scalar gets two regex engines, `scalar.Backtracking`
(regexp2, as now, and the default) and `scalar.Linear` (Go's regexp).
Nothing changes unless a program asks for `Linear`, in one of two ways:

- **One pattern, strictly:** `MustCompileWith(scalar.Linear, pattern,
  flags...)`, or `CompileWith`, which returns an error. A pattern
  `Linear` can never run is an error (a panic for `MustCompileWith`).
- **The whole program, forgivingly:** `scalar.DefaultEngine =
  scalar.Linear`, set once at the start, as `RegexTimeout` is. Pattern
  strings given to `M`, `Matches`, `Named`, `S` and `Split`, and to
  `Compile` and `MustCompile`, then use `Linear` where it can run them.
  A pattern it can never run uses `Backtracking`, and scalar warns,
  once for that pattern, through `scalar.RegexWarn`, a
  `func(msg string)` that writes to the standard logger unless it's
  replaced or set to nil.

A `*Regexp` compiled with a named engine keeps it, whatever the
default, so a library that needs lookaround can pin
`MustCompileWith(scalar.Backtracking, ...)`.

**The promise:** `Linear` never gives a different answer from
`Backtracking`, only a faster one. It doesn't promise never to
backtrack: where a match could differ, that match goes to
`Backtracking`, without a warning, as the answer is the same.

- **Accepted:** only the constructs known to behave the same in both
  engines. `\w`, `\d` and `\s` (and `\W`, `\D`, `\S`) are rewritten to
  regexp2's Unicode classes, and capture groups are renumbered to
  regexp2's order, named groups after unnamed ones. Anything not on the
  list is "can't run": lookaround, backreferences, atomic groups, `\G`,
  `\Z`, the `x` flag, POSIX classes, .NET class subtraction, `(?U)`,
  `(?P<name>)`, `\Q…\E`, and captures inside a repeated group that can
  match nothing, such as `(a?)*`.
- **Per match:** `Linear` checks the text first, and hands that match to
  `Backtracking` when the text could make the engines differ: `$`
  without `m` and text ending in a newline; `\b` or `\B` and text that
  isn't ASCII; the `i` flag and text holding a character whose case
  folds differently in the two engines, such as `ſ` or `İ`; and `g`,
  `Matches` or `S` with `g`, with a pattern that can match nothing.
  Text that isn't valid UTF-8 is matched as regexp2 matches it, with
  each bad byte read as U+FFFD.

The engine type is sealed: `Backtracking` and `Linear` are the only
engines. Other engines, such as coregex, could be let in later without
breaking anything. `Backtracking` keeps today's code path, so naming it
costs nothing.

**Why:** Go's regexp was 1.2 to 2.6 times as fast as regexp2 in the
experiment's benchmarks. Names that say what each engine does
(`Linear`, `Backtracking`) state the trade-off a program chooses. The
promise is that switching changes speed and nothing else, because the
data scalar is for, API and LLM JSON and messy text, is where non-ASCII
words and final newlines are normal, and a switch made to save labour
mustn't change answers quietly. No environment variable chooses the
engine: the environment shouldn't change what a program means.

**What's known:** the experiment branch (commit
200856d54c420e0753ae284dcf492463514bc565) has a seam that works from
match positions, Go's regexp behind it, S's expansion over positions,
and `TestEnginesAgainstPerl`, 43 cases through Perl, regexp2 and Go's
regexp. A probe on 2026-11-01 ran 26 patterns through regexp2 v1.12.0
and Go's regexp and found the differences listed above. In both,
`(?i)k` matches the Kelvin sign, `\p{Greek}` and `\x{e9}` work, `(?m)^`
and `(?m)$` agree, and `a{,2}` is literal. Perl's `re::engine::RE2`
is the precedent: it swaps Perl's engine for RE2, falling back to
Perl's engine, silently, for patterns RE2 can't run unless `-strict` is
given. It doesn't hide RE2's differences: its documentation lists
Unicode handling and final newline matching among its known bugs.
scalar's `Linear` goes further, so that the answers never differ.

**Steps:**

1. Baseline: `BenchmarkRegex` and `BenchmarkSExpand` on develop, median
   of 6.
2. Tests first: a table of the probe's cases and the experiment's,
   through both engines, expecting `Backtracking`'s answer; the regex
   tests run again with `DefaultEngine = Linear`; tests of the strict
   errors, the warning (once per pattern), pinning, and the per-match
   hand-over.
3. The code: the `Engine` type with its two values, `DefaultEngine`,
   `CompileWith`, `MustCompileWith`, `RegexWarn`; the allow-list scanner
   and rewriter; the per-match checks; the match loop over positions,
   from the experiment; the cache keyed by engine.
4. A fuzz target, `FuzzLinear`, generating patterns from the accepted
   constructs and text with non-ASCII letters, newlines and bad UTF-8,
   checking that `M` with `g`, `Named`, `S` and `Split` give the same
   answer on both engines. It joins the list in CLAUDE.md and CI.
5. Planted bugs: drop each per-match check and each rewrite in turn,
   and show the tests or the fuzzer catch it.
6. Benchmarks again: the default path, and the same cases under
   `Linear`.
7. Documents: the reference gets a section on the engines (which
   constructs `Linear` accepts, which text goes to `Backtracking`, the
   warning, strict against forgiving, the plain Go to use instead for
   typed code), written to be read cold; the README's feature table; the
   how-to guide (making a regex-heavy program faster); an example for
   each new exported name; the CHANGELOG.

**Done when:**

- Every existing test passes unchanged, and the default path's
  benchmarks are within noise of the baseline.
- The engine table, the regex tests under `Linear`, and `FuzzLinear`
  (60 seconds locally, and in CI) find no answer that differs, and
  each planted bug is caught.
- `Linear` is faster on the benchmarks' common cases. If it isn't, the
  item is dropped, and this record says why.
- The documents say plainly what `Linear` promises and what it doesn't,
  and their PDFs have been read.

## What happened

- **Outcome:** each Done when is met. Every existing test passes
  unchanged, and the default path's benchmarks are within noise of the
  baseline. The engine table, the random patterns, the regex tests run
  again on `Linear`, and `FuzzLinear` find no answer that differs, and
  each planted bug is caught. `Linear` is faster on every benchmark.
- **The design as built:** `Engine` is an `int` type with two constants,
  `Backtracking` (its zero value) and `Linear`; any other value panics,
  so it's sealed in effect. A `*Regexp` always holds regexp2's compiled
  pattern, so `Linear` refuses nothing regexp2 accepts silently, gives
  the same errors for bad patterns, and has it ready for a match it hands
  over. `regex_linear.go` holds the engine: a parser of regexp2's syntax
  that writes Go's, taking only what it knows, and the per-match checks.
- **Changes from the plan:**
  - Text that isn't valid UTF-8 goes to `Backtracking`, rather than being
    matched as regexp2 matches it. It's simpler, and gives the same
    answer.
  - The `i` flag needed more than the plan's check of the text. Reviewing
    the planted bugs showed that a pattern holding `ſ`, `İ` or the Kelvin
    sign differs even on ASCII text: `(?i)ſ` matches `s` in Go's regexp
    alone. These three, the only runes that fold differently (computed
    from Go's `unicode` tables, which both engines use), are refused
    under `i`, in the pattern or in a class range. And a pattern with
    other characters beyond ASCII under `i` needs ASCII text: `(?i)σ`
    matches the final `ς` in Go's regexp alone.
  - `Split` needs no hand-over for patterns that can match nothing: the
    empty matches Go's regexp drops are ones `Split` skips anyway.
  - A `[` inside a class is refused, though regexp2 reads `[[:alpha:]]`
    as a plain set, which Linear could copy. Such a pattern almost
    certainly means a POSIX class, which neither engine gives, and the
    warning says so.
  - The how-to's example of `DefaultEngine` sets `RegexWarn` to nil, as
    a warning comes once per pattern and `go test -count=2` would then
    see it only once. `TestRegexWarn` starts from an empty cache, for the
    same reason.
- **Measurements:** `BenchmarkRegex`, the median of 6 on an Apple M2,
  `Backtracking` against `Linear`: `M` with 5 captures 1.71 µs to 695 ns
  (2.5×), no match 202 to 105 ns (1.9×), `M` with `g` over 200 lines 141
  to 111 µs (1.3×), `Named` 925 to 402 ns (2.3×), `S` with `${1}` and `g`
  5.92 to 2.74 µs (2.2×), `S` with a function 4.85 to 2.42 µs (2.0×),
  `Split` 3.38 to 2.79 µs (1.2×), and `M` with `$` on text beyond ASCII
  4.01 to 1.19 µs (3.4×). The default path against develop at
  eaa4523a87ab6457665de03166e613f08f2e21fc, `BenchmarkRegex` and
  `BenchmarkSExpand` with `benchstat`: −0.13% overall, the largest change +1.36% (4 ns) on `S` with no match.
  Of the engine table's 390 patterns and flags, `Linear` took 324, and
  ran 87% of their matches on the table's texts itself.
- **Planted bugs,** each caught: dropping the check for `$` and a final
  newline, for `\b` and text beyond ASCII, for the fold traps in the text,
  for empty matches with `g`, or for bad UTF-8 (the table, the random
  patterns and `TestLinearHandsOver`); dropping the refusal of fold traps
  in the pattern (the table and `TestLinearTakes`); dropping the
  requirement of ASCII text for a pattern beyond ASCII under `i` (the
  table, once `(?i)σ+` was added: before, only `TestLinearHandsOver`);
  dropping the refusal of a capture in a repeat that can match nothing
  (the table and the random patterns); dropping the renumbering of
  groups (the random patterns and the regex tests on `Linear`); `\w`,
  `\s` or `\d` left as Go's, or `\w` without the joiners (the table); and
  dropping the refusal of `[` in a class (`TestLinearTakes`).
- **Found while testing:** `(a)?` was refused as a capture in a repeat
  that can match nothing, as the check looked at the repeat rather than
  its body. Fixed before commit.
- **Fuzzing:** `FuzzLinear` ran 683,531 inputs in 60 seconds with no
  difference; it then took the flags as input as well, after missing
  `(?i)ſ`, and ran 905,287 in 60 seconds with no difference. A ten-minute
  run then had a worker killed after eight minutes, on a pattern matching
  each of 8,000 bad bytes: the test's replacement expands `` $` `` and
  `$'` at every match, so its output grows as the square of the text, and
  that input took a gigabyte, on both engines alike. The fuzzer now skips
  text over 1 KiB. The next ten-minute run failed on `.+00` over 1 KiB of
  bad UTF-8, where both runs go to `Backtracking`: one hit the 100 ms
  `RegexTimeout` under the load of eight fuzz workers, and the other
  didn't. A timeout depends on the machine, not the pattern alone, so
  the comparison now skips a case where either run timed out. The third
  run found a bug that isn't `Linear`'s: when a later match times out,
  regexp2's `ReplaceFunc` returns an empty string and no error, so `S`
  with `g` on `Backtracking` could empty a scalar silently. It's fixed
  in its own commit, as the first finding of the wash-up before the next
  release. The fourth run found that regexp2 reads `\'0'` as a
  backreference to group 0: `\<name>` and `\'name'` are an old .NET form
  of `\k<name>`. Linear now refuses `\<` and `\'` outside a class. A fifth ten-minute run, after
  the wash-up's fixes, ran 8,128,256 inputs with no difference.
- **The wash-up** also found that a `RegexWarn` matching its own pattern
  deadlocked, as the warning was given inside a `sync.Once`; it's now an
  atomic flag. The fold traps are now named, `İ`, `ſ` and the Kelvin
  sign, with `TestFoldTraps` checking them against every rune, where a
  scan of every rune found them on the first pattern with `i`: that took
  about 90 ms, and the check of each rune of the text 8.9% less time on
  `M` with `i` over text beyond ASCII. The reference's caveats list the
  panics the engines add.
- **What's left:** nothing for this item. Two ideas, not on the roadmap:
  a pattern beyond ASCII under `i` could run on text beyond ASCII where
  no character's lowercase and fold differ; and other engines, such as
  coregex, could be let in by opening `Engine`.
