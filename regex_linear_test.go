package scalar

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// withEngine runs f with DefaultEngine set to e, and warnings collected
// rather than logged.
func withEngine(e Engine, f func()) (warnings []string) {
	defer func(e Engine, w func(string)) { DefaultEngine, RegexWarn = e, w }(DefaultEngine, RegexWarn)
	DefaultEngine = e
	RegexWarn = func(msg string) { warnings = append(warnings, msg) }
	f()
	return warnings
}

// regexOps runs every regex operation with pattern on text, and returns
// what each gave, panics included, as text to compare.
func regexOps(pattern, flags, text string, isBytes bool) string {
	sv := func() *SV {
		if isBytes {
			return Value([]byte(text))
		}
		return Value(text)
	}
	var b strings.Builder
	op := func(name string, f func() string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(&b, "%s: panic %v\n", name, r)
			}
		}()
		fmt.Fprintf(&b, "%s: %s\n", name, f())
	}
	op("M", func() string { return sv().M(pattern, flags).Dump() })
	op("Mg", func() string { return sv().M(pattern, flags+"g").Dump() })
	op("Matches", func() string {
		var all []string
		for m := range sv().Matches(pattern, flags) {
			if all = append(all, m.Dump()); len(all) == 5 {
				break
			}
		}
		return strings.Join(all, " ")
	})
	op("Named", func() string {
		h := sv().Named(pattern, flags)
		if h == nil {
			return "nil"
		}
		return h.Dump()
	})
	for _, repl := range []string{"<$&>", "[$1|${2}|$+|$`|$']", "${x}"} {
		op("S "+repl, func() string { return sv().S(pattern, repl, flags+"r").Dump() })
		op("S g "+repl, func() string { return sv().S(pattern, repl, flags+"gr").Dump() })
	}
	op("S func", func() string {
		return sv().S(pattern, func(m *AV) string { return fmt.Sprint(m.Length()) }, flags+"gr").Dump()
	})
	op("S count", func() string { s := sv(); return s.S(pattern, "-", flags+"g").Str() + " " + s.Dump() })
	if flags == "" {
		op("Split", func() string { return sv().Split(pattern).Dump() })
		op("Split 2", func() string { return sv().Split(pattern, 2).Dump() })
		op("Split -1", func() string { return sv().Split(pattern, -1).Dump() })
	}
	return b.String()
}

// linearDiffers runs the operations on both engines, and reports any
// difference.
func linearDiffers(t *testing.T, pattern, flags, text string, isBytes bool) {
	t.Helper()
	var back, lin string
	withEngine(Backtracking, func() { back = regexOps(pattern, flags, text, isBytes) })
	withEngine(Linear, func() { lin = regexOps(pattern, flags, text, isBytes) })
	// A timeout depends on the machine's load, not the pattern alone, so a
	// match on Backtracking may time out in one run and not the other.
	if strings.Contains(back, "match timeout") || strings.Contains(lin, "match timeout") {
		return
	}
	if back != lin {
		t.Errorf("pattern %q, flags %q, text %q, bytes %v:\nBacktracking:\n%s\nLinear:\n%s", pattern, flags, text, isBytes, back, lin)
	}
}

var linearPatterns = []string{
	// The everyday.
	`^M (\d+\.\d) - (\d+) km (\w+) of ([^,]+), (\w\w)`, `depth (\d+(?:\.\d+)?) km`,
	`(\d+) km`, `net (?<net>\w+)`, `(?<mag>\d\.\d)`, `\s*\|\s*`, `,`, `(,)`, ` `, ``,
	`(a)(x)?(b)`, `a|ab`, `.+?,`, `a.b`, `^b$`, `\bCA\b`, `\Bri`, `[A-Z]{2}$`,
	// Classes and their Unicode.
	`\w+`, `\W+`, `\d+`, `\D+`, `\s`, `\S+`, `[\w-]+`, `[^\w\s]+`, `[\W]`, `[\S]+`, `[\D]`,
	`\p{Lu}\p{Ll}+`, `\p{Greek}+`, `\PL+`, `[a-z-]`, `[-a]`, `[^a-c]+`, `[\]\-]`, `\x41\x{e9}`,
	`é+`, `[é-ü]`, `\.`, `\$\d+`,
	// Ends of lines, and m.
	`CA$`, `$`, `^`, `\A\w`, `\w\z`, `(?m)^\w+$`, `(?m)$`, `(?m:a$)|b$`,
	// Groups and their numbers.
	`(?<y>\d+)-(\d+)`, `(?<a>a)|(?<b>b)`, `(a)|b`, `(?:(a)|b)+`, `((a)(b))?`,
	// Empty matches.
	`x*`, `a|`, `\d*`, `(?:)`, `\b`, `a??`,
	// i.
	`(?i)ſ`, `(?i)İ`, `(?i)\x{212a}`, `(?i)[\x{100}-\x{200}]`, `(?i)[é-ü]`, `(?i)σ+`, `(?i)Σ`,
	`(?i)s`, `(?i)k`, `(?i)[a-z]+`, `(?i)[^a-z]+`, `(?i:ca) ok`, `(?i)été`, `(?i)\w+`, `(?i)i`,
	// What Linear can't run, so both run on Backtracking.
	`(?<=\$)\d+`, `\b(\w+) \1\b`, `(?>a+)b`, `CA\Z`, `[[:alpha:]]+`, `[a-z-[aeiou]]+`,
	`(a?)*`, `(a|)*b`, `(a*)+`, `\G\w`,
}

var linearTexts = []string{
	"", "a", "ab", "aab", "axb", "a1", "abc", "a,b,,,", "  a b  c ", "hello hello",
	quakeLine, quakeLine + "\n", "Ridgecrest, CA\n", "Ridgecrest, CA", "a\nb\n", "price: $100",
	"Zürich", "été ici", "depth ٣٤ km", "a\u00a0b", "a\u2003b\u0085c", "x\u200dy", "ſ S s", "İ i I",
	"\u212a k", "Σίσυφος", "a\xffb", "\xe9t\xe9", "near Ōtaki", "\tA\v",
}

// Linear gives Backtracking's answer for every pattern on every text.
func TestLinearMatchesBacktracking(t *testing.T) {
	for _, pattern := range linearPatterns {
		for _, text := range linearTexts {
			linearDiffers(t, pattern, "", text, false)
			for _, flags := range []string{"i", "m", "s", "ims"} {
				linearDiffers(t, pattern, flags, text, false)
			}
		}
	}
	for _, text := range []string{"\u0089PNG\r\n", "\u00e9t\u00e9", "a\u00ffb\n"} {
		for _, pattern := range []string{`^\x89(PNG)`, `\w+`, `\W`, `\b.`, `(?i)\xe9`, `.$`} {
			linearDiffers(t, pattern, "", text, true)
		}
	}
}

// Random patterns, built from pieces Linear takes and some it doesn't,
// give the same answers on both engines.
func TestLinearMatchesBacktrackingRandom(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	atoms := []string{
		"a", "b", "é", "s", "K", "ſ", "İ", ".", `\w`, `\W`, `\d`, `\s`, `\S`, `\b`, `\B`, "^", "$", `\z`,
		"[ab]", "[^a]", `[\w,]`, `[a-c]`, `\p{L}`, `\.`, ",", " ", `\n`, `\x{e9}`,
		"(a)", "(?:ab)", "(?<n>b)", "(a|)", "(?i)", "(?m)", "(?s)", "(?i:s)", "(?=a)", `\1`,
	}
	quants := []string{"", "", "", "*", "+", "?", "{2}", "{1,2}", "*?", "+?", "??"}
	texts := []string{"ab", "aab\n", "a b,c", "ſ sK\u212a", "été\nab", "İi", "a\u00a0\u200db", "x\xff"}
	for range 4000 {
		var p strings.Builder
		for range 1 + rng.IntN(5) {
			a := atoms[rng.IntN(len(atoms))]
			if rng.IntN(6) == 0 {
				a = "(" + a + "|" + atoms[rng.IntN(len(atoms))] + ")"
			}
			p.WriteString(a + quants[rng.IntN(len(quants))])
		}
		if _, err := Compile(p.String()); err != nil {
			continue
		}
		flags := []string{"", "i", "m", "s"}[rng.IntN(4)]
		linearDiffers(t, p.String(), flags, texts[rng.IntN(len(texts))], false)
	}
}

func FuzzLinear(f *testing.F) {
	for _, p := range linearPatterns {
		f.Add(p, quakeLine, uint8(0))
	}
	f.Add(`\b\w`, "été ici", uint8(0))
	f.Add(`s$`, "ſ\n", uint8(1))
	f.Fuzz(func(t *testing.T, pattern, text string, flags uint8) {
		// regexOps expands $` and $' at every match, so its output grows as
		// the square of the text: 8 KB of matches took a gigabyte.
		if len(text) > 1024 {
			return
		}
		if _, err := Compile(pattern); err != nil {
			return
		}
		linearDiffers(t, pattern, []string{"", "i", "m", "s"}[flags%4], text, false)
	})
}

// What Linear takes, and what it gives as the reason it can't.
func TestLinearTakes(t *testing.T) {
	for _, pattern := range []string{
		`^M (\d+\.\d) - (\d+) km`, `(?<net>\w+)`, `\s*\|\s*`, `[\w.-]+@[\w-]+`, `CA$`, `\bCA\b`,
		`(?i)ridgecrest`, `\p{Lu}`, `(?m)^\w+$`, `x*`, `(a)|b`, `\x{e9}+`, `a{2,3}?`,
	} {
		if _, err := CompileWith(Linear, pattern); err != nil {
			t.Errorf("Linear refused %q: %v", pattern, err)
		}
	}
	for pattern, reason := range map[string]string{
		`(?<=\$)\d+`:            "lookbehind",
		`(?!x)`:                 "lookahead",
		`(\w+) \1`:              "backreference",
		`\k<x>(?<x>a)`:          "named backreference",
		`(?>a+)`:                "atomic group",
		`CA\Z`:                  `\Z`,
		`\G`:                    `\G`,
		`[[:alpha:]]`:           "POSIX class",
		`[a-z-[aeiou]]`:         "class subtraction",
		`(a?)*`:                 "can match nothing",
		`(?n)a`:                 `inline flag 'n'`,
		`\e`:                    `\e`,
		`(?i)\p{Lu}`:            `\p or \P with the i flag`,
		`(?i)[\x{100}-\x{200}]`: "fold differently",
		`\p{White_Space}`:       "doesn't know",
		`a{1001}`:               "Go's regexp rejects",
		`(?#note)a`:             "comment group",
		`(?<a>x)|(?<a>y)`:       "used twice",
		`a{,2}`:                 "repeat count",
		`[a-c-e]`:               "- Linear can't place",
		`(a)?(?(1)c|d)`:         "conditional",
	} {
		_, err := CompileWith(Linear, pattern)
		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("CompileWith(Linear, %q) = %v, want an error about %s", pattern, err, reason)
		}
	}
	if _, err := CompileWith(Linear, `a b`, "x"); err == nil || !strings.Contains(err.Error(), "the x flag") {
		t.Errorf(`CompileWith(Linear, "a b", "x") = %v`, err)
	}
	if _, err := CompileWith(Linear, `(`); err == nil || !strings.Contains(err.Error(), "bad regex") {
		t.Errorf(`CompileWith(Linear, "(") = %v, want Backtracking's error`, err)
	}
}

// When a match goes to Backtracking.
func TestLinearHandsOver(t *testing.T) {
	for _, c := range []struct {
		pattern, flags, text string
		g, use               bool
	}{
		{`\w+`, "", "Zürich", false, true},
		{`CA$`, "", "Ridgecrest, CA", false, true},
		{`CA$`, "", "Ridgecrest, CA\n", false, false},
		{`CA$`, "m", "Ridgecrest, CA\n", false, true},
		{`\bé`, "", "café", false, false},
		{`\bc`, "", "café", false, false},
		{`\bc`, "", "cafe", false, true},
		{`s`, "i", "Straße", false, true},
		{`s`, "i", "ſ", false, false},
		{`i`, "i", "İ", false, false},
		{`k`, "i", "\u212a", false, false},
		{`été`, "i", "ÉTÉ", false, false},
		{`été`, "i", "ete", false, true},
		{`x*`, "", "axb", false, true},
		{`x*`, "", "axb", true, false},
		{`a`, "", "a\xffb", false, false},
		{`a`, "", "a\xffb", true, false},
	} {
		re := MustCompileWith(Linear, c.pattern, c.flags)
		if got := re.lin.use(c.text, c.g); got != c.use {
			t.Errorf("%q with %q on %q, g %v: Linear %v, want %v", c.pattern, c.flags, c.text, c.g, got, c.use)
		}
	}
}

// The regex tests pass on Linear too, except the timeout's, which Linear
// doesn't need.
func TestRegexOnLinear(t *testing.T) {
	warnings := withEngine(Linear, func() {
		for name, test := range map[string]func(*testing.T){
			"MatchAsCondition":          TestMatchAsCondition,
			"Match":                     TestMatch,
			"MatchGlobal":               TestMatchGlobal,
			"Named":                     TestNamed,
			"Subst":                     TestSubst,
			"Split":                     TestSplit,
			"RegexPanics":               TestRegexPanics,
			"CompiledRegexp":            TestCompiledRegexp,
			"Compile":                   TestCompile,
			"SubstPlainReplacement":     TestSubstPlainReplacement,
			"SubstExpandMatchesRegexp2": TestSubstExpandMatchesRegexp2,
			"SubstExpandMatchesPerl":    TestSubstExpandMatchesPerl,
			"SubstExpandLeftAlone":      TestSubstExpandLeftAlone,
		} {
			t.Run(name, test)
		}
	})
	t.Logf("%d patterns ran on Backtracking", len(warnings))
}

// A pattern Linear can't run warns once, and runs on Backtracking.
func TestRegexWarn(t *testing.T) {
	linearCache = boundedCache[linearKey, *linearCompiled]{limit: cacheLimit} // warned of nothing yet
	pattern := `(?<=\$)(\d+)\.(\d+) warn-once`
	warnings := withEngine(Linear, func() {
		for range 3 {
			if got := Value("$4.95 warn-once").M(pattern).Get(2).Str(); got != "95" {
				t.Errorf("M = %q, want 95", got)
			}
		}
		MustCompile(pattern)
	})
	want := "scalar: regex \"(?<=\\\\$)(\\\\d+)\\\\.(\\\\d+) warn-once\" uses lookbehind, which Linear can't run, so it runs on Backtracking"
	if len(warnings) != 1 || warnings[0] != want {
		t.Errorf("warnings = %q, want one: %s", warnings, want)
	}
	withEngine(Linear, func() {
		RegexWarn = nil
		Value("x").M(`(?<=y)x nil-warn`) // no panic
	})
}

// A *Regexp keeps the engine it was compiled with.
func TestRegexpKeepsEngine(t *testing.T) {
	var fromDefault *Regexp
	withEngine(Linear, func() {
		if MustCompileWith(Backtracking, `\d`).lin != nil {
			t.Error("Backtracking pattern took Linear from DefaultEngine")
		}
		fromDefault = MustCompile(`\d`)
	})
	if fromDefault.lin == nil {
		t.Error("MustCompile under DefaultEngine Linear didn't take Linear")
	}
	if MustCompileWith(Linear, `\d`).lin == nil {
		t.Error("MustCompileWith(Linear) didn't take Linear")
	}
	if MustCompile(`\d`).lin != nil {
		t.Error("MustCompile took Linear with DefaultEngine Backtracking")
	}
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "unknown engine") {
			t.Errorf("CompileWith(Engine(7)) = %v", r)
		}
	}()
	CompileWith(Engine(7), "a")
}

// MustCompileWith panics where CompileWith gives an error.
func TestMustCompileWithPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "Linear can't run") {
			t.Errorf("MustCompileWith(Linear, lookbehind) = %v", r)
		}
	}()
	MustCompileWith(Linear, `(?<=a)b`)
}
