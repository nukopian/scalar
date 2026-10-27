# TODO

## Export the path parser, for grammar's CheckPath

**From:** grammar's session, 2026-10-27 (grammar c1094f72bbe7031b3faf2fa27576ece30059f51e)
**Waiting on:** nothing

**What:** export the parser that `At` uses, so another package can
read a path into its steps exactly as `At` does: for example
`ParsePath(path string) ([]PathStep, error)`, each step saying whether
it's a hash key (a name, `{key}`, `["key"]`), an array index (`[0]`,
`[-1]`), or either (a number after a dot, as in `content.0`), with its
key and index. The names are scalar's to choose.

**Why:** grammar's roadmap 16 is a `(*Grammar).CheckPath` that says
whether a path, such as `m.At("per.weak")`, can ever be defined in a
match of a grammar, as a misspelt capture otherwise reads undef and
fails silently. It walks the path's steps through the grammar's
captures, so it needs them as `At` sees them. Copying `parseAt` into
grammar would let the two drift: a path `At` reads one way, checked
another. The plan is in grammar's
`docs/changes/2026-10-27-match-paths-checked-against-the-grammar.md`.

**What's known:** `parseAt` (`at.go`) already returns `[]atStep`, with
`stepKey`, `stepIndex` and `stepEither`, and `CheckPath` and
`template.go` call it; `compileAt` caches the steps for `At`, `AtOK`
and `DeleteAt`. grammar needs, for each step, its kind, its key (the
digits, for either) and its index, and nothing else.

**Requirements:**

1. One parser: `At`, `AtOK`, `DeleteAt`, `CheckPath` and templates go
   on using the same code as the export, so they can't disagree.
2. A malformed path returns the same error `CheckPath` does, offset
   and all; `CheckPath` may become a wrapper.
3. The steps returned are the caller's: a copy, not `atCache`'s slice,
   so changing them can't change what `At` does.
4. Go 1.25, as `go.mod` says.
5. An example, the reference beside `At` and `CheckPath`, and a
   CHANGELOG line; then a release, as grammar requires tagged versions.

**Not needed yet:** grammar may later check the fields of a struct
passed to `To` the same way (a `CheckTo[T]`), which would need how `To`
maps a field to a key. No change for that now; worth keeping in mind
when naming things.

**Done when** the parser is exported, tested and documented, in a
tagged release; then tell Iain grammar's TODO task "Check match paths
against the grammar (roadmap 16)" can go ahead.
