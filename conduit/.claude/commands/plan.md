# Plan Command

Create a detailed implementation plan for a new feature or component.

## Usage

```
/plan <feature-name>
```

## What This Does

1. Analyzes current codebase
2. Reviews existing documentation
3. Creates detailed implementation plan
4. Identifies dependencies and risks
5. Breaks work into phases

## Process

Use the `gox-planner` agent to:

1. **Understand the Request**
   - What feature is being requested?
   - What are the requirements?
   - What are the constraints?

2. **Analyze Current State**
   - What exists already?
   - What needs to be built?
   - Where are the gaps?

3. **Design Solution**
   - How should it work?
   - What are the key decisions?
   - What are the alternatives?

4. **Create Plan**
   - Break into phases
   - List specific tasks
   - Identify dependencies
   - Assess risks

5. **Document Plan**
   - Write plan to `plans/<feature-name>.md`
   - Include all sections
   - Make it actionable

## Output

Creates a file: `plans/<feature-name>-plan.md`

## Example

```bash
/plan csr-transpiler
```

This will analyze the need for a CSR transpiler and create a comprehensive implementation plan.

## Next Steps

After planning:
- Review plan with user
- Use `/implement` to execute plan
- Use `/test` to verify implementation
