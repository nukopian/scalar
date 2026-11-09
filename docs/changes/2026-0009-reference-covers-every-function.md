# 2026-0009. The reference covers every function, and agrees with the guide

**Status:** open

**Decisions:** D1 open: whether a test checks the reference names every
exported function.

**Origin:** roadmap 2026-0009, from Iain on 2026-11-09, while planning
2026-0008

**Started:** not yet

## Decisions

- [ ] **D1.** Should a test check that the reference names every
  exported function and method, as `TestEveryFeatureHasExample` does
  for the examples? A name alone isn't coverage: a mention in passing
  would pass it. The options: no test, and an audit at each wash-up;
  a test that each name appears, accepting that; or a test that each
  has a heading or a row in a feature's table, which is stricter but
  ties the reference's layout to the test. Recommended: the test that
  each name appears, with an explicit list of the ones covered under
  another's name (the YAML marshalling methods under YAML, for
  instance), as it catches the gap that matters, a function the
  reference never mentions, at little cost.

## What's known

A count on 2026-11-09, by name alone, of the 282 exported functions and
methods found six the reference doesn't mention: `AV.Items`,
`SV.Format`, `SV.MarshalYAML`, `AV.MarshalYAML`, `HV.MarshalYAML` and
`SV.UnmarshalYAML`. The how-to guide mentions about 68 of them.

## The plan

Written here when work starts.

## What happened

Nothing yet.
