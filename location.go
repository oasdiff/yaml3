package yaml

import "fmt"

const locTag = "x-location"

func addLocation(key, n *Node) *Node {

	// if this is an element we added, return
	if isLocationElement(key) {
		return n
	}

	switch n.Kind {
	case MappingNode:
		n.Content = append(n.Content, getMapLocation(key, n)...)
	case SequenceNode:
		n.Content = append(n.Content, getSeqLocation(key, n)...)
	}

	return n
}

func isLocationElement(key *Node) bool {
	return key.Line == 0
}

func getMapLocation(key, n *Node) []*Node {

	nodes := make([]*Node, 2)

	addMap(nodes, 0, locTag, append(getKeyLocation(key, "key"), getFieldLocations(n)...))

	return nodes
}

func getSeqLocation(key, n *Node) []*Node {

	nodes := make([]*Node, 1)

	addSeqElement(nodes, 0, "x", getKeyLocation(key, "key"))

	return nodes
}

func getKeyLocation(key *Node, title string) []*Node {
	nodes := make([]*Node, 2)

	addMap(nodes, 0, title, getLocationObject(key))

	return nodes
}

func getFieldLocations(n *Node) []*Node {
	nodes := make([]*Node, 2)

	addMap(nodes, 0, "fields", getFieldLocationsInternal(n, isScalar))

	return nodes
}

type Condition func(key, n *Node) bool

func isScalar(_, n *Node) bool {
	return n.Kind == ScalarNode
}

func all(_, _ *Node) bool {
	return true
}

func getFieldLocationsInternal(n *Node, condition Condition) []*Node {

	nodes := []*Node{}

	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		if condition(n.Content[i], n.Content[i+1]) {
			nodes = append(nodes, getKeyLocation(n.Content[i], n.Content[i].Value)...)
		}
	}
	return nodes
}

func getLocationObject(key *Node) []*Node {
	nodes := make([]*Node, 6)

	addInt(nodes, 0, "line", key.Line)
	addInt(nodes, 2, "column", key.Column)
	addString(nodes, 4, "name", key.Value)

	return nodes
}

func addInt(nodes []*Node, index int, title string, value int) {
	nodes[index] = &Node{
		Kind:  ScalarNode,
		Tag:   "!!str",
		Value: title,
	}
	nodes[index+1] = &Node{
		Kind:  ScalarNode,
		Tag:   "!!int",
		Value: fmt.Sprintf("%d", value),
	}

}

func addString(nodes []*Node, index int, title string, value string) {
	nodes[index] = &Node{
		Kind:  ScalarNode,
		Tag:   "!!str",
		Value: title,
	}
	nodes[index+1] = &Node{
		Kind:  ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}

func addMap(nodes []*Node, index int, title string, content []*Node) {
	nodes[index] = &Node{
		Kind:  ScalarNode,
		Tag:   "!!str",
		Value: title,
	}
	nodes[index+1] = &Node{
		Kind:    MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

func addSeqElement(nodes []*Node, index int, title string, content []*Node) {
	nodes[index] = &Node{
		Kind:    MappingNode,
		Value:   title,
		Tag:     "!!map",
		Content: content,
	}
}
