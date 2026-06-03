package irweb

import "github.com/dwlhm/nova/internal/core/contract"

// Node is the Web-specific lowered representation of one view node.
type Node struct {
	Kind   string
	Path   []int
	Source contract.Node
}

func LowerNode(node contract.Node, path []int) Node {
	return Node{
		Kind:   node.Kind,
		Path:   append([]int(nil), path...),
		Source: node,
	}
}
