# Changes

What changed in each release of scalar, newest first. Each release's
entry is taken from its signed tag message. Versions follow
[semantic versioning](https://semver.org): before v1, a breaking change
bumps the minor version, as v0.4.0 and v0.5.0 did.

Work collects under Unreleased on the `develop` branch, and becomes a
release's entry when `main` is moved to it and tagged
([docs/MAINTAINING.md](docs/MAINTAINING.md) has the steps).

## Unreleased

Nothing yet.

## v0.5.8 (2026-10-25)

Documentation only: where Array flattens Go maps, and a roadmap

No change in behaviour or in the API: only the docs and one doc comment.

- Docs: the reference's [Lists flatten](docs/REFERENCE.md#lists-flatten-as-in-perl)
  has a table of what a Go programmer might assume, starting with
  `scalar.Array(m1, m2)`, which is one list of two Go maps' keys and
  values, not an array of two hashes. `Array`'s doc comment says so too.
- Docs: a [roadmap](ROADMAP.md) says what's planned, and why, and the
  README links to it. It starts empty.

## v0.5.7 (2026-10-25)

Autovivify(false), a mode in which reading creates nothing

Adds to the API; nothing that compiled before stops compiling. One
change in behaviour: reading JSON or YAML into a locked scalar returns
an error, where it panicked.

- New: `Autovivify(false)`, on scalars, arrays and hashes, is the CPAN
  pragma's `no autovivification` for a value and everything it refers
  to: `Get`, `At`, `HV()`, `AV()`, `Exists` and `Delete` create nothing,
  and a missing level reads as undef, so data probed for optional fields
  writes out as it was read. A write still creates the levels it needs,
  and values stored later take the setting. `Autovivify(self, deeply)`
  sets a value and what's below it apart; `Autovivifies` and
  `AutovivifiesDeeply` report them. With `Lock` as well, a missing key
  reads as undef instead of panicking. `TestAutovivifyOffMatchesPerl`
  checks nearly 10,000 subscripts against the pragma.
- Changed: `UnmarshalJSON` and `UnmarshalYAML` into a locked scalar, as
  `json.Unmarshal` and `yaml.Unmarshal` call them, return an error with
  Perl's message, `Modification of a read-only value attempted`, instead
  of panicking, and leave the scalar as it was.
- Fixed: `UnmarshalYAML` wrote into a locked scalar, and YAML read into a
  missing element, such as `yaml.Unmarshal(data, h.Get("q"))`, was lost
  rather than stored, as JSON is.
- Docs: `Lock`'s comment said reads that would autovivify panic; that's
  so for a locked hash, but a locked undef reads through as undef.

## v0.5.6 (2026-10-22)

The Dump comparison with Perl needs Data::Dumper 2.188

None of the changes is to the code or the API.

- Docs: the reference said a boolean dumps as `!!1` or `!!0` "as in
  Perl 5.36 and later". It's Data::Dumper 2.188 and later, which comes
  with Perl 5.38; 5.36's Data::Dumper prints `1` and `''`. Dump itself is
  unchanged. `TestDumpMatchesPerl` now skips, saying why, on a Perl
  whose Data::Dumper is older, instead of failing.

## v0.5.5 (2026-10-22)

A weekly vulnerability scan, and a Security section in the README

None of the changes is to the code or the API.

- Security: govulncheck also runs every Monday on `main`, the latest
  release, so a vulnerability published after it is found, not only at
  the next push. GitHub's Dependabot alerts, secret scanning and push
  protection are on. The README has a Security section saying what's
  checked and how often, and how to report a vulnerability.
- The release steps check every check run on the commit before pushing
  `main`, and say what to do about a check stuck "in progress" or a
  `main` refused after its tag went up, both of which happened at
  grammar v0.1.5.

## v0.5.4 (2026-10-22)

The CI badge shows main's status

None of the changes is to the code or the API.

- The README's CI badge names `main`, so it shows the latest release's
  status: without a branch, GitHub's badge said "no status".

## v0.5.3 (2026-10-22)

Planned work goes in TODO.md on develop, and the maintainer's guide is scalar's own

None of the changes is to the code or the API.

- Planned work for a later session is listed in a `TODO.md` on
  `develop`, checked at the start of each session, and gone before a
  release, so it never reaches `main`. CLAUDE.md has the rules, and the
  release steps check for it.
- [docs/MAINTAINING.md](docs/MAINTAINING.md) is scalar's alone, now that
  grammar has its own guide, and `scripts/readme-pdf.sh` builds its PDF.

## v0.5.2 (2026-10-22)

A change log, a maintainer's guide, and the essay on how scalar was made

None of the changes is to the code or the API; this release publishes
the documentation and CI work since v0.5.1.

- Docs: [The process](docs/PROCESS.md), Iain's essay on how scalar and
  grammar were made, linked from the README and CLAUDE.md.
- CI scans the whole history for secrets with gitleaks on every push.
- This change log, a `develop` branch for work (`main` is the latest
  release, and moves only at the next), GitHub releases for every tag,
  and [docs/MAINTAINING.md](docs/MAINTAINING.md), the maintainer's guide.
- The repository is public, from 2026-10-20.

## v0.5.1 (2026-10-19)

HV.Set stores a new key without a placeholder. None of the changes is to
the API.

- HV.Set stores a new key directly, without first making the placeholder
  and closure Get makes for a missing key: two allocations fewer per new
  key. Behind fastPaths, compared with the general code by
  TestSetFastPathSame. Through grammar's benchmarks, a pay parse 2%
  faster, 8 allocations fewer.
- Docs: "What you might assume" says plainly that a missing key or index
  is never a panic for Exists and Delete, only a value of the wrong kind,
  as in Perl; and the reference and phrasebook give Perl's `@{ X }` and
  `%{ X }` as `X.AV()` and `X.HV()`.

## v0.5.0 (2026-10-19)

DeleteOK, the forgiving delete, says whether it removed anything.

- **Breaking:** DeletePath is now DeleteOK, on scalars, hashes and arrays,
  returning the value it removed and Go's comma-ok: ok is true if there
  was something there to remove, even if its value was null. Like GetOK,
  it never creates anything and never panics on the shape of the data.
  `s.DeletePath(k1, k2)` is `v, _ := s.DeleteOK(k1, k2)`.
- Docs: the comments in the read-through example under "What you might
  assume" line up.

## v0.4.0 (2026-10-19)

Get, Exists and Delete take a route of keys. Several of the changes are
breaking.

- Get, Exists and Delete take one key or a route of them, on scalars,
  hashes and arrays: `s.Get("content", 0, "text")` is
  `$s->{content}[0]{text}`. Several keys are always a route, never a
  slice; a slice is Values. A hash's first key is a string and an
  array's an int.
- **Breaking:** Path is gone: use Get with the same keys. PathOK is now
  GetOK.
- **Breaking:** MustPath, CompilePath and PathExpr are gone: At caches its
  parsing, and a benchmark showed compiled paths no longer paid.
  CheckPath(path) checks a path from outside the program.
- **Breaking:** Exists and Delete on a scalar panic where Perl dies, on
  something that isn't the hash or array the key asks for.
- Find on arrays and hashes, as on scalars.
- GetOK, AtOK and Find on a nil hash or array find nothing, as on a nil
  scalar.
- The reference presents nested access by mode, Perl's or looking
  without creating anything, with a table of what you might assume.

The commit adding more keys to AV and HV says its test covers 560 routes;
it covers 2,628.

## v0.3.0 (2026-10-19)

Get, Exists and Delete on a scalar, as Perl subscripts.

## v0.2.0 (2026-10-18)

scalar as a project of its own: its first release as the module
`github.com/nukopian/scalar`, with its history from 2026-10-11, when it
began inside job-meta.

- **Standalone module**, extracted with its history from the project it
  started in. No API changes on the way.
- **Go 1.25 or later.** The floor is set by `golang.org/x/text` v0.41.0,
  the oldest without known vulnerabilities.
- **Documentation**, all on earthquake data: a short README, a how-to
  guide, a reference, an example for every function and method, and a
  Perl → scalar → Go phrasebook whose every row is tested three ways.
- **Tested** against Perl itself, the JSONPath Compliance Test Suite and
  fuzzing, with CI on Linux, macOS and Windows, govulncheck and
  staticcheck.
