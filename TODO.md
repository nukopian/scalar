# TODO

## Name roadmap items and change records by year-serial ids

**From:** grammar's session, 2026-11-09 (grammar 0b725c90e72aeef93de6698cdb095151edbfeeba)
**Waiting on:** nothing

**What:** convert scalar's roadmap and change records to the ids the
shared practice now uses (claude-practice
f39a304c8ac747302e83d0d6e38bc6a2800b54f7), retroactively, as grammar did
in the commit above. Iain decided this on 2026-11-09, borrowing from
Ekorn-Pipeline and DBIx-Squirrel. Serves lines stay.

**What's known:** scalar has 5 items (Next number: 6) and 6 records, two of them for TODO tasks (2026-10-27-export-the-path-parser.md and 2026-11-02-hash-values-in-one-allocation.md), which take the next ids in the order they were opened.

**Shorthand:** Iain may name an item by its serial alone, as "26" or
"task 26", meaning this year's 2026-0026 in the project at hand; ask
before acting where it could mean another item, year or repository
(claude-practice 5e1151de10a0e75b777da2333f442791e1a60395). Nothing in
the repository needs changing for it, but memory notes can say so.

**Steps:**

1. `git -C ../claude-practice pull --ff-only`, and read PRACTICE.md's
   "The roadmap" and "Change records".
2. In ROADMAP.md, make each heading `### 2026-00NN. Title` (item N,
   added in 2026, keeps N as its serial); replace the opening's "Next
   number" paragraph with grammar's, ending "Last id:" and the highest id
   given, records included; turn **Moves with:** into **Depends on:**
   (one way: what must come first) and "split from N" into a **Split
   from:** line; give each idea **Added:** and **Size:** lines.
3. `git mv` each record in docs/changes/ from `DATE-rN-slug.md` to
   `2026-00NN-slug.md`, and each task record to the next id; prefix
   each title with its id, and convert the **Origin:** line's "roadmap
   N". Leave the rest of each record as written: plans aren't edited.
4. Rewrite every link to a renamed record (CHANGELOG, REFERENCE,
   docs/changes/README.md and any others), and "roadmap N" in current
   documents and the CHANGELOG's lines for shipped items to "roadmap
   2026-00NN". Leave past CHANGELOG entries that describe the old rule
   as they were.
5. Update docs/changes/README.md's naming rule and template (Decisions
   line, D1/D2 checkboxes, Depends on, status "open"), and
   MAINTAINING.md's account of the roadmap, as grammar's commit did.
6. Add a line under the CHANGELOG's Unreleased, rebuild the roadmap's
   and MAINTAINING's PDFs and read the changed pages, and run the
   project's checks before committing.

**Done when** no "Next number", "Moves with" or `-rN-` record name is
left outside history, every Markdown link resolves, and memory notes
naming items by number are updated.
