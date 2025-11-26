---
name: parser-development
description: Expert in building parsers for GoX components. Use when implementing AST generation, parsing JSX/CSS, handling hooks, or debugging parse errors.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Parser Development Skill

You are an expert in parser construction, specifically for hybrid languages combining Go, JSX, and CSS like GoX.

## Core Responsibilities

1. **AST Generation** - Build abstract syntax trees from token streams
2. **Component Parsing** - Parse component declarations with props
3. **Hook Parsing** - Extract useState, useEffect, etc.
4. **JSX Parsing** - Parse JSX elements, attributes, children
5. **CSS Parsing** - Parse style blocks with rules and properties
6. **Error Recovery** - Collect multiple errors without stopping

## Parser Architecture

### Parsing Strategy

**Type:** Recursive Descent with LL(2) lookahead
**Complexity:** ~500 lines/ms target
**Error Handling:** Multi-error collection

```go
type Parser struct {
    lexer     *lexer.Lexer
    curToken  lexer.Token
    peekToken lexer.Token
    errors    []error
    fset      *token.FileSet
    filename  string
}
```

### AST Structure

```go
// File represents a parsed GoX file
type File struct {
    Package    string
    Imports    []*ast.ImportSpec
    Components []*ComponentDecl
    Functions  []*ast.FuncDecl
    Types      []*ast.TypeSpec
}

// ComponentDecl represents a component
type ComponentDecl struct {
    Position token.Pos
    Name     *ast.Ident
    Params   *ast.FieldList // Props
    Body     *ComponentBody
}

// ComponentBody contains hooks, statements, render, style
type ComponentBody struct {
    Hooks  []*HookCall
    Stmts  []ast.Stmt
    Render *RenderBlock
    Style  *StyleBlock
}
```

## Implementation Guidelines

### 1. File Structure Parsing

```go
func (p *Parser) ParseFile() (*File, error) {
    file := &File{
        Components: []*ComponentDecl{},
        Functions:  []*ast.FuncDecl{},
        Types:      []*ast.TypeSpec{},
        Imports:    []*ast.ImportSpec{},
    }

    // Parse package declaration
    if p.curTokenIs(lexer.PACKAGE) {
        p.nextToken()
        if p.curTokenIs(lexer.IDENT) {
            file.Package = p.curToken.Literal
            p.nextToken()
        }
    }

    // Parse imports
    for p.curTokenIs(lexer.IMPORT) {
        imports := p.parseImports()
        file.Imports = append(file.Imports, imports...)
    }

    // Parse top-level declarations
    for !p.curTokenIs(lexer.EOF) {
        switch p.curToken.Type {
        case lexer.COMPONENT:
            comp := p.parseComponent()
            if comp != nil {
                file.Components = append(file.Components, comp)
            }
        case lexer.FUNC:
            p.skipToNextDeclaration()
        case lexer.TYPE:
            p.skipToNextDeclaration()
        default:
            p.nextToken()
        }
    }

    if len(p.errors) > 0 {
        return file, p.errors[0]
    }

    return file, nil
}
```

### 2. Component Declaration Parsing

```go
func (p *Parser) parseComponent() *ComponentDecl {
    comp := &ComponentDecl{
        Position: token.Pos(p.curToken.Line),
    }

    p.nextToken() // consume 'component'

    // Parse component name
    if !p.curTokenIs(lexer.IDENT) {
        p.addError("expected component name")
        return nil
    }
    comp.Name = &ast.Ident{Name: p.curToken.Literal}
    p.nextToken()

    // Parse parameters (props)
    if p.curTokenIs(lexer.LPAREN) {
        comp.Params = p.parseParameterList()
    }

    // Parse body
    if !p.curTokenIs(lexer.LBRACE) {
        p.addError("expected '{' after component declaration")
        return nil
    }

    comp.Body = p.parseComponentBody()

    return comp
}

func (p *Parser) parseParameterList() *ast.FieldList {
    fields := &ast.FieldList{
        Opening: token.Pos(p.curToken.Line),
        List:    []*ast.Field{},
    }

    p.nextToken() // consume '('

    for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
        field := &ast.Field{}

        // Parse parameter name
        if p.curTokenIs(lexer.IDENT) {
            field.Names = []*ast.Ident{{Name: p.curToken.Literal}}
            p.nextToken()
        }

        // Parse parameter type
        if p.curTokenIs(lexer.IDENT) {
            field.Type = &ast.Ident{Name: p.curToken.Literal}
            p.nextToken()
        }

        fields.List = append(fields.List, field)

        if p.curTokenIs(lexer.COMMA) {
            p.nextToken()
        }
    }

    if p.curTokenIs(lexer.RPAREN) {
        fields.Closing = token.Pos(p.curToken.Line)
        p.nextToken()
    }

    return fields
}
```

### 3. Hook Parsing

```go
func (p *Parser) parseHookStatement() *HookCall {
    // Match pattern: result1, result2 := gox.UseHook[Type](args)

    // Collect LHS identifiers
    results := []string{}
    if p.curTokenIs(lexer.IDENT) {
        results = append(results, p.curToken.Literal)
        p.nextToken()

        // Handle multiple results
        for p.curTokenIs(lexer.COMMA) {
            p.nextToken()
            if p.curTokenIs(lexer.IDENT) {
                results = append(results, p.curToken.Literal)
                p.nextToken()
            }
        }
    }

    // Expect :=
    if !p.curTokenIs(lexer.DEFINE) {
        return nil
    }
    p.nextToken()

    // Parse RHS: gox.UseState[int](0)
    // Skip 'gox.' if present
    if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "gox" {
        p.nextToken()
        if p.curTokenIs(lexer.DOT) {
            p.nextToken()
        }
    }

    // Hook name
    if !p.curTokenIs(lexer.IDENT) {
        return nil
    }
    hookName := p.curToken.Literal

    // Check if it's a known hook
    if !isHookName(hookName) {
        return nil
    }

    hook := &HookCall{
        Position: token.Pos(p.curToken.Line),
        Name:     hookName,
        Results:  results,
    }

    p.nextToken()

    // Parse type arguments [T]
    if p.curTokenIs(lexer.LBRACK) {
        hook.TypeArgs = p.parseTypeArguments()
    }

    // Parse arguments (initial)
    if p.curTokenIs(lexer.LPAREN) {
        hook.Args = p.parseArguments()
    }

    return hook
}

func isHookName(name string) bool {
    hooks := map[string]bool{
        "UseState":      true,
        "UseEffect":     true,
        "UseMemo":       true,
        "UseCallback":   true,
        "UseRef":        true,
        "UseContext":    true,
        "UseReducer":    true,
        "UseId":         true,
        "UseTransition": true,
    }
    return hooks[name]
}
```

### 4. JSX Parsing

```go
func (p *Parser) parseJSXElement() JSXNode {
    elem := &JSXElement{
        Position: token.Pos(p.curToken.Line),
    }

    // Consume <
    if !p.curTokenIs(lexer.JSX_LT) {
        p.addError("expected '<' to start JSX element")
        return nil
    }
    p.nextToken()

    // Parse tag name
    if !p.curTokenIs(lexer.IDENT) {
        p.addError("expected tag name")
        return nil
    }
    elem.Tag = p.curToken.Literal
    p.nextToken()

    // Parse attributes
    for !p.curTokenIs(lexer.JSX_GT) && !p.curTokenIs(lexer.JSX_SLASH) && !p.curTokenIs(lexer.EOF) {
        if p.curTokenIs(lexer.IDENT) {
            attr := p.parseJSXAttribute()
            if attr != nil {
                elem.Attrs = append(elem.Attrs, attr)
            }
        } else {
            p.nextToken()
        }
    }

    // Check for self-closing
    if p.curTokenIs(lexer.JSX_SLASH) {
        elem.SelfClosing = true
        p.nextToken() // consume '/>'
        return elem
    }

    // Consume >
    if !p.curTokenIs(lexer.JSX_GT) {
        p.addError("expected '>' or '/>'")
        return nil
    }
    p.nextToken()

    // Parse children
    elem.Children = p.parseJSXChildren(elem.Tag)

    // Parse closing tag
    if p.curTokenIs(lexer.JSX_SLASH) {
        p.nextToken() // consume '</'
        if p.curTokenIs(lexer.IDENT) {
            closingTag := p.curToken.Literal
            if closingTag != elem.Tag {
                p.addError(fmt.Sprintf("mismatched closing tag: expected '%s', got '%s'",
                    elem.Tag, closingTag))
            }
            p.nextToken()
        }
        if p.curTokenIs(lexer.JSX_GT) {
            p.nextToken()
        }
    }

    return elem
}

func (p *Parser) parseJSXAttribute() *JSXAttribute {
    attr := &JSXAttribute{
        Position: token.Pos(p.curToken.Line),
        Name:     p.curToken.Literal,
    }
    p.nextToken()

    // Expect =
    if !p.curTokenIs(lexer.ASSIGN) {
        p.addError("expected '=' after attribute name")
        return nil
    }
    p.nextToken()

    // Parse value: string literal or {expression}
    if p.curTokenIs(lexer.STRING) {
        attr.Value = &JSXText{
            Value:    p.curToken.Literal,
            Position: token.Pos(p.curToken.Line),
        }
        p.nextToken()
    } else if p.curTokenIs(lexer.JSX_LBRACE) {
        p.nextToken()
        // Parse expression until }
        attr.Value = p.parseExpression()
        if p.curTokenIs(lexer.JSX_RBRACE) {
            p.nextToken()
        }
    }

    return attr
}

func (p *Parser) parseJSXChildren(parentTag string) []JSXNode {
    children := []JSXNode{}

    for !p.curTokenIs(lexer.JSX_SLASH) && !p.curTokenIs(lexer.EOF) {
        switch p.curToken.Type {
        case lexer.JSX_LT:
            // Nested element
            child := p.parseJSXElement()
            if child != nil {
                children = append(children, child)
            }

        case lexer.JSX_TEXT:
            // Text content
            text := &JSXText{
                Value:    p.curToken.Literal,
                Position: token.Pos(p.curToken.Line),
            }
            children = append(children, text)
            p.nextToken()

        case lexer.JSX_LBRACE:
            // Expression
            p.nextToken()
            expr := p.parseExpression()
            if expr != nil {
                children = append(children, &JSXExpression{
                    Position: token.Pos(p.curToken.Line),
                    Expr:     expr,
                })
            }
            if p.curTokenIs(lexer.JSX_RBRACE) {
                p.nextToken()
            }

        default:
            p.nextToken()
        }
    }

    return children
}
```

### 5. CSS Parsing

```go
func (p *Parser) parseStyleBlock() *StyleBlock {
    style := &StyleBlock{
        Position: token.Pos(p.curToken.Line),
        Rules:    []*CSSRule{},
    }

    // Check for 'global' modifier
    if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "global" {
        style.Global = true
        p.nextToken()
    }

    // Expect {
    if !p.curTokenIs(lexer.LBRACE) {
        p.addError("expected '{' after 'style'")
        return nil
    }
    p.nextToken()

    // Parse CSS rules
    for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
        rule := p.parseCSSRule()
        if rule != nil {
            style.Rules = append(style.Rules, rule)
        }
    }

    if p.curTokenIs(lexer.RBRACE) {
        p.nextToken()
    }

    return style
}

func (p *Parser) parseCSSRule() *CSSRule {
    rule := &CSSRule{
        Position:   token.Pos(p.curToken.Line),
        Properties: []*CSSProperty{},
    }

    // Parse selector
    if p.curTokenIs(lexer.CSS_SELECTOR) || p.curTokenIs(lexer.IDENT) {
        rule.Selector = p.curToken.Literal
        p.nextToken()
    } else {
        p.addError("expected CSS selector")
        return nil
    }

    // Expect {
    if !p.curTokenIs(lexer.CSS_LBRACE) {
        p.addError("expected '{' after selector")
        return nil
    }
    p.nextToken()

    // Parse properties
    for !p.curTokenIs(lexer.CSS_RBRACE) && !p.curTokenIs(lexer.EOF) {
        prop := p.parseCSSProperty()
        if prop != nil {
            rule.Properties = append(rule.Properties, prop)
        }
    }

    if p.curTokenIs(lexer.CSS_RBRACE) {
        p.nextToken()
    }

    return rule
}

func (p *Parser) parseCSSProperty() *CSSProperty {
    prop := &CSSProperty{
        Position: token.Pos(p.curToken.Line),
    }

    // Parse property name
    if p.curTokenIs(lexer.CSS_PROPERTY) || p.curTokenIs(lexer.IDENT) {
        prop.Name = p.curToken.Literal
        p.nextToken()
    } else {
        return nil
    }

    // Expect :
    if !p.curTokenIs(lexer.COLON) {
        p.addError("expected ':' after property name")
        return nil
    }
    p.nextToken()

    // Parse property value
    if p.curTokenIs(lexer.CSS_VALUE) || p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING) {
        prop.Value = p.curToken.Literal
        p.nextToken()
    }

    // Expect ;
    if p.curTokenIs(lexer.SEMICOLON) {
        p.nextToken()
    }

    return prop
}
```

### 6. Error Handling

```go
func (p *Parser) addError(msg string) {
    // Limit errors to prevent spam
    if len(p.errors) >= 10 {
        return
    }

    err := fmt.Errorf("%s:%d:%d: %s",
        p.filename, p.curToken.Line, p.curToken.Column, msg)
    p.errors = append(p.errors, err)
}

func (p *Parser) skipToNextDeclaration() {
    braceCount := 0
    for !p.curTokenIs(lexer.EOF) {
        if p.curTokenIs(lexer.LBRACE) {
            braceCount++
        } else if p.curTokenIs(lexer.RBRACE) {
            braceCount--
            if braceCount <= 0 {
                p.nextToken()
                return
            }
        }
        p.nextToken()
    }
}

func (p *Parser) skipStatement() {
    for !p.curTokenIs(lexer.SEMICOLON) && !p.curTokenIs(lexer.EOF) {
        p.nextToken()
    }
    if p.curTokenIs(lexer.SEMICOLON) {
        p.nextToken()
    }
}
```

## Testing Strategy

### Component Parsing Tests

```go
func TestParseComponent(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name: "simple component",
            input: `component Counter() {
                render {
                    <div>Counter</div>
                }
            }`,
            expected: "Counter",
        },
        {
            name: "component with props",
            input: `component UserCard(user User, isAdmin bool) {
                render {
                    <div>{user.Name}</div>
                }
            }`,
            expected: "UserCard",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := lexer.New([]byte(tt.input), "test.gox")
            p := New(l, "test.gox")
            file, err := p.ParseFile()

            if err != nil {
                t.Fatalf("parse error: %v", err)
            }

            if len(file.Components) != 1 {
                t.Fatalf("expected 1 component, got %d", len(file.Components))
            }

            if file.Components[0].Name.Name != tt.expected {
                t.Errorf("expected component name %s, got %s",
                    tt.expected, file.Components[0].Name.Name)
            }
        })
    }
}
```

### Hook Parsing Tests

```go
func TestParseHooks(t *testing.T) {
    input := `component Counter(initial int) {
        count, setCount := gox.UseState[int](initial)
        name, setName := gox.UseState[string]("")

        gox.UseEffect(func() func() {
            fmt.Println("mounted")
            return func() {
                fmt.Println("cleanup")
            }
        }, []interface{}{})

        render {
            <div>{count}</div>
        }
    }`

    l := lexer.New([]byte(input), "test.gox")
    p := New(l, "test.gox")
    file, err := p.ParseFile()

    if err != nil {
        t.Fatalf("parse error: %v", err)
    }

    comp := file.Components[0]

    // Should have 3 hooks
    if len(comp.Body.Hooks) != 3 {
        t.Errorf("expected 3 hooks, got %d", len(comp.Body.Hooks))
    }

    // Check useState hooks
    if comp.Body.Hooks[0].Name != "UseState" {
        t.Errorf("expected UseState, got %s", comp.Body.Hooks[0].Name)
    }

    if len(comp.Body.Hooks[0].Results) != 2 {
        t.Errorf("expected 2 results from useState, got %d",
            len(comp.Body.Hooks[0].Results))
    }
}
```

### JSX Parsing Tests

```go
func TestParseJSX(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectedTag string
        numChildren int
    }{
        {
            name:        "simple element",
            input:       "<div>Hello</div>",
            expectedTag: "div",
            numChildren: 1,
        },
        {
            name:        "nested elements",
            input:       "<div><p>Hello</p><p>World</p></div>",
            expectedTag: "div",
            numChildren: 2,
        },
        {
            name:        "element with attributes",
            input:       `<div className="container" id="main">Content</div>`,
            expectedTag: "div",
            numChildren: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := lexer.New([]byte(tt.input), "test.gox")
            p := New(l, "test.gox")

            elem := p.parseJSXElement()
            jsxElem, ok := elem.(*JSXElement)
            if !ok {
                t.Fatalf("expected JSXElement")
            }

            if jsxElem.Tag != tt.expectedTag {
                t.Errorf("expected tag %s, got %s", tt.expectedTag, jsxElem.Tag)
            }

            if len(jsxElem.Children) != tt.numChildren {
                t.Errorf("expected %d children, got %d",
                    tt.numChildren, len(jsxElem.Children))
            }
        })
    }
}
```

## Performance Optimization Checklist

- [ ] LL(2) parsing (no backtracking)
- [ ] Pre-allocated slices for common cases
- [ ] Early returns for self-closing tags
- [ ] Skip strategies for error recovery
- [ ] Minimal allocations in hot path
- [ ] Reuse AST nodes from go/ast
- [ ] Benchmarks written and passing
- [ ] Target: ~500 lines/ms

## Common Pitfalls

❌ **Don't:**
- Backtrack on parse errors
- Panic on invalid input
- Allocate new slices for every list
- Ignore error cases
- Parse the same construct twice

✅ **Do:**
- Collect all errors before returning
- Use descriptive error messages
- Track file positions accurately
- Handle EOF gracefully
- Test edge cases thoroughly

## Debugging Checklist

1. **Print AST Structure**
   ```go
   ast.Print(file)
   ```

2. **Check Token Stream**
   - Is lexer returning correct tokens?
   - Are mode switches happening?

3. **Verify Error Messages**
   - Are positions accurate?
   - Are messages helpful?

4. **Test Incrementally**
   - Parse components alone
   - Add hooks
   - Add JSX
   - Add CSS

## Integration Points

**Consumes:**
- Token stream from Lexer
- Token position information

**Produces:**
- AST for Analyzer
- Error messages for user

**Used By:**
- Analyzer (semantic analysis)
- Error reporter

## File Structure

```
pkg/parser/
├── parser.go        # Main parser implementation
├── ast.go           # AST node definitions
├── parser_test.go   # Unit tests
└── bench_test.go    # Benchmarks
```

## Quick Reference

**Create Parser:**
```go
l := lexer.New(input, "Component.gox")
p := parser.New(l, "Component.gox")
```

**Parse File:**
```go
file, err := p.ParseFile()
if err != nil {
    // Handle errors
}
```

**Access Components:**
```go
for _, comp := range file.Components {
    fmt.Println(comp.Name.Name)
    for _, hook := range comp.Body.Hooks {
        fmt.Println(hook.Name)
    }
}
```

---

Remember: The parser transforms a flat token stream into a rich tree structure. Focus on correctness first, then optimize. A slow correct parser beats a fast incorrect one.
