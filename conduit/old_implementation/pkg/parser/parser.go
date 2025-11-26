package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/user/gox/pkg/lexer"
)

// Parser parses GoX source code
type Parser struct {
	lexer      *lexer.Lexer
	curToken   lexer.Token
	peekToken  lexer.Token
	errors     []error
	fset       *token.FileSet
	filename   string
}

// New creates a new Parser instance
func New(l *lexer.Lexer, filename string) *Parser {
	p := &Parser{
		lexer:    l,
		fset:     token.NewFileSet(),
		filename: filename,
		errors:   []error{},
	}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// ParseFile parses a complete GoX file
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
		} else {
			p.addError("expected package name")
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
			// Parse regular Go functions
			// For now, skip to the end of the function
			p.skipToNextDeclaration()
		case lexer.TYPE:
			// Parse type declarations
			// For now, skip to the end of the type
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

// parseComponent parses a component declaration
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

// parseComponentBody parses the body of a component
func (p *Parser) parseComponentBody() *ComponentBody {
	body := &ComponentBody{
		Hooks: []*HookCall{},
		Stmts: []ast.Stmt{},
	}

	p.nextToken() // consume '{'

	braceCount := 1
	for braceCount > 0 && !p.curTokenIs(lexer.EOF) {
		switch p.curToken.Type {
		case lexer.LBRACE:
			braceCount++
			p.nextToken()
		case lexer.RBRACE:
			braceCount--
			if braceCount == 0 {
				break
			}
			p.nextToken()
		case lexer.RENDER:
			p.nextToken()
			body.Render = p.parseRenderBlock()
		case lexer.STYLE:
			p.nextToken()
			body.Style = p.parseStyleBlock()
		case lexer.IDENT:
			// Try to parse as a hook statement (e.g., count, setCount := gox.UseState[int](0))
			hook := p.parseHookStatement()
			if hook != nil {
				body.Hooks = append(body.Hooks, hook)
			} else {
				// Not a hook, skip the entire statement
				p.skipStatement()
			}
		default:
			p.nextToken()
		}
	}

	p.nextToken() // consume final '}'

	return body
}

// parseHookStatement parses a complete hook statement including assignment
// e.g., count, setCount := gox.UseState[int](0)
func (p *Parser) parseHookStatement() *HookCall {
	// Collect left-hand side identifiers
	var results []string

	// First identifier
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}
	results = append(results, p.curToken.Literal)
	p.nextToken()

	// Check for comma (multiple return values)
	for p.curTokenIs(lexer.COMMA) {
		p.nextToken() // consume ','
		if p.curTokenIs(lexer.IDENT) {
			results = append(results, p.curToken.Literal)
			p.nextToken()
		}
	}

	// Check for := or =
	if !p.curTokenIs(lexer.DEFINE) && !p.curTokenIs(lexer.ASSIGN) {
		// Not an assignment, backtrack
		return nil
	}
	p.nextToken() // consume ':=' or '='

	// Now parse the hook call on the right side
	hook := p.parseHookCall()
	if hook == nil {
		return nil
	}

	// Set the results
	hook.Results = results

	return hook
}

// parseHookCall parses just the hook call part like gox.UseState[T](initial)
func (p *Parser) parseHookCall() *HookCall {
	hook := &HookCall{
		Position: token.Pos(p.curToken.Line),
	}

	// Handle gox.UseState pattern
	if p.curToken.Literal == "gox" {
		p.nextToken() // consume 'gox'
		if p.curTokenIs(lexer.PERIOD) {
			p.nextToken() // consume '.'
			if p.curTokenIs(lexer.IDENT) && strings.HasPrefix(p.curToken.Literal, "Use") {
				hook.Name = p.curToken.Literal
				p.nextToken()
			} else {
				return nil
			}
		} else {
			return nil
		}
	} else if strings.HasPrefix(p.curToken.Literal, "use") {
		hook.Name = p.curToken.Literal
		p.nextToken()
	} else {
		return nil
	}

	// Parse generic type args [T]
	if p.curTokenIs(lexer.LBRACK) {
		hook.TypeArgs = p.parseTypeArguments()
	}

	// Parse arguments (initial)
	if p.curTokenIs(lexer.LPAREN) {
		hook.Args = p.parseArgumentList()
	}

	return hook
}

// parseRenderBlock parses the render { ... } block
func (p *Parser) parseRenderBlock() *RenderBlock {
	render := &RenderBlock{
		Position: token.Pos(p.curToken.Line),
	}

	if !p.curTokenIs(lexer.LBRACE) {
		// Debug: print current token
		p.addError("expected '{' after 'render'")
		return nil
	}

	p.nextToken() // consume '{'

	// Debug: print current token before parsing JSX

	// Parse JSX
	render.Root = p.parseJSXElement()


	if !p.curTokenIs(lexer.RBRACE) {
		p.addError("expected '}' after render block")
		return nil
	}

	p.nextToken() // consume '}'

	return render
}

// parseJSXElement parses a JSX element
func (p *Parser) parseJSXElement() *JSXElement {
	elem := &JSXElement{
		Position:    token.Pos(p.curToken.Line),
		Attrs:       []*JSXAttribute{},
		Children:    []JSXNode{},
		SelfClosing: false,
	}

	// Debug: print current token

	// Expect <
	if !p.curTokenIs(lexer.JSX_LT) && !p.curTokenIs(lexer.LT) {
		// Try to parse as fragment or other JSX node
		return nil
	}
	p.nextToken() // consume '<'

	// Parse tag name - should be an IDENT token
	if !p.curTokenIs(lexer.IDENT) && !p.curTokenIs(lexer.JSX_TEXT) {
		p.addError("expected element tag name")
		return nil
	}
	elem.Tag = p.curToken.Literal
	p.nextToken()

	// Parse attributes
	for !p.curTokenIs(lexer.JSX_GT) && !p.curTokenIs(lexer.GT) &&
		!p.curTokenIs(lexer.JSX_SLASH) && !p.curTokenIs(lexer.SLASH) &&
		!p.curTokenIs(lexer.EOF) {

		if p.curTokenIs(lexer.IDENT) {
			attr := p.parseJSXAttribute()
			if attr != nil {
				elem.Attrs = append(elem.Attrs, attr)
			}
		} else {
			p.nextToken()
		}
	}

	// Check for self-closing tag
	if p.curTokenIs(lexer.JSX_SLASH) || p.curTokenIs(lexer.SLASH) {
		elem.SelfClosing = true
		p.nextToken() // consume '/'
		if !p.curTokenIs(lexer.JSX_GT) && !p.curTokenIs(lexer.GT) {
			p.addError("expected '>' after '/>'")
			return elem
		}
		p.nextToken() // consume '>'
		return elem
	}

	// Consume >
	if p.curTokenIs(lexer.JSX_GT) || p.curTokenIs(lexer.GT) {
		p.nextToken()
	} else {
		p.addError("expected '>'")
		return elem
	}

	// Parse children
	for !p.isClosingTag(elem.Tag) && !p.curTokenIs(lexer.EOF) {
		child := p.parseJSXChild()
		if child != nil {
			elem.Children = append(elem.Children, child)
		}
	}

	// Parse closing tag
	p.parseClosingTag(elem.Tag)

	return elem
}

// parseJSXAttribute parses a JSX attribute
func (p *Parser) parseJSXAttribute() *JSXAttribute {
	attr := &JSXAttribute{
		Position: token.Pos(p.curToken.Line),
		Name:     p.curToken.Literal,
	}
	p.nextToken()

	if p.curTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume '='

		if p.curTokenIs(lexer.STRING) {
			// String literal value
			attr.Value = JSXText{Value: p.curToken.Literal}
			p.nextToken()
		} else if p.curTokenIs(lexer.JSX_LBRACE) || p.curTokenIs(lexer.LBRACE) {
			// Expression value
			p.nextToken() // consume '{'
			// For now, skip the expression parsing
			// In a real implementation, we'd parse the Go expression
			braceCount := 1
			for braceCount > 0 && !p.curTokenIs(lexer.EOF) {
				if p.curTokenIs(lexer.LBRACE) || p.curTokenIs(lexer.JSX_LBRACE) {
					braceCount++
				} else if p.curTokenIs(lexer.RBRACE) || p.curTokenIs(lexer.JSX_RBRACE) {
					braceCount--
				}
				p.nextToken()
			}
		}
	}

	return attr
}

// parseJSXChild parses a child node in JSX
func (p *Parser) parseJSXChild() JSXNode {
	switch p.curToken.Type {
	case lexer.JSX_LT, lexer.LT:
		// Nested element
		return p.parseJSXElement()
	case lexer.JSX_LBRACE, lexer.LBRACE:
		// Expression
		p.nextToken() // consume '{'

		// Parse the expression content as a simple identifier for now
		// In a full implementation, this would use Go's parser
		expr := &JSXExpression{Position: token.Pos(p.curToken.Line)}

		// For simple cases, capture the identifier
		if p.curTokenIs(lexer.IDENT) {
			expr.Expr = &ast.Ident{
				Name: p.curToken.Literal,
				NamePos: token.Pos(p.curToken.Line),
			}
		}

		// Skip to matching }
		braceCount := 1
		for braceCount > 0 && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.LBRACE) || p.curTokenIs(lexer.JSX_LBRACE) {
				braceCount++
			} else if p.curTokenIs(lexer.RBRACE) || p.curTokenIs(lexer.JSX_RBRACE) {
				braceCount--
			}
			if braceCount > 0 {
				p.nextToken()
			}
		}
		p.nextToken() // consume the final '}'
		return expr
	case lexer.JSX_TEXT, lexer.IDENT, lexer.STRING:
		// Text content
		text := &JSXText{
			Value:    p.curToken.Literal,
			Position: token.Pos(p.curToken.Line),
		}
		p.nextToken()
		return text
	default:
		p.nextToken()
		return nil
	}
}

// parseStyleBlock parses the style { ... } block
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

	if !p.curTokenIs(lexer.LBRACE) {
		p.addError("expected '{' after 'style'")
		return nil
	}

	p.nextToken() // consume '{'

	// Parse CSS rules (simplified)
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		rule := p.parseCSSRule()
		if rule != nil {
			style.Rules = append(style.Rules, rule)
		}
	}

	p.nextToken() // consume '}'

	return style
}

// parseCSSRule parses a CSS rule
func (p *Parser) parseCSSRule() *CSSRule {
	// Simplified CSS parsing
	// In a real implementation, we'd need proper CSS tokenization
	rule := &CSSRule{
		Position:   token.Pos(p.curToken.Line),
		Properties: []*CSSProperty{},
	}

	// Parse selector
	if p.curTokenIs(lexer.CSS_SELECTOR) || p.curTokenIs(lexer.PERIOD) ||
	   p.curTokenIs(lexer.IDENT) {
		rule.Selector = p.curToken.Literal
		p.nextToken()

		// Consume the opening brace
		if p.curTokenIs(lexer.CSS_LBRACE) || p.curTokenIs(lexer.LBRACE) {
			p.nextToken()
		}

		// Parse properties
		for !p.curTokenIs(lexer.CSS_RBRACE) && !p.curTokenIs(lexer.RBRACE) &&
			!p.curTokenIs(lexer.EOF) {

			if p.curTokenIs(lexer.CSS_PROPERTY) || p.curTokenIs(lexer.IDENT) {
				prop := &CSSProperty{
					Position: token.Pos(p.curToken.Line),
					Name:     p.curToken.Literal,
				}
				p.nextToken()

				if p.curTokenIs(lexer.COLON) {
					p.nextToken()
					if p.curTokenIs(lexer.CSS_VALUE) || p.curTokenIs(lexer.IDENT) ||
					   p.curTokenIs(lexer.STRING) || p.curTokenIs(lexer.INT) {
						prop.Value = p.curToken.Literal
						rule.Properties = append(rule.Properties, prop)
					}
					p.nextToken()
				}

				// Skip semicolon if present
				if p.curTokenIs(lexer.SEMICOLON) {
					p.nextToken()
				}
			} else {
				p.nextToken()
			}
		}

		// Consume the closing brace
		if p.curTokenIs(lexer.CSS_RBRACE) || p.curTokenIs(lexer.RBRACE) {
			p.nextToken()
		}
	} else {
		p.nextToken()
	}

	return rule
}

// parseParameterList parses a function parameter list
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

		// Parse parameter type (simplified)
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

// parseTypeArguments parses generic type arguments [T]
func (p *Parser) parseTypeArguments() []ast.Expr {
	var args []ast.Expr

	p.nextToken() // consume '['

	for !p.curTokenIs(lexer.RBRACK) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			args = append(args, &ast.Ident{Name: p.curToken.Literal})
			p.nextToken()
		}

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	if p.curTokenIs(lexer.RBRACK) {
		p.nextToken()
	}

	return args
}

// parseArgumentList parses a function call argument list
func (p *Parser) parseArgumentList() []ast.Expr {
	var args []ast.Expr

	p.nextToken() // consume '('

	for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
		// Simplified expression parsing
		if p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.INT) ||
		   p.curTokenIs(lexer.STRING) || p.curTokenIs(lexer.FLOAT) {
			args = append(args, &ast.BasicLit{Value: p.curToken.Literal})
			p.nextToken()
		} else {
			p.nextToken()
		}

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	if p.curTokenIs(lexer.RPAREN) {
		p.nextToken()
	}

	return args
}

// parseImports parses import declarations
func (p *Parser) parseImports() []*ast.ImportSpec {
	var imports []*ast.ImportSpec

	p.nextToken() // consume 'import'

	if p.curTokenIs(lexer.LPAREN) {
		// Multiple imports
		p.nextToken() // consume '('
		for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.STRING) {
				spec := &ast.ImportSpec{
					Path: &ast.BasicLit{Value: p.curToken.Literal},
				}
				imports = append(imports, spec)
			}
			p.nextToken()
		}
		p.nextToken() // consume ')'
	} else if p.curTokenIs(lexer.STRING) {
		// Single import
		spec := &ast.ImportSpec{
			Path: &ast.BasicLit{Value: p.curToken.Literal},
		}
		imports = append(imports, spec)
		p.nextToken()
	}

	return imports
}

// Helper methods

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) addError(msg string) {
	err := fmt.Errorf("%s:%d:%d: %s",
		p.filename, p.curToken.Line, p.curToken.Column, msg)
	p.errors = append(p.errors, err)
}

func (p *Parser) isClosingTag(tagName string) bool {
	// Check if current token is '</' or JSX_SLASH ('</')
	result := p.curTokenIs(lexer.JSX_SLASH)
	return result
}

func (p *Parser) parseClosingTag(expectedTag string) {
	// Handle </
	if p.curTokenIs(lexer.JSX_SLASH) {
		p.nextToken() // consume '</'
		// Next should be the tag name as IDENT
		if (p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.JSX_TEXT)) && p.curToken.Literal == expectedTag {
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("expected closing tag for '%s'", expectedTag))
		}
		// Finally consume >
		if p.curTokenIs(lexer.JSX_GT) || p.curTokenIs(lexer.GT) {
			p.nextToken()
		}
	}
}

// skipStatement skips over a complete Go statement
func (p *Parser) skipStatement() {
	// Skip over the identifier
	if p.curTokenIs(lexer.IDENT) {
		p.nextToken()
	}

	// Check for assignment operators
	if p.curTokenIs(lexer.DEFINE) || p.curTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume := or =

		// Check if it's a function literal
		if p.curTokenIs(lexer.FUNC) {
			p.nextToken() // consume 'func'

			// Skip parameter list
			if p.curTokenIs(lexer.LPAREN) {
				p.skipParentheses()
			}

			// Skip return type if present
			for !p.curTokenIs(lexer.LBRACE) && !p.curTokenIs(lexer.EOF) &&
				!p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}

			// Skip function body
			if p.curTokenIs(lexer.LBRACE) {
				p.skipBraces()
			}
			return
		}
	}

	// Skip to the end of the statement
	for !p.curTokenIs(lexer.EOF) && !p.curTokenIs(lexer.RBRACE) &&
		!p.curTokenIs(lexer.SEMICOLON) && !p.curTokenIs(lexer.RENDER) &&
		!p.curTokenIs(lexer.STYLE) {
		if p.curTokenIs(lexer.LBRACE) {
			p.skipBraces()
		} else if p.curTokenIs(lexer.LPAREN) {
			p.skipParentheses()
		} else if p.curTokenIs(lexer.LBRACK) {
			p.skipBrackets()
		} else {
			// Check if next token starts a new statement
			p.nextToken()
			if p.curTokenIs(lexer.IDENT) {
				nextTok := p.peekToken
				if nextTok.Type == lexer.DEFINE || nextTok.Type == lexer.ASSIGN {
					// This is the start of a new statement
					break
				}
			}
		}
	}
}

// skipBraces skips over a brace-enclosed block
func (p *Parser) skipBraces() {
	if !p.curTokenIs(lexer.LBRACE) {
		return
	}
	braceCount := 1
	p.nextToken()
	for braceCount > 0 && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LBRACE) {
			braceCount++
		} else if p.curTokenIs(lexer.RBRACE) {
			braceCount--
		}
		if braceCount > 0 {
			p.nextToken()
		}
	}
	if p.curTokenIs(lexer.RBRACE) {
		p.nextToken()
	}
}

// skipParentheses skips over a parentheses-enclosed block
func (p *Parser) skipParentheses() {
	if !p.curTokenIs(lexer.LPAREN) {
		return
	}
	parenCount := 1
	p.nextToken()
	for parenCount > 0 && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LPAREN) {
			parenCount++
		} else if p.curTokenIs(lexer.RPAREN) {
			parenCount--
		}
		if parenCount > 0 {
			p.nextToken()
		}
	}
	if p.curTokenIs(lexer.RPAREN) {
		p.nextToken()
	}
}

// skipBrackets skips over a bracket-enclosed block
func (p *Parser) skipBrackets() {
	if !p.curTokenIs(lexer.LBRACK) {
		return
	}
	bracketCount := 1
	p.nextToken()
	for bracketCount > 0 && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LBRACK) {
			bracketCount++
		} else if p.curTokenIs(lexer.RBRACK) {
			bracketCount--
		}
		if bracketCount > 0 {
			p.nextToken()
		}
	}
	if p.curTokenIs(lexer.RBRACK) {
		p.nextToken()
	}
}

func (p *Parser) skipToNextDeclaration() {
	braceCount := 0
	for !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LBRACE) {
			braceCount++
		} else if p.curTokenIs(lexer.RBRACE) {
			braceCount--
			if braceCount == 0 {
				p.nextToken()
				return
			}
		}
		p.nextToken()
	}
}

// Errors returns all parsing errors
func (p *Parser) Errors() []error {
	return p.errors
}