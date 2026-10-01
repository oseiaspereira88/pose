# Autonomous implementation candidates for POSE 6.2.0

Snapshot: 2026-10-01. Sources: `pose followups --open --json`, nonterminal specs,
and roadmap `adoption-developer-experience`. There are 123 open follow-up records;
that number includes duplicates, historical statements and conditional work.
This inventory is a candidate queue, not proof that every reported defect still
reproduces. Check current code first, reuse canonical specs, and create a scoped
spec/ADR where required. No item below depends on macOS/Windows verification.

## First: bounded corrections and useful coverage

| Work | Existing source |
| --- | --- |
| Report a malformed draft's delivery-contract errors on that spec rather than on every other spec | `review-verify-retains-completed-scopes` |
| Compare authority claim schema against its own schema constant | `pose-abm-review-authority` |
| Diagnose malformed human authority issuer digest pins | `pose-abm-review-authority` |
| Respect `.gitignore` in all module-discovery walkers | `pose-update-instance-directory-completeness` |
| Stabilize self-exec validation fixtures and deterministic cleanup of the 500-commit attribution fixture | `pose-test-self-exec-flake-fix`, `pose-v6-release-readiness` |
| Disable automatic Git GC consistently in repository test fixtures | `pose-contract-adoption-registry` |
| Cover remaining assess/install/self-update/MCP and helper command surfaces | `pose-remaining-command-surfaces-coverage` |
| Normalize the absolute excluded-spec path in review scope projections | `pose-review-bundle-roadmap-path-portability` |
| Show dirty details inside a submodule in review subjects | `pose-review-subject-submodule-classification` |
| Bind a criterion's finding to the actual problem that criterion reports | `pose-attest-records-not-applicable` |

## Performance, evidence and review diagnostics

| Work | Existing source |
| --- | --- |
| Batch Git object reads with `cat-file --batch`; measure speed and preserve identical findings | `check-worker-count-is-the-machines` |
| Resolve structured `test:` references rather than accepting invented test names; choose a compatible enforcement boundary | `pose-abm-design-basis` |
| Report which specs own unattributed commits | `pose-abm-subject-evidence` |
| Carry forward-evidence warnings into review verification | `pose-seal-names-carried-forward-evidence` |
| Count legacy bundles without governing contracts or gates, to support future migration decisions | `pose-bundles-seal-the-contracts-that-govern-them`, `pose-bundle-findings-take-the-contract-the-legacy-path-had` |
| Diagnose disagreeing contract adoption and legacy policy fields | `pose-contract-adoption-registry` |
| Diagnose inherited changelog adoption dates | `pose-changelog-adoption-is-the-instances` |
| Name invalid declarations inside stale review profiles | `pose-doctor-reports-a-profile-left-behind` |
| Distinguish roadmaps with no cut criteria from roadmaps that passed their gate | `pose-roadmap-check-reaches-its-gate` |
| Explain update-time adoption incompatibility with older engines | `pose-release-notes-name-contract-adoption` |

## Linux-verifiable release and CI hardening

| Work | Existing source |
| --- | --- |
| Discover release download links throughout docs and README instead of a manual file list | `pose-docs-asset-parity` |
| Assert the published asset set independently of documentation links | `pose-package-channel-install-repair` |
| Verify clean-tree assertion placement around pre-GoReleaser steps | `pose-release-clean-tree-attribution` |
| Share the SBOM dependency parser between verifier and fixtures | `pose-sbom-negative-coverage` |
| Exercise redirects, rate limits and partial bodies using a local HTTP origin | `pose-release-cycle-debt-closure` |
| Compare archive fixtures with the actual GoReleaser layout | `pose-release-boundary-rehearsal` |
| Parse workflow YAML robustly, including anchors and flow mappings, if the change's ADR justifies it | `pose-workflow-event-ref-contract` |
| Use AST analysis for live policy access through aliases | `pose-only-the-signing-gate-is-read-live` |
| Check guard declarations per step when a suite runs twice in one job | `pose-the-guard-signal-is-declared-not-inherited` |
| Prepare a safe Dependabot runtime-manifest refresh/check solution | `pose-dependency-pin-refresh`, `pose-release-security-gate-integrity` |
| Add a Linux snapshot rehearsal and local parity checks for `scripts/verify.sh` | `pose-shellcheck-ci-gate`, `pose-community-contribution-surfaces` |

## Larger changes: scope and compatibility decisions first

| Work | Existing source |
| --- | --- |
| Select evidence profiles through the component-aware planner and derive producible evidence classes from registered checks | `pose-class-producers-reads-the-disjunction`, `pose-review-plan-producible-evidence-classes` |
| Represent intentionally imported evidence separately from missing producers | `pose-report-a-demanded-class-nothing-produces` |
| Give `validationProfile` a defined behavior or remove the unused field with a migration plan | `pose-monorepo-validation-advisory` |
| Decide how discovery recognizes root aliases and stale entries | `pose-update-instance-directory-completeness` |
| Compare obligation-changing flags across locales, expand actual CLI localization, and reconcile remaining golden coverage | `pose-locale-coverage-contract`, `pose-cli-output-rendering-system` |
| Reconcile public surfaces against the claims contract and derive version-claim documentation from its schema | `pose-public-claims-contract`, `pose-public-claims-onboarding` |
| Define configurable changelog headings and assess a default security-scan producer | `pose-changelog-and-dor-policy-types`, `pose-emittable-analysis-evidence-classes` |
| Add stack fixtures for poetry/pipenv/.NET; assess redundant discovery walkers and explicit Workers/Android validation profiles | `pose-stack-catalog-expansion`, `pose-stack-detection-consolidation`, `pose-upgrade-path-audit-fixes` |
| Review component scoping of other synthesized tools | `pose-review-subject-and-scope-precision` |

These are autonomous design/implementation candidates under the existing mandate,
not authorization to silently change major-version defaults, authority contracts,
or behavior expressly deferred until field observations exist.

## Existing deliveries: reconcile before adding code

- Review closeout evidence for `pose-canonical-positioning`,
  `pose-readme-evaluation-path`, `pose-first-governed-loop-quickstart`,
  `pose-sdd-migration-acquisition` and `pose-cli-output-rendering-system`.
  Separate locally satisfied requirements from external site work or human timing.
- Reassess historical release-recovery/security blockers against current release
  and green CI evidence; do not perform another publication just to clear stale text.
- Prepare the launch demo recording and repository embedding on Linux, subject
  to available recording tools; public publication remains a separate action.
- Reconcile old follow-ups about extension catalogs, stack extension publication,
  explicit extension targets, profile schema findings, help flags and the report
  first-path defect against already delivered successor specs. Do not reimplement.
- Prepare local RFC/discussion and scoped-issue drafts for community surfaces;
  sending or publishing those requires the corresponding authorization.

## Work that remains conditional or external

- Native Homebrew/WinGet runtime round: **SKIPPED/DEFERRED**, exclusively in
  `spec:pose-package-channels-deferred-native-verification`; manual dispatch
  after implementation, no current runner execution.
- Release tag/publication, real signing-identity tamper proof, public tap ownership
  and upstream WinGet submission retain their own prerequisites.
- Harne8 adoption/federation, coordinator transfer reconciliation and verified
  issuer adapters require explicit project binding and consumer-side evidence.
- Real Harness/Conductor/semantic-provider integration requires its endpoint or
  adoption contract; mocks alone cannot establish composed delivery.
- ABM field-pilot stop/go, human reading-time acceptance, new-locale validation,
  and changes explicitly conditional on observed operator need retain those conditions.
- OTLP log stability must be checked against current upstream releases before
  deciding whether its previously deferred adoption is now appropriate.

Suggested execution order: bounded defects and fixture reliability; release/CI
hardening; evidence diagnostics and Git performance; larger scoped features;
closeout reconciliation; final manual native round.
