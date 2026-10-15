package scalar

import (
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// A byte string is matched as characters from 0 to 255, one per byte (see
// SV.Bytes), so a pattern such as \x89PNG finds those bytes, and captures,
// substitutions and splits of a byte string are byte strings.
//
// Regex helpers use github.com/dlclark/regexp2, a backtracking engine with
// near-Perl syntax: lookahead, lookbehind, backreferences, atomic groups and
// named captures all work. Patterns are compiled on first use and cached, so
// they can be written inline as in Perl. The cache holds 4,096 patterns, so
// patterns built from data can't make it grow without end. They can also be compiled up front,
// as with Perl's qr//, with Compile or MustCompile: M, Named, S and Split
// take either a pattern string or a *Regexp.
//
// Flags are passed as an optional trailing string, as in Perl: M, Named
// and S take "i", "m", "s" and "x"; M and S also take "g", and S takes "r".
// Split's optional argument is its limit, so its flags go inside the
// pattern, as in (?i). Invalid patterns, unknown flags and timeouts panic,
// as Perl would die.
//
// One difference from Perl: named groups are numbered after all unnamed
// groups, so in (?<y>\d+)-(\d+) the unnamed group is $1 and y is $2. Use
// Named to get named groups by name.

// RegexTimeout caps how long a single match may run, so that a pathological
// pattern meeting unexpected input can't hang the program. Set it before
// first use; patterns already compiled keep the timeout they were built with.
var RegexTimeout = 100 * time.Millisecond

var regexCache = boundedCache[cacheKey, *regexp2.Regexp]{limit: cacheLimit}

type cacheKey struct {
	pattern string
	opts    regexp2.RegexOptions
	timeout time.Duration
}

// flagSet holds the single-letter flags of a Perl-style operator.
type flagSet string

func parseFlags(op, allowed string, flags []string) flagSet {
	f := strings.Join(flags, "")
	for _, c := range f {
		if !strings.ContainsRune(allowed, c) {
			panic(fmt.Sprintf("scalar: %s doesn't support flag %q (allowed: %s)", op, c, allowed))
		}
	}
	return flagSet(f)
}

func (f flagSet) has(c rune) bool { return strings.ContainsRune(string(f), c) }

// compile returns the compiled pattern for a pattern string and its flags,
// from the cache if it's been compiled before. A bad pattern panics.
func compile(pattern string, f flagSet) *regexp2.Regexp {
	rx, err := compileErr(pattern, f)
	if err != nil {
		panic("scalar: " + err.Error())
	}
	return rx
}

// compileErr is compile, returning a bad pattern as an error.
func compileErr(pattern string, f flagSet) (*regexp2.Regexp, error) {
	var opts regexp2.RegexOptions
	if f.has('i') {
		opts |= regexp2.IgnoreCase
	}
	if f.has('m') {
		opts |= regexp2.Multiline
	}
	if f.has('s') {
		opts |= regexp2.Singleline
	}
	if f.has('x') {
		opts |= regexp2.IgnorePatternWhitespace
	}

	key := cacheKey{pattern, opts, RegexTimeout}
	if re, ok := regexCache.load(key); ok {
		return re, nil
	}
	re, err := regexp2.Compile(pattern, opts)
	if err != nil {
		return nil, fmt.Errorf("bad regex %q: %v", pattern, err)
	}
	re.MatchTimeout = RegexTimeout
	return regexCache.store(key, re), nil
}

// Regexp is a compiled pattern, like the result of Perl's qr//. Its flags
// ("i", "m", "s", "x") are fixed when it's compiled; M, Named, S and Split
// accept it wherever they accept a pattern string:
//
//	re := scalar.MustCompile(`£([\d,]+)`, "i") // my $re = qr/£([\d,]+)/i;
//	s.M(re, "g")                               // $s =~ /$re/g
type Regexp struct {
	rx *regexp2.Regexp
}

// Compile compiles pattern with flags, any of "i", "m", "s" and "x", or
// returns an error if the pattern is invalid. An unknown flag panics, as it's
// a mistake in the program rather than in the pattern.
func Compile(pattern string, flags ...string) (*Regexp, error) {
	rx, err := compileErr(pattern, parseFlags("Compile", "imsx", flags))
	if err != nil {
		return nil, fmt.Errorf("scalar: %w", err)
	}
	return &Regexp{rx}, nil
}

// MustCompile is Compile, panicking if the pattern is invalid. Use it for
// patterns in the program, in package-level variables, so that a bad one
// fails as soon as the program starts, as with regexp.MustCompile.
func MustCompile(pattern string, flags ...string) *Regexp {
	return &Regexp{compile(pattern, parseFlags("MustCompile", "imsx", flags))}
}

// String returns the pattern as written.
func (r *Regexp) String() string { return r.rx.String() }

// regex resolves the pattern argument of op: a pattern string, compiled with
// the flags f, or a *Regexp, whose own flags apply. Giving a *Regexp the
// flags it was compiled with (or any others of "imsx") panics: they belong
// with Compile, as with qr//.
func regex(op string, re any, f flagSet) *regexp2.Regexp {
	switch re := re.(type) {
	case string:
		return compile(re, f)
	case *Regexp:
		if re == nil {
			panic(fmt.Sprintf("scalar: %s given a nil *Regexp", op))
		}
		for _, c := range "imsx" {
			if f.has(c) {
				panic(fmt.Sprintf("scalar: %s: flag %q belongs with Compile when the pattern is a *Regexp, as with qr//", op, c))
			}
		}
		return re.rx
	default:
		panic(fmt.Sprintf("scalar: %s pattern must be a string or *Regexp, got %T", op, re))
	}
}

// check panics if a regexp2 call failed. regexp2's timeout error quotes the
// entire input, so the message is truncated.
func check(rx *regexp2.Regexp, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if len(msg) > 200 {
		msg = strings.ToValidUTF8(msg[:200], "") + "..."
	}
	panic(fmt.Sprintf("scalar: regex %q: %s", rx.String(), msg))
}

// matches iterates over successive matches of rx in text. regexp2 works on
// runes, and match positions are rune offsets into text.
func matches(rx *regexp2.Regexp, text []rune) iter.Seq[*regexp2.Match] {
	return func(yield func(*regexp2.Match) bool) {
		m, err := rx.FindRunesMatch(text)
		for {
			check(rx, err)
			if m == nil || !yield(m) {
				return
			}
			m, err = rx.FindNextMatch(m)
		}
	}
}

// firstRegexMatch returns the first match of rx in str, or nil.
func firstRegexMatch(rx *regexp2.Regexp, str string) *regexp2.Match {
	m, err := rx.FindStringMatch(str)
	check(rx, err)
	return m
}

// matchAV turns a match into an array: index 0 is the whole match ($&),
// index n is group n ($n), and groups that didn't participate are undef.
func matchAV(m *regexp2.Match, isBytes bool) *AV {
	groups := m.Groups()
	a := &AV{items: make([]*SV, len(groups))}
	for i := range groups {
		a.items[i] = groupSV(&groups[i], isBytes)
	}
	return a
}

// groupSV is a group's captured text, or undef if the group didn't take part
// in the match.
func groupSV(g *regexp2.Group, isBytes bool) *SV {
	if g == nil || len(g.Captures) == 0 {
		return &SV{}
	}
	return textResult(g.String(), isBytes) // a byte string's captures are byte strings
}

// M is $s =~ /re/ with captures. It returns nil if there is no match;
// otherwise Get(0) is the whole match and Get(n) is $n. A nil *AV is safe
// to read, so s.M(re).Get(1).Str() is "" when nothing matched.
//
// With the "g" flag it is $s =~ /re/g: it returns one array reference per
// match, each laid out as above, or nil if nothing matched.
//
// re is a pattern string or a *Regexp (see Compile).
func (s *SV) M(re any, flags ...string) *AV {
	f := parseFlags("M", "imsxg", flags)
	rx := regex("M", re, f)
	str, isBytes := s.text()
	if !f.has('g') {
		if m := firstRegexMatch(rx, str); m != nil {
			return matchAV(m, isBytes)
		}
		return nil
	}
	var out *AV
	for m := range matches(rx, []rune(str)) {
		if out == nil {
			out = &AV{}
		}
		out.items = append(out.items, &SV{v: matchAV(m, isBytes)})
	}
	return out
}

// Named is $s =~ /re/ followed by reading %+. It returns nil if there
// is no match, otherwise a hash of named groups; ones that didn't
// participate are undef. re is a pattern string or a *Regexp.
func (s *SV) Named(re any, flags ...string) *HV {
	rx := regex("Named", re, parseFlags("Named", "imsx", flags))
	str, isBytes := s.text()
	m := firstRegexMatch(rx, str)
	if m == nil {
		return nil
	}
	h := newHV(0)
	for _, name := range rx.GetGroupNames() {
		if _, err := strconv.Atoi(name); err == nil {
			continue // unnamed groups are named by number
		}
		h.m[name] = groupSV(m.GroupByName(name), isBytes)
	}
	return h
}

// S is Perl's s///. repl is either a replacement string, in which $1,
// ${name}, $& and $$ are expanded, or a func(*AV) any (or func(*AV) string)
// that receives the match as M lays it out, like s///e.
//
// Flags: "g" replaces every match; "r" leaves s alone and returns the new
// string. Without "r", s is modified in place (only if something matched)
// and the number of substitutions is returned.
//
// re is a pattern string or a *Regexp.
func (s *SV) S(re any, repl any, flags ...string) *SV {
	f := parseFlags("S", "imsxgr", flags)
	limit := 1
	if f.has('g') {
		limit = -1
	}

	// A pattern that matches only itself, with plain replacement text, needs
	// no regex: S(",", "", "gr") is strings.ReplaceAll.
	pat, isString := re.(string)
	text, plain := repl.(string)
	if fastPaths && isString && plain && literalPattern(pat) && !f.has('i') && !f.has('x') && !strings.Contains(text, "$") {
		str, isBytes := s.text()
		n := strings.Count(str, pat)
		if limit == 1 {
			n = min(n, 1)
		}
		return s.substituted(strings.Replace(str, pat, text, limit), n, isBytes, f)
	}

	rx := regex("S", re, f)
	str, isBytes := s.text()
	n := 0
	var out string
	switch r := repl.(type) {
	case string:
		if !strings.Contains(r, "$") {
			// Plain text needs no expanding, so one pass replaces and counts.
			var err error
			out, err = rx.ReplaceFunc(str, func(regexp2.Match) string {
				n++
				return r
			}, -1, limit)
			check(rx, err)
			break
		}
		// regexp2 expands $1, ${name} and so on only in Replace, which
		// doesn't say how many it replaced, so count them first.
		for range matches(rx, []rune(str)) {
			n++
			if n == limit {
				break
			}
		}
		if n > 0 {
			var err error
			out, err = rx.Replace(str, r, -1, limit)
			check(rx, err)
		}
	case func(*AV) any:
		var err error
		out, err = rx.ReplaceFunc(str, func(m regexp2.Match) string {
			n++
			text, _ := operand(r(matchAV(&m, isBytes))).text()
			return text
		}, -1, limit)
		check(rx, err)
	case func(*AV) string:
		var err error
		out, err = rx.ReplaceFunc(str, func(m regexp2.Match) string {
			n++
			return r(matchAV(&m, isBytes))
		}, -1, limit)
		check(rx, err)
	default:
		panic(fmt.Sprintf("scalar: S replacement must be a string or func(*AV) any, got %T", repl))
	}
	if n == 0 {
		out = str
	}
	return s.substituted(out, n, isBytes, f)
}

// substituted finishes S: with "r", the new string; otherwise s takes it, if
// anything was replaced, and the count is returned.
func (s *SV) substituted(out string, n int, isBytes bool, f flagSet) *SV {
	if f.has('r') {
		return textResult(out, isBytes)
	}
	if n > 0 {
		s.Set(textResult(out, isBytes))
	}
	return Value(n)
}

// Split is Perl's split /re/, $s. Captured groups are included in the
// result (undef if they didn't participate), and trailing empty fields are
// dropped. A pattern of " " is awk mode: leading whitespace is ignored and
// fields are separated by runs of whitespace.
//
// An optional limit works as Perl's LIMIT: a positive limit returns at most
// that many fields (the last holding the rest of the string), and a negative
// limit keeps trailing empty fields.
//
// re is a pattern string or a *Regexp. Awk mode is only for the string " ",
// as in Perl, where split qr/ / splits on single spaces.
func (s *SV) Split(re any, limit ...int) *AV {
	str, isBytes := s.text()
	lim := 0
	if len(limit) > 0 {
		lim = limit[0]
	}
	if re == " " {
		str = strings.TrimLeft(str, perlSpace)
		re = `\s+`
	}
	out := &AV{}
	if str == "" {
		return out
	}

	rx := regex("Split", re, "")
	runes := []rune(str)
	pos, splits := 0, 0
	for m := range matches(rx, runes) {
		if lim > 0 && splits >= lim-1 {
			break
		}
		// A zero-width match can't split where the last match ended, or at
		// the start of the string.
		if m.Length == 0 && m.Index == pos {
			continue
		}
		out.Push(textResult(string(runes[pos:m.Index]), isBytes))
		if m.GroupCount() > 1 {
			groups := matchAV(m, isBytes)
			out.items = append(out.items, groups.items[1:]...)
		}
		pos = m.Index + m.Length
		splits++
	}
	out.Push(textResult(string(runes[pos:]), isBytes))

	if lim == 0 {
		for out.Length() > 0 && out.Get(-1).Str() == "" {
			out.Pop()
		}
	}
	return out
}

// Tr is Perl's tr/// (y///). from and to are character lists, and may
// contain ranges like "a-z"; write "\-" for a literal hyphen and "\\" for a
// backslash. If to is shorter than from, its last character is repeated.
//
// Flags: "d" deletes characters in from that have no counterpart in to;
// "s" squeezes runs of the same translated character into one; "c" uses
// every character *not* in from, all mapped to the last character of to
// (or deleted with "d" and an empty to); "r" leaves s alone and returns
// the new string. Without "r", s is modified in place and the number of
// characters matched is returned, so Tr("a-z", "") counts lowercase letters.
func (s *SV) Tr(from, to string, flags ...string) *SV {
	f := parseFlags("Tr", "cdsr", flags)
	del, squeeze := f.has('d'), f.has('s')
	fromSet, toList := expandTr(from), expandTr(to)

	// translate returns the replacement for c, whether c was in the search
	// list, and whether it should be deleted.
	var translate func(c rune) (rune, bool, bool)
	if f.has('c') {
		in := map[rune]bool{}
		for _, c := range fromSet {
			in[c] = true
		}
		translate = func(c rune) (rune, bool, bool) {
			switch {
			case in[c]:
				return c, false, false
			case len(toList) > 0:
				return toList[len(toList)-1], true, false
			}
			return c, true, del
		}
	} else {
		if len(toList) == 0 && !del {
			toList = fromSet
		}
		type mapping struct {
			to  rune
			del bool
		}
		table := map[rune]mapping{}
		for i, c := range fromSet {
			if _, seen := table[c]; seen {
				continue // first occurrence wins
			}
			switch {
			case i < len(toList):
				table[c] = mapping{to: toList[i]}
			case del:
				table[c] = mapping{del: true}
			default:
				table[c] = mapping{to: toList[len(toList)-1]}
			}
		}
		translate = func(c rune) (rune, bool, bool) {
			m, ok := table[c]
			if !ok {
				return c, false, false
			}
			return m.to, true, m.del
		}
	}

	str, isBytes := s.text()
	var out strings.Builder
	count := 0
	var last rune
	lastTranslated := false
	for _, c := range str {
		rep, matched, deleted := translate(c)
		if !matched {
			out.WriteRune(c)
			lastTranslated = false
			continue
		}
		count++
		if deleted {
			continue
		}
		if squeeze && lastTranslated && rep == last {
			continue
		}
		out.WriteRune(rep)
		last, lastTranslated = rep, true
	}

	if f.has('r') {
		return textResult(out.String(), isBytes)
	}
	if result := out.String(); result != str {
		s.Set(textResult(result, isBytes))
	}
	return Value(count)
}

// expandTr expands a tr/// character list: ranges like a-z, and the escapes
// \-, \\, \n, \t and \r. A hyphen at either end is literal.
func expandTr(spec string) []rune {
	var chars []rune
	escaped := map[int]bool{} // indexes in chars that came from escapes
	in := []rune(spec)
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c == '\\' && i+1 < len(in) {
			i++
			switch in[i] {
			case 'n':
				c = '\n'
			case 't':
				c = '\t'
			case 'r':
				c = '\r'
			default:
				c = in[i]
			}
			escaped[len(chars)] = true
		}
		chars = append(chars, c)
	}

	var out []rune
	for i := 0; i < len(chars); i++ {
		if i+2 < len(chars) && chars[i+1] == '-' && !escaped[i+1] && chars[i] <= chars[i+2] {
			for c := chars[i]; c <= chars[i+2]; c++ {
				out = append(out, c)
			}
			i += 2
			continue
		}
		out = append(out, chars[i])
	}
	return out
}
