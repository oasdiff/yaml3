package yaml

import "strconv"

const originTag = "__origin__"

// Origin data is encoded as Nodes so it survives the node -> JSON ->
// UnmarshalJSON conversion the decoder performs on the way to the caller's
// type. That makes every recorded line, column, count and field name its own
// heap-allocated Node carrying its own string, and on a large document those
// dominate the decode: a profile of a 23 MB spec put intNode alone at 486 MB
// of 3.07 GB allocated.
//
// These nodes are immutable leaf scalars -- nothing appends to their Content
// or rewrites their Value -- so equal values can share one Node. Two caches
// cover the two ways a spec repeats itself.
//
// Neither cache needs a size bound. The strings interned are mapping keys and
// scalar sequence items (not values like descriptions), so the distinct set is
// small by construction, and in the worst case the cache costs one pointer per
// Node that would have been allocated regardless.

// maxCachedInt covers essentially every column, line delta and count. Only an
// absolute line number in a large document exceeds it, and there is one of
// those per mapping against many small ones.
const maxCachedInt = 1024

var smallIntNodes = func() [maxCachedInt]*Node {
	var nodes [maxCachedInt]*Node
	for i := range nodes {
		nodes[i] = &Node{Kind: ScalarNode, Tag: "!!int", Value: strconv.Itoa(i)}
	}
	return nodes
}()

// originCache interns the nodes for one decode. It is per-decode rather than
// global so it needs no locking and is reclaimed with the decoder.
//
// It keys on the string itself and holds no notion of "the" file, because the
// file recorded in an origin is not constant: a $ref carries the decode into
// another document, and the origin has to name the file the element actually
// came from. Interning by value is correct either way -- a second file is
// simply a second entry.
type originCache struct {
	strs map[string]*Node
}

func newOriginCache() *originCache {
	return &originCache{strs: make(map[string]*Node)}
}

func (c *originCache) str(v string) *Node {
	if n, ok := c.strs[v]; ok {
		return n
	}
	n := &Node{Kind: ScalarNode, Tag: "!!str", Value: v}
	c.strs[v] = n
	return n
}

func isScalar(n *Node) bool {
	return n.Kind == ScalarNode
}

func isSequence(n *Node) bool {
	return n.Kind == SequenceNode
}

func isMapping(n *Node) bool {
	return n.Kind == MappingNode
}

func addOriginInSeq(n *Node, file string, c *originCache) *Node {
	if !isMapping(n) || len(n.Content) == 0 {
		return n
	}
	// in case of a sequence, we use the first element as the key
	return addOrigin(n.Content[0], n, file, c)
}

func addOriginInMap(key, n *Node, file string, c *originCache) *Node {
	if !isMapping(n) {
		return n
	}
	return addOrigin(key, n, file, c)
}

// addOrigin injects a compact __origin__ sequence into the mapping node n.
//
// Format: [file, key_name, key_line, key_col, nf, f1_name, f1_delta, f1_col, ..., ns, s1_name, s1_count, s1_l0_delta, s1_c0, ..., end_delta, end_col]
//
//   - file: source file path
//   - key_name:  the YAML key whose value is this mapping
//   - key_line, key_col: location of that key
//   - nf: number of scalar+sequence fields recorded
//   - per field: name (string), line delta from key_line (int), column (int)
//   - ns: number of sequence fields that have item locations
//   - per sequence: name (string), item count (int), then count × (line delta, col)
//   - end_delta, end_col: end of the whole mapping block — line delta from
//     key_line and absolute column of the position just past its last content.
//     Appended last so a consumer that stops after the sequences section
//     simply ignores it (backward compatible).
func addOrigin(key, n *Node, file string, c *originCache) *Node {
	if isOrigin(key) {
		return n
	}

	seq := buildOriginSeq(key, n, file, c)
	n.Content = append(n.Content,
		&Node{Kind: ScalarNode, Tag: "!!str", Value: originTag}, // Line==0 → isOrigin
		&Node{Kind: SequenceNode, Tag: "!!seq", Content: seq},
	)
	return n
}

func buildOriginSeq(key, n *Node, file string, c *originCache) []*Node {
	// Header: file, key_name, key_line, key_col
	nodes := []*Node{
		c.str(file),
		c.str(key.Value),
		intNode(key.Line),
		intNode(key.Column),
	}

	// Collect field and sequence data.
	var fieldNodes []*Node // nf × (name, delta, col)
	var seqNodes []*Node   // ns × (name, count, (delta, col)…)
	nf, ns := 0, 0

	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		k := n.Content[i]
		v := n.Content[i+1]
		if isOrigin(k) {
			continue
		}
		// Record the location of this field's key.
		nf++
		fieldNodes = append(fieldNodes,
			c.str(k.Value),
			intNode(k.Line-key.Line),
			intNode(k.Column),
		)
		if isSequence(v) {
			// Record locations of scalar items within the sequence.
			// Format per item: value_str, line_delta, col
			var itemNodes []*Node
			for _, item := range v.Content {
				if item.Kind == ScalarNode {
					itemNodes = append(itemNodes,
						c.str(item.Value),
						intNode(item.Line-key.Line),
						intNode(item.Column),
					)
				}
			}
			if len(itemNodes) > 0 {
				ns++
				seqNodes = append(seqNodes, c.str(k.Value), intNode(len(itemNodes)/3))
				seqNodes = append(seqNodes, itemNodes...)
			}
		}
	}

	nodes = append(nodes, intNode(nf))
	nodes = append(nodes, fieldNodes...)
	nodes = append(nodes, intNode(ns))
	nodes = append(nodes, seqNodes...)

	// Block end: line delta from key_line and absolute end column of the whole
	// mapping. Lets a consumer reconstruct the full block span
	// [key_line, key_line+end_delta] -- e.g. an entire endpoint operation block.
	endDelta, endCol := 0, 0
	if n.EndLine > 0 {
		endDelta = n.EndLine - key.Line
		endCol = n.EndColumn
	}
	nodes = append(nodes, intNode(endDelta), intNode(endCol))
	return nodes
}

// isOrigin returns true if the key is a synthetic origin node.
// Synthetic nodes have Line==0 (real YAML lines are 1-based).
func isOrigin(key *Node) bool {
	return key.Line == 0
}

func intNode(v int) *Node {
	if 0 <= v && v < maxCachedInt {
		return smallIntNodes[v]
	}
	return &Node{Kind: ScalarNode, Tag: "!!int", Value: strconv.Itoa(v)}
}
