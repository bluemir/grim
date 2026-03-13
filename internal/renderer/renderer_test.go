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

func TestRenderNestedEdges(t *testing.T) {
	doc, err := parser.Parse(`parent: {
  A
  B
  A -> B
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	// Nested edge should have prefixed IDs
	if !strings.Contains(svg, `data-id="parent.A--parent.B"`) {
		t.Error("expected nested edge with prefixed IDs parent.A--parent.B")
	}
	// Both child nodes should exist
	if !strings.Contains(svg, `data-id="parent.A"`) {
		t.Error("expected data-id for parent.A")
	}
	if !strings.Contains(svg, `data-id="parent.B"`) {
		t.Error("expected data-id for parent.B")
	}
}

func TestRenderVariableRowHeights(t *testing.T) {
	// A parent node with children should be taller than a simple node.
	// The second row's Y position should account for the taller first row.
	doc, err := parser.Parse(`parent: {
  child1
  child2
}
A
B
C
D
E
F
G
H
I
J
K`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	layout := BuildLayout(doc)

	// Find parent node and a node that should be in a later row
	var parentNode *LayoutNode
	for _, n := range layout.Nodes {
		if n.ID == "parent" {
			parentNode = n
		}
	}
	if parentNode == nil {
		t.Fatal("expected parent node")
	}

	// Parent should be taller than default
	if parentNode.H <= defaultNodeH {
		t.Errorf("parent height %.1f should be greater than default %.1f", parentNode.H, defaultNodeH)
	}

	// Nodes in subsequent rows should not overlap with the parent
	for _, n := range layout.Nodes {
		if n.ID == "parent" {
			continue
		}
		if n.Y > 0 && n.Y < parentNode.Y+parentNode.H && n.X < parentNode.X+parentNode.W && n.X+n.W > parentNode.X {
			// This node overlaps with parent vertically — only OK if it's in the same row (Y==0)
			if n.Y > 0 && n.Y < parentNode.H {
				t.Errorf("node %s at Y=%.1f overlaps with parent (H=%.1f)", n.ID, n.Y, parentNode.H)
			}
		}
	}
}

func TestRenderParentSeparatorLine(t *testing.T) {
	doc, err := parser.Parse(`parent: {
  child
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	svg := Render(doc)

	// Should contain a separator line at labelHeight below parent's top
	// The line should be a <line> element inside the parent's <g>
	if !strings.Contains(svg, "<line x1=") {
		t.Error("expected separator <line> element in parent node")
	}
}

func TestRenderNestedImplicitNodes(t *testing.T) {
	// Implicit nodes referenced only by edges inside a parent block
	// should be placed as children of that parent, not as top-level nodes.
	doc, err := parser.Parse(`C: {
  node1 -> node2
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	layout := BuildLayout(doc)

	// Find parent node C
	var parentNode *LayoutNode
	for _, n := range layout.Nodes {
		if n.ID == "C" {
			parentNode = n
		}
	}
	if parentNode == nil {
		t.Fatal("expected parent node C")
	}

	// node1 and node2 should be children of C, not top-level nodes
	if len(parentNode.Children) != 2 {
		t.Fatalf("expected 2 children in C, got %d", len(parentNode.Children))
	}

	childIDs := map[string]bool{}
	for _, child := range parentNode.Children {
		childIDs[child.ID] = true
	}
	if !childIDs["C.node1"] {
		t.Error("expected C.node1 as child of C")
	}
	if !childIDs["C.node2"] {
		t.Error("expected C.node2 as child of C")
	}

	// Implicit nodes should NOT appear as top-level nodes
	for _, n := range layout.Nodes {
		if n.ID == "C.node1" || n.ID == "C.node2" {
			t.Errorf("implicit node %s should not be a top-level node", n.ID)
		}
	}

	// SVG rendering should include the implicit nodes inside the parent
	svg := Render(doc)
	if !strings.Contains(svg, `data-id="C.node1"`) {
		t.Error("expected data-id for C.node1 in SVG")
	}
	if !strings.Contains(svg, `data-id="C.node2"`) {
		t.Error("expected data-id for C.node2 in SVG")
	}
}

func TestUnfixedNodesStartBelowFixedNodes(t *testing.T) {
	doc, _ := parser.Parse(`
fixed {
  @layout {x: 0, y: 200}
}
unfixed`)
	layout := BuildLayout(doc)
	var fixedNode, unfixedNode *LayoutNode
	for _, n := range layout.Nodes {
		if n.ID == "fixed" {
			fixedNode = n
		}
		if n.ID == "unfixed" {
			unfixedNode = n
		}
	}
	fixedBottom := fixedNode.Y + fixedNode.H
	if unfixedNode.Y < fixedBottom {
		t.Errorf("unfixed Y=%.1f should be >= fixed bottom %.1f", unfixedNode.Y, fixedBottom)
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
