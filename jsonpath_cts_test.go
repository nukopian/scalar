package scalar

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ctsCase is one case of the JSONPath Compliance Test Suite: a query, and
// either the document and its result (or any of several results, where
// the order isn't fixed), or that the query is invalid.
type ctsCase struct {
	Name     string   `json:"name"`
	Selector string   `json:"selector"`
	Document any      `json:"document"`
	Result   []any    `json:"result"`
	Results  [][]any  `json:"results"`
	Invalid  bool     `json:"invalid_selector"`
	Tags     []string `json:"tags"`
}

// ctsOutcome runs one case: "" if Find does what the suite says, or what
// it did instead.
func ctsOutcome(c ctsCase) (outcome string) {
	defer func() {
		if r := recover(); r != nil {
			if c.Invalid {
				outcome = ""
				return
			}
			outcome = fmt.Sprintf("panicked: %v", r)
		}
	}()
	data, err := json.Marshal(c.Document)
	if err != nil {
		return "the suite's document: " + err.Error()
	}
	doc, err := FromJSON(data)
	if err != nil {
		return "FromJSON: " + err.Error()
	}
	found := doc.Find(c.Selector)
	if c.Invalid {
		return "accepted an invalid query"
	}
	got := []any{}
	if found != nil {
		out, err := found.MarshalJSON()
		if err != nil {
			return "MarshalJSON: " + err.Error()
		}
		if err := json.Unmarshal(out, &got); err != nil {
			return "its JSON: " + err.Error()
		}
	}
	wants := c.Results
	if c.Result != nil || wants == nil {
		wants = [][]any{c.Result}
	}
	for _, want := range wants {
		if want == nil {
			want = []any{}
		}
		if reflect.DeepEqual(got, want) {
			return ""
		}
	}
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(wants[0])
	return fmt.Sprintf("got %s, want %s", g, w)
}

// isFilter reports whether a case is about filter selectors, which Find
// didn't support when the suite was added.
func isFilter(c ctsCase) bool {
	return strings.Contains(c.Selector, "?") || strings.HasPrefix(c.Name, "filter") || strings.HasPrefix(c.Name, "functions")
}

// The JSONPath Compliance Test Suite for RFC 9535 (testdata/cts): how much
// of it Find passes, filters apart. go test -run TestCompliance -v lists
// the cases it fails.
func TestCompliance(t *testing.T) {
	data, err := os.ReadFile("testdata/cts/cts.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Tests []ctsCase `json:"tests"`
	}
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	type tally struct{ n, pass int }
	counts := map[string]*tally{}
	var failures []string
	for _, c := range suite.Tests {
		group := "selectors"
		if isFilter(c) {
			group = "filters"
		}
		if counts[group] == nil {
			counts[group] = &tally{}
		}
		counts[group].n++
		if o := ctsOutcome(c); o == "" {
			counts[group].pass++
		} else {
			failures = append(failures, fmt.Sprintf("%-8s %s: %q: %s", group, c.Name, c.Selector, o))
		}
	}
	var groups []string
	for g := range counts {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for _, g := range groups {
		t.Logf("%-9s %d of %d", g, counts[g].pass, counts[g].n)
	}
	if testing.Verbose() {
		for _, f := range failures {
			t.Log(f)
		}
	}
	// What Find passed when this was last changed; raise these as it
	// passes more. Fewer is a regression.
	for g, floor := range map[string]int{"selectors": 243, "filters": 94} {
		if c := counts[g]; c == nil || c.pass < floor {
			t.Errorf("%s: %v passed, fewer than %d", g, c, floor)
		}
	}
}
