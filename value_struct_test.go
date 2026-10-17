package scalar

import (
	"reflect"
	"testing"
	"time"
)

type vsInner struct {
	Code  string `json:"code"`
	Title string `json:"title,omitempty"`
}

type vsBase struct {
	ID string `json:"id"`
}

type vsOuter struct {
	vsBase                    // promoted: "id"
	Name       string         `json:"name"`
	Skip       string         `json:"-"`
	Empty      string         `json:"empty,omitempty"`
	Zero       int            `json:"zero"`
	Ptr        *float64       `json:"ptr,omitempty"`
	NilPtr     *float64       `json:"nilPtr"`
	Groups     []vsInner      `json:"groups"`
	ByName     map[string]int `json:"byName,omitempty"`
	When       time.Time      `json:"when"`
	Untagged   bool
	unexported string
}

// A struct is a hash, as encoding/json would encode it (found dogfooding
// cmd/grammarrules, 2026-10-17: Value made a struct its fmt.Sprint).
func TestValueStruct(t *testing.T) {
	p := 3.5
	x := vsOuter{vsBase: vsBase{"a1"}, Name: "n", Skip: "s", Ptr: &p, Groups: []vsInner{{Code: "2", Title: "t"}, {Code: "21"}},
		When: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC), Untagged: true, unexported: "u"}
	v := Value(x)
	if v.Kind() != HashRef {
		t.Fatalf("Value(struct) is %v: %q", v.Kind(), v.Str())
	}
	keys := sortedKeys(v.HV())
	want := []string{"Untagged", "groups", "id", "name", "nilPtr", "ptr", "when", "zero"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys %q, want %q", keys, want)
	}
	if v.At("groups[1].code").Str() != "21" || v.At("groups[1].title").Defined() || v.At("nilPtr").Defined() ||
		v.At("ptr").Num() != 3.5 || v.At("when").Str() != "2026-10-17T09:00:00Z" || v.At("zero").Int() != 0 {
		t.Errorf("Value(struct) = %s", v.Dump())
	}
	// And back again, by Decode.
	var y vsOuter
	if err := v.Decode(&y); err != nil {
		t.Fatal(err)
	}
	x.Skip, x.unexported = "", ""
	if !reflect.DeepEqual(x, y) {
		t.Errorf("round trip: %+v, want %+v", y, x)
	}
	// Named basic types are what they're named for, as in JSON.
	type level int
	type kind string
	type ratio float64
	for in, want := range map[any]Kind{level(3): Int, kind("stated"): String, ratio(0.5): Float} {
		if got := Value(in).Kind(); got != want {
			t.Errorf("Value(%T) is %v, want %v", in, got, want)
		}
	}
	// A pointer to a struct is the struct; a nil one undef.
	if Value(&x).Kind() != HashRef || Value((*vsOuter)(nil)).Defined() {
		t.Error("pointers to structs")
	}
}
