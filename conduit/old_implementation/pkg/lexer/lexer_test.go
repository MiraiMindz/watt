package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `package main

import "gox"

component Counter(initial int) {
	count, setCount := gox.UseState[int](initial)

	render {
		<div className="counter">
			<h1>Count: {count}</h1>
			<button onClick={increment}>+</button>
		</div>
	}

	style {
		.counter {
			padding: 20px;
		}
	}
}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{PACKAGE, "package"},
		{IDENT, "main"},
		{IMPORT, "import"},
		{STRING, "gox"},
		{COMPONENT, "component"},
		{IDENT, "Counter"},
		{LPAREN, "("},
		{IDENT, "initial"},
		{IDENT, "int"},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{IDENT, "count"},
		{COMMA, ","},
		{IDENT, "setCount"},
		{DEFINE, ":="},
		{IDENT, "gox"},
		{PERIOD, "."},
		{IDENT, "UseState"},
		{LBRACK, "["},
		{IDENT, "int"},
		{RBRACK, "]"},
		{LPAREN, "("},
		{IDENT, "initial"},
		{RPAREN, ")"},
	}

	l := New([]byte(input), "test.gox")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComponentKeyword(t *testing.T) {
	input := `component Button() {}`

	l := New([]byte(input), "test.gox")

	tok := l.NextToken()
	if tok.Type != COMPONENT {
		t.Errorf("expected COMPONENT, got %v", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != IDENT || tok.Literal != "Button" {
		t.Errorf("expected Button identifier, got %v", tok)
	}
}

func TestRenderBlock(t *testing.T) {
	input := `render { <div>Hello</div> }`

	l := New([]byte(input), "test.gox")

	tok := l.NextToken()
	if tok.Type != RENDER {
		t.Errorf("expected RENDER, got %v", tok.Type)
	}
}

func TestStyleBlock(t *testing.T) {
	input := `style { .class { color: red; } }`

	l := New([]byte(input), "test.gox")

	tok := l.NextToken()
	if tok.Type != STYLE {
		t.Errorf("expected STYLE, got %v", tok.Type)
	}
}

func TestHookPattern(t *testing.T) {
	inputs := []string{
		`gox.UseState[int](0)`,
		`gox.UseEffect(func() {}, []interface{}{})`,
		`gox.UseMemo[string](compute, deps)`,
		`gox.UseRef[*Node](nil)`,
	}

	for _, input := range inputs {
		l := New([]byte(input), "test.gox")

		// Should tokenize "gox"
		tok := l.NextToken()
		if tok.Type != IDENT || tok.Literal != "gox" {
			t.Errorf("expected 'gox' identifier, got %v", tok)
		}

		// Should tokenize "."
		tok = l.NextToken()
		if tok.Type != PERIOD {
			t.Errorf("expected PERIOD, got %v", tok.Type)
		}

		// Should tokenize hook name
		tok = l.NextToken()
		if tok.Type != IDENT {
			t.Errorf("expected hook name identifier, got %v", tok)
		}
	}
}