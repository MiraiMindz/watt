---
name: transpiler-development
description: Expert in code generation for SSR and CSR modes. Use when generating Go code from AST, implementing VNode generation, or optimizing transpiler output.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Transpiler Development Skill

Expert in generating efficient Go code from GoX IR for both SSR (Server-Side Rendering) and CSR (Client-Side Rendering/WASM).

## SSR Transpiler Pattern

**Input:** ComponentIR from Analyzer
**Output:** Go struct + Render() method returning HTML string

```go
// Generated code structure
type <ComponentName> struct {
    *gox.Component
    <PropFields>
    <StateFields>
}

func New<ComponentName>(<props>) *<ComponentName> {
    c := &<ComponentName>{
        Component: gox.NewComponent(),
        <prop assignments>
    }
    <state initialization>
    return c
}

func (c *<ComponentName>) Render() string {
    gox.SetCurrentComponent(c.Component)
    defer func() { gox.SetCurrentComponent(nil) }()
    c.Component.hooks.Reset()

    return `<html template with ${interpolations}>`
}

func (c *<ComponentName>) <setterName>(value <Type>) {
    c.<field> = value
    c.RequestUpdate()
}
```

## CSR Transpiler Pattern

**Output:** VNode tree generation

```go
func (c *<ComponentName>) Render() *gox.VNode {
    gox.SetCurrentComponent(c.Component)
    defer func() { gox.SetCurrentComponent(nil) }()

    return gox.H("<tag>", gox.Props{
        "className": "...",
        "onClick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
            c.<handler>()
            return nil
        }),
    },
        <children VNodes>
    )
}
```

## Key Responsibilities

1. **IR → Go Code** - Transform ComponentIR to valid Go
2. **Expression Interpolation** - Handle ${} in templates
3. **Event Handler Wrapping** - Convert Go funcs to js.FuncOf
4. **State Setter Generation** - Create setter methods
5. **Type Safety** - Preserve type information

## Performance Targets

- ~100 components/s
- Zero runtime reflection
- Minimal allocations
- Inline event handlers when possible

## Testing

```go
func TestSSRTranspiler(t *testing.T) {
    ir := &analyzer.ComponentIR{
        Name: "Counter",
        State: []analyzer.StateVar{
            {Name: "count", Type: "int", Initial: "0"},
        },
    }

    code := ssr.Transpile(ir)

    // Verify generated code compiles
    // Verify Render() method exists
    // Verify setters exist
}
```
