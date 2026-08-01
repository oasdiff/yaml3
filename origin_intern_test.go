package yaml_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

const internDoc = "root:\n    hello: world\n    object:\n        foo: bar\n"

func decodeWithOrigin(c *C, file string) string {
	dec := yaml.NewDecoder(bytes.NewBufferString(internDoc))
	dec.Origin(true, file)
	var out any
	c.Assert(dec.Decode(&out), IsNil)
	b, err := yaml.Marshal(out)
	c.Assert(err, IsNil)
	return string(b)
}

// Origin nodes are interned, so a value occurring many times is one shared
// Node. The file name is the value most at risk from that: it is recorded on
// every mapping, and it is not constant across a document set, because a $ref
// carries the origin into another file which is decoded separately. A cache
// keyed on anything other than the string itself would smear one file's name
// over another's origins.
func (s *S) TestOrigin_InterningKeepsFilesDistinct(c *C) {
	base := decodeWithOrigin(c, "base.yaml")
	revision := decodeWithOrigin(c, "revision.json")

	c.Assert(strings.Contains(base, "base.yaml"), Equals, true)
	c.Assert(strings.Contains(base, "revision.json"), Equals, false)
	c.Assert(strings.Contains(revision, "revision.json"), Equals, true)
	c.Assert(strings.Contains(revision, "base.yaml"), Equals, false)
}

// Shared nodes are only safe while nothing mutates them. Decoding the same
// input twice must reproduce it byte for byte, so a decode that wrote through
// to a cached Node fails here rather than corrupting an unrelated document.
func (s *S) TestOrigin_InternedNodesAreNotMutated(c *C) {
	first := decodeWithOrigin(c, "base.yaml")
	decodeWithOrigin(c, "other.yaml")
	c.Assert(decodeWithOrigin(c, "base.yaml"), Equals, first)
}

// Line numbers past the small-int cache are allocated per use. Pin that they
// are still recorded correctly, since the cached and uncached paths build the
// Node differently.
func (s *S) TestOrigin_LineBeyondTheSmallIntCache(c *C) {
	var sb strings.Builder
	sb.WriteString("root:\n")
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&sb, "    filler%d: x\n", i)
	}
	sb.WriteString("    tail:\n        leaf: value\n")

	dec := yaml.NewDecoder(bytes.NewBufferString(sb.String()))
	dec.Origin(true, "big.yaml")
	var out any
	c.Assert(dec.Decode(&out), IsNil)

	root, ok := out.(map[string]any)["root"].(map[string]any)
	c.Assert(ok, Equals, true)
	tail, ok := root["tail"].(map[string]any)
	c.Assert(ok, Equals, true)
	origin, ok := tail["__origin__"].([]any)
	c.Assert(ok, Equals, true)

	// Header layout is [file, key_name, key_line, key_col, ...]. The key sits
	// past filler0..filler1999, so its line exceeds maxCachedInt.
	c.Assert(origin[0], Equals, "big.yaml")
	c.Assert(origin[1], Equals, "tail")
	c.Assert(toAnyInt(origin[2]), Equals, 2002)
}

func benchDoc(endpoints int) string {
	var sb strings.Builder
	sb.WriteString("openapi: 3.0.3\npaths:\n")
	for i := 0; i < endpoints; i++ {
		fmt.Fprintf(&sb, "    /resource/%d:\n        get:\n            operationId: get%d\n"+
			"            responses:\n                \"200\":\n                    description: ok\n", i, i)
	}
	return sb.String()
}

// The origin nodes dominate allocation on a large document, which is what the
// interning targets. Run with -benchmem; B/op is the number that moved.
func BenchmarkOriginDecode(b *testing.B) {
	doc := benchDoc(2000)
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	for i := 0; i < b.N; i++ {
		dec := yaml.NewDecoder(bytes.NewBufferString(doc))
		dec.Origin(true, "bench.yaml")
		var out any
		if err := dec.Decode(&out); err != nil {
			b.Fatal(err)
		}
	}
}
