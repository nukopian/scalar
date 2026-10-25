# To do

Work planned for the next session. CLAUDE.md's
[Planned work](CLAUDE.md#planned-work) has the rules: ask whether each
task should still be done, remove it in the commit that completes it, and
delete this file with the last one. It must be gone before a release.

## Say where a Go user would expect Array not to flatten a map

**From:** grammar's session, 2026-10-25 (grammar 81eaa595a99b735a89a5f8a62f6bb105135257b6)
**Waiting on:** nothing

**What:** in `docs/REFERENCE.md`, give "Lists flatten, as in Perl" a
"What you might assume" table, as the nesting section has, with a row
for building an array of hashes: `scalar.Array(map[string]any{...},
map[string]any{...})` is an array of their keys and values, not of two
hashes, as Perl's `(%a, %b)` is. For what was expected: wrap each one,
`scalar.Array(scalar.Value(m1), scalar.Value(m2))`, as Perl's `(\%a,
\%b)` does, or read the data with `FromJSON`. Consider rows for the same
with a Go slice, and with `Push`.

**Why:** grammar's session, checking whether scalar needed an analogue
of grammar's `ToAll`, built test rows with `scalar.Array` of two Go maps
and passed them to `scalar.To[[]Pay]`. It failed with `Decode: [0]: want
a hash for grammar_test.Pay, got "min"`, which pointed at `To`, not at
the flattening. The behaviour is right, and documented under Lists
flatten, but a Go programmer meets it without looking there: it reads
as a constructor taking two values.

**Files:** `docs/REFERENCE.md` (Lists flatten, as in Perl); perhaps a
sentence in `Array`'s doc comment (av.go) pointing to it; `CHANGELOG.md`
under Unreleased.

**Done when:** the table is there, each row's code is compiled and run
in a temporary test, and the reference PDF
(`scripts/readme-pdf.sh docs/REFERENCE.md`) reads well.

## jdc-lab's TODO.md is now on develop

**From:** jdc-lab's session, 2026-10-25 (jdc-lab abf51d723bf3437e9aecf662612b1bc9cc90187d)
**Waiting on:** nothing

**What:** jdc-lab now works as grammar and scalar do: work on `develop`,
`main` the latest release, and `TODO.md` on `develop`, never on `main`.
Change what scalar says about it:

- CLAUDE.md's Planned work: "on `develop` for grammar, on `main` for
  job-meta and jdc-lab" puts jdc-lab with grammar, on `develop`.
- docs/MAINTAINING.md's release step 7: its loop pulls whatever branch
  each working copy has checked out, so it reads jdc-lab's `TODO.md`
  only if that copy is on `develop`. Read `origin/develop` for grammar
  and jdc-lab instead, as grammar's step 7 does
  (`git -C "$d" fetch -q` and `git -C "$d" show origin/develop:TODO.md`).

**Why:** a task added to jdc-lab's `TODO.md` on `main` would never reach
jdc-lab's sessions, which work on `develop`, and step 7 could miss tasks
waiting on a scalar release.

**Done when** neither file says jdc-lab's `TODO.md` is on `main`, and
step 7's loop reads jdc-lab's `TODO.md` from `origin/develop` whatever
its working copy has checked out.

## A ROADMAP.md, as jdc-lab has

**From:** jdc-lab's session, 2026-10-25 (jdc-lab dedfbc7b228ede0069cd12c0393301288fe88f0f)
**Waiting on:** nothing

**What:** add a ROADMAP.md at the top of the repository, agreed with Iain
on 2026-10-25, and the rules that go with it, worded to match this
repository. jdc-lab's
[ROADMAP.md](https://github.com/nukopian/jdc-lab/blob/develop/ROADMAP.md)
and its docs/MAINTAINING.md (Branches, Before committing, Releases) are
the example.

- **What it holds:** features and outcomes, not session tasks: what the
  project means to do, and why. Three groups: **Now** (being worked on,
  each with a full plan in TODO.md's task format), **Next** (decided,
  not started: what, why, done when) and **Later** (ideas worth
  keeping, a line or two). Most useful first in each.
- **Its preamble is the project's own:** jdc-lab's is written for a lab
  and doesn't carry over. Keep its shape: a line on what the roadmap
  is (what the project means to do, and why; the CHANGELOG says what
  it has done), the three groups, and that a shipped item moves to the
  CHANGELOG. Replace the lab's **Serves** list, its six research
  questions, with what this project's items serve, if anything does
  (for a library, perhaps areas such as API, correctness, performance,
  parity with the language it follows, or docs), or leave Serves out.
  Drop the lab's note on samples, figures and runs, which has no
  counterpart here.
- **Past, present, future:** the CHANGELOG is the past, the README the
  present, ROADMAP.md the future. The README links to it, and lists it
  where it lists the files at the root.
- **Public, on `main`:** unlike TODO.md, it's in every release, so
  people following the project can see what's coming. The publishing
  rules apply to it as to the README. If the repository builds PDFs of
  its documents, it gets one; TODO.md still doesn't.
- **TODO.md comes first:** its tasks take priority over roadmap items.
  When a session takes up an item, its next steps go into TODO.md, or
  are done at once.
- **Kept current:** an item leaves when it ships, in the commit that
  adds it to the CHANGELOG; a change that ships, starts or drops a
  feature updates it in the same commit; the wash-up before a release
  checks it.
- **No GitHub issues** (Iain): the rule that a TODO.md task outlasting a
  release moves to a GitHub issue changes to: a feature moves, word for
  word, into ROADMAP.md; a decision awaiting Iain still goes to a Claude
  Doc. This is in CLAUDE.md's Planned work and in the release steps'
  check for TODO.md.

**Why:** TODO.md never reaches `main`, so nobody following the project
could see what's planned, and Iain doesn't want GitHub issues for it.

**What's known:** seed it from what the project already plans: feature
tasks in TODO.md, plans in the docs, and anything Iain names. Ask him
what belongs in Now.

**Done when** ROADMAP.md exists with its three groups, the README links
to it, CLAUDE.md and docs/MAINTAINING.md describe it, nothing says to
move tasks to GitHub issues, and this task is removed in the same
commit.
