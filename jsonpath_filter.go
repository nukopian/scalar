package scalar

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Filter selectors, [?...], as RFC 9535 section 2.3.5 defines them: a
// logical expression of comparisons, existence tests and function calls,
// tested against each child of the node the selector is applied to.

// logical is a logical expression: true or false for a child (cur), with
// root as the query's $.
type logical interface {
	test(cur, root *SV) bool
}

type orExpr []logical
type andExpr []logical
type notExpr struct{ e logical }

// existsExpr is a query used as a test: true if it selects anything.
type existsExpr struct{ q filterQuery }

// compareExpr is a comparison of two comparables.
type compareExpr struct {
	left, right comparand
	op          string
}

// funcTest is a function returning a logical value, used as a test.
type funcTest struct{ call *funcCall }

func (e orExpr) test(cur, root *SV) bool {
	for _, x := range e {
		if x.test(cur, root) {
			return true
		}
	}
	return false
}

func (e andExpr) test(cur, root *SV) bool {
	for _, x := range e {
		if !x.test(cur, root) {
			return false
		}
	}
	return true
}

func (e notExpr) test(cur, root *SV) bool    { return !e.e.test(cur, root) }
func (e existsExpr) test(cur, root *SV) bool { return e.q.exists(cur, root) }
func (e funcTest) test(cur, root *SV) bool   { return e.call.logical(cur, root) }

func (e compareExpr) test(cur, root *SV) bool {
	a, b := e.left.value(cur, root), e.right.value(cur, root)
	switch e.op {
	case "==":
		return a.equal(b)
	case "!=":
		return !a.equal(b)
	case "<":
		return a.less(b)
	case "<=":
		return a.less(b) || a.equal(b)
	case ">":
		return b.less(a)
	case ">=":
		return b.less(a) || a.equal(b)
	}
	return false
}

// filterQuery is a query inside a filter: relative to the child (@) or
// absolute ($).
type filterQuery struct {
	relative bool
	segs     []segment
	single   bool // singular, as worked out when parsed
}

func (q filterQuery) nodes(cur, root *SV) []*SV {
	from := root
	if q.relative {
		from = cur
	}
	return query(from, root, q.segs)
}

// exists reports whether q selects anything: for a singular query, by
// following it without allocating.
func (q filterQuery) exists(cur, root *SV) bool {
	if q.single {
		_, ok := q.one(cur, root)
		return ok
	}
	return len(q.nodes(cur, root)) > 0
}

// one follows a singular query, step by step, and returns the node it
// selects, if any.
func (q filterQuery) one(cur, root *SV) (*SV, bool) {
	n := root
	if q.relative {
		n = cur
	}
	for _, seg := range q.segs {
		sel := seg.selectors[0]
		switch v := n.val().(type) {
		case *HV:
			c, ok := v.m[sel.name]
			if sel.kind != selName || !ok {
				return nil, false
			}
			n = c
		case *AV:
			i, ok := v.index(sel.index)
			if sel.kind != selIndex || !ok || i >= len(v.items) {
				return nil, false
			}
			n = hole(v.items[i])
		default:
			return nil, false
		}
	}
	return n, true
}

// singular reports whether q can select at most one node: only names and
// indexes, one at a time, never descendants.
func (q filterQuery) singular() bool {
	for _, seg := range q.segs {
		if seg.descendant || len(seg.selectors) != 1 {
			return false
		}
		if k := seg.selectors[0].kind; k != selName && k != selIndex {
			return false
		}
	}
	return true
}

// value is a value in a comparison: a JSON value, or Nothing, as a
// singular query that selects nothing is.
type value struct {
	sv      *SV
	nothing bool
}

var nothing = value{nothing: true}

// comparand is what a comparison compares: a literal, a singular query, or
// a function returning a value.
type comparand interface {
	value(cur, root *SV) value
}

type literalValue struct{ v value }

func (l literalValue) value(_, _ *SV) value { return l.v }

type singularValue struct{ q filterQuery }

func (s singularValue) value(cur, root *SV) value {
	if n, ok := s.q.one(cur, root); ok {
		return value{sv: n}
	}
	return nothing
}

// jsonKind is a value's JSON type, for comparisons.
type jsonKind int

const (
	jNull jsonKind = iota
	jBool
	jNumber
	jString
	jArray
	jObject
)

func kindOf(s *SV) jsonKind {
	switch s.val().(type) {
	case nil:
		return jNull
	case bool:
		return jBool
	case int64, float64:
		return jNumber
	case string, []byte:
		return jString
	case *AV:
		return jArray
	case *HV:
		return jObject
	}
	return jNull
}

// textOf is a string's text: a byte string's bytes as Latin-1 characters,
// as Eq and Length have them.
func textOf(s *SV) string {
	t, _ := s.text()
	return t
}

// equal is ==, as RFC 9535 section 2.3.5.2.2 defines it: Nothing equals
// only Nothing; values of different JSON types are unequal; numbers
// compare as numbers (1 == 1.0, and exactly when both are integers),
// strings by their characters (a byte string's are Latin-1, as Eq has
// them), and arrays and objects element by element, safely when they
// loop back on themselves, as Equal does.
func (a value) equal(b value) bool {
	if a.nothing || b.nothing {
		return a.nothing && b.nothing
	}
	return jsonEqual(a.sv, b.sv, map[[2]any]bool{})
}

// jsonEqual is == for two values. seen holds the pairs of arrays or hashes
// being compared further up, which count as equal if they come round again.
func jsonEqual(a, b *SV, seen map[[2]any]bool) bool {
	ka, kb := kindOf(a), kindOf(b)
	if ka != kb {
		return false
	}
	switch ka {
	case jNull:
		return true
	case jBool:
		return a.val() == b.val()
	case jNumber:
		c, ok := numCompare(a, b)
		return ok && c == 0
	case jString:
		return textOf(a) == textOf(b)
	case jArray:
		x, y := a.val().(*AV), b.val().(*AV)
		if x == y || seen[[2]any{x, y}] {
			return true
		}
		if len(x.items) != len(y.items) {
			return false
		}
		seen[[2]any{x, y}] = true
		for i := range x.items {
			xe, _ := element(x, i)
			ye, _ := element(y, i)
			if !jsonEqual(xe, ye, seen) {
				return false
			}
		}
		return true
	case jObject:
		x, y := a.val().(*HV), b.val().(*HV)
		if x == y || seen[[2]any{x, y}] {
			return true
		}
		if len(x.m) != len(y.m) {
			return false
		}
		seen[[2]any{x, y}] = true
		for k, xv := range x.m {
			yv, ok := y.m[k]
			if !ok || !jsonEqual(xv, yv, seen) {
				return false
			}
		}
		return true
	}
	return false
}

// less is <: true only for two numbers, or two strings, by their
// characters' code points.
func (a value) less(b value) bool {
	if a.nothing || b.nothing {
		return false
	}
	ka, kb := kindOf(a.sv), kindOf(b.sv)
	switch {
	case ka == jNumber && kb == jNumber:
		c, ok := numCompare(a.sv, b.sv)
		return ok && c < 0
	case ka == jString && kb == jString:
		return textOf(a.sv) < textOf(b.sv) // UTF-8's byte order is code point order
	}
	return false
}

// Function extensions (RFC 9535 section 2.4), each with its parameters'
// and result's types, checked when the query is parsed.

type fnType int

const (
	valueType   fnType = iota // a JSON value, or Nothing
	logicalType               // true or false
	nodesType                 // a list of nodes
)

type funcDef struct {
	params []fnType
	result fnType
}

var funcs = map[string]funcDef{
	"length": {[]fnType{valueType}, valueType},
	"count":  {[]fnType{nodesType}, valueType},
	"match":  {[]fnType{valueType, valueType}, logicalType},
	"search": {[]fnType{valueType, valueType}, logicalType},
	"value":  {[]fnType{nodesType}, valueType},
}

// funcArg is an argument: a comparand (for a value) or a query (for
// nodes).
type funcArg struct {
	val comparand
	q   *filterQuery
}

type funcCall struct {
	name string
	args []funcArg
}

// value returns a value-typed function's result.
func (f *funcCall) value(cur, root *SV) value {
	switch f.name {
	case "length":
		v := f.args[0].val.value(cur, root)
		if v.nothing {
			return nothing
		}
		switch x := v.sv.val().(type) {
		case string, []byte:
			return number(v.sv.Length()) // characters, a byte string's being Latin-1
		case *AV:
			return number(len(x.items))
		case *HV:
			return number(len(x.m))
		}
		return nothing
	case "count":
		return number(len(f.args[0].q.nodes(cur, root)))
	case "value":
		if ns := f.args[0].q.nodes(cur, root); len(ns) == 1 {
			return value{sv: ns[0]}
		}
		return nothing
	}
	return nothing
}

// logical returns a logical function's result: match and search.
func (f *funcCall) logical(cur, root *SV) bool {
	s, re := f.args[0].val.value(cur, root), f.args[1].val.value(cur, root)
	if s.nothing || re.nothing || kindOf(s.sv) != jString || kindOf(re.sv) != jString {
		return false
	}
	r, ok := iRegexp(textOf(re.sv), f.name == "match")
	return ok && r.MatchString(textOf(s.sv))
}

func number(n int) value { return value{sv: &SV{v: int64(n)}} }

// funcValue is a value-typed function call used as a comparand.
type funcValue struct{ call *funcCall }

func (f funcValue) value(cur, root *SV) value { return f.call.value(cur, root) }

// I-Regexp (RFC 9485), as match and search take it, translated to Go's
// regexp as its section 5.4 says: "." is any character but a line feed or
// carriage return, and match's pattern is wrapped in \A(?: and )\z. The
// only escapes are of the metacharacters, \n, \r, \t, and the Unicode
// classes \p{..} and \P{..}. "^" and "$" are left as they are, so they
// anchor: the RFC's grammar makes them ordinary characters, but its own
// mapping to RE2 leaves them alone, and the compliance suite expects them
// to anchor. match matches the whole string, search any part of it. A
// pattern that isn't I-Regexp gives false, as the RFC says, not an error.

// iRegexps are the patterns translated: "m" or "s" (for match or search)
// and the pattern → the regexp, or nil if it isn't I-Regexp. It's bounded,
// as patterns may come from the data ($[?match(@.a, @.re)]).
var iRegexps = boundedCache[string, *regexp.Regexp]{limit: cacheLimit}

func iRegexp(pattern string, whole bool) (*regexp.Regexp, bool) {
	key := "s" + pattern
	if whole {
		key = "m" + pattern
	}
	if re, ok := iRegexps.load(key); ok {
		return re, re != nil
	}
	var re *regexp.Regexp
	if goRE, ok := translateIRegexp(pattern); ok {
		if whole {
			goRE = `\A(?:` + goRE + `)\z`
		}
		re, _ = regexp.Compile(goRE)
	}
	re = iRegexps.store(key, re)
	return re, re != nil
}

// translateIRegexp returns pattern as Go regexp syntax, and false if it
// isn't I-Regexp. It parses pattern by RFC 9485's grammar, so it takes no
// more than I-Regexp does, though Go's regexp would: no (?...) groups or
// flags, no \d or \w, no lazy quantifiers, braces only as {n}, {n,} or
// {n,m}, \p{..} only for Unicode categories.
func translateIRegexp(pattern string) (string, bool) {
	x := &iRegexpParser{src: pattern}
	if !x.alternatives() || x.i != len(x.src) {
		return "", false
	}
	return x.out.String(), true
}

type iRegexpParser struct {
	src string
	i   int
	out strings.Builder
}

func (x *iRegexpParser) peek() rune {
	if x.i >= len(x.src) {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(x.src[x.i:])
	return r
}

func (x *iRegexpParser) next() rune {
	r, size := utf8.DecodeRuneInString(x.src[x.i:])
	x.i += size
	return r
}

// alternatives: branch *( "|" branch ).
func (x *iRegexpParser) alternatives() bool {
	for {
		if !x.branch() {
			return false
		}
		if x.peek() != '|' {
			return true
		}
		x.next()
		x.out.WriteByte('|')
	}
}

// branch: *piece, each an atom and perhaps one quantifier.
func (x *iRegexpParser) branch() bool {
	for {
		switch x.peek() {
		case -1, '|', ')':
			return true
		}
		if !x.atom() || !x.quantifier() {
			return false
		}
	}
}

// atom: a normal character, ".", an escape, a class, or a group.
func (x *iRegexpParser) atom() bool {
	switch r := x.peek(); r {
	case utf8.RuneError:
		return false
	case '.':
		x.next()
		x.out.WriteString(`[^\n\r]`)
	case '\\':
		return x.escape(false)
	case '[':
		return x.class()
	case '(':
		x.next()
		if x.peek() == '?' {
			return false // (?:...), (?i) and the rest aren't I-Regexp
		}
		x.out.WriteByte('(')
		if !x.alternatives() || x.peek() != ')' {
			return false
		}
		x.next()
		x.out.WriteByte(')')
	case '?', '*', '+', '{', '}', ')', '|', ']':
		return false // a metacharacter where an atom must be
	default:
		x.next()
		x.out.WriteRune(r) // ^ and $ too: see iRegexp
	}
	return true
}

// quantifier: perhaps *, +, ? or {n}, {n,} or {n,m}; never two.
func (x *iRegexpParser) quantifier() bool {
	switch x.peek() {
	case '*', '+', '?':
		x.out.WriteRune(x.next())
	case '{':
		start := x.i
		x.next()
		if !x.digits() {
			return false
		}
		if x.peek() == ',' {
			x.next()
			x.digits()
		}
		if x.peek() != '}' {
			return false
		}
		x.next()
		x.out.WriteString(x.src[start:x.i])
	default:
		return true
	}
	switch x.peek() {
	case '*', '+', '?', '{':
		return false // a lazy quantifier, or two in a row
	}
	return true
}

func (x *iRegexpParser) digits() bool {
	start := x.i
	for x.i < len(x.src) && '0' <= x.src[x.i] && x.src[x.i] <= '9' {
		x.i++
	}
	return x.i > start
}

// escape: \ and one of ( ) * + - . ? [ \ ] ^ { | } n r t, or a category,
// \p{..} or \P{..}. In a class, only those that stand for one character
// (oneChar) or a category may come.
func (x *iRegexpParser) escape(inClass bool) bool {
	x.next()
	switch e := x.peek(); {
	case e >= 0 && strings.ContainsRune(`()*+-.?[\]^{|}`, e):
		x.next()
		x.out.WriteByte('\\')
		x.out.WriteRune(e)
	case e == 'n' || e == 'r' || e == 't':
		x.next()
		x.out.WriteByte('\\')
		x.out.WriteRune(e)
	case e == 'p' || e == 'P':
		x.next()
		end := strings.IndexByte(x.src[x.i:], '}')
		if x.peek() != '{' || end < 0 || !categories[x.src[x.i+1:x.i+end]] {
			return false
		}
		x.out.WriteByte('\\')
		x.out.WriteRune(e)
		x.out.WriteString(x.src[x.i : x.i+end+1])
		x.i += end + 1
	default:
		return false // \d, \w, \b and the rest aren't I-Regexp
	}
	return true
}

// categories are the Unicode categories \p{..} may name.
var categories = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range []string{"L", "Ll", "Lm", "Lo", "Lt", "Lu", "M", "Mc", "Me", "Mn", "N", "Nd", "Nl", "No",
		"P", "Pc", "Pd", "Pe", "Pf", "Pi", "Po", "Ps", "Z", "Zl", "Zp", "Zs", "S", "Sc", "Sk", "Sm", "So",
		"C", "Cc", "Cf", "Cn", "Co"} {
		m[c] = true
	}
	return m
}()

// class: "[" ["^"] ("-" / item) *item ["-"] "]", an item a character,
// a range of two, or a category; a character is any but - [ \ ], or an
// escape of one.
func (x *iRegexpParser) class() bool {
	x.next()
	x.out.WriteByte('[')
	if x.peek() == '^' {
		x.next()
		x.out.WriteByte('^')
	}
	first := true
	for {
		switch r := x.peek(); {
		case r == ']' && !first:
			x.next()
			x.out.WriteByte(']')
			return true
		case r == '-' && first:
			x.next()
			x.out.WriteString(`\-`)
		case r == '-':
			x.next()
			if x.peek() != ']' {
				return false // "-" only first or last, or in a range
			}
			x.out.WriteString(`\-`)
		default:
			isCat := r == '\\' && x.i+1 < len(x.src) && (x.src[x.i+1] == 'p' || x.src[x.i+1] == 'P')
			if !x.classChar() {
				return false
			}
			if !isCat && x.peek() == '-' && x.i+1 < len(x.src) && x.src[x.i+1] != ']' {
				x.next()
				x.out.WriteByte('-')
				if x.peek() == '\\' && x.i+1 < len(x.src) && (x.src[x.i+1] == 'p' || x.src[x.i+1] == 'P') {
					return false // a range ends in a character, not a category
				}
				if !x.classChar() {
					return false
				}
				if x.peek() == '-' && x.i+1 < len(x.src) && x.src[x.i+1] != ']' {
					return false // no "a-b-c"
				}
			}
		}
		first = false
	}
}

// classChar: one character of a class, or an escape.
func (x *iRegexpParser) classChar() bool {
	switch r := x.peek(); r {
	case -1, '[', ']', '-', utf8.RuneError:
		return false
	case '\\':
		return x.escape(true)
	default:
		x.next()
		x.out.WriteRune(r)
		return true
	}
}

// Parsing a filter's logical expression.

func (p *pathParser) logicalExpr() logical {
	or := orExpr{p.andExpr()}
	for {
		save := p.pos
		p.skipSpace()
		if !p.eat("||") {
			p.pos = save
			break
		}
		p.skipSpace()
		or = append(or, p.andExpr())
	}
	if len(or) == 1 {
		return or[0]
	}
	return or
}

func (p *pathParser) andExpr() logical {
	and := andExpr{p.basicExpr()}
	for {
		save := p.pos
		p.skipSpace()
		if !p.eat("&&") {
			p.pos = save
			break
		}
		p.skipSpace()
		and = append(and, p.basicExpr())
	}
	if len(and) == 1 {
		return and[0]
	}
	return and
}

// basicExpr parses a parenthesised expression, a comparison, or a test,
// each perhaps after "!" (but for a comparison).
func (p *pathParser) basicExpr() logical {
	defer p.nest()()
	if p.eat("!") {
		p.skipSpace()
		if p.eat("(") {
			e := p.parenRest()
			return notExpr{e}
		}
		return notExpr{p.testExpr()}
	}
	if p.eat("(") {
		return p.parenRest()
	}
	start := p.pos
	left, kind := p.operand()
	save := p.pos
	p.skipSpace()
	if op := p.comparisonOp(); op != "" {
		p.skipSpace()
		right, rkind := p.operand()
		return compareExpr{p.comparandOf(left, kind, start), p.comparandOf(right, rkind, save), op}
	}
	p.pos = save
	return p.testOf(left, kind, start)
}

func (p *pathParser) parenRest() logical {
	p.skipSpace()
	e := p.logicalExpr()
	p.skipSpace()
	if !p.eat(")") {
		p.fail("expected )")
	}
	return e
}

// testExpr parses a query or a function call used as a test.
func (p *pathParser) testExpr() logical {
	start := p.pos
	op, kind := p.operand()
	return p.testOf(op, kind, start)
}

func (p *pathParser) comparisonOp() string {
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.eat(op) {
			return op
		}
	}
	return ""
}

// operand parses a literal, a query or a function call, and says which.
func (p *pathParser) operand() (any, string) {
	switch c := p.peek(); {
	case c == '@' || c == '$':
		p.pos++
		q := filterQuery{relative: c == '@', segs: p.segments(true)}
		q.single = q.singular()
		return q, "query"
	case c == '\'' || c == '"':
		return value{sv: &SV{v: p.stringLiteral()}}, "literal"
	case c == '-' || '0' <= c && c <= '9':
		return value{sv: svOf(p.numberLiteral())}, "literal"
	case 'a' <= c && c <= 'z':
		name := p.functionName()
		switch {
		case p.peek() == '(':
			return p.functionCall(name), "function"
		case name == "true":
			return value{sv: &SV{v: true}}, "literal"
		case name == "false":
			return value{sv: &SV{v: false}}, "literal"
		case name == "null":
			return value{sv: &SV{}}, "literal"
		}
		p.fail("expected a literal, a query or a function call, not %q", name)
	}
	p.fail("expected a literal, a query or a function call")
	return nil, ""
}

// comparandOf checks that an operand can be compared, as RFC 9535
// section 2.4.9 has it: a literal, a singular query, or a function
// returning a value.
func (p *pathParser) comparandOf(op any, kind string, at int) comparand {
	switch kind {
	case "literal":
		return literalValue{op.(value)}
	case "query":
		q := op.(filterQuery)
		if !q.singular() {
			p.pos = at
			p.fail("a query compared must select one node at most: names and indexes only")
		}
		return singularValue{q}
	case "function":
		f := op.(*funcCall)
		if funcs[f.name].result != valueType {
			p.pos = at
			p.fail("%s() doesn't return a value to compare", f.name)
		}
		return funcValue{f}
	}
	return nil
}

// testOf checks that an operand can be a test: a query (true if it
// selects anything) or a function returning a logical value.
func (p *pathParser) testOf(op any, kind string, at int) logical {
	switch kind {
	case "query":
		return existsExpr{op.(filterQuery)}
	case "function":
		f := op.(*funcCall)
		if funcs[f.name].result != logicalType {
			p.pos = at
			p.fail("%s() returns a value, not true or false: compare it", f.name)
		}
		return funcTest{f}
	}
	p.pos = at
	p.fail("a literal isn't a test; compare it")
	return nil
}

// functionName parses a lower-case name: a letter, then letters, digits
// or _.
func (p *pathParser) functionName() string {
	start := p.pos
	for !p.done() {
		c := p.peek()
		if !('a' <= c && c <= 'z' || p.pos > start && (c == '_' || '0' <= c && c <= '9')) {
			break
		}
		p.pos++
	}
	return p.src[start:p.pos]
}

// functionCall parses a call's arguments, checking them against the
// function's parameters.
func (p *pathParser) functionCall(name string) *funcCall {
	def, ok := funcs[name]
	if !ok {
		p.fail("no function %s()", name)
	}
	p.eat("(")
	f := &funcCall{name: name}
	p.skipSpace()
	for !p.eat(")") {
		if len(f.args) > 0 {
			if !p.eat(",") {
				p.fail("expected , or ) in %s()", name)
			}
			p.skipSpace()
		}
		if len(f.args) >= len(def.params) {
			p.fail("%s() takes %d arguments", name, len(def.params))
		}
		at := p.pos
		op, kind := p.operand()
		var arg funcArg
		switch def.params[len(f.args)] {
		case valueType:
			arg.val = p.comparandOf(op, kind, at)
		case nodesType:
			if kind != "query" {
				p.pos = at
				p.fail("%s() takes a query", name)
			}
			q := op.(filterQuery)
			arg.q = &q
		}
		f.args = append(f.args, arg)
		p.skipSpace()
	}
	if len(f.args) != len(def.params) {
		p.fail("%s() takes %d arguments", name, len(def.params))
	}
	return f
}

// numberLiteral parses a JSON number, as RFC 9535 writes one: an integer
// (or -0), then perhaps a fraction and an exponent. An integer without
// either is an int64, as FromJSON makes one.
func (p *pathParser) numberLiteral() any {
	start := p.pos
	p.eat("-")
	digits := p.pos
	for !p.done() && '0' <= p.peek() && p.peek() <= '9' {
		p.pos++
	}
	if p.pos == digits || p.src[digits] == '0' && p.pos-digits > 1 {
		p.pos = start
		p.fail("a bad number")
	}
	isInt := true
	if p.peek() == '.' {
		p.pos++
		if p.done() || p.peek() < '0' || p.peek() > '9' {
			p.fail("a bad number: digits must follow .")
		}
		for !p.done() && '0' <= p.peek() && p.peek() <= '9' {
			p.pos++
		}
		isInt = false
	}
	if c := p.peek(); c == 'e' || c == 'E' {
		p.pos++
		if c := p.peek(); c == '+' || c == '-' {
			p.pos++
		}
		if p.done() || p.peek() < '0' || p.peek() > '9' {
			p.fail("a bad number: digits must follow e")
		}
		for !p.done() && '0' <= p.peek() && p.peek() <= '9' {
			p.pos++
		}
		isInt = false
	}
	text := p.src[start:p.pos]
	if isInt {
		if n, err := strconv.ParseInt(text, 10, 64); err == nil {
			return n
		}
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		p.fail("a bad number: %v", err)
	}
	return keepNumber(text, f)
}
