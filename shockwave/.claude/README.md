# Claude Code Ecosystem for Shockwave

This directory contains the complete Claude Code ecosystem for the Shockwave HTTP library, optimized for Claude Sonnet 4.5.

## 📚 Essential Reading

Start here to understand the ecosystem:

1. **[PROMPTING_GUIDE.md](PROMPTING_GUIDE.md)** - Best practices for working with Claude 4.5
2. **[AGENT_PATTERNS.md](AGENT_PATTERNS.md)** - How to use and create effective agents
3. **[SKILL_PATTERNS.md](SKILL_PATTERNS.md)** - Writing skills that activate reliably
4. **[../CLAUDE.md](../CLAUDE.md)** - Project constitution with Claude 4.5 interaction patterns

## Directory Structure

```
.claude/
├── agents/                 # Specialized autonomous agents
│   ├── performance-auditor.md
│   ├── protocol-validator.md
│   └── benchmark-runner.md
├── skills/                 # Automated expertise
│   ├── go-performance-optimization/
│   │   ├── SKILL.md
│   │   ├── context/
│   │   └── workflows/
│   ├── http-protocol-testing/
│   ├── memory-profiling/
│   └── benchmark-analysis/
├── commands/               # Quick action shortcuts
│   ├── bench.md
│   ├── profile-mem.md
│   ├── profile-cpu.md
│   ├── check-allocs.md
│   ├── compare-nethttp.md
│   └── test-protocol.md
├── hooks/                  # Automation scripts
│   ├── check_antipatterns.sh
│   ├── pre_commit_bench.sh
│   └── load_perf_context.sh
├── settings.json           # Configuration and hook definitions
└── README.md               # This file
```

## How to Use

### Commands (User-Invoked)

Type `/command-name` to run a command:

- `/bench` - Run comprehensive benchmarks
- `/profile-mem` - Memory profiling analysis
- `/profile-cpu` - CPU profiling analysis
- `/check-allocs` - Verify zero allocations
- `/compare-nethttp` - Compare with standard library
- `/test-protocol` - RFC compliance testing

### Skills (Auto-Invoked)

Claude automatically uses skills when relevant:

- When you ask about **performance optimization** → `go-performance-optimization` skill
- When you ask about **protocol compliance** → `http-protocol-testing` skill
- When you ask about **memory issues** → `memory-profiling` skill
- When you ask about **benchmarks** → `benchmark-analysis` skill

### Agents (Autonomous Workers)

Use agents for complex, multi-step tasks:

```
"Use the performance-auditor agent to analyze the HTTP/1.1 parser"
"Use the protocol-validator agent to check HTTP/2 compliance"
"Use the benchmark-runner agent to run full performance suite"
```

Agents work autonomously with restricted tool access and provide comprehensive reports.

### Hooks (Automatic Validation)

Hooks run automatically at key points:

#### PreToolUse
- **Before Write/Edit**: Checks for performance anti-patterns
- **Before git commit**: Validates benchmarks still pass

#### PostToolUse
- **After editing .go files**: Suggests running benchmarks if hot path modified
- **After writing tests**: Runs new tests to verify they pass

#### PrePrompt
- **Before each interaction**: Loads current performance metrics into context

## Workflow Examples

### Example 1: Optimizing Parser Performance

**User**: "Optimize the HTTP/1.1 request parser"

**Claude**:
1. Invokes `go-performance-optimization` skill automatically
2. Reads current parser implementation
3. Runs benchmarks to establish baseline
4. Analyzes escape analysis
5. Suggests optimizations
6. After editing, hook suggests running `/bench`
7. User runs `/bench` to verify improvement

### Example 2: Comprehensive Performance Audit

**User**: "Use the performance-auditor agent to analyze the codebase"

**Agent**:
1. Reviews hot path code for anti-patterns
2. Runs escape analysis on critical packages
3. Executes benchmark suite
4. Generates memory and CPU profiles
5. Provides detailed report with specific file:line references
6. Suggests prioritized optimizations

### Example 3: Protocol Compliance Check

**User**: "Validate HTTP/2 implementation against RFC 7540"

**Claude**:
1. Invokes `http-protocol-testing` skill
2. Or user can say: "Use protocol-validator agent for thorough check"
3. Runs RFC compliance tests
4. Checks frame parsing, HPACK, flow control
5. Tests security aspects
6. Generates compliance report with pass/fail for each section

## Customization

### Adding New Commands

Create a new file in `.claude/commands/`:

```markdown
# Command Description

Your task description here...

## Steps
1. Do this
2. Do that
```

### Creating New Skills

1. Create directory: `.claude/skills/your-skill/`
2. Add `SKILL.md` with frontmatter:
   ```markdown
   ---
   name: your-skill
   description: When to use this skill
   allowed-tools: Read, Bash, Grep
   ---

   # Your Skill Instructions
   ```
3. Optionally add `context/` and `workflows/` subdirectories

### Adding New Agents

Create `.claude/agents/your-agent.md`:

```markdown
---
name: your-agent
description: What this agent does
tools: Read, Grep, Bash
---

# Agent Instructions
```

### Configuring Hooks

Edit `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "pattern",
        "hooks": [
          {
            "type": "command",
            "description": "What it does",
            "command": "bash script.sh"
          }
        ]
      }
    ]
  }
}
```

## Performance Philosophy

This ecosystem enforces Shockwave's core principle:

**Performance is not just a feature, it's a philosophy.**

Every component is designed to:
- Measure before optimizing
- Validate with benchmarks
- Prevent regressions
- Maintain zero allocations in hot paths
- Follow RFC specifications exactly

## Best Practices

### When to Use Commands vs Skills vs Agents

- **Commands**: Quick, specific tasks you know you want
  - Example: `/bench` when you want to run benchmarks

- **Skills**: Let Claude decide when to use them
  - Example: Ask "How can I optimize this?" and Claude will invoke the right skill

- **Agents**: Complex, multi-step analysis or validation
  - Example: "Audit the entire codebase for performance issues"

### Benchmark-Driven Development

1. Before optimization: `/bench` to establish baseline
2. During optimization: Skills guide implementation
3. After optimization: Hooks suggest running `/bench`
4. Before commit: Hook validates no regressions

### Preventing Allocations

1. Edit hot path code → Hook warns about anti-patterns
2. After editing → Hook suggests `/check-allocs`
3. Run `/check-allocs` → Verifies 0 allocs/op
4. Commit → Hook validates benchmarks pass

## Troubleshooting

### Hooks Not Running

Check `.claude/hooks/*.sh` are executable:
```bash
chmod +x .claude/hooks/*.sh
```

### Skills Not Activating

Verify `SKILL.md` has proper frontmatter:
```markdown
---
name: skill-name
description: Clear description of when to use
---
```

### Commands Not Found

Commands must be in `.claude/commands/` with `.md` extension.

## Claude 4.5 Optimizations

This ecosystem leverages Claude Sonnet 4.5's advanced capabilities:

### Precise Instruction Following

Claude 4.5 excels when given:
- **Explicit goals**: "Achieve zero allocations in ParseRequest"
- **Context**: "...because allocations degrade p99 latency"
- **Success criteria**: "Target: 0 allocs/op, maintain RFC compliance"

See [PROMPTING_GUIDE.md](PROMPTING_GUIDE.md) for detailed examples.

### Native Subagent Orchestration

Claude 4.5 automatically delegates complex tasks to agents when beneficial:
```
"Audit pkg/shockwave/http11/ for performance issues"
→ Claude may invoke performance-auditor agent autonomously
```

See [AGENT_PATTERNS.md](AGENT_PATTERNS.md) for orchestration patterns.

### Parallel Tool Execution

Claude 4.5 aggressively parallelizes operations:
- Reads multiple files simultaneously
- Runs benchmarks and escape analysis concurrently
- Profiles CPU and memory in parallel

This speeds up performance analysis significantly.

### Long-Horizon Reasoning

For extended optimization sessions:
- Uses git for state tracking
- Maintains structured progress files
- Continues seamlessly across context windows
- Focuses on incremental, verifiable progress

See [PROMPTING_GUIDE.md](PROMPTING_GUIDE.md#long-horizon-reasoning-for-performance-work) for state management patterns.

### Extended Thinking

Claude 4.5 can reflect on benchmark results and profiling data before acting:
- Validates statistical significance (benchstat p-values)
- Checks for regressions across full benchmark suite
- Verifies escape analysis aligns with expectations
- Plans multi-step optimizations systematically

---

## Best Practices Summary

### For Users

**Optimizing code:**
```
Optimize ParseRequest in pkg/shockwave/http11/parser.go to achieve zero
allocations. This is critical because it's called for every request and
allocations degrade p99 latency. Target: 0 allocs/op, maintain RFC 7230
compliance.
```

**Analyzing performance:**
```
Use the performance-auditor agent to analyze pkg/shockwave/http11/ for
allocation hotspots. Focus on parser.go and response.go. Return file:line
references, escape analysis evidence, and prioritized recommendations.
```

**Running benchmarks:**
```
/bench
```

### For Skill/Agent Authors

**Create discoverable skills:**
- Keyword-rich descriptions matching user queries
- Clear workflows with verification steps
- Concrete before/after examples
- Context explaining rationale

**Create effective agents:**
- Clear mission and success criteria
- Restricted, appropriate toolsets
- Structured output format templates
- Evidence-based constraints

See [SKILL_PATTERNS.md](SKILL_PATTERNS.md) and [AGENT_PATTERNS.md](AGENT_PATTERNS.md) for comprehensive guidance.

---

## Learn More

### Core Documentation
- **[PROMPTING_GUIDE.md](PROMPTING_GUIDE.md)** - Claude 4.5 best practices
- **[AGENT_PATTERNS.md](AGENT_PATTERNS.md)** - Effective agent orchestration
- **[SKILL_PATTERNS.md](SKILL_PATTERNS.md)** - Writing discoverable skills
- **[../CLAUDE.md](../CLAUDE.md)** - Project constitution

### Reference Materials
- **Performance Guide**: `skills/go-performance-optimization/context/go-performance-guide.md`
- **Main README**: `../README.md`

---

**Built with Claude Sonnet 4.5 for high-performance Go** ⚡

*The benchmark is the source of truth.*
