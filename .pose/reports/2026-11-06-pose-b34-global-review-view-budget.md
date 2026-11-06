# B34: global review view budget failure — 2026-11-06

Scope: `spec:pose-governance-read-view-generation`. Source baseline `5c9f1a96`;
operational profiles/signing setup subsequently committed. This is a reproduced
failure and corrective plan, not delivery evidence or an approval.

The normal binary passed the canonical 189 checks on that source baseline.
The real final index published `gen-3acfce7ed391ed59`; artifact-check and
surface-check finished with exit 0 and no error findings. The following real
`pose check --strict` exited 1 with 381 errors. Its last failure reported loss
of the global review input closure and exhausted record/metadata allocation.
The output contained 245 generic fresh-review actions, 135 path-escape errors
and one final unknown-closure error. These are not 381 proven independent
historical defects. The exact first exhausted bound is not measured in this
log; “Git command budget” comes from the shared record/metadata allocator,
not a separate Git-command counter.

The native read-only Sol medium investigation `trn-78de82fe660bb649`, attempt
`tat-f74e1df0ae423cd646a9c9b099a4329e`, completed with exit 0. It traced all
independent candidate closeouts to the single `WithReviewOperationView` in
`nativeChecker.checkReviewCloseout`. Nested domain calls reused that view,
accumulating unrelated closures and competing for one operation's capacity.
Identical Git requests are already deduplicated; this evidence does not prove
blob deduplication or a larger limit alone would resolve the whole flow.

`artifactPathWithin` also converts every symlink-resolution error to false;
`readListedSpec` labels that result as an escape even when the cause is lost
capacity. `WithReviewOperationView` audits but returns callback failure first,
discarding an accompanying audit failure. Audit-only budget loss must not be
represented as observed content mutation.

Accepted D7 and the ADR amendment separate the audited policy/candidate/catalogue
coordinator from independently bounded scope operations with bounded concurrency.
Catalogue reuse retains exact authority and Files bindings. Each scope's children
and federated roots stay in its operation. Buffer findings until audits finish;
keep unknown localized while failing the overall gate. Do not truncate history,
raise production ceilings, remove candidates, weaken retention or trust, or
claim a simultaneous snapshot across every scope. The actual first bound and
whole-corpus outcome remain to be verified after implementation.

A standalone `review verify spec:pose-skill-command-parity` returned missing
subject-path blockers. That alone does not establish its closeout outcome:
completed-scope retention is a separate domain path and must be re-evaluated
under a healthy scope view. No historical approval or record has been edited.

Required regression plan: actual coordinator and native-checker paths under
small injected limits; serial/parallel fitting scopes whose union exceeds the
cap; one exhausted sibling; policy and inventory interleavings including empty,
disabled and roadmap branches; retained approvals versus rejection or unreadable
history; precise containment errors and binding refusals; unknown versus observed
change and preserved callback/audit causes. Focused race, the complete canonical
matrix and real corpus gates remain mandatory before closeout.
