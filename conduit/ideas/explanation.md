# Go Frontend Framework - Complete Syntax Reference

## Overview

This is a **Go-based frontend framework** that extends Go with reactive, component-based UI primitives. It's designed as a **superset of Go**, meaning all valid Go code works, plus additional syntax for building reactive user interfaces.

### Core Philosophy

1. **Go-idiomatic**: Leverages Go's strengths (goroutines, channels, context) rather than fighting them
2. **Reactive by default**: State changes automatically trigger re-renders
3. **Optimized automatically**: Memoization, batching, and reconciliation happen without explicit APIs
4. **Explicit dependencies**: Side effects declare what they depend on
5. **No colored functions**: Async operations use goroutines, not async/await syntax (except for the `await` helper)

---

## Component Declaration

Components are declared using the `component` keyword followed by a name and optional parameters.

```go
component ComponentName(param1 Type1, param2 Type2 = defaultValue) {
    // Component body
    return <JSX>
}
```

### Component Parameters (Props)

Props are simply function parameters with optional default values:

```go
component Button(
    label string = "Click me",
    variant string = "primary",
    disabled bool = false,
    onclick func() = nil,
) {
    return <button onclick={onclick}>{label}</button>
}

// Usage
<Button label="Submit" variant="secondary" />
```

**Key Points:**
- Props are immutable (function parameters)
- Default values are supported with `=` syntax
- Props can be any Go type including functions

---

## State Management

### State Block

Declares reactive state variables that trigger re-renders when changed.

```go
state {
    count int = 0
    name string = ""
    items []Item = []Item{}
}
```

**State API:**
- `variableName.set(newValue)` - Update state and trigger re-render
- Direct access: `{count}` in JSX or `count` in Go code

**Important:**
- All state updates are **automatically batched**
- Multiple `.set()` calls in the same function result in a single re-render
- State is local to the component instance

### Computed Values

Derived values from **state** that automatically update when dependencies change.

```go
compute {
    double int = count * 2
    isEven bool = count % 2 == 0
    filtered []Item = filterItems(items, query)
}
```

**Key Points:**
- Values are automatically memoized
- Recompute only when state dependencies change
- No manual dependency tracking needed
- Cannot have side effects (pure computations only)

### Derive Block

Derived values from **props** (and other non-state sources).

```go
component UserCard(user User, showEmail bool) {
    derive {
        fullName string = user.FirstName + " " + user.LastName
        shouldShowContact bool = showEmail && user.Email != ""
        initials string = string(user.FirstName[0]) + string(user.LastName[0])
    }
    
    return <div>{fullName}</div>
}
```

**Difference from `compute`:**
- `compute` - Derived from **state** (reactive variables)
- `derive` - Derived from **props** (function parameters)
- Both are automatically memoized

---

## Side Effects

### Effect Block

Runs side effects when dependencies change.

```go
effect (dependency1, dependency2) {
    // Effect code
    fmt.Println("Dependencies changed:", dependency1, dependency2)
    
    cleanup {
        // Cleanup code (runs before next effect or on unmount)
        fmt.Println("Cleaning up")
    }
    
    error {
        // Error handling
        fmt.Println("Effect failed:", err)
    }
}
```

**Effect without dependencies** (runs once on mount):
```go
effect () {
    fmt.Println("Component mounted")
    
    cleanup {
        fmt.Println("Component unmounted")
    }
}
```

**Key Points:**
- Dependencies are explicitly listed in parentheses
- Effects run after the component renders
- `cleanup` block runs before the next effect execution or on unmount
- Can contain async operations (goroutines)
- Multiple effects can be declared

### Watch Block

Explicit watcher with access to old and new values.

```go
watch (variable) (oldValue, newValue) {
    fmt.Printf("Changed from %v to %v\n", oldValue, newValue)
    
    if newValue == "" {
        results.set([]Result{})
    }
}
```

**Difference from `effect`:**
- `watch` provides both old and new values
- More explicit about what's being watched
- Typically used when you need the previous value for comparison

---

## Lifecycle Hooks

### Mount

Runs once when component is first rendered.

```go
mount {
    fmt.Println("Component mounted")
    initializeConnection()
}
```

### Unmount

Runs once when component is removed from the DOM.

```go
unmount {
    fmt.Println("Component unmounted")
    closeConnections()
}
```

### Cleanup

Runs before re-render or unmount (typically used inside `effect`).

```go
cleanup {
    cancelRequest()
    clearTimers()
}
```

**Lifecycle Order:**
1. Component renders
2. `mount` runs (first render only)
3. Effects run
4. On changes: `cleanup` → re-render → effects run
5. On unmount: `cleanup` → `unmount`

---

## Refs (Non-Reactive References)

Refs store mutable values that **don't** trigger re-renders when changed.

```go
component Timer() {
    ref intervalId int = 0
    ref renderCount int = 0
    ref inputRef *Element
    
    state {
        seconds int = 0
    }
    
    mount {
        inputRef.focus()
    }
    
    effect () {
        intervalId = setInterval(func() {
            seconds.set(seconds + 1)
        }, 1000)
        
        cleanup {
            clearInterval(intervalId)
        }
    }
    
    return <input ref={inputRef} />
}
```

**Use Cases:**
- DOM element references (`ref={inputRef}`)
- Storing interval/timeout IDs
- Tracking values without triggering re-renders
- Storing previous values for comparison

**Key Points:**
- Direct assignment: `refName = value` (no `.set()`)
- Direct access: `refName`
- Persists across re-renders
- Changes don't trigger re-renders

---

## Context (Cross-Component State)

### Context Definition

Define shared state structure.

```go
context ThemeContext {
    Theme string
    SetTheme func(string)
}

context AuthContext {
    User User
    IsAuthenticated bool
    Login func(string, string) error
    Logout func()
}
```

### Providing Context

Make context available to child components.

```go
component App() {
    state {
        theme string = "dark"
        user User = User{}
    }
    
    provide {
        ThemeContext: {
            Theme: theme,
            SetTheme: theme.set,
        },
        AuthContext: {
            User: user,
            IsAuthenticated: user.ID != "",
            Login: handleLogin,
            Logout: handleLogout,
        }
    }
    
    return <Layout>{children}</Layout>
}
```

### Consuming Context

Access context in any descendant component.

```go
component Header() {
    consume theme = ThemeContext
    consume auth = AuthContext
    
    return (
        <header class={theme.Theme}>
            {auth.IsAuthenticated ? (
                <UserMenu user={auth.User} />
            ) : (
                <LoginButton />
            )}
        </header>
    )
}
```

**Syntax:** `consume variableName = ContextName`

**Key Points:**
- Context flows down the component tree
- Any descendant can consume without prop drilling
- Multiple contexts can be consumed in one component
- Context updates trigger re-renders in consumers

---

## Async Operations

The framework supports **three patterns** for async operations, giving developers flexibility based on their needs.

### 1. Resources (Cached Data Fetching)

**Best for:** Shared data that should be cached and deduplicated.

```go
// Define a resource
resource UserResource(userId string) (User, error) {
    return fetchUser(userId)
}

resource PostsResource(userId string) ([]Post, error) {
    return fetchUserPosts(userId)
}

// Use in components
component UserProfile(userId string) {
    user, err := useResource(UserResource, userId)
    
    if err != nil {
        return <ErrorMessage error={err} />
    }
    
    return <div>{user.Name}</div>
}
```

**Key Features:**
- Automatic caching: Multiple components using the same resource + args share data
- Automatic deduplication: Only one request for the same resource + args
- Framework manages loading states via Suspense boundaries
- Suspends component rendering until data is ready

### 2. Await (Inline Async Operations)

**Best for:** One-off operations, form submissions, file uploads.

```go
component SubmitForm() {
    state {
        result string = ""
        submitting bool = false
    }
    
    handleSubmit := func(formData FormData) {
        submitting.set(true)
        
        // Blocks rendering until complete, but doesn't block UI
        response := await(submitToAPI(formData))
        
        result.set(response.Message)
        submitting.set(false)
    }
    
    return <form onsubmit={handleSubmit}>...</form>
}
```

**Key Features:**
- Synchronous-looking code that's actually async
- No colored functions problem
- Integrates with Suspense boundaries
- No automatic caching

### 3. Manual Goroutines (Full Control)

**Best for:** WebSockets, polling, custom async flows, complex cancellation.

```go
component LiveFeed() {
    state {
        messages []Message = []Message{}
        connected bool = false
    }
    
    effect () {
        ctx, cancel := context.WithCancel(context.Background())
        
        go func() {
            ws, err := connectWebSocket(ctx, "wss://example.com")
            if err != nil {
                return
            }
            
            connected.set(true)
            
            for {
                select {
                case <-ctx.Done():
                    ws.Close()
                    return
                case msg := <-ws.Messages:
                    messages.set(append(messages, msg))
                }
            }
        }()
        
        cleanup {
            cancel() // Cancels goroutine
        }
    }
    
    return <div>Messages: {len(messages)}</div>
}
```

**Key Features:**
- Full Go concurrency primitives (goroutines, channels, context)
- Complete control over cancellation
- Manual state management
- No framework magic

**When to Use What:**

| Pattern | Use Case | Caching | Complexity |
|---------|----------|---------|------------|
| `resource` + `useResource` | Fetching user data, posts, configs | ✅ Automatic | Low |
| `await` | Submit form, upload file, one-time API call | ❌ No | Lowest |
| Manual goroutines | WebSocket, polling, complex flows | 🔧 DIY | High |

---

## Suspense and Error Boundaries

### Suspense

Handles loading states for async operations.

```go
component App() {
    return (
        <Suspense fallback={<LoadingSpinner />}>
            <UserProfile userId="123" />
        </Suspense>
    )
}
```

**How it works:**
- Resources and `await` suspend rendering until ready
- Nearest `Suspense` ancestor shows the fallback
- Multiple suspending components share one loading state

### Error Boundary

Catches errors in child components.

```go
component App() {
    return (
        <ErrorBoundary fallback={<ErrorPage />}>
            <UserDashboard />
        </ErrorBoundary>
    )
}
```

**Error propagation:**
- Errors in resources, `await`, or component render are caught
- Nearest `ErrorBoundary` renders the fallback
- Prevents entire app from crashing

---

## JSX Syntax

### Basic JSX

```go
return (
    <div class="container">
        <h1>Hello, {name}</h1>
        <button onclick={handleClick}>Click me</button>
    </div>
)
```

### Interpolation

Use `{}` for expressions and `${}` for simpler string interpolation:

```go
return (
    <div>
        <h1>Count: {count}</h1>
        <p>Double: ${double}</p>
        <p>Is Even: {isEven ? "Yes" : "No"}</p>
    </div>
)
```

### Conditional Rendering

```go
// Ternary
{isLoggedIn ? <Dashboard /> : <Login />}

// Logical AND
{hasError && <ErrorMessage />}

// If-else in Go
if loading {
    return <Loading />
}

return <Content />
```

### Lists and Keys

```go
{items.map(item => (
    <ItemCard key={item.ID} item={item} />
))}
```

**Key points:**
- Always provide `key` for list items
- Keys help with reconciliation and performance
- Keys should be stable and unique

### Fragments

Return multiple elements without a wrapper:

```go
return (
    <>
        <Header />
        <Content />
        <Footer />
    </>
)
```

### Children

Components automatically receive `children`:

```go
component Card() {
    return (
        <div class="card">
            <div class="card-body">
                {children}
            </div>
        </div>
    )
}

// Usage
<Card>
    <h2>Title</h2>
    <p>Content</p>
</Card>
```

### Event Handlers

```go
// Inline function
<button onclick={count.set(count + 1)}>Increment</button>

// Named function
handleClick := func() {
    count.set(count + 1)
}
<button onclick={handleClick}>Increment</button>

// With event parameter
handleInput := func(e Event) {
    value.set(e.target.value)
}
<input oninput={handleInput} />
```

**Common events:**
- `onclick`, `ondblclick`
- `onchange`, `oninput`
- `onsubmit`
- `onmouseenter`, `onmouseleave`
- `onkeydown`, `onkeyup`

---

## Complete Example: Putting It All Together

```go
// Context definition
context UserContext {
    CurrentUser User
    SetUser func(User)
}

// Resource for data fetching
resource UserResource(userId string) (User, error) {
    return fetchUser(userId)
}

// Root component with context provider
component App() {
    state {
        currentUser User = User{}
    }
    
    provide {
        UserContext: {
            CurrentUser: currentUser,
            SetUser: currentUser.set,
        }
    }
    
    return (
        <Suspense fallback={<LoadingScreen />}>
            <ErrorBoundary fallback={<ErrorPage />}>
                <Dashboard />
            </ErrorBoundary>
        </Suspense>
    )
}

// Component using all features
component UserProfile(userId string) {
    // Context consumption
    consume userCtx = UserContext
    
    // Resource (cached data)
    user, err := useResource(UserResource, userId)
    
    // Refs
    ref inputRef *Element
    ref previousName string = ""
    
    // State
    state {
        isEditing bool = false
        name string = ""
    }
    
    // Derive from props
    derive {
        isOwnProfile bool = userCtx.CurrentUser.ID == userId
    }
    
    // Compute from state
    compute {
        hasChanges bool = name != user.Name
        isValidName bool = len(name) >= 2
    }
    
    // Lifecycle
    mount {
        fmt.Println("Profile mounted")
        name.set(user.Name)
    }
    
    unmount {
        fmt.Println("Profile unmounted")
    }
    
    // Effects
    effect (isEditing) {
        if isEditing {
            inputRef.focus()
        }
    }
    
    // Watch
    watch (name) (oldName, newName) {
        fmt.Printf("Name changed: %s -> %s\n", oldName, newName)
        previousName = oldName
    }
    
    // Event handlers
    handleSave := func() {
        updated := await(updateUser(userId, name))
        userCtx.SetUser(updated)
        isEditing.set(false)
    }
    
    // Error handling
    if err != nil {
        return <ErrorMessage error={err} />
    }
    
    // Render
    return (
        <div class="profile">
            {isEditing ? (
                <input 
                    ref={inputRef}
                    value={name}
                    oninput={e => name.set(e.target.value)}
                />
            ) : (
                <h1>{user.Name}</h1>
            )}
            
            {isOwnProfile && (
                isEditing ? (
                    <>
                        <Button 
                            label="Save" 
                            onclick={handleSave}
                            disabled={!hasChanges || !isValidName}
                        />
                        <Button 
                            label="Cancel"
                            onclick={isEditing.set(false)}
                        />
                    </>
                ) : (
                    <Button 
                        label="Edit"
                        onclick={isEditing.set(true)}
                    />
                )
            )}
        </div>
    )
}
```

---

## Advanced Patterns

### Debouncing with Goroutines

```go
effect (searchQuery) {
    if searchQuery == "" {
        results.set([]Result{})
        return
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        timer := time.NewTimer(300 * time.Millisecond)
        
        select {
        case <-ctx.Done():
            timer.Stop()
            return
        case <-timer.C:
            results, _ := searchAPI(ctx, searchQuery)
            results.set(results)
        }
    }()
    
    cleanup {
        cancel()
    }
}
```

### Polling

```go
effect () {
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                data, _ := fetchData(ctx)
                data.set(data)
            }
        }
    }()
    
    cleanup {
        cancel()
    }
}
```

### WebSocket Connection

```go
effect (channelId) {
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        ws, err := connectWebSocket(ctx, channelId)
        if err != nil {
            error.set(err)
            return
        }
        
        connected.set(true)
        
        for {
            select {
            case <-ctx.Done():
                ws.Close()
                return
            case msg := <-ws.Messages:
                messages.set(append(messages, msg))
            case err := <-ws.Errors:
                error.set(err)
                return
            }
        }
    }()
    
    cleanup {
        cancel()
    }
}
```

---

## Best Practices

### State Management
- Keep state as local as possible
- Use context for truly global state
- Prefer `compute` and `derive` over manual synchronization
- Don't put derived values in state

### Effects
- Always specify dependencies explicitly
- Use `cleanup` for cancellation and resource cleanup
- Avoid setting state in effects when possible (prefer `compute`)
- Keep effects focused and small

### Async Operations
- Use `resource` for data that should be cached
- Use `await` for one-off operations
- Use goroutines for complex flows (WebSockets, polling)
- Always cancel goroutines in `cleanup`

### Performance
- Framework handles memoization automatically
- Don't worry about batching - it's automatic
- Use `key` properly in lists
- Keep components small and focused

### Refs
- Use refs for DOM manipulation
- Use refs for values that shouldn't trigger re-renders
- Don't read refs during render (use in effects or event handlers)

---

## Summary of Keywords

| Keyword | Purpose |
|---------|---------|
| `component` | Define a component |
| `state` | Reactive state variables |
| `compute` | Derived values from state |
| `derive` | Derived values from props |
| `ref` | Non-reactive mutable references |
| `effect` | Side effects with dependencies |
| `watch` | Explicit watchers with old/new values |
| `mount` | Runs once on mount |
| `unmount` | Runs once on unmount |
| `cleanup` | Cleanup before re-run or unmount |
| `error` | Error handling in effects |
| `context` | Define shared state structure |
| `provide` | Provide context to children |
| `consume` | Consume context from ancestors |
| `resource` | Define cacheable async data source |
| `useResource` | Use a resource in a component |
| `await` | Inline async operation |
| `Suspense` | Loading boundary |
| `ErrorBoundary` | Error boundary |

---

## Type System

The framework uses Go's type system with some additions:

- `Element` - DOM element reference
- `Event` - Browser event
- `FormData` - Form data
- All standard Go types work as expected
- JSX returns renderable elements

---

This syntax provides a complete, production-ready framework that feels natural to Go developers while providing modern reactive UI capabilities.