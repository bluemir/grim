package renderer

import (
	"math"
	"strings"

	"github.com/bluemir/grim/internal/parser"
)

const (
	gridTotalWidth = 960.0
	gridColumns    = 12
	gridColWidth   = gridTotalWidth / gridColumns
	defaultNodeW   = 100.0
	defaultNodeH   = 40.0
	charWidth      = 8.0  // approximate width per character
	nodePadX       = 24.0 // horizontal padding for text
	nodePadY       = 8.0  // vertical padding
	canvasMargin   = 20.0 // margin around all content
	nestedPadding  = 16.0 // padding inside parent node for children
	labelHeight    = 24.0 // height reserved for parent label at top
)

// NodeStyle holds visual properties for a node.
type NodeStyle struct {
	Fill        string
	Stroke      string
	StrokeWidth float64
	FontSize    float64
	FontColor   string
}

// EdgeStyle holds visual properties for an edge.
type EdgeStyle struct {
	Stroke      string
	StrokeWidth float64
	FontSize    float64
}

// LayoutNode represents a positioned node ready for SVG rendering.
type LayoutNode struct {
	ID       string
	Line     int
	Label    string
	X, Y     float64
	W, H     float64
	Fixed    bool
	Style    NodeStyle
	Children []*LayoutNode
}

// LayoutEdge represents a positioned edge ready for SVG rendering.
type LayoutEdge struct {
	ID     string
	Line   int
	FromID string
	ToID   string
	Label  string
	Style  EdgeStyle
}

// LayoutResult contains all positioned elements and canvas dimensions.
type LayoutResult struct {
	Width  float64
	Height float64
	Nodes  []*LayoutNode
	Edges  []*LayoutEdge
}

var defaultNodeStyle = NodeStyle{
	Fill:        "#FFFFFF",
	Stroke:      "#333333",
	StrokeWidth: 1,
	FontSize:    14,
	FontColor:   "#333333",
}

var defaultEdgeStyle = EdgeStyle{
	Stroke:      "#333333",
	StrokeWidth: 1,
	FontSize:    12,
}

// BuildLayout walks the AST and produces a positioned layout.
func BuildLayout(doc *parser.Document) *LayoutResult {
	result := &LayoutResult{}
	nodeMap := make(map[string]*LayoutNode) // ID -> node for edge lookup

	// Phase 1: collect all nodes and edges from AST
	var unfixedNodes []*LayoutNode

	for _, stmt := range doc.Statements {
		switch s := stmt.(type) {
		case *parser.NodeDecl:
			node := buildNode(s, "")
			result.Nodes = append(result.Nodes, node)
			registerNodes(node, nodeMap)
			if !node.Fixed {
				unfixedNodes = append(unfixedNodes, node)
			}
		case *parser.EdgeDecl:
			edge := buildEdge(s)
			result.Edges = append(result.Edges, edge)
		}
	}

	// Phase 2: resolve implicit nodes (referenced in edges but not declared)
	for _, edge := range result.Edges {
		for _, id := range []string{edge.FromID, edge.ToID} {
			if _, ok := nodeMap[id]; !ok {
				node := &LayoutNode{
					ID:    id,
					Label: lastSegment(id),
					Style: defaultNodeStyle,
				}
				node.W, node.H = computeNodeSize(node.Label, nil)
				result.Nodes = append(result.Nodes, node)
				nodeMap[id] = node
				unfixedNodes = append(unfixedNodes, node)
			}
		}
	}

	// Phase 3: layout unfixed nodes using grid
	layoutGrid(unfixedNodes, gridTotalWidth)

	// Phase 4: layout children of each node
	for _, node := range result.Nodes {
		layoutChildren(node)
	}

	// Phase 5: compute canvas bounding box
	result.Width, result.Height = computeBounds(result.Nodes)

	return result
}

func buildNode(decl *parser.NodeDecl, parentPrefix string) *LayoutNode {
	id := parentPrefix + strings.Join(decl.Path, ".")
	node := &LayoutNode{
		ID:    id,
		Line:  decl.Line,
		Style: defaultNodeStyle,
	}

	// Extract label from shorthand
	if decl.Label != nil {
		node.Label = decl.Label.Value
	}

	// Process block metadata and children
	if decl.Block != nil {
		for _, meta := range decl.Block.Metadata {
			switch m := meta.(type) {
			case *parser.LayoutMeta:
				node.Fixed = true
				node.X = floatVal(m.Values, "x", 0)
				node.Y = floatVal(m.Values, "y", 0)
				if w, ok := m.Values["w"]; ok {
					node.W = toFloat(w)
				}
				if h, ok := m.Values["h"]; ok {
					node.H = toFloat(h)
				}
			case *parser.StyleMeta:
				applyNodeStyle(&node.Style, m.Values)
			case *parser.TextMeta:
				node.Label = m.Value
			}
		}

		// Recurse into children
		childPrefix := id + "."
		for _, child := range decl.Block.Children {
			if childDecl, ok := child.(*parser.NodeDecl); ok {
				childNode := buildNode(childDecl, childPrefix)
				node.Children = append(node.Children, childNode)
			}
		}
	}

	// If no label was set, use the last path segment as label
	if node.Label == "" {
		node.Label = lastSegment(id)
	}

	// Compute size if not fixed or if fixed without explicit w/h
	if node.W == 0 || node.H == 0 {
		w, h := computeNodeSize(node.Label, node.Children)
		if node.W == 0 {
			node.W = w
		}
		if node.H == 0 {
			node.H = h
		}
	}

	return node
}

func buildEdge(decl *parser.EdgeDecl) *LayoutEdge {
	fromID := strings.Join(decl.From, ".")
	toID := strings.Join(decl.To, ".")
	edge := &LayoutEdge{
		ID:     fromID + "--" + toID,
		Line:   decl.Line,
		FromID: fromID,
		ToID:   toID,
		Style:  defaultEdgeStyle,
	}

	if decl.Label != nil {
		edge.Label = decl.Label.Value
	}

	if decl.Block != nil {
		for _, meta := range decl.Block.Metadata {
			switch m := meta.(type) {
			case *parser.TextMeta:
				edge.Label = m.Value
			case *parser.StyleMeta:
				applyEdgeStyle(&edge.Style, m.Values)
			}
		}
	}

	return edge
}

func computeNodeSize(label string, children []*LayoutNode) (float64, float64) {
	textW := float64(len(label))*charWidth + nodePadX
	w := math.Max(defaultNodeW, textW)
	h := defaultNodeH

	if len(children) > 0 {
		// Parent node needs to be bigger to contain children
		childBoundsW, childBoundsH := computeChildBounds(children)
		w = math.Max(w, childBoundsW+nestedPadding*2)
		h = labelHeight + childBoundsH + nestedPadding*2
	}

	return w, h
}

func computeChildBounds(children []*LayoutNode) (float64, float64) {
	if len(children) == 0 {
		return 0, 0
	}
	var maxX, maxY float64
	for _, c := range children {
		right := c.X + c.W
		bottom := c.Y + c.H
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}
	return maxX, maxY
}

// layoutGrid places unfixed nodes in a 12-column grid.
func layoutGrid(nodes []*LayoutNode, totalWidth float64) {
	if len(nodes) == 0 {
		return
	}

	colWidth := totalWidth / gridColumns
	// Grid occupancy: rows of columns (true = occupied)
	grid := make([][]bool, 0)

	for _, node := range nodes {
		colSpan := int(math.Ceil(node.W / colWidth))
		if colSpan > gridColumns {
			colSpan = gridColumns
		}
		if colSpan < 1 {
			colSpan = 1
		}

		rowSpan := 1
		placed := false

		for row := 0; !placed; row++ {
			// Ensure row exists
			for len(grid) <= row {
				grid = append(grid, make([]bool, gridColumns))
			}

			for col := 0; col <= gridColumns-colSpan; col++ {
				fits := true
				for c := col; c < col+colSpan; c++ {
					if grid[row][c] {
						fits = false
						break
					}
				}
				if fits {
					// Place node
					node.X = float64(col) * colWidth
					node.Y = float64(row) * (defaultNodeH + nodePadY)
					// Mark cells as occupied
					for r := row; r < row+rowSpan; r++ {
						for len(grid) <= r {
							grid = append(grid, make([]bool, gridColumns))
						}
						for c := col; c < col+colSpan; c++ {
							grid[r][c] = true
						}
					}
					placed = true
					break
				}
			}
		}
	}
}

// layoutChildren positions children within a parent node.
func layoutChildren(parent *LayoutNode) {
	if len(parent.Children) == 0 {
		return
	}

	var unfixed []*LayoutNode
	childMap := make(map[string]*LayoutNode)

	for _, child := range parent.Children {
		childMap[child.ID] = child
		if !child.Fixed {
			unfixed = append(unfixed, child)
		}
		// Recursively layout nested children
		layoutChildren(child)
	}

	// Layout unfixed children within parent's interior
	interiorW := parent.W - nestedPadding*2
	if interiorW < defaultNodeW {
		interiorW = defaultNodeW
	}
	layoutGrid(unfixed, interiorW)

	// Offset all children relative to parent's content area
	for _, child := range parent.Children {
		child.X += nestedPadding
		child.Y += labelHeight + nestedPadding
	}

	// Expand parent to fit children if needed
	for _, child := range parent.Children {
		needW := child.X + child.W + nestedPadding
		needH := child.Y + child.H + nestedPadding
		if needW > parent.W {
			parent.W = needW
		}
		if needH > parent.H {
			parent.H = needH
		}
	}
}

func computeBounds(nodes []*LayoutNode) (float64, float64) {
	var maxX, maxY float64
	for _, n := range nodes {
		right := n.X + n.W + canvasMargin
		bottom := n.Y + n.H + canvasMargin
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}
	return maxX + canvasMargin, maxY + canvasMargin
}

func registerNodes(node *LayoutNode, nodeMap map[string]*LayoutNode) {
	nodeMap[node.ID] = node
	for _, child := range node.Children {
		registerNodes(child, nodeMap)
	}
}

func lastSegment(id string) string {
	parts := strings.Split(id, ".")
	return parts[len(parts)-1]
}

func floatVal(m map[string]interface{}, key string, def float64) float64 {
	v, ok := m[key]
	if !ok {
		return def
	}
	return toFloat(v)
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		return 0
	default:
		return 0
	}
}

func applyNodeStyle(style *NodeStyle, values map[string]interface{}) {
	if v, ok := values["fill"]; ok {
		style.Fill = toString(v)
	}
	if v, ok := values["stroke"]; ok {
		style.Stroke = toString(v)
	}
	if v, ok := values["stroke-width"]; ok {
		style.StrokeWidth = toFloat(v)
	}
	if v, ok := values["font-size"]; ok {
		style.FontSize = toFloat(v)
	}
	if v, ok := values["font-color"]; ok {
		style.FontColor = toString(v)
	}
}

func applyEdgeStyle(style *EdgeStyle, values map[string]interface{}) {
	if v, ok := values["stroke"]; ok {
		style.Stroke = toString(v)
	}
	if v, ok := values["stroke-width"]; ok {
		style.StrokeWidth = toFloat(v)
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
