package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType identifies the kind of token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNewline
	TokenIdent
	TokenDot
	TokenColon
	TokenArrow // ->
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenAt
	TokenString
	TokenNumber
	TokenComma
	TokenComment
	TokenHash
	TokenRawBlock // {| ... |}
)

var tokenNames = map[TokenType]string{
	TokenEOF:      "EOF",
	TokenNewline:  "Newline",
	TokenIdent:    "Ident",
	TokenDot:      "Dot",
	TokenColon:    "Colon",
	TokenArrow:    "Arrow",
	TokenLBrace:   "LBrace",
	TokenRBrace:   "RBrace",
	TokenLBracket: "LBracket",
	TokenRBracket: "RBracket",
	TokenAt:       "At",
	TokenString:   "String",
	TokenNumber:   "Number",
	TokenComma:    "Comma",
	TokenComment:  "Comment",
	TokenHash:     "Hash",
	TokenRawBlock: "RawBlock",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token is a lexical token with type, value, and position.
type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

func (t Token) String() string {
	return fmt.Sprintf("{%s %q L%d:C%d}", t.Type, t.Value, t.Line, t.Col)
}

// LexState is the state interface for the lexer state machine.
type LexState interface {
	Handle(l *Lexer, ch rune) (LexState, error)
}

// Lexer tokenizes .grim source text using a state machine.
type Lexer struct {
	input  []rune
	pos    int
	line   int
	col    int
	tokens []Token
	buf    []rune
	start  struct{ line, col int }
	errors []*ParseError
}

// Tokenize converts source text into a slice of tokens.
// It is permissive: unknown characters are emitted as Ident tokens and
// errors are collected rather than stopping lexing.
func Tokenize(input string) ([]Token, error) {
	l := &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
		col:   1,
	}

	state := LexState(lexDefault{})
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		var err error
		state, err = state.Handle(l, ch)
		if err != nil {
			if pe, ok := err.(*ParseError); ok {
				l.errors = append(l.errors, pe)
			}
			// Continue lexing — don't stop on errors
		}
	}
	// Flush any remaining state
	var flushErr error
	state, flushErr = state.Handle(l, 0) // sentinel EOF
	if flushErr != nil {
		if pe, ok := flushErr.(*ParseError); ok {
			l.errors = append(l.errors, pe)
		}
	}
	l.emit(TokenEOF, "")

	if len(l.errors) > 0 {
		me := &MultiParseError{Errors: l.errors}
		return l.tokens, me
	}
	return l.tokens, nil
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) peek() rune {
	if l.pos < len(l.input) {
		return l.input[l.pos]
	}
	return 0
}

func (l *Lexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx < len(l.input) {
		return l.input[idx]
	}
	return 0
}

func (l *Lexer) emit(typ TokenType, val string) {
	l.tokens = append(l.tokens, Token{
		Type:  typ,
		Value: val,
		Line:  l.start.line,
		Col:   l.start.col,
	})
}

func (l *Lexer) markStart() {
	l.start.line = l.line
	l.start.col = l.col
}

func (l *Lexer) bufReset() {
	l.buf = l.buf[:0]
}

func (l *Lexer) bufString() string {
	return string(l.buf)
}

// --- Lex States ---

// lexDefault skips whitespace and dispatches to other states.
type lexDefault struct{}

func (lexDefault) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 { // EOF sentinel
		return lexDefault{}, nil
	}

	switch {
	case ch == '\n':
		l.markStart()
		l.advance()
		l.emit(TokenNewline, "\n")
		return lexDefault{}, nil

	case ch == ' ' || ch == '\t' || ch == '\r':
		l.advance()
		return lexDefault{}, nil

	case ch == '#':
		l.markStart()
		l.advance()
		l.bufReset()
		return lexComment{}, nil

	case ch == '"':
		l.markStart()
		l.advance() // consume opening quote
		l.bufReset()
		return lexString{}, nil

	case ch == '-' && l.peekAt(1) == '>':
		l.markStart()
		l.advance()
		l.advance()
		l.emit(TokenArrow, "->")
		return lexDefault{}, nil

	case ch == '{' && l.peekAt(1) == '|':
		l.markStart()
		l.advance() // consume {
		l.advance() // consume |
		l.bufReset()
		return lexRawBlock{}, nil
	case ch == '{':
		l.markStart()
		l.advance()
		l.emit(TokenLBrace, "{")
		return lexDefault{}, nil
	case ch == '}':
		l.markStart()
		l.advance()
		l.emit(TokenRBrace, "}")
		return lexDefault{}, nil
	case ch == '[':
		l.markStart()
		l.advance()
		l.emit(TokenLBracket, "[")
		return lexDefault{}, nil
	case ch == ']':
		l.markStart()
		l.advance()
		l.emit(TokenRBracket, "]")
		return lexDefault{}, nil
	case ch == ':':
		l.markStart()
		l.advance()
		l.emit(TokenColon, ":")
		return lexDefault{}, nil
	case ch == '.':
		l.markStart()
		l.advance()
		l.emit(TokenDot, ".")
		return lexDefault{}, nil
	case ch == '@':
		l.markStart()
		l.advance()
		l.emit(TokenAt, "@")
		return lexDefault{}, nil
	case ch == ',':
		l.markStart()
		l.advance()
		l.emit(TokenComma, ",")
		return lexDefault{}, nil

	case isIdentStart(ch):
		l.markStart()
		l.bufReset()
		l.buf = append(l.buf, ch)
		l.advance()
		return lexIdent{}, nil

	case isDigitStart(ch):
		l.markStart()
		l.bufReset()
		l.buf = append(l.buf, ch)
		l.advance()
		return lexNumber{}, nil

	default:
		// Emit unknown characters as single-character Ident tokens.
		// This allows raw block content (markdown, latex) to pass through.
		l.markStart()
		l.advance()
		l.emit(TokenIdent, string(ch))
		return lexDefault{}, nil
	}
}

// lexIdent consumes identifier characters.
type lexIdent struct{}

func (lexIdent) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 {
		l.emit(TokenIdent, l.bufString())
		return lexDefault{}, nil
	}
	// Hyphen followed by > means arrow — flush identifier before it.
	if ch == '-' && l.peekAt(1) == '>' {
		l.emit(TokenIdent, l.bufString())
		return lexDefault{}.Handle(l, ch)
	}
	if isIdentContinue(ch) {
		l.buf = append(l.buf, ch)
		l.advance()
		return lexIdent{}, nil
	}
	l.emit(TokenIdent, l.bufString())
	return lexDefault{}.Handle(l, ch)
}

// lexString consumes a quoted string (handles \" escape).
type lexString struct{}

func (lexString) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 {
		// Unterminated string
		l.emit(TokenString, l.bufString())
		return lexDefault{}, &ParseError{
			Line:    l.start.line,
			Col:     l.start.col,
			Message: "unterminated string",
		}
	}
	if ch == '\\' && l.peekAt(1) == '"' {
		l.buf = append(l.buf, '"')
		l.advance()
		l.advance()
		return lexString{}, nil
	}
	if ch == '"' {
		l.advance()
		l.emit(TokenString, l.bufString())
		return lexDefault{}, nil
	}
	l.buf = append(l.buf, ch)
	l.advance()
	return lexString{}, nil
}

// lexComment consumes from # to end of line.
type lexComment struct{}

func (lexComment) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 {
		l.emit(TokenComment, l.bufString())
		return lexDefault{}, nil
	}
	if ch == '\n' {
		l.emit(TokenComment, l.bufString())
		// Don't consume the newline — let lexDefault emit it.
		return lexDefault{}, nil
	}
	l.buf = append(l.buf, ch)
	l.advance()
	return lexComment{}, nil
}

// lexNumber consumes digits and dots.
type lexNumber struct{}

func (lexNumber) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 {
		l.emit(TokenNumber, l.bufString())
		return lexDefault{}, nil
	}
	if ch >= '0' && ch <= '9' || ch == '.' {
		l.buf = append(l.buf, ch)
		l.advance()
		return lexNumber{}, nil
	}
	l.emit(TokenNumber, l.bufString())
	return lexDefault{}.Handle(l, ch)
}

// lexRawBlock consumes everything between {| and |}.
// Content is emitted as a single TokenRawBlock with leading/trailing whitespace trimmed.
type lexRawBlock struct{}

func (lexRawBlock) Handle(l *Lexer, ch rune) (LexState, error) {
	if ch == 0 {
		l.emit(TokenRawBlock, strings.TrimSpace(l.bufString()))
		return lexDefault{}, &ParseError{
			Line:    l.start.line,
			Col:     l.start.col,
			Message: "unterminated raw block, expected '|}'",
		}
	}
	if ch == '|' && l.peekAt(1) == '}' {
		l.advance() // consume |
		l.advance() // consume }
		l.emit(TokenRawBlock, strings.TrimSpace(l.bufString()))
		return lexDefault{}, nil
	}
	l.buf = append(l.buf, ch)
	l.advance()
	return lexRawBlock{}, nil
}

// --- helpers ---

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentContinue(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}

func isDigitStart(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
