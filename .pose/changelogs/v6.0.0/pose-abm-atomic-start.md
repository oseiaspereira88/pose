---
spec: pose-abm-atomic-start
category: added
breaking: false
refs:
---

`pose start spec:<slug>` previews, read-only and digest-bound, whether a draft spec can start: readiness, dependencies, declared review obligations and its R/A/D baseline. With `atomic_start_version: 1` adopted in the review policy, `--apply --digest` records that baseline in `.pose/starts/<slug>.json` and moves the spec to `in-progress` under a per-spec lock with compare-and-swap; it is idempotent, resumes after an interruption, and `--cancel` leaves no half transition. `--status` and the read-only MCP tool `pose_start_status` classify nodes as recorded before managed execution, introduced during it, or legacy-unbaselined, and flag hand edits under the capability. A start records observable precedence, not when anything was decided. No instance adopts the capability in this release.
