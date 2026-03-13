package parser

import (
	"strings"
	"testing"
)

func TestSetLayout_UpdateExistingLayout(t *testing.T) {
	src := "node {\n\t@layout { x: 10, y: 20, w: 100, h: 50 }\n}\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "node", 30, 40)
	result := Format(doc)
	// x, y updated
	if !strings.Contains(result, "x: 30") {
		t.Errorf("expected x: 30 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 40") {
		t.Errorf("expected y: 40 in output, got:\n%s", result)
	}
	// w, h preserved
	if !strings.Contains(result, "w: 100") {
		t.Errorf("expected w: 100 preserved in output, got:\n%s", result)
	}
	if !strings.Contains(result, "h: 50") {
		t.Errorf("expected h: 50 preserved in output, got:\n%s", result)
	}
}

func TestSetLayout_BlockWithoutLayout(t *testing.T) {
	src := "node {\n\t@shape rect\n}\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "node", 10, 20)
	result := Format(doc)
	if !strings.Contains(result, "@layout") {
		t.Errorf("expected @layout in output, got:\n%s", result)
	}
	if !strings.Contains(result, "x: 10") {
		t.Errorf("expected x: 10 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 20") {
		t.Errorf("expected y: 20 in output, got:\n%s", result)
	}
	// @shape still present
	if !strings.Contains(result, "@shape") {
		t.Errorf("expected @shape preserved in output, got:\n%s", result)
	}
}

func TestSetLayout_NodeWithoutBlock(t *testing.T) {
	src := "node\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "node", 5, 15)
	result := Format(doc)
	if !strings.Contains(result, "@layout") {
		t.Errorf("expected @layout in output, got:\n%s", result)
	}
	if !strings.Contains(result, "x: 5") {
		t.Errorf("expected x: 5 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 15") {
		t.Errorf("expected y: 15 in output, got:\n%s", result)
	}
}

func TestSetLayout_ImplicitTopLevel(t *testing.T) {
	src := ""
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "newnode", 100, 200)
	result := Format(doc)
	if !strings.Contains(result, "newnode") {
		t.Errorf("expected newnode in output, got:\n%s", result)
	}
	if !strings.Contains(result, "x: 100") {
		t.Errorf("expected x: 100 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 200") {
		t.Errorf("expected y: 200 in output, got:\n%s", result)
	}
}

func TestSetLayout_ImplicitNested_ParentExists(t *testing.T) {
	src := "parent {\n}\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "parent.child", 50, 60)
	result := Format(doc)
	if !strings.Contains(result, "parent") {
		t.Errorf("expected parent in output, got:\n%s", result)
	}
	if !strings.Contains(result, "child") {
		t.Errorf("expected child in output, got:\n%s", result)
	}
	if !strings.Contains(result, "x: 50") {
		t.Errorf("expected x: 50 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 60") {
		t.Errorf("expected y: 60 in output, got:\n%s", result)
	}
}

func TestSetLayout_ImplicitNested_BothAbsent(t *testing.T) {
	src := ""
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "parent.child", 7, 8)
	result := Format(doc)
	if !strings.Contains(result, "parent") {
		t.Errorf("expected parent in output, got:\n%s", result)
	}
	if !strings.Contains(result, "child") {
		t.Errorf("expected child in output, got:\n%s", result)
	}
	if !strings.Contains(result, "x: 7") {
		t.Errorf("expected x: 7 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 8") {
		t.Errorf("expected y: 8 in output, got:\n%s", result)
	}
}

func TestSetLayout_FlatDotted(t *testing.T) {
	// flat dotted declaration: parent.child { }
	src := "parent.child {\n}\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "parent.child", 11, 22)
	result := Format(doc)
	if !strings.Contains(result, "x: 11") {
		t.Errorf("expected x: 11 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 22") {
		t.Errorf("expected y: 22 in output, got:\n%s", result)
	}
}

func TestSetLayout_NestedBlock(t *testing.T) {
	// nested block: parent { child { } }
	src := "parent {\n\tchild {\n\t}\n}\n"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	SetLayout(doc, "parent.child", 33, 44)
	result := Format(doc)
	if !strings.Contains(result, "x: 33") {
		t.Errorf("expected x: 33 in output, got:\n%s", result)
	}
	if !strings.Contains(result, "y: 44") {
		t.Errorf("expected y: 44 in output, got:\n%s", result)
	}
	// parent should not have @layout
	parentLayoutIdx := strings.Index(result, "parent {")
	layoutIdx := strings.Index(result, "@layout")
	childIdx := strings.Index(result, "child {")
	if parentLayoutIdx >= 0 && layoutIdx >= 0 && layoutIdx < childIdx {
		t.Errorf("parent should not have @layout; only child should, got:\n%s", result)
	}
}
