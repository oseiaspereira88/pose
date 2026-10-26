# B18: Bound trusted delegated adapter execution

## Authority and scope

Canonical spec `pose-delegated-adapter-containment`, baseline `a2c0237add2adc789d1a338536713ccfd5e0b33f` after the maintainer merged PR206. Preserved R1–R10 and recorded the native start before code. Consumed knowledge:pose-b07-implementation; extended adapter/B07/B12/B16/B17 contracts through the linked ADR without rewriting accepted decisions. Implementation commits `1c4b18f` and `93bd35285e416b8c5724ffb32655adbaa5728196` carry the spec trailer.

## Implementation

Resolve explicit trusted-process or classified legacy-trusted compatibility and required host capabilities before reservation/spawn. Apply finite run/capture/output/retained-byte budgets, validate units/ranges/null/zero/unknown enforcement fields, and refuse unavailable isolation/resource requirements. Minimal environment excludes ambient tokens; explicit names authorize credentials. Fold reserved/duplicate names and normalize Windows environment keys before private HOME/TMP overrides. Preserve raw hashes, capture sessions privately, redact overlapping session/credential values together within byte bounds, then hash/canonicalize/sign retained artifacts. Partial boundaries withhold bytes rather than leaking credential prefixes or validating truncation.

Open the runtime root before spawn and read only bounded regular output through it. Bound Git setup capture/deadline, disable unsealed checkout hooks, propagate caller/CLI cancellation, bound pipes and observe Linux process-group termination. Cleanup uses captured descriptors for the exact workspace and Git registration, validates identity/backlinks/locks and bounds inventories, never deleting recursively through an adapter-controlled absolute path or pruning unrelated worktrees. Record cleanup OperationResult separately from the domain draft before immutable publication. Failed cleanup blocks authority/continuation, remains failure on replay and never respawns to compensate. CLI and existing MCP review-bundle projections expose the same safety record; metrics classify cleanup-incomplete drafts as unknown.

## Deterministic evidence

Final canonical strict validation passed 156/156 in 254.6s at the delivery-policy path `.pose/results/delivery-validation.json`, including `delegated-adapter-containment-integration`. The complete Go suite is part of that matrix. Final amended-source race passed domain/CLI/MCP in 15.902s/8.944s/2.246s. Vet/build passed; Windows cross-build passed using installed official Go1.26.10 (the distro GOROOT lacked Windows standard-library sources). Govulncheck found no vulnerabilities; CI-pinned gitleaks8.21.2 scanned both implementation commits with no leaks.

Strict spec lint passed with ten satisfied traces. Final artifact-check matched 31 claims/31 observed across the exact baseline-to-source change set, with no errors; 729 historical orphan warnings remain separate from current attribution. Strict structure check passed with 30 existing warnings. Integration inventory remains 64 contracts/1 observed/63 gaps; debt coverage remains explicitly incomplete due to two preexisting invalid specs (`pose-instance-engine-version-tracking`, `pose-knowledge-durable-reference-type`), with one covered marker and zero uncovered markers. Recurrence check passed.

Explicit index regenerated and verified current as `gen-7770162d99f1ebd0`. Surface-check passed 1 target/156 results/0 findings, provenance `sha256:cb172f8d95a03dee477e6aed6a69e5718b0f9fed5de4dc5171b8c62296b7450d`. Bundle `rvb-089d6bbf48270dc0` sealed 153 evidence records at source `93bd352`.

## Separate review and native closeout

Actual separate execution `/root/b18_review` reproduced cutoff/overlap/expansion leaks, descriptor/path cleanup races, lost runtime and failed replay. Permanent independent regressions cover finite setsid escape, exclusive registration deletion, preserved readers, case-folded environment aliases and no automatic respawn. Every finding was corrected and reviewed; final independent controls passed at the committed source.

The same separate execution published its own approved attestation `rva-013be502c850b7c2` as `agent:independent-b18-review`, with 15 criteria, 11 tool dispositions, no open findings and mapping `SD-ebdf5e74 → D1`. Auto-attest collected only mechanical evidence without apply; eight judgments were answered explicitly. Primary and reviewer independently confirmed fresh/approved/ready-to-close. Actual assurance is declared identity from a separate execution of the same vendor, without human confirmation or machine identity proof. Synthetic issuer fixtures were not promoted to real reviewer identity.

Native `pose close` applied done/completed_at 2026-10-26 UTC. Required post-close discover/update-state refreshed all three components (content/matrix changes). Final strict spec lint passed; native verification reported closed/fresh/approved, and closeout-check confirmed lifecycle_done=true, terminal=true, blockers=[] before publication. The spec has no residual follow-up requiring disposition. Final strict structure check again passed with 30 existing warnings; the post-close index was regenerated and verified current as `gen-25dc4c74affadc64`, and terminal verification remained true with no blockers. No follow-up disposition, instance policy adoption, real vendor, new sandbox backend, issuer, release, parent Harne8 gitlink or upstream contribution was activated.

## Limits

The backend requires a trusted local executable. A worktree shares Git and is not a sandbox; controlled fixtures demonstrate outside/transient access and finite escaped descendants. No OS memory/CPU quota, automatic expiry, aggregate disk quota or escaped-descendant containment is supplied. Windows/other platforms do not gain runtime qualification through cross-compilation. Filesystem calls remain subject to OS I/O latency. Exact-value redaction is not general DLP or protection against encoded secrets. Retain immutable evidence, including failed/rejected attempts, without deletion-based quota restoration. Export retained redacted artifacts with their complete inventory, excluding private sessions and issuer keys.
