package parser

import "fmt"

// ParseState is the state interface for the parser state machine.
type ParseState interface {
	Handle(p *Parser, tok Token) (ParseState, error)
}

// blockContext tracks the current { } nesting level.
type blockContext struct {
	block    *Block
	isEdge   bool     // true if this block belongs to an EdgeDecl
	fromPath []string // for edges
	toPath   []string // for edges
	label    *TextValue
	nodePath []string // for nodes
}

// --- stateTopLevel ---
// Entry state: expects Ident (node/edge), @, }, Comment, Newline, EOF.
type stateTopLevel struct {
	prevNewline bool // true if the last token was a newline (for blank line detection)
}

func (s stateTopLevel) Handle(p *Parser, tok Token) (ParseState, error) {
	switch tok.Type {
	case TokenNewline:
		if s.prevNewline && !p.lastIsBlankLine() {
			p.addStatement(&BlankLine{Line: tok.Line})
		}
		return stateTopLevel{prevNewline: true}, nil

	case TokenComment:
		p.addStatement(&Comment{Line: tok.Line, Text: tok.Value})
		return stateTopLevel{prevNewline: false}, nil

	case TokenEOF:
		return stateTopLevel{}, nil

	case TokenRBrace:
		if len(p.stack) == 0 {
			p.addError(tok.Line, tok.Col, "unexpected '}'")
			return stateTopLevel{}, nil
		}
		p.popBlock()
		return stateTopLevel{}, nil

	case TokenIdent:
		p.pathBuf = p.pathBuf[:0]
		p.pathBuf = append(p.pathBuf, tok.Value)
		p.pathLine = tok.Line
		return stateInPath{}, nil

	case TokenAt:
		p.metaLine = tok.Line
		return stateMetaKey{}, nil

	default:
		p.addError(tok.Line, tok.Col, fmt.Sprintf("unexpected token %s", tok.Type))
		return stateTopLevel{}, nil
	}
}

// --- stateInPath ---
// We've seen an Ident. Consume dots to build a full path.
type stateInPath struct{}

func (stateInPath) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type == TokenDot {
		return statePathDot{}, nil
	}
	// Path is complete — hand off to stateAfterIdent
	return stateAfterIdent{}.Handle(p, tok)
}

// statePathDot expects an Ident after a dot.
type statePathDot struct{}

func (statePathDot) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type == TokenIdent {
		p.pathBuf = append(p.pathBuf, tok.Value)
		return stateInPath{}, nil
	}
	p.addError(tok.Line, tok.Col, "expected identifier after '.'")
	return stateTopLevel{}, nil
}

// --- stateAfterIdent ---
// Path complete. Expects -> (edge), : (label/block), Newline/{/EOF (bare node).
type stateAfterIdent struct{}

func (stateAfterIdent) Handle(p *Parser, tok Token) (ParseState, error) {
	switch tok.Type {
	case TokenArrow:
		p.edgeFrom = copyPath(p.pathBuf)
		p.edgeDirection = ""
		return stateEdgeTarget{}, nil

	case TokenReverseArrow:
		p.edgeFrom = copyPath(p.pathBuf)
		p.edgeDirection = "reverse"
		return stateEdgeTarget{}, nil

	case TokenBiArrow:
		p.edgeFrom = copyPath(p.pathBuf)
		p.edgeDirection = "bidirectional"
		return stateEdgeTarget{}, nil

	case TokenColon:
		return stateAfterColon{
			path:   copyPath(p.pathBuf),
			line:   p.pathLine,
			isEdge: false,
		}, nil

	case TokenLBrace:
		path := copyPath(p.pathBuf)
		block := &Block{Line: tok.Line}
		node := &NodeDecl{Line: p.pathLine, Path: path, Block: block}
		p.addStatement(node)
		p.pushBlock(block, path, nil, nil)
		return stateTopLevel{}, nil

	case TokenNewline, TokenEOF, TokenComment:
		p.addStatement(&NodeDecl{Line: p.pathLine, Path: copyPath(p.pathBuf)})
		if tok.Type == TokenComment {
			p.addStatement(&Comment{Line: tok.Line, Text: tok.Value})
		}
		return stateTopLevel{prevNewline: tok.Type == TokenNewline}, nil

	default:
		p.addError(tok.Line, tok.Col, fmt.Sprintf("unexpected token %s after identifier", tok.Type))
		return stateTopLevel{}, nil
	}
}

// --- stateEdgeTarget ---
// After ->. Expects an Ident path for the target.
type stateEdgeTarget struct{}

func (stateEdgeTarget) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type == TokenIdent {
		p.edgeTo = p.edgeTo[:0]
		p.edgeTo = append(p.edgeTo, tok.Value)
		return stateEdgeTargetPath{}, nil
	}
	p.addError(tok.Line, tok.Col, "expected identifier after '->'")
	return stateTopLevel{}, nil
}

type stateEdgeTargetPath struct{}

func (stateEdgeTargetPath) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type == TokenDot {
		return stateEdgeTargetDot{}, nil
	}
	return stateAfterEdgeTarget{}.Handle(p, tok)
}

type stateEdgeTargetDot struct{}

func (stateEdgeTargetDot) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type == TokenIdent {
		p.edgeTo = append(p.edgeTo, tok.Value)
		return stateEdgeTargetPath{}, nil
	}
	p.addError(tok.Line, tok.Col, "expected identifier after '.'")
	return stateTopLevel{}, nil
}

// --- stateAfterEdgeTarget ---
// Edge target path complete. Expects : (label/block), Newline/EOF (bare edge), { (block).
type stateAfterEdgeTarget struct{}

func (stateAfterEdgeTarget) Handle(p *Parser, tok Token) (ParseState, error) {
	from := copyPath(p.edgeFrom)
	to := copyPath(p.edgeTo)
	dir := p.edgeDirection

	switch tok.Type {
	case TokenColon:
		return stateAfterColon{
			line:          p.pathLine,
			isEdge:        true,
			edgeFrom:      from,
			edgeTo:        to,
			edgeDirection: dir,
		}, nil

	case TokenLBrace:
		block := &Block{Line: tok.Line}
		edge := &EdgeDecl{Line: p.pathLine, From: from, To: to, Direction: dir, Block: block}
		p.addStatement(edge)
		p.pushBlock(block, nil, from, to)
		return stateTopLevel{}, nil

	case TokenNewline, TokenEOF, TokenComment:
		p.addStatement(&EdgeDecl{Line: p.pathLine, From: from, To: to, Direction: dir})
		if tok.Type == TokenComment {
			p.addStatement(&Comment{Line: tok.Line, Text: tok.Value})
		}
		return stateTopLevel{prevNewline: tok.Type == TokenNewline}, nil

	default:
		p.addError(tok.Line, tok.Col, fmt.Sprintf("unexpected token %s after edge target", tok.Type))
		return stateTopLevel{}, nil
	}
}

// --- stateAfterColon ---
// After a colon on a node or edge. Expects String, {, or rest-of-line text.
type stateAfterColon struct {
	path          []string
	line          int
	isEdge        bool
	edgeFrom      []string
	edgeTo        []string
	edgeDirection string
}

func (s stateAfterColon) Handle(p *Parser, tok Token) (ParseState, error) {
	switch tok.Type {
	case TokenString:
		label := &TextValue{Format: "plain", Value: tok.Value}
		if s.isEdge {
			p.addStatement(&EdgeDecl{Line: s.line, From: s.edgeFrom, To: s.edgeTo, Direction: s.edgeDirection, Label: label})
		} else {
			p.addStatement(&NodeDecl{Line: s.line, Path: s.path, Label: label})
		}
		return stateTopLevel{}, nil

	case TokenLBrace:
		block := &Block{Line: tok.Line}
		if s.isEdge {
			edge := &EdgeDecl{Line: s.line, From: s.edgeFrom, To: s.edgeTo, Direction: s.edgeDirection, Block: block}
			p.addStatement(edge)
			p.pushBlock(block, nil, s.edgeFrom, s.edgeTo)
		} else {
			node := &NodeDecl{Line: s.line, Path: s.path, Block: block}
			p.addStatement(node)
			p.pushBlock(block, s.path, nil, nil)
		}
		return stateTopLevel{}, nil

	case TokenNewline, TokenEOF:
		// colon with nothing after it
		if s.isEdge {
			p.addStatement(&EdgeDecl{Line: s.line, From: s.edgeFrom, To: s.edgeTo, Direction: s.edgeDirection})
		} else {
			p.addStatement(&NodeDecl{Line: s.line, Path: s.path})
		}
		return stateTopLevel{prevNewline: tok.Type == TokenNewline}, nil

	default:
		// Rest-of-line text as label (e.g., `A -> B: Read`)
		text := tok.Value
		for {
			next := p.peek()
			if next.Type == TokenNewline || next.Type == TokenEOF || next.Type == TokenComment {
				break
			}
			next = p.advance()
			text += " " + next.Value
		}
		label := &TextValue{Format: "plain", Value: text}
		if s.isEdge {
			p.addStatement(&EdgeDecl{Line: s.line, From: s.edgeFrom, To: s.edgeTo, Direction: s.edgeDirection, Label: label})
		} else {
			p.addStatement(&NodeDecl{Line: s.line, Path: s.path, Label: label})
		}
		return stateTopLevel{}, nil
	}
}

// --- stateMetaKey ---
// After @. Expects Ident (layout, style, shape, text, edge, icon).
type stateMetaKey struct{}

func (stateMetaKey) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type != TokenIdent {
		p.addError(tok.Line, tok.Col, "expected metadata keyword after '@'")
		return stateTopLevel{}, nil
	}

	switch tok.Value {
	case "text":
		return stateTextMeta{line: p.metaLine}, nil
	case "shape":
		return stateSimpleMeta{kind: "shape", line: p.metaLine}, nil
	case "layout", "style", "edge", "icon":
		return stateObjectMeta{kind: tok.Value, line: p.metaLine}, nil
	default:
		p.addError(tok.Line, tok.Col, fmt.Sprintf("unknown metadata directive @%s", tok.Value))
		return stateTopLevel{}, nil
	}
}

// --- stateTextMeta ---
// After @text. May have [format] bracket, then value or { raw block }.
type stateTextMeta struct {
	line   int
	format string
}

func (s stateTextMeta) Handle(p *Parser, tok Token) (ParseState, error) {
	if s.format == "" {
		s.format = "plain"
	}

	switch tok.Type {
	case TokenLBracket:
		// @text[format]
		return stateTextFormat{line: s.line}, nil

	case TokenRawBlock:
		// @text {| raw block |}
		p.addMetadata(&TextMeta{Line: s.line, Format: s.format, Value: tok.Value})
		return stateTopLevel{}, nil

	case TokenString:
		p.addMetadata(&TextMeta{Line: s.line, Format: s.format, Value: tok.Value})
		return stateTopLevel{}, nil

	case TokenNewline, TokenEOF:
		// @text with no value
		p.addMetadata(&TextMeta{Line: s.line, Format: s.format, Value: ""})
		return stateTopLevel{prevNewline: tok.Type == TokenNewline}, nil

	default:
		// Rest-of-line text: @text hello world
		text := tok.Value
		for {
			next := p.peek()
			if next.Type == TokenNewline || next.Type == TokenEOF || next.Type == TokenComment {
				break
			}
			next = p.advance()
			text += " " + next.Value
		}
		p.addMetadata(&TextMeta{Line: s.line, Format: s.format, Value: text})
		return stateTopLevel{}, nil
	}
}

// stateTextFormat reads the format name inside [].
type stateTextFormat struct {
	line int
}

func (s stateTextFormat) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type != TokenIdent {
		p.addError(tok.Line, tok.Col, "expected format name in @text[...]")
		return stateTopLevel{}, nil
	}
	format := tok.Value
	// Expect closing bracket
	next := p.advance()
	if next.Type != TokenRBracket {
		p.addError(next.Line, next.Col, "expected ']' after format name")
		return stateTopLevel{}, nil
	}
	return stateTextMeta{line: s.line, format: format}, nil
}

// --- stateSimpleMeta ---
// For @shape: expects a single value (rest of line).
type stateSimpleMeta struct {
	kind string
	line int
}

func (s stateSimpleMeta) Handle(p *Parser, tok Token) (ParseState, error) {
	switch tok.Type {
	case TokenNewline, TokenEOF:
		p.addError(tok.Line, tok.Col, fmt.Sprintf("expected value for @%s", s.kind))
		return stateTopLevel{}, nil
	default:
		value := tok.Value
		for {
			next := p.peek()
			if next.Type == TokenNewline || next.Type == TokenEOF || next.Type == TokenComment {
				break
			}
			next = p.advance()
			value += " " + next.Value
		}
		p.addMetadata(&ShapeMeta{Line: s.line, Value: value})
		return stateTopLevel{}, nil
	}
}

// --- stateObjectMeta ---
// For @layout, @style, @edge, @icon: expects { JSON-like object }.
type stateObjectMeta struct {
	kind string
	line int
}

func (s stateObjectMeta) Handle(p *Parser, tok Token) (ParseState, error) {
	if tok.Type != TokenLBrace {
		p.addError(tok.Line, tok.Col, fmt.Sprintf("expected '{' after @%s", s.kind))
		// Try to recover by skipping to newline
		for {
			next := p.peek()
			if next.Type == TokenNewline || next.Type == TokenEOF {
				break
			}
			p.advance()
		}
		return stateTopLevel{}, nil
	}

	values := p.parseJSONObject()

	switch s.kind {
	case "layout":
		p.addMetadata(&LayoutMeta{Line: s.line, Values: values})
	case "style":
		p.addMetadata(&StyleMeta{Line: s.line, Values: values})
	case "edge":
		p.addMetadata(&EdgeMeta{Line: s.line, Values: values})
	case "icon":
		p.addMetadata(&IconMeta{Line: s.line, Values: values})
	}
	return stateTopLevel{}, nil
}

// --- helpers ---

func copyPath(p []string) []string {
	c := make([]string, len(p))
	copy(c, p)
	return c
}
