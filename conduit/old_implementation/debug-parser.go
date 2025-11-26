package main

import (
	"fmt"
	"github.com/user/gox/pkg/lexer"
	"github.com/user/gox/pkg/parser"
)

func main() {
	source := `package main

import "gox"

component SimpleTest() {
	count, setCount := gox.UseState[int](0)

	render {
		<div>Hello World</div>
	}
}`

	// Test lexer
	fmt.Println("=== LEXER OUTPUT ===")
	l := lexer.New([]byte(source), "test.gox")

	for i := 0; i < 50; i++ {
		tok := l.NextToken()
		fmt.Printf("%d: %v = %q\n", i, tok.Type, tok.Literal)
		if tok.Type == lexer.EOF {
			break
		}
	}

	// Test parser
	fmt.Println("\n=== PARSER OUTPUT ===")
	l2 := lexer.New([]byte(source), "test.gox")
	p := parser.New(l2, "test.gox")

	// Debug: Show token progression
	fmt.Println("Token progression during parsing:")

	file, err := p.ParseFile()
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	}

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, e := range p.Errors() {
			fmt.Printf("  %v\n", e)
		}
	}

	fmt.Printf("Package: %s\n", file.Package)
	fmt.Printf("Components: %d\n", len(file.Components))

	if len(file.Components) > 0 {
		comp := file.Components[0]
		fmt.Printf("Component name: %s\n", comp.Name.Name)
		fmt.Printf("Has render block: %v\n", comp.Body.Render != nil)
		if comp.Body.Render != nil && comp.Body.Render.Root != nil {
			fmt.Printf("Render root tag: %s\n", comp.Body.Render.Root.Tag)
		}
		fmt.Printf("Hooks: %d\n", len(comp.Body.Hooks))
		for i, hook := range comp.Body.Hooks {
			fmt.Printf("  Hook %d: %s with %d results\n", i, hook.Name, len(hook.Results))
		}
	}
}