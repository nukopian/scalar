package scalar

import "testing"

// The arithmetic benchmarks time each kind of operand: integers, floats,
// numeric strings, and whole numbers beyond int64, which keep their digits.

func benchOp(b *testing.B, x, y *SV, op func(x, y *SV) *SV) {
	for b.Loop() {
		_ = op(x, y)
	}
}

func add(x, y *SV) *SV { return x.Add(y) }
func mul(x, y *SV) *SV { return x.Mul(y) }
func mod(x, y *SV) *SV { return x.Mod(y) }

func BenchmarkAddInt(b *testing.B)    { benchOp(b, Value(12345), Value(678), add) }
func BenchmarkAddFloat(b *testing.B)  { benchOp(b, Value(1.5), Value(2.25), add) }
func BenchmarkAddString(b *testing.B) { benchOp(b, Value("4.5"), Value("12"), add) }
func BenchmarkAddBig(b *testing.B)    { benchOp(b, Value(uint64(1)<<63+5), Value(7), add) }
func BenchmarkMulInt(b *testing.B)    { benchOp(b, Value(12345), Value(678), mul) }
func BenchmarkMulFloat(b *testing.B)  { benchOp(b, Value(1.5), Value(2.25), mul) }
func BenchmarkModInt(b *testing.B)    { benchOp(b, Value(12345), Value(678), mod) }
func BenchmarkModFloat(b *testing.B)  { benchOp(b, Value(1234.5), Value(6.5), mod) }

func BenchmarkNumLtInt(b *testing.B) {
	x, y := Value(12345), Value(678)
	for b.Loop() {
		_ = x.NumLt(y)
	}
}

func BenchmarkNumLtFloat(b *testing.B) {
	x, y := Value(1.5), Value(2.25)
	for b.Loop() {
		_ = x.NumLt(y)
	}
}

func BenchmarkInc(b *testing.B) {
	s := Value(0)
	for b.Loop() {
		s.Inc()
	}
}

// BenchmarkSortNum sorts 1,000 mixed numbers, the comparison-heavy path.
func BenchmarkSortNum(b *testing.B) {
	a := Array()
	for i := range 1000 {
		switch i % 3 {
		case 0:
			a.Push(i * 7919 % 1000)
		case 1:
			a.Push(float64(i*7919%1000) + 0.5)
		default:
			a.Push(uint64(1)<<63 + uint64(i*7919%1000))
		}
	}
	for b.Loop() {
		_ = a.Sort(NumCmp)
	}
}

// BenchmarkSortNumNoBig is BenchmarkSortNum without whole numbers beyond
// int64: what a sort costs that never meets one.
func BenchmarkSortNumNoBig(b *testing.B) {
	a := Array()
	for i := range 1000 {
		if i%2 == 0 {
			a.Push(i * 7919 % 1000)
		} else {
			a.Push(float64(i*7919%1000) + 0.5)
		}
	}
	for b.Loop() {
		_ = a.Sort(NumCmp)
	}
}
