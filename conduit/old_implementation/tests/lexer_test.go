package tests

import (
	"testing"
	"github.com/user/gox/pkg/lexer"
)

func TestBasicTokens(t *testing.T) {
	input := `package main`

	tests := []struct {
		expectedType    lexer.TokenType
		expectedLiteral string
	}{
		{lexer.PACKAGE, "package"},
		{lexer.IDENT, "main"},
		{lexer.EOF, ""},
	}

	l := lexer.New([]byte(input), "test.go")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%d, got=%d",
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

	tests := []struct {
		expectedType    lexer.TokenType
		expectedLiteral string
	}{
		{lexer.COMPONENT, "component"},
		{lexer.IDENT, "Button"},
		{lexer.LPAREN, "("},
		{lexer.RPAREN, ")"},
		{lexer.LBRACE, "{"},
		{lexer.RBRACE, "}"},
	}

	l := lexer.New([]byte(input), "test.gox")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%d, got=%d",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestJSXTokens(t *testing.T) {
	input := `<div>Hello</div>`

	l := lexer.New([]byte(input), "test.gox")
	l.PushMode(lexer.ModeJSX)

	tests := []struct {
		expectedType    lexer.TokenType
		expectedLiteral string
	}{
		{lexer.JSX_LT, "<"},
		{lexer.IDENT, "div"},
		{lexer.JSX_GT, ">"},
		{lexer.JSX_TEXT, "Hello"},
		{lexer.JSX_SLASH, "</"},
		{lexer.IDENT, "div"},
		{lexer.JSX_GT, ">"},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%d, got=%d",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestJSXAttributes(t *testing.T) {
	input := `<button className="btn" onClick={handler}>Click</button>`

	l := lexer.New([]byte(input), "test.gox")
	l.PushMode(lexer.ModeJSX)

	tests := []struct {
		expectedType    lexer.TokenType
		expectedLiteral string
	}{
		{lexer.JSX_LT, "<"},
		{lexer.IDENT, "button"},
		{lexer.IDENT, "className"},
		{lexer.ASSIGN, "="},
		{lexer.STRING, "btn"},
		{lexer.IDENT, "onClick"},
		{lexer.ASSIGN, "="},
		{lexer.JSX_LBRACE, "{"},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%d, got=%d",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if i >= len(tests)-1 {
			break
		}
	}
}

func TestRenderBlock(t *testing.T) {
	input := `render { <div>Test</div> }`

	l := lexer.New([]byte(input), "test.gox")

	// Should recognize render keyword and switch to JSX mode
	tok := l.NextToken()
	if tok.Type != lexer.RENDER {
		t.Fatalf("expected RENDER, got %d", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != lexer.LBRACE {
		t.Fatalf("expected LBRACE, got %d", tok.Type)
	}

	// Should now be in JSX mode
	tok = l.NextToken()
	if tok.Type != lexer.JSX_LT {
		t.Fatalf("expected JSX_LT, got %d", tok.Type)
	}
}