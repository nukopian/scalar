package scalar

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
func (e existsExpr) test(cur, root *SV) bool { return len(e.q.nodes(cur, root)) > 0 }
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
}

func (q filterQuery) nodes(cur, root *SV) []*SV {
	from := root
	if q.relative {
		from = cur
	}
	return query(from, root, q.segs)
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
	if ns := s.q.nodes(cur, root); len(ns) == 1 {
		return value{sv: ns[0]}
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

func numberOf(s *SV) float64 {
	switch v := s.val().(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	}
	return math.NaN()
}

// equal is ==, as RFC 9535 section 2.3.5.2.2 defines it: Nothing equals
// only Nothing; values of different JSON types are unequal; numbers
// compare as numbers (1 == 1.0), strings by their characters, and arrays
// and objects element by element.
func (a value) equal(b value) bool {
	if a.nothing || b.nothing {
		return a.nothing && b.nothing
	}
	return jsonEqual(a.sv, b.sv)
}

func jsonEqual(a, b *SV) bool {
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
		return numberOf(a) == numberOf(b)
	case jString:
		return a.Str() == b.Str()
	case jArray:
		x, y := a.val().(*AV), b.val().(*AV)
		if len(x.items) != len(y.items) {
			return false
		}
		for i := range x.items {
			xe, _ := element(x, i)
			ye, _ := element(y, i)
			if !jsonEqual(xe, ye) {
				return false
			}
		}
		return true
	case jObject:
		x, y := a.val().(*HV), b.val().(*HV)
		if len(x.m) != len(y.m) {
			return false
		}
		for k, xv := range x.m {
			yv, ok := y.m[k]
			if !ok || !jsonEqual(xv, yv) {
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
		return numberOf(a.sv) < numberOf(b.sv)
	case ka == jString && kb == jString:
		return a.sv.Str() < b.sv.Str() // byte order of UTF-8 is code point order
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
		case string:
			return number(utf8.RuneCountInString(x))
		case []byte:
			return number(utf8.RuneCount(x))
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
	r, ok := iRegexp(re.sv.Str(), f.name == "match")
	return ok && r.MatchString(s.sv.Str())
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

var iRegexps sync.Map // "m"/"s" + pattern → *regexp.Regexp, or nil if invalid

func iRegexp(pattern string, whole bool) (*regexp.Regexp, bool) {
	key := "s" + pattern
	if whole {
		key = "m" + pattern
	}
	if r, ok := iRegexps.Load(key); ok {
		re, _ := r.(*regexp.Regexp)
		return re, re != nil
	}
	var re *regexp.Regexp
	if goRE, ok := translateIRegexp(pattern); ok {
		if whole {
			goRE = `\A(?:` + goRE + `)\z`
		}
		re, _ = regexp.Compile(goRE)
	}
	iRegexps.Store(key, re)
	return re, re != nil
}

// translateIRegexp returns pattern as Go regexp syntax, and false if it
// isn't I-Regexp.
func translateIRegexp(pattern string) (string, bool) {
	var b strings.Builder
	inClass := false
	quantified := false // the last token was a quantifier, so a "?" now would make it lazy
	for i := 0; i < len(pattern); {
		r, size := utf8.DecodeRuneInString(pattern[i:])
		if r == utf8.RuneError && size == 1 {
			return "", false
		}
		i += size
		wasQuantified := quantified
		quantified = !inClass && (r == '*' || r == '+' || r == '?' || r == '}')
		switch {
		case r == '\\':
			if i >= len(pattern) {
				return "", false
			}
			e := pattern[i]
			i++
			switch {
			case strings.IndexByte(`()*+-.?[\]^{|}`, e) >= 0:
				b.WriteByte('\\')
				b.WriteByte(e)
			case e == 'n':
				b.WriteString(`\n`)
			case e == 'r':
				b.WriteString(`\r`)
			case e == 't':
				b.WriteString(`\t`)
			case e == 'p' || e == 'P':
				end := strings.IndexByte(pattern[i:], '}')
				if i >= len(pattern) || pattern[i] != '{' || end < 0 {
					return "", false
				}
				b.WriteByte('\\')
				b.WriteByte(e)
				b.WriteString(pattern[i : i+end+1])
				i += end + 1
			default:
				return "", false // \d, \w, \b and the rest aren't I-Regexp
			}
		case inClass:
			if r == ']' {
				inClass = false
			}
			if r == '[' {
				return "", false // no nested classes or subtraction
			}
			b.WriteRune(r)
		case r == '[':
			inClass = true
			b.WriteRune(r)
			if strings.HasPrefix(pattern[i:], "^") {
				b.WriteByte('^')
				i++
			}
			if strings.HasPrefix(pattern[i:], "]") {
				return "", false // an empty class
			}
		case r == '.':
			b.WriteString(`[^\n\r]`)
		case r == '?' && wasQuantified:
			return "", false // a lazy quantifier isn't I-Regexp
		default:
			b.WriteRune(r)
		}
	}
	if inClass {
		return "", false
	}
	return b.String(), true
}

// Parsing a filter's logical expression.

func (p *pathParser) logicalExpr() logical {
	defer p.nest()()
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
	if kind == "literal" {
		p.pos = start
		p.fail("a literal isn't a test; compare it")
	}
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
		return q, "query"
	case c == '\'' || c == '"':
		return value{sv: &SV{v: p.stringLiteral()}}, "literal"
	case c == '-' || '0' <= c && c <= '9':
		return value{sv: &SV{v: p.numberLiteral()}}, "literal"
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
	return f
}
