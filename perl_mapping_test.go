package scalar_test

// The phrasebook in docs/PERL.md: each task done in Perl, with scalar and
// in plain Go, on the same feed. TestMappingMatchesPerl checks that all
// three give the same answer, so the page can't drift from what the code
// does. Keep the two in step when changing either.

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nukopian/scalar"
)

const mappingFeed = `{"features": [
	{"id": "ci40123456", "properties": {"mag": 5.1, "place": "12 km NNE of Ridgecrest, CA", "felt": null, "net": "ci",
		"stations": " PAS, GSC ,pas,ISA"}},
	{"id": "nc75012345", "properties": {"mag": 2.3, "place": "5 km W of Cobb, CA", "felt": 12, "net": "nc"}},
	{"id": "us7000abcd", "properties": {"mag": 4.7, "place": null, "net": "us"}},
	{"id": "ak0251xyz", "properties": {"mag": 4.9, "place": "Kodiak Island region, Alaska", "felt": 3, "net": "ak"}}
]}`

// What every version must give, task by task.
var mappingWant = map[string]string{
	"default":     "unknown",
	"missing":     "null missing",
	"unique":      "PAS,GSC,ISA",
	"figures":     "12431 1204",
	"filter":      "12 km NNE of Ridgecrest, CA; unknown; Kodiak Island region, Alaska",
	"sort":        "ci40123456,ak0251xyz,us7000abcd,nc75012345",
	"count":       "2=1 4=2 5=1",
	"totals":      "5.1 15",
	"interpolate": "M5.1 near 12 km NNE of Ridgecrest, CA",
	"substitute":  "Ridgecrest, CA",
	"slice":       "5.1 ci",
	"deref":       "4 felt,mag,net,place,stations",
	"probe":       "none felt,mag,net,place,stations",
}

// mappingPerl is each task in Perl, printing "task<TAB>answer".
const mappingPerl = `
use v5.36;
use JSON::PP;
use List::Util qw(max sum0);
my $feed = decode_json(do { local $/; <STDIN> });
my @quakes = @{ $feed->{features} };
my $p = $quakes[0]{properties};
my %out;

$out{default} = $feed->{features}[2]{properties}{place} // "unknown";

$out{missing} = join " ", map { !exists $_->{felt} ? "missing" : !defined $_->{felt} ? "null" : $_->{felt} }
    $quakes[0]{properties}, $quakes[2]{properties};

my %seen;
$out{unique} = join ",", grep { !$seen{fc $_}++ } map { s/^\s+|\s+$//gr } split /,/, $p->{stations};

my $text = "Felt by 12,431 people in 1,204 ZIP codes";
$out{figures} = join " ", map { tr/,//dr } $text =~ /(\d[\d,]*)/g;

$out{filter} = join "; ", map { $_->{properties}{place} // "unknown" }
    grep { $_->{properties}{mag} >= 4.5 } @quakes;

$out{sort} = join ",", map { $_->{id} }
    sort { $b->{properties}{mag} <=> $a->{properties}{mag} } @quakes;

my %count;
$count{int $_->{properties}{mag}}++ for @quakes;
$out{count} = join " ", map { "$_=$count{$_}" } sort keys %count;

$out{totals} = join " ", max(map { $_->{properties}{mag} } @quakes),
    sum0(map { $_->{properties}{felt} // 0 } @quakes);

$out{interpolate} = "M$p->{mag} near $p->{place}";

$out{substitute} = $p->{place} =~ s/^\d+ km \w+ of //r;

$out{slice} = join " ", @{$p}{qw(mag net)};

$out{deref} = scalar(@{ $feed->{features} }) . " " . join ",", sort keys %{ $feed->{features}[0]{properties} };

# The CPAN pragma autovivification isn't in Perl's core; without it, this
# task is left out, and the test says Perl wasn't compared.
if (eval { require autovivification; 1 }) {
	eval q{
		no autovivification;
		$out{probe} = ($p->{products}{shakemap}[0]{code} // "none") . " " . join ",", sort keys %$p;
		1;
	} or die $@;
}

say "$_\t$out{$_}" for sort keys %out;
`

// mappingScalar is each task with scalar.
func mappingScalar(feed *scalar.SV) map[string]string {
	out := map[string]string{}
	p := feed.At("features[0].properties")

	out["default"] = feed.At("features[2].properties.place").Or("unknown").Str()

	var felt []string
	for _, props := range []*scalar.SV{p, feed.At("features[2].properties")} {
		switch v, ok := props.AtOK("felt"); {
		case !ok:
			felt = append(felt, "missing")
		case !v.Defined():
			felt = append(felt, "null")
		default:
			felt = append(felt, v.Str())
		}
	}
	out["missing"] = strings.Join(felt, " ")

	seen := scalar.Hash()
	out["unique"] = p.At("stations").Split(",").
		Map(func(s *scalar.SV) any { return s.Trim() }).
		Grep(func(s *scalar.SV) bool { return seen.Get(s.Fc().Str()).Inc().Int() == 1 }).
		Join(",")

	text := scalar.Value("Felt by 12,431 people in 1,204 ZIP codes")
	figures := scalar.Array()
	for m := range text.Matches(`(\d[\d,]*)`) {
		figures.Push(m.Get(1).S(`,`, "", "gr"))
	}
	out["figures"] = figures.Join(" ")

	out["filter"] = feed.Find("$.features[?@.properties.mag >= 4.5]").
		Map(func(q *scalar.SV) any { return q.At("properties.place").Or("unknown") }).
		Join("; ")

	out["sort"] = feed.At("features").AV().
		RevSortByNum(func(q *scalar.SV) any { return q.At("properties.mag") }).
		Map(func(q *scalar.SV) any { return q.At("id") }).
		Join(",")

	count := scalar.Hash()
	for _, q := range feed.At("features").AV().All() {
		count.Get(fmt.Sprint(q.At("properties.mag").Int())).Inc()
	}
	out["count"] = count.Keys().Sort(nil).
		Map(func(k *scalar.SV) any { return k.Str() + "=" + count.Get(k.Str()).Str() }).
		Join(" ")

	out["totals"] = feed.Find("$..mag").MaxNum().Str() + " " + feed.Find("$..felt").Sum0().Str()

	out["interpolate"] = p.Interpolate("M$mag near $place")

	out["substitute"] = p.At("place").S(`^\d+ km \w+ of `, "", "r").Str()

	out["slice"] = p.HV().Values("mag", "net").Join(" ")

	out["deref"] = fmt.Sprint(feed.Get("features").AV().Length(), " ",
		feed.Get("features", 0, "properties").HV().Keys().Sort(nil).Join(","))

	p.Autovivify(false)
	out["probe"] = fmt.Sprint(p.At("products.shakemap[0].code").Or("none"), " ", p.HV().Keys().Sort(nil).Join(","))

	return out
}

// plainFeed and plainQuake are the plain Go version's types for the feed.
type plainFeed struct {
	Features []plainQuake `json:"features"`
}

type plainQuake struct {
	ID         string `json:"id"`
	Properties struct {
		Mag      float64 `json:"mag"`
		Place    *string `json:"place"`
		Felt     *int    `json:"felt"`
		Net      string  `json:"net"`
		Stations string  `json:"stations"`
	} `json:"properties"`
}

var (
	figureRe   = regexp.MustCompile(`(\d[\d,]*)`)
	distanceRe = regexp.MustCompile(`^\d+ km \w+ of `)
)

// mappingPlain is each task in plain Go. missing reads the raw JSON, as a
// struct can't tell a missing field from a null one.
func mappingPlain(feed plainFeed, raw map[string]any) map[string]string {
	out := map[string]string{}
	p := feed.Features[0].Properties

	place := "unknown"
	if pl := feed.Features[2].Properties.Place; pl != nil {
		place = *pl
	}
	out["default"] = place

	var felt []string
	features := raw["features"].([]any)
	for _, i := range []int{0, 2} {
		props := features[i].(map[string]any)["properties"].(map[string]any)
		switch v, ok := props["felt"]; {
		case !ok:
			felt = append(felt, "missing")
		case v == nil:
			felt = append(felt, "null")
		default:
			felt = append(felt, fmt.Sprint(v))
		}
	}
	out["missing"] = strings.Join(felt, " ")

	seen := map[string]bool{}
	var unique []string
	for _, s := range strings.Split(p.Stations, ",") {
		s = strings.TrimSpace(s)
		if key := strings.ToLower(s); !seen[key] {
			seen[key] = true
			unique = append(unique, s)
		}
	}
	out["unique"] = strings.Join(unique, ",")

	text := "Felt by 12,431 people in 1,204 ZIP codes"
	var figures []string
	for _, m := range figureRe.FindAllStringSubmatch(text, -1) {
		figures = append(figures, strings.ReplaceAll(m[1], ",", ""))
	}
	out["figures"] = strings.Join(figures, " ")

	var places []string
	for _, q := range feed.Features {
		if q.Properties.Mag >= 4.5 {
			place := "unknown"
			if q.Properties.Place != nil {
				place = *q.Properties.Place
			}
			places = append(places, place)
		}
	}
	out["filter"] = strings.Join(places, "; ")

	quakes := slices.Clone(feed.Features)
	slices.SortStableFunc(quakes, func(a, b plainQuake) int {
		return cmp.Compare(b.Properties.Mag, a.Properties.Mag)
	})
	var ids []string
	for _, q := range quakes {
		ids = append(ids, q.ID)
	}
	out["sort"] = strings.Join(ids, ",")

	count := map[int]int{}
	for _, q := range feed.Features {
		count[int(q.Properties.Mag)]++
	}
	var counts []string
	for _, k := range slices.Sorted(maps.Keys(count)) {
		counts = append(counts, fmt.Sprintf("%d=%d", k, count[k]))
	}
	out["count"] = strings.Join(counts, " ")

	maxMag, sumFelt := 0.0, 0
	for _, q := range feed.Features {
		maxMag = max(maxMag, q.Properties.Mag)
		if q.Properties.Felt != nil {
			sumFelt += *q.Properties.Felt
		}
	}
	out["totals"] = fmt.Sprint(maxMag, " ", sumFelt)

	out["interpolate"] = fmt.Sprintf("M%v near %s", p.Mag, *p.Place)

	out["substitute"] = distanceRe.ReplaceAllString(*p.Place, "")

	out["slice"] = fmt.Sprint(p.Mag, " ", p.Net)

	props := raw["features"].([]any)[0].(map[string]any)["properties"].(map[string]any)
	out["deref"] = fmt.Sprint(len(feed.Features), " ", strings.Join(slices.Sorted(maps.Keys(props)), ","))

	code := "none"
	products, _ := props["products"].(map[string]any)
	if shakemap, _ := products["shakemap"].([]any); len(shakemap) > 0 {
		if c, ok := shakemap[0].(map[string]any)["code"].(string); ok {
			code = c
		}
	}
	out["probe"] = fmt.Sprint(code, " ", strings.Join(slices.Sorted(maps.Keys(props)), ","))

	return out
}

func TestMappingMatchesPerl(t *testing.T) {
	feed, err := scalar.FromJSON([]byte(mappingFeed))
	if err != nil {
		t.Fatal(err)
	}
	var typed plainFeed
	var raw map[string]any
	if err := json.Unmarshal([]byte(mappingFeed), &typed); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(mappingFeed), &raw); err != nil {
		t.Fatal(err)
	}
	// The page's last advice: once the shape is settled, To gives the
	// plain Go type, as encoding/json would.
	if converted, err := scalar.To[plainFeed](feed); err != nil || !reflect.DeepEqual(converted, typed) {
		t.Errorf("To[plainFeed] = %+v, %v; want %+v", converted, err, typed)
	}
	// And the note on case folding: Fc folds ß to ss, as Perl's fc does.
	if got := scalar.Value("Straße").Fc().Str(); got != "strasse" {
		t.Errorf(`Fc("Straße") = %q`, got)
	}

	versions := map[string]map[string]string{
		"scalar":   mappingScalar(feed),
		"plain Go": mappingPlain(typed, raw),
	}

	// Perl joins in where it can run; scalar and plain Go are compared
	// regardless.
	if why := scalar.PerlUnusable(); why != "" {
		t.Log("Perl not compared:", why)
	} else {
		cmd := exec.Command("perl", "-e", mappingPerl)
		cmd.Stdin = strings.NewReader(mappingFeed)
		stdout, err := cmd.Output()
		if err != nil {
			t.Fatalf("perl: %v", err)
		}
		perl := map[string]string{}
		sc := bufio.NewScanner(strings.NewReader(string(stdout)))
		for sc.Scan() {
			task, answer, _ := strings.Cut(sc.Text(), "\t")
			perl[task] = answer
		}
		if _, ok := perl["probe"]; !ok {
			t.Log("Perl not compared for probe: autovivification isn't installed (cpanm autovivification, or set PERL5LIB)")
			perl["probe"] = mappingWant["probe"]
		}
		versions["Perl"] = perl
	}

	for task, want := range mappingWant {
		for name, answers := range versions {
			if got := answers[task]; got != want {
				t.Errorf("%s, %s: got %q, want %q", task, name, got, want)
			}
		}
	}
	for name, answers := range versions {
		if len(answers) != len(mappingWant) {
			t.Errorf("%s answers %d tasks, want %d", name, len(answers), len(mappingWant))
		}
	}
}
