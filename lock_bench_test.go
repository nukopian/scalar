package scalar

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
)

// The benchmarks for what a lock might speed up (the reference's Locking
// section), on a realistic shape: a table of seismic statistics as an
// import gives it, every field a string, nested window → network →
// station → field. It's generated, from a fixed seed, in the shape of the
// ONS's ASHE pay table the measurements were first made on: 3 × 9 × 553 =
// 14,931 rows of 31 fields, a place name and 30 values, of which about
// 40% are decimals, 27% "x" (not computed), 19% whole numbers, 7.5%
// empty and 7% ".".

var (
	windows  = []string{"day", "week", "month"}
	networks = []string{"ak", "ci", "hv", "nc", "nn", "pr", "se", "us", "uw"}
	statKeys = []string{"events", "mag_median", "mag_median_change", "mag_mean", "mag_mean_change",
		"mag_p10", "mag_p20", "mag_p25", "mag_p30", "mag_p40", "mag_p60", "mag_p70", "mag_p75", "mag_p80", "mag_p90",
		"err_events", "err_median", "err_median_change", "err_mean", "err_mean_change",
		"err_p10", "err_p20", "err_p25", "err_p30", "err_p40", "err_p60", "err_p70", "err_p75", "err_p80", "err_p90"}
)

// station is the code of a network's i'th station: AAA, AAB and so on.
func station(i int) string {
	return string([]byte{byte('A' + i/676), byte('A' + i/26%26), byte('A' + i%26)})
}

var statTable = sync.OnceValue(func() *SV {
	r := rand.New(rand.NewPCG(1, 2))
	value := func() string {
		switch n := r.IntN(1000); {
		case n < 400:
			return fmt.Sprintf("%d.%d", r.IntN(100), r.IntN(10))
		case n < 670:
			return "x"
		case n < 860:
			return fmt.Sprint(r.IntN(100000))
		case n < 935:
			return ""
		default:
			return "."
		}
	}
	root := Hash()
	for _, w := range windows {
		for _, net := range networks {
			for i := range 553 {
				row := Hash("place", fmt.Sprintf("%d km %s of station %s, network %s", r.IntN(100), []string{"N", "NNE", "E", "SW", "W"}[r.IntN(5)], station(i), net))
				for _, k := range statKeys {
					row.Set(k, value())
				}
				root.Get(w).HV().Get(net).HV().Set(station(i), row)
			}
		}
	}
	return Value(root)
})

// table returns a fresh copy of the table, locked or not.
func table(b *testing.B, locked bool) *SV {
	t := statTable().Clone()
	if locked {
		t.Lock()
	}
	return t
}

var numFields = []string{"mag_median", "mag_mean", "mag_p10", "mag_p20", "mag_p25", "mag_p30", "mag_p40", "mag_p60", "mag_p70", "mag_p75", "mag_p80", "mag_p90"}

// Reading numbers stored as strings, as an import gives them: every row's
// magnitude median, mean and percentiles, as Num.
func benchNums(b *testing.B, locked bool) {
	t := table(b, locked)
	var rows []*SV
	for _, window := range t.HV().All() {
		for _, network := range window.HV().All() {
			for _, row := range network.HV().All() {
				rows = append(rows, row)
			}
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		sum := 0.0
		for _, row := range rows {
			h := row.HV()
			for _, f := range numFields {
				sum += h.Get(f).Num()
			}
		}
		_ = sum
	}
}

func BenchmarkNumsUnlocked(b *testing.B) { benchNums(b, false) }
func BenchmarkNumsLocked(b *testing.B)   { benchNums(b, true) }

// Looking up the same paths again and again, as a lookup table is used:
// 1,000 paths, each to a field of a row.
func benchAt(b *testing.B, locked bool) {
	t := table(b, locked)
	var paths []string
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		paths = append(paths, fmt.Sprintf("week.ci{%s}.%s", station(r.IntN(553)), numFields[r.IntN(len(numFields))]))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, p := range paths {
			_ = t.At(p)
		}
	}
}

func BenchmarkAtUnlocked(b *testing.B) { benchAt(b, false) }
func BenchmarkAtLocked(b *testing.B)   { benchAt(b, true) }

// Comparing two copies of the table: alike, and differing in one field.
func benchEqual(b *testing.B, locked, alike bool) {
	x, y := table(b, locked), table(b, false)
	if !alike {
		y.At("month.us{" + station(400) + "}.mag_median").Set("-1") // a station that exists: one field differs
	}
	if locked {
		y.Lock()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if x.Equal(y) != alike {
			b.Fatal("Equal is wrong")
		}
	}
}

func BenchmarkEqualAlikeUnlocked(b *testing.B)  { benchEqual(b, false, true) }
func BenchmarkEqualAlikeLocked(b *testing.B)    { benchEqual(b, true, true) }
func BenchmarkEqualDifferUnlocked(b *testing.B) { benchEqual(b, false, false) }
func BenchmarkEqualDifferLocked(b *testing.B)   { benchEqual(b, true, false) }
