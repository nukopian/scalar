package scalar

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// withFastPaths runs f with the fast paths on, then off, and returns both
// results as text.
func withFastPaths(f func() string) (fast, slow string) {
	defer func() { fastPaths = true }()
	fastPaths = true
	fast = f()
	fastPaths = false
	slow = f()
	return fast, slow
}

// Hash gives the same hash with or without its fast path, for every kind of
// list the fast path takes, and for some it doesn't.
func TestHashFastPathSame(t *testing.T) {
	ref := Scalar(Array(1, 2))
	hashRef := Scalar(Hash("a", 1))
	lists := [][]any{
		{},
		{"k"},
		{"a", 1, "b", "two", "c", 3.5, "d", true, "e", nil},
		{"a", 1, "a", 2},              // later keys win
		{"a", 1, "b"},                 // a trailing key maps to undef
		{"ref", ref, "hash", hashRef}, // references stay one value
		{"bytes", []byte{0xff, 0x00}, "n", int64(7)},
		{"sv", Scalar("x"), "undef", &SV{}},
		{"£", "45,000", "from", 0, "to", 7},
		{1, "one", 2, "two"},         // not string keys: no fast path
		{"a", Array(1, 2)},           // flattens: no fast path
		{Hash("x", 1), "y", 2},       // flattens: no fast path
		{"list", []string{"p", "q"}}, // flattens: no fast path
	}
	for _, kv := range lists {
		fast, slow := withFastPaths(func() string { return fmt.Sprint(Hash(kv...).Dump()) })
		if fast != slow {
			t.Errorf("Hash(%v):\nfast %s\nslow %s", kv, fast, slow)
		}
	}
	// Values are copies, as before: changing one afterwards changes nothing.
	v := Scalar("orig")
	h := Hash("k", v)
	v.Set("changed")
	if h.Get("k").Str() != "orig" {
		t.Error("Hash kept the caller's scalar instead of a copy")
	}
}

// S gives the same result and the same count with or without its fast
// path, over random strings and patterns.
func TestSFastPathSame(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 9))
	alphabet := []rune("ab,. £xé-")
	word := func(max int) string {
		r := make([]rune, rng.IntN(max+1))
		for i := range r {
			r[i] = alphabet[rng.IntN(len(alphabet))]
		}
		return string(r)
	}
	flagSets := [][]string{nil, {"g"}, {"r"}, {"gr"}, {"m"}, {"gs"}, {"i"}, {"gx"}}
	for range 3000 {
		text, pat, repl := word(12), word(3), word(2)
		flags := flagSets[rng.IntN(len(flagSets))]
		for _, bytes := range []bool{false, true} {
			fast, slow := withFastPaths(func() string {
				var s *SV
				if bytes {
					s = Scalar([]byte(strings.Map(func(r rune) rune {
						if r > 0xff {
							return 'z'
						}
						return r
					}, text)))
				} else {
					s = Scalar(text)
				}
				defer func() { recover() }() // a pattern such as "." is regex; both paths treat it the same
				got := s.S(pat, repl, flags...)
				return fmt.Sprintf("%q %q kind %v", got.Str(), s.Str(), s.Kind())
			})
			if fast != slow {
				t.Fatalf("S(%q, %q, %v) on %q (bytes %v):\nfast %s\nslow %s", pat, repl, flags, text, bytes, fast, slow)
			}
		}
	}
}

// Which patterns the literal fast path takes.
func TestLiteralPattern(t *testing.T) {
	for pat, want := range map[string]bool{
		",": true, "abc": true, "£": true, " ": true, "a-b": true,
		"": false, ".": false, "a+": false, `\d`: false, "(x)": false, "[a]": false,
		"^a": false, "a$": false, "a|b": false, "a{2}": false, "?": false, "*": false,
	} {
		if got := literalPattern(pat); got != want {
			t.Errorf("literalPattern(%q) = %v, want %v", pat, got, want)
		}
	}
}
