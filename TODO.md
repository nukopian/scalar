# To do

Work planned for the next session. CLAUDE.md's
[Planned work](CLAUDE.md#planned-work) has the rules: ask whether each
task should still be done, remove it in the commit that completes it, and
delete this file with the last one. It must be gone before a release.

## Check every check run before pushing `main`

**What:** in [docs/MAINTAINING.md](docs/MAINTAINING.md), step 4 of
Making a release, before `git push origin main "$v"`, add a check that
every check run on the commit has completed successfully. Before the
push it lists each check once, from `develop`; after it, twice, as
`main`'s push runs CI on the same commit:

```
gh api "repos/nukopian/scalar/commits/$(git rev-parse HEAD)/check-runs" --jq '.check_runs[] | "\(.name) \(.status) \(.conclusion)"'
```

Add to When something goes wrong: a check stuck "in progress" after all
its steps passed is re-run alone, with `gh run rerun` and `--job`, and
a paste-ready command that finds the stuck job's id from the commit's
check runs, so no one has to look the ids up by hand;
and if a push of `main` and the tag is refused after the tag went up,
the tag stays as it is: fix the check, push `main` to the tagged commit,
then run `gh release create`.

**Why:** on 2026-10-22, grammar v0.1.5's push to `main` was refused.
GitHub never marked its Windows job complete, though every step had
passed and `gh run watch` had reported the run as a success. The tag went
up in the same push, so the release was split until the job was re-run.

**Files:** `docs/MAINTAINING.md`, and `CHANGELOG.md` under Unreleased.
grammar's guide gets the same change (its own `TODO.md` has it).

**Done when:** step 4 shows the command, with the repository's own name;
the When-something-goes-wrong entry is there; the command, pasted on
`main` after a release, lists every check as `completed success`; and the
PDF (`scripts/readme-pdf.sh docs/MAINTAINING.md`) reads well.
