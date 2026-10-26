package scalar

// Clone returns a deep copy of s, as Storable's dclone does: arrays and
// hashes it refers to are copied too, all the way down, so changing the
// clone never changes the original. Compare Copy, which is shallow, as
// Perl's assignment is.
//
//	row := h.Get("defaults").Clone()   // my $row = dclone($h{defaults})
//	row.At("tags").AV().Push("new")    // doesn't touch $h{defaults}{tags}
//
// As with dclone, the copy has the original's shape: an array or hash that
// appears twice in s is copied once and appears twice in the clone, a
// reference that leads back into s leads back into the clone, and an
// array's holes stay holes. Byte strings stay byte strings.
func (s *SV) Clone() *SV {
	if s == nil {
		return &SV{}
	}
	return cloner{}.sv(s)
}

// Clone returns a deep copy of a, as dclone(\@a) does; see SV.Clone.
func (a *AV) Clone() *AV {
	if a == nil {
		return nil
	}
	return cloner{}.av(a)
}

// Clone returns a deep copy of h, as dclone(\%h) does; see SV.Clone.
func (h *HV) Clone() *HV {
	if h == nil {
		return nil
	}
	return cloner{}.hv(h)
}

// cloner deep-copies scalars, mapping each array and hash copied so far to
// its copy, so that one that's shared, or that loops back, is copied once.
type cloner map[any]any

func (c cloner) sv(s *SV) *SV {
	if s == nil {
		return nil // a hole in an array stays a hole
	}
	switch v := s.v.(type) {
	case *AV:
		return &SV{v: c.av(v)}
	case *HV:
		return &SV{v: c.hv(v)}
	case []byte:
		// Shared, not copied: nothing in scalar changes a byte string in
		// place, as every operation makes a new one, so the clone can't see
		// a change through it. Value and Bytes still copy at the boundary,
		// where the caller's own slice could change.
		return &SV{v: v}
	}
	return svOf(s.packed())
}

func (c cloner) av(a *AV) *AV {
	if a == nil {
		return nil
	}
	if out, ok := c[a]; ok {
		return out.(*AV)
	}
	out := &AV{items: make([]*SV, len(a.items))}
	c[a] = out
	for i, e := range a.items {
		out.items[i] = c.sv(e)
	}
	return out
}

func (c cloner) hv(h *HV) *HV {
	if h == nil {
		return nil
	}
	if out, ok := c[h]; ok {
		return out.(*HV)
	}
	out := &HV{m: make(map[string]*SV, len(h.m))}
	c[h] = out
	for k, e := range h.m {
		out.m[k] = c.sv(e)
	}
	return out
}
