# Roadmap

What scalar means to do, and why. The [CHANGELOG](CHANGELOG.md) says
what it has done, and the [README](README.md) what it is now.

The API is settling towards 1.0; new items come from using scalar, and
from what its users ask for.

Items come in four groups, the most useful first in each:

- **Now:** being worked on, each with a plan detailed enough to pick up
  cold, in its [change record](docs/changes/README.md); never more than
  three, so that it stays a queue;
- **Next:** decided, but not started: what, why, and when it's done;
- **Later:** meant to be done, but not yet planned, in a line or two;
- **Ideas:** might be done, or never; kept so they aren't lost, in a
  line or two, newest last.

Each item has an id, its year and a serial, such as 2026-0003. The year
is when the item was added, and the serial counts from 0001, starting
again each year. An id is given when the item is added, whichever group
it goes in, and is kept as the item moves, so that 2026-0003 means the
same item in a commit message years later. It names the item's change
record, its line in the CHANGELOG ("(roadmap 2026-0003)") and any branch
for it. A `TODO.md` task with a record takes an id from the same
sequence, though it never appears here. Items numbered 1 to 5 before
2026-11-09 kept their numbers as serials, so "roadmap 3" in older
commits is 2026-0003. Last id: 2026-0011.

Each item has a size too, a guess at how much work it is, as a T-shirt
size and its story points: XS 1, S 2, M 3 and L 5. Sizes compare items
with each other; they aren't a measure of time. An item bigger than L
is split into items that aren't, each saying what it was split from
("Split from:"). An item that can't be done before another says so
("Depends on:"), and is never in a group ahead of it.

An item whose questions aren't all settled says so, and its record
lists them. Work on it starts only once each is settled, or moved to
another item or task.

When an item ships, it leaves here for the CHANGELOG, whose line for it
ends with its id and links to its record. An item that's dropped is
deleted, and the commit that drops it names its id.

## Now

Nothing yet.

## Next

### 2026-0011. Guide chapters: text, counting, output and speed

**Serves:** learning

**Size:** L (5)

**Depends on:** 2026-0008 and 2026-0009

**What:** the rest of the guide: cleaning text and regex work;
counting and grouping, with sets, bags and the list utilities; shaping
output as JSON, YAML and dumps; making it faster, with `Linear` and
locking; and growing out of scalar, replacing it with plain Go once the
shape is known.

**Why:** these are the tasks scalar's users bring after reading their
data, and the last says plainly where scalar stops being the right
tool.

**Done when** each chapter's recipes are generated and checked as
2026-0008 sets out.

## Later

Nothing yet.

## Ideas

An idea is a heading with its id, an **Added:** date, a **Size:**
line, and a line or two on what and why. Each wash-up prunes them: each
is kept, promoted to Later or Next with its id, or dropped.

Nothing yet.
