package scalar

import (
	"encoding/json"
	"math/rand/v2"
	"os/exec"
	"strings"
	"testing"
)

var (
	eAcute = string(rune(0xe9))   // é
	smiley = string(rune(0x263a)) // a character beyond a byte
)

func TestFromDump(t *testing.T) {
	for dump, want := range map[string]any{
		`undef`:               nil,
		`!!1`:                 true,
		`!!0`:                 false,
		`42`:                  int64(42),
		`-7`:                  int64(-7),
		`1_000`:               int64(1000),
		`2.5`:                 2.5,
		`1e3`:                 1000.0,
		`.5`:                  0.5,
		`"2.5"`:               "2.5", // as Data::Dumper quotes it: a string
		`9223372036854775808`: 9223372036854775808.0,
		`"a\"b\$c\@d\n\t\\"`:  "a\"b$c@d\n\t\\",
		`"\0\1x\0011\177\e"`:  "\x00\x01x\x011\x7f\x1b",
		`"caf\x{e9}"`:         "caf" + eAcute,
		`"\x{263a}"`:          smiley,
		`"$5 me@example.com"`: "$5 me@example.com", // not interpolated
		`'it\'s a \\ \n'`:     `it's a \ \n`,
		"\"two\nlines\"":      "two\nlines",
		`"caf` + eAcute + `"`: "caf" + eAcute, // a literal character: text
		`""`:                  "",
		`''`:                  "",
		`"\777"`:              string(rune(0o777)),
		`"\x41\x"`:            "A\x00",
	} {
		got, err := FromDump(dump)
		if err != nil {
			t.Errorf("FromDump(%q): %v", dump, err)
			continue
		}
		if !got.Equal(want) || got.Kind() != Scalar(want).Kind() {
			t.Errorf("FromDump(%q) = %s (%v), want %s", dump, got.Dump(), got.Kind(), Scalar(want).Dump())
		}
	}

	// Byte strings: an octal or \xHH escape above 127, and nothing wider.
	for dump, isBytes := range map[string]bool{`"\211PNG"`: true, `"\xff"`: true, `"\351\x{e9}"`: false, `"PNG"`: false, `"\0"`: false} {
		if got := MustFromDump(dump); (got.Kind() == Bytes) != isBytes {
			t.Errorf("FromDump(%s): kind %v", dump, got.Kind())
		}
	}
	if b := MustFromDump(`"\211P\0"`).Bytes(); string(b) != "\x89P\x00" {
		t.Errorf("bytes: %q", b)
	}

	// Containers, keys in every form, comments, commas, bless.
	doc := MustFromDump(`
		# a posting, as a test might write it
		bless( {
			title => "Go Developer",      # a bare key
			'log level' => 'debug',
			"0" => "zero", -1, "minus one",  # a comma for =>, a negative bare key
			7 => [1, 2.5, "x", undef, !!1, [], {},],
			tags => bless([ "go", 'perl' ], 'My::List'),
		}, 'My::Posting' )
	`)
	want := map[string]any{"title": "Go Developer", "log level": "debug", "0": "zero", "-1": "minus one",
		"7": []any{1, 2.5, "x", nil, true, []any{}, map[string]any{}}, "tags": []any{"go", "perl"}}
	if d := doc.Diff(want); d != "" {
		t.Errorf("containers: %s", d)
	}

	// $VAR1 paths: shared parts and loops come back as they were.
	shared := MustFromDump(`{a => [1], b => $VAR1->{a}, c => {d => $VAR1->{'a'}, e => $VAR1}, f => [$VAR1->{"c"}{d}]}`)
	if shared.At("b").AV() != shared.At("a").AV() || shared.At("c.d").AV() != shared.At("a").AV() ||
		shared.At("f[0]").AV() != shared.At("a").AV() || shared.At("c.e").HV() != shared.HV() {
		t.Error("$VAR1 paths should be the arrays and hashes already read")
	}
	loop := MustFromDump(`[1, $VAR1, [$VAR1->[2]]]`)
	if loop.At("[1]").AV() != loop.AV() || loop.At("[2][0]").AV() != loop.At("[2]").AV() {
		t.Error("loops through arrays")
	}
}

func TestFromDumpErrors(t *testing.T) {
	for dump, want := range map[string]string{
		``:                "line 1, column 1: there's nothing to read",
		`   `:             "line 1, column 4: there's nothing to read",
		`{a => 1`:         "line 1, column 8: expected a comma or \"}\", found end of the dump",
		"[1,\n 2,\n x]":   "line 3, column 2: expected a value, found the word \"x\"",
		`{a 1}`:           `line 1, column 4: expected => after the key "a", found '1'`,
		`"open`:           `line 1, column 1: this string has no closing "`,
		`'open`:           `line 1, column 6: this string has no closing '`,
		`1 2`:             `line 1, column 3: unexpected '2' after the value`,
		`\'x'`:            `reference to a plain value`,
		`sub { "DUMMY" }`: `sub can't be read: FromDump reads data, not Perl code`,
		`qr/x/`:           `qr can't be read`,
		`$VAR1 = [1];`:    `$VAR1 refers to the value being read, but nothing has been read yet`,
		`[1, $VAR1->[5]]`: `$VAR1->[5] leads to nothing`,
		`[1, $VAR1->[0]]`: `$VAR1->[0] leads to a plain value`,
		`[$VAR2]`:         `only $VAR1 can be read`,
		`[$x]`:            `$x can't be read`,
		`bless({}, Foo)`:  `expected the class name, in quotes`,
		`!!2`:             `expected !!1 or !!0`,
		`"\x{zz}"`:        `\x{...} needs a hex character code`,
		`1e`:              `expected the exponent's digits`,
		`-`:               `expected a value`,
	} {
		_, err := FromDump(dump)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("FromDump(%q): %v\n want %q", dump, err, want)
		}
	}
	expectPanic(t, "MustFromDump", "bad dump", func() { MustFromDump("[") })
}

// randomStructure builds a random structure, with shared parts and loops,
// byte strings, text, numbers, booleans, undef and holes.
func randomStructure(rng *rand.Rand) *SV {
	var containers []*SV
	texts := []string{"", "x", "Go", "a b", "it's", `"q"`, "$5", "@x", "\\", "\n\t", "\x00\x01", "\x7f",
		"caf" + eAcute, smiley, "42", "007", "-1", "1.5", "0", "1e3", "#", "{", "]", "=>"}
	var build func(depth int) *SV
	build = func(depth int) *SV {
		switch n := rng.IntN(14); {
		case depth == 0 || n < 3 && depth < 4:
			if depth > 0 && len(containers) > 0 && rng.IntN(5) == 0 {
				return &SV{v: containers[rng.IntN(len(containers))].v} // shared, or a loop
			}
			if rng.IntN(2) == 0 {
				a := &AV{}
				s := &SV{v: a}
				containers = append(containers, s)
				for range rng.IntN(4) {
					a.items = append(a.items, build(depth+1))
				}
				if len(a.items) > 1 && rng.IntN(4) == 0 {
					a.items[0] = nil // a hole
				}
				return s
			}
			h := newHV(0)
			s := &SV{v: h}
			containers = append(containers, s)
			for range rng.IntN(4) {
				h.store(texts[rng.IntN(len(texts))], build(depth+1))
			}
			return s
		case n < 6:
			return Scalar(texts[rng.IntN(len(texts))])
		case n < 7:
			return Scalar([]byte{byte(rng.IntN(256)), 'P', byte(rng.IntN(256))})
		case n < 9:
			return Scalar([]int64{0, 7, -7, 999999999, 1000000000, -12345678901}[rng.IntN(6)])
		case n < 10:
			return Scalar([]float64{2.5, 1e21, 0.1, -3.75, 1e-5}[rng.IntN(5)])
		case n < 12:
			return Scalar(rng.IntN(2) == 0)
		}
		return &SV{}
	}
	return build(0)
}

// Dump's output reads back as an equal structure that dumps the same, for
// hundreds of random structures, shared parts and loops included.
func TestFromDumpRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 3))
	for i := range 500 {
		orig := randomStructure(rng)
		for _, dump := range []string{orig.Dump(), dump(orig, false)} {
			got, err := FromDump(dump)
			if err != nil {
				t.Fatalf("case %d: %v\n%s", i, err, dump)
			}
			if !got.Equal(orig) {
				t.Fatalf("case %d: %s\n%s", i, got.Diff(orig), dump)
			}
			if again := got.Dump(); again != orig.Dump() {
				t.Fatalf("case %d dumps differently:\n%s\nwas:\n%s", i, again, orig.Dump())
			}
		}
	}
}

// FromDump reads what Perl's Data::Dumper writes, with blessed arrays and
// hashes, under every combination of its layout settings, for hundreds of
// random structures. It needs perl, and is skipped without it.
func TestFromDumpMatchesPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	rng := rand.New(rand.NewPCG(5, 8))
	var cases []string
	for range 150 {
		s := randomStructure(rng)
		js, err := s.MarshalJSON()
		if err != nil || strings.Contains(s.Dump(), "$VAR1") || strings.Contains(s.Dump(), `\`) && rng.IntN(2) == 0 {
			continue // JSON can't carry loops; and keep some of the escapes
		}
		cases = append(cases, string(js))
	}
	in, _ := json.Marshal(cases)
	const script = `
use v5.36; use Data::Dumper; use JSON::PP; use Scalar::Util qw(reftype);
my $json = JSON::PP->new->boolean_values(builtin::false, builtin::true); # the outer decode_json has made characters already
my @out;
my $n = 0;
for my $j (@{ decode_json(do { local $/; <STDIN> }) }) {
	my $d = $json->decode($j);
	# bless some arrays and hashes: FromDump reads them as plain ones
	my @todo = ($d);
	while (my $x = shift @todo) {
		my $type = reftype($x) // next;
		push @todo, $type eq 'ARRAY' ? @$x : values %$x;
		bless $x, "My::Class" if $n++ % 3 == 0;
	}
	for my $indent (0, 1, 2) { for my $useqq (0, 1) { for my $quotekeys (0, 1) { for my $trailing (0, 1) {
		local $Data::Dumper::Terse = 1;
		local $Data::Dumper::Sortkeys = 1;
		local $Data::Dumper::Indent = $indent;
		local $Data::Dumper::Useqq = $useqq;
		local $Data::Dumper::Quotekeys = $quotekeys;
		local $Data::Dumper::Trailingcomma = $trailing;
		push @out, [$j, "$indent$useqq$quotekeys$trailing", Dumper($d)];
	} } } }
}
print JSON::PP->new->utf8->encode(\@out);
`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v: %s", err, stderr.String())
	}
	var dumps [][3]string
	if err := json.Unmarshal(out, &dumps); err != nil {
		t.Fatalf("perl's output: %v", err)
	}
	if len(dumps) < 1000 {
		t.Fatalf("only %d dumps", len(dumps))
	}
	for _, d := range dumps {
		want, _ := FromJSON([]byte(d[0]))
		got, err := FromDump(d[2])
		if err != nil {
			t.Errorf("settings %s: %v\n%s", d[1], err, d[2])
			continue
		}
		if diff := got.Diff(want); diff != "" {
			t.Errorf("settings %s: %s\n%s", d[1], diff, d[2])
		}
	}
	t.Logf("%d dumps from Perl read back", len(dumps))
}
