---
slug: pose-scaffold-allowlist
status: done
created_at: 2026-08-10
completed_at: 2026-08-10
supersedes:
depends_on: pose-scaffold-exclusion-policy
priority: 1
components: pose-mcp
delivers: governance:scaffold-allowlist
---

# Spec: The scaffold ships what an instance reads, and nothing else

## 1. Intent

### Goal
Invert the embedded scaffold from a denylist to an allowlist, so a new
product-level file is absent by default rather than distributed by default.

### Business value
`pose-scaffold-exclusion-policy` fixed one leak and named the reason it happened:
inclusion is the default, so every product file at the repository root is one
omission away from being distribution. That is a silent failure mode — nobody
decides to distribute, it just happens.

Auditing what was actually embedded showed the leak was not the exception. The
binary carried **4.8 MB** of scaffold and the installer reads almost none of it:
1.2 MB of this project's own specs, 472 KB of its reviews, 340 KB of release
manifests, 208 KB of ADRs, plus changelogs, assessments and results.
`.pose/specs` is created empty by `pose init`; the embedded copies are never
opened.

So the denylist was not merely risky in principle — it had already shipped
megabytes of this project's governance record inside every binary, across many
releases, without anyone deciding to.

### Constraints
- The installed instance must be unchanged. This is a packaging change; any
  difference in what a user gets is a regression, not an improvement.
- The allowlist has to be derived from what the installer and upgrader actually
  read, not from what looks reasonable.

### Non-goals
- Changing what `pose install` or `pose upgrade` deliver.
- Shrinking the binary as an end in itself. The size drop is a consequence of
  removing what was never read.

---

## 2. Requirements

### Functional
- R1: Only repository entries the installer or upgrader read shall be embedded.
- R2: `.pose/` shall be narrowed to machinery and contract files; this project's
  own governance record shall not be embedded.
- R3: A `pose install` after the change shall produce an identical instance.
- R4: The allowlist shall be the single source both the generator and the drift
  guard consult.

### Non-functional
- The embedded tree and binary shrink; neither is a target in itself.

### Security
- Narrows what is distributed. Nothing gains reach.

### Compatibility
- No instance loses anything: everything removed was never delivered.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy` — allowlist replaces the deny helpers.
- `pose-mcp/internal/scaffold/gen/main.go`, `scaffold_test.go` — consume it.

### Artifacts
- created: .pose/specs/pose-scaffold-allowlist/spec.md
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/gen/main.go
- modified: pose-mcp/internal/scaffold/scaffold_test.go

### Delivery targets
- governance:scaffold-allowlist module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None public.

### Technical risks
- **An omission now removes a file instead of adding one.** The failure inverts:
  a scaffold file left off the list stops being delivered. That is the trade —
  loud (an instance is missing machinery, and the drift guard plus the installer
  E2E both notice) instead of silent (a product file distributed forever) — but
  it is a trade, and the install comparison is what makes it safe to take.
- The allowlist was derived from a real install, so it reflects today's
  installer. A future delivery path reading a new tree needs the list extended.

---

## 4. Tasks

### Planning
- [x] Measure what is embedded against what the installer reads
- [x] Capture a baseline install to compare against

### Implementation
- [x] R1: allowlist of entries the installer reads
- [x] R2: narrow `.pose/` to machinery and contracts
- [x] R4: single source for generator and guard

### Validation
- [x] R3: compare the installed tree before and after
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The only claim that matters is that nothing changed for a user, and it is not
provable by reading the list — a plausible-looking allowlist that drops one
machinery file would pass review and break installs. So the change is validated
by installing into a fresh repository with the binary from before and after, and
comparing both the file set and the contents.

### Deterministic checks

#### Test
- Command: `pose install` into a fresh repository, before and after, then `diff -r`
- Scope: the delivered instance
- Expected: identical file set and contents

#### Build
- Command: `go -C pose-mcp test ./... -count=1` and `bash tests/install/run.sh`
- Scope: drift guard and installer E2E
- Expected: pass

### Execution log
- Date: 2026-08-10
- Environment: linux/amd64.
- Notes: the embedded tree drops from 514 files (4.8 MB) to 104 (1.9 MB) and the
  binary from 27.5 MB to 26.2 MB. Installing with each binary into a fresh git
  repository produces **53 files both times, an identical set**; `diff -r`
  reports only the scaffold placeholders resolved to each directory's own name
  (`inst-before` / `inst-after`), which is the installer working. The resulting
  instance passes `pose check --strict`, and the installer E2E passes.
- What left the binary: `.pose/specs` (1.2 MB), `.pose/reviews` (472 KB),
  `.pose/releases` (340 KB), `.pose/adr` (208 KB), `.pose/changelogs`,
  assessments, results, feedback, plus `README.md`, `CONTRIBUTING.md`,
  `SECURITY.md`, `scripts/`, `extensions/` and CI material.

### Results summary
- Successes: identical instance, 410 fewer embedded files, 1.3 MB smaller binary
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:scaffold-allowlist evidence:integration check:delivery-integration test:pose-mcp/internal/scaffold/scaffold_test.go — the embedded tree is the allowlist's closure, derived from what a real `pose install` reads
- R2 [satisfied] check:embedded-dist-drift — `.pose/` is narrowed to workflows, rules, templates, indexes, policy, review-profiles, schema-version and the legal texts; specs, reviews, releases, ADRs and changelogs are gone
- R3 [satisfied] check:install-parity — installs before and after produce the same 53 files with identical contents apart from resolved placeholders, and the instance passes `check --strict`
- R4 [satisfied] test:pose-mcp/internal/scaffold/scaffold_test.go — generator and guard both call `distpolicy.IsIncluded`

### Known gaps
- The list reflects today's delivery paths. A new one reading an unlisted tree
  fails by omission — loudly, but it fails.
- `install.sh` remains embedded although no delivery path reads it into an
  instance. It was kept because removing it could not be justified by the
  install comparison, which is the only evidence this spec trusts.

---

## 7. Final Report

### Delivered scope
The embedded scaffold is an allowlist of what the installer and upgrader read.
The instance is unchanged; the binary carries 1.3 MB less.

### Files and modules changed
- pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- pose-mcp/internal/scaffold/gen/main.go
- pose-mcp/internal/scaffold/scaffold_test.go

### Validation executed
- Command: before/after install comparison; scaffold suite; installer E2E
- Result: identical instance; all gates pass

### Residual risks
- Omission now under-delivers instead of over-delivering.

### Follow-ups

- [done] Removed. The two references were in the generator, using install.sh as a filesystem marker to locate the repository root — not a read from the embed; the earlier grep that suggested otherwise was counting exactly those. Dropping it leaves the installed tree identical to the pre-allowlist baseline and the installer E2E green. Original item: `install.sh` is still embedded and no delivery path reads it into an instance. It was left in place because removing it could not be justified by the install-comparison evidence this spec relies on — that comparison cannot show the absence of a use it never exercised. Establish whether any consumer reads it from the embed, and drop it if not. (owner:@pose-maintainers crit:low review:2026-11-13)
