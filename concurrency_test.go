package scalar

import (
	"sync"
	"testing"
)

// The patterns in the README's Concurrency section are race-free: run
// with go test -race. Each goroutine reads (and so may write) only its own
// Clone, and what's shared is a plain Go value from To.
func TestSharingAcrossGoroutines(t *testing.T) {
	type event struct {
		Place string `json:"place"`
		Felt  int    `json:"felt"`
	}
	doc, err := FromJSON([]byte(`{"place": "Cobb, CA", "felt": 1204, "meta": {"seen": 1}}`))
	if err != nil {
		t.Fatal(err)
	}
	shared, err := To[event](doc) // the shape is known: into Go
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		own := doc.Clone() // each goroutine's own copy
		wg.Go(func() {
			own.At("meta.missing.deeper") // a read that writes: safe on its own copy
			own.HV().Keys()               // and one that caches
			if own.At("place").Str() != shared.Place || shared.Felt != 1204 {
				t.Error("a copy or the shared struct changed")
			}
		})
	}
	wg.Wait()
	if doc.At("meta").HV().Exists("missing") {
		t.Error("a goroutine's reads reached the original through its Clone")
	}
}
