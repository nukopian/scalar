# Roadmap

What scalar means to do, and why. The [CHANGELOG](CHANGELOG.md) says
what it has done, and the [README](README.md) what it is now.

Items come in three groups, the most useful first in each:

- **Now:** being worked on, each with a plan detailed enough to pick up
  cold; never more than three, so that it stays a queue;
- **Next:** decided, but not started: what, why, and when it's done;
- **Later:** ideas worth keeping, in a line or two.

When an item ships, it moves from here to the CHANGELOG.

## Now

Nothing yet.

## Next

Nothing yet.

## Later

- **Possibly: a regex engine chosen per pattern.** Being considered, not
  decided. `CompileWith(scalar.Stdlib, pattern)` would match with Go's
  regexp, 1.2 to 2.6 times as fast in the experiment's benchmarks and
  never backtracking, at the cost of lookaround, backreferences and
  Unicode `\w`; other engines, such as coregex, could plug in the same
  way. The experiment, with its tests and measurements, is on the
  [`experiment/regex-engine`](https://github.com/nukopian/scalar/tree/experiment/regex-engine)
  branch.
