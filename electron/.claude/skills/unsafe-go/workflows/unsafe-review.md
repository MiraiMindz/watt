# Review Unsafe Code Usage

You are tasked with reviewing all uses of Go's `unsafe` package to ensure they are correct, safe, and properly documented.

## Your Task

1. **Find All Unsafe Usage**
   ```bash
   # Find all files using unsafe
   grep -rn "import.*unsafe" --include="*.go"

   # Find all unsafe operations
   grep -rn "unsafe\." --include="*.go"
   ```

2. **For Each Unsafe Usage, Verify:**
   - [ ] Has performance justification (>2x speedup or zero-alloc)
   - [ ] Safety invariants are documented
   - [ ] Bounds checking is present
   - [ ] Alignment is correct
   - [ ] Tests cover edge cases
   - [ ] Race detector passes

3. **Check Documentation**
   Each unsafe function should have:
   ```go
   // SAFETY INVARIANTS:
   // 1. [Invariant 1]
   // 2. [Invariant 2]
   //
   // SAFETY REASONING:
   // [Why this is safe]
   //
   // PERFORMANCE:
   // [Benchmark showing benefit]
   //
   // CAUTIONS:
   // [What can go wrong]
   ```

4. **Run Safety Checks**
   ```bash
   # Run race detector
   go test -race ./...

   # Check escape analysis
   go build -gcflags='-m' ./... 2>&1 | grep unsafe
   ```

## Output Format

```markdown
# Unsafe Code Review Report

## Summary
Found [X] uses of unsafe package in [Y] files.

## Files Using Unsafe
- [file:line] - [operation type]
- [file:line] - [operation type]

## Detailed Review

### [File:Function]

**Code:**
```go
[code snippet]
```

**Operation:** [Pointer arithmetic / Type conversion / etc.]

**Safety Checklist:**
- [✅/❌] Performance justification documented
- [✅/❌] Safety invariants documented
- [✅/❌] Bounds checking present
- [✅/❌] Alignment verified
- [✅/❌] Tests cover edge cases
- [✅/❌] Race detector passes

**Issues Found:**
- [List any issues]

**Recommendations:**
- [Suggestions for improvement]

## High-Risk Issues
[Critical issues that must be fixed]

## Recommendations
[Overall suggestions for unsafe code quality]

## Action Items
- [ ] [Fix critical issue 1]
- [ ] [Improve documentation for X]
- [ ] [Add tests for edge case Y]
```

## Notes
- Use the unsafe-code-reviewer agent for detailed analysis
- Flag any unsafe code without clear justification
- Suggest safe alternatives when possible
