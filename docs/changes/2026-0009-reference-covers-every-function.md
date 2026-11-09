# 2026-0009. The reference covers every function, and agrees with the guide

**Status:** open

**Decisions:** D1 settled: a test checks the reference names every
exported function, with a list of those covered under another's name.

**Origin:** roadmap 2026-0009, from Iain on 2026-11-09, while planning
2026-0008

**Started:** not yet

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

Written here when work starts.

## What happened

Nothing yet.
