# TODO

## No PDFs for the CHANGELOG and change records

**From:** jdc-lab's session, 2026-10-27 (jdc-lab 5410234270b2465a77dcedcdd8cb2c9356037698)
**Waiting on:** nothing

**What:** stop building PDFs of the CHANGELOG and the change records in
`docs/changes/`.

**Why:** they're read as Markdown, on GitHub or in an editor, and their
PDFs only pile up beside them. jdc-lab made the same change in the
commit above.

**What's known:** `scripts/readme-pdf.sh` builds every document's PDF
when run with no arguments, its list including `CHANGELOG.md` and
`docs/changes/*.md`.
jdc-lab's version of the script shows the change: the list loses them,
and `build` refuses them by name with a message, exit status 2.

**Steps:**

1. In `scripts/readme-pdf.sh`, take `CHANGELOG.md` and
   `docs/changes/*.md` out of the list, drop the `cd` if it was only
   for their glob, and refuse
   `*CHANGELOG.md` and `docs/changes/` files in `build`.
2. Say so wherever the documents describe the PDFs (MAINTAINING, and
   any rule to rebuild and read them after a change).
3. Delete the PDFs already built for them (they're git-ignored).
4. A line under Unreleased in the CHANGELOG.

**Done when** `readme-pdf.sh` with no arguments builds no PDF of the
CHANGELOG or a change record, and `readme-pdf.sh CHANGELOG.md` refuses.
