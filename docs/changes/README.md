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

- **It's opened when work starts.** The plan moves here from the
  roadmap or `TODO.md`, and the roadmap's Now item shrinks to a
  paragraph and a link. `TODO.md` keeps only the next steps for a
  session.
- **The plan isn't edited once work starts.** What changed goes under
  What happened, so the record shows how the plan held up.
- **What happened is added as the work goes,** and finished when the
  change ships or is dropped, in the same commit. When it ships, the
  status says "shipped, unreleased", and the release names its version.
- **It's named for the day work started and the change,** such as
  `2026-10-26-short-title.md`: dates never collide between sessions
  or repositories, and sort in order. A roadmap item keeps its number
  as its name in commits and the CHANGELOG, and the record gives it in
  its header.
- **It's public,** on `main` and in every release, like the rest of
  `docs/`, so nothing personal goes in it.

## The template

```markdown
# Title

**Status:** in progress | shipped, unreleased | shipped in v0.5.11 | dropped

**Origin:** roadmap 7 | a TODO task from grammar's session (grammar <commit>)

**Started:** 2026-10-26, in <commit>

**Finished:** 2026-10-27

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

None yet.
