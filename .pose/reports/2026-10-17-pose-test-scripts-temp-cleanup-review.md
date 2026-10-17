# Independent review, round 13: test-script temporary cleanup

Decision: **approved**, with no open findings, reviewed at `534653c6eed826a2b17d655904c8581a62ba215e` on PR #192 (`impl/pose-test-temp-cleanup`). F8 and F9 are independently resolved. The maintainer-authorized reduced R3 is evaluated as written; no arbitrary-shell proof is claimed. Native guarded closure has set the spec to done, completed 2026-10-17.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, maintainer authorized. This Codex/OpenAI execution reviews Claude/Anthropic implementation, a different vendor. The `agent:independent-` prefix is an engine-required declaration until `pose-delegated-review-run-provenance` replaces it. Identity and separation are declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied, not independently authenticated, and no human adoption of these conclusions is claimed.

Read the full branch diff, current correction, spec, AGENTS.md/POSE instructions, applicable rules and pose-review/pose-spec-closeout skills. Built and used the branch CLI. No production code, tests, policy, capabilities or adoption were persistently changed by the reviewer. Temporary adversarial mutations were sequential and restored; only authorized operational closeout artifacts were edited. Requirements and Technical Plan remain byte-for-byte unchanged from 534653c6. The preceding rejected account is retained in that commit.

## Findings and exhaustive sweep

No new finding remains. The two prior open findings were retested independently:

**F8 — resolved.** The current measurement producer stages quickstart/first-governed-loop.sh and lib/tempdir.sh under one /measure directory, preserving the source relationship. Ran the unchanged actual Docker producer against golang:1.26.9-bookworm and the published installer: pose 7.4.1, all three measured commands exit 0, 12.486 seconds for the measured command path, valid result file outside the scratch TMPDIR and no remaining scratch entries. This exercises the real staged dependency, rather than only inspecting cp commands. No host source checkout or bind mount was introduced by the script.

**F9 — resolved.** Ran the unchanged current script with local Docker fault injection: start exits 7, rm exits 9, the script preserves exit 7 and removes its host root. A successful-rm control also exits 7 and cleans up. The previous 971288c7 script under the same failing-rm injection exits 9 and leaves a root. The old leftover was observed then removed. Fault controls contact no Docker daemon/network and fabricate no measurement report; the separate actual Docker run supplies the real execution evidence.

F1/F3/F4/F5/F6 remain resolved in their original scopes: 12 independent permission/failure cases clean up; exact shipped setup markers are reached with exits 0/1/1/1; delivered INT/TERM produce 130/143 and clean up; missing-handler and old-permissions mutations fail; hygiene itself cleans up on signal and failure; composition docs name pose_test_tmp_remove. F7 passes eight helper/real-script cases using relative TMPDIR with CDPATH unset or exported as dot.

F2 was closed by the explicit maintainer-authorized R3 rescope, not by claiming a complete shell analyzer. The revised omission/order gate rejects a newly tracked script without the helper, init after mktemp, commented/indented/one-line conditional init, missing source and source after init. A correct form passes. Each of ten whole-file pins was independently changed and rejected; missing/no-longer-allocating pinned files and an obsolete entry fail. Explicit repin updates the actual digest and permits the reviewed change. All mutations and pins were restored. All ten current pinned scripts were read and their hashes match. Previously admitted exotic helper sabotage is now an explicit new-script review responsibility.

All 24 tracked *.sh paths are in the declared scan roots, with descendants included by Git pathspecs. Static exemptions for helper and hygiene are covered dynamically. The three target scripts retain their assertions and options; source/init precede allocation and supported cd behavior. The demo passes --verify and cleans up after GOTOOLCHAIN=invalid build failure. Original scripts make the gate fail, and the original quickstart without --binary leaves its build directory, proving the empty-TMPDIR condition can fail.

## Requirements and design judgment

R1: the registered hygiene producer plus independent mutable-state permission, error, unset-variable, signal and CDPATH probes support the helper contract. The corrected Docker trap no longer skips filesystem cleanup when another cleanup action fails.

R2: all three real scripts pass with individual absolute TMPDIR and leave it empty; relative/CDPATH variants also pass. Quickstart --binary and install --binary/--previous 6.2.0 run from another directory. The real Docker measurement proves the copied quickstart consumer still executes its assertions with the helper supplied.

R3: the revised guard rejects omission/order, changed pins and stale pins as promised. Its value is review-triggering, not arbitrary-shell semantics or authenticated reviewer identity. Checksums can be regenerated by an actor with write access; the explicit review obligation remains part of the workflow.

D1 remains proportionate to one trap per script: an owned root centralizes cleanup and includes ordinary Go/curl/git child work without tracking every later allocation. Read-only descendants are made owner-accessible, and path capture is independent of cwd/CDPATH. The observed design delta is the integration check; it is non-material in the sealed structural vocabulary, so no formal structural mapping is required. The causal judgment still connects D1 to the omitted-cleanup cause and revised R3 negative evidence. A real-script leak or changed assertion would falsify approval.

Residual boundaries: SIGKILL independently leaves a root and cannot run a trap; machine/power loss has the same inherent limitation. Persistent caches and tools ignoring TMPDIR remain outside the declared scope. The static rule reads lines; unusual syntax that defeats the helper requires reviewer inspection of new scripts. These are documented boundaries, not accepted unresolved findings. No corrective follow-up remains.

## Deterministic and native evidence

Global strict validation: **137/137**, zero failed/errored/skipped checks, bound to 534653c6. Explicit pose-mcp strict validation: **134/134**. Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning and TestLocalVerifyCoversCurrentCIGates pass. CI, local verification and the required test-scripts-remove-their-temp-files-integration producer invoke the exercised gate. No token was required or printed.

Strict surface-check passes with zero findings for capability:test-scripts-remove-their-temp-files and entrypoint tests/lib/tempdir.sh. Artifact-check passes with all 14 claims observed among 16 paths; the two extras are review artifacts, with no scoped finding. Design observation covers 12/12 files and one non-material validation-check delta. Tech-debt reports zero markers/uncovered debt; recurrence flags zero keys.

The required integration assessment was actually run: 64 contracts, 63 pre-existing unobserved MCP-consumer observations, identical to the committed baseline. No API/MCP contract changed; direct shell consumers and actual Docker execution establish the changed boundary. These baseline observations are not newly inferred defects of this delivery.

Global discover --if-stale --update-state ran and refreshed its three registered components (docs-site, mcp-enforce, pose-mcp). The nine generated output changes were retained externally and restored byte-for-byte, as instructed. Examples/scripts/tests have no registered assessment module; their recommended discovery slots are explicitly not-used. Separate validate slots with producer_coverage=none are also not-used; actual changed shell behavior is exercised through pose-mcp checks, direct runs and ShellCheck. Each component has a distinct tool disposition in the attestation.

Bundle: `rvb-6167567bf3fa6ac4`, digest `sha256:6167567bf3fa6ac4a095ea0490af9097a793c6111f1ec947dc78b7c3aef6b171`. Plan: `sha256:cdaf914770996c5c915bcc74f3ca3d4dba663ec652be30a2ae14001c674303b7`, different-actor, 16 criteria and 20 tool slots. Read the sealed plan and the auto-attest preview's eight pending judgments; every criterion received a spec-specific conclusion. All required active tools were executed. Attestation: `rva-5358c584e2a8bd4f`, approved, with explicit prepared/concluded/applied agent attribution and no human confirmation. Preview and apply succeeded. Completion tools were deferred in the immutable attestation, then actually executed after the native transition.

Final index, review verify and check --strict pass; review remains fresh, approved and closed. Native close and closeout-check pass; strict spec lint passes after updating the real requirement producers. Strict check succeeds with 29 repository warnings; no policy, capability or adoption change was made to suppress them. The first pre-transition closeout-check correctly requested the guarded lifecycle transition; native close fulfilled that request. Three command-input corrections (external typed-output path, unsupported auto-attest JSON flag, and an unrequired mapping) were rejected by the CLI, corrected to the supported contract, and retained in external logs. No domain blocker was bypassed.

## Closeout records and evidence integrity

The spec records its completed task, actual R1–R3 producers, delivered scope, reviewer, evidence and residual boundaries. The two roadmaps record this transverse PR #192 closeout without adding a B-unit or changing program/roadmap status. The spec is absent from B01–B44, so pose-implementation-plan.json is unchanged. Closed-spec SHA-256: `281588b5299d0bb6f05bb23317c1085a9fe5eab6fab7a538672250de197927a0`. Maintainer retains merge authority; this run does not merge the PR.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full outputs and reproducible probes remain outside the repository in /tmp/pose-temp-independent-r13. The compact result contains only exit codes, scoped facts, finding dispositions and SHA-256 hashes; it does not embed repository-wide native outputs. The external raw manifest hashes all retained command/probe logs. Prior rejected rounds remain available in Git history.

## Rules applied during review

- POSE review workflow and pose-review/pose-spec-closeout skills: controlled bugfix scope, different-actor declaration, effective plan and native lifecycle gates.
- Security: paths, permissions, signal delivery, stage layout, pin integrity and secret-safe evidence.
- Backend Go: branch build, vet, full tests and registered integration producers; no production Go change.
- Documentation style and delivery evidence: real requirement producers, authorized R3 boundary and compact evidence distinct from structural lint.
- Knowledge governance: prior knowledge and follow-ups inspected; no new corrective handoff or upstream contribution staged.
- Cloud-environment runtime guidance: local Docker health and unchanged proxy/trust settings; actual container run succeeds. No unrelated adoption or release publication is authorized.

No open finding or corrective follow-up remains. The containing Git commit and final delivery response record publication of these closeout artifacts to the PR branch; merge remains with the maintainer.
