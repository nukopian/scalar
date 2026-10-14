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
		"content[9].rule_set":      "",  // missing: undef
		"meta.missing.deeper[0].x": "",
	} {
		// A fresh document each time: reading through a missing step
		// autovivifies it, as Path and Perl do, which would change the
		// others' answers.
		if got := atDoc(t).At(path).Str(); got != want {
			t.Errorf("At(%q) = %q, want %q", path, got, want)
		}
	}
	// That autovivification, shown: content[9] now exists, as a hash.
	doc.At("content[9].rule_set")
	if doc.At("content").AV().Length() != 10 || doc.At("content[9]").Ref() != "HASH" {
		t.Error("reading through content[9] should create it, as Path does")
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
	if s.Path("a", "b", 1, "c").Int() != 1 || s.Path("list").Ref() != "ARRAY" ||
		s.Path("list", 0).Str() != "first" || s.Path("h").Ref() != "HASH" || s.Path("h", "0").Str() != "key" {
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
		"a..b":   "expected a name",
		"a.":     "expected a name after .",
		"a.[0]":  "expected a name after ., not [",
		"a[":     "expected ]",
		"a[x]":   "expected an index or a quoted key",
		"a[1.5]": "expected an index or a quoted key",
		`a["x`:   "unterminated quoted key",
		`a["x"`:  "expected ] after the quoted key",
		"a[0]b":  "expected . or [",
		".a":     "expected a name",
	} {
		expectPanic(t, path, want, func() { Scalar(nil).At(path) })
		if _, err := CompilePath(path); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("CompilePath(%q) error = %v, want it to contain %q", path, err, want)
		}
	}
}

func TestPathExpr(t *testing.T) {
	doc := atDoc(t)
	first := MustPath("content[0]")
	if got := doc.Path(first, "tags", 1).Str(); got != "y" {
		t.Errorf("Path(compiled, more keys) = %q", got)
	}
	if v, ok := doc.PathOK(first, "rule_set"); !ok || v.Str() != "A" {
		t.Error("PathOK with a compiled path")
	}
	if _, ok := doc.PathOK(first, "missing"); ok {
		t.Error("PathOK with a compiled path, then a missing key")
	}
	if first.String() != "content[0]" {
		t.Errorf("String = %q", first.String())
	}
	p, err := CompilePath("meta.0")
	if err != nil || doc.Path(p).Str() != "zero" {
		t.Errorf("CompilePath: %v", err)
	}
	expectPanic(t, "MustPath on a bad path", "bad path", func() { MustPath("a..b") })
}
