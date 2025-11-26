# GoX/Conduit Claude Code Ecosystem Map

## 🗺️ Complete System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       CLAUDE CODE WORKSHOP                               │
│                    (Complete Development Environment)                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    CLAUDE.md (Constitution)                     │    │
│  │  • Coding Standards         • Performance Targets               │    │
│  │  • Architecture Principles  • Testing Requirements              │    │
│  │  • Git Workflow            • Anti-Patterns to Avoid             │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                    ↓                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    SKILLS (Recipe Cards)                        │    │
│  │                  Model-Invoked Expertise                        │    │
│  ├────────────────────────────────────────────────────────────────┤    │
│  │                                                                 │    │
│  │  📘 lexer-dev                                                   │    │
│  │     • Multi-mode tokenization (Go/JSX/CSS)                     │    │
│  │     • Performance optimization (~1000 lines/ms)                │    │
│  │     • UTF-8 handling, mode switching                           │    │
│  │     Tools: Read, Write, Edit, Bash, Grep, Glob                 │    │
│  │                                                                 │    │
│  │  📗 parser-dev                                                  │    │
│  │     • AST generation, component/hook/JSX/CSS parsing          │    │
│  │     • Error recovery, LL(2) parsing                            │    │
│  │     • Performance target: ~500 lines/ms                        │    │
│  │     Tools: Read, Write, Edit, Bash, Grep, Glob                 │    │
│  │                                                                 │    │
│  │  📕 transpiler-dev                                              │    │
│  │     • SSR code generation (Go structs + Render())             │    │
│  │     • CSR code generation (VNode trees)                        │    │
│  │     • Expression interpolation, event handlers                 │    │
│  │     Tools: Read, Write, Edit, Bash, Grep, Glob                 │    │
│  │                                                                 │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                    ↓                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │              AGENTS (Specialized Contractors)                   │    │
│  │              Independent Claude Instances                       │    │
│  ├────────────────────────────────────────────────────────────────┤    │
│  │                                                                 │    │
│  │  🎯 gox-planner                                                 │    │
│  │     Role: Creates detailed implementation plans                │    │
│  │     Tools: Read, Grep, Glob (NO Write/Edit)                    │    │
│  │     Output: plans/<feature>-plan.md                            │    │
│  │     ──────────────────────────────────────                     │    │
│  │                                                                 │    │
│  │  💻 gox-implementer                                             │    │
│  │     Role: Implements features following plans                  │    │
│  │     Tools: Read, Write, Edit, Bash, Grep, Glob                 │    │
│  │     Output: Production code + tests                            │    │
│  │     ──────────────────────────────────────                     │    │
│  │                                                                 │    │
│  │  🔍 gox-reviewer                                                │    │
│  │     Role: Reviews code quality and standards                   │    │
│  │     Tools: Read, Bash, Grep, Glob (NO Write/Edit)              │    │
│  │     Output: Review report with recommendations                 │    │
│  │     ──────────────────────────────────────                     │    │
│  │                                                                 │    │
│  │  🧪 gox-tester                                                  │    │
│  │     Role: Writes comprehensive tests                           │    │
│  │     Tools: Read, Write, Edit, Bash, Grep, Glob                 │    │
│  │     Output: Test suites, coverage reports                      │    │
│  │                                                                 │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                    ↓                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │               COMMANDS (Quick Action Buttons)                   │    │
│  │                   User-Invoked Shortcuts                        │    │
│  ├────────────────────────────────────────────────────────────────┤    │
│  │                                                                 │    │
│  │  /plan <feature>                                                │    │
│  │     → Invokes gox-planner agent                                │    │
│  │     → Creates plans/<feature>-plan.md                          │    │
│  │                                                                 │    │
│  │  /implement <plan-file>                                         │    │
│  │     → Invokes gox-implementer agent                            │    │
│  │     → Follows plan, writes code + tests                        │    │
│  │                                                                 │    │
│  │  /test [package]                                                │    │
│  │     → Runs test suite, shows coverage                          │    │
│  │     → Reports performance benchmarks                           │    │
│  │                                                                 │    │
│  │  /review [files]                                                │    │
│  │     → Invokes gox-reviewer agent                               │    │
│  │     → Quality check, standards compliance                      │    │
│  │                                                                 │    │
│  │  /bench [package]                                               │    │
│  │     → Runs benchmarks vs targets                               │    │
│  │     → Suggests optimizations                                   │    │
│  │                                                                 │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                    ↓                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │              HOOKS (Security Guards & Automators)               │    │
│  │              Event-Driven Shell Scripts                         │    │
│  ├────────────────────────────────────────────────────────────────┤    │
│  │                                                                 │    │
│  │  ⚡ PreToolUse                                                  │    │
│  │     validate-write.sh                                          │    │
│  │     • Blocks writes to .env, credentials.json, etc.            │    │
│  │     • Warns about generated files                              │    │
│  │     → Runs BEFORE Write/Edit tools                             │    │
│  │                                                                 │    │
│  │  ⚡ PostToolUse                                                 │    │
│  │     auto-format.sh                                             │    │
│  │     • Runs gofmt on .go files automatically                    │    │
│  │     → Runs AFTER Write/Edit on Go files                        │    │
│  │                                                                 │    │
│  │     auto-test.sh                                               │    │
│  │     • Runs tests for modified package                          │    │
│  │     → Runs AFTER Write/Edit on *_test.go files                 │    │
│  │                                                                 │    │
│  │  ⚡ SubagentStop                                                │    │
│  │     suggest-next-step (LLM-based)                              │    │
│  │     • Analyzes completed work                                  │    │
│  │     • Suggests logical next step                               │    │
│  │     → Runs AFTER any subagent completes                        │    │
│  │                                                                 │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                    ↓                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                  SETTINGS (Configuration)                       │    │
│  ├────────────────────────────────────────────────────────────────┤    │
│  │                                                                 │    │
│  │  settings.json                                                  │    │
│  │     • Hook configurations                                      │    │
│  │     • Skill enablement                                         │    │
│  │     • Agent definitions                                        │    │
│  │     • Tool permissions                                         │    │
│  │     • Performance targets                                      │    │
│  │     • Git workflow rules                                       │    │
│  │                                                                 │    │
│  │  settings.local.json (developer overrides)                     │    │
│  │     • Personal preferences                                     │    │
│  │     • Debug settings                                           │    │
│  │     • Not committed to git                                     │    │
│  │                                                                 │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Workflow Examples

### Example 1: Feature Development (Sequential)

```
User: "Add CSS scoping feature"
  ↓
/plan css-scoping
  ↓
gox-planner agent
  • Reads existing code
  • Creates plans/css-scoping-plan.md
  ↓
User reviews plan
  ↓
/implement plans/css-scoping-plan.md
  ↓
gox-implementer agent
  • Reads plan
  • Studies existing patterns
  • Writes code incrementally
  • Writes tests
  ↓
Hooks activate automatically:
  • validate-write (PreToolUse)
  • auto-format (PostToolUse)
  • auto-test (PostToolUse)
  ↓
SubagentStop hook:
  "✅ Implementation complete. Next: /review for quality check"
  ↓
/review
  ↓
gox-reviewer agent
  • Checks code quality
  • Verifies tests
  • Ensures standards compliance
  • Reports findings
  ↓
/test
  ↓
Full test suite
  • All tests pass
  • Coverage: 85%
  ↓
/bench
  ↓
Performance benchmarks
  • All targets met
  ↓
Ready to commit!
```

### Example 2: Parallel Execution

```
User: "Implement lexer optimization AND add parser edge case tests"
  ↓
Launch in parallel:
  ├─ gox-implementer agent (lexer optimization)
  └─ gox-tester agent (parser tests)
       ↓ Both work simultaneously
  ├─ Lexer: optimized, tested, benchmarked
  └─ Parser: 15 new edge case tests added
       ↓
SubagentStop hooks (both agents):
  • "Lexer optimization complete. Performance: +15%"
  • "Parser tests complete. Coverage: 92%"
  ↓
User: /review (both components)
  ↓
gox-reviewer checks both
  ↓
All approved, ready to merge!
```

### Example 3: Skill Auto-Activation

```
User: "Optimize the lexer's UTF-8 handling"
  ↓
lexer-dev skill auto-activates
  • Provides UTF-8 optimization patterns
  • Suggests fast path for ASCII
  • Recommends benchmarks
  ↓
Implementation follows skill guidance
  ↓
Hooks validate and test
  ↓
Complete!
```

---

## 📊 Component Relationships

```
┌──────────────┐
│  CLAUDE.md   │  ← Constitution (all components follow this)
└──────┬───────┘
       │
   ┌───┴────────────────────────────────────┐
   │                                        │
   ▼                                        ▼
┌─────────┐                           ┌─────────┐
│ SKILLS  │ ◄─────────────────────────│ AGENTS  │
└────┬────┘                           └────┬────┘
     │                                     │
     │  Skills can invoke                  │
     │  other skills                       │
     │                                     │
     │  ┌────────────────────────────┐    │
     └──►│      COMMANDS            │◄───┘
        │ (invoke agents/skills)    │
        └────────────────────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │       HOOKS           │
        │ (validate & automate) │
        └───────────────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │     SETTINGS          │
        │ (configure all above) │
        └───────────────────────┘
```

**Flow of Control:**

1. **User** invokes command or makes request
2. **Command** determines which agent/skill to use
3. **Agent** executes work, may invoke skills
4. **Skill** provides expertise during execution
5. **Hooks** validate and automate around tool use
6. **Settings** configure behavior of all components

---

## 🎯 Decision Matrix: When to Use What?

| Task Type | Use This | Why |
|-----------|----------|-----|
| **Complex feature planning** | `/plan` command → gox-planner agent | Structured analysis, no accidental edits |
| **Implementation** | `/implement` command → gox-implementer agent | Full tool access, follows plans |
| **Quick code change** | Direct interaction | No overhead for simple tasks |
| **Code review** | `/review` command → gox-reviewer agent | Systematic quality checks |
| **Test writing** | gox-tester agent | Comprehensive edge case coverage |
| **Parallel independent tasks** | Multiple agents in parallel | 2x-4x speed improvement |
| **Domain expertise** | Skills auto-activate | Just mention the domain |
| **File validation** | Hooks (automatic) | Prevent errors before they happen |
| **Auto-formatting** | Hooks (automatic) | Consistent style without thinking |
| **Auto-testing** | Hooks (automatic) | Immediate feedback on changes |

---

## 💡 Advanced Patterns

### Pattern 1: Staged Pipeline

```
gox-planner → gox-implementer → gox-tester → gox-reviewer
```

Each agent's output becomes input for the next.

### Pattern 2: Divide and Conquer

```
Large feature:
  ├─ Agent 1: Core functionality
  ├─ Agent 2: Tests
  ├─ Agent 3: Documentation
  └─ Agent 4: Optimizations

All run in parallel, then merge.
```

### Pattern 3: Iterative Refinement

```
1. /plan feature
2. /implement (first pass)
3. /review (feedback)
4. /implement (refinements)
5. /test (verify)
6. /bench (optimize)
7. /review (final check)
```

### Pattern 4: Continuous Validation

```
Every file change:
  → PreToolUse hooks validate
  → Write/Edit happens
  → PostToolUse hooks format & test
  → SubagentStop suggests next step
```

---

## 📈 Performance Metrics

### Agent Speed
- **gox-planner:** 2-5 min (complexity-dependent)
- **gox-implementer:** 5-15 min per phase
- **gox-reviewer:** 1-3 min
- **gox-tester:** 3-8 min (coverage-dependent)

### Parallel Speedup
- **2 agents:** ~1.8x faster
- **3 agents:** ~2.5x faster
- **4 agents:** ~3.2x faster

### Automation Savings
- **Hooks:** ~10-20 manual steps saved per session
- **Skills:** ~30-40% faster implementation
- **Agents:** ~50-60% more thorough

---

## 🔧 Customization Points

### Add New Skill
1. Create `.claude/skills/<name>/SKILL.md`
2. Define expertise and patterns
3. Enable in `settings.json`

### Add New Agent
1. Create `.claude/agents/<name>.md`
2. Define role and tools
3. Add to `settings.json`
4. Set tool permissions

### Add New Command
1. Create `.claude/commands/<name>.md`
2. Document behavior
3. Link to agents/skills

### Add New Hook
1. Create script in `.claude/hooks/<event>/`
2. Make executable
3. Add to `settings.json` hooks config

---

## 🎓 Learning Path

**Beginner:**
1. Use commands: `/plan`, `/implement`, `/test`
2. Let hooks automate
3. Review `.claude/QUICKSTART.md`

**Intermediate:**
4. Invoke agents explicitly
5. Understand skill auto-activation
6. Customize settings.local.json

**Advanced:**
7. Create custom skills
8. Create custom agents
9. Write custom hooks
10. Optimize workflows

---

This ecosystem map provides a bird's-eye view of how all components work together to create a powerful, efficient development environment for GoX/Conduit!
