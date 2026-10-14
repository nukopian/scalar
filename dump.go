package scalar

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Dump returns s as Data::Dumper::Concise prints it: Data::Dumper with
// Terse, Indent = 1, Useqq, Quotekeys = 0, Sortkeys and Trailingcomma, so
// nested arrays and hashes are laid out one element to a line, hash keys
// are sorted and quoted only when they need to be, and strings are
// double-quoted, with escapes for anything that isn't printable ASCII.
//
//	fmt.Println(doc.Dump())   // print Dumper($doc)
//
// An array or hash that appears twice is printed in full the first time
// and as its path afterwards, as $VAR1->{"tags"}, so a reference that leads
// back into s is safe. Unlike Dumper, there's no newline at the end.
func (s *SV) Dump() string { return dump(s, true) }

// Dump returns a as Data::Dumper prints \@a; see SV.Dump.
func (a *AV) Dump() string { return dump(&SV{v: a}, true) }

// Dump returns h as Data::Dumper prints \%h; see SV.Dump.
func (h *HV) Dump() string { return dump(&SV{v: h}, true) }

// String returns a on one line, as Dump does but with Indent = 0, so
// fmt.Println(a) shows its elements, nested arrays and hashes included:
// [1,"a",[2]]. See SV.Dump.
func (a *AV) String() string { return dump(&SV{v: a}, false) }

// String returns h on one line, as Dump does but with Indent = 0, so
// fmt.Println(h) shows its keys and values: {a => 1,b => [2]}. See SV.Dump.
func (h *HV) String() string { return dump(&SV{v: h}, false) }

func dump(s *SV, indent bool) string {
	d := dumper{indent: indent, seen: map[any]string{}}
	d.sv(s, "$VAR1", 0)
	return d.b.String()
}

// dumper writes a structure as Data::Dumper does, remembering the path of
// each array and hash it has written, for when one appears again.
type dumper struct {
	b      strings.Builder
	indent bool
	seen   map[any]string
}

// bareNumber is what Data::Dumper writes without quotes: an integer of at
// most 9 digits.
var bareNumber = regexp.MustCompile(`^(?:0|-?[1-9]\d{0,8})$`)

// bareKey is a hash key that Data::Dumper with Quotekeys = 0 writes without
// quotes: a word, or a bare number.
var bareKey = regexp.MustCompile(`^(?:[A-Za-z_]\w*|0|-?[1-9]\d{0,8})$`)

func (d *dumper) sv(s *SV, path string, depth int) {
	switch v := s.val().(type) {
	case nil:
		d.b.WriteString("undef")
	case bool:
		if v {
			d.b.WriteString("!!1")
		} else {
			d.b.WriteString("!!0")
		}
	case *AV:
		d.av(v, path, depth)
	case *HV:
		d.hv(v, path, depth)
	default:
		str := s.Str()
		if bareNumber.MatchString(str) {
			d.b.WriteString(str)
		} else {
			d.b.WriteString(qquote(str, s.Kind() == Bytes))
		}
	}
}

// child is the path of an element of the container at path: $VAR1->[0] for
// the top, then $VAR1->[0]{"k"} below it, as Data::Dumper writes them.
func child(path, step string) string {
	if path == "$VAR1" {
		return path + "->" + step
	}
	return path + step
}

// open writes a container's opening bracket and checks whether it has been
// written already, in which case its path is written instead.
func (d *dumper) open(c any, path, bracket string) bool {
	if p, ok := d.seen[c]; ok {
		d.b.WriteString(p)
		return false
	}
	d.seen[c] = path
	d.b.WriteString(bracket)
	return true
}

// item starts the i'th element of a container at depth.
func (d *dumper) item(i, depth int) {
	if i > 0 {
		d.b.WriteByte(',')
	}
	if d.indent {
		d.b.WriteString("\n" + strings.Repeat("  ", depth+1))
	}
}

// close ends a container of n elements at depth, with a comma after the
// last when they're on lines of their own, as Trailingcomma gives.
func (d *dumper) close(n, depth int, bracket string) {
	if d.indent && n > 0 {
		d.b.WriteString(",\n" + strings.Repeat("  ", depth))
	}
	d.b.WriteString(bracket)
}

func (d *dumper) av(a *AV, path string, depth int) {
	if a == nil {
		d.b.WriteString("undef")
		return
	}
	if !d.open(a, path, "[") {
		return
	}
	for i, e := range a.items {
		d.item(i, depth)
		var p string // only an array or hash needs its path
		if isRef(e.val()) {
			p = child(path, "["+strconv.Itoa(i)+"]")
		}
		d.sv(e, p, depth+1) // a hole is undef
	}
	d.close(len(a.items), depth, "]")
}

func (d *dumper) hv(h *HV, path string, depth int) {
	if h == nil {
		d.b.WriteString("undef")
		return
	}
	if !d.open(h, path, "{") {
		return
	}
	keys := slices.Sorted(maps.Keys(h.m))
	for i, k := range keys {
		d.item(i, depth)
		key := k
		if !bareKey.MatchString(k) {
			key = qquote(k, false)
		}
		d.b.WriteString(key)
		d.b.WriteString(" => ")
		var p string // only an array or hash needs its path
		if isRef(h.m[k].val()) {
			p = child(path, "{"+key+"}")
		}
		d.sv(h.m[k], p, depth+1)
	}
	d.close(len(keys), depth, "}")
}

// cEscapes are the control characters qquote writes as C escapes.
var cEscapes = map[rune]string{'\a': `\a`, '\b': `\b`, '\t': `\t`, '\n': `\n`, '\f': `\f`, '\r': `\r`, '\x1b': `\e`}

// qquote double-quotes s as Data::Dumper's qquote does with Useqq: \, ", $
// and @ are escaped, control characters get their C escape or octal, and
// characters beyond ASCII are written as \x{e9} in text, or in octal, as
// \351, in a byte string.
func qquote(s string, isBytes bool) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); {
		c, size := rune(s[i]), 1
		raw := isBytes // a byte to write in octal, as in a byte string
		if !isBytes && c >= utf8.RuneSelf {
			c, size = utf8.DecodeRuneInString(s[i:])
			raw = c == utf8.RuneError && size == 1 // not UTF-8: a byte, as Perl would have it
			if raw {
				c = rune(s[i])
			}
		}
		i += size
		nextDigit := i < len(s) && '0' <= s[i] && s[i] <= '9'
		switch {
		case c == '\\' || c == '"' || c == '$' || c == '@':
			b.WriteByte('\\')
			b.WriteRune(c)
		case cEscapes[c] != "":
			b.WriteString(cEscapes[c])
		case c < 0x20 && !nextDigit:
			fmt.Fprintf(&b, `\%o`, c) // the shortest octal, unless a digit follows
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, `\%03o`, c)
		case c < 0x7f:
			b.WriteRune(c)
		case raw:
			fmt.Fprintf(&b, `\%o`, c)
		default:
			fmt.Fprintf(&b, `\x{%x}`, c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
