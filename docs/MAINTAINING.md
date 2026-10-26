# Maintaining

How scalar is kept: its branches, releases, and the git and GitHub work
done by hand. It's for whoever maintains it, Iain and Claude now. Every
command can be pasted as it stands. The few that need a value say so on
the line before, and set it in a variable on their first line.

[grammar](https://github.com/nukopian/grammar) is kept the same way, and
has [its own guide](https://github.com/nukopian/grammar/blob/main/docs/MAINTAINING.md),
with what's particular to it: the local scalar, the Raku tests, and
comparing job-meta's output before and after a release.

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
- [ROADMAP.md](../ROADMAP.md) holds what scalar means to do: features
  and outcomes, in Now, Next and Later, and Ideas, kept so they aren't
  lost. It's public, on both branches, and in every release. Now holds
  three items at most, so the roadmap stays a queue to pull from, not a
  list pushing work; an item in Now has a plan written to be picked up
  cold; one leaves when it ships, in the commit that adds it to the
  CHANGELOG, whose line ends "(roadmap 7)". Every item has a number, the
  next one when it's added (the roadmap's opening says which), kept as it
  moves between groups and never used again; a heading such as
  `### 7. Title` holds it, as a Markdown numbered list would renumber
  itself. An idea is that heading, the date it was added, and a line or
  two, and is added in a `docs:` commit whenever one comes up.
- `TODO.md`, when there is one, lists work planned for a later session:
  the concrete steps of a roadmap item being worked on, or tasks from
  other repositories' sessions. Its tasks come before the roadmap's
  items. It's committed on `develop` and gone before a release, so it
  never reaches `main`: each task is done, or, if it's a feature rather
  than a step, moved word for word into the roadmap
  ([CLAUDE.md](../CLAUDE.md#planned-work) has the rules).

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

First bring `develop` up to date and re-read `TODO.md`, if there is one:
sessions in grammar, job-meta and jdc-lab add tasks to it, mid-session
too, and a commit that completes a task removes it
([CLAUDE.md](../CLAUDE.md#planned-work)). Each task opens with **From**
(the session that added it, the date, and the commit that prompted it,
or the last commit and "work uncommitted") and **Waiting on** (what it
needs first, or nothing). A task this session adds to another
repository's `TODO.md` is committed and pushed there at once, after the
same pull, so `TODO.md` only ever changes through commits.

```
git pull --ff-only
if [ -e TODO.md ]; then cat TODO.md; else echo "no TODO.md"; fi
```

Then run these, and check that each one succeeds. Don't pipe `go test`
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
commit, and update [ROADMAP.md](../ROADMAP.md) if the change ships, starts
or drops a feature.

## Making a release

A release is a signed, annotated tag on `main`. Before v1, a breaking
change bumps the minor version (v0.5.1 to v0.6.0) and anything else the
patch (v0.5.1 to v0.5.2). Claude asks before tagging.

1. **Check `develop`:** CI is green on its last commit, there's no
   `TODO.md`, and the history has no secrets.

   ```
   cd ~/Git/github.com/nukopian/scalar
   git switch develop
   git pull --ff-only
   gh run list --branch develop --limit 1
   if [ -e TODO.md ]; then cat TODO.md; else echo "no TODO.md"; fi
   gitleaks git --redact --no-banner .
   ```

   If `TODO.md` is there, its tasks come first: do them, or move a
   feature word for word into ROADMAP.md, or a decision awaiting Iain
   to a Claude Doc, and delete the file before going on. Check
   ROADMAP.md too: what shipped is out of it, and Now is what's really
   being worked on. Prune the Ideas: list them by number, and Iain
   keeps, promotes or drops each. A promoted idea keeps its number; a
   dropped one is deleted, and the commit message names its number.

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

4. **Move `main` and tag it.** First check that every check run on the
   commit has completed successfully. `gh run watch` can report a run
   as passed while one of its jobs is still marked in progress, and
   `main`'s protection refuses the push until it isn't. Each check is
   listed once, from `develop`'s push:

   ```
   cd ~/Git/github.com/nukopian/scalar
   gh api "repos/nukopian/scalar/commits/$(git rev-parse develop)/check-runs" --jq '.check_runs[] | "\(.name) \(.status) \(.conclusion)"'
   ```

   Every line ends `completed success`; if one doesn't, see
   [When something goes wrong](#when-something-goes-wrong). Then the
   merge, which is a fast-forward, so `main` gets exactly the commits CI
   tested on `develop`, with their signatures. Set the version on the
   first line:

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

7. **Find the tasks waiting on it.** Look in the other repositories'
   `TODO.md` for tasks whose **Waiting on** this release meets, and tell
   Iain which can now be done. Nothing else tells their sessions it has
   landed. grammar's and jdc-lab's are on `develop`, and job-meta's on
   `main`, so the loop reads each from its remote branch, whatever the
   working copy has checked out:

   ```
   for r in grammar:develop jdc-lab:develop job-meta:main; do
     repo=${r%%:*}; branch=${r#*:}
     d=~/Git/github.com/nukopian/$repo
     git -C "$d" fetch -q
     if git -C "$d" cat-file -e "origin/$branch:TODO.md" 2>/dev/null; then echo "$repo:"; git -C "$d" show "origin/$branch:TODO.md" | grep -E '^## |^\*\*Waiting on'; else echo "$repo: no TODO.md"; fi
   done
   ```

For grammar's release steps, see
[its guide](https://github.com/nukopian/grammar/blob/main/docs/MAINTAINING.md#making-a-release).

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
- **A check stuck "in progress"** after all its steps passed, as
  grammar's Windows job was on 2026-10-22: GitHub never marked it
  complete. Re-run that job alone. This finds every unfinished check on
  `develop`'s commit and re-runs it:

  ```
  cd ~/Git/github.com/nukopian/scalar
  gh api "repos/nukopian/scalar/commits/$(git rev-parse develop)/check-runs" --jq '.check_runs[] | select(.status != "completed") | .details_url' | sed -E 's|.*/runs/([0-9]+)/job/([0-9]+).*|\1 \2|' | while read run job; do gh run rerun "$run" --job "$job"; done
  ```

  Then check the check runs again, as in step 4.
- **`main` refused after the tag went up:** `git push origin main "$v"`
  sends both, so the tag can be published while `main` is refused. Leave
  the tag as it is: it points at the right commit. Fix the check, then
  push `main` to the tagged commit and make the release. Set the version
  on the first line:

  ```
  v=v0.5.2
  cd ~/Git/github.com/nukopian/scalar
  git switch main
  git merge --ff-only "$v"
  git push origin main
  gh release create "$v" --verify-tag --title "$(git tag -l --format='%(contents:subject)' "$v")" --notes-file "$TMPDIR/notes-$v.txt"
  git switch develop
  ```

## Security checks

These run without anyone starting them, and the README's Security
section tells users about them, so keep the two in step.

- **govulncheck** runs on every push, in `test.yml`, and every Monday on
  `main`, in `vuln.yml`, so that a vulnerability published after a
  release is found. A failed weekly run means a new vulnerability in
  what scalar calls: fix it on `develop` and release, as for any bug.
  GitHub turns off a public repository's scheduled workflows after 60
  days without activity. To check it's still on, and turn it back on:

  ```
  gh workflow list --all -R nukopian/scalar
  gh workflow enable vuln.yml -R nukopian/scalar
  ```

  To run it by hand, on `main`:

  ```
  gh workflow run vuln.yml -R nukopian/scalar
  ```

- **Dependabot alerts** are on, and **Dependabot security updates** are
  off on purpose: they open pull requests against `main`, which moves
  only at a release. An alert is fixed on `develop`, like anything else.
- **Secret scanning and push protection** are on: push protection
  refuses a push with a recognised secret in it, before gitleaks would
  see it.

To see the settings, and any open alerts:

```
gh api repos/nukopian/scalar --jq '.security_and_analysis | to_entries[] | "\(.key) \(.value.status)"'
gh api repos/nukopian/scalar/vulnerability-alerts --silent && echo "Dependabot alerts on"
gh api repos/nukopian/scalar/dependabot/alerts --jq '.[] | select(.state == "open") | "\(.number) \(.security_advisory.summary)"'
gh api repos/nukopian/scalar/secret-scanning/alerts --jq '.[] | select(.state == "open") | "\(.number) \(.secret_type_display_name)"'
```

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
with List::UtilsBy, YAML::PP and autovivification.
