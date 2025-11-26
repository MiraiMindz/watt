# GoX/Conduit Claude Code Quick Start

Get up and running with the GoX development environment in 5 minutes!

## 🎯 Your First Task

### Scenario: Implement CSS Scoping Feature

Let's walk through the complete workflow using all the Claude Code features.

---

## Step 1: Plan the Feature

```
/plan css-scoping
```

**What happens:**
- `gox-planner` agent activates
- Reads existing code and docs
- Creates `plans/css-scoping-plan.md`
- Breaks work into phases
- Identifies dependencies

**Output:**
```markdown
# CSS Scoping Implementation Plan

## Overview
Implement CSS scoping to prevent style leakage between components...

## Implementation Phases

### Phase 1: Scope Generator
- Generate unique component IDs
- Add data attributes to elements
- Transform CSS selectors

### Phase 2: Integration
- Update parser to track component IDs
- Modify transpiler to inject scopes
- Update runtime to apply attributes

### Phase 3: Testing
- Unit tests for scope generation
- Integration tests for full pipeline
- Example components

## Success Criteria
- Styles scoped to component
- No global leakage
- Performance acceptable
```

---

## Step 2: Implement the Feature

```
/implement plans/css-scoping-plan.md
```

**What happens:**
- `gox-implementer` agent activates
- Reads the plan
- Studies existing code patterns
- Implements incrementally
- Writes tests as it goes
- Follows CLAUDE.md standards

**Hooks Activate:**
- ✅ `validate-write` - Checks protected files
- ✅ `auto-format` - Formats Go code
- ✅ `auto-test` - Runs tests after changes

**Output:**
```markdown
## Implementation Complete: CSS Scoping

**Files Created:**
- pkg/optimizer/scope.go
- pkg/optimizer/scope_test.go

**Files Modified:**
- pkg/transpiler/ssr/transpiler.go
- pkg/transpiler/csr/transpiler.go

**Tests:** 12 passing
**Coverage:** 85%

**Ready for review**
```

---

## Step 3: Review the Code

```
/review
```

**What happens:**
- `gox-reviewer` agent activates
- Checks code quality
- Verifies tests
- Checks performance
- Ensures standards compliance

**Output:**
```markdown
## Code Review: CSS Scoping

**Overall:** ✅ APPROVED with minor suggestions

### Strengths
- Clean implementation
- Comprehensive tests
- Good documentation

### Suggestions
**Minor:**
- Consider caching component IDs (line 42)
- Add example to docs

**Performance:**
- Benchmarks pass ✅
- No allocations in hot path ✅

**Recommendation:** APPROVE
```

---

## Step 4: Benchmark

```
/bench pkg/optimizer
```

**What happens:**
- Runs benchmarks
- Compares against targets
- Reports performance

**Output:**
```markdown
## Benchmark Results

### Optimizer
- Scope Generation: 5000 ops/sec ✅
- No allocations ✅
- Memory: 256 B/op ✅

**Status:** All targets met
```

---

## Step 5: Full Test Suite

```
/test
```

**What happens:**
- Runs all tests
- Reports coverage
- Identifies any failures

**Output:**
```markdown
## Test Results

**Status:** ✅ ALL PASSING
**Coverage:** 78% ✅ (target: 70%)
**Tests:** 142 passed, 0 failed

**Performance:**
- All benchmarks passing ✅

**Ready to merge**
```

---

## Bonus: Parallel Workflow

For maximum efficiency, run independent tasks in parallel:

```
Run gox-implementer on CSS scoping Phase 1 and
gox-tester on parser edge cases in parallel
```

Both agents work simultaneously, doubling productivity!

---

## 🎓 Learning the Features

### Skills Auto-Activate

Just mention the domain:

```
"Optimize the lexer tokenization"
```

→ **lexer-dev skill** automatically activates

### Explicit Skill Invocation

```
Use the parser-dev skill to fix JSX parsing bug
```

### Agent Workflows

**Pattern:** Planner → Implementer → Reviewer → Tester

```
1. /plan feature-x
2. /implement plans/feature-x-plan.md
3. /review
4. Use gox-tester to add edge cases
```

---

## 🔧 Common Tasks

### Fix a Bug

```
1. Read the bug report
2. /plan bug-fix-<name>
3. /implement plans/bug-fix-<name>-plan.md
4. /test pkg/affected
5. /review
```

### Optimize Performance

```
1. /bench pkg/slow-component
2. Use lexer-dev skill to optimize hot path
3. /bench pkg/slow-component (verify improvement)
4. /review
```

### Add Tests

```
Use gox-tester agent to add edge case tests for pkg/parser
```

### Refactor Code

```
1. /plan refactor-<component>
2. /implement plans/refactor-<component>-plan.md
3. /test (ensure no regressions)
4. /bench (ensure no performance loss)
5. /review
```

---

## 🎯 Hooks in Action

### Protected File Warning

Try:
```
Write to .env file
```

Hook blocks:
```
❌ ERROR: Cannot write to protected file (contains secrets)
```

### Auto-Format

After writing Go code:
```
🔧 Auto-formatting Go file: pkg/lexer/lexer.go
✅ Formatted successfully
```

### Auto-Test

After writing test file:
```
🧪 Running tests for package: pkg/parser
✅ Tests passed
```

---

## 💡 Pro Tips

**Tip 1:** Chain commands for full workflow
```
/plan feature && /implement plans/feature-plan.md && /test && /review
```

**Tip 2:** Use agents for complex analysis
```
Use gox-planner to analyze performance bottlenecks across the codebase
```

**Tip 3:** Parallel for speed
```
Run gox-implementer on feature-a and gox-implementer on feature-b in parallel
```

**Tip 4:** Skills for expertise
```
Use transpiler-dev skill to optimize VNode generation
```

**Tip 5:** Review before commit
```
/review && /bench && git add . && git commit
```

---

## 🚨 Troubleshooting

### Hook Not Running?

Check `.claude/settings.json` and verify:
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [...]
      }
    ]
  }
}
```

### Skill Not Activating?

Try explicit invocation:
```
Use the <skill-name> skill to <task>
```

### Agent Not Working?

Verify tool permissions in settings.json:
```json
{
  "toolPermissions": {
    "gox-implementer": ["Read", "Write", "Edit", "Bash", ...]
  }
}
```

### Tests Failing?

```
/test pkg/specific-package

# Or with verbose output
Run: go test -v ./pkg/specific-package
```

---

## 📚 Next Steps

1. **Read the docs:**
   - `.claude/README.md` - Full ecosystem guide
   - `CLAUDE.md` - Project standards
   - `GOX_COMPLETE_BLUEPRINT.md` - Architecture

2. **Try the examples:**
   - Look at `plans/` for example plans
   - Study agents in `.claude/agents/`
   - Review skills in `.claude/skills/`

3. **Customize:**
   - Add your own commands
   - Create project-specific skills
   - Customize hooks
   - Adjust settings.local.json

---

## 🎉 You're Ready!

You now know how to:
- ✅ Plan features with `/plan`
- ✅ Implement with `/implement`
- ✅ Test with `/test`
- ✅ Review with `/review`
- ✅ Benchmark with `/bench`
- ✅ Use agents for specialized work
- ✅ Leverage skills for expertise
- ✅ Benefit from automated hooks

**Start building GoX features with confidence!** 🚀

---

## Quick Reference Card

```
COMMANDS:
/plan <feature>          → Create implementation plan
/implement <plan-file>   → Implement feature
/test [package]          → Run tests
/review [files]          → Code review
/bench [package]         → Run benchmarks

AGENTS:
gox-planner              → Plans (Read, Grep, Glob)
gox-implementer          → Codes (Full tools)
gox-reviewer             → Reviews (Read, Bash, Grep, Glob)
gox-tester               → Tests (Full tools)

SKILLS:
lexer-dev                → Lexer expertise
parser-dev               → Parser expertise
transpiler-dev           → Code generation expertise

HOOKS:
validate-write           → Protect files (PreToolUse)
auto-format              → Format code (PostToolUse)
auto-test                → Run tests (PostToolUse)
suggest-next-step        → Next action (SubagentStop)
```

---

**Happy Coding!** 🚀
