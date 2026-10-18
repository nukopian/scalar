package scalar_test

// Examples of the List::Util functions. They're run as tests, and they make
// up docs/EXAMPLES.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/scalar"
)

var isRemote = func(s *scalar.SV) bool { return s.M(`(?i)\bremote\b`) != nil }

func ExampleAV_AnyMatch() {
	lines := scalar.Array("Office in Leeds", "Remote working available")
	fmt.Println(lines.AnyMatch(isRemote)) // any { /remote/i } @lines
	fmt.Println(scalar.Array().AnyMatch(isRemote))
	// Output:
	// true
	// false
}

func ExampleAV_AllMatch() {
	n := scalar.Array(2, 4, 6)
	fmt.Println(n.AllMatch(func(s *scalar.SV) bool { return s.Mod(2).Int() == 0 })) // all { $_ % 2 == 0 }
	fmt.Println(scalar.Array().AllMatch(isRemote))                                  // true for nothing
	// Output:
	// true
	// true
}

func ExampleAV_NoneMatch() {
	fmt.Println(scalar.Array("Leeds", "York").NoneMatch(isRemote))
	// Output:
	// true
}

func ExampleAV_FirstMatch() {
	lines := scalar.Array("Leeds office", "Salary: £45k", "Salary: £50k")
	salary := lines.FirstMatch(func(s *scalar.SV) bool { return s.Index("£") >= 0 })
	fmt.Println(salary) // first { index($_, "£") >= 0 } @lines
	fmt.Println(lines.FirstMatch(isRemote).Defined())
	// Output:
	// Salary: £45k
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
	fmt.Println(scalar.Array("Go", "go", "Go", "Perl").UniqueStr().Join(",")) // first of each, in order
	skills := scalar.Array("Go", "go", "Perl")
	fmt.Println(skills.Map(func(s *scalar.SV) any { return s.Fc() }).UniqueStr().Join(",")) // ignoring case
	// Output:
	// Go,go,Perl
	// go,perl
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
	titles := scalar.Array("data engineer", "Go Developer", "Analyst")
	byTitle := titles.SortByStr(func(t *scalar.SV) any { return t.Fc() }) // sort_by { fc } @titles
	fmt.Println(byTitle.Join(", "))
	// Output:
	// Analyst, data engineer, Go Developer
}

func ExampleAV_SortByNum() {
	jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "salary": 60000}, {"title": "SQL", "salary": 30000}]`))
	bySalary := jobs.AV().SortByNum(func(j *scalar.SV) any { return j.At("salary") }) // nsort_by { $_->{salary} }
	fmt.Println(bySalary.Get(0).At("title"), bySalary.Get(1).At("title"))
	// Output:
	// SQL Go
}

func ExampleAV_RevSortByStr() {
	words := scalar.Array("bb", "a", "dd", "e")
	fmt.Println(words.RevSortByStr(func(w *scalar.SV) any { return w.Length() }).Join(" ")) // equal keys keep their order
	// Output:
	// bb dd a e
}

func ExampleAV_RevSortByNum() {
	jobs, _ := scalar.FromJSON([]byte(`[{"title": "SQL", "salary": 30000}, {"title": "Go", "salary": 60000}, {"title": "Perl", "salary": 60000}]`))
	best := jobs.AV().RevSortByNum(func(j *scalar.SV) any { return j.At("salary") }) // the best paid first
	fmt.Println(best.Get(0).At("title"), best.Get(1).At("title"), best.Get(2).At("title"))
	// Output:
	// Go Perl SQL
}

func ExampleAV_MaxBy() {
	jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "salary": 60000}, {"title": "Perl", "salary": 60000}, {"title": "SQL", "salary": 30000}]`))
	best := jobs.AV().MaxBy(func(j *scalar.SV) any { return j.At("salary") }) // max_by { $_->{salary} }
	fmt.Println(best.At("title"))                                             // the first of the best paid
	// Output:
	// Go
}

func ExampleAV_MinBy() {
	words := scalar.Array("ccc", "a", "bb", "e")
	fmt.Println(words.MinBy(func(w *scalar.SV) any { return w.Length() })) // min_by { length } @words
	// Output:
	// a
}

func ExampleAV_UniqueBy() {
	jobs, _ := scalar.FromJSON([]byte(`[{"title": "Go", "level": "Senior"}, {"title": "SQL", "level": "Junior"}, {"title": "Perl", "level": "Senior"}]`))
	oneEach := jobs.AV().UniqueBy(func(j *scalar.SV) any { return j.At("level") }) // the first job at each level
	fmt.Println(oneEach.Length(), oneEach.Get(0).At("title"), oneEach.Get(1).At("title"))
	// Output:
	// 2 Go SQL
}

func ExampleAV_PartitionBy() {
	words := scalar.Array("bb", "a", "ccc", "dd", "e")
	byLength := words.PartitionBy(func(w *scalar.SV) any { return w.Length() }) // partition_by { length } @words
	fmt.Println(byLength)
	// Output:
	// {1 => ["a","e"],2 => ["bb","dd"],3 => ["ccc"]}
}

func ExampleAV_CountBy() {
	places := scalar.Array("Leeds", "London", "leeds", "LEEDS")
	where := places.CountBy(func(p *scalar.SV) any { return p.Fc() }) // a Bag: count_by's counts, with MostCommon
	fmt.Println(where.MostCommon(1).Get(0), where.Count("leeds"), where.Counts())
	// Output:
	// leeds 3 {leeds => 3,london => 1}
}

func ExampleAV_ExtractBy() {
	words := scalar.Array("bb", "a", "ccc", "e")
	long := words.ExtractBy(func(w *scalar.SV) bool { return w.Length() > 1 }) // moved out of words
	fmt.Println(long.Join(" "), "|", words.Join(" "))
	// Output:
	// bb ccc | a e
}
