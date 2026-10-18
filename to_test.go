package scalar

import (
	"errors"
	"reflect"
	"testing"
)

func TestTo(t *testing.T) {
	doc, err := FromJSON([]byte(`{"place": "Cobb", "stations": ["PAS", "GSC"], "depth": {"max": "5500"},
		"tsunami": true, "tags": {"ci": 3}}`))
	if err != nil {
		t.Fatal(err)
	}

	// Plain values, converted as Decode converts them.
	if got, err := To[string](doc.At("place")); err != nil || got != "Cobb" {
		t.Errorf("To[string] = %q, %v", got, err)
	}
	if got, err := To[int](doc.At("depth.max")); err != nil || got != 5500 {
		t.Errorf(`To[int] of "5500" = %d, %v`, got, err)
	}
	if got, err := To[bool](doc.At("tsunami")); err != nil || !got {
		t.Errorf("To[bool] = %v, %v", got, err)
	}

	// Containers: from a scalar holding one, or the array or hash itself.
	want := []string{"PAS", "GSC"}
	for _, v := range []any{doc.At("stations"), doc.At("stations").AV(), []any{"PAS", "GSC"}} {
		if got, err := To[[]string](v); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("To[[]string](%T) = %q, %v", v, got, err)
		}
	}
	if got, err := To[map[string]int](doc.At("tags").HV()); err != nil || got["ci"] != 3 {
		t.Errorf("To[map[string]int] = %v, %v", got, err)
	}
	type event struct {
		Place    string
		Stations []string
		Depth    struct{ Max int }
	}
	p, err := To[event](doc)
	if err != nil || p.Place != "Cobb" || len(p.Stations) != 2 || p.Depth.Max != 5500 {
		t.Errorf("To[event] = %+v, %v", p, err)
	}
	if got, err := To[*event](doc); err != nil || got == nil || got.Place != "Cobb" {
		t.Errorf("To[*event] = %+v, %v", got, err)
	}

	// any: the plain Go form, as Any gives it; *SV: a deep copy.
	if got, err := To[any](doc.At("stations")); err != nil || !reflect.DeepEqual(got, []any{"PAS", "GSC"}) {
		t.Errorf("To[any] = %#v, %v", got, err)
	}
	c, err := To[*SV](doc)
	if err != nil || !c.Equal(doc) || c.HV() == doc.HV() {
		t.Errorf("To[*SV] should be an equal deep copy: %v", err)
	}

	// Copies: changing the result doesn't change the source.
	got, _ := To[[]string](doc.At("stations"))
	got[0] = "changed"
	if doc.At("stations[0]").Str() != "PAS" {
		t.Error("To's result should share nothing with its source")
	}

	// Errors: the zero value, and a *DecodeError with a path for At.
	n, err := To[int](doc.At("place"))
	var de *DecodeError
	if n != 0 || !errors.As(err, &de) || de.Path != "" || err.Error() != `scalar: Decode: want int, got "Cobb"` {
		t.Errorf("To[int] of a word = %d, %v", n, err)
	}
	ps, err := To[event](Value(map[string]any{"Place": "x", "Depth": map[string]any{"Max": "lots"}}))
	if !reflect.DeepEqual(ps, event{}) || !errors.As(err, &de) || de.Path != "Depth.Max" {
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
