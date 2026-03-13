package renderer

import (
	"fmt"
	"strings"
)

const defaultGap = 4.0

// RenderSVG generates an SVG string from a LayoutResult.
func RenderSVG(layout *LayoutResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f">`+"\n",
		layout.Width, layout.Height,
	))

	// Defs: arrowhead marker
	b.WriteString(`  <defs>` + "\n")
	b.WriteString(`    <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">` + "\n")
	b.WriteString(`      <polygon points="0 0, 10 3.5, 0 7" fill="#333"/>` + "\n")
	b.WriteString(`    </marker>` + "\n")
	b.WriteString(`  </defs>` + "\n")

	// Build a flat node map for edge endpoint lookup (with absolute positions)
	nodeMap := make(map[string]*absoluteNode)
	for _, node := range layout.Nodes {
		collectAbsoluteNodes(node, node.X, node.Y, nodeMap)
	}

	// Render nodes
	for _, node := range layout.Nodes {
		renderNode(&b, node, node.X, node.Y, "  ")
	}

	// Render edges
	for _, edge := range layout.Edges {
		renderEdge(&b, edge, nodeMap)
	}

	b.WriteString("</svg>\n")
	return b.String()
}

type absoluteNode struct {
	X, Y, W, H float64
}

func collectAbsoluteNodes(node *LayoutNode, absX, absY float64, nodeMap map[string]*absoluteNode) {
	nodeMap[node.ID] = &absoluteNode{X: absX, Y: absY, W: node.W, H: node.H}
	for _, child := range node.Children {
		collectAbsoluteNodes(child, absX+child.X, absY+child.Y, nodeMap)
	}
}

func renderNode(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	b.WriteString(fmt.Sprintf(
		`%s<g data-id="%s" data-line="%d">`+"\n",
		indent, escapeXML(node.ID), node.Line,
	))

	// Rectangle
	b.WriteString(fmt.Sprintf(
		`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="0"/>`+"\n",
		indent, absX, absY, node.W, node.H,
		escapeXML(node.Style.Fill), escapeXML(node.Style.Stroke), node.Style.StrokeWidth,
	))

	// Label text
	if node.Label != "" {
		textX := absX + node.W/2
		var textY float64
		if len(node.Children) > 0 {
			// Parent node: label at top
			textY = absY + labelHeight/2 + node.Style.FontSize/2 - 2
		} else {
			// Leaf node: label centered
			textY = absY + node.H/2 + node.Style.FontSize/2 - 2
		}
		b.WriteString(fmt.Sprintf(
			`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			indent, textX, textY, node.Style.FontSize, escapeXML(node.Style.FontColor), escapeXML(node.Label),
		))
	}

	// Render children
	for _, child := range node.Children {
		renderNode(b, child, absX+child.X, absY+child.Y, indent+"  ")
	}

	b.WriteString(fmt.Sprintf("%s</g>\n", indent))
}

func renderEdge(b *strings.Builder, edge *LayoutEdge, nodeMap map[string]*absoluteNode) {
	from, ok := nodeMap[edge.FromID]
	if !ok {
		return
	}
	to, ok := nodeMap[edge.ToID]
	if !ok {
		return
	}

	// Centers
	fromCX := from.X + from.W/2
	fromCY := from.Y + from.H/2
	toCX := to.X + to.W/2
	toCY := to.Y + to.H/2

	// Intersection with source boundary (line from source center toward target center)
	x1, y1 := IntersectLineRect(fromCX, fromCY, toCX, toCY, from.X, from.Y, from.W, from.H)
	// Intersection with target boundary (line from target center toward source center)
	x2, y2 := IntersectLineRect(toCX, toCY, fromCX, fromCY, to.X, to.Y, to.W, to.H)

	// Apply gap
	x1, y1 = ApplyGap(x1, y1, fromCX, fromCY, defaultGap)
	x2, y2 = ApplyGap(x2, y2, toCX, toCY, defaultGap)

	b.WriteString(fmt.Sprintf(
		`  <g data-id="%s" data-line="%d">`+"\n",
		escapeXML(edge.ID), edge.Line,
	))
	b.WriteString(fmt.Sprintf(
		`    <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f" marker-end="url(#arrowhead)"/>`+"\n",
		x1, y1, x2, y2, escapeXML(edge.Style.Stroke), edge.Style.StrokeWidth,
	))

	// Edge label at midpoint
	if edge.Label != "" {
		midX := (x1 + x2) / 2
		midY := (y1+y2)/2 - 6 // slight offset above line
		b.WriteString(fmt.Sprintf(
			`    <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			midX, midY, edge.Style.FontSize, escapeXML(edge.Style.Stroke), escapeXML(edge.Label),
		))
	}

	b.WriteString("  </g>\n")
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
