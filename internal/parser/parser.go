package parser

import (
	"strconv"
	"strings"
)

// Parser converts a token stream into an AST using a state machine.
type Parser struct {
	tokens []Token
	pos    int
	doc    *Document
	errors *MultiParseError
	stack  []*blockContext

	// Scratch buffers for state communication
	pathBuf       []string
	pathLine      int
	edgeFrom      []string
	edgeTo        []string
	edgeDirection string
	metaLine      int
}

// Parse is the public entry point. It tokenizes input and builds an AST.
// Returns a partial Document even on error (best-effort parsing).
func Parse(input string) (*Document, error) {
	tokens, lexErr := Tokenize(input)

	p := &Parser{
		tokens: tokens,
		doc:    &Document{},
		errors: &MultiParseError{},
	}

	if lexErr != nil {
		if me, ok := lexErr.(*MultiParseError); ok {
			p.errors.Errors = append(p.errors.Errors, me.Errors...)
		} else if pe, ok := lexErr.(*ParseError); ok {
			p.errors.Errors = append(p.errors.Errors, pe)
		}
	}

	state := ParseState(stateTopLevel{})
	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		p.pos++
		var err error
		state, err = state.Handle(p, tok)
		if err != nil {
			if pe, ok := err.(*ParseError); ok {
				p.errors.Errors = append(p.errors.Errors, pe)
			}
		}
	}

	// Close any unclosed blocks
	for len(p.stack) > 0 {
		ctx := p.stack[len(p.stack)-1]
		p.addError(ctx.block.Line, 0, "unclosed block")
		p.stack = p.stack[:len(p.stack)-1]
	}

	if p.errors.HasErrors() {
		return p.doc, p.errors
	}
	return p.doc, nil
}

func (p *Parser) peek() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) advance() Token {
	if p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		p.pos++
		return tok
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) addError(line, col int, msg string) {
	p.errors.Add(line, col, msg)
}

func (p *Parser) addStatement(s Statement) {
	if len(p.stack) > 0 {
		ctx := p.stack[len(p.stack)-1]
		ctx.block.Children = append(ctx.block.Children, s)
	} else {
		p.doc.Statements = append(p.doc.Statements, s)
	}
}

func (p *Parser) addMetadata(m Metadata) {
	if len(p.stack) > 0 {
		ctx := p.stack[len(p.stack)-1]
		ctx.block.Metadata = append(ctx.block.Metadata, m)
	} else {
		p.addError(m.GetLine(), 0, "metadata directive outside of block")
	}
}

func (p *Parser) pushBlock(block *Block, nodePath []string, edgeFrom, edgeTo []string) {
	ctx := &blockContext{
		block:    block,
		nodePath: nodePath,
		fromPath: edgeFrom,
		toPath:   edgeTo,
		isEdge:   edgeFrom != nil,
	}
	p.stack = append(p.stack, ctx)
}

func (p *Parser) popBlock() {
	if len(p.stack) > 0 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}

// lastIsBlankLine reports whether the last statement added is a BlankLine,
// or whether there are no statements yet (to prevent leading blank lines).
// Used to collapse consecutive blank lines to one.
func (p *Parser) lastIsBlankLine() bool {
	var stmts []Statement
	if len(p.stack) > 0 {
		stmts = p.stack[len(p.stack)-1].block.Children
	} else {
		stmts = p.doc.Statements
	}
	if len(stmts) == 0 {
		return true // treat empty as "already blank" to prevent leading blank lines
	}
	_, ok := stmts[len(stmts)-1].(*BlankLine)
	return ok
}

// parseJSONObject parses a simplified JSON-like object (after the opening {).
// Supports: strings, numbers, nested objects, arrays.
func (p *Parser) parseJSONObject() map[string]interface{} {
	result := make(map[string]interface{})

	for p.pos < len(p.tokens) {
		tok := p.peek()

		switch tok.Type {
		case TokenRBrace:
			p.advance()
			return result

		case TokenNewline, TokenComma:
			p.advance()
			continue

		case TokenEOF:
			p.addError(tok.Line, tok.Col, "unterminated object")
			return result

		case TokenIdent, TokenString:
			key := tok.Value
			p.advance()

			// Expect colon
			next := p.peek()
			if next.Type == TokenColon {
				p.advance()
				value := p.parseJSONValue()
				result[key] = value
			} else {
				// Key without value — treat as boolean true
				result[key] = true
			}

		default:
			p.advance() // skip unexpected tokens
		}
	}
	return result
}

// parseJSONValue parses a single value in a JSON-like context.
func (p *Parser) parseJSONValue() interface{} {
	tok := p.peek()

	switch tok.Type {
	case TokenString:
		p.advance()
		return tok.Value

	case TokenNumber:
		p.advance()
		if strings.Contains(tok.Value, ".") {
			f, err := strconv.ParseFloat(tok.Value, 64)
			if err != nil {
				return tok.Value
			}
			return f
		}
		n, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			return tok.Value
		}
		return n

	case TokenIdent:
		p.advance()
		switch tok.Value {
		case "true":
			return true
		case "false":
			return false
		case "null", "nil":
			return nil
		default:
			return tok.Value
		}

	case TokenLBrace:
		p.advance()
		return p.parseJSONObject()

	case TokenLBracket:
		p.advance()
		return p.parseJSONArray()

	default:
		p.advance()
		return tok.Value
	}
}

// parseJSONArray parses a JSON-like array (after the opening [).
func (p *Parser) parseJSONArray() []interface{} {
	var result []interface{}

	for p.pos < len(p.tokens) {
		tok := p.peek()
		switch tok.Type {
		case TokenRBracket:
			p.advance()
			return result
		case TokenComma, TokenNewline:
			p.advance()
			continue
		case TokenEOF:
			p.addError(tok.Line, tok.Col, "unterminated array")
			return result
		default:
			result = append(result, p.parseJSONValue())
		}
	}
	return result
}
