# GoX/Conduit Development Guide

## Project Overview

**GoX** (also known as **Conduit**) is a React-inspired frontend framework for Go that compiles to both Server-Side Rendering (SSR) and Client-Side Rendering (WASM). It extends Go with `.gox` files combining:
- Go code (logic)
- JSX-like markup (UI)
- CSS (styling)

**Status:** Phase 1-9 complete (Lexer, Parser, Analyzer, SSR Transpiler, Runtime, Optimizer, CLI)
**In Progress:** Phase 10-12 (CSR Transpiler, Virtual DOM, WASM Runtime)
**Goal:** Production-ready React-like framework with Go's type safety and performance

---

## Architecture Principles

### 1. Compilation Pipeline
```
.gox Source
  ↓ Lexer (Multi-mode: Go/JSX/CSS)
Tokens
  ↓ Parser (Component declarations, hooks, JSX, CSS)
AST
  ↓ Analyzer (Semantic analysis, IR generation)
IR
  ↓ Optimizer (DCE, tree shaking, minification)
Optimized IR
  ↓ Transpiler (SSR or CSR mode)
.go Code
```

### 2. Core Components

**Required Components:**
- **Lexer** (`pkg/lexer/`) - Multi-mode tokenization
- **Parser** (`pkg/parser/`) - AST generation
- **Analyzer** (`pkg/analyzer/`) - Semantic analysis & IR
- **Optimizer** (`pkg/optimizer/`) - Production optimizations
- **Transpiler SSR** (`pkg/transpiler/ssr/`) - Go code generation
- **Transpiler CSR** (`pkg/transpiler/csr/`) - WASM code generation
- **Runtime** (`runtime/`) - Hooks, VNode, Component base
- **CLI** (`cmd/goxc/`) - Build tool

### 3. Language Features

**Component Syntax:**
```gox
component Counter(initial int) {
    count, setCount := gox.UseState[int](initial)

    render {
        <div className="counter">
            <p>Count: {count}</p>
            <button onClick={func() { setCount(count + 1) }}>+</button>
        </div>
    }

    style {
        .counter { text-align: center; }
        button { padding: 10px 20px; }
    }
}
```

**Supported Hooks:**
- `UseState[T]` - State management
- `UseEffect` - Side effects
- `UseMemo[T]` - Memoization
- `UseCallback[T]` - Callback memoization
- `UseRef[T]` - Mutable refs
- `UseContext[T]` - Context values
- `UseReducer[S, A]` - Redux-like state
- `UseId` - Unique IDs
- `UseTransition` - Non-urgent updates

---

## Coding Standards

### Go Code

**MUST:**
- Use Go 1.21+ features (generics for type-safe hooks)
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names (no single letters except `i`, `j` for loops)
- Add godoc comments for all exported functions/types
- Handle errors explicitly (no bare `_` without justification)
- Use `sync.RWMutex` for concurrent access
- Pre-allocate slices when size is known

**MUST NOT:**
- Use `interface{}` where generics can be used
- Panic in library code (return errors)
- Use global mutable state
- Import external dependencies (standard library only)

**Example:**
```go
// UseState creates a reactive state variable with type safety.
// Returns the current value and a setter function.
func UseState[T any](initial T) (T, func(T)) {
    comp := getCurrentComponent()
    if comp == nil {
        panic("UseState must be called within a component render")
    }

    comp.mu.Lock()
    defer comp.mu.Unlock()

    // Implementation...
}
```

### File Organization

```
pkg/
├── lexer/
│   ├── lexer.go         # Main lexer implementation
│   ├── token.go         # Token definitions
│   └── lexer_test.go    # Tests
├── parser/
│   ├── parser.go        # Parser implementation
│   ├── ast.go           # AST node definitions
│   └── parser_test.go
├── analyzer/
│   ├── analyzer.go      # Semantic analysis
│   ├── ir.go            # Intermediate representation
│   └── analyzer_test.go
├── optimizer/
│   ├── optimizer.go     # Optimization passes
│   └── optimizer_test.go
└── transpiler/
    ├── ssr/
    │   ├── transpiler.go
    │   └── transpiler_test.go
    └── csr/
        ├── transpiler.go
        └── transpiler_test.go
```

### Testing Requirements

**Unit Tests:**
- Minimum 70% coverage
- Test file for each source file (`*_test.go`)
- Table-driven tests for multiple cases
- Benchmark tests for performance-critical code

**Example:**
```go
func TestLexerGoMode(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []lexer.TokenType
    }{
        {
            name:  "component declaration",
            input: "component Counter()",
            expected: []lexer.TokenType{
                lexer.COMPONENT,
                lexer.IDENT,
                lexer.LPAREN,
                lexer.RPAREN,
            },
        },
        // More cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := lexer.New([]byte(tt.input), "test.gox")
            // Test implementation...
        })
    }
}
```

### Performance Guidelines

**Optimization Priorities:**
1. **Lexer:** Single-pass, no backtracking, ~1000 lines/ms target
2. **Parser:** LL(2) parsing, pre-allocated slices, ~500 lines/ms
3. **Analyzer:** Index-based lookups, ~200 components/s
4. **Optimizer:** 30-50% size reduction in production mode
5. **Runtime:** O(1) hook access, O(n) VNode diffing

**Critical Paths:**
- Hook state access (must be O(1))
- VNode diffing (must be O(n))
- Token scanning (single-pass only)

---

## Implementation Workflow

### Phase-by-Phase Development

**Current Phase (10-12): CSR & WASM**
1. CSR Transpiler - Generate VNode code
2. Virtual DOM - Diffing algorithm
3. WASM Runtime - Browser integration

**When Implementing New Features:**

1. **Read First**
   - Read related documentation (GOX_COMPLETE_BLUEPRINT.md, QUICK_REFERENCE.md)
   - Read existing code in the same component
   - Check IMPLEMENTATION_PLAN.md for context

2. **Design**
   - Write design doc in `ideas/` if complex
   - Get feedback before implementation
   - Update IR structures if needed

3. **Implement**
   - Write tests first (TDD)
   - Implement minimal viable version
   - Add godoc comments
   - Run `go fmt`

4. **Optimize**
   - Profile if performance-critical
   - Add benchmarks
   - Document optimizations in comments

5. **Document**
   - Update QUICK_REFERENCE.md
   - Add examples to `examples/`
   - Update IMPLEMENTATION_PLAN.md status

### Git Workflow

**Branch Naming:**
- `feat/phase-X-component-name` - New features
- `fix/issue-description` - Bug fixes
- `refactor/component-name` - Refactoring
- `docs/what-changed` - Documentation

**Commit Messages:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:** feat, fix, refactor, test, docs, perf, style
**Scope:** lexer, parser, analyzer, optimizer, ssr, csr, runtime, cli
**Example:**
```
feat(parser): add JSX fragment support

Implement parsing for <> and </> JSX fragments.
Adds FragmentNode to AST and updates parser to handle
empty tag syntax.

Closes #42
```

### Code Review Checklist

Before marking complete:
- [ ] Tests pass (`go test ./...`)
- [ ] Benchmarks pass (if applicable)
- [ ] Code formatted (`go fmt ./...`)
- [ ] No external dependencies added
- [ ] Godoc comments for exported items
- [ ] Error handling is explicit
- [ ] Thread safety considered (RWMutex if needed)
- [ ] Performance target met
- [ ] Examples updated
- [ ] QUICK_REFERENCE.md updated

---

## Error Handling

### Parser Errors

**Format:**
```
{filename}:{line}:{column}: {message}

component Counter() {
                   ^
expected '{' after component declaration
```

**Multiple Errors:**
- Collect all errors, don't stop at first
- Maximum 10 errors before stopping
- Use descriptive messages

**Example:**
```go
func (p *Parser) addError(msg string) {
    if len(p.errors) >= 10 {
        return
    }
    err := fmt.Errorf("%s:%d:%d: %s",
        p.filename, p.curToken.Line, p.curToken.Column, msg)
    p.errors = append(p.errors, err)
}
```

### Runtime Errors

**Panic Only For:**
- Programmer errors (hook outside component)
- Unrecoverable state

**Return Errors For:**
- Invalid input
- IO failures
- Compilation failures

---

## Development Tools

### CLI Usage

**Build:**
```bash
goxc build -mode=ssr -o=dist src/*.gox
goxc build -mode=csr -o=dist src/*.gox
```

**Watch Mode:**
```bash
goxc watch -mode=ssr src/
```

**Production:**
```bash
goxc build -mode=ssr -production -o=dist src/*.gox
```

### Testing

```bash
# All tests
go test ./...

# Specific package
go test -v ./pkg/lexer/

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. ./pkg/lexer/
```

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./pkg/lexer/
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./pkg/lexer/
go tool pprof mem.prof
```

---

## Anti-Patterns to Avoid

**DON'T:**
- ❌ Add external dependencies without discussion
- ❌ Use reflection for hot paths
- ❌ Allocate in tight loops
- ❌ Use global variables for component state
- ❌ Parse the same input multiple times
- ❌ Copy large structures (use pointers)
- ❌ Ignore errors with `_`
- ❌ Use `interface{}` instead of generics
- ❌ Panic in library code
- ❌ Block on I/O in rendering path

**DO:**
- ✅ Pre-allocate slices when size known
- ✅ Use sync.Pool for frequently allocated objects
- ✅ Profile before optimizing
- ✅ Write benchmarks for critical paths
- ✅ Use `strings.Builder` for string concatenation
- ✅ Return errors, handle gracefully
- ✅ Document performance characteristics
- ✅ Use generics for type safety
- ✅ Keep functions small and focused
- ✅ Write tests first

---

## Performance Targets

### Compilation Speed
- **Lexer:** ~1000 lines/ms
- **Parser:** ~500 lines/ms
- **Analyzer:** ~200 components/s
- **Transpiler:** ~100 components/s

### Runtime Performance
- **Hook Access:** O(1)
- **VNode Diff:** O(n) where n = node count
- **Re-render:** < 16ms (60fps)

### Optimization Gains
- **CSS Minification:** 30-50% reduction
- **HTML Minification:** 15-25% reduction
- **Dead Code Elimination:** 10-30% reduction
- **VNode Optimization:** 10-20% fewer nodes

---

## Documentation

### File Headers

```go
// Package lexer implements multi-mode tokenization for GoX source files.
// It supports three modes: Go, JSX, and CSS, with automatic mode switching
// based on context (render blocks → JSX, style blocks → CSS).
//
// The lexer is designed for single-pass scanning with no backtracking,
// achieving ~1000 lines/ms throughput.
package lexer
```

### Function Comments

```go
// New creates a new Lexer instance for the given input.
// The filename parameter is used for error reporting.
//
// The lexer starts in Go mode and automatically switches to JSX mode
// when entering render blocks and CSS mode for style blocks.
//
// Example:
//
//	input := []byte(`component Counter() { ... }`)
//	l := lexer.New(input, "Counter.gox")
//	for tok := l.NextToken(); tok.Type != lexer.EOF; tok = l.NextToken() {
//	    // Process token
//	}
func New(input []byte, filename string) *Lexer {
    // Implementation...
}
```

---

## Resources

### Key Documents
- **GOX_COMPLETE_BLUEPRINT.md** - Complete rebuild guide
- **IMPLEMENTATION_PLAN.md** - React-like implementation plan
- **QUICK_REFERENCE.md** - Fast reference for all features
- **ideas/syntax.gox** - Example syntax
- **ideas/explanation.md** - Language features explained

### Examples
- **examples/counter-ssr/** - SSR counter example
- **examples/todo-app/** - Full application example

---

## Communication

### Issue Reporting

**Format:**
```markdown
## Description
Brief description of the issue

## Steps to Reproduce
1. Step one
2. Step two

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Context
- GoX version:
- Go version:
- OS:

## Additional Info
Any relevant details
```

### Feature Requests

**Format:**
```markdown
## Feature Request
Clear description

## Use Case
Why is this needed?

## Proposed Solution
How should it work?

## Alternatives Considered
Other approaches

## Additional Context
Examples, references
```

---

## Quick Commands Reference

See `.claude/commands/` for available slash commands:
- `/plan` - Create implementation plan
- `/test` - Run test suite
- `/bench` - Run benchmarks
- `/review` - Code review checklist
- `/doc` - Generate documentation
- `/optimize` - Optimization suggestions

---

**Remember:** GoX aims to bring React's developer experience to Go's performance. Every decision should balance familiarity (for React developers) with Go idioms and type safety.
