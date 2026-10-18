package scalar

import (
	"encoding/json"
	"os/exec"
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

// perlUnusable says why the Perl differential tests can't run here, or ""
// if they can: they want perl 5.36 or later (macOS and Windows ship older
// ones), and go test -short skips them.
func perlUnusable() string {
	switch {
	case testing.Short():
		return "skipped with -short"
	case exec.Command("perl", "-e", "use v5.36").Run() != nil:
		if _, err := exec.LookPath("perl"); err != nil {
			return "perl isn't installed"
		}
		return "perl is older than 5.36"
	}
	return ""
}

// needPerl skips a Perl differential test that can't run here.
func needPerl(t *testing.T) {
	t.Helper()
	if why := perlUnusable(); why != "" {
		t.Skip(why)
	}
}
