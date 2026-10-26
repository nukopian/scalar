package scalar

import (
	"errors"
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// EXPERIMENTAL: a seam for regex engines other than regexp2. The default
// path in regex.go is untouched; a *Regexp compiled with CompileWith takes
// the path in this file instead, where scalar works from match positions
// alone, so any engine that can report them will do.

// Engine compiles patterns for a regex library. flags is any of "i", "m",
// "s" and "x"; an engine that can't honour one returns an error.
type Engine interface {
	Compile(pattern, flags string) (Matcher, error)
}

// Matcher is a pattern compiled by an Engine.
type Matcher interface {
	// All yields the successive matches of the pattern in s, each laid out
	// as regexp's FindStringSubmatchIndex lays it out: byte offsets into s,
	// a pair per group, and -1 for a group that didn't take part. An error,
	// such as a timeout, ends the matches.
	All(s string) iter.Seq2[[]int, error]
	// SubexpNames names the groups, as regexp's SubexpNames does: "" for
	// an unnamed group.
	SubexpNames() []string
	// String returns the pattern as written.
	String() string
}

// Stdlib is Go's regexp: RE2 syntax, in time linear in the input, so it
// never backtracks or times out. It has no lookaround, backreferences or
// atomic groups; \w, \d, \s and \b are ASCII; $ without "m" matches only at
// the very end; and it has no "x" flag.
var Stdlib Engine = stdlibEngine{}

// CompileWith is Compile, with engine in place of regexp2.
func CompileWith(engine Engine, pattern string, flags ...string) (*Regexp, error) {
	m, err := engine.Compile(pattern, string(parseFlags("CompileWith", "imsx", flags)))
	if err != nil {
		return nil, fmt.Errorf("scalar: bad regex %q: %w", pattern, err)
	}
	return &Regexp{m: m}, nil
}

// MustCompileWith is CompileWith, panicking if the pattern is invalid.
func MustCompileWith(engine Engine, pattern string, flags ...string) *Regexp {
	re, err := CompileWith(engine, pattern, flags...)
	if err != nil {
		panic(err.Error())
	}
	return re
}

// defaultEngine, if set, compiles pattern strings in place of regexp2, so
// that tests can run every pattern through another engine.
var defaultEngine Engine

type engineKey struct {
	engine         Engine
	pattern, flags string
}

var engineCache = boundedCache[engineKey, Matcher]{limit: cacheLimit}

func compileEngine(pattern string, f flagSet) Matcher {
	key := engineKey{defaultEngine, pattern, string(f)}
	if m, ok := engineCache.load(key); ok {
		return m
	}
	var flags []byte
	for _, c := range "imsx" {
		if f.has(c) {
			flags = append(flags, byte(c))
		}
	}
	m, err := defaultEngine.Compile(pattern, string(flags))
	if err != nil {
		panic(fmt.Sprintf("scalar: bad regex %q: %v", pattern, err))
	}
	return engineCache.store(key, m)
}

// checkEngine panics if a match failed, as check does.
func checkEngine(m Matcher, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if len(msg) > 200 {
		msg = strings.ToValidUTF8(msg[:200], "") + "..."
	}
	panic(fmt.Sprintf("scalar: regex %q: %s", m.String(), msg))
}

// engineMatches iterates over m's matches in str, panicking on an error.
func engineMatches(m Matcher, str string) iter.Seq[[]int] {
	return func(yield func([]int) bool) {
		for idx, err := range m.All(str) {
			checkEngine(m, err)
			if !yield(idx) {
				return
			}
		}
	}
}

func engineFirst(m Matcher, str string) []int {
	for idx := range engineMatches(m, str) {
		return idx
	}
	return nil
}

// indexAV is matchAV for a match laid out as positions.
func indexAV(str string, idx []int, isBytes bool) *AV {
	a := &AV{items: make([]*SV, len(idx)/2)}
	for i := range a.items {
		a.items[i] = indexSV(str, idx, i, isBytes)
	}
	return a
}

func indexSV(str string, idx []int, group int, isBytes bool) *SV {
	if idx[2*group] < 0 {
		return &SV{}
	}
	return textResult(str[idx[2*group]:idx[2*group+1]], isBytes)
}

func (s *SV) mEngine(m Matcher, f flagSet) *AV {
	str, isBytes := s.text()
	if !f.has('g') {
		if idx := engineFirst(m, str); idx != nil {
			return indexAV(str, idx, isBytes)
		}
		return nil
	}
	var out *AV
	for idx := range engineMatches(m, str) {
		if out == nil {
			out = &AV{}
		}
		out.items = append(out.items, &SV{v: indexAV(str, idx, isBytes)})
	}
	return out
}

func (s *SV) matchesEngine(m Matcher) iter.Seq[*AV] {
	str, isBytes := s.text()
	return func(yield func(*AV) bool) {
		for idx := range engineMatches(m, str) {
			if !yield(indexAV(str, idx, isBytes)) {
				return
			}
		}
	}
}

func (s *SV) namedEngine(m Matcher) *HV {
	str, isBytes := s.text()
	idx := engineFirst(m, str)
	if idx == nil {
		return nil
	}
	h := newHV(0)
	for i, name := range m.SubexpNames() {
		if name != "" {
			h.m[name] = indexSV(str, idx, i, isBytes)
		}
	}
	return h
}

func (s *SV) sEngine(m Matcher, repl any, limit int, f flagSet) *SV {
	str, isBytes := s.text()
	var replace func(idx []int) string
	switch r := repl.(type) {
	case string:
		names := m.SubexpNames()
		replace = func(idx []int) string { return expandIndex(r, str, idx, names) }
	case func(*AV) any:
		replace = func(idx []int) string {
			text, _ := operand(r(indexAV(str, idx, isBytes))).text()
			return text
		}
	case func(*AV) string:
		replace = func(idx []int) string { return r(indexAV(str, idx, isBytes)) }
	default:
		panic(fmt.Sprintf("scalar: S replacement must be a string or func(*AV) any, got %T", repl))
	}
	var out strings.Builder
	n, last := 0, 0
	for idx := range engineMatches(m, str) {
		out.WriteString(str[last:idx[0]])
		out.WriteString(replace(idx))
		last = idx[1]
		if n++; n == limit {
			break
		}
	}
	if n == 0 {
		return s.substituted(str, 0, isBytes, f)
	}
	out.WriteString(str[last:])
	return s.substituted(out.String(), n, isBytes, f)
}

// expandIndex is expand for a match laid out as positions, with the same
// rules, which TestRegexp2ThroughSeam checks by running S's expansion
// tests through the seam.
func expandIndex(repl, str string, idx []int, names []string) string {
	if !strings.Contains(repl, "$") {
		return repl
	}
	took := func(n int) bool { return n < len(idx)/2 && idx[2*n] >= 0 }
	group := func(n int) string {
		if !took(n) {
			return ""
		}
		return str[idx[2*n]:idx[2*n+1]]
	}
	named := func(name string) (string, bool) {
		k := indexOf(names, name)
		if k < 0 {
			return "", false
		}
		return group(k), true
	}

	var b strings.Builder
	for i := 0; i < len(repl); i++ {
		if repl[i] != '$' || i+1 == len(repl) {
			b.WriteByte(repl[i])
			continue
		}
		rest := repl[i+1:]
		switch c := rest[0]; {
		case c == '$':
			b.WriteByte('$')
			i++
		case c == '&':
			b.WriteString(group(0))
			i++
		case c == '`':
			b.WriteString(str[:idx[0]])
			i++
		case c == '\'':
			b.WriteString(str[idx[1]:])
			i++
		case c >= '0' && c <= '9':
			j := 1
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			n, err := strconv.Atoi(rest[:j])
			if err == nil {
				b.WriteString(group(n))
			}
			i += j
		case strings.HasPrefix(rest, "+{") && strings.Contains(rest, "}"):
			end := strings.IndexByte(rest, '}')
			text, _ := named(rest[2:end])
			b.WriteString(text)
			i += end + 1
		case c == '+' && !strings.HasPrefix(rest, "+{"):
			for n := len(idx)/2 - 1; n > 0; n-- {
				if took(n) {
					b.WriteString(group(n))
					break
				}
			}
			i++
		case c == '{' && strings.Contains(rest, "}"):
			end := strings.IndexByte(rest, '}')
			name := rest[1:end]
			if n, err := strconv.Atoi(name); err == nil {
				b.WriteString(group(n))
			} else if text, ok := named(name); ok {
				b.WriteString(text)
			} else {
				b.WriteString(repl[i : i+end+2])
			}
			i += end + 1
		default:
			b.WriteByte('$')
		}
	}
	return b.String()
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}

func (s *SV) splitEngine(m Matcher, str string, isBytes bool, lim int) *AV {
	out := &AV{}
	pos, splits := 0, 0
	for idx := range engineMatches(m, str) {
		if lim > 0 && splits >= lim-1 {
			break
		}
		// A zero-width match can't split where the last match ended, or at
		// the start of the string.
		if idx[0] == idx[1] && idx[0] == pos {
			continue
		}
		out.Push(textResult(str[pos:idx[0]], isBytes))
		for g := 1; g < len(idx)/2; g++ {
			out.Push(indexSV(str, idx, g, isBytes))
		}
		pos = idx[1]
		splits++
	}
	out.Push(textResult(str[pos:], isBytes))
	return out
}

// Go's regexp.

type stdlibEngine struct{}

func (stdlibEngine) Compile(pattern, flags string) (Matcher, error) {
	if strings.ContainsRune(flags, 'x') {
		return nil, errors.New(`no "x" flag in Go's regexp`)
	}
	src := pattern
	if flags != "" {
		src = "(?" + flags + ")" + pattern
	}
	rx, err := regexp.Compile(src)
	if err != nil {
		return nil, err
	}
	return stdlibMatcher{rx, pattern}, nil
}

type stdlibMatcher struct {
	rx      *regexp.Regexp
	pattern string
}

func (m stdlibMatcher) All(s string) iter.Seq2[[]int, error] {
	return func(yield func([]int, error) bool) {
		for _, idx := range m.rx.FindAllStringSubmatchIndex(s, -1) {
			if !yield(idx, nil) {
				return
			}
		}
	}
}

func (m stdlibMatcher) SubexpNames() []string { return m.rx.SubexpNames() }
func (m stdlibMatcher) String() string        { return m.pattern }

// regexp2 through the seam, to measure what the seam itself costs.

type regexp2Engine struct{}

func (regexp2Engine) Compile(pattern, flags string) (Matcher, error) {
	rx, err := compileErr(pattern, flagSet(flags))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, n := range rx.GetGroupNumbers() {
		name := rx.GroupNameFromNumber(n)
		if _, err := strconv.Atoi(name); err == nil {
			name = ""
		}
		names = append(names, name)
	}
	return regexp2Matcher{rx, names}, nil
}

type regexp2Matcher struct {
	rx    *regexp2.Regexp
	names []string
}

// All converts regexp2's rune offsets to byte offsets.
func (m regexp2Matcher) All(s string) iter.Seq2[[]int, error] {
	return func(yield func([]int, error) bool) {
		runes := []rune(s)
		byteAt := make([]int, 0, len(runes)+1)
		for off := range s {
			byteAt = append(byteAt, off)
		}
		byteAt = append(byteAt, len(s))
		match, err := m.rx.FindRunesMatch(runes)
		for {
			if err != nil {
				yield(nil, err)
				return
			}
			if match == nil {
				return
			}
			groups := match.Groups()
			idx := make([]int, 2*len(groups))
			for i, g := range groups {
				if len(g.Captures) == 0 {
					idx[2*i], idx[2*i+1] = -1, -1
					continue
				}
				idx[2*i], idx[2*i+1] = byteAt[g.Index], byteAt[g.Index+g.Length]
			}
			if !yield(idx, nil) {
				return
			}
			match, err = m.rx.FindNextMatch(match)
		}
	}
}

func (m regexp2Matcher) SubexpNames() []string { return m.names }
func (m regexp2Matcher) String() string        { return m.rx.String() }
