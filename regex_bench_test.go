package scalar

import (
	"slices"
	"strings"
	"testing"
)

const quakeLine = `M 4.6 - 23 km SSW of Ridgecrest, CA | depth 8.2 km | felt 1342 | net ci | code 40249711`

var quakeText = strings.Repeat(quakeLine+"\n", 200)

// BenchmarkSExpand measures S with a replacement string that refers to the
// match, on USGS-feed-shaped text.
func BenchmarkSExpand(b *testing.B) {
	for _, bench := range []struct {
		name, input, pattern, repl, flags string
	}{
		{"one", quakeLine, `(\d+) km`, "${1}km", "r"},
		{"g", quakeLine, `(\d+) km`, "${1}km", "gr"},
		{"named", quakeLine, `net (?<net>\w+)`, "network ${net}", "r"},
		{"g/long", quakeText, `depth (\d+(?:\.\d+)?) km`, "$1 km deep", "gr"},
		{"nomatch", quakeLine, `(Anchorage)`, "<$1>", "gr"},
	} {
		s := Value(bench.input)
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				s.S(bench.pattern, bench.repl, bench.flags)
			}
		})
	}
}

// BenchmarkRegex runs each regex operation on USGS-feed-shaped text, once
// per engine in benchEngines. On its own, it measures the default engine;
// regex_engine_test.go adds the others.
func BenchmarkRegex(b *testing.B) {
	ops := []struct {
		name, pattern string
		run           func(rx *Regexp)
	}{
		{"M/captures", `^M (\d+\.\d) - (\d+) km (\w+) of ([^,]+), (\w\w)`, func(rx *Regexp) {
			Value(quakeLine).M(rx)
		}},
		{"M/nomatch", `Anchorage`, func(rx *Regexp) {
			Value(quakeLine).M(rx)
		}},
		{"M/g", `depth (\d+(?:\.\d+)?) km`, func(rx *Regexp) {
			Value(quakeText).M(rx, "g")
		}},
		{"Named", `^M (?<mag>\d+\.\d) - (?<dist>\d+) km`, func(rx *Regexp) {
			Value(quakeLine).Named(rx)
		}},
		{"S/expand", `(\d+) km`, func(rx *Regexp) {
			Value(quakeLine).S(rx, "${1}km", "gr")
		}},
		{"S/func", `\d+\.\d`, func(rx *Regexp) {
			Value(quakeLine).S(rx, func(m *AV) string { return "[" + m.Get(0).Str() + "]" }, "gr")
		}},
		{"Split", `\s*\|\s*`, func(rx *Regexp) {
			Value(quakeLine).Split(rx)
		}},
	}
	names := slices.Sorted(func(yield func(string) bool) {
		for name := range benchEngines {
			if !yield(name) {
				return
			}
		}
	})
	for _, op := range ops {
		for _, engine := range names {
			rx := benchEngines[engine](op.pattern)
			b.Run(op.name+"/"+engine, func(b *testing.B) {
				for b.Loop() {
					op.run(rx)
				}
			})
		}
	}
}

// benchEngines compiles a pattern for each engine under test.
var benchEngines = map[string]func(pattern string) *Regexp{
	"default": func(p string) *Regexp { return MustCompile(p) },
}
