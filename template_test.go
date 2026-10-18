package scalar

import (
	"strings"
	"testing"
)

func TestInterpolate(t *testing.T) {
	doc, err := FromJSON([]byte(`{
		"place": "Cobb, CA", "network": "nc", "stations": ["PAS", "GSC"],
		"depth": {"max": 5500, "min": null}, "content": [{"rule_set": "A"}],
		"meta": {"log.level": "debug", "it's": "apostrophe", "a}b": "brace"}, "x": "X", "_u": "u"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	for tmpl, want := range map[string]string{
		"$place via $network: @stations":         "Cobb, CA via nc: PAS GSC",
		"Up to $depth{max}, from ${depth.max}":   "Up to 5500, from 5500",
		"$stations[1] $stations[-1] @{stations}": "GSC GSC PAS GSC",
		"$content[0]{rule_set}":                  "A",
		"${content[0].rule_set}":                 "A",
		"$meta{log.level} ${meta{log.level}}":    "debug debug",
		`$meta{it's} $meta{"it's"} $meta{a\}b}`:  "apostrophe apostrophe brace",
		"$x.y ${x}y $x-y $x_u":                   "X.y Xy X-y ",                      // $x_u is a name of its own, and missing
		`\$x \@x \\ $5 me\@example.com $ @ \n`:   `$x @x \ $5 me@example.com $ @ \n`, // literal
		"[$missing] [$depth{min}] [@nothing]":    "[] [] []",
		"@place":                                 "Cobb, CA", // @ of a plain value: its string
		"$_u":                                    "u",
		"":                                       "",
		"no variables":                           "no variables",
	} {
		if got := doc.Interpolate(tmpl); got != want {
			t.Errorf("Interpolate(%q) = %q, want %q", tmpl, got, want)
		}
	}
	if got := doc.Interpolate("$stations"); !strings.HasPrefix(got, "ARRAY(0x") {
		t.Errorf("$ of an array is its reference, as in Perl: %q", got)
	}

	// Nothing is created by a missing value.
	doc.Interpolate("$missing{deeper}[0] @{absent.list}")
	if doc.HV().Exists("missing") || doc.HV().Exists("absent") {
		t.Error("Interpolate shouldn't create anything")
	}

	// A compiled template, with a list separator; the original keeps its own.
	tm := MustTemplate("$place. Stations: @stations.")
	if got := doc.Interpolate(tm.ListSeparator(", ")); got != "Cobb, CA. Stations: PAS, GSC." {
		t.Errorf("ListSeparator: %q", got)
	}
	if got := doc.Interpolate(tm); got != "Cobb, CA. Stations: PAS GSC." || tm.String() != "$place. Stations: @stations." {
		t.Errorf("the original template changed: %q", got)
	}
	if got := doc.HV().Interpolate("$place"); got != "Cobb, CA" {
		t.Errorf("HV.Interpolate: %q", got)
	}
	holes := Hash("a", Value(Array(1)))
	holes.Get("a").AV().Set(2, 3)
	if got := holes.Interpolate("@a"); got != "1  3" {
		t.Errorf("a hole is empty: %q", got)
	}
}

func TestTemplateErrors(t *testing.T) {
	for tmpl, want := range map[string]string{
		"${place":    "at offset 1: no } to match the {",
		"${}":        "at offset 1: empty ${}",
		"$a[0":       "at offset 2: no ] to match the [",
		"$a{b":       "at offset 2: no } to match the {",
		"x $a[zero]": "at offset 2: bad path",
		"${a..b}":    "at offset 0: bad path",
	} {
		_, err := CompileTemplate(tmpl)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("CompileTemplate(%q): %v, want %q", tmpl, err, want)
		}
		expectPanic(t, tmpl, "bad template", func() { Hash().Interpolate(tmpl) })
	}
	expectPanic(t, "MustTemplate", "bad template", func() { MustTemplate("${") })
	expectPanic(t, "a nil *Template", "nil *Template", func() { Hash().Interpolate((*Template)(nil)) })
	expectPanic(t, "an int", "must be a string or a *Template", func() { Hash().Interpolate(42) })
}
