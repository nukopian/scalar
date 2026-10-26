package scalar

import (
	"strings"
	"testing"
)

const quakeLine = `M 4.6 - 23 km SSW of Ridgecrest, CA | depth 8.2 km | felt 1342 | net ci | code 40249711`

// BenchmarkSExpand measures S with a replacement string that refers to the
// match, on USGS-feed-shaped text.
func BenchmarkSExpand(b *testing.B) {
	quakeText := strings.Repeat(quakeLine+"\n", 200)
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
