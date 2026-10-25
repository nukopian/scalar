# To do

Work planned for the next session. CLAUDE.md's
[Planned work](CLAUDE.md#planned-work) has the rules: ask whether each
task should still be done, remove it in the commit that completes it, and
delete this file with the last one. It must be gone before a release.

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
  each with a full plan in TODO.md's task format; three items at most,
  so the roadmap stays a queue to pull from, not a fire-hose), **Next** (decided,
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
