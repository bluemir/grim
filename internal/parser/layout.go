package parser

import "strings"

// SetLayout sets the @layout x, y values for a node identified by nodeID.
// nodeID is a dot-separated path like "parent.child".
// The document is modified in place; call Format(doc) to get updated source.
func SetLayout(doc *Document, nodeID string, x, y int64) {
	segments := strings.Split(nodeID, ".")
	node := findNodeByPath(doc.Statements, segments)
	if node == nil {
		node = createNodeByPath(doc, segments)
	}
	updateLayoutMeta(node, x, y)
}

// findNodeByPath recursively searches for a node matching the given path segments.
// Handles both flat dotted (NodeDecl.Path = ["parent","child"]) and nested block patterns.
func findNodeByPath(stmts []Statement, segments []string) *NodeDecl {
	for _, stmt := range stmts {
		node, ok := stmt.(*NodeDecl)
		if !ok {
			continue
		}
		// Exact match
		if slicesEqual(node.Path, segments) {
			return node
		}
		// Prefix match: node.Path is a prefix of segments — recurse into children
		if len(node.Path) < len(segments) && slicesEqual(node.Path, segments[:len(node.Path)]) {
			if node.Block != nil {
				remaining := segments[len(node.Path):]
				if found := findNodeByPath(node.Block.Children, remaining); found != nil {
					return found
				}
			}
		}
	}
	return nil
}

// createNodeByPath finds the deepest existing ancestor and inserts a new node,
// always using nested block format. Returns the leaf NodeDecl that was created.
func createNodeByPath(doc *Document, segments []string) *NodeDecl {
	// Try to find the deepest existing ancestor
	for depth := len(segments) - 1; depth > 0; depth-- {
		ancestor := findNodeByPath(doc.Statements, segments[:depth])
		if ancestor != nil {
			if ancestor.Block == nil {
				ancestor.Block = &Block{}
			}
			remaining := segments[depth:]
			newNode := buildNestedNode(remaining)
			ancestor.Block.Children = append(ancestor.Block.Children, newNode)
			return deepestNode(newNode, remaining)
		}
	}
	// No ancestor found — create at top level as nested structure
	newNode := buildNestedNode(segments)
	doc.Statements = append(doc.Statements, newNode)
	return deepestNode(newNode, segments)
}

// buildNestedNode creates a nested NodeDecl chain for the given path segments.
// e.g. ["a","b"] → NodeDecl{Path:["a"], Block:{Children:[NodeDecl{Path:["b"]}]}}
func buildNestedNode(segments []string) *NodeDecl {
	if len(segments) == 1 {
		return &NodeDecl{Path: []string{segments[0]}}
	}
	inner := buildNestedNode(segments[1:])
	return &NodeDecl{
		Path:  []string{segments[0]},
		Block: &Block{Children: []Statement{inner}},
	}
}

// deepestNode returns the leaf NodeDecl in a nested chain built by buildNestedNode.
func deepestNode(node *NodeDecl, segments []string) *NodeDecl {
	current := node
	for i := 1; i < len(segments); i++ {
		if current.Block == nil {
			break
		}
		for _, child := range current.Block.Children {
			if n, ok := child.(*NodeDecl); ok {
				current = n
				break
			}
		}
	}
	return current
}

// updateLayoutMeta updates or inserts @layout { x, y } on the node.
// Preserves existing keys (w, h, etc.) in the LayoutMeta.
func updateLayoutMeta(node *NodeDecl, x, y int64) {
	if node.Block == nil {
		node.Block = &Block{}
	}
	for _, m := range node.Block.Metadata {
		if lm, ok := m.(*LayoutMeta); ok {
			lm.Values["x"] = x
			lm.Values["y"] = y
			return
		}
	}
	// No existing @layout — prepend to metadata
	lm := &LayoutMeta{
		Values: map[string]interface{}{"x": x, "y": y},
	}
	node.Block.Metadata = append([]Metadata{lm}, node.Block.Metadata...)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
