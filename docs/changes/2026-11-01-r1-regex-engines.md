# A regex engine chosen per pattern

**Status:** in progress

**Origin:** roadmap 1

**Started:** 2026-11-01, on `develop`, from
55165748fece092f29a023dcd9815df7867fd813

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

Nothing yet.
