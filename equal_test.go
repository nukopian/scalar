package scalar

import "testing"

// The cases from Test::More's is_deeply, with the results Perl gives.
func TestEqualMatchesIsDeeply(t *testing.T) {
	holey := Array(1)
	holey.Set(2, 3)
	loop1, loop2 := Hash(), Hash()
	loop1.Set("s", Value(loop1))
	loop2.Set("s", Value(loop2))
	for _, c := range []struct {
		got, want any
		diff      string
	}{
		{nil, "", `got undef, want ""`},
		{1, "1.0", `got 1, want "1.0"`},
		{1.0, 1, ""},
		{true, 1, ""},
		{false, "", ""},
		{"\u00E9", []byte{0xe9}, ""},  // a byte string equals text of the same characters
		{Array(1, nil, 3), holey, ""}, // a hole is undef
		{Array(1), Array(1, nil), `[1]: got does not exist, want undef`},
		{Hash("a", nil), Hash(), `a: got undef, want does not exist`},
		{loop1, loop2, ""},
		{map[string]any{"a": []any{1, map[string]any{"b": 2}}}, map[string]any{"a": []any{1, map[string]any{"b": 3}}}, `a[1].b: got 2, want 3`},
		{Array(1, 2), Array(1), `[1]: got 2, want does not exist`},
		{Hash(), Array(), `got a hash, want an array`},
		{Array(1), "ARRAY", `got an array, want "ARRAY"`},
	} {
		got := Value(c.got)
		if d := got.Diff(c.want); d != c.diff {
			t.Errorf("%s vs %v: Diff %q, want %q", got.Dump(), c.want, d, c.diff)
		}
		if got.Equal(c.want) != (c.diff == "") {
			t.Errorf("%s vs %v: Equal disagrees with Diff", got.Dump(), c.want)
		}
	}
}

func TestDiff(t *testing.T) {
	doc, _ := FromJSON([]byte(`{"place": "Cobb", "stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "x"}}`))
	for want, diff := range map[string]string{
		`{"place": "Cobb", "stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "x"}}`:         "",
		`{"place": "Anza", "stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "x"}}`:         `place: got "Cobb", want "Anza"`,
		`{"place": "Cobb", "stations": ["PAS"], "meta": {"log.level": "debug", "0": "x"}}`:                `stations[1]: got "GSC", want does not exist`,
		`{"place": "Cobb", "stations": ["PAS", "GSC"], "meta": {"log.level": "info", "0": "x"}}`:          `meta{log.level}: got "debug", want "info"`,
		`{"place": "Cobb", "stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "y"}}`:         `meta{0}: got "x", want "y"`,
		`{"stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "x"}}`:                          `place: got "Cobb", want does not exist`,
		`{"a": 1, "place": "Cobb", "stations": ["PAS", "GSC"], "meta": {"log.level": "debug", "0": "x"}}`: `a: got does not exist, want 1`,
	} {
		other, err := FromJSON([]byte(want))
		if err != nil {
			t.Fatal(err)
		}
		if d := doc.Diff(other); d != diff {
			t.Errorf("Diff = %q, want %q", d, diff)
		}
	}

	// The path from Diff leads At to the value; Go values compare too.
	if !doc.At("meta{log.level}").Equal("debug") || !doc.At("stations").Equal([]string{"PAS", "GSC"}) {
		t.Error("Equal with Go values")
	}
	if !doc.HV().Equal(doc.Clone()) || !doc.At("stations").AV().Equal(Array("PAS", "GSC")) {
		t.Error("Equal on arrays and hashes")
	}
	if d := doc.At("stations").AV().Diff([]any{"PAS", 3}); d != `[1]: got "GSC", want 3` {
		t.Errorf("AV.Diff = %q", d)
	}
	if d := doc.HV().Diff(nil); d != `got a hash, want undef` {
		t.Errorf("HV.Diff(nil) = %q", d)
	}
	var nilSV *SV
	if !nilSV.Equal(nil) || Value(nil).Equal(0) {
		t.Error("undef equals only undef")
	}
}
