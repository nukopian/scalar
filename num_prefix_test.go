package scalar

import (
	"regexp"
	"testing"
)

// oldNumPrefix is the regex numericPrefix replaced (2026-10-17, for speed:
// 185 ns a Num on a string, most of it the regex): the oracle it must
// agree with.
var oldNumPrefix = regexp.MustCompile(`^[+-]?(?:(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?|(?i:inf(?:inity)?|nan))`)

var numPrefixCases = []string{
	"", "0", "42", "-42", "+42", "4.1", "4.", ".5", ".", "+", "-", "+.", "1e5", "1e", "1e+", "1e+5x", "1E-3",
	"12abc", "abc", "  7", "x", "-.5e2", "1.2.3", "007", "inf", "Inf", "-INF", "infinity", "Infinityx", "infin",
	"nan", "NaN", "-nan", "nanx", "in", "na", "e5", "+-1", "0x1A", "1_000", "١٢", "9223372036854775808",
	"0 but true", "12,500", "≈45", "56914", "0.5e", "5e5e5", "-0", ".e5", "1.e5",
}

func TestNumericPrefix(t *testing.T) {
	for _, s := range numPrefixCases {
		if got, want := numericPrefix(s), oldNumPrefix.FindString(s); got != want {
			t.Errorf("numericPrefix(%q) = %q, want %q (the regex's)", s, got, want)
		}
	}
}

func FuzzNumericPrefix(f *testing.F) {
	for _, s := range numPrefixCases {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := numericPrefix(s), oldNumPrefix.FindString(s); got != want {
			t.Errorf("numericPrefix(%q) = %q, want %q (the regex's)", s, got, want)
		}
	})
}
