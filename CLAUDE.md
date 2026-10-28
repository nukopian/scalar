# Working on scalar

@../claude-practice/PRACTICE.md

The shared practice above, from
[nukopian/claude-practice](https://github.com/nukopian/claude-practice),
covers planned work, the roadmap, change records, branches, commits,
releases, wash-ups, documents and secrets. These are scalar's own rules
on top of it; where they differ, these win. scalar is developed by Iain
Campbell with Claude (see the README's Acknowledgements), and these
rules are how that partnership has run since the first commit.

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
  - XPath and JSONPath for `At`, `DeleteAt` and `Find`, with
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
- Fuzz targets: `FuzzReadJSON`, `FuzzNumericPrefix`, `FuzzFind` and
  `FuzzFaithful`. Run the relevant one after changing the code it covers,
  for example `go test -run '^$' -fuzz FuzzFind -fuzztime 60s`.
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

## Documents

The documents: `README.md` (what scalar is, a synopsis, the rationale),
`docs/HOWTO.md` (tasks), `docs/REFERENCE.md` (each feature in depth),
`docs/EXAMPLES.md` (one generated example per function), `docs/PERL.md`
(a Perl → scalar → plain Go phrasebook, whose every row
`TestMappingMatchesPerl` runs in all three and compares) and
`docs/PROCESS.md` (Iain's essay on how scalar and grammar were made, the
why behind these rules; his words, so change it only when he asks), with
`CHANGELOG.md` (what each release changed), `ROADMAP.md` (what scalar
means to do next, and why), `docs/changes/` (a record of each planned
change: the plan, and what happened) and `docs/MAINTAINING.md`
(branches, releases, and the git and GitHub work done by hand) beside
them.

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
- The PDFs (see the practice) are never committed here: module
  downloads would carry them.

Why: the documents are how people learn the package, its author included,
and gaps found by reading them have led to real features.

## How changes are made here

- **Show the Go.** When scalar code handles data that's already typed,
  show the plain Go version beside it and recommend it. scalar belongs at
  the edges, where the shape isn't known yet.
- **Benchmarks:** reverted ideas and their numbers go in the reference.
- **Wash-ups** re-run the Perl differential tests to show nothing else
  changed.
- **The other repositories** whose `TODO.md` this one hands tasks to,
  and checks at a release, are grammar's and jdc-lab's, both on
  `develop`, and job-meta's, on `main`.
- **`TestChangelogUnreleased`** checks the CHANGELOG's Unreleased
  section. [MAINTAINING.md](docs/MAINTAINING.md) has the release steps
  and the git and GitHub work done by hand.
