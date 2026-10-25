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
