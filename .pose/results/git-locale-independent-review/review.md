# Independent review: Git repository detection and locale

Reviewed on 2026-10-26 at implementation commit `f5275457fcd143f6c5387ec54b6a63b4d8cb4411`.
Reviewer: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI, authorized by the maintainer.
Implementer: `agent:claude-opus-5-5`, Claude/Anthropic session `f2fa76f7-0c77-4cc7-a1d5-da4fa953d6dc`.
The `agent:independent-` prefix is a declaration required by the engine for different-actor review. This execution uses another vendor. Identity is declared: pose-dist attestations are unsigned and cognitive independence is not machine-observable.

## Rules applied during review

- Change type: bugfix. Consulted `.pose/workflows/review.md`, `POSE.md`, `pose-review` and `pose-spec-closeout` skills.
- `.pose/rules/backend-go.md`: subprocess environment, error classification and regression behavior.
- `.pose/rules/security.md`: fixed argv, no shell evaluation in production, process-local environment, no credentials in changes.
- `.pose/rules/documentation-style.md`: requirements, decision D1 and changelog accurately describe the implementation.
- `.pose/rules/delivery-evidence.md`: current deterministic evidence, artifact reconciliation and capability composition gates.
- `.pose/rules/knowledge-governance.md`: consulted prior knowledge; no new time-bound handoff required by the locale fix.
- Frontend, infrastructure and deployment rules are inapplicable; no related files or contracts changed.

## Independent falsification and deterministic checks

Temporarily restored only `release_lifecycle.go` and `release_manifest_integrity.go` from the parent commit, retained the new tests, then restored both production files byte-for-byte in a `finally` block. No production or test changes remain.

| Production | Test process locale | New regression tests | Two upstream tests |
|---|---|---|---|
| Parent implementation | C | both fail | both pass |
| Parent implementation | pt_BR.UTF-8 | both fail | both fail |
| f5275457 | C | both pass | both pass |
| f5275457 | pt_BR.UTF-8 | both pass | both pass |

The installed Git actually emits Portuguese diagnostics under pt_BR; this is independently exercised by the upstream tests, in addition to the deterministic test shim. Full `go test ./... -count=1` passes under both `LANG=C LC_ALL=C` and `LANG=pt_BR.UTF-8 LC_ALL=pt_BR.UTF-8`. `go vet ./...`, `GOOS=windows go build ./...` and `go test -race ./internal/pose -count=1` pass. Adjacent logs contain the observed results. A Windows cross-build does not claim Windows runtime execution; the shell-based locale tests explicitly skip there.

## Complete locale-sensitive reading inventory

Searched all production Go files under `pose-mcp/internal`, excluding tests, for subprocess construction, stdout/stderr capture, `CombinedOutput`, `err.Error()`, string predicates, equality checks and regex parsing. Traced wrappers and their consumers. `command-sites.txt` and `git-reader-sites.txt` record the subprocess and Git-wrapper call sites.

| Site | Reading and locale dependency | Assessment |
|---|---|---|
| `pose/release_lifecycle.go:678` | Interprets Git stderr containing English `not a git repository`. | Sole fixed human-language Git diagnostic predicate. Its probe now runs through `gitOutputUntranslated` with C locale. |
| `pose/review_run_continuity.go:125`, `:181`, `:215`, `:236`, `:599` | Configured session, usage-limit and reset regexes read adapter transcript/output. | Potentially language-dependent by explicit adapter configuration; the engine has no built-in vendor message vocabulary. Unchanged, outside R1/R2. |
| `cli/report.go:271` | Reads a supplied validation log and recognizes English/Portuguese result labels. | Explicitly supports both POSE output languages. This is POSE log parsing, not a Git diagnostic or another unhandled Git locale dependency. |

Related text reads that are **not locale-dependent**:

- `pose/release_manifest_integrity.go:293,335` tests stderr presence, not words; `:670` renders diagnostics. Errors remain uncertain rather than absence. Ordinary `gitOutput` still inherits the complete user environment.
- `pose/git_batch.go:75,79` recognizes `missing` and `blob` in Git's stable cat-file batch protocol. Other Git readers consume object IDs, type/mode tokens, porcelain status, NUL-separated paths, ISO timestamps, explicit log formats, counts, repository file bytes or configured values. These tokens are not translated messages. No natural-language Git log date is parsed.
- `pose/review_attempt_ledger.go:471` requires the explicit adapter protocol `decision: approved`, `changes-requested` or `rejected`. These are contract tokens specified in the reviewer brief, not translations inferred from prose; other text is unknown.
- `cli/extension.go:192` captures cosign combined output solely to display an error after a failed exit; no phrase determines verification success.
- `cli/maintenance.go:75` filters `404` and `no such file` in Go/HTTP self-update errors, not subprocess stderr. Go's generated error vocabulary is not localized by the process locale. `pose/federated_acceptance.go:832,835`, `pose/review_closeout.go:1058`, `pose/spec_transfer.go:276` and `cli/start.go:282` inspect engine-generated error vocabulary, not translated external diagnostics.
- Validation, native CLI gates, ssh-keygen signing, gh issue creation and other subprocess consumers use exit status, structured JSON, opaque signature bytes or returned URLs; output text is retained for diagnosis. No additional locale-dependent external-command classification was found.

D1 was also checked dynamically with a temporary external Git wrapper while running the two upstream tests under pt_BR. `d1-git-env.jsonl` records twelve Git invocations: four repository-detection probes received `LC_ALL=C`; eight tag, HEAD, status and top-level queries retained `LC_ALL=pt_BR.UTF-8`. The wrapper delegated to the real Git and changed no repository files. The production helper sets only the child environment and never changes `os.Environ` globally.

## Technical judgment

R1 is addressed at its cause: translated stderr is normalized at the one interpretation boundary. R2 follows through the existing `ErrReleaseTagMissing` classification and is exercised by both release consumers. The ancestor `.git` guard remains intact, so an unreadable or lost gitdir is not newly classified as repository absence. No public API, CLI, file or MCP shape changes. Normal Git diagnostic language remains available to the operator.

D1 is smaller in behavior than normalizing every Git child. The shared implementation preserves ordinary calls with `cmd.Env == nil`; only the dedicated helper supplies an appended override, which Go deduplicates with the last value winning. There is no speculative locale framework, dependency or concurrency change. A Git implementation ignoring C locale or changing the recognized diagnostic would falsify the design; none was observed.

No medium-or-higher defect or lower-severity actionable defect was found in the reviewed change. Scanner diagnostics are not silently treated as passing semantic evidence: tech-debt scanned 287 source files with one covered marker but could not parse two unrelated legacy specs; integration assessment reported 63 unobserved consumers, which means a static repository reference was absent, not a demonstrated broken integration. This fix changes no provider contract. Recurrence reported no active threshold cluster and one pre-existing validation flapping signal; the independent bilingual reproduction directly addresses the locale-specific validation blind spot without changing policy.


## Native review and closeout outcome

Decision: approved, with no actionable findings in this change. Bundle `rvb-e0eb83adbc8bd1a8` has digest `sha256:e0eb83adbc8bd1a86a8aa89ff7017f61769e6b68f371a49aa2175fa1c4543536`; plan digest is `sha256:fffedeb876171d974acc4842d6a79465a9ef8249c3889f7eef5385ea95a4b061`. Attestation `rva-71fdab3c13270779` explicitly answers all 15 criteria and all 11 tool entries, including both unscoped and pose-mcp validation. Recommended integration and debt scans were executed but not used as clean passing proof for the reasons above. Completion gates were deferred until the attestation, then actually executed.

Strict validation passed 157/157 checks with no skips, errors or failures. Artifact reconciliation matched 7 claims to 7 actions in the single implementation commit; its 742 global orphan warnings concern inventory outside the spec reconciliation. Surface-check reported zero findings for the declared capability. Native close succeeded, strict spec lint passed, and all-spec lint passed across 453 specs. After the Final Report update and reindex, review verify remained fresh and approved with state closed; closeout-check confirmed terminal=true. The final strict global check passed with 30 existing warnings. The pre-transition closeout-check's lifecycle-not-done result is retained separately from the successful final terminal check.

Assessment/state outputs were not committed. The required closeout discovery ran; its generated output was excluded as requested. This commit carries review evidence and lifecycle/report changes only, plus the native regenerated indexes. See execution-summary.json and the final-* logs for the final outcomes.
