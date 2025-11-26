# /plan Command

Create a detailed architecture and implementation plan for a feature.

## Usage

```
/plan <feature-name>
```

## What This Command Does

1. Analyzes the feature requirements
2. Creates detailed architecture document
3. Plans implementation steps
4. Predicts performance characteristics
5. Documents trade-offs and design decisions

## Output

Creates: `docs/architecture/<feature-name>.md`

## Example

```
/plan context-pooling
```

Will create `docs/architecture/context-pooling.md` with:
- System design
- Type definitions
- Performance predictions
- Implementation roadmap
- Integration points with Shockwave

## When to Use

- Before implementing any new feature
- When designing system components
- When optimizing existing features
- When integrating with Shockwave

## Agent Used

This command invokes the **architect** agent which:
- Has Read, Grep, Glob tools only
- Creates documentation, not code
- Focuses on design and performance analysis
- Provides implementation guidance

---

*See MASTER_PROMPTS.md for detailed prompt guidelines*
