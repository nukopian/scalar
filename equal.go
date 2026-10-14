package scalar

import (
	"slices"
	"strconv"
)

// Equal reports whether s and other hold the same data, all the way down,
// as Test::More's is_deeply decides it. other may be a scalar or any Go
// value, so s.Equal(map[string]any{...}) works.
//
// Plain values are equal if they're equal as strings, as with Perl's eq, so
// 1 and 1.0 are equal but 1 and "1.0" aren't, a boolean true equals 1, and
// a byte string equals text of the same characters. Undef equals only
// undef, not "". Arrays are equal if they have the same length and equal
// elements, a hole counting as undef; hashes, if they have the same keys and
// equal values, so a missing key isn't the same as one whose value is
// undef. Structures that loop back on themselves compare safely.
//
// To see where two structures differ, use Diff.
func (s *SV) Equal(other any) bool { return s.Diff(other) == "" }

// Equal reports whether a and other hold the same data; see SV.Equal.
func (a *AV) Equal(other any) bool { return (&SV{v: a}).Equal(other) }

// Equal reports whether h and other hold the same data; see SV.Equal.
func (h *HV) Equal(other any) bool { return (&SV{v: h}).Equal(other) }

// Diff says where s and other first differ, as is_deeply reports it, or ""
// if they're equal (see Equal). It gives the path, as At takes it, then
// what's in s and what's in other:
//
//	salary.max: got 55000, want 60000
//	skills[2]: got "SQL", want does not exist
//
// Hash keys are taken in sorted order, so the first difference reported is
// always the same one.
func (s *SV) Diff(other any) string {
	d := differ{seen: map[[2]any]bool{}}
	return d.diff(s, Scalar(other), true, true, "")
}

// Diff says where a and other first differ; see SV.Diff.
func (a *AV) Diff(other any) string { return (&SV{v: a}).Diff(other) }

// Diff says where h and other first differ; see SV.Diff.
func (h *HV) Diff(other any) string { return (&SV{v: h}).Diff(other) }

// differ compares two structures, remembering the pairs of arrays and
// hashes it's comparing already, so that loops end: a pair met again is
// taken to be equal, as is_deeply does.
type differ struct {
	seen map[[2]any]bool
}

// diff compares x and y at path, where xOK and yOK say whether each exists.
func (d differ) diff(x, y *SV, xOK, yOK bool, path string) string {
	report := func(got, want string) string {
		msg := "got " + got + ", want " + want
		if path != "" {
			msg = path + ": " + msg
		}
		return msg
	}
	if !xOK || !yOK {
		if xOK == yOK {
			return ""
		}
		if !xOK {
			return report("does not exist", describe(y))
		}
		return report(describe(x), "does not exist")
	}

	xv, yv := x.val(), y.val()
	switch xc := xv.(type) {
	case *AV:
		yc, ok := yv.(*AV)
		if !ok || xc == nil || yc == nil {
			break
		}
		if xc == yc || d.seen[[2]any{xc, yc}] {
			return ""
		}
		d.seen[[2]any{xc, yc}] = true
		for i := range max(len(xc.items), len(yc.items)) {
			xe, xin := element(xc, i)
			ye, yin := element(yc, i)
			if msg := d.diff(xe, ye, xin, yin, path+"["+strconv.Itoa(i)+"]"); msg != "" {
				return msg
			}
		}
		return ""
	case *HV:
		yc, ok := yv.(*HV)
		if !ok || xc == nil || yc == nil {
			break
		}
		if xc == yc || d.seen[[2]any{xc, yc}] {
			return ""
		}
		d.seen[[2]any{xc, yc}] = true
		keys := make([]string, 0, len(xc.m)+len(yc.m))
		for k := range xc.m {
			keys = append(keys, k)
		}
		for k := range yc.m {
			if _, ok := xc.m[k]; !ok {
				keys = append(keys, k)
			}
		}
		slices.Sort(keys)
		for _, k := range keys {
			xe, xin := xc.m[k]
			ye, yin := yc.m[k]
			if msg := d.diff(xe, ye, xin, yin, keyPath(path, k)); msg != "" {
				return msg
			}
		}
		return ""
	}

	switch {
	case isRef(xv) || isRef(yv):
		// A reference against something else, or arrays and hashes of
		// different kinds: describe says which.
		if isRef(xv) && isRef(yv) && xv == yv {
			return ""
		}
	case !x.Defined() || !y.Defined():
		if x.Defined() == y.Defined() {
			return "" // both undef
		}
	case x.Eq(y):
		return ""
	}
	return report(describe(x), describe(y))
}

// element is a's i'th element, and whether it exists: a hole is undef, and
// beyond the end there's nothing.
func element(a *AV, i int) (*SV, bool) {
	if i >= len(a.items) {
		return nil, false
	}
	if a.items[i] == nil {
		return &SV{}, true
	}
	return a.items[i], true
}

func isRef(v any) bool {
	switch v.(type) {
	case *AV, *HV:
		return true
	}
	return false
}
