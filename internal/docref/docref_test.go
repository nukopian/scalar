package docref

import "testing"

func TestEscapeDollars(t *testing.T) {
	for in, want := range map[string]string{
		"my $x = $s":                 `my \$x = \$s`,
		"as `$x` and $y":             "as `$x` and \\$y", // not in a code span
		"text\n\tindented $code\n":   "text\n\tindented $code\n",
		"```\n$fenced\n```\nafter $": "```\n$fenced\n```\nafter \\$",
		"no dollars":                 "no dollars",
	} {
		if got := escapeDollars(in); got != want {
			t.Errorf("escapeDollars(%q) = %q, want %q", in, got, want)
		}
	}
}
