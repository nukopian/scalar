# TODO

## Keep a record of each planned change, in docs/changes/

**From:** grammar's session, 2026-10-26 (grammar aa0262168067f242e16096f2717479144839c2eb)
**Waiting on:** nothing

**What:** do what grammar's commit
aa0262168067f242e16096f2717479144839c2eb did: a `docs/changes/`
folder of change records, with a README holding the rules and the
template; the rule in `docs/MAINTAINING.md` and `CLAUDE.md`; the
roadmap's Now items shrunk to a paragraph and a link to their record;
the release step that names the version in each shipped record; the
PDF script building the records; and a CHANGELOG line.

**Why:** Iain decided it on 2026-10-26, for grammar, scalar and
jdc-lab alike. A plan's What, Why, What's known, Steps and Done when
were lost once the work shipped, and so was what happened to them: the
changes from the plan, the measurements, what was tried and reverted.
Some reached commit messages and the gaps, in pieces. A record keeps
it in one place, for looking back.

**The rules** (as settled):

- **Folder and names:** `docs/changes/`, one file per change, named for
  the day work started and the change, such as
  `2026-10-26-limit-errors-that-say-where.md`. Dates need no
  allocation between sessions or repositories and never collide; a
  roadmap item keeps its number as its name, given in the record's
  header and the CHANGELOG.
- **Which changes:** anything with a plan: a roadmap item that reaches
  Now, or a `TODO.md` task with steps. A planned change that's dropped
  keeps its record, saying why. Small fixes don't need one.
- **When:** opened when work starts. The plan moves into it word for
  word, and the roadmap's Now item shrinks to a paragraph and a link;
  `TODO.md` keeps only a session's next steps. The plan isn't edited
  after; what happened is added as the work goes, and the record is
  finished in the commit that ships or drops the change.
- **Contents:** a header (Status: in progress, shipped, unreleased,
  shipped in vX.Y.Z, or dropped; Origin; Started, with the commit;
  Finished), each line its own paragraph so they don't run together;
  The plan; What happened: the outcome against Done when, changes from
  the plan and why, measurements and how they were taken, what was
  tried and reverted, and what's left.
- **Links:** the CHANGELOG line for a shipped change links to its
  record, as in "(roadmap 7; [record](docs/changes/….md))"; a release
  turns "shipped, unreleased" into its version.
- **Public:** records are on `main`, so nothing personal goes in them.

**What's known:** grammar's `docs/changes/README.md` at
aa0262168067f242e16096f2717479144839c2eb has the wording, and its
`docs/MAINTAINING.md`, `CLAUDE.md`, `ROADMAP.md` and
`scripts/readme-pdf.sh` changes there are the pattern (the script needs
`cd "$root"` before the loop, as the records' glob is relative). Take
the release step from grammar's ded2edb9f438da9dc69a16af99c06ec8983ca6bd
instead: its `grep -n 'shipped, unreleased$'` matches only the status
lines and the records' list, where the first version listed the
records' README every time, as its rules contain the phrase. Its
first record, `docs/changes/2026-10-26-limit-errors-that-say-where.md`
at 6e61995ea186d466f7833385c0f2fa8bbdd419fb, is a worked example:
`git -C ../grammar show 6e61995ea186d466f7833385c0f2fa8bbdd419fb`.
scalar's Now is empty, so no plan has to move yet.

**Steps:**

1. `docs/changes/README.md`, from grammar's, for scalar, with "None
   yet." under Records.
2. `docs/MAINTAINING.md`: the roadmap bullet (a Now item's plan is in
   its record; the CHANGELOG line links to it), a `docs/changes/`
   bullet, and the release step that turns "shipped, unreleased" into
   the version, with its `grep`.
3. `CLAUDE.md`: the Now rule, a record rule, and `docs/changes/` in the
   list of documents. `ROADMAP.md`'s opening: Now's plans are in their
   records, and a shipped item's CHANGELOG line links to its record.
4. `scripts/readme-pdf.sh` builds `docs/changes/*.md`; CHANGELOG under
   Unreleased.
5. `docs/PROCESS.md` is Iain's essay: ask him whether it should mention
   change records, and don't edit it unasked.

**Done when** `docs/changes/` has its README, the rules are in
MAINTAINING and CLAUDE.md, every link resolves, and the PDFs are
rebuilt and the changed pages read.
