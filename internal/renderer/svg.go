package renderer

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
)

const defaultGap = 4.0

// RenderSVG generates an SVG string from a LayoutResult.
func RenderSVG(layout *LayoutResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n",
		layout.Width, layout.Height, layout.Width, layout.Height,
	))

	// Collect unique edge stroke colors for marker generation
	edgeColors := collectEdgeColors(layout.Edges)

	// Defs: arrowhead markers (one pair per unique edge stroke color)
	b.WriteString(`  <defs>` + "\n")
	for color := range edgeColors {
		safeColor := escapeXML(color)
		fwdID := markerID(color, false)
		revID := markerID(color, true)
		b.WriteString(fmt.Sprintf(
			`    <marker id="%s" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">`+"\n"+
				`      <polygon points="0 0, 10 3.5, 0 7" fill="%s"/>`+"\n"+
				`    </marker>`+"\n",
			fwdID, safeColor,
		))
		b.WriteString(fmt.Sprintf(
			`    <marker id="%s" markerWidth="10" markerHeight="7" refX="0" refY="3.5" orient="auto">`+"\n"+
				`      <polygon points="10 0, 0 3.5, 10 7" fill="%s"/>`+"\n"+
				`    </marker>`+"\n",
			revID, safeColor,
		))
	}
	b.WriteString(`  </defs>` + "\n")

	// Build a flat node map for edge endpoint lookup (with absolute positions)
	nodeMap := make(map[string]*absoluteNode)
	for _, node := range layout.Nodes {
		collectAbsoluteNodes(node, node.X, node.Y, nodeMap)
	}

	edgesByAncestor := groupEdgesByAncestor(layout.Edges)

	// Render nodes
	for _, node := range layout.Nodes {
		renderNode(&b, node, node.X, node.Y, node.X, node.Y, edgesByAncestor, nodeMap, "  ")
	}

	// Render root-level edges
	for _, edge := range edgesByAncestor[""] {
		renderEdge(&b, edge, nodeMap, 0, 0, "  ")
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

// commonContainerID returns the ID of the node whose <g> should contain the edge.
// "" means the edge belongs at SVG root level.
func commonContainerID(fromID, toID string) string {
	partsA := strings.Split(fromID, ".")
	partsB := strings.Split(toID, ".")
	var common []string
	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		if partsA[i] != partsB[i] {
			break
		}
		common = append(common, partsA[i])
	}
	commonStr := strings.Join(common, ".")
	if fromID == commonStr || toID == commonStr {
		if len(common) == 0 {
			return ""
		}
		common = common[:len(common)-1]
	}
	return strings.Join(common, ".")
}

func groupEdgesByAncestor(edges []*LayoutEdge) map[string][]*LayoutEdge {
	result := make(map[string][]*LayoutEdge)
	for _, edge := range edges {
		ancestor := commonContainerID(edge.FromID, edge.ToID)
		result[ancestor] = append(result[ancestor], edge)
	}
	return result
}

func renderNode(b *strings.Builder, node *LayoutNode, absX, absY, localX, localY float64, edgesByAncestor map[string][]*LayoutEdge, nodeMap map[string]*absoluteNode, indent string) {
	extraAttrs := ""
	if node.Style.Opacity != 1.0 && node.Style.Opacity != 0 {
		extraAttrs += fmt.Sprintf(` opacity="%.2f"`, node.Style.Opacity)
	}
	if node.Style.Shadow > 0 {
		extraAttrs += fmt.Sprintf(` filter="drop-shadow(2px 2px %.0fpx rgba(0,0,0,0.4))"`, node.Style.Shadow)
	}
	b.WriteString(fmt.Sprintf(
		`%s<g data-id="%s" data-line="%d" data-x="%.1f" data-y="%.1f" data-local-x="%.1f" data-local-y="%.1f" data-w="%.1f" data-h="%.1f" transform="translate(%.1f,%.1f)"%s>`+"\n",
		indent, escapeXML(node.ID), node.Line, absX, absY, localX, localY, node.W, node.H, localX, localY, extraAttrs,
	))

	// Stack shadow layers — drawn before main shape so main shape renders on top
	if node.Stack > 1 {
		const stackDelta = 4.0
		b.WriteString(fmt.Sprintf("%s  <g class=\"node-shadows\">\n", indent))
		for i := node.Stack - 1; i >= 1; i-- {
			dx := float64(i) * stackDelta
			dy := float64(i) * stackDelta
			renderShape(b, node, dx, dy, indent+"  ")
		}
		b.WriteString(fmt.Sprintf("%s  </g>\n", indent))
	}

	// Shape
	renderShape(b, node, 0, 0, indent)

	// Icon overlay
	if hasIcon(node) {
		renderIcon(b, node, 0, 0, indent)
	}

	// Label text
	if node.Label != "" {
		if node.TextFormat == "markdown" {
			renderMarkdownLabel(b, node, 0, 0, indent)
		} else {
			renderPlainTextLabel(b, node, 0, 0, indent)
		}
	}

	// Separator line between label and children area
	if len(node.Children) > 0 {
		sepY := labelHeight
		b.WriteString(fmt.Sprintf(
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent, 0.0, sepY, node.W, sepY, escapeXML(node.Style.Stroke), node.Style.StrokeWidth, dashAttr(node.Style.StrokeDash),
		))
	}

	// Render children
	for _, child := range node.Children {
		renderNode(b, child, absX+child.X, absY+child.Y, child.X, child.Y, edgesByAncestor, nodeMap, indent+"  ")
	}

	// Render edges whose common ancestor is this node
	for _, edge := range edgesByAncestor[node.ID] {
		renderEdge(b, edge, nodeMap, absX, absY, indent+"  ")
	}

	b.WriteString(fmt.Sprintf("%s</g>\n", indent))
}

func renderMarkdownLabel(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	var htmlBuf bytes.Buffer
	if err := mdRenderer.Convert([]byte(node.Label), &htmlBuf); err != nil {
		// Fallback to plain text on error
		b.WriteString(fmt.Sprintf(
			`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			indent, absX+node.W/2, absY+node.H/2+node.Style.FontSize/2-2,
			node.Style.FontSize, escapeXML(node.Style.FontColor), escapeXML(node.Label),
		))
		return
	}

	var foX, foY, foW, foH float64
	switch {
	case hasIcon(node):
		sz := node.IconSize
		switch node.IconPosition {
		case "bottom":
			foX, foY, foW, foH = absX+4, absY+4, node.W-8, node.H-sz-iconGap-4
		case "left":
			foX, foY, foW, foH = absX+sz+iconGap, absY+4, node.W-sz-iconGap-4, node.H-8
		case "right":
			foX, foY, foW, foH = absX+4, absY+4, node.W-sz-iconGap-4, node.H-8
		default: // "top"
			foX, foY, foW, foH = absX+4, absY+sz+iconGap, node.W-8, node.H-sz-iconGap-4
		}
	case len(node.Children) > 0:
		foX, foY, foW, foH = absX+4, absY+2, node.W-8, labelHeight-4
	default:
		foX, foY, foW, foH = absX+4, absY+4, node.W-8, node.H-8
	}

	b.WriteString(fmt.Sprintf(
		`%s  <foreignObject x="%.1f" y="%.1f" width="%.1f" height="%.1f">`+"\n",
		indent, foX, foY, foW, foH,
	))
	b.WriteString(fmt.Sprintf(
		`%s    <div xmlns="http://www.w3.org/1999/xhtml" style="font-size:%.0fpx;color:%s;overflow:hidden;padding:2px;box-sizing:border-box;width:100%%;height:100%%">`+"\n",
		indent, node.Style.FontSize, node.Style.FontColor,
	))
	b.WriteString(`<style>*{margin:0;padding:0;box-sizing:border-box}` +
		`h1,h2,h3,h4,h5,h6{font-size:inherit;font-weight:bold}` +
		`p,li{line-height:1.4}` +
		`ul,ol{padding-left:1.5em}</style>` + "\n")
	b.WriteString(htmlBuf.String())
	b.WriteString(fmt.Sprintf("%s    </div>\n", indent))
	b.WriteString(fmt.Sprintf("%s  </foreignObject>\n", indent))
}

func renderPlainTextLabel(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	textX := absX + node.W/2
	var centerY float64
	switch {
	case hasIcon(node):
		sz := node.IconSize
		switch node.IconPosition {
		case "bottom":
			labelAreaH := node.H - sz - iconGap
			centerY = absY + labelAreaH/2 + node.Style.FontSize/2 - 2
		case "left":
			textX = absX + (node.W+sz+iconGap)/2
			centerY = absY + node.H/2 + node.Style.FontSize/2 - 2
		case "right":
			textX = absX + (node.W-sz-iconGap)/2
			centerY = absY + node.H/2 + node.Style.FontSize/2 - 2
		default: // "top"
			iconAreaH := sz + iconGap
			labelAreaH := node.H - iconAreaH
			centerY = absY + iconAreaH + labelAreaH/2 + node.Style.FontSize/2 - 2
		}
	case len(node.Children) > 0:
		centerY = absY + labelHeight/2 + node.Style.FontSize/2 - 2
	default:
		centerY = absY + node.H/2 + node.Style.FontSize/2 - 2
	}

	lines := strings.Split(node.Label, "\n")
	if len(lines) == 1 {
		b.WriteString(fmt.Sprintf(
			`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			indent, textX, centerY, node.Style.FontSize, escapeXML(node.Style.FontColor), escapeXML(node.Label),
		))
		return
	}

	// Multiline: shift first line up to center the block vertically
	firstY := centerY - float64(len(lines)-1)*lineHeight/2
	b.WriteString(fmt.Sprintf(
		`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">`+"\n",
		indent, textX, firstY, node.Style.FontSize, escapeXML(node.Style.FontColor),
	))
	for i, line := range lines {
		dy := "0"
		if i > 0 {
			dy = fmt.Sprintf("%.0f", lineHeight)
		}
		b.WriteString(fmt.Sprintf(
			`%s    <tspan x="%.1f" dy="%s">%s</tspan>`+"\n",
			indent, textX, dy, escapeXML(line),
		))
	}
	b.WriteString(fmt.Sprintf("%s  </text>\n", indent))
}

func dashAttr(s string) string {
	if s == "" {
		return ""
	}
	return fmt.Sprintf(` stroke-dasharray="%s"`, s)
}

func renderShape(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	fill := escapeXML(node.Style.Fill)
	stroke := escapeXML(node.Style.Stroke)
	sw := node.Style.StrokeWidth
	dash := dashAttr(node.Style.StrokeDash)
	cx := absX + node.W/2
	cy := absY + node.H/2

	switch node.Shape {
	case "circle":
		r := node.W / 2
		if node.H/2 < r {
			r = node.H / 2
		}
		b.WriteString(fmt.Sprintf(
			`%s  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent, cx, cy, node.W/2, node.H/2, fill, stroke, sw, dash,
		))

	case "diamond":
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent,
			cx, absY, // top
			absX+node.W, cy, // right
			cx, absY+node.H, // bottom
			absX, cy, // left
			fill, stroke, sw, dash,
		))

	case "hexagon":
		// Flat-top hexagon
		inset := node.W * 0.25
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent,
			absX+inset, absY, // top-left
			absX+node.W-inset, absY, // top-right
			absX+node.W, cy, // right
			absX+node.W-inset, absY+node.H, // bottom-right
			absX+inset, absY+node.H, // bottom-left
			absX, cy, // left
			fill, stroke, sw, dash,
		))

	case "parallelogram":
		skew := node.W * 0.2
		b.WriteString(fmt.Sprintf(
			`%s  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent,
			absX+skew, absY, // top-left
			absX+node.W, absY, // top-right
			absX+node.W-skew, absY+node.H, // bottom-right
			absX, absY+node.H, // bottom-left
			fill, stroke, sw, dash,
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
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent, absX, bodyY, absX, bodyY+bodyH, stroke, sw, dash,
		))
		b.WriteString(fmt.Sprintf(
			`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent, absX+node.W, bodyY, absX+node.W, bodyY+bodyH, stroke, sw, dash,
		))
		// Top ellipse (full)
		b.WriteString(fmt.Sprintf(
			`%s  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent, cx, bodyY, node.W/2, ry, fill, stroke, sw, dash,
		))
		// Bottom ellipse (half, bottom arc only)
		b.WriteString(fmt.Sprintf(
			`%s  <path d="M%.1f,%.1f A%.1f,%.1f 0 0,0 %.1f,%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
			indent,
			absX, bodyY+bodyH,
			node.W/2, ry,
			absX+node.W, bodyY+bodyH,
			fill, stroke, sw, dash,
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
				`Z" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
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
			fill, stroke, sw, dash,
		))

	case "rounded":
		rx := 8.0
		if node.Style.BorderRadius >= 0 {
			rx = node.Style.BorderRadius
		}
		b.WriteString(fmt.Sprintf(
			`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="%.1f"%s/>`+"\n",
			indent, absX, absY, node.W, node.H, fill, stroke, sw, rx, dash,
		))

	default: // "rectangle" or unspecified
		rx := 0.0
		if node.Style.BorderRadius >= 0 {
			rx = node.Style.BorderRadius
		}
		b.WriteString(fmt.Sprintf(
			`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="%.1f"%s/>`+"\n",
			indent, absX, absY, node.W, node.H, fill, stroke, sw, rx, dash,
		))
	}
}

func renderIcon(b *strings.Builder, node *LayoutNode, absX, absY float64, indent string) {
	sz := node.IconSize
	var ix, iy float64
	switch node.IconPosition {
	case "bottom":
		ix = absX + (node.W-sz)/2
		iy = absY + node.H - sz - iconGap/2
	case "left":
		ix = absX + iconGap/2
		iy = absY + (node.H-sz)/2
	case "right":
		ix = absX + node.W - sz - iconGap/2
		iy = absY + (node.H-sz)/2
	default: // "top"
		ix = absX + (node.W-sz)/2
		iy = absY + iconGap/2
	}

	switch node.IconKind {
	case "user":
		renderUserIcon(b, ix, iy, sz, node.Style, indent)
	case "bot":
		renderBotIcon(b, ix, iy, sz, node.Style, indent)
	default:
		// URL-based <image> rendering
		b.WriteString(fmt.Sprintf(
			`%s  <image x="%.1f" y="%.1f" width="%.1f" height="%.1f" href="%s" preserveAspectRatio="xMidYMid meet"/>`+"\n",
			indent, ix, iy, sz, sz, escapeXML(node.IconURL),
		))
	}
}

// iconStroke returns stroke color and width for built-in icons,
// falling back to defaults when the node style suppresses them.
func iconStroke(style NodeStyle) (string, float64) {
	stroke := style.Stroke
	sw := style.StrokeWidth
	if stroke == "" || stroke == "none" {
		stroke = defaultNodeStyle.Stroke
	}
	if sw <= 0 {
		sw = defaultNodeStyle.StrokeWidth
	}
	return escapeXML(stroke), sw
}

// renderUserIcon draws a person silhouette (head + shoulders) within the given box.
func renderUserIcon(b *strings.Builder, ix, iy, sz float64, style NodeStyle, indent string) {
	fill := escapeXML(style.Fill)
	stroke, sw := iconStroke(style)
	dash := dashAttr(style.StrokeDash)
	cx := ix + sz/2

	headR := sz * 0.22
	headCY := iy + sz*0.27
	neckY := headCY + headR
	neckW := headR * 0.8
	bodyCtrlY := neckY + (iy+sz-neckY)*0.35
	// Body drawn first so head circle renders on top, covering the neck seam
	b.WriteString(fmt.Sprintf(
		`%s  <path d="M%.1f,%.1f Q%.1f,%.1f %.1f,%.1f L%.1f,%.1f Q%.1f,%.1f %.1f,%.1f Z" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
		indent,
		cx-neckW, neckY, // neck-left (start)
		ix+sz*0.05, bodyCtrlY, ix+sz*0.05, iy+sz, // left shoulder curve → bottom-left
		ix+sz*0.95, iy+sz, // bottom-right
		ix+sz*0.95, bodyCtrlY, cx+neckW, neckY, // right shoulder curve → neck-right
		fill, stroke, sw, dash,
	))
	// Head circle (on top)
	b.WriteString(fmt.Sprintf(
		`%s  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
		indent, cx, headCY, headR, fill, stroke, sw, dash,
	))
}

// renderBotIcon draws a robot icon (antenna + face + eyes) within the given box.
func renderBotIcon(b *strings.Builder, ix, iy, sz float64, style NodeStyle, indent string) {
	fill := escapeXML(style.Fill)
	stroke, sw := iconStroke(style)
	dash := dashAttr(style.StrokeDash)
	cx := ix + sz/2

	antennaH := sz * 0.15
	faceH := sz - antennaH
	faceTop := iy + antennaH
	ballR := sz * 0.05
	ballCY := iy + ballR
	eyeR := sz * 0.07
	eyeY := faceTop + faceH*0.38
	// Antenna line
	b.WriteString(fmt.Sprintf(
		`%s  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
		indent, cx, faceTop, cx, ballCY+ballR, stroke, sw, dash,
	))
	// Antenna ball
	b.WriteString(fmt.Sprintf(
		`%s  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="%.0f"%s/>`+"\n",
		indent, cx, ballCY, ballR, stroke, stroke, sw, dash,
	))
	// Face (rounded rectangle)
	b.WriteString(fmt.Sprintf(
		`%s  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.0f" rx="8"%s/>`+"\n",
		indent, ix, faceTop, sz, faceH, fill, stroke, sw, dash,
	))
	// Left eye
	b.WriteString(fmt.Sprintf(
		`%s  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="none"/>`+"\n",
		indent, cx-sz*0.2, eyeY, eyeR, stroke,
	))
	// Right eye
	b.WriteString(fmt.Sprintf(
		`%s  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="none"/>`+"\n",
		indent, cx+sz*0.2, eyeY, eyeR, stroke,
	))
}

func renderEdge(b *strings.Builder, edge *LayoutEdge, nodeMap map[string]*absoluteNode, ancestorAbsX, ancestorAbsY float64, indent string) {
	from, ok := nodeMap[edge.FromID]
	if !ok {
		return
	}
	to, ok := nodeMap[edge.ToID]
	if !ok {
		return
	}

	// Local positions (relative to ancestor's coordinate origin)
	fromLocalX := from.X - ancestorAbsX
	fromLocalY := from.Y - ancestorAbsY
	toLocalX := to.X - ancestorAbsX
	toLocalY := to.Y - ancestorAbsY

	// Centers in local coords
	fromCX := fromLocalX + from.W/2
	fromCY := fromLocalY + from.H/2
	toCX := toLocalX + to.W/2
	toCY := toLocalY + to.H/2

	// Direction for intersection: use first/last waypoint if available
	// Waypoints are stored in ancestor-local coords
	srcDirX, srcDirY := toCX, toCY
	dstDirX, dstDirY := fromCX, fromCY
	if len(edge.Waypoints) > 0 {
		srcDirX = edge.Waypoints[0].X
		srcDirY = edge.Waypoints[0].Y
		last := edge.Waypoints[len(edge.Waypoints)-1]
		dstDirX = last.X
		dstDirY = last.Y
	}

	// Anchor overrides: replace direction vectors with fixed-angle directions.
	const anchorDirLen = 10000.0
	var fromAnchorCenter, toAnchorCenter bool
	if edge.FromAnchor != "" && edge.FromAnchor != "-" {
		angle, isCenter := ParseAnchorAngle(edge.FromAnchor)
		if isCenter {
			fromAnchorCenter = true
		} else {
			srcDirX = fromCX + math.Sin(angle)*anchorDirLen
			srcDirY = fromCY - math.Cos(angle)*anchorDirLen
		}
	}
	if edge.ToAnchor != "" && edge.ToAnchor != "-" {
		angle, isCenter := ParseAnchorAngle(edge.ToAnchor)
		if isCenter {
			toAnchorCenter = true
		} else {
			dstDirX = toCX + math.Sin(angle)*anchorDirLen
			dstDirY = toCY - math.Cos(angle)*anchorDirLen
		}
	}

	// Intersection with source boundary (or center if anchor == "center")
	var x1, y1 float64
	if fromAnchorCenter {
		x1, y1 = fromCX, fromCY
	} else {
		x1, y1 = IntersectLineShape(fromCX, fromCY, srcDirX, srcDirY, fromLocalX, fromLocalY, from.W, from.H, from.Shape)
		x1, y1 = ApplyGap(x1, y1, fromCX, fromCY, defaultGap)
	}
	// Intersection with target boundary (or center if anchor == "center")
	var x2, y2 float64
	if toAnchorCenter {
		x2, y2 = toCX, toCY
	} else {
		x2, y2 = IntersectLineShape(toCX, toCY, dstDirX, dstDirY, toLocalX, toLocalY, to.W, to.H, to.Shape)
		x2, y2 = ApplyGap(x2, y2, toCX, toCY, defaultGap)
	}

	// Build points list: start, waypoints, end
	type pt struct{ x, y float64 }
	pts := []pt{{x1, y1}}
	for _, wp := range edge.Waypoints {
		pts = append(pts, pt{wp.X, wp.Y})
	}
	pts = append(pts, pt{x2, y2})

	// Build SVG points attribute string
	var ptsBuf strings.Builder
	for i, p := range pts {
		if i > 0 {
			ptsBuf.WriteString(" ")
		}
		ptsBuf.WriteString(fmt.Sprintf("%.1f,%.1f", p.x, p.y))
	}
	pointsStr := ptsBuf.String()

	fwdMarker := markerID(edge.Style.Stroke, false)
	revMarker := markerID(edge.Style.Stroke, true)

	var markerStart, markerEnd string
	switch edge.Direction {
	case "none":
		// no arrows
	case "reverse":
		markerStart = fmt.Sprintf(` marker-start="url(#%s)"`, revMarker)
	case "bidirectional":
		markerStart = fmt.Sprintf(` marker-start="url(#%s)"`, revMarker)
		markerEnd = fmt.Sprintf(` marker-end="url(#%s)"`, fwdMarker)
	default: // "" or "forward"
		markerEnd = fmt.Sprintf(` marker-end="url(#%s)"`, fwdMarker)
	}

	b.WriteString(fmt.Sprintf(
		`%s<g data-id="%s" data-line="%d" data-type="edge" data-waypoints='%s' data-from-anchor='%s' data-to-anchor='%s' data-from-id='%s' data-to-id='%s' data-ancestor-x="%.1f" data-ancestor-y="%.1f">`+"\n",
		indent, escapeXML(edge.ID), edge.Line, waypointsAttr(edge.Waypoints),
		edge.FromAnchor, edge.ToAnchor, escapeXML(edge.FromID), escapeXML(edge.ToID),
		ancestorAbsX, ancestorAbsY,
	))

	// Invisible hit-target polyline for easy click detection
	b.WriteString(fmt.Sprintf(
		`%s  <polyline points="%s" fill="none" stroke="transparent" stroke-width="12" pointer-events="stroke"/>`+"\n",
		indent, pointsStr,
	))

	// Visible polyline
	edgeDash := dashAttr(edge.Style.StrokeDash)
	b.WriteString(fmt.Sprintf(
		`%s  <polyline points="%s" fill="none" stroke="%s" stroke-width="%.0f"%s%s%s/>`+"\n",
		indent, pointsStr, escapeXML(edge.Style.Stroke), edge.Style.StrokeWidth, edgeDash, markerStart, markerEnd,
	))

	// Edge label at midpoint of points list
	if edge.Label != "" {
		midIdx := len(pts) / 2
		var midX, midY float64
		if len(pts)%2 == 0 {
			midX = (pts[midIdx-1].x + pts[midIdx].x) / 2
			midY = (pts[midIdx-1].y + pts[midIdx].y) / 2
		} else {
			midX = pts[midIdx].x
			midY = pts[midIdx].y
		}
		midY -= 6 // slight offset above line
		b.WriteString(fmt.Sprintf(
			`%s  <text x="%.1f" y="%.1f" text-anchor="middle" font-size="%.0f" fill="%s">%s</text>`+"\n",
			indent, midX, midY, edge.Style.FontSize, escapeXML(edge.Style.Stroke), escapeXML(edge.Label),
		))
	}

	b.WriteString(fmt.Sprintf("%s</g>\n", indent))
}

func waypointsAttr(wps []Point) string {
	if len(wps) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, wp := range wps {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`{"x":%.1f,"y":%.1f}`, wp.X, wp.Y))
	}
	sb.WriteString("]")
	return sb.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// sanitizeColorID converts a CSS color string into a safe XML ID fragment.
// e.g. "#333333" -> "333333", "green" -> "green", "rgb(1,2,3)" -> "rgb-1-2-3"
func sanitizeColorID(color string) string {
	color = strings.TrimPrefix(color, "#")
	var b strings.Builder
	for _, c := range color {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// markerID returns the marker element ID for the given color and direction.
func markerID(color string, start bool) string {
	suffix := sanitizeColorID(color)
	if start {
		return "arrowhead-start-" + suffix
	}
	return "arrowhead-" + suffix
}

// collectEdgeColors returns the set of unique stroke colors used by edges.
func collectEdgeColors(edges []*LayoutEdge) map[string]bool {
	colors := make(map[string]bool)
	for _, e := range edges {
		colors[e.Style.Stroke] = true
	}
	return colors
}
