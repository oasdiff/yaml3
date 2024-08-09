package yaml

import "fmt"

const locTag = "x-location"

func isScalar(n *Node) bool {
	return n.Kind == ScalarNode
}

func addLocation(key, n *Node) *Node {

	// if this is an element we added, return
	if isLocationElement(key) {
		return n
	}

	switch n.Kind {
	case MappingNode:
		n.Content = append(n.Content, getNamedMap(locTag, append(getKeyLocation(key), getNamedMap("fields", getFieldLocations(n))...))...)
	case SequenceNode:
		n.Content = append(n.Content, getMap(getNamedMap(locTag, append(getKeyLocation(key), getNamedMap("elements", getSequenceLocations(n))...))))
	}

	return n
}

func getFieldLocations(n *Node) []*Node {

	l := len(n.Content)
	size := 0
	for i := 0; i < l; i += 2 {
		if isScalar(n.Content[i+1]) {
			size += 2
		}
	}

	nodes := make([]*Node, 0, size)
	for i := 0; i < l; i += 2 {
		if isScalar(n.Content[i+1]) {
			nodes = append(nodes, getNodeLocation(n.Content[i])...)
		}
	}
	return nodes
}

func getSequenceLocations(n *Node) []*Node {

	size := 0
	for _, element := range n.Content {
		if isScalar(element) {
			size += 2
		}
	}

	nodes := make([]*Node, 0, size)
	for _, element := range n.Content {
		if isScalar(element) {
			nodes = append(nodes, getNodeLocation(element)...)
		}
	}

	return nodes
}

func isLocationElement(key *Node) bool {
	// we rely on the fact that the line number is 0 for elements that we added
	// a better design would be to use a dedicated field in the node
	return key.Line == 0
}

func getNodeLocation(n *Node) []*Node {
	return getNamedMap(n.Value, getLocationObject(n))
}

func getKeyLocation(n *Node) []*Node {
	return getNamedMap("key", getLocationObject(n))
}

func getNamedMap(title string, content []*Node) []*Node {
	return []*Node{
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: title,
		},
		getMap(content),
	}
}

func getMap(content []*Node) *Node {
	return &Node{
		Kind:    MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

func getLocationObject(key *Node) []*Node {
	return []*Node{
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "line",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!int",
			Value: fmt.Sprintf("%d", key.Line),
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "col",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!int",
			Value: fmt.Sprintf("%d", key.Column),
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "name",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!string",
			Value: key.Value,
		},
	}
}
