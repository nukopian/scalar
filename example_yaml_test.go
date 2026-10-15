package scalar_test

// Examples of reading and writing YAML. They're run as tests, and they make
// up the Reference section of README.md (see TestReference).

import (
	"fmt"

	"github.com/nukopian/job-meta/internal/scalar"
	"go.yaml.in/yaml/v3"
)

func ExampleFromYAML() {
	values, err := scalar.FromYAML([]byte(`
defaults: &defaults
  replicas: 2
  image: {repository: web, tag: "1.0"}
production:
  <<: *defaults        # a merge key: production gets the defaults
  replicas: 5
country: no            # YAML 1.2: the string "no", not false
`))
	if err != nil {
		panic(err)
	}
	fmt.Println(values.At("production.replicas"), values.At("production.image.tag"))
	fmt.Println(values.At("production.image").HV() == values.At("defaults.image").HV()) // one hash, shared
	fmt.Println(values.At("country"))
	// Output:
	// 5 1.0
	// true
	// no
}

func ExampleFromYAMLAll() {
	manifest := []byte(`
apiVersion: v1
kind: Service
metadata: {name: web}
---
apiVersion: apps/v1
kind: Deployment
metadata: {name: web}
spec: {replicas: 3}
---
`)
	docs, err := scalar.FromYAMLAll(manifest) // several documents, as kubectl takes them
	if err != nil {
		panic(err)
	}
	for _, d := range docs.All() {
		fmt.Println(d.At("kind"), d.At("metadata.name"), d.At("spec.replicas").Or("-"))
	}
	// Output:
	// Service web -
	// Deployment web 3
}

func ExampleSV_ToYAML() {
	cfg := scalar.Hash("name", "web", "replicas", 3, "country", "no", "version", "1.0",
		"ports", scalar.Value([]int{80, 443}))
	out, _ := scalar.Value(cfg).ToYAML() // keys sorted; strings that any reader could misread are quoted
	fmt.Print(string(out))
	// Output:
	// country: "no"
	// name: web
	// ports:
	//   - 80
	//   - 443
	// replicas: 3
	// version: "1.0"
}

func ExampleAV_ToYAML() {
	out, _ := scalar.Array("go", 1.0, nil, true).ToYAML()
	fmt.Print(string(out))
	// Output:
	// - go
	// - 1.0
	// - null
	// - true
}

func ExampleHV_ToYAML() {
	shared := scalar.Value(scalar.Hash("cpu", "500m"))
	h := scalar.Hash("requests", shared, "limits", shared) // the same hash twice
	out, _ := h.ToYAML()
	fmt.Print(string(out)) // an anchor, then an alias
	// Output:
	// limits: &1
	//   cpu: 500m
	// requests: *1
}

func ExampleSV_UnmarshalYAML() {
	var cfg struct {
		Name  string     `yaml:"name"`
		Extra *scalar.SV `yaml:"extra"` // a part whose shape varies
	}
	err := yaml.Unmarshal([]byte("name: web\nextra: {ports: [80, 443]}\n"), &cfg)
	fmt.Println(err, cfg.Name, cfg.Extra.At("ports[1]"))
	// Output:
	// <nil> web 443
}

func ExampleSV_MarshalYAML() {
	out, _ := yaml.Marshal(struct {
		Name  string     `yaml:"name"`
		Extra *scalar.SV `yaml:"extra"`
	}{"web", scalar.Value(map[string]any{"ports": []int{80}})})
	fmt.Print(string(out))
	// Output:
	// name: web
	// extra:
	//     ports:
	//         - 80
}

func ExampleAV_MarshalYAML() {
	out, _ := yaml.Marshal(map[string]any{"ports": scalar.Array(80, 443)})
	fmt.Print(string(out))
	// Output:
	// ports:
	//     - 80
	//     - 443
}

func ExampleHV_MarshalYAML() {
	out, _ := yaml.Marshal(map[string]any{"labels": scalar.Hash("app", "web")})
	fmt.Print(string(out))
	// Output:
	// labels:
	//     app: web
}
