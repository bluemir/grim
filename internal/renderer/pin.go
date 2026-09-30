package renderer

import (
	"math"
	"strings"

	"github.com/bluemir/grim/internal/parser"
)

// PinSiblings writes @layout {x, y} for every auto-laid-out sibling of nodeID
// at the position it is currently drawn at. Moving or resizing one node
// changes the auto grid (fixed nodes push the grid below them, and a node
// leaving the grid shifts the ones after it), so pinning its siblings first
// keeps them exactly where the user sees them. Only position is pinned; sizes
// stay automatic. Nodes at other nesting levels are left alone.
//
// The document is modified in place; call parser.Format(doc) to get the source.
func PinSiblings(doc *parser.Document, nodeID string) {
	layout := BuildLayout(doc)

	siblings := layout.Nodes
	nested := false
	if i := strings.LastIndex(nodeID, "."); i >= 0 {
		parent := findLayoutNode(layout.Nodes, nodeID[:i])
		if parent == nil {
			return
		}
		siblings = parent.Children
		nested = true
	}

	for _, n := range siblings {
		if n.ID == nodeID || n.Fixed {
			continue
		}
		y := n.Y
		if nested {
			// Fixed children are drawn below the parent's label; auto-laid-out
			// ones already include it, so take it out to land on the same spot.
			y -= labelHeight
		}
		parser.SetLayout(doc, n.ID, int64(math.Round(n.X)), int64(math.Round(y)))
	}
}

func findLayoutNode(nodes []*LayoutNode, id string) *LayoutNode {
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
		if strings.HasPrefix(id, n.ID+".") {
			if found := findLayoutNode(n.Children, id); found != nil {
				return found
			}
		}
	}
	return nil
}
