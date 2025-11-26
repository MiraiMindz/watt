// Package parser provides parsing for GoX source files
package parser

import (
	"go/ast"
	"go/token"
)

// File represents a complete GoX source file
type File struct {
	Package    string
	Imports    []*ast.ImportSpec
	Components []*ComponentDecl
	Functions  []*ast.FuncDecl
	Types      []*ast.TypeSpec
	Position   token.Pos
}

// ComponentDecl represents a GoX component declaration
type ComponentDecl struct {
	Name     *ast.Ident
	Doc      *ast.CommentGroup
	Params   *ast.FieldList // function parameters (props)
	Body     *ComponentBody
	Position token.Pos
}

// ComponentBody contains the body of a component
type ComponentBody struct {
	Stmts  []ast.Stmt    // Regular Go statements
	Hooks  []*HookCall   // useState, useEffect, etc.
	Render *RenderBlock  // JSX return
	Style  *StyleBlock   // CSS styles
}

// HookCall represents a React-style hook call
type HookCall struct {
	Name     string      // "useState", "useEffect", etc.
	TypeArgs []ast.Expr  // Generic type arguments
	Args     []ast.Expr  // Arguments
	Results  []string    // Return value names (e.g., [value, setValue])
	Position token.Pos
}

// RenderBlock contains the JSX render content
type RenderBlock struct {
	Root     *JSXElement
	Position token.Pos
}

// JSXNode is the interface for all JSX nodes
type JSXNode interface {
	jsxNode()
	Pos() token.Pos
}

// JSXElement represents a JSX element
type JSXElement struct {
	Tag         string
	Attrs       []*JSXAttribute
	Children    []JSXNode
	Position    token.Pos
	SelfClosing bool
}

func (JSXElement) jsxNode() {}
func (e JSXElement) Pos() token.Pos { return e.Position }

// JSXAttribute represents an attribute on a JSX element
type JSXAttribute struct {
	Name     string
	Value    JSXAttrValue
	Position token.Pos
}

// JSXAttrValue represents the value of a JSX attribute
type JSXAttrValue interface {
	jsxAttrValue()
}

// JSXText represents text content in JSX
type JSXText struct {
	Value    string
	Position token.Pos
}

func (JSXText) jsxNode() {}
func (JSXText) jsxAttrValue() {}
func (t JSXText) Pos() token.Pos { return t.Position }

// JSXExpression represents an embedded expression in JSX
type JSXExpression struct {
	Expr     ast.Expr
	Position token.Pos
}

func (JSXExpression) jsxNode() {}
func (JSXExpression) jsxAttrValue() {}
func (e JSXExpression) Pos() token.Pos { return e.Position }

// JSXFragment represents a React fragment (<>...</>)
type JSXFragment struct {
	Children []JSXNode
	Position token.Pos
}

func (JSXFragment) jsxNode() {}
func (f JSXFragment) Pos() token.Pos { return f.Position }

// JSXConditional represents conditional rendering in JSX
type JSXConditional struct {
	Condition ast.Expr
	Then      JSXNode
	Else      JSXNode // can be nil
	Position  token.Pos
}

func (JSXConditional) jsxNode() {}
func (c JSXConditional) Pos() token.Pos { return c.Position }

// JSXList represents list rendering (items.map(...))
type JSXList struct {
	Items    ast.Expr
	Iterator string    // variable name for each item
	Index    string    // variable name for index (optional)
	Body     JSXNode   // what to render for each item
	Position token.Pos
}

func (JSXList) jsxNode() {}
func (l JSXList) Pos() token.Pos { return l.Position }

// StyleBlock contains CSS styles
type StyleBlock struct {
	Rules    []*CSSRule
	Global   bool // whether styles are global or scoped
	Position token.Pos
}

// CSSRule represents a CSS rule
type CSSRule struct {
	Selector   string
	Properties []*CSSProperty
	Position   token.Pos
}

// CSSProperty represents a CSS property
type CSSProperty struct {
	Name     string
	Value    string
	Position token.Pos
}

// Visitor interface for walking the AST
type Visitor interface {
	Visit(node interface{}) (w Visitor)
}

// Walk traverses an AST in depth-first order
func Walk(v Visitor, node interface{}) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *File:
		for _, comp := range n.Components {
			Walk(v, comp)
		}
		for _, fn := range n.Functions {
			Walk(v, fn)
		}
		for _, typ := range n.Types {
			Walk(v, typ)
		}

	case *ComponentDecl:
		Walk(v, n.Body)

	case *ComponentBody:
		for _, hook := range n.Hooks {
			Walk(v, hook)
		}
		if n.Render != nil {
			Walk(v, n.Render)
		}
		if n.Style != nil {
			Walk(v, n.Style)
		}

	case *RenderBlock:
		if n.Root != nil {
			Walk(v, n.Root)
		}

	case *JSXElement:
		for _, attr := range n.Attrs {
			Walk(v, attr)
		}
		for _, child := range n.Children {
			Walk(v, child)
		}

	case *JSXFragment:
		for _, child := range n.Children {
			Walk(v, child)
		}

	case *JSXConditional:
		if n.Then != nil {
			Walk(v, n.Then)
		}
		if n.Else != nil {
			Walk(v, n.Else)
		}

	case *JSXList:
		if n.Body != nil {
			Walk(v, n.Body)
		}

	case *StyleBlock:
		for _, rule := range n.Rules {
			Walk(v, rule)
		}
	}

	v.Visit(nil)
}