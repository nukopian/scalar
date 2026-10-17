package scalar

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// Find runs a JSONPath query (RFC 9535) on s and returns every match, or nil
// if nothing matched. Like Get and Path, the matches are the stored values
// themselves, so changing one changes s. Matches of objects and arrays are
// references: use HV(), AV(), Path or Find on them to go further.
//
// Supported: the root $, member names (.name or ['name']), indexes ([0],
// [-1]), wildcards (.* or [*]), descendants (..name, ..*, ..[0]), slices
// ([start:end:step]) and unions ([0,2], ['a','b']). Filter expressions
// ([?...]) aren't: use Grep, FirstMatch and the other list utilities on the
// result instead.
//
// Unlike Path, Find never creates anything: a missing key or index simply
// matches nothing. A malformed path panics, as a bad regex does.
func (s *SV) Find(path string) *AV {
	if s == nil {
		return nil
	}
	nodes := []*SV{s}
	for _, seg := range compilePath(path) {
		var next []*SV
		for _, n := range nodes {
			if seg.descendant {
				visited := map[any]bool{}
				walk(n, visited, func(d *SV) { next = seg.apply(d, next) })
			} else {
				next = seg.apply(n, next)
			}
		}
		if len(next) == 0 {
			return nil
		}
		nodes = next
	}
	return &AV{items: nodes}
}

// walk visits n and then everything below it, in document order. JSON is a
// tree, but references can make an array or hash appear more than once, or
// even contain itself; visited makes sure each is visited only once.
func walk(n *SV, visited map[any]bool, visit func(*SV)) {
	switch v := n.v.(type) {
	case *AV, *HV:
		if visited[v] {
			return
		}
		visited[v] = true
	}
	visit(n)
	children(n, func(c *SV) { walk(c, visited, visit) })
}

// children calls f for each element of the array or hash that n refers to.
// Array holes don't exist, so they're skipped.
func children(n *SV, f func(*SV)) {
	switch v := n.v.(type) {
	case *AV:
		for _, c := range v.items {
			if c != nil {
				f(c)
			}
		}
	case *HV:
		for _, k := range v.keys() {
			f(v.m[k])
		}
	}
}

type segment struct {
	descendant bool // .. rather than . or [...]
	selectors  []selector
}

type selKind int

const (
	selName selKind = iota
	selIndex
	selSlice
	selWild
)

type selector struct {
	kind  selKind
	name  string
	index int
	// Slice bounds. A missing start or end takes the RFC default, which
	// depends on the direction of step.
	start, end       int
	hasStart, hasEnd bool
	step             int
}

// apply adds n's children selected by seg to out.
func (seg segment) apply(n *SV, out []*SV) []*SV {
	for _, sel := range seg.selectors {
		out = sel.apply(n, out)
	}
	return out
}

func (sel selector) apply(n *SV, out []*SV) []*SV {
	switch sel.kind {
	case selWild:
		children(n, func(c *SV) { out = append(out, c) })
	case selName:
		if h, ok := n.v.(*HV); ok {
			if c, ok := h.m[sel.name]; ok {
				out = append(out, c)
			}
		}
	case selIndex:
		if a, ok := n.v.(*AV); ok {
			if i, ok := a.index(sel.index); ok && i < len(a.items) && a.items[i] != nil {
				out = append(out, a.items[i])
			}
		}
	case selSlice:
		if a, ok := n.v.(*AV); ok {
			for _, i := range sel.sliceIndexes(len(a.items)) {
				if a.items[i] != nil {
					out = append(out, a.items[i])
				}
			}
		}
	}
	return out
}

// sliceIndexes follows RFC 9535 section 2.3.4.2.2.
func (sel selector) sliceIndexes(n int) []int {
	step := sel.step
	if step == 0 {
		return nil
	}
	normalize := func(i int) int {
		if i < 0 {
			return n + i
		}
		return i
	}
	start, end := 0, n
	if step < 0 {
		start, end = n-1, -n-1
	}
	if sel.hasStart {
		start = sel.start
	}
	if sel.hasEnd {
		end = sel.end
	}
	start, end = normalize(start), normalize(end)

	var out []int
	if step > 0 {
		lower, upper := min(max(start, 0), n), min(max(end, 0), n)
		for i := lower; i < upper; i += step {
			out = append(out, i)
		}
	} else {
		upper, lower := min(max(start, -1), n-1), min(max(end, -1), n-1)
		for i := upper; lower < i; i += step {
			out = append(out, i)
		}
	}
	return out
}

var pathCache sync.Map // string → []segment

func compilePath(path string) []segment {
	if segs, ok := pathCache.Load(path); ok {
		return segs.([]segment)
	}
	p := &pathParser{src: path}
	segs := p.parse()
	pathCache.Store(path, segs)
	return segs
}

type pathParser struct {
	src string
	pos int
}

func (p *pathParser) fail(format string, args ...any) {
	panic(fmt.Sprintf("scalar: bad JSONPath %q at offset %d: %s", p.src, p.pos, fmt.Sprintf(format, args...)))
}

func (p *pathParser) done() bool { return p.pos >= len(p.src) }

func (p *pathParser) peek() byte {
	if p.done() {
		return 0
	}
	return p.src[p.pos]
}

func (p *pathParser) eat(prefix string) bool {
	if strings.HasPrefix(p.src[p.pos:], prefix) {
		p.pos += len(prefix)
		return true
	}
	return false
}

func (p *pathParser) skipSpace() {
	for !p.done() && strings.IndexByte(" \t\n\r", p.peek()) >= 0 {
		p.pos++
	}
}

func (p *pathParser) parse() []segment {
	if !p.eat("$") {
		p.fail("a path must start with $")
	}
	var segs []segment
	for {
		// Space may come before a segment, but not at the end.
		before := p.pos
		p.skipSpace()
		if p.done() {
			if p.pos > before {
				p.fail("space after the last segment")
			}
			break
		}
		switch {
		case p.eat(".."):
			segs = append(segs, segment{descendant: true, selectors: p.dotOrBracket()})
		case p.eat("."):
			if p.peek() == '[' {
				p.fail("expected a name or * after .")
			}
			segs = append(segs, segment{selectors: p.dotOrBracket()})
		case p.peek() == '[':
			segs = append(segs, segment{selectors: p.bracket()})
		default:
			p.fail("expected ., .. or [")
		}
	}
	return segs
}

// dotOrBracket parses what follows . or ..: a name, *, or (after .. only)
// a bracketed selector list.
func (p *pathParser) dotOrBracket() []selector {
	switch {
	case p.peek() == '[':
		return p.bracket()
	case p.eat("*"):
		return []selector{{kind: selWild}}
	}
	return []selector{{kind: selName, name: p.memberName()}}
}

// memberName parses a shorthand name: a letter, _ or non-ASCII character,
// then any of those or digits. Other names need the ['name'] form.
func (p *pathParser) memberName() string {
	start := p.pos
	for !p.done() {
		r, size := utf8.DecodeRuneInString(p.src[p.pos:])
		ok := r == '_' || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r >= utf8.RuneSelf
		if p.pos > start {
			ok = ok || '0' <= r && r <= '9'
		}
		if !ok {
			break
		}
		p.pos += size
	}
	if p.pos == start {
		p.fail("expected a member name")
	}
	return p.src[start:p.pos]
}

// bracket parses [selector, selector, ...].
func (p *pathParser) bracket() []selector {
	p.eat("[")
	var sels []selector
	for {
		p.skipSpace()
		sels = append(sels, p.selector())
		p.skipSpace()
		if p.eat("]") {
			return sels
		}
		if !p.eat(",") {
			p.fail("expected , or ]")
		}
	}
}

func (p *pathParser) selector() selector {
	switch c := p.peek(); {
	case c == '*':
		p.pos++
		return selector{kind: selWild}
	case c == '\'' || c == '"':
		return selector{kind: selName, name: p.stringLiteral()}
	case c == '?':
		p.fail("filter expressions aren't supported; use Grep or FirstMatch on the result")
	}

	start, hasStart := p.integer()
	p.skipSpace()
	if !p.eat(":") {
		if !hasStart {
			p.fail("expected a name, index, slice or *")
		}
		return selector{kind: selIndex, index: start}
	}
	sel := selector{kind: selSlice, start: start, hasStart: hasStart, step: 1}
	p.skipSpace()
	sel.end, sel.hasEnd = p.integer()
	p.skipSpace()
	if p.eat(":") {
		p.skipSpace()
		if step, ok := p.integer(); ok {
			sel.step = step
		}
	}
	return sel
}

// maxExact is the largest integer an index or slice bound may be, as RFC
// 9535 says: the largest a JSON number holds exactly, 2^53-1.
const maxExact = 1<<53 - 1

// integer parses an optional signed integer, as RFC 9535 writes one: 0,
// or digits not starting with 0, perhaps after a minus, so not "01" or
// "-0", and within ±maxExact.
func (p *pathParser) integer() (int, bool) {
	start := p.pos
	p.eat("-")
	digits := p.pos
	for !p.done() && '0' <= p.peek() && p.peek() <= '9' {
		p.pos++
	}
	text := p.src[digits:p.pos]
	if text == "" {
		p.pos = start
		return 0, false
	}
	if text[0] == '0' && (len(text) > 1 || digits > start) {
		p.pos = start
		p.fail("an integer can't start with 0, or be -0")
	}
	n, err := strconv.Atoi(p.src[start:p.pos])
	if err != nil || n > maxExact || n < -maxExact {
		p.pos = start
		p.fail("an integer must be within ±%d", maxExact)
	}
	return n, true
}

// stringLiteral parses a quoted name, with JSON-style escapes plus \'.
func (p *pathParser) stringLiteral() string {
	quote := p.src[p.pos]
	p.pos++
	var b strings.Builder
	for {
		if p.done() {
			p.fail("unterminated string")
		}
		c := p.src[p.pos]
		p.pos++
		switch {
		case c == quote:
			return b.String()
		case c == '\\':
			p.escape(&b, quote)
		case c < 0x20:
			p.pos--
			p.fail("a control character (U+%04X) must be escaped", c)
		default:
			b.WriteByte(c)
		}
	}
}

// escape parses an escape after a backslash in a string quoted with
// quote: JSON's, but for the other quote, which needs none ("\'" is only
// in a single-quoted string, "\"" in a double-quoted one).
func (p *pathParser) escape(b *strings.Builder, quote byte) {
	if p.done() {
		p.fail("unterminated escape")
	}
	c := p.src[p.pos]
	p.pos++
	switch c {
	case 'b':
		b.WriteByte('\b')
	case 'f':
		b.WriteByte('\f')
	case 'n':
		b.WriteByte('\n')
	case 'r':
		b.WriteByte('\r')
	case 't':
		b.WriteByte('\t')
	case '/', '\\', quote:
		b.WriteByte(c)
	case 'u':
		// A character outside the Basic Multilingual Plane is written as a
		// surrogate pair, \uD8xx\uDCxx; either half on its own is invalid.
		r := p.hex4()
		switch {
		case 0xDC00 <= r && r < 0xE000:
			p.fail("unpaired low surrogate \\u%04X", r)
		case 0xD800 <= r && r < 0xDC00:
			if !p.eat(`\u`) {
				p.fail("unpaired high surrogate \\u%04X", r)
			}
			lo := p.hex4()
			if lo < 0xDC00 || lo >= 0xE000 {
				p.fail("high surrogate \\u%04X followed by \\u%04X, not a low surrogate", r, lo)
			}
			r = 0x10000 + (r-0xD800)<<10 + (lo - 0xDC00)
		}
		b.WriteRune(r)
	default:
		p.pos--
		p.fail("unknown escape \\%c", c)
	}
}

func (p *pathParser) hex4() rune {
	if p.pos+4 > len(p.src) {
		p.fail("short \\u escape")
	}
	n, err := strconv.ParseUint(p.src[p.pos:p.pos+4], 16, 32)
	if err != nil {
		p.fail("bad \\u escape")
	}
	p.pos += 4
	return rune(n)
}
