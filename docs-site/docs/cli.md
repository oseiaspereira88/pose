# CLI reference

**Doc type:** Reference &nbsp;·&nbsp; **Applies to:** POSE 7.x (current stable)

The `pose` CLI is a single native Go binary. Every command below executes
without Bash or Python fallbacks and works offline.

For daily work, start with [Operate an instance](operate.md) and
[Review and trust](review-and-trust.md). Consult [command compatibility](command-compatibility.md)
for deprecated spellings and supported aliases. The published baseline reviewed on
2026-10-20 is v7.4.1; sections labelled development describe pending changes.

## Setup, attention and authority

| Command | Purpose |
|---|---|
| `pose setup [--json|--no-input]` | Inspect configuration; interactive setup offers confirmed steps |
| `pose adopt --list` | List supported capabilities, prerequisites and recorded decisions |
| `pose state --governance [--scope REF] [--json]` | Explain effective policy and legacy cutoffs |
| `pose state --attention [--scope REF] [--actor ID] [--phase P] [--json]` | Project obligations, coverage, recipients and phase restrictions |
| `pose action list --present` | Group open material requests for a conversation |
| `pose action show ID --json` | Inspect a request, its journal and answer assurance |
| `pose identity list` | Inspect registered principals and public signing keys |
| `pose issuer list` | Inspect local issuers and trusted pins |
| `pose context --project-id ID --task XREF --json` | Resolve canonical project authority before qualified writes |

## Scaffold

| Command | Purpose |
|---|---|
| `pose init [--wizard [--yes]]` | Install POSE in the current repository when no instance exists (the installer's flags pass through), otherwise confirm it is installed; the wizard then detects stacks and seeds the validation matrix |
| `pose specs [--recent N] [--status S] [--since D] [--json]` | List and discover specifications chronologically (newest first) |
| `pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run]` | Inspect and migrate specifications to chronological layout with companion preservation |
| `pose new-spec <slug> [--folder\|--legacy]` | Create `.pose/specs/YYYY-MM-DD-<slug>.md` from the template; `--folder` writes `YYYY-MM-DD-<slug>/spec.md`, `--legacy` writes `<slug>/spec.md` |
| `pose new-roadmap <slug>` | Create a governed roadmap in `.pose/roadmaps/` |
| `pose new-adr "<title>"` | Create a dated ADR |
| `pose new-knowledge <type> <slug>` | Create handoff/note/decision-log (`--owner`, `--ttl-days`, `--restricted`) |

## Managed execution and contract changes

| Command | Purpose |
|---|---|
| `pose start spec:<slug>` | Preview readiness, dependencies, obligations and the R/A/D baseline without writing |
| `pose start spec:<slug> --apply --digest <digest>` | Apply the reviewed start under the adopted atomic-start contract |
| `pose start spec:<slug> --status` | Inspect the recorded baseline and managed-execution state |
| `pose amend <slug> --nodes --json` | Project versioned requirement, assumption and decision nodes |
| `pose close spec:<slug> --plan` | Inspect ordered closeout steps and current blockers |
| `pose close spec:<slug> --apply --reviewer <id>` | Execute closeout; stop where a judgment or authorization remains owed |
| `pose close spec:<slug> --resume --reviewer <id>` | Resume a recoverable closeout from its checkpoint |

Use digests from the current preview rather than reusing a stale plan. The
baseline establishes observable ordering, not when a decision was actually
made. See [instance operation](operate.md) and [review and trust](review-and-trust.md).

## Deterministic gates

| Command | Purpose |
|---|---|
| `pose check [--strict\|--tolerant]` | Structural integrity + matrix schema + task-map sync + spec graph + schema version |
| `pose validate [--strict\|--tolerant] [--stack s] [--module p\|--workspace w\|--root-only] [--changed-from A --changed-to B] [--explain] [--report [--report-task T]] [--json-out P] [--junit P] [--sarif P] [--emit-plan P]` | Run the validation matrix; `--report` writes a report that lists the commands this run executed and its result |
| `pose lint-spec <slug>\|--all [--ready-check] [--design-check]` | Section content and lifecycle gates; `--design-check` projects structured Assumption/Decision basis read-only with objective diagnostics |
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
| `pose review auto-attest <bundle-id\|scope-ref> [--reviewer <id>] [--apply]` | Extract matching evidence from validation results, resolve tool dispositions and report which criteria still await a reviewer's judgment; `--apply` records only when none do |
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
pose review auto-attest spec:customer-export --reviewer agent:reviewer-a
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
| `pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--waits] [--rework]` | Separate governance preparation, judgment, intervention, freshness and coverage dimensions, plus request waits (age, attributed wait, known blocking) and rework causes; no quality score |
| `pose migrate v7 --dry-run [--json]` | Read-only inventory of legacy representations a future major would convert, keep or refuse, with removal criteria; writes nothing |
| `pose assess design --spec S [--json] [--max-files N] [--max-bytes N]` | Bounded structural delta projection from the canonical review subject; no complexity score and no writes |
| `pose usage [--since-days N] [--tool NAME] [--surface cli\|mcp] [--json]` | Automatic local tool calls, outcomes, finding lifecycle and latency by CLI/MCP surface |
| `pose usage adjudicate --tool NAME --finding ID --verdict valid\|wont-fix\|false-positive --reason TEXT --by ALIAS` | Append a reviewed finding verdict to `.pose/usage/verdicts.jsonl` |
| `pose index` | Regenerate all indexes (repo-map, spec-graph, roadmaps…) |
| `pose report --task "..." [--outcome pass\|fail\|partial\|skipped\|unknown] [--spec S] [--since ref] [--change-from A --change-to B] [--validate-output P] [--git-stage] [...]` | Versionable report + history JSONL; `pose report --help` lists the supported flags |
| `pose public-claims [--strict\|--tolerant] [--json]` | Check that every surface a project declares (site, README, docs) claims the current candidate version, and report separately whether that candidate was prepared and which version has retained publication evidence, from `.pose/public/claims.json` (opt-in; start from `.pose/templates/public-claims.json`) |

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

POSE does not infer human verdicts. `pose usage adjudicate` records an explicit
decision in a tracked, append-only project journal. The finding ID must be a
bounded stable identifier; relative slash-separated check IDs are accepted,
while absolute paths, traversal and free-form text are rejected. Reports show
`valid`, `wont-fix`, `false-positive` and unmatched totals separately from automatic
observations; the false-positive rate uses only matched adjudicated findings.
Review the identity and reason before committing the journal. See
[Analytics and delivery metrics](analytics.md) for interpretation.

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

When enabled, agents ask for developer confirmation before staging feedback, bug reports, or stack extension proposals under `.pose/contributions/`. They ask separately before submitting anything to upstream GitHub (`oseiaspereira88/pose`). Staged feedback strictly isolates POSE engine behavior and is prohibited from containing proprietary source code, internal hostnames, or credentials.

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
sections the instance owns keep their content, and an edit the instance made
that a merge cannot keep in place is saved to `<file>.pose-backup` and reported.
The manifest records what POSE wrote to each section, so a release rewording
one of its own sections is not mistaken for such an edit. Machinery (rules,
workflows, templates, skills) is replaced file by file; the delivery manifest
records the digest of each file POSE delivered, so only a file the instance
edited since is backed up — a file a release changed and nobody touched is
refreshed quietly. A run that has delivered
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

The native governance engine can evaluate local repository artifacts offline.
`pose validate` executes the repository's declared tools: their network needs
follow those tools and checks. Installation, self-update, configured reviewer
commands and remote integrations also have their own execution requirements.
The local engine does not require a hosted POSE service.

## Output: two channels, one contract

POSE prints for two readers and keeps them apart.

| Channel | Carries | Where |
|---|---|---|
| **Result** | the verdict, the findings, the data a command was asked for | stdout |
| **Progress** | steps, module headers, usage, and the command's own failures | stderr |

For example, `pose check --json | jq` consumes the result channel. Keep stderr
when diagnosing a failed invocation: progress and parser errors are not findings.

| Flag / variable | Effect |
|---|---|
| `--json` | On commands supporting stdout JSON, print the command's result. In v7.4.1, `validate` is an exception: use `--json-out <path>`. Gates using the shared renderer carry `schema_version`, `command`, `outcome`, `findings[]` and `counts`; domain projections such as plans and stats have their own schemas. |
| `--json-out <path>` | Write the JSON result to a confined project-relative path on commands supporting this flag, retaining the human report. Prefer it to the deprecated `validate --json <path>`. |
| `--quiet` | Print the verdict alone. |
| `--verbose` | `pose validate` only: stream each check's output as it runs instead of capturing it. |
| `--color auto\|always\|never` | Force or suppress colour. |
| `NO_COLOR`, `POSE_COLOR=auto\|always\|never` | The same choice from the environment; the flag wins. |
| `POSE_LOCALE=en\|pt-BR` | The language of every message POSE composes. |
| `COLUMNS` | The width prose wraps to (80 by default). Contract lines are never wrapped. |

Colour, symbols and the spinner exist only when the stream is a terminal, and
the profile is resolved per stream: `pose validate 2>progress.log` keeps stdout
decorated and writes a clean log. `TERM=dumb`, a redirected stream and
`--quiet` all fall back to plain lines, and a terminal that cannot be trusted
with UTF-8 gets an ASCII profile (`[ok]`, `[x]`, `[!]`). Colour never carries
meaning alone: the state's word is printed in every profile.

A long run reports each step with its outcome and duration. On a terminal the
active step is one line that repaints with a spinner, a counter and the elapsed
time; anywhere else the same facts arrive one line per event, with no escape
sequences:

```text
[module] pose-mcp (go, mode=strict)
  -> test go test -count=1 ./...
  <- test pass 18.4s
  <- typecheck fail 2.1s exit=3
      internal/cli/x.go:42: unreachable code
      full output in .pose/results/validate.json
2 step(s) · 1 pass · 1 fail · 20.8s
Result: FAILURE — required check failed
```

**Exit codes.** `0` the gate passed, `1` the gate failed, `2` the command was
used wrongly (an unknown flag, a missing value, a path outside the project).

Lines machines depend on — the `Result:`/`Resultado:` verdicts and the
`name.field=value` diagnostics — are enumerated in the engine's
`testdata/contract-lines.json` and pinned by tests. Everything else is prose
that may be restyled between releases; read `--json` if you need stability.

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

### Release version source

A cut is compared with one authoritative version. In the engine repository that
is the engine's own version. Any other project declares the file that holds its
version in `.pose/policy/release.json`:

```json
{ "version_source": { "path": "VERSION", "kind": "text" } }
```

`kind` is `text` (one line) or `json` (a top-level string `key`, such as
`{"path": "package.json", "kind": "json", "key": "version"}`). The value is
`X.Y.Z`; a leading `v` and a `-dev` suffix are accepted and a pre-release tag is
not. The path must stay inside the project, the file is read once and bounded,
and the manifest records only the declared path, the kind and the compared
version. A project with no `version_source` outside the engine repository is
refused with a message naming the field. A policy that does not declare one
digests exactly as before.

`release prepare` moves each selected fragment from
`.pose/changelogs/unreleased/` to `.pose/changelogs/<version>/` and never edits
a spec. A spec that declared its fragment keeps the claim it wrote:
`artifact-check` resolves it through the manifest, which records each
fragment's spec and digest, and prints
`artifact.archived=<pending> -> <archived> (release <version>)`. A claim an
earlier release rewrote into `renamed: … -> …` resolves the same way. An
archived fragment edited after the cut, or claimed by a spec the manifest does
not name, stays a finding.
## Verified review authority

For the daily procedure, use [Review and trust](review-and-trust.md). The
sections below retain CLI-specific flags and record formats.

Review independence and identity assurance are separate. With
`identity_assurance: verified`, a reviewer prefix is only a declaration: the
attestation must carry a trusted, signed authority claim bound to the sealed
bundle, project audience, reviewer principal and required executions. Human
claims need a separate authority grant. Legacy bundles remain auditable under
the assurance mode sealed when they were created.

### Guided setup

Use [Operate an instance](operate.md#inspect-before-adopting) for setup,
capability decisions and configuration-review requests. The command table at
the start of this reference lists the read-only inspection forms.

### Delegated review

`pose review brief <bundle-id|scope>` prepares a deterministic brief for a
review, adjudication or smoke run (`--kind review|adjudication|smoke`).
`pose delegate review|adjudicate|smoke <bundle-id|scope>` launches the agent
configured in `.pose/policy/delegation.json` in a disposable worktree at the
sealed commit; `--dry-run` previews it and reserves nothing. Without
`--agent`, routing prefers an agent of the vendor opposite to the
implementation the bundle sealed, then falls back through
`routing.on_unavailable` (another model of the same vendor, a bounded wait for
a known usage-limit reset, the same model); `--agent <name>` bypasses it. Each
run records the routing decision and the independence it achieved, and
`pose delegate ls`, `resume`, `cancel` and `agents` follow the runs; `agents`
shows each agent as available, limited until a reset, missing or with a failed
probe. `pose review bundle <scope> --seal --lead <vendor>[/<model>]` declares
the implementation identity a seal records when the sealing session cannot be
detected. `pose review dispatch`, `resume` and `status` were removed;
`pose delegate` replaces them. The returned review remains subject to the
bundle's criteria, identity requirements and verification gates; dispatch
does not itself approve or close the spec.

#### Development branch: dispatch snapshot integrity

**Unreleased, in progress as of 2026-10-20.** The following describes pending
snapshot hardening rather than the published v7.4.1 implementation.

Dispatch verifies the sealed inputs in both the selected origin and the
disposable worktree before starting. After the command exits, it checks the
worktree's HEAD, materialized content (including ignored files), current bundle
and adapter configuration again. An unavailable Git observation fails the run;
`assume-unchanged` and `skip-worktree` do not authorize hidden edits. An unrelated
documentation commit does not stale a bundle whose semantic inputs still match.

Keep adapter output outside the worktree, using `{output}` or stdout. Adapters
cannot authenticate a regeneration producer, so writing assessments, indexes,
results or review artifacts inside the worktree fails the run. A directory name
does not grant permission. Disk comparison reads at most 100,000 entries and
256 MiB; exceeding either limit fails with an unavailable snapshot diagnostic.
Retry after resolving the reported cause; every attempt rechecks preconditions.

A completed run records observations at completion. It grants no current
fulfillment or attestation authority and does not isolate an adapter from the
host filesystem. If the origin becomes stale during execution, its transcript
is retained as a stale historical run and cannot fulfill the current review.

### Signed action answers

A project can verify who answered an action request without any external
service. Each principal registers the SSH public keys it signs with:

```bash
pose identity add --key ~/.ssh/id_ed25519_sk.pub --role maintainer --apply
pose identity list
pose identity remove human:ada --fingerprint SHA256:... --apply
```

With no principal, `identity add` suggests `human:<name>` from the git
identity; the name is only a name — the registered key is the proof. Keys live
in `.pose/policy/actions.json` under `keys`; only `ssh-ed25519` and
`sk-ssh-ed25519@openssh.com` are accepted, and a key belongs to one principal.
`pose action resolve … --sign <private key>` signs the answer with the user's
own `ssh-keygen -Y sign` (namespace `pose-action-answer`; POSE never reads the
key), and `--signature <file>` attaches a signature made elsewhere over the
bytes `pose action statement <act-id> --actor <p> --answer <a> --idempotency-key <k>`
prints. The statement binds project, request id and digest, actor, answer and
idempotency key. POSE verifies the SSHSIG natively, records the statement,
signature and key in the journal, and `pose action show` verifies it again.
Under `identity_assurance: verified`, an answer needs such a signature or a
trusted issuer's claim (Harne8 is one such issuer); a security key also records
whether it was touched, and `require_presence` refuses a human answer without
it. `pose doctor` reports human role holders that cannot prove an answer.

### Native attestation issuers

A project with POSE alone can sign review attestations. An issuer is an
Ed25519 key that lives outside the project, under `$POSE_ISSUER_HOME` or the
user config directory (`pose/issuers`), readable only by its owner; POSE never
prints it.

```bash
pose issuer init maintainer
pose issuer pin maintainer --attestations --apply
pose adopt signed-attestations --apply
pose review attest spec:my-spec --reviewer agent:reviewer-a --decision approved \
  --evidence unit:pkg --sign maintainer \
  --authority --review-execution review-2 \
  --implementation-principal agent:impl --implementation-execution impl-1 --apply
```

Grant `--human-authority` only to an issuer protected for that role; an account-local
key readable by agents is not proof of a person's confirmation.

`pin` adds `<name>#sha256:<public-key-digest>` to `trusted_attestation_issuers`
and, with `--human-authority`, to `human_authority_issuers`, keeping every pin
already there; it sets `authority_audience` and `authority_project` only when
they are empty. Native and external issuers coexist: a developer working with
the repository and POSE signs natively, one also using Harne8 signs through its
Conductor, and either pinned issuer's envelope counts. An envelope from an
unpinned key is refused. `--sign` produces the same envelope an external issuer
does and records it through the same verification; `--authority` adds the
reviewer authority claim that `verified-identity` reads. `pin name:<base64
public key>` trusts an issuer whose key is on another machine, and `pose issuer
rotate <name> --apply` replaces a key and pins the new one beside the old, so
attestations signed before the rotation keep verifying until the old pin is
removed. `pose issuer list` shows the pinned issuers and the keys on this
machine.

A project with history adopts signed attestations through the same command.
Signing stays required for every bundle, old ones included, so `pose adopt
signed-attestations` also seals the attestations recorded without a signature
into a ledger under `.pose/review-ledgers/`, signed by the local issuer pinned
for attestations (`--issuer <name>` when there are several). The ledger names
each attestation by its full content digest: those, unchanged, keep counting;
one recorded or changed afterwards does not, and no ledger can be sealed once
signing is required. The preview shows how many attestations will be sealed
and by which issuer.

### Retrospective ABM replay

`pose stats replay --json` scans local historical attestations and the newest sealed subject of each spec without writing project artifacts. It compares frozen contracts with hypothetical explicit judgment and structural causality invariants. Counts retain invalid inputs, unknown structural coverage, unbaselined nodes and denominators. Use `--limit N` (1–20000) to bound work; incomplete reports cannot support an exhaustive adoption claim. A counterfactual does not revoke historical approvals, record a baseline or prove reviewer utility. Experimental judgments remain opt-in.


## Development branch behavior

**Unreleased; reviewed 2026-10-20.** Do not assume these changes are present in
v7.4.1. Read [Releases and development](evolution.md) for source and status.

- `close … --apply|--resume --json` emits a single operation result: `operation`,
  `outcome`, steps run, confirmed effects, pending criteria, cause and next action.
  A no-op reports `already-satisfied`; it does not claim it reverified closeout.
- Invalid arguments to `specs`, `specs facts`, `start` and `close` are refused
  instead of ignored. Failed operations on the machine channel retain typed
  results; retries distinguish earlier effects from steps actually run now.
- `index --verify` checks one published generation and its inputs without writing.
  Missing, mixed or modified generation data is unavailable, not an empty graph.
- Invalid action journals disclose their location and a recovery proposal;
  the engine does not truncate them. Corrupt or mismatched closeout checkpoints
  stop execution before steps. `start --recover-lock <owner-digest>` only recovers
  an owner proven dead under the supported same-machine conditions.
- Shared approval preflight refuses approving records that verification would
  reject, including stale or superseded subjects, through every writer channel.
  Negative review decisions remain recordable audit history.

These are additive hardening and automation changes. They do not authorize an
agent to repair journals, remove unknown locks or accept incomplete proof.
