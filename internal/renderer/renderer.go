package renderer

import "github.com/bluemir/grim/internal/parser"

// Render takes a parsed Document and returns an SVG string.
func Render(doc *parser.Document) string {
	layout := BuildLayout(doc)
	return RenderSVG(layout)
}
