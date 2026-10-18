package scalar

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"testing"
)

// The benchmarks for what a lock might speed up (the README's Locking
// section), on a realistic shape: the ONS's ASHE pay table (pay.tsv,
// 14,931 rows of 34 fields) as an import gives it, every field a string,
// nested period → sheet → code → field. It isn't in this repository; set
// ASHE_PAY_TSV to its path to run them, or they skip.

var payTable = sync.OnceValue(func() *SV {
	f, err := os.Open(os.Getenv("ASHE_PAY_TSV"))
	if err != nil {
		return nil
	}
	defer f.Close()
	root := Hash()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	var header []string
	for sc.Scan() {
		fields := strings.Split(sc.Text(), "\t")
		if header == nil {
			header = fields
			continue
		}
		row := Hash()
		for i, name := range header[3:] {
			row.Set(name, fields[3+i])
		}
		root.Get(fields[0]).HV().Get(fields[1]).HV().Set(fields[2], row)
	}
	return Value(root)
})

// table returns a fresh copy of the pay table, locked or not.
func table(b *testing.B, locked bool) *SV {
	if payTable() == nil {
		b.Skip("set ASHE_PAY_TSV to the pay table's path")
	}
	t := payTable().Clone()
	if locked {
		t.Lock()
	}
	return t
}

var numFields = []string{"median", "mean", "p10", "p20", "p25", "p30", "p40", "p60", "p70", "p75", "p80", "p90"}

// Reading numbers stored as strings, as an import gives them: every row's
// median, mean and percentiles, as Num.
func benchNums(b *testing.B, locked bool) {
	t := table(b, locked)
	var rows []*SV
	for _, period := range t.HV().All() {
		for _, sheet := range period.HV().All() {
			for _, row := range sheet.HV().All() {
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
	var codes []string
	for code := range t.At("annual.Full-Time").HV().All() {
		if code != "" {
			codes = append(codes, code)
		}
	}
	for range 1000 {
		paths = append(paths, fmt.Sprintf("annual.Full-Time{%s}.%s", codes[r.IntN(len(codes))], numFields[r.IntN(len(numFields))]))
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
		y.At("hourly.Female Part-Time{9269}.median").Set("0")
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
