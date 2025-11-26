package main

import (
	"fmt"
	"github.com/user/gox/pkg/lexer"
	"github.com/user/gox/pkg/parser"
)

func main() {
	source := `component Test() {
	render {
		<div>Hello</div>
	}
}`

	fmt.Println("=== Testing render block parsing ===")

	// Create lexer
	l := lexer.New([]byte(source), "test.gox")

	// Manually step through tokens to debug
	fmt.Println("\nManually stepping through render block tokens:")

	// Skip to "render"
	for i := 0; i < 10; i++ {
		tok := l.NextToken()
		fmt.Printf("Token %d: Type=%v, Literal=%q\n", i, tok.Type, tok.Literal)
		if tok.Literal == "render" {
			fmt.Printf("Found 'render' with token type: %v\n", tok.Type)
			// Next should be LBRACE
			tok = l.NextToken()
			fmt.Printf("After render: %v = %q\n", tok.Type, tok.Literal)
			// Next should switch to JSX mode
			tok = l.NextToken()
			fmt.Printf("After {: %v = %q (should be JSX_LT)\n", tok.Type, tok.Literal)
			tok = l.NextToken()
			fmt.Printf("Next: %v = %q (should be tag name)\n", tok.Type, tok.Literal)
			break
		}
	}

	// Now test actual parsing
	fmt.Println("\n=== Full parse test ===")
	l2 := lexer.New([]byte(source), "test.gox")
	p2 := parser.New(l2, "test.gox")
	file, err := p2.ParseFile()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	if len(file.Components) > 0 {
		comp := file.Components[0]
		fmt.Printf("Component: %s\n", comp.Name.Name)
		fmt.Printf("Has render: %v\n", comp.Body != nil && comp.Body.Render != nil)
		if comp.Body != nil && comp.Body.Render != nil {
			fmt.Printf("Has root: %v\n", comp.Body.Render.Root != nil)
			if comp.Body.Render.Root != nil {
				fmt.Printf("Root tag: %s\n", comp.Body.Render.Root.Tag)
			}
		}
	}
}