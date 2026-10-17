package scalar

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// panicOf returns what f panicked with, or "" if it didn't.
func panicOf(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}

const (
	readOnly   = "Modification of a read-only value attempted"
	disallowed = "Attempt to access disallowed key 'x' in a restricted hash"
	undeleting = "Attempt to delete readonly key 'a' from a restricted hash"
)

func lockedDoc(t *testing.T) *SV {
	t.Helper()
	doc, err := FromJSON([]byte(`{"a": 1, "list": [1, {"b": 2}], "nested": {"c": "3"}, "nothing": null}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc.Lock()
}

// A locked hash behaves as Perl's Hash::Util lock_hash makes one (checked
// with Perl 5.42): its keys are fixed, and reading or writing one it
// doesn't have dies; its values are read-only; exists still answers.
func TestLockHash(t *testing.T) {
	doc := lockedDoc(t)
	h := doc.HV()
	if h.Get("a").Int() != 1 || h.Exists("x") || !doc.Locked() || !h.Locked() {
		t.Error("reading a locked hash")
	}
	for name, tc := range map[string]struct {
		f    func()
		want string
	}{
		"read a missing key":   {func() { h.Get("x") }, disallowed},
		"At a missing path":    {func() { doc.At("x.y") }, disallowed},
		"store a new key":      {func() { h.Set("x", 1) }, disallowed},
		"change a value":       {func() { h.Set("a", 2) }, readOnly},
		"change it in place":   {func() { h.Get("a").Inc() }, readOnly},
		"Push":                 {func() { h.Push("a", 5) }, readOnly},
		"delete":               {func() { h.Delete("a") }, undeleting},
		"Undef the hash":       {func() { h.Undef() }, undeleting},
		"a nested hash":        {func() { doc.At("nested.c").Set("4") }, readOnly},
		"a nested hash's keys": {func() { doc.At("nested").HV().Set("x", 1) }, disallowed},
		"undef in place":       {func() { h.Get("nothing").Set(1) }, readOnly},
	} {
		if got := panicOf(tc.f); !strings.Contains(got, tc.want) {
			t.Errorf("%s: panic %q, want %q", name, got, tc.want)
		}
	}
	// Reads that never create don't die.
	if v, ok := doc.AtOK("x.y"); ok || v.Defined() {
		t.Error("AtOK on a locked hash")
	}
	if doc.Find("$.x") != nil || doc.Find("$.list[1].b").Get(0).Int() != 2 {
		t.Error("Find on a locked hash")
	}
	// A locked undef doesn't autovivify when dereferenced.
	if h.Get("nothing").HV() != nil || h.Get("nothing").AV() != nil {
		t.Error("a locked undef autovivified")
	}
}

// A locked array is read-only, as Internals::SvREADONLY makes one, and so
// are its elements (a step beyond Perl, whose read-only array still lets
// its elements change): Lock freezes everything it reaches.
func TestLockArray(t *testing.T) {
	doc := lockedDoc(t)
	a := doc.At("list").AV()
	if a.Get(0).Int() != 1 || a.Get(9).Defined() || !a.Locked() {
		t.Error("reading a locked array, past its end too")
	}
	for name, f := range map[string]func(){
		"Set":            func() { a.Set(0, 9) },
		"an element":     func() { a.Get(0).Set(9) },
		"past the end":   func() { a.Get(9).Set(9) },
		"Push":           func() { a.Push(3) },
		"Unshift":        func() { a.Unshift(0) },
		"Pop":            func() { a.Pop() },
		"Shift":          func() { a.Shift() },
		"PopN":           func() { a.PopN(1) },
		"ShiftN":         func() { a.ShiftN(1) },
		"Splice":         func() { a.Splice(0, 1) },
		"Delete":         func() { a.Delete(0) },
		"SetLastIndex":   func() { a.SetLastIndex(5) },
		"Undef":          func() { a.Undef() },
		"SortInPlace":    func() { a.SortInPlace(NumCmp) },
		"ReverseInPlace": func() { a.ReverseInPlace() },
		"ExtractBy":      func() { a.ExtractBy(func(*SV) bool { return true }) },
		"nested":         func() { a.Get(1).HV().Set("b", 3) },
	} {
		if got := panicOf(f); !strings.Contains(got, readOnly) && !strings.Contains(got, "disallowed") {
			t.Errorf("%s: panic %q, want %q", name, got, readOnly)
		}
	}
	// What makes new arrays still works.
	if a.Sort(NumCmp).Length() != 2 || a.Reverse().Length() != 2 || a.Map(func(s *SV) any { return s }).Length() != 2 {
		t.Error("copies of a locked array")
	}
}

func TestLockScalar(t *testing.T) {
	s := Value("text").Lock()
	for name, f := range map[string]func(){
		"Set":           func() { s.Set("other") },
		"Inc":           func() { s.Inc() },
		"Undef":         func() { s.Undef() },
		"ConcatInPlace": func() { s.ConcatInPlace("!") },
	} {
		if got := panicOf(f); !strings.Contains(got, readOnly) {
			t.Errorf("%s: panic %q, want %q", name, got, readOnly)
		}
	}
	if s.Str() != "text" || s.Copy().Set("x").Str() != "x" {
		t.Error("reading a locked scalar, or changing its copy")
	}
}

// Unlock undoes Lock everywhere it reached; Lock and Unlock can be repeated;
// a Clone is unlocked; a structure that loops back on itself locks.
func TestUnlock(t *testing.T) {
	doc := lockedDoc(t).Lock() // twice is fine
	clone := doc.Clone()
	if clone.Locked() || clone.At("list").AV().Locked() {
		t.Error("a Clone of a locked structure is locked")
	}
	clone.At("x").Set(1)
	doc.Unlock().Unlock()
	if doc.Locked() || doc.At("list").AV().Locked() || doc.At("nested").HV().Locked() {
		t.Error("still locked after Unlock")
	}
	doc.At("list").AV().Push(3)
	doc.At("nested.c").Set("4")
	doc.At("x.y").Set(1) // autovivifies again
	if doc.At("x.y").Int() != 1 || doc.At("list").AV().Length() != 3 {
		t.Error("writing after Unlock")
	}
	loop := Hash("name", "loop")
	loop.Get("self").Set(loop)
	Value(loop).Lock()
	if !loop.Locked() || panicOf(func() { loop.Set("name", "x") }) == "" {
		t.Error("a structure that loops back on itself")
	}
}

// Every way of reading a locked structure is a pure read, so many
// goroutines can read one at once: go test -race. (Each keeps a position,
// so it isn't among them.)
func TestLockedConcurrentReads(t *testing.T) {
	doc := lockedDoc(t).Unlock()
	doc.At("list").AV().Set(5, "late") // holes at 2 to 4
	doc.Lock()
	other := doc.Clone()
	type shape struct {
		A    int
		List []any
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				_ = doc.At("list[1].b").Num() + doc.At("nested.c").Num()
				_ = doc.HV().Keys().Join(",")
				_ = doc.HV().Values().Length() + doc.HV().Pairs().Length()
				for range doc.HV().All() {
				}
				_ = doc.Find("$..b")
				_ = doc.HV().Exists("x")
				_, _ = doc.AtOK("x.y")
				_, _ = doc.PathOK("x", 1)

				list := doc.At("list").AV()
				_ = list.Get(3).Defined() || list.Get(40).Defined() // a hole, past the end
				_ = list.Sort(StrCmp).Join(",") + list.Join(",")
				for range list.All() {
				}

				_ = doc.Equal(other)
				_ = doc.Diff(other)
				_ = doc.Dump()
				_, _ = doc.MarshalJSON()
				_, _ = doc.ToYAML()
				_, _ = To[shape](doc)
				_ = doc.Clone()
				_ = doc.Interpolate("{a}-{nested.c}")
				_ = doc.At("nested.c").M(`(\d)`).Length()
			}
		})
	}
	wg.Wait()
}
