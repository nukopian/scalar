package scalar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"strings"
	"testing"
)

func TestDump(t *testing.T) {
	tags := Array(1, "x")
	doc := Hash("int", 42, "str", "42", "f", 2.5, "big", int64(12345678901), "s", "a\"b$c@d\n",
		"u", nil, "e", Scalar(Array()), "h", Scalar(Hash()), "t", Scalar(tags), "again", Scalar(tags),
		"txt", "caf\u00E9\u263A", "bytes", []byte("\x89P\x00"), "yes", true, "no", false)
	doc.Set("self", Scalar(doc))
	doc.Set("deep", Scalar(Hash("in", Scalar(tags))))
	want := `{
  again => [
    1,
    "x",
  ],
  big => "12345678901",
  bytes => "\211P\0",
  deep => {
    in => $VAR1->{again},
  },
  e => [],
  f => "2.5",
  h => {},
  int => 42,
  no => !!0,
  s => "a\"b\$c\@d\n",
  self => $VAR1,
  str => 42,
  t => $VAR1->{again},
  txt => "caf\x{e9}\x{263a}",
  u => undef,
  yes => !!1,
}`
	if got := doc.Dump(); got != want {
		t.Errorf("Dump:\n%s\nwant:\n%s", got, want)
	}

	for _, c := range []struct{ got, want string }{
		{Array(1, "a", Scalar(Array(2)), Scalar(Hash("k", nil))).String(), `[1,"a",[2],{k => undef}]`},
		{fmt.Sprint(Array("go", 1)), `["go",1]`},
		{fmt.Sprint(Hash("b", 2, "a", 1)), `{a => 1,b => 2}`},
		{Scalar("x").Dump(), `"x"`},
		{Scalar(nil).Dump(), `undef`},
		{(*SV)(nil).Dump(), `undef`},
		{(*AV)(nil).String(), `undef`},
		{Array().Dump(), `[]`},
		{Hash().Dump(), `{}`},
		{Scalar(1e21).Dump() + Scalar(0.1+0.2).Dump() + Scalar("007").Dump(), `"1e+21""0.3""007"`},
		{Scalar("\t\x01x\x011\x7f\u0100\u0080").Dump(), `"\t\1x\0011\177\x{100}\x{80}"`},
		{Scalar([]byte("\x80\xff9\xe9")).Dump(), `"\200\3779\351"`},
		{Scalar("ok\xffbad").Dump(), `"ok\377bad"`}, // not UTF-8: the byte, in octal
	} {
		if c.got != c.want {
			t.Errorf("got %s, want %s", c.got, c.want)
		}
	}

	// Holes print as undef; a loop through an array is its path.
	a := Array(1)
	a.Set(2, 3)
	a.Push(Scalar(a))
	if got := a.String(); got != `[1,undef,3,$VAR1]` {
		t.Errorf("holes and loops: %s", got)
	}
}

// Dump matches Perl's Data::Dumper::Concise (Data::Dumper's pure-Perl
// version, Useperl, with Concise's settings) for
// hundreds of random structures, with shared parts and loops. It needs perl
// on the PATH, and is skipped without it.
func TestDumpMatchesPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	rng := rand.New(rand.NewPCG(3, 5))
	chars := []string{"a", "Z", "0", "1", "9", " ", "-", "\"", "\\", "$", "@", "'", "\t", "\n", "\x00",
		"\x01", "\x1b", "\x7f", "\u00E9", "\u263A", "\u0100", "{", "}"}
	text := func() string {
		if rng.IntN(4) == 0 {
			return []string{"42", "007", "-1", "0", "1234567890", "123456789", "-0", "1.5", ""}[rng.IntN(9)]
		}
		var b strings.Builder
		for range rng.IntN(6) {
			b.WriteString(chars[rng.IntN(len(chars))])
		}
		return b.String()
	}
	// perlText is a Perl expression for a text string: every character
	// escaped, and upgraded, so that Perl treats it as characters.
	perlText := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			fmt.Fprintf(&b, `\x{%x}`, r)
		}
		return fmt.Sprintf(`do { my $s = "%s"; utf8::upgrade($s); $s }`, b.String())
	}

	type tcase struct {
		root *SV
		perl string // statements building $c0
	}
	var cases []tcase
	for range 300 {
		var stmts []string
		var containers []*SV
		var build func(depth int) (*SV, string)
		build = func(depth int) (*SV, string) {
			switch n := rng.IntN(12); {
			case n < 3 && depth < 4 || depth == 0:
				// A container, or now and then one built already: shared,
				// or a loop when it's an ancestor.
				if depth > 0 && len(containers) > 0 && rng.IntN(5) == 0 {
					i := rng.IntN(len(containers))
					return containers[i], fmt.Sprintf("$c%d", i)
				}
				id := len(containers)
				name := fmt.Sprintf("$c%d", id)
				var s *SV
				if rng.IntN(2) == 0 {
					s = Scalar(Array())
					containers = append(containers, s)
					stmts = append(stmts, fmt.Sprintf("my %s = [];", name))
					for range rng.IntN(4) {
						v, expr := build(depth + 1)
						s.AV().Push(v)
						stmts = append(stmts, fmt.Sprintf("push @{%s}, %s;", name, expr))
					}
				} else {
					s = Scalar(Hash())
					containers = append(containers, s)
					stmts = append(stmts, fmt.Sprintf("my %s = {};", name))
					for range rng.IntN(4) {
						k := text()
						v, expr := build(depth + 1)
						s.HV().Set(k, v)
						stmts = append(stmts, fmt.Sprintf("%s->{%s} = %s;", name, perlText(k), expr))
					}
				}
				return s, name
			case n < 5:
				s := text()
				return Scalar(s), perlText(s)
			case n < 6:
				b := []byte(text() + string([]byte{byte(rng.IntN(256))}))
				return Scalar(b), fmt.Sprintf(`pack("H*", "%s")`, hex.EncodeToString(b))
			case n < 8:
				i := []int64{0, 7, -7, 42, 999999999, 1000000000, -12345678901, 9223372036854775807}[rng.IntN(8)]
				return Scalar(i), fmt.Sprint(i)
			case n < 9:
				f := []string{"2.5", "1e21", "0.1", "1.0", "-3.75", "1e-5", "123456789.5"}[rng.IntN(7)]
				var x float64
				fmt.Sscan(f, &x)
				return Scalar(x), f
			case n < 10:
				b := rng.IntN(2) == 0
				return Scalar(b), map[bool]string{true: "!!1", false: "!!0"}[b]
			}
			return Scalar(nil), "undef"
		}
		root, _ := build(0)
		cases = append(cases, tcase{root, strings.Join(stmts, "\n")})
	}

	var script strings.Builder
	script.WriteString(`use v5.36; use Data::Dumper; use JSON::PP;
$Data::Dumper::Terse = 1; $Data::Dumper::Sortkeys = 1; $Data::Dumper::Useqq = 1; $Data::Dumper::Useperl = 1;
$Data::Dumper::Quotekeys = 0; $Data::Dumper::Trailingcomma = 1;
my @out;
`)
	for _, c := range cases {
		fmt.Fprintf(&script, "{\n%s\n$Data::Dumper::Indent = 1; my $one = Dumper($c0); chomp $one;\n"+
			"$Data::Dumper::Indent = 0; push @out, [$one, Dumper($c0)];\n}\n", c.perl)
	}
	script.WriteString("print encode_json(\\@out);\n")

	cmd := exec.Command("perl", "-e", script.String())
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v: %s", err, stderr.String())
	}
	var want [][2]string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("perl's output: %v", err)
	}
	for i, c := range cases {
		if got := c.root.Dump(); got != want[i][0] {
			t.Errorf("case %d, Indent 1:\nGo:\n%s\nPerl:\n%s\nbuilt by:\n%s", i, got, want[i][0], c.perl)
		}
		if got := dump(c.root, false); got != want[i][1] {
			t.Errorf("case %d, Indent 0:\nGo:   %s\nPerl: %s", i, got, want[i][1])
		}
	}
}
