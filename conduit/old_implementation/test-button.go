package main

import (
	"fmt"
	"github.com/user/gox/pkg/lexer"
)

func main() {
	source := `<button onClick={() => setCount(count + 1)}>Increment</button>`

	fmt.Println("=== Testing button with onClick ===")

	l := lexer.New([]byte(source), "test.gox")
	// Switch to JSX mode
	l.PushMode(lexer.ModeJSX)

	for i := 0; i < 30; i++ {
		tok := l.NextToken()
		fmt.Printf("Token %d: Type=%v, Literal=%q\n", i, tok.Type, tok.Literal)
		if tok.Type == lexer.EOF {
			break
		}
	}
}