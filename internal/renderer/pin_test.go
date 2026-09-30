package renderer

import (
	"strings"
	"testing"

	"github.com/bluemir/grim/internal/parser"
)

type pos struct{ X, Y float64 }

// absolutePositions returns every node's drawn position, keyed by ID.
func absolutePositions(t *testing.T, src string) map[string]pos {
	t.Helper()
	doc, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v\n%s", err, src)
	}
	out := map[string]pos{}
	var walk func(nodes []*LayoutNode, offX, offY float64)
	walk = func(nodes []*LayoutNode, offX, offY float64) {
		for _, n := range nodes {
			out[n.ID] = pos{offX + n.X, offY + n.Y}
			walk(n.Children, offX+n.X, offY+n.Y)
		}
	}
	walk(BuildLayout(doc).Nodes, 0, 0)
	return out
}

// moveWithPinning does what the editor does on drag end: pin the siblings,
// then write the moved node's new position.
func moveWithPinning(t *testing.T, src, id string, x, y int64) string {
	t.Helper()
	doc, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	PinSiblings(doc, id)
	parser.SetLayout(doc, id, x, y)
	return parser.Format(doc)
}

// assertUnmoved checks that every node except the moved one (and its
// descendants, which follow it) is drawn where it was before.
func assertUnmoved(t *testing.T, before, after map[string]pos, moved string) {
	t.Helper()
	for id, p := range before {
		if id == moved || strings.HasPrefix(id, moved+".") {
			continue
		}
		if got, ok := after[id]; !ok {
			t.Errorf("%s disappeared", id)
		} else if got != p {
			t.Errorf("%s moved from %v to %v", id, p, got)
		}
	}
}

func TestPinSiblingsTopLevel(t *testing.T) {
	src := "a\nb\nc\nd: \"a longer label here\"\ne\n"
	before := absolutePositions(t, src)

	out := moveWithPinning(t, src, "b", 400, 300)
	after := absolutePositions(t, out)

	assertUnmoved(t, before, after, "b")
	if after["b"] != (pos{400, 300}) {
		t.Errorf("b should be at the dropped position, got %v", after["b"])
	}
}

func TestPinSiblingsWithoutPinningGridReflows(t *testing.T) {
	// Guard for the test above: without pinning, the others really do move.
	src := "a\nb\nc\n"
	before := absolutePositions(t, src)

	doc, _ := parser.Parse(src)
	parser.SetLayout(doc, "b", 400, 300)
	after := absolutePositions(t, parser.Format(doc))

	if after["a"] == before["a"] && after["c"] == before["c"] {
		t.Fatal("expected unpinned siblings to reflow; the pinning test would prove nothing")
	}
}

func TestPinSiblingsNested(t *testing.T) {
	src := `outer {
    x
    y
    z
}
top
`
	before := absolutePositions(t, src)

	out := moveWithPinning(t, src, "outer.y", 40, 200)
	after := absolutePositions(t, out)

	assertUnmoved(t, before, after, "outer.y")
	// Only the moved node's level is pinned: the top-level "top" stays automatic.
	if strings.Contains(sectionOf(out, "top"), "@layout") {
		t.Errorf("top-level node should not be pinned when a nested node moves:\n%s", out)
	}
}

func TestPinSiblingsLeavesFixedAndDeeperNodesAlone(t *testing.T) {
	src := `a {
    @layout {x: 500, y: 10}
}
b {
    inner
}
c
`
	out := moveWithPinning(t, src, "c", 300, 300)

	doc, err := parser.Parse(out)
	if err != nil {
		t.Fatalf("parse error: %v\n%s", err, out)
	}
	layout := BuildLayout(doc)
	a := findLayoutNode(layout.Nodes, "a")
	if a.X != 500 || a.Y != 10 {
		t.Errorf("an already fixed sibling must keep its own @layout, got (%v, %v)", a.X, a.Y)
	}
	if inner := findLayoutNode(layout.Nodes, "b.inner"); inner.Fixed {
		t.Errorf("children of a sibling are another level and must not be pinned:\n%s", out)
	}
	if b := findLayoutNode(layout.Nodes, "b"); !b.Fixed {
		t.Errorf("b is a sibling of c and should be pinned:\n%s", out)
	}
}

func TestPinSiblingsImplicitNodes(t *testing.T) {
	// Nodes that only appear in edges become explicit declarations when pinned.
	src := "a -> b\nb -> c\n"
	before := absolutePositions(t, src)

	out := moveWithPinning(t, src, "a", 500, 300)
	after := absolutePositions(t, out)

	assertUnmoved(t, before, after, "a")
}

func TestPinSiblingsDottedIDs(t *testing.T) {
	src := "p.one\np.two\np.three\n"
	before := absolutePositions(t, src)

	out := moveWithPinning(t, src, "p.two", 40, 150)
	after := absolutePositions(t, out)

	assertUnmoved(t, before, after, "p.two")
}

// sectionOf returns the source lines of a top-level node's declaration.
func sectionOf(src, id string) string {
	var b strings.Builder
	in := false
	depth := 0
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !in && depth == 0 && (trimmed == id || strings.HasPrefix(trimmed, id+" ") || strings.HasPrefix(trimmed, id+":")) {
			in = true
		}
		if in {
			b.WriteString(line + "\n")
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if in && depth == 0 {
			in = false
		}
	}
	return b.String()
}
