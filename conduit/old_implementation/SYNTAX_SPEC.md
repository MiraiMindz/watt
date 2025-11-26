# GoX Syntax Specification

## Overview

GoX is a superset of Go that adds React-like component syntax for building web UIs. This document defines the syntax extensions.

---

## File Extension

`.gox` - GoX source files

---

## Component Declaration

### Basic Syntax

```gox
component ComponentName(prop1 Type1, prop2 Type2) {
    // component body
}
```

### With Props

```gox
component Button(text string, onClick func()) {
    render {
        <button onclick={onClick}>{text}</button>
    }
}
```

### Function Component Pattern

GoX follows React's function component pattern. Components are essentially functions that:
1. Accept props as parameters
2. Use hooks for state and effects
3. Return JSX from a render block

---

## Hooks

### useState[T]

```gox
value, setValue := gox.UseState[Type](initialValue)
```

**Examples:**

```gox
component Counter(initial int) {
    count, setCount := gox.UseState[int](initial)
    name, setName := gox.UseState[string]("John")
    items, setItems := gox.UseState[[]string]([]string{})
}
```

### useEffect

```gox
gox.UseEffect(func() func() {
    // setup code
    return func() {
        // cleanup code
    }
}, []interface{}{dep1, dep2})
```

**Examples:**

```gox
// Run once on mount
gox.UseEffect(func() func() {
    fmt.Println("Component mounted")
    return func() {
        fmt.Println("Component unmounted")
    }
}, []interface{}{})

// Run when count changes
gox.UseEffect(func() func() {
    fmt.Printf("Count is now: %d\n", count)
    return nil
}, []interface{}{count})
```

### useMemo[T]

```gox
memoized := gox.UseMemo[Type](func() Type {
    // expensive computation
    return result
}, []interface{}{dep1, dep2})
```

**Example:**

```gox
expensiveValue := gox.UseMemo[int](func() int {
    return computeExpensiveValue(a, b)
}, []interface{}{a, b})
```

### useCallback[T]

```gox
callback := gox.UseCallback[func()](func() {
    // callback function
}, []interface{}{dep1, dep2})
```

**Example:**

```gox
handleClick := gox.UseCallback[func()](func() {
    setCount(count + 1)
}, []interface{}{count})
```

### useRef[T]

```gox
ref := gox.UseRef[Type](initialValue)
// Access with ref.Current
```

**Example:**

```gox
inputRef := gox.UseRef[js.Value](js.Null())

render {
    <input ref={inputRef} />
}
```

### useContext[T]

```gox
value := gox.UseContext[Type](contextObject)
```

**Example:**

```gox
theme := gox.UseContext[string](ThemeContext)
```

---

## JSX Syntax

### Elements

```gox
<tagname attribute="value" attribute2={expression}>
    children
</tagname>
```

### Self-Closing Tags

```gox
<img src="image.jpg" alt="description" />
<input type="text" value={text} />
```

### Attributes

**Static values:**
```gox
<div class="container" id="main">
```

**Dynamic expressions:**
```gox
<div class={className} style={styleObj}>
```

**Boolean attributes:**
```gox
<input disabled={isDisabled} required />
```

### Event Handlers

```gox
<button onClick={handleClick}>Click</button>
<input onChange={handleChange} onInput={handleInput} />
<form onSubmit={handleSubmit}>
```

**Available events:**
- onClick, onDoubleClick
- onChange, onInput
- onSubmit
- onFocus, onBlur
- onKeyDown, onKeyUp, onKeyPress
- onMouseDown, onMouseUp, onMouseMove, onMouseEnter, onMouseLeave

### Children

**Text:**
```gox
<div>Hello World</div>
```

**Expressions:**
```gox
<div>{count}</div>
<div>{user.name}</div>
```

**Nested elements:**
```gox
<div>
    <h1>Title</h1>
    <p>Paragraph</p>
</div>
```

**Conditional rendering:**
```gox
{condition && <div>Shown when true</div>}
{condition ? <div>True</div> : <div>False</div>}
```

**List rendering:**
```gox
{items.map(func(item string, i int) *gox.VNode {
    return <li key={i}>{item}</li>
})}
```

### Fragments

```gox
<>
    <div>First</div>
    <div>Second</div>
</>
```

### Components

```gox
<Button text="Click me" onClick={handleClick} />
<UserProfile user={currentUser} />
```

---

## Style Block

### Basic Syntax

```gox
component MyComponent() {
    style {
        .class-name {
            property: value;
        }

        #id {
            property: value;
        }
    }
}
```

### Scoped Styles (Default)

Styles are scoped to the component by default:

```gox
component Card() {
    style {
        .card {
            border: 1px solid #ccc;
            padding: 20px;
        }
    }

    render {
        <div className="card">Content</div>
    }
}
```

Generated CSS:
```css
.card[data-gox-abc123] {
    border: 1px solid #ccc;
    padding: 20px;
}
```

### Global Styles

```gox
component App() {
    style global {
        body {
            margin: 0;
            font-family: sans-serif;
        }
    }
}
```

### Pseudo-classes

```gox
style {
    .button {
        background: blue;
    }

    .button:hover {
        background: darkblue;
    }

    .button:active {
        background: navy;
    }
}
```

### Media Queries

```gox
style {
    .container {
        width: 100%;
    }

    @media (min-width: 768px) {
        .container {
            width: 750px;
        }
    }
}
```

---

## Render Block

### Syntax

```gox
render {
    <jsx>...</jsx>
}
```

### Single Root Element

```gox
render {
    <div>
        <h1>Title</h1>
        <p>Content</p>
    </div>
}
```

### Fragment Root

```gox
render {
    <>
        <Header />
        <Main />
        <Footer />
    </>
}
```

### Conditional Rendering

```gox
render {
    <div>
        {isLoggedIn ? (
            <UserDashboard />
        ) : (
            <LoginForm />
        )}
    </div>
}
```

### List Rendering

```gox
render {
    <ul>
        {items.map(func(item Item, index int) *gox.VNode {
            return <li key={index}>{item.name}</li>
        })}
    </ul>
}
```

---

## Complete Component Example

```gox
package components

import (
    "fmt"
    "gox"
)

// TodoItem component
component TodoItem(todo Todo, onToggle func(), onDelete func()) {
    render {
        <div className="todo-item">
            <input
                type="checkbox"
                checked={todo.Done}
                onChange={onToggle}
            />
            <span className={todo.Done ? "done" : ""}>{todo.Text}</span>
            <button onClick={onDelete}>Delete</button>
        </div>
    }

    style {
        .todo-item {
            display: flex;
            align-items: center;
            padding: 10px;
            border-bottom: 1px solid #eee;
        }

        .todo-item span {
            flex: 1;
            margin: 0 10px;
        }

        .todo-item .done {
            text-decoration: line-through;
            color: #999;
        }
    }
}

// TodoList component
component TodoList() {
    todos, setTodos := gox.UseState[[]Todo]([]Todo{})
    input, setInput := gox.UseState[string]("")

    gox.UseEffect(func() func() {
        // Load todos from localStorage
        loadTodos()
        return nil
    }, []interface{}{})

    addTodo := func() {
        if input == "" {
            return
        }

        newTodo := Todo{
            ID:   generateID(),
            Text: input,
            Done: false,
        }

        setTodos(append(todos, newTodo))
        setInput("")
    }

    toggleTodo := func(id string) func() {
        return func() {
            newTodos := make([]Todo, len(todos))
            copy(newTodos, todos)

            for i := range newTodos {
                if newTodos[i].ID == id {
                    newTodos[i].Done = !newTodos[i].Done
                }
            }

            setTodos(newTodos)
        }
    }

    deleteTodo := func(id string) func() {
        return func() {
            newTodos := []Todo{}
            for _, todo := range todos {
                if todo.ID != id {
                    newTodos = append(newTodos, todo)
                }
            }
            setTodos(newTodos)
        }
    }

    render {
        <div className="todo-list">
            <h1>Todo List</h1>

            <div className="add-todo">
                <input
                    type="text"
                    value={input}
                    onInput={func(e js.Value) {
                        setInput(e.Get("target").Get("value").String())
                    }}
                    placeholder="Add a todo..."
                />
                <button onClick={addTodo}>Add</button>
            </div>

            <div className="todos">
                {todos.map(func(todo Todo, i int) *gox.VNode {
                    return <TodoItem
                        key={todo.ID}
                        todo={todo}
                        onToggle={toggleTodo(todo.ID)}
                        onDelete={deleteTodo(todo.ID)}
                    />
                })}
            </div>

            <div className="stats">
                <span>{countActive(todos)} remaining</span>
            </div>
        </div>
    }

    style {
        .todo-list {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }

        .add-todo {
            display: flex;
            margin-bottom: 20px;
        }

        .add-todo input {
            flex: 1;
            padding: 10px;
            border: 1px solid #ccc;
            border-radius: 4px 0 0 4px;
        }

        .add-todo button {
            padding: 10px 20px;
            background: #007bff;
            color: white;
            border: none;
            border-radius: 0 4px 4px 0;
            cursor: pointer;
        }

        .add-todo button:hover {
            background: #0056b3;
        }

        .stats {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 2px solid #eee;
            color: #666;
        }
    }
}

type Todo struct {
    ID   string
    Text string
    Done bool
}

func generateID() string {
    // Implementation
}

func countActive(todos []Todo) int {
    count := 0
    for _, todo := range todos {
        if !todo.Done {
            count++
        }
    }
    return count
}

func loadTodos() {
    // Implementation
}
```

---

## Type System

### Props Types

Props are regular Go function parameters:

```gox
component MyComponent(
    name string,
    age int,
    items []string,
    callback func(string),
    optional *int,
) {
    // ...
}
```

### Generic Components

Using Go's generics:

```gox
component List[T any](items []T, renderItem func(T) *gox.VNode) {
    render {
        <ul>
            {items.map(func(item T, i int) *gox.VNode {
                return <li key={i}>{renderItem(item)}</li>
            })}
        </ul>
    }
}

// Usage
<List[string]
    items={names}
    renderItem={func(name string) *gox.VNode {
        return <span>{name}</span>
    }}
/>
```

---

## Context API

### Creating Context

```gox
package contexts

import "gox"

var ThemeContext = gox.CreateContext[string]("light")
var UserContext = gox.CreateContext[*User](nil)
```

### Providing Context

```gox
component App() {
    theme, setTheme := gox.UseState[string]("dark")

    render {
        <ThemeContext.Provider value={theme}>
            <MainContent />
        </ThemeContext.Provider>
    }
}
```

### Consuming Context

```gox
component ThemedButton() {
    theme := gox.UseContext[string](ThemeContext)

    render {
        <button className={theme}>Click me</button>
    }
}
```

---

## Refs

### DOM Refs

```gox
component FocusInput() {
    inputRef := gox.UseRef[js.Value](js.Null())

    focusInput := func() {
        if !inputRef.Current.IsNull() {
            inputRef.Current.Call("focus")
        }
    }

    render {
        <div>
            <input ref={inputRef} />
            <button onClick={focusInput}>Focus Input</button>
        </div>
    }
}
```

### Value Refs

```gox
component Timer() {
    count := gox.UseRef[int](0)

    gox.UseEffect(func() func() {
        ticker := time.NewTicker(1 * time.Second)
        go func() {
            for range ticker.C {
                count.Current++
                fmt.Println(count.Current)
            }
        }()

        return func() {
            ticker.Stop()
        }
    }, []interface{}{})
}
```

---

## Compilation Modes

### SSR (Server-Side Rendering)

```bash
goxc build --mode=ssr -o dist/ src/*.gox
```

Generates:
- `.go` files with Render() string methods
- Optimized for server-side HTML generation
- No WASM overhead

### CSR (Client-Side Rendering / WASM)

```bash
goxc build --mode=csr -o dist/ src/*.gox
GOOS=js GOARCH=wasm go build -o dist/app.wasm dist/*.go
```

Generates:
- `.go` files with Render() *VNode methods
- Virtual DOM implementation
- Event handlers
- Compiles to WebAssembly

---

## Best Practices

### Component Naming

- Use PascalCase: `MyComponent`, `UserProfile`
- One component per file
- File name matches component name: `UserProfile.gox`

### State Management

- Keep state minimal
- Lift state up when needed
- Use context for global state

### Performance

- Use `useMemo` for expensive computations
- Use `useCallback` for function props
- Add keys to list items

### Styling

- Use scoped styles by default
- Keep styles close to components
- Use CSS variables for theming

---

## Interop with Regular Go

GoX files can import and use regular Go packages:

```gox
package main

import (
    "fmt"
    "time"
    "gox"
    "myapp/models"
    "myapp/services"
)

component UserDashboard(userID string) {
    user, setUser := gox.UseState[*models.User](nil)

    gox.UseEffect(func() func() {
        fetchedUser, err := services.GetUser(userID)
        if err == nil {
            setUser(fetchedUser)
        }
        return nil
    }, []interface{}{userID})

    render {
        <div>
            {user != nil ? (
                <div>
                    <h1>{user.Name}</h1>
                    <p>{user.Email}</p>
                </div>
            ) : (
                <div>Loading...</div>
            )}
        </div>
    }
}
```

Regular Go files can use generated components:

```go
package main

import "myapp/components"

func main() {
    // SSR
    counter := components.NewCounter(0)
    html := counter.Render()
    fmt.Println(html)
}
```
