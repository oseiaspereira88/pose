# CLI reference

**Doc type:** Reference &nbsp;·&nbsp; **Applies to:** POSE 5.x (current stable)

The `pose` CLI is a single native Go binary. Every command below executes
without Bash or Python fallbacks and works offline.

## Scaffold

| Command | Purpose |
|---|---|
| `pose init [--wizard [--yes]]` | Ensure the minimal structure; the wizard detects stacks and seeds the validation matrix |
| `pose specs [--recent N] [--status S] [--since D] [--json]` | List and discover specifications chronologically (newest first) |
| `pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run]` | Inspect and migrate specifications to chronological layout with companion preservation |
| `pose new-spec <slug> [--folder\|--legacy]` | Create `.pose/specs/YYYY-MM-DD-<slug>.md` from the template; `--folder` writes `YYYY-MM-DD-<slug>/spec.md`, `--legacy` writes `<slug>/spec.md` |
| `pose new-roadmap <slug>` | Create a governed roadmap in `.pose/roadmaps/` |
| `pose new-adr "<title>"` | Create a dated ADR |
| `pose new-knowledge <type> <slug>` | Create handoff/note/decision-log (`--owner`, `--ttl-days`, `--restricted`) |

## Deterministic gates

| Command | Purpose |
|---|---|
| `pose check [--strict\|--tolerant]` | Structural integrity + matrix schema + task-map sync + spec graph + schema version |
| `pose validate [--strict\|--tolerant] [--stack s] [--module p\|--workspace w\|--root-only] [--changed-from A --changed-to B] [--explain] [--report [--report-task T]] [--json P] [--junit P] [--sarif P] [--emit-plan P]` | Run the validation matrix; `--report` writes a report that lists the commands this run executed and its result |
| `pose lint-spec <slug>\|--all [--ready-check]` | Section content, DoR entry gate, done-lifecycle gate; warns in any status when a follow-up's ownership is outside its trailing group |
| `pose followups [--open\|--all] [--overdue] [--owner @alias] [--fail-overdue] [--similarity N] [--json]` | Aggregate follow-ups, overdue triage dates and near-duplicate candidates |
| `pose knowledge-check [--max-overdue N]` | Knowledge schema + overdue backlog |
| `pose recurrence-check [--window-days N] [--threshold T]` | Recurring failing task slugs |
| `pose history-check` | All history JSONL must be git-tracked |
| `pose artifact-check --spec S [--from A --to B]` | Reconcile declared artifacts with an immutable Git change set |
| `pose surface-check [--spec S] [--results P]` | Prove composition/reachability with current typed validation evidence |
| `pose roadmap-check <slug>` | Evaluate member closeout, registered cut criteria and required delivery findings |

## Component-aware and convergent review

| Command | Purpose |
|---|---|
| `pose review-plan <scope> [--json] [--explain]` | Resolve the deterministic component-aware plan, provenance, criteria, safe native-tool guidance and plan digest |
| `pose review bundle <scope> [--json] [--explain] [--seal]` | Prepare the semantic review subject or persist it as an immutable `rvb-` bundle after required evidence is current |
| `pose review auto-attest <bundle-id\|scope-ref> [--reviewer <id>] [--apply]` | Extract matching evidence from validation results, resolve tool dispositions and record attestation |
| `pose review attest <bundle-id> --reviewer ID --decision D --evidence REF [--criterion C] [--tool DISPOSITION] [--finding F] [--plan-digest SHA] [--apply]` | Preview or append a local `rva-` attestation bound to the exact sealed bundle |
| `pose review attest --envelope <project-relative-path> [--apply]` | Verify and preview/import a policy-trusted external attestation envelope |
| `pose review verify <scope\|bundle-id\|bundle-path> [--json]` | Verify bundle freshness, attestation completeness and closeout readiness |
| `pose review record <scope> --reviewer ID --decision D --evidence REF [--finding F] [--tool DISPOSITION] [--plan-digest SHA] [--apply]` | Record a review against the scope's current plan; when review bundles are enabled it requires a current sealed bundle and binds to it |
| `pose review-check <scope> [--json]` | Enforce the current review plan and accepted attempt/attestation |
| `pose closeout-check <scope> [--json]` | Evaluate hierarchical spec, milestone or roadmap closure |

`<scope>` is `spec:<slug>`, `milestone:<roadmap>/<id>` or
`roadmap:<slug>`. Component-aware planning and sealed bundles are explicit
policy opt-ins; legacy policies and historical attempts remain readable. The
convergent path is:

```bash
pose review bundle spec:customer-export --explain
pose validate --strict
pose review bundle spec:customer-export --seal
pose review auto-attest spec:customer-export --reviewer agent:reviewer-a --apply
pose review verify spec:customer-export
pose review-check spec:customer-export
pose closeout-check spec:customer-export
```

The bundle digest includes governed semantic inputs, attributed patch/tree
identity, consumed plan inputs and required evidence identities. Lifecycle
bookkeeping, generated state and the attestation itself stay outside that
digest, so approval does not invalidate its own subject. Semantic or source
changes create a superseding bundle with a typed delta; derived closeout
updates do not force a mechanical rereview. Plan resolution, bundle preview
and verification never execute recommended tools or widen caller authority.

### What a sealed bundle fixes

Sealing records, inside the bundle, everything later verification judges it
by, so an old approval is never re-judged by today's configuration:

- the attributed change sets and subject, the plan and the review profiles it
  selected per component, and the identities of the required evidence;
- the governance contracts in force when it was sealed — `component-aware`,
  `review-bundles`, `evidence-vocabulary` — so a contract adopted later never
  reaches it;
- the gates that decide it: whether `approved-with-reservations` closes, which
  severities an accepted risk may carry, and whether a prior criterion
  disposition may be reused.

The one setting read live is `require_signed_attestations`: tightening signing
deliberately applies to bundles sealed before the change. A bundle sealed by an
older engine carries none of these fields and is judged by the dated
`contract_adoptions` rule and conservative gates instead. Sealing warns about
evidence produced against a commit other than the head it approves — accepted
as current, but named so a reviewer can tell carried-forward evidence apart.

### What an attestation has to show

- A criterion recorded `passed` must cite evidence the sealed bundle contains,
  of a class the criterion accepts. `evidence_classes` is a disjunction: any one
  of the listed classes satisfies it.
- Evidence for a criterion or tool scoped to a component must come from that
  component. A result from a directory *inside* a component does not answer for
  the whole component; a module-wide run does answer for its subtree.
- `--criterion ID|disposition|evidence|rationale` records a disposition for a
  required criterion: `passed`, `not-applicable` (a rationale is required) or
  `finding` (the evidence slot names a finding this attestation records).
- `--finding ID|severity|disposition|action|evidence[|owner|rationale|review-by]`
  records a finding. Every finding needs a severity and an action. `resolved`
  and `wont-fix` close it; `accepted-risk` closes it only for a severity the
  sealed gates accept and only with an owner, a rationale and a review date;
  `open` and `changes-requested` block closeout.
- The decision must be `approved`, or `approved-with-reservations` where the
  sealed gates allow it.

## Discovery, metrics, artifacts

| Command | Purpose |
|---|---|
| `pose suggest [<type>] [--domain d] [--path p] [--json]` | Canonical trail: workflow + skill + rules |
| `pose stats [workflows\|tasks\|contexts] [--since-days N]` | Outcome aggregation from history |
| `pose usage [--since-days N] [--tool NAME] [--surface cli\|mcp] [--json]` | Automatic local tool calls, outcomes, finding lifecycle and latency by CLI/MCP surface |
| `pose index` | Regenerate all indexes (repo-map, spec-graph, roadmaps…) |
| `pose report --task "..." [--outcome pass\|fail\|partial\|skipped\|unknown] [--spec S] [--since ref] [--change-from A --change-to B] [--validate-output P] [--git-stage] [...]` | Versionable report + history JSONL; `pose report --help` lists all sixteen flags |
| `pose public-claims [--strict\|--tolerant] [--json]` | Check that every surface a project declares (site, README, docs) claims the version it actually released, from `.pose/public/claims.json` (opt-in; start from `.pose/templates/public-claims.json`) |

`pose usage` needs no counters from agents. POSE records recognized terminal
CLI commands and project-backed MCP tool calls at their execution boundaries;
the query itself is excluded. Exact structured gates contribute their stable
findings, so the report can distinguish total observations from unique, new,
resolved and reopened findings. Generic failures remain conservative instead
of parsing arbitrary terminal output.

The journal is best-effort and local-only, outside the tracked worktree. Its
allowlisted event schema never persists command arguments, output, repository
paths/names, source content, project/user identity or raw finding IDs; scope
and finding identities are project-local HMAC fingerprints. Recording failure
never changes the wrapped command's output or exit code. Use
`POSE_USAGE_DISABLED=1` to disable collection. These are POSE product-usage
signals, separate from DORA delivery metrics and unsuitable for individual
productivity scoring.
Set `POSE_USAGE_DIR` only when an operator needs an explicit absolute local
state directory (for example, a persistent container mount); the default Git
common-dir/user-cache resolution is preferred.

POSE does not infer the human adjudication states `valid`, `wont-fix` or
`false-positive`. Recording them is designed in the draft spec
`pose-usage-findings-adjudication`, which keeps verdicts separate from the
automatic observation counts and still has its storage decision open. See
[Analytics and delivery metrics](analytics.md) for interpretation and examples.

## DORA and adoption metrics

| Command | Purpose |
|---|---|
| `pose record-deployment --application A --environment E --deployment-kind planned\|rework --status success\|failure --source manual\|ci\|webhook [--deployed-at RFC3339] [--lead-time-seconds N] [--change-ref R]` | Ingest one schema-v2 deployment event |
| `pose record-incident --application A --environment E --started-at RFC3339 --severity minor\|major\|critical --source manual\|ci\|webhook [--resolved-at RFC3339] [--caused-by-deployment]` | Ingest one schema-v2 incident event |
| `pose dora-metrics [--application A] [--environment E] [--window-days N] [--json]` | The current 5 DORA metrics for one production environment; `E` defaults to `production` |
| `pose adoption-metrics [--json]` | Activation, time-to-first-gate, retention, task success — derived from specs/history POSE already owns |
| `pose events-housekeeping <list-expired\|purge> [--older-than-days N] [--apply]` | Retention/deletion for stored deployment/incident events |

Deployment and incident events are explicit input only — POSE never infers
them from commits — and carry no identity field beyond `application` and
`source`; every metric is a team/application aggregate, never an
individual score. The five metrics are deployment frequency, lead time for
changes, change failure rate, failed deployment recovery time and deployment
rework rate. Recovery includes only resolved incidents explicitly marked
`caused_by_deployment`; rework requires every scoped deployment to declare
`deployment_kind`, otherwise that metric reports `unavailable` instead of
guessing that legacy events were planned. Schema-v1 JSONL remains readable.
See [DORA metrics guide](https://dora.dev/guides/dora-metrics/).
For a side-by-side model of usage, adoption and delivery signals, see
[Analytics and delivery metrics](analytics.md).

## Semantic governance assist

| Command | Purpose |
|---|---|
| `pose semantic-suggest (--for <spec-slug>\|--query "text") [--top N] [--provider lexical] [--json]` | Advisory suggestions: related follow-ups, recurrence patterns and knowledge, each cited with score/rationale/provider |
| `pose suggest-feedback --for <spec-slug> --ref <artifact-ref> --kind knowledge\|followup\|recurrence --decision accept\|reject [--score N]` | Record a minimized accept/reject decision (never the candidate's content) |

Suggestions are advisory only — they never gate a check or mutate a spec.
`lexical` (deterministic, offline token/sequence similarity) is the only
approved provider today; sensitivity-restricted knowledge is filtered
before any retrieval, never suggested.

## Capability assessment

| Command | Purpose |
|---|---|
| `pose assess` | Validate `.pose/capabilities/assessment.md`: schema, typed evidence resolution, stable mechanism ids, staleness vs. policy |
| `pose assess init` | Scaffold the artifact with the method's 16 default mechanisms |
| `pose assess snapshot` | Append the current score vector to `history.jsonl` (append-only; no-op when unchanged) |
| `pose assess diff [--from <ts>] [--to <ts>] [--against <project-id>] [--json]` | Mechanical comparison between two snapshots (raised/lowered/added/retired), or a score matrix against another authorized root |
| `pose assess stale [--json]` | List mechanisms currently marked assessment-stale, with their pending trigger(s) |
| `pose assess request --mechanism <id> [--reason <text>]` | Manually mark one mechanism stale (the same path a UI-driven "flag for reassessment" action would call over MCP) |

Scores are human judgment (0-5; the target is not always 5) — the mechanism
validates structure and evidence, it never computes a score. Evidence uses
typed references (`spec:`/`report:`/`adr:`/`knowledge:`/`doc:`/`commit:`/
`check:`/`url:`); local types must resolve, the rest are syntactic
(offline contract). `pose check --strict` runs the same validation when the
artifact exists (opt-in by presence). Staleness thresholds live in
`.pose/policy/capabilities.json` (defaults: 30 days / 200 commits).

**Reassessment triggers** (spec `pose-capability-assessment-triggers`): a
post-event hook consumer marks a mechanism assessment-stale whenever a spec
closeout reaches components that materialize it — resolved via
`components_hit` when GraphForge is configured, or by matching the event's
touched files against a mechanism's declared `paths:` globs (a manual,
semicolon-separated fallback field on the mechanism) when it is not. A stale
mark never touches the score; it only records `since`/`trigger`/`hits` on
the mechanism and projects a synthetic, owned follow-up (origin
`capability:<mechanism>`) into `pose followups --open`, so the reassessment
demand is cobrável without a second store. `pose assess snapshot` clears
every pending mark on the mechanisms it scores, linking the clearance to the
new snapshot in `history.jsonl`. Without a component map and without any
`paths:` declared, the event logs a visible
`capability_mapping_unavailable` outcome instead of marking anything
silently. Anti-noise thresholds (`min_hits`, hit `level`, the follow-up's
default owner/review SLA) share the same `.pose/policy/capabilities.json`
policy file.

## Docs governance

| Command | Purpose |
|---|---|
| `pose docs-init [--profile library\|service\|cli\|monorepo]` | Scaffold `.pose/docs.json` with a profile's recommended `roots` — a recommendation, never mandatory |
| `pose docs-check [--json] [--explain <rule>]` | Validate the manifest: declared docs exist, undeclared docs are flagged, frontmatter/links/typed references resolve, staleness, and a security scan |
| `pose docs-review resolve <doc> [--no-change --reason <text>] [--commit <sha>]` | Close a doc's pending review marks: `updated` (default, captures the current commit unless `--commit` is given) or `no_change_needed` (`--reason` required) |
| `pose docs-review request <doc> [--reason <text>]` | Manually mark one doc for review (the same path a UI-driven "flag for review" action would call over MCP) |
| `pose docs-review request --all-stale` | Bridge every doc `docs-check` reports `stale` into an active, owned review-pending demand, in one call |

Opt-in by presence of `.pose/docs.json` — a project without the manifest
stays valid everywhere, same mechanic as the capability assessment above.
The manifest declares `roots` (governed doc directories) and `entries`
(one per doc: `path`, `doc_type` — Diátaxis `tutorial`/`howto`/`reference`/
`explanation`, or a custom value — `topics`, `owns`, `applies_to`,
optional `review_after`). Each declared doc needs a YAML frontmatter block
with at least `title` and `doc_type`. Seven deterministic, offline rules —
`missing`, `undeclared`, `missing_frontmatter`, `broken_link`,
`broken_reference`, `stale`, `security` — each with a configurable
severity (`error`/`warning`/`off`) in the manifest's `severities` field;
`pose docs-check --explain <rule>` documents the rationale. Staleness
compares an entry's own `review_after` (an absolute date) or, when unset,
the manifest's `default_review_days` counted from the doc's last touching
commit. The security scan reuses the same deterministic, offline
unsafe-instruction/secret-shaped pattern scan skills already run — defense
in depth, not a substitute for the dedicated gitleaks gate. `pose check
--strict` incorporates `docs-check` when the manifest exists (opt-in by
presence, same mechanic as capabilities); errors block, warnings surface
without blocking. Tool MCP: `pose_docs_state`.

**Review-pending triggers** (spec `pose-docs-assessment-followups`, third
consumer of the same post-event hook registry as the capability
reassessment triggers above — reused unmodified): a `docs-review`
consumer, registered on `spec_closeout`, resolves which components/files a
closeout reached (`components_hit` when configured, matched against
`owns:` entries declared as `component:<id>`; otherwise the event's
touched files matched against each doc's `owns:` paths/globs — a
directory prefix like `"site"` covers every file under it) and marks
every doc whose declared area was reached as review-pending — never
editing the doc itself. Marks accumulate in an append-only log, never
inside the doc's own file, and project a synthetic, owned demand into
`pose followups --open` (origin `docs:<doc-path>`), reusing the owner
declared on the manifest entry when present. `pose docs-review resolve`
closes every mark currently pending on a doc at once, recording the
outcome; `docs-check`'s own output (and `pose_docs_state`) additively
list what's still pending. Without a component map and without any
`owns:` declared, the event logs a visible signal instead of marking
anything silently — same degrade-by-absence contract as the capability
triggers, except here the path fallback is the mechanism's full-strength
path (`owns:` is expressed as paths by default), not a lesser one.
Anti-noise threshold (`min_hits`, hit `level`, the demand's default
owner/review SLA) is configurable, sharing the same policy shape as the
capability triggers in its own file (optional; absent means these same
conservative defaults).

## Cross-repository portfolio

| Command | Purpose |
|---|---|
| `pose portfolio-projection [--projects-dir DIR] [--max-staleness-days N] [--json]` | Reconcile dependencies, readiness, ownership and criticality across authorized repositories |

Only repositories registered via `HARNE8_PROJECTS_DIR` (or explicit
`POSE_PROJECT_ROOTS`) — the same allowlist the MCP server already uses —
ever enter a projection; nothing is discovered by an open filesystem
walk. Add `depends_on: xref:<project_id>/<spec-slug>` to a spec to
declare a cross-repository dependency (additive to the existing
`other-spec` / `milestone:...` / `roadmap:...` forms). The projection is
persisted to `.pose/reports/portfolio-projection.json`, explains every
blocked, stale or unauthorized/unknown cross-reference explicitly, and
tombstones artifacts that disappeared since the last run rather than
silently dropping them. Repositories remain authoritative; the
projection is a reconciled read, never a write back to another
repository.

## Harness evidence reconciliation

| Command | Purpose |
|---|---|
| `pose reconcile-evidence record --run-id ID --request-id ID --execution-id ID --plan-digest SHA --status success\|failure --source harness\|manual [--result-digest SHA] [--allow-supersede]` | Reconcile a Harness execution result into local evidence, identity-bound to the submitting Execution Identity |
| `pose reconcile-evidence list [--request-id ID] [--json]` | List recorded evidence |
| `pose reconcile-evidence housekeeping <list-expired\|purge> [--older-than-days N] [--apply]` | Retention for evidence records |

A second record for a `request_id` that already has evidence is rejected
unless `--allow-supersede` is passed — and even then the prior record is
never edited or removed, only superseded by a new, explicitly-linked one.
See [architecture: Harne8 control-plane composition](architecture.md#mechanism-15-harne8-control-plane-composition).

## Import existing SDD specs

```bash
# Preview every spec-kit feature under the tree without writing files.
pose import spec-kit .specify/specs --dry-run

# Import an OpenSpec capability, specs tree, or change directory.
pose import openspec openspec/changes/add-2fa
```

The importer is native, deterministic, and offline. It accepts a single
`spec.md`, a feature/capability directory, or a supported specs tree. spec-kit
imports consume sibling `plan.md` and `tasks.md` when available; OpenSpec
change imports consume `proposal.md`, `design.md`, `tasks.md`, and capability
specs below `specs/`.

Every unit becomes `.pose/specs/<slug>/spec.md`. POSE validates the complete
batch before writing, never overwrites an existing destination, rejects
symlinks, and reports every source section that still needs human curation.
Use `pose lint-spec <slug> --ready-check` after reviewing that report. The
first version intentionally does not support force-overwrite, bidirectional
sync, custom spec-kit presets, or OpenSpec schemas outside the documented
behavioral/change layout.

## Open-Source POSE Contributor Mode

| Command | Purpose |
|---|---|
| `pose contribute enable [--target <dir>]` | Opt into POSE open-source contribution; injects governed instructions into `AGENTS.md` and `POSE.md` |
| `pose contribute disable [--target <dir>]` | Disable contributor mode and remove contributor sections |
| `pose contribute status [--json]` | View contributor mode status, staged count, and privacy guardrails |
| `pose contribute stage --title "..." [--type bug\|enhancement\|limitation] [--body "..."]` | Stage a structured feedback artifact in `.pose/contributions/` |
| `pose contribute list [--json]` | List all staged feedback contributions awaiting developer adjudication |

When enabled, executing AI agents automatically stage feedback, bug reports, and stack extension proposals under `.pose/contributions/` whenever encountering workflow frictions. Staging is default and local, while submission to upstream GitHub (`oseiaspereira88/pose`) remains under full developer control. Staged feedback strictly isolates POSE engine behavior and is prohibited from containing proprietary source code, internal hostnames, or credentials.

## Maintenance

| Command | Purpose |
|---|---|
| `pose install <dir> [--locale <tag>] [--skip-mcp] [--force] [--no-backup] [--allow-non-git] [--project-id ID] [--project-name N]` | Install POSE into a Git repository and run the strict gate |
| `pose update [--dry-run] [--force] [--no-self] [--locale <tag>]` | Update the binary to the latest release and migrate the instance; `--no-self` keeps the current binary, `--force` also refreshes scaffolds, rules, workflows and MCP config |
| `pose doctor [--json] [--fix [--yes] [--only <check>]]` | Read-only diagnostics; `--fix` previews confined remediation, `--fix --yes` applies and rechecks it |
| `pose knowledge-housekeeping <op> [--apply]` | List/archive/purge expired knowledge |
| `pose reports-housekeeping <op> [--apply]` | Same for reports (never touches `history/`) |
| `pose hooks <install\|uninstall\|status>` | Git hooks: pre-commit check, post-merge reindex |
| `pose serve-mcp [--stdio]` | Start the MCP server (unified binary) |
| `pose version` | Binary version + instance schema version |

`pose update` replaces its own binary first and then hands off to it, so
migrations shipped with the new engine apply on the update that delivers them.
Managed manuals (`AGENTS.md`, `POSE.md`) are merged rather than overwritten:
sections the instance owns keep their content, and anything a merge cannot keep
in place is saved to `<file>.pose-backup` and reported. A run that has delivered
its files but finds the instance's own state invalid — a corrupt changelog
fragment, say — reports that through the final gate, which says whether the
failure predates the run; nothing is rolled back.

`pose doctor` diagnoses the instance without changing it. Beyond the binary,
dependencies, schema, skills, hooks and MCP configuration, it reports:

- `review.evidence-vocabulary` — a selected review profile demands a class no
  check may emit;
- `validate.class-producers` — a selected profile's criterion accepts only
  classes no registered check produces;
- `validate.evidence-class-coverage` — registered checks declare no
  `evidenceClass`, so their results never reach a review;
- `review.profile-schema` — a review profile below the schema the engine
  enforces, with the command that migrates it;
- `review.contract-adoption` — a governance contract with no adoption date;
- `review.scope-change-set` — a spec that `delivers:` a target but has no
  attributed change set;
- `<policy>.policy-keys`, for the review, delivery, artifact, capability, docs,
  release, state, changelog and definition-of-ready policies — keys the engine
  does not read, which is how a misspelled setting silently does nothing;
- `policy.delivery-roots`, `policy.artifact-roots`,
  `instance.config-completeness`, `machinery.retired-on-disk`,
  `module-metadata.orphan-entries`, `validate.redundant-workspace-execution`
  and `rules.stack-extension-available`.

`pose serve-mcp --stdio` exits on SIGTERM, and a server started before a
`pose update` keeps running the CLI installed at the path it started from.

Every gate is offline by design — no network calls, stdlib only. A gate
observed doing network I/O is a reportable bug (see SECURITY.md).

## Release lifecycle

| Command | Purpose |
|---|---|
| `pose release plan --version vX.Y.Z` | Preview the cut: fragments selected, version recommendation, blockers |
| `pose release prepare --version vX.Y.Z --apply` | Freeze the selected fragments, canonical notes and manifest |
| `pose release check --version vX.Y.Z [--strict]` | Validate the prepared snapshot |
| `pose release notes --version vX.Y.Z` | Print the frozen notes (what tagged CI publishes) |
| `pose release record --version vX.Y.Z --event tagged\|published\|verified\|failed\|yanked --evidence P` | Import provider or verification evidence as an append-only event |
| `pose release status --version vX.Y.Z` | Project the release's state from its events |
| `pose release open-next --version vX.Y.Z` | Confirm the latest release is verified and the next version is greater; changes nothing |
| `pose release backfill --from-git [--apply] [--json]` | Reconstruct release records from existing `v*` Git tags |

A release that introduces a governance contract says so in a **Compatibility**
section at the top of its notes: which contract, what it requires, and what
adopting it costs an engine that does not know it. That text comes from the
contract registry, so it cannot be forgotten. The compatibility alias
`release-notes --version` reads only the prepared snapshot; use `--preview`
explicitly for the pending queue.
