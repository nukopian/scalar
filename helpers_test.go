package scalar

import (
	"encoding/json"
	"strings"
	"testing"
)

// expectPanic fails unless f panics with a message containing want.
func expectPanic(t *testing.T, name, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Errorf("%s: didn't panic", name)
		} else if !strings.Contains(r.(string), want) {
			t.Errorf("%s: panic %q, want it to contain %q", name, r, want)
		}
	}()
	f()
}

// show renders an array with commas between elements, and holes and undef
// as "-".
func show(a *AV) string { return showSep(a, ",") }

// showSep is show with a chosen separator, for elements containing commas.
func showSep(a *AV, sep string) string {
	parts := make([]string, a.Length())
	for i, s := range a.All() {
		if s.Defined() {
			parts[i] = s.Str()
		} else {
			parts[i] = "-"
		}
	}
	return strings.Join(parts, sep)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
