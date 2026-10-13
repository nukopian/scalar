package scalar

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"math/rand/v2"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func hexOf(s *SV) string { return hex.EncodeToString([]byte(s.Str())) }

// The cases Perl 5.42's pack and unpack were checked against by hand.
func TestPack(t *testing.T) {
	for _, tt := range []struct {
		name, template string
		vals           []any
		want           string // hex
	}{
		{"hex", "H*", []any{"4a6f"}, "4a6f"},
		{"odd hex digits", "H3", []any{"abc"}, "abc0"},
		{"a count beyond the string", "H6", []any{"ab"}, "ab0000"},
		{"low nibble first", "h*", []any{"4a6f"}, "a4f6"},
		{"not hex digits", "H2", []any{"gZ"}, "03"},
		{"bits", "B*", []any{"01001010"}, "4a"},
		{"a part byte", "B5", []any{"10111"}, "b8"},
		{"lowest bit first", "b*", []any{"01001010"}, "52"},
		{"not bits", "B4", []any{"21x3"}, "50"},
		{"float32 little-endian", "f<", []any{1.5}, "0000c03f"},
		{"float32 big-endian", "f>", []any{1.5}, "3fc00000"},
		{"float64 big-endian", "d>", []any{3.14159265358979}, "400921fb54442d11"},
		{"F is a float64", "F<", []any{1.5}, "000000000000f83f"},
		{"several", "f<*", []any{1, 2}, "0000803f00000040"},
		{"a mixed template", "H2 f> B8", []any{"ff", 1, "10000001"}, "ff3f80000081"},
		{"a comment", "H2 # the flag byte\n B8", []any{"ff", "10000001"}, "ff81"},
		{"not a number", "f", []any{"abc"}, "00000000"},
		{"missing values", "H2 f>", nil, "0000000000"},
		{"infinity", "d>", []any{math.Inf(1)}, "7ff0000000000000"},
	} {
		if got := hexOf(Pack(tt.template, tt.vals...)); got != tt.want {
			t.Errorf("%s: Pack(%q, %v) = %s, want %s", tt.name, tt.template, tt.vals, got, tt.want)
		}
	}
	if got := hexOf(Pack("f", 1.5)); got != "0000c03f" {
		t.Errorf("f without < or > uses the machine's order, little-endian here: %s", got)
	}
}

func TestUnpack(t *testing.T) {
	for _, tt := range []struct {
		name, template string
		data           any
		want           string // joined with ","
	}{
		{"hex", "H*", "Jo", "4a6f"},
		{"odd hex digits", "H3", "Jo", "4a6"},
		{"a count beyond the data", "H9", "Jo", "4a6f"},
		{"low nibble first", "h*", "Jo", "a4f6"},
		{"bits", "B*", "J", "01001010"},
		{"a count beyond the data", "B12", "J", "01001010"},
		{"lowest bit first", "b8", "J", "01010010"},
		{"a float32 widens", "f", Pack("f", 0.1), "0.100000001490116"},
		{"several, a spare byte ignored", "f<*", Pack("f<*", 1, 2).Concat("x"), "1,2"},
		{"too few bytes", "d", "abc", ""},
		{"a []byte", "H*", []byte{0xde, 0xad}, "dead"},
		{"bits of a float", "B32", Pack("f>", 1.5), "00111111110000000000000000000000"},
		{"a mixed template", "H2 f> B8", Pack("H2 f> B8", "ff", 1, "10000001"), "ff,1,10000001"},
	} {
		if got := Unpack(tt.template, tt.data).Join(","); got != tt.want {
			t.Errorf("%s: Unpack(%q) = %q, want %q", tt.name, tt.template, got, tt.want)
		}
	}
}

func TestPackErrors(t *testing.T) {
	expectPanic(t, "an unsupported letter", "template letter 'N' isn't supported", func() { Pack("N", 1) })
	expectPanic(t, "an unsupported letter in Unpack", "template letter 'a' isn't supported", func() { Unpack("a*", "x") })
	expectPanic(t, "< on hex", "< and > apply only to f, d and F", func() { Pack("H<", "ab") })
}

// Pack and Unpack give exactly what Perl's do, for hundreds of random
// templates and values. It needs perl on the PATH, and is skipped without
// it.
func TestPackMatchesPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	rng := rand.New(rand.NewPCG(13, 10))

	type packCase struct {
		T string `json:"t"`
		V []any  `json:"v"`
	}
	type unpackCase struct {
		T   string `json:"t"`
		Hex string `json:"hex"`
	}
	randomTemplate := func() (string, []any) {
		var tmpl []string
		var vals []any
		for range 1 + rng.IntN(3) {
			letter := string("HhBbfdF"[rng.IntN(7)])
			item := letter
			if strings.Contains("fdF", letter) {
				item += []string{"", "<", ">"}[rng.IntN(3)]
			}
			count := 1
			switch rng.IntN(3) {
			case 1:
				count = rng.IntN(13)
				item += strconv.Itoa(count)
			case 2:
				item += "*"
				count = rng.IntN(4)
			}
			tmpl = append(tmpl, item)
			switch {
			case strings.Contains("Hh", letter):
				vals = append(vals, randomString(rng, "0123456789abcdefABCDEFgxZ ", 12))
			case strings.Contains("Bb", letter):
				vals = append(vals, randomString(rng, "0101x2", 20))
			default:
				for range count {
					vals = append(vals, randomFloat(rng))
				}
			}
		}
		return strings.Join(tmpl, " "), vals
	}

	var packs []packCase
	var unpacks []unpackCase
	for range 400 {
		tmpl, vals := randomTemplate()
		packs = append(packs, packCase{tmpl, vals})
		data := make([]byte, rng.IntN(24))
		for i := range data {
			data[i] = byte(rng.IntN(256))
		}
		utmpl, _ := randomTemplate()
		unpacks = append(unpacks, unpackCase{utmpl, hex.EncodeToString(data)})
	}

	in, _ := json.Marshal(map[string]any{"pack": packs, "unpack": unpacks})
	const script = `
use strict; use warnings; no warnings qw(numeric uninitialized);
use JSON::PP;
my $in = decode_json(do { local $/; <STDIN> });
my @pack = map { unpack("H*", pack($_->{t}, @{ $_->{v} // [] })) } @{ $in->{pack} };
my @unpack = map { [ map { "$_" } unpack($_->{t}, pack("H*", $_->{hex})) ] } @{ $in->{unpack} };
print encode_json({ pack => \@pack, unpack => \@unpack });
`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v: %s", err, stderr.String())
	}
	var want struct {
		Pack   []string   `json:"pack"`
		Unpack [][]string `json:"unpack"`
	}
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("perl's output: %v", err)
	}

	for i, c := range packs {
		if got := hexOf(Pack(c.T, c.V...)); got != want.Pack[i] {
			t.Errorf("Pack(%q, %v) = %s, Perl gives %s", c.T, c.V, got, want.Pack[i])
		}
	}
	for i, c := range unpacks {
		data, _ := hex.DecodeString(c.Hex)
		got := Unpack(c.T, data).StrSlice()
		if strings.Join(got, "|") != strings.Join(want.Unpack[i], "|") {
			t.Errorf("Unpack(%q, %s) = %q, Perl gives %q", c.T, c.Hex, got, want.Unpack[i])
		}
	}
}

func randomString(rng *rand.Rand, chars string, maxLen int) string {
	b := make([]byte, rng.IntN(maxLen+1))
	for i := range b {
		b[i] = chars[rng.IntN(len(chars))]
	}
	return string(b)
}

// randomFloat gives a float that JSON can carry exactly: ordinary numbers,
// big and small ones, and ones too big or too small for a float32.
func randomFloat(rng *rand.Rand) float64 {
	switch rng.IntN(6) {
	case 0:
		return float64(rng.IntN(2001) - 1000)
	case 1:
		return (rng.Float64() - 0.5) * 1e6
	case 2:
		return (rng.Float64() - 0.5) * 1e300 // beyond a float32
	case 3:
		return (rng.Float64() - 0.5) * 1e-40 // a float32 subnormal, or zero
	case 4:
		return 0.1
	}
	return rng.NormFloat64()
}
