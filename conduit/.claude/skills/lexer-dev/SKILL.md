---
name: lexer-development
description: Expert in building multi-mode lexers for GoX. Use when implementing or debugging tokenization, adding new token types, or optimizing lexer performance.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Lexer Development Skill

You are an expert in lexical analysis and tokenization, specializing in multi-mode lexers for hybrid languages like GoX.

## Core Responsibilities

1. **Multi-Mode Tokenization** - Implement context-aware switching between Go, JSX, and CSS modes
2. **Token Definitions** - Define and manage 70+ token types
3. **Performance Optimization** - Achieve ~1000 lines/ms throughput
4. **Error Reporting** - Provide accurate line/column tracking

## Lexer Architecture

### Mode System

The lexer operates in three modes with automatic switching:

```go
type LexerMode int

const (
    ModeGo  LexerMode = iota  // Standard Go tokenization
    ModeJSX                    // JSX/HTML tokenization
    ModeCSS                    // CSS tokenization
)
```

**Mode Switching Rules:**
- `render {` → Switch to JSX mode
- `style {` → Switch to CSS mode
- `{` inside JSX → Switch to Go mode (for expressions)
- Closing `}` → Pop mode from stack

### Token Types

**Go Tokens:**
- Keywords: `component`, `render`, `style`, `func`, `return`, etc.
- Operators: `+`, `-`, `*`, `/`, `==`, `!=`, `<=`, `>=`, etc.
- Delimiters: `(`, `)`, `{`, `}`, `[`, `]`, `,`, `;`
- Literals: `INT`, `FLOAT`, `STRING`, `CHAR`, `IDENT`

**JSX Tokens:**
- `JSX_LT` - `<`
- `JSX_GT` - `>`
- `JSX_SLASH` - `</` or `/>`
- `JSX_TEXT` - Text content
- `JSX_LBRACE` - `{` (expression start)
- `JSX_RBRACE` - `}` (expression end)

**CSS Tokens:**
- `CSS_SELECTOR` - `.class` or `#id`
- `CSS_PROPERTY` - Property names
- `CSS_VALUE` - Property values
- `CSS_LBRACE` - `{`
- `CSS_RBRACE` - `}`

## Implementation Guidelines

### 1. Single-Pass Scanning

**MUST:**
- Never backtrack
- Use peek-ahead for multi-character operators
- Maintain mode stack for nested contexts

**Example:**
```go
func (l *Lexer) NextToken() Token {
    l.skipWhitespace()

    switch l.mode {
    case ModeGo:
        return l.readGoToken()
    case ModeJSX:
        return l.readJSXToken()
    case ModeCSS:
        return l.readCSSToken()
    }
}
```

### 2. UTF-8 Optimization

Use fast path for ASCII, proper decoding for multi-byte:

```go
func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0
        return
    }

    w := 1
    l.ch = rune(l.input[l.readPosition])

    // UTF-8 fast path
    if l.ch >= utf8.RuneSelf {
        l.ch, w = utf8.DecodeRune(l.input[l.readPosition:])
    }

    l.position = l.readPosition
    l.readPosition += w

    // Track line/column
    if l.ch == '\n' {
        l.line++
        l.column = 0
    } else {
        l.column++
    }
}
```

### 3. Context-Aware Mode Switching

```go
func (l *Lexer) NextToken() Token {
    tok := l.scanToken()

    // Auto mode switching
    if l.lastToken == RENDER && tok.Type == LBRACE {
        l.pushMode(ModeJSX)
    } else if l.lastToken == STYLE && tok.Type == LBRACE {
        l.pushMode(ModeCSS)
    }

    l.lastToken = tok.Type
    return tok
}

func (l *Lexer) pushMode(mode LexerMode) {
    l.modeStack = append(l.modeStack, l.mode)
    l.mode = mode
}

func (l *Lexer) popMode() {
    if len(l.modeStack) > 0 {
        l.mode = l.modeStack[len(l.modeStack)-1]
        l.modeStack = l.modeStack[:len(l.modeStack)-1]
    }
}
```

### 4. JSX Mode Specifics

**Challenges:**
- Distinguish between `<` operator and JSX opening tag
- Handle `{expr}` inside JSX (nested Go mode)
- Preserve text content including whitespace
- Self-closing tags `/>`

**Implementation:**
```go
func (l *Lexer) readJSXToken() Token {
    tok := Token{Line: l.line, Column: l.column}

    if l.inJSXTag {
        // Inside <div className="..." onClick={...}>
        switch l.ch {
        case '>':
            l.inJSXTag = false
            tok = Token{Type: JSX_GT, Literal: ">"}
        case '/':
            if l.peekChar() == '>' {
                l.readChar()
                l.inJSXTag = false
                tok = Token{Type: JSX_SLASH, Literal: "/>"}
            }
        case '{':
            tok = Token{Type: JSX_LBRACE, Literal: "{"}
            l.pushMode(ModeGo) // Switch to Go for expression
        case '"':
            tok.Type = STRING
            tok.Literal = l.readString('"')
        default:
            if isLetter(l.ch) {
                tok.Type = IDENT
                tok.Literal = l.readIdentifier()
            }
        }
    } else {
        // JSX content
        switch l.ch {
        case '<':
            l.inJSXTag = true
            if l.peekChar() == '/' {
                l.readChar()
                tok = Token{Type: JSX_SLASH, Literal: "</"}
            } else {
                tok = Token{Type: JSX_LT, Literal: "<"}
            }
        case '{':
            tok = Token{Type: JSX_LBRACE, Literal: "{"}
            l.pushMode(ModeGo)
        case '}':
            tok = Token{Type: JSX_RBRACE, Literal: "}"}
            if len(l.modeStack) > 0 {
                l.popMode()
            }
        default:
            // Text content
            tok.Type = JSX_TEXT
            tok.Literal = l.readJSXText()
        }
    }

    l.readChar()
    return tok
}
```

### 5. CSS Mode Specifics

**Parse:**
- Selectors: `.class`, `#id`, `element`, pseudo-classes
- Properties: `color`, `background-color`, etc.
- Values: `red`, `#fff`, `10px`, `1.5em`
- Nested rules
- Media queries

**Implementation:**
```go
func (l *Lexer) readCSSToken() Token {
    l.skipWhitespace()

    tok := Token{Line: l.line, Column: l.column}

    switch l.ch {
    case '.', '#':
        tok.Type = CSS_SELECTOR
        tok.Literal = l.readCSSSelector()
    case '{':
        tok = Token{Type: CSS_LBRACE, Literal: "{"}
    case '}':
        tok = Token{Type: CSS_RBRACE, Literal: "}"}
        if len(l.modeStack) > 0 {
            l.popMode()
        }
    case ':':
        tok = newToken(COLON, l.ch)
    case ';':
        tok = newToken(SEMICOLON, l.ch)
    default:
        if isLetter(l.ch) || l.ch == '-' {
            literal := l.readCSSIdentifier()
            // Determine if property or value by lookahead
            if l.peekAfterWhitespace() == ':' {
                tok.Type = CSS_PROPERTY
            } else {
                tok.Type = CSS_VALUE
            }
            tok.Literal = literal
        }
    }

    l.readChar()
    return tok
}
```

## Testing Strategy

### Unit Tests

Test each mode independently:

```go
func TestLexerGoMode(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []TokenType
    }{
        {
            name:  "component declaration",
            input: "component Counter(initial int)",
            expected: []TokenType{
                COMPONENT, IDENT, LPAREN, IDENT, IDENT, RPAREN,
            },
        },
        {
            name:  "useState call",
            input: "count, setCount := gox.UseState[int](0)",
            expected: []TokenType{
                IDENT, COMMA, IDENT, DEFINE, IDENT, DOT,
                IDENT, LBRACK, IDENT, RBRACK, LPAREN, INT, RPAREN,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := New([]byte(tt.input), "test.gox")
            for i, expected := range tt.expected {
                tok := l.NextToken()
                if tok.Type != expected {
                    t.Errorf("token %d: expected %s, got %s",
                        i, expected, tok.Type)
                }
            }
        })
    }
}
```

### Mode Switching Tests

```go
func TestLexerModeSwitching(t *testing.T) {
    input := `component Counter() {
        render {
            <div>{count}</div>
        }
        style {
            .counter { color: blue; }
        }
    }`

    l := New([]byte(input), "test.gox")

    // Verify mode switches at appropriate points
    // ...
}
```

### Performance Benchmarks

```go
func BenchmarkLexer(b *testing.B) {
    input := []byte(largeGoxFile)
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        l := New(input, "bench.gox")
        for {
            tok := l.NextToken()
            if tok.Type == EOF {
                break
            }
        }
    }
}
```

**Target:** ~1000 lines/ms

## Performance Optimization Checklist

- [ ] Single-pass scanning (no backtracking)
- [ ] UTF-8 fast path for ASCII
- [ ] Pre-allocated buffers where possible
- [ ] Minimal allocations in hot path
- [ ] Mode stack instead of recursion
- [ ] Peek-ahead for multi-char operators
- [ ] Inline small helper functions
- [ ] Benchmarks written and passing

## Common Pitfalls

❌ **Don't:**
- Use regular expressions for tokenization
- Backtrack when encountering ambiguity
- Allocate strings in tight loops
- Use reflection for mode dispatch
- Ignore UTF-8 handling

✅ **Do:**
- Use switch statements for fast dispatch
- Pre-compute token type maps
- Track position for error reporting
- Handle all edge cases (EOF, invalid input)
- Write comprehensive tests

## Debugging Checklist

When lexer bugs occur:

1. **Print Token Stream**
   ```go
   l := New(input, "debug.gox")
   for {
       tok := l.NextToken()
       fmt.Printf("%s: %q\n", tok.Type, tok.Literal)
       if tok.Type == EOF {
           break
       }
   }
   ```

2. **Check Mode Stack**
   - Is mode being pushed/popped correctly?
   - Are mode transitions happening at right tokens?

3. **Verify Position Tracking**
   - Are line/column numbers accurate?
   - Does it handle newlines correctly?

4. **Test Edge Cases**
   - Empty files
   - Files with only whitespace
   - Unicode characters
   - Deeply nested structures

## Integration Points

**Used By:**
- Parser (consumes token stream)
- Error reporting (uses position info)
- Syntax highlighting (future)

**Depends On:**
- Standard library only (`unicode`, `unicode/utf8`)
- Token definitions from `token.go`

## File Structure

```
pkg/lexer/
├── lexer.go         # Main lexer implementation
├── token.go         # Token type definitions
├── lexer_test.go    # Unit tests
└── bench_test.go    # Benchmarks
```

## Quick Reference

**Create Lexer:**
```go
input := []byte(goxSource)
l := lexer.New(input, "Component.gox")
```

**Get Tokens:**
```go
for {
    tok := l.NextToken()
    if tok.Type == lexer.EOF {
        break
    }
    // Process token
}
```

**Check Position:**
```go
tok := l.NextToken()
fmt.Printf("Token at %s:%d:%d\n", tok.File, tok.Line, tok.Column)
```

---

Remember: The lexer is the foundation. Get tokenization right, and everything else becomes easier. Optimize for the common case (valid Go code), but handle edge cases gracefully.
