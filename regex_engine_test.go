package scalar

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/tabwriter"
)

// EXPERIMENTAL: tests and benchmarks for the regex engine seam.

func init() {
	benchEngines["regexp2-seam"] = func(p string) *Regexp { return MustCompileWith(regexp2Engine{}, p) }
	benchEngines["stdlib"] = func(p string) *Regexp { return MustCompileWith(Stdlib, p) }
}

// TestRegexp2ThroughSeam runs the regex tests with every pattern string
// compiled by regexp2 through the seam, so it checks that the seam's own
// work (positions, expansion, splitting) does what the default path does.
func TestRegexp2ThroughSeam(t *testing.T) {
	defaultEngine = regexp2Engine{}
	defer func() { defaultEngine = nil }()
	for name, test := range map[string]func(*testing.T){
		"MatchAsCondition":          TestMatchAsCondition,
		"Match":                     TestMatch,
		"MatchGlobal":               TestMatchGlobal,
		"Named":                     TestNamed,
		"Subst":                     TestSubst,
		"Split":                     TestSplit,
		"RegexPanics":               TestRegexPanics,
		"RegexTimeout":              TestRegexTimeout,
		"SubstPlainReplacement":     TestSubstPlainReplacement,
		"SubstExpandMatchesRegexp2": TestSubstExpandMatchesRegexp2,
		"SubstExpandMatchesPerl":    TestSubstExpandMatchesPerl,
		"SubstExpandLeftAlone":      TestSubstExpandLeftAlone,
	} {
		t.Run(name, test)
	}
}

// An engineCase is one regex operation, run through Perl and each engine.
type engineCase struct {
	Op       string `json:"op"` // M, Mg, Named, S, Split
	Pattern  string `json:"pattern"`
	Flags    string `json:"flags"`
	Input    string `json:"input"`
	Bytes    bool   `json:"bytes"` // Input is a byte string
	Repl     string `json:"repl"`
	PerlRepl string `json:"perlRepl"` // Repl as Perl spells it, if different
	Note     string `json:"-"`
}

var engineCases = []engineCase{
	// The everyday: what most scalar code does.
	{Op: "M", Pattern: `^M (\d+\.\d) - (\d+) km (\w+) of ([^,]+), (\w\w)`, Input: quakeLine, Note: "captures"},
	{Op: "M", Pattern: `(a)(x)?(b)`, Input: "ab", Note: "group that didn't take part"},
	{Op: "M", Pattern: `offshore`, Flags: "i", Input: "OFFSHORE Northern California", Note: "i"},
	{Op: "Mg", Pattern: `depth (\d+(?:\.\d+)?) km`, Input: quakeLine + "\n" + quakeLine, Note: "g"},
	{Op: "Mg", Pattern: `(\w+) (\d+)\.(\d+)`, Input: "Go 1.27, Perl 5.40", Note: "g with groups"},
	{Op: "M", Pattern: `^b$`, Flags: "m", Input: "a\nb", Note: "m"},
	{Op: "M", Pattern: `a.b`, Flags: "s", Input: "a\nb", Note: "s"},
	{Op: "M", Pattern: `.+?,`, Input: "Ridgecrest, CA, USA", Note: "lazy"},
	{Op: "M", Pattern: `a|ab`, Input: "ab", Note: "alternation is leftmost-first"},
	{Op: "M", Pattern: `\p{Lu}\p{Ll}+`, Input: "near Ōtaki", Note: `\p{...}`},
	{Op: "M", Pattern: `[[:alpha:]]+`, Input: "123abc", Note: "POSIX class"},
	{Op: "Named", Pattern: `net (?<net>\w+) \| code (?<code>\d+)`, Input: quakeLine, Note: "named, (?<n>)"},
	{Op: "Named", Pattern: `net (?P<net>\w+)`, Input: quakeLine, Note: "named, (?P<n>)"},
	{Op: "S", Pattern: `(\d+) km`, Flags: "g", Input: quakeLine, Repl: "${1}km", Note: "${1}"},
	{Op: "S", Pattern: `(?<mag>\d\.\d)`, Input: quakeLine, Repl: "[${mag}]", PerlRepl: "[$+{mag}]", Note: "${name}"},
	{Op: "S", Pattern: `CA`, Input: quakeLine, Repl: "<$&>", Note: "$&"},
	{Op: "S", Pattern: `x`, Flags: "g", Input: "no match", Repl: "y", Note: "no match"},
	{Op: "Split", Pattern: `\s*\|\s*`, Input: quakeLine, Note: "split"},
	{Op: "Split", Pattern: `(,)`, Input: "ci,us,ak", Note: "split keeps groups"},
	{Op: "Split", Pattern: ``, Input: "abc", Note: "split on empty pattern"},
	{Op: "Split", Pattern: ` `, Input: "  a b  c ", Note: "awk mode"},
	{Op: "Split", Pattern: `,`, Input: "a,b,,,", Note: "trailing empties dropped"},

	// Unicode: Perl's classes are Unicode.
	{Op: "M", Pattern: `\w+`, Input: "Zürich", Note: `\w on ü`},
	{Op: "M", Pattern: `\d+`, Input: "depth ٣٤ km", Note: `\d on Arabic-Indic digits`},
	{Op: "M", Pattern: `\s`, Input: "a b", Note: `\s on no-break space`},
	{Op: "Mg", Pattern: `\b\w`, Input: "été ici", Note: `\b next to é`},
	{Op: "M", Pattern: `ÉTÉ`, Flags: "i", Input: "été", Note: "i on é"},
	{Op: "M", Pattern: `strasse`, Flags: "i", Input: "Straße", Note: "i, ß as ss"},

	// Ends of lines.
	{Op: "M", Pattern: `CA$`, Input: "Ridgecrest, CA\n", Note: "$ before a final newline"},
	{Op: "M", Pattern: `CA\Z`, Input: "Ridgecrest, CA\n", Note: `\Z`},
	{Op: "M", Pattern: `CA\z`, Input: "Ridgecrest, CA\n", Note: `\z`},

	// Empty matches with g.
	{Op: "Mg", Pattern: `\d*`, Input: "a1", Note: "empty match after a match"},
	{Op: "S", Pattern: `x*`, Flags: "g", Input: "abc", Repl: "-", Note: "s/x*/-/g"},

	// What RE2 leaves out.
	{Op: "M", Pattern: `(?<=\$)\d+`, Input: "price: $100", Note: "lookbehind"},
	{Op: "M", Pattern: `^(?!.*\.exe$).*$`, Input: "foo.txt", Note: "negative lookahead"},
	{Op: "M", Pattern: `\b(\w+) \1\b`, Input: "hello hello", Note: "backreference"},
	{Op: "M", Pattern: `(?>a+)b`, Input: "aaab", Note: "atomic group"},
	{Op: "M", Pattern: `a++b`, Input: "aaab", Note: "possessive"},
	{Op: "M", Pattern: `[a-z]+ \d+  # letters then digits`, Flags: "x", Input: "abc123", Note: "x"},
	{Op: "M", Pattern: `\h+`, Input: "a \tb", Note: `\h`},
	{Op: "M", Pattern: `(?<y>\d+)-(\d+)`, Input: "2026-10", Note: "named and unnamed numbering"},

	// Byte strings: characters 0 to 255.
	{Op: "M", Pattern: `^\x89(PNG)`, Input: "\u0089PNG\r\n", Bytes: true, Note: `bytes, \x89`},
	{Op: "M", Pattern: `\w+`, Input: "\u00e9t\u00e9", Bytes: true, Note: `bytes, \w on \xe9`},
}

// perlEngineScript runs engineCases, read as JSON, and prints each result
// as JSON in the form engineResult gives.
const perlEngineScript = `
use v5.36;
use JSON::PP;
my $json = JSON::PP->new->utf8->canonical->allow_nonref;
my $cases = $json->decode(do { local $/; <STDIN> });
sub groups ($s) { [ map { defined $-[$_] ? substr($s, $-[$_], $+[$_] - $-[$_]) : undef } 0 .. $#+ ] }
for my $c (@$cases) {
	my $s = $c->{input};
	if ($c->{bytes}) { utf8::downgrade($s) }
	my $out = eval {
		my $f = $c->{flags} =~ s/[gr]//gr;
		my $re = $c->{op} eq 'Split' && $c->{pattern} eq ' ' ? ' '
			: $f ne '' ? qr/(?$f)$c->{pattern}/ : qr/$c->{pattern}/;
		my $op = $c->{op};
		if ($op eq 'M') { $s =~ $re ? groups($s) : undef }
		elsif ($op eq 'Mg') { my @all; push @all, groups($s) while $s =~ /$re/g; @all ? \@all : undef }
		elsif ($op eq 'Named') { $s =~ $re ? { map { ($_ => $+{$_}) } keys %- } : undef }
		elsif ($op eq 'S') {
			my $r = '"' . ($c->{perlRepl} || $c->{repl}) . '"';
			my $n = $c->{flags} =~ /g/ ? ($s =~ s/$re/$r/gee) : ($s =~ s/$re/$r/ee);
			{ out => $s, n => 0 + ($n || 0) }
		}
		elsif ($op eq 'Split') { [ split $re, $s ] }
	};
	$out = { error => JSON::PP::true } if $@;
	say $json->encode($out);
}
`

// engineResult runs c through scalar with the current defaultEngine, and
// returns the result as JSON, with byte strings as their characters.
func engineResult(c engineCase) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = `{"error":true}`
		}
	}()
	in := Value(c.Input)
	if c.Bytes {
		b := make([]byte, 0, len(c.Input))
		for _, r := range c.Input {
			b = append(b, byte(r))
		}
		in = Value(b)
	}
	var v any
	switch c.Op {
	case "M":
		v = plain(in.M(c.Pattern, c.Flags))
	case "Mg":
		v = plain(in.M(c.Pattern, c.Flags+"g"))
	case "Named":
		v = plain(in.Named(c.Pattern, c.Flags))
	case "S":
		n := in.S(c.Pattern, c.Repl, c.Flags)
		v = map[string]any{"out": plain(in), "n": n.Int()}
	case "Split":
		v = plain(in.Split(c.Pattern))
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false) // as Perl's JSON::PP writes < and >
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// plain turns a result into Go values for JSON: strings as their
// characters, undef and nil containers as nil.
func plain(v any) any {
	switch v := v.(type) {
	case *AV:
		if v == nil {
			return nil
		}
		out := []any{}
		for _, item := range v.items {
			out = append(out, plain(item))
		}
		return out
	case *HV:
		if v == nil {
			return nil
		}
		out := map[string]any{}
		for k, item := range v.m {
			out[k] = plain(item)
		}
		return out
	case *SV:
		if !v.Defined() {
			return nil
		}
		if a, ok := v.val().(*AV); ok {
			return plain(a)
		}
		s, _ := v.text()
		return s
	}
	panic(fmt.Sprintf("plain: %T", v))
}

// TestEnginesAgainstPerl runs engineCases through Perl, regexp2 as now,
// regexp2 through the seam, and Go's regexp. The seam must give what
// regexp2 gives; the other differences are the experiment's findings,
// logged as a table, and written to the file in SCALAR_ENGINE_REPORT if
// it's set.
func TestEnginesAgainstPerl(t *testing.T) {
	needPerl(t)
	in, err := json.Marshal(engineCases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("perl", "-e", perlEngineScript)
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v", err)
	}
	perl := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(perl) != len(engineCases) {
		t.Fatalf("perl gave %d results for %d cases", len(perl), len(engineCases))
	}

	run := func(e Engine) []string {
		defaultEngine = e
		defer func() { defaultEngine = nil }()
		var got []string
		for _, c := range engineCases {
			got = append(got, engineResult(c))
		}
		return got
	}
	native, seam, stdlib := run(nil), run(regexp2Engine{}), run(Stdlib)

	var report strings.Builder
	w := tabwriter.NewWriter(&report, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "case\tregexp2\tstdlib\tPerl gives\tregexp2 gives\tstdlib gives")
	mark := func(got, want string) string {
		switch {
		case got == want:
			return "same"
		case got == `{"error":true}`:
			return "ERROR"
		}
		return "DIFF"
	}
	for i, c := range engineCases {
		if seam[i] != native[i] {
			t.Errorf("%s: regexp2 through the seam gives %s, regexp2 gives %s", c.Note, seam[i], native[i])
		}
		row := []string{c.Note, mark(native[i], perl[i]), mark(stdlib[i], perl[i]), "", "", ""}
		if native[i] != perl[i] || stdlib[i] != perl[i] {
			row[3], row[4], row[5] = perl[i], native[i], stdlib[i]
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
	t.Log("\n" + report.String())
	if path := os.Getenv("SCALAR_ENGINE_REPORT"); path != "" {
		if err := os.WriteFile(path, []byte(report.String()), 0o644); err != nil {
			t.Error(err)
		}
	}
}
