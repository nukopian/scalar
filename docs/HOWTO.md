# scalar: how-to guide

How to get things done with [scalar](../README.md): getting started, then
recipes for common tasks, then two jobs done in plain Go and with scalar,
side by side. For how each feature behaves, see the
[reference](REFERENCE.md); for one example of every function and method,
see the [examples](EXAMPLES.md).

- [Quick start](#quick-start)
- [Recipes](#recipes)
  - [Reading a model's tool_use response](#reading-a-models-tool_use-response)
  - [Cleaning a list of skills](#cleaning-a-list-of-skills)
  - [Pulling salary figures out of text](#pulling-salary-figures-out-of-text)
  - [Counting skills across postings](#counting-skills-across-postings)
  - [Building records and writing JSON](#building-records-and-writing-json)
  - [Storing an embedding vector as bytes](#storing-an-embedding-vector-as-bytes)
- [Compared with plain Go](#compared-with-plain-go)

## Quick start

```go
import "github.com/nukopian/scalar"

resp, err := scalar.FromJSON(body)
if err != nil {
	return err
}

text := resp.Path("content", 0, "text").Str() // $resp->{content}[0]{text}
tokens := resp.At("usage.output_tokens").Int() // the same, as one string

usage, err := scalar.To[Usage](resp.At("usage")) // into your own struct, checked
```

If any step along the path is missing, you get undef back instead of a panic.
`Str()` on undef is `""` and `Int()` is `0`. `Path` takes keys and indices
as separate arguments; `At` takes the route as one string, in dot, bracket
or Perl's brace form (see [A path as one string](REFERENCE.md#a-path-as-one-string-at)).

## Recipes

Six short examples from real work, each combining several parts of the
package. The [reference](REFERENCE.md) explains each part in detail.

These recipes are also runnable examples in the repository's
`example_test.go`, so `go test` checks they still work. Keep the two in step when changing them.

### Reading a model's tool_use response

Find the `tool_use` block, then read its fields with defaults, telling a
missing field from a null one.

```go
resp, err := scalar.FromJSON(body)
if err != nil {
	return err
}

input := resp.Find("$.content[*]").FirstMatch(func(b *scalar.SV) bool {
	return b.Path("type").Eq("tool_use")
}).Path("input")

fmt.Println(input.Path("title"))                         // Senior Go Developer
fmt.Println(resp.At("usage.output_tokens").Or(0))        // 0, if the model didn't say

if v, ok := input.PathOK("salaryMax"); ok && !v.Defined() {
	fmt.Println("salaryMax: null")                         // present, but null
}
if _, ok := input.PathOK("location"); !ok {
	fmt.Println("location: missing")                       // left out entirely
}
input.HV().Get("location").OrSet("Unknown")              // $input->{location} //= "Unknown"
```

### Cleaning a list of skills

Trim each skill where it's stored, then drop duplicates regardless of case,
keeping the first spelling of each.

```go
skills := resp.Find("$..requiredSkills").Get(0).AV()   // [" Go ", "PostgreSQL", "go", "Docker "]

for _, s := range skills.All() {
	s.Set(s.Trim())                                      // changes the skills inside resp
}

// grep { !$seen{fc $_}++ } @skills
seen := scalar.Hash()
unique := skills.Grep(func(s *scalar.SV) bool {
	return seen.Get(s.Fc().Str()).Inc().Int() == 1
})
fmt.Println(unique.Join(", "))                         // Go, PostgreSQL, Docker
```

### Pulling salary figures out of text

The pattern is compiled once, when the program starts, as with Perl's
`qr//`.

```go
var salaryFigure = scalar.MustCompile(`£([\d,]+)`)

text := scalar.Value("Salary: £45,000 - £55,000 per annum")

figures := scalar.Array()
for m := range text.Matches(salaryFigure) {         // while ($text =~ /$salaryFigure/g)
	figures.Push(m.Get(1).S(`,`, "", "gr"))         // "45,000" → "45000"
}
fmt.Println(figures.MinNum(), figures.MaxNum())    // 45000 55000
```

### Counting skills across postings

Count each skill ignoring case in a `Bag`, then list the most common first.

```go
postings := [][]string{
	{"Go", "Docker"},
	{"go", "Kubernetes"},
	{"GO", "Docker", "Perl"},
}

var counts scalar.Bag                                // the zero value is an empty bag
for _, skills := range postings {
	for _, skill := range scalar.Array(skills).All() {
		counts.Insert(skill.Fc())                      // $counts{fc $skill}++
	}
}

for _, skill := range counts.MostCommon(3).All() {   // ties keep the order they were first seen in
	fmt.Println(skill, counts.Count(skill))            // go 3, docker 2, kubernetes 1
}
```

The same with a hash, `Inc` and a sort is in [Sorting](REFERENCE.md#sorting).

### Building records and writing JSON

Start each record from shared defaults, collect them in an array, and encode
the result.

```go
defaults := scalar.Hash("location", "Unknown", "remote", false)

rows := scalar.Array()
for _, title := range []string{"Go Developer", "Perl Developer"} {
	row := scalar.Hash(defaults, "title", title)   // %row = (%defaults, title => $title)
	rows.Push(scalar.Value(row))                  // push @rows, \%row
}
rows.Get(1).Path("remote").Set(true)             // changes only the second row

out, _ := json.Marshal(rows)
// [{"location":"Unknown","remote":false,"title":"Go Developer"},
//  {"location":"Unknown","remote":true,"title":"Perl Developer"}]
```

Each row is a separate copy of `defaults`, so changing one row changes neither
the others nor `defaults`.

### Storing an embedding vector as bytes

Pack a vector into 4 bytes per float32, for a blob column or a cache, and
unpack it again.

```go
vector := []float64{0.5, -0.25, 0.125}               // from the embedding model

blob := scalar.Pack("f<*", vector)                   // pack("f<*", @vector): little-endian float32s
blob.Length()                                        // 12: 4 bytes each
fmt.Printf("%x\n", blob)                             // 0000003f000080be0000003e, for a log

back := scalar.Unpack("f<*", blob.Bytes()).NumSlice()   // a []float64 again
```

`blob` is a [byte string](REFERENCE.md#byte-strings), and `blob.Bytes()` is the Go
`[]byte` to store.

## Compared with plain Go

Two jobs are awkward in plain Go.

**1. Reading loosely shaped JSON.** LLM output and third-party APIs return
JSON whose shape changes: fields go missing, nesting varies, and numbers turn
up as strings. In plain Go, reading one nested value means a chain of type
assertions, and each one needs checking:

```go
var resp map[string]any
if err := json.Unmarshal(body, &resp); err != nil { ... }
content, ok := resp["content"].([]any)
if !ok || len(content) == 0 { ... }
first, ok := content[0].(map[string]any)
if !ok { ... }
text, ok := first["text"].(string)
if !ok { ... }
```

Skip a check and the program panics on the first unexpected response. With
`scalar`, the same read is:

```go
resp, err := scalar.FromJSON(body)
if err != nil { ... }
text := resp.At("content[0].text").Str()   // or resp.Path("content", 0, "text")
```

If anything along the path is missing or the wrong type, `text` is just `""`.

**2. Heavy text processing.** Cleaning up text, such as job postings, means a
lot of regex work. Go's standard `regexp` package deliberately leaves out
lookahead, lookbehind and backreferences. `scalar` uses an engine that has
them, with a timeout so a bad pattern can't hang the program. It also wraps
them in short helpers modelled on Perl's `s///`, `split` and `tr///`.
