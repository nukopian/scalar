package scalar

import (
	"encoding/json"
	"slices"
	"testing"
)

// Sets behave as Set::Scalar 1.29 does: the expected values are what it
// printed for the same calls (perl -MSet::Scalar), apart from where
// TestSetExtras notes a deliberate difference.
func TestSetMatchesSetScalar(t *testing.T) {
	s := NewSet(1, "1", 1.0, "1.0", "Go", "go")
	if s.Length() != 4 || s.String() != "(1 1.0 Go go)" {
		t.Errorf("new(1, \"1\", 1.0, \"1.0\", Go, go) = %v (%d), want (1 1.0 Go go) (4)", s, s.Length())
	}

	a, b := NewSet("c", "a", "b"), NewSet("b", "c", "d")
	if a.String() != "(a b c)" || a.Members().Join(",") != "c,a,b" {
		t.Errorf("printed %v, members %s: printing sorts, members keep insertion order", a, a.Members().Join(","))
	}
	for name, tt := range map[string]struct {
		got  *Set
		want string
	}{
		"union":                {a.Union(b), "(a b c d)"},
		"intersection":         {a.Intersection(b), "(b c)"},
		"difference":           {a.Difference(b), "(a)"},
		"symmetric_difference": {a.SymmetricDifference(b), "(a d)"},
	} {
		if tt.got.String() != tt.want {
			t.Errorf("%s = %v, want %s", name, tt.got, tt.want)
		}
	}
	if !a.IsSubset(a) || a.IsProperSubset(a) || !NewSet().IsSubset(a) {
		t.Error("is_subset(a, a) is 1, is_proper_subset(a, a) is 0, and the empty set is a subset of a")
	}
	if NewSet(nil, "").Length() != 1 {
		t.Error("undef and \"\" are the same member")
	}
	a.Insert("z").Delete("a", "nope")
	if a.String() != "(b c z)" {
		t.Errorf("after insert z, delete a and nope: %v, want (b c z)", a)
	}
}

// Where sets go beyond Set::Scalar, or depart from it on purpose.
func TestSetExtras(t *testing.T) {
	// Set::Scalar says two empty sets aren't disjoint; they are.
	if !NewSet().IsDisjoint(NewSet()) {
		t.Error("empty sets have nothing in common, so they're disjoint")
	}
	a, b := NewSet(1, 2), NewSet(2, 3)
	if a.IsDisjoint(b) || !a.IsDisjoint(NewSet(9)) {
		t.Error("IsDisjoint")
	}
	if !NewSet(1, 2, 3).IsSuperset(a) || !a.IsEqual(NewSet(2, 1)) || a.IsEqual(b) {
		t.Error("IsSuperset and IsEqual")
	}
	if got := NewSet(1).Union(NewSet(2), NewSet(3)).Intersection(NewSet(2, 3), NewSet(3)); got.String() != "(3)" {
		t.Errorf("several others: %v", got)
	}

	// Members are copies, kept from the first insert; changing one changes
	// nothing.
	orig := Scalar("Go")
	s := NewSet(orig, "Go")
	orig.Set("changed")
	s.Members().Get(0).Set("also changed")
	if !s.Has("Go") || s.Has("changed") || s.Length() != 1 {
		t.Errorf("members should be the set's own copies: %v", s)
	}

	// Flattening, and the zero value.
	var z Set
	z.Insert(Array("a", "b"), []string{"c"})
	if z.String() != "(a b c)" {
		t.Errorf("zero value after inserting a list: %v", &z)
	}

	// Nil sets read as empty.
	var nilSet *Set
	if nilSet.Length() != 0 || nilSet.Has("a") || !nilSet.IsSubset(a) || !a.IsDisjoint(nilSet) ||
		a.Intersection(nilSet).Length() != 0 || a.Difference(nilSet).Length() != 2 || nilSet.String() != "()" {
		t.Error("a nil set should read as empty")
	}
	expectPanic(t, "Insert on nil", "nil *Set", func() { nilSet.Insert("a") })

	out, err := json.Marshal(NewSet("b", "a"))
	if err != nil || string(out) != `["b","a"]` {
		t.Errorf("JSON = %s, %v", out, err)
	}
}

// Bags behave as Set::Bag 1.012 does with over_delete: the expected values
// are what it printed for the same calls (perl -MSet::Bag).
func TestBagMatchesSetBag(t *testing.T) {
	b := NewBag().InsertN("apples", 3).InsertN("pears", 1)
	if b.String() != "(apples => 3, pears => 1)" {
		t.Errorf("bag = %v", b)
	}
	b.InsertN("apples", 2).InsertN("figs", 1)
	if b.String() != "(apples => 5, figs => 1, pears => 1)" {
		t.Errorf("after insert = %v", b)
	}
	b.DeleteN("pears", 5) // Set::Bag dies here without over_delete
	b.DeleteN("apples", 5)
	if b.String() != "(figs => 1)" || b.Count("apples") != 0 {
		t.Errorf("after deleting more than there is = %v", b)
	}

	x := NewBag().InsertN("a", 3).InsertN("b", 1)
	y := NewBag().InsertN("a", 1).InsertN("b", 2).InsertN("c", 1)
	for name, tt := range map[string]struct {
		got  *Bag
		want string
	}{
		"sum":          {x.Sum(y), "(a => 4, b => 3, c => 1)"},
		"union":        {x.Union(y), "(a => 3, b => 2, c => 1)"},
		"intersection": {x.Intersection(y), "(a => 1, b => 1)"},
		"difference":   {x.Difference(y), "(a => 2)"},
	} {
		if tt.got.String() != tt.want {
			t.Errorf("%s = %v, want %s", name, tt.got, tt.want)
		}
	}
	if x.String() != "(a => 3, b => 1)" {
		t.Errorf("the operations should leave x alone: %v", x)
	}
}

func TestBagExtras(t *testing.T) {
	skills := NewBag("Go", "Perl", "Go", "SQL", "Go", "Perl")
	if skills.Count("Go") != 3 || skills.Length() != 3 || skills.Total() != 6 {
		t.Errorf("counts: Go=%d, distinct=%d, total=%d", skills.Count("Go"), skills.Length(), skills.Total())
	}
	if got := skills.MostCommon(2).Join(","); got != "Go,Perl" {
		t.Errorf("MostCommon(2) = %s", got)
	}
	if got := NewBag("b", "a").MostCommon().Join(","); got != "b,a" {
		t.Errorf("ties keep insertion order: %s", got)
	}
	if got := skills.MostCommon(99).Length(); got != 3 {
		t.Errorf("MostCommon beyond the size = %d members", got)
	}
	if got := skills.Counts(); got.Get("Perl").Int() != 2 || got.Length() != 3 {
		t.Errorf("Counts = %v", got.Pairs())
	}
	skills.Delete("Go", "nope")
	if skills.Count("Go") != 2 {
		t.Error("Delete takes one away, and ignores what isn't there")
	}
	var seen []string
	for m, n := range NewBag("x", "y", "x").All() {
		seen = append(seen, m.Str()+"="+string(rune('0'+n)))
	}
	if !slices.Equal(seen, []string{"x=2", "y=1"}) {
		t.Errorf("All = %q", seen)
	}
	if NewBag().InsertN("a", 0).InsertN("b", -2).Length() != 0 {
		t.Error("a count of zero or less adds nothing")
	}
	out, err := json.Marshal(NewBag("z", "a", "z"))
	if err != nil || string(out) != `{"z":2,"a":1}` {
		t.Errorf("JSON = %s, %v", out, err)
	}

	var nilBag *Bag
	if nilBag.Count("a") != 0 || nilBag.Total() != 0 || nilBag.Sum(NewBag("a")).Count("a") != 1 || nilBag.String() != "()" {
		t.Error("a nil bag should read as empty")
	}
	expectPanic(t, "Insert on nil", "nil *Bag", func() { nilBag.Insert("a") })
	expectPanic(t, "negative MostCommon", "is negative", func() { NewBag("a").MostCommon(-1) })
}

func TestSetAndBagGoForms(t *testing.T) {
	s := NewSet("c", 1, "a")
	got := s.MembersSlice()
	if !slices.Equal(got, []string{"c", "1", "a"}) {
		t.Errorf("Set.MembersSlice = %q", got)
	}
	got[0] = "changed"
	if !s.Has("c") {
		t.Error("changing MembersSlice's result shouldn't change the set")
	}

	b := NewBag("Go", "Perl", "Go")
	if got := b.MembersSlice(); !slices.Equal(got, []string{"Go", "Perl"}) {
		t.Errorf("Bag.MembersSlice = %q", got)
	}
	m := b.CountsMap()
	if len(m) != 2 || m["Go"] != 2 || m["Perl"] != 1 {
		t.Errorf("CountsMap = %v", m)
	}
	m["Go"] = 99
	if b.Count("Go") != 2 {
		t.Error("changing CountsMap's result shouldn't change the bag")
	}

	var nilSet *Set
	var nilBag *Bag
	if len(nilSet.MembersSlice()) != 0 || len(nilBag.MembersSlice()) != 0 || nilBag.CountsMap() == nil {
		t.Error("nil sets and bags should give empty Go values, and CountsMap a usable map")
	}
}
