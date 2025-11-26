package main

import (
	"fmt"
	"github.com/user/gox/pkg/lexer"
	"github.com/user/gox/pkg/parser"
)

func main() {
	source := `component Test() {
	doubled := gox.UseMemo(func() int { return count * 2 }, []interface{}{count})

	render {
		<div>Test</div>
	}
}`

	l := lexer.New([]byte(source), "test.gox")
	p := parser.New(l, "test.gox")

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

	if len(file.Components) > 0 {
		comp := file.Components[0]
		fmt.Printf("Component: %s\n", comp.Name.Name)
		fmt.Printf("Hooks: %d\n", len(comp.Body.Hooks))
		fmt.Printf("Has render: %v\n", comp.Body.Render != nil)
		if comp.Body.Render != nil && comp.Body.Render.Root != nil {
			fmt.Printf("Render root tag: %s\n", comp.Body.Render.Root.Tag)
		}
	}
}