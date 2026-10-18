package docref

import "testing"

func TestEscapeProse(t *testing.T) {
	for in, want := range map[string]string{
		"my $x = $s":                 `my \$x = \$s`,
		"Perl's $#a = n":             `Perl's \$\#a = n`,
		"as `$x` and $y":             "as `$x` and \\$y", // not in a code span
		"text\n\tindented $code\n":   "text\n\tindented $code\n",
		"```\n$fenced\n```\nafter $": "```\n$fenced\n```\nafter \\$",
		"### A heading\nand # text":  "### A heading\nand \\# text",
		"no specials":                "no specials",
	} {
		if got := escapeProse(in); got != want {
			t.Errorf("escapeProse(%q) = %q, want %q", in, got, want)
		}
	}
}
