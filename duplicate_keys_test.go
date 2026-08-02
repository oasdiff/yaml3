package yaml_test

import (
	"bytes"

	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

type dupInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type dupDoc struct {
	Info dupInfo `yaml:"info"`
}

const dupSrc = "info:\n  title: a\n  title: b\n  version: \"1\"\n"

func decodeDup(c *C, allow bool) (dupDoc, error) {
	dec := yaml.NewDecoder(bytes.NewBufferString(dupSrc))
	dec.AllowDuplicateKeys(allow)
	var d dupDoc
	return d, dec.Decode(&d)
}

// The default is unchanged: a repeated key is an error, which is YAML's rule.
func (s *S) TestDuplicateKeys_RejectedByDefault(c *C) {
	dec := yaml.NewDecoder(bytes.NewBufferString(dupSrc))
	var d dupDoc
	err := dec.Decode(&d)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, `(?s).*already defined.*`)
}

func (s *S) TestDuplicateKeys_RejectedWhenDisallowed(c *C) {
	_, err := decodeDup(c, false)
	c.Assert(err, NotNil)
}

// Enabled, the last occurrence wins -- the same answer encoding/json gives for
// a repeated JSON object name. Asserting the surviving value, not merely the
// absence of an error: "does not fail" would also be satisfied by dropping the
// key or keeping the first, and neither would match encoding/json.
func (s *S) TestDuplicateKeys_LastOneWinsWhenAllowed(c *C) {
	d, err := decodeDup(c, true)
	c.Assert(err, IsNil)
	c.Assert(d.Info.Title, Equals, "b")
	// Sibling keys are untouched by the override.
	c.Assert(d.Info.Version, Equals, "1")
}

// The flag is per-decoder, so enabling it for one document must not leak into
// another built from the same package.
func (s *S) TestDuplicateKeys_FlagDoesNotLeak(c *C) {
	_, err := decodeDup(c, true)
	c.Assert(err, IsNil)

	dec := yaml.NewDecoder(bytes.NewBufferString(dupSrc))
	var d dupDoc
	c.Assert(dec.Decode(&d), NotNil)
}

// A duplicate inside a nested mapping is caught too, not just at the top level.
func (s *S) TestDuplicateKeys_Nested(c *C) {
	const src = "a:\n  b:\n    c: 1\n    c: 2\n"
	var out any

	dec := yaml.NewDecoder(bytes.NewBufferString(src))
	c.Assert(dec.Decode(&out), NotNil)

	dec = yaml.NewDecoder(bytes.NewBufferString(src))
	dec.AllowDuplicateKeys(true)
	out = nil
	c.Assert(dec.Decode(&out), IsNil)
	a := out.(map[string]any)["a"].(map[string]any)
	c.Assert(a["b"].(map[string]any)["c"], Equals, 2)
}

// Decoding into a generic map, rather than a struct, takes a different path in
// the decoder (mapping vs mappingStruct), so it needs its own coverage.
func (s *S) TestDuplicateKeys_IntoGenericMap(c *C) {
	dec := yaml.NewDecoder(bytes.NewBufferString(dupSrc))
	dec.AllowDuplicateKeys(true)
	var out any
	c.Assert(dec.Decode(&out), IsNil)
	info := out.(map[string]any)["info"].(map[string]any)
	c.Assert(info["title"], Equals, "b")
}
