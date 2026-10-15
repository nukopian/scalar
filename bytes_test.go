package scalar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestBytesKind(t *testing.T) {
	raw := []byte{0x89, 'P', 'N', 'G', 0x00, 0xff}
	s := Value(raw)
	raw[0] = 0 // the scalar has its own copy
	if s.Kind() != Bytes || s.Length() != 6 || s.Str() != "\x89PNG\x00\xff" {
		t.Errorf("kind %v, length %d, str %q", s.Kind(), s.Length(), s.Str())
	}
	b := s.Bytes()
	b[1] = 'X'
	if s.Str()[1] != 'P' {
		t.Error("Bytes should return a copy")
	}
	if got := Value("café").Bytes(); string(got) != "café" || len(got) != 5 {
		t.Errorf("Bytes of text gives its UTF-8: %q", got)
	}
	if Value([]byte(nil)).Defined() {
		t.Error("a nil []byte is undef, as a nil slice is")
	}
	if Value([]byte{}).Bool() || Value([]byte("0")).Bool() || !Value([]byte("00")).Bool() {
		t.Error("a byte string is false when empty or \"0\", as a string is")
	}
	if Value([]byte("42 apples")).Int() != 42 {
		t.Error("a byte string numifies as its text does")
	}

	// A []byte in a list is one value, not a list of numbers.
	if a := Array([]byte("ab"), "c"); a.Length() != 2 || a.Get(0).Kind() != Bytes {
		t.Errorf("Array([]byte, ...) = %d elements", a.Length())
	}

	// Pack gives one, so Length counts its bytes.
	if p := Pack("d>", 1.5); p.Kind() != Bytes || p.Length() != 8 {
		t.Errorf("Pack: kind %v, length %d", p.Kind(), p.Length())
	}

	// Formatting: printed as the bytes are, %x in hex, %q quoted.
	if got := fmt.Sprintf("%s|%v|%x|%q", Value([]byte("Jo")), Value([]byte("Jo")), s, Value([]byte{0xff})); got != `Jo|Jo|89504e4700ff|"\xff"` {
		t.Errorf("formats = %s", got)
	}

	// JSON: base64, as encoding/json gives a []byte; Any gives a copy.
	out, err := json.Marshal(Hash("blob", s))
	if err != nil || string(out) != `{"blob":"iVBORwD/"}` {
		t.Errorf("JSON = %s, %v", out, err)
	}
	if a := s.Any().([]byte); len(a) != 6 {
		t.Errorf("Any = %v", a)
	} else {
		a[0] = 1
		if s.Str()[0] != 0x89 {
			t.Error("Any should return a copy")
		}
	}

	// Text appended as characters: é (U+00E9) is one byte, ☺ makes text.
	if got := Value([]byte{0xe9}).Concat("é"); got.Kind() != Bytes || got.Str() != "\xe9\xe9" {
		t.Errorf("bytes . é = %v %q", got.Kind(), got.Str())
	}
	if got := Value([]byte{0xe9}).Concat("☺"); got.Kind() != String || got.Str() != "é☺" {
		t.Errorf("bytes . ☺ = %v %q", got.Kind(), got.Str())
	}
	if got := Value([]byte("x")).Concat(Value([]byte("é"))); got.Str() != "x\xc3\xa9" {
		t.Errorf("appending text's UTF-8 bytes: %q", got.Str())
	}
}

// Byte strings behave as Perl's do under use v5.36 (unicode_strings), for
// hundreds of random byte strings. It needs perl on the PATH, and is
// skipped without it.
func TestBytesMatchPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	rng := rand.New(rand.NewPCG(7, 11))
	patterns := []string{`\x89PNG`, `(\x00.)`, `\w+`, `(?i)\xc9`, `\s+`, `[\x80-\xff]+`, `^(.)(.)`, `(.)\1`, `\d+`}
	texts := []string{"", "x", "é", "☺"}

	type caseIn struct {
		B      string `json:"b"` // hex
		O, L   int
		Needle string `json:"needle"` // hex
		Pos    int    `json:"pos"`
		P      string `json:"p"`
		Text   string `json:"text"`
	}
	interesting := []byte("abcXYZ019 \t\n\x00\x85\x89\xa0\xaa\xb5\xc9\xdf\xe9\xff")
	var cases []caseIn
	for range 300 {
		b := make([]byte, rng.IntN(14))
		for i := range b {
			if rng.IntN(3) == 0 {
				b[i] = byte(rng.IntN(256))
			} else {
				b[i] = interesting[rng.IntN(len(interesting))]
			}
		}
		var needle []byte
		if len(b) > 0 && rng.IntN(2) == 0 {
			i := rng.IntN(len(b))
			needle = b[i : i+1+rng.IntN(min(2, len(b)-i))]
		} else {
			needle = []byte{interesting[rng.IntN(len(interesting))]}
		}
		cases = append(cases, caseIn{
			B: hex.EncodeToString(b), O: rng.IntN(31) - 15, L: rng.IntN(31) - 15,
			Needle: hex.EncodeToString(needle), Pos: rng.IntN(20) - 3,
			P: patterns[rng.IntN(len(patterns))], Text: texts[rng.IntN(len(texts))],
		})
	}

	in, _ := json.Marshal(cases)
	const script = `
use v5.36; use builtin qw(trim); no warnings;
use JSON::PP;
sub enc ($x) {
	return "U" unless defined $x;
	my $c = $x;
	if ($c =~ /[^\x00-\xff]/) { utf8::encode($c); return "T:" . unpack("H*", $c) }
	utf8::downgrade($c);
	return "B:" . unpack("H*", $c);
}
my @out;
for my $c (@{ decode_json(do { local $/; <STDIN> }) }) {
	my $b = pack("H*", $c->{b});
	my $n = pack("H*", $c->{needle});
	my $p = $c->{p};
	my $t = $c->{text};
	my @r;
	push @r, length($b), enc(substr($b, $c->{O}, $c->{L})), index($b, $n, $c->{pos}), rindex($b, $n, $c->{pos});
	push @r, map { enc($_) } lc($b), uc($b), fc($b), ucfirst($b), lcfirst($b), trim($b);
	push @r, ($b =~ /$p/) ? join(",", map { enc($_) } $&, @{^CAPTURE}) : "nomatch";
	push @r, enc(do { (my $x = $b) =~ s/$p/<>/g; $x });
	push @r, join(",", map { enc($_) } split /$p/, $b);
	push @r, enc(do { (my $x = $b) =~ tr/a-zA-Z/n-za-mN-ZA-M/; $x });
	push @r, enc(do { (my $x = $b) =~ tr/\x80-\xff//d; $x });
	push @r, enc($b . $t), ($b eq $t ? 1 : 0), ($b lt $t ? 1 : 0);
	push @out, \@r;
}
print encode_json(\@out);
`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v: %s", err, stderr.String())
	}
	var want [][]any
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("perl's output: %v", err)
	}

	enc := func(s *SV) string {
		switch {
		case !s.Defined():
			return "U"
		case s.Kind() == Bytes:
			return "B:" + hex.EncodeToString([]byte(s.Str()))
		}
		return "T:" + hex.EncodeToString([]byte(s.Str())) // text: Perl gives B unless it has characters beyond 255
	}
	encAll := func(a *AV) string {
		parts := make([]string, a.Length())
		for i, s := range a.All() {
			parts[i] = enc(s)
		}
		return strings.Join(parts, ",")
	}
	boolStr := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	names := []string{"length", "substr", "index", "rindex", "lc", "uc", "fc", "ucfirst", "lcfirst", "trim",
		"match", "s///g", "split", "tr rot13", "tr///d", "concat text", "eq text", "lt text"}

	for i, c := range cases {
		raw, _ := hex.DecodeString(c.B)
		needle, _ := hex.DecodeString(c.Needle)
		b := func() *SV { return Value(raw) }
		match := "nomatch"
		if m := b().M(c.P); m != nil {
			match = encAll(m)
		}
		got := []string{
			strconv.Itoa(b().Length()), enc(b().Substr(c.O, c.L)),
			strconv.Itoa(b().Index(Value(needle), c.Pos)), strconv.Itoa(b().RIndex(Value(needle), c.Pos)),
			enc(b().Lc()), enc(b().Uc()), enc(b().Fc()), enc(b().UcFirst()), enc(b().LcFirst()), enc(b().Trim()),
			match, enc(b().S(c.P, "<>", "gr")), encAll(b().Split(c.P)),
			enc(b().Tr("a-zA-Z", "n-za-mN-ZA-M", "r")), enc(b().Tr("\u0080-ÿ", "", "dr")),
			enc(b().Concat(c.Text)), boolStr(b().Eq(c.Text)), boolStr(b().Lt(c.Text)),
		}
		for j, g := range got {
			if w := fmt.Sprint(want[i][j]); g != w {
				t.Errorf("%s of %s (pattern %q, text %q, needle %s, %d/%d/%d): Go %s, Perl %s",
					names[j], c.B, c.P, c.Text, c.Needle, c.O, c.L, c.Pos, g, w)
			}
		}
	}
}
