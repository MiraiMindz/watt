# GoX/Conduit Claude Code Index

Quick navigation to all components of the development ecosystem.

## 📚 Documentation

| File | Purpose |
|------|---------|
| [README.md](README.md) | Complete ecosystem guide |
| [QUICKSTART.md](QUICKSTART.md) | 5-minute getting started guide |
| [ECOSYSTEM_MAP.md](ECOSYSTEM_MAP.md) | Visual system architecture |
| [INDEX.md](INDEX.md) | This file - navigation hub |

## 📋 Project Documents

| File | Purpose |
|------|---------|
| [../CLAUDE.md](../CLAUDE.md) | Project constitution and standards |
| [../GOX_COMPLETE_BLUEPRINT.md](../GOX_COMPLETE_BLUEPRINT.md) | Complete rebuild guide |
| [../IMPLEMENTATION_PLAN.md](../IMPLEMENTATION_PLAN.md) | Implementation roadmap |
| [../QUICK_REFERENCE.md](../QUICK_REFERENCE.md) | Fast feature reference |
| [../README.md](../README.md) | Project overview |

## 🎯 Skills

| Skill | Expertise Area | Tools |
|-------|---------------|-------|
| [lexer-dev](skills/lexer-dev/SKILL.md) | Multi-mode tokenization | Read, Write, Edit, Bash, Grep, Glob |
| [parser-dev](skills/parser-dev/SKILL.md) | AST generation & parsing | Read, Write, Edit, Bash, Grep, Glob |
| [transpiler-dev](skills/transpiler-dev/SKILL.md) | Code generation | Read, Write, Edit, Bash, Grep, Glob |

### When Skills Activate
- Automatically when mentioned in conversation
- Explicitly with: `Use <skill-name> skill to <task>`

## 🤖 Agents

| Agent | Role | Tools | Read-Only? |
|-------|------|-------|------------|
| [gox-planner](agents/gox-planner.md) | Creates implementation plans | Read, Grep, Glob | ✅ Yes |
| [gox-implementer](agents/gox-implementer.md) | Implements features | Read, Write, Edit, Bash, Grep, Glob | ❌ No |
| [gox-reviewer](agents/gox-reviewer.md) | Reviews code quality | Read, Bash, Grep, Glob | ✅ Yes |
| [gox-tester](agents/gox-tester.md) | Writes comprehensive tests | Read, Write, Edit, Bash, Grep, Glob | ❌ No |

### Invoking Agents
```
Use <agent-name> agent to <task>

# Parallel execution
Run <agent-1> on <task-1> and <agent-2> on <task-2> in parallel
```

## ⚡ Commands

| Command | Purpose | Invokes |
|---------|---------|---------|
| [/plan](commands/plan.md) | Create implementation plan | gox-planner |
| [/implement](commands/implement.md) | Implement from plan | gox-implementer |
| [/test](commands/test.md) | Run test suite | - |
| [/review](commands/review.md) | Code review | gox-reviewer |
| [/bench](commands/bench.md) | Run benchmarks | - |

### Usage
```
/plan <feature-name>
/implement plans/<feature-name>-plan.md
/test [package]
/review [files]
/bench [package]
```

## 🪝 Hooks

### PreToolUse

| Hook | Purpose | Blocks? |
|------|---------|---------|
| [validate-write.sh](hooks/pre-tool-use/validate-write.sh) | Prevent writes to protected files | ✅ Yes |

### PostToolUse

| Hook | Purpose | Auto-Run? |
|------|---------|-----------|
| [auto-format.sh](hooks/post-tool-use/auto-format.sh) | Format Go files with gofmt | ✅ Yes |
| [auto-test.sh](hooks/post-tool-use/auto-test.sh) | Run tests for modified packages | ✅ Yes |

### SubagentStop

| Hook | Purpose | Type |
|------|---------|------|
| suggest-next-step | Suggests next logical step | LLM-based |

## ⚙️ Configuration

| File | Purpose | Committed? |
|------|---------|------------|
| [settings.json](settings.json) | Main configuration | ✅ Yes |
| settings.local.json | Developer overrides | ❌ No |

### Key Settings Sections
- `hooks` - Hook configurations
- `skills` - Skill enablement
- `agents` - Agent definitions
- `toolPermissions` - Tool access per agent
- `performance` - Performance targets
- `git` - Git workflow rules

## 🎯 Quick Reference

### Common Workflows

**Plan → Implement → Test → Review:**
```
/plan feature-name
/implement plans/feature-name-plan.md
/test
/review
```

**Parallel Development:**
```
Run gox-implementer on feature-a and gox-implementer on feature-b in parallel
```

**Bug Fix:**
```
/plan bug-fix-issue-123
/implement plans/bug-fix-issue-123-plan.md
/test pkg/affected
/review
```

**Performance Optimization:**
```
/bench pkg/slow-component
Use lexer-dev skill to optimize
/bench pkg/slow-component
```

## 📊 Performance Targets

From `settings.json`:

| Component | Target | Current |
|-----------|--------|---------|
| Lexer | 1000 lines/ms | ✅ Implemented |
| Parser | 500 lines/ms | ✅ Implemented |
| Analyzer | 200 components/s | ✅ Implemented |
| Transpiler | 100 components/s | 🚧 In Progress |

## 🎓 Learning Path

1. **Start Here:** [QUICKSTART.md](QUICKSTART.md)
2. **Understand System:** [ECOSYSTEM_MAP.md](ECOSYSTEM_MAP.md)
3. **Learn Standards:** [../CLAUDE.md](../CLAUDE.md)
4. **Deep Dive:** [README.md](README.md)
5. **Use Commands:** Try `/plan`, `/implement`, `/test`
6. **Invoke Agents:** Practice agent workflows
7. **Create Custom:** Add your own skills/agents/hooks

## 🔗 External Links

- **GoX Documentation:** `../docs/`
- **Examples:** `../examples/`
- **Plans:** `../plans/` (created by gox-planner)

## 🆘 Getting Help

1. **Quick Start:** Read [QUICKSTART.md](QUICKSTART.md)
2. **Full Guide:** Read [README.md](README.md)
3. **Troubleshooting:** Check QUICKSTART troubleshooting section
4. **Standards:** Consult [../CLAUDE.md](../CLAUDE.md)
5. **Architecture:** Review [../GOX_COMPLETE_BLUEPRINT.md](../GOX_COMPLETE_BLUEPRINT.md)

## 📝 File Structure

```
.claude/
├── INDEX.md                     ← You are here
├── README.md                    ← Full guide
├── QUICKSTART.md                ← Getting started
├── ECOSYSTEM_MAP.md             ← Visual architecture
├── settings.json                ← Configuration
├── skills/                      ← Expertise modules
│   ├── lexer-dev/
│   ├── parser-dev/
│   └── transpiler-dev/
├── agents/                      ← Specialized workers
│   ├── gox-planner.md
│   ├── gox-implementer.md
│   ├── gox-reviewer.md
│   └── gox-tester.md
├── commands/                    ← Quick actions
│   ├── plan.md
│   ├── implement.md
│   ├── test.md
│   ├── review.md
│   └── bench.md
└── hooks/                       ← Automation
    ├── pre-tool-use/
    │   └── validate-write.sh
    └── post-tool-use/
        ├── auto-format.sh
        └── auto-test.sh
```

---

**Navigate wisely, code efficiently!** 🚀

Last Updated: 2025-01-15
Version: 1.0
