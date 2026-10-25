# To do

Work planned for the next session. CLAUDE.md's
[Planned work](CLAUDE.md#planned-work) has the rules: ask whether each
task should still be done, remove it in the commit that completes it, and
delete this file with the last one. It must be gone before a release.

## Three more Planned work rules, agreed with Iain

**From:** jdc-lab's session, 2026-10-25 (jdc-lab 5e5f6be33efaf551697bda1e87c5a4d8d0f7c3f6)
**Waiting on:** nothing

**What:** add three rules to CLAUDE.md's Planned work, worded to match
it, and to docs/MAINTAINING.md where it covers TODO.md and releases.
jdc-lab 5e5f6be33efaf551697bda1e87c5a4d8d0f7c3f6 has its version.

1. A task a session adds to another repository's TODO.md is committed
   and pushed there at once, after `git pull --ff-only`, as its own
   `docs:` commit (on `develop` for grammar and scalar, `main` for
   jdc-lab). TODO.md changes only through commits. A code fix made in
   another working copy to get past a bug may still stay uncommitted.
2. If the work that prompted a task isn't committed yet, From gives the
   last commit and adds "work uncommitted".
3. A release's last step: look in the other repositories' TODO.md for
   tasks waiting on it, and tell Iain which can now be done.

**Why:** an uncommitted task in another repository's TODO.md makes that
repository's `git pull --ff-only` refuse to run when the incoming commits
touch TODO.md, and the session that finishes the task never sees it in a
commit. And nothing tells a session that a release it's waiting on has
landed.

**Done when:** CLAUDE.md and MAINTAINING.md have the three rules, and
this task is removed in the same commit.
