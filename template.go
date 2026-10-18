package scalar

import (
	"fmt"
	"strings"
)

// Interpolate fills in a template from s, as Perl interpolates variables in
// a double-quoted string, with the keys of the hash s holds as the
// variables:
//
//	doc.Interpolate("M$mag near $place: @stations")    // "M5.1 near Ridgecrest, CA: PAS GSC"
//	doc.Interpolate("Depth $depth{km} km")             // $doc->{depth}{km}
//	doc.Interpolate("Depth ${depth.km} km")            // the same: any At path in ${...}
//
// $name is a value, and subscripts straight after it are part of it, as in
// Perl: $depth{km}, $stations[0], $content[0]{rule_set}. ${...} takes any
// path At takes, so ${depth.km} works too; $place.x is the place followed
// by ".x", as in Perl. @name and @{...} join the elements of an array with
// a space, as Perl's "@a" does with $" (MustTemplate's ListSeparator
// changes it); for anything else they give its string, as $ does.
//
// A missing value gives "", and nothing is created. As in Perl, a reference
// gives "ARRAY(0x...)" or "HASH(0x...)": use @ for an array. \$ and \@ give
// a literal $ and @, as in Perl, so an email address is written
// "me\@example.com"; a $ or @ not followed by a name or {, as in "$5", is
// literal too.
//
// template is a string or a *Template from MustTemplate or CompileTemplate.
// A string is parsed once and cached; a malformed one panics, as a bad regex
// does.
func (s *SV) Interpolate(template any) string {
	var t *Template
	switch v := template.(type) {
	case string:
		t = cachedTemplate(v)
	case *Template:
		if v == nil {
			panic("scalar: Interpolate with a nil *Template")
		}
		t = v
	default:
		panic(fmt.Sprintf("scalar: Interpolate's template must be a string or a *Template, not %T", template))
	}
	return t.fill(s)
}

// Interpolate fills in a template from h; see SV.Interpolate.
func (h *HV) Interpolate(template any) string { return (&SV{v: h}).Interpolate(template) }

// Template is a compiled template for Interpolate, like a compiled regex:
// a template used often is parsed once, a bad one fails as soon as the
// program starts, and the separator for lists can be set.
//
//	var summary = scalar.MustTemplate("$place. Stations: @stations.").ListSeparator(", ")
//	doc.Interpolate(summary)
type Template struct {
	src   string
	parts []templatePart
	sep   string
}

// templatePart is a piece of a template: text, or a value to fill in.
type templatePart struct {
	text  string   // text to copy, when steps is nil
	steps []atStep // the path of a value
	list  bool     // @: join an array's elements
}

// CompileTemplate compiles a template for Interpolate, or returns an error
// if it's malformed, for templates that come from outside the program.
func CompileTemplate(template string) (*Template, error) {
	parts, err := parseTemplateParts(template)
	if err != nil {
		return nil, err
	}
	return &Template{src: template, parts: parts, sep: " "}, nil
}

// MustTemplate is CompileTemplate, panicking if the template is malformed.
// Use it for templates in the program, in package-level variables, so that
// a bad one fails as soon as the program starts.
func MustTemplate(template string) *Template {
	t, err := CompileTemplate(template)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// ListSeparator returns a copy of t that joins the elements of an array
// with sep, as local $" = sep does in Perl. The default is a space.
func (t *Template) ListSeparator(sep string) *Template {
	c := *t
	c.sep = sep
	return &c
}

// String returns the template as written.
func (t *Template) String() string { return t.src }

var templateCache = boundedCache[string, *Template]{limit: cacheLimit}

func cachedTemplate(src string) *Template {
	if t, ok := templateCache.load(src); ok {
		return t
	}
	return templateCache.store(src, MustTemplate(src))
}

func (t *Template) fill(s *SV) string {
	var b strings.Builder
	for _, p := range t.parts {
		if p.steps == nil {
			b.WriteString(p.text)
			continue
		}
		v, ok := s.PathOK(&PathExpr{steps: p.steps})
		if !ok {
			continue // missing: ""
		}
		if a, isArray := v.val().(*AV); p.list && isArray {
			for i, e := range a.items {
				if i > 0 {
					b.WriteString(t.sep)
				}
				b.WriteString(e.Str()) // a hole is ""
			}
			continue
		}
		b.WriteString(v.Str())
	}
	return b.String()
}

// parseTemplateParts splits a template into text and the values to fill in.
func parseTemplateParts(src string) ([]templatePart, error) {
	fail := func(pos int, format string, args ...any) error {
		return fmt.Errorf("scalar: bad template %q at offset %d: %s", src, pos, fmt.Sprintf(format, args...))
	}
	var parts []templatePart
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			parts = append(parts, templatePart{text: text.String()})
			text.Reset()
		}
	}
	isNameStart := func(c byte) bool { return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }
	isNameChar := func(c byte) bool { return isNameStart(c) || '0' <= c && c <= '9' }

	for i := 0; i < len(src); {
		c := src[i]
		if c == '\\' && i+1 < len(src) && strings.IndexByte(`$@\`, src[i+1]) >= 0 {
			text.WriteByte(src[i+1])
			i += 2
			continue
		}
		if (c != '$' && c != '@') || i+1 >= len(src) || !isNameStart(src[i+1]) && src[i+1] != '{' {
			text.WriteByte(c)
			i++
			continue
		}

		start := i
		var path string
		if src[i+1] == '{' { // ${path} or @{path}
			end := matchClose(src, i+1)
			if end < 0 {
				return nil, fail(i+1, "no } to match the {")
			}
			path = src[i+2 : end]
			if path == "" {
				return nil, fail(i+1, "empty ${}")
			}
			i = end + 1
		} else { // $name, and any subscripts straight after it
			j := i + 1
			for j < len(src) && isNameChar(src[j]) {
				j++
			}
			for j < len(src) && (src[j] == '[' || src[j] == '{') {
				end := matchClose(src, j)
				if end < 0 {
					return nil, fail(j, "no %c to match the %c", map[byte]byte{'[': ']', '{': '}'}[src[j]], src[j])
				}
				j = end + 1
			}
			path = src[i+1 : j]
			i = j
		}
		steps, err := parseAt(path)
		if err != nil {
			return nil, fail(start, "%s", strings.TrimPrefix(err.Error(), "scalar: "))
		}
		flush()
		parts = append(parts, templatePart{steps: steps, list: c == '@'})
	}
	flush()
	return parts, nil
}

// matchClose returns the offset of the ] or } that closes the [ or { at
// src[i], allowing for nested brackets, quoted keys and backslash escapes
// as At reads them, or -1 if there isn't one.
func matchClose(src string, i int) int {
	depth := 0
	var quote byte
	for ; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '\\':
			i++ // skip the escaped character
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case (c == '"' || c == '\'') && (src[i-1] == '[' || src[i-1] == '{'):
			quote = c // a quoted key, as in ["k"]; a quote anywhere else is just a character
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
