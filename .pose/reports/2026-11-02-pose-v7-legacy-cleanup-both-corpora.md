# POSE 7 legacy cleanup, both corpora — 2026-11-02

Spec: [pose-roadmap-reviews-findings](../specs/2026-11-02-pose-roadmap-reviews-findings.md). It completes the plan in [pose-v7-legacy-cleanup-plan.md](pose-v7-legacy-cleanup-plan.md), which measured pose-dist only, against the gate of milestone `major-cleanup-plan`.

`pose migrate v7 --dry-run --json` writes nothing. On Harne8 the SHA-256 over every path and byte under `.pose` was `b40fc9d6187907f1` before and after the run: `find .pose -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum`, first 16 characters.

| Class | pose-dist `fcd11da9` | Harne8 `ba83cbfb` |
|---|---:|---:|
| `attestations-declared` | 1549 | 193 |
| `attestations-legacy-undifferentiated` | 1354 | 201 |
| `attestations-verified` | 0 | 10 |
| `flat-spec-amendment-journals` | 26 | 2 |
| `interrupted-transfers` | 0 | 10 |
| `policy-legacy-adoption-keys` | 5 | 5 |
| `policy-unknown-keys` | 0 | 0 |
| `sealed-bundles` | 1599 | 201 |
| `sealed-bundles-without-governing-contracts` | 483 | 12 |
| `spec-layout-flat` | 477 | 73 |
| `spec-layout-folder` | 3 | 0 |
| `spec-layout-legacy-folder` | 0 | 356 |
| `status-blocked` | 0 | 2 |

Complete: pose-dist True, Harne8 True. Writes: pose-dist [], Harne8 [].

Harne8 was measured at its pinned head by the engine of pose-dist `59358c28`, without adopting it. Nothing here converts a record: an apply stays future work, with its own recovery and revision guard.

## Retained independent reproduction

The independent review repeated this inventory with engine `f931afe0`; its pose-dist capture is at that revision and its Harne8 capture remains at `ba83cbfb`. The 13 class counts match this table. This is a later reproduction, not the raw output of the original `fcd11da9` measurement. The [retained fixtures and provenance](../../pose-mcp/internal/pose/testdata/cleanup-corpora/README.md) preserve this distinction. Harne8's private detail paths are omitted from the public summary; its complete captured output is retained in the local ignored evidence directory, with its raw digest recorded alongside the public export.
