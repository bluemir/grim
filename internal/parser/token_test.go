package parser

import (
	"testing"
)

// =============================================================================
// Helper
// =============================================================================

// expectTokens checks that token types and values match (ignoring position).
func expectTokens(t *testing.T, got []Token, expected ...Token) {
	t.Helper()
	if len(got) != len(expected) {
		t.Fatalf("expected %d tokens, got %d:\n  got: %v", len(expected), len(got), got)
	}
	for i, exp := range expected {
		if got[i].Type != exp.Type {
			t.Errorf("token[%d]: expected type %s, got %s (value=%q)", i, exp.Type, got[i].Type, got[i].Value)
		}
		if exp.Value != "" && got[i].Value != exp.Value {
			t.Errorf("token[%d]: expected value %q, got %q", i, exp.Value, got[i].Value)
		}
	}
}

// =============================================================================
// lexDefault — 각 분기별 테스트
// =============================================================================

func TestLexDefault_EmptyInput(t *testing.T) {
	tokens, err := Tokenize("")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens, Token{Type: TokenEOF})
}

func TestLexDefault_Newline(t *testing.T) {
	tokens, err := Tokenize("\n")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNewline, Value: "\n"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_MultipleNewlines(t *testing.T) {
	tokens, err := Tokenize("\n\n\n")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNewline},
		Token{Type: TokenNewline},
		Token{Type: TokenNewline},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_SpacesSkipped(t *testing.T) {
	tokens, err := Tokenize("   ")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens, Token{Type: TokenEOF})
}

func TestLexDefault_TabsSkipped(t *testing.T) {
	tokens, err := Tokenize("\t\t")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens, Token{Type: TokenEOF})
}

func TestLexDefault_CarriageReturnSkipped(t *testing.T) {
	tokens, err := Tokenize("\r")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens, Token{Type: TokenEOF})
}

func TestLexDefault_CRLFNewline(t *testing.T) {
	tokens, err := Tokenize("\r\n")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNewline, Value: "\n"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_HashStartsComment(t *testing.T) {
	tokens, err := Tokenize("# hello")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: " hello"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_QuoteStartsString(t *testing.T) {
	tokens, err := Tokenize(`"abc"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "abc"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_Arrow(t *testing.T) {
	tokens, err := Tokenize("->")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_LBrace(t *testing.T) {
	tokens, err := Tokenize("{")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_RBrace(t *testing.T) {
	tokens, err := Tokenize("}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_LBracket(t *testing.T) {
	tokens, err := Tokenize("[")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBracket, Value: "["},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_RBracket(t *testing.T) {
	tokens, err := Tokenize("]")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRBracket, Value: "]"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_Colon(t *testing.T) {
	tokens, err := Tokenize(":")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_Dot(t *testing.T) {
	tokens, err := Tokenize(".")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenDot, Value: "."},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_At(t *testing.T) {
	tokens, err := Tokenize("@")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenAt, Value: "@"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_Comma(t *testing.T) {
	tokens, err := Tokenize(",")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComma, Value: ","},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_IdentStart_Letter(t *testing.T) {
	tokens, err := Tokenize("abc")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "abc"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_IdentStart_Underscore(t *testing.T) {
	tokens, err := Tokenize("_private")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "_private"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_NumberStart(t *testing.T) {
	tokens, err := Tokenize("7")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "7"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_UnknownCharEmitsIdent(t *testing.T) {
	// Unknown characters (!, ~, ^, etc.) should be emitted as Ident tokens
	tokens, err := Tokenize("!")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "!"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_MultipleUnknownChars(t *testing.T) {
	tokens, err := Tokenize("!~^")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "!"},
		Token{Type: TokenIdent, Value: "~"},
		Token{Type: TokenIdent, Value: "^"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_HyphenNotArrow(t *testing.T) {
	// Standalone `-` (not followed by `>`) should be emitted as Ident
	tokens, err := Tokenize("- ")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "-"},
		Token{Type: TokenEOF},
	)
}

func TestLexDefault_AllSymbolsSequence(t *testing.T) {
	tokens, err := Tokenize("{} [] : . @ ,")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenLBracket, Value: "["},
		Token{Type: TokenRBracket, Value: "]"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenDot, Value: "."},
		Token{Type: TokenAt, Value: "@"},
		Token{Type: TokenComma, Value: ","},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// lexIdent — 각 분기별 테스트
// =============================================================================

func TestLexIdent_SimpleLetters(t *testing.T) {
	tokens, err := Tokenize("hello")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "hello"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_WithDigits(t *testing.T) {
	tokens, err := Tokenize("node1")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "node1"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_WithUnderscore(t *testing.T) {
	tokens, err := Tokenize("my_node")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "my_node"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_WithHyphen(t *testing.T) {
	tokens, err := Tokenize("my-node")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "my-node"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_MixedChars(t *testing.T) {
	tokens, err := Tokenize("my-node_v2")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "my-node_v2"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_FlushAtEOF(t *testing.T) {
	// Identifier right at end of input should be flushed
	tokens, err := Tokenize("abc")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Type != TokenIdent || tokens[0].Value != "abc" {
		t.Errorf("expected Ident 'abc', got %v", tokens[0])
	}
}

func TestLexIdent_HyphenThenArrow(t *testing.T) {
	// `abc->` should produce Ident("abc"), Arrow("->")
	tokens, err := Tokenize("abc->def")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "abc"},
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenIdent, Value: "def"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_HyphenedIdentThenArrow(t *testing.T) {
	tokens, err := Tokenize("my-node -> other")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "my-node"},
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenIdent, Value: "other"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_TerminatedByColon(t *testing.T) {
	tokens, err := Tokenize("node:")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_TerminatedByBrace(t *testing.T) {
	tokens, err := Tokenize("node{")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_TerminatedByDot(t *testing.T) {
	tokens, err := Tokenize("a.b")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "a"},
		Token{Type: TokenDot, Value: "."},
		Token{Type: TokenIdent, Value: "b"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_TerminatedBySpace(t *testing.T) {
	tokens, err := Tokenize("a b")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "a"},
		Token{Type: TokenIdent, Value: "b"},
		Token{Type: TokenEOF},
	)
}

func TestLexIdent_TerminatedByNewline(t *testing.T) {
	tokens, err := Tokenize("abc\n")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "abc"},
		Token{Type: TokenNewline},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// lexString — 각 분기별 테스트
// =============================================================================

func TestLexString_Empty(t *testing.T) {
	tokens, err := Tokenize(`""`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: ""},
		Token{Type: TokenEOF},
	)
}

func TestLexString_Simple(t *testing.T) {
	tokens, err := Tokenize(`"hello"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "hello"},
		Token{Type: TokenEOF},
	)
}

func TestLexString_WithSpaces(t *testing.T) {
	tokens, err := Tokenize(`"hello world"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "hello world"},
		Token{Type: TokenEOF},
	)
}

func TestLexString_EscapedQuote(t *testing.T) {
	tokens, err := Tokenize(`"say \"hi\""`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: `say "hi"`},
		Token{Type: TokenEOF},
	)
}

func TestLexString_SpecialChars(t *testing.T) {
	tokens, err := Tokenize(`"#@{}: ->"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "#@{}: ->"},
		Token{Type: TokenEOF},
	)
}

func TestLexString_WithNewlineInside(t *testing.T) {
	// Newline inside quotes should be included in the string value
	tokens, err := Tokenize("\"line1\nline2\"")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "line1\nline2"},
		Token{Type: TokenEOF},
	)
}

func TestLexString_Unterminated(t *testing.T) {
	tokens, err := Tokenize(`"no end`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
	// Should still emit the partial string token
	if len(tokens) < 1 || tokens[0].Type != TokenString {
		t.Errorf("expected partial String token, got %v", tokens)
	}
	if tokens[0].Value != "no end" {
		t.Errorf("expected value 'no end', got %q", tokens[0].Value)
	}
}

func TestLexString_UnterminatedEmpty(t *testing.T) {
	tokens, err := Tokenize(`"`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
	if tokens[0].Type != TokenString || tokens[0].Value != "" {
		t.Errorf("expected empty String token, got %v", tokens[0])
	}
}

func TestLexString_FollowedByToken(t *testing.T) {
	tokens, err := Tokenize(`"hello" node`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenString, Value: "hello"},
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// lexComment — 각 분기별 테스트
// =============================================================================

func TestLexComment_ToEndOfLine(t *testing.T) {
	tokens, err := Tokenize("# this is a comment\nnode")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: " this is a comment"},
		Token{Type: TokenNewline},
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenEOF},
	)
}

func TestLexComment_AtEOF(t *testing.T) {
	// Comment at EOF (no trailing newline) should be flushed
	tokens, err := Tokenize("# comment at eof")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: " comment at eof"},
		Token{Type: TokenEOF},
	)
}

func TestLexComment_Empty(t *testing.T) {
	// `#` followed immediately by newline
	tokens, err := Tokenize("#\nnode")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: ""},
		Token{Type: TokenNewline},
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenEOF},
	)
}

func TestLexComment_EmptyAtEOF(t *testing.T) {
	tokens, err := Tokenize("#")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: ""},
		Token{Type: TokenEOF},
	)
}

func TestLexComment_WithSpecialChars(t *testing.T) {
	tokens, err := Tokenize(`# {}: -> "quoted" @meta`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: ` {}: -> "quoted" @meta`},
		Token{Type: TokenEOF},
	)
}

func TestLexComment_MultipleLines(t *testing.T) {
	tokens, err := Tokenize("# line 1\n# line 2")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenComment, Value: " line 1"},
		Token{Type: TokenNewline},
		Token{Type: TokenComment, Value: " line 2"},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// lexNumber — 각 분기별 테스트
// =============================================================================

func TestLexNumber_SingleDigit(t *testing.T) {
	tokens, err := Tokenize("0")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "0"},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_Integer(t *testing.T) {
	tokens, err := Tokenize("42")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "42"},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_Float(t *testing.T) {
	tokens, err := Tokenize("3.14")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "3.14"},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_AtEOF(t *testing.T) {
	tokens, err := Tokenize("999")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Type != TokenNumber || tokens[0].Value != "999" {
		t.Errorf("expected Number '999', got %v", tokens[0])
	}
}

func TestLexNumber_TerminatedBySpace(t *testing.T) {
	tokens, err := Tokenize("42 abc")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "42"},
		Token{Type: TokenIdent, Value: "abc"},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_TerminatedByComma(t *testing.T) {
	tokens, err := Tokenize("100,")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "100"},
		Token{Type: TokenComma, Value: ","},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_TerminatedByBrace(t *testing.T) {
	tokens, err := Tokenize("100}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "100"},
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_TerminatedByNewline(t *testing.T) {
	tokens, err := Tokenize("100\n")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "100"},
		Token{Type: TokenNewline},
		Token{Type: TokenEOF},
	)
}

func TestLexNumber_MultipleNumbers(t *testing.T) {
	tokens, err := Tokenize("1 2 3")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenNumber, Value: "1"},
		Token{Type: TokenNumber, Value: "2"},
		Token{Type: TokenNumber, Value: "3"},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// lexRawBlock — 각 분기별 테스트
// =============================================================================

func TestLexRawBlock_Simple(t *testing.T) {
	tokens, err := Tokenize("{| hello |}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "hello"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_Multiline(t *testing.T) {
	tokens, err := Tokenize("{|\n  line1\n  line2\n|}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "line1\nline2"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_WithBraces(t *testing.T) {
	// Unmatched braces inside raw block should be fine
	tokens, err := Tokenize("{| function() { |}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "function() {"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_WithClosingBrace(t *testing.T) {
	// Standalone } inside raw block should NOT terminate it
	tokens, err := Tokenize("{| value } more |}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "value } more"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_Markdown(t *testing.T) {
	input := "{|\n  # Header\n  - list item\n  **bold**\n|}"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Type != TokenRawBlock {
		t.Fatalf("expected RawBlock, got %s", tokens[0].Type)
	}
	if tokens[0].Value == "" {
		t.Error("expected non-empty raw block value")
	}
}

func TestLexRawBlock_Latex(t *testing.T) {
	input := "{| \\frac{a+b}{c} |}"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "\\frac{a+b}{c}"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_Empty(t *testing.T) {
	tokens, err := Tokenize("{| |}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: ""},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_Unterminated(t *testing.T) {
	tokens, err := Tokenize("{| never closed")
	if err == nil {
		t.Fatal("expected error for unterminated raw block")
	}
	if tokens[0].Type != TokenRawBlock {
		t.Fatalf("expected partial RawBlock token, got %s", tokens[0].Type)
	}
	if tokens[0].Value != "never closed" {
		t.Errorf("expected 'never closed', got %q", tokens[0].Value)
	}
}

func TestLexRawBlock_FollowedByTokens(t *testing.T) {
	tokens, err := Tokenize("{| content |} node")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "content"},
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_PipeAlone(t *testing.T) {
	// | without } should NOT end the raw block
	tokens, err := Tokenize("{| a | b |}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenRawBlock, Value: "a | b"},
		Token{Type: TokenEOF},
	)
}

func TestLexRawBlock_BraceAlone(t *testing.T) {
	// { without preceding | should NOT start a raw block — it's a normal LBrace
	tokens, err := Tokenize("{ }")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// Position tracking
// =============================================================================

func TestLexPosition_LineTracking(t *testing.T) {
	tokens, err := Tokenize("a\nb\nc")
	if err != nil {
		t.Fatal(err)
	}
	// a => L1, \n => L1, b => L2, \n => L2, c => L3
	if tokens[0].Line != 1 {
		t.Errorf("token 'a': expected line 1, got %d", tokens[0].Line)
	}
	if tokens[2].Line != 2 {
		t.Errorf("token 'b': expected line 2, got %d", tokens[2].Line)
	}
	if tokens[4].Line != 3 {
		t.Errorf("token 'c': expected line 3, got %d", tokens[4].Line)
	}
}

func TestLexPosition_ColumnTracking(t *testing.T) {
	tokens, err := Tokenize("ab cd")
	if err != nil {
		t.Fatal(err)
	}
	// "ab" starts at col 1, "cd" starts at col 4
	if tokens[0].Col != 1 {
		t.Errorf("token 'ab': expected col 1, got %d", tokens[0].Col)
	}
	if tokens[1].Col != 4 {
		t.Errorf("token 'cd': expected col 4, got %d", tokens[1].Col)
	}
}

func TestLexPosition_AfterNewline(t *testing.T) {
	tokens, err := Tokenize("a\n  b")
	if err != nil {
		t.Fatal(err)
	}
	// 'b' should be at line 2, col 3
	bTok := tokens[2] // a, \n, b
	if bTok.Line != 2 || bTok.Col != 3 {
		t.Errorf("expected 'b' at L2:C3, got L%d:C%d", bTok.Line, bTok.Col)
	}
}

// =============================================================================
// Error collection (lexer continues on errors)
// =============================================================================

func TestLexError_UnterminatedStringEmitsPartial(t *testing.T) {
	// String that never closes — lexer should emit partial string and report error
	tokens, err := Tokenize(`"never closed`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
	if tokens[0].Type != TokenString || tokens[0].Value != "never closed" {
		t.Errorf("expected partial String token, got %v", tokens[0])
	}
}

func TestLexError_ContinuesAfterUnterminated(t *testing.T) {
	// Two separate unterminated strings — lexer only sees one because
	// everything after the first `"` becomes the string content up to EOF.
	tokens, err := Tokenize(`"unterminated`)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should still produce a partial String + EOF
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens (String + EOF), got %d: %v", len(tokens), tokens)
	}
}

// =============================================================================
// Complex / combined token sequences
// =============================================================================

func TestLex_NodeWithLabel(t *testing.T) {
	tokens, err := Tokenize(`node: "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenString, Value: "hello"},
		Token{Type: TokenEOF},
	)
}

func TestLex_EdgeWithArrow(t *testing.T) {
	tokens, err := Tokenize(`A -> B`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "A"},
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenIdent, Value: "B"},
		Token{Type: TokenEOF},
	)
}

func TestLex_EdgeWithLabel(t *testing.T) {
	tokens, err := Tokenize(`A -> B: "label"`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "A"},
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenIdent, Value: "B"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenString, Value: "label"},
		Token{Type: TokenEOF},
	)
}

func TestLex_DottedPath(t *testing.T) {
	tokens, err := Tokenize(`a.b.c`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "a"},
		Token{Type: TokenDot, Value: "."},
		Token{Type: TokenIdent, Value: "b"},
		Token{Type: TokenDot, Value: "."},
		Token{Type: TokenIdent, Value: "c"},
		Token{Type: TokenEOF},
	)
}

func TestLex_BlockSyntax(t *testing.T) {
	tokens, err := Tokenize("node: {\n}")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "node"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenNewline},
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenEOF},
	)
}

func TestLex_MetadataLayout(t *testing.T) {
	tokens, err := Tokenize(`@layout {x: 100, y: 200}`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenAt, Value: "@"},
		Token{Type: TokenIdent, Value: "layout"},
		Token{Type: TokenLBrace, Value: "{"},
		Token{Type: TokenIdent, Value: "x"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenNumber, Value: "100"},
		Token{Type: TokenComma, Value: ","},
		Token{Type: TokenIdent, Value: "y"},
		Token{Type: TokenColon, Value: ":"},
		Token{Type: TokenNumber, Value: "200"},
		Token{Type: TokenRBrace, Value: "}"},
		Token{Type: TokenEOF},
	)
}

func TestLex_TextBracketFormat(t *testing.T) {
	tokens, err := Tokenize(`@text[markdown] {| content |}`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenAt, Value: "@"},
		Token{Type: TokenIdent, Value: "text"},
		Token{Type: TokenLBracket, Value: "["},
		Token{Type: TokenIdent, Value: "markdown"},
		Token{Type: TokenRBracket, Value: "]"},
		Token{Type: TokenRawBlock, Value: "content"},
		Token{Type: TokenEOF},
	)
}

func TestLex_ArraySyntax(t *testing.T) {
	tokens, err := Tokenize(`["right", "left"]`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBracket, Value: "["},
		Token{Type: TokenString, Value: "right"},
		Token{Type: TokenComma, Value: ","},
		Token{Type: TokenString, Value: "left"},
		Token{Type: TokenRBracket, Value: "]"},
		Token{Type: TokenEOF},
	)
}

func TestLex_NestedObjects(t *testing.T) {
	tokens, err := Tokenize(`{a: {b: 1}}`)
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenLBrace},
		Token{Type: TokenIdent, Value: "a"},
		Token{Type: TokenColon},
		Token{Type: TokenLBrace},
		Token{Type: TokenIdent, Value: "b"},
		Token{Type: TokenColon},
		Token{Type: TokenNumber, Value: "1"},
		Token{Type: TokenRBrace},
		Token{Type: TokenRBrace},
		Token{Type: TokenEOF},
	)
}

func TestLex_HyphenedIdentBeforeArrow(t *testing.T) {
	tokens, err := Tokenize("my-node->other-node")
	if err != nil {
		t.Fatal(err)
	}
	expectTokens(t, tokens,
		Token{Type: TokenIdent, Value: "my-node"},
		Token{Type: TokenArrow, Value: "->"},
		Token{Type: TokenIdent, Value: "other-node"},
		Token{Type: TokenEOF},
	)
}

// =============================================================================
// TokenType.String()
// =============================================================================

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		tt   TokenType
		want string
	}{
		{TokenEOF, "EOF"},
		{TokenNewline, "Newline"},
		{TokenIdent, "Ident"},
		{TokenDot, "Dot"},
		{TokenColon, "Colon"},
		{TokenArrow, "Arrow"},
		{TokenLBrace, "LBrace"},
		{TokenRBrace, "RBrace"},
		{TokenLBracket, "LBracket"},
		{TokenRBracket, "RBracket"},
		{TokenAt, "At"},
		{TokenString, "String"},
		{TokenNumber, "Number"},
		{TokenComma, "Comma"},
		{TokenComment, "Comment"},
		{TokenHash, "Hash"},
		{TokenRawBlock, "RawBlock"},
		{TokenType(999), "TokenType(999)"},
	}
	for _, tc := range tests {
		got := tc.tt.String()
		if got != tc.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", int(tc.tt), got, tc.want)
		}
	}
}

// =============================================================================
// Token.String()
// =============================================================================

func TestToken_String(t *testing.T) {
	tok := Token{Type: TokenIdent, Value: "abc", Line: 3, Col: 5}
	s := tok.String()
	if s != `{Ident "abc" L3:C5}` {
		t.Errorf("Token.String() = %q, unexpected", s)
	}
}
