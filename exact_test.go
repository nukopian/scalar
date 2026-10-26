package scalar

import (
	"encoding/json"
	"math"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// SCALAR_EXACT picks the experiment's mode for the whole test run: off,
// text or uv.
func init() {
	switch os.Getenv("SCALAR_EXACT") {
	case "text":
		exactInts = exactText
	case "uv":
		exactInts = exactUV
	}
}

// inMode runs f with exactInts set to m.
func inMode(t *testing.T, m exactMode, f func(t *testing.T)) {
	old := exactInts
	defer func() { exactInts = old }()
	exactInts = m
	f(t)
}

var modes = []struct {
	name string
	m    exactMode
}{{"off", exactOff}, {"text", exactText}, {"uv", exactUV}}

// TestExactRoundTrip tabulates what each mode keeps of each number, read
// by both JSON readers.
func TestExactRoundTrip(t *testing.T) {
	type want struct{ str, json string }
	cases := []struct {
		in            string
		off, text, uv want
	}{
		{"12345678901234567890",
			want{"1.23456789012346e+19", "12345678901234567000"},
			want{"12345678901234567890", "12345678901234567890"},
			want{"12345678901234567890", "12345678901234567890"}},
		{"18446744073709551615",
			want{"1.84467440737096e+19", "18446744073709552000"},
			want{"18446744073709551615", "18446744073709551615"},
			want{"18446744073709551615", "18446744073709551615"}},
		{"18446744073709551616",
			want{"1.84467440737096e+19", "18446744073709552000"},
			want{"18446744073709551616", "18446744073709551616"},
			want{"1.84467440737096e+19", "18446744073709552000"}},
		{"-9223372036854775809",
			want{"-9.22337203685478e+18", "-9223372036854776000"},
			want{"-9223372036854775809", "-9223372036854775809"},
			want{"-9.22337203685478e+18", "-9223372036854776000"}},
		{"123456789012345678901234567890",
			want{"1.23456789012346e+29", "1.2345678901234568e+29"},
			want{"123456789012345678901234567890", "123456789012345678901234567890"},
			want{"1.23456789012346e+29", "1.2345678901234568e+29"}},
		{"1e20", want{"1e+20", "100000000000000000000"}, want{"1e+20", "100000000000000000000"}, want{"1e+20", "100000000000000000000"}},
		{"9223372036854775807", want{"9223372036854775807", "9223372036854775807"}, want{"9223372036854775807", "9223372036854775807"}, want{"9223372036854775807", "9223372036854775807"}},
	}
	for _, mode := range modes {
		inMode(t, mode.m, func(t *testing.T) {
			for _, c := range cases {
				w := map[exactMode]want{exactOff: c.off, exactText: c.text, exactUV: c.uv}[mode.m]
				for _, fast := range []bool{true, false} {
					fastPaths = fast
					s, err := FromJSON([]byte("[" + c.in + "]"))
					fastPaths = true
					if err != nil {
						t.Fatal(err)
					}
					e := s.Get(0)
					out, _ := json.Marshal(e)
					if e.Str() != w.str || string(out) != w.json {
						t.Errorf("%s fast=%v %s: Str %s JSON %s, want %s %s", mode.name, fast, c.in, e.Str(), out, w.str, w.json)
					}
					if _, isF := e.Any().(float64); !isF && strings.ContainsAny(w.str, "e") {
						t.Errorf("%s %s: Any is %T", mode.name, c.in, e.Any())
					}
				}
			}
		})
	}
}

// TestExactCarried checks that the digits survive what copies a value:
// Copy, Clone, Set from another scalar, a container's Get, Value, and
// writing as YAML, Dump, To and Decode.
func TestExactCarried(t *testing.T) {
	const id = "12345678901234567890"
	for _, mode := range modes[1:] {
		inMode(t, mode.m, func(t *testing.T) {
			s, _ := FromJSON([]byte(`{"seq":` + id + `}`))
			e := s.Get("seq")
			checks := map[string]string{
				"Copy":     e.Copy().Str(),
				"Clone":    s.Clone().Get("seq").Str(),
				"Set":      (&SV{}).Set(e).Str(),
				"Value":    Value(e).Str(),
				"FromJSON": must(FromJSON([]byte(id))).Str(),
				"Hash":     Hash("seq", e).Get("seq").Str(),
				"Uint64":   Value(uint64(12345678901234567890)).Str(),
				"Dump":     strings.TrimSpace(Value(e).Dump()),
				"YAML":     string(must(Value(e).ToYAML())),
			}
			var u uint64
			if err := e.Decode(&u); err != nil {
				checks["Decode uint64"] = err.Error()
			} else {
				checks["Decode uint64"] = big.NewInt(0).SetUint64(u).String()
			}
			b, err := To[*big.Int](e)
			if err != nil {
				checks["To *big.Int"] = err.Error()
			} else {
				checks["To *big.Int"] = b.String()
			}
			for k, got := range checks {
				if !strings.Contains(got, id) {
					t.Errorf("%s %s: %s", mode.name, k, got)
				}
			}
		})
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// TestExactMutatorsClear checks that every way of changing a scalar in
// place leaves no stale digits: the plan's risk, in exactText.
func TestExactMutatorsClear(t *testing.T) {
	const id = "12345678901234567890"
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
		"S":             func(s *SV) { s.S(`0$`, "1") },
		"Tr":            func(s *SV) { s.Tr("0", "9") },
		"SubstrReplace": func(s *SV) { s.SubstrReplace(0, 1, "9") },
		"AV":            func(s *SV) { s.Undef(); s.AV().Push(1) },
		"HV":            func(s *SV) { s.Undef(); s.HV().Set("k", 1) },
		"Apply":         func(s *SV) { s.Apply(func(e *SV) any { return 2.5 }) },
		"UnmarshalJSON": func(s *SV) { _ = json.Unmarshal([]byte("2.5"), s) },
		"UnmarshalYAML": func(s *SV) { v, _ := FromYAML([]byte("2.5")); s.Set(v) },
	}
	for _, mode := range modes[1:] {
		inMode(t, mode.m, func(t *testing.T) {
			for name, f := range muts {
				s := must(FromJSON([]byte("[" + id + "]"))).Get(0)
				func() {
					defer func() { recover() }()
					f(s)
				}()
				if _, isF := s.val().(float64); s.digits() != "" && (!isF || s.Num() != 1.2345678901234567e19) {
					t.Errorf("%s %s: %v keeps stale digits %s", mode.name, name, s.val(), s.digits())
				}
			}
		})
	}
}

// TestExactLikeFloat checks that a whole number kept exactly behaves as
// its float did, except where it's written out: the uv design's risk, a
// type switch that doesn't know uint64.
func TestExactLikeFloat(t *testing.T) {
	const id = "12345678901234567890"
	for _, mode := range modes[1:] {
		inMode(t, mode.m, func(t *testing.T) {
			e := must(FromJSON([]byte("[" + id + "]"))).Get(0)
			var f *SV
			inMode(t, exactOff, func(*testing.T) { f = must(FromJSON([]byte("[" + id + "]"))).Get(0) })
			type probe struct {
				name string
				get  func(s *SV) any
			}
			probes := []probe{
				{"Num", func(s *SV) any { return s.Num() }},
				{"Int", func(s *SV) any { return s.Int() }},
				{"Bool", func(s *SV) any { return s.Bool() }},
				{"Defined", func(s *SV) any { return s.Defined() }},
				{"Ref", func(s *SV) any { return s.Ref() }},
				{"Add", func(s *SV) any { return s.Add(1).Num() }},
				{"Mul", func(s *SV) any { return s.Mul(2).Num() }},
				{"Neg", func(s *SV) any { return s.Neg().Num() }},
				{"NumEq float", func(s *SV) any { return s.NumEq(1.2345678901234567e19) }},
				{"NumGt int", func(s *SV) any { return s.NumGt(math.MaxInt64) }},
				{"NumCmp", func(s *SV) any { return NumCmp(s, Value(1)) }},
				{"Any", func(s *SV) any { return s.Any() }},
				{"Length", func(s *SV) any { return s.Length() }},
				{"Find", func(s *SV) any { return Array(s).Find(`$[?@ > 1]`).Length() }},
				{"Sort", func(s *SV) any { return Array(s, 1).Sort(NumCmp).Get(0).Num() }},
			}
			for _, p := range probes {
				got, want := p.get(e), p.get(f)
				if got != want {
					t.Errorf("%s %s: %v, the float gives %v", mode.name, p.name, got, want)
				}
			}
			if e.Kind() != Int && e.Kind() != Float {
				t.Errorf("%s Kind: %v", mode.name, e.Kind())
			}
		})
	}
}

// TestExactCompare checks that nearby big ids compare exactly, as Perl's
// UVs do, rather than as equal floats.
func TestExactCompare(t *testing.T) {
	for _, mode := range modes[1:] {
		inMode(t, mode.m, func(t *testing.T) {
			s := must(FromJSON([]byte(`[12345678901234567890, 12345678901234567891, 9223372036854775807]`)))
			a, b, m := s.Get(0), s.Get(1), s.Get(2)
			if a.NumEq(b) || !a.NumLt(b) || !m.NumLt(a) || !a.NumGt(m) {
				t.Errorf("%s: %v %v %v %v", mode.name, a.NumEq(b), a.NumLt(b), m.NumLt(a), a.NumGt(m))
			}
		})
	}
}

// TestExactAgainstPerl compares each mode with JSON::PP: decoding a
// number, writing it back, and the string Perl gives it.
func TestExactAgainstPerl(t *testing.T) {
	needPerl(t)
	nums := []string{"12345678901234567890", "18446744073709551615", "18446744073709551616",
		"-9223372036854775808", "-9223372036854775809", "99999999999999999999",
		"100000000000000000000", "123456789012345678901234567890", "9007199254740993"}
	script := `use JSON::PP; my $j = JSON::PP->new; while (<STDIN>) { chomp; my $d = $j->decode("[$_]"); print $j->encode($d), "\t$d->[0]\n" }`
	cmd := exec.Command("perl", "-e", script)
	cmd.Stdin = strings.NewReader(strings.Join(nums, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, mode := range modes {
		inMode(t, mode.m, func(t *testing.T) {
			same := 0
			for i, n := range nums {
				s := must(FromJSON([]byte("[" + n + "]")))
				got := string(must(json.Marshal(s))) + "\t" + s.Get(0).Str()
				if got == lines[i] {
					same++
				} else {
					t.Logf("%s %s: scalar %q, Perl %q", mode.name, n, got, lines[i])
				}
			}
			t.Logf("%s: %d of %d as Perl", mode.name, same, len(nums))
		})
	}
}
