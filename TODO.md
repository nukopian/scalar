# TODO

## Number the roadmap's items, and add Ideas after Later

**From:** jdc-lab's session, 2026-10-26 (jdc-lab a1f78814fcade5d2522dee7c48cbe2ec2ace046b)
**Waiting on:** nothing

**What:** do what jdc-lab's commit
a1f78814fcade5d2522dee7c48cbe2ec2ace046b did to its roadmap: number
every item, add a fourth group, Ideas, after Later, and prune the ideas
at each wash-up.

**Why:** Iain decided it on 2026-10-26, for jdc-lab, grammar and scalar
alike: ideas are kept so they aren't lost, without a promise to do
them, and numbers let items be named in pruning and in commits.

**The rules** (as settled):

- **Groups:** Later now means meant to be done, but not yet planned.
  Ideas: might be done, or never; kept so they aren't lost. An idea is
  a numbered heading, the date it was added, and a line or two on what
  and why; no Serves line (it gains one when promoted). Newest last.
- **Numbers:** one sequence for the whole roadmap. An item gets the
  next number when it's added, in whichever group, keeps it as it
  moves between groups, and a number is never used again. The opening
  says which comes next ("Next number: 12"). Write it in the heading,
  `### 7. Title`, never as a Markdown numbered list, which renumbers
  itself when an item leaves.
- **Shipping:** the CHANGELOG line for a shipped item ends
  "(roadmap 7)".
- **Pruning:** each wash-up lists the ideas by number, and Iain keeps,
  promotes (number kept, Serves added) or drops each (deleted; the
  commit message names its number).
- **Adding:** an idea is added in a `docs:` commit whenever one comes
  up; when Iain mentions one in passing, Claude offers in one line to
  add it ("add as idea 12?").

**What's known:** scalar's roadmap has one item, in Later, written as a
bullet ("Possibly: a regex engine chosen per pattern"), not a heading:
it becomes `### 1. …`, and its "Possibly" suggests it belongs in Ideas,
though that's this session's call, with Iain. The opening's group list
also gives each group's contents ("in a line or two"); keep that style.

**Steps:**

1. ROADMAP.md: the opening's groups as above, the numbering paragraph
   with "Next number", and an empty `## Ideas` section after Later
   ("None yet." and a line on what an idea is). Number the existing
   items in reading order, Now first. jdc-lab's ROADMAP.md at
   a1f78814fcade5d2522dee7c48cbe2ec2ace046b has the wording.
2. Fix every link to a renamed heading: `rg -n 'ROADMAP\.md#' .`.
3. docs/MAINTAINING.md: the roadmap bullet gains the numbering and
   Ideas, and the release check that covers ROADMAP.md gains the
   pruning. Public files say "the maintainer", not Iain's name, unless
   they already name him.
4. CHANGELOG under Unreleased; the local CLAUDE.md's roadmap rule too.

**Done when** every item has a number in its heading, Ideas follows
Later, no link to the roadmap is broken (check the anchors with
`pandoc ROADMAP.md --from gfm -t html | grep -o 'id="[0-9][^"]*"'`),
and the ROADMAP and MAINTAINING PDFs are rebuilt and the changed pages
read.
