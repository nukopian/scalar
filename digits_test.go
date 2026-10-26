package scalar

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// The tests for numbers that keep their digits (idea 3): a number read as
// text, from JSON, YAML or a dump, keeps its text when its float can't
// give the number back, so that it's written out with every digit.

// faithfulOracle is faithful worked out exactly: the float's shortest
// form has the same value as the text.
func faithfulOracle(text string) bool {
	f, _ := strconv.ParseFloat(text, 64)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return false
	}
	want, ok := new(big.Rat).SetString(strings.TrimPrefix(text, "+"))
	if !ok {
		// An exponent too big for big.Rat, such as 1e-1000001: the float
		// is zero, which gives the number back only if it is zero.
		d, ok := parseDecimal(text)
		return !ok || (f == 0) == (d.digits == "")
	}
	got, _ := new(big.Rat).SetString(strconv.FormatFloat(f, 'g', -1, 64))
	return want.Cmp(got) == 0
}

var faithfulCases = []string{
	"0", "-0", "0.0", "1", "4.5", "-118.25", "33.7612", "0.1", "19.90", "1e3", "1E-3", "0.000001",
	"123456789012345", "1.23456789012345", "0.123456789012345", // 15 significant digits: always faithful
	"1234567890123456", "12345678901234567", "1.2345678901234567", "0.30000000000000004", // 16 and 17
	"9007199254740993", "9007199254740992", "1.00000000000000001", "1.5000000000000000000000",
	"1.23456789012345678901", "-132.48854465954707381603", "12345678901234567890", "-9223372036854775809",
	"18446744073709551616", "123456789012345678901234567890", "9223372036854775808",
	"1e400", "-1e400", "1e-400", "4.9e-324", "5e-324", "2.2250738585072014e-308", "2.225073858507201e-308",
	"1.7976931348623157e308", "1.7976931348623159e308", "0.000000000000000000000000000001",
	"100000000000000000000000", "1e22", "1e23", "123.456e-7", "+1.5", ".5", "5.",
	"1e1000001", "1e-1000001", "0e1000001", "10E1000000", // found by FuzzFaithful: exponents too big for big.Rat
}

// TestFaithful checks the fast test of whether a number's text must be
// kept against exact arithmetic.
func TestFaithful(t *testing.T) {
	for _, text := range faithfulCases {
		f, _ := strconv.ParseFloat(text, 64)
		if got, want := faithful(text, f), faithfulOracle(text); got != want {
			t.Errorf("faithful(%q) = %v, exact arithmetic says %v", text, got, want)
		}
	}
}

// FuzzFaithful is TestFaithful on the fuzzer's numbers, and checks that
// the kept text, as numberText writes it, is the same number.
func FuzzFaithful(f *testing.F) {
	for _, text := range faithfulCases {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		n, ok := parseDecimal(text)
		if !ok {
			return
		}
		fl, err := strconv.ParseFloat(text, 64)
		if err != nil && !strings.Contains(err.Error(), "range") {
			return // a form ParseFloat doesn't read, such as one with many leading zeros past its limit
		}
		if got, want := faithful(text, fl), faithfulOracle(text); got != want {
			t.Errorf("faithful(%q) = %v, exact arithmetic says %v", text, got, want)
		}
		canon := n.text()
		a, aok := new(big.Rat).SetString(strings.TrimPrefix(text, "+"))
		b, bok := new(big.Rat).SetString(canon)
		// big.Rat refuses exponents beyond a million; TestNumberText
		// covers those.
		if aok && bok && a.Cmp(b) != 0 {
			t.Errorf("%q is written %q, which isn't the same number", text, canon)
		}
		if !json.Valid([]byte(canon)) {
			t.Errorf("%q is written %q, which isn't JSON", text, canon)
		}
	})
}

// TestNumberText checks how a kept number is written, exponents beyond
// big.Rat's reach included.
func TestNumberText(t *testing.T) {
	for in, want := range map[string]string{
		"1.23456789012345678901":   "1.23456789012345678901",
		"-0001.5000":               "-1.5",
		"+.5":                      "0.5",
		"5.":                       "5",
		"12345678901234567890":     "12345678901234567890",
		"1e20":                     "100000000000000000000",
		"1e21":                     "1e21",
		"1e22":                     "1e22",
		"1.5e-6":                   "0.0000015",
		"1.5e-7":                   "1.5e-7",
		"10E1000000":               "1e1000001",
		"0.000e5":                  "0",
		"-0":                       "0",
		"123456789012345678901e-5": "1234567890123456.78901",
	} {
		d, ok := parseDecimal(in)
		if got := d.text(); !ok || got != want {
			t.Errorf("parseDecimal(%q).text() = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "-", ".", "1e", "1e+", "1.2.3", "0x1F", ".inf", "1_000", "e5", "1e5x"} {
		if _, ok := parseDecimal(bad); ok {
			t.Errorf("parseDecimal(%q) should fail", bad)
		}
	}
}

// sameNumber reports whether two numbers' texts are the same number.
func sameNumber(a, b string) bool {
	x, ok1 := new(big.Rat).SetString(a)
	y, ok2 := new(big.Rat).SetString(b)
	return ok1 && ok2 && x.Cmp(y) == 0
}

// numbersIn is every number in JSON, as its text.
func numbersIn(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var out []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if n, ok := tok.(json.Number); ok {
			out = append(out, string(n))
		}
	}
	return out
}

var keptDocs = []string{
	`[1.23456789012345678901, -132.48854465954707381603, 12345678901234567890, 18446744073709551616]`,
	`{"seq": 123456789012345678901234567890, "mag": 4.5, "felt": 12, "lat": 0.30000000000000004}`,
	`[1e400, -1e400, 4.9e-324, 1.5000000000000000000001, -0.000000000000000000001234567890123456789]`,
	`[9223372036854775808, -9223372036854775809, 1.7976931348623159e308, 2.225073858507201e-308]`,
	`[0, -0, 0.1, 19.90, 1e3, 33.7612, 9007199254740993]`,
}

// TestKeptRoundTrip reads numbers with both JSON readers and writes them
// back: each must be the same number it was, digit for digit.
func TestKeptRoundTrip(t *testing.T) {
	for _, doc := range keptDocs {
		in := numbersIn(t, []byte(doc))
		for _, fast := range []bool{true, false} {
			fastPaths = fast
			s, err := FromJSON([]byte(doc))
			fastPaths = true
			if err != nil {
				t.Fatalf("%s: %v", doc, err)
			}
			out, err := json.Marshal(s)
			if err != nil {
				t.Fatalf("%s (fast %v): %v", doc, fast, err)
			}
			got := numbersIn(t, out)
			if len(got) != len(in) {
				t.Fatalf("%s (fast %v): %d numbers back, want %d", doc, fast, len(got), len(in))
			}
			// Hash keys come back sorted, so match each number anywhere.
			for _, n := range in {
				found := false
				for _, g := range got {
					found = found || sameNumber(n, g)
				}
				if !found {
					t.Errorf("%s (fast %v): %s didn't come back: %v", doc, fast, n, got)
				}
			}
		}
	}
}

// TestKeepDigitsSwitch: with nothing to keep, the output is what it was
// without the feature, and with long numbers it isn't, so the comparison
// can tell.
func TestKeepDigitsSwitch(t *testing.T) {
	run := func(data []byte) (on, off string) {
		defer func() { keepDigits = true }()
		for _, k := range []bool{true, false} {
			keepDigits = k
			s := must(FromJSON(data))
			b := string(must(json.Marshal(s))) + "\n" + s.Dump() + "\n" + string(must(s.ToYAML()))
			if k {
				on = b
			} else {
				off = b
			}
		}
		return on, off
	}
	if on, off := run(feedPlain()); on != off {
		t.Error("the plain feed comes out differently with digits kept, but it has none to keep")
	}
	if on, off := run(feedLong()); on == off {
		t.Error("the long feed comes out the same either way, so the comparison can't tell")
	}
}

// TestKeptCarried checks that a kept number's digits survive what copies
// or writes a value.
func TestKeptCarried(t *testing.T) {
	const long = "1.23456789012345678901"
	const id = "12345678901234567890"
	doc := must(FromJSON([]byte(`{"x": ` + long + `, "seq": ` + id + `}`)))
	x, seq := doc.Get("x"), doc.Get("seq")
	checks := map[string]string{
		"Str":            x.Str(),
		"Copy":           x.Copy().Str(),
		"Clone":          doc.Clone().Get("x").Str(),
		"Set":            (&SV{}).Set(x).Str(),
		"Value":          Value(x).Str(),
		"Hash":           Hash("x", x).Get("x").Str(),
		"Array":          Array(x).Get(0).Str(),
		"FromJSON top":   must(FromJSON([]byte(long))).Str(),
		"Unmarshal":      func() string { var s SV; _ = json.Unmarshal([]byte(long), &s); return s.Str() }(),
		"json.Number":    Value(json.Number(long)).Str(),
		"YAML out":       string(must(x.ToYAML())),
		"YAML back":      must(FromYAML(must(Value(x).ToYAML()))).Str(),
		"FromYAML":       must(FromYAML([]byte(long))).Str(),
		"Dump":           x.Dump(),
		"FromDump":       must(FromDump(long)).Str(),
		"Sprintf %v":     fmt.Sprintf("%v", x),
		"To string":      must(To[string](x)),
		"To json.Number": string(must(To[json.Number](x))),
		"hash key":       Hash(x, 1).KeysSlice()[0],
	}
	for k, got := range checks {
		if !strings.Contains(got, long) {
			t.Errorf("%s: %s", k, got)
		}
	}
	if u, err := To[uint64](seq); err != nil || strconv.FormatUint(u, 10) != id {
		t.Errorf("To[uint64]: %v, %v", u, err)
	}
	if b, err := To[*big.Int](seq); err != nil || b.String() != id {
		t.Errorf("To[*big.Int]: %v, %v", b, err)
	}
	if got := fmt.Sprintf("%d", seq); got != id {
		t.Errorf("%%d: %s", got)
	}
	if got := Value(uint64(12345678901234567890)).Str(); got != id {
		t.Errorf("Value(uint64): %s", got)
	}
}

// TestKeptBehavesAsItsFloat: kept digits change only how a number is
// written: arithmetic, Kind and Any see the float, as before.
func TestKeptBehavesAsItsFloat(t *testing.T) {
	x := must(FromJSON([]byte(`[1.23456789012345678901]`))).Get(0)
	f := Value(1.23456789012345678901)
	for name, pair := range map[string][2]any{
		"Kind":    {x.Kind(), f.Kind()},
		"Num":     {x.Num(), f.Num()},
		"Any":     {x.Any(), f.Any()},
		"Add":     {x.Add(1).Str(), f.Add(1).Str()},
		"Int":     {x.Int(), f.Int()},
		"Bool":    {x.Bool(), f.Bool()},
		"%.3f":    {fmt.Sprintf("%.3f", x), fmt.Sprintf("%.3f", f)},
		"NumEq f": {x.NumEq(f), true},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s: %v, the float gives %v", name, pair[0], pair[1])
		}
	}
}

// TestKeptMutatorsClear: every way of changing a scalar in place drops the
// kept text, so a changed value never prints its old digits.
func TestKeptMutatorsClear(t *testing.T) {
	muts := map[string]func(s *SV){
		"Set":           func(s *SV) { s.Set(1.5) },
		"Undef":         func(s *SV) { s.Undef() },
		"Inc":           func(s *SV) { s.Inc() },
		"Dec":           func(s *SV) { s.Dec() },
		"AddInPlace":    func(s *SV) { s.AddInPlace(1) },
		"SubInPlace":    func(s *SV) { s.SubInPlace(1) },
		"MulInPlace":    func(s *SV) { s.MulInPlace(1) },
		"DivInPlace":    func(s *SV) { s.DivInPlace(1) },
		"ModInPlace":    func(s *SV) { s.ModInPlace(7) },
		"PowInPlace":    func(s *SV) { s.PowInPlace(1) },
		"ConcatInPlace": func(s *SV) { s.ConcatInPlace("") },
		"RepeatInPlace": func(s *SV) { s.RepeatInPlace(1) },
		"OrSet":         func(s *SV) { s.Set(nil).OrSet(1) },
		"S":             func(s *SV) { s.S(`1$`, "2") },
		"Tr":            func(s *SV) { s.Tr("0", "9") },
		"SubstrReplace": func(s *SV) { s.SubstrReplace(0, 1, "9") },
		"AV":            func(s *SV) { s.Undef(); s.AV().Push(1) },
		"HV":            func(s *SV) { s.Undef(); s.HV().Set("k", 1) },
		"Apply":         func(s *SV) { s.Apply(func(e *SV) any { return 2.5 }) },
		"UnmarshalJSON": func(s *SV) { _ = json.Unmarshal([]byte("2.5"), s) },
		"UnmarshalYAML": func(s *SV) { v, _ := FromYAML([]byte("2.5")); s.Set(v) },
	}
	for name, f := range muts {
		s := must(FromJSON([]byte(`[1.23456789012345678901]`))).Get(0)
		if s.digits() == "" {
			t.Fatal("nothing kept to start with")
		}
		f(s)
		if s.digits() != "" {
			t.Errorf("%s: %v keeps stale digits %s", name, s.val(), s.digits())
		}
	}
}

// TestKeptCompare: numbers that keep their digits compare by them, and so
// do strings of digits a float can't hold, so two ids that differ in the
// last digit are different, as Perl's are for whole numbers.
func TestKeptCompare(t *testing.T) {
	doc := must(FromJSON([]byte(`[12345678901234567890, 12345678901234567891, 1.23456789012345678901, 1.23456789012345678902]`)))
	a, b, c, d := doc.Get(0), doc.Get(1), doc.Get(2), doc.Get(3)
	for name, got := range map[string]bool{
		"ids ==":           a.NumEq(b),
		"ids <":            !a.NumLt(b),
		"decimals ==":      c.NumEq(d),
		"decimals <":       !c.NumLt(d),
		"id == itself":     !a.NumEq(Value("12345678901234567890")),
		"strings ==":       Value("12345678901234567890").NumEq("12345678901234567891"),
		"id > int64":       !a.NumGt(math.MaxInt64),
		"Equal":            a.Equal(b),
		"decimal vs float": !c.NumGt(1.2), // a float compares as a float
	} {
		if got {
			t.Errorf("%s: wrong", name)
		}
	}
	if n := Array(a, b, Value("12345678901234567891"), "12345678901234567890").UniqueNum().Length(); n != 2 {
		t.Errorf("UniqueNum kept %d, want 2", n)
	}
	if got := show(Array(b, c, a, d, 1).Sort(NumCmp)); got != "1,1.23456789012345678901,1.23456789012345678902,12345678901234567890,12345678901234567891" {
		t.Errorf("Sort: %s", got)
	}
	if got := Array(a, b).MaxNum().Str(); got != "12345678901234567891" {
		t.Errorf("MaxNum: %s", got)
	}
	data := must(FromJSON([]byte(`[{"seq":12345678901234567890},{"seq":12345678901234567891}]`)))
	if n := data.Find(`$[?@.seq == 12345678901234567891]`).Length(); n != 1 {
		t.Errorf("Find ==: %d, want 1", n)
	}
	if u, err := To[uint64](Value("12345678901234567890")); err != nil || u != 12345678901234567890 {
		t.Errorf("To[uint64] of a string of digits: %v, %v", u, err)
	}
}

// TestKeptAgainstJSONPP compares the numbers written back with JSON::PP's,
// with allow_bignum, which keeps a long number exactly as Math::BigInt or
// Math::BigFloat. Where JSON::PP keeps a number exactly, scalar must give
// the same number.
func TestKeptAgainstJSONPP(t *testing.T) {
	needPerl(t)
	doc := `[1.23456789012345678901, -132.48854465954707381603, 123456789012345678901234567890, 12345678901234567890, 0.30000000000000004, 1.5000000000000000000001, 4.5, 33.7612]`
	cmd := exec.Command("perl", "-MJSON::PP", "-e", `my $j = JSON::PP->new->allow_bignum; local $/; print $j->encode($j->decode(<STDIN>))`)
	cmd.Stdin = strings.NewReader(doc)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("perl: %v", err)
	}
	perl := numbersIn(t, out)
	ours := numbersIn(t, must(json.Marshal(must(FromJSON([]byte(doc))))))
	in := numbersIn(t, []byte(doc))
	for i := range in {
		if sameNumber(perl[i], in[i]) && !sameNumber(ours[i], in[i]) {
			t.Errorf("%s: JSON::PP keeps it as %s, scalar writes %s", in[i], perl[i], ours[i])
		}
	}
}

// must is v, panicking on an error, for a test's setup.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// TestKeptLockedConcurrent compares locked numbers that keep their digits
// from many goroutines: the cache of their parsed digits mustn't be
// written then, as a locked structure may be read concurrently. Run with
// -race.
func TestKeptLockedConcurrent(t *testing.T) {
	ids := must(FromJSON([]byte(`[12345678901234567891, 12345678901234567890]`))).AV().Lock()
	a, b := ids.Get(0), ids.Get(1) // the locked elements themselves, not copies
	done := make(chan int)
	for range 8 {
		go func() {
			done <- NumCmp(a, b)
		}()
	}
	for range 8 {
		if got := <-done; got != 1 {
			t.Errorf("NumCmp: %d, want 1", got)
		}
	}
}

// TestKeptCopyIsItsOwn: Copy, Clone and Sort's copies keep the digits and
// carry the parsed form, so a copy doesn't parse them again, but each has
// its own, so changing a copy leaves the original as it was.
func TestKeptCopyIsItsOwn(t *testing.T) {
	const id = "12345678901234567890"
	doc := must(FromJSON([]byte(`[` + id + `, 12345678901234567891]`)))
	orig := doc.Get(0)
	NumCmp(orig, doc.Get(1)) // they tie as floats, so this parses orig's digits
	if orig.rare.dec == nil {
		t.Fatal("comparing didn't cache the parsed digits")
	}
	for name, c := range map[string]*SV{
		"Copy":  orig.Copy(),
		"Clone": doc.Clone().Get(0),
		"Sort":  doc.AV().Sort(NumCmp).Get(0),
		"Value": Value(orig),
	} {
		if c.Str() != id {
			t.Errorf("%s: %s", name, c.Str())
		}
		if c.rare == nil || c.rare.dec != orig.rare.dec {
			t.Errorf("%s: the parsed digits weren't carried", name)
		}
		c.Set(1.5)
		if orig.Str() != id || orig.rare.dec == nil {
			t.Errorf("%s: changing the copy changed the original: %s", name, orig.Str())
		}
	}
}
