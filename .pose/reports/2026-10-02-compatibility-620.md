# POSE release compatibility report

- candidate: v6.2.0 / engine_version 6.2.0
- schema_version: 1
- commit: 552542f3b81409fdfa4dfa985f50ec8f4bab8bc1

- PASS: candidate binary reports 6.2.0 on every surface (stamped build)

## Contract gates (same candidate tree)
- PASS: version contract (CLI, MCP, registry, release pipeline)
- PASS: MCP catalog conformance (golden, docs, registry, schemas)
- PASS: compatibility matrix + schema upgrade fixtures
- PASS: embedded scaffold parity
- PASS: installer E2E (fresh install, verified download, doctor, strict gate)

## Supported prior-version upgrades
- PASS: 6.1.0 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 6.0.4 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 6.0.3 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 6.0.2 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 6.0.1 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 6.0.0 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 5.0.8 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 1.1.0 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 1.0.0 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 0.19.0 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)
- PASS: 0.18.2 → 6.2.0 (verified artifact; populated pt-BR + user-modified + spec/knowledge fixture; upgrade → strict gate → idempotent reapply → preservation verified)

Result: COMPATIBLE — release gate passed.
