package scalar

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// FromDump reads a structure written by Dump, or by Perl's Data::Dumper, and
// returns it, as FromJSON reads JSON. Dump's output reads back as an equal
// structure, shared parts and loops included, so FromDump(s.Dump()) is a
// copy of s. It also makes test data quick to write:
//
//	want := scalar.MustFromDump(`{place => "Cobb, CA", stations => ["PAS", "GSC"]}`)
//
// It reads hashes, arrays, undef, !!1 and !!0, numbers, and strings in
// double quotes, with Perl's escapes (\n, \t, \x{263a}, octal such as \351,
// \$, \@ and the rest), or in single quotes, where only \\ and \' are
// escapes. Hash keys may be quoted or bare words and numbers, and between a
// key and its value, => or a comma. Spaces, line breaks, trailing commas and
// # comments are allowed anywhere Perl allows them. $VAR1, and paths from it
// such as $VAR1->{tags}[0], are the array or hash at that place, written
// earlier in the dump, so shared parts and loops come back as they were.
// bless(..., 'Class') gives the array or hash inside it: scalar has no
// classes, so the class is dropped.
//
// A dump doesn't record types, so values come back with the types their
// text gives them, as Perl reads a dump back: an integer written bare, up
// to 9 digits, is an integer, so a string "42" comes back as the number 42;
// a quoted value is a string, so a float, written "2.5", and a longer
// integer, written "12345678901", come back as strings. They compare and
// calculate the same, as in Perl. A string is a byte string if it has an
// octal or \xHH escape above 127, as Dump writes one, or raw bytes that
// aren't UTF-8, as Data::Dumper writes one without Useqq, and no \x{...}
// escape; any other string is text, including one with characters written
// as they are, in UTF-8, as test data might have them. A $ or @ in a double-quoted string
// is taken as it is, not interpolated.
//
// Arrays and hashes may nest up to 10,000 levels deep, as with
// encoding/json; deeper is an error. It doesn't run any Perl: references to plain values (\'x'), code, regexes
// and anything else that isn't data give an error, saying where, as does
// the "$VAR1 = ...;" that Data::Dumper writes without Terse.
func FromDump(dump string) (s *SV, err error) {
	p := dumpParser{src: dump}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(dumpError)
			if !ok {
				panic(r)
			}
			s, err = nil, e
		}
	}()
	p.space()
	if p.pos == len(p.src) {
		p.fail("there's nothing to read")
	}
	p.value(func(v *SV) { s = v })
	p.space()
	if p.pos < len(p.src) {
		p.fail("unexpected %s after the value", p.describeNext())
	}
	return s, nil
}

// MustFromDump is FromDump, panicking if the dump is malformed. Use it for
// test data and other dumps written in the program.
func MustFromDump(dump string) *SV {
	s, err := FromDump(dump)
	if err != nil {
		panic(err.Error())
	}
	return s
}

// dumpError is a malformed dump, where it went wrong and why.
type dumpError struct {
	line, column int
	msg          string
}

func (e dumpError) Error() string {
	return fmt.Sprintf("scalar: bad dump at line %d, column %d: %s", e.line, e.column, e.msg)
}

// maxDumpDepth is how deeply arrays and hashes may nest in a dump, as
// encoding/json and the YAML package limit it: deeper would risk running
// out of stack.
const maxDumpDepth = 10000

type dumpParser struct {
	src   string
	pos   int
	root  *SV // the top-level value, for $VAR1
	depth int // how many arrays and hashes deep the parser is
}

// enter goes one array or hash deeper, failing past maxDumpDepth; the
// caller defers leave.
func (p *dumpParser) enter() {
	p.depth++
	if p.depth > maxDumpDepth {
		p.fail("arrays and hashes are nested more than %d levels deep", maxDumpDepth)
	}
}

func (p *dumpParser) leave() { p.depth-- }

// fail panics with a dumpError at the current position; FromDump recovers it.
func (p *dumpParser) fail(format string, args ...any) {
	before := p.src[:p.pos]
	line := strings.Count(before, "\n") + 1
	column := utf8.RuneCountInString(before[strings.LastIndexByte(before, '\n')+1:]) + 1
	panic(dumpError{line, column, fmt.Sprintf(format, args...)})
}

// describeNext names what's at the current position, for an error.
func (p *dumpParser) describeNext() string {
	if p.pos >= len(p.src) {
		return "end of the dump"
	}
	r, _ := utf8.DecodeRuneInString(p.src[p.pos:])
	return strconv.QuoteRune(r)
}

// space skips spaces, line breaks and # comments.
func (p *dumpParser) space() {
	for p.pos < len(p.src) {
		switch c := p.src[p.pos]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			p.pos++
		case c == '#':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

// expect skips space, then the given text, or fails.
func (p *dumpParser) expect(what string) {
	p.space()
	if !strings.HasPrefix(p.src[p.pos:], what) {
		p.fail("expected %q, found %s", what, p.describeNext())
	}
	p.pos += len(what)
}

// word reads a bare word: letters, digits and underscores, not starting
// with a digit.
func (p *dumpParser) word() string {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || p.pos > start && '0' <= c && c <= '9' {
			p.pos++
			continue
		}
		break
	}
	return p.src[start:p.pos]
}

// value reads one value and passes it to attach, which stores it in its
// container. An array or hash is attached before its contents are read, so
// that a $VAR1 path inside it can lead back to it, as in a loop.
func (p *dumpParser) value(attach func(*SV)) {
	p.space()
	if p.pos >= len(p.src) {
		p.fail("expected a value, found the end of the dump")
	}
	switch c := p.src[p.pos]; {
	case c == '{':
		p.hash(attach)
	case c == '[':
		p.array(attach)
	case c == '"':
		attach(p.doubleQuoted())
	case c == '\'':
		if str := p.singleQuoted(); utf8.ValidString(str) {
			attach(&SV{v: str})
		} else {
			attach(&SV{v: []byte(str)}) // raw bytes, as Data::Dumper writes a byte string without Useqq
		}
	case c == '$':
		attach(p.reference())
	case c == '!':
		switch {
		case strings.HasPrefix(p.src[p.pos:], "!!1"):
			attach(&SV{v: true})
		case strings.HasPrefix(p.src[p.pos:], "!!0"):
			attach(&SV{v: false})
		default:
			p.fail("expected !!1 or !!0")
		}
		p.pos += 3
	case c == '-' || c == '+' || c == '.' || '0' <= c && c <= '9':
		attach(p.number())
	case c == '\\':
		p.fail("a reference to a plain value, such as \\'x', can't be read: scalar holds references only to arrays and hashes")
	default:
		start := p.pos
		switch w := p.word(); w {
		case "undef":
			attach(&SV{})
		case "bless":
			p.expect("(")
			p.value(attach) // the array or hash, without its class
			p.expect(",")
			p.space()
			switch {
			case p.pos < len(p.src) && p.src[p.pos] == '\'':
				p.singleQuoted()
			case p.pos < len(p.src) && p.src[p.pos] == '"':
				p.doubleQuoted()
			default:
				p.fail("expected the class name, in quotes, found %s", p.describeNext())
			}
			p.expect(")")
		case "sub", "qr", "do":
			p.pos = start
			p.fail("%s can't be read: FromDump reads data, not Perl code", w)
		case "":
			p.fail("expected a value, found %s", p.describeNext())
		default:
			p.pos = start
			p.fail("expected a value, found the word %q", w)
		}
	}
}

// container records sv as the top-level value if it's the first one read,
// for $VAR1.
func (p *dumpParser) container(sv *SV) {
	if p.root == nil {
		p.root = sv
	}
}

func (p *dumpParser) hash(attach func(*SV)) {
	p.enter()
	defer p.leave()
	p.pos++ // past {
	h := newHV(0)
	sv := &SV{v: h}
	p.container(sv)
	attach(sv)
	for {
		p.space()
		if p.pos < len(p.src) && p.src[p.pos] == '}' {
			p.pos++
			return
		}
		key := p.key()
		p.space()
		switch {
		case strings.HasPrefix(p.src[p.pos:], "=>"):
			p.pos += 2
		case strings.HasPrefix(p.src[p.pos:], ","):
			p.pos++
		default:
			p.fail("expected => after the key %q, found %s", key, p.describeNext())
		}
		p.value(func(v *SV) { h.store(key, v) })
		p.endOfItem('}')
	}
}

func (p *dumpParser) array(attach func(*SV)) {
	p.enter()
	defer p.leave()
	p.pos++ // past [
	a := &AV{}
	sv := &SV{v: a}
	p.container(sv)
	attach(sv)
	for {
		p.space()
		if p.pos < len(p.src) && p.src[p.pos] == ']' {
			p.pos++
			return
		}
		p.value(func(v *SV) { a.items = append(a.items, v) })
		p.endOfItem(']')
	}
}

// endOfItem skips the comma after an element, or stops before the closing
// bracket.
func (p *dumpParser) endOfItem(closing byte) {
	p.space()
	switch {
	case p.pos < len(p.src) && p.src[p.pos] == ',':
		p.pos++
	case p.pos < len(p.src) && p.src[p.pos] == closing:
	default:
		p.fail("expected a comma or %q, found %s", string(closing), p.describeNext())
	}
}

// key reads a hash key: quoted, a bare word, or a bare number.
func (p *dumpParser) key() string {
	p.space()
	if p.pos >= len(p.src) {
		p.fail("expected a key, found the end of the dump")
	}
	switch c := p.src[p.pos]; {
	case c == '"':
		return p.doubleQuoted().Str()
	case c == '\'':
		return p.singleQuoted()
	case c == '-' || '0' <= c && c <= '9':
		start := p.pos
		if c == '-' {
			p.pos++
		}
		for p.pos < len(p.src) && '0' <= p.src[p.pos] && p.src[p.pos] <= '9' {
			p.pos++
		}
		if k := p.src[start:p.pos]; k != "-" {
			return k
		}
		p.pos = start
	}
	if w := p.word(); w != "" {
		return w
	}
	p.fail("expected a key, found %s", p.describeNext())
	return ""
}

// number reads a Perl numeric literal: an integer, or a decimal with an
// optional exponent, with _ allowed between digits, as in 1_000.
func (p *dumpParser) number() *SV {
	start := p.pos
	if c := p.src[p.pos]; c == '-' || c == '+' {
		p.pos++
	}
	digits := func() bool {
		from := p.pos
		for p.pos < len(p.src) && ('0' <= p.src[p.pos] && p.src[p.pos] <= '9' || p.src[p.pos] == '_' && p.pos > from) {
			p.pos++
		}
		return p.pos > from
	}
	whole := digits()
	isFloat := false
	if p.pos < len(p.src) && p.src[p.pos] == '.' {
		p.pos++
		isFloat = true
		if !digits() && !whole {
			p.pos = start
			p.fail("expected a number")
		}
	} else if !whole {
		p.pos = start
		p.fail("expected a value, found %s", p.describeNext())
	}
	if p.pos < len(p.src) && (p.src[p.pos] == 'e' || p.src[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.src) && (p.src[p.pos] == '-' || p.src[p.pos] == '+') {
			p.pos++
		}
		if !digits() {
			p.fail("expected the exponent's digits")
		}
		isFloat = true
	}
	lit := strings.ReplaceAll(p.src[start:p.pos], "_", "")
	if !isFloat {
		if i, err := strconv.ParseInt(lit, 10, 64); err == nil {
			return &SV{v: i}
		}
	}
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil && f == 0 {
		p.pos = start
		p.fail("can't read the number %q", lit)
	}
	return &SV{v: f} // out of range: ±Inf, as in Perl
}

// singleQuoted reads a '...' string, where only \\ and \' are escapes.
func (p *dumpParser) singleQuoted() string {
	p.pos++ // past '
	var b strings.Builder
	for {
		if p.pos >= len(p.src) {
			p.fail("this string has no closing '")
		}
		c := p.src[p.pos]
		switch {
		case c == '\'':
			p.pos++
			return b.String()
		case c == '\\' && p.pos+1 < len(p.src) && (p.src[p.pos+1] == '\\' || p.src[p.pos+1] == '\''):
			b.WriteByte(p.src[p.pos+1])
			p.pos += 2
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
}

// cEscapeChars are the letters of Perl's C-style escapes, and what they
// stand for.
var cEscapeChars = map[byte]rune{'n': '\n', 't': '\t', 'r': '\r', 'f': '\f', 'b': '\b', 'a': '\a', 'e': '\x1b'}

// doubleQuoted reads a "..." string with Perl's escapes, as a byte string
// if it has an octal or \xHH escape above 127 and nothing beyond, and as
// text otherwise.
func (p *dumpParser) doubleQuoted() *SV {
	start := p.pos
	p.pos++ // past "
	var chars []rune
	highByte, wide := false, false
	for {
		if p.pos >= len(p.src) {
			p.pos = start
			p.fail(`this string has no closing "`)
		}
		c := p.src[p.pos]
		if c == '"' {
			p.pos++
			break
		}
		if c != '\\' {
			r, size := utf8.DecodeRuneInString(p.src[p.pos:])
			switch {
			case r == utf8.RuneError && size == 1:
				r, highByte = rune(c), true // a raw byte that isn't UTF-8: a byte string
			case r >= utf8.RuneSelf:
				wide = true // a character beyond ASCII, as written: text
			}
			chars = append(chars, r)
			p.pos += size
			continue
		}
		p.pos++ // past \
		if p.pos >= len(p.src) {
			p.fail(`this string has no closing "`)
		}
		e := p.src[p.pos]
		switch {
		case cEscapeChars[e] != 0:
			chars = append(chars, cEscapeChars[e])
			p.pos++
		case '0' <= e && e <= '7': // octal: up to three digits
			n, end := 0, p.pos
			for end < len(p.src) && end < p.pos+3 && '0' <= p.src[end] && p.src[end] <= '7' {
				n = n*8 + int(p.src[end]-'0')
				end++
			}
			p.pos = end
			switch {
			case n > 0xff:
				wide = true // up to \777: beyond a byte
			case n > 0x7f:
				highByte = true
			}
			chars = append(chars, rune(n))
		case e == 'x':
			p.pos++
			if p.pos < len(p.src) && p.src[p.pos] == '{' {
				end := strings.IndexByte(p.src[p.pos:], '}')
				if end < 0 {
					p.fail(`\x{ has no closing }`)
				}
				n, err := strconv.ParseUint(strings.TrimSpace(p.src[p.pos+1:p.pos+end]), 16, 32)
				if err != nil || n > utf8.MaxRune {
					p.fail(`\x{...} needs a hex character code`)
				}
				chars = append(chars, rune(n))
				wide = true
				p.pos += end + 1
				continue
			}
			n, end := 0, p.pos // \xHH: up to two hex digits
			for end < len(p.src) && end < p.pos+2 && isHexDigit(p.src[end]) {
				d, _ := strconv.ParseUint(p.src[end:end+1], 16, 8)
				n = n*16 + int(d)
				end++
			}
			p.pos = end
			if n > 0x7f {
				highByte = true
			}
			chars = append(chars, rune(n))
		default: // \\, \", \$, \@ and any other character stand for themselves
			r, size := utf8.DecodeRuneInString(p.src[p.pos:])
			chars = append(chars, r)
			p.pos += size
		}
	}
	if highByte && !wide {
		b := make([]byte, len(chars))
		for i, r := range chars {
			b[i] = byte(r)
		}
		return &SV{v: b}
	}
	return &SV{v: string(chars)}
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// reference reads $VAR1, or a path from it such as $VAR1->{tags}[0], and
// returns a new reference to the array or hash already read at that place.
func (p *dumpParser) reference() *SV {
	start := p.pos
	if !strings.HasPrefix(p.src[p.pos:], "$VAR1") {
		p.pos++
		w := p.word()
		p.pos = start
		if strings.HasPrefix(w, "VAR") {
			p.fail("only $VAR1 can be read: a dump of one value")
		}
		p.fail("$%s can't be read: the only variable in a dump is $VAR1", w)
	}
	p.pos += len("$VAR1")
	if p.root == nil {
		p.pos = start
		p.fail("$VAR1 refers to the value being read, but nothing has been read yet")
	}
	cur := p.root
	for {
		if strings.HasPrefix(p.src[p.pos:], "->") {
			p.pos += 2
		}
		if p.pos >= len(p.src) || p.src[p.pos] != '{' && p.src[p.pos] != '[' {
			break
		}
		var next *SV
		if p.src[p.pos] == '{' {
			p.pos++
			key := p.key()
			p.expect("}")
			if h, ok := cur.v.(*HV); ok {
				next = h.m[key]
			}
		} else {
			p.pos++
			p.space()
			from := p.pos
			if p.pos < len(p.src) && p.src[p.pos] == '-' {
				p.pos++
			}
			for p.pos < len(p.src) && '0' <= p.src[p.pos] && p.src[p.pos] <= '9' {
				p.pos++
			}
			i, err := strconv.Atoi(p.src[from:p.pos])
			if err != nil {
				p.fail("expected an index in [...]")
			}
			p.expect("]")
			if a, ok := cur.v.(*AV); ok {
				if idx, inRange := a.index(i); inRange && idx < len(a.items) {
					next = a.items[idx]
				}
			}
		}
		if next == nil {
			p.fail("%s leads to nothing: a path from $VAR1 must lead to something already read", p.src[start:p.pos])
		}
		cur = next
	}
	if !isRef(cur.v) {
		p.fail("%s leads to a plain value: only an array or hash can appear twice", p.src[start:p.pos])
	}
	return &SV{v: cur.v}
}
