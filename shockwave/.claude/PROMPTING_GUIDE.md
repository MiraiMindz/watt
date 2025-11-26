# Claude 4.5 Prompting Guide for Shockwave

This guide documents best practices for working with Claude Sonnet 4.5 in the Shockwave project. It incorporates official Anthropic recommendations optimized for high-performance Go development.

---

## Core Principles

### 1. Be Explicit and Contextual

Claude 4.5 excels at precise instruction following. Always provide:
- **Explicit instructions** about what you want
- **Context** explaining why it matters
- **Success criteria** defining done

**Less effective:**
```
Optimize this function
```

**More effective:**
```
Optimize the ParseRequest function to achieve zero allocations. This is critical
because it's called for every HTTP request, and allocations here create GC pressure
that degrades p99 latency. Target: 0 allocs/op, maintain current throughput.
```

### 2. Add "Why" to Improve Performance

Explaining the motivation behind constraints helps Claude make better decisions.

**Example:**
```
NEVER use string concatenation in hot paths because this project targets
zero allocations for HTTP request handling. Every allocation in the critical
path causes GC pressure that degrades tail latency. Use pre-compiled []byte
constants or sync.Pool buffers instead.
```

### 3. Tell Claude What TO Do (Not What NOT To Do)

**Less effective:**
```
Don't use markdown in technical reports
```

**More effective:**
```
Write technical reports as smoothly flowing prose paragraphs with clear section
breaks. Reserve markdown only for inline code (`code`), code blocks (```...```),
and section headings (###).
```

---

## Working with Tools

### Encourage Parallel Tool Execution

Claude 4.5 excels at parallel tool calling. For Shockwave's performance-critical workflow:

```
<use_parallel_tool_calls>
When reading multiple files or running independent operations, execute them in
parallel. For example, when analyzing performance:
- Read parser.go, pool.go, and benchmark files simultaneously
- Run benchmarks, escape analysis, and CPU profiling in parallel

Only run operations sequentially when one depends on the output of another.
Never use placeholders or guess parameters.
</use_parallel_tool_calls>
```

### Default to Action (For Implementation Tasks)

```
<default_to_action>
When working on Shockwave optimizations, default to implementing changes rather
than only suggesting them. If you need clarification, use tools to discover
missing details (read files, run benchmarks) rather than asking questions.

The exception: Always ask before committing changes or modifying test files.
</default_to_action>
```

### Conservative Approach (For Research/Analysis)

```
<investigate_before_answering>
Never speculate about code you haven't opened. If analyzing performance issues:
1. Read the actual implementation first
2. Run benchmarks to measure current state
3. Use escape analysis to verify assumptions
4. Only then provide grounded, evidence-based recommendations

Make zero claims about code before investigating. The benchmark is the source of truth.
</investigate_before_answering>
```

---

## Long-Horizon Reasoning for Performance Work

### State Tracking Best Practices

For extended optimization sessions:

**Use structured state files:**
```json
// .claude/state/optimization_progress.json
{
  "baseline_benchmarks": {
    "BenchmarkParseRequest": {"ns_op": 1250, "allocs_op": 3, "b_op": 256},
    "BenchmarkWriteResponse": {"ns_op": 890, "allocs_op": 1, "b_op": 128}
  },
  "targets": {
    "BenchmarkParseRequest": {"allocs_op": 0, "status": "in_progress"},
    "BenchmarkWriteResponse": {"allocs_op": 0, "status": "completed"}
  },
  "files_modified": [
    "pkg/shockwave/http11/parser.go",
    "pkg/shockwave/http11/response.go"
  ]
}
```

**Use git for progress tracking:**
```
git log --oneline shows what's been optimized
git diff shows current changes
Each commit represents a verified optimization
```

**Use unstructured notes for context:**
```text
// progress_notes.txt
Session 3: Parser Optimization
- Eliminated string concatenation in status line (2 allocs saved)
- Switched to inline header array (1 alloc saved)
- Next: Investigate buffer pooling for request bodies
- Don't remove tests - this could break protocol compliance
```

### Multi-Session Workflow

When optimization spans multiple context windows:

1. **First session**: Establish baseline
   ```
   Run comprehensive benchmarks, create baseline.txt, commit results
   Write tests.json tracking zero-allocation targets
   Create optimization_plan.md with prioritized improvements
   ```

2. **Subsequent sessions**: Iterate on todo list
   ```
   Review progress_notes.txt and git log
   Load benchmark baselines from baseline.txt
   Work through optimization_plan.md incrementally
   Update state files after each change
   ```

3. **Context window management**:
   ```
   As you approach token budget limits, save state to files:
   - Update progress_notes.txt with current status
   - Commit working changes
   - Update optimization_plan.md with remaining work

   Fresh context windows should:
   - Review git log to understand what's been done
   - Read progress_notes.txt for context
   - Run benchmarks to verify current state
   - Continue from optimization_plan.md
   ```

### Complete Usage of Context

```
<efficient_context_usage>
Shockwave optimization tasks benefit from thorough work in each session. Plan
your work to efficiently use the full context window:

1. Group related optimizations (e.g., all parser changes together)
2. Run benchmarks after logical checkpoints, not after every tiny change
3. Make steady progress on complete components
4. Before context limit, ensure all changes are committed and state is saved

Work systematically until you complete logical milestones. Don't artificially
stop due to token budget concerns - state will be saved automatically.
</efficient_context_usage>
```

---

## Communication Style

### Progress Reporting

Claude 4.5's natural style is concise and grounded. For Shockwave, encourage updates:

```
After completing optimization work involving tool use, provide a brief summary:
- What was optimized (file:line)
- Benchmark improvement (before/after)
- Next logical step

Example:
"Optimized ParseRequest in parser.go:123. Reduced allocs from 3 to 0,
improved throughput 15%. Running full benchmark suite to verify no regressions."
```

### Technical Writing Format

```
<technical_writing_style>
For performance reports and architecture documents, use flowing prose with:
- Clear section headings (###)
- Inline code for function names (`ParseRequest`)
- Code blocks for examples (```go...```)
- Bullet points only for truly discrete lists (benchmark results, checklist items)

Avoid excessive markdown formatting. Write to be read by engineers, not to look pretty.
</technical_writing_style>
```

---

## Research and Analysis

### Structured Research Approach

For complex questions (e.g., "Why is HTTP/2 slower than HTTP/1.1?"):

```
<structured_research>
Break down complex investigations systematically:

1. Form hypotheses (allocations? syscalls? contention?)
2. Gather evidence for each (benchmarks, profiles, traces)
3. Track confidence levels in findings
4. Update research_notes.md with discoveries
5. Self-critique: "Am I missing something obvious?"
6. Synthesize findings into actionable recommendations

The goal is not just answers, but verified, benchmarked truth.
</structured_research>
```

---

## Subagent Orchestration

### When to Delegate

Claude 4.5 naturally delegates complex tasks. For Shockwave:

**Delegate when:**
- Task requires separate analysis context (full codebase audit)
- Specialized expertise needed (protocol compliance validation)
- Parallel work valuable (audit while implementing)

**Don't delegate when:**
- Simple file edits or single optimizations
- Quick benchmark runs
- Questions answerable by reading 2-3 files

### Agent Communication Pattern

```
When launching performance-auditor agent:

"Audit pkg/shockwave/http11/ for allocation hotspots. Focus on parser.go and
response.go. Return:
1. Specific file:line references for each issue
2. Escape analysis evidence
3. Benchmark data showing impact
4. Prioritized fix recommendations

This audit informs optimization work, so precision matters more than speed."
```

---

## Avoiding Common Pitfalls

### Don't Hard-Code Test Solutions

```
<principled_solutions>
When fixing failing tests, implement the correct general algorithm, not a solution
that just makes the specific test pass.

Bad: if input == "test-case-1" { return expected_output }
Good: Implement the actual HTTP/1.1 header parsing logic per RFC 7230

If a test is wrong, explain why rather than working around it. The test suite
ensures protocol compliance - never compromise correctness for convenience.
</principled_solutions>
```

### Minimize Helper Script Proliferation

```
<use_standard_tools>
Use standard tools directly rather than creating wrapper scripts:

Avoid: Writing helper.sh that calls go test with specific flags
Prefer: Using Bash tool with the actual go test command

Wrapper scripts add maintenance burden. Only create them if:
1. The command is very complex (>5 chained operations)
2. It will be reused across many sessions
3. It encapsulates Shockwave-specific knowledge
</use_standard_tools>
```

### Prevent File Clutter

```
<minimize_file_creation>
Avoid creating temporary files for iteration. Prefer:
- Reading/editing existing files
- Using Bash commands with pipes
- Storing small data in progress notes

If you create temporary files for analysis:
1. Use /tmp/ for truly temporary data
2. Clean up before completing tasks
3. Document why the file was necessary

Shockwave's repo should contain only production code, tests, and documentation.
</minimize_file_creation>
```

---

## Frontend/Visualization Work

For creating benchmark dashboards or performance visualization tools:

```
<visualization_excellence>
When creating performance dashboards, go beyond basics:

1. Use professional design (dark theme for consistency with terminal tools)
2. Include thoughtful interactions (hover states, drill-downs)
3. Apply data visualization best practices (appropriate chart types, clear axes)
4. Add context to data (show baseline, highlight regressions)
5. Make it functional on first try (test with real benchmark data)

Performance visualization helps spot trends. Make it useful, not just pretty.
</visualization_excellence>
```

---

## XML Tag Patterns

For complex instructions, use XML tags for clarity:

```
<zero_allocation_validation>
After optimizing any function in these critical paths:
- pkg/shockwave/http11/parser.go
- pkg/shockwave/http11/response.go
- pkg/shockwave/server/handler.go

Verify zero allocations:
1. Run: go test -bench=BenchmarkFunctionName -benchmem
2. Confirm: allocs/op = 0
3. If not zero, run escape analysis to find the leak

Zero allocations in hot paths is non-negotiable. Every allocation adds GC pressure.
</zero_allocation_validation>
```

---

## Model Self-Knowledge

For tools or documentation that reference Claude:

```
<model_identification>
The assistant is Claude, created by Anthropic.
Current model: Claude Sonnet 4.5 (claude-sonnet-4-5-20250929)

When generating code comments or docs, you may reference "AI-assisted" or
"machine-generated" but avoid claiming human authorship.
</model_identification>
```

---

## Thinking and Reflection

Encourage interleaved thinking for complex performance decisions:

```
After running benchmarks or profiling, reflect on the results:

<extended_thinking>
After receiving benchmark results, carefully analyze:
- Are the improvements real or noise? (Check benchstat p-value)
- Did we create regressions elsewhere? (Review full benchmark suite)
- What does escape analysis reveal? (Are we actually zero-alloc?)
- Does this align with our architectural principles?

Use thinking to plan next steps based on data, not assumptions.
</extended_thinking>
```

---

## Quick Reference

### For Optimization Tasks

```
1. Be explicit: "Optimize X to achieve zero allocations because [reason]"
2. Measure first: Read code, run benchmarks, profile
3. Provide context: Explain why this function is hot/critical
4. Use parallel tools: Read multiple files, run multiple benchmarks simultaneously
5. Track state: Update progress notes, commit incremental improvements
6. Verify: Benchmark after changes, check escape analysis
7. Report: Brief summary with file:line and before/after metrics
```

### For Analysis Tasks

```
1. Investigate first: Never speculate, always read the actual code
2. Gather evidence: Benchmarks, profiles, escape analysis
3. Form hypotheses: Multiple competing theories
4. Test systematically: One variable at a time
5. Document findings: Structured notes with evidence
6. Provide recommendations: Specific, actionable, prioritized
```

### For Agent Delegation

```
1. Clear scope: "Audit these specific packages for these specific issues"
2. Expected output: "Return file:line references and evidence"
3. Success criteria: "Find all allocations in hot paths"
4. Context: "This informs optimization work, so precision matters"
```

---

## Integration with Shockwave Philosophy

This guide complements CLAUDE.md's core principle:

**Performance is not negotiable. Every decision must be justified by benchmarks.**

Claude 4.5's precise instruction following + explicit performance requirements =
systematic, measurable improvements.

The benchmark is the source of truth. 🎯
