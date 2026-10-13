package scalar

import (
	"testing"

	"github.com/nukopian/job-meta/internal/docref"
)

// referenceGroups are the example files, in the order and under the
// headings they appear in the README's Reference.
var referenceGroups = []docref.Group{
	{File: "example_sv_test.go", Title: "Scalars"},
	{File: "example_str_test.go", Title: "Strings"},
	{File: "example_ops_test.go", Title: "Operators"},
	{File: "example_av_test.go", Title: "Arrays"},
	{File: "example_hv_test.go", Title: "Hashes"},
	{File: "example_list_test.go", Title: "List utilities"},
	{File: "example_set_test.go", Title: "Sets and bags"},
	{File: "example_regex_test.go", Title: "Regular expressions"},
	{File: "example_json_test.go", Title: "JSON and JSONPath"},
}

// Every exported function and method has an example.
func TestEveryFeatureHasExample(t *testing.T) { docref.CheckExamples(t, referenceGroups) }

// README.md's Reference section is what the examples generate. To update it:
//
//	go test ./internal/scalar -run TestReference -update
func TestReference(t *testing.T) { docref.CheckReadme(t, referenceGroups) }
