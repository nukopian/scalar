package scalar

import (
	"fmt"
	"sync"
	"testing"
)

func TestBoundedCache(t *testing.T) {
	c := boundedCache[string, int]{limit: 3}
	for i := range 3 {
		c.store(fmt.Sprint(i), i)
	}
	if v, ok := c.load("2"); !ok || v != 2 {
		t.Fatal("entries up to the limit should stay")
	}
	c.store("3", 3) // one more: it starts again, with just this one
	if _, ok := c.load("0"); ok {
		t.Error("passing the limit should empty the cache")
	}
	if v, ok := c.load("3"); !ok || v != 3 || c.n.Load() != 1 {
		t.Errorf("the entry that passed the limit should be kept: %v %v, n=%d", v, ok, c.n.Load())
	}
	if got := c.store("3", 99); got != 3 {
		t.Error("store should return the value already there")
	}

	// Many goroutines at once: never more than the limit, give or take
	// those mid-store.
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 1000 {
				c.store(fmt.Sprint(g, "-", i), i)
			}
		})
	}
	wg.Wait()
	n := 0
	c.m.Range(func(_, _ any) bool { n++; return true })
	if n > 3+8 {
		t.Errorf("%d entries, with a limit of 3", n)
	}
}

// Paths, patterns and templates built from data don't make the caches grow
// without end.
func TestCachesAreBounded(t *testing.T) {
	doc := Hash()
	s := Scalar("x")
	for i := range cacheLimit + 10 {
		doc.Interpolate(fmt.Sprintf("$k%d", i))
		Scalar(doc).At(fmt.Sprintf("k%d", i))
		s.M(fmt.Sprintf("x%d?", i))
	}
	for name, n := range map[string]int64{"At": atCache.n.Load(), "Interpolate": templateCache.n.Load(), "regex": regexCache.n.Load()} {
		if n > cacheLimit {
			t.Errorf("%s's cache has %d entries, over the limit of %d", name, n, cacheLimit)
		}
	}
}
