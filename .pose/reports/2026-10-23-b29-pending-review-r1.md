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