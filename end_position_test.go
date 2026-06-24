package yaml_test

import (
	yaml "github.com/oasdiff/yaml3"
	. "gopkg.in/check.v1"
)

// findKey returns the value node for the given key path in a decoded mapping.
func findKey(c *C, n *yaml.Node, path ...string) *yaml.Node {
	cur := n
	if cur.Kind == yaml.DocumentNode {
		cur = cur.Content[0]
	}
	for _, key := range path {
		c.Assert(cur.Kind, Equals, yaml.MappingNode, Commentf("node for key %q is not a mapping", key))
		var next *yaml.Node
		for i := 0; i+1 < len(cur.Content); i += 2 {
			if cur.Content[i].Value == key {
				next = cur.Content[i+1]
				break
			}
		}
		c.Assert(next, NotNil, Commentf("key %q not found", key))
		cur = next
	}
	return cur
}

// TestNodeEndPosition covers all four node kinds. The end position is recorded
// on two code paths: node() sets it for scalars and aliases (from the event's
// end_mark), while mapping() and sequence() derive it from their last child.
// Across all kinds the end means the same thing -- the position just past the
// last content character -- rather than the start of the following line, which
// is where the collection END token sits after a block dedent.
func (s *S) TestNodeEndPosition(c *C) {
	//  1: mapping:
	//  2:   a: 1
	//  3:   b: 2
	//  4: sequence:
	//  5:   - first
	//  6:   - second
	//  7: base: &anchor
	//  8:   key: value
	//  9: aliased: *anchor
	// 10: plain: text
	src := `mapping:
  a: 1
  b: 2
sequence:
  - first
  - second
base: &anchor
  key: value
aliased: *anchor
plain: text
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	// Mapping (last-child path): starts at its first entry (line 2, col 3) and
	// ends just past its last value `2` (line 3, col 7) -- the end of the actual
	// content, not the start of the following line.
	mapping := findKey(c, &doc, "mapping")
	c.Assert(mapping.Kind, Equals, yaml.MappingNode)
	c.Assert(mapping.Line, Equals, 2)
	c.Assert(mapping.Column, Equals, 3)
	c.Assert(mapping.EndLine, Equals, 3)
	c.Assert(mapping.EndColumn, Equals, 7)

	// Sequence (last-child path): ends just past its last item `second`
	// (line 6, col 11).
	seq := findKey(c, &doc, "sequence")
	c.Assert(seq.Kind, Equals, yaml.SequenceNode)
	c.Assert(seq.Line, Equals, 5)
	c.Assert(seq.Column, Equals, 3)
	c.Assert(seq.EndLine, Equals, 6)
	c.Assert(seq.EndColumn, Equals, 11)

	// Alias (node() path): a reference is a single token, so it starts and ends
	// on the same line -- and its end is the alias token's (cols 10-17), not the
	// span of the anchored mapping it points at.
	alias := findKey(c, &doc, "aliased")
	c.Assert(alias.Kind, Equals, yaml.AliasNode)
	c.Assert(alias.Line, Equals, 9)
	c.Assert(alias.Column, Equals, 10)
	c.Assert(alias.EndLine, Equals, 9)
	c.Assert(alias.EndColumn, Equals, 17)

	// Scalar (node() path): starts and ends on the same line (cols 8-12).
	scalar := findKey(c, &doc, "plain")
	c.Assert(scalar.Kind, Equals, yaml.ScalarNode)
	c.Assert(scalar.Line, Equals, 10)
	c.Assert(scalar.Column, Equals, 8)
	c.Assert(scalar.EndLine, Equals, 10)
	c.Assert(scalar.EndColumn, Equals, 12)
}

// TestNodeEndPositionLastInDoc covers a collection that is the last element in
// the document: it ends at EOF rather than at a dedent to a following sibling.
// The end must still be its last child's end, not an overshoot to the line
// past the last content -- and it must hold with or without a trailing newline.
func (s *S) TestNodeEndPositionLastInDoc(c *C) {
	// 1: top: 1
	// 2: block:
	// 3:   x: 10
	// 4:   y: 20
	src := `top: 1
block:
  x: 10
  y: 20
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	block := findKey(c, &doc, "block")
	c.Assert(block.Kind, Equals, yaml.MappingNode)
	c.Assert(block.EndLine, Equals, 4) // last content line, not the post-EOF line 5
	c.Assert(block.EndColumn, Equals, 8) // just past `20`

	// The document's root mapping ends where its last child does.
	root := doc.Content[0]
	c.Assert(root.EndLine, Equals, 4)
	c.Assert(root.EndColumn, Equals, 8)

	// A trailing sequence ends at its last item, again without overshooting EOF.
	var doc2 yaml.Node
	c.Assert(yaml.Unmarshal([]byte("top: 1\nlist:\n  - a\n  - bb\n"), &doc2), IsNil)
	list := findKey(c, &doc2, "list")
	c.Assert(list.Kind, Equals, yaml.SequenceNode)
	c.Assert(list.EndLine, Equals, 4)
	c.Assert(list.EndColumn, Equals, 7) // just past `bb`

	// Robust to a missing trailing newline on the final element.
	var doc3 yaml.Node
	c.Assert(yaml.Unmarshal([]byte("top: 1\nblock:\n  x: 10\n  y: 20"), &doc3), IsNil)
	block3 := findKey(c, &doc3, "block")
	c.Assert(block3.EndLine, Equals, 4)
	c.Assert(block3.EndColumn, Equals, 8)
}

// TestNodeEndPositionEmpty covers the fallback for empty collections: with no
// last child to borrow an end from, mapping()/sequence() fall back to the
// END-event mark, which for a flow `{}`/`[]` is just past the closing delimiter.
func (s *S) TestNodeEndPositionEmpty(c *C) {
	// 1: emptyMap: {}
	// 2: emptySeq: []
	src := `emptyMap: {}
emptySeq: []
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	em := findKey(c, &doc, "emptyMap")
	c.Assert(em.Kind, Equals, yaml.MappingNode)
	c.Assert(em.Content, HasLen, 0)
	c.Assert(em.Line, Equals, 1)
	c.Assert(em.EndLine, Equals, 1)
	c.Assert(em.EndColumn, Equals, 13) // just past `}` (cols 11-12)

	es := findKey(c, &doc, "emptySeq")
	c.Assert(es.Kind, Equals, yaml.SequenceNode)
	c.Assert(es.Content, HasLen, 0)
	c.Assert(es.Line, Equals, 2)
	c.Assert(es.EndLine, Equals, 2)
	c.Assert(es.EndColumn, Equals, 13) // just past `]` (cols 11-12)
}
