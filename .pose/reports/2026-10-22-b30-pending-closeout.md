# B30 pending-work reconciliation — 2026-10-22

Canonical spec: `pose-spec-facts-evidence-binding`. Main baseline `2d42b84e3a065bb30cf9e7882c19a4f2535453f7`; merge reconciliation `0bb312b`. Generated-index conflicts restored from main for regeneration; named validation checks unioned, preserving B07/B08 and B30. Docs EN/pt-BR and embedded scaffold merged cleanly. No root Harne8 gitlink or shared checkout was changed.

R1 separate review: `2026-10-22-b30-pending-review-r1.md`. All three high findings reproduced with deterministic Go tests before fix. Source fixes now pin before reading the graph, parse report JSON semantically and disclose unbound retained explicit attribution, and reject impossible literal Git paths without normalizing meaningful spaces or `root` filenames. Fixtures that claim fresh now use actual Git trailer attribution rather than synthetic unknown selectors. No previously requested change is declared accepted by test count alone.

Assessment tech-debt: one covered marker, no uncovered marker among scanned files; scan INCOMPLETE due two legacy invalid spec sources (instance-engine-version-tracking and knowledge-durable-reference-type). This is not a debt-free claim. Fresh native strict validation, artifact/surface, final independent adjudication and closeout pending.

Validation environment: Go1.26.10, TMPDIR=/var/tmp/pose-b07-validation-0bf729f, GOFLAGS=-buildvcs=false. The VCS flag isolates validation fixture builds from unrelated /tmp Git metadata; it does not change release stamping or fail-closed Git guards.
