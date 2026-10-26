package scalar

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The tests for what the wash-up before v0.6.0 found: % on floats beyond
// int64, ++ on strings of digits, and To into a type that reads text,
// from a number.

// perlOperand is a value both scalar and Perl can hold the same way: an
// integer, a float (a pure NV), a JSON number, or a string.
type perlOperand struct {
	T string `json:"t"` // "i", "f", "j" or "s"
	V string `json:"v"` // digits, a float's bits in hex, JSON, or the string
}

func (o perlOperand) sv() *SV {
	switch o.T {
	case "i":
		i, _ := strconv.ParseInt(o.V, 10, 64)
		return Value(i)
	case "f":
		b, _ := hex.DecodeString(o.V)
		return Value(math.Float64frombits(binary.BigEndian.Uint64(b)))
	case "j":
		return must(FromJSON([]byte(o.V)))
	}
	return Value(o.V)
}

func floatOperand(f float64) perlOperand {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], math.Float64bits(f))
	return perlOperand{"f", hex.EncodeToString(b[:])}
}

// perlEval runs each case through Perl: op is "%" or "++", and prints
// the result, or "die: " and the message.
func perlEval(t *testing.T, cases [][3]any) []string {
	t.Helper()
	script := `use v5.36; use JSON::PP; my $j = JSON::PP->new->allow_nonref; no warnings;
sub operand ($o) {
	return 0 + "$o->{v}" if $o->{t} eq "i";
	return unpack("d>", pack("H*", $o->{v})) if $o->{t} eq "f";
	return $j->decode($o->{v}) if $o->{t} eq "j";
	return "$o->{v}";
}
$| = 1;
while (my $line = <STDIN>) {
	my ($op, $a, $b) = @{$j->decode($line)};
	my ($x, $y) = (operand($a), $b ? operand($b) : undef);
	my $r = eval { $op eq "%" ? $x % $y : do { my $z = $x; $z++; $z } };
	if ($@) { (my $e = $@) =~ s/ at .*//s; say "die: $e" } else { say $r }
}`
	var in strings.Builder
	for _, c := range cases {
		in.Write(must(json.Marshal(c)))
		in.WriteByte('\n')
	}
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(in.String())
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v", err)
	}
	var res []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		res = append(res, sc.Text())
	}
	if len(res) != len(cases) {
		t.Fatalf("perl gave %d results for %d cases", len(res), len(cases))
	}
	return res
}

// TestModMatchesPerl: % on every pair of a pool of integers, floats on
// both sides of 2⁶³ and 2⁶⁴, kept JSON numbers and strings, as Perl's
// pp_modulo gives it.
func TestModMatchesPerl(t *testing.T) {
	needPerl(t)
	var pool []perlOperand
	for _, i := range []int64{0, 1, -1, 3, -3, 7, -7, 10, math.MaxInt64, math.MinInt64, 1<<53 + 1} {
		pool = append(pool, perlOperand{"i", strconv.FormatInt(i, 10)})
	}
	for _, f := range []float64{1.9, -1.9, 0.5, 2.5, -2.5, 7.75, 1e19, -1e19, 2e19, 1e20, -1e20, 1 << 63, 1 << 64, 1e300, 9007199254740994} {
		pool = append(pool, floatOperand(f))
	}
	for _, j := range []string{"12345678901234567890", "18446744073709551615", "18446744073709551616", "-9223372036854775809", "9223372036854775808"} {
		pool = append(pool, perlOperand{"j", j})
	}
	for _, s := range []string{"12345678901234567890", "7", "1.9", "abc", "18446744073709551614"} {
		pool = append(pool, perlOperand{"s", s})
	}
	var cases [][3]any
	for _, a := range pool {
		for _, b := range pool {
			cases = append(cases, [3]any{"%", a, b})
		}
	}
	want := perlEval(t, cases)
	bad := 0
	for i, c := range cases {
		a, b := c[1].(perlOperand), c[2].(perlOperand)
		got := func() (out string) {
			defer func() {
				if r := recover(); r != nil {
					out = "die: " + strings.TrimPrefix(fmt.Sprint(r), "scalar: ")
				}
			}()
			return a.sv().Mod(b.sv()).Str()
		}()
		if got != want[i] {
			if bad++; bad <= 30 {
				t.Errorf("%v %% %v: scalar %s, Perl %s", a, b, got, want[i])
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d differ from Perl", bad, len(cases))
	}
}

// TestIncDigitsMatchesPerl: ++ on strings, digits included, as Perl's
// sv_inc gives it: Perl increments every string of letters then digits
// as a string, so "007" is "008" and a number too big for an int64 keeps
// every digit. Where a number would print the same, scalar's is one.
func TestIncDigitsMatchesPerl(t *testing.T) {
	needPerl(t)
	ins := []string{"007", "099", "9", "0", "00", "123", "0009", "a9", "Zz99", "zz", "9223372036854775806",
		"9223372036854775807", "12345678901234567890", "18446744073709551615", "99999999999999999999", "1.5", " 7"}
	var cases [][3]any
	for _, s := range ins {
		cases = append(cases, [3]any{"++", perlOperand{"s", s}, nil})
	}
	want := perlEval(t, cases)
	for i, s := range ins {
		if got := Value(s).Inc().Str(); got != want[i] {
			t.Errorf("%q++: scalar %s, Perl %s", s, got, want[i])
		}
	}
	if k := Value("9").Inc().Kind(); k != Int {
		t.Errorf(`"9"++ is %v, want a number`, k)
	}
}

// TestToTextFromNumber: To and Decode fill a type that reads text, such as
// *big.Float or *big.Rat, from a number, with every digit a number keeps.
func TestToTextFromNumber(t *testing.T) {
	kept := must(FromJSON([]byte(`1.23456789012345678901`)))
	r, err := To[*big.Rat](kept)
	if want, _ := new(big.Rat).SetString("1.23456789012345678901"); err != nil || r.Cmp(want) != 0 {
		t.Errorf("To[*big.Rat] of a kept number: %v, %v", r, err)
	}
	r, err = To[*big.Rat](Value(0.1))
	if err != nil || r.String() != "1/10" {
		t.Errorf("To[*big.Rat](0.1): %v, %v, want 1/10, as 0.1 is the float's shortest form", r, err)
	}
	f, err := To[*big.Float](Value(1.5))
	if err != nil || f.String() != "1.5" {
		t.Errorf("To[*big.Float](1.5): %v, %v", f, err)
	}
	if r, err := To[*big.Rat](Value(42)); err != nil || r.RatString() != "42" {
		t.Errorf("To[*big.Rat](42): %v, %v", r, err)
	}
	if _, err := To[time.Time](Value(5)); err == nil {
		t.Error("To[time.Time](5) should fail")
	}
	if _, err := To[*big.Rat](Value(true)); err == nil {
		t.Error("To[*big.Rat](true) should fail: a boolean isn't text or a number")
	}
}
