# CI security refresh execution evidence

Collector: agent:codex-security-refresh-implementation. Static judgments are from the separate same-vendor/model R2 session 01a15ed2-054a-7480-9665-fdb3c1bef5be, not from this collector. No human confirmation or cryptographic identity claim.

## Executed checks

- Go 1.26.10 govulncheck v1.4.0: no vulnerabilities in pose-mcp and mcp-enforce.
- Gitleaks v8.21.2: full history passed with the exact fingerprint; mutating its allowed line exposed the historical finding; a synthetic token at the same report path remained detected.
- WorkflowSecurity, ArtifactIdentityContract, release-line registration, and sealed scanner metadata regressions passed. The metadata regression first reproduced the missing classification under isolated fixtures, then passed after the fix.
- Strict native validation at 2d840a1 passed 141/141 checks. A later collection at 32c80e8 passed 139/141, failing only the two builds with "error obtaining VCS status: exit status 128". Its unmodified native JSON is retained in .pose/results/ci-security-vcs-failure.json.
- The isolated Go 1.26.10 build immediately passed without source changes. Unrelated intermittent Git metadata under /tmp has previously interfered with this worktree's test environment. No external metadata was deleted and no Git-domain guard was weakened.
- Final collection uses TMPDIR=/var/tmp/pose-b07-validation-0bf729f, GOTOOLCHAIN=go1.26.10 and GOFLAGS=-buildvcs=false. The latter disables VCS stamping of validation binaries only; all 141 commands still run, including real Git unknown-state and materialization regressions. Release workflow flags are unchanged. Read the canonical .pose/results/delivery-validation.json for the actual final head, results and provenance; only a fresh passing result may be sealed.
- Native artifact-check --spec pose-ci-security-refresh --strict passed after all source commits; 12 declared claims were reconciled with Git, with existing orphan warnings retained. Native check --strict passed after the documentation reconciliation with 29 existing warnings.
- Integration assessment reran with existing heuristic consumer gaps. Tech-debt scan remains INCOMPLETE due the two invalid legacy spec sources; no complete debt-coverage claim is made.
- lint-spec --strict passed, with optional Tasks/Decisions warnings. Follow-up remains explicitly open for remote CI monitoring and merge decision.

## Limits

The separate reviewer inspected source, history and supplied reports, and ran read-only commands. It did not execute tests or verify a sealed bundle digest. Mechanical evidence is collected separately from those static judgments. Local closeout is not merge, release, deployment, engine adoption or production efficacy.
