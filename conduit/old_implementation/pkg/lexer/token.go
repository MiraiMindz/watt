// Package lexer provides tokenization for GoX source files
package lexer

import "fmt"

// TokenType represents the type of a lexical token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	IDENT   // abc
	INT     // 123
	FLOAT   // 123.45
	STRING  // "abc" or `abc`
	CHAR    // 'a'

	// Keywords (Go standard)
	PACKAGE
	IMPORT
	FUNC
	VAR
	CONST
	TYPE
	STRUCT
	INTERFACE
	IF
	ELSE
	FOR
	RETURN
	BREAK
	CONTINUE
	RANGE
	SWITCH
	CASE
	DEFAULT
	GOTO
	SELECT
	DEFER
	GO
	CHAN
	MAP

	// Keywords (GoX specific)
	COMPONENT // component keyword
	RENDER    // render block
	STYLE     // style block
	PROPS     // props (optional)

	// JSX-like tokens
	JSX_LT      // <
	JSX_GT      // >
	JSX_SLASH   // /> or </
	JSX_LBRACE  // { in JSX context
	JSX_RBRACE  // } in JSX context
	JSX_TEXT    // text content in JSX

	// CSS tokens
	CSS_SELECTOR
	CSS_PROPERTY
	CSS_VALUE
	CSS_LBRACE
	CSS_RBRACE

	// Operators & Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACK    // [
	RBRACK    // ]
	SEMICOLON // ;
	COMMA     // ,
	PERIOD    // .
	COLON     // :
	ASSIGN    // =
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	PERCENT   // %
	AND       // &
	OR        // |
	XOR       // ^
	SHL       // <<
	SHR       // >>
	ANDNOT    // &^
	LAND      // &&
	LOR       // ||
	ARROW     // <-
	INC       // ++
	DEC       // --
	EQ        // ==
	LT        // <
	GT        // >
	NOT       // !
	NEQ       // !=
	LEQ       // <=
	GEQ       // >=
	DEFINE    // :=
	ELLIPSIS  // ...
	WALRUS    // :=
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	STRING: "STRING",
	CHAR:   "CHAR",

	PACKAGE:   "package",
	IMPORT:    "import",
	FUNC:      "func",
	VAR:       "var",
	CONST:     "const",
	TYPE:      "type",
	STRUCT:    "struct",
	INTERFACE: "interface",
	IF:        "if",
	ELSE:      "else",
	FOR:       "for",
	RETURN:    "return",
	BREAK:     "break",
	CONTINUE:  "continue",
	RANGE:     "range",
	SWITCH:    "switch",
	CASE:      "case",
	DEFAULT:   "default",
	GOTO:      "goto",
	SELECT:    "select",
	DEFER:     "defer",
	GO:        "go",
	CHAN:      "chan",
	MAP:       "map",

	COMPONENT: "component",
	RENDER:    "render",
	STYLE:     "style",
	PROPS:     "props",

	JSX_LT:     "JSX_<",
	JSX_GT:     "JSX_>",
	JSX_SLASH:  "JSX_/",
	JSX_LBRACE: "JSX_{",
	JSX_RBRACE: "JSX_}",
	JSX_TEXT:   "JSX_TEXT",

	CSS_SELECTOR: "CSS_SELECTOR",
	CSS_PROPERTY: "CSS_PROPERTY",
	CSS_VALUE:    "CSS_VALUE",
	CSS_LBRACE:   "CSS_{",
	CSS_RBRACE:   "CSS_}",

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACE:    "{",
	RBRACE:    "}",
	LBRACK:    "[",
	RBRACK:    "]",
	SEMICOLON: ";",
	COMMA:     ",",
	PERIOD:    ".",
	COLON:     ":",
	ASSIGN:    "=",
	PLUS:      "+",
	MINUS:     "-",
	STAR:      "*",
	SLASH:     "/",
	PERCENT:   "%",
	AND:       "&",
	OR:        "|",
	XOR:       "^",
	SHL:       "<<",
	SHR:       ">>",
	ANDNOT:    "&^",
	LAND:      "&&",
	LOR:       "||",
	ARROW:     "<-",
	INC:       "++",
	DEC:       "--",
	EQ:        "==",
	LT:        "<",
	GT:        ">",
	NOT:       "!",
	NEQ:       "!=",
	LEQ:       "<=",
	GEQ:       ">=",
	DEFINE:    ":=",
	ELLIPSIS:  "...",
}

// String returns the string representation of the token type
func (t TokenType) String() string {
	s := ""
	if 0 <= t && t < TokenType(len(tokens)) {
		s = tokens[t]
	}
	if s == "" {
		s = fmt.Sprintf("TokenType(%d)", int(t))
	}
	return s
}

// Token represents a lexical token with its position information
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
	File    string
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("{Type:%v Literal:%q Pos:%d:%d}", t.Type, t.Literal, t.Line, t.Column)
}

// keywords contains the mapping from strings to keyword token types
var keywords = map[string]TokenType{
	"package":   PACKAGE,
	"import":    IMPORT,
	"func":      FUNC,
	"var":       VAR,
	"const":     CONST,
	"type":      TYPE,
	"struct":    STRUCT,
	"interface": INTERFACE,
	"if":        IF,
	"else":      ELSE,
	"for":       FOR,
	"return":    RETURN,
	"break":     BREAK,
	"continue":  CONTINUE,
	"range":     RANGE,
	"switch":    SWITCH,
	"case":      CASE,
	"default":   DEFAULT,
	"goto":      GOTO,
	"select":    SELECT,
	"defer":     DEFER,
	"go":        GO,
	"chan":      CHAN,
	"map":       MAP,
	"component": COMPONENT,
	"render":    RENDER,
	"style":     STYLE,
	"props":     PROPS,
}

// LookupKeyword checks if an identifier is a keyword and returns its type
func LookupKeyword(ident string) TokenType {
	if tok, isKeyword := keywords[ident]; isKeyword {
		return tok
	}
	return IDENT
}