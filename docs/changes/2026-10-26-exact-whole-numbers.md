# JSON whole numbers that keep their digits

**Status:** in progress

**Origin:** roadmap 2

**Started:** 2026-10-26, in d86c548e0cef495a174f025ef93383151667062c

**Finished:**

## The plan

The plan as it was when work started, from the roadmap's Next, word for
word.

**What:** a JSON whole number too big for an `int64`, such as the id
`12345678901234567890`, keeps its digits as text beside the `float64`
it becomes, so `Str()`, `MarshalJSON`, `Dump`, YAML output and
`scalar.To` into an exact type, such as shopspring/decimal's, give
every digit, while `Num()`, arithmetic and `Any()` still see the float.
Go `uint64` values above `int64`, which become floats too, keep theirs
the same way. Decimals stay as they are, as in Perl; idea 3 is keeping
theirs.

**Why:** big ids are common in API JSON, and today they lose their last
digits silently: `12345678901234567890` is written back as
`12345678901234567000`, and `To` can't win them back. Perl keeps them:
JSON::PP reads such a number as a plain string with every digit, and
Perl adds the float beside it when the number is first used as a
number, so both mean the same number (Perl's ordinary storage, not
Scalar::Util's `dualvar`, whose two can disagree). A number up to
2⁶⁴−1 is an exact UV in Perl, with exact arithmetic; scalar's arithmetic
there stays float, a deviation to document.

**How, as far as it's known:** the text goes in the scalar's `rare`
field, so the float stays in `v` and the code that switches on
`float64` is unchanged. Both JSON decoders end in
`normalizeJSONNumber`, and the fast path's integers of up to 18 digits
never reach it, so the common case costs nothing. Every write passes
through `vivify()`, which clears the text, so a changed value never
prints the old digits. `Copy` and `Clone` carry it.

**Done when** whole numbers that don't fit an `int64` round-trip through
`FromJSON` and `MarshalJSON` with every digit; a differential test
compares them with JSON::PP; a test shows that every mutator clears the
text; the JSON benchmarks show no real cost; the fast-paths comparison
and `FuzzReadJSON` cover it; and the reference's JSON section and
Caveats say what's kept, and what isn't: decimals, and exact arithmetic
above `int64`.

## What happened

- **Perl doesn't do what the plan says.** JSON::PP 4.18 reads an integer
  of up to 20 characters as `0+$v`: an exact IV or UV from -2⁶³ to
  2⁶⁴−1, and a lossy float outside it. Only a longer one stays a string,
  and JSON::PP writes that back quoted. So Perl keeps
  `12345678901234567890` as a UV, not as text beside a float.
- **Two designs were tried,** behind one switch on the
  [`experiment/exact-integers`](https://github.com/nukopian/scalar/tree/experiment/exact-integers)
  branch (d86c548e0cef495a174f025ef93383151667062c): the plan's, digits
  in `rare` beside the float, and Perl's, a `uint64` in `v`. Both kept
  the digits at no real cost: reading a feed of 2,000 earthquakes, each
  with an id beyond `int64`, cost the plan's 2.2% and Perl's 1.1%. The
  plan's design fails quietly, as a copy that forgets the digits gives
  today's float, and one did: `UnmarshalJSON` passed the fast reader's
  scalar to `Set`, which kept only `v`. Perl's fails loudly, as a type
  switch that forgets `uint64` gives a wrong answer that a test can catch.
  Perl's was chosen: it matches JSON::PP, adds no deviation, and makes
  exact arithmetic cheap.
- **Changes from the plan:** whole numbers up to 2⁶⁴−1 are a `uint64`,
  not text beside a float, so `Kind` is `Int` and `Any` gives the
  `uint64`; beyond 2⁶⁴−1 they're floats, as in Perl. Arithmetic is exact
  wherever Perl's is, following `pp_add`, `pp_divide`, `pp_modulo` and
  `pp_pow` in Perl's source, rather than float above `int64`. Deviations
  that do better than Perl are taken and documented, as Iain decided:
  exact `Sum`, `Product`, `MaxNum` and `MinNum`, where List::Util reads a
  UV as negative; `%d` printing a UV's digits; exact whole powers; and an
  infinite huge power. Idea 3 no longer reuses this storage.
- **Found on the way, on develop already:** every JSON decimal tried
  `ParseInt` first, and its failure allocated; skipping it made reading
  10% faster (6d63685a32deb1c4522bd717662cbd6bef582d93). `**` gave an
  exact integer where Perl gives a float, `%` was wrong for floats beyond
  `int64` (`1e19 % 7` was `0`, not `3`), and `"007"++` was `8`, not
  `"008"`; the build follows Perl in each. A negative zero prints as
  `-0`, where Perl prints `0`: not changed, and documented.
- **The build** is on the
  [`experiment/uv-integers`](https://github.com/nukopian/scalar/tree/experiment/uv-integers)
  branch (78abd7092a6d4dc3632d6eb95adc82ccd6f8004e), to be tested before
  it reaches develop. A differential test runs 77,222 cases of every
  operator through Perl 5.36 to 5.42, each difference in a named group;
  math/big checks the arithmetic, in a test and a new fuzz target; and
  JSON, Dump and YAML are compared with JSON::PP, Data::Dumper and
  YAML::PP.
- **Measurements,** against develop, median of 6 on an Apple M2: adding
  integers +0.7%, floats +4.4%, numeric strings +4.4%; a float comparison
  +5.0%; sorting 1,000 numbers +5.7%, or +32.4% when a third are big ids,
  which now compare exactly; reading and writing JSON within 2%.
- **What's left:** Iain reviews the branch and its deviations; then it's
  rebased onto develop, and this record is finished in the commit that
  ships it.
