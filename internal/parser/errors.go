package parser

import (
	"fmt"
	"strings"
)

// ParseError represents a single parsing error with location information.
type ParseError struct {
	Line    int
	Col     int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Col, e.Message)
}

// MultiParseError collects multiple parsing errors for best-effort parsing.
type MultiParseError struct {
	Errors []*ParseError
}

func (e *MultiParseError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d parse errors:\n", len(e.Errors))
	for _, err := range e.Errors {
		fmt.Fprintf(&b, "  %s\n", err.Error())
	}
	return b.String()
}

func (e *MultiParseError) Add(line, col int, msg string) {
	e.Errors = append(e.Errors, &ParseError{Line: line, Col: col, Message: msg})
}

func (e *MultiParseError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Diagnostics flattens a parse error returned by Parse into a slice of
// ParseErrors carrying line/col positions. It returns nil for a nil error.
// Unknown error types are reported as a single positionless diagnostic.
func Diagnostics(err error) []*ParseError {
	switch e := err.(type) {
	case nil:
		return nil
	case *MultiParseError:
		return e.Errors
	case *ParseError:
		return []*ParseError{e}
	default:
		return []*ParseError{{Line: 0, Col: 0, Message: err.Error()}}
	}
}
