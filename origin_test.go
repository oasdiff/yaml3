package yaml_test

import (
	"bytes"
	"fmt"

	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

func toAnyInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case uint64:
		return int(n)
	}
	return 0
}

func (s *S) TestOrigin_Disabled(c *C) {
	input := `
root:
    hello: world
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(false, "")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	c.Assert(buf.String(), Equals, input[1:])
}

func (s *S) TestOrigin_Map(c *C) {
	input := `
root:
    hello: world
    object:
        foo: bar
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "file.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	output := `
__origin__:
    - file.yaml
    - ""
    - 1
    - 1
    - 1
    - root
    - 0
    - 1
    - 0
    - 3
    - 17
root:
    __origin__:
        - file.yaml
        - root
        - 1
        - 1
        - 2
        - hello
        - 1
        - 5
        - object
        - 2
        - 5
        - 0
        - 3
        - 17
    hello: world
    object:
        __origin__:
            - file.yaml
            - object
            - 3
            - 5
            - 1
            - foo
            - 1
            - 9
            - 0
            - 1
            - 17
        foo: bar
`

	c.Assert(buf.String(), Equals, output[1:])
}

func (s *S) TestOrigin_SequenceOfMaps(c *C) {
	input := `
root:
    continents:
        - name: europe
          size: 10
        - name: america
          size: 20
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "file.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	output := `
__origin__:
    - file.yaml
    - ""
    - 1
    - 1
    - 1
    - root
    - 0
    - 1
    - 0
    - 5
    - 19
root:
    __origin__:
        - file.yaml
        - root
        - 1
        - 1
        - 1
        - continents
        - 1
        - 5
        - 0
        - 5
        - 19
    continents:
        - __origin__:
            - file.yaml
            - name
            - 3
            - 11
            - 2
            - name
            - 0
            - 11
            - size
            - 1
            - 11
            - 0
            - 1
            - 19
          name: europe
          size: 10
        - __origin__:
            - file.yaml
            - name
            - 5
            - 11
            - 2
            - name
            - 0
            - 11
            - size
            - 1
            - 11
            - 0
            - 1
            - 19
          name: america
          size: 20
`

	c.Assert(buf.String(), Equals, output[1:])
}

// TestOrigin_MapOfScalars verifies that getFieldLocations() records each
// scalar entry in a map's __origin__.fields, providing precise line/column
// for maps of atomic types (e.g. map[string]string in Go).
func (s *S) TestOrigin_MapOfScalars(c *C) {
	input := `
parent:
    name: test
    labels:
        env: production
        region: us-east
        version: "2.0"
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	output := `
__origin__:
    - spec.yaml
    - ""
    - 1
    - 1
    - 1
    - parent
    - 0
    - 1
    - 0
    - 5
    - 23
parent:
    __origin__:
        - spec.yaml
        - parent
        - 1
        - 1
        - 2
        - name
        - 1
        - 5
        - labels
        - 2
        - 5
        - 0
        - 5
        - 23
    labels:
        __origin__:
            - spec.yaml
            - labels
            - 3
            - 5
            - 3
            - env
            - 1
            - 9
            - region
            - 2
            - 9
            - version
            - 3
            - 9
            - 0
            - 3
            - 23
        env: production
        region: us-east
        version: "2.0"
    name: test
`

	c.Assert(buf.String(), Equals, output[1:])
}

// TestOrigin_SequenceOfScalars verifies that getSequenceLocations() records
// each scalar item in a sequence under the parent's __origin__.sequences,
// providing precise line/column for lists of atomic types (e.g. []string in Go).
func (s *S) TestOrigin_SequenceOfScalars(c *C) {
	input := `
schema:
    description: a test
    type:
        - string
        - "null"
        - integer
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	output := `
__origin__:
    - spec.yaml
    - ""
    - 1
    - 1
    - 1
    - schema
    - 0
    - 1
    - 0
    - 5
    - 18
schema:
    __origin__:
        - spec.yaml
        - schema
        - 1
        - 1
        - 2
        - description
        - 1
        - 5
        - type
        - 2
        - 5
        - 1
        - type
        - 3
        - string
        - 3
        - 11
        - "null"
        - 4
        - 11
        - integer
        - 5
        - 11
        - 5
        - 18
    description: a test
    type:
        - string
        - "null"
        - integer
`

	c.Assert(buf.String(), Equals, output[1:])
}

// TestOrigin_SequenceOfEmptyMaps verifies that addOriginInSeq does not panic
// when a sequence contains an empty mapping node (e.g. `- {}`).
// Regression test for https://github.com/oasdiff/oasdiff/issues/808.
func (s *S) TestOrigin_SequenceOfEmptyMaps(c *C) {
	input := `
root:
    items:
        - {}
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "file.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
}

// TestOrigin_AnchorAliasInMap verifies that a YAML alias used as a map value
// does not produce duplicate __origin__ keys in nested mapping nodes.
// Regression test for https://github.com/oasdiff/oasdiff/issues/821.
//
// The anchor's nested MappingNode (e.g. "properties") is shared by pointer
// with every alias that resolves to it. Without the fix, addOriginInMap appends
// __origin__ to that node once when the anchor is processed, then again for
// every alias expansion — resulting in a duplicate-key error on marshal.
func (s *S) TestOrigin_AnchorAliasInMap(c *C) {
	input := `
x-inner: &inner
    type: object
    properties:
        x:
            type: integer

x-outer:
    type: object
    properties:
        nested: *inner
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)

	// Must be able to marshal back — duplicate __origin__ keys would panic here.
	_, err = yaml.Marshal(out)
	c.Assert(err, IsNil)
}

// TestOrigin_AnchorAliasInSequence verifies that a YAML alias used as a
// sequence element does not produce duplicate __origin__ keys in nested
// mapping nodes inside the anchor.
// Regression test for https://github.com/oasdiff/oasdiff/issues/821.
func (s *S) TestOrigin_AnchorAliasInSequence(c *C) {
	input := `
x-pet: &pet
    name: dog
    metadata:
        version: "1.0"

pets:
    - *pet
    - *pet
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)

	_, err = yaml.Marshal(out)
	c.Assert(err, IsNil)
}

// TestOrigin_ManyAliasesNoExcessiveAliasing verifies that a spec with many
// aliases and nested mappings does not trigger the "excessive aliasing" check.
// Regression test for the follow-up to https://github.com/oasdiff/oasdiff/issues/821:
// fixing the duplicate __origin__ bug caused __origin__ metadata entries to be
// re-decoded during every alias expansion, inflating aliasCount and spuriously
// tripping the ratio check for large specs.
//
// The threshold is empirically derived: 5000 aliases of an anchor with 20
// properties reliably triggers "document contains excessive aliasing" when
// __origin__ entries are re-decoded during expansion (without the fix), and
// passes cleanly when those entries are skipped (with the fix).
func (s *S) TestOrigin_ManyAliasesNoExcessiveAliasing(c *C) {
	// Build an anchor with 20 scalar properties so __origin__ metadata is
	// injected and adds many extra nodes to the anchor's Content slice.
	props := ""
	for i := range 20 {
		props += fmt.Sprintf("        prop%d:\n            type: string\n", i)
	}
	anchor := fmt.Sprintf("x-schema: &schema\n    type: object\n    properties:\n%s", props)

	// 5000 aliases push aliasCount/decodeCount past the ratio threshold when
	// __origin__ entries inside the anchor are re-decoded on each expansion.
	input := anchor + "\nroot:\n    properties:\n"
	for i := range 5000 {
		input += fmt.Sprintf("        field%d: *schema\n", i)
	}

	dec := yaml.NewDecoder(bytes.NewBufferString(input))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
}

// TestOrigin_AliasPreservesSequences verifies that __origin__ data (including
// sequence-item locations) is preserved when a YAML alias is expanded.
// Regression test: the previous fix for excessive aliasing skipped __origin__
// entries entirely during expansion, which silently dropped all origin metadata
// from alias-expanded mappings.
func (s *S) TestOrigin_AliasPreservesSequences(c *C) {
	input := `
schema: &schema
    type: object
    required:
        - foo
        - bar
alias: *schema
`
	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)

	m := out.(map[string]any)
	alias := m["alias"].(map[string]any)
	origin := alias["__origin__"]
	c.Assert(origin, NotNil, Commentf("alias expansion must preserve __origin__"))

	// New compact format: []any [file, key_name, key_line, key_col, nf, ..., ns, seq_name, count, ...]
	// ns > 0 means sequences are present; verify the sequence list is non-empty.
	seq, ok := origin.([]any)
	c.Assert(ok, Equals, true, Commentf("origin must be a []any sequence"))
	// Find ns (at index 4 + nf*3): nf is at index 4 (after file at index 0).
	nf := toAnyInt(seq[4])
	nsIdx := 5 + nf*3
	c.Assert(nsIdx < len(seq), Equals, true, Commentf("sequence must contain ns field"))
	ns := toAnyInt(seq[nsIdx])
	c.Assert(ns > 0, Equals, true, Commentf("alias __origin__ must record sequence item locations"))
}

// TestOrigin_BlockEnd verifies the trailing end_delta/end_col appended to each
// __origin__ sequence reconstruct the end of the whole block (the position just
// past its last content), which is how kin-openapi recovers an endpoint's span.
func (s *S) TestOrigin_BlockEnd(c *C) {
	// 1: paths:
	// 2:   /pets:
	// 3:     get:
	// 4:       summary: list
	// 5:       x: "y"
	// 6:   /health:
	input := `paths:
  /pets:
    get:
      summary: list
      x: "y"
  /health:
    get:
      summary: ok
`
	dec := yaml.NewDecoder(bytes.NewBufferString(input))
	dec.Origin(true, "spec.yaml")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)

	get := out.(map[string]any)["paths"].(map[string]any)["/pets"].(map[string]any)["get"].(map[string]any)
	seq, ok := get["__origin__"].([]any)
	c.Assert(ok, Equals, true, Commentf("get block must carry __origin__"))

	keyLine := toAnyInt(seq[2]) // header: file, key_name, key_line, key_col, ...
	c.Assert(keyLine, Equals, 3, Commentf("get key is on line 3"))

	// end_delta, end_col are the last two entries.
	endDelta := toAnyInt(seq[len(seq)-2])
	endCol := toAnyInt(seq[len(seq)-1])
	endLine := keyLine + endDelta
	// When a block is followed by a dedented sibling, the end lands on the
	// block's last content line (x: "y" on line 5), inclusive. The key property
	// for block extraction: it covers the whole get block and does not bleed
	// into the /health sibling on line 6.
	c.Assert(endLine, Equals, 5, Commentf("get block should end at its last content line (5), not bleed into the sibling; got %d", endLine))
	// End column is just past the last content (`x: "y"` ends at col 12, so 13).
	c.Assert(endCol, Equals, 13, Commentf("end column should be just past the last content; got %d", endCol))
}

func (s *S) TestOrigin_DuplicateKey(c *C) {
	input := `
root:
    __origin__: test
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true, "")
	var out any
	err := dec.Decode(&out)
	c.Assert(err, ErrorMatches, "yaml: unmarshal errors:\n  line 0: mapping key \"__origin__\" already defined at line 2")
}
