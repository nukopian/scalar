package scalar

// Then calls f with s if s is defined, and returns what f returns as a new
// scalar; if s is undef, f isn't called, and Then returns undef. So a chain
// of steps stops quietly at the first one that gives undef, as Path does:
//
//	annual := posting.At("salary.text").
//		Then(parseFigure). // undef if there's no figure
//		Then(toAnnual).    // skipped if there wasn't one
//		Or(0)
//
// f may return a scalar or any Go value, and nil for undef. With Value as
// the unit, Then is the bind of a Maybe monad in which undef is Nothing:
// TestThenMonadLaws checks the laws.
func (s *SV) Then(f func(v *SV) any) *SV {
	if !s.Defined() {
		return &SV{}
	}
	return Value(f(s))
}

// When returns a copy of s if s is defined and pred(s) is true, and undef
// otherwise, without calling pred on undef. It's a guard for a chain:
//
//	posting.At("salary.max").When(isPositive).Then(toAnnual).Or("unknown")
func (s *SV) When(pred func(v *SV) bool) *SV {
	if !s.Defined() || !pred(s) {
		return &SV{}
	}
	return s.Copy()
}
