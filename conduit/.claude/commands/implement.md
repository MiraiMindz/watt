# Implement Command

Implement a feature following an existing plan.

## Usage

```
/implement <plan-file>
```

## What This Does

Uses `gox-implementer` agent to:

1. Read the implementation plan
2. Study existing code patterns
3. Implement the feature incrementally
4. Write tests as code is written
5. Follow all coding standards
6. Document the code

## Example

```bash
/implement plans/csr-transpiler-plan.md
```

## Process

1. **Read Plan** - Understand what to build
2. **Study Patterns** - Learn from existing code
3. **Implement** - Write code incrementally
4. **Test** - Verify each piece works
5. **Document** - Add godoc comments
6. **Review** - Self-check quality

## Output

```markdown
## Implementation Complete

**Feature:** CSR Transpiler
**Files Created:**
- pkg/transpiler/csr/transpiler.go
- pkg/transpiler/csr/transpiler_test.go

**Tests:** 15 passing
**Coverage:** 82%
**Performance:** Meets targets

**Ready for review**
```

## Next Steps

- Use `/review` to get code review
- Use `/test` to run full test suite
- Use `/doc` to generate documentation
