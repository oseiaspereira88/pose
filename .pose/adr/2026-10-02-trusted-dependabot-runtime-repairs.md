# ADR: Trusted Dependabot runtime repairs

## Status
Accepted, 2026-10-02. Spec: pose-dependabot-runtime-repair.

## Context
Action-pin updates repeatedly leave runtime evidence stale. Repairing PR code with elevated credentials would expose the repository to code execution under write authority. Token-driven Git writes alone do not trigger the normal push CI event.

## Decision
Use a default-branch workflow_run handler. Read the completed run's same-repository Dependabot PR as data; accept only existing workflows with SHA/version-comment changes and the derived runtime manifest. Execute only the trusted generator. Retain trusted deprecated-runtime policy. Create a one-path child commit through the Git-data API and update the ref without force, then explicitly dispatch CI. Do not approve or merge.

## Consequences
The job needs contents/actions write authority but no additional secrets, caches or PR checkout. Concurrent commits reject the non-fast-forward update. Forks, human PRs and mixed code changes get no repair. Deploying the workflow requires a default-branch merge before a real provider execution can be observed.

Rejected: execute PR scripts under pull_request_target (untrusted code receives authority); add a broad PAT for automatic push events (unnecessary extra secret); remove the runtime manifest (changes the independent evidence contract without evaluating it).
