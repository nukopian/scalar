package scalar

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os/exec"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestFromYAML(t *testing.T) {
	doc, err := FromYAML([]byte(`
nulls: [null, ~, ]
empty:
bools: [true, false, True]
ints: [42, -7, 0o17, 0x1F]
big: 99999999999999999999
floats: [2.5, 1e3, .inf, -.inf]
nan: .nan
strings: ["42", '007', no, yes, off, 2026-10-15, "it's", 'two words']
plain: 007
bin: !!binary aGk=
1: one
true: yes please
multi: |
  two
  lines
`))
	if err != nil {
		t.Fatal(err)
	}
	want := MustFromDump(`{
		nulls => [undef, undef], empty => undef,
		bools => [!!1, !!0, !!1],
		ints => [42, -7, 15, 31],
		strings => ["42", "007", "no", "yes", "off", "2026-10-15", "it's", "two words"],
		bin => "\150\151",
		1 => "one", true => "yes please",
		multi => "two\nlines\n",
	}`)
	for _, k := range want.HV().KeysSlice() {
		if d := doc.At("{" + k + "}").Diff(want.At("{" + k + "}")); d != "" {
			t.Errorf("%s: %s", k, d)
		}
	}
	if f := doc.At("floats").AV(); f.Get(0).Num() != 2.5 || f.Get(1).Num() != 1000 || !math.IsInf(f.Get(2).Num(), 1) || !math.IsInf(f.Get(3).Num(), -1) || !math.IsNaN(doc.At("nan").Num()) {
		t.Errorf("infinities and NaN: %s %v", f, doc.At("nan"))
	}
	if doc.At("big").Kind() != Float || doc.At("ints[0]").Kind() != Int || doc.At("bin").Kind() != Bytes || doc.At("strings[5]").Kind() != String {
		t.Error("kinds: big a float, ints integers, !!binary bytes, a date a string")
	}
	if got := doc.At("plain"); got.Kind() != Int || got.Int() != 7 {
		t.Errorf("an unquoted 007 is the integer 7, as YAML 1.2 has it: %s", got.Dump())
	}

	// Anchors, aliases, merge keys and loops.
	doc, err = FromYAML([]byte(`
base: &base {a: 1, b: [x], c: 3}
extra: &extra {c: 30, d: 4}
child:
  <<: [*base, *extra]
  b: [y]
again: *base
name: &n Ada
copy: *n
loop: &loop
  self: *loop
`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.At("again").HV() != doc.At("base").HV() {
		t.Error("an alias should be the same hash as its anchor")
	}
	if d := doc.At("child").Diff(map[string]any{"a": 1, "b": []any{"y"}, "c": 3, "d": 4}); d != "" {
		t.Errorf("merge keys: own keys win, then the earlier mapping: %s", d)
	}
	doc.At("child.a").Set(99)
	if doc.At("base.a").Int() != 1 {
		t.Error("a merged value should be a copy, not the merged mapping's own scalar")
	}
	if doc.At("loop.self").HV() != doc.At("loop").HV() {
		t.Error("an alias to an enclosing mapping should be a loop")
	}
	if doc.At("copy").Str() != "Ada" {
		t.Error("an alias to a plain value")
	}

	// Tags YAML doesn't define are dropped, keeping the value.
	cfn, err := FromYAML([]byte("Bucket: !Ref MyBucket\nArn: !GetAtt Bucket.Arn\nList: !Split [',', 'a,b']\n"))
	if err != nil || cfn.At("Bucket").Str() != "MyBucket" || cfn.At("Arn").Str() != "Bucket.Arn" || cfn.At("List[1]").Str() != "a,b" {
		t.Errorf("custom tags: %v %s", err, cfn.Dump())
	}

	// Empty input, several documents, and errors.
	if e, err := FromYAML(nil); err != nil || e.Defined() {
		t.Errorf("empty input: %v %v", e, err)
	}
	if _, err := FromYAML([]byte("a: 1\n---\nb: 2\n")); err == nil || !strings.Contains(err.Error(), "2 documents; use FromYAMLAll") {
		t.Errorf("two documents: %v", err)
	}
	all, err := FromYAMLAll([]byte("---\nkind: Service\n---\nkind: Deployment\n---\n")) // empty documents skipped
	if err != nil || all.Length() != 2 || all.Get(1).At("kind").Str() != "Deployment" {
		t.Errorf("FromYAMLAll: %v %s", err, all)
	}
	if all, err := FromYAMLAll([]byte("--- ~\n--- null\n")); err != nil || all.Length() != 2 {
		t.Errorf("an explicit null is a document: %v %s", err, all)
	}
	for src, want := range map[string]string{
		"a: [1, 2":           "scalar: bad YAML",
		"? [a, b]\n: 1\n":    "line 1, column 3: a key that's a mapping or a sequence can't be a hash key",
		"a: !!int x\n":       "line 1, column 4 (a)",
		"m: {<<: 1}\n":       "a merge key (<<) must refer to a mapping",
		"b: !!binary '%%'\n": "!!binary isn't valid base64",
	} {
		if _, err := FromYAML([]byte(src)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("FromYAML(%q): %v\n want %q", src, err, want)
		}
	}
}

func TestToYAML(t *testing.T) {
	tags := Scalar(Array("go"))
	h := Hash("title", "Go Developer", "zero", "007", "flag", "true", "no", "no", "f", 1.0, "inf", math.Inf(1),
		"n", nil, "yes", true, "count", 3, "blob", []byte("hi"), "tags", tags, "again", tags, "lines", "a\nb\n")
	h.Set("self", Scalar(h))
	holes := Array(1)
	holes.Set(2, 3)
	h.Set("holes", Scalar(holes))
	out, err := h.ToYAML()
	if err != nil {
		t.Fatal(err)
	}
	want := `&1
again: &2
  - go
blob: !!binary aGk=
count: 3
f: 1.0
flag: "true"
holes:
  - 1
  - null
  - 3
inf: .inf
lines: |
  a
  b
"n": null
"no": "no"
self: *1
tags: *2
title: Go Developer
"yes": true
zero: "007"
`
	if string(out) != want {
		t.Errorf("ToYAML:\n%s\nwant:\n%s", out, want)
	}
	back, err := FromYAML(out)
	if err != nil || !back.Equal(h) || back.At("self").HV() != back.HV() || back.At("tags").AV() != back.At("again").AV() {
		t.Errorf("read back: %v\n%s", err, back.Dump())
	}

	// In a struct, with the YAML package itself.
	type cfg struct {
		Name  string `yaml:"name"`
		Extra *SV    `yaml:"extra"`
	}
	var c cfg
	if err := yaml.Unmarshal([]byte("name: web\nextra: {replicas: 3, ports: [80, 443]}\n"), &c); err != nil || c.Extra.At("ports[1]").Int() != 443 {
		t.Errorf("UnmarshalYAML: %v %v", err, c.Extra)
	}
	if out, err := yaml.Marshal(c); err != nil || !strings.Contains(string(out), "replicas: 3") {
		t.Errorf("MarshalYAML: %v\n%s", err, out)
	}
	if out, err := yaml.Marshal(map[string]any{"a": Array(1), "h": Hash("k", "v")}); err != nil || string(out) != "a:\n    - 1\nh:\n    k: v\n" {
		t.Errorf("AV and HV MarshalYAML: %v\n%q", err, out)
	}
	// Strings YAML 1.1 readers would take as something else are quoted.
	for _, v := range []string{"y", "N", "yes", "Off", "on", "NULL", "~", "1_000", "1:30", "0b101", "0777", "0x1F", ".5", "1e3", "-.inf", ".NaN"} {
		out, _ := Scalar(v).ToYAML()
		if string(out) != `"`+v+`"`+"\n" {
			t.Errorf("ToYAML(%q) = %q", v, out)
		}
	}
	for _, v := range []string{"yesterday", "Norway", "a:b", "v1.2.3", "1.2.3"} {
		if out, _ := Scalar(v).ToYAML(); string(out) != v+"\n" {
			t.Errorf("ToYAML(%q) = %q: shouldn't be quoted", v, out)
		}
	}
	if out, _ := Scalar("caf\xe9").ToYAML(); string(out) != "!!binary Y2Fm6Q==\n" {
		t.Errorf("text that isn't UTF-8 goes as binary: %q", out)
	}
}

// ToYAML's output reads back with FromYAML as an equal structure that dumps
// the same, shared parts and loops included, for hundreds of random
// structures.
func TestYAMLRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 2))
	for i := range 500 {
		orig := randomStructure(rng)
		out, err := orig.ToYAML()
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		back, err := FromYAML(out)
		if err != nil {
			t.Fatalf("case %d: %v\n%s", i, err, out)
		}
		if typedDump(back) != typedDump(orig) {
			t.Fatalf("case %d reads back differently:\n%s\nwas:\n%s\nYAML:\n%s", i, typedDump(back), typedDump(orig), out)
		}
	}
}

// FromYAML reads what Perl's YAML::PP writes, and YAML::PP reads what ToYAML
// writes, for hundreds of random structures. It needs perl and YAML::PP, and
// is skipped without them.
func TestYAMLMatchesPerl(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl isn't installed")
	}
	if exec.Command("perl", "-MYAML::PP", "-e1").Run() != nil {
		t.Skip("YAML::PP isn't installed (cpanm YAML::PP, or set PERL5LIB)")
	}
	rng := rand.New(rand.NewPCG(17, 9))
	type tcase struct {
		JSON string `json:"json"`
		YAML string `json:"yaml"` // written by ToYAML
	}
	var cases []tcase
	for len(cases) < 200 {
		s := randomStructure(rng)
		js, err := s.MarshalJSON()
		if err != nil {
			continue
		}
		if hasBytesOrFloat(s) {
			continue // JSON can't carry bytes, and floats compare better without JSON's rounding
		}
		y, err := s.ToYAML()
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, tcase{string(js), string(y)})
	}
	in, _ := json.Marshal(cases)
	const script = `
use v5.36; use JSON::PP; use YAML::PP;
my $json = JSON::PP->new->canonical->boolean_values(builtin::false, builtin::true);
my $yp = YAML::PP->new(schema => ["Core"], boolean => "perl");
my @out;
for my $c (@{ decode_json(do { local $/; <STDIN> }) }) {
	my $data = $json->decode($c->{json});
	my $loaded = $yp->load_string($c->{yaml});
	push @out, [$yp->dump_string($data), $json->encode([$loaded])];
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
	var results [][2]string
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("perl's output: %v", err)
	}
	for i, c := range cases {
		want, _ := FromJSON([]byte(c.JSON))
		// Perl wrote it; Go reads it.
		got, err := FromYAML([]byte(results[i][0]))
		if err != nil {
			t.Errorf("case %d, reading YAML::PP's output: %v\n%s", i, err, results[i][0])
		} else if typedDump(got) != typedDump(want) {
			t.Errorf("case %d, reading YAML::PP's output: %s\n%s\nwant:\n%s", i, results[i][0], typedDump(got), typedDump(want))
		}
		// Go wrote it; Perl reads it.
		loaded, _ := FromJSON([]byte(results[i][1]))
		if typedDump(loaded.At("[0]")) != typedDump(want) {
			t.Errorf("case %d, YAML::PP reading ToYAML's output:\n%s\n%s\nwant:\n%s", i, c.YAML, typedDump(loaded.At("[0]")), typedDump(want))
		}
	}
}

// hasBytesOrFloat reports whether s holds a byte string or a float anywhere.
func hasBytesOrFloat(s *SV) bool {
	found := false
	s.Clone().Apply(func(v *SV) any {
		if k := v.Kind(); k == Bytes || k == Float {
			found = true
		}
		return v
	})
	return found
}
