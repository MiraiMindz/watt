# Conduit Implementation Plan (React-like .gox Superset)

## Overview
Conduit is Watt's high-performance, React-like templating engine that extends Go with component-based UI development. Using .gox files (a superset of Go), it provides a familiar React-like syntax with compile-time type safety, leveraging Electron's high-performance internals for maximum efficiency.

## Core Features
- **`.gox` Files**: Superset of Go with component syntax
- **Component Keyword**: Define UI components with JSX-like markup
- **State Management**: Dual API - keyword-based and React hooks
- **Style Blocks**: Scoped CSS within components
- **Expression Interpolation**: `${}` syntax for dynamic values
- **Multi-Target Compilation**:
  - WASM/JS for Client-Side Rendering (CSR)
  - Go files for Server-Side Rendering (SSR) via Bolt/Shockwave
- **Markdown Support**: Native markdown rendering in components

## Phase 1: Language Design & Parser (Week 1-2)

### 1.1 .gox Syntax Specification
```gox
package components

import (
    "fmt"
    "watt/electron/arena"
    "watt/conduit"
)

// Component declaration with 'component' keyword
component UserCard(user User, isAdmin bool) {
    // State declaration (keyword syntax)
    state {
        expanded: bool = false
        clickCount: int = 0
        hoverState: string = "none"
    }

    // React-like hooks API (alternative/complementary)
    liked, setLiked := conduit.UseState[boolean](false)
    comment, setComment := conduit.UseState[string]("initialValue")

    // Event handlers
    func handleClick() {
        state.clickCount++
        state.expanded = !state.expanded
        setLiked(!liked)
    }

    // Style block with scoped CSS
    style {
        .user-card {
            padding: 1rem;
            border: 1px solid #ddd;
            border-radius: 8px;
            transition: all 0.3s ease;
        }

        .user-card:hover {
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }

        .admin-badge {
            background: gold;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
        }

        .expanded {
            max-height: 500px;
        }

        @media (max-width: 768px) {
            .user-card {
                padding: 0.5rem;
            }
        }
    }

    // Render with JSX-like syntax and ${} expressions
    render {
        <div className="user-card ${state.expanded ? 'expanded' : ''}"
             onClick={handleClick}
             data-clicks="${state.clickCount}">

            <div className="header">
                <img src="${user.avatar}" alt="${user.name}" />
                <h2>${displayName}</h2>
                ${isAdmin && <span className="admin-badge">Admin</span>}
            </div>

            <div className="content">
                <!-- Markdown support -->
                <markdown>
                    ## About ${user.name}
                    ${user.bio}

                    **Joined**: ${user.joinDate}
                </markdown>

                ${state.expanded &&
                    <div className="details">
                        <p>Email: ${user.email}</p>
                        <p>Clicks: ${state.clickCount}</p>
                        <button onClick={() => setLiked(!liked)}>
                            ${liked ? '❤️' : '🤍'} Like
                        </button>
                    </div>
                }
            </div>

            <!-- Conditional rendering -->
            ${state.clickCount > 5 ?
                <Alert message="You really like clicking!" /> :
                null
            }

            <!-- List rendering -->
            <ul className="tags">
                ${for _, v in user.tags {
                    return <li key={v.id}>${v.name}</li>
                }}
            </ul>
        </div>
    }
}
```

### 1.2 Parser Implementation Using Electron
```go
package parser

import (
    "watt/electron/simd"
    "watt/electron/arena"
    "watt/electron/strings"
)

type GoxParser struct {
    arena    *arena.Arena
    interner *strings.StringInterner
    simd     bool
}

// Parse phases
func (p *GoxParser) Parse(source string) (*AST, error) {
    // Phase 1: Tokenize using Electron's SIMD string operations
    tokens := p.tokenize(source)

    // Phase 2: Build AST with arena allocation
    ast := p.buildAST(tokens)

    // Phase 3: Type checking and validation
    p.validate(ast)

    return ast, nil
}

// Fast tokenization with SIMD
func (p *GoxParser) tokenize(source string) []Token {
    if p.simd && simd.HasSIMD() {
        return p.tokenizeSIMD(source)
    }
    return p.tokenizeScalar(source)
}
```

## Phase 2: State Management System (Week 3-4)

### 2.1 Dual State API Design
```go
// Keyword-based state (compiled to efficient getters/setters)
type KeywordState struct {
    fields map[string]StateField
    dirty  uint64  // Bit flags for changed fields
}

type StateField struct {
    name     string
    typ      reflect.Type
    value    interface{}
    watchers []func(old, new interface{})
}

// React-like hooks state
type HooksState struct {
    states    []interface{}
    setters   []func(interface{})
    effects   []Effect
    memos     []Memo
    refs      []Ref
    contexts  []Context
}

// Unified state manager using Electron's lock-free structures
type StateManager struct {
    keyword  *KeywordState
    hooks    *HooksState
    updates  *electron.Queue[StateUpdate]  // Lock-free update queue
    renderer func()
}
```

### 2.2 State Synchronization
```go
// Automatic state synchronization between keyword and hooks
func (sm *StateManager) Sync() {
    // Batch updates using Electron's arena for efficiency
    arena := electron.NewArena(1024)
    defer arena.Free()

    updates := sm.collectUpdates(arena)
    sm.applyBatch(updates)

    if sm.hasChanges() {
        sm.renderer()
    }
}

// Reactive state with automatic tracking
func (s *KeywordState) Get(field string) interface{} {
    // Track dependency for automatic re-render
    s.trackDependency(field)
    return s.fields[field].value
}

func (s *KeywordState) Set(field string, value interface{}) {
    old := s.fields[field].value
    s.fields[field].value = value
    s.markDirty(field)
    s.notifyWatchers(field, old, value)
}
```

## Phase 3: Component System (Week 5-6)

### 3.1 Component Compiler
```go
type ComponentCompiler struct {
    parser   *GoxParser
    analyzer *TypeAnalyzer
    codegen  *CodeGenerator
}

// Compile modes
type CompileMode int
const (
    SSR CompileMode = iota  // Server-Side Rendering (Go output)
    CSR                      // Client-Side Rendering (WASM)
    Hybrid                   // Both SSR and CSR
)

func (cc *ComponentCompiler) Compile(source string, mode CompileMode) (*CompiledComponent, error) {
    // Parse .gox file
    ast := cc.parser.Parse(source)

    // Type analysis and checking
    cc.analyzer.Analyze(ast)

    // Generate code based on mode
    switch mode {
    case SSR:
        return cc.generateSSR(ast)
    case CSR:
        return cc.generateCSR(ast)
    case Hybrid:
        return cc.generateHybrid(ast)
    }
}
```

### 3.2 SSR Code Generation (Go Output)
```go
// Generated Go code from .gox component
package components

import (
    "watt/electron/arena"
    "watt/electron/strings"
)

type UserCardSSR struct {
    arena    *arena.Arena
    user     User
    isAdmin  bool
    state    *UserCardState
}

func (c *UserCardSSR) Render() string {
    // Use Electron's zero-allocation string building
    builder := strings.NewBuilder(c.arena)

    // Inline CSS (generated from style block)
    builder.WriteString(`<style>.user-card[data-gox-123]{...}</style>`)

    // Component HTML with pre-computed values
    builder.WriteString(`<div class="user-card" data-gox-123>`)
    // ... rest of rendered HTML
    builder.WriteString(`</div>`)

    return builder.String()
}

// Hydration data for CSR takeover
func (c *UserCardSSR) HydrationData() []byte {
    // Serialize initial state for client-side hydration
    return electron.encoding.MarshalBinary(c.state)
}
```

### 3.3 CSR Code Generation (WASM/JS)
```go
// Generated code for WASM compilation
package components

import (
    "syscall/js"
    "watt/electron/wasm"
)

type UserCardCSR struct {
    vdom     *VirtualDOM
    state    *ReactiveState
    element  js.Value
}

func (c *UserCardCSR) Mount(element js.Value) {
    c.element = element

    // Initial render
    vnode := c.render()
    c.vdom.Apply(element, vnode)

    // Setup event listeners
    c.attachEventListeners()

    // Setup state watchers
    c.state.Watch(func(changes []StateChange) {
        c.rerender(changes)
    })
}

func (c *UserCardCSR) render() *VNode {
    // Virtual DOM generation
    return h("div", Attrs{"className": "user-card"},
        // ... children
    )
}
```

## Phase 4: Style System (Week 7)

### 4.1 CSS-in-Gox Processing
```go
type StyleProcessor struct {
    scoper   *CSSScoper
    minifier *CSSMinifier
    autoprefixer *Autoprefixer
}

func (sp *StyleProcessor) Process(styles string, componentID string) ProcessedStyles {
    // Scope CSS to component
    scoped := sp.scoper.Scope(styles, componentID)

    // Add vendor prefixes
    prefixed := sp.autoprefixer.Process(scoped)

    // Minify for production
    minified := sp.minifier.Minify(prefixed)

    return ProcessedStyles{
        Development: scoped,
        Production:  minified,
        ComponentID: componentID,
    }
}

// CSS scoping with data attributes
func (s *CSSScoper) Scope(css string, id string) string {
    // Parse CSS using Electron's fast parser
    rules := electron.parse.CSS(css)

    // Add component scope
    for _, rule := range rules {
        rule.Selector = s.addScope(rule.Selector, id)
    }

    return electron.stringify.CSS(rules)
}
```

### 4.2 Runtime Style Injection
```go
// Client-side style management
type StyleManager struct {
    styles   map[string]string
    injected map[string]bool
    sheet    js.Value
}

func (sm *StyleManager) Inject(componentID string, css string) {
    if sm.injected[componentID] {
        return
    }

    // Create style element
    style := js.Global().Get("document").Call("createElement", "style")
    style.Set("innerHTML", css)
    style.Set("data-component", componentID)

    // Inject into head
    head := js.Global().Get("document").Get("head")
    head.Call("appendChild", style)

    sm.injected[componentID] = true
}
```

## Phase 5: Expression System (Week 8)

### 5.1 Template Expression Parser
```go
type ExpressionParser struct {
    // Parse ${} expressions in templates
}

func (ep *ExpressionParser) Parse(template string) []Expression {
    expressions := []Expression{}

    // Use Electron's SIMD for fast scanning
    positions := simd.FindAll(template, "${")

    for _, pos := range positions {
        end := ep.findClosing(template, pos)
        expr := template[pos+2:end]

        expressions = append(expressions, Expression{
            Start:  pos,
            End:    end,
            Source: expr,
            AST:    ep.parseExpression(expr),
        })
    }

    return expressions
}

// Expression evaluation with sandboxing
func (e *Expression) Evaluate(context *Context) (interface{}, error) {
    // Safe evaluation with type checking
    evaluator := NewSafeEvaluator(context)
    return evaluator.Eval(e.AST)
}
```

### 5.2 Markdown Support
```go
type MarkdownProcessor struct {
    parser   *markdown.Parser
    renderer *HTMLRenderer
}

func (mp *MarkdownProcessor) Process(md string, context *Context) string {
    // Parse markdown
    doc := mp.parser.Parse([]byte(md))

    // Process expressions in markdown
    mp.processExpressions(doc, context)

    // Render to HTML
    return mp.renderer.Render(doc)
}

// Process ${} expressions within markdown
func (mp *MarkdownProcessor) processExpressions(node *ast.Node, ctx *Context) {
    if node.Type == ast.Text {
        text := node.Literal
        if strings.Contains(text, "${") {
            // Replace expressions with evaluated values
            node.Literal = mp.evaluateExpressions(text, ctx)
        }
    }

    for child := node.FirstChild; child != nil; child = child.Next {
        mp.processExpressions(child, ctx)
    }
}
```

## Phase 6: Compilation Pipeline (Week 9-10)

### 6.1 Multi-Target Compiler
```go
type GoxCompiler struct {
    parser      *GoxParser
    transformer *ASTTransformer
    generators  map[CompileTarget]CodeGenerator
}

type CompileTarget string
const (
    TargetGoSSR   CompileTarget = "go-ssr"
    TargetWASM    CompileTarget = "wasm"
    TargetJS      CompileTarget = "js"
    TargetHybrid  CompileTarget = "hybrid"
)

func (gc *GoxCompiler) Compile(config CompileConfig) (*CompileResult, error) {
    // Parse all .gox files
    components := gc.parseComponents(config.Sources)

    // Transform AST based on target
    transformed := gc.transformer.Transform(components, config.Target)

    // Generate code
    generator := gc.generators[config.Target]
    result := generator.Generate(transformed)

    // Optimize using Electron's optimizers
    optimized := gc.optimize(result)

    return optimized, nil
}
```

### 6.2 WASM Optimization
```go
type WASMOptimizer struct {
    tinygo   *TinyGoCompiler
    binaryen *BinaryenOptimizer
}

func (wo *WASMOptimizer) Optimize(goCode string) ([]byte, error) {
    // Compile with TinyGo for smaller binaries
    wasm := wo.tinygo.Compile(goCode, TinyGoConfig{
        Target:     "wasm",
        Opt:        "z",  // Size optimization
        NoDebug:    true,
        Scheduler:  "none",
        GC:         "leaking",  // Use Electron's arena instead
    })

    // Further optimize with Binaryen
    optimized := wo.binaryen.Optimize(wasm, BinaryenConfig{
        OptLevel:    3,
        ShrinkLevel: 2,
        Converge:    true,
    })

    return optimized, nil
}
```

### 6.3 JavaScript Generation
```go
type JSGenerator struct {
    esVersion ESVersion
    minifier  *JSMinifier
}

func (jg *JSGenerator) Generate(component *Component) string {
    // Generate ES6+ JavaScript
    js := jg.generateComponent(component)

    // Add React-compatible runtime
    js = jg.addRuntime(js)

    // Minify for production
    if jg.minifier != nil {
        js = jg.minifier.Minify(js)
    }

    return js
}

// Generate React-compatible JavaScript
func (jg *JSGenerator) generateComponent(c *Component) string {
    return fmt.Sprintf(`
        class %s extends React.Component {
            constructor(props) {
                super(props);
                this.state = %s;
            }

            %s  // Methods

            render() {
                return %s;
            }
        }
    `, c.Name, c.InitialState(), c.Methods(), c.JSXTemplate())
}
```

## Phase 7: Runtime Implementation (Week 11)

### 7.1 Virtual DOM (CSR)
```go
type VirtualDOM struct {
    current  *VNode
    previous *VNode
    patches  []Patch
    arena    *arena.Arena
}

type VNode struct {
    Type     string
    Props    map[string]interface{}
    Children []VNode
    Key      string
}

func (vdom *VirtualDOM) Diff(old, new *VNode) []Patch {
    patches := []Patch{}

    // Efficient diffing using Electron's fast comparison
    if !electron.DeepEqual(old.Props, new.Props) {
        patches = append(patches, UpdateProps{...})
    }

    // Recursive child diffing with keys
    patches = append(patches, vdom.diffChildren(old.Children, new.Children)...)

    return patches
}

func (vdom *VirtualDOM) Apply(element js.Value, patches []Patch) {
    for _, patch := range patches {
        patch.Apply(element)
    }
}
```

### 7.2 Server-Side Rendering (SSR)
```go
type SSRRenderer struct {
    components map[string]ComponentFactory
    cache      *electron.Cache
    pool       *electron.BufferPool
}

func (r *SSRRenderer) Render(component string, props map[string]interface{}) string {
    // Check cache
    key := r.cacheKey(component, props)
    if cached, ok := r.cache.Get(key); ok {
        return cached.(string)
    }

    // Get buffer from pool
    buf := r.pool.Get(4096)
    defer r.pool.Put(buf)

    // Create component instance
    factory := r.components[component]
    instance := factory(props)

    // Render to string
    html := instance.RenderToString(buf)

    // Cache result
    r.cache.Set(key, html)

    return html
}
```

### 7.3 Hydration Bridge
```go
type HydrationBridge struct {
    // Bridge between SSR and CSR
}

func (hb *HydrationBridge) Hydrate(element js.Value, component Component) {
    // Extract SSR state from DOM
    stateData := hb.extractHydrationData(element)

    // Initialize component with SSR state
    component.InitWithState(stateData)

    // Attach event listeners
    hb.attachEventListeners(element, component)

    // Mark as hydrated
    element.Set("data-hydrated", true)
}
```

## Phase 8: Integration & Testing (Week 12)

### 8.1 Bolt/Shockwave Integration
```go
// Bolt handler with Conduit SSR
func UserProfileHandler(c *bolt.Context) error {
    userID := c.Param("id")
    user := getUserByID(userID)

    // Render component server-side
    html := conduit.Render("UserProfile", map[string]interface{}{
        "user": user,
        "isAdmin": c.Get("isAdmin"),
    })

    return c.HTML(200, html)
}

// Shockwave with streaming SSR
func StreamingSSR(w shockwave.ResponseWriter, r *shockwave.Request) {
    // Stream component chunks as they render
    stream := conduit.NewStreamRenderer(w)

    stream.WriteHead(`<!DOCTYPE html><html><head>...`)
    stream.WriteComponent("Header", headerProps)
    stream.WriteComponent("MainContent", contentProps)
    stream.WriteTail(`</body></html>`)
}
```

### 8.2 Development Tools
```go
// Hot Module Replacement for development
type HMRServer struct {
    watcher  *FileWatcher
    compiler *GoxCompiler
    ws       *websocket.Conn
}

func (hmr *HMRServer) Start() {
    hmr.watcher.Watch("**/*.gox", func(path string) {
        // Recompile changed component
        component := hmr.compiler.CompileFile(path)

        // Send update to browser
        hmr.ws.Send(HMRUpdate{
            Type:      "component-update",
            Component: component.Name,
            Code:      component.JSCode,
        })
    })
}

// DevTools Chrome Extension integration
type DevTools struct {
    bridge *DevToolsBridge
}

func (dt *DevTools) InspectComponent(id string) ComponentInfo {
    // Return component state, props, and render tree
}
```

## API Examples

### Basic Component
```gox
component HelloWorld(name string) {
    render {
        <div>Hello, ${name}!</div>
    }
}
```

### Stateful Component
```gox
component Counter() {
    state {
        count: int = 0
    }

    style {
        .counter {
            padding: 20px;
            text-align: center;
        }
    }

    render {
        <div className="counter">
            <h1>Count: ${state.count}</h1>
            <button onClick={() => state.count++}>Increment</button>
        </div>
    }
}
```

### Complex Component with Hooks
```gox
component TodoApp() {
    // Keyword state
    state {
        filter: string = "all"
    }

    // Hook state
    const [todos, setTodos] = useState([])
    const [input, setInput] = useState("")

    // Effects
    useEffect(() => {
        const saved = localStorage.getItem("todos")
        if (saved) setTodos(JSON.parse(saved))
    }, [])

    useEffect(() => {
        localStorage.setItem("todos", JSON.stringify(todos))
    }, [todos])

    // Computed
    const filtered = useMemo(() => {
        switch(state.filter) {
            case "active": return todos.filter(t => !t.done)
            case "completed": return todos.filter(t => t.done)
            default: return todos
        }
    }, [todos, state.filter])

    style {
        .todo-app {
            max-width: 600px;
            margin: 0 auto;
        }

        .filters button.active {
            font-weight: bold;
        }
    }

    render {
        <div className="todo-app">
            <h1>Todo App</h1>

            <div className="input">
                <input
                    value="${input}"
                    onChange={(e) => setInput(e.target.value)}
                    onKeyPress={(e) => {
                        if (e.key === "Enter" && input) {
                            setTodos([...todos, {
                                id: Date.now(),
                                text: input,
                                done: false
                            }])
                            setInput("")
                        }
                    }}
                    placeholder="Add a todo..."
                />
            </div>

            <div className="filters">
                <button
                    className="${state.filter === 'all' ? 'active' : ''}"
                    onClick={() => state.filter = "all"}>
                    All
                </button>
                <button
                    className="${state.filter === 'active' ? 'active' : ''}"
                    onClick={() => state.filter = "active"}>
                    Active
                </button>
                <button
                    className="${state.filter === 'completed' ? 'active' : ''}"
                    onClick={() => state.filter = "completed"}>
                    Completed
                </button>
            </div>

            <ul className="todos">
                ${filtered.map(todo =>
                    <TodoItem
                        key={todo.id}
                        todo={todo}
                        onToggle={() => {
                            setTodos(todos.map(t =>
                                t.id === todo.id ?
                                {...t, done: !t.done} : t
                            ))
                        }}
                        onDelete={() => {
                            setTodos(todos.filter(t => t.id !== todo.id))
                        }}
                    />
                )}
            </ul>

            <markdown>
                ## Stats
                - Total: ${todos.length}
                - Active: ${todos.filter(t => !t.done).length}
                - Completed: ${todos.filter(t => t.done).length}
            </markdown>
        </div>
    }
}
```

## Compilation Examples

### SSR Compilation
```bash
# Compile for server-side rendering
conduit compile --target=ssr --output=./dist/ssr ./src/**/*.gox

# Generated Go files can be used with Bolt
import "dist/ssr/components"

app.Get("/", func(c *bolt.Context) error {
    html := components.RenderHomePage(props)
    return c.HTML(200, html)
})
```

### CSR Compilation
```bash
# Compile to WASM
conduit compile --target=wasm --output=./dist/wasm ./src/**/*.gox
GOOS=js GOARCH=wasm go build -o app.wasm ./dist/wasm

# Compile to JavaScript
conduit compile --target=js --output=./dist/js ./src/**/*.gox
```

### Hybrid Compilation
```bash
# Generate both SSR and CSR
conduit compile --target=hybrid --output=./dist ./src/**/*.gox

# Results in:
# dist/
#   ssr/      - Go files for server rendering
#   wasm/     - Go files for WASM compilation
#   js/       - JavaScript files
#   styles/   - Extracted CSS files
```

## Performance Targets
- Parse time: <100ms for 1000 components
- Compile time: <1s for 100 components
- SSR render: <1ms per component
- CSR mount: <16ms (60fps)
- WASM size: <100KB for runtime + components
- JS size: <50KB minified + gzipped
- Zero allocations in render path (via Electron)

## Success Metrics
- [ ] .gox parser with full Go support
- [ ] Component keyword implementation
- [ ] State keyword with dual API
- [ ] Style block with scoping
- [ ] ${} expression interpolation
- [ ] Markdown support in components
- [ ] WASM compilation working
- [ ] JS compilation working
- [ ] SSR rendering functional
- [ ] CSR hydration working
- [ ] Hot Module Replacement
- [ ] DevTools integration
- [ ] 100% compatibility with old GoX features
- [ ] Performance targets met

## Dependencies
- Electron (all performance primitives)
- TinyGo (WASM compilation)
- Binaryen (WASM optimization)
- ESBuild (JavaScript bundling)
- Markdown parser
- CSS parser/minifier

## Risk Mitigation
- **Risk**: Parser complexity for Go superset
  - **Mitigation**: Reuse Go's parser, extend for new keywords
- **Risk**: WASM binary size
  - **Mitigation**: TinyGo + aggressive tree shaking
- **Risk**: State synchronization complexity
  - **Mitigation**: Clear separation, one-way data flow
- **Risk**: Performance regression from GoX
  - **Mitigation**: Electron primitives, zero-allocation design