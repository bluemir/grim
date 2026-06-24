package parser

import "testing"

func TestDiagnostics_Nil(t *testing.T) {
	if d := Diagnostics(nil); d != nil {
		t.Errorf("expected nil for nil error, got %v", d)
	}
}

func TestDiagnostics_Single(t *testing.T) {
	err := &ParseError{Line: 3, Col: 5, Message: "boom"}
	d := Diagnostics(err)
	if len(d) != 1 || d[0].Line != 3 || d[0].Col != 5 || d[0].Message != "boom" {
		t.Errorf("unexpected diagnostics: %v", d)
	}
}

func TestDiagnostics_Multi(t *testing.T) {
	me := &MultiParseError{}
	me.Add(1, 2, "first")
	me.Add(4, 0, "second")
	d := Diagnostics(me)
	if len(d) != 2 || d[0].Message != "first" || d[1].Line != 4 {
		t.Errorf("unexpected diagnostics: %v", d)
	}
}

func TestDiagnostics_FromParse(t *testing.T) {
	// Unterminated string should yield a positioned diagnostic.
	_, err := Parse(`a: "unterminated`)
	d := Diagnostics(err)
	if len(d) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	if d[0].Line == 0 {
		t.Errorf("expected a line position, got %v", d[0])
	}
}
