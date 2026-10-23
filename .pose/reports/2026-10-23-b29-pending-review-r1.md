**Source decision: changes requested.** Three concrete filesystem parity gaps remain in B29.

Reviewer execution: Codex, not Claude Opus 5.5. This was a separate read-only review execution; no agents were spawned.

Reviewed committed scope: `fe765cf8aaaaff3ec93e91818e69e0781f92116a..7d9c07cc722318fba3789887f81aa0bb0b5e937a`. Findings below are source-derived; reproductions were not executed because this phase prohibits writes.

**Concrete blockers**

1. **Medium — absolute symlink targets are normalized before filesystem traversal.**  
   [operation_files.go:518](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:518)

   `filepath.Clean(target)` removes components whose existence and symlink behavior the kernel must evaluate.

   Reproducible scenario: make a shipped machinery file link to the literal absolute target `/fixture/missing/../victim.md`, with `missing` absent and `victim.md` present. During `update --force`, native access fails because `missing` cannot be traversed. Preview instead resolves `/fixture/victim.md` and can stage a successful overwrite. If the removed component is itself a directory symlink, preview can select a different target, including misidentifying an external effect.

   This reaches the direct machinery writer at [machinery.go:277](/tmp/pose-pending-b29/pose-mcp/internal/cli/machinery.go:277). It is an existing-layout mismatch, not future concurrency uncertainty.

   **Smallest cohesive fix:** preserve absolute target components when resetting the resolver’s root; process `..` after traversing preceding components. Add parity cases for a missing component and a symlink component before `..`.

2. **Medium — staged permissions do not govern subsequent reads or directory traversal.**  
   [operation_files.go:576](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:576), [operation_files.go:606](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:606), [operation_files.go:796](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:796)

   Staged `ReadFile` and `Open` return bytes without checking readability. Staged directory writes check owner-write permission but omit search permission; resolution and staged enumeration likewise omit relevant permission checks.

   Reproducible scenarios, as an unprivileged user:

   - With umask `0111`, a newly created `0755` directory becomes `0644`. Preview still creates children inside it; native apply cannot search it. A missing shipped skill directory exercises this during update.
   - With umask `0400`, an exclusively created configuration-review spec becomes unreadable to its owner. Preview successfully rereads it and opens requests; native rereading fails at [configuration_review.go:378](/tmp/pose-pending-b29/pose-mcp/internal/cli/configuration_review.go:378).

   These permissions are already represented in staged nodes. They are not unknown future ACL or disk conditions.

   **Smallest cohesive fix:** enforce applicable staged read/search/write permissions consistently, accounting for process privilege, or mark unsupported permission evaluation INCOMPLETE. Add unprivileged subprocess parity tests with restrictive umasks.

3. **Medium — direct writes lose hard-link alias effects without declaring incomplete evaluation.**  
   [operation_files.go:838](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:838)

   Direct writes stage a replacement node keyed only by resolved pathname. They do not preserve the original file’s inode identity.

   Reproducible scenario: hard-link `.pose/rules/security.md` to another readable file, optionally outside the instance, with content differing from the shipped rule. Native machinery delivery overwrites the shared inode, changing both names. Preview changes only the managed pathname; reading the alias returns the original bytes, and the effect report omits the alias impact.

   Atomic publication should break that relationship; direct overwrite should preserve it. The distinction is part of this replacement’s contract.

   **Smallest cohesive fix:** detect multiply linked physical files before direct overwrite and report INCOMPLETE unless inode alias semantics are supported. This avoids requiring an exhaustive filesystem search. Cover direct overwrite versus atomic replacement with a hard-link fixture.

**Composition assessment**

- The shared update algorithm, explicit invocation-local Files and captured clock are present. Plain update and force installation use the same execution path as preview.
- Current-tree governance readers inspected across Stores, resolvers, configuration requests, index generation and final checks carry the projected view. Request reporting includes individual semantics rather than only counts.
- B28 rule, workflow, skill and suggest readers consume explicit Files. The force composition regression checks staged instruction availability.
- B30 preserves pinned-generation identity, latest-generation comparison, projected input bindings, result parsing/provenance comparison and explicit-attribution history reads.
- B38 keeps current-tree candidate validation separate from immutable tagged snapshots. Tagged version decoding uses committed blob bytes in memory. The regular-blob/type guard, source-policy agreement, declaration validation, byte limits, text/JSON decoding and semantic-version comparisons remain. Broken newer tags and unreadable inventory remain fail-closed in the inspected consumers.
- Native direct, exclusive and atomic publication remain distinct. Preview bypasses physical publisher locks and temporary publication. Usage suppression is invocation-local; onboarding creation uses Files.
- I found no additional concrete physical preview mutator in the inspected reachable paths. This conclusion does not eliminate the fidelity blockers above.

**Low risks and evidence limits**

- **Low — macOS umask assumption:** [operation_files_access.go:33](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files_access.go:33) reads Linux `/proc/self/status`, then defaults to `022`. On Darwin with umask `077`, projected creation modes differ from native modes. The six-platform build evidence establishes compilation, not runtime parity. Expose unavailable mode knowledge rather than silently treating the default as observed.
- Future ACL/security-policy changes, disk/quota exhaustion, durability failures, provider failures and concurrent changes remain runtime limitations. The preview text states these explicitly; I do not classify hypothetical later failures as current defects.
- The documented running-engine/no-self boundary is intentional and acceptable.
- The inotify audit is filesystem-event evidence, not syscall evidence. Its watched fixtures do not prove every possible filesystem layout.

**Rules applied during review**

Change type: mixed bugfix, structural refactor and documentation.

Applied `AGENTS.md`, `.agents/skills/pose-review/SKILL.md`, `.pose/workflows/review.md`, and:

- `backend-go.md`: interfaces, error propagation, synchronization and regression coverage.
- `security.md`: filesystem targets, confinement, publication integrity and sensitive output.
- `documentation-style.md`: contract wording and canonical references.
- `delivery-evidence.md`: distinction between implementation evidence, source review and delivery closure.
- `knowledge-governance.md` and `_base-recurrence.md`: residual-risk handling and recurring filesystem-fidelity failures.

Consulted the canonical B29 spec, prospective ADR, implementation report and historical design review. Historical copier reviews were treated as audit history. Automated ownership labels were not treated as human confirmation.

**Validation disposition**

Inspected the supplied full-suite, focused and full affected-package race logs; they record passing results. Vet and six-platform build logs are consistent with the implementation report. `git diff --check` passed during this review. No full suite was rerun.

Native context/bundle preparation, assessments, recurrence collection, validation, sealing, attestation and closure were deferred under the explicit read-only source-review instruction. No new bundle or plan digest was produced. No contributions, edits, staging or commits were made.

The concurrent native 146-check collection is outside this decision. **No final native result, sealed approval or spec closure is claimed.** Resolve the three blockers, add focused parity coverage, then perform fresh sealed adjudication.