package scalar

import (
	"errors"
	"reflect"
	"testing"
)

func TestTo(t *testing.T) {
	doc, err := FromJSON([]byte(`{"title": "Go", "skills": ["Go", "SQL"], "salary": {"max": "55000"},
		"remote": true, "tags": {"go": 3}}`))
	if err != nil {
		t.Fatal(err)
	}

	// Plain values, converted as Decode converts them.
	if got, err := To[string](doc.At("title")); err != nil || got != "Go" {
		t.Errorf("To[string] = %q, %v", got, err)
	}
	if got, err := To[int](doc.At("salary.max")); err != nil || got != 55000 {
		t.Errorf(`To[int] of "55000" = %d, %v`, got, err)
	}
	if got, err := To[bool](doc.At("remote")); err != nil || !got {
		t.Errorf("To[bool] = %v, %v", got, err)
	}

	// Containers: from a scalar holding one, or the array or hash itself.
	want := []string{"Go", "SQL"}
	for _, v := range []any{doc.At("skills"), doc.At("skills").AV(), []any{"Go", "SQL"}} {
		if got, err := To[[]string](v); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("To[[]string](%T) = %q, %v", v, got, err)
		}
	}
	if got, err := To[map[string]int](doc.At("tags").HV()); err != nil || got["go"] != 3 {
		t.Errorf("To[map[string]int] = %v, %v", got, err)
	}
	type posting struct {
		Title  string
		Skills []string
		Salary struct{ Max int }
	}
	p, err := To[posting](doc)
	if err != nil || p.Title != "Go" || len(p.Skills) != 2 || p.Salary.Max != 55000 {
		t.Errorf("To[posting] = %+v, %v", p, err)
	}
	if got, err := To[*posting](doc); err != nil || got == nil || got.Title != "Go" {
		t.Errorf("To[*posting] = %+v, %v", got, err)
	}

	// any: the plain Go form, as Any gives it; *SV: a deep copy.
	if got, err := To[any](doc.At("skills")); err != nil || !reflect.DeepEqual(got, []any{"Go", "SQL"}) {
		t.Errorf("To[any] = %#v, %v", got, err)
	}
	c, err := To[*SV](doc)
	if err != nil || !c.Equal(doc) || c.HV() == doc.HV() {
		t.Errorf("To[*SV] should be an equal deep copy: %v", err)
	}

	// Copies: changing the result doesn't change the source.
	got, _ := To[[]string](doc.At("skills"))
	got[0] = "changed"
	if doc.At("skills[0]").Str() != "Go" {
		t.Error("To's result should share nothing with its source")
	}

	// Errors: the zero value, and a *DecodeError with a path for At.
	n, err := To[int](doc.At("title"))
	var de *DecodeError
	if n != 0 || !errors.As(err, &de) || de.Path != "" || err.Error() != `scalar: Decode: want int, got "Go"` {
		t.Errorf("To[int] of a word = %d, %v", n, err)
	}
	ps, err := To[posting](Value(map[string]any{"Title": "x", "Salary": map[string]any{"Max": "lots"}}))
	if !reflect.DeepEqual(ps, posting{}) || !errors.As(err, &de) || de.Path != "Salary.Max" {
		t.Errorf("a failed struct should come back zero, with the path: %+v, %v", ps, err)
	}

	// Nothing: undef gives the zero value, without an error, as JSON's null does.
	var nilSV *SV
	var nilAV *AV
	var nilHV *HV
	for _, v := range []any{nil, nilSV, nilAV, nilHV, doc.At("missing")} {
		if got, err := To[[]string](v); err != nil || got != nil {
			t.Errorf("To[[]string](%#v) = %q, %v", v, got, err)
		}
	}
	if doc.HV().Exists("missing") {
		t.Error("To shouldn't create anything")
	}
}
