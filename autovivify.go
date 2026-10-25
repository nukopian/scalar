package scalar

// Autovivify sets whether reading through s creates what's missing, and
// returns s; see the reference's Autovivification section. It's on by
// default, as in Perl. Autovivify(false) is the CPAN pragma's
// no autovivification for s and everything it refers to, all the way
// down: Get, At, HV, AV, Exists and Delete create nothing, and a missing
// key, index or level reads as undef, while a write still creates the
// levels it needs. Values stored later take the setting of the array or
// hash they go into.
//
// With one setting, it's for s and everything below it. With two,
// Autovivify(self, deeply), the first is for s itself, an undef s
// dereferenced, and the second for everything below it. Autovivify() is
// Autovivify(true). More than two settings panics. Unlike the pragma, the
// setting belongs to the value, not to a block of code, and All, Values
// and Each create nothing, where the pragma's foreach and each do.
//
// Combined with Lock, a missing key reads as undef rather than panicking,
// and writes still panic. Clone gives a copy with the default setting.
func (s *SV) Autovivify(on ...bool) *SV {
	self, deeply := vivArgs(on)
	vivSV(s, self, deeply, map[any]bool{})
	return s
}

// Autovivifies reports whether reading through s creates what's missing,
// as Autovivify sets for s itself.
func (s *SV) Autovivifies() bool { return s == nil || !s.noViv }

// AutovivifiesDeeply reports whether reading through what s refers to
// creates what's missing, as Autovivify sets for everything below s.
func (s *SV) AutovivifiesDeeply() bool { return s == nil || !s.noVivDeep }

// Autovivify sets autovivification for a's elements and everything below
// them, as SV.Autovivify does, and returns a. With two settings, the first
// is for a's elements, and the second for everything below them.
func (a *AV) Autovivify(on ...bool) *AV {
	self, deeply := vivArgs(on)
	vivAV(a, self, deeply, map[any]bool{})
	return a
}

// Autovivifies reports whether reading through a's elements creates
// what's missing.
func (a *AV) Autovivifies() bool { return a == nil || !a.noViv }

// AutovivifiesDeeply reports whether reading below a's elements creates
// what's missing.
func (a *AV) AutovivifiesDeeply() bool { return a == nil || !a.noVivDeep }

// Autovivify sets autovivification for h's values and everything below
// them, as SV.Autovivify does, and returns h. With two settings, the first
// is for h's values, and the second for everything below them.
func (h *HV) Autovivify(on ...bool) *HV {
	self, deeply := vivArgs(on)
	vivHV(h, self, deeply, map[any]bool{})
	return h
}

// Autovivifies reports whether reading through h's values creates what's
// missing.
func (h *HV) Autovivifies() bool { return h == nil || !h.noViv }

// AutovivifiesDeeply reports whether reading below h's values creates
// what's missing.
func (h *HV) AutovivifiesDeeply() bool { return h == nil || !h.noVivDeep }

func vivArgs(on []bool) (self, deeply bool) {
	switch len(on) {
	case 0:
		return true, true
	case 1:
		return on[0], on[0]
	case 2:
		return on[0], on[1]
	}
	panic("scalar: Autovivify takes at most two settings")
}

// vivSV, vivAV and vivHV set autovivification on everything reachable,
// each container once, as lockSV and the rest lock it, so a structure
// that refers back to itself is set too, and the receiver keeps its own
// setting. A scalar's setting is for itself, dereferenced; an array's or
// a hash's is for its elements, which it gives to those stored later.
func vivSV(s *SV, self, deeply bool, seen map[any]bool) {
	if s == nil {
		return
	}
	s.noViv, s.noVivDeep = !self, !deeply
	switch v := s.v.(type) {
	case *AV:
		vivAV(v, deeply, deeply, seen)
	case *HV:
		vivHV(v, deeply, deeply, seen)
	}
}

func vivAV(a *AV, elems, deeply bool, seen map[any]bool) {
	if a == nil || seen[a] {
		return
	}
	seen[a] = true
	a.noViv, a.noVivDeep = !elems, !deeply
	for _, s := range a.items {
		vivSV(s, elems, deeply, seen) // a hole stays a hole
	}
}

func vivHV(h *HV, elems, deeply bool, seen map[any]bool) {
	if h == nil || seen[h] {
		return
	}
	seen[h] = true
	h.noViv, h.noVivDeep = !elems, !deeply
	for _, s := range h.m {
		vivSV(s, elems, deeply, seen)
	}
}

// adopt gives s, just stored in an array or hash with autovivification
// off, that container's setting, and everything s refers to the setting
// below. A container with the default setting leaves s as it is, so a
// store costs nothing extra unless the setting is in use.
func adopt(s *SV, noViv, noVivDeep bool) {
	if s == nil || !noViv && !noVivDeep {
		return
	}
	s.noViv, s.noVivDeep = noViv, noVivDeep
	s.adoptValue()
}

// adoptValue gives what s refers to s's setting for below it, when that's
// off, as Set does for a value stored in s.
func (s *SV) adoptValue() {
	if !s.noVivDeep {
		return
	}
	switch v := s.v.(type) {
	case *AV:
		vivAV(v, false, false, map[any]bool{})
	case *HV:
		vivHV(v, false, false, map[any]bool{})
	}
}

// attach stores a, which AV made for an undef with autovivification off,
// in that undef, on a's first write, so that reading creates nothing and
// writing creates every level it needs. If the undef has since been given
// another value, a stays on its own, as an array it no longer refers to
// would.
func (a *AV) attach() {
	s := a.owner
	a.owner = nil
	if s.pend() != a {
		return
	}
	s.vivify()
	s.v = a
}

// attach is AV.attach for a hash.
func (h *HV) attach() {
	s := h.owner
	h.owner = nil
	if s.pend() != h {
		return
	}
	s.vivify()
	s.v = h
}
