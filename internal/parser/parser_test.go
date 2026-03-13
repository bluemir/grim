package parser

import (
	"testing"
)

// =============================================================================
// stateTopLevel — 각 분기별 테스트
// =============================================================================

func TestParse_TopLevel_Newline(t *testing.T) {
	doc, err := Parse("\n\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 0 {
		t.Errorf("expected 0 statements for newlines only, got %d", len(doc.Statements))
	}
}

func TestParse_TopLevel_Comment(t *testing.T) {
	doc, err := Parse("# comment")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(doc.Statements))
	}
	c, ok := doc.Statements[0].(*Comment)
	if !ok {
		t.Fatalf("expected Comment, got %T", doc.Statements[0])
	}
	if c.Text != " comment" {
		t.Errorf("expected ' comment', got %q", c.Text)
	}
}

func TestParse_TopLevel_EOF(t *testing.T) {
	doc, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 0 {
		t.Errorf("expected 0 statements for empty input, got %d", len(doc.Statements))
	}
}

func TestParse_TopLevel_RBrace_Unmatched(t *testing.T) {
	doc, err := Parse("}")
	if err == nil {
		t.Fatal("expected error for unmatched '}'")
	}
	_ = doc // should still return a doc
}

func TestParse_TopLevel_RBrace_ClosesBlock(t *testing.T) {
	doc, err := Parse("n: {\n}")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Block == nil {
		t.Fatal("expected block")
	}
}

func TestParse_TopLevel_Ident(t *testing.T) {
	doc, err := Parse("myNode")
	if err != nil {
		t.Fatal(err)
	}
	node, ok := doc.Statements[0].(*NodeDecl)
	if !ok {
		t.Fatalf("expected NodeDecl, got %T", doc.Statements[0])
	}
	if node.Path[0] != "myNode" {
		t.Errorf("expected 'myNode', got %v", node.Path)
	}
}

func TestParse_TopLevel_At(t *testing.T) {
	// @ outside block should error
	_, err := Parse("@layout {x: 1}")
	if err == nil {
		t.Fatal("expected error for metadata outside block")
	}
}

func TestParse_TopLevel_UnexpectedToken(t *testing.T) {
	// Colon at top level is unexpected
	_, err := Parse(":")
	if err == nil {
		t.Fatal("expected error for unexpected token")
	}
}

// =============================================================================
// stateInPath / statePathDot — 경로 파싱 분기별 테스트
// =============================================================================

func TestParse_Path_Single(t *testing.T) {
	doc, err := Parse("abc")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if len(node.Path) != 1 || node.Path[0] != "abc" {
		t.Errorf("expected [abc], got %v", node.Path)
	}
}

func TestParse_Path_Dotted(t *testing.T) {
	doc, err := Parse("a.b.c")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if len(node.Path) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(node.Path), node.Path)
	}
	if node.Path[0] != "a" || node.Path[1] != "b" || node.Path[2] != "c" {
		t.Errorf("expected [a b c], got %v", node.Path)
	}
}

func TestParse_Path_DotWithoutIdent(t *testing.T) {
	// `a.` without following ident should produce an error
	_, err := Parse("a.")
	if err == nil {
		t.Fatal("expected error for trailing dot")
	}
}

func TestParse_Path_DoubleDot(t *testing.T) {
	// `a..b` has a dot followed by dot (not ident)
	_, err := Parse("a..b")
	if err == nil {
		t.Fatal("expected error for double dot")
	}
}

// =============================================================================
// stateAfterIdent — 식별자 이후 분기별 테스트
// =============================================================================

func TestParse_AfterIdent_Arrow(t *testing.T) {
	doc, err := Parse("A -> B")
	if err != nil {
		t.Fatal(err)
	}
	edge, ok := doc.Statements[0].(*EdgeDecl)
	if !ok {
		t.Fatalf("expected EdgeDecl, got %T", doc.Statements[0])
	}
	if edge.From[0] != "A" || edge.To[0] != "B" {
		t.Errorf("expected A -> B, got %v -> %v", edge.From, edge.To)
	}
}

func TestParse_AfterIdent_Colon(t *testing.T) {
	doc, err := Parse(`node: "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Label == nil || node.Label.Value != "hello" {
		t.Errorf("expected label 'hello', got %v", node.Label)
	}
}

func TestParse_AfterIdent_LBrace(t *testing.T) {
	// node { } without colon
	doc, err := Parse("node {\n}")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Block == nil {
		t.Fatal("expected block on node")
	}
}

func TestParse_AfterIdent_Newline(t *testing.T) {
	doc, err := Parse("bareNode\n")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Path[0] != "bareNode" {
		t.Errorf("expected 'bareNode', got %v", node.Path)
	}
	if node.Label != nil || node.Block != nil {
		t.Error("bare node should have no label or block")
	}
}

func TestParse_AfterIdent_EOF(t *testing.T) {
	doc, err := Parse("bareNode")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Path[0] != "bareNode" {
		t.Errorf("expected 'bareNode', got %v", node.Path)
	}
}

func TestParse_AfterIdent_Comment(t *testing.T) {
	doc, err := Parse("node # inline comment")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(doc.Statements))
	}
	if _, ok := doc.Statements[0].(*NodeDecl); !ok {
		t.Error("expected NodeDecl first")
	}
	if _, ok := doc.Statements[1].(*Comment); !ok {
		t.Error("expected Comment second")
	}
}

func TestParse_AfterIdent_UnexpectedToken(t *testing.T) {
	// e.g. `node @` — @ is unexpected after an identifier
	_, err := Parse("node @")
	if err == nil {
		t.Fatal("expected error for unexpected token after ident")
	}
}

// =============================================================================
// stateEdgeTarget / stateEdgeTargetPath / stateEdgeTargetDot
// =============================================================================

func TestParse_EdgeTarget_Simple(t *testing.T) {
	doc, err := Parse("A -> B")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if len(edge.To) != 1 || edge.To[0] != "B" {
		t.Errorf("expected To=[B], got %v", edge.To)
	}
}

func TestParse_EdgeTarget_Dotted(t *testing.T) {
	doc, err := Parse("A -> x.y.z")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if len(edge.To) != 3 || edge.To[0] != "x" || edge.To[1] != "y" || edge.To[2] != "z" {
		t.Errorf("expected To=[x y z], got %v", edge.To)
	}
}

func TestParse_EdgeTarget_MissingIdent(t *testing.T) {
	// `A -> :` — colon is not an ident, should error
	_, err := Parse("A -> :")
	if err == nil {
		t.Fatal("expected error for missing edge target")
	}
}

func TestParse_EdgeTarget_DotMissingIdent(t *testing.T) {
	// `A -> B.` — trailing dot with no ident
	_, err := Parse("A -> B.")
	if err == nil {
		t.Fatal("expected error for trailing dot in edge target")
	}
}

func TestParse_EdgeTarget_DottedFrom(t *testing.T) {
	doc, err := Parse("a.b.c -> d")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if len(edge.From) != 3 || edge.From[0] != "a" || edge.From[1] != "b" || edge.From[2] != "c" {
		t.Errorf("expected From=[a b c], got %v", edge.From)
	}
}

func TestParse_EdgeTarget_HyphenedNames(t *testing.T) {
	doc, err := Parse("my-node.inner -> other-node")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.From[0] != "my-node" || edge.From[1] != "inner" {
		t.Errorf("unexpected from: %v", edge.From)
	}
	if edge.To[0] != "other-node" {
		t.Errorf("unexpected to: %v", edge.To)
	}
}

// =============================================================================
// stateAfterEdgeTarget — 엣지 대상 이후 분기별 테스트
// =============================================================================

func TestParse_AfterEdgeTarget_Colon_String(t *testing.T) {
	doc, err := Parse(`A -> B: "label"`)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label == nil || edge.Label.Value != "label" {
		t.Errorf("expected label 'label', got %v", edge.Label)
	}
}

func TestParse_AfterEdgeTarget_Colon_RestOfLine(t *testing.T) {
	doc, err := Parse("A -> B: Read Data")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label == nil || edge.Label.Value != "Read Data" {
		t.Errorf("expected label 'Read Data', got %v", edge.Label)
	}
}

func TestParse_AfterEdgeTarget_LBrace(t *testing.T) {
	doc, err := Parse("A -> B {\n}")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Block == nil {
		t.Fatal("expected block on edge")
	}
}

func TestParse_AfterEdgeTarget_Newline(t *testing.T) {
	doc, err := Parse("A -> B\n")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label != nil || edge.Block != nil {
		t.Error("bare edge should have no label or block")
	}
}

func TestParse_AfterEdgeTarget_EOF(t *testing.T) {
	doc, err := Parse("A -> B")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label != nil || edge.Block != nil {
		t.Error("bare edge should have no label or block")
	}
}

func TestParse_AfterEdgeTarget_Comment(t *testing.T) {
	doc, err := Parse("A -> B # inline")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(doc.Statements))
	}
	if _, ok := doc.Statements[0].(*EdgeDecl); !ok {
		t.Error("expected EdgeDecl first")
	}
	if _, ok := doc.Statements[1].(*Comment); !ok {
		t.Error("expected Comment second")
	}
}

func TestParse_AfterEdgeTarget_UnexpectedToken(t *testing.T) {
	_, err := Parse("A -> B @")
	if err == nil {
		t.Fatal("expected error for unexpected token after edge target")
	}
}

// =============================================================================
// stateAfterColon — 콜론 이후 분기별 테스트
// =============================================================================

func TestParse_AfterColon_String_Node(t *testing.T) {
	doc, err := Parse(`node: "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Label == nil || node.Label.Value != "hello" {
		t.Errorf("expected 'hello', got %v", node.Label)
	}
	if node.Label.Format != "plain" {
		t.Errorf("expected format 'plain', got %q", node.Label.Format)
	}
}

func TestParse_AfterColon_String_Edge(t *testing.T) {
	doc, err := Parse(`A -> B: "call"`)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label == nil || edge.Label.Value != "call" {
		t.Errorf("expected 'call', got %v", edge.Label)
	}
}

func TestParse_AfterColon_LBrace_Node(t *testing.T) {
	doc, err := Parse("node: {\n}")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Block == nil {
		t.Fatal("expected block")
	}
}

func TestParse_AfterColon_LBrace_Edge(t *testing.T) {
	doc, err := Parse("A -> B: {\n}")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Block == nil {
		t.Fatal("expected block on edge")
	}
}

func TestParse_AfterColon_Newline_Node(t *testing.T) {
	// `node:` with nothing after → bare node with no label
	doc, err := Parse("node:\n")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Label != nil {
		t.Error("expected nil label for empty colon")
	}
}

func TestParse_AfterColon_Newline_Edge(t *testing.T) {
	doc, err := Parse("A -> B:\n")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label != nil {
		t.Error("expected nil label for empty colon on edge")
	}
}

func TestParse_AfterColon_EOF_Node(t *testing.T) {
	doc, err := Parse("node:")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Label != nil {
		t.Error("expected nil label")
	}
}

func TestParse_AfterColon_EOF_Edge(t *testing.T) {
	doc, err := Parse("A -> B:")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label != nil {
		t.Error("expected nil label")
	}
}

func TestParse_AfterColon_RestOfLine_Node(t *testing.T) {
	doc, err := Parse("node: hello world")
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Label == nil || node.Label.Value != "hello world" {
		t.Errorf("expected 'hello world', got %v", node.Label)
	}
}

func TestParse_AfterColon_RestOfLine_Edge(t *testing.T) {
	doc, err := Parse("A -> B: Read")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label == nil || edge.Label.Value != "Read" {
		t.Errorf("expected 'Read', got %v", edge.Label)
	}
}

func TestParse_AfterColon_RestOfLine_MultipleWords(t *testing.T) {
	doc, err := Parse("A -> B: API Call Request")
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if edge.Label == nil || edge.Label.Value != "API Call Request" {
		t.Errorf("expected 'API Call Request', got %v", edge.Label)
	}
}

// =============================================================================
// stateMetaKey — @ 이후 메타 키워드 분기별 테스트
// =============================================================================

func TestParse_MetaKey_Text(t *testing.T) {
	input := "n: {\n  @text hello\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if _, ok := node.Block.Metadata[0].(*TextMeta); !ok {
		t.Fatalf("expected TextMeta, got %T", node.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Shape(t *testing.T) {
	input := "n: {\n  @shape diamond\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if _, ok := node.Block.Metadata[0].(*ShapeMeta); !ok {
		t.Fatalf("expected ShapeMeta, got %T", node.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Layout(t *testing.T) {
	input := "n: {\n  @layout {x: 1}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if _, ok := node.Block.Metadata[0].(*LayoutMeta); !ok {
		t.Fatalf("expected LayoutMeta, got %T", node.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Style(t *testing.T) {
	input := "n: {\n  @style {fill: red}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if _, ok := node.Block.Metadata[0].(*StyleMeta); !ok {
		t.Fatalf("expected StyleMeta, got %T", node.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Edge(t *testing.T) {
	input := "A -> B: {\n  @edge {anchors: [\"left\"]}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	if _, ok := edge.Block.Metadata[0].(*EdgeMeta); !ok {
		t.Fatalf("expected EdgeMeta, got %T", edge.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Icon(t *testing.T) {
	input := "n: {\n  @icon {url: \"https://example.com/icon.svg\"}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if _, ok := node.Block.Metadata[0].(*IconMeta); !ok {
		t.Fatalf("expected IconMeta, got %T", node.Block.Metadata[0])
	}
}

func TestParse_MetaKey_Unknown(t *testing.T) {
	input := "n: {\n  @unknown value\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for unknown meta key")
	}
}

func TestParse_MetaKey_NonIdent(t *testing.T) {
	// `@ {` — brace is not an ident after @
	input := "n: {\n  @ {\n}\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for non-ident after @")
	}
}

func TestParse_MetaKey_OutsideBlock(t *testing.T) {
	_, err := Parse("@style {fill: red}")
	if err == nil {
		t.Fatal("expected error for metadata outside block")
	}
}

// =============================================================================
// stateTextMeta — @text 이후 분기별 테스트
// =============================================================================

func TestParse_TextMeta_Bracket_Markdown_RawBlock(t *testing.T) {
	input := "n: {\n  @text[markdown] {|\n    # Header\n    - item\n  |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "markdown" {
		t.Errorf("expected 'markdown', got %q", text.Format)
	}
	if text.Value == "" {
		t.Error("expected non-empty value for markdown raw block")
	}
}

func TestParse_TextMeta_Bracket_Latex_RawBlock(t *testing.T) {
	input := "n: {\n  @text[latex] {|\n    \\frac{1}{2}\n  |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "latex" {
		t.Errorf("expected 'latex', got %q", text.Format)
	}
	if text.Value == "" {
		t.Error("expected non-empty value for latex raw block")
	}
}

func TestParse_TextMeta_Bracket_Plain_String(t *testing.T) {
	input := "n: {\n  @text[plain] \"hello\"\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "plain" || text.Value != "hello" {
		t.Errorf("expected plain/'hello', got %s/%q", text.Format, text.Value)
	}
}

func TestParse_TextMeta_Bracket_Plain_RestOfLine(t *testing.T) {
	input := "n: {\n  @text[plain] hello world\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "plain" || text.Value != "hello world" {
		t.Errorf("expected plain/'hello world', got %s/%q", text.Format, text.Value)
	}
}

func TestParse_TextMeta_RawBlock(t *testing.T) {
	input := "n: {\n  @text {| content inside |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "plain" {
		t.Errorf("expected 'plain', got %q", text.Format)
	}
	if text.Value == "" {
		t.Error("expected non-empty raw block value")
	}
}

func TestParse_TextMeta_String(t *testing.T) {
	input := "n: {\n  @text \"hello world\"\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", text.Value)
	}
}

func TestParse_TextMeta_RestOfLine(t *testing.T) {
	input := "n: {\n  @text PostgreSQL\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "plain" || text.Value != "PostgreSQL" {
		t.Errorf("expected plain/'PostgreSQL', got %s/%q", text.Format, text.Value)
	}
}

func TestParse_TextMeta_RestOfLine_MultipleWords(t *testing.T) {
	input := "n: {\n  @text API Call\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Value != "API Call" {
		t.Errorf("expected 'API Call', got %q", text.Value)
	}
}

func TestParse_TextMeta_Newline_EmptyValue(t *testing.T) {
	input := "n: {\n  @text\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Value != "" {
		t.Errorf("expected empty value, got %q", text.Value)
	}
}

// =============================================================================
// stateTextFormat — @text[format] 분기별 테스트
// =============================================================================

func TestParse_TextFormat_NonIdent(t *testing.T) {
	input := "n: {\n  @text[123]\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for non-ident format")
	}
}

func TestParse_TextFormat_MissingRBracket(t *testing.T) {
	input := "n: {\n  @text[markdown hello\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for missing ']'")
	}
}

// =============================================================================
// stateSimpleMeta (@shape) — 분기별 테스트
// =============================================================================

func TestParse_Shape_SingleValue(t *testing.T) {
	input := "n: {\n  @shape cloud\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	shape := node.Block.Metadata[0].(*ShapeMeta)
	if shape.Value != "cloud" {
		t.Errorf("expected 'cloud', got %q", shape.Value)
	}
}

func TestParse_Shape_Rectangle(t *testing.T) {
	input := "n: {\n  @shape rectangle\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	shape := node.Block.Metadata[0].(*ShapeMeta)
	if shape.Value != "rectangle" {
		t.Errorf("expected 'rectangle', got %q", shape.Value)
	}
}

func TestParse_Shape_MissingValue(t *testing.T) {
	input := "n: {\n  @shape\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for missing shape value")
	}
}

// =============================================================================
// stateObjectMeta (@layout, @style, @edge, @icon) — 분기별 테스트
// =============================================================================

func TestParse_Layout_Values(t *testing.T) {
	input := "n: {\n  @layout {x: 100, y: 200, w: 300, h: 400}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	layout := node.Block.Metadata[0].(*LayoutMeta)
	if layout.Values["x"] != int64(100) {
		t.Errorf("expected x=100, got %v", layout.Values["x"])
	}
	if layout.Values["y"] != int64(200) {
		t.Errorf("expected y=200, got %v", layout.Values["y"])
	}
	if layout.Values["w"] != int64(300) {
		t.Errorf("expected w=300, got %v", layout.Values["w"])
	}
	if layout.Values["h"] != int64(400) {
		t.Errorf("expected h=400, got %v", layout.Values["h"])
	}
}

func TestParse_Layout_PositionOnly(t *testing.T) {
	input := "n: {\n  @layout {x: 50, y: 60}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	layout := node.Block.Metadata[0].(*LayoutMeta)
	if layout.Values["x"] != int64(50) || layout.Values["y"] != int64(60) {
		t.Errorf("unexpected values: %v", layout.Values)
	}
}

func TestParse_Style_Values(t *testing.T) {
	input := "n: {\n  @style {fill: blue, stroke: black, opacity: 0.5}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["fill"] != "blue" {
		t.Errorf("expected fill=blue, got %v", style.Values["fill"])
	}
	if style.Values["stroke"] != "black" {
		t.Errorf("expected stroke=black, got %v", style.Values["stroke"])
	}
	if style.Values["opacity"] != 0.5 {
		t.Errorf("expected opacity=0.5, got %v (%T)", style.Values["opacity"], style.Values["opacity"])
	}
}

func TestParse_Style_StringValue(t *testing.T) {
	input := "n: {\n  @style {fill: \"#3498db\"}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["fill"] != "#3498db" {
		t.Errorf("expected fill='#3498db', got %v", style.Values["fill"])
	}
}

func TestParse_EdgeMeta_Anchors(t *testing.T) {
	input := "A -> B: {\n  @edge {anchors: [\"right\", \"left\"]}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	em := edge.Block.Metadata[0].(*EdgeMeta)
	anchors, ok := em.Values["anchors"].([]interface{})
	if !ok {
		t.Fatalf("expected anchors array, got %T", em.Values["anchors"])
	}
	if len(anchors) != 2 || anchors[0] != "right" || anchors[1] != "left" {
		t.Errorf("unexpected anchors: %v", anchors)
	}
}

func TestParse_EdgeMeta_Waypoints(t *testing.T) {
	input := "A -> B: {\n  @edge {waypoints: [{x: 100, y: 200}]}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	em := edge.Block.Metadata[0].(*EdgeMeta)
	wps, ok := em.Values["waypoints"].([]interface{})
	if !ok {
		t.Fatalf("expected waypoints array, got %T", em.Values["waypoints"])
	}
	if len(wps) != 1 {
		t.Fatalf("expected 1 waypoint, got %d", len(wps))
	}
	wp, ok := wps[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected waypoint object, got %T", wps[0])
	}
	if wp["x"] != int64(100) || wp["y"] != int64(200) {
		t.Errorf("unexpected waypoint: %v", wp)
	}
}

func TestParse_EdgeMeta_Gap(t *testing.T) {
	input := "A -> B: {\n  @edge {gap: {start: 1, end: 1}}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	em := edge.Block.Metadata[0].(*EdgeMeta)
	gap, ok := em.Values["gap"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gap object, got %T", em.Values["gap"])
	}
	if gap["start"] != int64(1) || gap["end"] != int64(1) {
		t.Errorf("unexpected gap: %v", gap)
	}
}

func TestParse_Icon_WithUrl(t *testing.T) {
	input := "n: {\n  @icon {url: \"https://example.com/icon.svg\"}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	icon := node.Block.Metadata[0].(*IconMeta)
	if icon.Values["url"] != "https://example.com/icon.svg" {
		t.Errorf("unexpected url: %v", icon.Values["url"])
	}
}

func TestParse_Icon_WithAllAttrs(t *testing.T) {
	input := "n: {\n  @icon {url: \"https://x.com/i.svg\", position: top, size: 48}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	icon := node.Block.Metadata[0].(*IconMeta)
	if icon.Values["url"] != "https://x.com/i.svg" {
		t.Errorf("unexpected url: %v", icon.Values["url"])
	}
	if icon.Values["position"] != "top" {
		t.Errorf("unexpected position: %v", icon.Values["position"])
	}
	if icon.Values["size"] != int64(48) {
		t.Errorf("unexpected size: %v (%T)", icon.Values["size"], icon.Values["size"])
	}
}

func TestParse_ObjectMeta_MissingBrace(t *testing.T) {
	// `@layout x: 1` — missing { after @layout
	input := "n: {\n  @layout x: 1\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for missing '{' after @layout")
	}
}

// =============================================================================
// JSON-like parsing — parseJSONObject / parseJSONValue / parseJSONArray
// =============================================================================

func TestParse_JSON_BooleanTrue(t *testing.T) {
	input := "n: {\n  @style {visible: true}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["visible"] != true {
		t.Errorf("expected true, got %v (%T)", style.Values["visible"], style.Values["visible"])
	}
}

func TestParse_JSON_BooleanFalse(t *testing.T) {
	input := "n: {\n  @style {visible: false}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["visible"] != false {
		t.Errorf("expected false, got %v (%T)", style.Values["visible"], style.Values["visible"])
	}
}

func TestParse_JSON_Null(t *testing.T) {
	input := "n: {\n  @style {bg: null}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["bg"] != nil {
		t.Errorf("expected nil, got %v", style.Values["bg"])
	}
}

func TestParse_JSON_NestedObject(t *testing.T) {
	input := "n: {\n  @style {border: {width: 2, color: red}}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	border, ok := style.Values["border"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested object, got %T", style.Values["border"])
	}
	if border["width"] != int64(2) {
		t.Errorf("expected width=2, got %v", border["width"])
	}
	if border["color"] != "red" {
		t.Errorf("expected color=red, got %v", border["color"])
	}
}

func TestParse_JSON_EmptyObject(t *testing.T) {
	input := "n: {\n  @style {}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if len(style.Values) != 0 {
		t.Errorf("expected empty object, got %v", style.Values)
	}
}

func TestParse_JSON_EmptyArray(t *testing.T) {
	input := "A -> B: {\n  @edge {anchors: []}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[0].(*EdgeDecl)
	em := edge.Block.Metadata[0].(*EdgeMeta)
	anchors, ok := em.Values["anchors"].([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", em.Values["anchors"])
	}
	if len(anchors) != 0 {
		t.Errorf("expected empty array, got %v", anchors)
	}
}

func TestParse_JSON_NumberArray(t *testing.T) {
	input := "n: {\n  @style {dash: [5, 3]}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	dash, ok := style.Values["dash"].([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", style.Values["dash"])
	}
	if len(dash) != 2 || dash[0] != int64(5) || dash[1] != int64(3) {
		t.Errorf("unexpected dash array: %v", dash)
	}
}

func TestParse_JSON_KeyWithoutValue(t *testing.T) {
	input := "n: {\n  @style {bold}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["bold"] != true {
		t.Errorf("expected bold=true, got %v", style.Values["bold"])
	}
}

func TestParse_JSON_FloatValue(t *testing.T) {
	input := "n: {\n  @style {opacity: 0.75}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["opacity"] != 0.75 {
		t.Errorf("expected 0.75, got %v (%T)", style.Values["opacity"], style.Values["opacity"])
	}
}

func TestParse_JSON_StringKey(t *testing.T) {
	input := "n: {\n  @style {\"stroke-width\": 2}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	style := node.Block.Metadata[0].(*StyleMeta)
	if style.Values["stroke-width"] != int64(2) {
		t.Errorf("expected stroke-width=2, got %v", style.Values["stroke-width"])
	}
}

// =============================================================================
// Raw block ({| |}) — 분기별 테스트
// =============================================================================

func TestParse_RawBlock_UnmatchedBraces(t *testing.T) {
	// Unmatched { inside {| |} should be fine — no depth tracking needed
	input := "n: {\n  @text[markdown] {| code: `func() {` |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "markdown" {
		t.Errorf("expected 'markdown', got %q", text.Format)
	}
	if text.Value == "" {
		t.Error("expected non-empty value")
	}
}

func TestParse_RawBlock_UnmatchedClosingBrace(t *testing.T) {
	// Standalone } inside {| |} should NOT end the outer node block
	input := "n: {\n  @text {| JSON: } |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Value != "JSON: }" {
		t.Errorf("expected 'JSON: }', got %q", text.Value)
	}
}

func TestParse_RawBlock_NestedBraces(t *testing.T) {
	input := "n: {\n  @text[latex] {| \\frac{a}{b} |}\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	text := node.Block.Metadata[0].(*TextMeta)
	if text.Format != "latex" {
		t.Errorf("expected 'latex', got %q", text.Format)
	}
	// The raw block should contain the nested braces content
	if text.Value == "" {
		t.Error("expected non-empty raw block value")
	}
}

func TestParse_RawBlock_Unterminated(t *testing.T) {
	input := "n: {\n  @text[markdown] {| never closed"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for unterminated raw block")
	}
}

// =============================================================================
// Block nesting — 블록 중첩 테스트
// =============================================================================

func TestParse_Nesting_Simple(t *testing.T) {
	input := "outer: {\n  inner: \"hello\"\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	outer := doc.Statements[0].(*NodeDecl)
	if len(outer.Block.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(outer.Block.Children))
	}
	inner := outer.Block.Children[0].(*NodeDecl)
	if inner.Path[0] != "inner" || inner.Label.Value != "hello" {
		t.Errorf("unexpected inner: %v %v", inner.Path, inner.Label)
	}
}

func TestParse_Nesting_Deep(t *testing.T) {
	input := "a: {\n  b: {\n    c: {\n      d: \"leaf\"\n    }\n  }\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	a := doc.Statements[0].(*NodeDecl)
	b := a.Block.Children[0].(*NodeDecl)
	c := b.Block.Children[0].(*NodeDecl)
	d := c.Block.Children[0].(*NodeDecl)
	if d.Path[0] != "d" || d.Label.Value != "leaf" {
		t.Errorf("unexpected d: %v %v", d.Path, d.Label)
	}
}

func TestParse_Nesting_EdgeInBlock(t *testing.T) {
	input := "network: {\n  A -> B\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	edge := node.Block.Children[0].(*EdgeDecl)
	if edge.From[0] != "A" || edge.To[0] != "B" {
		t.Errorf("expected A -> B, got %v -> %v", edge.From, edge.To)
	}
}

func TestParse_Nesting_MetadataAndChildren(t *testing.T) {
	input := "n: {\n  @layout {x: 1}\n  @style {fill: red}\n  child: \"hi\"\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if len(node.Block.Metadata) != 2 {
		t.Errorf("expected 2 metadata, got %d", len(node.Block.Metadata))
	}
	if len(node.Block.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(node.Block.Children))
	}
}

func TestParse_Nesting_CommentInBlock(t *testing.T) {
	input := "n: {\n  # a comment\n  child\n}"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if len(node.Block.Children) != 2 {
		t.Fatalf("expected 2 children (comment + node), got %d", len(node.Block.Children))
	}
	if _, ok := node.Block.Children[0].(*Comment); !ok {
		t.Error("expected Comment first")
	}
	if _, ok := node.Block.Children[1].(*NodeDecl); !ok {
		t.Error("expected NodeDecl second")
	}
}

func TestParse_Nesting_Unclosed(t *testing.T) {
	input := "n: {\n  child"
	doc, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for unclosed block")
	}
	// Should still have partial AST
	if doc == nil || len(doc.Statements) == 0 {
		t.Fatal("expected partial AST")
	}
}

// =============================================================================
// Best-effort parsing — 에러 복구 테스트
// =============================================================================

func TestParse_BestEffort_UnclosedBlock(t *testing.T) {
	input := "validNode\nbroken: {\nanotherNode"
	doc, err := Parse(input)
	if err == nil {
		t.Fatal("expected error")
	}
	if doc == nil {
		t.Fatal("expected non-nil doc")
	}
	// validNode should be in doc
	found := false
	for _, s := range doc.Statements {
		if n, ok := s.(*NodeDecl); ok && len(n.Path) > 0 && n.Path[0] == "validNode" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'validNode' in best-effort parse")
	}
}

func TestParse_BestEffort_MultipleErrors(t *testing.T) {
	input := "}\n}\n}"
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected errors")
	}
	me, ok := err.(*MultiParseError)
	if !ok {
		t.Fatalf("expected MultiParseError, got %T", err)
	}
	if len(me.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d", len(me.Errors))
	}
}

// =============================================================================
// Line tracking — AST 라인 번호 테스트
// =============================================================================

func TestParse_LineTracking_Node(t *testing.T) {
	input := "\n\nnode"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	node := doc.Statements[0].(*NodeDecl)
	if node.Line != 3 {
		t.Errorf("expected node at line 3, got %d", node.Line)
	}
}

func TestParse_LineTracking_Edge(t *testing.T) {
	input := "A\nB -> C"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	edge := doc.Statements[1].(*EdgeDecl)
	if edge.Line != 2 {
		t.Errorf("expected edge at line 2, got %d", edge.Line)
	}
}

func TestParse_LineTracking_Comment(t *testing.T) {
	input := "\n# comment"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	c := doc.Statements[0].(*Comment)
	if c.Line != 2 {
		t.Errorf("expected comment at line 2, got %d", c.Line)
	}
}

// =============================================================================
// Multiple statements — 복합 시나리오 테스트
// =============================================================================

func TestParse_MultipleNodes(t *testing.T) {
	input := "A\nB\nC"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(doc.Statements))
	}
	for i, name := range []string{"A", "B", "C"} {
		node := doc.Statements[i].(*NodeDecl)
		if node.Path[0] != name {
			t.Errorf("statement[%d]: expected %q, got %v", i, name, node.Path)
		}
	}
}

func TestParse_NodesAndEdgesMixed(t *testing.T) {
	input := "A\nB\nA -> B\nC"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(doc.Statements))
	}
	if _, ok := doc.Statements[0].(*NodeDecl); !ok {
		t.Error("expected NodeDecl at 0")
	}
	if _, ok := doc.Statements[2].(*EdgeDecl); !ok {
		t.Error("expected EdgeDecl at 2")
	}
}

func TestParse_CommentsInterspersed(t *testing.T) {
	input := "# header\nA\n# mid\nB\n# footer"
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(doc.Statements))
	}
	for i, wantType := range []string{"comment", "node", "comment", "node", "comment"} {
		switch wantType {
		case "comment":
			if _, ok := doc.Statements[i].(*Comment); !ok {
				t.Errorf("statement[%d]: expected Comment, got %T", i, doc.Statements[i])
			}
		case "node":
			if _, ok := doc.Statements[i].(*NodeDecl); !ok {
				t.Errorf("statement[%d]: expected NodeDecl, got %T", i, doc.Statements[i])
			}
		}
	}
}

// =============================================================================
// Full example — note.md 전체 예제 통합 테스트
// =============================================================================

func TestParse_FullExample(t *testing.T) {
	input := `network: {
  @layout {x: 0, y: 0, w: 600, h: 400}
  @style  {fill: blue, stroke: black, shadow: 3}

  frontend: {
    @layout {x: 100, y: 150}
    @style  {fill: blue, stroke: black}
  }

  backend: {
    @layout {x: 400, y: 150}
  }

  frontend -> backend: {
    @edge {anchors: ["right", "left"], waypoints: [{x: 250, y: 150}], gap: {start: 1, end: 1}}
    @text API Call
  }
}

frontend -> database: Read

explanation: {
  @text[markdown] {|
    # I can do headers
    - lists
  |}
}

pg: {
  @shape cloud
  @text PostgreSQL
}

my-node: {
  inside-node: {
    inside-sq-node: "hello"
  }
}

my-node.inside-node.inside-sq-node -> pg

# comment`

	doc, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Top-level: network, BlankLine, edge, BlankLine, explanation, BlankLine,
	//            pg, BlankLine, my-node, BlankLine, dotted-edge, BlankLine, comment
	// BlankLine nodes are now preserved in the AST between top-level declarations.
	if len(doc.Statements) < 7 {
		t.Errorf("expected at least 7 top-level statements, got %d", len(doc.Statements))
	}

	// Helper: collect non-blank top-level statements for index-stable access
	var nonBlank []Statement
	for _, s := range doc.Statements {
		if _, ok := s.(*BlankLine); !ok {
			nonBlank = append(nonBlank, s)
		}
	}
	if len(nonBlank) < 7 {
		t.Fatalf("expected at least 7 non-blank top-level statements, got %d", len(nonBlank))
	}

	// Verify network
	network := nonBlank[0].(*NodeDecl)
	if network.Path[0] != "network" {
		t.Errorf("expected 'network', got %v", network.Path)
	}
	if len(network.Block.Metadata) != 2 {
		t.Errorf("expected 2 metadata on network, got %d", len(network.Block.Metadata))
	}
	// Filter blank lines from block children for index-stable access
	var networkChildren []Statement
	for _, s := range network.Block.Children {
		if _, ok := s.(*BlankLine); !ok {
			networkChildren = append(networkChildren, s)
		}
	}
	if len(networkChildren) != 3 {
		t.Errorf("expected 3 non-blank children in network (frontend, backend, edge), got %d", len(networkChildren))
	}

	// Verify frontend inside network
	fe := networkChildren[0].(*NodeDecl)
	if fe.Path[0] != "frontend" {
		t.Errorf("expected 'frontend', got %v", fe.Path)
	}
	if len(fe.Block.Metadata) != 2 {
		t.Errorf("expected 2 metadata on frontend, got %d", len(fe.Block.Metadata))
	}

	// Verify backend inside network
	be := networkChildren[1].(*NodeDecl)
	if be.Path[0] != "backend" {
		t.Errorf("expected 'backend', got %v", be.Path)
	}

	// Verify edge inside network
	innerEdge := networkChildren[2].(*EdgeDecl)
	if innerEdge.From[0] != "frontend" || innerEdge.To[0] != "backend" {
		t.Errorf("expected frontend -> backend edge")
	}
	if len(innerEdge.Block.Metadata) != 2 {
		t.Errorf("expected 2 metadata on inner edge, got %d", len(innerEdge.Block.Metadata))
	}

	// Verify top-level edge
	topEdge := nonBlank[1].(*EdgeDecl)
	if topEdge.From[0] != "frontend" || topEdge.To[0] != "database" {
		t.Errorf("expected frontend -> database edge")
	}
	if topEdge.Label == nil || topEdge.Label.Value != "Read" {
		t.Errorf("expected label 'Read', got %v", topEdge.Label)
	}

	// Verify explanation
	expl := nonBlank[2].(*NodeDecl)
	if expl.Path[0] != "explanation" {
		t.Errorf("expected 'explanation', got %v", expl.Path)
	}
	explText := expl.Block.Metadata[0].(*TextMeta)
	if explText.Format != "markdown" {
		t.Errorf("expected markdown format, got %q", explText.Format)
	}

	// Verify pg
	pg := nonBlank[3].(*NodeDecl)
	if pg.Path[0] != "pg" {
		t.Errorf("expected 'pg', got %v", pg.Path)
	}
	pgShape := pg.Block.Metadata[0].(*ShapeMeta)
	if pgShape.Value != "cloud" {
		t.Errorf("expected 'cloud', got %q", pgShape.Value)
	}

	// Verify dotted edge at end
	dottedEdge := nonBlank[5].(*EdgeDecl)
	if len(dottedEdge.From) != 3 {
		t.Errorf("expected 3 from segments, got %d", len(dottedEdge.From))
	}
	if dottedEdge.From[0] != "my-node" || dottedEdge.From[1] != "inside-node" || dottedEdge.From[2] != "inside-sq-node" {
		t.Errorf("unexpected from: %v", dottedEdge.From)
	}
	if dottedEdge.To[0] != "pg" {
		t.Errorf("expected To=[pg], got %v", dottedEdge.To)
	}

	// Verify trailing comment
	lastComment := nonBlank[6].(*Comment)
	if lastComment.Text != " comment" {
		t.Errorf("expected ' comment', got %q", lastComment.Text)
	}
}
