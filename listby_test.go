package scalar

import (
	"encoding/json"
	"math/rand/v2"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// The List::UtilsBy analogues give what List::UtilsBy gives, for hundreds
// of random lists, with numeric and string keys. It needs perl and
// List::UtilsBy, and is skipped without them.
func TestListByMatchesPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	if exec.Command("perl", "-MList::UtilsBy", "-e1").Run() != nil {
		t.Skip("List::UtilsBy isn't installed (cpanm List::UtilsBy, or set PERL5LIB)")
	}
	rng := rand.New(rand.NewPCG(9, 4))
	letters := "abcAB09"
	var lists [][]string
	for range 200 {
		words := make([]string, rng.IntN(8))
		for i := range words {
			b := make([]byte, rng.IntN(4))
			for j := range b {
				b[j] = letters[rng.IntN(len(letters))]
			}
			words[i] = string(b)
		}
		lists = append(lists, words)
	}
	in, _ := json.Marshal(lists)
	const script = `
use v5.36; use JSON::PP;
use List::UtilsBy qw(sort_by nsort_by rev_sort_by rev_nsort_by max_by min_by uniq_by partition_by count_by extract_by);
sub hashstr ($h) { join ";", map { "$_=" . (ref $h->{$_} ? join(",", @{$h->{$_}}) : $h->{$_}) } sort keys %$h }
my @out;
for my $w (@{ decode_json(do { local $/; <STDIN> }) }) {
	my @w = @$w;
	my @x = @w;
	my @got = extract_by { length > 1 } @x;
	push @out, [
		join(",", sort_by { substr($_, 0, 1) } @w),
		join(",", nsort_by { length } @w),
		join(",", rev_sort_by { substr($_, 0, 1) } @w),
		join(",", rev_nsort_by { length } @w),
		scalar(max_by { length } @w) // "UNDEF",
		scalar(min_by { length } @w) // "UNDEF",
		join(",", uniq_by { length } @w),
		hashstr({ partition_by { substr($_, -1) } @w }),
		hashstr({ count_by { length } @w }),
		join(",", @got) . "|" . join(",", @x),
	];
}
print encode_json(\@out);
`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v: %s", err, stderr.String())
	}
	var want [][]string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("perl's output: %v", err)
	}

	first := func(s *SV) any { return s.Substr(0, 1) }
	last := func(s *SV) any { return s.Substr(-1) }
	length := func(s *SV) any { return s.Length() }
	join := func(a *AV) string { return a.Join(",") }
	hashStr := func(h *HV) string {
		keys := h.KeysSlice()
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			v := h.Get(k)
			if v.Kind() == ArrayRef {
				parts[i] = k + "=" + v.AV().Join(",")
			} else {
				parts[i] = k + "=" + v.Str()
			}
		}
		return strings.Join(parts, ";")
	}
	orUndefStr := func(s *SV) string {
		if !s.Defined() {
			return "UNDEF"
		}
		return s.Str()
	}
	names := []string{"sort_by", "nsort_by", "rev_sort_by", "rev_nsort_by", "max_by", "min_by",
		"uniq_by", "partition_by", "count_by", "extract_by"}
	for i, words := range lists {
		a := Array(words)
		x := Array(words)
		got := x.ExtractBy(func(s *SV) bool { return s.Length() > 1 })
		gots := []string{
			join(a.SortByStr(first)), join(a.SortByNum(length)),
			join(a.RevSortByStr(first)), join(a.RevSortByNum(length)),
			orUndefStr(a.MaxBy(length)), orUndefStr(a.MinBy(length)),
			join(a.UniqueBy(length)), hashStr(a.PartitionBy(last)), hashStr(a.CountBy(length).Counts()),
			join(got) + "|" + join(x),
		}
		for j, g := range gots {
			if g != want[i][j] {
				t.Errorf("%s of %q: Go %q, Perl %q", names[j], words, g, want[i][j])
			}
		}
	}
}

func TestListBy(t *testing.T) {
	a := Array("bb", "a", "ccc")
	sorted := a.SortByNum(func(s *SV) any { return s.Length() })
	sorted.Get(0).Set("changed")
	if a.Get(1).Str() != "a" {
		t.Error("the sorts should return copies")
	}
	if m := a.MaxBy(func(s *SV) any { return s.Length() }); m != a.Get(2) {
		t.Error("MaxBy should return the element itself")
	}
	calls := 0
	Array(3, 1, 2, 5, 4).SortByNum(func(s *SV) any { calls++; return s })
	if calls != 5 {
		t.Errorf("the key function should be called once for each element, not %d times", calls)
	}
	if Array().MaxBy(func(s *SV) any { return s }).Defined() || Array().MinBy(func(s *SV) any { return s }).Defined() {
		t.Error("MaxBy and MinBy of an empty array are undef")
	}
	if p := Array("x").PartitionBy(func(s *SV) any { return nil }); p.Get("").AV().Join(",") != "x" {
		t.Error("an undef key groups under \"\"")
	}
	if b := Array("a", "b", "c").CountBy(func(s *SV) any { return s.Lt("c") }); b.MostCommon(1).Get(0).Str() != "1" || b.Count(1) != 2 {
		t.Errorf("CountBy: %s", b)
	}

	// ExtractBy: the elements themselves move; holes become undef.
	h := Array(1)
	h.Set(3, 4)
	kept := h.Get(0)
	got := h.ExtractBy(func(s *SV) bool { return s.Int() > 3 })
	if got.Join(",") != "4" || h.Length() != 3 || !h.Exists(1) || h.Get(0) != kept {
		t.Errorf("ExtractBy: got %s, left %s", got, h)
	}
	var nilAV *AV
	if nilAV.ExtractBy(func(s *SV) bool { return true }).Length() != 0 {
		t.Error("ExtractBy on nil gives an empty array")
	}
}
