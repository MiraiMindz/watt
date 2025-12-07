# Capacitor Claude Code Ecosystem Guide

This document explains the complete Claude Code ecosystem created for the Capacitor DAL/DTO project.

## Overview

The Capacitor project includes a comprehensive Claude Code ecosystem designed to help you build high-performance, zero-allocation Go code following the WATT Toolkit philosophy. This ecosystem includes:

- **CLAUDE.md**: Project constitution and coding standards
- **Agent Skills**: Reusable expertise for DAL generation and performance analysis
- **Slash Commands**: Quick actions for common development tasks
- **Agents**: Specialized workers for optimization tasks
- **Hooks**: Quality gates and validation
- **Configuration**: Project-specific settings

## Directory Structure

```
capacitor/
├── CLAUDE.md                          # Project constitution
├── CLAUDE_ECOSYSTEM_GUIDE.md          # This file
├── README.md                          # User-facing documentation
├── .claude/                           # Claude Code ecosystem
│   ├── settings.json                  # Project settings and hooks
│   ├── skills/                        # Agent Skills
│   │   ├── dal-generator/
│   │   │   ├── SKILL.md              # DAL generation expertise
│   │   │   └── POOLING.md            # Pooling strategies reference
│   │   └── performance-analyzer/
│   │       └── SKILL.md              # Performance profiling expertise
│   ├── commands/                      # Slash commands
│   │   ├── benchmark.md              # Run benchmarks
│   │   ├── gen-cache.md              # Generate cache layer
│   │   ├── gen-backend.md            # Generate database backend
│   │   ├── profile.md                # Performance profiling
│   │   ├── test.md                   # Run tests
│   │   └── compare.md                # Compare implementations
│   └── agents/                        # Specialized agents
│       └── cache-optimizer.md         # Cache optimization agent
├── pkg/                               # Source code
├── benchmarks/                        # Benchmark results
├── examples/                          # Usage examples
└── tools/                            # Development tools
```

## Component Details

### 1. CLAUDE.md - The Project Constitution

**Location**: `CLAUDE.md`

**Purpose**: The single source of truth for how the Capacitor project works. Claude reads this first to understand:
- Project goals and philosophy
- Architecture principles
- Code standards and patterns
- Performance requirements
- Development workflow

**Key Sections**:
- **Project Overview**: Zero-allocation philosophy, performance goals
- **Architecture Principles**: How to structure code
- **Code Standards**: Naming, testing, documentation requirements
- **Performance Patterns**: Pooling, inlining, unsafe operations
- **Error Handling**: Custom error types and wrapping
- **API Design**: Generic interfaces and builder patterns

**When to Update**:
- When adding new architectural patterns
- When establishing new code standards
- When defining new performance targets
- When changing development workflow

### 2. Agent Skills

#### dal-generator Skill

**Location**: `.claude/skills/dal-generator/SKILL.md`

**Triggers**: When you mention:
- "generate cache layer"
- "create DAL implementation"
- "new database backend"
- "implement DTO"

**What It Does**:
- Provides templates for cache layers, database backends, and DTOs
- Ensures zero-allocation patterns (sync.Pool, preallocated buffers)
- Generates comprehensive tests and benchmarks
- Follows project naming and structure conventions

**Supporting Files**:
- `POOLING.md`: Deep dive on object pooling strategies
- Templates would go in `templates/` (to be created as needed)
- Scripts would go in `scripts/` (to be created as needed)

**Example Use**:
```
You: "Create a new memory cache for storing User objects with string keys"

Claude: [Uses dal-generator skill]
- Asks about TTL, eviction strategy, size limits
- Generates cache implementation with sync.Pool
- Creates test file with table-driven tests
- Creates benchmark suite
- Provides usage example
```

#### performance-analyzer Skill

**Location**: `.claude/skills/performance-analyzer/SKILL.md`

**Triggers**: When you mention:
- "benchmark"
- "performance profile"
- "optimize"
- "allocations"
- "latency"

**What It Does**:
- Runs benchmarks with proper configuration
- Generates CPU, memory, and mutex profiles
- Analyzes allocation hotspots
- Compares performance with baselines
- Suggests specific optimizations
- Validates against performance targets

**Example Use**:
```
You: "Profile the cache Get operation and find why it's allocating"

Claude: [Uses performance-analyzer skill]
- Runs benchmark with -benchmem
- Generates memory profile
- Analyzes with pprof
- Identifies allocation source
- Suggests pool-based fix
- Shows before/after comparison
```

### 3. Slash Commands

Quick actions for common tasks. Type `/command-name` to use.

#### /benchmark

**What**: Run comprehensive benchmarks
**When**: After implementing or changing performance-critical code
**Output**: Benchmark results with allocation counts, comparisons

#### /gen-cache

**What**: Generate a complete cache layer implementation
**When**: Need a new cache type (memory, disk, network)
**Output**: Implementation, tests, benchmarks, documentation

#### /gen-backend

**What**: Generate a database backend implementation
**When**: Need to integrate a new database type
**Output**: Backend code, serializers, tests, docker-compose for testing

#### /profile

**What**: Run profiling and analyze results
**When**: Investigating performance issues
**Output**: CPU/memory profiles, hotspot analysis, optimization suggestions

#### /test

**What**: Run comprehensive test suite with coverage
**When**: Before committing, after changes
**Output**: Test results, coverage report, suggestions for uncovered code

#### /compare

**What**: Compare Capacitor with competitors or baseline
**When**: Validating performance claims
**Output**: Comparative benchmarks, analysis, graphs

### 4. Specialized Agents

#### cache-optimizer Agent

**Location**: `.claude/agents/cache-optimizer.md`

**Purpose**: Specialized agent that focuses exclusively on optimizing cache implementations to achieve zero-allocation performance.

**When to Use**:
```
You: "Optimize the memory cache to eliminate allocations"

Claude: [Launches cache-optimizer agent]
```

**What It Does**:
1. **Phase 1: Analysis**
   - Runs benchmarks for baseline
   - Generates CPU/memory profiles
   - Identifies allocation sources

2. **Phase 2: Optimization**
   - Adds object pooling
   - Eliminates interface boxing
   - Preallocates slices
   - Optimizes string operations
   - Reduces lock contention

3. **Phase 3: Verification**
   - Runs new benchmarks
   - Compares with baseline
   - Verifies zero allocations
   - Ensures tests still pass

4. **Phase 4: Documentation**
   - Documents all changes
   - Explains trade-offs
   - Updates performance metrics

**Expertise Areas**:
- sync.Pool usage patterns
- Per-CPU pools for hot paths
- Lock-free data structures
- Cache-line alignment
- Assembly optimization

### 5. Hooks - Quality Gates

**Location**: `.claude/settings.json`

Hooks run automatically at specific points in the development workflow.

#### PreToolUse Hooks

Run BEFORE Claude writes/edits files:

1. **prevent-panics**: Blocks panic() in library code
   - Why: Library code must return errors, not panic
   - Failure: Blocks the write operation

2. **prevent-any-type**: Blocks use of 'any' type
   - Why: Use generics or concrete types for type safety
   - Failure: Blocks the write operation

3. **check-godoc**: Warns if exported symbols lack documentation
   - Why: All public APIs need documentation
   - Failure: Warning only (doesn't block)

#### PostToolUse Hooks

Run AFTER Claude writes files:

1. **format-code**: Automatically runs gofmt
   - Why: Consistent code formatting
   - Effect: File is formatted in-place

#### SubagentStop Hooks

Run when a specialized agent completes:

1. **summarize-work**: Reviews agent's work
   - Checks if tests need to run
   - Verifies code quality
   - Suggests next steps

### 6. Settings and Configuration

**Location**: `.claude/settings.json`

**Key Settings**:

```json
{
  "quality_gates": {
    "min_test_coverage": 85,
    "require_benchmarks": true,
    "require_godoc": true,
    "no_panics": true,
    "no_any_type": true,
    "no_reflection_in_hot_paths": true
  },
  "performance": {
    "max_allocs_hot_path": 0,
    "max_latency_cache_get_ns": 100,
    "max_latency_cache_set_ns": 200
  }
}
```

## Workflow Examples

### Example 1: Creating a New Cache Layer

```
You: "I need a Redis-backed cache layer for User objects"

Claude: [Invokes dal-generator skill]
1. Asks clarifying questions:
   - Key type? (e.g., int64, string)
   - Serialization format? (JSON, MessagePack, Protobuf)
   - TTL requirements?
   - Connection pool settings?

2. Generates files:
   - pkg/cache/redis/cache.go (implementation)
   - pkg/cache/redis/cache_test.go (tests)
   - pkg/cache/redis/cache_bench_test.go (benchmarks)
   - examples/redis-cache/main.go (example)
   - pkg/cache/redis/README.md (documentation)

3. Hooks validate:
   - No panic() calls ✓
   - No 'any' type ✓
   - All exported symbols documented ✓
   - Code formatted ✓

4. Suggests next steps:
   - Run tests: make test
   - Run benchmarks: make bench-cache
   - Compare with baseline: make bench-compare
```

### Example 2: Optimizing Performance

```
You: "The memory cache Get operation is too slow"

Claude: [Uses performance-analyzer skill]
1. Runs benchmark:
   - Current: 250 ns/op, 2 allocs/op
   - Target: < 100 ns/op, 0 allocs/op

2. Generates profiles:
   - CPU profile shows lock contention
   - Memory profile shows entry allocation

3. Suggests optimizations:
   - Use RWMutex instead of Mutex
   - Add sync.Pool for entries
   - Preallocate map capacity

4. You accept suggestions, Claude applies them

5. Re-benchmarks:
   - New: 85 ns/op, 0 allocs/op ✓
   - Improvement: 66% faster, 100% fewer allocations

6. Updates documentation with new metrics
```

### Example 3: Running Comprehensive Tests

```
You: /test

Claude:
1. Runs tests with race detector
2. Generates coverage report
3. Checks coverage threshold (>85%)
4. Reports results:
   - All tests pass ✓
   - Coverage: 87% ✓
   - No race conditions ✓
5. Identifies uncovered functions
6. Suggests test cases to add
```

## Best Practices

### 1. Start with CLAUDE.md

Before asking Claude to write code, ask:
```
"Review CLAUDE.md and tell me the key principles for this project"
```

This ensures Claude understands the project context.

### 2. Use Skills for Generation

Instead of:
```
"Write a cache implementation"
```

Say:
```
"Generate a memory cache implementation for User objects with int64 keys"
```

This triggers the dal-generator skill which ensures all patterns are followed.

### 3. Always Benchmark

After any optimization:
```
/benchmark
/compare
```

Then:
```
"Compare these results with the performance targets in CLAUDE.md"
```

### 4. Let Hooks Protect You

The hooks will prevent common mistakes:
- Using panic() in library code
- Using 'any' instead of generics
- Missing documentation

If a hook blocks your operation, it's usually for good reason. Fix the issue rather than trying to bypass the hook.

### 5. Use Agents for Complex Tasks

For deep optimization work:
```
"Launch the cache-optimizer agent to eliminate all allocations from the memory cache"
```

The agent will systematically work through the optimization process.

## Customization

### Adding a New Skill

1. Create directory: `.claude/skills/your-skill/`
2. Create `SKILL.md` with frontmatter:
```yaml
---
name: your-skill
description: When to use this skill
allowed-tools: Read, Write, Edit, Bash
---

# Your Skill

Instructions for Claude...
```

3. Add supporting files (templates, scripts, references)

### Adding a New Command

1. Create `.claude/commands/your-command.md`
2. Write instructions for what the command should do
3. Test by typing `/your-command`

### Adding a New Hook

Edit `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "name": "your-check",
            "description": "What it checks",
            "command": "your shell command"
          }
        ]
      }
    ]
  }
}
```

## Troubleshooting

### Skills Not Triggering

**Problem**: Claude doesn't use the skill automatically

**Solution**:
- Be explicit: "Use the dal-generator skill to create..."
- Check the skill description matches your request
- Verify the skill's `description` in the frontmatter

### Hooks Blocking Operations

**Problem**: Hook prevents Claude from writing code

**Solution**:
- Read the hook error message
- Fix the underlying issue
- Don't try to bypass hooks - they enforce quality

### Agent Not Finding Files

**Problem**: Agent can't read project files

**Solution**:
- Check file paths in error messages
- Ensure files exist in expected locations
- Agent has access to Read, Glob, Grep tools

## Performance Targets Reference

Quick reference for Capacitor performance targets (from CLAUDE.md):

| Operation | Target | Notes |
|-----------|--------|-------|
| Memory Cache Get | < 100 ns/op, 0 allocs/op | Hot path |
| Memory Cache Set | < 200 ns/op, ≤1 alloc/op | For new entries |
| Disk Cache Get | < 100 μs/op, ≤2 allocs/op | SSD assumed |
| Disk Cache Set | < 200 μs/op, ≤2 allocs/op | SSD assumed |
| DB Backend Get | < 1 ms/op, ≤5 allocs/op | Local network |
| DB Backend Set | < 2 ms/op, ≤5 allocs/op | Local network |

## Summary

The Capacitor Claude Code ecosystem provides:

1. **CLAUDE.md**: Project constitution - read this first
2. **Skills**: Automated expertise for generation and analysis
3. **Commands**: Quick actions for common tasks
4. **Agents**: Specialized workers for complex optimization
5. **Hooks**: Quality gates that enforce standards
6. **Settings**: Project configuration

Together, these components ensure:
- Consistent code quality
- Zero-allocation performance
- Comprehensive testing
- Proper documentation
- Adherence to WATT Toolkit philosophy

Start with simple commands like `/test` and `/benchmark`, then explore skills for code generation, and finally use agents for deep optimization work.

The ecosystem is designed to help you build the fastest DAL/DTO library in Go while maintaining code quality and correctness.
