package main

import (
	"fmt"
	"github.com/user/gox/pkg/lexer"
)

func main() {
	source := `<div className="counter">Hello</div>`

	fmt.Println("=== Testing attribute lexing ===")

	l := lexer.New([]byte(source), "test.gox")
	l.PushMode(lexer.ModeJSX)

	for i := 0; i < 20; i++ {
		tok := l.NextToken()
		fmt.Printf("Token %d: Type=%v, Literal=%q\n", i, tok.Type, tok.Literal)
		if tok.Type == lexer.EOF {
			break
		}
	}
}