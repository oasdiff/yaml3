package yaml_test

import (
	"bytes"
	"fmt"

	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

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
root:
    __origin__:
        fields:
            hello:
                column: 5
                file: file.yaml
                line: 2
                name: hello
        key:
            column: 1
            file: file.yaml
            line: 1
            name: root
    hello: world
    object:
        __origin__:
            fields:
                foo:
                    column: 9
                    file: file.yaml
                    line: 4
                    name: foo
            key:
                column: 5
                file: file.yaml
                line: 3
                name: object
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
root:
    __origin__:
        fields:
            continents:
                column: 5
                file: file.yaml
                line: 2
                name: continents
        key:
            column: 1
            file: file.yaml
            line: 1
            name: root
    continents:
        - __origin__:
            fields:
                name:
                    column: 11
                    file: file.yaml
                    line: 3
                    name: name
                size:
                    column: 11
                    file: file.yaml
                    line: 4
                    name: size
            key:
                column: 11
                file: file.yaml
                line: 3
                name: name
          name: europe
          size: 10
        - __origin__:
            fields:
                name:
                    column: 11
                    file: file.yaml
                    line: 5
                    name: name
                size:
                    column: 11
                    file: file.yaml
                    line: 6
                    name: size
            key:
                column: 11
                file: file.yaml
                line: 5
                name: name
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
parent:
    __origin__:
        fields:
            name:
                column: 5
                file: spec.yaml
                line: 2
                name: name
        key:
            column: 1
            file: spec.yaml
            line: 1
            name: parent
    labels:
        __origin__:
            fields:
                env:
                    column: 9
                    file: spec.yaml
                    line: 4
                    name: env
                region:
                    column: 9
                    file: spec.yaml
                    line: 5
                    name: region
                version:
                    column: 9
                    file: spec.yaml
                    line: 6
                    name: version
            key:
                column: 5
                file: spec.yaml
                line: 3
                name: labels
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
schema:
    __origin__:
        fields:
            description:
                column: 5
                file: spec.yaml
                line: 2
                name: description
            type:
                column: 5
                file: spec.yaml
                line: 3
                name: type
        key:
            column: 1
            file: spec.yaml
            line: 1
            name: schema
        sequences:
            type:
                - column: 11
                  file: spec.yaml
                  line: 4
                  name: string
                - column: 11
                  file: spec.yaml
                  line: 5
                  name: "null"
                - column: 11
                  file: spec.yaml
                  line: 6
                  name: integer
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

	originMap := origin.(map[string]any)
	sequences := originMap["sequences"]
	c.Assert(sequences, NotNil, Commentf("alias __origin__ must contain sequences"))

	seqMap := sequences.(map[string]any)
	required := seqMap["required"]
	c.Assert(required, NotNil, Commentf("sequences must contain required field tracking"))
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
