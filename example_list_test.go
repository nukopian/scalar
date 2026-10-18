package scalar_test

// Examples of the List::Util functions. They're run as tests, and they make
// up docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
)

var isOffshore = func(s *scalar.SV) bool { return s.M(`(?i)\boffshore\b`) != nil }

func ExampleAV_AnyMatch() {
	places := scalar.Array("5 km W of Cobb, CA", "Offshore Northern California")
	fmt.Println(places.AnyMatch(isOffshore)) // any { /offshore/i } @places
	fmt.Println(scalar.Array().AnyMatch(isOffshore))
	// Output:
	// true
	// false
}

func ExampleAV_AllMatch() {
	n := scalar.Array(2, 4, 6)
	fmt.Println(n.AllMatch(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })) // all { $_ % 2 == 0 }
	fmt.Println(scalar.Array().AllMatch(isOffshore))                                // true for nothing
	// Output:
	// true
	// true
}

func ExampleAV_NoneMatch() {
	fmt.Println(scalar.Array("Ridgecrest", "Cobb").NoneMatch(isOffshore))
	// Output:
	// true
}

func ExampleAV_FirstMatch() {
	lines := scalar.Array("Origin: 35.7N 117.6W", "Magnitude: 5.1 ml", "Magnitude: 5.3 mw")
	mag := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("Magnitude") >= 0 })
	fmt.Println(mag) // first { index($_, "Magnitude") >= 0 } @lines
	fmt.Println(lines.FirstMatch(isOffshore).Defined())
	// Output:
	// Magnitude: 5.1 ml
	// false
}

func ExampleAV_MaxNum() {
	fmt.Println(scalar.Array("10", 9, "2.5").MaxNum()) // max: as numbers
	// Output:
	// 10
}

func ExampleAV_MinNum() {
	fmt.Println(scalar.Array("10", 9, "2.5").MinNum())
	// Output:
	// 2.5
}

func ExampleAV_MaxStr() {
	fmt.Println(scalar.Array("10", "9", "100").MaxStr()) // maxstr: as strings
	// Output:
	// 9
}

func ExampleAV_MinStr() {
	fmt.Println(scalar.Array("b", "a", "c").MinStr())
	// Output:
	// a
}

func ExampleAV_UniqueStr() {
	fmt.Println(scalar.Array("PAS", "pas", "PAS", "GSC").UniqueStr().Join(",")) // first of each, in order
	stations := scalar.Array("PAS", "pas", "GSC")
	fmt.Println(stations.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr().Join(",")) // ignoring case
	// Output:
	// PAS,pas,GSC
	// pas,gsc
}

func ExampleAV_UniqueNum() {
	fmt.Println(scalar.Array(1, "1.0", "01", 2).UniqueNum().Join(",")) // the same number, however written
	// Output:
	// 1,2
}

func ExampleAV_Sum() {
	fmt.Println(scalar.Array(1, "2", 3.5).Sum()) // sum @a
	fmt.Println(scalar.Array().Sum().Defined())  // undef for nothing
	// Output:
	// 6.5
	// false
}

func ExampleAV_Sum0() {
	fmt.Println(scalar.Array().Sum0()) // sum0: 0 for nothing
	// Output:
	// 0
}

func ExampleAV_Product() {
	fmt.Println(scalar.Array(2, 3, 4).Product())
	fmt.Println(scalar.Array().Product()) // 1 for nothing
	// Output:
	// 24
	// 1
}

func ExampleAV_Reduce() {
	words := scalar.Array("go", "perl", "c")
	longest := words.Reduce(func(a, b *scalar.SV) any { // reduce { length $b > length $a ? $b : $a } @words
		if b.Length() > a.Length() {
			return b
		}
		return a
	})
	joined := words.Reduce(func(a, b *scalar.SV) any { return a.Str() + "+" + b.Str() })
	fmt.Println(longest, joined)
	// Output:
	// perl go+perl+c
}

func ExampleAV_SortByStr() {
	places := scalar.Array("ridgecrest", "Cobb", "Anza")
	byName := places.SortByStr(func(p *scalar.SV) any { return p.Fc() }) // sort_by { fc } @places
	fmt.Println(byName.Join(", "))
	// Output:
	// Anza, Cobb, ridgecrest
}

func ExampleAV_SortByNum() {
	quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "mag": 5.1}, {"place": "Cobb", "mag": 2.3}]`))
	byMag := quakes.AV().SortByNum(func(q *scalar.SV) any { return q.At("mag") }) // nsort_by { $_->{mag} }
	fmt.Println(byMag.Get(0).At("place"), byMag.Get(1).At("place"))
	// Output:
	// Cobb Ridgecrest
}

func ExampleAV_RevSortByStr() {
	words := scalar.Array("bb", "a", "dd", "e")
	fmt.Println(words.RevSortByStr(func(w *scalar.SV) any { return w.Length() }).Join(" ")) // equal keys keep their order
	// Output:
	// bb dd a e
}

func ExampleAV_RevSortByNum() {
	quakes, _ := scalar.FromJSON([]byte(`[{"place": "Cobb", "mag": 2.3}, {"place": "Ridgecrest", "mag": 5.1}, {"place": "Anza", "mag": 5.1}]`))
	strongest := quakes.AV().RevSortByNum(func(q *scalar.SV) any { return q.At("mag") }) // the strongest first
	fmt.Println(strongest.Get(0).At("place"), strongest.Get(1).At("place"), strongest.Get(2).At("place"))
	// Output:
	// Ridgecrest Anza Cobb
}

func ExampleAV_MaxBy() {
	quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "mag": 5.1}, {"place": "Anza", "mag": 5.1}, {"place": "Cobb", "mag": 2.3}]`))
	strongest := quakes.AV().MaxBy(func(q *scalar.SV) any { return q.At("mag") }) // max_by { $_->{mag} }
	fmt.Println(strongest.At("place"))                                            // the first of the strongest
	// Output:
	// Ridgecrest
}

func ExampleAV_MinBy() {
	words := scalar.Array("ccc", "a", "bb", "e")
	fmt.Println(words.MinBy(func(w *scalar.SV) any { return w.Length() })) // min_by { length } @words
	// Output:
	// a
}

func ExampleAV_UniqueBy() {
	quakes, _ := scalar.FromJSON([]byte(`[{"place": "Ridgecrest", "net": "ci"}, {"place": "Cobb", "net": "nc"}, {"place": "Anza", "net": "ci"}]`))
	oneEach := quakes.AV().UniqueBy(func(q *scalar.SV) any { return q.At("net") }) // the first quake from each network
	fmt.Println(oneEach.Length(), oneEach.Get(0).At("place"), oneEach.Get(1).At("place"))
	// Output:
	// 2 Ridgecrest Cobb
}

func ExampleAV_PartitionBy() {
	words := scalar.Array("bb", "a", "ccc", "dd", "e")
	byLength := words.PartitionBy(func(w *scalar.SV) any { return w.Length() }) // partition_by { length } @words
	fmt.Println(byLength)
	// Output:
	// {1 => ["a","e"],2 => ["bb","dd"],3 => ["ccc"]}
}

func ExampleAV_CountBy() {
	places := scalar.Array("Ridgecrest", "Cobb", "ridgecrest", "RIDGECREST")
	where := places.CountBy(func(p *scalar.SV) any { return p.Fc() }) // a Bag: count_by's counts, with MostCommon
	fmt.Println(where.MostCommon(1).Get(0), where.Count("ridgecrest"), where.Counts())
	// Output:
	// ridgecrest 3 {cobb => 1,ridgecrest => 3}
}

func ExampleAV_ExtractBy() {
	words := scalar.Array("bb", "a", "ccc", "e")
	long := words.ExtractBy(func(w *scalar.SV) bool { return w.Length() > 1 }) // moved out of words
	fmt.Println(long.Join(" "), "|", words.Join(" "))
	// Output:
	// bb ccc | a e
}
