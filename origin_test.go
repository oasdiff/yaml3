package yaml_test

import (
	"bytes"

	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

func (s *S) TestOrigin_Disabled(c *C) {
	input := `
root:
    hello: world
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(false)
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	c.Assert(buf.String(), Equals, input[1:])
}

func (s *S) TestOrigin_Enabled(c *C) {
	input := `
root:
    hello: world
`

	dec := yaml.NewDecoder(bytes.NewBufferString(input[1:]))
	dec.Origin(true)
	var out any
	err := dec.Decode(&out)
	c.Assert(err, IsNil)
	result, err := yaml.Marshal(out)
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	buf.Write(result)

	output := `
root:
    hello: world
    origin:
        fields:
            hello:
                column: 5
                line: 2
                name: hello
        key:
            column: 1
            line: 1
            name: root
`

	c.Assert(buf.String(), Equals, output[1:])
}
