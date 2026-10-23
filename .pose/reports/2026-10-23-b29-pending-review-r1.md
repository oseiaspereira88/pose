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

## Separate source review R2 — actual response

**Source decision: changes requested.** All original R1 findings are addressed in source. Two additional reachable filesystem mismatches remain.

Reviewed `fe765cf8aaaaff3ec93e91818e69e0781f92116a..e7b768e619a044780f66349985ca1f3d272478c6`, including the corrections after `7d9c07c`, canonical B29 spec, accepted ADR, implementation report and actual historical R1. This was a separate, read-only Codex execution; no child agents were used.

**Concrete findings**

1. **Medium — direct writes fabricate regular-file behavior for special files.**  
   [operation_files.go:897](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:897)

   Physical lookup classifies every entry other than a directory or symlink as `previewFile`. Direct overwrite checks permissions and link count, then stages ordinary bytes without checking whether the physical target is regular.

   **Source-derived scenario:** on Linux, `.pose/rules/security.md` points to `/dev/null`. The reachable [machinery writer](/tmp/pose-pending-b29/pose-mcp/internal/cli/machinery.go:277) reads empty bytes and attempts delivery. With the usual writable, singly linked `/dev/null`, preview stages the shipped content, subsequent reads return it, and the effect report describes an external modification without recording a gap. Native direct writes discard those bytes; subsequent reads remain empty and the target remains a character device.

   This concerns an observable file type, not future provider or disk uncertainty.

   **Smallest fix:** before staging a direct overwrite of a physical entry, require `e.info.Mode().IsRegular()`. For unsupported special-file semantics, persist an `ErrPreviewUnsupported` gap and return INCOMPLETE. Preserve atomic replacement’s separate behavior. Add operation-level and shared-update reachability coverage.

2. **Medium — inability to enumerate a directory is incorrectly treated as proof that removal would fail.**  
   [operation_files.go:1068](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:1068)

   `Remove` enumerates a directory to establish emptiness. When enumeration fails, it records the read error as the native removal refusal without recording incomplete evaluation. Native removal of an empty directory requires appropriate access to its parent, rather than read permission on the directory being removed.

   **Source-derived scenario:** as an unprivileged user, replace `.claude/skills/pose-feature` with an empty directory of mode `000`, retaining a writable/searchable parent. During force update, [skill-link recreation](/tmp/pose-pending-b29/pose-mcp/internal/cli/install.go:455) removes that entry before creating its symlink. Native removal can succeed. Preview’s enumeration fails, leaves the directory present and makes symlink creation fail with existence. The internal enumeration bypasses `updateReadView`, so this path adds no evaluation gap and can finish with a complete change summary.

   **Smallest fix:** when directory emptiness cannot safely be established, report unsupported evaluation with a persistent gap instead of asserting a native removal refusal. Add an unprivileged empty-unreadable-directory control through force update. Check the analogous `RemoveAll` preflight while making this correction.

Neither finding was reproduced during this execution; both follow from the inspected source and reachable callers.

**Disposition of every R1 item**

| Historical item | Current disposition |
|---|---|
| Medium: absolute symlink targets cleaned before traversal | Addressed in source. Raw incoming and target components survive; preceding components are traversed before `..`. Regression coverage includes absent and symlink components, plus shared-update reachability. |
| Medium: staged permission bypasses | Addressed in source. Read, open, enumeration, search and write paths use staged access checks. Missing effective-access knowledge persists an INCOMPLETE gap. Physical-origin access remains kernel-backed. |
| Medium: direct hard-link alias loss | Addressed in source. Unknown or multiple physical link counts prevent optimistic direct overwrite. Atomic replacement severs the old identity independently. Operation and shared-update regressions cover this distinction. |
| Low: guessed platform umask `022` | Addressed in source. Observation availability is explicit; unavailable knowledge makes affected creation incomplete. Explicit-mode atomic publication remains independent when its parent exists. |

The additional trailing-separator correction also addresses the reported direct-path and literal-symlink-target cases in source. Its regression includes native refusal controls. I did not rerun those controls.

**Composition and mutation assessment**

The cumulative diff retains explicit invocation-local Files, captured time and the shared plain/force update algorithm. Discovery walks the original source tree. Projected governance reaches resolver-created Stores, configuration requests, generation computation and the force final gate. Request output retains capability, origin, question, addressed role and recommendation.

B28 availability readers consume staged instructions. B30 retains generation pinning, latest-generation comparison, projected bindings, result provenance and explicit history reads. B38 retains immutable tagged authority, strict validation and fail-closed handling; tagged version decoding now consumes committed blob bytes in memory.

I found no additional reachable physical preview mutator in the inspected paths. Native filesystem operations and publisher behavior remain separate from preview staging. This is source analysis, not a universal mutation-proof claim.

**Rules applied during review**

Change type: mixed bugfix, structural refactor and documentation. Applied root `AGENTS.md`, `POSE.md`, `pose-review` and the review workflow, with:

- `backend-go.md`: interfaces, errors, synchronization and regression coverage.
- `security.md`: filesystem targets, publication integrity and sensitive output.
- `documentation-style.md`: canonical references and contract wording.
- `delivery-evidence.md`: source approval versus verified delivery.
- `knowledge-governance.md` and `_base-recurrence.md`: residual-risk handling and recurring filesystem-fidelity gaps.

No deeper applicable AGENTS file was found. Knowledge search found no additional B29-specific entry. Frontend and infrastructure rules were not applicable.

**Validation and limits**

No tests, builds, native assessments or mutation audits were executed here. Reviewed regression code and reported results are distinguished from independently executed evidence. Earlier full-suite/race/vet/six-build/audit results precede the trailing-separator correction; pending native collection and current race/vulnerability results are not final review proof.

The cumulative `git diff --check` reports three trailing-space Markdown hard breaks in historical R1. This is nonblocking and does not justify altering the preserved review.

Context/bundle preparation, assessment writes, recurrence collection, sealing, attestation, handoff creation and closure remain deferred under the explicit source-only instruction; no new bundle or plan digest was produced. No edits, staging, commits, contributions or publication occurred.

Future concurrency, disk/quota, ACL/security-policy and provider uncertainty remain disclosed limits. Platform builds establish compilation only. The two findings require conservative INCOMPLETE handling, not universal kernel emulation. No sealed approval or B29 closure is claimed.

## Separate source review R3 — actual response

**Decision: CHANGES REQUESTED**

Reviewer: Codex, task-designated `gpt-6.1-sol`, separate read-only execution. Same-vendor review; no cross-vendor, human, cryptographic or cognitively verified independence claimed.

Reviewed source: `3f03b36a0bdcb2b457cc899712c3952642bc0550`, cumulative baseline `fe765cf8aaaaff3ec93e91818e69e0781f92116a`.

**Medium blocker — sticky-directory restrictions are bypassed during projected removal.**

At [operation_files.go:1082](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:1082), `Remove` authorizes removal using only `writableDir(parent)`. That reaches [operation_files.go:864](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:864), whose physical access check tests directory write/search permission. It does not evaluate sticky-bit restrictions on deleting someone else’s entry.

Concrete reachable layout: run unprivileged with `.claude/skills` pointing to a readable, writable/searchable sticky directory owned by another user, containing that user’s `pose-feature` entry. During force update, [install.go:455](/tmp/pose-pending-b29/pose-mcp/internal/cli/install.go:455) removes the entry before recreating its link. Native removal fails under sticky-directory ownership rules. Preview passes the parent access check, stages removal and successfully stages the replacement symlink, without an INCOMPLETE gap. It consequently reports different effects and skill-link diagnostics.

This is a source-derived existing-layout mismatch, not a reproduced test result or hypothetical future permission change.

**Smallest fix:** before projecting removal or replacement of an existing physical entry, detect sticky-parent restrictions. Where effective ownership/privilege cannot safely establish the result, persist `ErrPreviewUnsupported` and INCOMPLETE. Apply the same preflight to `RemoveAll` and atomic replacement. Add operation and shared force-update controls with an unprivileged caller and foreign-owned sticky-directory entries. No universal kernel emulation is required.

Historical findings inspected:

| Item | Disposition |
|---|---|
| R1 absolute-target traversal | Addressed: raw components survive until traversal. |
| R1 staged permissions | Addressed for the reported staged read/search/write cases; the distinct sticky-parent blocker remains. |
| R1 hard-link overwrite | Addressed: unsupported inode aliases persist a gap; atomic replacement remains separate. |
| R1 unavailable umask | Addressed: unavailable observation causes INCOMPLETE. |
| Parent trailing separator | Addressed: direct paths and literal link targets retain directory traversal requirements. |
| R2 special-device overwrite | Addressed: physical direct overwrite requires a regular file. |
| R2 unreadable directory removal | Addressed: unknown emptiness and recursive enumeration persist gaps. |

Applied root `AGENTS.md`, `POSE.md`, `pose-review`, the review workflow, and `backend-go`, security, documentation-style, delivery-evidence and knowledge-governance rules. The canonical spec and accepted D2 ADR require faithful shared execution or explicit incomplete evaluation; the blocker violates that requirement.

Per your immediate-return instruction, this is a **source-only rejection**. The readiness file was absent at the last inspection. Actual sealed bundle/plan digests, the plan’s 15 criterion judgments, structural `SD` mappings, native-output reconciliation and final tool dispositions remain **pending**; I will not invent their IDs or approve an absent subject. Reported validation success does not resolve this uncovered layout.

No writes, tests, native preparation commands, attestations, contributions or child agents were performed. Fresh sealed adjudication is required after correction, and current PR CI must be green before merge.

## Separate source review R4 — actual response

**Decision: CHANGES REQUESTED**

Reviewer: Codex, task-designated `gpt-6.1-sol`, separate read-only R4 execution. Same-vendor review; no cross-vendor, human, cryptographic or cognitively verified independence claimed.

Source: `24e3328d851f9714459ad27eee05c2f248304aa0`  
Cumulative baseline: `fe765cf8aaaaff3ec93e91818e69e0781f92116a`

**Medium blocker — net-change reporting bypasses the special-file read guard.**

[operation_files.go:1392](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:1392) calls `os.ReadFile(location)` directly when comparing a staged regular file with its original entry. It excludes original directories and symlinks, but does not exclude FIFOs or devices.

Concrete reachable scenario: an otherwise usable instance has a FIFO at `.pose/schema-version`, with no writer.

1. The initial [schema read](/tmp/pose-pending-b29/pose-mcp/internal/cli/maintenance.go:146) correctly rejects the special file and records an INCOMPLETE gap.
2. Execution continues with schema zero and [stages atomic replacement](/tmp/pose-pending-b29/pose-mcp/internal/cli/maintenance.go:178). `Publish` permits replacing this entry.
3. [Preview rendering calls `Changes`](/tmp/pose-pending-b29/pose-mcp/internal/cli/update_dryrun.go:55), whose `netChange` opens the original FIFO and blocks before displaying the INCOMPLETE result. With a writer, this read can consume FIFO data.

This is a source-derived reachable existing-layout defect, not an executed reproduction or a hypothetical later provider failure. The earlier special-read guard does not protect this reporting path.

**Smallest fix:** detect an original nonregular entry before the raw comparison read. Describe its replacement without reading its contents, or retain explicit INCOMPLETE evaluation where classification is unsupported. Add operation-level `Publish → Changes` and shared-update FIFO regressions, with bounded subprocess execution proving reporting terminates without opening/consuming the FIFO. Keep native apply unchanged.

Historical finding dispositions inspected in current source:

| Item | Disposition |
|---|---|
| R1 absolute-target traversal | Addressed: raw components survive traversal. |
| R1 staged permissions | Addressed: staged access checks preserve unsupported knowledge. |
| R1 hard-link overwrite | Addressed: unknown/multiple inode aliases persist INCOMPLETE. |
| R1 unavailable umask | Addressed: unavailable observation is explicit. |
| Parent trailing separator | Addressed: direct paths and literal targets retain directory requirements. |
| R2 special-device overwrite | Addressed in direct-write staging. The distinct reporting read bypass above remains. |
| R2 unreadable directory removal | Addressed: unknown emptiness/enumeration persists INCOMPLETE. |
| R3 sticky-directory authority | Addressed conservatively for removal, recursive children and atomic replacement. Owned-entry controls are not represented as foreign-owned reproductions. |

Rules applied: actual root `AGENTS.md`, `pose-review` skill and review workflow; security, delivery-evidence, documentation-style and knowledge-governance requirements, together with the canonical B29 spec and accepted D2 ADR. The blocker violates the requirement to retain unsupported special-file knowledge without performing ordinary-content reads.

Per the immediate-return instruction, this is a **source-only rejection**. The readiness file was absent at the last inspection. Actual sealed bundle/plan digests, the plan’s 15 criterion judgments, every structural SD mapping, payload provenance, artifact reconciliation, surface gate and plan-selected tool dispositions remain **pending**. No IDs or sealed judgments are inferred.

HEAD remained `24e3328`; the production subtree had no worktree differences. No tests, native preparation commands, writes, staging, commits, contributions, attestations or child agents were performed. Reported 146/146 validation does not resolve this uncovered path. Fresh sealed adjudication is required after correction; current PR CI must also be green before merge.

## Separate source review R5 — actual response

**Decision: CHANGES REQUESTED**

Reviewer: Codex, task-designated `gpt-6.1-sol`, separate read-only R5 execution in this conversation. Same-vendor review; no cross-vendor, human, cryptographic or cognitively verified independence claimed.

Source: `3570008dec67597a74e46079eb4d9e262255a596`  
Cumulative baseline: `fe765cf8aaaaff3ec93e91818e69e0781f92116a`

**Medium — recursive removal preflight hides native partial deletion without marking INCOMPLETE.**

[operation_files.go:1183](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:1183) preflights the entire tree before staging any removal. When a readable, nonempty nested directory lacks write permission, [operation_files.go:1201](/tmp/pose-pending-b29/pose-mcp/internal/pose/operation_files.go:1201) returns the access refusal without recording an evaluation gap.

Concrete reachable scenario, running unprivileged:

- An old index-generation directory qualifies for retention removal: beyond the two retained previous generations and older than ten minutes.
- Its parent and the generation directory are writable/searchable.
- It contains an ordinary removable file and a readable/searchable `0555` subdirectory containing another file.

Native `RemoveAll` removes the ordinary file while failing to empty the protected subdirectory. Preview preflight returns the nested permission error and stages **no deletions**. The reachable caller is [index_generation.go:839](/tmp/pose-pending-b29/pose-mcp/internal/pose/index_generation.go:839), reached through force update’s index publication and retention.

Because this error adds no gap, preview can report its other changes with a complete path-count verdict, omitting the native partial deletion. This violates R1 and D2’s explicit partial-effect reporting contract. It concerns existing permissions and contents, not hypothetical later concurrency, disk or provider failure.

**Smallest fix:** persist INCOMPLETE when recursive preflight encounters a refusal whose native partial effects are not modeled. Exact recursive deletion emulation is unnecessary. Add an unprivileged operation regression and an actual force-update retention regression, with native controls demonstrating the removable sibling disappears while the protected child remains.

This finding is source-derived; I did not execute a reproduction under the read-only constraint.

Earlier finding dispositions:

| Item | Disposition |
|---|---|
| R1 absolute symlink target traversal | Addressed: preserve components until traversal. |
| R1 staged permissions | Addressed for the reported read/search/write cases; unsupported access persists gaps. |
| R1 hard-link alias effects | Addressed: unknown/multiple links prevent optimistic direct overwrite. |
| R1 unavailable umask | Addressed: unavailable observation remains INCOMPLETE. |
| Parent trailing separator | Addressed for direct paths and literal link targets. |
| R2 special-device overwrite | Addressed: physical direct writes require qualified regular storage. |
| R2 unreadable directory removal/enumeration | Addressed: unknown emptiness/contents persist gaps. The distinct partial-removal defect above remains. |
| R3 sticky-parent authority | Addressed conservatively for removal and replacement, including recursive children. |
| Provider/default ACL/special-read gaps | Addressed with explicit unsupported knowledge. |
| R4 FIFO reporting hang | Addressed: metadata classifies nonregular originals before content comparison; replacement verbs include pt-BR. |

Rules applied: root `AGENTS.md`, `POSE.md`, `pose-review`, review workflow, `backend-go`, security, documentation-style, delivery-evidence, knowledge-governance and recurrence traceability. Change type: mixed bugfix, structural refactor and documentation. The canonical B29 spec, accepted ADR, implementation report, actual historical R1–R4 responses and current closeout report informed this review.

Implementation HEAD remained `3570008`; no production-source working-tree differences were observed. No tests, assessments, sealing, attestation, writes, child agents or publication were performed. Supplied validation claims were not independently executed here.

Per the immediate-return instruction for medium source blockers, this is a **source-only rejection**. The readiness file was absent at last inspection. No actual sealed bundle or plan digest was available; I therefore provide no invented 15-row criterion table, SD mappings or plan-selected tool dispositions. Sealed adjudication, native-output verification, artifact reconciliation and surface-gate review remain pending. Any eventual approval must also require current PR CI green before merge.

## Separate R6 source decision before readiness — actual response

**Decision: APPROVED — source-only. Final sealed adjudication remains pending.**

Reviewer: Codex, task-designated `gpt-6.1-sol`, separate read-only R6 session `01a168c9-5c9b-76f1-9449-ce5c3ab3544c`. Same-vendor execution; no cross-vendor, human, cryptographic or cognitively verified independence claimed.

Reviewed source: `a7af88ff481088ba63f7a8aeeed572f55e6715a2`, cumulative baseline `fe765cf8aaaaff3ec93e91818e69e0781f92116a`, including the delta from `3570008dec67597a74e46079eb4d9e262255a596` and its reachable composition. HEAD remained unchanged, with no production worktree differences at the final check.

The readiness wait limit was reached. `/tmp/pose-b29-sealed-a7af88f.json` remained absent at **2026-10-23 13:08:54 UTC**. Consequently, no sealed payload or sealed bundle/plan digest has been adjudicated. **Do not record this source-only decision as final sealed criterion approval.** The effective unsealed plan contains 15 required criteria; their final sealed judgments and structural mapping remain pending.

**Source conclusion**

No new medium-or-higher blocker was found. R5’s correction persists `INCOMPLETE` for recursive preflight failures and directory-parent refusals whose native partial effects are unmodeled. Those gaps survive tolerated retention errors and control the final verdict. Native removal remains unchanged.

The cumulative implementation preserves invocation-local Files and clock, lazy projected reads, original-tree discovery, the force governance gate, individual request semantics, and the explicit running-engine/no-self boundary. B28 staged instruction availability, B30 pinned generation/facts/history behavior, and B38 immutable tagged authority remain coherent in the inspected composition. Preview publication avoids physical staging and locks; tagged version decoding consumes committed bytes in memory.

**Historical dispositions**

| Finding | Source disposition |
|---|---|
| R1 raw absolute symlink traversal | Addressed: components survive until traversal. |
| R1 staged read/search/write permissions | Addressed: access checks retain unsupported knowledge. |
| R1 hard-link direct overwrite | Addressed: unknown/multiple aliases persist `INCOMPLETE`; atomic replacement stays distinct. |
| R1 unavailable umask | Addressed: unavailable observation is explicit. |
| Parent trailing-separator correction | Addressed for direct paths and literal link targets. |
| R2 special-file direct overwrite | Addressed: ordinary storage semantics must be established. |
| R2 unreadable directory emptiness/enumeration | Addressed: uncertainty persists rather than asserting native refusal. |
| R3 sticky-parent replacement authority | Addressed conservatively, including recursive children and atomic replacement. Owned controls are not foreign-owned reproductions. |
| Provider/default-ACL/special-read corrections | Retained with explicit unsupported knowledge. |
| R4 FIFO reporting hang | Addressed: nonregular original types are classified before content comparison; unknown provider comparison remains incomplete. |
| R5 recursive partial deletion | Addressed, including nested-protection and read-only-parent cases. |

The R5 before-fix log records both operation failures and the reachable retention failure; the after-fix focused log passes. These are inspected execution records, not reproductions executed by this reviewer. Historical copier reviews and prior rejected replacement reviews remain historical rejections, not approvals. No new finding or accepted-low-risk disposition is introduced.

**Rules applied**

Applied actual root `AGENTS.md`, `POSE.md`, `pose-review`, `.pose/workflows/review.md`, and backend-go, security, documentation-style, delivery-evidence, knowledge-governance and recurrence rules. Consulted the canonical spec, accepted D2 ADR, implementation report, complete R1–R5 history and current closeout. No deeper applicable AGENTS file or additional B29 knowledge artifact was found. Frontend and infrastructure rules were inapplicable.

**Evidence and tool dispositions**

| Tool/check | Disposition |
|---|---|
| Required strict validation, global and module | Inspected current canonical collection: **146/146 checks pass**, zero failed, errored or skipped checks, at `a7af88f`. Required `update-preview-integration` passes with nonempty selectors. |
| Required artifact-check | Inspected current output: no error findings. Its 598 repository-wide orphan warnings remain disclosed. |
| Required surface-check | Inspected current output: no findings; declared composed capability and production entrypoint are represented. |
| Recommended suggest-review | Executed read-only and applied relevant rules. |
| Recommended assess-design | Executed read-only. Actual plan requires material `SD-cb046cda`, the added D2 ADR; final sealed mapping is pending. |
| Recommended assess-discover | Used supplied current refreshed component evidence. |
| Recommended assess-integrate | Inspected supplied assessment; heuristic unobserved consumers remain, so this is not a clean integration certificate. |
| Recommended assess-tech-debt | Inspected supplied **INCOMPLETE** assessment: two invalid legacy specs prevent a debt-free conclusion. |
| Recurrence | Inspected supplied output: no threshold-flagged keys; flapping warning retained. |
| Completion-only review-check/closeout-check | Deferred, together with attestation and closure. |

Current native evidence identifies:

- Global provenance: `sha256:e4148f97d261f9a363bbbb6c0624d5dddb01b1626ff96bc05489943bd646dbb7`
- B29 scoped provenance: `sha256:4eaca93a30f971ff4e84605039f1da80541eb2f5e1bf84bf1812b09d656282c5`

Current operation/update race output passes. Platform binaries were inspected; builds establish compilation, not macOS/Windows runtime parity. The inotify audit supplies fixture-specific filesystem-event evidence, not syscall or universal-layout proof.

Artifact reconciliation uses the explicit native range ending at `aebe238` for merge-composed files and later spec-trailed commits for subsequent changes. The earlier 14 mismatches were resolved by explicit range observation and removal of four untouched planned claims, without changing attribution semantics.

No dependencies changed. Earlier govulncheck reported none; no fresh scanner execution is fabricated here. Any eventual approval remains conditional on **current final PR CI green before merge**, including required vulnerability and secret checks.

No child agents, edits, staging, commits, native write commands, contributions, publication, attestation or closeout were performed.

## R6 final sealed adjudication — actual resumed response

**Decision: APPROVED**

This completes the pending sealed adjudication for `spec:pose-update-dry-run-reports-the-whole-update`. Approval is conditional on **current final PR CI green before merge**, including required vulnerability and secret checks. Those future checks are not claimed as passed.

Reviewer: Codex, task-designated `gpt-6.1-sol`, same read-only R6 session `01a168c9-5c9b-76f1-9449-ce5c3ab3544c`. This is separate, same-vendor execution under declared identity assurance. No cross-vendor, human, cryptographic or cognitively verified independence is claimed. Automated governance ownership is not human confirmation.

**Exact sealed subject**

Inspected [the immutable bundle](/tmp/pose-pending-b29/.pose/review-bundles/rvb-c59b0c2cd1a61af8.json), sealed `2026-10-23T14:13:06Z`.

| Binding | Exact value |
|---|---|
| Source HEAD | `a7af88ff481088ba63f7a8aeeed572f55e6715a2` |
| Cumulative review baseline | `fe765cf8aaaaff3ec93e91818e69e0781f92116a` |
| Bundle | `rvb-c59b0c2cd1a61af8` |
| Bundle digest | `sha256:c59b0c2cd1a61af8374f50cab417628cc27abf076f733436da242b92f1a45558` |
| Native sealed plan digest | `sha256:95afc73c3aaf27978723e46a762977b1a5856386bf7ac6c0abf09233e80c73d3` |
| Implementation digest | `sha256:8c6fd86f873eebe3551bcae8274ce8d6223833605be88d26cf2a7e4d3ea1e746` |
| Patch digest | `sha256:516190785c60b70c0522842362050d75cde6b2921c7f5d4fdce96683b19a18f3` |
| Tree digest | `sha256:4cad4e209085fe4862dce0711276b3b5c09100ffe262cddc5c6fbcb05a4de639` |

I recalculated the canonical payload, native bundle-plan contract, implementation and tree digests successfully. All **110 subject entries** match current file bytes. Current read-only native preparation resolves to the same bundle ID and digest without blockers. The source remains unchanged.

The exact bundle-file SHA-256 is `c8e3824a438dad9596a2751cc9b72383af11b21e14c75ac7a1653eb0986616ba`.

**Required criterion judgments**

All **15 actual sealed criteria** are required and receive an explicit judgment below. Mechanical conclusions use the inspected sealed evidence; judgment conclusions reflect source and design assessment.

| criterion-id | passed/finding/not-applicable | individual reason |
|---|---|---|
| assumption-integrity | passed | Observed filesystem facts are distinguished from unsupported permission, inheritance, provider and alias knowledge. Unsupported evaluation persists `INCOMPLETE`; future apply uncertainty is disclosed. |
| backend-concurrency | passed | Files, captured time and preview state are invocation-local; staged access is synchronized. Sealed unit/integration evidence passes, with supplementary current affected race results passing. |
| backend-contracts | passed | Shared execution preserves direct, exclusive and atomic publication distinctions and native apply behavior. Sealed update-preview and install/upgrade integration checks cover the affected file and CLI contracts. |
| backend-input-errors | passed | Known refusals, unsupported evaluation and completed no-ops remain distinct. Negative operation/update cases, including recursive partial removal and special-file handling, pass in the registered integration evidence. |
| backend-integration-impact | passed | Projected reads reach force governance, requests and indexes. Sealed distributed-instruction, spec-facts and release-manifest integrations qualify B28/B30/B38 composition; heuristic assessment limits remain disclosed. |
| backend-observability | passed | Output identifies individual requests, affected paths, external-link relationships, refusals and evaluation gaps. Sealed tests cover actionable diagnostics and distinct final verdicts. |
| compatibility | passed | Existing flags and native apply semantics remain. Preview explicitly describes the running-engine/no-self boundary and preserves individual request semantics. No unrelated CLI, MCP or authority contract is relaxed. |
| correctness | passed | Current sealed unit, integration and build evidence passes. The cumulative source preserves lazy staged reads, original discovery, force checks and nonmutation boundaries; R1–R5 corrections remain present. |
| design-causality | passed | D2 removes copier reconstruction and second-planner drift by sharing execution and projected reads. The required structural mapping below connects the ADR to R1/R2. |
| documentation | passed | The accepted ADR, canonical spec and reports describe implemented boundaries and evidence limits, while retaining rejected copier and review history as historical. |
| operability | passed | Operators receive explicit no-op, refusal, failure or incomplete outcomes, with relevant causes and the self-update boundary explained. Stale-instance warnings direct operators to `pose update`. |
| requirements | passed | Registered `update-preview-integration` covers R1 effects/stamps/requests, R2 complete-no-op versus incomplete/refused behavior, and R3 stale-engine warning contracts. Evidence is current at the sealed source. |
| scope | passed | Propagation through reachable readers/writers is necessary for D2’s whole-update contract. Artifact reconciliation covers that scope; no unrelated release, publication or adoption work is introduced. |
| security | passed | Preview stages effects in memory, preserves path/authority checks and avoids physical publisher staging/locks. Current govulncheck reports no vulnerabilities; final PR security CI remains mandatory before merge. |
| solution-proportionality | passed | Explicit Files propagation is broader than the rejected copier but necessary for coherent discovery and governance reads. Conservative unsupported outcomes avoid a general kernel emulator. New update phases introducing unmodeled effects require reassessment. |

**Required material structural mapping**

The sealed plan contains exactly **one** material fact requiring mapping.

| SD-id | Rn or Dn | causal reason |
|---|---|---|
| SD-cb046cda | D2 | The added ADR requires one apply/preview execution algorithm with lazy original inputs and staged subsequent reads across force installation, discovery, requests, indexes and governance. This directly addresses R1’s omitted/drifting effects and R2’s false no-op/completeness problem while preserving native apply and explicit unsupported knowledge. |

**Evidence and tool dispositions**

The recollected canonical results name `a7af88ff481088ba63f7a8aeeed572f55e6715a2`: **146/146 checks pass**, with zero failed, optional-failed, errored or skipped checks. All sealed native evidence references match the canonical result IDs, classes, outcomes, source and provenance.

- Global provenance: `sha256:a73f64c8b4ec25f5bff8fd686a9ec714d87375d8ef04f8464efeb719e93c2d7d`
- B29 scoped provenance: `sha256:f4478c958054c41f9eeb57858b11b6f457028c50e97bdc821f910f0a7da44c6e`

The registered `pose-mcp/go/update-preview-integration` executes existing nonempty selectors with `-count=1` and passes. Sealed full module test/build evidence and collected vet results also pass; some full-suite Go results use the Go cache.

| Tool | Disposition |
|---|---|
| Required `validate`, global and `pose-mcp` | Satisfied by the inspected current canonical collection covering the module. No redundant suite rerun. |
| Required `artifact-check` | Current aligned output: 114 claims, no errors; 598 repository-wide orphan warnings retained. |
| Required `surface-check` | Current aligned output: no findings; composed capability reaches `pose-mcp/cmd/pose/main.go` with current validation evidence. |
| Recommended `suggest-review` | Used read-only during this review; applicable rules applied. |
| Recommended `assess-design` | Used read-only; sealed structural observation and implementation binding verified. |
| Recommended `assess-discover` | Used supplied refreshed component evidence. |
| Recommended `assess-integrate` | Inspected current assessment: 64 contracts evaluated, with 63 heuristic unobserved-consumer gaps. It is not a clean integration certificate. |
| Recommended `assess-tech-debt` | Inspected current **INCOMPLETE** scan: one covered marker, zero uncovered markers, but two invalid legacy specs prevent a debt-free conclusion. |
| Required completion-only `review-check`, `closeout-check` | Deferred until the parent records the actual judgments. |
| Attestation, review verification and closure | Left to the parent; none performed here. |

The mechanical auto-attest preview applies nothing and leaves eight judgment criteria pending. The explicit judgments above answer those criteria; no approval is inferred from that preview.

Current govulncheck reports **“No vulnerabilities found.”** No dependencies changed. Current affected operation/update race results pass. Platform builds establish compilation, not macOS/Windows runtime parity. Inotify evidence covers filesystem events in exercised fixtures, including transient effects; it is not syscall tracing or universal-layout proof.

**Historical findings and retained warnings**

| Historical finding | Final disposition |
|---|---|
| R1 absolute symlink traversal | Resolved: preserve raw components until traversal. |
| R1 staged permissions | Resolved: enforce staged access checks and retain unsupported knowledge. |
| R1 hard-link overwrite | Resolved: unknown/multiple aliases prevent optimistic direct overwrite; atomic replacement remains distinct. |
| R1 unavailable umask | Resolved: unavailable observation produces incomplete evaluation. |
| Parent trailing-separator issue | Resolved for direct paths and literal symlink targets. |
| R2 special-file direct writes | Resolved: require qualified ordinary-file semantics. |
| R2 unreadable directory emptiness/enumeration | Resolved: unknown removal knowledge persists gaps. |
| R3 sticky-parent replacement | Resolved conservatively, including recursive children and atomic replacement. Owned controls remain explicitly distinct from foreign-owned reproductions. |
| Provider/default-ACL/special-read corrections | Retained with explicit unsupported outcomes. |
| R4 FIFO reporting | Resolved: classify nonregular originals before content comparison; unknown provider comparison remains incomplete. |
| R5 recursive partial deletion | Resolved for recursive preflight failures and directory-parent refusal paths. Unprivileged operation and force-retention controls demonstrate native partial effects. |

No new finding or accepted-low-risk disposition is introduced. Rejected copier reviews and earlier R1–R5 rejections remain historical records.

The native exact-SHA range `fe765cf…a7af88f` aligns the current subject header while preserving the older `aebe238` observation. The trailer range’s contamination warning remains: its attributed paths are the subject, not every intervening commit. Missing component owner metadata and unmapped governance-path warnings remain disclosed. The original artifact mismatches were resolved through explicit native observation and removal of untouched planned claims, without changing engine attribution semantics.

Technical-debt incompleteness, integration heuristics, repository orphan warnings and recurrence flapping are not represented as newly discovered B29 defects or as clean scans.

**Rules applied**

Applied actual `AGENTS.md`, `POSE.md`, `pose-review` and review workflow, plus backend-go for interfaces/errors/concurrency; security for filesystem and authority boundaries; documentation-style for canonical contracts; delivery-evidence for current gates and provenance; knowledge-governance and recurrence rules for historical/residual reporting. Frontend and infrastructure rules were inapplicable.

No source writes, native writes, staging, commits, child agents, contributions, attestation, closure or publication were performed.

## Fresh reconciled governance review — actual response

**Decision: APPROVED**

Fresh governance adjudication for `spec:pose-update-dry-run-reports-the-whole-update`, conditional on **final-head PR CI green before merge**, including required security checks. Those future checks are not claimed as passed.

Reviewer: Codex, task-designated `gpt-6.1-sol`, new separate read-only session `01a169a1-f9ba-74c2-b293-c25b0853559c`. Same-vendor review with declared identity assurance; no authenticated, cross-vendor, human or cognitively verified independence claimed.

Inspected the [fresh sealed bundle](/tmp/pose-pending-b29/.pose/review-bundles/rvb-08bd807629520cd3.json), sealed `2026-10-23T16:40:55Z`.

| Binding | Exact value |
|---|---|
| Bundle | `rvb-08bd807629520cd3` |
| Bundle digest | `sha256:08bd807629520cd39457ecec0f2f4c7430c05966fe63ee18dc1a7efce6647de3` |
| Native sealed plan digest | `sha256:95afc73c3aaf27978723e46a762977b1a5856386bf7ac6c0abf09233e80c73d3` |
| Implementation digest | `sha256:8c6fd86f873eebe3551bcae8274ce8d6223833605be88d26cf2a7e4d3ea1e746` |
| Patch digest | `sha256:516190785c60b70c0522842362050d75cde6b2921c7f5d4fdce96683b19a18f3` |
| Tree digest | `sha256:4cad4e209085fe4862dce0711276b3b5c09100ffe262cddc5c6fbcb05a4de639` |
| Source HEAD | `a7af88ff481088ba63f7a8aeeed572f55e6715a2` |
| Cumulative review baseline | `fe765cf8aaaaff3ec93e91818e69e0781f92116a` |
| Bundle-file SHA-256 | `73ae74b2c9f9666751ab79f01a2bcb059f5f293e3f7bf60c84bc4011c4b8b14b` |

I recalculated the bundle, native plan-contract, implementation, patch and tree digests successfully. Native read-only preparation resolves the same bundle without blockers. All 110 subject entries match current bytes, including all 106 implementation entries. All 143 evidence references reconcile with the canonical collection or structural observation. The fresh payload differs from R6 only in semantic scope and contract-node basis.

The governance reconciliation is acceptable:

- The original start receipt is byte-identical to HEAD and the R6 subject: SHA-256 `1de1f409ee3031c8941fe5bc6e6f40f3047119a18ec06ee190f366c6b7ae5295`. Its `2026-10-10T03:59:06Z` record still contains R1–R3, D1 and D1’s original missing-basis diagnostic.
- The first D2 amendment line is unchanged. Its original date, author and declared assurance remain historical.
- Both appended events carry `2026-10-23T16:23:50Z`, author `@agent-codex` and `assurance: declared`. Their rationale explicitly disclaims pre-execution recording, new requirements and human confirmation.
- The baseline acknowledges existing nodes. The transition records D1 from active/hash `060d57882685` to withdrawn/hash `fc51a644343d`, with retrospective relations R1/R2. D1’s original text and date remain intact.
- R1–R3 text remains unchanged, with hashes `8e94285e0bd8`, `18774fda39bb` and `14d4fce633cd`. D2 remains active and unchanged at `489c4bd160ee`, reaching R1/R2/R3.
- The reconstructed current node digest matches the seal: `sha256:2a287b7c93a1f19ce8adef186db44f22ea53a218d127bb9163088f9f5fd3ee16`.
- Strict lint records three amendment events and zero failures. Native closure remains `done`, `completed_at: 2026-10-23`.

A current baseline can satisfy ledger parsing without proving historical precedence. Here the immutable receipt, preserved diagnostic, explicit retrospective rationale and fresh judgments prevent that mechanical success from being represented as original human authorization. No new governance gate or implementation contract is introduced.

The documented clean-start limitation of the recoverable plan does not negate the separately evidenced manual gates and native locked close. The records support actual R6 approval, native verification and native closure; no manual status edit or skip-evidence substitution is indicated.

The retained closed approval is **not** the basis of this decision: its delta explicitly marks all 15 criteria changed. These are fresh judgments after verifying implementation identity and inspecting actual R6, R1–R5 history, implementation evidence and current governance.

| id | passed/finding/not-applicable | individual reason |
|---|---|---|
| assumption-integrity | passed | Current reconciliation is explicitly retrospective and declared. Original diagnostics survive; unsupported filesystem knowledge remains INCOMPLETE. |
| backend-concurrency | passed | Invocation-local Files, captured time and synchronized staging remain unchanged; current sealed tests and inspected race evidence qualify them. |
| backend-contracts | passed | Direct, exclusive and atomic operations retain their distinctions. Governance reconciliation changes no CLI, API, MCP or native apply contract. |
| backend-input-errors | passed | Refusal, unsupported evaluation and complete no-op remain distinct; current integration evidence includes special-file and partial-removal controls. |
| backend-integration-impact | passed | Unchanged projected readers cover force governance, requests and indexes; B28/B30/B38 integration evidence remains applicable, with heuristic limits retained. |
| backend-observability | passed | Request semantics, effects, refusals and evaluation gaps remain actionable. Reconciliation records actual timing, authorship and transition rationale. |
| compatibility | passed | Existing flags, native apply and running-engine/no-self boundary remain intact; historical records stay addressable. |
| correctness | passed | Unchanged source matches R6 exactly; current sealed checks pass, and the ledger transition agrees with canonical node state. |
| design-causality | passed | Active D2 addresses copier drift through shared execution and projected reads; withdrawn D1 supplies no current structural authority. |
| documentation | passed | Current spec and closeout distinguish historical rejection, actual closure and retrospective reconciliation. Original review responses remain historical. |
| operability | passed | Preview outcomes and remediation remain explicit; the reconciliation preserves an auditable explanation of post-close work. |
| requirements | passed | R1–R3 are unchanged and retain current registered integration evidence; no baseline event is treated as a new requirement. |
| scope | passed | Changes are governance reconciliation and generated evidence/state; there are no source, dependency or validation-matrix changes since R6. |
| security | passed | Authority assurance remains declared without fabricated confirmation; unchanged preview confinement has current evidence, and inspected govulncheck reports no vulnerabilities. |
| solution-proportionality | passed | Appending reconciliation and withdrawing D1 is sufficient while preserving history. D2’s bounded operation view avoids unnecessary kernel emulation; new unmodeled update effects require reassessment. |

The actual sealed plan contains one material structural fact:

| SD-id | Rn or Dn | causal reason |
|---|---|---|
| SD-cb046cda | D2 | The added ADR requires shared apply/preview execution with original discovery and staged subsequent reads across installation, requests, indexes and governance. This addresses R1’s omitted effects and R2’s false completeness while preserving native apply and explicit unsupported outcomes. D2 remains active with a valid requirement basis. |

Tool dispositions follow the actual sealed plan:

| Tool | Disposition |
|---|---|
| Required `validate`, global and `pose-mcp` | Inspected canonical 146/146 passing collection at unchanged `a7af88f`; zero failures, optional failures, errors or skips. No redundant rerun. |
| Required `artifact-check` | Inspected fresh reconciled output: 114 claims, zero errors; 598 repository orphan warnings retained. |
| Required `surface-check` | Inspected fresh reconciled output: zero findings; composed capability reaches `pose-mcp/cmd/pose/main.go`. |
| Recommended `suggest-review` | Executed read-only; relevant rules applied. |
| Recommended `assess-design` | Executed read-only; observed identity and material fact match the seal. |
| Recommended `assess-discover` | Inspected supplied post-close evidence, including reuse of unchanged `pose-mcp` assessment. |
| Recommended `assess-integrate` | Inspected supplied assessment: 64 contracts, 63 heuristic unobserved-consumer gaps; no clean integration certification. |
| Recommended `assess-tech-debt` | Inspected INCOMPLETE assessment: one covered marker, zero uncovered, two invalid legacy specs preventing complete coverage. |
| Required completion-only `review-check`, `closeout-check` | Deferred for fresh attestation consumption; previous closure outputs do not certify this new adjudication. |
| Context, bundle and plan preparation | Executed read-only; canonical authority resolves without ambiguity and current preparation matches the fresh seal. |
| Lint/design-basis check | Inspected supplied strict lint and executed read-only design check: zero errors; optional section/design-field advisories retained. |
| Recurrence | Inspected supplied evidence: zero threshold-flagged keys; flapping warning retained. |
| Attestation, contribution, handoff writes and publication | Not performed under the explicit read-only scope. No new accepted-risk finding requires a new handoff. |

Canonical evidence retains global provenance `sha256:a73f64c8b4ec25f5bff8fd686a9ec714d87375d8ef04f8464efeb719e93c2d7d` and B29 scoped provenance `sha256:f4478c958054c41f9eeb57858b11b6f457028c50e97bdc821f910f0a7da44c6e`. The immutable `fe765cf…a7af88f` range aligns the source while preserving the older `aebe238` record. Contaminated trailer-range, missing-owner and unmapped governance-path warnings remain disclosed.

Actual R1–R5 corrections remain present, including traversal, permissions, aliases, special files, sticky parents, FIFO reporting and recursive partial retention. Inspected race results pass. Six-platform builds establish compilation; inotify establishes fixture-specific filesystem-event evidence. Neither proves universal kernel behavior or macOS/Windows runtime parity. Unsupported behavior remains explicitly INCOMPLETE.

Applied root AGENTS, POSE, the review skill/workflow, backend-go, security, documentation-style, delivery-evidence, knowledge-governance and recurrence rules. Frontend and infrastructure rules are inapplicable.

No new finding or accepted-low risk is introduced. No tests were independently executed here, and no writes, staging, commits, child agents, attestation, closure or publication were performed. **Final-head PR CI green remains mandatory before merge.**