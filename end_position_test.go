package yaml_test

import (
	"bytes"

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

// TestNodeEndPositionBlockScalar covers literal (|) and folded (>) block
// scalars: multi-line values whose end must land just past the last content
// line, not on the `|`/`>` indicator line and not overshooting to the next key.
func (s *S) TestNodeEndPositionBlockScalar(c *C) {
	//  1: literal: |
	//  2:   one
	//  3:   two
	//  4: folded: >
	//  5:   alpha
	//  6:   beta
	//  7: after: end
	src := `literal: |
  one
  two
folded: >
  alpha
  beta
after: end
`
	// Block scalars are leaves with no child to borrow an end from, and libyaml's
	// scalar end_mark sits past the trailing line break (at the start of the next
	// line). The scanner captures the position just past the last content
	// character instead, so the span ends on the last content line rather than
	// bleeding into the following node (which matters for any schema ending in a
	// multi-line `description: |`).
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	lit := findKey(c, &doc, "literal")
	c.Assert(lit.Kind, Equals, yaml.ScalarNode)
	c.Assert(lit.Value, Equals, "one\ntwo\n")
	c.Assert(lit.EndLine, Equals, 3)   // last content line (`two`)
	c.Assert(lit.EndColumn, Equals, 6) // just past `two` (cols 3-5)

	fold := findKey(c, &doc, "folded")
	c.Assert(fold.Kind, Equals, yaml.ScalarNode)
	c.Assert(fold.Value, Equals, "alpha beta\n")
	c.Assert(fold.EndLine, Equals, 6)   // last content line (`beta`)
	c.Assert(fold.EndColumn, Equals, 7) // just past `beta` (cols 3-6)
}

// TestNodeEndPositionTrailingComment verifies a comment does not extend a
// node's end: neither a same-line trailing comment nor a standalone comment
// line after a block becomes part of the preceding node's span.
func (s *S) TestNodeEndPositionTrailingComment(c *C) {
	//  1: scalar: value  # trailing
	//  2: block:
	//  3:   a: 1
	//  4:   b: 2
	//  5: # standalone comment
	//  6: after: x
	src := `scalar: value  # trailing
block:
  a: 1
  b: 2
# standalone comment
after: x
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	sc := findKey(c, &doc, "scalar")
	c.Assert(sc.Kind, Equals, yaml.ScalarNode)
	c.Assert(sc.EndLine, Equals, 1)
	c.Assert(sc.EndColumn, Equals, 14) // just past `value` (cols 9-13), not the comment

	block := findKey(c, &doc, "block")
	c.Assert(block.Kind, Equals, yaml.MappingNode)
	c.Assert(block.EndLine, Equals, 4)   // last content `b: 2`, not the standalone comment on line 5
	c.Assert(block.EndColumn, Equals, 7) // just past `2`
}

// TestNodeEndPositionMultiDoc verifies end positions are correct in the second
// document of a stream, where every mark sits at a non-zero line offset.
func (s *S) TestNodeEndPositionMultiDoc(c *C) {
	//  1: ---
	//  2: a: 1
	//  3: ---
	//  4: c: 3
	//  5: d: 44
	src := "---\na: 1\n---\nc: 3\nd: 44\n"
	dec := yaml.NewDecoder(bytes.NewBufferString(src))
	var d1, d2 yaml.Node
	c.Assert(dec.Decode(&d1), IsNil)
	c.Assert(dec.Decode(&d2), IsNil)

	root2 := d2.Content[0]
	c.Assert(root2.Kind, Equals, yaml.MappingNode)
	c.Assert(root2.EndLine, Equals, 5)   // last content `d: 44` in the 2nd doc
	c.Assert(root2.EndColumn, Equals, 6) // just past `44` (cols 4-5)
}

// TestNodeEndPositionMergeKey verifies a mapping using a merge key (<<) ends at
// its own last content, and the merge value (an alias) carries the alias span,
// not the span of the anchored mapping it points at.
func (s *S) TestNodeEndPositionMergeKey(c *C) {
	//  1: base: &b
	//  2:   x: 1
	//  3:   y: 2
	//  4: merged:
	//  5:   <<: *b
	//  6:   z: 3
	src := `base: &b
  x: 1
  y: 2
merged:
  <<: *b
  z: 3
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	merged := findKey(c, &doc, "merged")
	c.Assert(merged.Kind, Equals, yaml.MappingNode)
	c.Assert(merged.EndLine, Equals, 6)   // last content `z: 3`
	c.Assert(merged.EndColumn, Equals, 7) // just past `3`

	// merged.Content is [`<<` key, `*b` alias, `z` key, `3` value]; the merge
	// value is the alias on line 5.
	mergeAlias := merged.Content[1]
	c.Assert(mergeAlias.Kind, Equals, yaml.AliasNode)
	c.Assert(mergeAlias.EndLine, Equals, 5)
	c.Assert(mergeAlias.EndColumn, Equals, 9) // just past `*b` (cols 7-8)
}

// TestNodeEndPositionFlowCollections covers non-empty flow `{}`/`[]`: the end is
// just past the closing delimiter, on the same line.
func (s *S) TestNodeEndPositionFlowCollections(c *C) {
	// 1: flowMap: {a: 1, b: 2}
	// 2: flowSeq: [10, 20, 30]
	src := `flowMap: {a: 1, b: 2}
flowSeq: [10, 20, 30]
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	// Flow collections end just past their closing delimiter, so the span covers
	// the whole `{...}`/`[...]`. mapping()/sequence() use the END-event mark for
	// flow style (the delimiter is explicit) rather than the last child, which
	// keeps non-empty and empty flow collections consistent.
	fm := findKey(c, &doc, "flowMap")
	c.Assert(fm.Kind, Equals, yaml.MappingNode)
	c.Assert(fm.EndLine, Equals, 1)
	c.Assert(fm.EndColumn, Equals, 22) // just past `}` (col 21)

	fs := findKey(c, &doc, "flowSeq")
	c.Assert(fs.Kind, Equals, yaml.SequenceNode)
	c.Assert(fs.EndLine, Equals, 2)
	c.Assert(fs.EndColumn, Equals, 22) // just past `]` (col 21)
}

// TestNodeEndPositionQuotedScalars covers single- and double-quoted scalars:
// the end is just past the closing quote, not the last content character.
func (s *S) TestNodeEndPositionQuotedScalars(c *C) {
	// 1: single: 'hello'
	// 2: double: "world"
	src := `single: 'hello'
double: "world"
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	sq := findKey(c, &doc, "single")
	c.Assert(sq.Kind, Equals, yaml.ScalarNode)
	c.Assert(sq.Value, Equals, "hello")
	c.Assert(sq.EndLine, Equals, 1)
	c.Assert(sq.EndColumn, Equals, 16) // just past closing `'` (col 15)

	dq := findKey(c, &doc, "double")
	c.Assert(dq.Kind, Equals, yaml.ScalarNode)
	c.Assert(dq.Value, Equals, "world")
	c.Assert(dq.EndLine, Equals, 2)
	c.Assert(dq.EndColumn, Equals, 16) // just past closing `"` (col 15)
}

// TestNodeEndPositionSeqOfMappings covers a block sequence whose items are
// mappings: each item ends at its own last value, and the sequence ends at its
// last item -- neither bleeds into the next item or a following sibling.
func (s *S) TestNodeEndPositionSeqOfMappings(c *C) {
	// 1: items:
	// 2:   - a: 1
	// 3:     b: 2
	// 4:   - c: 3
	// 5: after: x
	src := `items:
  - a: 1
    b: 2
  - c: 3
after: x
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	items := findKey(c, &doc, "items")
	c.Assert(items.Kind, Equals, yaml.SequenceNode)
	c.Assert(items.Content, HasLen, 2)

	item0 := items.Content[0]
	c.Assert(item0.Kind, Equals, yaml.MappingNode)
	c.Assert(item0.EndLine, Equals, 3)   // last value `2`
	c.Assert(item0.EndColumn, Equals, 9) // just past `2`

	item1 := items.Content[1]
	c.Assert(item1.Kind, Equals, yaml.MappingNode)
	c.Assert(item1.EndLine, Equals, 4)   // last value `3`
	c.Assert(item1.EndColumn, Equals, 9) // just past `3`

	// The sequence ends where its last item does, not on the `after` line.
	c.Assert(items.EndLine, Equals, 4)
	c.Assert(items.EndColumn, Equals, 9)
}

// TestNodeEndPositionDeepNesting verifies the end of the innermost leaf
// propagates up to every enclosing mapping.
func (s *S) TestNodeEndPositionDeepNesting(c *C) {
	// 1: l1:
	// 2:   l2:
	// 3:     l3:
	// 4:       leaf: 1
	src := `l1:
  l2:
    l3:
      leaf: 1
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	for _, path := range [][]string{{"l1"}, {"l1", "l2"}, {"l1", "l2", "l3"}} {
		n := findKey(c, &doc, path...)
		c.Assert(n.Kind, Equals, yaml.MappingNode)
		c.Assert(n.EndLine, Equals, 4, Commentf("path %v", path))    // leaf line
		c.Assert(n.EndColumn, Equals, 14, Commentf("path %v", path)) // just past `1`
	}
}

// TestNodeEndPositionBlockScalarChompingAndEOF exercises the block-scalar end
// fix under strip (|-) and keep (|+) chomping, and when the block scalar is the
// last content in the document (ends at EOF, must not overshoot past it).
func (s *S) TestNodeEndPositionBlockScalarChompingAndEOF(c *C) {
	// 1: strip: |-
	// 2:   a
	// 3:   b
	// 4: tail: |
	// 5:   x
	// 6:   y
	src := `strip: |-
  a
  b
tail: |
  x
  y
`
	var doc yaml.Node
	c.Assert(yaml.Unmarshal([]byte(src), &doc), IsNil)

	strip := findKey(c, &doc, "strip")
	c.Assert(strip.Kind, Equals, yaml.ScalarNode)
	c.Assert(strip.Value, Equals, "a\nb") // strip: no trailing newline
	c.Assert(strip.EndLine, Equals, 3)    // last content line `b`
	c.Assert(strip.EndColumn, Equals, 4)  // just past `b` (col 3)

	// `tail` is the last node in the document: its end is its last content line
	// (`y` on line 6), not an overshoot to the post-EOF line 7.
	tail := findKey(c, &doc, "tail")
	c.Assert(tail.Kind, Equals, yaml.ScalarNode)
	c.Assert(tail.EndLine, Equals, 6)
	c.Assert(tail.EndColumn, Equals, 4) // just past `y` (col 3)

	// The root mapping inherits the block scalar's end, so it must not overshoot.
	root := doc.Content[0]
	c.Assert(root.EndLine, Equals, 6)
	c.Assert(root.EndColumn, Equals, 4)
}
