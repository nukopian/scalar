package scalar

import (
	"strings"
	"testing"
)

func TestInterpolate(t *testing.T) {
	doc, err := FromJSON([]byte(`{
		"title": "Go Developer", "location": "Leeds", "skills": ["Go", "SQL"],
		"salary": {"max": 55000, "min": null}, "content": [{"rule_set": "A"}],
		"meta": {"log.level": "debug", "it's": "apostrophe", "a}b": "brace"}, "x": "X", "_u": "u"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	for tmpl, want := range map[string]string{
		"$title in $location: @skills":           "Go Developer in Leeds: Go SQL",
		"Up to $salary{max}, from ${salary.max}": "Up to 55000, from 55000",
		"$skills[1] $skills[-1] @{skills}":       "SQL SQL Go SQL",
		"$content[0]{rule_set}":                  "A",
		"${content[0].rule_set}":                 "A",
		"$meta{log.level} ${meta{log.level}}":    "debug debug",
		`$meta{it's} $meta{"it's"} $meta{a\}b}`:  "apostrophe apostrophe brace",
		"$x.y ${x}y $x-y $x_u":                   "X.y Xy X-y ",                      // $x_u is a name of its own, and missing
		`\$x \@x \\ $5 me\@example.com $ @ \n`:   `$x @x \ $5 me@example.com $ @ \n`, // literal
		"[$missing] [$salary{min}] [@nothing]":   "[] [] []",
		"@title":                                 "Go Developer", // @ of a plain value: its string
		"$_u":                                    "u",
		"":                                       "",
		"no variables":                           "no variables",
	} {
		if got := doc.Interpolate(tmpl); got != want {
			t.Errorf("Interpolate(%q) = %q, want %q", tmpl, got, want)
		}
	}
	if got := doc.Interpolate("$skills"); !strings.HasPrefix(got, "ARRAY(0x") {
		t.Errorf("$ of an array is its reference, as in Perl: %q", got)
	}

	// Nothing is created by a missing value.
	doc.Interpolate("$missing{deeper}[0] @{absent.list}")
	if doc.HV().Exists("missing") || doc.HV().Exists("absent") {
		t.Error("Interpolate shouldn't create anything")
	}

	// A compiled template, with a list separator; the original keeps its own.
	tm := MustTemplate("$title. Skills: @skills.")
	if got := doc.Interpolate(tm.ListSeparator(", ")); got != "Go Developer. Skills: Go, SQL." {
		t.Errorf("ListSeparator: %q", got)
	}
	if got := doc.Interpolate(tm); got != "Go Developer. Skills: Go SQL." || tm.String() != "$title. Skills: @skills." {
		t.Errorf("the original template changed: %q", got)
	}
	if got := doc.HV().Interpolate("$title"); got != "Go Developer" {
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
		"${title":    "at offset 1: no } to match the {",
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
