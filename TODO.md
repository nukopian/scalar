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
