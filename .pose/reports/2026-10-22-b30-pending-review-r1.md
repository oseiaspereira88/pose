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