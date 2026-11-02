# To do

## Make a hash's values in one allocation in Hash's fast path

**From:** grammar's session, 2026-11-02 (grammar f163c99fba29ca7e6e1fdc6176e2e83e27ef28ee)
**Waiting on:** nothing

**What:** in `plainHash` (`fastpath.go`), make the value scalars of the
hash in one allocation, a `[]SV` of the hash's size, taking a pointer to
each element, rather than one `&SV{}` (through `Value`, then `svOf`) for
each value. A value that's already a `*SV` still goes through `Value`,
as it's a copy with its digits.

**Why:** grammar's roadmap 25 made its matcher fast enough that building
a match's scalar data is now about two thirds of a parse, and nine
tenths of its allocations. Each match is a `scalar.Hash("~", text,
"from", n, "to", n)`, so three of its scalars come from `plainHash`.
Tried in this working copy through grammar's `go.work`, then reverted:

| grammar benchmark | before | after |
|---|---|---|
| `BenchmarkParse` | 5.44 µs, 126 allocs | 4.92 µs (−9.6%), 86 allocs (−32%) |
| `BenchmarkFindLines` | 523 µs, 4,082 allocs | 512 µs (−2.3%), 2,982 (−27%) |
| `BenchmarkFindAll` | 26.7 µs, 260 allocs | 25.9 µs (−2.7%), 195 (−25%) |
| `BenchmarkAssert` | 1.21 ms | 1.15 ms (−5.4%), 21% fewer allocs |

The experiment was:

```go
h := newHV((len(kv) + 1) / 2)
block := make([]SV, (len(kv)+1)/2)
for i := 0; i < len(kv); i += 2 {
	v := &block[i/2]
	if i+1 < len(kv) {
		if s, ok := kv[i+1].(*SV); ok {
			v = Value(s)
		} else {
			v.setValue(normalize(kv[i+1]))
		}
	}
	h.m[kv[i].(string)] = v
}
```

**What's known:** it keeps every value of the hash alive while any one is
referenced, so a program holding one value of a big hash and dropping
the rest keeps more memory than now; `Hash`'s fast path is for building,
so that's probably fine, but say so. `svOf` sets a value through
`setValue`, as the loop does, so the scalars are the same; check
nothing else `svOf` or `Value` does is skipped. The same could apply to
`Array`'s fast path, if it has one, and to `HV.Set` making many keys,
though that can't know how many in advance.

**Steps:** behind `fastPaths`, as the rest of `plainHash` is;
`TestHashFastPathSame` already compares `Hash` with the fast paths on
and off, so check it covers a `*SV` value and an undef trailing key, and
plant a bug (a value stored in the wrong element) to show it notices.
Benchmark scalar's own `Hash` benchmarks, if any, and grammar's with
`go.work`. Then a scalar release, and a task in grammar's `TODO.md` to
take it up.

**Done when:** `Hash` makes its plain values in one allocation, the
differential test covers it, and grammar's `BenchmarkParse` shows the
gain against the released scalar.
