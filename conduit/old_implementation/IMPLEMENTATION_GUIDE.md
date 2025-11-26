# GoX Implementation Guide

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Project Structure](#project-structure)
3. [Phase 1: Lexer & Parser](#phase-1-lexer--parser)
4. [Phase 2: AST & IR](#phase-2-ast--ir)
5. [Phase 3: SSR Transpiler](#phase-3-ssr-transpiler)
6. [Phase 4: CSR/WASM Transpiler](#phase-4-csrwasm-transpiler)
7. [Phase 5: Runtime Library](#phase-5-runtime-library)
8. [Phase 6: Hook System](#phase-6-hook-system)
9. [Phase 7: Virtual DOM](#phase-7-virtual-dom)
10. [Phase 8: Build Toolchain](#phase-8-build-toolchain)
11. [Phase 9: Developer Experience](#phase-9-developer-experience)
12. [Implementation Timeline](#implementation-timeline)

---

## Architecture Overview

### High-Level Design

```
┌─────────────┐
│  .gox files │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  GoX Compiler   │
│  (lexer/parser) │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  AST   │
    └───┬────┘
        │
   ┌────┴─────┐
   │          │
   ▼          ▼
┌─────┐   ┌──────┐
│ SSR │   │ CSR  │
│Mode │   │(WASM)│
└──┬──┘   └───┬──┘
   │          │
   ▼          ▼
┌────────┐ ┌─────────┐
│.go file│ │.go+WASM │
│+ HTML  │ │+ VDom   │
└────────┘ └─────────┘
```

### Core Principles

1. **Single Source**: One .gox file contains component logic, markup, and styles
2. **Type Safety**: Full Go type system + generic hooks (useState[T])
3. **Dual Compilation**: Same source compiles to SSR or CSR
4. **React-Like DX**: Familiar API for React developers
5. **Go Native**: Integrates seamlessly with existing Go code

---

## Project Structure

```
gox/
├── cmd/
│   ├── goxc/              # Main compiler CLI
│   │   └── main.go
│   └── gox-dev/           # Development server
│       └── main.go
├── pkg/
│   ├── lexer/             # Tokenization
│   │   ├── lexer.go
│   │   ├── token.go
│   │   └── lexer_test.go
│   ├── parser/            # AST construction
│   │   ├── parser.go
│   │   ├── ast.go
│   │   └── parser_test.go
│   ├── analyzer/          # Semantic analysis
│   │   ├── analyzer.go
│   │   ├── scope.go
│   │   └── types.go
│   ├── transpiler/        # Code generation
│   │   ├── ssr/
│   │   │   ├── transpiler.go
│   │   │   ├── templates.go
│   │   │   └── codegen.go
│   │   └── csr/
│   │       ├── transpiler.go
│   │       ├── vdom.go
│   │       └── codegen.go
│   ├── runtime/           # Runtime library
│   │   ├── component.go
│   │   ├── hooks.go
│   │   ├── context.go
│   │   └── wasm/
│   │       ├── vdom.go
│   │       ├── diff.go
│   │       ├── reconciler.go
│   │       └── dom.go
│   └── cli/               # CLI utilities
│       ├── build.go
│       ├── watch.go
│       └── serve.go
├── runtime/               # User-facing runtime (separate module)
│   ├── gox.go
│   ├── hooks.go
│   ├── types.go
│   └── ssr/
│       └── renderer.go
├── examples/
│   ├── counter/
│   ├── todo-app/
│   └── ssr-blog/
├── docs/
│   ├── syntax.md
│   ├── api.md
│   └── guides/
├── go.mod
└── README.md
```

---

## Phase 1: Lexer & Parser

### 1.1 Token Design

```go
// pkg/lexer/token.go
package lexer

type TokenType int

const (
    // Go standard tokens (reuse go/token where possible)
    ILLEGAL TokenType = iota
    EOF
    COMMENT

    // Literals
    IDENT
    INT
    FLOAT
    STRING

    // Keywords (Go)
    PACKAGE
    IMPORT
    FUNC
    VAR
    CONST
    TYPE

    // Keywords (GoX specific)
    COMPONENT    // component
    PROPS        // props (optional, can use params)
    RENDER       // render block
    STYLE        // style block

    // JSX-like tokens
    JSX_LT       // <
    JSX_GT       // >
    JSX_SLASH    // />
    JSX_LBRACE   // {
    JSX_RBRACE   // }
    JSX_TEXT     // text content

    // CSS tokens
    CSS_SELECTOR
    CSS_PROPERTY
    CSS_VALUE
    CSS_LBRACE
    CSS_RBRACE

    // Operators & Delimiters
    LPAREN       // (
    RPAREN       // )
    LBRACE       // {
    RBRACE       // }
    LBRACK       // [
    RBRACK       // ]
    SEMICOLON    // ;
    COMMA        // ,
    PERIOD       // .
    COLON        // :
    ASSIGN       // =
    // ... more operators
)

type Token struct {
    Type    TokenType
    Literal string
    Line    int
    Column  int
    File    string
}
```

### 1.2 Lexer Implementation

```go
// pkg/lexer/lexer.go
package lexer

import (
    "unicode"
    "unicode/utf8"
)

type LexerMode int

const (
    ModeGo LexerMode = iota
    ModeJSX
    ModeCSS
)

type Lexer struct {
    input        []byte
    position     int  // current position
    readPosition int  // next position
    ch           rune // current char
    line         int
    column       int
    file         string
    mode         LexerMode
    modeStack    []LexerMode // for nested contexts
}

func New(input []byte, filename string) *Lexer {
    l := &Lexer{
        input:  input,
        line:   1,
        column: 0,
        file:   filename,
        mode:   ModeGo,
    }
    l.readChar()
    return l
}

func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0 // EOF
    } else {
        l.ch, _ = utf8.DecodeRune(l.input[l.readPosition:])
        l.position = l.readPosition
        l.readPosition += utf8.RuneLen(l.ch)
    }

    if l.ch == '\n' {
        l.line++
        l.column = 0
    } else {
        l.column++
    }
}

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

    return tok
}

func (l *Lexer) readGoToken() Token {
    // Implement Go tokenization
    // When encountering 'component' keyword, prepare for JSX mode
    // When encountering 'style' keyword, switch to CSS mode
    // Reuse logic from go/scanner where possible
}

func (l *Lexer) readJSXToken() Token {
    // Handle JSX syntax
    // Switch back to Go mode when encountering {expression}
}

func (l *Lexer) readCSSToken() Token {
    // Handle CSS syntax
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

### 1.3 Parser Implementation

```go
// pkg/parser/ast.go
package parser

import (
    "go/ast"
    "go/token"
)

// GoX-specific AST nodes

type ComponentDecl struct {
    Name       *ast.Ident
    Doc        *ast.CommentGroup
    Params     *ast.FieldList    // function parameters (props)
    Body       *ComponentBody
    Position   token.Pos
}

type ComponentBody struct {
    Stmts      []ast.Stmt        // Regular Go statements
    Hooks      []HookCall        // useState, useEffect, etc.
    Render     *RenderBlock      // JSX return
    Style      *StyleBlock       // CSS styles
}

type HookCall struct {
    Name       string            // "useState", "useEffect", etc.
    TypeArgs   []ast.Expr        // Generic type arguments
    Args       []ast.Expr        // Arguments
    Results    []ast.Expr        // Return values (state, setState)
}

type RenderBlock struct {
    Root       JSXElement
}

type JSXElement struct {
    Tag        string            // "div", "span", or Component name
    Attrs      []JSXAttribute
    Children   []JSXNode
    Position   token.Pos
    SelfClosing bool
}

type JSXAttribute struct {
    Name       string
    Value      JSXAttrValue      // string or {expression}
}

type JSXAttrValue interface {
    jsxAttrValue()
}

type JSXText struct {
    Value string
}

type JSXExpression struct {
    Expr ast.Expr
}

type JSXNode interface {
    jsxNode()
}

type StyleBlock struct {
    Rules []CSSRule
}

type CSSRule struct {
    Selector   string
    Properties []CSSProperty
}

type CSSProperty struct {
    Name  string
    Value string
}

// Implement jsxNode() and jsxAttrValue() interfaces
func (JSXElement) jsxNode() {}
func (JSXText) jsxNode() {}
func (JSXExpression) jsxNode() {}
func (JSXText) jsxAttrValue() {}
func (JSXExpression) jsxAttrValue() {}
```

```go
// pkg/parser/parser.go
package parser

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "gox/pkg/lexer"
)

type Parser struct {
    lexer      *lexer.Lexer
    curToken   lexer.Token
    peekToken  lexer.Token
    errors     []string
    fset       *token.FileSet
}

func New(l *lexer.Lexer) *Parser {
    p := &Parser{
        lexer: l,
        fset:  token.NewFileSet(),
    }
    p.nextToken()
    p.nextToken()
    return p
}

func (p *Parser) ParseFile() (*ast.File, []*ComponentDecl, error) {
    // Parse as Go file first to get package, imports, etc.
    goFile, err := parser.ParseFile(p.fset, p.lexer.Filename(), p.lexer.Source(), 0)
    if err != nil {
        return nil, nil, err
    }

    // Now parse GoX-specific constructs
    components := []*ComponentDecl{}

    for p.curToken.Type != lexer.EOF {
        if p.curToken.Type == lexer.COMPONENT {
            comp := p.parseComponent()
            if comp != nil {
                components = append(components, comp)
            }
        }
        p.nextToken()
    }

    return goFile, components, nil
}

func (p *Parser) parseComponent() *ComponentDecl {
    comp := &ComponentDecl{
        Position: p.curToken.Position,
    }

    p.nextToken() // consume 'component'

    // Parse name
    if p.curToken.Type != lexer.IDENT {
        p.error("expected component name")
        return nil
    }
    comp.Name = &ast.Ident{Name: p.curToken.Literal}
    p.nextToken()

    // Parse parameters (props)
    if p.curToken.Type == lexer.LPAREN {
        comp.Params = p.parseParameterList()
    }

    // Parse body
    if p.curToken.Type != lexer.LBRACE {
        p.error("expected '{'")
        return nil
    }
    comp.Body = p.parseComponentBody()

    return comp
}

func (p *Parser) parseComponentBody() *ComponentBody {
    body := &ComponentBody{}
    p.nextToken() // consume '{'

    for p.curToken.Type != lexer.RBRACE {
        // Try to parse hook calls
        if p.isHookCall() {
            hook := p.parseHookCall()
            body.Hooks = append(body.Hooks, hook)
        } else if p.curToken.Literal == "render" {
            // Parse render block
            p.nextToken() // consume 'render'
            body.Render = p.parseRenderBlock()
        } else if p.curToken.Literal == "style" {
            // Parse style block
            p.nextToken() // consume 'style'
            body.Style = p.parseStyleBlock()
        } else {
            // Regular Go statement
            stmt := p.parseStatement()
            body.Stmts = append(body.Stmts, stmt)
        }
        p.nextToken()
    }

    return body
}

func (p *Parser) parseHookCall() HookCall {
    // Parse useState[T](initial) pattern
    hook := HookCall{
        Name: p.curToken.Literal,
    }
    p.nextToken()

    // Parse generic type args
    if p.curToken.Type == lexer.LBRACK {
        hook.TypeArgs = p.parseTypeArguments()
    }

    // Parse arguments
    if p.curToken.Type == lexer.LPAREN {
        hook.Args = p.parseArgumentList()
    }

    return hook
}

func (p *Parser) parseRenderBlock() *RenderBlock {
    // Parse JSX-like syntax
    p.nextToken() // consume '{'

    render := &RenderBlock{}
    render.Root = p.parseJSXElement()

    return render
}

func (p *Parser) parseJSXElement() JSXElement {
    elem := JSXElement{}

    // <tag
    p.expectToken(lexer.JSX_LT)
    p.nextToken()

    // tag name
    elem.Tag = p.curToken.Literal
    p.nextToken()

    // attributes
    for p.curToken.Type != lexer.JSX_GT && p.curToken.Type != lexer.JSX_SLASH {
        attr := p.parseJSXAttribute()
        elem.Attrs = append(elem.Attrs, attr)
    }

    // Self-closing?
    if p.curToken.Type == lexer.JSX_SLASH {
        elem.SelfClosing = true
        p.nextToken()
        p.expectToken(lexer.JSX_GT)
        return elem
    }

    // >
    p.expectToken(lexer.JSX_GT)
    p.nextToken()

    // children
    for p.curToken.Type != lexer.JSX_LT || p.peekToken.Type != lexer.JSX_SLASH {
        child := p.parseJSXChild()
        elem.Children = append(elem.Children, child)
    }

    // </tag>
    p.expectToken(lexer.JSX_LT)
    p.nextToken()
    p.expectToken(lexer.JSX_SLASH)
    p.nextToken()
    p.expectToken(lexer.IDENT) // closing tag name
    p.nextToken()
    p.expectToken(lexer.JSX_GT)

    return elem
}

func (p *Parser) parseJSXAttribute() JSXAttribute {
    attr := JSXAttribute{
        Name: p.curToken.Literal,
    }
    p.nextToken()

    if p.curToken.Type == lexer.ASSIGN {
        p.nextToken()

        if p.curToken.Type == lexer.STRING {
            attr.Value = JSXText{Value: p.curToken.Literal}
        } else if p.curToken.Type == lexer.JSX_LBRACE {
            // {expression}
            p.nextToken()
            expr := p.parseExpression()
            attr.Value = JSXExpression{Expr: expr}
            p.expectToken(lexer.JSX_RBRACE)
        }
        p.nextToken()
    }

    return attr
}

func (p *Parser) parseStyleBlock() *StyleBlock {
    style := &StyleBlock{}
    p.nextToken() // consume '{'

    for p.curToken.Type != lexer.RBRACE {
        rule := p.parseCSSRule()
        style.Rules = append(style.Rules, rule)
    }

    return style
}

func (p *Parser) parseCSSRule() CSSRule {
    rule := CSSRule{
        Selector: p.curToken.Literal,
    }
    p.nextToken()
    p.expectToken(lexer.CSS_LBRACE)
    p.nextToken()

    for p.curToken.Type != lexer.CSS_RBRACE {
        prop := CSSProperty{
            Name: p.curToken.Literal,
        }
        p.nextToken()
        p.expectToken(lexer.COLON)
        p.nextToken()
        prop.Value = p.curToken.Literal
        p.nextToken()
        p.expectToken(lexer.SEMICOLON)
        p.nextToken()

        rule.Properties = append(rule.Properties, prop)
    }

    p.expectToken(lexer.CSS_RBRACE)
    return rule
}

// Helper methods
func (p *Parser) nextToken() {
    p.curToken = p.peekToken
    p.peekToken = p.lexer.NextToken()
}

func (p *Parser) expectToken(t lexer.TokenType) {
    if p.curToken.Type != t {
        p.error(fmt.Sprintf("expected %v, got %v", t, p.curToken.Type))
    }
}

func (p *Parser) error(msg string) {
    p.errors = append(p.errors, fmt.Sprintf("%s:%d:%d: %s",
        p.curToken.File, p.curToken.Line, p.curToken.Column, msg))
}
```

---

## Phase 2: AST & IR

### 2.1 Intermediate Representation

```go
// pkg/analyzer/ir.go
package analyzer

import (
    "go/ast"
    "gox/pkg/parser"
)

// IR represents the analyzed and normalized component structure
type ComponentIR struct {
    Name          string
    Props         []PropField
    State         []StateVar
    Effects       []EffectHook
    Memos         []MemoHook
    Refs          []RefHook
    Callbacks     []CallbackHook
    Context       []ContextHook
    CustomHooks   []CustomHookCall
    RenderLogic   *RenderIR
    Styles        *StyleIR
    Dependencies  []string      // Other components used
}

type PropField struct {
    Name     string
    Type     ast.Expr
    Optional bool
}

type StateVar struct {
    Name        string
    Type        ast.Expr
    InitValue   ast.Expr
    Setter      string        // setState function name
}

type EffectHook struct {
    Setup       ast.BlockStmt
    Cleanup     ast.BlockStmt
    Deps        []string      // Dependency list
}

type MemoHook struct {
    Name        string
    Type        ast.Expr
    Compute     ast.Expr
    Deps        []string
}

type RefHook struct {
    Name        string
    Type        ast.Expr
    InitValue   ast.Expr
}

type CallbackHook struct {
    Name        string
    Callback    *ast.FuncLit
    Deps        []string
}

type ContextHook struct {
    Name        string
    Type        ast.Expr
    ContextName string
}

type CustomHookCall struct {
    Name        string
    Args        []ast.Expr
    Results     []string
}

type RenderIR struct {
    VNode       *VNodeIR
}

type VNodeIR struct {
    Type        VNodeType
    Tag         string           // HTML tag or component name
    Props       map[string]IRExpr
    Children    []*VNodeIR
    Key         IRExpr
    Condition   IRExpr           // for conditional rendering
    Loop        *LoopIR          // for list rendering
}

type VNodeType int

const (
    VNodeElement VNodeType = iota
    VNodeComponent
    VNodeText
    VNodeExpression
    VNodeFragment
)

type LoopIR struct {
    Item        string
    Index       string
    Collection  IRExpr
    Body        *VNodeIR
}

type IRExpr struct {
    Type        IRExprType
    Value       interface{}
    Original    ast.Expr
}

type IRExprType int

const (
    IRExprLiteral IRExprType = iota
    IRExprIdentifier
    IRExprBinary
    IRExprCall
    IRExprMember
)

type StyleIR struct {
    Scoped      bool
    Rules       []CSSRuleIR
    ComponentID string        // For scoped styles
}

type CSSRuleIR struct {
    Selector    string
    Properties  map[string]string
    Scoped      bool
}
```

### 2.2 Semantic Analyzer

```go
// pkg/analyzer/analyzer.go
package analyzer

import (
    "fmt"
    "go/ast"
    "gox/pkg/parser"
)

type Analyzer struct {
    components map[string]*ComponentIR
    errors     []error
}

func New() *Analyzer {
    return &Analyzer{
        components: make(map[string]*ComponentIR),
    }
}

func (a *Analyzer) Analyze(file *ast.File, components []*parser.ComponentDecl) ([]*ComponentIR, error) {
    var irs []*ComponentIR

    for _, comp := range components {
        ir, err := a.analyzeComponent(comp)
        if err != nil {
            a.errors = append(a.errors, err)
            continue
        }

        irs = append(irs, ir)
        a.components[ir.Name] = ir
    }

    if len(a.errors) > 0 {
        return nil, fmt.Errorf("analysis errors: %v", a.errors)
    }

    return irs, nil
}

func (a *Analyzer) analyzeComponent(comp *parser.ComponentDecl) (*ComponentIR, error) {
    ir := &ComponentIR{
        Name: comp.Name.Name,
    }

    // Extract props from parameters
    if comp.Params != nil {
        ir.Props = a.extractProps(comp.Params)
    }

    // Analyze hooks
    for _, hook := range comp.Body.Hooks {
        switch hook.Name {
        case "useState":
            state := a.analyzeUseState(hook)
            ir.State = append(ir.State, state)
        case "useEffect":
            effect := a.analyzeUseEffect(hook)
            ir.Effects = append(ir.Effects, effect)
        case "useMemo":
            memo := a.analyzeUseMemo(hook)
            ir.Memos = append(ir.Memos, memo)
        case "useRef":
            ref := a.analyzeUseRef(hook)
            ir.Refs = append(ir.Refs, ref)
        case "useCallback":
            callback := a.analyzeUseCallback(hook)
            ir.Callbacks = append(ir.Callbacks, callback)
        case "useContext":
            ctx := a.analyzeUseContext(hook)
            ir.Context = append(ir.Context, ctx)
        default:
            // Custom hook
            custom := CustomHookCall{Name: hook.Name}
            ir.CustomHooks = append(ir.CustomHooks, custom)
        }
    }

    // Analyze render block
    if comp.Body.Render != nil {
        ir.RenderLogic = a.analyzeRender(comp.Body.Render)
    }

    // Analyze styles
    if comp.Body.Style != nil {
        ir.Styles = a.analyzeStyle(comp.Body.Style, ir.Name)
    }

    return ir, nil
}

func (a *Analyzer) extractProps(params *ast.FieldList) []PropField {
    var props []PropField
    for _, field := range params.List {
        for _, name := range field.Names {
            props = append(props, PropField{
                Name: name.Name,
                Type: field.Type,
            })
        }
    }
    return props
}

func (a *Analyzer) analyzeUseState(hook parser.HookCall) StateVar {
    // count, setCount := useState[int](0)
    state := StateVar{}

    if len(hook.Results) >= 2 {
        state.Name = hook.Results[0].(*ast.Ident).Name
        state.Setter = hook.Results[1].(*ast.Ident).Name
    }

    if len(hook.TypeArgs) > 0 {
        state.Type = hook.TypeArgs[0]
    }

    if len(hook.Args) > 0 {
        state.InitValue = hook.Args[0]
    }

    return state
}

func (a *Analyzer) analyzeUseEffect(hook parser.HookCall) EffectHook {
    effect := EffectHook{}

    // useEffect(func() { ... }, []string{deps})
    if len(hook.Args) > 0 {
        if fn, ok := hook.Args[0].(*ast.FuncLit); ok {
            effect.Setup = *fn.Body
        }
    }

    if len(hook.Args) > 1 {
        // Parse dependency array
        effect.Deps = a.parseDependencyArray(hook.Args[1])
    }

    return effect
}

func (a *Analyzer) analyzeRender(render *parser.RenderBlock) *RenderIR {
    return &RenderIR{
        VNode: a.jsxToVNode(render.Root),
    }
}

func (a *Analyzer) jsxToVNode(jsx parser.JSXElement) *VNodeIR {
    vnode := &VNodeIR{
        Tag:   jsx.Tag,
        Props: make(map[string]IRExpr),
    }

    // Determine if it's an HTML element or component
    if isUpperCase(jsx.Tag[0]) {
        vnode.Type = VNodeComponent
    } else {
        vnode.Type = VNodeElement
    }

    // Convert attributes
    for _, attr := range jsx.Attrs {
        vnode.Props[attr.Name] = a.jsxValueToIRExpr(attr.Value)
    }

    // Convert children
    for _, child := range jsx.Children {
        childVNode := a.jsxNodeToVNode(child)
        vnode.Children = append(vnode.Children, childVNode)
    }

    return vnode
}

func (a *Analyzer) analyzeStyle(style *parser.StyleBlock, componentName string) *StyleIR {
    styleIR := &StyleIR{
        Scoped:      true,
        ComponentID: componentName,
        Rules:       []CSSRuleIR{},
    }

    for _, rule := range style.Rules {
        ruleIR := CSSRuleIR{
            Selector:   rule.Selector,
            Properties: make(map[string]string),
            Scoped:     true,
        }

        for _, prop := range rule.Properties {
            ruleIR.Properties[prop.Name] = prop.Value
        }

        styleIR.Rules = append(styleIR.Rules, ruleIR)
    }

    return styleIR
}
```

---

## Phase 3: SSR Transpiler

### 3.1 SSR Code Generator

```go
// pkg/transpiler/ssr/transpiler.go
package ssr

import (
    "bytes"
    "fmt"
    "go/format"
    "text/template"
    "gox/pkg/analyzer"
)

type SSRTranspiler struct {
    templates *template.Template
}

func New() *SSRTranspiler {
    return &SSRTranspiler{
        templates: loadTemplates(),
    }
}

func (t *SSRTranspiler) Transpile(ir *analyzer.ComponentIR) ([]byte, error) {
    var buf bytes.Buffer

    // Generate component struct
    buf.WriteString(t.generateComponentStruct(ir))
    buf.WriteString("\n\n")

    // Generate constructor
    buf.WriteString(t.generateConstructor(ir))
    buf.WriteString("\n\n")

    // Generate Render method
    buf.WriteString(t.generateRenderMethod(ir))
    buf.WriteString("\n\n")

    // Generate helper methods (setState, effects, etc.)
    buf.WriteString(t.generateHelperMethods(ir))

    // Format the generated code
    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return buf.Bytes(), fmt.Errorf("format error: %w", err)
    }

    return formatted, nil
}

func (t *SSRTranspiler) generateComponentStruct(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    buf.WriteString(fmt.Sprintf("type %s struct {\n", ir.Name))
    buf.WriteString("    *gox.Component\n")

    // Props
    for _, prop := range ir.Props {
        buf.WriteString(fmt.Sprintf("    %s %s\n", prop.Name, t.typeToString(prop.Type)))
    }

    // State
    for _, state := range ir.State {
        buf.WriteString(fmt.Sprintf("    %s %s\n", state.Name, t.typeToString(state.Type)))
    }

    // Refs
    for _, ref := range ir.Refs {
        buf.WriteString(fmt.Sprintf("    %s *gox.Ref[%s]\n", ref.Name, t.typeToString(ref.Type)))
    }

    buf.WriteString("}\n")

    return buf.String()
}

func (t *SSRTranspiler) generateConstructor(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    // Function signature
    buf.WriteString(fmt.Sprintf("func New%s(", ir.Name))
    for i, prop := range ir.Props {
        if i > 0 {
            buf.WriteString(", ")
        }
        buf.WriteString(fmt.Sprintf("%s %s", prop.Name, t.typeToString(prop.Type)))
    }
    buf.WriteString(fmt.Sprintf(") *%s {\n", ir.Name))

    // Create instance
    buf.WriteString(fmt.Sprintf("    c := &%s{\n", ir.Name))
    buf.WriteString("        Component: gox.NewComponent(),\n")
    for _, prop := range ir.Props {
        buf.WriteString(fmt.Sprintf("        %s: %s,\n", prop.Name, prop.Name))
    }
    buf.WriteString("    }\n\n")

    // Initialize state
    for _, state := range ir.State {
        buf.WriteString(fmt.Sprintf("    c.%s = %s\n", state.Name, t.exprToString(state.InitValue)))
    }

    // Initialize refs
    for _, ref := range ir.Refs {
        buf.WriteString(fmt.Sprintf("    c.%s = gox.NewRef[%s](%s)\n",
            ref.Name, t.typeToString(ref.Type), t.exprToString(ref.InitValue)))
    }

    buf.WriteString("    return c\n")
    buf.WriteString("}\n")

    return buf.String()
}

func (t *SSRTranspiler) generateRenderMethod(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    buf.WriteString(fmt.Sprintf("func (c *%s) Render() string {\n", ir.Name))

    // Generate template string from VNode IR
    templateStr := t.vnodeToTemplate(ir.RenderLogic.VNode)
    buf.WriteString(fmt.Sprintf("    return fmt.Sprintf(`%s`", templateStr))

    // Add format args
    args := t.collectTemplateArgs(ir.RenderLogic.VNode)
    for _, arg := range args {
        buf.WriteString(fmt.Sprintf(", %s", arg))
    }
    buf.WriteString(")\n")

    buf.WriteString("}\n")

    return buf.String()
}

func (t *SSRTranspiler) vnodeToTemplate(vnode *analyzer.VNodeIR) string {
    var buf bytes.Buffer

    switch vnode.Type {
    case analyzer.VNodeElement:
        // <tag props>children</tag>
        buf.WriteString(fmt.Sprintf("<%s", vnode.Tag))

        // Attributes
        for name, value := range vnode.Props {
            if value.Type == analyzer.IRExprLiteral {
                buf.WriteString(fmt.Sprintf(` %s="%s"`, name, value.Value))
            } else {
                buf.WriteString(fmt.Sprintf(` %s="%%v"`, name))
            }
        }

        buf.WriteString(">")

        // Children
        for _, child := range vnode.Children {
            buf.WriteString(t.vnodeToTemplate(child))
        }

        buf.WriteString(fmt.Sprintf("</%s>", vnode.Tag))

    case analyzer.VNodeComponent:
        // Component reference - need to call its Render()
        buf.WriteString("%s") // Placeholder for component render

    case analyzer.VNodeText:
        buf.WriteString("%s") // Text placeholder

    case analyzer.VNodeExpression:
        buf.WriteString("%v") // Expression placeholder
    }

    return buf.String()
}

func (t *SSRTranspiler) collectTemplateArgs(vnode *analyzer.VNodeIR) []string {
    var args []string

    // Collect dynamic values from props
    for _, value := range vnode.Props {
        if value.Type != analyzer.IRExprLiteral {
            args = append(args, t.irExprToGo(value))
        }
    }

    // Recursively collect from children
    for _, child := range vnode.Children {
        args = append(args, t.collectTemplateArgs(child)...)
    }

    return args
}

func (t *SSRTranspiler) generateHelperMethods(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    // Generate setState methods
    for _, state := range ir.State {
        buf.WriteString(fmt.Sprintf("func (c *%s) %s(value %s) {\n",
            ir.Name, state.Setter, t.typeToString(state.Type)))
        buf.WriteString(fmt.Sprintf("    c.%s = value\n", state.Name))
        buf.WriteString("    c.RequestUpdate()\n")
        buf.WriteString("}\n\n")
    }

    return buf.String()
}

func (t *SSRTranspiler) typeToString(expr ast.Expr) string {
    // Convert AST type expression to string
    // Handle basic types, slices, maps, pointers, etc.
}

func (t *SSRTranspiler) exprToString(expr ast.Expr) string {
    // Convert AST expression to Go code string
}

func (t *SSRTranspiler) irExprToGo(expr analyzer.IRExpr) string {
    // Convert IR expression to Go code
}
```

### 3.2 SSR Runtime

```go
// runtime/ssr/renderer.go
package ssr

import (
    "bytes"
    "html/template"
)

type Component interface {
    Render() string
}

type Renderer struct {
    components map[string]Component
}

func NewRenderer() *Renderer {
    return &Renderer{
        components: make(map[string]Component),
    }
}

func (r *Renderer) RenderComponent(comp Component) (string, error) {
    return comp.Render(), nil
}

func (r *Renderer) RenderToString(comp Component) string {
    return comp.Render()
}

func (r *Renderer) RenderToHTML(comp Component) (template.HTML, error) {
    html := comp.Render()
    return template.HTML(html), nil
}
```

---

## Phase 4: CSR/WASM Transpiler

### 4.1 WASM Code Generator

```go
// pkg/transpiler/csr/transpiler.go
package csr

import (
    "bytes"
    "fmt"
    "go/format"
    "gox/pkg/analyzer"
)

type CSRTranspiler struct {
    vdomBuilder *VDomBuilder
}

func New() *CSRTranspiler {
    return &CSRTranspiler{
        vdomBuilder: NewVDomBuilder(),
    }
}

func (t *CSRTranspiler) Transpile(ir *analyzer.ComponentIR) ([]byte, error) {
    var buf bytes.Buffer

    // Generate component struct (similar to SSR but with WASM-specific fields)
    buf.WriteString(t.generateComponentStruct(ir))
    buf.WriteString("\n\n")

    // Generate constructor
    buf.WriteString(t.generateConstructor(ir))
    buf.WriteString("\n\n")

    // Generate Render method that returns VNode
    buf.WriteString(t.generateRenderMethod(ir))
    buf.WriteString("\n\n")

    // Generate lifecycle methods
    buf.WriteString(t.generateLifecycleMethods(ir))
    buf.WriteString("\n\n")

    // Generate setState with re-render trigger
    buf.WriteString(t.generateStateManagement(ir))
    buf.WriteString("\n\n")

    // Generate effect hooks
    buf.WriteString(t.generateEffectHooks(ir))

    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return buf.Bytes(), fmt.Errorf("format error: %w", err)
    }

    return formatted, nil
}

func (t *CSRTranspiler) generateComponentStruct(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    buf.WriteString(fmt.Sprintf("type %s struct {\n", ir.Name))
    buf.WriteString("    *gox.Component\n")

    // Props
    for _, prop := range ir.Props {
        buf.WriteString(fmt.Sprintf("    %s %s\n", prop.Name, t.typeToString(prop.Type)))
    }

    // State
    for _, state := range ir.State {
        buf.WriteString(fmt.Sprintf("    %s %s\n", state.Name, t.typeToString(state.Type)))
    }

    // Refs
    for _, ref := range ir.Refs {
        buf.WriteString(fmt.Sprintf("    %s *gox.Ref[%s]\n", ref.Name, t.typeToString(ref.Type)))
    }

    // WASM-specific fields
    buf.WriteString("    domNode js.Value\n")
    buf.WriteString("    mounted bool\n")

    buf.WriteString("}\n")

    return buf.String()
}

func (t *CSRTranspiler) generateRenderMethod(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    buf.WriteString(fmt.Sprintf("func (c *%s) Render() *gox.VNode {\n", ir.Name))

    // Generate VNode tree from IR
    vnodeCode := t.vdomBuilder.BuildVNode(ir.RenderLogic.VNode, "c")
    buf.WriteString(fmt.Sprintf("    return %s\n", vnodeCode))

    buf.WriteString("}\n")

    return buf.String()
}

func (t *CSRTranspiler) generateLifecycleMethods(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    // Mount
    buf.WriteString(fmt.Sprintf("func (c *%s) Mount(parent js.Value) {\n", ir.Name))
    buf.WriteString("    if c.mounted { return }\n")
    buf.WriteString("    vnode := c.Render()\n")
    buf.WriteString("    c.domNode = gox.CreateElement(vnode)\n")
    buf.WriteString("    parent.Call(\"appendChild\", c.domNode)\n")
    buf.WriteString("    c.mounted = true\n")
    buf.WriteString("    c.ComponentDidMount()\n")
    buf.WriteString("}\n\n")

    // Update
    buf.WriteString(fmt.Sprintf("func (c *%s) Update() {\n", ir.Name))
    buf.WriteString("    if !c.mounted { return }\n")
    buf.WriteString("    newVNode := c.Render()\n")
    buf.WriteString("    oldVNode := c.currentVNode\n")
    buf.WriteString("    gox.Patch(c.domNode, oldVNode, newVNode)\n")
    buf.WriteString("    c.currentVNode = newVNode\n")
    buf.WriteString("}\n\n")

    // Unmount
    buf.WriteString(fmt.Sprintf("func (c *%s) Unmount() {\n", ir.Name))
    buf.WriteString("    if !c.mounted { return }\n")
    buf.WriteString("    c.ComponentWillUnmount()\n")
    buf.WriteString("    c.domNode.Call(\"remove\")\n")
    buf.WriteString("    c.mounted = false\n")
    buf.WriteString("}\n\n")

    // DidMount hook
    buf.WriteString(fmt.Sprintf("func (c *%s) ComponentDidMount() {\n", ir.Name))
    for _, effect := range ir.Effects {
        if len(effect.Deps) == 0 {
            // Run once on mount
            buf.WriteString(t.blockStmtToString(&effect.Setup))
        }
    }
    buf.WriteString("}\n\n")

    // WillUnmount hook
    buf.WriteString(fmt.Sprintf("func (c *%s) ComponentWillUnmount() {\n", ir.Name))
    for _, effect := range ir.Effects {
        if effect.Cleanup.List != nil {
            buf.WriteString(t.blockStmtToString(&effect.Cleanup))
        }
    }
    buf.WriteString("}\n\n")

    return buf.String()
}

func (t *CSRTranspiler) generateStateManagement(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    for _, state := range ir.State {
        buf.WriteString(fmt.Sprintf("func (c *%s) %s(value %s) {\n",
            ir.Name, state.Setter, t.typeToString(state.Type)))
        buf.WriteString(fmt.Sprintf("    c.%s = value\n", state.Name))
        buf.WriteString("    c.Update() // Trigger re-render\n")
        buf.WriteString("}\n\n")
    }

    return buf.String()
}

func (t *CSRTranspiler) generateEffectHooks(ir *analyzer.ComponentIR) string {
    var buf bytes.Buffer

    buf.WriteString(fmt.Sprintf("func (c *%s) RunEffects() {\n", ir.Name))

    for i, effect := range ir.Effects {
        if len(effect.Deps) > 0 {
            // Check if dependencies changed
            buf.WriteString(fmt.Sprintf("    if c.depsChanged%d() {\n", i))
            buf.WriteString(t.blockStmtToString(&effect.Setup))
            buf.WriteString("    }\n")
        }
    }

    buf.WriteString("}\n\n")

    return buf.String()
}
```

### 4.2 VNode Builder

```go
// pkg/transpiler/csr/vdom_builder.go
package csr

import (
    "fmt"
    "strings"
    "gox/pkg/analyzer"
)

type VDomBuilder struct{}

func NewVDomBuilder() *VDomBuilder {
    return &VDomBuilder{}
}

func (b *VDomBuilder) BuildVNode(vnode *analyzer.VNodeIR, contextVar string) string {
    switch vnode.Type {
    case analyzer.VNodeElement:
        return b.buildElement(vnode, contextVar)
    case analyzer.VNodeComponent:
        return b.buildComponent(vnode, contextVar)
    case analyzer.VNodeText:
        return b.buildText(vnode, contextVar)
    case analyzer.VNodeExpression:
        return b.buildExpression(vnode, contextVar)
    default:
        return "nil"
    }
}

func (b *VDomBuilder) buildElement(vnode *analyzer.VNodeIR, contextVar string) string {
    var buf strings.Builder

    buf.WriteString("gox.H(")
    buf.WriteString(fmt.Sprintf(`"%s"`, vnode.Tag))

    // Props
    if len(vnode.Props) > 0 {
        buf.WriteString(", gox.Props{")
        for name, value := range vnode.Props {
            buf.WriteString(fmt.Sprintf(`"%s": %s, `, name, b.irExprToGo(value, contextVar)))
        }
        buf.WriteString("}")
    } else {
        buf.WriteString(", nil")
    }

    // Children
    if len(vnode.Children) > 0 {
        buf.WriteString(", ")
        for i, child := range vnode.Children {
            if i > 0 {
                buf.WriteString(", ")
            }
            buf.WriteString(b.BuildVNode(child, contextVar))
        }
    }

    buf.WriteString(")")

    return buf.String()
}

func (b *VDomBuilder) buildComponent(vnode *analyzer.VNodeIR, contextVar string) string {
    var buf strings.Builder

    buf.WriteString(fmt.Sprintf("New%s(", vnode.Tag))

    // Pass props as arguments
    first := true
    for name, value := range vnode.Props {
        if !first {
            buf.WriteString(", ")
        }
        buf.WriteString(b.irExprToGo(value, contextVar))
        first = false
    }

    buf.WriteString(").Render()")

    return buf.String()
}

func (b *VDomBuilder) buildText(vnode *analyzer.VNodeIR, contextVar string) string {
    return fmt.Sprintf(`gox.Text("%s")`, vnode.Tag)
}

func (b *VDomBuilder) buildExpression(vnode *analyzer.VNodeIR, contextVar string) string {
    // Dynamic expression
    return fmt.Sprintf("gox.Text(fmt.Sprint(%s))", b.irExprToGo(vnode.Props["value"], contextVar))
}

func (b *VDomBuilder) irExprToGo(expr analyzer.IRExpr, contextVar string) string {
    switch expr.Type {
    case analyzer.IRExprLiteral:
        return fmt.Sprintf(`"%v"`, expr.Value)
    case analyzer.IRExprIdentifier:
        ident := expr.Value.(string)
        return fmt.Sprintf("%s.%s", contextVar, ident)
    case analyzer.IRExprMember:
        // Handle c.state.field access
        return b.astExprToString(expr.Original)
    case analyzer.IRExprCall:
        return b.astExprToString(expr.Original)
    default:
        return fmt.Sprintf("%v", expr.Value)
    }
}

func (b *VDomBuilder) astExprToString(expr ast.Expr) string {
    // Convert AST expression to Go code string
    // Use go/printer or custom logic
}
```

---

## Phase 5: Runtime Library

### 5.1 Core Runtime Types

```go
// runtime/gox.go
package gox

import (
    "syscall/js"
)

// Component is the base type for all GoX components
type Component struct {
    id          string
    currentVNode *VNode
    hooks       *HookState
    context     map[string]interface{}
}

func NewComponent() *Component {
    return &Component{
        hooks:   NewHookState(),
        context: make(map[string]interface{}),
    }
}

func (c *Component) RequestUpdate() {
    // Queue a re-render
    scheduleUpdate(c)
}

// VNode represents a virtual DOM node
type VNode struct {
    Type     VNodeType
    Tag      string
    Props    Props
    Children []*VNode
    Key      string
    Ref      *Ref[js.Value]
}

type VNodeType int

const (
    VNodeTypeElement VNodeType = iota
    VNodeTypeComponent
    VNodeTypeText
    VNodeTypeFragment
)

type Props map[string]interface{}

// Helper functions for creating VNodes
func H(tag string, props Props, children ...*VNode) *VNode {
    return &VNode{
        Type:     VNodeTypeElement,
        Tag:      tag,
        Props:    props,
        Children: children,
    }
}

func Text(content string) *VNode {
    return &VNode{
        Type: VNodeTypeText,
        Tag:  content,
    }
}

func Fragment(children ...*VNode) *VNode {
    return &VNode{
        Type:     VNodeTypeFragment,
        Children: children,
    }
}

// Ref implementation
type Ref[T any] struct {
    Current T
}

func NewRef[T any](initial T) *Ref[T] {
    return &Ref[T]{Current: initial}
}

// Context implementation
type Context[T any] struct {
    defaultValue T
    Provider     *ContextProvider[T]
}

type ContextProvider[T any] struct {
    value T
}

func CreateContext[T any](defaultValue T) *Context[T] {
    return &Context[T]{
        defaultValue: defaultValue,
    }
}

func (ctx *Context[T]) Provide(value T) *ContextProvider[T] {
    return &ContextProvider[T]{value: value}
}
```

### 5.2 Hook System

```go
// runtime/hooks.go
package gox

import (
    "reflect"
)

type HookState struct {
    states    []interface{}
    effects   []Effect
    memos     []Memo
    callbacks []Callback
    refs      []interface{}
    index     int
}

func NewHookState() *HookState {
    return &HookState{
        states:    []interface{}{},
        effects:   []Effect{},
        memos:     []Memo{},
        callbacks: []Callback{},
        refs:      []interface{}{},
        index:     0,
    }
}

func (h *HookState) Reset() {
    h.index = 0
}

// useState implementation
var currentComponent *Component

func UseState[T any](initial T) (T, func(T)) {
    comp := currentComponent
    idx := comp.hooks.index
    comp.hooks.index++

    // Initialize on first render
    if idx >= len(comp.hooks.states) {
        comp.hooks.states = append(comp.hooks.states, initial)
    }

    state := comp.hooks.states[idx].(T)

    setter := func(newValue T) {
        comp.hooks.states[idx] = newValue
        comp.RequestUpdate()
    }

    return state, setter
}

// useEffect implementation
type Effect struct {
    Setup   func() func()
    Cleanup func()
    Deps    []interface{}
}

func UseEffect(setup func() func(), deps []interface{}) {
    comp := currentComponent
    idx := comp.hooks.index
    comp.hooks.index++

    // Check if deps changed
    if idx >= len(comp.hooks.effects) {
        // First run
        cleanup := setup()
        comp.hooks.effects = append(comp.hooks.effects, Effect{
            Setup:   setup,
            Cleanup: cleanup,
            Deps:    deps,
        })
    } else {
        effect := comp.hooks.effects[idx]
        if depsChanged(effect.Deps, deps) {
            // Run cleanup
            if effect.Cleanup != nil {
                effect.Cleanup()
            }
            // Run setup
            cleanup := setup()
            comp.hooks.effects[idx] = Effect{
                Setup:   setup,
                Cleanup: cleanup,
                Deps:    deps,
            }
        }
    }
}

// useMemo implementation
type Memo struct {
    Value interface{}
    Deps  []interface{}
}

func UseMemo[T any](compute func() T, deps []interface{}) T {
    comp := currentComponent
    idx := comp.hooks.index
    comp.hooks.index++

    if idx >= len(comp.hooks.memos) {
        // First run
        value := compute()
        comp.hooks.memos = append(comp.hooks.memos, Memo{
            Value: value,
            Deps:  deps,
        })
        return value
    }

    memo := comp.hooks.memos[idx]
    if depsChanged(memo.Deps, deps) {
        value := compute()
        comp.hooks.memos[idx] = Memo{
            Value: value,
            Deps:  deps,
        }
        return value
    }

    return memo.Value.(T)
}

// useCallback implementation
type Callback struct {
    Func interface{}
    Deps []interface{}
}

func UseCallback[T any](callback T, deps []interface{}) T {
    comp := currentComponent
    idx := comp.hooks.index
    comp.hooks.index++

    if idx >= len(comp.hooks.callbacks) {
        comp.hooks.callbacks = append(comp.hooks.callbacks, Callback{
            Func: callback,
            Deps: deps,
        })
        return callback
    }

    cb := comp.hooks.callbacks[idx]
    if depsChanged(cb.Deps, deps) {
        comp.hooks.callbacks[idx] = Callback{
            Func: callback,
            Deps: deps,
        }
        return callback
    }

    return cb.Func.(T)
}

// useRef implementation
func UseRef[T any](initial T) *Ref[T] {
    comp := currentComponent
    idx := comp.hooks.index
    comp.hooks.index++

    if idx >= len(comp.hooks.refs) {
        ref := NewRef(initial)
        comp.hooks.refs = append(comp.hooks.refs, ref)
        return ref
    }

    return comp.hooks.refs[idx].(*Ref[T])
}

// useContext implementation
func UseContext[T any](ctx *Context[T]) T {
    comp := currentComponent

    // Look up context value from component tree
    if provider, ok := comp.context[reflect.TypeOf(ctx).String()]; ok {
        return provider.(*ContextProvider[T]).value
    }

    return ctx.defaultValue
}

// Helper function
func depsChanged(oldDeps, newDeps []interface{}) bool {
    if len(oldDeps) != len(newDeps) {
        return true
    }

    for i := range oldDeps {
        if !reflect.DeepEqual(oldDeps[i], newDeps[i]) {
            return true
        }
    }

    return false
}
```

---

## Phase 6: Virtual DOM (WASM)

### 6.1 DOM Manipulation

```go
// runtime/wasm/dom.go
//go:build js && wasm

package wasm

import (
    "syscall/js"
    "gox"
)

var document js.Value

func init() {
    document = js.Global().Get("document")
}

// CreateElement creates a real DOM element from VNode
func CreateElement(vnode *gox.VNode) js.Value {
    if vnode == nil {
        return js.Null()
    }

    switch vnode.Type {
    case gox.VNodeTypeElement:
        return createHTMLElement(vnode)
    case gox.VNodeTypeText:
        return createTextNode(vnode)
    case gox.VNodeTypeFragment:
        return createFragment(vnode)
    default:
        return js.Null()
    }
}

func createHTMLElement(vnode *gox.VNode) js.Value {
    elem := document.Call("createElement", vnode.Tag)

    // Set attributes
    for key, value := range vnode.Props {
        setProperty(elem, key, value)
    }

    // Append children
    for _, child := range vnode.Children {
        childElem := CreateElement(child)
        if !childElem.IsNull() {
            elem.Call("appendChild", childElem)
        }
    }

    return elem
}

func createTextNode(vnode *gox.VNode) js.Value {
    return document.Call("createTextNode", vnode.Tag)
}

func createFragment(vnode *gox.VNode) js.Value {
    fragment := document.Call("createDocumentFragment")

    for _, child := range vnode.Children {
        childElem := CreateElement(child)
        if !childElem.IsNull() {
            fragment.Call("appendChild", childElem)
        }
    }

    return fragment
}

func setProperty(elem js.Value, key string, value interface{}) {
    switch key {
    case "className":
        elem.Set("className", value)
    case "style":
        setStyle(elem, value)
    case "onClick", "onChange", "onInput", "onSubmit":
        // Event handlers
        setEventListener(elem, key, value)
    default:
        // Regular attributes
        if key[:2] == "on" {
            setEventListener(elem, key, value)
        } else {
            elem.Call("setAttribute", key, value)
        }
    }
}

func setStyle(elem js.Value, style interface{}) {
    styleMap := style.(map[string]interface{})
    styleObj := elem.Get("style")

    for prop, value := range styleMap {
        styleObj.Set(prop, value)
    }
}

func setEventListener(elem js.Value, eventName string, handler interface{}) {
    // Convert "onClick" to "click"
    event := strings.ToLower(eventName[2:])

    callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        if fn, ok := handler.(func(js.Value)); ok {
            fn(args[0])
        }
        return nil
    })

    elem.Call("addEventListener", event, callback)
}
```

### 6.2 Diffing Algorithm

```go
// runtime/wasm/diff.go
//go:build js && wasm

package wasm

import (
    "syscall/js"
    "gox"
)

// Patch updates the real DOM to match the new VNode
func Patch(parent js.Value, oldVNode, newVNode *gox.VNode) {
    // Case 1: New node doesn't exist - remove old
    if newVNode == nil {
        if oldVNode != nil {
            parent.Call("removeChild", parent.Get("firstChild"))
        }
        return
    }

    // Case 2: Old node doesn't exist - create new
    if oldVNode == nil {
        elem := CreateElement(newVNode)
        parent.Call("appendChild", elem)
        return
    }

    // Case 3: Different types - replace
    if oldVNode.Type != newVNode.Type || oldVNode.Tag != newVNode.Tag {
        elem := CreateElement(newVNode)
        parent.Call("replaceChild", elem, parent.Get("firstChild"))
        return
    }

    // Case 4: Same type - update
    switch newVNode.Type {
    case gox.VNodeTypeElement:
        patchElement(parent.Get("firstChild"), oldVNode, newVNode)
    case gox.VNodeTypeText:
        patchText(parent.Get("firstChild"), oldVNode, newVNode)
    }
}

func patchElement(elem js.Value, oldVNode, newVNode *gox.VNode) {
    // Update properties
    patchProps(elem, oldVNode.Props, newVNode.Props)

    // Update children
    patchChildren(elem, oldVNode.Children, newVNode.Children)
}

func patchProps(elem js.Value, oldProps, newProps gox.Props) {
    // Remove old props
    for key := range oldProps {
        if _, exists := newProps[key]; !exists {
            removeProperty(elem, key)
        }
    }

    // Add/update new props
    for key, value := range newProps {
        oldValue, exists := oldProps[key]
        if !exists || oldValue != value {
            setProperty(elem, key, value)
        }
    }
}

func patchChildren(parent js.Value, oldChildren, newChildren []*gox.VNode) {
    // Simple algorithm - can be optimized with keys
    maxLen := len(oldChildren)
    if len(newChildren) > maxLen {
        maxLen = len(newChildren)
    }

    for i := 0; i < maxLen; i++ {
        var oldChild, newChild *gox.VNode

        if i < len(oldChildren) {
            oldChild = oldChildren[i]
        }
        if i < len(newChildren) {
            newChild = newChildren[i]
        }

        if oldChild == nil && newChild != nil {
            // Add new child
            elem := CreateElement(newChild)
            parent.Call("appendChild", elem)
        } else if oldChild != nil && newChild == nil {
            // Remove old child
            parent.Call("removeChild", parent.Get("childNodes").Index(i))
        } else if oldChild != nil && newChild != nil {
            // Patch existing child
            childNode := parent.Get("childNodes").Index(i)
            patchNode(childNode, oldChild, newChild)
        }
    }
}

func patchNode(node js.Value, oldVNode, newVNode *gox.VNode) {
    if oldVNode.Type != newVNode.Type || oldVNode.Tag != newVNode.Tag {
        // Replace
        newElem := CreateElement(newVNode)
        node.Get("parentNode").Call("replaceChild", newElem, node)
        return
    }

    switch newVNode.Type {
    case gox.VNodeTypeElement:
        patchElement(node, oldVNode, newVNode)
    case gox.VNodeTypeText:
        patchText(node, oldVNode, newVNode)
    }
}

func patchText(node js.Value, oldVNode, newVNode *gox.VNode) {
    if oldVNode.Tag != newVNode.Tag {
        node.Set("textContent", newVNode.Tag)
    }
}

func removeProperty(elem js.Value, key string) {
    switch key {
    case "className":
        elem.Set("className", "")
    case "style":
        elem.Set("style", "")
    default:
        elem.Call("removeAttribute", key)
    }
}
```

### 6.3 Reconciler & Scheduler

```go
// runtime/wasm/reconciler.go
//go:build js && wasm

package wasm

import (
    "syscall/js"
    "gox"
)

type Reconciler struct {
    rootComponent *gox.Component
    rootElement   js.Value
    updateQueue   []*gox.Component
    isRendering   bool
}

func NewReconciler(root *gox.Component, container js.Value) *Reconciler {
    return &Reconciler{
        rootComponent: root,
        rootElement:   container,
        updateQueue:   []*gox.Component{},
    }
}

func (r *Reconciler) Mount() {
    vnode := r.renderComponent(r.rootComponent)
    elem := CreateElement(vnode)
    r.rootElement.Call("appendChild", elem)
    r.rootComponent.currentVNode = vnode
}

func (r *Reconciler) ScheduleUpdate(comp *gox.Component) {
    r.updateQueue = append(r.updateQueue, comp)

    if !r.isRendering {
        r.processUpdates()
    }
}

func (r *Reconciler) processUpdates() {
    r.isRendering = true

    // Use requestAnimationFrame for batched updates
    js.Global().Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        for len(r.updateQueue) > 0 {
            comp := r.updateQueue[0]
            r.updateQueue = r.updateQueue[1:]

            r.updateComponent(comp)
        }

        r.isRendering = false
        return nil
    }))
}

func (r *Reconciler) updateComponent(comp *gox.Component) {
    oldVNode := comp.currentVNode
    newVNode := r.renderComponent(comp)

    // Find DOM node for this component
    // (simplified - would need component-to-DOM mapping)
    Patch(r.rootElement, oldVNode, newVNode)

    comp.currentVNode = newVNode
}

func (r *Reconciler) renderComponent(comp *gox.Component) *gox.VNode {
    // Set current component for hooks
    gox.currentComponent = comp

    // Reset hook index
    comp.hooks.Reset()

    // Call render (would be generated by transpiler)
    // This is a placeholder - actual render would call component's Render method
    vnode := &gox.VNode{
        Type: gox.VNodeTypeElement,
        Tag:  "div",
        Props: gox.Props{},
        Children: []*gox.VNode{},
    }

    return vnode
}
```

---

## Phase 7: Build Toolchain

### 7.1 CLI Tool

```go
// cmd/goxc/main.go
package main

import (
    "flag"
    "fmt"
    "os"
    "gox/pkg/cli"
)

func main() {
    buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
    mode := buildCmd.String("mode", "ssr", "Compilation mode: ssr or csr")
    output := buildCmd.String("o", "dist", "Output directory")

    watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)

    devCmd := flag.NewFlagSet("dev", flag.ExitOnError)
    port := devCmd.Int("port", 3000, "Development server port")

    if len(os.Args) < 2 {
        fmt.Println("Usage: goxc [build|watch|dev] [options]")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "build":
        buildCmd.Parse(os.Args[2:])
        cli.Build(*mode, *output, buildCmd.Args())
    case "watch":
        watchCmd.Parse(os.Args[2:])
        cli.Watch(watchCmd.Args())
    case "dev":
        devCmd.Parse(os.Args[2:])
        cli.Dev(*port, devCmd.Args())
    default:
        fmt.Printf("Unknown command: %s\n", os.Args[1])
        os.Exit(1)
    }
}
```

### 7.2 Build System

```go
// pkg/cli/build.go
package cli

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "gox/pkg/lexer"
    "gox/pkg/parser"
    "gox/pkg/analyzer"
    "gox/pkg/transpiler/ssr"
    "gox/pkg/transpiler/csr"
)

func Build(mode, outputDir string, files []string) error {
    fmt.Printf("Building in %s mode...\n", mode)

    // Create output directory
    os.MkdirAll(outputDir, 0755)

    for _, file := range files {
        if err := buildFile(file, mode, outputDir); err != nil {
            return fmt.Errorf("failed to build %s: %w", file, err)
        }
    }

    // Generate entry point
    if mode == "csr" {
        if err := generateWASMEntry(outputDir); err != nil {
            return err
        }
    }

    fmt.Println("Build complete!")
    return nil
}

func buildFile(file, mode, outputDir string) error {
    // Read source
    source, err := ioutil.ReadFile(file)
    if err != nil {
        return err
    }

    // Lex & Parse
    l := lexer.New(source, file)
    p := parser.New(l)
    goFile, components, err := p.ParseFile()
    if err != nil {
        return err
    }

    // Analyze
    a := analyzer.New()
    irs, err := a.Analyze(goFile, components)
    if err != nil {
        return err
    }

    // Transpile
    for _, ir := range irs {
        var generated []byte

        switch mode {
        case "ssr":
            transpiler := ssr.New()
            generated, err = transpiler.Transpile(ir)
        case "csr":
            transpiler := csr.New()
            generated, err = transpiler.Transpile(ir)
        default:
            return fmt.Errorf("unknown mode: %s", mode)
        }

        if err != nil {
            return err
        }

        // Write output
        outputFile := filepath.Join(outputDir, filepath.Base(file))
        outputFile = outputFile[:len(outputFile)-3] + "go" // .gox -> .go

        if err := ioutil.WriteFile(outputFile, generated, 0644); err != nil {
            return err
        }

        fmt.Printf("  Generated %s\n", outputFile)
    }

    return nil
}

func generateWASMEntry(outputDir string) error {
    entry := `
package main

import (
    "syscall/js"
    "gox/runtime/wasm"
)

func main() {
    // Initialize GoX runtime
    wasm.Init()

    // Mount root component
    root := js.Global().Get("document").Call("getElementById", "root")

    // TODO: Auto-generate based on main component
    // app := NewApp()
    // reconciler := wasm.NewReconciler(app, root)
    // reconciler.Mount()

    // Keep the program running
    select {}
}
`

    return ioutil.WriteFile(filepath.Join(outputDir, "main.go"), []byte(entry), 0644)
}
```

### 7.3 File Watcher

```go
// pkg/cli/watch.go
package cli

import (
    "fmt"
    "log"
    "path/filepath"
    "github.com/fsnotify/fsnotify"
)

func Watch(dirs []string) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()

    for _, dir := range dirs {
        if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if info.IsDir() {
                return watcher.Add(path)
            }
            return nil
        }); err != nil {
            return err
        }
    }

    fmt.Println("Watching for changes...")

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                if filepath.Ext(event.Name) == ".gox" {
                    fmt.Printf("\nChange detected: %s\n", event.Name)
                    if err := buildFile(event.Name, "ssr", "dist"); err != nil {
                        log.Printf("Build error: %v\n", err)
                    }
                }
            }
        case err := <-watcher.Errors:
            log.Printf("Watcher error: %v\n", err)
        }
    }
}
```

### 7.4 Dev Server

```go
// pkg/cli/serve.go
package cli

import (
    "fmt"
    "net/http"
    "log"
)

func Dev(port int, dirs []string) error {
    // Start file watcher in background
    go Watch(dirs)

    // Serve static files
    fs := http.FileServer(http.Dir("dist"))
    http.Handle("/", fs)

    addr := fmt.Sprintf(":%d", port)
    fmt.Printf("Starting dev server on http://localhost%s\n", addr)

    return http.ListenAndServe(addr, nil)
}
```

---

## Phase 8: CSS Management

### 8.1 CSS Extraction & Scoping

```go
// pkg/transpiler/css/processor.go
package css

import (
    "crypto/sha256"
    "fmt"
    "strings"
    "gox/pkg/analyzer"
)

type CSSProcessor struct {
    styles map[string]string // component name -> CSS
}

func New() *CSSProcessor {
    return &CSSProcessor{
        styles: make(map[string]string),
    }
}

func (p *CSSProcessor) Process(ir *analyzer.ComponentIR) string {
    if ir.Styles == nil {
        return ""
    }

    var css strings.Builder

    for _, rule := range ir.Styles.Rules {
        if ir.Styles.Scoped {
            // Add component scope
            selector := p.scopeSelector(rule.Selector, ir.Styles.ComponentID)
            css.WriteString(selector)
        } else {
            css.WriteString(rule.Selector)
        }

        css.WriteString(" {\n")

        for prop, value := range rule.Properties {
            css.WriteString(fmt.Sprintf("  %s: %s;\n", prop, value))
        }

        css.WriteString("}\n\n")
    }

    return css.String()
}

func (p *CSSProcessor) scopeSelector(selector, componentID string) string {
    // Generate unique data attribute
    hash := sha256.Sum256([]byte(componentID))
    scopeID := fmt.Sprintf("gox-%x", hash[:8])

    // Transform selector to include scope
    // .btn -> .btn[data-gox-abc123]
    parts := strings.Split(selector, ",")
    for i, part := range parts {
        part = strings.TrimSpace(part)
        parts[i] = fmt.Sprintf("%s[data-%s]", part, scopeID)
    }

    return strings.Join(parts, ", ")
}

func (p *CSSProcessor) Bundle(components []*analyzer.ComponentIR) string {
    var bundle strings.Builder

    for _, ir := range components {
        css := p.Process(ir)
        bundle.WriteString(css)
    }

    return bundle.String()
}
```

---

## Phase 9: Developer Experience

### 9.1 Error Messages

```go
// pkg/errors/reporter.go
package errors

import (
    "fmt"
    "strings"
)

type Error struct {
    File    string
    Line    int
    Column  int
    Message string
    Code    string
    Hint    string
}

func (e *Error) Error() string {
    var buf strings.Builder

    buf.WriteString(fmt.Sprintf("\n%s:%d:%d: error: %s\n", e.File, e.Line, e.Column, e.Message))

    if e.Code != "" {
        buf.WriteString(fmt.Sprintf("\n  %s\n", e.Code))
        buf.WriteString(fmt.Sprintf("  %s^\n", strings.Repeat(" ", e.Column-1)))
    }

    if e.Hint != "" {
        buf.WriteString(fmt.Sprintf("\nHint: %s\n", e.Hint))
    }

    return buf.String()
}
```

### 9.2 Debug Mode

```go
// runtime/debug.go
//go:build debug

package gox

import "log"

func Debug(format string, args ...interface{}) {
    log.Printf("[GoX] "+format, args...)
}

func TraceRender(component string) {
    log.Printf("[GoX] Rendering %s", component)
}

func TraceUpdate(component string, prop string) {
    log.Printf("[GoX] Updating %s.%s", component, prop)
}
```

---

## Implementation Timeline

### Phase 1: Foundation (Weeks 1-3)
- [ ] Basic lexer for Go + GoX keywords
- [ ] Parser for `component` keyword
- [ ] AST structures for components
- [ ] Simple transpiler (SSR only) for basic components
- [ ] Basic runtime (Component base type)

### Phase 2: Hooks & State (Weeks 4-6)
- [ ] useState implementation (transpiler + runtime)
- [ ] useEffect implementation
- [ ] Hook state management
- [ ] Re-render triggering

### Phase 3: JSX & Rendering (Weeks 7-9)
- [ ] JSX lexer/parser
- [ ] VNode IR
- [ ] SSR code generation for JSX
- [ ] Template rendering

### Phase 4: WASM Foundation (Weeks 10-12)
- [ ] Virtual DOM structures
- [ ] createElement implementation
- [ ] Basic diffing algorithm
- [ ] WASM event handlers

### Phase 5: CSR Transpiler (Weeks 13-15)
- [ ] CSR code generation
- [ ] Component lifecycle methods
- [ ] Hook implementation for WASM
- [ ] Reconciler

### Phase 6: Advanced Features (Weeks 16-18)
- [ ] useMemo, useCallback, useRef
- [ ] Context API
- [ ] CSS processing & scoping
- [ ] Fragment support

### Phase 7: Build Tooling (Weeks 19-21)
- [ ] CLI tool (goxc)
- [ ] File watcher
- [ ] Dev server with live reload
- [ ] Production build optimization

### Phase 8: Polish & DX (Weeks 22-24)
- [ ] Error messages
- [ ] Debug mode
- [ ] Documentation
- [ ] Example projects
- [ ] Testing utilities

---

## Example .gox File

```gox
package main

import (
    "fmt"
    "gox"
)

component Counter(initialCount int) {
    count, setCount := gox.UseState[int](initialCount)
    clicks, setClicks := gox.UseState[int](0)

    gox.UseEffect(func() func() {
        fmt.Println("Counter mounted")
        return func() {
            fmt.Println("Counter unmounted")
        }
    }, []interface{}{})

    handleIncrement := func() {
        setCount(count + 1)
        setClicks(clicks + 1)
    }

    style {
        .counter {
            padding: 20px;
            border: 1px solid #ccc;
            border-radius: 4px;
        }

        .count {
            font-size: 24px;
            font-weight: bold;
            color: #333;
        }

        .button {
            background: blue;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }

        .button:hover {
            background: darkblue;
        }
    }

    render {
        <div className="counter">
            <h2>Counter Component</h2>
            <div className="count">Count: {count}</div>
            <button className="button" onClick={handleIncrement}>
                Increment
            </button>
            <p>Button clicked {clicks} times</p>
        </div>
    }
}
```

**Generated SSR Output:**

```go
package main

import (
    "fmt"
    "gox"
)

type Counter struct {
    *gox.Component
    initialCount int
    count int
    clicks int
}

func NewCounter(initialCount int) *Counter {
    c := &Counter{
        Component: gox.NewComponent(),
        initialCount: initialCount,
    }
    c.count = initialCount
    c.clicks = 0
    return c
}

func (c *Counter) Render() string {
    return fmt.Sprintf(`<div class="counter" data-gox-abc123>
        <h2>Counter Component</h2>
        <div class="count">Count: %d</div>
        <button class="button">Increment</button>
        <p>Button clicked %d times</p>
    </div>`, c.count, c.clicks)
}

func (c *Counter) setCount(value int) {
    c.count = value
    c.RequestUpdate()
}

func (c *Counter) setClicks(value int) {
    c.clicks = value
    c.RequestUpdate()
}
```

**Generated CSR Output:**

```go
package main

import (
    "fmt"
    "syscall/js"
    "gox"
)

type Counter struct {
    *gox.Component
    initialCount int
    count int
    clicks int
    domNode js.Value
    mounted bool
}

func NewCounter(initialCount int) *Counter {
    c := &Counter{
        Component: gox.NewComponent(),
        initialCount: initialCount,
    }
    c.count = initialCount
    c.clicks = 0
    return c
}

func (c *Counter) Render() *gox.VNode {
    handleIncrement := func(e js.Value) {
        c.setCount(c.count + 1)
        c.setClicks(c.clicks + 1)
    }

    return gox.H("div", gox.Props{"className": "counter"},
        gox.H("h2", nil, gox.Text("Counter Component")),
        gox.H("div", gox.Props{"className": "count"},
            gox.Text(fmt.Sprintf("Count: %d", c.count)),
        ),
        gox.H("button", gox.Props{
            "className": "button",
            "onClick": handleIncrement,
        }, gox.Text("Increment")),
        gox.H("p", nil,
            gox.Text(fmt.Sprintf("Button clicked %d times", c.clicks)),
        ),
    )
}

func (c *Counter) setCount(value int) {
    c.count = value
    c.Update()
}

func (c *Counter) setClicks(value int) {
    c.clicks = value
    c.Update()
}

func (c *Counter) Mount(parent js.Value) { /* ... */ }
func (c *Counter) Update() { /* ... */ }
func (c *Counter) Unmount() { /* ... */ }
```

---

## Next Steps

1. **Start with Phase 1**: Build the lexer/parser for basic component syntax
2. **Create proof-of-concept**: Simple component that compiles to SSR
3. **Iterate**: Add one feature at a time (useState, JSX, effects, etc.)
4. **Test extensively**: Write tests for each phase
5. **Document**: Keep API docs and examples up to date
6. **Community**: Open source and gather feedback early

This is an ambitious project that will take significant time and effort, but the architecture is sound and the implementation is feasible. Start small, test often, and iterate based on real usage.
