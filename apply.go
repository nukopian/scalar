package scalar

// Apply calls f on every plain value in s, all the way down, and stores
// what f returns in its place, as Data::Rmap's rmap { $_ = ... } $s does.
// It changes s in place and returns it.
//
//	doc.Apply(func(v *scalar.SV) any {   // rmap { $_ = trim $_ } $doc
//		if v.Kind() == scalar.String {
//			return v.Trim()
//		}
//		return v
//	})
//
// f sees the plain values: strings, numbers, booleans, byte strings and
// undef. Arrays and hashes are walked through rather than passed to f, and
// hash keys are left as they are. Returning v leaves it unchanged; returning
// nil makes it undef. If f returns an array or hash, it's stored as a
// reference, and Apply doesn't walk into it.
//
// As with rmap, each array or hash is visited once, so one that appears
// twice, or a reference that loops back, is safe. Arrays are walked in
// order, and hashes in their own order, as Keys gives it. Unlike rmap,
// which visits an array's holes as undef and so fills them in, Apply skips
// them: they don't exist. For a changed copy, leaving s alone, use
// s.Clone().Apply(f).
func (s *SV) Apply(f func(v *SV) any) *SV {
	if s == nil {
		return nil
	}
	applier{f: f, seen: map[any]bool{}}.sv(s)
	return s
}

// Apply calls f on every plain value in a, all the way down, as rmap { ... }
// @a does; see SV.Apply. It changes a in place and returns it.
func (a *AV) Apply(f func(v *SV) any) *AV {
	applier{f: f, seen: map[any]bool{}}.av(a)
	return a
}

// Apply calls f on every plain value in h, all the way down, as rmap { ... }
// %h does; see SV.Apply. It changes h in place and returns it.
func (h *HV) Apply(f func(v *SV) any) *HV {
	applier{f: f, seen: map[any]bool{}}.hv(h)
	return h
}

// applier walks a structure for Apply, remembering the arrays and hashes
// it's been through, so that each is visited once.
type applier struct {
	f    func(*SV) any
	seen map[any]bool
}

func (ap applier) sv(s *SV) {
	switch v := s.val().(type) {
	case *AV:
		ap.av(v)
	case *HV:
		ap.hv(v)
	default:
		if r := ap.f(s); r != s { // returning s itself leaves it alone
			s.Set(r)
		}
	}
}

func (ap applier) av(a *AV) {
	if a == nil || ap.seen[a] {
		return
	}
	ap.seen[a] = true
	for _, e := range a.items {
		if e != nil { // a hole: nothing there
			ap.sv(e)
		}
	}
}

func (ap applier) hv(h *HV) {
	if h == nil || ap.seen[h] {
		return
	}
	ap.seen[h] = true
	for _, k := range h.keys() {
		if e := h.m[k]; e != nil {
			ap.sv(e)
		}
	}
}
