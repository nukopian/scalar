package scalar

import (
	"fmt"
	"iter"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Engine is a regex engine: Backtracking, the default, or Linear. Choose
// one for a pattern with CompileWith or MustCompileWith, or for every
// pattern with DefaultEngine.
type Engine int

const (
	// Backtracking is github.com/dlclark/regexp2, the engine scalar has
	// always used: near-Perl syntax, with lookaround, backreferences and
	// atomic groups. A match that runs longer than RegexTimeout panics.
	Backtracking Engine = iota

	// Linear is Go's regexp, which takes time in step with the text, so it
	// never needs a timeout. It never gives a different answer from
	// Backtracking, only a faster one: it takes only the patterns it can
	// run as Backtracking does, and hands a match to Backtracking when the
	// text could make them differ, such as text that isn't ASCII for a
	// pattern with \b. The reference lists what it takes and when it hands
	// over.
	Linear
)

// DefaultEngine is the engine for pattern strings given to M, Matches,
// Named, S and Split, and for Compile and MustCompile. Set it before first
// use, as with RegexTimeout. With Linear, a pattern Linear can't run uses
// Backtracking, and RegexWarn says so, once for that pattern.
var DefaultEngine = Backtracking

// RegexWarn is called when DefaultEngine is Linear and a pattern uses
// something Linear can't run, so it runs on Backtracking. It logs the
// message through the log package; replace it to send the warning
// elsewhere, or set it to nil to say nothing.
var RegexWarn = func(msg string) { log.Print(msg) }

// CompileWith is Compile, with engine in place of DefaultEngine. With
// Linear, a pattern Linear can't run is an error: it never falls back to
// Backtracking for the whole pattern, as DefaultEngine does. A *Regexp
// keeps its engine whatever DefaultEngine says, so a package can rely on
// Backtracking's lookaround while a program sets DefaultEngine to Linear.
func CompileWith(engine Engine, pattern string, flags ...string) (*Regexp, error) {
	return compileWith("CompileWith", engine, pattern, parseFlags("CompileWith", "imsx", flags))
}

// MustCompileWith is CompileWith, panicking if the pattern is invalid or,
// with Linear, uses something Linear can't run.
func MustCompileWith(engine Engine, pattern string, flags ...string) *Regexp {
	re, err := compileWith("MustCompileWith", engine, pattern, parseFlags("MustCompileWith", "imsx", flags))
	if err != nil {
		panic(err.Error())
	}
	return re
}

func compileWith(op string, engine Engine, pattern string, f flagSet) (*Regexp, error) {
	if engine != Backtracking && engine != Linear {
		panic(fmt.Sprintf("scalar: %s given an unknown engine, %d", op, engine))
	}
	rx, err := compileErr(pattern, f)
	if err != nil {
		return nil, fmt.Errorf("scalar: %w", err)
	}
	re := &Regexp{rx: rx}
	if engine == Linear {
		c := linearFor(pattern, f)
		if c.reason != "" {
			return nil, fmt.Errorf("scalar: Linear can't run regex %q: it uses %s", pattern, c.reason)
		}
		re.lin = c.l
	}
	return re, nil
}

// defaultLinear is the Linear form of a pattern compiled under
// DefaultEngine, or nil, warning if Linear can't run it.
func defaultLinear(pattern string, f flagSet) *linear {
	if DefaultEngine != Linear {
		return nil
	}
	return linearFor(pattern, f).warned(pattern)
}

// A linearCompiled is a pattern compiled for Linear, or why it couldn't be.
type linearCompiled struct {
	l      *linear
	reason string
	warn   sync.Once
}

func (c *linearCompiled) warned(pattern string) *linear {
	if c.reason != "" {
		c.warn.Do(func() {
			if w := RegexWarn; w != nil {
				w(fmt.Sprintf("scalar: regex %q uses %s, which Linear can't run, so it runs on Backtracking", pattern, c.reason))
			}
		})
	}
	return c.l
}

type linearKey struct{ pattern, flags string }

var linearCache = boundedCache[linearKey, *linearCompiled]{limit: cacheLimit}

func linearFor(pattern string, f flagSet) *linearCompiled {
	var flags []byte
	for _, c := range "imsx" {
		if f.has(c) {
			flags = append(flags, byte(c))
		}
	}
	key := linearKey{pattern, string(flags)}
	if c, ok := linearCache.load(key); ok {
		return c
	}
	l, reason := compileLinear(pattern, f)
	return linearCache.store(key, &linearCompiled{l: l, reason: reason})
}

// linear is a pattern compiled for Go's regexp, with what the text must be
// for it to give Backtracking's answer.
type linear struct {
	rx    *regexp.Regexp
	order []int    // order[i] is the Go group that is regexp2's group i
	names []string // names[i] is the name of regexp2's group i, or ""
	nl    bool     // $ without m, which differs on text ending in a newline
	ascii bool     // \b, \B, or something not ASCII under i: text must be ASCII
	fold  bool     // i, which differs on the few runes in foldTrap
	empty bool     // the pattern can match nothing, where g differs
}

// use reports whether a match of l in str gives Backtracking's answer. g is
// for M and S with "g", and Matches, which go on after an empty match.
func (l *linear) use(str string, g bool) bool {
	if l == nil || g && l.empty || l.nl && strings.HasSuffix(str, "\n") {
		return false
	}
	for i := 0; i < len(str); i++ {
		if str[i] >= utf8.RuneSelf {
			return l.useUnicode(str[i:])
		}
	}
	return true
}

func (l *linear) useUnicode(rest string) bool {
	if l.ascii || !utf8.ValidString(rest) {
		return false
	}
	if l.fold {
		for _, r := range rest {
			if foldTrap(r) {
				return false
			}
		}
	}
	return true
}

// foldTrap reports whether r, beyond ASCII, matches an ASCII pattern under
// i differently in the two engines: regexp2 lowercases it, as İ becomes i,
// and Go's regexp folds it, as ſ matches s.
func foldTrap(r rune) bool {
	if r < utf8.RuneSelf {
		return false
	}
	if unicode.ToLower(r) < utf8.RuneSelf {
		return true
	}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		if f < utf8.RuneSelf {
			return true
		}
	}
	return false
}

// first is the first match of l in str, laid out as regexp's
// FindStringSubmatchIndex lays it out, in regexp2's group order, or nil.
func (l *linear) first(str string) []int {
	return l.reorder(l.rx.FindStringSubmatchIndex(str))
}

func (l *linear) all(str string) iter.Seq[[]int] {
	return func(yield func([]int) bool) {
		for _, idx := range l.rx.FindAllStringSubmatchIndex(str, -1) {
			if !yield(l.reorder(idx)) {
				return
			}
		}
	}
}

func (l *linear) reorder(idx []int) []int {
	if idx == nil || l.order == nil {
		return idx
	}
	out := make([]int, len(idx))
	for i, g := range l.order {
		out[2*i], out[2*i+1] = idx[2*g], idx[2*g+1]
	}
	return out
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

func (l *linear) m(str string, isBytes, g bool) *AV {
	if !g {
		if idx := l.first(str); idx != nil {
			return indexAV(str, idx, isBytes)
		}
		return nil
	}
	var out *AV
	for idx := range l.all(str) {
		if out == nil {
			out = &AV{}
		}
		out.items = append(out.items, &SV{v: indexAV(str, idx, isBytes)})
	}
	return out
}

func (l *linear) named(str string, isBytes bool) *HV {
	idx := l.first(str)
	if idx == nil {
		return nil
	}
	h := newHV(0)
	for i, name := range l.names {
		if name != "" {
			h.m[name] = indexSV(str, idx, i, isBytes)
		}
	}
	return h
}

func (l *linear) s(str string, isBytes bool, repl any, limit int) (string, int) {
	var replace func(idx []int) string
	switch r := repl.(type) {
	case string:
		replace = func(idx []int) string { return expandIndex(r, str, idx, l.names) }
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
	matches := l.all(str)
	if limit == 1 {
		matches = func(yield func([]int) bool) {
			if idx := l.first(str); idx != nil {
				yield(idx)
			}
		}
	}
	for idx := range matches {
		out.WriteString(str[last:idx[0]])
		out.WriteString(replace(idx))
		last = idx[1]
		n++
	}
	if n == 0 {
		return str, 0
	}
	out.WriteString(str[last:])
	return out.String(), n
}

// expandIndex is expand for a match laid out as positions, with the same
// rules, which TestSubstExpand checks by running S's expansion tests on
// both engines.
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
		for k, n := range names {
			if n == name && name != "" {
				return group(k), true
			}
		}
		return "", false
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

func (l *linear) split(str string, isBytes bool, lim int) *AV {
	out := &AV{}
	pos, splits := 0, 0
	for idx := range l.all(str) {
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

// compileLinear translates a pattern in regexp2's syntax into Go's, taking
// only what Go's regexp runs as regexp2 does, or returns what it can't
// take, as a phrase for a message: "lookbehind".
func compileLinear(pattern string, f flagSet) (l *linear, reason string) {
	if f.has('x') {
		return nil, "the x flag"
	}
	p := &linearParser{in: []rune(pattern), l: &linear{}, fold: f.has('i'), multi: f.has('m')}
	defer func() {
		if r := recover(); r != nil {
			why, ok := r.(cantRun)
			if !ok {
				panic(r)
			}
			l, reason = nil, string(why)
		}
	}()
	root := p.alt()
	if p.pos < len(p.in) {
		p.cant("an unmatched )")
	}
	var prefix string
	for _, c := range "ims" {
		if f.has(c) {
			prefix += string(c)
		}
	}
	if prefix != "" {
		prefix = "(?" + prefix + ")"
	}
	rx, err := regexp.Compile(prefix + p.out.String())
	if err != nil {
		p.cant("syntax Go's regexp rejects (" + err.Error() + ")")
	}
	p.l.rx, p.l.empty = rx, root.nullable

	// regexp2 numbers named groups after unnamed ones.
	order, names := []int{0}, []string{""}
	for i, name := range p.groups {
		if name == "" {
			order, names = append(order, i+1), append(names, "")
		}
	}
	for i, name := range p.groups {
		if name != "" {
			order, names = append(order, i+1), append(names, name)
		}
	}
	p.l.names = names
	for i, g := range order {
		if i != g {
			p.l.order = order
			break
		}
	}
	return p.l, ""
}

// cantRun is what Linear can't run, raised as a panic within the parser.
type cantRun string

type linearParser struct {
	in     []rune
	pos    int
	out    strings.Builder
	groups []string // each capture group in order: its name, or ""
	l      *linear
	fold   bool // under i
	multi  bool // under m
}

// node is what the parser knows of a piece of the pattern.
type node struct {
	nullable bool // can match nothing
	captures bool // holds a capture group
	anchor   bool // matches a position, which can't be repeated
}

func (p *linearParser) cant(what string) { panic(cantRun(what)) }

func (p *linearParser) peek(c rune) bool { return p.pos < len(p.in) && p.in[p.pos] == c }

func (p *linearParser) next() rune {
	if p.pos == len(p.in) {
		p.cant("an unfinished construct")
	}
	c := p.in[p.pos]
	p.pos++
	return c
}

func (p *linearParser) alt() node {
	n := p.seq()
	for p.peek('|') {
		p.pos++
		p.out.WriteByte('|')
		m := p.seq()
		n = node{nullable: n.nullable || m.nullable, captures: n.captures || m.captures}
	}
	return n
}

func (p *linearParser) seq() node {
	n := node{nullable: true}
	for p.pos < len(p.in) && !p.peek('|') && !p.peek(')') {
		a := p.repeat()
		n.nullable = n.nullable && a.nullable
		n.captures = n.captures || a.captures
	}
	return n
}

// repeat is an atom and any quantifier after it.
func (p *linearParser) repeat() node {
	a := p.atom()
	if p.pos == len(p.in) {
		return a
	}
	start, body := p.pos, a.nullable
	switch p.in[p.pos] {
	case '*', '?':
		p.pos++
		a.nullable = true
	case '+':
		p.pos++
	case '{':
		lo, ok := p.count()
		if !ok {
			p.cant("a { that isn't a repeat count")
		}
		if lo == 0 {
			a.nullable = true
		}
	default:
		return a
	}
	if a.anchor {
		p.cant("a quantifier on an anchor")
	}
	if p.peek('?') {
		p.pos++
	}
	if p.pos < len(p.in) && strings.ContainsRune("*+?{", p.in[p.pos]) {
		p.cant("a possessive or nested quantifier")
	}
	if a.captures && body {
		p.cant("a capture group inside a repeat that can match nothing")
	}
	p.out.WriteString(string(p.in[start:p.pos]))
	return a
}

// count reads {n}, {n,} or {n,m}, and returns n.
func (p *linearParser) count() (int, bool) {
	end := p.pos + 1
	for end < len(p.in) && p.in[end] != '}' {
		end++
	}
	if end == len(p.in) {
		return 0, false
	}
	lo, hi, comma := strings.Cut(string(p.in[p.pos+1:end]), ",")
	n, err := strconv.Atoi(lo)
	if err != nil || lo[0] == '+' || lo[0] == '-' {
		return 0, false
	}
	if comma && hi != "" {
		if _, err := strconv.Atoi(hi); err != nil || hi[0] == '+' || hi[0] == '-' {
			return 0, false
		}
	}
	p.pos = end + 1
	return n, true
}

func (p *linearParser) atom() node {
	c := p.next()
	switch c {
	case '(':
		return p.group()
	case '[':
		p.class()
		return node{}
	case '.':
		p.out.WriteByte('.')
		return node{}
	case '^':
		p.out.WriteByte('^')
		return node{nullable: true, anchor: true}
	case '$':
		if !p.multi {
			p.l.nl = true
		}
		p.out.WriteByte('$')
		return node{nullable: true, anchor: true}
	case '\\':
		return p.escape()
	case '{', '*', '+', '?':
		p.cant(fmt.Sprintf("a %c Linear can't place", c))
	}
	p.char(c)
	p.out.WriteString(regexp.QuoteMeta(string(c)))
	return node{}
}

// char notes a character the pattern matches, for i.
func (p *linearParser) char(c rune) { p.chars(c, c) }

// chars notes the characters from lo to hi, which the pattern matches. Under
// i, Linear takes none that fold differently in the two engines, as ſ
// matches s in Go's regexp alone, and if any is beyond ASCII, it needs text
// that's ASCII.
func (p *linearParser) chars(lo, hi rune) {
	if !p.fold {
		return
	}
	p.l.fold = true
	if hi >= utf8.RuneSelf {
		p.l.ascii = true
	}
	for _, r := range foldTraps() {
		if lo <= r && r <= hi {
			p.cant(fmt.Sprintf("%q with the i flag, which the engines fold differently", r))
		}
	}
}

// foldTraps is every rune for which foldTrap is true.
var foldTraps = sync.OnceValue(func() []rune {
	var traps []rune
	for r := rune(utf8.RuneSelf); r <= unicode.MaxRune; r++ {
		if foldTrap(r) {
			traps = append(traps, r)
		}
	}
	return traps
})

func (p *linearParser) group() node {
	if !p.peek('?') {
		p.groups = append(p.groups, "")
		p.out.WriteByte('(')
		n := p.inner()
		n.captures = true
		return n
	}
	p.pos++
	switch c := p.next(); {
	case c == ':':
		p.out.WriteString("(?:")
		return p.inner()
	case c == '<' && (p.peek('=') || p.peek('!')):
		p.cant("lookbehind")
	case c == '<':
		name := p.name()
		p.groups = append(p.groups, name)
		p.out.WriteString("(?P<" + name + ">")
		n := p.inner()
		n.captures = true
		return n
	case c == '=' || c == '!':
		p.cant("lookahead")
	case c == '>':
		p.cant("an atomic group")
	case c == '#':
		p.cant("a comment group")
	case c == '(':
		p.cant("a conditional")
	case c == '\'':
		p.cant("a group named with quotes")
	case c == 'P':
		p.cant("(?P<name>), which Backtracking doesn't read")
	default:
		p.pos--
		return p.flags()
	}
	return node{}
}

// inner is a group's contents and its ), with the group's own flags.
func (p *linearParser) inner() node {
	fold, multi := p.fold, p.multi
	n := p.alt()
	if !p.peek(')') {
		p.cant("an unclosed group")
	}
	p.pos++
	p.out.WriteByte(')')
	p.fold, p.multi = fold, multi
	return node{nullable: n.nullable, captures: n.captures}
}

func (p *linearParser) name() string {
	start := p.pos
	for p.pos < len(p.in) && p.in[p.pos] != '>' {
		c := p.in[p.pos]
		if !(c == '_' || c < utf8.RuneSelf && (unicode.IsLetter(c) || p.pos > start && unicode.IsDigit(c))) {
			p.cant("a group name Linear can't take")
		}
		p.pos++
	}
	name := string(p.in[start:p.pos])
	if name == "" || p.pos == len(p.in) {
		p.cant("a group name Linear can't take")
	}
	p.pos++
	for _, g := range p.groups {
		if g == name {
			p.cant("a group name used twice")
		}
	}
	return name
}

// flags reads (?i), (?-m), (?s:...) and the like.
func (p *linearParser) flags() node {
	start, on := p.pos, true
	fold, multi := p.fold, p.multi
	for {
		c := p.next()
		switch c {
		case 'i':
			fold = on
		case 'm':
			multi = on
		case 's':
		case '-':
			on = false
			continue
		case ')':
			p.out.WriteString("(?" + string(p.in[start:p.pos]))
			p.fold, p.multi = fold, multi
			return node{nullable: true, anchor: true}
		case ':':
			p.out.WriteString("(?" + string(p.in[start:p.pos]))
			outerFold, outerMulti := p.fold, p.multi
			p.fold, p.multi = fold, multi
			n := p.inner()
			p.fold, p.multi = outerFold, outerMulti
			return n
		default:
			p.cant(fmt.Sprintf("the inline flag %q", c))
		}
	}
}

func (p *linearParser) escape() node {
	c := p.next()
	switch c {
	case 'A', 'z':
		p.out.WriteString(`\` + string(c))
		return node{nullable: true, anchor: true}
	case 'b', 'B':
		p.l.ascii = true
		p.out.WriteString(`\` + string(c))
		return node{nullable: true, anchor: true}
	case 'w', 'W', 's', 'S', 'd', 'D':
		p.out.WriteString(classEscape(c, false))
		return node{}
	case 'p', 'P':
		p.out.WriteString(p.property(c))
		return node{}
	}
	text, r := p.charEscape(c)
	p.char(r)
	p.out.WriteString(text)
	return node{}
}

// charEscape is an escape for one character, other than a class or anchor:
// its text for Go's regexp, and the character.
func (p *linearParser) charEscape(c rune) (string, rune) {
	switch c {
	case 't', 'n', 'r', 'f', 'v', 'a':
		return `\` + string(c), controlEscapes[c]
	case 'x':
		var hex string
		if p.peek('{') {
			end := p.pos
			for end < len(p.in) && p.in[end] != '}' {
				end++
			}
			if end == len(p.in) {
				p.cant(`an unfinished \x{`)
			}
			hex, p.pos = string(p.in[p.pos+1:end]), end+1
		} else if p.pos+2 <= len(p.in) {
			hex, p.pos = string(p.in[p.pos:p.pos+2]), p.pos+2
		}
		n, err := strconv.ParseUint(hex, 16, 32)
		if err != nil || hex == "" || n > unicode.MaxRune {
			p.cant(`a \x escape Linear can't read`)
		}
		return fmt.Sprintf(`\x{%x}`, n), rune(n)
	case 'Z', 'G':
		p.cant(`\` + string(c))
	case 'k':
		p.cant("a named backreference")
	case 'Q', 'E':
		p.cant(`\Q…\E`)
	}
	switch {
	case c >= '0' && c <= '9':
		p.cant("a backreference or octal escape")
	case c < utf8.RuneSelf && !unicode.IsLetter(c):
		return `\` + string(c), c
	}
	p.cant(fmt.Sprintf(`the escape \%c`, c))
	return "", 0
}

var controlEscapes = map[rune]rune{'t': '\t', 'n': '\n', 'r': '\r', 'f': '\f', 'v': '\v', 'a': '\a'}

// property reads \pL or \p{Name}, after its p or P.
func (p *linearParser) property(c rune) string {
	if p.fold {
		p.cant(`\p or \P with the i flag`)
	}
	var name string
	if p.peek('{') {
		end := p.pos
		for end < len(p.in) && p.in[end] != '}' {
			end++
		}
		if end == len(p.in) {
			p.cant(`an unfinished \p{`)
		}
		name, p.pos = string(p.in[p.pos+1:end]), end+1
	} else {
		name = string(p.next())
	}
	if unicode.Categories[name] == nil && unicode.Scripts[name] == nil {
		p.cant(fmt.Sprintf(`\%c{%s}, which Linear doesn't know`, c, name))
	}
	return `\` + string(c) + "{" + name + "}"
}

// class reads a character class, after its [.
func (p *linearParser) class() {
	p.out.WriteByte('[')
	if p.peek('^') {
		p.pos++
		p.out.WriteByte('^')
	}
	if p.peek(']') {
		p.cant("a class that starts with ]")
	}
	for first := true; ; first = false {
		c := p.next()
		switch {
		case c == ']':
			p.out.WriteByte(']')
			return
		case c == '[':
			p.cant("a [ in a class, as in a POSIX class or class subtraction")
		case c == '-' && p.peek('['):
			p.cant("class subtraction")
		case c == '-':
			if !first && !p.peek(']') {
				p.cant("a - Linear can't place in a class")
			}
			p.out.WriteString(`\-`)
			continue
		}
		text, lo, single := p.classItem(c)
		if single && p.peek('-') && p.pos+1 < len(p.in) && p.in[p.pos+1] != ']' {
			p.pos++
			hiText, hi, single := p.classItem(p.next())
			if !single {
				p.cant("a range ending in a class")
			}
			text += "-" + hiText
			p.chars(lo, hi)
		} else if single {
			p.char(lo)
		}
		p.out.WriteString(text)
	}
}

// classItem is one item in a class, starting with c: its text for Go's
// regexp, and, if it's a single character, which can start a range, the
// character.
func (p *linearParser) classItem(c rune) (string, rune, bool) {
	if c != '\\' {
		if c < utf8.RuneSelf && !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return `\` + string(c), c, true
		}
		return string(c), c, true
	}
	c = p.next()
	switch c {
	case 'w', 'W', 's', 'S', 'd', 'D':
		return classEscape(c, true), 0, false
	case 'p', 'P':
		return p.property(c), 0, false
	case 'b':
		p.cant(`\b in a class`)
	}
	text, r := p.charEscape(c)
	return text, r, true
}

// classEscape is \w, \s or \d, or their negations, as regexp2 defines them:
// for Unicode, with \w taking in the zero-width joiner and non-joiner. In a
// class, it's the class's contents.
func classEscape(c rune, inClass bool) string {
	word := `\p{L}\p{Mn}\p{Nd}\p{Pc}\x{200c}\x{200d}`
	switch c {
	case 'd':
		return `\p{Nd}`
	case 'D':
		return `\P{Nd}`
	case 'w':
		if inClass {
			return word
		}
		return "[" + word + "]"
	case 'W':
		if inClass {
			return notWord()
		}
		return "[^" + word + "]"
	case 's':
		if inClass {
			return space()
		}
		return "[" + space() + "]"
	default: // 'S'
		if inClass {
			return notSpace()
		}
		return "[^" + space() + "]"
	}
}

var (
	space    = sync.OnceValue(func() string { return runeRanges(unicode.IsSpace) })
	notSpace = sync.OnceValue(func() string { return runeRanges(func(r rune) bool { return !unicode.IsSpace(r) }) })
	notWord  = sync.OnceValue(func() string { return runeRanges(func(r rune) bool { return !regexp2IsWord(r) }) })
)

// regexp2IsWord is regexp2's \w.
func regexp2IsWord(r rune) bool {
	return unicode.In(r, unicode.L, unicode.Mn, unicode.Nd, unicode.Pc) || r == '‌' || r == '‍'
}

// runeRanges is the runes for which in is true, as the contents of a class.
func runeRanges(in func(rune) bool) string {
	var b strings.Builder
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !in(r) {
			continue
		}
		lo := r
		for r+1 <= unicode.MaxRune && in(r+1) {
			r++
		}
		fmt.Fprintf(&b, `\x{%x}`, lo)
		if r > lo {
			fmt.Fprintf(&b, `-\x{%x}`, r)
		}
	}
	return b.String()
}
