# /implement Command

Implement a feature from an architecture plan.

## Usage

```
/implement <feature-name>
```

## What This Command Does

1. Reads the architecture plan from `docs/architecture/<feature-name>.md`
2. Implements the feature according to the plan
3. Creates necessary files (core, pool, Shockwave adapters)
4. Writes comprehensive tests
5. Adds benchmarks
6. Updates documentation

## Requirements

- Architecture plan must exist: `docs/architecture/<feature-name>.md`
- Plan created by `/plan` command or architect agent

## Constraints

- MUST use Shockwave types (no net/http)
- MUST follow CLAUDE.md coding standards
- MUST achieve >80% test coverage
- MUST include benchmarks
- MUST verify zero-allocation claims (if applicable)

## Example

```
/implement context-pooling
```

Will create:
- `pool/context_pool.go` - Pool implementation
- `pool/context_pool_test.go` - Unit tests
- `pool/context_pool_bench_test.go` - Benchmarks
- Updated `core/context.go` - Add Reset() method

## Validation Steps

Before completion, the command verifies:
- [ ] Code compiles
- [ ] Tests pass
- [ ] Coverage >80%
- [ ] Benchmarks run successfully
- [ ] No net/http imports
- [ ] Code formatted with gofmt

## Agent Used

This command invokes the **implementer** agent which:
- Has Read, Write, Edit, Bash tools
- Follows architecture plans precisely
- Maintains zero-allocation code paths
- Writes tests and benchmarks alongside code

---

*See MASTER_PROMPTS.md for detailed implementation guidelines*
