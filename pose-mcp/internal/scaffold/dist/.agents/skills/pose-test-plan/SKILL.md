---
name: pose-test-plan
description: Use to define an explicit test plan before implementing medium or high-risk changes, sensitive contracts, or cross-service impact. Covers layers, negative scenarios, deterministic commands, and expected evidence. Trigger keywords - test plan, risk-based testing, regression strategy, contract test, cross-service, e2e plan.
when_to_use: The task has medium or high risk, touches HTTP, schema, or event contracts, or affects multiple services. Use before coding to define verifiable acceptance and avoid informal local-only testing.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write
---

# Skill: pose-test-plan

## Required reading

1. The applicable feature or bugfix workflow.
2. Applicable domain rules.
3. [`.pose/indexes/validation-matrix.json`](../../../.pose/indexes/validation-matrix.json).
4. [`.pose/indexes/module-metadata.json`](../../../.pose/indexes/module-metadata.json).

## Steps

1. Identify affected modules and real criticality.
2. Define unit coverage always, integration or contract coverage for medium risk and above, and end-to-end smoke coverage for high risk and above.
3. Map invalid input, denied authorization, timeout, unavailable dependencies, and documented fallback behavior.
4. List deterministic required and optional commands; reuse `pose validate --module <path> --report --report-task test-plan-baseline-<slug>`.
5. Define expected output, metric, or schema evidence for each command.
6. Attach the plan to the spec Validation section before implementation.
7. Update the validation matrix when a scenario should become a permanent gate, then run `pose check --strict`.

## Output requirements

- A spec Validation table with scenario, command, and expected evidence.
- Explicit negative scenarios, not only happy paths.
- Copy-pasteable commands without abstract placeholders in the final plan.
- Clear required versus optional classification.
- Valid matrix update when needed.
