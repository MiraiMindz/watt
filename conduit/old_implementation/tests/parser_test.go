package tests

import (
	"testing"
	"github.com/user/gox/pkg/lexer"
	"github.com/user/gox/pkg/parser"
)

func TestParseSimpleComponent(t *testing.T) {
	input := `
package main

import "gox"

component Button() {
	render {
		<div>Click me</div>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if file.Package != "main" {
		t.Errorf("wrong package name. got=%q, want=%q", file.Package, "main")
	}

	if len(file.Components) != 1 {
		t.Fatalf("wrong number of components. got=%d, want=1", len(file.Components))
	}

	comp := file.Components[0]
	if comp.Name.Name != "Button" {
		t.Errorf("wrong component name. got=%q, want=%q", comp.Name.Name, "Button")
	}

	if comp.Body.Render == nil {
		t.Fatal("component should have render block")
	}

	if comp.Body.Render.Root == nil {
		t.Fatal("render block should have root element")
	}

	if comp.Body.Render.Root.Tag != "div" {
		t.Errorf("wrong root tag. got=%q, want=%q", comp.Body.Render.Root.Tag, "div")
	}
}

func TestParseComponentWithState(t *testing.T) {
	input := `
component Counter() {
	count, setCount := gox.UseState[int](0)

	render {
		<div>{count}</div>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(file.Components) != 1 {
		t.Fatalf("wrong number of components. got=%d, want=1", len(file.Components))
	}

	comp := file.Components[0]
	if len(comp.Body.Hooks) != 1 {
		t.Fatalf("wrong number of hooks. got=%d, want=1", len(comp.Body.Hooks))
	}

	hook := comp.Body.Hooks[0]
	if hook.Name != "UseState" {
		t.Errorf("wrong hook name. got=%q, want=%q", hook.Name, "UseState")
	}

	if len(hook.Results) != 2 {
		t.Fatalf("wrong number of hook results. got=%d, want=2", len(hook.Results))
	}

	if hook.Results[0] != "count" {
		t.Errorf("wrong first result. got=%q, want=%q", hook.Results[0], "count")
	}

	if hook.Results[1] != "setCount" {
		t.Errorf("wrong second result. got=%q, want=%q", hook.Results[1], "setCount")
	}
}

func TestParseComponentWithProps(t *testing.T) {
	input := `
component Greeting(name string, age int) {
	render {
		<div>Hello {name}, you are {age} years old</div>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	comp := file.Components[0]
	if comp.Params == nil {
		t.Fatal("component should have parameters")
	}

	if len(comp.Params.List) != 2 {
		t.Fatalf("wrong number of parameters. got=%d, want=2", len(comp.Params.List))
	}
}

func TestParseJSXAttributes(t *testing.T) {
	input := `
component Button() {
	render {
		<button className="btn" onClick={handleClick}>Click</button>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	comp := file.Components[0]
	root := comp.Body.Render.Root

	if root.Tag != "button" {
		t.Errorf("wrong tag. got=%q, want=%q", root.Tag, "button")
	}

	if len(root.Attrs) != 2 {
		t.Fatalf("wrong number of attributes. got=%d, want=2", len(root.Attrs))
	}

	// Check className attribute
	if root.Attrs[0].Name != "className" {
		t.Errorf("wrong attribute name. got=%q, want=%q", root.Attrs[0].Name, "className")
	}

	// Check onClick attribute
	if root.Attrs[1].Name != "onClick" {
		t.Errorf("wrong attribute name. got=%q, want=%q", root.Attrs[1].Name, "onClick")
	}
}

func TestParseNestedJSX(t *testing.T) {
	input := `
component Card() {
	render {
		<div>
			<h1>Title</h1>
			<p>Content</p>
		</div>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	comp := file.Components[0]
	root := comp.Body.Render.Root

	if len(root.Children) != 2 {
		t.Fatalf("wrong number of children. got=%d, want=2", len(root.Children))
	}

	// Check first child (h1)
	firstChild, ok := root.Children[0].(*parser.JSXElement)
	if !ok {
		t.Fatal("first child should be JSXElement")
	}
	if firstChild.Tag != "h1" {
		t.Errorf("wrong first child tag. got=%q, want=%q", firstChild.Tag, "h1")
	}

	// Check second child (p)
	secondChild, ok := root.Children[1].(*parser.JSXElement)
	if !ok {
		t.Fatal("second child should be JSXElement")
	}
	if secondChild.Tag != "p" {
		t.Errorf("wrong second child tag. got=%q, want=%q", secondChild.Tag, "p")
	}
}

func TestParseFunctionVariables(t *testing.T) {
	input := `
component Button() {
	handleClick := func() {
		console.log("clicked")
	}

	render {
		<button onClick={handleClick}>Click</button>
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Should parse without errors even with function variables
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}

	comp := file.Components[0]
	if comp.Body.Render == nil {
		t.Fatal("component should have render block despite function variables")
	}
}

func TestParseStyleBlock(t *testing.T) {
	input := `
component Card() {
	render {
		<div className="card">Content</div>
	}

	style {
		.card {
			padding: 20px;
			background: white;
		}
	}
}
`

	l := lexer.New([]byte(input), "test.gox")
	p := parser.New(l, "test.gox")

	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	comp := file.Components[0]
	if comp.Body.Style == nil {
		t.Fatal("component should have style block")
	}

	if len(comp.Body.Style.Rules) == 0 {
		t.Error("style block should have CSS rules")
	}
}