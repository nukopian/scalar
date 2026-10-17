package scalar

// readOnlyMsg is Perl's message for a change to a read-only value.
const readOnlyMsg = "Modification of a read-only value attempted"

// Lock makes s, and everything it refers to, read-only, all the way down,
// and returns s; see the README's Locking section. Perl's Hash::Util is
// the model: a locked hash's keys are fixed, so reading or writing one it
// hasn't got panics ("Attempt to access disallowed key 'x' in a
// restricted hash"), as do deleting one and any change to a value
// ("Modification of a read-only value attempted"); a locked array can't
// grow, shrink or change. Unlike Perl's, a lock reaches nested arrays as
// well as hashes, and their elements. Reads that would autovivify panic
// instead (At("x.y") with no x), and reads that never create (AtOK, Find,
// Exists) answer as usual.
//
// Nothing a locked structure does when read writes anything, so it can be
// shared between goroutines for reading, except for Each, which keeps a
// position. Lock it before sharing it. Locking what's locked does nothing;
// Clone gives an unlocked copy.
func (s *SV) Lock() *SV {
	lockSV(s, true, map[any]bool{})
	return s
}

// Unlock undoes Lock, all the way down, and returns s. A part shared with
// another locked structure is unlocked for both.
func (s *SV) Unlock() *SV {
	lockSV(s, false, map[any]bool{})
	return s
}

// Locked reports whether Lock has made s read-only.
func (s *SV) Locked() bool { return s != nil && s.ro }

// Lock locks a and everything in it, as SV.Lock does, and returns a.
func (a *AV) Lock() *AV {
	lockAV(a, true, map[any]bool{})
	return a
}

// Unlock undoes Lock, as SV.Unlock does, and returns a.
func (a *AV) Unlock() *AV {
	lockAV(a, false, map[any]bool{})
	return a
}

// Locked reports whether a is locked.
func (a *AV) Locked() bool { return a != nil && a.locked }

// Lock locks h and everything in it, as SV.Lock does, and returns h.
func (h *HV) Lock() *HV {
	lockHV(h, true, map[any]bool{})
	return h
}

// Unlock undoes Lock, as SV.Unlock does, and returns h.
func (h *HV) Unlock() *HV {
	lockHV(h, false, map[any]bool{})
	return h
}

// Locked reports whether h is locked.
func (h *HV) Locked() bool { return h != nil && h.locked }

// lockSV, lockAV and lockHV set or clear the locks on everything
// reachable, each container once, however often it's reached, so a
// structure that refers back to itself locks too.
func lockSV(s *SV, on bool, seen map[any]bool) {
	if s == nil {
		return
	}
	s.ro = on
	switch v := s.v.(type) {
	case *AV:
		lockAV(v, on, seen)
	case *HV:
		lockHV(v, on, seen)
	}
}

func lockAV(a *AV, on bool, seen map[any]bool) {
	if a == nil || seen[a] {
		return
	}
	seen[a] = true
	a.locked = on
	for _, s := range a.items {
		lockSV(s, on, seen) // a hole stays a hole
	}
}

func lockHV(h *HV, on bool, seen map[any]bool) {
	if h == nil || seen[h] {
		return
	}
	seen[h] = true
	h.locked = on
	if on {
		h.keys() // the key order, worked out now, so that reading never writes it
	}
	for _, s := range h.m {
		lockSV(s, on, seen)
	}
}
