package renderer

import (
	"testing"

	"github.com/bluemir/grim/internal/parser"
)

func TestDottedPathNodeReparenting(t *testing.T) {
	// When a node is declared at the top level with a dotted path like
	// "pgdpb1.project-namespace.victoria-metrics.vm-select",
	// it should be reparented under its ancestor hierarchy, not stay top-level.
	input := `pgdpb1: {
    @layout { x: 516, y: 132 }
}
pgrpl1: {
    @layout { x: 728, y: 208 }
    project-namespace: {
    }
    system-namespace: {
    }
}
pgrpl1.project-namespace.promxy -> pgdpb1.project-namespace.victoria-metrics.vm-select
pgdpb1.project-namespace.victoria-metrics.vm-select: {
    @layout { x: 560, y: 658 }
}`

	doc, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	layout := BuildLayout(doc)

	// vm-select should NOT be a top-level node
	for _, n := range layout.Nodes {
		if n.ID == "pgdpb1.project-namespace.victoria-metrics.vm-select" {
			t.Error("vm-select should not be a top-level node")
		}
	}

	// Find pgdpb1 -> project-namespace -> victoria-metrics -> vm-select
	var pgdpb1 *LayoutNode
	for _, n := range layout.Nodes {
		if n.ID == "pgdpb1" {
			pgdpb1 = n
		}
	}
	if pgdpb1 == nil {
		t.Fatal("expected top-level pgdpb1 node")
	}

	var projNs *LayoutNode
	for _, c := range pgdpb1.Children {
		if c.ID == "pgdpb1.project-namespace" {
			projNs = c
		}
	}
	if projNs == nil {
		t.Fatal("expected pgdpb1.project-namespace as child of pgdpb1")
	}

	var vm *LayoutNode
	for _, c := range projNs.Children {
		if c.ID == "pgdpb1.project-namespace.victoria-metrics" {
			vm = c
		}
	}
	if vm == nil {
		t.Fatal("expected victoria-metrics as child of project-namespace")
	}

	var vmSelect *LayoutNode
	for _, c := range vm.Children {
		if c.ID == "pgdpb1.project-namespace.victoria-metrics.vm-select" {
			vmSelect = c
		}
	}
	if vmSelect == nil {
		t.Fatal("expected vm-select as child of victoria-metrics")
	}

	// promxy should be nested under pgrpl1.project-namespace (implicit)
	var pgrpl1 *LayoutNode
	for _, n := range layout.Nodes {
		if n.ID == "pgrpl1" {
			pgrpl1 = n
		}
	}
	if pgrpl1 == nil {
		t.Fatal("expected top-level pgrpl1 node")
	}

	var pgrpl1ProjNs *LayoutNode
	for _, c := range pgrpl1.Children {
		if c.ID == "pgrpl1.project-namespace" {
			pgrpl1ProjNs = c
		}
	}
	if pgrpl1ProjNs == nil {
		t.Fatal("expected pgrpl1.project-namespace")
	}

	found := false
	for _, c := range pgrpl1ProjNs.Children {
		if c.ID == "pgrpl1.project-namespace.promxy" {
			found = true
		}
	}
	if !found {
		t.Error("expected promxy as child of pgrpl1.project-namespace")
	}
}
