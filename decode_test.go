package scalar

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type upper string

func (u *upper) UnmarshalText(b []byte) error {
	*u = upper(strings.ToUpper(string(b)))
	return nil
}

type Base struct {
	ID   int `json:"id"`
	Note string
}

type Extra struct{ Source string }

type posting struct {
	Base
	*Extra
	Title    string            `json:"title"`
	Skills   []string          `json:"skills"`
	Salary   *salary           `json:"salary"`
	Remote   bool              `json:"remote"`
	Tags     map[string]int    `json:"tags"`
	ByYear   map[int]string    `json:"byYear"`
	Pair     [2]float64        `json:"pair"`
	Loose    *SV               `json:"loose"`
	List     *AV               `json:"list"`
	Meta     HV                `json:"meta"`
	Any      any               `json:"any"`
	Posted   time.Time         `json:"posted"`
	Code     upper             `json:"code"`
	Blob     []byte            `json:"blob"`
	Ignored  string            `json:"-"`
	Pointers []*int            `json:"pointers"`
	Nested   map[string][]bool `json:"nested"`
	private  string
}

type salary struct {
	Min, Max int
	Period   string `json:"period"`
}

func TestDecode(t *testing.T) {
	doc, err := FromJSON([]byte(`{
		"id": "7", "note": "case-insensitive", "Source": "cv-library",
		"title": "Go Developer", "skills": ["Go", "SQL"],
		"salary": {"min": "45000", "MAX": 55000.0, "period": "year"},
		"remote": "true", "tags": {"go": 3}, "byYear": {"2026": "now"},
		"pair": [1, "2.5", 99], "loose": {"x": [1]}, "list": ["a"], "meta": {"k": "v"},
		"any": {"n": 1}, "posted": "2026-10-14T09:00:00Z", "code": "ab",
		"blob": "aGk=", "Ignored": "no", "pointers": [1, null], "nested": {"a": [1, 0, false]},
		"private": "no", "unknown": "ignored"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var p posting
	if err := doc.Decode(&p); err != nil {
		t.Fatal(err)
	}
	one := 1
	want := posting{
		Base: Base{ID: 7, Note: "case-insensitive"}, Extra: &Extra{Source: "cv-library"},
		Title: "Go Developer", Skills: []string{"Go", "SQL"},
		Salary: &salary{Min: 45000, Max: 55000, Period: "year"}, Remote: true,
		Tags: map[string]int{"go": 3}, ByYear: map[int]string{2026: "now"}, Pair: [2]float64{1, 2.5},
		Any: map[string]any{"n": int64(1)}, Posted: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
		Code: "AB", Blob: []byte("hi"), Pointers: []*int{&one, nil}, Nested: map[string][]bool{"a": {true, false, false}},
	}
	got := p
	got.Loose, got.List, got.Meta = nil, nil, HV{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Decode:\n got %+v\nwant %+v", got, want)
	}

	if p.private != "" || p.Ignored != "" {
		t.Error(`unexported fields, and fields tagged "-", should be left alone`)
	}

	// The loose parts are deep copies.
	if p.Loose.At("x[0]").Int() != 1 || p.List.Get(0).Str() != "a" || p.Meta.Get("k").Str() != "v" {
		t.Errorf("loose parts: %s %s %s", p.Loose.Dump(), p.List, &p.Meta)
	}
	p.Loose.At("x").AV().Push(2)
	if doc.At("loose.x").AV().Length() != 1 {
		t.Error("a *SV field should be a deep copy")
	}

	// A byte string fills a []byte as it is.
	var b struct{ Blob []byte }
	if err := Hash("Blob", []byte{0x89, 'P'}).decodeVia(&b); err != nil || string(b.Blob) != "\x89P" {
		t.Errorf("bytes: %q %v", b.Blob, err)
	}

	// Undef: nil for pointers, slices and maps; anything else left alone.
	q := posting{Title: "kept", Skills: []string{"x"}, Salary: &salary{}, Tags: map[string]int{}}
	if err := Hash("title", nil, "skills", nil, "salary", nil, "tags", nil, "list", nil).decodeVia(&q); err != nil {
		t.Fatal(err)
	}
	if q.Title != "kept" || q.Skills != nil || q.Salary != nil || q.Tags != nil || q.List != nil {
		t.Errorf("undef: %+v", q)
	}
}

// decodeVia decodes a hash through a scalar holding it.
func (h *HV) decodeVia(v any) error { return Scalar(h).Decode(v) }

func TestDecodeErrors(t *testing.T) {
	type target struct {
		Salary struct {
			Max int `json:"max"`
		} `json:"salary"`
		Skills []int8          `json:"skills"`
		Meta   map[string]bool `json:"meta"`
		Count  uint            `json:"count"`
		Rate   float32         `json:"rate"`
		Name   string          `json:"name"`
		Year   map[int]string  `json:"year"`
		Fn     func()          `json:"fn"`
		Blob   []byte          `json:"blob"`
	}
	for js, want := range map[string]string{
		`{"salary": {"max": "competitive"}}`: `salary.max: want int, got "competitive"`,
		`{"salary": {"max": "45,000"}}`:      `salary.max: want int, got "45,000"`,
		`{"salary": {"max": 2.5}}`:           `salary.max: want int, got 2.5`,
		`{"salary": {"max": ""}}`:            `salary.max: want int, got ""`,
		`{"salary": [1]}`:                    `salary: want a hash for struct { Max int "json:\"max\"" }, got an array`,
		`{"skills": [1, 2, 300]}`:            `skills[2]: want int8, got 300`,
		`{"skills": "Go"}`:                   `skills: want an array, got "Go"`,
		`{"meta": {"log.level": "yes"}}`:     `meta{log.level}: want bool, got "yes"`,
		`{"meta": {"ok": 2}}`:                `meta.ok: want bool, got 2`,
		`{"salary": {"max": true}}`:          `salary.max: want int, got true`,
		`{"name": false}`:                    `name: want string, got false`,
		`{"rate": true}`:                     `rate: want float32, got true`,
		`{"count": -1}`:                      `count: want uint, got -1`,
		`{"rate": 1e300}`:                    `rate: want float32, got 1e+300`,
		`{"name": {"a": 1}}`:                 `name: want string, got a hash`,
		`{"year": {"twenty": "x"}}`:          `year: want int keys, got "twenty"`,
		`{"fn": 1}`:                          `fn: want func(), which Decode can't fill, got 1`,
		`{"blob": "not base64!"}`:            `blob: want base64 text for []byte, got "not base64!"`,
	} {
		doc, err := FromJSON([]byte(js))
		if err != nil {
			t.Fatal(err)
		}
		var v target
		err = doc.Decode(&v)
		var de *DecodeError
		if !errors.As(err, &de) || err.Error() != "scalar: Decode: "+want {
			t.Errorf("%s: %v\n want %s", js, err, want)
			continue
		}
		// The path leads At to the value.
		if _, ok := doc.AtOK(de.Path); !ok && !strings.Contains(want, "keys") {
			t.Errorf("%s: path %q doesn't lead to the value", js, de.Path)
		}
	}

	var n int
	for _, v := range []any{n, nil, (*int)(nil)} {
		if err := Scalar(1).Decode(v); err == nil || !strings.Contains(err.Error(), "non-nil pointer") {
			t.Errorf("Decode(%#v): %v", v, err)
		}
	}
	if err := Scalar("x").Decode(&n); err == nil || err.Error() != `scalar: Decode: want int, got "x"` {
		t.Errorf("at the top: %v", err)
	}

	// Several values that don't fit: always the first in sorted key order.
	type two struct{ A, B int }
	for range 20 {
		var v two
		if err := Hash("b", "x", "a", "y").decodeVia(&v); err == nil || err.Error() != `scalar: Decode: a: want int, got "y"` {
			t.Fatalf("with two bad values: %v", err)
		}
	}

	// A loop can't go into an any.
	loop := Hash()
	loop.Set("self", Scalar(loop))
	var x any
	if err := Scalar(loop).Decode(&x); err == nil {
		t.Error("a loop into an any should be an error")
	}
}

// On data that's already the right shape, Decode agrees with encoding/json.
func TestDecodeMatchesJSON(t *testing.T) {
	js := `{"id": 3, "note": "n", "Source": "s", "title": "t", "skills": ["a", "b"],
		"salary": {"Min": 1, "Max": 2, "period": "p"}, "remote": true, "tags": {"x": 1},
		"byYear": {"1999": "y"}, "pair": [0.5, 1], "posted": "2026-01-02T03:04:05Z",
		"code": "q", "blob": "aGk=", "pointers": [5], "nested": {"k": [true]}, "any": [1, "two", null]}`
	var want, got posting
	if err := json.Unmarshal([]byte(js), &want); err != nil {
		t.Fatal(err)
	}
	doc, _ := FromJSON([]byte(js))
	if err := doc.Decode(&got); err != nil {
		t.Fatal(err)
	}
	// encoding/json gives float64 for numbers in an any; scalar keeps
	// integers as int64.
	want.Any = []any{int64(1), "two", nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Decode and encoding/json differ:\nDecode %+v\njson   %+v", got, want)
	}
}
