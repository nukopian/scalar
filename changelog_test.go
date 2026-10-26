package scalar

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestChangelogUnreleased: CHANGELOG.md's Unreleased section holds
// either exactly "Nothing yet." or entries, never both and never nothing.
// A release leaves "Nothing yet." under a new Unreleased heading, and the
// first change after it replaces that line (docs/MAINTAINING.md, step 2).
func TestChangelogUnreleased(t *testing.T) {
	data, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkUnreleased(string(data)); err != nil {
		t.Error(err)
	}
}

// checkUnreleased says what's wrong with a changelog's Unreleased section.
func checkUnreleased(changelog string) error {
	_, rest, ok := strings.Cut(changelog, "\n## Unreleased\n")
	if !ok {
		return errors.New("CHANGELOG.md has no ## Unreleased heading")
	}
	section, _, _ := strings.Cut(rest, "\n## ")
	section = strings.TrimSpace(section)
	nothing := strings.Contains(section, "Nothing yet.")
	entries := strings.HasPrefix(section, "- ") || strings.Contains(section, "\n- ")
	switch {
	case section == "":
		return errors.New(`Unreleased is empty: a release leaves "Nothing yet." under it`)
	case nothing && entries:
		return errors.New(`Unreleased has entries and "Nothing yet.": the first entry replaces that line`)
	case nothing && section != "Nothing yet.":
		return fmt.Errorf("Unreleased holds %q: it's either \"Nothing yet.\" or entries", section)
	case !nothing && !entries:
		return fmt.Errorf("Unreleased holds %q, which isn't entries", section)
	}
	return nil
}

// TestCheckUnreleased: the check passes what a release and a first change
// leave, and catches each way of getting them wrong.
func TestCheckUnreleased(t *testing.T) {
	const head = "# Changes\n\nintro\n\n## Unreleased\n\n"
	const tail = "\n\n## v0.6.0 (2026-10-26)\n\n- a change\n"
	for name, c := range map[string]struct {
		section string
		ok      bool
	}{
		"after a release":         {"Nothing yet.", true},
		"after a first change":    {"- Fixed: something.", true},
		"several changes":         {"- One.\n- Two,\n  wrapped.", true},
		"forgotten at a release":  {"", false},
		"not replaced by a first": {"Nothing yet.\n\n- Fixed: something.", false},
		"something else":          {"Coming soon.", false},
	} {
		err := checkUnreleased(head + c.section + tail)
		if (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := checkUnreleased("# Changes\n\n## v0.6.0\n"); err == nil {
		t.Error("no Unreleased heading: should fail")
	}
}
