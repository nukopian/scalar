package scalar

import (
	"strings"
	"testing"
)

func atDoc(t *testing.T) *SV {
	t.Helper()
	doc, err := FromJSON([]byte(`{
		"content": [{"rule_set": "A", "tags": ["x", "y"]}, {"rule_set": "B"}],
		"meta": {"log.level": "debug", "0": "zero", "say \"hi\"": 1, "n": null}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestAt(t *testing.T) {
	doc := atDoc(t)
	for path, want := range map[string]string{
		"content[0].rule_set":      "A",
		"content.0.rule_set":       "A", // a number after a dot, into an array
		"content[-1].rule_set":     "B",
		"content.-1.rule_set":      "B",
		"content[0].tags[1]":       "y",
		"content.0.tags.1":         "y",
		`meta["log.level"]`:        "debug",
		`meta['log.level']`:        "debug",
		"meta.0":                   "zero", // a number after a dot, into a hash: the key
		`meta["0"]`:                "zero",
		`meta["say \"hi\""]`:       "1",
		"$.content[1].rule_set":    "B", // JSONPath style
		"{content}[0]{rule_set}":   "A", // Perl style
		"content[0]{tags}[-1]":     "y", // mixed
		"{content}.0.rule_set":     "A",
		"meta{log.level}":          "debug", // braces need no quotes
		"meta{0}":                  "zero",  // always a key, as in Perl
		`meta{"0"}`:                "zero",
		`meta{'say "hi"'}`:         "1",
		`meta{say "hi"}`:           "1", // spaces and quotes inside are the key
		"${meta}{n}":               "",
		"content[9].rule_set":      "", // missing: undef
		"meta.missing.deeper[0].x": "",
	} {
		// A fresh document each time: reading through a missing step
		// autovivifies it, as Get and Perl do, which would change the
		// others' answers.
		if got := atDoc(t).At(path).Str(); got != want {
			t.Errorf("At(%q) = %q, want %q", path, got, want)
		}
	}
	// That autovivification, shown: content[9] now exists, as a hash.
	doc.At("content[9].rule_set")
	if doc.At("content").AV().Length() != 10 || doc.At("content[9]").Ref() != "HASH" {
		t.Error("reading through content[9] should create it, as Get does")
	}
	if doc.At("") != doc || doc.At("$") != doc {
		t.Error("an empty path, or $ alone, is the scalar itself")
	}
	if !doc.At("content").Defined() || doc.At("content").Ref() != "ARRAY" {
		t.Error("At(content) should be the array")
	}

	// Writing autovivifies: names make hashes, indices make arrays, and a
	// number after a dot makes an array.
	var s SV
	s.At("a.b[1].c").Set(1)
	s.At("list.0").Set("first")
	s.At(`h["0"]`).Set("key")
	s.At("{g}{0}[0]").Set("braced") // {0} makes a hash, as $s->{g}{0}[0] does
	if s.Get("a", "b", 1, "c").Int() != 1 || s.Get("list").Ref() != "ARRAY" ||
		s.Get("list", 0).Str() != "first" || s.Get("h").Ref() != "HASH" || s.Get("h", "0").Str() != "key" ||
		s.Get("g").Ref() != "HASH" || s.Get("g", "0", 0).Str() != "braced" {
		out, _ := s.MarshalJSON()
		t.Errorf("after writing: %s", out)
	}
}

func TestAtOK(t *testing.T) {
	doc := atDoc(t)
	if v, ok := doc.AtOK("meta.n"); !ok || v.Defined() {
		t.Error("a null is present: ok true, value undef")
	}
	for _, path := range []string{"meta.missing", "content[9]", "content[0].rule_set.x", `content["0"]`, "meta[0]"} {
		if _, ok := doc.AtOK(path); ok {
			t.Errorf("AtOK(%q) should be false", path)
		}
	}
	if v, ok := doc.AtOK("content.0.tags[-1]"); !ok || v.Str() != "y" {
		t.Error("AtOK through an array by a dotted number")
	}
	var s SV
	if _, ok := s.AtOK("a.b"); ok || s.Defined() {
		t.Error("AtOK should never create anything")
	}
	var nilSV *SV
	if _, ok := nilSV.AtOK("a"); ok {
		t.Error("AtOK on nil should be false")
	}
}

func TestAtErrors(t *testing.T) {
	for path, want := range map[string]string{
		"a..b":    "expected a name",
		"a.":      "expected a name after .",
		"a.[0]":   "expected a name after ., not [",
		"a[":      "expected ]",
		"a[x]":    "expected an index or a quoted key",
		"a[1.5]":  "expected an index or a quoted key",
		`a["x`:    "unterminated quoted key",
		`a["x"`:   "expected ] after the quoted key",
		"a[0]b":   "expected ., [ or {",
		"a.{b}":   "expected a name after ., not {",
		"a{":      "expected }",
		"a{b":     "expected }",
		"a{}":     "expected a key in {...}",
		`a{"b"x}`: "expected } after the quoted key",
		"a{b}c":   "expected ., [ or {",
		".a":      "expected a name",
	} {
		expectPanic(t, path, want, func() { Value(nil).At(path) })
		if err := CheckPath(path); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("CheckPath(%q) error = %v, want it to contain %q", path, err, want)
		}
	}
	for _, path := range []string{"", "$", "content[0].rule_set", "{content}[0]{rule_set}", `meta["log.level"]`, "items[-1].name", "meta.0"} {
		if err := CheckPath(path); err != nil {
			t.Errorf("CheckPath(%q) = %v", path, err)
		}
	}
}

func TestDeleteAt(t *testing.T) {
	doc := atDoc(t)
	if got := doc.DeleteAt("content[0].rule_set"); got.Str() != "A" {
		t.Errorf("DeleteAt returned %q", got.Str())
	}
	if doc.At("content[0]").HV().Exists("rule_set") {
		t.Error("the key should be gone")
	}
	if got := doc.DeleteAt(`meta{log.level}`); got.Str() != "debug" || doc.At("meta").HV().Exists("log.level") {
		t.Error("braces: a key with a dot")
	}
	if got := doc.DeleteAt("meta.0"); got.Str() != "zero" {
		t.Error(".0 into a hash deletes the key")
	}

	// Arrays: the last element shrinks the array; one before it leaves a hole.
	if got := doc.DeleteAt("content[0].tags[-1]"); got.Str() != "y" || doc.At("content[0].tags").AV().Length() != 1 {
		t.Error("deleting the last element should shrink the array")
	}
	doc.At("list").AV().Push(1, 2, 3)
	doc.DeleteAt("list.0")
	if a := doc.At("list").AV(); a.Length() != 3 || a.Exists(0) {
		t.Error("deleting an element before the end should leave a hole")
	}

	// Nothing to delete: nothing created, and undef returned.
	before, _ := doc.MarshalJSON()
	for _, path := range []string{"missing.deeper[0]", "meta.missing", "content[9].x", "meta[0]",
		`content["0"]`, "content[0].rule_set.x", "meta.n.x", ""} {
		if got := doc.DeleteAt(path); got.Defined() {
			t.Errorf("DeleteAt(%q) = %q, want undef", path, got.Str())
		}
	}
	if after, _ := doc.MarshalJSON(); string(after) != string(before) {
		t.Errorf("DeleteAt of missing paths changed the document:\n%s\n%s", before, after)
	}
	if got := doc.DeleteAt("meta.n"); got.Defined() || doc.At("meta").HV().Exists("n") {
		t.Error("deleting a null removes the key and returns undef")
	}
	expectPanic(t, "a malformed path", "bad path", func() { doc.DeleteAt("a..b") })
}
