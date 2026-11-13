# ADR: sandboxed Claude review commands

## Status
Accepted (2026-11-12), implementation decision for `pose-claude-reviewer-runs-tests-in-sandbox`. Runtime qualification remains a delivery gate.

## Context

Governed review projects an agent's `commands.read` and `resume.command`. The existing restricted Claude profile cannot safely permit arbitrary Go test flags through a prefix allow-list. The task sandbox permits editing and cannot be substituted for a review. The trusted-process containment ADR deliberately does not promise filesystem isolation of the Claude process itself.

## Decision

Add an optional `review_profile` to a schema-2 delegation agent, containing a fresh `command` and, when the agent supports resume, `resume_command`. Select it only for governed `review`. Copy the resume configuration before projection. Omission preserves byte-identical legacy adapters; older strict parsers refuse the unknown field. Validate both commands and session placeholders before dispatch. No runtime fallback to `read` or task commands is allowed after selecting this profile. Existing command digests, policy baselines and provenance verification bind the selected adapter without a parallel ledger.

Configure this instance's Claude profile with safe/restricted mode, only Read/Grep/Glob/Bash, a dependency check, and sandbox settings `enabled`, `failIfUnavailable`, `allowUnsandboxedCommands:false`, no excluded commands and no hooks/MCP customizations. Deny Bash writes to the sealed working directory. Put Go build/test caches and temporary files in operation-owned scratch outside the sealed tree, while reading the existing module cache; downloading absent modules is not guaranteed. Use the same enforcement on resume. Keep the restricted read/task profiles and adjudication/smoke behavior unchanged.

The guarantee is Claude Code's OS-enforced shell boundary, including subprocesses, on qualified platforms. It does not turn the trusted Claude executable into an isolated engine backend or protect arbitrary credential reads. Existing checkout integrity and cleanup checks remain mandatory. A real execution must demonstrate permitted scratch writes, denied writes to sealed source, an outside sentinel and shared Git metadata, and actual traced Go test/vet execution. A configuration assertion or fake CLI is not proof of OS isolation. Linux is qualified on this machine; other platforms retain runtime refusal when the sandbox is unavailable and are not claimed qualified by cross-builds.

Rejected alternatives: widening the restricted read allow-list cannot contain executable flags; using the editable task profile violates sealed-source integrity; adding a separate agent name duplicates routing/session configuration; changing all read commands would change task/adjudication/smoke contracts; advertising full-process containment would contradict the backend evidence.

## Consequences

Review can execute tests without permitting source edits. Existing instances opt in by declaring the profile; installing engine code alone does not adopt it. Encode the absolute physical cwd and scratch paths with jq (an additional checked dependency), because relative `.` in inline settings did not protect the checkout during qualification. Claude may use its documented per-user temp fallback for long paths; exclusive full-process confinement is not claimed. Tests that write beside source or need unavailable dependencies are reported unreproducible, not silently relocated or approved. Revisit this decision before changing sandbox vendors/platforms, enabling file-writing tools, or claiming full-process isolation.

References: [Claude sandbox](https://code.claude.com/docs/en/sandboxing), [settings](https://code.claude.com/docs/en/settings-reference), [trusted-process decision](2026-10-25-bound-trusted-reviewer-execution-and-disclose-unavailable-isolation.md).

The first governed qualification exceeded the residual cleanup inventory with its Go build cache. Keep engine cleanup limits unchanged; instruct the reviewer to run `go clean -cache` as the final Bash command inside the mandatory sandbox. Reject outer-wrapper cleanup of the adapter-controlled path: a test could replace it with a symlink to host data. The engine still verifies cleanup with its unchanged owned-resource bounds; instructions alone never certify successful cleanup. Cleanup failure still fails the run; no broader filesystem isolation is implied. Capture failures retain only a closed-vocabulary stage/reason, never withheld raw output.
