# B34: read-view capacity calibration

Calibration performed on 2026-11-05 after implementation checkpoint
`74f5762fe2fbcea6ea25cb1d7f740a14b809bcfa`. All diagnostic builders were read-only
and retained outputs in memory. No diagnostic published an index generation.
The larger compiler-overlay limits described below are measurements, not
passing production gates or historical evidence rewritten under new limits.

## Production refusal and isolated diagnosis

The real index first refused its dependency depth. After D5 corrected that
scope, the next real index returned `invalid-federation-policy-path` for
`adaptive-instance-provisioning`. The federation policy was absent, an accepted
state. A read-only probe reproduced the build's order: module discovery followed
by federated material capture in the same view. It revealed the underlying
`governance read view budget exceeded` while validating the policy path.

| Same-view observation | Value |
| --- | ---: |
| Module manifests discovered before refusal | 3 |
| Observation records | 2,789 |
| Captured directory entries | 7,211 |
| Combined record budget consumed | 10,000 |
| Input bytes retained | 17,312,014 |
| Metadata bytes retained | 1,872,043 |

The material-only probe completed and audited with 2,558 records, 565 directory
entries, 19,038,075 input bytes and 593,439 metadata bytes. It did not reproduce
the combined build footprint. The false path classification must not conceal
an unknown closure when a callback fails after exhausting its view.

## Complete in-memory builder

The compiler overlay changed only diagnostic capacity and added a read-only
counter accessor. Production source limits remained 10,000 records and 128 MiB
throughout calibration. The overlay was not installed as the production CLI.
The probe called the same `buildIndexGenerationIn` and final view audit used
by the native index, without calling the publisher.

An initial measurement stopped on a duplicate artifact declaration introduced
in the B34 spec; the duplicate was removed. A second measurement, allowing
50,000 diagnostic records while preserving the 128 MiB input cap, stopped in
delivery-integrity Git capture: 6,404 records, 7,212 directory entries,
134,217,726 input bytes and 2,971,991 metadata bytes. It was incomplete.

The final calibration used a diagnostic cap of 50,000 records and 512 MiB,
preserving the 8 MiB metadata cap. It completed the builder and final audit in
147.63 seconds:

| Complete captured footprint | Value |
| --- | ---: |
| Observation records | 11,693 |
| Directory entries | 7,212 |
| Combined records | 18,905 |
| Input bytes | 370,690,278 (about 353.5 MiB) |
| Metadata bytes | 4,479,003 (about 4.3 MiB) |
| Encoded outputs in memory | 7 |
| Specs / roadmaps represented | 482 / 23 |
| Unknown producer reasons at builder completion | 0 |
| Final audit error | none |

Captured Git bytes grouped by command, without exposing output bodies:

| Command | Bytes |
| --- | ---: |
| show | 345,955,129 |
| rev-list | 3,093,737 |
| ls-files | 1,060,198 |
| log | 1,010,762 |
| diff-tree | 748,848 |
| literal-pathspec commands | 72,434 |
| rev-parse | 34,808 |
| show-ref | 11,673 |
| for-each-ref | 4,245 |

The operation retains complete observed history inputs; this measurement does
not authorize truncation or substituting HEAD/dirty for content. It is neither
a benchmark SLO nor a promise that every future repository fits the budget.

## Decision D6 and verification needed

Recalibrate the uniform read-view hard caps to **25,000 combined records** and
**512 MiB input**, leaving **8 MiB metadata**, the root/depth/graph limits and
one retry unchanged. This gives approximately 32% record and 45% input headroom
over the complete measured corpus. Actual allocation follows consumed inputs;
the caps are not preallocated buffers. Retention and final-audit buffers can
coexist, so these values are not an OS process-memory quota.

Keep capture, replay/fallback and final-audit accounting aligned with these
caps. Preserve the callback error and the demonstrated unknown/budget reason
together; genuine invalid-path errors remain distinguishable. Overflow must
still refuse publication. Use small near-boundary payloads and bounded I/O
spies for byte-budget regressions rather than allocating gigabytes in tests.

Normal-build focused capacity/callback tests passed after correcting the new
fixture's import and restricting its I/O spy to the subject rather than root
configuration. The expanded B34 integration passed with race detection:
domain 65.591 s, CLI 16.165 s and MCP 1.576 s. Full scaffold and distpolicy tests
passed in 1.272 s and 0.214 s. Oversized sparse inputs are rejected before reads;
small real payloads exercise the byte boundary without giant test allocations.

Actual index publication, current canonical validation and review are still
required. A successful calibration probe is not their result. Temporary probe
source files were removed; overlays/logs remain outside production artifacts.
