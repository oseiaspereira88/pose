# B30 pending-work review R1

Actual separate execution, same vendor/model family as lead. Task-designated gpt-6.1-sol; runtime identity only as reported. Claude unavailable due weekly limit. Session 01a1622b-9492-7c72-a368-f45143d60919; log /home/go/.local/state/secondary-agent/20261022T025305-codex-176332.

**Decision: changes requested — three high-severity code findings remain at `0bb312b45578102243d7a16fa83fbd277a7a3c61`.** Mechanical gates remain pending separately.

Reviewer identity: this session is OpenAI Codex, based on GPT-6, not Claude Opus 5.5. No human or cryptographic independence is claimed.

1. **High — Unpinned reads can still mix graph A with freshness/generation B.**
   Source: [spec_facts.go:251](/tmp/pose-pending-b30/pose-mcp/internal/pose/spec_facts.go:251), especially the comparison at line 260.

   **Reproducer:** Let `GetDeliveryIntegrity` finish reading generation A, then publish B before `specFactsFreshness` reads the generation. Both subsequent generation reads see B, so the new comparison succeeds. With unchanged bound inputs/results, facts can retain A’s paths and associated checks while reporting B’s generation and `fresh`.

   **Required fix:** Pin the generation before reading the graph and derive identity/freshness from that same view, or retain and compare the graph’s actual generation identity. Add a controlled publication-between-reads regression. The existing pinned-store test does not cover this unpinned interleaving. This leaves r4 F05 incompletely fixed; R1/R3/R7 remain affected.

2. **High — Explicit-attribution freshness checks miss valid JSON and deleted records.**
   Source: [spec_facts.go:594](/tmp/pose-pending-b30/pose-mcp/internal/pose/spec_facts.go:594), with the freshness decision at [line 324](/tmp/pose-pending-b30/pose-mcp/internal/pose/spec_facts.go:324).

   **Reproducer:** Starting from fresh facts, add a valid history record whose change set uses `"spec" : "alpha"` or `"spec":"al\u0070ha"`. The canonical loader decodes JSON, but `hasExplicitAttribution` recognizes neither spelling. Alternatively, index an explicit change set without trailer attribution, then delete its history record: the current scan finds nothing, while the reverse-loss check examines only `trailers:` selectors. Old explicit attribution can consequently remain `fresh`.

   **Required fix:** Decode records using shared attribution semantics and account for both current records and explicit attribution retained by the graph. Missing, changed, unreadable, or unbound attribution must conservatively lower freshness. Cover whitespace, JSON escapes, replacement, and deletion. This leaves r4 F04 incompletely fixed; R2/R3/R7 remain affected.

3. **High — “Well formed” paths can still describe an impossible Git delta and associate an unrelated module.**
   Source: [spec_facts.go:623](/tmp/pose-pending-b30/pose-mcp/internal/pose/spec_facts.go:623) and [line 549](/tmp/pose-pending-b30/pose-mcp/internal/pose/spec_facts.go:549).

   **Reproducer:** Supply a recorded change set with populated base/head/diff/commits and `{action:"modified", path:"api/../../web/file.go"}`, no declared components, and an `api` result carrying the matching scoped digest. The structural predicate accepts the path; the coverage check accepts its literal `api/` prefix. The result becomes associated although the path escapes the project and cannot be a Git-reported filename.

   **Required fix:** Validate canonical, project-relative observed paths—including both rename endpoints—before considering a change set resolved or using it for coverage. Preserve meaningful spaces and literal `root` filenames. Invalid foundations must remain candidates. Add traversal/internal-dot-segment regressions. This is an adjacent gap in r4 F02’s structural validation, affecting R1/R2/R7.

I inspected the full production diff against `origin/main`, R1–R8, all four historical reports, post-`ce83fbd` fixes, regression tests, and mutation definitions. The direct literal-path and action-injection fixes appear sound; result identity, source/severity preservation, and conservative reuse labeling are retained.

Rules applied: `pose-review`/review workflow; backend Go, security, documentation style, delivery evidence, knowledge governance, and recurrence traceability. No unrelated production expansion or new dependency was identified.

**Limitations:** These are source-derived reproductions, not executed Go probes. Only the JSON recognition counterexample was demonstrated in memory. No suite, assessments that write artifacts, seal, attestation, or lifecycle operation was run. Context resolved to the local in-progress spec; the live review-plan reported unavailable index evidence, consistent with collection still pending. Fresh strict validation, artifact/index/surface evidence, and immutable bundle adjudication remain outstanding. No files or Git refs were modified by this review.

# Final sealed adjudication — 2026-10-22

Actual new separate session 01a16343-76a5-7340-95c5-edaf8747ad8c; log /home/go/.local/state/secondary-agent/20261022T075847-codex-1091268. Original R1 changes-requested text above is unchanged.

**Decision: approved** for the exact sealed subject [rvb-caff3ed965d9226d](/tmp/pose-pending-b30/.pose/review-bundles/rvb-caff3ed965d9226d.json), at HEAD `6c643a5d0f9093e1fe598f99426f5d5c496c003a`.

Bundle digest: `sha256:caff3ed965d9226dbf16f7353e1f00a3312b8874c2604c01ce10f1f9b7a7c864`

Plan digest: `sha256:b06d9b4f6e01080b02894d037858857f6174aeda5ab97e441c88e5fdb7f5cb56`

This approval does **not** apply to intermediate bundle `rvb-c97b9ab09eedf825`. Historical r4 and R1 changes-requested decisions remain valid historical records.

I inspected the complete production-source diff against `origin/main`, regression implementations, supplied execution logs, canonical validation results, final surface report, and sealed payload. Independent recomputation matched the payload, patch, tree, and implementation digests. All 17 subject entries match HEAD and current filesystem bytes; consumed file hashes also match. Native read-only bundle preparation reproduced the exact sealed digest and its 140 evidence entries.

The three R1 high findings are **fixed**:

| Finding | Disposition and semantic basis |
|---|---|
| Generation identity borrowed across publication | Fixed. `DeriveSpecFacts` pins before graph access. Freshness retains the pinned generation identity and detects subsequent publication. The controlled second-read publication regression exercises the previously missed interleaving. |
| Explicit attribution missed by textual matching or deletion | Fixed. JSON decoding recognizes whitespace and escaped spellings. Retained non-trailer graph change sets independently prevent `fresh` after record loss. Unreadable or unbound attribution conservatively lowers freshness. |
| Impossible observed paths associate unrelated modules | Fixed. Canonical relative path validation rejects traversal, internal dot segments, repeated/trailing separators, absolute names and NUL, including rename endpoints. Meaningful spaces and literal `root` filenames remain literal. |

The before-fix log demonstrates all three failures; the targeted after-fix log passes. These are lead-executed tests whose source and output I inspected.

| Criterion ID | Judgment | Reasoning |
|---|---|---|
| `assumption-integrity` | **passed** | Association, currency and semantic adequacy remain distinct. Missing bindings and unbound history are disclosed; reuse and reviewer identity are not promoted into stronger assurance. |
| `backend-concurrency` | **passed** | Pinning precedes graph access, existing pins are retained, and publication cannot lend a newer identity to older facts. Controlled interleaving and recorded race evidence support this. |
| `backend-contracts` | **passed** | Shared result parsing preserves schema-1 projection semantics. Shared trailer parsing retains existing attribution behavior. No MCP, REST or event contract is added. |
| `backend-input-errors` | **passed** | Incomplete change sets, invalid paths, omitted modules and mismatched digests cannot establish association. Read and parsing failures lower confidence or return errors. |
| `backend-integration-impact` | **passed** | CLI text/JSON consume the same facts structure; index loading uses the shared result parser. Final composition evidence connects the declared capability to its production entrypoint. |
| `backend-observability` | **passed** | Facts expose source, severity, provenance, head comparison and limitations. Text escapes control characters, including actions, preventing physical evidence-line injection. |
| `compatibility` | **passed** | Existing fields remain; candidates, binding and freshness are additive. Narrowing `checks` is an intentional documented semantic correction, explicitly classified minor in the changelog. Consumers treating unscoped checks as proof must adapt. |
| `correctness` | **passed** | Association requires matching scoped provenance, applicable module coverage and complete resolved foundations. Structural result identity preserves distinct outcomes and sources. The R1 fixes address their causes. |
| `design-causality` | **passed** | D1 and the existing delivery-integrity/generation ADRs support explicit witnesses and conservative uncertainty. Pinning closes the temporal mismatch; semantic JSON decoding closes representation mismatch; literal path validation closes coverage confusion. Module coincidence or freshness alone cannot substitute for evidence. |
| `documentation` | **passed** | EN/pt-BR and embedded manuals describe candidate separation and read-only behavior. The spec retains historical decisions and all six low follow-ups. Generic freshness wording remains explicitly deferred. |
| `operability` | **passed** | Limitations give actionable remediation without rebuilding during reads or treating rebuild as semantic approval. Missing/corrupt indexes fail closed. |
| `requirements` | **passed** | R1–R2: association foundations; R3: freshness; R4: original-source reuse disclosure; R5: facts without judgment; R6: text/JSON parity; R7: negative regressions and controls; R8: read-only queries without mandatory rebuild. No unmet R1–R8 identified. |
| `scope` | **passed** | Changes remain within facts projection, parsing reuse, rendering, regression coverage and documentation. The contaminated commit range is disclosed; attributed paths define the sealed subject. |
| `security` | **passed** | Impossible paths cannot establish coverage; external text cannot inject evidence lines. No new dependency, credential, authority grant, policy adoption or automatic approval path appears. |
| `solution-proportionality` | **passed** | Reusing existing generation, graph and provenance contracts is sufficient. Rebuilding on each query or introducing a new snapshot framework would add unnecessary scope. This choice needs reconsideration if requirements demand atomic observation across Git, history and result-file mutations. |

The cheapest false-evidence shortcut is a matching scope label/digest combined with an incomplete foundation or misleading module path. Resolution and coverage guards now refuse it, with adversarial regressions in the binding test files. The named adversarial corpus has no dedicated B30 row; those direct tests provide the relevant enforcement. Facts still do not prove semantic adequacy.

All six historical r4 low follow-ups are acceptable as **owned deferred work**, without an unmet requirement:

| Follow-up | Disposition |
|---|---|
| L01 component-reference explanations | Retain; conservative candidate explanations do not fabricate association. Owner `@core`. |
| L02 unavailable projection versus returned error | Retain; current behavior fails closed. Owner `@core`. |
| L03 generic unknown explanation | Retain; status and specific reason remain conservative. Owner `@pose-maintainers`. |
| L04 complete structural ordering tie-break | Retain; distinct results remain preserved despite incomplete tie ordering. Owner `@pose-maintainers`. |
| L05 bounded trailer-query output | Retain; resource-bounding improvement remains necessary. Owner `@pose-maintainers`. |
| L06 run identity with zero classified checks | Retain; freshness covers the retained result projection and does not establish delivery from zero checks. Owner `@pose-maintainers`. |

Each remains open in Final Report with review date **2026-12-01**. No new blocking finding was identified.

Tool dispositions:

- **Required active evidence:** artifact-check has zero errors; completed final surface JSON has zero errors and no B30 findings. Canonical strict validation records **143 executed, 143 passed, zero skipped/errors**, at the reviewed HEAD and scoped digest `sha256:6a963c32f8261b5661274a86b48254b62fb15199e63831409fa550ad607394de`.
- **Recommended tools:** design assessment is observed and bound to the implementation digest; integration assessment records 63 repository-wide consumer gaps, without a new facts MCP contract. Discovery evidence predates the final fixes and is contextual evidence. Suggest-review output was inspected.
- **Tech debt:** `/tmp/pose-b30-tech-debt.log` is absent. Stored assessment and reconciliation report disclose one covered marker and an **incomplete** scan caused by two legacy invalid sources. This is not a clean pass or debt-free claim.
- **Other supplied checks:** strict check passes with 29 existing warnings; history and skills checks pass; recurrence reports zero flagged/flapping keys. Recorded race output passes for `internal/pose`, `internal/cli` and `internal/mcpserver`.
- **Completion-only:** native review verification/check, closeout-check and close remain pending parent attribution/attestation. No pass is claimed for them.

Rules applied: `pose-review`, review workflow, backend Go, security, documentation style, delivery evidence, knowledge governance and recurrence traceability. The recurring defects concern evidence-contract implementation and insufficient negative validation; the preventive action is the retained regression coverage.

My actual commands were read-only Git/source/log inspections, in-memory JSON/hash reconciliation, `pose context`, `pose index --verify`, and `pose review bundle … --explain`. Index verification passed; bundle preparation reproduced the sealed identity. `git diff --check` reported three Markdown hard-break trailing spaces in the historical R1 report. I ran no tests or assessments that persist artifacts and made no writes, attestation, lifecycle or Git changes.

Reviewer identity is **OpenAI Codex**, task-designated `gpt-6.1-sol`, in this separate same-vendor review session. I cannot independently authenticate the exact runtime model or cryptographic independence. Claude unavailability is user-reported; no cross-vendor review is claimed.

Collector provenance note: Claude unavailability was observed by the lead launcher (weekly-limit result in 20261022T025242-claude-171182), rather than reported by the human user. The reviewer correctly makes no cross-vendor assurance claim.
