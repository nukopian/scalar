# Working on scalar

These are the working rules for this repository. scalar is developed by
Iain Campbell with Claude (see the README's Acknowledgements), and these
rules are how that partnership has run since the first commit. Each rule
comes with the reason for it.

## What scalar is for

It aims to be easy to use, to give people a low barrier into Go, and to
get a proof of concept to results quickly. Raw speed comes second. It's a
halfway house for loosely shaped data (API and LLM JSON, YAML, messy
text) on the way to typed Go. Prefer clearer, more forgiving APIs and
good error messages over speed, unless a measurement shows a real cost.

## Perl is the baseline, and not the only source

- **Where Perl has a feature, match Perl's behaviour.** Where Go can't
  follow Perl, deviate pragmatically, and document each deviation in
  `docs/REFERENCE.md`, in the feature's section or under Caveats. Raise any new
  deviation with Iain before building it.
- **Other features are borrowed from elsewhere**, as the README's
  Rationale describes:
  - `Then`, `When`, `Or` and `OrSet` adapt the Maybe/Option style of
    programming to scalar's undef. They don't copy any one language's API,
    Scala's included. `TestThenMonadLaws` checks the monad laws, the rules
    that style itself sets.
  - XPath and JSONPath for `Path`, `At`, `DeleteAt` and `Find`, with
    Python's slices.
  - CPAN modules for sets and bags.
  - Go's iterators and generics for `All` and `To`.

  A new borrowing names its source in the reference. Where the source is
  concrete, such as a spec or a module, follow its behaviour and test
  against it. Where it's a style of programming, keep to that style's
  rules.
- **Check what Perl does before writing an expected output.** Use
  `perl -E` (5.36 or later). For CPAN modules, install them into a
  temporary directory with `cpanm -L` and set `PERL5LIB`. Don't change
  the system Perl.
- **Naming follows Go's convention:** `UcFirst`, `RIndex`, `QuoteMeta`.
  One-word Perl names (`Substr`, `Splice`) and short operators (`M`, `S`,
  `Tr`, `Lc`) keep Perl's spelling. The Perl-shaped result gets the Perl
  name, and the Go form gets a suffix: `Keys()` returns an array,
  `KeysSlice()` a Go slice. "Slice" only ever means a Go slice.
- **The exception: `Find` follows RFC 9535, not Perl.** The JSONPath
  Compliance Test Suite in `testdata/cts` must pass in full.

Why: scalar brings proven shortcuts from Perl and elsewhere into Go.
Faithfulness to each source makes it predictable for anyone who knows
that source, and the documented deviations make it honest for everyone
else.

## Tests

- Differential tests run the same cases through Perl and compare the
  results. They skip through `needPerl(t)` with `-short`, without Perl, or
  with a Perl older than 5.36, as macOS and Windows ship; three also skip
  without List::UtilsBy, YAML::PP or autovivification. `TestMappingMatchesPerl`
  checks `PerlUnusable` instead, so that it still compares scalar with plain
  Go, and leaves Perl out of its one task that needs autovivification
  without it, saying so.
  `TestDumpMatchesPerl` also skips with a Data::Dumper older than 2.188,
  which comes with Perl 5.38: 5.36's prints booleans as `1`, not `!!1`.
- On Iain's Mac, the Perl tests run in a perlbrew library of their own,
  `perl-5.36.3@scalar`, `perl-5.38.5@scalar`, `perl-5.40.5@scalar` or
  `perl-5.42.3@scalar`, with the modules they need. His default Perl is
  another project's, so name the library, which `go test` passes on to
  the `perl` it runs:

  ```
  perlbrew exec --quiet --with perl-5.42.3@scalar go test ./...
  ```
- Fuzz targets: `FuzzReadJSON`, `FuzzNumericPrefix` and `FuzzFind`. Run
  the relevant one after changing the code it covers, for example
  `go test -run '^$' -fuzz FuzzFind -fuzztime 60s`.
- Fast paths (`fastpath.go`, `fastjson.go`) sit behind the `fastPaths`
  switch. Each one has a test that runs with it on and off and compares
  the output, and a planted-bug check that the comparison catches a
  difference.
- The lock benchmarks run on a generated table, from a fixed seed, in the
  shape of the ONS ASHE pay table the reference's measurements were first
  made on. They need no data from outside the repository.
- scalar supports Go 1.25 and later, as `go.mod` says, and CI tests both
  1.25 and the newest Go. Don't use newer standard-library features unless
  raising the minimum is a deliberate decision. Dependencies set the floor
  too, and so does security: `golang.org/x/text` v0.41.0 is the oldest
  without known vulnerabilities (GO-2026-5970 reaches the case functions),
  and it needs Go 1.25. CI runs govulncheck.
- Run `gofmt -l .`, `go vet ./...` and `go test ./...` before every commit.
  CI (`.github/workflows/test.yml`) runs the same on every push: on Linux
  with the race detector, the Perl modules, govulncheck and staticcheck,
  failing if a Perl test skips; on macOS and Windows with `-short`; and 30
  seconds of each fuzz target. `.gitattributes` keeps files LF, so they
  compare byte for byte on Windows too.
- govulncheck also runs every Monday on `main`
  (`.github/workflows/vuln.yml`), so a vulnerability published after a
  release is found, and GitHub's Dependabot alerts, secret scanning and
  push protection are on. The README's Security section tells users
  this: keep it true when any of these change.
  [MAINTAINING.md](docs/MAINTAINING.md#security-checks) has the commands.
- gitleaks scans the whole history for secrets: in CI on every push, and
  by hand (`gitleaks git --redact .`) before a repository goes public,
  before a tag, and after any history rewrite, before it's pushed. A leak
  stops the work like a failing test: once history is published, it
  can't be taken back.

## Documentation keeps up with the code

A feature isn't done until its documentation is.

The documents: `README.md` (what scalar is, a synopsis, the rationale),
`docs/HOWTO.md` (tasks), `docs/REFERENCE.md` (each feature in depth),
`docs/EXAMPLES.md` (one generated example per function), `docs/PERL.md`
(a Perl → scalar → plain Go phrasebook, whose every row
`TestMappingMatchesPerl` runs in all three and compares) and
`docs/PROCESS.md` (Iain's essay on how scalar and grammar were made, the
why behind these rules; his words, so change it only when he asks), with
`CHANGELOG.md` (what each release changed), `ROADMAP.md` (what scalar
means to do next, and why) and `docs/MAINTAINING.md` (branches, releases,
and the git and GitHub work done by hand) beside them.

- Every exported function and method has an example in the right
  `example_*_test.go` file. `TestEveryFeatureHasExample` enforces this.
- Examples use earthquake data, as in the USGS feed: places, magnitudes,
  depths, felt reports, station codes and networks. Keep to that theme, and
  to fields the real feed has, unless an example needs something neutral.
- `docs/EXAMPLES.md` is generated from those examples. Edit the
  examples, then run `go test -run TestReference -update`.
- New features go into the README's feature table and into the
  reference, and into the how-to guide when they change how a task is
  best done.
- Compile Markdown snippets to check them, using a temporary test file
  that you delete afterwards.
- Every link between the documents must resolve, including the
  `#anchor`s.
- PDFs of the documents are for reading offline, and git-ignored:
  `scripts/readme-pdf.sh` builds them all (pandoc and Prince). Don't commit
  them; module downloads would carry them.

Why: the documents are how people learn the package, its author included,
and gaps found by reading them have led to real features.

## How changes are made

- **Measure, don't predict.** For a speculative performance change:
  branch from a clean tree, record baseline benchmarks, and profile first,
  because the profile's share sets a ceiling on the gain. Put the change
  behind a switch with a differential test. Keep what pays, revert what
  doesn't, and record reverted ideas and their numbers in the reference
  so they aren't retried blind. Predictions here have been wrong more than
  once.
- **Wash-ups.** After a run of features, review them together for crashes,
  memory, correctness, inefficiency and slop. Reproduce each finding
  before reporting it, rank the findings, and write failing tests before
  fixes. Re-run the Perl differential tests to show nothing else changed.
  The wash-up before a release also checks `ROADMAP.md`, and prunes its
  Ideas: list them by number, and Iain keeps, promotes or drops each (a
  dropped one is deleted, its number named in the commit). What shipped is
  out of it, and Now is what's really being worked on.
- **Dogfooding.** Use scalar in real programs, note where it fought you,
  and fix the frequent friction. Check the fix by reworking those
  programs with their output unchanged. Do this alongside each wash-up.
- **Show the Go.** When scalar code handles data that's already typed,
  show the plain Go version beside it and recommend it. scalar belongs at
  the edges, where the shape isn't known yet.

## Planned work

- **`ROADMAP.md` holds what scalar means to do:** features and outcomes,
  not session tasks. **Now** is what's being worked on, each item with a
  full plan in `TODO.md`'s task format, and at most three items, so it
  stays a queue to pull from, not a fire-hose (Iain, 2026-10-25): another
  starts only when one ships or goes back to Next; **Next** is decided
  but not started (what, why, done when); **Later** is meant to be done,
  but not yet planned, in a line or two; **Ideas** might be done, or
  never, kept so they aren't lost: a numbered heading, the date added
  and a line or two, newest last. Most useful first in each. The
  CHANGELOG is the past, the README the present, and the roadmap the
  future.
- **Every roadmap item has a number,** in its heading (`### 7. Title`,
  never a Markdown numbered list, which renumbers itself): the next one
  when it's added, which the roadmap's opening gives ("Next number: 2"),
  kept as it moves between groups and never used again (Iain,
  2026-10-26). A shipped item's CHANGELOG line ends "(roadmap 7)". When
  Iain mentions an idea in passing, offer in one line to add it ("add as
  idea 2?"), and add it in a `docs:` commit.
- **The roadmap is public, on `main`,** and in every release, so people
  following scalar can see what's coming. What's kept out of the README
  is kept out of it too. Its PDF is built with the others.
- **The roadmap is kept current:** an item leaves when it ships, in the
  commit that adds it to the CHANGELOG, and a change that ships, starts
  or drops a feature updates it in the same commit.
- **`TODO.md` comes first:** its tasks take priority over the roadmap's
  items. When a session takes up an item, its next steps go into
  `TODO.md`, or are done at once.
- **At the start of a session, check for `TODO.md`** at the top of the
  repository, on `develop`. If it's there, read it, ask Iain whether each
  task should still be done, and do the ones he confirms. Remove each
  task in the commit that completes it, so the history shows which
  commit did which, and delete the file with the last one. A task he
  drops is removed too, saying why in the commit message.
- **Before each commit, `git pull --ff-only` and re-read `TODO.md`.**
  Sessions in the other repositories add tasks here, mid-session too, and
  a commit that completes a task must remove it.
- **Work planned for a later session goes into `TODO.md`,** committed on
  `develop`, with enough to do it cold: what, why, the files involved,
  and how to check it's done. Not only in Claude's memory or the chat,
  which Iain can't review and a fresh session may not have.
- **Each task opens with where it came from and what it waits on,** so
  a session can see at a glance which tasks it can do now, and whom to
  tell when one is done. As grammar's first such task did:

  ```markdown
  ## Mark each match with scalar's Autovivify(false)

  **From:** scalar's session, 2026-10-25 (scalar 5fd3f0dd76c34de50c011d0389719abf85dcc0f2)
  **Waiting on:** a scalar release with Autovivify
  ```

  `From` names the repository whose session added it, the date, and the
  full hash of the commit that prompted it, or, if that work isn't
  committed yet, the last commit and "work uncommitted". `Waiting on`
  is "nothing" when the task can be done now.
- **A task added to another repository's `TODO.md` is committed and
  pushed there at once,** after `git pull --ff-only`, as its own `docs:`
  commit: on `develop` for grammar and jdc-lab, on `main` for job-meta.
  `TODO.md` changes only through commits. Left uncommitted, it makes
  that repository's next pull refuse to run, and the session that does
  the task never sees it arrive. A code fix made in another working copy
  to get past a bug may still stay uncommitted.
- **`TODO.md` never reaches `main`.** `main` is a fast-forward of
  `develop`, so a release waits until the file is gone: each task done,
  or, to outlast the release, a feature moved word for word into
  `ROADMAP.md`, or a decision awaiting Iain to a Claude Doc. There are
  no GitHub issues for planned work (Iain, 2026-10-25). The release steps
  in [MAINTAINING.md](docs/MAINTAINING.md) check for it.
- **A release ends by looking in the other repositories' `TODO.md`**
  for tasks waiting on it, and telling Iain which can now be done.
  Nothing else tells their sessions it has landed.

Why: planned work survives between sessions in the open, where Iain
reviews it like any other change, and a release never carries an
unfinished list, while the roadmap shows anyone following scalar what's
planned. grammar's CLAUDE.md has the same rule.

## Commits

- Conventional commits, such as `feat: ...`, `fix: ...`, `docs: ...`,
  `test: ...` or `ci: ...`. Say why as well as what.
- Claude commits and pushes changes once they're done and checked, and
  Iain reviews them. Commits are signed.
- Work is committed and pushed to `develop`. `main` is the latest release
  and moves only at a release, by a fast-forward to `develop`, with the
  tag on it; it's protected, so a push to it needs CI passing. Add what
  changed for users to `CHANGELOG.md` under Unreleased in the same
  commit, and update `ROADMAP.md` there if it ships, starts or drops a
  feature. [MAINTAINING.md](docs/MAINTAINING.md) has the release steps and the git
  and GitHub work done by hand.
- Every commit ends with a `Co-Authored-By` trailer for the Claude model
  that worked on it, currently
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Keep the README's Acknowledgements accurate: credit goes where the work
  was done.
