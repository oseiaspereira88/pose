# Quickstart

**Doc type:** Tutorial &nbsp;·&nbsp; **Applies to:** POSE 7.x

## Install

### Fast path — Linux and macOS

Run the installer from the root of the Git repository that should receive the
POSE contract:

```bash
curl -fsSLO https://github.com/oseiaspereira88/pose/releases/latest/download/install.sh && bash install.sh
```

The script resolves the latest release for the current OS and architecture,
installs `pose` into `~/.local/bin` and runs `pose install .` plus the final
strict gate when the current directory is a Git repository. Ensure
`~/.local/bin` is on `PATH`, then confirm the installation:

```bash
pose version
pose doctor
```

The one-liner tracks `latest` and optimizes first-time adoption. It relies on
HTTPS but does not independently verify the downloaded archive's checksum or
Sigstore identity. Use the pinned flow below when reproducibility or
supply-chain verification is required.

### Verified install

```bash
# with the native binary on PATH:
pose install /path/to/your/repo

# or from a release bundle containing install.sh beside the pose binary:
bash install.sh /path/to/your/repo
```

The installer copies workflows, rules, templates and skills, derives
`{{PROJECT_NAME}}`/`{{PROJECT_ID}}` from your directory name (override with
`--project-name` / `--project-id`), configures the same binary as the MCP
server, stamps the contract schema version and finishes with native `init`,
`index` and `check --strict` — installation only reports success if the gate
is green.

The v1 release archive is accompanied by SHA-256 checksums, keyless Sigstore
signatures, a CycloneDX SBOM and SLSA provenance. Use the release's own
`checksums.txt` before placing the binary on `PATH`; package-manager channel
support and its current limits are documented in [Package channels](package-channels.md).

Useful flags:

| Flag | Effect |
|---|---|
| `--locale pt-BR` | Install docs and templates in Brazilian Portuguese |
| `--force` | Overwrite an edited `AGENTS.md`/`POSE.md` on re-run |
| `--skip-mcp` | Skip the MCP server entirely |
| `--allow-non-git` | Install into a non-git directory (not recommended) |

Re-running the installer updates the machinery and **never touches your
instance content** (specs, ADRs, knowledge, reports, roadmaps). Custom rules,
workflows and templates you added are preserved.

## Detect your stacks

```bash
pose init --wizard        # interactive; --yes accepts all suggestions
```

The wizard detects modules by stack markers (`go.mod`, `package.json`,
`Cargo.toml`, `pom.xml`, `build.gradle`) and seeds them into the validation
matrix in `tolerant` mode — promote to `strict` when the checks stabilize.

## The first governed delivery

This is the whole point of the quickstart: not to show you twenty commands,
but to walk one piece of work through the lifecycle every later spec follows —
and to watch each gate **refuse for a reason you understand, and then pass**.
Every command below is the one a project runs; no status is edited by hand.
`tests/quickstart/first-governed-loop.sh` runs this page in CI and asserts the
outputs shown.

An all-green run would not show you anything. Any tool can agree with you.

### Chapter 1 — Set up

`pose install` ends by naming the next step:

```
[pose-install] next: `pose setup` — identity, the commit gate and capability decisions, one confirmed step at a time
```

```bash
pose setup
```

`pose setup` shows what is in force, what is new and what is missing, with one
next step. On a fresh install it asks for a maintainer: a request addressed to
the maintainer has nobody to answer it until someone holds the role. Register
yourself with a key — your git identity only suggests the name; the key is the
proof:

```bash
pose identity add --key ~/.ssh/id_ed25519_sk.pub --role maintainer --apply
pose hooks install
git add -A && git commit -m "Adopt POSE" -m "POSE-Spec: pose-onboarding"
pose doctor
```

```
identity.suggested=human:you (from git user.email; a name, not a proof — the key is the proof)
identity.presence=proves presence: every signature records whether the security key was touched
...
doctor: 0 error(s), 0 warning(s), 1 next step(s)
```

A security key (`ssh-keygen -t ed25519-sk`) is preferred for a person; a plain
`ssh-keygen -t ed25519` key works and `pose doctor` says it cannot prove
presence. The install also scaffolded `.pose/specs/<date>-pose-onboarding.md`,
which tracks these steps as governed work.

### Chapter 2 — The entry gate

```bash
pose new-spec customer-export
pose lint-spec customer-export --ready-check
```

```
Spec created: .pose/specs/2026-09-07-customer-export.md (status: draft)
[ERROR] customer-export: DoR: section Intent is missing, empty, or skeletal
spec.ready=false
```

The spec exists, but nothing about it is decided yet. POSE will not let work
start against a spec that has not said what it is for. Fill the **Intent**,
one acceptance criterion with a stable ID, and the files the work will touch:

```markdown
### Goal
Export customer records as CSV for audit.

### Business value
Auditors currently request exports by hand.
```

```markdown
- R1: The exporter shall write customer records as CSV.
```

```markdown
### Artifacts
- created: svc/export.go
- created: svc/export_test.go
```

```bash
pose lint-spec customer-export --ready-check     # spec.ready=true
```

### Chapter 3 — Start, implement, prove

```bash
pose start spec:customer-export                  # preview: start.ready=true, start.digest=...
pose start spec:customer-export --apply --digest <start.digest>
git add -A && git commit -m "Start customer-export" -m "POSE-Spec: customer-export"
```

`pose start` moves the spec to `in-progress` and records the baseline the work
is reconciled against. Implement, then run *your* repository's declared checks
— the commands in the validation matrix, not ones POSE invented — and commit
with the spec's trailer so the change set is attributed to it:

```bash
pose suggest feature          # the workflow, skill, rules and checks for this kind of work
pose validate --strict
git add -A && git commit -m "Export customers as CSV" -m "POSE-Spec: customer-export"
```

### Chapter 4 — A decision the agent cannot take alone

Midway, the agent hits a question only the maintainer can answer. It does not
guess; it opens an action request that restricts what the answer affects:

```bash
pose action open --origin spec:customer-export --kind decision \
  --question "Include inactive customers in the audit export?" \
  --option "include=Auditors see every customer" --option "exclude=Inactive customers are left out" \
  --recommend include --recipient-role maintainer --requested-by agent:impl \
  --target requirement:R1 --effect closeout:block --apply
pose state --attention --actor maintainer       # the request waits on the maintainer
pose close spec:customer-export                 # refused: 1 action request(s) restrict closeout
```

The maintainer answers with a signature from the key they registered; POSE
verifies it offline and keeps it in the journal:

```bash
pose action resolve <act-id> --actor human:you --answer include \
  --request-digest <digest> --expected-revision 1 --idempotency-key answer-1 \
  --sign ~/.ssh/id_ed25519_sk --apply
pose action show <act-id>
```

```
action.answer=include by human:you (verified)
action.signature=SHA256:... (presence asserted), re-verified from the journal
```

An agent can relay the same answer over MCP with `pose_action_resolve`, which
records only an answer the principal signed.

### Chapter 5 — The exit gate, review and close

```bash
pose close spec:customer-export --apply --reviewer agent:reviewer
```

```
closeout_plan.step.trace=blocked — requirement trace incomplete: R1 has no trace entry — declare each under `### Requirement trace`, e.g. `- R1 [satisfied] test:<TestName>` ...
```

Read that line slowly, because it is the product in one line. The checks
passed and the decision is answered — POSE is not disputing your code. It is
pointing out that **R1, the thing you promised, is not connected to any
evidence that it happened**. "Done" is a claim by whoever executed; POSE
requires it to be a property of the repository. Connect the promise to the
proof and commit:

```markdown
### Requirement trace
- R1 [satisfied] test:TestCustomerExportWritesCSV
```

```bash
git add -A && git commit -m "Trace R1 to its test" -m "POSE-Spec: customer-export"
pose close spec:customer-export --apply --reviewer agent:reviewer
```

This time `pose close` regenerates the evidence, indexes and seals the review
bundle, then stops for what only a reviewer can conclude — and prints the
attest command already filled with the sealed evidence and the required tools:

```
closeout_plan.stopped=waiting on a reviewer: ...
closeout_plan.attest=pose review attest rvb-... --reviewer <id> --decision approved --evidence unit:svc/go/test --tool 'artifact-check|-|passed|check:artifact|' ... --criterion 'security|passed|unit:svc/go/test|<your conclusion>' --apply
```

The reviewer writes each conclusion, runs it, and the plan resumes:

```bash
pose close spec:customer-export --resume
```

```
closeout_plan.result=closed spec:customer-export; commit the closeout (spec, index, results, bundle, attestation) in one commit with its POSE-Spec trailer
```

That is a governed delivery: an entry gate that refused to start
under-specified work, a recorded start, real repository checks, a decision the
maintainer proved, an exit gate that refused to close until every promise
pointed at evidence, and a review bound to a sealed subject.

If a requirement turned out not to apply, you say that instead — `[waived:
<reason>]` or `[withdrawn: <reason>]`. What you cannot do is stay silent,
which is the option most processes leave open.

## See the first analytics

Recognized CLI/MCP activity is observed automatically and locally. The query
does not count itself:

```bash
pose usage --since-days 30
pose adoption-metrics --json
```

These are product usage and adoption signals. They do not infer deployment
outcomes. To measure delivery, ingest explicit deployment/incident events and
run `pose dora-metrics`; see [Analytics and delivery metrics](analytics.md).

## Keep it healthy

```bash
pose check --strict       # structural integrity + graphs + schema version
pose validate --tolerant  # run the validation matrix
pose followups --open     # live backlog from all specs
pose setup                # what is in force, new and missing; the next step
pose update               # migrate the contract; asks about new capabilities in a review spec
pose hooks install        # pre-commit check + post-merge reindex
```

Requirements: Git plus Bash for the one-liner, or the native `pose` binary for
the verified/manual path. The POSE runtime itself needs no Bash or Python.
Platforms: Linux, macOS and Windows.
