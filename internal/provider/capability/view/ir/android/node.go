package irandroid

import "github.com/dwlhm/nova/internal/core/contract"

// Node is the Android-specific lowered representation of one view node.
type Node struct {
	Kind   string
	Path   []int
	Source contract.Node
	Number bool // text_input vs number_input hint for shared input lowering
}

func LowerNode(node contract.Node, path []int) Node {
	return Node{
		Kind:   node.Kind,
		Path:   append([]int(nil), path...),
		Source: node,
		Number: node.Kind == "number_input",
	}
}
