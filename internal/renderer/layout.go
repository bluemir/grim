package renderer

import (
	"fmt"
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

	// Icon shapes (user, bot): fixed-proportion icon with label below
	iconNodeW  = 60.0 // icon area width
	iconNodeH  = 60.0 // icon area height
	iconLabelH = 20.0 // label area below icon

	lineHeight = 20.0 // height per line for multiline text
)

// NodeStyle holds visual properties for a node.
type NodeStyle struct {
	Fill         string
	Stroke       string
	StrokeWidth  float64
	FontSize     float64
	FontColor    string
	StrokeDash   string  // "" = solid; e.g. "5,3"
	Shadow       float64 // 0 = none; blur radius in px
	Opacity      float64 // 0 = use default (1.0)
	BorderRadius float64 // -1 = shape default; >=0 = explicit rx value
}

// EdgeStyle holds visual properties for an edge.
type EdgeStyle struct {
	Stroke      string
	StrokeWidth float64
	FontSize    float64
}

// Point represents an x/y coordinate for waypoint routing.
type Point struct{ X, Y float64 }

// LayoutNode represents a positioned node ready for SVG rendering.
type LayoutNode struct {
	ID         string
	Line       int
	Label      string
	TextFormat string // "" or "plain" = plain text, "markdown" = markdown
	X, Y       float64
	W, H       float64
	Fixed      bool
	Shape      string // "rectangle", "rounded", "circle", "diamond", "cylinder", "cloud", "hexagon", "parallelogram"
	Stack      int    // number of stacked shadow copies (0 or 1 = no shadow)
	Style      NodeStyle
	Children   []*LayoutNode
}

// LayoutEdge represents a positioned edge ready for SVG rendering.
type LayoutEdge struct {
	ID         string
	Line       int
	FromID     string
	ToID       string
	Label      string
	Direction  string // "" = forward (->), "reverse" = <-, "bidirectional" = <->
	Style      EdgeStyle
	Waypoints  []Point // optional bend points
	FromAnchor string  // e.g. "90", "H3", "H1:30", "top", "" = auto
	ToAnchor   string
}

// LayoutResult contains all positioned elements and canvas dimensions.
type LayoutResult struct {
	Width  float64
	Height float64
	Nodes  []*LayoutNode
	Edges  []*LayoutEdge
}

var defaultNodeStyle = NodeStyle{
	Fill:         "#FFFFFF",
	Stroke:       "#333333",
	StrokeWidth:  1,
	FontSize:     14,
	FontColor:    "#333333",
	Opacity:      1.0,
	BorderRadius: -1, // "not set" → use shape default
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
			// Collect edges from nested blocks
			result.Edges = append(result.Edges, collectNestedEdges(s, "")...)
		case *parser.EdgeDecl:
			edge := buildEdge(s)
			result.Edges = append(result.Edges, edge)
		}
	}

	// Phase 1.5: reparent top-level nodes with dotted IDs under their proper parent
	reparentedSet := make(map[string]bool)
	for _, node := range result.Nodes {
		if dotIdx := strings.LastIndex(node.ID, "."); dotIdx != -1 {
			parentID := node.ID[:dotIdx]
			parent := ensureNode(parentID, nodeMap, result, &unfixedNodes)
			parent.Children = append(parent.Children, node)
			reparentedSet[node.ID] = true
		}
	}
	if len(reparentedSet) > 0 {
		filtered := result.Nodes[:0]
		for _, n := range result.Nodes {
			if !reparentedSet[n.ID] {
				filtered = append(filtered, n)
			}
		}
		result.Nodes = filtered

		filteredUnfixed := unfixedNodes[:0]
		for _, n := range unfixedNodes {
			if !reparentedSet[n.ID] {
				filteredUnfixed = append(filteredUnfixed, n)
			}
		}
		unfixedNodes = filteredUnfixed
	}

	// Phase 2: resolve implicit nodes (referenced in edges but not declared)
	for _, edge := range result.Edges {
		for _, id := range []string{edge.FromID, edge.ToID} {
			ensureNode(id, nodeMap, result, &unfixedNodes)
		}
	}

	// Phase 3: layout children first (so parent sizes are finalized)
	for _, node := range result.Nodes {
		layoutChildren(node)
	}

	// Phase 4: layout unfixed top-level nodes using grid (with correct sizes)
	var fixedMaxBottom float64
	for _, node := range result.Nodes {
		if node.Fixed {
			if bottom := node.Y + node.H; bottom > fixedMaxBottom {
				fixedMaxBottom = bottom
			}
		}
	}
	var unfixedStartY float64
	if fixedMaxBottom > 0 {
		unfixedStartY = fixedMaxBottom + nodePadY
	}
	layoutGrid(unfixedNodes, gridTotalWidth, unfixedStartY)

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
		node.TextFormat = decl.Label.Format
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
				if s, ok := m.Values["stack"]; ok {
					if n := int(toFloat(s)); n > 1 {
						node.Stack = n
					}
				}
			case *parser.StyleMeta:
				applyNodeStyle(&node.Style, m.Values)
			case *parser.TextMeta:
				node.Label = m.Value
				node.TextFormat = m.Format
			case *parser.ShapeMeta:
				node.Shape = m.Value
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
		if node.Shape == "user" || node.Shape == "bot" {
			// Icon shapes: maintain aspect ratio; iconLabelH is always added to h
			switch {
			case node.W == 0 && node.H == 0:
				node.W = iconNodeW
				node.H = iconNodeH + iconLabelH
			case node.W == 0: // h given → scale w
				node.W = iconNodeW * (node.H - iconLabelH) / iconNodeH
			default: // w given → scale h
				node.H = iconNodeH*node.W/iconNodeW + iconLabelH
			}
		} else {
			w, h := computeNodeSize(node.Label, node.Children)
			if node.W == 0 {
				node.W = w
			}
			if node.H == 0 {
				node.H = h
			}
		}
	}

	return node
}

func buildEdge(decl *parser.EdgeDecl) *LayoutEdge {
	fromID := strings.Join(decl.From, ".")
	toID := strings.Join(decl.To, ".")
	edge := &LayoutEdge{
		ID:        fromID + "--" + toID,
		Line:      decl.Line,
		FromID:    fromID,
		ToID:      toID,
		Direction: decl.Direction,
		Style:     defaultEdgeStyle,
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
			case *parser.EdgeMeta:
				if raw, ok := m.Values["waypoints"]; ok {
					if arr, ok := raw.([]interface{}); ok {
						for _, item := range arr {
							if pt, ok := item.(map[string]interface{}); ok {
								edge.Waypoints = append(edge.Waypoints, Point{
									X: floatVal(pt, "x", 0),
									Y: floatVal(pt, "y", 0),
								})
							}
						}
					}
				}
				if raw, ok := m.Values["anchors"]; ok {
					if arr, ok := raw.([]interface{}); ok {
						if len(arr) >= 1 {
							edge.FromAnchor = fmt.Sprint(arr[0])
						}
						if len(arr) >= 2 {
							edge.ToAnchor = fmt.Sprint(arr[1])
						}
					}
				}
			}
		}
	}

	return edge
}

func computeNodeSize(label string, children []*LayoutNode) (float64, float64) {
	lines := strings.Split(label, "\n")
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	textW := float64(maxLen)*charWidth + nodePadX
	w := math.Max(defaultNodeW, textW)
	h := math.Max(defaultNodeH, nodePadY*2+float64(len(lines))*lineHeight)

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

// layoutGrid places unfixed nodes in a 12-column grid with variable row heights.
func layoutGrid(nodes []*LayoutNode, totalWidth float64, startY float64) {
	if len(nodes) == 0 {
		return
	}

	colWidth := totalWidth / gridColumns
	// Grid occupancy: rows of columns (true = occupied)
	grid := make([][]bool, 0)
	// Track the maximum height of each row
	rowHeights := make([]float64, 0)
	// Track which row each node was placed in
	type placement struct {
		node *LayoutNode
		col  int
		row  int
	}
	var placements []placement

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
				rowHeights = append(rowHeights, defaultNodeH)
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
					// Place node in column
					node.X = float64(col) * colWidth
					// Update row height to accommodate this node
					if node.H > rowHeights[row] {
						rowHeights[row] = node.H
					}
					placements = append(placements, placement{node: node, col: col, row: row})
					// Mark cells as occupied
					for r := row; r < row+rowSpan; r++ {
						for len(grid) <= r {
							grid = append(grid, make([]bool, gridColumns))
							rowHeights = append(rowHeights, defaultNodeH)
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

	// Recompute Y positions using cumulative row heights
	for _, p := range placements {
		var y float64
		for r := 0; r < p.row; r++ {
			y += rowHeights[r] + nodePadY
		}
		p.node.Y = startY + y
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
	layoutGrid(unfixed, interiorW, 0)

	// Offset all children relative to parent's content area.
	// Fixed children (manual @layout) use only the header height as origin.
	// Auto-layout children additionally get nestedPadding on both axes.
	for _, child := range parent.Children {
		if child.Fixed {
			child.Y += labelHeight
		} else {
			child.X += nestedPadding
			child.Y += labelHeight + nestedPadding
		}
	}

	// Recalculate parent size from actual child positions
	lines := strings.Split(parent.Label, "\n")
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	textW := float64(maxLen)*charWidth + nodePadX
	parent.W = math.Max(defaultNodeW, textW)
	parent.H = labelHeight + nestedPadding // minimum: label area + bottom padding

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

func collectNestedEdges(decl *parser.NodeDecl, parentPrefix string) []*LayoutEdge {
	if decl.Block == nil {
		return nil
	}
	id := parentPrefix + strings.Join(decl.Path, ".")
	childPrefix := id + "."
	var edges []*LayoutEdge
	for _, child := range decl.Block.Children {
		switch c := child.(type) {
		case *parser.EdgeDecl:
			edge := buildEdge(c)
			edge.FromID = childPrefix + edge.FromID
			edge.ToID = childPrefix + edge.ToID
			edge.ID = edge.FromID + "--" + edge.ToID
			edges = append(edges, edge)
		case *parser.NodeDecl:
			edges = append(edges, collectNestedEdges(c, childPrefix)...)
		}
	}
	return edges
}

func registerNodes(node *LayoutNode, nodeMap map[string]*LayoutNode) {
	nodeMap[node.ID] = node
	for _, child := range node.Children {
		registerNodes(child, nodeMap)
	}
}

// ensureNode guarantees a node with the given ID exists in nodeMap,
// recursively creating intermediate parent nodes as needed.
func ensureNode(id string, nodeMap map[string]*LayoutNode, result *LayoutResult, unfixedNodes *[]*LayoutNode) *LayoutNode {
	if node, ok := nodeMap[id]; ok {
		return node
	}

	node := &LayoutNode{
		ID:    id,
		Label: lastSegment(id),
		Style: defaultNodeStyle,
	}
	node.W, node.H = computeNodeSize(node.Label, nil)
	nodeMap[id] = node

	if dotIdx := strings.LastIndex(id, "."); dotIdx != -1 {
		parentID := id[:dotIdx]
		parent := ensureNode(parentID, nodeMap, result, unfixedNodes)
		parent.Children = append(parent.Children, node)
	} else {
		result.Nodes = append(result.Nodes, node)
		*unfixedNodes = append(*unfixedNodes, node)
	}

	return node
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
		style.Fill = resolveColor(toString(v))
	}
	if v, ok := values["stroke"]; ok {
		style.Stroke = resolveColor(toString(v))
	}
	if v, ok := values["stroke-width"]; ok {
		style.StrokeWidth = toFloat(v)
	}
	if v, ok := values["font-size"]; ok {
		style.FontSize = toFloat(v)
	}
	if v, ok := values["font-color"]; ok {
		style.FontColor = resolveColor(toString(v))
	}
	if v, ok := values["dash"]; ok {
		switch d := v.(type) {
		case []interface{}:
			parts := make([]string, 0, len(d))
			for _, item := range d {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			style.StrokeDash = strings.Join(parts, ",")
		case string:
			style.StrokeDash = d
		}
	}
	if v, ok := values["shadow"]; ok {
		style.Shadow = toFloat(v)
	}
	if v, ok := values["opacity"]; ok {
		style.Opacity = toFloat(v)
	}
	if v, ok := values["border-radius"]; ok {
		style.BorderRadius = toFloat(v)
	}
}

func applyEdgeStyle(style *EdgeStyle, values map[string]interface{}) {
	if v, ok := values["stroke"]; ok {
		style.Stroke = resolveColor(toString(v))
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
