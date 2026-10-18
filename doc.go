// Package scalar gives Go dynamic values for messy data: JSON, YAML and
// text whose shape you don't know yet. Read, clean and reshape it with
// tools borrowed from Perl and JSONPath, then convert it to your own types
// with [To].
//
// An [SV] is a scalar: undef, a boolean, an integer, a float, a string, a
// byte string, or a reference to an [AV] (array) or [HV] (hash). Scalars
// convert between string, numeric and boolean contexts the way Perl does.
// A missing key, a JSON null or a pattern that didn't match is undef, an
// ordinary value that passes along a chain, and writing through a path
// that isn't there creates it, as Perl's autovivification does.
//
//	feed, err := scalar.FromJSON(body)
//	place := feed.At("features[0].properties.place").Or("unknown").Str()
//	big := feed.Find("$.features[?@.properties.mag >= 4.5]")
//
// # Concurrency
//
// Nothing here is safe for concurrent use, not even reading: At on a
// missing path creates the levels on the way, and a hash caches its key
// order as you read it. Keep each structure to one goroutine, and share
// data as Go values ([To], [SV.Decode]), a copy made with [SV.Clone], or a
// structure you've locked with [SV.Lock].
//
// # Documentation
//
// The [README] gives an overview, the [how-to guide] has recipes for
// common tasks, the [reference] covers each feature in depth, the
// [examples] show every function and method with its output, and the
// [phrasebook] shows everyday tasks in Perl, with scalar and in plain Go.
//
// [README]: https://github.com/nukopian/scalar#readme
// [how-to guide]: https://github.com/nukopian/scalar/blob/main/docs/HOWTO.md
// [reference]: https://github.com/nukopian/scalar/blob/main/docs/REFERENCE.md
// [examples]: https://github.com/nukopian/scalar/blob/main/docs/EXAMPLES.md
// [phrasebook]: https://github.com/nukopian/scalar/blob/main/docs/PERL.md
package scalar
