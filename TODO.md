# TODO

## Say scalar is shared as is, and tidy what a visitor sees

**From:** grammar's session, 2026-11-08 (grammar 150faae65fb614caa7f82eeae4044bb7f76c28eb)
**Waiting on:** nothing

**What:** bring scalar's public face into line with grammar's, as grammar
did in the commit above. scalar is made for Iain's own work and shared as
is: its public files should say so, and promise no support beyond that.

**What grammar's session found, so it isn't rediscovered:**

- **No CONTRIBUTING.md or SECURITY.md is needed.** The account-wide
  repository [nukopian/.github](https://github.com/nukopian/.github)
  provides both for every repository without its own, and GitHub links
  them from the repository's pages. The README's link to
  `https://github.com/nukopian/scalar/security/policy` resolves through
  it, and private vulnerability reporting is on (check with
  `gh api repos/nukopian/scalar/private-vulnerability-reporting -q .enabled`).
  Iain agreed on 2026-11-08 to keep one shared copy, not per-repository
  ones, so they can't drift.
- **The README's code should come first.** grammar's sat 70 lines down;
  scalar's starts at line 16, which is fine. Check it still does, and
  that its prose doesn't repeat itself once read top to bottom.
- **Changes on `develop` aren't seen until a release.** GitHub's front
  page shows `main`, and pkg.go.dev the latest tag.

**Steps:**

1. In the README's Status section, after the existing paragraph, add the
   paragraph grammar has, with the same link:

   > It's made for my own work, and shared as is. Bug reports and feature
   > requests are welcome as issues, but pull requests aren't accepted; the
   > [contributing guide](https://github.com/nukopian/.github/blob/main/CONTRIBUTING.md)
   > says why.

2. Read the README top to bottom as a first-time visitor would, and fix
   only what gets in the way.
3. The two branches on GitHub untouched since 2026-10-26,
   `experiment/exact-integers` and `experiment/regex-engine`: check
   that their findings are recorded (a change record, or the reference's
   gaps). Ask Iain before deleting either, as a deleted branch can't be
   restored by anyone without a local copy.
4. Ask Iain whether to set the repository's homepage to
   `https://pkg.go.dev/github.com/nukopian/scalar`, as DBIx-Squirrel's
   points to MetaCPAN. It's a GitHub setting:
   `gh repo edit nukopian/scalar --homepage https://pkg.go.dev/github.com/nukopian/scalar`.

**Done when:** the Status paragraph is in, the README is checked as above
and its PDF rebuilt and read, the branches are recorded and settled, the
homepage is decided, and grammar's session has been told nothing more
is needed from it.
