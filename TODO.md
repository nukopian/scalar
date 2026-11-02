# To do

## Match the GitHub description to the README

**From:** jdc-lab's session, 2026-11-02 (scalar c6974f668e49981355c7ceba260bcc282d8b0625)
**Waiting on:** Iain's go-ahead, as it's a GitHub setting

**What:** the repository's description reads "Dynamic values for messy
data in Go." The README's opening line says more: "Read, clean and
reshape messy data in Go, then get back to typed Go." The description is
what link previews and GitHub search show.

**Steps:** once Iain agrees,

    gh repo edit nukopian/scalar --description "Read, clean and reshape messy data in Go, then get back to typed Go."

**Check:** `gh repo view nukopian/scalar --json description`. Remove
this task in a `docs:` commit, since nothing else changes.

## Relicense so pkg.go.dev shows the documentation

**From:** jdc-lab's session, 2026-11-02 (scalar c6974f668e49981355c7ceba260bcc282d8b0625)
**Waiting on:** Iain's choice of licence

**What:** pkg.go.dev shows "Documentation not displayed due to license
restrictions" for scalar. It shows documentation only when every
licence it detects is on its
[accepted list](https://pkg.go.dev/license-policy). It finds
Artistic-1.0-Perl and GPL-1.0, which aren't on it, as well as GPL-2.0,
which is. grammar, under Artistic-2.0, displays.

**What's known:** every commit is by one author, so relicensing needs
no one else's agreement. Published versions keep their terms. The
choices weighed are Artistic-2.0 (as grammar), Artistic-2.0 OR
GPL-2.0-or-later, MIT, Apache-2.0, or keeping the current terms.

**Steps,** once the licence is chosen, in one `feat:` commit:

1. Replace LICENSE with the new text and SPDX line. Drop its stale
   sentence about "the grammar package in grammar/", since grammar is
   now a separate repository.
2. Update the README's Licence section, and add a CHANGELOG line under
   Unreleased saying the licence changes from the next release.
3. Add a task to grammar's `TODO.md` to update its LICENSE, which
   describes scalar's terms.

**Check:** after the release, pkg.go.dev shows the API for the new
version. Request a fetch from the page if it still shows the old one.
