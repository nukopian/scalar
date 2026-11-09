# 2026-0009. The reference covers every function, and agrees with the guide

**Status:** shipped in v0.6.7

**Decisions:** D1 settled: a test checks the reference names every
exported function, with a list of those covered under another's name.

**Origin:** roadmap 2026-0009, from Iain on 2026-11-09, while planning
2026-0008

**Started:** 2026-11-09, on `develop`, from
633882aff9f5a92d7f45dd2d888df02e69cb19a0

**Finished:** 2026-11-09

## Decisions

- [x] **D1.** Should a test check that the reference names every
  exported function and method, as `TestEveryFeatureHasExample` does
  for the examples? Yes, Iain decided on 2026-11-09, so that nothing is
  missed: a test that each name appears, with an explicit list of the
  ones covered under another's name (the YAML marshalling methods under
  YAML, for instance). It catches the gap that matters, a function the
  reference never mentions, at little cost. A name alone isn't
  coverage, as a mention in passing passes it, so the audit this item
  does by reading still judges depth. The options set aside: no test,
  and an audit at each wash-up, which a busy wash-up can skip; and a
  test that each has a heading or a row in a feature's table, stricter
  but tying the reference's layout to the test.

## What's known

A count on 2026-11-09, by name alone, of the 282 exported functions and
methods found six the reference doesn't mention: `AV.Items`,
`SV.Format`, `SV.MarshalYAML`, `AV.MarshalYAML`, `HV.MarshalYAML` and
`SV.UnmarshalYAML`. The how-to guide mentions about 68 of them.

## The plan

**Serves:** learning

**Size:** M (3)

**What:** check that the [reference](../REFERENCE.md) covers each
exported function and method, and that it and the
[how-to guide](../HOWTO.md) agree: the same terms, the same advice on
the same task, and links between them that land on the right section.
A first count, by name alone, found six not mentioned: `AV.Items`,
`SV.Format`, and the YAML marshalling methods of `SV`, `AV` and `HV`.
A test then fails on any exported function or method the reference
doesn't name, but for a listed few covered under another's name.

**Why:** the guide will lean on the reference for how each feature
behaves, so a gap or a disagreement there would be copied into every
chapter that cites it.

**Steps:**

1. A test in scalar, `TestReferenceNamesEveryFunction`, lists the
   package's exported functions and methods as docref does, and fails
   for each whose name doesn't appear in code in the reference: in a
   code span or a code block, as a whole word, so that "format" in a
   sentence doesn't count for `Format`. A method counts once its name
   appears, whichever type it's on, as the reference describes
   `Get` once for all three. A short list, each with its reason, names
   those covered under another's name.
2. Run it, and see what it finds beyond the first count's six, which
   counted any mention.
3. Fill each gap in the feature's section of the reference, or add it
   to the list with the reason, and check the test fails on a planted
   function with no mention.
4. Read the how-to guide beside the sections of the reference it
   relies on, for terms, advice and links that disagree, and fix each
   in both.
5. Record what was found, finish the record, and a CHANGELOG line.

**Done when** every exported function and method is covered in the
reference, the test fails on a function added without a mention there,
each disagreement found is fixed in one place and the other, and what
was found is in the record.

## What happened

- **Outcome:** done as planned. `TestReferenceNamesEveryFunction`, in
  `referencecover_test.go`, fails on any exported function or method
  the reference doesn't name in code, and
  `TestReferenceNamesEveryFunctionCatches` shows it fails on a planted
  one and passes one the reference names. Matching only code spans and
  blocks found the same six as the first count of any mention, and
  nothing more, so the reference already named the rest in code.
- **The six gaps filled,** each in its feature's section, with no
  function needing the `coveredElsewhere` list, which stays empty and
  fails if anything listed is named after all:
  - `AV.Items` under Arrays' Go slices, beside `StrSlice` and the
    rest: the elements themselves, for a slice to index or pass on
    rather than `All`'s iterator. A probe showed setting one sets it in
    the array, and a hole comes back as undef and stays a hole.
  - `SV.Format` under Formatting: it's what makes `%d` and the rest
    take Perl's context. A probe showed `fmt` calls it in a slice and in
    an exported struct field, but not in an unexported one, which `fmt`
    can't reach, so the reference says so.
  - `SV.MarshalYAML`, `AV.MarshalYAML`, `HV.MarshalYAML` and
    `SV.UnmarshalYAML` under YAML, where the text said scalars work
    with the YAML package without naming the methods.
- **The guide beside the reference:** no disagreement found. Terms
  (undef, holes, byte strings) are the same; the claims the guide makes
  about autovivification, `MostCommon`'s ties, `Linear`'s speed (1.2 to
  3.4 times) and what `Linear` can't run match the reference; and every
  `#anchor` link among the README, guide, reference, phrasebook,
  examples, roadmap and MAINTAINING resolves. The guide's recipes match
  their examples in `example_test.go`, but for comments and output
  laid out differently, which 2026-0008 removes by generating them.
- **Changes from the plan:** none.
- **What's left:** the station-cleaning recipe reaches the list with
  `config.Find("$..stations").Get(0).AV()`, where the reference's
  `config.Find("$.stations[*]")`, or `config.Get("stations").AV()`,
  is simpler. Not a disagreement, so left for the guide's rewrite in
  2026-0010.
