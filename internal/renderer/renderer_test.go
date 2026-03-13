package renderer

import (
	"strings"
	"testing"

	"github.com/bluemir/grim/internal/parser"
)

func TestRenderEmptyDocument(t *testing.T) {
	doc := &parser.Document{}
	svg := Render(doc)

	if !strings.Contains(svg, "<svg") {
		t.Error("expected SVG root element")
	}
	if !strings.Contains(svg, "</svg>") {
		t.Error("expected SVG closing tag")
	}
}

func TestRenderSingleNode(t *testing.T) {
	doc, err := parser.Parse("mynode")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-id="mynode"`) {
		t.Error("expected data-id for mynode")
	}
	if !strings.Contains(svg, ">mynode</text>") {
		t.Error("expected label text for mynode")
	}
}

func TestRenderNodeWithLabel(t *testing.T) {
	doc, err := parser.Parse(`mynode: "Hello World"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-id="mynode"`) {
		t.Error("expected data-id for mynode")
	}
	if !strings.Contains(svg, ">Hello World</text>") {
		t.Error("expected label text 'Hello World'")
	}
}

func TestRenderEdge(t *testing.T) {
	doc, err := parser.Parse("A -> B")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	// Both nodes should be rendered
	if !strings.Contains(svg, `data-id="A"`) {
		t.Error("expected data-id for A")
	}
	if !strings.Contains(svg, `data-id="B"`) {
		t.Error("expected data-id for B")
	}
	// Edge should be rendered
	if !strings.Contains(svg, `data-id="A--B"`) {
		t.Error("expected data-id for edge A--B")
	}
	if !strings.Contains(svg, `marker-end="url(#arrowhead)"`) {
		t.Error("expected arrowhead marker on edge")
	}
}

func TestRenderEdgeWithLabel(t *testing.T) {
	doc, err := parser.Parse(`A -> B: "API Call"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, ">API Call</text>") {
		t.Error("expected edge label 'API Call'")
	}
}

func TestRenderNestedNodes(t *testing.T) {
	doc, err := parser.Parse(`parent: {
  child
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-id="parent"`) {
		t.Error("expected data-id for parent")
	}
	if !strings.Contains(svg, `data-id="parent.child"`) {
		t.Error("expected data-id for parent.child")
	}
}

func TestRenderFixedLayout(t *testing.T) {
	doc, err := parser.Parse(`mynode: {
  @layout {x: 100, y: 200}
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-id="mynode"`) {
		t.Error("expected data-id for mynode")
	}
	// Fixed node should be at x=100, y=200
	if !strings.Contains(svg, `x="100.0"`) {
		t.Error("expected x=100.0 for fixed layout")
	}
	if !strings.Contains(svg, `y="200.0"`) {
		t.Error("expected y=200.0 for fixed layout")
	}
}

func TestRenderDataLineAttribute(t *testing.T) {
	doc, err := parser.Parse("nodeA\nnodeB")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-line="1"`) {
		t.Error("expected data-line=1 for first node")
	}
	if !strings.Contains(svg, `data-line="2"`) {
		t.Error("expected data-line=2 for second node")
	}
}

func TestRenderMultipleUnfixedNodes(t *testing.T) {
	doc, err := parser.Parse("A\nB\nC")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	// All three should appear
	for _, id := range []string{"A", "B", "C"} {
		if !strings.Contains(svg, `data-id="`+id+`"`) {
			t.Errorf("expected data-id for %s", id)
		}
	}
}

func TestRenderWithStyle(t *testing.T) {
	doc, err := parser.Parse(`mynode: {
  @style {fill: blue, stroke: red}
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `fill="blue"`) {
		t.Error("expected fill=blue from @style")
	}
	if !strings.Contains(svg, `stroke="red"`) {
		t.Error("expected stroke=red from @style")
	}
}

func TestRenderImplicitNodes(t *testing.T) {
	// Edge references nodes that aren't explicitly declared
	doc, err := parser.Parse("X -> Y")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, `data-id="X"`) {
		t.Error("expected implicit node X")
	}
	if !strings.Contains(svg, `data-id="Y"`) {
		t.Error("expected implicit node Y")
	}
}

func TestRenderSVGHasViewBox(t *testing.T) {
	doc, err := parser.Parse("A")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	if !strings.Contains(svg, "viewBox=") {
		t.Error("expected viewBox attribute on svg")
	}
}
