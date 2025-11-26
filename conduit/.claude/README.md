# GoX/Conduit Claude Code Ecosystem

Welcome to the Claude Code development environment for GoX/Conduit!

## 📋 Quick Start

### Common Workflows

**Plan a New Feature:**
```
/plan <feature-name>
```
Creates a detailed implementation plan using the `gox-planner` agent.

**Implement a Feature:**
```
/implement plans/<feature-name>-plan.md
```
Uses `gox-implementer` agent to code the feature following the plan.

**Run Tests:**
```
/test [package]
```
Runs tests and reports coverage.

**Code Review:**
```
/review [files]
```
Uses `gox-reviewer` agent to review code quality.

**Benchmark Performance:**
```
/bench [package]
```
Runs benchmarks and compares against targets.

---

## 🏗️ Project Structure

```
.claude/
├── README.md                    # This file
├── settings.json                # Configuration
├── skills/                      # Reusable expertise
│   ├── lexer-dev/
│   │   └── SKILL.md
│   ├── parser-dev/
│   │   └── SKILL.md
│   └── transpiler-dev/
│       └── SKILL.md
├── agents/                      # Specialized contractors
│   ├── gox-planner.md          # Plans features
│   ├── gox-implementer.md      # Writes code
│   ├── gox-reviewer.md         # Reviews code
│   └── gox-tester.md           # Tests code
├── commands/                    # Quick actions
│   ├── plan.md
│   ├── implement.md
│   ├── test.md
│   ├── review.md
│   └── bench.md
└── hooks/                       # Automation
    ├── pre-tool-use/
    │   └── validate-write.sh
    └── post-tool-use/
        ├── auto-format.sh
        └── auto-test.sh
```

---

## 🎯 Skills

Skills are like recipe cards that teach Claude how to approach specific types of tasks.

### Available Skills

**lexer-dev** - Multi-mode lexer expertise
- Tokenization strategies
- Mode switching (Go/JSX/CSS)
- Performance optimization
- Testing approaches

**parser-dev** - Parser construction expertise
- AST generation
- Component/hook/JSX/CSS parsing
- Error recovery
- Testing strategies

**transpiler-dev** - Code generation expertise
- SSR transpiler (Go structs + Render())
- CSR transpiler (VNode generation)
- Expression interpolation
- Event handler wrapping

### When Skills Activate

Skills activate automatically when:
- You mention their domain (e.g., "lexer", "parser")
- The task matches their description
- Claude determines they're relevant

### Manual Activation

You can also explicitly invoke a skill:
```
Use the lexer-dev skill to optimize tokenization
```

---

## 🤖 Agents

Agents are specialized contractors for specific jobs.

### gox-planner
**Role:** Creates implementation plans
**Tools:** Read, Grep, Glob (no write access)
**Use When:** Starting a new feature or component

**Example:**
```
Use gox-planner agent to create a plan for CSR transpiler
```

### gox-implementer
**Role:** Implements features following plans
**Tools:** Full access (Read, Write, Edit, Bash, Grep, Glob)
**Use When:** Ready to code a planned feature

**Example:**
```
Use gox-implementer agent to implement plans/csr-transpiler-plan.md
```

### gox-reviewer
**Role:** Reviews code for quality and standards
**Tools:** Read, Bash, Grep, Glob (no write access)
**Use When:** Code is complete and needs review

**Example:**
```
Use gox-reviewer agent to review the CSR transpiler implementation
```

### gox-tester
**Role:** Writes comprehensive tests
**Tools:** Read, Write, Edit, Bash, Grep, Glob
**Use When:** Need additional test coverage or edge case testing

**Example:**
```
Use gox-tester agent to add tests for pkg/transpiler/csr/
```

### Running Agents in Parallel

For maximum efficiency, run agents concurrently:
```
Run gox-implementer on feature A and gox-tester on feature B in parallel
```

---

## ⚡ Commands

Quick shortcuts for common tasks.

### /plan <feature>
Creates implementation plan using gox-planner agent.

**Output:** `plans/<feature>-plan.md`

### /implement <plan-file>
Implements feature using gox-implementer agent.

**Output:** New code + tests

### /test [package]
Runs tests, shows coverage and performance.

**Options:**
- `/test` - All tests
- `/test pkg/lexer` - Specific package
- `/test -coverage` - With coverage report
- `/test -bench` - Benchmarks only

### /review [files]
Code review using gox-reviewer agent.

**Output:** Review report with issues and recommendations

### /bench [package]
Runs benchmarks, compares against targets.

**Output:** Performance report with optimization suggestions

---

## 🪝 Hooks

Automated workflows at specific lifecycle points.

### Pre-Tool-Use Hooks

**validate-write.sh**
- Runs before Write/Edit
- Blocks writing to protected files (.env, credentials.json)
- Warns about writing to generated files

### Post-Tool-Use Hooks

**auto-format.sh**
- Runs after Write/Edit on .go files
- Automatically formats with `gofmt`
- Ensures consistent code style

**auto-test.sh**
- Runs after Write/Edit on *_test.go files
- Automatically runs tests for modified package
- Immediate feedback on test failures

### SubagentStop Hooks

**suggest-next-step**
- Runs when any subagent completes
- Suggests logical next step in workflow
- Helps maintain momentum

---

## ⚙️ Configuration

### settings.json

Main configuration file with:

**Hooks Configuration:**
```json
{
  "hooks": {
    "PreToolUse": [...],
    "PostToolUse": [...],
    "SubagentStop": [...]
  }
}
```

**Skills Configuration:**
```json
{
  "skills": {
    "enabled": ["lexer-dev", "parser-dev", "transpiler-dev"],
    "autoDiscovery": true
  }
}
```

**Agent Configuration:**
```json
{
  "agents": {
    "defaultModel": "sonnet",
    "available": ["gox-planner", "gox-implementer", ...]
  }
}
```

**Tool Permissions:**
```json
{
  "toolPermissions": {
    "gox-planner": ["Read", "Grep", "Glob"],
    "gox-implementer": ["Read", "Write", "Edit", "Bash", ...]
  }
}
```

### settings.local.json

Developer-specific overrides (not committed to git):
```json
{
  "development": {
    "autoFormat": false,
    "autoTest": false
  }
}
```

---

## 🎨 Development Patterns

### Pattern 1: Planned Development

1. **Plan:**
   ```
   /plan new-feature
   ```

2. **Review Plan:**
   - Read `plans/new-feature-plan.md`
   - Discuss and refine

3. **Implement:**
   ```
   /implement plans/new-feature-plan.md
   ```

4. **Review:**
   ```
   /review
   ```

5. **Test:**
   ```
   /test
   ```

### Pattern 2: Parallel Execution

For independent tasks:
```
Run gox-implementer on lexer optimization and
gox-tester on parser edge cases in parallel
```

### Pattern 3: Staged Pipeline

Planner → Implementer → Tester → Reviewer

```
1. /plan feature-x
2. /implement plans/feature-x-plan.md
3. Use gox-tester to add edge case tests
4. /review
```

### Pattern 4: Incremental Development

Build feature incrementally:

```
Phase 1: /implement foundation (types, interfaces)
Phase 2: /implement core-logic
Phase 3: /implement optimizations
```

---

## 📊 Performance Targets

From CLAUDE.md and monitored by benchmarks:

- **Lexer:** ~1000 lines/ms
- **Parser:** ~500 lines/ms
- **Analyzer:** ~200 components/s
- **Transpiler:** ~100 components/s
- **Coverage:** Minimum 70%, target 85%

Hooks automatically check these during development.

---

## 🚀 Advanced Usage

### Creating New Skills

1. Create directory: `.claude/skills/<skill-name>/`
2. Create `SKILL.md` with frontmatter:
   ```markdown
   ---
   name: skill-name
   description: When to use this skill
   allowed-tools: Read, Write, Edit
   ---

   # Skill Content
   ...
   ```

### Creating New Agents

1. Create file: `.claude/agents/<agent-name>.md`
2. Define role, tools, and process
3. Add to settings.json
4. Set tool permissions

### Creating New Commands

1. Create file: `.claude/commands/<command-name>.md`
2. Document usage and behavior
3. Optionally reference agents/skills

### Creating New Hooks

1. Create script: `.claude/hooks/<event>/<hook-name>.sh`
2. Make executable: `chmod +x <hook-name>.sh`
3. Add to settings.json hooks configuration
4. Use environment variables:
   - `$TOOL_NAME` - Name of tool being used
   - `$TOOL_INPUT` - Input to the tool
   - `$TOOL_OUTPUT` - Output from tool (PostToolUse only)

---

## 🔍 Debugging

### Verbose Mode

Enable in settings.local.json:
```json
{
  "debug": {
    "verbose": true,
    "logHooks": true,
    "logAgents": true
  }
}
```

### Check Hook Execution

Hooks output to console when running.

### Agent Output

Agents return detailed reports of their work.

---

## 📚 Resources

### Core Documents
- **CLAUDE.md** - Project constitution and coding standards
- **GOX_COMPLETE_BLUEPRINT.md** - Complete rebuild guide
- **IMPLEMENTATION_PLAN.md** - Implementation plan
- **QUICK_REFERENCE.md** - Fast reference

### Examples
- `examples/counter-ssr/` - SSR example
- `examples/todo-app/` - Full application

### Plans
- `plans/` - Implementation plans for features

---

## 🤝 Contributing to Claude Ecosystem

When adding new components:

1. **Document Thoroughly** - Clear descriptions and examples
2. **Test the Workflow** - Verify skills/agents/commands work
3. **Update README** - Keep this file current
4. **Share Patterns** - Document effective workflows

---

## 💡 Tips & Tricks

**Tip 1: Chain Commands**
```
/plan feature && /implement plans/feature-plan.md && /test && /review
```

**Tip 2: Specific Agent for Task**
```
Use gox-planner to analyze performance bottlenecks in the lexer
```

**Tip 3: Parallel for Speed**
```
Run gox-implementer on feature-a and feature-b in parallel
```

**Tip 4: Review Before Commit**
```
/review && git add . && git commit
```

**Tip 5: Benchmark Frequently**
```
/bench pkg/lexer
```

---

## 🎓 Learning Resources

New to Claude Code features? Check:

- Claude Code Documentation
- GoX/Conduit docs in root
- Example workflows in this README

---

**Remember:** This ecosystem is designed to accelerate GoX development. Use agents for complex tasks, skills for expertise, commands for quick actions, and hooks for automation. Together, they create a powerful, efficient development workflow.

Happy coding! 🚀
