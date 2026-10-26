# JSON decimals that keep their digits

**Status:** in progress, as an experiment

**Origin:** roadmap 3

**Started:** 2026-10-26, on the `experiment/exact-decimals` branch, from
cb9a7dcf956ca5c3facf2447904e453a89dedb5c

**Finished:**

## The plan

The idea as the roadmap had it when work started, word for word:

> Added 2026-10-26. A JSON decimal with more digits than a `float64`
> holds, such as `1.23456789012345678901`, keeps its text beside the
> float, so money and measurements survive a round trip and `scalar.To`
> into an exact decimal type. Perl doesn't do this by default: JSON::PP
> reads such a number as a float, `1.23456789012346`, unless
> `allow_bignum` makes it a Math::BigFloat. So it would be a deviation
> from Perl, documented in the reference, or an option like
> `allow_bignum`. Item 2's build holds whole numbers as a `uint64`, so this
> would need storage of its own; the
> [`experiment/exact-integers`](https://github.com/nukopian/scalar/tree/experiment/exact-integers)
> branch built and measured text beside the float, for whole numbers only:
> decimals weren't tried.

Then the experiment, as agreed on 2026-10-26.

**What:** a number read as text, from JSON, YAML, a dump or a JSONPath
literal, keeps its text beside its float when the float can't give the
number back: a decimal such as `1.23456789012345678901`, or a whole
number beyond `int64`. What survives is the value, not the spelling:
`19.90` may come back as `19.9`. It's on for every number, with no
option to ask for it. Arithmetic stays on the float, and any write to
the scalar drops the text. `To[uint64]` and `Decode` read a big whole
number's digits exactly, and comparisons and `UniqueNum` use the digits.

**Why:** ids and measurements lose digits silently today. An option, as
JSON::PP's `allow_bignum` is, would be brittle: `json.Unmarshal` into a
struct with scalar fields can't pass one. Keeping the text only where
it's needed costs ordinary numbers a count of their digits, as a
`float64` gives back any number of up to 15 significant digits.

**What's known:** the `experiment/exact-integers` branch, for roadmap 2,
built text beside the float for whole numbers, with a carrier type from
the readers into scalars, and tests that every copy carries the text and
every mutator drops it; they're reused. Money stays with plain Go at the
edges, `To` into an exact decimal type: the trouble with money is float
arithmetic, which this doesn't change.

**Steps:** a branch off develop; benchmarks first, with a feed of long
decimals and big ids; tests before code, with exact arithmetic
(`math/big`) and JSON::PP's `allow_bignum` as oracles; the code; planted
bugs to show the tests catch what matters; fuzzing; the full checks;
then a report of the cost and risk, for Iain to decide.

**Done when** the experiment's report says what it costs on ordinary and
long feeds, how many places in the code it touches and how each fails,
and whether it's recommended; nothing reaches develop without Iain's
decision.

## What happened

- **Found on develop, on the way:** `To[uint64]` of the string
  `"12345678901234567890"` fails, though it holds every digit; two
  different big ids read from JSON are `NumEq` and `Equal`; and
  `UniqueNum` keeps one of two different strings of digits. The
  experiment fixes each from the kept digits.
- **Also found:** scalar writes a float in JSON as Go does, so
  `0.1 + 0.2` is written `0.30000000000000004`, where JSON::PP writes
  `0.3`, to 15 significant digits. The reference doesn't say so; Iain is
  to decide whether to document it or match Perl, separately.
- **Built and measured,** on the `experiment/exact-decimals` branch, not
  yet pushed: about 100 lines changed in 15 files, and a new file,
  `digits.go`. Against develop, median of 6 on an Apple M2: reading
  ordinary JSON costs 2.9% more, with no extra allocations; 2,000 big
  ids, 5.9%; 6,000 long numbers, 9.3%; writing, 1.5 to 2.5%; comparing
  two floats, 3.3%; sorting 1,000 ordinary numbers, 4%.
- **Three costs were found and cut,** each measured. Comparing kept
  digits on every comparison made float comparison 58% slower and a sort
  of close big ids 517% slower; floats are now compared first, as
  rounding never reverses an order, and the digits only when floats
  tie. Keeping JSON's text as written made every number's text escape to
  the heap, 14,000 extra allocations on the ordinary feed; the text is
  now copied only when it's kept. Parsing kept digits on every
  comparison allocated; they're parsed once and cached, except in a
  locked scalar, which may be read concurrently: a race test shows a
  cache there would be a data race.
- **Copies,** 2026-10-26: `Copy`, `Clone` and `Value` of a scalar,
  which `Sort`'s copy goes through, made a kept number's copy through
  the readers' carrier, three allocations each. They now give the copy
  the text directly, with its own record of it, sharing the parsed
  digits, which never change. Cloning the long feed went from 4.81 ms
  to 4.49 ms, and the big-id feed from 4.68 ms to 4.40 ms; develop
  clones them in 4.23 to 4.36 ms, so a kept number now costs one
  allocation more than develop. A sort of close big ids gained only
  4.6%, to 0.26 ms against develop's 0.16 ms: its copies come from
  elements never compared, with no parse to share. A `uint64`, as in
  roadmap 2's build, copies for nothing; text always costs something,
  the price of one mechanism for decimals and big ids alike.
- **Tested:** the whole suite on Perl 5.36, 5.38, 5.40 and 5.42, and the
  copy change on 5.36 and 5.42; `-race`, `-short` and Go 1.25; staticcheck and govulncheck at CI's pinned
  versions. `FuzzFaithful` checks whether to keep a number against exact
  arithmetic (`math/big`), and found two flaws in that oracle, for
  exponents beyond `big.Rat`'s reach; JSON::PP's `allow_bignum` agrees
  on every number it keeps. A planted bug in each of copying, clearing
  on write, the keep-or-not test, comparison, the locked cache and a
  copy's own record was caught.
- **What's left:** Iain decides whether it lands, and whether roadmap 2's
  `uint64` build is dropped. Landing it needs the reference written
  first, saying where it differs from Perl: Perl reads such numbers as
  floats, and prints 15 digits where a kept number's `Str` gives all of
  them.
