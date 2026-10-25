# To do

Work planned for the next session. CLAUDE.md's
[Planned work](CLAUDE.md#planned-work) has the rules: ask whether each
task should still be done, remove it in the commit that completes it, and
delete this file with the last one. It must be gone before a release.

## A mode in which reading creates nothing, after Perl's `no autovivification`

**What:** a way to mark a structure so that reading it never creates
anything: `Get`, `At`, `HV()`, `AV()` and the rest give undef for a
missing key, index or level, and leave the structure as it was, rather
than autovivifying, as now, or panicking, as a locked hash does. The
model is the CPAN pragma
[autovivification](https://metacpan.org/pod/autovivification): its
`no autovivification;` turns off autovivification on fetch, `exists`
and `delete`, and leaves it on for store unless asked. Iain wants the
mode to be read-only too, so decide with him how it relates to `Lock`:
a separate switch, such as `NoVivify()`, that `Lock` can be combined
with, or a variant of `Lock` whose missing reads give undef instead of
panicking.

Perl's pragma is lexical; Go has no lexical scope, so the mode has to
belong to the value, reaching all the way down, as `Lock` does. That,
and the name, are deviations to raise with Iain before building, as
CLAUDE.md says. Decide too whether values stored into a marked structure
later are marked as well.

**Why:** autovivification on read surprises people, and the surprise
outlives the read. grammar (github.com/nukopian/grammar, 2026-10-25)
met it twice:

- The undef that means no match became a hash when read through:
  `m.At("made")` and then `m.Defined()` reported a match, and
  `m.String()` gave `HASH(0x…)`. grammar now returns a locked undef,
  which reads through quietly, as `lock_test.go` ("A locked undef
  doesn't autovivify when dereferenced") checks.
- Reading a capture a match doesn't have, `m.At("period.~")`, leaves an
  empty `period` behind. `AtOK` and `Exists` then find it, and
  `MarshalJSON` writes `"period":{}`. Raku creates nothing on a read, so this
  is listed under grammar's "Differences from Raku". `Lock` can't fix
  it: a locked hash panics on a missing key ("Attempt to access
  disallowed key 'period' in a restricted hash"), so `At`, `Get`, and
  grammar's `Values` and `To`, which read `made`, all panic. With this
  mode, grammar would mark each finished match in `options.result`
  (grammar.go), and the difference would go.

The same holds for any data read from JSON or YAML and then probed for
optional fields before it's written out again.

**Files:** `sv.go` (`vivify`, `HV`, `AV`, the `noCreate` and `ro`
markers), `hv.go` and `av.go` (`get`), `at.go` (`walk`), `lock.go` as the
pattern for a mode that reaches all the way down; tests beside them, an
example in `example_sv_test.go`, a differential test against Perl with
`autovivification` installed by `cpanm -L` into a temporary directory,
as CLAUDE.md describes; `docs/REFERENCE.md` (Nesting and
autovivification, and Locking), a row in `docs/PERL.md`, and
`CHANGELOG.md` under Unreleased. While there, `Lock`'s doc comment says
reads that would autovivify panic, which is true of a locked hash but
not of a locked undef: say so.

**Done when:** a marked structure, read through any accessor at any
depth, gives undef and is unchanged (`Equal` to a clone taken before,
and the same `MarshalJSON`); the Perl differential test agrees with
`no autovivification` on fetch, `exists` and `delete`; writes behave as
agreed with Iain; and the reference and PDF
(`scripts/readme-pdf.sh docs/REFERENCE.md`) read well. Then, in grammar,
mark each match, add a Raku test that reads a missing capture and
compares, and remove the "Reading a missing capture adds it to the
match" row from its Differences.
