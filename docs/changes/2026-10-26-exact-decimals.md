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
- **What's left:** the experiment is being built; this record is
  updated as it goes.
