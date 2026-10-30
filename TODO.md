# To do

## Name each roadmap item's record with rN after its date

**From:** jdc-lab's session, 2026-10-30 (claude-practice a098eb69ccdf093f118dbc7c8b559d0df8de61e5)
**Waiting on:** nothing

**What:** rename the change records of roadmap items to the practice's
new form, `YYYY-MM-DD-rN-short-title.md` (`2026-10-26-r7-short-title.md`
for roadmap 7), and update every link to them. A record for a `TODO.md`
task has no roadmap number and keeps its name.

**Why:** the practice now names records so (PRACTICE.md, Change
records): a record is looked up by its roadmap number far more often
than by its date, and a search for `r7` finds it, where a bare 7 matches
every date and figure. jdc-lab has renamed its own (jdc-lab
6e589dbf47910ddb758821ac283050fb221b04a4), as has grammar (grammar
78bee6780fc6e97a9bb17fab0956f4699531bfdf, which also rewords
`docs/changes/README.md`'s naming rule), which show the way.

**What's known:** the records to rename, from each one's Origin line:

- `2026-10-26-exact-decimals.md` → `2026-10-26-r3-exact-decimals.md`
- `2026-10-26-exact-whole-numbers.md` → `2026-10-26-r2-exact-whole-numbers.md`

Each basename is unique, so replacing it wherever it appears updates
every link: the CHANGELOG, ROADMAP.md, `docs/changes/README.md`'s list,
other records and any other document. jdc-lab's one link into this
repository's records is pinned to a tag, so nothing outside needs
changing.

**Steps:**

1. `git mv` each record above to its new name, and replace its old
   basename in every tracked file.
2. Reword the naming rule in `docs/changes/README.md` ("It's named for
   the day work started and the change") to give the rN form, and its
   example.
3. Rebuild and read the PDFs of any document whose text changed (not
   the CHANGELOG or records, which have none).

**Done when** no tracked file names a roadmap record without its rN,
every link to a record resolves, and `docs/changes/README.md` states
the new form.
