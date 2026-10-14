package scalar

import (
	"sync"
	"sync/atomic"
)

// cacheLimit is how many entries each of the package's caches (compiled
// patterns, At paths and templates) holds before it starts again empty.
// Patterns written in the program are few, and stay cached; patterns built
// from data, such as one for each skill name, would otherwise make a cache
// grow for as long as the program runs.
const cacheLimit = 4096

// boundedCache is a map safe for use by several goroutines that holds at
// most limit entries: adding one more empties it first. Emptying it costs
// at most one recompile for each entry still in use, which is cheap beside
// growing without end.
type boundedCache[K comparable, V any] struct {
	m     sync.Map
	n     atomic.Int64
	limit int64
}

func (c *boundedCache[K, V]) load(k K) (V, bool) {
	v, ok := c.m.Load(k)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// store adds v under k, unless another goroutine got there first, and
// returns whichever is in the cache.
func (c *boundedCache[K, V]) store(k K, v V) V {
	actual, loaded := c.m.LoadOrStore(k, v)
	if !loaded && c.n.Add(1) > c.limit {
		c.m.Clear()
		c.n.Store(0)
		c.m.Store(k, v)
		c.n.Add(1)
	}
	return actual.(V)
}
