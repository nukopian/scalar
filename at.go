package scalar

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// stepKind is what a step of an At path is.
type stepKind int

const (
	stepKey    stepKind = iota // a hash key: name, or ["quoted"]
	stepIndex                  // an array index: [0]
	stepEither                 // .0 after a dot: an index into an array, or a key into a hash
)

type atStep struct {
	kind  stepKind
	key   string // for stepKey, and the digits of a stepEither
	index int    // for stepIndex and stepEither
}

// At is Path with the route as one string, in dot or bracket form:
//
//	s.At("content[0].rule_set")     // $s->{content}[0]{rule_set}
//	s.At("content.0.rule_set")      // the same, if content holds an array
//	s.At(`meta["log.level"]`)       // a key with dots in it, quoted
//	s.At("items[-1].name")          // negative indices count from the end
//
// A name, or a quoted ["key"] or ['key'], is a hash key; a backslash in a
// quoted key escapes the next character, as in ["say \"hi\""]. [n] is an
// array index. A number after a dot, as in content.0, is whichever the
// container is: an index into an array, or the key "0" into a hash; writing
// through one that doesn't exist yet creates an array, so quote a hash key
// that looks like a number: ["0"]. A leading $ is allowed, so $.a[0] works,
// as in JSONPath. An empty path is s itself.
//
// It walks as Path does: intermediate levels autovivify, the final element
// doesn't until it's written to, and a missing step reads as undef. A
// malformed path panics, as a bad regex does.
func (s *SV) At(path string) *SV {
	cur := s
	for _, st := range compileAt(path) {
		cur = st.walk(cur)
	}
	return cur
}

// walk takes one step from cur, as Path does: through a hash or an array,
// autovivifying cur if it's undef.
func (st atStep) walk(cur *SV) *SV {
	switch st.kind {
	case stepKey:
		return cur.HV().Get(st.key)
	case stepIndex:
		return cur.AV().Get(st.index)
	}
	if _, isHash := cur.val().(*HV); isHash { // stepEither
		return cur.HV().Get(st.key)
	}
	return cur.AV().Get(st.index) // an array, or nothing yet: an array
}

// peek takes one step from cur, as PathOK does, without creating anything:
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

// AtOK is PathOK with the route as one string, written as for At: ok
// reports whether every step exists, so a value that's present but null
// gives ok true. Like PathOK, it never creates anything, and when ok is true,
// v is the stored value itself.
func (s *SV) AtOK(path string) (v *SV, ok bool) {
	return s.PathOK(&PathExpr{steps: compileAt(path)})
}

// PathExpr is a compiled At path, like a compiled regex: Path and PathOK
// accept one among their keys, so a path used often is parsed once, and a
// fixed prefix can be combined with keys that vary:
//
//	rule := scalar.MustPath("content[0].rule_set")
//	s.Path(rule)                // s.At("content[0].rule_set")
//	s.Path(rule, "name", 2)     // ...{name}[2]
type PathExpr struct {
	src   string
	steps []atStep
}

// CompilePath compiles an At path, or returns an error if it's malformed,
// for paths that come from outside the program.
func CompilePath(path string) (*PathExpr, error) {
	steps, err := parseAt(path)
	if err != nil {
		return nil, err
	}
	return &PathExpr{src: path, steps: steps}, nil
}

// MustPath is CompilePath, panicking if the path is malformed. Use it for
// paths in the program, in package-level variables, so that a bad one fails
// as soon as the program starts.
func MustPath(path string) *PathExpr {
	return &PathExpr{src: path, steps: compileAt(path)}
}

// String returns the path as written.
func (p *PathExpr) String() string { return p.src }

var atCache sync.Map // string → []atStep

// compileAt parses an At path, from the cache if it's been seen before. A
// malformed path panics, as a bad regex does.
func compileAt(path string) []atStep {
	if steps, ok := atCache.Load(path); ok {
		return steps.([]atStep)
	}
	steps, err := parseAt(path)
	if err != nil {
		panic(err.Error())
	}
	atCache.Store(path, steps)
	return steps
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
		case afterDot:
			end := i
			for end < len(path) && path[end] != '.' && path[end] != '[' {
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
			fail(i, "expected . or [")
		}
		afterDot = false
		if i < len(path) && path[i] == '.' {
			i++
			afterDot = true
			if i == len(path) {
				fail(i, "expected a name after .")
			}
			if path[i] == '[' {
				fail(i, "expected a name after ., not [")
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
				if i+1 >= len(path) || path[i+1] != ']' {
					fail(i+1, "expected ] after the quoted key")
				}
				return atStep{kind: stepKey, key: key.String()}, i + 2
			default:
				key.WriteByte(path[i])
			}
		}
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

// isInteger reports whether s is an optional minus sign and digits, nothing
// else: "0", "-1", but not "+1", " 1" or "1e3".
func isInteger(s string) bool {
	s = strings.TrimPrefix(s, "-")
	return s != "" && strings.Trim(s, "0123456789") == ""
}
