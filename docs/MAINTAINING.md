# Maintaining

How scalar and [grammar](https://github.com/nukopian/grammar) are kept:
their branches, releases, and the git and GitHub work done by hand. It's
for whoever maintains them, Iain and Claude now. Every command can be
pasted as it stands. The few that need a value say so on the line before,
and set it in a variable on their first line.

The repositories sit side by side, and job-meta, which uses both, sits
next to them:

```
~/Git/github.com/nukopian/scalar
~/Git/github.com/nukopian/grammar
~/Git/github.com/nukopian/job-meta
```

## Branches

- `develop` is where work happens. Every commit goes there, and CI runs
  on every push to it.
- `main` is the latest release, and nothing else. It moves only when a
  release is made, by a fast-forward to `develop`, and the release's tag
  is on it. Visitors to GitHub see `main`, so its README and docs describe
  what `go get` installs.
- There are no feature branches, for now: changes are made one at a time.

`main` is protected on GitHub. Pushes to it need signed commits, a linear
history (no merge commits) and CI passing on every job, and it can't be
force-pushed or deleted. This holds for admins too. To see the settings:

```
gh api repos/nukopian/scalar/branches/main/protection --jq '{signatures: .required_signatures.enabled, linear: .required_linear_history.enabled, checks: .required_status_checks.contexts, force_push: .allow_force_pushes.enabled, admins: .enforce_admins.enabled}'
```

For a fresh clone, or a checkout still on `main`, switch to `develop`:

```
cd ~/Git/github.com/nukopian/scalar
git fetch origin
git switch develop
git pull --ff-only
```

## Every commit

Run these first, and check that each one succeeds. Don't pipe `go test`
through `tail` or `grep` to judge it: the pipe's exit status is the last
command's, and hides a failure.

```
gofmt -l .
go vet ./...
go test ./...
```

Commits are signed, and use conventional prefixes (`feat:`, `fix:`,
`docs:`, `test:`, `ci:`, `refactor:`, `chore:`), saying why as well as what.
To check the signatures of the last ten (`G` is good, `N` is none):

```
git log --format='%h %G? %s' -10
```

Add what changed for users to `CHANGELOG.md` under Unreleased, in the same
commit.

## Making a release

A release is a signed, annotated tag on `main`. Before v1, a breaking
change bumps the minor version (v0.5.1 to v0.6.0) and anything else the
patch (v0.5.1 to v0.5.2). Claude asks before tagging.

1. **Check `develop`:** CI is green on its last commit, and the history
   has no secrets.

   ```
   cd ~/Git/github.com/nukopian/scalar
   git switch develop
   git pull --ff-only
   gh run list --branch develop --limit 1
   gitleaks git --redact --no-banner .
   ```

2. **Turn Unreleased into the release** in [CHANGELOG.md](../CHANGELOG.md):
   rename the heading to the version and date, as in
   `## v0.5.2 (2026-10-21)`, start the entry with a sentence that sums it
   up (it becomes the tag's title), and add an empty Unreleased above it.
   Commit (`docs: changes for v0.5.2`), push, and wait for CI.

3. **Write the tag message from the entry,** and read it. The entry is
   also the GitHub release's notes, and the tag message is the same with
   the version in front. Set the version on the first line:

   ```
   v=v0.5.2
   awk -v h="## $v (" 'index($0, h) == 1 {on = 1; next} /^## / {on = 0} on' CHANGELOG.md | sed '/./,$!d' > "$TMPDIR/notes-$v.txt"
   sed "1s/^/$v: /" "$TMPDIR/notes-$v.txt" > "$TMPDIR/tag-$v.txt"
   cat "$TMPDIR/tag-$v.txt"
   ```

4. **Move `main` and tag it.** The merge is a fast-forward, so `main` gets
   exactly the commits CI tested on `develop`, with their signatures. Set
   the version on the first line:

   ```
   v=v0.5.2
   git switch main
   git pull --ff-only
   git merge --ff-only develop
   git tag -s "$v" -F "$TMPDIR/tag-$v.txt"
   git push origin main "$v"
   gh release create "$v" --verify-tag --title "$(git tag -l --format='%(contents:subject)' "$v")" --notes-file "$TMPDIR/notes-$v.txt"
   git switch develop
   ```

   The release is the page GitHub shows for the tag, and the newest is
   marked Latest in the repository's sidebar. `go get` reads only the
   tag, so a tag without a release still installs, but GitHub would show
   an older version as the latest, as it did until 2026-10-20, when the
   releases from v0.3.0 on were added. `--verify-tag` refuses to make a
   tag of its own if the push didn't happen.

   If the merge says it can't fast-forward, `main` has a commit `develop`
   doesn't. Stop and find out why; don't make a merge commit.

5. **Check that the proxy serves it.** The first fetch makes the Go module
   proxy and checksum database record the tag, for good:

   ```
   v=v0.5.2
   GOPRIVATE= GOPROXY=https://proxy.golang.org go list -m "github.com/nukopian/scalar@$v"
   ```

6. **Bump what uses it.** A scalar release goes into grammar if grammar
   needs it, and both go into job-meta. Bump with `GOWORK=off`, so that
   the check is against the published module and not the local checkout
   that `go.work` points at. Set the version on the first line:

   ```
   v=v0.5.2
   cd ~/Git/github.com/nukopian/job-meta
   GOWORK=off go get "github.com/nukopian/scalar@$v"
   GOWORK=off go mod tidy
   GOWORK=off go test ./...
   ```

For grammar, it's the same, in `~/Git/github.com/nukopian/grammar`, with
`github.com/nukopian/grammar` in step 5.

## When something goes wrong

- **A bad commit pushed to `develop`:** fix it in a new commit, or revert
  it with `git revert`, and push. Don't force-push: Iain reviews what's
  pushed, and a rewrite moves it under him.
- **A mistake in a pushed commit message:** leave it, and correct it in
  the next release's change log, as v0.4.0's does.
- **A bad release:** never move or delete a published tag. The proxy and
  checksum database keep it whatever happens to the repository, and
  anyone's `go.sum` records its hash. Release a fixed version instead. To
  steer people off the bad one, add a `retract` directive to `go.mod` in
  the fix (`retract v0.5.2 // why`), so `go get` skips it and `go list -m
  -u` warns about it.
- **A commit made on `main` by mistake,** before it's pushed: move it to
  `develop`, and put `main` back. Do this only if `main` hasn't been pushed
  since:

  ```
  git switch develop
  git cherry-pick main
  git branch -f main origin/main
  ```

- **A push to `main` refused:** the protection did its job. Push to
  `develop` instead, or, at a release, wait for CI to pass on the commit.

## Rewriting history

Rarely, history has to change: a file that shouldn't have been committed,
such as the PDFs once were, or a package moving to its own repository.
It's done in a fresh clone, never in the working checkout, with
`git filter-repo` (`brew install git-filter-repo`). filter-repo drops
signatures, so each commit is then signed again with `git commit-tree -S`,
keeping its author, committer and dates, and the commit ids that messages
quote are rewritten to the new ones.

Before pushing, run gitleaks on the rewritten history, and compare the
final tree with the old one (`git diff old-head new-head` shows nothing
but what was meant to change). Push with `--force-with-lease`, naming the
full hash the remote is expected to be at, and the branch spelled out in
full (`refs/heads/main`), so that nothing else can be overwritten. Claude
writes out that command with the real hash, ready to paste. Lift `main`'s
protection for the push, and put it back straight after.

## Moving a package into its own repository

grammar left job-meta this way, and uksoc is next. In outline:

1. Rename the package inside job-meta first, so the move itself is only an
   extraction, and cut its dependencies on the rest of job-meta (uksoc
   copied in punct's one function).
2. In a fresh clone, `git filter-repo` keeps only the package's files,
   with their history, and moves them to the new repository's root. Sign
   the commits again, as above.
3. Give the new repository what scalar has: `CLAUDE.md`, `internal/docref`
   (identical in every repository), CI with the gitleaks job,
   `scripts/readme-pdf.sh`, `.gitignore` with `*.pdf`, a LICENSE, and a
   `develop` branch.
4. Tag v0.1.0, then switch job-meta to import it, and delete the old
   copy.

A private repository's CI can't fetch another private one without a
deploy key. Everything uksoc needs is public now, so it needs none.

## Visibility

Before a repository goes public, scan its whole history, since all of it
is published:

```
gitleaks git --redact --no-banner .
```

Then, with Iain's say-so, which is his to give:

```
gh repo edit nukopian/scalar --visibility public --accept-visibility-change-consequences
```

## Tools

`go` (the version `go.mod` asks for, or newer), `gh`, `gitleaks`,
`git-filter-repo`, and for the PDFs, `pandoc` and Prince. grammar's Raku
tests need Rakudo 2025.11, and scalar's Perl tests Perl 5.36 or later,
with List::UtilsBy and YAML::PP.
