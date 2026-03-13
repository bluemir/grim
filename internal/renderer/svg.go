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
	Shape      string
}

func collectAbsoluteNodes(node *LayoutNode, absX, absY float64, nodeMap map[string]*absoluteNode) {
	nodeMap[node.ID] = &absoluteNode{X: absX, Y: absY, W: node.W, H: node.H, Shape: node.Shape}
	for _, child := range node.Children {
		collectAbsoluteNodes(child, absX+child.X, absY+child.Y, nodeMap)
	}
}

func renderNode(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	b.WriteString(fmt.Sprintf(
		`%s<g data-id="%s" data-line="%d">`+"\n",
		indent, escapeXML(node.ID), node.Line,
	))

	// Shape
	renderShape(b, node, absX, absY, indent)

	// Label text
	if node.Label != "" {
		textX := absX + node.W/2
		var textY float64
		if len(node.Children) > 0 {
			textY = absY + labelHeight/2 + node.Style.FontSize/2 - 2
		} else {
			textY = absY + node.H/2 + node.Style.FontSize/2 - 2
		}
		b.WriteString(fmt.Sprintf(
			`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			indent, textX, textY, node.Style.FontSize, escapeXML(node.Style.FontColor), escapeXML(node.Label),
		))
	}

	// Separator line between label and children area
	if len(node.Children) > 0 {
		sepY := absY + labelHeight
		b.WriteString(fmt.Sprintf(
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent, absX, sepY, absX+node.W, sepY, escapeXML(node.Style.Stroke), node.Style.StrokeWidth,
		))
	}

	// Render children
	for _, child := range node.Children {
		renderNode(b, child, absX+child.X, absY+child.Y, indent+"  ")
	}

	b.WriteString(fmt.Sprintf("%s</g>\n", indent))
}

func renderShape(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	fill := escapeXML(node.Style.Fill)
	stroke := escapeXML(node.Style.Stroke)
	sw := node.Style.StrokeWidth
	cx := absX + node.W/2
	cy := absY + node.H/2

	switch node.Shape {
	case "circle":
		r := node.W / 2
		if node.H/2 < r {
			r = node.H / 2
		}
		b.WriteString(fmt.Sprintf(
			`%s  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent, cx, cy, node.W/2, node.H/2, fill, stroke, sw,
		))

	case "diamond":
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent,
			cx, absY, // top
			absX+node.W, cy, // right
			cx, absY+node.H, // bottom
			absX, cy, // left
			fill, stroke, sw,
		))

	case "hexagon":
		// Flat-top hexagon
		inset := node.W * 0.25
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent,
			absX+inset, absY, // top-left
			absX+node.W-inset, absY, // top-right
			absX+node.W, cy, // right
			absX+node.W-inset, absY+node.H, // bottom-right
			absX+inset, absY+node.H, // bottom-left
			absX, cy, // left
			fill, stroke, sw,
		))

	case "parallelogram":
		skew := node.W * 0.2
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent,
			absX+skew, absY, // top-left
			absX+node.W, absY, // top-right
			absX+node.W-skew, absY+node.H, // bottom-right
			absX, absY+node.H, // bottom-left
			fill, stroke, sw,
		))

	case "cylinder":
		ry := 8.0 // ellipse radius for top/bottom caps
		bodyY := absY + ry
		bodyH := node.H - 2*ry
		// Body rectangle
		b.WriteString(fmt.Sprintf(
			`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="none"/>`+"\n",
			indent, absX, bodyY, node.W, bodyH, fill,
		))
		// Left and right edges of the body
		b.WriteString(fmt.Sprintf(
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent, absX, bodyY, absX, bodyY+bodyH, stroke, sw,
		))
		b.WriteString(fmt.Sprintf(
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent, absX+node.W, bodyY, absX+node.W, bodyY+bodyH, stroke, sw,
		))
		// Top ellipse (full)
		b.WriteString(fmt.Sprintf(
			`%s  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent, cx, bodyY, node.W/2, ry, fill, stroke, sw,
		))
		// Bottom ellipse (half, bottom arc only)
		b.WriteString(fmt.Sprintf(
			`%s  <path d="M%.1f,%.1f A%.1f,%.1f 0 0,0 %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent,
			absX, bodyY+bodyH,
			node.W/2, ry,
			absX+node.W, bodyY+bodyH,
			fill, stroke, sw,
		))

	case "cloud":
		// Cloud shape using cubic bezier curves
		w := node.W
		h := node.H
		b.WriteString(fmt.Sprintf(
			`%s  <path d="`+
				`M%.1f,%.1f `+
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // top-left bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // top bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // top-right bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // right bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // bottom-right bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // bottom bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f `+ // bottom-left bump
				`C%.1f,%.1f %.1f,%.1f %.1f,%.1f`+ // left bump
				`Z" fill="%s" stroke="%s" stroke-width="%.0f"/>`+"\n",
			indent,
			absX+w*0.15, cy, // start left-middle
			absX-w*0.05, cy-h*0.3, absX+w*0.1, absY-h*0.1, absX+w*0.3, absY+h*0.1, // top-left
			absX+w*0.25, absY-h*0.15, absX+w*0.55, absY-h*0.15, absX+w*0.65, absY+h*0.1, // top
			absX+w*0.75, absY-h*0.05, absX+w*1.1, absY+h*0.15, absX+w*0.9, cy, // top-right
			absX+w*1.1, cy+h*0.1, absX+w*1.05, absY+h*0.85, absX+w*0.75, absY+h*0.85, // right
			absX+w*0.85, absY+h*1.05, absX+w*0.65, absY+h*1.1, absX+w*0.5, absY+h*0.9, // bottom-right
			absX+w*0.45, absY+h*1.1, absX+w*0.2, absY+h*1.05, absX+w*0.25, absY+h*0.85, // bottom
			absX+w*0.1, absY+h*0.95, absX-w*0.05, absY+h*0.8, absX+w*0.1, cy+h*0.1, // bottom-left
			absX-w*0.05, cy+h*0.05, absX-w*0.05, cy-h*0.1, absX+w*0.15, cy, // left
			fill, stroke, sw,
		))

	case "rounded":
		rx := 8.0
		b.WriteString(fmt.Sprintf(
			`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="%.0f"/>`+"\n",
			indent, absX, absY, node.W, node.H, fill, stroke, sw, rx,
		))

	default: // "rectangle" or unspecified
		b.WriteString(fmt.Sprintf(
			`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="0"/>`+"\n",
			indent, absX, absY, node.W, node.H, fill, stroke, sw,
		))
	}
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
	x1, y1 := IntersectLineShape(fromCX, fromCY, toCX, toCY, from.X, from.Y, from.W, from.H, from.Shape)
	// Intersection with target boundary (line from target center toward source center)
	x2, y2 := IntersectLineShape(toCX, toCY, fromCX, fromCY, to.X, to.Y, to.W, to.H, to.Shape)

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
