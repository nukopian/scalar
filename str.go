package scalar

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Perl's string functions. Like Perl's, they work on s in string context
// (undef is ""), count characters rather than bytes, and, apart from
// SubstrReplace, return a new scalar and leave s alone.

// Length is Perl's length($s): the number of characters (not bytes) in s
// in string context, so Scalar(12345).Length() is 5 and "café" is 4. Perl
// returns undef for undef; this returns 0.
func (s *SV) Length() int { return utf8.RuneCountInString(s.Str()) }

// Substr is Perl's substr($s, offset) or substr($s, offset, length): a new
// scalar holding part of s in string context, counted in characters.
//
// As in Perl, a negative offset counts from the end, an omitted length
// means "to the end", and a negative length leaves that many characters
// off the end. A substring partly outside the string gives the part that's
// inside; one entirely outside gives undef.
func (s *SV) Substr(offset int, length ...int) *SV {
	runes := []rune(s.Str())
	start, end, ok := substrBounds(len(runes), offset, length)
	if !ok {
		return &SV{}
	}
	return &SV{v: string(runes[start:end])}
}

// SubstrReplace is Perl's 4-argument substr($s, offset, length, replacement),
// or assigning to substr: it replaces that part of s with replacement (in
// string context), changing s in place, and returns the part it replaced.
// Offset and length work as in Substr. A substring entirely outside the
// string panics, as Perl dies with "substr outside of string".
func (s *SV) SubstrReplace(offset, length int, replacement any) *SV {
	runes := []rune(s.Str())
	start, end, ok := substrBounds(len(runes), offset, []int{length})
	if !ok {
		panic("scalar: substr outside of string")
	}
	removed := string(runes[start:end])
	s.Set(string(runes[:start]) + operand(replacement).Str() + string(runes[end:]))
	return &SV{v: removed}
}

// substrBounds turns Perl's substr offset and optional length into rune
// indexes 0 <= start <= end <= n, following Perl's
// translate_substr_offsets. ok is false if the substring lies entirely
// outside the string.
func substrBounds(n, offset int, length []int) (start, end int, ok bool) {
	start = offset
	if start < 0 && n > 0 {
		start += n
	}
	if start > n {
		return 0, 0, false
	}

	end = n
	if len(length) > 0 {
		switch l := length[0]; {
		case l < 0:
			end = n + l
		case start < 0:
			end = start + l
		default:
			end = min(start+l, n)
		}
	}

	switch {
	case end < 0 && start < 0:
		return 0, 0, false
	case end < 0:
		end = 0
	case start < 0:
		start = 0
	}
	end = min(max(end, start), n)
	return start, end, true
}

// Index is Perl's index($s, substr[, position]): the character position of
// the first occurrence of substr at or after position (default 0), or -1.
// Positions count characters, as in Substr, so the result can be passed
// straight to it; Go's strings.Index counts bytes.
func (s *SV) Index(substr any, position ...int) int {
	str, sub := s.Str(), operand(substr).Str()
	pos := clampPosition(position, 0, utf8.RuneCountInString(str))
	start := byteOffset(str, pos)
	i := strings.Index(str[start:], sub)
	if i < 0 {
		return -1
	}
	return pos + utf8.RuneCountInString(str[start:start+i])
}

// Rindex is Perl's rindex($s, substr[, position]): the character position
// of the last occurrence of substr starting at or before position (default
// the end of the string), or -1.
func (s *SV) Rindex(substr any, position ...int) int {
	str, sub := s.Str(), operand(substr).Str()
	n := utf8.RuneCountInString(str)
	pos := clampPosition(position, n, n)
	end := min(byteOffset(str, pos)+len(sub), len(str))
	i := strings.LastIndex(str[:end], sub)
	if i < 0 {
		return -1
	}
	return utf8.RuneCountInString(str[:i])
}

// clampPosition is the optional position argument of Index and Rindex,
// or def, kept within 0..n as Perl does.
func clampPosition(position []int, def, n int) int {
	if len(position) == 0 {
		return def
	}
	return min(max(position[0], 0), n)
}

// byteOffset is the byte offset of the nth character of str.
func byteOffset(str string, n int) int {
	for i := range str {
		if n == 0 {
			return i
		}
		n--
	}
	return len(str)
}

// Lc is Perl's lc: s in lower case, using Unicode's full case mappings.
func (s *SV) Lc() *SV { return &SV{v: cases.Lower(language.Und).String(s.Str())} }

// Uc is Perl's uc: s in upper case, using Unicode's full case mappings, so
// "straße" becomes "STRASSE".
func (s *SV) Uc() *SV { return &SV{v: cases.Upper(language.Und).String(s.Str())} }

// Fc is Perl's fc: s case-folded, for comparing strings regardless of case.
// Use it rather than Lc for that: Fc("Straße") and Fc("STRASSE") are equal,
// but their Lc forms aren't.
func (s *SV) Fc() *SV { return &SV{v: cases.Fold().String(s.Str())} }

// Ucfirst is Perl's ucfirst: s with its first character in title case.
func (s *SV) Ucfirst() *SV {
	return s.mapFirst(cases.Title(language.Und, cases.NoLower))
}

// Lcfirst is Perl's lcfirst: s with its first character in lower case.
func (s *SV) Lcfirst() *SV {
	return s.mapFirst(cases.Lower(language.Und))
}

func (s *SV) mapFirst(c cases.Caser) *SV {
	str := s.Str()
	_, size := utf8.DecodeRuneInString(str)
	return &SV{v: c.String(str[:size]) + str[size:]}
}

// Trim is Perl's trim (builtin::trim, Perl 5.36): s without leading or
// trailing whitespace, including Unicode spaces such as no-break space.
func (s *SV) Trim() *SV { return &SV{v: strings.TrimSpace(s.Str())} }

// Quotemeta is Perl's quotemeta: x in string context with every character
// that could mean something in a pattern backslash-escaped, so that it
// matches literally. Use it to build patterns from data:
//
//	s.M(`(?i)\b` + scalar.Quotemeta(skill))   // safe for "C++" or ".NET"
//
// Like Perl, it escapes every ASCII character other than letters, digits
// and underscore. It also escapes Unicode whitespace, so that patterns
// using the "x" flag don't ignore it.
func Quotemeta(x any) string {
	str := operand(x).Str()
	var b strings.Builder
	b.Grow(len(str) + len(str)/4)
	for _, r := range str {
		if r < utf8.RuneSelf && !isWordByte(byte(r)) || r >= utf8.RuneSelf && unicode.IsSpace(r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isWordByte(c byte) bool {
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}
