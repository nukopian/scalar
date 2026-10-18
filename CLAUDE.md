# Working on scalar

These are the working rules for this repository. scalar is developed by
Iain Campbell with Claude (see the README's Acknowledgements), and these
rules are how that partnership has run since the first commit. Each rule
comes with the reason for it.

## What scalar is for

It aims to be easy to use, to give people a low barrier into Go, and to
get a proof of concept to results quickly. Raw speed comes second. It's a
half-way house for loosely shaped data (API and LLM JSON, YAML, messy
text) on the way to typed Go. Prefer clearer, more forgiving APIs and
good error messages over speed, unless a measurement shows a real cost.

## Perl is the baseline, and not the only source

- **Where Perl has a feature, match Perl's behaviour.** Where Go can't
  follow Perl, deviate pragmatically, and document each deviation in the
  README, in the feature's section or under Caveats. Raise any new
  deviation with Iain before building it.
- **Other features are borrowed from elsewhere**, as the README's "Why
  scalar exists" describes:
  - `Then`, `When`, `Or` and `OrSet` adapt the Maybe/Option style of
    programming to scalar's undef. They don't copy any one language's API,
    Scala's included. `TestThenMonadLaws` checks the monad laws, the rules
    that style itself sets.
  - XPath and JSONPath for `Path`, `At`, `DeleteAt` and `Find`, with
    Python's slices.
  - CPAN modules for sets and bags.
  - Go's iterators and generics for `All` and `To`.

  A new borrowing names its source in the README. Where the source is
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
  results (`*_test.go` that call `perl`). They skip without Perl, and two
  skip without List::UtilsBy or YAML::PP.
- Fuzz targets: `FuzzReadJSON`, `FuzzNumericPrefix` and `FuzzFind`. Run
  the relevant one after changing the code it covers, for example
  `go test -run '^$' -fuzz FuzzFind -fuzztime 60s`.
- Fast paths (`fastpath.go`, `fastjson.go`) sit behind the `fastPaths`
  switch. Each one has a test that runs with it on and off and compares
  the output, and a planted-bug check that the comparison catches a
  difference.
- The lock benchmarks need the ONS ASHE pay table. Set `ASHE_PAY_TSV` to
  its path, or they skip.
- scalar supports Go 1.25 and later, as `go.mod` says, and CI tests both
  1.25 and the newest Go. Don't use newer standard-library features unless
  raising the minimum is a deliberate decision. Dependencies set the floor
  too, and so does security: `golang.org/x/text` v0.41.0 is the oldest
  without known vulnerabilities (GO-2026-5970 reaches the case functions),
  and it needs Go 1.25. CI runs govulncheck.
- Run `gofmt -l .`, `go vet ./...` and `go test ./...` before every commit.
  CI (`.github/workflows/test.yml`) runs the same on every push, with the
  race detector and the Perl modules installed, and fails if a Perl test
  skips.

## Documentation keeps up with the code

A feature isn't done until its documentation is.

The documents: `README.md` (what scalar is, a synopsis, the rationale),
`docs/HOWTO.md` (tasks), `docs/REFERENCE.md` (each feature in depth) and
`docs/EXAMPLES.md` (one generated example per function).

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
  doesn't, and record reverted ideas and their numbers in the README so
  they aren't retried blind. Predictions here have been wrong more than
  once.
- **Wash-ups.** After a run of features, review them together for crashes,
  memory, correctness, inefficiency and slop. Reproduce each finding
  before reporting it, rank the findings, and write failing tests before
  fixes. Re-run the Perl differential tests to show nothing else changed.
- **Dogfooding.** Use scalar in real programs, note where it fought you,
  and fix the frequent friction. Check the fix by reworking those
  programs with their output unchanged. Do this alongside each wash-up.
- **Show the Go.** When scalar code handles data that's already typed,
  show the plain Go version beside it and recommend it. scalar belongs at
  the edges, where the shape isn't known yet.

## Commits

- Conventional commits with a scope, e.g. `feat(scalar): ...` or
  `docs(scalar): ...`. Say why as well as what.
- Claude commits and pushes changes once they're done and checked, and
  Iain reviews them. Commits are signed.
- Every commit ends with a `Co-Authored-By` trailer for the Claude model
  that worked on it, currently
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Keep the README's Acknowledgements accurate: credit goes where the work
  was done.
