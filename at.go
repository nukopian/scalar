package scalar

import (
	"fmt"
	"strconv"
	"strings"
)

// stepKind is what a step of an At path is.
type stepKind int

const (
	stepKey    stepKind = iota // a hash key: name, {key} or ["quoted"]
	stepIndex                  // an array index: [0]
	stepEither                 // .0 after a dot: an index into an array, or a key into a hash
)

type atStep struct {
	kind  stepKind
	key   string // for stepKey, and the digits of a stepEither
	index int    // for stepIndex and stepEither
}

// At is Get with the route as one string, in dot, bracket or Perl's brace
// form:
//
//	s.At("content[0].rule_set")     // $s->{content}[0]{rule_set}
//	s.At("{content}[0]{rule_set}")  // the same, written as in Perl
//	s.At("content.0.rule_set")      // the same, if content holds an array
//	s.At("meta{log.level}")         // a key with dots in it, in braces
//	s.At(`meta["log.level"]`)       // or quoted
//	s.At("items[-1].name")          // negative indices count from the end
//
// A name, a {key} or a quoted ["key"] or ['key'] is a hash key; [n] is an
// array index. Inside braces, everything up to the closing } is the key,
// spaces included, and {"key"} and {'key'} may be quoted too, as in Perl. A
// backslash escapes the next character, as in ["say \"hi\""] or {a\}b}. A number after a dot, as in content.0, is whichever the
// container is: an index into an array, or the key "0" into a hash; writing
// through one that doesn't exist yet creates an array, so quote a hash key
// that looks like a number: ["0"]. A leading $ is allowed, so $.a[0] works,
// as in JSONPath. An empty path is s itself. A name ends at a ., [ or {.
//
// It walks as Get does: intermediate levels autovivify, the final element
// doesn't until it's written to, and a missing step reads as undef. A
// malformed path panics, as a bad regex does.
func (s *SV) At(path string) *SV {
	cur := s
	for _, st := range compileAt(path) {
		cur = st.walk(cur)
	}
	return cur
}

// walk takes one step from cur, as Get does: through a hash or an array,
// autovivifying cur if it's undef.
func (st atStep) walk(cur *SV) *SV {
	switch st.kind {
	case stepKey:
		return cur.HV().get(st.key)
	case stepIndex:
		return cur.AV().get(st.index)
	}
	if _, isHash := cur.val().(*HV); isHash { // stepEither
		return cur.HV().get(st.key)
	}
	return cur.AV().get(st.index) // an array, or nothing yet: an array
}

// peek takes one step from cur, as GetOK does, without creating anything:
// nil if the step doesn't exist.
func (st atStep) peek(cur *SV) *SV {
	switch v := cur.v.(type) {
	case *HV:
		if st.kind != stepIndex {
			return v.m[st.key]
		}
	case *AV:
		if st.kind != stepKey {
			if i, inRange := v.index(st.index); inRange && i < len(v.items) {
				return v.items[i] // nil for a hole, which doesn't exist
			}
		}
	}
	return nil
}

// AtOK is GetOK with the route as one string, written as for At: ok
// reports whether every step exists, so a value that's present but null
// gives ok true. Like GetOK, it never creates anything, and when ok is true,
// v is the stored value itself.
func (s *SV) AtOK(path string) (v *SV, ok bool) {
	return s.peekSteps(compileAt(path))
}

// peekSteps walks steps without creating anything, for AtOK and templates:
// the value at the end, and whether every step exists.
func (s *SV) peekSteps(steps []atStep) (v *SV, ok bool) {
	cur := s
	if cur == nil {
		return &SV{}, false
	}
	for _, st := range steps {
		if cur = st.peek(cur); cur == nil {
			return &SV{}, false
		}
	}
	return cur, true
}

// DeleteAt is Perl's delete $s->{content}[0]{rule_set}, with the route
// written as for At: it removes the last step's key or element, and returns
// its value, or undef if there was nothing there.
//
//	doc.DeleteAt("depth.error")     // delete $doc->{depth}{error}
//	doc.DeleteAt("stations[-1]")    // delete $doc->{stations}[-1]
//
// As with HV.Delete and AV.Delete, an element in the middle of an array
// leaves a hole, and deleting the last shrinks the array. Unlike Perl,
// which creates the levels on the way to what it deletes, DeleteAt never
// creates anything: if a step on the way is missing, or isn't an array or
// hash, there's nothing to delete. A malformed path panics, as for At.
func (s *SV) DeleteAt(path string) *SV {
	v, _ := s.deleteSteps(compileAt(path))
	return v
}

// DeleteOK is Delete for data you aren't sure of, with Go's comma-ok: it
// removes the last key or element as Delete does, and returns its value,
// with ok true if there was something there to remove, even if its value
// was undef. Unlike Delete, and unlike Perl, it never creates anything, and
// never panics on the shape of the data: if a level on the way is missing,
// or isn't an array or hash, or what the last key looks in isn't the kind
// it asks for, there's nothing to delete, and ok is false.
//
// Keys are as for Get, each taken whole, dots and all, so it suits keys
// that come from data, as delete $s->{$section}{$key} does in Perl:
//
//	doc.DeleteOK("depth", "error")                // delete $doc->{depth}{error}
//	if v, ok := doc.DeleteOK(section, key); ok {  // something was removed: v
//
// With no keys, there's nothing to delete, and ok is false. A key of the
// wrong type panics, as for Get, and so does deleting from a locked hash,
// as HV.Delete does.
func (s *SV) DeleteOK(keys ...any) (v *SV, ok bool) {
	for _, k := range keys {
		subscript("DeleteOK", k) // a bad key panics, even after a missing step
	}
	if len(keys) == 0 {
		return &SV{}, false
	}
	last := len(keys) - 1
	parent, ok := s.GetOK(keys[:last]...)
	if !ok {
		return &SV{}, false
	}
	return parent.deleteSteps([]atStep{subscript("DeleteOK", keys[last])})
}

// deleteSteps removes what the last of steps leads to, walking the others
// without creating anything, for Delete, DeleteAt and DeleteOK, and says
// whether there was anything there to remove.
func (s *SV) deleteSteps(steps []atStep) (*SV, bool) {
	if len(steps) == 0 || s == nil {
		return &SV{}, false // s itself can't be deleted
	}
	cur := s
	for _, st := range steps[:len(steps)-1] {
		if cur = st.peek(cur); cur == nil {
			return &SV{}, false
		}
	}
	last := steps[len(steps)-1]
	switch v := cur.val().(type) {
	case *HV:
		if last.kind != stepIndex {
			_, there := v.m[last.key]
			return v.Delete(last.key), there // a locked hash panics either way
		}
	case *AV:
		if last.kind != stepKey {
			there := v.Exists(last.index)
			return v.Delete(last.index), there
		}
	}
	return &SV{}, false
}

// CheckPath returns an error if path is malformed, and nil if At, AtOK and
// DeleteAt can take it, for paths that come from outside the program, which
// would otherwise panic as a bad regex does.
func CheckPath(path string) error {
	_, err := parseAt(path)
	return err
}

var atCache = boundedCache[string, []atStep]{limit: cacheLimit}

// compileAt parses an At path, from the cache if it's been seen before. A
// malformed path panics, as a bad regex does.
func compileAt(path string) []atStep {
	if steps, ok := atCache.load(path); ok {
		return steps
	}
	steps, err := parseAt(path)
	if err != nil {
		panic(err.Error())
	}
	return atCache.store(path, steps)
}

// atError is how parseAt reports a malformed path: fail panics with one,
// and parseAt recovers it as an error.
type atError struct{ msg string }

// parseAt parses an At path into steps, or returns an error if it's
// malformed.
func parseAt(path string) (steps []atStep, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(atError)
			if !ok {
				panic(r)
			}
			steps, err = nil, fmt.Errorf("%s", e.msg)
		}
	}()
	fail := func(pos int, format string, args ...any) {
		panic(atError{fmt.Sprintf("scalar: bad path %q at offset %d: %s", path, pos, fmt.Sprintf(format, args...))})
	}
	i := 0
	if strings.HasPrefix(path, "$") {
		i = 1
	}
	afterDot := i == 0 // a name may start the path, as if after a dot
	if i == 1 && strings.HasPrefix(path[1:], ".") {
		i, afterDot = 2, true
	}
	for i < len(path) {
		switch {
		case path[i] == '[':
			step, end := parseBracket(path, i, fail)
			steps = append(steps, step)
			i = end
		case path[i] == '{':
			step, end := parseBrace(path, i, fail)
			steps = append(steps, step)
			i = end
		case afterDot:
			end := i
			for end < len(path) && path[end] != '.' && path[end] != '[' && path[end] != '{' {
				end++
			}
			name := path[i:end]
			if name == "" {
				fail(i, "expected a name")
			}
			if n, err := strconv.Atoi(name); err == nil && isInteger(name) {
				steps = append(steps, atStep{kind: stepEither, key: name, index: n})
			} else {
				steps = append(steps, atStep{kind: stepKey, key: name})
			}
			i = end
		default:
			fail(i, "expected ., [ or {")
		}
		afterDot = false
		if i < len(path) && path[i] == '.' {
			i++
			afterDot = true
			if i == len(path) {
				fail(i, "expected a name after .")
			}
			if path[i] == '[' || path[i] == '{' {
				fail(i, "expected a name after ., not %c", path[i])
			}
		}
	}
	return steps, nil
}

// parseBracket parses [n] or ["key"] or ['key'] starting at path[i], and
// returns the step and the offset just after the closing bracket.
func parseBracket(path string, i int, fail func(int, string, ...any)) (atStep, int) {
	i++ // past [
	if i < len(path) && (path[i] == '"' || path[i] == '\'') {
		return parseQuoted(path, i, ']', fail)
	}
	end := strings.IndexByte(path[i:], ']')
	if end < 0 {
		fail(i, "expected ]")
	}
	digits := path[i : i+end]
	n, err := strconv.Atoi(digits)
	if err != nil || !isInteger(digits) {
		fail(i, "expected an index or a quoted key in [...], got %q", digits)
	}
	return atStep{kind: stepIndex, index: n}, i + end + 1
}

// parseBrace parses {key}, {"key"} or {'key'} starting at path[i], and
// returns the step and the offset just after the closing brace.
func parseBrace(path string, i int, fail func(int, string, ...any)) (atStep, int) {
	i++ // past {
	if i < len(path) && (path[i] == '"' || path[i] == '\'') {
		return parseQuoted(path, i, '}', fail)
	}
	var key strings.Builder
	for start := i; ; i++ {
		switch {
		case i >= len(path):
			fail(i, "expected }")
		case path[i] == '\\' && i+1 < len(path):
			i++
			key.WriteByte(path[i])
		case path[i] == '}':
			if i == start {
				fail(i, "expected a key in {...}")
			}
			return atStep{kind: stepKey, key: key.String()}, i + 1
		default:
			key.WriteByte(path[i])
		}
	}
}

// parseQuoted parses a quoted key starting at path[i], the quote, which must
// be followed by closer, and returns the step and the offset just after the
// closer.
func parseQuoted(path string, i int, closer byte, fail func(int, string, ...any)) (atStep, int) {
	quote := path[i]
	var key strings.Builder
	for i++; ; i++ {
		switch {
		case i >= len(path):
			fail(i, "unterminated quoted key")
		case path[i] == '\\' && i+1 < len(path):
			i++
			key.WriteByte(path[i])
		case path[i] == quote:
			if i+1 >= len(path) || path[i+1] != closer {
				fail(i+1, "expected %c after the quoted key", closer)
			}
			return atStep{kind: stepKey, key: key.String()}, i + 2
		default:
			key.WriteByte(path[i])
		}
	}
}

// isInteger reports whether s is an optional minus sign and digits, nothing
// else: "0", "-1", but not "+1", " 1" or "1e3".
func isInteger(s string) bool {
	s = strings.TrimPrefix(s, "-")
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// The same, starting at a hash or an array rather than a scalar: h.At("a.b")
// is $h{a}{b}, and a.At("[0].name") is $a[0]{name}. Writing through the
// result changes the hash or array itself. A nil hash or array reads as
// undef, and writing through it panics, as Get on one does.

// At is SV.At, starting at h.
func (h *HV) At(path string) *SV { return h.ref().At(path) }

// AtOK is SV.AtOK, starting at h.
func (h *HV) AtOK(path string) (v *SV, ok bool) { return h.probe().AtOK(path) }

// DeleteAt is SV.DeleteAt, starting at h.
func (h *HV) DeleteAt(path string) *SV { return h.ref().DeleteAt(path) }

// DeleteOK is SV.DeleteOK, starting at h.
func (h *HV) DeleteOK(keys ...any) (v *SV, ok bool) { return h.probe().DeleteOK(keys...) }

// GetOK is SV.GetOK, starting at h.
func (h *HV) GetOK(keys ...any) (v *SV, ok bool) { return h.probe().GetOK(keys...) }

// Find is SV.Find, with $ standing for h.
func (h *HV) Find(path string) *AV { return h.probe().Find(path) }

// At is SV.At, starting at a.
func (a *AV) At(path string) *SV { return a.ref().At(path) }

// AtOK is SV.AtOK, starting at a.
func (a *AV) AtOK(path string) (v *SV, ok bool) { return a.probe().AtOK(path) }

// DeleteAt is SV.DeleteAt, starting at a.
func (a *AV) DeleteAt(path string) *SV { return a.ref().DeleteAt(path) }

// DeleteOK is SV.DeleteOK, starting at a.
func (a *AV) DeleteOK(keys ...any) (v *SV, ok bool) { return a.probe().DeleteOK(keys...) }

// GetOK is SV.GetOK, starting at a.
func (a *AV) GetOK(keys ...any) (v *SV, ok bool) { return a.probe().GetOK(keys...) }

// Find is SV.Find, with $ standing for a.
func (a *AV) Find(path string) *AV { return a.probe().Find(path) }

// probe is a scalar holding a reference to h, for GetOK, AtOK, DeleteOK and Find,
// which look without creating anything: nil for a nil h, which isn't there,
// as for a nil scalar.
func (h *HV) probe() *SV {
	if h == nil {
		return nil
	}
	return &SV{v: h}
}

// probe is a scalar holding a reference to a, as HV.probe is for a hash.
func (a *AV) probe() *SV {
	if a == nil {
		return nil
	}
	return &SV{v: a}
}

// ref is a scalar holding a reference to h, for walking a path from it.
// For a nil h, it's a scalar that can't be created, so reading through it is
// undef and writing panics.
func (h *HV) ref() *SV {
	if h == nil {
		return &SV{noCreate: "Not a HASH reference"}
	}
	return &SV{v: h}
}

// ref is a scalar holding a reference to a, as HV.ref is for a hash.
func (a *AV) ref() *SV {
	if a == nil {
		return &SV{noCreate: notArrayRef}
	}
	return &SV{v: a}
}
