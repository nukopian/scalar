package scalar

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// FromYAML reads a YAML document and returns it, as FromJSON reads JSON:
// mappings become hashes, sequences arrays, and plain values scalars.
//
//	cfg, err := scalar.FromYAML(data)
//	replicas := cfg.At("spec.replicas").Int()
//
// Values take the types YAML 1.2 gives them: null, ~ and an empty value are
// undef; true and false are booleans; 42 is an integer and 2.5, .inf and
// .nan floats; !!binary is a byte string; and anything else is a string,
// including a quoted "42" or "007", a date, and no, which YAML 1.2 doesn't
// read as false. An integer too big for an int64 becomes a float.
//
// Anchors and aliases (&base, *base) give the same array or hash at each
// place, as shared parts, so a structure that refers back to itself comes
// back as it was. Merge keys (<<: *base) copy in the keys a mapping doesn't
// have itself, the earlier of several merged mappings winning. A key is its
// text as written, so 1: one gives the key "1"; a key that's a mapping or
// sequence is an error. Tags YAML doesn't define, such as CloudFormation's
// !Ref and !GetAtt, are dropped, keeping the value they're on: !Ref MyBucket
// reads as "MyBucket".
//
// FromYAML reads one document: for a stream of several, separated by ---,
// as Kubernetes manifests often are, it returns an error, and FromYAMLAll
// reads them all. Empty documents, such as after a trailing ---, are
// skipped, as Kubernetes skips them, and an input with none is undef.
func FromYAML(data []byte) (*SV, error) {
	docs, err := yamlDocuments(data)
	if err != nil {
		return nil, err
	}
	switch len(docs) {
	case 0:
		return &SV{}, nil
	case 1:
		return docs[0], nil
	}
	return nil, fmt.Errorf("scalar: FromYAML: the YAML has %d documents; use FromYAMLAll to read them all", len(docs))
}

// FromYAMLAll reads a stream of YAML documents, separated by ---, as in a
// Kubernetes manifest or Helm's output, and returns them as an array, in
// order. Each document is read as FromYAML reads one.
//
//	docs, err := scalar.FromYAMLAll(manifest)
//	for _, d := range docs.All() {
//		fmt.Println(d.At("kind"), d.At("metadata.name"))
//	}
func FromYAMLAll(data []byte) (*AV, error) {
	docs, err := yamlDocuments(data)
	if err != nil {
		return nil, err
	}
	return &AV{items: docs}, nil
}

func yamlDocuments(data []byte) ([]*SV, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []*SV
	for {
		var n yaml.Node
		if err := dec.Decode(&n); err != nil {
			if errors.Is(err, io.EOF) {
				return docs, nil
			}
			return nil, fmt.Errorf("scalar: bad YAML: %s", strings.TrimPrefix(err.Error(), "yaml: "))
		}
		if emptyYAMLDocument(&n) {
			continue // as after a trailing ---: nothing to read
		}
		sv, err := readYAML(&n)
		if err != nil {
			return nil, err
		}
		docs = append(docs, sv)
	}
}

// emptyYAMLDocument reports whether n is a document with nothing in it,
// which the YAML package gives as an empty null; an explicit null or ~ is a
// document of its own.
func emptyYAMLDocument(n *yaml.Node) bool {
	if len(n.Content) == 0 {
		return true
	}
	c := n.Content[0]
	return len(n.Content) == 1 && c.Kind == yaml.ScalarNode && c.ShortTag() == "!!null" && c.Value == ""
}

// UnmarshalYAML lets a scalar be a field of a struct read with the YAML
// package's Unmarshal, for a part of a document whose shape varies; it
// reads as FromYAML does. As with UnmarshalJSON, reading into a locked
// scalar is an error.
func (s *SV) UnmarshalYAML(n *yaml.Node) error {
	sv, err := readYAML(n)
	if err != nil {
		return err
	}
	return s.setRead(sv)
}

func readYAML(n *yaml.Node) (*SV, error) {
	r := yamlReader{seen: map[*yaml.Node]*SV{}}
	return r.node(n, "")
}

// yamlReader turns a YAML node into a scalar, remembering the mappings and
// sequences it has read, so that an alias to one gives the same array or
// hash.
type yamlReader struct {
	seen map[*yaml.Node]*SV
}

func (r *yamlReader) fail(n *yaml.Node, path, format string, args ...any) error {
	where := fmt.Sprintf("line %d, column %d", n.Line, n.Column)
	if path != "" {
		where += " (" + path + ")"
	}
	return fmt.Errorf("scalar: YAML at %s: %s", where, fmt.Sprintf(format, args...))
}

func (r *yamlReader) node(n *yaml.Node, path string) (*SV, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return &SV{}, nil
		}
		return r.node(n.Content[0], path)

	case yaml.AliasNode:
		if sv, ok := r.seen[n.Alias]; ok {
			return &SV{v: sv.v}, nil // the same array or hash
		}
		return r.node(n.Alias, path) // a plain value: read it again

	case yaml.SequenceNode:
		a := &AV{items: make([]*SV, 0, len(n.Content))}
		sv := &SV{v: a}
		r.seen[n] = sv // before the contents, so that an alias inside can lead back here
		for i, c := range n.Content {
			e, err := r.node(c, path+"["+strconv.Itoa(i)+"]")
			if err != nil {
				return nil, err
			}
			a.items = append(a.items, e)
		}
		return sv, nil

	case yaml.MappingNode:
		h := newHV(len(n.Content) / 2)
		sv := &SV{v: h}
		r.seen[n] = sv
		var merges []*yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Kind == yaml.ScalarNode && k.ShortTag() == "!!merge" {
				merges = append(merges, v)
				continue
			}
			key, err := r.key(k, path)
			if err != nil {
				return nil, err
			}
			e, err := r.node(v, keyPath(path, key))
			if err != nil {
				return nil, err
			}
			h.store(key, e)
		}
		// Merge keys: the keys of each merged mapping that this one doesn't
		// have, the earlier mapping winning, as the YAML spec has it.
		for _, m := range merges {
			sources := []*yaml.Node{m}
			if m.Kind == yaml.SequenceNode {
				sources = m.Content
			}
			for _, src := range sources {
				msv, err := r.node(src, path)
				if err != nil {
					return nil, err
				}
				mh, ok := msv.v.(*HV)
				if !ok {
					return nil, r.fail(src, path, "a merge key (<<) must refer to a mapping, or a list of them")
				}
				for _, mk := range mh.keys() {
					if _, exists := h.m[mk]; !exists {
						h.store(mk, &SV{v: mh.m[mk].v}) // the value, not the merged mapping's own scalar
					}
				}
			}
		}
		return sv, nil

	case yaml.ScalarNode:
		return r.scalar(n, path)
	}
	return nil, r.fail(n, path, "unexpected YAML node")
}

// key is a mapping key as a hash key: its text as written.
func (r *yamlReader) key(k *yaml.Node, path string) (string, error) {
	if k.Kind == yaml.AliasNode {
		k = k.Alias
	}
	if k.Kind != yaml.ScalarNode {
		return "", r.fail(k, path, "a key that's a mapping or a sequence can't be a hash key")
	}
	return k.Value, nil
}

func (r *yamlReader) scalar(n *yaml.Node, path string) (*SV, error) {
	switch n.ShortTag() {
	case "!!null":
		return &SV{}, nil
	case "!!bool":
		var b bool
		if err := n.Decode(&b); err != nil {
			return nil, r.fail(n, path, "%v", err)
		}
		return &SV{v: b}, nil
	case "!!int":
		var i int64
		if n.Decode(&i) == nil {
			return &SV{v: i}, nil
		}
		var u uint64
		if n.Decode(&u) == nil {
			return Value(u), nil // beyond int64: a float, as Value has it
		}
		var f float64
		if err := n.Decode(&f); err != nil {
			return nil, r.fail(n, path, "%v", err)
		}
		return &SV{v: f}, nil
	case "!!float":
		var f float64
		if err := n.Decode(&f); err != nil {
			return nil, r.fail(n, path, "%v", err)
		}
		return &SV{v: f}, nil
	case "!!binary":
		b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(n.Value), ""))
		if err != nil {
			return nil, r.fail(n, path, "!!binary isn't valid base64")
		}
		return &SV{v: b}, nil
	}
	// !!str, !!timestamp, and tags YAML doesn't define, such as !Ref: the
	// text as written.
	return &SV{v: n.Value}, nil
}

// ToYAML writes s as a YAML document, with two-space indentation, as
// Kubernetes and Helm use:
//
//	out, err := cfg.ToYAML()
//
// Hash keys are sorted, so the same data always gives the same YAML. Plain
// values are written so that every YAML reader reads them back as the same
// type: a string that looks like a number, a boolean or null to any reader,
// such as "007", "true", or "no", "on" and "y", which YAML 1.1 readers such
// as Kubernetes' and Python's take as booleans, is quoted, as a key or a
// value; a float always has a decimal point or exponent, so 1.0 doesn't
// come back as an integer; infinity and NaN are .inf and .nan; and a byte
// string is !!binary, in base64. An array or hash that appears more than
// once is given an anchor, &1, where it first appears, and is an alias, *1,
// after that, so shared parts, and structures that refer back to
// themselves, read back with FromYAML as they were. JSON can't do either.
func (s *SV) ToYAML() ([]byte, error) { return writeYAML(s) }

// ToYAML writes a as a YAML sequence; see SV.ToYAML.
func (a *AV) ToYAML() ([]byte, error) { return writeYAML(&SV{v: a}) }

// ToYAML writes h as a YAML mapping; see SV.ToYAML.
func (h *HV) ToYAML() ([]byte, error) { return writeYAML(&SV{v: h}) }

// MarshalYAML lets a scalar be a field of a struct written with the YAML
// package's Marshal; it writes as ToYAML does.
func (s *SV) MarshalYAML() (any, error) { return yamlNode(s), nil }

// MarshalYAML lets an array be written with the YAML package's Marshal.
func (a *AV) MarshalYAML() (any, error) { return yamlNode(&SV{v: a}), nil }

// MarshalYAML lets a hash be written with the YAML package's Marshal.
func (h *HV) MarshalYAML() (any, error) { return yamlNode(&SV{v: h}), nil }

func writeYAML(s *SV) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(yamlNode(s)); err != nil {
		return nil, fmt.Errorf("scalar: ToYAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("scalar: ToYAML: %w", err)
	}
	return buf.Bytes(), nil
}

// yamlNode builds the YAML node for s, with anchors on the arrays and
// hashes that appear more than once.
func yamlNode(s *SV) *yaml.Node {
	w := yamlWriter{count: map[any]int{}, nodes: map[any]*yaml.Node{}}
	w.countRefs(s)
	return w.node(s)
}

// yamlWriter builds YAML nodes, knowing how often each array and hash
// appears, and the node already built for each.
type yamlWriter struct {
	count   map[any]int
	nodes   map[any]*yaml.Node
	anchors int
}

// countRefs counts how often each array and hash appears in s, going into
// each only the first time.
func (w *yamlWriter) countRefs(s *SV) {
	c := s.val()
	if !isRef(c) {
		return
	}
	w.count[c]++
	if w.count[c] > 1 {
		return
	}
	switch v := c.(type) {
	case *AV:
		if v == nil {
			return
		}
		for _, e := range v.items {
			if e != nil {
				w.countRefs(e)
			}
		}
	case *HV:
		if v == nil {
			return
		}
		for _, e := range v.m {
			w.countRefs(e)
		}
	}
}

func (w *yamlWriter) node(s *SV) *yaml.Node {
	c := s.val()
	if isRef(c) {
		if n, ok := w.nodes[c]; ok { // written already: an alias to it
			return &yaml.Node{Kind: yaml.AliasNode, Alias: n, Value: n.Anchor}
		}
	}
	switch v := c.(type) {
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(v)}
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(v, 10)}
	case uint64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(v, 10)}
	case float64:
		if d := s.digits(); d != "" {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: d}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: yamlFloat(v)}
	case []byte:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: base64.StdEncoding.EncodeToString(v)}
	case string:
		if !utf8.ValidString(v) { // YAML is text: bytes that aren't UTF-8 go as binary
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: base64.StdEncoding.EncodeToString([]byte(v))}
		}
		return yamlString(v)
	case *AV:
		n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		w.remember(v, n)
		if v != nil {
			for _, e := range v.items {
				if e == nil {
					e = &SV{} // a hole: null
				}
				n.Content = append(n.Content, w.node(e))
			}
		}
		return n
	case *HV:
		n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		w.remember(v, n)
		if v != nil {
			for _, k := range slices.Sorted(maps.Keys(v.m)) {
				n.Content = append(n.Content, yamlString(k), w.node(v.m[k]))
			}
		}
		return n
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.Str()}
}

// remember records the node for an array or hash, with an anchor if it
// appears more than once.
func (w *yamlWriter) remember(c any, n *yaml.Node) {
	w.nodes[c] = n
	if w.count[c] > 1 {
		w.anchors++
		n.Anchor = strconv.Itoa(w.anchors)
	}
}

// yaml11Special matches the plain strings that a YAML 1.1 reader, such as
// the one Kubernetes uses or Python's PyYAML, takes as something else: a
// boolean (yes, no, on, off, y, n), null, or a number (with underscores, in
// base 60 such as 1:30, binary, octal or hex). YAML 1.2 reads most of them
// as strings, so the YAML package leaves them unquoted.
var yaml11Special = regexp.MustCompile(`^(?:[yYnN]|[yY]es|YES|[nN]o|NO|[oO]n|ON|[oO]ff|OFF|[tT]rue|TRUE|[fF]alse|FALSE|[nN]ull|NULL|~|` +
	`[-+]?(?:0b[01_]+|0x[0-9a-fA-F_]+|0[0-7_]+|[0-9][0-9_]*(?::[0-5]?[0-9])*(?:\.[0-9_]*)?(?:[eE][-+]?[0-9]+)?|\.[0-9_]+(?:[eE][-+]?[0-9]+)?)|` +
	`[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)

// yamlString is the node for a string, double-quoted if a YAML 1.1 reader
// would take it as anything else, so that every reader sees a string.
func yamlString(v string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
	if yaml11Special.MatchString(v) {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}

// yamlFloat writes f so that YAML reads it back as the same float.
func yamlFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	}
	out := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(out, ".eE") {
		out += ".0" // 1.0, not 1, which would read back as an integer
	}
	return out
}
