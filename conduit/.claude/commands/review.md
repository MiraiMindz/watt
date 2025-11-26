# Review Command

Perform comprehensive code review of recent changes.

## Usage

```
/review [files...]
```

## What This Does

Uses `gox-reviewer` agent to check:

- Code quality and Go idioms
- Performance and efficiency
- Test coverage
- Documentation
- Standards compliance

## Review Areas

1. **Code Quality** - Clean, readable, idiomatic
2. **Performance** - Efficient algorithms, minimal allocations
3. **Testing** - Comprehensive, passing
4. **Documentation** - Clear, complete
5. **Standards** - Follows CLAUDE.md

## Output

```markdown
## Code Review

**Overall:** ✅ APPROVED

### Strengths
- Well-structured
- Comprehensive tests
- Good performance

### Issues
**Medium Priority:**
- Consider pre-allocating slice at line 42

**Minor:**
- Typo in comment line 15

### Recommendation
APPROVE with minor fixes
```
