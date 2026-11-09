# Change records

How each planned change to scalar went: the plan as it was written,
then what happened. The [CHANGELOG](../../CHANGELOG.md) says what
changed for users, and links here; a record keeps the thinking behind
it, for anyone looking back, including whoever plans the next change.

A change gets a record when it has a plan: a roadmap item that reaches
Now, or a task in `TODO.md` with steps. A planned change that's dropped
keeps its record, saying why, as dead ends are the history most worth
having. Small fixes don't need one.

## How a record is kept

- **It's opened when its item first has a question to settle, or when
  work starts.** When work starts, the plan moves here from the
  roadmap or `TODO.md`, and the roadmap's Now item shrinks to a
  paragraph and a link. `TODO.md` keeps only the next steps for a
  session.
- **The plan isn't edited once work starts.** What changed goes under
  What happened, so the record shows how the plan held up.
- **What happened is added as the work goes,** and finished when the
  change ships or is dropped, in the same commit. When it ships, the
  status says "shipped, unreleased", and the release names its version.
- **It's named for its id, then the change,** such as
  `2026-0003-exact-decimals.md` for roadmap 2026-0003. A roadmap item's
  id is its own, kept as its name in commits and the CHANGELOG; a
  `TODO.md` task with steps takes the next id from the roadmap's
  opening, in the same sequence, though it never appears there. Ids
  sort in the order items were added; the **Started:** line gives the
  date. Records opened before 2026-11-09 were named for the day work
  started and "r" and the item's number, and were renamed to their ids,
  a task's taking the next id in the order they were opened.
- **Its questions are settled before work starts,** numbered D1, D2
  and on under Decisions: unchecked while open, checked once settled,
  with the answer, the date and why, and summed up in a **Decisions:**
  line under the status.
- **It's public,** on `main` and in every release, like the rest of
  `docs/`, so nothing personal goes in it.

## The template

```markdown
# 2026-0007. Title

**Status:** open | in progress | shipped, unreleased | shipped in v0.5.11 | dropped

**Decisions:** D1 settled: the answer, in a few words. D2 open: the question, in a few words. | None.

**Origin:** roadmap 2026-0007 | a TODO task from grammar's session (grammar <commit>)

**Depends on:** 2026-0003, if anything

**Started:** 2026-10-26, in <commit>

**Finished:** 2026-10-27

## Decisions

- [x] **D1.** The question? The answer, on 2026-10-26, and why.
- [ ] **D2.** A question still open? The options, and the
  recommendation.

## The plan

The plan as it was when work started, word for word: What, Why,
What's known, Steps, Done when.

## What happened

- **Outcome:** against Done when.
- **Changes from the plan,** and why.
- **Measurements,** and how they were taken.
- **Tried and reverted,** with the numbers.
- **What's left,** linked to its roadmap item or `TODO.md` task.
```

## Records

Newest first.

- [The reference covers every function, and agrees with the guide](2026-0009-reference-covers-every-function.md),
  roadmap 2026-0009: shipped, unreleased.
- [A how-to guide made from tested examples](2026-0008-a-how-to-guide-made-from-tested-examples.md),
  roadmap 2026-0008: open.
- [A hash's or an array's values in one allocation](2026-0007-hash-values-in-one-allocation.md),
  a task from grammar: shipped in v0.6.4.
- [Lazy values](2026-0005-lazy-values.md), roadmap 2026-0005: dropped.
- [A regex engine chosen per pattern](2026-0001-regex-engines.md),
  roadmap 2026-0001: shipped in v0.6.2.
- [Export the path parser](2026-0006-export-the-path-parser.md),
  a TODO task from grammar: shipped in v0.6.1.
- [JSON decimals that keep their digits](2026-0003-exact-decimals.md),
  roadmap 2026-0003: shipped in v0.6.0
- [JSON whole numbers that keep their digits](2026-0002-exact-whole-numbers.md),
  roadmap 2026-0002: dropped.
