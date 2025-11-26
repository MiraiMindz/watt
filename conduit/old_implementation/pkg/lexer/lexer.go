package lexer

import (
	"unicode"
	"unicode/utf8"
)

// LexerMode represents the current mode of the lexer
type LexerMode int

const (
	ModeGo LexerMode = iota
	ModeJSX
	ModeCSS
)

// Lexer tokenizes GoX source code
type Lexer struct {
	input        []byte
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           rune // current char under examination
	line         int
	column       int
	file         string
	mode         LexerMode
	modeStack    []LexerMode // for nested contexts
	lastToken    TokenType    // track last token for context-aware mode switching
	inJSXTag     bool         // track if we're inside a JSX tag (between < and >)
}

// New creates a new Lexer instance
func New(input []byte, filename string) *Lexer {
	l := &Lexer{
		input:     input,
		line:      1,
		column:    0,
		file:      filename,
		mode:      ModeGo,
		modeStack: []LexerMode{},
	}
	l.readChar()
	return l
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column
	tok.File = l.file

	switch l.mode {
	case ModeGo:
		tok = l.readGoToken()
	case ModeJSX:
		tok = l.readJSXToken()
	case ModeCSS:
		tok = l.readCSSToken()
	}

	// Track the last token and handle mode switching
	if l.lastToken == RENDER && tok.Type == LBRACE {
		// Switch to JSX mode after "render {"
		l.pushMode(ModeJSX)
	} else if l.lastToken == STYLE && tok.Type == LBRACE {
		// Switch to CSS mode after "style {"
		l.pushMode(ModeCSS)
	}

	l.lastToken = tok.Type

	return tok
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		w := 1
		l.ch = rune(l.input[l.readPosition])
		if l.ch >= utf8.RuneSelf {
			l.ch, w = utf8.DecodeRune(l.input[l.readPosition:])
		}
		l.position = l.readPosition
		l.readPosition += w
	}

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	ch := rune(l.input[l.readPosition])
	if ch >= utf8.RuneSelf {
		ch, _ = utf8.DecodeRune(l.input[l.readPosition:])
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readGoToken() Token {
	tok := Token{Line: l.line, Column: l.column, File: l.file}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: EQ, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(ASSIGN, l.ch)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: INC, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(PLUS, l.ch)
		}
	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: DEC, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(MINUS, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: NEQ, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(NOT, l.ch)
		}
	case '/':
		if l.peekChar() == '/' {
			// Comment
			tok.Type = COMMENT
			tok.Literal = l.readComment()
		} else if l.peekChar() == '*' {
			// Multi-line comment
			tok.Type = COMMENT
			tok.Literal = l.readMultilineComment()
		} else {
			tok = newToken(SLASH, l.ch)
		}
	case '*':
		tok = newToken(STAR, l.ch)
	case '<':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: ARROW, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: LEQ, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: SHL, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: GEQ, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: SHR, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(GT, l.ch)
		}
	case ';':
		tok = newToken(SEMICOLON, l.ch)
	case ',':
		tok = newToken(COMMA, l.ch)
	case '.':
		if l.peekChar() == '.' && l.peekCharN(2) == '.' {
			l.readChar()
			l.readChar()
			tok = Token{Type: ELLIPSIS, Literal: "...", Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(PERIOD, l.ch)
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: DEFINE, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(COLON, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: LAND, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else if l.peekChar() == '^' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: ANDNOT, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(AND, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = Token{Type: LOR, Literal: literal, Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = newToken(OR, l.ch)
		}
	case '^':
		tok = newToken(XOR, l.ch)
	case '%':
		tok = newToken(PERCENT, l.ch)
	case '{':
		tok = newToken(LBRACE, l.ch)
		// Check if we should enter JSX or CSS mode based on previous context
		// The mode was prepared when we saw "render" or "style" keyword
	case '}':
		tok = newToken(RBRACE, l.ch)
		// If we're in Go mode because of a JSX expression, pop back to JSX
		if len(l.modeStack) > 0 && l.modeStack[len(l.modeStack)-1] == ModeJSX {
			l.popMode()
		}
	case '(':
		tok = newToken(LPAREN, l.ch)
	case ')':
		tok = newToken(RPAREN, l.ch)
	case '[':
		tok = newToken(LBRACK, l.ch)
	case ']':
		tok = newToken(RBRACK, l.ch)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString('"')
	case '`':
		tok.Type = STRING
		tok.Literal = l.readString('`')
	case '\'':
		tok.Type = CHAR
		tok.Literal = l.readCharLiteral()
	case 0:
		tok.Literal = ""
		tok.Type = EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupKeyword(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readJSXToken() Token {
	tok := Token{Line: l.line, Column: l.column, File: l.file}

	// Skip whitespace in JSX mode
	if l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.skipWhitespace()
		if l.ch == 0 {
			return Token{Type: EOF, Line: tok.Line, Column: tok.Column, File: tok.File}
		}
	}

	// If we're inside a JSX tag, handle attributes differently
	if l.inJSXTag {
		switch l.ch {
		case '>':
			l.inJSXTag = false
			tok = Token{Type: JSX_GT, Literal: ">", Line: tok.Line, Column: tok.Column, File: tok.File}
			l.readChar()
			return tok
		case '/':
			if l.peekChar() == '>' {
				l.readChar()
				l.inJSXTag = false
				tok = Token{Type: JSX_SLASH, Literal: "/>", Line: tok.Line, Column: tok.Column, File: tok.File}
				l.readChar()
				return tok
			}
			tok = newToken(SLASH, l.ch)
			l.readChar()
			return tok
		case '=':
			tok = newToken(ASSIGN, l.ch)
			l.readChar()
			return tok
		case '"':
			tok.Type = STRING
			tok.Literal = l.readString('"')
			l.readChar() // consume closing quote
			return tok
		case '{':
			// Expression attribute value
			tok = Token{Type: JSX_LBRACE, Literal: "{", Line: tok.Line, Column: tok.Column, File: tok.File}
			l.pushMode(ModeGo)
			l.readChar()
			return tok
		default:
			// Read identifier (tag name or attribute name)
			if isLetter(l.ch) {
				tok.Type = IDENT
				tok.Literal = l.readIdentifier()
				return tok
			}
			tok = newToken(ILLEGAL, l.ch)
			l.readChar()
			return tok
		}
	}

	// Normal JSX content parsing
	switch l.ch {
	case '<':
		// Check for closing tag
		if l.peekChar() == '/' {
			l.readChar()
			l.inJSXTag = true
			tok = Token{Type: JSX_SLASH, Literal: "</", Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			l.inJSXTag = true
			tok = Token{Type: JSX_LT, Literal: "<", Line: tok.Line, Column: tok.Column, File: tok.File}
		}
	case '>':
		l.inJSXTag = false
		tok = Token{Type: JSX_GT, Literal: ">", Line: tok.Line, Column: tok.Column, File: tok.File}
	case '{':
		// Embedded expression
		tok = Token{Type: JSX_LBRACE, Literal: "{", Line: tok.Line, Column: tok.Column, File: tok.File}
		// Switch to Go mode for expression
		l.pushMode(ModeGo)
	case '}':
		// This could be end of expression or end of render block
		// Pop mode to return to previous context
		if len(l.modeStack) > 0 {
			l.popMode()
		}
		// Return as regular RBRACE if we're back in Go mode, otherwise JSX_RBRACE
		if l.mode == ModeGo {
			tok = Token{Type: RBRACE, Literal: "}", Line: tok.Line, Column: tok.Column, File: tok.File}
		} else {
			tok = Token{Type: JSX_RBRACE, Literal: "}", Line: tok.Line, Column: tok.Column, File: tok.File}
		}
	default:
		// Read JSX text content
		if l.ch != '<' && l.ch != '>' && l.ch != '{' && l.ch != '}' && l.ch != 0 {
			tok.Type = JSX_TEXT
			tok.Literal = l.readJSXText()
			return tok
		}
		tok = newToken(ILLEGAL, l.ch)
	}

	l.readChar()
	return tok
}

func (l *Lexer) readCSSToken() Token {
	tok := Token{Line: l.line, Column: l.column, File: l.file}

	l.skipWhitespace()

	switch l.ch {
	case '{':
		tok = Token{Type: CSS_LBRACE, Literal: "{", Line: tok.Line, Column: tok.Column, File: tok.File}
	case '}':
		tok = Token{Type: CSS_RBRACE, Literal: "}", Line: tok.Line, Column: tok.Column, File: tok.File}
		// Check if we should exit CSS mode
		if len(l.modeStack) > 0 {
			l.popMode()
		}
	case ':':
		tok = newToken(COLON, l.ch)
	case ';':
		tok = newToken(SEMICOLON, l.ch)
	case '.':
		// CSS class selector
		tok.Type = CSS_SELECTOR
		tok.Literal = l.readCSSSelector()
		return tok
	case '#':
		// CSS ID selector
		tok.Type = CSS_SELECTOR
		tok.Literal = l.readCSSSelector()
		return tok
	default:
		if isLetter(l.ch) || l.ch == '-' {
			// Could be a CSS property or value
			literal := l.readCSSIdentifier()
			// Simple heuristic: if next non-whitespace is ':', it's a property
			if l.peekAfterWhitespace() == ':' {
				tok.Type = CSS_PROPERTY
			} else {
				tok.Type = CSS_VALUE
			}
			tok.Literal = literal
			return tok
		} else if isDigit(l.ch) {
			// CSS value (number with unit)
			tok.Type = CSS_VALUE
			tok.Literal = l.readCSSValue()
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readNumber() (TokenType, string) {
	position := l.position
	tokenType := INT

	// Read integer part
	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for float
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = FLOAT
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Check for scientific notation
	if l.ch == 'e' || l.ch == 'E' {
		tokenType = FLOAT
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokenType, string(l.input[position:l.position])
}

func (l *Lexer) readString(delimiter rune) string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '\\' {
			l.readChar() // Skip escaped character
		} else if l.ch == delimiter || l.ch == 0 {
			break
		}
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readCharLiteral() string {
	position := l.position + 1
	l.readChar() // Move to the character after opening quote

	if l.ch == '\\' {
		// Handle escape sequences
		l.readChar() // Read the escaped character
	}

	endPos := l.position
	l.readChar() // This should be the closing quote

	if endPos > position {
		return string(l.input[position:endPos])
	}
	return ""
}

func (l *Lexer) readComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readMultilineComment() string {
	position := l.position
	l.readChar() // Skip '/'
	l.readChar() // Skip '*'

	for {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		if l.ch == 0 {
			break
		}
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readJSXText() string {
	position := l.position
	for l.ch != '<' && l.ch != '>' && l.ch != '{' && l.ch != '}' && l.ch != 0 {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readCSSSelector() string {
	position := l.position
	l.readChar() // Skip . or #
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readCSSIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readCSSValue() string {
	position := l.position
	// Read number
	for isDigit(l.ch) || l.ch == '.' {
		l.readChar()
	}
	// Read unit (px, em, %, etc.)
	for isLetter(l.ch) || l.ch == '%' {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) peekCharN(n int) rune {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return rune(l.input[pos])
}

func (l *Lexer) peekAfterWhitespace() rune {
	pos := l.readPosition
	for pos < len(l.input) {
		ch := rune(l.input[pos])
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			return ch
		}
		pos++
	}
	return 0
}

func (l *Lexer) PushMode(mode LexerMode) {
	l.modeStack = append(l.modeStack, l.mode)
	l.mode = mode
}

func (l *Lexer) pushMode(mode LexerMode) {
	l.PushMode(mode)
}

func (l *Lexer) popMode() {
	if len(l.modeStack) > 0 {
		l.mode = l.modeStack[len(l.modeStack)-1]
		l.modeStack = l.modeStack[:len(l.modeStack)-1]
	}
}

func newToken(tokenType TokenType, ch rune) Token {
	return Token{Type: tokenType, Literal: string(ch)}
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}