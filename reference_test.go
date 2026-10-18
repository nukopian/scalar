package scalar

import (
	"testing"

	"github.com/nukopian/scalar/internal/docref"
)

// referenceGroups are the example files, in the order and under the
// headings they appear under in docs/EXAMPLES.md.
var referenceGroups = []docref.Group{
	{File: "example_sv_test.go", Title: "Scalars"},
	{File: "example_str_test.go", Title: "Strings"},
	{File: "example_ops_test.go", Title: "Operators"},
	{File: "example_av_test.go", Title: "Arrays"},
	{File: "example_hv_test.go", Title: "Hashes"},
	{File: "example_list_test.go", Title: "List utilities"},
	{File: "example_set_test.go", Title: "Sets and bags"},
	{File: "example_regex_test.go", Title: "Regular expressions"},
	{File: "example_pack_test.go", Title: "Pack and unpack"},
	{File: "example_json_test.go", Title: "JSON and JSONPath"},
	{File: "example_yaml_test.go", Title: "YAML"},
	{File: "example_to_test.go", Title: "Into Go values"},
}

// Every exported function and method has an example.
func TestEveryFeatureHasExample(t *testing.T) { docref.CheckExamples(t, referenceGroups) }

// docs/EXAMPLES.md is what the examples generate. To update it:
//
//	go test -run TestReference -update
func TestReference(t *testing.T) { docref.CheckDoc(t, "docs/EXAMPLES.md", referenceGroups) }
