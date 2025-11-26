# Electron Claude Code Workspace

This directory contains the complete Claude Code workspace configuration for the Electron project - a comprehensive system of agents, skills, hooks, commands, and rules designed to optimize development of high-performance Go code.

## 📁 Directory Structure

```
.claude/
├── README.md                          # This file
├── settings.json                      # Hooks and preferences
├── agents/                            # Specialized subagents
│   ├── performance-architect.md       # Design review & planning
│   ├── unsafe-code-reviewer.md        # Unsafe code validation
│   ├── benchmark-engineer.md          # Benchmark creation
│   └── integration-tester.md          # Integration testing
├── skills/                            # Reusable expertise packages
│   ├── go-performance/
│   │   ├── SKILL.md                   # Performance optimization expertise
│   │   └── workflows/
│   │       ├── bench.md               # /bench command
│   │       └── profile.md             # /profile command
│   ├── unsafe-go/
│   │   ├── SKILL.md                   # Unsafe code expertise
│   │   └── workflows/
│   │       └── unsafe-review.md       # /unsafe-review command
│   ├── simd-optimization/
│   │   ├── SKILL.md                   # SIMD expertise
│   │   └── workflows/
│   │       └── simd-test.md           # /simd-test command
│   ├── concurrency-patterns/
│   │   └── SKILL.md                   # Concurrency expertise
│   └── go-testing/
│       └── SKILL.md                   # Testing expertise
└── scripts/
    ├── phase-status.md                # Phase progress tracking
    └── run-all-tests.sh               # Comprehensive test runner
```

## 🎯 Quick Start

### Using Agents

Agents are specialized subagents that work autonomously on specific tasks:

```bash
# Review a design for performance implications
"Use the performance-architect agent to review my arena allocator design"

# Validate unsafe code
"Run the unsafe-code-reviewer agent on the unsafe string conversions"

# Create comprehensive benchmarks
"Use the benchmark-engineer agent to create benchmarks for the pool package"

# Test component integration
"Run the integration-tester agent to test arena and pool integration"
```

### Using Skills

Skills are automatically invoked by Claude when relevant, providing specialized expertise:

- **go-performance** - Zero-allocation patterns, benchmarking, profiling
- **unsafe-go** - Safe unsafe patterns, pointer arithmetic, type conversions
- **simd-optimization** - Vectorized operations, CPU feature detection
- **concurrency-patterns** - Lock-free structures, memory ordering
- **go-testing** - Unit tests, benchmarks, fuzz tests

Skills activate automatically based on your request. Just ask naturally:
- "How do I write a zero-allocation string builder?" → go-performance skill activates
- "I need to implement a lock-free queue" → concurrency-patterns skill activates
- "Help me optimize this with SIMD" → simd-optimization skill activates

### Using Commands

Commands are slash commands for quick actions:

```bash
/bench              # Run benchmarks and compare to targets
/profile            # Profile CPU and memory usage
/unsafe-review      # Review all unsafe code
/simd-test          # Test SIMD implementations
```

### Using Scripts

Helper scripts for common workflows:

```bash
# Check current implementation phase status
# (Use the phase-status.md script content as a prompt)

# Run comprehensive test suite
chmod +x .claude/scripts/run-all-tests.sh
./.claude/scripts/run-all-tests.sh
```

## 🏗️ Architecture Overview

### The Rules (CLAUDE.md)

The root `CLAUDE.md` file is the "constitution" - it defines:
- Project philosophy and principles
- Code standards and conventions
- Performance targets
- Testing requirements
- Architecture decisions

### The Hooks (settings.json)

Hooks automate validation and enforcement:

**PreToolUse Hooks:**
- Block writes to vendor directories
- Validate Go file paths
- Run pre-commit checks before git commits

**PostToolUse Hooks:**
- Auto-format Go code with gofmt
- Run go vet on modified files

**SubagentStop Hooks:**
- Suggest next actions after agent completion

### The Agents

Specialized subagents with specific roles:

#### performance-architect
- **Role:** Design review and performance planning
- **Tools:** Read, Grep, Glob, Bash (read-only)
- **Output:** Detailed performance recommendations
- **Use when:** Designing new components, reviewing architectures

#### unsafe-code-reviewer
- **Role:** Validate unsafe code correctness and safety
- **Tools:** Read, Grep, Glob, Bash (read-only)
- **Output:** Safety analysis and recommendations
- **Use when:** Writing or reviewing unsafe code

#### benchmark-engineer
- **Role:** Create comprehensive benchmarks
- **Tools:** Read, Write, Edit, Bash
- **Output:** Benchmark code and performance reports
- **Use when:** Validating performance targets

#### integration-tester
- **Role:** Test component interactions
- **Tools:** Read, Write, Edit, Bash
- **Output:** Integration tests and test reports
- **Use when:** Testing system-level behavior

### The Skills

Reusable expertise packages that Claude invokes automatically:

#### go-performance
- Zero-allocation patterns
- Benchmark writing
- Performance profiling
- Escape analysis
- Memory management strategies

#### unsafe-go
- Safe unsafe patterns
- Pointer arithmetic
- Type conversions
- Memory layout optimization
- Comprehensive safety documentation

#### simd-optimization
- CPU feature detection
- Vectorized operations (AVX2, AVX-512, NEON)
- Batch processing
- Fallback patterns
- Platform-specific optimization

#### concurrency-patterns
- Lock-free data structures
- Memory ordering
- Synchronization primitives
- Work scheduling
- Race detection

#### go-testing
- Table-driven tests
- Benchmarks
- Fuzz tests
- Memory leak detection
- Coverage analysis

## 🎨 Workflow Patterns

### Pattern 1: Design → Review → Implement

```
1. You: "I need to implement an arena allocator"
2. Claude: Suggests using performance-architect agent
3. Agent: Reviews design, provides detailed plan
4. You: Review and approve plan
5. Claude: Implements based on plan (with go-performance skill)
6. You: "/bench" to validate performance
```

### Pattern 2: Implement → Test → Validate

```
1. Claude: Implements feature
2. You: "Use benchmark-engineer to create benchmarks"
3. Agent: Creates comprehensive benchmarks
4. You: "/bench" to check against targets
5. Claude: Optimizes if needed (using skills)
6. You: "Use integration-tester to test with other components"
```

### Pattern 3: Safety Review Pipeline

```
1. Claude: Implements unsafe code
2. You: "/unsafe-review" command
3. Command: Shows all unsafe usage
4. You: "Use unsafe-code-reviewer agent on [specific file]"
5. Agent: Detailed safety analysis
6. You: Review and address issues
7. Claude: Updates based on feedback
```

## 📊 Performance Targets

The workspace is configured to validate these targets:

| Component | Metric | Target | Validation |
|-----------|--------|--------|------------|
| Arena Allocation | Latency | <10ns/op | /bench command |
| Arena Allocation | Allocations | 0 allocs/op | /bench command |
| Pool Get/Put | Latency | <50ns/op | /bench command |
| Pool Get/Put | Allocations | 0 allocs/op | /bench command |
| SIMD String Compare | Throughput | >10GB/s | /simd-test command |
| Lock-Free Queue | Throughput | >10M ops/sec | /bench command |

## 🔧 Customization

### Adding New Commands

Create a markdown file in a skill's `workflows/` directory:

```markdown
# .claude/skills/my-skill/workflows/my-command.md

# Command Description

You are tasked with [description].

## Your Task
[Instructions...]

## Output Format
[Expected output...]
```

Use with: `/my-command`

### Adding New Skills

1. Create directory: `.claude/skills/my-skill/`
2. Create `SKILL.md` with frontmatter:

```markdown
---
name: my-skill
description: [When to use this skill]
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# My Skill

[Skill content...]
```

### Adding New Agents

Create a markdown file in `.claude/agents/`:

```markdown
---
name: my-agent
description: [What this agent does]
tools: Read, Write, Edit, Bash
---

# My Agent

You are [description].

## Your Responsibilities
[List responsibilities...]
```

## 🎓 Learning Resources

### For Go Performance
- Read `go-performance/SKILL.md` for patterns
- Study benchmark examples in `benchmark-engineer.md`
- Check performance targets in `CLAUDE.md`

### For Unsafe Code
- Read `unsafe-go/SKILL.md` for safe patterns
- Review safety checklist in `unsafe-code-reviewer.md`
- Study documentation template

### For SIMD
- Read `simd-optimization/SKILL.md` for techniques
- Check CPU feature detection patterns
- Study fallback strategies

### For Concurrency
- Read `concurrency-patterns/SKILL.md` for patterns
- Study lock-free examples
- Review memory ordering guide

## 🚀 Next Steps

1. **Start Development**
   - Review current phase: Use phase-status script
   - Start implementing: Follow IMPLEMENTATION_PLAN.md
   - Get design review: Use performance-architect agent

2. **Ensure Quality**
   - Write tests: Use go-testing skill
   - Create benchmarks: Use benchmark-engineer agent
   - Review unsafe code: Use unsafe-code-reviewer agent

3. **Validate Performance**
   - Run benchmarks: `/bench` command
   - Profile code: `/profile` command
   - Test SIMD: `/simd-test` command

4. **Test Integration**
   - Use integration-tester agent
   - Run comprehensive tests: `run-all-tests.sh`
   - Check for leaks

## 🎯 Success Metrics

The workspace helps you achieve:

- ✅ **Zero-allocation hot paths** - Validated by benchmarks
- ✅ **Performance targets met** - Tracked by /bench command
- ✅ **Safe unsafe code** - Reviewed by unsafe-code-reviewer
- ✅ **Comprehensive tests** - Guided by go-testing skill
- ✅ **Correct integration** - Verified by integration-tester
- ✅ **SIMD optimization** - Tested by /simd-test

## 🆘 Troubleshooting

### Agents not working?
- Check agent markdown file exists in `.claude/agents/`
- Verify frontmatter is correct (name, description, tools)
- Make sure you're invoking with correct syntax

### Skills not activating?
- Skills activate automatically based on context
- Check `description` in skill frontmatter matches your request
- Try being more specific in your request

### Commands not found?
- Commands are in `workflows/` subdirectories of skills
- Check the markdown file exists
- Command name matches the filename (without .md)

### Hooks not running?
- Check `.claude/settings.json` syntax
- Verify hook matchers are correct
- Test hooks with simple operations first

## 📝 Examples

### Example 1: Implementing a New Component

```
You: "I want to implement the arena allocator from Phase 1"

Claude: "I'll help you implement the arena allocator. First, let me use the
performance-architect agent to review the design and create a detailed plan."

[Agent runs, provides detailed design review]

You: "Looks good, proceed with implementation"

Claude: [Implements using go-performance skill]

You: "Use benchmark-engineer to create benchmarks"

[Agent creates comprehensive benchmarks]

You: "/bench"

[Benchmarks run, show results vs targets]

You: "/unsafe-review"

[Reviews unsafe pointer arithmetic in arena]

You: "Use unsafe-code-reviewer on arena/arena.go"

[Detailed safety analysis]
```

### Example 2: Optimizing Existing Code

```
You: "/profile"

[Profiling shows allocation hot spot in string building]

You: "How can I eliminate these allocations?"

Claude: [Uses go-performance skill to suggest zero-allocation pattern]

You: "Implement that optimization"

Claude: [Implements with unsafe string conversion]

You: "Use unsafe-code-reviewer to validate"

[Agent reviews, confirms safety]

You: "/bench"

[Benchmarks show 10x improvement, 0 allocs/op]
```

## 🎉 You're Ready!

This workspace gives you a complete development environment optimized for high-performance Go development. The agents, skills, hooks, and commands work together to ensure correctness, performance, and safety.

Start by checking your current phase with the phase-status script, then use the appropriate agents and skills as you implement components.

Happy coding! ⚡
